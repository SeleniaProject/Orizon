// Package types provides AST node definitions for effect inference.
// This module defines the abstract syntax tree nodes used by the effect system.
package types

// ASTNode represents a generic AST node interface
type ASTNode interface {
	Accept(visitor Visitor) interface{}
	String() string
	GetLocation() *SourceLocation
}

// Visitor interface for AST traversal
type Visitor interface {
	VisitFunctionDecl(*FunctionDecl) interface{}
	VisitCallExpr(*CallExpr) interface{}
	VisitAssignmentExpr(*AssignmentExpr) interface{}
	VisitIfStmt(*IfStmt) interface{}
	VisitForStmt(*ForStmt) interface{}
	VisitBlockStmt(*BlockStmt) interface{}
	VisitReturnStmt(*ReturnStmt) interface{}
	VisitThrowStmt(*ThrowStmt) interface{}
	VisitTryStmt(*TryStmt) interface{}
	VisitExpression(*Expression) interface{}
	VisitStatement(*Statement) interface{}
}

// BaseNode provides common functionality for AST nodes
type BaseNode struct {
	Location *SourceLocation
}

// GetLocation returns the source location of the node
func (bn *BaseNode) GetLocation() *SourceLocation {
	return bn.Location
}

// FunctionDecl represents a function declaration
type FunctionDecl struct {
	BaseNode
	Name       string
	Parameters []*FunctionParameter
	ReturnType *Type
	Body       *BlockStmt
	Effects    *EffectSignature
	Modifiers  []string
}

// Accept implements the Visitor pattern
func (fd *FunctionDecl) Accept(visitor Visitor) interface{} {
	return visitor.VisitFunctionDecl(fd)
}

// String returns string representation
func (fd *FunctionDecl) String() string {
	return "function " + fd.Name
}

// FunctionParameter represents a function parameter in effect analysis
type FunctionParameter struct {
	Name string
	Type *Type
}

// CallExpr represents a function call expression
type CallExpr struct {
	BaseNode
	Function  ASTNode
	Arguments []ASTNode
}

// Accept implements the Visitor pattern
func (ce *CallExpr) Accept(visitor Visitor) interface{} {
	return visitor.VisitCallExpr(ce)
}

// String returns string representation
func (ce *CallExpr) String() string {
	return "call"
}

// AssignmentExpr represents an assignment expression
type AssignmentExpr struct {
	BaseNode
	LHS      ASTNode
	RHS      ASTNode
	Operator string
}

// Accept implements the Visitor pattern
func (ae *AssignmentExpr) Accept(visitor Visitor) interface{} {
	return visitor.VisitAssignmentExpr(ae)
}

// String returns string representation
func (ae *AssignmentExpr) String() string {
	return "assignment"
}

// IfStmt represents an if statement
type IfStmt struct {
	BaseNode
	Condition ASTNode
	Then      ASTNode
	Else      ASTNode
}

// Accept implements the Visitor pattern
func (is *IfStmt) Accept(visitor Visitor) interface{} {
	return visitor.VisitIfStmt(is)
}

// String returns string representation
func (is *IfStmt) String() string {
	return "if"
}

// ForStmt represents a for loop statement
type ForStmt struct {
	BaseNode
	Init      ASTNode
	Condition ASTNode
	Update    ASTNode
	Body      ASTNode
}

// Accept implements the Visitor pattern
func (fs *ForStmt) Accept(visitor Visitor) interface{} {
	return visitor.VisitForStmt(fs)
}

// String returns string representation
func (fs *ForStmt) String() string {
	return "for"
}

// BlockStmt represents a block statement
type BlockStmt struct {
	BaseNode
	Statements []ASTNode
}

// Accept implements the Visitor pattern
func (bs *BlockStmt) Accept(visitor Visitor) interface{} {
	return visitor.VisitBlockStmt(bs)
}

// String returns string representation
func (bs *BlockStmt) String() string {
	return "block"
}

// ReturnStmt represents a return statement
type ReturnStmt struct {
	BaseNode
	Value ASTNode
}

// Accept implements the Visitor pattern
func (rs *ReturnStmt) Accept(visitor Visitor) interface{} {
	return visitor.VisitReturnStmt(rs)
}

// String returns string representation
func (rs *ReturnStmt) String() string {
	return "return"
}

// ThrowStmt represents a throw statement
type ThrowStmt struct {
	BaseNode
	Value ASTNode
}

// Accept implements the Visitor pattern
func (ts *ThrowStmt) Accept(visitor Visitor) interface{} {
	return visitor.VisitThrowStmt(ts)
}

// String returns string representation
func (ts *ThrowStmt) String() string {
	return "throw"
}

// TryStmt represents a try-catch statement
type TryStmt struct {
	BaseNode
	Body    ASTNode
	Catches []*CatchClause
	Finally ASTNode
}

// Accept implements the Visitor pattern
func (ts *TryStmt) Accept(visitor Visitor) interface{} {
	return visitor.VisitTryStmt(ts)
}

// String returns string representation
func (ts *TryStmt) String() string {
	return "try"
}

// CatchClause represents a catch clause
type CatchClause struct {
	BaseNode
	Parameter string
	Type      *Type
	Body      ASTNode
}

// Expression represents a generic expression
type Expression struct {
	BaseNode
	Value interface{}
}

// Accept implements the Visitor pattern
func (e *Expression) Accept(visitor Visitor) interface{} {
	return visitor.VisitExpression(e)
}

// String returns string representation
func (e *Expression) String() string {
	return "expression"
}

// Statement represents a generic statement
type Statement struct {
	BaseNode
	Kind string
}

// Accept implements the Visitor pattern
func (s *Statement) Accept(visitor Visitor) interface{} {
	return visitor.VisitStatement(s)
}

// String returns string representation
func (s *Statement) String() string {
	return "statement"
}
