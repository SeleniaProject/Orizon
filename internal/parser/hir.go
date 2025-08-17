// High-level Intermediate Representation (HIR) for Orizon language
// This file defines the HIR structure that serves as a bridge between
// the surface syntax (AST) and lower-level representations. HIR provides:
// 1. Type-erased representation with explicit type information
// 2. Desugared constructs for easier analysis and optimization
// 3. Explicit control flow and data flow information
// 4. Simplified structure for backend code generation

package parser

import (
	"fmt"
	"strings"
)

// ====== HIR Core Types ======

// HIRNode represents any node in the HIR
type HIRNode interface {
	String() string
	GetSpan() Span
	Accept(visitor HIRVisitor) interface{}
	GetHIRKind() HIRKind
}

// HIRKind represents the different kinds of HIR nodes
type HIRKind int

const (
	HIRKindModule HIRKind = iota
	HIRKindFunction
	HIRKindVariable
	HIRKindBlock
	HIRKindStatement
	HIRKindExpression
	HIRKindType
	HIRKindPattern
)

// HIRVisitor defines the visitor pattern for HIR traversal
type HIRVisitor interface {
	VisitHIRModule(*HIRModule) interface{}
	VisitHIRFunction(*HIRFunction) interface{}
	VisitHIRVariable(*HIRVariable) interface{}
	VisitHIRBlock(*HIRBlock) interface{}
	VisitHIRStatement(*HIRStatement) interface{}
	VisitHIRExpression(*HIRExpression) interface{}
	VisitHIRType(*HIRType) interface{}
	VisitHIRPattern(*HIRPattern) interface{}
}

// ====== HIR Module Structure ======

// HIRModule represents a complete compilation unit
type HIRModule struct {
	Span      Span
	Name      string
	Functions []*HIRFunction
	Variables []*HIRVariable
	Types     []*HIRTypeDefinition
	Imports   []*HIRImport
	Exports   []*HIRExport
	Metadata  *HIRModuleMetadata
}

func (m *HIRModule) String() string                        { return fmt.Sprintf("module %s", m.Name) }
func (m *HIRModule) GetSpan() Span                         { return m.Span }
func (m *HIRModule) Accept(visitor HIRVisitor) interface{} { return visitor.VisitHIRModule(m) }
func (m *HIRModule) GetHIRKind() HIRKind                   { return HIRKindModule }

// HIRModuleMetadata contains compilation metadata
type HIRModuleMetadata struct {
	Version      string
	Author       string
	License      string
	Dependencies map[string]string
	CompileFlags []string
}

// HIRImport represents module imports
type HIRImport struct {
	Span       Span
	ModuleName string
	Items      []string // specific items to import, empty means all
	Alias      string   // import alias
	IsPublic   bool     // re-export import
}

// HIRExport represents module exports
type HIRExport struct {
	Span      Span
	ItemName  string
	Alias     string
	IsDefault bool
}

// ====== HIR Function Structure ======

// HIRFunction represents a function in HIR
type HIRFunction struct {
	Span            Span
	Name            string
	Parameters      []*HIRParameter
	ReturnType      *HIRType
	Body            *HIRBlock
	IsPublic        bool
	IsExtern        bool
	IsGeneric       bool
	TypeParameters  []*HIRTypeParameter
	Constraints     []*HIRConstraint
	CallConvention  string
	Attributes      []string
	LocalVariables  []*HIRVariable
	ClosureCaptures []*HIRVariable
}

func (f *HIRFunction) String() string                        { return fmt.Sprintf("fn %s", f.Name) }
func (f *HIRFunction) GetSpan() Span                         { return f.Span }
func (f *HIRFunction) Accept(visitor HIRVisitor) interface{} { return visitor.VisitHIRFunction(f) }
func (f *HIRFunction) GetHIRKind() HIRKind                   { return HIRKindFunction }

// HIRParameter represents a function parameter
type HIRParameter struct {
	Span    Span
	Name    string
	Type    *HIRType
	IsRef   bool
	IsMut   bool
	Default *HIRExpression
}

// HIRTypeParameter represents a type parameter for generic functions
type HIRTypeParameter struct {
	Span     Span
	Name     string
	Bounds   []*HIRType
	Default  *HIRType
	Variance VarianceKind
}

// VarianceKind represents type parameter variance
type VarianceKind int

const (
	VarianceInvariant VarianceKind = iota
	VarianceCovariant
	VarianceContravariant
)

// HIRConstraint represents type constraints
type HIRConstraint struct {
	Span        Span
	Type        *HIRType
	Trait       *HIRType
	WhereClause string
}

// ====== HIR Variable Structure ======

// HIRVariable represents a variable declaration
type HIRVariable struct {
	Span         Span
	Name         string
	Type         *HIRType
	Initializer  *HIRExpression
	IsMutable    bool
	IsStatic     bool
	IsGlobal     bool
	Scope        ScopeKind
	LifetimeInfo *LifetimeInfo
}

func (v *HIRVariable) String() string                        { return fmt.Sprintf("var %s", v.Name) }
func (v *HIRVariable) GetSpan() Span                         { return v.Span }
func (v *HIRVariable) Accept(visitor HIRVisitor) interface{} { return visitor.VisitHIRVariable(v) }
func (v *HIRVariable) GetHIRKind() HIRKind                   { return HIRKindVariable }

// ScopeKind represents variable scope
type ScopeKind int

const (
	ScopeLocal ScopeKind = iota
	ScopeParameter
	ScopeGlobal
	ScopeClosure
	ScopeStatic
)

// LifetimeInfo contains lifetime analysis information
type LifetimeInfo struct {
	Lifetime   string
	Borrowck   BorrowKind
	Region     string
	Mutability MutabilityKind
}

// BorrowKind represents borrowing information
type BorrowKind int

const (
	BorrowNone BorrowKind = iota
	BorrowShared
	BorrowUnique
	BorrowMutable
)

// MutabilityKind represents mutability information
type MutabilityKind int

const (
	MutabilityImmutable MutabilityKind = iota
	MutabilityMutable
	MutabilityConst
)

// ====== HIR Block and Statement Structure ======

// HIRBlock represents a block of statements
type HIRBlock struct {
	Span       Span
	Statements []*HIRStatement
	Expression *HIRExpression // optional trailing expression
	Scope      *HIRScope
}

func (b *HIRBlock) String() string                        { return "block" }
func (b *HIRBlock) GetSpan() Span                         { return b.Span }
func (b *HIRBlock) Accept(visitor HIRVisitor) interface{} { return visitor.VisitHIRBlock(b) }
func (b *HIRBlock) GetHIRKind() HIRKind                   { return HIRKindBlock }

// HIRScope represents lexical scope information
type HIRScope struct {
	ID        int
	Parent    *HIRScope
	Children  []*HIRScope
	Variables map[string]*HIRVariable
	Types     map[string]*HIRType
	ScopeKind ScopeKind
}

// HIRStatement represents various kinds of statements
type HIRStatement struct {
	Span Span
	Kind HIRStatementKind
	Data interface{} // specific statement data
}

func (s *HIRStatement) String() string                        { return s.Kind.String() }
func (s *HIRStatement) GetSpan() Span                         { return s.Span }
func (s *HIRStatement) Accept(visitor HIRVisitor) interface{} { return visitor.VisitHIRStatement(s) }
func (s *HIRStatement) GetHIRKind() HIRKind                   { return HIRKindStatement }

// HIRStatementKind represents different statement types
type HIRStatementKind int

const (
	HIRStmtExpression HIRStatementKind = iota
	HIRStmtLet
	HIRStmtAssign
	HIRStmtReturn
	HIRStmtBreak
	HIRStmtContinue
	HIRStmtWhile
	HIRStmtFor
	HIRStmtIf
	HIRStmtMatch
	HIRStmtDefer
	HIRStmtUnsafe
)

func (k HIRStatementKind) String() string {
	switch k {
	case HIRStmtExpression:
		return "expr"
	case HIRStmtLet:
		return "let"
	case HIRStmtAssign:
		return "assign"
	case HIRStmtReturn:
		return "return"
	case HIRStmtBreak:
		return "break"
	case HIRStmtContinue:
		return "continue"
	case HIRStmtWhile:
		return "while"
	case HIRStmtFor:
		return "for"
	case HIRStmtIf:
		return "if"
	case HIRStmtMatch:
		return "match"
	case HIRStmtDefer:
		return "defer"
	case HIRStmtUnsafe:
		return "unsafe"
	default:
		return "unknown"
	}
}

// ====== HIR Statement Data Structures ======

// HIRLetStatement represents variable declarations
type HIRLetStatement struct {
	Variable    *HIRVariable
	Pattern     *HIRPattern
	Initializer *HIRExpression
}

// HIRAssignStatement represents assignments
type HIRAssignStatement struct {
	Target   *HIRExpression
	Value    *HIRExpression
	Operator AssignOperatorKind
}

// AssignOperatorKind represents assignment operators
type AssignOperatorKind int

const (
	AssignSimple AssignOperatorKind = iota
	AssignAdd
	AssignSub
	AssignMul
	AssignDiv
	AssignMod
	AssignAnd
	AssignOr
	AssignXor
	AssignShl
	AssignShr
)

// HIRReturnStatement represents return statements
type HIRReturnStatement struct {
	Value *HIRExpression // nil for bare return
}

// HIRIfStatement represents conditional statements
type HIRIfStatement struct {
	Condition *HIRExpression
	ThenBlock *HIRBlock
	ElseBlock *HIRBlock // nil if no else
}

// HIRWhileStatement represents while loops
type HIRWhileStatement struct {
	Condition *HIRExpression
	Body      *HIRBlock
	Label     string // for break/continue targeting
}

// HIRForStatement represents for loops
type HIRForStatement struct {
	Pattern  *HIRPattern
	Iterator *HIRExpression
	Body     *HIRBlock
	Label    string
}

// HIRMatchStatement represents pattern matching
type HIRMatchStatement struct {
	Scrutinee *HIRExpression
	Arms      []*HIRMatchArm
}

// HIRMatchArm represents a match arm
type HIRMatchArm struct {
	Span    Span
	Pattern *HIRPattern
	Guard   *HIRExpression // optional guard condition
	Body    *HIRBlock
}

// ====== HIR Expression Structure ======

// HIRExpression represents expressions in HIR
type HIRExpression struct {
	Span Span
	Type *HIRType // explicit type information
	Kind HIRExpressionKind
	Data interface{} // specific expression data
}

func (e *HIRExpression) String() string                        { return e.Kind.String() }
func (e *HIRExpression) GetSpan() Span                         { return e.Span }
func (e *HIRExpression) Accept(visitor HIRVisitor) interface{} { return visitor.VisitHIRExpression(e) }
func (e *HIRExpression) GetHIRKind() HIRKind                   { return HIRKindExpression }

// HIRExpressionKind represents different expression types
type HIRExpressionKind int

const (
	HIRExprLiteral HIRExpressionKind = iota
	HIRExprVariable
	HIRExprCall
	HIRExprMethodCall
	HIRExprFieldAccess
	HIRExprIndex
	HIRExprBinary
	HIRExprUnary
	HIRExprCast
	HIRExprRef
	HIRExprDeref
	HIRExprArray
	HIRExprStruct
	HIRExprTuple
	HIRExprClosure
	HIRExprBlock
	HIRExprIf
	HIRExprMatch
	HIRExprLoop
	HIRExprBreak
	HIRExprContinue
	HIRExprReturn
	HIRExprYield
	HIRExprAwait
	HIRExprTry
)

func (k HIRExpressionKind) String() string {
	switch k {
	case HIRExprLiteral:
		return "literal"
	case HIRExprVariable:
		return "variable"
	case HIRExprCall:
		return "call"
	case HIRExprMethodCall:
		return "method_call"
	case HIRExprFieldAccess:
		return "field_access"
	case HIRExprIndex:
		return "index"
	case HIRExprBinary:
		return "binary"
	case HIRExprUnary:
		return "unary"
	case HIRExprCast:
		return "cast"
	case HIRExprRef:
		return "ref"
	case HIRExprDeref:
		return "deref"
	case HIRExprArray:
		return "array"
	case HIRExprStruct:
		return "struct"
	case HIRExprTuple:
		return "tuple"
	case HIRExprClosure:
		return "closure"
	case HIRExprBlock:
		return "block"
	case HIRExprIf:
		return "if"
	case HIRExprMatch:
		return "match"
	case HIRExprLoop:
		return "loop"
	case HIRExprBreak:
		return "break"
	case HIRExprContinue:
		return "continue"
	case HIRExprReturn:
		return "return"
	case HIRExprYield:
		return "yield"
	case HIRExprAwait:
		return "await"
	case HIRExprTry:
		return "try"
	default:
		return "unknown"
	}
}

// ====== HIR Expression Data Structures ======

// HIRLiteralExpression represents literal values
type HIRLiteralExpression struct {
	Value interface{}
	Kind  LiteralKind
}

// HIRVariableExpression represents variable references
type HIRVariableExpression struct {
	Name     string
	Variable *HIRVariable // resolved variable reference
}

// HIRCallExpression represents function calls
type HIRCallExpression struct {
	Function  *HIRExpression
	Arguments []*HIRExpression
	TypeArgs  []*HIRType // type arguments for generic calls
}

// HIRMethodCallExpression represents method calls
type HIRMethodCallExpression struct {
	Receiver  *HIRExpression
	Method    string
	Arguments []*HIRExpression
	TypeArgs  []*HIRType
}

// HIRFieldAccessExpression represents field access
type HIRFieldAccessExpression struct {
	Object    *HIRExpression
	Field     string
	FieldType *HIRType
}

// HIRIndexExpression represents array/map indexing
type HIRIndexExpression struct {
	Object *HIRExpression
	Index  *HIRExpression
}

// HIRBinaryExpression represents binary operations
type HIRBinaryExpression struct {
	Left     *HIRExpression
	Right    *HIRExpression
	Operator BinaryOperatorKind
}

// BinaryOperatorKind represents binary operators
type BinaryOperatorKind int

const (
	BinOpAdd BinaryOperatorKind = iota
	BinOpSub
	BinOpMul
	BinOpDiv
	BinOpMod
	BinOpPow
	BinOpEq
	BinOpNe
	BinOpLt
	BinOpLe
	BinOpGt
	BinOpGe
	BinOpAnd
	BinOpOr
	BinOpXor
	BinOpShl
	BinOpShr
	BinOpLogicalAnd
	BinOpLogicalOr
	BinOpRange
	BinOpRangeInclusive
)

// HIRUnaryExpression represents unary operations
type HIRUnaryExpression struct {
	Operand  *HIRExpression
	Operator UnaryOperatorKind
}

// UnaryOperatorKind represents unary operators
type UnaryOperatorKind int

const (
	UnaryOpNeg UnaryOperatorKind = iota
	UnaryOpNot
	UnaryOpBitNot
	UnaryOpRef
	UnaryOpDeref
	UnaryOpMove
	UnaryOpCopy
)

// HIRCastExpression represents type casts
type HIRCastExpression struct {
	Expression *HIRExpression
	TargetType *HIRType
	CastKind   CastKind
}

// CastKind represents different cast types
type CastKind int

const (
	CastImplicit CastKind = iota
	CastExplicit
	CastUnsafe
	CastCoercion
)

// HIRArrayExpression represents array literals
type HIRArrayExpression struct {
	Elements []*HIRExpression
	Length   int
}

// HIRStructExpression represents struct literals
type HIRStructExpression struct {
	Type   *HIRType
	Fields []*HIRFieldInit
}

// HIRFieldInit represents struct field initialization
type HIRFieldInit struct {
	Name  string
	Value *HIRExpression
}

// HIRTupleExpression represents tuple literals
type HIRTupleExpression struct {
	Elements []*HIRExpression
}

// HIRClosureExpression represents closures/lambdas
type HIRClosureExpression struct {
	Parameters []*HIRParameter
	ReturnType *HIRType
	Body       *HIRBlock
	Captures   []*HIRVariable
}

// ====== HIR Type System ======

// HIRType represents type information in HIR
type HIRType struct {
	Span Span
	Kind HIRTypeKind
	Data interface{} // specific type data
}

func (t *HIRType) String() string                        { return t.Kind.String() }
func (t *HIRType) GetSpan() Span                         { return t.Span }
func (t *HIRType) Accept(visitor HIRVisitor) interface{} { return visitor.VisitHIRType(t) }
func (t *HIRType) GetHIRKind() HIRKind                   { return HIRKindType }

// HIRTypeKind represents different type kinds
type HIRTypeKind int

const (
	HIRTypePrimitive HIRTypeKind = iota
	HIRTypeArray
	HIRTypeSlice
	HIRTypePointer
	HIRTypeReference
	HIRTypeFunction
	HIRTypeStruct
	HIRTypeEnum
	HIRTypeTrait
	HIRTypeTuple
	HIRTypeGeneric
	HIRTypeAssociated
	HIRTypeDependent
	HIRTypeRefinement
	HIRTypeExistential
	HIRTypeUnion
	HIRTypeIntersection
	HIRTypeNever
	HIRTypeAny
)

func (k HIRTypeKind) String() string {
	switch k {
	case HIRTypePrimitive:
		return "primitive"
	case HIRTypeArray:
		return "array"
	case HIRTypeSlice:
		return "slice"
	case HIRTypePointer:
		return "pointer"
	case HIRTypeReference:
		return "reference"
	case HIRTypeFunction:
		return "function"
	case HIRTypeStruct:
		return "struct"
	case HIRTypeEnum:
		return "enum"
	case HIRTypeTrait:
		return "trait"
	case HIRTypeTuple:
		return "tuple"
	case HIRTypeGeneric:
		return "generic"
	case HIRTypeAssociated:
		return "associated"
	case HIRTypeDependent:
		return "dependent"
	case HIRTypeRefinement:
		return "refinement"
	case HIRTypeExistential:
		return "existential"
	case HIRTypeUnion:
		return "union"
	case HIRTypeIntersection:
		return "intersection"
	case HIRTypeNever:
		return "never"
	case HIRTypeAny:
		return "any"
	default:
		return "unknown"
	}
}

// ====== HIR Type Data Structures ======

// HIRPrimitiveType represents primitive types
type HIRPrimitiveType struct {
	Name string
	Size int // in bytes
}

// HIRArrayType represents array types
type HIRArrayType struct {
	ElementType *HIRType
	Length      int
}

// HIRSliceType represents slice types
type HIRSliceType struct {
	ElementType *HIRType
}

// HIRPointerType represents pointer types
type HIRPointerType struct {
	PointeeType *HIRType
	IsMutable   bool
}

// HIRReferenceType represents reference types
type HIRReferenceType struct {
	ReferentType *HIRType
	IsMutable    bool
	Lifetime     string
}

// HIRFunctionType represents function types
type HIRFunctionType struct {
	Parameters []*HIRType
	ReturnType *HIRType
	IsAsync    bool
	IsUnsafe   bool
}

// HIRStructType represents struct types
type HIRStructType struct {
	Name   string
	Fields []*HIRFieldType
}

// HIRFieldType represents struct field types
type HIRFieldType struct {
	Name string
	Type *HIRType
}

// HIREnumType represents enum types
type HIREnumType struct {
	Name     string
	Variants []*HIRVariantType
}

// HIRVariantType represents enum variant types
type HIRVariantType struct {
	Name   string
	Fields []*HIRType
}

// HIRTraitType represents trait types
type HIRTraitType struct {
	Name            string
	Methods         []*HIRMethodSignature
	AssociatedTypes []*HIRTraitAssociatedType
}

// HIRMethodSignature represents method signatures
type HIRMethodSignature struct {
	Name           string
	Parameters     []*HIRType
	ReturnType     *HIRType
	TypeParameters []*HIRTypeParameter
}

// HIRTraitAssociatedType represents a trait associated type item
type HIRTraitAssociatedType struct {
	Name   string
	Bounds []*HIRType
}

// HIRAliasType represents a type alias definition
type HIRAliasType struct {
	Target *HIRType
}

// HIRTupleType represents tuple types
type HIRTupleType struct {
	Elements []*HIRType
}

// HIRGenericType represents generic type parameters
type HIRGenericType struct {
	Name   string
	Bounds []*HIRType
}

// ====== HIR Pattern System ======

// HIRPattern represents patterns for destructuring
type HIRPattern struct {
	Span Span
	Type *HIRType
	Kind HIRPatternKind
	Data interface{} // specific pattern data
}

func (p *HIRPattern) String() string                        { return p.Kind.String() }
func (p *HIRPattern) GetSpan() Span                         { return p.Span }
func (p *HIRPattern) Accept(visitor HIRVisitor) interface{} { return visitor.VisitHIRPattern(p) }
func (p *HIRPattern) GetHIRKind() HIRKind                   { return HIRKindPattern }

// HIRPatternKind represents different pattern types
type HIRPatternKind int

const (
	HIRPatternWildcard HIRPatternKind = iota
	HIRPatternLiteral
	HIRPatternVariable
	HIRPatternStruct
	HIRPatternTuple
	HIRPatternArray
	HIRPatternEnum
	HIRPatternRef
	HIRPatternRange
	HIRPatternOr
	HIRPatternGuard
)

func (k HIRPatternKind) String() string {
	switch k {
	case HIRPatternWildcard:
		return "_"
	case HIRPatternLiteral:
		return "literal"
	case HIRPatternVariable:
		return "variable"
	case HIRPatternStruct:
		return "struct"
	case HIRPatternTuple:
		return "tuple"
	case HIRPatternArray:
		return "array"
	case HIRPatternEnum:
		return "enum"
	case HIRPatternRef:
		return "ref"
	case HIRPatternRange:
		return "range"
	case HIRPatternOr:
		return "or"
	case HIRPatternGuard:
		return "guard"
	default:
		return "unknown"
	}
}

// ====== HIR Type Definitions ======

// HIRTypeDefinition represents user-defined types
type HIRTypeDefinition struct {
	Span       Span
	Name       string
	Kind       TypeDefinitionKind
	TypeParams []*HIRTypeParameter
	Data       interface{} // specific type definition data
}

func (td *HIRTypeDefinition) String() string { return fmt.Sprintf("type %s", td.Name) }
func (td *HIRTypeDefinition) GetSpan() Span  { return td.Span }
func (td *HIRTypeDefinition) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitHIRType(&HIRType{Span: td.Span, Kind: HIRTypeStruct, Data: td})
}
func (td *HIRTypeDefinition) GetHIRKind() HIRKind { return HIRKindType }

// TypeDefinitionKind represents different type definition kinds
type TypeDefinitionKind int

const (
	TypeDefStruct TypeDefinitionKind = iota
	TypeDefEnum
	TypeDefTrait
	TypeDefAlias
	TypeDefNewtype
)

// ====== HIR Utilities ======

// NewHIRModule creates a new HIR module
func NewHIRModule(span Span, name string) *HIRModule {
	return &HIRModule{
		Span:      span,
		Name:      name,
		Functions: make([]*HIRFunction, 0),
		Variables: make([]*HIRVariable, 0),
		Types:     make([]*HIRTypeDefinition, 0),
		Imports:   make([]*HIRImport, 0),
		Exports:   make([]*HIRExport, 0),
		Metadata: &HIRModuleMetadata{
			Dependencies: make(map[string]string),
			CompileFlags: make([]string, 0),
		},
	}
}

// NewHIRFunction creates a new HIR function
func NewHIRFunction(span Span, name string) *HIRFunction {
	return &HIRFunction{
		Span:            span,
		Name:            name,
		Parameters:      make([]*HIRParameter, 0),
		TypeParameters:  make([]*HIRTypeParameter, 0),
		Constraints:     make([]*HIRConstraint, 0),
		Attributes:      make([]string, 0),
		LocalVariables:  make([]*HIRVariable, 0),
		ClosureCaptures: make([]*HIRVariable, 0),
	}
}

// NewHIRVariable creates a new HIR variable
func NewHIRVariable(span Span, name string, hirType *HIRType) *HIRVariable {
	return &HIRVariable{
		Span: span,
		Name: name,
		Type: hirType,
	}
}

// NewHIRBlock creates a new HIR block
func NewHIRBlock(span Span) *HIRBlock {
	return &HIRBlock{
		Span:       span,
		Statements: make([]*HIRStatement, 0),
		Scope: &HIRScope{
			Variables: make(map[string]*HIRVariable),
			Types:     make(map[string]*HIRType),
			Children:  make([]*HIRScope, 0),
		},
	}
}

// NewHIRExpression creates a new HIR expression
func NewHIRExpression(span Span, hirType *HIRType, kind HIRExpressionKind, data interface{}) *HIRExpression {
	return &HIRExpression{
		Span: span,
		Type: hirType,
		Kind: kind,
		Data: data,
	}
}

// NewHIRType creates a new HIR type
func NewHIRType(span Span, kind HIRTypeKind, data interface{}) *HIRType {
	return &HIRType{
		Span: span,
		Kind: kind,
		Data: data,
	}
}

// NewHIRPattern creates a new HIR pattern
func NewHIRPattern(span Span, hirType *HIRType, kind HIRPatternKind, data interface{}) *HIRPattern {
	return &HIRPattern{
		Span: span,
		Type: hirType,
		Kind: kind,
		Data: data,
	}
}

// ====== HIR Pretty Printing ======

// PrettyPrint returns a formatted string representation of the HIR
func (m *HIRModule) PrettyPrint() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("HIR Module: %s\n", m.Name))

	if len(m.Imports) > 0 {
		sb.WriteString("Imports:\n")
		for _, imp := range m.Imports {
			sb.WriteString(fmt.Sprintf("  import %s\n", imp.ModuleName))
		}
		sb.WriteString("\n")
	}

	if len(m.Types) > 0 {
		sb.WriteString("Types:\n")
		for _, typ := range m.Types {
			sb.WriteString(fmt.Sprintf("  type %s\n", typ.Name))
		}
		sb.WriteString("\n")
	}

	if len(m.Variables) > 0 {
		sb.WriteString("Variables:\n")
		for _, variable := range m.Variables {
			sb.WriteString(fmt.Sprintf("  %s\n", variable.String()))
		}
		sb.WriteString("\n")
	}

	if len(m.Functions) > 0 {
		sb.WriteString("Functions:\n")
		for _, function := range m.Functions {
			sb.WriteString(fmt.Sprintf("  %s\n", function.String()))
		}
	}

	return sb.String()
}
