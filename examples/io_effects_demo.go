// I/O Effects System Demonstration
// This program showcases the comprehensive I/O effect tracking system
// with static analysis, pure function guarantees, and I/O monad usage.

package main

import (
	"fmt"
	"time"

	"github.com/orizon-lang/orizon/internal/types"
)

// Example I/O operations with effect tracking
func main() {
	fmt.Println("=== Orizon I/O Effects System Demonstration ===")
	fmt.Println()

	// Demonstrate I/O effect creation and analysis
	demonstrateEffectCreation()
	fmt.Println()

	// Demonstrate I/O effect sets and operations
	demonstrateEffectSets()
	fmt.Println()

	// Demonstrate I/O constraints and security
	demonstrateConstraints()
	fmt.Println()

	// Demonstrate integration effects.
	integrationDemo()
	fmt.Println()

	// Demonstrate I/O signatures and function analysis
	demonstrateSignatures()
	fmt.Println()

	// Demonstrate I/O monad operations
	demonstrateIOMonad()
	fmt.Println()

	// Demonstrate I/O inference engine
	demonstrateInferenceEngine()
	fmt.Println()

	// Demonstrate purity checking.
	demonstratePurityChecker()
	fmt.Println()

	// Demonstrate real-world use cases.
	demonstrateRealWorldUsage()
}

func demonstrateEffectCreation() {
	fmt.Println("--- I/O Effect Creation and Analysis ---")

	// Create various I/O effects
	fileRead := types.NewIOEffect(types.IOEffectFileRead, types.IOPermissionRead)
	fileRead.Resource = "/etc/passwd"
	fileRead.Description = "Reading system password file"

	fileWrite := types.NewIOEffect(types.IOEffectFileWrite, types.IOPermissionWrite)
	fileWrite.Resource = "/tmp/output.txt"
	fileWrite.Description = "Writing temporary output"
	fileWrite.AddBehavior(types.IOBehaviorSideEffecting)

	networkReq := types.NewIOEffect(types.IOEffectHTTPRequest, types.IOPermissionReadWrite)
	networkReq.Resource = "https://api.example.com/data"
	networkReq.Description = "HTTP API request"
	networkReq.AddBehavior(types.IOBehaviorNonDeterministic)
	networkReq.AddBehavior(types.IOBehaviorBlocking)

	dbQuery := types.NewIOEffect(types.IOEffectDatabaseQuery, types.IOPermissionRead)
	dbQuery.Resource = "users_table"
	dbQuery.Description = "Query user information"
	dbQuery.AddBehavior(types.IOBehaviorDeterministic)
	dbQuery.Metadata["timeout"] = 5000

	// Analyze effect properties.
	effects := []*types.IOEffect{fileRead, fileWrite, networkReq, dbQuery}

	for i, effect := range effects {
		fmt.Printf("Effect %d: %s\n", i+1, effect.Kind.String())
		fmt.Printf("  Resource: %s\n", effect.Resource)
		fmt.Printf("  Permission: %s\n", effect.Permission.String())
		fmt.Printf("  Level: %s\n", effect.Level.String())
		fmt.Printf("  Pure: %v\n", effect.IsPure())
		fmt.Printf("  Read-only: %v\n", effect.IsReadOnly())
		fmt.Printf("  Write access: %v\n", effect.IsWriteAccess())
		fmt.Printf("  Behaviors: %v\n", getBehaviorStrings(effect.Behaviors))
		if len(effect.Metadata) > 0 {
			fmt.Printf("  Metadata: %v\n", effect.Metadata)
		}
		fmt.Println()
	}
}

func demonstrateEffectSets() {
	fmt.Println("--- I/O Effect Sets and Operations ---")

	// Create effect sets.
	fileOps := types.NewIOEffectSet()
	fileOps.Add(types.NewIOEffect(types.IOEffectFileRead, types.IOPermissionRead))
	fileOps.Add(types.NewIOEffect(types.IOEffectFileWrite, types.IOPermissionWrite))
	fileOps.Add(types.NewIOEffect(types.IOEffectFileCreate, types.IOPermissionWrite))

	networkOps := types.NewIOEffectSet()
	networkOps.Add(types.NewIOEffect(types.IOEffectNetworkConnect, types.IOPermissionReadWrite))
	networkOps.Add(types.NewIOEffect(types.IOEffectHTTPRequest, types.IOPermissionReadWrite))
	networkOps.Add(types.NewIOEffect(types.IOEffectFileWrite, types.IOPermissionWrite)) // Common effect

	fmt.Printf("File operations set size: %d\n", fileOps.Size())
	fmt.Printf("File operations pure: %v\n", fileOps.IsPure())
	fmt.Printf("Network operations set size: %d\n", networkOps.Size())
	fmt.Printf("Network operations pure: %v\n", networkOps.IsPure())

	// Set operations.
	union := fileOps.Union(networkOps)
	intersection := fileOps.Intersection(networkOps)
	difference := fileOps.Difference(networkOps)

	fmt.Printf("Union size: %d\n", union.Size())
	fmt.Printf("Intersection size: %d\n", intersection.Size())
	fmt.Printf("Difference size: %d\n", difference.Size())

	// Analyze combined effects.
	fmt.Printf("Combined effects pure: %v\n", union.IsPure())
	fmt.Printf("Combined effects kinds: %v\n", getEffectKinds(union))
}

func demonstrateConstraints() {
	fmt.Println("--- I/O Constraints and Security ---")

	// Create permission constraint.
	readOnlyConstraint := types.NewIOPermissionConstraint(types.IOPermissionRead)
	fmt.Println("Created read-only permission constraint")

	// Create resource constraint.
	resourceConstraint := types.NewIOResourceConstraint()
	resourceConstraint.AllowResource("/tmp/")
	resourceConstraint.AllowResource("/var/log/")
	resourceConstraint.DenyResource("/etc/")
	resourceConstraint.DenyResource("/root/")
	fmt.Println("Created resource constraint (allow: /tmp/, /var/log/; deny: /etc/, /root/)")

	// Test effects against constraints.
	testEffects := []*types.IOEffect{
		createTestEffect(types.IOEffectFileRead, types.IOPermissionRead, "/tmp/data.txt"),
		createTestEffect(types.IOEffectFileWrite, types.IOPermissionWrite, "/tmp/output.txt"),
		createTestEffect(types.IOEffectFileRead, types.IOPermissionRead, "/etc/passwd"),
		createTestEffect(types.IOEffectFileWrite, types.IOPermissionWrite, "/var/log/app.log"),
		createTestEffect(types.IOEffectFileDelete, types.IOPermissionFullAccess, "/root/secret.txt"),
	}

	fmt.Println("\nConstraint checking results:")
	for i, effect := range testEffects {
		fmt.Printf("Effect %d (%s on %s):\n", i+1, effect.Kind.String(), effect.Resource)

		permResult := readOnlyConstraint.Check(effect)
		fmt.Printf("  Permission constraint: %v\n", permResult)

		resourceResult := resourceConstraint.Check(effect)
		fmt.Printf("  Resource constraint: %v\n", resourceResult)

		overall := permResult && resourceResult
		fmt.Printf("  Overall allowed: %v\n", overall)
		fmt.Println()
	}
}

func demonstrateSignatures() {
	fmt.Println("--- I/O Signatures and Function Analysis ---")

	// Create function signatures with different I/O patterns

	// Pure function.
	mathFunc := types.NewIOSignature("calculateSum")
	mathFunc.AddEffect(types.NewIOEffect(types.IOEffectPure, types.IOPermissionNone))

	// File processing function.
	fileProcessor := types.NewIOSignature("processLogFile")
	fileProcessor.AddEffect(types.NewIOEffect(types.IOEffectFileRead, types.IOPermissionRead))
	fileProcessor.AddEffect(types.NewIOEffect(types.IOEffectFileWrite, types.IOPermissionWrite))
	fileProcessor.AddConstraint(types.NewIOPermissionConstraint(types.IOPermissionRead, types.IOPermissionWrite))

	// Network service function.
	apiService := types.NewIOSignature("fetchUserData")
	httpEffect := types.NewIOEffect(types.IOEffectHTTPRequest, types.IOPermissionReadWrite)
	httpEffect.AddBehavior(types.IOBehaviorNonDeterministic)
	httpEffect.AddBehavior(types.IOBehaviorBlocking)
	apiService.AddEffect(httpEffect)
	apiService.AddEffect(types.NewIOEffect(types.IOEffectDatabaseQuery, types.IOPermissionRead))

	// Database transaction function.
	dbTransaction := types.NewIOSignature("updateUserProfile")
	dbTransaction.AddEffect(types.NewIOEffect(types.IOEffectDatabaseQuery, types.IOPermissionRead))
	dbTransaction.AddEffect(types.NewIOEffect(types.IOEffectDatabaseUpdate, types.IOPermissionWrite))
	dbTransaction.AddEffect(types.NewIOEffect(types.IOEffectDatabaseCommit, types.IOPermissionWrite))

	signatures := []*types.IOSignature{mathFunc, fileProcessor, apiService, dbTransaction}

	for i, sig := range signatures {
		fmt.Printf("Function %d: %s\n", i+1, sig.FunctionName)
		fmt.Printf("  Pure: %v\n", sig.Pure)
		fmt.Printf("  Deterministic: %v\n", sig.Deterministic)
		fmt.Printf("  Idempotent: %v\n", sig.Idempotent)
		fmt.Printf("  Effects count: %d\n", sig.Effects.Size())
		fmt.Printf("  Constraints count: %d\n", len(sig.Constraints))

		if !sig.Effects.IsEmpty() {
			fmt.Printf("  Effect kinds: %v\n", getEffectKinds(sig.Effects))
		}

		fmt.Println()
	}
}

func demonstrateIOMonad() {
	fmt.Println("--- I/O Monad Operations ---")

	// Pure computation.
	fmt.Println("1. Pure computation:")
	pure := types.PureIO(42)
	result, err := pure.Run()
	fmt.Printf("   Result: %v, Error: %v\n", result, err)

	// Map operation.
	fmt.Println("2. Map operation (multiply by 2):")
	mapped := pure.Map(func(x interface{}) interface{} {
		return x.(int) * 2
	})
	result, err = mapped.Run()
	fmt.Printf("   Result: %v, Error: %v\n", result, err)

	// Bind operation.
	fmt.Println("3. Bind operation (add 10 and return new IO):")
	bound := pure.Bind(func(x interface{}) *types.IOMonad {
		return types.PureIO(x.(int) + 10)
	})
	result, err = bound.Run()
	fmt.Printf("   Result: %v, Error: %v\n", result, err)

	// Sequence operations.
	fmt.Println("4. Sequence operations:")
	monads := []*types.IOMonad{
		types.PureIO("first"),
		types.PureIO("second"),
		types.PureIO("third"),
	}
	sequence := types.Sequence(monads)
	result, err = sequence.Run()
	fmt.Printf("   Result: %v, Error: %v\n", result, err)

	// Parallel operations.
	fmt.Println("5. Parallel operations (with timing):")
	start := time.Now()
	slowMonads := []*types.IOMonad{
		types.NewIOMonad(func() (interface{}, error) {
			time.Sleep(100 * time.Millisecond)
			return "slow1", nil
		}),
		types.NewIOMonad(func() (interface{}, error) {
			time.Sleep(100 * time.Millisecond)
			return "slow2", nil
		}),
	}
	parallel := types.Parallel(slowMonads)
	result, err = parallel.Run()
	duration := time.Since(start)
	fmt.Printf("   Result: %v, Error: %v, Duration: %v\n", result, err, duration)
}

func demonstrateInferenceEngine() {
	fmt.Println("--- I/O Inference Engine ---")

	context := types.NewIOContext()
	engine := types.NewIOInferenceEngine(context)

	// Register some function signatures.
	printlnSig := types.NewIOSignature("println")
	printlnSig.AddEffect(types.NewIOEffect(types.IOEffectStdoutWrite, types.IOPermissionWrite))
	engine.RegisterFunction("println", printlnSig)

	fileReadSig := types.NewIOSignature("readFile")
	fileReadSig.AddEffect(types.NewIOEffect(types.IOEffectFileRead, types.IOPermissionRead))
	engine.RegisterFunction("readFile", fileReadSig)

	httpGetSig := types.NewIOSignature("httpGet")
	httpEffect := types.NewIOEffect(types.IOEffectHTTPRequest, types.IOPermissionReadWrite)
	httpEffect.AddBehavior(types.IOBehaviorNonDeterministic)
	httpGetSig.AddEffect(httpEffect)
	engine.RegisterFunction("httpGet", httpGetSig)

	fmt.Printf("Registered function signatures for demonstration\n")

	// Test inference on mock expressions.
	testCalls := []string{"println", "readFile", "httpGet", "unknownFunction"}

	for _, funcName := range testCalls {
		callExpr := &types.CallExpr{
			Function: &types.FunctionDecl{Name: funcName},
		}

		effects, err := engine.InferEffects(callExpr)
		if err != nil {
			fmt.Printf("Function '%s': Error - %v\n", funcName, err)
		} else {
			fmt.Printf("Function '%s': %d effects inferred\n", funcName, effects.Size())
			if !effects.IsEmpty() {
				fmt.Printf("  Effect kinds: %v\n", getEffectKinds(effects))
			}
		}
	}
}

func demonstratePurityChecker() {
	fmt.Println("--- Purity Checker ---")

	checker := types.NewIOPurityChecker(true)

	// Test pure function.
	pureFunc := types.NewIOSignature("mathOperation")
	pureFunc.AddEffect(types.NewIOEffect(types.IOEffectPure, types.IOPermissionNone))

	fmt.Println("1. Testing pure function:")
	violations := checker.CheckPurity(pureFunc)
	fmt.Printf("   Violations: %d\n", len(violations))
	err := checker.EnforcePurity(pureFunc)
	fmt.Printf("   Enforcement: %v\n", err == nil)

	// Test impure function.
	impureFunc := types.NewIOSignature("printMessage")
	impureFunc.AddEffect(types.NewIOEffect(types.IOEffectStdoutWrite, types.IOPermissionWrite))

	fmt.Println("2. Testing impure function:")
	violations = checker.CheckPurity(impureFunc)
	fmt.Printf("   Violations: %d\n", len(violations))
	err = checker.EnforcePurity(impureFunc)
	fmt.Printf("   Enforcement: %v\n", err == nil)

	// Whitelist the impure function.
	fmt.Println("3. Testing whitelisted impure function:")
	checker.AllowFunction("printMessage")
	violations = checker.CheckPurity(impureFunc)
	fmt.Printf("   Violations after whitelisting: %d\n", len(violations))

	// Test with strict mode disabled.
	relaxedChecker := types.NewIOPurityChecker(false)
	fmt.Println("4. Testing with relaxed purity checking:")
	err = relaxedChecker.EnforcePurity(impureFunc)
	fmt.Printf("   Relaxed enforcement: %v\n", err == nil)
}

func demonstrateRealWorldUsage() {
	fmt.Println("--- Real-World Usage Examples ---")

	// Example 1: File processing pipeline.
	fmt.Println("1. File Processing Pipeline Analysis:")
	pipeline := analyzeFileProcessingPipeline()
	fmt.Printf("   Total effects: %d\n", pipeline.Size())
	fmt.Printf("   Is pure: %v\n", pipeline.IsPure())
	fmt.Printf("   Effect kinds: %v\n", getEffectKinds(pipeline))

	// Example 2: Web API handler.
	fmt.Println("2. Web API Handler Analysis:")
	apiHandler := analyzeWebAPIHandler()
	fmt.Printf("   Total effects: %d\n", apiHandler.Size())
	fmt.Printf("   Is pure: %v\n", apiHandler.IsPure())
	fmt.Printf("   Effect kinds: %v\n", getEffectKinds(apiHandler))

	// Example 3: Database migration.
	fmt.Println("3. Database Migration Analysis:")
	migration := analyzeDatabaseMigration()
	fmt.Printf("   Total effects: %d\n", migration.Size())
	fmt.Printf("   Is pure: %v\n", migration.IsPure())
	fmt.Printf("   Effect kinds: %v\n", getEffectKinds(migration))

	// Example 4: Security audit.
	fmt.Println("4. Security Audit Example:")
	performSecurityAudit()
}

// Helper functions.

func getBehaviorStrings(behaviors []types.IOEffectBehavior) []string {
	result := make([]string, len(behaviors))
	for i, behavior := range behaviors {
		result[i] = behavior.String()
	}
	return result
}

func getEffectKinds(effectSet *types.IOEffectSet) []string {
	effects := effectSet.ToSlice()
	kinds := make([]string, len(effects))
	for i, effect := range effects {
		kinds[i] = effect.Kind.String()
	}
	return kinds
}

func createTestEffect(kind types.IOEffectKind, permission types.IOEffectPermission, resource string) *types.IOEffect {
	effect := types.NewIOEffect(kind, permission)
	effect.Resource = resource
	return effect
}

func analyzeFileProcessingPipeline() *types.IOEffectSet {
	// Simulates analysis of a file processing pipeline:.
	// 1. Read input files
	// 2. Process data (pure computation)
	// 3. Write results
	// 4. Update metadata
	// 5. Log completion

	effects := types.NewIOEffectSet()

	// File reading.
	effects.Add(types.NewIOEffect(types.IOEffectFileRead, types.IOPermissionRead))
	effects.Add(types.NewIOEffect(types.IOEffectDirectoryList, types.IOPermissionRead))

	// File writing.
	effects.Add(types.NewIOEffect(types.IOEffectFileWrite, types.IOPermissionWrite))
	effects.Add(types.NewIOEffect(types.IOEffectFileCreate, types.IOPermissionWrite))

	// Metadata operations.
	effects.Add(types.NewIOEffect(types.IOEffectFileMetadata, types.IOPermissionRead))

	// Logging.
	effects.Add(types.NewIOEffect(types.IOEffectStdoutWrite, types.IOPermissionWrite))

	return effects
}

func analyzeWebAPIHandler() *types.IOEffectSet {
	// Simulates analysis of a web API handler:.
	// 1. Receive HTTP request
	// 2. Validate input
	// 3. Query database
	// 4. Process data
	// 5. Send HTTP response
	// 6. Log request

	effects := types.NewIOEffectSet()

	// Network operations.
	effects.Add(types.NewIOEffect(types.IOEffectHTTPRequest, types.IOPermissionRead))
	effects.Add(types.NewIOEffect(types.IOEffectHTTPResponse, types.IOPermissionWrite))

	// Database operations.
	effects.Add(types.NewIOEffect(types.IOEffectDatabaseQuery, types.IOPermissionRead))

	// Logging.
	effects.Add(types.NewIOEffect(types.IOEffectStdoutWrite, types.IOPermissionWrite))

	return effects
}

func analyzeDatabaseMigration() *types.IOEffectSet {
	// Simulates analysis of a database migration:.
	// 1. Connect to database
	// 2. Begin transaction
	// 3. Read schema
	// 4. Execute updates
	// 5. Commit transaction
	// 6. Update migration table

	effects := types.NewIOEffectSet()

	// Database operations.
	effects.Add(types.NewIOEffect(types.IOEffectDatabaseConnect, types.IOPermissionReadWrite))
	effects.Add(types.NewIOEffect(types.IOEffectDatabaseTransaction, types.IOPermissionWrite))
	effects.Add(types.NewIOEffect(types.IOEffectDatabaseQuery, types.IOPermissionRead))
	effects.Add(types.NewIOEffect(types.IOEffectDatabaseUpdate, types.IOPermissionWrite))
	effects.Add(types.NewIOEffect(types.IOEffectDatabaseCommit, types.IOPermissionWrite))

	return effects
}

func performSecurityAudit() {
	// Create a security-focused I/O context
	context := types.NewIOContext()

	// Only allow read operations and specific safe directories.
	context.AllowPermission(types.IOPermissionRead)
	context.AllowKind(types.IOEffectFileRead)
	context.AllowKind(types.IOEffectDirectoryList)

	// Test various operations against security policy.
	testOperations := []*types.IOEffect{
		createTestEffect(types.IOEffectFileRead, types.IOPermissionRead, "/var/log/app.log"),
		createTestEffect(types.IOEffectFileWrite, types.IOPermissionWrite, "/tmp/test.txt"),
		createTestEffect(types.IOEffectFileDelete, types.IOPermissionFullAccess, "/etc/passwd"),
		createTestEffect(types.IOEffectNetworkConnect, types.IOPermissionReadWrite, "external-api.com"),
		createTestEffect(types.IOEffectDatabaseQuery, types.IOPermissionRead, "users_table"),
	}

	fmt.Println("   Security policy: Read-only access to files and directories")
	for i, operation := range testOperations {
		allowed := context.CanPerform(operation)
		status := "DENIED"
		if allowed {
			status = "ALLOWED"
		}
		fmt.Printf("   Operation %d (%s): %s\n", i+1, operation.Kind.String(), status)
	}
}
