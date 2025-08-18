package testing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestConfig contains configuration for test execution
type TestConfig struct {
	Timeout       time.Duration
	CompilerPath  string
	TempDir       string
	KeepTempFiles bool
	Verbose       bool
}

// DefaultTestConfig returns a default test configuration
func DefaultTestConfig() *TestConfig {
	// Try to find the compiler in common locations
	compilerPath := "orizon.exe"
	possiblePaths := []string{
		"./orizon.exe",
		"./build/orizon.exe",
		"orizon.exe",
		"orizon",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			compilerPath = path
			break
		}
	}

	return &TestConfig{
		Timeout:       30 * time.Second,
		CompilerPath:  compilerPath,
		TempDir:       "",
		KeepTempFiles: false,
		Verbose:       false,
	}
}

// TestResult represents the result of a test execution
type TestResult struct {
	Success     bool
	Output      string
	ErrorOutput string
	ExitCode    int
	Duration    time.Duration
	Error       error
}

// CompilerTest represents a single compiler test case
type CompilerTest struct {
	Name        string
	SourceCode  string
	ExpectedOut string
	ExpectedErr string
	ShouldFail  bool
	Config      *TestConfig
}

// TestFramework provides infrastructure for running compiler tests
type TestFramework struct {
	config  *TestConfig
	tempDir string
}

// NewTestFramework creates a new test framework instance
func NewTestFramework(config *TestConfig) (*TestFramework, error) {
	if config == nil {
		config = DefaultTestConfig()
	}

	// Create temp directory if not specified
	tempDir := config.TempDir
	if tempDir == "" {
		var err error
		tempDir, err = os.MkdirTemp("", "orizon_test_*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp directory: %v", err)
		}
	}

	return &TestFramework{
		config:  config,
		tempDir: tempDir,
	}, nil
}

// Cleanup removes temporary files if not configured to keep them
func (tf *TestFramework) Cleanup() error {
	if !tf.config.KeepTempFiles && tf.tempDir != "" {
		return os.RemoveAll(tf.tempDir)
	}
	return nil
}

// RunTest executes a single compiler test
func (tf *TestFramework) RunTest(test *CompilerTest) *TestResult {
	start := time.Now()
	result := &TestResult{}

	// Create source file
	sourceFile := filepath.Join(tf.tempDir, test.Name+".oriz")
	if err := os.WriteFile(sourceFile, []byte(test.SourceCode), 0644); err != nil {
		result.Error = fmt.Errorf("failed to write source file: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	// Prepare output file
	outputFile := filepath.Join(tf.tempDir, test.Name+".exe")

	// Run compiler
	cmd := exec.Command(tf.config.CompilerPath, "build", sourceFile, "-o", outputFile)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set timeout using context
	if tf.config.Timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), tf.config.Timeout)
		defer cancel()
		cmd = exec.CommandContext(ctx, tf.config.CompilerPath, "build", sourceFile, "-o", outputFile)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	// Execute compiler
	err := cmd.Run()
	result.Output = stdout.String()
	result.ErrorOutput = stderr.String()
	result.Duration = time.Since(start)

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		}
		if test.ShouldFail {
			result.Success = true // Compilation failure was expected
		} else {
			result.Error = fmt.Errorf("compilation failed: %v", err)
			result.Success = false
		}
		return result
	}

	// Check if compilation should have failed
	if test.ShouldFail {
		result.Error = fmt.Errorf("compilation succeeded but was expected to fail")
		result.Success = false
		return result
	}

	// Run the compiled program if compilation succeeded
	if !test.ShouldFail {
		runResult := tf.runProgram(outputFile)
		result.Output += "\n--- Program Output ---\n" + runResult.Output
		if runResult.Error != nil {
			result.Error = runResult.Error
			result.Success = false
			return result
		}
	}

	// Validate output
	if test.ExpectedOut != "" && !strings.Contains(result.Output, test.ExpectedOut) {
		result.Error = fmt.Errorf("expected output '%s' not found in actual output", test.ExpectedOut)
		result.Success = false
		return result
	}

	if test.ExpectedErr != "" && !strings.Contains(result.ErrorOutput, test.ExpectedErr) {
		result.Error = fmt.Errorf("expected error '%s' not found in actual error output", test.ExpectedErr)
		result.Success = false
		return result
	}

	result.Success = true
	return result
}

// runProgram executes a compiled program and returns its output
func (tf *TestFramework) runProgram(programPath string) *TestResult {
	start := time.Now()
	result := &TestResult{}

	cmd := exec.Command(programPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set timeout using context
	if tf.config.Timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), tf.config.Timeout)
		defer cancel()
		cmd = exec.CommandContext(ctx, programPath)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	err := cmd.Run()
	result.Output = stdout.String()
	result.ErrorOutput = stderr.String()
	result.Duration = time.Since(start)

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		}
		result.Error = fmt.Errorf("program execution failed: %v", err)
		result.Success = false
	} else {
		result.Success = true
	}

	return result
}

// RunTestSuite executes a collection of tests
func (tf *TestFramework) RunTestSuite(tests []*CompilerTest, t *testing.T) {
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			result := tf.RunTest(test)

			if tf.config.Verbose {
				t.Logf("Test %s completed in %v", test.Name, result.Duration)
				if result.Output != "" {
					t.Logf("Output: %s", result.Output)
				}
				if result.ErrorOutput != "" {
					t.Logf("Error Output: %s", result.ErrorOutput)
				}
			}

			if !result.Success {
				if result.Error != nil {
					t.Errorf("Test failed: %v", result.Error)
				} else {
					t.Errorf("Test failed without specific error")
				}
			}
		})
	}
}

// BenchmarkFramework provides infrastructure for performance testing
type BenchmarkFramework struct {
	framework *TestFramework
}

// NewBenchmarkFramework creates a new benchmark framework
func NewBenchmarkFramework(config *TestConfig) (*BenchmarkFramework, error) {
	framework, err := NewTestFramework(config)
	if err != nil {
		return nil, err
	}
	return &BenchmarkFramework{framework: framework}, nil
}

// BenchmarkTest represents a benchmark test case
type BenchmarkTest struct {
	Name       string
	SourceCode string
	Iterations int
}

// RunBenchmark executes a benchmark test
func (bf *BenchmarkFramework) RunBenchmark(benchmark *BenchmarkTest, b *testing.B) {
	test := &CompilerTest{
		Name:       benchmark.Name,
		SourceCode: benchmark.SourceCode,
		Config:     bf.framework.config,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := bf.framework.RunTest(test)
		if !result.Success {
			b.Fatalf("Benchmark failed: %v", result.Error)
		}
	}
}

// Cleanup cleans up benchmark framework resources
func (bf *BenchmarkFramework) Cleanup() error {
	return bf.framework.Cleanup()
}

// GoldenFileTest represents a golden file test case
type GoldenFileTest struct {
	Name         string
	SourceCode   string
	GoldenFile   string
	UpdateGolden bool
}

// RunGoldenFileTest executes a golden file test
func (tf *TestFramework) RunGoldenFileTest(test *GoldenFileTest, t *testing.T) {
	// Compile the source code
	compilerTest := &CompilerTest{
		Name:       test.Name,
		SourceCode: test.SourceCode,
		Config:     tf.config,
	}

	result := tf.RunTest(compilerTest)
	if !result.Success {
		t.Fatalf("Compilation failed: %v", result.Error)
	}

	// Read or create golden file
	goldenPath := test.GoldenFile
	var expectedOutput string

	if test.UpdateGolden {
		// Update golden file with current output
		if err := os.WriteFile(goldenPath, []byte(result.Output), 0644); err != nil {
			t.Fatalf("Failed to update golden file: %v", err)
		}
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	// Read expected output from golden file
	if goldenData, err := os.ReadFile(goldenPath); err != nil {
		if os.IsNotExist(err) {
			// Create golden file if it doesn't exist
			if err := os.WriteFile(goldenPath, []byte(result.Output), 0644); err != nil {
				t.Fatalf("Failed to create golden file: %v", err)
			}
			t.Logf("Created golden file: %s", goldenPath)
			return
		}
		t.Fatalf("Failed to read golden file: %v", err)
	} else {
		expectedOutput = string(goldenData)
	}

	// Compare output with golden file
	if result.Output != expectedOutput {
		t.Errorf("Output differs from golden file %s", goldenPath)
		t.Errorf("Expected:\n%s", expectedOutput)
		t.Errorf("Actual:\n%s", result.Output)
	}
}

// TestReporter provides test result reporting
type TestReporter struct {
	writer io.Writer
}

// NewTestReporter creates a new test reporter
func NewTestReporter(writer io.Writer) *TestReporter {
	if writer == nil {
		writer = os.Stdout
	}
	return &TestReporter{writer: writer}
}

// ReportTestResult reports a single test result
func (tr *TestReporter) ReportTestResult(test *CompilerTest, result *TestResult) {
	status := "PASS"
	if !result.Success {
		status = "FAIL"
	}

	fmt.Fprintf(tr.writer, "[%s] %s (%.2fs)\n", status, test.Name, result.Duration.Seconds())

	if !result.Success && result.Error != nil {
		fmt.Fprintf(tr.writer, "  Error: %v\n", result.Error)
	}
}

// ReportSummary reports a summary of test results
func (tr *TestReporter) ReportSummary(results []*TestResult) {
	total := len(results)
	passed := 0
	failed := 0
	totalDuration := time.Duration(0)

	for _, result := range results {
		if result.Success {
			passed++
		} else {
			failed++
		}
		totalDuration += result.Duration
	}

	fmt.Fprintf(tr.writer, "\n--- Test Summary ---\n")
	fmt.Fprintf(tr.writer, "Total: %d, Passed: %d, Failed: %d\n", total, passed, failed)
	fmt.Fprintf(tr.writer, "Total Duration: %.2fs\n", totalDuration.Seconds())

	if failed > 0 {
		fmt.Fprintf(tr.writer, "SOME TESTS FAILED\n")
	} else {
		fmt.Fprintf(tr.writer, "ALL TESTS PASSED\n")
	}
}
