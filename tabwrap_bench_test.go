package tabwrap

import (
	"strings"
	"testing"
)

func BenchmarkStringWidth(b *testing.B) {
	c := NewCondition()

	b.Run("ascii", func(b *testing.B) {
		for range b.N {
			c.StringWidth("hello world, this is a test string")
		}
	})

	b.Run("CJK", func(b *testing.B) {
		for range b.N {
			c.StringWidth("日本語のテスト文字列です")
		}
	})

	b.Run("tabs", func(b *testing.B) {
		for range b.N {
			c.StringWidth("col1\tcol2\tcol3\tcol4")
		}
	})
}

func BenchmarkExpandTab(b *testing.B) {
	c := NewCondition()
	s := "col1\tcol2\tcol3\tcol4"
	for range b.N {
		c.ExpandTab(s)
	}
}

func BenchmarkExpandTabFunc(b *testing.B) {
	c := NewCondition()
	s := "col1\tcol2\tcol3\tcol4"
	fn := func(nSpaces int) string {
		return "→" + strings.Repeat(" ", nSpaces-1)
	}
	for range b.N {
		c.ExpandTabFunc(s, fn)
	}
}

func BenchmarkWrap(b *testing.B) {
	c := NewCondition()

	b.Run("short", func(b *testing.B) {
		for range b.N {
			c.Wrap("hello world", 5)
		}
	})

	b.Run("long", func(b *testing.B) {
		s := strings.Repeat("hello world ", 20)
		for range b.N {
			c.Wrap(s, 40)
		}
	})

	b.Run("with_tabs", func(b *testing.B) {
		for range b.N {
			c.Wrap("col1\tcol2\tcol3\tcol4", 10)
		}
	})
}

func BenchmarkWrapSGR(b *testing.B) {
	c := &Condition{TabWidth: 4, ControlSequences: true}
	s := "\x1b[31m" + strings.Repeat("hello world ", 20) + "\x1b[0m"
	for range b.N {
		c.Wrap(s, 40)
	}
}

func BenchmarkTruncate(b *testing.B) {
	c := NewCondition()

	b.Run("no_tab", func(b *testing.B) {
		for range b.N {
			c.Truncate("hello world, this is a long string", 10, "...")
		}
	})

	b.Run("with_tab", func(b *testing.B) {
		for range b.N {
			c.Truncate("col1\tcol2\tcol3", 10, "...")
		}
	})
}

func BenchmarkFillRight(b *testing.B) {
	c := NewCondition()
	for range b.N {
		c.FillRight("hello", 20)
	}
}
