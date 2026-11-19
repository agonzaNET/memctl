# ADR-0016 - Log one line per tool call, without note content

## Status

Accepted

**Area:** Tools

## Context

When a tool behaves oddly (slow search, a conflict loop, a write that failed),
the log on stderr is the only record of what the client asked. Notes hold
personal and work information, so a log that copies queries, paths or note
text would spread that content into client log files that are kept and shared
far less carefully than the notes themselves.

## Decision

> After each tool call, `memctl` logs one `INFO` line named `tool call` to
> stderr as JSON, with the tool, the duration, the error code if any, and
> counts. It never logs note content, queries, titles or paths.

- **Fields:** `tool`, `duration_ms`, `error` (the code, only on an error), and
  fixed-name counts the tool sets, such as `results`, `notes`, `depth`.
- **Never:** note bodies, frontmatter values, search queries, titles, aliases,
  paths, or the text of an error message.
- Startup, shutdown, the index build and git failures log their own lines;
  `DEBUG` may add timing detail, still without content.
- stdout carries the MCP protocol and is never used for logs.

## Alternatives Considered

| Alternative | Reason not chosen |
|---|---|
| Log the full request and result | Copies note content into client logs. |
| Log nothing per call | No way to see which tool was slow or failed. |
| Log paths but not content | Paths and titles already say a lot about a person's notes. |

## Consequences

- **Positive:** Slow or failing calls are visible without exposing notes.
- **Negative:** Reproducing a problem needs the person to say which note or
  query was involved.

## References

- Design, "Deployment and config", in [`../design/solution-design.md`](../design/solution-design.md).
