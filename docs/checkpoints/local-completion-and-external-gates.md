# Local implementation completion and external gates
- Status: historical
- Captured-at: 2026-07-27T01:19:10+08:00
- Source-commit: uncommitted at 98856de56e6543b4806d899c611a2744f76686c0
- Superseded-by: none
- Current-state notice: Historical checkpoint — do not treat this document as current project status. Revalidate local completion, external gates, Release, CI, and ruleset claims from their current evidence records.

- Date: 2026-07-27
- Pushed base commit: `98856de56e6543b4806d899c611a2744f76686c0`
- Local implementation status: completion candidate; current follow-up slice is uncommitted
- Public release status: authorized by maintainer; not yet created

This checkpoint separates completed local engineering from legal, platform,
network, publication, and downstream evidence that cannot be inferred from a
passing local test suite. On 2026-07-27, maintainer `liuliang1` explicitly
authorized the G-03, G-05, G-07 through G-10, and G-12 actions described in the
governed execution plan. Authorization permits execution; it does not turn an
unrun check into passing evidence.

## Verified local evidence

The pushed base commit is present on `main` and `origin/main`. The current
working-tree completion slice adds safe initialization, effective layered
configuration, path overrides, and rule-aware diagnostics. It remains
uncommitted while the newly authorized governance and workflow files are added
and the expanded candidate receives a fresh seal. The earlier candidate passed:

```text
GOWORK=off GOTOOLCHAIN=local GOPROXY=off go test -count=1 ./...
GOWORK=off GOTOOLCHAIN=local GOPROXY=off go test -race -count=1 ./...
GOWORK=off GOTOOLCHAIN=local GOPROXY=off go vet ./...
git diff --check
jq empty schemas/*.json
```

All packages passed. The same source cross-built with `CGO_ENABLED=0` for
darwin, linux, and windows on amd64 and arm64. Cross-build success remains
compile evidence rather than native runtime evidence.

The current review-fix pass found and closed two additional fail-closed gaps:
`doctor` now diagnoses state/cache paths inside the repository and resolves the
baseline without fetching, while `packet task` refuses stale source or
configuration before disclosing an old packet. The earlier implementation seal
was invalidated when the authorized governance and workflow files were added; a
fresh seal must bind the expanded candidate before any commit is created.

## Implemented scope

- Deterministic Git intake, byte-accurate inventory, snapshots, durable state,
  crash recovery, locking, and stale-evidence invalidation.
- Safe command-free configuration initialization with an explicit local
  baseline, exclusive publication, no overwrite, and no implicit Gate trust.
- Built-in/user/repository configuration layering, exact `--config` and
  `DIFFDOSSIER_CONFIG` selection, content-bound `--baseline`, and absolute
  flag-over-environment state/cache path overrides.
- Read-only `doctor` diagnostics for effective configuration, local-only
  baseline resolution, repository-external paths, rule scopes and same-scope
  conflicts. Untrusted project Gate candidates are never executed.
- Contract/risk planning, task and contract packets, strict result import,
  finding decisions, fix authorization, refresh, and heterogeneous review
  comparison.
- Manual and mock Provider operation plus a plan-bound command protocol with
  minimal environment, packet scanning, egress grants, timeouts, and attempt
  ledgers. No automatic hosted Provider adapter is published.
- Plan-only project Gate DAGs, exact execution trust, worktree guards, evidence
  recording, verification, finalization, and private portable export.
- Terminal-run archive, content-bound retention plans, shared-blob protection,
  dry-run garbage collection, exact execution trust, and crash-safe retry.
- Public-derivative prepare/approve/create/revoke mechanics with distinct exact
  authorization plans, privacy-safe public records, and append-only tombstones.
  Only synthetic temporary-fixture tests were run; no real public derivative
  was approved or created.
- Deterministic six-target release-candidate archives, checksums, a minimal
  SPDX 2.3 SBOM, unsigned local provenance, and isolated native-host smoke
  verification. No candidate artifact was published.

## Current GitHub state

A read-only GitHub API check on 2026-07-27 reported:

| Item | Current state |
|---|---|
| Repository visibility | public |
| Default branch | `main` |
| Remote `main` | `98856de56e6543b4806d899c611a2744f76686c0` |
| Actions workflows | 0 |
| Repository rulesets | 0 |
| Protected `main` | no |
| Tags | 0 |
| Releases | 0 |

Consequently, local checks are not remote CI evidence and the repository has
not reached a public-beta release gate.

## Contribution provenance decision

The twelve public commits have no `Signed-off-by` trailer. On 2026-07-27, Liu
Liang confirmed the contribution rights, identified Liu Liang as copyright
holder, selected `prospective-with-recorded-exception`, and provided the future
sign-off identity `Liu Liang <lliang@outlook.com>`. The exact immutable commit
scope and policy are recorded in
`docs/governance/dco-history-exception.md`. No history rewrite, force push, or
retroactive trailer is permitted by that decision.

## Authorized CI and release boundary

The GitHub maintainer authorized the following exact permissions and repository
settings on 2026-07-27. The workflows are part of the current uncommitted
candidate and must pass fresh review before push.

Authorized native CI runners:

| Support evidence | GitHub-hosted runner |
|---|---|
| Linux amd64 Tier 1 | `ubuntu-24.04` |
| Windows amd64 Tier 1 | `windows-2025` |
| macOS amd64 Tier 1 | `macos-15-intel` |
| macOS arm64 Tier 1 | `macos-15` |
Tier 2 Linux and Windows arm64 remain cross-build-only in the initial workflow;
the lack of native execution remains documented. GitHub's current runner
reference lists the selected architecture-specific labels:
<https://docs.github.com/en/actions/reference/runners/github-hosted-runners>.

Official Action pins, re-resolved on 2026-07-27:

| Action | Release | Exact commit |
|---|---|---|
| `actions/checkout` | `v7.0.1` | `3d3c42e5aac5ba805825da76410c181273ba90b1` |
| `actions/setup-go` | `v7.0.0` | `b7ad1dad31e06c5925ef5d2fc7ad053ef454303e` |
| `actions/upload-artifact` | `v7.0.1` | `043fb46d1a93c77aae656e7c1c64a875d1fc6a0a` |
| `actions/attest` | `v4.2.0` | `f7c74d28b9d84cb8768d0b8ca14a4bac6ef463e6` |

The CI workflow should have only `contents: read`, use a current supported Go
1.25 patch, and run formatting, tests, race tests where supported, vet, schema
validation, native CLI/doctor smoke, and cross-build checks. It must not receive
repository secrets or write permissions.

A separate tag-triggered release workflow uses exactly
`contents: write`, `id-token: write`, and `attestations: write`; it should
build from a clean tag, verify checksums and native smoke evidence, attest the
exact artifacts, and create a non-draft Release only after all required jobs
pass. GitHub documents those attestation permissions at
<https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations>.

The proposed `main` ruleset should require pull requests, conversation
resolution, linear history, deletion and force-push blocking, and the
successful native CI check names. One approving review should be required only
after an independent reviewer is available; a sole maintainer cannot approve
their own pull request, so enabling that rule earlier would make changes
unmergeable. GitHub also requires a status check to have run successfully
before it can be selected as required, so the workflow must land and pass
before the final ruleset activation.

## Remaining gates

| Gate | Missing authoritative evidence | Safe behavior until approved |
|---|---|---|
| License and provenance | copyright holder, contribution authority, DCO treatment | no tag or Release |
| Native platform | real supported-patch CI on every Tier 1 runner | retain development-only support statement |
| GitHub governance | approved workflow permissions, active ruleset, required checks | local validation only |
| Provider integration | exact executable/version/model, packet, destination, credential source, and spend limit | manual/mock only |
| Automatic adapters | maintainer and Provider-account approval for an exact published adapter version | protocol documentation only |
| Private downstream migration | private parity, governance, rollback, and privacy evidence | no downstream replacement claim |
| Public derivatives | exact candidate hash, class, action, approval, and scan result | prepare/scan only; no real bundle write |
| Release | clean tag, successful CI, checksums, SBOM, attestation, clean install, and maintainer approval | no tag or Release |

## Completion rule

The uncommitted working tree may be described as a local completion candidate,
but the pushed repository is not yet at that checkpoint and the project is not
fully released or stable until every applicable row above has authoritative evidence. Evidence
from cross-compilation, mocks, synthetic fixtures, or local review cannot be
promoted to native CI, legal approval, real Provider execution, private
downstream acceptance, or publication approval.
