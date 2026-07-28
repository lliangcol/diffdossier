# Phase 7 local release-preparation checkpoint
- Status: historical
- Captured-at: 2026-07-26T21:48:20+08:00
- Source-commit: f2ae86965f2dc044669594ccd71fa95f27e35163
- Superseded-by: none
- Current-state notice: Historical checkpoint — do not treat this document as current project status. Revalidate Release, artifact, toolchain, platform, CI, and ruleset claims from their current evidence records.

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
public beta.

## Clean-commit candidate rehearsal

The local rehearsal ran twice from clean commit
`f2ae86965f2dc044669594ccd71fa95f27e35163` with candidate version
`phase7-f2ae869` and controlled builder `go1.26.0`.

- Both builds completed all six target archives with module network access and
  automatic toolchain download disabled.
- `releaseprep verify` covered all nine checksum entries for both builds.
- The first build was extracted in an isolated temporary directory; native
  macOS/amd64 `version --json` and `doctor --json` both passed and the embedded
  version/commit matched the manifest.
- `cmp` confirmed the two independently prepared `SHA256SUMS` files were
  byte-identical on this host and toolchain. This is same-environment
  reproducibility evidence, not a cross-host guarantee.

Recorded digests:

```text
544f98cf228c23f4bfd896658857e161074379c628f1ecd53a1a65407fd15dbd  diffdossier.spdx.json
faf6b53ca3007734850d646ea86608980b97ad7729460f12edf54e1c99e7be9b  diffdossier_phase7-f2ae869_darwin_amd64.tar.gz
5f331778fea81d979538853dc43de884182c63b1be3052a70bd335bb8bbe9356  diffdossier_phase7-f2ae869_darwin_arm64.tar.gz
164b8ad693eba39c18770b7d8421daa4958e30b637730628d18139779e53ea82  diffdossier_phase7-f2ae869_linux_amd64.tar.gz
907b9fcb1874af817062894c5fb98c6822f1ebe77b9c2f26ca72e24d01d73b4c  diffdossier_phase7-f2ae869_linux_arm64.tar.gz
23007c124567889e61f14e4a8802fb4cca6dd42279eead88c0fa9f0c1d55d00b  diffdossier_phase7-f2ae869_windows_amd64.zip
fc3022378c67fbea8bcdac5a05b806390d2c8db2cfe6f9145d3fd23a028b8b29  diffdossier_phase7-f2ae869_windows_arm64.zip
46362780265de448762fd5900cc0842d61ff8258c934a66b5e8a8f9379ddf0f9  provenance.json
6ca653c59f2816ae21d0e21cec4e2f22e95715adcb111861767b4ec9a7634f36  release-manifest.json
```

These candidate artifacts were not committed, uploaded, attested, tagged, or
released. G-03, G-07, and native-platform evidence gaps remain unchanged.
