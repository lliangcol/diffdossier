# Claude Code integration status

- Evidence date: 2026-07-27
- Status: manual flow plus API-key-only opt-in automatic adapter
- Automatic adapter: implemented in `diffdossier-provider`; publication authorized for the beta release
- Live model smoke: requires an exact command plan, trust binding, egress grant, and API-key authentication
- Locally observed CLI: `2.1.220 (Claude Code)`

## Verified capability

Anthropic's current CLI reference documents non-interactive `claude -p`,
single-result JSON output, validated `--json-schema` output, explicit tool
restriction with `--tools`, safe/minimal configuration modes, no-session
persistence, turn and dollar limits, and permission modes. Local
`claude --help` exposed these capabilities on the observed binary.

These capabilities are technically sufficient for a future wrapper to accept a
DiffDossier packet and return one normalized Result. They do not authorize an
automatic launch or any egress.

## Enabled adapter boundary

The adapter runs from a repository-external private directory, disables
all tools with `--tools ""`, disables session persistence and ambient
customizations, request the exact Result Schema, cap turns and spend, and bind
the executable, version, argv, environment-value digests, packet, model, and
working directory. It must not use `--dangerously-skip-permissions`.
DiffDossier's trust binding, egress grant, timeout, output bound, and Result
validation still apply.

The adapter always supplies `--bare`, so Claude Code accepts only
`ANTHROPIC_API_KEY` or a configured API-key helper and cannot silently reuse a
consumer OAuth session. Its bound arguments include the resolved non-symlink
CLI path, exact CLI and Schema digests, exact version output, model, pass ID,
perspective, one-turn limit, and dollar budget. The generic command Provider
still requires an exact private trust binding and egress grant before launch.

## Live compatibility evidence

On 2026-07-27, an authorized `public_synthetic` boundary smoke pinned the
observed CLI, `claude-sonnet-4-6`, the adapter binary, Result Schema 1.1, a
one-turn limit, and a USD 0.25 maximum. The serialized packet was 2,204 bytes,
contained one synthetic README path, and produced zero sensitive-data scan
findings.

No `ANTHROPIC_API_KEY` or API-key helper was available. The `--bare` adapter
attempt therefore failed without falling back to consumer OAuth. DiffDossier
recorded planned, started, and failed attempts, retained the run as contracted,
and created zero accepted Result artifacts. A successful Claude Code live
compatibility smoke is not claimed. Until a compliant API key is supplied and
the exact plan is rerun, the verified paths are the model-free tests and manual
packet/import flow.

## Terms, authentication, and data controls

Anthropic documents commercial terms for Team, Enterprise, and API users and
consumer terms for Free, Pro, and Max users. Its legal guidance says a
third-party product or service that interacts with Claude capabilities must use
API-key authentication through Claude Console or a supported cloud provider;
it must not route user requests through consumer-plan credentials. DiffDossier
therefore will not bundle, proxy, or reuse OAuth credentials. The operator
remains responsible for source rights, data settings, workspace policy, usage
limits, and current terms.

Official sources:

- <https://code.claude.com/docs/en/cli-usage>
- <https://code.claude.com/docs/en/model-config>
- <https://code.claude.com/docs/en/legal-and-compliance>
- <https://www.anthropic.com/legal/commercial-terms>
- <https://www.anthropic.com/legal/consumer-terms>
- <https://www.anthropic.com/news/updates-to-our-consumer-terms>
