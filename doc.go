// Package tabwrap provides tab-aware, grapheme-cluster-aware display width
// operations for terminal output.
//
// The package is intended for fixed-width terminal layouts such as tables, tree
// renderers, and status text. Package-level functions use default settings,
// including a tab width of 4. A [Condition] exposes the same operations with
// options for tab width, East Asian width, ECMA-48 control sequences, and
// trailing-space trimming.
//
// # Width Model
//
// Width is measured in terminal display columns per grapheme cluster, not per
// rune. Combining marks, emoji sequences, and other multi-rune graphemes are
// measured through [displaywidth].
//
// Tabs expand to tab stops. With the default tab width of 4, a tab at column 1
// advances to column 4, while a tab at column 4 advances to column 8. A
// [Condition] with TabWidth <= 0 uses the default width of 4. Newlines reset the
// current column, and the width of a multi-line string is the width of its
// widest line.
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
// across line breaks by resetting before the newline and replaying the active
// SGR sequences after it.
//
// [Truncate] is primarily a fitting helper. For a positive maxWidth, it expands
// tabs, truncates the input to fit, and appends the tail when truncation occurs;
// if the tail itself is too wide, the tail is truncated first. When maxWidth <=
// 0, Truncate returns tail as-is.
//
// Truncate intentionally ignores [Condition.ControlSequences8Bit], even when it
// is enabled for [StringWidth] or [Wrap]. This avoids treating raw C1 bytes
// inside UTF-8 data as standalone control sequences while truncating.
//
// # Conditions and Concurrency
//
// A Condition is a small option value. Its methods do not mutate the Condition
// and are safe to call concurrently as long as callers do not mutate the same
// Condition concurrently. Treat shared Conditions as immutable, or use separate
// Condition values for independent configuration.
//
// [displaywidth]: https://github.com/clipperhouse/displaywidth
package tabwrap
