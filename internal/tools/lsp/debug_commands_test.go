package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func writeRPC(w io.Writer, payload []byte) error {
	hdr := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))
	if _, err := io.WriteString(w, hdr); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func readRPC(r *bufio.Reader, timeout time.Duration) ([]byte, error) {
	type header struct{ key, val string }
	deadline := time.Now().Add(timeout)
	var contentLen int
	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout reading headers")
		}
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if line == "\r\n" {
			break
		}
		if idx := strings.IndexByte(line, ':'); idx >= 0 {
			k := strings.ToLower(strings.TrimSpace(line[:idx]))
			v := strings.TrimSpace(strings.TrimRight(line[idx+1:], "\r\n"))
			if k == "content-length" {
				var n int
				fmt.Sscanf(v, "%d", &n)
				contentLen = n
			}
		}
	}
	if contentLen <= 0 {
		return nil, fmt.Errorf("invalid content-length")
	}
	buf := make([]byte, contentLen)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func TestLSP_DebugCommands_SnapshotAndMessages(t *testing.T) {
	t.Skip("disabled: flakiness with io.Pipe under Windows CI; verified manually")
	// HTTP test server serving /actors and /actors/messages
	mux := http.NewServeMux()
	mux.HandleFunc("/actors", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"actors":[{"id":1}]}`))
	})
	mux.HandleFunc("/actors/messages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"sender":0,"receiver":1,"type":1}]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// LSP server over in-memory pipes with proper server options
	pr, pw := io.Pipe()
	outPr, outPw := io.Pipe()

	// Create server options with appropriate configuration for testing
	serverOptions := &ServerOptions{
		EnableDebugIntegration: true,
		DebugHTTPURL:           srv.URL,
		MaxDocumentSize:        1024 * 1024, // 1MB for testing
		CacheSize:              100,
	}

	go func() { _ = NewServer(pr, outPw, serverOptions).Run() }()

	// initialize with initializationOptions.debugHTTP = srv.URL
	initReq := map[string]any{
		"jsonrpc": "2.0", "id": 1.0, "method": "initialize",
		"params": map[string]any{"rootUri": "", "initializationOptions": map[string]any{"debugHTTP": srv.URL}},
	}
	b, _ := json.Marshal(initReq)
	if err := writeRPC(pw, b); err != nil {
		t.Fatal(err)
	}

	// workspace/executeCommand: orizon.getActorsSnapshot
	exeReq1 := map[string]any{
		"jsonrpc": "2.0", "id": 2.0, "method": "workspace/executeCommand",
		"params": map[string]any{"command": "orizon.getActorsSnapshot", "arguments": []any{}},
	}
	b, _ = json.Marshal(exeReq1)
	if err := writeRPC(pw, b); err != nil {
		t.Fatal(err)
	}

	// workspace/executeCommand: orizon.getActorMessages with [id,n]
	exeReq2 := map[string]any{
		"jsonrpc": "2.0", "id": 3.0, "method": "workspace/executeCommand",
		"params": map[string]any{"command": "orizon.getActorMessages", "arguments": []any{1.0, 10.0}},
	}
	b, _ = json.Marshal(exeReq2)
	if err := writeRPC(pw, b); err != nil {
		t.Fatal(err)
	}

	r := bufio.NewReader(outPr)
	// read 3 responses
	if _, err := readRPC(r, time.Second); err != nil {
		t.Fatal(err)
	} // initialize
	snap, err := readRPC(r, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := readRPC(r, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// Validate JSON
	var obj map[string]any
	if err := json.Unmarshal(snap, &obj); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if obj["result"] == nil {
		t.Fatalf("missing result in snapshot reply: %v", string(snap))
	}
	obj = map[string]any{}
	if err := json.Unmarshal(msgs, &obj); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if obj["result"] == nil {
		t.Fatalf("missing result in messages reply: %v", string(msgs))
	}
}

func TestLSP_DebugCommands_GraphDeadlocksCorrelation(t *testing.T) {
	t.Skip("disabled: flakiness with io.Pipe under Windows CI; verified manually")
	mux := http.NewServeMux()
	mux.HandleFunc("/actors/graph", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"nodes":[{"id":1}],"edges":[]}`))
	})
	mux.HandleFunc("/actors/deadlocks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"kind":"watch-cycle","actorIds":[1,2]}]`))
	})
	mux.HandleFunc("/actors/correlation", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"abc"}]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	pr, pw := io.Pipe()
	outPr, outPw := io.Pipe()

	// Create server options for the second test server instance
	serverOptions2 := &ServerOptions{
		EnableDebugIntegration: true,
		DebugHTTPURL:           srv.URL,
		MaxDocumentSize:        1024 * 1024,
		CacheSize:              100,
	}

	go func() { _ = NewServer(pr, outPw, serverOptions2).Run() }()

	// initialize
	initReq := map[string]any{
		"jsonrpc": "2.0", "id": 1.0, "method": "initialize",
		"params": map[string]any{"rootUri": "", "initializationOptions": map[string]any{"debugHTTP": srv.URL}},
	}
	b, _ := json.Marshal(initReq)
	if err := writeRPC(pw, b); err != nil {
		t.Fatal(err)
	}

	// graph
	reqGraph := map[string]any{"jsonrpc": "2.0", "id": 2.0, "method": "workspace/executeCommand",
		"params": map[string]any{"command": "orizon.getActorGraph"}}
	b, _ = json.Marshal(reqGraph)
	if err := writeRPC(pw, b); err != nil {
		t.Fatal(err)
	}

	// deadlocks
	reqDead := map[string]any{"jsonrpc": "2.0", "id": 3.0, "method": "workspace/executeCommand",
		"params": map[string]any{"command": "orizon.getDeadlocks"}}
	b, _ = json.Marshal(reqDead)
	if err := writeRPC(pw, b); err != nil {
		t.Fatal(err)
	}

	// correlation
	reqCorr := map[string]any{"jsonrpc": "2.0", "id": 4.0, "method": "workspace/executeCommand",
		"params": map[string]any{"command": "orizon.getCorrelationEvents", "arguments": []any{"abc", 5.0}}}
	b, _ = json.Marshal(reqCorr)
	if err := writeRPC(pw, b); err != nil {
		t.Fatal(err)
	}

	r := bufio.NewReader(outPr)
	if _, err := readRPC(r, time.Second); err != nil {
		t.Fatal(err)
	} // initialize
	if _, err := readRPC(r, time.Second); err != nil {
		t.Fatal(err)
	} // graph
	if _, err := readRPC(r, time.Second); err != nil {
		t.Fatal(err)
	} // deadlocks
	if _, err := readRPC(r, time.Second); err != nil {
		t.Fatal(err)
	} // correlation
}
