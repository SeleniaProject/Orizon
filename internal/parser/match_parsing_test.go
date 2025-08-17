package parser

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/lexer"
)

func TestParseMatchBasic(t *testing.T) {
	input := `
    func f(x: int) {
        match (x) {
            0 => return 0,
            1 => { return 1; },
        }
    }`

	l := lexer.New(input)
	p := NewParser(l, "test_match.oriz")

	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	if len(program.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(program.Declarations))
	}
	fn, ok := program.Declarations[0].(*FunctionDeclaration)
	if !ok {
		t.Fatalf("expected FunctionDeclaration, got %T", program.Declarations[0])
	}
	if fn.Body == nil || len(fn.Body.Statements) != 1 {
		t.Fatalf("expected 1 statement in body, got %d", len(fn.Body.Statements))
	}
	m, ok := fn.Body.Statements[0].(*MatchStatement)
	if !ok {
		t.Fatalf("expected MatchStatement, got %T", fn.Body.Statements[0])
	}
	if m.Expression == nil {
		t.Fatalf("expected scrutinee expression, got nil")
	}
	if len(m.Arms) != 2 {
		t.Fatalf("expected 2 arms, got %d", len(m.Arms))
	}
}

func TestParseMatchWithGuardAndNoCommas(t *testing.T) {
	input := `
    func f(x: int) {
        match (x) {
            2 if x > 1 => return 2
            3 => return 3
        }
    }`

	l := lexer.New(input)
	p := NewParser(l, "test_match2.oriz")

	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	fn, _ := program.Declarations[0].(*FunctionDeclaration)
	m, ok := fn.Body.Statements[0].(*MatchStatement)
	if !ok {
		t.Fatalf("expected MatchStatement, got %T", fn.Body.Statements[0])
	}
	if len(m.Arms) != 2 {
		t.Fatalf("expected 2 arms, got %d", len(m.Arms))
	}
	if m.Arms[0].Guard == nil {
		t.Fatalf("expected guard on first arm")
	}
}
