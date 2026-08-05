package vow_test

import (
	"testing"

	"github.com/mgiaccone/vow"
)

func TestTrim(t *testing.T) {
	if got := vow.Trim("  hello  "); got != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}
}

func TestLower(t *testing.T) {
	if got := vow.Lower("HeLLo"); got != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}
}

func TestUpper(t *testing.T) {
	if got := vow.Upper("HeLLo"); got != "HELLO" {
		t.Fatalf("got %q, want %q", got, "HELLO")
	}
}

func TestCollapse(t *testing.T) {
	cases := []struct{ in, want string }{
		{"  a   b\tc\n\nd  ", "a b c d"},
		{"", ""},
		{"   ", ""},
		{"already fine", "already fine"},
	}
	for _, c := range cases {
		if got := vow.Collapse(c.in); got != c.want {
			t.Errorf("Collapse(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
