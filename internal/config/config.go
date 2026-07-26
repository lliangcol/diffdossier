// Package config loads the versioned, deliberately small DiffDossier TOML surface.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const CurrentSchemaVersion = 1

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
	ID             string
	Argv           []string
	TimeoutSeconds int
	Blocking       bool
	CacheClass     string
	FinalAlways    bool
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
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	cfg := Default()
	section := ""
	gateIndex := -1
	seenFields := map[string]bool{}
	seenSections := map[string]bool{}
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[[") {
			if line != "[[gates]]" {
				return Config{}, lineError(lineNumber, "unknown array section")
			}
			cfg.Gates = append(cfg.Gates, Gate{})
			gateIndex = len(cfg.Gates) - 1
			section = "gates"
			continue
		}
		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return Config{}, lineError(lineNumber, "unterminated section")
			}
			section = strings.TrimSpace(line[1 : len(line)-1])
			if section != "review" && section != "state" && section != "risk" {
				return Config{}, lineError(lineNumber, "unknown section "+section)
			}
			if seenSections[section] {
				return Config{}, lineError(lineNumber, "duplicate section "+section)
			}
			seenSections[section] = true
			gateIndex = -1
			continue
		}
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			return Config{}, lineError(lineNumber, "expected key = value")
		}
		key = strings.TrimSpace(key)
		fieldID := fmt.Sprintf("%s/%d/%s", section, gateIndex, key)
		if seenFields[fieldID] {
			return Config{}, lineError(lineNumber, "duplicate field "+key)
		}
		seenFields[fieldID] = true
		if err := assign(&cfg, section, gateIndex, key, strings.TrimSpace(raw)); err != nil {
			return Config{}, lineError(lineNumber, err.Error())
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
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
		case "timeout_seconds":
			return parseInt(raw, &gate.TimeoutSeconds)
		case "blocking":
			return parseBool(raw, &gate.Blocking)
		case "cache_class":
			return parseString(raw, &gate.CacheClass)
		case "final_always":
			return parseBool(raw, &gate.FinalAlways)
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
