// Constraint generator for the Orizon type system
// This module generates type constraints from HIR nodes for type inference

package types

import (
	"fmt"

	"github.com/orizon-lang/orizon/internal/hir"
)

// ConstraintGenerator generates type constraints from HIR expressions
// This is used during type inference to build constraint systems
type ConstraintGenerator struct {
	constraints []*TypeInferenceConstraint
	nextTypeVar int
}

// TypeInferenceConstraint represents a constraint between two types during inference
type TypeInferenceConstraint struct {
	Left  *Type
	Right *Type
	Kind  TypeConstraintKind
}

// TypeConstraintKind represents the kind of type constraint
type TypeConstraintKind int

const (
	TypeConstraintEqual TypeConstraintKind = iota // Left = Right
	TypeConstraintSub                             // Left <: Right (subtype)
	TypeConstraintSuper                           // Left :> Right (supertype)
)

// NewConstraintGenerator creates a new constraint generator
func NewConstraintGenerator() *ConstraintGenerator {
	return &ConstraintGenerator{
		constraints: make([]*TypeInferenceConstraint, 0),
		nextTypeVar: 0,
	}
}

// GenerateConstraints generates constraints from an expression
// This is the main entry point for constraint generation
func (cg *ConstraintGenerator) GenerateConstraints(expr interface{}) (*Type, []*TypeInferenceConstraint, error) {
	switch e := expr.(type) {
	case *hir.HIRLiteral:
		return cg.generateLiteralConstraints(e)
	case *hir.HIRIdentifier:
		return cg.generateIdentifierConstraints(e)
	case *hir.HIRBinaryExpression:
		return cg.generateBinaryConstraints(e)
	case *hir.HIRUnaryExpression:
		return cg.generateUnaryConstraints(e)
	case *hir.HIRCallExpression:
		return cg.generateCallConstraints(e)
	case *hir.HIRFieldExpression:
		return cg.generateFieldConstraints(e)
	case *hir.HIRIndexExpression:
		return cg.generateIndexConstraints(e)
	default:
		// Fallback for unknown expression types
		voidType := &Type{Kind: TypeKindVoid}
		return voidType, cg.constraints, nil
	}
}

// generateLiteralConstraints generates constraints for literal expressions
func (cg *ConstraintGenerator) generateLiteralConstraints(lit *hir.HIRLiteral) (*Type, []*TypeInferenceConstraint, error) {
	// Determine type based on value
	switch lit.Value.(type) {
	case int, int32, int64:
		return &Type{Kind: TypeKindInt32}, cg.constraints, nil
	case float32, float64:
		return &Type{Kind: TypeKindFloat64}, cg.constraints, nil
	case string:
		return &Type{Kind: TypeKindString}, cg.constraints, nil
	case bool:
		return &Type{Kind: TypeKindBool}, cg.constraints, nil
	default:
		return &Type{Kind: TypeKindVoid}, cg.constraints, fmt.Errorf("unknown literal type")
	}
}

// generateIdentifierConstraints generates constraints for identifier expressions
func (cg *ConstraintGenerator) generateIdentifierConstraints(ident *hir.HIRIdentifier) (*Type, []*TypeInferenceConstraint, error) {
	// Create a fresh type variable for the identifier
	// In a real implementation, this would look up the identifier in the environment
	typeVar := cg.FreshTypeVariable()
	return typeVar, cg.constraints, nil
}

// generateBinaryConstraints generates constraints for binary expressions
func (cg *ConstraintGenerator) generateBinaryConstraints(bin *hir.HIRBinaryExpression) (*Type, []*TypeInferenceConstraint, error) {
	// Generate constraints for operands
	leftType, leftConstraints, err := cg.GenerateConstraints(bin.Left)
	if err != nil {
		return nil, nil, err
	}

	rightType, rightConstraints, err := cg.GenerateConstraints(bin.Right)
	if err != nil {
		return nil, nil, err
	}

	// Merge constraints
	cg.constraints = append(cg.constraints, leftConstraints...)
	cg.constraints = append(cg.constraints, rightConstraints...)

	// Generate constraints based on operator
	resultType := cg.FreshTypeVariable()
	switch bin.Operator {
	case "+", "-", "*", "/", "%":
		// Arithmetic: left = right = result
		cg.AddConstraint(&TypeInferenceConstraint{leftType, rightType, TypeConstraintEqual})
		cg.AddConstraint(&TypeInferenceConstraint{leftType, resultType, TypeConstraintEqual})
	case "==", "!=", "<", ">", "<=", ">=":
		// Comparison: left = right, result = bool
		cg.AddConstraint(&TypeInferenceConstraint{leftType, rightType, TypeConstraintEqual})
		boolType := &Type{Kind: TypeKindBool}
		cg.AddConstraint(&TypeInferenceConstraint{resultType, boolType, TypeConstraintEqual})
	case "&&", "||":
		// Logical: left = right = result = bool
		boolType := &Type{Kind: TypeKindBool}
		cg.AddConstraint(&TypeInferenceConstraint{leftType, boolType, TypeConstraintEqual})
		cg.AddConstraint(&TypeInferenceConstraint{rightType, boolType, TypeConstraintEqual})
		cg.AddConstraint(&TypeInferenceConstraint{resultType, boolType, TypeConstraintEqual})
	}

	return resultType, cg.constraints, nil
}

// generateUnaryConstraints generates constraints for unary expressions
func (cg *ConstraintGenerator) generateUnaryConstraints(unary *hir.HIRUnaryExpression) (*Type, []*TypeInferenceConstraint, error) {
	operandType, operandConstraints, err := cg.GenerateConstraints(unary.Operand)
	if err != nil {
		return nil, nil, err
	}

	cg.constraints = append(cg.constraints, operandConstraints...)

	resultType := cg.FreshTypeVariable()
	switch unary.Operator {
	case "-", "+":
		// Unary arithmetic: operand = result
		cg.AddConstraint(&TypeInferenceConstraint{operandType, resultType, TypeConstraintEqual})
	case "!":
		// Logical not: operand = result = bool
		boolType := &Type{Kind: TypeKindBool}
		cg.AddConstraint(&TypeInferenceConstraint{operandType, boolType, TypeConstraintEqual})
		cg.AddConstraint(&TypeInferenceConstraint{resultType, boolType, TypeConstraintEqual})
	case "*":
		// Dereference: result is pointed-to type
		ptrType := &Type{Kind: TypeKindPointer, Data: &PointerType{PointeeType: resultType}}
		cg.AddConstraint(&TypeInferenceConstraint{operandType, ptrType, TypeConstraintEqual})
	case "&":
		// Address-of: result is pointer to operand
		ptrType := &Type{Kind: TypeKindPointer, Data: &PointerType{PointeeType: operandType}}
		cg.AddConstraint(&TypeInferenceConstraint{resultType, ptrType, TypeConstraintEqual})
	}

	return resultType, cg.constraints, nil
}

// generateCallConstraints generates constraints for function call expressions
func (cg *ConstraintGenerator) generateCallConstraints(call *hir.HIRCallExpression) (*Type, []*TypeInferenceConstraint, error) {
	// Generate constraints for function
	funcType, funcConstraints, err := cg.GenerateConstraints(call.Function)
	if err != nil {
		return nil, nil, err
	}

	cg.constraints = append(cg.constraints, funcConstraints...)

	// Generate constraints for arguments
	argTypes := make([]*Type, len(call.Arguments))
	for i, arg := range call.Arguments {
		argType, argConstraints, err := cg.GenerateConstraints(arg)
		if err != nil {
			return nil, nil, err
		}
		argTypes[i] = argType
		cg.constraints = append(cg.constraints, argConstraints...)
	}

	// Create function type constraint
	returnType := cg.FreshTypeVariable()
	expectedFuncType := &Type{
		Kind: TypeKindFunction,
		Data: &FunctionType{
			Parameters: argTypes,
			ReturnType: returnType,
		},
	}

	cg.AddConstraint(&TypeInferenceConstraint{funcType, expectedFuncType, TypeConstraintEqual})

	return returnType, cg.constraints, nil
}

// generateFieldConstraints generates constraints for field access expressions
func (cg *ConstraintGenerator) generateFieldConstraints(field *hir.HIRFieldExpression) (*Type, []*TypeInferenceConstraint, error) {
	objectType, objectConstraints, err := cg.GenerateConstraints(field.Object)
	if err != nil {
		return nil, nil, err
	}

	cg.constraints = append(cg.constraints, objectConstraints...)

	// Create field type constraint
	fieldType := cg.FreshTypeVariable()

	// Object must be a struct with the given field
	// This is a simplified constraint - in practice, we'd need more sophisticated struct handling
	structType := &Type{
		Kind: TypeKindStruct,
		Data: &StructType{
			Fields: []StructField{{Name: field.Field, Type: fieldType}},
		},
	}

	cg.AddConstraint(&TypeInferenceConstraint{objectType, structType, TypeConstraintSub})

	return fieldType, cg.constraints, nil
}

// generateIndexConstraints generates constraints for index access expressions
func (cg *ConstraintGenerator) generateIndexConstraints(index *hir.HIRIndexExpression) (*Type, []*TypeInferenceConstraint, error) {
	arrayType, arrayConstraints, err := cg.GenerateConstraints(index.Array)
	if err != nil {
		return nil, nil, err
	}

	indexType, indexConstraints, err := cg.GenerateConstraints(index.Index)
	if err != nil {
		return nil, nil, err
	}

	cg.constraints = append(cg.constraints, arrayConstraints...)
	cg.constraints = append(cg.constraints, indexConstraints...)

	// Index must be an integer
	intType := &Type{Kind: TypeKindInt32}
	cg.AddConstraint(&TypeInferenceConstraint{indexType, intType, TypeConstraintEqual})

	// Array must be an array type, and result is element type
	elementType := cg.FreshTypeVariable()
	expectedArrayType := &Type{
		Kind: TypeKindArray,
		Data: &ArrayType{
			ElementType: elementType,
		},
	}

	cg.AddConstraint(&TypeInferenceConstraint{arrayType, expectedArrayType, TypeConstraintEqual})

	return elementType, cg.constraints, nil
}

// FreshTypeVariable creates a fresh type variable for constraint generation
func (cg *ConstraintGenerator) FreshTypeVariable() *Type {
	tv := NewTypeVar(cg.nextTypeVar, fmt.Sprintf("t%d", cg.nextTypeVar), nil)
	cg.nextTypeVar++
	return tv
}

// AddConstraint adds a constraint to the constraint set
func (cg *ConstraintGenerator) AddConstraint(constraint *TypeInferenceConstraint) {
	cg.constraints = append(cg.constraints, constraint)
}

// GetConstraints returns all generated constraints
func (cg *ConstraintGenerator) GetConstraints() []*TypeInferenceConstraint {
	return cg.constraints
}

// Reset clears all constraints and resets the generator state
func (cg *ConstraintGenerator) Reset() {
	cg.constraints = cg.constraints[:0]
	cg.nextTypeVar = 0
}
