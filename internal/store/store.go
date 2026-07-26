// Package store persists durable DiffDossier state outside target repositories.
package store

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lliangcol/diffdossier/internal/platform"
	"github.com/lliangcol/diffdossier/internal/snapshot"
)

type Store struct {
	Root string
}

type Repository struct {
	ID        string    `json:"repository_id"`
	LocalRoot string    `json:"local_root"`
	CreatedAt time.Time `json:"created_at"`
}

type Run struct {
	SchemaVersion string    `json:"schema_version"`
	ID            string    `json:"run_id"`
	RepositoryID  string    `json:"repository_id"`
	SnapshotID    string    `json:"snapshot_id"`
	State         string    `json:"state"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Event struct {
	Sequence     uint64          `json:"sequence"`
	Type         string          `json:"type"`
	RecordedAt   time.Time       `json:"recorded_at"`
	Payload      json.RawMessage `json:"payload"`
	PreviousHash string          `json:"previous_hash,omitempty"`
	EventHash    string          `json:"event_hash"`
}

type Lock struct {
	path string
	file *os.File
}

func Open(root string) (*Store, error) {
	if err := platform.EnsurePrivateDir(root); err != nil {
		return nil, fmt.Errorf("prepare state root: %w", err)
	}
	for _, relative := range []string{"repositories", filepath.Join("blobs", "sha256")} {
		if err := platform.EnsurePrivateDir(filepath.Join(root, relative)); err != nil {
			return nil, err
		}
	}
	return &Store{Root: filepath.Clean(root)}, nil
}

func (store *Store) Register(localRoot string) (Repository, error) {
	canonical, err := filepath.Abs(localRoot)
	if err != nil {
		return Repository{}, err
	}
	lock, err := acquire(filepath.Join(store.Root, "registry.lock"))
	if err != nil {
		return Repository{}, err
	}
	defer lock.Release()

	path := filepath.Join(store.Root, "repositories.json")
	var repositories []Repository
	if content, readErr := os.ReadFile(path); readErr == nil {
		if err := json.Unmarshal(content, &repositories); err != nil {
			return Repository{}, fmt.Errorf("decode repository registry: %w", err)
		}
	} else if !os.IsNotExist(readErr) {
		return Repository{}, readErr
	}
	for _, repository := range repositories {
		if filepath.Clean(repository.LocalRoot) == filepath.Clean(canonical) {
			return repository, nil
		}
	}
	id, err := randomID("repo-", 16)
	if err != nil {
		return Repository{}, err
	}
	repository := Repository{ID: id, LocalRoot: canonical, CreatedAt: time.Now().UTC()}
	repositories = append(repositories, repository)
	sort.Slice(repositories, func(i, j int) bool { return repositories[i].ID < repositories[j].ID })
	if err := atomicJSON(path, repositories); err != nil {
		return Repository{}, err
	}
	repositoryDir := filepath.Join(store.Root, "repositories", repository.ID)
	if err := atomicJSON(filepath.Join(repositoryDir, "repository.json"), repository); err != nil {
		return Repository{}, err
	}
	return repository, nil
}

func (store *Store) BeginRun(repository Repository, seal snapshot.Seal) (Run, string, error) {
	id, err := randomID("run-", 12)
	if err != nil {
		return Run{}, "", err
	}
	runDir := filepath.Join(store.Root, "repositories", repository.ID, "runs", id)
	for _, relative := range []string{
		"", "tasks", "packets", "results", "gates", "approvals", "logs", "locks", "reports",
	} {
		if err := platform.EnsurePrivateDir(filepath.Join(runDir, relative)); err != nil {
			return Run{}, "", err
		}
	}
	now := time.Now().UTC()
	run := Run{
		SchemaVersion: "1.0", ID: id, RepositoryID: repository.ID,
		SnapshotID: seal.SnapshotID, State: "PREPARED", CreatedAt: now, UpdatedAt: now,
	}
	if err := atomicJSON(filepath.Join(runDir, "run.json"), run); err != nil {
		return Run{}, "", err
	}
	if err := atomicJSON(filepath.Join(runDir, "snapshot.json"), seal); err != nil {
		return Run{}, "", err
	}
	if err := atomicJSON(filepath.Join(runDir, "inventory.json"), seal.Inventory); err != nil {
		return Run{}, "", err
	}
	for _, entry := range seal.Inventory.Entries {
		if entry.ContentHash != "" {
			if err := store.writeBlob(entry.ContentHash, entry.Content); err != nil {
				return Run{}, "", fmt.Errorf("persist current blob for %s: %w", entry.Path.BytesBase64, err)
			}
		}
		if entry.PreviousContentHash != "" {
			if err := store.writeBlob(entry.PreviousContentHash, entry.PreviousContent); err != nil {
				return Run{}, "", fmt.Errorf("persist previous blob for %s: %w", entry.Path.BytesBase64, err)
			}
		}
	}
	if _, err := store.AppendEvent(runDir, "run_prepared", map[string]string{"snapshot_id": seal.SnapshotID}); err != nil {
		return Run{}, "", err
	}
	return run, runDir, nil
}

func (store *Store) LoadRun(repositoryID, runID string) (Run, snapshot.Seal, error) {
	runDir := filepath.Join(store.Root, "repositories", repositoryID, "runs", runID)
	var run Run
	if err := readJSON(filepath.Join(runDir, "run.json"), &run); err != nil {
		return Run{}, snapshot.Seal{}, err
	}
	var seal snapshot.Seal
	if err := readJSON(filepath.Join(runDir, "snapshot.json"), &seal); err != nil {
		return Run{}, snapshot.Seal{}, err
	}
	if err := VerifyEventChain(runDir); err != nil {
		return Run{}, snapshot.Seal{}, err
	}
	return run, seal, nil
}

func (store *Store) writeBlob(digest string, content []byte) error {
	const prefix = "sha256:"
	if len(digest) != len(prefix)+64 || digest[:len(prefix)] != prefix {
		return errors.New("invalid blob digest")
	}
	actual := sha256.Sum256(content)
	actualHex := hex.EncodeToString(actual[:])
	if actualHex != digest[len(prefix):] {
		return errors.New("blob content does not match digest")
	}
	path := filepath.Join(store.Root, "blobs", "sha256", actualHex)
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return atomicBytes(path, content)
}

func (store *Store) LatestRun(repositoryID string) (Run, error) {
	pattern := filepath.Join(store.Root, "repositories", repositoryID, "runs", "*", "run.json")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return Run{}, err
	}
	if len(paths) == 0 {
		return Run{}, errors.New("no runs found")
	}
	var latest Run
	for _, path := range paths {
		var candidate Run
		if err := readJSON(path, &candidate); err != nil {
			return Run{}, err
		}
		if candidate.UpdatedAt.After(latest.UpdatedAt) {
			latest = candidate
		}
	}
	return latest, nil
}

func (store *Store) RunDir(repositoryID, runID string) (string, error) {
	if !validID(repositoryID, "repo-", 32) || !validID(runID, "run-", 24) {
		return "", errors.New("invalid repository or run ID")
	}
	return filepath.Join(store.Root, "repositories", repositoryID, "runs", runID), nil
}

func (store *Store) WriteRunJSON(runDir, relative string, value any) error {
	target, err := runArtifactPath(runDir, relative)
	if err != nil {
		return err
	}
	return atomicJSON(target, value)
}

func (store *Store) ReadRunJSON(runDir, relative string, target any) error {
	path, err := runArtifactPath(runDir, relative)
	if err != nil {
		return err
	}
	return readJSON(path, target)
}

func runArtifactPath(runDir, relative string) (string, error) {
	clean := filepath.Clean(relative)
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("artifact path escapes run")
	}
	target := filepath.Join(runDir, clean)
	relativeCheck, err := filepath.Rel(runDir, target)
	if err != nil || relativeCheck == ".." || strings.HasPrefix(relativeCheck, ".."+string(filepath.Separator)) {
		return "", errors.New("artifact path escapes run")
	}
	return target, nil
}

func (store *Store) UpdateRunState(runDir, next string) (Run, error) {
	lock, err := AcquireRunLock(runDir)
	if err != nil {
		return Run{}, err
	}
	defer lock.Release()
	return store.UpdateRunStateHeld(runDir, next, lock)
}

// UpdateRunStateHeld advances state while the caller owns the run write lock.
func (store *Store) UpdateRunStateHeld(runDir, next string, lock *Lock) (Run, error) {
	expectedLock := filepath.Join(runDir, "locks", "write.lock")
	if lock == nil || lock.file == nil || filepath.Clean(lock.path) != filepath.Clean(expectedLock) {
		return Run{}, errors.New("valid run write lock is required")
	}
	var run Run
	if err := readJSON(filepath.Join(runDir, "run.json"), &run); err != nil {
		return Run{}, err
	}
	if !allowedTransition(run.State, next) {
		return Run{}, fmt.Errorf("invalid run transition %s -> %s", run.State, next)
	}
	previous := run.State
	if _, err := store.AppendEvent(runDir, "state_transition", map[string]string{"from": previous, "to": next}); err != nil {
		return Run{}, err
	}
	run.State = next
	run.UpdatedAt = time.Now().UTC()
	if err := atomicJSON(filepath.Join(runDir, "run.json"), run); err != nil {
		return Run{}, err
	}
	return run, nil
}

func (store *Store) AppendEvent(runDir, eventType string, payload any) (Event, error) {
	lock, err := acquire(filepath.Join(runDir, "locks", "events.lock"))
	if err != nil {
		return Event{}, err
	}
	defer lock.Release()
	eventsPath := filepath.Join(runDir, "events.jsonl")
	previous, err := lastEvent(eventsPath)
	if err != nil {
		return Event{}, err
	}
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	event := Event{
		Sequence: previous.Sequence + 1, Type: eventType, RecordedAt: time.Now().UTC(),
		Payload: encodedPayload, PreviousHash: previous.EventHash,
	}
	event.EventHash, err = eventDigest(event)
	if err != nil {
		return Event{}, err
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		return Event{}, err
	}
	file, err := os.OpenFile(eventsPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return Event{}, err
	}
	defer file.Close()
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		return Event{}, err
	}
	if err := file.Sync(); err != nil {
		return Event{}, err
	}
	return event, nil
}

func VerifyEventChain(runDir string) error {
	path := filepath.Join(runDir, "events.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var previous string
	var sequence uint64
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return fmt.Errorf("decode event %d: %w", sequence+1, err)
		}
		if event.Sequence != sequence+1 || event.PreviousHash != previous {
			return fmt.Errorf("event chain discontinuity at sequence %d", event.Sequence)
		}
		expected, err := eventDigest(event)
		if err != nil {
			return err
		}
		if event.EventHash != expected {
			return fmt.Errorf("event hash mismatch at sequence %d", event.Sequence)
		}
		sequence = event.Sequence
		previous = event.EventHash
	}
	return scanner.Err()
}

func AcquireRunLock(runDir string) (*Lock, error) {
	return acquire(filepath.Join(runDir, "locks", "write.lock"))
}

func (lock *Lock) Release() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	closeErr := lock.file.Close()
	removeErr := os.Remove(lock.path)
	lock.file = nil
	if closeErr != nil {
		return closeErr
	}
	return removeErr
}

func acquire(path string) (*Lock, error) {
	if err := platform.EnsurePrivateDir(filepath.Dir(path)); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("lock already held: %s", path)
		}
		return nil, err
	}
	return &Lock{path: path, file: file}, nil
}

func atomicJSON(path string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return atomicBytes(path, encoded)
}

func atomicBytes(path string, content []byte) error {
	if err := platform.EnsurePrivateDir(filepath.Dir(path)); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".diffdossier-write-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := replaceFile(temporaryPath, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func readJSON(path string, target any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(content, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func lastEvent(path string) (Event, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Event{}, nil
		}
		return Event{}, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var event Event
	for scanner.Scan() {
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return Event{}, fmt.Errorf("decode event log: %w", err)
		}
	}
	return event, scanner.Err()
}

func eventDigest(event Event) (string, error) {
	canonical := struct {
		Sequence     uint64          `json:"sequence"`
		Type         string          `json:"type"`
		RecordedAt   time.Time       `json:"recorded_at"`
		Payload      json.RawMessage `json:"payload"`
		PreviousHash string          `json:"previous_hash,omitempty"`
	}{event.Sequence, event.Type, event.RecordedAt, event.Payload, event.PreviousHash}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func randomID(prefix string, bytesCount int) (string, error) {
	value := make([]byte, bytesCount)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(value), nil
}

func validID(value, prefix string, hexLength int) bool {
	if len(value) != len(prefix)+hexLength || !strings.HasPrefix(value, prefix) {
		return false
	}
	_, err := hex.DecodeString(value[len(prefix):])
	return err == nil
}

func allowedTransition(current, next string) bool {
	allowed := map[string]map[string]bool{
		"PREPARED":       {"CONTRACTED": true, "BLOCKED": true},
		"CONTRACTED":     {"REVIEWING": true, "BLOCKED": true},
		"REVIEWING":      {"REVIEWED": true, "BLOCKED": true},
		"REVIEWED":       {"FIX_AUTHORIZED": true, "GATED": true, "BLOCKED": true},
		"FIX_AUTHORIZED": {"MUTATED": true, "BLOCKED": true},
		"MUTATED":        {"PREPARED": true, "BLOCKED": true},
		"GATED":          {"REREVIEWED": true, "BLOCKED": true},
		"REREVIEWED":     {"FINALIZED": true, "BLOCKED": true},
		"FINALIZED":      {"EXPORTED": true},
		"BLOCKED":        {"PREPARED": true, "REVIEWING": true},
	}
	return allowed[current][next]
}
