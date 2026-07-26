package results

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/lliangcol/diffdossier/internal/inventory"
	"github.com/lliangcol/diffdossier/internal/planner"
)

func fixtureTask() planner.Task {
	return planner.Task{
		ID: "task-a", SnapshotID: "snap-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Perspectives: []string{"correctness"}, RequiredPasses: 1,
		Paths: []planner.PathRef{{
			Scope: inventory.ScopeCommitted, PathBytesBase64: "YQ==", RequiredCoverage: "fully_reviewed",
		}},
	}
}

func fixtureResult() Result {
	return Result{
		SchemaVersion: "1.0", TaskID: "task-a",
		SnapshotID:    "snap-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		TaskInputHash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Reviewer: Reviewer{
			Provider: "manual", Model: "human", ModelFamily: "human",
			PassID: "correctness-1", Perspective: "correctness",
			PromptDigest:     "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			ContextIsolation: "fresh manual review",
		},
		Coverage: []Coverage{{Scope: "merge_base_to_head", PathBytesBase64: "YQ==", Status: "fully_reviewed", Evidence: "read complete previous/current blobs"}},
		Findings: []Finding{}, NeedsConfirmation: []Confirmation{}, ResidualRisks: []ResidualRisk{}, Status: "completed",
	}
}

func TestValidateCompletedResult(t *testing.T) {
	result := fixtureResult()
	validation, err := Validate(result, fixtureTask(), result.TaskInputHash, result.Reviewer.PromptDigest)
	if err != nil || !validation.Completed {
		t.Fatalf("validation=%+v err=%v", validation, err)
	}
}

func TestValidateRejectsStaleExtraAndPromotedFinding(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Result)
	}{
		{"stale", func(result *Result) {
			result.TaskInputHash = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
		}},
		{"prompt mismatch", func(result *Result) {
			result.Reviewer.PromptDigest = "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
		}},
		{"extra coverage", func(result *Result) {
			result.Coverage = append(result.Coverage, Coverage{Scope: "untracked", PathBytesBase64: "Yg==", Status: "fully_reviewed", Evidence: "x"})
		}},
		{"confirmed finding", func(result *Result) {
			result.Findings = []Finding{{ID: "F1", Evidence: "x", Impact: "x", Verification: "x", State: "confirmed"}}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := fixtureResult()
			test.edit(&result)
			if _, err := Validate(result, fixtureTask(), fixtureResult().TaskInputHash, fixtureResult().Reviewer.PromptDigest); err == nil {
				t.Fatal("invalid result was accepted")
			}
		})
	}
}

func TestParseRejectsUnknownAndTrailingJSON(t *testing.T) {
	for _, input := range []string{
		`{"schema_version":"1.0","unknown":true}`,
		`{} {}`,
	} {
		if _, err := Parse(strings.NewReader(input)); err == nil {
			t.Fatalf("invalid JSON accepted: %s", input)
		}
	}
}

func TestFindingLinePresenceVersionsAndLegacyDigestShape(t *testing.T) {
	legacy := fixtureResult()
	legacy.Findings = []Finding{{
		ID: "F1", Severity: "low", Confidence: "high", Category: "correctness",
		PathBytesBase64: "YQ==", TriggerPathBytesBase64: "YQ==",
		Evidence: "exact evidence", Impact: "bounded impact", Remediation: "apply fix",
		Verification: "run focused test", State: "reported",
	}}
	encoded, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	withoutLine := bytes.Replace(encoded, []byte(`,"line":0`), nil, 1)
	if bytes.Equal(withoutLine, encoded) {
		t.Fatalf("fixture did not contain an explicit zero line: %s", encoded)
	}
	parsedLegacy, err := Parse(bytes.NewReader(withoutLine))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Validate(parsedLegacy, fixtureTask(), legacy.TaskInputHash, legacy.Reviewer.PromptDigest); err != nil {
		t.Fatalf("legacy 1.0 result was rejected: %v", err)
	}
	remarshaled, err := json.Marshal(parsedLegacy)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(remarshaled, withoutLine) {
		t.Fatalf("legacy digest shape changed:\nwant %s\n got %s", withoutLine, remarshaled)
	}

	currentMissingLine := bytes.Replace(withoutLine, []byte(`"schema_version":"1.0"`), []byte(`"schema_version":"1.1"`), 1)
	parsedCurrent, err := Parse(bytes.NewReader(currentMissingLine))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Validate(parsedCurrent, fixtureTask(), legacy.TaskInputHash, legacy.Reviewer.PromptDigest); err == nil {
		t.Fatal("1.1 result missing line was accepted")
	}
	currentWithLine := bytes.Replace(currentMissingLine, []byte(`,"evidence":"exact evidence"`), []byte(`,"line":0,"evidence":"exact evidence"`), 1)
	parsedCurrent, err = Parse(bytes.NewReader(currentWithLine))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Validate(parsedCurrent, fixtureTask(), legacy.TaskInputHash, legacy.Reviewer.PromptDigest); err != nil {
		t.Fatalf("1.1 result with explicit zero line was rejected: %v", err)
	}
	remarshaled, err = json.Marshal(parsedCurrent)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(remarshaled, []byte(`"line":0`)) {
		t.Fatalf("explicit zero line was not preserved: %s", remarshaled)
	}
}

func TestParseRejectsNullAndUnknownFindingFields(t *testing.T) {
	for _, fragment := range []string{`"line":null`, `"line":0,"unknown":true`} {
		input := `{"schema_version":"1.1","task_id":"task-a","snapshot_id":"snap-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","task_input_hash":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","reviewer":{"provider":"manual","model":"human","model_family":"human","pass_id":"correctness-1","perspective":"correctness","prompt_digest":"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","context_isolation":"fresh manual review"},"coverage":[],"findings":[{"id":"F1","severity":"low","confidence":"high","category":"correctness","path_bytes_base64":"YQ==","trigger_path_bytes_base64":"YQ==",` + fragment + `,"evidence":"e","impact":"i","remediation":"r","verification":"v","state":"reported"}],"needs_confirmation":[],"residual_risks":[],"status":"incomplete"}`
		if _, err := Parse(strings.NewReader(input)); err == nil {
			t.Fatalf("invalid finding accepted: %s", fragment)
		}
	}
}

func TestCompareSeparatesOverlapAndUnique(t *testing.T) {
	one, two := fixtureResult(), fixtureResult()
	one.Reviewer.PassID = "one"
	two.Reviewer.PassID = "two"
	one.Reviewer.ContextIsolation = "fresh-context-one"
	two.Reviewer.ContextIsolation = "fresh-context-two"
	shared := Finding{Category: "correctness", PathBytesBase64: "YQ==", TriggerPathBytesBase64: "YQ==", Line: 1, Impact: "wrong result", Severity: "high", Confidence: "high", Remediation: "fix", Verification: "test", State: "reported"}
	one.Findings = []Finding{shared, {Category: "security", PathBytesBase64: "Yg==", Impact: "leak", State: "reported"}}
	two.Findings = []Finding{shared}
	two.Findings[0].Severity = "low"
	comparison := Compare(one, two)
	if !comparison.Independent || len(comparison.Overlap) != 1 || len(comparison.Unique["one"]) != 1 || len(comparison.Unique["two"]) != 0 || len(comparison.Disagreements) != 1 {
		t.Fatalf("comparison=%+v", comparison)
	}
}
