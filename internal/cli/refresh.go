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
	"sort"
	"time"

	"github.com/lliangcol/diffdossier/internal/gitrepo"
	"github.com/lliangcol/diffdossier/internal/inventory"
	"github.com/lliangcol/diffdossier/internal/planner"
	"github.com/lliangcol/diffdossier/internal/snapshot"
	"github.com/lliangcol/diffdossier/internal/store"
	"github.com/lliangcol/diffdossier/internal/workflow"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

func runRefresh(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("refresh", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoFlag := flags.String("repo", ".", "target Git repository")
	configFlag := flags.String("config", "", "configuration file")
	baselineFlag := flags.String("baseline", "", "exact local baseline ref override")
	stateFlag := flags.String("state-dir", "", "durable state directory")
	runFlag := flags.String("run-id", "", "FIX_AUTHORIZED run ID")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 {
		return ExitUsage
	}
	ctx := context.Background()
	repo, err := gitrepo.Open(ctx, *repoFlag)
	if err != nil {
		return ExitEvidence
	}
	effective, err := loadEffectiveConfig(repo.Root, *configFlag, *baselineFlag)
	if err != nil {
		return ExitUsage
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
	run, oldSeal, err := stateStore.LoadRun(repository.ID, runID)
	if err != nil {
		return ExitEvidence
	}
	if run.State != "FIX_AUTHORIZED" {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_WORKFLOW_STATE", "refresh requires FIX_AUTHORIZED run"), ExitIncomplete)
	}
	oldRunDir, err := stateStore.RunDir(repository.ID, run.ID)
	if err != nil {
		return ExitEvidence
	}
	var oldPlan planner.Plan
	if err := stateStore.ReadRunJSON(oldRunDir, "plan.json", &oldPlan); err != nil {
		return ExitEvidence
	}
	digests, err := semanticDigests(repo, effective)
	if err != nil {
		return ExitEvidence
	}
	newSeal, err := snapshot.Capture(ctx, snapshot.Request{Repo: repo, Baseline: cfg.Baseline, InputDigests: digests, IncludeUntracked: cfg.IncludeUntracked, IncludeIgnored: cfg.IncludeIgnored})
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_SNAPSHOT_CAPTURE", err.Error()), ExitEvidence)
	}
	if newSeal.SnapshotID == oldSeal.SnapshotID {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_REFRESH_NO_MUTATION", "authorized fixer produced no snapshot change"), ExitIncomplete)
	}
	changed := changedInventoryPaths(oldSeal.Inventory, newSeal.Inventory)
	scopeDigest := workflow.MutationScopeDigest(changed)
	authorized := false
	approvalPaths, _ := filepath.Glob(filepath.Join(oldRunDir, "approvals", "fix-*.json"))
	for _, path := range approvalPaths {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var approval workflow.FixAuthorization
		if json.Unmarshal(content, &approval) == nil && workflow.VerifyFixAuthorization(approval, time.Now()) == nil && approval.SnapshotID == oldSeal.SnapshotID && approval.ScopeDigest == scopeDigest {
			authorized = true
			break
		}
	}
	if !authorized {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_FIX_SCOPE_MISMATCH", "actual mutation paths do not match an unexpired exact fix authorization"), ExitBlocked)
	}
	semanticChanged := !sameDigests(oldSeal.InputDigests, newSeal.InputDigests)
	invalidation := workflow.ComputeInvalidation(oldPlan, changed, semanticChanged)
	if _, err := stateStore.UpdateRunState(oldRunDir, "MUTATED"); err != nil {
		return ExitEvidence
	}
	newRun, newRunDir, err := stateStore.BeginClassifiedRun(repository, newSeal, run.DataClass)
	if err != nil {
		return ExitEvidence
	}
	if err := stateStore.WriteRunJSON(newRunDir, "invalidation.json", invalidation); err != nil {
		return ExitEvidence
	}
	if _, err := stateStore.AppendEvent(newRunDir, "run_refreshed", map[string]any{"previous_run_id": run.ID, "previous_snapshot_id": oldSeal.SnapshotID, "scope_digest": scopeDigest, "must_reload": invalidation.MustReload}); err != nil {
		return ExitEvidence
	}
	data := map[string]any{"previous_run_id": run.ID, "run_id": newRun.ID, "snapshot_id": newSeal.SnapshotID, "state": newRun.State, "must_reload": invalidation.MustReload}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "refreshed into %s; %d tasks must reload\n", newRun.ID, len(invalidation.MustReload))
	return ExitOK
}

func changedInventoryPaths(oldInventory, newInventory inventory.Result) []string {
	fingerprints := func(value inventory.Result) map[string]string {
		result := map[string]string{}
		for _, entry := range value.Entries {
			content, _ := json.Marshal(struct {
				Scope                                                inventory.Scope
				Status, Kind, Mode, ContentHash, PreviousContentHash string
				Size, PreviousSize                                   int64
			}{entry.Scope, entry.Status, entry.Kind, entry.Mode, entry.ContentHash, entry.PreviousContentHash, entry.Size, entry.PreviousSize})
			result[entry.Path.BytesBase64] = string(content)
		}
		return result
	}
	oldMap, newMap := fingerprints(oldInventory), fingerprints(newInventory)
	changed := []string{}
	seen := map[string]bool{}
	for path, value := range oldMap {
		if newMap[path] != value {
			changed = append(changed, path)
			seen[path] = true
		}
	}
	for path, value := range newMap {
		if oldMap[path] != value && !seen[path] {
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)
	return changed
}
