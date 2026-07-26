# Snapshot semantics

The prepare command resolves an explicit baseline to a local commit, records
HEAD and merge-base, and labels freshness local_only unless a separate future
fetch gate supplies remote evidence. It never guesses main or master and never
fetches implicitly.

Changed paths remain separated into merge_base_to_head, staged, unstaged,
untracked, and explicitly included ignored scopes. Git output is NUL-delimited.
The authoritative path identity is base64 of the original path bytes; a UTF-8
display value is supplementary.

Each entry records previous and current content independently, including mode,
SHA-256, Git object, binary, symlink target, submodule state, and Git LFS
pointer metadata where applicable. Deleted entries have current kind missing
and retain the previous blob.

Snapshot IDs bind length-prefixed canonical fields: Schema version, baseline
and HEAD evidence, index tree, semantic Git configuration, configuration and
public Schema digests, and the complete inventory. Capture inventories the
repository twice and refuses a mixed snapshot if either view differs.
Capture time and the random local repository ID do not affect the content ID.

Durable state is outside the target repository. Review-input bytes are
deduplicated under blobs/sha256, JSON writes use same-directory temporary files
and atomic replacement, and run events form a locally verifiable hash chain.
These hashes detect unexpected state changes; they are not signatures against
an attacker with the same local account privileges.
