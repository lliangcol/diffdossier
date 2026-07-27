package exporter

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	gatePlan := []byte(`{"gates":[{"requested_executable":"git","executable":"/Users/example/bin/git","argv":["git","/Users/example/repo/file"],"cwd":"C:\\Users\\example\\repo","cwd_class":"repository","path_bytes_base64":"/w=="}]}`)
	if err := os.WriteFile(filepath.Join(dir, "gate-plan.json"), gatePlan, 0o600); err != nil {
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
		if file.Name == "gate-plan.json" {
			reader, openErr := file.Open()
			if openErr != nil {
				t.Fatal(openErr)
			}
			content, readErr := io.ReadAll(reader)
			_ = reader.Close()
			if readErr != nil {
				t.Fatal(readErr)
			}
			text := string(content)
			if strings.Contains(text, "/Users/example") || strings.Contains(text, `C:\\Users`) {
				t.Fatalf("absolute path leaked: %s", text)
			}
			if !strings.Contains(text, `"executable": "git"`) || !strings.Contains(text, `"cwd": "."`) {
				t.Fatalf("portable path projection is incomplete: %s", text)
			}
			if !strings.Contains(text, `"path_bytes_base64": "/w=="`) {
				t.Fatalf("non-path evidence was altered: %s", text)
			}
		}
	}
	if manifest.RunDigest == "" {
		t.Fatal("missing manifest digest")
	}
	if len(manifest.PathSanitizedFiles) != 1 || manifest.PathSanitizedFiles[0] != "gate-plan.json" {
		t.Fatalf("path sanitization was not declared: %+v", manifest.PathSanitizedFiles)
	}
}

func TestPublicLifecycleExactApprovalAndTombstone(t *testing.T) {
	preparation, err := PreparePublic([]byte("synthetic summary\n"), publicschema.PublicSynthetic, "create", "policy", "", "")
	if err != nil {
		t.Fatal(err)
	}
	approval, err := policy.NewPublicApproval(preparation.Candidate, "owner", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := CreatePublic(preparation, approval, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.BundleDigest == "" || bundle.Action != "create" {
		t.Fatalf("bundle binding is incomplete: %+v", bundle)
	}
	changed := preparation
	changed.Candidate.Digest = "changed"
	if _, err := CreatePublic(changed, approval, nil); err == nil {
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
	if _, err := CreatePublic(preparation, policy.PublicApproval{}, nil); err == nil {
		t.Fatal("scan finding did not block")
	}
}

func TestRedactedSummaryRequiresMatchingPrivateApproval(t *testing.T) {
	content := []byte("derived summary")
	provisional, err := PreparePublic(content, publicschema.PrivateProject, "create", "policy", "", "placeholder")
	if err != nil {
		t.Fatal(err)
	}
	redaction, err := policy.NewRedactionApproval("sha256:source", provisional.Candidate.Digest, "sha256:manifest", provisional.Candidate.ScanDigest, "security", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	preparation, err := PreparePublic(content, publicschema.PrivateProject, "create", "policy", "", redaction.Digest)
	if err != nil {
		t.Fatal(err)
	}
	approval, err := policy.NewPublicApproval(preparation.Candidate, "owner", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreatePublic(preparation, approval, nil); err == nil {
		t.Fatal("missing redaction approval accepted")
	}
	if _, err := CreatePublic(preparation, approval, &redaction); err != nil {
		t.Fatal(err)
	}
	changed := redaction
	changed.DerivedContentDigest = "sha256:changed"
	if _, err := CreatePublic(preparation, approval, &changed); err == nil {
		t.Fatal("tampered redaction approval accepted")
	}
}

func TestPublicProjectPrepareRequiresRevision(t *testing.T) {
	if _, err := PreparePublic([]byte("public"), publicschema.PublicProject, "create", "policy", "", ""); err == nil {
		t.Fatal("public project without revision prepared")
	}
}

func TestPublicBundleContainsNoPrivateApprovalOrSourceIdentity(t *testing.T) {
	preparation, err := PreparePublic([]byte("synthetic"), publicschema.PublicSynthetic, "create", "policy", "", "")
	if err != nil {
		t.Fatal(err)
	}
	approval, err := policy.NewPublicApproval(preparation.Candidate, "private-approver", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := CreatePublic(preparation, approval, nil)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(bundle)
	text := string(encoded)
	for _, denied := range []string{"private-approver", "source_run_digest", "approved_by", "repository_id"} {
		if strings.Contains(text, denied) {
			t.Fatalf("public bundle leaked %q: %s", denied, text)
		}
	}
}
