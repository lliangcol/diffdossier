# Beta compatibility boundary

This applies to the observed public beta Pre-releases only. It is not a stable
support promise and must be revalidated for a future beta or stable line.

| Area | Beta boundary |
|---|---|
| Platforms | Six archives exist; native CI evidence covers Linux amd64, Windows amd64, macOS amd64, and macOS arm64. Linux/Windows arm64 are not native-runtime verified. No Tier is assigned. |
| Config | `diffdossier.toml` schema version `1` is the current beta contract. Unknown/invalid configuration fails closed; no future compatibility promise is implied. |
| CLI output | `pkg/schema` output envelope is version `1.0`; JSON consumers must handle only the documented beta fields. |
| Provider protocol | Provider handshake protocol is `1.0`; Review Result Schema is `1.1`, with digest-stable reads of legacy `1.0` results as currently implemented. |
| Other JSON schemas | Published files and declared versions are inventoried in `docs/governance/public-contract-inventory.md`; schemas without declared versions have no implied `1.0` guarantee. |

## Known limitations and unsupported uses

- No stable compatibility, support response time, or end-of-support commitment.
- No matching-platform download/install/synthetic smoke record for all six
  archives; GitHub attestation was missing/not verified for beta.3.
- No native Linux arm64 or Windows arm64 execution/race evidence.
- ACL, long-path, console, Unicode-normalization, case, XDG, symlink, and
  Windows Job Object semantic gaps remain as listed in the platform matrix.
- Automatic Provider execution remains explicit, content-bound opt-in; live
  credentials, billing, and real Provider behavior are not beta evidence.
- No package-manager distribution, stable API guarantee, or production approval.

See [platform evidence](platform-evidence-matrix.md), [public contract
inventory](governance/public-contract-inventory.md), and [beta.3 artifact
verification](beta3-artifact-verification.md).
