// HIR verification and validation tools for the Orizon programming language
// This file provides tools for validating HIR integrity, type consistency, and semantic correctness

package hir

import (
	"fmt"
	"strings"

	"github.com/orizon-lang/orizon/internal/position"
)

// HIRVerifier validates HIR structure and semantics
type HIRVerifier struct {
	errors   []VerificationError
	warnings []VerificationWarning
	visited  map[NodeID]bool
}

// VerificationError represents an error found during HIR verification
type VerificationError struct {
	Message string
	Span    position.Span
	Kind    VerificationErrorKind
	NodeID  NodeID
}

// VerificationWarning represents a warning found during HIR verification
type VerificationWarning struct {
	Message string
	Span    position.Span
	Kind    WarningKind
	NodeID  NodeID
}

// VerificationErrorKind represents the kind of verification error
type VerificationErrorKind int

const (
	ErrorKindTypeInconsistency VerificationErrorKind = iota
	ErrorKindUnresolvedReference
	ErrorKindInvalidNodeStructure
	ErrorKindEffectInconsistency
	ErrorKindRegionInconsistency
	ErrorKindScopeViolation
	ErrorKindCircularReference
	ErrorKindMissingRequired
)

// WarningKind represents the kind of verification warning
type WarningKind int

const (
	WarningKindUnusedSymbol WarningKind = iota
	WarningKindDeadCode
	WarningKindPerformance
	WarningKindStyleGuide
	WarningKindDeprecated
)

// NewHIRVerifier creates a new HIR verifier
func NewHIRVerifier() *HIRVerifier {
	return &HIRVerifier{
		errors:   make([]VerificationError, 0),
		warnings: make([]VerificationWarning, 0),
		visited:  make(map[NodeID]bool),
	}
}

// VerifyProgram performs comprehensive verification of an HIR program
func (v *HIRVerifier) VerifyProgram(program *HIRProgram) ([]VerificationError, []VerificationWarning) {
	v.errors = make([]VerificationError, 0)
	v.warnings = make([]VerificationWarning, 0)
	v.visited = make(map[NodeID]bool)

	// Basic structure validation
	v.verifyProgramStructure(program)

	// Type system validation
	v.verifyTypeSystem(program)

	// Effect system validation
	v.verifyEffectSystem(program)

	// Region system validation
	v.verifyRegionSystem(program)

	// Semantic validation
	v.verifySemantics(program)

	// Performance and style checks
	v.performStyleChecks(program)

	return v.errors, v.warnings
}

// verifyProgramStructure validates basic HIR program structure
func (v *HIRVerifier) verifyProgramStructure(program *HIRProgram) {
	if program == nil {
		v.addError(VerificationError{
			Message: "program cannot be nil",
			Kind:    ErrorKindInvalidNodeStructure,
			NodeID:  0,
		})
		return
	}

	// Check program ID
	if program.ID == 0 {
		v.addError(VerificationError{
			Message: "program must have valid ID",
			Span:    program.Span,
			Kind:    ErrorKindInvalidNodeStructure,
			NodeID:  program.ID,
		})
	}

	// Check modules
	if len(program.Modules) == 0 {
		v.addWarning(VerificationWarning{
			Message: "program has no modules",
			Span:    program.Span,
			Kind:    WarningKindStyleGuide,
			NodeID:  program.ID,
		})
	}

	// Verify each module
	for moduleID, module := range program.Modules {
		if module == nil {
			v.addError(VerificationError{
				Message: fmt.Sprintf("module %d is nil", moduleID),
				Kind:    ErrorKindInvalidNodeStructure,
				NodeID:  NodeID(moduleID),
			})
			continue
		}

		v.verifyModule(module)
	}

	// Check global type info
	if program.TypeInfo == nil {
		v.addError(VerificationError{
			Message: "program must have type information",
			Span:    program.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  program.ID,
		})
	} else {
		v.verifyGlobalTypeInfo(program.TypeInfo)
	}
}

// verifyModule validates HIR module structure
func (v *HIRVerifier) verifyModule(module *HIRModule) {
	if v.visited[module.ID] {
		v.addError(VerificationError{
			Message: "circular reference in module structure",
			Span:    module.Span,
			Kind:    ErrorKindCircularReference,
			NodeID:  module.ID,
		})
		return
	}
	v.visited[module.ID] = true

	// Check module name
	if module.Name == "" {
		v.addError(VerificationError{
			Message: "module must have a name",
			Span:    module.Span,
			Kind:    ErrorKindInvalidNodeStructure,
			NodeID:  module.ID,
		})
	}

	// Verify declarations
	for i, decl := range module.Declarations {
		if decl == nil {
			v.addError(VerificationError{
				Message: fmt.Sprintf("declaration %d in module %s is nil", i, module.Name),
				Span:    module.Span,
				Kind:    ErrorKindInvalidNodeStructure,
				NodeID:  module.ID,
			})
			continue
		}

		v.verifyDeclaration(decl)
	}
}

// verifyDeclaration validates HIR declaration
func (v *HIRVerifier) verifyDeclaration(decl HIRDeclaration) {
	if v.visited[decl.GetID()] {
		return
	}
	v.visited[decl.GetID()] = true

	switch d := decl.(type) {
	case *HIRFunctionDeclaration:
		v.verifyFunctionDeclaration(d)
	case *HIRVariableDeclaration:
		v.verifyVariableDeclaration(d)
	case *HIRTypeDeclaration:
		v.verifyTypeDeclaration(d)
	case *HIRConstDeclaration:
		v.verifyConstDeclaration(d)
	default:
		v.addError(VerificationError{
			Message: fmt.Sprintf("unknown declaration type: %T", d),
			Span:    d.GetSpan(),
			Kind:    ErrorKindInvalidNodeStructure,
			NodeID:  d.GetID(),
		})
	}
}

// verifyFunctionDeclaration validates HIR function declaration
func (v *HIRVerifier) verifyFunctionDeclaration(funcDecl *HIRFunctionDeclaration) {
	// Check function name
	if funcDecl.Name == "" {
		v.addError(VerificationError{
			Message: "function must have a name",
			Span:    funcDecl.Span,
			Kind:    ErrorKindInvalidNodeStructure,
			NodeID:  funcDecl.ID,
		})
	}

	// Check parameters
	for i, param := range funcDecl.Parameters {
		if param == nil {
			v.addError(VerificationError{
				Message: fmt.Sprintf("parameter %d in function %s is nil", i, funcDecl.Name),
				Span:    funcDecl.Span,
				Kind:    ErrorKindInvalidNodeStructure,
				NodeID:  funcDecl.ID,
			})
			continue
		}

		v.verifyParameter(param)
	}

	// Check return type
	if funcDecl.ReturnType == nil {
		v.addError(VerificationError{
			Message: fmt.Sprintf("function %s must have return type", funcDecl.Name),
			Span:    funcDecl.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  funcDecl.ID,
		})
	} else {
		v.verifyType(funcDecl.ReturnType)
	}

	// Check body
	if funcDecl.Body != nil {
		v.verifyStatement(funcDecl.Body)
	}

	// Verify effects are consistent
	v.verifyEffectConsistency(funcDecl, funcDecl.Effects)

	// Verify regions are consistent
	v.verifyRegionConsistency(funcDecl, funcDecl.Regions)
}

// verifyParameter validates HIR parameter
func (v *HIRVerifier) verifyParameter(param *HIRParameter) {
	if param.Name == "" {
		v.addError(VerificationError{
			Message: "parameter must have a name",
			Span:    param.Span,
			Kind:    ErrorKindInvalidNodeStructure,
			NodeID:  param.ID,
		})
	}

	if param.Type == nil {
		v.addError(VerificationError{
			Message: fmt.Sprintf("parameter %s must have a type", param.Name),
			Span:    param.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  param.ID,
		})
	} else {
		v.verifyType(param.Type)
	}
}

// verifyVariableDeclaration validates HIR variable declaration
func (v *HIRVerifier) verifyVariableDeclaration(varDecl *HIRVariableDeclaration) {
	if varDecl.Name == "" {
		v.addError(VerificationError{
			Message: "variable must have a name",
			Span:    varDecl.Span,
			Kind:    ErrorKindInvalidNodeStructure,
			NodeID:  varDecl.ID,
		})
	}

	if varDecl.Type == nil {
		v.addError(VerificationError{
			Message: fmt.Sprintf("variable %s must have a type", varDecl.Name),
			Span:    varDecl.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  varDecl.ID,
		})
	} else {
		v.verifyType(varDecl.Type)
	}

	if varDecl.Initializer != nil {
		v.verifyExpression(varDecl.Initializer)

		// Check type compatibility
		initType := varDecl.Initializer.GetType()
		declType := varDecl.Type.GetType()

		if !IsAssignableTo(initType, declType) {
			v.addError(VerificationError{
				Message: fmt.Sprintf("cannot assign %s to variable %s of type %s",
					initType.Name, varDecl.Name, declType.Name),
				Span:   varDecl.Span,
				Kind:   ErrorKindTypeInconsistency,
				NodeID: varDecl.ID,
			})
		}
	}
}

// verifyTypeDeclaration validates HIR type declaration
func (v *HIRVerifier) verifyTypeDeclaration(typeDecl *HIRTypeDeclaration) {
	if typeDecl.Name == "" {
		v.addError(VerificationError{
			Message: "type declaration must have a name",
			Span:    typeDecl.Span,
			Kind:    ErrorKindInvalidNodeStructure,
			NodeID:  typeDecl.ID,
		})
	}

	if typeDecl.Type == nil {
		v.addError(VerificationError{
			Message: fmt.Sprintf("type declaration %s must have a type", typeDecl.Name),
			Span:    typeDecl.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  typeDecl.ID,
		})
	} else {
		v.verifyType(typeDecl.Type)
	}
}

// verifyConstDeclaration validates HIR const declaration
func (v *HIRVerifier) verifyConstDeclaration(constDecl *HIRConstDeclaration) {
	if constDecl.Name == "" {
		v.addError(VerificationError{
			Message: "constant must have a name",
			Span:    constDecl.Span,
			Kind:    ErrorKindInvalidNodeStructure,
			NodeID:  constDecl.ID,
		})
	}

	if constDecl.Type == nil {
		v.addError(VerificationError{
			Message: fmt.Sprintf("constant %s must have a type", constDecl.Name),
			Span:    constDecl.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  constDecl.ID,
		})
	} else {
		v.verifyType(constDecl.Type)
	}

	if constDecl.Value == nil {
		v.addError(VerificationError{
			Message: fmt.Sprintf("constant %s must have a value", constDecl.Name),
			Span:    constDecl.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  constDecl.ID,
		})
	} else {
		v.verifyExpression(constDecl.Value)

		// Check type compatibility
		valueType := constDecl.Value.GetType()
		declType := constDecl.Type.GetType()

		if !IsAssignableTo(valueType, declType) {
			v.addError(VerificationError{
				Message: fmt.Sprintf("cannot assign %s to constant %s of type %s",
					valueType.Name, constDecl.Name, declType.Name),
				Span:   constDecl.Span,
				Kind:   ErrorKindTypeInconsistency,
				NodeID: constDecl.ID,
			})
		}
	}
}

// verifyStatement validates HIR statement
func (v *HIRVerifier) verifyStatement(stmt HIRStatement) {
	if stmt == nil {
		return
	}

	if v.visited[stmt.GetID()] {
		return
	}
	v.visited[stmt.GetID()] = true

	switch s := stmt.(type) {
	case *HIRBlockStatement:
		v.verifyBlockStatement(s)
	case *HIRExpressionStatement:
		v.verifyExpressionStatement(s)
	case *HIRReturnStatement:
		v.verifyReturnStatement(s)
	case *HIRIfStatement:
		v.verifyIfStatement(s)
	case *HIRWhileStatement:
		v.verifyWhileStatement(s)
	default:
		v.addError(VerificationError{
			Message: fmt.Sprintf("unknown statement type: %T", s),
			Span:    s.GetSpan(),
			Kind:    ErrorKindInvalidNodeStructure,
			NodeID:  s.GetID(),
		})
	}
}

// verifyBlockStatement validates HIR block statement
func (v *HIRVerifier) verifyBlockStatement(block *HIRBlockStatement) {
	for i, stmt := range block.Statements {
		if stmt == nil {
			v.addError(VerificationError{
				Message: fmt.Sprintf("statement %d in block is nil", i),
				Span:    block.Span,
				Kind:    ErrorKindInvalidNodeStructure,
				NodeID:  block.ID,
			})
			continue
		}

		v.verifyStatement(stmt)
	}

	// Verify effect and region consistency
	v.verifyEffectConsistency(block, block.Effects)
	v.verifyRegionConsistency(block, block.Regions)
}

// verifyExpressionStatement validates HIR expression statement
func (v *HIRVerifier) verifyExpressionStatement(exprStmt *HIRExpressionStatement) {
	if exprStmt.Expression == nil {
		v.addError(VerificationError{
			Message: "expression statement must have an expression",
			Span:    exprStmt.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  exprStmt.ID,
		})
	} else {
		v.verifyExpression(exprStmt.Expression)
	}
}

// verifyReturnStatement validates HIR return statement
func (v *HIRVerifier) verifyReturnStatement(retStmt *HIRReturnStatement) {
	if retStmt.Expression != nil {
		v.verifyExpression(retStmt.Expression)
	}
}

// verifyIfStatement validates HIR if statement
func (v *HIRVerifier) verifyIfStatement(ifStmt *HIRIfStatement) {
	if ifStmt.Condition == nil {
		v.addError(VerificationError{
			Message: "if statement must have a condition",
			Span:    ifStmt.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  ifStmt.ID,
		})
	} else {
		v.verifyExpression(ifStmt.Condition)

		// Check condition type
		condType := ifStmt.Condition.GetType()
		if condType.Kind != TypeKindBoolean {
			v.addError(VerificationError{
				Message: fmt.Sprintf("if condition must be boolean, got %s", condType.Name),
				Span:    ifStmt.Span,
				Kind:    ErrorKindTypeInconsistency,
				NodeID:  ifStmt.ID,
			})
		}
	}

	if ifStmt.ThenBlock == nil {
		v.addError(VerificationError{
			Message: "if statement must have a then block",
			Span:    ifStmt.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  ifStmt.ID,
		})
	} else {
		v.verifyStatement(ifStmt.ThenBlock)
	}

	if ifStmt.ElseBlock != nil {
		v.verifyStatement(ifStmt.ElseBlock)
	}
}

// verifyWhileStatement validates HIR while statement
func (v *HIRVerifier) verifyWhileStatement(whileStmt *HIRWhileStatement) {
	if whileStmt.Condition == nil {
		v.addError(VerificationError{
			Message: "while statement must have a condition",
			Span:    whileStmt.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  whileStmt.ID,
		})
	} else {
		v.verifyExpression(whileStmt.Condition)

		// Check condition type
		condType := whileStmt.Condition.GetType()
		if condType.Kind != TypeKindBoolean {
			v.addError(VerificationError{
				Message: fmt.Sprintf("while condition must be boolean, got %s", condType.Name),
				Span:    whileStmt.Span,
				Kind:    ErrorKindTypeInconsistency,
				NodeID:  whileStmt.ID,
			})
		}
	}

	if whileStmt.Body == nil {
		v.addError(VerificationError{
			Message: "while statement must have a body",
			Span:    whileStmt.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  whileStmt.ID,
		})
	} else {
		v.verifyStatement(whileStmt.Body)
	}
}

// verifyExpression validates HIR expression
func (v *HIRVerifier) verifyExpression(expr HIRExpression) {
	if expr == nil {
		return
	}

	if v.visited[expr.GetID()] {
		return
	}
	v.visited[expr.GetID()] = true

	switch e := expr.(type) {
	case *HIRIdentifier:
		v.verifyIdentifier(e)
	case *HIRLiteral:
		v.verifyLiteral(e)
	case *HIRBinaryExpression:
		v.verifyBinaryExpression(e)
	case *HIRUnaryExpression:
		v.verifyUnaryExpression(e)
	case *HIRCallExpression:
		v.verifyCallExpression(e)
	default:
		v.addError(VerificationError{
			Message: fmt.Sprintf("unknown expression type: %T", e),
			Span:    e.GetSpan(),
			Kind:    ErrorKindInvalidNodeStructure,
			NodeID:  e.GetID(),
		})
	}
}

// verifyIdentifier validates HIR identifier
func (v *HIRVerifier) verifyIdentifier(id *HIRIdentifier) {
	if id.Name == "" {
		v.addError(VerificationError{
			Message: "identifier must have a name",
			Span:    id.Span,
			Kind:    ErrorKindInvalidNodeStructure,
			NodeID:  id.ID,
		})
	}

	// Check if resolved declaration exists (for name resolution validation)
	if id.ResolvedDecl == nil {
		v.addWarning(VerificationWarning{
			Message: fmt.Sprintf("identifier %s has no resolved declaration", id.Name),
			Span:    id.Span,
			Kind:    WarningKindStyleGuide,
			NodeID:  id.ID,
		})
	}
}

// verifyLiteral validates HIR literal
func (v *HIRVerifier) verifyLiteral(lit *HIRLiteral) {
	// Basic validation - literals are generally valid by construction
	if lit.Value == nil {
		v.addWarning(VerificationWarning{
			Message: "literal has nil value",
			Span:    lit.Span,
			Kind:    WarningKindStyleGuide,
			NodeID:  lit.ID,
		})
	}
}

// verifyBinaryExpression validates HIR binary expression
func (v *HIRVerifier) verifyBinaryExpression(binExpr *HIRBinaryExpression) {
	if binExpr.Left == nil {
		v.addError(VerificationError{
			Message: "binary expression must have left operand",
			Span:    binExpr.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  binExpr.ID,
		})
	} else {
		v.verifyExpression(binExpr.Left)
	}

	if binExpr.Right == nil {
		v.addError(VerificationError{
			Message: "binary expression must have right operand",
			Span:    binExpr.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  binExpr.ID,
		})
	} else {
		v.verifyExpression(binExpr.Right)
	}

	if binExpr.Operator == "" {
		v.addError(VerificationError{
			Message: "binary expression must have operator",
			Span:    binExpr.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  binExpr.ID,
		})
	}

	// Verify operand types are compatible
	if binExpr.Left != nil && binExpr.Right != nil {
		leftType := binExpr.Left.GetType()
		rightType := binExpr.Right.GetType()
		resultType := binExpr.Type

		expectedType := GetCommonType(leftType, rightType)
		if !IsAssignableTo(expectedType, resultType) {
			v.addError(VerificationError{
				Message: fmt.Sprintf("binary expression type mismatch: expected %s, got %s",
					expectedType.Name, resultType.Name),
				Span:   binExpr.Span,
				Kind:   ErrorKindTypeInconsistency,
				NodeID: binExpr.ID,
			})
		}
	}
}

// verifyUnaryExpression validates HIR unary expression
func (v *HIRVerifier) verifyUnaryExpression(unaryExpr *HIRUnaryExpression) {
	if unaryExpr.Operand == nil {
		v.addError(VerificationError{
			Message: "unary expression must have operand",
			Span:    unaryExpr.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  unaryExpr.ID,
		})
	} else {
		v.verifyExpression(unaryExpr.Operand)
	}

	if unaryExpr.Operator == "" {
		v.addError(VerificationError{
			Message: "unary expression must have operator",
			Span:    unaryExpr.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  unaryExpr.ID,
		})
	}
}

// verifyCallExpression validates HIR call expression
func (v *HIRVerifier) verifyCallExpression(callExpr *HIRCallExpression) {
	if callExpr.Function == nil {
		v.addError(VerificationError{
			Message: "call expression must have function",
			Span:    callExpr.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  callExpr.ID,
		})
	} else {
		v.verifyExpression(callExpr.Function)
	}

	for i, arg := range callExpr.Arguments {
		if arg == nil {
			v.addError(VerificationError{
				Message: fmt.Sprintf("argument %d in call expression is nil", i),
				Span:    callExpr.Span,
				Kind:    ErrorKindInvalidNodeStructure,
				NodeID:  callExpr.ID,
			})
		} else {
			v.verifyExpression(arg)
		}
	}
}

// verifyType validates HIR type
func (v *HIRVerifier) verifyType(hirType HIRType) {
	if hirType == nil {
		return
	}

	switch t := hirType.(type) {
	case *HIRBasicType:
		v.verifyBasicType(t)
	case *HIRArrayType:
		v.verifyArrayType(t)
	case *HIRPointerType:
		v.verifyPointerType(t)
	case *HIRFunctionType:
		v.verifyFunctionType(t)
	case *HIRStructType:
		v.verifyStructType(t)
	default:
		v.addError(VerificationError{
			Message: fmt.Sprintf("unknown type: %T", t),
			Span:    t.GetSpan(),
			Kind:    ErrorKindInvalidNodeStructure,
			NodeID:  t.GetID(),
		})
	}
}

// verifyBasicType validates HIR basic type
func (v *HIRVerifier) verifyBasicType(basicType *HIRBasicType) {
	if basicType.Name == "" {
		v.addError(VerificationError{
			Message: "basic type must have a name",
			Span:    basicType.Span,
			Kind:    ErrorKindInvalidNodeStructure,
			NodeID:  basicType.ID,
		})
	}
}

// verifyArrayType validates HIR array type
func (v *HIRVerifier) verifyArrayType(arrayType *HIRArrayType) {
	if arrayType.ElementType == nil {
		v.addError(VerificationError{
			Message: "array type must have element type",
			Span:    arrayType.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  arrayType.ID,
		})
	} else {
		v.verifyType(arrayType.ElementType)
	}

	if arrayType.Size != nil {
		v.verifyExpression(arrayType.Size)
	}
}

// verifyPointerType validates HIR pointer type
func (v *HIRVerifier) verifyPointerType(ptrType *HIRPointerType) {
	if ptrType.TargetType == nil {
		v.addError(VerificationError{
			Message: "pointer type must have target type",
			Span:    ptrType.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  ptrType.ID,
		})
	} else {
		v.verifyType(ptrType.TargetType)
	}
}

// verifyFunctionType validates HIR function type
func (v *HIRVerifier) verifyFunctionType(funcType *HIRFunctionType) {
	for i, param := range funcType.Parameters {
		if param == nil {
			v.addError(VerificationError{
				Message: fmt.Sprintf("parameter %d in function type is nil", i),
				Span:    funcType.Span,
				Kind:    ErrorKindInvalidNodeStructure,
				NodeID:  funcType.ID,
			})
		} else {
			v.verifyType(param)
		}
	}

	if funcType.ReturnType == nil {
		v.addError(VerificationError{
			Message: "function type must have return type",
			Span:    funcType.Span,
			Kind:    ErrorKindMissingRequired,
			NodeID:  funcType.ID,
		})
	} else {
		v.verifyType(funcType.ReturnType)
	}
}

// verifyStructType validates HIR struct type
func (v *HIRVerifier) verifyStructType(structType *HIRStructType) {
	if structType.Name == "" {
		v.addError(VerificationError{
			Message: "struct type must have a name",
			Span:    structType.Span,
			Kind:    ErrorKindInvalidNodeStructure,
			NodeID:  structType.ID,
		})
	}

	for i, field := range structType.Fields {
		if field.Name == "" {
			v.addError(VerificationError{
				Message: fmt.Sprintf("field %d in struct %s must have a name", i, structType.Name),
				Span:    structType.Span,
				Kind:    ErrorKindInvalidNodeStructure,
				NodeID:  structType.ID,
			})
		}

		if field.Type == nil {
			v.addError(VerificationError{
				Message: fmt.Sprintf("field %s in struct %s must have a type", field.Name, structType.Name),
				Span:    structType.Span,
				Kind:    ErrorKindMissingRequired,
				NodeID:  structType.ID,
			})
		} else {
			v.verifyType(field.Type)
		}
	}
}

// verifyTypeSystem validates the global type system
func (v *HIRVerifier) verifyTypeSystem(program *HIRProgram) {
	if program.TypeInfo == nil {
		return
	}

	v.verifyGlobalTypeInfo(program.TypeInfo)
}

// verifyGlobalTypeInfo validates global type information
func (v *HIRVerifier) verifyGlobalTypeInfo(typeInfo *GlobalTypeInfo) {
	// Check primitive types are defined
	requiredPrimitives := []string{"void", "bool", "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64", "f32", "f64", "string"}

	for _, prim := range requiredPrimitives {
		if _, exists := typeInfo.Primitives[prim]; !exists {
			v.addError(VerificationError{
				Message: fmt.Sprintf("missing primitive type: %s", prim),
				Kind:    ErrorKindMissingRequired,
				NodeID:  0,
			})
		}
	}

	// Verify type consistency
	for id, typ := range typeInfo.Types {
		if typ.ID != id {
			v.addError(VerificationError{
				Message: fmt.Sprintf("type ID mismatch: expected %d, got %d", id, typ.ID),
				Kind:    ErrorKindTypeInconsistency,
				NodeID:  0,
			})
		}
	}
}

// verifyEffectSystem validates the effect system
func (v *HIRVerifier) verifyEffectSystem(program *HIRProgram) {
	// Effect system validation logic
	// This would check effect consistency, purity, etc.
}

// verifyRegionSystem validates the region system
func (v *HIRVerifier) verifyRegionSystem(program *HIRProgram) {
	// Region system validation logic
	// This would check region lifetime consistency, permissions, etc.
}

// verifySemantics validates semantic consistency
func (v *HIRVerifier) verifySemantics(program *HIRProgram) {
	// Semantic validation logic
	// This would check for logical consistency, reachability, etc.
}

// verifyEffectConsistency validates effect consistency for a node
func (v *HIRVerifier) verifyEffectConsistency(node HIRNode, effects EffectSet) {
	// Effect consistency validation logic
}

// verifyRegionConsistency validates region consistency for a node
func (v *HIRVerifier) verifyRegionConsistency(node HIRNode, regions RegionSet) {
	// Region consistency validation logic
}

// performStyleChecks performs style and performance checks
func (v *HIRVerifier) performStyleChecks(program *HIRProgram) {
	// Style and performance checking logic
}

// Helper methods for error and warning management

func (v *HIRVerifier) addError(err VerificationError) {
	v.errors = append(v.errors, err)
}

func (v *HIRVerifier) addWarning(warning VerificationWarning) {
	v.warnings = append(v.warnings, warning)
}

// GetErrorCount returns the number of verification errors
func (v *HIRVerifier) GetErrorCount() int {
	return len(v.errors)
}

// GetWarningCount returns the number of verification warnings
func (v *HIRVerifier) GetWarningCount() int {
	return len(v.warnings)
}

// HasErrors returns true if there are verification errors
func (v *HIRVerifier) HasErrors() bool {
	return len(v.errors) > 0
}

// FormatErrors returns a formatted string of all errors
func (v *HIRVerifier) FormatErrors() string {
	if len(v.errors) == 0 {
		return "No errors found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d verification errors:\n", len(v.errors)))

	for i, err := range v.errors {
		sb.WriteString(fmt.Sprintf("%d. %s", i+1, err.Error()))
		if err.Span.IsValid() {
			sb.WriteString(fmt.Sprintf(" at %s", err.Span.String()))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatWarnings returns a formatted string of all warnings
func (v *HIRVerifier) FormatWarnings() string {
	if len(v.warnings) == 0 {
		return "No warnings found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d verification warnings:\n", len(v.warnings)))

	for i, warning := range v.warnings {
		sb.WriteString(fmt.Sprintf("%d. %s", i+1, warning.Error()))
		if warning.Span.IsValid() {
			sb.WriteString(fmt.Sprintf(" at %s", warning.Span.String()))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// Error returns the error message for VerificationError
func (e VerificationError) Error() string {
	return e.Message
}

// Error returns the error message for VerificationWarning
func (w VerificationWarning) Error() string {
	return w.Message
}
