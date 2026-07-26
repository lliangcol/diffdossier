// Package policy enforces data classification and content-bound authorization.
package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lliangcol/diffdossier/pkg/schema"
)

var classRank = map[schema.DataClass]int{
	schema.PublicSynthetic: 0,
	schema.PublicProject:   1,
	schema.PrivateProject:  2,
	schema.SecretDenied:    3,
}

func MostRestrictive(classes ...schema.DataClass) (schema.DataClass, error) {
	if len(classes) == 0 {
		return schema.PrivateProject, nil
	}
	selected := schema.PublicSynthetic
	for _, class := range classes {
		rank, ok := classRank[class]
		if !ok {
			return "", fmt.Errorf("unknown data class %q", class)
		}
		if rank > classRank[selected] {
			selected = class
		}
	}
	return selected, nil
}

type TrustBinding struct {
	RepositoryID        string    `json:"repository_id"`
	SnapshotID          string    `json:"snapshot_id"`
	TaskInputDigest     string    `json:"task_input_digest"`
	ExecutionPlanDigest string    `json:"execution_plan_digest"`
	ConfigDigest        string    `json:"config_digest"`
	BinaryDigest        string    `json:"binary_digest"`
	Capability          string    `json:"capability"`
	ExpiresAt           time.Time `json:"expires_at"`
}

func (binding TrustBinding) Authorizes(candidate TrustBinding, now time.Time) bool {
	if binding.ExpiresAt.IsZero() || !now.Before(binding.ExpiresAt) {
		return false
	}
	return binding.RepositoryID == candidate.RepositoryID &&
		binding.SnapshotID == candidate.SnapshotID &&
		binding.TaskInputDigest == candidate.TaskInputDigest &&
		binding.ExecutionPlanDigest == candidate.ExecutionPlanDigest &&
		binding.ConfigDigest == candidate.ConfigDigest &&
		binding.BinaryDigest == candidate.BinaryDigest &&
		binding.Capability == candidate.Capability
}

type EgressRequest struct {
	Provider        string           `json:"provider"`
	SnapshotID      string           `json:"snapshot_id"`
	TaskInputDigest string           `json:"task_input_digest"`
	DataClass       schema.DataClass `json:"data_class"`
	Bytes           int64            `json:"bytes"`
}

type EgressGrant struct {
	Provider        string           `json:"provider"`
	SnapshotID      string           `json:"snapshot_id"`
	TaskInputDigest string           `json:"task_input_digest"`
	DataClass       schema.DataClass `json:"data_class"`
	MaxBytes        int64            `json:"max_bytes"`
	ExpiresAt       time.Time        `json:"expires_at"`
}

func (grant EgressGrant) Authorizes(request EgressRequest, now time.Time) bool {
	if request.DataClass == schema.SecretDenied || grant.ExpiresAt.IsZero() || !now.Before(grant.ExpiresAt) {
		return false
	}
	return grant.Provider == request.Provider &&
		grant.SnapshotID == request.SnapshotID &&
		grant.TaskInputDigest == request.TaskInputDigest &&
		grant.DataClass == request.DataClass &&
		request.Bytes >= 0 && request.Bytes <= grant.MaxBytes
}

type PublicCandidate struct {
	Digest                  string        `json:"digest"`
	ArtifactClass           ArtifactClass `json:"artifact_class"`
	Action                  string        `json:"action"`
	PolicyDigest            string        `json:"policy_digest"`
	ScanDigest              string        `json:"scan_digest"`
	PublicRevision          string        `json:"public_revision,omitempty"`
	RedactionApprovalDigest string        `json:"redaction_approval_digest,omitempty"`
}

type ArtifactClass string

const (
	ArtifactPublicSynthetic ArtifactClass = "public_synthetic"
	ArtifactPublicProject   ArtifactClass = "public_project"
	ArtifactRedactedSummary ArtifactClass = "redacted_summary"
)

type PublicApproval struct {
	Binding    schema.ApprovalBinding `json:"binding"`
	ApprovedBy string                 `json:"approved_by"`
	Digest     string                 `json:"digest"`
	Revoked    bool                   `json:"revoked"`
}

func NewPublicApproval(candidate PublicCandidate, approvedBy string, now time.Time) (PublicApproval, error) {
	if approvedBy == "" {
		return PublicApproval{}, errors.New("public approval requires approver")
	}
	approval := PublicApproval{Binding: schema.ApprovalBinding{SchemaVersion: "1.0", CandidateDigest: candidate.Digest, DataClass: schema.DataClass(candidate.ArtifactClass), Action: candidate.Action, PolicyDigest: candidate.PolicyDigest, ScanDigest: candidate.ScanDigest, RedactionApprovalDigest: candidate.RedactionApprovalDigest, ApprovedAt: now.UTC().Format(time.RFC3339Nano)}, ApprovedBy: approvedBy}
	approval.Digest = digestApproval(approval)
	return approval, nil
}

func (approval PublicApproval) Authorizes(candidate PublicCandidate) error {
	if approval.Revoked {
		return errors.New("public export approval is revoked")
	}
	if approval.Binding.SchemaVersion != "1.0" || approval.ApprovedBy == "" || approval.Digest == "" || approval.Digest != digestApproval(PublicApproval{Binding: approval.Binding, ApprovedBy: approval.ApprovedBy}) {
		return errors.New("public export approval integrity is invalid")
	}
	if _, err := time.Parse(time.RFC3339Nano, approval.Binding.ApprovedAt); err != nil {
		return errors.New("public export approval time is invalid")
	}
	if candidate.ArtifactClass != ArtifactPublicSynthetic &&
		candidate.ArtifactClass != ArtifactPublicProject &&
		candidate.ArtifactClass != ArtifactRedactedSummary {
		return errors.New("artifact class is not publicly exportable")
	}
	if candidate.ArtifactClass == ArtifactPublicProject && candidate.PublicRevision == "" {
		return errors.New("public_project requires a bound public revision")
	}
	if candidate.ArtifactClass == ArtifactRedactedSummary && candidate.RedactionApprovalDigest == "" {
		return errors.New("redacted_summary requires a private redaction approval digest")
	}
	if approval.Binding.CandidateDigest != candidate.Digest ||
		string(approval.Binding.DataClass) != string(candidate.ArtifactClass) ||
		approval.Binding.Action != candidate.Action ||
		approval.Binding.PolicyDigest != candidate.PolicyDigest ||
		approval.Binding.ScanDigest != candidate.ScanDigest || approval.Binding.RedactionApprovalDigest != candidate.RedactionApprovalDigest {
		return errors.New("public export approval does not exactly match candidate")
	}
	return nil
}

type RedactionApproval struct {
	SchemaVersion           string    `json:"schema_version"`
	SourceRunDigest         string    `json:"source_run_digest"`
	DerivedContentDigest    string    `json:"derived_content_digest"`
	RedactionManifestDigest string    `json:"redaction_manifest_digest"`
	ScanDigest              string    `json:"scan_digest"`
	ApprovedBy              string    `json:"approved_by"`
	ApprovedAt              time.Time `json:"approved_at"`
	Digest                  string    `json:"digest"`
}

func NewRedactionApproval(sourceRunDigest, derivedContentDigest, manifestDigest, scanDigest, approvedBy string, now time.Time) (RedactionApproval, error) {
	if sourceRunDigest == "" || derivedContentDigest == "" || manifestDigest == "" || scanDigest == "" || approvedBy == "" {
		return RedactionApproval{}, errors.New("redaction approval requires exact digests and approver")
	}
	approval := RedactionApproval{SchemaVersion: "1.0", SourceRunDigest: sourceRunDigest, DerivedContentDigest: derivedContentDigest, RedactionManifestDigest: manifestDigest, ScanDigest: scanDigest, ApprovedBy: approvedBy, ApprovedAt: now.UTC()}
	approval.Digest = digestRedaction(approval)
	return approval, nil
}
func (approval RedactionApproval) Authorizes(candidate PublicCandidate) error {
	claimed := approval.Digest
	approval.Digest = ""
	if approval.SchemaVersion != "1.0" || approval.ApprovedBy == "" || approval.ApprovedAt.IsZero() || claimed != digestRedaction(approval) || approval.DerivedContentDigest != candidate.Digest || approval.ScanDigest != candidate.ScanDigest || claimed != candidate.RedactionApprovalDigest {
		return errors.New("redaction approval does not exactly match candidate")
	}
	return nil
}

func digestApproval(approval PublicApproval) string {
	approval.Digest = ""
	approval.Revoked = false
	encoded, _ := json.Marshal(approval)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func digestRedaction(approval RedactionApproval) string {
	approval.Digest = ""
	encoded, _ := json.Marshal(approval)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}
