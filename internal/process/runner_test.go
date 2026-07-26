package process

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestRunArgvAndBoundedOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX printf")
	}
	output, err := Run(context.Background(), Spec{
		Executable: "/usr/bin/printf", Args: []string{"%s", "ok"}, Dir: t.TempDir(),
		Env: []string{}, MaxStdout: 16, MaxStderr: 16,
	})
	if err != nil || string(output.Stdout) != "ok" {
		t.Fatalf("output=%q err=%v", output.Stdout, err)
	}
	if _, err := Run(context.Background(), Spec{
		Executable: "/usr/bin/printf", Args: []string{"%s", "too-long"}, Dir: t.TempDir(),
		Env: []string{}, MaxStdout: 2, MaxStderr: 16,
	}); err == nil {
		t.Fatal("oversize stdout must fail")
	}
}

func TestRunTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX sleep")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := Run(ctx, Spec{
		Executable: "/bin/sleep", Args: []string{"10"}, Dir: t.TempDir(),
		Env: []string{}, MaxStdout: 16, MaxStderr: 16,
	})
	if err == nil {
		t.Fatal("timeout must fail")
	}
}

func TestRunDoesNotInheritEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX env")
	}
	marker := filepath.Join(t.TempDir(), "marker")
	if err := os.Setenv("DIFFDOSSIER_SECRET_TEST", marker); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("DIFFDOSSIER_SECRET_TEST") })
	output, err := Run(context.Background(), Spec{
		Executable: "/usr/bin/env", Dir: t.TempDir(), Env: []string{"SAFE=value"},
		MaxStdout: 1024, MaxStderr: 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(output.Stdout) != "SAFE=value\n" {
		t.Fatalf("unexpected inherited environment: %q", output.Stdout)
	}
}
