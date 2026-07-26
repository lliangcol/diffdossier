# ADR-0002: Local evidence and provider boundaries

- Status: Accepted
- Date: 2026-07-26
- Decisions: D-004, D-005, D-006, D-007, D-009, D-011

## Context

Review evidence must remain inspectable and resumable without granting a model, repository file, or provider implicit execution or mutation rights.

## Decision

- `manual` is the default provider. Automated providers are explicit opt-ins.
- Provider integration uses a versioned external-process JSON protocol. The core does not link a model SDK.
- Durable state is stored outside the target repository in schema-versioned, content-addressed files.
- Configuration precedence is CLI, allowlisted `DIFFDOSSIER_*` environment variables, repository configuration, user configuration, then built-in defaults. Secrets do not enter this chain.
- The core is read-only by default. A fix requires explicit authorization and is performed by an external actor.
- Evidence is reusable only when the source/task input, prompt, provider manifest, result schema, configuration, rules, gate definitions, and toolchain digests still match.

## Consequences

- Manual review can complete without model access or network access.
- Provider output remains untrusted input and cannot write checkpoints, run gates, authorize fixes, or determine the verdict.
- Repository content can constrain review but cannot grant network, command execution, modification, commit, push, or publication authority.
