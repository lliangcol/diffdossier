# ADR-0003: Verdict, trust, and data classification

- Status: Accepted
- Date: 2026-07-26
- Decisions: D-010, D-013, D-014

## Context

A review report must not be confused with merge approval, and repository-provided instructions must not expand operator authority or leak private inputs.

## Decision

- Canonical verdicts are `ready`, `not_ready`, `needs_confirmation`, and `not_reviewable`.
- Operator requests and confirmations outrank scoped project rules. Project rules may constrain work but cannot grant execution, egress, mutation, commit, push, or publication rights.
- Inputs are classified as `public_synthetic`, `public_project`, `private_project`, or `secret_denied`; a packet inherits the most restrictive included classification.
- `ready` only means the declared local scope satisfies its coverage, finding, and blocking-gate requirements. It never means merged, remotely approved, deployed, or defect-free.

## Consequences

- Ambiguous authority or evidence fails closed to `needs_confirmation` or `not_reviewable`.
- Public artifacts require a separate content-bound approval; private source objects and approval identities are not embedded in the public artifact.
- `secret_denied` content never enters a review packet.
