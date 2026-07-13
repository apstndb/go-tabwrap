// Package tabwrap provides tab-aware, grapheme-cluster-aware display width
// operations for terminal output.
//
// The package is intended for fixed-width terminal layouts such as tables, tree
// renderers, and status text. Package-level functions use default settings,
// including a tab width of 4. A [Condition] exposes the same operations with
// options for tab width, East Asian width, ECMA-48 control sequences, and
// trailing-space trimming. Use [Condition.Clone] to derive a modified copy of a
// configuration without mutating the original.
//
// # Width Model
//
// Width is measured in terminal display columns per grapheme cluster, not per
// rune. Combining marks, emoji sequences, and other multi-rune graphemes are
// measured through [displaywidth].
//
// Tabs expand to tab stops. With the default tab width of 4, a tab at column 1
// advances to column 4, while a tab at column 4 advances to column 8. A
// [Condition] with TabWidth <= 0 uses the default width of 4. LF, CRLF, and CR
// line breaks reset the current column, and the width of a multi-line string is
// the width of its widest line.
//
// [Condition.EastAsianWidth], [Condition.ControlSequences], and
// [Condition.ControlSequences8Bit] follow displaywidth.Options semantics for
// width measurement and wrapping.
//
// # Wrapping and Truncation
//
// [Wrap] performs hard wrapping by display columns. It does not search for word
// boundaries. If a single grapheme is wider than the requested width, Wrap emits
// that grapheme on its own line, so that line can be wider than the width. Tabs
// are treated as indivisible tokens and are expanded to spaces in the output.
// When control-sequence handling is enabled, Wrap carries recognized SGR state
// across line breaks by resetting before the line break and replaying the active
// SGR sequences after it.
//
// [Cut] consumes one render-ready fragment while preserving the exact unconsumed
// source in [CutResult.Rest]. Tabs may make the rendered text longer than the
// consumed source, so callers must use Rest rather than the byte length of
// [CutResult.Text] to advance input. Natural LF, CRLF, and CR are consumed and
// reported separately in [CutResult.LineBreak]. A positive width never splits a
// grapheme cluster; an overwide first token is emitted whole and reported through
// [CutResult.Overflow]. A non-positive Cut width is unbounded for one logical
// line.
//
// [WrapLines] exposes the same wrapping model as structured lines, with separate
// first-line and continuation widths. Natural line breaks are preserved while
// inserted hard wraps use LF. [Condition.TrimTrailingSpace] applies to each
// returned line. Non-positive wrapping lane widths are unbounded.
//
// [Truncate] is primarily a fitting helper. For a positive maxWidth, it expands
// tabs, truncates the input to fit, and appends the tail when truncation occurs;
// if the tail itself is too wide, the tail is truncated first. When maxWidth <=
// 0, nothing fits and Truncate returns an empty string. Use [TruncateInfo] when
// callers also need the resulting display width or a truncation flag.
//
// Truncate intentionally ignores [Condition.ControlSequences8Bit], even when it
// is enabled for [StringWidth] or [Wrap]. This avoids treating raw C1 bytes
// inside UTF-8 data as standalone control sequences while truncating.
//
// # Conditions and Concurrency
//
// A Condition is a small option value. Read-only operations use value receivers
// and do not mutate the Condition. They are safe to call concurrently as long as
// callers do not mutate the same source Condition while a receiver snapshot is
// being copied. Treat shared Conditions as immutable, or use [Condition.Clone]
// to derive independent configuration.
//
// [displaywidth]: https://github.com/clipperhouse/displaywidth
package tabwrap
