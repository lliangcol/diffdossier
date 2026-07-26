// Package risk assigns fail-closed review depth without lowering explicit policy.
package risk

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lliangcol/diffdossier/internal/contracts"
	"github.com/lliangcol/diffdossier/internal/inventory"
)

type Level string

const (
	L0 Level = "L0"
	L1 Level = "L1"
	L2 Level = "L2"
	L3 Level = "L3"
	L4 Level = "L4"
)

var rank = map[Level]int{L0: 0, L1: 1, L2: 2, L3: 3, L4: 4}

type Override struct {
	Glob   string `json:"glob"`
	Level  Level  `json:"level"`
	Reason string `json:"reason"`
}

type PathRisk struct {
	PathBytesBase64   string   `json:"path_bytes_base64"`
	Level             Level    `json:"level"`
	Reasons           []string `json:"reasons"`
	NeedsConfirmation bool     `json:"needs_confirmation"`
}

type Assessment struct {
	SchemaVersion string     `json:"schema_version"`
	Paths         []PathRisk `json:"paths"`
}

func Assess(entries []inventory.Entry, graph contracts.Graph, overrides []Override) (Assessment, error) {
	contractByPath := map[string][]string{}
	for _, contract := range graph.Contracts {
		contractByPath[contract.PathBytesBase64] = append(contractByPath[contract.PathBytesBase64], contract.Type)
	}
	result := Assessment{SchemaVersion: "1.0", Paths: []PathRisk{}}
	seen := map[string]bool{}
	for _, entry := range entries {
		if seen[entry.Path.BytesBase64] {
			continue
		}
		seen[entry.Path.BytesBase64] = true
		path := entry.Path.Display()
		level, reasons, known := inferred(path, entry, contractByPath[entry.Path.BytesBase64])
		for _, override := range overrides {
			matched, err := filepath.Match(override.Glob, filepath.ToSlash(path))
			if err != nil {
				return Assessment{}, fmt.Errorf("invalid risk glob %q: %w", override.Glob, err)
			}
			if matched {
				if rank[override.Level] > rank[level] {
					level = override.Level
				}
				reasons = append(reasons, "policy: "+override.Reason)
				known = true
			}
		}
		result.Paths = append(result.Paths, PathRisk{
			PathBytesBase64: entry.Path.BytesBase64, Level: level,
			Reasons: uniqueSorted(reasons), NeedsConfirmation: !known,
		})
	}
	sort.Slice(result.Paths, func(i, j int) bool { return result.Paths[i].PathBytesBase64 < result.Paths[j].PathBytesBase64 })
	return result, nil
}

func inferred(path string, entry inventory.Entry, contractTypes []string) (Level, []string, bool) {
	lower := strings.ToLower(filepath.ToSlash(path))
	if containsAny(lower, "payment", "refund", "billing", "migration", "/prod", "delete") {
		return L4, []string{"high-impact path signal"}, true
	}
	if containsAny(lower, "auth", "permission", "secret", "webhook", "external", "security") {
		return L3, []string{"security or external-boundary signal"}, true
	}
	for _, contractType := range contractTypes {
		switch contractType {
		case "payment_state":
			return L4, []string{"payment contract candidate"}, true
		case "message_external":
			return L3, []string{"message/external contract candidate"}, true
		case "database_sql", "api_schema", "configuration", "cli_protocol", "delivery":
			return L2, []string{"persistent contract candidate: " + contractType}, true
		}
	}
	if entry.Kind == "binary" || entry.Kind == "submodule" || entry.Kind == "lfs_pointer" {
		return L3, []string{"non-text artifact requires specialized review"}, true
	}
	if containsAny(lower, ".generated.", ".lock", "/generated/") {
		return L0, []string{"generated or lock metadata"}, true
	}
	if containsAny(lower, "_test.", "/test/", "/tests/", ".md") {
		return L1, []string{"test or documentation path"}, true
	}
	if containsAny(lower, ".go", ".rs", ".java", ".kt", ".py", ".js", ".ts", ".c", ".h", ".cpp") {
		return L2, []string{"source file"}, true
	}
	return L2, []string{"unclassified path fails closed"}, false
}

func Max(left, right Level) Level {
	if rank[right] > rank[left] {
		return right
	}
	return left
}

func RequiresOwner(level Level) bool {
	return rank[level] >= rank[L3]
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		seen[value] = true
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}
