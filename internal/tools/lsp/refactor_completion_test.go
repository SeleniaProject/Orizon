package lsp

import (
	"bufio"
	"io"
	"testing"
	"time"
)

func TestCompletionResolveAndExtractRefactor(t *testing.T) {
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
	if _, err := readFramedJSON(t, r, 2*time.Second); err != nil {
		t.Fatalf("init resp: %v", err)
	}

	// didOpen simple doc with one expression
	src := "func main() { let x = 1 + 2 }\n"
	writeFramedJSON(t, inW, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///a.oriz", "text": src, "version": 1},
		},
	})
	if _, err := readFramedJSON(t, r, 3*time.Second); err != nil { // diagnostics
		t.Fatalf("open diags: %v", err)
	}

	// completion at start should include keyword items with data
	writeFramedJSON(t, inW, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///a.oriz"},
			"position":     map[string]any{"line": 0, "character": 0},
		},
	})
	comp, err := readFramedJSON(t, r, 2*time.Second)
	if err != nil {
		t.Fatalf("completion: %v", err)
	}
	res := comp["result"].(map[string]any)
	items := res["items"].([]any)
	if len(items) == 0 {
		t.Fatalf("no completion items")
	}

	// pick first item and call resolve
	item := items[0].(map[string]any)
	writeFramedJSON(t, inW, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "completionItem/resolve", "params": item})
	if _, err := readFramedJSON(t, r, 2*time.Second); err != nil {
		t.Fatalf("resolve: %v", err)
	}

	// codeAction: select "1 + 2" to extract
	// find positions: the expression is on line 0, after "let x = " which is 12 chars in ASCII here
	writeFramedJSON(t, inW, map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "textDocument/codeAction",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///a.oriz"},
			"range": map[string]any{
				"start": map[string]any{"line": 0, "character": 18},
				"end":   map[string]any{"line": 0, "character": 23},
			},
			"context": map[string]any{"diagnostics": []any{}},
		},
	})
	acts, err := readFramedJSON(t, r, 2*time.Second)
	if err != nil {
		t.Fatalf("codeAction: %v", err)
	}
	if acts["result"] == nil {
		t.Fatalf("no actions")
	}
}
