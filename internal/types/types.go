// Basic type system implementation for Orizon language
// This module provides the foundation for all type operations

package types

import (
	"fmt"
	"strings"
)

// ====== Core Type System ======

// TypeKind represents the kind of a type in the Orizon type system
type TypeKind int

const (
	// Primitive types
	TypeKindVoid TypeKind = iota
	TypeKindBool
	TypeKindInt8
	TypeKindInt16
	TypeKindInt32
	TypeKindInt64
	TypeKindUint8
	TypeKindUint16
	TypeKindUint32
	TypeKindUint64
	TypeKindFloat32
	TypeKindFloat64
	TypeKindChar
	TypeKindString

	// Compound types
	TypeKindArray
	TypeKindSlice
	TypeKindPointer
	TypeKindStruct
	TypeKindEnum
	TypeKindUnion
	TypeKindTuple
	TypeKindFunction
	TypeKindChannel

	// Advanced types
	TypeKindGeneric
	TypeKindTypeVar
	TypeKindRefinement
	TypeKindLinear
	TypeKindEffect
	TypeKindDependent
	TypeKindTrait

	// Special types
	TypeKindAny
	TypeKindNever
	TypeKindUnknown
)

// String returns the string representation of a TypeKind
func (tk TypeKind) String() string {
	switch tk {
	case TypeKindVoid:
		return "void"
	case TypeKindBool:
		return "bool"
	case TypeKindInt8:
		return "int8"
	case TypeKindInt16:
		return "int16"
	case TypeKindInt32:
		return "int32"
	case TypeKindInt64:
		return "int64"
	case TypeKindUint8:
		return "uint8"
	case TypeKindUint16:
		return "uint16"
	case TypeKindUint32:
		return "uint32"
	case TypeKindUint64:
		return "uint64"
	case TypeKindFloat32:
		return "float32"
	case TypeKindFloat64:
		return "float64"
	case TypeKindChar:
		return "char"
	case TypeKindString:
		return "string"
	case TypeKindArray:
		return "array"
	case TypeKindSlice:
		return "slice"
	case TypeKindPointer:
		return "pointer"
	case TypeKindStruct:
		return "struct"
	case TypeKindEnum:
		return "enum"
	case TypeKindUnion:
		return "union"
	case TypeKindTuple:
		return "tuple"
	case TypeKindFunction:
		return "function"
	case TypeKindChannel:
		return "channel"
	case TypeKindGeneric:
		return "generic"
	case TypeKindTypeVar:
		return "typevar"
	case TypeKindRefinement:
		return "refinement"
	case TypeKindLinear:
		return "linear"
	case TypeKindEffect:
		return "effect"
	case TypeKindDependent:
		return "dependent"
	case TypeKindTrait:
		return "trait"
	case TypeKindAny:
		return "any"
	case TypeKindNever:
		return "never"
	case TypeKindUnknown:
		return "unknown"
	default:
		return "invalid"
	}
}

// Type represents a type in the Orizon type system
type Type struct {
	Kind TypeKind
	Size int         // Size in bytes, 0 for dynamic/unknown size
	Data interface{} // Type-specific data
}

// ====== Primitive Types ======

// PrimitiveType represents a primitive type
type PrimitiveType struct {
	Name   string
	Size   int
	Signed bool // For integer types
}

// ====== Compound Types ======

// ArrayType represents a fixed-size array type
type ArrayType struct {
	ElementType *Type
	Length      int
}

// SliceType represents a dynamic slice type
type SliceType struct {
	ElementType *Type
}

// PointerType represents a pointer type
type PointerType struct {
	PointeeType *Type
	IsNullable  bool
}

// StructType represents a struct type
type StructType struct {
	Name   string
	Fields []StructField
}

// StructField represents a field in a struct
type StructField struct {
	Name string
	Type *Type
	Tag  string // Optional metadata
}

// EnumType represents an enumeration type
type EnumType struct {
	Name     string
	Variants []EnumVariant
}

// EnumVariant represents a variant in an enum
type EnumVariant struct {
	Name  string
	Value interface{} // Optional associated value
	Type  *Type       // Optional associated type
}

// UnionType represents a union type
type UnionType struct {
	Name    string
	Members []UnionMember
}

// UnionMember represents a member of a union
type UnionMember struct {
	Name string
	Type *Type
}

// TupleType represents a tuple type
type TupleType struct {
	Elements []*Type
}

// FunctionType represents a function type
type FunctionType struct {
	Parameters []*Type
	ReturnType *Type
	IsVariadic bool
	IsAsync    bool
}

// ChannelType represents a channel type for concurrency
type ChannelType struct {
	ElementType *Type
	Direction   ChannelDirection
	Buffered    bool
	BufferSize  int
}

// ChannelDirection represents the direction of a channel
type ChannelDirection int

const (
	ChannelBidirectional ChannelDirection = iota
	ChannelSendOnly
	ChannelReceiveOnly
)

// ====== Advanced Types ======

// GenericType represents a generic type parameter
type GenericType struct {
	Name        string
	Constraints []*Type
	Variance    Variance
}

// Variance represents the variance of a generic type parameter
type Variance int

const (
	VarianceInvariant Variance = iota
	VarianceCovariant
	VarianceContravariant
)

// TypeVar represents a type variable during type inference
type TypeVar struct {
	ID          int
	Name        string
	Constraints []*Type
	Bound       *Type // Set during unification
}

// LinearType represents a linear type (used exactly once)
type LinearType struct {
	BaseType   *Type
	UsageCount int
	IsConsumed bool
}

// EffectType represents an effect type
type EffectType struct {
	BaseType *Type
	Effects  []Effect
}

// Effect represents a computational effect
type Effect struct {
	Name       string
	Parameters []string
	Handler    string // Optional handler
}

// DependentType represents a dependent type
type DependentType struct {
	BaseType   *Type
	Parameters []DependentParam
	Constraint string
}

// DependentParam represents a parameter in a dependent type
type DependentParam struct {
	Name string
	Type *Type
}

// ====== Type Construction Functions ======

// NewPrimitiveType creates a new primitive type
func NewPrimitiveType(kind TypeKind, size int, signed bool) *Type {
	return &Type{
		Kind: kind,
		Size: size,
		Data: &PrimitiveType{
			Name:   kind.String(),
			Size:   size,
			Signed: signed,
		},
	}
}

// NewArrayType creates a new array type
func NewArrayType(elementType *Type, length int) *Type {
	size := 0
	if elementType.Size > 0 {
		size = elementType.Size * length
	}

	return &Type{
		Kind: TypeKindArray,
		Size: size,
		Data: &ArrayType{
			ElementType: elementType,
			Length:      length,
		},
	}
}

// NewSliceType creates a new slice type
func NewSliceType(elementType *Type) *Type {
	return &Type{
		Kind: TypeKindSlice,
		Size: 24, // Typical slice header size (pointer + length + capacity)
		Data: &SliceType{
			ElementType: elementType,
		},
	}
}

// NewPointerType creates a new pointer type
func NewPointerType(pointeeType *Type, nullable bool) *Type {
	return &Type{
		Kind: TypeKindPointer,
		Size: 8, // 64-bit pointer
		Data: &PointerType{
			PointeeType: pointeeType,
			IsNullable:  nullable,
		},
	}
}

// NewStructType creates a new struct type
func NewStructType(name string, fields []StructField) *Type {
	// Calculate struct size (simple alignment)
	size := 0
	for _, field := range fields {
		size += field.Type.Size
	}

	return &Type{
		Kind: TypeKindStruct,
		Size: size,
		Data: &StructType{
			Name:   name,
			Fields: fields,
		},
	}
}

// NewEnumType creates a new enum type
func NewEnumType(name string, variants []EnumVariant) *Type {
	// Enum size is typically the size of the largest variant + discriminant
	maxSize := 8 // Discriminant size
	for _, variant := range variants {
		if variant.Type != nil && variant.Type.Size > maxSize-8 {
			maxSize = variant.Type.Size + 8
		}
	}

	return &Type{
		Kind: TypeKindEnum,
		Size: maxSize,
		Data: &EnumType{
			Name:     name,
			Variants: variants,
		},
	}
}

// NewUnionType creates a new union type
func NewUnionType(name string, members []UnionMember) *Type {
	// Union size is the size of the largest member
	maxSize := 0
	for _, member := range members {
		if member.Type.Size > maxSize {
			maxSize = member.Type.Size
		}
	}

	return &Type{
		Kind: TypeKindUnion,
		Size: maxSize,
		Data: &UnionType{
			Name:    name,
			Members: members,
		},
	}
}

// NewTupleType creates a new tuple type
func NewTupleType(elements []*Type) *Type {
	// Calculate tuple size
	size := 0
	for _, element := range elements {
		size += element.Size
	}

	return &Type{
		Kind: TypeKindTuple,
		Size: size,
		Data: &TupleType{
			Elements: elements,
		},
	}
}

// NewFunctionType creates a new function type
func NewFunctionType(parameters []*Type, returnType *Type, variadic bool, async bool) *Type {
	return &Type{
		Kind: TypeKindFunction,
		Size: 8, // Function pointer size
		Data: &FunctionType{
			Parameters: parameters,
			ReturnType: returnType,
			IsVariadic: variadic,
			IsAsync:    async,
		},
	}
}

// NewChannelType creates a new channel type
func NewChannelType(elementType *Type, direction ChannelDirection, buffered bool, bufferSize int) *Type {
	return &Type{
		Kind: TypeKindChannel,
		Size: 8, // Channel handle size
		Data: &ChannelType{
			ElementType: elementType,
			Direction:   direction,
			Buffered:    buffered,
			BufferSize:  bufferSize,
		},
	}
}

// NewGenericType creates a new generic type parameter
func NewGenericType(name string, constraints []*Type, variance Variance) *Type {
	return &Type{
		Kind: TypeKindGeneric,
		Size: 0, // Size determined when instantiated
		Data: &GenericType{
			Name:        name,
			Constraints: constraints,
			Variance:    variance,
		},
	}
}

// NewTypeVar creates a new type variable
func NewTypeVar(id int, name string, constraints []*Type) *Type {
	return &Type{
		Kind: TypeKindTypeVar,
		Size: 0, // Size determined when unified
		Data: &TypeVar{
			ID:          id,
			Name:        name,
			Constraints: constraints,
			Bound:       nil,
		},
	}
}

// NewRefinementType creates a new refinement type using proper refinement predicate
func NewRefinementType(baseType *Type, predicate string, variables []string) *Type {
	// Parse the predicate string into a proper RefinementPredicate
	var refinementPred RefinementPredicate
	if predicate == "true" {
		refinementPred = &TruePredicate{}
	} else if predicate == "false" {
		refinementPred = &FalsePredicate{}
	} else {
		// Default to true predicate for now - in practice would parse the string
		refinementPred = &TruePredicate{}
	}

	return &Type{
		Kind: TypeKindRefinement,
		Size: baseType.Size,
		Data: &RefinementType{
			BaseType:  baseType,
			Variable:  "x", // Default variable name
			Predicate: refinementPred,
		},
	}
}

// ====== Built-in Types ======

var (
	// Primitive types
	TypeVoid    = NewPrimitiveType(TypeKindVoid, 0, false)
	TypeBool    = NewPrimitiveType(TypeKindBool, 1, false)
	TypeInt8    = NewPrimitiveType(TypeKindInt8, 1, true)
	TypeInt16   = NewPrimitiveType(TypeKindInt16, 2, true)
	TypeInt32   = NewPrimitiveType(TypeKindInt32, 4, true)
	TypeInt64   = NewPrimitiveType(TypeKindInt64, 8, true)
	TypeUint8   = NewPrimitiveType(TypeKindUint8, 1, false)
	TypeUint16  = NewPrimitiveType(TypeKindUint16, 2, false)
	TypeUint32  = NewPrimitiveType(TypeKindUint32, 4, false)
	TypeUint64  = NewPrimitiveType(TypeKindUint64, 8, false)
	TypeFloat32 = NewPrimitiveType(TypeKindFloat32, 4, true)
	TypeFloat64 = NewPrimitiveType(TypeKindFloat64, 8, true)
	TypeChar    = NewPrimitiveType(TypeKindChar, 4, false)    // UTF-32
	TypeString  = NewPrimitiveType(TypeKindString, 16, false) // String header

	// Special types
	TypeAny     = &Type{Kind: TypeKindAny, Size: 0, Data: nil}
	TypeNever   = &Type{Kind: TypeKindNever, Size: 0, Data: nil}
	TypeUnknown = &Type{Kind: TypeKindUnknown, Size: 0, Data: nil}
)

// ====== Type Equivalence ======

// Equals checks if two types are equivalent
func (t *Type) Equals(other *Type) bool {
	if t == nil || other == nil {
		return t == other
	}

	if t.Kind != other.Kind {
		return false
	}

	switch t.Kind {
	case TypeKindVoid, TypeKindBool, TypeKindInt8, TypeKindInt16, TypeKindInt32, TypeKindInt64,
		TypeKindUint8, TypeKindUint16, TypeKindUint32, TypeKindUint64,
		TypeKindFloat32, TypeKindFloat64, TypeKindChar, TypeKindString,
		TypeKindAny, TypeKindNever, TypeKindUnknown:
		return true // Primitive types are equal if kinds match

	case TypeKindArray:
		tArray := t.Data.(*ArrayType)
		oArray := other.Data.(*ArrayType)
		return tArray.Length == oArray.Length && tArray.ElementType.Equals(oArray.ElementType)

	case TypeKindSlice:
		tSlice := t.Data.(*SliceType)
		oSlice := other.Data.(*SliceType)
		return tSlice.ElementType.Equals(oSlice.ElementType)

	case TypeKindPointer:
		tPointer := t.Data.(*PointerType)
		oPointer := other.Data.(*PointerType)
		return tPointer.IsNullable == oPointer.IsNullable &&
			tPointer.PointeeType.Equals(oPointer.PointeeType)

	case TypeKindStruct:
		tStruct := t.Data.(*StructType)
		oStruct := other.Data.(*StructType)
		if tStruct.Name != oStruct.Name || len(tStruct.Fields) != len(oStruct.Fields) {
			return false
		}
		for i, field := range tStruct.Fields {
			otherField := oStruct.Fields[i]
			if field.Name != otherField.Name || !field.Type.Equals(otherField.Type) {
				return false
			}
		}
		return true

	case TypeKindTuple:
		tTuple := t.Data.(*TupleType)
		oTuple := other.Data.(*TupleType)
		if len(tTuple.Elements) != len(oTuple.Elements) {
			return false
		}
		for i, element := range tTuple.Elements {
			if !element.Equals(oTuple.Elements[i]) {
				return false
			}
		}
		return true

	case TypeKindFunction:
		tFunc := t.Data.(*FunctionType)
		oFunc := other.Data.(*FunctionType)
		if len(tFunc.Parameters) != len(oFunc.Parameters) ||
			tFunc.IsVariadic != oFunc.IsVariadic ||
			tFunc.IsAsync != oFunc.IsAsync ||
			!tFunc.ReturnType.Equals(oFunc.ReturnType) {
			return false
		}
		for i, param := range tFunc.Parameters {
			if !param.Equals(oFunc.Parameters[i]) {
				return false
			}
		}
		return true

	case TypeKindTypeVar:
		tVar := t.Data.(*TypeVar)
		oVar := other.Data.(*TypeVar)
		return tVar.ID == oVar.ID

	default:
		// For other types, use pointer equality as a fallback
		return t == other
	}
}

// ====== Type Conversion Rules ======

// CanConvertTo checks if this type can be converted to another type
func (t *Type) CanConvertTo(target *Type) bool {
	if t.Equals(target) {
		return true
	}

	// Numeric conversions
	if t.IsNumeric() && target.IsNumeric() {
		return true // All numeric types can convert to each other
	}

	// Pointer conversions
	if t.Kind == TypeKindPointer && target.Kind == TypeKindPointer {
		tPtr := t.Data.(*PointerType)
		targetPtr := target.Data.(*PointerType)
		// Nullable pointer can convert to non-nullable pointer (null check required)
		// Non-nullable pointer cannot convert to nullable pointer (would break non-null guarantee)
		if tPtr.IsNullable && !targetPtr.IsNullable {
			return tPtr.PointeeType.Equals(targetPtr.PointeeType) // nullable -> non-nullable with runtime check
		}
		if !tPtr.IsNullable && targetPtr.IsNullable {
			return false // non-nullable -> nullable breaks non-null guarantee
		}
		return tPtr.PointeeType.Equals(targetPtr.PointeeType) // same nullability, check element type
	}

	// Array to slice conversion
	if t.Kind == TypeKindArray && target.Kind == TypeKindSlice {
		tArray := t.Data.(*ArrayType)
		targetSlice := target.Data.(*SliceType)
		return tArray.ElementType.Equals(targetSlice.ElementType)
	}

	// Any type can convert to Any
	if target.Kind == TypeKindAny {
		return true
	}

	// Never can convert to any type
	if t.Kind == TypeKindNever {
		return true
	}

	return false
}

// ====== Type Properties ======

// IsNumeric checks if the type is a numeric type
func (t *Type) IsNumeric() bool {
	switch t.Kind {
	case TypeKindInt8, TypeKindInt16, TypeKindInt32, TypeKindInt64,
		TypeKindUint8, TypeKindUint16, TypeKindUint32, TypeKindUint64,
		TypeKindFloat32, TypeKindFloat64:
		return true
	default:
		return false
	}
}

// IsInteger checks if the type is an integer type
func (t *Type) IsInteger() bool {
	switch t.Kind {
	case TypeKindInt8, TypeKindInt16, TypeKindInt32, TypeKindInt64,
		TypeKindUint8, TypeKindUint16, TypeKindUint32, TypeKindUint64:
		return true
	default:
		return false
	}
}

// IsFloat checks if the type is a floating-point type
func (t *Type) IsFloat() bool {
	switch t.Kind {
	case TypeKindFloat32, TypeKindFloat64:
		return true
	default:
		return false
	}
}

// IsSigned checks if the type is signed
func (t *Type) IsSigned() bool {
	if !t.IsNumeric() {
		return false
	}

	if prim, ok := t.Data.(*PrimitiveType); ok {
		return prim.Signed
	}

	return false
}

// IsPointer checks if the type is a pointer type
func (t *Type) IsPointer() bool {
	return t.Kind == TypeKindPointer
}

// IsAggregate checks if the type is an aggregate type
func (t *Type) IsAggregate() bool {
	switch t.Kind {
	case TypeKindArray, TypeKindStruct, TypeKindTuple, TypeKindUnion:
		return true
	default:
		return false
	}
}

// IsCallable checks if the type is callable (function or function pointer)
func (t *Type) IsCallable() bool {
	return t.Kind == TypeKindFunction
}

// ====== String Representation ======

// String returns the string representation of the type
func (t *Type) String() string {
	if t == nil {
		return "<nil>"
	}

	switch t.Kind {
	case TypeKindVoid, TypeKindBool, TypeKindInt8, TypeKindInt16, TypeKindInt32, TypeKindInt64,
		TypeKindUint8, TypeKindUint16, TypeKindUint32, TypeKindUint64,
		TypeKindFloat32, TypeKindFloat64, TypeKindChar, TypeKindString,
		TypeKindAny, TypeKindNever, TypeKindUnknown:
		return t.Kind.String()

	case TypeKindArray:
		array := t.Data.(*ArrayType)
		return fmt.Sprintf("[%d]%s", array.Length, array.ElementType.String())

	case TypeKindSlice:
		slice := t.Data.(*SliceType)
		return fmt.Sprintf("[]%s", slice.ElementType.String())

	case TypeKindPointer:
		pointer := t.Data.(*PointerType)
		nullable := ""
		if pointer.IsNullable {
			nullable = "?"
		}
		return fmt.Sprintf("*%s%s", pointer.PointeeType.String(), nullable)

	case TypeKindStruct:
		structType := t.Data.(*StructType)
		if structType.Name != "" {
			return structType.Name
		}

		var fields []string
		for _, field := range structType.Fields {
			fields = append(fields, fmt.Sprintf("%s: %s", field.Name, field.Type.String()))
		}
		return fmt.Sprintf("struct { %s }", strings.Join(fields, ", "))

	case TypeKindEnum:
		enumType := t.Data.(*EnumType)
		return enumType.Name

	case TypeKindUnion:
		unionType := t.Data.(*UnionType)
		return unionType.Name

	case TypeKindTuple:
		tuple := t.Data.(*TupleType)
		var elements []string
		for _, element := range tuple.Elements {
			elements = append(elements, element.String())
		}
		return fmt.Sprintf("(%s)", strings.Join(elements, ", "))

	case TypeKindFunction:
		function := t.Data.(*FunctionType)
		var params []string
		for _, param := range function.Parameters {
			params = append(params, param.String())
		}

		paramStr := strings.Join(params, ", ")
		if function.IsVariadic {
			paramStr += ", ..."
		}

		asyncStr := ""
		if function.IsAsync {
			asyncStr = "async "
		}

		return fmt.Sprintf("%sfn(%s) -> %s", asyncStr, paramStr, function.ReturnType.String())

	case TypeKindChannel:
		channel := t.Data.(*ChannelType)
		dirStr := ""
		switch channel.Direction {
		case ChannelSendOnly:
			dirStr = "send "
		case ChannelReceiveOnly:
			dirStr = "recv "
		}

		bufStr := ""
		if channel.Buffered {
			bufStr = fmt.Sprintf("<%d>", channel.BufferSize)
		}

		return fmt.Sprintf("%schan%s %s", dirStr, bufStr, channel.ElementType.String())

	case TypeKindGeneric:
		generic := t.Data.(*GenericType)
		if len(generic.Constraints) == 0 {
			return generic.Name
		}

		var constraints []string
		for _, constraint := range generic.Constraints {
			constraints = append(constraints, constraint.String())
		}
		return fmt.Sprintf("%s: %s", generic.Name, strings.Join(constraints, " + "))

	case TypeKindTypeVar:
		typeVar := t.Data.(*TypeVar)
		if typeVar.Bound != nil {
			return typeVar.Bound.String()
		}
		return fmt.Sprintf("'%s", typeVar.Name)

	case TypeKindRefinement:
		refinement := t.Data.(*RefinementType)
		return fmt.Sprintf("%s{%s}", refinement.BaseType.String(), refinement.Predicate)

	default:
		return fmt.Sprintf("<%s>", t.Kind.String())
	}
}

// ====== Type Registry ======

// TypeRegistry maintains a registry of all types in the system
type TypeRegistry struct {
	types         map[string]*Type
	nextTypeVarID int
}

// NewTypeRegistry creates a new type registry
func NewTypeRegistry() *TypeRegistry {
	registry := &TypeRegistry{
		types:         make(map[string]*Type),
		nextTypeVarID: 0,
	}

	// Register built-in types
	registry.RegisterType("void", TypeVoid)
	registry.RegisterType("bool", TypeBool)
	registry.RegisterType("int8", TypeInt8)
	registry.RegisterType("int16", TypeInt16)
	registry.RegisterType("int32", TypeInt32)
	registry.RegisterType("int64", TypeInt64)
	registry.RegisterType("uint8", TypeUint8)
	registry.RegisterType("uint16", TypeUint16)
	registry.RegisterType("uint32", TypeUint32)
	registry.RegisterType("uint64", TypeUint64)
	registry.RegisterType("float32", TypeFloat32)
	registry.RegisterType("float64", TypeFloat64)
	registry.RegisterType("char", TypeChar)
	registry.RegisterType("string", TypeString)
	registry.RegisterType("any", TypeAny)
	registry.RegisterType("never", TypeNever)

	return registry
}

// RegisterType registers a type with the given name
func (tr *TypeRegistry) RegisterType(name string, typeObj *Type) {
	tr.types[name] = typeObj
}

// LookupType looks up a type by name
func (tr *TypeRegistry) LookupType(name string) (*Type, bool) {
	typeObj, exists := tr.types[name]
	return typeObj, exists
}

// NewTypeVar creates a new type variable with a unique ID
func (tr *TypeRegistry) NewTypeVar(name string, constraints []*Type) *Type {
	id := tr.nextTypeVarID
	tr.nextTypeVarID++
	return NewTypeVar(id, name, constraints)
}

// GetAllTypes returns all registered types
func (tr *TypeRegistry) GetAllTypes() map[string]*Type {
	result := make(map[string]*Type)
	for name, typeObj := range tr.types {
		result[name] = typeObj
	}
	return result
}
