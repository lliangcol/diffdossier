package providers

import (
	"context"
	"errors"
	"strings"
	"time"
)

type Attempt struct {
	Sequence                int       `json:"sequence"`
	TaskID                  string    `json:"task_id"`
	Provider                string    `json:"provider"`
	PacketDigest            string    `json:"packet_digest"`
	ExecutionPlanDigest     string    `json:"execution_plan_digest,omitempty"`
	NetworkDestinationClass string    `json:"network_destination_class"`
	CredentialSource        string    `json:"credential_source"`
	Status                  string    `json:"status"`
	FailureClass            string    `json:"failure_class,omitempty"`
	RecordedAt              time.Time `json:"recorded_at"`
}

type AttemptLedger struct {
	SchemaVersion string    `json:"schema_version"`
	Attempts      []Attempt `json:"attempts"`
}

func AppendAttempt(ledger AttemptLedger, attempt Attempt) (AttemptLedger, error) {
	if ledger.SchemaVersion == "" {
		ledger = AttemptLedger{SchemaVersion: "1.0", Attempts: []Attempt{}}
	}
	if ledger.SchemaVersion != "1.0" || ledger.Attempts == nil {
		return AttemptLedger{}, errors.New("attempt ledger must use schema 1.0 with an attempts array")
	}
	if attempt.TaskID == "" || attempt.Provider == "" || attempt.PacketDigest == "" || attempt.Status == "" || attempt.RecordedAt.IsZero() {
		return AttemptLedger{}, errors.New("review attempt is incomplete")
	}
	attempt.Sequence = len(ledger.Attempts) + 1
	ledger.Attempts = append(ledger.Attempts, attempt)
	return ledger, nil
}

func FailureClass(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "quota"):
		return "quota_exhausted"
	case strings.Contains(message, "rate limit"):
		return "rate_limited"
	case strings.Contains(message, "login"), strings.Contains(message, "authentication"):
		return "login_required"
	default:
		return "provider_failed"
	}
}
