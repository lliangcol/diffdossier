package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lliangcol/diffdossier/internal/gitrepo"
	"github.com/lliangcol/diffdossier/internal/packets"
	"github.com/lliangcol/diffdossier/internal/policy"
	"github.com/lliangcol/diffdossier/internal/providers"
	"github.com/lliangcol/diffdossier/internal/redact"
	"github.com/lliangcol/diffdossier/internal/snapshot"
	"github.com/lliangcol/diffdossier/internal/store"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

type repeatedFlag []string

func (values *repeatedFlag) String() string { return strings.Join(*values, ",") }
func (values *repeatedFlag) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func runReview(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "run" {
		fmt.Fprintln(stderr, "usage: diffdossier review run --task-id ID [--provider manual|command] ...")
		return ExitUsage
	}
	flags := flag.NewFlagSet("review run", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repoFlag := flags.String("repo", ".", "target Git repository")
	configFlag := flags.String("config", "", "configuration file")
	baselineFlag := flags.String("baseline", "", "exact local baseline ref override")
	stateFlag := flags.String("state-dir", "", "durable state directory")
	runFlag := flags.String("run-id", "", "contracted run ID (default: latest)")
	taskFlag := flags.String("task-id", "", "task ID")
	providerFlag := flags.String("provider", "", "manual or command (default: review.default_provider)")
	executable := flags.String("executable", "", "absolute command Provider executable")
	workingDir := flags.String("provider-cwd", "", "repository-external Provider working directory")
	timeout := flags.Duration("timeout", 5*time.Minute, "Provider timeout")
	maxStdout := flags.Int64("max-stdout-bytes", 8*1024*1024, "Provider stdout limit")
	maxStderr := flags.Int64("max-stderr-bytes", 1024*1024, "Provider stderr limit")
	networkClass := flags.String("network-destination-class", "unknown", "none, local, external, or unknown")
	credentialSource := flags.String("credential-source", "none", "none, environment, stdin_proxy, or system_credential")
	trustDigest := flags.String("trust-execution-plan", "", "exact displayed command plan digest")
	trustFile := flags.String("trust-binding", "", "private trust binding JSON")
	egressFile := flags.String("egress-grant", "", "private egress grant JSON")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	var commandArgs repeatedFlag
	var environmentNames repeatedFlag
	flags.Var(&commandArgs, "arg", "command argv item (repeatable)")
	flags.Var(&environmentNames, "env", "allowlisted environment variable name (repeatable)")
	if err := flags.Parse(args[1:]); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || !validTaskID(*taskFlag) {
		return ExitUsage
	}

	resolved, err := resolveExportContext(*repoFlag, *stateFlag, *runFlag)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_REVIEW_CONTEXT", err.Error()), ExitEvidence)
	}
	if resolved.run.State != "CONTRACTED" && resolved.run.State != "REVIEWING" {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_WORKFLOW_STATE", "review run requires CONTRACTED or REVIEWING run"), ExitIncomplete)
	}
	effective, err := loadEffectiveConfig(resolved.repoRoot, *configFlag, *baselineFlag)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_CONFIG_INVALID", err.Error()), ExitUsage)
	}
	cfg := effective.Config
	providerName := *providerFlag
	if providerName == "" {
		providerName = cfg.Review.DefaultProvider
	}
	packet, err := loadTaskPacket(resolved, *taskFlag)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PACKET_INTEGRITY", err.Error()), ExitEvidence)
	}
	repo, err := gitrepo.Open(nilContext(), resolved.repoRoot)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_GIT_REPOSITORY", err.Error()), ExitEvidence)
	}
	digests, err := semanticDigests(repo, effective)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EVIDENCE_DIGEST", err.Error()), ExitEvidence)
	}
	request := snapshot.Request{
		Repo: repo, Baseline: cfg.Baseline, InputDigests: digests,
		IncludeUntracked: cfg.IncludeUntracked, IncludeIgnored: cfg.IncludeIgnored,
	}
	if err := snapshot.VerifyFresh(nilContext(), request, resolved.seal); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_SNAPSHOT_STALE", err.Error()), ExitStale)
	}
	if providerName == "command" {
		packet, err = packets.Materialize(packet, resolved.stateStore.ReadBlob)
		if err != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PACKET_MATERIALIZE", err.Error()), ExitEvidence)
		}
	}
	packetBytes, _ := json.Marshal(packet)
	packetDigest := digestBytes(packetBytes)
	scanFindings, err := redact.Scan(packetBytes)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PACKET_SCAN", err.Error()), ExitEvidence)
	}
	scanBytes, _ := json.Marshal(scanFindings)
	redactionDigest := digestBytes(scanBytes)
	disclosure := map[string]any{
		"provider":                  providerName,
		"paths":                     packet.Files,
		"total_bytes":               packet.TotalBytes,
		"data_class":                packet.DataClass,
		"redaction_findings":        scanFindings,
		"redaction_digest":          redactionDigest,
		"scan_scope":                "serialized_provider_packet",
		"network_destination_class": *networkClass,
		"credential_source":         *credentialSource,
		"strong_os_sandbox":         false,
	}
	if providerName == "manual" {
		disclosure["network_destination_class"] = "none"
		disclosure["credential_source"] = "none"
		attempt := providers.Attempt{
			TaskID: packet.TaskID, Provider: "manual", PacketDigest: packetDigest,
			NetworkDestinationClass: "none", CredentialSource: "none",
			Status: "packet_ready", RecordedAt: time.Now().UTC(),
		}
		if err := appendReviewAttempt(resolved, attempt); err != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_REVIEW_ATTEMPT", err.Error()), ExitEvidence)
		}
		data := map[string]any{"run_id": resolved.run.ID, "task_id": packet.TaskID, "executed": false, "manual_import_required": true, "disclosure": disclosure, "packet": packet}
		if *jsonOutput {
			return writeJSON(stdout, stderr, publicschema.Success(data))
		}
		fmt.Fprintf(stdout, "manual packet ready for %s; no Provider executed\n", packet.TaskID)
		return ExitOK
	}
	if providerName != "command" || *executable == "" || *workingDir == "" {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PROVIDER_CONFIG", "command Provider requires absolute --executable and --provider-cwd"), ExitUsage)
	}
	environment, err := resolveProviderEnvironment(environmentNames)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PROVIDER_ENV", err.Error()), ExitUsage)
	}
	commandConfig := providers.CommandConfig{
		RepositoryID: resolved.repository.ID, RepositoryRoot: resolved.repoRoot,
		Executable: *executable, Args: commandArgs, WorkingDir: *workingDir, Env: environment,
		Timeout: *timeout, MaxStdout: *maxStdout, MaxStderr: *maxStderr,
		ConfigDigest: digests["config"], DataClass: packet.DataClass,
		NetworkDestinationClass: *networkClass, CredentialSource: *credentialSource,
		RedactionDigest: redactionDigest,
	}
	plan, err := providers.DescribeCommand(commandConfig, packet)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PROVIDER_PLAN", err.Error()), ExitEvidence)
	}
	planPath := filepath.Join("reviews", "plans", packet.TaskID+"-"+strings.TrimPrefix(plan.ExecutionPlanDigest, "sha256:")[:16]+".json")
	if err := resolved.stateStore.WriteRunJSON(resolved.runDir, planPath, plan); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
	}
	if err := appendReviewAttempt(resolved, providers.Attempt{
		TaskID: packet.TaskID, Provider: "command", PacketDigest: packetDigest,
		ExecutionPlanDigest:     plan.ExecutionPlanDigest,
		NetworkDestinationClass: *networkClass, CredentialSource: *credentialSource,
		Status: "planned", RecordedAt: time.Now().UTC(),
	}); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_REVIEW_ATTEMPT", err.Error()), ExitEvidence)
	}
	if *trustDigest == "" {
		data := map[string]any{"run_id": resolved.run.ID, "task_id": packet.TaskID, "executed": false, "authorization_required": true, "blocked_by_scan": len(scanFindings) > 0, "disclosure": disclosure, "command_plan": plan}
		if *jsonOutput {
			return writeJSON(stdout, stderr, publicschema.Success(data))
		}
		fmt.Fprintf(stdout, "command Provider plan %s; review disclosure and authorize exact plan before execution\n", plan.ExecutionPlanDigest)
		return ExitOK
	}
	if *trustDigest != plan.ExecutionPlanDigest || *trustFile == "" || *egressFile == "" {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PROVIDER_UNAUTHORIZED", "execution requires matching plan digest, trust binding, and egress grant"), ExitEvidence)
	}
	if len(scanFindings) > 0 {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PROVIDER_EGRESS_DENIED", "packet scan has findings; command Provider was not executed"), ExitEvidence)
	}
	var trust policy.TrustBinding
	if err := readStrictJSON(*trustFile, &trust); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PROVIDER_TRUST", err.Error()), ExitEvidence)
	}
	var egress policy.EgressGrant
	if err := readStrictJSON(*egressFile, &egress); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PROVIDER_EGRESS", err.Error()), ExitEvidence)
	}
	command, err := providers.NewCommand(commandConfig, packet, trust, egress)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PROVIDER_CONFIG", err.Error()), ExitEvidence)
	}
	if err := appendReviewAttempt(resolved, providers.Attempt{
		TaskID: packet.TaskID, Provider: "command", PacketDigest: packetDigest,
		ExecutionPlanDigest:     plan.ExecutionPlanDigest,
		NetworkDestinationClass: *networkClass, CredentialSource: *credentialSource,
		Status: "started", RecordedAt: time.Now().UTC(),
	}); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_REVIEW_ATTEMPT", err.Error()), ExitEvidence)
	}
	result, err := command.Review(nilContext(), packet)
	if err != nil {
		_ = appendReviewAttempt(resolved, providers.Attempt{
			TaskID: packet.TaskID, Provider: "command", PacketDigest: packetDigest,
			ExecutionPlanDigest:     plan.ExecutionPlanDigest,
			NetworkDestinationClass: *networkClass, CredentialSource: *credentialSource,
			Status: "failed", FailureClass: providers.FailureClass(err), RecordedAt: time.Now().UTC(),
		})
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PROVIDER_FAILED", err.Error()), ExitEvidence)
	}
	resultFile, err := os.CreateTemp("", "diffdossier-provider-result-*.json")
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RESULT_TEMP", err.Error()), ExitInternal)
	}
	resultPath := resultFile.Name()
	defer os.Remove(resultPath)
	encodeErr := json.NewEncoder(resultFile).Encode(result)
	closeErr := resultFile.Close()
	if encodeErr != nil || closeErr != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RESULT_TEMP", errors.Join(encodeErr, closeErr).Error()), ExitInternal)
	}
	var recordOutput, recordError bytes.Buffer
	recordCode := runRecord([]string{
		"task", "--repo", resolved.repoRoot, "--config", *configFlag, "--baseline", *baselineFlag, "--state-dir", resolved.stateStore.Root,
		"--run-id", resolved.run.ID, "--task-id", packet.TaskID, "--result", resultPath, "--json",
	}, &recordOutput, &recordError)
	if recordCode != ExitOK {
		_ = appendReviewAttempt(resolved, providers.Attempt{
			TaskID: packet.TaskID, Provider: "command", PacketDigest: packetDigest,
			ExecutionPlanDigest:     plan.ExecutionPlanDigest,
			NetworkDestinationClass: *networkClass, CredentialSource: *credentialSource,
			Status: "failed", FailureClass: "provider_failed", RecordedAt: time.Now().UTC(),
		})
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_RESULT_INVALID", strings.TrimSpace(recordError.String())), recordCode)
	}
	if err := appendReviewAttempt(resolved, providers.Attempt{
		TaskID: packet.TaskID, Provider: "command", PacketDigest: packetDigest,
		ExecutionPlanDigest:     plan.ExecutionPlanDigest,
		NetworkDestinationClass: *networkClass, CredentialSource: *credentialSource,
		Status: "completed", RecordedAt: time.Now().UTC(),
	}); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_REVIEW_ATTEMPT", err.Error()), ExitEvidence)
	}
	if _, err := io.Copy(stdout, &recordOutput); err != nil {
		fmt.Fprintf(stderr, "write review result output: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

func appendReviewAttempt(resolved exportContext, attempt providers.Attempt) error {
	lock, err := store.AcquireRunLock(resolved.runDir)
	if err != nil {
		return err
	}
	defer lock.Release()
	ledger := providers.AttemptLedger{SchemaVersion: "1.0", Attempts: []providers.Attempt{}}
	if err := resolved.stateStore.ReadRunJSON(resolved.runDir, "reviews/attempts.json", &ledger); err != nil && !os.IsNotExist(err) {
		return err
	}
	ledger, err = providers.AppendAttempt(ledger, attempt)
	if err != nil {
		return err
	}
	if err := resolved.stateStore.WriteRunJSON(resolved.runDir, "reviews/attempts.json", ledger); err != nil {
		return err
	}
	_, err = resolved.stateStore.AppendEvent(resolved.runDir, "provider_attempt_recorded", map[string]any{
		"sequence": ledger.Attempts[len(ledger.Attempts)-1].Sequence,
		"task_id":  attempt.TaskID, "provider": attempt.Provider,
		"status": attempt.Status, "execution_plan_digest": attempt.ExecutionPlanDigest,
	})
	return err
}

func resolveProviderEnvironment(names []string) ([]string, error) {
	seen := map[string]bool{}
	values := make([]string, 0, len(names))
	for _, name := range names {
		if name == "" || strings.ContainsAny(name, "=\x00") || seen[name] {
			return nil, fmt.Errorf("invalid or duplicate environment name %q", name)
		}
		value, present := os.LookupEnv(name)
		if !present {
			return nil, fmt.Errorf("allowlisted environment variable %q is not set", name)
		}
		seen[name] = true
		values = append(values, name+"="+value)
	}
	return values, nil
}

func readStrictJSON(path string, target any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("JSON file must contain exactly one object")
	}
	return nil
}

func digestBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}
