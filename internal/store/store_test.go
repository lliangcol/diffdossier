package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

func TestArtifactContainmentAndStateTransition(t *testing.T) {
	store, _ := Open(filepath.Join(t.TempDir(), "state"))
	repository, _ := store.Register(t.TempDir())
	run, runDir, _ := store.BeginRun(repository, snapshot.Seal{SnapshotID: "snap-test"})
	if err := store.WriteRunJSON(runDir, "tasks/task.json", map[string]bool{"ok": true}); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteRunJSON(runDir, "../escape.json", map[string]bool{}); err == nil {
		t.Fatal("artifact path escape must fail")
	}
	updated, err := store.UpdateRunState(runDir, "CONTRACTED")
	if err != nil || updated.State != "CONTRACTED" {
		t.Fatalf("run=%+v err=%v", updated, err)
	}
	if _, err := store.UpdateRunState(runDir, "FINALIZED"); err == nil {
		t.Fatal("invalid transition must fail")
	}
	if run.State != "PREPARED" {
		t.Fatalf("original run mutated: %+v", run)
	}
}

func TestHeldStateTransitionRequiresMatchingRunLock(t *testing.T) {
	stateStore, _ := Open(filepath.Join(t.TempDir(), "state"))
	repository, _ := stateStore.Register(t.TempDir())
	_, runDir, _ := stateStore.BeginRun(repository, snapshot.Seal{SnapshotID: "snap-test"})
	if _, err := stateStore.UpdateRunStateHeld(runDir, "CONTRACTED", nil); err == nil {
		t.Fatal("state transition without run lock was accepted")
	}
	lock, err := AcquireRunLock(runDir)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	updated, err := stateStore.UpdateRunStateHeld(runDir, "CONTRACTED", lock)
	if err != nil || updated.State != "CONTRACTED" {
		t.Fatalf("run=%+v err=%v", updated, err)
	}
}

func TestLoadRunRejectsStateAndJournalTampering(t *testing.T) {
	stateStore, _ := Open(filepath.Join(t.TempDir(), "state"))
	repository, _ := stateStore.Register(t.TempDir())
	run, runDir, _ := stateStore.BeginRun(repository, snapshot.Seal{SnapshotID: "snap-test"})
	run.State = "FINALIZED"
	if err := atomicJSON(filepath.Join(runDir, "run.json"), run); err != nil {
		t.Fatal(err)
	}
	if _, _, err := stateStore.LoadRun(repository.ID, run.ID); err == nil {
		t.Fatal("tampered run state accepted")
	}
}

func TestLoadRunRejectsTruncatedTransitionJournal(t *testing.T) {
	stateStore, _ := Open(filepath.Join(t.TempDir(), "state"))
	repository, _ := stateStore.Register(t.TempDir())
	run, runDir, _ := stateStore.BeginRun(repository, snapshot.Seal{SnapshotID: "snap-test"})
	updated, err := stateStore.UpdateRunState(runDir, "CONTRACTED")
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(runDir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	newline := bytes.IndexByte(content, '\n')
	if newline < 0 {
		t.Fatal("missing first event")
	}
	if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), content[:newline+1], 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := stateStore.LoadRun(repository.ID, updated.ID); err == nil {
		t.Fatal("truncated transition journal accepted")
	}
	_ = run
}

func TestRecoverRunRequiresExactJournalState(t *testing.T) {
	stateStore, _ := Open(filepath.Join(t.TempDir(), "state"))
	repository, _ := stateStore.Register(t.TempDir())
	run, runDir, _ := stateStore.BeginRun(repository, snapshot.Seal{SnapshotID: "snap-test"})
	if _, err := stateStore.AppendEvent(runDir, "state_transition", map[string]string{"from": "PREPARED", "to": "CONTRACTED"}); err != nil {
		t.Fatal(err)
	}
	if _, err := stateStore.RecoverRun(repository.ID, run.ID, "REVIEWING"); err == nil {
		t.Fatal("wrong journal state authorized recovery")
	}
	recovered, err := stateStore.RecoverRun(repository.ID, run.ID, "CONTRACTED")
	if err != nil || recovered.State != "CONTRACTED" {
		t.Fatalf("recovered=%+v err=%v", recovered, err)
	}
	if _, _, err := stateStore.LoadRun(repository.ID, run.ID); err != nil {
		t.Fatal(err)
	}
}

func TestEventChainRejectsEmptyJournal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyEventChain(dir); err == nil {
		t.Fatal("empty event chain accepted")
	}
}

func TestStaleLockRecoveryAndOwnershipCheck(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lock")
	content, _ := json.Marshal(lockMetadata{PID: 2147483647, Token: "stale", CreatedAt: time.Now().Add(-time.Hour)})
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	replacement, _ := json.Marshal(lockMetadata{PID: os.Getpid(), Token: "other", CreatedAt: time.Now()})
	if err := os.WriteFile(path, replacement, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err == nil {
		t.Fatal("release removed another owner's lock")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("replacement lock was removed")
	}
}
