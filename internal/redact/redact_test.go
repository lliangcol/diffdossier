package redact

import (
	"bytes"
	"testing"
)

func TestRedactCanariesWithoutPersistingMatch(t *testing.T) {
	secret := []byte("access_token=abcdefghijklmnop")
	output, manifest, err := Redact(append([]byte("before "), secret...))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(output, secret) || !bytes.Contains(output, []byte("[REDACTED:assigned-secret]")) {
		t.Fatalf("redaction failed: %s", output)
	}
	encoded := []byte(manifest.Findings[0].MatchDigest)
	if bytes.Contains(encoded, secret) {
		t.Fatal("manifest persisted secret")
	}
}
func TestScanBudgetFailsClosed(t *testing.T) {
	if _, err := Scan(make([]byte, MaxScanBytes+1)); err == nil {
		t.Fatal("oversize scan accepted")
	}
}

func TestKnownOpaqueValueIsRedacted(t *testing.T) {
	secret := "opaque-value-with-no-keyword"
	output, manifest, err := RedactKnown([]byte("echo "+secret), []string{secret})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(output, []byte(secret)) || len(manifest.Findings) != 1 || manifest.Findings[0].Rule != "known-value" {
		t.Fatalf("output=%s manifest=%+v", output, manifest)
	}
}

func FuzzRedactionIsDeterministicAndIdempotent(f *testing.F) {
	f.Add([]byte("access_token=abcdefghijklmnop"))
	f.Add([]byte("ordinary log"))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > MaxScanBytes {
			return
		}
		one, _, err := Redact(input)
		if err != nil {
			t.Fatal(err)
		}
		two, _, err := Redact(input)
		if err != nil || !bytes.Equal(one, two) {
			t.Fatal("redaction is not deterministic")
		}
		again, _, err := Redact(one)
		if err != nil || !bytes.Equal(one, again) {
			t.Fatal("redaction is not idempotent")
		}
	})
}
