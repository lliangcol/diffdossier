# Maintainers

## Current transition model

Liu Liang is the sole current maintainer and repository decision owner. This
is a transitional operating model, not evidence of independent review.

## Decisions and review

- Routine repository decisions may be made by the maintainer after recording
  scope and validation.
- Security, release, dependency, network, command-execution, public-data,
  compatibility and permission changes require a content-bound maintainer
  decision under `GOVERNANCE.md` and the applicable ADR/authorization rule.
- An automated agent, Provider, repository ownership, or a passing check cannot
  substitute for an independent human approval.
- The current ruleset intentionally requires zero approving reviews. A required
  independent approval is enabled only after a second qualified reviewer is
  available and has completed the review process below.

## Succession, absence, and conflicts

- No successor is currently designated. The maintainer reviews this document
  every 90 days and after any material change in availability.
- If the maintainer is unavailable, no new Release, privilege grant, security
  disclosure, public export, or external Provider approval may be inferred or
  delegated by automation; urgent security reports use the private channel in
  `SECURITY.md` until an authorized human takes responsibility.
- A maintainer with a personal, financial, employment, or authorship conflict
  records it and seeks an independent qualified reviewer before accepting the
  affected risk or release decision. If none is available, the decision remains
  deferred rather than being called independent.

## Adding an independent reviewer

A candidate must be explicitly invited under a separate authorization, receive
the minimum repository permission, and demonstrate review of a security,
compatibility, or Release change. Only then may governance and ruleset approval
requirements be reconsidered.

## Independent-approval trigger

The project does not presently require an approving review because no second
qualified reviewer is available. Reconsider that requirement only after a
named reviewer has been authorized, has successfully completed an independent
security, compatibility, or Release review, and the maintainer records the
role, scope, permissions, revocation path, and review date. Until then, a sole
maintainer’s content-bound decision remains auditable but is not described as
independent approval.
