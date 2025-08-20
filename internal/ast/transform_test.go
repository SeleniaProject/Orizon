package ast

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/position"
)

// createTestSpanTransform creates a basic position span for transformation testing.
func createTestSpanTransform(line, col int) position.Span {
	return position.Span{
		Start: position.Position{Filename: "test.oriz", Line: line, Column: col},
		End:   position.Position{Filename: "test.oriz", Line: line, Column: col + 1},
	}
}

// TestTransformationPipeline tests the transformation pipeline functionality.
func TestTransformationPipeline(t *testing.T) {
	span := createTestSpanTransform(1, 1)

	// Create a simple binary expression: 1 + 2.
	left := &Literal{Span: span, Kind: LiteralInteger, Value: 1, Raw: "1"}
	right := &Literal{Span: span, Kind: LiteralInteger, Value: 2, Raw: "2"}
	expr := &BinaryExpression{
		Span:     span,
		Left:     left,
		Operator: OpAdd,
		Right:    right,
	}

	// Create transformation pipeline.
	pipeline := NewTransformationPipeline()

	// Test empty pipeline.
	result, err := pipeline.Transform(expr)
	if err != nil {
		t.Errorf("Empty pipeline failed: %v", err)
	}

	if result != expr {
		t.Error("Empty pipeline should return original node")
	}
}

// TestConstantFolding tests basic constant folding functionality.
func TestConstantFolding(t *testing.T) {
	span := createTestSpanTransform(1, 1)

	// Create a simple binary expression: 1 + 2.
	left := &Literal{Span: span, Kind: LiteralInteger, Value: 1, Raw: "1"}
	right := &Literal{Span: span, Kind: LiteralInteger, Value: 2, Raw: "2"}
	expr := &BinaryExpression{
		Span:     span,
		Left:     left,
		Operator: OpAdd,
		Right:    right,
	}

	// Apply constant folding.
	transformer := &ConstantFoldingTransformer{}

	result, err := transformer.Transform(expr)
	if err != nil {
		t.Errorf("Constant folding failed: %v", err)
	}

	// Check if result is a literal with value 3.
	if litResult, ok := result.(*Literal); ok {
		if litResult.Kind != LiteralInteger {
			t.Error("Expected integer literal result")
		}
		// Handle both int and int64 types.
		var resultValue int64
		switch v := litResult.Value.(type) {
		case int:
			resultValue = int64(v)
		case int64:
			resultValue = v
		default:
			t.Errorf("Expected integer value, got %T", litResult.Value)

			return
		}

		if resultValue != 3 {
			t.Errorf("Expected result value 3, got %v", resultValue)
		}
	} else {
		t.Error("Expected literal result from constant folding")
	}
}

// TestASTBuilder tests the AST builder functionality.
func TestASTBuilder(t *testing.T) {
	span := createTestSpanTransform(1, 1)

	builder := NewASTBuilder()
	if builder == nil {
		t.Error("NewASTBuilder returned nil")
	}

	builderWithSpan := NewASTBuilderWithSpan(span)
	if builderWithSpan == nil {
		t.Error("NewASTBuilderWithSpan returned nil")
	}

	if builderWithSpan.defaultSpan != span {
		t.Error("ASTBuilder default span not set correctly")
	}
}

// TestTransformationError tests transformation error functionality.
func TestTransformationError(t *testing.T) {
	span := createTestSpanTransform(1, 1)
	message := "test error"

	err := NewTransformationError(message, span)
	if err == nil {
		t.Error("NewTransformationError returned nil")
	}

	if err.Message != message {
		t.Errorf("Expected error message %q, got %q", message, err.Message)
	}

	if err.Span != span {
		t.Error("Error span not set correctly")
	}

	errorString := err.Error()
	if errorString == "" {
		t.Error("Error string is empty")
	}
}

// TestTransformationWithValidation tests transformation with validation.
func TestTransformationWithValidation(t *testing.T) {
	span := createTestSpanTransform(2, 5)

	// Create a function declaration for testing.
	param := &Parameter{
		Span: span,
		Name: &Identifier{Span: span, Value: "x"},
		Type: &BasicType{Span: span, Kind: BasicInt},
	}

	fn := &FunctionDeclaration{
		Span:       span,
		Name:       &Identifier{Span: span, Value: "test"},
		Parameters: []*Parameter{param},
		ReturnType: &BasicType{Span: span, Kind: BasicInt},
		Body: &BlockStatement{
			Span:       span,
			Statements: []Statement{},
		},
	}

	// Create pipeline with validation.
	pipeline := NewTransformationPipeline()
	result, err := pipeline.Transform(fn)
	if err != nil {
		t.Errorf("Transformation with validation failed: %v", err)
	}

	if result == nil {
		t.Error("Transformation result is nil")
	}
}
