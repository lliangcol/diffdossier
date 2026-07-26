// Package gates expands untrusted project gate declarations into inspectable,
// content-bound execution plans. Planning never starts a child process.
package gates

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/lliangcol/diffdossier/internal/config"
	"github.com/lliangcol/diffdossier/internal/policy"
	"github.com/lliangcol/diffdossier/internal/snapshot"
)

type EnvBinding struct {
	Name        string `json:"name"`
	ValueDigest string `json:"value_digest"`
	Present     bool   `json:"present"`
}

type ExpandedGate struct {
	ID                  string       `json:"id"`
	RequestedExecutable string       `json:"requested_executable"`
	Executable          string       `json:"executable"`
	ExecutableDigest    string       `json:"executable_digest"`
	ExecutableClass     string       `json:"executable_class"`
	Argv                []string     `json:"argv"`
	Cwd                 string       `json:"cwd"`
	CwdClass            string       `json:"cwd_class"`
	Environment         []EnvBinding `json:"environment"`
	WhenPaths           []string     `json:"when_paths"`
	DependsOn           []string     `json:"depends_on"`
	TimeoutSeconds      int          `json:"timeout_seconds"`
	ResourceClass       string       `json:"resource_class"`
	CacheClass          string       `json:"cache_class"`
	Blocking            bool         `json:"blocking"`
	FinalAlways         bool         `json:"final_always"`
	NetworkClass        string       `json:"network_class"`
	ExpectedWrites      []string     `json:"expected_writes"`
	RedactionPolicy     string       `json:"redaction_policy"`
	ShellMode           bool         `json:"shell_mode"`
	DefinitionDigest    string       `json:"definition_digest"`
}

type Plan struct {
	SchemaVersion  string              `json:"schema_version"`
	RepositoryID   string              `json:"repository_id"`
	SnapshotID     string              `json:"snapshot_id"`
	ConfigDigest   string              `json:"config_digest"`
	BinaryDigest   string              `json:"binary_digest"`
	Gates          []ExpandedGate      `json:"gates"`
	PlanDigest     string              `json:"plan_digest"`
	TrustCandidate policy.TrustBinding `json:"trust_candidate"`
}

type PlanRequest struct {
	RepositoryID     string
	RepositoryRoot   string
	Seal             snapshot.Seal
	ConfigDigest     string
	BinaryDigest     string
	Gates            []config.Gate
	LookupExecutable func(string) (string, error)
	Getenv           func(string) (string, bool)
}

func BuildPlan(request PlanRequest) (Plan, error) {
	if request.RepositoryID == "" || request.Seal.SnapshotID == "" || request.ConfigDigest == "" || request.BinaryDigest == "" {
		return Plan{}, errors.New("repository, snapshot, config, and binary digests are required")
	}
	root, err := filepath.Abs(request.RepositoryRoot)
	if err != nil {
		return Plan{}, err
	}
	lookup := request.LookupExecutable
	if lookup == nil {
		lookup = exec.LookPath
	}
	getenv := request.Getenv
	if getenv == nil {
		getenv = os.LookupEnv
	}
	changed := changedPaths(request.Seal)
	selected := map[string]config.Gate{}
	for _, gate := range request.Gates {
		if matchesAny(gate.WhenPaths, changed) {
			selected[gate.ID] = gate
		}
	}
	// A selected leaf pulls in all dependencies, even if a dependency's own
	// when_paths did not match.
	byID := map[string]config.Gate{}
	for _, gate := range request.Gates {
		byID[gate.ID] = gate
	}
	for changedSelection := true; changedSelection; {
		changedSelection = false
		for _, gate := range selected {
			for _, dependency := range gate.DependsOn {
				if _, ok := selected[dependency]; !ok {
					selected[dependency] = byID[dependency]
					changedSelection = true
				}
			}
		}
	}
	order, err := topological(selected)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{SchemaVersion: "1.0", RepositoryID: request.RepositoryID, SnapshotID: request.Seal.SnapshotID, ConfigDigest: request.ConfigDigest, BinaryDigest: request.BinaryDigest, Gates: []ExpandedGate{}}
	for _, id := range order {
		expanded, expandErr := expand(root, selected[id], lookup, getenv)
		if expandErr != nil {
			return Plan{}, fmt.Errorf("expand gate %q: %w", id, expandErr)
		}
		plan.Gates = append(plan.Gates, expanded)
	}
	plan.PlanDigest, err = digest(struct {
		SchemaVersion string         `json:"schema_version"`
		RepositoryID  string         `json:"repository_id"`
		SnapshotID    string         `json:"snapshot_id"`
		ConfigDigest  string         `json:"config_digest"`
		BinaryDigest  string         `json:"binary_digest"`
		Gates         []ExpandedGate `json:"gates"`
	}{plan.SchemaVersion, plan.RepositoryID, plan.SnapshotID, plan.ConfigDigest, plan.BinaryDigest, plan.Gates})
	if err != nil {
		return Plan{}, err
	}
	plan.TrustCandidate = policy.TrustBinding{RepositoryID: plan.RepositoryID, SnapshotID: plan.SnapshotID, TaskInputDigest: "gate-dag", ExecutionPlanDigest: plan.PlanDigest, ConfigDigest: plan.ConfigDigest, BinaryDigest: plan.BinaryDigest, Capability: "gate:run"}
	return plan, nil
}

func expand(root string, gate config.Gate, lookup func(string) (string, error), getenv func(string) (string, bool)) (ExpandedGate, error) {
	cwd, class, err := resolvePath(root, gate.Cwd)
	if err != nil {
		return ExpandedGate{}, err
	}
	requested := gate.Argv[0]
	executable := requested
	if strings.ContainsAny(requested, `/\\`) {
		if !filepath.IsAbs(requested) {
			executable = filepath.Join(cwd, requested)
		}
	} else {
		executable, err = lookup(requested)
		if err != nil {
			return ExpandedGate{}, fmt.Errorf("resolve executable: %w", err)
		}
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return ExpandedGate{}, err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return ExpandedGate{}, fmt.Errorf("resolve executable symlinks: %w", err)
	}
	executableClass := classify(root, executable)
	executableDigest, err := fileDigest(executable)
	if err != nil {
		return ExpandedGate{}, fmt.Errorf("digest executable: %w", err)
	}
	env := make([]EnvBinding, 0, len(gate.EnvAllowlist))
	seenEnv := map[string]bool{}
	for _, name := range gate.EnvAllowlist {
		if name == "" || strings.Contains(name, "=") || seenEnv[name] {
			return ExpandedGate{}, fmt.Errorf("invalid or duplicate environment name %q", name)
		}
		seenEnv[name] = true
		value, present := getenv(name)
		sum := sha256.Sum256([]byte(value))
		env = append(env, EnvBinding{Name: name, ValueDigest: "sha256:" + hex.EncodeToString(sum[:]), Present: present})
	}
	sort.Slice(env, func(i, j int) bool { return env[i].Name < env[j].Name })
	expanded := ExpandedGate{ID: gate.ID, RequestedExecutable: requested, Executable: executable, ExecutableDigest: executableDigest, ExecutableClass: executableClass, Argv: append([]string(nil), gate.Argv...), Cwd: cwd, CwdClass: class, Environment: env, WhenPaths: sorted(gate.WhenPaths), DependsOn: sorted(gate.DependsOn), TimeoutSeconds: gate.TimeoutSeconds, ResourceClass: gate.ResourceClass, CacheClass: gate.CacheClass, Blocking: gate.Blocking, FinalAlways: gate.FinalAlways, NetworkClass: gate.NetworkClass, ExpectedWrites: sorted(gate.ExpectedWrites), RedactionPolicy: gate.RedactionPolicy, ShellMode: isShellMode(executable, gate.Argv)}
	expanded.DefinitionDigest, err = digest(struct {
		ID, RequestedExecutable, Executable, ExecutableDigest string
		Argv                                                  []string
		Cwd                                                   string
		Environment                                           []EnvBinding
		DependsOn                                             []string
		TimeoutSeconds                                        int
		ResourceClass, CacheClass                             string
		Blocking, FinalAlways                                 bool
		NetworkClass                                          string
		ExpectedWrites                                        []string
		RedactionPolicy                                       string
		ShellMode                                             bool
	}{expanded.ID, expanded.RequestedExecutable, expanded.Executable, expanded.ExecutableDigest, expanded.Argv, expanded.Cwd, expanded.Environment, expanded.DependsOn, expanded.TimeoutSeconds, expanded.ResourceClass, expanded.CacheClass, expanded.Blocking, expanded.FinalAlways, expanded.NetworkClass, expanded.ExpectedWrites, expanded.RedactionPolicy, expanded.ShellMode})
	return expanded, err
}

func isShellMode(executable string, argv []string) bool {
	name := strings.ToLower(filepath.Base(executable))
	if name != "sh" && name != "bash" && name != "zsh" && name != "dash" && name != "cmd.exe" && name != "powershell.exe" && name != "pwsh" {
		return false
	}
	for _, argument := range argv[1:] {
		if argument == "-c" || strings.EqualFold(argument, "/c") || strings.EqualFold(argument, "-command") {
			return true
		}
	}
	return false
}

func resolvePath(root, value string) (string, string, error) {
	path := value
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", "", fmt.Errorf("resolve cwd: %w", err)
	}
	return resolved, classify(root, resolved), nil
}

func classify(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "repository"
	}
	return "external"
}

func fileDigest(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
func digest(value any) (string, error) {
	content, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
func sorted(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	if out == nil {
		return []string{}
	}
	return out
}

func changedPaths(seal snapshot.Seal) []string {
	paths := []string{}
	for _, entry := range seal.Inventory.Entries {
		raw, err := base64.StdEncoding.DecodeString(entry.Path.BytesBase64)
		if err == nil {
			paths = append(paths, filepath.ToSlash(string(raw)))
		}
	}
	return paths
}

func matchesAny(patterns, paths []string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		re, err := globRegexp(pattern)
		if err != nil {
			continue
		}
		for _, path := range paths {
			if re.MatchString(path) {
				return true
			}
		}
	}
	return false
}

func globRegexp(pattern string) (*regexp.Regexp, error) {
	pattern = filepath.ToSlash(pattern)
	var out strings.Builder
	out.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					out.WriteString("(?:.*/)?")
					i += 2
				} else {
					out.WriteString(".*")
					i++
				}
			} else {
				out.WriteString("[^/]*")
			}
		case '?':
			out.WriteString("[^/]")
		default:
			out.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	out.WriteString("$")
	return regexp.Compile(out.String())
}

func topological(gates map[string]config.Gate) ([]string, error) {
	state := map[string]uint8{}
	order := []string{}
	ids := make([]string, 0, len(gates))
	for id := range gates {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 1 {
			return fmt.Errorf("gate dependency cycle at %q", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		deps := sorted(gates[id].DependsOn)
		for _, dep := range deps {
			if _, ok := gates[dep]; !ok {
				return fmt.Errorf("gate %q depends on unavailable %q", id, dep)
			}
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[id] = 2
		order = append(order, id)
		return nil
	}
	for _, id := range ids {
		if err := visit(id); err != nil {
			return nil, err
		}
	}
	return order, nil
}
