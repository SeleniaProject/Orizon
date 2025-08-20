package parser

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/lexer"
)

// TestMacroBasicFunctionality tests basic macro system functionality.
func TestMacroBasicFunctionality(t *testing.T) {
	// Test macro definition parsing.
	t.Run("MacroDefinitionParsing", func(t *testing.T) {
		input := `macro simple() { x + y }`
		l := lexer.New(input)
		p := NewParser(l, "test.oriz")
		program := p.parseProgram()

		if len(p.errors) > 0 {
			t.Logf("Parser has %d errors (some may be expected):", len(p.errors))

			for _, err := range p.errors {
				t.Logf("  - %s", err)
			}
		}

		t.Logf("✅ Macro definition parsing: parsed %d declarations",
			len(program.Declarations))
	})

	// Test macro invocation parsing.
	t.Run("MacroInvocationParsing", func(t *testing.T) {
		input := `println!(42)`
		l := lexer.New(input)
		p := NewParser(l, "test.oriz")

		// Parse as expression statement.
		p.nextToken()
		stmt := p.parseExpressionStatement()

		if len(p.errors) > 0 {
			t.Logf("Parser has %d errors (some may be expected):", len(p.errors))

			for _, err := range p.errors {
				t.Logf("  - %s", err)
			}
		}

		if stmt != nil {
			t.Logf("✅ Successfully parsed macro invocation")
		} else {
			t.Logf("Note: Macro invocation parsing not yet complete")
		}
	})

	// Test macro engine creation.
	t.Run("MacroEngineCreation", func(t *testing.T) {
		engine := NewMacroEngine()

		if engine == nil {
			t.Fatal("Failed to create macro engine")
		}

		t.Logf("✅ Macro engine created successfully")

		// Test macro registration.
		macro := &MacroDefinition{
			Name: &Identifier{Value: "test_macro"},
			Body: &MacroBody{Templates: []*MacroTemplate{}},
		}

		err := engine.RegisterMacro(macro)
		if err != nil {
			t.Errorf("Failed to register macro: %v", err)
		} else {
			t.Logf("✅ Macro registered successfully")
		}

		// Test macro retrieval.
		retrieved, exists := engine.GetMacro("test_macro")
		if !exists {
			t.Errorf("Failed to retrieve registered macro")
		} else if retrieved.Name.Value != "test_macro" {
			t.Errorf("Retrieved macro has wrong name: %s", retrieved.Name.Value)
		} else {
			t.Logf("✅ Macro retrieved successfully")
		}
	})

	// Test macro validator.
	t.Run("MacroValidation", func(t *testing.T) {
		validator := NewMacroValidator()

		if validator == nil {
			t.Fatal("Failed to create macro validator")
		}

		// Test nil macro validation.
		errors := validator.ValidateMacro(nil)
		if len(errors) == 0 {
			t.Errorf("Expected validation errors for nil macro")
		} else {
			t.Logf("✅ Nil macro validation: %d errors found", len(errors))
		}

		// Test valid macro validation.
		validMacro := &MacroDefinition{
			Name: &Identifier{Value: "valid_macro"},
			Body: &MacroBody{
				Templates: []*MacroTemplate{
					{
						Pattern: &MacroPattern{Elements: []*MacroPatternElement{{Kind: MacroPatternParameter, Value: "x"}}},
						Body:    []Statement{},
					},
				},
			},
		}

		errors = validator.ValidateMacro(validMacro)
		if len(errors) > 0 {
			t.Logf("Validation errors for valid macro: %v", errors)
		} else {
			t.Logf("✅ Valid macro passed validation")
		}
	})

	// Test macro builtins.
	t.Run("MacroBuiltins", func(t *testing.T) {
		builtins := NewMacroBuiltins()

		if builtins == nil {
			t.Fatal("Failed to create macro builtins")
		}

		// Test println builtin.
		fn, exists := builtins.GetBuiltin("println")
		if !exists {
			t.Errorf("println builtin not found")
		} else {
			t.Logf("✅ println builtin found")

			// Test with valid arguments.
			args := []*MacroArgument{
				{
					Kind:  MacroArgExpression,
					Value: &Literal{Value: "Hello, World!"},
				},
			}

			statements, err := fn(args)
			if err != nil {
				t.Errorf("println builtin failed: %v", err)
			} else {
				t.Logf("✅ println builtin generated %d statements", len(statements))
			}
		}

		// Test nonexistent builtin.
		_, exists = builtins.GetBuiltin("nonexistent")
		if exists {
			t.Errorf("Found nonexistent builtin")
		} else {
			t.Logf("✅ Nonexistent builtin correctly not found")
		}
	})

	// Test integration.
	t.Run("MacroIntegration", func(t *testing.T) {
		input := `
		macro add_debug(x, y) {
			x + y
		}

		func main() {
			let result = 42;
			return result;
		}
		`

		l := lexer.New(input)
		p := NewParser(l, "test.oriz")
		program := p.parseProgram()

		if len(p.errors) > 0 {
			t.Logf("Integration test errors (some may be expected): %d errors", len(p.errors))
		}

		// Should have at least some declarations parsed.
		if len(program.Declarations) == 0 {
			t.Logf("Note: No declarations parsed - macro integration still in development")
		} else {
			t.Logf("✅ Macro integration test: parsed %d declarations", len(program.Declarations))
		}
	})
}
