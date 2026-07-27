// Package exporter creates deterministic private archives and prepares public
// derivative candidates. Approval, creation, and revocation remain separate
// content-bound operations.
package exporter

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lliangcol/diffdossier/internal/policy"
	"github.com/lliangcol/diffdossier/internal/redact"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

type PortableManifest struct {
	SchemaVersion      string            `json:"schema_version"`
	RunDigest          string            `json:"run_digest"`
	Files              map[string]string `json:"files"`
	PathSanitizedFiles []string          `json:"path_sanitized_files,omitempty"`
}

// Portable builds a deterministic private archive. Locks, logs, and trust or
// approval records are excluded so an export cannot silently transfer authority.
func Portable(runDir string) ([]byte, PortableManifest, error) {
	files := map[string][]byte{}
	digests := map[string]string{}
	pathSanitized := []string{}
	err := filepath.WalkDir(runDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(runDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "locks/") || strings.HasPrefix(rel, "logs/") || strings.HasPrefix(rel, "approvals/") || strings.Contains(rel, "trust") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content, sanitized, err := sanitizePortablePaths(rel, content)
		if err != nil {
			return err
		}
		if sanitized {
			pathSanitized = append(pathSanitized, rel)
		}
		files[rel] = content
		digests[rel] = sha(content)
		return nil
	})
	if err != nil {
		return nil, PortableManifest{}, err
	}
	sort.Strings(pathSanitized)
	manifest := PortableManifest{SchemaVersion: "1.0", Files: digests, PathSanitizedFiles: pathSanitized}
	manifest.RunDigest = digestMap(digests)
	encoded, _ := json.MarshalIndent(manifest, "", "  ")
	files["portable-manifest.json"] = append(encoded, '\n')
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	fixed := time.Unix(0, 0).UTC()
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetModTime(fixed)
		header.SetMode(0o600)
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return nil, PortableManifest{}, err
		}
		if _, err := writer.Write(files[name]); err != nil {
			return nil, PortableManifest{}, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, PortableManifest{}, err
	}
	return output.Bytes(), manifest, nil
}

func sanitizePortablePaths(name string, content []byte) ([]byte, bool, error) {
	if strings.HasSuffix(name, ".json") {
		var value any
		if err := json.Unmarshal(content, &value); err != nil {
			return nil, false, err
		}
		sanitized := sanitizePortableValue(value, "")
		if !sanitized {
			return content, false, nil
		}
		encoded, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return nil, false, err
		}
		return append(encoded, '\n'), true, nil
	}
	if strings.HasSuffix(name, ".jsonl") {
		lines := bytes.Split(content, []byte{'\n'})
		sanitized := false
		for index, line := range lines {
			if len(bytes.TrimSpace(line)) == 0 {
				continue
			}
			var value any
			if err := json.Unmarshal(line, &value); err != nil {
				return nil, false, err
			}
			if sanitizePortableValue(value, "") {
				sanitized = true
				encoded, err := json.Marshal(value)
				if err != nil {
					return nil, false, err
				}
				lines[index] = encoded
			}
		}
		if sanitized {
			return bytes.Join(lines, []byte{'\n'}), true, nil
		}
	}
	return content, false, nil
}

func sanitizePortableValue(value any, contextKey string) bool {
	sanitized := false
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			text, ok := item.(string)
			if ok && isPortablePathField(key) && isAbsolutePortablePath(text) {
				replacement := "[portable-path]"
				if key == "cwd" && typed["cwd_class"] == "repository" {
					replacement = "."
				} else if key == "executable" {
					if requested, requestedOK := typed["requested_executable"].(string); requestedOK && !isAbsolutePortablePath(requested) {
						replacement = requested
					}
				}
				typed[key] = replacement
				sanitized = true
				continue
			}
			if sanitizePortableValue(item, key) {
				sanitized = true
			}
		}
	case []any:
		for index, item := range typed {
			text, ok := item.(string)
			if ok && isPortablePathList(contextKey) && isAbsolutePortablePath(text) {
				typed[index] = "[portable-path]"
				sanitized = true
				continue
			}
			if sanitizePortableValue(item, contextKey) {
				sanitized = true
			}
		}
	}
	return sanitized
}

func isPortablePathField(key string) bool {
	switch key {
	case "cwd", "executable", "requested_executable", "path", "repo", "repository_root", "working_directory":
		return true
	}
	return strings.HasSuffix(key, "_path") || strings.HasSuffix(key, "_dir") || strings.HasSuffix(key, "_directory") || strings.HasSuffix(key, "_root")
}

func isPortablePathList(key string) bool {
	switch key {
	case "argv", "expected_writes", "paths":
		return true
	default:
		return false
	}
}

func isAbsolutePortablePath(value string) bool {
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\\`) {
		return true
	}
	return len(value) >= 3 && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) && value[1] == ':' && (value[2] == '/' || value[2] == '\\')
}

type ScanFinding struct {
	Rule   string `json:"rule"`
	Offset int    `json:"offset"`
}
type PublicPreparation struct {
	SchemaVersion   string                 `json:"schema_version"`
	Candidate       policy.PublicCandidate `json:"candidate"`
	ScanFindings    []ScanFinding          `json:"scan_findings"`
	PreparedContent []byte                 `json:"-"`
}

func PreparePublic(content []byte, dataClass publicschema.DataClass, action, policyDigest, publicRevision, redactionApprovalDigest string) (PublicPreparation, error) {
	if action != "create" && action != "replace" {
		return PublicPreparation{}, errors.New("public action must be create or replace")
	}
	artifactClass := policy.ArtifactClass(dataClass)
	if dataClass == publicschema.SecretDenied {
		return PublicPreparation{}, errors.New("secret_denied cannot produce a public derivative")
	}
	if dataClass == publicschema.PrivateProject {
		artifactClass = policy.ArtifactRedactedSummary
		if redactionApprovalDigest == "" {
			return PublicPreparation{}, errors.New("private_project derivative requires redaction approval")
		}
	}
	if artifactClass != policy.ArtifactPublicSynthetic && artifactClass != policy.ArtifactPublicProject && artifactClass != policy.ArtifactRedactedSummary {
		return PublicPreparation{}, errors.New("unsupported public artifact class")
	}
	if artifactClass == policy.ArtifactPublicProject && strings.TrimSpace(publicRevision) == "" {
		return PublicPreparation{}, errors.New("public_project requires confirmed public revision")
	}
	secretFindings, scanErr := redact.Scan(content)
	if scanErr != nil {
		return PublicPreparation{}, scanErr
	}
	findings := make([]ScanFinding, 0, len(secretFindings))
	for _, finding := range secretFindings {
		findings = append(findings, ScanFinding{Rule: finding.Rule, Offset: finding.Offset})
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Offset != findings[j].Offset {
			return findings[i].Offset < findings[j].Offset
		}
		return findings[i].Rule < findings[j].Rule
	})
	scanBytes, _ := json.Marshal(findings)
	candidate := policy.PublicCandidate{Digest: sha(content), ArtifactClass: artifactClass, Action: action, PolicyDigest: policyDigest, ScanDigest: sha(scanBytes), PublicRevision: publicRevision, RedactionApprovalDigest: redactionApprovalDigest}
	return PublicPreparation{SchemaVersion: "1.0", Candidate: candidate, ScanFindings: findings, PreparedContent: append([]byte(nil), content...)}, nil
}

type PublicBundle struct {
	SchemaVersion        string               `json:"schema_version"`
	Content              []byte               `json:"content"`
	ContentDigest        string               `json:"content_digest"`
	ArtifactClass        policy.ArtifactClass `json:"artifact_class"`
	Action               string               `json:"action"`
	PolicyDigest         string               `json:"policy_digest"`
	ScanDigest           string               `json:"scan_digest"`
	PublicRevision       string               `json:"public_revision,omitempty"`
	ApprovalRecordDigest string               `json:"public_export_approval_record_digest"`
	BundleDigest         string               `json:"bundle_digest"`
}

func CreatePublic(preparation PublicPreparation, approval policy.PublicApproval, redactionApproval *policy.RedactionApproval) (PublicBundle, error) {
	if len(preparation.ScanFindings) > 0 {
		return PublicBundle{}, errors.New("public candidate scan has findings")
	}
	if err := approval.Authorizes(preparation.Candidate); err != nil {
		return PublicBundle{}, err
	}
	if preparation.Candidate.ArtifactClass == policy.ArtifactRedactedSummary {
		if redactionApproval == nil {
			return PublicBundle{}, errors.New("redacted summary requires private redaction approval")
		}
		if err := redactionApproval.Authorizes(preparation.Candidate); err != nil {
			return PublicBundle{}, err
		}
	}
	approvalBytes, _ := json.Marshal(approval)
	bundle := PublicBundle{
		SchemaVersion: "1.0", Content: append([]byte(nil), preparation.PreparedContent...),
		ContentDigest: preparation.Candidate.Digest, ArtifactClass: preparation.Candidate.ArtifactClass,
		Action: preparation.Candidate.Action, PolicyDigest: preparation.Candidate.PolicyDigest,
		ScanDigest: preparation.Candidate.ScanDigest, PublicRevision: preparation.Candidate.PublicRevision,
		ApprovalRecordDigest: sha(approvalBytes),
	}
	bundle.BundleDigest = digestJSON(bundle)
	return bundle, nil
}

func VerifyPublicBundle(bundle PublicBundle) error {
	claimed := bundle.BundleDigest
	bundle.BundleDigest = ""
	if claimed == "" || claimed != digestJSON(bundle) {
		return errors.New("public bundle integrity is invalid")
	}
	if bundle.ContentDigest != sha(bundle.Content) || bundle.Action == "" ||
		bundle.ApprovalRecordDigest == "" || bundle.PolicyDigest == "" || bundle.ScanDigest == "" {
		return errors.New("public bundle content binding is invalid")
	}
	return nil
}

type Revocation struct {
	SchemaVersion        string    `json:"schema_version"`
	ApprovalRecordDigest string    `json:"approval_record_digest"`
	ExportDigest         string    `json:"export_digest"`
	Reason               string    `json:"reason"`
	RevokedAt            time.Time `json:"revoked_at"`
	TombstoneDigest      string    `json:"tombstone_digest"`
}

func Revoke(approvalDigest, exportDigest, reason string, now time.Time) (Revocation, error) {
	if approvalDigest == "" || exportDigest == "" || strings.TrimSpace(reason) == "" {
		return Revocation{}, errors.New("revocation requires approval, export, and reason")
	}
	value := Revocation{SchemaVersion: "1.0", ApprovalRecordDigest: approvalDigest, ExportDigest: exportDigest, Reason: reason, RevokedAt: now.UTC()}
	encoded, _ := json.Marshal(value)
	value.TombstoneDigest = sha(encoded)
	return value, nil
}

func PreparationDigest(preparation PublicPreparation) string {
	return digestJSON(struct {
		SchemaVersion string                 `json:"schema_version"`
		Candidate     policy.PublicCandidate `json:"candidate"`
		ScanFindings  []ScanFinding          `json:"scan_findings"`
	}{preparation.SchemaVersion, preparation.Candidate, preparation.ScanFindings})
}

func ApprovalPlanDigest(preparation PublicPreparation, approvedBy string) string {
	return digestJSON(struct {
		Operation         string `json:"operation"`
		PreparationDigest string `json:"preparation_digest"`
		ApprovedBy        string `json:"approved_by"`
	}{"approve", PreparationDigest(preparation), approvedBy})
}

func CreatePlanDigest(preparation PublicPreparation, approval policy.PublicApproval) string {
	return digestJSON(struct {
		Operation         string `json:"operation"`
		PreparationDigest string `json:"preparation_digest"`
		ApprovalDigest    string `json:"approval_digest"`
	}{"create", PreparationDigest(preparation), approval.Digest})
}

func RevocationPlanDigest(approvalDigest, exportDigest, reason string) string {
	return digestJSON(struct {
		Operation      string `json:"operation"`
		ApprovalDigest string `json:"approval_digest"`
		ExportDigest   string `json:"export_digest"`
		Reason         string `json:"reason"`
	}{"revoke", approvalDigest, exportDigest, reason})
}

func sha(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func digestMap(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hash := sha256.New()
	for _, key := range keys {
		hash.Write([]byte(key))
		hash.Write([]byte{0})
		hash.Write([]byte(values[key]))
		hash.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func digestJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return sha(encoded)
}
