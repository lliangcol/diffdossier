# DiffDossier

DiffDossier is a planned local-first, provider-neutral, evidence-driven orchestrator for reviewing and fixing large Git changes.

The repository is in early Phase 1 implementation. No public release, stable CLI, or platform support is claimed yet.

## Current boundaries

- The public repository is `github.com/lliangcol/diffdossier`; local changes have not been committed or pushed.
- The Go module path is `github.com/lliangcol/diffdossier`.
- The initial CLI exposes only `version` and help.
- The compatibility spike uses only the Go standard library and the local Git executable.
- No networked provider, project-defined command, automatic fix, remote write/CI workflow, or public export is enabled.

## Local verification

```text
go test ./...
go vet ./...
go run ./cmd/diffdossier version --json
```

See [the architecture decision records](docs/adr/README.md) and [the platform compatibility spike](docs/platform-compatibility.md).
