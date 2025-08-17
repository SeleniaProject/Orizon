package parser

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/lexer"
)

func TestParseLoopBasic(t *testing.T) {
	input := `
    func f() {
        loop {
            break;
        }
    }`

	l := lexer.New(input)
	p := NewParser(l, "test_loop.oriz")
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
	if len(fn.Body.Statements) != 1 {
		t.Fatalf("expected 1 statement in body, got %d", len(fn.Body.Statements))
	}
	ws, ok := fn.Body.Statements[0].(*WhileStatement)
	if !ok {
		t.Fatalf("expected WhileStatement (lowered loop), got %T", fn.Body.Statements[0])
	}
	lit, ok := ws.Condition.(*Literal)
	if !ok || lit.Kind != LiteralBool || lit.Value != true {
		t.Fatalf("expected true literal condition, got %#v", ws.Condition)
	}
}
