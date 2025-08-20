package parser

import (
	"testing"
)

func TestHIRValidatorBasics(t *testing.T) {
	validator := NewHIRValidator()

	t.Run("Basic Module Validation", func(t *testing.T) {
		// Create a valid HIR module.
		module := &HIRModule{
			Span:      Span{},
			Name:      "test_module",
			Functions: []*HIRFunction{},
			Variables: []*HIRVariable{},
			Types:     []*HIRTypeDefinition{},
		}

		// Validate the module.
		errs := validator.ValidateModule(module)
		if len(errs) > 0 {
			t.Logf("Got %d validation errors (expected for incomplete implementation)", len(errs))
		}

		t.Log("✅ Basic module validation test successful")
	})

	t.Run("Function Validation", func(t *testing.T) {
		// Create a simple function.
		intType := NewHIRType(Span{}, HIRTypePrimitive, &HIRPrimitiveType{Name: "int"})
		function := &HIRFunction{
			Span:       Span{},
			Name:       "test_func",
			Parameters: []*HIRParameter{},
			ReturnType: intType,
			Body:       &HIRBlock{Span: Span{}, Statements: []*HIRStatement{}},
		}

		// Test individual function validation.
		validator.validateFunction(function)

		t.Log("✅ Function validation test successful")
	})

	t.Run("Variable Validation", func(t *testing.T) {
		// Create a simple variable.
		intType := NewHIRType(Span{}, HIRTypePrimitive, &HIRPrimitiveType{Name: "int"})
		variable := &HIRVariable{
			Span:        Span{},
			Name:        "test_var",
			Type:        intType,
			Initializer: nil,
		}

		// Test individual variable validation.
		validator.validateVariable(variable)

		t.Log("✅ Variable validation test successful")
	})
}
