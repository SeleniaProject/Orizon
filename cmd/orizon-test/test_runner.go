package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/orizon-lang/orizon/internal/testing"
)

// TestRunner orchestrates the execution of all test suites
type TestRunner struct {
	config    *testing.TestConfig
	generator *testing.ReportGenerator
}

// NewTestRunner creates a new test runner
func NewTestRunner(config *testing.TestConfig) *TestRunner {
	return &TestRunner{
		config:    config,
		generator: testing.NewReportGenerator(),
	}
}

// RunAllTests executes all test suites and generates comprehensive reports
func (tr *TestRunner) RunAllTests() error {
	fmt.Println("Starting Orizon Compiler Test Suite")
	fmt.Println("====================================")

	// Set environment information
	env := &testing.TestEnvironment{
		CompilerVersion: "1.0.0-dev",
		Platform:        runtime.GOOS,
		Architecture:    runtime.GOARCH,
		GoVersion:       runtime.Version(),
		Variables: map[string]string{
			"ORIZON_HOME": os.Getenv("ORIZON_HOME"),
			"PATH":        os.Getenv("PATH"),
		},
	}
	tr.generator.SetEnvironment(env)

	// Run unit tests
	if err := tr.runUnitTests(); err != nil {
		return fmt.Errorf("unit tests failed: %v", err)
	}

	// Run integration tests
	if err := tr.runIntegrationTests(); err != nil {
		return fmt.Errorf("integration tests failed: %v", err)
	}

	// Run end-to-end tests
	if err := tr.runE2ETests(); err != nil {
		return fmt.Errorf("e2e tests failed: %v", err)
	}

	// Run benchmark tests
	if err := tr.runBenchmarkTests(); err != nil {
		return fmt.Errorf("benchmark tests failed: %v", err)
	}

	// Finalize and generate reports
	tr.generator.Finalize()
	return tr.generateReports()
}

// runUnitTests executes unit test suite
func (tr *TestRunner) runUnitTests() error {
	fmt.Println("\nRunning Unit Tests...")

	suite := &testing.TestSuite{
		Name:  "Unit Tests",
		Tests: make([]*testing.TestCase, 0),
	}
	start := time.Now()

	// Create test framework
	framework, err := testing.NewTestFramework(tr.config)
	if err != nil {
		return err
	}
	defer framework.Cleanup()

	// Define unit tests
	unitTests := tr.createUnitTests()

	// Execute each test
	for _, test := range unitTests {
		result := framework.RunTest(test)
		testCase := testing.ConvertTestResultToCase(test, result)
		suite.Tests = append(suite.Tests, testCase)

		if result.Success {
			suite.Passed++
			if tr.config.Verbose {
				fmt.Printf("  ✓ %s (%.2fs)\n", test.Name, result.Duration.Seconds())
			}
		} else {
			suite.Failed++
			fmt.Printf("  ✗ %s (%.2fs): %v\n", test.Name, result.Duration.Seconds(), result.Error)
		}
	}

	suite.Duration = time.Since(start)
	tr.generator.AddSuite(suite)

	fmt.Printf("Unit Tests: %d passed, %d failed (%.2fs)\n",
		suite.Passed, suite.Failed, suite.Duration.Seconds())

	return nil
}

// runIntegrationTests executes integration test suite
func (tr *TestRunner) runIntegrationTests() error {
	fmt.Println("\nRunning Integration Tests...")

	suite := &testing.TestSuite{
		Name:  "Integration Tests",
		Tests: make([]*testing.TestCase, 0),
	}
	start := time.Now()

	// Create test framework
	framework, err := testing.NewTestFramework(tr.config)
	if err != nil {
		return err
	}
	defer framework.Cleanup()

	// Define integration tests
	integrationTests := tr.createIntegrationTests()

	// Execute each test
	for _, test := range integrationTests {
		result := framework.RunTest(test)
		testCase := testing.ConvertTestResultToCase(test, result)
		suite.Tests = append(suite.Tests, testCase)

		if result.Success {
			suite.Passed++
			if tr.config.Verbose {
				fmt.Printf("  ✓ %s (%.2fs)\n", test.Name, result.Duration.Seconds())
			}
		} else {
			suite.Failed++
			fmt.Printf("  ✗ %s (%.2fs): %v\n", test.Name, result.Duration.Seconds(), result.Error)
		}
	}

	suite.Duration = time.Since(start)
	tr.generator.AddSuite(suite)

	fmt.Printf("Integration Tests: %d passed, %d failed (%.2fs)\n",
		suite.Passed, suite.Failed, suite.Duration.Seconds())

	return nil
}

// runE2ETests executes end-to-end test suite
func (tr *TestRunner) runE2ETests() error {
	fmt.Println("\nRunning End-to-End Tests...")

	suite := &testing.TestSuite{
		Name:  "End-to-End Tests",
		Tests: make([]*testing.TestCase, 0),
	}
	start := time.Now()

	// Create test framework
	framework, err := testing.NewTestFramework(tr.config)
	if err != nil {
		return err
	}
	defer framework.Cleanup()

	// Define e2e tests
	e2eTests := tr.createE2ETests()

	// Execute each test
	for _, test := range e2eTests {
		result := framework.RunTest(test)
		testCase := testing.ConvertTestResultToCase(test, result)
		suite.Tests = append(suite.Tests, testCase)

		if result.Success {
			suite.Passed++
			if tr.config.Verbose {
				fmt.Printf("  ✓ %s (%.2fs)\n", test.Name, result.Duration.Seconds())
			}
		} else {
			suite.Failed++
			fmt.Printf("  ✗ %s (%.2fs): %v\n", test.Name, result.Duration.Seconds(), result.Error)
		}
	}

	suite.Duration = time.Since(start)
	tr.generator.AddSuite(suite)

	fmt.Printf("End-to-End Tests: %d passed, %d failed (%.2fs)\n",
		suite.Passed, suite.Failed, suite.Duration.Seconds())

	return nil
}

// runBenchmarkTests executes benchmark test suite
func (tr *TestRunner) runBenchmarkTests() error {
	fmt.Println("\nRunning Benchmark Tests...")

	// Skip benchmark tests if benchmark framework is not fully implemented
	fmt.Println("  Skipping benchmark tests (implementation pending)")

	suite := &testing.TestSuite{
		Name:    "Benchmark Tests",
		Tests:   make([]*testing.TestCase, 0),
		Passed:  0,
		Failed:  0,
		Skipped: 1,
	}

	// Add a placeholder test case
	testCase := &testing.TestCase{
		Name:      "benchmark_placeholder",
		ClassName: "BenchmarkTest",
		Duration:  time.Millisecond,
		Status:    testing.TestStatusSkipped,
		Output:    "Benchmark tests not yet implemented",
	}

	suite.Tests = append(suite.Tests, testCase)
	suite.Duration = time.Millisecond
	tr.generator.AddSuite(suite)

	fmt.Printf("Benchmark Tests: skipped (implementation pending)\n")
	return nil
}

// generateReports creates comprehensive test reports
func (tr *TestRunner) generateReports() error {
	fmt.Println("\nGenerating Test Reports...")

	// Ensure reports directory exists
	reportsDir := "test_reports"
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return fmt.Errorf("failed to create reports directory: %v", err)
	}

	// Generate JSON report
	jsonFile := filepath.Join(reportsDir, "test_report.json")
	if err := tr.generator.SaveToFile(jsonFile, "json"); err != nil {
		return fmt.Errorf("failed to generate JSON report: %v", err)
	}
	fmt.Printf("  Generated JSON report: %s\n", jsonFile)

	// Generate XML report (JUnit format)
	xmlFile := filepath.Join(reportsDir, "test_report.xml")
	if err := tr.generator.SaveToFile(xmlFile, "xml"); err != nil {
		return fmt.Errorf("failed to generate XML report: %v", err)
	}
	fmt.Printf("  Generated XML report: %s\n", xmlFile)

	// Generate HTML report
	htmlFile := filepath.Join(reportsDir, "test_report.html")
	if err := tr.generator.SaveToFile(htmlFile, "html"); err != nil {
		return fmt.Errorf("failed to generate HTML report: %v", err)
	}
	fmt.Printf("  Generated HTML report: %s\n", htmlFile)

	return nil
}

// createUnitTests defines basic unit tests
func (tr *TestRunner) createUnitTests() []*testing.CompilerTest {
	return []*testing.CompilerTest{
		{
			Name: "basic_hello_world",
			SourceCode: `
fn main() {
    println("Hello, World!");
}`,
			ExpectedOut: "Hello, World!",
			ShouldFail:  false,
		},
		{
			Name: "arithmetic_operations",
			SourceCode: `
fn main() {
    let result = 2 + 3 * 4;
    println(result);
}`,
			ExpectedOut: "14",
			ShouldFail:  false,
		},
		{
			Name: "function_calls",
			SourceCode: `
fn add(a: i32, b: i32) -> i32 {
    return a + b;
}

fn main() {
    let result = add(5, 3);
    println(result);
}`,
			ExpectedOut: "8",
			ShouldFail:  false,
		},
	}
}

// createIntegrationTests defines integration tests
func (tr *TestRunner) createIntegrationTests() []*testing.CompilerTest {
	return []*testing.CompilerTest{
		{
			Name: "struct_operations",
			SourceCode: `
struct Point {
    x: i32,
    y: i32,
}

fn main() {
    let p = Point { x: 10, y: 20 };
    println(p.x + p.y);
}`,
			ExpectedOut: "30",
			ShouldFail:  false,
		},
		{
			Name: "control_flow",
			SourceCode: `
fn main() {
    let x = 5;
    if x > 3 {
        println("greater");
    } else {
        println("lesser");
    }
}`,
			ExpectedOut: "greater",
			ShouldFail:  false,
		},
	}
}

// createE2ETests defines end-to-end tests
func (tr *TestRunner) createE2ETests() []*testing.CompilerTest {
	return []*testing.CompilerTest{
		{
			Name: "fibonacci_recursive",
			SourceCode: `
fn fibonacci(n: i32) -> i32 {
    if n <= 1 {
        return n;
    }
    return fibonacci(n - 1) + fibonacci(n - 2);
}

fn main() {
    println(fibonacci(10));
}`,
			ExpectedOut: "55",
			ShouldFail:  false,
		},
	}
}

// createBenchmarkTests defines benchmark tests
func (tr *TestRunner) createBenchmarkTests() []*testing.BenchmarkTest {
	return []*testing.BenchmarkTest{
		{
			Name: "compilation_speed",
			SourceCode: `
fn main() {
    println("Benchmark test");
}`,
			Iterations: 100,
		},
	}
}

// RunTestsWithConfig runs all tests with the given configuration
func RunTestsWithConfig(config *testing.TestConfig) error {
	runner := NewTestRunner(config)

	fmt.Printf("Orizon Compiler Test Runner\n")
	fmt.Printf("Using compiler: %s\n", config.CompilerPath)
	fmt.Printf("Timeout: %v\n", config.Timeout)
	fmt.Printf("Verbose: %v\n", config.Verbose)
	fmt.Println()

	// Run all tests
	if err := runner.RunAllTests(); err != nil {
		return fmt.Errorf("test execution failed: %v", err)
	}

	fmt.Println("\nTest execution completed successfully!")
	return nil
}
