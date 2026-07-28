// Package doccheck validates repository-local documentation facts that must
// stay aligned without depending on a live GitHub query.
package doccheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var releaseDocuments = map[string][]string{
	"README.md":               {"Public beta Pre-releases exist;", "no stable release"},
	"SUPPORT.md":              {"public beta Pre-releases", "no stable release"},
	"docs/install.md":         {"Public beta Pre-releases exist.", "supported stable releases"},
	"docs/release-process.md": {"public beta Pre-releases", "no stable release"},
}

// Check validates the deliberately small set of release and checkpoint facts
// that this repository can establish from versioned files alone. It does not
// assert that a Release is still visible on GitHub.
func Check(root string) error {
	for path, required := range releaseDocuments {
		contents, err := read(root, path)
		if err != nil {
			return err
		}
		for _, fragment := range required {
			if !strings.Contains(contents, fragment) {
				return fmt.Errorf("%s is missing required release-state text %q", path, fragment)
			}
		}
	}

	inventory, err := read(root, "docs/release-evidence-inventory.md")
	if err != nil {
		return err
	}
	for _, version := range []string{"v0.1.0-beta.1", "v0.1.0-beta.2", "v0.1.0-beta.3"} {
		if !strings.Contains(inventory, version) {
			return fmt.Errorf("docs/release-evidence-inventory.md is missing %s", version)
		}
	}

	return checkCheckpoints(root)
}

func checkCheckpoints(root string) error {
	paths, err := filepath.Glob(filepath.Join(root, "docs", "checkpoints", "*.md"))
	if err != nil {
		return fmt.Errorf("glob checkpoint files: %w", err)
	}
	for _, path := range paths {
		if filepath.Base(path) == "README.md" {
			continue
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", filepath.ToSlash(path), err)
		}
		text := string(contents)
		for _, field := range []string{
			"- Status: historical\n",
			"- Captured-at: ",
			"- Source-commit: ",
			"- Superseded-by: ",
			"- Current-state notice: Historical checkpoint — do not treat this document as current project status.",
		} {
			if !strings.Contains(text, field) {
				return fmt.Errorf("%s is missing required checkpoint metadata %q", filepath.ToSlash(path), strings.TrimSpace(field))
			}
		}
	}
	return nil
}

func read(root, path string) (string, error) {
	contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(contents), nil
}
