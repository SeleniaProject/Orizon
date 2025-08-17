package parser

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/lexer"
)

// Test that an unclosed struct does not swallow following top-level declarations
func TestRecovery_UnclosedStruct_AllowsNextDecl(t *testing.T) {
	input := `
        struct S {
            x: int
        // missing closing brace
        func ok() { return; }
    `
	l := lexer.NewWithFilename(input, "decl_recover1.oriz")
	p := NewParser(l, "decl_recover1.oriz")

	prog, errs := p.Parse()
	if len(errs) == 0 {
		t.Fatalf("expected errors for unclosed struct, got none")
	}
	// Should at least parse the following function
	foundFunc := false
	for _, d := range prog.Declarations {
		if _, ok := d.(*FunctionDeclaration); ok {
			foundFunc = true
			break
		}
	}
	if !foundFunc {
		t.Fatalf("expected subsequent function declaration to be parsed despite prior error")
	}
}

// Test that encountering a new top-level token inside a trait body triggers recovery and allows next decl
func TestRecovery_TraitMissingBrace_SyncToNextDecl(t *testing.T) {
	input := `
        trait T {
            func m() -> void;
        // missing '}' here
        enum E { A }
        func after() { return; }
    `
	l := lexer.NewWithFilename(input, "decl_recover2.oriz")
	p := NewParser(l, "decl_recover2.oriz")

	prog, errs := p.Parse()
	if len(errs) == 0 {
		t.Fatalf("expected errors for unclosed trait, got none")
	}
	// Ensure at least one of enum or function after is parsed
	found := 0
	for _, d := range prog.Declarations {
		switch d.(type) {
		case *EnumDeclaration, *FunctionDeclaration:
			found++
		}
	}
	if found == 0 {
		t.Fatalf("expected parser to recover and parse subsequent declarations")
	}
}
