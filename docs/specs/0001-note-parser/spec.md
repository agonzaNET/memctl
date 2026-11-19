# SPEC-0001 - Note parser

## Status

Implemented

<!--
Status values: Draft | Accepted | Implemented | Superseded by SPEC-NNNN | Deprecated
Lifecycle: Draft (under review) → Accepted (approved; the code may be built,
but at least one acceptance criterion, for example a live test, is still
open) → Implemented (shipped; code satisfies this spec and all acceptance
criteria are ticked). Superseded by SPEC-NNNN and Deprecated are terminal.
-->

**Governing ADRs:** ADR-0002, ADR-0005, ADR-0010, ADR-0011

## Purpose

Turn the bytes of one Markdown note into the structure the index and the tools
use: frontmatter fields, title, headings, observations, relations and links,
plus a list of problems for `memctl check`. Every later capability (index,
search, graph walk, check, writes) reads notes only through this parser.

## Scope

**In scope**

- `internal/parse`: one pure function from (relative path, bytes) to a parsed
  note. No file system access, no index, no git.
- YAML frontmatter with `go.yaml.in/yaml/v3` (ADR-0011), the only importer of
  that module (ADR-0010).
- The line-based Markdown structure of design D17: ATX headings, fenced code
  blocks, the `## Observations` and `## Relations` sections, `[[links]]`.
- The problems `memctl check` reports about a single note.

**Out of scope**

- Reading files, walking the notes root, change detection (SPEC-0003).
- Link resolution, duplicate titles and dangling links, which need all notes
  (SPEC-0002 for resolution, SPEC-0005 for reporting).
- Tokenising text for search (SPEC-0002).
- Writing notes and serialising frontmatter (milestone M2). This spec only
  keeps what a later write needs to preserve unknown keys.
- Full CommonMark: setext headings, indented code blocks, HTML blocks, inline
  code spans (beyond hiding links, R18) and link reference definitions carry
  no structure (design D17).

## Inputs

- `rel`: the note's path relative to the notes root, slash-separated, ending in
  `.md` (for example `runbooks/PAN-OS upgrade procedure.md`). Used for the file-stem title
  fallback and in problems.
- `data`: the raw file bytes. UTF-8; a leading UTF-8 byte-order mark is
  allowed. Line endings `\n` or `\r\n`.

## Outputs

A `parse.Note` value:

| Field | Content |
|---|---|
| `Title` | Frontmatter `title`, trimmed; if absent or empty, the file stem of `rel` (`runbooks/PAN-OS upgrade procedure.md` → `PAN-OS upgrade procedure`) |
| `TitleFromFrontmatter` | Whether `Title` came from the frontmatter |
| `Type` | Frontmatter `type`, trimmed; empty if absent |
| `Tags`, `Aliases` | Frontmatter lists, each item trimmed, empty items dropped, order kept, duplicates removed |
| `Created`, `Updated` | Frontmatter timestamps as `time.Time`; zero if absent or invalid |
| `Frontmatter` | Every top-level key in file order, each with its 1-based line and its raw YAML source lines (the key line up to the line before the next key), so a later write can keep unknown keys and their order; no yaml.v3 type leaks out of the package. `HasFrontmatter` and `FrontmatterEnd` (the closing delimiter's line) locate the block |
| `Body` | Everything after the frontmatter block, unchanged |
| `BodyLine` | The 1-based file line where `Body` starts |
| `Headings` | Each ATX heading outside fences: level 1–6, text, line |
| `Sections` | Each Observations or Relations section: kind, first line (its heading) and last line |
| `Observations` | Each observation: category, text, line |
| `Relations` | Each relation: type, target, label, line |
| `Links` | Each `[[link]]` outside fences and outside relation lines (the `links_to` edges): target, label, line |
| `Problems` | Each problem: code, line, message (see "Error handling") |

Example, for the note in the design ("Note format v1"):

- `Title` `PAN-OS upgrade procedure`, `Type` `runbook`, `Tags`
  `[palo-alto firewall]`, `Aliases` `[PAN-OS upgrade]`.
- `Headings`: `1 PAN-OS upgrade procedure`, `2 Observations`, `2 Relations`.
- `Observations`: `prereq` "Download content updates before the base image",
  `risk` "HA pair must be upgraded passive first".
- `Relations`: `implements → ODR-017`, `applies_to → PA-850`.
- `Links`: `PA-850` from the body line.
- `Problems`: none.

## Behaviour & Requirements

### Frontmatter

- **R1** — The parser MUST treat the file as having frontmatter only when its
  first line (after an optional byte-order mark) is exactly `---`, and the block
  ends at the next line that is exactly `---` or `...`. Trailing spaces on the
  delimiter lines MUST be ignored.
- **R2** — The block MUST be decoded with yaml.v3. The known keys `title`,
  `type`, `tags`, `aliases`, `created` and `updated` MUST be read; every other
  key MUST be kept in `Frontmatter` with its raw value and its position, and
  MUST NOT cause a problem.
- **R3** — `tags` and `aliases` MUST accept a YAML list of strings or a single
  string (one item). Any other shape MUST give the problem `bad_field` and an
  empty list.
- **R4** — `created` and `updated` MUST accept RFC 3339 timestamps and plain
  dates (`2026-11-19`, read as midnight UTC). Any other value MUST give
  `bad_field` and a zero time; the raw value stays in `Frontmatter`.
- **R5** — A block that is not valid YAML, or whose top level is not a mapping,
  MUST give the problem `bad_frontmatter`. The parser MUST then treat all known
  fields as absent, MUST still start `Body` after the closing delimiter, and
  MUST still parse the body.
- **R6** — A first line `---` with no closing delimiter MUST give
  `bad_frontmatter`, and the whole file MUST be parsed as body (no frontmatter).
- **R7** — If `title` is absent or empty, `Title` MUST be the file stem of
  `rel`, and `TitleFromFrontmatter` MUST be false. A first-level heading MUST
  NOT be used as the title.

### Lines and fences

- **R8** — The parser MUST strip one leading UTF-8 byte-order mark and MUST
  treat `\r\n` like `\n`. Reported line numbers MUST be 1-based file lines,
  counting the frontmatter lines.
- **R9** — A fence MUST open on a line with at most 3 leading spaces followed by
  at least 3 backticks or at least 3 tildes (backtick fences: no backtick in the
  rest of the line). It MUST close on a line with at most 3 leading spaces and
  at least as many of the same character, and nothing else but spaces. A fence
  that is never closed MUST run to the end of the file.
- **R10** — Inside a fence, the opening and closing lines included, the parser
  MUST NOT recognise headings, observations, relations or links, and MUST NOT
  report the problems of R17–R20.

### Headings and sections

- **R11** — An ATX heading MUST be a line with at most 3 leading spaces, 1 to 6
  `#`, then a space, tab or the end of the line. The text MUST be trimmed, with
  an optional closing sequence of `#` (preceded by a space) removed. `#tag` and
  `#123` at the start of a line MUST NOT be headings.
- **R12** — The Observations section MUST start at a level-2 heading whose text
  is `Observations`, and the Relations section at a level-2 heading whose text
  is `Relations`, both compared case-insensitively. A section MUST end at the
  next heading of level 1 or 2. Deeper headings inside a section MUST NOT end it,
  and their lines carry no observation or relation.

### Observations

- **R13** — Inside the Observations section, a line MUST be an observation when
  it matches `- [category] text`: the bullet `-` at the start of the line, one
  space, `[`, a category matching `[a-z][a-z0-9_-]+`, `]`, one or more spaces,
  and non-empty text. The text MUST be trimmed and kept as written, including
  any `#`, `(…)` or `[[link]]` in it.
- **R14** — GitHub task items (`- [ ]`, `- [x]`, `- [X]`), indented bullets,
  `*` or `+` bullets and other lines in the section MUST NOT be observations,
  and MUST NOT give a problem.
- **R15** — A `[[link]]` inside an observation's text, outside code spans,
  MUST also be recorded in `Links`.

### Relations

- **R16** — Inside the Relations section, a line MUST be a relation when it
  matches `- type [[Target]]` or `- type [[Target|label]]`: the bullet `-` at
  the start of the line, one space, a type matching `[a-z][a-z0-9_]*`, one or
  more spaces, and a link; text after the link MUST be allowed and ignored. A
  relation's link MUST NOT also be recorded in `Links`. Other lines in the
  section MUST be parsed for links like body text (R18).

### Links

- **R17** — Outside fences, every `[[target]]` and `[[target|label]]` on a line
  MUST be a link, in order of appearance, several per line allowed. `target` and
  `label` MUST be trimmed. The target MUST NOT contain `[`, `]`, `|` or a line
  break. It is kept as written: a path target such as `network/intpxy` or
  `hosts/PA-850.md` and a title target are not told apart here; resolution is
  SPEC-0002's (design D23). A link with an empty target MUST be ignored and give the problem
  `bad_link`.
- **R18** — Links in the frontmatter MUST NOT be recorded. Links inside inline
  code spans MUST NOT be recorded and MUST NOT give a problem. A code span opens
  at a run of one or more backticks and closes at the next run of exactly the
  same length on the same line; a run with no closing run on its line is
  literal text. Code spans carry no other structure (design D17).

### Problems

- **R19** — Outside the Observations section, a line that matches the
  observation form of R13 MUST give the problem `observation_outside_section`.
  Outside the Relations section, a line that matches the relation form of R16
  MUST give `relation_outside_section`. The line MUST NOT be parsed as an
  observation or relation; its links MUST be recorded per R17.
- **R20** — The parser MUST report unsupported forms as problems and give them
  no structure: `setext_heading` for a line of only `=` or only `-` (at least 2
  characters) directly after a non-blank, non-heading text line; `html_block`
  for a line that starts (after at most 3 spaces) with `<!--` or with `<` and a
  letter; `indented_code` for a line indented by 4 or more spaces, or a tab,
  that follows a blank line and is not inside a list item started by `-`, `*`,
  `+` or `1.`. Each MUST be reported once per block, at its first line.
- **R21** — Invalid UTF-8 in `data` MUST give the problem `invalid_utf8` at the
  first affected line; the parser MUST still return a note.
- **R22** — The parser MUST NOT fail: every input, including empty bytes,
  binary data and a file of only frontmatter, MUST return a `Note` and a
  possibly empty problem list. It MUST NOT panic.

### Boundaries

- **R23** — `internal/parse` MUST NOT read files, use the clock, log, or import
  any module other than the standard library and `go.yaml.in/yaml/v3`, and MUST be
  the only package that imports yaml.v3 (ADR-0010).
- **R24** — The parser MUST be deterministic: the same input gives the same
  `Note`, field for field, including the order of lists and problems (by line,
  then by code).

## Error handling

The parser never returns an error; everything that is wrong with a note is a
problem in `Note.Problems`, with a code, a 1-based line and a one-sentence
message that names the line's issue and how to fix it. `memctl check`
(SPEC-0005) prints them; the index (SPEC-0002) uses the note anyway.

| Code | Raised by |
|---|---|
| `bad_frontmatter` | R5, R6 |
| `bad_field` | R3, R4 |
| `bad_link` | R17 |
| `observation_outside_section` | R19 |
| `relation_outside_section` | R19 |
| `setext_heading` | R20 |
| `html_block` | R20 |
| `indented_code` | R20 |
| `invalid_utf8` | R21 |

## Acceptance criteria — "Done looks like"

- [x] **R1, R2** — The design's example note parses to the values listed under
  "Outputs", and an extra key `source: manual` is kept in `Frontmatter` with no
  problem.
- [x] **R3** — `tags: firewall` gives one tag; `tags: {a: 1}` gives `bad_field`.
- [x] **R4** — `created: 2026-11-19` and an RFC 3339 value parse; `created: soon`
  gives `bad_field` and a zero time.
- [x] **R5, R6** — Broken YAML and an unclosed `---` each give
  `bad_frontmatter`, and the body is still parsed as stated.
- [x] **R7** — A note without `title` at `hosts/PA-850.md` has the title `PA-850`,
  even with a `# Main firewall` heading.
- [x] **R8** — A CRLF file with a byte-order mark gives the same note as its LF
  copy, with the same line numbers.
- [x] **R9, R10** — Headings, observations, relations, links and problems inside
  backtick and tilde fences (including a longer closing fence and an unclosed
  fence) are ignored.
- [x] **R11** — `## Title ##`, `#` alone and `###### six` are headings; `#tag`,
  `#123` and `####### seven` are not.
- [x] **R12** — `## observations` opens the section; `### Detail` inside it does
  not end it; `## Next` and `# Top` do.
- [x] **R13, R14, R15** — Table test of observation lines: valid, task items,
  one-letter category, upper-case category, indented, `*` bullet, missing text,
  and a link inside the text.
- [x] **R16** — Table test of relation lines: with label, with trailing text,
  bad type, missing link; the relation's link is not in `Links`.
- [x] **R17, R18** — Several links per line, labels, an empty target, a link in
  frontmatter, a link in `` `[[x]]` `` and in a double-backtick span (ignored),
  and a link after an unclosed backtick (recorded).
- [x] **R19** — An observation line in the body and a relation line under
  `## Observations` each give their problem, and their links are recorded.
- [x] **R20** — Each unsupported form gives its problem once, at its first line;
  `---` after a blank line (a thematic break) gives none.
- [x] **R21, R22** — Invalid UTF-8, empty input and random bytes return a note
  without a panic; a fuzz test (`FuzzParse`) runs in `go test` with its seed
  corpus.
- [x] **R23** — `internal/arch/imports_test.go` lists `go.yaml.in/yaml/v3` →
  `internal/parse` and passes; `internal/parse` imports nothing else outside the
  standard library.
- [x] **R24** — Parsing the same input twice gives equal notes (`reflect.DeepEqual`).

## References

- Design: "Note format v1", D2, D16, D17, D23 in
  [`../../design/solution-design.md`](../../design/solution-design.md).
- [ADR-0010](../../adr/0010-one-importer-per-third-party-module.md),
  [ADR-0011](../../adr/0011-mcp-sdk-and-yaml-v3.md).
- SPEC-0002 (index), SPEC-0003 (freshness check), SPEC-0005 (`memctl check`) —
  planned.
