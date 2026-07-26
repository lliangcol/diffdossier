package providers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/lliangcol/diffdossier/internal/packets"
	"github.com/lliangcol/diffdossier/internal/policy"
	processrunner "github.com/lliangcol/diffdossier/internal/process"
	"github.com/lliangcol/diffdossier/internal/results"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

var (
	ErrCommandUnauthorized = errors.New("command Provider execution is not authorized")
	ErrHandshakeInvalid    = errors.New("command Provider handshake is incompatible")
)

const commandCapability = "provider:command:review"

type CommandConfig struct {
	RepositoryID            string
	RepositoryRoot          string
	Executable              string
	Args                    []string
	WorkingDir              string
	Env                     []string
	Timeout                 time.Duration
	MaxStdout               int64
	MaxStderr               int64
	ConfigDigest            string
	DataClass               publicschema.DataClass
	NetworkDestinationClass string
	CredentialSource        string
	RedactionDigest         string
	Now                     func() time.Time
}

type CommandPlan struct {
	Provider                string                 `json:"provider"`
	Executable              string                 `json:"executable"`
	Argv                    []string               `json:"argv"`
	WorkingDir              string                 `json:"working_dir"`
	Environment             map[string]string      `json:"environment_value_digests"`
	TimeoutMilliseconds     int64                  `json:"timeout_milliseconds"`
	MaxStdout               int64                  `json:"max_stdout_bytes"`
	MaxStderr               int64                  `json:"max_stderr_bytes"`
	DataClass               publicschema.DataClass `json:"data_class"`
	NetworkDestinationClass string                 `json:"network_destination_class"`
	CredentialSource        string                 `json:"credential_source"`
	RedactionDigest         string                 `json:"redaction_digest"`
	InputBytes              int64                  `json:"input_bytes"`
	StrongOSSandbox         bool                   `json:"strong_os_sandbox"`
	ExecutionPlanDigest     string                 `json:"execution_plan_digest"`
	BinaryDigest            string                 `json:"binary_digest"`
	TrustCandidate          policy.TrustBinding    `json:"trust_candidate"`
	EgressRequest           policy.EgressRequest   `json:"egress_request"`
}

type Command struct {
	config CommandConfig
	packet packets.Packet
	plan   CommandPlan
	trust  policy.TrustBinding
	egress policy.EgressGrant
}

type commandRequest struct {
	ProtocolVersion string          `json:"protocol_version"`
	Operation       string          `json:"operation"`
	Packet          *packets.Packet `json:"packet,omitempty"`
}

func DescribeCommand(config CommandConfig, packet packets.Packet) (CommandPlan, error) {
	if config.RepositoryID == "" || config.ConfigDigest == "" {
		return CommandPlan{}, errors.New("repository_id and config digest are required")
	}
	if packet.SnapshotID == "" || packet.TaskInputHash == "" {
		return CommandPlan{}, errors.New("packet must be bound to a snapshot and task input")
	}
	if config.Timeout <= 0 || config.MaxStdout < 1 || config.MaxStderr < 1 {
		return CommandPlan{}, errors.New("positive timeout and output limits are required")
	}
	if config.DataClass != publicschema.PublicSynthetic &&
		config.DataClass != publicschema.PublicProject &&
		config.DataClass != publicschema.PrivateProject {
		return CommandPlan{}, errors.New("command Provider cannot receive denied or unclassified data")
	}
	if !contains([]string{"none", "local", "external", "unknown"}, config.NetworkDestinationClass) {
		return CommandPlan{}, errors.New("invalid network destination class")
	}
	if !contains([]string{"none", "environment", "stdin_proxy", "system_credential"}, config.CredentialSource) {
		return CommandPlan{}, errors.New("invalid credential source")
	}
	if !strings.HasPrefix(config.RedactionDigest, "sha256:") || len(config.RedactionDigest) != len("sha256:")+64 {
		return CommandPlan{}, errors.New("redaction digest is required")
	}
	executable, binaryDigest, err := validateExecutable(config.Executable)
	if err != nil {
		return CommandPlan{}, err
	}
	workingDir, err := validateWorkingDir(config.RepositoryRoot, config.WorkingDir)
	if err != nil {
		return CommandPlan{}, err
	}
	environment, err := environmentDigests(config.Env)
	if err != nil {
		return CommandPlan{}, err
	}
	request := commandRequest{ProtocolVersion: "1.0", Operation: "review", Packet: &packet}
	input, err := json.Marshal(request)
	if err != nil {
		return CommandPlan{}, err
	}
	plan := CommandPlan{
		Provider: "command", Executable: executable, Argv: append([]string(nil), config.Args...),
		WorkingDir: workingDir, Environment: environment,
		TimeoutMilliseconds: config.Timeout.Milliseconds(), MaxStdout: config.MaxStdout, MaxStderr: config.MaxStderr,
		DataClass: config.DataClass, InputBytes: int64(len(input)), StrongOSSandbox: false,
		NetworkDestinationClass: config.NetworkDestinationClass,
		CredentialSource:        config.CredentialSource,
		RedactionDigest:         config.RedactionDigest,
		BinaryDigest:            binaryDigest,
	}
	digestInput := struct {
		Provider                string                 `json:"provider"`
		Executable              string                 `json:"executable"`
		Argv                    []string               `json:"argv"`
		WorkingDir              string                 `json:"working_dir"`
		Environment             map[string]string      `json:"environment_value_digests"`
		TimeoutMilliseconds     int64                  `json:"timeout_milliseconds"`
		MaxStdout               int64                  `json:"max_stdout_bytes"`
		MaxStderr               int64                  `json:"max_stderr_bytes"`
		DataClass               publicschema.DataClass `json:"data_class"`
		NetworkDestinationClass string                 `json:"network_destination_class"`
		CredentialSource        string                 `json:"credential_source"`
		RedactionDigest         string                 `json:"redaction_digest"`
		SnapshotID              string                 `json:"snapshot_id"`
		TaskInputDigest         string                 `json:"task_input_digest"`
		BinaryDigest            string                 `json:"binary_digest"`
	}{plan.Provider, plan.Executable, plan.Argv, plan.WorkingDir, plan.Environment, plan.TimeoutMilliseconds,
		plan.MaxStdout, plan.MaxStderr, plan.DataClass, plan.NetworkDestinationClass,
		plan.CredentialSource, plan.RedactionDigest, packet.SnapshotID, packet.TaskInputHash, plan.BinaryDigest}
	encoded, err := json.Marshal(digestInput)
	if err != nil {
		return CommandPlan{}, err
	}
	digest := sha256.Sum256(encoded)
	plan.ExecutionPlanDigest = "sha256:" + hex.EncodeToString(digest[:])
	plan.TrustCandidate = policy.TrustBinding{
		RepositoryID: config.RepositoryID, SnapshotID: packet.SnapshotID, TaskInputDigest: packet.TaskInputHash,
		ExecutionPlanDigest: plan.ExecutionPlanDigest, ConfigDigest: config.ConfigDigest,
		BinaryDigest: plan.BinaryDigest, Capability: commandCapability,
	}
	plan.EgressRequest = policy.EgressRequest{
		Provider: "command", SnapshotID: packet.SnapshotID, TaskInputDigest: packet.TaskInputHash,
		DataClass: config.DataClass, Bytes: plan.InputBytes,
	}
	return plan, nil
}

func NewCommand(config CommandConfig, packet packets.Packet, trust policy.TrustBinding, egress policy.EgressGrant) (*Command, error) {
	plan, err := DescribeCommand(config, packet)
	if err != nil {
		return nil, err
	}
	return &Command{config: config, packet: packet, plan: plan, trust: trust, egress: egress}, nil
}

func (command *Command) Plan() CommandPlan { return command.plan }

func (command *Command) Handshake(ctx context.Context) (publicschema.ProviderHandshake, error) {
	if err := command.authorize(); err != nil {
		return publicschema.ProviderHandshake{}, err
	}
	output, err := command.invoke(ctx, commandRequest{ProtocolVersion: "1.0", Operation: "handshake"})
	if err != nil {
		return publicschema.ProviderHandshake{}, err
	}
	var handshake publicschema.ProviderHandshake
	decodeErr := decodeStrict(output, &handshake)
	if decodeErr != nil || !handshake.Valid() || !contains(handshake.Capabilities, "review") || !contains(handshake.Capabilities, "structured-result") {
		return publicschema.ProviderHandshake{}, errors.Join(ErrHandshakeInvalid, decodeErr)
	}
	return handshake, nil
}

func (command *Command) Review(ctx context.Context, packet packets.Packet) (results.Result, error) {
	if packet.SnapshotID != command.packet.SnapshotID || packet.TaskID != command.packet.TaskID || packet.TaskInputHash != command.packet.TaskInputHash {
		return results.Result{}, errors.New("review packet does not match authorized command plan")
	}
	if err := command.authorize(); err != nil {
		return results.Result{}, err
	}
	handshake, err := command.Handshake(ctx)
	if err != nil {
		return results.Result{}, err
	}
	if command.plan.NetworkDestinationClass == "none" && handshake.NetworkAccess != "none" {
		return results.Result{}, errors.Join(ErrHandshakeInvalid, errors.New("Provider requires or may use network but execution plan declares none"))
	}
	reviewRequest := commandRequest{ProtocolVersion: "1.0", Operation: "review", Packet: &packet}
	if int64(len(mustJSON(reviewRequest))) > handshake.MaxInputBytes {
		return results.Result{}, errors.New("packet exceeds Provider handshake max_input_bytes")
	}
	output, err := command.invoke(ctx, reviewRequest)
	if err != nil {
		return results.Result{}, err
	}
	return results.Parse(bytes.NewReader(output))
}

func (command *Command) authorize() error {
	now := time.Now().UTC()
	if command.config.Now != nil {
		now = command.config.Now().UTC()
	}
	executable, binaryDigest, err := validateExecutable(command.config.Executable)
	if err != nil || executable != command.plan.Executable || binaryDigest != command.plan.BinaryDigest {
		return ErrCommandUnauthorized
	}
	workingDir, err := validateWorkingDir(command.config.RepositoryRoot, command.config.WorkingDir)
	if err != nil || workingDir != command.plan.WorkingDir {
		return ErrCommandUnauthorized
	}
	candidate := command.plan.TrustCandidate
	candidate.ExpiresAt = command.trust.ExpiresAt
	if !command.trust.Authorizes(candidate, now) || !command.egress.Authorizes(command.plan.EgressRequest, now) {
		return ErrCommandUnauthorized
	}
	return nil
}

func (command *Command) invoke(ctx context.Context, request commandRequest) ([]byte, error) {
	input, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	callContext, cancel := context.WithTimeout(ctx, command.config.Timeout)
	defer cancel()
	output, err := processrunner.Run(callContext, processrunner.Spec{
		Executable: command.plan.Executable, Args: command.plan.Argv, Dir: command.plan.WorkingDir,
		Env: append([]string(nil), command.config.Env...), Stdin: input,
		MaxStdout: command.config.MaxStdout, MaxStderr: command.config.MaxStderr,
	})
	if err != nil {
		return nil, fmt.Errorf("command Provider: %w", err)
	}
	return output.Stdout, nil
}

func validateExecutable(path string) (string, string, error) {
	if !filepath.IsAbs(path) {
		return "", "", errors.New("command executable must be absolute")
	}
	clean := filepath.Clean(path)
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return "", "", fmt.Errorf("resolve command executable: %w", err)
	}
	if resolved != clean {
		return "", "", errors.New("command executable must not be a symlink")
	}
	info, err := os.Stat(clean)
	if err != nil {
		return "", "", err
	}
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return "", "", errors.New("command executable must be a regular executable file")
	}
	content, err := os.ReadFile(clean)
	if err != nil {
		return "", "", err
	}
	digest := sha256.Sum256(content)
	return clean, "sha256:" + hex.EncodeToString(digest[:]), nil
}

func validateWorkingDir(repositoryRoot, workingDir string) (string, error) {
	if !filepath.IsAbs(repositoryRoot) || !filepath.IsAbs(workingDir) {
		return "", errors.New("repository root and Provider working directory must be absolute")
	}
	resolvedRepository, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return "", err
	}
	resolvedWorkingDir, err := filepath.EvalSymlinks(workingDir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolvedWorkingDir)
	if err != nil || !info.IsDir() {
		return "", errors.New("Provider working directory must exist and be a directory")
	}
	relative, err := filepath.Rel(resolvedRepository, resolvedWorkingDir)
	if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("Provider working directory must be outside the target repository")
	}
	return resolvedWorkingDir, nil
}

func environmentDigests(environment []string) (map[string]string, error) {
	result := map[string]string{}
	for _, entry := range environment {
		name, value, ok := strings.Cut(entry, "=")
		if !ok || name == "" || strings.ContainsAny(name, "\x00=") {
			return nil, errors.New("Provider environment entries must be NAME=value")
		}
		if _, duplicate := result[name]; duplicate {
			return nil, fmt.Errorf("duplicate Provider environment variable %q", name)
		}
		digest := sha256.Sum256([]byte(value))
		result[name] = "sha256:" + hex.EncodeToString(digest[:])
	}
	return result, nil
}

func decodeStrict(content []byte, target any) error {
	if !utf8.Valid(content) {
		return errors.New("Provider output is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("Provider output must contain exactly one JSON object")
	}
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func mustJSON(value any) []byte {
	content, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return content
}
