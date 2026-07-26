package platform

import (
	"path/filepath"
	"testing"
)

func TestResolvePaths(t *testing.T) {
	env := map[string]string{
		"XDG_CONFIG_HOME": "/xdg/config",
		"XDG_STATE_HOME":  "/xdg/state",
		"XDG_CACHE_HOME":  "/xdg/cache",
		"APPDATA":         "C:/Users/test/AppData/Roaming",
		"LOCALAPPDATA":    "C:/Users/test/AppData/Local",
	}
	getenv := func(key string) string { return env[key] }
	tests := []struct{ goos, wantState string }{
		{"linux", filepath.Join("/xdg/state", "diffdossier")},
		{"darwin", filepath.Join("/home/test", "Library", "Application Support", "DiffDossier", "State")},
		{"windows", filepath.Join("C:/Users/test/AppData/Local", "DiffDossier", "State")},
	}
	for _, test := range tests {
		t.Run(test.goos, func(t *testing.T) {
			got := ResolvePaths(test.goos, "/home/test", getenv)
			if got.StateDir != test.wantState {
				t.Fatalf("state = %q, want %q", got.StateDir, test.wantState)
			}
		})
	}
}

func TestEnsurePrivateDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state")
	if err := EnsurePrivateDir(path); err != nil {
		t.Fatal(err)
	}
}

func TestPrivateTempDir(t *testing.T) {
	path, err := PrivateTempDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) == "diffdossier-" {
		t.Fatalf("temporary path is not randomized: %s", path)
	}
}

func TestCapabilitiesExposeUnverifiedWindowsBoundaries(t *testing.T) {
	capabilities := ResolveCapabilities("windows", "arm64")
	if capabilities.ProcessTree != "job_object_implemented_native_test_pending" || capabilities.PrivateStatePermissions != "windows_acl_native_test_pending" {
		t.Fatalf("capabilities=%+v", capabilities)
	}
}
