package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPrepareWritesStateOutsideRepository(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "config", "user.email", "fixture@example.invalid")
	runGit(t, repo, "config", "user.name", "Fixture")
	if err := os.WriteFile(filepath.Join(repo, "file.txt"), []byte("one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "file.txt")
	runGit(t, repo, "commit", "-qm", "initial")
	if err := os.WriteFile(filepath.Join(repo, "diffdossier.toml"), []byte("schema_version = 1\nbaseline = \"HEAD\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "state")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"prepare", "--repo", repo, "--state-dir", state, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var envelope struct {
		Status string         `json:"status"`
		Data   map[string]any `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Status != "ok" || envelope.Data["snapshot_id"] == "" {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(repo, ".diffdossier")); !os.IsNotExist(err) {
		t.Fatalf("prepare wrote runtime state into target repository: %v", err)
	}
}

func TestPrepareUsesEnvironmentPathsAndEffectiveConfigChangesGoStale(t *testing.T) {
	repo := initializedRepo(t)
	configPath := filepath.Join(t.TempDir(), "selected.toml")
	if err := os.WriteFile(configPath, []byte("baseline = \"HEAD\"\ninclude_untracked = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "state")
	t.Setenv("DIFFDOSSIER_CONFIG", configPath)
	t.Setenv("DIFFDOSSIER_STATE_DIR", state)
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"prepare", "--repo", repo, "--json"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("prepare code=%d stderr=%s", code, stderr.String())
	}
	if err := os.WriteFile(configPath, []byte("baseline = \"HEAD\"\ninclude_untracked = false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"plan", "--repo", repo, "--json"}, &stdout, &stderr); code != ExitStale {
		t.Fatalf("plan code=%d want=%d stdout=%s stderr=%s", code, ExitStale, stdout.String(), stderr.String())
	}
}

func TestBaselineFlagMustBeRepeatedAcrossFreshnessChecks(t *testing.T) {
	repo := initializedRepo(t)
	configPath := filepath.Join(t.TempDir(), "without-baseline.toml")
	if err := os.WriteFile(configPath, []byte("schema_version = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "state")
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"prepare", "--repo", repo, "--config", configPath, "--baseline", "HEAD", "--state-dir", state, "--json"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("prepare code=%d stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"plan", "--repo", repo, "--config", configPath, "--baseline", "HEAD", "--state-dir", state, "--json"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("plan code=%d stderr=%s", code, stderr.String())
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
