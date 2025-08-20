// Hindley-Milner Type Inference Engine for the Orizon programming language.
// This file implements Algorithm W and related type inference functionality.

package hir

import (
	"fmt"
	"strings"
)

// =============================================================================.
// Type Inference Core Structures.
// =============================================================================.

// TypeInferenceEngine represents the main type inference system.
type TypeInferenceEngine struct {
	substitutions  *Substitution
	environment    *TypeEnvironment
	constraints    []TypeConstraint
	errors         []TypeInferenceError
	typeVarCounter int
}

// TypeVariable represents a type variable in the inference system.
type TypeVariable struct {
	Name  string
	ID    int
	Kind  TypeKind
	Level int
	Bound bool
}

// TypeScheme represents a polymorphic type scheme (∀α.τ).
type TypeScheme struct {
	Type      TypeInfo
	Variables []TypeVariable
}

// TypeEnvironment represents the typing environment (Γ).
type TypeEnvironment struct {
	parent   *TypeEnvironment      // Parent environment for scoping
	bindings map[string]TypeScheme // Variable name to type scheme mappings
	level    int                   // Current quantification level
}

// Substitution represents a substitution map from type variables to types.
type Substitution struct {
	mappings map[int]TypeInfo // Type variable ID to type mappings
}

// TypeInferenceError represents an error during type inference.
type TypeInferenceError struct {
	Expected *TypeInfo
	Actual   *TypeInfo
	Message  string
	Location string
	Context  string
	Kind     InferenceErrorKind
}

// InferenceErrorKind represents different kinds of inference errors.
type InferenceErrorKind int

const (
	ErrorUnificationFailure InferenceErrorKind = iota
	ErrorOccursCheck
	ErrorUndefinedVariable
	ErrorTypeMismatch
	ErrorInfiniteType
	ErrorKindMismatch
	ErrorConstraintViolation
)

// =============================================================================.
// Type Inference Engine Implementation.
// =============================================================================.

// NewTypeInferenceEngine creates a new type inference engine.
func NewTypeInferenceEngine() *TypeInferenceEngine {
	return &TypeInferenceEngine{
		typeVarCounter: 0,
		constraints:    make([]TypeConstraint, 0),
		substitutions:  NewSubstitution(),
		environment:    NewTypeEnvironment(),
		errors:         make([]TypeInferenceError, 0),
	}
}

// InferType performs type inference on an HIR expression using Algorithm W.
func (engine *TypeInferenceEngine) InferType(expr HIRExpression) (TypeInfo, error) {
	// Reset engine state for new inference.
	engine.clearState()

	// Perform type inference.
	inferredType, err := engine.inferExpression(expr)
	if err != nil {
		return TypeInfo{}, err
	}

	// Apply final substitutions.
	finalType := engine.substitutions.Apply(inferredType)

	// Check for any remaining errors.
	if len(engine.errors) > 0 {
		return TypeInfo{}, fmt.Errorf("type inference failed: %s", engine.formatErrors())
	}

	return finalType, nil
}

// inferExpression implements the core type inference algorithm for expressions.
func (engine *TypeInferenceEngine) inferExpression(expr HIRExpression) (TypeInfo, error) {
	switch e := expr.(type) {
	case *HIRLiteral:
		return engine.inferLiteral(e)
	default:
		// For now, return the expression's existing type.
		// This provides basic compatibility while the full inference is being built.
		return e.GetType(), nil
	}
}

// inferLiteral infers the type of literal expressions.
func (engine *TypeInferenceEngine) inferLiteral(lit *HIRLiteral) (TypeInfo, error) {
	// Literals have known concrete types.
	return lit.GetType(), nil
}

// freshTypeVariable creates a new unique type variable.
func (engine *TypeInferenceEngine) freshTypeVariable() TypeInfo {
	engine.typeVarCounter++

	return TypeInfo{
		Kind:       TypeKindVariable,
		Name:       fmt.Sprintf("'t%d", engine.typeVarCounter),
		Parameters: []TypeInfo{},
		VariableID: &engine.typeVarCounter,
	}
}

// instantiate creates a fresh instance of a type scheme.
func (engine *TypeInferenceEngine) instantiate(scheme TypeScheme) TypeInfo {
	if len(scheme.Variables) == 0 {
		return scheme.Type
	}

	// Create substitution map for quantified variables.
	substitution := NewSubstitution()

	for _, variable := range scheme.Variables {
		freshVar := engine.freshTypeVariable()
		substitution.Add(variable.ID, freshVar)
	}

	// Apply substitution to the type.
	return substitution.Apply(scheme.Type)
}

// generalize creates a type scheme by quantifying free type variables.
func (engine *TypeInferenceEngine) generalize(typeInfo TypeInfo) TypeScheme {
	freeVars := engine.getFreeVariables(typeInfo)
	envVars := engine.environment.getFreeVariables()

	// Quantify variables that are free in the type but not in the environment.
	var quantifiedVars []TypeVariable

	for _, variable := range freeVars {
		if !contains(envVars, variable) {
			quantifiedVars = append(quantifiedVars, variable)
		}
	}

	return TypeScheme{
		Variables: quantifiedVars,
		Type:      typeInfo,
	}
}

// getFreeVariables extracts free type variables from a type.
func (engine *TypeInferenceEngine) getFreeVariables(typeInfo TypeInfo) []TypeVariable {
	var variables []TypeVariable

	switch typeInfo.Kind {
	case TypeKindVariable:
		if typeInfo.VariableID != nil {
			variables = append(variables, TypeVariable{
				ID:   *typeInfo.VariableID,
				Name: typeInfo.Name,
			})
		}
	case TypeKindFunction, TypeKindTuple:
		for _, param := range typeInfo.Parameters {
			variables = append(variables, engine.getFreeVariables(param)...)
		}
	case TypeKindStruct:
		if typeInfo.StructInfo != nil {
			for _, field := range typeInfo.StructInfo.Fields {
				// Note: FieldLayout doesn't have Type field, using Fields from TypeInfo instead.
				for _, fieldInfo := range typeInfo.Fields {
					if fieldInfo.Name == field.Name {
						variables = append(variables, engine.getFreeVariables(fieldInfo.Type)...)

						break
					}
				}
			}
		}
	}

	return removeDuplicateVariables(variables)
}

// =============================================================================.
// Unification Algorithm Implementation.
// =============================================================================.

// unify implements the unification algorithm for type checking.
func (engine *TypeInferenceEngine) unify(type1, type2 TypeInfo) error {
	// Apply current substitutions.
	t1 := engine.substitutions.Apply(type1)
	t2 := engine.substitutions.Apply(type2)

	// Check if types are already equal.
	if t1.Equals(t2) {
		return nil
	}

	// Handle type variables.
	if t1.Kind == TypeKindVariable {
		return engine.unifyVariable(t1, t2)
	}

	if t2.Kind == TypeKindVariable {
		return engine.unifyVariable(t2, t1)
	}

	// Check kind compatibility.
	if t1.Kind != t2.Kind {
		return fmt.Errorf("cannot unify types of different kinds: %s and %s", t1.Name, t2.Name)
	}

	// Unify composite types.
	switch t1.Kind {
	case TypeKindFunction:
		return engine.unifyFunction(t1, t2)
	case TypeKindTuple:
		return engine.unifyTuple(t1, t2)
	case TypeKindStruct:
		return engine.unifyStruct(t1, t2)
	default:
		// For primitive types, they should already be equal if unifiable.
		return fmt.Errorf("cannot unify incompatible types: %s and %s", t1.Name, t2.Name)
	}
}

// unifyVariable unifies a type variable with another type.
func (engine *TypeInferenceEngine) unifyVariable(variable, other TypeInfo) error {
	if variable.VariableID == nil {
		return fmt.Errorf("invalid type variable: missing ID")
	}

	// Occurs check: prevent infinite types.
	if engine.occursCheck(*variable.VariableID, other) {
		return fmt.Errorf("occurs check failed: infinite type")
	}

	// Add substitution.
	engine.substitutions.Add(*variable.VariableID, other)

	return nil
}

// unifyFunction unifies two function types.
func (engine *TypeInferenceEngine) unifyFunction(func1, func2 TypeInfo) error {
	if len(func1.Parameters) != len(func2.Parameters) {
		return fmt.Errorf("function arity mismatch: %d vs %d",
			len(func1.Parameters), len(func2.Parameters))
	}

	// Unify all parameter types.
	for i := 0; i < len(func1.Parameters); i++ {
		err := engine.unify(func1.Parameters[i], func2.Parameters[i])
		if err != nil {
			return fmt.Errorf("parameter %d unification failed: %w", i, err)
		}
	}

	return nil
}

// unifyTuple unifies two tuple types.
func (engine *TypeInferenceEngine) unifyTuple(tuple1, tuple2 TypeInfo) error {
	if len(tuple1.Parameters) != len(tuple2.Parameters) {
		return fmt.Errorf("tuple arity mismatch: %d vs %d",
			len(tuple1.Parameters), len(tuple2.Parameters))
	}

	// Unify all element types.
	for i := 0; i < len(tuple1.Parameters); i++ {
		err := engine.unify(tuple1.Parameters[i], tuple2.Parameters[i])
		if err != nil {
			return fmt.Errorf("tuple element %d unification failed: %w", i, err)
		}
	}

	return nil
}

// unifyStruct unifies two struct types.
func (engine *TypeInferenceEngine) unifyStruct(struct1, struct2 TypeInfo) error {
	// Check field count using TypeInfo.Fields instead of StructInfo.Fields
	if len(struct1.Fields) != len(struct2.Fields) {
		return fmt.Errorf("struct field count mismatch")
	}

	// Unify corresponding fields using FieldInfo.
	for i, field1 := range struct1.Fields {
		field2 := struct2.Fields[i]

		// Field names must match.
		if field1.Name != field2.Name {
			return fmt.Errorf("struct field name mismatch: %s vs %s", field1.Name, field2.Name)
		}

		// Unify field types.
		err := engine.unify(field1.Type, field2.Type)
		if err != nil {
			return fmt.Errorf("struct field %s unification failed: %w", field1.Name, err)
		}
	}

	return nil
} // occursCheck checks if a type variable occurs in a type (prevents infinite types)
func (engine *TypeInferenceEngine) occursCheck(varID int, typeInfo TypeInfo) bool {
	switch typeInfo.Kind {
	case TypeKindVariable:
		return typeInfo.VariableID != nil && *typeInfo.VariableID == varID
	case TypeKindFunction, TypeKindTuple:
		for _, param := range typeInfo.Parameters {
			if engine.occursCheck(varID, param) {
				return true
			}
		}
	case TypeKindStruct:
		// Check in FieldInfo instead of FieldLayout.
		for _, field := range typeInfo.Fields {
			if engine.occursCheck(varID, field.Type) {
				return true
			}
		}
	}

	return false
}

// =============================================================================.
// Type Environment Implementation.
// =============================================================================.

// NewTypeEnvironment creates a new empty type environment.
func NewTypeEnvironment() *TypeEnvironment {
	return &TypeEnvironment{
		parent:   nil,
		bindings: make(map[string]TypeScheme),
		level:    0,
	}
}

// Extend creates a new environment that extends this one.
func (env *TypeEnvironment) Extend() *TypeEnvironment {
	return &TypeEnvironment{
		parent:   env,
		bindings: make(map[string]TypeScheme),
		level:    env.level + 1,
	}
}

// Copy creates a shallow copy of the environment.
func (env *TypeEnvironment) Copy() *TypeEnvironment {
	newBindings := make(map[string]TypeScheme)
	for name, scheme := range env.bindings {
		newBindings[name] = scheme
	}

	return &TypeEnvironment{
		parent:   env.parent,
		bindings: newBindings,
		level:    env.level,
	}
}

// Bind adds a variable binding to the environment.
func (env *TypeEnvironment) Bind(name string, scheme TypeScheme) {
	env.bindings[name] = scheme
}

// Lookup searches for a variable binding in the environment.
func (env *TypeEnvironment) Lookup(name string) (TypeScheme, bool) {
	// Check current environment.
	if scheme, exists := env.bindings[name]; exists {
		return scheme, true
	}

	// Check parent environments.
	if env.parent != nil {
		return env.parent.Lookup(name)
	}

	return TypeScheme{}, false
}

// getFreeVariables gets all free variables in the environment.
func (env *TypeEnvironment) getFreeVariables() []TypeVariable {
	var variables []TypeVariable

	// Collect from current environment.
	for _, scheme := range env.bindings {
		// Get free variables from the type (excluding quantified ones).
		freeInType := getFreeVariablesFromType(scheme.Type)
		for _, freeVar := range freeInType {
			if !isQuantified(freeVar, scheme.Variables) {
				variables = append(variables, freeVar)
			}
		}
	}

	// Collect from parent environments.
	if env.parent != nil {
		parentVars := env.parent.getFreeVariables()
		variables = append(variables, parentVars...)
	}

	return removeDuplicateVariables(variables)
}

// =============================================================================.
// Substitution Implementation.
// =============================================================================.

// NewSubstitution creates a new empty substitution.
func NewSubstitution() *Substitution {
	return &Substitution{
		mappings: make(map[int]TypeInfo),
	}
}

// Add adds a new substitution mapping.
func (subst *Substitution) Add(varID int, typeInfo TypeInfo) {
	subst.mappings[varID] = typeInfo
}

// AddByName adds a substitution mapping using variable name.
func (subst *Substitution) AddByName(varName string, typeInfo TypeInfo) {
	// Generate same ID from name as used elsewhere.
	varID := 0
	for i, char := range varName {
		varID += int(char) + i
	}

	subst.mappings[varID] = typeInfo
}

// Apply applies the substitution to a type.
func (subst *Substitution) Apply(typeInfo TypeInfo) TypeInfo {
	switch typeInfo.Kind {
	case TypeKindVariable:
		varID := 0
		if typeInfo.VariableID != nil {
			varID = *typeInfo.VariableID
		} else {
			// Generate same ID from name as used in constraint solver.
			for i, char := range typeInfo.Name {
				varID += int(char) + i
			}
		}

		if replacement, exists := subst.mappings[varID]; exists {
			// Recursively apply substitution in case of chains.
			return subst.Apply(replacement)
		}

		return typeInfo

	case TypeKindFunction, TypeKindTuple:
		// Apply to all parameters.
		newParams := make([]TypeInfo, len(typeInfo.Parameters))
		for i, param := range typeInfo.Parameters {
			newParams[i] = subst.Apply(param)
		}

		result := typeInfo
		result.Parameters = newParams

		return result

	case TypeKindStruct:
		// Apply to TypeInfo.Fields instead of StructInfo.Fields
		newFields := make([]FieldInfo, len(typeInfo.Fields))
		for i, field := range typeInfo.Fields {
			newFields[i] = FieldInfo{
				Name:    field.Name,
				Type:    subst.Apply(field.Type),
				Offset:  field.Offset,
				Private: field.Private,
				Span:    field.Span,
			}
		}

		result := typeInfo
		result.Fields = newFields

		return result

	default:
		// Primitive types are not affected by substitution.
		return typeInfo
	}
}

// Compose composes this substitution with another.
func (subst *Substitution) Compose(other *Substitution) *Substitution {
	result := NewSubstitution()

	// Apply other substitution to this substitution's mappings.
	for varID, typeInfo := range subst.mappings {
		result.mappings[varID] = other.Apply(typeInfo)
	}

	// Add mappings from other that are not in this substitution.
	for varID, typeInfo := range other.mappings {
		if _, exists := result.mappings[varID]; !exists {
			result.mappings[varID] = typeInfo
		}
	}

	return result
}

// =============================================================================.
// Helper Methods.
// =============================================================================.

// clearState resets the engine state for a new inference session.
func (engine *TypeInferenceEngine) clearState() {
	engine.constraints = engine.constraints[:0]
	engine.substitutions = NewSubstitution()
	engine.errors = engine.errors[:0]
}

// addError adds a type inference error to the error list.
func (engine *TypeInferenceEngine) addError(kind InferenceErrorKind, message, location string,
	expected, actual *TypeInfo, context string,
) {
	engine.errors = append(engine.errors, TypeInferenceError{
		Kind:     kind,
		Message:  message,
		Location: location,
		Expected: expected,
		Actual:   actual,
		Context:  context,
	})
}

// formatErrors formats all collected errors into a readable string.
func (engine *TypeInferenceEngine) formatErrors() string {
	var sb strings.Builder

	for i, err := range engine.errors {
		if i > 0 {
			sb.WriteString("; ")
		}

		sb.WriteString(err.Message)
	}

	return sb.String()
}

// =============================================================================.
// Utility Functions.
// =============================================================================.

// contains checks if a slice contains a specific type variable.
func contains(variables []TypeVariable, target TypeVariable) bool {
	for _, variable := range variables {
		if variable.ID == target.ID {
			return true
		}
	}

	return false
}

// removeDuplicateVariables removes duplicate type variables from a slice.
func removeDuplicateVariables(variables []TypeVariable) []TypeVariable {
	seen := make(map[int]bool)

	var result []TypeVariable

	for _, variable := range variables {
		if !seen[variable.ID] {
			seen[variable.ID] = true

			result = append(result, variable)
		}
	}

	return result
}

// getFreeVariablesFromType extracts free variables from a type.
func getFreeVariablesFromType(typeInfo TypeInfo) []TypeVariable {
	var variables []TypeVariable

	switch typeInfo.Kind {
	case TypeKindVariable:
		if typeInfo.VariableID != nil {
			variables = append(variables, TypeVariable{
				ID:   *typeInfo.VariableID,
				Name: typeInfo.Name,
			})
		}
	case TypeKindFunction, TypeKindTuple:
		for _, param := range typeInfo.Parameters {
			variables = append(variables, getFreeVariablesFromType(param)...)
		}
	case TypeKindStruct:
		// Use TypeInfo.Fields instead of StructInfo.Fields
		for _, field := range typeInfo.Fields {
			variables = append(variables, getFreeVariablesFromType(field.Type)...)
		}
	}

	return variables
}

// isQuantified checks if a variable is quantified in a type scheme.
func isQuantified(variable TypeVariable, quantified []TypeVariable) bool {
	for _, quantVar := range quantified {
		if quantVar.ID == variable.ID {
			return true
		}
	}

	return false
}
