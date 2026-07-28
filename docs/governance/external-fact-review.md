# External fact review register

This register prevents a one-time external lookup from being presented as a
permanent current fact. It complements the repository-local checks in
`internal/doccheck`: those checks must not query the network and cannot prove
GitHub state, platform availability, or a human decision.

## Review rules

- A review record names the reviewer role, source URL or API endpoint, query
  time with timezone, observed result, exact candidate SHA/version where
  applicable, and the next expiry or trigger.
- A source may be unavailable. Record that result as `unknown`, retain the
  previous record only as historical evidence, and do not perform an external
  action on its basis.
- A conflicting result stops the dependent task until the named owner resolves
  it in a new, content-bound decision. A stale result is not an approval.
- A repository document may link to a captured audit, but it must state the
  capture date and its boundary rather than silently calling the audit live.

## Required review register

| Fact | Reviewer / decision owner | Authoritative source and minimum capture | Valid until / required refresh | Unknown or conflict handling |
|---|---|---|---|---|
| Repository owner and maintainer role mapping | Liu Liang acting as maintainer; identity changes require the affected person’s explicit confirmation | GitHub repository owner field plus [`identity-and-roles.md`](identity-and-roles.md); record both query time and the human confirmation reference | 90 days, before a permission, role, copyright, DCO, or approval-routing change | Do not infer a person from a GitHub handle; stop role-dependent approvals until explicit confirmation is recorded. |
| `main` ruleset and branch protections | Repository owner/maintainer | GitHub ruleset detail and, separately, legacy branch-protection endpoint; capture rule IDs, target, enforcement and each rule parameter | 7 days, before changing rules, claiming mergeability, adding required checks, or opening a protected-branch delivery decision | Treat unavailable detail as `unknown`; do not use a list entry, prior screenshot, or a successful CI run as proof of the current configuration. |
| Required checks and their stable names | Repository owner/maintainer | Ruleset detail plus successful runs on the intended branch/SHA; capture exact job names, workflow revision and whether the rule is enforced | 7 days, before enabling a required check or declaring a PR mergeable | Do not configure or claim a required check from a historical run. Reconcile rename, missing job, or branch mismatch before any setting change. |
| Private Vulnerability Reporting (PVR) and private fallback | Repository owner/maintainer and security contact | GitHub PVR setting and `SECURITY.md`; capture enabled/disabled state, fallback route and confirmation that it is private | 30 days, before publishing a security entrypoint, changing PVR, or relying on the fallback for a report | If PVR/fallback privacy cannot be read back, state `unknown` and do not invite sensitive reports through an unverified public route. |
| Releases, tags, assets, checksums, SBOM and provenance | Release decision owner | GitHub Release/tag pages or API plus the candidate commit and asset hashes; record query time and artifact-specific verification separately | Before every Release decision, publication, rollback or support statement; prior list captures expire after 7 days | A visible release is not asset, attestation, installation, support, or current-branch proof. Missing/changed data blocks the corresponding release task. |
| CI and Release workflow result | Maintainer for local workflow changes; release owner for publication gates | GitHub Actions run URL, workflow revision, exact SHA, conclusion and relevant job names | Before relying on a run for merge/release evidence; re-check after any workflow, source, runner, toolchain, or ruleset change | Historical successful runs remain historical. If current run data is unavailable, record `unknown` rather than carrying forward PASS. |
| Platform evidence and support tier | Maintainer; a support-tier declaration additionally needs explicit owner decision | Matching native runner/run, exact artifact hash, native install/smoke record, and [`platform-evidence-matrix.md`](../platform-evidence-matrix.md) | Before a Tier declaration, supported-install claim, release decision, or any platform-bound troubleshooting guidance; otherwise review every 30 days | Cross-build and artifact existence do not substitute for native execution. Missing platform evidence leaves the tier unassigned. |

## Current captured records

The records below are dated evidence, not live state:

- [`github-governance-audit.md`](github-governance-audit.md) captures public
  repository, ruleset, PVR, Actions and historic run observations on
  2026-07-28.
- [`../release-evidence-inventory.md`](../release-evidence-inventory.md) and
  [`../beta3-artifact-verification.md`](../beta3-artifact-verification.md)
  distinguish visible Releases from asset-level verification.
- [`../platform-evidence-matrix.md`](../platform-evidence-matrix.md) preserves
  runner and artifact boundaries and leaves support tiers unassigned.

None of these records authorizes a GitHub setting change, Release, merge,
publication, support promise, or security disclosure.
