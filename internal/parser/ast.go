// Package parser implements the Orizon language parser and AST definitions
package parser

import (
	"fmt"
	"strings"

	"github.com/orizon-lang/orizon/internal/lexer"
)

// Position represents a source code position with file, line, and column information
type Position struct {
	File   string // File path
	Line   int    // Line number (1-based)
	Column int    // Column number (1-based)
	Offset int    // Byte offset (0-based)
}

// String returns a string representation of the position
func (p Position) String() string {
	if p.File != "" {
		return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column)
	}
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// Span represents a source code span from start to end position
type Span struct {
	Start Position
	End   Position
}

// String returns a string representation of the span
func (s Span) String() string {
	if s.Start.File == s.End.File {
		if s.Start.Line == s.End.Line {
			return fmt.Sprintf("%s:%d:%d-%d", s.Start.File, s.Start.Line, s.Start.Column, s.End.Column)
		}
		return fmt.Sprintf("%s:%d:%d-%d:%d", s.Start.File, s.Start.Line, s.Start.Column, s.End.Line, s.End.Column)
	}
	return fmt.Sprintf("%s-%s", s.Start.String(), s.End.String())
}

// Node represents the base interface for all AST nodes
type Node interface {
	// GetSpan returns the source span for this node
	GetSpan() Span
	// String returns a string representation of the node
	String() string
	// Accept implements the visitor pattern
	Accept(visitor Visitor) interface{}
}

// Statement represents all statement nodes
type Statement interface {
	Node
	statementNode()
}

// Expression represents all expression nodes
type Expression interface {
	Node
	expressionNode()
}

// Declaration represents all declaration nodes
type Declaration interface {
	Statement
	declarationNode()
}

// Type represents all type nodes
type Type interface {
	Node
	typeNode()
}

// ====== Program and Module Structure ======

// Program represents the root of the AST
type Program struct {
	Span         Span
	Declarations []Declaration
}

func (p *Program) GetSpan() Span                      { return p.Span }
func (p *Program) String() string                     { return "Program" }
func (p *Program) Accept(visitor Visitor) interface{} { return visitor.VisitProgram(p) }
func (p *Program) statementNode()                     {}

// ====== Declarations ======

// FunctionDeclaration represents a function declaration
type FunctionDeclaration struct {
	Span       Span
	Name       *Identifier
	Parameters []*Parameter
	ReturnType Type
	Body       *BlockStatement
	IsPublic   bool
	IsAsync    bool
	Generics   []*GenericParameter
}

func (f *FunctionDeclaration) GetSpan() Span  { return f.Span }
func (f *FunctionDeclaration) String() string { return fmt.Sprintf("func %s", f.Name.Value) }
func (f *FunctionDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitFunctionDeclaration(f)
}
func (f *FunctionDeclaration) statementNode()   {}
func (f *FunctionDeclaration) declarationNode() {}

// Parameter represents a function parameter
type Parameter struct {
	Span     Span
	Name     *Identifier
	TypeSpec Type
	IsMut    bool
}

func (p *Parameter) GetSpan() Span                      { return p.Span }
func (p *Parameter) String() string                     { return fmt.Sprintf("%s: %s", p.Name.Value, p.TypeSpec.String()) }
func (p *Parameter) Accept(visitor Visitor) interface{} { return visitor.VisitParameter(p) }

// VariableDeclaration represents a variable declaration
type VariableDeclaration struct {
	Span        Span
	Name        *Identifier
	TypeSpec    Type
	Initializer Expression
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

// ====== Statements ======

// BlockStatement represents a block of statements
type BlockStatement struct {
	Span       Span
	Statements []Statement
}

func (b *BlockStatement) GetSpan() Span                      { return b.Span }
func (b *BlockStatement) String() string                     { return "Block" }
func (b *BlockStatement) Accept(visitor Visitor) interface{} { return visitor.VisitBlockStatement(b) }
func (b *BlockStatement) statementNode()                     {}

// ExpressionStatement represents an expression used as a statement
type ExpressionStatement struct {
	Span       Span
	Expression Expression
}

func (e *ExpressionStatement) GetSpan() Span  { return e.Span }
func (e *ExpressionStatement) String() string { return "ExprStmt" }
func (e *ExpressionStatement) Accept(visitor Visitor) interface{} {
	return visitor.VisitExpressionStatement(e)
}
func (e *ExpressionStatement) statementNode()   {}
func (e *ExpressionStatement) declarationNode() {} // Allow expression statements as top-level declarations

// ReturnStatement represents a return statement
type ReturnStatement struct {
	Span  Span
	Value Expression
}

func (r *ReturnStatement) GetSpan() Span                      { return r.Span }
func (r *ReturnStatement) String() string                     { return "return" }
func (r *ReturnStatement) Accept(visitor Visitor) interface{} { return visitor.VisitReturnStatement(r) }
func (r *ReturnStatement) statementNode()                     {}

// IfStatement represents an if statement
type IfStatement struct {
	Span      Span
	Condition Expression
	ThenStmt  Statement
	ElseStmt  Statement // can be nil
}

func (i *IfStatement) GetSpan() Span                      { return i.Span }
func (i *IfStatement) String() string                     { return "if" }
func (i *IfStatement) Accept(visitor Visitor) interface{} { return visitor.VisitIfStatement(i) }
func (i *IfStatement) statementNode()                     {}

// WhileStatement represents a while loop
type WhileStatement struct {
	Span      Span
	Condition Expression
	Body      Statement
}

func (w *WhileStatement) GetSpan() Span                      { return w.Span }
func (w *WhileStatement) String() string                     { return "while" }
func (w *WhileStatement) Accept(visitor Visitor) interface{} { return visitor.VisitWhileStatement(w) }
func (w *WhileStatement) statementNode()                     {}

// ====== Expressions ======

// Identifier represents an identifier
type Identifier struct {
	Span  Span
	Value string
}

func (i *Identifier) GetSpan() Span                      { return i.Span }
func (i *Identifier) String() string                     { return i.Value }
func (i *Identifier) Accept(visitor Visitor) interface{} { return visitor.VisitIdentifier(i) }
func (i *Identifier) expressionNode()                    {}

// Literal represents literal values
type Literal struct {
	Span  Span
	Value interface{}
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

// BinaryExpression represents binary operations
type BinaryExpression struct {
	Span     Span
	Left     Expression
	Operator *Operator
	Right    Expression
}

func (b *BinaryExpression) GetSpan() Span { return b.Span }
func (b *BinaryExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left, b.Operator.Value, b.Right)
}
func (b *BinaryExpression) Accept(visitor Visitor) interface{} {
	return visitor.VisitBinaryExpression(b)
}
func (b *BinaryExpression) expressionNode() {}

// UnaryExpression represents unary operations
type UnaryExpression struct {
	Span     Span
	Operator *Operator
	Operand  Expression
}

func (u *UnaryExpression) GetSpan() Span                      { return u.Span }
func (u *UnaryExpression) String() string                     { return fmt.Sprintf("(%s%s)", u.Operator.Value, u.Operand) }
func (u *UnaryExpression) Accept(visitor Visitor) interface{} { return visitor.VisitUnaryExpression(u) }
func (u *UnaryExpression) expressionNode()                    {}

// CallExpression represents function calls
type CallExpression struct {
	Span      Span
	Function  Expression
	Arguments []Expression
}

func (c *CallExpression) GetSpan() Span                      { return c.Span }
func (c *CallExpression) String() string                     { return fmt.Sprintf("%s(...)", c.Function) }
func (c *CallExpression) Accept(visitor Visitor) interface{} { return visitor.VisitCallExpression(c) }
func (c *CallExpression) expressionNode()                    {}

// AssignmentExpression represents assignment operations
type AssignmentExpression struct {
	Span     Span
	Left     Expression
	Operator *Operator
	Right    Expression
}

func (a *AssignmentExpression) GetSpan() Span { return a.Span }
func (a *AssignmentExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", a.Left, a.Operator.Value, a.Right)
}
func (a *AssignmentExpression) Accept(visitor Visitor) interface{} {
	return visitor.VisitAssignmentExpression(a)
}
func (a *AssignmentExpression) expressionNode() {}

// TernaryExpression represents ternary conditional expressions (condition ? true_expr : false_expr)
type TernaryExpression struct {
	Span      Span
	Condition Expression
	TrueExpr  Expression
	FalseExpr Expression
}

func (t *TernaryExpression) GetSpan() Span { return t.Span }
func (t *TernaryExpression) String() string {
	return fmt.Sprintf("(%s ? %s : %s)", t.Condition, t.TrueExpr, t.FalseExpr)
}
func (t *TernaryExpression) Accept(visitor Visitor) interface{} {
	return visitor.VisitTernaryExpression(t)
}
func (t *TernaryExpression) expressionNode() {}

// MacroDefinition represents a macro definition
type MacroDefinition struct {
	Span       Span
	Name       *Identifier
	Parameters []*MacroParameter
	Body       *MacroBody
	IsPublic   bool
	IsHygienic bool // Whether this is a hygienic macro
}

func (m *MacroDefinition) GetSpan() Span                      { return m.Span }
func (m *MacroDefinition) String() string                     { return fmt.Sprintf("macro %s", m.Name.Value) }
func (m *MacroDefinition) Accept(visitor Visitor) interface{} { return visitor.VisitMacroDefinition(m) }
func (m *MacroDefinition) statementNode()                     {}
func (m *MacroDefinition) declarationNode()                   {}

// ====== New Declarations: Struct / Enum / Trait / Impl / Import / Export ======

// StructDeclaration represents a struct type declaration
type StructDeclaration struct {
	Span     Span
	Name     *Identifier
	Fields   []*StructField // reuse fields from StructType
	IsPublic bool
	Generics []*GenericParameter // optional generic parameters
}

func (d *StructDeclaration) GetSpan() Span  { return d.Span }
func (d *StructDeclaration) String() string { return fmt.Sprintf("struct %s", d.Name.Value) }
func (d *StructDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitStructDeclaration(d)
}
func (d *StructDeclaration) statementNode()   {}
func (d *StructDeclaration) declarationNode() {}

// EnumDeclaration represents an enum type declaration
type EnumDeclaration struct {
	Span     Span
	Name     *Identifier
	Variants []*EnumVariant // reuse from EnumType
	IsPublic bool
	Generics []*GenericParameter
}

func (d *EnumDeclaration) GetSpan() Span                      { return d.Span }
func (d *EnumDeclaration) String() string                     { return fmt.Sprintf("enum %s", d.Name.Value) }
func (d *EnumDeclaration) Accept(visitor Visitor) interface{} { return visitor.VisitEnumDeclaration(d) }
func (d *EnumDeclaration) statementNode()                     {}
func (d *EnumDeclaration) declarationNode()                   {}

// TraitDeclaration represents a trait declaration (method signatures only)
type TraitDeclaration struct {
	Span            Span
	Name            *Identifier
	Methods         []*TraitMethod // signatures
	IsPublic        bool
	Generics        []*GenericParameter
	AssociatedTypes []*AssociatedType
}

func (d *TraitDeclaration) GetSpan() Span  { return d.Span }
func (d *TraitDeclaration) String() string { return fmt.Sprintf("trait %s", d.Name.Value) }
func (d *TraitDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitTraitDeclaration(d)
}
func (d *TraitDeclaration) statementNode()   {}
func (d *TraitDeclaration) declarationNode() {}

// ImplBlock represents an impl block (optional trait for type)
type ImplBlock struct {
	Span         Span
	Trait        Type                   // optional; nil for inherent impl
	ForType      Type                   // required
	Items        []*FunctionDeclaration // method implementations
	Generics     []*GenericParameter
	WhereClauses []*WherePredicate
}

func (i *ImplBlock) GetSpan() Span                      { return i.Span }
func (i *ImplBlock) String() string                     { return "impl" }
func (i *ImplBlock) Accept(visitor Visitor) interface{} { return visitor.VisitImplBlock(i) }
func (i *ImplBlock) statementNode()                     {}
func (i *ImplBlock) declarationNode()                   {}

// ImportDeclaration represents an import statement
type ImportDeclaration struct {
	Span     Span
	Path     []*Identifier // module path segments
	Alias    *Identifier   // optional alias
	IsPublic bool          // re-export import (pub import)
}

func (d *ImportDeclaration) GetSpan() Span  { return d.Span }
func (d *ImportDeclaration) String() string { return "import" }
func (d *ImportDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitImportDeclaration(d)
}
func (d *ImportDeclaration) statementNode()   {}
func (d *ImportDeclaration) declarationNode() {}

// ExportItem represents a single exported item
type ExportItem struct {
	Span  Span
	Name  *Identifier
	Alias *Identifier // optional alias via "as" (not yet parsed)
}

// ExportDeclaration represents an export statement
type ExportDeclaration struct {
	Span  Span
	Items []*ExportItem // empty means export of nothing (should be handled by parser)
}

func (d *ExportDeclaration) GetSpan() Span  { return d.Span }
func (d *ExportDeclaration) String() string { return "export" }
func (d *ExportDeclaration) Accept(visitor Visitor) interface{} {
	return visitor.VisitExportDeclaration(d)
}
func (d *ExportDeclaration) statementNode()   {}
func (d *ExportDeclaration) declarationNode() {}

// ====== Generics / Where / Associated Types ======

// GenericParamKind indicates the kind of generic parameter
type GenericParamKind int

const (
	GenericParamType GenericParamKind = iota
	GenericParamConst
	GenericParamLifetime
)

// GenericParameter models a single generic parameter: T, const N: usize, or 'a
type GenericParameter struct {
	Span Span
	Kind GenericParamKind
	Name *Identifier // for type/const; nil for lifetime
	// For lifetime parameter
	Lifetime string // e.g., 'a
	// For const parameter
	ConstType Type // type of const parameter
	// Optional trait bounds for type parameters
	Bounds []Type
}

// WherePredicate represents a where-clause predicate: Type : Bound + Bound
type WherePredicate struct {
	Span   Span
	Target Type
	Bounds []Type
}

// AssociatedType represents a trait associated type item: type Name [: Bounds] ;
type AssociatedType struct {
	Span   Span
	Name   *Identifier
	Bounds []Type
}

// TypeAliasDeclaration represents: type Name = Type ;
type TypeAliasDeclaration struct {
	Span     Span
	Name     *Identifier
	Aliased  Type
	IsPublic bool
}

func (d *TypeAliasDeclaration) GetSpan() Span  { return d.Span }
func (d *TypeAliasDeclaration) String() string { return "type alias" }
func (d *TypeAliasDeclaration) Accept(visitor Visitor) interface{} {
	// No visitor yet; return nil
	return nil
}
func (d *TypeAliasDeclaration) statementNode()   {}
func (d *TypeAliasDeclaration) declarationNode() {}

// NewtypeDeclaration represents: newtype Name = Type ;
// Semantically distinct from alias: creates a nominal type wrapping the base type.
type NewtypeDeclaration struct {
	Span     Span
	Name     *Identifier
	Base     Type
	IsPublic bool
}

func (d *NewtypeDeclaration) GetSpan() Span  { return d.Span }
func (d *NewtypeDeclaration) String() string { return "newtype" }
func (d *NewtypeDeclaration) Accept(visitor Visitor) interface{} {
	// No visitor usage yet
	return nil
}
func (d *NewtypeDeclaration) statementNode()   {}
func (d *NewtypeDeclaration) declarationNode() {}

// MacroParameter represents a macro parameter with optional type constraints
type MacroParameter struct {
	Span         Span
	Name         *Identifier
	Kind         MacroParameterKind
	Constraint   *MacroConstraint // Optional type/pattern constraint
	IsVariadic   bool             // Whether this parameter accepts multiple arguments
	DefaultValue Expression       // Optional default value
}

func (p *MacroParameter) GetSpan() Span                      { return p.Span }
func (p *MacroParameter) String() string                     { return fmt.Sprintf("$%s", p.Name.Value) }
func (p *MacroParameter) Accept(visitor Visitor) interface{} { return visitor.VisitMacroParameter(p) }

// MacroParameterKind defines the kind of macro parameter
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

// MacroConstraint represents constraints on macro parameters
type MacroConstraint struct {
	Span        Span
	Kind        MacroConstraintKind
	TypePattern Type        // For type constraints
	ValueRange  *ValueRange // For value constraints
	Pattern     string      // For pattern constraints
}

func (c *MacroConstraint) GetSpan() Span                      { return c.Span }
func (c *MacroConstraint) String() string                     { return "constraint" }
func (c *MacroConstraint) Accept(visitor Visitor) interface{} { return visitor.VisitMacroConstraint(c) }

// MacroConstraintKind defines types of macro constraints
type MacroConstraintKind int

const (
	MacroConstraintType MacroConstraintKind = iota
	MacroConstraintValue
	MacroConstraintPattern
)

// ValueRange represents a range of values for constraints
type ValueRange struct {
	Min Expression
	Max Expression
}

// MacroBody represents the body of a macro definition
type MacroBody struct {
	Span      Span
	Templates []*MacroTemplate // Multiple templates for pattern matching
}

func (b *MacroBody) GetSpan() Span                      { return b.Span }
func (b *MacroBody) String() string                     { return "macro_body" }
func (b *MacroBody) Accept(visitor Visitor) interface{} { return visitor.VisitMacroBody(b) }

// MacroTemplate represents a single template in a macro body
type MacroTemplate struct {
	Span     Span
	Pattern  *MacroPattern // Pattern to match against
	Body     []Statement   // Code to generate
	Guard    Expression    // Optional guard expression
	Priority int           // Priority for template selection
}

func (t *MacroTemplate) GetSpan() Span                      { return t.Span }
func (t *MacroTemplate) String() string                     { return "template" }
func (t *MacroTemplate) Accept(visitor Visitor) interface{} { return visitor.VisitMacroTemplate(t) }

// MacroPattern represents a pattern for macro template matching
type MacroPattern struct {
	Span     Span
	Elements []*MacroPatternElement
}

func (p *MacroPattern) GetSpan() Span                      { return p.Span }
func (p *MacroPattern) String() string                     { return "pattern" }
func (p *MacroPattern) Accept(visitor Visitor) interface{} { return visitor.VisitMacroPattern(p) }

// MacroPatternElement represents an element in a macro pattern
type MacroPatternElement struct {
	Span       Span
	Kind       MacroPatternKind
	Value      string           // Literal text or parameter name
	Constraint *MacroConstraint // Optional constraint
	Repetition *MacroRepetition // Optional repetition specifier
}

func (e *MacroPatternElement) GetSpan() Span  { return e.Span }
func (e *MacroPatternElement) String() string { return e.Value }
func (e *MacroPatternElement) Accept(visitor Visitor) interface{} {
	return visitor.VisitMacroPatternElement(e)
}

// MacroPatternKind defines types of macro pattern elements
type MacroPatternKind int

const (
	MacroPatternLiteral   MacroPatternKind = iota // Literal text that must match exactly
	MacroPatternParameter                         // Parameter reference ($param)
	MacroPatternWildcard                          // Wildcard match (_)
	MacroPatternGroup                             // Grouped pattern (...)
)

// MacroRepetition represents repetition in macro patterns
type MacroRepetition struct {
	Span      Span
	Kind      MacroRepetitionKind
	Min       int    // Minimum repetitions
	Max       int    // Maximum repetitions (-1 for unlimited)
	Separator string // Optional separator
}

// MacroRepetitionKind defines types of repetition
type MacroRepetitionKind int

const (
	MacroRepeatZeroOrMore MacroRepetitionKind = iota // *
	MacroRepeatOneOrMore                             // +
	MacroRepeatOptional                              // ?
	MacroRepeatExact                                 // {n}
	MacroRepeatRange                                 // {min,max}
)

// MacroInvocation represents a macro invocation in code
type MacroInvocation struct {
	Span      Span
	Name      *Identifier
	Arguments []*MacroArgument
	Context   *MacroContext // Context for hygienic expansion
}

func (i *MacroInvocation) GetSpan() Span                      { return i.Span }
func (i *MacroInvocation) String() string                     { return fmt.Sprintf("%s!(...)", i.Name.Value) }
func (i *MacroInvocation) Accept(visitor Visitor) interface{} { return visitor.VisitMacroInvocation(i) }
func (i *MacroInvocation) expressionNode()                    {}
func (i *MacroInvocation) statementNode()                     {} // Macros can be both expressions and statements

// MacroArgument represents an argument passed to a macro
type MacroArgument struct {
	Span  Span
	Value interface{} // Can be Expression, Statement, Type, etc.
	Kind  MacroArgumentKind
}

func (a *MacroArgument) GetSpan() Span                      { return a.Span }
func (a *MacroArgument) String() string                     { return "arg" }
func (a *MacroArgument) Accept(visitor Visitor) interface{} { return visitor.VisitMacroArgument(a) }

// MacroArgumentKind defines types of macro arguments
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

// MacroContext represents the context for hygienic macro expansion
type MacroContext struct {
	Span           Span
	ScopeId        uint64            // Unique scope identifier
	CapturedNames  map[string]string // Mapping of captured names to unique names
	ExpansionDepth int               // Depth of macro expansion to prevent infinite recursion
	SourceLocation Position          // Original source location for debugging
}

func (c *MacroContext) GetSpan() Span                      { return c.Span }
func (c *MacroContext) String() string                     { return "context" }
func (c *MacroContext) Accept(visitor Visitor) interface{} { return visitor.VisitMacroContext(c) }

// ====== Types ======

// ====== Types ======

// BasicType represents primitive types
type BasicType struct {
	Span Span
	Name string
}

func (b *BasicType) GetSpan() Span                      { return b.Span }
func (b *BasicType) String() string                     { return b.Name }
func (b *BasicType) Accept(visitor Visitor) interface{} { return visitor.VisitBasicType(b) }
func (b *BasicType) typeNode()                          {}

// ====== Operators ======

// Operator represents operators with precedence and associativity
type Operator struct {
	Span          Span
	Value         string
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

// ====== Visitor Pattern ======

// Visitor defines the visitor interface for AST traversal
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
	VisitBasicType(*BasicType) interface{}
	VisitOperator(*Operator) interface{}
	// Macro-related visitor methods
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
	// Extended type system visitor methods
	VisitArrayType(*ArrayType) interface{}
	VisitFunctionType(*FunctionType) interface{}
	VisitStructType(*StructType) interface{}
	VisitEnumType(*EnumType) interface{}
	VisitTraitType(*TraitType) interface{}
	VisitGenericType(*GenericType) interface{}
	VisitReferenceType(*ReferenceType) interface{}
	VisitPointerType(*PointerType) interface{}
	// Extended expression and statement visitor methods
	VisitArrayExpression(*ArrayExpression) interface{}
	VisitIndexExpression(*IndexExpression) interface{}
	VisitMemberExpression(*MemberExpression) interface{}
	VisitStructExpression(*StructExpression) interface{}
	VisitForStatement(*ForStatement) interface{}
	VisitBreakStatement(*BreakStatement) interface{}
	VisitContinueStatement(*ContinueStatement) interface{}
	VisitMatchStatement(*MatchStatement) interface{}
	// Dependent type system visitor methods
	VisitDependentFunctionType(*DependentFunctionType) interface{}
	VisitDependentParameter(*DependentParameter) interface{}
	VisitRefinementType(*RefinementType) interface{}
	VisitSizedArrayType(*SizedArrayType) interface{}
	VisitIndexType(*IndexType) interface{}
	VisitProofType(*ProofType) interface{}
}

// ====== AST Builder Utilities ======

// NewProgram creates a new Program node
func NewProgram(span Span, declarations []Declaration) *Program {
	return &Program{
		Span:         span,
		Declarations: declarations,
	}
}

// NewIdentifier creates a new Identifier node
func NewIdentifier(span Span, value string) *Identifier {
	return &Identifier{
		Span:  span,
		Value: value,
	}
}

// NewLiteral creates a new Literal node
func NewLiteral(span Span, value interface{}, kind LiteralKind) *Literal {
	return &Literal{
		Span:  span,
		Value: value,
		Kind:  kind,
	}
}

// NewOperator creates a new Operator node
func NewOperator(span Span, value string, precedence int, assoc Associativity, kind OperatorKind) *Operator {
	return &Operator{
		Span:          span,
		Value:         value,
		Precedence:    precedence,
		Associativity: assoc,
		Kind:          kind,
	}
}

// ====== Position Conversion Utilities ======

// TokenToPosition converts a lexer.Token to a Position
func TokenToPosition(token lexer.Token) Position {
	return Position{
		File:   "", // Will be set by parser
		Line:   token.Line,
		Column: token.Column,
		Offset: token.Span.Start.Offset,
	}
}

// TokenToSpan converts a lexer.Token to a Span
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

// SpanBetween creates a span between two positions
func SpanBetween(start, end Position) Span {
	return Span{Start: start, End: end}
}

// CombineSpans combines multiple spans into one encompassing span
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

// ====== Pretty Printing ======

// PrettyPrint returns a formatted string representation of the AST
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
