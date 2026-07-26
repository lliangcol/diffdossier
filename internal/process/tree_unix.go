//go:build !windows

package process

import (
	"context"
	"os/exec"
	"syscall"
)

func configureTree(command *exec.Cmd) error {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return nil
}

func afterStart(*exec.Cmd) error { return nil }
func finalizeTree(*exec.Cmd)     {}

func watchCancellation(ctx context.Context, command *exec.Cmd, done <-chan struct{}) {
	go func() {
		select {
		case <-ctx.Done():
			terminateTree(command)
		case <-done:
		}
	}()
}

func terminateTree(command *exec.Cmd) {
	if command.Process != nil {
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	}
}
