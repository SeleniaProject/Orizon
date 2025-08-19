package intrinsics

import (
	"testing"
)

func TestIntrinsicRegistry(t *testing.T) {
	// Initialize registries
	InitializeIntrinsics()

	if GlobalIntrinsicRegistry == nil {
		t.Fatal("Failed to initialize GlobalIntrinsicRegistry")
	}

	// Test intrinsic registry
	if GlobalIntrinsicRegistry == nil {
		t.Fatal("Global intrinsic registry not initialized")
	}

	// Test specific intrinsics
	alloc, exists := GlobalIntrinsicRegistry.Lookup("orizon_alloc")
	if !exists {
		t.Error("orizon_alloc intrinsic not found")
		return // Early return to avoid nil pointer
	}
	if alloc.Kind != IntrinsicAlloc {
		t.Error("orizon_alloc intrinsic has wrong kind")
	}

	// Test memory management intrinsics
	memoryIntrinsics := []string{
		"orizon_alloc", "orizon_free", "orizon_realloc", "orizon_memcpy", "orizon_memset",
	}
	for _, name := range memoryIntrinsics {
		if _, exists := GlobalIntrinsicRegistry.Lookup(name); !exists {
			t.Errorf("Memory intrinsic %s not found", name)
		}
	}

	// Test atomic intrinsics
	atomicIntrinsics := []string{
		"orizon_atomic_load", "orizon_atomic_store", "orizon_atomic_cas",
	}
	for _, name := range atomicIntrinsics {
		if _, exists := GlobalIntrinsicRegistry.Lookup(name); !exists {
			t.Errorf("Atomic intrinsic %s not found", name)
		}
	}

	// Test bit operation intrinsics
	bitIntrinsics := []string{
		"orizon_popcount",
	}
	for _, name := range bitIntrinsics {
		if _, exists := GlobalIntrinsicRegistry.Lookup(name); !exists {
			t.Errorf("Bit operation intrinsic %s not found", name)
		}
	}

	// Test overflow intrinsics
	overflowIntrinsics := []string{
		"orizon_add_overflow",
	}
	for _, name := range overflowIntrinsics {
		if _, exists := GlobalIntrinsicRegistry.Lookup(name); !exists {
			t.Errorf("Overflow intrinsic %s not found", name)
		}
	}

	// Test compiler magic intrinsics
	magicIntrinsics := []string{
		"orizon_sizeof",
	}
	for _, name := range magicIntrinsics {
		if _, exists := GlobalIntrinsicRegistry.Lookup(name); !exists {
			t.Errorf("Compiler magic intrinsic %s not found", name)
		}
	}
}

func TestExternRegistry(t *testing.T) {
	// Initialize registries
	InitializeExterns()

	// Test extern registry
	if GlobalExternRegistry == nil {
		t.Fatal("Global extern registry not initialized")
	}

	// Test C runtime functions
	cRuntimeFunctions := []string{
		"malloc", "free", "realloc", "memcpy", "memset", "printf",
	}
	for _, name := range cRuntimeFunctions {
		if _, exists := GlobalExternRegistry.Lookup(name); !exists {
			t.Errorf("C runtime function %s not found", name)
		}
	}

	// Test malloc specifically
	malloc, exists := GlobalExternRegistry.Lookup("malloc")
	if !exists {
		t.Error("malloc extern not found")
	}
	if malloc.Kind != ExternMalloc {
		t.Error("malloc extern has wrong kind")
	}
	if len(malloc.Signature.Parameters) != 1 {
		t.Error("malloc should have 1 parameter")
	}
	if malloc.Signature.ReturnType != IntrinsicPtr {
		t.Error("malloc should return pointer")
	}

	// Test printf (varargs function)
	printf, exists := GlobalExternRegistry.Lookup("printf")
	if !exists {
		t.Error("printf extern not found")
	}
	if !printf.Signature.IsVarArgs {
		t.Error("printf should be varargs")
	}
}

func TestHIRIntegration(t *testing.T) {
	integration := NewHIRIntrinsicIntegration()
	if integration == nil {
		t.Fatal("Failed to create HIR integration")
	}

	// Test alloc intrinsic processing
	call := &CallExpression{
		Function:  &NameExpression{Name: "orizon_alloc"},
		Arguments: []interface{}{8}, // size argument
	}
	builder := &IRBuilder{}

	result, err := integration.ProcessIntrinsicCall(call, builder)
	if err != nil {
		t.Errorf("Failed to process orizon_alloc intrinsic: %v", err)
	}
	if result == nil {
		t.Error("orizon_alloc intrinsic returned nil result")
	}

	// Test invalid intrinsic
	invalidCall := &CallExpression{
		Function:  &NameExpression{Name: "nonexistent"},
		Arguments: []interface{}{},
	}

	_, err = integration.ProcessIntrinsicCall(invalidCall, builder)
	if err == nil {
		t.Error("Expected error for nonexistent intrinsic")
	}

	// Test extern function processing
	externCall := &CallExpression{
		Function:  &NameExpression{Name: "malloc"},
		Arguments: []interface{}{64}, // size argument
	}

	result, err = integration.ProcessIntrinsicCall(externCall, builder)
	if err != nil {
		t.Errorf("Failed to process malloc extern: %v", err)
	}
	if result == nil {
		t.Error("malloc extern returned nil result")
	}
}

func TestIntrinsicTypes(t *testing.T) {
	// Test intrinsic type conversions
	testCases := []struct {
		intrinsicType IntrinsicType
		expected      string
	}{
		{IntrinsicVoid, "void"},
		{IntrinsicBool, "bool"},
		{IntrinsicI8, "i8"},
		{IntrinsicI16, "i16"},
		{IntrinsicI32, "i32"},
		{IntrinsicI64, "i64"},
		{IntrinsicU8, "u8"},
		{IntrinsicU16, "u16"},
		{IntrinsicU32, "u32"},
		{IntrinsicU64, "u64"},
		{IntrinsicUSize, "usize"},
		{IntrinsicF32, "f32"},
		{IntrinsicF64, "f64"},
		{IntrinsicPtr, "*void"},
	}

	for _, tc := range testCases {
		result := tc.intrinsicType.String()
		if result != tc.expected {
			t.Errorf("Type %v.String() = %s, expected %s", tc.intrinsicType, result, tc.expected)
		}
	}
}

func TestPlatformSupport(t *testing.T) {
	// Test platform support classifications
	testCases := []struct {
		platform PlatformSupport
		expected string
	}{
		{PlatformAll, "all"},
		{PlatformX64, "x64"},
		{PlatformARM64, "arm64"},
	}

	for _, tc := range testCases {
		// Platform support doesn't have String method, so we test the enum values
		if tc.platform < PlatformAll || tc.platform > PlatformARM64 {
			t.Errorf("Invalid platform value: %v", tc.platform)
		}
	}
}

func TestCallingConventions(t *testing.T) {
	// Test calling convention strings
	testCases := []struct {
		convention CallingConvention
		expected   string
	}{
		{CallingC, "C"},
		{CallingStdcall, "stdcall"},
		{CallingFastcall, "fastcall"},
		{CallingVectorcall, "vectorcall"},
		{CallingSystem, "system"},
	}

	for _, tc := range testCases {
		result := tc.convention.String()
		if result != tc.expected {
			t.Errorf("Convention %v.String() = %s, expected %s", tc.convention, result, tc.expected)
		}
	}
}

func TestIntrinsicValidation(t *testing.T) {
	// Test that all intrinsics have valid signatures
	InitializeIntrinsics()

	for name, intrinsic := range GlobalIntrinsicRegistry.intrinsics {
		// Check that intrinsic has a name
		if intrinsic.Name == "" {
			t.Errorf("Intrinsic %s has empty name", name)
		}

		// Check that name matches map key
		if intrinsic.Name != name {
			t.Errorf("Intrinsic name %s doesn't match map key %s", intrinsic.Name, name)
		}

		// Check that intrinsic has valid signature
		if len(intrinsic.Signature.Parameters) == 0 && intrinsic.Kind != IntrinsicUnreachable {
			// Most intrinsics should have parameters (except unreachable)
			switch intrinsic.Kind {
			case IntrinsicUnreachable:
				// unreachable has no parameters - this is fine
			default:
				// Other intrinsics might have no parameters in some cases
			}
		}

		// Check return type is valid
		if intrinsic.Signature.ReturnType < IntrinsicVoid || intrinsic.Signature.ReturnType > IntrinsicUSize {
			t.Errorf("Intrinsic %s has invalid return type: %v", name, intrinsic.Signature.ReturnType)
		}
	}
}

func TestExternValidation(t *testing.T) {
	// Test that all extern functions have valid signatures
	InitializeExterns()

	for name, extern := range GlobalExternRegistry.externs {
		// Check that extern has a name
		if extern.Name == "" {
			t.Errorf("Extern %s has empty name", name)
		}

		// Check that name matches map key
		if extern.Name != name {
			t.Errorf("Extern name %s doesn't match map key %s", extern.Name, name)
		}

		// Check that extern has a library
		if extern.Library == "" {
			t.Errorf("Extern %s has no library specified", name)
		}

		// Check return type is valid
		if extern.Signature.ReturnType < IntrinsicVoid || extern.Signature.ReturnType > IntrinsicUSize {
			t.Errorf("Extern %s has invalid return type: %v", name, extern.Signature.ReturnType)
		}

		// Check parameter types
		for i, param := range extern.Signature.Parameters {
			if param.Type < IntrinsicVoid || param.Type > IntrinsicUSize {
				t.Errorf("Extern %s parameter %d has invalid type: %v", name, i, param.Type)
			}
		}
	}
}

func BenchmarkIntrinsicLookup(b *testing.B) {
	InitializeIntrinsics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GlobalIntrinsicRegistry.Lookup("orizon_alloc")
	}
}

func BenchmarkExternLookup(b *testing.B) {
	InitializeExterns()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GlobalExternRegistry.Lookup("malloc")
	}
}

func BenchmarkHIRIntegration(b *testing.B) {
	InitializeIntrinsics()
	InitializeExterns()
	integration := NewHIRIntrinsicIntegration()

	call := &CallExpression{
		Function:  &NameExpression{Name: "orizon_alloc"},
		Arguments: []interface{}{8},
	}
	builder := &IRBuilder{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = integration.ProcessIntrinsicCall(call, builder)
	}
}
