package tabwrap

import (
	"strings"

	"github.com/clipperhouse/displaywidth"
)

// Condition configures display width behaviour. Read-only operations use value
// receivers, so callers can copy and reuse a Condition as a small option value.
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
	// produced by Wrap or WrapLines when true. This applies after wrapping, while
	// preserving trailing zero-width graphemes on the line (for example, ANSI
	// control sequences when ControlSequences or ControlSequences8Bit are enabled).
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

func (c Condition) tabWidth() int {
	if c.TabWidth <= 0 {
		return 4
	}
	return c.TabWidth
}

func (c Condition) options() displaywidth.Options {
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

func (c Condition) newDisplayScanner(s string, opts displaywidth.Options) displayScanner {
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

func (c Condition) stringWidth(s string, opts displaywidth.Options) int {
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
func (c Condition) StringWidth(s string) int {
	return c.stringWidth(s, c.options())
}

// ExpandTab replaces every tab with spaces according to tab stops.
// Columns reset at each line break.
func (c Condition) ExpandTab(s string) string {
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
func (c Condition) ExpandTabFunc(s string, fn func(nSpaces int) string) string {
	return c.expandTabFunc(s, c.options(), fn)
}

func (c Condition) expandTabFunc(s string, opts displaywidth.Options, fn func(nSpaces int) string) string {
	expanded, _ := c.expandTabFuncAndWidth(s, opts, fn)
	return expanded
}

func (c Condition) expandTabFuncAndWidth(s string, opts displaywidth.Options, fn func(nSpaces int) string) (string, int) {
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

func (c Condition) expandTabSpacesWithOptions(s string, opts displaywidth.Options) string {
	expanded, _ := c.expandTabSpacesWithOptionsAndWidth(s, opts)
	return expanded
}

func (c Condition) expandTabSpacesWithOptionsAndWidth(s string, opts displaywidth.Options) (string, int) {
	return c.expandTabFuncAndWidth(s, opts, func(nSpaces int) string {
		return strings.Repeat(" ", nSpaces)
	})
}

// CutResult describes one source-preserving display-width cut.
type CutResult struct {
	// Text is the render-ready fragment. Tabs are expanded from column zero.
	Text string
	// LineBreak is a natural LF, CRLF, or CR consumed after Text.
	LineBreak string
	// Rest is the exact unconsumed suffix of the input source.
	Rest string
	// Width is the display width of Text.
	Width int
	// Overflow reports that the first visible token was wider than width and was
	// consumed whole to guarantee progress.
	Overflow bool
}

// Cut consumes one render-ready fragment of s.
//
// For a positive width, Cut consumes complete grapheme clusters and tabs until
// the next token would exceed width. If the first visible token is wider than
// width, Cut consumes it whole and reports Overflow. A non-positive width is
// unbounded for the first logical line.
//
// A terminating LF, CRLF, or CR is consumed into LineBreak. Rest is always an
// exact suffix of s and starts after that line break, so repeated calls on a
// non-empty input always make progress. Text is render-ready and expands tabs,
// while source consumption is represented only by Rest. Cut does not apply
// TrimTrailingSpace or inject SGR reset/replay sequences. It follows the same
// control-sequence options as [Condition.Wrap], including ControlSequences8Bit.
func (c Condition) Cut(s string, width int) CutResult {
	opts := c.options()
	scanner := c.newDisplayScanner(s, opts)

	var b strings.Builder
	result := CutResult{}
	consumed := 0
	textEnd := 0
	hasTab := false
	for {
		token, ok := scanner.next()
		if !ok {
			break
		}
		if token.lineBreak {
			result.LineBreak = token.text
			consumed += len(token.text)
			break
		}

		wouldOverflow := width > 0 && result.Width+token.width > width
		if wouldOverflow && result.Width > 0 {
			break
		}
		if wouldOverflow {
			result.Overflow = true
		}

		if token.tab {
			if !hasTab {
				// Keep the hint bounded to this cut. Growing for len(s) makes
				// iterative bounded cuts allocate quadratically over the suffixes.
				b.Grow(consumed + token.width)
				b.WriteString(s[:consumed])
				hasTab = true
			}
			writeSpaces(&b, token.width)
		} else if hasTab {
			b.WriteString(token.text)
		}
		result.Width += token.width
		consumed += len(token.text)
		textEnd = consumed
	}

	if hasTab {
		result.Text = b.String()
	} else {
		result.Text = s[:textEnd]
	}
	result.Rest = s[consumed:]
	return result
}

func writeSpaces(b *strings.Builder, count int) {
	for range count {
		b.WriteByte(' ')
	}
}

// WrapLine is one render-ready line returned by [Condition.WrapLines].
type WrapLine struct {
	// Text is the line content without its emitted line break.
	Text string
	// LineBreak is the emitted break after Text. Natural LF, CRLF, and CR are
	// preserved; inserted hard wraps use LF. It is empty on the final line.
	LineBreak string
	// Width is the display width of Text.
	Width int
	// Overflow reports that an overwide token was emitted whole on this line.
	Overflow bool
}

// WrapLines wraps s into render-ready lines with separate first-line and
// continuation widths.
//
// firstWidth applies only to the first output line. Every later output line
// uses restWidth, after either a natural or inserted break. Non-positive lane
// widths are unbounded. Natural LF, CRLF, and CR are preserved in LineBreak;
// inserted hard wraps use LF.
//
// Tabs are expanded, TrimTrailingSpace applies to each line, and SGR state is
// reset before each break and replayed on the next line exactly as in
// [Condition.Wrap]. ControlSequences8Bit is honored. N emitted line breaks
// produce N+1 results, so empty input returns one empty line and a trailing
// natural break returns a trailing empty line.
func (c Condition) WrapLines(s string, firstWidth int, restWidth int) []WrapLine {
	_, lines := c.wrap(s, firstWidth, restWidth, true)
	return lines
}

func (c Condition) wrap(s string, firstWidth int, restWidth int, collectLines bool) (string, []WrapLine) {
	opts := c.options()
	trackSGR := (c.ControlSequences || c.ControlSequences8Bit) && (firstWidth > 0 || restWidth > 0)
	resetSGR := "\x1b[0m"
	if c.ControlSequences8Bit && !c.ControlSequences {
		resetSGR = "\x9b0m"
	}

	var b strings.Builder
	if !collectLines {
		b.Grow(len(s))
	}
	var lineBuilder strings.Builder
	width := firstWidth
	col := 0
	var lines []WrapLine
	if collectLines {
		lines = make([]WrapLine, 0, 1)
	}

	var sgrState sgrStyleState
	writeString := func(text string) {
		if collectLines {
			lineBuilder.WriteString(text)
		} else {
			b.WriteString(text)
		}
	}
	appendLine := func(lineBreak string) {
		if !collectLines {
			return
		}
		text := lineBuilder.String()
		if c.TrimTrailingSpace {
			text = trimTrailingLineSpace(text, opts)
		}
		lineWidth := c.stringWidth(text, opts)
		lines = append(lines, WrapLine{
			Text:      text,
			LineBreak: lineBreak,
			Width:     lineWidth,
			Overflow:  width > 0 && lineWidth > width,
		})
		lineBuilder.Reset()
	}
	emitLineBreak := func(lineBreak string) {
		if trackSGR && sgrState.active() {
			writeString(resetSGR)
		}
		if collectLines {
			appendLine(lineBreak)
		} else {
			b.WriteString(lineBreak)
		}
		if trackSGR {
			if collectLines {
				sgrState.writeTo(&lineBuilder)
			} else {
				sgrState.writeTo(&b)
			}
		}
		width = restWidth
		col = 0
	}

	gs := opts.StringGraphemes(s)
	for gs.Next() {
		value := gs.Value()
		tokenWidth := gs.Width()

		if trackSGR && tokenWidth == 0 && len(value) > 0 && (value[0] == '\x1b' || value[0] == '\x9b') {
			sgrState.apply(value)
			writeString(value)
			continue
		}

		if isLineBreak(value) {
			emitLineBreak(value)
			continue
		}

		if value == "\t" {
			tokenWidth = c.tabWidth() - col%c.tabWidth()
			if width > 0 && col+tokenWidth > width && col > 0 {
				emitLineBreak("\n")
				tokenWidth = c.tabWidth()
			}
			if collectLines {
				writeSpaces(&lineBuilder, tokenWidth)
			} else {
				writeSpaces(&b, tokenWidth)
			}
			col += tokenWidth
			continue
		}

		if width > 0 && col+tokenWidth > width && col > 0 {
			emitLineBreak("\n")
		}
		writeString(value)
		col += tokenWidth
	}

	if collectLines {
		appendLine("")
		return "", lines
	}
	output := b.String()
	if c.TrimTrailingSpace {
		output = trimWrappedLinesRight(output, opts)
	}
	return output, nil
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
func (c Condition) Wrap(s string, width int) string {
	output, _ := c.wrap(s, width, width, false)
	return output
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
	return s == "\n" || s == "\r" || s == "\r\n"
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

type sgrStyleState struct {
	segments []sgrStyleSegment
}

// sgrStyleSegment stores active SGR params from one input sequence. Removed or
// replaced params are deleted so replay stays bounded and reflects terminal state.
type sgrStyleSegment struct {
	prefix string
	params []sgrStyleParam
}

type sgrStyleParam struct {
	key  string
	text string
}

func (s *sgrStyleState) active() bool {
	for _, segment := range s.segments {
		if len(segment.params) > 0 {
			return true
		}
	}
	return false
}

func (s *sgrStyleState) writeTo(b *strings.Builder) {
	for _, segment := range s.segments {
		if len(segment.params) == 0 {
			continue
		}
		b.WriteString(segment.prefix)
		for i, param := range segment.params {
			if i > 0 {
				b.WriteByte(';')
			}
			b.WriteString(param.text)
		}
		b.WriteByte('m')
	}
}

func (s *sgrStyleState) apply(seq string) {
	params, prefix, ok := sgrParamsAndPrefix(seq)
	if !ok || hasPrivateCSIParameter(params) {
		return
	}

	parts := strings.Split(params, ";")
	if len(parts) == 0 {
		parts = []string{""}
	}

	segmentIndex := len(s.segments)
	s.segments = append(s.segments, sgrStyleSegment{prefix: prefix})
	for i := 0; i < len(parts); i++ {
		op := sgrParamOperation(parts, i)
		i += op.extra

		switch op.kind {
		case sgrParamReset:
			s.clear()
			segmentIndex = len(s.segments)
			s.segments = append(s.segments, sgrStyleSegment{prefix: prefix})
		case sgrParamRemove:
			for _, key := range op.keys {
				s.remove(key)
			}
		case sgrParamSet:
			s.remove(op.key)
			s.segments[segmentIndex].params = append(s.segments[segmentIndex].params, sgrStyleParam{
				key:  op.key,
				text: strings.Join(parts[i-op.extra:i+1], ";"),
			})
		}
	}
	s.compact()
}

func (s *sgrStyleState) clear() {
	s.segments = s.segments[:0]
}

func (s *sgrStyleState) remove(key string) {
	for i := range s.segments {
		keep := s.segments[i].params[:0]
		for _, param := range s.segments[i].params {
			if param.key != key {
				keep = append(keep, param)
			}
		}
		s.segments[i].params = keep
	}
}

func (s *sgrStyleState) compact() {
	segments := s.segments[:0]
	for _, segment := range s.segments {
		if len(segment.params) > 0 {
			segments = append(segments, segment)
		}
	}
	s.segments = segments
}

type sgrParamKind int

const (
	sgrParamIgnore sgrParamKind = iota
	sgrParamReset
	sgrParamRemove
	sgrParamSet
)

type sgrParamOp struct {
	kind  sgrParamKind
	key   string
	keys  []string
	extra int
}

func sgrParamOperation(parts []string, i int) sgrParamOp {
	code, ok := sgrParamCode(parts[i])
	if !ok {
		return sgrParamOp{kind: sgrParamSet, key: "unknown"}
	}

	switch {
	case code == 0:
		return sgrParamOp{kind: sgrParamReset}
	case code == 22:
		return sgrParamOp{kind: sgrParamRemove, keys: []string{"bold", "faint"}}
	case code == 23:
		return sgrParamOp{kind: sgrParamRemove, keys: []string{"italic", "fraktur"}}
	case code == 24:
		return sgrParamOp{kind: sgrParamRemove, keys: []string{"underline"}}
	case code == 25:
		return sgrParamOp{kind: sgrParamRemove, keys: []string{"blink", "rapid-blink"}}
	case code == 27:
		return sgrParamOp{kind: sgrParamRemove, keys: []string{"inverse"}}
	case code == 28:
		return sgrParamOp{kind: sgrParamRemove, keys: []string{"conceal"}}
	case code == 29:
		return sgrParamOp{kind: sgrParamRemove, keys: []string{"strike"}}
	case code == 39:
		return sgrParamOp{kind: sgrParamRemove, keys: []string{"fg"}}
	case code == 49:
		return sgrParamOp{kind: sgrParamRemove, keys: []string{"bg"}}
	case code == 54:
		return sgrParamOp{kind: sgrParamRemove, keys: []string{"frame", "encircle"}}
	case code == 55:
		return sgrParamOp{kind: sgrParamRemove, keys: []string{"overline"}}
	case code == 59:
		return sgrParamOp{kind: sgrParamRemove, keys: []string{"underline-color"}}
	case code == 1:
		return sgrParamOp{kind: sgrParamSet, key: "bold"}
	case code == 2:
		return sgrParamOp{kind: sgrParamSet, key: "faint"}
	case code == 3:
		return sgrParamOp{kind: sgrParamSet, key: "italic"}
	case code == 20:
		return sgrParamOp{kind: sgrParamSet, key: "fraktur"}
	case code == 4 || code == 21:
		return sgrParamOp{kind: sgrParamSet, key: "underline"}
	case code == 5:
		return sgrParamOp{kind: sgrParamSet, key: "blink"}
	case code == 6:
		return sgrParamOp{kind: sgrParamSet, key: "rapid-blink"}
	case code == 7:
		return sgrParamOp{kind: sgrParamSet, key: "inverse"}
	case code == 8:
		return sgrParamOp{kind: sgrParamSet, key: "conceal"}
	case code == 9:
		return sgrParamOp{kind: sgrParamSet, key: "strike"}
	case code == 10:
		return sgrParamOp{kind: sgrParamRemove, keys: []string{"font"}}
	case code >= 11 && code <= 19:
		return sgrParamOp{kind: sgrParamSet, key: "font"}
	case code >= 30 && code <= 37 || code >= 90 && code <= 97:
		return sgrParamOp{kind: sgrParamSet, key: "fg"}
	case code >= 40 && code <= 47 || code >= 100 && code <= 107:
		return sgrParamOp{kind: sgrParamSet, key: "bg"}
	case code == 38:
		return sgrExtendedColorOperation(parts, i, "fg")
	case code == 48:
		return sgrExtendedColorOperation(parts, i, "bg")
	case code == 51:
		return sgrParamOp{kind: sgrParamSet, key: "frame"}
	case code == 52:
		return sgrParamOp{kind: sgrParamSet, key: "encircle"}
	case code == 53:
		return sgrParamOp{kind: sgrParamSet, key: "overline"}
	case code == 58:
		return sgrExtendedColorOperation(parts, i, "underline-color")
	default:
		return sgrParamOp{kind: sgrParamSet, key: "unknown"}
	}
}

func sgrExtendedColorOperation(parts []string, i int, key string) sgrParamOp {
	if strings.Contains(parts[i], ":") {
		return sgrParamOp{kind: sgrParamSet, key: key}
	}
	if i+1 >= len(parts) {
		return sgrParamOp{kind: sgrParamSet, key: key}
	}

	mode, ok := sgrParamCode(parts[i+1])
	if !ok {
		return sgrParamOp{kind: sgrParamSet, key: key}
	}

	switch mode {
	case 5:
		extra := len(parts) - i - 1
		if extra > 2 {
			extra = 2
		}
		return sgrParamOp{kind: sgrParamSet, key: key, extra: extra}
	case 2:
		extra := len(parts) - i - 1
		if extra > 4 {
			extra = 4
		}
		return sgrParamOp{kind: sgrParamSet, key: key, extra: extra}
	}
	return sgrParamOp{kind: sgrParamSet, key: key}
}

func sgrParamCode(param string) (int, bool) {
	if before, _, ok := strings.Cut(param, ":"); ok {
		param = before
	}
	if param == "" {
		return 0, true
	}
	code := 0
	for i := 0; i < len(param); i++ {
		if param[i] < '0' || param[i] > '9' {
			return 0, false
		}
		code = code*10 + int(param[i]-'0')
		if code > 10000 {
			return 0, false
		}
	}
	return code, true
}

func sgrParamsAndPrefix(s string) (params string, prefix string, ok bool) {
	params, ok = sgrParams(s)
	if !ok {
		return "", "", false
	}
	if s[0] == '\x9b' {
		return params, "\x9b", true
	}
	return params, "\x1b[", true
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
// relative to the column where tail is appended. When maxWidth <= 0, nothing
// fits and an empty string is returned.
//
// ControlSequences8Bit follows displaywidth and is ignored here, even when it
// is enabled for StringWidth and Wrap. This can make 8-bit C1 sequences count
// as zero-width for measurement but not for truncation; go-tabwrap keeps that
// behavior to avoid mis-parsing UTF-8 byte sequences as standalone C1 controls.
func (c Condition) Truncate(s string, maxWidth int, tail string) string {
	return c.TruncateInfo(s, maxWidth, tail).Text
}

// TruncateInfo truncates s like [Condition.Truncate] and also returns the
// resulting display width and whether truncation occurred.
//
// Multi-line input is truncated line by line so the reported width follows the
// package width model: the widest output line determines Width. When maxWidth
// is non-positive, Text is empty, Width is zero, and Truncated reports whether
// s was non-empty.
func (c Condition) TruncateInfo(s string, maxWidth int, tail string) TruncateResult {
	if maxWidth <= 0 {
		return TruncateResult{Truncated: s != ""}
	}

	opts := c.options()
	opts.ControlSequences8Bit = false

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

func (c Condition) truncateLineInfo(s string, maxWidth int, tail string, opts displaywidth.Options) TruncateResult {
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
func (c Condition) FillLeft(s string, width int) string {
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
func (c Condition) FillRight(s string, width int) string {
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

// Cut consumes one render-ready fragment using default settings.
func Cut(s string, width int) CutResult {
	return defaultCondition.Cut(s, width)
}

// WrapLines wraps s with separate first-line and continuation widths using
// default settings.
func WrapLines(s string, firstWidth int, restWidth int) []WrapLine {
	return defaultCondition.WrapLines(s, firstWidth, restWidth)
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
