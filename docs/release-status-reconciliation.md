# Release-status reconciliation matrix

This matrix reconciles release-state wording as observed on 2026-07-28. Its
live source is the read-only GitHub Release listing: `v0.1.0-beta.1`,
`v0.1.0-beta.2`, and `v0.1.0-beta.3` are Pre-releases; no stable release was
listed at that query time. The [platform evidence matrix](platform-evidence-matrix.md)
is the authority for platform and installation-evidence boundaries.

| Document | Existing statement class | Current fact / historical interpretation | Authority | Required wording or action |
|---|---|---|---|---|
| `README.md` | Current claims that no public release exists and tag/Release remain gated | Outdated: three public beta Pre-releases exist. Accurate boundary: no stable release, no support Tier, no current-branch Release validation, and further Release actions remain gated. | Live Release listing; `docs/platform-evidence-matrix.md` | Replace “No public release” with “Three public beta Pre-releases exist; no stable release is claimed.” Keep remaining gate and support boundaries. |
| `SUPPORT.md` | Current claim that no stable **or public beta** exists | Outdated public-beta clause. The development-`main` boundary remains accurate. | Live Release listing; release process | State that beta releases exist but have no stable compatibility/support promise; retain main-as-development-snapshot wording. |
| `docs/install.md` | Current claim that no public release exists and hypothetical installation wording | Outdated first sentence. The generic checksum/attestation instructions remain useful but are not four-platform smoke evidence. | Live Release listing; beta.3 asset listing; platform matrix | Identify beta.3 as the latest observed public beta and keep “not a supported stable release” plus artifact-specific verification boundary. |
| `docs/release-process.md` | Opening sentence says no public release exists; later beta.1 authorization is historical | Opening sentence is outdated. Authorization and beta.1 exception are historical process records, not current approval for another release. | Live Release listing; dated text in this document | State that public beta Pre-releases exist and that the document governs future releases; retain dated beta.1 history explicitly as historical. |
| `SECURITY.md` | “No stable release exists yet” and latest-main targeting | Accurate at query time; it makes no false public-beta denial. Support-line policy still needs a future maintenance decision. | Live Release listing; security policy | No release-state correction required; retain as current wording and link later terminology/support work if needed. |
| `CHANGELOG.md` | `Unreleased` plus historical implementation bullets saying local tooling does not publish and workflows can publish | `Unreleased` remains appropriate for unreleased changes, but completed beta releases need versioned historical sections before the file can serve as a complete release history. Existing past-tense workflow bullet is historical implementation information, not proof of a current release. | Live Release listing; dated changelog text | Add beta version sections from verified Release notes/assets in a later release-documentation task; do not fabricate notes from this matrix. |

## Rules for consuming this matrix

- “Public beta exists” means only that the named GitHub Pre-release was visible
  at the captured query time. It does not certify assets, checksums,
  attestations, installation, compatibility, support, mergeability, or
  production readiness.
- “No stable release” is a time-sensitive external fact; re-check GitHub before
  using it as a current-facing statement.
- Historic phrases in checkpoint, authorization, or changelog context should
  retain their capture-time meaning and be labelled rather than rewritten as
  current fact.
- DD-REL-006 owns the actual cross-document correction of “no public Release”;
  DD-DOC-008 owns shared terminology; DD-DOC-009/010 own drift detection and
  future external-review expiry rules.
