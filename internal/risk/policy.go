package risk

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func LoadOverrides(root string, policyFiles []string) ([]Override, error) {
	var result []Override
	for _, relative := range policyFiles {
		path, err := ResolvePolicyPath(root, relative)
		if err != nil {
			return nil, fmt.Errorf("load risk policy %q: %w", relative, err)
		}
		content, err := parsePolicy(path)
		if err != nil {
			return nil, fmt.Errorf("load risk policy %q: %w", relative, err)
		}
		result = append(result, content...)
	}
	return result, nil
}

func ResolvePolicyPath(root, relative string) (string, error) {
	clean := filepath.Clean(relative)
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("risk policy escapes repository: %q", relative)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(root, clean))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("risk policy symlink escapes repository: %q", relative)
	}
	return resolved, nil
}

func parsePolicy(path string) ([]Override, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	schemaVersion := 0
	var rules []Override
	current := -1
	seen := map[string]bool{}
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		if line == "[[rules]]" {
			rules = append(rules, Override{})
			current = len(rules) - 1
			continue
		}
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected key = value", lineNumber)
		}
		key, raw = strings.TrimSpace(key), strings.TrimSpace(raw)
		fieldID := fmt.Sprintf("%d/%s", current, key)
		if seen[fieldID] {
			return nil, fmt.Errorf("line %d: duplicate field %s", lineNumber, key)
		}
		seen[fieldID] = true
		if current < 0 {
			if key != "schema_version" {
				return nil, fmt.Errorf("line %d: unknown root field %s", lineNumber, key)
			}
			value, err := strconv.Atoi(raw)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid schema_version", lineNumber)
			}
			schemaVersion = value
			continue
		}
		value, err := strconv.Unquote(raw)
		if err != nil {
			return nil, fmt.Errorf("line %d: expected quoted string", lineNumber)
		}
		switch key {
		case "glob":
			rules[current].Glob = value
		case "level":
			rules[current].Level = Level(value)
		case "reason":
			rules[current].Reason = value
		default:
			return nil, fmt.Errorf("line %d: unknown rule field %s", lineNumber, key)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if schemaVersion != 1 {
		return nil, fmt.Errorf("unsupported schema_version %d", schemaVersion)
	}
	for index, rule := range rules {
		if rule.Glob == "" || rule.Reason == "" {
			return nil, fmt.Errorf("rule %d requires glob and reason", index+1)
		}
		if _, ok := rank[rule.Level]; !ok {
			return nil, fmt.Errorf("rule %d has invalid level %q", index+1, rule.Level)
		}
		if _, err := filepath.Match(rule.Glob, "probe"); err != nil {
			return nil, fmt.Errorf("rule %d: %w", index+1, err)
		}
	}
	if len(rules) == 0 {
		return nil, errors.New("risk policy requires at least one rule")
	}
	return rules, nil
}
