package astbridge

import (
	"testing"

	aast "github.com/orizon-lang/orizon/internal/ast"
	p "github.com/orizon-lang/orizon/internal/parser"
)

// Test that function parameter name spans are preserved through Parser -> AST -> Parser roundtrip.
func TestParameterNameSpan_Preserved_RoundTrip(t *testing.T) {
	// Construct a parser.FunctionDeclaration with explicit spans
	paramName := &p.Identifier{Value: "arg", Span: p.Span{Start: p.Position{Line: 3, Column: 7}, End: p.Position{Line: 3, Column: 10}}}
	param := &p.Parameter{Span: p.Span{Start: p.Position{Line: 3, Column: 5}, End: p.Position{Line: 3, Column: 15}}, Name: paramName, TypeSpec: &p.BasicType{Name: "int"}}
	fn := &p.FunctionDeclaration{
		Span:       p.Span{Start: p.Position{Line: 2, Column: 1}, End: p.Position{Line: 4, Column: 1}},
		Name:       &p.Identifier{Value: "foo", Span: p.Span{Start: p.Position{Line: 2, Column: 6}, End: p.Position{Line: 2, Column: 9}}},
		Parameters: []*p.Parameter{param},
		ReturnType: nil,
		Body:       &p.BlockStatement{},
	}
	pprog := &p.Program{Declarations: []p.Declaration{fn}}

	// Convert to core AST.
	ap, err := FromParserProgram(pprog)
	if err != nil {
		t.Fatalf("FromParserProgram error: %v", err)
	}

	afd, ok := ap.Declarations[0].(*aast.FunctionDeclaration)
	if !ok {
		t.Fatalf("expected FunctionDeclaration, got %T", ap.Declarations[0])
	}
	// Verify parameter name span preserved.
	if len(afd.Parameters) != 1 {
		t.Fatalf("expected 1 parameter")
	}

	got := afd.Parameters[0].Name.Span
	if got.Start.Line != 3 || got.Start.Column != 7 || got.End.Line != 3 || got.End.Column != 10 {
		t.Fatalf("parameter name span not preserved in AST: %+v", got)
	}

	// Round-trip back to parser and verify again.
	pback, err := ToParserProgram(ap)
	if err != nil {
		t.Fatalf("ToParserProgram error: %v", err)
	}

	pfn, ok := pback.Declarations[0].(*p.FunctionDeclaration)
	if !ok {
		t.Fatalf("expected parser FunctionDeclaration, got %T", pback.Declarations[0])
	}

	if len(pfn.Parameters) != 1 {
		t.Fatalf("expected 1 parameter after round-trip")
	}

	nsp := pfn.Parameters[0].Name.Span
	if nsp.Start.Line != 3 || nsp.Start.Column != 7 || nsp.End.Line != 3 || nsp.End.Column != 10 {
		t.Fatalf("parameter name span not preserved after round-trip: %+v", nsp)
	}
}
