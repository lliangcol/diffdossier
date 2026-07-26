# Security Policy

## Supported versions

No stable release exists yet. Security fixes currently target the latest commit
on main; this policy will list supported release lines before v1.0.

## Reporting

Do not open a public issue for a suspected vulnerability or leaked secret.
Use GitHub private vulnerability reporting for this repository. If that feature
is unavailable, contact the repository owner through a private GitHub channel
and include only the minimum reproduction needed.

Do not include real credentials, private source, or third-party personal data.
Maintainers will acknowledge a report, assess severity, coordinate a fix, and
publish remediation information after affected users can update.

DiffDossier treats target repositories, Provider output, project rules, file
names, and configured commands as untrusted input. Its local hashes detect
unexpected changes; they are not a security signature against an attacker with
the same account privileges.
