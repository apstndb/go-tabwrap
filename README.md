# go-tabwrap

[![Go Reference](https://pkg.go.dev/badge/github.com/apstndb/go-tabwrap.svg)](https://pkg.go.dev/github.com/apstndb/go-tabwrap)

Tab-aware, grapheme-cluster-aware display width utilities for Go terminal
output.

It provides `StringWidth`, `ExpandTab`, `ExpandTabFunc`, `Cut`, `Wrap`,
`WrapLines`, `Truncate`, `TruncateInfo`, `FillLeft`, and `FillRight` for CLI
tables, tree renderers, and other fixed-width terminal layouts. Full behavior
notes and runnable examples live on
[pkg.go.dev](https://pkg.go.dev/github.com/apstndb/go-tabwrap).

## Install

```sh
go get github.com/apstndb/go-tabwrap
```

## Showcase

```go
package main

import (
	"fmt"

	"github.com/apstndb/go-tabwrap"
)

func main() {
	fmt.Println(tabwrap.StringWidth("a\tb"))                       // 5
	fmt.Printf("%q\n", tabwrap.Truncate("hello world", 8, "...")) // "hello..."
	fmt.Printf("%q\n", tabwrap.FillLeft("42", 5))                  // "   42"
	fmt.Printf("%q\n", tabwrap.FillRight("42", 5))                 // "42   "

	c := &tabwrap.Condition{TrimTrailingSpace: true}
	fmt.Printf("%q\n", c.Wrap("hello world", 5))                   // "hello\n worl\nd"
	fmt.Printf("%q\n", c.Wrap("ab\tcd", 4))                        // "ab\ncd"
}
```

`Truncate` is a fitting helper and intentionally ignores
`Condition.ControlSequences8Bit`; `TruncateInfo` also reports the fitted width
and whether truncation occurred. A non-positive truncation cap returns an empty
string, while non-positive `Wrap`, `Cut`, and `WrapLines` lane widths are
unbounded. See the package documentation for that distinction, source-preserving
cuts, structured wrapping, and LF/CRLF/CR line-break handling.

## Acknowledgements

This package stands on the shoulders of:

- [mattn/go-runewidth](https://github.com/mattn/go-runewidth) — the long-standing
  standard for terminal string width in Go.
- [clipperhouse/displaywidth](https://github.com/clipperhouse/displaywidth) — the
  grapheme-cluster-aware width engine used by go-tabwrap.

## License

MIT
