// Package types implements Phase 2.3.1 Refinement Types for the Orizon compiler.
// This system provides type refinement through predicates for more precise type checking.
package types

import (
	"fmt"
)

// RefinementPredicate represents a predicate in a refinement type.
type RefinementPredicate interface {
	// String returns a string representation of the predicate.
	String() string

	// Evaluate evaluates the predicate against a value.
	Evaluate(value interface{}) (bool, error)

	// Variables returns the set of variables used in the predicate.
	Variables() []string

	// Substitute substitutes variables in the predicate.
	Substitute(substitutions map[string]interface{}) RefinementPredicate

	// Simplify performs logical simplification of the predicate.
	Simplify() RefinementPredicate
}

// PredicateKind represents the kind of predicate.
type PredicateKind int

const (
	PredicateKindTrue PredicateKind = iota
	PredicateKindFalse
	PredicateKindVariable
	PredicateKindConstant
	PredicateKindComparison
	PredicateKindLogical
	PredicateKindArithmetic
	PredicateKindApplication
)

// TruePredicate represents the always-true predicate.
type TruePredicate struct{}

func (p *TruePredicate) String() string {
	return "true"
}

func (p *TruePredicate) Evaluate(value interface{}) (bool, error) {
	return true, nil
}

func (p *TruePredicate) Variables() []string {
	return []string{}
}

func (p *TruePredicate) Substitute(substitutions map[string]interface{}) RefinementPredicate {
	return p
}

func (p *TruePredicate) Simplify() RefinementPredicate {
	return p
}

// FalsePredicate represents the always-false predicate.
type FalsePredicate struct{}

func (p *FalsePredicate) String() string {
	return "false"
}

func (p *FalsePredicate) Evaluate(value interface{}) (bool, error) {
	return false, nil
}

func (p *FalsePredicate) Variables() []string {
	return []string{}
}

func (p *FalsePredicate) Substitute(substitutions map[string]interface{}) RefinementPredicate {
	return p
}

func (p *FalsePredicate) Simplify() RefinementPredicate {
	return p
}

// VariablePredicate represents a variable in a predicate.
type VariablePredicate struct {
	Name string
}

func (p *VariablePredicate) String() string {
	return p.Name
}

func (p *VariablePredicate) Evaluate(value interface{}) (bool, error) {
	return false, fmt.Errorf("cannot evaluate unbound variable: %s", p.Name)
}

func (p *VariablePredicate) Variables() []string {
	return []string{p.Name}
}

func (p *VariablePredicate) Substitute(substitutions map[string]interface{}) RefinementPredicate {
	if val, exists := substitutions[p.Name]; exists {
		return &ConstantPredicate{Value: val}
	}

	return p
}

func (p *VariablePredicate) Simplify() RefinementPredicate {
	return p
}

// ConstantPredicate represents a constant value in a predicate.
type ConstantPredicate struct {
	Value interface{}
}

func (p *ConstantPredicate) String() string {
	return fmt.Sprintf("%v", p.Value)
}

func (p *ConstantPredicate) Evaluate(value interface{}) (bool, error) {
	// Constants evaluate to themselves in boolean context.
	if b, ok := p.Value.(bool); ok {
		return b, nil
	}

	return false, fmt.Errorf("cannot evaluate non-boolean constant as predicate")
}

func (p *ConstantPredicate) Variables() []string {
	return []string{}
}

func (p *ConstantPredicate) Substitute(substitutions map[string]interface{}) RefinementPredicate {
	return p
}

func (p *ConstantPredicate) Simplify() RefinementPredicate {
	return p
}

// ComparisonOperator represents comparison operators.
type ComparisonOperator int

const (
	CompOpEQ ComparisonOperator = iota // ==
	CompOpNE                           // !=
	CompOpLT                           // <
	CompOpLE                           // <=
	CompOpGT                           // >
	CompOpGE                           // >=
)

func (op ComparisonOperator) String() string {
	switch op {
	case CompOpEQ:
		return "=="
	case CompOpNE:
		return "!="
	case CompOpLT:
		return "<"
	case CompOpLE:
		return "<="
	case CompOpGT:
		return ">"
	case CompOpGE:
		return ">="
	default:
		return "unknown"
	}
}

// ComparisonPredicate represents comparison predicates.
type ComparisonPredicate struct {
	Left     RefinementPredicate
	Right    RefinementPredicate
	Operator ComparisonOperator
}

func (p *ComparisonPredicate) String() string {
	return fmt.Sprintf("(%s %s %s)", p.Left.String(), p.Operator.String(), p.Right.String())
}

func (p *ComparisonPredicate) Evaluate(value interface{}) (bool, error) {
	// Get raw values from operands.
	var leftVal, rightVal interface{}

	if constPred, ok := p.Left.(*ConstantPredicate); ok {
		leftVal = constPred.Value
	} else if varPred, ok := p.Left.(*VariablePredicate); ok {
		if env, ok := value.(map[string]interface{}); ok {
			if val, exists := env[varPred.Name]; exists {
				leftVal = val
			} else {
				return false, fmt.Errorf("undefined variable: %s", varPred.Name)
			}
		} else {
			return false, fmt.Errorf("variable evaluation requires environment")
		}
	} else {
		// For other predicate types, evaluate as boolean.
		boolVal, err := p.Left.Evaluate(value)
		if err != nil {
			return false, err
		}

		leftVal = boolVal
	}

	if constPred, ok := p.Right.(*ConstantPredicate); ok {
		rightVal = constPred.Value
	} else if varPred, ok := p.Right.(*VariablePredicate); ok {
		if env, ok := value.(map[string]interface{}); ok {
			if val, exists := env[varPred.Name]; exists {
				rightVal = val
			} else {
				return false, fmt.Errorf("undefined variable: %s", varPred.Name)
			}
		} else {
			return false, fmt.Errorf("variable evaluation requires environment")
		}
	} else {
		// For other predicate types, evaluate as boolean.
		boolVal, err := p.Right.Evaluate(value)
		if err != nil {
			return false, err
		}

		rightVal = boolVal
	}

	// Perform comparison based on value types.
	return p.compareValues(leftVal, rightVal)
}

func (p *ComparisonPredicate) compareValues(left, right interface{}) (bool, error) {
	// Convert to common numeric type if possible.
	leftNum, leftOk := p.toNumber(left)
	rightNum, rightOk := p.toNumber(right)

	if leftOk && rightOk {
		return p.compareNumbers(leftNum, rightNum), nil
	}

	// Fall back to direct comparison for non-numeric types.
	switch p.Operator {
	case CompOpEQ:
		return left == right, nil
	case CompOpNE:
		return left != right, nil
	default:
		return false, fmt.Errorf("comparison operator %s not supported for non-numeric values", p.Operator.String())
	}
}

func (p *ComparisonPredicate) toNumber(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case float32:
		return float64(v), true
	default:
		return 0, false
	}
}

func (p *ComparisonPredicate) compareNumbers(left, right float64) bool {
	switch p.Operator {
	case CompOpEQ:
		return left == right
	case CompOpNE:
		return left != right
	case CompOpLT:
		return left < right
	case CompOpLE:
		return left <= right
	case CompOpGT:
		return left > right
	case CompOpGE:
		return left >= right
	default:
		return false
	}
}

func (p *ComparisonPredicate) Variables() []string {
	vars := make(map[string]bool)
	for _, v := range p.Left.Variables() {
		vars[v] = true
	}

	for _, v := range p.Right.Variables() {
		vars[v] = true
	}

	result := make([]string, 0, len(vars))
	for v := range vars {
		result = append(result, v)
	}

	return result
}

func (p *ComparisonPredicate) Substitute(substitutions map[string]interface{}) RefinementPredicate {
	return &ComparisonPredicate{
		Left:     p.Left.Substitute(substitutions),
		Operator: p.Operator,
		Right:    p.Right.Substitute(substitutions),
	}
}

func (p *ComparisonPredicate) Simplify() RefinementPredicate {
	left := p.Left.Simplify()
	right := p.Right.Simplify()

	// If both sides are constants, evaluate directly.
	if leftConst, leftOk := left.(*ConstantPredicate); leftOk {
		if rightConst, rightOk := right.(*ConstantPredicate); rightOk {
			result, err := p.compareValues(leftConst.Value, rightConst.Value)
			if err == nil {
				if result {
					return &TruePredicate{}
				} else {
					return &FalsePredicate{}
				}
			}
		}
	}

	return &ComparisonPredicate{
		Left:     left,
		Operator: p.Operator,
		Right:    right,
	}
}

// LogicalOperator represents logical operators.
type LogicalOperator int

const (
	LogOpAnd     LogicalOperator = iota // &&
	LogOpOr                             // ||
	LogOpNot                            // !
	LogOpImplies                        // =>
	LogOpIff                            // <=>
)

func (op LogicalOperator) String() string {
	switch op {
	case LogOpAnd:
		return "&&"
	case LogOpOr:
		return "||"
	case LogOpNot:
		return "!"
	case LogOpImplies:
		return "=>"
	case LogOpIff:
		return "<=>"
	default:
		return "unknown"
	}
}

// LogicalPredicate represents logical combinations of predicates.
type LogicalPredicate struct {
	Left     RefinementPredicate
	Right    RefinementPredicate
	Operator LogicalOperator
}

func (p *LogicalPredicate) String() string {
	switch p.Operator {
	case LogOpNot:
		return fmt.Sprintf("(!%s)", p.Right.String())
	default:
		return fmt.Sprintf("(%s %s %s)", p.Left.String(), p.Operator.String(), p.Right.String())
	}
}

func (p *LogicalPredicate) Evaluate(value interface{}) (bool, error) {
	switch p.Operator {
	case LogOpNot:
		rightVal, err := p.Right.Evaluate(value)
		if err != nil {
			return false, err
		}

		return !rightVal, nil

	case LogOpAnd:
		leftVal, err := p.Left.Evaluate(value)
		if err != nil {
			return false, err
		}

		if !leftVal {
			return false, nil // Short-circuit evaluation
		}

		return p.Right.Evaluate(value)

	case LogOpOr:
		leftVal, err := p.Left.Evaluate(value)
		if err != nil {
			return false, err
		}

		if leftVal {
			return true, nil // Short-circuit evaluation
		}

		return p.Right.Evaluate(value)

	case LogOpImplies:
		leftVal, err := p.Left.Evaluate(value)
		if err != nil {
			return false, err
		}

		if !leftVal {
			return true, nil // False implies anything
		}

		return p.Right.Evaluate(value)

	case LogOpIff:
		leftVal, err := p.Left.Evaluate(value)
		if err != nil {
			return false, err
		}

		rightVal, err := p.Right.Evaluate(value)
		if err != nil {
			return false, err
		}

		return leftVal == rightVal, nil

	default:
		return false, fmt.Errorf("unknown logical operator: %s", p.Operator.String())
	}
}

func (p *LogicalPredicate) Variables() []string {
	vars := make(map[string]bool)

	if p.Left != nil {
		for _, v := range p.Left.Variables() {
			vars[v] = true
		}
	}

	for _, v := range p.Right.Variables() {
		vars[v] = true
	}

	result := make([]string, 0, len(vars))
	for v := range vars {
		result = append(result, v)
	}

	return result
}

func (p *LogicalPredicate) Substitute(substitutions map[string]interface{}) RefinementPredicate {
	var left RefinementPredicate
	if p.Left != nil {
		left = p.Left.Substitute(substitutions)
	}

	return &LogicalPredicate{
		Operator: p.Operator,
		Left:     left,
		Right:    p.Right.Substitute(substitutions),
	}
}

func (p *LogicalPredicate) Simplify() RefinementPredicate {
	var left RefinementPredicate
	if p.Left != nil {
		left = p.Left.Simplify()
	}

	right := p.Right.Simplify()

	switch p.Operator {
	case LogOpNot:
		// Double negation elimination.
		if notPred, ok := right.(*LogicalPredicate); ok && notPred.Operator == LogOpNot {
			return notPred.Right.Simplify()
		}
		// Negation of constants.
		if _, ok := right.(*TruePredicate); ok {
			return &FalsePredicate{}
		}

		if _, ok := right.(*FalsePredicate); ok {
			return &TruePredicate{}
		}

	case LogOpAnd:
		// Identity laws.
		if _, ok := left.(*TruePredicate); ok {
			return right
		}

		if _, ok := right.(*TruePredicate); ok {
			return left
		}
		// Annihilation laws.
		if _, ok := left.(*FalsePredicate); ok {
			return &FalsePredicate{}
		}

		if _, ok := right.(*FalsePredicate); ok {
			return &FalsePredicate{}
		}

	case LogOpOr:
		// Identity laws.
		if _, ok := left.(*FalsePredicate); ok {
			return right
		}

		if _, ok := right.(*FalsePredicate); ok {
			return left
		}
		// Annihilation laws.
		if _, ok := left.(*TruePredicate); ok {
			return &TruePredicate{}
		}

		if _, ok := right.(*TruePredicate); ok {
			return &TruePredicate{}
		}
	}

	return &LogicalPredicate{
		Operator: p.Operator,
		Left:     left,
		Right:    right,
	}
}

// RefinementType represents a type with an associated predicate.
type RefinementType struct {
	Predicate RefinementPredicate
	BaseType  *Type
	Variable  string
}

// String returns a string representation of the refinement type.
func (rt *RefinementType) String() string {
	return fmt.Sprintf("{%s:%s | %s}", rt.Variable, rt.BaseType.String(), rt.Predicate.String())
}

// IsSubtypeOf checks if this refinement type is a subtype of another.
func (rt *RefinementType) IsSubtypeOf(other *RefinementType) (bool, error) {
	// Base types must be compatible.
	if !rt.BaseType.IsAssignableFrom(other.BaseType) {
		return false, nil
	}

	// For refinement subtyping: {x:T | P} <: {x:T | Q} iff P => Q.
	// We need to check if our predicate implies the other predicate.
	implication := &LogicalPredicate{
		Operator: LogOpImplies,
		Left:     rt.Predicate,
		Right:    other.Predicate,
	}

	// For now, we'll use a simple syntactic check.
	// In a full implementation, this would involve SMT solving.
	return rt.checkImplicationSyntactically(implication), nil
}

// checkImplicationSyntactically performs a simple syntactic check for implications.
func (rt *RefinementType) checkImplicationSyntactically(implication *LogicalPredicate) bool {
	// Simplify the implication.
	simplified := implication.Simplify()

	// If it simplifies to true, the implication holds.
	if _, ok := simplified.(*TruePredicate); ok {
		return true
	}

	// For more complex cases, we'd need a proper theorem prover.
	// For now, we'll be conservative and return false.
	return false
}

// RefinementChecker handles type checking with refinement types.
type RefinementChecker struct {
	baseChecker   *BidirectionalChecker
	predicateEnv  map[string]RefinementPredicate
	refinementEnv map[string]*RefinementType
	constraints   []RefinementConstraint
}

// RefinementConstraint represents a constraint to be checked.
type RefinementConstraint struct {
	Predicate RefinementPredicate
	Message   string
	Location  SourceLocation
}

// NewRefinementChecker creates a new refinement type checker.
func NewRefinementChecker(baseChecker *BidirectionalChecker) *RefinementChecker {
	return &RefinementChecker{
		baseChecker:   baseChecker,
		predicateEnv:  make(map[string]RefinementPredicate),
		refinementEnv: make(map[string]*RefinementType),
		constraints:   make([]RefinementConstraint, 0),
	}
}

// CheckRefinementType checks an expression against a refinement type.
func (rc *RefinementChecker) CheckRefinementType(expr Expr, refinementType *RefinementType) (*RefinementType, error) {
	// First check the base type.
	baseType, err := rc.baseChecker.CheckExpression(expr, refinementType.BaseType)
	if err != nil {
		return nil, err
	}

	// Generate constraints for the predicate.
	constraint := RefinementConstraint{
		Predicate: refinementType.Predicate,
		Location:  SourceLocation{Line: 0, Column: 0}, // Would be filled in with actual location
		Message:   fmt.Sprintf("Expression must satisfy refinement predicate: %s", refinementType.Predicate.String()),
	}

	rc.constraints = append(rc.constraints, constraint)

	// Return the refined type.
	return &RefinementType{
		BaseType:  baseType,
		Variable:  refinementType.Variable,
		Predicate: refinementType.Predicate,
	}, nil
}

// SynthesizeRefinementType attempts to synthesize a refinement type for an expression.
func (rc *RefinementChecker) SynthesizeRefinementType(expr Expr) (*RefinementType, error) {
	// For now, create a simple refinement type with true predicate.
	// In practice, this would analyze the expression to generate constraints.
	baseType := &Type{Kind: TypeKindInt32} // Simplified base type

	return &RefinementType{
		BaseType:  baseType,
		Variable:  "x",
		Predicate: &TruePredicate{},
	}, nil
}

// generatePredicateForExpression generates a predicate for an expression.
func (rc *RefinementChecker) generatePredicateForExpression(expr Expr) RefinementPredicate {
	switch e := expr.(type) {
	case *LiteralExpr:
		// For literals, generate v == literal.
		return &ComparisonPredicate{
			Left:     &VariablePredicate{Name: "v"},
			Operator: CompOpEQ,
			Right:    &ConstantPredicate{Value: e.Value},
		}

	case *VariableExpr:
		// For variables, look up any known refinement.
		if refinement, exists := rc.refinementEnv[e.Name]; exists {
			// Substitute the variable name.
			substitutions := map[string]interface{}{
				refinement.Variable: &VariablePredicate{Name: "v"},
			}

			return refinement.Predicate.Substitute(substitutions)
		}
		// Otherwise, just return true.
		return &TruePredicate{}

	default:
		// For other expressions, return true (no refinement).
		return &TruePredicate{}
	}
}

// SolveConstraints attempts to solve all accumulated constraints.
func (rc *RefinementChecker) SolveConstraints() ([]RefinementError, error) {
	var errors []RefinementError

	for _, constraint := range rc.constraints {
		// For now, we'll just check if the predicate is obviously false.
		simplified := constraint.Predicate.Simplify()

		if _, ok := simplified.(*FalsePredicate); ok {
			errors = append(errors, RefinementError{
				Message:   constraint.Message,
				Location:  constraint.Location,
				Predicate: constraint.Predicate,
			})
		}
	}

	return errors, nil
}

// RefinementError represents an error in refinement type checking.
type RefinementError struct {
	Predicate RefinementPredicate
	Message   string
	Location  SourceLocation
}

func (re RefinementError) Error() string {
	locationStr := fmt.Sprintf("%s:%d:%d", re.Location.File, re.Location.Line, re.Location.Column)

	return fmt.Sprintf("Refinement error at %s: %s (predicate: %s)",
		locationStr, re.Message, re.Predicate.String())
}

// PredicateParser handles parsing of refinement predicates.
type PredicateParser struct {
	input    string
	position int
	current  rune
}

// NewPredicateParser creates a new predicate parser.
func NewPredicateParser(input string) *PredicateParser {
	p := &PredicateParser{
		input:    input,
		position: 0,
	}
	p.advance()

	return p
}

func (p *PredicateParser) advance() {
	if p.position < len(p.input) {
		p.current = rune(p.input[p.position])
		p.position++
	} else {
		p.current = 0 // EOF
	}
}

func (p *PredicateParser) skipWhitespace() {
	for p.current == ' ' || p.current == '\t' || p.current == '\n' || p.current == '\r' {
		p.advance()
	}
}

// ParsePredicate parses a predicate from a string.
func (p *PredicateParser) ParsePredicate() (RefinementPredicate, error) {
	p.skipWhitespace()

	return p.parseOr()
}

func (p *PredicateParser) parseOr() (RefinementPredicate, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.current != 0 {
		p.skipWhitespace()
		// Check for ||.
		if p.position-1+2 <= len(p.input) && p.input[p.position-1:p.position-1+2] == "||" {
			p.advance()
			p.advance()

			right, err := p.parseAnd()
			if err != nil {
				return nil, err
			}

			left = &LogicalPredicate{
				Operator: LogOpOr,
				Left:     left,
				Right:    right,
			}
		} else {
			break
		}
	}

	return left, nil
}

func (p *PredicateParser) parseAnd() (RefinementPredicate, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	for p.current != 0 {
		p.skipWhitespace()
		// Check for &&.
		if p.position-1+2 <= len(p.input) && p.input[p.position-1:p.position-1+2] == "&&" {
			p.advance()
			p.advance()

			right, err := p.parseComparison()
			if err != nil {
				return nil, err
			}

			left = &LogicalPredicate{
				Operator: LogOpAnd,
				Left:     left,
				Right:    right,
			}
		} else {
			break
		}
	}

	return left, nil
}

func (p *PredicateParser) parseComparison() (RefinementPredicate, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	p.skipWhitespace()

	var op ComparisonOperator

	var matched bool

	// Check for two-character operators first.
	if p.position-1+2 <= len(p.input) {
		twoChar := p.input[p.position-1 : p.position-1+2]
		if twoChar == "==" {
			op = CompOpEQ
			matched = true

			p.advance()
			p.advance()
		} else if twoChar == "!=" {
			op = CompOpNE
			matched = true

			p.advance()
			p.advance()
		} else if twoChar == "<=" {
			op = CompOpLE
			matched = true

			p.advance()
			p.advance()
		} else if twoChar == ">=" {
			op = CompOpGE
			matched = true

			p.advance()
			p.advance()
		}
	}

	// Check for single-character operators if no two-character match.
	if !matched {
		if p.current == '<' {
			op = CompOpLT
			matched = true

			p.advance()
		} else if p.current == '>' {
			op = CompOpGT
			matched = true

			p.advance()
		}
	}

	if matched {
		p.skipWhitespace()

		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}

		return &ComparisonPredicate{
			Left:     left,
			Operator: op,
			Right:    right,
		}, nil
	}

	return left, nil
}

func (p *PredicateParser) parsePrimary() (RefinementPredicate, error) {
	p.skipWhitespace()

	if p.current == '!' {
		p.advance()

		pred, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}

		return &LogicalPredicate{
			Operator: LogOpNot,
			Right:    pred,
		}, nil
	}

	if p.current == '(' {
		p.advance()

		pred, err := p.parseOr()
		if err != nil {
			return nil, err
		}

		p.skipWhitespace()

		if p.current != ')' {
			return nil, fmt.Errorf("expected ')'")
		}

		p.advance()

		return pred, nil
	}

	// Check for "true".
	if p.position-1+4 <= len(p.input) && p.input[p.position-1:p.position-1+4] == "true" {
		p.advance()
		p.advance()
		p.advance()
		p.advance()

		return &TruePredicate{}, nil
	}

	// Check for "false".
	if p.position-1+5 <= len(p.input) && p.input[p.position-1:p.position-1+5] == "false" {
		p.advance()
		p.advance()
		p.advance()
		p.advance()
		p.advance()

		return &FalsePredicate{}, nil
	}

	// Parse identifier or number.
	if p.current == 0 {
		return nil, fmt.Errorf("unexpected end of input")
	}

	// Check if current character is valid start.
	if !((p.current >= 'a' && p.current <= 'z') ||
		(p.current >= 'A' && p.current <= 'Z') ||
		(p.current >= '0' && p.current <= '9') ||
		p.current == '_') {
		return nil, fmt.Errorf("unexpected character: %c", p.current)
	}

	start := p.position - 1 // Position of current character

	charCount := 0 // Count characters we actually consume
	for (p.current >= 'a' && p.current <= 'z') ||
		(p.current >= 'A' && p.current <= 'Z') ||
		(p.current >= '0' && p.current <= '9') ||
		p.current == '_' {
		charCount++

		p.advance()
	}

	end := start + charCount // Position after consuming valid characters

	token := p.input[start:end]

	// Check if token is empty.
	if len(token) == 0 {
		return nil, fmt.Errorf("empty token")
	}

	// Try to parse as number.
	if token[0] >= '0' && token[0] <= '9' {
		// Simple integer parsing for demonstration.
		val := 0

		for _, ch := range token {
			if ch >= '0' && ch <= '9' {
				val = val*10 + int(ch-'0')
			} else {
				return nil, fmt.Errorf("invalid number: %s", token)
			}
		}

		return &ConstantPredicate{Value: val}, nil
	}

	// Otherwise, it's a variable.
	return &VariablePredicate{Name: token}, nil
}

func (p *PredicateParser) match(expected string) bool {
	// Check if we have enough characters remaining.
	if p.position-1+len(expected) > len(p.input) {
		return false
	}

	// Check if the string matches from current position (position-1 because position is already advanced).
	actual := p.input[p.position-1 : p.position-1+len(expected)]
	if actual == expected {
		// Advance past the matched string (we already consumed the first character).
		for i := 0; i < len(expected)-1; i++ {
			p.advance()
		}

		p.advance() // Advance past the last character

		return true
	}

	return false
}

// Common refinement type constructors.

// NewPositiveType creates a refinement type for positive numbers.
func NewPositiveType(baseType *Type) *RefinementType {
	return &RefinementType{
		BaseType: baseType,
		Variable: "v",
		Predicate: &ComparisonPredicate{
			Left:     &VariablePredicate{Name: "v"},
			Operator: CompOpGT,
			Right:    &ConstantPredicate{Value: 0},
		},
	}
}

// NewNonZeroType creates a refinement type for non-zero numbers.
func NewNonZeroType(baseType *Type) *RefinementType {
	return &RefinementType{
		BaseType: baseType,
		Variable: "v",
		Predicate: &ComparisonPredicate{
			Left:     &VariablePredicate{Name: "v"},
			Operator: CompOpNE,
			Right:    &ConstantPredicate{Value: 0},
		},
	}
}

// NewRangeType creates a refinement type for values in a range.
func NewRangeType(baseType *Type, min, max interface{}) *RefinementType {
	minPred := &ComparisonPredicate{
		Left:     &VariablePredicate{Name: "v"},
		Operator: CompOpGE,
		Right:    &ConstantPredicate{Value: min},
	}

	maxPred := &ComparisonPredicate{
		Left:     &VariablePredicate{Name: "v"},
		Operator: CompOpLE,
		Right:    &ConstantPredicate{Value: max},
	}

	return &RefinementType{
		BaseType: baseType,
		Variable: "v",
		Predicate: &LogicalPredicate{
			Operator: LogOpAnd,
			Left:     minPred,
			Right:    maxPred,
		},
	}
}
