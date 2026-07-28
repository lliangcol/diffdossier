# ADR-NNNN: <short, imperative decision title>

- Status: Proposed | Accepted | Superseded by ADR-NNNN | Rejected
- Date: YYYY-MM-DD
- Decision owner: <role; name only when approved for publication>
- Related plan/task: <DD-... or issue/PR>
- Decision class: public contract | dependency | network | command execution | release | security | public data

## Context

Describe the user or operator problem, affected components, current behavior,
and the evidence snapshot (including commit SHA, data classification, and any
external facts with their query time). State what is unknown. Do not include
credentials, private paths, private source, or unapproved data.

## Decision

State the selected design, its precise boundary, and the non-goals. For a
public contract, identify versions and compatibility policy. For a dependency,
network, command-execution, release, security, or public-data decision,
identify the default-deny behavior and the explicit opt-in or authorization
boundary.

## Alternatives considered

List the viable alternatives, why each was not selected, and any conditions
that would make the decision need revisiting.

## Consequences and verification

List positive and negative consequences, migration/rollback behavior, tests or
review required before implementation, and the evidence that will show the
decision is implemented as written. Link follow-up tasks; an ADR is not an
implementation, approval, commit, push, Release, Provider invocation, or
public-data authorization.

## Required approvals and external gates

Name the approving role, exact scope, and expiry (if applicable). Enumerate
each external write or sensitive action separately—such as network egress,
command execution, GitHub setting change, tag, Release, package publication,
security disclosure, or public export. Record `none` only when no such action
is in scope; do not infer an approval from this ADR.
