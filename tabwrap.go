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
	// ControlSequences8Bit treats 8-bit ECMA-48 escape sequences as zero-width
	// when true. This extends ControlSequences to cover the 8-bit C1 control
	// codes (0x80–0x9F based sequences).
	ControlSequences8Bit bool
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
		EastAsianWidth:       c.EastAsianWidth,
		ControlSequences:     c.ControlSequences,
		ControlSequences8Bit: c.ControlSequences8Bit,
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
	return c.ExpandTabFunc(s, func(nSpaces int) string {
		return strings.Repeat(" ", nSpaces)
	})
}

// ExpandTabFunc replaces every tab by calling fn with the number of spaces
// the tab would normally expand to (based on the current column and tab width).
// The column advances by nSpaces regardless of what fn returns, so the caller
// is responsible for returning a string whose display width equals nSpaces if
// alignment matters. Columns reset at each newline.
//
// ExpandTabFunc panics if fn is nil.
func (c *Condition) ExpandTabFunc(s string, fn func(nSpaces int) string) string {
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
			nSpaces := tw - col%tw
			b.WriteString(fn(nSpaces))
			col += nSpaces
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
//
// When ControlSequences is true, SGR (Select Graphic Rendition) state is
// carried across line breaks: a reset is emitted before each newline and the
// active SGR sequences are replayed after it. This ensures each output line
// is independently styled.
func (c *Condition) Wrap(s string, width int) string {
	if width <= 0 {
		return c.ExpandTab(s)
	}

	opts := c.options()
	tw := c.tabWidth()
	trackSGR := c.ControlSequences

	var b strings.Builder
	b.Grow(len(s))
	col := 0
	var sgrState []string

	// emitNewline writes a line break. When SGR tracking is active, it emits
	// a reset before the newline and replays the current SGR state after it.
	emitNewline := func() {
		if trackSGR && len(sgrState) > 0 {
			b.WriteString("\x1b[0m")
		}
		b.WriteByte('\n')
		if trackSGR {
			for _, seq := range sgrState {
				b.WriteString(seq)
			}
		}
	}

	gs := opts.StringGraphemes(s)
	for gs.Next() {
		v := gs.Value()
		w := gs.Width()

		// Track SGR sequences (zero-width escape sequences starting with ESC).
		if trackSGR && w == 0 && len(v) > 0 && v[0] == '\x1b' {
			if isSGR(v) {
				if isSGRReset(v) {
					sgrState = sgrState[:0]
				} else {
					sgrState = append(sgrState, v)
				}
			}
			b.WriteString(v)
			continue
		}

		switch v {
		case "\n":
			emitNewline()
			col = 0
		case "\t":
			spaces := tw - col%tw
			if col+spaces > width && col > 0 {
				emitNewline()
				col = 0
				spaces = tw
			}
			for range spaces {
				b.WriteByte(' ')
			}
			col += spaces
		default:
			if col+w > width && col > 0 {
				emitNewline()
				col = 0
			}
			b.WriteString(v)
			col += w
		}
	}
	return b.String()
}

// isSGR reports whether s is a CSI SGR (Select Graphic Rendition) sequence.
// SGR sequences have the form ESC [ <params> m.
func isSGR(s string) bool {
	return len(s) >= 3 && s[0] == '\x1b' && s[1] == '[' && s[len(s)-1] == 'm'
}

// isSGRReset reports whether s is an SGR reset sequence.
func isSGRReset(s string) bool {
	return s == "\x1b[0m" || s == "\x1b[m"
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
