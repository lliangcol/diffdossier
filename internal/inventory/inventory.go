// Package inventory captures byte-accurate changed-path metadata from Git.
package inventory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/lliangcol/diffdossier/internal/gitrepo"
)

type Scope string

const (
	ScopeCommitted Scope = "merge_base_to_head"
	ScopeStaged    Scope = "staged"
	ScopeUnstaged  Scope = "unstaged"
	ScopeUntracked Scope = "untracked"
	ScopeIgnored   Scope = "ignored_explicit"
)

type Options struct {
	IncludeUntracked bool
	IncludeIgnored   []string
	MaxBlobBytes     int64
	MaxTotalBytes    int64
}

const DefaultMaxBlobBytes int64 = 64 * 1024 * 1024
const DefaultMaxTotalBytes int64 = 256 * 1024 * 1024

type PathIdentity struct {
	BytesBase64 string  `json:"path_bytes_base64"`
	UTF8        *string `json:"path_utf8,omitempty"`
}

func (path PathIdentity) Raw() ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(path.BytesBase64)
	if err != nil {
		return nil, fmt.Errorf("decode path identity: %w", err)
	}
	return decoded, nil
}

func (path PathIdentity) Display() string {
	if path.UTF8 != nil {
		return *path.UTF8
	}
	return "base64:" + path.BytesBase64
}

type Entry struct {
	Scope               Scope         `json:"scope"`
	Status              string        `json:"status"`
	Path                PathIdentity  `json:"path"`
	OldPath             *PathIdentity `json:"old_path,omitempty"`
	Kind                string        `json:"kind"`
	Mode                string        `json:"mode,omitempty"`
	Size                int64         `json:"size"`
	ContentHash         string        `json:"content_hash,omitempty"`
	Binary              bool          `json:"binary"`
	LinkTarget          *PathIdentity `json:"link_target,omitempty"`
	GitObject           string        `json:"git_object,omitempty"`
	LFSOID              string        `json:"lfs_oid,omitempty"`
	LFSSize             int64         `json:"lfs_size,omitempty"`
	SubmoduleDirty      bool          `json:"submodule_dirty,omitempty"`
	Content             []byte        `json:"-"`
	PreviousKind        string        `json:"previous_kind,omitempty"`
	PreviousMode        string        `json:"previous_mode,omitempty"`
	PreviousSize        int64         `json:"previous_size,omitempty"`
	PreviousContentHash string        `json:"previous_content_hash,omitempty"`
	PreviousGitObject   string        `json:"previous_git_object,omitempty"`
	PreviousContent     []byte        `json:"-"`
}

type Result struct {
	SchemaVersion string  `json:"schema_version"`
	Entries       []Entry `json:"entries"`
}

func Capture(ctx context.Context, repo *gitrepo.Repo, revisions gitrepo.RevisionEvidence, requested ...Options) (Result, error) {
	options := Options{IncludeUntracked: true}
	if len(requested) > 0 {
		options = requested[0]
	}
	if options.MaxBlobBytes < 1 {
		options.MaxBlobBytes = DefaultMaxBlobBytes
	}
	if options.MaxTotalBytes < 1 {
		options.MaxTotalBytes = DefaultMaxTotalBytes
	}
	entries := []Entry{}
	committed, err := diffEntries(ctx, repo, ScopeCommitted, revisions.MergeBase, "HEAD")
	if err != nil {
		return Result{}, err
	}
	entries = append(entries, committed...)
	staged, err := diffEntries(ctx, repo, ScopeStaged, "--cached")
	if err != nil {
		return Result{}, err
	}
	entries = append(entries, staged...)
	unstaged, err := diffEntries(ctx, repo, ScopeUnstaged)
	if err != nil {
		return Result{}, err
	}
	entries = append(entries, unstaged...)
	if options.IncludeUntracked {
		untracked, err := untrackedEntries(ctx, repo, ScopeUntracked, nil)
		if err != nil {
			return Result{}, err
		}
		entries = append(entries, untracked...)
	}
	if len(options.IncludeIgnored) > 0 {
		ignored, err := untrackedEntries(ctx, repo, ScopeIgnored, options.IncludeIgnored)
		if err != nil {
			return Result{}, err
		}
		entries = append(entries, ignored...)
	}
	submodules, err := dirtySubmoduleEntries(ctx, repo)
	if err != nil {
		return Result{}, err
	}
	entries = appendUnique(entries, submodules...)
	var capturedBytes int64
	for index := range entries {
		if err := enrich(ctx, repo, revisions, &entries[index]); err != nil {
			return Result{}, err
		}
		if entries[index].Size > options.MaxBlobBytes || entries[index].PreviousSize > options.MaxBlobBytes {
			return Result{}, fmt.Errorf("changed blob exceeds %d-byte safety budget", options.MaxBlobBytes)
		}
		capturedBytes += entries[index].Size + entries[index].PreviousSize
		if capturedBytes > options.MaxTotalBytes {
			return Result{}, fmt.Errorf("changed content exceeds %d-byte total safety budget", options.MaxTotalBytes)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Scope != entries[j].Scope {
			return entries[i].Scope < entries[j].Scope
		}
		return entries[i].Path.BytesBase64 < entries[j].Path.BytesBase64
	})
	return Result{SchemaVersion: "1.0", Entries: entries}, nil
}

func appendUnique(entries []Entry, candidates ...Entry) []Entry {
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		seen[string(entry.Scope)+"\x00"+entry.Path.BytesBase64] = true
	}
	for _, candidate := range candidates {
		key := string(candidate.Scope) + "\x00" + candidate.Path.BytesBase64
		if !seen[key] {
			entries = append(entries, candidate)
			seen[key] = true
		}
	}
	return entries
}

func dirtySubmoduleEntries(ctx context.Context, repo *gitrepo.Repo) ([]Entry, error) {
	output, err := repo.Git(ctx, "ls-files", "--stage", "-z")
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, record := range splitNUL(output) {
		metadata, rawPath, ok := bytes.Cut(record, []byte{'\t'})
		if !ok {
			continue
		}
		parts := bytes.Fields(metadata)
		if len(parts) < 3 || string(parts[0]) != "160000" {
			continue
		}
		if err := validateRelative(rawPath); err != nil {
			return nil, err
		}
		_, dirty, stateErr := submoduleWorktreeState(ctx, repo, rawPath, string(parts[1]))
		if stateErr != nil {
			continue
		}
		if dirty {
			entries = append(entries, Entry{Scope: ScopeUnstaged, Status: "M", Path: identity(rawPath)})
		}
	}
	return entries, nil
}

func diffEntries(ctx context.Context, repo *gitrepo.Repo, scope Scope, extra ...string) ([]Entry, error) {
	args := []string{"diff", "--name-status", "-z", "--find-renames", "--find-copies"}
	args = append(args, extra...)
	output, err := repo.Git(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("inventory %s: %w", scope, err)
	}
	fields := splitNUL(output)
	entries := []Entry{}
	for index := 0; index < len(fields); {
		status := string(fields[index])
		index++
		if status == "" || index >= len(fields) {
			return nil, fmt.Errorf("malformed NUL Git inventory for %s", scope)
		}
		entry := Entry{Scope: scope, Status: status}
		if status[0] == 'R' || status[0] == 'C' {
			if index+1 >= len(fields) {
				return nil, fmt.Errorf("malformed rename/copy inventory for %s", scope)
			}
			oldPath := identity(fields[index])
			entry.OldPath = &oldPath
			entry.Path = identity(fields[index+1])
			index += 2
		} else {
			entry.Path = identity(fields[index])
			index++
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func untrackedEntries(ctx context.Context, repo *gitrepo.Repo, scope Scope, paths []string) ([]Entry, error) {
	args := []string{"ls-files", "--others", "--exclude-standard", "-z"}
	if scope == ScopeIgnored {
		args = []string{"ls-files", "--others", "--ignored", "--exclude-standard", "-z", "--"}
		for _, path := range paths {
			if err := validateRelative([]byte(path)); err != nil {
				return nil, err
			}
			args = append(args, path)
		}
	}
	output, err := repo.Git(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("inventory untracked: %w", err)
	}
	fields := splitNUL(output)
	entries := make([]Entry, 0, len(fields))
	for _, field := range fields {
		entries = append(entries, Entry{Scope: scope, Status: "?", Path: identity(field)})
	}
	return entries, nil
}

func enrich(ctx context.Context, repo *gitrepo.Repo, revisions gitrepo.RevisionEvidence, entry *Entry) error {
	path := decode(entry.Path)
	if err := validateRelative(path); err != nil {
		return err
	}
	if !strings.HasPrefix(entry.Status, "D") {
		source, mode, object, err := currentSource(ctx, repo, *entry, path)
		if err != nil {
			return fmt.Errorf("read current %s path %q: %w", entry.Scope, display(path), err)
		}
		describeCurrent(entry, source, mode, object)
	} else {
		entry.Kind = "missing"
	}
	if !strings.HasPrefix(entry.Status, "A") && entry.Scope != ScopeUntracked && entry.Scope != ScopeIgnored {
		previousPath := path
		if entry.OldPath != nil {
			previousPath = decode(*entry.OldPath)
		}
		previous, mode, object, err := previousSource(ctx, repo, revisions, *entry, previousPath)
		if err != nil {
			return fmt.Errorf("read previous %s path %q: %w", entry.Scope, display(previousPath), err)
		}
		describePrevious(entry, previous, mode, object)
	}
	return nil
}

func describeCurrent(entry *Entry, source []byte, mode, object string) {
	entry.Mode = mode
	entry.GitObject = object
	if source == nil {
		entry.Kind = "missing"
		return
	}
	entry.Size = int64(len(source))
	entry.Content = append([]byte(nil), source...)
	digest := sha256.Sum256(source)
	entry.ContentHash = "sha256:" + hex.EncodeToString(digest[:])
	entry.Binary = bytes.IndexByte(source[:min(len(source), 8192)], 0) >= 0
	switch mode {
	case "120000":
		entry.Kind = "symlink"
		target := identity(source)
		entry.LinkTarget = &target
	case "160000":
		entry.Kind = "submodule"
		entry.SubmoduleDirty = bytes.Contains(source, []byte{0})
	default:
		if oid, size, ok := parseLFSPointer(source); ok {
			entry.Kind = "lfs_pointer"
			entry.LFSOID = oid
			entry.LFSSize = size
		} else if entry.Binary {
			entry.Kind = "binary"
		} else {
			entry.Kind = "regular"
		}
	}
}

func describePrevious(entry *Entry, source []byte, mode, object string) {
	if source == nil {
		return
	}
	entry.PreviousMode = mode
	entry.PreviousGitObject = object
	entry.PreviousSize = int64(len(source))
	entry.PreviousContent = append([]byte(nil), source...)
	digest := sha256.Sum256(source)
	entry.PreviousContentHash = "sha256:" + hex.EncodeToString(digest[:])
	switch mode {
	case "120000":
		entry.PreviousKind = "symlink"
	case "160000":
		entry.PreviousKind = "submodule"
	default:
		if _, _, ok := parseLFSPointer(source); ok {
			entry.PreviousKind = "lfs_pointer"
		} else if bytes.IndexByte(source[:min(len(source), 8192)], 0) >= 0 {
			entry.PreviousKind = "binary"
		} else {
			entry.PreviousKind = "regular"
		}
	}
}

func currentSource(ctx context.Context, repo *gitrepo.Repo, entry Entry, path []byte) ([]byte, string, string, error) {
	switch entry.Scope {
	case ScopeCommitted:
		return gitBlob(ctx, repo, "HEAD", path)
	case ScopeStaged:
		return gitBlob(ctx, repo, "", path)
	case ScopeUnstaged:
		if mode, object, _ := gitMetadata(ctx, repo, "", path); mode == "160000" {
			state, _, statusErr := submoduleWorktreeState(ctx, repo, path, object)
			if statusErr != nil {
				return []byte(object), mode, object, nil
			}
			return state, mode, object, nil
		}
		return worktreeBlob(repo.Root, path)
	case ScopeUntracked, ScopeIgnored:
		return worktreeBlob(repo.Root, path)
	default:
		return nil, "", "", errors.New("unknown inventory scope")
	}
}

func previousSource(ctx context.Context, repo *gitrepo.Repo, revisions gitrepo.RevisionEvidence, entry Entry, path []byte) ([]byte, string, string, error) {
	switch entry.Scope {
	case ScopeCommitted:
		return gitBlob(ctx, repo, revisions.MergeBase, path)
	case ScopeStaged:
		return gitBlob(ctx, repo, "HEAD", path)
	case ScopeUnstaged:
		return gitBlob(ctx, repo, "", path)
	default:
		return nil, "", "", nil
	}
}

func submoduleWorktreeState(ctx context.Context, repo *gitrepo.Repo, path []byte, indexObject string) ([]byte, bool, error) {
	full := filepath.Join(repo.Root, string(path))
	info, err := os.Lstat(full)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return []byte(indexObject), false, err
	}
	head, err := repo.Git(ctx, "-C", string(path), "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return []byte(indexObject), false, err
	}
	status, err := repo.Git(ctx, "-C", string(path), "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return []byte(indexObject), false, err
	}
	trimmedHead := bytes.TrimSpace(head)
	dirty := string(trimmedHead) != indexObject || len(status) > 0
	state := append([]byte(indexObject), 0)
	state = append(state, trimmedHead...)
	state = append(state, 0)
	state = append(state, status...)
	return state, dirty, nil
}

func gitBlob(ctx context.Context, repo *gitrepo.Repo, revision string, path []byte) ([]byte, string, string, error) {
	spec := ":" + string(path)
	if revision != "" && revision != "HEAD" {
		spec = revision + ":" + string(path)
	} else if revision == "HEAD" {
		spec = "HEAD:" + string(path)
	}
	mode, object, err := gitMetadata(ctx, repo, revision, path)
	if err != nil {
		return nil, "", "", err
	}
	if mode == "160000" {
		return []byte(object), mode, object, nil
	}
	content, err := repo.Git(ctx, "show", spec)
	if err != nil {
		return nil, "", "", err
	}
	return content, mode, object, nil
}

func gitMetadata(ctx context.Context, repo *gitrepo.Repo, revision string, path []byte) (string, string, error) {
	modeArgs := []string{"ls-files", "--stage", "-z", "--", string(path)}
	if revision != "" {
		modeArgs = []string{"ls-tree", "-z", revision, "--", string(path)}
	}
	metadata, err := repo.Git(ctx, modeArgs...)
	if err != nil {
		return "", "", err
	}
	mode, object := parseObjectMetadata(metadata)
	if mode == "" {
		return "", "", errors.New("Git object metadata missing")
	}
	return mode, object, nil
}

func worktreeBlob(root string, path []byte) ([]byte, string, string, error) {
	full := filepath.Join(root, string(path))
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", "", nil
		}
		return nil, "", "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(full)
		return []byte(target), "120000", "", err
	}
	if !info.Mode().IsRegular() {
		return nil, "", "", fmt.Errorf("unsupported worktree file type %s", info.Mode().Type())
	}
	if info.Size() > DefaultMaxBlobBytes {
		return nil, "", "", fmt.Errorf("worktree blob exceeds %d-byte safety budget", DefaultMaxBlobBytes)
	}
	content, err := os.ReadFile(full)
	mode := "100644"
	if info.Mode().Perm()&0o111 != 0 {
		mode = "100755"
	}
	return content, mode, "", err
}

func parseObjectMetadata(metadata []byte) (string, string) {
	line := metadata
	if index := bytes.IndexByte(line, 0); index >= 0 {
		line = line[:index]
	}
	parts := bytes.Fields(line)
	if len(parts) < 3 {
		return "", ""
	}
	for _, candidate := range parts[1:3] {
		if (len(candidate) == 40 || len(candidate) == 64) && isHex(candidate) {
			return string(parts[0]), string(candidate)
		}
	}
	return "", ""
}

func isHex(value []byte) bool {
	for _, character := range value {
		if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f') || (character >= 'A' && character <= 'F')) {
			return false
		}
	}
	return true
}

func parseLFSPointer(content []byte) (string, int64, bool) {
	if len(content) > 1024 || !bytes.HasPrefix(content, []byte("version https://git-lfs.github.com/spec/v1\n")) {
		return "", 0, false
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[1], "oid sha256:") || !strings.HasPrefix(lines[2], "size ") {
		return "", 0, false
	}
	var size int64
	if _, err := fmt.Sscanf(lines[2], "size %d", &size); err != nil || size < 0 {
		return "", 0, false
	}
	oid := strings.TrimPrefix(lines[1], "oid ")
	if len(oid) != len("sha256:")+64 {
		return "", 0, false
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(oid, "sha256:")); err != nil {
		return "", 0, false
	}
	return oid, size, true
}

func identity(raw []byte) PathIdentity {
	copyOfRaw := append([]byte(nil), raw...)
	result := PathIdentity{BytesBase64: base64.StdEncoding.EncodeToString(copyOfRaw)}
	if utf8.Valid(copyOfRaw) {
		value := string(copyOfRaw)
		result.UTF8 = &value
	}
	return result
}

func decode(path PathIdentity) []byte {
	decoded, _ := base64.StdEncoding.DecodeString(path.BytesBase64)
	return decoded
}

func validateRelative(path []byte) error {
	if len(path) == 0 || bytes.IndexByte(path, 0) >= 0 {
		return errors.New("empty or NUL-containing path")
	}
	value := string(path)
	if filepath.IsAbs(value) || filepath.Clean(value) == ".." || strings.HasPrefix(filepath.Clean(value), ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes repository: %q", display(path))
	}
	return nil
}

func display(path []byte) string {
	if utf8.Valid(path) {
		return string(path)
	}
	return fmt.Sprintf("base64:%s", base64.StdEncoding.EncodeToString(path))
}

func splitNUL(output []byte) [][]byte {
	if len(output) == 0 {
		return nil
	}
	parts := bytes.Split(output, []byte{0})
	if len(parts[len(parts)-1]) == 0 {
		parts = parts[:len(parts)-1]
	}
	return parts
}
