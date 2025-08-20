// Package hir defines the High-level Intermediate Representation (HIR) for the Orizon programming language.
// Phase 1.4.1: HIR設計と実装 - High-level semantic intermediate representation
// This package provides semantically rich intermediate representation that captures.
// the intent and meaning of the source program while enabling high-level optimizations.
//
// HIR is designed to bridge the gap between AST and lower-level IRs, providing:.
// - Semantic information (types, effects, regions).
// - High-level constructs preservation.
// - Optimization-friendly representation.
// - Debugging and analysis support.
package hir

import (
	"fmt"
	"strings"

	"github.com/orizon-lang/orizon/internal/position"
)

// NodeID uniquely identifies an HIR node within a program.
type NodeID uint64

// ModuleID uniquely identifies a module within a program.
type ModuleID uint64

// TypeID uniquely identifies a type within the type system.
type TypeID uint64

// EffectID uniquely identifies an effect within the effect system.
type EffectID uint64

// RegionID uniquely identifies a memory region within the region system.
type RegionID uint64

// HIRNode is the base interface for all HIR nodes.
// Every HIR node carries semantic information and metadata.
type HIRNode interface {
	// GetID returns the unique identifier for this node.
	GetID() NodeID
	// GetSpan returns the source span covered by this node.
	GetSpan() position.Span
	// GetType returns the type information for this node.
	GetType() TypeInfo
	// GetEffects returns the effect set for this node.
	GetEffects() EffectSet
	// GetRegions returns the region set for this node.
	GetRegions() RegionSet
	// String returns a human-readable representation of the node.
	String() string
	// Accept implements the visitor pattern for HIR traversal.
	Accept(visitor HIRVisitor) interface{}
	// GetChildren returns child nodes for tree traversal.
	GetChildren() []HIRNode
}

// HIRStatement represents all statement nodes in the HIR.
type HIRStatement interface {
	HIRNode
	hirStatementNode() // Marker method to distinguish statements
}

// HIRExpression represents all expression nodes in the HIR.
type HIRExpression interface {
	HIRNode
	hirExpressionNode() // Marker method to distinguish expressions
}

// HIRDeclaration represents all declaration nodes in the HIR.
type HIRDeclaration interface {
	HIRNode
	hirDeclarationNode() // Marker method to distinguish declarations
}

// HIRType represents all type nodes in the HIR.
type HIRType interface {
	HIRNode
	hirTypeNode() // Marker method to distinguish types
}

// TypeInfo contains complete type information for HIR nodes.
type TypeInfo struct {
	VariableID  *int
	StructInfo  *StructLayout
	Effects     EffectSet
	Name        string
	Constraints []TypeConstraint
	Parameters  []TypeInfo
	Fields      []FieldInfo
	Methods     []MethodInfo
	ID          TypeID
	Alignment   int64
	Size        int64
	Kind        TypeKind
	Properties  TypeProperties
}

// 型等価性判定.
func (t TypeInfo) Equals(other TypeInfo) bool {
	if t.Kind != other.Kind || t.Name != other.Name {
		return false
	}

	if len(t.Parameters) != len(other.Parameters) {
		return false
	}

	for i := range t.Parameters {
		if !t.Parameters[i].Equals(other.Parameters[i]) {
			return false
		}
	}

	if len(t.Fields) != len(other.Fields) {
		return false
	}

	for i := range t.Fields {
		if t.Fields[i].Name != other.Fields[i].Name ||
			!t.Fields[i].Type.Equals(other.Fields[i].Type) {
			return false
		}
	}

	return true
}

// 型変換規則（暗黙変換可能か）.
func (t TypeInfo) CanConvertTo(target TypeInfo) bool {
	if t.Equals(target) {
		return true
	}
	// int <-> float 暗黙変換可能.
	if (t.Kind == TypeKindInteger && target.Kind == TypeKindFloat) ||
		(t.Kind == TypeKindFloat && target.Kind == TypeKindInteger) {
		return true
	}
	// void型は何にも変換不可.
	if t.Kind == TypeKindVoid || target.Kind == TypeKindVoid {
		return false
	}
	// 配列型は要素型が変換可能ならOK.
	if t.Kind == TypeKindArray && target.Kind == TypeKindArray {
		if len(t.Parameters) > 0 && len(target.Parameters) > 0 {
			return t.Parameters[0].CanConvertTo(target.Parameters[0])
		}
	}
	// 構造体型は同名・同フィールド型ならOK.
	if t.Kind == TypeKindStruct && target.Kind == TypeKindStruct {
		return t.Equals(target)
	}
	// 関数型はパラメータ・戻り値型が変換可能ならOK.
	if t.Kind == TypeKindFunction && target.Kind == TypeKindFunction {
		if len(t.Parameters) != len(target.Parameters) {
			return false
		}

		for i := range t.Parameters {
			if !t.Parameters[i].CanConvertTo(target.Parameters[i]) {
				return false
			}
		}

		return true
	}

	return false
}

// TypeKind represents the fundamental kind of a type.
type TypeKind int

const (
	TypeKindUnknown TypeKind = iota
	TypeKindInvalid          // For invalid types
	TypeKindVoid
	TypeKindBoolean
	TypeKindInteger
	TypeKindFloat
	TypeKindString
	TypeKindArray
	TypeKindSlice
	TypeKindPointer
	TypeKindFunction
	TypeKindStruct
	TypeKindInterface
	TypeKindGeneric
	TypeKindTypeParameter
	TypeKindVariable // For type variables in inference
	TypeKindTuple    // For tuple types
	TypeKindSkolem   // For skolem constants
	// Advanced type system kinds.
	TypeKindHigherRank  // For rank-N polymorphic types
	TypeKindDependent   // For dependent types
	TypeKindEffect      // For effect types
	TypeKindLinear      // For linear types
	TypeKindRefinement  // For refinement types
	TypeKindType        // For type universe
	TypeKindApplication // For type applications
)

// TypeProperties represents compile-time properties of types.
type TypeProperties struct {
	Immutable   bool // Type is immutable
	Linear      bool // Type has linear/affine usage
	Copyable    bool // Type can be copied
	Movable     bool // Type can be moved
	Droppable   bool // Type can be dropped
	Send        bool // Type can be sent across threads
	Sync        bool // Type can be shared across threads
	ZeroSized   bool // Type has zero size
	Transparent bool // Type is transparent for optimization
}

// FieldInfo represents a field in a struct/record type.
type FieldInfo struct {
	Type    TypeInfo
	Name    string
	Span    position.Span
	Offset  int64
	Private bool
}

// MethodInfo represents a method associated with a type.
type MethodInfo struct {
	Signature TypeInfo
	Receiver  TypeInfo
	Name      string
	Effects   EffectSet
	Span      position.Span
	Static    bool
	Private   bool
}

// TypeConstraint represents a constraint on a type parameter.
type TypeConstraint struct {
	Target    TypeInfo
	Predicate string
	Span      position.Span
	Kind      HirConstraintKind
}

// HirConstraintKind represents the kind of type constraint.
type HirConstraintKind int

const (
	HirConstraintKindImplements HirConstraintKind = iota
	HirConstraintKindExtends
	HirConstraintKindEquals
	HirConstraintKindPredicate
)

// EffectSet represents the set of effects that an HIR node may produce.
type EffectSet struct {
	Effects map[EffectID]Effect
	Pure    bool // True if no effects
}

// Effect represents a computational effect.
type Effect struct {
	Description string
	ID          EffectID
	Kind        EffectKind
	Modality    EffectModality
	Scope       EffectScope
}

// EffectKind represents the kind of effect.
type EffectKind int

const (
	EffectKindMemoryRead EffectKind = iota
	EffectKindMemoryWrite
	EffectKindMemoryAlloc
	EffectKindMemoryFree
	EffectKindIO
	EffectKindNetworkIO
	EffectKindFileIO
	EffectKindSystemCall
	EffectKindException
	EffectKindNonTermination
	EffectKindNonDeterminism
)

// EffectModality represents whether an effect must, may, or cannot occur.
type EffectModality int

const (
	EffectModalityMust EffectModality = iota
	EffectModalityMay
	EffectModalityCannot
)

// EffectScope represents the scope of an effect.
type EffectScope int

const (
	EffectScopeLocal EffectScope = iota
	EffectScopeGlobal
	EffectScopeModule
)

// RegionSet represents the set of memory regions that an HIR node may access.
type RegionSet struct {
	Regions map[RegionID]Region
}

// Region represents a memory region.
type Region struct {
	Lifetime    Lifetime
	ID          RegionID
	Kind        RegionKind
	Size        int64
	Permissions RegionPermissions
}

// RegionKind represents the kind of memory region.
type RegionKind int

const (
	RegionKindStack RegionKind = iota
	RegionKindHeap
	RegionKindStatic
	RegionKindParameter
	RegionKindReturn
)

// Lifetime represents the lifetime of a memory region.
type Lifetime struct {
	Named string
	Start position.Span
	End   position.Span
}

// RegionPermissions represents permissions for a memory region.
type RegionPermissions struct {
	Read    bool
	Write   bool
	Execute bool
	Share   bool
}

// IRMetadata contains metadata for HIR nodes.
type IRMetadata struct {
	Annotations  map[string]interface{}
	SourceFile   string
	Dependencies []NodeID
	Dependents   []NodeID
	DebugInfo    DebugInfo
	OptLevel     int
}

// DebugInfo contains debugging information.
type DebugInfo struct {
	OriginalSource string
	InlineDepth    int
	Inlined        bool
	OptimizedAway  bool
}

// HIRProgram represents a complete HIR program.
type HIRProgram struct {
	Modules    map[ModuleID]*HIRModule
	TypeInfo   *GlobalTypeInfo
	EffectInfo *GlobalEffectInfo
	RegionInfo *GlobalRegionInfo
	Metadata   IRMetadata
	Span       position.Span
	ID         NodeID
}

// HIRModule represents a module in the HIR.
type HIRModule struct {
	Name         string
	Metadata     IRMetadata
	Declarations []HIRDeclaration
	Exports      []string
	Imports      []ImportInfo
	Span         position.Span
	ID           NodeID
	ModuleID     ModuleID
}

// ImportInfo represents an import declaration.
type ImportInfo struct {
	ModuleName string
	Alias      string
	Items      []string // Specific imported items
	Span       position.Span
}

// GlobalTypeInfo maintains global type information.
type GlobalTypeInfo struct {
	Types      map[TypeID]TypeInfo
	Primitives map[string]TypeID
	Interfaces map[string]TypeID
	NextTypeID TypeID
}

// GlobalEffectInfo maintains global effect information.
type GlobalEffectInfo struct {
	Effects      map[EffectID]Effect
	NextEffectID EffectID
}

// GlobalRegionInfo maintains global region information.
type GlobalRegionInfo struct {
	Regions      map[RegionID]Region
	NextRegionID RegionID
}

// HIRVisitor defines the visitor pattern interface for HIR traversal.
type HIRVisitor interface {
	// Program and module visits.
	VisitProgram(node *HIRProgram) interface{}
	VisitModule(node *HIRModule) interface{}

	// Declaration visits.
	VisitFunctionDeclaration(node *HIRFunctionDeclaration) interface{}
	VisitVariableDeclaration(node *HIRVariableDeclaration) interface{}
	VisitTypeDeclaration(node *HIRTypeDeclaration) interface{}
	VisitConstDeclaration(node *HIRConstDeclaration) interface{}

	// Statement visits.
	VisitBlockStatement(node *HIRBlockStatement) interface{}
	VisitExpressionStatement(node *HIRExpressionStatement) interface{}
	VisitReturnStatement(node *HIRReturnStatement) interface{}
	VisitIfStatement(node *HIRIfStatement) interface{}
	VisitWhileStatement(node *HIRWhileStatement) interface{}
	VisitForStatement(node *HIRForStatement) interface{}
	VisitBreakStatement(node *HIRBreakStatement) interface{}
	VisitContinueStatement(node *HIRContinueStatement) interface{}
	VisitAssignStatement(node *HIRAssignStatement) interface{}
	// Exception handling.
	VisitThrowStatement(node *HIRThrowStatement) interface{}
	VisitTryCatchStatement(node *HIRTryCatchStatement) interface{}

	// Expression visits.
	VisitIdentifier(node *HIRIdentifier) interface{}
	VisitLiteral(node *HIRLiteral) interface{}
	VisitBinaryExpression(node *HIRBinaryExpression) interface{}
	VisitUnaryExpression(node *HIRUnaryExpression) interface{}
	VisitCallExpression(node *HIRCallExpression) interface{}
	VisitIndexExpression(node *HIRIndexExpression) interface{}
	VisitFieldExpression(node *HIRFieldExpression) interface{}
	VisitCastExpression(node *HIRCastExpression) interface{}
	VisitArrayExpression(node *HIRArrayExpression) interface{}
	VisitStructExpression(node *HIRStructExpression) interface{}

	// Type visits.
	VisitBasicType(node *HIRBasicType) interface{}
	VisitArrayType(node *HIRArrayType) interface{}
	VisitPointerType(node *HIRPointerType) interface{}
	VisitFunctionType(node *HIRFunctionType) interface{}
	VisitStructType(node *HIRStructType) interface{}
	VisitInterfaceType(node *HIRInterfaceType) interface{}
	VisitGenericType(node *HIRGenericType) interface{}
}

// Helper functions for HIR construction.

// NewHIRProgram creates a new HIR program.
func NewHIRProgram() *HIRProgram {
	return &HIRProgram{
		ID:         generateNodeID(),
		Modules:    make(map[ModuleID]*HIRModule),
		TypeInfo:   NewGlobalTypeInfo(),
		EffectInfo: NewGlobalEffectInfo(),
		RegionInfo: NewGlobalRegionInfo(),
		Metadata:   IRMetadata{},
	}
}

// NewGlobalTypeInfo creates a new global type information structure.
func NewGlobalTypeInfo() *GlobalTypeInfo {
	info := &GlobalTypeInfo{
		Types:      make(map[TypeID]TypeInfo),
		Primitives: make(map[string]TypeID),
		Interfaces: make(map[string]TypeID),
		NextTypeID: 1,
	}

	// Initialize primitive types.
	info.initializePrimitiveTypes()

	return info
}

// NewGlobalEffectInfo creates a new global effect information structure.
func NewGlobalEffectInfo() *GlobalEffectInfo {
	return &GlobalEffectInfo{
		Effects:      make(map[EffectID]Effect),
		NextEffectID: 1,
	}
}

// NewGlobalRegionInfo creates a new global region information structure.
func NewGlobalRegionInfo() *GlobalRegionInfo {
	return &GlobalRegionInfo{
		Regions:      make(map[RegionID]Region),
		NextRegionID: 1,
	}
}

// initializePrimitiveTypes sets up built-in primitive types.
func (gti *GlobalTypeInfo) initializePrimitiveTypes() {
	primitives := []struct {
		name string
		kind TypeKind
		size int64
	}{
		{"void", TypeKindVoid, 0},
		{"bool", TypeKindBoolean, 1},
		{"i8", TypeKindInteger, 1},
		{"i16", TypeKindInteger, 2},
		{"i32", TypeKindInteger, 4},
		{"i64", TypeKindInteger, 8},
		{"u8", TypeKindInteger, 1},
		{"u16", TypeKindInteger, 2},
		{"u32", TypeKindInteger, 4},
		{"u64", TypeKindInteger, 8},
		{"f32", TypeKindFloat, 4},
		{"f64", TypeKindFloat, 8},
		{"string", TypeKindString, -1}, // Variable size
	}

	for _, prim := range primitives {
		typeID := gti.NextTypeID
		gti.NextTypeID++

		typeInfo := TypeInfo{
			ID:   typeID,
			Kind: prim.kind,
			Name: prim.name,
			Size: prim.size,
			Properties: TypeProperties{
				Copyable:    true,
				Movable:     true,
				Droppable:   true,
				Send:        true,
				Sync:        true,
				ZeroSized:   prim.size == 0,
				Transparent: true,
			},
		}

		gti.Types[typeID] = typeInfo
		gti.Primitives[prim.name] = typeID
	}
}

// Node ID generation (simple counter for now).
var nextNodeID NodeID = 1

func generateNodeID() NodeID {
	id := nextNodeID
	nextNodeID++

	return id
}

// Utility functions for effect sets.

// NewEffectSet creates a new empty effect set.
func NewEffectSet() EffectSet {
	return EffectSet{
		Effects: make(map[EffectID]Effect),
		Pure:    true,
	}
}

// AddEffect adds an effect to the effect set.
func (es *EffectSet) AddEffect(effect Effect) {
	es.Effects[effect.ID] = effect
	es.Pure = false
}

// HasEffect checks if the effect set contains a specific effect.
func (es *EffectSet) HasEffect(effectID EffectID) bool {
	_, exists := es.Effects[effectID]

	return exists
}

// Union returns the union of two effect sets.
func (es *EffectSet) Union(other EffectSet) EffectSet {
	result := NewEffectSet()

	for id, effect := range es.Effects {
		result.Effects[id] = effect
	}

	for id, effect := range other.Effects {
		result.Effects[id] = effect
	}

	result.Pure = es.Pure && other.Pure

	return result
}

// Utility functions for region sets.

// NewRegionSet creates a new empty region set.
func NewRegionSet() RegionSet {
	return RegionSet{
		Regions: make(map[RegionID]Region),
	}
}

// AddRegion adds a region to the region set.
func (rs *RegionSet) AddRegion(region Region) {
	rs.Regions[region.ID] = region
}

// HasRegion checks if the region set contains a specific region.
func (rs *RegionSet) HasRegion(regionID RegionID) bool {
	_, exists := rs.Regions[regionID]

	return exists
}

// Union returns the union of two region sets.
func (rs *RegionSet) Union(other RegionSet) RegionSet {
	result := NewRegionSet()

	for id, region := range rs.Regions {
		result.Regions[id] = region
	}

	for id, region := range other.Regions {
		result.Regions[id] = region
	}

	return result
}

// String implementations for debugging.

func (tk TypeKind) String() string {
	switch tk {
	case TypeKindVoid:
		return "void"
	case TypeKindBoolean:
		return "bool"
	case TypeKindInteger:
		return "int"
	case TypeKindFloat:
		return "float"
	case TypeKindString:
		return "string"
	case TypeKindArray:
		return "array"
	case TypeKindSlice:
		return "slice"
	case TypeKindPointer:
		return "pointer"
	case TypeKindFunction:
		return "function"
	case TypeKindStruct:
		return "struct"
	case TypeKindInterface:
		return "interface"
	case TypeKindGeneric:
		return "generic"
	case TypeKindTypeParameter:
		return "type_parameter"
	default:
		return "unknown"
	}
}

func (ti TypeInfo) String() string {
	return fmt.Sprintf("Type{%s: %s}", ti.Name, ti.Kind.String())
}

func (ek EffectKind) String() string {
	switch ek {
	case EffectKindMemoryRead:
		return "memory_read"
	case EffectKindMemoryWrite:
		return "memory_write"
	case EffectKindMemoryAlloc:
		return "memory_alloc"
	case EffectKindMemoryFree:
		return "memory_free"
	case EffectKindIO:
		return "io"
	case EffectKindNetworkIO:
		return "network_io"
	case EffectKindFileIO:
		return "file_io"
	case EffectKindSystemCall:
		return "system_call"
	case EffectKindException:
		return "exception"
	case EffectKindNonTermination:
		return "non_termination"
	case EffectKindNonDeterminism:
		return "non_determinism"
	default:
		return "unknown"
	}
}

func (e Effect) String() string {
	return fmt.Sprintf("Effect{%s: %s}", e.Description, e.Kind.String())
}

func (rk RegionKind) String() string {
	switch rk {
	case RegionKindStack:
		return "stack"
	case RegionKindHeap:
		return "heap"
	case RegionKindStatic:
		return "static"
	case RegionKindParameter:
		return "parameter"
	case RegionKindReturn:
		return "return"
	default:
		return "unknown"
	}
}

func (r Region) String() string {
	return fmt.Sprintf("Region{%s: %d bytes}", r.Kind.String(), r.Size)
}

// Implementation of HIRProgram methods.

func (p *HIRProgram) GetID() NodeID          { return p.ID }
func (p *HIRProgram) GetSpan() position.Span { return p.Span }
func (p *HIRProgram) GetType() TypeInfo {
	// Program doesn't have a type, return void.
	if p.TypeInfo != nil {
		if voidID, exists := p.TypeInfo.Primitives["void"]; exists {
			return p.TypeInfo.Types[voidID]
		}
	}

	return TypeInfo{Kind: TypeKindVoid, Name: "void"}
}
func (p *HIRProgram) GetEffects() EffectSet                 { return NewEffectSet() }
func (p *HIRProgram) GetRegions() RegionSet                 { return NewRegionSet() }
func (p *HIRProgram) Accept(visitor HIRVisitor) interface{} { return visitor.VisitProgram(p) }
func (p *HIRProgram) GetChildren() []HIRNode {
	children := make([]HIRNode, 0, len(p.Modules))
	for _, module := range p.Modules {
		children = append(children, module)
	}

	return children
}

func (p *HIRProgram) String() string {
	var sb strings.Builder

	sb.WriteString("HIRProgram {\n")

	for _, module := range p.Modules {
		sb.WriteString(fmt.Sprintf("  %s\n", module.String()))
	}

	sb.WriteString("}")

	return sb.String()
}

// Implementation of HIRModule methods.

func (m *HIRModule) GetID() NodeID          { return m.ID }
func (m *HIRModule) GetSpan() position.Span { return m.Span }
func (m *HIRModule) GetType() TypeInfo {
	return TypeInfo{Kind: TypeKindVoid, Name: "void"}
}
func (m *HIRModule) GetEffects() EffectSet                 { return NewEffectSet() }
func (m *HIRModule) GetRegions() RegionSet                 { return NewRegionSet() }
func (m *HIRModule) Accept(visitor HIRVisitor) interface{} { return visitor.VisitModule(m) }
func (m *HIRModule) GetChildren() []HIRNode {
	children := make([]HIRNode, len(m.Declarations))
	for i, decl := range m.Declarations {
		children[i] = decl
	}

	return children
}

func (m *HIRModule) String() string {
	return fmt.Sprintf("HIRModule{%s: %d declarations}", m.Name, len(m.Declarations))
}
