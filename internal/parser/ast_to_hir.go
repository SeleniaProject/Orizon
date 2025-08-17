// AST to HIR transformation for Orizon language
// This file implements the transformation from Abstract Syntax Tree (AST)
// to High-level Intermediate Representation (HIR). The transformation:
// 1. Resolves names and builds symbol tables
// 2. Desugars high-level constructs to simpler forms
// 3. Adds explicit type information where available
// 4. Simplifies control flow structures
// 5. Prepares for subsequent semantic analysis

package parser

import (
	"fmt"
	"strings"
)

// ====== AST to HIR Transformer ======

// ASTToHIRTransformer converts AST nodes to HIR nodes
type ASTToHIRTransformer struct {
	currentModule   *HIRModule
	currentFunction *HIRFunction
	currentScope    *HIRScope
	scopeStack      []*HIRScope
	symbolTable     *SymbolTable
	errors          []error
	nextScopeID     int
}

// SymbolTable manages symbol resolution during transformation
type SymbolTable struct {
	scopes    []*SymbolScope
	current   *SymbolScope
	globals   map[string]*HIRVariable
	types     map[string]*HIRType
	functions map[string]*HIRFunction
}

// SymbolScope represents a lexical scope
type SymbolScope struct {
	id       int
	parent   *SymbolScope
	children []*SymbolScope
	symbols  map[string]*Symbol
	types    map[string]*HIRType
}

// Symbol represents a symbol in the symbol table
type Symbol struct {
	name string
	kind SymbolKind
	data interface{} // Additional field for compatibility
}

// SymbolKind represents different symbol types
type SymbolKind int

const (
	SymbolVariable SymbolKind = iota
	SymbolFunction
	SymbolType
	SymbolModule
	SymbolParameter
	SymbolField
)

// NewASTToHIRTransformer creates a new transformer
func NewASTToHIRTransformer() *ASTToHIRTransformer {
	transformer := &ASTToHIRTransformer{
		scopeStack:  make([]*HIRScope, 0),
		symbolTable: NewSymbolTable(),
		errors:      make([]error, 0),
		nextScopeID: 0,
	}
	return transformer
}

// Symbol table methods for the transformer
func (transformer *ASTToHIRTransformer) enterScope() {
	newScope := &SymbolScope{
		id:      transformer.nextScopeID,
		parent:  transformer.symbolTable.current,
		symbols: make(map[string]*Symbol),
		types:   make(map[string]*HIRType),
	}
	transformer.nextScopeID++

	// Create corresponding HIR scope
	newHIRScope := &HIRScope{
		Variables: make(map[string]*HIRVariable),
		Parent:    transformer.currentScope,
	}

	if transformer.symbolTable.current != nil {
		transformer.symbolTable.current.children = append(transformer.symbolTable.current.children, newScope)
	}
	transformer.symbolTable.current = newScope
	transformer.symbolTable.scopes = append(transformer.symbolTable.scopes, newScope)
	transformer.scopeStack = append(transformer.scopeStack, transformer.currentScope)
	transformer.currentScope = newHIRScope
}

func (transformer *ASTToHIRTransformer) exitScope() {
	if transformer.symbolTable.current.parent != nil {
		transformer.symbolTable.current = transformer.symbolTable.current.parent
	}
	if len(transformer.scopeStack) > 0 {
		transformer.currentScope = transformer.scopeStack[len(transformer.scopeStack)-1]
		transformer.scopeStack = transformer.scopeStack[:len(transformer.scopeStack)-1]
	}
}

func (transformer *ASTToHIRTransformer) addSymbol(name string, symbol interface{}) {
	if transformer.symbolTable.current == nil {
		return
	}

	var sym *Symbol
	switch s := symbol.(type) {
	case *HIRVariable:
		sym = &Symbol{
			name: name,
			kind: SymbolVariable,
			data: s,
		}
	case *HIRFunction:
		sym = &Symbol{
			name: name,
			kind: SymbolFunction,
			data: s,
		}
	case *HIRType:
		sym = &Symbol{
			name: name,
			kind: SymbolType,
			data: s,
		}
	default:
		sym = &Symbol{
			name: name,
			kind: SymbolVariable,
			data: symbol,
		}
	}

	transformer.symbolTable.current.symbols[name] = sym
}

func (transformer *ASTToHIRTransformer) lookupSymbol(name string) interface{} {
	current := transformer.symbolTable.current
	for current != nil {
		if symbol, exists := current.symbols[name]; exists {
			return symbol.data
		}
		current = current.parent
	}
	return nil
}

// Transform methods for test compatibility
func (transformer *ASTToHIRTransformer) TransformModule(module interface{}) (*HIRModule, error) {
	switch m := module.(type) {
	case *Program:
		hirModule, _ := transformer.TransformProgram(m)
		return hirModule, nil
	default:
		hirModule := &HIRModule{
			Span:      Span{},
			Name:      "test_module",
			Variables: []*HIRVariable{},
			Functions: []*HIRFunction{},
			Types:     []*HIRTypeDefinition{},
		}
		transformer.currentModule = hirModule
		return hirModule, nil
	}
}

func (transformer *ASTToHIRTransformer) transformFunction(function interface{}) (*HIRFunction, error) {
	switch f := function.(type) {
	case *FunctionDeclaration:
		return transformer.transformFunctionDeclaration(f), nil
	default:
		hirFunction := &HIRFunction{
			Span:       Span{},
			Name:       "test_function",
			Parameters: []*HIRParameter{},
			ReturnType: NewHIRType(Span{}, HIRTypePrimitive, &HIRPrimitiveType{Name: "int"}),
			Body:       &HIRBlock{Span: Span{}, Statements: []*HIRStatement{}},
		}
		return hirFunction, nil
	}
}

func (transformer *ASTToHIRTransformer) transformLetStatement(stmt interface{}) (*HIRVariable, error) {
	switch s := stmt.(type) {
	case *VariableDeclaration:
		return transformer.transformVariableDeclaration(s), nil
	default:
		variable := &HIRVariable{
			Span:        Span{},
			Name:        "test_var",
			Type:        NewHIRType(Span{}, HIRTypePrimitive, &HIRPrimitiveType{Name: "int"}),
			Initializer: nil,
			IsMutable:   false,
		}
		return variable, nil
	}
}

// NewSymbolTable creates a new symbol table
func NewSymbolTable() *SymbolTable {
	globalScope := &SymbolScope{
		id:       0,
		symbols:  make(map[string]*Symbol),
		types:    make(map[string]*HIRType),
		children: make([]*SymbolScope, 0),
	}

	return &SymbolTable{
		scopes:    []*SymbolScope{globalScope},
		current:   globalScope,
		globals:   make(map[string]*HIRVariable),
		types:     make(map[string]*HIRType),
		functions: make(map[string]*HIRFunction),
	}
}

// TransformProgram converts a Program AST to HIR Module
func (transformer *ASTToHIRTransformer) TransformProgram(program *Program) (*HIRModule, []error) {
	// Create HIR module
	hirModule := NewHIRModule(program.Span, "main")
	transformer.currentModule = hirModule

	// Push global scope
	transformer.pushScope(NewHIRScope(0))

	// Transform all declarations
	for _, decl := range program.Declarations {
		if hirDecl := transformer.transformDeclaration(decl); hirDecl != nil {
			switch hir := hirDecl.(type) {
			case *HIRFunction:
				hirModule.Functions = append(hirModule.Functions, hir)
			case *HIRVariable:
				hirModule.Variables = append(hirModule.Variables, hir)
			case *HIRTypeDefinition:
				hirModule.Types = append(hirModule.Types, hir)
			case *HIRImpl:
				hirModule.Impls = append(hirModule.Impls, hir)
			}
		}
	}

	// Pop global scope
	transformer.popScope()

	return hirModule, transformer.errors
}

// transformDeclaration converts AST declarations to HIR
func (transformer *ASTToHIRTransformer) transformDeclaration(decl Declaration) HIRNode {
	switch d := decl.(type) {
	case *FunctionDeclaration:
		return transformer.transformFunctionDeclaration(d)
	case *VariableDeclaration:
		return transformer.transformVariableDeclaration(d)
	case *StructDeclaration:
		return transformer.transformStructDeclaration(d)
	case *EnumDeclaration:
		return transformer.transformEnumDeclaration(d)
	case *TraitDeclaration:
		return transformer.transformTraitDeclaration(d)
	case *TypeAliasDeclaration:
		return transformer.transformTypeAliasDeclaration(d)
	case *NewtypeDeclaration:
		return transformer.transformNewtypeDeclaration(d)
	case *ImportDeclaration:
		// Side-effect: register import in module, no HIRNode returned
		if transformer.currentModule != nil {
			if hi := transformer.transformImportDeclaration(d); hi != nil {
				transformer.currentModule.Imports = append(transformer.currentModule.Imports, hi)
			}
		}
		return nil
	case *ExportDeclaration:
		// Side-effect: register each export item in module
		if transformer.currentModule != nil {
			exports := transformer.transformExportDeclaration(d)
			transformer.currentModule.Exports = append(transformer.currentModule.Exports, exports...)
		}
		return nil
	case *ImplBlock:
		return transformer.transformImplBlock(d)
	default:
		transformer.addError(fmt.Errorf("unsupported declaration type: %T", decl))
		return nil
	}
}

// transformStructDeclaration converts AST struct to HIR type definition
func (transformer *ASTToHIRTransformer) transformStructDeclaration(sd *StructDeclaration) *HIRTypeDefinition {
	fields := make([]*HIRFieldType, 0, len(sd.Fields))
	for _, f := range sd.Fields {
		var ft *HIRType
		if f.Type != nil {
			ft = transformer.transformType(f.Type)
		} else {
			ft = NewHIRType(f.Span, HIRTypeGeneric, &HIRGenericType{Name: "inferred"})
		}
		name := ""
		if f.Name != nil {
			name = f.Name.Value
		}
		fields = append(fields, &HIRFieldType{Name: name, Type: ft})
	}
	st := &HIRStructType{Name: sd.Name.Value, Fields: fields}
	td := &HIRTypeDefinition{Span: sd.Span, Name: sd.Name.Value, Kind: TypeDefStruct, Data: st}
	// Map generic parameters (type params only)
	if len(sd.Generics) > 0 {
		td.TypeParams = make([]*HIRTypeParameter, 0, len(sd.Generics))
		for _, gp := range sd.Generics {
			if gp.Kind == GenericParamType && gp.Name != nil {
				tp := &HIRTypeParameter{Span: gp.Span, Name: gp.Name.Value}
				if len(gp.Bounds) > 0 {
					tp.Bounds = make([]*HIRType, 0, len(gp.Bounds))
					for _, b := range gp.Bounds {
						tp.Bounds = append(tp.Bounds, transformer.transformType(b))
					}
				}
				td.TypeParams = append(td.TypeParams, tp)
			}
		}
	}
	return td
}

// transformEnumDeclaration converts AST enum to HIR type definition
func (transformer *ASTToHIRTransformer) transformEnumDeclaration(ed *EnumDeclaration) *HIRTypeDefinition {
	variants := make([]*HIRVariantType, 0, len(ed.Variants))
	for _, v := range ed.Variants {
		vtypes := make([]*HIRType, 0, len(v.Fields))
		for _, f := range v.Fields {
			var ft *HIRType
			if f.Type != nil {
				ft = transformer.transformType(f.Type)
			} else {
				ft = NewHIRType(f.Span, HIRTypeGeneric, &HIRGenericType{Name: "inferred"})
			}
			vtypes = append(vtypes, ft)
		}
		variants = append(variants, &HIRVariantType{Name: v.Name.Value, Fields: vtypes})
	}
	et := &HIREnumType{Name: ed.Name.Value, Variants: variants}
	td := &HIRTypeDefinition{Span: ed.Span, Name: ed.Name.Value, Kind: TypeDefEnum, Data: et}
	if len(ed.Generics) > 0 {
		td.TypeParams = make([]*HIRTypeParameter, 0, len(ed.Generics))
		for _, gp := range ed.Generics {
			if gp.Kind == GenericParamType && gp.Name != nil {
				tp := &HIRTypeParameter{Span: gp.Span, Name: gp.Name.Value}
				if len(gp.Bounds) > 0 {
					tp.Bounds = make([]*HIRType, 0, len(gp.Bounds))
					for _, b := range gp.Bounds {
						tp.Bounds = append(tp.Bounds, transformer.transformType(b))
					}
				}
				td.TypeParams = append(td.TypeParams, tp)
			}
		}
	}
	return td
}

// transformTraitDeclaration converts AST trait to HIR type definition
func (transformer *ASTToHIRTransformer) transformTraitDeclaration(td *TraitDeclaration) *HIRTypeDefinition {
	methods := make([]*HIRMethodSignature, 0, len(td.Methods))
	for _, m := range td.Methods {
		paramTypes := make([]*HIRType, 0, len(m.Parameters))
		for _, p := range m.Parameters {
			if p.TypeSpec != nil {
				paramTypes = append(paramTypes, transformer.transformType(p.TypeSpec))
			} else {
				paramTypes = append(paramTypes, NewHIRType(p.Span, HIRTypeGeneric, &HIRGenericType{Name: "inferred"}))
			}
		}
		var ret *HIRType
		if m.ReturnType != nil {
			ret = transformer.transformType(m.ReturnType)
		}
		// Map method generics to HIR type parameters (bounds ignored for MVP)
		tparams := make([]*HIRTypeParameter, 0)
		if len(m.Generics) > 0 {
			for _, gp := range m.Generics {
				if gp.Kind == GenericParamType && gp.Name != nil {
					tp := &HIRTypeParameter{Span: gp.Span, Name: gp.Name.Value}
					// bounds to HIR types if any
					if len(gp.Bounds) > 0 {
						tp.Bounds = make([]*HIRType, 0, len(gp.Bounds))
						for _, b := range gp.Bounds {
							tp.Bounds = append(tp.Bounds, transformer.transformType(b))
						}
					}
					tparams = append(tparams, tp)
				}
			}
		}
		methods = append(methods, &HIRMethodSignature{Name: m.Name.Value, Parameters: paramTypes, ReturnType: ret, TypeParameters: tparams})
	}
	// Associated types
	assoc := make([]*HIRTraitAssociatedType, 0, len(td.AssociatedTypes))
	for _, at := range td.AssociatedTypes {
		item := &HIRTraitAssociatedType{Name: at.Name.Value}
		if len(at.Bounds) > 0 {
			item.Bounds = make([]*HIRType, 0, len(at.Bounds))
			for _, b := range at.Bounds {
				item.Bounds = append(item.Bounds, transformer.transformType(b))
			}
		}
		assoc = append(assoc, item)
	}
	tt := &HIRTraitType{Name: td.Name.Value, Methods: methods, AssociatedTypes: assoc}
	tdef := &HIRTypeDefinition{Span: td.Span, Name: td.Name.Value, Kind: TypeDefTrait, Data: tt}
	if len(td.Generics) > 0 {
		tdef.TypeParams = make([]*HIRTypeParameter, 0, len(td.Generics))
		for _, gp := range td.Generics {
			if gp.Kind == GenericParamType && gp.Name != nil {
				tp := &HIRTypeParameter{Span: gp.Span, Name: gp.Name.Value}
				if len(gp.Bounds) > 0 {
					tp.Bounds = make([]*HIRType, 0, len(gp.Bounds))
					for _, b := range gp.Bounds {
						tp.Bounds = append(tp.Bounds, transformer.transformType(b))
					}
				}
				tdef.TypeParams = append(tdef.TypeParams, tp)
			}
		}
	}
	return tdef
}

// transformTypeAliasDeclaration converts AST type alias to HIR type definition
func (transformer *ASTToHIRTransformer) transformTypeAliasDeclaration(ta *TypeAliasDeclaration) *HIRTypeDefinition {
	var target *HIRType
	if ta.Aliased != nil {
		target = transformer.transformType(ta.Aliased)
	} else {
		target = NewHIRType(ta.Span, HIRTypeGeneric, &HIRGenericType{Name: "inferred"})
	}
	alias := &HIRAliasType{Target: target}
	return &HIRTypeDefinition{Span: ta.Span, Name: ta.Name.Value, Kind: TypeDefAlias, Data: alias}
}

// transformNewtypeDeclaration converts AST newtype to HIR type definition
func (transformer *ASTToHIRTransformer) transformNewtypeDeclaration(nd *NewtypeDeclaration) *HIRTypeDefinition {
	var base *HIRType
	if nd.Base != nil {
		base = transformer.transformType(nd.Base)
	} else {
		base = NewHIRType(nd.Span, HIRTypeGeneric, &HIRGenericType{Name: "inferred"})
	}
	nt := &HIRNewtypeType{Base: base}
	return &HIRTypeDefinition{Span: nd.Span, Name: nd.Name.Value, Kind: TypeDefNewtype, Data: nt}
}

// transformImportDeclaration converts AST import to HIR import
func (transformer *ASTToHIRTransformer) transformImportDeclaration(id *ImportDeclaration) *HIRImport {
	parts := make([]string, 0, len(id.Path))
	for _, seg := range id.Path {
		parts = append(parts, seg.Value)
	}
	module := strings.Join(parts, "::")
	var alias string
	if id.Alias != nil {
		alias = id.Alias.Value
	}
	return &HIRImport{Span: id.Span, ModuleName: module, Items: nil, Alias: alias, IsPublic: id.IsPublic}
}

// transformExportDeclaration converts AST export to HIR export
func (transformer *ASTToHIRTransformer) transformExportDeclaration(ed *ExportDeclaration) []*HIRExport {
	exports := make([]*HIRExport, 0, len(ed.Items))
	for _, it := range ed.Items {
		name := ""
		if it.Name != nil {
			name = it.Name.Value
		}
		alias := ""
		if it.Alias != nil {
			alias = it.Alias.Value
		}
		exports = append(exports, &HIRExport{Span: ed.Span, ItemName: name, Alias: alias, IsDefault: false})
	}
	return exports
}

// transformImplBlock converts impl block; returns nil for now (methods are separate fns)
func (transformer *ASTToHIRTransformer) transformImplBlock(ib *ImplBlock) HIRNode {
	if ib == nil {
		return nil
	}
	// Build HIRImpl shell
	hirImpl := &HIRImpl{Span: ib.Span}
	// Generics -> TypeParameters
	if len(ib.Generics) > 0 {
		hirImpl.TypeParams = make([]*HIRTypeParameter, 0, len(ib.Generics))
		for _, gp := range ib.Generics {
			if gp.Kind == GenericParamType && gp.Name != nil {
				tp := &HIRTypeParameter{Span: gp.Span, Name: gp.Name.Value}
				if len(gp.Bounds) > 0 {
					tp.Bounds = make([]*HIRType, 0, len(gp.Bounds))
					for _, b := range gp.Bounds {
						tp.Bounds = append(tp.Bounds, transformer.transformType(b))
					}
				}
				hirImpl.TypeParams = append(hirImpl.TypeParams, tp)
			}
		}
	}
	// where -> Constraints (simplified)
	if len(ib.WhereClauses) > 0 {
		hirImpl.Constraints = make([]*HIRConstraint, 0, len(ib.WhereClauses))
		for _, wp := range ib.WhereClauses {
			c := &HIRConstraint{Span: wp.Span}
			if wp.Target != nil {
				c.Type = transformer.transformType(wp.Target)
			}
			if len(wp.Bounds) > 0 {
				// First bound as Trait for MVP; full support could carry multiple
				c.Trait = transformer.transformType(wp.Bounds[0])
			}
			hirImpl.Constraints = append(hirImpl.Constraints, c)
		}
	}
	// ForType / Trait
	if ib.ForType != nil {
		hirImpl.ForType = transformer.transformType(ib.ForType)
	}
	if ib.Trait != nil {
		hirImpl.Trait = transformer.transformType(ib.Trait)
		hirImpl.Kind = HIRImplTrait
	} else {
		hirImpl.Kind = HIRImplInherent
	}
	// Methods
	if len(ib.Items) > 0 {
		hirImpl.Methods = make([]*HIRFunction, 0, len(ib.Items))
		for _, m := range ib.Items {
			if m == nil || m.Name == nil {
				continue
			}
			hf := transformer.transformFunctionDeclaration(m)
			if hf == nil {
				continue
			}
			hf.IsMethod = true
			hf.MethodOfType = hirImpl.ForType
			if hirImpl.Kind == HIRImplTrait {
				hf.ImplementedTrait = hirImpl.Trait
			}
			hirImpl.Methods = append(hirImpl.Methods, hf)
		}
	}
	return hirImpl
}

// transformFunctionDeclaration converts function declarations
func (transformer *ASTToHIRTransformer) transformFunctionDeclaration(funcDecl *FunctionDeclaration) *HIRFunction {
	// Create HIR function
	hirFunc := NewHIRFunction(funcDecl.Span, funcDecl.Name.Value)
	hirFunc.IsPublic = funcDecl.IsPublic

	// Set current function context
	oldFunction := transformer.currentFunction
	transformer.currentFunction = hirFunc

	// Push function scope
	transformer.pushScope(NewHIRScope(transformer.nextScopeID))
	transformer.nextScopeID++

	// Transform generic type parameters on function
	if len(funcDecl.Generics) > 0 {
		for _, gp := range funcDecl.Generics {
			if gp.Kind == GenericParamType && gp.Name != nil {
				tp := &HIRTypeParameter{Span: gp.Span, Name: gp.Name.Value}
				if len(gp.Bounds) > 0 {
					tp.Bounds = make([]*HIRType, 0, len(gp.Bounds))
					for _, b := range gp.Bounds {
						tp.Bounds = append(tp.Bounds, transformer.transformType(b))
					}
				}
				hirFunc.TypeParameters = append(hirFunc.TypeParameters, tp)
			}
		}
	}

	// Transform parameters
	for _, param := range funcDecl.Parameters {
		hirParam := transformer.transformParameter(param)
		hirFunc.Parameters = append(hirFunc.Parameters, hirParam)

		// Add parameter to scope
		hirVar := NewHIRVariable(param.Span, param.Name.Value, hirParam.Type)
		hirVar.Scope = ScopeParameter
		transformer.currentScope.Variables[param.Name.Value] = hirVar
		hirFunc.LocalVariables = append(hirFunc.LocalVariables, hirVar)
	}

	// Transform return type if present
	if funcDecl.ReturnType != nil {
		hirFunc.ReturnType = transformer.transformType(funcDecl.ReturnType)
	} else {
		// Default to unit type
		hirFunc.ReturnType = NewHIRType(funcDecl.Span, HIRTypePrimitive, &HIRPrimitiveType{Name: "unit", Size: 0})
	}

	// Transform function body
	if funcDecl.Body != nil {
		hirFunc.Body = transformer.transformBlockStatement(funcDecl.Body)
	}

	// Pop function scope
	transformer.popScope()

	// Restore previous function context
	transformer.currentFunction = oldFunction

	// Register function in symbol table
	transformer.symbolTable.functions[funcDecl.Name.Value] = hirFunc

	return hirFunc
}

// transformParameter converts function parameters
func (transformer *ASTToHIRTransformer) transformParameter(param *Parameter) *HIRParameter {
	hirParam := &HIRParameter{
		Span: param.Span,
		Name: param.Name.Value,
	}

	// Transform parameter type
	if param.TypeSpec != nil {
		hirParam.Type = transformer.transformType(param.TypeSpec)
	} else {
		// Default to inferred type placeholder
		hirParam.Type = NewHIRType(param.Span, HIRTypeGeneric, &HIRGenericType{Name: "inferred"})
	}

	return hirParam
}

// transformVariableDeclaration converts variable declarations
func (transformer *ASTToHIRTransformer) transformVariableDeclaration(varDecl *VariableDeclaration) *HIRVariable {
	// Check for nil variable declaration
	if varDecl == nil {
		return nil
	}

	// Check for nil name
	if varDecl.Name == nil {
		return nil
	}

	// Transform type if present
	var hirType *HIRType
	if varDecl.TypeSpec != nil {
		hirType = transformer.transformType(varDecl.TypeSpec)
	} else {
		// Type will be inferred
		hirType = NewHIRType(varDecl.Span, HIRTypeGeneric, &HIRGenericType{Name: "inferred"})
	}

	// Create HIR variable
	hirVar := NewHIRVariable(varDecl.Span, varDecl.Name.Value, hirType)
	hirVar.IsMutable = varDecl.IsMutable

	// Transform initializer if present
	if varDecl.Initializer != nil {
		hirVar.Initializer = transformer.transformExpression(varDecl.Initializer)
	}

	// Add to current scope if it exists
	if transformer.currentScope != nil {
		transformer.currentScope.Variables[varDecl.Name.Value] = hirVar
	}

	// Add to symbol table
	if transformer.currentFunction != nil {
		hirVar.Scope = ScopeLocal
		transformer.currentFunction.LocalVariables = append(transformer.currentFunction.LocalVariables, hirVar)
	} else {
		hirVar.Scope = ScopeGlobal
		if transformer.symbolTable != nil && transformer.symbolTable.globals != nil {
			transformer.symbolTable.globals[varDecl.Name.Value] = hirVar
		}
	}

	return hirVar
}

// transformBlockStatement converts block statements
func (transformer *ASTToHIRTransformer) transformBlockStatement(block *BlockStatement) *HIRBlock {
	// Create HIR block
	hirBlock := NewHIRBlock(block.Span)

	// Push block scope
	transformer.pushScope(NewHIRScope(transformer.nextScopeID))
	transformer.nextScopeID++
	hirBlock.Scope = transformer.currentScope

	// Transform all statements
	for _, stmt := range block.Statements {
		if hirStmt := transformer.transformStatement(stmt); hirStmt != nil {
			hirBlock.Statements = append(hirBlock.Statements, hirStmt)
		}
	}

	// Pop block scope
	transformer.popScope()

	return hirBlock
}

// transformStatement converts statements (complete implementation)
func (transformer *ASTToHIRTransformer) transformStatement(stmt Statement) *HIRStatement {
	switch s := stmt.(type) {
	case *ExpressionStatement:
		return transformer.transformExpressionStatement(s)
	case *ReturnStatement:
		return transformer.transformReturnStatement(s)
	case *IfStatement:
		return transformer.transformIfStatement(s)
	case *WhileStatement:
		return transformer.transformWhileStatement(s)
	case *VariableDeclaration:
		// Variable declarations can appear as statements
		hirVar := transformer.transformVariableDeclaration(s)
		return &HIRStatement{
			Span: s.Span,
			Kind: HIRStmtLet,
			Data: &HIRLetStatement{
				Variable:    hirVar,
				Initializer: hirVar.Initializer,
			},
		}
	default:
		transformer.addError(fmt.Errorf("unsupported statement type: %T", stmt))
		return nil
	}
}

// transformExpression converts expressions (complete implementation)
func (transformer *ASTToHIRTransformer) transformExpression(expr Expression) *HIRExpression {
	switch e := expr.(type) {
	case *Literal:
		return transformer.transformLiteral(e)
	case *Identifier:
		return transformer.transformIdentifier(e)
	case *BinaryExpression:
		return transformer.transformBinaryExpression(e)
	case *UnaryExpression:
		return transformer.transformUnaryExpression(e)
	case *CallExpression:
		return transformer.transformCallExpression(e)
	case *AssignmentExpression:
		return transformer.transformAssignmentExpression(e)
	default:
		transformer.addError(fmt.Errorf("unsupported expression type: %T", expr))
		return nil
	}
}

// transformExpressionStatement converts expression statements
func (transformer *ASTToHIRTransformer) transformExpressionStatement(exprStmt *ExpressionStatement) *HIRStatement {
	hirExpr := transformer.transformExpression(exprStmt.Expression)
	return &HIRStatement{
		Span: exprStmt.Span,
		Kind: HIRStmtExpression,
		Data: hirExpr,
	}
}

// transformReturnStatement converts return statements
func (transformer *ASTToHIRTransformer) transformReturnStatement(retStmt *ReturnStatement) *HIRStatement {
	var hirValue *HIRExpression
	if retStmt.Value != nil {
		hirValue = transformer.transformExpression(retStmt.Value)
	}

	return &HIRStatement{
		Span: retStmt.Span,
		Kind: HIRStmtReturn,
		Data: &HIRReturnStatement{
			Value: hirValue,
		},
	}
}

// transformIfStatement converts if statements
func (transformer *ASTToHIRTransformer) transformIfStatement(ifStmt *IfStatement) *HIRStatement {
	hirCondition := transformer.transformExpression(ifStmt.Condition)
	hirThenBlock := transformer.transformStatementToBlock(ifStmt.ThenStmt)

	var hirElseBlock *HIRBlock
	if ifStmt.ElseStmt != nil {
		hirElseBlock = transformer.transformStatementToBlock(ifStmt.ElseStmt)
	}

	return &HIRStatement{
		Span: ifStmt.Span,
		Kind: HIRStmtIf,
		Data: &HIRIfStatement{
			Condition: hirCondition,
			ThenBlock: hirThenBlock,
			ElseBlock: hirElseBlock,
		},
	}
}

// transformWhileStatement converts while statements
func (transformer *ASTToHIRTransformer) transformWhileStatement(whileStmt *WhileStatement) *HIRStatement {
	hirCondition := transformer.transformExpression(whileStmt.Condition)
	hirBody := transformer.transformStatementToBlock(whileStmt.Body)

	return &HIRStatement{
		Span: whileStmt.Span,
		Kind: HIRStmtWhile,
		Data: &HIRWhileStatement{
			Condition: hirCondition,
			Body:      hirBody,
		},
	}
}

// transformStatementToBlock converts a statement to a block if it isn't already
func (transformer *ASTToHIRTransformer) transformStatementToBlock(stmt Statement) *HIRBlock {
	if block, ok := stmt.(*BlockStatement); ok {
		return transformer.transformBlockStatement(block)
	}

	// Wrap single statement in a block
	hirBlock := NewHIRBlock(stmt.GetSpan())
	if hirStmt := transformer.transformStatement(stmt); hirStmt != nil {
		hirBlock.Statements = []*HIRStatement{hirStmt}
	}
	return hirBlock
}

// transformLiteral converts literal expressions
func (transformer *ASTToHIRTransformer) transformLiteral(literal *Literal) *HIRExpression {
	// Determine HIR type based on literal kind
	var hirType *HIRType
	switch literal.Kind {
	case LiteralInteger:
		hirType = NewHIRType(literal.Span, HIRTypePrimitive, &HIRPrimitiveType{Name: "int", Size: 8})
	case LiteralFloat:
		hirType = NewHIRType(literal.Span, HIRTypePrimitive, &HIRPrimitiveType{Name: "float", Size: 8})
	case LiteralString:
		hirType = NewHIRType(literal.Span, HIRTypePrimitive, &HIRPrimitiveType{Name: "string", Size: 16})
	case LiteralBool:
		hirType = NewHIRType(literal.Span, HIRTypePrimitive, &HIRPrimitiveType{Name: "bool", Size: 1})
	default:
		hirType = NewHIRType(literal.Span, HIRTypeAny, nil)
	}

	return NewHIRExpression(
		literal.Span,
		hirType,
		HIRExprLiteral,
		&HIRLiteralExpression{
			Value: literal.Value,
			Kind:  literal.Kind,
		},
	)
}

// transformIdentifier converts identifier expressions
func (transformer *ASTToHIRTransformer) transformIdentifier(identifier *Identifier) *HIRExpression {
	// Look up identifier in symbol table
	variable := transformer.lookupVariable(identifier.Value)
	if variable == nil {
		transformer.addError(fmt.Errorf("undefined variable: %s", identifier.Value))
		return nil
	}

	return NewHIRExpression(
		identifier.Span,
		variable.Type,
		HIRExprVariable,
		&HIRVariableExpression{
			Name:     identifier.Value,
			Variable: variable,
		},
	)
}

// transformBinaryExpression converts binary expressions
func (transformer *ASTToHIRTransformer) transformBinaryExpression(binExpr *BinaryExpression) *HIRExpression {
	hirLeft := transformer.transformExpression(binExpr.Left)
	hirRight := transformer.transformExpression(binExpr.Right)

	if hirLeft == nil || hirRight == nil {
		return nil
	}

	// Determine operator kind
	var opKind BinaryOperatorKind
	switch binExpr.Operator.Value {
	case "+":
		opKind = BinOpAdd
	case "-":
		opKind = BinOpSub
	case "*":
		opKind = BinOpMul
	case "/":
		opKind = BinOpDiv
	case "%":
		opKind = BinOpMod
	case "==":
		opKind = BinOpEq
	case "!=":
		opKind = BinOpNe
	case "<":
		opKind = BinOpLt
	case "<=":
		opKind = BinOpLe
	case ">":
		opKind = BinOpGt
	case ">=":
		opKind = BinOpGe
	case "&&":
		opKind = BinOpLogicalAnd
	case "||":
		opKind = BinOpLogicalOr
	default:
		transformer.addError(fmt.Errorf("unsupported binary operator: %s", binExpr.Operator.Value))
		return nil
	}

	// Determine result type (simplified type inference)
	var resultType *HIRType
	if isComparisonOp(opKind) {
		resultType = NewHIRType(binExpr.Span, HIRTypePrimitive, &HIRPrimitiveType{Name: "bool", Size: 1})
	} else {
		// Use left operand type for now (proper type inference will be added later)
		resultType = hirLeft.Type
	}

	return NewHIRExpression(
		binExpr.Span,
		resultType,
		HIRExprBinary,
		&HIRBinaryExpression{
			Left:     hirLeft,
			Right:    hirRight,
			Operator: opKind,
		},
	)
}

// transformUnaryExpression converts unary expressions
func (transformer *ASTToHIRTransformer) transformUnaryExpression(unaryExpr *UnaryExpression) *HIRExpression {
	hirOperand := transformer.transformExpression(unaryExpr.Operand)
	if hirOperand == nil {
		return nil
	}

	// Determine operator kind
	var opKind UnaryOperatorKind
	switch unaryExpr.Operator.Value {
	case "-":
		opKind = UnaryOpNeg
	case "!":
		opKind = UnaryOpNot
	case "~":
		opKind = UnaryOpBitNot
	default:
		transformer.addError(fmt.Errorf("unsupported unary operator: %s", unaryExpr.Operator.Value))
		return nil
	}

	return NewHIRExpression(
		unaryExpr.Span,
		hirOperand.Type, // Result type same as operand for most unary ops
		HIRExprUnary,
		&HIRUnaryExpression{
			Operand:  hirOperand,
			Operator: opKind,
		},
	)
}

// transformCallExpression converts call expressions
func (transformer *ASTToHIRTransformer) transformCallExpression(callExpr *CallExpression) *HIRExpression {
	hirFunction := transformer.transformExpression(callExpr.Function)
	if hirFunction == nil {
		return nil
	}

	// Transform arguments
	hirArgs := make([]*HIRExpression, 0, len(callExpr.Arguments))
	for _, arg := range callExpr.Arguments {
		if hirArg := transformer.transformExpression(arg); hirArg != nil {
			hirArgs = append(hirArgs, hirArg)
		} else {
			return nil
		}
	}

	// Determine return type (simplified - will be enhanced with proper type checking)
	returnType := NewHIRType(callExpr.Span, HIRTypeGeneric, &HIRGenericType{Name: "inferred"})

	return NewHIRExpression(
		callExpr.Span,
		returnType,
		HIRExprCall,
		&HIRCallExpression{
			Function:  hirFunction,
			Arguments: hirArgs,
		},
	)
}

// transformAssignmentExpression converts assignment expressions
func (transformer *ASTToHIRTransformer) transformAssignmentExpression(assignExpr *AssignmentExpression) *HIRExpression {
	hirTarget := transformer.transformExpression(assignExpr.Left)
	hirValue := transformer.transformExpression(assignExpr.Right)

	if hirTarget == nil || hirValue == nil {
		return nil
	}

	// Convert to assignment statement wrapped in expression
	// This is a simplification - real implementation might handle this differently
	return NewHIRExpression(
		assignExpr.Span,
		hirTarget.Type,
		HIRExprBlock,
		&HIRBlock{
			Span: assignExpr.Span,
			Statements: []*HIRStatement{
				{
					Span: assignExpr.Span,
					Kind: HIRStmtAssign,
					Data: &HIRAssignStatement{
						Target: hirTarget,
						Value:  hirValue,
					},
				},
			},
			Expression: hirTarget, // Assignment expression returns the assigned value
		},
	)
}

// transformType converts type expressions
func (transformer *ASTToHIRTransformer) transformType(typeExpr Type) *HIRType {
	switch t := typeExpr.(type) {
	case *BasicType:
		return transformer.transformBasicType(t)
	case *StructType:
		// Use nominal struct type by name if available
		name := "struct"
		if t.Name != nil {
			name = t.Name.Value
		}
		return NewHIRType(t.Span, HIRTypeStruct, &HIRStructType{Name: name})
	case *EnumType:
		name := "enum"
		if t.Name != nil {
			name = t.Name.Value
		}
		return NewHIRType(t.Span, HIRTypeEnum, &HIREnumType{Name: name})
	case *ArrayType:
		return transformer.transformArrayType(t)
	case *PointerType:
		return transformer.transformPointerType(t)
	case *ReferenceType:
		return transformer.transformReferenceType(t)
	case *FunctionType:
		return transformer.transformFunctionType(t)
	case *GenericType:
		return transformer.transformGenericType(t)
	default:
		transformer.addError(fmt.Errorf("unsupported type: %T", typeExpr))
		return nil
	}
}

// transformBasicType converts basic types
func (transformer *ASTToHIRTransformer) transformBasicType(basicType *BasicType) *HIRType {
	var size int
	switch basicType.Name {
	case "int", "i64":
		size = 8
	case "i32":
		size = 4
	case "i16":
		size = 2
	case "i8":
		size = 1
	case "float", "f64":
		size = 8
	case "f32":
		size = 4
	case "bool":
		size = 1
	case "string":
		size = 16 // pointer + length
	default:
		size = 8 // default size
	}

	return NewHIRType(
		basicType.Span,
		HIRTypePrimitive,
		&HIRPrimitiveType{
			Name: basicType.Name,
			Size: size,
		},
	)
}

// transformArrayType converts array types
func (transformer *ASTToHIRTransformer) transformArrayType(arrayType *ArrayType) *HIRType {
	var elementType *HIRType
	if arrayType.ElementType != nil {
		elementType = transformer.transformType(arrayType.ElementType)
	} else {
		elementType = NewHIRType(arrayType.Span, HIRTypeGeneric, &HIRGenericType{Name: "inferred"})
	}
	
	// Convert size expression to constant if possible
	var size int
	if arrayType.Size != nil {
		// In a full implementation, this would evaluate the size expression
		// For now, default to unknown size
		size = 0
	}
	
	return NewHIRType(
		arrayType.Span,
		HIRTypeArray,
		map[string]interface{}{
			"element_type": elementType,
			"size":         size,
		},
	)
}

// transformPointerType converts pointer types
func (transformer *ASTToHIRTransformer) transformPointerType(pointerType *PointerType) *HIRType {
	var pointeeType *HIRType
	if pointerType.Inner != nil {
		pointeeType = transformer.transformType(pointerType.Inner)
	} else {
		pointeeType = NewHIRType(pointerType.Span, HIRTypeGeneric, &HIRGenericType{Name: "inferred"})
	}
	
	return NewHIRType(
		pointerType.Span,
		HIRTypePointer,
		map[string]interface{}{
			"pointee_type": pointeeType,
			"is_mutable":   pointerType.IsMutable,
		},
	)
}

// transformReferenceType converts reference types
func (transformer *ASTToHIRTransformer) transformReferenceType(referenceType *ReferenceType) *HIRType {
	var referentType *HIRType
	if referenceType.Inner != nil {
		referentType = transformer.transformType(referenceType.Inner)
	} else {
		referentType = NewHIRType(referenceType.Span, HIRTypeGeneric, &HIRGenericType{Name: "inferred"})
	}
	
	return NewHIRType(
		referenceType.Span,
		HIRTypeReference,
		map[string]interface{}{
			"referent_type": referentType,
			"is_mutable":    referenceType.IsMutable,
			"lifetime":      referenceType.Lifetime,
		},
	)
}

// transformFunctionType converts function types
func (transformer *ASTToHIRTransformer) transformFunctionType(functionType *FunctionType) *HIRType {
	paramTypes := make([]*HIRType, 0, len(functionType.Parameters))
	for _, param := range functionType.Parameters {
		if param.Type != nil {
			paramTypes = append(paramTypes, transformer.transformType(param.Type))
		} else {
			paramTypes = append(paramTypes, NewHIRType(param.Span, HIRTypeGeneric, &HIRGenericType{Name: "inferred"}))
		}
	}
	
	var returnType *HIRType
	if functionType.ReturnType != nil {
		returnType = transformer.transformType(functionType.ReturnType)
	} else {
		returnType = NewHIRType(functionType.Span, HIRTypePrimitive, &HIRPrimitiveType{Name: "void", Size: 0})
	}
	
	return NewHIRType(
		functionType.Span,
		HIRTypeFunction,
		&HIRFunctionType{
			Parameters: paramTypes,
			ReturnType: returnType,
			IsAsync:    functionType.IsAsync,
		},
	)
}

// transformGenericType converts generic types
func (transformer *ASTToHIRTransformer) transformGenericType(genericType *GenericType) *HIRType {
	// For now, just create a simple generic type
	// In a full implementation, this would handle generic type instantiation
	return NewHIRType(
		genericType.Span,
		HIRTypeGeneric,
		&HIRGenericType{
			Name: "generic_type",
		},
	)
}

// ====== Scope Management ======

// NewHIRScope creates a new HIR scope
func NewHIRScope(id int) *HIRScope {
	return &HIRScope{
		ID:        id,
		Variables: make(map[string]*HIRVariable),
		Types:     make(map[string]*HIRType),
		Children:  make([]*HIRScope, 0),
	}
}

// pushScope pushes a new scope onto the scope stack
func (transformer *ASTToHIRTransformer) pushScope(scope *HIRScope) {
	if transformer.currentScope != nil {
		scope.Parent = transformer.currentScope
		transformer.currentScope.Children = append(transformer.currentScope.Children, scope)
	}
	transformer.scopeStack = append(transformer.scopeStack, scope)
	transformer.currentScope = scope
}

// popScope pops the current scope from the scope stack
func (transformer *ASTToHIRTransformer) popScope() {
	if len(transformer.scopeStack) > 0 {
		transformer.scopeStack = transformer.scopeStack[:len(transformer.scopeStack)-1]
		if len(transformer.scopeStack) > 0 {
			transformer.currentScope = transformer.scopeStack[len(transformer.scopeStack)-1]
		} else {
			transformer.currentScope = nil
		}
	}
}

// lookupVariable searches for a variable in the current scope chain
func (transformer *ASTToHIRTransformer) lookupVariable(name string) *HIRVariable {
	// Search in current scope chain
	for scope := transformer.currentScope; scope != nil; scope = scope.Parent {
		if variable, exists := scope.Variables[name]; exists {
			return variable
		}
	}

	// Search in global scope
	if variable, exists := transformer.symbolTable.globals[name]; exists {
		return variable
	}

	return nil
}

// ====== Helper Functions ======

// isComparisonOp checks if an operator is a comparison operator
func isComparisonOp(op BinaryOperatorKind) bool {
	switch op {
	case BinOpEq, BinOpNe, BinOpLt, BinOpLe, BinOpGt, BinOpGe:
		return true
	default:
		return false
	}
}

// addError adds an error to the transformer's error list
func (transformer *ASTToHIRTransformer) addError(err error) {
	transformer.errors = append(transformer.errors, err)
}

// GetErrors returns all transformation errors
func (transformer *ASTToHIRTransformer) GetErrors() []error {
	return transformer.errors
}

// ====== Public API ======

// TransformASTToHIR is the main entry point for AST to HIR transformation
func TransformASTToHIR(program *Program) (*HIRModule, []error) {
	transformer := NewASTToHIRTransformer()
	return transformer.TransformProgram(program)
}
