# GitHub governance audit

Read-only audit captured 2026-07-28. External state may change; this record is
not an authorization to modify settings.

| Area | Observed result | Evidence level / follow-up |
|---|---|---|
| Repository | Public, default branch `main`, not archived. | GitHub REST repository response. |
| Ruleset | Active `Protect main` ruleset, ID `19772012`, target `branch`; updated 2026-07-28 to require PR/rebase, thread resolution, linear history, no deletion/force push, and checks `DCO` plus `Quality and cross-build`, with zero approving reviews. | REST `PUT` then `GET` readback after repository-scoped authorization; no bypass actors. |
| Legacy branch protection | `GET branches/main/protection` returned HTTP 404 “Branch not protected”. | This does not negate the ruleset; it shows no legacy branch-protection endpoint configuration. |
| Required checks | `DCO` and `Quality and cross-build` are enforced by the ruleset. | Current configuration readback; future successful runs remain a separate DD-GOV-013 review. |
| Private Vulnerability Reporting | Enabled on 2026-07-28 after an explicit repository-scoped authorization. | `PUT` then `GET` of the REST PVR endpoint returned `enabled: true`; recheck before relying on the setting. |
| Actions permissions | Enabled; `allowed_actions: all`; SHA pinning required: false. | REST Actions permissions response; workflow files use pins but repository setting remains an audit finding. |
| Releases | beta.1, beta.2, beta.3 are public Pre-releases. | GitHub CLI Release listing. |
| CI / Release runs | Latest visible CI #30286989079 and Release #30266531344 succeeded on historical SHAs. | GitHub CLI run listing; not current-branch proof. |

No GitHub setting, Release, branch, or workflow was modified during this audit.
