# Checkpoint metadata specification

Checkpoint files preserve bounded evidence from a past capture. They are not a
current project-status, Release, platform-support, governance, or authorization
source unless a current document explicitly revalidates and cites their
underlying evidence.

## Required metadata

Every Markdown file directly under `docs/checkpoints/` must place this block
immediately below its H1 title, in this order:

```text
- Status: current | historical | superseded
- Captured-at: YYYY-MM-DDTHH:MM:SS±HH:MM
- Source-commit: <full 40-character Git SHA | uncommitted at <full base SHA> | none>
- Superseded-by: <repository-relative current document path | none>
- Current-state notice: <required notice text>
```

`Captured-at` is the time the checkpoint evidence was captured, not the date a
later editor adds metadata. `Source-commit` names the exact source snapshot;
for an uncommitted candidate it must also name the full committed base. `none`
is permitted only for a pre-commit bootstrap record and must be explained in
the checkpoint body.

Use these exact status meanings:

- `current`: a named owner has revalidated every time-sensitive claim for the
  stated source commit and the document itself is the designated current record.
- `historical`: past evidence retained for traceability; it must not be cited as
  current fact without a fresh source or live check.
- `superseded`: past evidence replaced by `Superseded-by`; the replacement path
  must exist and explain the relationship.

For `historical` and `superseded`, `Current-state notice` must begin with:

> Historical checkpoint — do not treat this document as current project status.

It may then name the specific evidence class that is historical. A `current`
record must instead identify its revalidation owner, evidence date, and expiry
or review trigger in the notice.

## Example

```markdown
# Phase X checkpoint

- Status: historical
- Captured-at: 2026-07-28T11:39:44+08:00
- Source-commit: 0123456789abcdef0123456789abcdef01234567
- Superseded-by: none
- Current-state notice: Historical checkpoint — do not treat this document as current project status. Revalidate Release and platform claims from their current evidence records.

## Captured evidence
...
```

## Maintenance rules

Do not rewrite a checkpoint's captured findings to make them look current.
Classify each existing checkpoint and add the required metadata in `DD-DOC-004`;
if a current statement needs correction, publish or update the designated
current document and point `Superseded-by` to it. Any change to the metadata
schema, status vocabulary, or meaning requires a decision record under the ADR
rule before implementation.
