# go-tabwrap

[![Go Reference](https://pkg.go.dev/badge/github.com/apstndb/go-tabwrap.svg)](https://pkg.go.dev/github.com/apstndb/go-tabwrap)

Tab-aware, grapheme-cluster-aware display width utilities for Go terminal
output.

It provides `StringWidth`, `ExpandTab`, `ExpandTabFunc`, `Wrap`, `Truncate`,
`FillLeft`, and `FillRight` for CLI tables, tree renderers, and other
fixed-width terminal layouts. Full behavior notes and runnable examples live on
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
`Condition.ControlSequences8Bit`; see the package documentation for that
compatibility note and the full width model.

## Acknowledgements

This package stands on the shoulders of:

- [mattn/go-runewidth](https://github.com/mattn/go-runewidth) — the long-standing
  standard for terminal string width in Go.
- [clipperhouse/displaywidth](https://github.com/clipperhouse/displaywidth) — the
  grapheme-cluster-aware width engine used by go-tabwrap.

## License

MIT
