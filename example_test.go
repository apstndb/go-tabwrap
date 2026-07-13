package tabwrap_test

import (
	"fmt"
	"strings"

	"github.com/apstndb/go-tabwrap"
)

func ExampleStringWidth() {
	fmt.Println(tabwrap.StringWidth("e\u0301"))
	fmt.Println(tabwrap.StringWidth("a\tb"))
	fmt.Println(tabwrap.StringWidth("ab\n日本"))
	// Output:
	// 1
	// 5
	// 4
}

func ExampleWrap() {
	fmt.Println(tabwrap.Wrap("hello world", 5))
	// Output:
	// hello
	//  worl
	// d
}

func ExampleCut() {
	result := tabwrap.Cut("\tX", 4)
	fmt.Printf("text=%q rest=%q width=%d overflow=%t\n", result.Text, result.Rest, result.Width, result.Overflow)
	// Output: text="    " rest="X" width=4 overflow=false
}

func ExampleWrapLines() {
	for _, line := range tabwrap.WrapLines("abcdef", 2, 3) {
		fmt.Printf("text=%q break=%q width=%d\n", line.Text, line.LineBreak, line.Width)
	}
	// Output:
	// text="ab" break="\n" width=2
	// text="cde" break="\n" width=3
	// text="f" break="" width=1
}

func ExampleTruncate() {
	fmt.Println(tabwrap.Truncate("hello world", 8, "..."))
	fmt.Println(tabwrap.Truncate("日本語テスト", 7, "..."))
	// Output:
	// hello...
	// 日本...
}

func ExampleTruncateInfo() {
	result := tabwrap.TruncateInfo("hello world", 8, "...")
	fmt.Println(result.Text)
	fmt.Println(result.Width)
	fmt.Println(result.Truncated)
	// Output:
	// hello...
	// 8
	// true
}

func ExampleFillLeft() {
	fmt.Printf("%q\n", tabwrap.FillLeft("42", 5))
	// Output: "   42"
}

func ExampleFillRight() {
	fmt.Printf("%q\n", tabwrap.FillRight("42", 5))
	// Output: "42   "
}

func ExampleExpandTabFunc() {
	out := tabwrap.ExpandTabFunc("a\tb\nabc\t", func(nSpaces int) string {
		return strings.Repeat(".", nSpaces)
	})
	fmt.Println(out)
	// Output:
	// a...b
	// abc.
}

func ExampleCondition_trimTrailingSpace() {
	c := &tabwrap.Condition{TrimTrailingSpace: true}
	fmt.Println(c.Wrap("ab\tcd", 4))
	// Output:
	// ab
	// cd
}
