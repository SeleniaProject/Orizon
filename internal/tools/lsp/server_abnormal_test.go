package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"runtime"
	"testing"
	"time"
)

// writeRaw writes raw bytes to w.
func writeRaw(t *testing.T, w io.Writer, b []byte) {
	t.Helper()
	if _, err := w.Write(b); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// readJSONFrame reads one JSON-RPC response from r.
func readJSONFrame(t *testing.T, r *bufio.Reader, timeout time.Duration) (map[string]any, error) {
	t.Helper()
	ch := make(chan struct {
		msg map[string]any
		err error
	}, 1)
	go func() {
		// parse headers
		contentLength := 0
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				ch <- struct {
					msg map[string]any
					err error
				}{nil, err}
				return
			}
			if line == "\r\n" {
				break
			}
			if idx := bytes.IndexByte([]byte(line), ':'); idx >= 0 {
				name := bytes.ToLower(bytes.TrimSpace([]byte(line[:idx])))
				if string(name) == "content-length" {
					val := bytes.TrimSpace([]byte(line[idx+1:]))
					val = bytes.TrimRight(val, "\r\n")
					n := 0
					for i := 0; i < len(val); i++ {
						n = n*10 + int(val[i]-'0')
					}
					contentLength = n
				}
			}
		}
		if contentLength <= 0 {
			ch <- struct {
				msg map[string]any
				err error
			}{nil, io.ErrUnexpectedEOF}
			return
		}
		body := make([]byte, contentLength)
		if _, err := io.ReadFull(r, body); err != nil {
			ch <- struct {
				msg map[string]any
				err error
			}{nil, err}
			return
		}
		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			ch <- struct {
				msg map[string]any
				err error
			}{nil, err}
			return
		}
		ch <- struct {
			msg map[string]any
			err error
		}{m, nil}
	}()
	select {
	case res := <-ch:
		return res.msg, res.err
	case <-time.After(timeout):
		return nil, io.EOF
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

	srv := NewServer(inR, outW)
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
	if errObj, ok := msg["error"].(map[string]any); !ok || int(errObj["code"].(float64)) != -32600 {
		t.Fatalf("expected -32600 for oversized headers, got: %v", msg)
	}

	// Stop the server by closing input
	_ = inW.Close()
	<-done
}

func TestServerRejectsHugeContentLength(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows due to pipe behavior timing differences")
	}
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()

	srv := NewServer(inR, outW)
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	rd := bufio.NewReader(outR)

	// Content-Length above server hard cap
	writeRaw(t, inW, []byte("Content-Length: 99999999\r\n\r\n"))
	msg, err := readJSONFrame(t, rd, 5*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if errObj, ok := msg["error"].(map[string]any); !ok || int(errObj["code"].(float64)) != -32600 {
		t.Fatalf("expected -32600 for huge content length, got: %v", msg)
	}

	_ = inW.Close()
	<-done
}

func TestServerRejectsMalformedJSON(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows due to pipe behavior timing differences")
	}
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()

	srv := NewServer(inR, outW)
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	rd := bufio.NewReader(outR)

	// Correct headers but malformed JSON body
	body := []byte("{\"jsonrpc\": \"2.0\", \"id\": 1, \"method\": \"initialize\", \"params\": ") // truncated
	var hdr bytes.Buffer
	hdr.WriteString("Content-Length: ")
	hdr.WriteString(itoa(len(body)))
	hdr.WriteString("\r\n\r\n")
	writeRaw(t, inW, hdr.Bytes())
	writeRaw(t, inW, body)

	msg, err := readJSONFrame(t, rd, 5*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if errObj, ok := msg["error"].(map[string]any); !ok || int(errObj["code"].(float64)) != -32700 {
		t.Fatalf("expected -32700 for parse error, got: %v", msg)
	}

	_ = inW.Close()
	<-done
}
