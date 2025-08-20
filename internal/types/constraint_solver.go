// Package types implements Phase 2.2.2 constraint-based type inference for the Orizon compiler.
// This extends the Hindley-Milner system with advanced constraint solving capabilities.
package types

import (
	"errors"
	"fmt"
	"strings"
)

// SourceLocation represents source code location for error reporting.
type SourceLocation struct {
	File   string
	Line   int
	Column int
}

// ExtendedConstraintKind represents extended constraint types for constraint-based inference.
type ExtendedConstraintKind int

const (
	// ExtendedUnificationConstraint represents type equality constraints (t1 = t2).
	ExtendedUnificationConstraint ExtendedConstraintKind = iota

	// ExtendedSubtypeConstraint represents subtyping constraints (t1 <: t2).
	ExtendedSubtypeConstraint

	// ExtendedApplicationConstraint represents function application constraints.
	ExtendedApplicationConstraint
)

// String returns a string representation of the extended constraint kind.
func (eck ExtendedConstraintKind) String() string {
	switch eck {
	case ExtendedUnificationConstraint:
		return "unification"
	case ExtendedSubtypeConstraint:
		return "subtype"
	case ExtendedApplicationConstraint:
		return "application"
	default:
		return "unknown"
	}
}

// ExtendedConstraint represents an extended type constraint in the constraint system.
type ExtendedConstraint struct {
	Left        *Type
	Right       *Type
	Description string
	Location    SourceLocation
	Kind        ExtendedConstraintKind
}

// String returns a string representation of the extended constraint.
func (c *ExtendedConstraint) String() string {
	switch c.Kind {
	case ExtendedUnificationConstraint:
		return fmt.Sprintf("%s = %s", c.Left.String(), c.Right.String())
	case ExtendedSubtypeConstraint:
		return fmt.Sprintf("%s <: %s", c.Left.String(), c.Right.String())
	case ExtendedApplicationConstraint:
		return fmt.Sprintf("apply %s to %s", c.Left.String(), c.Right.String())
	default:
		return fmt.Sprintf("unknown constraint: %s", c.Kind.String())
	}
}

// ConstraintSet represents a collection of extended type constraints.
type ConstraintSet struct {
	constraints []*ExtendedConstraint
	solved      []*ExtendedConstraint
	deferred    []*ExtendedConstraint
}

// NewConstraintSet creates a new constraint set.
func NewConstraintSet() *ConstraintSet {
	return &ConstraintSet{
		constraints: make([]*ExtendedConstraint, 0),
		solved:      make([]*ExtendedConstraint, 0),
		deferred:    make([]*ExtendedConstraint, 0),
	}
}

// AddUnificationConstraint adds a unification constraint.
func (cs *ConstraintSet) AddUnificationConstraint(left, right *Type, loc SourceLocation, desc string) {
	constraint := &ExtendedConstraint{
		Kind:        ExtendedUnificationConstraint,
		Left:        left,
		Right:       right,
		Location:    loc,
		Description: desc,
	}
	cs.constraints = append(cs.constraints, constraint)
}

// GetConstraints returns all constraints in the set.
func (cs *ConstraintSet) GetConstraints() []*ExtendedConstraint {
	return cs.constraints
}

// MarkSolved marks a constraint as solved and moves it to the solved set.
func (cs *ConstraintSet) MarkSolved(constraint *ExtendedConstraint) {
	// Remove from constraints.
	for i, c := range cs.constraints {
		if c == constraint {
			cs.constraints = append(cs.constraints[:i], cs.constraints[i+1:]...)

			break
		}
	}

	// Add to solved.
	cs.solved = append(cs.solved, constraint)
}

// Size returns the total number of constraints.
func (cs *ConstraintSet) Size() int {
	return len(cs.constraints)
}

// IsEmpty returns true if there are no constraints to solve.
func (cs *ConstraintSet) IsEmpty() bool {
	return len(cs.constraints) == 0
}

// ConstraintSolver solves type constraints using various algorithms.
type ConstraintSolver struct {
	inference   *InferenceEngine
	maxSteps    int
	currentStep int
	verbose     bool
}

// NewConstraintSolver creates a new constraint solver.
func NewConstraintSolver(inference *InferenceEngine) *ConstraintSolver {
	return &ConstraintSolver{
		inference:   inference,
		maxSteps:    1000, // Prevent infinite loops
		currentStep: 0,
		verbose:     false,
	}
}

// SetVerbose enables or disables verbose output for debugging.
func (cs *ConstraintSolver) SetVerbose(verbose bool) {
	cs.verbose = verbose
}

// SolveConstraints attempts to solve a set of constraints.
func (cs *ConstraintSolver) SolveConstraints(constraints *ConstraintSet) error {
	cs.currentStep = 0

	if cs.verbose {
		fmt.Printf("Starting constraint solving with %d constraints\n", constraints.Size())
	}

	for !constraints.IsEmpty() && cs.currentStep < cs.maxSteps {
		cs.currentStep++

		if cs.verbose {
			fmt.Printf("Step %d: %d constraints remaining\n", cs.currentStep, constraints.Size())
		}

		// Try to solve one constraint.
		progress := false

		for _, constraint := range constraints.GetConstraints() {
			if cs.tryToSolveConstraint(constraint, constraints) {
				progress = true

				break
			}
		}

		// If no progress was made, we're stuck.
		if !progress {
			return cs.reportUnsolvableConstraints(constraints)
		}
	}

	if cs.currentStep >= cs.maxSteps {
		return fmt.Errorf("constraint solving exceeded maximum steps (%d)", cs.maxSteps)
	}

	if cs.verbose {
		fmt.Printf("Constraint solving completed in %d steps\n", cs.currentStep)
	}

	return nil
}

// tryToSolveConstraint attempts to solve a single constraint.
func (cs *ConstraintSolver) tryToSolveConstraint(constraint *ExtendedConstraint, constraints *ConstraintSet) bool {
	switch constraint.Kind {
	case ExtendedUnificationConstraint:
		return cs.solveUnificationConstraint(constraint, constraints)
	case ExtendedSubtypeConstraint:
		return cs.solveSubtypeConstraint(constraint, constraints)
	case ExtendedApplicationConstraint:
		return cs.solveApplicationConstraint(constraint, constraints)
	default:
		return false
	}
}

// solveUnificationConstraint solves unification constraints (t1 = t2).
func (cs *ConstraintSolver) solveUnificationConstraint(constraint *ExtendedConstraint, constraints *ConstraintSet) bool {
	left := cs.inference.ApplySubstitutions(constraint.Left)
	right := cs.inference.ApplySubstitutions(constraint.Right)

	if cs.verbose {
		fmt.Printf("  Solving unification: %s = %s\n", left.String(), right.String())
	}

	// Try to unify the types.
	err := cs.inference.Unify(left, right)
	if err != nil {
		if cs.verbose {
			fmt.Printf("    Failed: %v\n", err)
		}

		return false
	}

	// Constraint solved successfully.
	constraints.MarkSolved(constraint)

	return true
}

// solveSubtypeConstraint solves subtyping constraints (t1 <: t2).
func (cs *ConstraintSolver) solveSubtypeConstraint(constraint *ExtendedConstraint, constraints *ConstraintSet) bool {
	subtype := cs.inference.ApplySubstitutions(constraint.Left)
	supertype := cs.inference.ApplySubstitutions(constraint.Right)

	if cs.verbose {
		fmt.Printf("  Solving subtype: %s <: %s\n", subtype.String(), supertype.String())
	}

	// Check if subtype relationship holds.
	if cs.isSubtype(subtype, supertype) {
		constraints.MarkSolved(constraint)

		return true
	}

	return false
}

// solveApplicationConstraint solves function application constraints.
func (cs *ConstraintSolver) solveApplicationConstraint(constraint *ExtendedConstraint, constraints *ConstraintSet) bool {
	funcType := cs.inference.ApplySubstitutions(constraint.Left)
	argType := cs.inference.ApplySubstitutions(constraint.Right)

	if cs.verbose {
		fmt.Printf("  Solving application: %s applied to %s\n", funcType.String(), argType.String())
	}

	// Check if function type is callable with the given argument.
	if funcType.Kind != TypeKindFunction {
		return false
	}

	funcData := funcType.Data.(*FunctionType)
	if len(funcData.Parameters) == 0 {
		return false
	}

	// Unify argument type with first parameter type.
	err := cs.inference.Unify(argType, funcData.Parameters[0])
	if err != nil {
		return false
	}

	constraints.MarkSolved(constraint)

	return true
}

// isSubtype checks if one type is a subtype of another.
func (cs *ConstraintSolver) isSubtype(subtype, supertype *Type) bool {
	// Apply substitutions first.
	subtype = cs.inference.ApplySubstitutions(subtype)
	supertype = cs.inference.ApplySubstitutions(supertype)

	// Reflexivity: every type is a subtype of itself.
	if subtype.Equals(supertype) {
		return true
	}

	// Type variables can be subtypes of anything (for now).
	if subtype.Kind == TypeKindTypeVar || supertype.Kind == TypeKindTypeVar {
		return true
	}

	// Function subtyping (contravariant in parameters, covariant in return type).
	if subtype.Kind == TypeKindFunction && supertype.Kind == TypeKindFunction {
		subFunc := subtype.Data.(*FunctionType)
		superFunc := supertype.Data.(*FunctionType)

		// Check parameter count.
		if len(subFunc.Parameters) != len(superFunc.Parameters) {
			return false
		}

		// Check parameter types (contravariant).
		for i, subParam := range subFunc.Parameters {
			superParam := superFunc.Parameters[i]
			if !cs.isSubtype(superParam, subParam) { // Note: reversed order
				return false
			}
		}

		// Check return type (covariant).
		return cs.isSubtype(subFunc.ReturnType, superFunc.ReturnType)
	}

	// Default: no subtype relationship.
	return false
}

// reportUnsolvableConstraints generates an error report for unsolvable constraints.
func (cs *ConstraintSolver) reportUnsolvableConstraints(constraints *ConstraintSet) error {
	var errorMsg strings.Builder

	errorMsg.WriteString("Unable to solve the following constraints:\n")

	for i, constraint := range constraints.GetConstraints() {
		errorMsg.WriteString(fmt.Sprintf("  %d: %s", i+1, constraint.String()))

		if constraint.Description != "" {
			errorMsg.WriteString(fmt.Sprintf(" (%s)", constraint.Description))
		}

		if constraint.Location.File != "" {
			errorMsg.WriteString(fmt.Sprintf(" at %s:%d:%d",
				constraint.Location.File, constraint.Location.Line, constraint.Location.Column))
		}

		errorMsg.WriteString("\n")
	}

	return errors.New(errorMsg.String())
}

// ConstraintBasedInference combines constraint generation and solving.
type ConstraintBasedInference struct {
	solver    *ConstraintSolver
	inference *InferenceEngine
}

// NewConstraintBasedInference creates a new constraint-based inference system.
func NewConstraintBasedInference(inference *InferenceEngine) *ConstraintBasedInference {
	solver := NewConstraintSolver(inference)

	return &ConstraintBasedInference{
		solver:    solver,
		inference: inference,
	}
}

// InferTypeWithConstraints performs type inference using constraint-based approach.
func (cbi *ConstraintBasedInference) InferTypeWithConstraints(expr Expr) (*Type, error) {
	// For now, use simple constraint generation based on expression type.
	constraints := NewConstraintSet()

	// Generate basic constraints for the expression.
	resultType, err := cbi.generateBasicConstraints(expr, constraints)
	if err != nil {
		return nil, fmt.Errorf("constraint generation failed: %w", err)
	}

	// Solve the constraints.
	err = cbi.solver.SolveConstraints(constraints)
	if err != nil {
		return nil, fmt.Errorf("constraint solving failed: %w", err)
	}

	// Apply final substitutions to the result type.
	finalType := cbi.inference.ApplySubstitutions(resultType)

	return finalType, nil
}

// generateBasicConstraints generates basic constraints for an expression.
func (cbi *ConstraintBasedInference) generateBasicConstraints(expr Expr, constraints *ConstraintSet) (*Type, error) {
	switch e := expr.(type) {
	case *LiteralExpr:
		// Literals have concrete types, no constraints needed.
		switch e.Value.(type) {
		case int32:
			return TypeInt32, nil
		case int64:
			return TypeInt64, nil
		case float32:
			return TypeFloat32, nil
		case float64:
			return TypeFloat64, nil
		case string:
			return TypeString, nil
		case bool:
			return TypeBool, nil
		default:
			return nil, fmt.Errorf("unsupported literal type: %T", e.Value)
		}
	case *VariableExpr:
		// Look up variable in environment.
		scheme, exists := cbi.inference.LookupVariable(e.Name)
		if !exists {
			return nil, fmt.Errorf("undefined variable: %s", e.Name)
		}

		return cbi.inference.Instantiate(scheme), nil
	case *ApplicationExpr:
		// Generate constraints for function application.
		funcType, err := cbi.generateBasicConstraints(e.Function, constraints)
		if err != nil {
			return nil, err
		}

		argType, err := cbi.generateBasicConstraints(e.Argument, constraints)
		if err != nil {
			return nil, err
		}

		// Create fresh type variable for result.
		resultType := cbi.inference.FreshTypeVar()

		// Create expected function type: argType -> resultType.
		expectedFuncType := NewFunctionType([]*Type{argType}, resultType, false, false)

		// Add unification constraint: funcType = expectedFuncType.
		constraints.AddUnificationConstraint(
			funcType,
			expectedFuncType,
			SourceLocation{},
			fmt.Sprintf("function application: %s applied to %s", funcType.String(), argType.String()),
		)

		return resultType, nil
	default:
		// For other expressions, fall back to regular HM inference.
		return cbi.inference.InferType(expr)
	}
}

// SetVerbose enables or disables verbose output.
func (cbi *ConstraintBasedInference) SetVerbose(verbose bool) {
	cbi.solver.SetVerbose(verbose)
}
