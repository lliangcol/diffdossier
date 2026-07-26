package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lliangcol/diffdossier/internal/buildinfo"
	"github.com/lliangcol/diffdossier/internal/contracts"
	"github.com/lliangcol/diffdossier/internal/gitrepo"
	"github.com/lliangcol/diffdossier/internal/platform"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

// Run executes the command without terminating the process, which keeps CLI behavior testable.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return ExitUsage
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(stdout)
		return ExitOK
	case "version":
		return runVersion(args[1:], stdout, stderr)
	case "config":
		return runConfig(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "prepare":
		return runPrepare(args[1:], stdout, stderr)
	case "plan":
		return runPlan(args[1:], stdout, stderr)
	case "status":
		return runStatus(args[1:], stdout, stderr)
	case "packet":
		return runPacket(args[1:], stdout, stderr)
	case "record":
		return runRecord(args[1:], stdout, stderr)
	case "review":
		return runReview(args[1:], stdout, stderr)
	case "gates":
		return runGates(args[1:], stdout, stderr)
	case "export":
		return runExport(args[1:], stdout, stderr)
	case "verify":
		return runVerify(args[1:], stdout, stderr, false)
	case "finalize":
		return runVerify(args[1:], stdout, stderr, true)
	case "finding":
		return runFinding(args[1:], stdout, stderr)
	case "fix":
		return runFix(args[1:], stdout, stderr)
	case "refresh":
		return runRefresh(args[1:], stdout, stderr)
	case "recover":
		return runRecover(args[1:], stdout, stderr)
	case "run":
		return runRun(args[1:], stdout, stderr)
	case "gc":
		return runGC(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return ExitUsage
	}
}

func runConfig(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "validate" {
		fmt.Fprintln(stderr, "usage: diffdossier config validate [--repo PATH] [--config PATH] [--baseline REF] [--json]")
		return ExitUsage
	}
	flags := flag.NewFlagSet("config validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "target Git repository")
	configPath := flags.String("config", "", "configuration file (default: <repo>/diffdossier.toml)")
	baseline := flags.String("baseline", "", "exact local baseline ref override")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args[1:]); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 {
		return ExitUsage
	}

	resolvedRepo, err := filepath.Abs(*repo)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", err.Error()), ExitUsage)
	}
	effective, err := loadEffectiveConfig(resolvedRepo, *configPath, *baseline)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_CONFIG_INVALID", err.Error()), ExitUsage)
	}

	result := map[string]any{
		"config_digest":  effective.Digest,
		"config_sources": effective.Sources,
		"schema_version": effective.Config.SchemaVersion,
		"status":         "valid",
	}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(result))
	}
	_, err = fmt.Fprintf(stdout, "configuration valid: %s (schema %d)\n", effective.Digest, effective.Config.SchemaVersion)
	if err != nil {
		fmt.Fprintf(stderr, "write config validation output: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoFlag := flags.String("repo", ".", "target Git repository")
	configFlag := flags.String("config", "", "configuration file")
	baselineFlag := flags.String("baseline", "", "exact local baseline ref override")
	stateFlag := flags.String("state-dir", "", "durable state directory")
	cacheFlag := flags.String("cache-dir", "", "rebuildable cache directory")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 {
		return ExitUsage
	}
	stateRoot, err := resolveStateRoot(*stateFlag)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", err.Error()), ExitUsage)
	}
	cacheRoot, err := resolveCacheRoot(*cacheFlag)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", err.Error()), ExitUsage)
	}
	result := map[string]any{
		"state_dir":             stateRoot,
		"cache_dir":             cacheRoot,
		"network_default":       "none",
		"default_provider":      "manual",
		"telemetry":             false,
		"strong_os_sandbox":     false,
		"platform_capabilities": platform.CurrentCapabilities(),
		"commands_executed":     0,
	}
	absoluteRepo, absErr := filepath.Abs(*repoFlag)
	if absErr != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", absErr.Error()), ExitUsage)
	}
	result["requested_repo"] = filepath.Clean(absoluteRepo)
	repo, repoErr := gitrepo.Open(context.Background(), absoluteRepo)
	if repoErr != nil {
		result["status"] = "needs_confirmation"
		result["repository_status"] = "not_git_repository"
		result["repository_error"] = repoErr.Error()
		result["rules"] = []contracts.Rule{}
		result["rule_conflicts"] = []ruleConflict{}
	} else {
		result["repository_status"] = "valid"
		result["repository_root"] = repo.Root
		if stateErr := requireOutsideRepository(repo.Root, stateRoot); stateErr != nil {
			result["status"] = "needs_confirmation"
			result["state_status"] = "invalid"
			result["state_error"] = stateErr.Error()
		} else {
			result["state_status"] = "valid"
		}
		if cacheErr := requireOutsideRepository(repo.Root, cacheRoot); cacheErr != nil {
			result["status"] = "needs_confirmation"
			result["cache_status"] = "invalid"
			result["cache_error"] = "cache-dir must be outside the target repository"
		} else {
			result["cache_status"] = "valid"
		}
		effective, configErr := loadEffectiveConfig(repo.Root, *configFlag, *baselineFlag)
		if configErr != nil {
			result["status"] = "needs_confirmation"
			result["config_status"] = "invalid"
			result["config_error"] = configErr.Error()
		} else {
			result["config_status"] = "valid"
			result["config_digest"] = effective.Digest
			result["config_sources"] = effective.Sources
			result["default_provider"] = effective.Config.Review.DefaultProvider
			result["effective_config"] = map[string]any{
				"schema_version":    effective.Config.SchemaVersion,
				"baseline":          effective.Config.Baseline,
				"include_untracked": effective.Config.IncludeUntracked,
				"gate_count":        len(effective.Config.Gates),
				"risk_policy_files": effective.Config.Risk.PolicyFiles,
			}
			revisions, baselineErr := repo.Resolve(context.Background(), effective.Config.Baseline)
			if baselineErr != nil {
				result["status"] = "needs_confirmation"
				result["baseline_status"] = "unresolved"
				result["baseline_error"] = baselineErr.Error()
			} else {
				result["baseline_status"] = "resolved_local_only"
				result["baseline_commit"] = revisions.BaselineCommit
				result["head_commit"] = revisions.HeadCommit
				result["merge_base"] = revisions.MergeBase
				result["remote_fetch_proof"] = revisions.RemoteFetchProof
			}
		}
		rules, ruleErr := contracts.DiscoverRules(repo.Root)
		if ruleErr != nil {
			result["status"] = "needs_confirmation"
			result["rule_status"] = "invalid"
			result["rule_error"] = ruleErr.Error()
		} else {
			conflicts := findRuleConflicts(rules)
			result["rules"] = rules
			result["rule_conflicts"] = conflicts
			result["effective_policy"] = map[string]any{
				"precedence":   "operator > nearest scoped project rule > repository root rule > built-in defaults",
				"source_order": effectiveRuleOrder(rules),
			}
			if len(conflicts) > 0 {
				result["status"] = "needs_confirmation"
				result["rule_status"] = "needs_confirmation"
			} else {
				result["rule_status"] = "resolved"
			}
		}
	}
	if _, ok := result["status"]; !ok {
		result["status"] = "ready"
	}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(result))
	}
	_, err = fmt.Fprintf(stdout, "DiffDossier doctor\nstatus: %s\nstate: %s\ncache: %s\nnetwork: none\nprovider: %s\ntelemetry: disabled\n", result["status"], stateRoot, cacheRoot, result["default_provider"])
	if err != nil {
		fmt.Fprintf(stderr, "write doctor output: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

type ruleConflict struct {
	Scope  string   `json:"scope"`
	Paths  []string `json:"paths"`
	Reason string   `json:"reason"`
}

func findRuleConflicts(rules []contracts.Rule) []ruleConflict {
	byScope := map[string][]contracts.Rule{}
	for _, rule := range rules {
		byScope[rule.Scope] = append(byScope[rule.Scope], rule)
	}
	conflicts := []ruleConflict{}
	scopes := make([]string, 0, len(byScope))
	for scope := range byScope {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	for _, scope := range scopes {
		scoped := byScope[scope]
		if len(scoped) < 2 {
			continue
		}
		digests := map[string]bool{}
		paths := make([]string, 0, len(scoped))
		for _, rule := range scoped {
			digests[rule.Digest] = true
			paths = append(paths, rule.Path)
		}
		sort.Strings(paths)
		if len(digests) > 1 {
			conflicts = append(conflicts, ruleConflict{Scope: scope, Paths: paths, Reason: "multiple non-identical rule sources govern the same scope"})
		}
	}
	return conflicts
}

func effectiveRuleOrder(rules []contracts.Rule) []string {
	ordered := append([]contracts.Rule{}, rules...)
	sort.Slice(ordered, func(i, j int) bool {
		leftDepth := strings.Count(ordered[i].Scope, "/")
		rightDepth := strings.Count(ordered[j].Scope, "/")
		if ordered[i].Scope != "" {
			leftDepth++
		}
		if ordered[j].Scope != "" {
			rightDepth++
		}
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		return ordered[i].Path < ordered[j].Path
	})
	order := make([]string, 0, len(ordered))
	for _, rule := range ordered {
		order = append(order, rule.Path)
	}
	return order
}

func writeJSON(stdout, stderr io.Writer, value any) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintf(stderr, "encode JSON output: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

func writeFailure(stdout, stderr io.Writer, jsonOutput bool, problem publicschema.Problem, code int) int {
	if jsonOutput {
		if writeCode := writeJSON(stdout, stderr, publicschema.Failure(problem)); writeCode != ExitOK {
			return writeCode
		}
		return code
	}
	fmt.Fprintln(stderr, problem.Message)
	return code
}

func runVersion(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("version", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil {
		return ExitUsage
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "version does not accept positional arguments")
		return ExitUsage
	}

	info := buildinfo.Current()
	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(info); err != nil {
			fmt.Fprintf(stderr, "encode version output: %v\n", err)
			return ExitInternal
		}
		return ExitOK
	}

	if _, err := fmt.Fprintf(
		stdout,
		"diffdossier %s commit=%s built=%s go=%s %s/%s cgo=%s\n",
		info.Version,
		info.Commit,
		info.BuildDate,
		info.GoVersion,
		info.OS,
		info.Architecture,
		info.CGOEnabled,
	); err != nil {
		fmt.Fprintf(stderr, "write version output: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, "Usage:")
	fmt.Fprintln(writer, "  diffdossier version [--json]")
	fmt.Fprintln(writer, "  diffdossier doctor [--repo PATH] [--config PATH] [--baseline REF] [--state-dir PATH] [--cache-dir PATH] [--json]")
	fmt.Fprintln(writer, "  diffdossier init --baseline REF [--repo PATH] [--json]")
	fmt.Fprintln(writer, "  diffdossier prepare [--repo PATH] [--config PATH] [--baseline REF] [--state-dir PATH] [--json]")
	fmt.Fprintln(writer, "  diffdossier plan [--repo PATH] [--config PATH] [--baseline REF] [--state-dir PATH] [--run-id ID] [--json]")
	fmt.Fprintln(writer, "  diffdossier status [--repo PATH] [--state-dir PATH] [--run-id ID] [--json]")
	fmt.Fprintln(writer, "  diffdossier packet contract [--repo PATH] [--config PATH] [--baseline REF] [--state-dir PATH] [--run-id ID] [--json]")
	fmt.Fprintln(writer, "  diffdossier packet task --task-id ID [--repo PATH] [--config PATH] [--baseline REF] [--state-dir PATH] [--run-id ID] [--json]")
	fmt.Fprintln(writer, "  diffdossier record contract [--repo PATH] [--config PATH] [--baseline REF] [--state-dir PATH] [--run-id ID] [--json]")
	fmt.Fprintln(writer, "  diffdossier record task --task-id ID --result PATH [--repo PATH] [--config PATH] [--baseline REF] [--state-dir PATH] [--run-id ID] [--json]")
	fmt.Fprintln(writer, "  diffdossier review run --task-id ID [--config PATH] [--baseline REF] [--provider manual|command] [provider authorization options] [--json]")
	fmt.Fprintln(writer, "  diffdossier gates plan [--repo PATH] [--config PATH] [--baseline REF] [--state-dir PATH] [--run-id ID] [--json]")
	fmt.Fprintln(writer, "  diffdossier gates run --trust-execution-plan DIGEST [--trust-shell] [--repo PATH] [--config PATH] [--baseline REF] [--state-dir PATH] [--run-id ID] [--json]")
	fmt.Fprintln(writer, "  diffdossier export portable --output PATH [--repo PATH] [--state-dir PATH] [--run-id ID] [--json]")
	fmt.Fprintln(writer, "  diffdossier export public prepare --input PATH --class CLASS --action ACTION --policy-digest DIGEST [--repo PATH] [--state-dir PATH] [--run-id ID] [--json]")
	fmt.Fprintln(writer, "  diffdossier export public approve --preparation-digest DIGEST --operator NAME [--trust-public-approval DIGEST] [options]")
	fmt.Fprintln(writer, "  diffdossier export public create --preparation-digest DIGEST --approval-digest DIGEST --output PATH [--trust-public-create DIGEST] [options]")
	fmt.Fprintln(writer, "  diffdossier export public revoke --approval-digest DIGEST --export-digest DIGEST --reason TEXT --output PATH [--trust-public-revoke DIGEST] [options]")
	fmt.Fprintln(writer, "  diffdossier verify [--repo PATH] [--config PATH] [--baseline REF] [--state-dir PATH] [--run-id ID] [--json]")
	fmt.Fprintln(writer, "  diffdossier finalize [--repo PATH] [--config PATH] [--baseline REF] [--state-dir PATH] [--run-id ID] [--json]")
	fmt.Fprintln(writer, "  diffdossier finding confirm|reject|accept-risk --finding-id ID --operator NAME [options]")
	fmt.Fprintln(writer, "  diffdossier fix authorize --finding-ids ID[,ID] --scope-digest DIGEST --operator NAME --expires-at RFC3339 [options]")
	fmt.Fprintln(writer, "  diffdossier refresh [--repo PATH] [--config PATH] [--baseline REF] [--state-dir PATH] [--run-id ID] [--json]")
	fmt.Fprintln(writer, "  diffdossier recover --trust-journal-state STATE [--repo PATH] [--state-dir PATH] [--run-id ID] [--json]")
	fmt.Fprintln(writer, "  diffdossier run archive --reason TEXT [--pin] [--repo PATH] [--state-dir PATH] [--run-id ID] [--json]")
	fmt.Fprintln(writer, "  diffdossier gc [plan] [--repo PATH] [--config PATH] [--baseline REF] [--state-dir PATH] [--as-of RFC3339] [--json]")
	fmt.Fprintln(writer, "  diffdossier gc run --trust-gc-plan DIGEST [--state-dir PATH] [--json]")
	fmt.Fprintln(writer, "  diffdossier config validate [--repo PATH] [--config PATH] [--baseline REF] [--json]")
	fmt.Fprintln(writer, "  diffdossier help")
}
