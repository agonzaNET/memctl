# Stage plan 0003 — M3 release

Stage: design milestone M3. No spec: M3 holds no application code, so it is
governed by ADR-0012 (build targets and the release workflow) and ADR-0008
(pre-1.0 versioning), as the workflow's carve-out for build and release
artifacts says. This plan is the short written plan that carve-out asks for;
it settles the details ADR-0012 leaves open.

Exit of M3 (design): a release binary runs end to end on a fresh notes root.

## Details settled here

ADR-0012 decides the three targets, the manual `workflow_dispatch` trigger
with an existing `vX.Y.Z` tag, and that the workflow never pushes code or
creates or moves tags. This plan adds:

| Topic | Choice | Why |
|---|---|---|
| Gate | The workflow runs `make verify` on the tag before it builds | A release never ships a tree whose tests fail on Linux |
| Artifacts | One `memctl_<X.Y.Z>_<os>_<arch>.tar.gz` per target, holding `memctl`, `LICENSE` and `README.md` | The archive keeps the executable bit; the licence travels with the binary |
| Checksums | `SHA256SUMS` over the archives, attached to the release | Lets a user check a download with `sha256sum -c` |
| Release notes | The `## [X.Y.Z]` section of `CHANGELOG.md`, cut by `scripts/release-notes.sh`; no section is an error | One source of release notes (ADR-0007); a forgotten CHANGELOG release step fails the release instead of shipping empty notes |
| Version | `make build VERSION=X.Y.Z`, the tag without its `v` | The same string as the CHANGELOG heading and `memctl version` (Makefile) |
| Existing release | `gh release create --verify-tag` fails if the release exists | A release is never overwritten; a rerun after a fix needs a new tag |
| Actions | `actions/checkout` and `actions/setup-go` pinned by commit SHA, with the version in a comment; Go from `.mise.toml` | A moved action tag cannot change a release; the release uses the pinned toolchain |
| Permissions | `contents: read` for the workflow, `contents: write` for the release job only; `persist-credentials: false`; the tag input passed through the environment, never inlined in a script | Least privilege; no script injection through the input |
| Local build | `scripts/release-build.sh` builds the same archives; `make dist VERSION=X.Y.Z` runs it | The release build can be tried before a tag exists |

## File map

| File | Change | Why |
|---|---|---|
| `scripts/release-notes.sh` | create | Print the CHANGELOG section of one version |
| `scripts/release-notes.test.sh` | create | Regression test of the cut, as for `check-docs.sh` |
| `scripts/release-build.sh` | create | Build the three archives and `SHA256SUMS` into a folder |
| `.github/workflows/release.yml` | create | The release workflow |
| `Makefile` | modify | `dist` target; `verify` also runs the script tests |
| `README.md` | modify | Status, install from a release, client setup, flags, notes, git, `memctl check` |
| `CLAUDE.md`, `docs/agents/environment.md` | modify | Layout and the release steps |
| `CHANGELOG.md` | modify | `Added`: release binaries |
| `docs/design/YYYY-MM-DD-m3-release-check.md` | create (T6) | Verification record of the exit check |

## Tasks

- [x] **T1 — Release notes cut.** `scripts/release-notes.sh X.Y.Z [CHANGELOG]`
  prints the body of `## [X.Y.Z] - …` up to the next `## ` heading or the
  link references; exits 1 when the section is missing or empty. A test
  script with cases: a middle section, the last section before the link
  references, a missing version, an empty section, a version that is a
  prefix of another.
- [x] **T2 — Release build.** `scripts/release-build.sh X.Y.Z OUTDIR` runs
  `make build` for each target with `VERSION`, packs the archives and writes
  `SHA256SUMS`. `make dist VERSION=X.Y.Z` runs it into `dist/`. `make verify`
  runs the script tests.
- [x] **T3 — Workflow.** `.github/workflows/release.yml` as settled above.
- [x] **T4 — README.** Install from a release (download, `sha256sum -c`, the
  macOS quarantine attribute), Claude Code and Claude Desktop setup, flags and
  environment variables, the note format in brief, what git sees,
  `memctl check`, building from source.
- [x] **T5 — Docs.** `CLAUDE.md` layout, `environment.md` release steps,
  CHANGELOG entry.
- [x] **T6 — Exit check.** Locally: `make dist`, unpack the archive for the
  machine, run it end to end on a fresh git notes root (write, read, edit,
  move, delete over stdio; commits; `memctl check`). Then by the maintainer,
  after M2's live check: rename `[Unreleased]` to `[0.1.0] - <date>` in a
  commit, tag `v0.1.0`, push, start the workflow, download one archive, check
  it against `SHA256SUMS`, and run it on a fresh root. Write the record under
  `docs/design/`.

## Definition of done

- `make verify` passes, including the script tests.
- `make dist VERSION=0.1.0` builds three archives and `SHA256SUMS` locally.
- The workflow has run once for `v0.1.0`: the release holds three archives,
  `SHA256SUMS` and the CHANGELOG section as notes.
- A downloaded release binary ran end to end on a fresh notes root; the
  record is under `docs/design/` and listed in its README.

## Notes

- The maintainer pushes, tags and starts the workflow; agents never do.
- An automatic trigger on tag push stays a later decision (ADR-0012).
