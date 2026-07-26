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

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
