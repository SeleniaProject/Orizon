// Package io provides MIR (Mid-level Intermediate Representation) integration
// for I/O operations in the Orizon compiler. This enables code generation
// for file I/O, console I/O, and threading primitives.
package io

import (
	"github.com/orizon-lang/orizon/internal/mir"
)

// IOMIRIntegration provides MIR-level code generation for I/O operations
type IOMIRIntegration struct {
	module *mir.Module
}

// NewIOMIRIntegration creates a new I/O MIR integration
func NewIOMIRIntegration(module *mir.Module) *IOMIRIntegration {
	return &IOMIRIntegration{
		module: module,
	}
}

// File I/O MIR Operations

// GenerateFileOpenFunction generates a MIR function for opening files
func (iom *IOMIRIntegration) GenerateFileOpenFunction() *mir.Function {
	function := &mir.Function{
		Name: "orizon_io_file_open",
		Parameters: []mir.Value{
			{Kind: mir.ValRef, Ref: "path", Class: mir.ClassInt},
			{Kind: mir.ValRef, Ref: "mode", Class: mir.ClassInt},
		},
		Blocks: []*mir.BasicBlock{
			{
				Name: "entry",
				Instr: []mir.Instr{
					&mir.Call{
						Dst:    "handle",
						Callee: "system_open_file",
						Args: []mir.Value{
							{Kind: mir.ValRef, Ref: "path", Class: mir.ClassInt},
							{Kind: mir.ValRef, Ref: "mode", Class: mir.ClassInt},
						},
					},
					&mir.Ret{
						Val: &mir.Value{Kind: mir.ValRef, Ref: "handle", Class: mir.ClassInt},
					},
				},
			},
		},
	}

	return function
}

// GenerateFileCloseFunction generates a MIR function for closing files
func (iom *IOMIRIntegration) GenerateFileCloseFunction() *mir.Function {
	function := &mir.Function{
		Name: "orizon_io_file_close",
		Parameters: []mir.Value{
			{Kind: mir.ValRef, Ref: "handle", Class: mir.ClassInt},
		},
		Blocks: []*mir.BasicBlock{
			{
				Name: "entry",
				Instr: []mir.Instr{
					&mir.Call{
						Dst:    "result",
						Callee: "system_close_file",
						Args: []mir.Value{
							{Kind: mir.ValRef, Ref: "handle", Class: mir.ClassInt},
						},
					},
					&mir.Ret{
						Val: &mir.Value{Kind: mir.ValRef, Ref: "result", Class: mir.ClassInt},
					},
				},
			},
		},
	}

	return function
}

// GenerateFileReadFunction generates a MIR function for reading from files
func (iom *IOMIRIntegration) GenerateFileReadFunction() *mir.Function {
	function := &mir.Function{
		Name: "orizon_io_file_read",
		Parameters: []mir.Value{
			{Kind: mir.ValRef, Ref: "handle", Class: mir.ClassInt},
			{Kind: mir.ValRef, Ref: "buffer", Class: mir.ClassInt},
			{Kind: mir.ValRef, Ref: "size", Class: mir.ClassInt},
		},
		Blocks: []*mir.BasicBlock{
			{
				Name: "entry",
				Instr: []mir.Instr{
					&mir.Call{
						Dst:    "bytes_read",
						Callee: "system_read_file",
						Args: []mir.Value{
							{Kind: mir.ValRef, Ref: "handle", Class: mir.ClassInt},
							{Kind: mir.ValRef, Ref: "buffer", Class: mir.ClassInt},
							{Kind: mir.ValRef, Ref: "size", Class: mir.ClassInt},
						},
					},
					&mir.Ret{
						Val: &mir.Value{Kind: mir.ValRef, Ref: "bytes_read", Class: mir.ClassInt},
					},
				},
			},
		},
	}

	return function
}

// GenerateFileWriteFunction generates a MIR function for writing to files
func (iom *IOMIRIntegration) GenerateFileWriteFunction() *mir.Function {
	function := &mir.Function{
		Name: "orizon_io_file_write",
		Parameters: []mir.Value{
			{Kind: mir.ValRef, Ref: "handle", Class: mir.ClassInt},
			{Kind: mir.ValRef, Ref: "buffer", Class: mir.ClassInt},
			{Kind: mir.ValRef, Ref: "size", Class: mir.ClassInt},
		},
		Blocks: []*mir.BasicBlock{
			{
				Name: "entry",
				Instr: []mir.Instr{
					&mir.Call{
						Dst:    "bytes_written",
						Callee: "system_write_file",
						Args: []mir.Value{
							{Kind: mir.ValRef, Ref: "handle", Class: mir.ClassInt},
							{Kind: mir.ValRef, Ref: "buffer", Class: mir.ClassInt},
							{Kind: mir.ValRef, Ref: "size", Class: mir.ClassInt},
						},
					},
					&mir.Ret{
						Val: &mir.Value{Kind: mir.ValRef, Ref: "bytes_written", Class: mir.ClassInt},
					},
				},
			},
		},
	}

	return function
}

// Console I/O MIR Operations

// GenerateConsoleWriteFunction generates a MIR function for writing to console
func (iom *IOMIRIntegration) GenerateConsoleWriteFunction() *mir.Function {
	function := &mir.Function{
		Name: "orizon_io_console_write",
		Parameters: []mir.Value{
			{Kind: mir.ValRef, Ref: "stream", Class: mir.ClassInt},
			{Kind: mir.ValRef, Ref: "buffer", Class: mir.ClassInt},
			{Kind: mir.ValRef, Ref: "size", Class: mir.ClassInt},
		},
		Blocks: []*mir.BasicBlock{
			{
				Name: "entry",
				Instr: []mir.Instr{
					&mir.Call{
						Dst:    "bytes_written",
						Callee: "system_console_write",
						Args: []mir.Value{
							{Kind: mir.ValRef, Ref: "stream", Class: mir.ClassInt},
							{Kind: mir.ValRef, Ref: "buffer", Class: mir.ClassInt},
							{Kind: mir.ValRef, Ref: "size", Class: mir.ClassInt},
						},
					},
					&mir.Ret{
						Val: &mir.Value{Kind: mir.ValRef, Ref: "bytes_written", Class: mir.ClassInt},
					},
				},
			},
		},
	}

	return function
}

// GenerateConsoleReadFunction generates a MIR function for reading from console
func (iom *IOMIRIntegration) GenerateConsoleReadFunction() *mir.Function {
	function := &mir.Function{
		Name: "orizon_io_console_read",
		Parameters: []mir.Value{
			{Kind: mir.ValRef, Ref: "buffer", Class: mir.ClassInt},
			{Kind: mir.ValRef, Ref: "size", Class: mir.ClassInt},
		},
		Blocks: []*mir.BasicBlock{
			{
				Name: "entry",
				Instr: []mir.Instr{
					&mir.Call{
						Dst:    "bytes_read",
						Callee: "system_console_read",
						Args: []mir.Value{
							{Kind: mir.ValRef, Ref: "buffer", Class: mir.ClassInt},
							{Kind: mir.ValRef, Ref: "size", Class: mir.ClassInt},
						},
					},
					&mir.Ret{
						Val: &mir.Value{Kind: mir.ValRef, Ref: "bytes_read", Class: mir.ClassInt},
					},
				},
			},
		},
	}

	return function
}

// Threading MIR Operations

// GenerateThreadCreateFunction generates a MIR function for creating threads
func (iom *IOMIRIntegration) GenerateThreadCreateFunction() *mir.Function {
	function := &mir.Function{
		Name: "orizon_thread_create",
		Parameters: []mir.Value{
			{Kind: mir.ValRef, Ref: "function_ptr", Class: mir.ClassInt},
			{Kind: mir.ValRef, Ref: "data", Class: mir.ClassInt},
			{Kind: mir.ValRef, Ref: "priority", Class: mir.ClassInt},
		},
		Blocks: []*mir.BasicBlock{
			{
				Name: "entry",
				Instr: []mir.Instr{
					&mir.Call{
						Dst:    "thread_handle",
						Callee: "system_create_thread",
						Args: []mir.Value{
							{Kind: mir.ValRef, Ref: "function_ptr", Class: mir.ClassInt},
							{Kind: mir.ValRef, Ref: "data", Class: mir.ClassInt},
							{Kind: mir.ValRef, Ref: "priority", Class: mir.ClassInt},
						},
					},
					&mir.Ret{
						Val: &mir.Value{Kind: mir.ValRef, Ref: "thread_handle", Class: mir.ClassInt},
					},
				},
			},
		},
	}

	return function
}

// GenerateThreadJoinFunction generates a MIR function for joining threads
func (iom *IOMIRIntegration) GenerateThreadJoinFunction() *mir.Function {
	function := &mir.Function{
		Name: "orizon_thread_join",
		Parameters: []mir.Value{
			{Kind: mir.ValRef, Ref: "thread_handle", Class: mir.ClassInt},
			{Kind: mir.ValRef, Ref: "timeout", Class: mir.ClassInt},
		},
		Blocks: []*mir.BasicBlock{
			{
				Name: "entry",
				Instr: []mir.Instr{
					&mir.Call{
						Dst:    "result",
						Callee: "system_join_thread",
						Args: []mir.Value{
							{Kind: mir.ValRef, Ref: "thread_handle", Class: mir.ClassInt},
							{Kind: mir.ValRef, Ref: "timeout", Class: mir.ClassInt},
						},
					},
					&mir.Ret{
						Val: &mir.Value{Kind: mir.ValRef, Ref: "result", Class: mir.ClassInt},
					},
				},
			},
		},
	}

	return function
}

// Synchronization MIR Operations

// GenerateMutexCreateFunction generates a MIR function for creating mutexes
func (iom *IOMIRIntegration) GenerateMutexCreateFunction() *mir.Function {
	function := &mir.Function{
		Name:       "orizon_mutex_create",
		Parameters: []mir.Value{},
		Blocks: []*mir.BasicBlock{
			{
				Name: "entry",
				Instr: []mir.Instr{
					&mir.Call{
						Dst:    "mutex_handle",
						Callee: "system_create_mutex",
						Args:   []mir.Value{},
					},
					&mir.Ret{
						Val: &mir.Value{Kind: mir.ValRef, Ref: "mutex_handle", Class: mir.ClassInt},
					},
				},
			},
		},
	}

	return function
}

// GenerateMutexLockFunction generates a MIR function for locking mutexes
func (iom *IOMIRIntegration) GenerateMutexLockFunction() *mir.Function {
	function := &mir.Function{
		Name: "orizon_mutex_lock",
		Parameters: []mir.Value{
			{Kind: mir.ValRef, Ref: "mutex_handle", Class: mir.ClassInt},
		},
		Blocks: []*mir.BasicBlock{
			{
				Name: "entry",
				Instr: []mir.Instr{
					&mir.Call{
						Dst:    "result",
						Callee: "system_lock_mutex",
						Args: []mir.Value{
							{Kind: mir.ValRef, Ref: "mutex_handle", Class: mir.ClassInt},
						},
					},
					&mir.Ret{
						Val: &mir.Value{Kind: mir.ValRef, Ref: "result", Class: mir.ClassInt},
					},
				},
			},
		},
	}

	return function
}

// GenerateChannelCreateFunction generates a MIR function for creating channels
func (iom *IOMIRIntegration) GenerateChannelCreateFunction() *mir.Function {
	function := &mir.Function{
		Name: "orizon_channel_create",
		Parameters: []mir.Value{
			{Kind: mir.ValRef, Ref: "capacity", Class: mir.ClassInt},
		},
		Blocks: []*mir.BasicBlock{
			{
				Name: "entry",
				Instr: []mir.Instr{
					&mir.Call{
						Dst:    "channel_handle",
						Callee: "system_create_channel",
						Args: []mir.Value{
							{Kind: mir.ValRef, Ref: "capacity", Class: mir.ClassInt},
						},
					},
					&mir.Ret{
						Val: &mir.Value{Kind: mir.ValRef, Ref: "channel_handle", Class: mir.ClassInt},
					},
				},
			},
		},
	}

	return function
}

// GetIOFunctions returns all I/O-related MIR functions
func (iom *IOMIRIntegration) GetIOFunctions() []*mir.Function {
	return []*mir.Function{
		iom.GenerateFileOpenFunction(),
		iom.GenerateFileCloseFunction(),
		iom.GenerateFileReadFunction(),
		iom.GenerateFileWriteFunction(),
		iom.GenerateConsoleWriteFunction(),
		iom.GenerateConsoleReadFunction(),
		iom.GenerateThreadCreateFunction(),
		iom.GenerateThreadJoinFunction(),
		iom.GenerateMutexCreateFunction(),
		iom.GenerateMutexLockFunction(),
		iom.GenerateChannelCreateFunction(),
	}
}

// GetExternalFunctions returns the list of external functions needed for I/O operations
func (iom *IOMIRIntegration) GetExternalFunctions() []string {
	return []string{
		// File I/O functions
		"system_open_file",
		"system_close_file",
		"system_read_file",
		"system_write_file",

		// Console I/O functions
		"system_console_write",
		"system_console_read",

		// Threading functions
		"system_create_thread",
		"system_join_thread",

		// Synchronization functions
		"system_create_mutex",
		"system_lock_mutex",
		"system_create_channel",
	}
}
