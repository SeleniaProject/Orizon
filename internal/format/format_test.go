package format

import (
	"testing"
)

func TestFormatText_TrailingSpaceAndNewline_LF(t *testing.T) {
	in := "a  \n b\t\t  \n"
	got := FormatText(in, Options{PreserveNewlineStyle: false})
	want := "a\n b\n"
	if got != want {
		t.Fatalf("LF trim failed: got=%q want=%q", got, want)
	}
}

func TestFormatText_EnsureTrailingNewline_WhenMissing(t *testing.T) {
	in := "no-newline"
	got := FormatText(in, Options{PreserveNewlineStyle: false})
	want := "no-newline\n"
	if got != want {
		t.Fatalf("ensure trailing newline failed: got=%q want=%q", got, want)
	}
}

func TestFormatText_PreserveCRLF_OnFiles(t *testing.T) {
	in := "x  \r\ny\t \r\n"
	got := FormatText(in, Options{PreserveNewlineStyle: true})
	want := "x\r\ny\r\n"
	if got != want {
		t.Fatalf("CRLF preservation failed: got=%q want=%q", got, want)
	}
}

func TestFormatText_EmptyInput_ProducesSingleNewline(t *testing.T) {
	got := FormatText("", Options{PreserveNewlineStyle: false})
	want := "\n"
	if got != want {
		t.Fatalf("empty input formatting failed: got=%q want=%q", got, want)
	}
}
