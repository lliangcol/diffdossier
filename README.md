# DiffDossier

DiffDossier is a local-first, provider-neutral, evidence-driven orchestrator
for reviewing and fixing large Git changes.

The repository has completed local Phase 7 release-preparation engineering. No
public release, stable CLI, or Tier 1 platform support is claimed yet.

## Current boundaries

- The initial CLI foundation is published on `main` at `github.com/lliangcol/diffdossier`; no release or stability promise exists yet.
- The Go module path is `github.com/lliangcol/diffdossier`.
- The current CLI exposes version, doctor, strict config validate,
  repository-external prepare snapshots, and deterministic plan/manual packet
  generation. It can strictly import task results, govern findings and fix
  authorization, refresh invalidated work, plan trusted Gates, verify/finalize
  evidence, create private portable exports, archive terminal runs, and execute
  content-bound retention plans. Public prepare/approve/create/revoke commands
  exist behind separate exact G-12 plan echoes and remain zero-action by
  default.
- The internal Provider boundary includes model-free manual/mock implementations
  and an authorization-gated argv-only command protocol. No automatic Codex or
  Claude Code adapter is published.
- The compatibility spike uses only the Go standard library and the local Git executable.
- Project-defined Gate execution requires an exact, one-run plan digest and a
  separate shell-mode acknowledgement. No automatic fix, remote write/CI
  workflow, public bundle action, or networked Provider is enabled by default.
- Security/reliability controls now include bounded Git/blob capture, log
  redaction manifests, event-journal/run-state integrity checks, stale-lock
  recovery, Unix process-group termination, and a Windows Job Object
  implementation. Native Windows/Linux/macOS-arm64 verification is still
  pending.
- Interrupted write-ahead transitions can be resumed only with
  `recover --trust-journal-state <exact-state>`; recovery does not guess.
- `gc` is a dry-run by default. Destructive retention requires the exact
  `--trust-gc-plan` digest; unexported, pinned, shared-blob, and
  public-export-evidence runs are protected.
- Maintainer tooling can prepare deterministic multi-platform candidate
  archives, checksums, a minimal SPDX SBOM, and unsigned local provenance. A
  real tag, GitHub attestation, Actions workflow, and Release remain gated.

## Local verification

```text
go test ./...
go vet ./...
go run ./cmd/diffdossier version --json
go run ./cmd/diffdossier doctor --json
go run ./cmd/diffdossier config validate --repo . --config diffdossier.example.toml --json
go run ./cmd/diffdossier prepare --repo . --config diffdossier.example.toml --state-dir /absolute/private/state --json
go run ./cmd/diffdossier plan --repo . --config diffdossier.example.toml --state-dir /absolute/private/state --json
go run ./cmd/diffdossier record task --repo . --state-dir /absolute/private/state --task-id task-... --result /absolute/result.json --json
go run ./cmd/diffdossier gates plan --repo . --state-dir /absolute/private/state --json
go run ./cmd/diffdossier verify --repo . --state-dir /absolute/private/state --json
go run ./cmd/diffdossier finalize --repo . --state-dir /absolute/private/state --json
go run ./cmd/diffdossier export portable --repo . --state-dir /absolute/private/state --output /absolute/private/run.zip --json
go run ./cmd/diffdossier run archive --repo . --state-dir /absolute/private/state --reason "retention" --json
go run ./cmd/diffdossier gc plan --repo . --state-dir /absolute/private/state --json
# Review the dry-run, then explicitly execute its exact digest:
go run ./cmd/diffdossier gc run --state-dir /absolute/private/state --trust-gc-plan sha256:... --json
```

See [configuration](docs/configuration.md), [snapshot semantics](docs/snapshots.md),
[planning and packets](docs/planning-and-packets.md), [the architecture decision
records](docs/adr/README.md), and [the platform compatibility
spike](docs/platform-compatibility.md). Provider boundaries and manual
integration status are documented in [providers and results](docs/providers-and-results.md),
[Codex](docs/integrations/codex.md), and [Claude Code](docs/integrations/claude-code.md).
The Phase 5 workflow, Gate, report, and export contracts are described in
[workflow, Gates, and reporting](docs/workflow-gates-reporting.md).
Release preparation and install verification are documented in
[the release process](docs/release-process.md) and [install guide](docs/install.md).

## Governance and security

The project is Apache-2.0 licensed, uses DCO sign-off, and accepts only
synthetic or explicitly approved public data in this repository. Read
[CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), and
[GOVERNANCE.md](GOVERNANCE.md) before contributing.
Current support boundaries are in [SUPPORT.md](SUPPORT.md).
