# Public contract inventory

This is the source inventory required by `DD-SCH-001`, captured from the local
tree at `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`. It describes existing
contracts only; it neither changes a contract nor proves external adoption.

## Baseline policy

- Owner: the DiffDossier maintainer, Liu Liang; the current role mapping is in
  [`identity-and-roles.md`](identity-and-roles.md).
- Stability: every listed artifact is **beta / pre-1.0**. Its declared version
  is the exact compatibility boundary, not a stable-release promise.
- Compatibility: producers and consumers must preserve the listed declared
  version. A breaking change requires a version change, compatibility analysis,
  ADR, and maintainer review under `CONTRIBUTING.md` and `GOVERNANCE.md`.
- Baseline test entrance: `pkg/schema.TestPublishedSchemasAreValidJSON` parses
  every `schemas/*.schema.json` and requires `$schema` and `$id`. It does not
  prove producer/consumer conformance; the implementation test paths below are
  the current targeted entrances.

## Published JSON Schema files

| Schema | Declared version | Producer → consumer | Targeted test entrance |
|---|---|---|---|
| `archive-record.schema.json` | 1.0 | `internal/store` archive writer → archive/retention reader | `internal/store` tests |
| `config.schema.json` | 1 | config file author → `internal/config` / CLI | `internal/config` tests |
| `contract-graph.schema.json` | 1.0 | `internal/contracts` / `internal/cli plan` → persisted run reader | `internal/contracts`, `internal/cli` tests |
| `contract-packet.schema.json` | 1.0 | contract-packet producer → external/manual reviewer | `internal/packets` tests |
| `egress-grant.schema.json` | not declared | egress-approval author → `internal/providers` | `internal/providers` tests |
| `event.schema.json` | not declared | `internal/store` event writer → event-chain reader | `internal/store` tests |
| `finding-ledger.schema.json` | 1.0 | `internal/workflow` / CLI → verify and report | `internal/workflow`, `internal/cli` tests |
| `fix-authorization.schema.json` | 1.0 | human authorization record → `internal/workflow` | `internal/workflow` tests |
| `gate-plan.schema.json` | 1.0 | `internal/gates` / CLI → Gate executor and verifier | `internal/gates`, `internal/cli` tests |
| `gc-execution.schema.json` | 1.0 | retention executor → retention audit reader | `internal/store` tests |
| `gc-plan.schema.json` | 1.0 | retention planner → retention executor | `internal/store` tests |
| `invalidation.schema.json` | 1.0 | CLI refresh → verifier/report reader | `internal/cli` tests |
| `inventory.schema.json` | 1.0 | `internal/inventory` → snapshot/store reader | `internal/inventory`, `internal/snapshot` tests |
| `output-envelope.schema.json` | 1.0 | CLI JSON output → CLI/API user | `internal/cli` tests; Go envelope types below |
| `packet.schema.json` | 1.0 | `internal/packets` → manual/mock/command Provider | `internal/packets`, `internal/providers` tests |
| `plan.schema.json` | 1.0 | `internal/planner` / CLI → record, Gate, verify | `internal/planner`, `internal/cli` tests |
| `portable-manifest.schema.json` | 1.0 | `internal/exporter` → portable-export consumer | `internal/exporter` tests |
| `problem.schema.json` | not declared | CLI error producer → CLI/API user | `internal/cli` tests; Go `Problem` below |
| `provider-attempt-ledger.schema.json` | 1.0 | CLI review → review/verify reader | `internal/cli` tests |
| `provider-command-plan.schema.json` | not declared | command-plan author → `internal/providers.Command` | `internal/providers` tests |
| `provider-handshake.schema.json` | not declared | Provider adapter → `internal/providers` / CLI | `internal/adapters`, `internal/providers` tests |
| `public-bundle.schema.json` | 1.0 | `internal/exporter` → public bundle/revocation input | `internal/exporter` tests |
| `public-export-approval.schema.json` | not declared | human approval → CLI/exporter create path | `internal/exporter`, `internal/policy` tests |
| `public-export-preparation.schema.json` | 1.0 | CLI/exporter preparation → approval/create path | `internal/exporter`, `internal/cli` tests |
| `public-export-tombstone.schema.json` | 1.0 | exporter revocation → public-consumer audit | `internal/exporter` tests |
| `redaction-approval.schema.json` | 1.0 | human redaction approval → exporter public-create path | `internal/exporter`, `internal/policy` tests |
| `redaction-log-manifest.schema.json` | 1.0 | Gate log redactor → Gate evidence/exporter | `internal/gates`, `internal/exporter` tests |
| `redaction-manifest.schema.json` | 1.0 | redaction pipeline → exporter approval/create path | `internal/exporter` tests |
| `release-manifest.schema.json` | 1.0 | `internal/releaseprep` → release verifier | `internal/releaseprep` tests |
| `release-provenance.schema.json` | 1.0 | `internal/releaseprep` → release verifier | `internal/releaseprep` tests |
| `report.schema.json` | 1.0 | `internal/reporting` / CLI → report reader | `internal/reporting`, `internal/cli` tests |
| `result-index.schema.json` | 1.0 | `internal/results` / CLI → record and verify | `internal/results`, `internal/cli` tests |
| `review-comparison.schema.json` | 1.0 | `internal/results.Compare` → record/report reader | `internal/results`, `internal/cli` tests |
| `review-result.schema.json` | 1.1 | external Provider → `internal/adapters` / `internal/results`; packaged by release prep | `internal/adapters`, `internal/results`, `internal/releaseprep` tests |
| `risk-assessment.schema.json` | 1.0 | `internal/risk` / CLI → planner/report reader | `internal/risk`, `internal/cli` tests |
| `run.schema.json` | 1.0 | `internal/store` / CLI → store and verifier | `internal/store`, `internal/cli` tests |
| `snapshot.schema.json` | 1.0 | `internal/snapshot` / store → planner and verifier | `internal/snapshot`, `internal/store` tests |
| `task.schema.json` | 1.0 | `internal/planner` → packet/review/record paths | `internal/planner`, `internal/packets`, `internal/cli` tests |
| `trust-binding.schema.json` | not declared | human trust record → `internal/gates` executor | `internal/gates` tests |

`not declared` is an inventory finding: the file has no top-level
`properties.schema_version.const`; it must not be silently treated as version
`1.0`. Normalizing those contracts is separate public-contract work and needs
an ADR before implementation.

## Exported Go types

Only `pkg/schema` is the public Go package in this tree. Its exported types
and helpers are the shared Go surface; the `internal/` packages above are not
importable contracts for external consumers.

| Go API | Version/stability | Producer → consumer | Test entrance |
|---|---|---|---|
| `Version`, `Envelope`, `Problem`, `NewError`, `Success`, `Failure` | 1.0 / beta | `internal/cli` JSON output → CLI/API user | `pkg/schema` and `internal/cli` tests |
| `DataClass` and its four constants | beta; no independent wire version | CLI/policy/provider configuration → policy, packets, providers | `internal/policy`, `internal/packets`, `internal/providers` tests |
| `ProviderHandshake` and `Valid` | protocol 1.0 / beta | Provider/adapter → provider runner and CLI | `internal/adapters`, `internal/providers` tests |
| `ApprovalBinding` | schema 1.0 / beta | approval author → policy/exporter consumer | `internal/policy`, `internal/exporter` tests |

## Boundaries and follow-up

The JSON Schema files are published source artifacts, but this inventory does
not claim that all are runtime-loaded. Source inspection shows that
`review-result.schema.json` is embedded and copied into release artifacts;
the remaining files currently have the baseline parse/identifier test plus the
listed implementation-path tests. DD-SCH-001 records that distinction rather
than asserting full JSON Schema conformance coverage. Future contract changes
must update this inventory and add the compatibility evidence required by the
governance rules.
