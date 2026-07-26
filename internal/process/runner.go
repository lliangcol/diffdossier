// Package process runs argv-only child processes with bounded I/O.
package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

type Spec struct {
	Executable string
	Args       []string
	Dir        string
	Env        []string
	Stdin      []byte
	MaxStdout  int64
	MaxStderr  int64
}

type Output struct {
	Stdout []byte
	Stderr []byte
}

func Run(ctx context.Context, spec Spec) (Output, error) {
	if spec.MaxStdout < 1 || spec.MaxStderr < 1 {
		return Output{}, errors.New("process output limits must be positive")
	}
	command := exec.CommandContext(ctx, spec.Executable, spec.Args...)
	command.Dir = spec.Dir
	command.Env = append([]string(nil), spec.Env...)
	command.Stdin = bytes.NewReader(spec.Stdin)
	if err := configureTree(command); err != nil {
		return Output{}, err
	}
	defer finalizeTree(command)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return Output{}, err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return Output{}, err
	}
	if err := command.Start(); err != nil {
		return Output{}, err
	}
	if err := afterStart(command); err != nil {
		terminateTree(command)
		_ = command.Wait()
		return Output{}, err
	}
	done := make(chan struct{})
	defer close(done)
	watchCancellation(ctx, command, done)

	var stdoutBytes, stderrBytes []byte
	var stdoutErr, stderrErr error
	var wait sync.WaitGroup
	limitError := make(chan error, 2)
	wait.Add(2)
	go func() {
		defer wait.Done()
		stdoutBytes, stdoutErr = readBounded(stdout, spec.MaxStdout)
		if stdoutErr != nil {
			limitError <- stdoutErr
		}
	}()
	go func() {
		defer wait.Done()
		stderrBytes, stderrErr = readBounded(stderr, spec.MaxStderr)
		if stderrErr != nil {
			limitError <- stderrErr
		}
	}()
	processDone := make(chan error, 1)
	go func() { processDone <- command.Wait() }()
	var waitErr error
	select {
	case outputErr := <-limitError:
		terminateTree(command)
		waitErr = <-processDone
		if outputErr != nil {
			waitErr = errors.Join(waitErr, outputErr)
		}
	case waitErr = <-processDone:
	}
	wait.Wait()
	if ctx.Err() != nil {
		return Output{Stdout: stdoutBytes, Stderr: stderrBytes}, ctx.Err()
	}
	if stdoutErr != nil || stderrErr != nil {
		return Output{Stdout: stdoutBytes, Stderr: stderrBytes}, errors.Join(stdoutErr, stderrErr)
	}
	if waitErr != nil {
		return Output{Stdout: stdoutBytes, Stderr: stderrBytes}, fmt.Errorf("process failed: %w", waitErr)
	}
	return Output{Stdout: stdoutBytes, Stderr: stderrBytes}, nil
}

func readBounded(reader io.Reader, limit int64) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return content[:limit], errors.New("process output exceeded configured limit")
	}
	return content, nil
}
