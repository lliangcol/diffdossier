package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/lliangcol/diffdossier/internal/config"
	"github.com/lliangcol/diffdossier/internal/gitrepo"
	"github.com/lliangcol/diffdossier/internal/platform"
	"github.com/lliangcol/diffdossier/internal/store"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

func runRun(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "archive" {
		fmt.Fprintln(stderr, "usage: diffdossier run archive --reason TEXT [--pin] [--repo PATH] [--state-dir PATH] [--run-id ID] [--json]")
		return ExitUsage
	}
	flags := flag.NewFlagSet("run archive", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "target Git repository")
	state := flags.String("state-dir", "", "durable state directory")
	runID := flags.String("run-id", "", "terminal run ID (default: latest active)")
	reason := flags.String("reason", "", "archive reason")
	pinned := flags.Bool("pin", false, "prevent automatic garbage collection")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args[1:]); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || *reason == "" {
		return ExitUsage
	}
	resolved, err := resolveExportContext(*repo, *state, *runID)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_ARCHIVE_CONTEXT", err.Error()), ExitEvidence)
	}
	record, err := resolved.stateStore.ArchiveRun(resolved.repository.ID, resolved.run.ID, *reason, *pinned, time.Now().UTC())
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_ARCHIVE_FAILED", err.Error()), ExitEvidence)
	}
	data := map[string]any{"archive": record, "read_only": true}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "archived run %s (%s, pinned=%t)\n", record.RunID, record.RecordDigest, record.Pinned)
	return ExitOK
}

func runGC(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "run" {
		return runGCExecute(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "plan" {
		args = args[1:]
	}
	return runGCPlan(args, stdout, stderr)
}

func runGCPlan(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gc plan", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoFlag := flags.String("repo", ".", "target Git repository")
	configFlag := flags.String("config", "", "configuration file")
	stateFlag := flags.String("state-dir", "", "durable state directory")
	asOfFlag := flags.String("as-of", "", "exact RFC3339 cutoff anchor (default: now)")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 {
		return ExitUsage
	}
	repo, err := gitrepo.Open(context.Background(), *repoFlag)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_GIT_REPOSITORY", err.Error()), ExitEvidence)
	}
	configPath := *configFlag
	if configPath == "" {
		configPath = filepath.Join(repo.Root, "diffdossier.toml")
	} else if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(repo.Root, configPath)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_CONFIG_INVALID", err.Error()), ExitUsage)
	}
	stateRoot, err := resolveStateRoot(*stateFlag)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", err.Error()), ExitUsage)
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
	asOf := time.Now().UTC()
	if *asOfFlag != "" {
		asOf, err = time.Parse(time.RFC3339Nano, *asOfFlag)
		if err != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_TIME", "as-of must be RFC3339"), ExitUsage)
		}
	}
	plan, err := stateStore.PlanGC(repository.ID, cfg.State.RetentionDays, asOf)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_GC_PLAN", err.Error()), ExitEvidence)
	}
	data := map[string]any{"dry_run": true, "plan": plan, "execution_required": len(plan.Candidates) > 0}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "GC dry-run: %d runs, %d blobs; execute only with --trust-gc-plan %s\n", len(plan.Candidates), len(plan.BlobDigests), plan.PlanDigest)
	return ExitOK
}

func runGCExecute(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gc run", flag.ContinueOnError)
	flags.SetOutput(stderr)
	stateFlag := flags.String("state-dir", "", "durable state directory")
	trust := flags.String("trust-gc-plan", "", "exact GC plan digest")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || *trust == "" {
		return ExitUsage
	}
	stateRoot, err := resolveStateRoot(*stateFlag)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", err.Error()), ExitUsage)
	}
	stateStore, err := store.Open(stateRoot)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_STORE", err.Error()), ExitEvidence)
	}
	execution, err := stateStore.ExecuteGC(*trust, time.Now().UTC())
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_GC_EXECUTION", err.Error()), ExitEvidence)
	}
	data := map[string]any{"execution": execution}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "GC executed: %d runs, %d blobs (%s)\n", execution.RemovedRuns, execution.RemovedBlobs, execution.PlanDigest)
	return ExitOK
}

func resolveStateRoot(stateRoot string) (string, error) {
	if stateRoot == "" {
		paths, err := platform.DefaultPaths()
		if err != nil {
			return "", err
		}
		stateRoot = paths.StateDir
	}
	if !filepath.IsAbs(stateRoot) {
		return "", errors.New("state-dir must be absolute")
	}
	return filepath.Clean(stateRoot), nil
}
