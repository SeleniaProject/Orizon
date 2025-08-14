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
	srv := NewServer(inR, outW)
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
	if caps["semanticTokensProvider"] == nil {
		t.Fatalf("missing semanticTokensProvider")
	}
	if _, ok := caps["inlayHintProvider"]; !ok {
		t.Fatalf("missing inlayHintProvider")
	}

	// exit
	writeFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	<-done
}

func TestSemanticTokensFullReturnsDataArray(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer inW.Close()
	defer outR.Close()
	srv := NewServer(inR, outW)
	done := make(chan struct{})
	go func() { _ = srv.Run(); close(done) }()
	r := bufio.NewReader(outR)

	// initialize
	writeFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	if _, err := readFramedJSON(t, r, 3*time.Second); err != nil {
		t.Fatalf("init: %v", err)
	}
	// open document
	src := "func main() { let x = 1 + 2 }\n"
	writeFramedJSON(t, inW, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///t.oriz", "text": src, "version": 1},
		},
	})
	if _, err := readFramedJSON(t, r, 3*time.Second); err != nil {
		t.Fatalf("diags: %v", err)
	}
	// request semantic tokens
	writeFramedJSON(t, inW, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/semanticTokens/full",
		"params":  map[string]any{"textDocument": map[string]any{"uri": "file:///t.oriz"}},
	})
	resp, err := readFramedJSON(t, r, 3*time.Second)
	if err != nil {
		t.Fatalf("semTokens: %v", err)
	}
	res := resp["result"].(map[string]any)
	if res == nil {
		t.Fatalf("no result")
	}
	if _, ok := res["data"].([]any); !ok {
		// Some clients/json decoders may decode []uint32 as []any; accept that it exists
		if _, ok2 := res["data"].([]uint32); !ok2 {
			t.Fatalf("data not array-like: %T", res["data"])
		}
	}
	// exit
	writeFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	<-done
}
