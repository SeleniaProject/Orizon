package unit

import (
	"testing"

	orizonTesting "github.com/orizon-lang/orizon/internal/testing"
)

// TestLexerBasics tests basic lexer functionality
func TestLexerBasics(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "basic_tokens",
			SourceCode: `
fn main() {
    let x = 42;
    print(x);
}`,
			ExpectedOut: "42",
			ShouldFail:  false,
		},
		{
			Name: "string_literals",
			SourceCode: `
fn main() {
    let msg = "Hello, World!";
    print(msg);
}`,
			ExpectedOut: "Hello, World!",
			ShouldFail:  false,
		},
		{
			Name: "number_literals",
			SourceCode: `
fn main() {
    let a = 123;
    let b = 456;
    let c = a + b;
    print(c);
}`,
			ExpectedOut: "579",
			ShouldFail:  false,
		},
		{
			Name: "invalid_syntax",
			SourceCode: `
fn main() {
    let x = ;  // Invalid syntax
}`,
			ShouldFail: true,
		},
	}

	framework.RunTestSuite(tests, t)
}

// TestParserBasics tests basic parser functionality
func TestParserBasics(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "function_declaration",
			SourceCode: `
fn add(a: i32, b: i32) -> i32 {
    return a + b;
}

fn main() {
    let result = add(5, 3);
    print(result);
}`,
			ExpectedOut: "8",
			ShouldFail:  false,
		},
		{
			Name: "if_statement",
			SourceCode: `
fn main() {
    let x = 10;
    if x > 5 {
        print("greater");
    } else {
        print("lesser");
    }
}`,
			ExpectedOut: "greater",
			ShouldFail:  false,
		},
		{
			Name: "while_loop",
			SourceCode: `
fn main() {
    let mut i = 0;
    while i < 3 {
        print(i);
        i = i + 1;
    }
}`,
			ExpectedOut: "012",
			ShouldFail:  false,
		},
		{
			Name: "struct_declaration",
			SourceCode: `
struct Point {
    x: i32,
    y: i32,
}

fn main() {
    let p = Point { x: 10, y: 20 };
    print(p.x);
    print(p.y);
}`,
			ExpectedOut: "1020",
			ShouldFail:  false,
		},
	}

	framework.RunTestSuite(tests, t)
}

// TestTypeChecker tests type checking functionality
func TestTypeChecker(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "type_inference",
			SourceCode: `
fn main() {
    let x = 42;        // inferred as i32
    let y = 3.14;      // inferred as f64
    let z = "hello";   // inferred as string
    print(x);
}`,
			ExpectedOut: "42",
			ShouldFail:  false,
		},
		{
			Name: "type_mismatch",
			SourceCode: `
fn main() {
    let x: i32 = "string";  // Type mismatch
}`,
			ShouldFail: true,
		},
		{
			Name: "function_type_checking",
			SourceCode: `
fn add(a: i32, b: i32) -> i32 {
    return a + b;
}

fn main() {
    let result = add(1, 2);
    print(result);
}`,
			ExpectedOut: "3",
			ShouldFail:  false,
		},
		{
			Name: "wrong_argument_type",
			SourceCode: `
fn add(a: i32, b: i32) -> i32 {
    return a + b;
}

fn main() {
    let result = add("1", 2);  // Wrong argument type
}`,
			ShouldFail: true,
		},
		{
			Name: "generic_function",
			SourceCode: `
fn identity<T>(x: T) -> T {
    return x;
}

fn main() {
    let a = identity(42);
    let b = identity("hello");
    print(a);
}`,
			ExpectedOut: "42",
			ShouldFail:  false,
		},
	}

	framework.RunTestSuite(tests, t)
}

// TestCoreTypes tests core type functionality
func TestCoreTypes(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "option_type",
			SourceCode: `
fn divide(a: i32, b: i32) -> Option<i32> {
    if b == 0 {
        return None;
    } else {
        return Some(a / b);
    }
}

fn main() {
    let result = divide(10, 2);
    match result {
        Some(value) => print(value),
        None => print("division by zero"),
    }
}`,
			ExpectedOut: "5",
			ShouldFail:  false,
		},
		{
			Name: "result_type",
			SourceCode: `
fn safe_divide(a: i32, b: i32) -> Result<i32, String> {
    if b == 0 {
        return Err("division by zero");
    } else {
        return Ok(a / b);
    }
}

fn main() {
    let result = safe_divide(10, 2);
    match result {
        Ok(value) => print(value),
        Err(msg) => print(msg),
    }
}`,
			ExpectedOut: "5",
			ShouldFail:  false,
		},
		{
			Name: "vec_operations",
			SourceCode: `
fn main() {
    let mut v = Vec::new();
    v.push(1);
    v.push(2);
    v.push(3);
    
    for i in 0..v.len() {
        print(v[i]);
    }
}`,
			ExpectedOut: "123",
			ShouldFail:  false,
		},
		{
			Name: "string_operations",
			SourceCode: `
fn main() {
    let s1 = "Hello";
    let s2 = " World";
    let combined = s1 + s2;
    print(combined);
}`,
			ExpectedOut: "Hello World",
			ShouldFail:  false,
		},
	}

	framework.RunTestSuite(tests, t)
}

// TestMemoryManagement tests memory management functionality
func TestMemoryManagement(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "basic_allocation",
			SourceCode: `
fn main() {
    let ptr = alloc(1024);
    if ptr != null {
        print("allocation successful");
        free(ptr);
    }
}`,
			ExpectedOut: "allocation successful",
			ShouldFail:  false,
		},
		{
			Name: "arena_allocator",
			SourceCode: `
fn main() {
    let arena = Arena::new(4096);
    let ptr1 = arena.alloc(100);
    let ptr2 = arena.alloc(200);
    
    if ptr1 != null && ptr2 != null {
        print("arena allocation successful");
    }
    
    // Arena will be automatically cleaned up
}`,
			ExpectedOut: "arena allocation successful",
			ShouldFail:  false,
		},
		{
			Name: "memory_leak_detection",
			SourceCode: `
fn main() {
    let ptr = alloc(1024);
    // Intentionally not freeing ptr to test leak detection
    print("done");
}`,
			ExpectedOut: "done",
			ShouldFail:  false, // Should compile but may generate warnings
		},
	}

	framework.RunTestSuite(tests, t)
}

// TestIntrinsics tests compiler intrinsics functionality
func TestIntrinsics(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "sizeof_intrinsic",
			SourceCode: `
fn main() {
    let size_i32 = sizeof(i32);
    let size_i64 = sizeof(i64);
    print(size_i32);
    print(size_i64);
}`,
			ExpectedOut: "48",
			ShouldFail:  false,
		},
		{
			Name: "atomic_operations",
			SourceCode: `
fn main() {
    let mut x = 0;
    let old_value = atomic_add(&x, 5);
    let current_value = atomic_load(&x);
    print(current_value);
}`,
			ExpectedOut: "5",
			ShouldFail:  false,
		},
		{
			Name: "bit_operations",
			SourceCode: `
fn main() {
    let value = 0b11110000;
    let zeros = count_leading_zeros(value);
    let ones = popcount(value);
    print(zeros);
    print(ones);
}`,
			ExpectedOut: "04",
			ShouldFail:  false,
		},
		{
			Name: "overflow_detection",
			SourceCode: `
fn main() {
    let (result, overflow) = add_overflow(i32::MAX, 1);
    if overflow {
        print("overflow detected");
    } else {
        print(result);
    }
}`,
			ExpectedOut: "overflow detected",
			ShouldFail:  false,
		},
	}

	framework.RunTestSuite(tests, t)
}

// TestErrorHandling tests error handling functionality
func TestErrorHandling(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "panic_handling",
			SourceCode: `
fn divide(a: i32, b: i32) -> i32 {
    if b == 0 {
        panic("division by zero");
    }
    return a / b;
}

fn main() {
    let result = divide(10, 0);
    print(result);
}`,
			ShouldFail: false, // Should compile but panic at runtime
		},
		{
			Name: "error_propagation",
			SourceCode: `
fn may_fail() -> Result<i32, String> {
    return Err("something went wrong");
}

fn call_may_fail() -> Result<i32, String> {
    let value = may_fail()?;  // Error propagation
    return Ok(value * 2);
}

fn main() {
    match call_may_fail() {
        Ok(value) => print(value),
        Err(msg) => print(msg),
    }
}`,
			ExpectedOut: "something went wrong",
			ShouldFail:  false,
		},
	}

	framework.RunTestSuite(tests, t)
}
