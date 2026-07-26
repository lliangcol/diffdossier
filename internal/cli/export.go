package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/lliangcol/diffdossier/internal/exporter"
	"github.com/lliangcol/diffdossier/internal/gitrepo"
	"github.com/lliangcol/diffdossier/internal/platform"
	"github.com/lliangcol/diffdossier/internal/store"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

func runExport(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "portable" {
		return runExportPortable(args[1:], stdout, stderr)
	}
	if len(args) > 1 && args[0] == "public" && args[1] == "prepare" {
		return runExportPublicPrepare(args[2:], stdout, stderr)
	}
	fmt.Fprintln(stderr, "usage: diffdossier export portable ... | diffdossier export public prepare ...")
	return ExitUsage
}

type exportContext struct {
	stateStore *store.Store
	repository store.Repository
	run        store.Run
	runDir     string
	repoRoot   string
}

func resolveExportContext(repoPath, stateRoot, runID string) (exportContext, error) {
	repo, err := gitrepo.Open(nilContext(), repoPath)
	if err != nil {
		return exportContext{}, err
	}
	if stateRoot == "" {
		paths, pathErr := platform.DefaultPaths()
		if pathErr != nil {
			return exportContext{}, pathErr
		}
		stateRoot = paths.StateDir
	}
	if !filepath.IsAbs(stateRoot) {
		return exportContext{}, errors.New("state-dir must be absolute")
	}
	if err := requireOutsideRepository(repo.Root, stateRoot); err != nil {
		return exportContext{}, err
	}
	stateStore, err := store.Open(stateRoot)
	if err != nil {
		return exportContext{}, err
	}
	repository, err := stateStore.Register(repo.Root)
	if err != nil {
		return exportContext{}, err
	}
	if runID == "" {
		latest, latestErr := stateStore.LatestRun(repository.ID)
		if latestErr != nil {
			return exportContext{}, latestErr
		}
		runID = latest.ID
	}
	run, _, err := stateStore.LoadRun(repository.ID, runID)
	if err != nil {
		return exportContext{}, err
	}
	runDir, err := stateStore.RunDir(repository.ID, run.ID)
	if err != nil {
		return exportContext{}, err
	}
	return exportContext{stateStore: stateStore, repository: repository, run: run, runDir: runDir, repoRoot: repo.Root}, nil
}

func runExportPortable(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("export portable", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "target Git repository")
	state := flags.String("state-dir", "", "durable state directory")
	run := flags.String("run-id", "", "run ID")
	output := flags.String("output", "", "new portable ZIP path")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || *output == "" {
		return ExitUsage
	}
	absolute, err := filepath.Abs(*output)
	if err != nil {
		return ExitUsage
	}
	context, err := resolveExportContext(*repo, *state, *run)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_CONTEXT", err.Error()), ExitEvidence)
	}
	if err := requireOutsideRepository(context.repoRoot, absolute); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", "portable export output must be outside target repository"), ExitUsage)
	}
	content, manifest, err := exporter.Portable(context.runDir)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_PORTABLE", err.Error()), ExitEvidence)
	}
	file, err := os.OpenFile(absolute, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_WRITE", err.Error()), ExitEvidence)
	}
	_, writeErr := file.Write(content)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_WRITE", errors.Join(writeErr, closeErr).Error()), ExitEvidence)
	}
	data := map[string]any{"run_id": context.run.ID, "output": absolute, "run_digest": manifest.RunDigest, "bytes": len(content)}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "portable export written: %s (%s)\n", absolute, manifest.RunDigest)
	return ExitOK
}

func runExportPublicPrepare(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("export public prepare", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "target Git repository")
	state := flags.String("state-dir", "", "durable state directory")
	run := flags.String("run-id", "", "run ID")
	input := flags.String("input", "", "candidate input")
	class := flags.String("class", "", "public_synthetic, public_project, or private_project")
	action := flags.String("action", "create", "create or replace")
	policyDigest := flags.String("policy-digest", "", "policy digest")
	revision := flags.String("public-revision", "", "confirmed public revision")
	redaction := flags.String("redaction-approval-digest", "", "private redaction approval digest")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || *input == "" || *class == "" || *policyDigest == "" {
		return ExitUsage
	}
	context, err := resolveExportContext(*repo, *state, *run)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_CONTEXT", err.Error()), ExitEvidence)
	}
	content, err := os.ReadFile(*input)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_READ", err.Error()), ExitEvidence)
	}
	preparation, err := exporter.PreparePublic(content, publicschema.DataClass(*class), *action, *policyDigest, *revision, *redaction)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PUBLIC_PREPARE", err.Error()), ExitEvidence)
	}
	if err := context.stateStore.WriteRunJSON(context.runDir, "exports/public-preparation.json", preparation); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
	}
	if _, err := context.stateStore.AppendEvent(context.runDir, "public_export_prepared", map[string]any{"candidate_digest": preparation.Candidate.Digest, "action": preparation.Candidate.Action, "scan_findings": len(preparation.ScanFindings)}); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_EVENT", err.Error()), ExitEvidence)
	}
	data := map[string]any{"run_id": context.run.ID, "preparation": preparation, "approval_required": true, "bundle_created": false}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "public candidate prepared: %s (%d scan findings); approval required, no bundle created\n", preparation.Candidate.Digest, len(preparation.ScanFindings))
	return ExitOK
}

// nilContext avoids inventing cancellation semantics for short, local-only
// repository discovery.
func nilContext() context.Context { return context.Background() }
