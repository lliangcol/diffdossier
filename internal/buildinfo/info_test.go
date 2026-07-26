package buildinfo

import "testing"

func TestCurrentHasStableIdentityFields(t *testing.T) {
	info := Current()
	if info.SchemaVersion != "1.0" {
		t.Fatalf("SchemaVersion = %q, want 1.0", info.SchemaVersion)
	}
	if info.Name != "DiffDossier" {
		t.Fatalf("Name = %q, want DiffDossier", info.Name)
	}
	if info.Version == "" || info.Commit == "" || info.BuildDate == "" {
		t.Fatalf("build identity contains empty fields: %+v", info)
	}
	if info.GoVersion == "" || info.OS == "" || info.Architecture == "" {
		t.Fatalf("runtime identity contains empty fields: %+v", info)
	}
}

func TestBuildSettingFallback(t *testing.T) {
	const fallback = "fallback"
	if got := buildSetting("DIFFDOSSIER_SETTING_THAT_DOES_NOT_EXIST", fallback); got != fallback {
		t.Fatalf("buildSetting fallback = %q, want %q", got, fallback)
	}
}
