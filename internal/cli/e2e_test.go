package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/lliangcol/diffdossier/internal/packets"
	"github.com/lliangcol/diffdossier/internal/planner"
	"github.com/lliangcol/diffdossier/internal/results"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

func TestModelFreeLifecycleFinalizesAndExports(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "config", "user.email", "fixture@example.invalid")
	runGit(t, repo, "config", "user.name", "Fixture")
	configContent := "schema_version = 1\nbaseline = \"HEAD~1\"\ninclude_untracked = true\n\n[review]\nmax_files_per_task = 8\nmax_packet_bytes = 250000\ndefault_provider = \"manual\"\n\n[state]\nretention_days = 30\n\n[risk]\npolicy_files = []\n"
	if err := os.WriteFile(filepath.Join(repo, "diffdossier.toml"), []byte(configContent), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-qm", "base")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("one\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-qm", "change")
	state := filepath.Join(t.TempDir(), "state")
	prepare := runJSONCommand(t, []string{"prepare", "--repo", repo, "--state-dir", state, "--json"})
	runID := prepare["run_id"].(string)
	runDir := prepare["state_path"].(string)
	runJSONCommand(t, []string{"plan", "--repo", repo, "--state-dir", state, "--run-id", runID, "--json"})
	var plan planner.Plan
	readJSONFile(t, filepath.Join(runDir, "plan.json"), &plan)
	if len(plan.Tasks) == 0 {
		t.Fatal("expected review tasks")
	}
	for _, task := range plan.Tasks {
		packet, err := packets.Build(task, publicschema.PrivateProject)
		if err != nil {
			t.Fatal(err)
		}
		for pass := 0; pass < task.RequiredPasses; pass++ {
			coverage := []results.Coverage{}
			for _, path := range task.Paths {
				coverage = append(coverage, results.Coverage{Scope: string(path.Scope), PathBytesBase64: path.PathBytesBase64, Status: path.RequiredCoverage, Evidence: "model-free fixture coverage"})
			}
			review := results.Result{SchemaVersion: "1.0", TaskID: task.ID, SnapshotID: task.SnapshotID, TaskInputHash: packet.TaskInputHash, Reviewer: results.Reviewer{Provider: "manual", Model: "fixture", ModelFamily: "fixture-" + strconv.Itoa(pass), PassID: "pass-" + strconv.Itoa(pass), Perspective: task.Perspectives[pass], PromptDigest: packet.PromptDigest, ContextIsolation: "isolated-" + strconv.Itoa(pass)}, Coverage: coverage, Findings: []results.Finding{}, NeedsConfirmation: []results.Confirmation{}, ResidualRisks: []results.ResidualRisk{}, Status: "completed"}
			path := filepath.Join(t.TempDir(), "result.json")
			content, _ := json.Marshal(review)
			if err := os.WriteFile(path, content, 0o600); err != nil {
				t.Fatal(err)
			}
			runJSONCommand(t, []string{"record", "task", "--repo", repo, "--state-dir", state, "--run-id", runID, "--task-id", task.ID, "--result", path, "--json"})
		}
	}
	runJSONCommand(t, []string{"gates", "plan", "--repo", repo, "--state-dir", state, "--run-id", runID, "--json"})
	verified := runJSONCommand(t, []string{"verify", "--repo", repo, "--state-dir", state, "--run-id", runID, "--json"})
	if verified["state"] != "REREVIEWED" {
		t.Fatalf("verify state=%v", verified["state"])
	}
	finalized := runJSONCommand(t, []string{"finalize", "--repo", repo, "--state-dir", state, "--run-id", runID, "--json"})
	if finalized["state"] != "FINALIZED" {
		t.Fatalf("final state=%v", finalized["state"])
	}
	output := filepath.Join(t.TempDir(), "portable.zip")
	runJSONCommand(t, []string{"export", "portable", "--repo", repo, "--state-dir", state, "--run-id", runID, "--output", output, "--json"})
	if info, err := os.Stat(output); err != nil || info.Size() == 0 {
		t.Fatalf("portable export: %v", err)
	}
}

func runJSONCommand(t *testing.T, args []string) map[string]any {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("command %s code=%d stderr=%s stdout=%s", strings.Join(args, " "), code, stderr.String(), stdout.String())
	}
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}
func readJSONFile(t *testing.T, path string, target any) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, target); err != nil {
		t.Fatal(err)
	}
}
