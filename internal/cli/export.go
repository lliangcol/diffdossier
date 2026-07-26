package cli

import (
	"context"
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

	"github.com/lliangcol/diffdossier/internal/exporter"
	"github.com/lliangcol/diffdossier/internal/gitrepo"
	"github.com/lliangcol/diffdossier/internal/platform"
	"github.com/lliangcol/diffdossier/internal/policy"
	"github.com/lliangcol/diffdossier/internal/snapshot"
	"github.com/lliangcol/diffdossier/internal/store"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

func runExport(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "portable" {
		return runExportPortable(args[1:], stdout, stderr)
	}
	if len(args) > 1 && args[0] == "public" && args[1] == "prepare" {
		return runExportPublicPrepare(args[2:], stdout, stderr)
	}
	if len(args) > 1 && args[0] == "public" && args[1] == "approve" {
		return runExportPublicApprove(args[2:], stdout, stderr)
	}
	if len(args) > 1 && args[0] == "public" && args[1] == "create" {
		return runExportPublicCreate(args[2:], stdout, stderr)
	}
	if len(args) > 1 && args[0] == "public" && args[1] == "revoke" {
		return runExportPublicRevoke(args[2:], stdout, stderr)
	}
	fmt.Fprintln(stderr, "usage: diffdossier export portable ... | diffdossier export public prepare|approve|create|revoke ...")
	return ExitUsage
}

type exportContext struct {
	stateStore *store.Store
	repository store.Repository
	run        store.Run
	seal       snapshot.Seal
	runDir     string
	repoRoot   string
}

func resolveExportContext(repoPath, stateRoot, runID string) (exportContext, error) {
	repo, err := gitrepo.Open(nilContext(), repoPath)
	if err != nil {
		return exportContext{}, err
	}
	if stateRoot == "" {
		paths, pathErr := platform.DefaultPaths()
		if pathErr != nil {
			return exportContext{}, pathErr
		}
		stateRoot = paths.StateDir
	}
	if !filepath.IsAbs(stateRoot) {
		return exportContext{}, errors.New("state-dir must be absolute")
	}
	if err := requireOutsideRepository(repo.Root, stateRoot); err != nil {
		return exportContext{}, err
	}
	stateStore, err := store.Open(stateRoot)
	if err != nil {
		return exportContext{}, err
	}
	repository, err := stateStore.Register(repo.Root)
	if err != nil {
		return exportContext{}, err
	}
	if runID == "" {
		latest, latestErr := stateStore.LatestRun(repository.ID)
		if latestErr != nil {
			return exportContext{}, latestErr
		}
		runID = latest.ID
	}
	run, seal, err := stateStore.LoadRun(repository.ID, runID)
	if err != nil {
		return exportContext{}, err
	}
	runDir, err := stateStore.RunDir(repository.ID, run.ID)
	if err != nil {
		return exportContext{}, err
	}
	return exportContext{stateStore: stateStore, repository: repository, run: run, seal: seal, runDir: runDir, repoRoot: repo.Root}, nil
}

func runExportPortable(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("export portable", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "target Git repository")
	state := flags.String("state-dir", "", "durable state directory")
	run := flags.String("run-id", "", "run ID")
	output := flags.String("output", "", "new portable ZIP path")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || *output == "" {
		return ExitUsage
	}
	absolute, err := filepath.Abs(*output)
	if err != nil {
		return ExitUsage
	}
	context, err := resolveExportContext(*repo, *state, *run)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_CONTEXT", err.Error()), ExitEvidence)
	}
	if err := requireOutsideRepository(context.repoRoot, absolute); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", "portable export output must be outside target repository"), ExitUsage)
	}
	content, manifest, err := exporter.Portable(context.runDir)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_PORTABLE", err.Error()), ExitEvidence)
	}
	if err := writeExclusiveBytes(absolute, content); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_WRITE", err.Error()), ExitEvidence)
	}
	runState := context.run.State
	if context.run.State == "FINALIZED" {
		if _, err := context.stateStore.AppendEvent(context.runDir, "portable_exported", map[string]any{"run_digest": manifest.RunDigest, "bytes": len(content)}); err != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_EVENT", err.Error()), ExitEvidence)
		}
		updated, err := context.stateStore.UpdateRunState(context.runDir, "EXPORTED")
		if err != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_WORKFLOW_STATE", err.Error()), ExitEvidence)
		}
		runState = updated.State
	}
	data := map[string]any{"run_id": context.run.ID, "output": absolute, "run_digest": manifest.RunDigest, "bytes": len(content), "state": runState}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "portable export written: %s (%s)\n", absolute, manifest.RunDigest)
	return ExitOK
}

func runExportPublicPrepare(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("export public prepare", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "target Git repository")
	state := flags.String("state-dir", "", "durable state directory")
	run := flags.String("run-id", "", "run ID")
	input := flags.String("input", "", "candidate input")
	class := flags.String("class", "", "public_synthetic, public_project, or private_project")
	action := flags.String("action", "create", "create or replace")
	policyDigest := flags.String("policy-digest", "", "policy digest")
	revision := flags.String("public-revision", "", "confirmed public revision")
	redaction := flags.String("redaction-approval-digest", "", "private redaction approval digest")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || *input == "" || *class == "" || *policyDigest == "" {
		return ExitUsage
	}
	context, err := resolveExportContext(*repo, *state, *run)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_CONTEXT", err.Error()), ExitEvidence)
	}
	content, err := os.ReadFile(*input)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_READ", err.Error()), ExitEvidence)
	}
	preparation, err := exporter.PreparePublic(content, publicschema.DataClass(*class), *action, *policyDigest, *revision, *redaction)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PUBLIC_PREPARE", err.Error()), ExitEvidence)
	}
	preparationDigest := exporter.PreparationDigest(preparation)
	preparationPath := filepath.Join("exports", "public", "preparations", digestHex(preparationDigest)+".json")
	candidatePath := filepath.Join("exports", "public", "candidates", digestHex(preparation.Candidate.Digest)+".bin")
	if err := context.stateStore.WriteRunBytes(context.runDir, candidatePath, content); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
	}
	if err := context.stateStore.WriteRunJSON(context.runDir, preparationPath, preparation); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
	}
	if _, err := context.stateStore.AppendEvent(context.runDir, "public_export_prepared", map[string]any{"preparation_digest": preparationDigest, "candidate_digest": preparation.Candidate.Digest, "action": preparation.Candidate.Action, "scan_findings": len(preparation.ScanFindings)}); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_EVENT", err.Error()), ExitEvidence)
	}
	data := map[string]any{"run_id": context.run.ID, "preparation_digest": preparationDigest, "preparation": preparation, "approval_required": true, "bundle_created": false}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "public candidate prepared: %s (%d scan findings); approval required, no bundle created\n", preparationDigest, len(preparation.ScanFindings))
	return ExitOK
}

func runExportPublicApprove(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("export public approve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "target Git repository")
	state := flags.String("state-dir", "", "durable state directory")
	runID := flags.String("run-id", "", "run ID")
	preparationDigest := flags.String("preparation-digest", "", "exact preparation digest")
	operator := flags.String("operator", "", "accountable approver")
	redactionFile := flags.String("redaction-approval", "", "private redaction approval JSON")
	trust := flags.String("trust-public-approval", "", "exact displayed approval plan digest")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || *preparationDigest == "" || strings.TrimSpace(*operator) == "" {
		return ExitUsage
	}
	context, err := resolveExportContext(*repo, *state, *runID)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_CONTEXT", err.Error()), ExitEvidence)
	}
	preparation, err := loadPublicPreparation(context, *preparationDigest)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PUBLIC_PREPARATION", err.Error()), ExitEvidence)
	}
	var redactionApproval *policy.RedactionApproval
	if preparation.Candidate.ArtifactClass == policy.ArtifactRedactedSummary {
		if *redactionFile == "" {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_REDACTION_APPROVAL", "redacted summary requires --redaction-approval"), ExitEvidence)
		}
		var approval policy.RedactionApproval
		if err := readStrictJSON(*redactionFile, &approval); err != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_REDACTION_APPROVAL", err.Error()), ExitEvidence)
		}
		if err := approval.Authorizes(preparation.Candidate); err != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_REDACTION_APPROVAL", err.Error()), ExitEvidence)
		}
		redactionApproval = &approval
	}
	planDigest := exporter.ApprovalPlanDigest(preparation, strings.TrimSpace(*operator))
	if *trust == "" {
		data := map[string]any{"approved": false, "authorization_required": true, "approval_plan_digest": planDigest, "preparation_digest": *preparationDigest, "candidate": preparation.Candidate}
		if *jsonOutput {
			return writeJSON(stdout, stderr, publicschema.Success(data))
		}
		fmt.Fprintf(stdout, "public approval plan %s; no approval created\n", planDigest)
		return ExitOK
	}
	if *trust != planDigest {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PUBLIC_UNAUTHORIZED", "trust-public-approval does not match exact approval plan"), ExitEvidence)
	}
	if len(preparation.ScanFindings) > 0 {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PUBLIC_SCAN", "candidate scan has findings"), ExitEvidence)
	}
	approval, err := policy.NewPublicApproval(preparation.Candidate, strings.TrimSpace(*operator), time.Now().UTC())
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PUBLIC_APPROVAL", err.Error()), ExitEvidence)
	}
	if redactionApproval != nil {
		path := filepath.Join("approvals", "redaction-"+digestHex(redactionApproval.Digest)+".json")
		if err := context.stateStore.WriteRunJSON(context.runDir, path, redactionApproval); err != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
		}
	}
	approvalPath := filepath.Join("approvals", "public-"+digestHex(approval.Digest)+".json")
	if err := context.stateStore.WriteRunJSON(context.runDir, approvalPath, approval); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
	}
	if _, err := context.stateStore.AppendEvent(context.runDir, "public_export_approved", map[string]string{"preparation_digest": *preparationDigest, "approval_digest": approval.Digest, "approval_plan_digest": planDigest}); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_EVENT", err.Error()), ExitEvidence)
	}
	data := map[string]any{"approved": true, "approval_digest": approval.Digest, "approval_plan_digest": planDigest}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "public export approved: %s\n", approval.Digest)
	return ExitOK
}

func runExportPublicCreate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("export public create", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "target Git repository")
	state := flags.String("state-dir", "", "durable state directory")
	runID := flags.String("run-id", "", "run ID")
	preparationDigest := flags.String("preparation-digest", "", "exact preparation digest")
	approvalDigest := flags.String("approval-digest", "", "exact private approval digest")
	output := flags.String("output", "", "new public bundle JSON path")
	trust := flags.String("trust-public-create", "", "exact displayed create plan digest")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || *preparationDigest == "" || *approvalDigest == "" || *output == "" {
		return ExitUsage
	}
	context, err := resolveExportContext(*repo, *state, *runID)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_CONTEXT", err.Error()), ExitEvidence)
	}
	absoluteOutput, err := filepath.Abs(*output)
	if err != nil || requireOutsideRepository(context.repoRoot, absoluteOutput) != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", "public output must be outside target repository"), ExitUsage)
	}
	preparation, err := loadPublicPreparation(context, *preparationDigest)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PUBLIC_PREPARATION", err.Error()), ExitEvidence)
	}
	var approval policy.PublicApproval
	if !validSHA256(*approvalDigest) {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PUBLIC_APPROVAL", "invalid approval digest"), ExitUsage)
	}
	if err := context.stateStore.ReadRunJSON(context.runDir, filepath.Join("approvals", "public-"+digestHex(*approvalDigest)+".json"), &approval); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PUBLIC_APPROVAL", err.Error()), ExitEvidence)
	}
	planDigest := exporter.CreatePlanDigest(preparation, approval)
	if *trust == "" {
		data := map[string]any{"created": false, "authorization_required": true, "create_plan_digest": planDigest, "preparation_digest": *preparationDigest, "approval_digest": approval.Digest}
		if *jsonOutput {
			return writeJSON(stdout, stderr, publicschema.Success(data))
		}
		fmt.Fprintf(stdout, "public create plan %s; no bundle created\n", planDigest)
		return ExitOK
	}
	if *trust != planDigest {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PUBLIC_UNAUTHORIZED", "trust-public-create does not match exact create plan"), ExitEvidence)
	}
	var redactionApproval *policy.RedactionApproval
	if preparation.Candidate.ArtifactClass == policy.ArtifactRedactedSummary {
		var value policy.RedactionApproval
		digest := preparation.Candidate.RedactionApprovalDigest
		if !validSHA256(digest) {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_REDACTION_APPROVAL", "invalid bound redaction approval digest"), ExitEvidence)
		}
		if err := context.stateStore.ReadRunJSON(context.runDir, filepath.Join("approvals", "redaction-"+digestHex(digest)+".json"), &value); err != nil {
			return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_REDACTION_APPROVAL", err.Error()), ExitEvidence)
		}
		redactionApproval = &value
	}
	bundle, err := exporter.CreatePublic(preparation, approval, redactionApproval)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PUBLIC_CREATE", err.Error()), ExitEvidence)
	}
	if err := writeExclusiveJSON(absoluteOutput, bundle); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_WRITE", err.Error()), ExitEvidence)
	}
	bundleRelative := "exports/public-bundle-" + digestHex(bundle.BundleDigest) + ".json"
	if err := context.stateStore.WriteRunJSON(context.runDir, bundleRelative, bundle); err != nil {
		_ = os.Remove(absoluteOutput)
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
	}
	if _, err := context.stateStore.AppendEvent(context.runDir, "public_export_created", map[string]string{"bundle_digest": bundle.BundleDigest, "approval_digest": approval.Digest, "create_plan_digest": planDigest}); err != nil {
		_ = context.stateStore.RemoveRunArtifact(context.runDir, bundleRelative)
		_ = os.Remove(absoluteOutput)
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_EVENT", err.Error()), ExitEvidence)
	}
	data := map[string]any{"created": true, "bundle_digest": bundle.BundleDigest, "output": absoluteOutput, "approval_record_digest": bundle.ApprovalRecordDigest}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "public bundle created: %s\n", bundle.BundleDigest)
	return ExitOK
}

func runExportPublicRevoke(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("export public revoke", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "target Git repository")
	state := flags.String("state-dir", "", "durable state directory")
	runID := flags.String("run-id", "", "run ID")
	approvalDigest := flags.String("approval-digest", "", "exact public approval record digest from the bundle")
	exportDigest := flags.String("export-digest", "", "exact public bundle digest")
	reason := flags.String("reason", "", "revocation reason")
	output := flags.String("output", "", "new public tombstone JSON path")
	trust := flags.String("trust-public-revoke", "", "exact displayed revoke plan digest")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 || !validSHA256(*approvalDigest) || !validSHA256(*exportDigest) || strings.TrimSpace(*reason) == "" || *output == "" {
		return ExitUsage
	}
	context, err := resolveExportContext(*repo, *state, *runID)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_CONTEXT", err.Error()), ExitEvidence)
	}
	absoluteOutput, err := filepath.Abs(*output)
	if err != nil || requireOutsideRepository(context.repoRoot, absoluteOutput) != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", "tombstone output must be outside target repository"), ExitUsage)
	}
	var bundle exporter.PublicBundle
	if err := context.stateStore.ReadRunJSON(context.runDir, "exports/public-bundle-"+digestHex(*exportDigest)+".json", &bundle); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PUBLIC_BUNDLE", err.Error()), ExitEvidence)
	}
	if err := exporter.VerifyPublicBundle(bundle); err != nil || bundle.BundleDigest != *exportDigest || bundle.ApprovalRecordDigest != *approvalDigest {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PUBLIC_BUNDLE", "bundle does not match exact export and approval digests"), ExitEvidence)
	}
	planDigest := exporter.RevocationPlanDigest(*approvalDigest, *exportDigest, strings.TrimSpace(*reason))
	if *trust == "" {
		data := map[string]any{"revoked": false, "authorization_required": true, "revoke_plan_digest": planDigest, "external_copies_recalled": false}
		if *jsonOutput {
			return writeJSON(stdout, stderr, publicschema.Success(data))
		}
		fmt.Fprintf(stdout, "public revoke plan %s; no tombstone created and external copies cannot be recalled\n", planDigest)
		return ExitOK
	}
	if *trust != planDigest {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PUBLIC_UNAUTHORIZED", "trust-public-revoke does not match exact revoke plan"), ExitEvidence)
	}
	tombstone, err := exporter.Revoke(*approvalDigest, *exportDigest, strings.TrimSpace(*reason), time.Now().UTC())
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PUBLIC_REVOKE", err.Error()), ExitEvidence)
	}
	if err := writeExclusiveJSON(absoluteOutput, tombstone); err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_EXPORT_WRITE", err.Error()), ExitEvidence)
	}
	tombstoneRelative := "exports/public-tombstone-" + digestHex(tombstone.TombstoneDigest) + ".json"
	if err := context.stateStore.WriteRunJSON(context.runDir, tombstoneRelative, tombstone); err != nil {
		_ = os.Remove(absoluteOutput)
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_WRITE", err.Error()), ExitEvidence)
	}
	if _, err := context.stateStore.AppendEvent(context.runDir, "public_export_revoked", map[string]string{"bundle_digest": *exportDigest, "approval_digest": *approvalDigest, "tombstone_digest": tombstone.TombstoneDigest, "revoke_plan_digest": planDigest}); err != nil {
		_ = context.stateStore.RemoveRunArtifact(context.runDir, tombstoneRelative)
		_ = os.Remove(absoluteOutput)
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_STATE_EVENT", err.Error()), ExitEvidence)
	}
	data := map[string]any{"revoked": true, "tombstone": tombstone, "output": absoluteOutput, "external_copies_recalled": false}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(data))
	}
	fmt.Fprintf(stdout, "public tombstone created: %s; external copies cannot be recalled\n", tombstone.TombstoneDigest)
	return ExitOK
}

func loadPublicPreparation(context exportContext, digest string) (exporter.PublicPreparation, error) {
	if !validSHA256(digest) {
		return exporter.PublicPreparation{}, errors.New("invalid preparation digest")
	}
	var preparation exporter.PublicPreparation
	path := filepath.Join("exports", "public", "preparations", digestHex(digest)+".json")
	if err := context.stateStore.ReadRunJSON(context.runDir, path, &preparation); err != nil {
		return exporter.PublicPreparation{}, err
	}
	if !validSHA256(preparation.Candidate.Digest) {
		return exporter.PublicPreparation{}, errors.New("invalid candidate digest")
	}
	content, err := os.ReadFile(filepath.Join(context.runDir, "exports", "public", "candidates", digestHex(preparation.Candidate.Digest)+".bin"))
	if err != nil {
		return exporter.PublicPreparation{}, err
	}
	var dataClass publicschema.DataClass
	switch preparation.Candidate.ArtifactClass {
	case policy.ArtifactPublicSynthetic:
		dataClass = publicschema.PublicSynthetic
	case policy.ArtifactPublicProject:
		dataClass = publicschema.PublicProject
	case policy.ArtifactRedactedSummary:
		dataClass = publicschema.PrivateProject
	default:
		return exporter.PublicPreparation{}, errors.New("unknown public artifact class")
	}
	rebuilt, err := exporter.PreparePublic(content, dataClass, preparation.Candidate.Action, preparation.Candidate.PolicyDigest, preparation.Candidate.PublicRevision, preparation.Candidate.RedactionApprovalDigest)
	if err != nil || exporter.PreparationDigest(rebuilt) != digest || !sameCanonicalJSON(rebuilt, preparation) {
		return exporter.PublicPreparation{}, errors.New("public preparation does not match stored content")
	}
	rebuilt.PreparedContent = content
	return rebuilt, nil
}

func writeExclusiveJSON(path string, value any) error {
	content, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return writeExclusiveBytes(path, append(content, '\n'))
}

func writeExclusiveBytes(path string, content []byte) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".diffdossier-export-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Link(temporaryPath, path); err != nil {
		return err
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func digestHex(value string) string { return strings.TrimPrefix(value, "sha256:") }

// nilContext avoids inventing cancellation semantics for short, local-only
// repository discovery.
func nilContext() context.Context { return context.Background() }
