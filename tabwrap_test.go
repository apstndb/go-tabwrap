package tabwrap

import (
	"strings"
	"testing"

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
		{"only tabs", "\t\t\t", 12},
		{"tab with newline", "ab\t\ncd\t", 4},
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
		{"tab with newline wrap", "ab\t\ncd", 10, "ab  \ncd"},
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
		{"tab expands before left padding", "a\tb", 8, "   a   b"},
		{"tab exact width unchanged", "a\tb", 5, "a\tb"},
		{"only first line tabs expand", "a\tb\nc\td", 8, "   a   b\nc\td"},
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
