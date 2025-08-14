// HIR type implementations for the Orizon programming language
// This file contains concrete implementations of HIR type nodes

package hir

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/position"
)

// =============================================================================
// HIR Type Nodes
// =============================================================================

// HIRBasicType represents a basic/primitive type in HIR
type HIRBasicType struct {
	ID       NodeID
	Kind     TypeKind
	Name     string
	Type     TypeInfo
	Metadata IRMetadata
	Span     position.Span
}

func (bt *HIRBasicType) GetID() NodeID          { return bt.ID }
func (bt *HIRBasicType) GetSpan() position.Span { return bt.Span }
func (bt *HIRBasicType) GetType() TypeInfo      { return bt.Type }
func (bt *HIRBasicType) GetEffects() EffectSet  { return NewEffectSet() }
func (bt *HIRBasicType) GetRegions() RegionSet  { return NewRegionSet() }
func (bt *HIRBasicType) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitBasicType(bt)
}
func (bt *HIRBasicType) GetChildren() []HIRNode {
	return []HIRNode{}
}
func (bt *HIRBasicType) hirTypeNode() {}
func (bt *HIRBasicType) String() string {
	return fmt.Sprintf("HIRBasicType{%s}", bt.Name)
}

// HIRArrayType represents an array type in HIR
type HIRArrayType struct {
	ID          NodeID
	ElementType HIRType
	Size        HIRExpression // Optional size expression
	Type        TypeInfo
	Metadata    IRMetadata
	Span        position.Span
}

func (at *HIRArrayType) GetID() NodeID          { return at.ID }
func (at *HIRArrayType) GetSpan() position.Span { return at.Span }
func (at *HIRArrayType) GetType() TypeInfo      { return at.Type }
func (at *HIRArrayType) GetEffects() EffectSet  { return NewEffectSet() }
func (at *HIRArrayType) GetRegions() RegionSet  { return NewRegionSet() }
func (at *HIRArrayType) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitArrayType(at)
}
func (at *HIRArrayType) GetChildren() []HIRNode {
	children := []HIRNode{at.ElementType}
	if at.Size != nil {
		children = append(children, at.Size)
	}
	return children
}
func (at *HIRArrayType) hirTypeNode() {}
func (at *HIRArrayType) String() string {
	if at.Size != nil {
		return fmt.Sprintf("HIRArrayType{[%s]%s}", at.Size.String(), at.ElementType.String())
	}
	return fmt.Sprintf("HIRArrayType{[]%s}", at.ElementType.String())
}

// HIRPointerType represents a pointer type in HIR
type HIRPointerType struct {
	ID         NodeID
	TargetType HIRType
	Mutable    bool
	Type       TypeInfo
	Metadata   IRMetadata
	Span       position.Span
}

func (pt *HIRPointerType) GetID() NodeID          { return pt.ID }
func (pt *HIRPointerType) GetSpan() position.Span { return pt.Span }
func (pt *HIRPointerType) GetType() TypeInfo      { return pt.Type }
func (pt *HIRPointerType) GetEffects() EffectSet  { return NewEffectSet() }
func (pt *HIRPointerType) GetRegions() RegionSet  { return NewRegionSet() }
func (pt *HIRPointerType) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitPointerType(pt)
}
func (pt *HIRPointerType) GetChildren() []HIRNode {
	return []HIRNode{pt.TargetType}
}
func (pt *HIRPointerType) hirTypeNode() {}
func (pt *HIRPointerType) String() string {
	if pt.Mutable {
		return fmt.Sprintf("HIRPointerType{*mut %s}", pt.TargetType.String())
	}
	return fmt.Sprintf("HIRPointerType{*%s}", pt.TargetType.String())
}

// HIRFunctionType represents a function type in HIR
type HIRFunctionType struct {
	ID         NodeID
	Parameters []HIRType
	ReturnType HIRType
	Effects    EffectSet
	Type       TypeInfo
	Metadata   IRMetadata
	Span       position.Span
}

func (ft *HIRFunctionType) GetID() NodeID          { return ft.ID }
func (ft *HIRFunctionType) GetSpan() position.Span { return ft.Span }
func (ft *HIRFunctionType) GetType() TypeInfo      { return ft.Type }
func (ft *HIRFunctionType) GetEffects() EffectSet  { return ft.Effects }
func (ft *HIRFunctionType) GetRegions() RegionSet  { return NewRegionSet() }
func (ft *HIRFunctionType) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitFunctionType(ft)
}
func (ft *HIRFunctionType) GetChildren() []HIRNode {
	children := make([]HIRNode, len(ft.Parameters)+1)
	for i, param := range ft.Parameters {
		children[i] = param
	}
	children[len(ft.Parameters)] = ft.ReturnType
	return children
}
func (ft *HIRFunctionType) hirTypeNode() {}
func (ft *HIRFunctionType) String() string {
	return fmt.Sprintf("HIRFunctionType{fn(%d) -> %s}", len(ft.Parameters), ft.ReturnType.String())
}

// HIRStructType represents a struct type in HIR
type HIRStructType struct {
	ID       NodeID
	Name     string
	Fields   []HIRStructField
	Type     TypeInfo
	Metadata IRMetadata
	Span     position.Span
}

// HIRStructField represents a field in a struct type
type HIRStructField struct {
	Name    string
	Type    HIRType
	Private bool
	Span    position.Span
}

func (st *HIRStructType) GetID() NodeID          { return st.ID }
func (st *HIRStructType) GetSpan() position.Span { return st.Span }
func (st *HIRStructType) GetType() TypeInfo      { return st.Type }
func (st *HIRStructType) GetEffects() EffectSet  { return NewEffectSet() }
func (st *HIRStructType) GetRegions() RegionSet  { return NewRegionSet() }
func (st *HIRStructType) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitStructType(st)
}
func (st *HIRStructType) GetChildren() []HIRNode {
	children := make([]HIRNode, len(st.Fields))
	for i, field := range st.Fields {
		children[i] = field.Type
	}
	return children
}
func (st *HIRStructType) hirTypeNode() {}
func (st *HIRStructType) String() string {
	return fmt.Sprintf("HIRStructType{%s: %d fields}", st.Name, len(st.Fields))
}

// HIRInterfaceType represents an interface type in HIR
type HIRInterfaceType struct {
	ID       NodeID
	Name     string
	Methods  []HIRInterfaceMethod
	Type     TypeInfo
	Metadata IRMetadata
	Span     position.Span
}

// HIRInterfaceMethod represents a method in an interface type
type HIRInterfaceMethod struct {
	Name      string
	Signature HIRType
	Effects   EffectSet
	Span      position.Span
}

func (it *HIRInterfaceType) GetID() NodeID          { return it.ID }
func (it *HIRInterfaceType) GetSpan() position.Span { return it.Span }
func (it *HIRInterfaceType) GetType() TypeInfo      { return it.Type }
func (it *HIRInterfaceType) GetEffects() EffectSet  { return NewEffectSet() }
func (it *HIRInterfaceType) GetRegions() RegionSet  { return NewRegionSet() }
func (it *HIRInterfaceType) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitInterfaceType(it)
}
func (it *HIRInterfaceType) GetChildren() []HIRNode {
	children := make([]HIRNode, len(it.Methods))
	for i, method := range it.Methods {
		children[i] = method.Signature
	}
	return children
}
func (it *HIRInterfaceType) hirTypeNode() {}
func (it *HIRInterfaceType) String() string {
	return fmt.Sprintf("HIRInterfaceType{%s: %d methods}", it.Name, len(it.Methods))
}

// HIRGenericType represents a generic type in HIR
type HIRGenericType struct {
	ID          NodeID
	Name        string
	BaseType    HIRType
	TypeArgs    []HIRType
	Constraints []TypeConstraint
	Type        TypeInfo
	Metadata    IRMetadata
	Span        position.Span
}

func (gt *HIRGenericType) GetID() NodeID          { return gt.ID }
func (gt *HIRGenericType) GetSpan() position.Span { return gt.Span }
func (gt *HIRGenericType) GetType() TypeInfo      { return gt.Type }
func (gt *HIRGenericType) GetEffects() EffectSet  { return NewEffectSet() }
func (gt *HIRGenericType) GetRegions() RegionSet  { return NewRegionSet() }
func (gt *HIRGenericType) Accept(visitor HIRVisitor) interface{} {
	return visitor.VisitGenericType(gt)
}
func (gt *HIRGenericType) GetChildren() []HIRNode {
	children := []HIRNode{gt.BaseType}
	for _, arg := range gt.TypeArgs {
		children = append(children, arg)
	}
	return children
}
func (gt *HIRGenericType) hirTypeNode() {}
func (gt *HIRGenericType) String() string {
	return fmt.Sprintf("HIRGenericType{%s<%d>}", gt.Name, len(gt.TypeArgs))
}

// =============================================================================
// HIR Type Utilities
// =============================================================================

// HIRTypeBuilder provides a builder for creating HIR types
type HIRTypeBuilder struct {
	program *HIRProgram
}

// NewHIRTypeBuilder creates a new HIR type builder
func NewHIRTypeBuilder(program *HIRProgram) *HIRTypeBuilder {
	return &HIRTypeBuilder{program: program}
}

// BuildBasicType creates a basic type HIR node
func (tb *HIRTypeBuilder) BuildBasicType(name string, span position.Span) *HIRBasicType {
	var typeInfo TypeInfo
	var kind TypeKind

	// Look up the type in the global type info
	if tb.program.TypeInfo != nil {
		if typeID, exists := tb.program.TypeInfo.Primitives[name]; exists {
			typeInfo = tb.program.TypeInfo.Types[typeID]
			kind = typeInfo.Kind
		}
	}

	// Fallback for unknown types
	if typeInfo.Name == "" {
		kind = TypeKindUnknown
		typeInfo = TypeInfo{
			ID:   TypeID(generateNodeID()),
			Kind: kind,
			Name: name,
			Size: -1,
		}
	}

	return &HIRBasicType{
		ID:       generateNodeID(),
		Kind:     kind,
		Name:     name,
		Type:     typeInfo,
		Metadata: IRMetadata{},
		Span:     span,
	}
}

// BuildArrayType creates an array type HIR node
func (tb *HIRTypeBuilder) BuildArrayType(elementType HIRType, size HIRExpression, span position.Span) *HIRArrayType {
	typeInfo := TypeInfo{
		ID:         TypeID(generateNodeID()),
		Kind:       TypeKindArray,
		Name:       fmt.Sprintf("[]%s", elementType.GetType().Name),
		Size:       -1, // Dynamic size
		Parameters: []TypeInfo{elementType.GetType()},
		Properties: TypeProperties{
			Copyable:  elementType.GetType().Properties.Copyable,
			Movable:   true,
			Droppable: true,
			Send:      elementType.GetType().Properties.Send,
			Sync:      elementType.GetType().Properties.Sync,
		},
	}

	return &HIRArrayType{
		ID:          generateNodeID(),
		ElementType: elementType,
		Size:        size,
		Type:        typeInfo,
		Metadata:    IRMetadata{},
		Span:        span,
	}
}

// BuildPointerType creates a pointer type HIR node
func (tb *HIRTypeBuilder) BuildPointerType(targetType HIRType, mutable bool, span position.Span) *HIRPointerType {
	typeInfo := TypeInfo{
		ID:         TypeID(generateNodeID()),
		Kind:       TypeKindPointer,
		Name:       fmt.Sprintf("*%s", targetType.GetType().Name),
		Size:       8, // Assume 64-bit pointers
		Parameters: []TypeInfo{targetType.GetType()},
		Properties: TypeProperties{
			Copyable:  true,
			Movable:   true,
			Droppable: true,
			Send:      targetType.GetType().Properties.Send,
			Sync:      targetType.GetType().Properties.Sync,
		},
	}

	return &HIRPointerType{
		ID:         generateNodeID(),
		TargetType: targetType,
		Mutable:    mutable,
		Type:       typeInfo,
		Metadata:   IRMetadata{},
		Span:       span,
	}
}

// BuildFunctionType creates a function type HIR node
func (tb *HIRTypeBuilder) BuildFunctionType(parameters []HIRType, returnType HIRType, effects EffectSet, span position.Span) *HIRFunctionType {
	typeInfo := TypeInfo{
		ID:   TypeID(generateNodeID()),
		Kind: TypeKindFunction,
		Name: fmt.Sprintf("fn(%d) -> %s", len(parameters), returnType.GetType().Name),
		Size: 8, // Function pointer size
		Parameters: func() []TypeInfo {
			out := make([]TypeInfo, 0, len(parameters)+1)
			for _, p := range parameters {
				out = append(out, p.GetType())
			}
			out = append(out, returnType.GetType())
			return out
		}(),
		Properties: TypeProperties{
			Copyable:  true,
			Movable:   true,
			Droppable: true,
			Send:      effects.Pure,
			Sync:      effects.Pure,
		},
	}

	return &HIRFunctionType{
		ID:         generateNodeID(),
		Parameters: parameters,
		ReturnType: returnType,
		Effects:    effects,
		Type:       typeInfo,
		Metadata:   IRMetadata{},
		Span:       span,
	}
}

// BuildStructType creates a struct type HIR node
func (tb *HIRTypeBuilder) BuildStructType(name string, fields []HIRStructField, span position.Span) *HIRStructType {
	var totalSize int64 = 0
	fieldInfos := make([]FieldInfo, len(fields))

	for i, field := range fields {
		fieldType := field.Type.GetType()
		fieldInfos[i] = FieldInfo{
			Name:    field.Name,
			Type:    fieldType,
			Offset:  totalSize,
			Private: field.Private,
			Span:    field.Span,
		}
		if fieldType.Size > 0 {
			totalSize += fieldType.Size
		}
	}

	typeInfo := TypeInfo{
		ID:     TypeID(generateNodeID()),
		Kind:   TypeKindStruct,
		Name:   name,
		Size:   totalSize,
		Fields: fieldInfos,
		Properties: TypeProperties{
			Copyable:  true, // Assume copyable unless proven otherwise
			Movable:   true,
			Droppable: true,
			Send:      true,
			Sync:      true,
		},
	}

	return &HIRStructType{
		ID:       generateNodeID(),
		Name:     name,
		Fields:   fields,
		Type:     typeInfo,
		Metadata: IRMetadata{},
		Span:     span,
	}
}

// BuildInterfaceType creates an interface type HIR node
func (tb *HIRTypeBuilder) BuildInterfaceType(name string, methods []HIRInterfaceMethod, span position.Span) *HIRInterfaceType {
	methodInfos := make([]MethodInfo, len(methods))

	for i, method := range methods {
		methodInfos[i] = MethodInfo{
			Name:      method.Name,
			Signature: method.Signature.GetType(),
			Static:    false,
			Private:   false,
			Span:      method.Span,
		}
	}

	typeInfo := TypeInfo{
		ID:      TypeID(generateNodeID()),
		Kind:    TypeKindInterface,
		Name:    name,
		Size:    16, // Interface typically has vtable pointer + data pointer
		Methods: methodInfos,
		Properties: TypeProperties{
			Copyable:  false, // Interfaces are typically not copyable
			Movable:   true,
			Droppable: true,
			Send:      false,
			Sync:      false,
		},
	}

	return &HIRInterfaceType{
		ID:       generateNodeID(),
		Name:     name,
		Methods:  methods,
		Type:     typeInfo,
		Metadata: IRMetadata{},
		Span:     span,
	}
}

// BuildGenericType creates a generic type HIR node
func (tb *HIRTypeBuilder) BuildGenericType(name string, baseType HIRType, typeArgs []HIRType, constraints []TypeConstraint, span position.Span) *HIRGenericType {
	typeInfo := TypeInfo{
		ID:          TypeID(generateNodeID()),
		Kind:        TypeKindGeneric,
		Name:        name,
		Size:        -1, // Size depends on instantiation
		Constraints: constraints,
		Properties: TypeProperties{
			Copyable:  false, // Generic types have unknown properties
			Movable:   true,
			Droppable: true,
			Send:      false,
			Sync:      false,
		},
	}

	return &HIRGenericType{
		ID:          generateNodeID(),
		Name:        name,
		BaseType:    baseType,
		TypeArgs:    typeArgs,
		Constraints: constraints,
		Type:        typeInfo,
		Metadata:    IRMetadata{},
		Span:        span,
	}
}

// =============================================================================
// HIR Type Utilities and Helpers
// =============================================================================

// IsAssignableTo checks if one type can be assigned to another
func IsAssignableTo(from, to TypeInfo) bool {
	// Same type
	if from.ID == to.ID {
		return true
	}

	// Same kind and compatible properties
	if from.Kind == to.Kind {
		switch from.Kind {
		case TypeKindInteger, TypeKindFloat:
			// Allow widening conversions
			return from.Size <= to.Size
		case TypeKindString, TypeKindBoolean, TypeKindVoid:
			return true
		case TypeKindPointer:
			// Pointer compatibility rules would go here
			return true
		default:
			return false
		}
	}

	return false
}

// GetCommonType finds the common type for two types (for binary operations)
func GetCommonType(left, right TypeInfo) TypeInfo {
	// Same type
	if left.ID == right.ID {
		return left
	}

	// Numeric promotion rules
	if left.Kind == TypeKindInteger && right.Kind == TypeKindInteger {
		if left.Size >= right.Size {
			return left
		}
		return right
	}

	if left.Kind == TypeKindFloat && right.Kind == TypeKindFloat {
		if left.Size >= right.Size {
			return left
		}
		return right
	}

	if left.Kind == TypeKindFloat && right.Kind == TypeKindInteger {
		return left
	}

	if left.Kind == TypeKindInteger && right.Kind == TypeKindFloat {
		return right
	}

	// Default to the left type
	return left
}

// CalculateStructSize calculates the size of a struct with proper alignment
func CalculateStructSize(fields []FieldInfo) int64 {
	var size int64 = 0
	var maxAlignment int64 = 1

	for _, field := range fields {
		// Calculate alignment for this field
		fieldAlignment := field.Type.Alignment
		if fieldAlignment <= 0 {
			fieldAlignment = field.Type.Size
		}
		if fieldAlignment <= 0 {
			fieldAlignment = 1
		}

		// Track maximum alignment
		if fieldAlignment > maxAlignment {
			maxAlignment = fieldAlignment
		}

		// Align the current offset
		if size%fieldAlignment != 0 {
			size += fieldAlignment - (size % fieldAlignment)
		}

		// Add field size
		if field.Type.Size > 0 {
			size += field.Type.Size
		}
	}

	// Final struct alignment
	if size%maxAlignment != 0 {
		size += maxAlignment - (size % maxAlignment)
	}

	return size
}

// =============================================================================
// 複合型システム: Phase 2.1.2 実装
// =============================================================================

// CompoundTypeKind represents different kinds of compound types
type CompoundTypeKind int

const (
	CompoundTypeStruct CompoundTypeKind = iota
	CompoundTypeEnum
	CompoundTypeTuple
	CompoundTypeUnion
)

// StructLayout represents memory layout information for struct types
type StructLayout struct {
	Size       int64                  // Total size in bytes
	Alignment  int64                  // Required alignment
	Fields     []FieldLayout          // Field layout information
	Extensions map[string]interface{} // Extensions for advanced type systems
}

// FieldLayout represents layout information for a single field
type FieldLayout struct {
	Name      string
	Offset    int64
	Size      int64
	Alignment int64
}

// EnumVariant represents a variant in an algebraic data type/enum
type EnumVariant struct {
	Name   string
	Tag    int64        // Discriminant value
	Fields []FieldInfo  // Associated data fields (if any)
	Layout StructLayout // Memory layout for this variant
}

// AlgebraicDataType represents enum/union types with variants
type AlgebraicDataType struct {
	Name      string
	Variants  []EnumVariant
	TagSize   int64 // Size of discriminant tag
	TotalSize int64 // Total size including largest variant
	Alignment int64 // Required alignment
}

// TupleLayout represents layout for tuple types
type TupleLayout struct {
	ElementLayouts []FieldLayout
	TotalSize      int64
	Alignment      int64
}

// CalculateStructLayout computes complete layout information for a struct
func CalculateStructLayout(fields []FieldInfo) StructLayout {
	var layout StructLayout
	var currentOffset int64 = 0
	var maxAlignment int64 = 1

	layout.Fields = make([]FieldLayout, len(fields))

	for i, field := range fields {
		// Get field alignment
		fieldAlignment := field.Type.Alignment
		if fieldAlignment <= 0 {
			fieldAlignment = field.Type.Size
		}
		if fieldAlignment <= 0 {
			fieldAlignment = 1
		}

		// Track maximum alignment
		if fieldAlignment > maxAlignment {
			maxAlignment = fieldAlignment
		}

		// Align current offset
		if currentOffset%fieldAlignment != 0 {
			currentOffset += fieldAlignment - (currentOffset % fieldAlignment)
		}

		// Store field layout
		layout.Fields[i] = FieldLayout{
			Name:      field.Name,
			Offset:    currentOffset,
			Size:      field.Type.Size,
			Alignment: fieldAlignment,
		}

		// Advance offset
		currentOffset += field.Type.Size
	}

	// Final struct padding
	if currentOffset%maxAlignment != 0 {
		currentOffset += maxAlignment - (currentOffset % maxAlignment)
	}

	layout.Size = currentOffset
	layout.Alignment = maxAlignment
	return layout
}

// CalculateEnumLayout computes layout for algebraic data type/enum
func CalculateEnumLayout(variants []EnumVariant, tagSize int64) AlgebraicDataType {
	var adt AlgebraicDataType
	adt.Variants = variants
	adt.TagSize = tagSize

	var maxVariantSize int64 = 0
	var maxAlignment int64 = tagSize

	// Calculate layout for each variant
	for i := range adt.Variants {
		variant := &adt.Variants[i]
		if len(variant.Fields) > 0 {
			variant.Layout = CalculateStructLayout(variant.Fields)
			if variant.Layout.Size > maxVariantSize {
				maxVariantSize = variant.Layout.Size
			}
			if variant.Layout.Alignment > maxAlignment {
				maxAlignment = variant.Layout.Alignment
			}
		}
	}

	// Total size = tag + largest variant + padding
	adt.TotalSize = tagSize + maxVariantSize
	if adt.TotalSize%maxAlignment != 0 {
		adt.TotalSize += maxAlignment - (adt.TotalSize % maxAlignment)
	}
	adt.Alignment = maxAlignment

	return adt
}

// CalculateTupleLayout computes layout for tuple types
func CalculateTupleLayout(elementTypes []TypeInfo) TupleLayout {
	var layout TupleLayout
	var currentOffset int64 = 0
	var maxAlignment int64 = 1

	layout.ElementLayouts = make([]FieldLayout, len(elementTypes))

	for i, elemType := range elementTypes {
		elemAlignment := elemType.Alignment
		if elemAlignment <= 0 {
			elemAlignment = elemType.Size
		}
		if elemAlignment <= 0 {
			elemAlignment = 1
		}

		if elemAlignment > maxAlignment {
			maxAlignment = elemAlignment
		}

		// Align current offset
		if currentOffset%elemAlignment != 0 {
			currentOffset += elemAlignment - (currentOffset % elemAlignment)
		}

		layout.ElementLayouts[i] = FieldLayout{
			Name:      fmt.Sprintf("_%d", i), // Tuple elements are accessed by index
			Offset:    currentOffset,
			Size:      elemType.Size,
			Alignment: elemAlignment,
		}

		currentOffset += elemType.Size
	}

	// Final tuple padding
	if currentOffset%maxAlignment != 0 {
		currentOffset += maxAlignment - (currentOffset % maxAlignment)
	}

	layout.TotalSize = currentOffset
	layout.Alignment = maxAlignment
	return layout
}

// CreateStructType creates a new struct TypeInfo with computed layout
func CreateStructType(name string, fields []FieldInfo) TypeInfo {
	layout := CalculateStructLayout(fields)

	return TypeInfo{
		Kind:      TypeKindStruct,
		Name:      name,
		Size:      layout.Size,
		Alignment: layout.Alignment,
		Fields:    fields,
	}
}

// CreateEnumType creates a new enum TypeInfo (algebraic data type)
func CreateEnumType(name string, variants []EnumVariant) TypeInfo {
	layout := CalculateEnumLayout(variants, 8) // 8-byte tag by default

	// Convert variants to Parameters for storage in TypeInfo
	var parameters []TypeInfo
	for _, variant := range variants {
		variantType := TypeInfo{
			Kind:      TypeKindStruct, // Each variant is like a struct
			Name:      variant.Name,
			Size:      variant.Layout.Size,
			Alignment: variant.Layout.Alignment,
			Fields:    variant.Fields,
		}
		parameters = append(parameters, variantType)
	}

	return TypeInfo{
		Kind:       TypeKindGeneric, // Use Generic kind for now, will add Enum kind later
		Name:       name,
		Size:       layout.TotalSize,
		Alignment:  layout.Alignment,
		Parameters: parameters,
	}
}

// CreateTupleType creates a new tuple TypeInfo
func CreateTupleType(name string, elementTypes []TypeInfo) TypeInfo {
	layout := CalculateTupleLayout(elementTypes)

	return TypeInfo{
		Kind:       TypeKindGeneric, // Use Generic kind for now, will add Tuple kind later
		Name:       name,
		Size:       layout.TotalSize,
		Alignment:  layout.Alignment,
		Parameters: elementTypes,
	}
}

// =============================================================================
// 関数型システム: Phase 2.1.3 実装
// =============================================================================

// FunctionSignature represents a complete function signature
type FunctionSignature struct {
	Name        string           // Function name
	Parameters  []Parameter      // Function parameters
	ReturnType  TypeInfo         // Return type
	TypeParams  []TypeParameter  // Generic type parameters
	Constraints []TypeConstraint // Type constraints on generics
	Effects     EffectSet        // Side effects this function may have
	IsVariadic  bool             // Whether function accepts variable arguments
	IsAsync     bool             // Whether function is asynchronous
	IsPure      bool             // Whether function is pure (no side effects)
}

// Parameter represents a function parameter
type Parameter struct {
	Name    string         // Parameter name
	Type    TypeInfo       // Parameter type
	Default *HIRExpression // Default value (if any)
	IsRef   bool           // Pass by reference
	IsMut   bool           // Mutable parameter
}

// TypeParameter represents a generic type parameter
type TypeParameter struct {
	Name        string           // Type parameter name (e.g., T, U)
	Constraints []TypeConstraint // Constraints on this type parameter
	Variance    Variance         // Covariance/contravariance
	Bounds      []TypeInfo       // Upper/lower bounds
}

// Additional constraint kinds for function types
const (
	ConstraintNumeric    HirConstraintKind = iota + 4 // Must be numeric type (continuing from hir.go)
	ConstraintComparable                              // Must support comparison
	ConstraintCopyable                                // Must be copyable
)

// Variance represents type parameter variance
type Variance int

const (
	VarianceInvariant     Variance = iota // Invariant (default)
	VarianceCovariant                     // Covariant (+T)
	VarianceContravariant                 // Contravariant (-T)
)

// ClosureEnvironment represents captured variables in a closure
type ClosureEnvironment struct {
	CapturedVars []CapturedVariable // Variables captured from outer scope
	CaptureMode  CaptureMode        // How variables are captured
}

// CapturedVariable represents a variable captured by a closure
type CapturedVariable struct {
	Name     string        // Variable name
	Type     TypeInfo      // Variable type
	Mode     CaptureMode   // How this variable is captured
	Original HIRExpression // Reference to original variable
}

// CaptureMode represents how variables are captured in closures
type CaptureMode int

const (
	CaptureByValue CaptureMode = iota // Copy the value
	CaptureByRef                      // Capture by reference
	CaptureByMove                     // Move ownership
)

// ClosureType represents a closure type with environment
type ClosureType struct {
	Signature   FunctionSignature  // Function signature
	Environment ClosureEnvironment // Captured environment
	Layout      ClosureLayout      // Memory layout information
}

// ClosureLayout represents memory layout for closures
type ClosureLayout struct {
	FunctionPtr   int64           // Offset to function pointer
	Environment   int64           // Offset to environment data
	CaptureLayout []CaptureLayout // Layout for each captured variable
	TotalSize     int64           // Total closure size
	Alignment     int64           // Required alignment
}

// CaptureLayout represents layout for a captured variable
type CaptureLayout struct {
	Offset    int64 // Offset within environment
	Size      int64 // Size of captured data
	Alignment int64 // Alignment requirement
}

// PartialApplication represents a partially applied function
type PartialApplication struct {
	OriginalFunc    FunctionSignature // Original function
	AppliedArgs     []HIRExpression   // Already applied arguments
	RemainingParams []Parameter       // Parameters still to be applied
	ResultType      TypeInfo          // Type after partial application
}

// CreateFunctionType creates a function TypeInfo from signature
func CreateFunctionType(signature FunctionSignature) TypeInfo {
	// Convert parameters to TypeInfo Parameters slice
	var paramTypes []TypeInfo
	for _, param := range signature.Parameters {
		paramTypes = append(paramTypes, param.Type)
	}
	// Add return type as last parameter
	paramTypes = append(paramTypes, signature.ReturnType)

	// Compute function pointer size (typically 8 bytes on 64-bit systems)
	functionSize := int64(8)

	return TypeInfo{
		Kind:       TypeKindFunction,
		Name:       signature.Name,
		Size:       functionSize,
		Alignment:  8,
		Parameters: paramTypes,
		Properties: TypeProperties{
			Copyable:  true,
			Movable:   true,
			Droppable: true,
		},
	}
}

// CreateClosureType creates a closure TypeInfo with environment
func CreateClosureType(signature FunctionSignature, environment ClosureEnvironment) TypeInfo {
	layout := CalculateClosureLayout(environment)

	// Convert to TypeInfo - use name to encode closure info
	closureName := fmt.Sprintf("closure<%s>", signature.Name)

	// Parameters include function signature + environment info
	var paramTypes []TypeInfo
	for _, param := range signature.Parameters {
		paramTypes = append(paramTypes, param.Type)
	}
	paramTypes = append(paramTypes, signature.ReturnType)

	return TypeInfo{
		Kind:       TypeKindFunction, // Closures are still function types
		Name:       closureName,
		Size:       layout.TotalSize,
		Alignment:  layout.Alignment,
		Parameters: paramTypes,
		Properties: TypeProperties{
			Copyable:  false, // Closures may capture non-copyable data
			Movable:   true,
			Droppable: true,
		},
	}
}

// CalculateClosureLayout computes memory layout for closure
func CalculateClosureLayout(env ClosureEnvironment) ClosureLayout {
	var layout ClosureLayout
	currentOffset := int64(8) // Start after function pointer
	maxAlignment := int64(8)  // Function pointer alignment

	layout.FunctionPtr = 0
	layout.Environment = 8
	layout.CaptureLayout = make([]CaptureLayout, len(env.CapturedVars))

	for i, captured := range env.CapturedVars {
		varAlignment := captured.Type.Alignment
		if varAlignment <= 0 {
			varAlignment = captured.Type.Size
		}
		if varAlignment <= 0 {
			varAlignment = 1
		}

		if varAlignment > maxAlignment {
			maxAlignment = varAlignment
		}

		// Align current offset
		if currentOffset%varAlignment != 0 {
			currentOffset += varAlignment - (currentOffset % varAlignment)
		}

		layout.CaptureLayout[i] = CaptureLayout{
			Offset:    currentOffset,
			Size:      captured.Type.Size,
			Alignment: varAlignment,
		}

		currentOffset += captured.Type.Size
	}

	// Final alignment
	if currentOffset%maxAlignment != 0 {
		currentOffset += maxAlignment - (currentOffset % maxAlignment)
	}

	layout.TotalSize = currentOffset
	layout.Alignment = maxAlignment
	return layout
}

// CreatePartialApplication creates a partially applied function type
func CreatePartialApplication(original FunctionSignature, appliedArgs []HIRExpression) PartialApplication {
	// Determine which parameters are already applied
	remainingParams := make([]Parameter, 0)
	appliedCount := len(appliedArgs)

	for i := appliedCount; i < len(original.Parameters); i++ {
		remainingParams = append(remainingParams, original.Parameters[i])
	}

	// Create new function type for the partial application
	var resultType TypeInfo
	if len(remainingParams) == 0 {
		// Fully applied - return the return type
		resultType = original.ReturnType
	} else {
		// Partially applied - return a new function type
		newSignature := FunctionSignature{
			Name:        fmt.Sprintf("partial<%s>", original.Name),
			Parameters:  remainingParams,
			ReturnType:  original.ReturnType,
			TypeParams:  original.TypeParams,
			Constraints: original.Constraints,
		}
		resultType = CreateFunctionType(newSignature)
	}

	return PartialApplication{
		OriginalFunc:    original,
		AppliedArgs:     appliedArgs,
		RemainingParams: remainingParams,
		ResultType:      resultType,
	}
}

// IsCompatibleSignature checks if two function signatures are compatible
func IsCompatibleSignature(from, to FunctionSignature) bool {
	// Same number of parameters
	if len(from.Parameters) != len(to.Parameters) {
		return false
	}

	// Parameter types must be compatible (contravariant)
	for i := range from.Parameters {
		if !to.Parameters[i].Type.CanConvertTo(from.Parameters[i].Type) {
			return false
		}
	}

	// Return type must be compatible (covariant)
	return from.ReturnType.CanConvertTo(to.ReturnType)
}

// CanPartiallyApply checks if a function can be partially applied with given arguments
func CanPartiallyApply(signature FunctionSignature, args []HIRExpression) bool {
	if len(args) > len(signature.Parameters) {
		return false // Too many arguments
	}

	// Check that provided arguments match expected parameter types
	for i, arg := range args {
		if i >= len(signature.Parameters) {
			return false
		}

		argType := arg.GetType()
		paramType := signature.Parameters[i].Type

		if !argType.CanConvertTo(paramType) {
			return false
		}
	}

	return true
}
