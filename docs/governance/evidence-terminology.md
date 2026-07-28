# Evidence terminology

These terms deliberately separate local evidence from human approval and
external state. They apply throughout repository documentation unless a
schema or protocol defines a more specific field value.

| Term | Meaning | Does not mean |
|---|---|---|
| `ready` | The declared local scope has sufficient stated evidence for its next named gate. | Merged, approved, released, deployed, or defect-free. |
| `verified` | A named assertion was checked by its stated method against the stated snapshot or source. | A broader claim, future validity, or external approval. |
| `finalized` | The local workflow reached its `FINALIZED` state after its configured verification. | A public release or human acceptance. |
| `approved` | The named authorized role gave the exact, auditable approval described by the record. | Execution occurred, succeeded, or remains valid after scope changes. |
| `mergeable` | All repository-defined merge requirements for the exact change are currently proven. | Merged; local `ready` does not establish this state. |
| `released` | A named version/tag and its intended artifacts were actually published and read back from the named distribution channel. | Stable, supported, installed, or production-approved. |
| Tier 1 | An explicitly declared support tier with named platforms, minimum versions, support obligations, and current evidence. | A CI runner or cross-build target. No Tier 1 platform is currently assigned. |
| Tier 2 | An explicitly declared lower support tier with its own named limits and evidence. | A planned or artifact-only platform. No Tier 2 platform is currently assigned. |

Current evidence details belong in the [platform evidence matrix](../platform-evidence-matrix.md)
and [release-status reconciliation](../release-status-reconciliation.md). A
time-sensitive Release, CI, ruleset, or external fact must be revalidated at
the time it is presented as current.
