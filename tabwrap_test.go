package tabwrap

import (
	"strings"
	"testing"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := c.FillLeft(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("FillLeft(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
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
	if got := Truncate("hello world", 8, "..."); got != "hello..." {
		t.Errorf("Truncate = %q, want %q", got, "hello...")
	}
	if got := FillLeft("hi", 5); got != "   hi" {
		t.Errorf("FillLeft = %q, want %q", got, "   hi")
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

	t.Run("Wrap with ControlSequences", func(t *testing.T) {
		t.Parallel()
		c := &Condition{TabWidth: 4, ControlSequences: true}
		// "hello" is 5 visible chars, should not wrap at width 5
		got := c.Wrap(styled, 5)
		if strings.Contains(got, "\n") {
			t.Errorf("Wrap(%q, 5) should not wrap, got %q", styled, got)
		}
	})
}
