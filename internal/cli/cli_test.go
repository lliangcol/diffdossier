package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lliangcol/diffdossier/internal/buildinfo"
)

func TestVersionJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"version", "--json"}, &stdout, &stderr)
	if exitCode != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr=%q", exitCode, ExitOK, stderr.String())
	}

	var info buildinfo.Info
	if err := json.Unmarshal(stdout.Bytes(), &info); err != nil {
		t.Fatalf("decode JSON: %v; output=%q", err, stdout.String())
	}
	if info.SchemaVersion != "1.0" || info.Name != "DiffDossier" {
		t.Fatalf("unexpected version identity: %+v", info)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestVersionText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"version"}, &stdout, &stderr)
	if exitCode != ExitOK {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitOK)
	}
	if !strings.HasPrefix(stdout.String(), "diffdossier ") {
		t.Fatalf("stdout = %q, want diffdossier prefix", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--help"}, &stdout, &stderr)
	if exitCode != ExitOK {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitOK)
	}
	if !strings.Contains(stdout.String(), "diffdossier version") {
		t.Fatalf("stdout = %q, want version usage", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestVersionHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"version", "--help"}, &stdout, &stderr)
	if exitCode != ExitOK {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitOK)
	}
	if !strings.Contains(stderr.String(), "emit stable JSON") {
		t.Fatalf("stderr = %q, want version flag help", stderr.String())
	}
}

func TestMissingAndUnknownCommandsAreUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing", args: nil},
		{name: "unknown", args: []string{"unknown"}},
		{name: "extra version argument", args: []string{"version", "extra"}},
		{name: "unknown version flag", args: []string{"version", "--unknown"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if exitCode := Run(test.args, &stdout, &stderr); exitCode != ExitUsage {
				t.Fatalf("exit code = %d, want %d", exitCode, ExitUsage)
			}
			if stderr.Len() == 0 {
				t.Fatal("stderr is empty, want usage diagnostic")
			}
		})
	}
}

func TestVersionOutputFailureIsInternalError(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "text", args: []string{"version"}},
		{name: "json", args: []string{"version", "--json"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer
			exitCode := Run(test.args, failingWriter{}, &stderr)
			if exitCode != ExitInternal {
				t.Fatalf("exit code = %d, want %d", exitCode, ExitInternal)
			}
			if !strings.Contains(stderr.String(), "version output") {
				t.Fatalf("stderr = %q, want output failure diagnostic", stderr.String())
			}
		})
	}
}

func TestConfigValidateJSON(t *testing.T) {
	repo := t.TempDir()
	content := "schema_version = 1\nbaseline = \"HEAD~1\"\n"
	if err := os.WriteFile(filepath.Join(repo, "diffdossier.toml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"config", "validate", "--repo", repo, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope["schema_version"] != "1.0" || envelope["status"] != "ok" {
		t.Fatalf("unexpected envelope: %#v", envelope)
	}
}

func TestConfigValidateFailureJSON(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "diffdossier.toml"), []byte("schema_version = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"config", "validate", "--repo", repo, "--json"}, &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("code=%d want=%d", code, ExitUsage)
	}
	if !strings.Contains(stdout.String(), "DD_CONFIG_INVALID") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestDoctorJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"doctor", "--json"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "\"network_default\":\"none\"") {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestDoctorReportsRuleConflictsWithoutExecutingConfiguredGate(t *testing.T) {
	repo := initializedRepo(t)
	marker := filepath.Join(t.TempDir(), "must-not-exist")
	configuration := `baseline = "HEAD"
[[gates]]
id = "malicious-candidate"
argv = ["sh", "-c", "touch ` + marker + `"]
cwd = "."
timeout_seconds = 30
resource_class = "cpu"
cache_class = "host_volatile"
network_class = "none"
`
	if err := os.WriteFile(filepath.Join(repo, "diffdossier.toml"), []byte(configuration), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "CLAUDE.md"), []byte("two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "state")
	cache := filepath.Join(t.TempDir(), "cache")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"doctor", "--repo", repo, "--state-dir", state, "--cache-dir", cache, "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status":"needs_confirmation"`) || !strings.Contains(stdout.String(), `"commands_executed":0`) || !strings.Contains(stdout.String(), `"rule_conflicts"`) {
		t.Fatalf("stdout=%s", stdout.String())
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("doctor executed an untrusted Gate candidate: %v", err)
	}
}

func TestDoctorHonorsStateAndCacheEnvironment(t *testing.T) {
	repo := initializedRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "diffdossier.toml"), []byte("baseline = \"HEAD\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "state")
	cache := filepath.Join(t.TempDir(), "cache")
	t.Setenv("DIFFDOSSIER_STATE_DIR", state)
	t.Setenv("DIFFDOSSIER_CACHE_DIR", cache)
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"doctor", "--repo", repo, "--json"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), state) || !strings.Contains(stdout.String(), cache) {
		t.Fatalf("stdout=%s", stdout.String())
	}
}

func TestDoctorDiagnosesStateAndCacheInsideRepository(t *testing.T) {
	repo := initializedRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "diffdossier.toml"), []byte("baseline = \"HEAD\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"doctor", "--repo", repo, "--state-dir", filepath.Join(repo, "state"), "--cache-dir", filepath.Join(repo, "cache"), "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"state_status":"invalid"`) || !strings.Contains(stdout.String(), `"cache_status":"invalid"`) || !strings.Contains(stdout.String(), `"status":"needs_confirmation"`) {
		t.Fatalf("stdout=%s", stdout.String())
	}
}

func TestDoctorReportsUnresolvedBaselineAsNeedsConfirmation(t *testing.T) {
	repo := initializedRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "diffdossier.toml"), []byte("baseline = \"refs/heads/missing\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"doctor", "--repo", repo, "--json"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"baseline_status":"unresolved"`) || !strings.Contains(stdout.String(), `"status":"needs_confirmation"`) {
		t.Fatalf("stdout=%s", stdout.String())
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
