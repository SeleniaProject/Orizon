// Package types provides exception effect system for type-level exception tracking.
// This module implements comprehensive exception effect tracking, try-catch typing,.
// and exception safety guarantees for the Orizon language.
package types

import (
	"fmt"
	"strings"
	"sync"
)

// ExceptionEffectKind represents different kinds of exception effects.
type ExceptionEffectKind int

const (
	// Exception throwing effects.
	ExceptionEffectThrows ExceptionEffectKind = iota
	ExceptionEffectArithmeticError
	ExceptionEffectNullPointer
	ExceptionEffectIndexOutOfBounds
	ExceptionEffectTypeError
	ExceptionEffectIOError

	// Exception handling effects.
	ExceptionEffectCatches
	ExceptionEffectHandles
)

// String returns the string representation of an ExceptionEffectKind.
func (eek ExceptionEffectKind) String() string {
	switch eek {
	case ExceptionEffectThrows:
		return "Throws"
	case ExceptionEffectArithmeticError:
		return "ArithmeticError"
	case ExceptionEffectNullPointer:
		return "NullPointer"
	case ExceptionEffectIndexOutOfBounds:
		return "IndexOutOfBounds"
	case ExceptionEffectTypeError:
		return "TypeError"
	case ExceptionEffectIOError:
		return "IOError"
	case ExceptionEffectCatches:
		return "Catches"
	case ExceptionEffectHandles:
		return "Handles"
	default:
		return fmt.Sprintf("Unknown(%d)", int(eek))
	}
}

// ExceptionEffect represents an exception effect in the type system.
type ExceptionEffect struct {
	Spec          *ExceptionSpec
	Context       map[string]interface{}
	TypeName      string
	Description   string
	Kind          ExceptionEffectKind
	Level         EffectLevel
	Deterministic bool
	Safe          bool
}

// NewExceptionEffect creates a new ExceptionEffect.
func NewExceptionEffect(kind ExceptionEffectKind, level EffectLevel) *ExceptionEffect {
	return &ExceptionEffect{
		Kind:          kind,
		Level:         level,
		Context:       make(map[string]interface{}),
		Deterministic: true,
		Safe:          false, // Exceptions are generally not safe
	}
}

// String returns the string representation of an ExceptionEffect.
func (ee *ExceptionEffect) String() string {
	return fmt.Sprintf("%s(level=%d)", ee.Kind.String(), ee.Level)
}

// ExceptionKind represents different categories of exceptions.
type ExceptionKind int

const (
	// Built-in exception types.
	ExceptionNone ExceptionKind = iota
	ExceptionRuntime
	ExceptionNullPointer
	ExceptionIndexOutOfBounds
	ExceptionDivisionByZero
	ExceptionStackOverflow
	ExceptionOutOfMemory
	ExceptionInvalidOperation
	ExceptionTypeError
	ExceptionParseError
	// I/O exceptions.
	ExceptionIOError
	ExceptionFileNotFound
	ExceptionPermissionDenied
	ExceptionNetworkTimeout
	ExceptionConnectionFailed
	// Concurrency exceptions.
	ExceptionDeadlock
	ExceptionRaceCondition
	ExceptionThreadAbort
	ExceptionSynchronization
	// System exceptions.
	ExceptionSystemError
	ExceptionResourceExhausted
	ExceptionSecurityViolation
	ExceptionConfigurationError
	// User-defined exceptions.
	ExceptionUserDefined
	ExceptionCustom
)

// String returns the string representation of an ExceptionKind.
func (ek ExceptionKind) String() string {
	switch ek {
	case ExceptionNone:
		return "None"
	case ExceptionRuntime:
		return "Runtime"
	case ExceptionNullPointer:
		return "NullPointer"
	case ExceptionIndexOutOfBounds:
		return "IndexOutOfBounds"
	case ExceptionDivisionByZero:
		return "DivisionByZero"
	case ExceptionStackOverflow:
		return "StackOverflow"
	case ExceptionOutOfMemory:
		return "OutOfMemory"
	case ExceptionInvalidOperation:
		return "InvalidOperation"
	case ExceptionTypeError:
		return "TypeError"
	case ExceptionParseError:
		return "ParseError"
	case ExceptionIOError:
		return "IOError"
	case ExceptionFileNotFound:
		return "FileNotFound"
	case ExceptionPermissionDenied:
		return "PermissionDenied"
	case ExceptionNetworkTimeout:
		return "NetworkTimeout"
	case ExceptionConnectionFailed:
		return "ConnectionFailed"
	case ExceptionDeadlock:
		return "Deadlock"
	case ExceptionRaceCondition:
		return "RaceCondition"
	case ExceptionThreadAbort:
		return "ThreadAbort"
	case ExceptionSynchronization:
		return "Synchronization"
	case ExceptionSystemError:
		return "SystemError"
	case ExceptionResourceExhausted:
		return "ResourceExhausted"
	case ExceptionSecurityViolation:
		return "SecurityViolation"
	case ExceptionConfigurationError:
		return "ConfigurationError"
	case ExceptionUserDefined:
		return "UserDefined"
	case ExceptionCustom:
		return "Custom"
	default:
		return fmt.Sprintf("Unknown(%d)", int(ek))
	}
}

// ExceptionSeverity represents the severity level of an exception.
type ExceptionSeverity int

const (
	ExceptionSeverityInfo ExceptionSeverity = iota
	ExceptionSeverityWarning
	ExceptionSeverityError
	ExceptionSeverityCritical
	ExceptionSeverityFatal
)

// String returns the string representation of an ExceptionSeverity.
func (es ExceptionSeverity) String() string {
	switch es {
	case ExceptionSeverityInfo:
		return "Info"
	case ExceptionSeverityWarning:
		return "Warning"
	case ExceptionSeverityError:
		return "Error"
	case ExceptionSeverityCritical:
		return "Critical"
	case ExceptionSeverityFatal:
		return "Fatal"
	default:
		return fmt.Sprintf("Unknown(%d)", int(es))
	}
}

// ExceptionRecovery represents recovery strategies for exceptions.
type ExceptionRecovery int

const (
	RecoveryNone ExceptionRecovery = iota
	RecoveryRetry
	RecoveryFallback
	RecoveryPropagate
	RecoveryTerminate
	RecoveryIgnore
	RecoveryLog
	RecoveryCustom
)

// String returns the string representation of an ExceptionRecovery.
func (er ExceptionRecovery) String() string {
	switch er {
	case RecoveryNone:
		return "None"
	case RecoveryRetry:
		return "Retry"
	case RecoveryFallback:
		return "Fallback"
	case RecoveryPropagate:
		return "Propagate"
	case RecoveryTerminate:
		return "Terminate"
	case RecoveryIgnore:
		return "Ignore"
	case RecoveryLog:
		return "Log"
	case RecoveryCustom:
		return "Custom"
	default:
		return fmt.Sprintf("Unknown(%d)", int(er))
	}
}

// ExceptionSpec represents an exception specification.
type ExceptionSpec struct {
	Location   *SourceLocation
	Metadata   map[string]interface{}
	Parent     *ExceptionSpec
	Message    string
	TypeName   string
	Conditions []string
	Children   []*ExceptionSpec
	Kind       ExceptionKind
	Severity   ExceptionSeverity
	Recovery   ExceptionRecovery
}

// NewExceptionSpec creates a new ExceptionSpec.
func NewExceptionSpec(kind ExceptionKind, severity ExceptionSeverity) *ExceptionSpec {
	return &ExceptionSpec{
		Kind:       kind,
		Severity:   severity,
		Recovery:   RecoveryPropagate,
		Conditions: make([]string, 0),
		Metadata:   make(map[string]interface{}),
		Children:   make([]*ExceptionSpec, 0),
	}
}

// String returns the string representation of an ExceptionSpec.
func (es *ExceptionSpec) String() string {
	return fmt.Sprintf("%s[%s]", es.Kind.String(), es.Severity.String())
}

// IsSubtypeOf checks if this exception is a subtype of another.
func (es *ExceptionSpec) IsSubtypeOf(other *ExceptionSpec) bool {
	if es.Kind == other.Kind {
		return true
	}

	// Check parent hierarchy.
	current := es.Parent
	for current != nil {
		if current.Kind == other.Kind {
			return true
		}

		current = current.Parent
	}

	return false
}

// AddChild adds a child exception.
func (es *ExceptionSpec) AddChild(child *ExceptionSpec) {
	child.Parent = es
	es.Children = append(es.Children, child)
}

// ExceptionSet represents a collection of exception specifications.
type ExceptionSet struct {
	exceptions map[ExceptionKind]*ExceptionSpec
	mu         sync.RWMutex
}

// NewExceptionSet creates a new empty ExceptionSet.
func NewExceptionSet() *ExceptionSet {
	return &ExceptionSet{
		exceptions: make(map[ExceptionKind]*ExceptionSpec),
	}
}

// Add adds an exception to the set.
func (es *ExceptionSet) Add(exception *ExceptionSpec) {
	es.mu.Lock()
	defer es.mu.Unlock()

	if existing, exists := es.exceptions[exception.Kind]; exists {
		// Keep the higher severity.
		if exception.Severity > existing.Severity {
			es.exceptions[exception.Kind] = exception
		}
	} else {
		es.exceptions[exception.Kind] = exception
	}
}

// Remove removes an exception from the set.
func (es *ExceptionSet) Remove(kind ExceptionKind) {
	es.mu.Lock()
	defer es.mu.Unlock()
	delete(es.exceptions, kind)
}

// Contains checks if the set contains an exception of the given kind.
func (es *ExceptionSet) Contains(kind ExceptionKind) bool {
	es.mu.RLock()
	defer es.mu.RUnlock()
	_, exists := es.exceptions[kind]

	return exists
}

// Get retrieves an exception by kind.
func (es *ExceptionSet) Get(kind ExceptionKind) (*ExceptionSpec, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	exception, exists := es.exceptions[kind]

	return exception, exists
}

// Size returns the number of exceptions in the set.
func (es *ExceptionSet) Size() int {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return len(es.exceptions)
}

// IsEmpty checks if the exception set is empty.
func (es *ExceptionSet) IsEmpty() bool {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return len(es.exceptions) == 0
}

// Union creates a new ExceptionSet containing exceptions from both sets.
func (es *ExceptionSet) Union(other *ExceptionSet) *ExceptionSet {
	result := NewExceptionSet()

	es.mu.RLock()
	for _, exception := range es.exceptions {
		result.Add(exception)
	}
	es.mu.RUnlock()

	other.mu.RLock()
	for _, exception := range other.exceptions {
		result.Add(exception)
	}
	other.mu.RUnlock()

	return result
}

// Intersection creates a new ExceptionSet containing exceptions common to both sets.
func (es *ExceptionSet) Intersection(other *ExceptionSet) *ExceptionSet {
	result := NewExceptionSet()

	es.mu.RLock()
	other.mu.RLock()
	defer es.mu.RUnlock()
	defer other.mu.RUnlock()

	for kind, exception := range es.exceptions {
		if _, exists := other.exceptions[kind]; exists {
			result.Add(exception)
		}
	}

	return result
}

// Subtract creates a new ExceptionSet with exceptions from this set not in other.
func (es *ExceptionSet) Subtract(other *ExceptionSet) *ExceptionSet {
	result := NewExceptionSet()

	es.mu.RLock()
	other.mu.RLock()
	defer es.mu.RUnlock()
	defer other.mu.RUnlock()

	for kind, exception := range es.exceptions {
		if _, exists := other.exceptions[kind]; !exists {
			result.Add(exception)
		}
	}

	return result
}

// ToSlice returns all exceptions as a slice.
func (es *ExceptionSet) ToSlice() []*ExceptionSpec {
	es.mu.RLock()
	defer es.mu.RUnlock()

	exceptions := make([]*ExceptionSpec, 0, len(es.exceptions))
	for _, exception := range es.exceptions {
		exceptions = append(exceptions, exception)
	}

	return exceptions
}

// String returns the string representation of the ExceptionSet.
func (es *ExceptionSet) String() string {
	if es.IsEmpty() {
		return "NoExceptions"
	}

	exceptions := es.ToSlice()

	var strs []string

	for _, exception := range exceptions {
		strs = append(strs, exception.String())
	}

	return "{" + strings.Join(strs, ", ") + "}"
}

// TryBlock represents a try block with exception handling.
type TryBlock struct {
	Body         ASTNode
	FinallyBlock *FinallyBlock
	Exceptions   *ExceptionSet
	Location     *SourceLocation
	CatchBlocks  []*CatchBlock
	Resources    []string
}

// NewTryBlock creates a new TryBlock.
func NewTryBlock(body ASTNode) *TryBlock {
	return &TryBlock{
		Body:        body,
		CatchBlocks: make([]*CatchBlock, 0),
		Exceptions:  NewExceptionSet(),
		Resources:   make([]string, 0),
	}
}

// AddCatchBlock adds a catch block.
func (tb *TryBlock) AddCatchBlock(catchBlock *CatchBlock) {
	tb.CatchBlocks = append(tb.CatchBlocks, catchBlock)
}

// SetFinallyBlock sets the finally block.
func (tb *TryBlock) SetFinallyBlock(finallyBlock *FinallyBlock) {
	tb.FinallyBlock = finallyBlock
}

// CatchBlock represents a catch block.
type CatchBlock struct {
	Body           ASTNode
	Guard          ASTNode
	Location       *SourceLocation
	Parameter      string
	ExceptionTypes []*ExceptionSpec
	Recovery       ExceptionRecovery
}

// NewCatchBlock creates a new CatchBlock.
func NewCatchBlock(exceptionTypes []*ExceptionSpec, parameter string, body ASTNode) *CatchBlock {
	return &CatchBlock{
		ExceptionTypes: exceptionTypes,
		Parameter:      parameter,
		Body:           body,
		Recovery:       RecoveryPropagate,
	}
}

// CanHandle checks if this catch block can handle the given exception.
func (cb *CatchBlock) CanHandle(exception *ExceptionSpec) bool {
	for _, exceptionType := range cb.ExceptionTypes {
		if exception.IsSubtypeOf(exceptionType) {
			return true
		}
	}

	return false
}

// FinallyBlock represents a finally block.
type FinallyBlock struct {
	Body       ASTNode
	Location   *SourceLocation
	Cleanup    []string
	Resources  []string
	Guaranteed bool
}

// NewFinallyBlock creates a new FinallyBlock.
func NewFinallyBlock(body ASTNode) *FinallyBlock {
	return &FinallyBlock{
		Body:       body,
		Cleanup:    make([]string, 0),
		Resources:  make([]string, 0),
		Guaranteed: true,
	}
}

// ThrowStatement represents a throw statement.
type ThrowStatement struct {
	Expression  ASTNode
	Condition   ASTNode
	Exception   *ExceptionSpec
	Location    *SourceLocation
	Conditional bool
}

// NewThrowStatement creates a new ThrowStatement.
func NewThrowStatement(exception *ExceptionSpec, expression ASTNode) *ThrowStatement {
	return &ThrowStatement{
		Exception:   exception,
		Expression:  expression,
		Conditional: false,
	}
}

// ExceptionSignature represents the exception signature of a function.
type ExceptionSignature struct {
	Throws       *ExceptionSet
	Catches      *ExceptionSet
	Propagates   *ExceptionSet
	FunctionName string
	Guarantees   []string
	Contracts    []*ExceptionContract
	Safety       ExceptionSafety
}

// NewExceptionSignature creates a new ExceptionSignature.
func NewExceptionSignature() *ExceptionSignature {
	return &ExceptionSignature{
		Throws:     NewExceptionSet(),
		Catches:    NewExceptionSet(),
		Propagates: NewExceptionSet(),
		Guarantees: make([]string, 0),
		Safety:     SafetyBasic,
		Contracts:  make([]*ExceptionContract, 0),
	}
}

// ExceptionSafety represents exception safety levels.
type ExceptionSafety int

const (
	SafetyNone ExceptionSafety = iota
	SafetyBasic
	SafetyStrong
	SafetyNoThrow
	SafetyNoFail
)

// String returns the string representation of ExceptionSafety.
func (es ExceptionSafety) String() string {
	switch es {
	case SafetyNone:
		return "None"
	case SafetyBasic:
		return "Basic"
	case SafetyStrong:
		return "Strong"
	case SafetyNoThrow:
		return "NoThrow"
	case SafetyNoFail:
		return "NoFail"
	default:
		return fmt.Sprintf("Unknown(%d)", int(es))
	}
}

// ExceptionContract represents exception contracts.
type ExceptionContract struct {
	ExceptionSpec  *ExceptionSpec
	Preconditions  []string
	Postconditions []string
	Invariants     []string
	Recovery       ExceptionRecovery
}

// NewExceptionContract creates a new ExceptionContract.
func NewExceptionContract(spec *ExceptionSpec) *ExceptionContract {
	return &ExceptionContract{
		Preconditions:  make([]string, 0),
		Postconditions: make([]string, 0),
		Invariants:     make([]string, 0),
		ExceptionSpec:  spec,
		Recovery:       RecoveryPropagate,
	}
}

// ExceptionAnalyzer provides exception analysis capabilities.
type ExceptionAnalyzer struct {
	flowAnalyzer  *ExceptionFlowAnalyzer
	safetyChecker *ExceptionSafetyChecker
	pathAnalyzer  *ExceptionPathAnalyzer
	statistics    *ExceptionStatistics
}

// NewExceptionAnalyzer creates a new exception analyzer.
func NewExceptionAnalyzer() *ExceptionAnalyzer {
	return &ExceptionAnalyzer{
		flowAnalyzer:  NewExceptionFlowAnalyzer(),
		safetyChecker: NewExceptionSafetyChecker(),
		pathAnalyzer:  NewExceptionPathAnalyzer(),
		statistics:    NewExceptionStatistics(),
	}
}

// AnalyzeExceptions analyzes exception behavior for the given AST node.
func (ea *ExceptionAnalyzer) AnalyzeExceptions(node ASTNode) (*ExceptionSignature, error) {
	ea.statistics.IncrementAnalysis()

	signature := NewExceptionSignature()

	// Perform flow analysis.
	flowResult, err := ea.flowAnalyzer.AnalyzeFlow(node)
	if err != nil {
		return nil, fmt.Errorf("exception flow analysis failed: %w", err)
	}

	signature.Throws = flowResult.ThrownExceptions
	signature.Catches = flowResult.CaughtExceptions

	// Perform safety analysis.
	safetyResult, err := ea.safetyChecker.CheckSafety(node, signature)
	if err != nil {
		return nil, fmt.Errorf("exception safety check failed: %w", err)
	}

	signature.Safety = safetyResult.SafetyLevel
	signature.Guarantees = safetyResult.Guarantees

	// Perform path analysis.
	pathResult, err := ea.pathAnalyzer.AnalyzePaths(node)
	if err != nil {
		return nil, fmt.Errorf("exception path analysis failed: %w", err)
	}

	signature.Propagates = pathResult.PropagatedExceptions

	return signature, nil
}

// ExceptionFlowAnalyzer analyzes exception flow.
type ExceptionFlowAnalyzer struct {
	callGraph    *ExceptionCallGraph
	dominators   map[string][]string
	reachability map[string]bool
}

// NewExceptionFlowAnalyzer creates a new exception flow analyzer.
func NewExceptionFlowAnalyzer() *ExceptionFlowAnalyzer {
	return &ExceptionFlowAnalyzer{
		callGraph:    NewExceptionCallGraph(),
		dominators:   make(map[string][]string),
		reachability: make(map[string]bool),
	}
}

// ExceptionFlowResult represents the result of exception flow analysis.
type ExceptionFlowResult struct {
	ThrownExceptions *ExceptionSet
	CaughtExceptions *ExceptionSet
	FlowGraph        *ExceptionFlowGraph
	UnhandledPaths   []string
}

// AnalyzeFlow analyzes exception flow for the given node.
func (efa *ExceptionFlowAnalyzer) AnalyzeFlow(node ASTNode) (*ExceptionFlowResult, error) {
	result := &ExceptionFlowResult{
		ThrownExceptions: NewExceptionSet(),
		CaughtExceptions: NewExceptionSet(),
		UnhandledPaths:   make([]string, 0),
		FlowGraph:        NewExceptionFlowGraph(),
	}

	// Implementation would traverse AST and build flow graph.
	return result, nil
}

// ExceptionSafetyChecker checks exception safety guarantees.
type ExceptionSafetyChecker struct {
	invariants map[string][]string
	contracts  []*ExceptionContract
	violations []string
}

// NewExceptionSafetyChecker creates a new exception safety checker.
func NewExceptionSafetyChecker() *ExceptionSafetyChecker {
	return &ExceptionSafetyChecker{
		invariants: make(map[string][]string),
		contracts:  make([]*ExceptionContract, 0),
		violations: make([]string, 0),
	}
}

// ExceptionSafetyResult represents the result of safety analysis.
type ExceptionSafetyResult struct {
	Guarantees      []string
	Violations      []string
	Recommendations []string
	SafetyLevel     ExceptionSafety
}

// CheckSafety checks exception safety for the given node and signature.
func (esc *ExceptionSafetyChecker) CheckSafety(node ASTNode, signature *ExceptionSignature) (*ExceptionSafetyResult, error) {
	result := &ExceptionSafetyResult{
		SafetyLevel:     SafetyBasic,
		Guarantees:      make([]string, 0),
		Violations:      make([]string, 0),
		Recommendations: make([]string, 0),
	}

	// Determine safety level based on exception handling.
	if signature.Throws.IsEmpty() {
		result.SafetyLevel = SafetyNoThrow
		result.Guarantees = append(result.Guarantees, "No exceptions thrown")
	} else if signature.Catches.Size() >= signature.Throws.Size() {
		result.SafetyLevel = SafetyStrong
		result.Guarantees = append(result.Guarantees, "All exceptions handled")
	}

	return result, nil
}

// ExceptionPathAnalyzer analyzes exception propagation paths.
type ExceptionPathAnalyzer struct {
	coverage   map[string]float64
	paths      []ExceptionPath
	complexity int
}

// NewExceptionPathAnalyzer creates a new exception path analyzer.
func NewExceptionPathAnalyzer() *ExceptionPathAnalyzer {
	return &ExceptionPathAnalyzer{
		paths:      make([]ExceptionPath, 0),
		coverage:   make(map[string]float64),
		complexity: 0,
	}
}

// ExceptionPathResult represents the result of path analysis.
type ExceptionPathResult struct {
	PropagatedExceptions *ExceptionSet
	CoverageReport       map[string]float64
	CriticalPaths        []ExceptionPath
	Recommendations      []string
}

// AnalyzePaths analyzes exception propagation paths.
func (epa *ExceptionPathAnalyzer) AnalyzePaths(node ASTNode) (*ExceptionPathResult, error) {
	result := &ExceptionPathResult{
		PropagatedExceptions: NewExceptionSet(),
		CoverageReport:       make(map[string]float64),
		CriticalPaths:        make([]ExceptionPath, 0),
		Recommendations:      make([]string, 0),
	}

	// Implementation would analyze exception propagation paths.
	return result, nil
}

// Supporting types.

// ExceptionCallGraph represents call graph with exception information.
type ExceptionCallGraph struct {
	nodes map[string]*ExceptionCallNode
	edges []ExceptionCallEdge
}

// NewExceptionCallGraph creates a new exception call graph.
func NewExceptionCallGraph() *ExceptionCallGraph {
	return &ExceptionCallGraph{
		nodes: make(map[string]*ExceptionCallNode),
		edges: make([]ExceptionCallEdge, 0),
	}
}

// ExceptionCallNode represents a node in the exception call graph.
type ExceptionCallNode struct {
	Exceptions *ExceptionSet
	Name       string
	Safety     ExceptionSafety
}

// ExceptionCallEdge represents an edge in the exception call graph.
type ExceptionCallEdge struct {
	Exceptions *ExceptionSet
	From       string
	To         string
}

// ExceptionFlowGraph represents exception flow graph.
type ExceptionFlowGraph struct {
	nodes map[string]*ExceptionFlowNode
	edges []ExceptionFlowEdge
}

// NewExceptionFlowGraph creates a new exception flow graph.
func NewExceptionFlowGraph() *ExceptionFlowGraph {
	return &ExceptionFlowGraph{
		nodes: make(map[string]*ExceptionFlowNode),
		edges: make([]ExceptionFlowEdge, 0),
	}
}

// ExceptionFlowNode represents a node in the exception flow graph.
type ExceptionFlowNode struct {
	Exceptions *ExceptionSet
	ID         string
	Type       string
	Handled    bool
}

// ExceptionFlowEdge represents an edge in the exception flow graph.
type ExceptionFlowEdge struct {
	Exceptions  *ExceptionSet
	From        string
	To          string
	Conditional bool
}

// ExceptionPath represents an exception propagation path.
type ExceptionPath struct {
	Exceptions  *ExceptionSet
	Nodes       []string
	Probability float64
	Handled     bool
	Critical    bool
}

// ExceptionStatistics tracks exception analysis statistics.
type ExceptionStatistics struct {
	TotalAnalyses      int64
	ExceptionsAnalyzed int64
	SafetyViolations   int64
	PathsCovered       int64
	mu                 sync.RWMutex
}

// NewExceptionStatistics creates new exception statistics.
func NewExceptionStatistics() *ExceptionStatistics {
	return &ExceptionStatistics{}
}

// IncrementAnalysis increments analysis count.
func (es *ExceptionStatistics) IncrementAnalysis() {
	es.mu.Lock()
	defer es.mu.Unlock()

	es.TotalAnalyses++
}

// ExceptionStatsSnapshot is a read-only snapshot of ExceptionStatistics without locks.
type ExceptionStatsSnapshot struct {
	TotalAnalyses      int64
	ExceptionsAnalyzed int64
	SafetyViolations   int64
	PathsCovered       int64
}

// GetStats returns a lock-free snapshot of current statistics.
func (es *ExceptionStatistics) GetStats() ExceptionStatsSnapshot {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return ExceptionStatsSnapshot{
		TotalAnalyses:      es.TotalAnalyses,
		ExceptionsAnalyzed: es.ExceptionsAnalyzed,
		SafetyViolations:   es.SafetyViolations,
		PathsCovered:       es.PathsCovered,
	}
}
