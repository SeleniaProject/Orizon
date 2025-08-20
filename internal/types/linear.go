// Package types implements Phase 2.4.1 Linear Type System for the Orizon compiler.
// This system provides linear types and uniqueness checking for safe resource management.
package types

import (
	"fmt"
	"strings"
)

// LinearityKind represents different kinds of linearity.
type LinearityKind int

const (
	LinearKindAffine       LinearityKind = iota // Use at most once
	LinearKindLinear                            // Use exactly once
	LinearKindRelevant                          // Use at least once
	LinearKindUnrestricted                      // Use any number of times
)

// String returns a string representation of the linearity kind.
func (lk LinearityKind) String() string {
	switch lk {
	case LinearKindAffine:
		return "affine"
	case LinearKindLinear:
		return "linear"
	case LinearKindRelevant:
		return "relevant"
	case LinearKindUnrestricted:
		return "unrestricted"
	default:
		return "unknown"
	}
}

// LinearTypeWrapper represents a linear type with usage constraints.
type LinearTypeWrapper struct {
	BaseType      *Type
	UniqueId      string
	Linearity     LinearityKind
	UsageCount    int
	IsConsumed    bool
	MoveSemantics bool
}

// String returns a string representation of the linear type.
func (lt *LinearTypeWrapper) String() string {
	suffix := ""

	switch lt.Linearity {
	case LinearKindAffine:
		suffix = "?"
	case LinearKindLinear:
		suffix = "!"
	case LinearKindRelevant:
		suffix = "+"
	case LinearKindUnrestricted:
		suffix = "*"
	}

	consumed := ""
	if lt.IsConsumed {
		consumed = " (consumed)"
	}

	return fmt.Sprintf("%s%s%s", lt.BaseType.String(), suffix, consumed)
}

// CanUse checks if the linear type can be used again.
func (lt *LinearTypeWrapper) CanUse() bool {
	if lt.IsConsumed {
		return false
	}

	switch lt.Linearity {
	case LinearKindAffine:
		return lt.UsageCount == 0
	case LinearKindLinear:
		return lt.UsageCount == 0
	case LinearKindRelevant:
		return true
	case LinearKindUnrestricted:
		return true
	default:
		return false
	}
}

// Use marks the linear type as used.
func (lt *LinearTypeWrapper) Use() error {
	if !lt.CanUse() {
		return fmt.Errorf("linear type %s cannot be used (already consumed or used)", lt.String())
	}

	lt.UsageCount++

	switch lt.Linearity {
	case LinearKindLinear, LinearKindAffine:
		lt.IsConsumed = true
	}

	return nil
}

// Clone creates a copy of the linear type (for move semantics).
func (lt *LinearTypeWrapper) Clone() *LinearTypeWrapper {
	return &LinearTypeWrapper{
		BaseType:      lt.BaseType,
		Linearity:     lt.Linearity,
		UsageCount:    0,
		IsConsumed:    false,
		MoveSemantics: lt.MoveSemantics,
		UniqueId:      generateUniqueId(),
	}
}

// IsLinearlyEquivalent checks if two linear types are equivalent.
func (lt *LinearTypeWrapper) IsLinearlyEquivalent(other *LinearTypeWrapper) bool {
	return lt.BaseType.Equals(other.BaseType) && lt.Linearity == other.Linearity
}

// LinearVariable represents a variable with linear constraints.
type LinearVariable struct {
	Type     *LinearTypeWrapper
	Name     string
	Borrows  []*LinearBorrow
	Location SourceLocation
	LastUsed SourceLocation
	IsMoved  bool
}

// String returns a string representation of the linear variable.
func (lv *LinearVariable) String() string {
	moved := ""
	if lv.IsMoved {
		moved = " (moved)"
	}

	return fmt.Sprintf("%s: %s%s", lv.Name, lv.Type.String(), moved)
}

// CanBorrow checks if the variable can be borrowed.
func (lv *LinearVariable) CanBorrow(borrowKind BorrowKind) bool {
	if lv.IsMoved {
		return false
	}

	if lv.Type.IsConsumed {
		return false
	}

	// Check for conflicting borrows.
	for _, borrow := range lv.Borrows {
		if borrow.IsActive && borrow.ConflictsWith(borrowKind) {
			return false
		}
	}

	return true
}

// BorrowKind represents different kinds of borrows.
type BorrowKind int

const (
	BorrowKindShared    BorrowKind = iota // &T
	BorrowKindMutable                     // &mut T
	BorrowKindUniqueRef                   // &uniq T
)

// String returns a string representation of the borrow kind.
func (bk BorrowKind) String() string {
	switch bk {
	case BorrowKindShared:
		return "&"
	case BorrowKindMutable:
		return "&mut"
	case BorrowKindUniqueRef:
		return "&uniq"
	default:
		return "&unknown"
	}
}

// ConflictsWith checks if this borrow conflicts with another.
func (bk BorrowKind) ConflictsWith(other BorrowKind) bool {
	// Mutable borrows conflict with everything.
	if bk == BorrowKindMutable || other == BorrowKindMutable {
		return true
	}

	// Unique borrows conflict with everything.
	if bk == BorrowKindUniqueRef || other == BorrowKindUniqueRef {
		return true
	}

	// Shared borrows don't conflict with each other.
	return false
}

// LinearBorrow represents a borrow of a linear variable.
type LinearBorrow struct {
	Lifetime *Lifetime
	Borrower string
	Location SourceLocation
	Kind     BorrowKind
	IsActive bool
}

// ConflictsWith checks if this borrow conflicts with a borrow kind.
func (lb *LinearBorrow) ConflictsWith(kind BorrowKind) bool {
	return lb.Kind.ConflictsWith(kind)
}

// Lifetime represents the lifetime of a borrow.
type Lifetime struct {
	Name     string
	Start    SourceLocation
	End      SourceLocation
	IsStatic bool
}

// String returns a string representation of the lifetime.
func (l *Lifetime) String() string {
	if l.IsStatic {
		return "'static"
	}

	return fmt.Sprintf("'%s", l.Name)
}

// Contains checks if this lifetime contains another.
func (l *Lifetime) Contains(other *Lifetime) bool {
	if l.IsStatic {
		return true
	}

	if other.IsStatic {
		return false
	}

	// Simplified lifetime containment check.
	return l.Start.IsBefore(other.Start) && l.End.IsAfter(other.End)
}

// IsBefore checks if this location is before another.
func (sl SourceLocation) IsBefore(other SourceLocation) bool {
	if sl.File != other.File {
		return sl.File < other.File
	}

	if sl.Line != other.Line {
		return sl.Line < other.Line
	}

	return sl.Column < other.Column
}

// IsAfter checks if this location is after another.
func (sl SourceLocation) IsAfter(other SourceLocation) bool {
	return other.IsBefore(sl)
}

// LinearContext represents the context for linear type checking.
type LinearContext struct {
	Variables map[string]*LinearVariable
	Borrows   map[string]*LinearBorrow
	Moves     map[string]SourceLocation
	Parent    *LinearContext
	Scope     string
}

// NewLinearContext creates a new linear context.
func NewLinearContext() *LinearContext {
	return &LinearContext{
		Variables: make(map[string]*LinearVariable),
		Borrows:   make(map[string]*LinearBorrow),
		Moves:     make(map[string]SourceLocation),
		Parent:    nil,
		Scope:     "global",
	}
}

// Extend creates a new child context.
func (lc *LinearContext) Extend(scope string) *LinearContext {
	return &LinearContext{
		Variables: make(map[string]*LinearVariable),
		Borrows:   make(map[string]*LinearBorrow),
		Moves:     make(map[string]SourceLocation),
		Parent:    lc,
		Scope:     scope,
	}
}

// AddVariable adds a linear variable to the context.
func (lc *LinearContext) AddVariable(name string, varType *LinearTypeWrapper, location SourceLocation) error {
	if _, exists := lc.Variables[name]; exists {
		return fmt.Errorf("variable %s already exists in scope %s", name, lc.Scope)
	}

	lc.Variables[name] = &LinearVariable{
		Name:     name,
		Type:     varType,
		Location: location,
		IsMoved:  false,
		Borrows:  make([]*LinearBorrow, 0),
	}

	return nil
}

// LookupVariable looks up a variable in the context hierarchy.
func (lc *LinearContext) LookupVariable(name string) (*LinearVariable, bool) {
	if variable, exists := lc.Variables[name]; exists {
		return variable, true
	}

	if lc.Parent != nil {
		return lc.Parent.LookupVariable(name)
	}

	return nil, false
}

// UseVariable marks a variable as used.
func (lc *LinearContext) UseVariable(name string, location SourceLocation) error {
	variable, exists := lc.LookupVariable(name)
	if !exists {
		return fmt.Errorf("variable %s not found", name)
	}

	if variable.IsMoved {
		return fmt.Errorf("use of moved variable %s at %s:%d:%d",
			name, location.File, location.Line, location.Column)
	}

	err := variable.Type.Use()
	if err != nil {
		return fmt.Errorf("cannot use variable %s: %w", name, err)
	}

	variable.LastUsed = location

	return nil
}

// MoveVariable marks a variable as moved.
func (lc *LinearContext) MoveVariable(name string, location SourceLocation) error {
	variable, exists := lc.LookupVariable(name)
	if !exists {
		return fmt.Errorf("variable %s not found", name)
	}

	if variable.IsMoved {
		return fmt.Errorf("variable %s already moved", name)
	}

	if !variable.Type.MoveSemantics {
		return fmt.Errorf("variable %s does not support move semantics", name)
	}

	variable.IsMoved = true
	lc.Moves[name] = location

	// Invalidate all borrows.
	for _, borrow := range variable.Borrows {
		borrow.IsActive = false
	}

	return nil
}

// BorrowVariable creates a borrow of a variable.
func (lc *LinearContext) BorrowVariable(name string, borrowKind BorrowKind, borrower string, location SourceLocation) (*LinearBorrow, error) {
	variable, exists := lc.LookupVariable(name)
	if !exists {
		return nil, fmt.Errorf("variable %s not found", name)
	}

	if !variable.CanBorrow(borrowKind) {
		return nil, fmt.Errorf("cannot borrow variable %s as %s", name, borrowKind.String())
	}

	borrow := &LinearBorrow{
		Kind:     borrowKind,
		Borrower: borrower,
		Location: location,
		IsActive: true,
		Lifetime: &Lifetime{
			Name:     fmt.Sprintf("'%s_%s", name, borrower),
			Start:    location,
			IsStatic: false,
		},
	}

	variable.Borrows = append(variable.Borrows, borrow)
	lc.Borrows[borrower] = borrow

	return borrow, nil
}

// LinearityChecker performs linear type checking.
type LinearityChecker struct {
	context       *LinearContext
	moveSemantics map[*Type]bool
	constraints   []LinearityConstraint
}

// LinearityConstraint represents a constraint in linear type checking.
type LinearityConstraint struct {
	Variable    string
	Description string
	Location    SourceLocation
	Kind        LinearityConstraintKind
}

// LinearityConstraintKind represents kinds of linearity constraints.
type LinearityConstraintKind int

const (
	ConstraintKindMustUse LinearityConstraintKind = iota
	ConstraintKindUseOnce
	ConstraintKindNoBorrow
	ConstraintKindNoMove
)

// String returns a string representation of the constraint kind.
func (lck LinearityConstraintKind) String() string {
	switch lck {
	case ConstraintKindMustUse:
		return "must-use"
	case ConstraintKindUseOnce:
		return "use-once"
	case ConstraintKindNoBorrow:
		return "no-borrow"
	case ConstraintKindNoMove:
		return "no-move"
	default:
		return "unknown"
	}
}

// NewLinearityChecker creates a new linearity checker.
func NewLinearityChecker() *LinearityChecker {
	return &LinearityChecker{
		context:       NewLinearContext(),
		constraints:   make([]LinearityConstraint, 0),
		moveSemantics: make(map[*Type]bool),
	}
}

// LinearMoveExpr represents a move expression.
type LinearMoveExpr struct {
	Variable string
	Location SourceLocation
}

// Accept implements the Expr interface.
func (e *LinearMoveExpr) Accept(visitor ExprVisitor) (*Type, error) {
	// Linear move expressions require special handling.
	return nil, fmt.Errorf("linear move expressions require linear type checker")
}

func (e *LinearMoveExpr) String() string {
	return fmt.Sprintf("move %s", e.Variable)
}

// LinearBorrowExpr represents a borrow expression.
type LinearBorrowExpr struct {
	Variable string
	Location SourceLocation
	Kind     BorrowKind
}

// Accept implements the Expr interface.
func (e *LinearBorrowExpr) Accept(visitor ExprVisitor) (*Type, error) {
	// Linear borrow expressions require special handling.
	return nil, fmt.Errorf("linear borrow expressions require linear type checker")
}

func (e *LinearBorrowExpr) String() string {
	return fmt.Sprintf("%s %s", e.Kind.String(), e.Variable)
}

// LinearAssignmentExpr represents a linear assignment expression.
type LinearAssignmentExpr struct {
	Target   string
	Value    Expr
	Location SourceLocation
}

// Accept implements the Expr interface.
func (e *LinearAssignmentExpr) Accept(visitor ExprVisitor) (*Type, error) {
	// Linear assignment expressions require special handling.
	return nil, fmt.Errorf("linear assignment expressions require linear type checker")
}

func (e *LinearAssignmentExpr) String() string {
	return fmt.Sprintf("%s = %s", e.Target, e.Value.String())
}

// LinearFunctionCallExpr represents a linear function call expression.
type LinearFunctionCallExpr struct {
	Function  string
	Arguments []Expr
	Location  SourceLocation
}

// Accept implements the Expr interface.
func (e *LinearFunctionCallExpr) Accept(visitor ExprVisitor) (*Type, error) {
	// Linear function call expressions require special handling.
	return nil, fmt.Errorf("linear function call expressions require linear type checker")
}

func (e *LinearFunctionCallExpr) String() string {
	args := make([]string, len(e.Arguments))
	for i, arg := range e.Arguments {
		args[i] = arg.String()
	}

	return fmt.Sprintf("%s(%s)", e.Function, strings.Join(args, ", "))
}

// CheckLinearity performs linearity checking on an expression.
func (lc *LinearityChecker) CheckLinearity(expr Expr) error {
	switch e := expr.(type) {
	case *VariableExpr:
		return lc.checkVariableUsage(e)
	case *LinearMoveExpr:
		return lc.checkLinearMoveExpression(e)
	case *LinearBorrowExpr:
		return lc.checkLinearBorrowExpression(e)
	case *LinearAssignmentExpr:
		return lc.checkLinearAssignment(e)
	case *LinearFunctionCallExpr:
		return lc.checkLinearFunctionCall(e)
	default:
		// Recursively check subexpressions.
		return lc.checkSubExpressions(expr)
	}
}

// checkVariableUsage checks the usage of a variable.
func (lc *LinearityChecker) checkVariableUsage(expr *VariableExpr) error {
	// Note: VariableExpr from algorithm_w.go doesn't have Location field
	// We'll use a default location for now.
	defaultLocation := SourceLocation{File: "unknown", Line: 0, Column: 0}

	return lc.context.UseVariable(expr.Name, defaultLocation)
}

// checkLinearMoveExpression checks a move expression.
func (lc *LinearityChecker) checkLinearMoveExpression(expr *LinearMoveExpr) error {
	return lc.context.MoveVariable(expr.Variable, expr.Location)
}

// checkLinearBorrowExpression checks a borrow expression.
func (lc *LinearityChecker) checkLinearBorrowExpression(expr *LinearBorrowExpr) error {
	borrowId := fmt.Sprintf("borrow_%s_%d", expr.Variable, expr.Location.Line)
	_, err := lc.context.BorrowVariable(expr.Variable, expr.Kind, borrowId, expr.Location)

	return err
}

// checkLinearAssignment checks an assignment for linearity violations.
func (lc *LinearityChecker) checkLinearAssignment(expr *LinearAssignmentExpr) error {
	// Check if the assigned value is linear.
	if lc.isLinearExpression(expr.Value) {
		// Linear values must be moved, not copied.
		if moveExpr, ok := expr.Value.(*LinearMoveExpr); ok {
			return lc.checkLinearMoveExpression(moveExpr)
		} else {
			return fmt.Errorf("linear value must be explicitly moved in assignment")
		}
	}

	return lc.CheckLinearity(expr.Value)
}

// checkLinearFunctionCall checks a function call for linearity.
func (lc *LinearityChecker) checkLinearFunctionCall(expr *LinearFunctionCallExpr) error {
	// Check each argument.
	for i, arg := range expr.Arguments {
		err := lc.CheckLinearity(arg)
		if err != nil {
			return fmt.Errorf("argument %d: %w", i, err)
		}

		// Check if linear arguments are properly moved.
		if lc.isLinearExpression(arg) {
			if _, ok := arg.(*LinearMoveExpr); !ok {
				return fmt.Errorf("linear argument %d must be explicitly moved", i)
			}
		}
	}

	return nil
}

// checkSubExpressions recursively checks subexpressions.
func (lc *LinearityChecker) checkSubExpressions(expr Expr) error {
	// This would be implemented based on the specific AST structure.
	// For now, return nil (simplified).
	return nil
}

// isLinearExpression checks if an expression has linear type.
func (lc *LinearityChecker) isLinearExpression(expr Expr) bool {
	// This would check the type of the expression.
	// For now, simplified implementation.
	return false
}

// ValidateLinearity validates all linearity constraints.
func (lc *LinearityChecker) ValidateLinearity() []LinearityError {
	var errors []LinearityError

	// Check that all linear variables have been used.
	for name, variable := range lc.context.Variables {
		if variable.Type.Linearity == LinearKindLinear && variable.Type.UsageCount == 0 {
			errors = append(errors, LinearityError{
				Kind:     ErrorKindUnusedLinear,
				Variable: name,
				Location: variable.Location,
				Message:  fmt.Sprintf("linear variable %s must be used", name),
			})
		}

		if variable.Type.Linearity == LinearKindRelevant && variable.Type.UsageCount == 0 {
			errors = append(errors, LinearityError{
				Kind:     ErrorKindUnusedRelevant,
				Variable: name,
				Location: variable.Location,
				Message:  fmt.Sprintf("relevant variable %s must be used at least once", name),
			})
		}
	}

	// Check for active borrows at end of scope.
	for borrower, borrow := range lc.context.Borrows {
		if borrow.IsActive {
			errors = append(errors, LinearityError{
				Kind:     ErrorKindActiveBorrow,
				Variable: borrower,
				Location: borrow.Location,
				Message:  fmt.Sprintf("borrow %s is still active at end of scope", borrower),
			})
		}
	}

	return errors
}

// LinearityError represents a linearity checking error.
type LinearityError struct {
	Variable string
	Message  string
	Location SourceLocation
	Kind     LinearityErrorKind
}

// LinearityErrorKind represents kinds of linearity errors.
type LinearityErrorKind int

const (
	ErrorKindUnusedLinear LinearityErrorKind = iota
	ErrorKindUnusedRelevant
	ErrorKindDoubleUse
	ErrorKindUseAfterMove
	ErrorKindBorrowConflict
	ErrorKindActiveBorrow
)

// String returns a string representation of the error kind.
func (lek LinearityErrorKind) String() string {
	switch lek {
	case ErrorKindUnusedLinear:
		return "unused-linear"
	case ErrorKindUnusedRelevant:
		return "unused-relevant"
	case ErrorKindDoubleUse:
		return "double-use"
	case ErrorKindUseAfterMove:
		return "use-after-move"
	case ErrorKindBorrowConflict:
		return "borrow-conflict"
	case ErrorKindActiveBorrow:
		return "active-borrow"
	default:
		return "unknown"
	}
}

// Error implements the error interface.
func (le LinearityError) Error() string {
	return fmt.Sprintf("Linearity error (%s) at %s:%d:%d: %s",
		le.Kind.String(), le.Location.File, le.Location.Line, le.Location.Column, le.Message)
}

// LinearityAnalyzer performs static analysis for linearity.
type LinearityAnalyzer struct {
	checker   *LinearityChecker
	flowGraph *ControlFlowGraph
	reachable map[string]bool
}

// ControlFlowGraph represents the control flow of a program.
type ControlFlowGraph struct {
	Nodes map[string]*CFGNode
	Edges map[string][]*CFGEdge
}

// CFGNode represents a node in the control flow graph.
type CFGNode struct {
	Id           string
	Statement    interface{} // Generic statement interface
	Variables    map[string]*LinearVariable
	Successors   []*CFGNode
	Predecessors []*CFGNode
}

// CFGEdge represents an edge in the control flow graph.
type CFGEdge struct {
	From      *CFGNode
	To        *CFGNode
	Condition Expr
}

// NewLinearityAnalyzer creates a new linearity analyzer.
func NewLinearityAnalyzer() *LinearityAnalyzer {
	return &LinearityAnalyzer{
		checker: NewLinearityChecker(),
		flowGraph: &ControlFlowGraph{
			Nodes: make(map[string]*CFGNode),
			Edges: make(map[string][]*CFGEdge),
		},
		reachable: make(map[string]bool),
	}
}

// AnalyzeFunction analyzes a function for linearity.
func (la *LinearityAnalyzer) AnalyzeFunction(function *Function) []LinearityError {
	// Build control flow graph.
	la.buildControlFlowGraph(function)

	// Perform reachability analysis.
	la.computeReachability()

	// Check linearity constraints.
	return la.checkLinearityConstraints()
}

// buildControlFlowGraph constructs the control flow graph.
func (la *LinearityAnalyzer) buildControlFlowGraph(function *Function) {
	// This would build the CFG from the function's statements.
	// Simplified implementation.
}

// computeReachability computes reachable nodes.
func (la *LinearityAnalyzer) computeReachability() {
	// This would perform reachability analysis.
	// Simplified implementation.
}

// checkLinearityConstraints checks all linearity constraints.
func (la *LinearityAnalyzer) checkLinearityConstraints() []LinearityError {
	return la.checker.ValidateLinearity()
}

// Function represents a function for analysis.
type Function struct {
	ReturnType *Type
	Name       string
	Parameters []*Parameter
	Body       []interface{}
}

// Parameter represents a function parameter.
type Parameter struct {
	Type *LinearTypeWrapper
	Name string
}

// Common linear type constructors.

// NewLinearTypeWrapper creates a new linear type.
func NewLinearTypeWrapper(baseType *Type, linearity LinearityKind) *LinearTypeWrapper {
	return &LinearTypeWrapper{
		BaseType:      baseType,
		Linearity:     linearity,
		UsageCount:    0,
		IsConsumed:    false,
		MoveSemantics: linearity == LinearKindLinear || linearity == LinearKindAffine,
		UniqueId:      generateUniqueId(),
	}
}

// NewAffineType creates a new affine type (use at most once).
func NewAffineType(baseType *Type) *LinearTypeWrapper {
	return NewLinearTypeWrapper(baseType, LinearKindAffine)
}

// NewStrictLinearType creates a new linear type (use exactly once).
func NewStrictLinearType(baseType *Type) *LinearTypeWrapper {
	return NewLinearTypeWrapper(baseType, LinearKindLinear)
}

// NewRelevantType creates a new relevant type (use at least once).
func NewRelevantType(baseType *Type) *LinearTypeWrapper {
	return NewLinearTypeWrapper(baseType, LinearKindRelevant)
}

// NewUnrestrictedType creates a new unrestricted type.
func NewUnrestrictedType(baseType *Type) *LinearTypeWrapper {
	return NewLinearTypeWrapper(baseType, LinearKindUnrestricted)
}

// generateUniqueId generates a unique identifier.
func generateUniqueId() string {
	// Simplified unique ID generation.
	return fmt.Sprintf("id_%d", len("temp"))
}

// LinearTypeKind represents linear type kinds in the type system.
const (
	TypeKindLinearWrapper TypeKind = 200
)

// Remove conflicting type constant.

// Resource management types.

// ResourceType represents a type that manages resources.
type ResourceType struct {
	BaseType   *Type
	Finalizer  string
	Resource   ResourceKind
	IsAcquired bool
	IsReleased bool
}

// ResourceKind represents different kinds of resources.
type ResourceKind int

const (
	ResourceKindFile ResourceKind = iota
	ResourceKindMemory
	ResourceKindSocket
	ResourceKindMutex
	ResourceKindDatabase
)

// String returns a string representation of the resource kind.
func (rk ResourceKind) String() string {
	switch rk {
	case ResourceKindFile:
		return "file"
	case ResourceKindMemory:
		return "memory"
	case ResourceKindSocket:
		return "socket"
	case ResourceKindMutex:
		return "mutex"
	case ResourceKindDatabase:
		return "database"
	default:
		return "unknown"
	}
}

// AcquireResource marks a resource as acquired.
func (rt *ResourceType) AcquireResource() error {
	if rt.IsAcquired {
		return fmt.Errorf("resource %s already acquired", rt.Resource.String())
	}

	rt.IsAcquired = true

	return nil
}

// ReleaseResource marks a resource as released.
func (rt *ResourceType) ReleaseResource() error {
	if !rt.IsAcquired {
		return fmt.Errorf("resource %s not acquired", rt.Resource.String())
	}

	if rt.IsReleased {
		return fmt.Errorf("resource %s already released", rt.Resource.String())
	}

	rt.IsReleased = true

	return nil
}

// NewResourceType creates a new resource type.
func NewResourceType(baseType *Type, resource ResourceKind, finalizer string) *ResourceType {
	return &ResourceType{
		BaseType:   baseType,
		Resource:   resource,
		Finalizer:  finalizer,
		IsAcquired: false,
		IsReleased: false,
	}
}

// LinearityInference performs inference for linear types.
type LinearityInference struct {
	context     *LinearContext
	unification *LinearUnification
	constraints []LinearityConstraint
}

// LinearUnification handles unification of linear types.
type LinearUnification struct {
	substitutions map[string]*LinearTypeWrapper
}

// NewLinearUnification creates a new linear type unification system.
func NewLinearUnification() *LinearUnification {
	return &LinearUnification{
		substitutions: make(map[string]*LinearTypeWrapper),
	}
}

// UnifyLinear attempts to unify two linear types.
func (lu *LinearUnification) UnifyLinear(type1, type2 *LinearTypeWrapper) error {
	if !type1.BaseType.Equals(type2.BaseType) {
		return fmt.Errorf("cannot unify linear types with different base types: %s and %s",
			type1.BaseType.String(), type2.BaseType.String())
	}

	// Unify linearity kinds (more restrictive wins).
	unifiedLinearity := lu.unifyLinearity(type1.Linearity, type2.Linearity)

	type1.Linearity = unifiedLinearity
	type2.Linearity = unifiedLinearity

	return nil
}

// unifyLinearity unifies two linearity kinds.
func (lu *LinearUnification) unifyLinearity(l1, l2 LinearityKind) LinearityKind {
	// Linear is most restrictive.
	if l1 == LinearKindLinear || l2 == LinearKindLinear {
		return LinearKindLinear
	}

	// Affine is next most restrictive.
	if l1 == LinearKindAffine || l2 == LinearKindAffine {
		return LinearKindAffine
	}

	// Relevant requires at least one use.
	if l1 == LinearKindRelevant || l2 == LinearKindRelevant {
		return LinearKindRelevant
	}

	// Unrestricted is least restrictive.
	return LinearKindUnrestricted
}
