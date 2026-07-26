package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/lliangcol/diffdossier/internal/config"
	"github.com/lliangcol/diffdossier/internal/gitrepo"
	"github.com/lliangcol/diffdossier/internal/platform"
	"github.com/lliangcol/diffdossier/internal/snapshot"
	"github.com/lliangcol/diffdossier/internal/store"
	"github.com/lliangcol/diffdossier/internal/workflow"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

type mutableRunContext struct {
	stateStore *store.Store
	run        store.Run
	seal       snapshot.Seal
	runDir     string
	request    snapshot.Request
}

func loadMutableRun(repoPath, configPath, stateRoot, runID string) (mutableRunContext, error) {
	ctx := context.Background()
	repo, err := gitrepo.Open(ctx, repoPath)
	if err != nil {
		return mutableRunContext{}, err
	}
	if configPath == "" {
		configPath = filepath.Join(repo.Root, "diffdossier.toml")
	} else if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(repo.Root, configPath)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return mutableRunContext{}, err
	}
	if stateRoot == "" {
		paths, pathErr := platform.DefaultPaths()
		if pathErr != nil {
			return mutableRunContext{}, pathErr
		}
		stateRoot = paths.StateDir
	}
	if !filepath.IsAbs(stateRoot) {
		return mutableRunContext{}, errors.New("state-dir must be absolute")
	}
	if err := requireOutsideRepository(repo.Root, stateRoot); err != nil {
		return mutableRunContext{}, err
	}
	stateStore, err := store.Open(stateRoot)
	if err != nil {
		return mutableRunContext{}, err
	}
	repository, err := stateStore.Register(repo.Root)
	if err != nil {
		return mutableRunContext{}, err
	}
	if runID == "" {
		latest, latestErr := stateStore.LatestRun(repository.ID)
		if latestErr != nil {
			return mutableRunContext{}, latestErr
		}
		runID = latest.ID
	}
	run, seal, err := stateStore.LoadRun(repository.ID, runID)
	if err != nil {
		return mutableRunContext{}, err
	}
	digests, err := semanticDigests(repo, configPath, cfg.Risk.PolicyFiles)
	if err != nil {
		return mutableRunContext{}, err
	}
	request := snapshot.Request{Repo: repo, Baseline: cfg.Baseline, InputDigests: digests, IncludeUntracked: cfg.IncludeUntracked, IncludeIgnored: cfg.IncludeIgnored}
	if err := snapshot.VerifyFresh(ctx, request, seal); err != nil {
		return mutableRunContext{}, err
	}
	runDir, err := stateStore.RunDir(repository.ID, run.ID)
	if err != nil {
		return mutableRunContext{}, err
	}
	return mutableRunContext{stateStore: stateStore, run: run, seal: seal, runDir: runDir, request: request}, nil
}

func runFinding(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || (args[0] != "confirm" && args[0] != "reject" && args[0] != "accept-risk") {
		fmt.Fprintln(stderr, "usage: diffdossier finding confirm|reject|accept-risk ...")
		return ExitUsage
	}
	action := args[0]
	flags := flag.NewFlagSet("finding "+action, flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "target Git repository")
	configPath := flags.String("config", "", "configuration file")
	state := flags.String("state-dir", "", "durable state directory")
	runID := flags.String("run-id", "", "run ID")
	findingID := flags.String("finding-id", "", "finding ID")
	operator := flags.String("operator", "", "decision operator")
	reason := flags.String("reason", "", "decision reason")
	owner := flags.String("owner", "", "accepted risk owner")
	trigger := flags.String("review-trigger", "", "accepted risk review trigger")
	expiresRaw := flags.String("expires-at", "", "accepted risk expiry RFC3339")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args[1:]); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || *findingID == "" || *operator == "" {
		return ExitUsage
	}
	context, err := loadMutableRun(*repo, *configPath, *state, *runID)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_WORKFLOW_CONTEXT", err.Error()), ExitStale)
	}
	var ledger workflow.FindingLedger
	if err := context.stateStore.ReadRunJSON(context.runDir, "findings.json", &ledger); err != nil {
		return ExitEvidence
	}
	next := map[string]string{"confirm": "confirmed", "reject": "rejected", "accept-risk": "accepted_risk"}[action]
	var expires *time.Time
	if *expiresRaw != "" {
		parsed, parseErr := time.Parse(time.RFC3339, *expiresRaw)
		if parseErr != nil {
			return ExitUsage
		}
		expires = &parsed
	}
	updated, err := workflow.TransitionFinding(ledger, *findingID, next, *operator, *reason, *owner, *trigger, expires, time.Now())
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_FINDING_TRANSITION", err.Error()), ExitUsage)
	}
	lock, err := store.AcquireRunLock(context.runDir)
	if err != nil {
		return ExitEvidence
	}
	defer lock.Release()
	if err := context.stateStore.WriteRunJSON(context.runDir, "findings.json", updated); err != nil {
		return ExitEvidence
	}
	if _, err := context.stateStore.AppendEvent(context.runDir, "finding_transition", map[string]string{"finding_id": *findingID, "state": next, "operator": *operator}); err != nil {
		return ExitEvidence
	}
	data := map[string]any{"run_id": context.run.ID, "finding_id": *findingID, "state": next}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "finding %s -> %s\n", *findingID, next)
	return ExitOK
}

func runFix(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "authorize" {
		fmt.Fprintln(stderr, "usage: diffdossier fix authorize ...")
		return ExitUsage
	}
	flags := flag.NewFlagSet("fix authorize", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "target Git repository")
	configPath := flags.String("config", "", "configuration file")
	state := flags.String("state-dir", "", "durable state directory")
	runID := flags.String("run-id", "", "run ID")
	idsRaw := flags.String("finding-ids", "", "comma-separated exact finding IDs")
	scope := flags.String("scope-digest", "", "exact external fixer scope digest")
	operator := flags.String("operator", "", "authorizing operator")
	expiresRaw := flags.String("expires-at", "", "authorization expiry RFC3339")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args[1:]); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || *idsRaw == "" || *scope == "" || *operator == "" || *expiresRaw == "" {
		return ExitUsage
	}
	expires, err := time.Parse(time.RFC3339, *expiresRaw)
	if err != nil {
		return ExitUsage
	}
	context, err := loadMutableRun(*repo, *configPath, *state, *runID)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_WORKFLOW_CONTEXT", err.Error()), ExitStale)
	}
	if context.run.State != "REVIEWED" {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_WORKFLOW_STATE", "fix authorization requires REVIEWED run"), ExitIncomplete)
	}
	var ledger workflow.FindingLedger
	if err := context.stateStore.ReadRunJSON(context.runDir, "findings.json", &ledger); err != nil {
		return ExitEvidence
	}
	ids := []string{}
	for _, id := range strings.Split(*idsRaw, ",") {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	authorization, updated, err := workflow.AuthorizeFix(ledger, context.seal.SnapshotID, ids, *operator, *scope, time.Now(), expires)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_FIX_AUTHORIZATION", err.Error()), ExitUsage)
	}
	lock, err := store.AcquireRunLock(context.runDir)
	if err != nil {
		return ExitEvidence
	}
	defer lock.Release()
	if err := context.stateStore.WriteRunJSON(context.runDir, "findings.json", updated); err != nil {
		return ExitEvidence
	}
	if err := context.stateStore.WriteRunJSON(context.runDir, filepath.Join("approvals", "fix-"+strings.TrimPrefix(authorization.Digest, "sha256:")[:16]+".json"), authorization); err != nil {
		return ExitEvidence
	}
	if _, err := context.stateStore.AppendEvent(context.runDir, "fix_authorized", map[string]any{"digest": authorization.Digest, "finding_ids": authorization.FindingIDs}); err != nil {
		return ExitEvidence
	}
	updatedRun, err := context.stateStore.UpdateRunStateHeld(context.runDir, "FIX_AUTHORIZED", lock)
	if err != nil {
		return ExitEvidence
	}
	data := map[string]any{"run_id": updatedRun.ID, "state": updatedRun.State, "authorization": authorization, "source_writes": 0}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "fix authorized for %d findings; no source files were modified\n", len(ids))
	return ExitOK
}
