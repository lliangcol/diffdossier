package workflow

import (
	"testing"
	"time"

	"github.com/lliangcol/diffdossier/internal/planner"
	"github.com/lliangcol/diffdossier/internal/results"
)

func TestFindingLifecycleAndExactFixAuthorization(t *testing.T) {
	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	result := results.Result{
		TaskID: "task-a", SnapshotID: "snap-a",
		Reviewer: results.Reviewer{Provider: "manual"},
		Findings: []results.Finding{{ID: "F-1", State: "reported"}},
	}
	ledger, err := ImportFindings(FindingLedger{}, result, now)
	if err != nil {
		t.Fatal(err)
	}
	ledgerID := ledger.Findings[0].ID
	ledger, err = TransitionFinding(ledger, ledgerID, "confirmed", "owner", "", "", "", nil, now)
	if err != nil {
		t.Fatal(err)
	}
	authorization, ledger, err := AuthorizeFix(ledger, "snap-a", []string{ledgerID}, "owner", "sha256:scope", now, now.Add(time.Hour))
	if err != nil || authorization.Digest == "" || ledger.Findings[0].State != "fix_authorized" {
		t.Fatalf("authorization=%+v ledger=%+v err=%v", authorization, ledger, err)
	}
	if _, _, err := AuthorizeFix(ledger, "snap-a", []string{ledgerID}, "owner", "sha256:scope", now, now.Add(time.Hour)); err == nil {
		t.Fatal("already authorized finding was reauthorized")
	}
}

func TestAcceptedRiskRequiresBoundedOwnership(t *testing.T) {
	now := time.Now().UTC()
	ledger := FindingLedger{SchemaVersion: "1.0", Findings: []FindingRecord{{ID: "finding-f", Finding: results.Finding{ID: "F"}, State: "confirmed"}}}
	if _, err := TransitionFinding(ledger, "finding-f", "accepted_risk", "owner", "reason", "", "", nil, now); err == nil {
		t.Fatal("unbounded accepted risk was allowed")
	}
	expires := now.Add(time.Hour)
	if _, err := TransitionFinding(ledger, "finding-f", "accepted_risk", "owner", "reason", "team", "release", &expires, now); err != nil {
		t.Fatal(err)
	}
}

func TestFixAuthorizationIsAtomicAndVerifiable(t *testing.T) {
	now := time.Now().UTC()
	ledger := FindingLedger{SchemaVersion: "1.0", Findings: []FindingRecord{{ID: "finding-f1", Finding: results.Finding{ID: "F-1"}, SnapshotID: "snap", State: "confirmed"}}}
	_, _, err := AuthorizeFix(ledger, "snap", []string{"finding-f1", "missing"}, "owner", "scope", now, now.Add(time.Hour))
	if err == nil {
		t.Fatal("invalid authorization succeeded")
	}
	// On error the returned ledger is deliberately zero; ensure the input did
	// not share a mutated backing array.
	if ledger.Findings[0].State != "confirmed" {
		t.Fatal("failed authorization mutated input ledger")
	}
	authorization, _, err := AuthorizeFix(ledger, "snap", []string{"finding-f1"}, "owner", "scope", now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyFixAuthorization(authorization, now); err != nil {
		t.Fatal(err)
	}
	authorization.ScopeDigest = "changed"
	if err := VerifyFixAuthorization(authorization, now); err == nil {
		t.Fatal("tampered authorization verified")
	}
}

func TestMutationScopeDigestIsOrderIndependent(t *testing.T) {
	if MutationScopeDigest([]string{"b", "a"}) != MutationScopeDigest([]string{"a", "b"}) {
		t.Fatal("scope digest depends on order")
	}
}

func TestProviderFindingIDsAreNamespacedPerPass(t *testing.T) {
	now := time.Now()
	base := results.Result{TaskID: "task", SnapshotID: "snap", Reviewer: results.Reviewer{Provider: "manual", PassID: "one"}, Findings: []results.Finding{{ID: "F-1", State: "reported"}}}
	ledger, err := ImportFindings(FindingLedger{}, base, now)
	if err != nil {
		t.Fatal(err)
	}
	base.Reviewer.PassID = "two"
	ledger, err = ImportFindings(ledger, base, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(ledger.Findings) != 2 || ledger.Findings[0].ID == ledger.Findings[1].ID {
		t.Fatalf("ledger IDs not independent: %+v", ledger.Findings)
	}
}

func TestInvalidationIncludesDependencyAndContractPeers(t *testing.T) {
	plan := planner.Plan{Tasks: []planner.Task{
		{ID: "a", Paths: []planner.PathRef{{PathBytesBase64: "cGF5"}}, DependencyTasks: []string{"b"}, ContractTypes: []string{"api"}},
		{ID: "b", Paths: []planner.PathRef{{PathBytesBase64: "bG9naWM="}}, ContractTypes: []string{"logic"}},
		{ID: "c", Paths: []planner.PathRef{{PathBytesBase64: "dGVzdA=="}}, ContractTypes: []string{"api"}},
	}}
	result := ComputeInvalidation(plan, []string{"cGF5"}, false)
	if len(result.MustReload) != 3 {
		t.Fatalf("invalidation=%+v", result)
	}
	all := ComputeInvalidation(plan, nil, true)
	if len(all.MustReload) != 3 {
		t.Fatalf("semantic invalidation=%+v", all)
	}
}
