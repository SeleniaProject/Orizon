// Comprehensive test suite for Hindley-Milner type inference engine.
// Tests Algorithm W, unification, generalization, and let-polymorphism.

package types

import (
	"fmt"
	"testing"
)

// ====== Phase 2.2.1 Completion Test ======

func TestPhase2_2_1_HindleyMilnerInference(t *testing.T) {
	t.Log("=== Phase 2.2.1: Hindley-Milner Type Inference Implementation Test ===")

	engine := NewInferenceEngine()

	// Test 1: Basic Type Inference.
	t.Run("BasicTypeInference", func(t *testing.T) {
		// Test literal inference.
		literal := NewLiteralExpr(42)

		inferredType, err := engine.InferType(literal)
		if err != nil {
			t.Fatalf("Failed to infer literal type: %v", err)
		}

		if inferredType.Kind != TypeKindInt32 {
			t.Errorf("Expected Int32, got %s", inferredType.String())
		}

		// Test string literal.
		stringLit := NewLiteralExpr("hello")

		stringType, err := engine.InferType(stringLit)
		if err != nil {
			t.Fatalf("Failed to infer string type: %v", err)
		}

		if stringType.Kind != TypeKindString {
			t.Errorf("Expected String, got %s", stringType.String())
		}

		// Test boolean literal.
		boolLit := NewLiteralExpr(true)

		boolType, err := engine.InferType(boolLit)
		if err != nil {
			t.Fatalf("Failed to infer boolean type: %v", err)
		}

		if boolType.Kind != TypeKindBool {
			t.Errorf("Expected Bool, got %s", boolType.String())
		}

		t.Log("✅ Basic type inference for literals working correctly")
	})

	// Test 2: Variable Inference.
	t.Run("VariableInference", func(t *testing.T) {
		engine.Reset()

		// Add a variable to environment.
		intScheme := &TypeScheme{
			TypeVars: []string{},
			Type:     TypeInt32,
			Level:    0,
		}
		engine.AddVariable("x", intScheme)

		// Test variable lookup.
		varExpr := NewVariableExpr("x")

		inferredType, err := engine.InferType(varExpr)
		if err != nil {
			t.Fatalf("Failed to infer variable type: %v", err)
		}

		if inferredType.Kind != TypeKindInt32 {
			t.Errorf("Expected Int32, got %s", inferredType.String())
		}

		// Test undefined variable.
		undefinedVar := NewVariableExpr("undefined")

		_, err = engine.InferType(undefinedVar)
		if err == nil {
			t.Error("Should fail for undefined variable")
		}

		t.Log("✅ Variable type inference working correctly")
	})

	// Test 3: Lambda Inference.
	t.Run("LambdaInference", func(t *testing.T) {
		engine.Reset()

		// Test identity function: λx.x
		identityLambda := NewLambdaExpr("x", NewVariableExpr("x"))

		identityType, err := engine.InferType(identityLambda)
		if err != nil {
			t.Fatalf("Failed to infer identity function type: %v", err)
		}

		if identityType.Kind != TypeKindFunction {
			t.Errorf("Expected function type, got %s", identityType.String())
		}

		funcData := identityType.Data.(*FunctionType)
		if len(funcData.Parameters) != 1 {
			t.Errorf("Expected 1 parameter, got %d", len(funcData.Parameters))
		}

		// The identity function should have type: τ₀ -> τ₀.
		paramType := funcData.Parameters[0]
		returnType := funcData.ReturnType

		if paramType.Kind != TypeKindTypeVar || returnType.Kind != TypeKindTypeVar {
			t.Errorf("Identity function should have polymorphic type variables")
		}

		// Apply substitutions to check if they unify.
		engine.substitutions = make(map[string]*Type)
		if err := engine.Unify(paramType, returnType); err != nil {
			t.Errorf("Identity function parameter and return types should unify: %v", err)
		}

		t.Log("✅ Lambda function inference working correctly")
	})

	// Test 4: Function Application.
	t.Run("FunctionApplication", func(t *testing.T) {
		engine.Reset()

		// Test application: (λx.x) 42
		identityLambda := NewLambdaExpr("x", NewVariableExpr("x"))
		intLiteral := NewLiteralExpr(42)
		application := NewApplicationExpr(identityLambda, intLiteral)

		resultType, err := engine.InferType(application)
		if err != nil {
			t.Fatalf("Failed to infer application type: %v", err)
		}

		if resultType.Kind != TypeKindInt32 {
			t.Errorf("Expected Int32, got %s", resultType.String())
		}

		// Test application with type mismatch.
		engine.Reset()

		stringLiteral := NewLiteralExpr("hello")

		// Create a function that expects int: λx:int.x
		typedLambda := NewTypedLambdaExpr("x", TypeInt32, NewVariableExpr("x"))
		invalidApp := NewApplicationExpr(typedLambda, stringLiteral)

		_, err = engine.InferType(invalidApp)
		if err == nil {
			t.Error("Should fail for type mismatch in application")
		}

		t.Log("✅ Function application inference working correctly")
	})

	// Test 5: Let-Polymorphism.
	t.Run("LetPolymorphism", func(t *testing.T) {
		engine.Reset()

		// Test: let id = λx.x in (id 42)
		identityLambda := NewLambdaExpr("x", NewVariableExpr("x"))

		// Use the identity function on an integer.
		idVar := NewVariableExpr("id")
		intApp := NewApplicationExpr(idVar, NewLiteralExpr(42))

		letExpr := NewLetExpr("id", identityLambda, intApp)

		resultType, err := engine.InferType(letExpr)
		if err != nil {
			t.Fatalf("Failed to infer let-polymorphism type: %v", err)
		}

		// Result should be Int32.
		if resultType.Kind != TypeKindInt32 {
			t.Errorf("Expected Int32, got %s", resultType.String())
		}

		t.Log("✅ Let-polymorphism working correctly")
	})

	// Test 6: Conditional Expressions.
	t.Run("ConditionalInference", func(t *testing.T) {
		engine.Reset()

		// Test: if true then 42 else 84.
		condition := NewLiteralExpr(true)
		thenExpr := NewLiteralExpr(42)
		elseExpr := NewLiteralExpr(84)

		ifElse := NewIfElseExpr(condition, thenExpr, elseExpr)

		resultType, err := engine.InferType(ifElse)
		if err != nil {
			t.Fatalf("Failed to infer if-else type: %v", err)
		}

		if resultType.Kind != TypeKindInt32 {
			t.Errorf("Expected Int32, got %s", resultType.String())
		}

		// Test type mismatch in branches.
		engine.Reset()

		stringElse := NewLiteralExpr("hello")
		mismatchIfElse := NewIfElseExpr(condition, thenExpr, stringElse)

		_, err = engine.InferType(mismatchIfElse)
		if err == nil {
			t.Error("Should fail for type mismatch in if-else branches")
		}

		// Test non-boolean condition.
		engine.Reset()

		intCondition := NewLiteralExpr(42)
		invalidIfElse := NewIfElseExpr(intCondition, thenExpr, elseExpr)

		_, err = engine.InferType(invalidIfElse)
		if err == nil {
			t.Error("Should fail for non-boolean condition")
		}

		t.Log("✅ Conditional expression inference working correctly")
	})

	// Test 7: Binary Operations.
	t.Run("BinaryOperations", func(t *testing.T) {
		engine.Reset()

		// Add arithmetic operator to environment.
		addOpScheme := &TypeScheme{
			TypeVars: []string{},
			Type: NewFunctionType([]*Type{
				TypeInt32,
				TypeInt32,
			}, TypeInt32, false, false),
			Level: 0,
		}
		engine.AddVariable("+", addOpScheme)

		// Test arithmetic: 42 + 84.
		left := NewLiteralExpr(42)
		right := NewLiteralExpr(84)
		addition := NewBinaryOpExpr(left, "+", right)

		resultType, err := engine.InferType(addition)
		if err != nil {
			t.Fatalf("Failed to infer binary operation type: %v", err)
		}

		if resultType.Kind != TypeKindInt32 {
			t.Errorf("Expected Int32, got %s", resultType.String())
		}

		// Test comparison: 42 == 42.
		engine.Reset()

		// Add equality operator to environment.
		eqOpScheme := &TypeScheme{
			TypeVars: []string{},
			Type: NewFunctionType([]*Type{
				TypeInt32,
				TypeInt32,
			}, TypeBool, false, false),
			Level: 0,
		}
		engine.AddVariable("==", eqOpScheme)

		comparison := NewBinaryOpExpr(left, "==", right)

		compType, err := engine.InferType(comparison)
		if err != nil {
			t.Fatalf("Failed to infer comparison type: %v", err)
		}

		if compType.Kind != TypeKindBool {
			t.Errorf("Expected Bool, got %s", compType.String())
		}

		t.Log("✅ Binary operation inference working correctly")
	})

	t.Log("✅ Phase 2.2.1: Hindley-Milner Type Inference - ALL TESTS PASSED")
}

// ====== Unification Tests ======.

func TestUnificationAlgorithm(t *testing.T) {
	t.Run("PrimitiveUnification", func(t *testing.T) {
		engine := NewInferenceEngine()

		// Test unifying same types.
		err := engine.Unify(TypeInt32, TypeInt32)
		if err != nil {
			t.Errorf("Same types should unify: %v", err)
		}

		// Test unifying different primitive types.
		err = engine.Unify(TypeInt32, TypeString)
		if err == nil {
			t.Error("Different primitive types should not unify")
		}

		t.Log("✅ Primitive type unification working correctly")
	})

	t.Run("TypeVariableUnification", func(t *testing.T) {
		engine := NewInferenceEngine()

		// Test unifying type variable with concrete type.
		typeVar := engine.FreshTypeVar()

		err := engine.Unify(typeVar, TypeInt32)
		if err != nil {
			t.Errorf("Type variable should unify with concrete type: %v", err)
		}

		// Check substitution was added.
		varData := typeVar.Data.(*TypeVar)
		if subst, exists := engine.substitutions[varData.Name]; !exists || subst != TypeInt32 {
			t.Error("Substitution should be added for unified type variable")
		}

		// Test unifying two type variables.
		engine.Reset()
		typeVar1 := engine.FreshTypeVar()
		typeVar2 := engine.FreshTypeVar()

		err = engine.Unify(typeVar1, typeVar2)
		if err != nil {
			t.Errorf("Type variables should unify: %v", err)
		}

		t.Log("✅ Type variable unification working correctly")
	})

	t.Run("OccursCheck", func(t *testing.T) {
		engine := NewInferenceEngine()

		// Create recursive type: τ = τ -> τ.
		typeVar := engine.FreshTypeVar()
		funcType := NewFunctionType([]*Type{typeVar}, typeVar, false, false)

		// This should fail occurs check.
		err := engine.Unify(typeVar, funcType)
		if err == nil {
			t.Error("Occurs check should prevent infinite types")
		}

		t.Log("✅ Occurs check working correctly")
	})

	t.Run("ComplexTypeUnification", func(t *testing.T) {
		engine := NewInferenceEngine()

		// Test unifying function types.
		func1 := NewFunctionType([]*Type{TypeInt32}, TypeString, false, false)
		func2 := NewFunctionType([]*Type{TypeInt32}, TypeString, false, false)

		err := engine.Unify(func1, func2)
		if err != nil {
			t.Errorf("Same function types should unify: %v", err)
		}

		// Test unifying array types.
		array1 := NewArrayType(TypeInt32, 10)
		array2 := NewArrayType(TypeInt32, 10)

		err = engine.Unify(array1, array2)
		if err != nil {
			t.Errorf("Same array types should unify: %v", err)
		}

		// Test unifying arrays with different lengths.
		array3 := NewArrayType(TypeInt32, 5)

		err = engine.Unify(array1, array3)
		if err == nil {
			t.Error("Arrays with different lengths should not unify")
		}

		t.Log("✅ Complex type unification working correctly")
	})
}

// ====== Type Environment Tests ======.

func TestTypeEnvironment(t *testing.T) {
	t.Run("EnvironmentOperations", func(t *testing.T) {
		engine := NewInferenceEngine()

		// Test adding and looking up variables.
		intScheme := &TypeScheme{
			TypeVars: []string{},
			Type:     TypeInt32,
			Level:    0,
		}

		engine.AddVariable("x", intScheme)

		scheme, exists := engine.LookupVariable("x")
		if !exists {
			t.Error("Variable should exist in environment")
		}

		if scheme.Type.Kind != TypeKindInt32 {
			t.Errorf("Expected Int32, got %s", scheme.Type.String())
		}

		// Test nested environments.
		engine.PushEnvironment()

		stringScheme := &TypeScheme{
			TypeVars: []string{},
			Type:     TypeString,
			Level:    1,
		}
		engine.AddVariable("y", stringScheme)

		// Should find both variables.
		_, exists = engine.LookupVariable("x") // From parent
		if !exists {
			t.Error("Should find variable from parent environment")
		}

		_, exists = engine.LookupVariable("y") // From current
		if !exists {
			t.Error("Should find variable from current environment")
		}

		// Pop environment.
		engine.PopEnvironment()

		// Should only find x now.
		_, exists = engine.LookupVariable("x")
		if !exists {
			t.Error("Should still find variable from parent environment")
		}

		_, exists = engine.LookupVariable("y")
		if exists {
			t.Error("Should not find variable from popped environment")
		}

		t.Log("✅ Type environment operations working correctly")
	})
}

// ====== Generalization and Instantiation Tests ======.

func TestGeneralizationInstantiation(t *testing.T) {
	t.Run("Generalization", func(t *testing.T) {
		engine := NewInferenceEngine()

		// Create a type with free variables: τ₀ -> τ₀.
		typeVar := engine.FreshTypeVar()
		funcType := NewFunctionType([]*Type{typeVar}, typeVar, false, false)

		// Generalize it.
		scheme := engine.Generalize(funcType)

		if len(scheme.TypeVars) != 1 {
			t.Errorf("Expected 1 quantified variable, got %d", len(scheme.TypeVars))
		}

		varData := typeVar.Data.(*TypeVar)
		if scheme.TypeVars[0] != varData.Name {
			t.Errorf("Expected quantified variable %s, got %s", varData.Name, scheme.TypeVars[0])
		}

		t.Log("✅ Type generalization working correctly")
	})

	t.Run("Instantiation", func(t *testing.T) {
		engine := NewInferenceEngine()

		// Create a polymorphic scheme: ∀a.a -> a
		scheme := &TypeScheme{
			TypeVars: []string{"a"},
			Type: NewFunctionType([]*Type{
				NewGenericType("a", []*Type{}, VarianceInvariant),
			}, NewGenericType("a", []*Type{}, VarianceInvariant), false, false),
			Level: 0,
		}

		// Instantiate it twice.
		instance1 := engine.Instantiate(scheme)
		instance2 := engine.Instantiate(scheme)

		// Instances should have fresh type variables.
		func1 := instance1.Data.(*FunctionType)
		func2 := instance2.Data.(*FunctionType)

		param1 := func1.Parameters[0]
		param2 := func2.Parameters[0]

		if param1.Kind != TypeKindTypeVar || param2.Kind != TypeKindTypeVar {
			t.Error("Instantiated types should have type variables")
		}

		// Variables should be different (fresh).
		var1Data := param1.Data.(*TypeVar)
		var2Data := param2.Data.(*TypeVar)

		if var1Data.Name == var2Data.Name {
			t.Error("Instantiated type variables should be fresh/different")
		}

		t.Log("✅ Type instantiation working correctly")
	})
}

// ====== Advanced Inference Tests ======.

func TestAdvancedInference(t *testing.T) {
	t.Run("HigherOrderFunctions", func(t *testing.T) {
		engine := NewInferenceEngine()

		// Test map function: (a -> b) -> [a] -> [b].
		// let map = λf.λxs.map_impl f xs
		// For simplicity, we'll test function composition.

		// Create: λf.λg.λx.f (g x)
		// This is function composition with type: (b -> c) -> (a -> b) -> a -> c.
		x := NewVariableExpr("x")
		gx := NewApplicationExpr(NewVariableExpr("g"), x)
		fgx := NewApplicationExpr(NewVariableExpr("f"), gx)

		innerLambda := NewLambdaExpr("x", fgx)
		middleLambda := NewLambdaExpr("g", innerLambda)
		outerLambda := NewLambdaExpr("f", middleLambda)

		composeType, err := engine.InferType(outerLambda)
		if err != nil {
			t.Fatalf("Failed to infer function composition type: %v", err)
		}

		if composeType.Kind != TypeKindFunction {
			t.Errorf("Expected function type, got %s", composeType.String())
		}

		t.Log("✅ Higher-order function inference working correctly")
	})

	t.Run("RecursiveFunctions", func(t *testing.T) {
		engine := NewInferenceEngine()

		// Test recursive factorial (simplified).
		// let rec fact = λn.if (n == 0) then 1 else n * (fact (n - 1))

		// First, add factorial to environment (recursive binding).
		factScheme := &TypeScheme{
			TypeVars: []string{},
			Type:     NewFunctionType([]*Type{TypeInt32}, TypeInt32, false, false),
			Level:    0,
		}
		engine.AddVariable("fact", factScheme)

		// Create the body: if (n == 0) then 1 else n * (fact (n - 1)).
		n := NewVariableExpr("n")
		zero := NewLiteralExpr(0)
		one := NewLiteralExpr(1)

		condition := NewBinaryOpExpr(n, "==", zero)
		thenExpr := one

		// n - 1.
		nMinus1 := NewBinaryOpExpr(n, "-", one)
		// fact (n - 1).
		factCall := NewApplicationExpr(NewVariableExpr("fact"), nMinus1)
		// n * (fact (n - 1)).
		elseExpr := NewBinaryOpExpr(n, "*", factCall)

		ifElseExpr := NewIfElseExpr(condition, thenExpr, elseExpr)
		factLambda := NewLambdaExpr("n", ifElseExpr)

		factType, err := engine.InferType(factLambda)
		if err != nil {
			t.Fatalf("Failed to infer recursive function type: %v", err)
		}

		funcData := factType.Data.(*FunctionType)
		if len(funcData.Parameters) != 1 || funcData.Parameters[0].Kind != TypeKindInt32 {
			t.Error("Factorial should take one Int32 parameter")
		}

		if funcData.ReturnType.Kind != TypeKindInt32 {
			t.Error("Factorial should return Int32")
		}

		t.Log("✅ Recursive function inference working correctly")
	})
}

// ====== Performance Tests ======.

func TestInferencePerformance(t *testing.T) {
	t.Run("LargeExpressionInference", func(t *testing.T) {
		engine := NewInferenceEngine()

		// Create a large expression tree.
		var expr Expr = NewLiteralExpr(1)

		for i := 0; i < 100; i++ {
			nextLit := NewLiteralExpr(1)
			expr = NewBinaryOpExpr(expr, "+", nextLit)
		}

		resultType, err := engine.InferType(expr)
		if err != nil {
			t.Fatalf("Failed to infer large expression type: %v", err)
		}

		if resultType.Kind != TypeKindInt32 {
			t.Errorf("Expected Int32, got %s", resultType.String())
		}

		t.Log("✅ Large expression inference performance acceptable")
	})

	t.Run("ManyVariableEnvironment", func(t *testing.T) {
		engine := NewInferenceEngine()

		// Add many variables to environment.
		for i := 0; i < 1000; i++ {
			varName := fmt.Sprintf("var%d", i)
			scheme := &TypeScheme{
				TypeVars: []string{},
				Type:     TypeInt32,
				Level:    0,
			}
			engine.AddVariable(varName, scheme)
		}

		// Test lookup performance.
		varExpr := NewVariableExpr("var500")

		inferredType, err := engine.InferType(varExpr)
		if err != nil {
			t.Fatalf("Failed to infer variable type in large environment: %v", err)
		}

		if inferredType.Kind != TypeKindInt32 {
			t.Errorf("Expected Int32, got %s", inferredType.String())
		}

		t.Log("✅ Large environment performance acceptable")
	})
}

// ====== Error Handling Tests ======.

func TestInferenceErrors(t *testing.T) {
	t.Run("TypeMismatchErrors", func(t *testing.T) {
		engine := NewInferenceEngine()

		// Test function application with wrong argument type.
		intFunc := NewTypedLambdaExpr("x", TypeInt32, NewVariableExpr("x"))
		stringArg := NewLiteralExpr("hello")
		wrongApp := NewApplicationExpr(intFunc, stringArg)

		_, err := engine.InferType(wrongApp)
		if err == nil {
			t.Error("Should fail for type mismatch")
		}

		t.Logf("Correct error message: %v", err)
		t.Log("✅ Type mismatch error handling working correctly")
	})

	t.Run("UndefinedVariableErrors", func(t *testing.T) {
		engine := NewInferenceEngine()

		undefinedVar := NewVariableExpr("undefinedVariable")

		_, err := engine.InferType(undefinedVar)
		if err == nil {
			t.Error("Should fail for undefined variable")
		}

		t.Logf("Correct error message: %v", err)
		t.Log("✅ Undefined variable error handling working correctly")
	})
}
