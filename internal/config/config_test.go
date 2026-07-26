package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validConfig = `schema_version = 1
baseline = "refs/remotes/origin/main"
include_untracked = true
include_ignored = [".generated/report.json"]

[review]
max_files_per_task = 8
max_packet_bytes = 250000
default_provider = "manual"

[state]
retention_days = 30

[risk]
policy_files = [".diffdossier/risk.toml"]

[[gates]]
id = "unit-test"
argv = ["go", "test", "./..."]
timeout_seconds = 900
blocking = true
cache_class = "worktree_deterministic"
final_always = true
`

func TestLoadValid(t *testing.T) {
	path := writeConfig(t, validConfig)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Baseline != "refs/remotes/origin/main" || len(cfg.Gates) != 1 || !cfg.Gates[0].FinalAlways {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestStringArrayPreservesCommaInArgument(t *testing.T) {
	content := strings.Replace(validConfig, `argv = ["go", "test", "./..."]`, `argv = ["go", "test", "-run=One,Two"]`, 1)
	cfg, err := Load(writeConfig(t, content))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Gates[0].Argv[2]; got != "-run=One,Two" {
		t.Fatalf("argument = %q", got)
	}
}

func TestLoadRejectsUnknownAndInvalid(t *testing.T) {
	tests := []struct{ name, content, want string }{
		{"unknown", validConfig + "\nunknown = true\n", "unknown field"},
		{"schema", strings.Replace(validConfig, "schema_version = 1", "schema_version = 2", 1), "unsupported schema_version"},
		{"baseline", strings.Replace(validConfig, "baseline = \"refs/remotes/origin/main\"\n", "", 1), "baseline is required"},
		{"cache", strings.Replace(validConfig, "worktree_deterministic", "forever", 1), "invalid cache_class"},
		{"duplicate", strings.Replace(validConfig, "schema_version = 1", "schema_version = 1\nschema_version = 1", 1), "duplicate field"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load(writeConfig(t, test.content))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "diffdossier.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
