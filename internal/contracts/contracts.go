// Package contracts discovers scoped project rules and candidate contract edges.
package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lliangcol/diffdossier/internal/inventory"
)

type Rule struct {
	Path   string `json:"path"`
	Scope  string `json:"scope"`
	Digest string `json:"digest"`
}

type Contract struct {
	ID              string   `json:"id"`
	Type            string   `json:"type"`
	PathBytesBase64 string   `json:"path_bytes_base64"`
	Signals         []string `json:"signals"`
	CandidateOnly   bool     `json:"candidate_only"`
}

type Graph struct {
	SchemaVersion string     `json:"schema_version"`
	Rules         []Rule     `json:"rules"`
	Contracts     []Contract `json:"contracts"`
}

var excludedDirectories = map[string]bool{
	".git": true, "node_modules": true, ".venv": true, "vendor": true,
}

func DiscoverRules(root string) ([]Rule, error) {
	rules := []Rule{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && excludedDirectories[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || (entry.Name() != "AGENTS.md" && entry.Name() != "CLAUDE.md") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(content) > 1024*1024 {
			return fmt.Errorf("rule file exceeds 1 MiB: %s", path)
		}
		digest := sha256.Sum256(content)
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		scope := filepath.ToSlash(filepath.Dir(relative))
		if scope == "." {
			scope = ""
		}
		rules = append(rules, Rule{
			Path: filepath.ToSlash(relative), Scope: scope,
			Digest: "sha256:" + hex.EncodeToString(digest[:]),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(rules, func(i, j int) bool { return rules[i].Path < rules[j].Path })
	return rules, nil
}

func Build(entries []inventory.Entry, rules []Rule) Graph {
	contracts := make([]Contract, 0, len(entries))
	for _, entry := range entries {
		path := strings.ToLower(entry.Path.Display())
		types, signals := classify(path)
		for index, contractType := range types {
			digest := sha256.Sum256([]byte(contractType + "\x00" + entry.Path.BytesBase64))
			contracts = append(contracts, Contract{
				ID: "contract-" + hex.EncodeToString(digest[:8]), Type: contractType,
				PathBytesBase64: entry.Path.BytesBase64, Signals: signals[index], CandidateOnly: true,
			})
		}
	}
	sort.Slice(contracts, func(i, j int) bool {
		if contracts[i].Type != contracts[j].Type {
			return contracts[i].Type < contracts[j].Type
		}
		return contracts[i].PathBytesBase64 < contracts[j].PathBytesBase64
	})
	return Graph{SchemaVersion: "1.0", Rules: append([]Rule{}, rules...), Contracts: contracts}
}

func classify(path string) ([]string, [][]string) {
	types := []string{}
	signals := [][]string{}
	add := func(contractType string, found ...string) {
		types = append(types, contractType)
		signals = append(signals, found)
	}
	switch {
	case strings.HasSuffix(path, ".sql") || strings.Contains(path, "/mapper/"):
		add("database_sql", "path")
	}
	if containsAny(path, "/api/", "/controller/", "/dto/", "schema") {
		add("api_schema", "path")
	}
	if containsAny(path, "payment", "refund", "billing", "entitlement") {
		add("payment_state", "path")
	}
	if containsAny(path, "/mq/", "kafka", "rocketmq", "webhook") {
		add("message_external", "path")
	}
	if containsAny(path, ".github/workflows", "dockerfile", "/deploy/", "rollback") {
		add("delivery", "path")
	}
	if containsAny(path, ".toml", ".yaml", ".yml", ".json", "/config/") {
		add("configuration", "path")
	}
	if containsAny(path, "/cmd/", "/cli/", "schemas/") {
		add("cli_protocol", "path")
	}
	return types, signals
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}
