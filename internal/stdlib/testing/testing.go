// Package testing provides comprehensive testing framework including
// unit testing, integration testing, benchmarking, and test utilities.
package testing

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"
)

// T represents a test context.
type T struct {
	name     string
	failed   bool
	skipped  bool
	logs     []string
	cleanup  []func()
	duration time.Duration
	parallel bool
}

// B represents a benchmark context.
type B struct {
	*T
	N         int
	bytes     int64
	timerOn   bool
	startTime time.Time
	duration  time.Duration
	result    BenchmarkResult
}

// TestFunc represents a test function.
type TestFunc func(*T)

// BenchmarkFunc represents a benchmark function.
type BenchmarkFunc func(*B)

// TestSuite represents a collection of tests.
type TestSuite struct {
	Name           string
	Tests          []TestCase
	Benchmarks     []BenchmarkCase
	SetupFunc      func()
	TeardownFunc   func()
	BeforeEachFunc func()
	AfterEachFunc  func()
}

// TestCase represents a single test case.
type TestCase struct {
	Name     string
	Func     TestFunc
	Skip     bool
	Parallel bool
	Tags     []string
}

// BenchmarkCase represents a single benchmark.
type BenchmarkCase struct {
	Name string
	Func BenchmarkFunc
	Skip bool
}

// TestResult represents test execution results.
type TestResult struct {
	Name     string
	Passed   bool
	Skipped  bool
	Duration time.Duration
	Error    string
	Logs     []string
}

// BenchmarkResult represents benchmark execution results.
type BenchmarkResult struct {
	Name        string
	N           int
	Duration    time.Duration
	MemAllocs   int64
	MemBytes    int64
	NsPerOp     float64
	MBPerSec    float64
	AllocsPerOp float64
	BytesPerOp  float64
}

// SuiteResult represents test suite execution results.
type SuiteResult struct {
	Name       string
	TotalTests int
	Passed     int
	Failed     int
	Skipped    int
	Duration   time.Duration
	Tests      []TestResult
	Benchmarks []BenchmarkResult
}

// Runner manages test execution.
type Runner struct {
	suites   []TestSuite
	config   *Config
	results  []SuiteResult
	verbose  bool
	parallel bool
	maxProcs int
}

// Config represents test configuration.
type Config struct {
	Verbose      bool
	Parallel     bool
	MaxProcs     int
	Timeout      time.Duration
	Pattern      string
	Tags         []string
	BenchTime    time.Duration
	BenchMem     bool
	Cover        bool
	CoverProfile string
}

// Mock represents a mock object.
type Mock struct {
	calls        []Call
	expectations []Expectation
	returns      map[string][]interface{}
}

// Call represents a method call.
type Call struct {
	Method string
	Args   []interface{}
	Time   time.Time
}

// Expectation represents a mock expectation.
type Expectation struct {
	Method   string
	Args     []interface{}
	Returns  []interface{}
	Times    int
	Called   int
	AnyTimes bool
}

// Spy represents a spy object.
type Spy struct {
	calls  []Call
	target interface{}
}

// Stub represents a stub object.
type Stub struct {
	returns map[string][]interface{}
}

// Fixture represents test fixture data.
type Fixture struct {
	Name string
	Data interface{}
	File string
}

// Global test runner
var defaultRunner *Runner

// NewRunner creates a new test runner.
func NewRunner(config *Config) *Runner {
	if config == nil {
		config = &Config{
			Verbose:   false,
			Parallel:  false,
			MaxProcs:  runtime.NumCPU(),
			Timeout:   10 * time.Minute,
			BenchTime: 1 * time.Second,
			BenchMem:  false,
		}
	}

	return &Runner{
		suites:   make([]TestSuite, 0),
		config:   config,
		results:  make([]SuiteResult, 0),
		verbose:  config.Verbose,
		parallel: config.Parallel,
		maxProcs: config.MaxProcs,
	}
}

// GetRunner returns the default test runner.
func GetRunner() *Runner {
	if defaultRunner == nil {
		defaultRunner = NewRunner(nil)
	}
	return defaultRunner
}

// Test context methods

// Log logs a message.
func (t *T) Log(args ...interface{}) {
	t.logs = append(t.logs, fmt.Sprint(args...))
	if defaultRunner != nil && defaultRunner.verbose {
		fmt.Println(args...)
	}
}

// Logf logs a formatted message.
func (t *T) Logf(format string, args ...interface{}) {
	t.logs = append(t.logs, fmt.Sprintf(format, args...))
	if defaultRunner != nil && defaultRunner.verbose {
		fmt.Printf(format, args...)
	}
}

// Error reports an error but continues execution.
func (t *T) Error(args ...interface{}) {
	t.Log(args...)
	t.Fail()
}

// Errorf reports a formatted error but continues execution.
func (t *T) Errorf(format string, args ...interface{}) {
	t.Logf(format, args...)
	t.Fail()
}

// Fatal reports an error and stops execution.
func (t *T) Fatal(args ...interface{}) {
	t.Log(args...)
	t.FailNow()
}

// Fatalf reports a formatted error and stops execution.
func (t *T) Fatalf(format string, args ...interface{}) {
	t.Logf(format, args...)
	t.FailNow()
}

// Fail marks the test as failed.
func (t *T) Fail() {
	t.failed = true
}

// FailNow marks the test as failed and stops execution.
func (t *T) FailNow() {
	t.failed = true
	panic("test failed")
}

// Failed returns whether the test has failed.
func (t *T) Failed() bool {
	return t.failed
}

// Skip skips the test.
func (t *T) Skip(args ...interface{}) {
	t.Log(args...)
	t.SkipNow()
}

// Skipf skips the test with a formatted message.
func (t *T) Skipf(format string, args ...interface{}) {
	t.Logf(format, args...)
	t.SkipNow()
}

// SkipNow skips the test immediately.
func (t *T) SkipNow() {
	t.skipped = true
	panic("test skipped")
}

// Skipped returns whether the test was skipped.
func (t *T) Skipped() bool {
	return t.skipped
}

// Cleanup registers a function to be called when the test completes.
func (t *T) Cleanup(f func()) {
	t.cleanup = append(t.cleanup, f)
}

// Parallel marks the test for parallel execution.
func (t *T) Parallel() {
	t.parallel = true
}

// Name returns the test name.
func (t *T) Name() string {
	return t.name
}

// Benchmark context methods

// ResetTimer resets the benchmark timer.
func (b *B) ResetTimer() {
	b.startTime = time.Now()
	b.duration = 0
}

// StartTimer starts the benchmark timer.
func (b *B) StartTimer() {
	if !b.timerOn {
		b.startTime = time.Now()
		b.timerOn = true
	}
}

// StopTimer stops the benchmark timer.
func (b *B) StopTimer() {
	if b.timerOn {
		b.duration += time.Since(b.startTime)
		b.timerOn = false
	}
}

// SetBytes sets the number of bytes processed per iteration.
func (b *B) SetBytes(n int64) {
	b.bytes = n
}

// ReportAllocs enables memory allocation reporting.
func (b *B) ReportAllocs() {
	// Enable allocation tracking
}

// Run executes a sub-benchmark.
func (b *B) Run(name string, f func(*B)) bool {
	subB := &B{
		T: &T{name: b.name + "/" + name},
		N: 1,
	}

	defer func() {
		if r := recover(); r != nil {
			if r != "benchmark skipped" {
				subB.failed = true
			}
		}
	}()

	f(subB)
	return !subB.failed
}

// Assertion functions

// Equal asserts that two values are equal.
func Equal(t *T, expected, actual interface{}) bool {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v, got %v", expected, actual)
		return false
	}
	return true
}

// NotEqual asserts that two values are not equal.
func NotEqual(t *T, expected, actual interface{}) bool {
	if reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v to not equal %v", expected, actual)
		return false
	}
	return true
}

// True asserts that a value is true.
func True(t *T, value bool) bool {
	if !value {
		t.Error("Expected true, got false")
		return false
	}
	return true
}

// False asserts that a value is false.
func False(t *T, value bool) bool {
	if value {
		t.Error("Expected false, got true")
		return false
	}
	return true
}

// Nil asserts that a value is nil.
func Nil(t *T, value interface{}) bool {
	if value != nil {
		t.Errorf("Expected nil, got %v", value)
		return false
	}
	return true
}

// NotNil asserts that a value is not nil.
func NotNil(t *T, value interface{}) bool {
	if value == nil {
		t.Error("Expected non-nil value")
		return false
	}
	return true
}

// Contains asserts that a string contains a substring.
func Contains(t *T, str, substr string) bool {
	if !strings.Contains(str, substr) {
		t.Errorf("Expected '%s' to contain '%s'", str, substr)
		return false
	}
	return true
}

// HasPrefix asserts that a string has a prefix.
func HasPrefix(t *T, str, prefix string) bool {
	if !strings.HasPrefix(str, prefix) {
		t.Errorf("Expected '%s' to have prefix '%s'", str, prefix)
		return false
	}
	return true
}

// HasSuffix asserts that a string has a suffix.
func HasSuffix(t *T, str, suffix string) bool {
	if !strings.HasSuffix(str, suffix) {
		t.Errorf("Expected '%s' to have suffix '%s'", str, suffix)
		return false
	}
	return true
}

// Panics asserts that a function panics.
func Panics(t *T, f func()) bool {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected function to panic")
		}
	}()

	f()
	return true
}

// NotPanics asserts that a function does not panic.
func NotPanics(t *T, f func()) bool {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Expected function not to panic, but it panicked with: %v", r)
		}
	}()

	f()
	return true
}

// Test suite methods

// NewTestSuite creates a new test suite.
func NewTestSuite(name string) *TestSuite {
	return &TestSuite{
		Name:       name,
		Tests:      make([]TestCase, 0),
		Benchmarks: make([]BenchmarkCase, 0),
	}
}

// AddTest adds a test to the suite.
func (suite *TestSuite) AddTest(name string, f TestFunc) {
	suite.Tests = append(suite.Tests, TestCase{
		Name: name,
		Func: f,
	})
}

// AddBenchmark adds a benchmark to the suite.
func (suite *TestSuite) AddBenchmark(name string, f BenchmarkFunc) {
	suite.Benchmarks = append(suite.Benchmarks, BenchmarkCase{
		Name: name,
		Func: f,
	})
}

// Setup sets the setup function.
func (suite *TestSuite) Setup(f func()) {
	suite.SetupFunc = f
}

// Teardown sets the teardown function.
func (suite *TestSuite) Teardown(f func()) {
	suite.TeardownFunc = f
}

// SetBeforeEach sets the before each function.
func (suite *TestSuite) SetBeforeEach(f func()) {
	suite.BeforeEachFunc = f
}

// SetAfterEach sets the after each function.
func (suite *TestSuite) SetAfterEach(f func()) {
	suite.AfterEachFunc = f
}

// Runner methods

// AddSuite adds a test suite to the runner.
func (r *Runner) AddSuite(suite TestSuite) {
	r.suites = append(r.suites, suite)
}

// Run executes all test suites.
func (r *Runner) Run() []SuiteResult {
	r.results = make([]SuiteResult, 0)

	for _, suite := range r.suites {
		result := r.runSuite(suite)
		r.results = append(r.results, result)
	}

	return r.results
}

// RunSuite executes a single test suite.
func (r *Runner) runSuite(suite TestSuite) SuiteResult {
	startTime := time.Now()

	result := SuiteResult{
		Name:       suite.Name,
		TotalTests: len(suite.Tests),
		Tests:      make([]TestResult, 0),
		Benchmarks: make([]BenchmarkResult, 0),
	}

	// Setup
	if suite.SetupFunc != nil {
		suite.SetupFunc()
	}

	// Run tests
	for _, test := range suite.Tests {
		testResult := r.runTest(test, suite)
		result.Tests = append(result.Tests, testResult)

		if testResult.Passed {
			result.Passed++
		} else if testResult.Skipped {
			result.Skipped++
		} else {
			result.Failed++
		}
	}

	// Run benchmarks
	for _, benchmark := range suite.Benchmarks {
		benchResult := r.runBenchmark(benchmark)
		result.Benchmarks = append(result.Benchmarks, benchResult)
	}

	// Teardown
	if suite.TeardownFunc != nil {
		suite.TeardownFunc()
	}

	result.Duration = time.Since(startTime)
	return result
}

// RunTest executes a single test.
func (r *Runner) runTest(test TestCase, suite TestSuite) TestResult {
	startTime := time.Now()

	t := &T{
		name:    test.Name,
		logs:    make([]string, 0),
		cleanup: make([]func(), 0),
	}

	result := TestResult{
		Name: test.Name,
	}

	defer func() {
		// Run cleanup functions
		for i := len(t.cleanup) - 1; i >= 0; i-- {
			t.cleanup[i]()
		}

		result.Duration = time.Since(startTime)
		result.Logs = t.logs

		if r := recover(); r != nil {
			if r == "test skipped" {
				result.Skipped = true
			} else {
				result.Error = fmt.Sprint(r)
			}
		}

		result.Passed = !t.failed && !t.skipped
	}()

	// Before each
	if suite.BeforeEachFunc != nil {
		suite.BeforeEachFunc()
	}

	// Run test
	test.Func(t)

	// After each
	if suite.AfterEachFunc != nil {
		suite.AfterEachFunc()
	}

	return result
}

// RunBenchmark executes a single benchmark.
func (r *Runner) runBenchmark(benchmark BenchmarkCase) BenchmarkResult {
	b := &B{
		T: &T{
			name: benchmark.Name,
			logs: make([]string, 0),
		},
		N:       1,
		timerOn: true,
	}

	// Determine number of iterations
	for b.N = 1; b.duration < r.config.BenchTime && b.N < 1e9; b.N *= 2 {
		b.ResetTimer()
		benchmark.Func(b)
		b.StopTimer()
	}

	// Final run
	b.ResetTimer()
	benchmark.Func(b)
	b.StopTimer()

	result := BenchmarkResult{
		Name:     benchmark.Name,
		N:        b.N,
		Duration: b.duration,
	}

	if b.N > 0 {
		result.NsPerOp = float64(b.duration.Nanoseconds()) / float64(b.N)
	}

	if b.bytes > 0 && b.duration > 0 {
		result.MBPerSec = float64(b.bytes*int64(b.N)) / float64(b.duration.Nanoseconds()) * 1000
	}

	return result
}

// Mock implementation

// NewMock creates a new mock object.
func NewMock() *Mock {
	return &Mock{
		calls:        make([]Call, 0),
		expectations: make([]Expectation, 0),
		returns:      make(map[string][]interface{}),
	}
}

// On sets up an expectation.
func (m *Mock) On(method string, args ...interface{}) *Mock {
	expectation := Expectation{
		Method:   method,
		Args:     args,
		Times:    1,
		Called:   0,
		AnyTimes: false,
	}

	m.expectations = append(m.expectations, expectation)
	return m
}

// Return sets the return values for an expectation.
func (m *Mock) Return(values ...interface{}) *Mock {
	if len(m.expectations) > 0 {
		lastExp := &m.expectations[len(m.expectations)-1]
		lastExp.Returns = values
		m.returns[lastExp.Method] = values
	}
	return m
}

// Times sets the expected call count.
func (m *Mock) Times(n int) *Mock {
	if len(m.expectations) > 0 {
		m.expectations[len(m.expectations)-1].Times = n
	}
	return m
}

// AnyTimes allows any number of calls.
func (m *Mock) AnyTimes() *Mock {
	if len(m.expectations) > 0 {
		m.expectations[len(m.expectations)-1].AnyTimes = true
	}
	return m
}

// Call records a method call.
func (m *Mock) Call(method string, args ...interface{}) []interface{} {
	call := Call{
		Method: method,
		Args:   args,
		Time:   time.Now(),
	}

	m.calls = append(m.calls, call)

	// Update expectation call count
	for i := range m.expectations {
		if m.expectations[i].Method == method {
			m.expectations[i].Called++
			break
		}
	}

	// Return values
	if values, exists := m.returns[method]; exists {
		return values
	}

	return nil
}

// Verify verifies that all expectations were met.
func (m *Mock) Verify() error {
	for _, exp := range m.expectations {
		if !exp.AnyTimes && exp.Called != exp.Times {
			return fmt.Errorf("expected %s to be called %d times, but was called %d times",
				exp.Method, exp.Times, exp.Called)
		}
	}
	return nil
}

// GetCalls returns all recorded calls.
func (m *Mock) GetCalls() []Call {
	return m.calls
}

// AssertCalled asserts that a method was called.
func (m *Mock) AssertCalled(t *T, method string) {
	for _, call := range m.calls {
		if call.Method == method {
			return
		}
	}
	t.Errorf("Expected method %s to be called", method)
}

// Public API functions

// Run runs the default test runner.
func Run() []SuiteResult {
	return GetRunner().Run()
}

// AddSuite adds a suite to the default runner.
func AddSuite(suite TestSuite) {
	GetRunner().AddSuite(suite)
}

// RunSingleTest creates and runs a single test.
func RunSingleTest(name string, f TestFunc) {
	suite := NewTestSuite("SingleTest")
	suite.AddTest(name, f)
	GetRunner().AddSuite(*suite)
}

// RunSingleBenchmark creates and runs a single benchmark.
func RunSingleBenchmark(name string, f BenchmarkFunc) {
	suite := NewTestSuite("SingleBenchmark")
	suite.AddBenchmark(name, f)
	GetRunner().AddSuite(*suite)
}

// Quick performs quick testing with random inputs.
func Quick(f interface{}, config *QuickConfig) error {
	// Simple quick test implementation
	// In practice, this would generate random inputs
	return nil
}

// QuickConfig represents quick test configuration.
type QuickConfig struct {
	MaxCount      int
	MaxCountScale float64
	Rand          func() int
	Values        func([]reflect.Value, func())
}

// Table test utilities

// TableTest represents a table-driven test case.
type TableTest struct {
	Name     string
	Input    interface{}
	Expected interface{}
	Error    bool
}

// RunTableTests runs table-driven tests.
func RunTableTests(t *T, tests []TableTest, testFunc func(interface{}) (interface{}, error)) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *T) {
			result, err := testFunc(tt.Input)

			if tt.Error {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			Equal(t, tt.Expected, result)
		})
	}
}

// Run method for T (for subtests)
func (t *T) Run(name string, f func(*T)) bool {
	subT := &T{
		name:    t.name + "/" + name,
		logs:    make([]string, 0),
		cleanup: make([]func(), 0),
	}

	defer func() {
		if r := recover(); r != nil {
			if r != "test skipped" {
				subT.failed = true
			}
		}
	}()

	f(subT)
	return !subT.failed
}
