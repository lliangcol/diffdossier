package policy

import (
	"testing"
	"time"

	"github.com/lliangcol/diffdossier/pkg/schema"
)

func TestMostRestrictive(t *testing.T) {
	got, err := MostRestrictive(schema.PublicSynthetic, schema.PrivateProject, schema.PublicProject)
	if err != nil || got != schema.PrivateProject {
		t.Fatalf("got=%q err=%v", got, err)
	}
	got, err = MostRestrictive()
	if err != nil || got != schema.PrivateProject {
		t.Fatalf("empty classification must fail safe to private_project: got=%q err=%v", got, err)
	}
}

func TestTrustBindingRequiresExactUnexpiredMatch(t *testing.T) {
	now := time.Now().UTC()
	trusted := TrustBinding{
		RepositoryID: "repo", SnapshotID: "snap-a", TaskInputDigest: "task-a",
		ExecutionPlanDigest: "plan-a", ConfigDigest: "config-a", BinaryDigest: "binary-a",
		Capability: "gates.run", ExpiresAt: now.Add(time.Hour),
	}
	if !trusted.Authorizes(trusted, now) {
		t.Fatal("exact binding should authorize")
	}
	changed := trusted
	changed.SnapshotID = "snap-b"
	if trusted.Authorizes(changed, now) {
		t.Fatal("changed snapshot must invalidate trust")
	}
	if trusted.Authorizes(trusted, now.Add(2*time.Hour)) {
		t.Fatal("expired trust must not authorize")
	}
}

func TestEgressFailsClosed(t *testing.T) {
	now := time.Now().UTC()
	grant := EgressGrant{
		Provider: "command", SnapshotID: "snap-a", TaskInputDigest: "task-a",
		DataClass: schema.PrivateProject, MaxBytes: 100, ExpiresAt: now.Add(time.Hour),
	}
	request := EgressRequest{
		Provider: "command", SnapshotID: "snap-a", TaskInputDigest: "task-a",
		DataClass: schema.PrivateProject, Bytes: 100,
	}
	if !grant.Authorizes(request, now) {
		t.Fatal("exact egress grant should authorize")
	}
	request.DataClass = schema.SecretDenied
	if grant.Authorizes(request, now) {
		t.Fatal("secret_denied must never authorize")
	}
}

func TestPublicApprovalIsContentBound(t *testing.T) {
	candidate := PublicCandidate{
		Digest: "sha256:a", ArtifactClass: ArtifactPublicProject, Action: "create",
		PolicyDigest: "sha256:p", ScanDigest: "sha256:s", PublicRevision: "abc123",
	}
	approval, err := NewPublicApproval(candidate, "owner", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := approval.Authorizes(candidate); err != nil {
		t.Fatal(err)
	}
	candidate.Digest = "sha256:b"
	if err := approval.Authorizes(candidate); err == nil {
		t.Fatal("changed candidate must require new approval")
	}
	invalid := candidate
	invalid.ArtifactClass = "private_project"
	if err := approval.Authorizes(invalid); err == nil {
		t.Fatal("private source classification must not be an export artifact class")
	}
}

func TestRedactedSummaryRequiresSeparateApproval(t *testing.T) {
	candidate := PublicCandidate{
		Digest: "sha256:a", ArtifactClass: ArtifactRedactedSummary, Action: "create",
		PolicyDigest: "sha256:p", ScanDigest: "sha256:s",
	}
	// Public approval alone cannot replace the independently bound redaction approval.
	approval, err := NewPublicApproval(candidate, "owner", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := approval.Authorizes(candidate); err == nil {
		t.Fatal("redacted summary without redaction approval must fail")
	}
	redaction, err := NewRedactionApproval("sha256:source", candidate.Digest, "sha256:manifest", candidate.ScanDigest, "security", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	candidate.RedactionApprovalDigest = redaction.Digest
	approval, err = NewPublicApproval(candidate, "owner", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := approval.Authorizes(candidate); err != nil {
		t.Fatal(err)
	}
	if err := redaction.Authorizes(candidate); err != nil {
		t.Fatal(err)
	}
}
