// Package kernel provides kernel intrinsics for the Orizon compiler
package kernel

import (
	"github.com/orizon-lang/orizon/internal/intrinsics"
)

// RegisterKernelIntrinsics registers all kernel-related intrinsics
func RegisterKernelIntrinsics() {
	// Memory management intrinsics
	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_alloc_page",
		Kind: intrinsics.IntrinsicSizeof, // Using existing kind as placeholder
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{},
			ReturnType: intrinsics.IntrinsicUSize,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategoryMemory,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_free_page",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "addr", Type: intrinsics.IntrinsicUSize},
			},
			ReturnType: intrinsics.IntrinsicBool,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategoryMemory,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_get_memory_info",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "total", Type: intrinsics.IntrinsicPtr},
				{Name: "free", Type: intrinsics.IntrinsicPtr},
				{Name: "used", Type: intrinsics.IntrinsicPtr},
			},
			ReturnType: intrinsics.IntrinsicVoid,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategoryMemory,
		Platform: intrinsics.PlatformAll,
	})

	// Process management intrinsics
	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_create_process",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "name_ptr", Type: intrinsics.IntrinsicPtr},
				{Name: "name_len", Type: intrinsics.IntrinsicUSize},
				{Name: "entry_point", Type: intrinsics.IntrinsicPtr},
				{Name: "stack_size", Type: intrinsics.IntrinsicUSize},
			},
			ReturnType: intrinsics.IntrinsicU32,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategorySystem,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_kill_process",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "pid", Type: intrinsics.IntrinsicU32},
			},
			ReturnType: intrinsics.IntrinsicBool,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategorySystem,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_get_current_pid",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{},
			ReturnType: intrinsics.IntrinsicU32,
			IsUnsafe:   false,
		},
		Category: intrinsics.CategorySystem,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_yield",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{},
			ReturnType: intrinsics.IntrinsicVoid,
			IsUnsafe:   false,
		},
		Category: intrinsics.CategorySystem,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_sleep",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "ms", Type: intrinsics.IntrinsicU64},
			},
			ReturnType: intrinsics.IntrinsicVoid,
			IsUnsafe:   false,
		},
		Category: intrinsics.CategorySystem,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_get_uptime",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{},
			ReturnType: intrinsics.IntrinsicU64,
			IsUnsafe:   false,
		},
		Category: intrinsics.CategorySystem,
		Platform: intrinsics.PlatformAll,
	})

	// File system intrinsics
	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_open_file",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "path_ptr", Type: intrinsics.IntrinsicPtr},
				{Name: "path_len", Type: intrinsics.IntrinsicUSize},
				{Name: "flags", Type: intrinsics.IntrinsicU32},
			},
			ReturnType: intrinsics.IntrinsicPtr,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategoryIO,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_close_file",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "file_ptr", Type: intrinsics.IntrinsicPtr},
			},
			ReturnType: intrinsics.IntrinsicBool,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategoryIO,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_read_file",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "file_ptr", Type: intrinsics.IntrinsicPtr},
				{Name: "buffer_ptr", Type: intrinsics.IntrinsicPtr},
				{Name: "buffer_len", Type: intrinsics.IntrinsicUSize},
			},
			ReturnType: intrinsics.IntrinsicI32,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategoryIO,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_write_file",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "file_ptr", Type: intrinsics.IntrinsicPtr},
				{Name: "data_ptr", Type: intrinsics.IntrinsicPtr},
				{Name: "data_len", Type: intrinsics.IntrinsicUSize},
			},
			ReturnType: intrinsics.IntrinsicI32,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategoryIO,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_create_file",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "path_ptr", Type: intrinsics.IntrinsicPtr},
				{Name: "path_len", Type: intrinsics.IntrinsicUSize},
				{Name: "permissions", Type: intrinsics.IntrinsicU16},
			},
			ReturnType: intrinsics.IntrinsicPtr,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategoryIO,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_mkdir",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "path_ptr", Type: intrinsics.IntrinsicPtr},
				{Name: "path_len", Type: intrinsics.IntrinsicUSize},
				{Name: "permissions", Type: intrinsics.IntrinsicU16},
			},
			ReturnType: intrinsics.IntrinsicBool,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategoryIO,
		Platform: intrinsics.PlatformAll,
	})

	// Hardware abstraction intrinsics
	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_disable_interrupts",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{},
			ReturnType: intrinsics.IntrinsicVoid,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategorySystem,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_enable_interrupts",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{},
			ReturnType: intrinsics.IntrinsicVoid,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategorySystem,
		Platform: intrinsics.PlatformAll,
	})

	// Port I/O intrinsics
	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_port_read_byte",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "port", Type: intrinsics.IntrinsicU16},
			},
			ReturnType: intrinsics.IntrinsicU8,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategorySystem,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_port_write_byte",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "port", Type: intrinsics.IntrinsicU16},
				{Name: "value", Type: intrinsics.IntrinsicU8},
			},
			ReturnType: intrinsics.IntrinsicVoid,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategorySystem,
		Platform: intrinsics.PlatformAll,
	})

	// Memory-mapped I/O intrinsics
	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_read_volatile8",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "addr", Type: intrinsics.IntrinsicPtr},
			},
			ReturnType: intrinsics.IntrinsicU8,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategoryMemory,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_write_volatile8",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "addr", Type: intrinsics.IntrinsicPtr},
				{Name: "value", Type: intrinsics.IntrinsicU8},
			},
			ReturnType: intrinsics.IntrinsicVoid,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategoryMemory,
		Platform: intrinsics.PlatformAll,
	})

	// Console output intrinsic
	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_write_console",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{
				{Name: "data_ptr", Type: intrinsics.IntrinsicPtr},
				{Name: "data_len", Type: intrinsics.IntrinsicUSize},
			},
			ReturnType: intrinsics.IntrinsicI32,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategoryIO,
		Platform: intrinsics.PlatformAll,
	})

	// Kernel initialization intrinsics
	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_bootstrap",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{},
			ReturnType: intrinsics.IntrinsicBool,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategorySystem,
		Platform: intrinsics.PlatformAll,
	})

	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_main",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{},
			ReturnType: intrinsics.IntrinsicVoid,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategorySystem,
		Platform: intrinsics.PlatformAll,
	})

	// Demo and test intrinsics
	intrinsics.GlobalIntrinsicRegistry.Register(&intrinsics.IntrinsicInfo{
		Name: "orizon_kernel_demo_os",
		Kind: intrinsics.IntrinsicKernelCall,
		Signature: intrinsics.IntrinsicSignature{
			Parameters: []intrinsics.IntrinsicParameter{},
			ReturnType: intrinsics.IntrinsicVoid,
			IsUnsafe:   true,
		},
		Category: intrinsics.CategorySystem,
		Platform: intrinsics.PlatformAll,
	})
}
