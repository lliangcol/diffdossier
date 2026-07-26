# DiffDossier

DiffDossier is a local-first, provider-neutral, evidence-driven orchestrator
for reviewing and fixing large Git changes.

The repository is in early Phase 1 implementation. No public release, stable CLI, or platform support is claimed yet.

## Current boundaries

- The initial CLI foundation is published on `main` at `github.com/lliangcol/diffdossier`; no release or stability promise exists yet.
- The Go module path is `github.com/lliangcol/diffdossier`.
- The current CLI exposes version, doctor, and strict config validate; the
  remaining workflow is under active implementation.
- The compatibility spike uses only the Go standard library and the local Git executable.
- No networked provider, project-defined command, automatic fix, remote write/CI workflow, or public export is enabled.

## Local verification

```text
go test ./...
go vet ./...
go run ./cmd/diffdossier version --json
go run ./cmd/diffdossier doctor --json
go run ./cmd/diffdossier config validate --repo . --config diffdossier.example.toml --json
```

See [configuration](docs/configuration.md), [the architecture decision
records](docs/adr/README.md), and [the platform compatibility
spike](docs/platform-compatibility.md).

## Governance and security

The project is Apache-2.0 licensed, uses DCO sign-off, and accepts only
synthetic or explicitly approved public data in this repository. Read
[CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), and
[GOVERNANCE.md](GOVERNANCE.md) before contributing.
