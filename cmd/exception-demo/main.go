// Package main demonstrates exception effect system capabilities in Orizon.
// This demo shows exception tracking, try-catch typing, and exception safety guarantees.
package main

import (
	"fmt"
)

// ExceptionKind represents different categories of exceptions
type ExceptionKind int

const (
	ExceptionNone ExceptionKind = iota
	ExceptionRuntime
	ExceptionNullPointer
	ExceptionIndexOutOfBounds
	ExceptionDivisionByZero
	ExceptionStackOverflow
	ExceptionOutOfMemory
	ExceptionIOError
	ExceptionFileNotFound
	ExceptionPermissionDenied
	ExceptionNetworkTimeout
	ExceptionDeadlock
	ExceptionSystemError
	ExceptionSecurityViolation
)

func (ek ExceptionKind) String() string {
	names := []string{
		"None", "Runtime", "NullPointer", "IndexOutOfBounds",
		"DivisionByZero", "StackOverflow", "OutOfMemory",
		"IOError", "FileNotFound", "PermissionDenied",
		"NetworkTimeout", "Deadlock", "SystemError", "SecurityViolation",
	}
	if int(ek) < len(names) {
		return names[ek]
	}
	return fmt.Sprintf("Unknown(%d)", int(ek))
}

// ExceptionSeverity represents the severity level of an exception
type ExceptionSeverity int

const (
	SeverityInfo ExceptionSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
	SeverityFatal
)

func (es ExceptionSeverity) String() string {
	names := []string{"Info", "Warning", "Error", "Critical", "Fatal"}
	if int(es) < len(names) {
		return names[es]
	}
	return fmt.Sprintf("Unknown(%d)", int(es))
}

// ExceptionRecovery represents recovery strategies
type ExceptionRecovery int

const (
	RecoveryNone ExceptionRecovery = iota
	RecoveryRetry
	RecoveryFallback
	RecoveryPropagate
	RecoveryTerminate
	RecoveryIgnore
	RecoveryLog
)

func (er ExceptionRecovery) String() string {
	names := []string{"None", "Retry", "Fallback", "Propagate", "Terminate", "Ignore", "Log"}
	if int(er) < len(names) {
		return names[er]
	}
	return fmt.Sprintf("Unknown(%d)", int(er))
}

// ExceptionSafety represents exception safety levels
type ExceptionSafety int

const (
	SafetyNone ExceptionSafety = iota
	SafetyBasic
	SafetyStrong
	SafetyNoThrow
	SafetyNoFail
)

func (es ExceptionSafety) String() string {
	names := []string{"None", "Basic", "Strong", "NoThrow", "NoFail"}
	if int(es) < len(names) {
		return names[es]
	}
	return fmt.Sprintf("Unknown(%d)", int(es))
}

// ExceptionSpec represents an exception specification
type ExceptionSpec struct {
	Kind     ExceptionKind
	Severity ExceptionSeverity
	Recovery ExceptionRecovery
	Message  string
	TypeName string
	Parent   *ExceptionSpec
	Children []*ExceptionSpec
}

func NewExceptionSpec(kind ExceptionKind, severity ExceptionSeverity) *ExceptionSpec {
	return &ExceptionSpec{
		Kind:     kind,
		Severity: severity,
		Recovery: RecoveryPropagate,
		Children: make([]*ExceptionSpec, 0),
	}
}

func (es *ExceptionSpec) String() string {
	return fmt.Sprintf("%s[%s]", es.Kind.String(), es.Severity.String())
}

func (es *ExceptionSpec) AddChild(child *ExceptionSpec) {
	child.Parent = es
	es.Children = append(es.Children, child)
}

func (es *ExceptionSpec) IsSubtypeOf(other *ExceptionSpec) bool {
	if es.Kind == other.Kind {
		return true
	}

	current := es.Parent
	for current != nil {
		if current.Kind == other.Kind {
			return true
		}
		current = current.Parent
	}

	return false
}

// ExceptionSet represents a collection of exception specifications
type ExceptionSet struct {
	exceptions map[ExceptionKind]*ExceptionSpec
}

func NewExceptionSet() *ExceptionSet {
	return &ExceptionSet{exceptions: make(map[ExceptionKind]*ExceptionSpec)}
}

func (es *ExceptionSet) Add(exception *ExceptionSpec) {
	es.exceptions[exception.Kind] = exception
}

func (es *ExceptionSet) Contains(kind ExceptionKind) bool {
	_, exists := es.exceptions[kind]
	return exists
}

func (es *ExceptionSet) Size() int {
	return len(es.exceptions)
}

func (es *ExceptionSet) IsEmpty() bool {
	return len(es.exceptions) == 0
}

func (es *ExceptionSet) ToSlice() []*ExceptionSpec {
	var exceptions []*ExceptionSpec
	for _, exception := range es.exceptions {
		exceptions = append(exceptions, exception)
	}
	return exceptions
}

func (es *ExceptionSet) String() string {
	if es.IsEmpty() {
		return "NoExceptions"
	}
	var specs []string
	for _, spec := range es.exceptions {
		specs = append(specs, spec.String())
	}
	return fmt.Sprintf("{%v}", specs)
}

func (es *ExceptionSet) Union(other *ExceptionSet) *ExceptionSet {
	result := NewExceptionSet()
	for _, exception := range es.exceptions {
		result.Add(exception)
	}
	for _, exception := range other.exceptions {
		result.Add(exception)
	}
	return result
}

// TryBlock represents a try-catch-finally construct
type TryBlock struct {
	Name         string
	Throws       *ExceptionSet
	Catches      *ExceptionSet
	Propagates   *ExceptionSet
	Safety       ExceptionSafety
	FinallyBlock bool
	Resources    []string
}

func NewTryBlock(name string) *TryBlock {
	return &TryBlock{
		Name:       name,
		Throws:     NewExceptionSet(),
		Catches:    NewExceptionSet(),
		Propagates: NewExceptionSet(),
		Safety:     SafetyBasic,
		Resources:  make([]string, 0),
	}
}

func (tb *TryBlock) AddThrows(spec *ExceptionSpec) {
	tb.Throws.Add(spec)
}

func (tb *TryBlock) AddCatches(spec *ExceptionSpec) {
	tb.Catches.Add(spec)
}

func (tb *TryBlock) String() string {
	return fmt.Sprintf("TryBlock[%s]: throws=%s, catches=%s, safety=%s",
		tb.Name, tb.Throws.String(), tb.Catches.String(), tb.Safety.String())
}

// CatchBlock represents a catch clause
type CatchBlock struct {
	ExceptionTypes []*ExceptionSpec
	Parameter      string
	Recovery       ExceptionRecovery
}

func NewCatchBlock(exceptionTypes []*ExceptionSpec, parameter string) *CatchBlock {
	return &CatchBlock{
		ExceptionTypes: exceptionTypes,
		Parameter:      parameter,
		Recovery:       RecoveryPropagate,
	}
}

func (cb *CatchBlock) CanHandle(exception *ExceptionSpec) bool {
	for _, exceptionType := range cb.ExceptionTypes {
		if exception.IsSubtypeOf(exceptionType) {
			return true
		}
	}
	return false
}

func (cb *CatchBlock) String() string {
	var types []string
	for _, t := range cb.ExceptionTypes {
		types = append(types, t.String())
	}
	return fmt.Sprintf("catch(%s %s)", cb.Parameter, types)
}

// ExceptionSignature represents the complete exception signature of a function
type ExceptionSignature struct {
	FunctionName string
	Throws       *ExceptionSet
	Catches      *ExceptionSet
	Propagates   *ExceptionSet
	Safety       ExceptionSafety
	Guarantees   []string
}

func NewExceptionSignature(name string) *ExceptionSignature {
	return &ExceptionSignature{
		FunctionName: name,
		Throws:       NewExceptionSet(),
		Catches:      NewExceptionSet(),
		Propagates:   NewExceptionSet(),
		Safety:       SafetyBasic,
		Guarantees:   make([]string, 0),
	}
}

func (es *ExceptionSignature) String() string {
	return fmt.Sprintf("%s: throws=%s, safety=%s",
		es.FunctionName, es.Throws.String(), es.Safety.String())
}

func main() {
	fmt.Println("🎭 Orizon Exception Effect System Demo")
	fmt.Println("======================================")

	// Demo 1: Exception Hierarchy and Specifications
	fmt.Println("\n📍 Demo 1: Exception Hierarchy")

	// Create exception hierarchy
	runtime := NewExceptionSpec(ExceptionRuntime, SeverityError)
	nullPointer := NewExceptionSpec(ExceptionNullPointer, SeverityError)
	indexBounds := NewExceptionSpec(ExceptionIndexOutOfBounds, SeverityError)
	divisionByZero := NewExceptionSpec(ExceptionDivisionByZero, SeverityCritical)

	runtime.AddChild(nullPointer)
	runtime.AddChild(indexBounds)
	runtime.AddChild(divisionByZero)

	fmt.Printf("Exception Hierarchy:\n")
	fmt.Printf("  %s\n", runtime)
	fmt.Printf("    ├─ %s\n", nullPointer)
	fmt.Printf("    ├─ %s\n", indexBounds)
	fmt.Printf("    └─ %s\n", divisionByZero)

	// Test subtype relationships
	fmt.Printf("\nSubtype Tests:\n")
	fmt.Printf("  NullPointer is subtype of Runtime: %v\n", nullPointer.IsSubtypeOf(runtime))
	fmt.Printf("  Runtime is subtype of NullPointer: %v\n", runtime.IsSubtypeOf(nullPointer))

	// Demo 2: Exception Sets and Operations
	fmt.Println("\n📍 Demo 2: Exception Sets")

	set1 := NewExceptionSet()
	set1.Add(nullPointer)
	set1.Add(indexBounds)

	set2 := NewExceptionSet()
	set2.Add(divisionByZero)
	set2.Add(NewExceptionSpec(ExceptionIOError, SeverityError))

	fmt.Printf("Set 1: %s\n", set1)
	fmt.Printf("Set 2: %s\n", set2)

	union := set1.Union(set2)
	fmt.Printf("Union: %s\n", union)

	// Demo 3: Try-Catch-Finally Blocks
	fmt.Println("\n📍 Demo 3: Try-Catch-Finally Blocks")

	// Create try blocks for different scenarios
	fileOperations := NewTryBlock("FileOperations")
	fileOperations.AddThrows(NewExceptionSpec(ExceptionFileNotFound, SeverityError))
	fileOperations.AddThrows(NewExceptionSpec(ExceptionPermissionDenied, SeverityError))
	fileOperations.AddThrows(NewExceptionSpec(ExceptionIOError, SeverityCritical))
	fileOperations.AddCatches(NewExceptionSpec(ExceptionIOError, SeverityError))
	fileOperations.FinallyBlock = true
	fileOperations.Resources = []string{"file_handle", "buffer"}

	networkOperations := NewTryBlock("NetworkOperations")
	networkOperations.AddThrows(NewExceptionSpec(ExceptionNetworkTimeout, SeverityError))
	networkOperations.AddThrows(NewExceptionSpec(ExceptionSystemError, SeverityCritical))
	networkOperations.AddCatches(NewExceptionSpec(ExceptionNetworkTimeout, SeverityError))

	fmt.Printf("%s\n", fileOperations)
	fmt.Printf("%s\n", networkOperations)

	// Demo 4: Catch Block Handling
	fmt.Println("\n📍 Demo 4: Catch Block Handling")

	// Create catch blocks
	runtimeCatch := NewCatchBlock([]*ExceptionSpec{runtime}, "e")
	runtimeCatch.Recovery = RecoveryLog

	ioCatch := NewCatchBlock([]*ExceptionSpec{NewExceptionSpec(ExceptionIOError, SeverityError)}, "ioE")
	ioCatch.Recovery = RecoveryRetry

	fmt.Printf("Catch blocks:\n")
	fmt.Printf("  %s (recovery: %s)\n", runtimeCatch, runtimeCatch.Recovery)
	fmt.Printf("  %s (recovery: %s)\n", ioCatch, ioCatch.Recovery)

	// Test exception handling
	fmt.Printf("\nException Handling Tests:\n")
	fmt.Printf("  Runtime catch can handle NullPointer: %v\n", runtimeCatch.CanHandle(nullPointer))
	fmt.Printf("  Runtime catch can handle IOError: %v\n", runtimeCatch.CanHandle(NewExceptionSpec(ExceptionIOError, SeverityError)))
	fmt.Printf("  IO catch can handle FileNotFound: %v\n", ioCatch.CanHandle(NewExceptionSpec(ExceptionFileNotFound, SeverityError)))

	// Demo 5: Exception Safety Levels
	fmt.Println("\n📍 Demo 5: Exception Safety Levels")

	functions := []struct {
		name    string
		throws  []*ExceptionSpec
		catches []*ExceptionSpec
		safety  ExceptionSafety
		desc    string
	}{
		{
			name:   "PureMathFunction",
			throws: []*ExceptionSpec{},
			safety: SafetyNoThrow,
			desc:   "Mathematical computation with no exceptions",
		},
		{
			name:    "SafeFileReader",
			throws:  []*ExceptionSpec{NewExceptionSpec(ExceptionFileNotFound, SeverityError)},
			catches: []*ExceptionSpec{NewExceptionSpec(ExceptionFileNotFound, SeverityError)},
			safety:  SafetyStrong,
			desc:    "File reader with complete exception handling",
		},
		{
			name: "DatabaseConnection",
			throws: []*ExceptionSpec{
				NewExceptionSpec(ExceptionNetworkTimeout, SeverityError),
				NewExceptionSpec(ExceptionSystemError, SeverityCritical),
			},
			catches: []*ExceptionSpec{NewExceptionSpec(ExceptionNetworkTimeout, SeverityError)},
			safety:  SafetyBasic,
			desc:    "Database connection with partial exception handling",
		},
		{
			name:   "CriticalSystemCall",
			throws: []*ExceptionSpec{NewExceptionSpec(ExceptionSystemError, SeverityFatal)},
			safety: SafetyNone,
			desc:   "System call that may fail catastrophically",
		},
	}

	for _, fn := range functions {
		signature := NewExceptionSignature(fn.name)
		for _, spec := range fn.throws {
			signature.Throws.Add(spec)
		}
		for _, spec := range fn.catches {
			signature.Catches.Add(spec)
		}
		signature.Safety = fn.safety

		fmt.Printf("\nFunction: %s\n", fn.name)
		fmt.Printf("  Description: %s\n", fn.desc)
		fmt.Printf("  Signature: %s\n", signature)
		fmt.Printf("  Safety Level: %s\n", fn.safety)

		// Analyze safety
		if signature.Throws.IsEmpty() {
			fmt.Printf("  ✅ No exceptions thrown - highest safety\n")
		} else if signature.Catches.Size() >= signature.Throws.Size() {
			fmt.Printf("  ✅ All exceptions handled - strong safety\n")
		} else if signature.Catches.Size() > 0 {
			fmt.Printf("  ⚠️  Partial exception handling - basic safety\n")
		} else {
			fmt.Printf("  ❌ No exception handling - potential danger\n")
		}
	}

	// Demo 6: Exception Propagation Analysis
	fmt.Println("\n📍 Demo 6: Exception Propagation Analysis")

	callChain := []struct {
		name    string
		throws  *ExceptionSet
		catches *ExceptionSet
	}{
		{
			name: "lowLevelFunction",
			throws: func() *ExceptionSet {
				set := NewExceptionSet()
				set.Add(NewExceptionSpec(ExceptionFileNotFound, SeverityError))
				set.Add(NewExceptionSpec(ExceptionPermissionDenied, SeverityError))
				return set
			}(),
			catches: NewExceptionSet(),
		},
		{
			name: "middleLevelFunction",
			throws: func() *ExceptionSet {
				set := NewExceptionSet()
				set.Add(NewExceptionSpec(ExceptionPermissionDenied, SeverityError))
				set.Add(NewExceptionSpec(ExceptionIOError, SeverityError))
				return set
			}(),
			catches: func() *ExceptionSet {
				set := NewExceptionSet()
				set.Add(NewExceptionSpec(ExceptionFileNotFound, SeverityError))
				return set
			}(),
		},
		{
			name:   "topLevelFunction",
			throws: NewExceptionSet(),
			catches: func() *ExceptionSet {
				set := NewExceptionSet()
				set.Add(NewExceptionSpec(ExceptionIOError, SeverityError))
				set.Add(NewExceptionSpec(ExceptionPermissionDenied, SeverityError))
				return set
			}(),
		},
	}

	fmt.Printf("Exception Propagation Chain:\n")
	propagated := NewExceptionSet()

	for i, fn := range callChain {
		fmt.Printf("  %d. %s\n", i+1, fn.name)
		fmt.Printf("     Throws: %s\n", fn.throws)
		fmt.Printf("     Catches: %s\n", fn.catches)

		// Calculate propagated exceptions
		combined := propagated.Union(fn.throws)
		for _, caught := range fn.catches.ToSlice() {
			if combined.Contains(caught.Kind) {
				delete(combined.exceptions, caught.Kind)
			}
		}
		propagated = combined

		fmt.Printf("     Propagates: %s\n", propagated)
	}

	// Demo 7: Real-world Exception Scenarios
	fmt.Println("\n📍 Demo 7: Real-world Scenarios")

	scenarios := []struct {
		name        string
		description string
		exceptions  *ExceptionSet
		safety      ExceptionSafety
		recovery    ExceptionRecovery
	}{
		{
			name:        "WebServerRequest",
			description: "HTTP request handler with comprehensive error handling",
			exceptions: func() *ExceptionSet {
				set := NewExceptionSet()
				set.Add(NewExceptionSpec(ExceptionNetworkTimeout, SeverityWarning))
				set.Add(NewExceptionSpec(ExceptionPermissionDenied, SeverityError))
				return set
			}(),
			safety:   SafetyStrong,
			recovery: RecoveryFallback,
		},
		{
			name:        "DatabaseTransaction",
			description: "ACID transaction with rollback capability",
			exceptions: func() *ExceptionSet {
				set := NewExceptionSet()
				set.Add(NewExceptionSpec(ExceptionDeadlock, SeverityError))
				set.Add(NewExceptionSpec(ExceptionSystemError, SeverityCritical))
				return set
			}(),
			safety:   SafetyStrong,
			recovery: RecoveryRetry,
		},
		{
			name:        "FileProcessor",
			description: "Batch file processor with error logging",
			exceptions: func() *ExceptionSet {
				set := NewExceptionSet()
				set.Add(NewExceptionSpec(ExceptionFileNotFound, SeverityWarning))
				set.Add(NewExceptionSpec(ExceptionOutOfMemory, SeverityFatal))
				return set
			}(),
			safety:   SafetyBasic,
			recovery: RecoveryLog,
		},
		{
			name:        "RealTimeSystem",
			description: "Real-time system with no exception tolerance",
			exceptions:  NewExceptionSet(),
			safety:      SafetyNoFail,
			recovery:    RecoveryTerminate,
		},
	}

	for _, scenario := range scenarios {
		fmt.Printf("\nScenario: %s\n", scenario.name)
		fmt.Printf("  Description: %s\n", scenario.description)
		fmt.Printf("  Exceptions: %s\n", scenario.exceptions)
		fmt.Printf("  Safety: %s\n", scenario.safety)
		fmt.Printf("  Recovery: %s\n", scenario.recovery)

		// Risk assessment
		if scenario.exceptions.IsEmpty() {
			fmt.Printf("  🟢 Risk Level: Very Low (No exceptions)\n")
		} else {
			riskLevel := "Low"
			hasHighSeverity := false
			for _, spec := range scenario.exceptions.ToSlice() {
				if spec.Severity >= SeverityCritical {
					hasHighSeverity = true
					break
				}
			}

			if hasHighSeverity {
				riskLevel = "High"
				fmt.Printf("  🔴 Risk Level: %s (Critical exceptions present)\n", riskLevel)
			} else {
				fmt.Printf("  🟡 Risk Level: %s (Manageable exceptions)\n", riskLevel)
			}
		}
	}

	fmt.Println("\n🎉 Exception Effect System Demo Completed!")
	fmt.Println("==========================================")
	fmt.Println("The exception system successfully demonstrates:")
	fmt.Println("✅ Type-level exception tracking and classification")
	fmt.Println("✅ Exception hierarchy and subtype relationships")
	fmt.Println("✅ Try-catch-finally block typing and analysis")
	fmt.Println("✅ Exception safety level guarantees")
	fmt.Println("✅ Exception propagation and flow analysis")
	fmt.Println("✅ Real-world exception handling scenarios")
	fmt.Println("✅ Comprehensive exception effect management")
}
