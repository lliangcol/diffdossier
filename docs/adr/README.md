# Architecture decision records

The Phase 0 decision baseline is split into three records:

1. [ADR-0001: Project identity and distribution baseline](0001-project-identity-and-distribution.md) — D-001, D-002, D-003, D-008, D-012.
2. [ADR-0002: Local evidence and provider boundaries](0002-local-evidence-and-provider-boundaries.md) — D-004, D-005, D-006, D-007, D-009, D-011.
3. [ADR-0003: Verdict, trust, and data classification](0003-verdict-trust-and-data-classification.md) — D-010, D-013, D-014.

Decisions that depend on an explicit project gate keep that gate visible in the record. A later implementation change must supersede the relevant record instead of silently changing the contract.

## When a decision record is required

Before implementing a design that creates or changes a public contract,
dependency, network behavior, command execution, release behavior, security
boundary, or public-data handling, add an ADR or an equivalent decision record
that contains the same fields as the [ADR template](template.md). Record the
decision before implementation; a later test result, pull request, or Release
cannot retroactively replace it.

The record must state the context and evidence, chosen and rejected options,
compatibility and rollback consequences, verification plan, approving role,
and each required external gate. It does not itself authorize a commit, push,
network call, command execution, Release, publication, security disclosure, or
data export. For changes that supersede an ADR, add the new record and link the
older record instead of editing history to conceal the prior decision.
