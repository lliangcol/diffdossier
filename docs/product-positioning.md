# Product positioning

DiffDossier is a local-first, Provider-neutral evidence-control layer for
reviewing large Git changes that cannot safely be understood in one prompt.

## Boundaries

- Start with the smallest safe change; use the full Dossier workflow only when
  a change cannot reasonably be split without losing relevant evidence.
- It does not replace ordinary small-change code review or make every change a
  large-review problem.
- It does not call a Provider, execute commands, mutate source, fix findings,
  publish evidence, create a Release, or change a remote service by default.
- Provider output and automated findings remain evidence requiring the
  configured, content-bound authorization and applicable human decision.

See [`README.md`](../README.md) for the current product entrypoint and
[`governance/evidence-terminology.md`](governance/evidence-terminology.md) for
the meaning of verification and approval terms.
