package gates

import (
	"context"
	"errors"
	"time"

	"github.com/lliangcol/diffdossier/internal/policy"
)

var ErrExecutionUnauthorized = errors.New("gate execution is not authorized")

type Executor interface {
	Execute(context.Context, ExpandedGate) error
}

type Evidence struct {
	SchemaVersion    string    `json:"schema_version"`
	GateID           string    `json:"gate_id"`
	SnapshotID       string    `json:"snapshot_id"`
	PlanDigest       string    `json:"plan_digest"`
	DefinitionDigest string    `json:"definition_digest"`
	Status           string    `json:"status"`
	CacheHit         bool      `json:"cache_hit"`
	FinalRun         bool      `json:"final_run"`
	RecordedAt       time.Time `json:"recorded_at"`
}

// Run executes only an already-expanded plan with an exact, unexpired trust
// binding. The caller supplies freshness guards so mutations can fail closed.
func Run(ctx context.Context, plan Plan, trust policy.TrustBinding, now time.Time, executor Executor, before, after func() error, cached map[string]Evidence, final bool) ([]Evidence, error) {
	if !trust.Authorizes(plan.TrustCandidate, now) {
		return nil, ErrExecutionUnauthorized
	}
	if executor == nil || before == nil || after == nil {
		return nil, errors.New("executor and freshness guards are required")
	}
	if err := before(); err != nil {
		return nil, err
	}
	results := []Evidence{}
	passed := map[string]bool{}
	for _, gate := range plan.Gates {
		if gate.Executable != "" {
			currentDigest, digestErr := fileDigest(gate.Executable)
			if digestErr != nil || currentDigest != gate.ExecutableDigest {
				return results, errors.Join(errors.New("gate executable changed after planning"), digestErr)
			}
		}
		for _, dependency := range gate.DependsOn {
			if !passed[dependency] {
				return results, errors.New("gate dependency did not pass")
			}
		}
		cacheKey := plan.SnapshotID + "\x00" + gate.DefinitionDigest + "\x00" + plan.BinaryDigest + "\x00" + gate.ExecutableDigest
		if prior, ok := cached[cacheKey]; ok && gate.CacheClass == "worktree_deterministic" && !(final && gate.FinalAlways) && prior.Status == "pass" {
			prior.CacheHit = true
			prior.RecordedAt = now.UTC()
			results = append(results, prior)
			passed[gate.ID] = true
			continue
		}
		gateCtx, cancel := context.WithTimeout(ctx, time.Duration(gate.TimeoutSeconds)*time.Second)
		err := executor.Execute(gateCtx, gate)
		cancel()
		status := "pass"
		if err != nil {
			status = "fail"
		}
		evidence := Evidence{SchemaVersion: "1.0", GateID: gate.ID, SnapshotID: plan.SnapshotID, PlanDigest: plan.PlanDigest, DefinitionDigest: gate.DefinitionDigest, Status: status, FinalRun: final, RecordedAt: now.UTC()}
		results = append(results, evidence)
		if freshnessErr := after(); freshnessErr != nil {
			return results, errors.Join(errors.New("gate caused unexpected repository mutation; snapshot is stale"), freshnessErr)
		}
		if err != nil && gate.Blocking {
			return results, err
		}
		passed[gate.ID] = err == nil
	}
	return results, nil
}
