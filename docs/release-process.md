# Release process

DiffDossier has no public release yet. This document defines the evidence that
must exist before a maintainer creates one. On 2026-07-27, maintainer
`liuliang1` explicitly authorized the pinned CI and release workflows, the
main ruleset described below, Provider compatibility smoke, ONM migration, the
`v0.1.0-beta.1` tag, attestation, and public prerelease operation. Execution
still fails closed until each preceding check has produced its own evidence.

## Required approvals and evidence

1. The maintainer or legal owner confirms `LICENSE`, `NOTICE`, contribution
   provenance, and the dependency/license inventory (G-03). Any new third-party
   Go module requires the separate G-04 review.
2. The GitHub maintainer approves the exact Actions permissions, pinned Action
   commits, protected-branch rules, tag, and Release operation (G-07).
3. Native supported-platform checks pass. Cross-compilation alone is not Tier 1
   runtime evidence.
4. The exact release commit is clean, reviewed, and tagged with a v-prefixed
   Semantic Version. The tag and `HEAD` must resolve to the same commit.
5. Tests, race tests where supported, vet, formatting, schema validation,
   security tests, and model-free E2E pass against that commit.

The G-03 decision identifies Liu Liang as copyright holder and records the
pre-enforcement commit range without rewriting it in
`docs/governance/dco-history-exception.md`. New contribution commits require
DCO sign-off as `Liu Liang <lliang@outlook.com>`.

## Local artifact preparation

The maintainer-only `releaseprep` command uses only the Go standard library. It
builds with `CGO_ENABLED=0`, `-trimpath`, and `-buildvcs=false`; embeds the exact
version, commit, and commit time; and packages `LICENSE`, `NOTICE`, and
`README.md` with each binary.

For an authorized, already-tagged release:

```text
go run ./cmd/releaseprep build \
  --repo . \
  --output /absolute/new/dist \
  --version v0.1.0 \
  --commit <full-40-character-commit>
go run ./cmd/releaseprep verify --dir /absolute/empty/dist --smoke
```

The output path must not exist. Artifacts are built in a sibling staging
directory, self-verified, and renamed into place only after the complete set is
valid, so a failed build cannot leave a plausible partial release directory.
The normal mode also fails if the tree is dirty, the version is not SemVer, the
tag does not exist, or the tag does not point to `HEAD`. `--candidate` permits
an untagged local rehearsal only; candidate artifacts must never be attached to
a GitHub Release.

The build disables workspace inheritance, automatic toolchain download,
module-network access, and ambient Go flags. Each archive contains the core
`diffdossier` binary, the optional `diffdossier-provider` adapter, and the exact
`review-result.schema.json` contract. Reproduction requires the same
recorded builder reported by the same controlled `go env GOVERSION`
environment; deterministic packaging is not a claim that different Go
toolchains produce identical bytes.

The output contains six archives for darwin/linux/windows on amd64/arm64,
`SHA256SUMS`, a versioned release manifest, a minimal SPDX 2.3 source/module
SBOM, and `provenance.json`. The provenance is explicitly unsigned local
evidence. It is not a GitHub attestation and must not be described as one.

## Publication boundary

After isolated checksum and install verification, an authorized workflow may
produce GitHub-hosted attestations and attach the exact artifacts to the exact
tag. Tokens must use minimum permissions and every third-party Action must be
pinned to a full commit SHA. Package-manager publication is a later, separate
gate.

Rollback keeps the previous supported binary, checksum, and compatible config
available. A release is never repaired in place: revoke or mark it affected,
create a new commit and version, rerun all evidence, and publish a new release.
