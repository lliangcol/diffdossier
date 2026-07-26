# Security and reliability boundaries

Repository files, paths, rules, Gate definitions, and Provider output are
untrusted data. Trusted prompts are fixed application inputs. Project commands
remain inspect-only until an operator supplies the exact execution-plan digest;
shell mode requires a second acknowledgement. Provider egress remains denied
without exact snapshot/task/data-class/byte-bound grants, and `secret_denied`
cannot enter a packet or public derivative.

Git command output is capped at 64 MiB. Individual changed blobs are capped at
64 MiB and a snapshot at 256 MiB of current-plus-previous changed content.
Budget overflow fails with a diagnostic; it is never treated as a truncated or
complete review. Result parsing, process stdout/stderr, and public scans have
their own fixed limits.

Gate stdout/stderr is scanned and redacted before private persistence. The
manifest stores rule names, offsets, and match digests, never the matched
secret. Every present allowlisted environment value is also redacted by exact
value, even when it does not resemble a credential. The scanner is defense in depth, not proof that arbitrary text contains
no sensitive information. Public export separately requires a clean scan and
exact content-bound approval. A redacted summary additionally requires a
private redaction approval bound to source-run, derived-content, manifest, scan,
approver, and time; the public bundle contains none of those private identities.

Run events are SHA-256 chained. Load verifies a non-empty `run_prepared` origin,
valid transition order, snapshot binding, and equality between journal-derived
and persisted run state. This detects accidental corruption and partial local
tampering but is not protection against an attacker who can rewrite all state;
external attestations are required for independent tamper evidence.

Locks store PID, random token, and UTC creation time. Dead-PID locks are
recovered, live/unknown locks fail closed, and release removes only the exact
owned token. Unix cancellation uses process groups and has a local grandchild
test. Windows uses `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`; amd64/arm64 builds pass,
but native Windows execution remains required before a Tier 1 claim.

State transitions use the event journal as write-ahead evidence. A crash after
the transition event but before `run.json` replacement is detected on load;
`recover --trust-journal-state <STATE>` repairs it only when the operator echoes
the exact valid journal-derived state. Recovery never guesses a later state.

Terminal runs may be made immutable with `run archive`. The archive record
binds the repository/run identity, terminal state, final event hash, complete
regular-file tree digest, pin decision, reason, and timestamp. Archived runs
are excluded from implicit latest-run selection and normal mutation commands.

`gc` only writes a private dry-run plan. Automatic candidates must be archived
`EXPORTED` runs older than the configured retention interval. `FINALIZED`
but unexported runs, blocked/incomplete runs, explicit pins, and runs holding a
public export/redaction approval, public bundle, or revocation tombstone are
never selected. Blob removal is global-reference aware, so a blob retained by
any repository/run survives.

Execution requires `--trust-gc-plan` to exactly match the content-bound plan.
Immediately before removal, DiffDossier revalidates every archive/event/tree
digest, eligibility decision, protection marker, and global blob reference
set. Runs are first renamed into a plan-specific private trash directory;
execution records make successful retries idempotent. This protects accidental
retention races and crashes, but it is not a substitute for filesystem backups.
