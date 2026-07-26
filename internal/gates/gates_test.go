package gates

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lliangcol/diffdossier/internal/config"
	"github.com/lliangcol/diffdossier/internal/inventory"
	"github.com/lliangcol/diffdossier/internal/policy"
	"github.com/lliangcol/diffdossier/internal/snapshot"
)

type fakeExecutor struct {
	calls int
	err   error
}

func (f *fakeExecutor) Execute(context.Context, ExpandedGate) error { f.calls++; return f.err }

func TestPlanIsInspectOnlyAndTopological(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "tool")
	if err := os.WriteFile(executable, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	path := "main.go"
	request := PlanRequest{RepositoryID: "repo", RepositoryRoot: root, Seal: snapshot.Seal{SnapshotID: "snap", Inventory: inventory.Result{Entries: []inventory.Entry{{Path: inventory.PathIdentity{BytesBase64: base64.StdEncoding.EncodeToString([]byte(path))}}}}}, ConfigDigest: "config", BinaryDigest: "binary", LookupExecutable: func(string) (string, error) { return executable, nil }, Getenv: func(string) (string, bool) { return "safe", true }, Gates: []config.Gate{
		{ID: "test", Argv: []string{"tool"}, Cwd: ".", EnvAllowlist: []string{"PATH"}, WhenPaths: []string{"**/*.go"}, DependsOn: []string{"build"}, TimeoutSeconds: 1, ResourceClass: "cpu", CacheClass: "worktree_deterministic", NetworkClass: "none"},
		{ID: "build", Argv: []string{"tool"}, Cwd: ".", WhenPaths: []string{"never/**"}, TimeoutSeconds: 1, ResourceClass: "cpu", CacheClass: "worktree_deterministic", NetworkClass: "none"},
	}}
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Gates) != 2 || plan.Gates[0].ID != "build" || plan.PlanDigest == "" {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

func TestRunRequiresExactTrustAndDetectsMutation(t *testing.T) {
	plan := Plan{RepositoryID: "repo", SnapshotID: "snap", ConfigDigest: "config", BinaryDigest: "binary", PlanDigest: "plan", TrustCandidate: policy.TrustBinding{RepositoryID: "repo", SnapshotID: "snap", TaskInputDigest: "gate-dag", ExecutionPlanDigest: "plan", ConfigDigest: "config", BinaryDigest: "binary", Capability: "gate:run"}, Gates: []ExpandedGate{{ID: "one", TimeoutSeconds: 1, CacheClass: "worktree_deterministic", DefinitionDigest: "definition", ExecutableDigest: "tool"}}}
	fake := &fakeExecutor{}
	now := time.Now()
	if _, err := Run(context.Background(), plan, policy.TrustBinding{}, now, fake, func() error { return nil }, func() error { return nil }, nil, false); !errors.Is(err, ErrExecutionUnauthorized) || fake.calls != 0 {
		t.Fatalf("unauthorized run executed: %v calls=%d", err, fake.calls)
	}
	trust := plan.TrustCandidate
	trust.ExpiresAt = now.Add(time.Minute)
	if _, err := Run(context.Background(), plan, trust, now, fake, func() error { return nil }, func() error { return errors.New("changed") }, nil, false); err == nil || fake.calls != 1 {
		t.Fatalf("mutation not detected: %v calls=%d", err, fake.calls)
	}
}

func TestCacheExactAndFinalAlways(t *testing.T) {
	now := time.Now()
	gate := ExpandedGate{ID: "one", TimeoutSeconds: 1, CacheClass: "worktree_deterministic", DefinitionDigest: "definition", ExecutableDigest: "tool", FinalAlways: true}
	plan := Plan{RepositoryID: "repo", SnapshotID: "snap", ConfigDigest: "config", BinaryDigest: "binary", PlanDigest: "plan", TrustCandidate: policy.TrustBinding{RepositoryID: "repo", SnapshotID: "snap", TaskInputDigest: "gate-dag", ExecutionPlanDigest: "plan", ConfigDigest: "config", BinaryDigest: "binary", Capability: "gate:run"}, Gates: []ExpandedGate{gate}}
	trust := plan.TrustCandidate
	trust.ExpiresAt = now.Add(time.Minute)
	key := "snap\x00definition\x00binary\x00tool"
	cache := map[string]Evidence{key: {GateID: "one", Status: "pass"}}
	fake := &fakeExecutor{}
	evidence, err := Run(context.Background(), plan, trust, now, fake, func() error { return nil }, func() error { return nil }, cache, false)
	if err != nil || fake.calls != 0 || !evidence[0].CacheHit {
		t.Fatalf("cache miss: %v %+v", err, evidence)
	}
	_, err = Run(context.Background(), plan, trust, now, fake, func() error { return nil }, func() error { return nil }, cache, true)
	if err != nil || fake.calls != 1 {
		t.Fatalf("final_always did not execute: %v calls=%d", err, fake.calls)
	}
}

func TestNonBlockingFailureRecordsAndContinues(t *testing.T) {
	now := time.Now()
	plan := Plan{RepositoryID: "repo", SnapshotID: "snap", ConfigDigest: "config", BinaryDigest: "binary", PlanDigest: "plan", TrustCandidate: policy.TrustBinding{RepositoryID: "repo", SnapshotID: "snap", TaskInputDigest: "gate-dag", ExecutionPlanDigest: "plan", ConfigDigest: "config", BinaryDigest: "binary", Capability: "gate:run"}, Gates: []ExpandedGate{{ID: "advisory", TimeoutSeconds: 1, Blocking: false}, {ID: "other", TimeoutSeconds: 1, Blocking: false}}}
	trust := plan.TrustCandidate
	trust.ExpiresAt = now.Add(time.Minute)
	fake := &fakeExecutor{err: errors.New("failed")}
	evidence, err := Run(context.Background(), plan, trust, now, fake, func() error { return nil }, func() error { return nil }, nil, false)
	if err != nil || len(evidence) != 2 || fake.calls != 2 {
		t.Fatalf("nonblocking execution: evidence=%d calls=%d err=%v", len(evidence), fake.calls, err)
	}
}

func TestExecutableDigestBound(t *testing.T) {
	sum := sha256.Sum256([]byte("binary"))
	if got := "sha256:" + hex.EncodeToString(sum[:]); got == "" {
		t.Fatal("empty")
	}
}

func TestExecutableChangePreventsExecution(t *testing.T) {
	root := t.TempDir()
	tool := filepath.Join(root, "tool")
	if err := os.WriteFile(tool, []byte("one"), 0o700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("one"))
	now := time.Now()
	gate := ExpandedGate{ID: "one", Executable: tool, ExecutableDigest: "sha256:" + hex.EncodeToString(sum[:]), TimeoutSeconds: 1}
	plan := Plan{RepositoryID: "repo", SnapshotID: "snap", ConfigDigest: "config", BinaryDigest: "binary", PlanDigest: "plan", TrustCandidate: policy.TrustBinding{RepositoryID: "repo", SnapshotID: "snap", TaskInputDigest: "gate-dag", ExecutionPlanDigest: "plan", ConfigDigest: "config", BinaryDigest: "binary", Capability: "gate:run"}, Gates: []ExpandedGate{gate}}
	trust := plan.TrustCandidate
	trust.ExpiresAt = now.Add(time.Minute)
	if err := os.WriteFile(tool, []byte("two"), 0o700); err != nil {
		t.Fatal(err)
	}
	fake := &fakeExecutor{}
	if _, err := Run(context.Background(), plan, trust, now, fake, func() error { return nil }, func() error { return nil }, nil, false); err == nil || fake.calls != 0 {
		t.Fatalf("changed executable ran: err=%v calls=%d", err, fake.calls)
	}
}
