package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/lliangcol/diffdossier/internal/contracts"
	"github.com/lliangcol/diffdossier/internal/gitrepo"
	"github.com/lliangcol/diffdossier/internal/packets"
	"github.com/lliangcol/diffdossier/internal/planner"
	"github.com/lliangcol/diffdossier/internal/risk"
	"github.com/lliangcol/diffdossier/internal/snapshot"
	"github.com/lliangcol/diffdossier/internal/store"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

func runPlan(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("plan", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoFlag := flags.String("repo", ".", "target Git repository")
	configFlag := flags.String("config", "", "configuration file")
	baselineFlag := flags.String("baseline", "", "exact local baseline ref override")
	stateFlag := flags.String("state-dir", "", "durable state directory")
	runFlag := flags.String("run-id", "", "prepared run ID (default: latest)")
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
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", "state-dir must be absolute"), ExitUsage)
	}
	if err := requireOutsideRepository(repo.Root, stateRoot); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", err.Error()), ExitUsage)
	}
	stateStore, err := store.Open(stateRoot)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_STORE", err.Error()), ExitEvidence)
	}
	repository, err := stateStore.Register(repo.Root)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_REGISTER", err.Error()), ExitEvidence)
	}
	runID := *runFlag
	if runID == "" {
		latest, latestErr := stateStore.LatestRun(repository.ID)
		if latestErr != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_RUN", latestErr.Error()), ExitEvidence)
		}
		runID = latest.ID
	}
	run, seal, err := stateStore.LoadRun(repository.ID, runID)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_RUN", err.Error()), ExitEvidence)
	}
	if run.State != "PREPARED" {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_WORKFLOW_STATE", "plan requires PREPARED run"), ExitIncomplete)
	}
	digests, err := semanticDigests(repo, effective)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EVIDENCE_DIGEST", err.Error()), ExitEvidence)
	}
	request := snapshot.Request{
		Repo: repo, Baseline: cfg.Baseline, InputDigests: digests,
		IncludeUntracked: cfg.IncludeUntracked, IncludeIgnored: cfg.IncludeIgnored,
	}
	if err := snapshot.VerifyFresh(ctx, request, seal); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_SNAPSHOT_STALE", err.Error()), ExitStale)
	}
	rules, err := contracts.DiscoverRules(repo.Root)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RULE_DISCOVERY", err.Error()), ExitEvidence)
	}
	graph := contracts.Build(seal.Inventory.Entries, rules)
	overrides, err := risk.LoadOverrides(repo.Root, cfg.Risk.PolicyFiles)
	if err != nil && len(cfg.Risk.PolicyFiles) > 0 {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RISK_POLICY", err.Error()), ExitUsage)
	}
	assessment, err := risk.Assess(seal.Inventory.Entries, graph, overrides)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RISK_ASSESSMENT", err.Error()), ExitEvidence)
	}
	plan := planner.Build(seal.SnapshotID, seal.Inventory.Entries, graph, assessment, planner.Limits{
		MaxFiles: cfg.Review.MaxFilesPerTask, MaxPacketBytes: int64(cfg.Review.MaxPacketBytes),
	})
	runDir, err := stateStore.RunDir(repository.ID, run.ID)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_RUN", err.Error()), ExitEvidence)
	}
	if err := stateStore.WriteRunJSON(runDir, "contract.json", graph); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
	}
	if err := stateStore.WriteRunJSON(runDir, "risk.json", assessment); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
	}
	if err := stateStore.WriteRunJSON(runDir, "plan.json", plan); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
	}
	needsConfirmation := 0
	for _, task := range plan.Tasks {
		if task.NeedsConfirmation {
			needsConfirmation++
		}
		if err := stateStore.WriteRunJSON(runDir, filepath.Join("tasks", task.ID+".json"), task); err != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
		}
		packet, packetErr := packets.Build(task, publicschema.PrivateProject)
		if packetErr != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PACKET_BUILD", packetErr.Error()), ExitEvidence)
		}
		if err := stateStore.WriteRunJSON(runDir, filepath.Join("packets", task.ID+".json"), packet); err != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
		}
	}
	if _, err := stateStore.UpdateRunState(runDir, "CONTRACTED"); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_WORKFLOW_STATE", err.Error()), ExitEvidence)
	}
	result := map[string]any{
		"run_id": run.ID, "snapshot_id": seal.SnapshotID, "state": "CONTRACTED",
		"task_count": len(plan.Tasks), "path_count": len(plan.Coverage),
		"needs_confirmation_tasks": needsConfirmation,
	}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(result))
	}
	if _, err := fmt.Fprintf(stdout, "planned %d tasks for %d paths (%d need confirmation)\n", len(plan.Tasks), len(plan.Coverage), needsConfirmation); err != nil {
		fmt.Fprintf(stderr, "write plan output: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}
