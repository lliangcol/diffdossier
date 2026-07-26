//go:build !windows

package process

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCancellationKillsGrandchildProcessGroup(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "child.pid")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := Run(ctx, Spec{Executable: "/bin/sh", Args: []string{"-c", "/bin/sleep 30 & child=$!; echo $child > child.pid; wait"}, Dir: dir, Env: []string{}, MaxStdout: 1024, MaxStderr: 1024})
		done <- err
	}()
	markerDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(markerDeadline) {
		if _, statErr := os.Stat(marker); statErr == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	err := <-done
	if !errors.Is(err, context.DeadlineExceeded) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("run error=%v", err)
		}
	}
	content, readErr := os.ReadFile(marker)
	if readErr != nil {
		t.Fatal(readErr)
	}
	pid, parseErr := strconv.Atoi(strings.TrimSpace(string(content)))
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		killErr := syscall.Kill(pid, 0)
		if errors.Is(killErr, syscall.ESRCH) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("grandchild pid %d remained alive after process-group cancellation", pid)
}
