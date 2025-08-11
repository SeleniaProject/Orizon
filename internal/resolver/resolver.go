// HIR symbol resolver for the Orizon programming language
// This file implements the main resolution logic for HIR programs

package resolver

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/hir"
)

// Resolver performs symbol resolution on HIR programs
type Resolver struct {
	symbolTable     *SymbolTable
	currentModule   *hir.HIRModule
	currentFunction *hir.HIRFunctionDeclaration

	// Resolution state
	resolutionStack []ResolutionContext
	genericContext  []GenericScope

	// Configuration
	config ResolverConfig
}

// ResolutionContext represents the current resolution context
type ResolutionContext struct {
	Kind     ContextKind
	ScopeID  ScopeID
	NodeID   hir.NodeID
	Name     string
	TypeInfo *hir.TypeInfo
}

// ContextKind represents the kind of resolution context
type ContextKind int

const (
	ContextKindGlobal ContextKind = iota
	ContextKindModule
	ContextKindFunction
	ContextKindBlock
	ContextKindExpression
	ContextKindType
)

// GenericScope represents a scope with generic type parameters
type GenericScope struct {
	Parameters map[string]*GenericParameter
	ParentID   *ScopeID
}

// ResolverConfig contains resolver configuration
type ResolverConfig struct {
	StrictTypeChecking     bool
	AllowShadowing         bool
	RequireExplicitTypes   bool
	EnableGenericInference bool
}

// NewResolver creates a new resolver
func NewResolver(symbolTable *SymbolTable) *Resolver {
	return &Resolver{
		symbolTable:     symbolTable,
		resolutionStack: []ResolutionContext{},
		genericContext:  []GenericScope{},
		config: ResolverConfig{
			StrictTypeChecking:     true,
			AllowShadowing:         false,
			RequireExplicitTypes:   false,
			EnableGenericInference: true,
		},
	}
}

// ResolveProgram resolves symbols in an HIR program
func (r *Resolver) ResolveProgram(program *hir.HIRProgram) error {
	// Enter global scope
	globalContext := ResolutionContext{
		Kind:    ContextKindGlobal,
		ScopeID: r.symbolTable.GetCurrentScope(),
		NodeID:  program.ID,
		Name:    "global",
	}
	r.pushContext(globalContext)
	defer r.popContext()

	// First pass: collect all module symbols
	for _, module := range program.Modules {
		if err := r.collectModuleSymbols(module); err != nil {
			return fmt.Errorf("failed to collect module symbols: %w", err)
		}
	}

	// Second pass: resolve all symbols
	for _, module := range program.Modules {
		if err := r.resolveModule(module); err != nil {
			return fmt.Errorf("failed to resolve module %s: %w", module.Name, err)
		}
	}

	// Third pass: validate all resolutions
	if err := r.validateResolutions(program); err != nil {
		return fmt.Errorf("failed to validate resolutions: %w", err)
	}

	return nil
}

// collectModuleSymbols collects all symbols from modules without resolving them
func (r *Resolver) collectModuleSymbols(module *hir.HIRModule) error {
	// Create module scope
	moduleScope := r.symbolTable.CreateScope(ScopeKindModule, module.Name, module.Span)
	r.symbolTable.EnterScope(moduleScope)
	defer r.symbolTable.ExitScope()

	r.currentModule = module

	// Collect symbols from declarations
	for _, decl := range module.Declarations {
		if err := r.collectDeclarationSymbol(decl); err != nil {
			return err
		}
	}

	return nil
}

// collectDeclarationSymbol collects a symbol from a declaration
func (r *Resolver) collectDeclarationSymbol(decl hir.HIRDeclaration) error {
	switch d := decl.(type) {
	case *hir.HIRFunctionDeclaration:
		return r.collectFunctionSymbol(d)
	case *hir.HIRVariableDeclaration:
		return r.collectVariableSymbol(d)
	case *hir.HIRTypeDeclaration:
		return r.collectTypeSymbol(d)
	case *hir.HIRConstDeclaration:
		return r.collectConstantSymbol(d)
	default:
		return fmt.Errorf("unknown declaration type: %T", decl)
	}
}

// collectFunctionSymbol collects a function symbol
func (r *Resolver) collectFunctionSymbol(funcDecl *hir.HIRFunctionDeclaration) error {
	symbol := &Symbol{
		Name:       funcDecl.Name,
		Kind:       SymbolKindFunction,
		Type:       r.buildFunctionType(funcDecl),
		Visibility: VisibilityPublic, // Default visibility for now
		DeclSpan:   funcDecl.Span,
		ScopeID:    r.symbolTable.GetCurrentScope(),
		ModuleID:   r.currentModule.ID,
		HIRNode:    funcDecl,
		IsGeneric:  funcDecl.Generic,
		IsExported: true, // Default for now
	}

	// Handle generic parameters
	if symbol.IsGeneric {
		for _, param := range funcDecl.TypeParams {
			genericParam := GenericParameter{
				Name:        param.Name,
				Constraints: []hir.TypeInfo{param},
				DefaultType: &param,
				Span:        funcDecl.Span,
			}
			symbol.TypeParameters = append(symbol.TypeParameters, genericParam)
		}
	}

	return r.symbolTable.DefineSymbol(symbol)
}

// collectVariableSymbol collects a variable symbol
func (r *Resolver) collectVariableSymbol(varDecl *hir.HIRVariableDeclaration) error {
	symbol := &Symbol{
		Name:       varDecl.Name,
		Kind:       SymbolKindVariable,
		Type:       varDecl.Type.GetType(),
		Visibility: VisibilityPublic, // Default visibility for now
		DeclSpan:   varDecl.Span,
		ScopeID:    r.symbolTable.GetCurrentScope(),
		ModuleID:   r.currentModule.ID,
		HIRNode:    varDecl,
		IsMutable:  varDecl.Mutable,
		IsExported: true, // Default for now
	}

	return r.symbolTable.DefineSymbol(symbol)
}

// collectTypeSymbol collects a type symbol
func (r *Resolver) collectTypeSymbol(typeDecl *hir.HIRTypeDeclaration) error {
	symbol := &Symbol{
		Name:       typeDecl.Name,
		Kind:       SymbolKindType,
		Type:       typeDecl.Type.GetType(),
		Visibility: VisibilityPublic, // Default visibility for now
		DeclSpan:   typeDecl.Span,
		ScopeID:    r.symbolTable.GetCurrentScope(),
		ModuleID:   r.currentModule.ID,
		HIRNode:    typeDecl,
		IsGeneric:  typeDecl.Generic,
		IsExported: true, // Default for now
	}

	return r.symbolTable.DefineSymbol(symbol)
}

// collectConstantSymbol collects a constant symbol
func (r *Resolver) collectConstantSymbol(constDecl *hir.HIRConstDeclaration) error {
	symbol := &Symbol{
		Name:       constDecl.Name,
		Kind:       SymbolKindConstant,
		Type:       constDecl.Type.GetType(),
		Visibility: VisibilityPublic, // Default visibility for now
		DeclSpan:   constDecl.Span,
		ScopeID:    r.symbolTable.GetCurrentScope(),
		ModuleID:   r.currentModule.ID,
		HIRNode:    constDecl,
		IsMutable:  false,
		IsExported: true, // Default for now
	}

	return r.symbolTable.DefineSymbol(symbol)
}

// resolveModule resolves all symbols in a module
func (r *Resolver) resolveModule(module *hir.HIRModule) error {
	// Create module scope and enter it
	moduleScope := r.symbolTable.CreateScope(ScopeKindModule, module.Name, module.Span)
	r.symbolTable.EnterScope(moduleScope)
	defer r.symbolTable.ExitScope()

	r.currentModule = module

	// Create module context
	moduleContext := ResolutionContext{
		Kind:    ContextKindModule,
		ScopeID: moduleScope,
		NodeID:  module.ID,
		Name:    module.Name,
	}
	r.pushContext(moduleContext)
	defer r.popContext()

	// Resolve imports first
	for _, importInfo := range module.Imports {
		if err := r.resolveImport(&importInfo); err != nil {
			return err
		}
	}

	// Resolve declarations
	for _, decl := range module.Declarations {
		if err := r.resolveDeclaration(decl); err != nil {
			return err
		}
	}

	return nil
}

// resolveImport resolves an import
func (r *Resolver) resolveImport(importInfo *hir.ImportInfo) error {
	// Create import info for symbol table
	stImportInfo := &ImportInfo{
		ModulePath:      importInfo.ModuleName,
		Alias:           importInfo.Alias,
		ImportedSymbols: make(map[string]string),
		IsWildcard:      false, // Will be set based on specific import syntax
		ImportSpan:      importInfo.Span,
		ModuleID:        r.currentModule.ID,
	}

	// Add to symbol table
	return r.symbolTable.AddImport(stImportInfo)
} // resolveDeclaration resolves a declaration
func (r *Resolver) resolveDeclaration(decl hir.HIRDeclaration) error {
	switch d := decl.(type) {
	case *hir.HIRFunctionDeclaration:
		return r.resolveFunctionDeclaration(d)
	case *hir.HIRVariableDeclaration:
		return r.resolveVariableDeclaration(d)
	case *hir.HIRTypeDeclaration:
		return r.resolveTypeDeclaration(d)
	case *hir.HIRConstDeclaration:
		return r.resolveConstantDeclaration(d)
	default:
		return fmt.Errorf("unknown declaration type: %T", decl)
	}
}

// resolveFunctionDeclaration resolves a function declaration
func (r *Resolver) resolveFunctionDeclaration(funcDecl *hir.HIRFunctionDeclaration) error {
	// Create function scope
	funcScope := r.symbolTable.CreateScope(ScopeKindFunction, funcDecl.Name, funcDecl.Span)
	r.symbolTable.EnterScope(funcScope)
	defer r.symbolTable.ExitScope()

	r.currentFunction = funcDecl

	// Create function context
	funcContext := ResolutionContext{
		Kind:    ContextKindFunction,
		ScopeID: funcScope,
		NodeID:  funcDecl.ID,
		Name:    funcDecl.Name,
	}
	r.pushContext(funcContext)
	defer r.popContext()

	// Handle generic parameters
	if funcDecl.Generic {
		if err := r.enterGenericScope(funcDecl.TypeParams); err != nil {
			return err
		}
		defer r.exitGenericScope()
	}

	// Add parameters to scope
	for _, param := range funcDecl.Parameters {
		paramSymbol := &Symbol{
			Name:       param.Name,
			Kind:       SymbolKindParameter,
			Type:       param.Type.GetType(),
			Visibility: VisibilityPrivate,
			DeclSpan:   param.Span,
			ScopeID:    funcScope,
			ModuleID:   r.currentModule.ID,
			HIRNode:    param,
			IsMutable:  false, // Parameters are immutable by default
		}

		if err := r.symbolTable.DefineSymbol(paramSymbol); err != nil {
			return err
		}
	}

	// Resolve return type
	if funcDecl.ReturnType != nil {
		if err := r.resolveType(funcDecl.ReturnType); err != nil {
			return err
		}
	}

	// Resolve function body
	if funcDecl.Body != nil {
		if err := r.resolveStatement(funcDecl.Body); err != nil {
			return err
		}
	}

	return nil
}

// resolveVariableDeclaration resolves a variable declaration
func (r *Resolver) resolveVariableDeclaration(varDecl *hir.HIRVariableDeclaration) error {
	// Resolve type if present
	if varDecl.Type != nil {
		if err := r.resolveType(varDecl.Type); err != nil {
			return err
		}
	}

	// Resolve initializer if present
	if varDecl.Initializer != nil {
		if err := r.resolveExpression(varDecl.Initializer); err != nil {
			return err
		}

		// Type inference if no explicit type
		if varDecl.Type == nil && r.config.EnableGenericInference {
			inferredType := r.inferExpressionType(varDecl.Initializer)
			if inferredType != nil {
				// Note: In a full implementation, we would update the HIR node here
				// For now, we just note that type inference would occur
			}
		}
	}

	return nil
}

// resolveTypeDeclaration resolves a type declaration
func (r *Resolver) resolveTypeDeclaration(typeDecl *hir.HIRTypeDeclaration) error {
	// Resolve the type definition
	return r.resolveType(typeDecl.Type)
}

// resolveConstantDeclaration resolves a constant declaration
func (r *Resolver) resolveConstantDeclaration(constDecl *hir.HIRConstDeclaration) error {
	// Resolve type
	if err := r.resolveType(constDecl.Type); err != nil {
		return err
	}

	// Resolve value
	return r.resolveExpression(constDecl.Value)
}

// resolveStatement resolves a statement
func (r *Resolver) resolveStatement(stmt hir.HIRStatement) error {
	switch s := stmt.(type) {
	case *hir.HIRBlockStatement:
		return r.resolveBlockStatement(s)
	case *hir.HIRExpressionStatement:
		return r.resolveExpression(s.Expression)
	case *hir.HIRReturnStatement:
		if s.Expression != nil {
			return r.resolveExpression(s.Expression)
		}
		return nil
	default:
		return fmt.Errorf("unknown statement type: %T", stmt)
	}
}

// resolveBlockStatement resolves a block statement
func (r *Resolver) resolveBlockStatement(block *hir.HIRBlockStatement) error {
	// Create block scope
	blockScope := r.symbolTable.CreateScope(ScopeKindBlock, "block", block.Span)
	r.symbolTable.EnterScope(blockScope)
	defer r.symbolTable.ExitScope()

	// Resolve all statements
	for _, stmt := range block.Statements {
		if err := r.resolveStatement(stmt); err != nil {
			return err
		}
	}

	return nil
}

// resolveExpression resolves an expression
func (r *Resolver) resolveExpression(expr hir.HIRExpression) error {
	switch e := expr.(type) {
	case *hir.HIRIdentifier:
		return r.resolveIdentifier(e)
	case *hir.HIRLiteral:
		return nil // Literals don't need resolution
	case *hir.HIRBinaryExpression:
		return r.resolveBinaryExpression(e)
	case *hir.HIRUnaryExpression:
		return r.resolveUnaryExpression(e)
	case *hir.HIRCallExpression:
		return r.resolveCallExpression(e)
	default:
		return fmt.Errorf("unknown expression type: %T", expr)
	}
}

// resolveIdentifier resolves an identifier
func (r *Resolver) resolveIdentifier(id *hir.HIRIdentifier) error {
	symbol, err := r.symbolTable.LookupSymbol(id.Name)
	if err != nil {
		return err
	}

	// Store resolved symbol reference (would need to extend HIRIdentifier)
	// id.ResolvedSymbol = symbol

	// Update usage count
	symbol.UsageCount++
	symbol.LastUsedSpan = id.Span

	return nil
}

// resolveBinaryExpression resolves a binary expression
func (r *Resolver) resolveBinaryExpression(binary *hir.HIRBinaryExpression) error {
	if err := r.resolveExpression(binary.Left); err != nil {
		return err
	}
	return r.resolveExpression(binary.Right)
}

// resolveUnaryExpression resolves a unary expression
func (r *Resolver) resolveUnaryExpression(unary *hir.HIRUnaryExpression) error {
	return r.resolveExpression(unary.Operand)
}

// resolveCallExpression resolves a call expression
func (r *Resolver) resolveCallExpression(call *hir.HIRCallExpression) error {
	// Resolve function expression
	if err := r.resolveExpression(call.Function); err != nil {
		return err
	}

	// Resolve arguments
	for _, arg := range call.Arguments {
		if err := r.resolveExpression(arg); err != nil {
			return err
		}
	}

	return nil
}

// resolveType resolves a type reference
func (r *Resolver) resolveType(hirType hir.HIRType) error {
	// For now, just validate that the type exists
	// More sophisticated type resolution will be added later
	return nil
} // validateResolutions validates all symbol resolutions
func (r *Resolver) validateResolutions(program *hir.HIRProgram) error {
	// Check for unused symbols
	for _, module := range program.Modules {
		if err := r.checkUnusedSymbols(module); err != nil {
			return err
		}
	}

	return nil
}

// checkUnusedSymbols checks for unused symbols in a module
func (r *Resolver) checkUnusedSymbols(module *hir.HIRModule) error {
	// Get module symbols
	moduleSymbols := r.symbolTable.moduleSymbols[module.ID]

	for _, symbol := range moduleSymbols {
		if symbol.UsageCount == 0 && !symbol.IsExported {
			warning := ResolutionWarning{
				Kind:    WarningKindUnusedSymbol,
				Message: fmt.Sprintf("unused %s '%s'", symbol.Kind, symbol.Name),
				Span:    symbol.DeclSpan,
				Symbol:  symbol.Name,
			}
			r.symbolTable.warnings = append(r.symbolTable.warnings, warning)
		}
	}

	return nil
}

// Helper methods

// pushContext pushes a resolution context onto the stack
func (r *Resolver) pushContext(context ResolutionContext) {
	r.resolutionStack = append(r.resolutionStack, context)
}

// popContext pops a resolution context from the stack
func (r *Resolver) popContext() {
	if len(r.resolutionStack) > 0 {
		r.resolutionStack = r.resolutionStack[:len(r.resolutionStack)-1]
	}
}

// getCurrentContext returns the current resolution context
func (r *Resolver) getCurrentContext() *ResolutionContext {
	if len(r.resolutionStack) == 0 {
		return nil
	}
	return &r.resolutionStack[len(r.resolutionStack)-1]
}

// enterGenericScope enters a generic scope with type parameters
func (r *Resolver) enterGenericScope(params []hir.TypeInfo) error {
	genericScope := GenericScope{
		Parameters: make(map[string]*GenericParameter),
		ParentID:   &r.symbolTable.currentScope,
	}

	for _, param := range params {
		genericParam := &GenericParameter{
			Name:        param.Name,
			Constraints: []hir.TypeInfo{param},
			DefaultType: &param,
			Span:        r.symbolTable.scopes[r.symbolTable.currentScope].Span,
		}
		genericScope.Parameters[param.Name] = genericParam
	}

	r.genericContext = append(r.genericContext, genericScope)
	return nil
}

// exitGenericScope exits the current generic scope
func (r *Resolver) exitGenericScope() {
	if len(r.genericContext) > 0 {
		r.genericContext = r.genericContext[:len(r.genericContext)-1]
	}
}

// buildFunctionType builds a function type from a function declaration
func (r *Resolver) buildFunctionType(funcDecl *hir.HIRFunctionDeclaration) hir.TypeInfo {
	// For now, return a basic function type
	// More sophisticated type building will be added later
	return hir.TypeInfo{
		ID:   hir.TypeID(funcDecl.ID),
		Kind: hir.TypeKindFunction,
		Name: fmt.Sprintf("func %s", funcDecl.Name),
	}
}

// inferExpressionType infers the type of an expression
func (r *Resolver) inferExpressionType(expr hir.HIRExpression) *hir.TypeInfo {
	switch e := expr.(type) {
	case *hir.HIRLiteral:
		return r.inferLiteralType(e)
	case *hir.HIRIdentifier:
		// In a full implementation, we would look up the identifier's type
		return nil
	case *hir.HIRBinaryExpression:
		return r.inferBinaryExpressionType(e)
	}
	return nil
}

// inferLiteralType infers the type of a literal
func (r *Resolver) inferLiteralType(literal *hir.HIRLiteral) *hir.TypeInfo {
	// Use the literal's type information directly if available
	if literal.Type.Kind != hir.TypeKindUnknown {
		return &literal.Type
	}

	// Simple type inference based on value type
	switch literal.Value.(type) {
	case int, int32, int64:
		return &hir.TypeInfo{Kind: hir.TypeKindInteger, Name: "i32"}
	case float32, float64:
		return &hir.TypeInfo{Kind: hir.TypeKindFloat, Name: "f64"}
	case bool:
		return &hir.TypeInfo{Kind: hir.TypeKindBoolean, Name: "bool"}
	case string:
		return &hir.TypeInfo{Kind: hir.TypeKindString, Name: "string"}
	default:
		return nil
	}
}

// inferBinaryExpressionType infers the type of a binary expression
func (r *Resolver) inferBinaryExpressionType(binary *hir.HIRBinaryExpression) *hir.TypeInfo {
	leftType := r.inferExpressionType(binary.Left)
	rightType := r.inferExpressionType(binary.Right)

	if leftType != nil && rightType != nil {
		return r.getCommonType(leftType, rightType)
	}

	return leftType // fallback to left type
}

// getCommonType determines the common type of two types
func (r *Resolver) getCommonType(left, right *hir.TypeInfo) *hir.TypeInfo {
	// Simple type promotion rules
	if left.Kind == right.Kind {
		return left
	}

	// Integer to float promotion
	if (left.Kind == hir.TypeKindInteger && right.Kind == hir.TypeKindFloat) ||
		(left.Kind == hir.TypeKindFloat && right.Kind == hir.TypeKindInteger) {
		return &hir.TypeInfo{Kind: hir.TypeKindFloat, Name: "f64"}
	}

	return left // fallback to left type
}
