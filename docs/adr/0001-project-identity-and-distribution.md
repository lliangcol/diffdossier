# ADR-0001: Project identity and distribution baseline

- Status: Accepted for local implementation; public release gates remain open
- Date: 2026-07-26
- Decisions: D-001, D-002, D-003, D-008, D-012

## Context

DiffDossier needs a provider-neutral identity, a portable implementation language, a bounded VCS model, and a reproducible distribution path. G-02 was satisfied on 2026-07-26 when the user confirmed the public repository owned by `lliangcol`.

## Decision

- Use `DiffDossier` as the working project and product name. Revisit it if a fresh registry or trademark review finds a conflict.
- Use `github.com/lliangcol/diffdossier` as the public repository and Go module path.
- Implement the core in Go and preserve a `CGO_ENABLED=0` build unless a separately reviewed requirement proves that impossible.
- Support Git only for the MVP.
- Use Apache-2.0 as the intended license, subject to the final License/NOTICE/source review in G-03 before public release.
- Target native release binaries with checksums, SBOM, and provenance. Package-manager channels are later additions.

## Consequences

- A single executable can provide the core workflow without requiring Python, Node, or a shell at runtime.
- Platform support must be established by native CI and runtime evidence, not cross-compilation alone.
- The public module path and remote are fixed. A commit, push, tag, release, or package reservation still requires its own applicable gate.
