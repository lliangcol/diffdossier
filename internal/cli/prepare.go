package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/lliangcol/diffdossier/internal/config"
	"github.com/lliangcol/diffdossier/internal/gitrepo"
	"github.com/lliangcol/diffdossier/internal/platform"
	"github.com/lliangcol/diffdossier/internal/snapshot"
	"github.com/lliangcol/diffdossier/internal/store"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
	embeddedschemas "github.com/lliangcol/diffdossier/schemas"
)

func runPrepare(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("prepare", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoFlag := flags.String("repo", ".", "target Git repository")
	configFlag := flags.String("config", "", "configuration file")
	stateFlag := flags.String("state-dir", "", "durable state directory")
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
	stateRoot := *stateFlag
	if stateRoot == "" {
		paths, pathErr := platform.DefaultPaths()
		if pathErr != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PLATFORM_PATHS", pathErr.Error()), ExitInternal)
		}
		stateRoot = paths.StateDir
	}
	if !filepath.IsAbs(stateRoot) {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", "state-dir must be absolute"), ExitUsage)
	}
	digests, err := semanticDigests(configPath)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EVIDENCE_DIGEST", err.Error()), ExitEvidence)
	}
	seal, err := snapshot.Capture(ctx, snapshot.Request{
		Repo: repo, Baseline: cfg.Baseline, InputDigests: digests,
		IncludeUntracked: cfg.IncludeUntracked, IncludeIgnored: cfg.IncludeIgnored,
	})
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_SNAPSHOT_CAPTURE", err.Error()), ExitEvidence)
	}
	stateStore, err := store.Open(stateRoot)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_STORE", err.Error()), ExitEvidence)
	}
	repository, err := stateStore.Register(repo.Root)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_REGISTER", err.Error()), ExitEvidence)
	}
	run, runDir, err := stateStore.BeginRun(repository, seal)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_RUN", err.Error()), ExitEvidence)
	}
	result := map[string]any{
		"repository_id": repository.ID, "run_id": run.ID, "snapshot_id": seal.SnapshotID,
		"state": run.State, "state_path": runDir, "freshness": seal.Revisions.Freshness,
		"path_count": len(seal.Inventory.Entries),
	}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(result))
	}
	if _, err := fmt.Fprintf(stdout, "prepared %s at %s (%d scoped path entries, %s)\n", run.ID, seal.SnapshotID, len(seal.Inventory.Entries), seal.Revisions.Freshness); err != nil {
		fmt.Fprintf(stderr, "write prepare output: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

func semanticDigests(configPath string) (map[string]string, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(content)
	result, err := embeddedschemas.Digests()
	if err != nil {
		return nil, err
	}
	result["config"] = "sha256:" + hex.EncodeToString(digest[:])
	return result, nil
}
