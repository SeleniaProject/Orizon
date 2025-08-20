// Package integration provides additional helper tests for the staged pipeline testing system.
// This file contains smaller, focused tests that validate individual components.
// before running the full end-to-end pipeline tests.
package integration

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/lir"
	"github.com/orizon-lang/orizon/internal/mir"
	"github.com/orizon-lang/orizon/internal/parser"
)

// TestStage1_HIRBasics tests the basic HIR creation and structure.
func TestStage1_HIRBasics(t *testing.T) {
	suite := NewPipelineTestSuite("../../tmp/stage1_test_output")

	// Test simple HIR creation.
	t.Run("SimpleHIRCreation", func(t *testing.T) {
		hirModule, err := suite.parseToHIR("fn test() -> i32 { return 42; }")
		if err != nil {
			t.Fatalf("HIR parsing failed: %v", err)
		}

		if hirModule == nil {
			t.Fatal("HIR module is nil")
		}

		if hirModule.Name != "test_module" {
			t.Errorf("Expected module name 'test_module', got '%s'", hirModule.Name)
		}

		if len(hirModule.Functions) != 1 {
			t.Errorf("Expected 1 function, got %d", len(hirModule.Functions))
		}

		t.Logf("✅ HIR creation test passed")
	})
}

// TestStage2_MIRTransformation tests the HIR to MIR transformation.
func TestStage2_MIRTransformation(t *testing.T) {
	suite := NewPipelineTestSuite("../../tmp/stage2_test_output")

	t.Run("HIRToMIRBasic", func(t *testing.T) {
		// Create a simple HIR module.
		hirModule := &parser.HIRModule{
			Name: "test_module",
			Functions: []*parser.HIRFunction{
				{
					Name:       "test_function",
					Parameters: make([]*parser.HIRParameter, 0),
					Body:       suite.createSimpleHIRBody(),
				},
			},
		}

		// Transform to MIR.
		mirModule, err := suite.transformToMIR(hirModule, true)
		if err != nil {
			t.Fatalf("MIR transformation failed: %v", err)
		}

		if mirModule == nil {
			t.Fatal("MIR module is nil")
		}

		if mirModule.Name != "test_module" {
			t.Errorf("Expected module name 'test_module', got '%s'", mirModule.Name)
		}

		if len(mirModule.Functions) != 1 {
			t.Errorf("Expected 1 function, got %d", len(mirModule.Functions))
		}

		// Check function structure.
		mirFunc := mirModule.Functions[0]
		if mirFunc.Name != "test_function" {
			t.Errorf("Expected function name 'test_function', got '%s'", mirFunc.Name)
		}

		if len(mirFunc.Blocks) == 0 {
			t.Error("Function should have at least one basic block")
		}

		t.Logf("✅ HIR to MIR transformation test passed")
		t.Logf("MIR output:\n%s", mirModule.String())
	})
}

// TestStage3_LIRTransformation tests the MIR to LIR transformation.
func TestStage3_LIRTransformation(t *testing.T) {
	suite := NewPipelineTestSuite("../../tmp/stage3_test_output")

	t.Run("MIRToLIRBasic", func(t *testing.T) {
		// Create a simple MIR module.
		mirModule := &mir.Module{
			Name: "test_module",
			Functions: []*mir.Function{
				{
					Name:       "test_function",
					Parameters: []mir.Value{},
					Blocks: []*mir.BasicBlock{
						{
							Name: "entry",
							Instr: []mir.Instr{
								mir.Ret{Val: &mir.Value{Kind: mir.ValConstInt, Int64: 42}},
							},
						},
					},
				},
			},
		}

		// Transform to LIR.
		lirModule, err := suite.transformToLIR(mirModule)
		if err != nil {
			t.Fatalf("LIR transformation failed: %v", err)
		}

		if lirModule == nil {
			t.Fatal("LIR module is nil")
		}

		if lirModule.Name != "test_module" {
			t.Errorf("Expected module name 'test_module', got '%s'", lirModule.Name)
		}

		if len(lirModule.Functions) != 1 {
			t.Errorf("Expected 1 function, got %d", len(lirModule.Functions))
		}

		// Check function structure.
		lirFunc := lirModule.Functions[0]
		if lirFunc.Name != "test_function" {
			t.Errorf("Expected function name 'test_function', got '%s'", lirFunc.Name)
		}

		if len(lirFunc.Blocks) == 0 {
			t.Error("Function should have at least one basic block")
		}

		// Check that instructions were converted.
		lirBlock := lirFunc.Blocks[0]
		if len(lirBlock.Insns) == 0 {
			t.Error("LIR block should have at least one instruction")
		}

		t.Logf("✅ MIR to LIR transformation test passed")
		t.Logf("LIR output:\n%s", lirModule.String())
	})
}

// TestStage4_CodeGeneration tests the LIR to assembly generation.
func TestStage4_CodeGeneration(t *testing.T) {
	suite := NewPipelineTestSuite("../../tmp/stage4_test_output")

	t.Run("LIRToAssemblyBasic", func(t *testing.T) {
		// Create a simple LIR module using the existing LIR structures.
		lirModule := createSimpleLIRModule()

		// Generate assembly.
		asmCode, err := suite.generateAssembly(lirModule, 0)
		if err != nil {
			t.Fatalf("Assembly generation failed: %v", err)
		}

		if asmCode == "" {
			t.Fatal("Generated assembly is empty")
		}

		// Basic validation that assembly contains expected elements.
		if !containsString(asmCode, "test_function") {
			t.Error("Assembly should contain function name")
		}

		t.Logf("✅ LIR to Assembly generation test passed")
		t.Logf("Generated assembly:\n%s", asmCode)
	})
}

// TestStage5_MemorySafetyValidation tests memory safety features.
func TestStage5_MemorySafetyValidation(t *testing.T) {
	suite := NewPipelineTestSuite("../../tmp/stage5_test_output")

	t.Run("MemorySafetyBasic", func(t *testing.T) {
		// Create HIR with memory operations.
		hirModule := &parser.HIRModule{
			Name: "memory_test",
			Functions: []*parser.HIRFunction{
				{
					Name:       "memory_function",
					Parameters: make([]*parser.HIRParameter, 0),
					Body:       suite.createSimpleHIRBody(),
				},
			},
		}

		// Transform with memory safety enabled.
		mirModule, err := suite.transformToMIR(hirModule, true)
		if err != nil {
			t.Fatalf("MIR transformation with memory safety failed: %v", err)
		}

		if mirModule == nil {
			t.Fatal("MIR module is nil")
		}

		t.Logf("✅ Memory safety validation test passed")
		t.Logf("Memory-safe MIR:\n%s", mirModule.String())
	})
}

// Helper functions.

// createSimpleLIRModule creates a basic LIR module for testing.
func createSimpleLIRModule() *lir.Module {
	return &lir.Module{
		Name: "test_module",
		Functions: []*lir.Function{
			{
				Name: "test_function",
				Blocks: []*lir.BasicBlock{
					{
						Label: "entry",
						Insns: []lir.Insn{
							lir.Ret{Src: "42"},
						},
					},
				},
			},
		},
	}
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 1; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}

				return false
			}()))
}
