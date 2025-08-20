package intrinsics

// ExternKind represents the type of external function.
type ExternKind int

const (
	// C Runtime Library.
	ExternMalloc ExternKind = iota
	ExternFree
	ExternRealloc
	ExternMemcpy
	ExternMemset
	ExternMemmove
	ExternStrlen
	ExternStrcmp
	ExternPrintf
	ExternSprintf

	// File I/O.
	ExternFopen
	ExternFclose
	ExternFread
	ExternFwrite
	ExternFseek
	ExternFtell

	// System Calls (Windows).
	ExternVirtualAlloc
	ExternVirtualFree
	ExternGetCurrentProcess
	ExternGetCurrentThread
	ExternCreateThread
	ExternWaitForSingleObject
	ExternCloseHandle

	// System Calls (Unix/Linux).
	ExternMmap
	ExternMunmap
	ExternOpen
	ExternClose
	ExternRead
	ExternWrite
	ExternPthread_create
	ExternPthread_join
)

// ExternInfo describes an external function declaration.
type ExternInfo struct {
	Name      string
	Library   string
	Signature ExternSignature
	Kind      ExternKind
	Platform  PlatformSupport
	Calling   CallingConvention
}

// ExternSignature describes external function signature.
type ExternSignature struct {
	Parameters []ExternParameter
	ReturnType IntrinsicType
	IsVarArgs  bool
}

// ExternParameter describes an external function parameter.
type ExternParameter struct {
	Name string
	Type IntrinsicType
}

// CallingConvention specifies the calling convention.
type CallingConvention int

const (
	CallingC          CallingConvention = iota // C calling convention
	CallingStdcall                             // Windows stdcall
	CallingFastcall                            // Fastcall
	CallingVectorcall                          // Vectorcall
	CallingSystem                              // System default
)

// ExternRegistry manages external function declarations.
type ExternRegistry struct {
	externs    map[string]*ExternInfo
	byLibrary  map[string][]*ExternInfo
	byPlatform map[PlatformSupport][]*ExternInfo
}

// GlobalExternRegistry contains all external function declarations.
var GlobalExternRegistry *ExternRegistry

// NewExternRegistry creates a new external function registry.
func NewExternRegistry() *ExternRegistry {
	return &ExternRegistry{
		externs:    make(map[string]*ExternInfo),
		byLibrary:  make(map[string][]*ExternInfo),
		byPlatform: make(map[PlatformSupport][]*ExternInfo),
	}
}

// Register registers an external function.
func (er *ExternRegistry) Register(info *ExternInfo) {
	er.externs[info.Name] = info
	er.byLibrary[info.Library] = append(er.byLibrary[info.Library], info)
	er.byPlatform[info.Platform] = append(er.byPlatform[info.Platform], info)
}

// Lookup finds an external function by name.
func (er *ExternRegistry) Lookup(name string) (*ExternInfo, bool) {
	info, exists := er.externs[name]

	return info, exists
}

// GetByLibrary returns all externals in a library.
func (er *ExternRegistry) GetByLibrary(library string) []*ExternInfo {
	return er.byLibrary[library]
}

// GetByPlatform returns all externals for a platform.
func (er *ExternRegistry) GetByPlatform(platform PlatformSupport) []*ExternInfo {
	return er.byPlatform[platform]
}

// InitializeExterns initializes the global external function registry.
func InitializeExterns() {
	GlobalExternRegistry = NewExternRegistry()

	// Register C runtime functions.
	registerCRuntimeExterns()

	// Register platform-specific functions.
	registerWindowsExterns()
	registerUnixExterns()
}

// C Runtime Library Functions.
func registerCRuntimeExterns() {
	// malloc(size: usize) -> *void.
	GlobalExternRegistry.Register(&ExternInfo{
		Name: "malloc",
		Kind: ExternMalloc,
		Signature: ExternSignature{
			Parameters: []ExternParameter{
				{Name: "size", Type: IntrinsicUSize},
			},
			ReturnType: IntrinsicPtr,
		},
		Library:  "msvcrt.dll", // Windows
		Platform: PlatformAll,
		Calling:  CallingC,
	})

	// free(ptr: *void) -> void.
	GlobalExternRegistry.Register(&ExternInfo{
		Name: "free",
		Kind: ExternFree,
		Signature: ExternSignature{
			Parameters: []ExternParameter{
				{Name: "ptr", Type: IntrinsicPtr},
			},
			ReturnType: IntrinsicVoid,
		},
		Library:  "msvcrt.dll",
		Platform: PlatformAll,
		Calling:  CallingC,
	})

	// realloc(ptr: *void, new_size: usize) -> *void.
	GlobalExternRegistry.Register(&ExternInfo{
		Name: "realloc",
		Kind: ExternRealloc,
		Signature: ExternSignature{
			Parameters: []ExternParameter{
				{Name: "ptr", Type: IntrinsicPtr},
				{Name: "new_size", Type: IntrinsicUSize},
			},
			ReturnType: IntrinsicPtr,
		},
		Library:  "msvcrt.dll",
		Platform: PlatformAll,
		Calling:  CallingC,
	})

	// memcpy(dest: *void, src: *void, count: usize) -> *void.
	GlobalExternRegistry.Register(&ExternInfo{
		Name: "memcpy",
		Kind: ExternMemcpy,
		Signature: ExternSignature{
			Parameters: []ExternParameter{
				{Name: "dest", Type: IntrinsicPtr},
				{Name: "src", Type: IntrinsicPtr},
				{Name: "count", Type: IntrinsicUSize},
			},
			ReturnType: IntrinsicPtr,
		},
		Library:  "msvcrt.dll",
		Platform: PlatformAll,
		Calling:  CallingC,
	})

	// memset(ptr: *void, value: i32, count: usize) -> *void.
	GlobalExternRegistry.Register(&ExternInfo{
		Name: "memset",
		Kind: ExternMemset,
		Signature: ExternSignature{
			Parameters: []ExternParameter{
				{Name: "ptr", Type: IntrinsicPtr},
				{Name: "value", Type: IntrinsicI32},
				{Name: "count", Type: IntrinsicUSize},
			},
			ReturnType: IntrinsicPtr,
		},
		Library:  "msvcrt.dll",
		Platform: PlatformAll,
		Calling:  CallingC,
	})

	// printf(format: *i8, ...) -> i32
	GlobalExternRegistry.Register(&ExternInfo{
		Name: "printf",
		Kind: ExternPrintf,
		Signature: ExternSignature{
			Parameters: []ExternParameter{
				{Name: "format", Type: IntrinsicPtr},
			},
			ReturnType: IntrinsicI32,
			IsVarArgs:  true,
		},
		Library:  "msvcrt.dll",
		Platform: PlatformAll,
		Calling:  CallingC,
	})
}

// Windows-specific external functions.
func registerWindowsExterns() {
	// VirtualAlloc(lpAddress: *void, dwSize: usize, flAllocationType: u32, flProtect: u32) -> *void.
	GlobalExternRegistry.Register(&ExternInfo{
		Name: "VirtualAlloc",
		Kind: ExternVirtualAlloc,
		Signature: ExternSignature{
			Parameters: []ExternParameter{
				{Name: "lpAddress", Type: IntrinsicPtr},
				{Name: "dwSize", Type: IntrinsicUSize},
				{Name: "flAllocationType", Type: IntrinsicU32},
				{Name: "flProtect", Type: IntrinsicU32},
			},
			ReturnType: IntrinsicPtr,
		},
		Library:  "kernel32.dll",
		Platform: PlatformX64, // Windows x64
		Calling:  CallingStdcall,
	})

	// VirtualFree(lpAddress: *void, dwSize: usize, dwFreeType: u32) -> bool.
	GlobalExternRegistry.Register(&ExternInfo{
		Name: "VirtualFree",
		Kind: ExternVirtualFree,
		Signature: ExternSignature{
			Parameters: []ExternParameter{
				{Name: "lpAddress", Type: IntrinsicPtr},
				{Name: "dwSize", Type: IntrinsicUSize},
				{Name: "dwFreeType", Type: IntrinsicU32},
			},
			ReturnType: IntrinsicBool,
		},
		Library:  "kernel32.dll",
		Platform: PlatformX64,
		Calling:  CallingStdcall,
	})

	// GetCurrentProcess() -> *void.
	GlobalExternRegistry.Register(&ExternInfo{
		Name: "GetCurrentProcess",
		Kind: ExternGetCurrentProcess,
		Signature: ExternSignature{
			Parameters: []ExternParameter{},
			ReturnType: IntrinsicPtr,
		},
		Library:  "kernel32.dll",
		Platform: PlatformX64,
		Calling:  CallingStdcall,
	})
}

// Unix/Linux-specific external functions.
func registerUnixExterns() {
	// mmap(addr: *void, length: usize, prot: i32, flags: i32, fd: i32, offset: i64) -> *void.
	GlobalExternRegistry.Register(&ExternInfo{
		Name: "mmap",
		Kind: ExternMmap,
		Signature: ExternSignature{
			Parameters: []ExternParameter{
				{Name: "addr", Type: IntrinsicPtr},
				{Name: "length", Type: IntrinsicUSize},
				{Name: "prot", Type: IntrinsicI32},
				{Name: "flags", Type: IntrinsicI32},
				{Name: "fd", Type: IntrinsicI32},
				{Name: "offset", Type: IntrinsicI64},
			},
			ReturnType: IntrinsicPtr,
		},
		Library:  "libc.so.6",
		Platform: PlatformAll, // Unix-like systems
		Calling:  CallingC,
	})

	// munmap(addr: *void, length: usize) -> i32.
	GlobalExternRegistry.Register(&ExternInfo{
		Name: "munmap",
		Kind: ExternMunmap,
		Signature: ExternSignature{
			Parameters: []ExternParameter{
				{Name: "addr", Type: IntrinsicPtr},
				{Name: "length", Type: IntrinsicUSize},
			},
			ReturnType: IntrinsicI32,
		},
		Library:  "libc.so.6",
		Platform: PlatformAll,
		Calling:  CallingC,
	})
}

// IsExtern checks if a function name is an external function.
func IsExtern(name string) bool {
	if GlobalExternRegistry == nil {
		return false
	}

	_, exists := GlobalExternRegistry.Lookup(name)

	return exists
}

// GetExtern returns external function info for a function name.
func GetExtern(name string) (*ExternInfo, bool) {
	if GlobalExternRegistry == nil {
		return nil, false
	}

	return GlobalExternRegistry.Lookup(name)
}

// String returns the string representation of a calling convention.
func (cc CallingConvention) String() string {
	switch cc {
	case CallingC:
		return "C"
	case CallingStdcall:
		return "stdcall"
	case CallingFastcall:
		return "fastcall"
	case CallingVectorcall:
		return "vectorcall"
	case CallingSystem:
		return "system"
	default:
		return "unknown"
	}
}
