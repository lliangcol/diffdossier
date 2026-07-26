//go:build windows

package process

import (
	"context"
	"os/exec"
)

// Windows Job Object containment is exposed as an unverified capability until
// the native Tier 1 implementation and tests land in the platform phase.
func configureTree(*exec.Cmd) {}

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
		_ = command.Process.Kill()
	}
}
