// Additional AST node definitions for complete type coverage
// This file extends the existing AST with additional node types
// needed for a complete Orizon language implementation.

package parser

import (
	"fmt"
	"strings"
)

// ====== Additional Type Nodes ======

// ArrayType represents an array type [T; size] or [T]
type ArrayType struct {
	Span        Span
	ElementType Type
	Size        Expression // nil for dynamic arrays
	IsDynamic   bool       // true for [T], false for [T; size]
}

func (at *ArrayType) GetSpan() Span { return at.Span }
func (at *ArrayType) String() string {
	if at.IsDynamic {
		return fmt.Sprintf("[%s]", at.ElementType.String())
	}
	return fmt.Sprintf("[%s; %s]", at.ElementType.String(), at.Size.String())
}
func (at *ArrayType) Accept(visitor Visitor) interface{} { return visitor.VisitArrayType(at) }
func (at *ArrayType) typeNode()                          {}
func (at *ArrayType) GetNodeKind() NodeKind              { return NodeKindArrayType }
func (at *ArrayType) Clone() TypeSafeNode {
	clone := *at
	if at.ElementType != nil {
		clone.ElementType = at.ElementType.(TypeSafeNode).Clone().(Type)
	}
	if at.Size != nil {
		clone.Size = at.Size.(TypeSafeNode).Clone().(Expression)
	}
	return &clone
}
func (at *ArrayType) Equals(other TypeSafeNode) bool {
	if ot, ok := other.(*ArrayType); ok {
		return at.IsDynamic == ot.IsDynamic &&
			((at.ElementType == nil && ot.ElementType == nil) ||
				(at.ElementType != nil && ot.ElementType != nil &&
					at.ElementType.(TypeSafeNode).Equals(ot.ElementType.(TypeSafeNode)))) &&
			((at.Size == nil && ot.Size == nil) ||
				(at.Size != nil && ot.Size != nil &&
					at.Size.(TypeSafeNode).Equals(ot.Size.(TypeSafeNode))))
	}
	return false
}
func (at *ArrayType) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, 2)
	if at.ElementType != nil {
		children = append(children, at.ElementType.(TypeSafeNode))
	}
	if at.Size != nil {
		children = append(children, at.Size.(TypeSafeNode))
	}
	return children
}
func (at *ArrayType) ReplaceChild(index int, newChild TypeSafeNode) error {
	children := at.GetChildren()
	if index < 0 || index >= len(children) {
		return fmt.Errorf("index %d out of range for ArrayType children", index)
	}
	if index == 0 && at.ElementType != nil {
		if newType, ok := newChild.(Type); ok {
			at.ElementType = newType
			return nil
		}
		return fmt.Errorf("expected Type for ElementType, got %T", newChild)
	}
	if index == 1 && at.Size != nil {
		if newExpr, ok := newChild.(Expression); ok {
			at.Size = newExpr
			return nil
		}
		return fmt.Errorf("expected Expression for Size, got %T", newChild)
	}
	return fmt.Errorf("invalid child index %d for ArrayType", index)
}

// FunctionType represents a function type (param1: Type1, param2: Type2) -> ReturnType
type FunctionType struct {
	Span       Span
	Parameters []*FunctionTypeParameter
	ReturnType Type
	IsAsync    bool
}

type FunctionTypeParameter struct {
	Span Span
	Name string // Optional parameter name
	Type Type
}

func (ft *FunctionType) GetSpan() Span { return ft.Span }
func (ft *FunctionType) String() string {
	var params []string
	for _, p := range ft.Parameters {
		if p.Name != "" {
			params = append(params, fmt.Sprintf("%s: %s", p.Name, p.Type.String()))
		} else {
			params = append(params, p.Type.String())
		}
	}
	prefix := ""
	if ft.IsAsync {
		prefix = "async "
	}
	return fmt.Sprintf("%s(%s) -> %s", prefix, strings.Join(params, ", "), ft.ReturnType.String())
}
func (ft *FunctionType) Accept(visitor Visitor) interface{} { return visitor.VisitFunctionType(ft) }
func (ft *FunctionType) typeNode()                          {}
func (ft *FunctionType) GetNodeKind() NodeKind              { return NodeKindFunctionType }
func (ft *FunctionType) Clone() TypeSafeNode {
	clone := *ft
	clone.Parameters = make([]*FunctionTypeParameter, len(ft.Parameters))
	for i, param := range ft.Parameters {
		paramClone := *param
		if param.Type != nil {
			paramClone.Type = param.Type.(TypeSafeNode).Clone().(Type)
		}
		clone.Parameters[i] = &paramClone
	}
	if ft.ReturnType != nil {
		clone.ReturnType = ft.ReturnType.(TypeSafeNode).Clone().(Type)
	}
	return &clone
}
func (ft *FunctionType) Equals(other TypeSafeNode) bool {
	if ot, ok := other.(*FunctionType); ok {
		if ft.IsAsync != ot.IsAsync || len(ft.Parameters) != len(ot.Parameters) {
			return false
		}
		for i, param := range ft.Parameters {
			otherParam := ot.Parameters[i]
			if param.Name != otherParam.Name ||
				!param.Type.(TypeSafeNode).Equals(otherParam.Type.(TypeSafeNode)) {
				return false
			}
		}
		return ((ft.ReturnType == nil && ot.ReturnType == nil) ||
			(ft.ReturnType != nil && ot.ReturnType != nil &&
				ft.ReturnType.(TypeSafeNode).Equals(ot.ReturnType.(TypeSafeNode))))
	}
	return false
}
func (ft *FunctionType) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, len(ft.Parameters)+1)
	for _, param := range ft.Parameters {
		if param.Type != nil {
			children = append(children, param.Type.(TypeSafeNode))
		}
	}
	if ft.ReturnType != nil {
		children = append(children, ft.ReturnType.(TypeSafeNode))
	}
	return children
}
func (ft *FunctionType) ReplaceChild(index int, newChild TypeSafeNode) error {
	children := ft.GetChildren()
	if index < 0 || index >= len(children) {
		return fmt.Errorf("index %d out of range for FunctionType children", index)
	}
	newType, ok := newChild.(Type)
	if !ok {
		return fmt.Errorf("expected Type, got %T", newChild)
	}
	if index < len(ft.Parameters) {
		ft.Parameters[index].Type = newType
		return nil
	}
	if index == len(ft.Parameters) && ft.ReturnType != nil {
		ft.ReturnType = newType
		return nil
	}
	return fmt.Errorf("invalid child index %d for FunctionType", index)
}

// StructType represents a struct type
type StructType struct {
	Span   Span
	Name   *Identifier
	Fields []*StructField
}

type StructField struct {
	Span     Span
	Name     *Identifier
	Type     Type
	IsPublic bool
	Tags     map[string]string // Optional field tags
}

func (st *StructType) GetSpan() Span { return st.Span }
func (st *StructType) String() string {
	if st.Name != nil {
		return fmt.Sprintf("struct %s", st.Name.Value)
	}
	return "struct { ... }"
}
func (st *StructType) Accept(visitor Visitor) interface{} { return visitor.VisitStructType(st) }
func (st *StructType) typeNode()                          {}
func (st *StructType) GetNodeKind() NodeKind              { return NodeKindStructType }
func (st *StructType) Clone() TypeSafeNode {
	clone := *st
	if st.Name != nil {
		clone.Name = st.Name.Clone().(*Identifier)
	}
	clone.Fields = make([]*StructField, len(st.Fields))
	for i, field := range st.Fields {
		fieldClone := *field
		if field.Name != nil {
			fieldClone.Name = field.Name.Clone().(*Identifier)
		}
		if field.Type != nil {
			fieldClone.Type = field.Type.(TypeSafeNode).Clone().(Type)
		}
		if field.Tags != nil {
			fieldClone.Tags = make(map[string]string)
			for k, v := range field.Tags {
				fieldClone.Tags[k] = v
			}
		}
		clone.Fields[i] = &fieldClone
	}
	return &clone
}
func (st *StructType) Equals(other TypeSafeNode) bool {
	if ot, ok := other.(*StructType); ok {
		if len(st.Fields) != len(ot.Fields) {
			return false
		}
		if !((st.Name == nil && ot.Name == nil) ||
			(st.Name != nil && ot.Name != nil && st.Name.Equals(ot.Name))) {
			return false
		}
		for i, field := range st.Fields {
			otherField := ot.Fields[i]
			if !field.Name.Equals(otherField.Name) ||
				!field.Type.(TypeSafeNode).Equals(otherField.Type.(TypeSafeNode)) ||
				field.IsPublic != otherField.IsPublic {
				return false
			}
		}
		return true
	}
	return false
}
func (st *StructType) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, len(st.Fields)*2+1)
	if st.Name != nil {
		children = append(children, st.Name)
	}
	for _, field := range st.Fields {
		if field.Name != nil {
			children = append(children, field.Name)
		}
		if field.Type != nil {
			children = append(children, field.Type.(TypeSafeNode))
		}
	}
	return children
}
func (st *StructType) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("ReplaceChild not implemented for StructType")
}

// EnumType represents an enum type
type EnumType struct {
	Span     Span
	Name     *Identifier
	Variants []*EnumVariant
}

type EnumVariant struct {
	Span   Span
	Name   *Identifier
	Fields []*StructField // Optional associated data
	Value  Expression     // Optional explicit value
}

func (et *EnumType) GetSpan() Span { return et.Span }
func (et *EnumType) String() string {
	if et.Name != nil {
		return fmt.Sprintf("enum %s", et.Name.Value)
	}
	return "enum { ... }"
}
func (et *EnumType) Accept(visitor Visitor) interface{} { return visitor.VisitEnumType(et) }
func (et *EnumType) typeNode()                          {}
func (et *EnumType) GetNodeKind() NodeKind              { return NodeKindEnumType }
func (et *EnumType) Clone() TypeSafeNode {
	clone := *et
	if et.Name != nil {
		clone.Name = et.Name.Clone().(*Identifier)
	}
	clone.Variants = make([]*EnumVariant, len(et.Variants))
	for i, variant := range et.Variants {
		variantClone := *variant
		if variant.Name != nil {
			variantClone.Name = variant.Name.Clone().(*Identifier)
		}
		if variant.Value != nil {
			variantClone.Value = variant.Value.(TypeSafeNode).Clone().(Expression)
		}
		variantClone.Fields = make([]*StructField, len(variant.Fields))
		for j, field := range variant.Fields {
			fieldClone := *field
			if field.Name != nil {
				fieldClone.Name = field.Name.Clone().(*Identifier)
			}
			if field.Type != nil {
				fieldClone.Type = field.Type.(TypeSafeNode).Clone().(Type)
			}
			variantClone.Fields[j] = &fieldClone
		}
		clone.Variants[i] = &variantClone
	}
	return &clone
}
func (et *EnumType) Equals(other TypeSafeNode) bool {
	if ot, ok := other.(*EnumType); ok {
		if len(et.Variants) != len(ot.Variants) {
			return false
		}
		if !((et.Name == nil && ot.Name == nil) ||
			(et.Name != nil && ot.Name != nil && et.Name.Equals(ot.Name))) {
			return false
		}
		for i, variant := range et.Variants {
			otherVariant := ot.Variants[i]
			if !variant.Name.Equals(otherVariant.Name) ||
				len(variant.Fields) != len(otherVariant.Fields) {
				return false
			}
			if !((variant.Value == nil && otherVariant.Value == nil) ||
				(variant.Value != nil && otherVariant.Value != nil &&
					variant.Value.(TypeSafeNode).Equals(otherVariant.Value.(TypeSafeNode)))) {
				return false
			}
		}
		return true
	}
	return false
}
func (et *EnumType) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0)
	if et.Name != nil {
		children = append(children, et.Name)
	}
	for _, variant := range et.Variants {
		if variant.Name != nil {
			children = append(children, variant.Name)
		}
		if variant.Value != nil {
			children = append(children, variant.Value.(TypeSafeNode))
		}
		for _, field := range variant.Fields {
			if field.Name != nil {
				children = append(children, field.Name)
			}
			if field.Type != nil {
				children = append(children, field.Type.(TypeSafeNode))
			}
		}
	}
	return children
}
func (et *EnumType) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("ReplaceChild not implemented for EnumType")
}

// TraitType represents a trait type
type TraitType struct {
	Span    Span
	Name    *Identifier
	Methods []*TraitMethod
}

type TraitMethod struct {
	Span       Span
	Name       *Identifier
	Parameters []*Parameter
	ReturnType Type
	IsAsync    bool
	Generics   []*GenericParameter
}

func (tt *TraitType) GetSpan() Span { return tt.Span }
func (tt *TraitType) String() string {
	if tt.Name != nil {
		return fmt.Sprintf("trait %s", tt.Name.Value)
	}
	return "trait { ... }"
}
func (tt *TraitType) Accept(visitor Visitor) interface{} { return visitor.VisitTraitType(tt) }
func (tt *TraitType) typeNode()                          {}
func (tt *TraitType) GetNodeKind() NodeKind              { return NodeKindTraitType }
func (tt *TraitType) Clone() TypeSafeNode {
	clone := *tt
	if tt.Name != nil {
		clone.Name = tt.Name.Clone().(*Identifier)
	}
	clone.Methods = make([]*TraitMethod, len(tt.Methods))
	for i, method := range tt.Methods {
		methodClone := *method
		if method.Name != nil {
			methodClone.Name = method.Name.Clone().(*Identifier)
		}
		if method.ReturnType != nil {
			methodClone.ReturnType = method.ReturnType.(TypeSafeNode).Clone().(Type)
		}
		methodClone.Parameters = make([]*Parameter, len(method.Parameters))
		for j, param := range method.Parameters {
			methodClone.Parameters[j] = param.Clone().(*Parameter)
		}
		if method.Generics != nil {
			methodClone.Generics = make([]*GenericParameter, len(method.Generics))
			copy(methodClone.Generics, method.Generics)
		}
		clone.Methods[i] = &methodClone
	}
	return &clone
}
func (tt *TraitType) Equals(other TypeSafeNode) bool {
	if ot, ok := other.(*TraitType); ok {
		if len(tt.Methods) != len(ot.Methods) {
			return false
		}
		if !((tt.Name == nil && ot.Name == nil) ||
			(tt.Name != nil && ot.Name != nil && tt.Name.Equals(ot.Name))) {
			return false
		}
		for i, method := range tt.Methods {
			otherMethod := ot.Methods[i]
			if !method.Name.Equals(otherMethod.Name) ||
				method.IsAsync != otherMethod.IsAsync ||
				len(method.Parameters) != len(otherMethod.Parameters) {
				return false
			}
		}
		return true
	}
	return false
}
func (tt *TraitType) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0)
	if tt.Name != nil {
		children = append(children, tt.Name)
	}
	for _, method := range tt.Methods {
		if method.Name != nil {
			children = append(children, method.Name)
		}
		if method.ReturnType != nil {
			children = append(children, method.ReturnType.(TypeSafeNode))
		}
		for _, param := range method.Parameters {
			children = append(children, param)
		}
	}
	return children
}
func (tt *TraitType) ReplaceChild(index int, newChild TypeSafeNode) error {
	return fmt.Errorf("ReplaceChild not implemented for TraitType")
}

// GenericType represents a generic type with type parameters
type GenericType struct {
	Span           Span
	BaseType       Type
	TypeParameters []Type
}

// ReferenceType represents a reference type like &T or &mut T
type ReferenceType struct {
	Span      Span
	Inner     Type
	IsMutable bool
	Lifetime  string // optional, empty if elided
}

// PointerType represents a raw pointer type like *T or *mut T
type PointerType struct {
	Span      Span
	Inner     Type
	IsMutable bool
}

func (pt *PointerType) GetSpan() Span { return pt.Span }
func (pt *PointerType) String() string {
	if pt.IsMutable {
		return "*mut " + pt.Inner.String()
	}
	return "*" + pt.Inner.String()
}
func (pt *PointerType) Accept(visitor Visitor) interface{} { return visitor.VisitPointerType(pt) }
func (pt *PointerType) typeNode()                          {}
func (pt *PointerType) GetNodeKind() NodeKind              { return NodeKindPointerType }
func (pt *PointerType) Clone() TypeSafeNode {
	clone := *pt
	if pt.Inner != nil {
		clone.Inner = pt.Inner.(TypeSafeNode).Clone().(Type)
	}
	return &clone
}
func (pt *PointerType) Equals(other TypeSafeNode) bool {
	if o, ok := other.(*PointerType); ok {
		if pt.IsMutable != o.IsMutable {
			return false
		}
		if (pt.Inner == nil) != (o.Inner == nil) {
			return false
		}
		if pt.Inner == nil {
			return true
		}
		return pt.Inner.(TypeSafeNode).Equals(o.Inner.(TypeSafeNode))
	}
	return false
}
func (pt *PointerType) GetChildren() []TypeSafeNode {
	if pt.Inner == nil {
		return nil
	}
	return []TypeSafeNode{pt.Inner.(TypeSafeNode)}
}
func (pt *PointerType) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index != 0 || pt.Inner == nil {
		return fmt.Errorf("invalid child index %d for PointerType", index)
	}
	if nt, ok := newChild.(Type); ok {
		pt.Inner = nt
		return nil
	}
	return fmt.Errorf("expected Type, got %T", newChild)
}

func (rt *ReferenceType) GetSpan() Span { return rt.Span }
func (rt *ReferenceType) String() string {
	if rt.IsMutable {
		return "&mut " + rt.Inner.String()
	}
	return "&" + rt.Inner.String()
}
func (rt *ReferenceType) Accept(visitor Visitor) interface{} { return visitor.VisitReferenceType(rt) }
func (rt *ReferenceType) typeNode()                          {}
func (rt *ReferenceType) GetNodeKind() NodeKind              { return NodeKindReferenceType }
func (rt *ReferenceType) Clone() TypeSafeNode {
	clone := *rt
	if rt.Inner != nil {
		clone.Inner = rt.Inner.(TypeSafeNode).Clone().(Type)
	}
	return &clone
}
func (rt *ReferenceType) Equals(other TypeSafeNode) bool {
	if o, ok := other.(*ReferenceType); ok {
		if rt.IsMutable != o.IsMutable {
			return false
		}
		if (rt.Inner == nil) != (o.Inner == nil) {
			return false
		}
		if rt.Inner == nil {
			return true
		}
		return rt.Inner.(TypeSafeNode).Equals(o.Inner.(TypeSafeNode))
	}
	return false
}
func (rt *ReferenceType) GetChildren() []TypeSafeNode {
	if rt.Inner == nil {
		return nil
	}
	return []TypeSafeNode{rt.Inner.(TypeSafeNode)}
}
func (rt *ReferenceType) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index != 0 || rt.Inner == nil {
		return fmt.Errorf("invalid child index %d for ReferenceType", index)
	}
	if nt, ok := newChild.(Type); ok {
		rt.Inner = nt
		return nil
	}
	return fmt.Errorf("expected Type, got %T", newChild)
}

func (gt *GenericType) GetSpan() Span { return gt.Span }
func (gt *GenericType) String() string {
	var params []string
	for _, param := range gt.TypeParameters {
		params = append(params, param.String())
	}
	return fmt.Sprintf("%s<%s>", gt.BaseType.String(), strings.Join(params, ", "))
}
func (gt *GenericType) Accept(visitor Visitor) interface{} { return visitor.VisitGenericType(gt) }
func (gt *GenericType) typeNode()                          {}
func (gt *GenericType) GetNodeKind() NodeKind              { return NodeKindGenericType }
func (gt *GenericType) Clone() TypeSafeNode {
	clone := *gt
	if gt.BaseType != nil {
		clone.BaseType = gt.BaseType.(TypeSafeNode).Clone().(Type)
	}
	clone.TypeParameters = make([]Type, len(gt.TypeParameters))
	for i, param := range gt.TypeParameters {
		clone.TypeParameters[i] = param.(TypeSafeNode).Clone().(Type)
	}
	return &clone
}
func (gt *GenericType) Equals(other TypeSafeNode) bool {
	if ot, ok := other.(*GenericType); ok {
		if len(gt.TypeParameters) != len(ot.TypeParameters) {
			return false
		}
		if !gt.BaseType.(TypeSafeNode).Equals(ot.BaseType.(TypeSafeNode)) {
			return false
		}
		for i, param := range gt.TypeParameters {
			if !param.(TypeSafeNode).Equals(ot.TypeParameters[i].(TypeSafeNode)) {
				return false
			}
		}
		return true
	}
	return false
}
func (gt *GenericType) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, len(gt.TypeParameters)+1)
	if gt.BaseType != nil {
		children = append(children, gt.BaseType.(TypeSafeNode))
	}
	for _, param := range gt.TypeParameters {
		children = append(children, param.(TypeSafeNode))
	}
	return children
}
func (gt *GenericType) ReplaceChild(index int, newChild TypeSafeNode) error {
	children := gt.GetChildren()
	if index < 0 || index >= len(children) {
		return fmt.Errorf("index %d out of range for GenericType children", index)
	}
	newType, ok := newChild.(Type)
	if !ok {
		return fmt.Errorf("expected Type, got %T", newChild)
	}
	if index == 0 && gt.BaseType != nil {
		gt.BaseType = newType
		return nil
	}
	if index > 0 && index <= len(gt.TypeParameters) {
		gt.TypeParameters[index-1] = newType
		return nil
	}
	return fmt.Errorf("invalid child index %d for GenericType", index)
}
