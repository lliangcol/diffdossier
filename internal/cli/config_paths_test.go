package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEffectiveConfigPrecedence(t *testing.T) {
	repo := t.TempDir()
	project := filepath.Join(repo, "diffdossier.toml")
	environment := filepath.Join(repo, "environment.toml")
	explicit := filepath.Join(repo, "explicit.toml")
	for path, baseline := range map[string]string{project: "project", environment: "environment", explicit: "explicit"} {
		if err := os.WriteFile(path, []byte("baseline = \""+baseline+"\"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("DIFFDOSSIER_CONFIG", environment)
	effective, err := loadEffectiveConfig(repo, "")
	if err != nil || effective.Config.Baseline != "environment" || effective.Sources[0].Kind != "explicit" {
		t.Fatalf("environment selection: effective=%+v err=%v", effective, err)
	}
	effective, err = loadEffectiveConfig(repo, "explicit.toml")
	if err != nil || effective.Config.Baseline != "explicit" {
		t.Fatalf("CLI selection: effective=%+v err=%v", effective, err)
	}
}

func TestResolveStateAndCachePrecedenceAndAbsoluteRequirement(t *testing.T) {
	environmentState := filepath.Join(t.TempDir(), "environment-state")
	environmentCache := filepath.Join(t.TempDir(), "environment-cache")
	flagState := filepath.Join(t.TempDir(), "flag-state")
	t.Setenv("DIFFDOSSIER_STATE_DIR", environmentState)
	t.Setenv("DIFFDOSSIER_CACHE_DIR", environmentCache)
	state, err := resolveStateRoot("")
	if err != nil || state != filepath.Clean(environmentState) {
		t.Fatalf("state=%q err=%v", state, err)
	}
	cache, err := resolveCacheRoot("")
	if err != nil || cache != filepath.Clean(environmentCache) {
		t.Fatalf("cache=%q err=%v", cache, err)
	}
	state, err = resolveStateRoot(flagState)
	if err != nil || state != filepath.Clean(flagState) {
		t.Fatalf("flag state=%q err=%v", state, err)
	}
	if _, err := resolveStateRoot("relative"); err == nil {
		t.Fatal("relative state path must be rejected")
	}
	if _, err := resolveCacheRoot("relative"); err == nil {
		t.Fatal("relative cache path must be rejected")
	}
}
