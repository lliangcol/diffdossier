// Package config loads the versioned, deliberately small DiffDossier TOML surface.
package config

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const CurrentSchemaVersion = 1

// MinimalDocument returns a deterministic, command-free starter configuration.
// The caller must supply the baseline explicitly because DiffDossier never
// guesses a branch name.
func MinimalDocument(baseline string) ([]byte, error) {
	if strings.TrimSpace(baseline) == "" {
		return nil, errors.New("baseline is required; DiffDossier does not guess main or master")
	}
	document := fmt.Sprintf(
		"schema_version = %d\n"+
			"baseline = %s\n"+
			"include_untracked = true\n"+
			"include_ignored = []\n\n"+
			"[review]\n"+
			"max_files_per_task = 8\n"+
			"max_packet_bytes = 250000\n"+
			"default_provider = \"manual\"\n\n"+
			"[state]\n"+
			"retention_days = 30\n\n"+
			"[risk]\n"+
			"policy_files = []\n",
		CurrentSchemaVersion,
		strconv.Quote(baseline),
	)
	return []byte(document), nil
}

type Config struct {
	SchemaVersion    int
	Baseline         string
	IncludeUntracked bool
	IncludeIgnored   []string
	Review           Review
	State            State
	Risk             Risk
	Gates            []Gate
}

// Source records a configuration layer without exposing its contents.
type Source struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

// Effective is the validated, deterministic result of applying configuration layers.
type Effective struct {
	Config  Config   `json:"config"`
	Sources []Source `json:"sources"`
	Digest  string   `json:"digest"`
}

type Review struct {
	MaxFilesPerTask int
	MaxPacketBytes  int
	DefaultProvider string
}

type State struct {
	RetentionDays int
}

type Risk struct {
	PolicyFiles []string
}

type Gate struct {
	ID              string
	Argv            []string
	Cwd             string
	EnvAllowlist    []string
	WhenPaths       []string
	DependsOn       []string
	TimeoutSeconds  int
	ResourceClass   string
	Blocking        bool
	CacheClass      string
	FinalAlways     bool
	NetworkClass    string
	ExpectedWrites  []string
	RedactionPolicy string
}

func Default() Config {
	return Config{
		SchemaVersion:    CurrentSchemaVersion,
		IncludeUntracked: true,
		Review:           Review{MaxFilesPerTask: 8, MaxPacketBytes: 250000, DefaultProvider: "manual"},
		State:            State{RetentionDays: 30},
	}
}

func Load(path string) (Config, error) {
	effective, err := LoadExact(path)
	return effective.Config, err
}

// LoadExact loads one required configuration file over built-in defaults.
// It is used for an explicit --config or DIFFDOSSIER_CONFIG selection.
func LoadExact(path string) (Effective, error) {
	return LoadExactWithBaseline(path, "")
}

// LoadExactWithBaseline applies an optional exact CLI baseline after parsing.
func LoadExactWithBaseline(path, baseline string) (Effective, error) {
	cfg, source, err := loadLayer(Default(), "explicit", path)
	if err != nil {
		return Effective{}, err
	}
	sources := []Source{source}
	applyBaselineOverride(&cfg, &sources, baseline)
	return finish(cfg, sources)
}

// LoadEffective applies built-in defaults, an optional user layer, then the
// required repository layer. Arrays in a higher layer replace lower arrays.
func LoadEffective(userPath, repositoryPath string) (Effective, error) {
	return LoadEffectiveWithBaseline(userPath, repositoryPath, "")
}

// LoadEffectiveWithBaseline applies an optional exact CLI baseline after all files.
func LoadEffectiveWithBaseline(userPath, repositoryPath, baseline string) (Effective, error) {
	cfg := Default()
	sources := []Source{}
	if userPath != "" {
		loaded, source, err := loadLayer(cfg, "user", userPath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return Effective{}, err
			}
		} else {
			cfg = loaded
			sources = append(sources, source)
		}
	}
	var source Source
	var err error
	cfg, source, err = loadLayer(cfg, "repository", repositoryPath)
	if err != nil {
		return Effective{}, err
	}
	sources = append(sources, source)
	applyBaselineOverride(&cfg, &sources, baseline)
	return finish(cfg, sources)
}

func loadLayer(cfg Config, kind, path string) (Config, Source, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, Source{}, fmt.Errorf("open config %s: %w", path, err)
	}
	defer file.Close()

	hasher := sha256.New()
	scanner := bufio.NewScanner(io.TeeReader(file, hasher))
	scanner.Split(bufio.ScanLines)
	section := ""
	gateIndex := -1
	gatesReplaced := false
	seenFields := map[string]bool{}
	seenSections := map[string]bool{}
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[[") {
			if line != "[[gates]]" {
				return Config{}, Source{}, lineError(lineNumber, "unknown array section")
			}
			if !gatesReplaced {
				cfg.Gates = nil
				gatesReplaced = true
			}
			cfg.Gates = append(cfg.Gates, Gate{})
			gateIndex = len(cfg.Gates) - 1
			section = "gates"
			continue
		}
		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return Config{}, Source{}, lineError(lineNumber, "unterminated section")
			}
			section = strings.TrimSpace(line[1 : len(line)-1])
			if section != "review" && section != "state" && section != "risk" {
				return Config{}, Source{}, lineError(lineNumber, "unknown section "+section)
			}
			if seenSections[section] {
				return Config{}, Source{}, lineError(lineNumber, "duplicate section "+section)
			}
			seenSections[section] = true
			gateIndex = -1
			continue
		}
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			return Config{}, Source{}, lineError(lineNumber, "expected key = value")
		}
		key = strings.TrimSpace(key)
		fieldID := fmt.Sprintf("%s/%d/%s", section, gateIndex, key)
		if seenFields[fieldID] {
			return Config{}, Source{}, lineError(lineNumber, "duplicate field "+key)
		}
		seenFields[fieldID] = true
		if err := assign(&cfg, section, gateIndex, key, strings.TrimSpace(raw)); err != nil {
			return Config{}, Source{}, lineError(lineNumber, err.Error())
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, Source{}, fmt.Errorf("read config %s: %w", path, err)
	}
	validatedLayer := cfg
	if strings.TrimSpace(validatedLayer.Baseline) == "" {
		validatedLayer.Baseline = "pending-higher-precedence-baseline"
	}
	if err := validatedLayer.Validate(); err != nil {
		return Config{}, Source{}, fmt.Errorf("validate config %s: %w", path, err)
	}
	return cfg, Source{Kind: kind, Path: path, Digest: "sha256:" + hex.EncodeToString(hasher.Sum(nil))}, nil
}

func applyBaselineOverride(cfg *Config, sources *[]Source, baseline string) {
	if strings.TrimSpace(baseline) == "" {
		return
	}
	cfg.Baseline = baseline
	digest := sha256.Sum256([]byte(baseline))
	*sources = append(*sources, Source{Kind: "cli", Path: "baseline", Digest: "sha256:" + hex.EncodeToString(digest[:])})
}

func finish(cfg Config, sources []Source) (Effective, error) {
	if err := cfg.Validate(); err != nil {
		return Effective{}, err
	}
	encoded, err := json.Marshal(cfg)
	if err != nil {
		return Effective{}, fmt.Errorf("encode effective config: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return Effective{
		Config: cfg, Sources: append([]Source{}, sources...),
		Digest: "sha256:" + hex.EncodeToString(digest[:]),
	}, nil
}

func (c Config) Validate() error {
	if c.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("unsupported schema_version %d; only %d is accepted and no migration is required yet", c.SchemaVersion, CurrentSchemaVersion)
	}
	if strings.TrimSpace(c.Baseline) == "" {
		return errors.New("baseline is required; DiffDossier does not guess main or master")
	}
	if c.Review.MaxFilesPerTask < 1 || c.Review.MaxPacketBytes < 1 {
		return errors.New("review limits must be positive")
	}
	if c.Review.DefaultProvider == "" {
		return errors.New("review.default_provider is required")
	}
	if c.State.RetentionDays < 1 {
		return errors.New("state.retention_days must be positive")
	}
	for _, path := range c.IncludeIgnored {
		clean := strings.ReplaceAll(path, "\\", "/")
		if clean == "" || strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
			return fmt.Errorf("include_ignored path %q must stay repository-relative", path)
		}
	}
	seen := map[string]bool{}
	for _, gate := range c.Gates {
		if gate.ID == "" || len(gate.Argv) == 0 {
			return errors.New("every gate requires id and non-empty argv")
		}
		if seen[gate.ID] {
			return fmt.Errorf("duplicate gate id %q", gate.ID)
		}
		seen[gate.ID] = true
		if gate.TimeoutSeconds < 1 {
			return fmt.Errorf("gate %q timeout_seconds must be positive", gate.ID)
		}
		switch gate.CacheClass {
		case "worktree_deterministic", "host_volatile", "time_network", "external":
		default:
			return fmt.Errorf("gate %q has invalid cache_class %q", gate.ID, gate.CacheClass)
		}
		if gate.Cwd == "" {
			return fmt.Errorf("gate %q cwd is required", gate.ID)
		}
		if gate.ResourceClass != "cpu" && gate.ResourceClass != "io" && gate.ResourceClass != "exclusive" && gate.ResourceClass != "network" && gate.ResourceClass != "external" {
			return fmt.Errorf("gate %q has invalid resource_class %q", gate.ID, gate.ResourceClass)
		}
		if gate.NetworkClass != "none" && gate.NetworkClass != "local" && gate.NetworkClass != "external" {
			return fmt.Errorf("gate %q has invalid network_class %q", gate.ID, gate.NetworkClass)
		}
		for _, relative := range append(append([]string{}, gate.WhenPaths...), gate.ExpectedWrites...) {
			clean := strings.ReplaceAll(relative, "\\", "/")
			if clean == "" || strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
				return fmt.Errorf("gate %q path %q must stay repository-relative", gate.ID, relative)
			}
		}
	}
	for _, gate := range c.Gates {
		for _, dependency := range gate.DependsOn {
			if dependency == gate.ID || !seen[dependency] {
				return fmt.Errorf("gate %q has invalid dependency %q", gate.ID, dependency)
			}
		}
	}
	byID := map[string]Gate{}
	for _, gate := range c.Gates {
		byID[gate.ID] = gate
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("gate dependency cycle at %q", id)
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		for _, dependency := range byID[id].DependsOn {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	for id := range byID {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func assign(cfg *Config, section string, gateIndex int, key, raw string) error {
	qualified := key
	if section != "" {
		qualified = section + "." + key
	}
	switch qualified {
	case "schema_version":
		return parseInt(raw, &cfg.SchemaVersion)
	case "baseline":
		return parseString(raw, &cfg.Baseline)
	case "include_untracked":
		return parseBool(raw, &cfg.IncludeUntracked)
	case "include_ignored":
		return parseStrings(raw, &cfg.IncludeIgnored)
	case "review.max_files_per_task":
		return parseInt(raw, &cfg.Review.MaxFilesPerTask)
	case "review.max_packet_bytes":
		return parseInt(raw, &cfg.Review.MaxPacketBytes)
	case "review.default_provider":
		return parseString(raw, &cfg.Review.DefaultProvider)
	case "state.retention_days":
		return parseInt(raw, &cfg.State.RetentionDays)
	case "risk.policy_files":
		return parseStrings(raw, &cfg.Risk.PolicyFiles)
	}
	if section == "gates" && gateIndex >= 0 {
		gate := &cfg.Gates[gateIndex]
		switch key {
		case "id":
			return parseString(raw, &gate.ID)
		case "argv":
			return parseStrings(raw, &gate.Argv)
		case "cwd":
			return parseString(raw, &gate.Cwd)
		case "env_allowlist":
			return parseStrings(raw, &gate.EnvAllowlist)
		case "when_paths":
			return parseStrings(raw, &gate.WhenPaths)
		case "depends_on":
			return parseStrings(raw, &gate.DependsOn)
		case "timeout_seconds":
			return parseInt(raw, &gate.TimeoutSeconds)
		case "blocking":
			return parseBool(raw, &gate.Blocking)
		case "resource_class":
			return parseString(raw, &gate.ResourceClass)
		case "cache_class":
			return parseString(raw, &gate.CacheClass)
		case "final_always":
			return parseBool(raw, &gate.FinalAlways)
		case "network_class":
			return parseString(raw, &gate.NetworkClass)
		case "expected_writes":
			return parseStrings(raw, &gate.ExpectedWrites)
		case "redaction_policy":
			return parseString(raw, &gate.RedactionPolicy)
		}
	}
	return fmt.Errorf("unknown field %q", qualified)
}

func stripComment(line string) string {
	inString := false
	escaped := false
	for index, char := range line {
		if char == '\\' && inString {
			escaped = !escaped
			continue
		}
		if char == '"' && !escaped {
			inString = !inString
		}
		if char == '#' && !inString {
			return line[:index]
		}
		escaped = false
	}
	return line
}

func parseString(raw string, target *string) error {
	value, err := strconv.Unquote(raw)
	if err != nil {
		return fmt.Errorf("expected quoted string: %w", err)
	}
	*target = value
	return nil
}

func parseInt(raw string, target *int) error {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fmt.Errorf("expected integer: %w", err)
	}
	*target = value
	return nil
}

func parseBool(raw string, target *bool) error {
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fmt.Errorf("expected boolean: %w", err)
	}
	*target = value
	return nil
}

func parseStrings(raw string, target *[]string) error {
	raw = strings.TrimSpace(raw)
	if len(raw) < 2 || raw[0] != '[' || raw[len(raw)-1] != ']' {
		return errors.New("expected string array")
	}
	body := strings.TrimSpace(raw[1 : len(raw)-1])
	if body == "" {
		*target = nil
		return nil
	}
	values := []string{}
	for len(body) > 0 {
		body = strings.TrimSpace(body)
		if body == "" || body[0] != '"' {
			return errors.New("expected quoted string in array")
		}
		end := -1
		escaped := false
		for index := 1; index < len(body); index++ {
			if body[index] == '\\' {
				escaped = !escaped
				continue
			}
			if body[index] == '"' && !escaped {
				end = index
				break
			}
			escaped = false
		}
		if end < 0 {
			return errors.New("unterminated string in array")
		}
		value, err := strconv.Unquote(body[:end+1])
		if err != nil {
			return fmt.Errorf("invalid string in array: %w", err)
		}
		values = append(values, value)
		body = strings.TrimSpace(body[end+1:])
		if body == "" {
			break
		}
		if body[0] != ',' {
			return errors.New("expected comma between array values")
		}
		body = body[1:]
	}
	*target = values
	return nil
}

func lineError(line int, message string) error {
	return fmt.Errorf("config line %d: %s", line, message)
}
