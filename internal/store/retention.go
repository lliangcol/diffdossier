package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lliangcol/diffdossier/internal/snapshot"
)

type ArchiveRecord struct {
	SchemaVersion string    `json:"schema_version"`
	RepositoryID  string    `json:"repository_id"`
	RunID         string    `json:"run_id"`
	RunState      string    `json:"run_state"`
	ArchivedAt    time.Time `json:"archived_at"`
	Pinned        bool      `json:"pinned"`
	Reason        string    `json:"reason"`
	LastEventHash string    `json:"last_event_hash"`
	ContentDigest string    `json:"content_digest"`
	RecordDigest  string    `json:"record_digest"`
}

type GCCandidate struct {
	RepositoryID  string `json:"repository_id"`
	RunID         string `json:"run_id"`
	ArchiveDigest string `json:"archive_digest"`
	ContentDigest string `json:"content_digest"`
}

type GCPlan struct {
	SchemaVersion string        `json:"schema_version"`
	RepositoryID  string        `json:"repository_id"`
	AsOf          time.Time     `json:"as_of"`
	RetentionDays int           `json:"retention_days"`
	Candidates    []GCCandidate `json:"candidates"`
	BlobDigests   []string      `json:"blob_digests"`
	PlanDigest    string        `json:"plan_digest"`
}

type GCExecution struct {
	SchemaVersion string    `json:"schema_version"`
	PlanDigest    string    `json:"plan_digest"`
	ExecutedAt    time.Time `json:"executed_at"`
	RemovedRuns   int       `json:"removed_runs"`
	RemovedBlobs  int       `json:"removed_blobs"`
}

type archiveEventPayload struct {
	Pinned bool   `json:"pinned"`
	Reason string `json:"reason"`
	State  string `json:"state"`
}

func (store *Store) ArchiveRun(repositoryID, runID, reason string, pinned bool, now time.Time) (ArchiveRecord, error) {
	if !validID(repositoryID, "repo-", 32) || !validID(runID, "run-", 24) {
		return ArchiveRecord{}, errors.New("invalid repository or run ID")
	}
	if strings.TrimSpace(reason) == "" {
		return ArchiveRecord{}, errors.New("archive reason is required")
	}
	if now.IsZero() {
		return ArchiveRecord{}, errors.New("archive time is required")
	}
	global, err := acquire(filepath.Join(store.Root, "locks", "gc.lock"))
	if err != nil {
		return ArchiveRecord{}, err
	}
	defer global.Release()

	runDir := filepath.Join(store.Root, "repositories", repositoryID, "runs", runID)
	if err := requireRealDirectory(runDir); err != nil {
		return ArchiveRecord{}, err
	}
	runLock, err := AcquireRunLock(runDir)
	if err != nil {
		return ArchiveRecord{}, err
	}
	defer runLock.Release()
	if _, err := os.Lstat(filepath.Join(runDir, "archive.json")); err == nil {
		return ArchiveRecord{}, ErrRunArchived
	} else if !os.IsNotExist(err) {
		return ArchiveRecord{}, err
	}
	var run Run
	if err := readJSON(filepath.Join(runDir, "run.json"), &run); err != nil {
		return ArchiveRecord{}, err
	}
	if run.ID != runID || run.RepositoryID != repositoryID {
		return ArchiveRecord{}, errors.New("run identity does not match archive target")
	}
	if run.State != "FINALIZED" && run.State != "EXPORTED" && run.State != "BLOCKED" {
		return ArchiveRecord{}, fmt.Errorf("run state %s is not terminal enough to archive", run.State)
	}
	if err := VerifyEventChain(runDir); err != nil {
		return ArchiveRecord{}, err
	}
	if err := VerifyRunJournal(runDir, run); err != nil {
		return ArchiveRecord{}, err
	}
	last, err := lastEvent(filepath.Join(runDir, "events.jsonl"))
	if err != nil {
		return ArchiveRecord{}, err
	}
	payload := archiveEventPayload{Pinned: pinned, Reason: strings.TrimSpace(reason), State: run.State}
	if last.Type == "run_archived" {
		var recorded archiveEventPayload
		if err := json.Unmarshal(last.Payload, &recorded); err != nil || recorded != payload {
			return ArchiveRecord{}, errors.New("existing archive event does not match requested archive")
		}
		now = last.RecordedAt
	} else {
		last, err = store.AppendEvent(runDir, "run_archived", payload)
		if err != nil {
			return ArchiveRecord{}, err
		}
		now = last.RecordedAt
	}
	contentDigest, err := runContentDigest(runDir)
	if err != nil {
		return ArchiveRecord{}, err
	}
	record := ArchiveRecord{
		SchemaVersion: "1.0",
		RepositoryID:  repositoryID,
		RunID:         runID,
		RunState:      run.State,
		ArchivedAt:    now.UTC(),
		Pinned:        pinned,
		Reason:        strings.TrimSpace(reason),
		LastEventHash: last.EventHash,
		ContentDigest: contentDigest,
	}
	record.RecordDigest = digestJSON(record)
	if err := atomicJSON(filepath.Join(runDir, "archive.json"), record); err != nil {
		return ArchiveRecord{}, err
	}
	return record, nil
}

func (store *Store) PlanGC(repositoryID string, retentionDays int, asOf time.Time) (GCPlan, error) {
	if !validID(repositoryID, "repo-", 32) {
		return GCPlan{}, errors.New("invalid repository ID")
	}
	if retentionDays < 1 || asOf.IsZero() {
		return GCPlan{}, errors.New("positive retention and exact as-of time are required")
	}
	global, err := acquire(filepath.Join(store.Root, "locks", "gc.lock"))
	if err != nil {
		return GCPlan{}, err
	}
	defer global.Release()

	cutoff := asOf.UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	runDirs, err := filepath.Glob(filepath.Join(store.Root, "repositories", repositoryID, "runs", "run-*"))
	if err != nil {
		return GCPlan{}, err
	}
	sort.Strings(runDirs)
	plan := GCPlan{SchemaVersion: "1.0", RepositoryID: repositoryID, AsOf: asOf.UTC(), RetentionDays: retentionDays}
	candidateKeys := map[string]bool{}
	for _, runDir := range runDirs {
		if err := requireRealDirectory(runDir); err != nil {
			return GCPlan{}, err
		}
		record, err := readArchiveRecord(runDir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return GCPlan{}, err
		}
		if record.RepositoryID != repositoryID ||
			record.RunID != filepath.Base(runDir) ||
			!validDigest(record.ContentDigest) ||
			!validDigest(record.LastEventHash) {
			return GCPlan{}, errors.New("archive record identity or digest is invalid")
		}
		if record.Pinned ||
			record.RunState != "EXPORTED" ||
			record.ArchivedAt.After(cutoff) {
			continue
		}
		protected, err := publicEvidenceProtected(runDir)
		if err != nil {
			return GCPlan{}, err
		}
		if protected {
			continue
		}
		if err := verifyArchivedRun(runDir, record); err != nil {
			return GCPlan{}, err
		}
		plan.Candidates = append(plan.Candidates, GCCandidate{
			RepositoryID:  repositoryID,
			RunID:         record.RunID,
			ArchiveDigest: record.RecordDigest,
			ContentDigest: record.ContentDigest,
		})
		candidateKeys[repositoryID+"/"+record.RunID] = true
	}
	retained, candidateRefs, err := store.blobReferences(candidateKeys)
	if err != nil {
		return GCPlan{}, err
	}
	for digest := range candidateRefs {
		if !retained[digest] {
			plan.BlobDigests = append(plan.BlobDigests, digest)
		}
	}
	sort.Strings(plan.BlobDigests)
	plan.PlanDigest = digestJSON(plan)
	if err := atomicJSON(store.gcPlanPath(plan.PlanDigest), plan); err != nil {
		return GCPlan{}, err
	}
	return plan, nil
}

func (store *Store) ExecuteGC(expectedPlanDigest string, now time.Time) (GCExecution, error) {
	if !validDigest(expectedPlanDigest) || now.IsZero() {
		return GCExecution{}, errors.New("exact GC plan digest and execution time are required")
	}
	global, err := acquire(filepath.Join(store.Root, "locks", "gc.lock"))
	if err != nil {
		return GCExecution{}, err
	}
	defer global.Release()

	executionPath := store.gcExecutionPath(expectedPlanDigest)
	var completed GCExecution
	if err := readJSON(executionPath, &completed); err == nil {
		if completed.PlanDigest != expectedPlanDigest {
			return GCExecution{}, errors.New("GC execution record does not match requested plan")
		}
		if err := os.RemoveAll(store.gcTrashPath(expectedPlanDigest)); err != nil {
			return GCExecution{}, err
		}
		return completed, nil
	} else if !os.IsNotExist(err) {
		return GCExecution{}, err
	}
	var plan GCPlan
	if err := readJSON(store.gcPlanPath(expectedPlanDigest), &plan); err != nil {
		return GCExecution{}, err
	}
	claimed := plan.PlanDigest
	plan.PlanDigest = ""
	if claimed != expectedPlanDigest || digestJSON(plan) != expectedPlanDigest {
		return GCExecution{}, errors.New("GC plan integrity check failed")
	}
	plan.PlanDigest = claimed
	if !validID(plan.RepositoryID, "repo-", 32) || plan.RetentionDays < 1 || plan.AsOf.IsZero() {
		return GCExecution{}, errors.New("GC plan metadata is invalid")
	}
	candidateKeys := map[string]bool{}
	for _, candidate := range plan.Candidates {
		if candidate.RepositoryID != plan.RepositoryID ||
			!validID(candidate.RepositoryID, "repo-", 32) ||
			!validID(candidate.RunID, "run-", 24) ||
			!validDigest(candidate.ArchiveDigest) ||
			!validDigest(candidate.ContentDigest) {
			return GCExecution{}, errors.New("GC candidate identity is invalid")
		}
		candidateKeys[candidate.RepositoryID+"/"+candidate.RunID] = true
		runDir, err := store.candidateLocation(expectedPlanDigest, candidate)
		if err != nil {
			return GCExecution{}, err
		}
		record, err := readArchiveRecord(runDir)
		if err != nil || record.RecordDigest != candidate.ArchiveDigest || record.ContentDigest != candidate.ContentDigest {
			return GCExecution{}, errors.New("archived run no longer matches GC plan")
		}
		if record.Pinned || record.RunState != "EXPORTED" || record.ArchivedAt.After(plan.AsOf.Add(-time.Duration(plan.RetentionDays)*24*time.Hour)) {
			return GCExecution{}, errors.New("archived run is no longer eligible for GC")
		}
		protected, protectErr := publicEvidenceProtected(runDir)
		if protectErr != nil || protected {
			return GCExecution{}, errors.New("archived run contains protected public-export evidence")
		}
		if err := verifyArchivedRun(runDir, record); err != nil {
			return GCExecution{}, err
		}
	}
	retained, candidateRefs, err := store.blobReferences(candidateKeys)
	if err != nil {
		return GCExecution{}, err
	}
	for _, candidate := range plan.Candidates {
		runDir, err := store.candidateLocation(expectedPlanDigest, candidate)
		if err != nil {
			return GCExecution{}, err
		}
		if err := addSnapshotReferences(filepath.Join(runDir, "snapshot.json"), candidateRefs); err != nil {
			return GCExecution{}, err
		}
	}
	expectedBlobs := make([]string, 0)
	for digest := range candidateRefs {
		if !retained[digest] {
			expectedBlobs = append(expectedBlobs, digest)
		}
	}
	sort.Strings(expectedBlobs)
	if !equalStrings(expectedBlobs, plan.BlobDigests) {
		return GCExecution{}, errors.New("blob reference set changed after GC planning")
	}

	trashRoot := store.gcTrashPath(expectedPlanDigest)
	for _, candidate := range plan.Candidates {
		source := filepath.Join(store.Root, "repositories", candidate.RepositoryID, "runs", candidate.RunID)
		if _, err := os.Lstat(source); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return GCExecution{}, err
		}
		target := filepath.Join(trashRoot, candidate.RepositoryID, candidate.RunID)
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return GCExecution{}, err
		}
		if err := os.Rename(source, target); err != nil {
			return GCExecution{}, err
		}
	}
	for _, digest := range plan.BlobDigests {
		path, err := store.blobPath(digest)
		if err != nil {
			return GCExecution{}, err
		}
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			return GCExecution{}, errors.New("planned blob is not a regular file")
		}
		content, err := os.ReadFile(path)
		if err != nil || bytesDigest(content) != digest {
			return GCExecution{}, errors.New("planned blob content does not match its digest")
		}
		if err := os.Remove(path); err != nil {
			return GCExecution{}, err
		}
	}
	execution := GCExecution{
		SchemaVersion: "1.0",
		PlanDigest:    expectedPlanDigest,
		ExecutedAt:    now.UTC(),
		RemovedRuns:   len(plan.Candidates),
		RemovedBlobs:  len(plan.BlobDigests),
	}
	if err := atomicJSON(executionPath, execution); err != nil {
		return GCExecution{}, err
	}
	if err := os.RemoveAll(trashRoot); err != nil {
		return GCExecution{}, err
	}
	return execution, nil
}

func readArchiveRecord(runDir string) (ArchiveRecord, error) {
	info, err := os.Lstat(filepath.Join(runDir, "archive.json"))
	if err != nil {
		return ArchiveRecord{}, err
	}
	if !info.Mode().IsRegular() {
		return ArchiveRecord{}, errors.New("archive record is not a regular file")
	}
	var record ArchiveRecord
	if err := readJSON(filepath.Join(runDir, "archive.json"), &record); err != nil {
		return ArchiveRecord{}, err
	}
	claimed := record.RecordDigest
	record.RecordDigest = ""
	if claimed == "" || digestJSON(record) != claimed {
		return ArchiveRecord{}, errors.New("archive record integrity check failed")
	}
	record.RecordDigest = claimed
	return record, nil
}

func verifyArchivedRun(runDir string, record ArchiveRecord) error {
	if err := VerifyEventChain(runDir); err != nil {
		return err
	}
	last, err := lastEvent(filepath.Join(runDir, "events.jsonl"))
	if err != nil || last.Type != "run_archived" || last.EventHash != record.LastEventHash {
		return errors.New("archive event binding is invalid")
	}
	var payload archiveEventPayload
	if err := json.Unmarshal(last.Payload, &payload); err != nil ||
		payload.Pinned != record.Pinned ||
		payload.Reason != record.Reason ||
		payload.State != record.RunState ||
		!last.RecordedAt.Equal(record.ArchivedAt) {
		return errors.New("archive metadata does not match archive event")
	}
	digest, err := runContentDigest(runDir)
	if err != nil {
		return err
	}
	if digest != record.ContentDigest {
		return errors.New("archived run content changed")
	}
	return nil
}

func runContentDigest(runDir string) (string, error) {
	digests := map[string]string{}
	err := filepath.WalkDir(runDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(runDir, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == "." {
			return nil
		}
		if relative == "locks" || strings.HasPrefix(relative, "locks/") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if relative == "archive.json" {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("archived run contains non-regular entry %s", relative)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digests[relative] = bytesDigest(content)
		return nil
	})
	if err != nil {
		return "", err
	}
	return digestStringMap(digests), nil
}

func publicEvidenceProtected(runDir string) (bool, error) {
	for _, pattern := range []string{
		filepath.Join(runDir, "approvals", "public-*"),
		filepath.Join(runDir, "approvals", "redaction-*"),
		filepath.Join(runDir, "exports", "public-bundle*"),
		filepath.Join(runDir, "exports", "public-tombstone*"),
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return false, err
		}
		if len(matches) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (store *Store) blobReferences(candidates map[string]bool) (map[string]bool, map[string]bool, error) {
	retained := map[string]bool{}
	candidateRefs := map[string]bool{}
	repositories, err := filepath.Glob(filepath.Join(store.Root, "repositories", "repo-*"))
	if err != nil {
		return nil, nil, err
	}
	for _, repositoryDir := range repositories {
		repositoryID := filepath.Base(repositoryDir)
		runDirs, err := filepath.Glob(filepath.Join(repositoryDir, "runs", "run-*"))
		if err != nil {
			return nil, nil, err
		}
		for _, runDir := range runDirs {
			var seal snapshot.Seal
			if err := readJSON(filepath.Join(runDir, "snapshot.json"), &seal); err != nil {
				return nil, nil, err
			}
			target := retained
			if candidates[repositoryID+"/"+filepath.Base(runDir)] {
				target = candidateRefs
			}
			for _, entry := range seal.Inventory.Entries {
				addEntryReferences(entry.ContentHash, entry.PreviousContentHash, target)
			}
		}
	}
	return retained, candidateRefs, nil
}

func addSnapshotReferences(path string, target map[string]bool) error {
	var seal snapshot.Seal
	if err := readJSON(path, &seal); err != nil {
		return err
	}
	for _, entry := range seal.Inventory.Entries {
		addEntryReferences(entry.ContentHash, entry.PreviousContentHash, target)
	}
	return nil
}

func addEntryReferences(current, previous string, target map[string]bool) {
	if validDigest(current) {
		target[current] = true
	}
	if validDigest(previous) {
		target[previous] = true
	}
}

func (store *Store) candidateLocation(planDigest string, candidate GCCandidate) (string, error) {
	active := filepath.Join(store.Root, "repositories", candidate.RepositoryID, "runs", candidate.RunID)
	if _, err := os.Lstat(active); err == nil {
		return active, requireRealDirectory(active)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	trash := filepath.Join(store.gcTrashPath(planDigest), candidate.RepositoryID, candidate.RunID)
	if _, err := os.Lstat(trash); err != nil {
		return "", errors.New("planned archived run is missing")
	}
	return trash, requireRealDirectory(trash)
}

func requireRealDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("run path must be a real directory")
	}
	return nil
}

func (store *Store) blobPath(digest string) (string, error) {
	if !validDigest(digest) {
		return "", errors.New("invalid blob digest")
	}
	return filepath.Join(store.Root, "blobs", "sha256", strings.TrimPrefix(digest, "sha256:")), nil
}

func (store *Store) gcPlanPath(digest string) string {
	return filepath.Join(store.Root, "gc", "plans", strings.TrimPrefix(digest, "sha256:")+".json")
}

func (store *Store) gcExecutionPath(digest string) string {
	return filepath.Join(store.Root, "gc", "executions", strings.TrimPrefix(digest, "sha256:")+".json")
}

func (store *Store) gcTrashPath(digest string) string {
	return filepath.Join(store.Root, "gc", "trash", strings.TrimPrefix(digest, "sha256:"))
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func digestJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return bytesDigest(encoded)
}

func bytesDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func digestStringMap(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hash := sha256.New()
	for _, key := range keys {
		hash.Write([]byte(key))
		hash.Write([]byte{0})
		hash.Write([]byte(values[key]))
		hash.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
