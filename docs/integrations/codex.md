# Codex integration status

- Evidence date: 2026-07-26
- Status: manual packet/result flow only
- Automatic adapter: not published; G-09 remains required
- Live model smoke: not run; G-05 and exact G-10 authorization remain required
- Locally observed CLI: `codex-cli 0.146.0-alpha.3.1`

## Verified capability

The current official Codex manual documents `codex exec` as the stable,
non-interactive command. It supports a read-only sandbox by default, ephemeral
sessions, JSONL event output, a JSON Schema for the final response, an explicit
working directory, and single-invocation `CODEX_API_KEY` authentication.
Local `codex exec --help` exposed the corresponding flags on the observed
binary.

These capabilities are technically sufficient for a future wrapper to accept a
DiffDossier packet, run outside the target repository, request the
`review-result` Schema, and return one normalized result. They do not by
themselves authorize DiffDossier to launch Codex or send project data.

## Candidate boundary, not an enabled adapter

A future adapter must use `codex exec --ephemeral --sandbox read-only`, isolate
or explicitly suppress ambient configuration, bind the exact executable,
version, argv, environment-value digests, packet, output Schema, model, and
working directory, and pass only credential variables approved for that
invocation. It must not use a bypass-sandbox flag. DiffDossier must still apply
its command trust binding, egress grant, timeout, output bound, and untrusted
Result validation.

The wrapper and any example invocation remain unpublished until the maintainer
approves G-09 for a specific Codex version. A live smoke additionally requires
G-05 and the exact per-run G-10 execution plan.

## Terms and data controls

The applicable terms depend on authentication: OpenAI states that ChatGPT terms
apply to personal ChatGPT sign-in, while the corresponding services agreement
applies to API and business/enterprise use. Account credentials are never
bundled, proxied, or persisted by DiffDossier. The operator remains responsible
for source rights, workspace policy, data controls, usage limits, and current
terms.

Official sources:

- <https://developers.openai.com/codex/codex-manual.md>
- <https://help.openai.com/en/articles/11369540-codex-in-chatgpt-faq>
- <https://openai.com/policies/terms-of-use/>
- <https://openai.com/policies/services-agreement/>
