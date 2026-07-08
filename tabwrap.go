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
	// ControlSequences8Bit treats 8-bit C1 ECMA-48 escape sequences as zero-width
	// when true. It can be enabled independently of ControlSequences; enabling
	// both covers both the 7-bit and 8-bit forms. Truncate follows displaywidth
	// and ignores this option.
	ControlSequences8Bit bool
	// TrimTrailingSpace removes trailing spaces and tabs from each output line
	// produced by Wrap when true. This applies after wrapping, while preserving
	// trailing zero-width graphemes on the line (for example, ANSI control
	// sequences when ControlSequences or ControlSequences8Bit are enabled).
	TrimTrailingSpace bool
}

// NewCondition returns a Condition with default settings (TabWidth = 4).
func NewCondition() *Condition {
	return &Condition{TabWidth: 4}
}

// Clone returns an independent copy of c with defaults normalized.
//
// If c is nil, Clone returns [NewCondition]().
func (c *Condition) Clone() *Condition {
	if c == nil {
		return NewCondition()
	}
	clone := *c
	if clone.TabWidth <= 0 {
		clone.TabWidth = 4
	}
	return &clone
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

type scanToken struct {
	text      string
	width     int
	tab       bool
	lineBreak bool
}

type displayScanner struct {
	graphemes displaywidth.Graphemes[string]
	tabWidth  int
	col       int
	maxWidth  int
}

func (c *Condition) newDisplayScanner(s string, opts displaywidth.Options) displayScanner {
	return displayScanner{
		graphemes: opts.StringGraphemes(s),
		tabWidth:  c.tabWidth(),
	}
}

func (s *displayScanner) next() (scanToken, bool) {
	if !s.graphemes.Next() {
		return scanToken{}, false
	}

	text := s.graphemes.Value()
	token := scanToken{
		text: text,
	}

	if isLineBreak(text) {
		token.lineBreak = true
		s.finishLine()
		s.col = 0
		return token, true
	}

	if text == "\t" {
		token.tab = true
		token.width = s.tabWidth - s.col%s.tabWidth
	} else {
		token.width = s.graphemes.Width()
	}
	s.col += token.width
	return token, true
}

func (s *displayScanner) finishLine() {
	if s.col > s.maxWidth {
		s.maxWidth = s.col
	}
}

func (s *displayScanner) lineWidth() int {
	s.finishLine()
	return s.maxWidth
}

func (c *Condition) stringWidth(s string, opts displaywidth.Options) int {
	scanner := c.newDisplayScanner(s, opts)
	for {
		if _, ok := scanner.next(); !ok {
			break
		}
	}
	return scanner.lineWidth()
}

// StringWidth returns the display width of s in terminal columns.
//
// Width is measured by grapheme cluster, not rune. Tabs expand to tab stops,
// LF, CRLF, and CR line breaks reset the column, and for multi-line strings
// the result is the width of the widest line. EastAsianWidth, ControlSequences, and
// ControlSequences8Bit affect how individual graphemes are counted.
func (c *Condition) StringWidth(s string) int {
	return c.stringWidth(s, c.options())
}

// ExpandTab replaces every tab with spaces according to tab stops.
// Columns reset at each line break.
func (c *Condition) ExpandTab(s string) string {
	return c.expandTabFunc(s, c.options(), func(nSpaces int) string {
		return strings.Repeat(" ", nSpaces)
	})
}

// ExpandTabFunc replaces every tab by calling fn with the number of spaces
// the tab would normally expand to (based on the current column and tab width).
// The column advances by nSpaces regardless of what fn returns, so the caller
// is responsible for returning a string whose display width equals nSpaces if
// alignment matters. Columns reset at each line break.
//
// ExpandTabFunc panics if fn is nil and s contains a tab, because fn is only
// called when a tab is encountered.
func (c *Condition) ExpandTabFunc(s string, fn func(nSpaces int) string) string {
	return c.expandTabFunc(s, c.options(), fn)
}

func (c *Condition) expandTabFunc(s string, opts displaywidth.Options, fn func(nSpaces int) string) string {
	expanded, _ := c.expandTabFuncAndWidth(s, opts, fn)
	return expanded
}

func (c *Condition) expandTabFuncAndWidth(s string, opts displaywidth.Options, fn func(nSpaces int) string) (string, int) {
	var b strings.Builder
	b.Grow(len(s))

	scanner := c.newDisplayScanner(s, opts)
	for {
		token, ok := scanner.next()
		if !ok {
			break
		}
		if token.lineBreak {
			b.WriteString(token.text)
			continue
		}
		if token.tab {
			b.WriteString(fn(token.width))
			continue
		}
		b.WriteString(token.text)
	}
	return b.String(), scanner.lineWidth()
}

func (c *Condition) expandTabSpacesWithOptions(s string, opts displaywidth.Options) string {
	expanded, _ := c.expandTabSpacesWithOptionsAndWidth(s, opts)
	return expanded
}

func (c *Condition) expandTabSpacesWithOptionsAndWidth(s string, opts displaywidth.Options) (string, int) {
	return c.expandTabFuncAndWidth(s, opts, func(nSpaces int) string {
		return strings.Repeat(" ", nSpaces)
	})
}

// Wrap wraps s to fit within width display columns.
//
// Tabs are indivisible tokens: if a tab does not fit on the current line the
// entire tab moves to the next line. Tabs in the output are expanded to
// spaces so the result is render-ready.
//
// Existing line breaks are preserved. When width <= 0 the string is returned
// with tabs expanded but no wrapping applied.
//
// When control-sequence handling is enabled, Wrap carries across line breaks
// only those SGR (Select Graphic Rendition) sequences that are recognized as
// zero-width under the active options: 7-bit sequences when ControlSequences
// is true, and 8-bit sequences when ControlSequences8Bit is true. For those
// sequences, a reset is emitted before each newline and the active SGR
// sequences are replayed after it so each output line remains independently
// styled.
func (c *Condition) Wrap(s string, width int) string {
	if width <= 0 {
		result := c.ExpandTab(s)
		if c.TrimTrailingSpace {
			return trimWrappedLinesRight(result, c.options())
		}
		return result
	}

	opts := c.options()
	tw := c.tabWidth()
	trackSGR := c.ControlSequences || c.ControlSequences8Bit
	resetSGR := "\x1b[0m"
	if c.ControlSequences8Bit && !c.ControlSequences {
		resetSGR = "\x9b0m"
	}

	var b strings.Builder
	b.Grow(len(s))
	col := 0
	var sgrState []string

	// emitLineBreak writes a line break. When SGR tracking is active, it emits
	// a reset before the line break and replays the current SGR state after it.
	emitLineBreak := func(lineBreak string) {
		if trackSGR && len(sgrState) > 0 {
			b.WriteString(resetSGR)
		}
		b.WriteString(lineBreak)
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

		// Track SGR sequences (zero-width escape sequences starting with ESC
		// or, when ControlSequences8Bit is enabled, with the 8-bit CSI byte 0x9b).
		if trackSGR && w == 0 && len(v) > 0 && (v[0] == '\x1b' || v[0] == '\x9b') {
			if isSGR(v) {
				if sgrStartsWithReset(v) {
					sgrState = sgrState[:0]
				}
				if !isSGRReset(v) {
					sgrState = append(sgrState, v)
				}
			}
			b.WriteString(v)
			continue
		}

		if isLineBreak(v) {
			emitLineBreak(v)
			col = 0
			continue
		}

		switch v {
		case "\t":
			spaces := tw - col%tw
			if col+spaces > width && col > 0 {
				emitLineBreak("\n")
				col = 0
				spaces = tw
			}
			for range spaces {
				b.WriteByte(' ')
			}
			col += spaces
		default:
			if col+w > width && col > 0 {
				emitLineBreak("\n")
				col = 0
			}
			b.WriteString(v)
			col += w
		}
	}
	result := b.String()
	if c.TrimTrailingSpace {
		return trimWrappedLinesRight(result, opts)
	}
	return result
}

func trimWrappedLinesRight(s string, opts displaywidth.Options) string {
	var b strings.Builder
	b.Grow(len(s))

	for {
		line, lineBreak, rest := cutLineBreak(s)
		b.WriteString(trimTrailingLineSpace(line, opts))
		b.WriteString(lineBreak)
		if lineBreak == "" {
			return b.String()
		}
		s = rest
	}
}

func trimTrailingLineSpace(s string, opts displaywidth.Options) string {
	if !opts.ControlSequences && !opts.ControlSequences8Bit {
		return strings.TrimRight(s, " \t")
	}

	gs := opts.StringGraphemes(s)
	lastNonSpace := -1
	lastVisible := -1
	count := 0

	for gs.Next() {
		if gs.Width() > 0 {
			lastVisible = count
			if gs.Value() != " " && gs.Value() != "\t" {
				lastNonSpace = count
			}
		}
		count++
	}

	if lastVisible == lastNonSpace {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))
	gs = opts.StringGraphemes(s)
	for i := 0; gs.Next(); i++ {
		if i <= lastNonSpace || gs.Width() == 0 {
			b.WriteString(gs.Value())
		}
	}

	return b.String()
}

func isLineBreak(s string) bool {
	return len(s) > 0 && lineBreakLenAt(s, 0) == len(s)
}

func lineBreakLenAt(s string, i int) int {
	if i >= len(s) {
		return 0
	}
	switch s[i] {
	case '\n':
		return 1
	case '\r':
		if i+1 < len(s) && s[i+1] == '\n' {
			return 2
		}
		return 1
	default:
		return 0
	}
}

func cutLineBreak(s string) (line string, lineBreak string, rest string) {
	for i := 0; i < len(s); i++ {
		if n := lineBreakLenAt(s, i); n > 0 {
			return s[:i], s[i : i+n], s[i+n:]
		}
	}
	return s, "", ""
}

func containsLineBreak(s string) bool {
	return strings.ContainsAny(s, "\r\n")
}

// isSGR reports whether s is a CSI SGR (Select Graphic Rendition) sequence.
// It recognises both 7-bit (ESC [ <params> m) and 8-bit (0x9b <params> m) forms.
func isSGR(s string) bool {
	params, ok := sgrParams(s)
	if !ok {
		return false
	}
	return !hasPrivateCSIParameter(params)
}

func sgrParams(s string) (string, bool) {
	if len(s) < 2 || s[len(s)-1] != 'm' {
		return "", false
	}

	// 7-bit: ESC [ ... m
	if len(s) >= 3 && s[0] == '\x1b' && s[1] == '[' {
		return s[2 : len(s)-1], true
	}
	// 8-bit: 0x9b ... m
	if s[0] == '\x9b' {
		return s[1 : len(s)-1], true
	}
	return "", false
}

func hasPrivateCSIParameter(params string) bool {
	if params == "" {
		return false
	}
	switch params[0] {
	case '<', '=', '>', '?':
		return true
	default:
		return false
	}
}

// isSGRReset reports whether s is an SGR reset sequence.
func isSGRReset(s string) bool {
	params, ok := sgrParams(s)
	return ok && !hasPrivateCSIParameter(params) && allSGRParamsReset(params)
}

func sgrStartsWithReset(s string) bool {
	params, ok := sgrParams(s)
	if !ok || hasPrivateCSIParameter(params) {
		return false
	}
	first, _, _ := strings.Cut(params, ";")
	return isZeroSGRParam(first)
}

func allSGRParamsReset(params string) bool {
	for {
		param, rest, found := strings.Cut(params, ";")
		if !isZeroSGRParam(param) {
			return false
		}
		if !found {
			return true
		}
		params = rest
	}
}

func isZeroSGRParam(param string) bool {
	for i := 0; i < len(param); i++ {
		if param[i] != '0' {
			return false
		}
	}
	return true
}

// TruncateResult describes the outcome of a truncation operation.
type TruncateResult struct {
	// Text is the truncation result.
	Text string
	// Width is the display width of Text using Truncate's width model.
	Width int
	// Truncated reports whether any input line was shortened or replaced.
	Truncated bool
}

// Truncate truncates s to fit within positive maxWidth display columns,
// appending tail if truncation occurs. Tabs are expanded before measuring. If
// tail itself is too wide to fit, it is truncated first so the result still
// fits maxWidth. Tabs in tail are expanded independently before fitting, not
// relative to the column where tail is appended. When maxWidth <= 0, tail is
// returned as-is.
//
// ControlSequences8Bit follows displaywidth and is ignored here, even when it
// is enabled for StringWidth and Wrap. This can make 8-bit C1 sequences count
// as zero-width for measurement but not for truncation; go-tabwrap keeps that
// behavior to avoid mis-parsing UTF-8 byte sequences as standalone C1 controls.
func (c *Condition) Truncate(s string, maxWidth int, tail string) string {
	return c.TruncateInfo(s, maxWidth, tail).Text
}

// TruncateInfo truncates s like [Condition.Truncate] and also returns the
// resulting display width and whether truncation occurred.
//
// Multi-line input is truncated line by line so the reported width follows the
// package width model: the widest output line determines Width.
func (c *Condition) TruncateInfo(s string, maxWidth int, tail string) TruncateResult {
	opts := c.options()
	opts.ControlSequences8Bit = false

	if maxWidth <= 0 {
		return c.truncateLineInfo(s, maxWidth, tail, opts)
	}

	if strings.Contains(tail, "\t") {
		tail = c.expandTabSpacesWithOptions(tail, opts)
	}
	tail = opts.TruncateString(tail, maxWidth, "")

	if !containsLineBreak(s) {
		return c.truncateLineInfo(s, maxWidth, tail, opts)
	}

	var b strings.Builder
	b.Grow(len(s))
	result := TruncateResult{}
	for {
		line, lineBreak, rest := cutLineBreak(s)
		lineResult := c.truncateLineInfo(line, maxWidth, tail, opts)
		b.WriteString(lineResult.Text)
		if lineResult.Width > result.Width {
			result.Width = lineResult.Width
		}
		result.Truncated = result.Truncated || lineResult.Truncated
		b.WriteString(lineBreak)
		if lineBreak == "" {
			break
		}
		s = rest
	}
	result.Text = b.String()
	return result
}

func (c *Condition) truncateLineInfo(s string, maxWidth int, tail string, opts displaywidth.Options) TruncateResult {
	if maxWidth <= 0 {
		return TruncateResult{
			Text:      tail,
			Width:     c.stringWidth(tail, opts),
			Truncated: s != tail,
		}
	}

	expanded := s
	width := c.stringWidth(expanded, opts)
	if strings.Contains(s, "\t") {
		expanded, width = c.expandTabSpacesWithOptionsAndWidth(s, opts)
	}
	if width <= maxWidth {
		return TruncateResult{
			Text:  expanded,
			Width: width,
		}
	}

	text := opts.TruncateString(expanded, maxWidth, tail)
	return TruncateResult{
		Text:      text,
		Width:     c.stringWidth(text, opts),
		Truncated: true,
	}
}

// FillLeft pads s on the left with spaces to reach width display columns.
// For multi-line strings, padding is added only at the start of the full
// string, so only the first line changes. If another line is already at least
// width columns wide, s is returned unchanged. Width is measured using the same
// rules as [Condition.StringWidth]. When left padding is needed, tabs in the
// first line are expanded first so the added spaces do not shift later tab
// stops there.
func (c *Condition) FillLeft(s string, width int) string {
	opts := c.options()
	sw := c.stringWidth(s, opts)
	if sw >= width {
		return s
	}
	first, lineBreak, rest := cutLineBreak(s)
	var firstWidth int
	if strings.Contains(first, "\t") {
		first, firstWidth = c.expandTabSpacesWithOptionsAndWidth(first, opts)
	} else if lineBreak == "" {
		firstWidth = sw
	} else {
		firstWidth = c.stringWidth(first, opts)
	}
	pad := width - firstWidth
	var b strings.Builder
	totalLen := len(first) + pad
	if lineBreak != "" {
		totalLen += len(lineBreak) + len(rest)
	}
	b.Grow(totalLen)
	b.WriteString(strings.Repeat(" ", pad))
	b.WriteString(first)
	if lineBreak != "" {
		b.WriteString(lineBreak)
		b.WriteString(rest)
	}
	return b.String()
}

// FillRight pads s on the right with spaces to reach width display columns.
// For multi-line strings, padding is computed from the widest line but is
// added only at the end of the full string, so only the last line changes.
// Width is measured using the same rules as [Condition.StringWidth].
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
// See [Condition.StringWidth] for the width model.
func StringWidth(s string) int {
	return defaultCondition.StringWidth(s)
}

// ExpandTab replaces every tab with spaces using default settings.
func ExpandTab(s string) string {
	return defaultCondition.ExpandTab(s)
}

// ExpandTabFunc replaces every tab using a custom callback with default settings.
//
// ExpandTabFunc panics if fn is nil and s contains a tab, because fn is only
// called when a tab is encountered.
func ExpandTabFunc(s string, fn func(nSpaces int) string) string {
	return defaultCondition.ExpandTabFunc(s, fn)
}

// Wrap wraps s to fit within width display columns using default settings.
func Wrap(s string, width int) string {
	return defaultCondition.Wrap(s, width)
}

// Truncate truncates s using default settings.
func Truncate(s string, maxWidth int, tail string) string {
	return defaultCondition.Truncate(s, maxWidth, tail)
}

// TruncateInfo truncates s using default settings and returns width metadata.
func TruncateInfo(s string, maxWidth int, tail string) TruncateResult {
	return defaultCondition.TruncateInfo(s, maxWidth, tail)
}

// FillLeft pads s on the left using default settings.
func FillLeft(s string, width int) string {
	return defaultCondition.FillLeft(s, width)
}

// FillRight pads s on the right using default settings.
func FillRight(s string, width int) string {
	return defaultCondition.FillRight(s, width)
}
