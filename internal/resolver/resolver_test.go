// Symbol resolution tests for the Orizon programming language.
// This file provides comprehensive tests for symbol resolution and scope management.

package resolver

import (
	"fmt"
	"testing"

	"github.com/orizon-lang/orizon/internal/hir"
	"github.com/orizon-lang/orizon/internal/position"
)

// Test symbol table creation.
func TestSymbolTableCreation(t *testing.T) {
	st := NewSymbolTable()

	if st == nil {
		t.Fatal("NewSymbolTable() returned nil")
	}

	if st.rootScopeID == 0 {
		t.Error("Root scope ID should be non-zero")
	}

	if st.currentScope != st.rootScopeID {
		t.Error("Current scope should be root scope initially")
	}

	// Check root scope exists.
	rootScope, err := st.GetScope(st.rootScopeID)
	if err != nil {
		t.Fatalf("Failed to get root scope: %v", err)
	}

	if rootScope.Kind != ScopeKindGlobal {
		t.Errorf("Root scope should be global, got %s", rootScope.Kind)
	}

	if rootScope.Name != "global" {
		t.Errorf("Root scope name should be 'global', got '%s'", rootScope.Name)
	}
}

// Test scope creation and management.
func TestScopeManagement(t *testing.T) {
	st := NewSymbolTable()

	span := position.Span{
		Start: position.Position{Line: 1, Column: 1},
		End:   position.Position{Line: 10, Column: 1},
	}

	// Create a function scope.
	funcScope := st.CreateScope(ScopeKindFunction, "testFunc", span)

	if funcScope == 0 {
		t.Error("CreateScope should return non-zero scope ID")
	}

	// Enter the function scope.
	err := st.EnterScope(funcScope)
	if err != nil {
		t.Fatalf("Failed to enter scope: %v", err)
	}

	if st.GetCurrentScope() != funcScope {
		t.Error("Current scope should be function scope")
	}

	// Create a block scope inside function.
	blockScope := st.CreateScope(ScopeKindBlock, "block", span)

	err = st.EnterScope(blockScope)
	if err != nil {
		t.Fatalf("Failed to enter block scope: %v", err)
	}

	// Check scope hierarchy.
	blockScopeObj, err := st.GetScope(blockScope)
	if err != nil {
		t.Fatalf("Failed to get block scope: %v", err)
	}

	if blockScopeObj.ParentID == nil {
		t.Error("Block scope should have parent")
	} else if *blockScopeObj.ParentID != funcScope {
		t.Errorf("Block scope parent should be function scope (%d), got %d", funcScope, *blockScopeObj.ParentID)
	}

	// Exit scopes.
	err = st.ExitScope()
	if err != nil {
		t.Fatalf("Failed to exit block scope: %v", err)
	}

	currentScope := st.GetCurrentScope()
	if currentScope != funcScope {
		t.Errorf("Should be back in function scope (%d), got %d", funcScope, currentScope)
	}

	err = st.ExitScope()
	if err != nil {
		t.Fatalf("Failed to exit function scope: %v", err)
	}

	currentScope = st.GetCurrentScope()
	if currentScope != st.rootScopeID {
		t.Errorf("Should be back in root scope (%d), got %d", st.rootScopeID, currentScope)
	}
} // Test symbol definition and lookup
func TestSymbolDefinitionAndLookup(t *testing.T) {
	st := NewSymbolTable()

	span := position.Span{
		Start: position.Position{Line: 1, Column: 1},
		End:   position.Position{Line: 1, Column: 10},
	}

	// Define a variable symbol.
	symbol := &Symbol{
		Name:       "testVar",
		Kind:       SymbolKindVariable,
		Type:       hir.TypeInfo{Kind: hir.TypeKindInteger, Name: "i32"},
		Visibility: VisibilityPublic,
		DeclSpan:   span,
		IsMutable:  true,
	}

	err := st.DefineSymbol(symbol)
	if err != nil {
		t.Fatalf("Failed to define symbol: %v", err)
	}

	// Look up the symbol.
	foundSymbol, err := st.LookupSymbol("testVar")
	if err != nil {
		t.Fatalf("Failed to lookup symbol: %v", err)
	}

	if foundSymbol.Name != "testVar" {
		t.Errorf("Expected symbol name 'testVar', got '%s'", foundSymbol.Name)
	}

	if foundSymbol.Kind != SymbolKindVariable {
		t.Errorf("Expected variable symbol, got %s", foundSymbol.Kind)
	}

	if foundSymbol.Type.Name != "i32" {
		t.Errorf("Expected type 'i32', got '%s'", foundSymbol.Type.Name)
	}

	// Test undefined symbol lookup.
	_, err = st.LookupSymbol("undefinedVar")
	if err == nil {
		t.Error("Lookup of undefined symbol should return error")
	}
}

// Test symbol shadowing.
func TestSymbolShadowing(t *testing.T) {
	st := NewSymbolTable()
	st.allowShadowing = true // Enable shadowing for this test

	span := position.Span{
		Start: position.Position{Line: 1, Column: 1},
		End:   position.Position{Line: 1, Column: 10},
	}

	// Define a variable in global scope.
	globalSymbol := &Symbol{
		Name:       "x",
		Kind:       SymbolKindVariable,
		Type:       hir.TypeInfo{Kind: hir.TypeKindInteger, Name: "i32"},
		Visibility: VisibilityPublic,
		DeclSpan:   span,
	}

	err := st.DefineSymbol(globalSymbol)
	if err != nil {
		t.Fatalf("Failed to define global symbol: %v", err)
	}

	// Create function scope.
	funcScope := st.CreateScope(ScopeKindFunction, "func", span)
	st.EnterScope(funcScope)

	// Define a variable with same name in function scope.
	localSymbol := &Symbol{
		Name:       "x",
		Kind:       SymbolKindVariable,
		Type:       hir.TypeInfo{Kind: hir.TypeKindString, Name: "string"},
		Visibility: VisibilityPrivate,
		DeclSpan:   span,
	}

	err = st.DefineSymbol(localSymbol)
	if err != nil {
		t.Fatalf("Failed to define local symbol: %v", err)
	}

	// Clear caches to ensure proper lookup.
	st.invalidateCache("x")

	// Lookup should find local symbol.
	foundSymbol, err := st.LookupSymbol("x")
	if err != nil {
		t.Fatalf("Failed to lookup symbol: %v", err)
	}

	if foundSymbol.Type.Name != "string" {
		t.Errorf("Expected local symbol (string), got %s", foundSymbol.Type.Name)
	}

	// Exit function scope.
	st.ExitScope()

	// Clear caches again.
	st.invalidateCache("x")

	// Lookup should now find global symbol.
	foundSymbol, err = st.LookupSymbol("x")
	if err != nil {
		t.Fatalf("Failed to lookup symbol after scope exit: %v", err)
	}

	if foundSymbol.Type.Name != "i32" {
		t.Errorf("Expected global symbol (i32), got %s", foundSymbol.Type.Name)
	}
} // Test resolver creation
func TestResolverCreation(t *testing.T) {
	st := NewSymbolTable()
	resolver := NewResolver(st)

	if resolver == nil {
		t.Fatal("NewResolver() returned nil")
	}

	if resolver.symbolTable != st {
		t.Error("Resolver should reference the symbol table")
	}

	if !resolver.config.StrictTypeChecking {
		t.Error("Resolver should have strict type checking enabled by default")
	}

	if resolver.config.AllowShadowing {
		t.Error("Resolver should not allow shadowing by default")
	}
}

// Test HIR program resolution.
func TestHIRProgramResolution(t *testing.T) {
	st := NewSymbolTable()
	resolver := NewResolver(st)

	// Create a simple HIR program.
	program := hir.NewHIRProgram()

	// Create main module.
	mainModule := &hir.HIRModule{
		ID:           hir.NodeID(1),
		ModuleID:     1,
		Name:         "main",
		Declarations: []hir.HIRDeclaration{},
		Exports:      []string{},
		Imports:      []hir.ImportInfo{},
		Metadata:     hir.IRMetadata{},
		Span: position.Span{
			Start: position.Position{Line: 1, Column: 1},
			End:   position.Position{Line: 10, Column: 1},
		},
	}

	program.Modules[1] = mainModule

	// Resolve the program.
	err := resolver.ResolveProgram(program)
	if err != nil {
		t.Fatalf("Failed to resolve HIR program: %v", err)
	}

	// Check for any resolution errors.
	errors := st.GetErrors()
	if len(errors) > 0 {
		t.Errorf("Resolution had %d errors: %v", len(errors), errors)
	}
}

// Test function resolution.
func TestFunctionResolution(t *testing.T) {
	st := NewSymbolTable()
	resolver := NewResolver(st)

	span := position.Span{
		Start: position.Position{Line: 1, Column: 1},
		End:   position.Position{Line: 5, Column: 1},
	}

	// Create function declaration.
	funcDecl := &hir.HIRFunctionDeclaration{
		ID:         hir.NodeID(1),
		Name:       "testFunc",
		Parameters: []*hir.HIRParameter{},
		ReturnType: &hir.HIRBasicType{
			ID:   hir.NodeID(2),
			Kind: hir.TypeKindVoid,
			Name: "void",
			Span: span,
		},
		Body: &hir.HIRBlockStatement{
			ID:         hir.NodeID(3),
			Statements: []hir.HIRStatement{},
			Effects:    hir.NewEffectSet(),
			Regions:    hir.NewRegionSet(),
			Metadata:   hir.IRMetadata{},
			Span:       span,
		},
		Generic:    false,
		TypeParams: []hir.TypeInfo{},
		Effects:    hir.NewEffectSet(),
		Regions:    hir.NewRegionSet(),
		Metadata:   hir.IRMetadata{},
		Span:       span,
	}

	// Create module and add function.
	module := &hir.HIRModule{
		ID:           hir.NodeID(100),
		ModuleID:     1,
		Name:         "test",
		Declarations: []hir.HIRDeclaration{funcDecl},
		Exports:      []string{},
		Imports:      []hir.ImportInfo{},
		Metadata:     hir.IRMetadata{},
		Span:         span,
	}

	resolver.currentModule = module

	// Collect function symbol.
	err := resolver.collectFunctionSymbol(funcDecl)
	if err != nil {
		t.Fatalf("Failed to collect function symbol: %v", err)
	}

	// Resolve function declaration.
	err = resolver.resolveFunctionDeclaration(funcDecl)
	if err != nil {
		t.Fatalf("Failed to resolve function declaration: %v", err)
	}

	// Check that function symbol was defined.
	symbol, err := st.LookupSymbol("testFunc")
	if err != nil {
		t.Fatalf("Function symbol not found: %v", err)
	}

	if symbol.Kind != SymbolKindFunction {
		t.Errorf("Expected function symbol, got %s", symbol.Kind)
	}

	if symbol.Name != "testFunc" {
		t.Errorf("Expected function name 'testFunc', got '%s'", symbol.Name)
	}
}

// Test variable resolution.
func TestVariableResolution(t *testing.T) {
	st := NewSymbolTable()
	resolver := NewResolver(st)

	span := position.Span{
		Start: position.Position{Line: 1, Column: 1},
		End:   position.Position{Line: 1, Column: 15},
	}

	// Create variable declaration.
	varDecl := &hir.HIRVariableDeclaration{
		ID:   hir.NodeID(1),
		Name: "testVar",
		Type: &hir.HIRBasicType{
			ID:   hir.NodeID(2),
			Kind: hir.TypeKindInteger,
			Name: "i32",
			Span: span,
		},
		Initializer: &hir.HIRLiteral{
			ID:       hir.NodeID(3),
			Value:    42,
			Type:     hir.TypeInfo{Kind: hir.TypeKindInteger, Name: "i32"},
			Metadata: hir.IRMetadata{},
			Span:     span,
		},
		Mutable:  false,
		Effects:  hir.NewEffectSet(),
		Regions:  hir.NewRegionSet(),
		Metadata: hir.IRMetadata{},
		Span:     span,
	}

	// Create module.
	module := &hir.HIRModule{
		ID:       hir.NodeID(100),
		ModuleID: 1,
		Name:     "test",
		Span:     span,
	}

	resolver.currentModule = module

	// Collect variable symbol.
	err := resolver.collectVariableSymbol(varDecl)
	if err != nil {
		t.Fatalf("Failed to collect variable symbol: %v", err)
	}

	// Resolve variable declaration.
	err = resolver.resolveVariableDeclaration(varDecl)
	if err != nil {
		t.Fatalf("Failed to resolve variable declaration: %v", err)
	}

	// Check that variable symbol was defined.
	symbol, err := st.LookupSymbol("testVar")
	if err != nil {
		t.Fatalf("Variable symbol not found: %v", err)
	}

	if symbol.Kind != SymbolKindVariable {
		t.Errorf("Expected variable symbol, got %s", symbol.Kind)
	}

	if symbol.Name != "testVar" {
		t.Errorf("Expected variable name 'testVar', got '%s'", symbol.Name)
	}

	if symbol.IsMutable {
		t.Error("Variable should not be mutable")
	}
}

// Test import resolution.
func TestImportResolution(t *testing.T) {
	st := NewSymbolTable()
	resolver := NewResolver(st)

	span := position.Span{
		Start: position.Position{Line: 1, Column: 1},
		End:   position.Position{Line: 1, Column: 20},
	}

	// Create import info.
	importInfo := &hir.ImportInfo{
		ModuleName: "std.io",
		Alias:      "io",
		Items:      []string{"print", "println"},
		Span:       span,
	}

	// Create module.
	module := &hir.HIRModule{
		ID:       hir.NodeID(100),
		ModuleID: 1,
		Name:     "test",
		Span:     span,
	}

	resolver.currentModule = module

	// Resolve import.
	err := resolver.resolveImport(importInfo)
	if err != nil {
		t.Fatalf("Failed to resolve import: %v", err)
	}

	// Check that import was added.
	if len(st.imports) == 0 {
		t.Error("Import should have been added to symbol table")
	}

	importFound := false

	for path := range st.imports {
		if path == "std.io" {
			importFound = true

			break
		}
	}

	if !importFound {
		t.Error("Import 'std.io' not found in symbol table")
	}
}

// Test type inference.
func TestTypeInference(t *testing.T) {
	st := NewSymbolTable()
	resolver := NewResolver(st)

	span := position.Span{
		Start: position.Position{Line: 1, Column: 1},
		End:   position.Position{Line: 1, Column: 5},
	}

	// Test integer literal inference.
	intLiteral := &hir.HIRLiteral{
		ID:       hir.NodeID(1),
		Value:    42,
		Type:     hir.TypeInfo{Kind: hir.TypeKindInteger, Name: "i32"},
		Metadata: hir.IRMetadata{},
		Span:     span,
	}

	inferredType := resolver.inferLiteralType(intLiteral)
	if inferredType == nil {
		t.Fatal("Failed to infer integer literal type")
	}

	if inferredType.Kind != hir.TypeKindInteger {
		t.Errorf("Expected integer type, got %s", inferredType.Kind)
	}

	// Test string literal inference.
	stringLiteral := &hir.HIRLiteral{
		ID:       hir.NodeID(2),
		Value:    "hello",
		Type:     hir.TypeInfo{Kind: hir.TypeKindString, Name: "string"},
		Metadata: hir.IRMetadata{},
		Span:     span,
	}

	inferredType = resolver.inferLiteralType(stringLiteral)
	if inferredType == nil {
		t.Fatal("Failed to infer string literal type")
	}

	if inferredType.Kind != hir.TypeKindString {
		t.Errorf("Expected string type, got %s", inferredType.Kind)
	}
}

// Test symbol table statistics.
func TestSymbolTableStatistics(t *testing.T) {
	st := NewSymbolTable()

	span := position.Span{
		Start: position.Position{Line: 1, Column: 1},
		End:   position.Position{Line: 1, Column: 10},
	}

	// Add some symbols.
	for i := 0; i < 5; i++ {
		symbol := &Symbol{
			Name:       fmt.Sprintf("symbol%d", i),
			Kind:       SymbolKindVariable,
			Type:       hir.TypeInfo{Kind: hir.TypeKindInteger, Name: "i32"},
			Visibility: VisibilityPublic,
			DeclSpan:   span,
		}
		st.DefineSymbol(symbol)
	}

	// Perform some lookups.
	for i := 0; i < 3; i++ {
		st.LookupSymbol(fmt.Sprintf("symbol%d", i))
	}

	// Get statistics.
	stats := st.GetStatistics()

	if stats.TotalSymbols != 5 {
		t.Errorf("Expected 5 symbols, got %d", stats.TotalSymbols)
	}

	if stats.LookupCount < 3 {
		t.Errorf("Expected at least 3 lookups, got %d", stats.LookupCount)
	}

	if stats.TotalScopes < 1 {
		t.Errorf("Expected at least 1 scope, got %d", stats.TotalScopes)
	}
}

// Benchmark symbol lookup.
func BenchmarkSymbolLookup(b *testing.B) {
	st := NewSymbolTable()

	span := position.Span{
		Start: position.Position{Line: 1, Column: 1},
		End:   position.Position{Line: 1, Column: 10},
	}

	// Add many symbols.
	for i := 0; i < 1000; i++ {
		symbol := &Symbol{
			Name:       fmt.Sprintf("symbol%d", i),
			Kind:       SymbolKindVariable,
			Type:       hir.TypeInfo{Kind: hir.TypeKindInteger, Name: "i32"},
			Visibility: VisibilityPublic,
			DeclSpan:   span,
		}
		st.DefineSymbol(symbol)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		symbolName := fmt.Sprintf("symbol%d", i%1000)
		_, _ = st.LookupSymbol(symbolName)
	}
}

// Benchmark scope management.
func BenchmarkScopeManagement(b *testing.B) {
	st := NewSymbolTable()

	span := position.Span{
		Start: position.Position{Line: 1, Column: 1},
		End:   position.Position{Line: 10, Column: 1},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create scope.
		scopeID := st.CreateScope(ScopeKindFunction, fmt.Sprintf("func%d", i), span)

		// Enter scope.
		st.EnterScope(scopeID)

		// Exit scope.
		st.ExitScope()
	}
}
