// Tests for the ownership and borrow checking system.
// This file tests the integration of lifetime management, borrow checking,.
// and ownership tracking for memory safety validation.

package mir

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/parser"
)

func TestLifetimeManager_Basic(t *testing.T) {
	lm := NewLifetimeManager()

	// Test static lifetime exists.
	static, exists := lm.GetLifetime(StaticLifetime)
	if !exists {
		t.Fatal("Static lifetime should exist by default")
	}

	if static.Kind != LifetimeStatic {
		t.Errorf("Expected LifetimeStatic, got %v", static.Kind)
	}

	// Test creating named lifetime.
	scope := lm.CreateScope("test_scope", "test_func", "test_block", nil)
	origin := LifetimeOrigin{Kind: OriginParameter, Function: "test_func"}

	lifetime := lm.CreateNamedLifetime("a", scope, origin)
	if lifetime.ID != "'a" {
		t.Errorf("Expected lifetime ID 'a, got %s", lifetime.ID)
	}

	if lifetime.Kind != LifetimeNamed {
		t.Errorf("Expected LifetimeNamed, got %v", lifetime.Kind)
	}

	// Test retrieving lifetime.
	retrieved, exists := lm.GetLifetime("'a")
	if !exists {
		t.Fatal("Named lifetime 'a should exist")
	}

	if retrieved.ID != lifetime.ID {
		t.Errorf("Retrieved lifetime doesn't match created lifetime")
	}
}

func TestLifetimeManager_Constraints(t *testing.T) {
	lm := NewLifetimeManager()

	// Create two lifetimes.
	scope := lm.CreateScope("test_scope", "test_func", "test_block", nil)
	origin := LifetimeOrigin{Kind: OriginParameter, Function: "test_func"}

	lt1 := lm.CreateNamedLifetime("a", scope, origin)
	lt2 := lm.CreateNamedLifetime("b", scope, origin)

	// Add outlives constraint: 'a: 'b.
	lm.AddOutlivesConstraint(lt1.ID, lt2.ID, "test constraint")

	if len(lm.constraints) != 1 {
		t.Errorf("Expected 1 constraint, got %d", len(lm.constraints))
	}

	constraint := lm.constraints[0]
	if constraint.Kind != ConstraintOutlives {
		t.Errorf("Expected ConstraintOutlives, got %v", constraint.Kind)
	}

	if constraint.From != lt1.ID {
		t.Errorf("Expected constraint from %s, got %s", lt1.ID, constraint.From)
	}

	if constraint.To != lt2.ID {
		t.Errorf("Expected constraint to %s, got %s", lt2.ID, constraint.To)
	}
}

func TestBorrowChecker_Basic(t *testing.T) {
	lm := NewLifetimeManager()
	bc := NewBorrowChecker(lm)

	// Create test values.
	borrowed := Value{Kind: ValRef, Ref: "%x"}
	borrower := Value{Kind: ValRef, Ref: "%ref_x"}
	lifetime := lm.GenerateLifetimeID()

	origin := BorrowOrigin{
		Function: "test_func",
		Block:    "entry",
		Stmt:     0,
		Source:   "test borrow",
	}

	// Create immutable borrow.
	borrow := bc.CreateBorrow(BorrowImmutable, borrowed, borrower, lifetime, origin)

	if borrow.Kind != BorrowImmutable {
		t.Errorf("Expected BorrowImmutable, got %v", borrow.Kind)
	}

	if borrow.Borrowed.Ref != borrowed.Ref {
		t.Errorf("Expected borrowed %s, got %s", borrowed.Ref, borrow.Borrowed.Ref)
	}

	// Check active borrows.
	activeBorrows := bc.GetActiveBorrows(borrowed)
	if len(activeBorrows) != 1 {
		t.Errorf("Expected 1 active borrow, got %d", len(activeBorrows))
	}

	if activeBorrows[0].ID != borrow.ID {
		t.Errorf("Active borrow ID doesn't match")
	}
}

func TestBorrowChecker_MutableBorrowExclusion(t *testing.T) {
	lm := NewLifetimeManager()
	bc := NewBorrowChecker(lm)

	borrowed := Value{Kind: ValRef, Ref: "%x"}
	borrower1 := Value{Kind: ValRef, Ref: "%ref1_x"}
	borrower2 := Value{Kind: ValRef, Ref: "%ref2_x"}
	lifetime := lm.GenerateLifetimeID()

	origin := BorrowOrigin{
		Function: "test_func",
		Block:    "entry",
		Stmt:     0,
		Source:   "test borrow",
	}

	// Create first mutable borrow.
	borrow1 := bc.CreateBorrow(BorrowMutable, borrowed, borrower1, lifetime, origin)

	// Create second mutable borrow (should conflict).
	borrow2 := bc.CreateBorrow(BorrowMutable, borrowed, borrower2, lifetime, origin)

	// Check borrow rules.
	err := bc.CheckBorrowRules()
	if err == nil {
		t.Error("Expected error for conflicting mutable borrows")
	}

	// Check that both borrows exist but rules fail.
	if len(bc.borrows) != 2 {
		t.Errorf("Expected 2 borrows, got %d", len(bc.borrows))
	}

	_ = borrow1
	_ = borrow2
}

func TestOwnershipManager_Basic(t *testing.T) {
	lm := NewLifetimeManager()
	bc := NewBorrowChecker(lm)
	om := NewOwnershipManager(bc)

	// Create test values.
	owner := Value{Kind: ValRef, Ref: "%owner"}
	owned := Value{Kind: ValRef, Ref: "%owned"}
	lifetime := lm.GenerateLifetimeID()

	traits := OwnershipTraits{
		Copy:  false,
		Clone: true,
		Drop:  true,
		Send:  true,
		Sync:  false,
		Unpin: true,
		Sized: true,
	}

	origin := OwnershipOrigin{
		Kind:     OwnershipOriginAllocation,
		Function: "test_func",
		Block:    "entry",
		Stmt:     0,
		Source:   "test allocation",
	}

	// Create ownership.
	ownership := om.CreateOwnership(OwnershipOwned, owner, owned, lifetime, traits, origin)

	if ownership.Kind != OwnershipOwned {
		t.Errorf("Expected OwnershipOwned, got %v", ownership.Kind)
	}

	if ownership.Owner.Ref != owner.Ref {
		t.Errorf("Expected owner %s, got %s", owner.Ref, ownership.Owner.Ref)
	}

	// Check value state.
	state := om.GetValueState(owned)
	if state != StateOwned {
		t.Errorf("Expected StateOwned, got %v", state)
	}
}

func TestOwnershipManager_MoveOperations(t *testing.T) {
	lm := NewLifetimeManager()
	bc := NewBorrowChecker(lm)
	om := NewOwnershipManager(bc)

	from := Value{Kind: ValRef, Ref: "%from"}
	to := Value{Kind: ValRef, Ref: "%to"}

	point := MovePoint{
		Function: "test_func",
		Block:    "entry",
		Stmt:     1,
	}

	// Set initial state.
	om.SetValueState(from, StateOwned)

	// Create move operation.
	move := om.CreateMove(MoveImplicit, from, to, point, "")

	if move.Kind != MoveImplicit {
		t.Errorf("Expected MoveImplicit, got %v", move.Kind)
	}

	if move.From.Ref != from.Ref {
		t.Errorf("Expected move from %s, got %s", from.Ref, move.From.Ref)
	}

	// Check states after move.
	fromState := om.GetValueState(from)
	if fromState != StateMoved {
		t.Errorf("Expected StateMoved for source, got %v", fromState)
	}

	toState := om.GetValueState(to)
	if toState != StateOwned {
		t.Errorf("Expected StateOwned for destination, got %v", toState)
	}
}

func TestIntegratedOwnershipAndBorrows(t *testing.T) {
	// Create the integrated system.
	lm := NewLifetimeManager()
	bc := NewBorrowChecker(lm)
	om := NewOwnershipManager(bc)

	// Create a test MIR function.
	function := &Function{
		Name:       "test_function",
		Parameters: []Value{},
		Blocks: []*BasicBlock{
			{
				Name: "entry",
				Instr: []Instr{
					Alloca{Dst: "%x.addr", Name: "x"},
					Store{Addr: Value{Kind: ValRef, Ref: "%x.addr"}, Val: Value{Kind: ValConstInt, Int64: 42}},
					Load{Dst: "%temp", Addr: Value{Kind: ValRef, Ref: "%x.addr"}},
					Ret{Val: &Value{Kind: ValRef, Ref: "%temp"}},
				},
			},
		},
	}

	// Test lifetime inference.
	err := lm.InferLifetimes(function)
	if err != nil {
		t.Fatalf("Lifetime inference failed: %v", err)
	}

	// Test borrow checking.
	err = bc.CheckFunction(function)
	if err != nil {
		t.Fatalf("Borrow checking failed: %v", err)
	}

	// Test ownership validation.
	err = om.validateFunction(function)
	if err != nil {
		t.Fatalf("Ownership validation failed: %v", err)
	}

	t.Logf("Lifetime manager: %d lifetimes, %d constraints",
		len(lm.lifetimes), len(lm.constraints))
	t.Logf("Borrow checker: %d borrows", len(bc.borrows))
	t.Logf("Ownership manager: %d ownerships, %d moves",
		len(om.ownerships), len(om.moveOps))
}

func TestComprehensiveMemorySafety(t *testing.T) {
	// Test a more complex scenario with multiple borrows and ownership transfers.
	lm := NewLifetimeManager()
	bc := NewBorrowChecker(lm)
	om := NewOwnershipManager(bc)

	// Create test HIR module (simplified).
	hirModule := &parser.HIRModule{
		Name: "memory_safety_test",
		Functions: []*parser.HIRFunction{
			{
				Name:       "complex_test",
				Parameters: []*parser.HIRParameter{},
				Body: &parser.HIRBlock{
					Statements: []*parser.HIRStatement{
						{
							Kind: parser.HIRStmtReturn,
							Data: parser.HIRReturnStatement{
								Value: &parser.HIRExpression{
									Kind: parser.HIRExprLiteral,
									Data: parser.HIRLiteralExpression{
										Value: "42",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Transform to MIR.
	transformer := NewHIRToMIRTransformer()

	mirModule, err := transformer.TransformModule(hirModule)
	if err != nil {
		t.Fatalf("HIR to MIR transformation failed: %v", err)
	}

	// Validate with all systems.
	err = lm.ValidateLifetimes(mirModule)
	if err != nil {
		t.Fatalf("Lifetime validation failed: %v", err)
	}

	err = bc.ValidateBorrowRules(mirModule)
	if err != nil {
		t.Fatalf("Borrow rule validation failed: %v", err)
	}

	err = om.ValidateOwnership(mirModule)
	if err != nil {
		t.Fatalf("Ownership validation failed: %v", err)
	}

	t.Logf("Comprehensive memory safety validation passed for module: %s", mirModule.Name)
}

func TestBorrowCheckerWithRealMIR(t *testing.T) {
	// Test with actual MIR instructions.
	lm := NewLifetimeManager()
	bc := NewBorrowChecker(lm)

	function := &Function{
		Name:       "borrow_test",
		Parameters: []Value{},
		Blocks: []*BasicBlock{
			{
				Name: "entry",
				Instr: []Instr{
					// Allocate local variable.
					Alloca{Dst: "%x.addr", Name: "x"},
					// Store value.
					Store{
						Addr: Value{Kind: ValRef, Ref: "%x.addr"},
						Val:  Value{Kind: ValConstInt, Int64: 10},
					},
					// Load value (creates implicit borrow).
					Load{
						Dst:  "%temp1",
						Addr: Value{Kind: ValRef, Ref: "%x.addr"},
					},
					// Another load (should be allowed - immutable borrows).
					Load{
						Dst:  "%temp2",
						Addr: Value{Kind: ValRef, Ref: "%x.addr"},
					},
					// Try to store again (might conflict with borrows).
					Store{
						Addr: Value{Kind: ValRef, Ref: "%x.addr"},
						Val:  Value{Kind: ValConstInt, Int64: 20},
					},
					Ret{Val: &Value{Kind: ValRef, Ref: "%temp1"}},
				},
			},
		},
	}

	// Run borrow analysis.
	for i, block := range function.Blocks {
		for j, instr := range block.Instr {
			point := BorrowPoint{
				Function: function.Name,
				Block:    block.Name,
				Stmt:     j,
			}

			err := bc.AnalyzeBorrows(instr, point)
			if err != nil {
				t.Errorf("Borrow analysis failed at block %d, instruction %d: %v", i, j, err)
			}
		}
	}

	// Check final borrow state.
	t.Logf("Final borrow checker state:\n%s", bc.String())

	if len(bc.GetErrors()) > 0 {
		t.Logf("Borrow checker found %d errors:", len(bc.GetErrors()))

		for i, err := range bc.GetErrors() {
			t.Logf("  Error %d: %v", i+1, err)
		}
	}
}

func TestOwnershipWithDrops(t *testing.T) {
	lm := NewLifetimeManager()
	bc := NewBorrowChecker(lm)
	om := NewOwnershipManager(bc)

	// Test drop operations.
	value := Value{Kind: ValRef, Ref: "%droppable"}
	point := MovePoint{
		Function: "test_func",
		Block:    "entry",
		Stmt:     5,
	}

	// Set value as owned.
	om.SetValueState(value, StateOwned)

	// Create drop.
	err := om.CreateDrop(value, point)
	if err != nil {
		t.Fatalf("Drop creation failed: %v", err)
	}

	// Check state after drop.
	state := om.GetValueState(value)
	if state != StateDropped {
		t.Errorf("Expected StateDropped, got %v", state)
	}

	// Try to drop again (should fail).
	err = om.CreateDrop(value, point)
	if err == nil {
		t.Error("Expected error when dropping already dropped value")
	}
}

func TestCopyVsMove(t *testing.T) {
	lm := NewLifetimeManager()
	bc := NewBorrowChecker(lm)
	om := NewOwnershipManager(bc)

	// Create a copyable value.
	copyableValue := Value{Kind: ValRef, Ref: "%copyable"}
	copyableTraits := OwnershipTraits{
		Copy:  true,
		Clone: true,
		Drop:  false, // Copy types typically don't need drops
		Send:  true,
		Sync:  true,
		Unpin: true,
		Sized: true,
	}

	origin := OwnershipOrigin{
		Kind:     OwnershipOriginAllocation,
		Function: "test_func",
		Block:    "entry",
		Stmt:     0,
		Source:   "copyable allocation",
	}

	om.CreateOwnership(OwnershipOwned, copyableValue, copyableValue, "", copyableTraits, origin)

	// Test copy behavior.
	if !om.CanCopy(copyableValue) {
		t.Error("Expected copyable value to be copyable")
	}

	if om.ShouldMove(copyableValue, "assignment") {
		t.Error("Expected copy for assignment context")
	}

	if !om.ShouldMove(copyableValue, "return") {
		t.Error("Expected move for return context")
	}

	// Create a non-copyable value.
	nonCopyableValue := Value{Kind: ValRef, Ref: "%non_copyable"}
	nonCopyableTraits := OwnershipTraits{
		Copy:  false,
		Clone: true,
		Drop:  true,
		Send:  true,
		Sync:  false,
		Unpin: true,
		Sized: true,
	}

	om.CreateOwnership(OwnershipOwned, nonCopyableValue, nonCopyableValue, "", nonCopyableTraits, origin)

	if om.CanCopy(nonCopyableValue) {
		t.Error("Expected non-copyable value to not be copyable")
	}

	if !om.ShouldMove(nonCopyableValue, "assignment") {
		t.Error("Expected move for non-copyable value")
	}
}
