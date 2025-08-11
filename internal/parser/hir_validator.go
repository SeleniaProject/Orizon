// HIR verification and validation for Orizon language
// This file implements validation and verification tools for HIR:
// 1. Structural consistency checks
// 2. Type coherence validation
// 3. Control flow analysis
// 4. Symbol resolution verification
// 5. Semantic constraint checking

package parser

import (
	"fmt"
	"strings"
)

// ====== HIR Validator ======

// HIRValidator performs validation on HIR structures
type HIRValidator struct {
	errors   []HIRValidationError
	warnings []HIRValidationWarning
	context  *ValidationContext
}

// HIRValidationError represents a validation error
type HIRValidationError struct {
	Span    Span
	Message string
	Code    ErrorCode
}

// HIRValidationWarning represents a validation warning
type HIRValidationWarning struct {
	Span    Span
	Message string
	Code    WarningCode
}

// ErrorCode represents validation error types
type ErrorCode int

const (
	ErrorUnresolvedSymbol ErrorCode = iota
	ErrorTypeMismatch
	ErrorInvalidControlFlow
	ErrorDuplicateDeclaration
	ErrorInvalidScope
	ErrorMissingReturn
	ErrorUnreachableCode
	ErrorInvalidTypeUsage
)

// WarningCode represents validation warning types
type WarningCode int

const (
	WarningUnusedVariable WarningCode = iota
	WarningUnusedFunction
	WarningUnusedParameter
	WarningDeadCode
	WarningPerformanceIssue
	WarningStyleIssue
)

// ValidationContext provides context for validation
type ValidationContext struct {
	currentModule   *HIRModule
	currentFunction *HIRFunction
	currentScope    *HIRScope
	symbolTable     *SymbolTable
	typeTable       map[string]*HIRType
	controlFlow     *ControlFlowGraph
}

// ControlFlowGraph represents control flow information
type ControlFlowGraph struct {
	nodes []CFGNode
	edges []CFGEdge
}

// CFGNode represents a control flow node
type CFGNode struct {
	id   int
	kind CFGNodeKind
	span Span
	data interface{}
}

// CFGNodeKind represents control flow node types
type CFGNodeKind int

const (
	CFGNodeEntry CFGNodeKind = iota
	CFGNodeExit
	CFGNodeStatement
	CFGNodeExpression
	CFGNodeBranch
	CFGNodeLoop
	CFGNodeCall
	CFGNodeReturn
)

// CFGEdge represents a control flow edge
type CFGEdge struct {
	from      int
	to        int
	condition *HIRExpression // for conditional edges
}

// NewHIRValidator creates a new HIR validator
func NewHIRValidator() *HIRValidator {
	return &HIRValidator{
		errors:   make([]HIRValidationError, 0),
		warnings: make([]HIRValidationWarning, 0),
		context: &ValidationContext{
			typeTable: make(map[string]*HIRType),
			controlFlow: &ControlFlowGraph{
				nodes: make([]CFGNode, 0),
				edges: make([]CFGEdge, 0),
			},
		},
	}
}

// ValidateModule validates an HIR module
func (validator *HIRValidator) ValidateModule(module *HIRModule) []HIRValidationError {
	validator.context.currentModule = module

	// Build symbol table
	validator.buildSymbolTable(module)

	// Validate types
	validator.validateTypes(module)

	// Validate functions
	for _, function := range module.Functions {
		validator.validateFunction(function)
	}

	// Validate variables
	for _, variable := range module.Variables {
		validator.validateVariable(variable)
	}

	// Check for unused symbols
	validator.checkUnusedSymbols()

	return validator.errors
}

// buildSymbolTable constructs the symbol table for validation
func (validator *HIRValidator) buildSymbolTable(module *HIRModule) {
	symbolTable := NewSymbolTable()
	validator.context.symbolTable = symbolTable

	// Register functions
	for _, function := range module.Functions {
		if existing, exists := symbolTable.functions[function.Name]; exists {
			validator.addError(function.Span, fmt.Sprintf("duplicate function declaration: %s", function.Name), ErrorDuplicateDeclaration)
			validator.addError(existing.Span, fmt.Sprintf("previous declaration here"), ErrorDuplicateDeclaration)
		} else {
			symbolTable.functions[function.Name] = function
		}
	}

	// Register global variables
	for _, variable := range module.Variables {
		if existing, exists := symbolTable.globals[variable.Name]; exists {
			validator.addError(variable.Span, fmt.Sprintf("duplicate variable declaration: %s", variable.Name), ErrorDuplicateDeclaration)
			validator.addError(existing.Span, fmt.Sprintf("previous declaration here"), ErrorDuplicateDeclaration)
		} else {
			symbolTable.globals[variable.Name] = variable
		}
	}

	// Register types
	for _, typeDef := range module.Types {
		if existing, exists := symbolTable.types[typeDef.Name]; existing != nil && exists {
			validator.addError(typeDef.Span, fmt.Sprintf("duplicate type declaration: %s", typeDef.Name), ErrorDuplicateDeclaration)
		} else {
			hirType := NewHIRType(typeDef.Span, HIRTypeStruct, typeDef)
			symbolTable.types[typeDef.Name] = hirType
			validator.context.typeTable[typeDef.Name] = hirType
		}
	}
}

// validateTypes validates type definitions and usage
func (validator *HIRValidator) validateTypes(module *HIRModule) {
	for _, typeDef := range module.Types {
		validator.validateTypeDefinition(typeDef)
	}
}

// validateTypeDefinition validates a single type definition
func (validator *HIRValidator) validateTypeDefinition(typeDef *HIRTypeDefinition) {
	// Check for recursive type definitions without indirection
	// This is a simplified check - full implementation would be more complex
	if validator.isRecursiveType(typeDef, make(map[string]bool)) {
		validator.addWarning(typeDef.Span, fmt.Sprintf("potentially recursive type: %s", typeDef.Name), WarningPerformanceIssue)
	}
}

// isRecursiveType checks if a type is potentially recursive
func (validator *HIRValidator) isRecursiveType(typeDef *HIRTypeDefinition, visited map[string]bool) bool {
	if visited[typeDef.Name] {
		return true
	}

	visited[typeDef.Name] = true

	// For now, just return false - real implementation would analyze the type structure
	return false
}

// validateFunction validates a function
func (validator *HIRValidator) validateFunction(function *HIRFunction) {
	validator.context.currentFunction = function

	// Check for missing return statements
	if function.ReturnType != nil && !validator.hasReturnStatement(function.Body) {
		if !validator.isUnitType(function.ReturnType) {
			validator.addError(function.Span, fmt.Sprintf("function %s missing return statement", function.Name), ErrorMissingReturn)
		}
	}

	// Validate function body
	if function.Body != nil {
		validator.validateBlock(function.Body)
	}

	// Check parameter usage
	for _, param := range function.Parameters {
		if !validator.isParameterUsed(param, function.Body) {
			validator.addWarning(function.Span, fmt.Sprintf("unused parameter: %s", param.Name), WarningUnusedParameter)
		}
	}

	// Build control flow graph
	validator.buildControlFlowGraph(function)

	// Validate control flow
	validator.validateControlFlow(function)
}

// hasReturnStatement checks if a block has a return statement
func (validator *HIRValidator) hasReturnStatement(block *HIRBlock) bool {
	for _, stmt := range block.Statements {
		if stmt.Kind == HIRStmtReturn {
			return true
		}

		// Check nested blocks
		switch data := stmt.Data.(type) {
		case *HIRIfStatement:
			if validator.hasReturnStatement(data.ThenBlock) {
				if data.ElseBlock == nil || validator.hasReturnStatement(data.ElseBlock) {
					return true
				}
			}
		case *HIRWhileStatement:
			// While loops don't guarantee execution, so don't count
		}
	}

	return false
}

// isUnitType checks if a type is the unit type
func (validator *HIRValidator) isUnitType(hirType *HIRType) bool {
	if hirType.Kind == HIRTypePrimitive {
		if primitive, ok := hirType.Data.(*HIRPrimitiveType); ok {
			return primitive.Name == "unit"
		}
	}
	return false
}

// isParameterUsed checks if a parameter is used in the function body
func (validator *HIRValidator) isParameterUsed(param *HIRParameter, block *HIRBlock) bool {
	// Simple implementation - real version would do proper usage analysis
	return validator.isVariableUsedInBlock(param.Name, block)
}

// isVariableUsedInBlock checks if a variable is used in a block
func (validator *HIRValidator) isVariableUsedInBlock(varName string, block *HIRBlock) bool {
	for _, stmt := range block.Statements {
		if validator.isVariableUsedInStatement(varName, stmt) {
			return true
		}
	}

	if block.Expression != nil {
		return validator.isVariableUsedInExpression(varName, block.Expression)
	}

	return false
}

// isVariableUsedInStatement checks if a variable is used in a statement
func (validator *HIRValidator) isVariableUsedInStatement(varName string, stmt *HIRStatement) bool {
	switch data := stmt.Data.(type) {
	case *HIRExpression:
		return validator.isVariableUsedInExpression(varName, data)
	case *HIRIfStatement:
		return validator.isVariableUsedInExpression(varName, data.Condition) ||
			validator.isVariableUsedInBlock(varName, data.ThenBlock) ||
			(data.ElseBlock != nil && validator.isVariableUsedInBlock(varName, data.ElseBlock))
	case *HIRWhileStatement:
		return validator.isVariableUsedInExpression(varName, data.Condition) ||
			validator.isVariableUsedInBlock(varName, data.Body)
	case *HIRReturnStatement:
		return data.Value != nil && validator.isVariableUsedInExpression(varName, data.Value)
	case *HIRAssignStatement:
		return validator.isVariableUsedInExpression(varName, data.Target) ||
			validator.isVariableUsedInExpression(varName, data.Value)
	}

	return false
}

// isVariableUsedInExpression checks if a variable is used in an expression
func (validator *HIRValidator) isVariableUsedInExpression(varName string, expr *HIRExpression) bool {
	switch data := expr.Data.(type) {
	case *HIRVariableExpression:
		return data.Name == varName
	case *HIRBinaryExpression:
		return validator.isVariableUsedInExpression(varName, data.Left) ||
			validator.isVariableUsedInExpression(varName, data.Right)
	case *HIRUnaryExpression:
		return validator.isVariableUsedInExpression(varName, data.Operand)
	case *HIRCallExpression:
		if validator.isVariableUsedInExpression(varName, data.Function) {
			return true
		}
		for _, arg := range data.Arguments {
			if validator.isVariableUsedInExpression(varName, arg) {
				return true
			}
		}
	case *HIRBlock:
		return validator.isVariableUsedInBlock(varName, data)
	}

	return false
}

// validateBlock validates a block
func (validator *HIRValidator) validateBlock(block *HIRBlock) {
	// Validate all statements
	for _, stmt := range block.Statements {
		validator.validateStatement(stmt)
	}

	// Validate expression if present
	if block.Expression != nil {
		validator.validateExpression(block.Expression)
	}
}

// validateStatement validates a statement
func (validator *HIRValidator) validateStatement(stmt *HIRStatement) {
	switch data := stmt.Data.(type) {
	case *HIRExpression:
		validator.validateExpression(data)
	case *HIRIfStatement:
		validator.validateExpression(data.Condition)
		validator.validateBlock(data.ThenBlock)
		if data.ElseBlock != nil {
			validator.validateBlock(data.ElseBlock)
		}
	case *HIRWhileStatement:
		validator.validateExpression(data.Condition)
		validator.validateBlock(data.Body)
	case *HIRReturnStatement:
		if data.Value != nil {
			validator.validateExpression(data.Value)
		}
	case *HIRAssignStatement:
		validator.validateExpression(data.Target)
		validator.validateExpression(data.Value)
	case *HIRLetStatement:
		if data.Initializer != nil {
			validator.validateExpression(data.Initializer)
		}
		validator.validateVariable(data.Variable)
	}
}

// validateExpression validates an expression
func (validator *HIRValidator) validateExpression(expr *HIRExpression) {
	switch data := expr.Data.(type) {
	case *HIRVariableExpression:
		// Check if variable is resolved
		if data.Variable == nil {
			validator.addError(expr.Span, fmt.Sprintf("unresolved variable: %s", data.Name), ErrorUnresolvedSymbol)
		}
	case *HIRBinaryExpression:
		validator.validateExpression(data.Left)
		validator.validateExpression(data.Right)
		// TODO: Add type compatibility checking
	case *HIRUnaryExpression:
		validator.validateExpression(data.Operand)
	case *HIRCallExpression:
		validator.validateExpression(data.Function)
		for _, arg := range data.Arguments {
			validator.validateExpression(arg)
		}
	case *HIRBlock:
		validator.validateBlock(data)
	}
}

// validateVariable validates a variable
func (validator *HIRValidator) validateVariable(variable *HIRVariable) {
	// Check if type is valid
	if variable.Type != nil {
		validator.validateType(variable.Type)
	}

	// Check if initializer is present for immutable variables without type
	if !variable.IsMutable && variable.Type.Kind == HIRTypeGeneric && variable.Initializer == nil {
		validator.addError(variable.Span, fmt.Sprintf("variable %s needs explicit type or initializer", variable.Name), ErrorInvalidTypeUsage)
	}
}

// validateType validates a type
func (validator *HIRValidator) validateType(hirType *HIRType) {
	switch hirType.Kind {
	case HIRTypePrimitive:
		// Primitive types are always valid
	case HIRTypeStruct:
		// Check if struct type is defined
		if data, ok := hirType.Data.(*HIRTypeDefinition); ok {
			if _, exists := validator.context.typeTable[data.Name]; !exists {
				validator.addError(hirType.Span, fmt.Sprintf("undefined type: %s", data.Name), ErrorUnresolvedSymbol)
			}
		}
	case HIRTypeGeneric:
		// Generic types are placeholders, validation depends on context
	default:
		// Other types would need specific validation logic
	}
}

// buildControlFlowGraph constructs a control flow graph for a function
func (validator *HIRValidator) buildControlFlowGraph(function *HIRFunction) {
	// Reset control flow graph
	validator.context.controlFlow.nodes = make([]CFGNode, 0)
	validator.context.controlFlow.edges = make([]CFGEdge, 0)

	// Add entry node
	entryNode := CFGNode{
		id:   0,
		kind: CFGNodeEntry,
		span: function.Span,
		data: function,
	}
	validator.context.controlFlow.nodes = append(validator.context.controlFlow.nodes, entryNode)

	// Build CFG for function body
	if function.Body != nil {
		validator.buildCFGForBlock(function.Body, 0)
	}
}

// buildCFGForBlock builds CFG nodes for a block
func (validator *HIRValidator) buildCFGForBlock(block *HIRBlock, entryNodeID int) int {
	currentNodeID := entryNodeID

	for _, stmt := range block.Statements {
		nextNodeID := len(validator.context.controlFlow.nodes)

		stmtNode := CFGNode{
			id:   nextNodeID,
			kind: CFGNodeStatement,
			span: stmt.Span,
			data: stmt,
		}
		validator.context.controlFlow.nodes = append(validator.context.controlFlow.nodes, stmtNode)

		// Add edge from current to statement
		if currentNodeID >= 0 {
			edge := CFGEdge{from: currentNodeID, to: nextNodeID}
			validator.context.controlFlow.edges = append(validator.context.controlFlow.edges, edge)
		}

		currentNodeID = nextNodeID

		// Handle control flow statements
		if stmt.Kind == HIRStmtReturn {
			currentNodeID = -1 // No continuation after return
		}
	}

	return currentNodeID
}

// validateControlFlow validates control flow properties
func (validator *HIRValidator) validateControlFlow(function *HIRFunction) {
	// Check for unreachable code
	reachable := validator.findReachableNodes()

	for i, node := range validator.context.controlFlow.nodes {
		if !reachable[i] && node.kind == CFGNodeStatement {
			validator.addWarning(node.span, "unreachable code", WarningDeadCode)
		}
	}
}

// findReachableNodes finds all reachable nodes in the CFG
func (validator *HIRValidator) findReachableNodes() map[int]bool {
	reachable := make(map[int]bool)

	// Start from entry node
	var dfs func(int)
	dfs = func(nodeID int) {
		if reachable[nodeID] {
			return
		}
		reachable[nodeID] = true

		// Visit all successors
		for _, edge := range validator.context.controlFlow.edges {
			if edge.from == nodeID {
				dfs(edge.to)
			}
		}
	}

	if len(validator.context.controlFlow.nodes) > 0 {
		dfs(0) // Start from entry node
	}

	return reachable
}

// checkUnusedSymbols checks for unused symbols
func (validator *HIRValidator) checkUnusedSymbols() {
	// Check unused functions
	for _, function := range validator.context.currentModule.Functions {
		if function.Name != "main" && !function.IsPublic {
			// Simple heuristic - in real implementation, would track usage
			validator.addWarning(function.Span, fmt.Sprintf("unused function: %s", function.Name), WarningUnusedFunction)
		}
	}

	// Check unused global variables
	for _, variable := range validator.context.currentModule.Variables {
		if !variable.IsGlobal {
			continue
		}
		// Simple heuristic - in real implementation, would track usage
		validator.addWarning(variable.Span, fmt.Sprintf("unused variable: %s", variable.Name), WarningUnusedVariable)
	}
}

// ====== Error and Warning Management ======

// addError adds a validation error
func (validator *HIRValidator) addError(span Span, message string, code ErrorCode) {
	validator.errors = append(validator.errors, HIRValidationError{
		Span:    span,
		Message: message,
		Code:    code,
	})
}

// addWarning adds a validation warning
func (validator *HIRValidator) addWarning(span Span, message string, code WarningCode) {
	validator.warnings = append(validator.warnings, HIRValidationWarning{
		Span:    span,
		Message: message,
		Code:    code,
	})
}

// GetErrors returns all validation errors
func (validator *HIRValidator) GetErrors() []HIRValidationError {
	return validator.errors
}

// GetWarnings returns all validation warnings
func (validator *HIRValidator) GetWarnings() []HIRValidationWarning {
	return validator.warnings
}

// ====== Error String Methods ======

func (e HIRValidationError) String() string {
	return fmt.Sprintf("Error at %s: %s", e.Span.String(), e.Message)
}

func (w HIRValidationWarning) String() string {
	return fmt.Sprintf("Warning at %s: %s", w.Span.String(), w.Message)
}

// ====== Public API ======

// ValidateHIR validates an HIR module and returns errors and warnings
func ValidateHIR(module *HIRModule) ([]HIRValidationError, []HIRValidationWarning) {
	validator := NewHIRValidator()
	errors := validator.ValidateModule(module)
	warnings := validator.GetWarnings()
	return errors, warnings
}

// FormatValidationResults formats validation results as a readable string
func FormatValidationResults(errors []HIRValidationError, warnings []HIRValidationWarning) string {
	var sb strings.Builder

	if len(errors) > 0 {
		sb.WriteString(fmt.Sprintf("Validation Errors (%d):\n", len(errors)))
		for _, err := range errors {
			sb.WriteString(fmt.Sprintf("  %s\n", err.String()))
		}
		sb.WriteString("\n")
	}

	if len(warnings) > 0 {
		sb.WriteString(fmt.Sprintf("Validation Warnings (%d):\n", len(warnings)))
		for _, warn := range warnings {
			sb.WriteString(fmt.Sprintf("  %s\n", warn.String()))
		}
		sb.WriteString("\n")
	}

	if len(errors) == 0 && len(warnings) == 0 {
		sb.WriteString("✅ HIR validation passed with no errors or warnings\n")
	}

	return sb.String()
}
