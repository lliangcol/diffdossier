# Beta Claude Code smoke exception

- Decision ID: `BETA-EXCEPTION-001`
- Decision date: 2026-07-27
- Maintainer and accepted-risk owner: `liuliang1`
- Release scope: `v0.1.0-beta.1` only
- Decision: `defer-successful-claude-live-smoke`

The maintainer explicitly authorizes the first public beta to proceed without
a successful Claude Code live model smoke. The attempted `public_synthetic`
boundary run had no compliant API key, failed without an accepted Result, and
did not fall back to consumer OAuth. That failure remains evidence of the
authentication boundary; it is not successful provider compatibility evidence.

## Compensating controls

- Claude Code support remains opt-in, API-key-only, tool-free, one-turn,
  spend-capped, and governed by an exact command plan, trust binding, egress
  grant, CLI/version digest, model, Schema digest, task input, and perspective.
- Model-free unit, protocol, structured-output, hostile-input, timeout, and
  process-boundary tests remain release-blocking.
- The manual packet/import workflow remains the supported fallback.
- Documentation must state that successful Claude Code live compatibility is
  unverified for this beta.
- No consumer OAuth, bundled credential, or unbound retry is permitted.

## Review triggers and expiry

This exception expires before any stable release. It must also be reviewed
when the Claude Code CLI version, adapter invocation, Result Schema, credential
mode, or release scope changes. A compliant API key becoming available is a
trigger to run a fresh exact-plan smoke, not permission to reuse the prior
trust or egress binding.

The exception does not waive any other release, security, CI, attestation,
privacy, ONM migration, or publication requirement.
