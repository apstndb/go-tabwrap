package tabwrap

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/clipperhouse/displaywidth"
)

func TestStringWidth(t *testing.T) {
	t.Parallel()
	c := NewCondition()

	tests := []struct {
		name string
		s    string
		want int
	}{
		{"empty", "", 0},
		{"ascii", "hello", 5},
		{"tab default", "\t", 4},
		{"tab after 1 char", "a\t", 4},
		{"tab after 2 chars", "ab\t", 4},
		{"tab after 3 chars", "abc\t", 4},
		{"tab after 4 chars", "abcd\t", 8},
		{"two tabs", "\t\t", 8},
		{"CJK", "日本語", 6},
		{"mixed ascii CJK", "a日b", 4},
		{"newline takes max", "abc\nabcdef", 6},
		{"CRLF takes max", "abc\r\nabcdef", 6},
		{"CR takes max", "abc\rabcdef", 6},
		{"only tabs", "\t\t\t", 12},
		{"tab with newline", "ab\t\ncd\t", 4},
		{"tab with CRLF", "ab\t\r\ncd\t", 4},
		{"tab with CR", "ab\t\rcd\t", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := c.StringWidth(tt.s)
			if got != tt.want {
				t.Errorf("StringWidth(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestStringWidthCustomTabWidth(t *testing.T) {
	t.Parallel()
	c := &Condition{TabWidth: 8}

	tests := []struct {
		name string
		s    string
		want int
	}{
		{"tab width 8", "\t", 8},
		{"tab after 1 char", "a\t", 8},
		{"tab after 7 chars", "abcdefg\t", 8},
		{"tab after 8 chars", "abcdefgh\t", 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := c.StringWidth(tt.s)
			if got != tt.want {
				t.Errorf("StringWidth(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestExpandTab(t *testing.T) {
	t.Parallel()
	c := NewCondition()

	tests := []struct {
		name string
		s    string
		want string
	}{
		{"no tabs", "hello", "hello"},
		{"single tab", "\t", "    "},
		{"tab after 1", "a\t", "a   "},
		{"tab after 3", "abc\t", "abc "},
		{"tab after 4", "abcd\t", "abcd    "},
		{"two tabs", "\t\t", "        "},
		{"with newline", "ab\t\ncd\t", "ab  \ncd  "},
		{"with CRLF", "ab\t\r\ncd\t", "ab  \r\ncd  "},
		{"with CR", "ab\t\rcd\t", "ab  \rcd  "},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := c.ExpandTab(tt.s)
			if got != tt.want {
				t.Errorf("ExpandTab(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestEastAsianWidth(t *testing.T) {
	t.Parallel()

	defaultCond := NewCondition()
	eastAsian := &Condition{TabWidth: 4, EastAsianWidth: true}

	if got := defaultCond.StringWidth("○"); got != 1 {
		t.Errorf("default StringWidth(○) = %d, want 1", got)
	}
	if got := eastAsian.StringWidth("○"); got != 2 {
		t.Errorf("EastAsianWidth StringWidth(○) = %d, want 2", got)
	}
	if got := defaultCond.Wrap("○○", 2); got != "○○" {
		t.Errorf("default Wrap ambiguous width = %q, want %q", got, "○○")
	}
	if got := eastAsian.Wrap("○○", 2); got != "○\n○" {
		t.Errorf("EastAsianWidth Wrap ambiguous width = %q, want %q", got, "○\n○")
	}
	if got := eastAsian.Truncate("○○", 3, "."); got != "○." {
		t.Errorf("EastAsianWidth Truncate ambiguous width = %q, want %q", got, "○.")
	}
}

func TestExpandTabFunc(t *testing.T) {
	t.Parallel()
	c := NewCondition()

	t.Run("arrow marker", func(t *testing.T) {
		t.Parallel()
		got := c.ExpandTabFunc("abc\tdef", func(nSpaces int) string {
			return "→" + strings.Repeat(" ", nSpaces-1)
		})
		want := "abc→def"
		if got != want {
			t.Errorf("ExpandTabFunc(%q) = %q, want %q", "abc\tdef", got, want)
		}
	})

	t.Run("identity with spaces", func(t *testing.T) {
		t.Parallel()
		// ExpandTabFunc with spaces should behave identically to ExpandTab
		input := "a\tbc\t\nde\t"
		got := c.ExpandTabFunc(input, func(nSpaces int) string {
			return strings.Repeat(" ", nSpaces)
		})
		want := c.ExpandTab(input)
		if got != want {
			t.Errorf("ExpandTabFunc with spaces = %q, want %q (same as ExpandTab)", got, want)
		}
	})

	t.Run("tab at start", func(t *testing.T) {
		t.Parallel()
		got := c.ExpandTabFunc("\thi", func(nSpaces int) string {
			return "→" + strings.Repeat("·", nSpaces-1)
		})
		want := "→···hi"
		if got != want {
			t.Errorf("ExpandTabFunc(%q) = %q, want %q", "\thi", got, want)
		}
	})

	t.Run("multiple tabs", func(t *testing.T) {
		t.Parallel()
		got := c.ExpandTabFunc("a\tb\t", func(nSpaces int) string {
			if nSpaces < 2 {
				return strings.Repeat(".", nSpaces)
			}
			return "[" + strings.Repeat("-", nSpaces-2) + "]"
		})
		// "a" at col 1, tab nSpaces=3: "[-]"
		// col advances to 4, "b" at col 5, tab nSpaces=3: "[-]"
		want := "a[-]b[-]"
		if got != want {
			t.Errorf("ExpandTabFunc = %q, want %q", got, want)
		}
	})

	t.Run("with newline", func(t *testing.T) {
		t.Parallel()
		got := c.ExpandTabFunc("ab\t\ncd\t", func(nSpaces int) string {
			return "→" + strings.Repeat(" ", nSpaces-1)
		})
		want := "ab→ \ncd→ "
		if got != want {
			t.Errorf("ExpandTabFunc = %q, want %q", got, want)
		}
	})

	t.Run("with CRLF and CR", func(t *testing.T) {
		t.Parallel()
		got := c.ExpandTabFunc("ab\t\r\ncd\t\ref\t", func(nSpaces int) string {
			return "→" + strings.Repeat(" ", nSpaces-1)
		})
		want := "ab→ \r\ncd→ \ref→ "
		if got != want {
			t.Errorf("ExpandTabFunc = %q, want %q", got, want)
		}
	})

	assertPanics := func(t *testing.T, name string, fn func()) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("%s: did not panic", name)
				}
			}()
			fn()
		})
	}

	assertPanics(t, "nil func panics with tab", func() {
		c.ExpandTabFunc("a\tb", nil)
	})

	t.Run("nil func without tab does not panic", func(t *testing.T) {
		t.Parallel()
		if got := c.ExpandTabFunc("abc", nil); got != "abc" {
			t.Errorf("ExpandTabFunc without tabs = %q, want %q", got, "abc")
		}
	})

	t.Run("package-level nil func without tab does not panic", func(t *testing.T) {
		t.Parallel()
		if got := ExpandTabFunc("abc", nil); got != "abc" {
			t.Errorf("ExpandTabFunc without tabs = %q, want %q", got, "abc")
		}
	})
}

func TestWrap(t *testing.T) {
	t.Parallel()
	c := NewCondition()

	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{"no wrap needed", "hello", 10, "hello"},
		{"exact fit", "hello", 5, "hello"},
		{"wrap mid-word", "helloworld", 5, "hello\nworld"},
		{"wrap with spaces", "hello world", 5, "hello\n worl\nd"},
		{"empty string", "", 10, ""},
		{"width zero", "hello", 0, "hello"},
		{"tab no wrap", "\t", 10, "    "},
		{"tab fits exactly at 4", "\t", 4, "    "},
		{"tab fits exactly after abc", "abc\t", 4, "abc "},
		{"tab wraps to next line", "abcd\t", 4, "abcd\n    "},
		{"CJK wrap", "日本語", 4, "日本\n語"},
		{"newline preserved", "ab\ncd", 10, "ab\ncd"},
		{"CRLF preserved", "ab\r\ncd", 10, "ab\r\ncd"},
		{"CR preserved", "ab\rcd", 10, "ab\rcd"},
		{"tab with newline wrap", "ab\t\ncd", 10, "ab  \ncd"},
		{"tab with CRLF wrap", "ab\t\r\ncd", 10, "ab  \r\ncd"},
		{"tab with CR wrap", "ab\t\rcd", 10, "ab  \rcd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := c.Wrap(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("Wrap(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
			}
		})
	}
}

func TestWrapTrimTrailingSpace(t *testing.T) {
	t.Parallel()

	t.Run("trimmed plain output", func(t *testing.T) {
		t.Parallel()
		c := &Condition{TabWidth: 4, TrimTrailingSpace: true}

		tests := []struct {
			name  string
			s     string
			width int
			want  string
		}{
			{"tab before wrap boundary", "ab\tcd", 4, "ab\ncd"},
			{"tab at end of line", "abc\t", 4, "abc"},
			{"natural newline", "ab\t\ncd\t", 10, "ab\ncd"},
			{"natural CRLF", "ab\t\r\ncd\t", 10, "ab\r\ncd"},
			{"natural CR", "ab\t\rcd\t", 10, "ab\rcd"},
			{"width zero still trims", "abc\t", 0, "abc"},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := c.Wrap(tt.s, tt.width)
				if got != tt.want {
					t.Errorf("Wrap(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
				}
			})
		}
	})

	t.Run("preserves trailing control sequences", func(t *testing.T) {
		t.Parallel()
		c := &Condition{TabWidth: 4, ControlSequences: true, TrimTrailingSpace: true}
		red := "\x1b[31m"
		reset := "\x1b[0m"

		got := c.Wrap(red+"ab\tcd"+reset, 4)
		want := red + "ab" + reset + "\n" + red + "cd" + reset
		if got != want {
			t.Errorf("Wrap styled trim = %q, want %q", got, want)
		}
	})

	t.Run("trims spaces around interleaved zero-width sequences", func(t *testing.T) {
		t.Parallel()
		opts := displaywidth.Options{ControlSequences: true}
		input := "ab \x1b[0m \x1b[31m"
		got := trimTrailingLineSpace(input, opts)
		want := "ab\x1b[0m\x1b[31m"
		if got != want {
			t.Errorf("trimTrailingLineSpace(%q) = %q, want %q", input, got, want)
		}
	})
}

func TestTruncate(t *testing.T) {
	t.Parallel()
	c := NewCondition()

	tests := []struct {
		name     string
		s        string
		maxWidth int
		tail     string
		want     string
	}{
		{"no truncation", "hello", 10, "...", "hello"},
		{"exact fit", "hello", 5, "...", "hello"},
		{"truncate with tail", "hello world", 8, "...", "hello..."},
		{"truncate clamps wide tail", "hello", 1, "...", "."},
		{"empty string", "", 5, "...", ""},
		{"CJK truncate", "日本語テスト", 7, "...", "日本..."},
		{"tab in string fits", "a\tb", 5, "...", "a   b"},
		{"tab in string truncated", "a\tbc", 5, "...", "a ..."},
		{"tail tab expands independently", "hello world", 8, "\tX", "hel    X"},
		{"width zero", "hello", 0, "...", "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := c.Truncate(tt.s, tt.maxWidth, tt.tail)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d, %q) = %q, want %q", tt.s, tt.maxWidth, tt.tail, got, tt.want)
			}
			if tt.maxWidth > 0 && c.StringWidth(got) > tt.maxWidth {
				t.Errorf("Truncate(%q, %d, %q) visible width = %d, want <= %d", tt.s, tt.maxWidth, tt.tail, c.StringWidth(got), tt.maxWidth)
			}
		})
	}
}

func TestTruncateInfo(t *testing.T) {
	t.Parallel()
	c := NewCondition()

	tests := []struct {
		name string
		s    string
		max  int
		tail string
		want TruncateResult
	}{
		{
			name: "single line no truncation expands tabs",
			s:    "a\tb",
			max:  5,
			tail: "...",
			want: TruncateResult{Text: "a   b", Width: 5, Truncated: false},
		},
		{
			name: "single line truncation",
			s:    "hello world",
			max:  8,
			tail: "...",
			want: TruncateResult{Text: "hello...", Width: 8, Truncated: true},
		},
		{
			name: "non-positive width preserves tail bytes",
			s:    "hello",
			max:  0,
			tail: "\t",
			want: TruncateResult{Text: "\t", Width: 4, Truncated: true},
		},
		{
			name: "non-positive width returns one tail for multi-line input",
			s:    "hello\nworld",
			max:  0,
			tail: "...",
			want: TruncateResult{Text: "...", Width: 3, Truncated: true},
		},
		{
			name: "multi-line widest line fits",
			s:    "abc\ndef",
			max:  3,
			tail: "...",
			want: TruncateResult{Text: "abc\ndef", Width: 3, Truncated: false},
		},
		{
			name: "multi-line truncates each line independently",
			s:    "abcdef\r\nefghij\rkl",
			max:  5,
			tail: "...",
			want: TruncateResult{Text: "ab...\r\nef...\rkl", Width: 5, Truncated: true},
		},
		{
			name: "multi-line expands tabs per line",
			s:    "a\tb\nabcdef",
			max:  5,
			tail: ".",
			want: TruncateResult{Text: "a   b\nabcd.", Width: 5, Truncated: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := c.TruncateInfo(tt.s, tt.max, tt.tail)
			if got != tt.want {
				t.Errorf("TruncateInfo(%q, %d, %q) = %+v, want %+v", tt.s, tt.max, tt.tail, got, tt.want)
			}
			if text := c.Truncate(tt.s, tt.max, tt.tail); text != got.Text {
				t.Errorf("Truncate(%q, %d, %q) = %q, want TruncateInfo.Text %q", tt.s, tt.max, tt.tail, text, got.Text)
			}
		})
	}
}

func TestFillLeft(t *testing.T) {
	t.Parallel()
	c := NewCondition()

	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{"pad needed", "hi", 5, "   hi"},
		{"exact width", "hello", 5, "hello"},
		{"wider than width", "hello world", 5, "hello world"},
		{"empty string", "", 3, "   "},
		{"CJK", "日本", 6, "  日本"},
		{"first line not widest", "a\nbc", 3, "  a\nbc"},
		{"first line not widest with CRLF", "a\r\nbc", 3, "  a\r\nbc"},
		{"first line not widest with CR", "a\rbc", 3, "  a\rbc"},
		{"tab expands before left padding", "a\tb", 8, "   a   b"},
		{"tab exact width unchanged", "a\tb", 5, "a\tb"},
		{"only first line tabs expand", "a\tb\nc\td", 8, "   a   b\nc\td"},
		{"only first line tabs expand with CR", "a\tb\rc\td", 8, "   a   b\rc\td"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := c.FillLeft(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("FillLeft(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
			}
			if c.StringWidth(got) != max(c.StringWidth(tt.want), tt.width) {
				t.Errorf("FillLeft(%q, %d) visible width = %d, want %d", tt.s, tt.width, c.StringWidth(got), max(c.StringWidth(tt.want), tt.width))
			}
		})
	}
}

func TestFillMultiLineContracts(t *testing.T) {
	t.Parallel()
	c := NewCondition()

	if got, want := c.FillLeft("a\nbc", 3), "  a\nbc"; got != want {
		t.Errorf("FillLeft multi-line = %q, want %q", got, want)
	}
	if got, want := c.FillRight("a\nbc", 3), "a\nbc "; got != want {
		t.Errorf("FillRight multi-line = %q, want %q", got, want)
	}
}

func TestFillRight(t *testing.T) {
	t.Parallel()
	c := NewCondition()

	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{"pad needed", "hi", 5, "hi   "},
		{"exact width", "hello", 5, "hello"},
		{"wider than width", "hello world", 5, "hello world"},
		{"empty string", "", 3, "   "},
		{"CJK", "日本", 6, "日本  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := c.FillRight(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("FillRight(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
			}
		})
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	t.Parallel()

	if got := StringWidth("hello"); got != 5 {
		t.Errorf("StringWidth = %d, want 5", got)
	}
	if got := ExpandTab("a\tb"); got != "a   b" {
		t.Errorf("ExpandTab = %q, want %q", got, "a   b")
	}
	if got := ExpandTabFunc("abc\td", func(n int) string {
		return "→" + strings.Repeat(" ", n-1)
	}); got != "abc→d" {
		t.Errorf("ExpandTabFunc = %q, want %q", got, "abc→d")
	}
	if got := Wrap("helloworld", 5); got != "hello\nworld" {
		t.Errorf("Wrap = %q, want %q", got, "hello\nworld")
	}
	if got := Truncate("hello world", 8, "..."); got != "hello..." {
		t.Errorf("Truncate = %q, want %q", got, "hello...")
	}
	if got := Truncate("hello", 1, "..."); got != "." {
		t.Errorf("Truncate wide tail = %q, want %q", got, ".")
	}
	if got := TruncateInfo("hello world", 8, "..."); got != (TruncateResult{Text: "hello...", Width: 8, Truncated: true}) {
		t.Errorf("TruncateInfo = %+v, want hello... width 8 truncated", got)
	}
	if got := FillLeft("hi", 5); got != "   hi" {
		t.Errorf("FillLeft = %q, want %q", got, "   hi")
	}
	if got := FillLeft("a\tb", 8); got != "   a   b" {
		t.Errorf("FillLeft tab = %q, want %q", got, "   a   b")
	}
	if got := FillRight("hi", 5); got != "hi   " {
		t.Errorf("FillRight = %q, want %q", got, "hi   ")
	}
}

func TestConditionZeroTabWidth(t *testing.T) {
	t.Parallel()
	c := &Condition{TabWidth: 0}
	if got := c.StringWidth("\t"); got != 4 {
		t.Errorf("zero TabWidth: StringWidth(tab) = %d, want 4", got)
	}
}

func TestConditionClone(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver returns defaults", func(t *testing.T) {
		t.Parallel()
		var c *Condition
		got := c.Clone()
		if got == nil {
			t.Fatal("Clone() returned nil")
		}
		if got.TabWidth != 4 {
			t.Errorf("Clone().TabWidth = %d, want 4", got.TabWidth)
		}
	})

	t.Run("copies options and normalizes tab width", func(t *testing.T) {
		t.Parallel()
		c := &Condition{
			TabWidth:             0,
			EastAsianWidth:       true,
			ControlSequences:     true,
			ControlSequences8Bit: true,
			TrimTrailingSpace:    true,
		}

		got := c.Clone()
		if got == c {
			t.Fatal("Clone() returned the original pointer")
		}
		if got.TabWidth != 4 {
			t.Errorf("Clone().TabWidth = %d, want 4", got.TabWidth)
		}
		if !got.EastAsianWidth || !got.ControlSequences || !got.ControlSequences8Bit || !got.TrimTrailingSpace {
			t.Errorf("Clone() did not preserve options: %+v", got)
		}

		got.TabWidth = 8
		if c.TabWidth != 0 {
			t.Errorf("mutating clone changed original TabWidth to %d, want 0", c.TabWidth)
		}
	})
}

func TestControlSequences(t *testing.T) {
	t.Parallel()

	red := "\x1b[31m"
	reset := "\x1b[0m"
	styled := red + "hello" + reset

	t.Run("without ControlSequences", func(t *testing.T) {
		t.Parallel()
		c := NewCondition()
		// Without ControlSequences, escape bytes contribute to width
		got := c.StringWidth(styled)
		if got <= 5 {
			t.Errorf("expected width > 5 without ControlSequences, got %d", got)
		}
	})

	t.Run("with ControlSequences", func(t *testing.T) {
		t.Parallel()
		c := &Condition{TabWidth: 4, ControlSequences: true}
		got := c.StringWidth(styled)
		if got != 5 {
			t.Errorf("StringWidth(%q) with ControlSequences = %d, want 5", styled, got)
		}
	})

	t.Run("Truncate with ControlSequences", func(t *testing.T) {
		t.Parallel()
		c := &Condition{TabWidth: 4, ControlSequences: true}
		got := c.Truncate(red+"hello world"+reset, 8, "...")
		// Should truncate based on visible width (5 visible + "...")
		if c.StringWidth(got) > 8 {
			t.Errorf("Truncate visible width = %d, want <= 8", c.StringWidth(got))
		}
	})

	t.Run("FillRight with ControlSequences", func(t *testing.T) {
		t.Parallel()
		c := &Condition{TabWidth: 4, ControlSequences: true}
		got := c.FillRight(styled, 10)
		if c.StringWidth(got) != 10 {
			t.Errorf("FillRight visible width = %d, want 10", c.StringWidth(got))
		}
	})

	t.Run("Wrap with ControlSequences no wrap", func(t *testing.T) {
		t.Parallel()
		c := &Condition{TabWidth: 4, ControlSequences: true}
		// "hello" is 5 visible chars, should not wrap at width 5
		got := c.Wrap(styled, 5)
		if strings.Contains(got, "\n") {
			t.Errorf("Wrap(%q, 5) should not wrap, got %q", styled, got)
		}
	})
}

func TestControlSequences8Bit(t *testing.T) {
	t.Parallel()
	// 8-bit CSI: 0x9b is the 8-bit equivalent of ESC [
	csi8 := "\x9b31m"  // 8-bit CSI SGR red
	reset8 := "\x9b0m" // 8-bit CSI SGR reset
	styled := csi8 + "hello" + reset8

	t.Run("without ControlSequences8Bit", func(t *testing.T) {
		t.Parallel()
		c := &Condition{TabWidth: 4, ControlSequences: true}
		got := c.StringWidth(styled)
		if got <= 5 {
			t.Errorf("expected width > 5 without ControlSequences8Bit, got %d", got)
		}
	})

	t.Run("with ControlSequences8Bit", func(t *testing.T) {
		t.Parallel()
		c := &Condition{TabWidth: 4, ControlSequences: true, ControlSequences8Bit: true}
		got := c.StringWidth(styled)
		if got != 5 {
			t.Errorf("StringWidth with ControlSequences8Bit = %d, want 5", got)
		}
	})

	t.Run("with ControlSequences8Bit only", func(t *testing.T) {
		t.Parallel()
		c := &Condition{TabWidth: 4, ControlSequences8Bit: true}
		got := c.StringWidth(styled)
		if got != 5 {
			t.Errorf("StringWidth with ControlSequences8Bit only = %d, want 5", got)
		}
	})

	t.Run("Truncate ignores ControlSequences8Bit", func(t *testing.T) {
		t.Parallel()
		s := csi8 + "hello world" + reset8
		defaultCond := NewCondition()
		c := &Condition{TabWidth: 4, ControlSequences8Bit: true}

		got := c.Truncate(s, 8, "...")
		want := defaultCond.Truncate(s, 8, "...")
		if got != want {
			t.Errorf("Truncate with ControlSequences8Bit = %q, want %q", got, want)
		}
	})

	t.Run("Truncate ignores ControlSequences8Bit with tabs", func(t *testing.T) {
		t.Parallel()
		s := csi8 + "a\tbc" + reset8
		defaultCond := NewCondition()
		c := &Condition{TabWidth: 4, ControlSequences8Bit: true}

		got := c.Truncate(s, 5, "...")
		want := defaultCond.Truncate(s, 5, "...")
		if got != want {
			t.Errorf("Truncate with ControlSequences8Bit and tabs = %q, want %q", got, want)
		}
	})
}

func TestWrapSGRCarryOver8Bit(t *testing.T) {
	t.Parallel()
	red8 := "\x9b31m"
	reset8 := "\x9b0m"

	reset7 := "\x1b[0m"

	tests := []struct {
		name     string
		c        *Condition
		resetMid string
	}{
		{
			name:     "with ControlSequences8Bit only",
			c:        &Condition{TabWidth: 4, ControlSequences8Bit: true},
			resetMid: reset8,
		},
		{
			name:     "with ControlSequences and ControlSequences8Bit",
			c:        &Condition{TabWidth: 4, ControlSequences: true, ControlSequences8Bit: true},
			resetMid: reset7,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			t.Run("single color wrap", func(t *testing.T) {
				t.Parallel()
				got := tt.c.Wrap(red8+"helloworld"+reset8, 5)
				want := red8 + "hello" + tt.resetMid + "\n" + red8 + "world" + reset8
				if got != want {
					t.Errorf("Wrap 8-bit:\n got  %q\n want %q", got, want)
				}
			})

			t.Run("line independence", func(t *testing.T) {
				t.Parallel()
				input := red8 + "hello world test" + reset8
				got := tt.c.Wrap(input, 5)
				lines := strings.Split(got, "\n")

				for i, line := range lines {
					if !strings.HasPrefix(line, red8) {
						t.Errorf("line %d %q: does not start with 8-bit red sequence", i, line)
					}
					// Reset may be 7-bit (emitNewline) or 8-bit (from input)
					if !strings.Contains(line, reset7) && !strings.Contains(line, reset8) {
						t.Errorf("line %d %q: does not contain any reset sequence", i, line)
					}
					w := tt.c.StringWidth(line)
					if w > 5 {
						t.Errorf("line %d visible width = %d, want <= 5", i, w)
					}
				}
			})
		})
	}
}

func TestWrapSGRCarryOver(t *testing.T) {
	t.Parallel()
	c := &Condition{TabWidth: 4, ControlSequences: true}

	red := "\x1b[31m"
	bold := "\x1b[1m"
	dim := "\x1b[2m"
	reset := "\x1b[0m"

	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{
			name:  "single color wrap",
			s:     red + "helloworld" + reset,
			width: 5,
			// At wrap break: emit reset, newline, replay red
			want: red + "hello" + reset + "\n" + red + "world" + reset,
		},
		{
			name:  "no wrap needed",
			s:     red + "hello" + reset,
			width: 10,
			want:  red + "hello" + reset,
		},
		{
			name:  "multiple SGR sequences",
			s:     bold + red + "helloworld" + reset,
			width: 5,
			want:  bold + red + "hello" + reset + "\n" + bold + red + "world" + reset,
		},
		{
			name:  "reset mid-text clears state",
			s:     red + "he" + reset + "lloworld",
			width: 5,
			// After reset, no SGR state to carry over
			want: red + "he" + reset + "llo\nworld",
		},
		{
			name:  "natural newline carries state",
			s:     red + "ab\ncd" + reset,
			width: 10,
			want:  red + "ab" + reset + "\n" + red + "cd" + reset,
		},
		{
			name:  "natural CRLF carries state",
			s:     red + "ab\r\ncd" + reset,
			width: 10,
			want:  red + "ab" + reset + "\r\n" + red + "cd" + reset,
		},
		{
			name:  "natural CR carries state",
			s:     red + "ab\rcd" + reset,
			width: 10,
			want:  red + "ab" + reset + "\r" + red + "cd" + reset,
		},
		{
			name:  "dim NULL wrap",
			s:     dim + "NULL value here" + reset,
			width: 10,
			want:  dim + "NULL value" + reset + "\n" + dim + " here" + reset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := c.Wrap(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("Wrap(%q, %d):\n got  %q\n want %q", tt.s, tt.width, got, tt.want)
			}
		})
	}
}

func TestWrapSGRTrackingTightening(t *testing.T) {
	t.Parallel()
	c := &Condition{TabWidth: 4, ControlSequences: true}

	red := "\x1b[31m"
	bold := "\x1b[1m"
	reset := "\x1b[0m"
	private := "\x1b[?25m"
	resetRed := "\x1b[0;31m"
	zeroResetRed := "\x1b[00;31m"
	zeroReset := "\x1b[00m"
	green := "\x1b[32m"
	blue := "\x1b[34m"
	boldOff := "\x1b[22m"
	fgOff := "\x1b[39m"
	bgGreen := "\x1b[42m"
	bgOff := "\x1b[49m"
	boldRed := "\x1b[1;31m"
	boldRedOff := "\x1b[22;39m"
	redGreenBGFGOff := "\x1b[31;42;39m"
	fg256 := "\x1b[38;5;196m"
	fgRGB := "\x1b[38;2;1;2;3m"
	bgRGB := "\x1b[48;2;4;5;6m"
	malformedFG256 := "\x1b[38;5m"
	malformedFGRGB := "\x1b[38;2;1;2m"
	malformedBG256 := "\x1b[48;5m"
	malformedUnderline256 := "\x1b[58;5m"
	underlineColorOff := "\x1b[59m"
	fraktur := "\x1b[20m"
	italicFrakturOff := "\x1b[23m"
	unknown60 := "\x1b[60m"
	unknown61 := "\x1b[61m"

	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{
			name:  "private CSI m is not carried",
			s:     red + "ab" + private + "cdef" + reset,
			width: 3,
			want:  red + "ab" + private + "c" + reset + "\n" + red + "def" + reset,
		},
		{
			name:  "leading reset composite drops previous state",
			s:     bold + resetRed + "abcdef" + reset,
			width: 3,
			want:  bold + resetRed + "abc" + reset + "\n" + red + "def" + reset,
		},
		{
			name:  "all-zero leading reset composite drops previous state",
			s:     bold + zeroResetRed + "abcdef" + reset,
			width: 3,
			want:  bold + zeroResetRed + "abc" + reset + "\n" + red + "def" + reset,
		},
		{
			name:  "all-zero reset clears state",
			s:     red + "ab" + zeroReset + "cdef",
			width: 3,
			want:  red + "ab" + zeroReset + "c\n" + "def",
		},
		{
			name:  "repeated foreground changes replay only latest color",
			s:     red + green + "abc" + blue + "def" + reset,
			width: 3,
			want:  red + green + "abc" + blue + reset + "\n" + blue + "def" + reset,
		},
		{
			name:  "intensity off removes bold and keeps foreground",
			s:     bold + red + "ab" + boldOff + "cdef" + reset,
			width: 3,
			want:  bold + red + "ab" + boldOff + "c" + reset + "\n" + red + "def" + reset,
		},
		{
			name:  "foreground off removes color and keeps bold",
			s:     bold + red + "ab" + fgOff + "cdef" + reset,
			width: 3,
			want:  bold + red + "ab" + fgOff + "c" + reset + "\n" + bold + "def" + reset,
		},
		{
			name:  "background off removes background and keeps foreground",
			s:     red + bgGreen + "ab" + bgOff + "cdef" + reset,
			width: 3,
			want:  red + bgGreen + "ab" + bgOff + "c" + reset + "\n" + red + "def" + reset,
		},
		{
			name:  "composite off removes bold and foreground",
			s:     boldRed + "ab" + boldRedOff + "cdef",
			width: 3,
			want:  boldRed + "ab" + boldRedOff + "c\n" + "def",
		},
		{
			name:  "composite foreground off keeps background",
			s:     redGreenBGFGOff + "abcdef" + reset,
			width: 3,
			want:  redGreenBGFGOff + "abc" + reset + "\n" + bgGreen + "def" + reset,
		},
		{
			name:  "extended foreground replaces previous foreground",
			s:     fg256 + "abc" + fgRGB + "def" + reset,
			width: 3,
			want:  fg256 + "abc" + fgRGB + reset + "\n" + fgRGB + "def" + reset,
		},
		{
			name:  "foreground off removes extended color and keeps background",
			s:     fgRGB + bgRGB + "ab" + fgOff + "cdef" + reset,
			width: 3,
			want:  fgRGB + bgRGB + "ab" + fgOff + "c" + reset + "\n" + bgRGB + "def" + reset,
		},
		{
			name:  "malformed 256 foreground is consumed before foreground off",
			s:     malformedFG256 + fgOff + "abcdef",
			width: 3,
			want:  malformedFG256 + fgOff + "abc\n" + "def",
		},
		{
			name:  "malformed RGB foreground is consumed before foreground off",
			s:     malformedFGRGB + fgOff + "abcdef",
			width: 3,
			want:  malformedFGRGB + fgOff + "abc\n" + "def",
		},
		{
			name:  "malformed 256 background is consumed before background off",
			s:     malformedBG256 + bgOff + "abcdef",
			width: 3,
			want:  malformedBG256 + bgOff + "abc\n" + "def",
		},
		{
			name:  "malformed underline color is consumed before underline color off",
			s:     malformedUnderline256 + underlineColorOff + "abcdef",
			width: 3,
			want:  malformedUnderline256 + underlineColorOff + "abc\n" + "def",
		},
		{
			name:  "fraktur off removes fraktur",
			s:     fraktur + "ab" + italicFrakturOff + "cdef",
			width: 3,
			want:  fraktur + "ab" + italicFrakturOff + "c\n" + "def",
		},
		{
			name:  "unknown SGR replay remains bounded",
			s:     unknown60 + "abc" + unknown61 + "def" + reset,
			width: 3,
			want:  unknown60 + "abc" + unknown61 + reset + "\n" + unknown61 + "def" + reset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := c.Wrap(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("Wrap(%q, %d):\n got  %q\n want %q", tt.s, tt.width, got, tt.want)
			}
		})
	}
}

func TestWrapSGRTrackingTightening8Bit(t *testing.T) {
	t.Parallel()

	reset7 := "\x1b[0m"
	blue7 := "\x1b[34m"

	red8 := "\x9b31m"
	bold8 := "\x9b1m"
	resetRed8 := "\x9b0;31m"
	green8 := "\x9b32m"
	blue8 := "\x9b34m"
	reset8 := "\x9b0m"
	boldOff8 := "\x9b22m"
	fgOff8 := "\x9b39m"
	fgRGB8 := "\x9b38;2;1;2;3m"
	bgRGB8 := "\x9b48;2;4;5;6m"
	malformedFG2568 := "\x9b38;5m"

	tests := []struct {
		name  string
		c     *Condition
		s     string
		width int
		want  string
	}{
		{
			name:  "leading reset composite drops previous 8-bit state",
			c:     &Condition{TabWidth: 4, ControlSequences8Bit: true},
			s:     bold8 + resetRed8 + "abcdef" + reset8,
			width: 3,
			want:  bold8 + resetRed8 + "abc" + reset8 + "\n" + red8 + "def" + reset8,
		},
		{
			name:  "repeated 8-bit foreground changes replay only latest color",
			c:     &Condition{TabWidth: 4, ControlSequences8Bit: true},
			s:     red8 + green8 + "abc" + blue8 + "def" + reset8,
			width: 3,
			want:  red8 + green8 + "abc" + blue8 + reset8 + "\n" + blue8 + "def" + reset8,
		},
		{
			name:  "8-bit foreground off removes color and keeps bold",
			c:     &Condition{TabWidth: 4, ControlSequences8Bit: true},
			s:     bold8 + red8 + "ab" + fgOff8 + "cdef" + reset8,
			width: 3,
			want:  bold8 + red8 + "ab" + fgOff8 + "c" + reset8 + "\n" + bold8 + "def" + reset8,
		},
		{
			name:  "8-bit intensity off removes bold and keeps foreground",
			c:     &Condition{TabWidth: 4, ControlSequences8Bit: true},
			s:     bold8 + red8 + "ab" + boldOff8 + "cdef" + reset8,
			width: 3,
			want:  bold8 + red8 + "ab" + boldOff8 + "c" + reset8 + "\n" + red8 + "def" + reset8,
		},
		{
			name:  "8-bit extended foreground off keeps background",
			c:     &Condition{TabWidth: 4, ControlSequences8Bit: true},
			s:     fgRGB8 + bgRGB8 + "ab" + fgOff8 + "cdef" + reset8,
			width: 3,
			want:  fgRGB8 + bgRGB8 + "ab" + fgOff8 + "c" + reset8 + "\n" + bgRGB8 + "def" + reset8,
		},
		{
			name:  "malformed 8-bit foreground is consumed before foreground off",
			c:     &Condition{TabWidth: 4, ControlSequences8Bit: true},
			s:     malformedFG2568 + fgOff8 + "abcdef",
			width: 3,
			want:  malformedFG2568 + fgOff8 + "abc\n" + "def",
		},
		{
			name:  "mixed 8-bit and 7-bit foreground replays latest prefix",
			c:     &Condition{TabWidth: 4, ControlSequences: true, ControlSequences8Bit: true},
			s:     red8 + "abc" + blue7 + "def" + reset7,
			width: 3,
			want:  red8 + "abc" + blue7 + reset7 + "\n" + blue7 + "def" + reset7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.c.Wrap(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("Wrap(%q, %d):\n got  %q\n want %q", tt.s, tt.width, got, tt.want)
			}
		})
	}
}

func TestWrapWithoutControlSequences(t *testing.T) {
	t.Parallel()
	// When ControlSequences is false, escape bytes are visible chars
	// and wrapping happens differently — just verify no panic.
	c := &Condition{TabWidth: 4, ControlSequences: false}
	red := "\x1b[31m"
	reset := "\x1b[0m"
	_ = c.Wrap(red+"helloworld"+reset, 5)
}

func TestWrapSGRCarryOverLineIndependence(t *testing.T) {
	t.Parallel()
	c := &Condition{TabWidth: 4, ControlSequences: true}

	dim := "\x1b[2m"
	reset := "\x1b[0m"

	// Verify each line is independently styled
	input := dim + "hello world test" + reset
	got := c.Wrap(input, 5)
	lines := strings.Split(got, "\n")

	for i, line := range lines {
		// Each line should start with dim (if non-empty visible content)
		if !strings.HasPrefix(line, dim) {
			t.Errorf("line %d %q: does not start with dim sequence", i, line)
		}
		// Each line (except possibly the last if it ends with reset from input)
		// should contain a reset
		if !strings.Contains(line, reset) {
			t.Errorf("line %d %q: does not contain reset sequence", i, line)
		}
	}

	// Verify visible width of each line is correct
	for i, line := range lines {
		w := c.StringWidth(line)
		if w > 5 {
			t.Errorf("line %d visible width = %d, want <= 5", i, w)
		}
	}
}

func FuzzExpandTabPreservesWidth(f *testing.F) {
	for _, seed := range []string{
		"",
		"a\tb",
		"ab\t\r\ncd\t",
		"日本\t語",
		"e\u0301\tz",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, s string) {
		if !utf8.ValidString(s) {
			t.Skip()
		}
		c := NewCondition()
		if got, want := c.StringWidth(c.ExpandTab(s)), c.StringWidth(s); got != want {
			t.Fatalf("StringWidth(ExpandTab(%q)) = %d, want %d", s, got, want)
		}
	})
}

func FuzzTruncateSingleLineWidth(f *testing.F) {
	f.Add("hello world", "...", 8)
	f.Add("a\tbc", "...", 5)
	f.Add("日本語テスト", "...", 7)

	f.Fuzz(func(t *testing.T, s, tail string, maxWidth int) {
		if !utf8.ValidString(s) || !utf8.ValidString(tail) {
			t.Skip()
		}
		if strings.ContainsAny(s, "\r\n") || strings.ContainsAny(tail, "\r\n") {
			t.Skip()
		}
		maxWidth %= 40
		if maxWidth < 0 {
			maxWidth = -maxWidth
		}
		maxWidth++

		c := NewCondition()
		got := c.Truncate(s, maxWidth, tail)
		if width := c.StringWidth(got); width > maxWidth {
			t.Fatalf("StringWidth(Truncate(%q, %d, %q)) = %d, want <= %d; got %q", s, maxWidth, tail, width, maxWidth, got)
		}
	})
}
