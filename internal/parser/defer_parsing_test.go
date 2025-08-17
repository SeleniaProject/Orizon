package parser

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/lexer"
)

func TestParseDeferBlock(t *testing.T) {
	input := `
    func f() {
        defer { return; }
    }`
	l := lexer.New(input)
	p := NewParser(l, "test_defer.oriz")
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	fn := program.Declarations[0].(*FunctionDeclaration)
	if len(fn.Body.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(fn.Body.Statements))
	}
	if _, ok := fn.Body.Statements[0].(*DeferStatement); !ok {
		t.Fatalf("expected DeferStatement, got %T", fn.Body.Statements[0])
	}
}

func TestParseDeferExpr(t *testing.T) {
	input := `
    func f() {
        defer log(1);
    }`
	l := lexer.New(input)
	p := NewParser(l, "test_defer2.oriz")
	program, errs := p.Parse()
	if len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	fn := program.Declarations[0].(*FunctionDeclaration)
	if len(fn.Body.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(fn.Body.Statements))
	}
	if _, ok := fn.Body.Statements[0].(*DeferStatement); !ok {
		t.Fatalf("expected DeferStatement, got %T", fn.Body.Statements[0])
	}
}
