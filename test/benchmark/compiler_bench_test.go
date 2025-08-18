package benchmark

import (
	"testing"

	orizonTesting "github.com/orizon-lang/orizon/internal/testing"
)

// BenchmarkSimpleCompilation benchmarks compilation of simple programs
func BenchmarkSimpleCompilation(b *testing.B) {
	framework, err := orizonTesting.NewBenchmarkFramework(nil)
	if err != nil {
		b.Fatalf("Failed to create benchmark framework: %v", err)
	}
	defer framework.Cleanup()

	benchmark := &orizonTesting.BenchmarkTest{
		Name: "simple_hello",
		SourceCode: `
fn main() {
    print("Hello, World!");
}`,
	}

	framework.RunBenchmark(benchmark, b)
}

// BenchmarkComplexCompilation benchmarks compilation of complex programs
func BenchmarkComplexCompilation(b *testing.B) {
	framework, err := orizonTesting.NewBenchmarkFramework(nil)
	if err != nil {
		b.Fatalf("Failed to create benchmark framework: %v", err)
	}
	defer framework.Cleanup()

	benchmark := &orizonTesting.BenchmarkTest{
		Name: "complex_program",
		SourceCode: `
struct Vector3 {
    x: f32,
    y: f32,
    z: f32,
}

impl Vector3 {
    fn new(x: f32, y: f32, z: f32) -> Self {
        Vector3 { x, y, z }
    }
    
    fn dot(&self, other: &Vector3) -> f32 {
        self.x * other.x + self.y * other.y + self.z * other.z
    }
    
    fn cross(&self, other: &Vector3) -> Vector3 {
        Vector3::new(
            self.y * other.z - self.z * other.y,
            self.z * other.x - self.x * other.z,
            self.x * other.y - self.y * other.x
        )
    }
    
    fn magnitude(&self) -> f32 {
        (self.x * self.x + self.y * self.y + self.z * self.z).sqrt()
    }
    
    fn normalize(&self) -> Vector3 {
        let mag = self.magnitude();
        Vector3::new(self.x / mag, self.y / mag, self.z / mag)
    }
}

fn main() {
    let v1 = Vector3::new(1.0, 2.0, 3.0);
    let v2 = Vector3::new(4.0, 5.0, 6.0);
    
    let dot_product = v1.dot(&v2);
    let cross_product = v1.cross(&v2);
    let normalized = v1.normalize();
    
    print(dot_product);
    print(cross_product.x);
    print(normalized.magnitude());
}`,
	}

	framework.RunBenchmark(benchmark, b)
}

// BenchmarkLargeFileCompilation benchmarks compilation of large files
func BenchmarkLargeFileCompilation(b *testing.B) {
	framework, err := orizonTesting.NewBenchmarkFramework(nil)
	if err != nil {
		b.Fatalf("Failed to create benchmark framework: %v", err)
	}
	defer framework.Cleanup()

	// Generate a large source file
	largeSource := generateLargeSource(5000)

	benchmark := &orizonTesting.BenchmarkTest{
		Name:       "large_file",
		SourceCode: largeSource,
	}

	framework.RunBenchmark(benchmark, b)
}

// BenchmarkManyFunctionsCompilation benchmarks compilation with many functions
func BenchmarkManyFunctionsCompilation(b *testing.B) {
	framework, err := orizonTesting.NewBenchmarkFramework(nil)
	if err != nil {
		b.Fatalf("Failed to create benchmark framework: %v", err)
	}
	defer framework.Cleanup()

	// Generate source with many functions
	manyFunctionsSource := generateManyFunctions(500)

	benchmark := &orizonTesting.BenchmarkTest{
		Name:       "many_functions",
		SourceCode: manyFunctionsSource,
	}

	framework.RunBenchmark(benchmark, b)
}

// BenchmarkGenericCompilation benchmarks compilation with generics
func BenchmarkGenericCompilation(b *testing.B) {
	framework, err := orizonTesting.NewBenchmarkFramework(nil)
	if err != nil {
		b.Fatalf("Failed to create benchmark framework: %v", err)
	}
	defer framework.Cleanup()

	benchmark := &orizonTesting.BenchmarkTest{
		Name: "generics",
		SourceCode: `
trait Display {
    fn display(&self) -> String;
}

struct Container<T> where T: Display {
    value: T,
}

impl<T> Container<T> where T: Display {
    fn new(value: T) -> Self {
        Container { value }
    }
    
    fn show(&self) -> String {
        self.value.display()
    }
}

impl Display for i32 {
    fn display(&self) -> String {
        self.to_string()
    }
}

impl Display for String {
    fn display(&self) -> String {
        self.clone()
    }
}

fn process<T: Display>(items: Vec<T>) -> Vec<String> {
    let mut results = Vec::new();
    for item in items {
        results.push(item.display());
    }
    results
}

fn main() {
    let int_container = Container::new(42);
    let string_container = Container::new("Hello".to_string());
    
    let int_items = vec![1, 2, 3, 4, 5];
    let string_items = vec!["a".to_string(), "b".to_string(), "c".to_string()];
    
    let int_results = process(int_items);
    let string_results = process(string_items);
    
    print(int_container.show());
    print(string_container.show());
    print(int_results.len());
    print(string_results.len());
}`,
	}

	framework.RunBenchmark(benchmark, b)
}

// BenchmarkOptimizationLevels benchmarks different optimization levels
func BenchmarkOptimizationLevels(b *testing.B) {
	sourceCode := `
fn fibonacci(n: i32) -> i32 {
    if n <= 1 {
        return n;
    }
    return fibonacci(n - 1) + fibonacci(n - 2);
}

fn main() {
    let result = fibonacci(20);
    print(result);
}`

	optimizationLevels := []string{"O0", "O1", "O2", "O3"}

	for _, level := range optimizationLevels {
		b.Run("optimization_"+level, func(b *testing.B) {
			config := orizonTesting.DefaultTestConfig()
			// Note: In a real implementation, you would modify the compiler flags here
			// config.CompilerFlags = []string{"-" + level}

			framework, err := orizonTesting.NewBenchmarkFramework(config)
			if err != nil {
				b.Fatalf("Failed to create benchmark framework: %v", err)
			}
			defer framework.Cleanup()

			benchmark := &orizonTesting.BenchmarkTest{
				Name:       "fibonacci_" + level,
				SourceCode: sourceCode,
			}

			framework.RunBenchmark(benchmark, b)
		})
	}
}

// Helper functions for generating benchmark code

func generateLargeSource(lines int) string {
	var code string
	code += "fn main() {\n"
	for i := 0; i < lines; i++ {
		code += "    let var" + string(rune('0'+i%10)) + " = " + string(rune('0'+i%10)) + ";\n"
	}
	code += "    print(\"Large file compiled\");\n"
	code += "}\n"
	return code
}

func generateManyFunctions(count int) string {
	var code string

	// Generate many small functions
	for i := 0; i < count; i++ {
		digit := i % 10
		code += "fn func" + string(rune('0'+digit)) + "_" + string(rune('0'+(i/10)%10)) + "() -> i32 {\n"
		code += "    return " + string(rune('0'+digit)) + ";\n"
		code += "}\n\n"
	}

	// Main function that calls some of them
	code += "fn main() {\n"
	code += "    let sum = func0_0() + func1_0() + func2_0();\n"
	code += "    print(sum);\n"
	code += "}\n"

	return code
}
