// Type-safe AST extensions for Orizon language.
// This file provides enhanced type safety, visitor pattern improvements,.
// and AST transformation infrastructure for the Orizon compiler.

package parser

import (
	"fmt"
	"reflect"
	"strings"
)

// ====== Enhanced Type Safety ======.

// NodeKind represents the kind of AST node for type-safe operations.
type NodeKind int

const (
	// Declaration kinds.
	NodeKindProgram NodeKind = iota
	NodeKindFunctionDeclaration
	NodeKindVariableDeclaration
	NodeKindParameter
	NodeKindMacroDefinition
	NodeKindStructDeclaration
	NodeKindEnumDeclaration
	NodeKindTraitDeclaration
	NodeKindImplBlock
	NodeKindImportDeclaration
	NodeKindExportDeclaration

	// Statement kinds.
	NodeKindBlockStatement
	NodeKindExpressionStatement
	NodeKindReturnStatement
	NodeKindIfStatement
	NodeKindWhileStatement
	NodeKindForStatement
	NodeKindBreakStatement
	NodeKindContinueStatement

	// Expression kinds.
	NodeKindIdentifier
	NodeKindLiteral
	NodeKindBinaryExpression
	NodeKindUnaryExpression
	NodeKindCallExpression
	NodeKindAssignmentExpression
	NodeKindTernaryExpression
	NodeKindMacroInvocation
	NodeKindArrayExpression
	NodeKindIndexExpression
	NodeKindMemberExpression

	// Type kinds.
	NodeKindBasicType
	NodeKindArrayType
	NodeKindFunctionType
	NodeKindStructType
	NodeKindEnumType
	NodeKindTraitType
	NodeKindGenericType
	NodeKindReferenceType
	NodeKindPointerType

	// Macro-specific kinds.
	NodeKindMacroParameter
	NodeKindMacroBody
	NodeKindMacroTemplate
	NodeKindMacroPattern
	NodeKindMacroPatternElement
	NodeKindMacroArgument
	NodeKindMacroContext
	// Dependent type system node kinds.
	NodeKindDependentFunctionType
	NodeKindDependentParameter
	NodeKindRefinementType
	NodeKindSizedArrayType
	NodeKindIndexType
	NodeKindProofType
)

// String returns the string representation of the node kind.
func (nk NodeKind) String() string {
	switch nk {
	case NodeKindProgram:
		return "Program"
	case NodeKindFunctionDeclaration:
		return "FunctionDeclaration"
	case NodeKindVariableDeclaration:
		return "VariableDeclaration"
	case NodeKindParameter:
		return "Parameter"
	case NodeKindMacroDefinition:
		return "MacroDefinition"
	case NodeKindStructDeclaration:
		return "StructDeclaration"
	case NodeKindEnumDeclaration:
		return "EnumDeclaration"
	case NodeKindTraitDeclaration:
		return "TraitDeclaration"
	case NodeKindImplBlock:
		return "ImplBlock"
	case NodeKindImportDeclaration:
		return "ImportDeclaration"
	case NodeKindExportDeclaration:
		return "ExportDeclaration"
	case NodeKindBlockStatement:
		return "BlockStatement"
	case NodeKindExpressionStatement:
		return "ExpressionStatement"
	case NodeKindReturnStatement:
		return "ReturnStatement"
	case NodeKindIfStatement:
		return "IfStatement"
	case NodeKindWhileStatement:
		return "WhileStatement"
	case NodeKindForStatement:
		return "ForStatement"
	case NodeKindBreakStatement:
		return "BreakStatement"
	case NodeKindContinueStatement:
		return "ContinueStatement"
	case NodeKindIdentifier:
		return "Identifier"
	case NodeKindLiteral:
		return "Literal"
	case NodeKindBinaryExpression:
		return "BinaryExpression"
	case NodeKindUnaryExpression:
		return "UnaryExpression"
	case NodeKindCallExpression:
		return "CallExpression"
	case NodeKindAssignmentExpression:
		return "AssignmentExpression"
	case NodeKindTernaryExpression:
		return "TernaryExpression"
	case NodeKindMacroInvocation:
		return "MacroInvocation"
	case NodeKindArrayExpression:
		return "ArrayExpression"
	case NodeKindIndexExpression:
		return "IndexExpression"
	case NodeKindMemberExpression:
		return "MemberExpression"
	case NodeKindBasicType:
		return "BasicType"
	case NodeKindArrayType:
		return "ArrayType"
	case NodeKindFunctionType:
		return "FunctionType"
	case NodeKindStructType:
		return "StructType"
	case NodeKindEnumType:
		return "EnumType"
	case NodeKindTraitType:
		return "TraitType"
	case NodeKindGenericType:
		return "GenericType"
	case NodeKindReferenceType:
		return "ReferenceType"
	case NodeKindPointerType:
		return "PointerType"
	case NodeKindMacroParameter:
		return "MacroParameter"
	case NodeKindMacroBody:
		return "MacroBody"
	case NodeKindMacroTemplate:
		return "MacroTemplate"
	case NodeKindMacroPattern:
		return "MacroPattern"
	case NodeKindMacroPatternElement:
		return "MacroPatternElement"
	case NodeKindMacroArgument:
		return "MacroArgument"
	case NodeKindMacroContext:
		return "MacroContext"
	case NodeKindDependentFunctionType:
		return "DependentFunctionType"
	case NodeKindDependentParameter:
		return "DependentParameter"
	case NodeKindRefinementType:
		return "RefinementType"
	case NodeKindSizedArrayType:
		return "SizedArrayType"
	case NodeKindIndexType:
		return "IndexType"
	case NodeKindProofType:
		return "ProofType"
	default:
		return fmt.Sprintf("Unknown(%d)", int(nk))
	}
}

// TypeSafeNode extends the Node interface with type safety features.
type TypeSafeNode interface {
	Node
	// GetNodeKind returns the specific kind of this node.
	GetNodeKind() NodeKind
	// Clone creates a deep copy of this node.
	Clone() TypeSafeNode
	// Equals checks structural equality with another node.
	Equals(other TypeSafeNode) bool
	// GetChildren returns all child nodes.
	GetChildren() []TypeSafeNode
	// ReplaceChild replaces a child node at the given index.
	ReplaceChild(index int, newChild TypeSafeNode) error
}

// ====== Enhanced Visitor Pattern ======.

// TypedVisitor provides type-safe visitor methods with generic return types.
type TypedVisitor[T any] interface {
	VisitProgram(*Program) T
	VisitFunctionDeclaration(*FunctionDeclaration) T
	VisitParameter(*Parameter) T
	VisitVariableDeclaration(*VariableDeclaration) T
	VisitBlockStatement(*BlockStatement) T
	VisitExpressionStatement(*ExpressionStatement) T
	VisitReturnStatement(*ReturnStatement) T
	VisitIfStatement(*IfStatement) T
	VisitWhileStatement(*WhileStatement) T
	VisitIdentifier(*Identifier) T
	VisitLiteral(*Literal) T
	VisitBinaryExpression(*BinaryExpression) T
	VisitUnaryExpression(*UnaryExpression) T
	VisitCallExpression(*CallExpression) T
	VisitAssignmentExpression(*AssignmentExpression) T
	VisitTernaryExpression(*TernaryExpression) T
	VisitBasicType(*BasicType) T
	VisitMacroDefinition(*MacroDefinition) T
	VisitMacroParameter(*MacroParameter) T
	VisitMacroBody(*MacroBody) T
	VisitMacroTemplate(*MacroTemplate) T
	VisitMacroPattern(*MacroPattern) T
	VisitMacroPatternElement(*MacroPatternElement) T
	VisitMacroInvocation(*MacroInvocation) T
	VisitMacroArgument(*MacroArgument) T
	VisitMacroContext(*MacroContext) T
	// Additional type nodes.
	VisitArrayType(*ArrayType) T
	VisitFunctionType(*FunctionType) T
	VisitStructType(*StructType) T
	VisitEnumType(*EnumType) T
	VisitTraitType(*TraitType) T
	VisitGenericType(*GenericType) T
	VisitReferenceType(*ReferenceType) T
	VisitPointerType(*PointerType) T
	// Additional expression nodes.
	VisitArrayExpression(*ArrayExpression) T
	VisitIndexExpression(*IndexExpression) T
	VisitMemberExpression(*MemberExpression) T
	// Additional statement nodes.
	VisitForStatement(*ForStatement) T
	VisitBreakStatement(*BreakStatement) T
	VisitContinueStatement(*ContinueStatement) T
}

// WalkVisitor provides a default implementation for tree walking.
type WalkVisitor struct {
	PreVisit  func(Node) bool // Return false to skip subtree
	PostVisit func(Node)      // Called after visiting subtree
	OnError   func(error)     // Called when an error occurs
}

// Walk traverses the AST using the visitor.
func (w *WalkVisitor) Walk(node Node) {
	if w.PreVisit != nil && !w.PreVisit(node) {
		return
	}

	// Visit children based on node type.
	switch n := node.(type) {
	case *Program:
		for _, decl := range n.Declarations {
			w.Walk(decl)
		}
	case *FunctionDeclaration:
		for _, param := range n.Parameters {
			w.Walk(param)
		}

		if n.ReturnType != nil {
			w.Walk(n.ReturnType)
		}

		if n.Body != nil {
			w.Walk(n.Body)
		}
	case *BlockStatement:
		for _, stmt := range n.Statements {
			w.Walk(stmt)
		}
	case *BinaryExpression:
		w.Walk(n.Left)
		w.Walk(n.Right)
	case *UnaryExpression:
		w.Walk(n.Operand)
	case *CallExpression:
		w.Walk(n.Function)

		for _, arg := range n.Arguments {
			w.Walk(arg)
		}
	case *AssignmentExpression:
		w.Walk(n.Left)
		w.Walk(n.Right)
	case *TernaryExpression:
		w.Walk(n.Condition)
		w.Walk(n.TrueExpr)
		w.Walk(n.FalseExpr)
	case *IfStatement:
		w.Walk(n.Condition)
		w.Walk(n.ThenStmt)

		if n.ElseStmt != nil {
			w.Walk(n.ElseStmt)
		}
	case *WhileStatement:
		w.Walk(n.Condition)
		w.Walk(n.Body)
	case *ReturnStatement:
		if n.Value != nil {
			w.Walk(n.Value)
		}
	case *ExpressionStatement:
		w.Walk(n.Expression)
	case *VariableDeclaration:
		if n.TypeSpec != nil {
			w.Walk(n.TypeSpec)
		}

		if n.Initializer != nil {
			w.Walk(n.Initializer)
		}
	case *MacroDefinition:
		for _, param := range n.Parameters {
			w.Walk(param)
		}

		if n.Body != nil {
			w.Walk(n.Body)
		}
	case *MacroInvocation:
		w.Walk(n.Name)

		for _, arg := range n.Arguments {
			w.Walk(arg)
		}
	}

	if w.PostVisit != nil {
		w.PostVisit(node)
	}
}

// ====== AST Transformation Infrastructure ======.

// Transformer provides a framework for AST transformations.
type Transformer interface {
	// Transform applies a transformation to the given node.
	Transform(node Node) (Node, error)
	// CanTransform checks if this transformer can handle the given node.
	CanTransform(node Node) bool
	// GetName returns the name of this transformer.
	GetName() string
}

// TransformationPipeline manages a sequence of transformations.
type TransformationPipeline struct {
	transformers []Transformer
	options      TransformationOptions
}

// TransformationOptions configures the transformation pipeline.
type TransformationOptions struct {
	MaxIterations  int
	SkipErrors     bool
	DebugMode      bool
	PreserveSpans  bool
	ValidateOutput bool
}

// NewTransformationPipeline creates a new transformation pipeline.
func NewTransformationPipeline(options TransformationOptions) *TransformationPipeline {
	if options.MaxIterations <= 0 {
		options.MaxIterations = 10
	}

	return &TransformationPipeline{
		transformers: make([]Transformer, 0),
		options:      options,
	}
}

// AddTransformer adds a transformer to the pipeline.
func (tp *TransformationPipeline) AddTransformer(t Transformer) {
	tp.transformers = append(tp.transformers, t)
}

// Transform applies all transformers to the AST.
func (tp *TransformationPipeline) Transform(root Node) (Node, error) {
	current := root
	iteration := 0

	for iteration < tp.options.MaxIterations {
		changed := false

		for _, transformer := range tp.transformers {
			if transformer.CanTransform(current) {
				if tp.options.DebugMode {
					fmt.Printf("Applying transformer %s (iteration %d)\n",
						transformer.GetName(), iteration)
				}

				newNode, err := transformer.Transform(current)
				if err != nil {
					if tp.options.SkipErrors {
						continue
					}

					return nil, fmt.Errorf("transformation failed with %s: %w",
						transformer.GetName(), err)
				}

				if newNode != current {
					current = newNode
					changed = true

					if tp.options.ValidateOutput {
						if err := ValidateAST(current); err != nil {
							return nil, fmt.Errorf("AST validation failed after %s: %w",
								transformer.GetName(), err)
						}
					}
				}
			}
		}

		if !changed {
			break
		}

		iteration++
	}

	if iteration >= tp.options.MaxIterations {
		return current, fmt.Errorf("transformation pipeline reached maximum iterations (%d)",
			tp.options.MaxIterations)
	}

	return current, nil
}

// ====== AST Validation ======.

// ValidationError represents an AST validation error.
type ValidationError struct {
	Node    Node
	Message string
	Span    Span
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("validation error at %s: %s", ve.Span.String(), ve.Message)
}

// Validator provides AST validation functionality.
type Validator struct {
	errors   []ValidationError
	warnings []ValidationError
	strict   bool // Strict mode treats warnings as errors
}

// NewValidator creates a new AST validator.
func NewValidator(strict bool) *Validator {
	return &Validator{
		errors:   make([]ValidationError, 0),
		warnings: make([]ValidationError, 0),
		strict:   strict,
	}
}

// ValidateAST validates an AST node and its children.
func ValidateAST(root Node) error {
	validator := NewValidator(false)

	return validator.Validate(root)
}

// Validate validates an AST node.
func (v *Validator) Validate(node Node) error {
	// Reset validation state.
	v.errors = v.errors[:0]
	v.warnings = v.warnings[:0]

	// Walk the AST and validate each node.
	walker := &WalkVisitor{
		PreVisit: func(n Node) bool {
			v.validateNode(n)

			return true
		},
		OnError: func(err error) {
			v.errors = append(v.errors, ValidationError{
				Node:    node,
				Message: err.Error(),
				Span:    node.GetSpan(),
			})
		},
	}

	walker.Walk(node)

	// Check for errors.
	if len(v.errors) > 0 {
		messages := make([]string, len(v.errors))
		for i, err := range v.errors {
			messages[i] = err.Error()
		}

		return fmt.Errorf("validation failed with %d errors:\n%s",
			len(v.errors), strings.Join(messages, "\n"))
	}

	// Check for warnings in strict mode.
	if v.strict && len(v.warnings) > 0 {
		messages := make([]string, len(v.warnings))
		for i, warn := range v.warnings {
			messages[i] = warn.Error()
		}

		return fmt.Errorf("validation failed with %d warnings (strict mode):\n%s",
			len(v.warnings), strings.Join(messages, "\n"))
	}

	return nil
}

// validateNode validates a specific node.
func (v *Validator) validateNode(node Node) {
	switch n := node.(type) {
	case *Program:
		if len(n.Declarations) == 0 {
			v.addWarning(n, "empty program")
		}
	case *FunctionDeclaration:
		if n.Name == nil {
			v.addError(n, "function declaration missing name")
		}

		if n.Body == nil {
			v.addError(n, "function declaration missing body")
		}
	case *VariableDeclaration:
		if n.Name == nil {
			v.addError(n, "variable declaration missing name")
		}

		if n.TypeSpec == nil && n.Initializer == nil {
			v.addError(n, "variable declaration must have either type or initializer")
		}
	case *BinaryExpression:
		if n.Left == nil {
			v.addError(n, "binary expression missing left operand")
		}

		if n.Right == nil {
			v.addError(n, "binary expression missing right operand")
		}

		if n.Operator == nil || n.Operator.Value == "" {
			v.addError(n, "binary expression missing operator")
		}
	case *UnaryExpression:
		if n.Operand == nil {
			v.addError(n, "unary expression missing operand")
		}

		if n.Operator == nil || n.Operator.Value == "" {
			v.addError(n, "unary expression missing operator")
		}
	case *CallExpression:
		if n.Function == nil {
			v.addError(n, "call expression missing function")
		}
	case *Identifier:
		if n.Value == "" {
			v.addError(n, "identifier has empty value")
		}
	case *MacroDefinition:
		if n.Name == nil {
			v.addError(n, "macro definition missing name")
		}

		if n.Body == nil {
			v.addError(n, "macro definition missing body")
		}
	case *MacroInvocation:
		if n.Name == nil {
			v.addError(n, "macro invocation missing name")
		}
	}
}

// addError adds a validation error.
func (v *Validator) addError(node Node, message string) {
	v.errors = append(v.errors, ValidationError{
		Node:    node,
		Message: message,
		Span:    node.GetSpan(),
	})
}

// addWarning adds a validation warning.
func (v *Validator) addWarning(node Node, message string) {
	v.warnings = append(v.warnings, ValidationError{
		Node:    node,
		Message: message,
		Span:    node.GetSpan(),
	})
}

// ====== AST Utility Functions ======.

// CollectValidationReports walks the AST and returns validation errors and warnings without returning an error.
// This is designed for tooling (e.g., LSP diagnostics) to surface all issues in one pass.
func CollectValidationReports(root Node) ([]ValidationError, []ValidationError) {
	v := NewValidator(false)
	v.errors = v.errors[:0]
	v.warnings = v.warnings[:0]

	walker := &WalkVisitor{
		PreVisit: func(n Node) bool {
			v.validateNode(n)

			return true
		},
		OnError: func(err error) {
			v.errors = append(v.errors, ValidationError{
				Node:    root,
				Message: err.Error(),
				Span:    root.GetSpan(),
			})
		},
	}

	walker.Walk(root)

	errs := make([]ValidationError, len(v.errors))
	copy(errs, v.errors)
	warns := make([]ValidationError, len(v.warnings))
	copy(warns, v.warnings)

	return errs, warns
}

// GetNodeType returns the reflect.Type of an AST node.
func GetNodeType(node Node) reflect.Type {
	return reflect.TypeOf(node)
}

// IsDeclaration checks if a node is a declaration.
func IsDeclaration(node Node) bool {
	_, ok := node.(Declaration)

	return ok
}

// IsStatement checks if a node is a statement.
func IsStatement(node Node) bool {
	_, ok := node.(Statement)

	return ok
}

// IsExpression checks if a node is an expression.
func IsExpression(node Node) bool {
	_, ok := node.(Expression)

	return ok
}

// IsType checks if a node is a type.
func IsType(node Node) bool {
	_, ok := node.(Type)

	return ok
}

// FindNodesByType finds all nodes of a specific type in the AST.
func FindNodesByType[T Node](root Node, nodeType reflect.Type) []T {
	var results []T

	walker := &WalkVisitor{
		PreVisit: func(n Node) bool {
			if reflect.TypeOf(n) == nodeType {
				if typed, ok := n.(T); ok {
					results = append(results, typed)
				}
			}

			return true
		},
	}

	walker.Walk(root)

	return results
}

// CountNodes counts the total number of nodes in the AST.
func CountNodes(root Node) int {
	count := 0
	walker := &WalkVisitor{
		PreVisit: func(n Node) bool {
			count++

			return true
		},
	}
	walker.Walk(root)

	return count
}

// GetDepth calculates the maximum depth of the AST.
func GetDepth(root Node) int {
	maxDepth := 0
	currentDepth := 0

	walker := &WalkVisitor{
		PreVisit: func(n Node) bool {
			currentDepth++
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}

			return true
		},
		PostVisit: func(n Node) {
			currentDepth--
		},
	}

	walker.Walk(root)

	return maxDepth
}
