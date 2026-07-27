package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lliangcol/diffdossier/internal/packets"
	"github.com/lliangcol/diffdossier/internal/planner"
	"github.com/lliangcol/diffdossier/internal/results"
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

func TestRecordTaskCompletesEveryRequiredPass(t *testing.T) {
	repo, state, runDir := plannedFixture(t)
	taskPaths, err := filepath.Glob(filepath.Join(runDir, "tasks", "*.json"))
	if err != nil || len(taskPaths) == 0 {
		t.Fatalf("task paths=%v err=%v", taskPaths, err)
	}
	for _, taskPath := range taskPaths {
		var task planner.Task
		readFixtureJSON(t, taskPath, &task)
		var packet packets.Packet
		readFixtureJSON(t, filepath.Join(runDir, "packets", task.ID+".json"), &packet)
		for passIndex, perspective := range task.Perspectives {
			coverage := make([]results.Coverage, 0, len(task.Paths))
			for _, path := range task.Paths {
				coverage = append(coverage, results.Coverage{
					Scope: string(path.Scope), PathBytesBase64: path.PathBytesBase64,
					Status: path.RequiredCoverage, Evidence: "fixture reviewed exact current and previous blobs",
				})
			}
			result := results.Result{
				SchemaVersion: "1.0", TaskID: task.ID, SnapshotID: task.SnapshotID,
				TaskInputHash: packet.TaskInputHash,
				Reviewer: results.Reviewer{
					Provider: "manual", Model: "human-fixture", ModelFamily: "human",
					PassID: task.ID + "-" + perspective, Perspective: perspective,
					PromptDigest: packet.PromptDigest, ContextIsolation: "independent fixture pass " + perspective,
				},
				Coverage: coverage, Findings: []results.Finding{},
				NeedsConfirmation: []results.Confirmation{}, ResidualRisks: []results.ResidualRisk{},
				Status: "completed",
			}
			resultPath := filepath.Join(t.TempDir(), "result.json")
			writeFixtureJSON(t, resultPath, result)
			var stdout, stderr bytes.Buffer
			code := Run([]string{
				"record", "task", "--repo", repo, "--state-dir", state,
				"--task-id", task.ID, "--result", resultPath, "--json",
			}, &stdout, &stderr)
			if code != ExitOK {
				t.Fatalf("pass=%d code=%d stderr=%s stdout=%s", passIndex, code, stderr.String(), stdout.String())
			}
		}
	}
	var run struct {
		State string `json:"state"`
	}
	readFixtureJSON(t, filepath.Join(runDir, "run.json"), &run)
	if run.State != "REVIEWED" {
		t.Fatalf("run state=%s, want REVIEWED", run.State)
	}
	if _, err := os.Stat(filepath.Join(runDir, "results", "index.json")); err != nil {
		t.Fatalf("result index missing: %v", err)
	}
}

func TestRecordBatchCompletesEveryRequiredPass(t *testing.T) {
	repo, state, runDir := plannedFixture(t)
	taskPaths, err := filepath.Glob(filepath.Join(runDir, "tasks", "*.json"))
	if err != nil || len(taskPaths) == 0 {
		t.Fatalf("task paths=%v err=%v", taskPaths, err)
	}
	entries := []map[string]string{}
	for _, taskPath := range taskPaths {
		var task planner.Task
		readFixtureJSON(t, taskPath, &task)
		var packet packets.Packet
		readFixtureJSON(t, filepath.Join(runDir, "packets", task.ID+".json"), &packet)
		for passIndex, perspective := range task.Perspectives {
			coverage := make([]results.Coverage, 0, len(task.Paths))
			for _, path := range task.Paths {
				coverage = append(coverage, results.Coverage{
					Scope: string(path.Scope), PathBytesBase64: path.PathBytesBase64,
					Status: path.RequiredCoverage, Evidence: "batch fixture reviewed exact current and previous blobs",
				})
			}
			result := results.Result{
				SchemaVersion: "1.0", TaskID: task.ID, SnapshotID: task.SnapshotID,
				TaskInputHash: packet.TaskInputHash,
				Reviewer: results.Reviewer{
					Provider: "manual", Model: "human-fixture", ModelFamily: "human",
					PassID: task.ID + "-batch-" + perspective, Perspective: perspective,
					PromptDigest: packet.PromptDigest, ContextIsolation: "independent batch pass " + perspective,
				},
				Coverage: coverage, Findings: []results.Finding{},
				NeedsConfirmation: []results.Confirmation{}, ResidualRisks: []results.ResidualRisk{},
				Status: "completed",
			}
			resultPath := filepath.Join(t.TempDir(), fmt.Sprintf("result-%d.json", passIndex))
			writeFixtureJSON(t, resultPath, result)
			entries = append(entries, map[string]string{"task_id": task.ID, "result_path": resultPath})
		}
	}
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	writeFixtureJSON(t, manifestPath, map[string]any{"schema_version": "1.0", "results": entries})
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"record", "batch", "--repo", repo, "--state-dir", state,
		"--manifest", manifestPath, "--json",
	}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var run struct {
		State string `json:"state"`
	}
	readFixtureJSON(t, filepath.Join(runDir, "run.json"), &run)
	if run.State != "REVIEWED" {
		t.Fatalf("run state=%s, want REVIEWED", run.State)
	}
	var index results.Index
	readFixtureJSON(t, filepath.Join(runDir, "results", "index.json"), &index)
	if len(index.Records) != len(entries) {
		t.Fatalf("records=%d, want %d", len(index.Records), len(entries))
	}
}

func TestRecordTaskRejectsStaleSnapshotBeforeImport(t *testing.T) {
	repo, state, runDir := plannedFixture(t)
	taskPaths, _ := filepath.Glob(filepath.Join(runDir, "tasks", "*.json"))
	var task planner.Task
	readFixtureJSON(t, taskPaths[0], &task)
	resultPath := filepath.Join(t.TempDir(), "result.json")
	writeFixtureJSON(t, resultPath, map[string]any{})
	if err := os.WriteFile(filepath.Join(repo, "reviewed.txt"), []byte("mutated\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"record", "task", "--repo", repo, "--state-dir", state,
		"--task-id", task.ID, "--result", resultPath, "--json",
	}, &stdout, &stderr)
	if code != ExitStale || !bytes.Contains(stdout.Bytes(), []byte("DD_SNAPSHOT_STALE")) {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
}

func TestPacketTaskRejectsStaleSnapshotBeforeDisclosure(t *testing.T) {
	repo, state, runDir := plannedFixture(t)
	taskPaths, _ := filepath.Glob(filepath.Join(runDir, "tasks", "*.json"))
	var task planner.Task
	readFixtureJSON(t, taskPaths[0], &task)
	if err := os.WriteFile(filepath.Join(repo, "reviewed.txt"), []byte("mutated\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"packet", "task", "--repo", repo, "--state-dir", state,
		"--task-id", task.ID, "--json",
	}, &stdout, &stderr)
	if code != ExitStale || !bytes.Contains(stdout.Bytes(), []byte("DD_SNAPSHOT_STALE")) {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
}

func TestRecordTaskRejectsTamperedPlanBeforeImport(t *testing.T) {
	repo, state, runDir := plannedFixture(t)
	taskPaths, _ := filepath.Glob(filepath.Join(runDir, "tasks", "*.json"))
	var task planner.Task
	readFixtureJSON(t, taskPaths[0], &task)
	var plan planner.Plan
	planPath := filepath.Join(runDir, "plan.json")
	readFixtureJSON(t, planPath, &plan)
	plan.SnapshotID = "snap-bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	writeFixtureJSON(t, planPath, plan)
	resultPath := filepath.Join(t.TempDir(), "result.json")
	writeFixtureJSON(t, resultPath, map[string]any{})
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"record", "task", "--repo", repo, "--state-dir", state,
		"--task-id", task.ID, "--result", resultPath, "--json",
	}, &stdout, &stderr)
	if code != ExitEvidence || !bytes.Contains(stdout.Bytes(), []byte("DD_PLAN_INTEGRITY")) {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
}

func plannedFixture(t *testing.T) (string, string, string) {
	t.Helper()
	repo := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "fixture@example.invalid"}, {"config", "user.name", "Fixture"}} {
		runPlanGit(t, repo, args...)
	}
	if err := os.WriteFile(filepath.Join(repo, "reviewed.txt"), []byte("before\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runPlanGit(t, repo, "add", "reviewed.txt")
	runPlanGit(t, repo, "commit", "-qm", "initial")
	if err := os.WriteFile(filepath.Join(repo, "reviewed.txt"), []byte("after\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := "schema_version = 1\nbaseline = \"HEAD\"\ninclude_untracked = false\n"
	if err := os.WriteFile(filepath.Join(repo, "diffdossier.toml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "state")
	for _, args := range [][]string{
		{"prepare", "--repo", repo, "--state-dir", state, "--json"},
		{"plan", "--repo", repo, "--state-dir", state, "--json"},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code != ExitOK {
			t.Fatalf("%v code=%d stderr=%s stdout=%s", args, code, stderr.String(), stdout.String())
		}
	}
	runDirs, err := filepath.Glob(filepath.Join(state, "repositories", "*", "runs", "*"))
	if err != nil || len(runDirs) != 1 {
		t.Fatalf("run dirs=%v err=%v", runDirs, err)
	}
	return repo, state, runDirs[0]
}

func readFixtureJSON(t *testing.T, path string, target any) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, target); err != nil {
		t.Fatal(err)
	}
}

func writeFixtureJSON(t *testing.T, path string, value any) {
	t.Helper()
	content, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
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
