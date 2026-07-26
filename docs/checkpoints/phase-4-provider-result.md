# Phase 4 Provider and Result checkpoint

- Date: 2026-07-26
- Scope: T4.1, T4.2, T4.3, and T4.6 implementation; T4.4/T4.5 manual-only evidence
- Automatic Codex/Claude Code adapters: not published
- Live Provider calls: not run

## Implemented

- Model-free manual and mock Providers.
- Argv-only command Provider with repository-external cwd, exact environment,
  executable hashing and revalidation, command-plan digest, trust binding,
  egress binding, timeout, bounded output, process-tree cancellation, strict
  handshake, and no strong-sandbox claim.
- Strict 8 MiB UTF-8 JSON Result parser, exact task/snapshot/input binding,
  evidence-bound coverage, Provider-limited finding states, required
  perspectives, immutable pass index, heterogeneous comparison, and
  `record task` workflow.
- Current official capability and terms evidence for manual Codex and Claude
  Code integration.

## Verified locally

- Unauthorized and changed-binary command plans produce zero Provider
  execution.
- Valid fake command handshake and review succeed.
- Incompatible handshake, timeout, oversized output, malformed Result,
  unknown fields, stale bindings, extra coverage, and Provider-promoted
  findings fail closed.
- Manual `prepare -> plan -> record task` reaches `REVIEWED` only after all
  required task perspectives complete.
- No model or Provider network call is part of these tests.

## Not verified

- G-09 approval or an automatic Codex/Claude Code adapter.
- G-05/G-10 live Provider execution, credentials, quota, latency, or billing.
- Native Windows Job Object termination and native Windows/Linux/macOS-arm64
  execution.
- Strong OS-level network or filesystem isolation.
