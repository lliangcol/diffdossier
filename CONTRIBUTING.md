# Contributing to DiffDossier

DiffDossier accepts focused issues and pull requests. By contributing, you
certify the Developer Certificate of Origin 1.1 using a Signed-off-by trailer.

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
