// Package parser implements the Orizon language parser and AST definitions.
package parser

import (
	"fmt"
	"strings"

	"github.com/orizon-lang/orizon/internal/lexer"
)

// Position represents a source code position with file, line, and column information.
type Position struct {
	File   string // File path
	Line   int    // Line number (1-based)
	Column int    // Column number (1-based)
	Offset int    // Byte offset (0-based)
}

// String returns a string representation of the position.
func (p Position) String() string {
	if p.File != "" {
		return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column)
	}

	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// Span represents a source code span from start to end position.
type Span struct {
	Start Position
	End   Position
}

// String returns a string representation of the span.
func (s Span) String() string {
	if s.Start.File == s.End.File {
		if s.Start.Line == s.End.Line {
			return fmt.Sprintf("%s:%d:%d-%d", s.Start.File, s.Start.Line, s.Start.Column, s.End.Column)
		}

		return fmt.Sprintf("%s:%d:%d-%d:%d", s.Start.File, s.Start.Line, s.Start.Column, s.End.Line, s.End.Column)
	}

	return fmt.Sprintf("%s-%s", s.Start.String(), s.End.String())
}

// Node represents the base interface for all AST nodes.
type Node interface {
	// GetSpan returns the source span for this node.
	GetSpan() Span
	// String returns a string representation of the node.
	String() string
	// Accept implements the visitor pattern.
	Accept(visitor Visitor) interface{}
}

// Statement represents all statement nodes.
type Statement interface {
	Node
	statementNode()
}

// Expression represents all expression nodes.
type Expression interface {
	Node
	expressionNode()
}

// Declaration represents all declaration nodes.
type Declaration interface {
	Statement
	declarationNode()
}

// Type represents all type nodes.
type Type interface {
	Node
	typeNode()
}

// ====== Program and Module Structure ======.

// Program represents the root of the AST.
type Program struct {
	Declarations []Declaration
	Span         Span
}

func (p *Program) GetSpan() Span                      { return p.Span }
func (p *Program) String() string                     { return "Program" }
func (p *Program) Accept(visitor Visitor) interface{} { return visitor.VisitProgram(p) }
func (p *Program) statementNode()                     {}

// ====== Declarations ======.

// FunctionDeclaration represents a function declaration.
type FunctionDeclaration struct {
	ReturnType  Type
	Name        *Identifier
	Body        *BlockStatement
	Effects     *EffectAnnotation
	Parameters  []*Parameter
	Generics    []*GenericParameter
	WhereClause []*WherePredicate
	Span        Span
	IsPublic    bool
	IsAsync     bool
}

func (f *FunctionDeclaration) GetSpan() Span  { return f.Span }
func (f *FunctionDeclaration) String() string { return fmt.Sprintf("func %s", f.Name.Value) }
func (f *FunctionDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitFunctionDeclaration(f)
}
func (f *FunctionDeclaration) statementNode()   {}
func (f *FunctionDeclaration) declarationNode() {}

// Parameter represents a function parameter.
type Parameter struct {
	TypeSpec Type
	Name     *Identifier
	Span     Span
	IsMut    bool
}

func (p *Parameter) GetSpan() Span                      { return p.Span }
func (p *Parameter) String() string                     { return fmt.Sprintf("%s: %s", p.Name.Value, p.TypeSpec.String()) }
func (p *Parameter) Accept(visitor Visitor) interface{} { return visitor.VisitParameter(p) }

// VariableDeclaration represents a variable declaration.
type VariableDeclaration struct {
	TypeSpec    Type
	Initializer Expression
	Name        *Identifier
	Span        Span
	IsMutable   bool
	IsPublic    bool
}

func (v *VariableDeclaration) GetSpan() Span  { return v.Span }
func (v *VariableDeclaration) String() string { return fmt.Sprintf("let %s", v.Name.Value) }
func (v *VariableDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitVariableDeclaration(v)
}
func (v *VariableDeclaration) statementNode()   {}
func (v *VariableDeclaration) declarationNode() {}

// ====== Statements ======.

// BlockStatement represents a block of statements.
type BlockStatement struct {
	Statements []Statement
	Span       Span
}

func (b *BlockStatement) GetSpan() Span                      { return b.Span }
func (b *BlockStatement) String() string                     { return "Block" }
func (b *BlockStatement) Accept(visitor Visitor) interface{} { return visitor.VisitBlockStatement(b) }
func (b *BlockStatement) statementNode()                     {}

// ExpressionStatement represents an expression used as a statement.
type ExpressionStatement struct {
	Expression Expression
	Span       Span
}

func (e *ExpressionStatement) GetSpan() Span  { return e.Span }
func (e *ExpressionStatement) String() string { return "ExprStmt" }
func (e *ExpressionStatement) Accept(visitor Visitor) interface{} {
	return visitor.VisitExpressionStatement(e)
}
func (e *ExpressionStatement) statementNode()   {}
func (e *ExpressionStatement) declarationNode() {} // Allow expression statements as top-level declarations

// ReturnStatement represents a return statement.
type ReturnStatement struct {
	Value Expression
	Span  Span
}

func (r *ReturnStatement) GetSpan() Span                      { return r.Span }
func (r *ReturnStatement) String() string                     { return "return" }
func (r *ReturnStatement) Accept(visitor Visitor) interface{} { return visitor.VisitReturnStatement(r) }
func (r *ReturnStatement) statementNode()                     {}

// IfStatement represents an if statement.
type IfStatement struct {
	Condition Expression
	ThenStmt  Statement
	ElseStmt  Statement
	Span      Span
}

func (i *IfStatement) GetSpan() Span                      { return i.Span }
func (i *IfStatement) String() string                     { return "if" }
func (i *IfStatement) Accept(visitor Visitor) interface{} { return visitor.VisitIfStatement(i) }
func (i *IfStatement) statementNode()                     {}

// WhileStatement represents a while loop.
type WhileStatement struct {
	Condition Expression
	Body      Statement
	Span      Span
}

func (w *WhileStatement) GetSpan() Span                      { return w.Span }
func (w *WhileStatement) String() string                     { return "while" }
func (w *WhileStatement) Accept(visitor Visitor) interface{} { return visitor.VisitWhileStatement(w) }
func (w *WhileStatement) statementNode()                     {}

// ====== Expressions ======.

// Identifier represents an identifier.
type Identifier struct {
	Value string
	Span  Span
}

func (i *Identifier) GetSpan() Span                      { return i.Span }
func (i *Identifier) String() string                     { return i.Value }
func (i *Identifier) Accept(visitor Visitor) interface{} { return visitor.VisitIdentifier(i) }
func (i *Identifier) expressionNode()                    {}

// Literal represents literal values.
type Literal struct {
	Value interface{}
	Span  Span
	Kind  LiteralKind
}

type LiteralKind int

const (
	LiteralInteger LiteralKind = iota
	LiteralFloat
	LiteralString
	LiteralBool
	LiteralNull
)

func (l *Literal) GetSpan() Span                      { return l.Span }
func (l *Literal) String() string                     { return fmt.Sprintf("%v", l.Value) }
func (l *Literal) Accept(visitor Visitor) interface{} { return visitor.VisitLiteral(l) }
func (l *Literal) expressionNode()                    {}

// BinaryExpression represents binary operations.
type BinaryExpression struct {
	Left     Expression
	Right    Expression
	Operator *Operator
	Span     Span
}

func (b *BinaryExpression) GetSpan() Span { return b.Span }
func (b *BinaryExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left, b.Operator.Value, b.Right)
}

func (b *BinaryExpression) Accept(visitor Visitor) interface{} {
	return visitor.VisitBinaryExpression(b)
}
func (b *BinaryExpression) expressionNode() {}

// UnaryExpression represents unary operations.
type UnaryExpression struct {
	Operand  Expression
	Operator *Operator
	Span     Span
}

func (u *UnaryExpression) GetSpan() Span                      { return u.Span }
func (u *UnaryExpression) String() string                     { return fmt.Sprintf("(%s%s)", u.Operator.Value, u.Operand) }
func (u *UnaryExpression) Accept(visitor Visitor) interface{} { return visitor.VisitUnaryExpression(u) }
func (u *UnaryExpression) expressionNode()                    {}

// CallExpression represents function calls.
type CallExpression struct {
	Function  Expression
	Arguments []Expression
	Span      Span
}

func (c *CallExpression) GetSpan() Span                      { return c.Span }
func (c *CallExpression) String() string                     { return fmt.Sprintf("%s(...)", c.Function) }
func (c *CallExpression) Accept(visitor Visitor) interface{} { return visitor.VisitCallExpression(c) }
func (c *CallExpression) expressionNode()                    {}

// AssignmentExpression represents assignment operations.
type AssignmentExpression struct {
	Left     Expression
	Right    Expression
	Operator *Operator
	Span     Span
}

func (a *AssignmentExpression) GetSpan() Span { return a.Span }
func (a *AssignmentExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", a.Left, a.Operator.Value, a.Right)
}

func (a *AssignmentExpression) Accept(visitor Visitor) interface{} {
	return visitor.VisitAssignmentExpression(a)
}
func (a *AssignmentExpression) expressionNode() {}

// TernaryExpression represents ternary conditional expressions (condition ? true_expr : false_expr).
type TernaryExpression struct {
	Condition Expression
	TrueExpr  Expression
	FalseExpr Expression
	Span      Span
}

func (t *TernaryExpression) GetSpan() Span { return t.Span }
func (t *TernaryExpression) String() string {
	return fmt.Sprintf("(%s ? %s : %s)", t.Condition, t.TrueExpr, t.FalseExpr)
}

func (t *TernaryExpression) Accept(visitor Visitor) interface{} {
	return visitor.VisitTernaryExpression(t)
}
func (t *TernaryExpression) expressionNode() {}

// RefinementTypeExpression represents a refinement type expression.
type RefinementTypeExpression struct {
	BaseType  Type
	Predicate Expression
	Variable  *Identifier
	Span      Span
}

func (r *RefinementTypeExpression) GetSpan() Span { return r.Span }
func (r *RefinementTypeExpression) String() string {
	return fmt.Sprintf("{%s: %s | %s}", r.Variable.Value, r.BaseType.String(), r.Predicate.String())
}

func (r *RefinementTypeExpression) Accept(visitor Visitor) interface{} {
	return visitor.VisitRefinementTypeExpression(r)
}
func (r *RefinementTypeExpression) expressionNode() {}
func (r *RefinementTypeExpression) typeNode()       {} // Implement Type interface

// MacroDefinition represents a macro definition.
type MacroDefinition struct {
	Name       *Identifier
	Body       *MacroBody
	Parameters []*MacroParameter
	Span       Span
	IsPublic   bool
	IsHygienic bool
}

func (m *MacroDefinition) GetSpan() Span                      { return m.Span }
func (m *MacroDefinition) String() string                     { return fmt.Sprintf("macro %s", m.Name.Value) }
func (m *MacroDefinition) Accept(visitor Visitor) interface{} { return visitor.VisitMacroDefinition(m) }
func (m *MacroDefinition) statementNode()                     {}
func (m *MacroDefinition) declarationNode()                   {}

// ====== New Declarations: Struct / Enum / Trait / Impl / Import / Export ======

// StructDeclaration represents a struct type declaration.
type StructDeclaration struct {
	Name        *Identifier
	Fields      []*StructField
	Generics    []*GenericParameter
	WhereClause []*WherePredicate
	Span        Span
	IsPublic    bool
}

func (d *StructDeclaration) GetSpan() Span  { return d.Span }
func (d *StructDeclaration) String() string { return fmt.Sprintf("struct %s", d.Name.Value) }
func (d *StructDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitStructDeclaration(d)
}
func (d *StructDeclaration) statementNode()   {}
func (d *StructDeclaration) declarationNode() {}

// EnumDeclaration represents an enum type declaration.
type EnumDeclaration struct {
	Name        *Identifier
	Variants    []*EnumVariant
	Generics    []*GenericParameter
	WhereClause []*WherePredicate
	Span        Span
	IsPublic    bool
}

func (d *EnumDeclaration) GetSpan() Span                      { return d.Span }
func (d *EnumDeclaration) String() string                     { return fmt.Sprintf("enum %s", d.Name.Value) }
func (d *EnumDeclaration) Accept(visitor Visitor) interface{} { return visitor.VisitEnumDeclaration(d) }
func (d *EnumDeclaration) statementNode()                     {}
func (d *EnumDeclaration) declarationNode()                   {}

// TraitDeclaration represents a trait declaration (method signatures only).
type TraitDeclaration struct {
	Name            *Identifier
	Methods         []*TraitMethod
	Generics        []*GenericParameter
	WhereClause     []*WherePredicate
	AssociatedTypes []*AssociatedType
	Span            Span
	IsPublic        bool
}

func (d *TraitDeclaration) GetSpan() Span  { return d.Span }
func (d *TraitDeclaration) String() string { return fmt.Sprintf("trait %s", d.Name.Value) }
func (d *TraitDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitTraitDeclaration(d)
}
func (d *TraitDeclaration) statementNode()   {}
func (d *TraitDeclaration) declarationNode() {}

// ImplBlock represents an impl block (optional trait for type).
type ImplBlock struct {
	Trait        Type
	ForType      Type
	Items        []*FunctionDeclaration
	Generics     []*GenericParameter
	WhereClauses []*WherePredicate
	Span         Span
}

func (i *ImplBlock) GetSpan() Span                      { return i.Span }
func (i *ImplBlock) String() string                     { return "impl" }
func (i *ImplBlock) Accept(visitor Visitor) interface{} { return visitor.VisitImplBlock(i) }
func (i *ImplBlock) statementNode()                     {}
func (i *ImplBlock) declarationNode()                   {}

// ImportDeclaration represents an import statement.
type ImportDeclaration struct {
	Alias    *Identifier
	Path     []*Identifier
	Span     Span
	IsPublic bool
}

func (d *ImportDeclaration) GetSpan() Span  { return d.Span }
func (d *ImportDeclaration) String() string { return "import" }
func (d *ImportDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitImportDeclaration(d)
}
func (d *ImportDeclaration) statementNode()   {}
func (d *ImportDeclaration) declarationNode() {}

// ExportItem represents a single exported item.
type ExportItem struct {
	Name  *Identifier
	Alias *Identifier
	Span  Span
}

// ExportDeclaration represents an export statement.
type ExportDeclaration struct {
	Items []*ExportItem
	Span  Span
}

func (d *ExportDeclaration) GetSpan() Span  { return d.Span }
func (d *ExportDeclaration) String() string { return "export" }
func (d *ExportDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitExportDeclaration(d)
}
func (d *ExportDeclaration) statementNode()   {}
func (d *ExportDeclaration) declarationNode() {}

// ====== Generics / Where / Associated Types ======

// GenericParamKind indicates the kind of generic parameter.
type GenericParamKind int

const (
	GenericParamType GenericParamKind = iota
	GenericParamConst
	GenericParamLifetime
)

// GenericParameter models a single generic parameter: T, const N: usize, or 'a.
type GenericParameter struct {
	ConstType   Type
	DefaultType Type
	Name        *Identifier
	Lifetime    string
	Bounds      []Type
	Span        Span
	Kind        GenericParamKind
}

func (gp *GenericParameter) GetSpan() Span { return gp.Span }
func (gp *GenericParameter) String() string {
	switch gp.Kind {
	case GenericParamType:
		if len(gp.Bounds) > 0 {
			bounds := make([]string, len(gp.Bounds))
			for i, bound := range gp.Bounds {
				bounds[i] = bound.String()
			}

			return fmt.Sprintf("%s: %s", gp.Name.Value, strings.Join(bounds, " + "))
		}

		return gp.Name.Value
	case GenericParamConst:
		return fmt.Sprintf("const %s: %s", gp.Name.Value, gp.ConstType.String())
	case GenericParamLifetime:
		return "'" + gp.Lifetime
	default:
		return "unknown"
	}
}

func (gp *GenericParameter) Accept(visitor Visitor) interface{} {
	return visitor.VisitGenericParameter(gp)
}

// WherePredicate represents a where-clause predicate: Type : Bound + Bound.
type WherePredicate struct {
	Target Type
	Bounds []Type
	Span   Span
}

func (wp *WherePredicate) GetSpan() Span { return wp.Span }
func (wp *WherePredicate) String() string {
	bounds := make([]string, len(wp.Bounds))
	for i, bound := range wp.Bounds {
		bounds[i] = bound.String()
	}

	return fmt.Sprintf("%s: %s", wp.Target.String(), strings.Join(bounds, " + "))
}
func (wp *WherePredicate) Accept(visitor Visitor) interface{} { return visitor.VisitWherePredicate(wp) }

// AssociatedType represents a trait associated type item: type Name [: Bounds] ;.
type AssociatedType struct {
	Name   *Identifier
	Bounds []Type
	Span   Span
}

func (at *AssociatedType) GetSpan() Span { return at.Span }
func (at *AssociatedType) String() string {
	if len(at.Bounds) > 0 {
		bounds := make([]string, len(at.Bounds))
		for i, bound := range at.Bounds {
			bounds[i] = bound.String()
		}

		return fmt.Sprintf("type %s: %s", at.Name.Value, strings.Join(bounds, " + "))
	}

	return fmt.Sprintf("type %s", at.Name.Value)
}
func (at *AssociatedType) Accept(visitor Visitor) interface{} { return visitor.VisitAssociatedType(at) }

// TypeAliasDeclaration represents: type Name = Type ;.
type TypeAliasDeclaration struct {
	Aliased  Type
	Name     *Identifier
	Span     Span
	IsPublic bool
}

func (d *TypeAliasDeclaration) GetSpan() Span  { return d.Span }
func (d *TypeAliasDeclaration) String() string { return "type alias" }
func (d *TypeAliasDeclaration) Accept(visitor Visitor) interface{} {
	// No visitor yet; return nil.
	return nil
}
func (d *TypeAliasDeclaration) statementNode()   {}
func (d *TypeAliasDeclaration) declarationNode() {}

// NewtypeDeclaration represents: newtype Name = Type ;.
// Semantically distinct from alias: creates a nominal type wrapping the base type.
type NewtypeDeclaration struct {
	Base     Type
	Name     *Identifier
	Span     Span
	IsPublic bool
}

func (d *NewtypeDeclaration) GetSpan() Span  { return d.Span }
func (d *NewtypeDeclaration) String() string { return "newtype" }
func (d *NewtypeDeclaration) Accept(visitor Visitor) interface{} {
	// No visitor usage yet.
	return nil
}
func (d *NewtypeDeclaration) statementNode()   {}
func (d *NewtypeDeclaration) declarationNode() {}

// ====== Effect System ======.

// EffectDeclaration represents an effect declaration: effect IO;.
type EffectDeclaration struct {
	Name *Identifier
	Span Span
}

func (e *EffectDeclaration) GetSpan() Span                      { return e.Span }
func (e *EffectDeclaration) String() string                     { return "effect " + e.Name.Value }
func (e *EffectDeclaration) Accept(visitor Visitor) interface{} { return nil }
func (e *EffectDeclaration) statementNode()                     {}
func (e *EffectDeclaration) declarationNode()                   {}

// EffectAnnotation represents an effect annotation: effects(io, alloc, unsafe).
type EffectAnnotation struct {
	Effects []*Effect
	Span    Span
}

func (e *EffectAnnotation) GetSpan() Span                      { return e.Span }
func (e *EffectAnnotation) String() string                     { return "effects(...)" }
func (e *EffectAnnotation) Accept(visitor Visitor) interface{} { return nil }

// Effect represents a single effect in an effect annotation.
type Effect struct {
	Name *Identifier
	Span Span
}

func (e *Effect) GetSpan() Span                      { return e.Span }
func (e *Effect) String() string                     { return e.Name.Value }
func (e *Effect) Accept(visitor Visitor) interface{} { return nil }

// ====== Template Strings and Interpolation ======.

// TemplateString represents a template string with interpolated expressions.
type TemplateString struct {
	Elements []*TemplateElement
	Span     Span
}

func (t *TemplateString) GetSpan() Span                      { return t.Span }
func (t *TemplateString) String() string                     { return "template_string" }
func (t *TemplateString) Accept(visitor Visitor) interface{} { return nil }
func (t *TemplateString) expressionNode()                    {}

// TemplateElement represents an element within a template string.
type TemplateElement struct {
	Expression Expression
	Text       string
	Span       Span
	IsText     bool
}

func (t *TemplateElement) GetSpan() Span { return t.Span }
func (t *TemplateElement) String() string {
	if t.IsText {
		return fmt.Sprintf("text(%s)", t.Text)
	}

	return fmt.Sprintf("interp(%s)", t.Expression.String())
}
func (t *TemplateElement) Accept(visitor Visitor) interface{} { return nil }

// MacroParameter represents a macro parameter with optional type constraints.
type MacroParameter struct {
	DefaultValue Expression
	Name         *Identifier
	Constraint   *MacroConstraint
	Span         Span
	Kind         MacroParameterKind
	IsVariadic   bool
}

func (p *MacroParameter) GetSpan() Span                      { return p.Span }
func (p *MacroParameter) String() string                     { return fmt.Sprintf("$%s", p.Name.Value) }
func (p *MacroParameter) Accept(visitor Visitor) interface{} { return visitor.VisitMacroParameter(p) }

// MacroParameterKind defines the kind of macro parameter.
type MacroParameterKind int

const (
	MacroParamExpression MacroParameterKind = iota // $expr
	MacroParamStatement                            // $stmt
	MacroParamType                                 // $type
	MacroParamIdentifier                           // $ident
	MacroParamLiteral                              // $literal
	MacroParamPattern                              // $pat
	MacroParamBlock                                // $block
	MacroParamToken                                // $token - for low-level macros
)

// MacroConstraint represents constraints on macro parameters.
type MacroConstraint struct {
	TypePattern Type
	ValueRange  *ValueRange
	Pattern     string
	Span        Span
	Kind        MacroConstraintKind
}

func (c *MacroConstraint) GetSpan() Span                      { return c.Span }
func (c *MacroConstraint) String() string                     { return "constraint" }
func (c *MacroConstraint) Accept(visitor Visitor) interface{} { return visitor.VisitMacroConstraint(c) }

// MacroConstraintKind defines types of macro constraints.
type MacroConstraintKind int

const (
	MacroConstraintType MacroConstraintKind = iota
	MacroConstraintValue
	MacroConstraintPattern
)

// ValueRange represents a range of values for constraints.
type ValueRange struct {
	Min Expression
	Max Expression
}

// MacroBody represents the body of a macro definition.
type MacroBody struct {
	Templates []*MacroTemplate
	Span      Span
}

func (b *MacroBody) GetSpan() Span                      { return b.Span }
func (b *MacroBody) String() string                     { return "macro_body" }
func (b *MacroBody) Accept(visitor Visitor) interface{} { return visitor.VisitMacroBody(b) }

// MacroTemplate represents a single template in a macro body.
type MacroTemplate struct {
	Guard    Expression
	Pattern  *MacroPattern
	Body     []Statement
	Span     Span
	Priority int
}

func (t *MacroTemplate) GetSpan() Span                      { return t.Span }
func (t *MacroTemplate) String() string                     { return "template" }
func (t *MacroTemplate) Accept(visitor Visitor) interface{} { return visitor.VisitMacroTemplate(t) }

// MacroPattern represents a pattern for macro template matching.
type MacroPattern struct {
	Elements []*MacroPatternElement
	Span     Span
}

func (p *MacroPattern) GetSpan() Span                      { return p.Span }
func (p *MacroPattern) String() string                     { return "pattern" }
func (p *MacroPattern) Accept(visitor Visitor) interface{} { return visitor.VisitMacroPattern(p) }

// MacroPatternElement represents an element in a macro pattern.
type MacroPatternElement struct {
	Constraint *MacroConstraint
	Repetition *MacroRepetition
	Value      string
	Span       Span
	Kind       MacroPatternKind
}

func (e *MacroPatternElement) GetSpan() Span  { return e.Span }
func (e *MacroPatternElement) String() string { return e.Value }
func (e *MacroPatternElement) Accept(visitor Visitor) interface{} {
	return visitor.VisitMacroPatternElement(e)
}

// MacroPatternKind defines types of macro pattern elements.
type MacroPatternKind int

const (
	MacroPatternLiteral   MacroPatternKind = iota // Literal text that must match exactly
	MacroPatternParameter                         // Parameter reference ($param)
	MacroPatternWildcard                          // Wildcard match (_)
	MacroPatternGroup                             // Grouped pattern (...)
)

// MacroRepetition represents repetition in macro patterns.
type MacroRepetition struct {
	Separator string
	Span      Span
	Kind      MacroRepetitionKind
	Min       int
	Max       int
}

// MacroRepetitionKind defines types of repetition.
type MacroRepetitionKind int

const (
	MacroRepeatZeroOrMore MacroRepetitionKind = iota // *
	MacroRepeatOneOrMore                             // +
	MacroRepeatOptional                              // ?
	MacroRepeatExact                                 // {n}
	MacroRepeatRange                                 // {min,max}
)

// MacroInvocation represents a macro invocation in code.
type MacroInvocation struct {
	Name      *Identifier
	Context   *MacroContext
	Arguments []*MacroArgument
	Span      Span
}

func (i *MacroInvocation) GetSpan() Span {
	if i == nil {
		// Return zero span for nil MacroInvocation to prevent panics
		return Span{}
	}
	return i.Span
}
func (i *MacroInvocation) String() string {
	if i == nil || i.Name == nil {
		return "nil_macro!()"
	}
	return fmt.Sprintf("%s!(...)", i.Name.Value)
}
func (i *MacroInvocation) Accept(visitor Visitor) interface{} {
	if i == nil {
		return nil
	}
	return visitor.VisitMacroInvocation(i)
}
func (i *MacroInvocation) expressionNode() {}
func (i *MacroInvocation) statementNode()  {} // Macros can be both expressions and statements

// MacroArgument represents an argument passed to a macro.
type MacroArgument struct {
	Value interface{}
	Span  Span
	Kind  MacroArgumentKind
}

func (a *MacroArgument) GetSpan() Span                      { return a.Span }
func (a *MacroArgument) String() string                     { return "arg" }
func (a *MacroArgument) Accept(visitor Visitor) interface{} { return visitor.VisitMacroArgument(a) }

// MacroArgumentKind defines types of macro arguments.
type MacroArgumentKind int

const (
	MacroArgExpression MacroArgumentKind = iota
	MacroArgStatement
	MacroArgType
	MacroArgIdentifier
	MacroArgLiteral
	MacroArgBlock
	MacroArgTokenStream
)

// MacroContext represents the context for hygienic macro expansion.
type MacroContext struct {
	CapturedNames  map[string]string
	Span           Span
	SourceLocation Position
	ScopeId        uint64
	ExpansionDepth int
}

func (c *MacroContext) GetSpan() Span                      { return c.Span }
func (c *MacroContext) String() string                     { return "context" }
func (c *MacroContext) Accept(visitor Visitor) interface{} { return visitor.VisitMacroContext(c) }

// ====== Types ======.

// ====== Types ======.

// BasicType represents primitive types.
type BasicType struct {
	Name string
	Span Span
}

func (b *BasicType) GetSpan() Span                      { return b.Span }
func (b *BasicType) String() string                     { return b.Name }
func (b *BasicType) Accept(visitor Visitor) interface{} { return visitor.VisitBasicType(b) }
func (b *BasicType) typeNode()                          {}

// TupleType represents tuple types: (T1, T2, ...) or unit type: ().
type TupleType struct {
	Elements []Type
	Span     Span
}

func (t *TupleType) GetSpan() Span { return t.Span }
func (t *TupleType) String() string {
	if len(t.Elements) == 0 {
		return "()"
	}

	parts := make([]string, len(t.Elements))
	for i, elem := range t.Elements {
		parts[i] = elem.String()
	}

	return "(" + strings.Join(parts, ", ") + ")"
}
func (t *TupleType) Accept(visitor Visitor) interface{} { return nil }
func (t *TupleType) typeNode()                          {}

// DependentType represents a dependent type with where clause: Type where Constraint.
type DependentType struct {
	BaseType   Type
	Constraint Expression
	Span       Span
}

func (d *DependentType) GetSpan() Span { return d.Span }
func (d *DependentType) String() string {
	return fmt.Sprintf("%s where %s", d.BaseType.String(), d.Constraint.String())
}
func (d *DependentType) Accept(visitor Visitor) interface{} { return nil }
func (d *DependentType) typeNode()                          {}

// ====== Operators ======.

// Operator represents operators with precedence and associativity.
type Operator struct {
	Value         string
	Span          Span
	Precedence    int
	Associativity Associativity
	Kind          OperatorKind
}

type Associativity int

const (
	LeftAssociative Associativity = iota
	RightAssociative
	NonAssociative
)

type OperatorKind int

const (
	BinaryOp OperatorKind = iota
	UnaryOp
	AssignmentOp
)

func (o *Operator) GetSpan() Span                      { return o.Span }
func (o *Operator) String() string                     { return o.Value }
func (o *Operator) Accept(visitor Visitor) interface{} { return visitor.VisitOperator(o) }

// ====== Visitor Pattern ======.

// Visitor defines the visitor interface for AST traversal.
type Visitor interface {
	VisitProgram(*Program) interface{}
	VisitFunctionDeclaration(*FunctionDeclaration) interface{}
	VisitParameter(*Parameter) interface{}
	VisitVariableDeclaration(*VariableDeclaration) interface{}
	VisitStructDeclaration(*StructDeclaration) interface{}
	VisitEnumDeclaration(*EnumDeclaration) interface{}
	VisitTraitDeclaration(*TraitDeclaration) interface{}
	VisitImplBlock(*ImplBlock) interface{}
	VisitImportDeclaration(*ImportDeclaration) interface{}
	VisitExportDeclaration(*ExportDeclaration) interface{}
	VisitBlockStatement(*BlockStatement) interface{}
	VisitExpressionStatement(*ExpressionStatement) interface{}
	VisitReturnStatement(*ReturnStatement) interface{}
	VisitIfStatement(*IfStatement) interface{}
	VisitWhileStatement(*WhileStatement) interface{}
	VisitIdentifier(*Identifier) interface{}
	VisitLiteral(*Literal) interface{}
	VisitBinaryExpression(*BinaryExpression) interface{}
	VisitUnaryExpression(*UnaryExpression) interface{}
	VisitCallExpression(*CallExpression) interface{}
	VisitAssignmentExpression(*AssignmentExpression) interface{}
	VisitTernaryExpression(*TernaryExpression) interface{}
	VisitRefinementTypeExpression(*RefinementTypeExpression) interface{}
	VisitBasicType(*BasicType) interface{}
	VisitOperator(*Operator) interface{}
	// Macro-related visitor methods.
	VisitMacroDefinition(*MacroDefinition) interface{}
	VisitMacroParameter(*MacroParameter) interface{}
	VisitMacroConstraint(*MacroConstraint) interface{}
	VisitMacroBody(*MacroBody) interface{}
	VisitMacroTemplate(*MacroTemplate) interface{}
	VisitMacroPattern(*MacroPattern) interface{}
	VisitMacroPatternElement(*MacroPatternElement) interface{}
	VisitMacroInvocation(*MacroInvocation) interface{}
	VisitMacroArgument(*MacroArgument) interface{}
	VisitMacroContext(*MacroContext) interface{}
	// Extended type system visitor methods.
	VisitArrayType(*ArrayType) interface{}
	VisitFunctionType(*FunctionType) interface{}
	VisitStructType(*StructType) interface{}
	VisitEnumType(*EnumType) interface{}
	VisitTraitType(*TraitType) interface{}
	VisitGenericType(*GenericType) interface{}
	VisitReferenceType(*ReferenceType) interface{}
	VisitPointerType(*PointerType) interface{}
	// Extended expression and statement visitor methods.
	VisitArrayExpression(*ArrayExpression) interface{}
	VisitIndexExpression(*IndexExpression) interface{}
	VisitMemberExpression(*MemberExpression) interface{}
	VisitStructExpression(*StructExpression) interface{}
	VisitRangeExpression(*RangeExpression) interface{}
	VisitForStatement(*ForStatement) interface{}
	VisitForInStatement(*ForInStatement) interface{}
	VisitBreakStatement(*BreakStatement) interface{}
	VisitContinueStatement(*ContinueStatement) interface{}
	VisitMatchStatement(*MatchStatement) interface{}
	VisitMatchArm(*MatchArm) interface{}
	// Pattern matching visitor methods.
	VisitLiteralPattern(*LiteralPattern) interface{}
	VisitVariablePattern(*VariablePattern) interface{}
	VisitConstructorPattern(*ConstructorPattern) interface{}
	VisitGuardPattern(*GuardPattern) interface{}
	VisitWildcardPattern(*WildcardPattern) interface{}
	// Generics and where-clause visitor methods.
	VisitGenericParameter(*GenericParameter) interface{}
	VisitWherePredicate(*WherePredicate) interface{}
	VisitAssociatedType(*AssociatedType) interface{}
}

// ====== AST Builder Utilities ======.

// NewProgram creates a new Program node.
func NewProgram(span Span, declarations []Declaration) *Program {
	return &Program{
		Span:         span,
		Declarations: declarations,
	}
}

// NewIdentifier creates a new Identifier node.
func NewIdentifier(span Span, value string) *Identifier {
	return &Identifier{
		Span:  span,
		Value: value,
	}
}

// NewLiteral creates a new Literal node.
func NewLiteral(span Span, value interface{}, kind LiteralKind) *Literal {
	return &Literal{
		Span:  span,
		Value: value,
		Kind:  kind,
	}
}

// NewOperator creates a new Operator node.
func NewOperator(span Span, value string, precedence int, assoc Associativity, kind OperatorKind) *Operator {
	return &Operator{
		Span:          span,
		Value:         value,
		Precedence:    precedence,
		Associativity: assoc,
		Kind:          kind,
	}
}

// ====== Position Conversion Utilities ======.

// TokenToPosition converts a lexer.Token to a Position.
func TokenToPosition(token lexer.Token) Position {
	return Position{
		File:   "", // Will be set by parser
		Line:   token.Line,
		Column: token.Column,
		Offset: token.Span.Start.Offset,
	}
}

// TokenToSpan converts a lexer.Token to a Span.
func TokenToSpan(token lexer.Token) Span {
	start := TokenToPosition(token)
	end := Position{
		File:   start.File,
		Line:   token.Span.End.Line,
		Column: token.Span.End.Column,
		Offset: token.Span.End.Offset,
	}

	return Span{Start: start, End: end}
}

// SpanBetween creates a span between two positions.
func SpanBetween(start, end Position) Span {
	return Span{Start: start, End: end}
}

// CombineSpans combines multiple spans into one encompassing span.
func CombineSpans(spans ...Span) Span {
	if len(spans) == 0 {
		return Span{}
	}

	result := spans[0]
	for _, span := range spans[1:] {
		if span.Start.Offset < result.Start.Offset {
			result.Start = span.Start
		}

		if span.End.Offset > result.End.Offset {
			result.End = span.End
		}
	}

	return result
}

// ====== Pretty Printing ======.

// PrettyPrint returns a formatted string representation of the AST.
func PrettyPrint(node Node) string {
	printer := &astPrinter{indent: 0}

	return printer.print(node)
}

type astPrinter struct {
	indent int
}

func (p *astPrinter) print(node Node) string {
	if node == nil {
		return "<nil>"
	}

	var result strings.Builder

	result.WriteString(strings.Repeat("  ", p.indent))
	result.WriteString(node.String())

	switch n := node.(type) {
	case *Program:
		result.WriteString("\n")

		p.indent++
		for _, decl := range n.Declarations {
			result.WriteString(p.print(decl))
			result.WriteString("\n")
		}

		p.indent--

	case *FunctionDeclaration:
		result.WriteString("\n")

		p.indent++
		result.WriteString(p.print(n.Body))

		p.indent--

	case *BlockStatement:
		result.WriteString("\n")

		p.indent++
		for _, stmt := range n.Statements {
			result.WriteString(p.print(stmt))
			result.WriteString("\n")
		}

		p.indent--

	case *BinaryExpression:
		result.WriteString("\n")

		p.indent++
		result.WriteString(p.print(n.Left))
		result.WriteString("\n")
		result.WriteString(p.print(n.Right))

		p.indent--
	}

	return result.String()
}
