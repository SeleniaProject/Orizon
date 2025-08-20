// Lifetime management system for Orizon MIR.
// This file implements lifetime tracking, inference, and validation for.
// memory safety in the Orizon language. It provides:
// 1. Lifetime variable representation and management
// 2. Lifetime constraint tracking and solving
// 3. Lifetime inference for function signatures and local variables
// 4. Integration with the borrow checker

package mir

import (
	"fmt"
	"strings"
)

// ====== Lifetime Core Types ======.

// LifetimeID represents a unique lifetime identifier.
type LifetimeID string

// Special lifetime constants.
const (
	StaticLifetime    LifetimeID = "'static"
	UnknownLifetime   LifetimeID = "'unknown"
	AnonymousLifetime LifetimeID = "'_"
)

// Lifetime represents a lifetime in the MIR.
type Lifetime struct {
	Scope       *LifetimeScope
	ID          LifetimeID
	Origin      LifetimeOrigin
	Constraints []*LifetimeConstraint
	Kind        LifetimeKind
}

// LifetimeKind classifies different types of lifetimes.
type LifetimeKind int

const (
	LifetimeStatic LifetimeKind = iota // 'static - lives for entire program
	LifetimeNamed                      // Named lifetime parameter like 'a
	LifetimeLocal                      // Local scope lifetime
	LifetimeTemp                       // Temporary lifetime for expressions
	LifetimeAnon                       // Anonymous lifetime '_
)

func (lk LifetimeKind) String() string {
	switch lk {
	case LifetimeStatic:
		return "static"
	case LifetimeNamed:
		return "named"
	case LifetimeLocal:
		return "local"
	case LifetimeTemp:
		return "temp"
	case LifetimeAnon:
		return "anon"
	default:
		return "unknown"
	}
}

// LifetimeOrigin tracks where a lifetime comes from.
type LifetimeOrigin struct {
	Function string
	Block    string
	Kind     LifetimeOriginKind
	Stmt     int
}

// LifetimeOriginKind classifies lifetime origins.
type LifetimeOriginKind int

const (
	OriginParameter LifetimeOriginKind = iota // Function parameter
	OriginLocal                               // Local variable
	OriginBorrow                              // Borrow expression
	OriginReturn                              // Return type
	OriginInferred                            // Inferred by compiler
)

// LifetimeScope represents the scope where a lifetime is valid.
type LifetimeScope struct {
	Parent    *LifetimeScope
	ID        string
	Function  string
	Block     string
	Children  []*LifetimeScope
	StartStmt int
	EndStmt   int
}

// ====== Lifetime Constraints ======.

// LifetimeConstraint represents a relationship between lifetimes.
type LifetimeConstraint struct {
	Origin *LifetimeOrigin
	From   LifetimeID
	To     LifetimeID
	Reason string
	Kind   ConstraintKind
}

// ConstraintKind represents different types of lifetime constraints.
type ConstraintKind int

const (
	ConstraintOutlives ConstraintKind = iota // 'a: 'b means 'a outlives 'b
	ConstraintEqual                          // 'a = 'b means lifetimes are equal
	ConstraintSubtype                        // 'a <: 'b means 'a is subtype of 'b
)

func (ck ConstraintKind) String() string {
	switch ck {
	case ConstraintOutlives:
		return "outlives"
	case ConstraintEqual:
		return "equal"
	case ConstraintSubtype:
		return "subtype"
	default:
		return "unknown"
	}
}

func (lc *LifetimeConstraint) String() string {
	switch lc.Kind {
	case ConstraintOutlives:
		return fmt.Sprintf("%s: %s", lc.From, lc.To)
	case ConstraintEqual:
		return fmt.Sprintf("%s = %s", lc.From, lc.To)
	case ConstraintSubtype:
		return fmt.Sprintf("%s <: %s", lc.From, lc.To)
	default:
		return fmt.Sprintf("%s ? %s", lc.From, lc.To)
	}
}

// ====== Lifetime Manager ======.

// LifetimeManager manages lifetime inference and checking.
type LifetimeManager struct {
	lifetimes   map[LifetimeID]*Lifetime
	scopes      map[string]*LifetimeScope
	constraints []*LifetimeConstraint
	errors      []error
	counter     int
}

// NewLifetimeManager creates a new lifetime manager.
func NewLifetimeManager() *LifetimeManager {
	lm := &LifetimeManager{
		lifetimes:   make(map[LifetimeID]*Lifetime),
		constraints: make([]*LifetimeConstraint, 0),
		scopes:      make(map[string]*LifetimeScope),
		counter:     0,
		errors:      make([]error, 0),
	}

	// Add the static lifetime.
	lm.addStaticLifetime()

	return lm
}

// addStaticLifetime adds the 'static lifetime.
func (lm *LifetimeManager) addStaticLifetime() {
	staticLifetime := &Lifetime{
		ID:   StaticLifetime,
		Kind: LifetimeStatic,
		Origin: LifetimeOrigin{
			Kind: OriginParameter,
		},
	}
	lm.lifetimes[StaticLifetime] = staticLifetime
}

// GenerateLifetimeID generates a unique lifetime ID.
func (lm *LifetimeManager) GenerateLifetimeID() LifetimeID {
	lm.counter++

	return LifetimeID(fmt.Sprintf("'lt_%d", lm.counter))
}

// CreateLifetime creates a new lifetime with the given properties.
func (lm *LifetimeManager) CreateLifetime(kind LifetimeKind, scope *LifetimeScope, origin LifetimeOrigin) *Lifetime {
	id := lm.GenerateLifetimeID()
	lifetime := &Lifetime{
		ID:          id,
		Kind:        kind,
		Scope:       scope,
		Constraints: make([]*LifetimeConstraint, 0),
		Origin:      origin,
	}
	lm.lifetimes[id] = lifetime

	return lifetime
}

// CreateNamedLifetime creates a named lifetime (like 'a).
func (lm *LifetimeManager) CreateNamedLifetime(name string, scope *LifetimeScope, origin LifetimeOrigin) *Lifetime {
	id := LifetimeID("'" + name)
	if existing, exists := lm.lifetimes[id]; exists {
		return existing
	}

	lifetime := &Lifetime{
		ID:          id,
		Kind:        LifetimeNamed,
		Scope:       scope,
		Constraints: make([]*LifetimeConstraint, 0),
		Origin:      origin,
	}
	lm.lifetimes[id] = lifetime

	return lifetime
}

// GetLifetime retrieves a lifetime by ID.
func (lm *LifetimeManager) GetLifetime(id LifetimeID) (*Lifetime, bool) {
	lifetime, exists := lm.lifetimes[id]

	return lifetime, exists
}

// ====== Constraint Management ======.

// AddConstraint adds a lifetime constraint.
func (lm *LifetimeManager) AddConstraint(kind ConstraintKind, from, to LifetimeID, reason string, origin *LifetimeOrigin) {
	constraint := &LifetimeConstraint{
		Kind:   kind,
		From:   from,
		To:     to,
		Reason: reason,
		Origin: origin,
	}
	lm.constraints = append(lm.constraints, constraint)

	// Add constraint to the from lifetime as well.
	if fromLifetime, exists := lm.lifetimes[from]; exists {
		fromLifetime.Constraints = append(fromLifetime.Constraints, constraint)
	}
}

// AddOutlivesConstraint adds an outlives constraint: from: to.
func (lm *LifetimeManager) AddOutlivesConstraint(from, to LifetimeID, reason string) {
	lm.AddConstraint(ConstraintOutlives, from, to, reason, nil)
}

// AddEqualConstraint adds an equality constraint: from = to.
func (lm *LifetimeManager) AddEqualConstraint(from, to LifetimeID, reason string) {
	lm.AddConstraint(ConstraintEqual, from, to, reason, nil)
}

// ====== Scope Management ======.

// CreateScope creates a new lifetime scope.
func (lm *LifetimeManager) CreateScope(id, function, block string, parent *LifetimeScope) *LifetimeScope {
	scope := &LifetimeScope{
		ID:        id,
		Parent:    parent,
		Children:  make([]*LifetimeScope, 0),
		Function:  function,
		Block:     block,
		StartStmt: -1,
		EndStmt:   -1,
	}

	if parent != nil {
		parent.Children = append(parent.Children, scope)
	}

	lm.scopes[id] = scope

	return scope
}

// GetScope retrieves a scope by ID.
func (lm *LifetimeManager) GetScope(id string) (*LifetimeScope, bool) {
	scope, exists := lm.scopes[id]

	return scope, exists
}

// IsAncestorScope checks if ancestor is an ancestor of descendant.
func (lm *LifetimeManager) IsAncestorScope(ancestor, descendant *LifetimeScope) bool {
	if descendant == nil {
		return false
	}

	if descendant.Parent == ancestor {
		return true
	}

	return lm.IsAncestorScope(ancestor, descendant.Parent)
}

// ====== Lifetime Inference ======.

// InferLifetimes performs lifetime inference for a function.
func (lm *LifetimeManager) InferLifetimes(function *Function) error {
	if function == nil {
		return fmt.Errorf("cannot infer lifetimes for nil function")
	}

	// Create function scope.
	funcScope := lm.CreateScope(
		fmt.Sprintf("func_%s", function.Name),
		function.Name,
		"",
		nil,
	)

	// Process parameters.
	for _, param := range function.Parameters {
		if err := lm.inferParameterLifetime(param, funcScope); err != nil {
			lm.errors = append(lm.errors, err)
		}
	}

	// Process blocks.
	for _, block := range function.Blocks {
		if err := lm.inferBlockLifetimes(block, funcScope); err != nil {
			lm.errors = append(lm.errors, err)
		}
	}

	return nil
}

// inferParameterLifetime infers lifetime for a function parameter.
func (lm *LifetimeManager) inferParameterLifetime(param Value, scope *LifetimeScope) error {
	// For now, create a lifetime for reference parameters.
	// In MIR, pointers are represented as ClassInt.
	if param.Kind == ValRef {
		origin := LifetimeOrigin{
			Kind:     OriginParameter,
			Function: scope.Function,
		}
		lm.CreateLifetime(LifetimeLocal, scope, origin)
	}

	return nil
}

// inferBlockLifetimes infers lifetimes for instructions in a block.
func (lm *LifetimeManager) inferBlockLifetimes(block *BasicBlock, funcScope *LifetimeScope) error {
	// Create block scope.
	blockScope := lm.CreateScope(
		fmt.Sprintf("block_%s", block.Name),
		funcScope.Function,
		block.Name,
		funcScope,
	)

	// Process each instruction.
	for i, instr := range block.Instr {
		if err := lm.inferInstructionLifetimes(instr, blockScope, i); err != nil {
			lm.errors = append(lm.errors, err)
		}
	}

	return nil
}

// inferInstructionLifetimes infers lifetimes for a single instruction.
func (lm *LifetimeManager) inferInstructionLifetimes(instr Instr, scope *LifetimeScope, stmtIndex int) error {
	switch inst := instr.(type) {
	case Alloca:
		// Local allocations get local lifetimes.
		origin := LifetimeOrigin{
			Kind:     OriginLocal,
			Function: scope.Function,
			Block:    scope.Block,
			Stmt:     stmtIndex,
		}
		lm.CreateLifetime(LifetimeLocal, scope, origin)

	case Load:
		// Loads might create borrows.
		origin := LifetimeOrigin{
			Kind:     OriginBorrow,
			Function: scope.Function,
			Block:    scope.Block,
			Stmt:     stmtIndex,
		}
		lm.CreateLifetime(LifetimeTemp, scope, origin)

	case Store:
		// Stores might affect lifetime constraints.
		// For now, no specific action needed.

	case Call:
		// Function calls need lifetime checking.
		return lm.inferCallLifetimes(inst, scope, stmtIndex)

	default:
		// Other instructions don't affect lifetimes directly.
	}

	return nil
}

// inferCallLifetimes infers lifetimes for function calls.
func (lm *LifetimeManager) inferCallLifetimes(call Call, scope *LifetimeScope, stmtIndex int) error {
	// For now, create temporary lifetimes for call results.
	if call.Callee != "" {
		origin := LifetimeOrigin{
			Kind:     OriginInferred,
			Function: scope.Function,
			Block:    scope.Block,
			Stmt:     stmtIndex,
		}
		lm.CreateLifetime(LifetimeTemp, scope, origin)
	}

	return nil
}

// ====== Constraint Solving ======.

// SolveConstraints attempts to solve all lifetime constraints.
func (lm *LifetimeManager) SolveConstraints() error {
	// Simple constraint solving - more sophisticated algorithms needed for production.
	for _, constraint := range lm.constraints {
		if err := lm.checkConstraint(constraint); err != nil {
			lm.errors = append(lm.errors, err)
		}
	}

	if len(lm.errors) > 0 {
		return fmt.Errorf("lifetime constraint solving failed with %d errors", len(lm.errors))
	}

	return nil
}

// checkConstraint validates a single constraint.
func (lm *LifetimeManager) checkConstraint(constraint *LifetimeConstraint) error {
	fromLifetime, fromExists := lm.lifetimes[constraint.From]
	toLifetime, toExists := lm.lifetimes[constraint.To]

	if !fromExists {
		return fmt.Errorf("lifetime %s not found in constraint %s", constraint.From, constraint)
	}

	if !toExists {
		return fmt.Errorf("lifetime %s not found in constraint %s", constraint.To, constraint)
	}

	switch constraint.Kind {
	case ConstraintOutlives:
		return lm.checkOutlivesConstraint(fromLifetime, toLifetime, constraint)
	case ConstraintEqual:
		return lm.checkEqualConstraint(fromLifetime, toLifetime, constraint)
	case ConstraintSubtype:
		return lm.checkSubtypeConstraint(fromLifetime, toLifetime, constraint)
	default:
		return fmt.Errorf("unknown constraint kind: %s", constraint.Kind)
	}
}

// checkOutlivesConstraint checks if 'from' outlives 'to'.
func (lm *LifetimeManager) checkOutlivesConstraint(from, to *Lifetime, constraint *LifetimeConstraint) error {
	// Static lifetime outlives everything.
	if from.ID == StaticLifetime {
		return nil
	}

	// Nothing outlives static (except static itself).
	if to.ID == StaticLifetime && from.ID != StaticLifetime {
		return fmt.Errorf("lifetime %s cannot outlive 'static: %s", from.ID, constraint.Reason)
	}

	// Check scope relationship.
	if from.Scope != nil && to.Scope != nil {
		if !lm.IsAncestorScope(from.Scope, to.Scope) && from.Scope != to.Scope {
			return fmt.Errorf("lifetime %s does not outlive %s: %s", from.ID, to.ID, constraint.Reason)
		}
	}

	return nil
}

// checkEqualConstraint checks if two lifetimes are equal.
func (lm *LifetimeManager) checkEqualConstraint(from, to *Lifetime, constraint *LifetimeConstraint) error {
	// For now, just check if they're the same lifetime.
	if from.ID != to.ID {
		return fmt.Errorf("lifetimes %s and %s are not equal: %s", from.ID, to.ID, constraint.Reason)
	}

	return nil
}

// checkSubtypeConstraint checks if 'from' is a subtype of 'to'.
func (lm *LifetimeManager) checkSubtypeConstraint(from, to *Lifetime, constraint *LifetimeConstraint) error {
	// Subtyping for lifetimes is the same as outlives.
	return lm.checkOutlivesConstraint(from, to, constraint)
}

// ====== Error Management ======.

// GetErrors returns all accumulated errors.
func (lm *LifetimeManager) GetErrors() []error {
	return lm.errors
}

// ClearErrors clears all accumulated errors.
func (lm *LifetimeManager) ClearErrors() {
	lm.errors = make([]error, 0)
}

// ====== Debug and Reporting ======.

// String returns a string representation of the lifetime manager.
func (lm *LifetimeManager) String() string {
	var b strings.Builder

	b.WriteString("LifetimeManager {\n")

	b.WriteString("  Lifetimes:\n")

	for id, lifetime := range lm.lifetimes {
		b.WriteString(fmt.Sprintf("    %s: %s\n", id, lifetime.Kind))
	}

	b.WriteString("  Constraints:\n")

	for _, constraint := range lm.constraints {
		b.WriteString(fmt.Sprintf("    %s (%s)\n", constraint, constraint.Reason))
	}

	b.WriteString("}\n")

	return b.String()
}

// ValidateLifetimes performs a complete lifetime validation for a module.
func (lm *LifetimeManager) ValidateLifetimes(module *Module) error {
	if module == nil {
		return fmt.Errorf("cannot validate lifetimes for nil module")
	}

	// Process each function.
	for _, function := range module.Functions {
		if err := lm.InferLifetimes(function); err != nil {
			return fmt.Errorf("lifetime inference failed for function %s: %w", function.Name, err)
		}
	}

	// Solve all constraints.
	if err := lm.SolveConstraints(); err != nil {
		return fmt.Errorf("constraint solving failed: %w", err)
	}

	return nil
}
