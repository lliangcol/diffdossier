# Isolated release candidate rehearsal

This is a local DD-REL-008 rehearsal, performed on 2026-07-28. It did not
create a tag, GitHub Release, upload, attestation, package publication, or
support declaration.

## Isolated input

- Detached local worktree commit:
  `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`.
- Candidate-only version: `rehearsal-d5097d1e`.
- Builder: `go1.25.12 windows/amd64` deployed in the current user scope.
- Mode: `releaseprep build --candidate`; candidate outputs are explicitly not
  release-attachable.

## Fault injection and recovery

| Step | Result | Boundary |
|---|---|---|
| Build with an already existing, empty output directory | Failed with `release output path must not already exist` (exit 1). The directory remained empty. | The failure happened before staging/build output, so no plausible partial release set was left. |
| Remove only the task-created empty directory | Succeeded after confirming the exact directory was empty. | No repository file, tag, Release, or remote object was removed. |
| Rebuild using an absent candidate output path | Succeeded with six target archives, `SHA256SUMS`, SPDX SBOM, manifest and unsigned provenance. | The output remained in the detached worktree’s temporary directory. |
| `releaseprep verify --smoke` | Passed: `FilesChecked=9`, six artifacts, candidate flag true, exact input commit; smoke target `windows/amd64`. | This is a local host smoke, not six-platform installation evidence or Release approval. |

## Candidate identity

`release-manifest.json` reported:

- `candidate: true`;
- version `rehearsal-d5097d1e`;
- commit `d5097d1e2b2c9e8c6aa327919feb4a6fd9c4261b`;
- six archives for darwin/linux/windows on amd64/arm64.

The local provenance is unsigned. This rehearsal does not establish a GitHub
attestation, a reproducible-build comparison, native installation of the five
non-host targets, current candidate CI, a release decision, or any stable/Tier
support claim. Those remain fail-closed gates in
[`next-beta-release-checklist.md`](next-beta-release-checklist.md).

## Cleanup

The candidate output and detached worktree were task-created temporary files.
They were removed after the verification capture; rerun this rehearsal from a
fresh detached worktree instead of reusing its candidate directory.
