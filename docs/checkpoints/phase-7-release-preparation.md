# Phase 7 local release-preparation checkpoint

- Date: 2026-07-26
- Status: local release engineering implemented; public release blocked
- Public tag or GitHub Release created: no
- GitHub Actions or ruleset changed: no

## Implemented locally

- A standard-library-only maintainer command builds six `CGO_ENABLED=0`
  darwin/linux/windows amd64/arm64 binaries from a clean exact commit.
- Release mode requires a v-prefixed Semantic Version tag pointing to `HEAD`.
  Candidate mode permits an untagged rehearsal and labels the manifest.
- Archives use commit-time metadata and deterministic ordering and include the
  license, notice, and README.
- `SHA256SUMS`, release manifest, minimal SPDX 2.3 source/module SBOM, and
  explicitly unsigned local provenance are generated together.
- Verification rejects missing, extra, duplicate, traversing, or tampered
  checksum targets and non-regular release-directory entries. It can run
  isolated `version --json` and `doctor --json` smoke checks for the native host
  artifact.
- Builds disable ambient Go flags/workspaces, toolchain download, and module
  network access. A staged set is self-verified and atomically renamed into its
  previously absent output path, so failures do not expose partial candidates.
- The recorded Go version is read from the same controlled `go` environment
  used for artifact builds, rather than inferred from the maintainer tool's own
  runtime.

## Current dependency and license inventory

- `go list -m all` reports only `github.com/lliangcol/diffdossier` and the Go
  toolchain module; the project declares no third-party Go module.
- The implementation imports only the Go standard library and invokes the local
  Git executable for repository operations.
- Apache-2.0 `LICENSE` and the current `NOTICE` exist, but technical inspection
  is not legal or Owner approval.

## Blocking evidence gaps

- G-03: final license, notice, copyright ownership, and source provenance review.
- G-07: exact GitHub Actions permissions/pins, ruleset, tag, and Release approval.
- Native Windows, Linux, and macOS-arm64 runtime evidence required before any
  Tier 1 or public-beta support claim.
- GitHub-hosted signed attestation and remote required checks do not exist.

Therefore this checkpoint is not Release evidence and does not authorize a
public beta. The local candidate verification record must be appended only after
running against the clean committed implementation.
