package lsp

import (
	"bufio"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"
)

// testWriteFramedJSON writes a JSON object as a framed LSP message.
func testWriteFramedJSON(t *testing.T, w io.Writer, v any) {
	testWriteFramedRaw(t, w, "Content-Length", v)
}

// testWriteFramedRaw writes any value as a framed message with specified header.
func testWriteFramedRaw(t *testing.T, w io.Writer, headerName string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := io.WriteString(w, headerName+": "); err != nil {
		t.Fatalf("header: %v", err)
	}
	if _, err := io.WriteString(w, strconv.Itoa(len(data))); err != nil {
		t.Fatalf("len: %v", err)
	}
	if _, err := io.WriteString(w, "\r\n\r\n"); err != nil {
		t.Fatalf("crlf: %v", err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatalf("body: %v", err)
	}
}

func TestProtocolBasics(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()
	srv := NewServer(inR, outW, &ServerOptions{MaxDocumentSize: 1024 * 1024, CacheSize: 100})
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	r := bufio.NewReader(outR)

	// Test initialize request.
	initReq := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}}
	testWriteFramedJSON(t, inW, initReq)
	msg, err := testReadFramedJSON(t, r, 3*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg["id"] != float64(1) || msg["result"] == nil {
		t.Fatalf("unexpected response: %v", msg)
	}
	// exit.
	testWriteFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	<-done
}

func TestInitializedNotificationHasNoResponse(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()
	srv := NewServer(inR, outW, &ServerOptions{MaxDocumentSize: 1024 * 1024, CacheSize: 100})
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	r := bufio.NewReader(outR)

	// initialized notification (no id) → expect no response within timeout.
	initialized := map[string]any{"jsonrpc": "2.0", "method": "initialized", "params": map[string]any{}}
	testWriteFramedJSON(t, inW, initialized)
	if _, err := testReadFramedJSON(t, r, 200*time.Millisecond); err == nil {
		t.Fatalf("expected no response to initialized notification")
	}

	// exit to stop server.
	testWriteFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	<-done
}

func TestUnknownMethodReturnsError(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()
	srv := NewServer(inR, outW, &ServerOptions{MaxDocumentSize: 1024 * 1024, CacheSize: 100})
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	r := bufio.NewReader(outR)

	// Send unknown method request.
	req := map[string]any{"jsonrpc": "2.0", "id": 42, "method": "foo/bar"}
	testWriteFramedJSON(t, inW, req)
	msg, err := testReadFramedJSON(t, r, 3*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg["error"] == nil {
		t.Fatalf("expected error for unknown method: %v", msg)
	}
	// exit.
	testWriteFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	<-done
}

func TestLowercaseContentLengthHeader(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()
	srv := NewServer(inR, outW, &ServerOptions{MaxDocumentSize: 1024 * 1024, CacheSize: 100})
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	r := bufio.NewReader(outR)

	// content-length (lowercase) should work too.
	testWriteFramedRaw(t, inW, "content-length", map[string]any{"jsonrpc": "2.0", "id": 5, "method": "initialize", "params": map[string]any{}})
	msg, err := testReadFramedJSON(t, r, 3*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	res := msg["result"].(map[string]any)
	if res == nil {
		t.Fatalf("missing result in initialize: %v", msg)
	}
	caps := res["capabilities"].(map[string]any)
	if caps == nil || caps["textDocumentSync"] == nil {
		t.Fatalf("missing capabilities.textDocumentSync: %v", msg)
	}
	// exit.
	testWriteFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	<-done
}

func TestCapabilitiesShape(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()
	srv := NewServer(inR, outW, &ServerOptions{MaxDocumentSize: 1024 * 1024, CacheSize: 100})
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	r := bufio.NewReader(outR)

	initReq := map[string]any{"jsonrpc": "2.0", "id": 11, "method": "initialize", "params": map[string]any{}}
	testWriteFramedJSON(t, inW, initReq)
	msg, err := testReadFramedJSON(t, r, 3*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	res := msg["result"].(map[string]any)
	caps := res["capabilities"].(map[string]any)
	if caps["positionEncoding"] != "utf-16" {
		t.Fatalf("unexpected positionEncoding: %v", caps["positionEncoding"])
	}

	// check textDocumentSync sub-capabilities.
	tds := caps["textDocumentSync"].(map[string]any)
	if tds["openClose"] != true {
		t.Fatalf("expected openClose: true, got %v", tds["openClose"])
	}
	if tds["change"] != float64(1) {
		t.Fatalf("expected change: 1, got %v", tds["change"])
	}

	// exit.
	testWriteFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	<-done
}

// testReadFramedJSON reads one framed JSON message with timeout.
func testReadFramedJSON(t *testing.T, r *bufio.Reader, timeout time.Duration) (map[string]any, error) {
	done := make(chan struct {
		msg map[string]any
		err error
	})
	go func() {
		var result struct {
			msg map[string]any
			err error
		}
		defer func() { done <- result }()

		// Read headers.
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				result.err = err
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break // End of headers
			}
			if strings.HasPrefix(strings.ToLower(line), "content-length:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) != 2 {
					result.err = io.ErrUnexpectedEOF
					return
				}
				lenStr := strings.TrimSpace(parts[1])
				contentLen, err := strconv.Atoi(lenStr)
				if err != nil {
					result.err = err
					return
				}

				// Read body.
				body := make([]byte, contentLen)
				if _, err := io.ReadFull(r, body); err != nil {
					result.err = err
					return
				}

				var msg map[string]any
				if err := json.Unmarshal(body, &msg); err != nil {
					result.err = err
					return
				}
				result.msg = msg
				return
			}
		}
		result.err = io.ErrUnexpectedEOF
	}()

	select {
	case result := <-done:
		return result.msg, result.err
	case <-time.After(timeout):
		return nil, io.ErrUnexpectedEOF
	}
}
