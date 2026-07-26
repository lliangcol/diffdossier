package cli

import (
	"bytes"
	"encoding/json"
	"errors"
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

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
