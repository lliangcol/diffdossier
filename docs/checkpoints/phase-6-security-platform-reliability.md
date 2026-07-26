# Phase 6 security, platform, and reliability checkpoint

- Date: 2026-07-26
- Native host: macOS/amd64
- Native Windows/Linux/macOS-arm64 execution: not available
- Remote CI workflow or ruleset changes: not authorized (G-07 pending)

## Implemented and locally verified

- Prompt/data separation and secret-denied/egress fail-closed tests.
- Shared canary scanner, deterministic Gate-log redaction, no matched secret in
  manifests, exact allowlisted-environment-value redaction, scan byte budget,
  and fuzz seeds for deterministic/idempotent redaction.
- 64 MiB Git output and blob budgets plus 256 MiB changed-content budget.
- Run-state/journal binding, empty/truncated chain rejection, stale-lock
  recovery, exact-state write-ahead journal recovery, lock-token ownership, and
  private state permissions on POSIX.
- Unix process-group cancellation kills a spawned grandchild.
- Windows Job Object containment implemented using the standard library and
  Win32 calls; process/store tests cross-compile for windows/amd64 and
  windows/arm64.
- Public synthetic/project/redacted-summary separation, exact public approval,
  exact private redaction approval, tamper rejection, and zero private
  source/approver/repository identity in public bundle fixtures.

## Performance sample, not a service-level claim

On the local Intel i7-4980HQ macOS/amd64 host, three benchmark iterations gave
approximately 51.3 ms/op and 25.6 MiB/op for planning 10,000 synthetic paths;
1 MiB redaction measured approximately 168.9 ms/op (6.21 MiB/s) and 1.05 MiB/op.
Three samples are insufficient for stable P95 or release claims.

## Remaining evidence gaps

- Native Windows Job Object, ACL, UTF-8 console, long-path, PATHEXT, symlink,
  case-insensitive, and crash fixtures.
- Native Linux amd64/arm64 XDG/permission/process behavior.
- Native macOS arm64 and filesystem normalization behavior.
- Strong OS filesystem/network sandboxing, external attestation, remote CI,
  and release artifact verification.
