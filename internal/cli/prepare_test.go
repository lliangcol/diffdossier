package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lliangcol/diffdossier/internal/packets"
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

func TestPrepareBindsPublicSyntheticClassificationIntoPackets(t *testing.T) {
	repo := initializedRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "diffdossier.toml"), []byte("schema_version = 1\nbaseline = \"HEAD\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "state")
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"prepare", "--repo", repo, "--state-dir", state, "--data-class", "public_synthetic", "--json"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("prepare code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"plan", "--repo", repo, "--state-dir", state, "--json"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("plan code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	runPaths, err := filepath.Glob(filepath.Join(state, "repositories", "*", "runs", "*", "run.json"))
	if err != nil || len(runPaths) != 1 {
		t.Fatalf("run paths=%v err=%v", runPaths, err)
	}
	var run struct {
		DataClass string `json:"data_class"`
	}
	readFixtureJSON(t, runPaths[0], &run)
	if run.DataClass != "public_synthetic" {
		t.Fatalf("run data_class=%q", run.DataClass)
	}
	packetPaths, err := filepath.Glob(filepath.Join(filepath.Dir(runPaths[0]), "packets", "task-*.json"))
	if err != nil || len(packetPaths) == 0 {
		t.Fatalf("packet paths=%v err=%v", packetPaths, err)
	}
	var packet packets.Packet
	readFixtureJSON(t, packetPaths[0], &packet)
	if packet.DataClass != "public_synthetic" {
		t.Fatalf("packet data_class=%q", packet.DataClass)
	}
}

func TestPreparePublicProjectRequiresExactRevision(t *testing.T) {
	repo := initializedRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "diffdossier.toml"), []byte("schema_version = 1\nbaseline = \"HEAD\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "state")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"prepare", "--repo", repo, "--state-dir", state, "--data-class", "public_project", "--json"}, &stdout, &stderr)
	if code != ExitBlocked || !bytes.Contains(stdout.Bytes(), []byte("DD_PUBLIC_REVISION_REQUIRED")) {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
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
