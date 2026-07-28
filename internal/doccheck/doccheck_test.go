package doccheck

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCheckRepositoryDocumentation(t *testing.T) {
	if err := Check(repositoryRoot(t)); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRejectsInjectedReleaseDrift(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, repositoryRoot(t), root)

	path := filepath.Join(root, "README.md")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(contents), "Public beta Pre-releases exist;", "No public beta releases exist;", 1)), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Check(root); err == nil {
		t.Fatal("Check accepted deliberately injected release-state drift")
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("determine test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func copyFixture(t *testing.T, source, destination string) {
	t.Helper()
	for path := range releaseDocuments {
		copyFile(t, filepath.Join(source, filepath.FromSlash(path)), filepath.Join(destination, filepath.FromSlash(path)))
	}
	copyFile(t, filepath.Join(source, "docs", "release-evidence-inventory.md"), filepath.Join(destination, "docs", "release-evidence-inventory.md"))
	checkpoints, err := filepath.Glob(filepath.Join(source, "docs", "checkpoints", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range checkpoints {
		copyFile(t, path, filepath.Join(destination, "docs", "checkpoints", filepath.Base(path)))
	}
}

func copyFile(t *testing.T, source, destination string) {
	t.Helper()
	contents, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, contents, 0o600); err != nil {
		t.Fatal(err)
	}
}
