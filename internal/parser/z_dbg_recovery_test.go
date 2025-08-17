package parser

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/lexer"
)

func Test__DBG_ErrorToleranceTrace(t *testing.T) {
	input := `
    func good1() { return 1; }
    
    func bad syntax here invalid
    
    func good2() { return 2; }
    
    let invalid = ;
    
    let valid = 42;
    `

	t.Log("-- Tokens --")
	lx := lexer.New(input)
	for {
		tok := lx.NextToken()
		t.Logf("tok=%v lit=%q", tok.Type, tok.Literal)
		if tok.Type == lexer.TokenEOF {
			break
		}
	}

	l := lexer.New(input)
	p := NewParser(l, "dbg.oriz")
	prog, errs := p.Parse()
	t.Logf("errs=%d decls=%d", len(errs), len(prog.Declarations))
	for i, d := range prog.Declarations {
		switch dd := d.(type) {
		case *FunctionDeclaration:
			t.Logf("decl[%d]=func %s", i, dd.Name.Value)
		case *VariableDeclaration:
			t.Logf("decl[%d]=var %s", i, dd.Name.Value)
		default:
			t.Logf("decl[%d]=%T", i, d)
		}
	}
}

func Test__DBG_ErrorToleranceTrace_FromIntegration(t *testing.T) {
	input := `
	func good1() { return 1; }
	
	func bad syntax here invalid
	
	func good2() { return 2; }
	
	let invalid = ;
	
	let valid = 42;
	`

	t.Log("-- Tokens (integration style) --")
	lx := lexer.New(input)
	for {
		tok := lx.NextToken()
		t.Logf("tok=%v lit=%q", tok.Type, tok.Literal)
		if tok.Type == lexer.TokenEOF {
			break
		}
	}

	l := lexer.New(input)
	p := NewParser(l, "dbg2.oriz")
	prog, errs := p.Parse()
	t.Logf("errs=%d decls=%d", len(errs), len(prog.Declarations))
	for i, d := range prog.Declarations {
		switch dd := d.(type) {
		case *FunctionDeclaration:
			t.Logf("decl[%d]=func %s", i, dd.Name.Value)
		case *VariableDeclaration:
			t.Logf("decl[%d]=var %s", i, dd.Name.Value)
		default:
			t.Logf("decl[%d]=%T", i, d)
		}
	}
}

func Test__DBG_ParseSingleValidFunc(t *testing.T) {
	input := `func valid() { return 42; }`
	l := lexer.New(input)
	p := NewParser(l, "single.oriz")
	prog, errs := p.Parse()
	t.Logf("errs=%d decls=%d", len(errs), len(prog.Declarations))
	if len(prog.Declarations) == 0 {
		t.Fatalf("failed to parse single valid function")
	}
}

func Test__DBG_BadThenGood(t *testing.T) {
	input := `
    func bad syntax here invalid
    func good() { return 3; }
    `
	lx := lexer.New(input)
	t.Log("-- Tokens (bad then good) --")
	for {
		tok := lx.NextToken()
		t.Logf("tok=%v lit=%q", tok.Type, tok.Literal)
		if tok.Type == lexer.TokenEOF {
			break
		}
	}
	l := lexer.New(input)
	p := NewParser(l, "dbg3.oriz")
	prog, errs := p.Parse()
	t.Logf("errs=%d decls=%d", len(errs), len(prog.Declarations))
	for i, d := range prog.Declarations {
		switch dd := d.(type) {
		case *FunctionDeclaration:
			t.Logf("decl[%d]=func %s", i, dd.Name.Value)
		default:
			t.Logf("decl[%d]=%T", i, d)
		}
	}
}

func Test__DBG_StepParseBadThenGood(t *testing.T) {
	input := `
    func bad syntax here invalid
    func good() { return 3; }
    `
	l := lexer.New(input)
	p := NewParser(l, "dbg4.oriz")
	decls := 0
	steps := 0
	for !p.currentTokenIs(lexer.TokenEOF) && steps < 50 {
		steps++
		if p.currentTokenIs(lexer.TokenWhitespace) || p.currentTokenIs(lexer.TokenComment) || p.currentTokenIs(lexer.TokenNewline) {
			t.Logf("skip trivia: %v %q", p.current.Type, p.current.Literal)
			p.nextToken()
			continue
		}
		t.Logf("top: current=%v(%q) peek=%v(%q)", p.current.Type, p.current.Literal, p.peek.Type, p.peek.Literal)
		if d := p.parseDeclaration(); d != nil {
			decls++
			switch dd := d.(type) {
			case *FunctionDeclaration:
				t.Logf("decl: func %s", dd.Name.Value)
			default:
				t.Logf("decl: %T", d)
			}
			p.nextToken()
		} else {
			t.Logf("decl=nil at current=%v(%q) peek=%v(%q)", p.current.Type, p.current.Literal, p.peek.Type, p.peek.Literal)
			// Emulate parseProgram behavior after failure
			continue
		}
	}
	t.Logf("done: decls=%d errors=%d", decls, len(p.errors))
}
