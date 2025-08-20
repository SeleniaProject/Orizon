package stringsx

import "testing"

func TestTrimPrefixSuffix(t *testing.T) {
	if TrimPrefix("foobar", "foo") != "bar" {
		t.Fatal()
	}

	if TrimPrefix("foobar", "baz") != "foobar" {
		t.Fatal()
	}

	if TrimSuffix("foobar", "bar") != "foo" {
		t.Fatal()
	}

	if TrimSuffix("foobar", "baz") != "foobar" {
		t.Fatal()
	}
}

func TestSplitOnce(t *testing.T) {
	l, r := SplitOnce("a=b=c", "=")
	if l != "a" || r != "b=c" {
		t.Fatalf("%q %q", l, r)
	}

	l, r = SplitOnce("abc", ":")
	if l != "abc" || r != "" {
		t.Fatalf("%q %q", l, r)
	}
}
