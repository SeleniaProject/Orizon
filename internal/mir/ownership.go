// Ownership system for Orizon MIR
// This file implements ownership tracking and validation to ensure
// memory safety through compile-time ownership checking. It provides:
// 1. Move semantics and ownership transfer tracking
// 2. Copy vs Move trait integration
// 3. Drop trait and destructor management
// 4. Ownership-based access control

package mir

import (
	"fmt"
	"strings"
)

// ====== Ownership Core Types ======

// OwnershipKind represents different types of ownership
type OwnershipKind int

const (
	OwnershipOwned    OwnershipKind = iota // T - owned value
	OwnershipBorrowed                      // &T - borrowed reference
	OwnershipMutable                       // &mut T - mutable reference
	OwnershipShared                        // Arc<T> - shared ownership
	OwnershipWeak                          // Weak<T> - weak reference
)

func (ok OwnershipKind) String() string {
	switch ok {
	case OwnershipOwned:
		return "owned"
	case OwnershipBorrowed:
		return "borrowed"
	case OwnershipMutable:
		return "mut_borrowed"
	case OwnershipShared:
		return "shared"
	case OwnershipWeak:
		return "weak"
	default:
		return "unknown"
	}
}

// Ownership represents ownership information for a value
type Ownership struct {
	ID       OwnershipID     // Unique identifier
	Kind     OwnershipKind   // Type of ownership
	Owner    Value           // The owning value
	Owned    Value           // The owned value
	Lifetime LifetimeID      // Associated lifetime
	Traits   OwnershipTraits // Traits affecting ownership
	State    OwnershipState  // Current state
	Origin   OwnershipOrigin // Where ownership was established
}

// OwnershipID represents a unique ownership identifier
type OwnershipID string

// OwnershipTraits represents traits that affect ownership behavior
type OwnershipTraits struct {
	Copy  bool // Type implements Copy
	Clone bool // Type implements Clone
	Drop  bool // Type implements Drop
	Send  bool // Type is Send (thread-safe to move)
	Sync  bool // Type is Sync (thread-safe to share)
	Unpin bool // Type is Unpin (safe to move in memory)
	Sized bool // Type has known size at compile time
}

// OwnershipState tracks the current state of ownership
type OwnershipState int

const (
	StateUninitialized OwnershipState = iota // Value not yet initialized
	StateOwned                               // Value is owned
	StateMoved                               // Value has been moved
	StateBorrowed                            // Value is currently borrowed
	StateDropped                             // Value has been dropped
	StateInvalid                             // Value is in invalid state
)

func (os OwnershipState) String() string {
	switch os {
	case StateUninitialized:
		return "uninitialized"
	case StateOwned:
		return "owned"
	case StateMoved:
		return "moved"
	case StateBorrowed:
		return "borrowed"
	case StateDropped:
		return "dropped"
	case StateInvalid:
		return "invalid"
	default:
		return "unknown"
	}
}

// OwnershipOrigin tracks where ownership was established
type OwnershipOrigin struct {
	Kind     OwnershipOriginKind
	Function string
	Block    string
	Stmt     int
	Source   string
}

// OwnershipOriginKind classifies ownership origins
type OwnershipOriginKind int

const (
	OwnershipOriginAllocation OwnershipOriginKind = iota // Value allocated
	OwnershipOriginParameter                             // Function parameter
	OwnershipOriginReturn                                // Function return
	OwnershipOriginMove                                  // Value moved
	OwnershipOriginCopy                                  // Value copied
	OwnershipOriginBorrow                                // Value borrowed
)

// ====== Move Operations ======

// MoveOperation represents a move operation
type MoveOperation struct {
	ID        MoveID      // Unique identifier
	From      Value       // Source value being moved
	To        Value       // Destination value
	Kind      MoveKind    // Type of move
	Point     MovePoint   // Where the move occurs
	Ownership OwnershipID // Associated ownership
}

// MoveID represents a unique move operation identifier
type MoveID string

// MoveKind represents different types of moves
type MoveKind int

const (
	MoveExplicit MoveKind = iota // Explicit move (move(x))
	MoveImplicit                 // Implicit move (assignment)
	MoveReturn                   // Return value move
	MoveCall                     // Function call argument move
	MoveDrop                     // Move for drop
)

func (mk MoveKind) String() string {
	switch mk {
	case MoveExplicit:
		return "explicit"
	case MoveImplicit:
		return "implicit"
	case MoveReturn:
		return "return"
	case MoveCall:
		return "call"
	case MoveDrop:
		return "drop"
	default:
		return "unknown"
	}
}

// MovePoint represents where a move operation occurs
type MovePoint struct {
	Function string
	Block    string
	Stmt     int
}

func (mp MovePoint) String() string {
	return fmt.Sprintf("%s::%s[%d]", mp.Function, mp.Block, mp.Stmt)
}

// ====== Ownership Manager ======

// OwnershipManager manages ownership tracking and validation
type OwnershipManager struct {
	ownerships    map[OwnershipID]*Ownership
	moveOps       map[MoveID]*MoveOperation
	valueStates   map[Value]OwnershipState // Current state of each value
	borrowChecker *BorrowChecker           // Integration with borrow checker
	counter       int                      // For generating unique IDs
	errors        []error
}

// NewOwnershipManager creates a new ownership manager
func NewOwnershipManager(bc *BorrowChecker) *OwnershipManager {
	return &OwnershipManager{
		ownerships:    make(map[OwnershipID]*Ownership),
		moveOps:       make(map[MoveID]*MoveOperation),
		valueStates:   make(map[Value]OwnershipState),
		borrowChecker: bc,
		counter:       0,
		errors:        make([]error, 0),
	}
}

// GenerateOwnershipID generates a unique ownership ID
func (om *OwnershipManager) GenerateOwnershipID() OwnershipID {
	om.counter++
	return OwnershipID(fmt.Sprintf("own_%d", om.counter))
}

// GenerateMoveID generates a unique move ID
func (om *OwnershipManager) GenerateMoveID() MoveID {
	om.counter++
	return MoveID(fmt.Sprintf("move_%d", om.counter))
}

// ====== Ownership Tracking ======

// CreateOwnership creates and tracks new ownership
func (om *OwnershipManager) CreateOwnership(kind OwnershipKind, owner, owned Value, lifetime LifetimeID, traits OwnershipTraits, origin OwnershipOrigin) *Ownership {
	id := om.GenerateOwnershipID()

	ownership := &Ownership{
		ID:       id,
		Kind:     kind,
		Owner:    owner,
		Owned:    owned,
		Lifetime: lifetime,
		Traits:   traits,
		State:    StateOwned,
		Origin:   origin,
	}

	om.ownerships[id] = ownership
	om.valueStates[owned] = StateOwned

	return ownership
}

// GetOwnership retrieves ownership by ID
func (om *OwnershipManager) GetOwnership(id OwnershipID) (*Ownership, bool) {
	ownership, exists := om.ownerships[id]
	return ownership, exists
}

// GetValueState returns the current ownership state of a value
func (om *OwnershipManager) GetValueState(value Value) OwnershipState {
	if state, exists := om.valueStates[value]; exists {
		return state
	}
	return StateUninitialized
}

// SetValueState sets the ownership state of a value
func (om *OwnershipManager) SetValueState(value Value, state OwnershipState) {
	om.valueStates[value] = state
}

// ====== Move Operations ======

// CreateMove creates and tracks a move operation
func (om *OwnershipManager) CreateMove(kind MoveKind, from, to Value, point MovePoint, ownership OwnershipID) *MoveOperation {
	move := om.createMoveWithoutStateChange(kind, from, to, point, ownership)

	// Update value states
	om.SetValueState(from, StateMoved)
	if to.Ref != "" { // Only set to owned if destination is not empty
		om.SetValueState(to, StateOwned)
	}

	return move
}

// createMoveWithoutStateChange creates a move operation without changing value states
func (om *OwnershipManager) createMoveWithoutStateChange(kind MoveKind, from, to Value, point MovePoint, ownership OwnershipID) *MoveOperation {
	id := om.GenerateMoveID()

	move := &MoveOperation{
		ID:        id,
		From:      from,
		To:        to,
		Kind:      kind,
		Point:     point,
		Ownership: ownership,
	}

	om.moveOps[id] = move
	return move
}

// GetMove retrieves a move operation by ID
func (om *OwnershipManager) GetMove(id MoveID) (*MoveOperation, bool) {
	move, exists := om.moveOps[id]
	return move, exists
}

// ====== Ownership Validation ======

// ValidateOwnership performs comprehensive ownership validation
func (om *OwnershipManager) ValidateOwnership(module *Module) error {
	if module == nil {
		return fmt.Errorf("cannot validate ownership for nil module")
	}

	// Validate each function
	for _, function := range module.Functions {
		if err := om.validateFunction(function); err != nil {
			return fmt.Errorf("ownership validation failed for function %s: %v", function.Name, err)
		}
	}

	if len(om.errors) > 0 {
		return fmt.Errorf("ownership validation failed with %d errors", len(om.errors))
	}

	return nil
}

// validateFunction validates ownership for a single function
func (om *OwnershipManager) validateFunction(function *Function) error {
	// Validate each block
	for _, block := range function.Blocks {
		if err := om.validateBlock(block, function.Name); err != nil {
			return err
		}
	}
	return nil
}

// validateBlock validates ownership for a basic block
func (om *OwnershipManager) validateBlock(block *BasicBlock, functionName string) error {
	for i, instr := range block.Instr {
		point := MovePoint{
			Function: functionName,
			Block:    block.Name,
			Stmt:     i,
		}

		if err := om.validateInstruction(instr, point); err != nil {
			om.errors = append(om.errors, err)
		}
	}
	return nil
}

// validateInstruction validates ownership for a single instruction
func (om *OwnershipManager) validateInstruction(instr Instr, point MovePoint) error {
	switch inst := instr.(type) {
	case Alloca:
		return om.validateAlloca(inst, point)
	case Load:
		return om.validateLoad(inst, point)
	case Store:
		return om.validateStore(inst, point)
	case Call:
		return om.validateCall(inst, point)
	case BinOp:
		return om.validateBinOp(inst, point)
	default:
		return nil
	}
}

// validateAlloca validates ownership for allocation instructions
func (om *OwnershipManager) validateAlloca(alloca Alloca, point MovePoint) error {
	// Allocations create new ownership
	addr := Value{Kind: ValRef, Ref: alloca.Dst}

	origin := OwnershipOrigin{
		Kind:     OwnershipOriginAllocation,
		Function: point.Function,
		Block:    point.Block,
		Stmt:     point.Stmt,
		Source:   fmt.Sprintf("alloca %s", alloca.Name),
	}

	// Default traits for allocated values
	traits := OwnershipTraits{
		Copy:  false, // Default to non-copyable
		Clone: true,  // Most types can be cloned
		Drop:  true,  // Most types need dropping
		Send:  true,  // Default to Send
		Sync:  false, // Default to not Sync
		Unpin: true,  // Default to Unpin
		Sized: true,  // Stack allocations are sized
	}

	om.CreateOwnership(OwnershipOwned, addr, addr, "", traits, origin)
	return nil
}

// validateLoad validates ownership for load instructions
func (om *OwnershipManager) validateLoad(load Load, point MovePoint) error {
	// Check if the loaded value can be accessed
	state := om.GetValueState(load.Addr)

	switch state {
	case StateMoved:
		return fmt.Errorf("cannot load from moved value %s at %s", load.Addr.Ref, point)
	case StateDropped:
		return fmt.Errorf("cannot load from dropped value %s at %s", load.Addr.Ref, point)
	case StateInvalid:
		return fmt.Errorf("cannot load from invalid value %s at %s", load.Addr.Ref, point)
	}

	return nil
}

// validateStore validates ownership for store instructions
func (om *OwnershipManager) validateStore(store Store, point MovePoint) error {
	// Check if the store destination can be written to
	destState := om.GetValueState(store.Addr)

	switch destState {
	case StateMoved:
		return fmt.Errorf("cannot store to moved value %s at %s", store.Addr.Ref, point)
	case StateDropped:
		return fmt.Errorf("cannot store to dropped value %s at %s", store.Addr.Ref, point)
	case StateInvalid:
		return fmt.Errorf("cannot store to invalid value %s at %s", store.Addr.Ref, point)
	}

	// Check if the stored value can be moved/copied
	srcState := om.GetValueState(store.Val)

	switch srcState {
	case StateMoved:
		return fmt.Errorf("cannot store moved value %s at %s", store.Val.Ref, point)
	case StateDropped:
		return fmt.Errorf("cannot store dropped value %s at %s", store.Val.Ref, point)
	}

	return nil
}

// validateCall validates ownership for function calls
func (om *OwnershipManager) validateCall(call Call, point MovePoint) error {
	// Check each argument
	for i, arg := range call.Args {
		if err := om.validateCallArgument(arg, i, point); err != nil {
			return err
		}
	}
	return nil
}

// validateCallArgument validates ownership for a call argument
func (om *OwnershipManager) validateCallArgument(arg Value, argIndex int, point MovePoint) error {
	state := om.GetValueState(arg)

	switch state {
	case StateMoved:
		return fmt.Errorf("cannot pass moved value %s as argument %d at %s",
			arg.Ref, argIndex, point)
	case StateDropped:
		return fmt.Errorf("cannot pass dropped value %s as argument %d at %s",
			arg.Ref, argIndex, point)
	case StateInvalid:
		return fmt.Errorf("cannot pass invalid value %s as argument %d at %s",
			arg.Ref, argIndex, point)
	}

	// For now, assume all arguments are moved (more sophisticated analysis needed)
	if state == StateOwned {
		om.SetValueState(arg, StateMoved)
	}

	return nil
}

// validateBinOp validates ownership for binary operations
func (om *OwnershipManager) validateBinOp(binop BinOp, point MovePoint) error {
	// Check both operands
	if err := om.validateValueAccess(binop.LHS, point); err != nil {
		return err
	}
	if err := om.validateValueAccess(binop.RHS, point); err != nil {
		return err
	}
	return nil
}

// validateValueAccess validates that a value can be accessed
func (om *OwnershipManager) validateValueAccess(value Value, point MovePoint) error {
	state := om.GetValueState(value)

	switch state {
	case StateMoved:
		return fmt.Errorf("cannot access moved value %s at %s", value.Ref, point)
	case StateDropped:
		return fmt.Errorf("cannot access dropped value %s at %s", value.Ref, point)
	case StateInvalid:
		return fmt.Errorf("cannot access invalid value %s at %s", value.Ref, point)
	}

	return nil
}

// ====== Copy vs Move Logic ======

// CanCopy checks if a value can be copied instead of moved
func (om *OwnershipManager) CanCopy(value Value) bool {
	// Find ownership for this value
	for _, ownership := range om.ownerships {
		if ownership.Owned.Ref == value.Ref {
			return ownership.Traits.Copy
		}
	}

	// Default to non-copyable
	return false
}

// ShouldMove determines if a value should be moved
func (om *OwnershipManager) ShouldMove(value Value, context string) bool {
	// If value can be copied and it's a small context, prefer copy
	if om.CanCopy(value) {
		switch context {
		case "assignment", "function_arg":
			return false // Prefer copy for these contexts
		case "return", "explicit_move":
			return true // Prefer move for these contexts
		}
	}

	// Default to move
	return true
}

// ====== Drop and Destruction ======

// CreateDrop creates a drop operation for a value
func (om *OwnershipManager) CreateDrop(value Value, point MovePoint) error {
	state := om.GetValueState(value)

	if state == StateDropped {
		return fmt.Errorf("value %s already dropped at %s", value.Ref, point)
	}

	if state == StateMoved {
		return fmt.Errorf("cannot drop moved value %s at %s", value.Ref, point)
	}

	// Create drop move operation first (before changing state)
	dropMove := om.createMoveWithoutStateChange(MoveDrop, value, Value{}, point, "")
	_ = dropMove // Use the created move

	// Mark as dropped (after creating move to avoid state conflict)
	om.SetValueState(value, StateDropped)

	return nil
}

// NeedsDrop checks if a value needs to be explicitly dropped
func (om *OwnershipManager) NeedsDrop(value Value) bool {
	for _, ownership := range om.ownerships {
		if ownership.Owned.Ref == value.Ref {
			return ownership.Traits.Drop
		}
	}
	return false
}

// ====== Error Management ======

// GetErrors returns all accumulated errors
func (om *OwnershipManager) GetErrors() []error {
	return om.errors
}

// ClearErrors clears all accumulated errors
func (om *OwnershipManager) ClearErrors() {
	om.errors = make([]error, 0)
}

// ====== Debug and Reporting ======

// String returns a string representation of the ownership manager
func (om *OwnershipManager) String() string {
	var b strings.Builder
	b.WriteString("OwnershipManager {\n")

	b.WriteString("  Ownerships:\n")
	for id, ownership := range om.ownerships {
		b.WriteString(fmt.Sprintf("    %s: %s %s -> %s (state: %s)\n",
			id, ownership.Kind, ownership.Owner.Ref, ownership.Owned.Ref, ownership.State))
	}

	b.WriteString("  Value States:\n")
	for value, state := range om.valueStates {
		b.WriteString(fmt.Sprintf("    %s: %s\n", value.Ref, state))
	}

	b.WriteString("  Move Operations:\n")
	for id, move := range om.moveOps {
		b.WriteString(fmt.Sprintf("    %s: %s %s -> %s at %s\n",
			id, move.Kind, move.From.Ref, move.To.Ref, move.Point))
	}

	b.WriteString("}\n")
	return b.String()
}
