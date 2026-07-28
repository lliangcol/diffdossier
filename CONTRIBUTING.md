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

Use a supported Go toolchain and run:

    gofmt -w <changed-go-files>
    go test ./...
    go test -race ./...
    go vet ./...
    git diff --check

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
reports.
