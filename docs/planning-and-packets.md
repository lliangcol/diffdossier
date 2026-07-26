# Planning and packet contracts

The plan command operates only on a PREPARED run and first proves that the
current repository, configuration, rule files, risk policy, public Schemas,
Prompt, Provider manifest, toolchain, and binary still match its snapshot.
Stale input exits with code 4 and writes no planned state.

Rule discovery recognizes scoped AGENTS.md and CLAUDE.md files while excluding
dependency directories. Rules can constrain review but cannot authorize
network access, command execution, source mutation, commit, push, or
publication.

Contract discovery combines path signals into candidate edges such as API,
database, message/external, delivery, configuration, and CLI protocol.
Heuristics remain candidate_only; they are not presented as verified business
facts.

Risk is L0 through L4. Project policy may raise risk but never lower it.
Unknown paths fail closed with needs_confirmation. L3 and L4 tasks require an
accountable owner and two perspectives: correctness and failure-recovery.

Tasks are grouped by risk and contract before file and byte budgets are
applied. Each path belongs to exactly one task. Tasks sharing a contract form a
deterministic acyclic dependency graph. A single oversized file is retained in
full by blob reference, marked incomplete, and requires a complete read; it is
never silently truncated.

The initial packet Provider is manual. Packets are private_project by default,
contain only a fixed untrusted-data Prompt and content-addressed previous and
current blob references, and do not invoke a model or send bytes outside the
machine. secret_denied content cannot enter a packet.
