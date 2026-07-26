// Package reporting builds deterministic verdicts from verified evidence.
package reporting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lliangcol/diffdossier/internal/results"
	"github.com/lliangcol/diffdossier/internal/workflow"
)

type GateEvidence struct {
	ID       string `json:"id"`
	Blocking bool   `json:"blocking"`
	Status   string `json:"status"`
	Digest   string `json:"digest"`
}

type Input struct {
	RunID               string
	SnapshotID          string
	Baseline            string
	Head                string
	Worktree            string
	ReviewComplete      bool
	Reviewability       string
	Coverage            map[string]int
	Findings            workflow.FindingLedger
	NeedsConfirmation   []results.Confirmation
	ResidualRisks       []results.ResidualRisk
	Gates               []GateEvidence
	Comparisons         []results.Comparison
	HumanMergeNotes     []string
	EvidenceLimitations []string
	Now                 time.Time
}

type Report struct {
	SchemaVersion       string                   `json:"schema_version"`
	Verdict             string                   `json:"verdict"`
	RunID               string                   `json:"run_id"`
	SnapshotID          string                   `json:"snapshot_id"`
	Baseline            string                   `json:"baseline"`
	Head                string                   `json:"head"`
	Worktree            string                   `json:"worktree"`
	Reviewability       string                   `json:"reviewability"`
	Coverage            map[string]int           `json:"changed_path_coverage"`
	Findings            []workflow.FindingRecord `json:"findings"`
	NeedsConfirmation   []results.Confirmation   `json:"needs_confirmation"`
	Gates               []GateEvidence           `json:"gates"`
	ReviewerComparisons []results.Comparison     `json:"reviewer_comparisons"`
	HumanMergeNotes     []string                 `json:"human_merge_notes"`
	ResidualRisks       []results.ResidualRisk   `json:"residual_risks"`
	EvidenceLimitations []string                 `json:"evidence_limitations"`
	GeneratedAt         time.Time                `json:"generated_at"`
}

func Build(input Input) Report {
	report := Report{
		SchemaVersion: "1.0", RunID: input.RunID, SnapshotID: input.SnapshotID,
		Baseline: input.Baseline, Head: input.Head, Worktree: input.Worktree,
		Reviewability: input.Reviewability, Coverage: nonNilMap(input.Coverage),
		Findings:            append([]workflow.FindingRecord(nil), input.Findings.Findings...),
		NeedsConfirmation:   nonNilConfirmations(input.NeedsConfirmation),
		Gates:               append([]GateEvidence(nil), input.Gates...),
		ReviewerComparisons: append([]results.Comparison(nil), input.Comparisons...),
		HumanMergeNotes:     nonNilStrings(input.HumanMergeNotes),
		ResidualRisks:       nonNilResiduals(input.ResidualRisks),
		EvidenceLimitations: nonNilStrings(input.EvidenceLimitations),
		GeneratedAt:         input.Now.UTC(),
	}
	sort.Slice(report.Findings, func(i, j int) bool { return report.Findings[i].ID < report.Findings[j].ID })
	sort.Slice(report.Gates, func(i, j int) bool { return report.Gates[i].ID < report.Gates[j].ID })
	sort.Strings(report.HumanMergeNotes)
	sort.Strings(report.EvidenceLimitations)
	report.Verdict = verdict(input, report)
	return report
}

func verdict(input Input, report Report) string {
	if input.Reviewability != "reviewable" || !input.ReviewComplete || len(report.EvidenceLimitations) > 0 {
		return "not_reviewable"
	}
	if len(report.NeedsConfirmation) > 0 {
		return "needs_confirmation"
	}
	for _, finding := range report.Findings {
		if finding.State == "needs_confirmation" || (finding.State == "accepted_risk" && (finding.ExpiresAt == nil || !input.Now.Before(*finding.ExpiresAt))) {
			return "needs_confirmation"
		}
		if (finding.Finding.Severity == "critical" || finding.Finding.Severity == "high") &&
			finding.State != "rejected" && finding.State != "verified_closed" && finding.State != "accepted_risk" {
			return "not_ready"
		}
	}
	for _, gate := range report.Gates {
		if gate.Blocking && gate.Status != "pass" {
			return "not_ready"
		}
	}
	return "ready"
}

func JSON(report Report) ([]byte, error) {
	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}

func Markdown(report Report) []byte {
	var output bytes.Buffer
	fmt.Fprintf(&output, "# DiffDossier report\n\nVerdict: **%s**\n\n", report.Verdict)
	fmt.Fprintf(&output, "- Run: `%s`\n- Snapshot: `%s`\n- Baseline: `%s`\n- HEAD: `%s`\n- Worktree: `%s`\n- Reviewability: `%s`\n",
		report.RunID, report.SnapshotID, report.Baseline, report.Head, report.Worktree, report.Reviewability)
	writeSection(&output, "Human Merge Notes", report.HumanMergeNotes)
	writeSection(&output, "Evidence Limitations", report.EvidenceLimitations)
	fmt.Fprintf(&output, "\n## Changed-path Coverage\n\n")
	coverageKeys := make([]string, 0, len(report.Coverage))
	for key := range report.Coverage {
		coverageKeys = append(coverageKeys, key)
	}
	sort.Strings(coverageKeys)
	if len(coverageKeys) == 0 {
		output.WriteString("- None recorded.\n")
	}
	for _, key := range coverageKeys {
		fmt.Fprintf(&output, "- %s: %d\n", key, report.Coverage[key])
	}
	fmt.Fprintf(&output, "\n## Findings\n\n")
	if len(report.Findings) == 0 {
		output.WriteString("- None.\n")
	} else {
		for _, finding := range report.Findings {
			fmt.Fprintf(&output, "- %s (provider=%s) [%s/%s] %s: %s\n", finding.ID, finding.Finding.ID, finding.Finding.Severity, finding.State, finding.Finding.Category, finding.Finding.Impact)
		}
	}
	fmt.Fprintf(&output, "\n## Needs Confirmation\n\n")
	if len(report.NeedsConfirmation) == 0 {
		output.WriteString("- None.\n")
	} else {
		for _, item := range report.NeedsConfirmation {
			fmt.Fprintf(&output, "- %s [%s]: %s\n", item.ID, item.Owner, item.Question)
		}
	}
	fmt.Fprintf(&output, "\n## Gates\n\n")
	if len(report.Gates) == 0 {
		output.WriteString("- None recorded.\n")
	} else {
		for _, gate := range report.Gates {
			fmt.Fprintf(&output, "- %s: %s (blocking=%t, digest=%s)\n", gate.ID, gate.Status, gate.Blocking, gate.Digest)
		}
	}
	fmt.Fprintf(&output, "\n## Reviewer and Provider Comparisons\n\n")
	if len(report.ReviewerComparisons) == 0 {
		output.WriteString("- None recorded.\n")
	} else {
		for _, comparison := range report.ReviewerComparisons {
			fmt.Fprintf(&output, "- Passes %s: independent=%t, overlap=%d, disagreements=%d\n", strings.Join(comparison.PassIDs, ", "), comparison.Independent, len(comparison.Overlap), len(comparison.Disagreements))
		}
	}
	fmt.Fprintf(&output, "\n## Residual Risks\n\n")
	if len(report.ResidualRisks) == 0 {
		output.WriteString("- None.\n")
	} else {
		for _, risk := range report.ResidualRisks {
			fmt.Fprintf(&output, "- %s [%s]: %s (review: %s)\n", risk.ID, risk.Owner, risk.Risk, risk.ReviewTrigger)
		}
	}
	return output.Bytes()
}

func writeSection(output *bytes.Buffer, title string, values []string) {
	fmt.Fprintf(output, "\n## %s\n\n", title)
	if len(values) == 0 {
		output.WriteString("- None.\n")
		return
	}
	for _, value := range values {
		fmt.Fprintf(output, "- %s\n", strings.TrimSpace(value))
	}
}

func nonNilMap(value map[string]int) map[string]int {
	if value == nil {
		return map[string]int{}
	}
	return value
}

func nonNilStrings(value []string) []string {
	if value == nil {
		return []string{}
	}
	return append([]string(nil), value...)
}

func nonNilConfirmations(value []results.Confirmation) []results.Confirmation {
	if value == nil {
		return []results.Confirmation{}
	}
	return append([]results.Confirmation(nil), value...)
}

func nonNilResiduals(value []results.ResidualRisk) []results.ResidualRisk {
	if value == nil {
		return []results.ResidualRisk{}
	}
	return append([]results.ResidualRisk(nil), value...)
}
