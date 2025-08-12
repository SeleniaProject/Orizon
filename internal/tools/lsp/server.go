package lsp

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
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
	in  *bufio.Reader
	out io.Writer
    docs map[string]string
}

func NewServer(r io.Reader, w io.Writer) *Server { return &Server{in: bufio.NewReader(r), out: w, docs: make(map[string]string)} }

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
			if len(line) >= 16 && (line[:15] == "Content-Length" || line[:15] == "content-length") {
				// Content-Length: N\r\n
				var n int
				for i := 0; i < len(line); i++ {
					if line[i] >= '0' && line[i] <= '9' {
						n = n*10 + int(line[i]-'0')
					}
				}
				contentLength = n
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
			s.reply(req.ID, map[string]any{"capabilities": map[string]any{"textDocumentSync": 1, "completionProvider": map[string]any{}}})
        case "initialized":
            s.reply(req.ID, nil)
		case "shutdown":
			s.reply(req.ID, nil)
		case "exit":
			return nil
        case "textDocument/didOpen":
            var p struct{ TextDocument struct{ URI string `json:"uri"`; Text string `json:"text"` } `json:"textDocument"` }
            _ = json.Unmarshal(req.Params, &p)
            if p.TextDocument.URI != "" { s.docs[p.TextDocument.URI] = p.TextDocument.Text }
            if len(req.ID) > 0 { s.reply(req.ID, nil) }
        case "textDocument/didChange":
            var p struct{ TextDocument struct{ URI string `json:"uri"` } `json:"textDocument"`; ContentChanges []struct{ Text string `json:"text"` } `json:"contentChanges"` }
            _ = json.Unmarshal(req.Params, &p)
            if len(p.ContentChanges) > 0 { s.docs[p.TextDocument.URI] = p.ContentChanges[len(p.ContentChanges)-1].Text }
            if len(req.ID) > 0 { s.reply(req.ID, nil) }
        case "textDocument/completion":
            type item struct{ Label string `json:"label"`; Kind int `json:"kind,omitempty"` }
            s.reply(req.ID, map[string]any{"isIncomplete": false, "items": []item{{Label: "fn"}, {Label: "let"}, {Label: "type"}}})
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
	// LSP framing
	io.WriteString(s.out, "Content-Length: ")
	io.WriteString(s.out, itoa(len(data)))
	io.WriteString(s.out, "\r\n\r\n")
	s.out.Write(data)
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
