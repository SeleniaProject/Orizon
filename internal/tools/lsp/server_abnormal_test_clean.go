package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// writeRaw writes raw bytes to the writer for testing edge cases
func writeRaw(t *testing.T, w io.Writer, data []byte) {
	t.Helper()
	if _, err := w.Write(data); err != nil {
		t.Fatalf("write raw: %v", err)
	}
}

// readJSONFrame reads one JSON-RPC response from r.
func readJSONFrame(t *testing.T, r *bufio.Reader, timeout time.Duration) (map[string]any, error) {
	t.Helper()
	type result struct {
		msg map[string]any
		err error
	}
	done := make(chan result)
	go func() {
		var res result
		defer func() { done <- res }()

		// Read headers
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				res.err = err
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break // End of headers
			}
			if strings.HasPrefix(strings.ToLower(line), "content-length:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) != 2 {
					res.err = io.ErrUnexpectedEOF
					return
				}
				lenStr := strings.TrimSpace(parts[1])
				contentLen, err := strconv.Atoi(lenStr)
				if err != nil {
					res.err = err
					return
				}

				// Read body
				body := make([]byte, contentLen)
				if _, err := io.ReadFull(r, body); err != nil {
					res.err = err
					return
				}

				var msg map[string]any
				if err := json.Unmarshal(body, &msg); err != nil {
					res.err = err
					return
				}
				res.msg = msg
				return
			}
		}
		res.err = io.ErrUnexpectedEOF
	}()

	select {
	case result := <-done:
		return result.msg, result.err
	case <-time.After(timeout):
		return nil, io.ErrUnexpectedEOF
	}
}

func TestServerRejectsOversizedHeaders(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows due to pipe behavior timing differences")
	}
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()

	srv := NewServer(inR, outW, &ServerOptions{MaxDocumentSize: 1024 * 1024, CacheSize: 100})
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	rd := bufio.NewReader(outR)

	// Build >100 header lines (no body)
	var hdr bytes.Buffer
	for i := 0; i < 101; i++ {
		hdr.WriteString("X-Foo: bar\r\n")
	}
	hdr.WriteString("Content-Length: 2\r\n\r\n")
	writeRaw(t, inW, hdr.Bytes())

	msg, err := readJSONFrame(t, rd, 5*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg["error"] == nil {
		t.Fatalf("expected error for too many headers")
	}
	<-done
}

func TestServerRejectsOversizedContent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows due to pipe behavior timing differences")
	}
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()

	srv := NewServer(inR, outW, &ServerOptions{MaxDocumentSize: 1024 * 1024, CacheSize: 100})
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	rd := bufio.NewReader(outR)

	// Content-Length above server hard cap
	writeRaw(t, inW, []byte("Content-Length: 99999999\r\n\r\n"))
	msg, err := readJSONFrame(t, rd, 5*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg["error"] == nil {
		t.Fatalf("expected error for oversized content")
	}
	<-done
}

func TestServerRejectsInvalidHeaders(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows due to pipe behavior timing differences")
	}
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()

	srv := NewServer(inR, outW, &ServerOptions{MaxDocumentSize: 1024 * 1024, CacheSize: 100})
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	rd := bufio.NewReader(outR)

	// Invalid header - no colon separator
	writeRaw(t, inW, []byte("ContentLength 50\r\n\r\n"))
	msg, err := readJSONFrame(t, rd, 5*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg["error"] == nil {
		t.Fatalf("expected error for invalid header format")
	}
	<-done
}
