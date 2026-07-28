# Governance

DiffDossier is maintained by Liu Liang. The GitHub repository owner account is
[`lliangcol`](https://github.com/lliangcol); `liuliang1` is the established
maintainer handle for the same person. The canonical role mapping is recorded
in [`docs/governance/identity-and-roles.md`](docs/governance/identity-and-roles.md).
Decisions are recorded in issues, pull requests, architecture decision records,
and governance records.

- Apache-2.0 is the project license.
- Contributions use DCO sign-off; no CLA is required.
- The pre-enforcement history exception is recorded in
  `docs/governance/dco-history-exception.md`; it does not add trailers to or
  rewrite those commits.
- Stable CLI, configuration, JSON Schema, and Provider Protocol changes require
  compatibility analysis and maintainer review.
- Security, dependency, CI, release, network, command-execution, and public-data
  changes require explicit owner review.
- Releases use Semantic Versioning and must bind source, binaries, checksums,
  SBOM, and provenance to the same tag and commit.

As the maintainer group grows, this document will expand the roles, decision
rules, and succession expectations. Repository ownership alone is not evidence
that an individual review, legal, or security gate has passed; approvals must
remain explicit and auditable.
