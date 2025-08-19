package lsp

import (
	"bufio"
	"io"
	"testing"
	"time"
)

func TestCapabilitiesIncludeSemanticTokensAndInlayHints(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()
	srv := NewServer(inR, outW, &ServerOptions{MaxDocumentSize: 1024 * 1024, CacheSize: 100})
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	r := bufio.NewReader(outR)

	writeFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	msg, err := readFramedJSON(t, r, 3*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	res := msg["result"].(map[string]any)
	caps := res["capabilities"].(map[string]any)

	semanticTokens := caps["semanticTokensProvider"]
	if semanticTokens == nil {
		t.Fatalf("missing semanticTokensProvider capability")
	}

	inlayHint := caps["inlayHintProvider"]
	if inlayHint == nil {
		t.Fatalf("missing inlayHintProvider capability")
	}

	writeFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	<-done
}

func TestSemanticTokensRequestHandlesRangeParameter(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()
	srv := NewServer(inR, outW, &ServerOptions{MaxDocumentSize: 1024 * 1024, CacheSize: 100})
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	r := bufio.NewReader(outR)

	// initialize
	writeFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	_, _ = readFramedJSON(t, r, 3*time.Second)

	// initialized
	writeFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "method": "initialized", "params": map[string]any{}})
	_, _ = readFramedJSON(t, r, 200*time.Millisecond)

	// textDocument/didOpen
	writeFramedJSON(t, inW, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri":        "file:///tmp/test.oriz",
				"languageId": "orizon",
				"version":    1,
				"text":       "func add(a: i32, b: i32) -> i32 { return a + b; }",
			},
		},
	})

	// semantic tokens request with range
	writeFramedJSON(t, inW, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/semanticTokens/range",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///tmp/test.oriz"},
			"range": map[string]any{
				"start": map[string]any{"line": 0, "character": 0},
				"end":   map[string]any{"line": 0, "character": 20},
			},
		},
	})
	msg, err := readFramedJSON(t, r, 3*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg["error"] != nil {
		t.Fatalf("semantic tokens request failed: %v", msg["error"])
	}

	writeFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	<-done
}
