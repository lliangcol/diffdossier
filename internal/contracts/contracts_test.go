package contracts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lliangcol/diffdossier/internal/inventory"
)

func TestDiscoverRulesExcludesDependencies(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "AGENTS.md"), "root")
	write(t, filepath.Join(root, "service", "CLAUDE.md"), "local")
	write(t, filepath.Join(root, "node_modules", "AGENTS.md"), "dependency")
	rules, err := DiscoverRules(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 || rules[0].Digest == "" {
		t.Fatalf("rules=%+v", rules)
	}
}

func TestBuildMarksHeuristicContractsCandidateOnly(t *testing.T) {
	path := "api/payment/schema.json"
	entries := []inventory.Entry{{Path: inventory.PathIdentity{BytesBase64: "YXBpL3BheW1lbnQvc2NoZW1hLmpzb24=", UTF8: &path}}}
	graph := Build(entries, nil)
	if len(graph.Contracts) < 3 {
		t.Fatalf("expected multiple contract edges: %+v", graph.Contracts)
	}
	for _, contract := range graph.Contracts {
		if !contract.CandidateOnly {
			t.Fatal("path heuristic must not be promoted to verified contract fact")
		}
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
