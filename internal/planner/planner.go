// Package planner creates deterministic, budgeted review task graphs.
package planner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"github.com/lliangcol/diffdossier/internal/contracts"
	"github.com/lliangcol/diffdossier/internal/inventory"
	"github.com/lliangcol/diffdossier/internal/risk"
)

type PathRef struct {
	Scope            inventory.Scope `json:"scope"`
	PathBytesBase64  string          `json:"path_bytes_base64"`
	DisplayPath      string          `json:"display_path"`
	CurrentHash      string          `json:"current_hash,omitempty"`
	PreviousHash     string          `json:"previous_hash,omitempty"`
	Bytes            int64           `json:"bytes"`
	RequiredCoverage string          `json:"required_coverage"`
}

type Task struct {
	SchemaVersion     string     `json:"schema_version"`
	ID                string     `json:"task_id"`
	SnapshotID        string     `json:"snapshot_id"`
	Risk              risk.Level `json:"risk"`
	ContractTypes     []string   `json:"contract_types"`
	Paths             []PathRef  `json:"paths"`
	DependencyTasks   []string   `json:"dependency_tasks"`
	TotalBytes        int64      `json:"total_bytes"`
	Oversized         bool       `json:"oversized"`
	NeedsConfirmation bool       `json:"needs_confirmation"`
	RequiredPasses    int        `json:"required_passes"`
	Perspectives      []string   `json:"perspectives"`
}

type Plan struct {
	SchemaVersion string         `json:"schema_version"`
	SnapshotID    string         `json:"snapshot_id"`
	Tasks         []Task         `json:"tasks"`
	Coverage      map[string]int `json:"coverage_counts"`
}

type Limits struct {
	MaxFiles       int
	MaxPacketBytes int64
}

func Build(snapshotID string, entries []inventory.Entry, graph contracts.Graph, assessment risk.Assessment, limits Limits) Plan {
	if limits.MaxFiles < 1 {
		limits.MaxFiles = 8
	}
	if limits.MaxPacketBytes < 1 {
		limits.MaxPacketBytes = 250000
	}
	riskByPath := map[string]risk.PathRisk{}
	for _, item := range assessment.Paths {
		riskByPath[item.PathBytesBase64] = item
	}
	contractsByPath := map[string][]string{}
	for _, contract := range graph.Contracts {
		contractsByPath[contract.PathBytesBase64] = append(contractsByPath[contract.PathBytesBase64], contract.Type)
	}
	groups := map[string][]inventory.Entry{}
	for _, entry := range entries {
		contractTypes := unique(contractsByPath[entry.Path.BytesBase64])
		group := string(riskByPath[entry.Path.BytesBase64].Level) + "|" + strings.Join(contractTypes, ",")
		if len(contractTypes) == 0 {
			path := filepathGroup(entry.Path.Display())
			group += "|" + path
		}
		groups[group] = append(groups[group], entry)
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	plan := Plan{SchemaVersion: "1.0", SnapshotID: snapshotID, Tasks: []Task{}, Coverage: map[string]int{}}
	for _, key := range keys {
		items := groups[key]
		sort.Slice(items, func(i, j int) bool {
			if items[i].Path.BytesBase64 != items[j].Path.BytesBase64 {
				return items[i].Path.BytesBase64 < items[j].Path.BytesBase64
			}
			return items[i].Scope < items[j].Scope
		})
		for len(items) > 0 {
			count, bytes := sliceSize(items, limits)
			plan.Tasks = append(plan.Tasks, makeTask(snapshotID, items[:count], riskByPath, contractsByPath, bytes, limits.MaxPacketBytes))
			items = items[count:]
		}
	}
	for index := range plan.Tasks {
		seenPaths := map[string]bool{}
		for _, path := range plan.Tasks[index].Paths {
			if !seenPaths[path.PathBytesBase64] {
				plan.Coverage[path.PathBytesBase64]++
				seenPaths[path.PathBytesBase64] = true
			}
		}
	}
	linkDependencies(plan.Tasks)
	return plan
}

func sliceSize(entries []inventory.Entry, limits Limits) (int, int64) {
	count := 0
	var total int64
	seenPaths := map[string]bool{}
	for index, entry := range entries {
		bytes := entry.Size + entry.PreviousSize
		newPath := !seenPaths[entry.Path.BytesBase64]
		if index > 0 && newPath && (len(seenPaths) >= limits.MaxFiles || total+bytes > limits.MaxPacketBytes) {
			break
		}
		seenPaths[entry.Path.BytesBase64] = true
		total += bytes
		count++
	}
	if count == 0 {
		return 1, entries[0].Size + entries[0].PreviousSize
	}
	return count, total
}

func makeTask(snapshotID string, entries []inventory.Entry, riskByPath map[string]risk.PathRisk, contractsByPath map[string][]string, total int64, limit int64) Task {
	task := Task{
		SchemaVersion: "1.0", SnapshotID: snapshotID, Risk: risk.L0,
		TotalBytes: total, Oversized: total > limit, RequiredPasses: 1,
		Perspectives: []string{"correctness"}, ContractTypes: []string{},
		DependencyTasks: []string{}, Paths: []PathRef{},
	}
	contractSet := map[string]bool{}
	for _, entry := range entries {
		pathRisk := riskByPath[entry.Path.BytesBase64]
		task.Risk = risk.Max(task.Risk, pathRisk.Level)
		task.NeedsConfirmation = task.NeedsConfirmation || pathRisk.NeedsConfirmation
		for _, contractType := range contractsByPath[entry.Path.BytesBase64] {
			contractSet[contractType] = true
		}
		coverage := "fully_reviewed"
		if pathRisk.Level == risk.L0 {
			coverage = "mechanically_verified"
		}
		if entry.Kind == "binary" || entry.Kind == "submodule" || entry.Kind == "lfs_pointer" {
			coverage = "fully_reviewed"
			task.NeedsConfirmation = true
		}
		task.Paths = append(task.Paths, PathRef{
			Scope: entry.Scope, PathBytesBase64: entry.Path.BytesBase64, DisplayPath: entry.Path.Display(),
			CurrentHash: entry.ContentHash, PreviousHash: entry.PreviousContentHash,
			Bytes: entry.Size + entry.PreviousSize, RequiredCoverage: coverage,
		})
	}
	task.NeedsConfirmation = task.NeedsConfirmation || risk.RequiresOwner(task.Risk)
	if risk.RequiresOwner(task.Risk) {
		task.RequiredPasses = 2
		task.Perspectives = []string{"correctness", "failure-recovery"}
	}
	for contractType := range contractSet {
		task.ContractTypes = append(task.ContractTypes, contractType)
	}
	sort.Strings(task.ContractTypes)
	encoded, _ := json.Marshal(struct {
		SnapshotID string     `json:"snapshot_id"`
		Risk       risk.Level `json:"risk"`
		Contracts  []string   `json:"contracts"`
		Paths      []PathRef  `json:"paths"`
	}{snapshotID, task.Risk, task.ContractTypes, task.Paths})
	digest := sha256.Sum256(encoded)
	task.ID = "task-" + hex.EncodeToString(digest[:12])
	return task
}

func linkDependencies(tasks []Task) {
	for left := range tasks {
		for right := 0; right < left; right++ {
			if !overlap(tasks[left].ContractTypes, tasks[right].ContractTypes) {
				continue
			}
			tasks[left].DependencyTasks = append(tasks[left].DependencyTasks, tasks[right].ID)
		}
		sort.Strings(tasks[left].DependencyTasks)
	}
}

func overlap(left, right []string) bool {
	for _, l := range left {
		for _, r := range right {
			if l == r {
				return true
			}
		}
	}
	return false
}

func filepathGroup(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	if index := strings.IndexByte(path, '/'); index >= 0 {
		return path[:index]
	}
	return "."
}

func unique(values []string) []string {
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
