package parser

import (
	"testing"
	
	"github.com/orizon-lang/orizon/internal/lexer"
)

func TestHIRConversion_Basic(t *testing.T) {
	code := `let x = 42;`

	l := lexer.New(code)
	p := NewParser(l, "test.oriz")

	program, errors := p.Parse()

	if len(errors) > 0 {
		t.Fatalf("Parse failed: %v", errors)
	}

	// Convert AST to HIR
	transformer := NewASTToHIRTransformer()
	hirModule, err := transformer.TransformModule(program)

	if err != nil {
		t.Fatalf("HIR transformation failed: %v", err)
	}

	if hirModule == nil {
		t.Fatal("HIR conversion failed")
	}

	// Check if we have any variables converted
	if len(hirModule.Variables) == 0 {
		t.Error("Expected at least one variable")
	}
}

func TestHIRConversion_StructDeclaration(t *testing.T) {
	transformer := NewASTToHIRTransformer()

	source := `struct Point { x: i32, y: i32 }`

	l := lexer.New(source)
	parser := NewParser(l, "test.oriz")
	ast, err := parser.Parse()
	
	if err != nil && len(err) > 0 {
		t.Fatalf("Parse failed: %v", err)
	}

	// Convert to HIR
	hirModule, transformErr := transformer.TransformModule(ast)
	if transformErr != nil {
		t.Fatalf("HIR transformation failed: %v", transformErr)
	}
	if hirModule == nil {
		t.Fatal("HIR conversion returned nil")
	}

	// Check that types were converted
	if len(hirModule.Types) == 0 {
		t.Error("No types found in HIR module")
	}

	// Verify type has expected structure
	for _, hirType := range hirModule.Types {
		if hirType.Name == "" {
			t.Error("Type name is empty")
		}
		if hirType.Data == nil {
			t.Error("Type data is nil")
		}
	}
}

func TestHIRConversion_ComplexTypes(t *testing.T) {
	transformer := NewASTToHIRTransformer()

	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "Array type",
			source: `let arr: [i32; 5];`,
		},
		{
			name:   "Pointer type", 
			source: `let ptr: *i32;`,
		},
		{
			name:   "Reference type",
			source: `let ref: &i32;`,
		},
		{
			name:   "Function type",
			source: `func add(a: i32, b: i32) -> i32 { return a + b; }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.source)
			parser := NewParser(l, "test.oriz")
			ast, err := parser.Parse()
			
			if err != nil && len(err) > 0 {
				t.Fatalf("Parse failed: %v", err)
			}

			// Convert to HIR
			hirModule, transformErr := transformer.TransformModule(ast)
			if transformErr != nil {
				t.Fatalf("HIR transformation failed: %v", transformErr)
			}
			if hirModule == nil {
				t.Fatal("HIR conversion returned nil")
			}

			// For complex types, just verify we get some result
			// Detailed validation would require checking specific HIR node types
			t.Logf("HIR conversion successful for %s", tt.name)
		})
	}
}
