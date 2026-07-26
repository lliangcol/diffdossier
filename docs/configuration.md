# Configuration contract

DiffDossier reads diffdossier.toml from the explicit --repo root unless
--config names another file. A relative --config path is resolved against the
explicit repository root, never an accidental working directory.

Configuration precedence is CLI, allowlisted DIFFDOSSIER_* environment
variables, repository configuration, user configuration, then built-in
defaults. Secrets are excluded from every configuration layer.

Schema version 1 rejects unknown fields and unsupported versions. There is no
predecessor to migrate. Future versions must provide an explicit dry-run
migration or reject input; they must never reinterpret an old document
silently.

The baseline field is required. DiffDossier does not guess main or master, and
resolving a local ref is not evidence that the remote is fresh.

    diffdossier config validate --repo /absolute/repository
    diffdossier config validate --repo /absolute/repository --json
    diffdossier doctor --json

The logical JSON model is published in schemas/config.schema.json. The TOML
surface accepted by the current parser is demonstrated by
diffdossier.example.toml; unsupported TOML features fail closed.

Optional risk policy files use the strict format shown in
policies/risk.example.toml. Policy rules may raise inferred risk but can never
lower it. Invalid, missing, escaping, or unknown policy input blocks planning.
