// Package tabwrap provides tab-aware, grapheme-cluster-aware display width
// operations for terminal/fixed-width output.
//
// It wraps [clipperhouse/displaywidth] to add tab-stop handling, line wrapping,
// truncation, and padding — the common building blocks for CLI table renderers
// and TUI applications.
//
// Key differences from [mattn/go-runewidth]:
//   - Grapheme-cluster-aware (emoji, combining characters) via displaywidth.
//   - Built-in tab-stop expansion in every operation.
//
// Key additions over [clipperhouse/displaywidth]:
//   - Tab-aware StringWidth, ExpandTab, Wrap, Truncate, FillLeft, FillRight.
package tabwrap

import (
	"strings"

	"github.com/clipperhouse/displaywidth"
)

// Condition configures display width behaviour.
type Condition struct {
	// TabWidth is the number of columns per tab stop. Zero or negative defaults to 4.
	TabWidth int
	// EastAsianWidth treats ambiguous East Asian characters as width 2 when true.
	EastAsianWidth bool
	// ControlSequences treats 7-bit ANSI escape sequences (CSI, OSC, etc.)
	// as zero-width when true. This allows correct width measurement of
	// strings containing terminal color codes and other SGR sequences.
	ControlSequences bool
}

// NewCondition returns a Condition with default settings (TabWidth = 4).
func NewCondition() *Condition {
	return &Condition{TabWidth: 4}
}

func (c *Condition) tabWidth() int {
	if c.TabWidth <= 0 {
		return 4
	}
	return c.TabWidth
}

func (c *Condition) options() displaywidth.Options {
	return displaywidth.Options{
		EastAsianWidth:   c.EastAsianWidth,
		ControlSequences: c.ControlSequences,
	}
}

// StringWidth returns the display width of s, handling tabs and newlines.
// For multi-line strings it returns the width of the widest line.
func (c *Condition) StringWidth(s string) int {
	opts := c.options()
	tw := c.tabWidth()

	maxW := 0
	col := 0
	gs := opts.StringGraphemes(s)
	for gs.Next() {
		v := gs.Value()
		switch v {
		case "\n":
			if col > maxW {
				maxW = col
			}
			col = 0
		case "\t":
			col += tw - col%tw
		default:
			col += gs.Width()
		}
	}
	if col > maxW {
		maxW = col
	}
	return maxW
}

// ExpandTab replaces every tab with spaces according to tab stops.
// Columns reset at each newline.
func (c *Condition) ExpandTab(s string) string {
	opts := c.options()
	tw := c.tabWidth()

	var b strings.Builder
	b.Grow(len(s))
	col := 0
	gs := opts.StringGraphemes(s)
	for gs.Next() {
		v := gs.Value()
		switch v {
		case "\n":
			b.WriteByte('\n')
			col = 0
		case "\t":
			spaces := tw - col%tw
			for range spaces {
				b.WriteByte(' ')
			}
			col += spaces
		default:
			b.WriteString(v)
			col += gs.Width()
		}
	}
	return b.String()
}

// Wrap wraps s to fit within width display columns.
//
// Tabs are indivisible tokens: if a tab does not fit on the current line the
// entire tab moves to the next line. Tabs in the output are expanded to
// spaces so the result is render-ready.
//
// Existing newlines are preserved. When width <= 0 the string is returned
// with tabs expanded but no wrapping applied.
func (c *Condition) Wrap(s string, width int) string {
	if width <= 0 {
		return c.ExpandTab(s)
	}

	opts := c.options()
	tw := c.tabWidth()

	var b strings.Builder
	b.Grow(len(s))
	col := 0

	gs := opts.StringGraphemes(s)
	for gs.Next() {
		v := gs.Value()
		switch v {
		case "\n":
			b.WriteByte('\n')
			col = 0
		case "\t":
			spaces := tw - col%tw
			if col+spaces > width && col > 0 {
				b.WriteByte('\n')
				col = 0
				spaces = tw
			}
			for range spaces {
				b.WriteByte(' ')
			}
			col += spaces
		default:
			w := gs.Width()
			if col+w > width && col > 0 {
				b.WriteByte('\n')
				col = 0
			}
			b.WriteString(v)
			col += w
		}
	}
	return b.String()
}

// Truncate truncates s to fit within maxWidth display columns, appending tail
// if truncation occurs. Tabs are expanded before measuring.
func (c *Condition) Truncate(s string, maxWidth int, tail string) string {
	if maxWidth <= 0 {
		return tail
	}

	if !strings.Contains(s, "\t") {
		return c.options().TruncateString(s, maxWidth, tail)
	}

	expanded := c.ExpandTab(s)
	return c.options().TruncateString(expanded, maxWidth, tail)
}

// FillLeft pads s on the left with spaces to reach width display columns.
// If s is already at least width columns wide it is returned unchanged.
func (c *Condition) FillLeft(s string, width int) string {
	w := c.StringWidth(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

// FillRight pads s on the right with spaces to reach width display columns.
// If s is already at least width columns wide it is returned unchanged.
func (c *Condition) FillRight(s string, width int) string {
	w := c.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// Package-level convenience functions use a default Condition (TabWidth = 4).
var defaultCondition = NewCondition()

// StringWidth returns the display width of s using default settings.
func StringWidth(s string) int {
	return defaultCondition.StringWidth(s)
}

// Truncate truncates s using default settings.
func Truncate(s string, maxWidth int, tail string) string {
	return defaultCondition.Truncate(s, maxWidth, tail)
}

// FillLeft pads s on the left using default settings.
func FillLeft(s string, width int) string {
	return defaultCondition.FillLeft(s, width)
}

// FillRight pads s on the right using default settings.
func FillRight(s string, width int) string {
	return defaultCondition.FillRight(s, width)
}
