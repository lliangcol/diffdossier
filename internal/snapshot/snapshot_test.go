package snapshot

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lliangcol/diffdossier/internal/gitrepo"
)

func TestDeterministicAndStale(t *testing.T) {
	dir := fixtureRepo(t)
	repo, err := gitrepo.Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	request := Request{Repo: repo, Baseline: "HEAD", IncludeUntracked: true, InputDigests: map[string]string{"config": "sha256:a"}}
	first, err := Capture(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Capture(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.SnapshotID != second.SnapshotID {
		t.Fatalf("same input produced %s and %s", first.SnapshotID, second.SnapshotID)
	}
	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("change"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyFresh(context.Background(), request, first); err == nil {
		t.Fatal("worktree mutation must make snapshot stale")
	}
}

func TestInputDigestInvalidatesSeal(t *testing.T) {
	dir := fixtureRepo(t)
	repo, _ := gitrepo.Open(context.Background(), dir)
	one, _ := Capture(context.Background(), Request{Repo: repo, Baseline: "HEAD", IncludeUntracked: true, InputDigests: map[string]string{"config": "sha256:a"}})
	two, _ := Capture(context.Background(), Request{Repo: repo, Baseline: "HEAD", IncludeUntracked: true, InputDigests: map[string]string{"config": "sha256:b"}})
	if one.SnapshotID == two.SnapshotID {
		t.Fatal("changed semantic input digest must invalidate seal")
	}
}

func fixtureRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init", "-q")
	run(t, dir, "git", "config", "user.email", "fixture@example.invalid")
	run(t, dir, "git", "config", "user.name", "Fixture")
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "file.txt")
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
