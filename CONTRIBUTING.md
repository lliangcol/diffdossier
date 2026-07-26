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

Fixtures in this public repository must be synthetic or approved public-project
data. Never contribute credentials, private source, packets, results, logs, or
reports.
