package hir

import (
	"testing"
)

// BenchmarkTypeEquality benchmarks type equality operations
func BenchmarkTypeEquality(b *testing.B) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}
	intType2 := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := intType.Equals(intType2)
		if !result {
			b.Error("Types should be equal")
		}
	}
}

// BenchmarkTypeConversion benchmarks type conversion checks
func BenchmarkTypeConversion(b *testing.B) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}
	floatType := TypeInfo{Kind: TypeKindFloat, Name: "float", Size: 4, Alignment: 4}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := intType.CanConvertTo(floatType)
		if !result {
			b.Error("Int should convert to float")
		}
	}
}

// BenchmarkStructLayout benchmarks struct layout calculation
func BenchmarkStructLayout(b *testing.B) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}
	floatType := TypeInfo{Kind: TypeKindFloat, Name: "float", Size: 4, Alignment: 4}

	fields := []FieldInfo{
		{Name: "x", Type: intType},
		{Name: "y", Type: floatType},
		{Name: "z", Type: intType},
		{Name: "w", Type: floatType},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		layout := CalculateStructLayout(fields)
		if layout.Size <= 0 {
			b.Error("Invalid layout size")
		}
	}
}

// BenchmarkFunctionTypeCreation benchmarks function type creation
func BenchmarkFunctionTypeCreation(b *testing.B) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}

	signature := FunctionSignature{
		Name: "testFunc",
		Parameters: []Parameter{
			{Name: "a", Type: intType},
			{Name: "b", Type: intType},
			{Name: "c", Type: intType},
		},
		ReturnType: intType,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		funcType := CreateFunctionType(signature)
		if funcType.Kind != TypeKindFunction {
			b.Error("Invalid function type")
		}
	}
}

// BenchmarkClosureLayout benchmarks closure layout calculation
func BenchmarkClosureLayout(b *testing.B) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}
	floatType := TypeInfo{Kind: TypeKindFloat, Name: "float", Size: 4, Alignment: 4}

	env := ClosureEnvironment{
		CapturedVars: []CapturedVariable{
			{Name: "x", Type: intType, Mode: CaptureByValue},
			{Name: "y", Type: floatType, Mode: CaptureByValue},
			{Name: "z", Type: intType, Mode: CaptureByValue},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		layout := CalculateClosureLayout(env)
		if layout.TotalSize <= 0 {
			b.Error("Invalid closure layout")
		}
	}
}

// BenchmarkComplexTypeOperations benchmarks complex type operations
func BenchmarkComplexTypeOperations(b *testing.B) {
	intType := TypeInfo{Kind: TypeKindInteger, Name: "int", Size: 4, Alignment: 4}
	floatType := TypeInfo{Kind: TypeKindFloat, Name: "float", Size: 4, Alignment: 4}

	// Create struct type
	fields := []FieldInfo{
		{Name: "x", Type: intType},
		{Name: "y", Type: floatType},
	}
	structType := CreateStructType("Point", fields)

	// Create array type
	arrayType := TypeInfo{Kind: TypeKindArray, Name: "[]int", Parameters: []TypeInfo{intType}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Multiple operations
		_ = structType.Equals(structType)
		_ = arrayType.CanConvertTo(arrayType)
		_ = CalculateStructLayout(fields)

		// Function type operations
		signature := FunctionSignature{
			Name: "process",
			Parameters: []Parameter{
				{Name: "data", Type: structType},
				{Name: "arr", Type: arrayType},
			},
			ReturnType: intType,
		}
		_ = CreateFunctionType(signature)
	}
}
