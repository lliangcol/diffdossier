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
	SchemaVersion string            `json:"schema_version"`
	RunDigest     string            `json:"run_digest"`
	Files         map[string]string `json:"files"`
}

// Portable builds a deterministic private archive. Locks, logs, and trust or
// approval records are excluded so an export cannot silently transfer authority.
func Portable(runDir string) ([]byte, PortableManifest, error) {
	files := map[string][]byte{}
	digests := map[string]string{}
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
		files[rel] = content
		digests[rel] = sha(content)
		return nil
	})
	if err != nil {
		return nil, PortableManifest{}, err
	}
	manifest := PortableManifest{SchemaVersion: "1.0", Files: digests}
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
	PolicyDigest         string               `json:"policy_digest"`
	ScanDigest           string               `json:"scan_digest"`
	ApprovalRecordDigest string               `json:"public_export_approval_record_digest"`
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
	return PublicBundle{SchemaVersion: "1.0", Content: append([]byte(nil), preparation.PreparedContent...), ContentDigest: preparation.Candidate.Digest, ArtifactClass: preparation.Candidate.ArtifactClass, PolicyDigest: preparation.Candidate.PolicyDigest, ScanDigest: preparation.Candidate.ScanDigest, ApprovalRecordDigest: sha(approvalBytes)}, nil
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
