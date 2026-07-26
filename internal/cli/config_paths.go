package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lliangcol/diffdossier/internal/config"
	"github.com/lliangcol/diffdossier/internal/platform"
)

func loadEffectiveConfig(repoRoot, flagValue string, baselineOverride ...string) (config.Effective, error) {
	baseline := ""
	if len(baselineOverride) > 0 {
		baseline = baselineOverride[0]
	}
	selected := flagValue
	if selected == "" {
		selected = os.Getenv("DIFFDOSSIER_CONFIG")
	}
	if selected != "" {
		path, err := resolveRepositoryRelative(repoRoot, selected, "config")
		if err != nil {
			return config.Effective{}, err
		}
		return config.LoadExactWithBaseline(path, baseline)
	}
	paths, err := platform.DefaultPaths()
	if err != nil {
		return config.Effective{}, err
	}
	return config.LoadEffectiveWithBaseline(paths.ConfigFile, filepath.Join(repoRoot, "diffdossier.toml"), baseline)
}

func resolveRepositoryRelative(repoRoot, value, label string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s path is empty", label)
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value), nil
	}
	if repoRoot == "" || !filepath.IsAbs(repoRoot) {
		return "", fmt.Errorf("relative %s path requires an absolute repository root", label)
	}
	return filepath.Clean(filepath.Join(repoRoot, value)), nil
}

func resolveStateRoot(stateRoot string) (string, error) {
	if stateRoot == "" {
		stateRoot = os.Getenv("DIFFDOSSIER_STATE_DIR")
	}
	if stateRoot == "" {
		paths, err := platform.DefaultPaths()
		if err != nil {
			return "", err
		}
		stateRoot = paths.StateDir
	}
	if !filepath.IsAbs(stateRoot) {
		return "", errors.New("state-dir must be absolute")
	}
	return filepath.Clean(stateRoot), nil
}

func resolveCacheRoot(cacheRoot string) (string, error) {
	if cacheRoot == "" {
		cacheRoot = os.Getenv("DIFFDOSSIER_CACHE_DIR")
	}
	if cacheRoot == "" {
		paths, err := platform.DefaultPaths()
		if err != nil {
			return "", err
		}
		cacheRoot = paths.CacheDir
	}
	if !filepath.IsAbs(cacheRoot) {
		return "", errors.New("cache-dir must be absolute")
	}
	return filepath.Clean(cacheRoot), nil
}
