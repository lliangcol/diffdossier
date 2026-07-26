package store

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lliangcol/diffdossier/internal/inventory"
	"github.com/lliangcol/diffdossier/internal/snapshot"
)

func TestRegisterIsStableButNotPathDerived(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state")
	store, err := Open(state)
	if err != nil {
		t.Fatal(err)
	}
	repoPath := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoPath, 0o700); err != nil {
		t.Fatal(err)
	}
	one, err := store.Register(repoPath)
	if err != nil {
		t.Fatal(err)
	}
	two, err := store.Register(repoPath)
	if err != nil {
		t.Fatal(err)
	}
	if one.ID != two.ID || len(one.ID) != len("repo-")+32 {
		t.Fatalf("repository identity is not stable random ID: %#v %#v", one, two)
	}
}

func TestRunPersistsAndLocks(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	repository, err := store.Register(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	seal := snapshot.Seal{SchemaVersion: "1.0", SnapshotID: "snap-test"}
	run, runDir, err := store.BeginRun(repository, seal)
	if err != nil {
		t.Fatal(err)
	}
	loaded, loadedSeal, err := store.LoadRun(repository.ID, run.ID)
	if err != nil || loaded.ID != run.ID || loadedSeal.SnapshotID != seal.SnapshotID {
		t.Fatalf("load run=%+v seal=%+v err=%v", loaded, loadedSeal, err)
	}
	lock, err := AcquireRunLock(runDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AcquireRunLock(runDir); err == nil {
		t.Fatal("second writer must not acquire run lock")
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendEvent(runDir, "test", map[string]bool{"ok": true}); err != nil {
		t.Fatal(err)
	}
	if err := VerifyEventChain(runDir); err != nil {
		t.Fatal(err)
	}
	latest, err := store.LatestRun(repository.ID)
	if err != nil || latest.ID != run.ID || time.Since(latest.CreatedAt) > time.Minute {
		t.Fatalf("latest=%+v err=%v", latest, err)
	}
}

func TestBeginRunPersistsContentAddressedBlob(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	repository, _ := store.Register(t.TempDir())
	content := []byte("review input")
	digest := sha256.Sum256(content)
	digestHex := hex.EncodeToString(digest[:])
	seal := snapshot.Seal{
		SchemaVersion: "1.0", SnapshotID: "snap-test",
		Inventory: inventory.Result{SchemaVersion: "1.0", Entries: []inventory.Entry{{
			ContentHash: "sha256:" + digestHex, Content: content, Size: int64(len(content)),
		}}},
	}
	if _, _, err := store.BeginRun(repository, seal); err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(filepath.Join(store.Root, "blobs", "sha256", digestHex))
	if err != nil || string(stored) != string(content) {
		t.Fatalf("stored=%q err=%v", stored, err)
	}
}
