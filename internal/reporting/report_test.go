package reporting

import (
	"strings"
	"testing"
	"time"

	"github.com/lliangcol/diffdossier/internal/results"
	"github.com/lliangcol/diffdossier/internal/workflow"
)

func TestVerdictPrecedence(t *testing.T) {
	base := Input{Reviewability: "reviewable", ReviewComplete: true, Now: time.Now(), Gates: []GateEvidence{{ID: "test", Blocking: true, Status: "pass"}}}
	tests := []struct {
		name string
		edit func(*Input)
		want string
	}{
		{"ready", func(*Input) {}, "ready"},
		{"not reviewable", func(input *Input) { input.ReviewComplete = false }, "not_reviewable"},
		{"confirmation", func(input *Input) { input.NeedsConfirmation = []results.Confirmation{{ID: "C"}} }, "needs_confirmation"},
		{"finding", func(input *Input) {
			input.Findings = workflow.FindingLedger{Findings: []workflow.FindingRecord{{ID: "finding-f", Finding: results.Finding{ID: "F", Severity: "high"}, State: "confirmed"}}}
		}, "not_ready"},
		{"expired accepted risk", func(input *Input) {
			expired := input.Now.Add(-time.Minute)
			input.Findings = workflow.FindingLedger{Findings: []workflow.FindingRecord{{ID: "finding-f", Finding: results.Finding{ID: "F", Severity: "high"}, State: "accepted_risk", ExpiresAt: &expired}}}
		}, "needs_confirmation"},
		{"gate", func(input *Input) { input.Gates[0].Status = "fail" }, "not_ready"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := base
			input.Gates = append([]GateEvidence(nil), base.Gates...)
			test.edit(&input)
			if got := Build(input).Verdict; got != test.want {
				t.Fatalf("verdict=%s want=%s", got, test.want)
			}
		})
	}
}

func TestStableJSONAndMarkdownContainRequiredSections(t *testing.T) {
	report := Build(Input{RunID: "run", SnapshotID: "snap", Reviewability: "reviewable", ReviewComplete: true, Now: time.Unix(0, 0).UTC()})
	jsonOutput, err := JSON(report)
	if err != nil || !strings.Contains(string(jsonOutput), "\"human_merge_notes\": []") {
		t.Fatalf("json=%s err=%v", jsonOutput, err)
	}
	markdown := string(Markdown(report))
	for _, section := range []string{"Verdict:", "Human Merge Notes", "Evidence Limitations", "Changed-path Coverage", "Findings", "Needs Confirmation", "Gates", "Reviewer and Provider Comparisons", "Residual Risks"} {
		if !strings.Contains(markdown, section) {
			t.Fatalf("missing %s in %s", section, markdown)
		}
	}
}
