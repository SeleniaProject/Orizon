package main

import (
	"bytes"
	"testing"

	orifmt "github.com/orizon-lang/orizon/internal/format"
)

func TestFormat_TrimTrailingSpaces_LF(t *testing.T) {
	in := []byte("a  \n b\t\t  \n")
	got := orifmt.FormatBytes(in, orifmt.Options{PreserveNewlineStyle: false})
	want := []byte("a\n b\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("LF trim failed:\n got=%q\nwant=%q", string(got), string(want))
	}
}

func TestFormat_EnsureTrailingNewline_WhenMissing(t *testing.T) {
	in := []byte("no-newline")
	got := orifmt.FormatBytes(in, orifmt.Options{PreserveNewlineStyle: false})
	want := []byte("no-newline\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("ensure trailing newline failed:\n got=%q\nwant=%q", string(got), string(want))
	}
}

func TestFormat_PreserveCRLF_OnFiles(t *testing.T) {
	in := []byte("x  \r\ny\t \r\n") // CRLF input
	got := orifmt.FormatBytes(in, orifmt.Options{PreserveNewlineStyle: true})
	want := []byte("x\r\ny\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("CRLF preservation failed:\n got=%q\nwant=%q", string(got), string(want))
	}
}

func TestFormat_EmptyInput_ProducesSingleNewline(t *testing.T) {
	got := orifmt.FormatBytes(nil, orifmt.Options{PreserveNewlineStyle: false})
	want := []byte("\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("empty input formatting failed: got=%q want=%q", string(got), string(want))
	}
}
