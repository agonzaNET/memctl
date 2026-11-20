# SPEC-0002 - In-memory index: link resolution, search, graph walk and listing

## Status

Accepted

> Amended on 2026-11-20 for the findings of the live check
> ([`2026-11-20-live-check.md`](../../design/2026-11-20-live-check.md)),
> delivered by stage plan 0004: R7 and R17 changed; R23 to R26 added. Until
> the stage is implemented, the code matches the version of 2026-11-19.

<!--
Status values: Draft | Accepted | Implemented | Superseded by SPEC-NNNN | Deprecated
Lifecycle: Draft (under review) → Accepted (approved; the code may be built,
but at least one acceptance criterion, for example a live test, is still
open) → Implemented (shipped; code satisfies this spec and all acceptance
criteria are ticked). Superseded by SPEC-NNNN and Deprecated are terminal.
-->

**Governing ADRs:** ADR-0002, ADR-0005, ADR-0017

## Purpose

Hold every parsed note of one notes root in memory and answer the questions
the tools ask: which note a link or a `ref` names, which notes match a search
query, which notes are connected to a note, and which notes are in a folder or
have a type or tag. The index is rebuilt from the files on every start and
never written to disk (design D14).

## Scope

**In scope**

- `internal/index`: adding, replacing and removing notes; link and `ref`
  resolution (D23); the tokenizer and BM25F search with filters and snippets;
  backlinks and the graph walk behind `context` (D9); filtered listing; the
  list of all links with their resolution, for `memctl check`.

**Out of scope**

- Parsing a note (SPEC-0001); the index receives `parse.Note` values.
- Reading files, walking the notes root and deciding which notes changed
  (SPEC-0003).
- Formatting results for the agent, argument validation and error codes of the
  tools (SPEC-0004), and the text of `memctl check` (SPEC-0005).
- Locking: the index is not safe for concurrent use; the note service
  serialises every call (design, "Tool writes").
- Semantic or vector search (v2).

## Inputs

- `Put(path, note, meta)`: a relative note path (slash-separated, ends in
  `.md`), its `parse.Note`, and `Meta{ModTime, Size, SHA256}` from the caller.
  Replaces any entry with the same path.
- `Remove(path)`.
- `Resolve(target)`: a link target or a tool `ref`, as written.
- `Search(Query{Text, Type, Tags, Folder, Limit})`.
- `Walk(start, depth, relTypes)`: a resolved note path, a depth, an optional
  list of edge types.
- `List(Filter{Folder, Type, Tag})`.
- `Links()`: no input.

## Outputs

- `Resolution{Status, Path, Candidates, Stale, Step, Also}` with `Status` one of
  `resolved`, `ambiguous`, `dangling`, and `Step` one of `path`, `title`,
  `alias`, `stem`.
- `Result{Hits, Total}` for a search, with
  `[]Hit{Path, Title, Type, Tags, Score, Snippet, SnippetLine}`.
- `Graph{Nodes, Edges, Truncated}` for a walk: nodes with their distance from
  the start, edges with source, type, target and the resolution of the target.
- `[]Entry{Path, Title, Type, Tags, Updated, ModTime}` for a listing.
- `[]LinkReport{Source, Line, Kind, Type, Target, Resolution}` for `Links()`.
- `Get(path)` returns the stored note and meta; `Titled(title)` returns every
  path whose title equals `title` (for the shared-title notice of `note_write`).

## Behaviour & Requirements

### Storage

- **R1** — After `Put`, `Get(path)` MUST return that note and meta, and every
  other operation MUST reflect it; after `Remove`, none MAY return the path.
  `Put` of an existing path MUST replace the entry, and `Remove` of an unknown
  path MUST be a no-op.
- **R2** — The result of every operation MUST depend only on the current set of
  entries, not on the order in which they were added, replaced or removed.

### Resolution (design D23, ADR-0017)

- **R3** — `Resolve` MUST normalise the target by trimming spaces and folding
  case (`strings.ToLower`). For the path step it MUST also strip one leading
  `./` or `/` and one trailing `.md` (any case).
- **R4** — `Resolve` MUST try four steps in order, comparing normalised forms:
  1. **path**: note paths without `.md`;
  2. **title**: note titles;
  3. **alias**: note aliases; an alias that contains `/` or ends in `.md` is
     normalised like a path (R3);
  4. **stem**: file names without folder and `.md`.

  At the first step that matches any note, one distinct note MUST give
  `resolved` with its path, and two or more MUST give `ambiguous` with every
  matching path in `Candidates`, sorted. Later steps MUST NOT be tried. No match
  at any step MUST give `dangling`.
- **R5** — A resolution that succeeds in the alias step through a path-like
  alias MUST set `Stale` to true; every other resolution MUST set it false.
- **R6** — A note that matches a step through several of its own names (for
  example two aliases that normalise the same) MUST count once.
- **R23** — A `resolved` result MUST set `Step` to the step that matched, and
  `Also` to every other note that the target matches at a later step, sorted
  (a path ref `alpha` that resolves to `alpha.md` while `archive/alpha.md` has
  the title `Alpha` gives `Also` = `archive/alpha.md`). `ambiguous` and
  `dangling` results MUST leave both empty.

### Search

- **R7** — The tokenizer MUST lower-case the text and split it into maximal runs
  of Unicode letters (`unicode.IsLetter`) and digits (`unicode.IsDigit`), the
  *parts*; everything else separates parts. Two or more parts in a row, each
  pair separated by exactly one of `-`, `.`, `_`, `/`, `:` and nothing else, form
  a *compound*; the longest such chain MUST be emitted as one more token, its
  lower-cased text from its first to its last part. It MUST NOT stem or drop
  stop words. `PA-850` gives `pa`, `850`, `pa-850`; `10.11.0.51.` gives `10`,
  `11`, `0`, `51`, `10.11.0.51`; `a - b` and `a--b` give `a`, `b`; `Größe`
  gives `größe`.
- **R24** — Indexing MUST use every token of R7, parts and compounds, and the
  field lengths of R9 MUST count them all. The query terms MUST be each
  compound of the query and each part that is not inside a compound:
  `IT-104 rollback` gives `it-104`, `rollback`. So a compound in the query
  matches only notes that hold the whole compound, and a part alone (`104`)
  still matches notes that hold it inside a compound.
- **R8** — Five fields MUST be indexed per note: `title`, `aliases` (all aliases
  joined), `tags` (all tags joined), `observations` (all observation texts
  joined) and `body` (the body without the lines recorded as observations,
  fenced code included).
- **R9** — Scoring MUST be BM25F with `k1 = 1.2`, per-field `b = 0.75`, field
  weights title 3, aliases 2, tags 2, observations 1.5, body 1, and
  `idf = ln(1 + (N − df + 0.5) / (df + 0.5))`, where `N` is the number of notes
  and `df` the number of notes in which the term occurs in any field. For each
  query term the weighted, length-normalised term frequencies of the fields are
  summed, saturated with `k1`, and multiplied by its `idf`; a note's score is the
  sum over the distinct query terms. Query terms combine with OR: a note needs
  only one of them, and matching more raises its score (R10).
- **R10** — A note MUST be a hit when it contains at least one query term in any
  field and passes every filter. Hits MUST be ordered by score, descending, then
  by path, ascending.
- **R11** — Filters MUST combine with AND: `Type` equal, case-insensitively;
  every tag in `Tags` present, case-insensitively; `Folder` a path prefix ending
  at a `/` boundary (`ops` matches `ops/a.md` and `ops/x/b.md`, not
  `opsx/c.md`); an empty filter matches everything.
- **R12** — `Limit` MUST default to 10 when zero and MUST be capped at 50.
- **R25** — `Search` MUST return in `Total` the number of hits before `Limit`
  is applied, and each hit MUST carry the note's tags in `Tags`, as written.
- **R13** — A query with no token after tokenising MUST return the error
  `index.ErrEmptyQuery` and no hits.
- **R14** — Each hit's snippet MUST be the observation with the most distinct
  query terms, or, when no observation has one, the body line with the most;
  ties go to the earlier line. The snippet MUST be the line's text, trimmed and
  cut to 200 runes with `…` when longer, and `SnippetLine` its file line. A hit
  that matches only in the title or aliases MUST have the first non-empty body
  line, or an empty snippet when the body is empty.

### Graph (design D9)

- **R15** — A note's outgoing edges MUST be its relations (edge type = relation
  type) and its body links (edge type `links_to`, design D9), each resolved per
  R4 at query time.
  Its incoming edges (backlinks) MUST be every resolved edge of another note
  whose target is this note.
- **R16** — `Walk` MUST do a breadth-first walk from `start` over outgoing and
  incoming resolved edges, up to `depth` hops; `depth` MUST default to 1 and be
  limited to 1–3 (`index.ErrBadDepth` otherwise). When `relTypes` is not empty,
  only edges of those types MUST be followed and returned.
- **R17** — The result MUST list each reached note once, at its shortest
  distance, with the start at distance 0, ordered by distance, then path. It
  MUST list every resolved edge whose source and target are both listed notes
  (only of the types in `relTypes` when it is not empty), including edges
  between two notes at the maximum depth that the walk did not follow, plus
  the start note's own outgoing edges whose target is `dangling` or `ambiguous`
  (these are listed, never followed). Edges with the same source, type and
  target MUST be listed once, at their first line. Edges MUST be ordered by
  source path, then line, then target. A note's links to itself MUST NOT
  appear.
- **R18** — The walk MUST stop adding notes at 200 and set `Truncated`.
- **R19** — `Walk` on a path that is not in the index MUST return
  `index.ErrNotFound`.

### Listing and link report

- **R20** — `List` MUST return every note that passes the filters of R11 (with
  one optional tag), sorted by path.
- **R26** — Each entry MUST carry the file's modification time from its `Meta`
  in `ModTime`, next to `Updated` from the frontmatter, so a listing can show
  when a note without `updated` last changed.
- **R21** — `Links()` MUST return every relation and link of every note with its
  resolution, sorted by source path, then line, then target, with `Kind`
  `relation` or `link`.

### Performance

- **R22** — With 5,000 synthetic notes of about 300 words each, a benchmark
  SHOULD show: building the index by `Put` in under 2 s, `Search` in under 20 ms
  per query, and `Walk` at depth 3 in under 20 ms, on the development machine.
  The numbers are recorded in the stage plan's verification notes; missing one
  is a finding for the design's scale risk, not a failed test.

## Error handling

| Error | Raised by | Tool layer maps it to (SPEC-0004) |
|---|---|---|
| `index.ErrEmptyQuery` | R13 | an invalid-argument error |
| `index.ErrBadDepth` | R16 | an invalid-argument error |
| `index.ErrNotFound` | R19 | not found |

Resolution never errors: `ambiguous` and `dangling` are results (R4).

## Acceptance criteria — "Done looks like"

- [x] **R1, R2** — A property test applies random sequences of `Put` and `Remove`
  and compares every operation's result with a fresh index built from the final
  set.
- [x] **R3, R4** — Table test: `[[network/intpxy]]`, `network/intpxy.md`,
  `./network/intpxy`, a unique title, a title shared by two notes (ambiguous,
  even though one note's stem also matches), a unique alias, a stem shared by
  `hosts/intpxy.md` and `services/intpxy.md` (ambiguous), and an unknown target.
- [x] **R5** — A note with alias `old/place` resolves `[[old/place]]` with
  `Stale` true; a plain alias gives `Stale` false.
- [x] **R6** — A note whose two aliases normalise the same resolves, not
  ambiguous.
- [ ] **R7** — Tokenizer table: `PA-850`, `C#`, `10.11.0.51`, `10.11.0.51.`,
  `11.1.16-h2`, `fw-inf-oob-net-eu1-fw1`, `aa:bb:cc`, `a - b`, `a--b`,
  `http://x`, `Größe`, mixed English and German, empty string.
- [ ] **R24** — Query terms of `IT-104 rollback` are `it-104`, `rollback`; on a
  fixture where one note holds `IT-104` and nine hold `104` or `IT` alone, a
  search for `IT-104` returns the one note, and a search for `104` returns every
  note with `104`, the compound one included. Field lengths count compounds.
- [x] **R8, R9** — On a three-note fixture, scores match hand-computed BM25F
  values to 1e-9; a title match outranks the same term only in the body; a term
  found only in a tag gives a hit; a note with two of three query terms ranks
  above one with a single term.
- [x] **R10, R11, R12** — Filters (type, two tags, folder boundary), ordering
  ties by path, default and capped limit.
- [ ] **R25** — `Total` with 25 hits and the default limit is 25 while 10 hits
  are returned; hits carry their tags.
- [x] **R13** — `"--- !!"` returns `ErrEmptyQuery`.
- [x] **R14** — Snippet from an observation, from the body, title-only match,
  a line over 200 runes.
- [x] **R15, R16, R18, R19** — Graph fixture: relations, body links and
  backlinks at depth 1 to 3, `relTypes` filter, dangling and ambiguous edges of
  the start listed but not followed, 250 connected notes truncated at 200,
  unknown start.
- [ ] **R17** — On the graph fixture at depth 2: an edge between two notes at
  distance 2 is listed; with `relTypes`, an edge of another type between two
  listed notes is not; a note that links to the same target twice gives one
  edge at the first line, while a `links_to` and a relation to the same target
  give two; a self-link is hidden.
- [x] **R20, R21** — Listing with each filter; link report ordering and kinds.
- [ ] **R23** — Table test: path, title, alias and stem resolutions give their
  `Step`; a path ref whose text is also another note's title gives that note in
  `Also`; ambiguous and dangling results leave both empty.
- [ ] **R26** — A listed note without `updated` carries the `ModTime` it was
  `Put` with.
- [x] **R22** — `BenchmarkBuild`, `BenchmarkSearch`, `BenchmarkWalk` exist and
  their results on the development machine are recorded in the stage plan.
- [ ] **R22 (again)** — The benchmarks are rerun with compound tokens and the
  edges among listed notes, and the results are recorded in stage plan 0004.

## References

- Design: D9, D14, D23, D24, "In-memory index", "MCP tools" in
  [`../../design/solution-design.md`](../../design/solution-design.md).
- [ADR-0017](../../adr/0017-path-identity-and-link-resolution.md).
- Live check of 2026-11-20, findings C1, C2, L3, L4, L6, L8:
  [`2026-11-20-live-check.md`](../../design/2026-11-20-live-check.md).
- SPEC-0001 (note parser); SPEC-0003 (notes root and freshness check),
  SPEC-0004 (MCP read tools), SPEC-0005 (`memctl check`) — planned.
