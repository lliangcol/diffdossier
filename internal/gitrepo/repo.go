// Package gitrepo provides shell-free, local-only Git repository inspection.
package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const MaxGitOutputBytes = 64 * 1024 * 1024

type Repo struct {
	Root string
}

type RevisionEvidence struct {
	BaselineRef      string    `json:"baseline_ref"`
	BaselineCommit   string    `json:"baseline_commit"`
	HeadCommit       string    `json:"head_commit"`
	MergeBase        string    `json:"merge_base"`
	ResolvedAt       time.Time `json:"resolved_at"`
	Freshness        string    `json:"freshness"`
	RemoteFetchProof bool      `json:"remote_fetch_proof"`
}

func Open(ctx context.Context, path string) (*Repo, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	output, err := runAt(ctx, abs, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("discover Git repository: %w", err)
	}
	root := strings.TrimSpace(string(output))
	if root == "" {
		return nil, errors.New("Git returned an empty repository root")
	}
	return &Repo{Root: filepath.Clean(root)}, nil
}

func (repo *Repo) Git(ctx context.Context, args ...string) ([]byte, error) {
	return runAt(ctx, repo.Root, args...)
}

func (repo *Repo) Resolve(ctx context.Context, baseline string) (RevisionEvidence, error) {
	if strings.TrimSpace(baseline) == "" {
		return RevisionEvidence{}, errors.New("baseline is required; refusing to guess main or master")
	}
	base, err := repo.Git(ctx, "rev-parse", "--verify", baseline+"^{commit}")
	if err != nil {
		return RevisionEvidence{}, fmt.Errorf("resolve baseline %q: %w", baseline, err)
	}
	head, err := repo.Git(ctx, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return RevisionEvidence{}, fmt.Errorf("resolve HEAD: %w", err)
	}
	mergeBase, err := repo.Git(ctx, "merge-base", strings.TrimSpace(string(base)), strings.TrimSpace(string(head)))
	if err != nil {
		return RevisionEvidence{}, fmt.Errorf("resolve merge-base: %w", err)
	}
	return RevisionEvidence{
		BaselineRef: baseline, BaselineCommit: strings.TrimSpace(string(base)),
		HeadCommit: strings.TrimSpace(string(head)), MergeBase: strings.TrimSpace(string(mergeBase)),
		ResolvedAt: time.Now().UTC(), Freshness: "local_only", RemoteFetchProof: false,
	}, nil
}

func (repo *Repo) SemanticConfig(ctx context.Context) (map[string]string, error) {
	keys := []string{"core.autocrlf", "core.eol", "core.ignorecase", "core.symlinks", "core.filemode"}
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		output, err := repo.Git(ctx, "config", "--get", key)
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
				result[key] = "unset"
				continue
			}
			return nil, fmt.Errorf("read git config %s: %w", key, err)
		}
		result[key] = strings.TrimSpace(string(output))
	}
	return result, nil
}

func runAt(ctx context.Context, dir string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = dir
	command.Env = []string{"PATH=" + lookupPath(), "LC_ALL=C", "LANG=C"}
	stdout := &boundedBuffer{limit: MaxGitOutputBytes}
	stderr := &boundedBuffer{limit: 1024 * 1024}
	command.Stdout = stdout
	command.Stderr = stderr
	err := command.Run()
	if err != nil {
		message := strings.TrimSpace(string(stderr.Bytes()))
		if message != "" {
			return nil, fmt.Errorf("%s: %w", message, err)
		}
		return nil, err
	}
	if stdout.exceeded {
		return nil, fmt.Errorf("Git output exceeded %d-byte safety budget", MaxGitOutputBytes)
	}
	return stdout.Bytes(), nil
}

type boundedBuffer struct {
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func (writer *boundedBuffer) Write(value []byte) (int, error) {
	original := len(value)
	remaining := writer.limit - writer.buffer.Len()
	if remaining > 0 {
		take := len(value)
		if take > remaining {
			take = remaining
		}
		_, _ = writer.buffer.Write(value[:take])
	}
	if original > remaining {
		writer.exceeded = true
	}
	return original, nil
}
func (writer *boundedBuffer) Bytes() []byte { return writer.buffer.Bytes() }

func lookupPath() string {
	path, err := exec.LookPath("git")
	if err != nil {
		return ""
	}
	return filepath.Dir(path)
}
