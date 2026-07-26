package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/lliangcol/diffdossier/internal/contracts"
	"github.com/lliangcol/diffdossier/internal/gates"
	"github.com/lliangcol/diffdossier/internal/gitrepo"
	"github.com/lliangcol/diffdossier/internal/planner"
	"github.com/lliangcol/diffdossier/internal/reporting"
	"github.com/lliangcol/diffdossier/internal/results"
	"github.com/lliangcol/diffdossier/internal/risk"
	"github.com/lliangcol/diffdossier/internal/snapshot"
	"github.com/lliangcol/diffdossier/internal/store"
	"github.com/lliangcol/diffdossier/internal/workflow"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

func runVerify(args []string, stdout, stderr io.Writer, finalize bool) int {
	name := "verify"
	if finalize {
		name = "finalize"
	}
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoFlag := flags.String("repo", ".", "target Git repository")
	configFlag := flags.String("config", "", "configuration file")
	baselineFlag := flags.String("baseline", "", "exact local baseline ref override")
	stateFlag := flags.String("state-dir", "", "durable state directory")
	runFlag := flags.String("run-id", "", "run ID (default: latest)")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 {
		return ExitUsage
	}
	ctx := context.Background()
	repo, err := gitrepo.Open(ctx, *repoFlag)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_GIT_REPOSITORY", err.Error()), ExitEvidence)
	}
	effective, err := loadEffectiveConfig(repo.Root, *configFlag, *baselineFlag)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_CONFIG_INVALID", err.Error()), ExitUsage)
	}
	cfg := effective.Config
	stateRoot, err := resolveStateRoot(*stateFlag)
	if err != nil {
		return ExitUsage
	}
	stateStore, err := store.Open(stateRoot)
	if err != nil {
		return ExitEvidence
	}
	repository, err := stateStore.Register(repo.Root)
	if err != nil {
		return ExitEvidence
	}
	runID := *runFlag
	if runID == "" {
		latest, latestErr := stateStore.LatestRun(repository.ID)
		if latestErr != nil {
			return ExitEvidence
		}
		runID = latest.ID
	}
	run, seal, err := stateStore.LoadRun(repository.ID, runID)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_RUN", err.Error()), ExitEvidence)
	}
	runDir, err := stateStore.RunDir(repository.ID, run.ID)
	if err != nil {
		return ExitEvidence
	}
	digests, err := semanticDigests(repo, effective)
	if err != nil {
		return ExitEvidence
	}
	request := snapshot.Request{Repo: repo, Baseline: cfg.Baseline, InputDigests: digests, IncludeUntracked: cfg.IncludeUntracked, IncludeIgnored: cfg.IncludeIgnored}
	if err := snapshot.VerifyFresh(ctx, request, seal); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_SNAPSHOT_STALE", err.Error()), ExitStale)
	}
	var plan planner.Plan
	if err := stateStore.ReadRunJSON(runDir, "plan.json", &plan); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PLAN_INTEGRITY", err.Error()), ExitEvidence)
	}
	rules, err := contracts.DiscoverRules(repo.Root)
	if err != nil {
		return ExitEvidence
	}
	graph := contracts.Build(seal.Inventory.Entries, rules)
	overrides, err := risk.LoadOverrides(repo.Root, cfg.Risk.PolicyFiles)
	if err != nil && len(cfg.Risk.PolicyFiles) > 0 {
		return ExitEvidence
	}
	assessment, err := risk.Assess(seal.Inventory.Entries, graph, overrides)
	if err != nil {
		return ExitEvidence
	}
	rebuiltPlan := planner.Build(seal.SnapshotID, seal.Inventory.Entries, graph, assessment, planner.Limits{MaxFiles: cfg.Review.MaxFilesPerTask, MaxPacketBytes: int64(cfg.Review.MaxPacketBytes)})
	if !sameCanonicalJSON(plan, rebuiltPlan) {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PLAN_INTEGRITY", "stored plan does not match deterministic reconstruction"), ExitEvidence)
	}
	index := results.Index{SchemaVersion: "1.0", Records: []results.Record{}}
	if err := stateStore.ReadRunJSON(runDir, "results/index.json", &index); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RESULT_INDEX", err.Error()), ExitEvidence)
	}
	if err := verifyResultIndex(stateStore, runDir, index, plan); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RESULT_INDEX", err.Error()), ExitEvidence)
	}
	ledger := workflow.FindingLedger{SchemaVersion: "1.0", Findings: []workflow.FindingRecord{}}
	if err := stateStore.ReadRunJSON(runDir, "findings.json", &ledger); err != nil && !os.IsNotExist(err) {
		return ExitEvidence
	}
	coverage := map[string]int{}
	confirmations := []results.Confirmation{}
	residuals := []results.ResidualRisk{}
	byTask := map[string][]results.Result{}
	for _, record := range index.Records {
		var result results.Result
		if err := stateStore.ReadRunJSON(runDir, record.ResultPath, &result); err != nil {
			return ExitEvidence
		}
		byTask[result.TaskID] = append(byTask[result.TaskID], result)
		for _, item := range result.Coverage {
			coverage[item.Status]++
		}
		confirmations = append(confirmations, result.NeedsConfirmation...)
		residuals = append(residuals, result.ResidualRisks...)
	}
	comparisons := []results.Comparison{}
	taskIDs := make([]string, 0, len(byTask))
	for id := range byTask {
		taskIDs = append(taskIDs, id)
	}
	sort.Strings(taskIDs)
	for _, id := range taskIDs {
		if len(byTask[id]) > 1 {
			comparisons = append(comparisons, results.Compare(byTask[id]...))
		}
	}
	gateEvidence := []reporting.GateEvidence{}
	expectedGatePlan, err := gates.BuildPlan(gates.PlanRequest{RepositoryID: repository.ID, RepositoryRoot: repo.Root, Seal: seal, ConfigDigest: digests["config"], BinaryDigest: digests["binary"], Gates: cfg.Gates})
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_GATE_PLAN", err.Error()), ExitEvidence)
	}
	var gatePlan gates.Plan
	gatePlanErr := stateStore.ReadRunJSON(runDir, "gates/plan.json", &gatePlan)
	if gatePlanErr == nil && !sameCanonicalJSON(gatePlan, expectedGatePlan) {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_GATE_PLAN", "stored Gate plan does not match current exact plan"), ExitStale)
	}
	if gatePlanErr != nil && !os.IsNotExist(gatePlanErr) {
		return ExitEvidence
	}
	if gatePlanErr != nil {
		gatePlan = expectedGatePlan
	}
	if len(gatePlan.Gates) > 0 {
		var recorded []gates.Evidence
		if err := stateStore.ReadRunJSON(runDir, "gates/evidence.json", &recorded); err != nil && !os.IsNotExist(err) {
			return ExitEvidence
		}
		byID := map[string]gates.Evidence{}
		for _, item := range recorded {
			byID[item.GateID] = item
		}
		for _, gate := range gatePlan.Gates {
			item, ok := byID[gate.ID]
			status := "not_run"
			digest := ""
			if ok && item.SnapshotID == seal.SnapshotID && item.PlanDigest == gatePlan.PlanDigest && item.DefinitionDigest == gate.DefinitionDigest && (!gate.FinalAlways || item.FinalRun) {
				status = item.Status
				digest = item.DefinitionDigest
			}
			gateEvidence = append(gateEvidence, reporting.GateEvidence{ID: gate.ID, Blocking: gate.Blocking, Status: status, Digest: digest})
		}
	}
	reviewability := "reviewable"
	if len(plan.Tasks) == 0 {
		reviewability = "not_reviewable"
	}
	notes := []string{"This local verdict is not merge approval."}
	if !seal.Revisions.RemoteFetchProof {
		notes = append(notes, "Remote freshness was not fetched or verified for this run.")
	}
	report := reporting.Build(reporting.Input{RunID: run.ID, SnapshotID: seal.SnapshotID, Baseline: seal.Revisions.BaselineCommit, Head: seal.Revisions.HeadCommit, Worktree: seal.Revisions.Freshness, ReviewComplete: results.ReviewComplete(index, plan), Reviewability: reviewability, Coverage: coverage, Findings: ledger, NeedsConfirmation: confirmations, ResidualRisks: residuals, Gates: gateEvidence, Comparisons: comparisons, HumanMergeNotes: notes, EvidenceLimitations: []string{}, Now: time.Now()})
	if err := stateStore.WriteRunJSON(runDir, "reports/report.json", report); err != nil {
		return ExitEvidence
	}
	if err := stateStore.WriteRunBytes(runDir, "reports/report.md", reporting.Markdown(report)); err != nil {
		return ExitEvidence
	}
	if report.Verdict == "ready" && run.State == "REVIEWED" {
		updated, transitionErr := stateStore.UpdateRunState(runDir, "GATED")
		if transitionErr != nil {
			return ExitEvidence
		}
		run = updated
		updated, transitionErr = stateStore.UpdateRunState(runDir, "REREVIEWED")
		if transitionErr != nil {
			return ExitEvidence
		}
		run = updated
	}
	if finalize && report.Verdict == "ready" && run.State == "REREVIEWED" {
		updated, transitionErr := stateStore.UpdateRunState(runDir, "FINALIZED")
		if transitionErr != nil {
			return ExitEvidence
		}
		run = updated
	}
	data := map[string]any{"run_id": run.ID, "state": run.State, "report": report}
	if *jsonOutput {
		if code := writeJSON(stdout, stderr, publicschema.Success(data)); code != ExitOK {
			return code
		}
	} else {
		fmt.Fprintf(stdout, "%s: verdict=%s state=%s\n", name, report.Verdict, run.State)
	}
	if report.Verdict == "ready" {
		if finalize && run.State != "FINALIZED" {
			return ExitIncomplete
		}
		return ExitOK
	}
	if report.Verdict == "not_ready" {
		return ExitBlocked
	}
	return ExitIncomplete
}
