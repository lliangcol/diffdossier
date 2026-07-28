# Beta release evidence inventory

Captured by read-only GitHub queries on 2026-07-28. This is an inventory, not
artifact verification or current-branch release approval.

| Version | Annotated tag object | Release commit | Published | Release run | Assets observed | Gaps |
|---|---|---|---|---|---|---|
| `v0.1.0-beta.1` | `d1528347259ab60999c14705d8d82912ccda6d14` | `a2ece60d08a9902b33fe78ce03276aab93eef5f9` | 2026-07-27T00:01:14Z | [#30226340832](https://github.com/lliangcol/diffdossier/actions/runs/30226340832), success | six archives, `SHA256SUMS`, SPDX SBOM, provenance, release manifest | no fresh download, checksum, attestation, embedded-version, or native-install verification |
| `v0.1.0-beta.2` | `d9b840a77e05ee39e1c9bea82e5fa5610ea87866` | `5085fbf874111be647c702f563492d4f1c70c718` | 2026-07-27T01:02:57Z | [#30228700712](https://github.com/lliangcol/diffdossier/actions/runs/30228700712), success | six archives, `SHA256SUMS`, SPDX SBOM, provenance, release manifest | no fresh download, checksum, attestation, embedded-version, or native-install verification |
| `v0.1.0-beta.3` | `9a6ad953a5a537350c84fc6d80ebfb957ddf27e1` | `3c46e62740143b62293f1abf526a1e159084e522` | 2026-07-27T12:39:48Z | [#30266531344](https://github.com/lliangcol/diffdossier/actions/runs/30266531344), success | six archives, `SHA256SUMS`, SPDX SBOM, provenance, release manifest | verification reserved for DD-REL-002/003 |

All three observed Releases are GitHub Pre-releases. The precise asset names,
platform boundaries, and Release-run evidence for beta.3 are recorded in
[`platform-evidence-matrix.md`](platform-evidence-matrix.md).
