package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/lliangcol/diffdossier/internal/config"
	"github.com/lliangcol/diffdossier/internal/gitrepo"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

func runInit(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoFlag := flags.String("repo", ".", "target Git repository")
	baselineFlag := flags.String("baseline", "", "explicit local baseline ref")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 {
		return ExitUsage
	}
	if *baselineFlag == "" {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError(
			"DD_INIT_BASELINE_REQUIRED",
			"baseline is required; DiffDossier does not guess main or master",
		), ExitUsage)
	}

	ctx := context.Background()
	repo, err := gitrepo.Open(ctx, *repoFlag)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_GIT_REPOSITORY", err.Error()), ExitEvidence)
	}
	revisions, err := repo.Resolve(ctx, *baselineFlag)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_INIT_BASELINE_INVALID", err.Error()), ExitEvidence)
	}
	document, err := config.MinimalDocument(*baselineFlag)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_INIT_BASELINE_INVALID", err.Error()), ExitUsage)
	}
	target := filepath.Join(repo.Root, "diffdossier.toml")
	if err := createConfigExclusive(target, document); err != nil {
		code := "DD_INIT_WRITE"
		exit := ExitEvidence
		if errors.Is(err, os.ErrExist) {
			code = "DD_INIT_EXISTS"
			exit = ExitUsage
		}
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError(code, err.Error()), exit)
	}

	result := map[string]any{
		"baseline_commit":  revisions.BaselineCommit,
		"baseline_ref":     revisions.BaselineRef,
		"commands_enabled": false,
		"config_path":      target,
		"created":          true,
		"freshness":        revisions.Freshness,
		"schema_version":   config.CurrentSchemaVersion,
	}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(result))
	}
	if _, err := fmt.Fprintf(stdout, "created %s for baseline %s (%s); no commands enabled\n", target, revisions.BaselineRef, revisions.Freshness); err != nil {
		fmt.Fprintf(stderr, "write init output: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

func createConfigExclusive(target string, content []byte) (returnErr error) {
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("%s already exists; refusing to overwrite: %w", target, os.ErrExist)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect config target: %w", err)
	}

	temporary, err := os.CreateTemp(filepath.Dir(target), ".diffdossier-init-*")
	if err != nil {
		return fmt.Errorf("create config staging file: %w", err)
	}
	temporaryPath := temporary.Name()
	temporaryOpen := true
	defer func() {
		if temporaryOpen {
			if closeErr := temporary.Close(); returnErr == nil && closeErr != nil {
				returnErr = fmt.Errorf("close config staging file: %w", closeErr)
			}
		}
		if removeErr := os.Remove(temporaryPath); returnErr == nil && removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			_ = os.Remove(target)
			returnErr = fmt.Errorf("remove config staging file: %w", removeErr)
		}
	}()

	if err := temporary.Chmod(0o644); err != nil {
		return fmt.Errorf("set config permissions: %w", err)
	}
	if _, err := temporary.Write(content); err != nil {
		return fmt.Errorf("write config staging file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync config staging file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close config staging file: %w", err)
	}
	temporaryOpen = false
	if _, err := config.Load(temporaryPath); err != nil {
		return fmt.Errorf("validate staged config: %w", err)
	}
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("%s appeared while initializing; refusing to overwrite: %w", target, os.ErrExist)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reinspect config target: %w", err)
	}
	if err := os.Link(temporaryPath, target); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%s appeared while initializing; refusing to overwrite: %w", target, os.ErrExist)
		}
		return fmt.Errorf("publish config without overwrite: %w", err)
	}
	return nil
}
