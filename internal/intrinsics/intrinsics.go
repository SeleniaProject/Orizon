// Package intrinsics provides compiler-dependent intrinsic functions and external declarations.
// for the Orizon programming language. These are essential for self-hosting and low-level operations.
package intrinsics

import "fmt"

// IntrinsicKind represents the type of intrinsic function.
type IntrinsicKind int

const (
	// Memory Management Intrinsics.
	IntrinsicAlloc IntrinsicKind = iota
	IntrinsicFree
	IntrinsicRealloc
	IntrinsicMemcpy
	IntrinsicMemset
	IntrinsicMemmove

	// Atomic Operations.
	IntrinsicAtomicLoad
	IntrinsicAtomicStore
	IntrinsicAtomicCAS
	IntrinsicAtomicAdd
	IntrinsicAtomicSub
	IntrinsicAtomicXor
	IntrinsicAtomicOr
	IntrinsicAtomicAnd

	// Bit Operations.
	IntrinsicCtlz // Count leading zeros
	IntrinsicCttz // Count trailing zeros
	IntrinsicPopcount
	IntrinsicBswap // Byte swap

	// Arithmetic with Overflow.
	IntrinsicAddOverflow
	IntrinsicSubOverflow
	IntrinsicMulOverflow

	// Compiler Magic.
	IntrinsicSizeof
	IntrinsicAlignof
	IntrinsicUnreachable
	IntrinsicAssume
	IntrinsicTypeName
	IntrinsicTypeID

	// SIMD Operations (basic).
	IntrinsicVecAdd
	IntrinsicVecSub
	IntrinsicVecMul
	IntrinsicVecDiv

	// Architecture-specific (x64).
	IntrinsicRdtsc
	IntrinsicCpuid
	IntrinsicPrefetch
)

// IntrinsicInfo describes an intrinsic function.
type IntrinsicInfo struct {
	Name      string
	Signature IntrinsicSignature
	Kind      IntrinsicKind
	Category  IntrinsicCategory
	Platform  PlatformSupport
}

// IntrinsicSignature describes function signature.
type IntrinsicSignature struct {
	Parameters []IntrinsicParameter
	ReturnType IntrinsicType
	IsVarArgs  bool
	IsUnsafe   bool
}

// IntrinsicParameter describes a parameter.
type IntrinsicParameter struct {
	Name string
	Type IntrinsicType
}

// IntrinsicType represents intrinsic parameter/return types.
type IntrinsicType int

const (
	IntrinsicVoid IntrinsicType = iota
	IntrinsicBool
	IntrinsicI8
	IntrinsicI16
	IntrinsicI32
	IntrinsicI64
	IntrinsicU8
	IntrinsicU16
	IntrinsicU32
	IntrinsicU64
	IntrinsicF32
	IntrinsicF64
	IntrinsicPtr
	IntrinsicSize
	IntrinsicISize
	IntrinsicUSize
)

// IntrinsicCategory categorizes intrinsics.
type IntrinsicCategory int

const (
	CategoryMemory IntrinsicCategory = iota
	CategoryAtomic
	CategoryBitwise
	CategoryArithmetic
	CategoryCompilerMagic
	CategorySIMD
	CategoryArchSpecific
)

// PlatformSupport indicates platform availability.
type PlatformSupport int

const (
	PlatformAll PlatformSupport = iota
	PlatformX64
	PlatformARM64
	PlatformWasm
)

// GlobalIntrinsicRegistry contains all available intrinsics.
var GlobalIntrinsicRegistry *IntrinsicRegistry

// IntrinsicRegistry manages intrinsic function definitions.
type IntrinsicRegistry struct {
	intrinsics map[string]*IntrinsicInfo
	byCategory map[IntrinsicCategory][]*IntrinsicInfo
}

// NewIntrinsicRegistry creates a new intrinsic registry.
func NewIntrinsicRegistry() *IntrinsicRegistry {
	return &IntrinsicRegistry{
		intrinsics: make(map[string]*IntrinsicInfo),
		byCategory: make(map[IntrinsicCategory][]*IntrinsicInfo),
	}
}

// Register registers an intrinsic function.
func (ir *IntrinsicRegistry) Register(info *IntrinsicInfo) {
	ir.intrinsics[info.Name] = info
	ir.byCategory[info.Category] = append(ir.byCategory[info.Category], info)
}

// Lookup finds an intrinsic by name.
func (ir *IntrinsicRegistry) Lookup(name string) (*IntrinsicInfo, bool) {
	info, exists := ir.intrinsics[name]

	return info, exists
}

// GetByCategory returns all intrinsics in a category.
func (ir *IntrinsicRegistry) GetByCategory(category IntrinsicCategory) []*IntrinsicInfo {
	return ir.byCategory[category]
}

// GetAll returns all registered intrinsics.
func (ir *IntrinsicRegistry) GetAll() map[string]*IntrinsicInfo {
	return ir.intrinsics
}

// InitializeIntrinsics initializes the global intrinsic registry.
func InitializeIntrinsics() {
	GlobalIntrinsicRegistry = NewIntrinsicRegistry()

	// Register memory management intrinsics.
	registerMemoryIntrinsics()

	// Register atomic operations.
	registerAtomicIntrinsics()

	// Register bit operations.
	registerBitIntrinsics()

	// Register arithmetic with overflow.
	registerArithmeticIntrinsics()

	// Register compiler magic.
	registerCompilerMagicIntrinsics()

	// Register SIMD operations.
	registerSIMDIntrinsics()

	// Register architecture-specific.
	registerArchSpecificIntrinsics()
}

// Memory Management Intrinsics.
func registerMemoryIntrinsics() {
	// orizon_alloc(size: usize) -> *void.
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_alloc",
		Kind: IntrinsicAlloc,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{
				{Name: "size", Type: IntrinsicUSize},
			},
			ReturnType: IntrinsicPtr,
			IsUnsafe:   true,
		},
		Category: CategoryMemory,
		Platform: PlatformAll,
	})

	// orizon_free(ptr: *void) -> void.
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_free",
		Kind: IntrinsicFree,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{
				{Name: "ptr", Type: IntrinsicPtr},
			},
			ReturnType: IntrinsicVoid,
			IsUnsafe:   true,
		},
		Category: CategoryMemory,
		Platform: PlatformAll,
	})

	// orizon_realloc(ptr: *void, new_size: usize) -> *void.
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_realloc",
		Kind: IntrinsicRealloc,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{
				{Name: "ptr", Type: IntrinsicPtr},
				{Name: "new_size", Type: IntrinsicUSize},
			},
			ReturnType: IntrinsicPtr,
			IsUnsafe:   true,
		},
		Category: CategoryMemory,
		Platform: PlatformAll,
	})

	// orizon_memcpy(dest: *void, src: *void, count: usize) -> *void.
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_memcpy",
		Kind: IntrinsicMemcpy,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{
				{Name: "dest", Type: IntrinsicPtr},
				{Name: "src", Type: IntrinsicPtr},
				{Name: "count", Type: IntrinsicUSize},
			},
			ReturnType: IntrinsicPtr,
			IsUnsafe:   true,
		},
		Category: CategoryMemory,
		Platform: PlatformAll,
	})

	// orizon_memset(ptr: *void, value: i32, count: usize) -> *void.
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_memset",
		Kind: IntrinsicMemset,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{
				{Name: "ptr", Type: IntrinsicPtr},
				{Name: "value", Type: IntrinsicI32},
				{Name: "count", Type: IntrinsicUSize},
			},
			ReturnType: IntrinsicPtr,
			IsUnsafe:   true,
		},
		Category: CategoryMemory,
		Platform: PlatformAll,
	})
}

// Atomic Operations Intrinsics.
func registerAtomicIntrinsics() {
	// orizon_atomic_load(ptr: *T) -> T.
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_atomic_load",
		Kind: IntrinsicAtomicLoad,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{
				{Name: "ptr", Type: IntrinsicPtr},
			},
			ReturnType: IntrinsicU64, // Generic, will be specialized
			IsUnsafe:   true,
		},
		Category: CategoryAtomic,
		Platform: PlatformAll,
	})

	// orizon_atomic_store(ptr: *T, value: T) -> void.
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_atomic_store",
		Kind: IntrinsicAtomicStore,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{
				{Name: "ptr", Type: IntrinsicPtr},
				{Name: "value", Type: IntrinsicU64},
			},
			ReturnType: IntrinsicVoid,
			IsUnsafe:   true,
		},
		Category: CategoryAtomic,
		Platform: PlatformAll,
	})

	// orizon_atomic_cas(ptr: *T, expected: T, desired: T) -> bool.
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_atomic_cas",
		Kind: IntrinsicAtomicCAS,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{
				{Name: "ptr", Type: IntrinsicPtr},
				{Name: "expected", Type: IntrinsicU64},
				{Name: "desired", Type: IntrinsicU64},
			},
			ReturnType: IntrinsicBool,
			IsUnsafe:   true,
		},
		Category: CategoryAtomic,
		Platform: PlatformAll,
	})
}

// Bit Operations Intrinsics.
func registerBitIntrinsics() {
	// orizon_ctlz(value: u64) -> u32.
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_ctlz",
		Kind: IntrinsicCtlz,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{
				{Name: "value", Type: IntrinsicU64},
			},
			ReturnType: IntrinsicU32,
		},
		Category: CategoryBitwise,
		Platform: PlatformAll,
	})

	// orizon_cttz(value: u64) -> u32.
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_cttz",
		Kind: IntrinsicCttz,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{
				{Name: "value", Type: IntrinsicU64},
			},
			ReturnType: IntrinsicU32,
		},
		Category: CategoryBitwise,
		Platform: PlatformAll,
	})

	// orizon_popcount(value: u64) -> u32.
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_popcount",
		Kind: IntrinsicPopcount,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{
				{Name: "value", Type: IntrinsicU64},
			},
			ReturnType: IntrinsicU32,
		},
		Category: CategoryBitwise,
		Platform: PlatformAll,
	})
}

// Arithmetic with Overflow Intrinsics.
func registerArithmeticIntrinsics() {
	// orizon_add_overflow(a: T, b: T) -> (T, bool).
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_add_overflow",
		Kind: IntrinsicAddOverflow,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{
				{Name: "a", Type: IntrinsicU64},
				{Name: "b", Type: IntrinsicU64},
			},
			ReturnType: IntrinsicU64, // Returns tuple (result, overflow)
		},
		Category: CategoryArithmetic,
		Platform: PlatformAll,
	})
}

// Compiler Magic Intrinsics.
func registerCompilerMagicIntrinsics() {
	// orizon_sizeof(type: type) -> usize.
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_sizeof",
		Kind: IntrinsicSizeof,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{
				{Name: "type", Type: IntrinsicPtr}, // Type token
			},
			ReturnType: IntrinsicUSize,
		},
		Category: CategoryCompilerMagic,
		Platform: PlatformAll,
	})

	// orizon_alignof(type: type) -> usize.
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_alignof",
		Kind: IntrinsicAlignof,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{
				{Name: "type", Type: IntrinsicPtr}, // Type token
			},
			ReturnType: IntrinsicUSize,
		},
		Category: CategoryCompilerMagic,
		Platform: PlatformAll,
	})

	// orizon_unreachable() -> !.
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_unreachable",
		Kind: IntrinsicUnreachable,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{},
			ReturnType: IntrinsicVoid, // Actually never returns
		},
		Category: CategoryCompilerMagic,
		Platform: PlatformAll,
	})
}

// SIMD Operations Intrinsics.
func registerSIMDIntrinsics() {
	// Basic vector operations for 128-bit vectors.
	// These would be expanded based on target architecture.
}

// Architecture-Specific Intrinsics.
func registerArchSpecificIntrinsics() {
	// x64-specific intrinsics.
	// orizon_rdtsc() -> u64.
	GlobalIntrinsicRegistry.Register(&IntrinsicInfo{
		Name: "orizon_rdtsc",
		Kind: IntrinsicRdtsc,
		Signature: IntrinsicSignature{
			Parameters: []IntrinsicParameter{},
			ReturnType: IntrinsicU64,
			IsUnsafe:   true,
		},
		Category: CategoryArchSpecific,
		Platform: PlatformX64,
	})
}

// String returns the string representation of an intrinsic type.
func (it IntrinsicType) String() string {
	switch it {
	case IntrinsicVoid:
		return "void"
	case IntrinsicBool:
		return "bool"
	case IntrinsicI8:
		return "i8"
	case IntrinsicI16:
		return "i16"
	case IntrinsicI32:
		return "i32"
	case IntrinsicI64:
		return "i64"
	case IntrinsicU8:
		return "u8"
	case IntrinsicU16:
		return "u16"
	case IntrinsicU32:
		return "u32"
	case IntrinsicU64:
		return "u64"
	case IntrinsicF32:
		return "f32"
	case IntrinsicF64:
		return "f64"
	case IntrinsicPtr:
		return "*void"
	case IntrinsicSize:
		return "isize"
	case IntrinsicISize:
		return "isize"
	case IntrinsicUSize:
		return "usize"
	default:
		return "unknown"
	}
}

// GetIntrinsicType converts a type to IntrinsicType with type system integration.
func GetIntrinsicType(typeName string) IntrinsicType {
	// Handle basic types.
	switch typeName {
	case "bool", "boolean":
		return IntrinsicBool
	case "i8", "int8":
		return IntrinsicI8
	case "i16", "int16":
		return IntrinsicI16
	case "i32", "int32", "int":
		return IntrinsicI32
	case "i64", "int64":
		return IntrinsicI64
	case "u8", "uint8", "byte":
		return IntrinsicU8
	case "u16", "uint16":
		return IntrinsicU16
	case "u32", "uint32", "uint":
		return IntrinsicU32
	case "u64", "uint64":
		return IntrinsicU64
	case "f32", "float32", "float":
		return IntrinsicF32
	case "f64", "float64", "double":
		return IntrinsicF64
	case "usize":
		return IntrinsicUSize
	case "isize":
		return IntrinsicISize
	case "string", "str":
		return IntrinsicPtr // Strings are represented as pointers
	case "void":
		return IntrinsicVoid
	default:
		// Handle pointer types.
		if len(typeName) > 0 && typeName[0] == '*' {
			return IntrinsicPtr
		}
		// Handle array types.
		if len(typeName) > 2 && typeName[0] == '[' && typeName[len(typeName)-1] == ']' {
			return IntrinsicPtr // Arrays are represented as pointers
		}
		// Handle slice types.
		if len(typeName) > 2 && typeName[:2] == "[]" {
			return IntrinsicPtr // Slices are represented as pointers
		}
		// Handle generic or complex types.
		if len(typeName) > 0 && typeName[0] >= 'A' && typeName[0] <= 'Z' {
			// Could be a user-defined type or generic.
			return IntrinsicVoid // Default to void for unknown types
		}

		return IntrinsicVoid
	}
}

// IsIntrinsic checks if a function name is an intrinsic.
func IsIntrinsic(name string) bool {
	if GlobalIntrinsicRegistry == nil {
		return false
	}

	_, exists := GlobalIntrinsicRegistry.Lookup(name)

	return exists
}

// GetIntrinsic returns intrinsic info for a function name.
func GetIntrinsic(name string) (*IntrinsicInfo, bool) {
	if GlobalIntrinsicRegistry == nil {
		return nil, false
	}

	return GlobalIntrinsicRegistry.Lookup(name)
}

// ValidateIntrinsicCall validates that an intrinsic call has correct arguments.
func ValidateIntrinsicCall(intrinsic *IntrinsicInfo, args []IntrinsicType) error {
	if len(args) != len(intrinsic.Signature.Parameters) {
		return fmt.Errorf("intrinsic %s expects %d arguments, got %d",
			intrinsic.Name, len(intrinsic.Signature.Parameters), len(args))
	}

	for i, param := range intrinsic.Signature.Parameters {
		if args[i] != param.Type {
			return fmt.Errorf("intrinsic %s argument %d: expected %s, got %s",
				intrinsic.Name, i, param.Type.String(), args[i].String())
		}
	}

	return nil
}

// GetIntrinsicReturnType returns the return type of an intrinsic call.
func GetIntrinsicReturnType(intrinsic *IntrinsicInfo, args []IntrinsicType) (IntrinsicType, error) {
	if err := ValidateIntrinsicCall(intrinsic, args); err != nil {
		return IntrinsicVoid, err
	}

	return intrinsic.Signature.ReturnType, nil
}

// ConvertToNativeType converts an IntrinsicType to a platform-specific native type.
func ConvertToNativeType(intrinsicType IntrinsicType) string {
	switch intrinsicType {
	case IntrinsicBool:
		return "bool"
	case IntrinsicI8:
		return "int8_t"
	case IntrinsicI16:
		return "int16_t"
	case IntrinsicI32:
		return "int32_t"
	case IntrinsicI64:
		return "int64_t"
	case IntrinsicU8:
		return "uint8_t"
	case IntrinsicU16:
		return "uint16_t"
	case IntrinsicU32:
		return "uint32_t"
	case IntrinsicU64:
		return "uint64_t"
	case IntrinsicF32:
		return "float"
	case IntrinsicF64:
		return "double"
	case IntrinsicPtr:
		return "void*"
	case IntrinsicUSize:
		return "size_t"
	case IntrinsicISize:
		return "ssize_t"
	case IntrinsicVoid:
		return "void"
	default:
		return "void"
	}
}

// IsNumericType checks if the intrinsic type is numeric.
func IsNumericType(intrinsicType IntrinsicType) bool {
	switch intrinsicType {
	case IntrinsicI8, IntrinsicI16, IntrinsicI32, IntrinsicI64,
		IntrinsicU8, IntrinsicU16, IntrinsicU32, IntrinsicU64,
		IntrinsicF32, IntrinsicF64, IntrinsicUSize, IntrinsicISize:
		return true
	default:
		return false
	}
}

// IsIntegerType checks if the intrinsic type is an integer.
func IsIntegerType(intrinsicType IntrinsicType) bool {
	switch intrinsicType {
	case IntrinsicI8, IntrinsicI16, IntrinsicI32, IntrinsicI64,
		IntrinsicU8, IntrinsicU16, IntrinsicU32, IntrinsicU64,
		IntrinsicUSize, IntrinsicISize:
		return true
	default:
		return false
	}
}

// IsFloatType checks if the intrinsic type is a floating point type.
func IsFloatType(intrinsicType IntrinsicType) bool {
	return intrinsicType == IntrinsicF32 || intrinsicType == IntrinsicF64
}

// GetTypeSizeInBytes returns the size of an intrinsic type in bytes.
func GetTypeSizeInBytes(intrinsicType IntrinsicType) int {
	switch intrinsicType {
	case IntrinsicBool, IntrinsicI8, IntrinsicU8:
		return 1
	case IntrinsicI16, IntrinsicU16:
		return 2
	case IntrinsicI32, IntrinsicU32, IntrinsicF32:
		return 4
	case IntrinsicI64, IntrinsicU64, IntrinsicF64:
		return 8
	case IntrinsicPtr, IntrinsicUSize, IntrinsicISize:
		return 8 // Assuming 64-bit platform
	case IntrinsicVoid:
		return 0
	default:
		return 0
	}
}
