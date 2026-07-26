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
cwd = "."
env_allowlist = ["PATH", "GOCACHE"]
when_paths = ["**/*.go", "go.mod"]
depends_on = []
timeout_seconds = 900
resource_class = "cpu"
blocking = true
cache_class = "worktree_deterministic"
final_always = true
network_class = "none"
expected_writes = [".diffdossier-tmp"]
redaction_policy = "test-output"
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
		{"dependency", strings.Replace(validConfig, "depends_on = []", `depends_on = ["missing"]`, 1), "invalid dependency"},
		{"cycle", strings.Replace(validConfig, "depends_on = []", `depends_on = ["unit-test"]`, 1), "invalid dependency"},
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

func TestLoadEffectiveAppliesUserThenRepositoryPrecedence(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user.toml")
	repository := filepath.Join(root, "repository.toml")
	if err := os.WriteFile(user, []byte(`schema_version = 1
baseline = "user-main"
include_untracked = false
[review]
max_files_per_task = 3
[[gates]]
id = "user-gate"
argv = ["false"]
cwd = "."
timeout_seconds = 1
resource_class = "cpu"
cache_class = "host_volatile"
network_class = "none"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repository, []byte(`baseline = "repo-main"
[review]
max_packet_bytes = 1234
[[gates]]
id = "repo-gate"
argv = ["go", "test", "./..."]
cwd = "."
timeout_seconds = 60
resource_class = "cpu"
cache_class = "worktree_deterministic"
network_class = "none"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	effective, err := LoadEffective(user, repository)
	if err != nil {
		t.Fatal(err)
	}
	if effective.Config.Baseline != "repo-main" || effective.Config.IncludeUntracked || effective.Config.Review.MaxFilesPerTask != 3 || effective.Config.Review.MaxPacketBytes != 1234 {
		t.Fatalf("unexpected layered config: %+v", effective.Config)
	}
	if len(effective.Config.Gates) != 1 || effective.Config.Gates[0].ID != "repo-gate" {
		t.Fatalf("higher-layer gates must replace lower gates: %+v", effective.Config.Gates)
	}
	if len(effective.Sources) != 2 || effective.Sources[0].Kind != "user" || effective.Sources[1].Kind != "repository" || effective.Digest == "" {
		t.Fatalf("unexpected effective metadata: %+v", effective)
	}
}

func TestLoadEffectiveAllowsMissingUserButRequiresRepository(t *testing.T) {
	root := t.TempDir()
	repository := filepath.Join(root, "repository.toml")
	if err := os.WriteFile(repository, []byte("baseline = \"HEAD\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	effective, err := LoadEffective(filepath.Join(root, "missing-user.toml"), repository)
	if err != nil {
		t.Fatal(err)
	}
	if len(effective.Sources) != 1 || effective.Sources[0].Kind != "repository" {
		t.Fatalf("sources=%+v", effective.Sources)
	}
	if _, err := LoadEffective("", filepath.Join(root, "missing-project.toml")); err == nil {
		t.Fatal("missing repository config must fail closed")
	}
}

func TestLoadEffectiveRejectsInvalidOptionalUserLayer(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user.toml")
	repository := filepath.Join(root, "repository.toml")
	if err := os.WriteFile(user, []byte("unknown = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repository, []byte("baseline = \"HEAD\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadEffective(user, repository); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error=%v", err)
	}
}

func TestLoadEffectiveRejectsSemanticallyInvalidOverriddenUserLayer(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user.toml")
	repository := filepath.Join(root, "repository.toml")
	if err := os.WriteFile(user, []byte("[review]\nmax_files_per_task = 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repository, []byte("baseline = \"HEAD\"\n[review]\nmax_files_per_task = 8\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadEffective(user, repository); err == nil || !strings.Contains(err.Error(), "review limits must be positive") {
		t.Fatalf("error=%v", err)
	}
}

func TestLoadExactDoesNotMergeUserConfiguration(t *testing.T) {
	path := writeConfig(t, "baseline = \"exact\"\n")
	effective, err := LoadExact(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(effective.Sources) != 1 || effective.Sources[0].Kind != "explicit" || effective.Config.Baseline != "exact" {
		t.Fatalf("effective=%+v", effective)
	}
}

func TestBaselineOverrideCompletesAndBindsEffectiveConfig(t *testing.T) {
	path := writeConfig(t, "schema_version = 1\n")
	effective, err := LoadExactWithBaseline(path, "refs/heads/review-base")
	if err != nil {
		t.Fatal(err)
	}
	if effective.Config.Baseline != "refs/heads/review-base" || len(effective.Sources) != 2 || effective.Sources[1].Kind != "cli" {
		t.Fatalf("effective=%+v", effective)
	}
	withoutOverride, err := LoadExactWithBaseline(path, "other")
	if err != nil {
		t.Fatal(err)
	}
	if effective.Digest == withoutOverride.Digest {
		t.Fatal("baseline override must change the effective configuration digest")
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
