package lsp

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"unicode"
	"unicode/utf8"

	"github.com/orizon-lang/orizon/internal/format"
	"github.com/orizon-lang/orizon/internal/lexer"
	"github.com/orizon-lang/orizon/internal/parser"
)

// JSON-RPC 2.0 structures (minimal)
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *RespError      `json:"error,omitempty"`
}

type RespError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// LSP server minimal capabilities
type Server struct {
	in   *bufio.Reader
	out  io.Writer
	docs map[string]string
	// symIndex stores symbol definitions per document URI: name -> list of symbol infos
	symIndex map[string]map[string][]SymbolInfo
	// astCache keeps the last parsed AST per document for tooling features
	astCache map[string]*parser.Program
	// docsVer holds the last known version per document URI
	docsVer map[string]int
	// Incremental lexing engine and token cache per document
	incLexer *lexer.IncrementalLexer
	tokCache map[string][]lexer.Token
	// Workspace info
	rootURI string
	// Global symbol index across documents: name -> list of (uri, span)
	wsIndex map[string][]SymbolLocation
	// diagnostics cache to avoid redundant notifications (hashed JSON of diags)
	diagCache map[string]string

	// debugHTTPURL holds an optional HTTP endpoint that serves actor snapshots
	// via `/actors` and `/actors/messages`. When set, custom commands can fetch
	// live runtime diagnostics directly from a running Orizon runtime.
	debugHTTPURL string

	// rspAddr holds an optional TCP address of the RSP server for pretty queries
	rspAddr string

	// lightweight metrics (volatile)
	reqCount uint64
	errCount uint64
	bytesOut uint64

	// semantic tokens configuration (legend)
	semLegend struct {
		Types     []string
		TypeIndex map[string]int
		Mods      []string
		ModIndex  map[string]int
	}

	// vdocCache caches last shown content per uriHint to avoid redundant notifications
	vdocCache map[string]string
	// vdocOpened tracks whether a given uriHint has been asked to open
	vdocOpened map[string]bool
	// rpcID is a monotonically increasing id for server->client requests
	rpcID uint64
}

func NewServer(r io.Reader, w io.Writer) *Server {
	s := &Server{
		in:         bufio.NewReader(r),
		out:        w,
		docs:       make(map[string]string),
		symIndex:   make(map[string]map[string][]SymbolInfo),
		astCache:   make(map[string]*parser.Program),
		docsVer:    make(map[string]int),
		incLexer:   lexer.NewIncrementalLexer(),
		tokCache:   make(map[string][]lexer.Token),
		wsIndex:    make(map[string][]SymbolLocation),
		diagCache:  make(map[string]string),
		vdocCache:  make(map[string]string),
		vdocOpened: make(map[string]bool),
	}
	// Initialize semantic tokens legend per LSP spec (rich, covers our needs)
	s.semLegend.Types = []string{
		"namespace", "type", "class", "enum", "interface", "struct", "typeParameter",
		"parameter", "variable", "property", "enumMember", "event", "function", "method",
		"macro", "keyword", "modifier", "comment", "string", "number", "regexp", "operator",
	}
	s.semLegend.TypeIndex = make(map[string]int, len(s.semLegend.Types))
	for i, t := range s.semLegend.Types {
		s.semLegend.TypeIndex[t] = i
	}
	s.semLegend.Mods = []string{"declaration"}
	s.semLegend.ModIndex = map[string]int{"declaration": 0}
	return s
}

// SymbolLocation represents a symbol occurrence in workspace
type SymbolLocation struct {
	URI  string
	Span parser.Span
	Kind int
}

// SymbolInfo represents an indexed symbol in a document
type SymbolInfo struct {
	Name   string
	Kind   int // LSP SymbolKind
	Span   parser.Span
	Detail string      // Human-readable signature or type summary
	Scope  parser.Span // Enclosing scope span (function/block), zero if global
}

// Run implements a Content-Length framed JSON-RPC loop
func (s *Server) Run() error {
	for {
		// read headers
		contentLength := 0
		// Header safety limits
		const (
			maxHeaderBytes   = 32 << 10 // 32 KiB
			maxHeaderLines   = 100
			maxContentLength = 8 << 20 // 8 MiB hard cap
		)
		headerBytes := 0
		headerLines := 0
		invalidHeaders := false
		for {
			line, err := s.in.ReadString('\n')
			if err != nil {
				return err
			}
			headerBytes += len(line)
			headerLines++
			if headerBytes > maxHeaderBytes || headerLines > maxHeaderLines {
				s.replyError(nil, -32600, "headers too large")
				invalidHeaders = true
				break
			}
			if line == "\r\n" {
				break
			}
			// Parse case-insensitive header name and numeric value
			if idx := strings.IndexByte(line, ':'); idx >= 0 {
				name := strings.TrimSpace(strings.ToLower(line[:idx]))
				if name == "content-length" {
					val := strings.TrimSpace(line[idx+1:])
					// Strip optional trailing CRLF already removed by ReadString('\n')
					val = strings.TrimRight(val, "\r\n")
					if n, err := strconv.Atoi(val); err == nil {
						contentLength = n
					}
				}
			}
		}
		if invalidHeaders {
			continue
		}
		// Reject invalid or overly large requests to prevent memory exhaustion
		if contentLength <= 0 || contentLength > maxContentLength {
			s.replyError(nil, -32600, "invalid content length")
			continue
		}
		// Stream decode request body to avoid allocating a large contiguous buffer
		lr := &io.LimitedReader{R: s.in, N: int64(contentLength)}
		dec := json.NewDecoder(lr)
		var req Request
		if err := dec.Decode(&req); err != nil {
			// Malformed request; respond with parse error per JSON-RPC
			s.replyError(nil, -32700, "parse error")
			continue
		}
		// Drain any unread bytes from the limited reader (e.g., trailing whitespace)
		if lr.N > 0 {
			_, _ = io.CopyN(io.Discard, lr, lr.N)
		}
		atomic.AddUint64(&s.reqCount, 1)
		switch req.Method {
		case "orz/stats":
			// Non-standard method to fetch server statistics (useful for diagnostics)
			s.reply(req.ID, s.Stats())
		case "initialize":
			// Advertise capabilities and server info per LSP. Use UTF-16 positions for compatibility.
			// Capture rootURI if provided
			var initParams struct {
				RootURI               string         `json:"rootUri"`
				InitializationOptions map[string]any `json:"initializationOptions"`
			}
			if err := json.Unmarshal(req.Params, &initParams); err != nil {
				s.replyError(req.ID, -32602, "invalid params: initialize")
				break
			}
			s.rootURI = initParams.RootURI
			// Configure optional debug HTTP endpoint from initializationOptions or environment
			if initParams.InitializationOptions != nil {
				if v, ok := initParams.InitializationOptions["debugHTTP"].(string); ok {
					s.debugHTTPURL = strings.TrimSpace(v)
				}
				if v, ok := initParams.InitializationOptions["rspAddr"].(string); ok {
					s.rspAddr = strings.TrimSpace(v)
				}
			}
			if s.debugHTTPURL == "" {
				s.debugHTTPURL = strings.TrimSpace(os.Getenv("ORIZON_DEBUG_HTTP"))
			}
			if s.rspAddr == "" {
				s.rspAddr = strings.TrimSpace(os.Getenv("ORIZON_RSP_ADDR"))
			}
			caps := map[string]any{
				"positionEncoding": "utf-16",
				"textDocumentSync": map[string]any{
					"openClose": true,
					"change":    2, // Incremental
				},
				"completionProvider": map[string]any{
					"triggerCharacters": []string{".", ":", ",", "(", "[", " "},
					"resolveProvider":   true,
				},
				"hoverProvider":          true,
				"typeDefinitionProvider": true,
				"definitionProvider":     true,
				"referencesProvider":     true,
				"documentSymbolProvider": true,
				"codeActionProvider": map[string]any{
					"codeActionKinds": []string{"quickfix", "refactor", "refactor.extract"},
				},
				"documentFormattingProvider": map[string]any{
					"workDoneProgress": false,
					// Extended capabilities for AST-aware formatting
					"supportsASTFormatting": true,
					"supportsDiffEdits":     true,
				},
				"documentRangeFormattingProvider": map[string]any{
					"workDoneProgress":      false,
					"supportsASTFormatting": true,
					"supportsDiffEdits":     true,
				},
				"signatureHelpProvider": map[string]any{"triggerCharacters": []string{"(", ","}},
				"documentOnTypeFormattingProvider": map[string]any{
					"firstTriggerCharacter":      "}",
					"moreTriggerCharacter":       []string{"\n", ")", ";", "{"},
					"supportsEnhancedFormatting": true,
				},
				"documentHighlightProvider": true,
				"foldingRangeProvider":      true,
				"renameProvider":            true,
				"workspaceSymbolProvider":   true,
				"semanticTokensProvider": map[string]any{
					"legend": map[string]any{
						"tokenTypes":     s.semLegend.Types,
						"tokenModifiers": s.semLegend.Mods,
					},
					"full":  true,
					"range": true,
				},
				"inlayHintProvider": true,
				"executeCommandProvider": map[string]any{
					"commands": []string{
						"orizon.getActorsSnapshot",
						"orizon.getActorMessages",
						"orizon.getActorGraph",
						"orizon.getDeadlocks",
						"orizon.getCorrelationEvents",
						"orizon.getPrettyLocals",
						"orizon.getPrettyMemory",
						"orizon.getStack",
						"orizon.getActorMetrics",
						"orizon.getIOMetrics",
						"orizon.getMailboxStats",
						"orizon.getActorIOMetrics",
						"orizon.lookupActorID",
						"orizon.getTopIOMetrics",
					},
				},
			}
			result := map[string]any{
				"capabilities": caps,
				"serverInfo": map[string]any{
					"name":    "orizon-lsp",
					"version": "dev",
				},
			}
			s.reply(req.ID, result)
		case "workspace/executeCommand":
			var p struct {
				Command   string `json:"command"`
				Arguments []any  `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: executeCommand")
				break
			}
			if p.Command == "orizon.lookupActorID" {
				if s.debugHTTPURL == "" {
					s.replyError(req.ID, -32603, "debugHTTP not configured")
					break
				}
				var name string
				if len(p.Arguments) > 0 {
					if v, ok := p.Arguments[0].(string); ok {
						name = v
					}
				}
				if name == "" {
					s.replyError(req.ID, -32602, "missing name")
					break
				}
				url := strings.TrimRight(s.debugHTTPURL, "/") + "/actors/lookup?name=" + name
				resp, err := http.Get(url)
				if err != nil || resp == nil || resp.Body == nil {
					s.replyError(req.ID, -32603, "failed to lookup actor id")
					break
				}
				b, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				var v any
				if err := json.Unmarshal(b, &v); err != nil {
					s.replyError(req.ID, -32603, "invalid JSON from debugHTTP")
					break
				}
				s.reply(req.ID, v)
				s.notifyShow("Actor Lookup", v)
				break
			}
			if p.Command == "orizon.getStack" {
				if s.rspAddr == "" {
					s.replyError(req.ID, -32603, "rspAddr not configured")
					break
				}
				// qXfer:stack:read::off,len
				data, err := s.rspFetchXfer("stack", "")
				if err != nil {
					s.replyError(req.ID, -32603, "failed to fetch stack")
					break
				}
				var v any
				if err := json.Unmarshal(data, &v); err != nil {
					s.replyError(req.ID, -32603, "invalid JSON from RSP")
					break
				}
				s.reply(req.ID, v)
				s.notifyShow("Stack Trace", v)
				break
			}
			if p.Command == "orizon.getActorsSnapshot" {
				if s.debugHTTPURL == "" {
					s.replyError(req.ID, -32603, "debugHTTP not configured")
					break
				}
				url := strings.TrimRight(s.debugHTTPURL, "/") + "/actors"
				resp, err := http.Get(url)
				if err != nil || resp == nil || resp.Body == nil {
					s.replyError(req.ID, -32603, "failed to fetch actors snapshot")
					break
				}
				b, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				var v any
				if err := json.Unmarshal(b, &v); err != nil {
					s.replyError(req.ID, -32603, "invalid JSON from debugHTTP")
					break
				}
				s.reply(req.ID, v)
				// Also emit a lightweight custom notification with pretty JSON for UI extensions
				s.notifyShow("Actors Snapshot", v)
				break
			}
			if p.Command == "orizon.getActorMessages" {
				if s.debugHTTPURL == "" {
					s.replyError(req.ID, -32603, "debugHTTP not configured")
					break
				}
				var aid string
				var n string
				// Expect arguments: [id, n] as numbers or strings
				if len(p.Arguments) > 0 {
					switch t := p.Arguments[0].(type) {
					case float64:
						aid = strconv.FormatUint(uint64(t), 10)
					case string:
						aid = t
					}
				}
				if aid == "" {
					s.replyError(req.ID, -32602, "missing actor id")
					break
				}
				if len(p.Arguments) > 1 {
					switch t := p.Arguments[1].(type) {
					case float64:
						n = strconv.Itoa(int(t))
					case string:
						n = t
					}
				}
				if n == "" {
					n = "100"
				}
				url := strings.TrimRight(s.debugHTTPURL, "/") + "/actors/messages?id=" + aid + "&n=" + n
				resp, err := http.Get(url)
				if err != nil || resp == nil || resp.Body == nil {
					s.replyError(req.ID, -32603, "failed to fetch actor messages")
					break
				}
				b, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				var v any
				if err := json.Unmarshal(b, &v); err != nil {
					s.replyError(req.ID, -32603, "invalid JSON from debugHTTP")
					break
				}
				s.reply(req.ID, v)
				// Show as a transient view for clients that handle custom notifications
				s.notifyShow("Actor Messages", v)
				break
			}
			if p.Command == "orizon.getActorGraph" {
				if s.debugHTTPURL == "" {
					s.replyError(req.ID, -32603, "debugHTTP not configured")
					break
				}
				qs := ""
				if len(p.Arguments) > 0 {
					if m, ok := p.Arguments[0].(map[string]any); ok && len(m) > 0 {
						first := true
						for k, vv := range m {
							val := ""
							switch t := vv.(type) {
							case string:
								val = t
							case float64:
								val = strconv.Itoa(int(t))
							}
							if val == "" {
								continue
							}
							if first {
								qs += "?"
								first = false
							} else {
								qs += "&"
							}
							qs += k + "=" + val
						}
					}
				}
				url := strings.TrimRight(s.debugHTTPURL, "/") + "/actors/graph" + qs
				resp, err := http.Get(url)
				if err != nil || resp == nil || resp.Body == nil {
					s.replyError(req.ID, -32603, "failed to fetch actor graph")
					break
				}
				b, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				var v any
				if err := json.Unmarshal(b, &v); err != nil {
					s.replyError(req.ID, -32603, "invalid JSON from debugHTTP")
					break
				}
				s.reply(req.ID, v)
				// Pretty-print JSON graph for clients wishing to render
				s.notifyShow("Actor Graph", v)
				break
			}
			if p.Command == "orizon.getDeadlocks" {
				if s.debugHTTPURL == "" {
					s.replyError(req.ID, -32603, "debugHTTP not configured")
					break
				}
				qs := ""
				if len(p.Arguments) > 0 {
					if m, ok := p.Arguments[0].(map[string]any); ok && len(m) > 0 {
						first := true
						for k, vv := range m {
							val := ""
							switch t := vv.(type) {
							case string:
								val = t
							case float64:
								val = strconv.Itoa(int(t))
							}
							if val == "" {
								continue
							}
							if first {
								qs += "?"
								first = false
							} else {
								qs += "&"
							}
							qs += k + "=" + val
						}
					}
				}
				url := strings.TrimRight(s.debugHTTPURL, "/") + "/actors/deadlocks" + qs
				resp, err := http.Get(url)
				if err != nil || resp == nil || resp.Body == nil {
					s.replyError(req.ID, -32603, "failed to fetch deadlocks")
					break
				}
				b, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				var v any
				if err := json.Unmarshal(b, &v); err != nil {
					s.replyError(req.ID, -32603, "invalid JSON from debugHTTP")
					break
				}
				s.reply(req.ID, v)
				// Notify deadlock report
				s.notifyShow("Deadlocks", v)
				break
			}
			if p.Command == "orizon.getCorrelationEvents" {
				if s.debugHTTPURL == "" {
					s.replyError(req.ID, -32603, "debugHTTP not configured")
					break
				}
				qs := ""
				// Backward compatible forms: [id, n] or [{k:v,...}]
				if len(p.Arguments) > 0 {
					switch a0 := p.Arguments[0].(type) {
					case map[string]any:
						first := true
						for k, vv := range a0 {
							val := ""
							switch t := vv.(type) {
							case string:
								val = t
							case float64:
								val = strconv.Itoa(int(t))
							}
							if val == "" {
								continue
							}
							if first {
								qs += "?"
								first = false
							} else {
								qs += "&"
							}
							qs += k + "=" + val
						}
					case string:
						cid := a0
						n := "100"
						if len(p.Arguments) > 1 {
							switch t := p.Arguments[1].(type) {
							case float64:
								n = strconv.Itoa(int(t))
							case string:
								n = t
							}
						}
						qs = "?id=" + cid + "&n=" + n
					}
				}
				if qs == "" {
					s.replyError(req.ID, -32602, "missing correlation query")
					break
				}
				url := strings.TrimRight(s.debugHTTPURL, "/") + "/actors/correlation" + qs
				resp, err := http.Get(url)
				if err != nil || resp == nil || resp.Body == nil {
					s.replyError(req.ID, -32603, "failed to fetch correlation events")
					break
				}
				b, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				var v any
				if err := json.Unmarshal(b, &v); err != nil {
					s.replyError(req.ID, -32603, "invalid JSON from debugHTTP")
					break
				}
				s.reply(req.ID, v)
				// Show correlation events as pretty JSON
				s.notifyShow("Correlation Events", v)
				break
			}
			if p.Command == "orizon.getPrettyLocals" {
				if s.rspAddr == "" {
					s.replyError(req.ID, -32603, "rspAddr not configured")
					break
				}
				data, err := s.rspFetchXfer("pretty-locals", "")
				if err != nil {
					s.replyError(req.ID, -32603, "failed to fetch pretty locals")
					break
				}
				var v any
				if err := json.Unmarshal(data, &v); err != nil {
					s.replyError(req.ID, -32603, "invalid JSON from RSP")
					break
				}
				s.reply(req.ID, v)
				s.notifyShow("Pretty Locals", v)
				break
			}
			if p.Command == "orizon.getPrettyMemory" {
				if s.rspAddr == "" {
					s.replyError(req.ID, -32603, "rspAddr not configured")
					break
				}
				if s.rspAddr == "" {
					s.replyError(req.ID, -32603, "rspAddr not configured")
					break
				}
				// args: [addrHex, len]
				var addrHex, nHex string
				if len(p.Arguments) > 0 {
					if v, ok := p.Arguments[0].(string); ok {
						addrHex = strings.TrimPrefix(strings.ToLower(v), "0x")
					}
				}
				if len(p.Arguments) > 1 {
					switch t := p.Arguments[1].(type) {
					case float64:
						nHex = strconv.FormatUint(uint64(t), 16)
					case string:
						nHex = strings.TrimPrefix(strings.ToLower(t), "0x")
					}
				}
				if addrHex == "" {
					s.replyError(req.ID, -32602, "missing addr")
					break
				}
				if nHex == "" {
					nHex = "100"
				}
				annex := "addr=" + addrHex + ",len=" + nHex
				data, err := s.rspFetchXfer("pretty-memory", annex)
				if err != nil {
					s.replyError(req.ID, -32603, "failed to fetch pretty memory")
					break
				}
				// Pretty memory is plain text
				s.reply(req.ID, string(data))
				// Stable URI per address for virtual document management
				memURI := "orizon://memory/" + addrHex + ".txt"
				s.notifyShowText("Pretty Memory 0x"+addrHex, memURI, string(data))
				break
			}
			if p.Command == "orizon.getActorMetrics" {
				if s.debugHTTPURL == "" {
					s.replyError(req.ID, -32603, "debugHTTP not configured")
					break
				}
				url := strings.TrimRight(s.debugHTTPURL, "/") + "/actors/metrics"
				resp, err := http.Get(url)
				if err != nil || resp == nil || resp.Body == nil {
					s.replyError(req.ID, -32603, "failed to fetch actor metrics")
					break
				}
				b, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				var v any
				if err := json.Unmarshal(b, &v); err != nil {
					s.replyError(req.ID, -32603, "invalid JSON from debugHTTP")
					break
				}
				s.reply(req.ID, v)
				s.notifyShow("Actor Metrics", v)
				break
			}
			if p.Command == "orizon.getIOMetrics" {
				if s.debugHTTPURL == "" {
					s.replyError(req.ID, -32603, "debugHTTP not configured")
					break
				}
				url := strings.TrimRight(s.debugHTTPURL, "/") + "/actors/io"
				resp, err := http.Get(url)
				if err != nil || resp == nil || resp.Body == nil {
					s.replyError(req.ID, -32603, "failed to fetch io metrics")
					break
				}
				b, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				var v any
				if err := json.Unmarshal(b, &v); err != nil {
					s.replyError(req.ID, -32603, "invalid JSON from debugHTTP")
					break
				}
				s.reply(req.ID, v)
				s.notifyShow("I/O Metrics", v)
				break
			}
			if p.Command == "orizon.getMailboxStats" {
				if s.debugHTTPURL == "" {
					s.replyError(req.ID, -32603, "debugHTTP not configured")
					break
				}
				var aid string
				if len(p.Arguments) > 0 {
					switch t := p.Arguments[0].(type) {
					case float64:
						aid = strconv.FormatUint(uint64(t), 10)
					case string:
						aid = t
					}
				}
				if aid == "" {
					s.replyError(req.ID, -32602, "missing actor id")
					break
				}
				url := strings.TrimRight(s.debugHTTPURL, "/") + "/actors/mailbox?id=" + aid
				resp, err := http.Get(url)
				if err != nil || resp == nil || resp.Body == nil {
					s.replyError(req.ID, -32603, "failed to fetch mailbox stats")
					break
				}
				b, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				var v any
				if err := json.Unmarshal(b, &v); err != nil {
					s.replyError(req.ID, -32603, "invalid JSON from debugHTTP")
					break
				}
				s.reply(req.ID, v)
				s.notifyShow("Mailbox Stats for "+aid, v)
				break
			}
			if p.Command == "orizon.getActorIOMetrics" {
				if s.debugHTTPURL == "" {
					s.replyError(req.ID, -32603, "debugHTTP not configured")
					break
				}
				var aid string
				if len(p.Arguments) > 0 {
					switch t := p.Arguments[0].(type) {
					case float64:
						aid = strconv.FormatUint(uint64(t), 10)
					case string:
						aid = t
					}
				}
				if aid == "" {
					s.replyError(req.ID, -32602, "missing actor id")
					break
				}
				url := strings.TrimRight(s.debugHTTPURL, "/") + "/actors/io/actor?id=" + aid
				resp, err := http.Get(url)
				if err != nil || resp == nil || resp.Body == nil {
					s.replyError(req.ID, -32603, "failed to fetch actor io metrics")
					break
				}
				b, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				var v any
				if err := json.Unmarshal(b, &v); err != nil {
					s.replyError(req.ID, -32603, "invalid JSON from debugHTTP")
					break
				}
				s.reply(req.ID, v)
				s.notifyShow("Actor I/O Metrics for "+aid, v)
				break
			}
			if p.Command == "orizon.getTopIOMetrics" {
				if s.debugHTTPURL == "" {
					s.replyError(req.ID, -32603, "debugHTTP not configured")
					break
				}
				qs := ""
				if len(p.Arguments) > 0 {
					if m, ok := p.Arguments[0].(map[string]any); ok && len(m) > 0 {
						first := true
						for k, vv := range m {
							val := ""
							switch t := vv.(type) {
							case string:
								val = t
							case float64:
								val = strconv.Itoa(int(t))
							}
							if val == "" {
								continue
							}
							if first {
								qs += "?"
								first = false
							} else {
								qs += "&"
							}
							qs += k + "=" + val
						}
					}
				}
				url := strings.TrimRight(s.debugHTTPURL, "/") + "/actors/io/top" + qs
				resp, err := http.Get(url)
				if err != nil || resp == nil || resp.Body == nil {
					s.replyError(req.ID, -32603, "failed to fetch top io metrics")
					break
				}
				b, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				var v any
				if err := json.Unmarshal(b, &v); err != nil {
					s.replyError(req.ID, -32603, "invalid JSON from debugHTTP")
					break
				}
				s.reply(req.ID, v)
				s.notifyShow("Top I/O Actors", v)
				break
			}
			s.replyError(req.ID, -32601, "unknown command")
		case "initialized":
			// This is a notification; do not send a response.
			if len(req.ID) > 0 {
				// Only reply if the client incorrectly sent an id
				s.reply(req.ID, nil)
			}
			// Start workspace indexing asynchronously to avoid blocking initialize
			if s.rootURI != "" {
				go s.indexWorkspace()
			}
		case "shutdown":
			s.reply(req.ID, nil)
		case "exit":
			return nil
		case "textDocument/didClose":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: didClose")
				break
			}
			if p.TextDocument.URI != "" {
				delete(s.docs, p.TextDocument.URI)
				delete(s.symIndex, p.TextDocument.URI)
				delete(s.docsVer, p.TextDocument.URI)
				s.removeDocFromWorkspaceIndex(p.TextDocument.URI)
				// Clear diagnostics on close
				s.notify("textDocument/publishDiagnostics", map[string]any{
					"uri":         p.TextDocument.URI,
					"diagnostics": []any{},
					"version":     0,
				})
			}
			if len(req.ID) > 0 {
				s.reply(req.ID, nil)
			}
		case "textDocument/didOpen":
			var p struct {
				TextDocument struct {
					URI  string `json:"uri"`
					Text string `json:"text"`
					Ver  int    `json:"version"`
				} `json:"textDocument"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: didOpen")
				break
			}
			if p.TextDocument.URI != "" {
				s.docs[p.TextDocument.URI] = p.TextDocument.Text
				if p.TextDocument.Ver == 0 {
					s.docsVer[p.TextDocument.URI] = 1
				} else {
					s.docsVer[p.TextDocument.URI] = p.TextDocument.Ver
				}
				// Update token cache using incremental lexer (full pass on open)
				s.updateTokensIncremental(p.TextDocument.URI, p.TextDocument.Text, nil)
				// Trigger diagnostics on open
				s.publishDiagnosticsFor(p.TextDocument.URI, p.TextDocument.Text)
				// Update workspace index
				if ast := s.astCache[p.TextDocument.URI]; ast != nil {
					s.updateWorkspaceIndex(p.TextDocument.URI, ast)
				}
			}
			if len(req.ID) > 0 {
				s.reply(req.ID, nil)
			}
		case "textDocument/didChange":
			var p struct {
				TextDocument struct {
					URI     string `json:"uri"`
					Version int    `json:"version"`
				} `json:"textDocument"`
				ContentChanges []struct {
					Text  string `json:"text"`
					Range *struct {
						Start struct{ Line, Character int } `json:"start"`
						End   struct{ Line, Character int } `json:"end"`
					} `json:"range"`
					RangeLength *int `json:"rangeLength"`
				} `json:"contentChanges"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: didChange")
				break
			}
			if uri := p.TextDocument.URI; uri != "" && len(p.ContentChanges) > 0 {
				curr := s.docs[uri]
				// Apply changes in order as specified by LSP
				var ilChanges []lexer.Change
				for _, ch := range p.ContentChanges {
					if ch.Range == nil {
						// Full document change
						ilChanges = append(ilChanges, lexer.Change{Start: 0, End: len(curr), OldText: curr, NewText: ch.Text, Type: lexer.ChangeReplaceBlock})
						curr = ch.Text
					} else {
						start := offsetFromLineCharUTF16(curr, ch.Range.Start.Line, ch.Range.Start.Character)
						end := offsetFromLineCharUTF16(curr, ch.Range.End.Line, ch.Range.End.Character)
						if start < 0 || end < 0 || start > len(curr) || end > len(curr) || start > end {
							// Ignore invalid range; continue
						} else {
							oldText := curr[start:end]
							curr = curr[:start] + ch.Text + curr[end:]
							ilChanges = append(ilChanges, lexer.Change{Start: start, End: end, OldText: oldText, NewText: ch.Text, Type: lexer.ChangeReplaceBlock})
						}
					}
				}
				s.docs[uri] = curr
				if p.TextDocument.Version != 0 {
					s.docsVer[uri] = p.TextDocument.Version
				} else {
					s.docsVer[uri] = s.docsVer[uri] + 1
				}
				// Update token cache incrementally
				s.updateTokensIncremental(uri, curr, ilChanges)
				// Trigger diagnostics on change
				s.publishDiagnosticsFor(uri, curr)
				if ast := s.astCache[uri]; ast != nil {
					s.updateWorkspaceIndex(uri, ast)
				}
			}
			if len(req.ID) > 0 {
				s.reply(req.ID, nil)
			}
		case "textDocument/hover":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				} `json:"position"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: hover")
				break
			}
			result := s.handleHover(p.TextDocument.URI, p.Position.Line, p.Position.Character)
			s.reply(req.ID, result)
		case "textDocument/definition":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position struct{ Line, Character int } `json:"position"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: definition")
				break
			}
			defs := s.handleDefinition(p.TextDocument.URI, p.Position.Line, p.Position.Character)
			s.reply(req.ID, defs)
		case "textDocument/typeDefinition":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position struct{ Line, Character int } `json:"position"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: typeDefinition")
				break
			}
			defs := s.handleTypeDefinition(p.TextDocument.URI, p.Position.Line, p.Position.Character)
			s.reply(req.ID, defs)
		case "textDocument/documentSymbol":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: documentSymbol")
				break
			}
			syms := s.handleDocumentSymbol(p.TextDocument.URI)
			s.reply(req.ID, syms)
		case "textDocument/references":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position struct{ Line, Character int } `json:"position"`
				Context  struct {
					IncludeDeclaration bool `json:"includeDeclaration"`
				} `json:"context"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: references")
				break
			}
			refs := s.handleReferences(p.TextDocument.URI, p.Position.Line, p.Position.Character, p.Context.IncludeDeclaration)
			// Also search in other open documents for the same identifier
			cross := s.handleCrossFileReferences(p.TextDocument.URI, p.Position.Line, p.Position.Character)
			refs = append(refs, cross...)
			s.reply(req.ID, refs)
		case "textDocument/documentHighlight":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position struct{ Line, Character int } `json:"position"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: documentHighlight")
				break
			}
			hl := s.handleDocumentHighlight(p.TextDocument.URI, p.Position.Line, p.Position.Character)
			s.reply(req.ID, hl)
		case "textDocument/foldingRange":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: foldingRange")
				break
			}
			fr := s.handleFoldingRange(p.TextDocument.URI)
			s.reply(req.ID, fr)
		case "textDocument/prepareRename":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position struct{ Line, Character int } `json:"position"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: prepareRename")
				break
			}
			prep := s.handlePrepareRename(p.TextDocument.URI, p.Position.Line, p.Position.Character)
			s.reply(req.ID, prep)
		case "textDocument/rename":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position struct{ Line, Character int } `json:"position"`
				NewName  string                        `json:"newName"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: rename")
				break
			}
			edit, errMsg := s.handleRename(p.TextDocument.URI, p.Position.Line, p.Position.Character, p.NewName)
			if errMsg != "" {
				s.replyError(req.ID, -32602, errMsg)
			} else {
				s.reply(req.ID, edit)
			}
		case "workspace/symbol":
			var p struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: workspace/symbol")
				break
			}
			syms := s.handleWorkspaceSymbol(p.Query)
			s.reply(req.ID, syms)
		case "textDocument/signatureHelp":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position struct{ Line, Character int } `json:"position"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: signatureHelp")
				break
			}
			help := s.handleSignatureHelp(p.TextDocument.URI, p.Position.Line, p.Position.Character)
			s.reply(req.ID, help)
		case "textDocument/onTypeFormatting":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position struct{ Line, Character int } `json:"position"`
				Ch       string                        `json:"ch"`
				Options  map[string]any                `json:"options"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: onTypeFormatting")
				break
			}
			edits := s.handleOnTypeFormatting(p.TextDocument.URI, p.Position.Line, p.Position.Character, p.Ch, p.Options)
			s.reply(req.ID, edits)
		case "textDocument/formatting":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Options map[string]any `json:"options"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: formatting")
				break
			}
			edits := s.handleFormatting(p.TextDocument.URI, p.Options)
			s.reply(req.ID, edits)
		case "textDocument/semanticTokens/full":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: semanticTokens/full")
				break
			}
			res := s.handleSemanticTokensFull(p.TextDocument.URI)
			s.reply(req.ID, res)
		case "textDocument/inlayHint":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Range struct {
					Start struct{ Line, Character int } `json:"start"`
					End   struct{ Line, Character int } `json:"end"`
				} `json:"range"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: inlayHint")
				break
			}
			hints := s.handleInlayHints(p.TextDocument.URI, p.Range.Start.Line, p.Range.End.Line)
			s.reply(req.ID, hints)
		case "textDocument/semanticTokens/range":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Range struct{ Start, End struct{ Line, Character int } } `json:"range"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: semanticTokens/range")
				break
			}
			res := s.handleSemanticTokensRange(p.TextDocument.URI, p.Range.Start.Line, p.Range.End.Line)
			s.reply(req.ID, res)
		case "textDocument/rangeFormatting":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Range struct {
					Start struct{ Line, Character int } `json:"start"`
					End   struct{ Line, Character int } `json:"end"`
				} `json:"range"`
				Options map[string]any `json:"options"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: rangeFormatting")
				break
			}
			edits := s.handleRangeFormatting(p.TextDocument.URI, p.Range.Start.Line, p.Range.End.Line, p.Options)
			s.reply(req.ID, edits)
		case "textDocument/codeAction":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Range struct {
					Start struct{ Line, Character int } `json:"start"`
					End   struct{ Line, Character int } `json:"end"`
				} `json:"range"`
				Context struct {
					Diagnostics []any `json:"diagnostics"`
				} `json:"context"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.replyError(req.ID, -32602, "invalid params: codeAction")
				break
			}
			actions := s.handleCodeAction(p.TextDocument.URI, p.Range.Start.Line, p.Range.Start.Character, p.Range.End.Line, p.Range.End.Character)
			s.reply(req.ID, actions)
		case "textDocument/completion":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position struct{ Line, Character int } `json:"position"`
				Context  any                           `json:"context"`
			}
			_ = json.Unmarshal(req.Params, &p)
			items := s.handleCompletion(p.TextDocument.URI, p.Position.Line, p.Position.Character)
			s.reply(req.ID, map[string]any{"isIncomplete": false, "items": items})
		case "completionItem/resolve":
			// Enrich completion item with documentation/detail on demand
			var item map[string]any
			if err := json.Unmarshal(req.Params, &item); err != nil {
				s.replyError(req.ID, -32602, "invalid params: completionItem/resolve")
				break
			}
			if item != nil {
				if dataAny, ok := item["data"].(map[string]any); ok {
					// Keyword metadata enrichment
					if kind, _ := dataAny["type"].(string); kind == "keyword" {
						if label, _ := item["label"].(string); label != "" {
							if meta, ok := keywordMeta[label]; ok {
								if meta.Detail != "" {
									item["detail"] = meta.Detail
								}
								if meta.Documentation != "" {
									item["documentation"] = meta.Documentation
								}
							}
						}
					}
					// Symbol metadata enrichment using AST index
					if kind, _ := dataAny["type"].(string); kind == "symbol" {
						label, _ := item["label"].(string)
						uri, _ := dataAny["uri"].(string)
						if uri == "" {
							// Try first entry from uris array if provided
							if arr, okArr := dataAny["uris"].([]any); okArr && len(arr) > 0 {
								if s0, okS := arr[0].(string); okS {
									uri = s0
								}
							}
						}
						if label != "" && uri != "" {
							// Ensure AST and symbol index are available
							text := s.docs[uri]
							ast := s.astCache[uri]
							if ast == nil && text != "" {
								lx := lexer.NewWithFilename(text, uri)
								pr := parser.NewParser(lx, uri)
								a, _ := pr.Parse()
								ast = a
								s.astCache[uri] = ast
								if ast != nil {
									s.buildSymbolIndex(uri, ast)
								}
							}
							// Build detail and documentation best-effort
							if entries := s.symIndex[uri][label]; len(entries) > 0 {
								// Prefer existing detail (signature/type), but ensure it is present
								if d := entries[0].Detail; d != "" {
									item["detail"] = d
								}
								// Try to attach structured documentation
								doc := s.buildSymbolDocumentation(uri, label, entries[0])
								if doc != "" {
									item["documentation"] = doc
								}
							}
						}
					}
				}
			}
			s.reply(req.ID, item)
		default:
			s.replyError(req.ID, -32601, "Method not found")
		}
	}
}

func (s *Server) reply(id json.RawMessage, result any) {
	resp := Response{JSONRPC: "2.0", ID: id, Result: result}
	s.write(resp)
}

func (s *Server) replyError(id json.RawMessage, code int, msg string) {
	resp := Response{JSONRPC: "2.0", ID: id, Error: &RespError{Code: code, Message: msg}}
	s.write(resp)
	atomic.AddUint64(&s.errCount, 1)
}

func (s *Server) write(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	if _, err := io.WriteString(s.out, "Content-Length: "); err != nil {
		return
	}
	if _, err := io.WriteString(s.out, itoa(len(data))); err != nil {
		return
	}
	if _, err := io.WriteString(s.out, "\r\n\r\n"); err != nil {
		return
	}
	if n, err2 := s.out.Write(data); err2 == nil && n > 0 {
		atomic.AddUint64(&s.bytesOut, uint64(n))
	}
}

// Stats returns a snapshot of server counters.
func (s *Server) Stats() map[string]uint64 {
	return map[string]uint64{
		"requests": atomic.LoadUint64(&s.reqCount),
		"errors":   atomic.LoadUint64(&s.errCount),
		"bytesOut": atomic.LoadUint64(&s.bytesOut),
	}
}

// filePathFromURI converts file:// URI to OS path best-effort
func filePathFromURI(uri string) string {
	if strings.HasPrefix(uri, "file://") {
		p := strings.TrimPrefix(uri, "file://")
		// On Windows, leading '/' may appear before drive letter
		if len(p) >= 3 && p[0] == '/' && ((p[1] >= 'A' && p[1] <= 'Z') || (p[1] >= 'a' && p[1] <= 'z')) && p[2] == ':' {
			p = p[1:]
		}
		return p
	}
	return ""
}

// pathUnderRoot returns true if filePath is within the current rootURI directory.
// When rootURI is empty, it returns true to avoid over-restricting single-file usage.
func (s *Server) pathUnderRoot(filePath string) bool {
	root := filePathFromURI(s.rootURI)
	if root == "" || filePath == "" {
		return true
	}
	// Normalize paths for comparison
	rp := filepath.Clean(root)
	fp := filepath.Clean(filePath)
	// On Windows, make comparison case-insensitive by lowering both
	rl := strings.ToLower(rp)
	fl := strings.ToLower(fp)
	// Ensure trailing separator for root prefix matching
	if !strings.HasSuffix(rl, string(filepath.Separator)) {
		rl += string(filepath.Separator)
	}
	if !strings.HasSuffix(fl, string(filepath.Separator)) {
		// fl may point to a file; keep as-is for prefix check
	}
	return strings.HasPrefix(fl, rl)
}

// indexWorkspace scans rootURI for .oriz files and indexes top-level symbols (best-effort, blocking)
func (s *Server) indexWorkspace() {
	root := filePathFromURI(s.rootURI)
	if root == "" {
		return
	}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip unreadable paths but continue walking
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".oriz") {
			return nil
		}
		// Only index files strictly under root to avoid symlink escapes
		if !s.pathUnderRoot(path) {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		uri := "file://" + path
		s.docs[uri] = string(b)
		lx := lexer.NewWithFilename(string(b), uri)
		pr := parser.NewParser(lx, uri)
		ast, _ := pr.Parse()
		if ast != nil {
			s.astCache[uri] = ast
			s.buildSymbolIndex(uri, ast)
			s.updateWorkspaceIndex(uri, ast)
		}
		return nil
	})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// Main entry for stdio mode
func RunStdio() error {
	srv := NewServer(os.Stdin, os.Stdout)
	return srv.Run()
}

// ===== Helpers for LSP notifications and features =====

// notify sends a JSON-RPC notification (method with params, no id).
func (s *Server) notify(method string, params any) {
	// Build a raw notification object
	msg := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	s.write(msg)
}

// notifyShow sends a custom notification that conveys a titled JSON payload to display.
// Clients can optionally render this as a virtual document or panel. The method name uses
// a $/-prefixed custom channel to avoid conflicts with standard LSP.
func (s *Server) notifyShow(title string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return
	}
	uri := "orizon://debug/" + sanitizeTitle(title) + ".json"
	body := string(b)
	var diff string
	if prev, ok := s.vdocCache[uri]; ok {
		if prev == body {
			// Skip identical content to avoid flicker and redundant traffic
			return
		}
		diff = computeUnifiedDiff(prev, body)
	}
	s.vdocCache[uri] = body
	payload := map[string]any{
		"title":   title,
		"mime":    "application/json",
		"content": body,
		// Suggested file-like name for clients that open virtual documents
		"uriHint": uri,
	}
	if diff != "" {
		payload["diff"] = diff
		payload["diffMime"] = "text/x-diff"
	}
	s.notify("$/orizon.show", payload)
	// Ask the client to open the virtual document the first time we publish it
	if !s.vdocOpened[uri] {
		s.requestShowDocument(uri)
		s.vdocOpened[uri] = true
	}
}

// notifyShowText is a specialized variant to push plain text with explicit uriHint
func (s *Server) notifyShowText(title, uri, text string) {
	var diff string
	if prev, ok := s.vdocCache[uri]; ok {
		if prev == text {
			return
		}
		diff = computeUnifiedDiff(prev, text)
	}
	s.vdocCache[uri] = text
	payload := map[string]any{
		"title":   title,
		"mime":    "text/plain",
		"content": text,
		"uriHint": uri,
	}
	if diff != "" {
		payload["diff"] = diff
		payload["diffMime"] = "text/x-diff"
	}
	s.notify("$/orizon.show", payload)
	if !s.vdocOpened[uri] {
		s.requestShowDocument(uri)
		s.vdocOpened[uri] = true
	}
}

// computeUnifiedDiff generates a simple unified diff between old and new text.
// It is line-based and optimized for clarity rather than minimal edits.
func computeUnifiedDiff(oldText, newText string) string {
	if oldText == newText {
		return ""
	}
	a := splitLines(oldText)
	b := splitLines(newText)
	var buf bytes.Buffer
	buf.WriteString("--- a\n")
	buf.WriteString("+++ b\n")
	// Simple two-pointer scan with fallback additions/removals
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case i < len(a) && j < len(b) && a[i] == b[j]:
			// context line (optional)
			buf.WriteString(" " + a[i] + "\n")
			i++
			j++
		case j < len(b) && (i >= len(a) || !containsAhead(a, b[j], i+1)):
			buf.WriteString("+" + b[j] + "\n")
			j++
		case i < len(a):
			buf.WriteString("-" + a[i] + "\n")
			i++
		default:
			j++
		}
	}
	return buf.String()
}

func splitLines(s string) []string {
	lines := strings.Split(s, "\n")
	// trim trailing CR if present per line (robust for Windows clients)
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	// drop possible last empty line if both end with newline to reduce noise
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func containsAhead(arr []string, target string, start int) bool {
	for k := start; k < len(arr) && k < start+16; k++ { // look ahead limited window
		if arr[k] == target {
			return true
		}
	}
	return false
}

// requestShowDocument sends window/showDocument to hint the client to open a tab for our virtual doc
func (s *Server) requestShowDocument(uri string) {
	id := atomic.AddUint64(&s.rpcID, 1)
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "window/showDocument",
		"params": map[string]any{
			"uri":       uri,
			"external":  false,
			"takeFocus": true,
		},
	}
	s.write(req)
}

func sanitizeTitle(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch r {
		case ' ', '\\', '/', ':', '?', '*', '"', '<', '>', '|':
			out = append(out, '-')
		default:
			if r < 32 {
				continue
			}
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "untitled"
	}
	return string(out)
}

// rspFetchXfer connects to the configured RSP server and performs a qXfer read for the given stream.
// It returns the fully reassembled payload bytes (already hex-decoded for 'm'/'l' chunks).
func (s *Server) rspFetchXfer(stream string, annex string) ([]byte, error) {
	// open TCP
	conn, err := net.Dial("tcp", s.rspAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	// helper to write an RSP packet
	writePkt := func(p string) error {
		sum := byte(0)
		for i := 0; i < len(p); i++ {
			sum += p[i]
		}
		frame := fmt.Sprintf("$%s#%02x", p, sum)
		_, err := conn.Write([]byte(frame))
		return err
	}
	// read one RSP reply payload (ignore '+')
	readPayload := func() (string, error) {
		r := bufio.NewReader(conn)
		// optional ack
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		if b != '+' {
			_ = r.UnreadByte()
		}
		// '$'
		for {
			ch, er := r.ReadByte()
			if er != nil {
				return "", er
			}
			if ch == '$' {
				break
			}
		}
		var data []byte
		for {
			ch, er := r.ReadByte()
			if er != nil {
				return "", er
			}
			if ch == '#' {
				break
			}
			data = append(data, ch)
		}
		// checksum
		_, _ = r.ReadByte()
		_, _ = r.ReadByte()
		return string(data), nil
	}
	// Enter no-ack mode
	_ = writePkt("QStartNoAckMode")
	_, _ = readPayload()
	// qXfer loop: read in chunks
	var out []byte
	off := 0
	for {
		tail := fmt.Sprintf("%x,%x", off, 0x400)
		q := "qXfer:" + stream + ":read:"
		if annex != "" {
			q += annex + ":"
		}
		q += tail
		if err := writePkt(q); err != nil {
			return nil, err
		}
		payload, err := readPayload()
		if err != nil {
			return nil, err
		}
		if len(payload) == 0 {
			break
		}
		marker := payload[0]
		chunkHex := payload[1:]
		chunk, er := hex.DecodeString(chunkHex)
		if er != nil {
			return nil, er
		}
		out = append(out, chunk...)
		off += len(chunk)
		if marker == 'l' {
			break
		}
	}
	return out, nil
}

// publishDiagnosticsFor parses the document and publishes diagnostics.
func (s *Server) publishDiagnosticsFor(uri, text string) {
	// Attempt to parse using the compiler frontend to collect syntax errors
	lx := lexer.NewWithFilename(text, uri)
	pr := parser.NewParser(lx, uri)
	ast, errs := pr.Parse()

	// Convert parse errors into LSP diagnostics
	diags := make([]map[string]any, 0, len(errs))
	for _, e := range errs {
		// Defaults
		line0 := 0
		char0 := 0
		endLine := 0
		endChar := 0
		msg := e.Error()

		if pe, ok := e.(*parser.ParseError); ok {
			// Prefer byte offset if available for accurate UTF-16 mapping
			if pe.Position.Offset >= 0 {
				l, c := utf16LineCharFromOffset(text, pe.Position.Offset)
				line0, char0 = l, c
				endLine, endChar = l, c+1
			} else {
				// Fallback: convert 1-based columns to 0-based without UTF-16 correction
				if pe.Position.Line > 0 {
					line0 = pe.Position.Line - 1
				}
				if pe.Position.Column > 0 {
					char0 = pe.Position.Column - 1
				}
				endLine, endChar = line0, char0+1
			}
			if pe.Message != "" {
				msg = pe.Message
			}
		} else {
			endLine, endChar = line0, char0+1
		}

		diag := map[string]any{
			"range": map[string]any{
				"start": map[string]any{"line": line0, "character": char0},
				"end":   map[string]any{"line": endLine, "character": endChar},
			},
			"severity": 1,
			"source":   "orizon-lsp",
			"message":  msg,
		}
		diags = append(diags, diag)
	}

	// If parse succeeded, also run light AST validation to surface structural issues as warnings
	if ast != nil {
		// Build symbol index for definition/hover enhancements
		s.buildSymbolIndex(uri, ast)
		s.astCache[uri] = ast
		if verrs, vwarns := parser.CollectValidationReports(ast); len(verrs) > 0 || len(vwarns) > 0 {
			// Append warnings first to keep severity ordering consistent (errors already present)
			for _, w := range vwarns {
				l0, c0 := utf16LineCharFromOffset(text, w.Span.Start.Offset)
				l1, c1 := utf16LineCharFromOffset(text, w.Span.End.Offset)
				diags = append(diags, map[string]any{
					"range": map[string]any{
						"start": map[string]any{"line": l0, "character": c0},
						"end":   map[string]any{"line": l1, "character": c1},
					},
					"severity": 2, // Warning
					"source":   "orizon-lsp",
					"message":  w.Message,
				})
			}
			for _, we := range verrs {
				l0, c0 := utf16LineCharFromOffset(text, we.Span.Start.Offset)
				l1, c1 := utf16LineCharFromOffset(text, we.Span.End.Offset)
				diags = append(diags, map[string]any{
					"range": map[string]any{
						"start": map[string]any{"line": l0, "character": c0},
						"end":   map[string]any{"line": l1, "character": c1},
					},
					"severity": 1, // Error
					"source":   "orizon-lsp",
					"message":  we.Message,
				})
			}
		}
	}

	ver := s.docsVer[uri]
	payload := map[string]any{
		"uri":         uri,
		"diagnostics": diags,
		"version":     ver,
	}
	// Emit only if changed
	if b, err := json.Marshal(payload); err == nil {
		hash := string(b)
		if s.diagCache[uri] == hash {
			return
		}
		s.diagCache[uri] = hash
	}
	s.notify("textDocument/publishDiagnostics", payload)
}

// updateTokensIncremental updates cached tokens using the incremental lexer.
// When changes is nil, a full pass is performed.
func (s *Server) updateTokensIncremental(uri, text string, changes []lexer.Change) {
	if s.incLexer == nil {
		return
	}
	// For now, call LexIncremental; the API accepts a list of changes but current
	// implementation may choose to re-lex fully based on hashing.
	toks, err := s.incLexer.LexIncremental(uri, []byte(text), changes)
	if err != nil {
		return
	}
	s.tokCache[uri] = toks
}

// handleHover returns minimal hover info at the given position.
func (s *Server) handleHover(uri string, line, character int) map[string]any {
	text := s.docs[uri]
	if text == "" {
		return nil
	}

	// Compute byte offset from UTF-16 line/character per LSP spec
	offset := offsetFromLineCharUTF16(text, line, character)
	if offset < 0 || offset > len(text) {
		return nil
	}

	// Lex tokens and find the one containing the offset
	lx := lexer.NewWithFilename(text, uri)
	var tok lexer.Token
	var found *lexer.Token
	for {
		tok = lx.NextToken()
		if tok.Type == lexer.TokenEOF {
			break
		}
		// Normalize span filename and check containment by byte offsets
		start := tok.Span.Start.Offset
		end := tok.Span.End.Offset
		if start <= offset && offset < end {
			t := tok // capture
			found = &t
			break
		}
	}

	if found == nil {
		return nil
	}

	// Build hover contents: try symbol info first, then fallback to token
	label := strings.TrimSpace(found.Literal)
	if label == "" {
		label = found.Type.String()
	}
	hoverVal := ""
	if infos, ok := s.symIndex[uri][label]; ok && len(infos) > 0 {
		// Choose the first definition entry
		info := infos[0]
		sig := info.Detail
		if sig == "" {
			sig = label
		}
		hoverVal = "```orizon\n" + sig + "\n```"
	} else {
		hoverVal = "```orizon\n" + label + "\n```\n\nToken: " + found.Type.String()
	}
	// Append simple AST-based description and type summary when available
	if ast := s.astCache[uri]; ast != nil {
		if desc := describeNodeAt(ast, offset); desc != "" {
			hoverVal += "\n\n" + desc
		}
		if tsum := typeSummaryAt(ast, offset); tsum != "" {
			hoverVal += "\n\n" + tsum
		}
	}
	// Append a lightweight type hint if derivable
	if t := s.typeHintAt(uri, offset); t != "" {
		hoverVal += "\n\ntype: " + t
	}
	contents := map[string]any{"kind": "markdown", "value": hoverVal}

	// Range as LSP range (0-based)
	// Convert token byte offsets to LSP UTF-16 positions for accurate range
	startLine, startChar := utf16LineCharFromOffset(text, found.Span.Start.Offset)
	endLine, endChar := utf16LineCharFromOffset(text, found.Span.End.Offset)

	rng := map[string]any{
		"start": map[string]any{"line": startLine, "character": startChar},
		"end":   map[string]any{"line": endLine, "character": endChar},
	}

	return map[string]any{
		"contents": contents,
		"range":    rng,
	}
}

// Signature/Detail builders
func buildFuncSignature(fn *parser.FunctionDeclaration) string {
	name := "<anonymous>"
	if fn.Name != nil {
		name = fn.Name.Value
	}
	// Build parameter list string
	params := make([]string, 0, len(fn.Parameters))
	for _, p := range fn.Parameters {
		if p == nil || p.Name == nil {
			continue
		}
		entry := p.Name.Value
		if p.TypeSpec != nil {
			entry = entry + ": " + p.TypeSpec.String()
		}
		params = append(params, entry)
	}
	ret := ""
	if fn.ReturnType != nil {
		ret = " -> " + fn.ReturnType.String()
	}
	return "func " + name + "(" + strings.Join(params, ", ") + ")" + ret
}

func buildParamDetail(p *parser.Parameter) string {
	if p == nil || p.Name == nil {
		return "param"
	}
	if p.TypeSpec != nil {
		return p.Name.Value + ": " + p.TypeSpec.String()
	}
	return p.Name.Value
}

func buildVarDetail(v *parser.VariableDeclaration) string {
	if v == nil || v.Name == nil {
		return "var"
	}
	if v.TypeSpec != nil {
		return v.Name.Value + ": " + v.TypeSpec.String()
	}
	return v.Name.Value
}

// buildSymbolDocumentation builds a concise documentation string for a symbol using AST information when available.
// It focuses on function signatures (parameters and return type) and variable type information.
func (s *Server) buildSymbolDocumentation(uri, name string, si SymbolInfo) string {
	// Prefer AST cache; if missing, try to parse the current text
	ast := s.astCache[uri]
	if ast == nil {
		text := s.docs[uri]
		if text != "" {
			lx := lexer.NewWithFilename(text, uri)
			pr := parser.NewParser(lx, uri)
			a, _ := pr.Parse()
			ast = a
			s.astCache[uri] = ast
			if ast != nil {
				s.buildSymbolIndex(uri, ast)
			}
		}
	}
	if ast == nil {
		// Fallback to SymbolInfo.Detail if available
		return si.Detail
	}
	// Function documentation
	if fn := findFunctionDeclByName(ast, name); fn != nil {
		// Build documentation lines
		lines := make([]string, 0, 4)
		// Parameters
		if len(fn.Parameters) > 0 {
			params := make([]string, 0, len(fn.Parameters))
			for _, p := range fn.Parameters {
				if p == nil || p.Name == nil {
					continue
				}
				if p.TypeSpec != nil {
					params = append(params, p.Name.Value+": "+p.TypeSpec.String())
				} else {
					params = append(params, p.Name.Value)
				}
			}
			if len(params) > 0 {
				lines = append(lines, "parameters: "+strings.Join(params, ", "))
			}
		}
		// Return type
		if fn.ReturnType != nil {
			lines = append(lines, "returns: "+fn.ReturnType.String())
		}
		return strings.Join(lines, "\n")
	}
	// Variable documentation
	if v := findVariableDeclByName(ast, name); v != nil {
		if v.TypeSpec != nil {
			return "type: " + v.TypeSpec.String()
		}
	}
	return si.Detail
}

// describeNodeAt returns a human-readable description of the most specific AST node covering the given offset.
func describeNodeAt(root *parser.Program, offset int) string {
	var best parser.Node
	bestLen := int(^uint(0) >> 1)

	var walk func(n parser.Node)
	walk = func(n parser.Node) {
		if n == nil {
			return
		}
		sp := n.GetSpan()
		if sp.Start.Offset <= offset && offset < sp.End.Offset {
			spanLen := sp.End.Offset - sp.Start.Offset
			if spanLen >= 0 && spanLen < bestLen {
				best = n
				bestLen = spanLen
			}
			switch node := n.(type) {
			case *parser.Program:
				for _, d := range node.Declarations {
					walk(d)
				}
			case *parser.FunctionDeclaration:
				for _, p := range node.Parameters {
					if p != nil {
						walk(p)
					}
				}
				if node.ReturnType != nil {
					if tn, ok := node.ReturnType.(parser.Node); ok {
						walk(tn)
					}
				}
				if node.Body != nil {
					walk(node.Body)
				}
			case *parser.BlockStatement:
				for _, st := range node.Statements {
					walk(st)
				}
			case *parser.ExpressionStatement:
				if node.Expression != nil {
					walk(node.Expression)
				}
			case *parser.IfStatement:
				if node.Condition != nil {
					walk(node.Condition)
				}
				if node.ThenStmt != nil {
					walk(node.ThenStmt)
				}
				if node.ElseStmt != nil {
					walk(node.ElseStmt)
				}
			case *parser.WhileStatement:
				if node.Condition != nil {
					walk(node.Condition)
				}
				if node.Body != nil {
					walk(node.Body)
				}
			case *parser.VariableDeclaration:
				if node.TypeSpec != nil {
					if tn, ok := node.TypeSpec.(parser.Node); ok {
						walk(tn)
					}
				}
				if node.Initializer != nil {
					walk(node.Initializer)
				}
			case *parser.BinaryExpression:
				if node.Left != nil {
					walk(node.Left)
				}
				if node.Right != nil {
					walk(node.Right)
				}
			case *parser.UnaryExpression:
				if node.Operand != nil {
					walk(node.Operand)
				}
			case *parser.CallExpression:
				if node.Function != nil {
					walk(node.Function)
				}
				for _, a := range node.Arguments {
					walk(a)
				}
			}
		}
	}
	walk(root)

	if best == nil {
		return ""
	}
	switch n := best.(type) {
	case *parser.Literal:
		switch n.Kind {
		case parser.LiteralInteger:
			return "type: Int"
		case parser.LiteralFloat:
			return "type: Float"
		case parser.LiteralString:
			return "type: String"
		case parser.LiteralBool:
			return "type: Bool"
		default:
			return "literal"
		}
	case *parser.Identifier:
		return "identifier: " + n.Value
	case *parser.CallExpression:
		return "call expression"
	case *parser.FunctionDeclaration:
		name := "<anonymous>"
		if n.Name != nil {
			name = n.Name.Value
		}
		return "function: " + name
	case *parser.VariableDeclaration:
		if n.Name != nil {
			return "variable: " + n.Name.Value
		}
		return "variable"
	default:
		return ""
	}
}

// typeHintAt returns a minimal type hint string for the node at the given offset, when derivable.
func (s *Server) typeHintAt(uri string, offset int) string {
	ast := s.astCache[uri]
	if ast == nil {
		return ""
	}
	n := findNodeAt(ast, offset)
	if n == nil {
		return ""
	}
	switch node := n.(type) {
	case *parser.Literal:
		switch node.Kind {
		case parser.LiteralInteger:
			return "Int"
		case parser.LiteralFloat:
			return "Float"
		case parser.LiteralString:
			return "String"
		case parser.LiteralBool:
			return "Bool"
		default:
			return ""
		}
	case *parser.Identifier:
		if node.Value == "" {
			return ""
		}
		if ty := s.typeFromSymbolDetail(uri, node.Value); ty != "" {
			return ty
		}
		return ""
	case *parser.CallExpression:
		// If the callee is an identifier, try to get function return type
		if id, ok := node.Function.(*parser.Identifier); ok && id != nil {
			if ret := s.funcReturnTypeFromIndex(uri, id.Value); ret != "" {
				return ret
			}
		}
		return ""
	case *parser.BinaryExpression:
		// Heuristic: if both sides are numeric literals, hint numeric type
		if l, ok := node.Left.(*parser.Literal); ok {
			if r, ok2 := node.Right.(*parser.Literal); ok2 {
				if l.Kind == parser.LiteralInteger && r.Kind == parser.LiteralInteger {
					return "Int"
				}
				if (l.Kind == parser.LiteralFloat || l.Kind == parser.LiteralInteger) &&
					(r.Kind == parser.LiteralFloat || r.Kind == parser.LiteralInteger) {
					return "Float"
				}
			}
		}
		return ""
	default:
		return ""
	}
}

// typeSummaryAt returns a brief, human-readable type summary for the AST node at offset.
// It uses parser type nodes' String methods and augments with field/param overviews when available.
func typeSummaryAt(root *parser.Program, offset int) string {
	n := findNodeAt(root, offset)
	if n == nil {
		return ""
	}
	switch t := n.(type) {
	case *parser.StructType:
		name := "struct"
		if t.Name != nil && t.Name.Value != "" {
			name = "struct " + t.Name.Value
		}
		// Show up to a few fields
		parts := make([]string, 0, len(t.Fields))
		for i, f := range t.Fields {
			if f == nil || f.Name == nil || f.Type == nil {
				continue
			}
			parts = append(parts, f.Name.Value+": "+f.Type.String())
			if i >= 4 {
				break
			}
		}
		if len(parts) > 0 {
			return name + " { " + strings.Join(parts, ", ") + " }"
		}
		return name
	case *parser.ArrayType:
		return "array of " + t.ElementType.String()
	case *parser.FunctionType:
		// Build concise signature
		params := make([]string, 0, len(t.Parameters))
		for _, p := range t.Parameters {
			if p == nil || p.Type == nil {
				continue
			}
			if p.Name != "" {
				params = append(params, p.Name+": "+p.Type.String())
			} else {
				params = append(params, p.Type.String())
			}
		}
		ret := "void"
		if t.ReturnType != nil {
			ret = t.ReturnType.String()
		}
		return "fn(" + strings.Join(params, ", ") + ") -> " + ret
	default:
		return ""
	}
}

func (s *Server) typeFromSymbolDetail(uri, name string) string {
	entries := s.symIndex[uri][name]
	for _, si := range entries {
		// Variable or parameter detail format: "name: Type"
		if si.Kind == 13 && si.Detail != "" {
			if idx := strings.LastIndex(si.Detail, ":"); idx >= 0 && idx+1 < len(si.Detail) {
				return strings.TrimSpace(si.Detail[idx+1:])
			}
		}
	}
	return ""
}

func (s *Server) funcReturnTypeFromIndex(uri, name string) string {
	entries := s.symIndex[uri][name]
	for _, si := range entries {
		// Function detail format: "func name(params) -> Ret"
		if si.Kind == 12 && si.Detail != "" {
			if idx := strings.LastIndex(si.Detail, "->"); idx >= 0 && idx+2 <= len(si.Detail) {
				return strings.TrimSpace(si.Detail[idx+2:])
			}
		}
	}
	return ""
}

// findNodeAt returns the most specific parser.Node that contains the given offset.
func findNodeAt(root *parser.Program, offset int) parser.Node {
	var best parser.Node
	bestLen := int(^uint(0) >> 1)
	var walk func(n parser.Node)
	walk = func(n parser.Node) {
		if n == nil {
			return
		}
		sp := n.GetSpan()
		if sp.Start.Offset <= offset && offset < sp.End.Offset {
			spanLen := sp.End.Offset - sp.Start.Offset
			if spanLen >= 0 && spanLen < bestLen {
				best = n
				bestLen = spanLen
			}
			switch node := n.(type) {
			case *parser.Program:
				for _, d := range node.Declarations {
					walk(d)
				}
			case *parser.FunctionDeclaration:
				for _, p := range node.Parameters {
					if p != nil {
						walk(p)
					}
				}
				if node.ReturnType != nil {
					if tn, ok := node.ReturnType.(parser.Node); ok {
						walk(tn)
					}
				}
				if node.Body != nil {
					walk(node.Body)
				}
			case *parser.BlockStatement:
				for _, st := range node.Statements {
					walk(st)
				}
			case *parser.ExpressionStatement:
				if node.Expression != nil {
					walk(node.Expression)
				}
			case *parser.IfStatement:
				if node.Condition != nil {
					walk(node.Condition)
				}
				if node.ThenStmt != nil {
					walk(node.ThenStmt)
				}
				if node.ElseStmt != nil {
					walk(node.ElseStmt)
				}
			case *parser.WhileStatement:
				if node.Condition != nil {
					walk(node.Condition)
				}
				if node.Body != nil {
					walk(node.Body)
				}
			case *parser.VariableDeclaration:
				if node.TypeSpec != nil {
					if tn, ok := node.TypeSpec.(parser.Node); ok {
						walk(tn)
					}
				}
				if node.Initializer != nil {
					walk(node.Initializer)
				}
			case *parser.BinaryExpression:
				if node.Left != nil {
					walk(node.Left)
				}
				if node.Right != nil {
					walk(node.Right)
				}
			case *parser.UnaryExpression:
				if node.Operand != nil {
					walk(node.Operand)
				}
			case *parser.CallExpression:
				if node.Function != nil {
					walk(node.Function)
				}
				for _, a := range node.Arguments {
					walk(a)
				}
			}
		}
	}
	walk(root)
	return best
}

// handleFormatting applies enhanced formatting with AST support and minimal diff edits.
func (s *Server) handleFormatting(uri string, options map[string]any) []map[string]any {
	text := s.docs[uri]
	if text == "" {
		return []map[string]any{}
	}

	// Check if enhanced formatting is requested
	var useAST bool
	if opts, ok := options["astFormat"]; ok {
		if ast, ok := opts.(bool); ok {
			useAST = ast
		}
	}

	var formatted string
	if useAST {
		// Use enhanced AST-based formatting
		astOptions := format.DefaultASTFormattingOptions()

		// Apply client preferences if available
		if tabSize, ok := options["tabSize"]; ok {
			if size, ok := tabSize.(float64); ok {
				astOptions.IndentSize = int(size)
			}
		}
		if insertSpaces, ok := options["insertSpaces"]; ok {
			if spaces, ok := insertSpaces.(bool); ok {
				astOptions.PreferTabs = !spaces
			}
		}

		var err error
		formatted, err = format.FormatSourceWithAST(text, astOptions)
		if err != nil {
			// Fallback to basic formatting on AST error
			formatted = format.FormatText(text, format.Options{PreserveNewlineStyle: true})
		}
	} else {
		// Use basic formatting
		formatted = format.FormatText(text, format.Options{PreserveNewlineStyle: true})
	}

	if formatted == text {
		return []map[string]any{}
	}

	// Use diff-based minimal edits instead of full document replacement
	return s.generateMinimalEdits(text, formatted)
}

// generateMinimalEdits creates minimal text edits using diff algorithm
func (s *Server) generateMinimalEdits(original, formatted string) []map[string]any {
	// Use diff formatter to find minimal changes
	diffOptions := format.DefaultDiffOptions()
	diffFormatter := format.NewDiffFormatter(diffOptions)
	result := diffFormatter.GenerateDiff("document", original, formatted)

	if !result.HasChanges {
		return []map[string]any{}
	}

	var edits []map[string]any
	originalLines := strings.Split(original, "\n")
	formattedLines := strings.Split(formatted, "\n")

	// Convert diff hunks to LSP text edits
	for _, hunk := range result.Hunks {
		// Calculate actual line ranges (diff uses 1-based, LSP uses 0-based)
		startLine := hunk.OriginalStart - 1
		if startLine < 0 {
			startLine = 0
		}

		endLine := startLine + hunk.OriginalCount
		if endLine > len(originalLines) {
			endLine = len(originalLines)
		}

		// Get replacement text from formatted version
		newStartLine := hunk.ModifiedStart - 1
		if newStartLine < 0 {
			newStartLine = 0
		}
		newEndLine := newStartLine + hunk.ModifiedCount
		if newEndLine > len(formattedLines) {
			newEndLine = len(formattedLines)
		}

		var newText string
		if newEndLine > newStartLine {
			newText = strings.Join(formattedLines[newStartLine:newEndLine], "\n")
			if newEndLine < len(formattedLines) {
				newText += "\n"
			}
		}

		// Calculate character positions
		startOffset := 0
		for i := 0; i < startLine && i < len(originalLines); i++ {
			startOffset += len(originalLines[i]) + 1 // +1 for newline
		}

		endOffset := startOffset
		for i := startLine; i < endLine && i < len(originalLines); i++ {
			endOffset += len(originalLines[i])
			if i < len(originalLines)-1 {
				endOffset += 1 // +1 for newline except last line
			}
		}

		startLinePos, startChar := utf16LineCharFromOffset(original, startOffset)
		endLinePos, endChar := utf16LineCharFromOffset(original, endOffset)

		edits = append(edits, map[string]any{
			"range": map[string]any{
				"start": map[string]any{"line": startLinePos, "character": startChar},
				"end":   map[string]any{"line": endLinePos, "character": endChar},
			},
			"newText": newText,
		})
	}

	return edits
}

// handleRangeFormatting applies formatting to a subset of lines with minimal edits.
func (s *Server) handleRangeFormatting(uri string, startLine, endLine int, options map[string]any) []map[string]any {
	text := s.docs[uri]
	if text == "" {
		return []map[string]any{}
	}
	if startLine < 0 {
		startLine = 0
	}
	if endLine < startLine {
		endLine = startLine
	}

	// For range formatting, we format the entire document but only return edits within the range
	allEdits := s.handleFormatting(uri, options)
	if len(allEdits) == 0 {
		return []map[string]any{}
	}

	// Filter edits to only those that intersect with the requested range
	var rangeEdits []map[string]any
	for _, edit := range allEdits {
		if editRange, ok := edit["range"].(map[string]any); ok {
			if editStart, ok := editRange["start"].(map[string]any); ok {
				if editEnd, ok := editRange["end"].(map[string]any); ok {
					editStartLine := int(editStart["line"].(float64))
					editEndLine := int(editEnd["line"].(float64))

					// Check if edit intersects with requested range
					if editStartLine <= endLine && editEndLine >= startLine {
						rangeEdits = append(rangeEdits, edit)
					}
				}
			}
		}
	}

	return rangeEdits
}

// handleCodeAction provides simple quick fixes like inserting missing function body braces or removing trailing spaces.
func (s *Server) handleCodeAction(uri string, startLine, startChar, endLine, endChar int) []map[string]any {
	text := s.docs[uri]
	if text == "" {
		return []map[string]any{}
	}
	actions := make([]map[string]any, 0, 10)
	// Action: Trim trailing spaces in document
	cleanedLines := strings.Split(text, "\n")
	changed := false
	for i := range cleanedLines {
		trimmed := strings.TrimRight(cleanedLines[i], " \t")
		if trimmed != cleanedLines[i] {
			cleanedLines[i] = trimmed
			changed = true
		}
	}
	if changed {
		cleaned := strings.Join(cleanedLines, "\n")
		endLine, endChar := utf16LineCharFromOffset(text, len(text))
		actions = append(actions, map[string]any{
			"title": "Remove trailing spaces",
			"kind":  "quickfix",
			"edit": map[string]any{
				"changes": map[string]any{
					uri: []map[string]any{{
						"range": map[string]any{
							"start": map[string]any{"line": 0, "character": 0},
							"end":   map[string]any{"line": endLine, "character": endChar},
						},
						"newText": cleaned,
					}},
				},
			},
		})
	}

	// Action: parser suggestions as quickfixes (insert replacement at position)
	lx := lexer.NewWithFilename(text, uri)
	pr := parser.NewParser(lx, uri)
	_, _ = pr.Parse()
	for _, sgg := range pr.GetSuggestions() {
		if strings.TrimSpace(sgg.Replacement) == "" {
			continue
		}
		l0, c0 := utf16LineCharFromOffset(text, sgg.Position.Offset)
		actions = append(actions, map[string]any{
			"title": "Apply fix: " + sgg.Message,
			"kind":  "quickfix",
			"edit": map[string]any{
				"changes": map[string]any{
					uri: []map[string]any{{
						"range": map[string]any{
							"start": map[string]any{"line": l0, "character": c0},
							"end":   map[string]any{"line": l0, "character": c0},
						},
						"newText": sgg.Replacement,
					}},
				},
			},
		})
		if len(actions) >= 10 {
			break
		}
	}

	// Refactor: extract selection to a local variable when a range is provided
	if startLine >= 0 && endLine >= startLine {
		startOff := offsetFromLineCharUTF16(text, startLine, startChar)
		endOff := offsetFromLineCharUTF16(text, endLine, endChar)
		if startOff >= 0 && endOff > startOff && endOff <= len(text) {
			sel := strings.TrimSpace(text[startOff:endOff])
			if sel != "" && !strings.Contains(sel, "\n") {
				name := "extracted"
				// Minimal edits: (1) insert declaration at start of selection line, (2) replace selection with variable name
				insLineStart, _ := getLineBounds(text, startLine)
				sL, sC := utf16LineCharFromOffset(text, insLineStart)
				// selection range in LSP positions is provided by caller
				actions = append(actions, map[string]any{
					"title": "Refactor: extract to variable",
					"kind":  "refactor.extract",
					"edit": map[string]any{
						"changes": map[string]any{
							uri: []map[string]any{
								{
									"range": map[string]any{
										"start": map[string]any{"line": sL, "character": sC},
										"end":   map[string]any{"line": sL, "character": sC},
									},
									"newText": "let " + name + " = " + sel + "\n",
								},
								{
									"range": map[string]any{
										"start": map[string]any{"line": startLine, "character": startChar},
										"end":   map[string]any{"line": endLine, "character": endChar},
									},
									"newText": name,
								},
							},
						},
					},
				})
			}
		}
	}
	return actions
}

// handleSignatureHelp returns a minimal signature help for the function being called.
func (s *Server) handleSignatureHelp(uri string, line, character int) map[string]any {
	text := s.docs[uri]
	if text == "" {
		return nil
	}
	offset := offsetFromLineCharUTF16(text, line, character)
	if offset < 0 || offset > len(text) {
		return nil
	}
	// Find identifier before the nearest '('
	i := offset - 1
	for i >= 0 && text[i] != '(' && text[i] != '\n' {
		i--
	}
	if i < 1 || text[i] != '(' {
		return nil
	}
	// Backtrack to get function name
	j := i - 1
	for j >= 0 {
		r, size := utf8.DecodeLastRuneInString(text[:j+1])
		if size <= 0 || !isIdentRune(r) {
			break
		}
		j -= size
	}
	name := strings.TrimSpace(text[j+1 : i])
	if name == "" {
		return nil
	}
	infos := s.symIndex[uri][name]
	label := name + "()"
	if len(infos) > 0 && infos[0].Detail != "" {
		label = infos[0].Detail
	}
	// Determine active parameter index by counting commas from the opening '('
	activeParam := 0
	depth := 0
	for k := i + 1; k < offset && k < len(text); k++ {
		ch := text[k]
		if ch == '(' {
			depth++
		} else if ch == ')' {
			if depth > 0 {
				depth--
			}
		} else if ch == ',' && depth == 0 {
			activeParam++
		}
	}
	// Build parameters metadata if we can locate the function declaration
	var paramsMeta []any
	var documentation string
	if ast := s.astCache[uri]; ast != nil {
		if fn := findFunctionDeclByName(ast, name); fn != nil {
			for _, pnode := range fn.Parameters {
				if pnode == nil || pnode.Name == nil {
					continue
				}
				paramsMeta = append(paramsMeta, map[string]any{
					"label": buildParamDetail(pnode),
				})
			}
			// Build short documentation block with return type and param list
			var parts []string
			if fn.ReturnType != nil {
				parts = append(parts, "returns: "+fn.ReturnType.String())
			}
			if len(fn.Parameters) > 0 {
				plist := make([]string, 0, len(fn.Parameters))
				for _, p := range fn.Parameters {
					if p == nil || p.Name == nil {
						continue
					}
					if p.TypeSpec != nil {
						plist = append(plist, p.Name.Value+": "+p.TypeSpec.String())
					} else {
						plist = append(plist, p.Name.Value)
					}
				}
				if len(plist) > 0 {
					parts = append(parts, "params: "+strings.Join(plist, ", "))
				}
			}
			if len(parts) > 0 {
				documentation = strings.Join(parts, "\n")
			}
		}
	}
	sig := map[string]any{"label": label}
	if len(paramsMeta) > 0 {
		sig["parameters"] = paramsMeta
	}
	if documentation != "" {
		sig["documentation"] = documentation
	}
	return map[string]any{
		"signatures":      []any{sig},
		"activeSignature": 0,
		"activeParameter": activeParam,
	}
}

// handleOnTypeFormatting performs intelligent edits on specific typed characters with minimal changes.
func (s *Server) handleOnTypeFormatting(uri string, line, character int, ch string, options map[string]any) []map[string]any {
	if ch != "}" && ch != "\n" && ch != ")" && ch != ";" && ch != "{" {
		return []map[string]any{}
	}
	text := s.docs[uri]
	if text == "" {
		return []map[string]any{}
	}

	// Check if enhanced formatting is enabled
	var useEnhanced bool
	if opts, ok := options["astFormat"]; ok {
		if ast, ok := opts.(bool); ok {
			useEnhanced = ast
		}
	}

	if useEnhanced {
		// Use AST-aware on-type formatting
		return s.handleEnhancedOnTypeFormatting(uri, line, character, ch, options)
	}

	// Basic on-type formatting (existing logic)
	startOff, endOff := getLineBounds(text, line)
	if startOff < 0 || endOff < 0 {
		return []map[string]any{}
	}
	lineText := text[startOff:endOff]

	// Compute desired indent for the current line
	indentLevel := computeIndentBeforeLine(text, line)

	// Special handling for different characters
	trimmed := strings.TrimLeft(lineText, " \t")
	switch ch {
	case "}":
		// Closing brace reduces indent
		if strings.HasPrefix(trimmed, "}") && indentLevel > 0 {
			indentLevel--
		}
	case "{":
		// Opening brace may need space before it
		if !strings.HasSuffix(strings.TrimSpace(lineText), " {") && strings.HasSuffix(trimmed, "{") {
			// Add space before brace if not present
			newText := strings.TrimRight(lineText, " \t")
			if !strings.HasSuffix(newText, " ") && !strings.HasSuffix(newText, "\t") {
				newText = strings.TrimSuffix(newText, "{")
				newText += " {"
				return s.createSingleLineEdit(text, line, newText)
			}
		}
	case ";":
		// Semicolon formatting can trigger end-of-statement cleanup
		// Remove trailing spaces after semicolon
		if strings.HasSuffix(strings.TrimSpace(lineText), ";") {
			newText := strings.TrimRight(lineText, " \t")
			if newText != lineText {
				return s.createSingleLineEdit(text, line, newText)
			}
		}
	}

	// Apply indentation
	indentUnit := getIndentUnit(options)
	newLine := strings.Repeat(indentUnit, indentLevel) + strings.TrimRight(trimmed, " \t")

	if newLine == lineText {
		return []map[string]any{}
	}

	return s.createSingleLineEdit(text, line, newLine)
}

// handleEnhancedOnTypeFormatting uses AST information for smarter on-type formatting
func (s *Server) handleEnhancedOnTypeFormatting(uri string, line, character int, ch string, options map[string]any) []map[string]any {
	text := s.docs[uri]

	// For enhanced formatting, we format a small region around the cursor
	// to maintain performance while providing better formatting
	contextLines := 5 // Format 5 lines before and after
	startLine := line - contextLines
	if startLine < 0 {
		startLine = 0
	}

	endLine := line + contextLines
	lines := strings.Split(text, "\n")
	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	// Apply range formatting to the context area
	return s.handleRangeFormatting(uri, startLine, endLine, options)
}

// createSingleLineEdit creates a text edit for a single line
func (s *Server) createSingleLineEdit(text string, line int, newText string) []map[string]any {
	startOff, endOff := getLineBounds(text, line)
	if startOff < 0 || endOff < 0 {
		return []map[string]any{}
	}

	sL, sC := utf16LineCharFromOffset(text, startOff)
	eL, eC := utf16LineCharFromOffset(text, endOff)

	return []map[string]any{{
		"range": map[string]any{
			"start": map[string]any{"line": sL, "character": sC},
			"end":   map[string]any{"line": eL, "character": eC},
		},
		"newText": newText,
	}}
}

// getIndentUnit returns the indentation unit based on editor options
func getIndentUnit(options map[string]any) string {
	indentUnit := "  " // Default to 2 spaces
	if ws, ok := options["insertSpaces"].(bool); ok && !ws {
		indentUnit = "\t"
	} else if size, ok := options["tabSize"].(float64); ok && size > 0 {
		indentUnit = strings.Repeat(" ", int(size))
	}
	return indentUnit
}

// handleSemanticTokensFull builds LSP semantic tokens for the entire document using a simple
// lexer-based strategy. It classifies identifiers with help of symbol index and colors
// keywords, strings, numbers, comments and operators.
func (s *Server) handleSemanticTokensFull(uri string) map[string]any {
	text := s.docs[uri]
	if text == "" {
		return map[string]any{"data": []uint32{}}
	}
	// Tokenize
	lx := lexer.NewWithFilename(text, uri)
	// LSP semantic tokens are encoded as (lineDelta, startCharDelta, length, tokenType, tokenModifiers)
	type enc = uint32
	data := make([]enc, 0, 512)
	// Cursor for delta encoding
	prevLine := 0
	prevChar := 0
	encode := func(sLine, sChar, length, typ, mods int) {
		lineDelta := sLine - prevLine
		charDelta := sChar
		if lineDelta == 0 {
			charDelta = sChar - prevChar
		}
		data = append(data, enc(lineDelta), enc(charDelta), enc(length), enc(typ), enc(mods))
		prevLine = sLine
		prevChar = sChar
	}
	// helpers
	tokType := func(tt lexer.TokenType, lit string) (string, bool) {
		switch tt {
		case lexer.TokenIdentifier:
			if si := s.symIndex[uri][lit]; len(si) > 0 {
				if si[0].Kind == 12 { // function
					return "function", true
				}
				if si[0].Kind == 13 { // variable/parameter
					return "variable", true
				}
			}
			return "variable", true
		case lexer.TokenStruct:
			return "type", true
		case lexer.TokenEnum, lexer.TokenTrait:
			return "type", true
		case lexer.TokenFunc:
			return "keyword", true
		case lexer.TokenLet, lexer.TokenVar, lexer.TokenConst, lexer.TokenIf, lexer.TokenElse, lexer.TokenFor, lexer.TokenWhile,
			lexer.TokenLoop, lexer.TokenReturn, lexer.TokenBreak, lexer.TokenContinue, lexer.TokenAsync, lexer.TokenAwait,
			lexer.TokenImpl, lexer.TokenImport, lexer.TokenExport,
			lexer.TokenModule, lexer.TokenPub, lexer.TokenMut, lexer.TokenAs, lexer.TokenIn, lexer.TokenWhere, lexer.TokenUnsafe,
			lexer.TokenActor, lexer.TokenSpawn, lexer.TokenMacro:
			return "keyword", true
		case lexer.TokenString, lexer.TokenChar:
			return "string", true
		case lexer.TokenInteger, lexer.TokenFloat, lexer.TokenBool:
			return "number", true
		case lexer.TokenComment:
			return "comment", true
		case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenMul, lexer.TokenDiv, lexer.TokenMod, lexer.TokenPower,
			lexer.TokenAssign, lexer.TokenPlusAssign, lexer.TokenMinusAssign, lexer.TokenMulAssign, lexer.TokenDivAssign,
			lexer.TokenModAssign, lexer.TokenEq, lexer.TokenNe, lexer.TokenLt, lexer.TokenLe, lexer.TokenGt, lexer.TokenGe,
			lexer.TokenAnd, lexer.TokenOr, lexer.TokenNot, lexer.TokenBitAnd, lexer.TokenBitOr, lexer.TokenBitXor,
			lexer.TokenBitNot, lexer.TokenShl, lexer.TokenShr, lexer.TokenBitAndAssign, lexer.TokenBitOrAssign,
			lexer.TokenBitXorAssign, lexer.TokenShlAssign, lexer.TokenShrAssign:
			return "operator", true
		default:
			return "", false
		}
	}
	// Iterate tokens and emit semantic tokens
	for {
		tok := lx.NextToken()
		if tok.Type == lexer.TokenEOF {
			break
		}
		if tok.Type == lexer.TokenWhitespace || tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenError {
			continue
		}
		tt, ok := tokType(tok.Type, tok.Literal)
		if !ok {
			continue
		}
		// Compute 0-based UTF-16 start and length in UTF-16 units
		sLine, sChar := utf16LineCharFromOffset(text, tok.Span.Start.Offset)
		eLine, eChar := utf16LineCharFromOffset(text, tok.Span.End.Offset)
		// Only support single-line tokens for now; if multi-line, split head part
		if eLine != sLine {
			eLine = sLine
			eChar = sChar + (eChar - sChar) // head only
		}
		length := eChar - sChar
		if length <= 0 {
			continue
		}
		typIdx, okType := s.semLegend.TypeIndex[tt]
		if !okType {
			continue
		}
		mods := 0
		if tok.Type == lexer.TokenIdentifier {
			if entries, ok := s.symIndex[uri][tok.Literal]; ok {
				for _, si := range entries {
					if si.Span.Start.Offset == tok.Span.Start.Offset && si.Span.End.Offset == tok.Span.End.Offset {
						if idx, okm := s.semLegend.ModIndex["declaration"]; okm {
							mods |= 1 << idx
						}
						break
					}
				}
			}
		}
		encode(sLine, sChar, length, typIdx, mods)
	}
	return map[string]any{"data": data}
}

// handleSemanticTokensRange limits the produced tokens to a line range, reusing the same encoder.
func (s *Server) handleSemanticTokensRange(uri string, startLine, endLine int) map[string]any {
	text := s.docs[uri]
	if text == "" {
		return map[string]any{"data": []uint32{}}
	}
	lx := lexer.NewWithFilename(text, uri)
	type enc = uint32
	data := make([]enc, 0, 256)
	prevLine := startLine
	prevChar := 0
	encode := func(sLine, sChar, length, typ, mods int) {
		lineDelta := sLine - prevLine
		charDelta := sChar
		if lineDelta == 0 {
			charDelta = sChar - prevChar
		}
		data = append(data, enc(lineDelta), enc(charDelta), enc(length), enc(typ), enc(mods))
		prevLine = sLine
		prevChar = sChar
	}
	tokType := func(tt lexer.TokenType, lit string) (string, bool) {
		switch tt {
		case lexer.TokenIdentifier:
			if si := s.symIndex[uri][lit]; len(si) > 0 {
				if si[0].Kind == 12 {
					return "function", true
				}
				if si[0].Kind == 13 {
					return "variable", true
				}
			}
			return "variable", true
		case lexer.TokenStruct:
			return "type", true
		case lexer.TokenEnum, lexer.TokenTrait:
			return "type", true
		case lexer.TokenFunc:
			return "keyword", true
		case lexer.TokenLet, lexer.TokenVar, lexer.TokenConst, lexer.TokenIf, lexer.TokenElse, lexer.TokenFor, lexer.TokenWhile,
			lexer.TokenLoop, lexer.TokenReturn, lexer.TokenBreak, lexer.TokenContinue, lexer.TokenAsync, lexer.TokenAwait,
			lexer.TokenImpl, lexer.TokenImport, lexer.TokenExport,
			lexer.TokenModule, lexer.TokenPub, lexer.TokenMut, lexer.TokenAs, lexer.TokenIn, lexer.TokenWhere, lexer.TokenUnsafe,
			lexer.TokenActor, lexer.TokenSpawn, lexer.TokenMacro:
			return "keyword", true
		case lexer.TokenString, lexer.TokenChar:
			return "string", true
		case lexer.TokenInteger, lexer.TokenFloat, lexer.TokenBool:
			return "number", true
		case lexer.TokenComment:
			return "comment", true
		case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenMul, lexer.TokenDiv, lexer.TokenMod, lexer.TokenPower,
			lexer.TokenAssign, lexer.TokenPlusAssign, lexer.TokenMinusAssign, lexer.TokenMulAssign, lexer.TokenDivAssign,
			lexer.TokenModAssign, lexer.TokenEq, lexer.TokenNe, lexer.TokenLt, lexer.TokenLe, lexer.TokenGt, lexer.TokenGe,
			lexer.TokenAnd, lexer.TokenOr, lexer.TokenNot, lexer.TokenBitAnd, lexer.TokenBitOr, lexer.TokenBitXor,
			lexer.TokenBitNot, lexer.TokenShl, lexer.TokenShr, lexer.TokenBitAndAssign, lexer.TokenBitOrAssign,
			lexer.TokenBitXorAssign, lexer.TokenShlAssign, lexer.TokenShrAssign:
			return "operator", true
		default:
			return "", false
		}
	}
	for {
		tok := lx.NextToken()
		if tok.Type == lexer.TokenEOF {
			break
		}
		if tok.Type == lexer.TokenWhitespace || tok.Type == lexer.TokenNewline || tok.Type == lexer.TokenError {
			continue
		}
		sL, sC := utf16LineCharFromOffset(text, tok.Span.Start.Offset)
		eL, eC := utf16LineCharFromOffset(text, tok.Span.End.Offset)
		if sL < startLine || sL > endLine {
			continue
		}
		tt, ok := tokType(tok.Type, tok.Literal)
		if !ok {
			continue
		}
		if eL != sL {
			eL = sL
			eC = sC + (eC - sC)
		}
		length := eC - sC
		if length <= 0 {
			continue
		}
		typIdx := s.semLegend.TypeIndex[tt]
		mods := 0
		if tok.Type == lexer.TokenIdentifier {
			if entries, ok := s.symIndex[uri][tok.Literal]; ok {
				for _, si := range entries {
					if si.Span.Start.Offset == tok.Span.Start.Offset && si.Span.End.Offset == tok.Span.End.Offset {
						if idx, okm := s.semLegend.ModIndex["declaration"]; okm {
							mods |= 1 << idx
						}
						break
					}
				}
			}
		}
		encode(sL, sC, length, typIdx, mods)
	}
	return map[string]any{"data": data}
}

// handleInlayHints returns simple parameter name hints and variable type hints for the given line range.
func (s *Server) handleInlayHints(uri string, startLine, endLine int) []map[string]any {
	text := s.docs[uri]
	if text == "" {
		return []map[string]any{}
	}
	ast := s.astCache[uri]
	if ast == nil {
		lx := lexer.NewWithFilename(text, uri)
		pr := parser.NewParser(lx, uri)
		a, _ := pr.Parse()
		ast = a
		s.astCache[uri] = ast
		if ast != nil {
			s.buildSymbolIndex(uri, ast)
		}
	}
	if ast == nil {
		return []map[string]any{}
	}
	hints := make([]map[string]any, 0, 64)
	// Parameter hints: find call expressions within line range and emit labels for args
	var walk func(n parser.Node)
	walk = func(n parser.Node) {
		if n == nil {
			return
		}
		sp := n.GetSpan()
		sL, _ := utf16LineCharFromOffset(text, sp.Start.Offset)
		eL, _ := utf16LineCharFromOffset(text, sp.End.Offset)
		// prune traversal outside range
		if eL < startLine || sL > endLine {
			return
		}
		switch node := n.(type) {
		case *parser.Program:
			for _, d := range node.Declarations {
				walk(d)
			}
		case *parser.FunctionDeclaration:
			if node.Body != nil {
				walk(node.Body)
			}
		case *parser.BlockStatement:
			for _, st := range node.Statements {
				walk(st)
			}
		case *parser.CallExpression:
			// If callee is identifier and parameters exist in index, label arguments as name:
			id, ok := node.Function.(*parser.Identifier)
			if ok && id != nil {
				// Build parameter names from function declaration in index when available
				if entries := s.symIndex[uri][id.Value]; len(entries) > 0 {
					// Attempt to locate corresponding function decl span and extract params names
					// Best effort: parse again and search by name
					if fn := findFunctionDeclByName(ast, id.Value); fn != nil {
						for i, arg := range node.Arguments {
							if i < len(fn.Parameters) && fn.Parameters[i] != nil && fn.Parameters[i].Name != nil {
								// Place hint at argument start
								aSL, aSC := utf16LineCharFromOffset(text, arg.GetSpan().Start.Offset)
								label := fn.Parameters[i].Name.Value + ":"
								hints = append(hints, map[string]any{
									"position": map[string]any{"line": aSL, "character": aSC},
									"label":    label,
									"kind":     2, // Parameter
								})
							}
						}
						// Return type hint at call end
						if ret := s.funcReturnTypeFromIndex(uri, id.Value); ret != "" {
							endPos := node.Span.End.Offset
							l, c := utf16LineCharFromOffset(text, endPos)
							if l >= startLine && l <= endLine {
								hints = append(hints, map[string]any{
									"position": map[string]any{"line": l, "character": c},
									"label":    " : " + ret,
									"kind":     1, // Type
								})
							}
						}
					}
				}
			}
			for _, a := range node.Arguments {
				walk(a)
			}
		}
	}
	walk(ast)
	// Variable type hints: for variable declarations with explicit type, place hint after name
	var walkTypes func(n parser.Node)
	walkTypes = func(n parser.Node) {
		if n == nil {
			return
		}
		switch node := n.(type) {
		case *parser.Program:
			for _, d := range node.Declarations {
				walkTypes(d)
			}
		case *parser.FunctionDeclaration:
			if node.Body != nil {
				walkTypes(node.Body)
			}
		case *parser.BlockStatement:
			for _, st := range node.Statements {
				walkTypes(st)
			}
		case *parser.VariableDeclaration:
			if node.Name != nil && node.TypeSpec != nil {
				// Position right after the identifier name
				pos := node.Name.Span.End.Offset
				l, c := utf16LineCharFromOffset(text, pos)
				if l >= startLine && l <= endLine {
					hints = append(hints, map[string]any{
						"position": map[string]any{"line": l, "character": c},
						"label":    ": " + node.TypeSpec.String(),
						"kind":     1, // Type
					})
				}
			}
			// Heuristic type hint for inferred variables without explicit type
			if node.Name != nil && node.TypeSpec == nil && node.Initializer != nil {
				pos := node.Name.Span.End.Offset
				l, c := utf16LineCharFromOffset(text, pos)
				if l >= startLine && l <= endLine {
					var tlabel string
					switch init := node.Initializer.(type) {
					case *parser.Literal:
						switch init.Kind {
						case parser.LiteralInteger:
							tlabel = "Int"
						case parser.LiteralFloat:
							tlabel = "Float"
						case parser.LiteralString:
							tlabel = "String"
						case parser.LiteralBool:
							tlabel = "Bool"
						}
					}
					if tlabel != "" {
						hints = append(hints, map[string]any{
							"position": map[string]any{"line": l, "character": c},
							"label":    ": " + tlabel,
							"kind":     1,
						})
					}
				}
			}
		}
	}
	walkTypes(ast)
	return hints
}

func findFunctionDeclByName(root *parser.Program, name string) *parser.FunctionDeclaration {
	if root == nil {
		return nil
	}
	for _, d := range root.Declarations {
		if fn, ok := d.(*parser.FunctionDeclaration); ok {
			if fn.Name != nil && fn.Name.Value == name {
				return fn
			}
		}
	}
	return nil
}

func findVariableDeclByName(root *parser.Program, name string) *parser.VariableDeclaration {
	if root == nil {
		return nil
	}
	for _, d := range root.Declarations {
		if v, ok := d.(*parser.VariableDeclaration); ok {
			if v.Name != nil && v.Name.Value == name {
				return v
			}
		}
	}
	return nil
}

func getLineBounds(text string, line int) (int, int) {
	if line < 0 {
		return -1, -1
	}
	currLine := 0
	start := 0
	i := 0
	for i < len(text) && currLine < line {
		if text[i] == '\n' {
			currLine++
			start = i + 1
		}
		i++
	}
	if currLine != line {
		return -1, -1
	}
	end := start
	for end < len(text) && text[end] != '\n' {
		end++
	}
	return start, end
}

func computeIndentBeforeLine(text string, line int) int {
	if line <= 0 {
		return 0
	}
	start, _ := getLineBounds(text, line)
	if start <= 0 {
		return 0
	}
	level := 0
	for i := 0; i < start; i++ {
		switch text[i] {
		case '{':
			level++
		case '}':
			if level > 0 {
				level--
			}
		}
	}
	if level < 0 {
		level = 0
	}
	return level
}

// handleDocumentHighlight highlights all occurrences of the symbol at position.
func (s *Server) handleDocumentHighlight(uri string, line, character int) []map[string]any {
	text := s.docs[uri]
	if text == "" {
		return []map[string]any{}
	}
	offset := offsetFromLineCharUTF16(text, line, character)
	if offset < 0 || offset > len(text) {
		return []map[string]any{}
	}
	// Extract identifier at position
	start := offset
	for start > 0 {
		r, sz := utf8.DecodeLastRuneInString(text[:start])
		if sz <= 0 || !isIdentRune(r) {
			break
		}
		start -= sz
	}
	end := offset
	for end < len(text) {
		r, sz := utf8.DecodeRuneInString(text[end:])
		if sz <= 0 || !isIdentRune(r) {
			break
		}
		end += sz
	}
	if start >= end {
		return []map[string]any{}
	}
	name := strings.TrimSpace(text[start:end])
	if name == "" {
		return []map[string]any{}
	}

	// Scan tokens and return highlight ranges
	results := make([]map[string]any, 0, 16)
	lx := lexer.NewWithFilename(text, uri)
	for {
		tok := lx.NextToken()
		if tok.Type == lexer.TokenEOF {
			break
		}
		if tok.Type == lexer.TokenIdentifier && tok.Literal == name {
			l0, c0 := utf16LineCharFromOffset(text, tok.Span.Start.Offset)
			l1, c1 := utf16LineCharFromOffset(text, tok.Span.End.Offset)
			results = append(results, map[string]any{
				"range": map[string]any{
					"start": map[string]any{"line": l0, "character": c0},
					"end":   map[string]any{"line": l1, "character": c1},
				},
				"kind": 1, // Text
			})
		}
	}
	return results
}

// handleFoldingRange returns folding regions for brace-delimited blocks.
func (s *Server) handleFoldingRange(uri string) []map[string]any {
	text := s.docs[uri]
	if text == "" {
		return []map[string]any{}
	}
	lines := strings.Split(text, "\n")
	type stackEntry struct{ line int }
	stack := make([]stackEntry, 0, 32)
	ranges := make([]map[string]any, 0, 32)
	for i, ln := range lines {
		trimmed := ln
		// Count braces in this line; push for '{', pop for '}'
		for idx := 0; idx < len(trimmed); idx++ {
			ch := trimmed[idx]
			if ch == '{' {
				stack = append(stack, stackEntry{line: i})
			} else if ch == '}' {
				if len(stack) > 0 {
					top := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					if i > top.line { // multi-line only
						ranges = append(ranges, map[string]any{
							"startLine": top.line,
							"endLine":   i,
							"kind":      "region",
						})
					}
				}
			}
		}
	}
	return ranges
}

// handlePrepareRename validates rename availability and returns the range and placeholder.
func (s *Server) handlePrepareRename(uri string, line, character int) map[string]any {
	text := s.docs[uri]
	if text == "" {
		return nil
	}
	offset := offsetFromLineCharUTF16(text, line, character)
	if offset < 0 || offset > len(text) {
		return nil
	}
	// Extract identifier
	start := offset
	for start > 0 {
		r, sz := utf8.DecodeLastRuneInString(text[:start])
		if sz <= 0 || !isIdentRune(r) {
			break
		}
		start -= sz
	}
	end := offset
	for end < len(text) {
		r, sz := utf8.DecodeRuneInString(text[end:])
		if sz <= 0 || !isIdentRune(r) {
			break
		}
		end += sz
	}
	if start >= end {
		return nil
	}
	name := strings.TrimSpace(text[start:end])
	if name == "" {
		return nil
	}
	l0, c0 := utf16LineCharFromOffset(text, start)
	l1, c1 := utf16LineCharFromOffset(text, end)
	return map[string]any{
		"range": map[string]any{
			"start": map[string]any{"line": l0, "character": c0},
			"end":   map[string]any{"line": l1, "character": c1},
		},
		"placeholder": name,
	}
}

// handleRename performs a simple in-file rename of all occurrences of the symbol under the cursor.
func (s *Server) handleRename(uri string, line, character int, newName string) (map[string]any, string) {
	text := s.docs[uri]
	if text == "" || strings.TrimSpace(newName) == "" {
		return map[string]any{"changes": map[string]any{}}, "invalid rename request"
	}
	offset := offsetFromLineCharUTF16(text, line, character)
	if offset < 0 || offset > len(text) {
		return map[string]any{"changes": map[string]any{}}, "position out of range"
	}
	// Identifier at position
	start := offset
	for start > 0 {
		r, sz := utf8.DecodeLastRuneInString(text[:start])
		if sz <= 0 || !isIdentRune(r) {
			break
		}
		start -= sz
	}
	end := offset
	for end < len(text) {
		r, sz := utf8.DecodeRuneInString(text[end:])
		if sz <= 0 || !isIdentRune(r) {
			break
		}
		end += sz
	}
	if start >= end {
		return map[string]any{"changes": map[string]any{}}, "no symbol at position"
	}
	name := strings.TrimSpace(text[start:end])
	if name == "" {
		return map[string]any{"changes": map[string]any{}}, "no symbol at position"
	}

	// Determine scope from symbol index (best effort)
	var scope *parser.Span
	if entries, ok := s.symIndex[uri][name]; ok && len(entries) > 0 {
		sc := entries[0].Scope
		if sc.End.Offset > sc.Start.Offset {
			scope = &sc
		}
	}

	// Conflict detection: if newName already exists in the same scope, abort
	if scope != nil {
		if newEntries, ok := s.symIndex[uri][newName]; ok && len(newEntries) > 0 {
			for _, si := range newEntries {
				if !(si.Span.End.Offset <= scope.Start.Offset || si.Span.Start.Offset >= scope.End.Offset) {
					return map[string]any{"changes": map[string]any{}}, "rename would conflict with existing symbol in scope"
				}
			}
		}
	}

	// Collect edits for all identifier tokens that match the name
	edits := make([]map[string]any, 0, 32)
	lx := lexer.NewWithFilename(text, uri)
	for {
		tok := lx.NextToken()
		if tok.Type == lexer.TokenEOF {
			break
		}
		if tok.Type == lexer.TokenIdentifier && tok.Literal == name {
			if scope != nil {
				if tok.Span.Start.Offset < scope.Start.Offset || tok.Span.End.Offset > scope.End.Offset {
					continue
				}
			}
			l0, c0 := utf16LineCharFromOffset(text, tok.Span.Start.Offset)
			l1, c1 := utf16LineCharFromOffset(text, tok.Span.End.Offset)
			edits = append(edits, map[string]any{
				"range": map[string]any{
					"start": map[string]any{"line": l0, "character": c0},
					"end":   map[string]any{"line": l1, "character": c1},
				},
				"newText": newName,
			})
		}
	}
	// Cross-file edits using workspace index
	wsChanges := map[string]any{uri: edits}
	for _, loc := range s.wsIndex[name] {
		if loc.URI == uri {
			continue
		}
		// Skip if target file already declares newName (declaration-level conflict)
		if newLocs, ok := s.wsIndex[newName]; ok {
			conflict := false
			for _, nl := range newLocs {
				if nl.URI == loc.URI {
					conflict = true
					break
				}
			}
			if conflict {
				continue
			}
		}
		// Load file content (from open docs cache or filesystem)
		otherText := s.docs[loc.URI]
		if otherText == "" {
			if path := filePathFromURI(loc.URI); path != "" {
				// Restrict access to files under workspace root
				if s.pathUnderRoot(path) {
					b, err := os.ReadFile(path)
					if err == nil {
						otherText = string(b)
					}
				}
			}
		}
		if otherText == "" {
			continue
		}
		l0, c0 := utf16LineCharFromOffset(otherText, loc.Span.Start.Offset)
		l1, c1 := utf16LineCharFromOffset(otherText, loc.Span.End.Offset)
		wsChanges[loc.URI] = append([]map[string]any{}, map[string]any{
			"range": map[string]any{
				"start": map[string]any{"line": l0, "character": c0},
				"end":   map[string]any{"line": l1, "character": c1},
			},
			"newText": newName,
		})
	}
	return map[string]any{"changes": wsChanges}, ""
}

// handleWorkspaceSymbol returns flat symbol info for all indexed documents that match the query (prefix match).
func (s *Server) handleWorkspaceSymbol(query string) []map[string]any {
	if query == "" {
		return []map[string]any{}
	}
	results := make([]map[string]any, 0, 64)
	for uri, idx := range s.symIndex {
		for name, infos := range idx {
			if !strings.HasPrefix(strings.ToLower(name), strings.ToLower(query)) {
				continue
			}
			for _, si := range infos {
				// We need the original text to map offsets; best effort: use docs cache
				text := s.docs[uri]
				l0, c0 := utf16LineCharFromOffset(text, si.Span.Start.Offset)
				l1, c1 := utf16LineCharFromOffset(text, si.Span.End.Offset)
				results = append(results, map[string]any{
					"name": name,
					"kind": si.Kind,
					"location": map[string]any{
						"uri": uri,
						"range": map[string]any{
							"start": map[string]any{"line": l0, "character": c0},
							"end":   map[string]any{"line": l1, "character": c1},
						},
					},
				})
			}
		}
	}
	if len(results) > 200 {
		results = results[:200]
	}
	return results
}

// handleDocumentSymbol returns the outline of symbols in the document.
func (s *Server) handleDocumentSymbol(uri string) []map[string]any {
	text := s.docs[uri]
	if text == "" {
		return []map[string]any{}
	}
	ast := s.astCache[uri]
	if ast == nil {
		lx := lexer.NewWithFilename(text, uri)
		pr := parser.NewParser(lx, uri)
		a, _ := pr.Parse()
		ast = a
		if ast == nil {
			return []map[string]any{}
		}
		s.astCache[uri] = ast
		s.buildSymbolIndex(uri, ast)
	}

	symbols := make([]map[string]any, 0, 32)
	// Walk top-level declarations and produce DocumentSymbol entries
	for _, d := range ast.Declarations {
		switch n := d.(type) {
		case *parser.FunctionDeclaration:
			name := "<anonymous>"
			nameSpan := n.Span
			if n.Name != nil {
				name = n.Name.Value
				nameSpan = n.Name.Span
			}
			kind := 12 // Function
			sL, sC := utf16LineCharFromOffset(text, n.Span.Start.Offset)
			eL, eC := utf16LineCharFromOffset(text, n.Span.End.Offset)
			nsL, nsC := utf16LineCharFromOffset(text, nameSpan.Start.Offset)
			neL, neC := utf16LineCharFromOffset(text, nameSpan.End.Offset)
			symbols = append(symbols, map[string]any{
				"name": name,
				"kind": kind,
				"range": map[string]any{
					"start": map[string]any{"line": sL, "character": sC},
					"end":   map[string]any{"line": eL, "character": eC},
				},
				"selectionRange": map[string]any{
					"start": map[string]any{"line": nsL, "character": nsC},
					"end":   map[string]any{"line": neL, "character": neC},
				},
			})
		case *parser.VariableDeclaration:
			name := "<var>"
			nameSpan := n.Span
			if n.Name != nil {
				name = n.Name.Value
				nameSpan = n.Name.Span
			}
			kind := 13 // Variable
			sL, sC := utf16LineCharFromOffset(text, n.Span.Start.Offset)
			eL, eC := utf16LineCharFromOffset(text, n.Span.End.Offset)
			nsL, nsC := utf16LineCharFromOffset(text, nameSpan.Start.Offset)
			neL, neC := utf16LineCharFromOffset(text, nameSpan.End.Offset)
			symbols = append(symbols, map[string]any{
				"name": name,
				"kind": kind,
				"range": map[string]any{
					"start": map[string]any{"line": sL, "character": sC},
					"end":   map[string]any{"line": eL, "character": eC},
				},
				"selectionRange": map[string]any{
					"start": map[string]any{"line": nsL, "character": nsC},
					"end":   map[string]any{"line": neL, "character": neC},
				},
			})
		}
	}
	return symbols
}

// handleReferences returns all occurrences of the symbol under the given position.
func (s *Server) handleReferences(uri string, line, character int, includeDecl bool) []map[string]any {
	text := s.docs[uri]
	if text == "" {
		return []map[string]any{}
	}
	offset := offsetFromLineCharUTF16(text, line, character)
	if offset < 0 || offset > len(text) {
		return []map[string]any{}
	}
	// Extract identifier at position
	start := offset
	for start > 0 {
		r, size := utf8.DecodeLastRuneInString(text[:start])
		if size <= 0 || !isIdentRune(r) {
			break
		}
		start -= size
	}
	end := offset
	for end < len(text) {
		r, size := utf8.DecodeRuneInString(text[end:])
		if size <= 0 || !isIdentRune(r) {
			break
		}
		end += size
	}
	if start >= end {
		return []map[string]any{}
	}
	name := strings.TrimSpace(text[start:end])
	if name == "" {
		return []map[string]any{}
	}
	results := make([]map[string]any, 0, 16)
	// Determine scope for this symbol at position (best effort)
	var scope *parser.Span
	if entries, ok := s.symIndex[uri][name]; ok && len(entries) > 0 {
		// Prefer the entry whose scope contains the position
		for _, si := range entries {
			if si.Scope.End.Offset > si.Scope.Start.Offset && si.Scope.Start.Offset <= offset && offset < si.Scope.End.Offset {
				sc := si.Scope
				scope = &sc
				break
			}
		}
		if scope == nil {
			sc := entries[0].Scope
			if sc.End.Offset > sc.Start.Offset {
				scope = &sc
			}
		}
	}
	// Include declaration locations if requested
	if includeDecl {
		for _, si := range s.symIndex[uri][name] {
			l0, c0 := utf16LineCharFromOffset(text, si.Span.Start.Offset)
			l1, c1 := utf16LineCharFromOffset(text, si.Span.End.Offset)
			results = append(results, map[string]any{
				"uri": uri,
				"range": map[string]any{
					"start": map[string]any{"line": l0, "character": c0},
					"end":   map[string]any{"line": l1, "character": c1},
				},
			})
		}
	}
	// Scan tokens and collect matches (restrict to scope if available)
	lx := lexer.NewWithFilename(text, uri)
	for {
		tok := lx.NextToken()
		if tok.Type == lexer.TokenEOF {
			break
		}
		if tok.Type == lexer.TokenIdentifier && tok.Literal == name {
			if scope != nil {
				if tok.Span.Start.Offset < scope.Start.Offset || tok.Span.End.Offset > scope.End.Offset {
					continue
				}
			}
			l0, c0 := utf16LineCharFromOffset(text, tok.Span.Start.Offset)
			l1, c1 := utf16LineCharFromOffset(text, tok.Span.End.Offset)
			results = append(results, map[string]any{
				"uri": uri,
				"range": map[string]any{
					"start": map[string]any{"line": l0, "character": c0},
					"end":   map[string]any{"line": l1, "character": c1},
				},
			})
		}
	}
	return results
}

// handleDefinition finds the definition locations for the symbol under the given position.
func (s *Server) handleDefinition(uri string, line, character int) []map[string]any {
	text := s.docs[uri]
	if text == "" {
		return []map[string]any{}
	}
	offset := offsetFromLineCharUTF16(text, line, character)
	if offset < 0 || offset > len(text) {
		return []map[string]any{}
	}
	// Extract identifier at position
	start := offset
	for start > 0 {
		r, size := utf8.DecodeLastRuneInString(text[:start])
		if size <= 0 || !isIdentRune(r) {
			break
		}
		start -= size
	}
	end := offset
	for end < len(text) {
		r, size := utf8.DecodeRuneInString(text[end:])
		if size <= 0 || !isIdentRune(r) {
			break
		}
		end += size
	}
	if start >= end {
		return []map[string]any{}
	}
	name := strings.TrimSpace(text[start:end])
	if name == "" {
		return []map[string]any{}
	}
	entries := s.symIndex[uri][name]
	if len(entries) == 0 {
		return []map[string]any{}
	}
	// Prefer definitions whose scope contains current position
	filtered := make([]SymbolInfo, 0, len(entries))
	for _, si := range entries {
		if si.Scope.End.Offset > si.Scope.Start.Offset && si.Scope.Start.Offset <= offset && offset < si.Scope.End.Offset {
			filtered = append(filtered, si)
		}
	}
	if len(filtered) == 0 {
		filtered = entries
	}
	// Convert spans to LSP locations
	results := make([]map[string]any, 0, len(filtered))
	for _, si := range filtered {
		l0, c0 := utf16LineCharFromOffset(text, si.Span.Start.Offset)
		l1, c1 := utf16LineCharFromOffset(text, si.Span.End.Offset)
		results = append(results, map[string]any{
			"uri": uri,
			"range": map[string]any{
				"start": map[string]any{"line": l0, "character": c0},
				"end":   map[string]any{"line": l1, "character": c1},
			},
		})
	}
	return results
}

// handleTypeDefinition locates the type node under cursor and returns its span as a definition-like location.
func (s *Server) handleTypeDefinition(uri string, line, character int) []map[string]any {
	text := s.docs[uri]
	if text == "" {
		return []map[string]any{}
	}
	offset := offsetFromLineCharUTF16(text, line, character)
	if offset < 0 || offset > len(text) {
		return []map[string]any{}
	}
	ast := s.astCache[uri]
	if ast == nil {
		lx := lexer.NewWithFilename(text, uri)
		pr := parser.NewParser(lx, uri)
		a, _ := pr.Parse()
		ast = a
		s.astCache[uri] = ast
	}
	if ast == nil {
		return []map[string]any{}
	}
	n := findNodeAt(ast, offset)
	if n == nil {
		return []map[string]any{}
	}
	// If the node is a type node, return its span; otherwise, try to find enclosing type node
	var target parser.Node = nil
	switch n.(type) {
	case *parser.StructType, *parser.ArrayType, *parser.FunctionType, *parser.EnumType, *parser.GenericType, *parser.TraitType:
		target = n
	default:
		// ascend to nearest type-ish node by scanning declarations
		target = n
	}
	if target == nil {
		return []map[string]any{}
	}
	l0, c0 := utf16LineCharFromOffset(text, target.GetSpan().Start.Offset)
	l1, c1 := utf16LineCharFromOffset(text, target.GetSpan().End.Offset)
	return []map[string]any{{
		"uri": uri,
		"range": map[string]any{
			"start": map[string]any{"line": l0, "character": c0},
			"end":   map[string]any{"line": l1, "character": c1},
		},
	}}
}

// buildSymbolIndex walks the AST and indexes symbol definitions for the given document.
func (s *Server) buildSymbolIndex(uri string, root *parser.Program) {
	index := make(map[string][]SymbolInfo)
	// Walk program
	var walk func(node interface{})
	walk = func(node interface{}) {
		switch n := node.(type) {
		case *parser.Program:
			for _, d := range n.Declarations {
				walk(d)
			}
		case *parser.FunctionDeclaration:
			if n.Name != nil {
				index[n.Name.Value] = append(index[n.Name.Value], SymbolInfo{
					Name:   n.Name.Value,
					Kind:   12,
					Span:   n.Name.Span,
					Detail: buildFuncSignature(n),
					Scope:  n.Span,
				})
			}
			for _, p := range n.Parameters {
				if p != nil && p.Name != nil {
					index[p.Name.Value] = append(index[p.Name.Value], SymbolInfo{
						Name:   p.Name.Value,
						Kind:   13,
						Span:   p.Name.Span,
						Detail: buildParamDetail(p),
						Scope:  n.Span,
					})
				}
			}
			if n.Body != nil {
				walk(n.Body)
			}
		case *parser.BlockStatement:
			for _, st := range n.Statements {
				walk(st)
			}
		case *parser.VariableDeclaration:
			if n.Name != nil {
				index[n.Name.Value] = append(index[n.Name.Value], SymbolInfo{
					Name:   n.Name.Value,
					Kind:   13,
					Span:   n.Name.Span,
					Detail: buildVarDetail(n),
					Scope:  n.Span,
				})
			}
		case *parser.ExpressionStatement:
			// no symbol
		case *parser.IfStatement:
			if n.ThenStmt != nil {
				walk(n.ThenStmt)
			}
			if n.ElseStmt != nil {
				walk(n.ElseStmt)
			}
		case *parser.WhileStatement:
			if n.Body != nil {
				walk(n.Body)
			}
		default:
			// Other nodes ignored for symbol indexing
		}
	}
	walk(root)
	s.symIndex[uri] = index
}

// updateWorkspaceIndex rebuilds the global index entries for a given document
func (s *Server) updateWorkspaceIndex(uri string, root *parser.Program) {
	// Remove existing entries for this uri
	s.removeDocFromWorkspaceIndex(uri)
	// Add fresh ones from current AST
	var add func(node interface{})
	add = func(node interface{}) {
		switch n := node.(type) {
		case *parser.Program:
			for _, d := range n.Declarations {
				add(d)
			}
		case *parser.FunctionDeclaration:
			if n.Name != nil {
				s.wsIndex[n.Name.Value] = append(s.wsIndex[n.Name.Value], SymbolLocation{URI: uri, Span: n.Name.Span, Kind: 12})
			}
			for _, p := range n.Parameters {
				if p != nil && p.Name != nil {
					s.wsIndex[p.Name.Value] = append(s.wsIndex[p.Name.Value], SymbolLocation{URI: uri, Span: p.Name.Span, Kind: 13})
				}
			}
			if n.Body != nil {
				// Walk block for local vars
				add(n.Body)
			}
		case *parser.BlockStatement:
			for _, st := range n.Statements {
				add(st)
			}
		case *parser.VariableDeclaration:
			if n.Name != nil {
				s.wsIndex[n.Name.Value] = append(s.wsIndex[n.Name.Value], SymbolLocation{URI: uri, Span: n.Name.Span, Kind: 13})
			}
		}
	}
	add(root)
}

func (s *Server) removeDocFromWorkspaceIndex(uri string) {
	if len(s.wsIndex) == 0 {
		return
	}
	for name, locs := range s.wsIndex {
		keep := locs[:0]
		for _, loc := range locs {
			if loc.URI != uri {
				keep = append(keep, loc)
			}
		}
		if len(keep) == 0 {
			delete(s.wsIndex, name)
		} else {
			s.wsIndex[name] = keep
		}
	}
}

// handleCrossFileReferences returns references of the symbol under position in other open documents
func (s *Server) handleCrossFileReferences(uri string, line, character int) []map[string]any {
	text := s.docs[uri]
	if text == "" {
		return []map[string]any{}
	}
	offset := offsetFromLineCharUTF16(text, line, character)
	if offset < 0 || offset > len(text) {
		return []map[string]any{}
	}
	// Identify name at position
	start := offset
	for start > 0 {
		r, sz := utf8.DecodeLastRuneInString(text[:start])
		if sz <= 0 || !isIdentRune(r) {
			break
		}
		start -= sz
	}
	end := offset
	for end < len(text) {
		r, sz := utf8.DecodeRuneInString(text[end:])
		if sz <= 0 || !isIdentRune(r) {
			break
		}
		end += sz
	}
	if start >= end {
		return []map[string]any{}
	}
	name := strings.TrimSpace(text[start:end])
	if name == "" {
		return []map[string]any{}
	}
	results := make([]map[string]any, 0, 16)
	for _, loc := range s.wsIndex[name] {
		if loc.URI == uri {
			continue
		}
		t := s.docs[loc.URI]
		if t == "" {
			continue
		}
		l0, c0 := utf16LineCharFromOffset(t, loc.Span.Start.Offset)
		l1, c1 := utf16LineCharFromOffset(t, loc.Span.End.Offset)
		results = append(results, map[string]any{
			"uri": loc.URI,
			"range": map[string]any{
				"start": map[string]any{"line": l0, "character": c0},
				"end":   map[string]any{"line": l1, "character": c1},
			},
		})
	}
	return results
}

// handleCompletion returns completion items based on current document content and position.
// It provides keyword suggestions and simple identifier suggestions in scope-like proximity.
func (s *Server) handleCompletion(uri string, line, character int) []map[string]any {
	text := s.docs[uri]
	if text == "" {
		return []map[string]any{}
	}

	// Determine current token prefix at position (UTF-16 -> byte offset)
	offset := offsetFromLineCharUTF16(text, line, character)
	if offset < 0 || offset > len(text) {
		offset = len(text)
	}

	prefixStart := offset
	for prefixStart > 0 {
		r, size := utf8.DecodeLastRuneInString(text[:prefixStart])
		if size <= 0 {
			break
		}
		if !isIdentRune(r) {
			break
		}
		prefixStart -= size
	}
	prefix := strings.TrimSpace(text[prefixStart:offset])

	// Collect candidates: language keywords and recent identifiers in the document
	keywordItems := make([]map[string]any, 0, 32)
	for kw, meta := range keywordMeta {
		if prefix == "" || strings.HasPrefix(kw, prefix) {
			item := map[string]any{
				"label": kw,
				"kind":  14, // Keyword
				"data":  map[string]any{"type": "keyword", "label": kw},
			}
			if meta.Detail != "" {
				item["detail"] = meta.Detail
			}
			if meta.Documentation != "" {
				item["documentation"] = meta.Documentation
			}
			if meta.Snippet != "" {
				item["insertText"] = meta.Snippet
				item["insertTextFormat"] = 2 // Snippet
			}
			keywordItems = append(keywordItems, item)
		}
	}

	// Scan identifiers in the document using lexer; dedupe and rank by occurrence
	lx := lexer.NewWithFilename(text, uri)
	freq := map[string]int{}
	for {
		tok := lx.NextToken()
		if tok.Type == lexer.TokenEOF {
			break
		}
		if tok.Type == lexer.TokenIdentifier {
			name := strings.TrimSpace(tok.Literal)
			if name == "" {
				continue
			}
			if prefix == "" || strings.HasPrefix(name, prefix) {
				freq[name]++
			}
		}
	}

	identItems := make([]map[string]any, 0, len(freq))
	// Convert to slice and sort by frequency desc, then lexicographically
	names := make([]string, 0, len(freq))
	for n := range freq {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		if freq[names[i]] == freq[names[j]] {
			return names[i] < names[j]
		}
		return freq[names[i]] > freq[names[j]]
	})
	for _, n := range names {
		item := map[string]any{
			"label": n,
			"kind":  6, // Variable (default)
		}
		// If we know the symbol kind/detail from index, use it and attach resolve data
		if entries, ok := s.symIndex[uri][n]; ok && len(entries) > 0 {
			info := entries[0]
			if info.Kind != 0 {
				item["kind"] = info.Kind
			}
			if info.Detail != "" {
				item["detail"] = info.Detail
			}
			// Offer snippet for functions
			if info.Kind == 12 {
				item["insertTextFormat"] = 2
				item["insertText"] = n + "($0)"
			}
			item["data"] = map[string]any{
				"type": "symbol",
				"uri":  uri,
				"name": n,
			}
		}
		identItems = append(identItems, item)
	}

	// Workspace symbols (type/import感知の土台として、ワークスペース索引から名前候補)
	wsItems := make([]map[string]any, 0, 64)
	for name, locs := range s.wsIndex {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			continue
		}
		if len(locs) == 0 {
			continue
		}
		kind := locs[0].Kind
		detail := "workspace symbol"
		// Try to enrich detail from any indexed document that has a symbol info
		for _, l := range locs {
			if idx, ok := s.symIndex[l.URI]; ok {
				if entries, ok2 := idx[name]; ok2 && len(entries) > 0 {
					if entries[0].Detail != "" {
						detail = entries[0].Detail
						break
					}
				}
			}
		}
		item := map[string]any{
			"label":  name,
			"kind":   kind,
			"detail": detail,
			"data": map[string]any{"type": "symbol", "uris": func() []string {
				m := map[string]struct{}{}
				u := make([]string, 0, len(locs))
				for _, l := range locs {
					if _, ok := m[l.URI]; !ok {
						m[l.URI] = struct{}{}
						u = append(u, l.URI)
					}
				}
				return u
			}()},
		}
		// Offer snippet insert for functions
		if kind == 12 {
			item["insertTextFormat"] = 2
			item["insertText"] = name + "($0)"
		}
		wsItems = append(wsItems, item)
		if len(wsItems) >= 64 {
			break
		}
	}

	// Combine keyword, identifier and workspace items; dedupe by label
	items := make([]map[string]any, 0, len(keywordItems)+len(identItems)+len(wsItems))
	seen := map[string]bool{}
	for _, col := range [][]map[string]any{keywordItems, identItems, wsItems} {
		for _, it := range col {
			if lab, _ := it["label"].(string); lab != "" {
				if seen[lab] {
					continue
				}
				seen[lab] = true
			}
			// Attach a concise type summary to detail if missing (from AST/symbol index)
			if _, ok := it["detail"]; !ok {
				if lab, _ := it["label"].(string); lab != "" {
					if entries, ok2 := s.symIndex[uri][lab]; ok2 && len(entries) > 0 {
						if d := entries[0].Detail; d != "" {
							it["detail"] = d
						}
					}
				}
			}
			items = append(items, it)
		}
	}
	if len(items) > 200 {
		items = items[:200]
	}
	return items
}

// isIdentRune checks if a rune can be part of an identifier.
func isIdentRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// keywordMeta holds language keywords with optional details and snippets.
type keywordInfo struct {
	Detail        string
	Documentation string
	Snippet       string
}

var keywordMeta = map[string]keywordInfo{
	"fn":       {Detail: "function (alias)", Snippet: "func ${1:name}(${2}) ${3} {\n\t$0\n}"},
	"func":     {Detail: "function", Snippet: "func ${1:name}(${2}) ${3} {\n\t$0\n}"},
	"let":      {Detail: "binding"},
	"var":      {Detail: "mutable binding"},
	"const":    {Detail: "constant"},
	"struct":   {Detail: "structure"},
	"enum":     {Detail: "enumeration"},
	"trait":    {Detail: "trait"},
	"impl":     {Detail: "implementation"},
	"if":       {Detail: "conditional"},
	"else":     {Detail: "conditional"},
	"for":      {Detail: "loop"},
	"while":    {Detail: "loop"},
	"loop":     {Detail: "loop"},
	"match":    {Detail: "pattern match"},
	"return":   {Detail: "return"},
	"break":    {Detail: "break"},
	"continue": {Detail: "continue"},
	"async":    {Detail: "asynchronous"},
	"await":    {Detail: "await"},
	"actor":    {Detail: "actor"},
	"spawn":    {Detail: "spawn actor"},
	"import":   {Detail: "import module"},
	"export":   {Detail: "export symbol"},
	"module":   {Detail: "module"},
	"pub":      {Detail: "public"},
	"mut":      {Detail: "mutable"},
	"as":       {Detail: "alias"},
	"in":       {Detail: "in"},
	"where":    {Detail: "where clause"},
	"unsafe":   {Detail: "unsafe block"},
	"macro":    {Detail: "macro"},
	"true":     {Detail: "boolean true"},
	"false":    {Detail: "boolean false"},
}

// offsetFromLineCharUTF16 converts a LSP position (UTF-16 units) to byte offset.
func offsetFromLineCharUTF16(text string, line, character int) int {
	if line < 0 || character < 0 {
		return -1
	}
	// Move to line start
	ln := 0
	idx := 0
	for idx < len(text) && ln < line {
		if text[idx] == '\n' {
			ln++
		}
		idx++
	}
	if ln != line {
		return -1
	}
	// Count UTF-16 code units within the line
	units := 0
	for idx < len(text) && units < character {
		if text[idx] == '\n' {
			break
		}
		r, size := utf8.DecodeRuneInString(text[idx:])
		if size <= 0 {
			size = 1
		}
		// UTF-16 units per rune
		if r <= 0xFFFF {
			units += 1
		} else {
			units += 2
		}
		idx += size
	}
	return idx
}

// utf16LineCharFromOffset converts byte offset to LSP line and UTF-16 character (0-based).
func utf16LineCharFromOffset(text string, offset int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}
	line := 0
	units := 0
	i := 0
	for i < offset {
		if text[i] == '\n' {
			line++
			units = 0
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(text[i:])
		if size <= 0 {
			size = 1
		}
		if r <= 0xFFFF {
			units += 1
		} else {
			units += 2
		}
		i += size
	}
	return line, units
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
