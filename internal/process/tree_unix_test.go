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
	pid := 0
	for time.Now().Before(markerDeadline) {
		content, readErr := os.ReadFile(marker)
		if readErr == nil {
			parsed, parseErr := strconv.Atoi(strings.TrimSpace(string(content)))
			if parseErr == nil && parsed > 0 {
				pid = parsed
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if pid == 0 {
		cancel()
		<-done
		t.Fatal("grandchild PID marker was not fully written before deadline")
	}
	cancel()
	err := <-done
	if !errors.Is(err, context.DeadlineExceeded) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("run error=%v", err)
		}
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
