// Package ast defines the Abstract Syntax Tree (AST) nodes for the Orizon programming language.
// Phase 1.3.1: 型安全AST定義 - Type-safe AST implementation with visitor pattern
// This package provides strongly-typed AST nodes with comprehensive position tracking.
// and transformation infrastructure for the Orizon compiler.
//
// Integration with position package provides unified source location tracking,.
// enabling precise error reporting, debugging support, and IDE integration.
// All AST nodes implement the Node interface with visitor pattern support.
// for extensible traversal and transformation operations.
package ast

import (
	"fmt"
	"strings"

	"github.com/orizon-lang/orizon/internal/position"
)

// Node is the base interface for all AST nodes.
// Every AST node must provide position information for error reporting and debugging.
type Node interface {
	// GetSpan returns the source span covered by this node.
	GetSpan() position.Span
	// String returns a human-readable representation of the node.
	String() string
	// Accept implements the visitor pattern for AST traversal.
	Accept(visitor Visitor) interface{}
}

// Statement represents all statement nodes in the AST.
type Statement interface {
	Node
	statementNode() // Marker method to distinguish statements
}

// Expression represents all expression nodes in the AST.
type Expression interface {
	Node
	expressionNode() // Marker method to distinguish expressions
}

// Declaration represents all declaration nodes in the AST.
type Declaration interface {
	Node
	declarationNode() // Marker method to distinguish declarations
}

// Type represents all type nodes in the AST.
type Type interface {
	Node
	typeNode() // Marker method to distinguish types
}

// ===== Program Structure =====.

// Program represents the root of the AST - a complete Orizon source file.
type Program struct {
	Declarations []Declaration
	Comments     []Comment
	Span         position.Span
}

func (p *Program) GetSpan() position.Span { return p.Span }
func (p *Program) String() string {
	var parts []string
	for _, decl := range p.Declarations {
		parts = append(parts, decl.String())
	}

	return strings.Join(parts, "\n")
}
func (p *Program) Accept(visitor Visitor) interface{} { return visitor.VisitProgram(p) }

// Comment represents a comment in the source code.
type Comment struct {
	Text    string
	Span    position.Span
	IsBlock bool
}

func (c *Comment) GetSpan() position.Span { return c.Span }
func (c *Comment) String() string {
	if c.IsBlock {
		return fmt.Sprintf("/* %s */", c.Text)
	}

	return fmt.Sprintf("// %s", c.Text)
}
func (c *Comment) Accept(visitor Visitor) interface{} { return visitor.VisitComment(c) }

// ===== Declarations =====.

// FunctionDeclaration represents a function definition.
type FunctionDeclaration struct {
	ReturnType Type
	Name       *Identifier
	Body       *BlockStatement
	Parameters []*Parameter
	Attributes []Attribute
	Comments   []Comment
	Span       position.Span
	IsExported bool
}

func (f *FunctionDeclaration) GetSpan() position.Span { return f.Span }
func (f *FunctionDeclaration) declarationNode()       {}
func (f *FunctionDeclaration) String() string {
	var params []string
	for _, p := range f.Parameters {
		params = append(params, p.String())
	}

	retType := ""
	if f.ReturnType != nil {
		retType = " " + f.ReturnType.String()
	}

	exported := ""
	if f.IsExported {
		exported = "pub "
	}

	return fmt.Sprintf("%sfunc %s(%s)%s %s",
		exported, f.Name.String(), strings.Join(params, ", "), retType, f.Body.String())
}

func (f *FunctionDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitFunctionDeclaration(f)
}

// Parameter represents a function parameter.
type Parameter struct {
	Type         Type
	DefaultValue Expression
	Name         *Identifier
	Span         position.Span
	IsMutable    bool
}

func (p *Parameter) GetSpan() position.Span { return p.Span }
func (p *Parameter) String() string {
	mut := ""
	if p.IsMutable {
		mut = "mut "
	}

	def := ""
	if p.DefaultValue != nil {
		def = " = " + p.DefaultValue.String()
	}

	return fmt.Sprintf("%s%s: %s%s", mut, p.Name.String(), p.Type.String(), def)
}
func (p *Parameter) Accept(visitor Visitor) interface{} { return visitor.VisitParameter(p) }

// VariableDeclaration represents a variable declaration (let, var, const).
type VariableDeclaration struct {
	Type       Type
	Value      Expression
	Name       *Identifier
	Span       position.Span
	Kind       VarKind
	IsMutable  bool
	IsExported bool
}

// VarKind represents the kind of variable declaration.
type VarKind int

const (
	VarKindLet   VarKind = iota // Immutable by default, can be made mutable
	VarKindVar                  // Always mutable
	VarKindConst                // Always immutable, compile-time constant
)

func (vk VarKind) String() string {
	switch vk {
	case VarKindLet:
		return "let"
	case VarKindVar:
		return "var"
	case VarKindConst:
		return "const"
	default:
		return "unknown"
	}
}

func (v *VariableDeclaration) GetSpan() position.Span { return v.Span }
func (v *VariableDeclaration) declarationNode()       {}
func (v *VariableDeclaration) statementNode()         {} // VariableDeclaration can also be used as a statement
func (v *VariableDeclaration) String() string {
	mut := ""
	if v.IsMutable && v.Kind == VarKindLet {
		mut = "mut "
	}

	typ := ""
	if v.Type != nil {
		typ = ": " + v.Type.String()
	}

	val := ""
	if v.Value != nil {
		val = " = " + v.Value.String()
	}

	exported := ""
	if v.IsExported {
		exported = "pub "
	}

	return fmt.Sprintf("%s%s %s%s%s%s", exported, v.Kind.String(), mut, v.Name.String(), typ, val)
}

func (v *VariableDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitVariableDeclaration(v)
}

// TypeDeclaration represents a type alias or new type definition.
type TypeDeclaration struct {
	Type       Type
	Name       *Identifier
	Comments   []Comment
	Span       position.Span
	IsAlias    bool
	IsExported bool
}

func (t *TypeDeclaration) GetSpan() position.Span { return t.Span }
func (t *TypeDeclaration) declarationNode()       {}
func (t *TypeDeclaration) String() string {
	keyword := "newtype"
	if t.IsAlias {
		keyword = "type"
	}

	exported := ""
	if t.IsExported {
		exported = "pub "
	}

	return fmt.Sprintf("%s%s %s = %s", exported, keyword, t.Name.String(), t.Type.String())
}
func (t *TypeDeclaration) Accept(visitor Visitor) interface{} { return visitor.VisitTypeDeclaration(t) }

// ===== New Declarations: Struct / Enum / Trait / Impl =====

// GenericParamKind indicates the kind of generic parameter.
type GenericParamKind int

const (
	GenericParamType GenericParamKind = iota
	GenericParamConst
	GenericParamLifetime
)

// GenericParameter models a single generic parameter.
type GenericParameter struct {
	ConstType Type
	Name      *Identifier
	Lifetime  string
	Bounds    []Type
	Span      position.Span
	Kind      GenericParamKind
}

func (g *GenericParameter) GetSpan() position.Span { return g.Span }
func (g *GenericParameter) String() string         { return "generic" }
func (g *GenericParameter) Accept(visitor Visitor) interface{} {
	return visitor.VisitGenericParameter(g)
}

// WherePredicate represents a where-clause predicate.
type WherePredicate struct {
	Target Type
	Bounds []Type
	Span   position.Span
}

func (w *WherePredicate) GetSpan() position.Span             { return w.Span }
func (w *WherePredicate) String() string                     { return "where" }
func (w *WherePredicate) Accept(visitor Visitor) interface{} { return visitor.VisitWherePredicate(w) }

// StructField represents a field in a struct or enum variant.
type StructField struct {
	Type     Type
	Name     *Identifier
	Span     position.Span
	IsPublic bool
}

func (sf *StructField) GetSpan() position.Span { return sf.Span }
func (sf *StructField) String() string {
	if sf.Name != nil {
		return fmt.Sprintf("%s: %s", sf.Name.String(), sf.Type.String())
	}

	return sf.Type.String()
}
func (sf *StructField) Accept(visitor Visitor) interface{} { return visitor.VisitStructField(sf) }

// StructDeclaration represents a struct type declaration.
type StructDeclaration struct {
	Name       *Identifier
	Fields     []*StructField
	Generics   []*GenericParameter
	Span       position.Span
	IsExported bool
}

func (d *StructDeclaration) GetSpan() position.Span { return d.Span }
func (d *StructDeclaration) String() string         { return fmt.Sprintf("struct %s", d.Name.String()) }
func (d *StructDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitStructDeclaration(d)
}
func (d *StructDeclaration) statementNode()   {}
func (d *StructDeclaration) declarationNode() {}

// EnumVariant represents a single variant of an enum.
type EnumVariant struct {
	Value  Expression
	Name   *Identifier
	Fields []*StructField
	Span   position.Span
}

func (v *EnumVariant) GetSpan() position.Span             { return v.Span }
func (v *EnumVariant) String() string                     { return v.Name.String() }
func (v *EnumVariant) Accept(visitor Visitor) interface{} { return visitor.VisitEnumVariant(v) }

// EnumDeclaration represents an enum type declaration.
type EnumDeclaration struct {
	Name       *Identifier
	Variants   []*EnumVariant
	Generics   []*GenericParameter
	Span       position.Span
	IsExported bool
}

func (d *EnumDeclaration) GetSpan() position.Span             { return d.Span }
func (d *EnumDeclaration) String() string                     { return fmt.Sprintf("enum %s", d.Name.String()) }
func (d *EnumDeclaration) Accept(visitor Visitor) interface{} { return visitor.VisitEnumDeclaration(d) }
func (d *EnumDeclaration) statementNode()                     {}
func (d *EnumDeclaration) declarationNode()                   {}

// TraitMethod represents a trait method signature.
type TraitMethod struct {
	ReturnType Type
	Name       *Identifier
	Parameters []*Parameter
	Generics   []*GenericParameter
	Span       position.Span
	IsAsync    bool
}

func (m *TraitMethod) GetSpan() position.Span             { return m.Span }
func (m *TraitMethod) String() string                     { return fmt.Sprintf("fn %s(...)", m.Name.String()) }
func (m *TraitMethod) Accept(visitor Visitor) interface{} { return visitor.VisitTraitMethod(m) }

// AssociatedType represents a trait associated type item.
type AssociatedType struct {
	Name   *Identifier
	Bounds []Type
	Span   position.Span
}

func (a *AssociatedType) GetSpan() position.Span             { return a.Span }
func (a *AssociatedType) String() string                     { return fmt.Sprintf("type %s", a.Name.String()) }
func (a *AssociatedType) Accept(visitor Visitor) interface{} { return visitor.VisitAssociatedType(a) }

// TraitDeclaration represents a trait declaration (method signatures only).
type TraitDeclaration struct {
	Name            *Identifier
	Methods         []*TraitMethod
	Generics        []*GenericParameter
	AssociatedTypes []*AssociatedType
	Span            position.Span
	IsExported      bool
}

func (d *TraitDeclaration) GetSpan() position.Span { return d.Span }
func (d *TraitDeclaration) String() string         { return fmt.Sprintf("trait %s", d.Name.String()) }
func (d *TraitDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitTraitDeclaration(d)
}
func (d *TraitDeclaration) statementNode()   {}
func (d *TraitDeclaration) declarationNode() {}

// ImplDeclaration represents an impl block.
type ImplDeclaration struct {
	Trait        Type
	ForType      Type
	Methods      []*FunctionDeclaration
	Generics     []*GenericParameter
	WhereClauses []*WherePredicate
	Span         position.Span
}

func (i *ImplDeclaration) GetSpan() position.Span             { return i.Span }
func (i *ImplDeclaration) String() string                     { return "impl" }
func (i *ImplDeclaration) Accept(visitor Visitor) interface{} { return visitor.VisitImplDeclaration(i) }
func (i *ImplDeclaration) statementNode()                     {}
func (i *ImplDeclaration) declarationNode()                   {}

// ImportDeclaration represents an import statement at top level.
type ImportDeclaration struct {
	Alias      *Identifier
	Path       []*Identifier
	Span       position.Span
	IsExported bool
}

func (d *ImportDeclaration) GetSpan() position.Span { return d.Span }
func (d *ImportDeclaration) declarationNode()       {}
func (d *ImportDeclaration) String() string {
	var segs []string
	for _, s := range d.Path {
		segs = append(segs, s.String())
	}

	alias := ""
	if d.Alias != nil {
		alias = " as " + d.Alias.String()
	}

	exported := ""
	if d.IsExported {
		exported = "pub "
	}

	return fmt.Sprintf("%simport %s%s", exported, strings.Join(segs, "::"), alias)
}

func (d *ImportDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitImportDeclaration(d)
}

// ExportItem represents a single exported symbol with optional alias.
type ExportItem struct {
	Name  *Identifier
	Alias *Identifier
	Span  position.Span
}

func (e *ExportItem) GetSpan() position.Span { return e.Span }
func (e *ExportItem) String() string {
	if e.Alias != nil {
		return fmt.Sprintf("%s as %s", e.Name.String(), e.Alias.String())
	}

	return e.Name.String()
}
func (e *ExportItem) Accept(visitor Visitor) interface{} { return visitor.VisitExportItem(e) }

// ExportDeclaration represents an export statement: export { a, b as c }.
type ExportDeclaration struct {
	Items []*ExportItem
	Span  position.Span
}

func (d *ExportDeclaration) GetSpan() position.Span { return d.Span }
func (d *ExportDeclaration) declarationNode()       {}
func (d *ExportDeclaration) String() string {
	parts := make([]string, 0, len(d.Items))
	for _, it := range d.Items {
		parts = append(parts, it.String())
	}

	return fmt.Sprintf("export { %s }", strings.Join(parts, ", "))
}

func (d *ExportDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitExportDeclaration(d)
}

// ===== Statements =====.

// BlockStatement represents a block of statements enclosed in braces.
type BlockStatement struct {
	Statements []Statement
	Span       position.Span
}

func (b *BlockStatement) GetSpan() position.Span { return b.Span }
func (b *BlockStatement) statementNode()         {}
func (b *BlockStatement) String() string {
	if len(b.Statements) == 0 {
		return "{}"
	}

	var parts []string
	for _, stmt := range b.Statements {
		parts = append(parts, "  "+stmt.String())
	}

	return fmt.Sprintf("{\n%s\n}", strings.Join(parts, "\n"))
}
func (b *BlockStatement) Accept(visitor Visitor) interface{} { return visitor.VisitBlockStatement(b) }

// ExpressionStatement represents a statement consisting of a single expression.
type ExpressionStatement struct {
	Expression Expression
	Span       position.Span
}

func (e *ExpressionStatement) GetSpan() position.Span { return e.Span }
func (e *ExpressionStatement) statementNode()         {}
func (e *ExpressionStatement) String() string         { return e.Expression.String() + ";" }
func (e *ExpressionStatement) Accept(visitor Visitor) interface{} {
	return visitor.VisitExpressionStatement(e)
}

// ReturnStatement represents a return statement.
type ReturnStatement struct {
	Value Expression
	Span  position.Span
}

func (r *ReturnStatement) GetSpan() position.Span { return r.Span }
func (r *ReturnStatement) statementNode()         {}
func (r *ReturnStatement) String() string {
	if r.Value == nil {
		return "return"
	}

	return "return " + r.Value.String()
}
func (r *ReturnStatement) Accept(visitor Visitor) interface{} { return visitor.VisitReturnStatement(r) }

// IfStatement represents an if-else conditional statement.
type IfStatement struct {
	Condition Expression
	ElseBlock Statement
	ThenBlock *BlockStatement
	Span      position.Span
}

func (i *IfStatement) GetSpan() position.Span { return i.Span }
func (i *IfStatement) statementNode()         {}
func (i *IfStatement) String() string {
	result := fmt.Sprintf("if %s %s", i.Condition.String(), i.ThenBlock.String())
	if i.ElseBlock != nil {
		result += " else " + i.ElseBlock.String()
	}

	return result
}
func (i *IfStatement) Accept(visitor Visitor) interface{} { return visitor.VisitIfStatement(i) }

// WhileStatement represents a while loop.
type WhileStatement struct {
	Condition Expression
	Body      *BlockStatement
	Span      position.Span
}

func (w *WhileStatement) GetSpan() position.Span { return w.Span }
func (w *WhileStatement) statementNode()         {}
func (w *WhileStatement) String() string {
	return fmt.Sprintf("while %s %s", w.Condition.String(), w.Body.String())
}
func (w *WhileStatement) Accept(visitor Visitor) interface{} { return visitor.VisitWhileStatement(w) }

// ===== Expressions =====.

// Identifier represents an identifier (variable name, function name, etc.)
type Identifier struct {
	Value string
	Span  position.Span
}

func (i *Identifier) GetSpan() position.Span             { return i.Span }
func (i *Identifier) expressionNode()                    {}
func (i *Identifier) String() string                     { return i.Value }
func (i *Identifier) Accept(visitor Visitor) interface{} { return visitor.VisitIdentifier(i) }

// Literal represents literal values (integers, floats, strings, booleans).
type Literal struct {
	Value interface{}
	Raw   string
	Span  position.Span
	Kind  LiteralKind
}

// LiteralKind represents the kind of literal.
type LiteralKind int

const (
	LiteralInteger LiteralKind = iota
	LiteralFloat
	LiteralString
	LiteralBoolean
	LiteralCharacter
	LiteralNull
)

func (lk LiteralKind) String() string {
	switch lk {
	case LiteralInteger:
		return "integer"
	case LiteralFloat:
		return "float"
	case LiteralString:
		return "string"
	case LiteralBoolean:
		return "boolean"
	case LiteralCharacter:
		return "character"
	case LiteralNull:
		return "null"
	default:
		return "unknown"
	}
}

func (l *Literal) GetSpan() position.Span             { return l.Span }
func (l *Literal) expressionNode()                    {}
func (l *Literal) String() string                     { return l.Raw }
func (l *Literal) Accept(visitor Visitor) interface{} { return visitor.VisitLiteral(l) }

// BinaryExpression represents binary operations (a + b, a == b, etc.)
type BinaryExpression struct {
	Left     Expression
	Right    Expression
	Span     position.Span
	Operator Operator
}

func (b *BinaryExpression) GetSpan() position.Span { return b.Span }
func (b *BinaryExpression) expressionNode()        {}
func (b *BinaryExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left.String(), b.Operator.String(), b.Right.String())
}

func (b *BinaryExpression) Accept(visitor Visitor) interface{} {
	return visitor.VisitBinaryExpression(b)
}

// UnaryExpression represents unary operations (-a, !a, etc.)
type UnaryExpression struct {
	Operand  Expression
	Span     position.Span
	Operator Operator
}

func (u *UnaryExpression) GetSpan() position.Span { return u.Span }
func (u *UnaryExpression) expressionNode()        {}
func (u *UnaryExpression) String() string {
	return fmt.Sprintf("(%s%s)", u.Operator.String(), u.Operand.String())
}
func (u *UnaryExpression) Accept(visitor Visitor) interface{} { return visitor.VisitUnaryExpression(u) }

// CallExpression represents function calls.
type CallExpression struct {
	Function  Expression
	Arguments []Expression
	Span      position.Span
}

func (c *CallExpression) GetSpan() position.Span { return c.Span }
func (c *CallExpression) expressionNode()        {}
func (c *CallExpression) String() string {
	var args []string
	for _, arg := range c.Arguments {
		args = append(args, arg.String())
	}

	return fmt.Sprintf("%s(%s)", c.Function.String(), strings.Join(args, ", "))
}
func (c *CallExpression) Accept(visitor Visitor) interface{} { return visitor.VisitCallExpression(c) }

// MemberExpression represents member access (obj.field, obj.method()).
type MemberExpression struct {
	Object Expression
	Member *Identifier
	Span   position.Span
}

func (m *MemberExpression) GetSpan() position.Span { return m.Span }
func (m *MemberExpression) expressionNode()        {}
func (m *MemberExpression) String() string {
	return fmt.Sprintf("%s.%s", m.Object.String(), m.Member.String())
}

func (m *MemberExpression) Accept(visitor Visitor) interface{} {
	return visitor.VisitMemberExpression(m)
}

// ===== Types =====.

// BasicType represents basic built-in types (int, float, string, bool).
type BasicType struct {
	Span position.Span // Source span of the type
	Kind BasicKind     // Kind of basic type
}

// BasicKind represents the kind of basic type.
type BasicKind int

const (
	BasicInt BasicKind = iota
	BasicFloat
	BasicString
	BasicBool
	BasicChar
	BasicVoid
)

func (bk BasicKind) String() string {
	switch bk {
	case BasicInt:
		return "int"
	case BasicFloat:
		return "float"
	case BasicString:
		return "string"
	case BasicBool:
		return "bool"
	case BasicChar:
		return "char"
	case BasicVoid:
		return "void"
	default:
		return "unknown"
	}
}

func (b *BasicType) GetSpan() position.Span             { return b.Span }
func (b *BasicType) typeNode()                          {}
func (b *BasicType) String() string                     { return b.Kind.String() }
func (b *BasicType) Accept(visitor Visitor) interface{} { return visitor.VisitBasicType(b) }

// IdentifierType represents user-defined types referenced by name.
type IdentifierType struct {
	Name *Identifier
	Span position.Span
}

func (i *IdentifierType) GetSpan() position.Span             { return i.Span }
func (i *IdentifierType) typeNode()                          {}
func (i *IdentifierType) String() string                     { return i.Name.String() }
func (i *IdentifierType) Accept(visitor Visitor) interface{} { return visitor.VisitIdentifierType(i) }

// ===== Operators =====.

// Operator represents all operators in the language.
type Operator int

const (
	// Arithmetic operators.
	OpAdd Operator = iota // +
	OpSub                 // -
	OpMul                 // *
	OpDiv                 // /
	OpMod                 // %
	OpPow                 // **

	// Comparison operators.
	OpEq // ==
	OpNe // !=
	OpLt // <
	OpLe // <=
	OpGt // >
	OpGe // >=

	// Logical operators.
	OpAnd // &&
	OpOr  // ||
	OpNot // !

	// Bitwise operators.
	OpBitAnd // &
	OpBitOr  // |
	OpBitXor // ^
	OpBitNot // ~
	OpShl    // <<
	OpShr    // >>

	// Assignment operators.
	OpAssign    // =
	OpAddAssign // +=
	OpSubAssign // -=
	OpMulAssign // *=
	OpDivAssign // /=
	OpModAssign // %=
)

func (op Operator) String() string {
	switch op {
	case OpAdd:
		return "+"
	case OpSub:
		return "-"
	case OpMul:
		return "*"
	case OpDiv:
		return "/"
	case OpMod:
		return "%"
	case OpPow:
		return "**"
	case OpEq:
		return "=="
	case OpNe:
		return "!="
	case OpLt:
		return "<"
	case OpLe:
		return "<="
	case OpGt:
		return ">"
	case OpGe:
		return ">="
	case OpAnd:
		return "&&"
	case OpOr:
		return "||"
	case OpNot:
		return "!"
	case OpBitAnd:
		return "&"
	case OpBitOr:
		return "|"
	case OpBitXor:
		return "^"
	case OpBitNot:
		return "~"
	case OpShl:
		return "<<"
	case OpShr:
		return ">>"
	case OpAssign:
		return "="
	case OpAddAssign:
		return "+="
	case OpSubAssign:
		return "-="
	case OpMulAssign:
		return "*="
	case OpDivAssign:
		return "/="
	case OpModAssign:
		return "%="
	default:
		return "unknown"
	}
}

// ===== Attributes =====.

// Attribute represents function or type attributes.
type Attribute struct {
	Name *Identifier
	Args []Expression
	Span position.Span
}

func (a *Attribute) GetSpan() position.Span { return a.Span }
func (a *Attribute) String() string {
	if len(a.Args) == 0 {
		return fmt.Sprintf("@%s", a.Name.String())
	}

	var args []string
	for _, arg := range a.Args {
		args = append(args, arg.String())
	}

	return fmt.Sprintf("@%s(%s)", a.Name.String(), strings.Join(args, ", "))
}
func (a *Attribute) Accept(visitor Visitor) interface{} { return visitor.VisitAttribute(a) }
