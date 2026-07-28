# Platform evidence matrix

This matrix is the current repository-record evidence for platform claims. It
is not a support promise, a current-branch CI result, an artifact-download
check, or a Release approval. It was reconstructed on 2026-07-28 from the
specified immutable GitHub Actions runs and workflow definitions.

## Evidence scope

| Item | Current evidence | Boundary |
|---|---|---|
| Support tier | No OS/architecture has an assigned [Tier 1 or Tier 2](governance/evidence-terminology.md) support declaration. | A successful CI job is evidence for its stated checks only. |
| Go version | CI and Release workflows install Go `1.25.12` with `GOTOOLCHAIN=local`, `GOWORK=off`, and `-mod=readonly`. | This is the tested workflow toolchain, not a supported minimum Go version. |
| Minimum OS / Git | Not declared. | Do not infer a minimum version from a GitHub-hosted runner image. |
| Current-branch evidence | None. | The current branch SHA is `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`; the latest visible CI/Release runs target earlier SHAs. |

## Native CI evidence

CI run [#30286989079](https://github.com/lliangcol/diffdossier/actions/runs/30286989079)
completed successfully for commit `aa611755554711dd44fab388f488fd2867ed093e`
on 2026-07-27. Each native job ran `go test -count=1 ./...`,
`go test -race -count=1 ./...`, `spikes/platform`, a CLI `version --json`
smoke, and a read-only `doctor --json` smoke.

| OS / architecture | Runner | Native test | Race | CLI / doctor smoke | Tier | Evidence |
|---|---|---|---|---|---|---|
| Linux amd64 | `ubuntu-24.04` | pass | pass | pass | unassigned | [job 90047123073](https://github.com/lliangcol/diffdossier/actions/runs/30286989079/job/90047123073) |
| Windows amd64 | `windows-2025` | pass | pass | pass | unassigned | [job 90047122993](https://github.com/lliangcol/diffdossier/actions/runs/30286989079/job/90047122993) |
| macOS amd64 | `macos-15-intel` | pass | pass | pass | unassigned | [job 90047123011](https://github.com/lliangcol/diffdossier/actions/runs/30286989079/job/90047123011) |
| macOS arm64 | `macos-15` | pass | pass | pass | unassigned | [job 90047123050](https://github.com/lliangcol/diffdossier/actions/runs/30286989079/job/90047123050) |

## Release-run and artifact evidence

Release run [#30266531344](https://github.com/lliangcol/diffdossier/actions/runs/30266531344)
completed successfully for tag `v0.1.0-beta.3`, commit
`3c46e62740143b62293f1abf526a1e159084e522`. Its four native jobs ran tests,
race tests, vet, and the platform spike; the publish job ran release-set build
and `releaseprep verify --smoke` on `ubuntu-24.04` before publishing.

| OS / architecture | Native Release job | Race | Release asset | Installation smoke | Boundary |
|---|---|---|---|---|---|
| Linux amd64 | [pass](https://github.com/lliangcol/diffdossier/actions/runs/30266531344/job/89978430938) on `ubuntu-24.04` | pass | `diffdossier_v0.1.0-beta.3_linux_amd64.tar.gz` | releaseprep native-host smoke on publish runner; no download/install record | not a four-platform installed-artifact smoke |
| Windows amd64 | [pass](https://github.com/lliangcol/diffdossier/actions/runs/30266531344/job/89978430898) on `windows-2025` | pass | `diffdossier_v0.1.0-beta.3_windows_amd64.zip` | not verified | no release-asset install record |
| macOS amd64 | [pass](https://github.com/lliangcol/diffdossier/actions/runs/30266531344/job/89978430907) on `macos-15-intel` | pass | `diffdossier_v0.1.0-beta.3_darwin_amd64.tar.gz` | not verified | no release-asset install record |
| macOS arm64 | [pass](https://github.com/lliangcol/diffdossier/actions/runs/30266531344/job/89978430917) on `macos-15` | pass | `diffdossier_v0.1.0-beta.3_darwin_arm64.tar.gz` | not verified | no release-asset install record |
| Linux arm64 | no native job | not verified | `diffdossier_v0.1.0-beta.3_linux_arm64.tar.gz` | not verified | asset existence only |
| Windows arm64 | no native job | not verified | `diffdossier_v0.1.0-beta.3_windows_arm64.zip` | not verified | asset existence only |

The beta.3 Release page also lists `SHA256SUMS`, SPDX SBOM, provenance, and
release manifest. This matrix records their presence only; it does not claim a
fresh download, checksum comparison, or attestation verification.

## Cross-build evidence

The `Quality and cross-build` job in CI #30286989079 passed a
`CGO_ENABLED=0` build for `darwin`, `linux`, and `windows`, each on `amd64`
and `arm64`, on an `ubuntu-24.04` runner. It is compilation evidence for all
six targets and **never native runtime evidence**.

## Unverified semantics and remaining gaps

- Native Linux arm64 and Windows arm64 execution and race behavior.
- Download, checksum, installation, `version`, `doctor`, and synthetic smoke
  for every beta.3 artifact on its matching native platform.
- Windows ACL, UTF-8 console, long-path, PATHEXT, symlink and
  case-insensitive-collision behavior; Windows Job Object runtime behavior.
- Linux XDG and permission behavior, and macOS filesystem normalization.
- Cross-platform symlink, ACL, Unicode-normalization, and case semantics.
- A supported minimum OS, Git, and Go version, plus an explicit support-tier
  policy.

See `docs/platform-compatibility.md` for the historical local spike details.
DD-DOC-006 owns reconciling that narrative with this current matrix.
