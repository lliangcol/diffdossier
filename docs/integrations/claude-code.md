# Claude Code integration status

- Evidence date: 2026-07-26
- Status: manual packet/result flow only
- Automatic adapter: not published; G-09 remains required
- Live model smoke: not run; G-05 and exact G-10 authorization remain required
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

## Candidate boundary, not an enabled adapter

A future adapter must run from a repository-external private directory, disable
all tools with `--tools ""`, disable session persistence and ambient
customizations, request the exact Result Schema, cap turns and spend, and bind
the executable, version, argv, environment-value digests, packet, model, and
working directory. It must not use `--dangerously-skip-permissions`.
DiffDossier's trust binding, egress grant, timeout, output bound, and Result
validation still apply.

The wrapper and invocation example remain unpublished until G-09 is approved
for a specific Claude Code version. A live smoke additionally requires G-05 and
the exact per-run G-10 execution plan.

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
- <https://code.claude.com/docs/en/legal-and-compliance>
- <https://www.anthropic.com/legal/commercial-terms>
- <https://www.anthropic.com/legal/consumer-terms>
- <https://www.anthropic.com/news/updates-to-our-consumer-terms>
