// Borrow checking system for Orizon MIR
// This file implements the borrow checker that ensures memory safety
// by tracking borrows, validating borrow rules, and preventing data races.
// It provides:
// 1. Borrow tracking and validation
// 2. Mutable and immutable borrow rules enforcement
// 3. Lifetime-based access checking
// 4. Integration with the lifetime system

package mir

import (
	"fmt"
	"strings"
)

// ====== Borrow Core Types ======

// BorrowKind represents the type of borrow
type BorrowKind int

const (
	BorrowImmutable BorrowKind = iota // &T - immutable borrow
	BorrowMutable                     // &mut T - mutable borrow
	BorrowShared                      // Special case for shared references
)

func (bk BorrowKind) String() string {
	switch bk {
	case BorrowImmutable:
		return "&"
	case BorrowMutable:
		return "&mut"
	case BorrowShared:
		return "&shared"
	default:
		return "&unknown"
	}
}

// Borrow represents a borrow operation in MIR
type Borrow struct {
	ID          BorrowID      // Unique identifier for this borrow
	Kind        BorrowKind    // Type of borrow
	Borrowed    Value         // The value being borrowed
	Borrower    Value         // The reference value created
	Lifetime    LifetimeID    // Lifetime of the borrow
	Region      *BorrowRegion // Region where borrow is active
	Origin      BorrowOrigin  // Where the borrow originated
	Constraints []*BorrowConstraint
}

// BorrowID represents a unique borrow identifier
type BorrowID string

// BorrowOrigin tracks where a borrow comes from
type BorrowOrigin struct {
	Function string
	Block    string
	Stmt     int
	Source   string // Human-readable source location
}

// BorrowRegion represents the region where a borrow is active
type BorrowRegion struct {
	Start BorrowPoint // Where borrow becomes active
	End   BorrowPoint // Where borrow expires
	Kind  RegionKind  // Type of region
}

// BorrowPoint represents a specific point in the control flow
type BorrowPoint struct {
	Function string
	Block    string
	Stmt     int // Statement index within the block
}

func (bp BorrowPoint) String() string {
	return fmt.Sprintf("%s::%s[%d]", bp.Function, bp.Block, bp.Stmt)
}

// RegionKind classifies different types of borrow regions
type RegionKind int

const (
	RegionLocal  RegionKind = iota // Local scope region
	RegionParam                    // Parameter region
	RegionReturn                   // Return value region
	RegionTemp                     // Temporary expression region
)

// ====== Borrow Constraints ======

// BorrowConstraint represents a constraint on borrows
type BorrowConstraint struct {
	Kind   BorrowConstraintKind
	Borrow BorrowID
	Target Value
	Reason string
}

// BorrowConstraintKind represents different types of borrow constraints
type BorrowConstraintKind int

const (
	ConstraintNoMove         BorrowConstraintKind = iota // Value cannot be moved while borrowed
	ConstraintNoMutate                                   // Value cannot be mutated during immutable borrow
	ConstraintExclusive                                  // Mutable borrow must be exclusive
	ConstraintBorrowOutlives                             // Borrow must outlive certain points
)

func (bck BorrowConstraintKind) String() string {
	switch bck {
	case ConstraintNoMove:
		return "no_move"
	case ConstraintNoMutate:
		return "no_mutate"
	case ConstraintExclusive:
		return "exclusive"
	case ConstraintBorrowOutlives:
		return "outlives"
	default:
		return "unknown"
	}
}

// ====== Borrow Checker ======

// BorrowChecker performs borrow checking for MIR
type BorrowChecker struct {
	lifetimeManager *LifetimeManager
	borrows         map[BorrowID]*Borrow
	activeBarrows   map[Value][]*Borrow // Value -> active borrows
	borrowCounter   int
	errors          []error
}

// NewBorrowChecker creates a new borrow checker
func NewBorrowChecker(lm *LifetimeManager) *BorrowChecker {
	return &BorrowChecker{
		lifetimeManager: lm,
		borrows:         make(map[BorrowID]*Borrow),
		activeBarrows:   make(map[Value][]*Borrow),
		borrowCounter:   0,
		errors:          make([]error, 0),
	}
}

// GenerateBorrowID generates a unique borrow ID
func (bc *BorrowChecker) GenerateBorrowID() BorrowID {
	bc.borrowCounter++
	return BorrowID(fmt.Sprintf("borrow_%d", bc.borrowCounter))
}

// ====== Borrow Tracking ======

// CreateBorrow creates and tracks a new borrow
func (bc *BorrowChecker) CreateBorrow(kind BorrowKind, borrowed, borrower Value, lifetime LifetimeID, origin BorrowOrigin) *Borrow {
	id := bc.GenerateBorrowID()

	borrow := &Borrow{
		ID:          id,
		Kind:        kind,
		Borrowed:    borrowed,
		Borrower:    borrower,
		Lifetime:    lifetime,
		Origin:      origin,
		Constraints: make([]*BorrowConstraint, 0),
	}

	bc.borrows[id] = borrow

	// Track active borrows
	if _, exists := bc.activeBarrows[borrowed]; !exists {
		bc.activeBarrows[borrowed] = make([]*Borrow, 0)
	}
	bc.activeBarrows[borrowed] = append(bc.activeBarrows[borrowed], borrow)

	return borrow
}

// GetBorrow retrieves a borrow by ID
func (bc *BorrowChecker) GetBorrow(id BorrowID) (*Borrow, bool) {
	borrow, exists := bc.borrows[id]
	return borrow, exists
}

// GetActiveBorrows returns all active borrows for a value
func (bc *BorrowChecker) GetActiveBorrows(value Value) []*Borrow {
	if borrows, exists := bc.activeBarrows[value]; exists {
		return borrows
	}
	return make([]*Borrow, 0)
}

// ====== Borrow Validation ======

// CheckFunction performs borrow checking for an entire function
func (bc *BorrowChecker) CheckFunction(function *Function) error {
	if function == nil {
		return fmt.Errorf("cannot check borrows for nil function")
	}

	// Check each block
	for _, block := range function.Blocks {
		if err := bc.checkBlock(block, function.Name); err != nil {
			return fmt.Errorf("borrow check failed in block %s: %v", block.Name, err)
		}
	}

	return nil
}

// checkBlock performs borrow checking for a basic block
func (bc *BorrowChecker) checkBlock(block *BasicBlock, functionName string) error {
	for i, instr := range block.Instr {
		point := BorrowPoint{
			Function: functionName,
			Block:    block.Name,
			Stmt:     i,
		}

		if err := bc.checkInstruction(instr, point); err != nil {
			bc.errors = append(bc.errors, err)
		}
	}
	return nil
}

// checkInstruction performs borrow checking for a single instruction
func (bc *BorrowChecker) checkInstruction(instr Instr, point BorrowPoint) error {
	switch inst := instr.(type) {
	case Load:
		return bc.checkLoad(inst, point)
	case Store:
		return bc.checkStore(inst, point)
	case Call:
		return bc.checkCall(inst, point)
	case BinOp:
		return bc.checkBinOp(inst, point)
	default:
		// Other instructions don't directly affect borrows
		return nil
	}
}

// checkLoad checks borrow rules for load instructions
func (bc *BorrowChecker) checkLoad(load Load, point BorrowPoint) error {
	// Check if the loaded value has any active mutable borrows
	activeBorrows := bc.GetActiveBorrows(load.Addr)

	for _, borrow := range activeBorrows {
		if borrow.Kind == BorrowMutable {
			return fmt.Errorf("cannot load from %s: value is mutably borrowed at %s",
				load.Addr.Ref, point)
		}
	}

	return nil
}

// checkStore checks borrow rules for store instructions
func (bc *BorrowChecker) checkStore(store Store, point BorrowPoint) error {
	// Check if the stored-to location has any active borrows
	activeBorrows := bc.GetActiveBorrows(store.Addr)

	for _, borrow := range activeBorrows {
		if borrow.Kind == BorrowImmutable {
			return fmt.Errorf("cannot store to %s: value is immutably borrowed at %s",
				store.Addr.Ref, point)
		}
		if borrow.Kind == BorrowMutable {
			return fmt.Errorf("cannot store to %s: value is already mutably borrowed at %s",
				store.Addr.Ref, point)
		}
	}

	return nil
}

// checkCall checks borrow rules for function calls
func (bc *BorrowChecker) checkCall(call Call, point BorrowPoint) error {
	// Check each argument for borrow conflicts
	for i, arg := range call.Args {
		if err := bc.checkCallArgument(arg, i, point); err != nil {
			return err
		}
	}
	return nil
}

// checkCallArgument checks borrow rules for a function call argument
func (bc *BorrowChecker) checkCallArgument(arg Value, argIndex int, point BorrowPoint) error {
	// If the argument is a borrowed value, check for conflicts
	activeBorrows := bc.GetActiveBorrows(arg)

	// For now, assume all function arguments can potentially be borrowed
	// More sophisticated analysis would check function signatures
	for _, borrow := range activeBorrows {
		if borrow.Kind == BorrowMutable {
			// Mutable borrows generally cannot be passed to functions
			// unless the function signature explicitly allows it
			return fmt.Errorf("cannot pass mutably borrowed value %s to function at %s",
				arg.Ref, point)
		}
	}

	return nil
}

// checkBinOp checks borrow rules for binary operations
func (bc *BorrowChecker) checkBinOp(binop BinOp, point BorrowPoint) error {
	// Check both operands for borrow conflicts
	if err := bc.checkValueUsage(binop.LHS, point); err != nil {
		return err
	}
	if err := bc.checkValueUsage(binop.RHS, point); err != nil {
		return err
	}
	return nil
}

// checkValueUsage checks if a value can be used at a given point
func (bc *BorrowChecker) checkValueUsage(value Value, point BorrowPoint) error {
	activeBorrows := bc.GetActiveBorrows(value)

	for _, borrow := range activeBorrows {
		// Check if the borrow is still active at this point
		if bc.isBorrowActiveAt(borrow, point) {
			// Value is borrowed, check if this usage is allowed
			if borrow.Kind == BorrowMutable {
				return fmt.Errorf("cannot use value %s: mutably borrowed at %s", value.Ref, point)
			}
		}
	}

	return nil
}

// isBorrowActiveAt checks if a borrow is active at a given point
func (bc *BorrowChecker) isBorrowActiveAt(borrow *Borrow, point BorrowPoint) bool {
	if borrow.Region == nil {
		// If no region specified, assume it's active
		return true
	}

	// Check if point is within the borrow region
	// This is a simplified check - real implementation would need
	// more sophisticated control flow analysis
	return bc.isPointInRegion(point, borrow.Region)
}

// isPointInRegion checks if a point is within a borrow region
func (bc *BorrowChecker) isPointInRegion(point BorrowPoint, region *BorrowRegion) bool {
	// Simplified check: same function and block, statement in range
	if point.Function != region.Start.Function {
		return false
	}
	if point.Block != region.Start.Block {
		return false
	}

	return point.Stmt >= region.Start.Stmt && point.Stmt <= region.End.Stmt
}

// ====== Borrow Rules Enforcement ======

// ValidateBorrowRules validates all borrow rules for a module
func (bc *BorrowChecker) ValidateBorrowRules(module *Module) error {
	if module == nil {
		return fmt.Errorf("cannot validate borrow rules for nil module")
	}

	// Check each function
	for _, function := range module.Functions {
		if err := bc.CheckFunction(function); err != nil {
			return fmt.Errorf("borrow validation failed for function %s: %v", function.Name, err)
		}
	}

	if len(bc.errors) > 0 {
		return fmt.Errorf("borrow checking failed with %d errors", len(bc.errors))
	}

	return nil
}

// CheckBorrowRules checks the core borrow checker rules
func (bc *BorrowChecker) CheckBorrowRules() error {
	for _, borrow := range bc.borrows {
		if err := bc.validateBorrow(borrow); err != nil {
			bc.errors = append(bc.errors, err)
		}
	}

	if len(bc.errors) > 0 {
		return fmt.Errorf("borrow rule validation failed")
	}

	return nil
}

// validateBorrow validates a single borrow against all rules
func (bc *BorrowChecker) validateBorrow(borrow *Borrow) error {
	// Rule 1: At any given time, you can have either one mutable reference
	// or any number of immutable references
	if err := bc.checkExclusiveMutableBorrow(borrow); err != nil {
		return err
	}

	// Rule 2: References must always be valid (lifetime checking)
	if err := bc.checkBorrowLifetime(borrow); err != nil {
		return err
	}

	return nil
}

// checkExclusiveMutableBorrow ensures mutable borrows are exclusive
func (bc *BorrowChecker) checkExclusiveMutableBorrow(borrow *Borrow) error {
	if borrow.Kind != BorrowMutable {
		return nil // Only applies to mutable borrows
	}

	activeBorrows := bc.GetActiveBorrows(borrow.Borrowed)

	for _, other := range activeBorrows {
		if other.ID != borrow.ID {
			// Another borrow exists for the same value
			return fmt.Errorf("mutable borrow %s conflicts with existing borrow %s",
				borrow.ID, other.ID)
		}
	}

	return nil
}

// checkBorrowLifetime ensures borrow lifetime is valid
func (bc *BorrowChecker) checkBorrowLifetime(borrow *Borrow) error {
	// Check with lifetime manager
	if bc.lifetimeManager != nil {
		lifetime, exists := bc.lifetimeManager.GetLifetime(borrow.Lifetime)
		if !exists {
			return fmt.Errorf("borrow %s references unknown lifetime %s",
				borrow.ID, borrow.Lifetime)
		}

		// Ensure lifetime is valid for the borrow's usage
		if lifetime.Kind == LifetimeTemp && borrow.Kind == BorrowMutable {
			return fmt.Errorf("mutable borrow %s cannot use temporary lifetime %s",
				borrow.ID, borrow.Lifetime)
		}
	}

	return nil
}

// ====== Error Management ======

// GetErrors returns all accumulated errors
func (bc *BorrowChecker) GetErrors() []error {
	return bc.errors
}

// ClearErrors clears all accumulated errors
func (bc *BorrowChecker) ClearErrors() {
	bc.errors = make([]error, 0)
}

// ====== Debug and Reporting ======

// String returns a string representation of the borrow checker
func (bc *BorrowChecker) String() string {
	var b strings.Builder
	b.WriteString("BorrowChecker {\n")

	b.WriteString("  Borrows:\n")
	for id, borrow := range bc.borrows {
		b.WriteString(fmt.Sprintf("    %s: %s %s -> %s (lifetime: %s)\n",
			id, borrow.Kind, borrow.Borrowed.Ref, borrow.Borrower.Ref, borrow.Lifetime))
	}

	b.WriteString("  Active Borrows:\n")
	for value, borrows := range bc.activeBarrows {
		b.WriteString(fmt.Sprintf("    %s: [", value.Ref))
		for i, borrow := range borrows {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(string(borrow.ID))
		}
		b.WriteString("]\n")
	}

	b.WriteString("}\n")
	return b.String()
}

// ====== Integration with HIR-to-MIR Transformer ======

// AnalyzeBorrows performs borrow analysis during HIR-to-MIR transformation
func (bc *BorrowChecker) AnalyzeBorrows(instr Instr, point BorrowPoint) error {
	// Automatically detect and track borrows during transformation
	switch inst := instr.(type) {
	case Load:
		// Load operations might create implicit borrows
		return bc.analyzeBorrowFromLoad(inst, point)
	case Store:
		// Store operations affect borrow validity
		return bc.analyzeBorrowFromStore(inst, point)
	case Call:
		// Function calls might create or invalidate borrows
		return bc.analyzeBorrowFromCall(inst, point)
	default:
		return nil
	}
}

// analyzeBorrowFromLoad analyzes borrows created by load operations
func (bc *BorrowChecker) analyzeBorrowFromLoad(load Load, point BorrowPoint) error {
	// Create an implicit immutable borrow for the loaded value
	origin := BorrowOrigin{
		Function: point.Function,
		Block:    point.Block,
		Stmt:     point.Stmt,
		Source:   fmt.Sprintf("load at %s", point),
	}

	// For now, create a temporary lifetime
	lifetimeID := bc.lifetimeManager.GenerateLifetimeID()

	bc.CreateBorrow(BorrowImmutable, load.Addr, Value{Kind: ValRef, Ref: "temp_ref"}, lifetimeID, origin)

	return nil
}

// analyzeBorrowFromStore analyzes borrows affected by store operations
func (bc *BorrowChecker) analyzeBorrowFromStore(store Store, point BorrowPoint) error {
	// Store operations might invalidate existing borrows
	activeBorrows := bc.GetActiveBorrows(store.Addr)

	for _, borrow := range activeBorrows {
		if borrow.Kind == BorrowImmutable {
			// Invalidate immutable borrows when storing
			bc.invalidateBorrow(borrow, point)
		}
	}

	return nil
}

// analyzeBorrowFromCall analyzes borrows created or affected by function calls
func (bc *BorrowChecker) analyzeBorrowFromCall(call Call, point BorrowPoint) error {
	// Function calls might create borrows through their arguments
	for i, arg := range call.Args {
		if arg.Kind == ValRef {
			// Argument might be borrowed by the function
			origin := BorrowOrigin{
				Function: point.Function,
				Block:    point.Block,
				Stmt:     point.Stmt,
				Source:   fmt.Sprintf("call arg %d at %s", i, point),
			}

			lifetimeID := bc.lifetimeManager.GenerateLifetimeID()
			bc.CreateBorrow(BorrowImmutable, arg, Value{Kind: ValRef, Ref: "call_borrow"}, lifetimeID, origin)
		}
	}

	return nil
}

// invalidateBorrow marks a borrow as invalid/expired
func (bc *BorrowChecker) invalidateBorrow(borrow *Borrow, point BorrowPoint) {
	// Remove from active borrows
	if borrows, exists := bc.activeBarrows[borrow.Borrowed]; exists {
		filtered := make([]*Borrow, 0)
		for _, b := range borrows {
			if b.ID != borrow.ID {
				filtered = append(filtered, b)
			}
		}
		bc.activeBarrows[borrow.Borrowed] = filtered
	}
}
