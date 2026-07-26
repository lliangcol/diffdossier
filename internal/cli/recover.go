package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/lliangcol/diffdossier/internal/gitrepo"
	"github.com/lliangcol/diffdossier/internal/store"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

func runRecover(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("recover", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoFlag := flags.String("repo", ".", "target Git repository")
	stateFlag := flags.String("state-dir", "", "durable state directory")
	runFlag := flags.String("run-id", "", "run ID (default: latest)")
	expected := flags.String("trust-journal-state", "", "exact journal-derived state to recover")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || *expected == "" {
		return ExitUsage
	}
	repo, err := gitrepo.Open(context.Background(), *repoFlag)
	if err != nil {
		return ExitEvidence
	}
	stateRoot, err := resolveStateRoot(*stateFlag)
	if err != nil || requireOutsideRepository(repo.Root, stateRoot) != nil {
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
	recovered, err := stateStore.RecoverRun(repository.ID, runID, *expected)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RUN_RECOVERY", err.Error()), ExitEvidence)
	}
	data := map[string]any{"run_id": recovered.ID, "state": recovered.State, "recovered": true}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "recovered %s to journal state %s\n", recovered.ID, recovered.State)
	return ExitOK
}
