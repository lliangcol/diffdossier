// Package platform resolves operating-system-specific configuration and state paths.
package platform

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

type Paths struct {
	ConfigFile string `json:"config_path"`
	StateDir   string `json:"state_dir"`
	CacheDir   string `json:"cache_dir"`
}

func DefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	return ResolvePaths(runtime.GOOS, home, os.Getenv), nil
}

func ResolvePaths(goos, home string, getenv func(string) string) Paths {
	switch goos {
	case "darwin":
		base := filepath.Join(home, "Library", "Application Support", "DiffDossier")
		return Paths{
			ConfigFile: filepath.Join(base, "config.toml"),
			StateDir:   filepath.Join(base, "State"),
			CacheDir:   filepath.Join(home, "Library", "Caches", "DiffDossier"),
		}
	case "windows":
		roaming := getenv("APPDATA")
		local := getenv("LOCALAPPDATA")
		if roaming == "" {
			roaming = filepath.Join(home, "AppData", "Roaming")
		}
		if local == "" {
			local = filepath.Join(home, "AppData", "Local")
		}
		return Paths{
			ConfigFile: filepath.Join(roaming, "DiffDossier", "config.toml"),
			StateDir:   filepath.Join(local, "DiffDossier", "State"),
			CacheDir:   filepath.Join(local, "DiffDossier", "Cache"),
		}
	default:
		configHome := getenv("XDG_CONFIG_HOME")
		stateHome := getenv("XDG_STATE_HOME")
		cacheHome := getenv("XDG_CACHE_HOME")
		if configHome == "" {
			configHome = filepath.Join(home, ".config")
		}
		if stateHome == "" {
			stateHome = filepath.Join(home, ".local", "state")
		}
		if cacheHome == "" {
			cacheHome = filepath.Join(home, ".cache")
		}
		return Paths{
			ConfigFile: filepath.Join(configHome, "diffdossier", "config.toml"),
			StateDir:   filepath.Join(stateHome, "diffdossier"),
			CacheDir:   filepath.Join(cacheHome, "diffdossier"),
		}
	}
}

// EnsurePrivateDir creates a user-private directory and rejects insecure POSIX modes.
func EnsurePrivateDir(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("state and cache paths must be absolute")
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0o700); err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Mode().Perm()&0o077 != 0 {
			return errors.New("directory permissions are not private")
		}
	}
	return nil
}

// PrivateTempDir creates a random, private temporary workspace outside the target repository.
func PrivateTempDir(base string) (string, error) {
	if base != "" && !filepath.IsAbs(base) {
		return "", errors.New("temporary base path must be absolute")
	}
	path, err := os.MkdirTemp(base, "diffdossier-")
	if err != nil {
		return "", err
	}
	if err := EnsurePrivateDir(path); err != nil {
		_ = os.RemoveAll(path)
		return "", err
	}
	return path, nil
}
