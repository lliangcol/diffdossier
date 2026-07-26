package risk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOverrides(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "risk.toml")
	content := "schema_version = 1\n[[rules]]\nglob = \"db/*\"\nlevel = \"L4\"\nreason = \"migration\"\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	rules, err := LoadOverrides(root, []string{"risk.toml"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].Level != L4 {
		t.Fatalf("rules=%+v", rules)
	}
}

func TestLoadOverridesRejectsEscape(t *testing.T) {
	if _, err := LoadOverrides(t.TempDir(), []string{"../risk.toml"}); err == nil {
		t.Fatal("path escape must fail")
	}
}

func TestLoadOverridesRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	external := filepath.Join(t.TempDir(), "risk.toml")
	if err := os.WriteFile(external, []byte("schema_version = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, "risk.toml")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := LoadOverrides(root, []string{"risk.toml"}); err == nil {
		t.Fatal("policy symlink escape must fail")
	}
}
