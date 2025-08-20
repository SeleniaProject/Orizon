// AST to HIR converter for the Orizon programming language.
// This file implements the transformation from AST to HIR with semantic enrichment.

package hir

import (
	"fmt"
	"strings"

	"github.com/orizon-lang/orizon/internal/ast"
	"github.com/orizon-lang/orizon/internal/position"
)

// ASTToHIRConverter converts AST nodes to HIR nodes with semantic analysis.
type ASTToHIRConverter struct {
	program     *HIRProgram
	typeBuilder *HIRTypeBuilder
	symbolTable *SymbolTable
	errors      []ConversionError
}

// ConversionError represents an error during AST to HIR conversion.
type ConversionError struct {
	Message string
	Span    position.Span
	Kind    ErrorKind
}

// ErrorKind represents the kind of conversion error.
type ErrorKind int

const (
	ErrorKindTypeError ErrorKind = iota
	ErrorKindNameResolution
	ErrorKindScopeError
	ErrorKindEffectError
	ErrorKindRegionError
)

// SymbolTable manages symbol resolution during conversion.
type SymbolTable struct {
	symbols   map[string]*Symbol
	scopes    []*Scope
	currentID NodeID
}

// Scope represents a lexical scope.
type Scope struct {
	Parent  *Scope
	Symbols map[string]*Symbol
	Level   int
}

// Symbol represents a symbol in the symbol table.
type Symbol struct {
	Name        string
	Type        TypeInfo
	Declaration HIRDeclaration
	Span        position.Span
	Mutable     bool
	Used        bool
}

// NewASTToHIRConverter creates a new AST to HIR converter.
func NewASTToHIRConverter() *ASTToHIRConverter {
	program := NewHIRProgram()

	return &ASTToHIRConverter{
		program:     program,
		typeBuilder: NewHIRTypeBuilder(program),
		symbolTable: NewSymbolTable(),
		errors:      make([]ConversionError, 0),
	}
}

// NewSymbolTable creates a new symbol table.
func NewSymbolTable() *SymbolTable {
	globalScope := &Scope{
		Parent:  nil,
		Symbols: make(map[string]*Symbol),
		Level:   0,
	}

	symbolTable := &SymbolTable{
		scopes:    []*Scope{globalScope},
		currentID: 1,
		symbols:   make(map[string]*Symbol),
	}

	// Add built-in functions to global scope.
	builtinFunctions := []struct {
		name       string
		paramTypes []TypeInfo
		returnType TypeInfo
	}{
		{"print", []TypeInfo{{Kind: TypeKindString, Name: "string"}}, TypeInfo{Kind: TypeKindVoid, Name: "void"}},
		{"println", []TypeInfo{{Kind: TypeKindString, Name: "string"}}, TypeInfo{Kind: TypeKindVoid, Name: "void"}},
		{"exit", []TypeInfo{{Kind: TypeKindInteger, Name: "int"}}, TypeInfo{Kind: TypeKindVoid, Name: "void"}},
	}

	for _, builtin := range builtinFunctions {
		// Include both parameter types and return type in Parameters slice.
		allTypes := make([]TypeInfo, 0, len(builtin.paramTypes)+1)
		allTypes = append(allTypes, builtin.paramTypes...)
		allTypes = append(allTypes, builtin.returnType)

		funcType := TypeInfo{
			Kind:       TypeKindFunction,
			Name:       builtin.name,
			Parameters: allTypes,
		}

		symbol := &Symbol{
			Name:        builtin.name,
			Type:        funcType,
			Declaration: nil, // Built-in, no declaration
			Used:        false,
		}

		globalScope.Symbols[builtin.name] = symbol
		symbolTable.symbols[builtin.name] = symbol
	}

	return symbolTable
}

// ConvertProgram converts an AST program to HIR program.
func (c *ASTToHIRConverter) ConvertProgram(astProgram *ast.Program) (*HIRProgram, []ConversionError) {
	c.program.ID = generateNodeID()
	c.program.Span = astProgram.GetSpan()

	// Create main module.
	mainModule := &HIRModule{
		ID:           generateNodeID(),
		ModuleID:     1,
		Name:         "main",
		Declarations: make([]HIRDeclaration, 0),
		Exports:      make([]string, 0),
		Imports:      make([]ImportInfo, 0),
		Metadata:     IRMetadata{},
		Span:         astProgram.GetSpan(),
	}

	// Convert declarations.
	for _, decl := range astProgram.Declarations {
		hirDecl := c.convertDeclaration(decl)
		if hirDecl != nil {
			mainModule.Declarations = append(mainModule.Declarations, hirDecl)
		}
	}

	c.program.Modules[1] = mainModule

	return c.program, c.errors
}

// convertDeclaration converts an AST declaration to HIR declaration.
func (c *ASTToHIRConverter) convertDeclaration(astDecl ast.Declaration) HIRDeclaration {
	switch decl := astDecl.(type) {
	case *ast.FunctionDeclaration:
		return c.convertFunctionDeclaration(decl)
	case *ast.VariableDeclaration:
		return c.convertVariableDeclaration(decl)
	case *ast.TypeDeclaration:
		return c.convertTypeDeclaration(decl)
	default:
		c.addError(ConversionError{
			Message: fmt.Sprintf("unsupported declaration type: %T", decl),
			Span:    decl.GetSpan(),
			Kind:    ErrorKindTypeError,
		})

		return nil
	}
}

// convertFunctionDeclaration converts an AST function declaration to HIR.
func (c *ASTToHIRConverter) convertFunctionDeclaration(astFunc *ast.FunctionDeclaration) HIRDeclaration {
	// First, add function to symbol table BEFORE processing body.
	// This allows recursive calls and forward references.
	// Convert return type.
	var hirReturnType HIRType
	if astFunc.ReturnType != nil {
		hirReturnType = c.convertType(astFunc.ReturnType)
	} else {
		hirReturnType = c.typeBuilder.BuildBasicType("void", astFunc.GetSpan())
	}

	// Build parameter types for function signature.
	paramTypes := make([]HIRType, len(astFunc.Parameters))
	for i, param := range astFunc.Parameters {
		paramTypes[i] = c.convertType(param.Type)
	}

	// Build function type and add to global symbol table.
	funcType := c.typeBuilder.BuildFunctionType(
		paramTypes,
		hirReturnType,
		NewEffectSet(), // Will be updated after analyzing body
		astFunc.GetSpan(),
	)

	c.symbolTable.AddSymbol(astFunc.Name.Value, &Symbol{
		Name:        astFunc.Name.Value,
		Type:        funcType.GetType(),
		Declaration: nil, // Will be set after creating HIRFunctionDeclaration
		Span:        astFunc.GetSpan(),
		Mutable:     false,
		Used:        false,
	})

	// Enter new scope for function body.
	c.symbolTable.PushScope()
	defer c.symbolTable.PopScope()

	// Convert parameters.
	hirParams := make([]*HIRParameter, len(astFunc.Parameters))

	for i, param := range astFunc.Parameters {
		hirType := c.convertType(param.Type)

		// Convert default value if provided.
		var hirDefault HIRExpression
		if param.DefaultValue != nil {
			hirDefault = c.convertExpression(param.DefaultValue)
			// Type check: default value must be compatible with parameter type.
			if hirDefault != nil {
				defaultType := hirDefault.GetType()
				paramType := hirType.GetType()

				if !c.isTypeCompatible(defaultType, paramType) {
					c.addError(ConversionError{
						Message: fmt.Sprintf("default value type %s is not compatible with parameter type %s",
							defaultType.Name, paramType.Name),
						Span: param.GetSpan(),
						Kind: ErrorKindTypeError,
					})

					hirDefault = nil
				}
			}
		}

		hirParam := &HIRParameter{
			ID:       generateNodeID(),
			Name:     param.Name.Value,
			Type:     hirType,
			Default:  hirDefault,
			Metadata: IRMetadata{},
			Span:     param.GetSpan(),
		}
		hirParams[i] = hirParam

		// Add parameter to local scope.
		c.symbolTable.AddSymbol(param.Name.Value, &Symbol{
			Name:        param.Name.Value,
			Type:        hirType.GetType(),
			Declaration: nil, // Will be set later
			Span:        param.GetSpan(),
			Mutable:     param.IsMutable,
			Used:        false,
		})
	}

	// Convert body.
	var hirBody *HIRBlockStatement
	if astFunc.Body != nil {
		hirBody = c.convertBlockStatement(astFunc.Body)
	}

	// Analyze effects and regions.
	effects := c.analyzeStatementEffects(hirBody)
	regions := c.analyzeStatementRegions(hirBody)

	// Check for generic type parameters.
	typeParams := make([]TypeInfo, 0)
	isGeneric := false

	// In a real implementation, you'd parse the AST for type parameter declarations.
	// For now, detect potential generic patterns in the function name or parameters.
	if c.detectGenericPattern(astFunc) {
		isGeneric = true
		typeParams = c.extractTypeParameters(astFunc)
	}

	hirFunc := &HIRFunctionDeclaration{
		ID:         generateNodeID(),
		Name:       astFunc.Name.Value,
		Parameters: hirParams,
		ReturnType: hirReturnType,
		Body:       hirBody,
		Generic:    isGeneric,
		TypeParams: typeParams,
		Effects:    effects,
		Regions:    regions,
		Metadata:   IRMetadata{},
		Span:       astFunc.GetSpan(),
	}

	// Update the symbol's declaration reference.
	if symbol := c.symbolTable.LookupSymbol(astFunc.Name.Value); symbol != nil {
		symbol.Declaration = hirFunc
	}

	return hirFunc
}

// isTypeCompatible checks if two types are compatible for assignment.
func (c *ASTToHIRConverter) isTypeCompatible(sourceType, targetType TypeInfo) bool {
	// Exact match.
	if sourceType.Kind == targetType.Kind && sourceType.Name == targetType.Name {
		return true
	}

	// Allow implicit conversions for compatible types.
	// For now, be strict and only allow exact matches.
	// TODO: Add more sophisticated type compatibility rules.
	return false
}

// convertVariableDeclaration converts an AST variable declaration to HIR.
func (c *ASTToHIRConverter) convertVariableDeclaration(astVar *ast.VariableDeclaration) HIRDeclaration {
	// Convert type.
	var hirType HIRType
	if astVar.Type != nil {
		hirType = c.convertType(astVar.Type)
	} else {
		// Type inference from initializer.
		if astVar.Value != nil {
			hirInit := c.convertExpression(astVar.Value)
			if hirInit != nil {
				// Create type from initializer.
				initType := hirInit.GetType()
				hirType = c.typeBuilder.BuildBasicType(initType.Name, astVar.GetSpan())
			} else {
				c.addError(ConversionError{
					Message: "cannot infer type for variable without explicit type or valid initializer",
					Span:    astVar.GetSpan(),
					Kind:    ErrorKindTypeError,
				})

				return nil
			}
		} else {
			c.addError(ConversionError{
				Message: "variable declaration must have either type annotation or initializer",
				Span:    astVar.GetSpan(),
				Kind:    ErrorKindTypeError,
			})

			return nil
		}
	}

	// Convert initializer.
	var hirInit HIRExpression
	if astVar.Value != nil {
		hirInit = c.convertExpression(astVar.Value)
	}

	// Analyze effects and regions.
	effects := NewEffectSet()
	regions := NewRegionSet()

	if hirInit != nil {
		effects = hirInit.GetEffects()
		regions = hirInit.GetRegions()
	}

	hirVar := &HIRVariableDeclaration{
		ID:          generateNodeID(),
		Name:        astVar.Name.Value,
		Type:        hirType,
		Initializer: hirInit,
		Mutable:     astVar.IsMutable,
		Effects:     effects,
		Regions:     regions,
		Metadata:    IRMetadata{},
		Span:        astVar.GetSpan(),
	}

	// Add variable to symbol table.
	c.symbolTable.AddSymbol(astVar.Name.Value, &Symbol{
		Name:        astVar.Name.Value,
		Type:        hirType.GetType(),
		Declaration: hirVar,
		Span:        astVar.GetSpan(),
		Mutable:     astVar.IsMutable,
		Used:        false,
	})

	return hirVar
}

// convertTypeDeclaration converts an AST type declaration to HIR.
func (c *ASTToHIRConverter) convertTypeDeclaration(astType *ast.TypeDeclaration) HIRDeclaration {
	hirType := c.convertType(astType.Type)
	if hirType == nil {
		return nil
	}

	hirTypeDecl := &HIRTypeDeclaration{
		ID:       generateNodeID(),
		Name:     astType.Name.Value,
		Type:     hirType,
		Generic:  c.detectTypeGenericPattern(astType),
		Params:   c.extractTypeGenericParameters(astType),
		Metadata: IRMetadata{},
		Span:     astType.GetSpan(),
	}

	// Add type to symbol table.
	c.symbolTable.AddSymbol(astType.Name.Value, &Symbol{
		Name:        astType.Name.Value,
		Type:        hirType.GetType(),
		Declaration: hirTypeDecl,
		Span:        astType.GetSpan(),
		Mutable:     false,
		Used:        false,
	})

	return hirTypeDecl
}

// convertStatement converts an AST statement to HIR statement.
func (c *ASTToHIRConverter) convertStatement(astStmt ast.Statement) HIRStatement {
	switch stmt := astStmt.(type) {
	case *ast.BlockStatement:
		return c.convertBlockStatement(stmt)
	case *ast.ExpressionStatement:
		return c.convertExpressionStatement(stmt)
	case *ast.ReturnStatement:
		return c.convertReturnStatement(stmt)
	case *ast.IfStatement:
		return c.convertIfStatement(stmt)
	case *ast.WhileStatement:
		return c.convertWhileStatement(stmt)
	case *ast.VariableDeclaration:
		// Variable declarations can appear as statements (let statements).
		hirDecl := c.convertVariableDeclaration(stmt)
		if hirDecl == nil {
			return nil
		}
		// Cast the declaration as a statement (it implements both interfaces).
		if hirVar, ok := hirDecl.(*HIRVariableDeclaration); ok {
			return hirVar
		}

		return nil
	default:
		c.addError(ConversionError{
			Message: fmt.Sprintf("unsupported statement type: %T", stmt),
			Span:    stmt.GetSpan(),
			Kind:    ErrorKindTypeError,
		})

		return nil
	}
}

// convertBlockStatement converts an AST block statement to HIR.
func (c *ASTToHIRConverter) convertBlockStatement(astBlock *ast.BlockStatement) *HIRBlockStatement {
	// Enter new scope for block.
	c.symbolTable.PushScope()
	defer c.symbolTable.PopScope()

	hirStmts := make([]HIRStatement, 0, len(astBlock.Statements))
	combinedEffects := NewEffectSet()
	combinedRegions := NewRegionSet()

	for _, astStmt := range astBlock.Statements {
		hirStmt := c.convertStatement(astStmt)
		if hirStmt != nil {
			hirStmts = append(hirStmts, hirStmt)
			combinedEffects = combinedEffects.Union(hirStmt.GetEffects())
			combinedRegions = combinedRegions.Union(hirStmt.GetRegions())
		}
	}

	return &HIRBlockStatement{
		ID:         generateNodeID(),
		Statements: hirStmts,
		Effects:    combinedEffects,
		Regions:    combinedRegions,
		Metadata:   IRMetadata{},
		Span:       astBlock.GetSpan(),
	}
}

// convertExpressionStatement converts an AST expression statement to HIR.
func (c *ASTToHIRConverter) convertExpressionStatement(astExprStmt *ast.ExpressionStatement) HIRStatement {
	hirExpr := c.convertExpression(astExprStmt.Expression)
	if hirExpr == nil {
		return nil
	}

	return &HIRExpressionStatement{
		ID:         generateNodeID(),
		Expression: hirExpr,
		Effects:    hirExpr.GetEffects(),
		Regions:    hirExpr.GetRegions(),
		Metadata:   IRMetadata{},
		Span:       astExprStmt.GetSpan(),
	}
}

// convertReturnStatement converts an AST return statement to HIR.
func (c *ASTToHIRConverter) convertReturnStatement(astReturn *ast.ReturnStatement) HIRStatement {
	var hirExpr HIRExpression

	effects := NewEffectSet()
	regions := NewRegionSet()

	if astReturn.Value != nil {
		hirExpr = c.convertExpression(astReturn.Value)
		if hirExpr != nil {
			effects = hirExpr.GetEffects()
			regions = hirExpr.GetRegions()
		}
	}

	return &HIRReturnStatement{
		ID:         generateNodeID(),
		Expression: hirExpr,
		Effects:    effects,
		Regions:    regions,
		Metadata:   IRMetadata{},
		Span:       astReturn.GetSpan(),
	}
}

// convertIfStatement converts an AST if statement to HIR.
func (c *ASTToHIRConverter) convertIfStatement(astIf *ast.IfStatement) HIRStatement {
	hirCond := c.convertExpression(astIf.Condition)
	if hirCond == nil {
		return nil
	}

	hirThen := c.convertStatement(astIf.ThenBlock)
	if hirThen == nil {
		return nil
	}

	var hirElse HIRStatement
	if astIf.ElseBlock != nil {
		hirElse = c.convertStatement(astIf.ElseBlock)
	}

	// Combine effects and regions.
	effects := hirCond.GetEffects()
	effects = effects.Union(hirThen.GetEffects())
	regions := hirCond.GetRegions()
	regions = regions.Union(hirThen.GetRegions())

	if hirElse != nil {
		effects = effects.Union(hirElse.GetEffects())
		regions = regions.Union(hirElse.GetRegions())
	}

	return &HIRIfStatement{
		ID:        generateNodeID(),
		Condition: hirCond,
		ThenBlock: hirThen,
		ElseBlock: hirElse,
		Effects:   effects,
		Regions:   regions,
		Metadata:  IRMetadata{},
		Span:      astIf.GetSpan(),
	}
}

// convertWhileStatement converts an AST while statement to HIR.
func (c *ASTToHIRConverter) convertWhileStatement(astWhile *ast.WhileStatement) HIRStatement {
	hirCond := c.convertExpression(astWhile.Condition)
	if hirCond == nil {
		return nil
	}

	hirBody := c.convertStatement(astWhile.Body)
	if hirBody == nil {
		return nil
	}

	// Combine effects and regions.
	effects := hirCond.GetEffects()
	effects = effects.Union(hirBody.GetEffects())
	regions := hirCond.GetRegions()
	regions = regions.Union(hirBody.GetRegions())

	return &HIRWhileStatement{
		ID:        generateNodeID(),
		Condition: hirCond,
		Body:      hirBody,
		Effects:   effects,
		Regions:   regions,
		Metadata:  IRMetadata{},
		Span:      astWhile.GetSpan(),
	}
}

// convertExpression converts an AST expression to HIR expression.
func (c *ASTToHIRConverter) convertExpression(astExpr ast.Expression) HIRExpression {
	switch expr := astExpr.(type) {
	case *ast.Identifier:
		return c.convertIdentifier(expr)
	case *ast.Literal:
		return c.convertLiteral(expr)
	case *ast.BinaryExpression:
		return c.convertBinaryExpression(expr)
	case *ast.UnaryExpression:
		return c.convertUnaryExpression(expr)
	case *ast.CallExpression:
		return c.convertCallExpression(expr)
	default:
		c.addError(ConversionError{
			Message: fmt.Sprintf("unsupported expression type: %T", expr),
			Span:    expr.GetSpan(),
			Kind:    ErrorKindTypeError,
		})

		return nil
	}
}

// convertIdentifier converts an AST identifier to HIR identifier.
func (c *ASTToHIRConverter) convertIdentifier(astId *ast.Identifier) HIRExpression {
	// Look up symbol in symbol table.
	symbol := c.symbolTable.LookupSymbol(astId.Value)
	if symbol == nil {
		c.addError(ConversionError{
			Message: fmt.Sprintf("undefined identifier: %s", astId.Value),
			Span:    astId.GetSpan(),
			Kind:    ErrorKindNameResolution,
		})

		return nil
	}

	// Mark symbol as used.
	symbol.Used = true

	// Analyze effects (reading a variable may have effects).
	effects := NewEffectSet()

	if symbol.Type.Kind == TypeKindPointer {
		// Reading through a pointer has memory read effect.
		memReadEffect := Effect{
			ID:          EffectID(generateNodeID()),
			Kind:        EffectKindMemoryRead,
			Description: fmt.Sprintf("reading variable %s", astId.Value),
			Modality:    EffectModalityMay,
			Scope:       EffectScopeLocal,
		}
		effects.AddEffect(memReadEffect)
	}

	return &HIRIdentifier{
		ID:           generateNodeID(),
		Name:         astId.Value,
		ResolvedDecl: symbol.Declaration,
		Type:         symbol.Type,
		Effects:      effects,
		Regions:      NewRegionSet(),
		Metadata:     IRMetadata{},
		Span:         astId.GetSpan(),
	}
}

// convertLiteral converts an AST literal to HIR literal.
func (c *ASTToHIRConverter) convertLiteral(astLit *ast.Literal) HIRExpression {
	// Determine type from literal value.
	var typeInfo TypeInfo

	switch astLit.Kind {
	case ast.LiteralInteger:
		if intID, exists := c.program.TypeInfo.Primitives["i32"]; exists {
			typeInfo = c.program.TypeInfo.Types[intID]
		}
	case ast.LiteralFloat:
		if floatID, exists := c.program.TypeInfo.Primitives["f64"]; exists {
			typeInfo = c.program.TypeInfo.Types[floatID]
		}
	case ast.LiteralBoolean:
		if boolID, exists := c.program.TypeInfo.Primitives["bool"]; exists {
			typeInfo = c.program.TypeInfo.Types[boolID]
		}
	case ast.LiteralString:
		if stringID, exists := c.program.TypeInfo.Primitives["string"]; exists {
			typeInfo = c.program.TypeInfo.Types[stringID]
		}
	default:
		c.addError(ConversionError{
			Message: fmt.Sprintf("unknown literal kind: %s", astLit.Kind.String()),
			Span:    astLit.GetSpan(),
			Kind:    ErrorKindTypeError,
		})

		return nil
	}

	return &HIRLiteral{
		ID:       generateNodeID(),
		Value:    astLit.Value,
		Type:     typeInfo,
		Metadata: IRMetadata{},
		Span:     astLit.GetSpan(),
	}
}

// convertBinaryExpression converts an AST binary expression to HIR.
func (c *ASTToHIRConverter) convertBinaryExpression(astBin *ast.BinaryExpression) HIRExpression {
	hirLeft := c.convertExpression(astBin.Left)
	if hirLeft == nil {
		return nil
	}

	hirRight := c.convertExpression(astBin.Right)
	if hirRight == nil {
		return nil
	}

	// Type checking and resolution.
	leftType := hirLeft.GetType()
	rightType := hirRight.GetType()
	resultType := c.resolveBinaryOperationType(astBin.Operator.String(), leftType, rightType, astBin.GetSpan())

	// Combine effects and regions.
	effects := hirLeft.GetEffects()
	effects = effects.Union(hirRight.GetEffects())
	regions := hirLeft.GetRegions()
	regions = regions.Union(hirRight.GetRegions())

	return &HIRBinaryExpression{
		ID:       generateNodeID(),
		Left:     hirLeft,
		Operator: astBin.Operator.String(),
		Right:    hirRight,
		Type:     resultType,
		Effects:  effects,
		Regions:  regions,
		Metadata: IRMetadata{},
		Span:     astBin.GetSpan(),
	}
}

// convertUnaryExpression converts an AST unary expression to HIR.
func (c *ASTToHIRConverter) convertUnaryExpression(astUnary *ast.UnaryExpression) HIRExpression {
	hirOperand := c.convertExpression(astUnary.Operand)
	if hirOperand == nil {
		return nil
	}

	// Type checking and resolution.
	operandType := hirOperand.GetType()
	resultType := c.resolveUnaryOperationType(astUnary.Operator.String(), operandType, astUnary.GetSpan())

	return &HIRUnaryExpression{
		ID:       generateNodeID(),
		Operator: astUnary.Operator.String(),
		Operand:  hirOperand,
		Type:     resultType,
		Effects:  hirOperand.GetEffects(),
		Regions:  hirOperand.GetRegions(),
		Metadata: IRMetadata{},
		Span:     astUnary.GetSpan(),
	}
}

// convertCallExpression converts an AST call expression to HIR.
func (c *ASTToHIRConverter) convertCallExpression(astCall *ast.CallExpression) HIRExpression {
	hirFunc := c.convertExpression(astCall.Function)
	if hirFunc == nil {
		return nil
	}

	hirArgs := make([]HIRExpression, len(astCall.Arguments))
	combinedEffects := hirFunc.GetEffects()
	combinedRegions := hirFunc.GetRegions()

	for i, astArg := range astCall.Arguments {
		hirArg := c.convertExpression(astArg)
		if hirArg == nil {
			return nil
		}

		hirArgs[i] = hirArg
		combinedEffects = combinedEffects.Union(hirArg.GetEffects())
		combinedRegions = combinedRegions.Union(hirArg.GetRegions())
	}

	// Type checking - function call.
	funcType := hirFunc.GetType()

	var resultType TypeInfo

	if funcType.Kind == TypeKindFunction {
		// Extract return type from function signature.
		// For function types, the last parameter in Parameters slice is the return type.
		if len(funcType.Parameters) > 0 {
			// The last type parameter is the return type.
			resultType = funcType.Parameters[len(funcType.Parameters)-1]
		} else {
			// Fallback to void if no return type specified.
			resultType = TypeInfo{Kind: TypeKindVoid, Name: "void"}
		}

		// If calling a built-in function, lookup the expected return type.
		if id, ok := hirFunc.(*HIRIdentifier); ok {
			if symbol := c.symbolTable.LookupSymbol(id.Name); symbol != nil {
				if symbol.Type.Kind == TypeKindFunction && len(symbol.Type.Parameters) > 0 {
					resultType = symbol.Type.Parameters[len(symbol.Type.Parameters)-1]
				}
			}
		}
	} else {
		c.addError(ConversionError{
			Message: "attempt to call non-function",
			Span:    astCall.GetSpan(),
			Kind:    ErrorKindTypeError,
		})

		return nil
	}

	// Function calls have potential side effects.
	callEffect := Effect{
		ID:          EffectID(generateNodeID()),
		Kind:        EffectKindIO, // Conservative assumption
		Description: "function call",
		Modality:    EffectModalityMay,
		Scope:       EffectScopeLocal,
	}
	combinedEffects.AddEffect(callEffect)

	return &HIRCallExpression{
		ID:        generateNodeID(),
		Function:  hirFunc,
		Arguments: hirArgs,
		Type:      resultType,
		Effects:   combinedEffects,
		Regions:   combinedRegions,
		Metadata:  IRMetadata{},
		Span:      astCall.GetSpan(),
	}
}

// convertType converts an AST type to HIR type.
func (c *ASTToHIRConverter) convertType(astType ast.Type) HIRType {
	switch typ := astType.(type) {
	case *ast.BasicType:
		return c.typeBuilder.BuildBasicType(typ.Kind.String(), typ.GetSpan())
	default:
		c.addError(ConversionError{
			Message: fmt.Sprintf("unsupported type: %T", typ),
			Span:    typ.GetSpan(),
			Kind:    ErrorKindTypeError,
		})

		return nil
	}
}

// Helper methods for type resolution.

func (c *ASTToHIRConverter) resolveBinaryOperationType(operator string, left, right TypeInfo, span position.Span) TypeInfo {
	switch operator {
	case "+", "-", "*", "/", "%":
		if left.Kind == TypeKindInteger && right.Kind == TypeKindInteger {
			return GetCommonType(left, right)
		}

		if left.Kind == TypeKindFloat && right.Kind == TypeKindFloat {
			return GetCommonType(left, right)
		}

		if (left.Kind == TypeKindInteger && right.Kind == TypeKindFloat) ||
			(left.Kind == TypeKindFloat && right.Kind == TypeKindInteger) {
			return GetCommonType(left, right)
		}

	case "==", "!=", "<", ">", "<=", ">=":
		if boolID, exists := c.program.TypeInfo.Primitives["bool"]; exists {
			return c.program.TypeInfo.Types[boolID]
		}

	case "&&", "||":
		if left.Kind == TypeKindBoolean && right.Kind == TypeKindBoolean {
			return left
		}
	}

	c.addError(ConversionError{
		Message: fmt.Sprintf("invalid binary operation: %s %s %s", left.Name, operator, right.Name),
		Span:    span,
		Kind:    ErrorKindTypeError,
	})

	return TypeInfo{Kind: TypeKindUnknown, Name: "unknown"}
}

func (c *ASTToHIRConverter) resolveUnaryOperationType(operator string, operand TypeInfo, span position.Span) TypeInfo {
	switch operator {
	case "-":
		if operand.Kind == TypeKindInteger || operand.Kind == TypeKindFloat {
			return operand
		}

	case "!":
		if operand.Kind == TypeKindBoolean {
			return operand
		}
	}

	c.addError(ConversionError{
		Message: fmt.Sprintf("invalid unary operation: %s %s", operator, operand.Name),
		Span:    span,
		Kind:    ErrorKindTypeError,
	})

	return TypeInfo{Kind: TypeKindUnknown, Name: "unknown"}
}

// Effect and region analysis methods.

func (c *ASTToHIRConverter) analyzeStatementEffects(stmt HIRStatement) EffectSet {
	if stmt == nil {
		return NewEffectSet()
	}

	return stmt.GetEffects()
}

func (c *ASTToHIRConverter) analyzeStatementRegions(stmt HIRStatement) RegionSet {
	if stmt == nil {
		return NewRegionSet()
	}

	return stmt.GetRegions()
}

// Symbol table operations.

func (st *SymbolTable) PushScope() {
	newScope := &Scope{
		Parent:  st.scopes[len(st.scopes)-1],
		Symbols: make(map[string]*Symbol),
		Level:   len(st.scopes),
	}
	st.scopes = append(st.scopes, newScope)
}

func (st *SymbolTable) PopScope() {
	if len(st.scopes) > 1 {
		st.scopes = st.scopes[:len(st.scopes)-1]
	}
}

func (st *SymbolTable) AddSymbol(name string, symbol *Symbol) {
	currentScope := st.scopes[len(st.scopes)-1]
	currentScope.Symbols[name] = symbol
	st.symbols[name] = symbol
}

func (st *SymbolTable) LookupSymbol(name string) *Symbol {
	// Search from current scope upward.
	for i := len(st.scopes) - 1; i >= 0; i-- {
		if symbol, exists := st.scopes[i].Symbols[name]; exists {
			return symbol
		}
	}

	return nil
}

// Error handling.

func (c *ASTToHIRConverter) addError(err ConversionError) {
	c.errors = append(c.errors, err)
}

func (e ConversionError) Error() string {
	return fmt.Sprintf("%s at %s", e.Message, e.Span.String())
}

// Generic type handling methods.

// detectGenericPattern detects if a function has generic type parameters.
func (c *ASTToHIRConverter) detectGenericPattern(astFunc *ast.FunctionDeclaration) bool {
	// Simple heuristics to detect generic patterns:.
	// 1. Function name contains generic patterns (e.g., contains<T>, map<K,V>)
	// 2. Parameters or return types use type variables
	// 3. Function uses type parameters in body
	// Check for common generic naming patterns.
	name := astFunc.Name.Value
	if strings.Contains(name, "<") && strings.Contains(name, ">") {
		return true
	}

	// Check for type variable usage in parameters.
	for _, param := range astFunc.Parameters {
		if c.containsTypeVariable(param.Type) {
			return true
		}
	}

	// Check return type for type variables.
	if astFunc.ReturnType != nil && c.containsTypeVariable(astFunc.ReturnType) {
		return true
	}

	return false
}

// extractTypeParameters extracts type parameters from a generic function.
func (c *ASTToHIRConverter) extractTypeParameters(astFunc *ast.FunctionDeclaration) []TypeInfo {
	typeParams := make([]TypeInfo, 0)

	// In a real implementation, this would parse explicit type parameter declarations.
	// For now, infer type parameters from usage patterns.

	if c.detectGenericPattern(astFunc) {
		// Create a basic type parameter for demonstration.
		typeParam := TypeInfo{
			Kind: TypeKindGeneric,
			Name: "T", // In reality, extract from declarations
		}
		typeParams = append(typeParams, typeParam)
	}

	return typeParams
}

// detectTypeGenericPattern detects if a type declaration is generic.
func (c *ASTToHIRConverter) detectTypeGenericPattern(astType *ast.TypeDeclaration) bool {
	// Check for generic patterns in type declarations.
	name := astType.Name.Value

	// Look for generic naming patterns.
	if strings.Contains(name, "<") && strings.Contains(name, ">") {
		return true
	}

	// Check if the type definition contains type variables.
	if astType.Type != nil && c.containsTypeVariable(astType.Type) {
		return true
	}

	return false
}

// extractTypeGenericParameters extracts type parameters from a generic type.
func (c *ASTToHIRConverter) extractTypeGenericParameters(astType *ast.TypeDeclaration) []TypeInfo {
	typeParams := make([]TypeInfo, 0)

	if c.detectTypeGenericPattern(astType) {
		// Create basic type parameters for demonstration.
		typeParam := TypeInfo{
			Kind: TypeKindGeneric,
			Name: "T",
		}
		typeParams = append(typeParams, typeParam)
	}

	return typeParams
}

// containsTypeVariable checks if a type contains type variables (indicating generics).
func (c *ASTToHIRConverter) containsTypeVariable(astType ast.Node) bool {
	// This is a simplified check - in reality, you'd do AST traversal.
	// to find type variable references.
	if astType == nil {
		return false
	}

	// Check if the type string representation contains common type variable patterns.
	typeStr := astType.String()

	// Look for single uppercase letters (common type variable convention).
	for _, char := range typeStr {
		if char >= 'A' && char <= 'Z' && len(typeStr) == 1 {
			return true
		}
	}

	// Look for generic syntax patterns.
	if strings.Contains(typeStr, "<") && strings.Contains(typeStr, ">") {
		return true
	}

	return false
}

// GetErrors returns all conversion errors.
func (c *ASTToHIRConverter) GetErrors() []ConversionError {
	return c.errors
}

// HasErrors returns true if there are any conversion errors.
func (c *ASTToHIRConverter) HasErrors() bool {
	return len(c.errors) > 0
}
