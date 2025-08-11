package parser

import (
	"testing"
)

// TestTypeNodeKindEnum tests the NodeKind enumeration
func TestTypeNodeKindEnum(t *testing.T) {
	tests := []struct {
		name     string
		nodeKind NodeKind
		expected string
	}{
		{"Program", NodeKindProgram, "Program"},
		{"FunctionDeclaration", NodeKindFunctionDeclaration, "FunctionDeclaration"},
		{"ArrayType", NodeKindArrayType, "ArrayType"},
		{"StructType", NodeKindStructType, "StructType"},
		{"ArrayExpression", NodeKindArrayExpression, "ArrayExpression"},
		{"ForStatement", NodeKindForStatement, "ForStatement"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.nodeKind.String()
			if result != tt.expected {
				t.Errorf("NodeKind.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestTypeSafeNodeClone tests the Clone functionality
func TestTypeSafeNodeClone(t *testing.T) {
	// Test Identifier clone
	original := &Identifier{
		Span:  Span{},
		Value: "testVar",
	}

	cloned := original.Clone().(*Identifier)
	if cloned == original {
		t.Error("Clone should return a different instance")
	}
	if cloned.Value != original.Value {
		t.Errorf("Clone should preserve value: got %v, want %v", cloned.Value, original.Value)
	}

	// Test Literal clone
	lit := &Literal{
		Span:  Span{},
		Value: 42,
		Kind:  LiteralInteger,
	}

	clonedLit := lit.Clone().(*Literal)
	if clonedLit == lit {
		t.Error("Clone should return a different instance")
	}
	if clonedLit.Value != lit.Value {
		t.Errorf("Clone should preserve value: got %v, want %v", clonedLit.Value, lit.Value)
	}
}

// TestTypeSafeNodeEquals tests the Equals functionality
func TestTypeSafeNodeEquals(t *testing.T) {
	// Test Identifier equals
	id1 := &Identifier{Span: Span{}, Value: "test"}
	id2 := &Identifier{Span: Span{}, Value: "test"}
	id3 := &Identifier{Span: Span{}, Value: "different"}

	if !id1.Equals(id2) {
		t.Error("Identical identifiers should be equal")
	}
	if id1.Equals(id3) {
		t.Error("Different identifiers should not be equal")
	}

	// Test Literal equals
	lit1 := &Literal{Span: Span{}, Value: 42, Kind: LiteralInteger}
	lit2 := &Literal{Span: Span{}, Value: 42, Kind: LiteralInteger}
	lit3 := &Literal{Span: Span{}, Value: 43, Kind: LiteralInteger}

	if !lit1.Equals(lit2) {
		t.Error("Identical literals should be equal")
	}
	if lit1.Equals(lit3) {
		t.Error("Different literals should not be equal")
	}
}

// TestTypeSafeNodeGetChildren tests the GetChildren functionality
func TestTypeSafeNodeGetChildren(t *testing.T) {
	// Test Identifier (leaf node)
	id := &Identifier{Span: Span{}, Value: "test"}
	children := id.GetChildren()
	if len(children) != 0 {
		t.Errorf("Identifier should have no children, got %d", len(children))
	}

	// Test BinaryExpression
	left := &Identifier{Span: Span{}, Value: "a"}
	right := &Identifier{Span: Span{}, Value: "b"}
	op := &Operator{Span: Span{}, Value: "+", Kind: BinaryOp}

	binExpr := &BinaryExpression{
		Span:     Span{},
		Left:     left,
		Right:    right,
		Operator: op,
	}

	children = binExpr.GetChildren()
	if len(children) != 3 {
		t.Errorf("BinaryExpression should have 3 children, got %d", len(children))
	}
}

// TestArrayType tests the ArrayType node
func TestArrayType(t *testing.T) {
	elementType := &BasicType{Span: Span{}, Name: "int"}
	size := &Literal{Span: Span{}, Value: 10, Kind: LiteralInteger}

	// Static array
	staticArray := &ArrayType{
		Span:        Span{},
		ElementType: elementType,
		Size:        size,
		IsDynamic:   false,
	}

	if staticArray.GetNodeKind() != NodeKindArrayType {
		t.Errorf("ArrayType should have NodeKindArrayType, got %v", staticArray.GetNodeKind())
	}

	expectedStr := "[int; 10]"
	if staticArray.String() != expectedStr {
		t.Errorf("ArrayType.String() = %v, want %v", staticArray.String(), expectedStr)
	}

	// Dynamic array
	dynamicArray := &ArrayType{
		Span:        Span{},
		ElementType: elementType,
		Size:        nil,
		IsDynamic:   true,
	}

	expectedDynStr := "[int]"
	if dynamicArray.String() != expectedDynStr {
		t.Errorf("Dynamic ArrayType.String() = %v, want %v", dynamicArray.String(), expectedDynStr)
	}

	// Test children
	children := staticArray.GetChildren()
	if len(children) != 2 {
		t.Errorf("Static ArrayType should have 2 children, got %d", len(children))
	}

	dynChildren := dynamicArray.GetChildren()
	if len(dynChildren) != 1 {
		t.Errorf("Dynamic ArrayType should have 1 child, got %d", len(dynChildren))
	}
}

// TestStructType tests the StructType node
func TestStructType(t *testing.T) {
	field1 := &StructField{
		Span:     Span{},
		Name:     &Identifier{Span: Span{}, Value: "field1"},
		Type:     &BasicType{Span: Span{}, Name: "int"},
		IsPublic: true,
	}

	field2 := &StructField{
		Span:     Span{},
		Name:     &Identifier{Span: Span{}, Value: "field2"},
		Type:     &BasicType{Span: Span{}, Name: "string"},
		IsPublic: false,
	}

	structType := &StructType{
		Span:   Span{},
		Name:   &Identifier{Span: Span{}, Value: "MyStruct"},
		Fields: []*StructField{field1, field2},
	}

	if structType.GetNodeKind() != NodeKindStructType {
		t.Errorf("StructType should have NodeKindStructType, got %v", structType.GetNodeKind())
	}

	expectedStr := "struct MyStruct"
	if structType.String() != expectedStr {
		t.Errorf("StructType.String() = %v, want %v", structType.String(), expectedStr)
	}

	// Test children (name + field names + field types)
	children := structType.GetChildren()
	expectedChildren := 1 + len(structType.Fields)*2 // name + (field name + field type) for each field
	if len(children) != expectedChildren {
		t.Errorf("StructType should have %d children, got %d", expectedChildren, len(children))
	}
}

// TestFunctionType tests the FunctionType node
func TestFunctionType(t *testing.T) {
	param1 := &FunctionTypeParameter{
		Span: Span{},
		Type: &BasicType{Span: Span{}, Name: "int"},
	}
	param2 := &FunctionTypeParameter{
		Span: Span{},
		Type: &BasicType{Span: Span{}, Name: "string"},
	}
	returnType := &BasicType{Span: Span{}, Name: "bool"}

	funcType := &FunctionType{
		Span:       Span{},
		Parameters: []*FunctionTypeParameter{param1, param2},
		ReturnType: returnType,
	}

	if funcType.GetNodeKind() != NodeKindFunctionType {
		t.Errorf("FunctionType should have NodeKindFunctionType, got %v", funcType.GetNodeKind())
	}

	expectedStr := "(int, string) -> bool"
	if funcType.String() != expectedStr {
		t.Errorf("FunctionType.String() = %v, want %v", funcType.String(), expectedStr)
	}

	// Test children
	children := funcType.GetChildren()
	expectedChildren := len(funcType.Parameters) + 1 // parameters + return type
	if len(children) != expectedChildren {
		t.Errorf("FunctionType should have %d children, got %d", expectedChildren, len(children))
	}
}

// TestArrayExpression tests the ArrayExpression node
func TestArrayExpression(t *testing.T) {
	elem1 := &Literal{Span: Span{}, Value: 1, Kind: LiteralInteger}
	elem2 := &Literal{Span: Span{}, Value: 2, Kind: LiteralInteger}
	elem3 := &Literal{Span: Span{}, Value: 3, Kind: LiteralInteger}

	arrayExpr := &ArrayExpression{
		Span:     Span{},
		Elements: []Expression{elem1, elem2, elem3},
	}

	if arrayExpr.GetNodeKind() != NodeKindArrayExpression {
		t.Errorf("ArrayExpression should have NodeKindArrayExpression, got %v", arrayExpr.GetNodeKind())
	}

	expectedStr := "[1, 2, 3]"
	if arrayExpr.String() != expectedStr {
		t.Errorf("ArrayExpression.String() = %v, want %v", arrayExpr.String(), expectedStr)
	}

	// Test children
	children := arrayExpr.GetChildren()
	if len(children) != len(arrayExpr.Elements) {
		t.Errorf("ArrayExpression should have %d children, got %d", len(arrayExpr.Elements), len(children))
	}
}

// TestForStatement tests the ForStatement node (simplified)
func TestForStatement(t *testing.T) {
	init := &VariableDeclaration{
		Span:        Span{},
		Name:        &Identifier{Span: Span{}, Value: "i"},
		TypeSpec:    &BasicType{Span: Span{}, Name: "int"},
		Initializer: &Literal{Span: Span{}, Value: 0, Kind: LiteralInteger},
		IsMutable:   true,
	}

	condition := &BinaryExpression{
		Span:     Span{},
		Left:     &Identifier{Span: Span{}, Value: "i"},
		Right:    &Literal{Span: Span{}, Value: 10, Kind: LiteralInteger},
		Operator: &Operator{Span: Span{}, Value: "<", Kind: BinaryOp},
	}

	body := &BlockStatement{
		Span:       Span{},
		Statements: []Statement{},
	}

	forStmt := &ForStatement{
		Span:      Span{},
		Init:      init,
		Condition: condition,
		Update:    nil, // Simplified for testing
		Body:      body,
	}

	if forStmt.GetNodeKind() != NodeKindForStatement {
		t.Errorf("ForStatement should have NodeKindForStatement, got %v", forStmt.GetNodeKind())
	}

	expectedStr := "for (...) { ... }"
	if forStmt.String() != expectedStr {
		t.Errorf("ForStatement.String() = %v, want %v", forStmt.String(), expectedStr)
	}

	// Test children (init, condition, body - update is nil)
	children := forStmt.GetChildren()
	expectedChildren := 3
	if len(children) != expectedChildren {
		t.Errorf("ForStatement should have %d children, got %d", expectedChildren, len(children))
	}
}

// TestValidator tests the AST validation functionality (simplified)
func TestValidator(t *testing.T) {
	validator := NewValidator(true) // strict mode

	// Test valid node
	validId := &Identifier{Span: Span{}, Value: "validName"}
	err := validator.Validate(validId)
	if err != nil {
		t.Errorf("Valid identifier should pass validation: %v", err)
	}

	// Test invalid node (empty identifier)
	invalidId := &Identifier{Span: Span{}, Value: ""}
	err = validator.Validate(invalidId)
	if err == nil {
		t.Error("Invalid identifier should fail validation")
	}
}

// TestBasicCloneEquality tests basic Clone and Equals operations
func TestBasicCloneEquality(t *testing.T) {
	// Test basic type
	basicType := &BasicType{Span: Span{}, Name: "int"}
	clonedBasic := basicType.Clone().(*BasicType)

	if !basicType.Equals(clonedBasic) {
		t.Error("Cloned BasicType should equal original")
	}

	if clonedBasic == basicType {
		t.Error("Clone should create new instance")
	}

	// Test array type
	arrayType := &ArrayType{
		Span:        Span{},
		ElementType: basicType,
		IsDynamic:   true,
	}

	clonedArray := arrayType.Clone().(*ArrayType)
	if !arrayType.Equals(clonedArray) {
		t.Error("Cloned ArrayType should equal original")
	}
}

// TestNodeKindValidation tests NodeKind values
func TestNodeKindValidation(t *testing.T) {
	// Test that all new node kinds have valid string representations
	nodeKinds := []NodeKind{
		NodeKindArrayType,
		NodeKindFunctionType,
		NodeKindStructType,
		NodeKindEnumType,
		NodeKindTraitType,
		NodeKindGenericType,
		NodeKindArrayExpression,
		NodeKindIndexExpression,
		NodeKindMemberExpression,
		NodeKindStructExpression,
		NodeKindForStatement,
		NodeKindBreakStatement,
		NodeKindContinueStatement,
		NodeKindMatchStatement,
	}

	for _, kind := range nodeKinds {
		str := kind.String()
		if str == "" {
			t.Errorf("NodeKind %d should have a non-empty string representation", kind)
		}
	}
}
