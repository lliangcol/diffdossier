# Historical beta rollback drill

This desktop exercise uses the already-published `v0.1.0-beta.3` as historical
input only. It performs no GitHub write, tag move, asset deletion, Release edit,
or user notification.

## Scenario and outcome

| Step | Procedure | Result for beta.3 |
|---|---|---|
| Identify | Record the affected version, exact Release URL, commit, asset hashes, reason, scope, and decision owner. | Candidate: `v0.1.0-beta.3`, commit `3c46e62740143b62293f1abf526a1e159084e522`; no actual incident declared. |
| Contain | Do not overwrite artifacts, move/delete the tag, or amend the Release in place. Stop distribution/automation only with separately authorized external actions. | No external action performed. |
| Mark affected | With explicit authorization, add an affected/deprecated notice to the existing Release and publish an advisory path appropriate to the impact. Preserve original asset evidence. | Planned only; no notice created. |
| Remediate | Create a new audited commit and a new, never-reused SemVer tag; rerun all required release evidence. | Planned only; no tag created. |
| Upgrade / rollback | Direct users to the replacement version; if no safe replacement exists, direct them to the last explicitly supported version and its recorded checksums/config boundary. | No supported beta line is declared, so this cannot nominate beta.2 automatically. |
| Read back | Re-read Release page, tag target, assets, checksums, attestation state, and matching native smoke records. | Not applicable: no external mutation was authorized. |

## Invariants

- A Release is never repaired in place and a tag is never reused.
- Checksums, SBOM, provenance, Release manifest, affected-version notice, and
  replacement tag must remain independently traceable.
- “Rollback” means a user-directed move to a known-safe published version; it
  does not mean rewriting Git history or deleting evidence.
- An affected-version mark, advisory, new tag, new Release, or notification is
  an external action requiring precise authorization.
