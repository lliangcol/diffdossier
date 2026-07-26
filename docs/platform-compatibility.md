# Phase 0 platform compatibility spike

Status: local spike complete; native platform matrix incomplete

This spike tests the minimum assumptions needed before DiffDossier declares an operating-system support tier. It does not establish Tier 1 support by itself.

## Covered by the local spike

- creation and exact NUL-delimited parsing of Git paths containing spaces, tabs, newlines, Chinese, and emoji;
- private state directory and file permissions where POSIX permission bits are available;
- same-directory atomic replacement;
- exclusive lock-file acquisition and reacquisition;
- cancellation of a direct child process;
- `CGO_ENABLED=0` cross-builds for planned Tier 1 and Tier 2 OS/architecture pairs.

## Deliberately not claimed

- Native Windows process-tree cancellation through Job Objects, ACL verification, console behavior, long paths, or case-insensitive collision behavior.
- Native Linux XDG behavior or permissions.
- Native macOS arm64 runtime behavior.
- Symlink, submodule, LFS, invalid UTF-8 path, crash recovery, or concurrent multi-process semantics.
- Any minimum supported OS, Git, or Go version.

Those claims require native fixtures and CI evidence in later phases. Cross-build success is compile evidence only.

## Local execution

```text
go run ./spikes/platform/main.go
```

The program writes only to an OS temporary directory and removes it on exit. It invokes the local `git` executable without a shell.

## Evidence from 2026-07-26

Local toolchain:

- host: macOS amd64;
- Go: `go1.26.0`;
- Git: `2.53.0`.

Native macOS/amd64 execution passed:

- `git-nul-paths`;
- `private-state`;
- `atomic-replace`;
- `exclusive-lock`;
- `direct-child-cancel`.

With `CGO_ENABLED=0`, the same source cross-built for:

- `darwin/amd64` and `darwin/arm64`;
- `linux/amd64` and `linux/arm64`;
- `windows/amd64` and `windows/arm64`.

These cross-builds are compile evidence only. Windows and Linux native runtime checks, macOS arm64 native execution, process-tree semantics, and native permission models remain unverified.

The module declares Go 1.25.0 as its language baseline because Go 1.26 generates new modules with that baseline and Go 1.25 remains supported under the official two-newer-releases policy. The current local runtime evidence is still Go 1.26.0 only; a native Go 1.25 test remains pending.

As of this checkpoint, the official release history lists Go 1.26.5 and Go 1.25.12 with security fixes. Therefore the local Go 1.26.0 results are development evidence only. Release validation must rerun on a current supported patch version.
