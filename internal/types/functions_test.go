// Comprehensive test suite for function type system
// Tests closures, higher-order functions, partial application, and async functions

package types

import (
	"fmt"
	"testing"
)

// ====== Phase 2.1.3 Completion Test ======

func TestPhase2_1_3_FunctionTypeSystem(t *testing.T) {
	t.Log("=== Phase 2.1.3: Function Type System Implementation Test ===")

	// Test 1: Closure Types
	t.Run("ClosureTypes", func(t *testing.T) {
		// Create base function type
		baseFunc := NewFunctionType([]*Type{TypeInt32}, TypeInt32, false, false)

		// Create captured variables
		capturedVars := []CapturedVariable{
			{
				Name:        "multiplier",
				Type:        TypeInt32,
				CaptureKind: CaptureByValue,
				Source:      TypeInt32,
			},
			{
				Name:        "counter",
				Type:        TypeInt32,
				CaptureKind: CaptureByReference,
				Source:      TypeInt32,
			},
		}

		// Create closure type
		closureType := NewClosureType(baseFunc.Data.(*FunctionType), capturedVars, CaptureModeImplicit)

		if closureType.Kind != TypeKindFunction {
			t.Errorf("Expected closure type kind to be Function, got %v", closureType.Kind)
		}

		if !closureType.IsCallable() {
			t.Error("Closure type should be callable")
		}

		// Test closure environment
		closureData := closureType.Data.(*ClosureType)
		if len(closureData.CapturedVars) != 2 {
			t.Errorf("Expected 2 captured variables, got %d", len(closureData.CapturedVars))
		}

		// Test capture kinds
		if closureData.CapturedVars[0].CaptureKind != CaptureByValue {
			t.Error("First captured variable should be by value")
		}
		if closureData.CapturedVars[1].CaptureKind != CaptureByReference {
			t.Error("Second captured variable should be by reference")
		}

		// Test purity
		if closureType.IsPure() {
			t.Error("Closure with reference captures should not be pure")
		}
	})

	// Test 2: Higher-Order Function Types
	t.Run("HigherOrderTypes", func(t *testing.T) {
		// Create generic parameters
		tParam := NewGenericType("T", []*Type{}, VarianceInvariant)
		uParam := NewGenericType("U", []*Type{}, VarianceInvariant)

		// Create mapper function type: T -> U
		mapperFunc := NewFunctionType([]*Type{tParam}, uParam, false, false)

		// Create higher-order map function: (T -> U) -> [T] -> [U]
		inputArray := NewSliceType(tParam)
		outputArray := NewSliceType(uParam)

		baseFunc := NewFunctionType([]*Type{mapperFunc, inputArray}, outputArray, false, false)
		hoType := NewHigherOrderType(baseFunc.Data.(*FunctionType), []GenericParameter{
			{Name: "T", Constraints: []*Type{}, Variance: VarianceInvariant},
			{Name: "U", Constraints: []*Type{}, Variance: VarianceInvariant},
		})

		if !hoType.IsCallable() {
			t.Error("Higher-order type should be callable")
		}

		hoData := hoType.Data.(*HigherOrderType)
		if len(hoData.TypeParams) != 2 {
			t.Errorf("Expected 2 type parameters, got %d", len(hoData.TypeParams))
		}

		// Test string representation
		hoStr := hoType.FunctionString()
		if hoStr == "" {
			t.Error("Higher-order function string representation should not be empty")
		}
	})

	// Test 3: Partial Application
	t.Run("PartialApplication", func(t *testing.T) {
		// Create function type: (int, int, int) -> int
		originalFunc := NewFunctionType([]*Type{TypeInt32, TypeInt32, TypeInt32}, TypeInt32, false, false)

		// Partially apply first argument
		partialType, err := originalFunc.ApplyPartially([]*Type{TypeInt32})
		if err != nil {
			t.Fatalf("Failed to create partial application: %v", err)
		}

		if !partialType.IsCallable() {
			t.Error("Partially applied function should be callable")
		}

		// Check remaining arity
		if partialType.GetArity() != 2 {
			t.Errorf("Expected arity 2 after partial application, got %d", partialType.GetArity())
		}

		// Apply second argument
		partialType2, err := partialType.ApplyPartially([]*Type{TypeInt32})
		if err != nil {
			t.Fatalf("Failed to apply second argument: %v", err)
		}

		if partialType2.GetArity() != 1 {
			t.Errorf("Expected arity 1 after second partial application, got %d", partialType2.GetArity())
		}

		// Fully apply
		resultType, err := partialType2.ApplyPartially([]*Type{TypeInt32})
		if err != nil {
			t.Fatalf("Failed to fully apply function: %v", err)
		}

		if resultType.Kind != TypeKindInt32 {
			t.Errorf("Expected result type Int32, got %v", resultType.Kind)
		}
	})

	// Test 4: Function Callability
	t.Run("FunctionCallability", func(t *testing.T) {
		// Create function type: (int, string) -> bool
		funcType := NewFunctionType([]*Type{TypeInt32, TypeString}, TypeBool, false, false)

		// Test valid call
		if !funcType.IsCallableWith([]*Type{TypeInt32, TypeString}) {
			t.Error("Function should be callable with correct argument types")
		}

		// Test invalid call - wrong number of arguments
		if funcType.IsCallableWith([]*Type{TypeInt32}) {
			t.Error("Function should not be callable with too few arguments")
		}

		// Test invalid call - wrong argument types
		if funcType.IsCallableWith([]*Type{TypeString, TypeInt32}) {
			t.Error("Function should not be callable with wrong argument types")
		}

		// Test variadic function
		variadicFunc := NewFunctionType([]*Type{TypeInt32, TypeString}, TypeBool, true, false)

		if !variadicFunc.IsCallableWith([]*Type{TypeInt32, TypeString}) {
			t.Error("Variadic function should be callable with exact arguments")
		}

		if !variadicFunc.IsCallableWith([]*Type{TypeInt32, TypeString, TypeString}) {
			t.Error("Variadic function should be callable with extra arguments")
		}

		if !variadicFunc.IsCallableWith([]*Type{TypeInt32}) {
			t.Error("Variadic function should be callable with minimum arguments (last param is variadic)")
		}
	})

	// Test 5: Async Function Types
	t.Run("AsyncFunctionTypes", func(t *testing.T) {
		// Create base async function
		baseFunc := NewFunctionType([]*Type{TypeString}, TypeInt32, false, true)

		// Create async function type with Promise-like return
		promiseType := NewGenericType("Promise", []*Type{TypeInt32}, VarianceCovariant)
		errorType := NewGenericType("Error", []*Type{}, VarianceInvariant)

		asyncFunc := NewAsyncFunctionType(baseFunc.Data.(*FunctionType), promiseType, errorType)

		if !asyncFunc.IsCallable() {
			t.Error("Async function should be callable")
		}

		asyncData := asyncFunc.Data.(*AsyncFunctionType)
		if asyncData.BaseFunction.ReturnType.Kind != TypeKindInt32 {
			t.Error("Async function base return type should be preserved")
		}

		// Test result type
		resultType, err := asyncFunc.GetCallResultType([]*Type{TypeString})
		if err != nil {
			t.Fatalf("Failed to get async function result type: %v", err)
		}

		if resultType.Kind != TypeKindGeneric {
			t.Error("Async function should return Promise-like type")
		}

		// Test string representation
		asyncStr := asyncFunc.FunctionString()
		if asyncStr == "" {
			t.Error("Async function string representation should not be empty")
		}
	})

	// Test 6: Generator Types
	t.Run("GeneratorTypes", func(t *testing.T) {
		// Create generator: yields int, returns string, accepts bool
		genType := NewGeneratorType(TypeInt32, TypeString, TypeBool)

		if genType.Kind != TypeKindFunction {
			t.Errorf("Expected generator type kind to be Function, got %v", genType.Kind)
		}

		genData := genType.Data.(*GeneratorType)
		if genData.YieldType.Kind != TypeKindInt32 {
			t.Error("Generator yield type should be Int32")
		}
		if genData.ReturnType.Kind != TypeKindString {
			t.Error("Generator return type should be String")
		}
		if genData.SendType.Kind != TypeKindBool {
			t.Error("Generator send type should be Bool")
		}

		// Test string representation
		genStr := genType.FunctionString()
		if genStr == "" {
			t.Error("Generator string representation should not be empty")
		}
	})

	// Test 7: Function Composition
	t.Run("FunctionComposition", func(t *testing.T) {
		// Create functions f: int -> string, g: bool -> int
		f := NewFunctionType([]*Type{TypeInt32}, TypeString, false, false)
		g := NewFunctionType([]*Type{TypeBool}, TypeInt32, false, false)

		// Compose f(g(x)): bool -> string
		composed, err := ComposeFunction(f, g)
		if err != nil {
			t.Fatalf("Failed to compose functions: %v", err)
		}

		if !composed.IsCallable() {
			t.Error("Composed function should be callable")
		}

		composedFunc := composed.Data.(*FunctionType)
		if len(composedFunc.Parameters) != 1 || composedFunc.Parameters[0].Kind != TypeKindBool {
			t.Error("Composed function should take bool parameter")
		}
		if composedFunc.ReturnType.Kind != TypeKindString {
			t.Error("Composed function should return string")
		}

		// Test invalid composition
		h := NewFunctionType([]*Type{TypeString, TypeInt32}, TypeBool, false, false)
		_, err = ComposeFunction(h, g)
		if err == nil {
			t.Error("Should fail to compose functions with incompatible signatures")
		}
	})

	t.Log("✅ Phase 2.1.3: Function Type System - ALL TESTS PASSED")
}

// ====== Closure Analysis Tests ======

func TestClosureAnalysis(t *testing.T) {
	t.Run("ClosureEnvironment", func(t *testing.T) {
		// Create closure environment
		env := &ClosureEnvironment{
			Variables: map[string]*Type{
				"x": TypeInt32,
				"y": TypeString,
				"z": NewSliceType(TypeInt32),
			},
			Parent: nil,
			Size:   0,
		}

		// Create base function
		funcType := NewFunctionType([]*Type{TypeBool}, TypeInt32, false, false)

		// Analyze closure
		freeVars := []string{"x", "y", "z"}
		closureType, err := AnalyzeClosure(funcType.Data.(*FunctionType), env, freeVars)
		if err != nil {
			t.Fatalf("Failed to analyze closure: %v", err)
		}

		closureData := closureType.Data.(*ClosureType)
		if len(closureData.CapturedVars) != 3 {
			t.Errorf("Expected 3 captured variables, got %d", len(closureData.CapturedVars))
		}

		// Check capture kinds
		for _, captured := range closureData.CapturedVars {
			switch captured.Name {
			case "x":
				// Int32 should be captured by value
				if captured.CaptureKind != CaptureByValue {
					t.Errorf("Variable 'x' should be captured by value, got %v", captured.CaptureKind)
				}
			case "y":
				// String should be captured by value
				if captured.CaptureKind != CaptureByValue {
					t.Errorf("Variable 'y' should be captured by value, got %v", captured.CaptureKind)
				}
			case "z":
				// Slice should be captured by reference
				if captured.CaptureKind != CaptureByReference {
					t.Errorf("Variable 'z' should be captured by reference, got %v", captured.CaptureKind)
				}
			}
		}
	})

	t.Run("NestedClosures", func(t *testing.T) {
		// Test nested closure environments
		parentEnv := &ClosureEnvironment{
			Variables: map[string]*Type{
				"outer": TypeInt32,
			},
			Parent: nil,
			Size:   4,
		}

		childEnv := &ClosureEnvironment{
			Variables: map[string]*Type{
				"inner": TypeString,
			},
			Parent: parentEnv,
			Size:   8,
		}

		// Create nested closure
		funcType := NewFunctionType([]*Type{}, TypeVoid, false, false)
		closureType, err := AnalyzeClosure(funcType.Data.(*FunctionType), childEnv, []string{"inner", "outer"})
		if err != nil {
			t.Fatalf("Failed to analyze nested closure: %v", err)
		}

		// Should capture both variables
		closureData := closureType.Data.(*ClosureType)
		if len(closureData.CapturedVars) != 2 {
			t.Errorf("Expected 2 captured variables in nested closure, got %d", len(closureData.CapturedVars))
		}
	})
}

// ====== Higher-Order Function Tests ======

func TestHigherOrderFunctions(t *testing.T) {
	registry := NewFunctionTypeRegistry()

	t.Run("MapFunction", func(t *testing.T) {
		mapType, exists := registry.GetCommonType("map")
		if !exists {
			t.Fatal("Map function type should be registered")
		}

		if !mapType.IsCallable() {
			t.Error("Map function should be callable")
		}

		// Test map string representation
		mapStr := mapType.FunctionString()
		if mapStr == "" {
			t.Error("Map function string representation should not be empty")
		}
	})

	t.Run("FilterFunction", func(t *testing.T) {
		filterType, exists := registry.GetCommonType("filter")
		if !exists {
			t.Fatal("Filter function type should be registered")
		}

		if !filterType.IsCallable() {
			t.Error("Filter function should be callable")
		}

		hoData := filterType.Data.(*HigherOrderType)
		if len(hoData.TypeParams) != 1 {
			t.Errorf("Filter should have 1 type parameter, got %d", len(hoData.TypeParams))
		}
	})

	t.Run("ReduceFunction", func(t *testing.T) {
		reduceType, exists := registry.GetCommonType("reduce")
		if !exists {
			t.Fatal("Reduce function type should be registered")
		}

		if !reduceType.IsCallable() {
			t.Error("Reduce function should be callable")
		}

		hoData := reduceType.Data.(*HigherOrderType)
		if len(hoData.TypeParams) != 2 {
			t.Errorf("Reduce should have 2 type parameters, got %d", len(hoData.TypeParams))
		}

		// Check parameter names
		expectedParams := []string{"T", "Acc"}
		for i, param := range hoData.TypeParams {
			if param.Name != expectedParams[i] {
				t.Errorf("Expected type parameter %s, got %s", expectedParams[i], param.Name)
			}
		}
	})

	t.Run("CommonFunctionTypes", func(t *testing.T) {
		// Test unary int function
		unaryInt, exists := registry.GetCommonType("unary_int")
		if !exists {
			t.Fatal("Unary int function type should be registered")
		}

		if unaryInt.GetArity() != 1 {
			t.Errorf("Unary function should have arity 1, got %d", unaryInt.GetArity())
		}

		// Test binary int function
		binaryInt, exists := registry.GetCommonType("binary_int")
		if !exists {
			t.Fatal("Binary int function type should be registered")
		}

		if binaryInt.GetArity() != 2 {
			t.Errorf("Binary function should have arity 2, got %d", binaryInt.GetArity())
		}

		// Test predicate function
		predicate, exists := registry.GetCommonType("predicate")
		if !exists {
			t.Fatal("Predicate function type should be registered")
		}

		predicateFunc := predicate.Data.(*FunctionType)
		if predicateFunc.ReturnType.Kind != TypeKindBool {
			t.Error("Predicate should return bool")
		}
	})
}

// ====== Function Type Inference Tests ======

func TestFunctionTypeInference(t *testing.T) {
	t.Run("BasicInference", func(t *testing.T) {
		context := &InferenceContext{
			Variables:   make(map[string]*Type),
			Functions:   make(map[string]*Type),
			TypeVars:    make(map[string]*Type),
			Constraints: []Constraint{},
		}

		// Infer simple function type
		params := []*Type{TypeInt32, TypeString}
		returnType := TypeBool

		inferred := InferFunctionType(params, returnType, context)

		if !inferred.IsCallable() {
			t.Error("Inferred type should be callable")
		}

		funcData := getFunctionTypeData(inferred)
		if funcData == nil {
			t.Fatal("Should be able to extract function data")
		}

		if len(funcData.Parameters) != 2 {
			t.Errorf("Expected 2 parameters, got %d", len(funcData.Parameters))
		}

		if funcData.ReturnType.Kind != TypeKindBool {
			t.Error("Return type should be bool")
		}
	})

	t.Run("GenericInference", func(t *testing.T) {
		context := &InferenceContext{
			Variables:   make(map[string]*Type),
			Functions:   make(map[string]*Type),
			TypeVars:    make(map[string]*Type),
			Constraints: []Constraint{},
		}

		// Create generic parameters
		tParam := NewGenericType("T", []*Type{}, VarianceInvariant)

		// Infer generic function type
		params := []*Type{tParam, tParam}
		returnType := tParam

		inferred := InferFunctionType(params, returnType, context)

		// Should create higher-order type
		if inferred.Data == nil {
			t.Fatal("Inferred type should have data")
		}

		if hoData, ok := inferred.Data.(*HigherOrderType); ok {
			if len(hoData.TypeParams) != 1 {
				t.Errorf("Expected 1 type parameter, got %d", len(hoData.TypeParams))
			}
			if hoData.TypeParams[0].Name != "T" {
				t.Errorf("Expected type parameter 'T', got %s", hoData.TypeParams[0].Name)
			}
		} else {
			t.Error("Should infer higher-order type for generic function")
		}
	})
}

// ====== Performance Tests ======

func TestFunctionTypePerformance(t *testing.T) {
	t.Run("ClosureCreation", func(t *testing.T) {
		baseFunc := NewFunctionType([]*Type{TypeInt32}, TypeInt32, false, false)

		// Create many captured variables
		var capturedVars []CapturedVariable
		for i := 0; i < 100; i++ {
			capturedVars = append(capturedVars, CapturedVariable{
				Name:        fmt.Sprintf("var%d", i),
				Type:        TypeInt32,
				CaptureKind: CaptureByValue,
				Source:      TypeInt32,
			})
		}

		// Time closure creation
		for i := 0; i < 1000; i++ {
			closure := NewClosureType(baseFunc.Data.(*FunctionType), capturedVars, CaptureModeImplicit)
			if closure == nil {
				t.Error("Failed to create closure")
			}
		}
	})

	t.Run("PartialApplication", func(t *testing.T) {
		// Create function with many parameters
		var params []*Type
		for i := 0; i < 10; i++ {
			params = append(params, TypeInt32)
		}

		originalFunc := NewFunctionType(params, TypeInt32, false, false)

		// Time partial applications
		current := originalFunc
		for i := 0; i < 9; i++ {
			partial, err := current.ApplyPartially([]*Type{TypeInt32})
			if err != nil {
				t.Fatalf("Failed partial application at step %d: %v", i, err)
			}
			current = partial
		}

		// Final application should return result type
		result, err := current.ApplyPartially([]*Type{TypeInt32})
		if err != nil {
			t.Fatalf("Failed final application: %v", err)
		}

		if result.Kind != TypeKindInt32 {
			t.Error("Final result should be Int32")
		}
	})

	t.Run("FunctionComposition", func(t *testing.T) {
		// Create chain of functions
		f1 := NewFunctionType([]*Type{TypeInt32}, TypeString, false, false)
		f2 := NewFunctionType([]*Type{TypeBool}, TypeInt32, false, false)
		f3 := NewFunctionType([]*Type{TypeFloat64}, TypeBool, false, false)

		// Compose f1(f2(f3(x)))
		composed12, err := ComposeFunction(f1, f2)
		if err != nil {
			t.Fatalf("Failed to compose f1 and f2: %v", err)
		}

		final, err := ComposeFunction(composed12, f3)
		if err != nil {
			t.Fatalf("Failed to compose with f3: %v", err)
		}

		// Check final type: float64 -> string
		finalFunc := final.Data.(*FunctionType)
		if finalFunc.Parameters[0].Kind != TypeKindFloat64 {
			t.Error("Final composed function should take float64")
		}
		if finalFunc.ReturnType.Kind != TypeKindString {
			t.Error("Final composed function should return string")
		}
	})
}

// ====== Error Handling Tests ======

func TestFunctionTypeErrors(t *testing.T) {
	t.Run("InvalidPartialApplication", func(t *testing.T) {
		// Try to partially apply non-function
		nonFunc := TypeInt32
		_, err := nonFunc.ApplyPartially([]*Type{TypeString})
		if err == nil {
			t.Error("Should fail to partially apply non-function type")
		}
	})

	t.Run("InvalidComposition", func(t *testing.T) {
		// Try to compose incompatible functions
		f1 := NewFunctionType([]*Type{TypeString}, TypeInt32, false, false)
		f2 := NewFunctionType([]*Type{TypeBool}, TypeFloat64, false, false)

		_, err := ComposeFunction(f1, f2)
		if err == nil {
			t.Error("Should fail to compose incompatible functions")
		}
	})

	t.Run("InvalidCallability", func(t *testing.T) {
		funcType := NewFunctionType([]*Type{TypeInt32, TypeString}, TypeBool, false, false)

		// Test with wrong argument count
		if funcType.IsCallableWith([]*Type{TypeInt32}) {
			t.Error("Should not be callable with too few arguments")
		}

		if funcType.IsCallableWith([]*Type{TypeInt32, TypeString, TypeBool}) {
			t.Error("Should not be callable with too many arguments")
		}

		// Test with wrong argument types
		if funcType.IsCallableWith([]*Type{TypeString, TypeInt32}) {
			t.Error("Should not be callable with wrong argument types")
		}
	})
}

// ====== Integration Tests ======

func TestFunctionTypeIntegration(t *testing.T) {
	t.Run("ComplexFunctionPipeline", func(t *testing.T) {
		registry := NewFunctionTypeRegistry()

		// Create a complex function pipeline using higher-order functions
		mapType, _ := registry.GetCommonType("map")
		filterType, _ := registry.GetCommonType("filter")
		reduceType, _ := registry.GetCommonType("reduce")

		// All should be callable
		if !mapType.IsCallable() || !filterType.IsCallable() || !reduceType.IsCallable() {
			t.Error("All higher-order functions should be callable")
		}

		// Test string representations
		mapStr := mapType.FunctionString()
		filterStr := filterType.FunctionString()
		reduceStr := reduceType.FunctionString()

		if mapStr == "" || filterStr == "" || reduceStr == "" {
			t.Error("All function string representations should be non-empty")
		}

		t.Logf("Map type: %s", mapStr)
		t.Logf("Filter type: %s", filterStr)
		t.Logf("Reduce type: %s", reduceStr)
	})

	t.Run("ClosureWithPartialApplication", func(t *testing.T) {
		// Create closure
		baseFunc := NewFunctionType([]*Type{TypeInt32, TypeInt32}, TypeInt32, false, false)
		capturedVars := []CapturedVariable{
			{Name: "multiplier", Type: TypeInt32, CaptureKind: CaptureByValue, Source: TypeInt32},
		}

		closureType := NewClosureType(baseFunc.Data.(*FunctionType), capturedVars, CaptureModeImplicit)

		// Partially apply closure
		partialClosure, err := closureType.ApplyPartially([]*Type{TypeInt32})
		if err != nil {
			t.Fatalf("Failed to partially apply closure: %v", err)
		}

		if partialClosure.GetArity() != 1 {
			t.Errorf("Partially applied closure should have arity 1, got %d", partialClosure.GetArity())
		}

		// Check that it's still callable
		if !partialClosure.IsCallable() {
			t.Error("Partially applied closure should still be callable")
		}
	})
}
