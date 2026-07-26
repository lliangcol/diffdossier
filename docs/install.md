# Install and verify

No public DiffDossier release exists yet. Do not install Phase 7 candidate
artifacts as if they were supported releases.

When an authorized release exists, download the archive for the exact operating
system and architecture together with `SHA256SUMS`. Verify the archive before
extracting it:

```text
# Linux
sha256sum -c SHA256SUMS

# macOS
shasum -a 256 -c SHA256SUMS

# PowerShell (compare this value with the matching SHA256SUMS entry)
Get-FileHash .\diffdossier_<version>_windows_amd64.zip -Algorithm SHA256
```

Extract the archive, place `diffdossier` (or `diffdossier.exe`) on a trusted
PATH, and confirm the embedded identity. Keep `diffdossier-provider` and
`review-result.schema.json` together in a trusted location if automatic Codex
or Claude Code integration is needed; neither is required for manual review.

```text
diffdossier version --json
diffdossier doctor --json
```

The reported version and full commit must match the release manifest and tag.
If the release includes a GitHub-hosted attestation, verify it against the
repository as an additional control:

```text
gh attestation verify <downloaded-archive> --repo lliangcol/diffdossier
```

A checksum detects accidental or malicious byte changes relative to the
published digest. It does not establish who approved the release. An unsigned
local `provenance.json` is build evidence only, not an attestation.
