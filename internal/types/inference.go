// Hindley-Milner type inference engine for Orizon language
// Implements Algorithm W with unification and type variable management

package types

import (
	"fmt"
	"strings"
)

// ====== Type Inference Engine ======

// InferenceEngine manages type inference state and operations
type InferenceEngine struct {
	// Type variable generation
	nextTypeVarId int
	typeVarPrefix string

	// Unification state
	substitutions map[string]*Type
	constraints   []Constraint

	// Type environments
	globalEnv  *TypeEnvironment
	currentEnv *TypeEnvironment

	// Error handling
	errors []InferenceError

	// Configuration
	config InferenceConfig
}

// InferenceConfig controls inference behavior
type InferenceConfig struct {
	EnableLetPolymorphism bool
	MaxUnificationDepth   int
	EnableOccursCheck     bool
	EnableTypeClasses     bool
	VerboseMode           bool
}

// TypeEnvironment represents a type environment (Γ)
type TypeEnvironment struct {
	Variables map[string]*TypeScheme
	Parent    *TypeEnvironment
	Level     int // For let-polymorphism
}

// TypeScheme represents a polymorphic type scheme (∀α₁...αₙ.τ)
type TypeScheme struct {
	TypeVars []string // Quantified type variables
	Type     *Type    // The type
	Level    int      // Generalization level
}

// InferenceError represents type inference errors
type InferenceError struct {
	Message  string
	Location SourceLocation
	Context  string
	Hint     string
}

// ====== Constraint System ======

// ConstraintKind was already defined in functions.go - reusing it
// Additional constraint kinds for HM inference
const (
	ConstraintInstance ConstraintKind = iota + 100 // τ₁ ◁ σ (instance)
	ConstraintExplicit                             // Explicit type annotation
)

// ====== Type Variable Management ======

// NewInferenceEngine creates a new type inference engine
func NewInferenceEngine() *InferenceEngine {
	config := InferenceConfig{
		EnableLetPolymorphism: true,
		MaxUnificationDepth:   1000,
		EnableOccursCheck:     true,
		EnableTypeClasses:     false, // Will be implemented in Phase 2.2.2
		VerboseMode:           false,
	}

	globalEnv := &TypeEnvironment{
		Variables: make(map[string]*TypeScheme),
		Parent:    nil,
		Level:     0,
	}

	engine := &InferenceEngine{
		nextTypeVarId: 0,
		typeVarPrefix: "τ",
		substitutions: make(map[string]*Type),
		constraints:   []Constraint{},
		globalEnv:     globalEnv,
		currentEnv:    globalEnv,
		errors:        []InferenceError{},
		config:        config,
	}

	// Add built-in types to global environment
	engine.initializeBuiltins()

	return engine
}

// initializeBuiltins adds built-in functions and types to the global environment
func (ie *InferenceEngine) initializeBuiltins() {
	// Add built-in arithmetic operations
	binaryIntOp := &TypeScheme{
		TypeVars: []string{},
		Type:     NewFunctionType([]*Type{TypeInt32, TypeInt32}, TypeInt32, false, false),
		Level:    0,
	}

	ie.globalEnv.Variables["+"] = binaryIntOp
	ie.globalEnv.Variables["-"] = binaryIntOp
	ie.globalEnv.Variables["*"] = binaryIntOp
	ie.globalEnv.Variables["/"] = binaryIntOp

	// Add comparison operations
	comparisonOp := &TypeScheme{
		TypeVars: []string{"a"},
		Type: NewFunctionType([]*Type{
			NewGenericType("a", []*Type{}, VarianceInvariant),
			NewGenericType("a", []*Type{}, VarianceInvariant),
		}, TypeBool, false, false),
		Level: 0,
	}

	ie.globalEnv.Variables["=="] = comparisonOp
	ie.globalEnv.Variables["!="] = comparisonOp
	ie.globalEnv.Variables["<"] = comparisonOp
	ie.globalEnv.Variables[">"] = comparisonOp

	// Add list operations (polymorphic)
	// head :: [a] -> a
	listHeadType := &TypeScheme{
		TypeVars: []string{"a"},
		Type: NewFunctionType([]*Type{
			NewSliceType(NewGenericType("a", []*Type{}, VarianceInvariant)),
		}, NewGenericType("a", []*Type{}, VarianceInvariant), false, false),
		Level: 0,
	}
	ie.globalEnv.Variables["head"] = listHeadType

	// tail :: [a] -> [a]
	listTailType := &TypeScheme{
		TypeVars: []string{"a"},
		Type: NewFunctionType([]*Type{
			NewSliceType(NewGenericType("a", []*Type{}, VarianceInvariant)),
		}, NewSliceType(NewGenericType("a", []*Type{}, VarianceInvariant)), false, false),
		Level: 0,
	}
	ie.globalEnv.Variables["tail"] = listTailType
}

// FreshTypeVar generates a fresh type variable
func (ie *InferenceEngine) FreshTypeVar() *Type {
	varName := fmt.Sprintf("%s%d", ie.typeVarPrefix, ie.nextTypeVarId)
	id := ie.nextTypeVarId
	ie.nextTypeVarId++

	return NewTypeVar(id, varName, []*Type{})
}

// FreshTypeVars generates multiple fresh type variables
func (ie *InferenceEngine) FreshTypeVars(count int) []*Type {
	vars := make([]*Type, count)
	for i := 0; i < count; i++ {
		vars[i] = ie.FreshTypeVar()
	}
	return vars
}

// ====== Type Environment Operations ======

// PushEnvironment creates a new nested environment
func (ie *InferenceEngine) PushEnvironment() {
	newEnv := &TypeEnvironment{
		Variables: make(map[string]*TypeScheme),
		Parent:    ie.currentEnv,
		Level:     ie.currentEnv.Level + 1,
	}
	ie.currentEnv = newEnv
}

// PopEnvironment returns to parent environment
func (ie *InferenceEngine) PopEnvironment() {
	if ie.currentEnv.Parent != nil {
		ie.currentEnv = ie.currentEnv.Parent
	}
}

// LookupVariable looks up a variable in the current environment
func (ie *InferenceEngine) LookupVariable(name string) (*TypeScheme, bool) {
	env := ie.currentEnv
	for env != nil {
		if scheme, exists := env.Variables[name]; exists {
			return scheme, true
		}
		env = env.Parent
	}
	return nil, false
}

// AddVariable adds a variable to the current environment
func (ie *InferenceEngine) AddVariable(name string, scheme *TypeScheme) {
	ie.currentEnv.Variables[name] = scheme
}

// ====== Unification Algorithm ======

// Unify implements the unification algorithm (Algorithm U)
func (ie *InferenceEngine) Unify(t1, t2 *Type) error {
	// Apply current substitutions
	t1 = ie.ApplySubstitutions(t1)
	t2 = ie.ApplySubstitutions(t2)

	// Same type reference
	if t1 == t2 {
		return nil
	}

	// Type variable cases
	if t1.Kind == TypeKindTypeVar {
		return ie.unifyTypeVar(t1, t2)
	}
	if t2.Kind == TypeKindTypeVar {
		return ie.unifyTypeVar(t2, t1)
	}

	// Different kinds cannot unify
	if t1.Kind != t2.Kind {
		return fmt.Errorf("cannot unify %s with %s: incompatible kinds", t1.String(), t2.String())
	}

	// Unify based on type structure
	return ie.unifyStructural(t1, t2)
}

// unifyTypeVar unifies a type variable with another type
func (ie *InferenceEngine) unifyTypeVar(typeVar, other *Type) error {
	varData := typeVar.Data.(*TypeVar)
	varName := varData.Name

	// Already has substitution
	if subst, exists := ie.substitutions[varName]; exists {
		return ie.Unify(subst, other)
	}

	// Occurs check - prevent infinite types
	if ie.config.EnableOccursCheck && ie.occursCheck(varName, other) {
		return fmt.Errorf("occurs check failed: %s occurs in %s", varName, other.String())
	}

	// Add substitution
	ie.substitutions[varName] = other
	return nil
}

// unifyStructural unifies types based on their structure
func (ie *InferenceEngine) unifyStructural(t1, t2 *Type) error {
	switch t1.Kind {
	case TypeKindArray:
		return ie.unifyArrayTypes(t1, t2)
	case TypeKindSlice:
		return ie.unifySliceTypes(t1, t2)
	case TypeKindFunction:
		return ie.unifyFunctionTypes(t1, t2)
	case TypeKindStruct:
		return ie.unifyStructTypes(t1, t2)
	case TypeKindPointer:
		return ie.unifyPointerTypes(t1, t2)
	default:
		// Primitive types unify if they're the same kind
		if t1.Kind == t2.Kind {
			return nil
		}
		return fmt.Errorf("cannot unify %s with %s", t1.String(), t2.String())
	}
}

// unifyArrayTypes unifies array types
func (ie *InferenceEngine) unifyArrayTypes(t1, t2 *Type) error {
	arr1 := t1.Data.(*ArrayType)
	arr2 := t2.Data.(*ArrayType)

	if arr1.Length != arr2.Length {
		return fmt.Errorf("array length mismatch: %d vs %d", arr1.Length, arr2.Length)
	}

	return ie.Unify(arr1.ElementType, arr2.ElementType)
}

// unifySliceTypes unifies slice types
func (ie *InferenceEngine) unifySliceTypes(t1, t2 *Type) error {
	slice1 := t1.Data.(*SliceType)
	slice2 := t2.Data.(*SliceType)

	return ie.Unify(slice1.ElementType, slice2.ElementType)
}

// unifyFunctionTypes unifies function types
func (ie *InferenceEngine) unifyFunctionTypes(t1, t2 *Type) error {
	func1 := getFunctionTypeData(t1)
	func2 := getFunctionTypeData(t2)

	if func1 == nil || func2 == nil {
		return fmt.Errorf("invalid function types for unification")
	}

	// Parameter count must match
	if len(func1.Parameters) != len(func2.Parameters) {
		return fmt.Errorf("function parameter count mismatch: %d vs %d",
			len(func1.Parameters), len(func2.Parameters))
	}

	// Unify parameters
	for i, param1 := range func1.Parameters {
		param2 := func2.Parameters[i]
		if err := ie.Unify(param1, param2); err != nil {
			return fmt.Errorf("parameter %d unification failed: %v", i, err)
		}
	}

	// Unify return types
	return ie.Unify(func1.ReturnType, func2.ReturnType)
}

// unifyStructTypes unifies struct types
func (ie *InferenceEngine) unifyStructTypes(t1, t2 *Type) error {
	struct1 := t1.Data.(*StructType)
	struct2 := t2.Data.(*StructType)

	if len(struct1.Fields) != len(struct2.Fields) {
		return fmt.Errorf("struct field count mismatch: %d vs %d",
			len(struct1.Fields), len(struct2.Fields))
	}

	// Unify fields by name
	for _, field1 := range struct1.Fields {
		found := false
		for _, field2 := range struct2.Fields {
			if field1.Name == field2.Name {
				if err := ie.Unify(field1.Type, field2.Type); err != nil {
					return fmt.Errorf("field %s unification failed: %v", field1.Name, err)
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("field %s not found in second struct", field1.Name)
		}
	}

	return nil
}

// unifyPointerTypes unifies pointer types
func (ie *InferenceEngine) unifyPointerTypes(t1, t2 *Type) error {
	ptr1 := t1.Data.(*PointerType)
	ptr2 := t2.Data.(*PointerType)

	return ie.Unify(ptr1.PointeeType, ptr2.PointeeType)
}

// occursCheck implements the occurs check to prevent infinite types
func (ie *InferenceEngine) occursCheck(varName string, t *Type) bool {
	switch t.Kind {
	case TypeKindTypeVar:
		varData := t.Data.(*TypeVar)
		return varName == varData.Name

	case TypeKindArray:
		arrayData := t.Data.(*ArrayType)
		return ie.occursCheck(varName, arrayData.ElementType)

	case TypeKindSlice:
		sliceData := t.Data.(*SliceType)
		return ie.occursCheck(varName, sliceData.ElementType)

	case TypeKindFunction:
		funcData := getFunctionTypeData(t)
		if funcData == nil {
			return false
		}

		// Check parameters
		for _, param := range funcData.Parameters {
			if ie.occursCheck(varName, param) {
				return true
			}
		}

		// Check return type
		return ie.occursCheck(varName, funcData.ReturnType)

	case TypeKindPointer:
		ptrData := t.Data.(*PointerType)
		return ie.occursCheck(varName, ptrData.PointeeType)

	default:
		return false
	}
}

// ====== Substitution Operations ======

// ApplySubstitutions applies current substitutions to a type
func (ie *InferenceEngine) ApplySubstitutions(t *Type) *Type {
	switch t.Kind {
	case TypeKindTypeVar:
		varData := t.Data.(*TypeVar)
		if subst, exists := ie.substitutions[varData.Name]; exists {
			// Recursively apply substitutions
			return ie.ApplySubstitutions(subst)
		}
		return t

	case TypeKindArray:
		arrayData := t.Data.(*ArrayType)
		newElementType := ie.ApplySubstitutions(arrayData.ElementType)
		if newElementType != arrayData.ElementType {
			return NewArrayType(newElementType, arrayData.Length)
		}
		return t

	case TypeKindSlice:
		sliceData := t.Data.(*SliceType)
		newElementType := ie.ApplySubstitutions(sliceData.ElementType)
		if newElementType != sliceData.ElementType {
			return NewSliceType(newElementType)
		}
		return t

	case TypeKindFunction:
		funcData := getFunctionTypeData(t)
		if funcData == nil {
			return t
		}

		changed := false
		newParams := make([]*Type, len(funcData.Parameters))
		for i, param := range funcData.Parameters {
			newParam := ie.ApplySubstitutions(param)
			newParams[i] = newParam
			if newParam != param {
				changed = true
			}
		}

		newReturnType := ie.ApplySubstitutions(funcData.ReturnType)
		if newReturnType != funcData.ReturnType {
			changed = true
		}

		if changed {
			return NewFunctionType(newParams, newReturnType, funcData.IsVariadic, funcData.IsAsync)
		}
		return t

	case TypeKindPointer:
		ptrData := t.Data.(*PointerType)
		newElementType := ie.ApplySubstitutions(ptrData.PointeeType)
		if newElementType != ptrData.PointeeType {
			return NewPointerType(newElementType, ptrData.IsNullable)
		}
		return t

	case TypeKindGeneric:
		genericData := t.Data.(*GenericType)
		if subst, exists := ie.substitutions[genericData.Name]; exists {
			// Recursively apply substitutions
			return ie.ApplySubstitutions(subst)
		}
		// If no substitution exists, treat as unbound generic type
		return t

	default:
		return t
	}
}

// ComposeSubstitutions composes two substitution sets
func (ie *InferenceEngine) ComposeSubstitutions(s1, s2 map[string]*Type) map[string]*Type {
	result := make(map[string]*Type)

	// Apply s2 to s1
	for var1, type1 := range s1 {
		result[var1] = ie.applySubstitutionMap(s2, type1)
	}

	// Add s2 (avoiding conflicts)
	for var2, type2 := range s2 {
		if _, exists := result[var2]; !exists {
			result[var2] = type2
		}
	}

	return result
}

// applySubstitutionMap applies a substitution map to a type
func (ie *InferenceEngine) applySubstitutionMap(substMap map[string]*Type, t *Type) *Type {
	oldSubsts := ie.substitutions
	ie.substitutions = substMap
	result := ie.ApplySubstitutions(t)
	ie.substitutions = oldSubsts
	return result
}

// ====== Type Generalization (Let-polymorphism) ======

// Generalize creates a type scheme by quantifying over free type variables
func (ie *InferenceEngine) Generalize(t *Type) *TypeScheme {
	if !ie.config.EnableLetPolymorphism {
		return &TypeScheme{
			TypeVars: []string{},
			Type:     t,
			Level:    ie.currentEnv.Level,
		}
	}

	// Apply current substitutions
	t = ie.ApplySubstitutions(t)

	// Find free type variables not in the environment
	freeVars := ie.findFreeTypeVars(t)
	envVars := ie.getEnvironmentTypeVars()

	var quantifiedVars []string
	for _, freeVar := range freeVars {
		if !ie.containsString(envVars, freeVar) {
			quantifiedVars = append(quantifiedVars, freeVar)
		}
	}

	return &TypeScheme{
		TypeVars: quantifiedVars,
		Type:     t,
		Level:    ie.currentEnv.Level,
	}
}

// Instantiate creates a fresh instance of a type scheme
func (ie *InferenceEngine) Instantiate(scheme *TypeScheme) *Type {
	if len(scheme.TypeVars) == 0 {
		return scheme.Type
	}

	// Create fresh type variables for quantified variables
	substitutions := make(map[string]*Type)
	for _, typeVar := range scheme.TypeVars {
		substitutions[typeVar] = ie.FreshTypeVar()
	}

	// Apply substitutions to the type
	oldSubsts := ie.substitutions
	ie.substitutions = ie.ComposeSubstitutions(substitutions, ie.substitutions)
	result := ie.ApplySubstitutions(scheme.Type)
	ie.substitutions = oldSubsts

	return result
}

// findFreeTypeVars finds all free type variables in a type
func (ie *InferenceEngine) findFreeTypeVars(t *Type) []string {
	seen := make(map[string]bool)
	var vars []string
	ie.collectFreeTypeVars(t, seen, &vars)
	return vars
}

// collectFreeTypeVars recursively collects free type variables
func (ie *InferenceEngine) collectFreeTypeVars(t *Type, seen map[string]bool, vars *[]string) {
	switch t.Kind {
	case TypeKindTypeVar:
		varData := t.Data.(*TypeVar)
		if !seen[varData.Name] {
			seen[varData.Name] = true
			*vars = append(*vars, varData.Name)
		}

	case TypeKindArray:
		arrayData := t.Data.(*ArrayType)
		ie.collectFreeTypeVars(arrayData.ElementType, seen, vars)

	case TypeKindSlice:
		sliceData := t.Data.(*SliceType)
		ie.collectFreeTypeVars(sliceData.ElementType, seen, vars)

	case TypeKindFunction:
		funcData := getFunctionTypeData(t)
		if funcData != nil {
			for _, param := range funcData.Parameters {
				ie.collectFreeTypeVars(param, seen, vars)
			}
			ie.collectFreeTypeVars(funcData.ReturnType, seen, vars)
		}

	case TypeKindPointer:
		ptrData := t.Data.(*PointerType)
		ie.collectFreeTypeVars(ptrData.PointeeType, seen, vars)
	}
}

// getEnvironmentTypeVars gets all type variables in the current environment
func (ie *InferenceEngine) getEnvironmentTypeVars() []string {
	var vars []string
	env := ie.currentEnv

	for env != nil {
		for _, scheme := range env.Variables {
			freeVars := ie.findFreeTypeVars(scheme.Type)
			for _, freeVar := range freeVars {
				if !ie.containsString(vars, freeVar) {
					vars = append(vars, freeVar)
				}
			}
		}
		env = env.Parent
	}

	return vars
}

// containsString checks if a slice contains a string
func (ie *InferenceEngine) containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ====== Error Handling ======

// AddError adds an inference error
func (ie *InferenceEngine) AddError(message string, location SourceLocation, context string) {
	ie.errors = append(ie.errors, InferenceError{
		Message:  message,
		Location: location,
		Context:  context,
		Hint:     "",
	})
}

// GetErrors returns all inference errors
func (ie *InferenceEngine) GetErrors() []InferenceError {
	return ie.errors
}

// ClearErrors clears all inference errors
func (ie *InferenceEngine) ClearErrors() {
	ie.errors = []InferenceError{}
}

// ====== String Representations ======

// String representation for TypeScheme
func (ts *TypeScheme) String() string {
	if len(ts.TypeVars) == 0 {
		return ts.Type.String()
	}

	return fmt.Sprintf("∀%s.%s", strings.Join(ts.TypeVars, ","), ts.Type.String())
}

// String representation for TypeEnvironment
func (te *TypeEnvironment) String() string {
	var entries []string
	for name, scheme := range te.Variables {
		entries = append(entries, fmt.Sprintf("%s: %s", name, scheme.String()))
	}
	return fmt.Sprintf("Γ{%s}", strings.Join(entries, ", "))
}

// String representation for InferenceError
func (ie *InferenceError) String() string {
	if ie.Hint != "" {
		return fmt.Sprintf("%s:%d:%d: %s (%s). Hint: %s",
			ie.Location.File, ie.Location.Line, ie.Location.Column,
			ie.Message, ie.Context, ie.Hint)
	}
	return fmt.Sprintf("%s:%d:%d: %s (%s)",
		ie.Location.File, ie.Location.Line, ie.Location.Column,
		ie.Message, ie.Context)
}
