# DiffDossier

DiffDossier is a local-first, provider-neutral, evidence-driven orchestrator
for reviewing and fixing large Git changes.

The repository is in Phase 4 implementation. No public release, stable CLI, or platform support is claimed yet.

## Current boundaries

- The initial CLI foundation is published on `main` at `github.com/lliangcol/diffdossier`; no release or stability promise exists yet.
- The Go module path is `github.com/lliangcol/diffdossier`.
- The current CLI exposes version, doctor, strict config validate,
  repository-external prepare snapshots, and deterministic plan/manual packet
  generation. It can strictly import task results with `record task`.
- The internal Provider boundary includes model-free manual/mock implementations
  and an authorization-gated argv-only command protocol. No automatic Codex or
  Claude Code adapter is published.
- The compatibility spike uses only the Go standard library and the local Git executable.
- No networked provider, project-defined command, automatic fix, remote write/CI workflow, or public export is enabled.

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
```

See [configuration](docs/configuration.md), [snapshot semantics](docs/snapshots.md),
[planning and packets](docs/planning-and-packets.md), [the architecture decision
records](docs/adr/README.md), and [the platform compatibility
spike](docs/platform-compatibility.md). Provider boundaries and manual
integration status are documented in [providers and results](docs/providers-and-results.md),
[Codex](docs/integrations/codex.md), and [Claude Code](docs/integrations/claude-code.md).

## Governance and security

The project is Apache-2.0 licensed, uses DCO sign-off, and accepts only
synthetic or explicitly approved public data in this repository. Read
[CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), and
[GOVERNANCE.md](GOVERNANCE.md) before contributing.
