package main

import (
	"fmt"
	"os"

	"github.com/orizon-lang/orizon/internal/testing"
)

func main() {
	fmt.Println("Orizon Compiler Testing Framework Demo")
	fmt.Println("=====================================")

	// Create test configuration.
	config := testing.DefaultTestConfig()
	config.Verbose = true

	// Create test framework.
	framework, err := testing.NewTestFramework(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create test framework: %v\n", err)
		os.Exit(1)
	}
	defer framework.Cleanup()

	// Create a simple test.
	test := &testing.CompilerTest{
		Name: "hello_world_test",
		SourceCode: `
fn main() {
    println("Hello, Testing Framework!");
}`,
		ExpectedOut: "Hello, Testing Framework!",
		ShouldFail:  false,
	}

	// Run the test.
	result := framework.RunTest(test)

	// Display results.
	if result.Success {
		fmt.Printf("✓ Test '%s' passed (%.2fs)\n", test.Name, result.Duration.Seconds())

		if result.Output != "" {
			fmt.Printf("  Output: %s\n", result.Output)
		}
	} else {
		fmt.Printf("✗ Test '%s' failed (%.2fs): %v\n", test.Name, result.Duration.Seconds(), result.Error)

		if result.ErrorOutput != "" {
			fmt.Printf("  Error: %s\n", result.ErrorOutput)
		}
	}

	// Demonstrate report generation.
	fmt.Println("\nGenerating Test Report...")

	generator := testing.NewReportGenerator()

	// Create a test suite.
	suite := &testing.TestSuite{
		Name:     "Demo Test Suite",
		Tests:    []*testing.TestCase{testing.ConvertTestResultToCase(test, result)},
		Duration: result.Duration,
	}

	if result.Success {
		suite.Passed = 1
	} else {
		suite.Failed = 1
	}

	generator.AddSuite(suite)
	generator.Finalize()

	// Generate JSON report.
	fmt.Println("Generated test report structure")
	fmt.Printf("Total Tests: %d\n", 1)
	fmt.Printf("Passed: %d\n", suite.Passed)
	fmt.Printf("Failed: %d\n", suite.Failed)
	fmt.Printf("Duration: %v\n", suite.Duration)

	fmt.Println("\nTesting framework is ready for comprehensive compiler validation!")
}
