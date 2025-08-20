// Package types implements Phase 2.3.2 Index Types for the Orizon compiler.
// This system provides array bounds checking and length-dependent types for safer memory access.
package types

import (
	"fmt"
)

// IndexExpression represents expressions used in array indexing.
type IndexExpression interface {
	// String returns a string representation of the index expression.
	String() string

	// Evaluate attempts to evaluate the expression to a concrete value.
	Evaluate(env map[string]interface{}) (int64, error)

	// Variables returns the set of variables used in the expression.
	Variables() []string

	// Substitute substitutes variables in the expression.
	Substitute(substitutions map[string]interface{}) IndexExpression

	// Simplify performs algebraic simplification.
	Simplify() IndexExpression

	// IsStatic returns true if the expression can be evaluated at compile time.
	IsStatic() bool
}

// IndexExpressionKind represents different kinds of index expressions.
type IndexExpressionKind int

const (
	IndexKindConstant IndexExpressionKind = iota
	IndexKindVariable
	IndexKindBinary
	IndexKindLength
	IndexKindMin
	IndexKindMax
)

// ConstantIndexExpr represents a constant index value.
type ConstantIndexExpr struct {
	Value int64
}

func (e *ConstantIndexExpr) String() string {
	return fmt.Sprintf("%d", e.Value)
}

func (e *ConstantIndexExpr) Evaluate(env map[string]interface{}) (int64, error) {
	return e.Value, nil
}

func (e *ConstantIndexExpr) Variables() []string {
	return []string{}
}

func (e *ConstantIndexExpr) Substitute(substitutions map[string]interface{}) IndexExpression {
	return e
}

func (e *ConstantIndexExpr) Simplify() IndexExpression {
	return e
}

func (e *ConstantIndexExpr) IsStatic() bool {
	return true
}

// VariableIndexExpr represents a variable used in indexing.
type VariableIndexExpr struct {
	Name string
}

func (e *VariableIndexExpr) String() string {
	return e.Name
}

func (e *VariableIndexExpr) Evaluate(env map[string]interface{}) (int64, error) {
	if val, exists := env[e.Name]; exists {
		if intVal, ok := val.(int64); ok {
			return intVal, nil
		}

		if intVal, ok := val.(int); ok {
			return int64(intVal), nil
		}

		return 0, fmt.Errorf("variable %s is not an integer", e.Name)
	}

	return 0, fmt.Errorf("undefined variable: %s", e.Name)
}

func (e *VariableIndexExpr) Variables() []string {
	return []string{e.Name}
}

func (e *VariableIndexExpr) Substitute(substitutions map[string]interface{}) IndexExpression {
	if val, exists := substitutions[e.Name]; exists {
		if intVal, ok := val.(int64); ok {
			return &ConstantIndexExpr{Value: intVal}
		}

		if intVal, ok := val.(int); ok {
			return &ConstantIndexExpr{Value: int64(intVal)}
		}
	}

	return e
}

func (e *VariableIndexExpr) Simplify() IndexExpression {
	return e
}

func (e *VariableIndexExpr) IsStatic() bool {
	return false
}

// BinaryIndexOperator represents binary operators for index expressions.
type BinaryIndexOperator int

const (
	IndexOpAdd BinaryIndexOperator = iota // +
	IndexOpSub                            // -
	IndexOpMul                            // *
	IndexOpDiv                            // /
	IndexOpMod                            // %
	IndexOpMin                            // min
	IndexOpMax                            // max
)

func (op BinaryIndexOperator) String() string {
	switch op {
	case IndexOpAdd:
		return "+"
	case IndexOpSub:
		return "-"
	case IndexOpMul:
		return "*"
	case IndexOpDiv:
		return "/"
	case IndexOpMod:
		return "%"
	case IndexOpMin:
		return "min"
	case IndexOpMax:
		return "max"
	default:
		return "unknown"
	}
}

// BinaryIndexExpr represents binary operations on indices.
type BinaryIndexExpr struct {
	Left     IndexExpression
	Right    IndexExpression
	Operator BinaryIndexOperator
}

func (e *BinaryIndexExpr) String() string {
	switch e.Operator {
	case IndexOpMin, IndexOpMax:
		return fmt.Sprintf("%s(%s, %s)", e.Operator.String(), e.Left.String(), e.Right.String())
	default:
		return fmt.Sprintf("(%s %s %s)", e.Left.String(), e.Operator.String(), e.Right.String())
	}
}

func (e *BinaryIndexExpr) Evaluate(env map[string]interface{}) (int64, error) {
	leftVal, err := e.Left.Evaluate(env)
	if err != nil {
		return 0, err
	}

	rightVal, err := e.Right.Evaluate(env)
	if err != nil {
		return 0, err
	}

	switch e.Operator {
	case IndexOpAdd:
		return leftVal + rightVal, nil
	case IndexOpSub:
		return leftVal - rightVal, nil
	case IndexOpMul:
		return leftVal * rightVal, nil
	case IndexOpDiv:
		if rightVal == 0 {
			return 0, fmt.Errorf("division by zero")
		}

		return leftVal / rightVal, nil
	case IndexOpMod:
		if rightVal == 0 {
			return 0, fmt.Errorf("modulo by zero")
		}

		return leftVal % rightVal, nil
	case IndexOpMin:
		if leftVal < rightVal {
			return leftVal, nil
		}

		return rightVal, nil
	case IndexOpMax:
		if leftVal > rightVal {
			return leftVal, nil
		}

		return rightVal, nil
	default:
		return 0, fmt.Errorf("unknown binary operator: %s", e.Operator.String())
	}
}

func (e *BinaryIndexExpr) Variables() []string {
	vars := make(map[string]bool)
	for _, v := range e.Left.Variables() {
		vars[v] = true
	}

	for _, v := range e.Right.Variables() {
		vars[v] = true
	}

	result := make([]string, 0, len(vars))
	for v := range vars {
		result = append(result, v)
	}

	return result
}

func (e *BinaryIndexExpr) Substitute(substitutions map[string]interface{}) IndexExpression {
	return &BinaryIndexExpr{
		Left:     e.Left.Substitute(substitutions),
		Operator: e.Operator,
		Right:    e.Right.Substitute(substitutions),
	}
}

func (e *BinaryIndexExpr) Simplify() IndexExpression {
	left := e.Left.Simplify()
	right := e.Right.Simplify()

	// If both operands are constants, evaluate directly.
	if leftConst, leftOk := left.(*ConstantIndexExpr); leftOk {
		if rightConst, rightOk := right.(*ConstantIndexExpr); rightOk {
			result, err := e.evaluateConstants(leftConst.Value, rightConst.Value)
			if err == nil {
				return &ConstantIndexExpr{Value: result}
			}
		}
	}

	// Algebraic simplifications.
	switch e.Operator {
	case IndexOpAdd:
		// x + 0 = x.
		if rightConst, ok := right.(*ConstantIndexExpr); ok && rightConst.Value == 0 {
			return left
		}
		// 0 + x = x.
		if leftConst, ok := left.(*ConstantIndexExpr); ok && leftConst.Value == 0 {
			return right
		}

	case IndexOpSub:
		// x - 0 = x.
		if rightConst, ok := right.(*ConstantIndexExpr); ok && rightConst.Value == 0 {
			return left
		}
		// x - x = 0 (if both sides are identical).
		if e.expressionsEqual(left, right) {
			return &ConstantIndexExpr{Value: 0}
		}

	case IndexOpMul:
		// x * 0 = 0.
		if rightConst, ok := right.(*ConstantIndexExpr); ok && rightConst.Value == 0 {
			return &ConstantIndexExpr{Value: 0}
		}
		// 0 * x = 0.
		if leftConst, ok := left.(*ConstantIndexExpr); ok && leftConst.Value == 0 {
			return &ConstantIndexExpr{Value: 0}
		}
		// x * 1 = x.
		if rightConst, ok := right.(*ConstantIndexExpr); ok && rightConst.Value == 1 {
			return left
		}
		// 1 * x = x.
		if leftConst, ok := left.(*ConstantIndexExpr); ok && leftConst.Value == 1 {
			return right
		}

	case IndexOpDiv:
		// x / 1 = x
		if rightConst, ok := right.(*ConstantIndexExpr); ok && rightConst.Value == 1 {
			return left
		}
		// x / x = 1 (if both sides are identical and non-zero)
		if e.expressionsEqual(left, right) {
			return &ConstantIndexExpr{Value: 1}
		}
	}

	return &BinaryIndexExpr{
		Left:     left,
		Operator: e.Operator,
		Right:    right,
	}
}

func (e *BinaryIndexExpr) evaluateConstants(left, right int64) (int64, error) {
	switch e.Operator {
	case IndexOpAdd:
		return left + right, nil
	case IndexOpSub:
		return left - right, nil
	case IndexOpMul:
		return left * right, nil
	case IndexOpDiv:
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}

		return left / right, nil
	case IndexOpMod:
		if right == 0 {
			return 0, fmt.Errorf("modulo by zero")
		}

		return left % right, nil
	case IndexOpMin:
		if left < right {
			return left, nil
		}

		return right, nil
	case IndexOpMax:
		if left > right {
			return left, nil
		}

		return right, nil
	default:
		return 0, fmt.Errorf("unknown operator")
	}
}

func (e *BinaryIndexExpr) expressionsEqual(left, right IndexExpression) bool {
	// Simple structural equality check.
	return left.String() == right.String()
}

func (e *BinaryIndexExpr) IsStatic() bool {
	return e.Left.IsStatic() && e.Right.IsStatic()
}

// LengthIndexExpr represents the length of an array or slice.
type LengthIndexExpr struct {
	ArrayName string
}

func (e *LengthIndexExpr) String() string {
	return fmt.Sprintf("len(%s)", e.ArrayName)
}

func (e *LengthIndexExpr) Evaluate(env map[string]interface{}) (int64, error) {
	lengthKey := fmt.Sprintf("len(%s)", e.ArrayName)
	if val, exists := env[lengthKey]; exists {
		if intVal, ok := val.(int64); ok {
			return intVal, nil
		}

		if intVal, ok := val.(int); ok {
			return int64(intVal), nil
		}
	}

	return 0, fmt.Errorf("cannot determine length of %s", e.ArrayName)
}

func (e *LengthIndexExpr) Variables() []string {
	return []string{e.ArrayName}
}

func (e *LengthIndexExpr) Substitute(substitutions map[string]interface{}) IndexExpression {
	lengthKey := fmt.Sprintf("len(%s)", e.ArrayName)
	if val, exists := substitutions[lengthKey]; exists {
		if intVal, ok := val.(int64); ok {
			return &ConstantIndexExpr{Value: intVal}
		}

		if intVal, ok := val.(int); ok {
			return &ConstantIndexExpr{Value: int64(intVal)}
		}
	}

	return e
}

func (e *LengthIndexExpr) Simplify() IndexExpression {
	return e
}

func (e *LengthIndexExpr) IsStatic() bool {
	return false // Length is generally not known at compile time
}

// IndexBound represents bounds for array indices.
type IndexBound struct {
	Lower IndexExpression
	Upper IndexExpression
}

// String returns a string representation of the index bound.
func (ib *IndexBound) String() string {
	return fmt.Sprintf("[%s..%s)", ib.Lower.String(), ib.Upper.String())
}

// Contains checks if an index expression is within the bounds.
func (ib *IndexBound) Contains(index IndexExpression, env map[string]interface{}) (bool, error) {
	indexVal, err := index.Evaluate(env)
	if err != nil {
		return false, err
	}

	lowerVal, err := ib.Lower.Evaluate(env)
	if err != nil {
		return false, err
	}

	upperVal, err := ib.Upper.Evaluate(env)
	if err != nil {
		return false, err
	}

	return indexVal >= lowerVal && indexVal < upperVal, nil
}

// IndexType represents a type with associated index bounds.
type IndexType struct {
	BaseType      *Type
	ElementType   *Type
	LengthExpr    IndexExpression
	IndexBounds   *IndexBound
	IsFixedLength bool
}

// String returns a string representation of the index type.
func (it *IndexType) String() string {
	if it.IsFixedLength {
		return fmt.Sprintf("Array[%s, %s]", it.LengthExpr.String(), it.ElementType.String())
	}

	return fmt.Sprintf("Slice[%s]", it.ElementType.String())
}

// GetBounds returns the valid index bounds for this type.
func (it *IndexType) GetBounds() *IndexBound {
	if it.IndexBounds != nil {
		return it.IndexBounds
	}

	// Default bounds: [0..length)
	return &IndexBound{
		Lower: &ConstantIndexExpr{Value: 0},
		Upper: it.LengthExpr,
	}
}

// IsValidIndex checks if an index is valid for this type.
func (it *IndexType) IsValidIndex(index IndexExpression, env map[string]interface{}) (bool, error) {
	bounds := it.GetBounds()

	return bounds.Contains(index, env)
}

// IndexConstraint represents a constraint on array indexing.
type IndexConstraint struct {
	ArrayExpr Expr
	IndexExpr IndexExpression
	ArrayType *IndexType
	Message   string
	Location  SourceLocation
}

// IndexChecker handles index bounds checking.
type IndexChecker struct {
	environment map[string]*IndexType
	lengthEnv   map[string]IndexExpression
	constraints []IndexConstraint
}

// NewIndexChecker creates a new index bounds checker.
func NewIndexChecker() *IndexChecker {
	return &IndexChecker{
		constraints: make([]IndexConstraint, 0),
		environment: make(map[string]*IndexType),
		lengthEnv:   make(map[string]IndexExpression),
	}
}

// AddArrayVariable adds an array variable to the environment.
func (ic *IndexChecker) AddArrayVariable(name string, indexType *IndexType) {
	ic.environment[name] = indexType

	if indexType.LengthExpr != nil {
		lengthKey := fmt.Sprintf("len(%s)", name)
		ic.lengthEnv[lengthKey] = indexType.LengthExpr
	}
}

// CheckArrayAccess checks bounds for array access expressions.
func (ic *IndexChecker) CheckArrayAccess(arrayExpr Expr, index IndexExpression, location SourceLocation) error {
	// Try to determine the array type.
	arrayType, err := ic.getArrayType(arrayExpr)
	if err != nil {
		return err
	}

	// Add constraint for bounds checking.
	constraint := IndexConstraint{
		ArrayExpr: arrayExpr,
		IndexExpr: index,
		ArrayType: arrayType,
		Location:  location,
		Message:   fmt.Sprintf("Array index %s must be within bounds %s", index.String(), arrayType.GetBounds().String()),
	}

	ic.constraints = append(ic.constraints, constraint)

	return nil
}

// getArrayType attempts to determine the type of an array expression.
func (ic *IndexChecker) getArrayType(arrayExpr Expr) (*IndexType, error) {
	switch expr := arrayExpr.(type) {
	case *VariableExpr:
		if indexType, exists := ic.environment[expr.Name]; exists {
			return indexType, nil
		}

		return nil, fmt.Errorf("unknown array variable: %s", expr.Name)

	default:
		// For other expressions, we'd need more sophisticated type inference.
		return nil, fmt.Errorf("cannot determine type of array expression: %s", arrayExpr.String())
	}
}

// SolveIndexConstraints checks all collected index constraints.
func (ic *IndexChecker) SolveIndexConstraints() ([]IndexError, error) {
	var errors []IndexError

	// Build evaluation environment from length expressions.
	env := make(map[string]interface{})

	for name, lengthExpr := range ic.lengthEnv {
		if lengthExpr.IsStatic() {
			if val, err := lengthExpr.Evaluate(nil); err == nil {
				env[name] = val
			}
		}
	}

	for _, constraint := range ic.constraints {
		// Try to check the constraint.
		valid, err := constraint.ArrayType.IsValidIndex(constraint.IndexExpr, env)
		if err != nil {
			// Cannot determine statically - would need runtime check.
			continue
		}

		if !valid {
			errors = append(errors, IndexError{
				Message:   constraint.Message,
				Location:  constraint.Location,
				IndexExpr: constraint.IndexExpr,
				Bounds:    constraint.ArrayType.GetBounds(),
			})
		}
	}

	return errors, nil
}

// IndexError represents an error in index bounds checking.
type IndexError struct {
	IndexExpr IndexExpression
	Bounds    *IndexBound
	Message   string
	Location  SourceLocation
}

func (ie IndexError) Error() string {
	return fmt.Sprintf("Index bounds error at line %d, column %d: %s (index: %s, bounds: %s)",
		ie.Location.Line, ie.Location.Column, ie.Message, ie.IndexExpr.String(), ie.Bounds.String())
}

// ArrayAccessExpr represents array access expressions.
type ArrayAccessExpr struct {
	Array Expr
	Index IndexExpression
}

func (e *ArrayAccessExpr) String() string {
	return fmt.Sprintf("%s[%s]", e.Array.String(), e.Index.String())
}

func (e *ArrayAccessExpr) Accept(visitor ExprVisitor) (*Type, error) {
	// This would be implemented by a visitor that handles array access.
	return nil, fmt.Errorf("array access visitor not implemented")
}

// SliceExpr represents slice expressions.
type SliceExpr struct {
	Array Expr
	Start IndexExpression
	End   IndexExpression
}

func (e *SliceExpr) String() string {
	if e.Start != nil && e.End != nil {
		return fmt.Sprintf("%s[%s:%s]", e.Array.String(), e.Start.String(), e.End.String())
	} else if e.Start != nil {
		return fmt.Sprintf("%s[%s:]", e.Array.String(), e.Start.String())
	} else if e.End != nil {
		return fmt.Sprintf("%s[:%s]", e.Array.String(), e.End.String())
	} else {
		return fmt.Sprintf("%s[:]", e.Array.String())
	}
}

func (e *SliceExpr) Accept(visitor ExprVisitor) (*Type, error) {
	// This would be implemented by a visitor that handles slice expressions.
	return nil, fmt.Errorf("slice expression visitor not implemented")
}

// DependentArrayType represents array types with length-dependent properties.
type DependentArrayType struct {
	ElementType *Type
	LengthBound *RefinementType
	IndexBounds *IndexBound
	LengthVar   string
}

// String returns a string representation of the dependent array type.
func (dat *DependentArrayType) String() string {
	return fmt.Sprintf("Array[%s:%s, %s]", dat.LengthVar, dat.LengthBound.String(), dat.ElementType.String())
}

// CreateIndexType creates an IndexType from a DependentArrayType.
func (dat *DependentArrayType) CreateIndexType(length int64) *IndexType {
	lengthExpr := &ConstantIndexExpr{Value: length}

	return &IndexType{
		BaseType:      NewArrayType(dat.ElementType, int(length)),
		ElementType:   dat.ElementType,
		LengthExpr:    lengthExpr,
		IndexBounds:   dat.IndexBounds,
		IsFixedLength: true,
	}
}

// IndexExpressionParser handles parsing of index expressions.
type IndexExpressionParser struct {
	input    string
	position int
	current  rune
}

// NewIndexExpressionParser creates a new index expression parser.
func NewIndexExpressionParser(input string) *IndexExpressionParser {
	p := &IndexExpressionParser{
		input:    input,
		position: 0,
		current:  0,
	}
	p.advance()

	return p
}

func (p *IndexExpressionParser) advance() {
	if p.position < len(p.input) {
		p.current = rune(p.input[p.position])
		p.position++
	} else {
		p.current = 0 // EOF
	}
}

func (p *IndexExpressionParser) skipWhitespace() {
	for p.current == ' ' || p.current == '\t' || p.current == '\n' || p.current == '\r' {
		p.advance()
	}
}

// ParseIndexExpression parses an index expression from a string.
func (p *IndexExpressionParser) ParseIndexExpression() (IndexExpression, error) {
	p.skipWhitespace()

	return p.parseAddSub()
}

func (p *IndexExpressionParser) parseAddSub() (IndexExpression, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return nil, err
	}

	for p.current != 0 {
		p.skipWhitespace()

		var op BinaryIndexOperator

		var matched bool

		if p.current == '+' {
			op = IndexOpAdd
			matched = true

			p.advance()
		} else if p.current == '-' {
			op = IndexOpSub
			matched = true

			p.advance()
		}

		if matched {
			right, err := p.parseMulDiv()
			if err != nil {
				return nil, err
			}

			left = &BinaryIndexExpr{
				Left:     left,
				Operator: op,
				Right:    right,
			}
		} else {
			break
		}
	}

	return left, nil
}

func (p *IndexExpressionParser) parseMulDiv() (IndexExpression, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for p.current != 0 {
		p.skipWhitespace()

		var op BinaryIndexOperator

		var matched bool

		if p.current == '*' {
			op = IndexOpMul
			matched = true

			p.advance()
		} else if p.current == '/' {
			op = IndexOpDiv
			matched = true

			p.advance()
		} else if p.current == '%' {
			op = IndexOpMod
			matched = true

			p.advance()
		}

		if matched {
			right, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}

			left = &BinaryIndexExpr{
				Left:     left,
				Operator: op,
				Right:    right,
			}
		} else {
			break
		}
	}

	return left, nil
}

func (p *IndexExpressionParser) parsePrimary() (IndexExpression, error) {
	p.skipWhitespace()

	if p.current == '(' {
		p.advance()

		expr, err := p.parseAddSub()
		if err != nil {
			return nil, err
		}

		p.skipWhitespace()

		if p.current != ')' {
			return nil, fmt.Errorf("expected ')'")
		}

		p.advance()

		return expr, nil
	}

	// Check for len() function.
	if p.current == 'l' && p.match("len(") {
		start := p.position - 1 // Current position is already after "len("
		// We're now positioned after "len(", find the matching ")".
		for p.current != ')' && p.current != 0 {
			p.advance()
		}

		if p.current != ')' {
			return nil, fmt.Errorf("expected ')' after 'len('")
		}

		arrayName := p.input[start : p.position-1] // Extract content between ( and )
		p.advance()                                // consume ')'

		return &LengthIndexExpr{ArrayName: arrayName}, nil
	}

	// Parse identifier or number.
	if p.current == 0 {
		return nil, fmt.Errorf("unexpected end of input")
	}

	isValidStart := (p.current >= 'a' && p.current <= 'z') ||
		(p.current >= 'A' && p.current <= 'Z') ||
		(p.current >= '0' && p.current <= '9') ||
		p.current == '_'

	if !isValidStart {
		return nil, fmt.Errorf("unexpected character: %c at position %d", p.current, p.position-1)
	}

	// Collect token characters - fix position calculation.
	tokenStart := p.position - 1 // Save current character position

	// Advance through all token characters.
	count := 0

	for (p.current >= 'a' && p.current <= 'z') ||
		(p.current >= 'A' && p.current <= 'Z') ||
		(p.current >= '0' && p.current <= '9') ||
		p.current == '_' {
		p.advance()

		count++
		if count > 10 { // Safety check to prevent infinite loop
			return nil, fmt.Errorf("infinite loop detected in token parsing")
		}
	}

	tokenEnd := tokenStart + count // Calculate end based on characters consumed

	token := p.input[tokenStart:tokenEnd]
	if token == "" {
		return nil, fmt.Errorf("empty token at position %d (tokenStart=%d, tokenEnd=%d, pos=%d, current=%c)", tokenStart, tokenStart, tokenEnd, p.position, p.current)
	}

	// Try to parse as number.
	if token[0] >= '0' && token[0] <= '9' {
		val := int64(0)

		for _, ch := range token {
			if ch >= '0' && ch <= '9' {
				val = val*10 + int64(ch-'0')
			} else {
				return nil, fmt.Errorf("invalid number: %s", token)
			}
		}

		return &ConstantIndexExpr{Value: val}, nil
	}

	// Otherwise, it's a variable.
	return &VariableIndexExpr{Name: token}, nil
}

func (p *IndexExpressionParser) match(expected string) bool {
	// Check if we have enough characters remaining.
	startPos := p.position - 1
	if startPos < 0 || startPos+len(expected) > len(p.input) {
		return false
	}

	// Check if the string matches.
	actual := p.input[startPos : startPos+len(expected)]
	if actual == expected {
		// Advance past the matched string (we already consumed the first character).
		for i := 1; i < len(expected); i++ {
			p.advance()
		}
		// One more advance to get past the entire matched string.
		p.advance()

		return true
	}

	return false
}

// Common index type constructors.

// NewFixedArray creates a fixed-size array type with bounds checking.
func NewFixedArray(elementType *Type, length int64) *IndexType {
	return &IndexType{
		BaseType:    NewArrayType(elementType, int(length)),
		ElementType: elementType,
		LengthExpr:  &ConstantIndexExpr{Value: length},
		IndexBounds: &IndexBound{
			Lower: &ConstantIndexExpr{Value: 0},
			Upper: &ConstantIndexExpr{Value: length},
		},
		IsFixedLength: true,
	}
}

// NewDynamicSlice creates a dynamic slice type.
func NewDynamicSlice(elementType *Type, lengthVar string) *IndexType {
	return &IndexType{
		BaseType:    NewSliceType(elementType),
		ElementType: elementType,
		LengthExpr:  &LengthIndexExpr{ArrayName: lengthVar},
		IndexBounds: &IndexBound{
			Lower: &ConstantIndexExpr{Value: 0},
			Upper: &LengthIndexExpr{ArrayName: lengthVar},
		},
		IsFixedLength: false,
	}
}

// NewBoundedArray creates an array with custom bounds.
func NewBoundedArray(elementType *Type, length IndexExpression, lower, upper IndexExpression) *IndexType {
	return &IndexType{
		BaseType:    NewSliceType(elementType),
		ElementType: elementType,
		LengthExpr:  length,
		IndexBounds: &IndexBound{
			Lower: lower,
			Upper: upper,
		},
		IsFixedLength: length.IsStatic(),
	}
}
