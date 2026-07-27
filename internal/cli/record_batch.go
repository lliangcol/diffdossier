package cli

import (
	"context"
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

const (
	maxBatchResults          = 1000
	maxBatchTotalResultBytes = 64 * 1024 * 1024
)

type recordBatchManifest struct {
	SchemaVersion string             `json:"schema_version"`
	Results       []recordBatchEntry `json:"results"`
}

type recordBatchEntry struct {
	TaskID     string `json:"task_id"`
	ResultPath string `json:"result_path"`
}

type pendingBatchResult struct {
	task       planner.Task
	result     results.Result
	validation results.Validation
}

func runRecordBatch(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("record batch", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoFlag := flags.String("repo", ".", "target Git repository")
	configFlag := flags.String("config", "", "configuration file")
	baselineFlag := flags.String("baseline", "", "exact local baseline ref override")
	stateFlag := flags.String("state-dir", "", "durable state directory")
	runFlag := flags.String("run-id", "", "contracted run ID (default: latest)")
	manifestFlag := flags.String("manifest", "", "JSON manifest of task result files")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || *manifestFlag == "" {
		return ExitUsage
	}

	fail := func(code int, problemCode, message string) int {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError(problemCode, message), code)
	}
	ctx := context.Background()
	repo, err := gitrepo.Open(ctx, *repoFlag)
	if err != nil {
		return fail(ExitEvidence, "DD_GIT_REPOSITORY", err.Error())
	}
	effective, err := loadEffectiveConfig(repo.Root, *configFlag, *baselineFlag)
	if err != nil {
		return fail(ExitUsage, "DD_CONFIG_INVALID", err.Error())
	}
	stateRoot, err := resolveStateRoot(*stateFlag)
	if err != nil {
		return fail(ExitUsage, "DD_USAGE_INVALID_PATH", "state-dir must be absolute")
	}
	if err := requireOutsideRepository(repo.Root, stateRoot); err != nil {
		return fail(ExitUsage, "DD_USAGE_INVALID_PATH", err.Error())
	}
	stateStore, err := store.Open(stateRoot)
	if err != nil {
		return fail(ExitEvidence, "DD_STATE_STORE", err.Error())
	}
	repository, err := stateStore.Register(repo.Root)
	if err != nil {
		return fail(ExitEvidence, "DD_STATE_REGISTER", err.Error())
	}
	runID := *runFlag
	if runID == "" {
		latest, latestErr := stateStore.LatestRun(repository.ID)
		if latestErr != nil {
			return fail(ExitEvidence, "DD_STATE_RUN", latestErr.Error())
		}
		runID = latest.ID
	}
	run, seal, err := stateStore.LoadRun(repository.ID, runID)
	if err != nil {
		return fail(ExitEvidence, "DD_STATE_RUN", err.Error())
	}
	if run.State != "CONTRACTED" && run.State != "REVIEWING" {
		return fail(ExitIncomplete, "DD_WORKFLOW_STATE", "record batch requires CONTRACTED or REVIEWING run")
	}
	digests, err := semanticDigests(repo, effective)
	if err != nil {
		return fail(ExitEvidence, "DD_EVIDENCE_DIGEST", err.Error())
	}
	request := snapshot.Request{Repo: repo, Baseline: effective.Config.Baseline, InputDigests: digests, IncludeUntracked: effective.Config.IncludeUntracked, IncludeIgnored: effective.Config.IncludeIgnored}
	if err := snapshot.VerifyFresh(ctx, request, seal); err != nil {
		return fail(ExitStale, "DD_SNAPSHOT_STALE", err.Error())
	}
	runDir, err := stateStore.RunDir(repository.ID, run.ID)
	if err != nil {
		return fail(ExitEvidence, "DD_STATE_RUN", err.Error())
	}
	rules, err := contracts.DiscoverRules(repo.Root)
	if err != nil {
		return fail(ExitEvidence, "DD_RULE_DISCOVERY", err.Error())
	}
	graph := contracts.Build(seal.Inventory.Entries, rules)
	overrides, err := risk.LoadOverrides(repo.Root, effective.Config.Risk.PolicyFiles)
	if err != nil && len(effective.Config.Risk.PolicyFiles) > 0 {
		return fail(ExitUsage, "DD_RISK_POLICY", err.Error())
	}
	assessment, err := risk.Assess(seal.Inventory.Entries, graph, overrides)
	if err != nil {
		return fail(ExitEvidence, "DD_RISK_ASSESSMENT", err.Error())
	}
	rebuiltPlan := planner.Build(seal.SnapshotID, seal.Inventory.Entries, graph, assessment, planner.Limits{MaxFiles: effective.Config.Review.MaxFilesPerTask, MaxPacketBytes: int64(effective.Config.Review.MaxPacketBytes)})
	var storedPlan planner.Plan
	if err := stateStore.ReadRunJSON(runDir, "plan.json", &storedPlan); err != nil || !sameCanonicalJSON(storedPlan, rebuiltPlan) {
		return fail(ExitEvidence, "DD_PLAN_INTEGRITY", "stored plan does not match deterministic reconstruction")
	}

	manifestInfo, err := os.Stat(*manifestFlag)
	if err != nil || !manifestInfo.Mode().IsRegular() || manifestInfo.Size() > results.MaxResultBytes {
		return fail(ExitEvidence, "DD_RESULT_BATCH_READ", "batch manifest must be a readable regular file no larger than 8 MiB")
	}
	manifestFile, err := os.Open(*manifestFlag)
	if err != nil {
		return fail(ExitEvidence, "DD_RESULT_BATCH_READ", err.Error())
	}
	decoder := json.NewDecoder(io.LimitReader(manifestFile, results.MaxResultBytes+1))
	decoder.DisallowUnknownFields()
	var manifest recordBatchManifest
	decodeErr := decoder.Decode(&manifest)
	var trailing any
	trailingErr := decoder.Decode(&trailing)
	closeErr := manifestFile.Close()
	if decodeErr != nil || closeErr != nil || manifest.SchemaVersion != "1.0" || len(manifest.Results) == 0 || len(manifest.Results) > maxBatchResults {
		return fail(ExitUsage, "DD_RESULT_BATCH_SCHEMA", "batch manifest must use schema 1.0 and contain 1 to 1000 results")
	}
	if trailingErr != io.EOF {
		return fail(ExitUsage, "DD_RESULT_BATCH_SCHEMA", "batch manifest must contain exactly one JSON object")
	}
	tasks := make(map[string]planner.Task, len(rebuiltPlan.Tasks))
	for _, task := range rebuiltPlan.Tasks {
		tasks[task.ID] = task
	}
	pending := make([]pendingBatchResult, 0, len(manifest.Results))
	seenEntries := map[string]bool{}
	var totalResultBytes int64
	for _, entry := range manifest.Results {
		task, ok := tasks[entry.TaskID]
		if !ok || !validTaskID(entry.TaskID) || !filepath.IsAbs(entry.ResultPath) {
			return fail(ExitEvidence, "DD_TASK_NOT_FOUND", "batch entry must reference a known task and an absolute result path")
		}
		var storedTask planner.Task
		if err := stateStore.ReadRunJSON(runDir, filepath.Join("tasks", task.ID+".json"), &storedTask); err != nil || !sameCanonicalJSON(storedTask, task) {
			return fail(ExitEvidence, "DD_TASK_INTEGRITY", "stored task does not match deterministic reconstruction")
		}
		packet, err := packets.Build(task, run.DataClass)
		if err != nil {
			return fail(ExitEvidence, "DD_PACKET_BUILD", err.Error())
		}
		var storedPacket packets.Packet
		if err := stateStore.ReadRunJSON(runDir, filepath.Join("packets", task.ID+".json"), &storedPacket); err != nil || !sameCanonicalJSON(storedPacket, packet) {
			return fail(ExitEvidence, "DD_PACKET_INTEGRITY", "stored packet does not match deterministic reconstruction")
		}
		resultInfo, err := os.Stat(entry.ResultPath)
		if err != nil || !resultInfo.Mode().IsRegular() || resultInfo.Size() > results.MaxResultBytes {
			return fail(ExitEvidence, "DD_RESULT_READ", "batch result must be a readable regular file no larger than 8 MiB")
		}
		totalResultBytes += resultInfo.Size()
		if totalResultBytes > maxBatchTotalResultBytes {
			return fail(ExitUsage, "DD_RESULT_BATCH_LIMIT", "batch result files exceed the 64 MiB aggregate limit")
		}
		resultFile, err := os.Open(entry.ResultPath)
		if err != nil {
			return fail(ExitEvidence, "DD_RESULT_READ", err.Error())
		}
		reviewResult, parseErr := results.Parse(resultFile)
		closeErr := resultFile.Close()
		if parseErr != nil || closeErr != nil {
			if parseErr != nil {
				return fail(ExitEvidence, "DD_RESULT_SCHEMA", parseErr.Error())
			}
			return fail(ExitEvidence, "DD_RESULT_READ", closeErr.Error())
		}
		validation, err := results.Validate(reviewResult, task, packet.TaskInputHash, packet.PromptDigest)
		if err != nil {
			code := ExitEvidence
			if strings.Contains(err.Error(), "bind the current task input") {
				code = ExitStale
			}
			return fail(code, "DD_RESULT_INVALID", err.Error())
		}
		key := task.ID + "\x00" + reviewResult.Reviewer.PassID
		if seenEntries[key] {
			return fail(ExitEvidence, "DD_RESULT_DUPLICATE", "batch contains a duplicate task/pass")
		}
		seenEntries[key] = true
		pending = append(pending, pendingBatchResult{task: task, result: reviewResult, validation: validation})
	}
	finalDigests, err := semanticDigests(repo, effective)
	if err != nil || !sameDigests(digests, finalDigests) {
		return fail(ExitStale, "DD_SNAPSHOT_STALE", "semantic inputs changed while validating result batch")
	}
	if err := snapshot.VerifyFresh(ctx, request, seal); err != nil {
		return fail(ExitStale, "DD_SNAPSHOT_STALE", err.Error())
	}

	runLock, err := store.AcquireRunLock(runDir)
	if err != nil {
		return fail(ExitEvidence, "DD_STATE_LOCK", err.Error())
	}
	defer runLock.Release()
	var currentRun store.Run
	if err := stateStore.ReadRunJSON(runDir, "run.json", &currentRun); err != nil {
		return fail(ExitEvidence, "DD_STATE_RUN", err.Error())
	}
	if currentRun.State != "CONTRACTED" && currentRun.State != "REVIEWING" {
		return fail(ExitIncomplete, "DD_WORKFLOW_STATE", "record batch requires CONTRACTED or REVIEWING run")
	}
	index := results.Index{SchemaVersion: "1.0", Records: []results.Record{}}
	if err := stateStore.ReadRunJSON(runDir, "results/index.json", &index); err != nil && !os.IsNotExist(err) {
		return fail(ExitEvidence, "DD_RESULT_INDEX", err.Error())
	}
	if err := verifyResultIndex(stateStore, runDir, index, rebuiltPlan, run.DataClass); err != nil {
		return fail(ExitEvidence, "DD_RESULT_INDEX", err.Error())
	}
	ledger := workflow.FindingLedger{SchemaVersion: "1.0", Findings: []workflow.FindingRecord{}}
	if err := stateStore.ReadRunJSON(runDir, "findings.json", &ledger); err != nil && !os.IsNotExist(err) {
		return fail(ExitEvidence, "DD_FINDING_LEDGER", err.Error())
	}
	now := time.Now().UTC()
	updatedIndex := index
	updatedLedger := ledger
	for _, item := range pending {
		resultPath := results.ResultPath(item.task.ID, item.result.Reviewer.PassID)
		updatedIndex, err = results.Append(updatedIndex, item.result, item.validation, resultPath, now)
		if err != nil {
			return fail(ExitEvidence, "DD_RESULT_DUPLICATE", err.Error())
		}
		updatedLedger, err = workflow.ImportFindings(updatedLedger, item.result, now)
		if err != nil {
			return fail(ExitEvidence, "DD_FINDING_LEDGER", err.Error())
		}
	}
	for _, item := range pending {
		if err := stateStore.WriteRunJSON(runDir, results.ResultPath(item.task.ID, item.result.Reviewer.PassID), item.result); err != nil {
			return fail(ExitEvidence, "DD_STATE_WRITE", err.Error())
		}
	}
	if err := stateStore.WriteRunJSON(runDir, "results/index.json", updatedIndex); err != nil {
		return fail(ExitEvidence, "DD_STATE_WRITE", err.Error())
	}
	if err := stateStore.WriteRunJSON(runDir, "findings.json", updatedLedger); err != nil {
		return fail(ExitEvidence, "DD_STATE_WRITE", err.Error())
	}
	for taskID := range tasks {
		byTask := []results.Result{}
		for _, record := range updatedIndex.Records {
			if record.TaskID != taskID {
				continue
			}
			var reviewResult results.Result
			if err := stateStore.ReadRunJSON(runDir, record.ResultPath, &reviewResult); err != nil {
				return fail(ExitEvidence, "DD_RESULT_INDEX", err.Error())
			}
			byTask = append(byTask, reviewResult)
		}
		if len(byTask) >= 2 {
			if err := stateStore.WriteRunJSON(runDir, filepath.Join("results", taskID, "comparison.json"), results.Compare(byTask...)); err != nil {
				return fail(ExitEvidence, "DD_STATE_WRITE", err.Error())
			}
		}
	}
	for _, item := range pending {
		resultDigest, digestErr := results.Digest(item.result)
		if digestErr != nil {
			return fail(ExitEvidence, "DD_RESULT_INDEX", digestErr.Error())
		}
		if _, err := stateStore.AppendEvent(runDir, "task_result_recorded", map[string]any{"task_id": item.task.ID, "pass_id": item.result.Reviewer.PassID, "result_digest": resultDigest, "batch": true, "completed": item.validation.Completed}); err != nil {
			return fail(ExitEvidence, "DD_STATE_EVENT", err.Error())
		}
	}
	complete := results.ReviewComplete(updatedIndex, rebuiltPlan)
	if currentRun.State == "CONTRACTED" {
		if _, err := stateStore.UpdateRunStateHeld(runDir, "REVIEWING", runLock); err != nil {
			return fail(ExitEvidence, "DD_WORKFLOW_STATE", err.Error())
		}
		currentRun.State = "REVIEWING"
	}
	if complete {
		if _, err := stateStore.UpdateRunStateHeld(runDir, "REVIEWED", runLock); err != nil {
			return fail(ExitEvidence, "DD_WORKFLOW_STATE", err.Error())
		}
	}
	data := map[string]any{"run_id": run.ID, "result_count": len(pending), "review_complete": complete}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	if _, err := fmt.Fprintf(stdout, "recorded %d results (review complete=%t)\n", len(pending), complete); err != nil {
		return ExitInternal
	}
	return ExitOK
}
