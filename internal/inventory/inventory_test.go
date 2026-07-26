package inventory

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lliangcol/diffdossier/internal/gitrepo"
)

func TestCaptureSeparatesScopesAndSpecialPaths(t *testing.T) {
	dir := initFixture(t)
	base := gitOutput(t, dir, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("staged\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "tracked.txt")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("unstaged\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	specialPaths := inventoryFixturePaths(runtime.GOOS)
	for _, name := range specialPaths {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	repo, err := gitrepo.Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	revisions, err := repo.Resolve(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Capture(context.Background(), repo, revisions)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[Scope]int{}
	for _, entry := range result.Entries {
		counts[entry.Scope]++
		if entry.Path.BytesBase64 == "" || entry.ContentHash == "" {
			t.Fatalf("missing path identity or content hash: %+v", entry)
		}
	}
	if counts[ScopeStaged] != 1 || counts[ScopeUnstaged] != 1 || counts[ScopeUntracked] != len(specialPaths) {
		t.Fatalf("unexpected scope counts: %#v", counts)
	}
}

func inventoryFixturePaths(goos string) []string {
	paths := []string{"space name.txt", "中文-🙂.txt"}
	if goos != "windows" {
		paths = append(paths, "line\nbreak.txt")
	}
	return paths
}

func TestInventoryFixturePathsRespectOperatingSystemRules(t *testing.T) {
	for _, name := range inventoryFixturePaths("windows") {
		if strings.ContainsAny(name, "\t\n\r") {
			t.Fatalf("Windows fixture contains a forbidden control character: %q", name)
		}
	}
	if len(inventoryFixturePaths("linux")) != len(inventoryFixturePaths("windows"))+1 {
		t.Fatal("POSIX fixture must retain newline path coverage")
	}
}

func TestCaptureInvalidUTF8PathOnPOSIX(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("fixture requires a filesystem that accepts invalid UTF-8 path bytes")
	}
	dir := initFixture(t)
	name := string([]byte{'i', 'n', 'v', 0xff})
	if err := os.WriteFile(filepath.Join(dir, name), []byte("bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo, _ := gitrepo.Open(context.Background(), dir)
	revisions, _ := repo.Resolve(context.Background(), "HEAD")
	result, err := Capture(context.Background(), repo, revisions)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range result.Entries {
		if entry.Path.UTF8 == nil {
			return
		}
	}
	t.Fatal("invalid UTF-8 path did not preserve a base64-only identity")
}

func TestCaptureLFSPointer(t *testing.T) {
	dir := initFixture(t)
	pointer := "version https://git-lfs.github.com/spec/v1\noid sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\nsize 42\n"
	if err := os.WriteFile(filepath.Join(dir, "asset.bin"), []byte(pointer), 0o600); err != nil {
		t.Fatal(err)
	}
	repo, _ := gitrepo.Open(context.Background(), dir)
	revisions, _ := repo.Resolve(context.Background(), "HEAD")
	result, err := Capture(context.Background(), repo, revisions)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range result.Entries {
		if entry.Kind == "lfs_pointer" && entry.LFSSize == 42 {
			return
		}
	}
	t.Fatal("LFS pointer metadata not detected")
}

func TestCaptureExplicitIgnoredPath(t *testing.T) {
	dir := initFixture(t)
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("ignored.txt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("included"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo, _ := gitrepo.Open(context.Background(), dir)
	revisions, _ := repo.Resolve(context.Background(), "HEAD")
	result, err := Capture(context.Background(), repo, revisions, Options{IncludeUntracked: true, IncludeIgnored: []string{"ignored.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range result.Entries {
		if entry.Scope == ScopeIgnored {
			return
		}
	}
	t.Fatal("explicit ignored path was not captured")
}

func TestCaptureFailsClosedOnBlobAndTotalBudgets(t *testing.T) {
	dir := initFixture(t)
	if err := os.WriteFile(filepath.Join(dir, "large.txt"), []byte("12345"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo, _ := gitrepo.Open(context.Background(), dir)
	revisions, _ := repo.Resolve(context.Background(), "HEAD")
	if _, err := Capture(context.Background(), repo, revisions, Options{IncludeUntracked: true, MaxBlobBytes: 4, MaxTotalBytes: 100}); err == nil {
		t.Fatal("blob budget overflow accepted")
	}
	if _, err := Capture(context.Background(), repo, revisions, Options{IncludeUntracked: true, MaxBlobBytes: 100, MaxTotalBytes: 4}); err == nil {
		t.Fatal("total budget overflow accepted")
	}
}

func TestCaptureRenameDeleteBinaryAndSymlink(t *testing.T) {
	dir := initFixture(t)
	base := gitOutput(t, dir, "rev-parse", "HEAD")
	run(t, dir, "git", "mv", "tracked.txt", "renamed.txt")
	if err := os.WriteFile(filepath.Join(dir, "binary.bin"), []byte{0, 1, 2}, 0o600); err != nil {
		t.Fatal(err)
	}
	symlinkAvailable := os.Symlink("renamed.txt", filepath.Join(dir, "link")) == nil
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-qm", "change")
	repo, _ := gitrepo.Open(context.Background(), dir)
	revisions, _ := repo.Resolve(context.Background(), base)
	result, err := Capture(context.Background(), repo, revisions)
	if err != nil {
		t.Fatal(err)
	}
	seenRename, seenBinary, seenLink := false, false, false
	for _, entry := range result.Entries {
		seenRename = seenRename || entry.OldPath != nil
		seenBinary = seenBinary || entry.Kind == "binary"
		seenLink = seenLink || entry.Kind == "symlink"
	}
	if !seenRename || !seenBinary || (symlinkAvailable && !seenLink) {
		t.Fatalf("rename=%v binary=%v symlink=%v entries=%+v", seenRename, seenBinary, seenLink, result.Entries)
	}
	if !symlinkAvailable {
		t.Log("symlink creation is unavailable on this host; rename and binary coverage still ran")
	}
}

func TestCaptureDeletionKeepsPreviousBlobAndMarksCurrentMissing(t *testing.T) {
	dir := initFixture(t)
	base := gitOutput(t, dir, "rev-parse", "HEAD")
	if err := os.Remove(filepath.Join(dir, "tracked.txt")); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "-A")
	repo, _ := gitrepo.Open(context.Background(), dir)
	revisions, _ := repo.Resolve(context.Background(), base)
	result, err := Capture(context.Background(), repo, revisions)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range result.Entries {
		if entry.Scope == ScopeStaged && entry.Status == "D" {
			if entry.Kind != "missing" || entry.ContentHash != "" || entry.PreviousContentHash == "" || string(entry.PreviousContent) != "initial\n" {
				t.Fatalf("unexpected deletion entry: %+v", entry)
			}
			return
		}
	}
	t.Fatal("staged deletion not captured")
}

func TestCaptureSubmoduleCommitAndDirtyState(t *testing.T) {
	sub := initFixture(t)
	parent := initFixture(t)
	base := gitOutput(t, parent, "rev-parse", "HEAD")
	run(t, parent, "git", "-c", "protocol.file.allow=always", "submodule", "add", "-q", sub, "vendor/sub")
	run(t, parent, "git", "commit", "-qm", "add submodule")
	repo, _ := gitrepo.Open(context.Background(), parent)
	revisions, _ := repo.Resolve(context.Background(), base)
	result, err := Capture(context.Background(), repo, revisions)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range result.Entries {
		if entry.Kind == "submodule" && entry.GitObject != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("submodule gitlink metadata not captured")
	}
	if err := os.WriteFile(filepath.Join(parent, "vendor", "sub", "dirty.txt"), []byte("dirty"), 0o600); err != nil {
		t.Fatal(err)
	}
	dirtyResult, err := Capture(context.Background(), repo, revisions)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range dirtyResult.Entries {
		if entry.Scope == ScopeUnstaged && entry.Kind == "submodule" && entry.SubmoduleDirty {
			return
		}
	}
	t.Fatal("dirty submodule state not captured")
}

func initFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init", "-q")
	run(t, dir, "git", "config", "user.email", "fixture@example.invalid")
	run(t, dir, "git", "config", "user.name", "Fixture")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("initial\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "tracked.txt")
	run(t, dir, "git", "commit", "-qm", "initial")
	return dir
}

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(bytesTrimSpace(output))
}

func bytesTrimSpace(value []byte) []byte {
	start, end := 0, len(value)
	for start < end && (value[start] == ' ' || value[start] == '\n' || value[start] == '\r' || value[start] == '\t') {
		start++
	}
	for end > start && (value[end-1] == ' ' || value[end-1] == '\n' || value[end-1] == '\r' || value[end-1] == '\t') {
		end--
	}
	return value[start:end]
}
