# Providers and review results

DiffDossier treats repository content, Provider output, and configured external commands as untrusted data.

The default `manual` Provider writes private packets and never launches a model. The `mock` Provider is for deterministic tests. A `command` Provider uses an absolute executable and an argv array; shell parsing is not supported.

`review run` emits the exact paths, byte count, data class, defensive scan
result, network destination class, credential source, and Provider before any
external execution. Manual mode only records that the packet is ready.

Before a command can run, callers produce a command plan containing the
executable and binary digest, argv, repository-external working directory,
hashes of the exact environment values, timeout and output bounds, data class,
redaction digest, network/credential categories, snapshot, task-input digest,
and the fact that no strong OS sandbox is claimed. Both
`--trust-execution-plan` matching the displayed plan, a matching unexpired
trust binding, and a matching unexpired egress grant are required. Environment
flags name variables to copy; their values do not appear in argv or the plan.
The executable and working directory are revalidated immediately before every
process launch.

Command requests are single JSON objects on stdin. Operations are `handshake` and `review`; stdout must contain one compatible JSON object. Stderr is diagnostic only. Both streams are bounded, the environment is replaced rather than inherited, timeouts terminate the process tree where the platform implementation supports it, and all output is parsed as untrusted input.

For command execution, content-addressed current/previous blobs are read back
from private state, rehashed, and embedded in the serialized Provider packet.
That exact serialized packet is scanned and its scan digest is part of the
execution plan. A scan finding prevents execution. A plan declaring no network
also rejects a handshake that reports required or unknown network access.

`diffdossier record task` imports a result only when the current snapshot is still fresh and the result binds the exact task input. Providers may report findings but cannot mark them confirmed. A task counts only completed results with its required coverage and perspectives. Result files and an immutable pass index stay in the private state directory.

Every manual, planned, started, completed, or classified failure attempt is
appended to `reviews/attempts.json`. Switching Provider therefore retains the
prior reviewer/provider/pass history; timeout, quota, rate-limit, login, and
other failures are not mistaken for completed review.

Codex and Claude Code remain manual integrations until their current official capabilities, authorization method, non-interactive behavior, and applicable terms are recorded and a separately authorized live smoke is performed. No live model call is part of the default test suite.
