# Next beta release checklist

Candidate commit: `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b` (documentation
planning snapshot only; **not approved for a tag or Release**).

Every row must be explicitly `pass` with evidence bound to the exact candidate
commit. Any missing, stale, failed, or unapproved row is `fail-closed` and
blocks tag/Release creation.

| Gate | Required evidence | Current status |
|---|---|---|
| Version and clean tag | New unused SemVer beta tag points to clean exact candidate commit. | blocked — no version decision or tag authorization |
| CI / required checks | Current candidate CI, required checks, DCO and ruleset review. | missing — historical runs are not candidate evidence |
| Six artifacts | Deterministic archives bound to candidate, with embedded version/commit. | missing |
| Checksums / SBOM / manifest / provenance | Generated and locally verified as one candidate set. | missing |
| GitHub attestation | Attestation exists and verifies for every intended archive. | missing — beta.3 verification found no attestation |
| Native install smoke | Matching-platform download, checksum, install, version, doctor, synthetic smoke. | missing |
| Known limitations | Candidate release notes link beta compatibility, attestation and platform limits. | missing |
| Case / documentation | Approved reproducible case and privacy-cleared documentation inputs. | missing |
| Maintainer approval | Content-bound approval for version, candidate SHA, limits, release action and rollback owner. | blocked — requires maintainer decision |

No row authorizes another row. Passing local artifact verification does not
authorize GitHub writes; a separate exact authorization is required before
tagging, publishing, or changing any Release.
