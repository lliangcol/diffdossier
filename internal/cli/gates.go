package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lliangcol/diffdossier/internal/config"
	"github.com/lliangcol/diffdossier/internal/gates"
	"github.com/lliangcol/diffdossier/internal/gitrepo"
	"github.com/lliangcol/diffdossier/internal/platform"
	"github.com/lliangcol/diffdossier/internal/policy"
	"github.com/lliangcol/diffdossier/internal/process"
	"github.com/lliangcol/diffdossier/internal/redact"
	"github.com/lliangcol/diffdossier/internal/snapshot"
	"github.com/lliangcol/diffdossier/internal/store"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

func runGates(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "run" {
		return runGatesExecute(args[1:], stdout, stderr)
	}
	if len(args) == 0 || args[0] != "plan" {
		fmt.Fprintln(stderr, "usage: diffdossier gates plan ... | diffdossier gates run --trust-execution-plan DIGEST ...")
		return ExitUsage
	}
	flags := flag.NewFlagSet("gates plan", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoFlag := flags.String("repo", ".", "target Git repository")
	configFlag := flags.String("config", "", "configuration file")
	stateFlag := flags.String("state-dir", "", "durable state directory")
	runFlag := flags.String("run-id", "", "prepared run ID (default: latest)")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args[1:]); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 {
		return ExitUsage
	}
	ctx := context.Background()
	repo, err := gitrepo.Open(ctx, *repoFlag)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_GIT_REPOSITORY", err.Error()), ExitEvidence)
	}
	configPath := *configFlag
	if configPath == "" {
		configPath = filepath.Join(repo.Root, "diffdossier.toml")
	} else if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(repo.Root, configPath)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_CONFIG_INVALID", err.Error()), ExitUsage)
	}
	stateRoot := *stateFlag
	if stateRoot == "" {
		paths, pathErr := platform.DefaultPaths()
		if pathErr != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PLATFORM_PATHS", pathErr.Error()), ExitInternal)
		}
		stateRoot = paths.StateDir
	}
	if !filepath.IsAbs(stateRoot) {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", "state-dir must be absolute"), ExitUsage)
	}
	if err := requireOutsideRepository(repo.Root, stateRoot); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", err.Error()), ExitUsage)
	}
	stateStore, err := store.Open(stateRoot)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_STORE", err.Error()), ExitEvidence)
	}
	repository, err := stateStore.Register(repo.Root)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_REGISTER", err.Error()), ExitEvidence)
	}
	runID := *runFlag
	if runID == "" {
		latest, latestErr := stateStore.LatestRun(repository.ID)
		if latestErr != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_RUN", latestErr.Error()), ExitEvidence)
		}
		runID = latest.ID
	}
	run, seal, err := stateStore.LoadRun(repository.ID, runID)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_RUN", err.Error()), ExitEvidence)
	}
	digests, err := semanticDigests(repo, configPath, cfg.Risk.PolicyFiles)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EVIDENCE_DIGEST", err.Error()), ExitEvidence)
	}
	request := snapshot.Request{Repo: repo, Baseline: cfg.Baseline, InputDigests: digests, IncludeUntracked: cfg.IncludeUntracked, IncludeIgnored: cfg.IncludeIgnored}
	if err := snapshot.VerifyFresh(ctx, request, seal); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_SNAPSHOT_STALE", err.Error()), ExitStale)
	}
	plan, err := gates.BuildPlan(gates.PlanRequest{RepositoryID: repository.ID, RepositoryRoot: repo.Root, Seal: seal, ConfigDigest: digests["config"], BinaryDigest: digests["binary"], Gates: cfg.Gates})
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_GATE_PLAN", err.Error()), ExitEvidence)
	}
	runDir, err := stateStore.RunDir(repository.ID, run.ID)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_RUN", err.Error()), ExitEvidence)
	}
	if err := stateStore.WriteRunJSON(runDir, "gates/plan.json", plan); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
	}
	if _, err := stateStore.AppendEvent(runDir, "gate_plan_created", map[string]any{"plan_digest": plan.PlanDigest, "gate_count": len(plan.Gates)}); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_EVENT", err.Error()), ExitEvidence)
	}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(plan))
	}
	if _, err := fmt.Fprintf(stdout, "planned %d gates for %s; trust execution plan %s before any run\n", len(plan.Gates), plan.SnapshotID, plan.PlanDigest); err != nil {
		return ExitInternal
	}
	return ExitOK
}

type processGateExecutor struct {
	stateStore *store.Store
	runDir     string
}

func (executor processGateExecutor) Execute(ctx context.Context, gate gates.ExpandedGate) error {
	env := []string{}
	knownValues := []string{}
	for _, binding := range gate.Environment {
		value, present := os.LookupEnv(binding.Name)
		sum := sha256.Sum256([]byte(value))
		if present != binding.Present || "sha256:"+hex.EncodeToString(sum[:]) != binding.ValueDigest {
			return errors.New("allowed environment changed after planning")
		}
		if present {
			env = append(env, binding.Name+"="+value)
			knownValues = append(knownValues, value)
		}
	}
	output, err := process.Run(ctx, process.Spec{Executable: gate.Executable, Args: gate.Argv[1:], Dir: gate.Cwd, Env: env, MaxStdout: 4 * 1024 * 1024, MaxStderr: 4 * 1024 * 1024})
	stdoutRedacted, stdoutManifest, stdoutErr := redact.RedactKnown(output.Stdout, knownValues)
	stderrRedacted, stderrManifest, stderrErr := redact.RedactKnown(output.Stderr, knownValues)
	if stdoutErr != nil || stderrErr != nil {
		return errors.Join(err, stdoutErr, stderrErr)
	}
	if writeErr := executor.stateStore.WriteRunBytes(executor.runDir, filepath.Join("logs", gate.ID+".stdout"), stdoutRedacted); writeErr != nil {
		return errors.Join(err, writeErr)
	}
	if writeErr := executor.stateStore.WriteRunBytes(executor.runDir, filepath.Join("logs", gate.ID+".stderr"), stderrRedacted); writeErr != nil {
		return errors.Join(err, writeErr)
	}
	if writeErr := executor.stateStore.WriteRunJSON(executor.runDir, filepath.Join("logs", gate.ID+".redaction.json"), map[string]any{"stdout": stdoutManifest, "stderr": stderrManifest}); writeErr != nil {
		return errors.Join(err, writeErr)
	}
	return err
}

func runGatesExecute(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gates run", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoFlag := flags.String("repo", ".", "target Git repository")
	configFlag := flags.String("config", "", "configuration file")
	stateFlag := flags.String("state-dir", "", "durable state directory")
	runFlag := flags.String("run-id", "", "run ID")
	trustDigest := flags.String("trust-execution-plan", "", "exact plan digest shown by gates plan")
	trustShell := flags.Bool("trust-shell", false, "separately authorize shell-mode gates")
	finalRun := flags.Bool("final", false, "record this execution as the final-always run")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || *trustDigest == "" {
		return ExitUsage
	}
	ctx := context.Background()
	repo, err := gitrepo.Open(ctx, *repoFlag)
	if err != nil {
		return ExitEvidence
	}
	configPath := *configFlag
	if configPath == "" {
		configPath = filepath.Join(repo.Root, "diffdossier.toml")
	} else if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(repo.Root, configPath)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return ExitUsage
	}
	stateRoot := *stateFlag
	if stateRoot == "" {
		paths, pathErr := platform.DefaultPaths()
		if pathErr != nil {
			return ExitInternal
		}
		stateRoot = paths.StateDir
	}
	if !filepath.IsAbs(stateRoot) {
		return ExitUsage
	}
	if err := requireOutsideRepository(repo.Root, stateRoot); err != nil {
		return ExitUsage
	}
	stateStore, err := store.Open(stateRoot)
	if err != nil {
		return ExitEvidence
	}
	repository, err := stateStore.Register(repo.Root)
	if err != nil {
		return ExitEvidence
	}
	runID := *runFlag
	if runID == "" {
		latest, latestErr := stateStore.LatestRun(repository.ID)
		if latestErr != nil {
			return ExitEvidence
		}
		runID = latest.ID
	}
	run, seal, err := stateStore.LoadRun(repository.ID, runID)
	if err != nil {
		return ExitEvidence
	}
	if run.State != "REVIEWED" {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_WORKFLOW_STATE", "gates run requires REVIEWED run"), ExitIncomplete)
	}
	runDir, err := stateStore.RunDir(repository.ID, run.ID)
	if err != nil {
		return ExitEvidence
	}
	digests, err := semanticDigests(repo, configPath, cfg.Risk.PolicyFiles)
	if err != nil {
		return ExitEvidence
	}
	request := snapshot.Request{Repo: repo, Baseline: cfg.Baseline, InputDigests: digests, IncludeUntracked: cfg.IncludeUntracked, IncludeIgnored: cfg.IncludeIgnored}
	if err := snapshot.VerifyFresh(ctx, request, seal); err != nil {
		return ExitStale
	}
	plan, err := gates.BuildPlan(gates.PlanRequest{RepositoryID: repository.ID, RepositoryRoot: repo.Root, Seal: seal, ConfigDigest: digests["config"], BinaryDigest: digests["binary"], Gates: cfg.Gates})
	if err != nil {
		return ExitEvidence
	}
	var stored gates.Plan
	if err := stateStore.ReadRunJSON(runDir, "gates/plan.json", &stored); err != nil || !sameCanonicalJSON(stored, plan) {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_GATE_PLAN", "run requires the current persisted exact Gate plan"), ExitStale)
	}
	if *trustDigest != plan.PlanDigest {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_GATE_TRUST", "trust digest does not match exact plan"), ExitBlocked)
	}
	for _, gate := range plan.Gates {
		if gate.ShellMode && !*trustShell {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_GATE_SHELL_TRUST", "shell-mode Gate requires separate --trust-shell confirmation"), ExitBlocked)
		}
	}
	now := time.Now()
	trust := policy.TrustBinding{RepositoryID: plan.RepositoryID, SnapshotID: plan.SnapshotID, TaskInputDigest: "gate-dag", ExecutionPlanDigest: plan.PlanDigest, ConfigDigest: plan.ConfigDigest, BinaryDigest: plan.BinaryDigest, Capability: "gate:run", ExpiresAt: now.Add(5 * time.Minute)}
	fresh := func() error { return snapshot.VerifyFresh(ctx, request, seal) }
	evidence, err := gates.Run(ctx, plan, trust, now, processGateExecutor{stateStore: stateStore, runDir: runDir}, fresh, fresh, nil, *finalRun)
	if err != nil {
		code := ExitBlocked
		if strings.Contains(err.Error(), "stale") || strings.Contains(err.Error(), "mutation") {
			code = ExitStale
		}
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_GATE_RUN", err.Error()), code)
	}
	if err := stateStore.WriteRunJSON(runDir, "gates/evidence.json", evidence); err != nil {
		return ExitEvidence
	}
	if _, err := stateStore.AppendEvent(runDir, "gates_completed", map[string]any{"plan_digest": plan.PlanDigest, "gate_count": len(evidence)}); err != nil {
		return ExitEvidence
	}
	data := map[string]any{"run_id": run.ID, "plan_digest": plan.PlanDigest, "evidence": evidence}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "completed %d gates for exact plan %s\n", len(evidence), plan.PlanDigest)
	return ExitOK
}
