# Contributing to DiffDossier

DiffDossier accepts focused issues and pull requests. By contributing, you
certify the Developer Certificate of Origin 1.1 using a Signed-off-by trailer.
Use `git commit --signoff` with a name and email you are authorized to publish.
Sign-off is a contribution-rights certification, not a cryptographic commit
signature.

The commits through `98856de56e6543b4806d899c611a2744f76686c0` predate
prospective DCO enforcement. Their exact scope and the maintainer's recorded
contribution-rights confirmation are documented in
[`docs/governance/dco-history-exception.md`](docs/governance/dco-history-exception.md).
All later contribution commits require a `Signed-off-by` trailer.

For project-maintainer provenance records, Liu Liang is the copyright holder
and the prospective DCO sign-off identity is `Liu Liang <lliang@outlook.com>`.
The GitHub repository owner account is `lliangcol` and the established
maintainer handle is `liuliang1`; see
[`docs/governance/identity-and-roles.md`](docs/governance/identity-and-roles.md).
This does not change the requirement that every contributor signs off using a
name and email they are authorized to publish.

## Development

Use Go `1.25.12`, the toolchain used by the repository workflows. The project
uses no third-party Go modules; keep `GOWORK=off`, `GOTOOLCHAIN=local`, and
`GOFLAGS=-mod=readonly` when reproducing CI locally. On a host with an existing
C compiler, also run the race command; if that compiler is unavailable, record
the race result as an environment limitation rather than a pass.

Run the current local test matrix:

    gofmt -w <changed-go-files>
    go test ./...
    go test -race ./...
    go vet ./...
    git diff --check

The CI `Quality and cross-build` job checks formatting, vet, offline document
consistency, published schemas and six cross-build targets. The native jobs
run ordinary tests on Linux amd64, Windows amd64, macOS amd64 and macOS arm64;
their historical success is not current-branch evidence or a support-tier
promise. See [`docs/platform-evidence-matrix.md`](docs/platform-evidence-matrix.md)
for the exact boundary.

Do not add third-party dependencies, change stable CLI or Schema contracts, or
enable network, command execution, source mutation, CI/CD, or release behavior
without a documented design and maintainer review.

Before implementing any public-contract, dependency, network,
command-execution, release, security, or public-data design change, create an
ADR or equivalent decision record using the [ADR rule and template](docs/adr/README.md).
The record must precede implementation and does not replace the explicit
approval required for any external or sensitive action.

Fixtures in this public repository must be synthetic or approved public-project
data. Never contribute credentials, private source, packets, results, logs, or
reports. Public issue forms are not a safe channel for security disclosures;
follow [`SECURITY.md`](SECURITY.md) and do not include sensitive data in a
public Issue, pull request, or Discussion.

## First contribution path

1. Start with a documentation or focused test-only change. Read the relevant
   contract/compatibility document before changing CLI, Schema, Provider or
   Reporter behavior.
2. Use the Issue Forms for synthetic or already-public information only, then
   explain scope, data classification, validation and authorization boundary in
   the PR template.
3. Make the smallest change, run the matrix above, and include exact results
   plus any unavailable checks in the PR. Do not create an external Issue or PR
   as part of a local rehearsal.
4. Add an authorized `Signed-off-by` trailer to each contribution commit when
   it is ready for a maintainer to review.

The current beta boundaries for config, JSON output, Provider protocol and
published schemas are in [`docs/beta-compatibility.md`](docs/beta-compatibility.md)
and [`docs/governance/public-contract-inventory.md`](docs/governance/public-contract-inventory.md).
They are inventories, not stable compatibility guarantees. Changes to these
areas still require the ADR and maintainer-review rules above.
