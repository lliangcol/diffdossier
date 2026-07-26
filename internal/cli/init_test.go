package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/lliangcol/diffdossier/internal/config"
)

func TestInitCreatesMinimalCommandFreeConfig(t *testing.T) {
	repo := initializedRepo(t)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"init", "--repo", repo, "--baseline", "HEAD", "--json"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var envelope struct {
		Status string `json:"status"`
		Data   struct {
			BaselineRef     string `json:"baseline_ref"`
			CommandsEnabled bool   `json:"commands_enabled"`
			ConfigPath      string `json:"config_path"`
			Created         bool   `json:"created"`
			Freshness       string `json:"freshness"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Status != "ok" || envelope.Data.BaselineRef != "HEAD" || !envelope.Data.Created || envelope.Data.CommandsEnabled || envelope.Data.Freshness != "local_only" {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
	resolvedRepo, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(resolvedRepo, "diffdossier.toml")
	if envelope.Data.ConfigPath != path {
		t.Fatalf("config_path=%q want=%q", envelope.Data.ConfigPath, path)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Baseline != "HEAD" || len(cfg.Gates) != 0 || cfg.Review.DefaultProvider != "manual" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o644 {
			t.Fatalf("mode=%o want=644", info.Mode().Perm())
		}
	}
	if matches, err := filepath.Glob(filepath.Join(repo, ".diffdossier-init-*")); err != nil || len(matches) != 0 {
		t.Fatalf("staging files remain: %v err=%v", matches, err)
	}
}

func TestInitRefusesOverwriteAndSymlink(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, string) []byte
	}{
		{name: "regular", prepare: func(t *testing.T, path string) []byte {
			content := []byte("do not replace\n")
			if err := os.WriteFile(path, content, 0o600); err != nil {
				t.Fatal(err)
			}
			return content
		}},
		{name: "symlink", prepare: func(t *testing.T, path string) []byte {
			if runtime.GOOS == "windows" {
				t.Skip("symlink setup requires platform privileges")
			}
			target := filepath.Join(filepath.Dir(path), "outside.toml")
			content := []byte("outside\n")
			if err := os.WriteFile(target, content, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, path); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			return content
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := initializedRepo(t)
			path := filepath.Join(repo, "diffdossier.toml")
			original := test.prepare(t, path)
			var stdout, stderr bytes.Buffer
			code := Run([]string{"init", "--repo", repo, "--baseline", "HEAD", "--json"}, &stdout, &stderr)
			if code != ExitUsage || !strings.Contains(stdout.String(), "DD_INIT_EXISTS") {
				t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
			}
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(content, original) {
				t.Fatalf("existing target changed: %q", content)
			}
		})
	}
}

func TestInitRejectsMissingOrUnresolvedBaselineWithoutWrite(t *testing.T) {
	repo := initializedRepo(t)
	for _, args := range [][]string{
		{"init", "--repo", repo, "--json"},
		{"init", "--repo", repo, "--baseline", "refs/heads/missing", "--json"},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code == ExitOK {
			t.Fatalf("args=%v unexpectedly succeeded: %s", args, stdout.String())
		}
		if _, err := os.Lstat(filepath.Join(repo, "diffdossier.toml")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("args=%v wrote config: %v", args, err)
		}
	}
}

func TestInitRejectsNonRepository(t *testing.T) {
	directory := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"init", "--repo", directory, "--baseline", "HEAD", "--json"}, &stdout, &stderr)
	if code != ExitEvidence || !strings.Contains(stdout.String(), "DD_GIT_REPOSITORY") {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Lstat(filepath.Join(directory, "diffdossier.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("non-repository init wrote config: %v", err)
	}
}

func TestConcurrentInitPublishesExactlyOnce(t *testing.T) {
	repo := initializedRepo(t)
	const workers = 8
	codes := make(chan int, workers)
	var wait sync.WaitGroup
	for index := 0; index < workers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			var stdout, stderr bytes.Buffer
			codes <- Run([]string{"init", "--repo", repo, "--baseline", "HEAD", "--json"}, &stdout, &stderr)
		}()
	}
	wait.Wait()
	close(codes)
	successes := 0
	for code := range codes {
		if code == ExitOK {
			successes++
		} else if code != ExitUsage {
			t.Fatalf("unexpected concurrent init exit code %d", code)
		}
	}
	if successes != 1 {
		t.Fatalf("successes=%d want=1", successes)
	}
	if _, err := config.Load(filepath.Join(repo, "diffdossier.toml")); err != nil {
		t.Fatalf("published config is invalid: %v", err)
	}
	if matches, err := filepath.Glob(filepath.Join(repo, ".diffdossier-init-*")); err != nil || len(matches) != 0 {
		t.Fatalf("staging files remain: %v err=%v", matches, err)
	}
}

func initializedRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "config", "user.email", "fixture@example.invalid")
	runGit(t, repo, "config", "user.name", "Fixture")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-qm", "initial")
	return repo
}
