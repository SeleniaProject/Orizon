package integration

import (
	"testing"

	orizonTesting "github.com/orizon-lang/orizon/internal/testing"
)

// TestCompilerPipeline tests the entire compiler pipeline
func TestCompilerPipeline(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "simple_program",
			SourceCode: `
fn main() {
    print("Hello, Orizon!");
}`,
			ExpectedOut: "Hello, Orizon!",
			ShouldFail:  false,
		},
		{
			Name: "function_calls",
			SourceCode: `
fn factorial(n: i32) -> i32 {
    if n <= 1 {
        return 1;
    } else {
        return n * factorial(n - 1);
    }
}

fn main() {
    let result = factorial(5);
    print(result);
}`,
			ExpectedOut: "120",
			ShouldFail:  false,
		},
		{
			Name: "complex_data_structures",
			SourceCode: `
struct Person {
    name: String,
    age: i32,
}

fn create_person(name: String, age: i32) -> Person {
    return Person { name: name, age: age };
}

fn main() {
    let person = create_person("Alice", 30);
    print(person.name);
    print(person.age);
}`,
			ExpectedOut: "Alice30",
			ShouldFail:  false,
		},
		{
			Name: "generic_programming",
			SourceCode: `
fn max<T: Ord>(a: T, b: T) -> T {
    if a > b {
        return a;
    } else {
        return b;
    }
}

fn main() {
    let int_max = max(10, 20);
    let float_max = max(3.14, 2.71);
    print(int_max);
    print(float_max);
}`,
			ExpectedOut: "203.14",
			ShouldFail:  false,
		},
	}

	framework.RunTestSuite(tests, t)
}

// TestCodeGeneration tests code generation quality
func TestCodeGeneration(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "arithmetic_optimization",
			SourceCode: `
fn main() {
    let a = 5;
    let b = 10;
    let c = a + b * 2 - 3;  // Should optimize to 5 + 20 - 3 = 22
    print(c);
}`,
			ExpectedOut: "22",
			ShouldFail:  false,
		},
		{
			Name: "loop_optimization",
			SourceCode: `
fn main() {
    let mut sum = 0;
    for i in 1..=10 {
        sum += i;
    }
    print(sum);  // 1+2+...+10 = 55
}`,
			ExpectedOut: "55",
			ShouldFail:  false,
		},
		{
			Name: "function_inlining",
			SourceCode: `
#[inline]
fn square(x: i32) -> i32 {
    return x * x;
}

fn main() {
    let result = square(5);
    print(result);
}`,
			ExpectedOut: "25",
			ShouldFail:  false,
		},
		{
			Name: "register_allocation",
			SourceCode: `
fn complex_calculation(a: i32, b: i32, c: i32, d: i32) -> i32 {
    let x = a + b;
    let y = c + d;
    let z = x * y;
    let w = x + y;
    return z - w;
}

fn main() {
    let result = complex_calculation(1, 2, 3, 4);
    print(result);
}`,
			ExpectedOut: "11", // (1+2)*(3+4) - (1+2+3+4) = 3*7 - 10 = 21 - 10 = 11
			ShouldFail:  false,
		},
	}

	framework.RunTestSuite(tests, t)
}

// TestStandardLibrary tests standard library integration
func TestStandardLibrary(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "collections",
			SourceCode: `
use std::collections::Vec;
use std::collections::HashMap;

fn main() {
    let mut v = Vec::new();
    v.push(1);
    v.push(2);
    v.push(3);
    
    let mut map = HashMap::new();
    map.insert("key1", "value1");
    map.insert("key2", "value2");
    
    print(v.len());
    print(map.len());
}`,
			ExpectedOut: "32",
			ShouldFail:  false,
		},
		{
			Name: "string_manipulation",
			SourceCode: `
use std::string::String;

fn main() {
    let mut s = String::from("Hello");
    s.push_str(", ");
    s.push_str("World!");
    
    let words: Vec<&str> = s.split(' ').collect();
    print(words.len());
    print(s);
}`,
			ExpectedOut: "2Hello, World!",
			ShouldFail:  false,
		},
		{
			Name: "file_io",
			SourceCode: `
use std::fs::File;
use std::io::Write;

fn main() {
    let mut file = File::create("test.txt").unwrap();
    file.write_all(b"Hello, File!").unwrap();
    
    let content = std::fs::read_to_string("test.txt").unwrap();
    print(content);
}`,
			ExpectedOut: "Hello, File!",
			ShouldFail:  false,
		},
	}

	framework.RunTestSuite(tests, t)
}

// TestConcurrency tests concurrency features
func TestConcurrency(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "basic_threading",
			SourceCode: `
use std::thread;

fn worker(id: i32) {
    print(id);
}

fn main() {
    let mut handles = Vec::new();
    
    for i in 0..3 {
        let handle = thread::spawn(move || {
            worker(i);
        });
        handles.push(handle);
    }
    
    for handle in handles {
        handle.join().unwrap();
    }
}`,
			ExpectedOut: "012", // Order may vary due to threading
			ShouldFail:  false,
		},
		{
			Name: "channel_communication",
			SourceCode: `
use std::sync::mpsc;
use std::thread;

fn main() {
    let (tx, rx) = mpsc::channel();
    
    thread::spawn(move || {
        tx.send("Hello from thread").unwrap();
    });
    
    let received = rx.recv().unwrap();
    print(received);
}`,
			ExpectedOut: "Hello from thread",
			ShouldFail:  false,
		},
		{
			Name: "mutex_synchronization",
			SourceCode: `
use std::sync::{Arc, Mutex};
use std::thread;

fn main() {
    let counter = Arc::new(Mutex::new(0));
    let mut handles = Vec::new();
    
    for _ in 0..3 {
        let counter = Arc::clone(&counter);
        let handle = thread::spawn(move || {
            let mut num = counter.lock().unwrap();
            *num += 1;
        });
        handles.push(handle);
    }
    
    for handle in handles {
        handle.join().unwrap();
    }
    
    let final_count = *counter.lock().unwrap();
    print(final_count);
}`,
			ExpectedOut: "3",
			ShouldFail:  false,
		},
	}

	framework.RunTestSuite(tests, t)
}

// TestInteroperability tests language interoperability
func TestInteroperability(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "c_interop",
			SourceCode: `
extern "C" {
    fn printf(format: *const i8, ...) -> i32;
    fn malloc(size: usize) -> *mut u8;
    fn free(ptr: *mut u8);
}

fn main() {
    let ptr = malloc(1024);
    if ptr != null {
        printf("Allocated %d bytes\n", 1024);
        free(ptr);
    }
}`,
			ExpectedOut: "Allocated 1024 bytes",
			ShouldFail:  false,
		},
		{
			Name: "system_calls",
			SourceCode: `
use std::process::Command;

fn main() {
    let output = Command::new("echo")
        .arg("Hello from system call")
        .output()
        .unwrap();
    
    let stdout = String::from_utf8(output.stdout).unwrap();
    print(stdout.trim());
}`,
			ExpectedOut: "Hello from system call",
			ShouldFail:  false,
		},
	}

	framework.RunTestSuite(tests, t)
}

// TestPerformance tests performance characteristics
func TestPerformance(t *testing.T) {
	framework, err := orizonTesting.NewTestFramework(nil)
	if err != nil {
		t.Fatalf("Failed to create test framework: %v", err)
	}
	defer framework.Cleanup()

	tests := []*orizonTesting.CompilerTest{
		{
			Name: "large_compilation",
			SourceCode: `
// Generate a large function with many operations
fn large_function() -> i32 {
    let mut result = 0;
    ` + generateLargeFunction(1000) + `
    return result;
}

fn main() {
    let result = large_function();
    print(result);
}`,
			ShouldFail: false,
		},
		{
			Name: "deep_recursion",
			SourceCode: `
fn deep_recursion(n: i32) -> i32 {
    if n <= 0 {
        return 0;
    }
    return 1 + deep_recursion(n - 1);
}

fn main() {
    let result = deep_recursion(100);
    print(result);
}`,
			ExpectedOut: "100",
			ShouldFail:  false,
		},
	}

	framework.RunTestSuite(tests, t)
}

// generateLargeFunction generates a large function body for testing
func generateLargeFunction(size int) string {
	var body string
	for i := 0; i < size; i++ {
		digit := i % 10
		body += "result += " + string(rune('0'+digit)) + ";\n"
	}
	return body
}
