package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPrepareThenPlanModelFree(t *testing.T) {
	repo := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "fixture@example.invalid"}, {"config", "user.name", "Fixture"}} {
		runPlanGit(t, repo, args...)
	}
	if err := os.WriteFile(filepath.Join(repo, "api.go"), []byte("package fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runPlanGit(t, repo, "add", "api.go")
	runPlanGit(t, repo, "commit", "-qm", "initial")
	if err := os.WriteFile(filepath.Join(repo, "api.go"), []byte("package fixture\n// changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "diffdossier.toml"), []byte("schema_version = 1\nbaseline = \"HEAD\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "state")
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"prepare", "--repo", repo, "--state-dir", state, "--json"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("prepare code=%d stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"plan", "--repo", repo, "--state-dir", state, "--json"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("plan code=%d stderr=%s", code, stderr.String())
	}
	var envelope struct {
		Status string         `json:"status"`
		Data   map[string]any `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Status != "ok" || envelope.Data["task_count"].(float64) < 1 {
		t.Fatalf("unexpected plan: %s", stdout.String())
	}
	if matches, _ := filepath.Glob(filepath.Join(state, "repositories", "*", "runs", "*", "packets", "*.json")); len(matches) == 0 {
		t.Fatal("manual packets were not persisted")
	}
}

func runPlanGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
