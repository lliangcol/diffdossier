// Package workflow implements evidence-bound workflow transitions without
// modifying a target repository.
package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/lliangcol/diffdossier/internal/results"
)

type FindingRecord struct {
	ID            string           `json:"finding_id"`
	Finding       results.Finding  `json:"finding"`
	TaskID        string           `json:"task_id"`
	SnapshotID    string           `json:"snapshot_id"`
	Reviewer      results.Reviewer `json:"reviewer"`
	State         string           `json:"state"`
	StateOperator string           `json:"state_operator"`
	Reason        string           `json:"reason,omitempty"`
	Owner         string           `json:"owner,omitempty"`
	ExpiresAt     *time.Time       `json:"expires_at,omitempty"`
	ReviewTrigger string           `json:"review_trigger,omitempty"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

type FindingLedger struct {
	SchemaVersion string          `json:"schema_version"`
	Findings      []FindingRecord `json:"findings"`
}

type FixAuthorization struct {
	SchemaVersion string    `json:"schema_version"`
	SnapshotID    string    `json:"snapshot_id"`
	FindingIDs    []string  `json:"finding_ids"`
	ScopeDigest   string    `json:"scope_digest"`
	AuthorizedBy  string    `json:"authorized_by"`
	AuthorizedAt  time.Time `json:"authorized_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Digest        string    `json:"digest"`
}

func ImportFindings(ledger FindingLedger, result results.Result, now time.Time) (FindingLedger, error) {
	if ledger.SchemaVersion == "" {
		ledger = FindingLedger{SchemaVersion: "1.0", Findings: []FindingRecord{}}
	}
	if ledger.SchemaVersion != "1.0" || ledger.Findings == nil {
		return FindingLedger{}, errors.New("invalid finding ledger")
	}
	seen := map[string]bool{}
	for _, record := range ledger.Findings {
		seen[record.ID] = true
	}
	for _, finding := range result.Findings {
		if finding.State != "reported" && finding.State != "needs_confirmation" {
			return FindingLedger{}, errors.New("only Provider-originated finding states can be imported")
		}
		ledgerID := findingLedgerID(result, finding.ID)
		if seen[ledgerID] {
			return FindingLedger{}, fmt.Errorf("duplicate finding import %q", ledgerID)
		}
		seen[ledgerID] = true
		ledger.Findings = append(ledger.Findings, FindingRecord{
			ID: ledgerID, Finding: finding, TaskID: result.TaskID, SnapshotID: result.SnapshotID,
			Reviewer: result.Reviewer, State: finding.State, StateOperator: "provider:" + result.Reviewer.Provider,
			UpdatedAt: now.UTC(),
		})
	}
	sort.Slice(ledger.Findings, func(i, j int) bool { return ledger.Findings[i].ID < ledger.Findings[j].ID })
	return ledger, nil
}

func TransitionFinding(ledger FindingLedger, id, next, operator, reason, owner, reviewTrigger string, expiresAt *time.Time, now time.Time) (FindingLedger, error) {
	if operator == "" {
		return FindingLedger{}, errors.New("finding transition requires an operator")
	}
	for index := range ledger.Findings {
		record := &ledger.Findings[index]
		if record.ID != id {
			continue
		}
		if !allowedFindingTransition(record.State, next) {
			return FindingLedger{}, fmt.Errorf("invalid finding transition %s -> %s", record.State, next)
		}
		if (next == "rejected" || next == "accepted_risk") && reason == "" {
			return FindingLedger{}, errors.New("rejected and accepted_risk require a reason")
		}
		if next == "accepted_risk" && (owner == "" || reviewTrigger == "" || expiresAt == nil || !now.Before(*expiresAt)) {
			return FindingLedger{}, errors.New("accepted_risk requires owner, future expiry, and review trigger")
		}
		record.State = next
		record.StateOperator = operator
		record.Reason = reason
		record.Owner = owner
		record.ReviewTrigger = reviewTrigger
		record.ExpiresAt = expiresAt
		if next != "accepted_risk" {
			record.Owner = ""
			record.ReviewTrigger = ""
			record.ExpiresAt = nil
		}
		record.UpdatedAt = now.UTC()
		return ledger, nil
	}
	return FindingLedger{}, fmt.Errorf("finding %q not found", id)
}

func AuthorizeFix(ledger FindingLedger, snapshotID string, findingIDs []string, operator, scopeDigest string, now, expiresAt time.Time) (FixAuthorization, FindingLedger, error) {
	if snapshotID == "" || operator == "" || scopeDigest == "" || len(findingIDs) == 0 || !now.Before(expiresAt) {
		return FixAuthorization{}, FindingLedger{}, errors.New("fix authorization requires exact scope, operator, findings, and future expiry")
	}
	ledger.Findings = append([]FindingRecord(nil), ledger.Findings...)
	ids := append([]string(nil), findingIDs...)
	sort.Strings(ids)
	for index, id := range ids {
		if index > 0 && id == ids[index-1] {
			return FixAuthorization{}, FindingLedger{}, errors.New("duplicate finding in fix authorization")
		}
		found := false
		for ledgerIndex := range ledger.Findings {
			record := &ledger.Findings[ledgerIndex]
			if record.ID == id {
				found = true
				if record.SnapshotID != snapshotID || record.State != "confirmed" {
					return FixAuthorization{}, FindingLedger{}, errors.New("fix authorization requires confirmed findings on the exact snapshot")
				}
				record.State = "fix_authorized"
				record.StateOperator = operator
				record.UpdatedAt = now.UTC()
			}
		}
		if !found {
			return FixAuthorization{}, FindingLedger{}, fmt.Errorf("finding %q not found", id)
		}
	}
	authorization := FixAuthorization{
		SchemaVersion: "1.0", SnapshotID: snapshotID, FindingIDs: ids,
		ScopeDigest: scopeDigest, AuthorizedBy: operator, AuthorizedAt: now.UTC(), ExpiresAt: expiresAt.UTC(),
	}
	encoded, err := json.Marshal(authorization)
	if err != nil {
		return FixAuthorization{}, FindingLedger{}, err
	}
	digest := sha256.Sum256(encoded)
	authorization.Digest = "sha256:" + hex.EncodeToString(digest[:])
	return authorization, ledger, nil
}

func findingLedgerID(result results.Result, providerID string) string {
	digest := sha256.Sum256([]byte(result.SnapshotID + "\x00" + result.TaskID + "\x00" + result.Reviewer.PassID + "\x00" + providerID))
	return "finding-" + hex.EncodeToString(digest[:12])
}

func VerifyFixAuthorization(authorization FixAuthorization, now time.Time) error {
	if authorization.SchemaVersion != "1.0" || authorization.SnapshotID == "" || authorization.ScopeDigest == "" || authorization.AuthorizedBy == "" || len(authorization.FindingIDs) == 0 || !now.Before(authorization.ExpiresAt) {
		return errors.New("fix authorization is invalid or expired")
	}
	claimed := authorization.Digest
	authorization.Digest = ""
	encoded, err := json.Marshal(authorization)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(encoded)
	if claimed != "sha256:"+hex.EncodeToString(digest[:]) {
		return errors.New("fix authorization digest mismatch")
	}
	return nil
}

func MutationScopeDigest(pathBytesBase64 []string) string {
	paths := append([]string(nil), pathBytesBase64...)
	sort.Strings(paths)
	hasher := sha256.New()
	for _, path := range paths {
		hasher.Write([]byte(path))
		hasher.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}

func allowedFindingTransition(current, next string) bool {
	allowed := map[string]map[string]bool{
		"reported":           {"confirmed": true, "rejected": true, "needs_confirmation": true},
		"needs_confirmation": {"confirmed": true, "rejected": true},
		"confirmed":          {"fix_authorized": true, "accepted_risk": true},
		"fix_authorized":     {"fixed_unverified": true},
		"fixed_unverified":   {"verified_closed": true, "confirmed": true},
	}
	return allowed[current][next]
}
