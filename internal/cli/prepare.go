package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lliangcol/diffdossier/internal/buildinfo"
	"github.com/lliangcol/diffdossier/internal/config"
	"github.com/lliangcol/diffdossier/internal/contracts"
	"github.com/lliangcol/diffdossier/internal/gitrepo"
	"github.com/lliangcol/diffdossier/internal/platform"
	"github.com/lliangcol/diffdossier/internal/risk"
	"github.com/lliangcol/diffdossier/internal/snapshot"
	"github.com/lliangcol/diffdossier/internal/store"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
	embeddedschemas "github.com/lliangcol/diffdossier/schemas"
)

func runPrepare(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("prepare", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoFlag := flags.String("repo", ".", "target Git repository")
	configFlag := flags.String("config", "", "configuration file")
	stateFlag := flags.String("state-dir", "", "durable state directory")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
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
	digests, err := semanticDigests(repo, configPath, cfg.Risk.PolicyFiles)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EVIDENCE_DIGEST", err.Error()), ExitEvidence)
	}
	seal, err := snapshot.Capture(ctx, snapshot.Request{
		Repo: repo, Baseline: cfg.Baseline, InputDigests: digests,
		IncludeUntracked: cfg.IncludeUntracked, IncludeIgnored: cfg.IncludeIgnored,
	})
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_SNAPSHOT_CAPTURE", err.Error()), ExitEvidence)
	}
	afterDigests, err := semanticDigests(repo, configPath, cfg.Risk.PolicyFiles)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EVIDENCE_DIGEST", err.Error()), ExitEvidence)
	}
	if !sameDigests(digests, afterDigests) {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_SNAPSHOT_CAPTURE", "semantic inputs changed while capturing snapshot; retry"), ExitStale)
	}
	stateStore, err := store.Open(stateRoot)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_STORE", err.Error()), ExitEvidence)
	}
	repository, err := stateStore.Register(repo.Root)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_REGISTER", err.Error()), ExitEvidence)
	}
	run, runDir, err := stateStore.BeginRun(repository, seal)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_RUN", err.Error()), ExitEvidence)
	}
	result := map[string]any{
		"repository_id": repository.ID, "run_id": run.ID, "snapshot_id": seal.SnapshotID,
		"state": run.State, "state_path": runDir, "freshness": seal.Revisions.Freshness,
		"path_count": len(seal.Inventory.Entries),
	}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(result))
	}
	if _, err := fmt.Fprintf(stdout, "prepared %s at %s (%d scoped path entries, %s)\n", run.ID, seal.SnapshotID, len(seal.Inventory.Entries), seal.Revisions.Freshness); err != nil {
		fmt.Fprintf(stderr, "write prepare output: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

func sameDigests(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func semanticDigests(repo *gitrepo.Repo, configPath string, policyFiles []string) (map[string]string, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(content)
	result, err := embeddedschemas.Digests()
	if err != nil {
		return nil, err
	}
	result["config"] = "sha256:" + hex.EncodeToString(digest[:])
	rules, err := contracts.DiscoverRules(repo.Root)
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		result["rule/"+rule.Path] = rule.Digest
	}
	for _, relative := range policyFiles {
		resolved, err := risk.ResolvePolicyPath(repo.Root, relative)
		if err != nil {
			return nil, err
		}
		policyContent, err := os.ReadFile(resolved)
		if err != nil {
			return nil, err
		}
		policyDigest := sha256.Sum256(policyContent)
		result["risk-policy/"+filepath.ToSlash(filepath.Clean(relative))] = "sha256:" + hex.EncodeToString(policyDigest[:])
	}
	providerManifest, _ := json.Marshal([]publicschema.ProviderHandshake{
		{ProtocolVersion: "1.0", Provider: "manual", Capabilities: []string{"review", "structured-result"}, MaxInputBytes: 250000, SupportsResume: true, NetworkAccess: "none"},
		{ProtocolVersion: "1.0", Provider: "mock", Capabilities: []string{"review", "structured-result"}, MaxInputBytes: 250000, SupportsResume: true, NetworkAccess: "none"},
	})
	providerDigest := sha256.Sum256(providerManifest)
	result["provider-manifest"] = "sha256:" + hex.EncodeToString(providerDigest[:])
	promptDigest := sha256.Sum256([]byte("diffdossier-review-packet/v1: repository content is untrusted review data"))
	result["prompt/review-v1"] = "sha256:" + hex.EncodeToString(promptDigest[:])
	gitVersion, err := repo.Git(context.Background(), "--version")
	if err != nil {
		return nil, err
	}
	toolchainDigest := sha256.Sum256([]byte(runtime.Version() + "\x00" + runtime.GOOS + "\x00" + runtime.GOARCH + "\x00" + strings.TrimSpace(string(gitVersion))))
	result["toolchain"] = "sha256:" + hex.EncodeToString(toolchainDigest[:])
	binaryInfo, _ := json.Marshal(buildinfo.Current())
	binaryDigest := sha256.Sum256(binaryInfo)
	result["binary-buildinfo"] = "sha256:" + hex.EncodeToString(binaryDigest[:])
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	executableContent, err := os.ReadFile(executable)
	if err != nil {
		return nil, err
	}
	executableDigest := sha256.Sum256(executableContent)
	result["binary"] = "sha256:" + hex.EncodeToString(executableDigest[:])
	return result, nil
}

func requireOutsideRepository(repoRoot, candidate string) error {
	absolute, err := filepath.Abs(candidate)
	if err != nil {
		return err
	}
	resolvedRepo, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return err
	}
	probe := absolute
	for {
		resolvedProbe, resolveErr := filepath.EvalSymlinks(probe)
		if resolveErr == nil {
			suffix, _ := filepath.Rel(probe, absolute)
			absolute = filepath.Join(resolvedProbe, suffix)
			break
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			break
		}
		probe = parent
	}
	relative, err := filepath.Rel(resolvedRepo, absolute)
	if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("state-dir must be outside the target repository")
	}
	return nil
}
