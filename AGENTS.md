# AGENTS.md — go-tabwrap agent guide

## What this module is

`github.com/apstndb/go-tabwrap` is a small, dependency-light Go library providing
tab-aware, grapheme-cluster-aware display width operations for terminal output:
`StringWidth`, `ExpandTab`, `ExpandTabFunc`, `Wrap`, `Truncate`, `FillLeft`,
`FillRight`, all available as package-level functions (default settings) and as
methods on `Condition`. It wraps `github.com/clipperhouse/displaywidth` (grapheme
iteration and per-grapheme width) and adds tab-stop expansion, wrapping,
truncation, and padding on top.

## Width model (invariants to preserve)

- Width is measured in terminal display columns per grapheme cluster (UAX #29),
  never per rune.
- Tabs expand to the next tab stop (`TabWidth`, default 4; `<= 0` falls back to 4).
- Newlines reset the column; the width of a multi-line string is the width of its
  widest line.
- `EastAsianWidth`, `ControlSequences` (7-bit), and `ControlSequences8Bit` follow
  `displaywidth.Options` semantics. Exception: `Truncate` deliberately forces
  `ControlSequences8Bit = false` (parsing raw C1 bytes during truncation can break
  UTF-8 boundaries — documented in godoc and README).
- `Wrap` treats tabs as indivisible tokens and carries SGR state across line
  breaks (reset before newline, replay after).

## Layout

Single-package flat layout: `tabwrap.go` (all code), `tabwrap_test.go`,
`tabwrap_bench_test.go`. Keep it flat; do not introduce subpackages without a
strong reason.

## Verification

```
go test -race ./...
go vet ./...
golangci-lint run
go test -bench . -benchmem   # when touching hot paths
```

All three of test/vet/lint must be clean before pushing. CI (GitHub Actions) runs
the test matrix on Linux/macOS/Windows plus golangci-lint.

## Versioning and release policy

- v0 policy: breaking changes bump **minor**; everything else (including new public
  API) bumps **patch**. Never re-tag a published version.
- GitHub release notes are the per-version source of truth for behavior changes and
  version requirements. No in-repo CHANGELOG.
- User-facing guidance belongs in godoc (doc.go sections and runnable `Example` tests);
  README stays a thin showcase pointing to pkg.go.dev.

## Known importers (check before breaking changes)

- `github.com/apstndb/spanner-mycli` (`internal/mycli`, `internal/mycli/format`) —
  table/streaming formatters; uses `Condition` literals, `StringWidth`, `Truncate`,
  `FillLeft`, `FillRight`, incl. `ControlSequences: true` for styled cells.
- `github.com/apstndb/spannerplan` (`asciitable`, `plantree`, `treerender`) — uses
  `NewCondition()` with `TrimTrailingSpace = true`.

## Current improvement backlog

See open GitHub issues for deferred/backlog items.
