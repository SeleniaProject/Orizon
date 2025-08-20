// Package types implements Phase 2.3.3 Dependent Function Types for the Orizon compiler.
// This system provides Pi types and dependent pattern matching for advanced type-level computation.
package types

import (
	"fmt"
	"strings"
)

// DependentTypeKind represents different kinds of dependent types.
type DependentTypeKind int

const (
	DependentKindPi DependentTypeKind = iota
	DependentKindSigma
	DependentKindInductive
	DependentKindUniverse
)

// Universe represents type universes for stratified type theory.
type Universe struct {
	Level int
}

// String returns a string representation of the universe.
func (u *Universe) String() string {
	if u.Level == 0 {
		return "Type"
	}

	return fmt.Sprintf("Type%d", u.Level)
}

// IsSubUniverseOf checks if this universe is a sub-universe of another.
func (u *Universe) IsSubUniverseOf(other *Universe) bool {
	return u.Level <= other.Level
}

// TypeUniverse0 represents the universe of small types.
var TypeUniverse0 = &Universe{Level: 0}

// TypeUniverse1 represents the universe of type constructors.
var TypeUniverse1 = &Universe{Level: 1}

// PiType represents dependent function types (Π x:A. B(x)).
type PiType struct {
	ParamType  *Type
	ReturnType *Type
	Universe   *Universe
	ParamName  string
	IsImplicit bool
}

// String returns a string representation of the Pi type.
func (pt *PiType) String() string {
	if pt.IsImplicit {
		return fmt.Sprintf("Π{%s:%s}. %s", pt.ParamName, pt.ParamType.String(), pt.ReturnType.String())
	}

	return fmt.Sprintf("Π(%s:%s). %s", pt.ParamName, pt.ParamType.String(), pt.ReturnType.String())
}

// Apply applies the Pi type to an argument, performing substitution.
func (pt *PiType) Apply(arg *Type) (*Type, error) {
	// Substitute the parameter in the return type.
	substituted := pt.substituteInType(pt.ReturnType, pt.ParamName, arg)

	return substituted, nil
}

// substituteInType performs type-level substitution.
func (pt *PiType) substituteInType(target *Type, varName string, replacement *Type) *Type {
	switch target.Kind {
	case TypeKindVariable:
		if varData, ok := target.Data.(*VariableTypeData); ok && varData.Name == varName {
			return replacement
		}

		return target

	case TypeKindFunction:
		if funcData, ok := target.Data.(*FunctionType); ok {
			// Substitute in parameter types.
			newParams := make([]*Type, len(funcData.Parameters))
			for i, param := range funcData.Parameters {
				newParams[i] = pt.substituteInType(param, varName, replacement)
			}

			// Substitute in return type.
			newReturnType := pt.substituteInType(funcData.ReturnType, varName, replacement)

			return &Type{
				Kind: TypeKindFunction,
				Data: &FunctionType{
					Parameters: newParams,
					ReturnType: newReturnType,
					IsVariadic: funcData.IsVariadic,
					IsAsync:    funcData.IsAsync,
				},
			}
		}

		return target

	case TypeKindArray:
		if arrayData, ok := target.Data.(*ArrayType); ok {
			newElementType := pt.substituteInType(arrayData.ElementType, varName, replacement)

			return &Type{
				Kind: TypeKindArray,
				Data: &ArrayType{
					ElementType: newElementType,
					Length:      arrayData.Length,
				},
			}
		}

		return target

	default:
		return target
	}
}

// VariableTypeData represents type variables.
type VariableTypeData struct {
	Name string
}

// String returns a string representation of the variable type.
func (vtd *VariableTypeData) String() string {
	return vtd.Name
}

// SigmaType represents dependent pair types (Σ x:A. B(x)).
type SigmaType struct {
	FirstType  *Type
	SecondType *Type
	Universe   *Universe
	FirstName  string
}

// String returns a string representation of the Sigma type.
func (st *SigmaType) String() string {
	return fmt.Sprintf("Σ(%s:%s). %s", st.FirstName, st.FirstType.String(), st.SecondType.String())
}

// Project extracts a component from a dependent pair.
func (st *SigmaType) Project(pair *Type, component int) (*Type, error) {
	switch component {
	case 0:
		return st.FirstType, nil
	case 1:
		// For the second component, we need the actual first component value.
		// This is simplified - in practice we'd need the actual term.
		return st.SecondType, nil
	default:
		return nil, fmt.Errorf("invalid component index: %d", component)
	}
}

// DependentLambda represents lambda terms in dependent type theory.
type DependentLambda struct {
	Body      DependentTerm
	ParamType *Type
	Type      *PiType
	ParamName string
}

// String returns a string representation of the dependent lambda.
func (dl *DependentLambda) String() string {
	return fmt.Sprintf("λ%s:%s. %s", dl.ParamName, dl.ParamType.String(), dl.Body.String())
}

// GetType returns the type of the dependent lambda.
func (dl *DependentLambda) GetType() *Type {
	return &Type{
		Kind: TypeKindPi,
		Data: dl.Type,
	}
}

// Normalize normalizes the dependent lambda.
func (dl *DependentLambda) Normalize() DependentTerm {
	return dl
}

// Apply applies the lambda to an argument.
func (dl *DependentLambda) Apply(arg DependentTerm) (DependentTerm, error) {
	// Substitute the parameter in the body.
	substituted := dl.substituteInTerm(dl.Body, dl.ParamName, arg)

	return substituted, nil
}

// substituteInTerm performs term-level substitution.
func (dl *DependentLambda) substituteInTerm(target DependentTerm, varName string, replacement DependentTerm) DependentTerm {
	switch term := target.(type) {
	case *DependentVariable:
		if term.Name == varName {
			return replacement
		}

		return term

	case *DependentApplication:
		newFunc := dl.substituteInTerm(term.Function, varName, replacement)
		newArg := dl.substituteInTerm(term.Argument, varName, replacement)

		return &DependentApplication{
			Function: newFunc,
			Argument: newArg,
		}

	case *DependentLambda:
		if term.ParamName == varName {
			// Variable is shadowed, don't substitute.
			return term
		}

		newBody := dl.substituteInTerm(term.Body, varName, replacement)

		return &DependentLambda{
			ParamName: term.ParamName,
			ParamType: term.ParamType,
			Body:      newBody,
			Type:      term.Type,
		}

	default:
		return term
	}
}

// DependentTerm represents terms in dependent type theory.
type DependentTerm interface {
	String() string
	GetType() *Type
	Normalize() DependentTerm
}

// DependentVariable represents variables in dependent terms.
type DependentVariable struct {
	Type *Type
	Name string
}

func (dv *DependentVariable) String() string {
	return dv.Name
}

func (dv *DependentVariable) GetType() *Type {
	return dv.Type
}

func (dv *DependentVariable) Normalize() DependentTerm {
	return dv
}

// DependentApplication represents function application in dependent type theory.
type DependentApplication struct {
	Function DependentTerm
	Argument DependentTerm
}

func (da *DependentApplication) String() string {
	return fmt.Sprintf("(%s %s)", da.Function.String(), da.Argument.String())
}

func (da *DependentApplication) GetType() *Type {
	funcType := da.Function.GetType()
	if piType, ok := funcType.Data.(*PiType); ok {
		result, _ := piType.Apply(da.Argument.GetType())

		return result
	}

	return nil
}

func (da *DependentApplication) Normalize() DependentTerm {
	normalizedFunc := da.Function.Normalize()
	normalizedArg := da.Argument.Normalize()

	// If function is a lambda, perform beta reduction.
	if lambda, ok := normalizedFunc.(*DependentLambda); ok {
		result, _ := lambda.Apply(normalizedArg)

		return result.Normalize()
	}

	return &DependentApplication{
		Function: normalizedFunc,
		Argument: normalizedArg,
	}
}

// DependentPair represents pairs in dependent type theory.
type DependentPair struct {
	First  DependentTerm
	Second DependentTerm
	Type   *SigmaType
}

func (dp *DependentPair) String() string {
	return fmt.Sprintf("(%s, %s)", dp.First.String(), dp.Second.String())
}

func (dp *DependentPair) GetType() *Type {
	return &Type{
		Kind: TypeKindSigma,
		Data: dp.Type,
	}
}

func (dp *DependentPair) Normalize() DependentTerm {
	return &DependentPair{
		First:  dp.First.Normalize(),
		Second: dp.Second.Normalize(),
		Type:   dp.Type,
	}
}

// TypeKindSigma represents Sigma types.
const TypeKindSigma TypeKind = 100

// TypeKindPi represents Pi types.
const TypeKindPi TypeKind = 101

// TypeKindVariable represents type variables.
const TypeKindVariable TypeKind = 102

// DependentPattern represents patterns in dependent pattern matching.
type DependentPattern interface {
	String() string
	Match(term DependentTerm) (*PatternMatchResult, error)
	GetBindings() []string
}

// PatternMatchResult represents the result of pattern matching.
type PatternMatchResult struct {
	Bindings    map[string]DependentTerm
	Constraints []DependentConstraint
	Matched     bool
}

// DependentConstraint represents constraints arising from dependent pattern matching.
type DependentConstraint struct {
	Left  DependentTerm
	Right DependentTerm
	Kind  string // "equal", "type", etc.
}

// VariablePattern matches any term and binds it to a variable.
type VariablePattern struct {
	Type *Type
	Name string
}

func (vp *VariablePattern) String() string {
	if vp.Type != nil {
		return fmt.Sprintf("%s:%s", vp.Name, vp.Type.String())
	}

	return vp.Name
}

func (vp *VariablePattern) Match(term DependentTerm) (*PatternMatchResult, error) {
	bindings := map[string]DependentTerm{
		vp.Name: term,
	}

	var constraints []DependentConstraint
	if vp.Type != nil {
		constraints = append(constraints, DependentConstraint{
			Left:  &DependentTypeOf{Term: term},
			Right: &DependentTypeConstant{Type: vp.Type},
			Kind:  "type",
		})
	}

	return &PatternMatchResult{
		Matched:     true,
		Bindings:    bindings,
		Constraints: constraints,
	}, nil
}

func (vp *VariablePattern) GetBindings() []string {
	return []string{vp.Name}
}

// ConstructorPattern matches constructor applications.
type ConstructorPattern struct {
	Constructor string
	Args        []DependentPattern
}

func (cp *ConstructorPattern) String() string {
	if len(cp.Args) == 0 {
		return cp.Constructor
	}

	argStrs := make([]string, len(cp.Args))
	for i, arg := range cp.Args {
		argStrs[i] = arg.String()
	}

	return fmt.Sprintf("%s(%s)", cp.Constructor, strings.Join(argStrs, ", "))
}

func (cp *ConstructorPattern) Match(term DependentTerm) (*PatternMatchResult, error) {
	// This is simplified - in practice we'd check if term is a constructor application.
	// matching cp.Constructor and recursively match arguments
	return &PatternMatchResult{
		Matched:     false,
		Bindings:    make(map[string]DependentTerm),
		Constraints: []DependentConstraint{},
	}, nil
}

func (cp *ConstructorPattern) GetBindings() []string {
	var bindings []string
	for _, arg := range cp.Args {
		bindings = append(bindings, arg.GetBindings()...)
	}

	return bindings
}

// PairPattern matches dependent pairs.
type PairPattern struct {
	First  DependentPattern
	Second DependentPattern
}

func (pp *PairPattern) String() string {
	return fmt.Sprintf("(%s, %s)", pp.First.String(), pp.Second.String())
}

func (pp *PairPattern) Match(term DependentTerm) (*PatternMatchResult, error) {
	if pair, ok := term.(*DependentPair); ok {
		firstResult, err := pp.First.Match(pair.First)
		if err != nil || !firstResult.Matched {
			return &PatternMatchResult{Matched: false}, err
		}

		secondResult, err := pp.Second.Match(pair.Second)
		if err != nil || !secondResult.Matched {
			return &PatternMatchResult{Matched: false}, err
		}

		// Merge bindings.
		allBindings := make(map[string]DependentTerm)
		for k, v := range firstResult.Bindings {
			allBindings[k] = v
		}

		for k, v := range secondResult.Bindings {
			allBindings[k] = v
		}

		// Merge constraints.
		allConstraints := append(firstResult.Constraints, secondResult.Constraints...)

		return &PatternMatchResult{
			Matched:     true,
			Bindings:    allBindings,
			Constraints: allConstraints,
		}, nil
	}

	return &PatternMatchResult{Matched: false}, nil
}

func (pp *PairPattern) GetBindings() []string {
	return append(pp.First.GetBindings(), pp.Second.GetBindings()...)
}

// DependentCase represents a case in dependent pattern matching.
type DependentCase struct {
	Pattern DependentPattern
	Guard   DependentTerm // Optional guard condition
	Body    DependentTerm
}

// DependentMatch represents dependent pattern matching.
type DependentMatch struct {
	Scrutinee DependentTerm
	Cases     []DependentCase
}

func (dm *DependentMatch) String() string {
	var casesStr []string

	for _, c := range dm.Cases {
		if c.Guard != nil {
			casesStr = append(casesStr, fmt.Sprintf("| %s when %s => %s",
				c.Pattern.String(), c.Guard.String(), c.Body.String()))
		} else {
			casesStr = append(casesStr, fmt.Sprintf("| %s => %s",
				c.Pattern.String(), c.Body.String()))
		}
	}

	return fmt.Sprintf("match %s with\n%s", dm.Scrutinee.String(), strings.Join(casesStr, "\n"))
}

func (dm *DependentMatch) GetType() *Type {
	// In practice, this would perform type checking to ensure all cases have the same type.
	if len(dm.Cases) > 0 {
		return dm.Cases[0].Body.GetType()
	}

	return nil
}

func (dm *DependentMatch) Normalize() DependentTerm {
	normalizedScrutinee := dm.Scrutinee.Normalize()

	for _, caseClause := range dm.Cases {
		result, err := caseClause.Pattern.Match(normalizedScrutinee)
		if err == nil && result.Matched {
			// Check guard if present.
			if caseClause.Guard != nil {
				// Substitute bindings in guard.
				guardWithBindings := dm.substituteBindings(caseClause.Guard, result.Bindings)
				normalizedGuard := guardWithBindings.Normalize()

				// Simplified guard evaluation - in practice would be more sophisticated.
				if !dm.evaluateGuard(normalizedGuard) {
					continue
				}
			}

			// Substitute bindings in body.
			bodyWithBindings := dm.substituteBindings(caseClause.Body, result.Bindings)

			return bodyWithBindings.Normalize()
		}
	}

	// No case matched - return the match expression itself.
	return dm
}

func (dm *DependentMatch) substituteBindings(term DependentTerm, bindings map[string]DependentTerm) DependentTerm {
	switch t := term.(type) {
	case *DependentVariable:
		if replacement, exists := bindings[t.Name]; exists {
			return replacement
		}

		return t

	case *DependentApplication:
		newFunc := dm.substituteBindings(t.Function, bindings)
		newArg := dm.substituteBindings(t.Argument, bindings)

		return &DependentApplication{
			Function: newFunc,
			Argument: newArg,
		}

	case *DependentPair:
		newFirst := dm.substituteBindings(t.First, bindings)
		newSecond := dm.substituteBindings(t.Second, bindings)

		return &DependentPair{
			First:  newFirst,
			Second: newSecond,
			Type:   t.Type,
		}

	default:
		return term
	}
}

func (dm *DependentMatch) evaluateGuard(guard DependentTerm) bool {
	// Simplified guard evaluation - in practice would check if guard normalizes to true.
	return true
}

// DependentTypeOf represents type-of expressions.
type DependentTypeOf struct {
	Term DependentTerm
}

func (dto *DependentTypeOf) String() string {
	return fmt.Sprintf("typeof(%s)", dto.Term.String())
}

func (dto *DependentTypeOf) GetType() *Type {
	return TypeUniverse0.ToType()
}

func (dto *DependentTypeOf) Normalize() DependentTerm {
	return &DependentTypeConstant{Type: dto.Term.GetType()}
}

// DependentTypeConstant represents type constants in dependent terms.
type DependentTypeConstant struct {
	Type *Type
}

func (dtc *DependentTypeConstant) String() string {
	return dtc.Type.String()
}

func (dtc *DependentTypeConstant) GetType() *Type {
	return TypeUniverse0.ToType()
}

func (dtc *DependentTypeConstant) Normalize() DependentTerm {
	return dtc
}

// ToType converts a universe to a type.
func (u *Universe) ToType() *Type {
	return &Type{
		Kind: TypeKindUniverse,
		Data: u,
	}
}

// TypeKindUniverse represents type universes.
const TypeKindUniverse TypeKind = 103

// DependentTypeChecker handles type checking for dependent types.
type DependentTypeChecker struct {
	context     *DependentContext
	constraints []DependentConstraint
}

// DependentContext represents the typing context for dependent types.
type DependentContext struct {
	Variables map[string]*Type
	Types     map[string]*Universe
	Parent    *DependentContext
	Level     int
}

// NewDependentContext creates a new dependent typing context.
func NewDependentContext() *DependentContext {
	return &DependentContext{
		Variables: make(map[string]*Type),
		Types:     make(map[string]*Universe),
		Level:     0,
	}
}

// Extend creates a new context with an additional variable binding.
func (dc *DependentContext) Extend(name string, varType *Type) *DependentContext {
	return &DependentContext{
		Variables: map[string]*Type{name: varType},
		Types:     make(map[string]*Universe),
		Parent:    dc,
		Level:     dc.Level + 1,
	}
}

// Lookup finds a variable's type in the context.
func (dc *DependentContext) Lookup(name string) (*Type, bool) {
	if varType, exists := dc.Variables[name]; exists {
		return varType, true
	}

	if dc.Parent != nil {
		return dc.Parent.Lookup(name)
	}

	return nil, false
}

// NewDependentTypeChecker creates a new dependent type checker.
func NewDependentTypeChecker() *DependentTypeChecker {
	return &DependentTypeChecker{
		context:     NewDependentContext(),
		constraints: make([]DependentConstraint, 0),
	}
}

// CheckType checks that a term has a given type.
func (dtc *DependentTypeChecker) CheckType(term DependentTerm, expectedType *Type) error {
	inferredType := dtc.InferType(term)

	if !dtc.typesEqual(inferredType, expectedType) {
		return fmt.Errorf("type mismatch: expected %s, got %s",
			expectedType.String(), inferredType.String())
	}

	return nil
}

// InferType infers the type of a dependent term.
func (dtc *DependentTypeChecker) InferType(term DependentTerm) *Type {
	switch t := term.(type) {
	case *DependentVariable:
		if varType, exists := dtc.context.Lookup(t.Name); exists {
			return varType
		}

		return nil

	case *DependentApplication:
		funcType := dtc.InferType(t.Function)
		argType := dtc.InferType(t.Argument)

		if piType, ok := funcType.Data.(*PiType); ok {
			// Check argument type matches parameter type.
			if dtc.typesEqual(argType, piType.ParamType) {
				result, _ := piType.Apply(argType)

				return result
			}
		}

		return nil

	case *DependentLambda:
		return &Type{
			Kind: TypeKindPi,
			Data: t.Type,
		}

	case *DependentPair:
		return &Type{
			Kind: TypeKindSigma,
			Data: t.Type,
		}

	default:
		return nil
	}
}

// typesEqual checks if two types are equal (up to normalization).
func (dtc *DependentTypeChecker) typesEqual(type1, type2 *Type) bool {
	if type1 == nil || type2 == nil {
		return type1 == type2
	}

	if type1.Kind != type2.Kind {
		return false
	}

	// Simplified equality check - in practice would involve normalization.
	return type1.String() == type2.String()
}

// CheckPattern checks a pattern against a type.
func (dtc *DependentTypeChecker) CheckPattern(pattern DependentPattern, patternType *Type) error {
	switch p := pattern.(type) {
	case *VariablePattern:
		if p.Type != nil && !dtc.typesEqual(p.Type, patternType) {
			return fmt.Errorf("pattern type mismatch: expected %s, got %s",
				patternType.String(), p.Type.String())
		}

		return nil

	case *PairPattern:
		if sigmaType, ok := patternType.Data.(*SigmaType); ok {
			err := dtc.CheckPattern(p.First, sigmaType.FirstType)
			if err != nil {
				return err
			}

			return dtc.CheckPattern(p.Second, sigmaType.SecondType)
		}

		return fmt.Errorf("expected sigma type for pair pattern, got %s", patternType.String())

	default:
		return nil
	}
}

// Common dependent type constructors.

// NewPiType creates a new Pi type.
func NewPiType(paramName string, paramType *Type, returnType *Type) *Type {
	return &Type{
		Kind: TypeKindPi,
		Data: &PiType{
			ParamName:  paramName,
			ParamType:  paramType,
			ReturnType: returnType,
			Universe:   TypeUniverse0,
			IsImplicit: false,
		},
	}
}

// NewSigmaType creates a new Sigma type.
func NewSigmaType(firstName string, firstType *Type, secondType *Type) *Type {
	return &Type{
		Kind: TypeKindSigma,
		Data: &SigmaType{
			FirstName:  firstName,
			FirstType:  firstType,
			SecondType: secondType,
			Universe:   TypeUniverse0,
		},
	}
}

// NewVariableType creates a new type variable.
func NewVariableType(name string) *Type {
	return &Type{
		Kind: TypeKindVariable,
		Data: &VariableTypeData{Name: name},
	}
}

// NewDependentLambda creates a new dependent lambda.
func NewDependentLambda(paramName string, paramType *Type, body DependentTerm) *DependentLambda {
	piType := &PiType{
		ParamName:  paramName,
		ParamType:  paramType,
		ReturnType: body.GetType(),
		Universe:   TypeUniverse0,
		IsImplicit: false,
	}

	return &DependentLambda{
		ParamName: paramName,
		ParamType: paramType,
		Body:      body,
		Type:      piType,
	}
}

// NewDependentPair creates a new dependent pair.
func NewDependentPair(first, second DependentTerm, sigmaType *SigmaType) *DependentPair {
	return &DependentPair{
		First:  first,
		Second: second,
		Type:   sigmaType,
	}
}

// DependentTypeInference handles type inference for dependent types.
type DependentTypeInference struct {
	checker     *DependentTypeChecker
	unification *DependentUnification
}

// DependentUnification handles unification of dependent types.
type DependentUnification struct {
	substitutions map[string]*Type
}

// NewDependentUnification creates a new dependent type unification system.
func NewDependentUnification() *DependentUnification {
	return &DependentUnification{
		substitutions: make(map[string]*Type),
	}
}

// Unify attempts to unify two dependent types.
func (du *DependentUnification) Unify(type1, type2 *Type) error {
	if type1.Kind != type2.Kind {
		return fmt.Errorf("cannot unify types of different kinds: %s and %s",
			type1.String(), type2.String())
	}

	switch type1.Kind {
	case TypeKindVariable:
		data1 := type1.Data.(*VariableTypeData)
		if existing, exists := du.substitutions[data1.Name]; exists {
			return du.Unify(existing, type2)
		} else {
			du.substitutions[data1.Name] = type2

			return nil
		}

	case TypeKindPi:
		pi1 := type1.Data.(*PiType)
		pi2 := type2.Data.(*PiType)

		err := du.Unify(pi1.ParamType, pi2.ParamType)
		if err != nil {
			return err
		}

		return du.Unify(pi1.ReturnType, pi2.ReturnType)

	case TypeKindSigma:
		sigma1 := type1.Data.(*SigmaType)
		sigma2 := type2.Data.(*SigmaType)

		err := du.Unify(sigma1.FirstType, sigma2.FirstType)
		if err != nil {
			return err
		}

		return du.Unify(sigma1.SecondType, sigma2.SecondType)

	default:
		// For primitive types, check structural equality.
		if type1.String() == type2.String() {
			return nil
		}

		return fmt.Errorf("cannot unify %s and %s", type1.String(), type2.String())
	}
}

// ApplySubstitutions applies accumulated substitutions to a type.
func (du *DependentUnification) ApplySubstitutions(targetType *Type) *Type {
	switch targetType.Kind {
	case TypeKindVariable:
		data := targetType.Data.(*VariableTypeData)
		if subst, exists := du.substitutions[data.Name]; exists {
			return du.ApplySubstitutions(subst)
		}

		return targetType

	case TypeKindPi:
		pi := targetType.Data.(*PiType)

		return &Type{
			Kind: TypeKindPi,
			Data: &PiType{
				ParamName:  pi.ParamName,
				ParamType:  du.ApplySubstitutions(pi.ParamType),
				ReturnType: du.ApplySubstitutions(pi.ReturnType),
				Universe:   pi.Universe,
				IsImplicit: pi.IsImplicit,
			},
		}

	case TypeKindSigma:
		sigma := targetType.Data.(*SigmaType)

		return &Type{
			Kind: TypeKindSigma,
			Data: &SigmaType{
				FirstName:  sigma.FirstName,
				FirstType:  du.ApplySubstitutions(sigma.FirstType),
				SecondType: du.ApplySubstitutions(sigma.SecondType),
				Universe:   sigma.Universe,
			},
		}

	default:
		return targetType
	}
}
