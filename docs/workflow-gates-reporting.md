# Workflow, Gates, reporting, and exports

Provider findings enter the private ledger only as `reported` or
`needs_confirmation`. An operator can confirm or reject them; accepted risk
requires a reason, owner, future expiry, and review trigger. `fix authorize`
records a content-bound authorization for confirmed finding IDs, the current
snapshot, and an exact mutation-scope digest. It never changes source files.
Ledger IDs are derived from snapshot, task, pass, and Provider-local ID so two
independent reviewers can safely use the same local finding label.

After an external fixer runs, `refresh` compares the actual changed path set
with an unexpired authorization. A mismatch blocks the refresh. A match creates
a new PREPARED run, preserves the previous run as MUTATED, and records direct,
dependency-neighbor, dependent-neighbor, shared-contract, or semantic-input
reasons for every task in `must_reload`.

`gates plan` is safe for untrusted repository configuration: it only resolves
and hashes executable, argv, cwd, environment, network/resource/cache policy,
expected writes, dependencies, snapshot, configuration, and the DiffDossier
binary. `gates run` requires `--trust-execution-plan` to equal that exact plan
digest. Shell-mode argv also requires `--trust-shell`. Any environment,
executable, plan, configuration, binary, or source snapshot change invalidates
the authorization. Commands use bounded argv-only process execution, and an
unexpected repository mutation makes the snapshot stale.
Gates marked `final_always` require evidence from `gates run --final`; an older
ordinary pass cannot satisfy finalization.

`verify` deterministically rebuilds the review plan, validates every indexed
Result, checks Gate evidence against the exact current plan, and writes JSON and
Markdown reports. Reports separate verdict, snapshot/revisions/worktree,
coverage, findings, confirmations, Gate evidence, reviewer differences, human
merge notes, residual risks, and evidence limitations. `finalize` enters
FINALIZED only for `ready`; ready is local evidence for a next gate, not merge
approval.

`export portable` creates a deterministic private ZIP outside the target
repository. It keeps the event chain but excludes locks, logs, trust, and
approval records so authority is not silently transferred. `export public
prepare` only materializes a candidate and secret scan in private state.
`approve`, `create`, and `revoke` each have a dry planning call that performs
no public action and returns a distinct content-bound G-12 plan digest. The
corresponding `--trust-public-*` echo is required before the private approval,
public bundle, or append-only tombstone is created. Public bundles contain only
the approved public/derived content, policy/scan metadata, and the hash of the
private approval record; they omit approver, repository, run, source path, and
private approval objects. A tombstone explicitly states that external copies
cannot be recalled.
