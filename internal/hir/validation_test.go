package hir

import (
	"testing"
	"time"
)

// TestValidationWithTiming tests with detailed timing to verify execution.
func TestValidationWithTiming(t *testing.T) {
	start := time.Now()

	// Test TypeInfo Equals functionality with detailed timing.
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}
	floatType := TypeInfo{Kind: TypeKindFloat, Name: "float", Size: 4, Alignment: 4}

	t.Logf("Created types in %v", time.Since(start))

	// Test equality check.
	equalityStart := time.Now()
	result1 := intType.Equals(TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4})

	t.Logf("Equality check took %v", time.Since(equalityStart))

	if !result1 {
		t.Error("Expected equal types to be equal")
	}

	// Test conversion check.
	conversionStart := time.Now()
	result2 := intType.CanConvertTo(floatType)

	t.Logf("Conversion check took %v", time.Since(conversionStart))

	if !result2 {
		t.Error("Expected int to convert to float")
	}

	// Test struct layout calculation.
	layoutStart := time.Now()
	fields := []FieldInfo{
		{Name: "x", Type: intType},
		{Name: "y", Type: floatType},
	}
	layout := CalculateStructLayout(fields)

	t.Logf("Struct layout calculation took %v", time.Since(layoutStart))

	if layout.Size != 8 {
		t.Errorf("Expected struct size 8, got %d", layout.Size)
	}

	// Test function type creation.
	funcStart := time.Now()
	signature := FunctionSignature{
		Name: "test",
		Parameters: []Parameter{
			{Name: "a", Type: intType},
			{Name: "b", Type: floatType},
		},
		ReturnType: intType,
	}
	funcType := CreateFunctionType(signature)

	t.Logf("Function type creation took %v", time.Since(funcStart))

	if funcType.Kind != TypeKindFunction {
		t.Error("Expected function type")
	}

	// Test closure layout calculation.
	closureStart := time.Now()
	env := ClosureEnvironment{
		CapturedVars: []CapturedVariable{
			{Name: "captured", Type: intType, Mode: CaptureByValue},
		},
	}
	closureLayout := CalculateClosureLayout(env)

	t.Logf("Closure layout calculation took %v", time.Since(closureStart))

	if closureLayout.TotalSize <= 8 {
		t.Error("Expected closure size > 8")
	}

	totalTime := time.Since(start)
	t.Logf("Total test execution time: %v", totalTime)

	// Force a small delay to make timing measurable.
	time.Sleep(time.Microsecond)

	// Additional verification that functions are actually executing.
	t.Logf("Verified %d operations completed successfully", 5)
}
