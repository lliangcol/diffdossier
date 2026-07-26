// Package results validates untrusted Provider review output.
package results

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/lliangcol/diffdossier/internal/planner"
)

const MaxResultBytes = 8 * 1024 * 1024

type Reviewer struct {
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	ModelFamily      string `json:"model_family"`
	PassID           string `json:"pass_id"`
	Perspective      string `json:"perspective"`
	PromptDigest     string `json:"prompt_digest"`
	ContextIsolation string `json:"context_isolation"`
}

type Coverage struct {
	Scope           string `json:"scope"`
	PathBytesBase64 string `json:"path_bytes_base64"`
	Status          string `json:"status"`
	Evidence        string `json:"evidence"`
}

type Finding struct {
	ID                     string `json:"id"`
	Severity               string `json:"severity"`
	Confidence             string `json:"confidence"`
	Category               string `json:"category"`
	PathBytesBase64        string `json:"path_bytes_base64"`
	TriggerPathBytesBase64 string `json:"trigger_path_bytes_base64"`
	Line                   int    `json:"line,omitempty"`
	Evidence               string `json:"evidence"`
	Impact                 string `json:"impact"`
	Remediation            string `json:"remediation"`
	Verification           string `json:"verification"`
	State                  string `json:"state"`
	linePresenceKnown      bool
	linePresent            bool
}

// UnmarshalJSON records whether line was present so legacy 1.0 results that
// omitted the field remain distinguishable from current 1.1 results with an
// explicit zero line.
func (finding *Finding) UnmarshalJSON(content []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(content, &fields); err != nil {
		return err
	}
	line, present := fields["line"]
	if present && bytes.Equal(bytes.TrimSpace(line), []byte("null")) {
		return errors.New("finding line cannot be null")
	}
	type plainFinding Finding
	var decoded plainFinding
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("finding must contain exactly one JSON object")
	}
	*finding = Finding(decoded)
	finding.linePresenceKnown = true
	finding.linePresent = present
	return nil
}

// MarshalJSON preserves the omission of line in parsed legacy 1.0 results,
// while programmatic findings and parsed 1.1 findings always emit the field.
func (finding Finding) MarshalJSON() ([]byte, error) {
	type plainFinding Finding
	if finding.linePresenceKnown && !finding.linePresent {
		return json.Marshal(plainFinding(finding))
	}
	return json.Marshal(struct {
		plainFinding
		Line int `json:"line"`
	}{plainFinding: plainFinding(finding), Line: finding.Line})
}

type Confirmation struct {
	ID       string `json:"id"`
	Question string `json:"question"`
	Owner    string `json:"owner"`
}

type ResidualRisk struct {
	ID            string `json:"id"`
	Risk          string `json:"risk"`
	Owner         string `json:"owner"`
	ReviewTrigger string `json:"review_trigger"`
}

type Result struct {
	SchemaVersion     string         `json:"schema_version"`
	TaskID            string         `json:"task_id"`
	SnapshotID        string         `json:"snapshot_id"`
	TaskInputHash     string         `json:"task_input_hash"`
	Reviewer          Reviewer       `json:"reviewer"`
	Coverage          []Coverage     `json:"coverage"`
	Findings          []Finding      `json:"findings"`
	NeedsConfirmation []Confirmation `json:"needs_confirmation"`
	ResidualRisks     []ResidualRisk `json:"residual_risks"`
	Status            string         `json:"status"`
}

type Validation struct {
	Accepted  bool `json:"accepted"`
	Completed bool `json:"completed"`
}

func Parse(reader io.Reader) (Result, error) {
	limited := io.LimitReader(reader, MaxResultBytes+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return Result{}, err
	}
	if len(content) > MaxResultBytes {
		return Result{}, errors.New("result exceeds 8 MiB limit")
	}
	if !utf8.Valid(content) {
		return Result{}, errors.New("result is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var result Result
	if err := decoder.Decode(&result); err != nil {
		return Result{}, fmt.Errorf("decode result: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Result{}, errors.New("result must contain exactly one JSON object")
	}
	return result, nil
}

func Validate(result Result, task planner.Task, taskInputHash, promptDigest string) (Validation, error) {
	if result.SchemaVersion != "1.0" && result.SchemaVersion != "1.1" {
		return Validation{}, errors.New("unsupported result schema_version")
	}
	if !validDigest(result.SnapshotID, "snap-") || !validDigest(result.TaskInputHash, "sha256:") {
		return Validation{}, errors.New("result contains malformed binding digest")
	}
	if result.TaskID != task.ID || result.SnapshotID != task.SnapshotID || result.TaskInputHash != taskInputHash {
		return Validation{}, errors.New("result does not bind the current task input")
	}
	if allZeroDigest(result.SnapshotID) || allZeroDigest(result.TaskInputHash) {
		return Validation{}, errors.New("documentation-only zero digest is not a valid result binding")
	}
	if result.Reviewer.Provider == "" || result.Reviewer.PassID == "" || result.Reviewer.Perspective == "" ||
		result.Reviewer.Model == "" || result.Reviewer.ModelFamily == "" || result.Reviewer.ContextIsolation == "" {
		return Validation{}, errors.New("reviewer identity, pass, perspective, and context isolation are required")
	}
	if result.Reviewer.PromptDigest != promptDigest || !validDigest(result.Reviewer.PromptDigest, "sha256:") {
		return Validation{}, errors.New("reviewer prompt digest does not bind the packet")
	}
	if len(result.Reviewer.PassID) > 128 || strings.ContainsAny(result.Reviewer.PassID, "\x00\r\n") {
		return Validation{}, errors.New("reviewer pass_id is invalid")
	}
	if !stringIn(result.Reviewer.Perspective, task.Perspectives) {
		return Validation{}, errors.New("reviewer perspective is not required by the task")
	}
	if result.Status != "incomplete" && result.Status != "completed" {
		return Validation{}, errors.New("result status must be incomplete or completed")
	}
	if result.Coverage == nil || result.Findings == nil || result.NeedsConfirmation == nil || result.ResidualRisks == nil {
		return Validation{}, errors.New("result collection fields must be present JSON arrays")
	}
	pathSet := map[string]bool{}
	for _, path := range task.Paths {
		pathSet[path.PathBytesBase64] = true
	}
	findingIDs := map[string]bool{}
	for _, finding := range result.Findings {
		if finding.ID == "" || finding.Severity == "" || finding.Confidence == "" || finding.Category == "" ||
			finding.PathBytesBase64 == "" || finding.TriggerPathBytesBase64 == "" || finding.Evidence == "" || finding.Impact == "" ||
			finding.Remediation == "" || finding.Verification == "" {
			return Validation{}, errors.New("finding is missing a required evidence or remediation field")
		}
		if findingIDs[finding.ID] {
			return Validation{}, fmt.Errorf("duplicate finding id %q", finding.ID)
		}
		findingIDs[finding.ID] = true
		if !pathSet[finding.PathBytesBase64] || !pathSet[finding.TriggerPathBytesBase64] {
			return Validation{}, errors.New("finding path or trigger path is outside the task")
		}
		if finding.State != "reported" && finding.State != "needs_confirmation" {
			return Validation{}, fmt.Errorf("Provider cannot import finding state %q", finding.State)
		}
		if !stringIn(finding.Severity, []string{"critical", "high", "medium", "low", "info"}) ||
			!stringIn(finding.Confidence, []string{"high", "medium", "low"}) {
			return Validation{}, errors.New("finding severity or confidence is invalid")
		}
		if result.SchemaVersion == "1.1" && finding.linePresenceKnown && !finding.linePresent {
			return Validation{}, errors.New("result schema_version 1.1 requires finding line")
		}
		if finding.Line < 0 {
			return Validation{}, errors.New("finding line cannot be negative")
		}
	}
	confirmationIDs := map[string]bool{}
	for _, confirmation := range result.NeedsConfirmation {
		if confirmation.ID == "" || confirmation.Question == "" || confirmation.Owner == "" || confirmationIDs[confirmation.ID] {
			return Validation{}, errors.New("needs_confirmation entries require unique id, question, and owner")
		}
		confirmationIDs[confirmation.ID] = true
	}
	residualIDs := map[string]bool{}
	for _, residual := range result.ResidualRisks {
		if residual.ID == "" || residual.Risk == "" || residual.Owner == "" || residual.ReviewTrigger == "" || residualIDs[residual.ID] {
			return Validation{}, errors.New("residual_risks entries require unique id, risk, owner, and review_trigger")
		}
		residualIDs[residual.ID] = true
	}
	expected := map[string]string{}
	for _, path := range task.Paths {
		expected[coverageKey(string(path.Scope), path.PathBytesBase64)] = path.RequiredCoverage
	}
	seen := map[string]bool{}
	complete := true
	for _, coverage := range result.Coverage {
		key := coverageKey(coverage.Scope, coverage.PathBytesBase64)
		required, ok := expected[key]
		if !ok {
			return Validation{}, fmt.Errorf("coverage includes path outside task: %s", key)
		}
		if seen[key] {
			return Validation{}, fmt.Errorf("duplicate coverage path: %s", key)
		}
		seen[key] = true
		if !validCoverageStatus(coverage.Status) || coverage.Evidence == "" {
			return Validation{}, fmt.Errorf("invalid coverage evidence for %s", key)
		}
		if !coverageSatisfies(required, coverage.Status) {
			complete = false
		}
	}
	if len(seen) != len(expected) {
		complete = false
	}
	if result.Status == "completed" && !complete {
		return Validation{}, errors.New("completed result does not satisfy exact required coverage")
	}
	return Validation{Accepted: true, Completed: result.Status == "completed" && complete}, nil
}

func validCoverageStatus(status string) bool {
	switch status {
	case "fully_reviewed", "mechanically_verified", "invariant_checked", "sampled", "excluded_with_reason", "not_reviewed":
		return true
	default:
		return false
	}
}

func coverageSatisfies(required, actual string) bool {
	switch required {
	case "fully_reviewed":
		return actual == "fully_reviewed"
	case "mechanically_verified":
		return actual == "mechanically_verified" || actual == "invariant_checked" || actual == "fully_reviewed"
	case "invariant_checked":
		return actual == "invariant_checked" || actual == "fully_reviewed"
	default:
		return false
	}
}

func allZeroDigest(value string) bool {
	index := strings.LastIndexByte(value, ':')
	if index < 0 {
		index = strings.LastIndexByte(value, '-')
	}
	if index < 0 || index+1 >= len(value) {
		return false
	}
	for _, char := range value[index+1:] {
		if char != '0' {
			return false
		}
	}
	return true
}

func validDigest(value, prefix string) bool {
	if len(value) != len(prefix)+64 || !strings.HasPrefix(value, prefix) {
		return false
	}
	_, err := hex.DecodeString(value[len(prefix):])
	return err == nil
}

func coverageKey(scope, path string) string {
	return scope + "\x00" + path
}

func stringIn(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

type Comparison struct {
	SchemaVersion        string              `json:"schema_version"`
	PassIDs              []string            `json:"pass_ids"`
	Reviewers            []Reviewer          `json:"reviewers"`
	Independent          bool                `json:"independent"`
	IndependenceEvidence []string            `json:"independence_evidence"`
	Overlap              []string            `json:"overlap"`
	Unique               map[string][]string `json:"unique"`
	Disagreements        []string            `json:"disagreements"`
}

func Compare(inputs ...Result) Comparison {
	comparison := Comparison{
		SchemaVersion: "1.0", PassIDs: []string{}, Overlap: []string{},
		Reviewers: []Reviewer{}, IndependenceEvidence: []string{},
		Unique: map[string][]string{}, Disagreements: []string{},
	}
	counts := map[string]int{}
	variants := map[string]map[string]bool{}
	contexts := map[string]bool{}
	promptDigest := ""
	independent := len(inputs) > 1
	for _, result := range inputs {
		passID := result.Reviewer.PassID
		comparison.PassIDs = append(comparison.PassIDs, passID)
		comparison.Reviewers = append(comparison.Reviewers, result.Reviewer)
		contextKey := result.Reviewer.Provider + "\x00" + result.Reviewer.ModelFamily + "\x00" + result.Reviewer.ContextIsolation
		if contexts[contextKey] || result.Reviewer.PromptDigest == "" {
			independent = false
		}
		contexts[contextKey] = true
		if promptDigest == "" {
			promptDigest = result.Reviewer.PromptDigest
		} else if promptDigest != result.Reviewer.PromptDigest {
			independent = false
		}
		comparison.IndependenceEvidence = append(comparison.IndependenceEvidence, result.Reviewer.PassID+":"+contextKey)
		seen := map[string]bool{}
		for _, finding := range result.Findings {
			fingerprint := findingFingerprint(finding)
			if !seen[fingerprint] {
				counts[fingerprint]++
				seen[fingerprint] = true
			}
			if variants[fingerprint] == nil {
				variants[fingerprint] = map[string]bool{}
			}
			variants[fingerprint][findingVariant(finding)] = true
		}
	}
	sort.Strings(comparison.PassIDs)
	sort.Strings(comparison.IndependenceEvidence)
	comparison.Independent = independent
	sort.Slice(comparison.Reviewers, func(i, j int) bool { return comparison.Reviewers[i].PassID < comparison.Reviewers[j].PassID })
	for fingerprint, count := range counts {
		if count == len(inputs) && len(inputs) > 1 {
			comparison.Overlap = append(comparison.Overlap, fingerprint)
		}
		if len(variants[fingerprint]) > 1 {
			comparison.Disagreements = append(comparison.Disagreements, fingerprint)
		}
	}
	for _, result := range inputs {
		passID := result.Reviewer.PassID
		comparison.Unique[passID] = []string{}
		seenUnique := map[string]bool{}
		for _, finding := range result.Findings {
			fingerprint := findingFingerprint(finding)
			if counts[fingerprint] == 1 && !seenUnique[fingerprint] {
				comparison.Unique[passID] = append(comparison.Unique[passID], fingerprint)
				seenUnique[fingerprint] = true
			}
		}
		sort.Strings(comparison.Unique[passID])
	}
	sort.Strings(comparison.Overlap)
	sort.Strings(comparison.Disagreements)
	return comparison
}

func findingFingerprint(finding Finding) string {
	value := strings.Join([]string{
		finding.Category, finding.PathBytesBase64, finding.TriggerPathBytesBase64, fmt.Sprint(finding.Line),
		strings.ToLower(strings.Join(strings.Fields(finding.Impact), " ")),
	}, "\x00")
	digest := sha256Bytes([]byte(value))
	return fmt.Sprintf("finding:%x", digest[:12])
}

func findingVariant(finding Finding) string {
	value := strings.Join([]string{
		finding.Severity, finding.Confidence, finding.State,
		strings.ToLower(strings.Join(strings.Fields(finding.Remediation), " ")),
		strings.ToLower(strings.Join(strings.Fields(finding.Verification), " ")),
	}, "\x00")
	digest := sha256Bytes([]byte(value))
	return fmt.Sprintf("variant:%x", digest[:12])
}

func sha256Bytes(value []byte) [32]byte {
	return sha256.Sum256(value)
}
