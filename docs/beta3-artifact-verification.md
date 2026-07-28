# beta.3 artifact verification record

The public `v0.1.0-beta.3` assets were downloaded read-only on 2026-07-28 to a
temporary local directory. This record is limited to the stated byte and
metadata checks; it is not native installation smoke or release approval.

| Check | Result | Evidence / boundary |
|---|---|---|
| Six platform archives | verified | Local SHA-256 matched the corresponding `SHA256SUMS` entries for darwin/linux/windows amd64 and arm64. |
| SBOM, provenance, release manifest | verified | Local SHA-256 matched `SHA256SUMS` for all three metadata assets. |
| Manifest identity | verified | `release-manifest.json` reports `v0.1.0-beta.3`, commit `3c46e62740143b62293f1abf526a1e159084e522`, and six artifacts. |
| Provenance identity | verified | `provenance.json` reports the same version and commit. It remains unsigned local provenance, not an attestation. |
| GitHub attestation | missing / not verified | `gh attestation verify` for the Linux amd64 archive returned GitHub API HTTP 404 for its SHA-256 attestation lookup. No other asset is inferred attested. |
| Embedded executable version | not verified | Requires extraction and native execution; DD-REL-003 owns matching-platform smoke. |

The GitHub Release asset metadata independently listed the same ten files and
their SHA-256 digests at the capture time. The temporary files are not
repository artifacts and are not a distribution approval.
