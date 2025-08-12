package lsp

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

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
}

func NewServer(r io.Reader, w io.Writer) *Server {
	return &Server{
		in:       bufio.NewReader(r),
		out:      w,
		docs:     make(map[string]string),
		symIndex: make(map[string]map[string][]SymbolInfo),
		astCache: make(map[string]*parser.Program),
	}
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
		for {
			line, err := s.in.ReadString('\n')
			if err != nil {
				return err
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
		if contentLength <= 0 {
			continue
		}
		buf := make([]byte, contentLength)
		if _, err := io.ReadFull(s.in, buf); err != nil {
			return err
		}
		var req Request
		if err := json.Unmarshal(buf, &req); err != nil {
			continue
		}
		switch req.Method {
		case "initialize":
			// Advertise capabilities and server info per LSP. Use UTF-16 positions for compatibility.
			caps := map[string]any{
				"positionEncoding": "utf-16",
				"textDocumentSync": map[string]any{
					"openClose": true,
					"change":    1, // TextDocumentSyncKind: Incremental(2) not yet; use Full(1)
				},
				"completionProvider": map[string]any{
					"triggerCharacters": []string{".", ":", ",", "(", "[", " "},
				},
				"hoverProvider":              true,
				"definitionProvider":         true,
				"referencesProvider":         true,
				"documentSymbolProvider":     true,
				"codeActionProvider":         true,
				"documentFormattingProvider": true,
				"signatureHelpProvider":      map[string]any{"triggerCharacters": []string{"(", ","}},
				"documentOnTypeFormattingProvider": map[string]any{
					"firstTriggerCharacter": "}",
					"moreTriggerCharacter":  []string{"\n", ")", ";"},
				},
				"documentHighlightProvider": true,
				"foldingRangeProvider":      true,
				"renameProvider":            true,
				"workspaceSymbolProvider":   true,
			}
			result := map[string]any{
				"capabilities": caps,
				"serverInfo": map[string]any{
					"name":    "orizon-lsp",
					"version": "dev",
				},
			}
			s.reply(req.ID, result)
		case "initialized":
			// This is a notification; do not send a response.
			if len(req.ID) > 0 {
				// Only reply if the client incorrectly sent an id
				s.reply(req.ID, nil)
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
			_ = json.Unmarshal(req.Params, &p)
			if p.TextDocument.URI != "" {
				delete(s.docs, p.TextDocument.URI)
				delete(s.symIndex, p.TextDocument.URI)
				// Clear diagnostics on close
				s.notify("textDocument/publishDiagnostics", map[string]any{
					"uri":         p.TextDocument.URI,
					"diagnostics": []any{},
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
				} `json:"textDocument"`
			}
			_ = json.Unmarshal(req.Params, &p)
			if p.TextDocument.URI != "" {
				s.docs[p.TextDocument.URI] = p.TextDocument.Text
				// Trigger diagnostics on open
				s.publishDiagnosticsFor(p.TextDocument.URI, p.TextDocument.Text)
			}
			if len(req.ID) > 0 {
				s.reply(req.ID, nil)
			}
		case "textDocument/didChange":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				ContentChanges []struct {
					Text string `json:"text"`
				} `json:"contentChanges"`
			}
			_ = json.Unmarshal(req.Params, &p)
			if len(p.ContentChanges) > 0 {
				s.docs[p.TextDocument.URI] = p.ContentChanges[len(p.ContentChanges)-1].Text
				// Trigger diagnostics on change
				s.publishDiagnosticsFor(p.TextDocument.URI, s.docs[p.TextDocument.URI])
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
			_ = json.Unmarshal(req.Params, &p)
			result := s.handleHover(p.TextDocument.URI, p.Position.Line, p.Position.Character)
			s.reply(req.ID, result)
		case "textDocument/definition":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position struct{ Line, Character int } `json:"position"`
			}
			_ = json.Unmarshal(req.Params, &p)
			defs := s.handleDefinition(p.TextDocument.URI, p.Position.Line, p.Position.Character)
			s.reply(req.ID, defs)
		case "textDocument/documentSymbol":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
			}
			_ = json.Unmarshal(req.Params, &p)
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
			_ = json.Unmarshal(req.Params, &p)
			refs := s.handleReferences(p.TextDocument.URI, p.Position.Line, p.Position.Character, p.Context.IncludeDeclaration)
			s.reply(req.ID, refs)
		case "textDocument/documentHighlight":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position struct{ Line, Character int } `json:"position"`
			}
			_ = json.Unmarshal(req.Params, &p)
			hl := s.handleDocumentHighlight(p.TextDocument.URI, p.Position.Line, p.Position.Character)
			s.reply(req.ID, hl)
		case "textDocument/foldingRange":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
			}
			_ = json.Unmarshal(req.Params, &p)
			fr := s.handleFoldingRange(p.TextDocument.URI)
			s.reply(req.ID, fr)
		case "textDocument/prepareRename":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position struct{ Line, Character int } `json:"position"`
			}
			_ = json.Unmarshal(req.Params, &p)
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
			_ = json.Unmarshal(req.Params, &p)
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
			_ = json.Unmarshal(req.Params, &p)
			syms := s.handleWorkspaceSymbol(p.Query)
			s.reply(req.ID, syms)
		case "textDocument/signatureHelp":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Position struct{ Line, Character int } `json:"position"`
			}
			_ = json.Unmarshal(req.Params, &p)
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
			_ = json.Unmarshal(req.Params, &p)
			edits := s.handleOnTypeFormatting(p.TextDocument.URI, p.Position.Line, p.Position.Character, p.Ch, p.Options)
			s.reply(req.ID, edits)
		case "textDocument/formatting":
			var p struct {
				TextDocument struct {
					URI string `json:"uri"`
				} `json:"textDocument"`
				Options map[string]any `json:"options"`
			}
			_ = json.Unmarshal(req.Params, &p)
			edits := s.handleFormatting(p.TextDocument.URI, p.Options)
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
			_ = json.Unmarshal(req.Params, &p)
			actions := s.handleCodeAction(p.TextDocument.URI)
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
}

func (s *Server) write(v any) {
	data, _ := json.Marshal(v)
	// LSP framing (ignore write errors intentionally after best effort)
	_, _ = io.WriteString(s.out, "Content-Length: ")
	_, _ = io.WriteString(s.out, itoa(len(data)))
	_, _ = io.WriteString(s.out, "\r\n\r\n")
	_, _ = s.out.Write(data)
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

	s.notify("textDocument/publishDiagnostics", map[string]any{
		"uri":         uri,
		"diagnostics": diags,
	})
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
	// Append simple AST-based description when available
	if ast := s.astCache[uri]; ast != nil {
		if desc := describeNodeAt(ast, offset); desc != "" {
			hoverVal += "\n\n" + desc
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

// handleFormatting applies a simple formatting by re-printing the AST or falling back to trimming whitespace.
func (s *Server) handleFormatting(uri string, options map[string]any) []map[string]any {
	// Lightweight, safe formatter: trim trailing spaces and re-indent braces-based blocks
	text := s.docs[uri]
	if text == "" {
		return []map[string]any{}
	}
	lines := strings.Split(text, "\n")
	indent := 0
	indentUnit := "  " // default 2 spaces
	if ws, ok := options["insertSpaces"].(bool); ok && !ws {
		indentUnit = "\t"
	} else if size, ok := options["tabSize"].(float64); ok && size > 0 {
		indentUnit = strings.Repeat(" ", int(size))
	}
	for i, line := range lines {
		// Remove trailing whitespace
		raw := strings.TrimRight(line, " \t\r")

		// Compute delta for current line: if it starts with '}', outdent first
		trimmedLeft := strings.TrimLeft(raw, " \t")
		leadingClose := strings.HasPrefix(trimmedLeft, "}")
		// Adjust indent before applying
		if leadingClose && indent > 0 {
			indent--
		}
		// Apply indent
		lines[i] = strings.Repeat(indentUnit, indent) + trimmedLeft

		// Update indent for next line based on braces balance of this line
		open := strings.Count(trimmedLeft, "{")
		close := strings.Count(trimmedLeft, "}")
		indent += open
		if indent < 0 {
			indent = 0
		}
		if close > open {
			// If more closing braces, reduce indent but not below zero
			diff := close - open
			if indent >= diff {
				indent -= diff
			} else {
				indent = 0
			}
		}
	}
	formatted := strings.Join(lines, "\n")
	if formatted == text {
		return []map[string]any{}
	}
	endLine, endChar := utf16LineCharFromOffset(text, len(text))
	return []map[string]any{
		{
			"range": map[string]any{
				"start": map[string]any{"line": 0, "character": 0},
				"end":   map[string]any{"line": endLine, "character": endChar},
			},
			"newText": formatted,
		},
	}
}

// handleRangeFormatting applies formatting to a subset of lines.
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
	startOff, _ := getLineBounds(text, startLine)
	_, endOff := getLineBounds(text, endLine)
	if startOff < 0 || endOff < 0 {
		return []map[string]any{}
	}
	// Format whole document, then slice the changed span (simple but consistent)
	full := s.handleFormatting(uri, options)
	if len(full) == 0 {
		return []map[string]any{}
	}
	// Replace the full text edit with a range-limited edit using the newly formatted content segment
	newTextAny := full[0]["newText"]
	newDoc, _ := newTextAny.(string)
	if newDoc == "" {
		return []map[string]any{}
	}
	// Extract the segment for the requested line range
	seg := newDoc[startOff:]
	if endOff <= len(newDoc) {
		seg = newDoc[startOff:endOff]
	}
	sL, sC := utf16LineCharFromOffset(text, startOff)
	eL, eC := utf16LineCharFromOffset(text, endOff)
	return []map[string]any{{
		"range": map[string]any{
			"start": map[string]any{"line": sL, "character": sC},
			"end":   map[string]any{"line": eL, "character": eC},
		},
		"newText": seg,
	}}
}

// handleCodeAction provides simple quick fixes like inserting missing function body braces or removing trailing spaces.
func (s *Server) handleCodeAction(uri string) []map[string]any {
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
	sig := map[string]any{"label": label}
	return map[string]any{
		"signatures":      []any{sig},
		"activeSignature": 0,
		"activeParameter": activeParam,
	}
}

// handleOnTypeFormatting performs small edits on specific typed characters (e.g., '}' or newline).
func (s *Server) handleOnTypeFormatting(uri string, line, character int, ch string, options map[string]any) []map[string]any {
	if ch != "}" && ch != "\n" && ch != ")" && ch != ";" {
		return []map[string]any{}
	}
	text := s.docs[uri]
	if text == "" {
		return []map[string]any{}
	}
	// Compute desired indent for the current line and rebuild line content
	startOff, endOff := getLineBounds(text, line)
	if startOff < 0 || endOff < 0 {
		return []map[string]any{}
	}
	lineText := text[startOff:endOff]
	// Desired indent based on brace balance up to this line
	indentLevel := computeIndentBeforeLine(text, line)
	// If line starts with '}', reduce indent for this line
	trimmed := strings.TrimLeft(lineText, " \t")
	if strings.HasPrefix(trimmed, "}") && indentLevel > 0 {
		indentLevel--
	}
	indentUnit := "  "
	if ws, ok := options["insertSpaces"].(bool); ok && !ws {
		indentUnit = "\t"
	} else if size, ok := options["tabSize"].(float64); ok && size > 0 {
		indentUnit = strings.Repeat(" ", int(size))
	}
	newLine := strings.Repeat(indentUnit, indentLevel) + strings.TrimRight(trimmed, " \t")
	if newLine == lineText {
		return []map[string]any{}
	}
	// Produce single-line edit
	sL, sC := utf16LineCharFromOffset(text, startOff)
	eL, eC := utf16LineCharFromOffset(text, endOff)
	return []map[string]any{{
		"range": map[string]any{
			"start": map[string]any{"line": sL, "character": sC},
			"end":   map[string]any{"line": eL, "character": eC},
		},
		"newText": newLine,
	}}
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
	return map[string]any{
		"changes": map[string]any{
			uri: edits,
		},
	}, ""
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
		// If we know the symbol kind/detail from index, use it
		if entries, ok := s.symIndex[uri][n]; ok && len(entries) > 0 {
			info := entries[0]
			if info.Kind != 0 {
				item["kind"] = info.Kind
			}
			if info.Detail != "" {
				item["detail"] = info.Detail
			}
		}
		identItems = append(identItems, item)
	}

	// Combine keyword items and identifier items; cap to a reasonable size
	items := append(keywordItems, identItems...)
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
