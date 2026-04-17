# go-tabwrap

Tab-aware, grapheme-cluster-aware display width utilities for Go.

Provides `StringWidth`, `ExpandTab`, `Wrap`, `Truncate`, `FillLeft`, and `FillRight` — the common building blocks for CLI table renderers and TUI applications.

## Features

- **Grapheme-cluster-aware** — emoji sequences and combining characters are measured correctly (via [displaywidth]).
- **Tab-stop expansion** — every operation handles `\t` as an elastic tab stop, not a single character.
- **Line wrapping** — `Wrap` breaks text to fit a column width. Tabs are indivisible: if a tab does not fit, it moves to the next line.
- **Optional trailing-space trimming** — `Condition.TrimTrailingSpace` removes trailing spaces and tabs from each output line produced by `Wrap`.
- **ANSI escape sequence aware** — optional `ControlSequences` mode treats SGR and other 7-bit escape sequences as zero-width, allowing correct measurement of styled terminal output. `Wrap` carries SGR state across line breaks, so each output line is independently styled.
- **East Asian Width** — optional treatment of ambiguous characters as double-width.

## Width semantics

`StringWidth` measures terminal display columns by **grapheme cluster**, not by
rune count. That means emoji sequences, combining characters, and other
multi-rune graphemes are counted as a single visible unit according to
[displaywidth](https://github.com/clipperhouse/displaywidth).

Tabs expand to tab stops, newlines reset the column, and the width of a
multi-line string is the width of its widest line. `EastAsianWidth`,
`ControlSequences`, and `ControlSequences8Bit` adjust how individual graphemes
are counted, and `FillLeft`, `FillRight`, and `Wrap` all use the same width
model.

## Install

```
go get github.com/apstndb/go-tabwrap
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/apstndb/go-tabwrap"
)

func main() {
	// Package-level functions use default settings (TabWidth = 4).
	fmt.Println(tabwrap.StringWidth("hello"))        // 5
	fmt.Println(tabwrap.StringWidth("a\tb"))          // 5 (tab expands to 3 spaces)
	fmt.Println(tabwrap.StringWidth("日本語"))        // 6

	fmt.Println(tabwrap.Truncate("hello world", 8, "...")) // "hello..."
	fmt.Println(tabwrap.FillLeft("42", 5))                  // "   42"
	fmt.Println(tabwrap.FillRight("hi", 5))                 // "hi   "

	// Use Condition for custom tab width or East Asian Width.
	c := &tabwrap.Condition{TabWidth: 8}
	fmt.Println(c.StringWidth("\t"))            // 8
	fmt.Println(c.Wrap("hello world", 5))       // "hello\n world"
	fmt.Println(c.ExpandTab("a\tb"))            // "a       b"

	trimmed := &tabwrap.Condition{TabWidth: 4, TrimTrailingSpace: true}
	fmt.Println(trimmed.Wrap("ab\tcd", 4))      // "ab\ncd"

	// ANSI escape sequences: measure visible width only.
	ansi := &tabwrap.Condition{TabWidth: 4, ControlSequences: true}
	styled := "\x1b[31mhello\x1b[0m"
	fmt.Println(ansi.StringWidth(styled))       // 5 (escape sequences ignored)

	// Wrap carries SGR state across line breaks.
	wrapped := ansi.Wrap("\x1b[31mhelloworld\x1b[0m", 5)
	// Result: "\x1b[31mhello\x1b[0m\n\x1b[31mworld\x1b[0m"
	// Each line is independently styled.
	fmt.Println(wrapped)
}
```

## API

### Package-level (default: TabWidth = 4)

| Function | Description |
|---|---|
| `StringWidth(s) int` | Display width of s (tab & newline aware) |
| `ExpandTab(s) string` | Replace tabs with spaces |
| `ExpandTabFunc(s, fn) string` | Replace tabs using a custom callback |
| `Wrap(s, width) string` | Wrap to width columns (tabs expanded) |
| `Truncate(s, maxWidth, tail) string` | Truncate s, append tail if truncated |
| `FillLeft(s, width) string` | Left-pad with spaces |
| `FillRight(s, width) string` | Right-pad with spaces |

### Condition

`Condition` provides all the above functions as methods, with configurable fields:

| Field | Default | Description |
|---|---|---|
| `TabWidth` | 4 | Columns per tab stop |
| `EastAsianWidth` | false | Treat ambiguous EA chars as width 2 |
| `ControlSequences` | false | Treat 7-bit ANSI escapes as zero-width |
| `ControlSequences8Bit` | false | Treat 8-bit ECMA-48 escapes as zero-width |
| `TrimTrailingSpace` | false | Trim trailing spaces and tabs from each `Wrap` output line |

Additional methods:

| Method | Description |
|---|---|
| `ExpandTab(s) string` | Replace tabs with spaces |
| `ExpandTabFunc(s, fn) string` | Replace tabs using a custom callback |
| `Wrap(s, width) string` | Wrap to width columns (tabs expanded) |

## Acknowledgements

This package stands on the shoulders of:

- [mattn/go-runewidth](https://github.com/mattn/go-runewidth) — the long-standing standard for terminal string width in Go. go-tabwrap provides a similar API shape while adding tab-awareness and grapheme-cluster support.
- [clipperhouse/displaywidth](https://github.com/clipperhouse/displaywidth) — the underlying grapheme-cluster-aware width engine that powers go-tabwrap. go-tabwrap adds tab-stop handling, wrapping, truncation, and padding on top.

## License

MIT

[displaywidth]: https://github.com/clipperhouse/displaywidth
