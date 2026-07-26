package releaseprep

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var pinnedActionPattern = regexp.MustCompile(`^uses: actions/[a-z0-9-]+@[0-9a-f]{40}(?: # v[0-9]+(?:\.[0-9]+){0,2})?$`)

func TestGitHubWorkflowsPinActionsAndKeepTrustBoundaries(t *testing.T) {
	root := filepath.Join("..", "..", ".github", "workflows")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("found %d workflow entries, want 2", len(entries))
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yml" {
			t.Fatalf("unexpected workflow entry %q", entry.Name())
		}
		content, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		text := string(content)
		for _, forbidden := range []string{
			"pull_request_target:", "workflow_run:", "secrets.",
			"persist-credentials: true", "dangerously", "sudo ", "curl ", "wget ",
		} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s contains forbidden workflow text %q", entry.Name(), forbidden)
			}
		}
		for _, line := range strings.Split(text, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "uses:") && !pinnedActionPattern.MatchString(trimmed) {
				t.Errorf("%s has an unpinned or non-official Action: %s", entry.Name(), trimmed)
			}
		}
		if !strings.Contains(text, "GO_VERSION: \"1.25.12\"") {
			t.Errorf("%s does not pin the approved Go security patch", entry.Name())
		}
	}
}

func TestCIWorkflowIsReadOnly(t *testing.T) {
	content := readWorkflow(t, "ci.yml")
	if !strings.Contains(content, "permissions:\n  contents: read") {
		t.Fatal("CI must declare global contents: read")
	}
	if strings.Contains(content, "contents: write") ||
		strings.Contains(content, "id-token: write") ||
		strings.Contains(content, "attestations: write") {
		t.Fatal("CI must not request write or identity permissions")
	}
}

func TestReleaseWritePermissionsAreScopedToPublishJob(t *testing.T) {
	content := readWorkflow(t, "release.yml")
	for _, permission := range []string{"contents: write", "id-token: write", "attestations: write"} {
		if strings.Count(content, permission) != 1 {
			t.Fatalf("release workflow must declare %q exactly once", permission)
		}
	}
	global := strings.SplitN(content, "jobs:", 2)[0]
	if !strings.Contains(global, "permissions:\n  contents: read") ||
		strings.Contains(global, "write") {
		t.Fatal("release workflow global permissions must remain contents: read")
	}
}

func readWorkflow(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", name))
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(content), "\r\n", "\n")
}
