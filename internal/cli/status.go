package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

func runStatus(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("status", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "target Git repository")
	state := flags.String("state-dir", "", "durable state directory")
	runID := flags.String("run-id", "", "run ID (default: latest active)")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 {
		return ExitUsage
	}
	resolved, err := resolveExportContext(*repo, *state, *runID)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATUS_CONTEXT", err.Error()), ExitEvidence)
	}
	data := map[string]any{
		"run_id":                resolved.run.ID,
		"repository_id":         resolved.repository.ID,
		"snapshot_id":           resolved.run.SnapshotID,
		"state":                 resolved.run.State,
		"task_count":            countJSONFiles(filepath.Join(resolved.runDir, "tasks")),
		"result_artifact_count": countNestedJSONFiles(filepath.Join(resolved.runDir, "results")),
		"gate_artifact_count":   countJSONFiles(filepath.Join(resolved.runDir, "gates")),
	}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "run %s: %s (%d tasks, %d result artifacts, %d gate artifacts)\n", resolved.run.ID, resolved.run.State, data["task_count"], data["result_artifact_count"], data["gate_artifact_count"])
	return ExitOK
}

func countJSONFiles(root string) int {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			count++
		}
	}
	return count
}

func countNestedJSONFiles(root string) int {
	count := 0
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			count++
		}
		return nil
	})
	return count
}
