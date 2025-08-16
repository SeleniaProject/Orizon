package core

import "testing"

func TestStrings(t *testing.T) {
	if !Contains("hello", "ell") {
		t.Fatalf("contains")
	}
	if !HasPrefix("foobar", "foo") || !HasSuffix("foobar", "bar") {
		t.Fatalf("affix")
	}
	if Join([]string{"a", "b", "c"}, ",") != "a,b,c" {
		t.Fatalf("join")
	}
	if len(Split("a|b|c", "|")) != 3 {
		t.Fatalf("split")
	}
	if Trim("  x  ", " ") != "x" {
		t.Fatalf("trim")
	}
	if ToUpper("x") != "X" || ToLower("X") != "x" {
		t.Fatalf("case")
	}
	if RuneCount("åäö") != 3 {
		t.Fatalf("runes")
	}
	if Reverse("ab☕") != "☕ba" {
		t.Fatalf("reverse")
	}
}
