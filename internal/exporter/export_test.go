package exporter

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lliangcol/diffdossier/internal/policy"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

func TestPortableExcludesAuthorityAndIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "run.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "approvals"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "approvals", "trust.json"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	one, manifest, err := Portable(dir)
	if err != nil {
		t.Fatal(err)
	}
	two, _, _ := Portable(dir)
	if !bytes.Equal(one, two) {
		t.Fatal("portable archive is not deterministic")
	}
	reader, err := zip.NewReader(bytes.NewReader(one), int64(len(one)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range reader.File {
		if file.Name == "approvals/trust.json" {
			t.Fatal("authority leaked")
		}
	}
	if manifest.RunDigest == "" {
		t.Fatal("missing manifest digest")
	}
}

func TestPublicLifecycleExactApprovalAndTombstone(t *testing.T) {
	preparation, err := PreparePublic([]byte("synthetic summary\n"), publicschema.PublicSynthetic, "create", "policy", "", "")
	if err != nil {
		t.Fatal(err)
	}
	approval := policy.PublicApproval{Binding: publicschema.ApprovalBinding{CandidateDigest: preparation.Candidate.Digest, DataClass: publicschema.PublicSynthetic, Action: "create", PolicyDigest: "policy", ScanDigest: preparation.Candidate.ScanDigest}}
	bundle, err := CreatePublic(preparation, approval)
	if err != nil {
		t.Fatal(err)
	}
	changed := preparation
	changed.Candidate.Digest = "changed"
	if _, err := CreatePublic(changed, approval); err == nil {
		t.Fatal("changed candidate used old approval")
	}
	tombstone, err := Revoke(bundle.ApprovalRecordDigest, bundle.ContentDigest, "withdrawn", time.Unix(0, 0))
	if err != nil || tombstone.TombstoneDigest == "" {
		t.Fatalf("revocation=%+v err=%v", tombstone, err)
	}
}

func TestPublicSecretScanBlocksCreate(t *testing.T) {
	preparation, err := PreparePublic([]byte("api_key=abcdefghijk"), publicschema.PublicSynthetic, "create", "policy", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(preparation.ScanFindings) == 0 {
		t.Fatal("secret not detected")
	}
	if _, err := CreatePublic(preparation, policy.PublicApproval{}); err == nil {
		t.Fatal("scan finding did not block")
	}
}
