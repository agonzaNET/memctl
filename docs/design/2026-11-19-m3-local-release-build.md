# M3 release build checked locally — 2026-11-19

What was checked: the release archives as `scripts/release-build.sh` builds
them (`make dist VERSION=0.0.9`, stage plan 0003 T6, local part), on the
development VM (linux/arm64), before any tag or GitHub release exists. The
run of the release workflow itself, and a downloaded release binary, are the
maintainer's part of T6 and get their own record.

## Findings

- `make dist` builds `memctl_0.0.9_linux_amd64.tar.gz`,
  `memctl_0.0.9_linux_arm64.tar.gz` and `memctl_0.0.9_darwin_arm64.tar.gz`
  and `SHA256SUMS`; `sha256sum -c` passes for all three.
- Each archive holds `memctl` (0755), `LICENSE` and `README.md` (0644), owned
  by 0/0.
- The unpacked linux/arm64 binary prints `memctl 0.0.9`.
- On a fresh git notes root, over stdio: `note_write` (twice), `search`,
  `note_read`, `note_edit` with the read `sha256`, `context`, `note_move`,
  `note_write` and `note_delete` all succeed; git holds two commits by
  `memctl <memctl@localhost>`, one per burst; no `ERROR` log line.
- `memctl check` on that root reports the one link to the moved note's old
  path as a stale-link warning and exits 0, as SPEC-0005 says.
- `actionlint` 1.7.12 reports no problem in `.github/workflows/release.yml`.

Nothing found here changes a spec, an ADR or the plan.
