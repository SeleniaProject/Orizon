package ast

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/position"
)

// createTestSpan creates a basic position span for testing
func createTestSpan(line, col int) position.Span {
	return position.Span{
		Start: position.Position{Filename: "test.oriz", Line: line, Column: col},
		End:   position.Position{Filename: "test.oriz", Line: line, Column: col + 1},
	}
}

// TestBasicNodeTypes tests basic AST node creation and functionality
func TestBasicNodeTypes(t *testing.T) {
	span := createTestSpan(1, 1)

	// Test Identifier
	id := &Identifier{Span: span, Value: "testVar"}
	if id.GetSpan() != span {
		t.Error("Identifier span not set correctly")
	}
	if id.String() != "testVar" {
		t.Errorf("Expected 'testVar', got '%s'", id.String())
	}

	// Test Literal
	lit := &Literal{
		Span:  span,
		Kind:  LiteralInteger,
		Value: 42,
		Raw:   "42",
	}
	if lit.GetSpan() != span {
		t.Error("Literal span not set correctly")
	}
	if lit.Value != 42 {
		t.Errorf("Expected 42, got %v", lit.Value)
	}

	// Test BasicType
	basicType := &BasicType{Span: span, Kind: BasicInt}
	if basicType.GetSpan() != span {
		t.Error("BasicType span not set correctly")
	}
	if basicType.String() != "int" {
		t.Errorf("Expected 'int', got '%s'", basicType.String())
	}
}

// TestBinaryExpression tests binary expression functionality
func TestBinaryExpression(t *testing.T) {
	span := createTestSpan(1, 1)

	left := &Literal{Span: span, Kind: LiteralInteger, Value: 1, Raw: "1"}
	right := &Literal{Span: span, Kind: LiteralInteger, Value: 2, Raw: "2"}

	binExpr := &BinaryExpression{
		Span:     span,
		Left:     left,
		Operator: OpAdd,
		Right:    right,
	}

	if binExpr.GetSpan() != span {
		t.Error("BinaryExpression span not set correctly")
	}
	if binExpr.Operator != OpAdd {
		t.Error("BinaryExpression operator not set correctly")
	}
	if binExpr.Left != left || binExpr.Right != right {
		t.Error("BinaryExpression operands not set correctly")
	}
}

// TestFunctionDeclaration tests function declaration functionality
func TestFunctionDeclaration(t *testing.T) {
	span := createTestSpan(1, 1)

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

	if fn.GetSpan() != span {
		t.Error("FunctionDeclaration span not set correctly")
	}
	if fn.Name.Value != "test" {
		t.Error("FunctionDeclaration name not set correctly")
	}
	if len(fn.Parameters) != 1 {
		t.Error("FunctionDeclaration parameters not set correctly")
	}
}

// TestProgram tests program node functionality
func TestProgram(t *testing.T) {
	span := createTestSpan(1, 1)

	fn := &FunctionDeclaration{
		Span: span,
		Name: &Identifier{Span: span, Value: "main"},
		Body: &BlockStatement{Span: span, Statements: []Statement{}},
	}

	program := &Program{
		Span:         span,
		Declarations: []Declaration{fn},
		Comments:     []Comment{},
	}

	if program.GetSpan() != span {
		t.Error("Program span not set correctly")
	}
	if len(program.Declarations) != 1 {
		t.Error("Program declarations not set correctly")
	}
}

// TestVisitorPattern tests the visitor pattern implementation
func TestVisitorPattern(t *testing.T) {
	span := createTestSpan(1, 1)

	// Create a simple literal for testing
	lit := &Literal{Span: span, Kind: LiteralInteger, Value: 42, Raw: "42"}

	// Create a mock visitor
	visitor := &MockVisitor{}

	// Test visitor pattern
	result := lit.Accept(visitor)
	if result == nil {
		t.Error("Visitor pattern returned nil")
	}
}

// MockVisitor implements the Visitor interface for testing
type MockVisitor struct{}

func (m *MockVisitor) VisitProgram(node *Program) interface{}                         { return node }
func (m *MockVisitor) VisitComment(node *Comment) interface{}                         { return node }
func (m *MockVisitor) VisitFunctionDeclaration(node *FunctionDeclaration) interface{} { return node }
func (m *MockVisitor) VisitParameter(node *Parameter) interface{}                     { return node }
func (m *MockVisitor) VisitVariableDeclaration(node *VariableDeclaration) interface{} { return node }
func (m *MockVisitor) VisitTypeDeclaration(node *TypeDeclaration) interface{}         { return node }
func (m *MockVisitor) VisitBlockStatement(node *BlockStatement) interface{}           { return node }
func (m *MockVisitor) VisitExpressionStatement(node *ExpressionStatement) interface{} { return node }
func (m *MockVisitor) VisitReturnStatement(node *ReturnStatement) interface{}         { return node }
func (m *MockVisitor) VisitIfStatement(node *IfStatement) interface{}                 { return node }
func (m *MockVisitor) VisitWhileStatement(node *WhileStatement) interface{}           { return node }
func (m *MockVisitor) VisitIdentifier(node *Identifier) interface{}                   { return node }
func (m *MockVisitor) VisitLiteral(node *Literal) interface{}                         { return node }
func (m *MockVisitor) VisitBinaryExpression(node *BinaryExpression) interface{}       { return node }
func (m *MockVisitor) VisitUnaryExpression(node *UnaryExpression) interface{}         { return node }
func (m *MockVisitor) VisitCallExpression(node *CallExpression) interface{}           { return node }
func (m *MockVisitor) VisitBasicType(node *BasicType) interface{}                     { return node }
func (m *MockVisitor) VisitIdentifierType(node *IdentifierType) interface{}           { return node }
func (m *MockVisitor) VisitAttribute(node *Attribute) interface{}                     { return node }
