// Package ast - Visitor pattern implementation for AST traversal
// Phase 1.3.1: Visitor パターン実装 - Comprehensive visitor pattern for type-safe AST traversal
// This file implements the visitor pattern to enable safe and extensible AST transformations,
// analysis passes, and code generation while maintaining strong typing guarantees.
package ast

import (
	"fmt"
	"strings"
)

// Visitor defines the interface for AST node visitors
// This pattern allows for type-safe traversal and transformation of the AST
// without modifying the AST node types themselves, following the open/closed principle
type Visitor interface {
	// Program and structure visitors
	VisitProgram(node *Program) interface{}
	VisitComment(node *Comment) interface{}

	// Declaration visitors
	VisitFunctionDeclaration(node *FunctionDeclaration) interface{}
	VisitParameter(node *Parameter) interface{}
	VisitVariableDeclaration(node *VariableDeclaration) interface{}
	VisitTypeDeclaration(node *TypeDeclaration) interface{}
	VisitImportDeclaration(node *ImportDeclaration) interface{}
	VisitExportDeclaration(node *ExportDeclaration) interface{}
	VisitExportItem(node *ExportItem) interface{}

	// Statement visitors
	VisitBlockStatement(node *BlockStatement) interface{}
	VisitExpressionStatement(node *ExpressionStatement) interface{}
	VisitReturnStatement(node *ReturnStatement) interface{}
	VisitIfStatement(node *IfStatement) interface{}
	VisitWhileStatement(node *WhileStatement) interface{}

	// Expression visitors
	VisitIdentifier(node *Identifier) interface{}
	VisitLiteral(node *Literal) interface{}
	VisitBinaryExpression(node *BinaryExpression) interface{}
	VisitUnaryExpression(node *UnaryExpression) interface{}
	VisitCallExpression(node *CallExpression) interface{}

	// Type visitors
	VisitBasicType(node *BasicType) interface{}
	VisitIdentifierType(node *IdentifierType) interface{}

	// Attribute visitors
	VisitAttribute(node *Attribute) interface{}
}

// BaseVisitor provides a default implementation of the Visitor interface
// that simply returns nil for all visits. This allows concrete visitors
// to only override the methods they need, following the composition pattern.
type BaseVisitor struct{}

func (v *BaseVisitor) VisitProgram(node *Program) interface{}                         { return nil }
func (v *BaseVisitor) VisitComment(node *Comment) interface{}                         { return nil }
func (v *BaseVisitor) VisitFunctionDeclaration(node *FunctionDeclaration) interface{} { return nil }
func (v *BaseVisitor) VisitParameter(node *Parameter) interface{}                     { return nil }
func (v *BaseVisitor) VisitVariableDeclaration(node *VariableDeclaration) interface{} { return nil }
func (v *BaseVisitor) VisitTypeDeclaration(node *TypeDeclaration) interface{}         { return nil }
func (v *BaseVisitor) VisitImportDeclaration(node *ImportDeclaration) interface{}     { return nil }
func (v *BaseVisitor) VisitExportDeclaration(node *ExportDeclaration) interface{}     { return nil }
func (v *BaseVisitor) VisitExportItem(node *ExportItem) interface{}                   { return nil }
func (v *BaseVisitor) VisitBlockStatement(node *BlockStatement) interface{}           { return nil }
func (v *BaseVisitor) VisitExpressionStatement(node *ExpressionStatement) interface{} { return nil }
func (v *BaseVisitor) VisitReturnStatement(node *ReturnStatement) interface{}         { return nil }
func (v *BaseVisitor) VisitIfStatement(node *IfStatement) interface{}                 { return nil }
func (v *BaseVisitor) VisitWhileStatement(node *WhileStatement) interface{}           { return nil }
func (v *BaseVisitor) VisitIdentifier(node *Identifier) interface{}                   { return nil }
func (v *BaseVisitor) VisitLiteral(node *Literal) interface{}                         { return nil }
func (v *BaseVisitor) VisitBinaryExpression(node *BinaryExpression) interface{}       { return nil }
func (v *BaseVisitor) VisitUnaryExpression(node *UnaryExpression) interface{}         { return nil }
func (v *BaseVisitor) VisitCallExpression(node *CallExpression) interface{}           { return nil }
func (v *BaseVisitor) VisitBasicType(node *BasicType) interface{}                     { return nil }
func (v *BaseVisitor) VisitIdentifierType(node *IdentifierType) interface{}           { return nil }
func (v *BaseVisitor) VisitAttribute(node *Attribute) interface{}                     { return nil }

// WalkingVisitor provides a recursive visitor that automatically traverses
// the entire AST tree structure. Concrete visitors can embed this to get
// automatic tree walking behavior and only override specific node types.
type WalkingVisitor struct {
	BaseVisitor
	visitor Visitor // The actual visitor to delegate to
}

// NewWalkingVisitor creates a new walking visitor that delegates to the provided visitor
func NewWalkingVisitor(visitor Visitor) *WalkingVisitor {
	return &WalkingVisitor{visitor: visitor}
}

// Walk traverses the AST starting from the given node
func (w *WalkingVisitor) Walk(node Node) interface{} {
	if node == nil {
		return nil
	}
	return node.Accept(w)
}

// VisitProgram walks through all declarations in the program
func (w *WalkingVisitor) VisitProgram(node *Program) interface{} {
	result := w.visitor.VisitProgram(node)

	// Walk through all declarations
	for _, decl := range node.Declarations {
		if decl != nil {
			decl.Accept(w)
		}
	}

	// Walk through comments
	for _, comment := range node.Comments {
		comment.Accept(w)
	}

	return result
}

// VisitFunctionDeclaration walks through function components
func (w *WalkingVisitor) VisitFunctionDeclaration(node *FunctionDeclaration) interface{} {
	result := w.visitor.VisitFunctionDeclaration(node)

	// Walk function name
	if node.Name != nil {
		node.Name.Accept(w)
	}

	// Walk parameters
	for _, param := range node.Parameters {
		if param != nil {
			param.Accept(w)
		}
	}

	// Walk return type
	if node.ReturnType != nil {
		node.ReturnType.Accept(w)
	}

	// Walk function body
	if node.Body != nil {
		node.Body.Accept(w)
	}

	// Walk attributes
	for _, attr := range node.Attributes {
		attr.Accept(w)
	}

	return result
}

// VisitParameter walks through parameter components
func (w *WalkingVisitor) VisitParameter(node *Parameter) interface{} {
	result := w.visitor.VisitParameter(node)

	// Walk parameter name
	if node.Name != nil {
		node.Name.Accept(w)
	}

	// Walk parameter type
	if node.Type != nil {
		node.Type.Accept(w)
	}

	// Walk default value
	if node.DefaultValue != nil {
		node.DefaultValue.Accept(w)
	}

	return result
}

// VisitVariableDeclaration walks through variable declaration components
func (w *WalkingVisitor) VisitVariableDeclaration(node *VariableDeclaration) interface{} {
	result := w.visitor.VisitVariableDeclaration(node)

	// Walk variable name
	if node.Name != nil {
		node.Name.Accept(w)
	}

	// Walk variable type
	if node.Type != nil {
		node.Type.Accept(w)
	}

	// Walk initial value
	if node.Value != nil {
		node.Value.Accept(w)
	}

	return result
}

// VisitTypeDeclaration walks through type declaration components
func (w *WalkingVisitor) VisitTypeDeclaration(node *TypeDeclaration) interface{} {
	result := w.visitor.VisitTypeDeclaration(node)

	// Walk type name
	if node.Name != nil {
		node.Name.Accept(w)
	}

	// Walk underlying type
	if node.Type != nil {
		node.Type.Accept(w)
	}

	return result
}

// VisitImportDeclaration walks through import components
func (w *WalkingVisitor) VisitImportDeclaration(node *ImportDeclaration) interface{} {
	result := w.visitor.VisitImportDeclaration(node)
	for _, seg := range node.Path {
		if seg != nil {
			seg.Accept(w)
		}
	}
	if node.Alias != nil {
		node.Alias.Accept(w)
	}
	return result
}

// VisitExportDeclaration walks through export items
func (w *WalkingVisitor) VisitExportDeclaration(node *ExportDeclaration) interface{} {
	result := w.visitor.VisitExportDeclaration(node)
	for _, it := range node.Items {
		if it != nil {
			it.Accept(w)
		}
	}
	return result
}

// VisitExportItem walks through export item parts
func (w *WalkingVisitor) VisitExportItem(node *ExportItem) interface{} {
	result := w.visitor.VisitExportItem(node)
	if node.Name != nil {
		node.Name.Accept(w)
	}
	if node.Alias != nil {
		node.Alias.Accept(w)
	}
	return result
}

// VisitBlockStatement walks through all statements in the block
func (w *WalkingVisitor) VisitBlockStatement(node *BlockStatement) interface{} {
	result := w.visitor.VisitBlockStatement(node)

	// Walk all statements
	for _, stmt := range node.Statements {
		if stmt != nil {
			stmt.Accept(w)
		}
	}

	return result
}

// VisitExpressionStatement walks through the expression
func (w *WalkingVisitor) VisitExpressionStatement(node *ExpressionStatement) interface{} {
	result := w.visitor.VisitExpressionStatement(node)

	// Walk the expression
	if node.Expression != nil {
		node.Expression.Accept(w)
	}

	return result
}

// VisitReturnStatement walks through the return value
func (w *WalkingVisitor) VisitReturnStatement(node *ReturnStatement) interface{} {
	result := w.visitor.VisitReturnStatement(node)

	// Walk return value
	if node.Value != nil {
		node.Value.Accept(w)
	}

	return result
}

// VisitIfStatement walks through if statement components
func (w *WalkingVisitor) VisitIfStatement(node *IfStatement) interface{} {
	result := w.visitor.VisitIfStatement(node)

	// Walk condition
	if node.Condition != nil {
		node.Condition.Accept(w)
	}

	// Walk then block
	if node.ThenBlock != nil {
		node.ThenBlock.Accept(w)
	}

	// Walk else block
	if node.ElseBlock != nil {
		node.ElseBlock.Accept(w)
	}

	return result
}

// VisitWhileStatement walks through while statement components
func (w *WalkingVisitor) VisitWhileStatement(node *WhileStatement) interface{} {
	result := w.visitor.VisitWhileStatement(node)

	// Walk condition
	if node.Condition != nil {
		node.Condition.Accept(w)
	}

	// Walk body
	if node.Body != nil {
		node.Body.Accept(w)
	}

	return result
}

// VisitBinaryExpression walks through binary expression operands
func (w *WalkingVisitor) VisitBinaryExpression(node *BinaryExpression) interface{} {
	result := w.visitor.VisitBinaryExpression(node)

	// Walk left operand
	if node.Left != nil {
		node.Left.Accept(w)
	}

	// Walk right operand
	if node.Right != nil {
		node.Right.Accept(w)
	}

	return result
}

// VisitUnaryExpression walks through unary expression operand
func (w *WalkingVisitor) VisitUnaryExpression(node *UnaryExpression) interface{} {
	result := w.visitor.VisitUnaryExpression(node)

	// Walk operand
	if node.Operand != nil {
		node.Operand.Accept(w)
	}

	return result
}

// VisitCallExpression walks through call expression components
func (w *WalkingVisitor) VisitCallExpression(node *CallExpression) interface{} {
	result := w.visitor.VisitCallExpression(node)

	// Walk function expression
	if node.Function != nil {
		node.Function.Accept(w)
	}

	// Walk arguments
	for _, arg := range node.Arguments {
		if arg != nil {
			arg.Accept(w)
		}
	}

	return result
}

// VisitIdentifierType walks through identifier type name
func (w *WalkingVisitor) VisitIdentifierType(node *IdentifierType) interface{} {
	result := w.visitor.VisitIdentifierType(node)

	// Walk type name
	if node.Name != nil {
		node.Name.Accept(w)
	}

	return result
}

// VisitAttribute walks through attribute components
func (w *WalkingVisitor) VisitAttribute(node *Attribute) interface{} {
	result := w.visitor.VisitAttribute(node)

	// Walk attribute name
	if node.Name != nil {
		node.Name.Accept(w)
	}

	// Walk attribute arguments
	for _, arg := range node.Args {
		if arg != nil {
			arg.Accept(w)
		}
	}

	return result
}

// For leaf nodes, delegate directly to the visitor
func (w *WalkingVisitor) VisitComment(node *Comment) interface{} { return w.visitor.VisitComment(node) }
func (w *WalkingVisitor) VisitIdentifier(node *Identifier) interface{} {
	return w.visitor.VisitIdentifier(node)
}
func (w *WalkingVisitor) VisitLiteral(node *Literal) interface{} { return w.visitor.VisitLiteral(node) }
func (w *WalkingVisitor) VisitBasicType(node *BasicType) interface{} {
	return w.visitor.VisitBasicType(node)
}

// TransformingVisitor provides a visitor that can transform AST nodes
// It returns new nodes instead of modifying existing ones, ensuring immutability
type TransformingVisitor struct {
	BaseVisitor
}

// Transform applies transformations to an AST node and returns the transformed node
func (t *TransformingVisitor) Transform(node Node) Node {
	if node == nil {
		return nil
	}

	result := node.Accept(t)
	if result == nil {
		return node // No transformation applied, return original
	}

	if transformedNode, ok := result.(Node); ok {
		return transformedNode
	}

	return node // Invalid transformation result, return original
}

// Example concrete visitor implementations

// PrettyPrintVisitor creates a formatted string representation of the AST
type PrettyPrintVisitor struct {
	BaseVisitor
	indent int
}

// NewPrettyPrintVisitor creates a new pretty print visitor
func NewPrettyPrintVisitor() *PrettyPrintVisitor {
	return &PrettyPrintVisitor{indent: 0}
}

// PrettyPrint returns a formatted string representation of the AST
func (p *PrettyPrintVisitor) PrettyPrint(node Node) string {
	if node == nil {
		return "<nil>"
	}

	result := node.Accept(p)
	if str, ok := result.(string); ok {
		return str
	}
	return node.String()
}

func (p *PrettyPrintVisitor) getIndent() string {
	return strings.Repeat("  ", p.indent)
}

func (p *PrettyPrintVisitor) VisitProgram(node *Program) interface{} {
	result := "Program\n"
	p.indent++
	for _, decl := range node.Declarations {
		if decl != nil {
			declStr := p.PrettyPrint(decl)
			result += p.getIndent() + declStr + "\n"
		}
	}
	p.indent--
	return result
}

func (p *PrettyPrintVisitor) VisitFunctionDeclaration(node *FunctionDeclaration) interface{} {
	result := fmt.Sprintf("func %s", node.Name.Value)
	if node.Body != nil {
		p.indent++
		bodyStr := p.PrettyPrint(node.Body)
		result += "\n" + p.getIndent() + bodyStr
		p.indent--
	}
	return result
}

func (p *PrettyPrintVisitor) VisitImportDeclaration(node *ImportDeclaration) interface{} {
	return node.String()
}

func (p *PrettyPrintVisitor) VisitExportDeclaration(node *ExportDeclaration) interface{} {
	return node.String()
}

func (p *PrettyPrintVisitor) VisitExportItem(node *ExportItem) interface{} { return node.String() }

func (p *PrettyPrintVisitor) VisitBlockStatement(node *BlockStatement) interface{} {
	result := "Block"
	p.indent++
	for _, stmt := range node.Statements {
		if stmt != nil {
			stmtStr := p.PrettyPrint(stmt)
			result += "\n" + p.getIndent() + stmtStr
		}
	}
	p.indent--
	return result
}

func (p *PrettyPrintVisitor) VisitExpressionStatement(node *ExpressionStatement) interface{} {
	return "ExprStmt"
}

func (p *PrettyPrintVisitor) VisitReturnStatement(node *ReturnStatement) interface{} {
	return "return"
}

func (p *PrettyPrintVisitor) VisitVariableDeclaration(node *VariableDeclaration) interface{} {
	return fmt.Sprintf("%s %s", node.Kind.String(), node.Name.Value)
}

// NodeCountVisitor counts the number of nodes in the AST
type NodeCountVisitor struct {
	BaseVisitor
	count int
}

// NewNodeCountVisitor creates a new node count visitor
func NewNodeCountVisitor() *NodeCountVisitor {
	return &NodeCountVisitor{count: 0}
}

// Count returns the total number of nodes visited
func (n *NodeCountVisitor) Count(node Node) int {
	n.count = 0
	walker := NewWalkingVisitor(n)
	walker.Walk(node)
	return n.count
}

// Override base visitor methods to increment count
func (n *NodeCountVisitor) VisitProgram(node *Program) interface{} { n.count++; return nil }
func (n *NodeCountVisitor) VisitComment(node *Comment) interface{} { n.count++; return nil }
func (n *NodeCountVisitor) VisitFunctionDeclaration(node *FunctionDeclaration) interface{} {
	n.count++
	return nil
}
func (n *NodeCountVisitor) VisitParameter(node *Parameter) interface{} { n.count++; return nil }
func (n *NodeCountVisitor) VisitVariableDeclaration(node *VariableDeclaration) interface{} {
	n.count++
	return nil
}
func (n *NodeCountVisitor) VisitTypeDeclaration(node *TypeDeclaration) interface{} {
	n.count++
	return nil
}
func (n *NodeCountVisitor) VisitImportDeclaration(node *ImportDeclaration) interface{} {
	n.count++
	return nil
}
func (n *NodeCountVisitor) VisitExportDeclaration(node *ExportDeclaration) interface{} {
	n.count++
	return nil
}
func (n *NodeCountVisitor) VisitExportItem(node *ExportItem) interface{} { n.count++; return nil }
func (n *NodeCountVisitor) VisitBlockStatement(node *BlockStatement) interface{} {
	n.count++
	return nil
}
func (n *NodeCountVisitor) VisitExpressionStatement(node *ExpressionStatement) interface{} {
	n.count++
	return nil
}
func (n *NodeCountVisitor) VisitReturnStatement(node *ReturnStatement) interface{} {
	n.count++
	return nil
}
func (n *NodeCountVisitor) VisitIfStatement(node *IfStatement) interface{} { n.count++; return nil }
func (n *NodeCountVisitor) VisitWhileStatement(node *WhileStatement) interface{} {
	n.count++
	return nil
}
func (n *NodeCountVisitor) VisitIdentifier(node *Identifier) interface{} { n.count++; return nil }
func (n *NodeCountVisitor) VisitLiteral(node *Literal) interface{}       { n.count++; return nil }
func (n *NodeCountVisitor) VisitBinaryExpression(node *BinaryExpression) interface{} {
	n.count++
	return nil
}
func (n *NodeCountVisitor) VisitUnaryExpression(node *UnaryExpression) interface{} {
	n.count++
	return nil
}
func (n *NodeCountVisitor) VisitCallExpression(node *CallExpression) interface{} {
	n.count++
	return nil
}
func (n *NodeCountVisitor) VisitBasicType(node *BasicType) interface{} { n.count++; return nil }
func (n *NodeCountVisitor) VisitIdentifierType(node *IdentifierType) interface{} {
	n.count++
	return nil
}
func (n *NodeCountVisitor) VisitAttribute(node *Attribute) interface{} { n.count++; return nil }
