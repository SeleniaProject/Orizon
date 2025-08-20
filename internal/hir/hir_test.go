// HIR package tests for the Orizon programming language.
// This file provides comprehensive tests for HIR construction, conversion, and verification.

package hir

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/ast"
	"github.com/orizon-lang/orizon/internal/position"
)

// Test HIR program creation.
func TestHIRProgramCreation(t *testing.T) {
	program := NewHIRProgram()

	if program == nil {
		t.Fatal("NewHIRProgram() returned nil")
	}

	if program.ID == 0 {
		t.Error("Program should have non-zero ID")
	}

	if program.Modules == nil {
		t.Error("Program should have initialized modules map")
	}

	if program.TypeInfo == nil {
		t.Error("Program should have type information")
	}

	if program.EffectInfo == nil {
		t.Error("Program should have effect information")
	}

	if program.RegionInfo == nil {
		t.Error("Program should have region information")
	}
}

// Test global type information initialization.
func TestGlobalTypeInfoInitialization(t *testing.T) {
	typeInfo := NewGlobalTypeInfo()

	if typeInfo == nil {
		t.Fatal("NewGlobalTypeInfo() returned nil")
	}

	// Check primitive types are initialized.
	requiredPrimitives := []string{"void", "bool", "i32", "f64", "string"}

	for _, prim := range requiredPrimitives {
		if _, exists := typeInfo.Primitives[prim]; !exists {
			t.Errorf("Missing primitive type: %s", prim)
		}
	}

	// Check type consistency.
	for name, id := range typeInfo.Primitives {
		if typ, exists := typeInfo.Types[id]; !exists {
			t.Errorf("Type %s has ID %d but no corresponding type info", name, id)
		} else if typ.Name != name {
			t.Errorf("Type name mismatch: expected %s, got %s", name, typ.Name)
		}
	}
}

// Test HIR type builder.
func TestHIRTypeBuilder(t *testing.T) {
	program := NewHIRProgram()
	builder := NewHIRTypeBuilder(program)

	span := position.Span{
		Start: position.Position{Line: 1, Column: 1},
		End:   position.Position{Line: 1, Column: 4},
	}

	// Test basic type creation.
	intType := builder.BuildBasicType("i32", span)
	if intType == nil {
		t.Fatal("BuildBasicType returned nil")
	}

	if intType.Name != "i32" {
		t.Errorf("Expected type name i32, got %s", intType.Name)
	}

	// Test array type creation.
	arrayType := builder.BuildArrayType(intType, nil, span)
	if arrayType == nil {
		t.Fatal("BuildArrayType returned nil")
	}

	if arrayType.ElementType != intType {
		t.Error("Array element type not set correctly")
	}

	// Test pointer type creation.
	ptrType := builder.BuildPointerType(intType, false, span)
	if ptrType == nil {
		t.Fatal("BuildPointerType returned nil")
	}

	if ptrType.TargetType != intType {
		t.Error("Pointer target type not set correctly")
	}

	if ptrType.Mutable {
		t.Error("Pointer should not be mutable by default")
	}
}

// Test AST to HIR conversion.
func TestASTToHIRConversion(t *testing.T) {
	converter := NewASTToHIRConverter()

	// Create a simple AST program.
	astProgram := &ast.Program{
		Span: position.Span{
			Start: position.Position{Line: 1, Column: 1},
			End:   position.Position{Line: 10, Column: 1},
		},
		Declarations: []ast.Declaration{},
	}

	// Convert to HIR.
	hirProgram, errors := converter.ConvertProgram(astProgram)

	if hirProgram == nil {
		t.Fatal("ConvertProgram returned nil HIR program")
	}

	if len(errors) > 0 {
		t.Errorf("Conversion had %d errors: %v", len(errors), errors)
	}

	// Check basic structure.
	if hirProgram.ID == 0 {
		t.Error("HIR program should have non-zero ID")
	}

	if len(hirProgram.Modules) == 0 {
		t.Error("HIR program should have at least one module")
	}

	// Check main module.
	mainModule, exists := hirProgram.Modules[1]
	if !exists {
		t.Fatal("HIR program should have main module with ID 1")
	}

	if mainModule.Name != "main" {
		t.Errorf("Expected main module name 'main', got '%s'", mainModule.Name)
	}
}

// Test HIR function declaration conversion.
func TestHIRFunctionDeclarationConversion(t *testing.T) {
	converter := NewASTToHIRConverter()

	// Create AST function declaration.
	astFunc := &ast.FunctionDeclaration{
		Span: position.Span{
			Start: position.Position{Line: 1, Column: 1},
			End:   position.Position{Line: 5, Column: 1},
		},
		Name: &ast.Identifier{
			Span:  position.Span{Start: position.Position{Line: 1, Column: 5}, End: position.Position{Line: 1, Column: 8}},
			Value: "test",
		},
		Parameters: []*ast.Parameter{},
		ReturnType: &ast.BasicType{
			Span: position.Span{Start: position.Position{Line: 1, Column: 12}, End: position.Position{Line: 1, Column: 16}},
			Kind: ast.BasicVoid,
		},
		Body: &ast.BlockStatement{
			Span:       position.Span{Start: position.Position{Line: 2, Column: 1}, End: position.Position{Line: 4, Column: 1}},
			Statements: []ast.Statement{},
		},
	}

	// Convert function declaration.
	hirFunc := converter.convertFunctionDeclaration(astFunc)

	if hirFunc == nil {
		t.Fatal("convertFunctionDeclaration returned nil")
	}

	// Type assert to HIRFunctionDeclaration to access fields.
	funcDecl, ok := hirFunc.(*HIRFunctionDeclaration)
	if !ok {
		t.Fatal("Expected HIRFunctionDeclaration")
	}

	if funcDecl.Name != "test" {
		t.Errorf("Expected function name 'test', got '%s'", funcDecl.Name)
	}

	if len(funcDecl.Parameters) != 0 {
		t.Errorf("Expected 0 parameters, got %d", len(funcDecl.Parameters))
	}

	if funcDecl.ReturnType == nil {
		t.Error("Function should have return type")
	}

	if funcDecl.Body == nil {
		t.Error("Function should have body")
	}
}

// Test HIR variable declaration conversion.
func TestHIRVariableDeclarationConversion(t *testing.T) {
	converter := NewASTToHIRConverter()

	// Create AST variable declaration.
	astVar := &ast.VariableDeclaration{
		Span: position.Span{
			Start: position.Position{Line: 1, Column: 1},
			End:   position.Position{Line: 1, Column: 15},
		},
		Name: &ast.Identifier{
			Span:  position.Span{Start: position.Position{Line: 1, Column: 5}, End: position.Position{Line: 1, Column: 6}},
			Value: "x",
		},
		Type: &ast.BasicType{
			Span: position.Span{Start: position.Position{Line: 1, Column: 8}, End: position.Position{Line: 1, Column: 11}},
			Kind: ast.BasicInt,
		},
		Value: &ast.Literal{
			Span:  position.Span{Start: position.Position{Line: 1, Column: 14}, End: position.Position{Line: 1, Column: 15}},
			Kind:  ast.LiteralInteger,
			Value: 42,
			Raw:   "42",
		},
		Kind:      ast.VarKindLet,
		IsMutable: false,
	}

	// Convert variable declaration.
	hirVar := converter.convertVariableDeclaration(astVar)

	if hirVar == nil {
		t.Fatal("convertVariableDeclaration returned nil")
	}

	// Type assert to HIRVariableDeclaration to access fields.
	varDecl, ok := hirVar.(*HIRVariableDeclaration)
	if !ok {
		t.Fatal("Expected HIRVariableDeclaration")
	}

	if varDecl.Name != "x" {
		t.Errorf("Expected variable name 'x', got '%s'", varDecl.Name)
	}

	if varDecl.Type == nil {
		t.Error("Variable should have type")
	}

	if varDecl.Initializer == nil {
		t.Error("Variable should have initializer")
	}

	if varDecl.Mutable {
		t.Error("Variable should not be mutable")
	}
}

// Test HIR expression conversion.
func TestHIRExpressionConversion(t *testing.T) {
	converter := NewASTToHIRConverter()

	// Test literal conversion.
	astLit := &ast.Literal{
		Span:  position.Span{Start: position.Position{Line: 1, Column: 1}, End: position.Position{Line: 1, Column: 3}},
		Kind:  ast.LiteralInteger,
		Value: 42,
		Raw:   "42",
	}

	hirLit := converter.convertLiteral(astLit)

	if hirLit == nil {
		t.Fatal("convertLiteral returned nil")
	}

	// Type assert to HIRLiteral to access fields.
	litExpr, ok := hirLit.(*HIRLiteral)
	if !ok {
		t.Fatal("Expected HIRLiteral")
	}

	if litExpr.Value != 42 {
		t.Errorf("Expected literal value 42, got %v", litExpr.Value)
	}

	// Test identifier conversion.
	astId := &ast.Identifier{
		Span:  position.Span{Start: position.Position{Line: 1, Column: 1}, End: position.Position{Line: 1, Column: 2}},
		Value: "x",
	}

	// Add symbol to symbol table first.
	converter.symbolTable.AddSymbol("x", &Symbol{
		Name: "x",
		Type: TypeInfo{Kind: TypeKindInteger, Name: "i32"},
		Span: astId.Span,
	})

	hirId := converter.convertIdentifier(astId)

	if hirId == nil {
		t.Fatal("convertIdentifier returned nil")
	}

	// Type assert to HIRIdentifier to access fields.
	idExpr, ok := hirId.(*HIRIdentifier)
	if !ok {
		t.Fatal("Expected HIRIdentifier")
	}

	if idExpr.Name != "x" {
		t.Errorf("Expected identifier name 'x', got '%s'", idExpr.Name)
	}
}

// Test HIR binary expression conversion.
func TestHIRBinaryExpressionConversion(t *testing.T) {
	converter := NewASTToHIRConverter()

	// Create AST binary expression (1 + 2).
	astBin := &ast.BinaryExpression{
		Span: position.Span{
			Start: position.Position{Line: 1, Column: 1},
			End:   position.Position{Line: 1, Column: 5},
		},
		Left: &ast.Literal{
			Span:  position.Span{Start: position.Position{Line: 1, Column: 1}, End: position.Position{Line: 1, Column: 2}},
			Kind:  ast.LiteralInteger,
			Value: 1,
			Raw:   "1",
		},
		Operator: ast.OpAdd,
		Right: &ast.Literal{
			Span:  position.Span{Start: position.Position{Line: 1, Column: 4}, End: position.Position{Line: 1, Column: 5}},
			Kind:  ast.LiteralInteger,
			Value: 2,
			Raw:   "2",
		},
	}

	hirBin := converter.convertBinaryExpression(astBin)

	if hirBin == nil {
		t.Fatal("convertBinaryExpression returned nil")
	}

	// Type assert to HIRBinaryExpression to access fields.
	binExpr, ok := hirBin.(*HIRBinaryExpression)
	if !ok {
		t.Fatal("Expected HIRBinaryExpression")
	}

	if binExpr.Operator != "+" {
		t.Errorf("Expected operator '+', got '%s'", binExpr.Operator)
	}

	if binExpr.Left == nil {
		t.Error("Binary expression should have left operand")
	}

	if binExpr.Right == nil {
		t.Error("Binary expression should have right operand")
	}
}

// Test HIR verification.
func TestHIRVerification(t *testing.T) {
	verifier := NewHIRVerifier()

	// Create a valid HIR program.
	program := NewHIRProgram()

	// Create main module.
	mainModule := &HIRModule{
		ID:           generateNodeID(),
		ModuleID:     1,
		Name:         "main",
		Declarations: []HIRDeclaration{},
		Exports:      []string{},
		Imports:      []ImportInfo{},
		Metadata:     IRMetadata{},
		Span: position.Span{
			Start: position.Position{Line: 1, Column: 1},
			End:   position.Position{Line: 10, Column: 1},
		},
	}

	program.Modules[1] = mainModule

	// Verify program.
	errors, warnings := verifier.VerifyProgram(program)

	if len(errors) > 0 {
		t.Errorf("Valid HIR program should have no errors, got %d: %v", len(errors), errors)
	}

	// Warnings are acceptable for minimal programs.
	t.Logf("Verification completed with %d warnings", len(warnings))
}

// Test HIR verification with errors.
func TestHIRVerificationWithErrors(t *testing.T) {
	verifier := NewHIRVerifier()

	// Create an invalid HIR program (nil modules).
	program := &HIRProgram{
		ID:         generateNodeID(),
		Modules:    nil, // This should cause an error
		TypeInfo:   nil, // This should also cause an error
		EffectInfo: nil,
		RegionInfo: nil,
		Metadata:   IRMetadata{},
		Span: position.Span{
			Start: position.Position{Line: 1, Column: 1},
			End:   position.Position{Line: 1, Column: 1},
		},
	}

	// Verify program.
	errors, _ := verifier.VerifyProgram(program)

	if len(errors) == 0 {
		t.Error("Invalid HIR program should have errors")
	}

	t.Logf("Found expected errors: %d", len(errors))
}

// Test effect set operations.
func TestEffectSetOperations(t *testing.T) {
	// Test empty effect set.
	effects1 := NewEffectSet()
	if !effects1.Pure {
		t.Error("New effect set should be pure")
	}

	// Test adding effect.
	effect := Effect{
		ID:          1,
		Kind:        EffectKindMemoryRead,
		Description: "test memory read",
		Modality:    EffectModalityMay,
		Scope:       EffectScopeLocal,
	}

	effects1.AddEffect(effect)

	if effects1.Pure {
		t.Error("Effect set should not be pure after adding effect")
	}

	if !effects1.HasEffect(1) {
		t.Error("Effect set should contain added effect")
	}

	// Test effect union.
	effects2 := NewEffectSet()
	effect2 := Effect{
		ID:          2,
		Kind:        EffectKindMemoryWrite,
		Description: "test memory write",
		Modality:    EffectModalityMust,
		Scope:       EffectScopeGlobal,
	}
	effects2.AddEffect(effect2)

	combined := effects1.Union(effects2)

	if combined.Pure {
		t.Error("Combined effect set should not be pure")
	}

	if !combined.HasEffect(1) {
		t.Error("Combined effect set should contain first effect")
	}

	if !combined.HasEffect(2) {
		t.Error("Combined effect set should contain second effect")
	}
}

// Test region set operations.
func TestRegionSetOperations(t *testing.T) {
	// Test empty region set.
	regions1 := NewRegionSet()

	// Test adding region.
	region := Region{
		ID:   1,
		Kind: RegionKindStack,
		Lifetime: Lifetime{
			Start: position.Span{Start: position.Position{Line: 1, Column: 1}, End: position.Position{Line: 1, Column: 1}},
			End:   position.Span{Start: position.Position{Line: 10, Column: 1}, End: position.Position{Line: 10, Column: 1}},
			Named: "local",
		},
		Permissions: RegionPermissions{
			Read:    true,
			Write:   true,
			Execute: false,
			Share:   false,
		},
		Size: 1024,
	}

	regions1.AddRegion(region)

	if !regions1.HasRegion(1) {
		t.Error("Region set should contain added region")
	}

	// Test region union.
	regions2 := NewRegionSet()
	region2 := Region{
		ID:   2,
		Kind: RegionKindHeap,
		Lifetime: Lifetime{
			Named: "heap",
		},
		Permissions: RegionPermissions{
			Read:  true,
			Write: true,
		},
		Size: 2048,
	}
	regions2.AddRegion(region2)

	combined := regions1.Union(regions2)

	if !combined.HasRegion(1) {
		t.Error("Combined region set should contain first region")
	}

	if !combined.HasRegion(2) {
		t.Error("Combined region set should contain second region")
	}
}

// Test type assignability.
func TestTypeAssignability(t *testing.T) {
	// Create type infos.
	i32Type := TypeInfo{ID: 1, Kind: TypeKindInteger, Name: "i32", Size: 4}
	i64Type := TypeInfo{ID: 2, Kind: TypeKindInteger, Name: "i64", Size: 8}
	boolType := TypeInfo{ID: 3, Kind: TypeKindBoolean, Name: "bool", Size: 1}

	// Test same type assignability.
	if !IsAssignableTo(i32Type, i32Type) {
		t.Error("Same types should be assignable")
	}

	// Test integer widening.
	if !IsAssignableTo(i32Type, i64Type) {
		t.Error("i32 should be assignable to i64")
	}

	// Test incompatible types.
	if IsAssignableTo(boolType, i32Type) {
		t.Error("bool should not be assignable to i32")
	}
}

// Test common type resolution.
func TestCommonTypeResolution(t *testing.T) {
	i32Type := TypeInfo{ID: 1, Kind: TypeKindInteger, Name: "i32", Size: 4}
	i64Type := TypeInfo{ID: 2, Kind: TypeKindInteger, Name: "i64", Size: 8}
	f64Type := TypeInfo{ID: 3, Kind: TypeKindFloat, Name: "f64", Size: 8}

	// Test integer promotion.
	common := GetCommonType(i32Type, i64Type)
	if common.ID != i64Type.ID {
		t.Errorf("Common type of i32 and i64 should be i64, got %s", common.Name)
	}

	// Test float promotion.
	common = GetCommonType(i32Type, f64Type)
	if common.ID != f64Type.ID {
		t.Errorf("Common type of i32 and f64 should be f64, got %s", common.Name)
	}
}

// Benchmark HIR program creation.
func BenchmarkHIRProgramCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		program := NewHIRProgram()
		_ = program
	}
}

// Benchmark AST to HIR conversion.
func BenchmarkASTToHIRConversion(b *testing.B) {
	// Create a simple AST program.
	astProgram := &ast.Program{
		Span: position.Span{
			Start: position.Position{Line: 1, Column: 1},
			End:   position.Position{Line: 10, Column: 1},
		},
		Declarations: []ast.Declaration{},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		converter := NewASTToHIRConverter()
		_, _ = converter.ConvertProgram(astProgram)
	}
}

// Benchmark HIR verification.
func BenchmarkHIRVerification(b *testing.B) {
	// Create a HIR program.
	program := NewHIRProgram()
	mainModule := &HIRModule{
		ID:           generateNodeID(),
		ModuleID:     1,
		Name:         "main",
		Declarations: []HIRDeclaration{},
		Exports:      []string{},
		Imports:      []ImportInfo{},
		Metadata:     IRMetadata{},
		Span: position.Span{
			Start: position.Position{Line: 1, Column: 1},
			End:   position.Position{Line: 10, Column: 1},
		},
	}
	program.Modules[1] = mainModule

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		verifier := NewHIRVerifier()
		_, _ = verifier.VerifyProgram(program)
	}
}
