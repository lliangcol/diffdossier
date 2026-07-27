package cli

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lliangcol/diffdossier/internal/contracts"
	"github.com/lliangcol/diffdossier/internal/gitrepo"
	"github.com/lliangcol/diffdossier/internal/packets"
	"github.com/lliangcol/diffdossier/internal/planner"
	"github.com/lliangcol/diffdossier/internal/results"
	"github.com/lliangcol/diffdossier/internal/risk"
	"github.com/lliangcol/diffdossier/internal/snapshot"
	"github.com/lliangcol/diffdossier/internal/store"
	"github.com/lliangcol/diffdossier/internal/workflow"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

func runRecord(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "contract" {
		return runPlan(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "batch" {
		return runRecordBatch(args[1:], stdout, stderr)
	}
	if len(args) == 0 || args[0] != "task" {
		fmt.Fprintln(stderr, "usage: diffdossier record contract ... | diffdossier record task --task-id ID --result PATH ... | diffdossier record batch --manifest PATH ...")
		return ExitUsage
	}
	flags := flag.NewFlagSet("record task", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoFlag := flags.String("repo", ".", "target Git repository")
	configFlag := flags.String("config", "", "configuration file")
	baselineFlag := flags.String("baseline", "", "exact local baseline ref override")
	stateFlag := flags.String("state-dir", "", "durable state directory")
	runFlag := flags.String("run-id", "", "contracted run ID (default: latest)")
	taskFlag := flags.String("task-id", "", "task ID")
	resultFlag := flags.String("result", "", "Review Result JSON file")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args[1:]); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || !validTaskID(*taskFlag) || *resultFlag == "" {
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
	if run.State != "CONTRACTED" && run.State != "REVIEWING" {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_WORKFLOW_STATE", "record task requires CONTRACTED or REVIEWING run"), ExitIncomplete)
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
	runDir, err := stateStore.RunDir(repository.ID, run.ID)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_RUN", err.Error()), ExitEvidence)
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
	rebuiltPlan := planner.Build(seal.SnapshotID, seal.Inventory.Entries, graph, assessment, planner.Limits{
		MaxFiles: cfg.Review.MaxFilesPerTask, MaxPacketBytes: int64(cfg.Review.MaxPacketBytes),
	})
	var storedPlan planner.Plan
	if err := stateStore.ReadRunJSON(runDir, "plan.json", &storedPlan); err != nil || !sameCanonicalJSON(storedPlan, rebuiltPlan) {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PLAN_INTEGRITY", "stored plan does not match deterministic reconstruction"), ExitEvidence)
	}
	var task planner.Task
	for _, candidate := range rebuiltPlan.Tasks {
		if candidate.ID == *taskFlag {
			task = candidate
			break
		}
	}
	if task.ID == "" {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_TASK_NOT_FOUND", "task is not present in the reconstructed plan"), ExitEvidence)
	}
	var storedTask planner.Task
	if err := stateStore.ReadRunJSON(runDir, filepath.Join("tasks", *taskFlag+".json"), &storedTask); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_TASK_NOT_FOUND", err.Error()), ExitEvidence)
	}
	if !sameCanonicalJSON(storedTask, task) {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_TASK_INTEGRITY", "stored task does not match deterministic reconstruction"), ExitEvidence)
	}
	packet, err := packets.Build(task, run.DataClass)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PACKET_BUILD", err.Error()), ExitEvidence)
	}
	var storedPacket packets.Packet
	if err := stateStore.ReadRunJSON(runDir, filepath.Join("packets", *taskFlag+".json"), &storedPacket); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PACKET_NOT_FOUND", err.Error()), ExitEvidence)
	}
	if !sameCanonicalJSON(storedPacket, packet) {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PACKET_INTEGRITY", "stored packet does not match deterministic reconstruction"), ExitEvidence)
	}
	resultFile, err := os.Open(*resultFlag)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RESULT_READ", err.Error()), ExitEvidence)
	}
	reviewResult, err := results.Parse(resultFile)
	closeErr := resultFile.Close()
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RESULT_SCHEMA", err.Error()), ExitEvidence)
	}
	if closeErr != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RESULT_READ", closeErr.Error()), ExitEvidence)
	}
	validation, err := results.Validate(reviewResult, task, packet.TaskInputHash, packet.PromptDigest)
	if err != nil {
		code := ExitEvidence
		if strings.Contains(err.Error(), "bind the current task input") {
			code = ExitStale
		}
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RESULT_INVALID", err.Error()), code)
	}
	finalDigests, err := semanticDigests(repo, effective)
	if err != nil || !sameDigests(digests, finalDigests) {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_SNAPSHOT_STALE", "semantic inputs changed while validating result"), ExitStale)
	}
	if err := snapshot.VerifyFresh(ctx, request, seal); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_SNAPSHOT_STALE", err.Error()), ExitStale)
	}
	runLock, err := store.AcquireRunLock(runDir)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_LOCK", err.Error()), ExitEvidence)
	}
	defer runLock.Release()
	var currentRun store.Run
	if err := stateStore.ReadRunJSON(runDir, "run.json", &currentRun); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_RUN", err.Error()), ExitEvidence)
	}
	if currentRun.State != "CONTRACTED" && currentRun.State != "REVIEWING" {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_WORKFLOW_STATE", "record task requires CONTRACTED or REVIEWING run"), ExitIncomplete)
	}
	index := results.Index{SchemaVersion: "1.0", Records: []results.Record{}}
	if err := stateStore.ReadRunJSON(runDir, "results/index.json", &index); err != nil && !os.IsNotExist(err) {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RESULT_INDEX", err.Error()), ExitEvidence)
	}
	if err := verifyResultIndex(stateStore, runDir, index, rebuiltPlan, run.DataClass); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RESULT_INDEX", err.Error()), ExitEvidence)
	}
	resultPath := results.ResultPath(task.ID, reviewResult.Reviewer.PassID)
	updatedIndex, err := results.Append(index, reviewResult, validation, resultPath, time.Now())
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RESULT_DUPLICATE", err.Error()), ExitEvidence)
	}
	ledger := workflow.FindingLedger{SchemaVersion: "1.0", Findings: []workflow.FindingRecord{}}
	if err := stateStore.ReadRunJSON(runDir, "findings.json", &ledger); err != nil && !os.IsNotExist(err) {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_FINDING_LEDGER", err.Error()), ExitEvidence)
	}
	updatedLedger, err := workflow.ImportFindings(ledger, reviewResult, time.Now())
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_FINDING_LEDGER", err.Error()), ExitEvidence)
	}
	resultDigest := ""
	for _, record := range updatedIndex.Records {
		if record.TaskID == task.ID && record.PassID == reviewResult.Reviewer.PassID {
			resultDigest = record.ResultDigest
			break
		}
	}
	if err := stateStore.WriteRunJSON(runDir, resultPath, reviewResult); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
	}
	if err := stateStore.WriteRunJSON(runDir, "results/index.json", updatedIndex); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
	}
	if err := stateStore.WriteRunJSON(runDir, "findings.json", updatedLedger); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
	}
	comparison, comparisonWritten, err := buildTaskComparison(stateStore, runDir, index, reviewResult)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RESULT_COMPARE", err.Error()), ExitEvidence)
	}
	if comparisonWritten {
		if err := stateStore.WriteRunJSON(runDir, filepath.Join("results", task.ID, "comparison.json"), comparison); err != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
		}
	}
	if _, err := stateStore.AppendEvent(runDir, "task_result_recorded", map[string]any{
		"task_id": task.ID, "pass_id": reviewResult.Reviewer.PassID,
		"result_digest": resultDigest, "completed": validation.Completed,
	}); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_EVENT", err.Error()), ExitEvidence)
	}
	if currentRun.State == "CONTRACTED" {
		if _, err := stateStore.UpdateRunStateHeld(runDir, "REVIEWING", runLock); err != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_WORKFLOW_STATE", err.Error()), ExitEvidence)
		}
		currentRun.State = "REVIEWING"
	}
	complete := results.ReviewComplete(updatedIndex, rebuiltPlan)
	if complete {
		if _, err := stateStore.UpdateRunStateHeld(runDir, "REVIEWED", runLock); err != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_WORKFLOW_STATE", err.Error()), ExitEvidence)
		}
	}
	data := map[string]any{
		"run_id": run.ID, "task_id": task.ID, "pass_id": reviewResult.Reviewer.PassID,
		"result_digest":    resultDigest,
		"result_completed": validation.Completed, "review_complete": complete,
	}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	if _, err := fmt.Fprintf(stdout, "recorded %s/%s (result completed=%t, review complete=%t)\n", task.ID, reviewResult.Reviewer.PassID, validation.Completed, complete); err != nil {
		return ExitInternal
	}
	return ExitOK
}

func sameCanonicalJSON(left, right any) bool {
	leftBytes, leftErr := json.Marshal(left)
	rightBytes, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}

func buildTaskComparison(stateStore *store.Store, runDir string, index results.Index, current results.Result) (results.Comparison, bool, error) {
	inputs := []results.Result{current}
	for _, record := range index.Records {
		if record.TaskID != current.TaskID {
			continue
		}
		var previous results.Result
		if err := stateStore.ReadRunJSON(runDir, record.ResultPath, &previous); err != nil {
			return results.Comparison{}, false, err
		}
		inputs = append(inputs, previous)
	}
	if len(inputs) < 2 {
		return results.Comparison{}, false, nil
	}
	return results.Compare(inputs...), true, nil
}

func verifyResultIndex(stateStore *store.Store, runDir string, index results.Index, plan planner.Plan, dataClass publicschema.DataClass) error {
	if index.SchemaVersion != "1.0" || index.Records == nil {
		return errors.New("result index must use schema 1.0 with a records array")
	}
	tasks := map[string]planner.Task{}
	for _, task := range plan.Tasks {
		tasks[task.ID] = task
	}
	seen := map[string]bool{}
	for _, record := range index.Records {
		key := record.TaskID + "\x00" + record.PassID
		if seen[key] {
			return errors.New("result index contains a duplicate task/pass")
		}
		seen[key] = true
		task, ok := tasks[record.TaskID]
		if !ok || record.ResultPath != results.ResultPath(record.TaskID, record.PassID) {
			return errors.New("result index references an unknown task or non-canonical path")
		}
		packet, err := packets.Build(task, dataClass)
		if err != nil {
			return err
		}
		var reviewResult results.Result
		if err := stateStore.ReadRunJSON(runDir, record.ResultPath, &reviewResult); err != nil {
			return err
		}
		validation, err := results.Validate(reviewResult, task, packet.TaskInputHash, packet.PromptDigest)
		if err != nil {
			return err
		}
		if err := results.VerifyRecord(record, reviewResult, validation); err != nil {
			return err
		}
	}
	return nil
}

func validTaskID(value string) bool {
	if len(value) != len("task-")+24 || !strings.HasPrefix(value, "task-") {
		return false
	}
	_, err := hex.DecodeString(value[len("task-"):])
	return err == nil
}
