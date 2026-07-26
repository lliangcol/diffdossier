package buildinfo

import (
	"runtime"
	"runtime/debug"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// Info is the stable machine-readable build identity exposed by the version command.
type Info struct {
	SchemaVersion string `json:"schema_version"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	Commit        string `json:"commit"`
	BuildDate     string `json:"build_date"`
	GoVersion     string `json:"go_version"`
	OS            string `json:"os"`
	Architecture  string `json:"architecture"`
	CGOEnabled    string `json:"cgo_enabled"`
}

// Current returns build identity without consulting the network or target repository.
func Current() Info {
	return Info{
		SchemaVersion: "1.0",
		Name:          "DiffDossier",
		Version:       Version,
		Commit:        Commit,
		BuildDate:     BuildDate,
		GoVersion:     runtime.Version(),
		OS:            runtime.GOOS,
		Architecture:  runtime.GOARCH,
		CGOEnabled:    buildSetting("CGO_ENABLED", "unknown"),
	}
}

func buildSetting(key, fallback string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fallback
	}
	for _, setting := range info.Settings {
		if setting.Key == key {
			return setting.Value
		}
	}
	return fallback
}
