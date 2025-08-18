package codegen

import (
	"strings"
	"testing"

	"github.com/orizon-lang/orizon/internal/layout"
	"github.com/orizon-lang/orizon/internal/lir"
)

func TestLayoutAwareEmitter(t *testing.T) {
	emitter := NewLayoutAwareEmitter()

	if emitter.Calculator == nil {
		t.Error("Layout calculator should be initialized")
	}

	if emitter.TypeLayouts == nil {
		t.Error("Type layouts map should be initialized")
	}
}

func TestEmitWithLayouts(t *testing.T) {
	emitter := NewLayoutAwareEmitter()

	tests := []struct {
		name     string
		module   *lir.Module
		contains []string // Expected strings in output
	}{
		{
			name: "simple_function",
			module: &lir.Module{
				Name: "test_layout",
				Functions: []*lir.Function{
					{
						Name: "test_func",
						Blocks: []*lir.BasicBlock{
							{
								Label: "entry",
								Insns: []lir.Insn{
									lir.Mov{Src: "42", Dst: "%1"},
									lir.Ret{Src: "%1"},
								},
							},
						},
					},
				},
			},
			contains: []string{
				"module test_layout (layout-aware)",
				"Data structure layout definitions",
				"Slice header:",
				"String header:",
				"test_func:",
				"push rbp",
				"pop rbp",
				"ret",
			},
		},
		{
			name: "array_allocation",
			module: &lir.Module{
				Name: "array_test",
				Functions: []*lir.Function{
					{
						Name: "array_func",
						Blocks: []*lir.BasicBlock{
							{
								Label: "entry",
								Insns: []lir.Insn{
									lir.Alloc{Name: "array_10_i32", Dst: "%arr"},
									lir.Mov{Src: "42", Dst: "%val"},
									lir.Store{Addr: "%arr", Val: "%val"},
									lir.Load{Dst: "%result", Addr: "%arr"},
									lir.Ret{Src: "%result"},
								},
							},
						},
					},
				},
			},
			contains: []string{
				"array_func:",
				"layout stack space",
				"array initialization",
				"mov qword ptr",
				"mov rax, qword ptr",
			},
		},
		{
			name: "slice_operations",
			module: &lir.Module{
				Name: "slice_test",
				Functions: []*lir.Function{
					{
						Name: "slice_func",
						Blocks: []*lir.BasicBlock{
							{
								Label: "entry",
								Insns: []lir.Insn{
									lir.Alloc{Name: "slice_i32", Dst: "%slice"},
									lir.Call{
										Callee: "slice_append",
										Args:   []string{"%slice", "42"},
										Dst:    "%new_slice",
									},
									lir.Ret{Src: "%new_slice"},
								},
							},
						},
					},
				},
			},
			contains: []string{
				"slice_func:",
				"slice header initialization",
				"data ptr = null",
				"len = 0",
				"cap = 0",
				"slice operation: slice_append",
			},
		},
		{
			name: "function_call_with_layouts",
			module: &lir.Module{
				Name: "call_test",
				Functions: []*lir.Function{
					{
						Name: "call_func",
						Blocks: []*lir.BasicBlock{
							{
								Label: "entry",
								Insns: []lir.Insn{
									lir.Mov{Src: "10", Dst: "%arg1"},
									lir.Mov{Src: "20", Dst: "%arg2"},
									lir.Call{
										Callee: "external_func",
										Args:   []string{"%arg1", "%arg2"},
										Dst:    "%result",
									},
									lir.Ret{Src: "%result"},
								},
							},
						},
					},
				},
			},
			contains: []string{
				"call_func:",
				"shadow space",
				"mov rcx,",
				"mov rdx,",
				"call external_func",
				"return value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asm, err := emitter.EmitWithLayouts(tt.module)
			if err != nil {
				t.Fatalf("Failed to emit assembly: %v", err)
			}

			if asm == "" {
				t.Fatal("No assembly generated")
			}

			for _, expected := range tt.contains {
				if !strings.Contains(asm, expected) {
					t.Errorf("Expected string not found in assembly: %q", expected)
				}
			}

			if testing.Verbose() {
				t.Logf("Generated assembly for %s:\n%s", tt.name, asm)
			}
		})
	}
}

func TestLayoutRequirementAnalysis(t *testing.T) {
	emitter := NewLayoutAwareEmitter()

	function := &lir.Function{
		Name: "analysis_test",
		Blocks: []*lir.BasicBlock{
			{
				Label: "entry",
				Insns: []lir.Insn{
					lir.Alloc{Name: "array_5_i64", Dst: "%arr1"},
					lir.Alloc{Name: "slice_i32", Dst: "%slice1"},
					lir.Alloc{Name: "normal_var", Dst: "%var1"},
					lir.Load{Dst: "%val", Addr: "%arr1"},
					lir.Store{Addr: "%slice1", Val: "%val"},
				},
			},
		},
	}

	req := emitter.analyzeLayoutRequirements(function)

	// Check that array allocation was detected
	if len(req.LocalArrays) != 1 {
		t.Errorf("Expected 1 array allocation, got %d", len(req.LocalArrays))
	}

	// Check that slice allocation was detected
	if len(req.LocalSlices) != 1 {
		t.Errorf("Expected 1 slice allocation, got %d", len(req.LocalSlices))
	}

	// Check stack offset calculation
	stackSize := emitter.calculateStackRequirements(req)
	if stackSize == 0 {
		t.Error("Expected non-zero stack size for layout requirements")
	}

	t.Logf("Stack requirements: %d bytes", stackSize)
	t.Logf("Arrays: %d, Slices: %d", len(req.LocalArrays), len(req.LocalSlices))
}

func TestLayoutAwareInstructions(t *testing.T) {
	emitter := NewLayoutAwareEmitter()

	// Create dummy layout requirements
	req := &LayoutRequirement{
		LocalArrays:  make(map[string]*layout.ArrayLayout),
		LocalSlices:  make(map[string]*layout.SliceLayout),
		LocalStructs: make(map[string]*layout.StructLayout),
		StackOffset:  map[string]int64{"%arr": 16, "%slice": 64},
	}

	tests := []struct {
		name        string
		instruction lir.Insn
		contains    []string
	}{
		{
			name:        "layout_aware_alloc",
			instruction: lir.Alloc{Name: "array_10_i32", Dst: "%arr"},
			contains:    []string{"lea rax, [rbp-16]", "%arr"},
		},
		{
			name:        "layout_aware_load",
			instruction: lir.Load{Dst: "%val", Addr: "%arr"},
			contains:    []string{"mov rax, qword ptr", "%val = load %arr"},
		},
		{
			name:        "layout_aware_store",
			instruction: lir.Store{Addr: "%slice", Val: "%val"},
			contains:    []string{"mov qword ptr", "store %slice, %val"},
		},
		{
			name:        "basic_mov",
			instruction: lir.Mov{Src: "42", Dst: "%reg"},
			contains:    []string{"mov %reg, 42"},
		},
		{
			name:        "basic_add",
			instruction: lir.Add{Dst: "%result", LHS: "%a", RHS: "%b"},
			contains:    []string{"mov %result, %a", "add %result, %b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asm, err := emitter.emitLayoutAwareInstruction(tt.instruction, req)
			if err != nil {
				t.Fatalf("Failed to emit instruction: %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(asm, expected) {
					t.Errorf("Expected string not found in instruction assembly: %q\nGenerated: %s", expected, asm)
				}
			}
		})
	}
}

func TestSliceLayoutGeneration(t *testing.T) {
	emitter := NewLayoutAwareEmitter()

	// Test slice header initialization
	instruction := lir.Alloc{Name: "slice_i32", Dst: "%slice"}

	req := &LayoutRequirement{
		LocalArrays:  make(map[string]*layout.ArrayLayout),
		LocalSlices:  make(map[string]*layout.SliceLayout),
		LocalStructs: make(map[string]*layout.StructLayout),
		StackOffset:  map[string]int64{"%slice": 32},
	}

	// Add slice layout to requirements
	sliceLayout, _ := emitter.Calculator.CalculateSliceLayout("i32", 4, 4)
	req.LocalSlices["%slice"] = sliceLayout

	asm, err := emitter.emitLayoutAwareAlloc(instruction, req)
	if err != nil {
		t.Fatalf("Failed to emit slice allocation: %v", err)
	}

	expectedParts := []string{
		"lea rax, [rbp-32]",
		"slice header initialization",
		"data ptr = null",
		"len = 0",
		"cap = 0",
	}

	for _, expected := range expectedParts {
		if !strings.Contains(asm, expected) {
			t.Errorf("Expected slice initialization part not found: %q\nGenerated: %s", expected, asm)
		}
	}
}

func TestArrayLayoutGeneration(t *testing.T) {
	emitter := NewLayoutAwareEmitter()

	// Test array allocation and initialization
	instruction := lir.Alloc{Name: "array_10_i32", Dst: "%arr"}

	req := &LayoutRequirement{
		LocalArrays:  make(map[string]*layout.ArrayLayout),
		LocalSlices:  make(map[string]*layout.SliceLayout),
		LocalStructs: make(map[string]*layout.StructLayout),
		StackOffset:  map[string]int64{"%arr": 48},
	}

	// Add array layout to requirements
	arrayLayout, _ := emitter.Calculator.CalculateArrayLayout("i32", 4, 4, 10)
	req.LocalArrays["%arr"] = arrayLayout

	asm, err := emitter.emitLayoutAwareAlloc(instruction, req)
	if err != nil {
		t.Fatalf("Failed to emit array allocation: %v", err)
	}

	expectedParts := []string{
		"lea rax, [rbp-48]",
		"array initialization",
		"10 elements",
		"4 bytes",
	}

	for _, expected := range expectedParts {
		if !strings.Contains(asm, expected) {
			t.Errorf("Expected array initialization part not found: %q\nGenerated: %s", expected, asm)
		}
	}
}

func TestLayoutAwareCallGeneration(t *testing.T) {
	emitter := NewLayoutAwareEmitter()
	req := &LayoutRequirement{
		LocalArrays:  make(map[string]*layout.ArrayLayout),
		LocalSlices:  make(map[string]*layout.SliceLayout),
		LocalStructs: make(map[string]*layout.StructLayout),
		StackOffset:  make(map[string]int64),
	}

	tests := []struct {
		name     string
		call     lir.Call
		contains []string
	}{
		{
			name: "slice_operation",
			call: lir.Call{
				Callee: "slice_append",
				Args:   []string{"%slice", "%val"},
				Dst:    "%result",
			},
			contains: []string{"slice operation: slice_append"},
		},
		{
			name: "array_operation",
			call: lir.Call{
				Callee: "array_get",
				Args:   []string{"%arr", "%index"},
				Dst:    "%element",
			},
			contains: []string{"array operation: array_get"},
		},
		{
			name: "string_operation",
			call: lir.Call{
				Callee: "string_concat",
				Args:   []string{"%str1", "%str2"},
				Dst:    "%result",
			},
			contains: []string{"string operation: string_concat"},
		},
		{
			name: "regular_function",
			call: lir.Call{
				Callee: "regular_func",
				Args:   []string{"%arg1", "%arg2"},
				Dst:    "%result",
			},
			contains: []string{"shadow space", "mov rcx,", "mov rdx,", "call regular_func"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asm, err := emitter.emitLayoutAwareCall(tt.call, req)
			if err != nil {
				t.Fatalf("Failed to emit call: %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(asm, expected) {
					t.Errorf("Expected call part not found: %q\nGenerated: %s", expected, asm)
				}
			}
		})
	}
}

func BenchmarkLayoutAwareEmission(b *testing.B) {
	emitter := NewLayoutAwareEmitter()

	module := &lir.Module{
		Name: "benchmark_module",
		Functions: []*lir.Function{
			{
				Name: "benchmark_func",
				Blocks: []*lir.BasicBlock{
					{
						Label: "entry",
						Insns: []lir.Insn{
							lir.Alloc{Name: "array_100_i64", Dst: "%big_array"},
							lir.Alloc{Name: "slice_i32", Dst: "%slice"},
							lir.Mov{Src: "42", Dst: "%val"},
							lir.Store{Addr: "%big_array", Val: "%val"},
							lir.Load{Dst: "%result", Addr: "%big_array"},
							lir.Call{
								Callee: "slice_append",
								Args:   []string{"%slice", "%result"},
								Dst:    "%new_slice",
							},
							lir.Ret{Src: "%new_slice"},
						},
					},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := emitter.EmitWithLayouts(module)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}
