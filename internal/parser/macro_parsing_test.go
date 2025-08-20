package parser

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/lexer"
)

// Ensures macro templates terminate correctly at '}' and don't require trailing arrows.
func TestMacroTemplateTerminationAtBrace(t *testing.T) {
	input := `macro m() {
        _ -> 1
    }`
	p := NewParser(lexer.New(input), "test.oriz")
	_ = p.parseProgram()

	if len(p.errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", p.errors)
	}
}

// Ensures block arguments are accepted in macro invocations, e.g., time_execution!({ ... }).
func TestMacroInvocationBlockArgument(t *testing.T) {
	input := `func main(){ time_execution!({ let x = 0; }); }`
	p := NewParser(lexer.New(input), "test.oriz")
	_ = p.parseProgram()

	if len(p.errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", p.errors)
	}
}

// Ensures nested function declarations inside macro bodies are parsed.
func TestNestedFunctionInMacroBody(t *testing.T) {
	input := `macro g(){ _ -> { func inner() -> int { return 1; } } }`
	p := NewParser(lexer.New(input), "test.oriz")
	_ = p.parseProgram()

	if len(p.errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", p.errors)
	}
}

// Ensures variadic marker ... on macro parameters is recognized syntactically.
func TestMacroVariadicParameterMarker(t *testing.T) {
	input := `macro println_multi(args...){ _ -> {} }`
	p := NewParser(lexer.New(input), "test.oriz")
	_ = p.parseProgram()

	if len(p.errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", p.errors)
	}
}
