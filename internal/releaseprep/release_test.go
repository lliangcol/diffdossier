package releaseprep

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestChecksumsRejectTamperingAndExtraFiles(t *testing.T) {
	root := t.TempDir()
	manifest := writeValidReleaseFixture(t, root)
	if err := os.WriteFile(filepath.Join(root, "extra"), []byte("unexpected"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(root, false); err == nil || !strings.Contains(err.Error(), "coverage mismatch") {
		t.Fatalf("Verify() error = %v, want coverage mismatch", err)
	}
	if err := os.Remove(filepath.Join(root, "extra")); err != nil {
		t.Fatal(err)
	}
	if err := writeChecksums(root); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(root, false); err != nil {
		t.Fatalf("Verify() = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, manifest.Artifacts[0].Name), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(root, false); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Verify() error = %v, want checksum mismatch", err)
	}
}

func TestVerifyRejectsNonRegularEntry(t *testing.T) {
	root := t.TempDir()
	writeValidReleaseFixture(t, root)
	if err := os.Mkdir(filepath.Join(root, "unexpected-directory"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(root, false); err == nil || !strings.Contains(err.Error(), "non-regular") {
		t.Fatalf("Verify() error = %v, want non-regular entry", err)
	}
}

func TestPrepareStagingRequiresAbsentOutput(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "dist")
	staging, err := prepareStaging(output)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(staging) != root {
		t.Fatalf("staging path %q is not beside output %q", staging, output)
	}
	if err := os.RemoveAll(staging); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(output, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareStaging(output); err == nil || !strings.Contains(err.Error(), "must not already exist") {
		t.Fatalf("prepareStaging() error = %v, want existing-output rejection", err)
	}
}

func writeValidReleaseFixture(t *testing.T, root string) Manifest {
	t.Helper()
	manifest := Manifest{SchemaVersion: schemaVersion, Project: "DiffDossier", Version: "phase7-test", Commit: strings.Repeat("a", 40), SourceDate: "2026-07-26T00:00:00Z", GoVersion: "go-test", Candidate: true}
	for _, target := range targets {
		name := "diffdossier_" + manifest.Version + "_" + target.OS + "_" + target.Arch + "." + target.Format
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
		digest, size, err := fileDigest(path)
		if err != nil {
			t.Fatal(err)
		}
		manifest.Artifacts = append(manifest.Artifacts, Artifact{Name: name, OS: target.OS, Arch: target.Arch, Format: target.Format, SHA256: digest, Size: size})
	}
	if err := writeJSON(filepath.Join(root, "release-manifest.json"), manifest); err != nil {
		t.Fatal(err)
	}
	provenance := Provenance{SchemaVersion: schemaVersion, Kind: "local_unsigned_provenance", Source: "https://github.com/lliangcol/diffdossier", Version: manifest.Version, Commit: manifest.Commit, SourceDate: manifest.SourceDate, GoVersion: manifest.GoVersion, Artifacts: manifest.Artifacts}
	if err := writeJSON(filepath.Join(root, "provenance.json"), provenance); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(root, "diffdossier.spdx.json"), newSBOM(manifest)); err != nil {
		t.Fatal(err)
	}
	if err := writeChecksums(root); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func TestDeterministicArchives(t *testing.T) {
	stage := t.TempDir()
	for _, name := range []string{"LICENSE", "NOTICE", "README.md", "diffdossier", "diffdossier.exe"} {
		if err := os.WriteFile(filepath.Join(stage, name), []byte(name+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	date := time.Date(2026, 7, 26, 1, 2, 3, 0, time.UTC)
	for _, test := range []struct {
		name  string
		write func(string) error
	}{
		{name: "tar.gz", write: func(path string) error { return writeTarGz(path, stage, "bundle", date) }},
		{name: "zip", write: func(path string) error { return writeZip(path, stage, "bundle", date) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			first := filepath.Join(t.TempDir(), "first."+test.name)
			second := filepath.Join(t.TempDir(), "second."+test.name)
			if err := test.write(first); err != nil {
				t.Fatal(err)
			}
			if err := test.write(second); err != nil {
				t.Fatal(err)
			}
			firstData, _ := os.ReadFile(first)
			secondData, _ := os.ReadFile(second)
			if !bytes.Equal(firstData, secondData) {
				t.Fatal("archive bytes differ for identical inputs")
			}
		})
	}
}

func TestReplaceEnvIsDeterministic(t *testing.T) {
	got := replaceEnv([]string{"PATH=/bin", "GOOS=old", "GOARCH=old"}, map[string]string{"GOOS": "linux", "GOARCH": "arm64"})
	want := []string{"PATH=/bin", "GOARCH=arm64", "GOOS=linux"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("replaceEnv() = %#v, want %#v", got, want)
	}
}

func TestControlledGoEnvOverridesAmbientBuildInputs(t *testing.T) {
	got := controlledGoEnv([]string{"PATH=/bin", "GOFLAGS=-tags=ambient", "GOWORK=/tmp/ambient.work", "GOTOOLCHAIN=auto"}, Target{OS: "linux", Arch: "arm64"})
	joined := "|" + strings.Join(got, "|") + "|"
	for _, want := range []string{"|GOFLAGS=|", "|GOWORK=off|", "|GOTOOLCHAIN=local|", "|GOPROXY=off|", "|GOOS=linux|", "|GOARCH=arm64|"} {
		if !strings.Contains(joined, want) {
			t.Errorf("controlledGoEnv() = %#v, missing %s", got, want)
		}
	}
	for _, forbidden := range []string{"GOFLAGS=-tags=ambient", "GOWORK=/tmp/ambient.work", "GOTOOLCHAIN=auto"} {
		if strings.Contains(joined, forbidden) {
			t.Errorf("controlledGoEnv() retained %s", forbidden)
		}
	}
}

func TestValidSemVer(t *testing.T) {
	for _, value := range []string{"v0.1.0", "v1.2.3-beta.1", "v1.2.3+build.7", "v1.2.3-rc.1+build"} {
		if !validSemVer(value) {
			t.Errorf("validSemVer(%q) = false", value)
		}
	}
	for _, value := range []string{"1.2.3", "v01.2.3", "v1.02.3", "v1.2", "v1.2.3-", "v1.2.3-01", "v1.2.3+", "v1.2.3+x..y", "v1.2.3_/x"} {
		if validSemVer(value) {
			t.Errorf("validSemVer(%q) = true", value)
		}
	}
}

func TestReadChecksumsRejectsTraversalAndDuplicates(t *testing.T) {
	root := t.TempDir()
	digest := strings.Repeat("0", 64)
	path := filepath.Join(root, "SHA256SUMS")
	if err := os.WriteFile(path, []byte(digest+"  ../escape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	checksums, err := readChecksums(path)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base("../escape") == "../escape" || checksums["../escape"] == "" {
		t.Fatal("test precondition failed")
	}
	if err := os.WriteFile(path, []byte(digest+"  same\n"+digest+"  same\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readChecksums(path); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("readChecksums() error = %v, want duplicate", err)
	}
}
