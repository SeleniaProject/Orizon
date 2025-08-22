// Package kernel provides interrupt handling and system call functionality
package kernel

import (
	"fmt"
	"sync"
	"unsafe"
)

// ============================================================================
// Interrupt handling
// ============================================================================

// InterruptHandler represents an interrupt handler function
type InterruptHandler func(context *InterruptContext)

// InterruptContext contains the CPU state during an interrupt
type InterruptContext struct {
	// General purpose registers
	RAX, RBX, RCX, RDX uint64
	RSI, RDI, RBP, RSP uint64
	R8, R9, R10, R11   uint64
	R12, R13, R14, R15 uint64

	// Segment registers
	CS, DS, ES, FS, GS, SS uint16

	// Control registers
	RIP, RFLAGS uint64

	// Error code (if applicable)
	ErrorCode uint64

	// Interrupt number
	InterruptNumber uint8
}

// IDTEntry represents an entry in the Interrupt Descriptor Table
type IDTEntry struct {
	OffsetLow    uint16 // Offset bits 0-15
	Selector     uint16 // Code segment selector
	IST          uint8  // Interrupt Stack Table offset
	TypeAttr     uint8  // Type and attributes
	OffsetMiddle uint16 // Offset bits 16-31
	OffsetHigh   uint32 // Offset bits 32-63
	Reserved     uint32 // Reserved (must be zero)
}

// IDTR represents the Interrupt Descriptor Table Register
type IDTR struct {
	Limit uint16  // Size of IDT - 1
	Base  uintptr // Base address of IDT
}

// InterruptManager manages interrupt handling
type InterruptManager struct {
	idt         [256]IDTEntry
	handlers    [256]InterruptHandler
	idtr        IDTR
	mutex       sync.RWMutex
	initialized bool
}

// GlobalInterruptManager provides global access to interrupt management
var GlobalInterruptManager *InterruptManager

// NewInterruptManager creates a new interrupt manager
func NewInterruptManager() *InterruptManager {
	im := &InterruptManager{}
	im.idtr.Limit = uint16(unsafe.Sizeof(im.idt)) - 1
	im.idtr.Base = uintptr(unsafe.Pointer(&im.idt[0]))
	return im
}

// InitializeInterrupts initializes the interrupt system
func InitializeInterrupts() error {
	if GlobalInterruptManager != nil && GlobalInterruptManager.initialized {
		return fmt.Errorf("interrupts already initialized")
	}

	GlobalInterruptManager = NewInterruptManager()

	// Set up default interrupt handlers
	GlobalInterruptManager.SetHandler(0, DivideByZeroHandler)
	GlobalInterruptManager.SetHandler(1, DebugHandler)
	GlobalInterruptManager.SetHandler(3, BreakpointHandler)
	GlobalInterruptManager.SetHandler(6, InvalidOpcodeHandler)
	GlobalInterruptManager.SetHandler(13, GeneralProtectionHandler)
	GlobalInterruptManager.SetHandler(14, PageFaultHandler)
	GlobalInterruptManager.SetHandler(0x80, SystemCallHandler) // Linux-style syscall

	// Install IDT
	err := GlobalInterruptManager.InstallIDT()
	if err != nil {
		return fmt.Errorf("failed to install IDT: %w", err)
	}

	GlobalInterruptManager.initialized = true
	return nil
}

// SetHandler sets an interrupt handler for a specific interrupt number
func (im *InterruptManager) SetHandler(interrupt uint8, handler InterruptHandler) {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	im.handlers[interrupt] = handler
	im.setIDTEntry(interrupt, handler)
}

// setIDTEntry sets up an IDT entry for an interrupt handler
func (im *InterruptManager) setIDTEntry(interrupt uint8, handler InterruptHandler) {
	if handler == nil {
		return
	}

	// Get handler address
	handlerAddr := getHandlerAddress(handler)

	// Set up IDT entry
	entry := &im.idt[interrupt]
	entry.OffsetLow = uint16(handlerAddr & 0xFFFF)
	entry.OffsetMiddle = uint16((handlerAddr >> 16) & 0xFFFF)
	entry.OffsetHigh = uint32((handlerAddr >> 32) & 0xFFFFFFFF)
	entry.Selector = 0x08 // Kernel code segment
	entry.IST = 0         // No IST
	entry.TypeAttr = 0x8E // Present, DPL=0, Interrupt Gate
	entry.Reserved = 0
}

// getHandlerAddress returns the address of an interrupt handler
// This is a placeholder - in real implementation, this would be assembly stubs
func getHandlerAddress(handler InterruptHandler) uintptr {
	// In a real kernel, this would return the address of assembly wrapper code
	// that saves registers, calls the Go handler, and restores registers
	return uintptr(unsafe.Pointer(&handler))
}

// InstallIDT installs the Interrupt Descriptor Table
func (im *InterruptManager) InstallIDT() error {
	// Platform-specific implementation
	// On x86_64, this would use the `lidt` instruction
	// For now, this is a placeholder
	return nil
}

// ============================================================================
// Default interrupt handlers
// ============================================================================

// DivideByZeroHandler handles divide by zero exceptions
func DivideByZeroHandler(ctx *InterruptContext) {
	fmt.Printf("EXCEPTION: Divide by zero at RIP=0x%x\n", ctx.RIP)
	// In a real kernel, this might kill the process or panic
}

// DebugHandler handles debug exceptions
func DebugHandler(ctx *InterruptContext) {
	fmt.Printf("DEBUG: Debug exception at RIP=0x%x\n", ctx.RIP)
}

// BreakpointHandler handles breakpoint exceptions
func BreakpointHandler(ctx *InterruptContext) {
	fmt.Printf("BREAKPOINT: Breakpoint at RIP=0x%x\n", ctx.RIP)
}

// InvalidOpcodeHandler handles invalid opcode exceptions
func InvalidOpcodeHandler(ctx *InterruptContext) {
	fmt.Printf("EXCEPTION: Invalid opcode at RIP=0x%x\n", ctx.RIP)
}

// GeneralProtectionHandler handles general protection faults
func GeneralProtectionHandler(ctx *InterruptContext) {
	fmt.Printf("EXCEPTION: General protection fault at RIP=0x%x, Error=0x%x\n",
		ctx.RIP, ctx.ErrorCode)
}

// PageFaultHandler handles page fault exceptions
func PageFaultHandler(ctx *InterruptContext) {
	// Get the faulting address from CR2 register
	faultAddr := getCR2() // Platform-specific function
	fmt.Printf("EXCEPTION: Page fault at RIP=0x%x, Address=0x%x, Error=0x%x\n",
		ctx.RIP, faultAddr, ctx.ErrorCode)

	// Handle page fault (allocate page, etc.)
	handlePageFault(faultAddr, ctx.ErrorCode)
}

// getCR2 returns the value of the CR2 register (faulting address)
func getCR2() uintptr {
	// Platform-specific implementation
	// On x86_64, this would read the CR2 register
	return 0 // Placeholder
}

// handlePageFault handles a page fault by allocating memory if needed
func handlePageFault(addr uintptr, errorCode uint64) {
	// Check if this is a valid access that just needs a page allocated
	present := (errorCode & 1) == 0
	write := (errorCode & 2) != 0
	user := (errorCode & 4) != 0

	fmt.Printf("Page fault: addr=0x%x, present=%v, write=%v, user=%v\n",
		addr, present, write, user)

	if present {
		// Page not present - try to allocate
		if GlobalKernel != nil {
			pageAddr, err := GlobalKernel.PhysicalMemory.AllocatePage()
			if err == nil {
				// Map the page (this would involve page table manipulation)
				fmt.Printf("Allocated page 0x%x for fault at 0x%x\n", pageAddr, addr)
			}
		}
	}
}

// ============================================================================
// System call interface
// ============================================================================

// SystemCallNumber represents different system calls
type SystemCallNumber uint64

const (
	SysRead SystemCallNumber = iota
	SysWrite
	SysOpen
	SysClose
	SysExit
	SysBrk
	SysMmap
	SysMunmap
	// Add more system calls as needed
)

// SystemCallHandler handles system call interrupts
func SystemCallHandler(ctx *InterruptContext) {
	// System call number is typically in RAX
	syscallNum := SystemCallNumber(ctx.RAX)

	// Arguments are typically in RDI, RSI, RDX, R10, R8, R9
	arg1 := ctx.RDI
	arg2 := ctx.RSI
	arg3 := ctx.RDX
	arg4 := ctx.R10
	arg5 := ctx.R8
	arg6 := ctx.R9

	// Dispatch system call
	result := dispatchSystemCall(syscallNum, arg1, arg2, arg3, arg4, arg5, arg6)

	// Return value goes in RAX
	ctx.RAX = result
}

// dispatchSystemCall dispatches a system call to the appropriate handler
func dispatchSystemCall(num SystemCallNumber, arg1, arg2, arg3, arg4, arg5, arg6 uint64) uint64 {
	switch num {
	case SysRead:
		return handleSysRead(arg1, arg2, arg3)
	case SysWrite:
		return handleSysWrite(arg1, arg2, arg3)
	case SysOpen:
		return handleSysOpen(arg1, arg2, arg3)
	case SysClose:
		return handleSysClose(arg1)
	case SysExit:
		return handleSysExit(arg1)
	case SysBrk:
		return handleSysBrk(arg1)
	case SysMmap:
		return handleSysMmap(arg1, arg2, arg3, arg4, arg5, arg6)
	case SysMunmap:
		return handleSysMunmap(arg1, arg2)
	default:
		fmt.Printf("Unknown system call: %d\n", num)
		return ^uint64(0) // Return -1 (error)
	}
}

// ============================================================================
// System call implementations
// ============================================================================

// handleSysRead handles the read system call
func handleSysRead(fd, buf, count uint64) uint64 {
	fmt.Printf("SYS_READ: fd=%d, buf=0x%x, count=%d\n", fd, buf, count)
	// Placeholder implementation
	return 0
}

// handleSysWrite handles the write system call
func handleSysWrite(fd, buf, count uint64) uint64 {
	if fd == 1 || fd == 2 { // stdout or stderr
		// Write to console
		data := (*[1 << 30]byte)(unsafe.Pointer(uintptr(buf)))[:count:count]
		fmt.Printf("%s", string(data))
		return count
	}
	fmt.Printf("SYS_WRITE: fd=%d, buf=0x%x, count=%d\n", fd, buf, count)
	return ^uint64(0) // Error
}

// handleSysOpen handles the open system call
func handleSysOpen(path, flags, mode uint64) uint64 {
	pathStr := cStringToGo(uintptr(path))
	fmt.Printf("SYS_OPEN: path=%s, flags=0x%x, mode=0x%x\n", pathStr, flags, mode)
	// Placeholder implementation
	return 3 // Return fake file descriptor
}

// handleSysClose handles the close system call
func handleSysClose(fd uint64) uint64 {
	fmt.Printf("SYS_CLOSE: fd=%d\n", fd)
	return 0
}

// handleSysExit handles the exit system call
func handleSysExit(code uint64) uint64 {
	fmt.Printf("SYS_EXIT: code=%d\n", code)
	// In a real kernel, this would terminate the process
	return 0
}

// handleSysBrk handles the brk system call (heap management)
func handleSysBrk(addr uint64) uint64 {
	fmt.Printf("SYS_BRK: addr=0x%x\n", addr)
	// Placeholder implementation
	return addr
}

// handleSysMmap handles the mmap system call
func handleSysMmap(addr, length, prot, flags, fd, offset uint64) uint64 {
	fmt.Printf("SYS_MMAP: addr=0x%x, len=%d, prot=0x%x, flags=0x%x, fd=%d, off=%d\n",
		addr, length, prot, flags, fd, offset)

	// Simple implementation: allocate pages
	if GlobalKernel != nil {
		pageCount := (length + DefaultPageSize - 1) / DefaultPageSize
		startAddr := addr
		if startAddr == 0 {
			// Let kernel choose address
			startAddr = 0x40000000 // Placeholder base address
		}

		for i := uint64(0); i < pageCount; i++ {
			physPage, err := GlobalKernel.PhysicalMemory.AllocatePage()
			if err != nil {
				return ^uint64(0) // Error
			}

			// Map virtual page to physical page
			// This would involve page table manipulation
			_ = physPage
		}

		return startAddr
	}

	return ^uint64(0) // Error
}

// handleSysMunmap handles the munmap system call
func handleSysMunmap(addr, length uint64) uint64 {
	fmt.Printf("SYS_MUNMAP: addr=0x%x, len=%d\n", addr, length)
	// Placeholder implementation
	return 0
}

// ============================================================================
// Helper functions
// ============================================================================

// cStringToGo converts a C-style null-terminated string to Go string
func cStringToGo(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}

	// Find the length of the string
	p := (*[1 << 30]byte)(unsafe.Pointer(ptr))
	var length int
	for i := 0; i < len(p); i++ {
		if p[i] == 0 {
			length = i
			break
		}
	}

	return string(p[:length])
}

// ============================================================================
// Kernel API functions for Orizon
// ============================================================================

// KernelRegisterInterruptHandler registers an interrupt handler
func KernelRegisterInterruptHandler(interrupt uint8, handler uintptr) bool {
	if GlobalInterruptManager == nil {
		return false
	}

	// Convert function pointer to Go function
	// This is a simplified conversion - real implementation would be more complex
	goHandler := func(ctx *InterruptContext) {
		// Call the Orizon function (this is a placeholder)
		fmt.Printf("Calling Orizon interrupt handler at 0x%x for interrupt %d\n",
			handler, interrupt)
	}

	GlobalInterruptManager.SetHandler(interrupt, goHandler)
	return true
}

// KernelDisableInterrupts disables interrupts
func KernelDisableInterrupts() {
	DisableInterrupts()
}

// KernelEnableInterrupts enables interrupts
func KernelEnableInterrupts() {
	EnableInterrupts()
}

// KernelSystemCall performs a system call
func KernelSystemCall(num, arg1, arg2, arg3, arg4, arg5, arg6 uint64) uint64 {
	return dispatchSystemCall(SystemCallNumber(num), arg1, arg2, arg3, arg4, arg5, arg6)
}
