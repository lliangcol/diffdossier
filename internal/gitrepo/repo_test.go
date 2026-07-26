package gitrepo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestOpenAndResolveAreLocalOnly(t *testing.T) {
	repoDir := initRepo(t)
	repo, err := Open(context.Background(), repoDir)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := repo.Resolve(context.Background(), "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if evidence.BaselineCommit != evidence.HeadCommit || evidence.Freshness != "local_only" || evidence.RemoteFetchProof {
		t.Fatalf("unexpected evidence: %+v", evidence)
	}
}

func TestResolveRequiresBaseline(t *testing.T) {
	repoDir := initRepo(t)
	repo, err := Open(context.Background(), repoDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Resolve(context.Background(), ""); err == nil {
		t.Fatal("missing baseline should fail")
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init", "-q")
	run(t, dir, "git", "config", "user.email", "fixture@example.invalid")
	run(t, dir, "git", "config", "user.name", "Fixture")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("one\n"), 0o600); err != nil {
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

func TestBoundedBufferDoesNotGrowPastLimit(t *testing.T) {
	writer := &boundedBuffer{limit: 4}
	count, err := writer.Write([]byte("123456"))
	if err != nil || count != 6 {
		t.Fatalf("write=%d err=%v", count, err)
	}
	if string(writer.Bytes()) != "1234" || !writer.exceeded {
		t.Fatalf("buffer=%q exceeded=%t", writer.Bytes(), writer.exceeded)
	}
}
