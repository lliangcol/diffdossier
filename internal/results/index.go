package results

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/lliangcol/diffdossier/internal/planner"
)

type Record struct {
	TaskID           string    `json:"task_id"`
	PassID           string    `json:"pass_id"`
	Perspective      string    `json:"perspective"`
	Provider         string    `json:"provider"`
	Model            string    `json:"model"`
	ModelFamily      string    `json:"model_family"`
	PromptDigest     string    `json:"prompt_digest"`
	ContextIsolation string    `json:"context_isolation"`
	ResultPath       string    `json:"result_path"`
	ResultDigest     string    `json:"result_digest"`
	Completed        bool      `json:"completed"`
	RecordedAt       time.Time `json:"recorded_at"`
}

type Index struct {
	SchemaVersion string   `json:"schema_version"`
	Records       []Record `json:"records"`
}

func Append(index Index, result Result, validation Validation, resultPath string, now time.Time) (Index, error) {
	if index.SchemaVersion == "" {
		index = Index{SchemaVersion: "1.0", Records: []Record{}}
	}
	if index.SchemaVersion != "1.0" {
		return Index{}, errors.New("unsupported result index schema")
	}
	for _, record := range index.Records {
		if record.TaskID == result.TaskID && record.PassID == result.Reviewer.PassID {
			return Index{}, fmt.Errorf("task %s already has pass_id %s", result.TaskID, result.Reviewer.PassID)
		}
	}
	digest, err := Digest(result)
	if err != nil {
		return Index{}, err
	}
	index.Records = append(index.Records, Record{
		TaskID: result.TaskID, PassID: result.Reviewer.PassID, Perspective: result.Reviewer.Perspective,
		Provider: result.Reviewer.Provider, Model: result.Reviewer.Model, ModelFamily: result.Reviewer.ModelFamily,
		PromptDigest: result.Reviewer.PromptDigest, ContextIsolation: result.Reviewer.ContextIsolation, ResultPath: resultPath,
		ResultDigest: digest, Completed: validation.Completed,
		RecordedAt: now.UTC(),
	})
	sort.Slice(index.Records, func(i, j int) bool {
		if index.Records[i].TaskID != index.Records[j].TaskID {
			return index.Records[i].TaskID < index.Records[j].TaskID
		}
		return index.Records[i].PassID < index.Records[j].PassID
	})
	return index, nil
}

func Digest(result Result) (string, error) {
	content, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func VerifyRecord(record Record, result Result, validation Validation) error {
	digest, err := Digest(result)
	if err != nil {
		return err
	}
	if record.TaskID != result.TaskID || record.PassID != result.Reviewer.PassID ||
		record.Perspective != result.Reviewer.Perspective || record.Provider != result.Reviewer.Provider ||
		record.Model != result.Reviewer.Model || record.ModelFamily != result.Reviewer.ModelFamily ||
		record.PromptDigest != result.Reviewer.PromptDigest ||
		record.ContextIsolation != result.Reviewer.ContextIsolation ||
		record.ResultDigest != digest || record.Completed != validation.Completed ||
		record.RecordedAt.IsZero() {
		return errors.New("result index record does not match persisted result")
	}
	return nil
}

func ReviewComplete(index Index, plan planner.Plan) bool {
	for _, task := range plan.Tasks {
		completed := map[string]bool{}
		independentContexts := map[string]bool{}
		for _, record := range index.Records {
			if record.TaskID == task.ID && record.Completed {
				completed[record.Perspective] = true
				independentContexts[record.Provider+"\x00"+record.ModelFamily+"\x00"+record.ContextIsolation] = true
			}
		}
		if len(completed) < task.RequiredPasses || len(independentContexts) < task.RequiredPasses {
			return false
		}
		for _, perspective := range task.Perspectives {
			if !completed[perspective] {
				return false
			}
		}
	}
	return len(plan.Tasks) > 0
}

func ResultPath(taskID, passID string) string {
	digest := sha256.Sum256([]byte(passID))
	return fmt.Sprintf("results/%s/pass-%s.json", taskID, hex.EncodeToString(digest[:8]))
}
