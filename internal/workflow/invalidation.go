package workflow

import (
	"sort"

	"github.com/lliangcol/diffdossier/internal/planner"
)

type Invalidation struct {
	SchemaVersion string              `json:"schema_version"`
	MustReload    []string            `json:"must_reload"`
	Reasons       map[string][]string `json:"reasons"`
}

func ComputeInvalidation(plan planner.Plan, changedPaths []string, semanticInputsChanged bool) Invalidation {
	result := Invalidation{SchemaVersion: "1.0", MustReload: []string{}, Reasons: map[string][]string{}}
	if semanticInputsChanged {
		for _, task := range plan.Tasks {
			addReload(&result, task.ID, "semantic input changed")
		}
		return finalizeInvalidation(result)
	}
	changed := map[string]bool{}
	for _, path := range changedPaths {
		changed[path] = true
	}
	taskByID := map[string]planner.Task{}
	for _, task := range plan.Tasks {
		taskByID[task.ID] = task
		for _, path := range task.Paths {
			if changed[path.PathBytesBase64] {
				addReload(&result, task.ID, "direct path changed")
			}
		}
	}
	for {
		before := len(result.MustReload)
		for _, task := range plan.Tasks {
			if hasReload(result, task.ID) {
				for _, dependency := range task.DependencyTasks {
					addReload(&result, dependency, "dependency neighbor changed")
				}
				for _, peer := range plan.Tasks {
					for _, dependency := range peer.DependencyTasks {
						if dependency == task.ID {
							addReload(&result, peer.ID, "dependent neighbor changed")
						}
					}
				}
			}
		}
		if len(result.MustReload) == before {
			break
		}
	}
	contractTypes := map[string]bool{}
	for id := range taskByID {
		if hasReload(result, id) {
			for _, contractType := range taskByID[id].ContractTypes {
				contractTypes[contractType] = true
			}
		}
	}
	for _, task := range plan.Tasks {
		for _, contractType := range task.ContractTypes {
			if contractTypes[contractType] {
				addReload(&result, task.ID, "shared contract consumer changed")
			}
		}
	}
	return finalizeInvalidation(result)
}

func addReload(result *Invalidation, taskID, reason string) {
	if !hasReload(*result, taskID) {
		result.MustReload = append(result.MustReload, taskID)
	}
	for _, existing := range result.Reasons[taskID] {
		if existing == reason {
			return
		}
	}
	result.Reasons[taskID] = append(result.Reasons[taskID], reason)
}

func hasReload(result Invalidation, taskID string) bool {
	for _, id := range result.MustReload {
		if id == taskID {
			return true
		}
	}
	return false
}

func finalizeInvalidation(result Invalidation) Invalidation {
	sort.Strings(result.MustReload)
	for id := range result.Reasons {
		sort.Strings(result.Reasons[id])
	}
	return result
}
