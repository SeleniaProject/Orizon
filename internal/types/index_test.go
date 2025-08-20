// Package types tests for Phase 2.3.2 Index Types implementation
package types

import (
	"testing"
)

func TestPhase2_3_2_IndexTypes(t *testing.T) {
	t.Run("ConstantIndexExpressions", func(t *testing.T) {
		// Test constant index expression.
		constExpr := &ConstantIndexExpr{Value: 42}

		if constExpr.String() != "42" {
			t.Errorf("expected '42', got '%s'", constExpr.String())
		}

		val, err := constExpr.Evaluate(nil)
		if err != nil {
			t.Fatalf("constant evaluation failed: %v", err)
		}

		if val != 42 {
			t.Errorf("expected 42, got %d", val)
		}

		if !constExpr.IsStatic() {
			t.Error("constant expression should be static")
		}

		vars := constExpr.Variables()
		if len(vars) != 0 {
			t.Errorf("constant should have no variables, got %v", vars)
		}
	})

	t.Run("VariableIndexExpressions", func(t *testing.T) {
		// Test variable index expression.
		varExpr := &VariableIndexExpr{Name: "i"}

		if varExpr.String() != "i" {
			t.Errorf("expected 'i', got '%s'", varExpr.String())
		}

		if varExpr.IsStatic() {
			t.Error("variable expression should not be static")
		}

		vars := varExpr.Variables()
		if len(vars) != 1 || vars[0] != "i" {
			t.Errorf("expected variables [i], got %v", vars)
		}

		// Test evaluation with environment.
		env := map[string]interface{}{
			"i": int64(10),
		}

		val, err := varExpr.Evaluate(env)
		if err != nil {
			t.Fatalf("variable evaluation failed: %v", err)
		}

		if val != 10 {
			t.Errorf("expected 10, got %d", val)
		}

		// Test evaluation without environment.
		_, err = varExpr.Evaluate(nil)
		if err == nil {
			t.Error("expected error for undefined variable")
		}
	})

	t.Run("BinaryIndexExpressions", func(t *testing.T) {
		// Test addition: 5 + 3.
		left := &ConstantIndexExpr{Value: 5}
		right := &ConstantIndexExpr{Value: 3}
		addExpr := &BinaryIndexExpr{
			Left:     left,
			Operator: IndexOpAdd,
			Right:    right,
		}

		if addExpr.String() != "(5 + 3)" {
			t.Errorf("expected '(5 + 3)', got '%s'", addExpr.String())
		}

		val, err := addExpr.Evaluate(nil)
		if err != nil {
			t.Fatalf("addition evaluation failed: %v", err)
		}

		if val != 8 {
			t.Errorf("expected 8, got %d", val)
		}

		if !addExpr.IsStatic() {
			t.Error("constant addition should be static")
		}

		// Test subtraction: 10 - 4.
		subExpr := &BinaryIndexExpr{
			Left:     &ConstantIndexExpr{Value: 10},
			Operator: IndexOpSub,
			Right:    &ConstantIndexExpr{Value: 4},
		}

		val, err = subExpr.Evaluate(nil)
		if err != nil {
			t.Fatalf("subtraction evaluation failed: %v", err)
		}

		if val != 6 {
			t.Errorf("expected 6, got %d", val)
		}

		// Test multiplication: 3 * 4.
		mulExpr := &BinaryIndexExpr{
			Left:     &ConstantIndexExpr{Value: 3},
			Operator: IndexOpMul,
			Right:    &ConstantIndexExpr{Value: 4},
		}

		val, err = mulExpr.Evaluate(nil)
		if err != nil {
			t.Fatalf("multiplication evaluation failed: %v", err)
		}

		if val != 12 {
			t.Errorf("expected 12, got %d", val)
		}

		// Test division: 15 / 3
		divExpr := &BinaryIndexExpr{
			Left:     &ConstantIndexExpr{Value: 15},
			Operator: IndexOpDiv,
			Right:    &ConstantIndexExpr{Value: 3},
		}

		val, err = divExpr.Evaluate(nil)
		if err != nil {
			t.Fatalf("division evaluation failed: %v", err)
		}

		if val != 5 {
			t.Errorf("expected 5, got %d", val)
		}

		// Test division by zero.
		divZeroExpr := &BinaryIndexExpr{
			Left:     &ConstantIndexExpr{Value: 10},
			Operator: IndexOpDiv,
			Right:    &ConstantIndexExpr{Value: 0},
		}

		_, err = divZeroExpr.Evaluate(nil)
		if err == nil {
			t.Error("expected error for division by zero")
		}

		// Test modulo: 17 % 5.
		modExpr := &BinaryIndexExpr{
			Left:     &ConstantIndexExpr{Value: 17},
			Operator: IndexOpMod,
			Right:    &ConstantIndexExpr{Value: 5},
		}

		val, err = modExpr.Evaluate(nil)
		if err != nil {
			t.Fatalf("modulo evaluation failed: %v", err)
		}

		if val != 2 {
			t.Errorf("expected 2, got %d", val)
		}

		// Test min function: min(7, 3).
		minExpr := &BinaryIndexExpr{
			Left:     &ConstantIndexExpr{Value: 7},
			Operator: IndexOpMin,
			Right:    &ConstantIndexExpr{Value: 3},
		}

		if minExpr.String() != "min(7, 3)" {
			t.Errorf("expected 'min(7, 3)', got '%s'", minExpr.String())
		}

		val, err = minExpr.Evaluate(nil)
		if err != nil {
			t.Fatalf("min evaluation failed: %v", err)
		}

		if val != 3 {
			t.Errorf("expected 3, got %d", val)
		}

		// Test max function: max(7, 3).
		maxExpr := &BinaryIndexExpr{
			Left:     &ConstantIndexExpr{Value: 7},
			Operator: IndexOpMax,
			Right:    &ConstantIndexExpr{Value: 3},
		}

		if maxExpr.String() != "max(7, 3)" {
			t.Errorf("expected 'max(7, 3)', got '%s'", maxExpr.String())
		}

		val, err = maxExpr.Evaluate(nil)
		if err != nil {
			t.Fatalf("max evaluation failed: %v", err)
		}

		if val != 7 {
			t.Errorf("expected 7, got %d", val)
		}
	})

	t.Run("ExpressionSimplification", func(t *testing.T) {
		// Test constant folding: 5 + 3 -> 8.
		addExpr := &BinaryIndexExpr{
			Left:     &ConstantIndexExpr{Value: 5},
			Operator: IndexOpAdd,
			Right:    &ConstantIndexExpr{Value: 3},
		}

		simplified := addExpr.Simplify()
		if constExpr, ok := simplified.(*ConstantIndexExpr); ok {
			if constExpr.Value != 8 {
				t.Errorf("expected simplified to 8, got %d", constExpr.Value)
			}
		} else {
			t.Error("5 + 3 should simplify to constant 8")
		}

		// Test addition identity: x + 0 -> x.
		varExpr := &VariableIndexExpr{Name: "x"}
		identityExpr := &BinaryIndexExpr{
			Left:     varExpr,
			Operator: IndexOpAdd,
			Right:    &ConstantIndexExpr{Value: 0},
		}

		simplified = identityExpr.Simplify()
		if varSimplified, ok := simplified.(*VariableIndexExpr); ok {
			if varSimplified.Name != "x" {
				t.Errorf("expected simplified to variable x, got %s", varSimplified.Name)
			}
		} else {
			t.Error("x + 0 should simplify to x")
		}

		// Test multiplication by zero: x * 0 -> 0.
		zeroExpr := &BinaryIndexExpr{
			Left:     varExpr,
			Operator: IndexOpMul,
			Right:    &ConstantIndexExpr{Value: 0},
		}

		simplified = zeroExpr.Simplify()
		if constExpr, ok := simplified.(*ConstantIndexExpr); ok {
			if constExpr.Value != 0 {
				t.Errorf("expected simplified to 0, got %d", constExpr.Value)
			}
		} else {
			t.Error("x * 0 should simplify to 0")
		}

		// Test multiplication identity: x * 1 -> x.
		identityMulExpr := &BinaryIndexExpr{
			Left:     varExpr,
			Operator: IndexOpMul,
			Right:    &ConstantIndexExpr{Value: 1},
		}

		simplified = identityMulExpr.Simplify()
		if varSimplified, ok := simplified.(*VariableIndexExpr); ok {
			if varSimplified.Name != "x" {
				t.Errorf("expected simplified to variable x, got %s", varSimplified.Name)
			}
		} else {
			t.Error("x * 1 should simplify to x")
		}

		// Test subtraction identity: x - 0 -> x.
		subIdentityExpr := &BinaryIndexExpr{
			Left:     varExpr,
			Operator: IndexOpSub,
			Right:    &ConstantIndexExpr{Value: 0},
		}

		simplified = subIdentityExpr.Simplify()
		if varSimplified, ok := simplified.(*VariableIndexExpr); ok {
			if varSimplified.Name != "x" {
				t.Errorf("expected simplified to variable x, got %s", varSimplified.Name)
			}
		} else {
			t.Error("x - 0 should simplify to x")
		}
	})

	t.Run("LengthIndexExpressions", func(t *testing.T) {
		// Test length expression.
		lengthExpr := &LengthIndexExpr{ArrayName: "arr"}

		if lengthExpr.String() != "len(arr)" {
			t.Errorf("expected 'len(arr)', got '%s'", lengthExpr.String())
		}

		if lengthExpr.IsStatic() {
			t.Error("length expression should not be static")
		}

		vars := lengthExpr.Variables()
		if len(vars) != 1 || vars[0] != "arr" {
			t.Errorf("expected variables [arr], got %v", vars)
		}

		// Test evaluation with length in environment.
		env := map[string]interface{}{
			"len(arr)": int64(10),
		}

		val, err := lengthExpr.Evaluate(env)
		if err != nil {
			t.Fatalf("length evaluation failed: %v", err)
		}

		if val != 10 {
			t.Errorf("expected 10, got %d", val)
		}

		// Test substitution.
		substitutions := map[string]interface{}{
			"len(arr)": int64(20),
		}

		substituted := lengthExpr.Substitute(substitutions)
		if constExpr, ok := substituted.(*ConstantIndexExpr); ok {
			if constExpr.Value != 20 {
				t.Errorf("expected substituted to 20, got %d", constExpr.Value)
			}
		} else {
			t.Error("length substitution should result in constant")
		}
	})

	t.Run("IndexBounds", func(t *testing.T) {
		// Test index bounds: [0..10)
		bounds := &IndexBound{
			Lower: &ConstantIndexExpr{Value: 0},
			Upper: &ConstantIndexExpr{Value: 10},
		}

		if bounds.String() != "[0..10)" {
			t.Errorf("expected '[0..10)', got '%s'", bounds.String())
		}

		// Test contains check for valid index.
		validIndex := &ConstantIndexExpr{Value: 5}

		contains, err := bounds.Contains(validIndex, nil)
		if err != nil {
			t.Fatalf("bounds checking failed: %v", err)
		}

		if !contains {
			t.Error("index 5 should be within bounds [0..10)")
		}

		// Test contains check for invalid index (too low).
		invalidLowIndex := &ConstantIndexExpr{Value: -1}

		contains, err = bounds.Contains(invalidLowIndex, nil)
		if err != nil {
			t.Fatalf("bounds checking failed: %v", err)
		}

		if contains {
			t.Error("index -1 should not be within bounds [0..10)")
		}

		// Test contains check for invalid index (too high).
		invalidHighIndex := &ConstantIndexExpr{Value: 10}

		contains, err = bounds.Contains(invalidHighIndex, nil)
		if err != nil {
			t.Fatalf("bounds checking failed: %v", err)
		}

		if contains {
			t.Error("index 10 should not be within bounds [0..10) (exclusive upper bound)")
		}

		// Test edge case: upper bound - 1.
		edgeIndex := &ConstantIndexExpr{Value: 9}

		contains, err = bounds.Contains(edgeIndex, nil)
		if err != nil {
			t.Fatalf("bounds checking failed: %v", err)
		}

		if !contains {
			t.Error("index 9 should be within bounds [0..10)")
		}
	})

	t.Run("IndexTypes", func(t *testing.T) {
		// Test fixed array type.
		fixedArray := NewFixedArray(TypeInt32, 10)

		if !fixedArray.IsFixedLength {
			t.Error("fixed array should have fixed length")
		}

		if fixedArray.ElementType != TypeInt32 {
			t.Error("fixed array should have Int32 element type")
		}

		expectedString := "Array[10, int32]"
		if fixedArray.String() != expectedString {
			t.Errorf("expected '%s', got '%s'", expectedString, fixedArray.String())
		}

		// Test bounds.
		bounds := fixedArray.GetBounds()
		if bounds.String() != "[0..10)" {
			t.Errorf("expected bounds '[0..10)', got '%s'", bounds.String())
		}

		// Test valid index check.
		validIndex := &ConstantIndexExpr{Value: 5}

		valid, err := fixedArray.IsValidIndex(validIndex, nil)
		if err != nil {
			t.Fatalf("index validation failed: %v", err)
		}

		if !valid {
			t.Error("index 5 should be valid for array of length 10")
		}

		// Test invalid index check.
		invalidIndex := &ConstantIndexExpr{Value: 15}

		valid, err = fixedArray.IsValidIndex(invalidIndex, nil)
		if err != nil {
			t.Fatalf("index validation failed: %v", err)
		}

		if valid {
			t.Error("index 15 should not be valid for array of length 10")
		}
	})

	t.Run("DynamicSlices", func(t *testing.T) {
		// Test dynamic slice type.
		dynamicSlice := NewDynamicSlice(TypeInt32, "data")

		if dynamicSlice.IsFixedLength {
			t.Error("dynamic slice should not have fixed length")
		}

		if dynamicSlice.ElementType != TypeInt32 {
			t.Error("dynamic slice should have Int32 element type")
		}

		expectedString := "Slice[int32]"
		if dynamicSlice.String() != expectedString {
			t.Errorf("expected '%s', got '%s'", expectedString, dynamicSlice.String())
		}

		// Test bounds with length variable.
		bounds := dynamicSlice.GetBounds()
		if bounds.String() != "[0..len(data))" {
			t.Errorf("expected bounds '[0..len(data))', got '%s'", bounds.String())
		}
	})

	t.Run("BoundedArrays", func(t *testing.T) {
		// Test bounded array with custom bounds.
		lengthExpr := &ConstantIndexExpr{Value: 20}
		lowerBound := &ConstantIndexExpr{Value: 5}
		upperBound := &ConstantIndexExpr{Value: 15}

		boundedArray := NewBoundedArray(TypeFloat64, lengthExpr, lowerBound, upperBound)

		if boundedArray.ElementType != TypeFloat64 {
			t.Error("bounded array should have Float64 element type")
		}

		bounds := boundedArray.GetBounds()
		if bounds.String() != "[5..15)" {
			t.Errorf("expected bounds '[5..15)', got '%s'", bounds.String())
		}

		// Test index within custom bounds.
		validIndex := &ConstantIndexExpr{Value: 10}

		valid, err := boundedArray.IsValidIndex(validIndex, nil)
		if err != nil {
			t.Fatalf("index validation failed: %v", err)
		}

		if !valid {
			t.Error("index 10 should be valid for bounded array [5..15)")
		}

		// Test index outside custom bounds (but within array length).
		invalidIndex := &ConstantIndexExpr{Value: 3}

		valid, err = boundedArray.IsValidIndex(invalidIndex, nil)
		if err != nil {
			t.Fatalf("index validation failed: %v", err)
		}

		if valid {
			t.Error("index 3 should not be valid for bounded array [5..15)")
		}
	})

	t.Run("OperatorStringRepresentation", func(t *testing.T) {
		// Test binary operator string representations.
		operators := []struct {
			expected string
			op       BinaryIndexOperator
		}{
			{IndexOpAdd, "+"},
			{IndexOpSub, "-"},
			{IndexOpMul, "*"},
			{IndexOpDiv, "/"},
			{IndexOpMod, "%"},
			{IndexOpMin, "min"},
			{IndexOpMax, "max"},
		}

		for _, test := range operators {
			if test.op.String() != test.expected {
				t.Errorf("operator %d: expected '%s', got '%s'",
					test.op, test.expected, test.op.String())
			}
		}
	})
}

func TestIndexChecker(t *testing.T) {
	t.Run("IndexCheckerCreation", func(t *testing.T) {
		checker := NewIndexChecker()

		if len(checker.constraints) != 0 {
			t.Error("new index checker should have no constraints")
		}

		if len(checker.environment) != 0 {
			t.Error("new index checker should have empty environment")
		}
	})

	t.Run("ArrayVariableManagement", func(t *testing.T) {
		checker := NewIndexChecker()

		// Add a fixed array variable.
		fixedArray := NewFixedArray(TypeInt32, 10)
		checker.AddArrayVariable("myArray", fixedArray)

		if arrayType, exists := checker.environment["myArray"]; !exists {
			t.Error("array variable should be added to environment")
		} else if arrayType != fixedArray {
			t.Error("stored array type should match added type")
		}

		// Check length environment.
		lengthKey := "len(myArray)"
		if lengthExpr, exists := checker.lengthEnv[lengthKey]; !exists {
			t.Error("length expression should be added to length environment")
		} else {
			if constExpr, ok := lengthExpr.(*ConstantIndexExpr); !ok || constExpr.Value != 10 {
				t.Error("length expression should be constant 10")
			}
		}
	})

	t.Run("BoundsConstraintCollection", func(t *testing.T) {
		checker := NewIndexChecker()

		// Add an array variable.
		fixedArray := NewFixedArray(TypeInt32, 5)
		checker.AddArrayVariable("testArray", fixedArray)

		// Create array access expression.
		arrayVar := &VariableExpr{Name: "testArray"}
		index := &ConstantIndexExpr{Value: 3}
		location := SourceLocation{Line: 10, Column: 5}

		err := checker.CheckArrayAccess(arrayVar, index, location)
		if err != nil {
			t.Fatalf("array access checking failed: %v", err)
		}

		// Should have collected one constraint.
		if len(checker.constraints) != 1 {
			t.Errorf("expected 1 constraint, got %d", len(checker.constraints))
		}

		constraint := checker.constraints[0]
		if constraint.Location.Line != 10 || constraint.Location.Column != 5 {
			t.Error("constraint should preserve location information")
		}
	})

	t.Run("ConstraintSolvingValid", func(t *testing.T) {
		checker := NewIndexChecker()

		// Add an array variable.
		fixedArray := NewFixedArray(TypeInt32, 10)
		checker.AddArrayVariable("validArray", fixedArray)

		// Create a valid array access.
		arrayVar := &VariableExpr{Name: "validArray"}
		validIndex := &ConstantIndexExpr{Value: 5}
		location := SourceLocation{Line: 1, Column: 1}

		err := checker.CheckArrayAccess(arrayVar, validIndex, location)
		if err != nil {
			t.Fatalf("array access checking failed: %v", err)
		}

		// Solve constraints.
		errors, err := checker.SolveIndexConstraints()
		if err != nil {
			t.Fatalf("constraint solving failed: %v", err)
		}

		// Should have no errors for valid index.
		if len(errors) != 0 {
			t.Errorf("expected no errors for valid index, got %d", len(errors))
		}
	})

	t.Run("ConstraintSolvingInvalid", func(t *testing.T) {
		checker := NewIndexChecker()

		// Add an array variable.
		fixedArray := NewFixedArray(TypeInt32, 5)
		checker.AddArrayVariable("invalidArray", fixedArray)

		// Create an invalid array access (index out of bounds).
		arrayVar := &VariableExpr{Name: "invalidArray"}
		invalidIndex := &ConstantIndexExpr{Value: 10}
		location := SourceLocation{Line: 2, Column: 3}

		err := checker.CheckArrayAccess(arrayVar, invalidIndex, location)
		if err != nil {
			t.Fatalf("array access checking failed: %v", err)
		}

		// Solve constraints.
		errors, err := checker.SolveIndexConstraints()
		if err != nil {
			t.Fatalf("constraint solving failed: %v", err)
		}

		// Should have one error for invalid index.
		if len(errors) != 1 {
			t.Errorf("expected 1 error for invalid index, got %d", len(errors))
		}

		if len(errors) > 0 {
			indexErr := errors[0]
			if indexErr.Location.Line != 2 || indexErr.Location.Column != 3 {
				t.Error("error should preserve location information")
			}
		}
	})

	t.Run("UnknownArrayHandling", func(t *testing.T) {
		checker := NewIndexChecker()

		// Try to check access to unknown array.
		unknownArray := &VariableExpr{Name: "unknownArray"}
		index := &ConstantIndexExpr{Value: 0}
		location := SourceLocation{Line: 1, Column: 1}

		err := checker.CheckArrayAccess(unknownArray, index, location)
		if err == nil {
			t.Error("expected error for unknown array variable")
		}
	})
}

func TestIndexExpressionParser(t *testing.T) {
	t.Run("BasicParsing", func(t *testing.T) {
		// Test parsing constant.
		parser := NewIndexExpressionParser("42")

		expr, err := parser.ParseIndexExpression()
		if err != nil {
			t.Fatalf("parsing '42' failed: %v", err)
		}

		if constExpr, ok := expr.(*ConstantIndexExpr); !ok || constExpr.Value != 42 {
			t.Error("'42' should parse as constant 42")
		}

		// Test parsing variable.
		parser = NewIndexExpressionParser("i")

		expr, err = parser.ParseIndexExpression()
		if err != nil {
			t.Fatalf("parsing 'i' failed: %v", err)
		}

		if varExpr, ok := expr.(*VariableIndexExpr); !ok || varExpr.Name != "i" {
			t.Error("'i' should parse as variable i")
		}

		// Test parsing length function.
		parser = NewIndexExpressionParser("len(arr)")

		expr, err = parser.ParseIndexExpression()
		if err != nil {
			t.Fatalf("parsing 'len(arr)' failed: %v", err)
		}

		if lengthExpr, ok := expr.(*LengthIndexExpr); !ok || lengthExpr.ArrayName != "arr" {
			t.Error("'len(arr)' should parse as length expression for arr")
		}
	})

	t.Run("BinaryExpressionParsing", func(t *testing.T) {
		// Test parsing addition.
		parser := NewIndexExpressionParser("i + 1")

		expr, err := parser.ParseIndexExpression()
		if err != nil {
			t.Fatalf("parsing 'i + 1' failed: %v", err)
		}

		if binExpr, ok := expr.(*BinaryIndexExpr); ok {
			if binExpr.Operator != IndexOpAdd {
				t.Error("should parse '+' operator")
			}

			if varExpr, ok := binExpr.Left.(*VariableIndexExpr); !ok || varExpr.Name != "i" {
				t.Error("left operand should be variable 'i'")
			}

			if constExpr, ok := binExpr.Right.(*ConstantIndexExpr); !ok || constExpr.Value != 1 {
				t.Error("right operand should be constant 1")
			}
		} else {
			t.Error("'i + 1' should parse as binary expression")
		}

		// Test parsing multiplication with precedence.
		parser = NewIndexExpressionParser("2 * 3 + 4")

		expr, err = parser.ParseIndexExpression()
		if err != nil {
			t.Fatalf("parsing '2 * 3 + 4' failed: %v", err)
		}

		// Should parse as (2 * 3) + 4 due to precedence.
		if binExpr, ok := expr.(*BinaryIndexExpr); ok {
			if binExpr.Operator != IndexOpAdd {
				t.Error("top level should be addition")
			}

			if leftBin, ok := binExpr.Left.(*BinaryIndexExpr); ok {
				if leftBin.Operator != IndexOpMul {
					t.Error("left operand should be multiplication")
				}
			} else {
				t.Error("left operand should be binary expression")
			}
		} else {
			t.Error("'2 * 3 + 4' should parse as binary expression")
		}
	})

	t.Run("ParenthesesParsing", func(t *testing.T) {
		// Test parsing with parentheses.
		parser := NewIndexExpressionParser("(i + 1)")

		expr, err := parser.ParseIndexExpression()
		if err != nil {
			t.Fatalf("parsing '(i + 1)' failed: %v", err)
		}

		if binExpr, ok := expr.(*BinaryIndexExpr); ok {
			if binExpr.Operator != IndexOpAdd {
				t.Error("should parse as addition")
			}
		} else {
			t.Error("'(i + 1)' should parse as binary expression")
		}

		// Test precedence with parentheses: 2 * (3 + 4).
		parser = NewIndexExpressionParser("2 * (3 + 4)")

		expr, err = parser.ParseIndexExpression()
		if err != nil {
			t.Fatalf("parsing '2 * (3 + 4)' failed: %v", err)
		}

		// Should parse as 2 * (3 + 4).
		if binExpr, ok := expr.(*BinaryIndexExpr); ok {
			if binExpr.Operator != IndexOpMul {
				t.Error("top level should be multiplication")
			}

			if rightBin, ok := binExpr.Right.(*BinaryIndexExpr); ok {
				if rightBin.Operator != IndexOpAdd {
					t.Error("right operand should be addition")
				}
			} else {
				t.Error("right operand should be binary expression")
			}
		} else {
			t.Error("'2 * (3 + 4)' should parse as binary expression")
		}
	})

	t.Run("ComplexExpressionParsing", func(t *testing.T) {
		// Test complex expression: len(arr) - 1.
		parser := NewIndexExpressionParser("len(data) - 1")

		expr, err := parser.ParseIndexExpression()
		if err != nil {
			t.Fatalf("parsing 'len(data) - 1' failed: %v", err)
		}

		if binExpr, ok := expr.(*BinaryIndexExpr); ok {
			if binExpr.Operator != IndexOpSub {
				t.Error("should parse as subtraction")
			}

			if lengthExpr, ok := binExpr.Left.(*LengthIndexExpr); !ok || lengthExpr.ArrayName != "data" {
				t.Error("left operand should be length of 'data'")
			}

			if constExpr, ok := binExpr.Right.(*ConstantIndexExpr); !ok || constExpr.Value != 1 {
				t.Error("right operand should be constant 1")
			}
		} else {
			t.Error("'len(data) - 1' should parse as binary expression")
		}
	})
}

func TestArrayAccessExpressions(t *testing.T) {
	t.Run("ArrayAccessString", func(t *testing.T) {
		// Test array access string representation.
		arrayVar := &VariableExpr{Name: "myArray"}
		index := &ConstantIndexExpr{Value: 5}

		arrayAccess := &ArrayAccessExpr{
			Array: arrayVar,
			Index: index,
		}

		expected := "myArray[5]"
		if arrayAccess.String() != expected {
			t.Errorf("expected '%s', got '%s'", expected, arrayAccess.String())
		}
	})

	t.Run("SliceExpressionString", func(t *testing.T) {
		arrayVar := &VariableExpr{Name: "data"}

		// Test full slice: data[1:5].
		startIndex := &ConstantIndexExpr{Value: 1}
		endIndex := &ConstantIndexExpr{Value: 5}

		fullSlice := &SliceExpr{
			Array: arrayVar,
			Start: startIndex,
			End:   endIndex,
		}

		expected := "data[1:5]"
		if fullSlice.String() != expected {
			t.Errorf("expected '%s', got '%s'", expected, fullSlice.String())
		}

		// Test start-only slice: data[2:].
		startOnlySlice := &SliceExpr{
			Array: arrayVar,
			Start: &ConstantIndexExpr{Value: 2},
			End:   nil,
		}

		expected = "data[2:]"
		if startOnlySlice.String() != expected {
			t.Errorf("expected '%s', got '%s'", expected, startOnlySlice.String())
		}

		// Test end-only slice: data[:3].
		endOnlySlice := &SliceExpr{
			Array: arrayVar,
			Start: nil,
			End:   &ConstantIndexExpr{Value: 3},
		}

		expected = "data[:3]"
		if endOnlySlice.String() != expected {
			t.Errorf("expected '%s', got '%s'", expected, endOnlySlice.String())
		}

		// Test full slice: data[:].
		fullDataSlice := &SliceExpr{
			Array: arrayVar,
			Start: nil,
			End:   nil,
		}

		expected = "data[:]"
		if fullDataSlice.String() != expected {
			t.Errorf("expected '%s', got '%s'", expected, fullDataSlice.String())
		}
	})
}

func TestDependentArrayTypes(t *testing.T) {
	t.Run("DependentArrayTypeCreation", func(t *testing.T) {
		// Create a dependent array type.
		lengthBound := NewPositiveType(TypeInt32)

		depArrayType := &DependentArrayType{
			ElementType: TypeFloat64,
			LengthVar:   "n",
			LengthBound: lengthBound,
		}

		expectedString := "Array[n:{v:int32 | (v > 0)}, float64]"
		if depArrayType.String() != expectedString {
			t.Errorf("expected '%s', got '%s'", expectedString, depArrayType.String())
		}

		// Test creating concrete index type.
		indexType := depArrayType.CreateIndexType(10)

		if !indexType.IsFixedLength {
			t.Error("created index type should have fixed length")
		}

		if indexType.ElementType != TypeFloat64 {
			t.Error("created index type should preserve element type")
		}

		// Test bounds.
		bounds := indexType.GetBounds()
		if bounds.String() != "[0..10)" {
			t.Errorf("expected bounds '[0..10)', got '%s'", bounds.String())
		}
	})
}

func TestComplexIndexExpressions(t *testing.T) {
	t.Run("VariableSubstitution", func(t *testing.T) {
		// Test complex expression with substitution: (i + j) * 2.
		iVar := &VariableIndexExpr{Name: "i"}
		jVar := &VariableIndexExpr{Name: "j"}

		addExpr := &BinaryIndexExpr{
			Left:     iVar,
			Operator: IndexOpAdd,
			Right:    jVar,
		}

		mulExpr := &BinaryIndexExpr{
			Left:     addExpr,
			Operator: IndexOpMul,
			Right:    &ConstantIndexExpr{Value: 2},
		}

		// Test variable collection.
		vars := mulExpr.Variables()
		if len(vars) < 2 {
			t.Errorf("complex expression should have at least 2 variables, got %d", len(vars))
		}

		// Check that both i and j are present.
		varSet := make(map[string]bool)
		for _, v := range vars {
			varSet[v] = true
		}

		if !varSet["i"] || !varSet["j"] {
			t.Error("complex expression should include both i and j variables")
		}

		// Test substitution: i = 3, j = 4.
		substitutions := map[string]interface{}{
			"i": int64(3),
			"j": int64(4),
		}

		substituted := mulExpr.Substitute(substitutions)

		// Should evaluate to (3 + 4) * 2 = 14.
		val, err := substituted.Evaluate(nil)
		if err != nil {
			t.Fatalf("substituted expression evaluation failed: %v", err)
		}

		if val != 14 {
			t.Errorf("expected 14, got %d", val)
		}
	})

	t.Run("NestedLengthExpressions", func(t *testing.T) {
		// Test expression: len(arr1) + len(arr2).
		len1 := &LengthIndexExpr{ArrayName: "arr1"}
		len2 := &LengthIndexExpr{ArrayName: "arr2"}

		sumLengths := &BinaryIndexExpr{
			Left:     len1,
			Operator: IndexOpAdd,
			Right:    len2,
		}

		if sumLengths.String() != "(len(arr1) + len(arr2))" {
			t.Errorf("unexpected string representation: %s", sumLengths.String())
		}

		// Test evaluation with length environment.
		env := map[string]interface{}{
			"len(arr1)": int64(5),
			"len(arr2)": int64(7),
		}

		val, err := sumLengths.Evaluate(env)
		if err != nil {
			t.Fatalf("nested length expression evaluation failed: %v", err)
		}

		if val != 12 {
			t.Errorf("expected 12, got %d", val)
		}
	})
}
