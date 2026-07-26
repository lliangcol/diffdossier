// Package snapshot creates deterministic, content-addressed review seals.
package snapshot

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"time"

	"github.com/lliangcol/diffdossier/internal/gitrepo"
	"github.com/lliangcol/diffdossier/internal/inventory"
)

type Seal struct {
	SchemaVersion string                   `json:"schema_version"`
	SnapshotID    string                   `json:"snapshot_id"`
	CapturedAt    time.Time                `json:"captured_at"`
	Revisions     gitrepo.RevisionEvidence `json:"revisions"`
	IndexTree     string                   `json:"index_tree"`
	GitConfig     map[string]string        `json:"git_config"`
	InputDigests  map[string]string        `json:"input_digests"`
	Inventory     inventory.Result         `json:"inventory"`
}

type Request struct {
	Repo             *gitrepo.Repo
	Baseline         string
	InputDigests     map[string]string
	IncludeUntracked bool
	IncludeIgnored   []string
}

func Capture(ctx context.Context, request Request) (Seal, error) {
	revisions, err := request.Repo.Resolve(ctx, request.Baseline)
	if err != nil {
		return Seal{}, err
	}
	items, err := inventory.Capture(ctx, request.Repo, revisions, inventory.Options{
		IncludeUntracked: request.IncludeUntracked,
		IncludeIgnored:   request.IncludeIgnored,
	})
	if err != nil {
		return Seal{}, err
	}
	index, err := request.Repo.Git(ctx, "write-tree")
	if err != nil {
		return Seal{}, fmt.Errorf("capture index tree: %w", err)
	}
	gitConfig, err := request.Repo.SemanticConfig(ctx)
	if err != nil {
		return Seal{}, err
	}
	afterRevisions, err := request.Repo.Resolve(ctx, request.Baseline)
	if err != nil {
		return Seal{}, err
	}
	afterItems, err := inventory.Capture(ctx, request.Repo, afterRevisions, inventory.Options{
		IncludeUntracked: request.IncludeUntracked,
		IncludeIgnored:   request.IncludeIgnored,
	})
	if err != nil {
		return Seal{}, err
	}
	if !sameRevisions(revisions, afterRevisions) || !sameInventory(items, afterItems) {
		return Seal{}, fmt.Errorf("repository changed while capturing snapshot; retry from a fresh state")
	}
	seal := Seal{
		SchemaVersion: "1.0", CapturedAt: time.Now().UTC(), Revisions: revisions,
		IndexTree: string(bytesTrimSpace(index)), GitConfig: gitConfig,
		InputDigests: cloneMap(request.InputDigests), Inventory: items,
	}
	seal.SnapshotID, err = contentID(seal)
	if err != nil {
		return Seal{}, err
	}
	return seal, nil
}

func sameRevisions(left, right gitrepo.RevisionEvidence) bool {
	return left.BaselineRef == right.BaselineRef &&
		left.BaselineCommit == right.BaselineCommit &&
		left.HeadCommit == right.HeadCommit &&
		left.MergeBase == right.MergeBase &&
		left.Freshness == right.Freshness &&
		left.RemoteFetchProof == right.RemoteFetchProof
}

func sameInventory(left, right inventory.Result) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}

func VerifyFresh(ctx context.Context, request Request, expected Seal) error {
	current, err := Capture(ctx, request)
	if err != nil {
		return err
	}
	if current.SnapshotID != expected.SnapshotID {
		return fmt.Errorf("snapshot stale: expected %s, current %s", expected.SnapshotID, current.SnapshotID)
	}
	return nil
}

func contentID(seal Seal) (string, error) {
	revisions := revisionCanonical{
		BaselineRef: seal.Revisions.BaselineRef, BaselineCommit: seal.Revisions.BaselineCommit,
		HeadCommit: seal.Revisions.HeadCommit, MergeBase: seal.Revisions.MergeBase,
		Freshness: seal.Revisions.Freshness, RemoteFetchProof: seal.Revisions.RemoteFetchProof,
	}
	encodedRevisions, err := json.Marshal(revisions)
	if err != nil {
		return "", err
	}
	encodedConfig, err := json.Marshal(seal.GitConfig)
	if err != nil {
		return "", err
	}
	encodedInputs, err := json.Marshal(seal.InputDigests)
	if err != nil {
		return "", err
	}
	encodedInventory, err := json.Marshal(seal.Inventory)
	if err != nil {
		return "", err
	}
	hasher := sha256.New()
	for _, field := range [][]byte{
		[]byte(seal.SchemaVersion), encodedRevisions, []byte(seal.IndexTree),
		encodedConfig, encodedInputs, encodedInventory,
	} {
		writeLengthPrefixed(hasher, field)
	}
	return "snap-" + hex.EncodeToString(hasher.Sum(nil)), nil
}

func writeLengthPrefixed(target hash.Hash, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = target.Write(length[:])
	_, _ = target.Write(value)
}

type revisionCanonical struct {
	BaselineRef      string `json:"baseline_ref"`
	BaselineCommit   string `json:"baseline_commit"`
	HeadCommit       string `json:"head_commit"`
	MergeBase        string `json:"merge_base"`
	Freshness        string `json:"freshness"`
	RemoteFetchProof bool   `json:"remote_fetch_proof"`
}

func cloneMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
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
