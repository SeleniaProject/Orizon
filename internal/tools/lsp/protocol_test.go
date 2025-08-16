package lsp

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

// writeFramedRaw writes a frame with a custom header name casing.
func writeFramedRaw(t *testing.T, w io.Writer, headerName string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := io.WriteString(w, headerName+": "); err != nil {
		t.Fatalf("header: %v", err)
	}
	if _, err := io.WriteString(w, itoa(len(data))); err != nil {
		t.Fatalf("len: %v", err)
	}
	if _, err := io.WriteString(w, "\r\n\r\n"); err != nil {
		t.Fatalf("crlf: %v", err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatalf("body: %v", err)
	}
}

func TestInitializedNotificationHasNoResponse(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()
	srv := NewServer(inR, outW)
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	r := bufio.NewReader(outR)

	// initialize request → expect response
	initReq := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}}
	writeFramedJSON(t, inW, initReq)
	if _, err := readFramedJSON(t, r, 3*time.Second); err != nil {
		t.Fatalf("initialize resp: %v", err)
	}

	// initialized notification (no id) → expect no response within timeout
	initialized := map[string]any{"jsonrpc": "2.0", "method": "initialized", "params": map[string]any{}}
	writeFramedJSON(t, inW, initialized)
	if _, err := readFramedJSON(t, r, 200*time.Millisecond); err == nil {
		t.Fatalf("expected no response to initialized notification")
	}

	// exit to stop server
	writeFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	<-done
}

func TestUnknownMethodReturnsError(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()
	srv := NewServer(inR, outW)
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	r := bufio.NewReader(outR)

	// Send unknown method request
	req := map[string]any{"jsonrpc": "2.0", "id": 42, "method": "foo/bar"}
	writeFramedJSON(t, inW, req)
	msg, err := readFramedJSON(t, r, 3*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg["error"] == nil {
		t.Fatalf("expected error for unknown method: %v", msg)
	}
	// exit
	writeFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	<-done
}

func TestLowercaseContentLengthHeader(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()
	srv := NewServer(inR, outW)
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	r := bufio.NewReader(outR)

	// Use lowercase header name to verify case-insensitive parsing
	initReq := map[string]any{"jsonrpc": "2.0", "id": 7, "method": "initialize", "params": map[string]any{}}
	writeFramedRaw(t, inW, strings.ToLower("Content-Length"), initReq)
	msg, err := readFramedJSON(t, r, 3*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	res, ok := msg["result"].(map[string]any)
	if !ok || res == nil {
		t.Fatalf("missing result in initialize: %v", msg)
	}
	caps := res["capabilities"].(map[string]any)
	if caps == nil || caps["textDocumentSync"] == nil {
		t.Fatalf("missing capabilities.textDocumentSync: %v", msg)
	}
	// exit
	writeFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	<-done
}

func TestCapabilitiesShape(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()
	srv := NewServer(inR, outW)
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	r := bufio.NewReader(outR)

	initReq := map[string]any{"jsonrpc": "2.0", "id": 11, "method": "initialize", "params": map[string]any{}}
	writeFramedJSON(t, inW, initReq)
	msg, err := readFramedJSON(t, r, 3*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	res := msg["result"].(map[string]any)
	caps := res["capabilities"].(map[string]any)
	if caps["positionEncoding"] != "utf-16" {
		t.Fatalf("unexpected positionEncoding: %v", caps["positionEncoding"])
	}
	tds := caps["textDocumentSync"].(map[string]any)
	if tds == nil || tds["change"].(float64) != 2 {
		t.Fatalf("textDocumentSync.change is not Incremental(2): %v", tds)
	}
	// exit
	writeFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	<-done
}
