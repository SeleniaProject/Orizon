package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"
)

// Simple helpers to avoid importing strings/strconv to keep test local and deterministic.
func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c - 'A' + 'a'
		}
		b[i] = c
	}
	return string(b)
}

func trimSpace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	j := len(s)
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}

func atoi(s string) (int, bool) {
	n := 0
	if s == "" {
		return 0, false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
		n = n*10 + int(s[i]-'0')
	}
	return n, true
}

// writeFramedJSON writes a JSON-RPC message with LSP framing to w.
func writeFramedJSON(t *testing.T, w io.Writer, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := io.WriteString(w, "Content-Length: "); err != nil {
		t.Fatalf("header write: %v", err)
	}
	if _, err := io.WriteString(w, itoa(len(data))); err != nil {
		t.Fatalf("len write: %v", err)
	}
	if _, err := io.WriteString(w, "\r\n\r\n"); err != nil {
		t.Fatalf("crlf write: %v", err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatalf("body write: %v", err)
	}
}

// readFramedJSON reads a single framed JSON message and returns it as a generic map.
func readFramedJSON(t *testing.T, r *bufio.Reader, timeout time.Duration) (map[string]any, error) {
	t.Helper()
	type result struct {
		msg map[string]any
		err error
	}
	ch := make(chan result, 1)
	go func() {
		// Read headers until blank line; track Content-Length
		contentLength := 0
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				ch <- result{nil, err}
				return
			}
			if line == "\r\n" {
				// Ignore stray blank lines before headers
				if contentLength == 0 {
					continue
				}
				break
			}
			if idx := indexByte(line, ':'); idx >= 0 {
				name := toLower(trimSpace(line[:idx]))
				if name == "content-length" {
					val := trimSpace(line[idx+1:])
					// strip trailing CRLF if present
					for len(val) > 0 && (val[len(val)-1] == '\r' || val[len(val)-1] == '\n') {
						val = val[:len(val)-1]
					}
					if n, ok := atoi(val); ok {
						contentLength = n
					}
				}
			}
		}
		if contentLength <= 0 {
			ch <- result{nil, io.ErrUnexpectedEOF}
			return
		}
		body := make([]byte, contentLength)
		if _, err := io.ReadFull(r, body); err != nil {
			ch <- result{nil, err}
			return
		}
		var msg map[string]any
		if err := json.Unmarshal(body, &msg); err != nil {
			ch <- result{nil, err}
			return
		}
		ch <- result{msg, nil}
	}()

	select {
	case res := <-ch:
		return res.msg, res.err
	case <-time.After(timeout):
		return nil, io.EOF
	}
}

func TestInitializeOpenHoverShutdown(t *testing.T) {
	// Create in/out pipes to drive the server in-process
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()

	srv := NewServer(inR, outW)

	// Run server in background goroutine
	done := make(chan struct{})
	go func() {
		_ = srv.Run()
		close(done)
	}()

	outReader := bufio.NewReader(outR)

	// 1) initialize request → expect response with capabilities
	initReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{},
	}
	writeFramedJSON(t, inW, initReq)

	msg, err := readFramedJSON(t, outReader, 5*time.Second)
	if err != nil {
		t.Fatalf("read initialize response: %v", err)
	}
	if _, ok := msg["result"].(map[string]any); !ok {
		t.Fatalf("initialize missing result: %v", msg)
	}
	caps := msg["result"].(map[string]any)["capabilities"]
	if caps == nil {
		t.Fatalf("initialize missing capabilities: %v", msg)
	}

	// 2) didOpen notification → expect publishDiagnostics notification
	source := "func main() { let x = 1 }\n"
	didOpen := map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":  "file:///test.oriz",
				"text": source,
			},
		},
	}
	writeFramedJSON(t, inW, didOpen)

	// The next message should be a publishDiagnostics notification
	diagMsg, err := readFramedJSON(t, outReader, 5*time.Second)
	if err != nil {
		t.Fatalf("read diagnostics: %v", err)
	}
	if m, ok := diagMsg["method"].(string); !ok || m != "textDocument/publishDiagnostics" {
		t.Fatalf("expected publishDiagnostics, got: %v", diagMsg)
	}

	// 3) hover request over identifier x (line 0, near position of x)
	// Find position of 'x' in UTF-16 units
	bytePos := bytes.IndexByte([]byte(source), 'x')
	if bytePos <= 0 {
		t.Fatalf("x not found in source")
	}
	line, ch := utf16LineCharFromOffset(source, bytePos)
	hoverReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/hover",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///test.oriz"},
			"position":     map[string]any{"line": line, "character": ch},
		},
	}
	writeFramedJSON(t, inW, hoverReq)

	hoverResp, err := readFramedJSON(t, outReader, 5*time.Second)
	if err != nil {
		t.Fatalf("read hover response: %v", err)
	}
	if _, ok := hoverResp["result"].(map[string]any); !ok {
		t.Fatalf("hover missing result: %v", hoverResp)
	}

	// 4) shutdown request + exit notification
	shutdown := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "shutdown",
		"params":  map[string]any{},
	}
	writeFramedJSON(t, inW, shutdown)
	if _, err := readFramedJSON(t, outReader, 5*time.Second); err != nil {
		t.Fatalf("read shutdown response: %v", err)
	}

	exit := map[string]any{
		"jsonrpc": "2.0",
		"method":  "exit",
		"params":  map[string]any{},
	}
	writeFramedJSON(t, inW, exit)

	// Wait for server loop to exit
	select {
	case <-done:
		// ok
	case <-time.After(5 * time.Second):
		t.Fatalf("server did not exit after 'exit' message")
	}
}
