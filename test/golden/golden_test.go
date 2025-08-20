package golden_test

import (
	"path/filepath"
	"testing"

	orizonTesting "github.com/orizon-lang/orizon/internal/testing"
)

// TestGoldenFiles tests output against golden files.
func TestGoldenFiles(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	goldenDir := "../../test/golden"

	tests := []*orizonTesting.GoldenFileTest{
		{
			Name: "simple_hello",
			SourceCode: `
fn main() {
    print("Hello, World!");
}`,
			GoldenFile: filepath.Join(goldenDir, "simple_hello.golden"),
		},
		{
			Name: "arithmetic_operations",
			SourceCode: `
fn main() {
    let a = 10;
    let b = 5;
    print(a + b);
    print(a - b);
    print(a * b);
    print(a / b);
}`,
			GoldenFile: filepath.Join(goldenDir, "arithmetic_operations.golden"),
		},
		{
			Name: "function_calls",
			SourceCode: `
fn square(x: i32) -> i32 {
    return x * x;
}

fn main() {
    let result = square(5);
    print(result);
}`,
			GoldenFile: filepath.Join(goldenDir, "function_calls.golden"),
		},
		{
			Name: "control_flow",
			SourceCode: `
fn main() {
    for i in 0..5 {
        if i % 2 == 0 {
            print("even");
        } else {
            print("odd");
        }
    }
}`,
			GoldenFile: filepath.Join(goldenDir, "control_flow.golden"),
		},
		{
			Name: "struct_operations",
			SourceCode: `
struct Point {
    x: i32,
    y: i32,
}

fn main() {
    let p = Point { x: 10, y: 20 };
    print(p.x);
    print(p.y);
    
    let mut p2 = p;
    p2.x = 30;
    print(p2.x);
}`,
			GoldenFile: filepath.Join(goldenDir, "struct_operations.golden"),
		},
		{
			Name: "error_handling",
			SourceCode: `
fn divide(a: i32, b: i32) -> Result<i32, String> {
    if b == 0 {
        return Err("Division by zero");
    }
    return Ok(a / b);
}

fn main() {
    match divide(10, 2) {
        Ok(result) => print(result),
        Err(msg) => print(msg),
    }
    
    match divide(10, 0) {
        Ok(result) => print(result),
        Err(msg) => print(msg),
    }
}`,
			GoldenFile: filepath.Join(goldenDir, "error_handling.golden"),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			framework.RunGoldenFileTest(test, t)
		})
	}
}

// TestUpdateGoldenFiles can be used to update golden files.
func TestUpdateGoldenFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping golden file update in short mode")
	}

	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	goldenDir := "../../test/golden"

	tests := []*orizonTesting.GoldenFileTest{
		{
			Name: "simple_hello",
			SourceCode: `
fn main() {
    print("Hello, World!");
}`,
			GoldenFile:   filepath.Join(goldenDir, "simple_hello.golden"),
			UpdateGolden: true,
		},
		// Add more tests here when updating golden files.
	}

	for _, test := range tests {
		t.Run(test.Name+"_update", func(t *testing.T) {
			framework.RunGoldenFileTest(test, t)
		})
	}
}
