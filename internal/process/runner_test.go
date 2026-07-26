package process

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunArgvAndBoundedOutput(t *testing.T) {
	spec := processHelperSpec(t, "output")
	spec.Env = append(spec.Env, "HELPER_OUTPUT=ok")
	output, err := Run(context.Background(), spec)
	if err != nil || string(output.Stdout) != "ok" {
		t.Fatalf("output=%q err=%v", output.Stdout, err)
	}
	spec = processHelperSpec(t, "output")
	spec.Env = append(spec.Env, "HELPER_OUTPUT=too-long")
	spec.MaxStdout = 2
	if _, err := Run(context.Background(), spec); err == nil {
		t.Fatal("oversize stdout must fail")
	}
}

func TestRunTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := Run(ctx, processHelperSpec(t, "sleep"))
	if err == nil {
		t.Fatal("timeout must fail")
	}
}

func TestRunDoesNotInheritEnvironment(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "marker")
	if err := os.Setenv("DIFFDOSSIER_SECRET_TEST", marker); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("DIFFDOSSIER_SECRET_TEST") })
	spec := processHelperSpec(t, "environment")
	spec.Env = append(spec.Env, "SAFE=value")
	output, err := Run(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	var environment struct {
		Safe   string `json:"safe"`
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal(output.Stdout, &environment); err != nil {
		t.Fatal(err)
	}
	if environment.Safe != "value" || environment.Secret != "" {
		t.Fatalf("unexpected inherited environment: %+v", environment)
	}
}

func TestProcessHelper(t *testing.T) {
	if os.Getenv("GO_WANT_PROCESS_HELPER") != "1" {
		return
	}
	switch os.Getenv("HELPER_MODE") {
	case "output":
		_, _ = os.Stdout.WriteString(os.Getenv("HELPER_OUTPUT"))
	case "sleep":
		time.Sleep(10 * time.Second)
	case "environment":
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{
			"safe": os.Getenv("SAFE"), "secret": os.Getenv("DIFFDOSSIER_SECRET_TEST"),
		})
	default:
		os.Exit(2)
	}
	os.Exit(0)
}

func processHelperSpec(t *testing.T, mode string) Spec {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return Spec{
		Executable: executable,
		Args:       []string{"-test.run=TestProcessHelper"},
		Dir:        t.TempDir(),
		Env:        []string{"GO_WANT_PROCESS_HELPER=1", "HELPER_MODE=" + mode},
		MaxStdout:  1024,
		MaxStderr:  1024,
	}
}
