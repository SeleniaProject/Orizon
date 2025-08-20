// Compound type system implementation for Orizon language.
// This module provides complex type constructions and operations.

package types

import (
	"fmt"
	"sort"
	"strings"
)

// ====== Advanced Compound Types ======.

// VariantType represents a variant (sum) type.
type VariantType struct {
	Name     string
	Variants []VariantOption
}

// VariantOption represents an option in a variant type.
type VariantOption struct {
	Type *Type
	Name string
}

// RecordType represents a record (product) type with named fields.
type RecordType struct {
	Name   string
	Fields []RecordField
}

// RecordField represents a field in a record type.
type RecordField struct {
	Default    interface{}
	Type       *Type
	Name       string
	Visibility Visibility
	Optional   bool
}

// Visibility represents field visibility.
type Visibility int

const (
	VisibilityPrivate Visibility = iota
	VisibilityPublic
	VisibilityInternal
)

// NewlineType represents a newtype (nominal wrapper).
type NewlineType struct {
	BaseType *Type
	Name     string
}

// InterfaceType represents an interface type.
type InterfaceType struct {
	Name    string
	Methods []MethodSignature
	Extends []*Type // Interface inheritance
}

// MethodSignature represents a method signature in an interface.
type MethodSignature struct {
	ReturnType *Type
	Name       string
	Parameters []*Type
	IsStatic   bool
	IsAsync    bool
}

// TraitType represents a trait (similar to Rust traits or Haskell type classes).
type TraitType struct {
	DefaultImpl map[string]string
	Name        string
	TypeParams  []GenericParameter
	Methods     []MethodSignature
	AssocTypes  []AssociatedType
	Constraints []*Type
}

// GenericParameter represents a generic type parameter.
type GenericParameter struct {
	Default     *Type
	Name        string
	Constraints []*Type
	Variance    Variance
}

// AssociatedType represents an associated type in a trait.
type AssociatedType struct {
	Default     *Type
	Name        string
	Constraints []*Type
}

// ====== Type Layout and Memory ======.

// TypeLayout represents the memory layout of a type.
type TypeLayout struct {
	Fields    []FieldLayout
	Padding   []PaddingInfo
	Size      int
	Alignment int
}

// FieldLayout represents the layout of a field within a type.
type FieldLayout struct {
	Type   *Type
	Name   string
	Offset int
	Size   int
}

// PaddingInfo represents padding information.
type PaddingInfo struct {
	Reason string
	Offset int
	Size   int
}

// ====== Type Construction Functions ======.

// NewVariantType creates a new variant type.
func NewVariantType(name string, variants []VariantOption) *Type {
	// Calculate size (largest variant + discriminant).
	maxSize := 0
	for _, variant := range variants {
		if variant.Type != nil && variant.Type.Size > maxSize {
			maxSize = variant.Type.Size
		}
	}

	size := maxSize + 8 // Add discriminant size

	return &Type{
		Kind: TypeKindEnum, // Reuse enum kind for variants
		Size: size,
		Data: &VariantType{
			Name:     name,
			Variants: variants,
		},
	}
}

// NewRecordType creates a new record type.
func NewRecordType(name string, fields []RecordField) *Type {
	layout := CalculateLayout(fields)

	return &Type{
		Kind: TypeKindStruct, // Reuse struct kind for records
		Size: layout.Size,
		Data: &RecordType{
			Name:   name,
			Fields: fields,
		},
	}
}

// NewNewtypeType creates a new newtype.
func NewNewtypeType(name string, baseType *Type) *Type {
	return &Type{
		Kind: baseType.Kind, // Inherit kind from base type
		Size: baseType.Size,
		Data: &NewlineType{
			Name:     name,
			BaseType: baseType,
		},
	}
}

// NewInterfaceType creates a new interface type.
func NewInterfaceType(name string, methods []MethodSignature, extends []*Type) *Type {
	return &Type{
		Kind: TypeKindStruct, // Interfaces are implemented as vtables
		Size: 16,             // vtable pointer + data pointer
		Data: &InterfaceType{
			Name:    name,
			Methods: methods,
			Extends: extends,
		},
	}
}

// NewTraitType creates a new trait type.
func NewTraitType(name string, params []GenericParameter, methods []MethodSignature) *Type {
	return &Type{
		Kind: TypeKindTrait,
		Size: 0, // Traits don't have runtime representation
		Data: &TraitType{
			Name:        name,
			TypeParams:  params,
			Methods:     methods,
			AssocTypes:  []AssociatedType{},
			Constraints: []*Type{},
			DefaultImpl: make(map[string]string),
		},
	}
}

// ====== Layout Calculation ======.

// CalculateLayout calculates the memory layout for a set of fields.
func CalculateLayout(fields []RecordField) TypeLayout {
	layout := TypeLayout{
		Size:      0,
		Alignment: 1,
		Fields:    make([]FieldLayout, 0, len(fields)),
		Padding:   []PaddingInfo{},
	}

	currentOffset := 0

	for _, field := range fields {
		fieldAlign := GetTypeAlignment(field.Type)
		if fieldAlign > layout.Alignment {
			layout.Alignment = fieldAlign
		}

		// Add padding for alignment.
		alignedOffset := AlignTo(currentOffset, fieldAlign)
		if alignedOffset > currentOffset {
			layout.Padding = append(layout.Padding, PaddingInfo{
				Offset: currentOffset,
				Size:   alignedOffset - currentOffset,
				Reason: fmt.Sprintf("Alignment for field %s", field.Name),
			})
		}

		// Add field.
		layout.Fields = append(layout.Fields, FieldLayout{
			Name:   field.Name,
			Offset: alignedOffset,
			Size:   field.Type.Size,
			Type:   field.Type,
		})

		currentOffset = alignedOffset + field.Type.Size
	}

	// Final alignment.
	finalSize := AlignTo(currentOffset, layout.Alignment)
	if finalSize > currentOffset {
		layout.Padding = append(layout.Padding, PaddingInfo{
			Offset: currentOffset,
			Size:   finalSize - currentOffset,
			Reason: "Final struct alignment",
		})
	}

	layout.Size = finalSize

	return layout
}

// GetTypeAlignment returns the alignment requirement for a type.
func GetTypeAlignment(t *Type) int {
	switch t.Kind {
	case TypeKindBool, TypeKindInt8, TypeKindUint8:
		return 1
	case TypeKindInt16, TypeKindUint16:
		return 2
	case TypeKindInt32, TypeKindUint32, TypeKindFloat32:
		return 4
	case TypeKindInt64, TypeKindUint64, TypeKindFloat64, TypeKindPointer:
		return 8
	case TypeKindArray:
		arrayType := t.Data.(*ArrayType)

		return GetTypeAlignment(arrayType.ElementType)
	case TypeKindStruct:
		// Handle both StructType and RecordType.
		switch data := t.Data.(type) {
		case *StructType:
			maxAlign := 1

			for _, field := range data.Fields {
				align := GetTypeAlignment(field.Type)
				if align > maxAlign {
					maxAlign = align
				}
			}

			return maxAlign
		case *RecordType:
			maxAlign := 1

			for _, field := range data.Fields {
				align := GetTypeAlignment(field.Type)
				if align > maxAlign {
					maxAlign = align
				}
			}

			return maxAlign
		default:
			return 8
		}
	default:
		return 8 // Default to pointer alignment
	}
}

// AlignTo aligns an offset to the specified alignment.
func AlignTo(offset, alignment int) int {
	if alignment <= 1 {
		return offset
	}

	return (offset + alignment - 1) &^ (alignment - 1)
}

// ====== Type Compatibility and Conversion ======.

// IsAssignableFrom checks if this type can be assigned from another type.
func (t *Type) IsAssignableFrom(other *Type) bool {
	if t.Equals(other) {
		return true
	}

	// Check for implicit conversions.
	switch t.Kind {
	case TypeKindAny:
		return true // Any accepts all types

	case TypeKindFloat32, TypeKindFloat64:
		return other.IsInteger() // Integers can be implicitly converted to floats

	case TypeKindPointer:
		if other.Kind == TypeKindPointer {
			tPtr := t.Data.(*PointerType)
			oPtr := other.Data.(*PointerType)
			// Can assign non-nullable to nullable.
			return tPtr.IsNullable || !oPtr.IsNullable
		}

	case TypeKindSlice:
		if other.Kind == TypeKindArray {
			tSlice := t.Data.(*SliceType)
			oArray := other.Data.(*ArrayType)

			return tSlice.ElementType.Equals(oArray.ElementType)
		}
	}

	// Check interface compatibility.
	if t.Kind == TypeKindStruct {
		if interfaceType, ok := t.Data.(*InterfaceType); ok {
			return t.ImplementsInterface(other, interfaceType)
		}
	}

	return false
}

// ImplementsInterface checks if a type implements an interface.
func (t *Type) ImplementsInterface(implType *Type, interfaceType *InterfaceType) bool {
	// This is a simplified check - in practice, you'd need to verify.
	// that all interface methods are implemented by the type.
	switch implType.Kind {
	case TypeKindStruct:
		// Check if struct has all required methods.
		// This would require a method registry or reflection system.
		return true // Simplified for now

	default:
		return false
	}
}

// ====== Subtyping Relations ======.

// IsSubtypeOf checks if this type is a subtype of another type.
func (t *Type) IsSubtypeOf(supertype *Type) bool {
	if t.Equals(supertype) {
		return true
	}

	// Never is a subtype of all types (bottom type).
	if t.Kind == TypeKindNever {
		return true
	}

	switch supertype.Kind {
	case TypeKindAny:
		return true // Everything is a subtype of Any

	case TypeKindStruct:
		// Check structural subtyping for records.
		if t.Kind == TypeKindStruct {
			return t.IsStructuralSubtype(supertype)
		}

	case TypeKindFunction:
		// Function subtyping (contravariant in parameters, covariant in return).
		if t.Kind == TypeKindFunction {
			return t.IsFunctionSubtype(supertype)
		}
	}

	return false
}

// IsStructuralSubtype checks structural subtyping between struct types.
func (t *Type) IsStructuralSubtype(supertype *Type) bool {
	tStruct := t.Data.(*StructType)
	superStruct := supertype.Data.(*StructType)

	// Subtype must have at least all fields of supertype.
	tFieldMap := make(map[string]*Type)
	for _, field := range tStruct.Fields {
		tFieldMap[field.Name] = field.Type
	}

	for _, superField := range superStruct.Fields {
		tFieldType, exists := tFieldMap[superField.Name]
		if !exists {
			return false // Missing field
		}

		if !tFieldType.IsSubtypeOf(superField.Type) {
			return false // Incompatible field type
		}
	}

	return true
}

// IsFunctionSubtype checks function subtyping.
func (t *Type) IsFunctionSubtype(supertype *Type) bool {
	tFunc := t.Data.(*FunctionType)
	superFunc := supertype.Data.(*FunctionType)

	// Check parameter count.
	if len(tFunc.Parameters) != len(superFunc.Parameters) {
		return false
	}

	// Parameters are contravariant.
	for i, tParam := range tFunc.Parameters {
		superParam := superFunc.Parameters[i]
		if !superParam.IsSubtypeOf(tParam) {
			return false
		}
	}

	// Return type is covariant.
	return tFunc.ReturnType.IsSubtypeOf(superFunc.ReturnType)
}

// ====== Type Unification ======.

// Unify attempts to unify two types, returning the most general unified type.
func Unify(t1, t2 *Type) (*Type, error) {
	if t1 == nil || t2 == nil {
		return nil, fmt.Errorf("cannot unify nil types")
	}

	if t1.Equals(t2) {
		return t1, nil
	}

	// Handle type variables.
	if t1.Kind == TypeKindTypeVar {
		return unifyTypeVar(t1, t2)
	}

	if t2.Kind == TypeKindTypeVar {
		return unifyTypeVar(t2, t1)
	}

	// Handle specific unification cases.
	switch {
	case t1.Kind == TypeKindAny || t2.Kind == TypeKindAny:
		return TypeAny, nil

	case t1.Kind == TypeKindNever:
		return t2, nil

	case t2.Kind == TypeKindNever:
		return t1, nil

	case t1.IsNumeric() && t2.IsNumeric():
		return unifyNumericTypes(t1, t2), nil

	case t1.Kind == TypeKindArray && t2.Kind == TypeKindArray:
		return unifyArrayTypes(t1, t2)

	case t1.Kind == TypeKindFunction && t2.Kind == TypeKindFunction:
		return unifyFunctionTypes(t1, t2)

	default:
		return nil, fmt.Errorf("cannot unify types %s and %s", t1.String(), t2.String())
	}
}

// unifyTypeVar unifies a type variable with another type.
func unifyTypeVar(typeVar, other *Type) (*Type, error) {
	tv := typeVar.Data.(*TypeVar)

	// Check if already bound.
	if tv.Bound != nil {
		return Unify(tv.Bound, other)
	}

	// Check constraints.
	for _, constraint := range tv.Constraints {
		if !other.IsSubtypeOf(constraint) {
			return nil, fmt.Errorf("type %s does not satisfy constraint %s", other.String(), constraint.String())
		}
	}

	// Bind the type variable.
	tv.Bound = other

	return other, nil
}

// unifyNumericTypes unifies two numeric types.
func unifyNumericTypes(t1, t2 *Type) *Type {
	// Promotion rules: larger types subsume smaller ones.
	typeRank := map[TypeKind]int{
		TypeKindInt8:    1,
		TypeKindUint8:   1,
		TypeKindInt16:   2,
		TypeKindUint16:  2,
		TypeKindInt32:   3,
		TypeKindUint32:  3,
		TypeKindInt64:   4,
		TypeKindUint64:  4,
		TypeKindFloat32: 5,
		TypeKindFloat64: 6,
	}

	rank1 := typeRank[t1.Kind]
	rank2 := typeRank[t2.Kind]

	if rank1 >= rank2 {
		return t1
	}

	return t2
}

// unifyArrayTypes unifies two array types.
func unifyArrayTypes(t1, t2 *Type) (*Type, error) {
	array1 := t1.Data.(*ArrayType)
	array2 := t2.Data.(*ArrayType)

	if array1.Length != array2.Length {
		return nil, fmt.Errorf("array lengths don't match: %d vs %d", array1.Length, array2.Length)
	}

	elementType, err := Unify(array1.ElementType, array2.ElementType)
	if err != nil {
		return nil, fmt.Errorf("cannot unify array element types: %w", err)
	}

	return NewArrayType(elementType, array1.Length), nil
}

// unifyFunctionTypes unifies two function types.
func unifyFunctionTypes(t1, t2 *Type) (*Type, error) {
	func1 := t1.Data.(*FunctionType)
	func2 := t2.Data.(*FunctionType)

	if len(func1.Parameters) != len(func2.Parameters) {
		return nil, fmt.Errorf("function parameter counts don't match")
	}

	// Unify parameters.
	var unifiedParams []*Type

	for i, param1 := range func1.Parameters {
		param2 := func2.Parameters[i]

		unifiedParam, err := Unify(param1, param2)
		if err != nil {
			return nil, fmt.Errorf("cannot unify parameter %d: %w", i, err)
		}

		unifiedParams = append(unifiedParams, unifiedParam)
	}

	// Unify return type.
	unifiedReturn, err := Unify(func1.ReturnType, func2.ReturnType)
	if err != nil {
		return nil, fmt.Errorf("cannot unify return types: %w", err)
	}

	return NewFunctionType(unifiedParams, unifiedReturn,
		func1.IsVariadic && func2.IsVariadic,
		func1.IsAsync && func2.IsAsync), nil
}

// ====== Type Substitution ======.

// Substitute performs type substitution, replacing type variables with concrete types.
func Substitute(t *Type, substitutions map[int]*Type) *Type {
	if t == nil {
		return nil
	}

	switch t.Kind {
	case TypeKindTypeVar:
		tv := t.Data.(*TypeVar)
		if replacement, exists := substitutions[tv.ID]; exists {
			return replacement
		}

		return t

	case TypeKindArray:
		array := t.Data.(*ArrayType)
		newElementType := Substitute(array.ElementType, substitutions)

		if newElementType != array.ElementType {
			return NewArrayType(newElementType, array.Length)
		}

		return t

	case TypeKindSlice:
		slice := t.Data.(*SliceType)
		newElementType := Substitute(slice.ElementType, substitutions)

		if newElementType != slice.ElementType {
			return NewSliceType(newElementType)
		}

		return t

	case TypeKindPointer:
		pointer := t.Data.(*PointerType)
		newPointeeType := Substitute(pointer.PointeeType, substitutions)

		if newPointeeType != pointer.PointeeType {
			return NewPointerType(newPointeeType, pointer.IsNullable)
		}

		return t

	case TypeKindFunction:
		function := t.Data.(*FunctionType)

		var newParams []*Type

		changed := false

		for _, param := range function.Parameters {
			newParam := Substitute(param, substitutions)
			newParams = append(newParams, newParam)

			if newParam != param {
				changed = true
			}
		}

		newReturn := Substitute(function.ReturnType, substitutions)
		if newReturn != function.ReturnType {
			changed = true
		}

		if changed {
			return NewFunctionType(newParams, newReturn, function.IsVariadic, function.IsAsync)
		}

		return t

	case TypeKindStruct:
		structType := t.Data.(*StructType)

		var newFields []StructField

		changed := false

		for _, field := range structType.Fields {
			newFieldType := Substitute(field.Type, substitutions)
			newFields = append(newFields, StructField{
				Name: field.Name,
				Type: newFieldType,
				Tag:  field.Tag,
			})

			if newFieldType != field.Type {
				changed = true
			}
		}

		if changed {
			return NewStructType(structType.Name, newFields)
		}

		return t

	default:
		return t
	}
}

// ====== Type Normalization ======.

// Normalize normalizes a type by resolving type variables and simplifying structure.
func Normalize(t *Type) *Type {
	if t == nil {
		return nil
	}

	switch t.Kind {
	case TypeKindTypeVar:
		tv := t.Data.(*TypeVar)
		if tv.Bound != nil {
			return Normalize(tv.Bound)
		}

		return t

	case TypeKindArray:
		array := t.Data.(*ArrayType)
		normalizedElement := Normalize(array.ElementType)

		if normalizedElement != array.ElementType {
			return NewArrayType(normalizedElement, array.Length)
		}

		return t

	case TypeKindFunction:
		function := t.Data.(*FunctionType)

		var normalizedParams []*Type

		changed := false

		for _, param := range function.Parameters {
			normalizedParam := Normalize(param)
			normalizedParams = append(normalizedParams, normalizedParam)

			if normalizedParam != param {
				changed = true
			}
		}

		normalizedReturn := Normalize(function.ReturnType)
		if normalizedReturn != function.ReturnType {
			changed = true
		}

		if changed {
			return NewFunctionType(normalizedParams, normalizedReturn,
				function.IsVariadic, function.IsAsync)
		}

		return t

	default:
		return t
	}
}

// ====== Type Formatting and Display ======.

// FormatType formats a type with detailed information.
func FormatType(t *Type, detail bool) string {
	if t == nil {
		return "<nil>"
	}

	if !detail {
		return t.String()
	}

	switch t.Kind {
	case TypeKindStruct:
		structType := t.Data.(*StructType)

		var fields []string

		for _, field := range structType.Fields {
			fields = append(fields, fmt.Sprintf("  %s: %s", field.Name, field.Type.String()))
		}

		return fmt.Sprintf("struct %s {\n%s\n}", structType.Name, strings.Join(fields, "\n"))

	case TypeKindEnum:
		if variantType, ok := t.Data.(*VariantType); ok {
			var variants []string

			for _, variant := range variantType.Variants {
				if variant.Type != nil {
					variants = append(variants, fmt.Sprintf("  %s(%s)", variant.Name, variant.Type.String()))
				} else {
					variants = append(variants, fmt.Sprintf("  %s", variant.Name))
				}
			}

			return fmt.Sprintf("enum %s {\n%s\n}", variantType.Name, strings.Join(variants, "\n"))
		}

		return t.String()

	case TypeKindFunction:
		function := t.Data.(*FunctionType)

		var params []string

		for i, param := range function.Parameters {
			params = append(params, fmt.Sprintf("param%d: %s", i, param.String()))
		}

		modifiers := []string{}
		if function.IsAsync {
			modifiers = append(modifiers, "async")
		}

		if function.IsVariadic {
			modifiers = append(modifiers, "variadic")
		}

		modStr := ""
		if len(modifiers) > 0 {
			modStr = strings.Join(modifiers, " ") + " "
		}

		return fmt.Sprintf("%sfunction(\n  %s\n) -> %s",
			modStr, strings.Join(params, ",\n  "), function.ReturnType.String())

	default:
		return t.String()
	}
}

// ====== Type Utilities ======.

// GetAllTypeVars returns all type variables in a type.
func GetAllTypeVars(t *Type) []*Type {
	var typeVars []*Type

	visitType(t, func(visited *Type) {
		if visited.Kind == TypeKindTypeVar {
			typeVars = append(typeVars, visited)
		}
	})

	return typeVars
}

// visitType recursively visits all types in a type structure.
func visitType(t *Type, visitor func(*Type)) {
	if t == nil {
		return
	}

	visitor(t)

	switch t.Kind {
	case TypeKindArray:
		array := t.Data.(*ArrayType)
		visitType(array.ElementType, visitor)

	case TypeKindSlice:
		slice := t.Data.(*SliceType)
		visitType(slice.ElementType, visitor)

	case TypeKindPointer:
		pointer := t.Data.(*PointerType)
		visitType(pointer.PointeeType, visitor)

	case TypeKindFunction:
		function := t.Data.(*FunctionType)
		for _, param := range function.Parameters {
			visitType(param, visitor)
		}

		visitType(function.ReturnType, visitor)

	case TypeKindStruct:
		structType := t.Data.(*StructType)
		for _, field := range structType.Fields {
			visitType(field.Type, visitor)
		}
	}
}

// TypeComplexity returns a measure of type complexity.
func TypeComplexity(t *Type) int {
	if t == nil {
		return 0
	}

	switch t.Kind {
	case TypeKindVoid, TypeKindBool, TypeKindInt8, TypeKindInt16, TypeKindInt32, TypeKindInt64,
		TypeKindUint8, TypeKindUint16, TypeKindUint32, TypeKindUint64,
		TypeKindFloat32, TypeKindFloat64, TypeKindChar, TypeKindString:
		return 1

	case TypeKindArray:
		array := t.Data.(*ArrayType)

		return 1 + TypeComplexity(array.ElementType)

	case TypeKindSlice:
		slice := t.Data.(*SliceType)

		return 1 + TypeComplexity(slice.ElementType)

	case TypeKindPointer:
		pointer := t.Data.(*PointerType)

		return 1 + TypeComplexity(pointer.PointeeType)

	case TypeKindFunction:
		function := t.Data.(*FunctionType)
		complexity := 1 + TypeComplexity(function.ReturnType)

		for _, param := range function.Parameters {
			complexity += TypeComplexity(param)
		}

		return complexity

	case TypeKindStruct:
		structType := t.Data.(*StructType)
		complexity := 1

		for _, field := range structType.Fields {
			complexity += TypeComplexity(field.Type)
		}

		return complexity

	default:
		return 1
	}
}

// ====== Type Sorting and Ordering ======.

// TypesByComplexity implements sort.Interface for sorting types by complexity.
type TypesByComplexity []*Type

func (t TypesByComplexity) Len() int      { return len(t) }
func (t TypesByComplexity) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t TypesByComplexity) Less(i, j int) bool {
	return TypeComplexity(t[i]) < TypeComplexity(t[j])
}

// SortTypesByComplexity sorts a slice of types by complexity.
func SortTypesByComplexity(types []*Type) {
	sort.Sort(TypesByComplexity(types))
}
