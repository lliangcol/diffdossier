# Public reproducible-case selection criteria

All criteria are mandatory. Passing automated screening does not replace the
required security and license review before a case is selected or published.

| Criterion | Required evidence | Reject when |
|---|---|---|
| Public source and license | Canonical upstream URL, license text/SPDX identifier, redistribution analysis. | License is absent, incompatible, unclear, or source is not public. |
| Immutable input | Full base/head SHA, fetched-object verification, and reconstruction instructions. | Branch-only reference, mutable archive, or unavailable objects. |
| Sufficient size | File count, changed lines, and byte count recorded from the frozen diff. | Scope is too small to exercise the claimed large-change workflow. |
| Cross-module risk | At least two independently owned modules/contracts and a concrete dependency, data-flow, security, or migration risk. | Cosmetic-only or single-module churn. |
| Redistribution | License and repository terms permit the exact fixture/report/media derivative planned. | Redistribution, screenshot, or report rights are uncertain. |
| Privacy and security | Source, paths, commits, issue context, generated reports, and media are scanned for personal, credential, private, or non-public data. | Any sensitive or unapproved data is present or cannot be confidently excluded. |

## Review record required before DD-CASE-001 completion

Review decision, 2026-07-28: the maintainer confirmed this candidate-neutral
standard for security and license screening, with no waiver. It applies only
to selection criteria; each future case must independently prove its license,
fixed commits, redistribution rights, and privacy/security scan.

This decision selects no repository and authorizes no download, redistribution,
public derivative, Provider call, or external contact.
