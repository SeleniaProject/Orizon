package allocator

import (
	"testing"
	"unsafe"
)

// TestMIRIntegration tests MIR integration for allocator.
func TestMIRIntegration(t *testing.T) {
	config := defaultConfig()
	mir := NewMIRIntegration(SystemAllocatorKind, config)

	t.Run("AllocInstruction", func(t *testing.T) {
		instructions := mir.GenerateAllocInstruction("result", 1024, 8)

		if len(instructions) == 0 {
			t.Fatal("No instructions generated")
		}

		// Check for required instructions.
		hasLoadImmediate := false
		hasCallRuntime := false
		hasNullCheck := false

		for _, instr := range instructions {
			switch instr.Op {
			case "load_immediate":
				hasLoadImmediate = true
			case "call_runtime":
				hasCallRuntime = true
			case "null_check":
				hasNullCheck = true
			}
		}

		if !hasLoadImmediate {
			t.Error("Missing load_immediate instruction")
		}

		if !hasCallRuntime {
			t.Error("Missing call_runtime instruction")
		}

		if !hasNullCheck {
			t.Error("Missing null_check instruction")
		}
	})

	t.Run("FreeInstruction", func(t *testing.T) {
		instructions := mir.GenerateFreeInstruction("ptr")

		if len(instructions) == 0 {
			t.Fatal("No instructions generated")
		}

		// Should have null check and call_runtime.
		hasNullCheck := false
		hasCallRuntime := false

		for _, instr := range instructions {
			switch instr.Op {
			case "null_check":
				hasNullCheck = true
			case "call_runtime":
				hasCallRuntime = true
			}
		}

		if !hasNullCheck {
			t.Error("Missing null_check instruction")
		}

		if !hasCallRuntime {
			t.Error("Missing call_runtime instruction")
		}
	})

	t.Run("ArrayAllocInstruction", func(t *testing.T) {
		instructions := mir.GenerateArrayAllocInstruction("result", 8, "count")

		if len(instructions) == 0 {
			t.Fatal("No instructions generated")
		}

		// Should have multiplication and runtime call.
		hasMul := false
		hasCallRuntime := false

		for _, instr := range instructions {
			switch instr.Op {
			case "mul":
				hasMul = true
			case "call_runtime":
				hasCallRuntime = true
			}
		}

		if !hasMul {
			t.Error("Missing mul instruction for size calculation")
		}

		if !hasCallRuntime {
			t.Error("Missing call_runtime instruction")
		}
	})

	t.Run("OptimizeInstructions", func(t *testing.T) {
		// Create instructions with redundant null checks.
		instructions := []MIRInstruction{
			{Op: "load_immediate", Operands: []string{"1024"}, Result: "size"},
			{Op: "null_check", Operands: []string{"ptr"}},
			{Op: "null_check", Operands: []string{"ptr"}}, // Redundant
			{Op: "call_runtime", Operands: []string{"alloc"}},
		}

		optimized := mir.OptimizeInstructions(instructions)

		// Should remove redundant null check.
		nullCheckCount := 0

		for _, instr := range optimized {
			if instr.Op == "null_check" {
				nullCheckCount++
			}
		}

		if nullCheckCount != 1 {
			t.Errorf("Expected 1 null_check after optimization, got %d", nullCheckCount)
		}
	})
}

// TestX64Integration tests x64 assembly integration.
func TestX64Integration(t *testing.T) {
	config := defaultConfig()
	x64 := NewX64Integration(SystemAllocatorKind, config)

	t.Run("AllocCode", func(t *testing.T) {
		instructions := x64.GenerateAllocCode("rcx", "rax")

		if len(instructions) == 0 {
			t.Fatal("No instructions generated")
		}

		// Should have function call and register management.
		hasCall := false
		hasPush := false
		hasPop := false

		for _, instr := range instructions {
			switch instr.Mnemonic {
			case "call":
				hasCall = true
			case "push":
				hasPush = true
			case "pop":
				hasPop = true
			}
		}

		if !hasCall {
			t.Error("Missing call instruction")
		}

		if !hasPush {
			t.Error("Missing register save (push)")
		}

		if !hasPop {
			t.Error("Missing register restore (pop)")
		}
	})

	t.Run("InlinedArenaAlloc", func(t *testing.T) {
		x64Arena := NewX64Integration(ArenaAllocatorKind, config)
		instructions := x64Arena.GenerateInlinedArenaAlloc(64, "rax")

		if len(instructions) == 0 {
			t.Fatal("No instructions generated")
		}

		// Should have arena pointer manipulation.
		hasLoad := false
		hasLea := false
		hasCmp := false

		for _, instr := range instructions {
			switch instr.Mnemonic {
			case "mov":
				if len(instr.Operands) > 1 && instr.Operands[1] == "qword ptr [global_arena_current]" {
					hasLoad = true
				}
			case "lea":
				hasLea = true
			case "cmp":
				hasCmp = true
			}
		}

		if !hasLoad {
			t.Error("Missing arena current pointer load")
		}

		if !hasLea {
			t.Error("Missing lea instruction for pointer calculation")
		}

		if !hasCmp {
			t.Error("Missing bounds check comparison")
		}
	})

	t.Run("FormatInstructions", func(t *testing.T) {
		instructions := []X64Instruction{
			{Mnemonic: "mov", Operands: []string{"rax", "rbx"}, Comment: "Test move"},
			{Mnemonic: "test_label:", Operands: []string{}, Comment: ""},
			{Mnemonic: "ret", Operands: []string{}, Comment: "Return"},
		}

		formatted := x64.FormatInstructions(instructions)

		if formatted == "" {
			t.Fatal("No formatted output")
		}

		// Should contain proper assembly formatting.
		if !contains(formatted, "mov") {
			t.Error("Missing mov instruction in formatted output")
		}

		if !contains(formatted, "test_label:") {
			t.Error("Missing label in formatted output")
		}

		if !contains(formatted, "Test move") {
			t.Error("Missing comment in formatted output")
		}
	})
}

// TestRuntimeIntegration tests complete runtime integration.
func TestRuntimeIntegration(t *testing.T) {
	// Test system allocator integration.
	t.Run("SystemAllocatorRuntime", func(t *testing.T) {
		config := defaultConfig()
		systemAlloc := NewSystemAllocator(config)

		err := InitializeRuntime(systemAlloc)
		if err != nil {
			t.Fatalf("Failed to initialize runtime: %v", err)
		}

		defer ShutdownRuntime()

		// Test runtime allocation.
		ptr := RuntimeAlloc(1024)
		if ptr == nil {
			t.Fatal("Runtime allocation failed")
		}

		// Test memory access.
		data := (*[1024]byte)(ptr)
		data[0] = 42
		data[1023] = 43

		if data[0] != 42 || data[1023] != 43 {
			t.Error("Memory access failed")
		}

		RuntimeFree(ptr)
	})

	t.Run("ArenaAllocatorRuntime", func(t *testing.T) {
		config := defaultConfig()

		arenaAlloc, err := NewArenaAllocator(64*1024, config)
		if err != nil {
			t.Fatalf("Failed to create arena allocator: %v", err)
		}

		err = InitializeRuntime(arenaAlloc)
		if err != nil {
			t.Fatalf("Failed to initialize runtime: %v", err)
		}

		defer ShutdownRuntime()

		// Test multiple allocations.
		var ptrs []unsafe.Pointer

		for i := 0; i < 10; i++ {
			ptr := RuntimeAlloc(512)
			if ptr == nil {
				t.Fatalf("Arena allocation %d failed", i)
			}

			ptrs = append(ptrs, ptr)
		}

		// Verify allocations are in order (arena property).
		for i := 1; i < len(ptrs); i++ {
			prev := uintptr(ptrs[i-1])
			curr := uintptr(ptrs[i])

			if curr <= prev {
				t.Error("Arena allocations not in order")
			}
		}
	})

	t.Run("SliceAllocation", func(t *testing.T) {
		config := defaultConfig()
		allocator := NewSystemAllocator(config)

		err := InitializeRuntime(allocator)
		if err != nil {
			t.Fatalf("Failed to initialize runtime: %v", err)
		}

		defer ShutdownRuntime()

		// Test slice allocation.
		header := RuntimeAllocSlice(4, 10, 20)
		if header == nil {
			t.Fatal("Slice allocation failed")
		}

		if header.Len != 10 {
			t.Errorf("Slice length wrong: got %d, want 10", header.Len)
		}

		if header.Cap != 20 {
			t.Errorf("Slice capacity wrong: got %d, want 20", header.Cap)
		}

		if header.Data == nil {
			t.Error("Slice data is nil")
		}

		// Test slice access.
		slice := (*[20]uint32)(header.Data)
		for i := 0; i < 10; i++ {
			slice[i] = uint32(i * 2)
		}

		for i := 0; i < 10; i++ {
			if slice[i] != uint32(i*2) {
				t.Errorf("Slice data corrupted at %d", i)
			}
		}

		RuntimeFreeSlice(header)
	})

	t.Run("StringPooling", func(t *testing.T) {
		config := defaultConfig()
		allocator := NewSystemAllocator(config)

		err := InitializeRuntime(allocator)
		if err != nil {
			t.Fatalf("Failed to initialize runtime: %v", err)
		}

		defer ShutdownRuntime()

		testStr := "Hello, World!"

		// Allocate same string twice.
		ptr1 := RuntimeAllocString(testStr)
		ptr2 := RuntimeAllocString(testStr)

		if ptr1 == nil || ptr2 == nil {
			t.Fatal("String allocation failed")
		}

		// Should hit string pool on second allocation.
		stats := GetRuntimeStats()
		if stats.StringPool.Hits == 0 {
			t.Error("String pool should have hits")
		}
	})
}

// TestAllocatorInteroperability tests interoperability between different allocator types.
func TestAllocatorInteroperability(t *testing.T) {
	t.Run("AllocatorSwitch", func(t *testing.T) {
		// Test switching between allocator types.
		config := defaultConfig()
		config.EnableTracking = true

		// Start with system allocator.
		err := Initialize(SystemAllocatorKind, WithTracking(true))
		if err != nil {
			t.Fatalf("Failed to initialize system allocator: %v", err)
		}

		ptr1 := Alloc(1024)
		if ptr1 == nil {
			t.Fatal("System allocation failed")
		}

		stats1 := GetStats()
		if stats1.AllocationCount == 0 {
			t.Error("System allocator should show allocations")
		}

		Free(ptr1)

		// Switch to arena allocator.
		err = Initialize(ArenaAllocatorKind, WithArenaSize(32*1024))
		if err != nil {
			t.Fatalf("Failed to initialize arena allocator: %v", err)
		}

		ptr2 := Alloc(1024)
		if ptr2 == nil {
			t.Fatal("Arena allocation failed")
		}

		stats2 := GetStats()
		if stats2.AllocationCount == 0 {
			t.Error("Arena allocator should show allocations")
		}

		// Arena allocator doesn't support individual free.
		// so we just reset.
		if arena, ok := GlobalAllocator.(*ArenaAllocatorImpl); ok {
			arena.Reset()
		}
	})
}

// TestPerformanceCharacteristics tests performance characteristics of allocators.
func TestPerformanceCharacteristics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("AllocationSpeed", func(t *testing.T) {
		config := defaultConfig()
		config.EnableTracking = false // Disable for performance

		allocators := map[string]Allocator{
			"System": NewSystemAllocator(config),
		}

		// Add arena allocator.
		if arena, err := NewArenaAllocator(1024*1024, config); err == nil {
			allocators["Arena"] = arena
		}

		// Add pool allocator.
		if pool, err := NewPoolAllocator([]uintptr{64, 128, 256, 512, 1024}, config); err == nil {
			allocators["Pool"] = pool
		}

		for name, allocator := range allocators {
			t.Run(name, func(t *testing.T) {
				const numAllocs = 1000

				// Allocate.
				ptrs := make([]unsafe.Pointer, numAllocs)
				for i := 0; i < numAllocs; i++ {
					ptrs[i] = allocator.Alloc(256)
					if ptrs[i] == nil {
						t.Fatalf("Allocation %d failed", i)
					}
				}

				// Free (if supported).
				for _, ptr := range ptrs {
					allocator.Free(ptr)
				}

				// Reset if arena.
				if arena, ok := allocator.(*ArenaAllocatorImpl); ok {
					arena.Reset()
				}
			})
		}
	})
}

// Helper function to check if string contains substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr ||
		(len(s) > len(substr) && contains(s[1:], substr))
}
