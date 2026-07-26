package adapters

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lliangcol/diffdossier/internal/inventory"
	"github.com/lliangcol/diffdossier/internal/packets"
	"github.com/lliangcol/diffdossier/internal/planner"
	processrunner "github.com/lliangcol/diffdossier/internal/process"
	"github.com/lliangcol/diffdossier/internal/results"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
	"github.com/lliangcol/diffdossier/schemas"
)

func TestReviewResultSchemaMeetsStructuredOutputSubset(t *testing.T) {
	content, err := schemas.Read("review-result.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var root any
	if err := json.Unmarshal(content, &root); err != nil {
		t.Fatal(err)
	}
	assertStructuredOutputSchema(t, "$", root)
}

func TestCodexAdapterUsesIsolatedReadonlyInvocationAndBindsResult(t *testing.T) {
	args, request, packet := adapterFixture(t, "codex")
	var calls []processrunner.Spec
	invoke := func(_ context.Context, spec processrunner.Spec) (processrunner.Output, error) {
		calls = append(calls, spec)
		if len(calls) == 1 {
			return processrunner.Output{Stdout: []byte("fixture-cli 1.0\n")}, nil
		}
		return processrunner.Output{Stdout: validProviderResult(t, packet)}, nil
	}
	var stdout, stderr bytes.Buffer
	if code := run(args, bytes.NewReader(request), &stdout, &stderr, invoke); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if len(calls) != 2 {
		t.Fatalf("calls=%d", len(calls))
	}
	joined := "|" + strings.Join(calls[1].Args, "|") + "|"
	for _, required := range []string{"|exec|", "|--ephemeral|", "|--sandbox|read-only|", "|--ignore-user-config|", "|--ignore-rules|", "|--skip-git-repo-check|", "|approval_policy=\"never\"|"} {
		if !strings.Contains(joined, required) {
			t.Errorf("Codex args missing %s: %v", required, calls[1].Args)
		}
	}
	for _, forbidden := range []string{"dangerously", "full-auto", "bypass"} {
		if strings.Contains(strings.ToLower(joined), forbidden) {
			t.Errorf("Codex args contain forbidden capability %q: %v", forbidden, calls[1].Args)
		}
	}
	if calls[1].Dir == "" || !filepath.IsAbs(calls[1].Dir) || !bytes.Contains(calls[1].Stdin, []byte("PACKET_JSON")) {
		t.Fatalf("downstream execution was not external-cwd and packet-only")
	}
	var result results.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Reviewer.Provider != "codex" || result.Reviewer.Model != "fixture-model" || result.Reviewer.PassID != "pass-1" || result.Reviewer.PromptDigest != packet.PromptDigest {
		t.Fatalf("trusted reviewer binding missing: %+v", result.Reviewer)
	}
}

func TestClaudeAdapterUsesBareAPIKeyModeAndStructuredOutput(t *testing.T) {
	args, request, packet := adapterFixture(t, "claude-code")
	var calls []processrunner.Spec
	invoke := func(_ context.Context, spec processrunner.Spec) (processrunner.Output, error) {
		calls = append(calls, spec)
		if len(calls) == 1 {
			return processrunner.Output{Stdout: []byte("fixture-cli 1.0\n")}, nil
		}
		result := validProviderResult(t, packet)
		return processrunner.Output{Stdout: []byte(`{"type":"result","structured_output":` + string(result) + `}`)}, nil
	}
	var stdout, stderr bytes.Buffer
	if code := run(args, bytes.NewReader(request), &stdout, &stderr, invoke); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	joined := "|" + strings.Join(calls[1].Args, "|") + "|"
	for _, required := range []string{"|--bare|", "|--output-format|json|", "|--tools||", "|--permission-mode|dontAsk|", "|--no-session-persistence|", "|--max-turns|1|", "|--max-budget-usd|0.25|"} {
		if !strings.Contains(joined, required) {
			t.Errorf("Claude args missing %s: %v", required, calls[1].Args)
		}
	}
	if strings.Contains(joined, "dangerously-skip-permissions") {
		t.Fatalf("Claude invocation bypassed permissions: %v", calls[1].Args)
	}
}

func TestAdapterRejectsDigestAndVersionDrift(t *testing.T) {
	args, request, _ := adapterFixture(t, "codex")
	badDigest := append([]string(nil), args...)
	for index := range badDigest {
		if badDigest[index] == "--cli-digest" {
			badDigest[index+1] = "sha256:" + strings.Repeat("0", 64)
		}
	}
	calls := 0
	invoke := func(_ context.Context, _ processrunner.Spec) (processrunner.Output, error) {
		calls++
		return processrunner.Output{Stdout: []byte("wrong version\n")}, nil
	}
	var stdout, stderr bytes.Buffer
	if code := run(badDigest, bytes.NewReader(request), &stdout, &stderr, invoke); code == 0 || calls != 0 {
		t.Fatalf("digest drift code=%d calls=%d", code, calls)
	}
	stdout.Reset()
	stderr.Reset()
	if code := run(args, bytes.NewReader(request), &stdout, &stderr, invoke); code == 0 || calls != 1 {
		t.Fatalf("version drift code=%d calls=%d stderr=%s", code, calls, stderr.String())
	}
}

func adapterFixture(t *testing.T, provider string) ([]string, []byte, packets.Packet) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		t.Fatal(err)
	}
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	schemaPath := filepath.Clean(filepath.Join(filepath.Dir(current), "..", "..", "schemas", "review-result.schema.json"))
	task := planner.Task{
		SchemaVersion: "1.0", ID: "task-adapter", SnapshotID: "snap-" + strings.Repeat("a", 64),
		Paths:           []planner.PathRef{{Scope: inventory.ScopeUnstaged, PathBytesBase64: "Zml4dHVyZS5nbw==", DisplayPath: "fixture.go", RequiredCoverage: "fully_reviewed"}},
		DependencyTasks: []string{}, ContractTypes: []string{}, Perspectives: []string{"correctness"}, RequiredPasses: 1,
	}
	packet, err := packets.Build(task, publicschema.PublicSynthetic)
	if err != nil {
		t.Fatal(err)
	}
	request, err := json.Marshal(commandRequest{ProtocolVersion: "1.0", Operation: "review", Packet: &packet})
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"--provider", provider, "--cli", executable, "--cli-digest", fileSHA256(t, executable), "--cli-version", "fixture-cli 1.0", "--schema", schemaPath, "--schema-digest", fileSHA256(t, schemaPath), "--model", "fixture-model", "--pass-id", "pass-1", "--perspective", "correctness"}
	return args, request, packet
}

func validProviderResult(t *testing.T, packet packets.Packet) []byte {
	t.Helper()
	result := results.Result{
		SchemaVersion: "1.0", TaskID: packet.TaskID, SnapshotID: packet.SnapshotID, TaskInputHash: packet.TaskInputHash,
		Reviewer: results.Reviewer{Provider: "untrusted", Model: "untrusted", ModelFamily: "untrusted", PassID: "untrusted", Perspective: "correctness", PromptDigest: packet.PromptDigest, ContextIsolation: "untrusted"},
		Coverage: []results.Coverage{{Scope: string(packet.Task.Paths[0].Scope), PathBytesBase64: packet.Task.Paths[0].PathBytesBase64, Status: "fully_reviewed", Evidence: "reviewed exact synthetic packet"}},
		Findings: []results.Finding{}, NeedsConfirmation: []results.Confirmation{}, ResidualRisks: []results.ResidualRisk{}, Status: "completed",
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func assertStructuredOutputSchema(t *testing.T, path string, value any) {
	t.Helper()
	node, ok := value.(map[string]any)
	if !ok {
		return
	}
	if _, hasConst := node["const"]; hasConst && node["type"] == nil {
		t.Errorf("%s has const without type", path)
	}
	if properties, ok := node["properties"].(map[string]any); ok {
		if node["type"] != "object" || node["additionalProperties"] != false {
			t.Errorf("%s object must set type=object and additionalProperties=false", path)
		}
		required := map[string]bool{}
		if list, ok := node["required"].([]any); ok {
			for _, item := range list {
				name, _ := item.(string)
				required[name] = true
			}
		}
		for name, property := range properties {
			if !required[name] {
				t.Errorf("%s property %q is not required", path, name)
			}
			assertStructuredOutputSchema(t, path+".properties."+name, property)
		}
	}
	if items, ok := node["items"]; ok {
		assertStructuredOutputSchema(t, path+".items", items)
	}
}
