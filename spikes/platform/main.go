package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const childMode = "--compatibility-spike-child"

func main() {
	if len(os.Args) == 2 && os.Args[1] == childMode {
		time.Sleep(30 * time.Second)
		return
	}

	checks := []struct {
		name string
		fn   func() error
	}{
		{name: "git-nul-paths", fn: checkGitNULPaths},
		{name: "private-state", fn: checkPrivateState},
		{name: "atomic-replace", fn: checkAtomicReplace},
		{name: "exclusive-lock", fn: checkExclusiveLock},
		{name: "direct-child-cancel", fn: checkDirectChildCancellation},
	}

	failed := false
	for _, check := range checks {
		if err := check.fn(); err != nil {
			failed = true
			fmt.Printf("FAIL %s: %v\n", check.name, err)
			continue
		}
		fmt.Printf("PASS %s\n", check.name)
	}

	fmt.Printf("INFO platform=%s/%s go=%s\n", runtime.GOOS, runtime.GOARCH, runtime.Version())
	if failed {
		os.Exit(1)
	}
}

func checkGitNULPaths() error {
	repo, err := os.MkdirTemp("", "diffdossier-git-paths-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(repo)

	if output, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %w: %s", err, bytes.TrimSpace(output))
	}

	names := compatibilityPathNames(runtime.GOOS)
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(repo, name), []byte("fixture\n"), 0o600); err != nil {
			return fmt.Errorf("write %q: %w", name, err)
		}
	}

	cmd := exec.Command("git", "status", "--porcelain=v1", "-z", "--untracked-files=all")
	cmd.Dir = repo
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}

	actual, err := parsePorcelainV1Z(output)
	if err != nil {
		return err
	}
	sort.Strings(names)
	sort.Strings(actual)
	if strings.Join(names, "\x00") != strings.Join(actual, "\x00") {
		return fmt.Errorf("path mismatch: got %q want %q", actual, names)
	}
	return nil
}

func compatibilityPathNames(goos string) []string {
	names := []string{"plain.txt", "space name.txt", "中文.txt", "emoji-😀.txt"}
	if goos != "windows" {
		names = append(names, "tab\tname.txt", "line\nname.txt")
	}
	return names
}

func parsePorcelainV1Z(data []byte) ([]string, error) {
	if len(data) == 0 || data[len(data)-1] != 0 {
		return nil, errors.New("git status output is not NUL terminated")
	}

	records := bytes.Split(data[:len(data)-1], []byte{0})
	paths := make([]string, 0, len(records))
	for _, record := range records {
		if len(record) < 4 || record[2] != ' ' {
			return nil, fmt.Errorf("invalid porcelain record %q", record)
		}
		paths = append(paths, string(record[3:]))
	}
	return paths, nil
}

func checkPrivateState() error {
	root, err := os.MkdirTemp("", "diffdossier-private-state-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)

	state := filepath.Join(root, "state")
	if err := os.Mkdir(state, 0o700); err != nil {
		return err
	}
	file := filepath.Join(state, "checkpoint.json")
	if err := os.WriteFile(file, []byte("{}\n"), 0o600); err != nil {
		return err
	}

	if runtime.GOOS != "windows" {
		dirInfo, err := os.Stat(state)
		if err != nil {
			return err
		}
		fileInfo, err := os.Stat(file)
		if err != nil {
			return err
		}
		if got := dirInfo.Mode().Perm(); got != 0o700 {
			return fmt.Errorf("directory permissions %04o, want 0700", got)
		}
		if got := fileInfo.Mode().Perm(); got != 0o600 {
			return fmt.Errorf("file permissions %04o, want 0600", got)
		}
	}
	return nil
}

func checkAtomicReplace() error {
	root, err := os.MkdirTemp("", "diffdossier-atomic-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)

	target := filepath.Join(root, "state.json")
	temporary := filepath.Join(root, "state.json.tmp")
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(temporary, []byte("new"), 0o600); err != nil {
		return err
	}
	if err := os.Rename(temporary, target); err != nil {
		return err
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return err
	}
	if string(content) != "new" {
		return fmt.Errorf("replacement content %q, want %q", content, "new")
	}
	return nil
}

func checkExclusiveLock() error {
	root, err := os.MkdirTemp("", "diffdossier-lock-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)

	path := filepath.Join(root, "run.lock")
	first, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := first.Close(); err != nil {
		return err
	}
	second, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err == nil {
		second.Close()
		return errors.New("second exclusive lock acquisition unexpectedly succeeded")
	}
	if !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("second lock error: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	third, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("lock reacquisition: %w", err)
	}
	return third.Close()
}

func checkDirectChildCancellation() error {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], childMode)
	err := cmd.Run()
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("context error %v, want deadline exceeded", ctx.Err())
	}
	if err == nil {
		return errors.New("child process exited successfully after cancellation")
	}
	return nil
}
