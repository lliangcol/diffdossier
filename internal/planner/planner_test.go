package planner

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/lliangcol/diffdossier/internal/contracts"
	"github.com/lliangcol/diffdossier/internal/inventory"
	"github.com/lliangcol/diffdossier/internal/risk"
)

func TestBuildIsDeterministicAndCoversEachPath(t *testing.T) {
	paths := []string{"api/one.go", "api/two.go", "docs/readme.md"}
	entries := make([]inventory.Entry, 0, len(paths))
	for _, path := range paths {
		value := path
		entries = append(entries, inventory.Entry{
			Scope: inventory.ScopeCommitted, Kind: "regular", Size: 10,
			Path: inventory.PathIdentity{BytesBase64: encode(path), UTF8: &value},
		})
	}
	graph := contracts.Build(entries, nil)
	assessment, _ := risk.Assess(entries, graph, nil)
	one := Build("snap-a", entries, graph, assessment, Limits{MaxFiles: 1, MaxPacketBytes: 100})
	two := Build("snap-a", entries, graph, assessment, Limits{MaxFiles: 1, MaxPacketBytes: 100})
	if len(one.Tasks) != 3 || one.Tasks[0].ID != two.Tasks[0].ID {
		t.Fatalf("plans not deterministic or budgeted: %+v %+v", one, two)
	}
	for _, count := range one.Coverage {
		if count != 1 {
			t.Fatalf("path assigned %d times", count)
		}
	}
}

func TestEmptyPlanUsesArraysNotNull(t *testing.T) {
	plan := Build("snap-a", nil, contracts.Graph{}, risk.Assessment{}, Limits{})
	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "\"tasks\":null") {
		t.Fatalf("empty tasks must be an array: %s", encoded)
	}
}

func TestOversizedSingleFileIsNotTruncated(t *testing.T) {
	path := "large.go"
	entry := inventory.Entry{
		Scope: inventory.ScopeUntracked, Kind: "regular", Size: 1000,
		Path: inventory.PathIdentity{BytesBase64: encode(path), UTF8: &path},
	}
	graph := contracts.Build([]inventory.Entry{entry}, nil)
	assessment, _ := risk.Assess([]inventory.Entry{entry}, graph, nil)
	plan := Build("snap-a", []inventory.Entry{entry}, graph, assessment, Limits{MaxFiles: 8, MaxPacketBytes: 100})
	if len(plan.Tasks) != 1 || !plan.Tasks[0].Oversized || plan.Tasks[0].Paths[0].Bytes != 1000 {
		t.Fatalf("large input was lost or silently truncated: %+v", plan)
	}
}

func encode(value string) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	data := []byte(value)
	result := ""
	for len(data) >= 3 {
		value := uint(data[0])<<16 | uint(data[1])<<8 | uint(data[2])
		result += string([]byte{alphabet[value>>18], alphabet[(value>>12)&63], alphabet[(value>>6)&63], alphabet[value&63]})
		data = data[3:]
	}
	if len(data) == 1 {
		value := uint(data[0]) << 16
		result += string([]byte{alphabet[value>>18], alphabet[(value>>12)&63], '=', '='})
	} else if len(data) == 2 {
		value := uint(data[0])<<16 | uint(data[1])<<8
		result += string([]byte{alphabet[value>>18], alphabet[(value>>12)&63], alphabet[(value>>6)&63], '='})
	}
	return result
}
