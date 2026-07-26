package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lliangcol/diffdossier/internal/packets"
	"github.com/lliangcol/diffdossier/internal/planner"
	"github.com/lliangcol/diffdossier/internal/policy"
	"github.com/lliangcol/diffdossier/internal/providers"
	"github.com/lliangcol/diffdossier/internal/results"
	"github.com/lliangcol/diffdossier/internal/snapshot"
	"github.com/lliangcol/diffdossier/internal/store"
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
	contractPacket := runJSONCommand(t, []string{"packet", "contract", "--repo", repo, "--state-dir", state, "--run-id", runID, "--json"})
	if contractPacket["packet"].(map[string]any)["input_digest"] == "" {
		t.Fatal("contract packet has no input digest")
	}
	runJSONCommand(t, []string{"record", "contract", "--repo", repo, "--state-dir", state, "--run-id", runID, "--json"})
	var plan planner.Plan
	readJSONFile(t, filepath.Join(runDir, "plan.json"), &plan)
	if len(plan.Tasks) == 0 {
		t.Fatal("expected review tasks")
	}
	status := runJSONCommand(t, []string{"status", "--repo", repo, "--state-dir", state, "--run-id", runID, "--json"})
	if status["state"] != "CONTRACTED" || status["task_count"] != float64(len(plan.Tasks)) {
		t.Fatalf("status=%+v", status)
	}
	firstTask := plan.Tasks[0]
	runJSONCommand(t, []string{"review", "run", "--repo", repo, "--state-dir", state, "--run-id", runID, "--task-id", firstTask.ID, "--provider", "manual", "--json"})
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		t.Fatal(err)
	}
	providerDir := t.TempDir()
	providerMarker := filepath.Join(t.TempDir(), "provider-executed")
	t.Setenv("GO_WANT_CLI_PROVIDER_HELPER", "1")
	t.Setenv("DD_CLI_PROVIDER_MARKER", providerMarker)
	providerArgs := []string{
		"review", "run", "--repo", repo, "--state-dir", state, "--run-id", runID,
		"--task-id", firstTask.ID, "--provider", "command",
		"--executable", executable, "--arg", "-test.run=TestCLICommandProviderHelper",
		"--provider-cwd", providerDir, "--env", "GO_WANT_CLI_PROVIDER_HELPER",
		"--env", "DD_CLI_PROVIDER_MARKER", "--network-destination-class", "none",
		"--credential-source", "none", "--json",
	}
	commandPlanOutput := runJSONCommand(t, providerArgs)
	if commandPlanOutput["executed"] != false {
		t.Fatalf("untrusted command Provider executed: %+v", commandPlanOutput)
	}
	if _, err := os.Stat(providerMarker); !os.IsNotExist(err) {
		t.Fatalf("untrusted Provider created marker: %v", err)
	}
	encodedPlan, _ := json.Marshal(commandPlanOutput["command_plan"])
	var commandPlan providers.CommandPlan
	if err := json.Unmarshal(encodedPlan, &commandPlan); err != nil {
		t.Fatal(err)
	}
	trust := commandPlan.TrustCandidate
	trust.ExpiresAt = time.Now().Add(time.Hour)
	egress := policy.EgressGrant{
		Provider: commandPlan.EgressRequest.Provider, SnapshotID: commandPlan.EgressRequest.SnapshotID,
		TaskInputDigest: commandPlan.EgressRequest.TaskInputDigest, DataClass: commandPlan.EgressRequest.DataClass,
		MaxBytes: commandPlan.EgressRequest.Bytes, ExpiresAt: time.Now().Add(time.Hour),
	}
	trustPath := filepath.Join(t.TempDir(), "trust.json")
	egressPath := filepath.Join(t.TempDir(), "egress.json")
	writeJSONFile(t, trustPath, trust)
	writeJSONFile(t, egressPath, egress)
	authorizedArgs := append(append([]string{}, providerArgs[:len(providerArgs)-1]...),
		"--trust-execution-plan", commandPlan.ExecutionPlanDigest,
		"--trust-binding", trustPath, "--egress-grant", egressPath, "--json")
	runJSONCommand(t, authorizedArgs)
	if _, err := os.Stat(providerMarker); err != nil {
		t.Fatalf("authorized Provider did not execute: %v", err)
	}
	for _, task := range plan.Tasks {
		taskPacket := runJSONCommand(t, []string{"packet", "task", "--repo", repo, "--state-dir", state, "--run-id", runID, "--task-id", task.ID, "--json"})
		if taskPacket["packet"].(map[string]any)["task_id"] != task.ID {
			t.Fatalf("task packet=%+v", taskPacket)
		}
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
	var attempts providers.AttemptLedger
	readJSONFile(t, filepath.Join(runDir, "reviews", "attempts.json"), &attempts)
	if len(attempts.Attempts) < 5 || attempts.Attempts[0].Provider != "manual" {
		t.Fatalf("Provider switching ledger=%+v", attempts)
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
	archived := runJSONCommand(t, []string{"run", "archive", "--repo", repo, "--state-dir", state, "--run-id", runID, "--reason", "E2E retention", "--json"})
	archive := archived["archive"].(map[string]any)
	if archive["run_state"] != "EXPORTED" {
		t.Fatalf("archive state=%v", archive["run_state"])
	}
	planned := runJSONCommand(t, []string{"gc", "plan", "--repo", repo, "--state-dir", state, "--as-of", "2100-01-01T00:00:00Z", "--json"})
	gcPlan := planned["plan"].(map[string]any)
	executed := runJSONCommand(t, []string{"gc", "run", "--state-dir", state, "--trust-gc-plan", gcPlan["plan_digest"].(string), "--json"})
	gcExecution := executed["execution"].(map[string]any)
	if gcExecution["removed_runs"] != float64(1) {
		t.Fatalf("GC execution=%+v", gcExecution)
	}
	if _, err := os.Stat(runDir); !os.IsNotExist(err) {
		t.Fatalf("archived run still exists after GC: %v", err)
	}
}

func TestCLICommandProviderHelper(t *testing.T) {
	if os.Getenv("GO_WANT_CLI_PROVIDER_HELPER") != "1" {
		return
	}
	if marker := os.Getenv("DD_CLI_PROVIDER_MARKER"); marker != "" {
		_ = os.WriteFile(marker, []byte("executed"), 0o600)
	}
	var request struct {
		ProtocolVersion string          `json:"protocol_version"`
		Operation       string          `json:"operation"`
		Packet          *packets.Packet `json:"packet,omitempty"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&request); err != nil {
		os.Exit(2)
	}
	if request.Operation == "handshake" {
		_ = json.NewEncoder(os.Stdout).Encode(publicschema.ProviderHandshake{
			ProtocolVersion: "1.0", Provider: "cli-fixture",
			Capabilities:  []string{"review", "structured-result"},
			MaxInputBytes: 1 << 20, SupportsResume: true, NetworkAccess: "none",
		})
		os.Exit(0)
	}
	if request.Packet == nil {
		os.Exit(3)
	}
	coverage := make([]results.Coverage, 0, len(request.Packet.Task.Paths))
	for _, path := range request.Packet.Task.Paths {
		coverage = append(coverage, results.Coverage{
			Scope: string(path.Scope), PathBytesBase64: path.PathBytesBase64,
			Status: path.RequiredCoverage, Evidence: "authorized command fixture",
		})
	}
	_ = json.NewEncoder(os.Stdout).Encode(results.Result{
		SchemaVersion: "1.0", TaskID: request.Packet.TaskID,
		SnapshotID: request.Packet.SnapshotID, TaskInputHash: request.Packet.TaskInputHash,
		Reviewer: results.Reviewer{
			Provider: "cli-fixture", Model: "fixture", ModelFamily: "command-fixture",
			PassID: "command-pass", Perspective: request.Packet.Task.Perspectives[0],
			PromptDigest: request.Packet.PromptDigest, ContextIsolation: "fresh command process",
		},
		Coverage: coverage, Findings: []results.Finding{},
		NeedsConfirmation: []results.Confirmation{}, ResidualRisks: []results.ResidualRisk{},
		Status: "incomplete",
	})
	os.Exit(0)
}

func TestPublicExportCLIRequiresExactG12Plans(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	repo, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "state")
	stateStore, err := store.Open(state)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := stateStore.Register(repo)
	if err != nil {
		t.Fatal(err)
	}
	run, runDir, err := stateStore.BeginRun(repository, snapshot.Seal{SchemaVersion: "1.0", SnapshotID: "snap-public-cli"})
	if err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(t.TempDir(), "public-input.txt")
	if err := os.WriteFile(input, []byte("synthetic public summary\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	prepared := runJSONCommand(t, []string{
		"export", "public", "prepare", "--repo", repo, "--state-dir", state, "--run-id", run.ID,
		"--input", input, "--class", "public_synthetic", "--action", "create",
		"--policy-digest", "sha256:" + strings.Repeat("a", 64), "--json",
	})
	preparationDigest := prepared["preparation_digest"].(string)
	approvalPlan := runJSONCommand(t, []string{
		"export", "public", "approve", "--repo", repo, "--state-dir", state, "--run-id", run.ID,
		"--preparation-digest", preparationDigest, "--operator", "fixture-owner", "--json",
	})
	if approvalPlan["approved"] != false {
		t.Fatalf("approval dry-run mutated state: %+v", approvalPlan)
	}
	if matches, _ := filepath.Glob(filepath.Join(runDir, "approvals", "public-*")); len(matches) != 0 {
		t.Fatalf("approval existed before G12 trust: %+v", matches)
	}
	approved := runJSONCommand(t, []string{
		"export", "public", "approve", "--repo", repo, "--state-dir", state, "--run-id", run.ID,
		"--preparation-digest", preparationDigest, "--operator", "fixture-owner",
		"--trust-public-approval", approvalPlan["approval_plan_digest"].(string), "--json",
	})
	approvalDigest := approved["approval_digest"].(string)
	bundleOutput := filepath.Join(t.TempDir(), "public-bundle.json")
	createPlan := runJSONCommand(t, []string{
		"export", "public", "create", "--repo", repo, "--state-dir", state, "--run-id", run.ID,
		"--preparation-digest", preparationDigest, "--approval-digest", approvalDigest,
		"--output", bundleOutput, "--json",
	})
	if _, err := os.Stat(bundleOutput); !os.IsNotExist(err) {
		t.Fatalf("bundle existed before G12 trust: %v", err)
	}
	created := runJSONCommand(t, []string{
		"export", "public", "create", "--repo", repo, "--state-dir", state, "--run-id", run.ID,
		"--preparation-digest", preparationDigest, "--approval-digest", approvalDigest,
		"--output", bundleOutput, "--trust-public-create", createPlan["create_plan_digest"].(string), "--json",
	})
	bundleDigest := created["bundle_digest"].(string)
	approvalRecordDigest := created["approval_record_digest"].(string)
	bundleBytes, err := os.ReadFile(bundleOutput)
	if err != nil {
		t.Fatal(err)
	}
	for _, denied := range []string{"fixture-owner", repository.ID, run.ID, "approved_by"} {
		if bytes.Contains(bundleBytes, []byte(denied)) {
			t.Fatalf("public bundle leaked %q: %s", denied, bundleBytes)
		}
	}
	tombstoneOutput := filepath.Join(t.TempDir(), "public-tombstone.json")
	revokePlan := runJSONCommand(t, []string{
		"export", "public", "revoke", "--repo", repo, "--state-dir", state, "--run-id", run.ID,
		"--approval-digest", approvalRecordDigest, "--export-digest", bundleDigest,
		"--reason", "fixture withdrawal", "--output", tombstoneOutput, "--json",
	})
	if _, err := os.Stat(tombstoneOutput); !os.IsNotExist(err) {
		t.Fatalf("tombstone existed before G12 trust: %v", err)
	}
	revoked := runJSONCommand(t, []string{
		"export", "public", "revoke", "--repo", repo, "--state-dir", state, "--run-id", run.ID,
		"--approval-digest", approvalRecordDigest, "--export-digest", bundleDigest,
		"--reason", "fixture withdrawal", "--output", tombstoneOutput,
		"--trust-public-revoke", revokePlan["revoke_plan_digest"].(string), "--json",
	})
	if revoked["external_copies_recalled"] != false {
		t.Fatalf("revocation overclaimed recall: %+v", revoked)
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

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	content, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
}
