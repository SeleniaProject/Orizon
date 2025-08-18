package types

import (
	"testing"

	"github.com/orizon-lang/orizon/internal/mir"
)

// TestCoreMIRIntegration tests MIR integration for core types
func TestCoreMIRIntegration(t *testing.T) {
	module := &mir.Module{
		Name:      "test_module",
		Functions: []*mir.Function{},
	}

	integration := NewCoreTypeMIRIntegration(module)
	if integration == nil {
		t.Fatal("Failed to create CoreTypeMIRIntegration")
	}

	if integration.module != module {
		t.Error("MIR module not properly set")
	}
}

// TestOptionOperations tests Option type MIR operations
func TestOptionOperations(t *testing.T) {
	module := &mir.Module{
		Name:      "option_test",
		Functions: []*mir.Function{},
	}

	integration := NewCoreTypeMIRIntegration(module)
	optionFuncs := integration.GenerateOptionOperations()

	if len(optionFuncs) == 0 {
		t.Error("No Option functions generated")
	}

	// Check that we have the expected function
	found := false
	for _, fn := range optionFuncs {
		if fn.Name == "option_some" {
			found = true
			if len(fn.Parameters) != 1 {
				t.Errorf("Expected 1 parameter for option_some, got %d", len(fn.Parameters))
			}
			if len(fn.Blocks) != 1 {
				t.Errorf("Expected 1 block for option_some, got %d", len(fn.Blocks))
			}
			break
		}
	}

	if !found {
		t.Error("option_some function not found")
	}
}

// TestResultOperations tests Result type MIR operations
func TestResultOperations(t *testing.T) {
	module := &mir.Module{
		Name:      "result_test",
		Functions: []*mir.Function{},
	}

	integration := NewCoreTypeMIRIntegration(module)
	resultFuncs := integration.GenerateResultOperations()

	if len(resultFuncs) == 0 {
		t.Error("No Result functions generated")
	}

	// Check that we have the expected function
	found := false
	for _, fn := range resultFuncs {
		if fn.Name == "result_ok" {
			found = true
			if len(fn.Parameters) != 1 {
				t.Errorf("Expected 1 parameter for result_ok, got %d", len(fn.Parameters))
			}
			if len(fn.Blocks) != 1 {
				t.Errorf("Expected 1 block for result_ok, got %d", len(fn.Blocks))
			}
			break
		}
	}

	if !found {
		t.Error("result_ok function not found")
	}
}

// TestGetAllCoreFunctions tests getting all core type functions
func TestGetAllCoreFunctions(t *testing.T) {
	module := &mir.Module{
		Name:      "all_core_test",
		Functions: []*mir.Function{},
	}

	integration := NewCoreTypeMIRIntegration(module)
	allFuncs := integration.GetAllCoreFunctions()

	if len(allFuncs) < 2 {
		t.Errorf("Expected at least 2 core functions, got %d", len(allFuncs))
	}

	// Verify we have both option and result functions
	hasOption := false
	hasResult := false

	for _, fn := range allFuncs {
		if fn.Name == "option_some" {
			hasOption = true
		}
		if fn.Name == "result_ok" {
			hasResult = true
		}
	}

	if !hasOption {
		t.Error("Option function not found in all core functions")
	}
	if !hasResult {
		t.Error("Result function not found in all core functions")
	}
}

// TestRegisterCoreFunctions tests registering functions with the module
func TestRegisterCoreFunctions(t *testing.T) {
	module := &mir.Module{
		Name:      "register_test",
		Functions: []*mir.Function{},
	}

	integration := NewCoreTypeMIRIntegration(module)

	// Initially no functions
	if len(module.Functions) != 0 {
		t.Errorf("Expected 0 functions initially, got %d", len(module.Functions))
	}

	// Register core functions
	integration.RegisterCoreFunctions()

	// Should now have functions
	if len(module.Functions) == 0 {
		t.Error("No functions registered with module")
	}

	// Check that functions are properly added
	functionNames := make(map[string]bool)
	for _, fn := range module.Functions {
		functionNames[fn.Name] = true
	}

	expectedFunctions := []string{"option_some", "result_ok"}
	for _, expected := range expectedFunctions {
		if !functionNames[expected] {
			t.Errorf("Expected function %s not found in module", expected)
		}
	}
}

// TestMIRModuleIntegrity tests that the MIR module remains valid
func TestMIRModuleIntegrity(t *testing.T) {
	module := &mir.Module{
		Name:      "integrity_test",
		Functions: []*mir.Function{},
	}

	integration := NewCoreTypeMIRIntegration(module)
	integration.RegisterCoreFunctions()

	// Verify module structure
	if module.Name != "integrity_test" {
		t.Error("Module name was modified")
	}

	// Verify all functions have required fields
	for i, fn := range module.Functions {
		if fn.Name == "" {
			t.Errorf("Function %d has empty name", i)
		}
		if fn.Blocks == nil {
			t.Errorf("Function %s has nil blocks", fn.Name)
		}
		if len(fn.Blocks) == 0 {
			t.Errorf("Function %s has no blocks", fn.Name)
		}

		// Verify each block
		for j, block := range fn.Blocks {
			if block.Name == "" {
				t.Errorf("Function %s block %d has empty name", fn.Name, j)
			}
			if block.Instr == nil {
				t.Errorf("Function %s block %s has nil instructions", fn.Name, block.Name)
			}
		}
	}
}

// BenchmarkMIRGeneration benchmarks MIR function generation
func BenchmarkMIRGeneration(b *testing.B) {
	module := &mir.Module{
		Name:      "bench_test",
		Functions: []*mir.Function{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		integration := NewCoreTypeMIRIntegration(module)
		_ = integration.GetAllCoreFunctions()
	}
}

// BenchmarkFunctionRegistration benchmarks function registration
func BenchmarkFunctionRegistration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		module := &mir.Module{
			Name:      "bench_register",
			Functions: []*mir.Function{},
		}
		integration := NewCoreTypeMIRIntegration(module)
		integration.RegisterCoreFunctions()
	}
}
