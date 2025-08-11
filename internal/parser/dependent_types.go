// Dependent type system implementation for Orizon language
// This file implements the core dependent type functionality including
// type-level computation, dependent function types, and refinement types.

package parser

import (
	"fmt"
	"strings"
)

// ====== Dependent Type System Core ======

// DependentType represents a type that depends on a value
type DependentType interface {
	Type
	// GetDependency returns the value this type depends on
	GetDependency() Expression
	// Substitute replaces the dependent variable with a concrete value
	Substitute(variable string, value Expression) DependentType
	// IsRefined returns true if this is a refinement type
	IsRefined() bool
	// GetConstraints returns the constraints on the dependent value
	GetConstraints() []TypeConstraint
	// IsEquivalent checks if this type is equivalent to another
	IsEquivalent(other DependentType) bool
}

// TypeConstraint represents a constraint on a dependent type
type TypeConstraint struct {
	Span      Span
	Variable  *Identifier    // The variable being constrained
	Predicate Expression     // Boolean expression that must be true
	Kind      ConstraintKind // The kind of constraint
}

// ConstraintKind represents different kinds of type constraints
type ConstraintKind int

const (
	ConstraintEquality      ConstraintKind = iota // x == value
	ConstraintRange                               // min <= x <= max
	ConstraintPredicate                           // arbitrary boolean expression
	ConstraintInvariant                           // invariant that must always hold
	ConstraintPrecondition                        // precondition for function calls
	ConstraintPostcondition                       // postcondition for function returns
)

// String returns a string representation of the constraint kind
func (ck ConstraintKind) String() string {
	switch ck {
	case ConstraintEquality:
		return "equality"
	case ConstraintRange:
		return "range"
	case ConstraintPredicate:
		return "predicate"
	case ConstraintInvariant:
		return "invariant"
	case ConstraintPrecondition:
		return "precondition"
	case ConstraintPostcondition:
		return "postcondition"
	default:
		return "unknown"
	}
}

// ====== Dependent Type Node Implementations ======

// DependentFunctionType represents a function type with dependent parameters
// For example: (x: Int) -> Vec<x>
type DependentFunctionType struct {
	Span        Span
	Parameters  []*DependentParameter
	ReturnType  DependentType
	Constraints []TypeConstraint
	IsTotal     bool // Whether the function is total (defined for all inputs)
}

// DependentParameter represents a parameter in a dependent function
type DependentParameter struct {
	Span        Span
	Name        *Identifier
	Type        DependentType
	IsImplicit  bool // Whether this parameter is implicit (inferred)
	Constraints []TypeConstraint
}

// RefinementType represents a type refined with a predicate
// For example: {x: Int | x > 0} (positive integers)
type RefinementType struct {
	Span      Span
	BaseType  Type
	Variable  *Identifier // The variable bound in the refinement
	Predicate Expression  // Boolean expression that must be true
}

// SizedArrayType represents an array type with size as a dependent value
// For example: Array<T, n> where n is a compile-time constant
type SizedArrayType struct {
	Span        Span
	ElementType Type
	Size        Expression // Dependent size expression
	SizeType    Type       // Type of the size (usually Nat or Int)
}

// IndexType represents a type that represents valid indices for an array
// For example: Index<n> represents integers from 0 to n-1
type IndexType struct {
	Span     Span
	MaxValue Expression // Maximum value (exclusive)
}

// ProofType represents a type that encodes a logical proposition
// For example: Proof<x + y == y + x> represents a proof of commutativity
type ProofType struct {
	Span        Span
	Proposition Expression // The proposition being proved
}

// ====== AST Node Interface Implementations ======

// DependentFunctionType implementations
func (dft *DependentFunctionType) GetSpan() Span { return dft.Span }
func (dft *DependentFunctionType) String() string {
	var params []string
	for _, param := range dft.Parameters {
		params = append(params, param.String())
	}
	return fmt.Sprintf("(%s) -> %s", strings.Join(params, ", "), dft.ReturnType.String())
}
func (dft *DependentFunctionType) Accept(visitor Visitor) interface{} {
	return visitor.VisitDependentFunctionType(dft)
}
func (dft *DependentFunctionType) typeNode()             {}
func (dft *DependentFunctionType) GetNodeKind() NodeKind { return NodeKindDependentFunctionType }

// DependentType interface implementations for DependentFunctionType
func (dft *DependentFunctionType) GetDependency() Expression {
	// Return the first parameter as the primary dependency
	if len(dft.Parameters) > 0 {
		return dft.Parameters[0].Name
	}
	return nil
}
func (dft *DependentFunctionType) Substitute(variable string, value Expression) DependentType {
	// Create a copy with substituted types
	newParams := make([]*DependentParameter, len(dft.Parameters))
	for i, param := range dft.Parameters {
		newParams[i] = param.Substitute(variable, value)
	}
	newReturnType := dft.ReturnType.Substitute(variable, value)

	return &DependentFunctionType{
		Span:        dft.Span,
		Parameters:  newParams,
		ReturnType:  newReturnType,
		Constraints: dft.Constraints, // TODO: substitute in constraints
		IsTotal:     dft.IsTotal,
	}
}
func (dft *DependentFunctionType) IsRefined() bool {
	return len(dft.Constraints) > 0
}
func (dft *DependentFunctionType) GetConstraints() []TypeConstraint {
	return dft.Constraints
}

// IsEquivalent checks if this dependent function type is equivalent to another
func (dft *DependentFunctionType) IsEquivalent(other DependentType) bool {
	if otherFunc, ok := other.(*DependentFunctionType); ok {
		if len(dft.Parameters) != len(otherFunc.Parameters) {
			return false
		}
		for i, param := range dft.Parameters {
			if param.Name.Value != otherFunc.Parameters[i].Name.Value ||
				!param.Type.IsEquivalent(otherFunc.Parameters[i].Type) {
				return false
			}
		}
		return dft.ReturnType.IsEquivalent(otherFunc.ReturnType)
	}
	return false
}

// TypeSafeNode implementations for DependentFunctionType
func (dft *DependentFunctionType) Clone() TypeSafeNode {
	clone := *dft
	clone.Parameters = make([]*DependentParameter, len(dft.Parameters))
	for i, param := range dft.Parameters {
		clone.Parameters[i] = param.Clone().(*DependentParameter)
	}
	if dft.ReturnType != nil {
		clone.ReturnType = dft.ReturnType.(TypeSafeNode).Clone().(DependentType)
	}
	clone.Constraints = make([]TypeConstraint, len(dft.Constraints))
	copy(clone.Constraints, dft.Constraints)
	return &clone
}
func (dft *DependentFunctionType) Equals(other TypeSafeNode) bool {
	if o, ok := other.(*DependentFunctionType); ok {
		return len(dft.Parameters) == len(o.Parameters) &&
			dft.IsTotal == o.IsTotal &&
			((dft.ReturnType == nil && o.ReturnType == nil) ||
				(dft.ReturnType != nil && o.ReturnType != nil &&
					dft.ReturnType.(TypeSafeNode).Equals(o.ReturnType.(TypeSafeNode))))
	}
	return false
}
func (dft *DependentFunctionType) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, len(dft.Parameters)+1)
	for _, param := range dft.Parameters {
		children = append(children, param)
	}
	if dft.ReturnType != nil {
		children = append(children, dft.ReturnType.(TypeSafeNode))
	}
	return children
}
func (dft *DependentFunctionType) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index < 0 || index >= len(dft.Parameters)+1 {
		return fmt.Errorf("index %d out of range for DependentFunctionType children", index)
	}
	if index < len(dft.Parameters) {
		if newParam, ok := newChild.(*DependentParameter); ok {
			dft.Parameters[index] = newParam
			return nil
		}
		return fmt.Errorf("expected DependentParameter, got %T", newChild)
	}
	// Return type
	if newType, ok := newChild.(DependentType); ok {
		dft.ReturnType = newType
		return nil
	}
	return fmt.Errorf("expected DependentType for return type, got %T", newChild)
}

// DependentParameter implementations
func (dp *DependentParameter) GetSpan() Span { return dp.Span }
func (dp *DependentParameter) String() string {
	implicit := ""
	if dp.IsImplicit {
		implicit = "implicit "
	}
	constraints := ""
	if len(dp.Constraints) > 0 {
		constraints = fmt.Sprintf(" | %d constraints", len(dp.Constraints))
	}
	return fmt.Sprintf("%s%s: %s%s", implicit, dp.Name.Value, dp.Type.String(), constraints)
}
func (dp *DependentParameter) Accept(visitor Visitor) interface{} {
	return visitor.VisitDependentParameter(dp)
}
func (dp *DependentParameter) GetNodeKind() NodeKind { return NodeKindDependentParameter }

// Substitute method for DependentParameter
func (dp *DependentParameter) Substitute(variable string, value Expression) *DependentParameter {
	newType := dp.Type.Substitute(variable, value)
	return &DependentParameter{
		Span:        dp.Span,
		Name:        dp.Name,
		Type:        newType,
		IsImplicit:  dp.IsImplicit,
		Constraints: dp.Constraints, // TODO: substitute in constraints
	}
}

// TypeSafeNode implementations for DependentParameter
func (dp *DependentParameter) Clone() TypeSafeNode {
	clone := *dp
	if dp.Name != nil {
		clone.Name = dp.Name.Clone().(*Identifier)
	}
	if dp.Type != nil {
		clone.Type = dp.Type.(TypeSafeNode).Clone().(DependentType)
	}
	clone.Constraints = make([]TypeConstraint, len(dp.Constraints))
	copy(clone.Constraints, dp.Constraints)
	return &clone
}
func (dp *DependentParameter) Equals(other TypeSafeNode) bool {
	if o, ok := other.(*DependentParameter); ok {
		return dp.IsImplicit == o.IsImplicit &&
			((dp.Name == nil && o.Name == nil) ||
				(dp.Name != nil && o.Name != nil && dp.Name.Equals(o.Name))) &&
			((dp.Type == nil && o.Type == nil) ||
				(dp.Type != nil && o.Type != nil &&
					dp.Type.(TypeSafeNode).Equals(o.Type.(TypeSafeNode))))
	}
	return false
}
func (dp *DependentParameter) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, 2)
	if dp.Name != nil {
		children = append(children, dp.Name)
	}
	if dp.Type != nil {
		children = append(children, dp.Type.(TypeSafeNode))
	}
	return children
}
func (dp *DependentParameter) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index < 0 || index >= 2 {
		return fmt.Errorf("index %d out of range for DependentParameter children", index)
	}
	if index == 0 {
		if newName, ok := newChild.(*Identifier); ok {
			dp.Name = newName
			return nil
		}
		return fmt.Errorf("expected Identifier for name, got %T", newChild)
	}
	if newType, ok := newChild.(DependentType); ok {
		dp.Type = newType
		return nil
	}
	return fmt.Errorf("expected DependentType for type, got %T", newChild)
}

// RefinementType implementations
func (rt *RefinementType) GetSpan() Span { return rt.Span }
func (rt *RefinementType) String() string {
	return fmt.Sprintf("{%s: %s | %s}", rt.Variable.Value, rt.BaseType.String(), rt.Predicate.String())
}
func (rt *RefinementType) Accept(visitor Visitor) interface{} {
	return visitor.VisitRefinementType(rt)
}
func (rt *RefinementType) typeNode()             {}
func (rt *RefinementType) GetNodeKind() NodeKind { return NodeKindRefinementType }

// DependentType interface implementations for RefinementType
func (rt *RefinementType) GetDependency() Expression {
	return rt.Variable
}
func (rt *RefinementType) Substitute(variable string, value Expression) DependentType {
	if rt.Variable.Value == variable {
		// TODO: Replace the variable in the predicate
		return rt // For now, return unchanged
	}
	return rt
}
func (rt *RefinementType) IsRefined() bool {
	return true
}
func (rt *RefinementType) GetConstraints() []TypeConstraint {
	return []TypeConstraint{
		{
			Span:      rt.Span,
			Variable:  rt.Variable,
			Predicate: rt.Predicate,
			Kind:      ConstraintPredicate,
		},
	}
}

// IsEquivalent checks if this refinement type is equivalent to another
func (rt *RefinementType) IsEquivalent(other DependentType) bool {
	if otherRef, ok := other.(*RefinementType); ok {
		baseEquiv := false
		if bt1, ok1 := rt.BaseType.(*BasicType); ok1 {
			if bt2, ok2 := otherRef.BaseType.(*BasicType); ok2 {
				baseEquiv = bt1.Name == bt2.Name
			}
		}
		return baseEquiv &&
			rt.Variable.Value == otherRef.Variable.Value &&
			rt.Predicate.String() == otherRef.Predicate.String()
	}
	return false
}

// TypeSafeNode implementations for RefinementType
func (rt *RefinementType) Clone() TypeSafeNode {
	clone := *rt
	if rt.BaseType != nil {
		clone.BaseType = rt.BaseType.(TypeSafeNode).Clone().(Type)
	}
	if rt.Variable != nil {
		clone.Variable = rt.Variable.Clone().(*Identifier)
	}
	if rt.Predicate != nil {
		clone.Predicate = rt.Predicate.(TypeSafeNode).Clone().(Expression)
	}
	return &clone
}
func (rt *RefinementType) Equals(other TypeSafeNode) bool {
	if o, ok := other.(*RefinementType); ok {
		return ((rt.BaseType == nil && o.BaseType == nil) ||
			(rt.BaseType != nil && o.BaseType != nil &&
				rt.BaseType.(TypeSafeNode).Equals(o.BaseType.(TypeSafeNode)))) &&
			((rt.Variable == nil && o.Variable == nil) ||
				(rt.Variable != nil && o.Variable != nil && rt.Variable.Equals(o.Variable))) &&
			((rt.Predicate == nil && o.Predicate == nil) ||
				(rt.Predicate != nil && o.Predicate != nil &&
					rt.Predicate.(TypeSafeNode).Equals(o.Predicate.(TypeSafeNode))))
	}
	return false
}
func (rt *RefinementType) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, 3)
	if rt.BaseType != nil {
		children = append(children, rt.BaseType.(TypeSafeNode))
	}
	if rt.Variable != nil {
		children = append(children, rt.Variable)
	}
	if rt.Predicate != nil {
		children = append(children, rt.Predicate.(TypeSafeNode))
	}
	return children
}
func (rt *RefinementType) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index < 0 || index >= 3 {
		return fmt.Errorf("index %d out of range for RefinementType children", index)
	}
	switch index {
	case 0:
		if newType, ok := newChild.(Type); ok {
			rt.BaseType = newType
			return nil
		}
		return fmt.Errorf("expected Type for base type, got %T", newChild)
	case 1:
		if newVar, ok := newChild.(*Identifier); ok {
			rt.Variable = newVar
			return nil
		}
		return fmt.Errorf("expected Identifier for variable, got %T", newChild)
	case 2:
		if newPred, ok := newChild.(Expression); ok {
			rt.Predicate = newPred
			return nil
		}
		return fmt.Errorf("expected Expression for predicate, got %T", newChild)
	}
	return fmt.Errorf("invalid child index %d for RefinementType", index)
}

// SizedArrayType implementations
func (sat *SizedArrayType) GetSpan() Span { return sat.Span }
func (sat *SizedArrayType) String() string {
	return fmt.Sprintf("Array<%s, %s>", sat.ElementType.String(), sat.Size.String())
}
func (sat *SizedArrayType) Accept(visitor Visitor) interface{} {
	return visitor.VisitSizedArrayType(sat)
}
func (sat *SizedArrayType) typeNode()             {}
func (sat *SizedArrayType) GetNodeKind() NodeKind { return NodeKindSizedArrayType }

// DependentType interface implementations for SizedArrayType
func (sat *SizedArrayType) GetDependency() Expression {
	return sat.Size
}
func (sat *SizedArrayType) Substitute(variable string, value Expression) DependentType {
	// TODO: Implement substitution in size expression
	return sat
}
func (sat *SizedArrayType) IsRefined() bool {
	return false
}
func (sat *SizedArrayType) GetConstraints() []TypeConstraint {
	// Size must be non-negative
	return []TypeConstraint{
		{
			Span:     sat.Span,
			Variable: &Identifier{Span: sat.Span, Value: "size"},
			Predicate: &BinaryExpression{
				Span:     sat.Span,
				Left:     sat.Size,
				Operator: &Operator{Span: sat.Span, Value: ">=", Kind: BinaryOp},
				Right:    &Literal{Span: sat.Span, Value: 0, Kind: LiteralInteger},
			},
			Kind: ConstraintRange,
		},
	}
}

// IsEquivalent checks if this sized array type is equivalent to another
func (sat *SizedArrayType) IsEquivalent(other DependentType) bool {
	if otherArray, ok := other.(*SizedArrayType); ok {
		elemEquiv := false
		if elem1, ok1 := sat.ElementType.(*BasicType); ok1 {
			if elem2, ok2 := otherArray.ElementType.(*BasicType); ok2 {
				elemEquiv = elem1.Name == elem2.Name
			}
		}
		return elemEquiv && sat.Size.String() == otherArray.Size.String()
	}
	return false
}

// TypeSafeNode implementations for SizedArrayType
func (sat *SizedArrayType) Clone() TypeSafeNode {
	clone := *sat
	if sat.ElementType != nil {
		clone.ElementType = sat.ElementType.(TypeSafeNode).Clone().(Type)
	}
	if sat.Size != nil {
		clone.Size = sat.Size.(TypeSafeNode).Clone().(Expression)
	}
	if sat.SizeType != nil {
		clone.SizeType = sat.SizeType.(TypeSafeNode).Clone().(Type)
	}
	return &clone
}
func (sat *SizedArrayType) Equals(other TypeSafeNode) bool {
	if o, ok := other.(*SizedArrayType); ok {
		return ((sat.ElementType == nil && o.ElementType == nil) ||
			(sat.ElementType != nil && o.ElementType != nil &&
				sat.ElementType.(TypeSafeNode).Equals(o.ElementType.(TypeSafeNode)))) &&
			((sat.Size == nil && o.Size == nil) ||
				(sat.Size != nil && o.Size != nil &&
					sat.Size.(TypeSafeNode).Equals(o.Size.(TypeSafeNode)))) &&
			((sat.SizeType == nil && o.SizeType == nil) ||
				(sat.SizeType != nil && o.SizeType != nil &&
					sat.SizeType.(TypeSafeNode).Equals(o.SizeType.(TypeSafeNode))))
	}
	return false
}
func (sat *SizedArrayType) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, 3)
	if sat.ElementType != nil {
		children = append(children, sat.ElementType.(TypeSafeNode))
	}
	if sat.Size != nil {
		children = append(children, sat.Size.(TypeSafeNode))
	}
	if sat.SizeType != nil {
		children = append(children, sat.SizeType.(TypeSafeNode))
	}
	return children
}
func (sat *SizedArrayType) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index < 0 || index >= 3 {
		return fmt.Errorf("index %d out of range for SizedArrayType children", index)
	}
	switch index {
	case 0:
		if newType, ok := newChild.(Type); ok {
			sat.ElementType = newType
			return nil
		}
		return fmt.Errorf("expected Type for element type, got %T", newChild)
	case 1:
		if newSize, ok := newChild.(Expression); ok {
			sat.Size = newSize
			return nil
		}
		return fmt.Errorf("expected Expression for size, got %T", newChild)
	case 2:
		if newSizeType, ok := newChild.(Type); ok {
			sat.SizeType = newSizeType
			return nil
		}
		return fmt.Errorf("expected Type for size type, got %T", newChild)
	}
	return fmt.Errorf("invalid child index %d for SizedArrayType", index)
}

// IndexType implementations
func (it *IndexType) GetSpan() Span { return it.Span }
func (it *IndexType) String() string {
	return fmt.Sprintf("Index<%s>", it.MaxValue.String())
}
func (it *IndexType) Accept(visitor Visitor) interface{} {
	return visitor.VisitIndexType(it)
}
func (it *IndexType) typeNode()             {}
func (it *IndexType) GetNodeKind() NodeKind { return NodeKindIndexType }

// DependentType interface implementations for IndexType
func (it *IndexType) GetDependency() Expression {
	return it.MaxValue
}
func (it *IndexType) Substitute(variable string, value Expression) DependentType {
	// TODO: Implement substitution in max value expression
	return it
}
func (it *IndexType) IsRefined() bool {
	return true
}
func (it *IndexType) GetConstraints() []TypeConstraint {
	return []TypeConstraint{
		{
			Span:     it.Span,
			Variable: &Identifier{Span: it.Span, Value: "index"},
			Predicate: &BinaryExpression{
				Span:     it.Span,
				Left:     &Identifier{Span: it.Span, Value: "index"},
				Operator: &Operator{Span: it.Span, Value: ">=", Kind: BinaryOp},
				Right:    &Literal{Span: it.Span, Value: 0, Kind: LiteralInteger},
			},
			Kind: ConstraintRange,
		},
		{
			Span:     it.Span,
			Variable: &Identifier{Span: it.Span, Value: "index"},
			Predicate: &BinaryExpression{
				Span:     it.Span,
				Left:     &Identifier{Span: it.Span, Value: "index"},
				Operator: &Operator{Span: it.Span, Value: "<", Kind: BinaryOp},
				Right:    it.MaxValue,
			},
			Kind: ConstraintRange,
		},
	}
}

// IsEquivalent checks if this index type is equivalent to another
func (it *IndexType) IsEquivalent(other DependentType) bool {
	if otherIndex, ok := other.(*IndexType); ok {
		return it.MaxValue.String() == otherIndex.MaxValue.String()
	}
	return false
}

// TypeSafeNode implementations for IndexType
func (it *IndexType) Clone() TypeSafeNode {
	clone := *it
	if it.MaxValue != nil {
		clone.MaxValue = it.MaxValue.(TypeSafeNode).Clone().(Expression)
	}
	return &clone
}
func (it *IndexType) Equals(other TypeSafeNode) bool {
	if o, ok := other.(*IndexType); ok {
		return ((it.MaxValue == nil && o.MaxValue == nil) ||
			(it.MaxValue != nil && o.MaxValue != nil &&
				it.MaxValue.(TypeSafeNode).Equals(o.MaxValue.(TypeSafeNode))))
	}
	return false
}
func (it *IndexType) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, 1)
	if it.MaxValue != nil {
		children = append(children, it.MaxValue.(TypeSafeNode))
	}
	return children
}
func (it *IndexType) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index != 0 {
		return fmt.Errorf("index %d out of range for IndexType children", index)
	}
	if newMax, ok := newChild.(Expression); ok {
		it.MaxValue = newMax
		return nil
	}
	return fmt.Errorf("expected Expression for max value, got %T", newChild)
}

// ProofType implementations
func (pt *ProofType) GetSpan() Span { return pt.Span }
func (pt *ProofType) String() string {
	return fmt.Sprintf("Proof<%s>", pt.Proposition.String())
}
func (pt *ProofType) Accept(visitor Visitor) interface{} {
	return visitor.VisitProofType(pt)
}
func (pt *ProofType) typeNode()             {}
func (pt *ProofType) GetNodeKind() NodeKind { return NodeKindProofType }

// DependentType interface implementations for ProofType
func (pt *ProofType) GetDependency() Expression {
	return pt.Proposition
}
func (pt *ProofType) Substitute(variable string, value Expression) DependentType {
	// TODO: Implement substitution in proposition
	return pt
}
func (pt *ProofType) IsRefined() bool {
	return false
}
func (pt *ProofType) GetConstraints() []TypeConstraint {
	return []TypeConstraint{
		{
			Span:      pt.Span,
			Variable:  &Identifier{Span: pt.Span, Value: "proof"},
			Predicate: pt.Proposition,
			Kind:      ConstraintInvariant,
		},
	}
}

// IsEquivalent checks if this proof type is equivalent to another
func (pt *ProofType) IsEquivalent(other DependentType) bool {
	if otherProof, ok := other.(*ProofType); ok {
		return pt.Proposition.String() == otherProof.Proposition.String()
	}
	return false
}

// TypeSafeNode implementations for ProofType
func (pt *ProofType) Clone() TypeSafeNode {
	clone := *pt
	if pt.Proposition != nil {
		clone.Proposition = pt.Proposition.(TypeSafeNode).Clone().(Expression)
	}
	return &clone
}
func (pt *ProofType) Equals(other TypeSafeNode) bool {
	if o, ok := other.(*ProofType); ok {
		return ((pt.Proposition == nil && o.Proposition == nil) ||
			(pt.Proposition != nil && o.Proposition != nil &&
				pt.Proposition.(TypeSafeNode).Equals(o.Proposition.(TypeSafeNode))))
	}
	return false
}
func (pt *ProofType) GetChildren() []TypeSafeNode {
	children := make([]TypeSafeNode, 0, 1)
	if pt.Proposition != nil {
		children = append(children, pt.Proposition.(TypeSafeNode))
	}
	return children
}
func (pt *ProofType) ReplaceChild(index int, newChild TypeSafeNode) error {
	if index != 0 {
		return fmt.Errorf("index %d out of range for ProofType children", index)
	}
	if newProp, ok := newChild.(Expression); ok {
		pt.Proposition = newProp
		return nil
	}
	return fmt.Errorf("expected Expression for proposition, got %T", newChild)
}

// ====== Type Constraint Utilities ======

// EvaluateConstraint evaluates a type constraint for a given value
func EvaluateConstraint(constraint TypeConstraint, value interface{}) (bool, error) {
	switch constraint.Kind {
	case ConstraintEquality:
		// TODO: Implement equality checking
		return true, nil
	case ConstraintRange:
		// TODO: Implement range checking
		return true, nil
	case ConstraintPredicate:
		// TODO: Implement predicate evaluation
		return true, nil
	default:
		return false, fmt.Errorf("unsupported constraint kind: %s", constraint.Kind)
	}
}

// CheckConstraints checks all constraints for a dependent type
func CheckConstraints(depType DependentType, value interface{}) []error {
	var errors []error
	constraints := depType.GetConstraints()

	for _, constraint := range constraints {
		satisfied, err := EvaluateConstraint(constraint, value)
		if err != nil {
			errors = append(errors, fmt.Errorf("error evaluating constraint: %w", err))
		} else if !satisfied {
			errors = append(errors, fmt.Errorf("constraint not satisfied: %s", constraint.Predicate.String()))
		}
	}

	return errors
}

// ====== Dependent Type Utilities ======

// InferDependentType attempts to infer a dependent type from an expression
func InferDependentType(expr Expression) (DependentType, error) {
	switch e := expr.(type) {
	case *Literal:
		// For literal values, create a refinement type
		baseType := inferLiteralType(e)
		variable := &Identifier{Span: e.Span, Value: "x"}
		predicate := &BinaryExpression{
			Span:     e.Span,
			Left:     variable,
			Operator: &Operator{Span: e.Span, Value: "==", Kind: BinaryOp},
			Right:    e,
		}
		return &RefinementType{
			Span:      e.Span,
			BaseType:  baseType,
			Variable:  variable,
			Predicate: predicate,
		}, nil
	default:
		return nil, fmt.Errorf("cannot infer dependent type for expression: %T", expr)
	}
}

// inferLiteralType infers the base type for a literal
func inferLiteralType(lit *Literal) Type {
	switch lit.Kind {
	case LiteralInteger:
		return &BasicType{Span: lit.Span, Name: "Int"}
	case LiteralFloat:
		return &BasicType{Span: lit.Span, Name: "Float"}
	case LiteralString:
		return &BasicType{Span: lit.Span, Name: "String"}
	case LiteralBool:
		return &BasicType{Span: lit.Span, Name: "Bool"}
	default:
		return &BasicType{Span: lit.Span, Name: "Any"}
	}
}

// SimplifyDependentType attempts to simplify a dependent type by evaluating constraints
func SimplifyDependentType(depType DependentType) DependentType {
	// TODO: Implement type simplification
	return depType
}

// UnifyDependentTypes attempts to unify two dependent types
func UnifyDependentTypes(type1, type2 DependentType) (DependentType, error) {
	// TODO: Implement dependent type unification
	return nil, fmt.Errorf("dependent type unification not yet implemented")
}
