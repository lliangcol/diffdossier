package providers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lliangcol/diffdossier/internal/packets"
	"github.com/lliangcol/diffdossier/internal/planner"
	"github.com/lliangcol/diffdossier/internal/policy"
	"github.com/lliangcol/diffdossier/internal/results"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

func TestCommandProviderRejectsUntrustedWithoutExecution(t *testing.T) {
	config, packet, marker := commandFixture(t, "valid")
	command, err := NewCommand(config, packet, policy.TrustBinding{}, policy.EgressGrant{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := command.Review(context.Background(), packet); !errors.Is(err, ErrCommandUnauthorized) {
		t.Fatalf("review error=%v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("untrusted command executed; marker err=%v", err)
	}
}

func TestCommandProviderHandshakeAndReview(t *testing.T) {
	config, packet, marker := commandFixture(t, "valid")
	command := authorizedCommand(t, config, packet)
	handshake, err := command.Handshake(context.Background())
	if err != nil || handshake.Provider != "fixture-command" {
		t.Fatalf("handshake=%+v err=%v", handshake, err)
	}
	result, err := command.Review(context.Background(), packet)
	if err != nil {
		t.Fatal(err)
	}
	if result.TaskID != packet.TaskID || result.TaskInputHash != packet.TaskInputHash {
		t.Fatalf("result is not packet-bound: %+v", result)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("authorized helper did not execute: %v", err)
	}
}

func TestCommandProviderBinaryReplacementInvalidatesTrust(t *testing.T) {
	config, packet, marker := commandFixture(t, "valid")
	content, err := os.ReadFile(config.Executable)
	if err != nil {
		t.Fatal(err)
	}
	copyPath := filepath.Join(t.TempDir(), "provider-copy")
	if err := os.WriteFile(copyPath, content, 0o700); err != nil {
		t.Fatal(err)
	}
	config.Executable, err = filepath.EvalSymlinks(copyPath)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := DescribeCommand(config, packet)
	if err != nil {
		t.Fatal(err)
	}
	trust, egress := grants(plan, config.Now())
	command, err := NewCommand(config, packet, trust, egress)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.Executable, append(content, byte('\n')), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := command.Review(context.Background(), packet); !errors.Is(err, ErrCommandUnauthorized) {
		t.Fatalf("replacement must invalidate trust, got %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("replaced command executed; marker err=%v", err)
	}
}

func TestCommandProviderNetworkDeclarationCannotExceedPlan(t *testing.T) {
	config, packet, _ := commandFixture(t, "valid")
	config.NetworkDestinationClass = "none"
	command := authorizedCommand(t, config, packet)
	if _, err := command.Review(context.Background(), packet); !errors.Is(err, ErrHandshakeInvalid) {
		t.Fatalf("network declaration mismatch was accepted: %v", err)
	}
}

func TestCommandProviderRejectsHandshakeTimeoutAndOversize(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want error
	}{
		{name: "incompatible", mode: "bad-handshake", want: ErrHandshakeInvalid},
		{name: "timeout", mode: "sleep"},
		{name: "oversize", mode: "oversize"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, packet, _ := commandFixture(t, test.mode)
			if test.mode == "timeout" {
				config.Timeout = 20 * time.Millisecond
			}
			if test.mode == "oversize" {
				config.MaxStdout = 32
			}
			command := authorizedCommand(t, config, packet)
			_, err := command.Handshake(context.Background())
			if err == nil {
				t.Fatal("hostile Provider output was accepted")
			}
			if test.want != nil && !errors.Is(err, test.want) {
				t.Fatalf("error=%v, want %v", err, test.want)
			}
		})
	}
}

func TestCommandProviderHelper(t *testing.T) {
	if os.Getenv("GO_WANT_COMMAND_PROVIDER_HELPER") != "1" {
		return
	}
	if marker := os.Getenv("DD_MARKER"); marker != "" {
		_ = os.WriteFile(marker, []byte("executed"), 0o600)
	}
	switch os.Getenv("DD_MODE") {
	case "sleep":
		time.Sleep(time.Second)
		os.Exit(0)
	case "oversize":
		_, _ = os.Stdout.Write([]byte(strings.Repeat("x", 1024)))
		os.Exit(0)
	case "bad-handshake":
		_, _ = io.WriteString(os.Stdout, `{"protocol_version":"2.0","provider":"fixture-command","capabilities":["review"],"max_input_bytes":1000000,"supports_resume":false,"network_access":"unknown"}`)
		os.Exit(0)
	}
	var request commandRequest
	if err := json.NewDecoder(os.Stdin).Decode(&request); err != nil {
		os.Exit(2)
	}
	if request.Operation == "handshake" {
		_ = json.NewEncoder(os.Stdout).Encode(publicschema.ProviderHandshake{
			ProtocolVersion: "1.0", Provider: "fixture-command",
			Capabilities: []string{"review", "structured-result"}, MaxInputBytes: 1000000,
			SupportsResume: false, NetworkAccess: "unknown",
		})
		os.Exit(0)
	}
	if request.Packet == nil {
		os.Exit(3)
	}
	_ = json.NewEncoder(os.Stdout).Encode(results.Result{
		SchemaVersion: "1.1", TaskID: request.Packet.TaskID, SnapshotID: request.Packet.SnapshotID,
		TaskInputHash: request.Packet.TaskInputHash,
		Reviewer: results.Reviewer{
			Provider: "fixture-command", Model: "fixture", ModelFamily: "fixture",
			PassID: "fixture-1", Perspective: "correctness",
			PromptDigest: request.Packet.PromptDigest, ContextIsolation: "fresh helper process",
		},
		Coverage: []results.Coverage{}, Findings: []results.Finding{},
		NeedsConfirmation: []results.Confirmation{}, ResidualRisks: []results.ResidualRisk{}, Status: "incomplete",
	})
	os.Exit(0)
}

func commandFixture(t *testing.T, mode string) (CommandConfig, packets.Packet, string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		t.Fatal(err)
	}
	repositoryRoot := t.TempDir()
	workingDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "executed")
	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	task := planner.Task{
		SchemaVersion: "1.0", ID: "task-command",
		SnapshotID: "snap-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Paths:      []planner.PathRef{}, DependencyTasks: []string{}, ContractTypes: []string{},
		Perspectives: []string{"correctness"}, RequiredPasses: 1,
	}
	packet, err := packets.Build(task, publicschema.PrivateProject)
	if err != nil {
		t.Fatal(err)
	}
	config := CommandConfig{
		RepositoryID: "repo-fixture", RepositoryRoot: repositoryRoot, Executable: executable,
		Args: []string{"-test.run=TestCommandProviderHelper"}, WorkingDir: workingDir,
		Env:     []string{"GO_WANT_COMMAND_PROVIDER_HELPER=1", "DD_MODE=" + mode, "DD_MARKER=" + marker},
		Timeout: 2 * time.Second, MaxStdout: 1 << 20, MaxStderr: 1 << 16,
		ConfigDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		DataClass:    publicschema.PrivateProject, NetworkDestinationClass: "unknown",
		CredentialSource: "none",
		RedactionDigest:  "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		Now:              func() time.Time { return now },
	}
	return config, packet, marker
}

func authorizedCommand(t *testing.T, config CommandConfig, packet packets.Packet) *Command {
	t.Helper()
	plan, err := DescribeCommand(config, packet)
	if err != nil {
		t.Fatal(err)
	}
	trust, egress := grants(plan, config.Now())
	command, err := NewCommand(config, packet, trust, egress)
	if err != nil {
		t.Fatal(err)
	}
	return command
}

func grants(plan CommandPlan, now time.Time) (policy.TrustBinding, policy.EgressGrant) {
	trust := plan.TrustCandidate
	trust.ExpiresAt = now.Add(time.Hour)
	egress := policy.EgressGrant{
		Provider: plan.EgressRequest.Provider, SnapshotID: plan.EgressRequest.SnapshotID,
		TaskInputDigest: plan.EgressRequest.TaskInputDigest, DataClass: plan.EgressRequest.DataClass,
		MaxBytes: plan.EgressRequest.Bytes, ExpiresAt: now.Add(time.Hour),
	}
	return trust, egress
}
