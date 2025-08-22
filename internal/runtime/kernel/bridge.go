// Package kernel provides the bridge between Orizon code and kernel functionality
package kernel

import (
	"fmt"
	"unsafe"
)

// ============================================================================
// Kernel API Bridge for Orizon
// ============================================================================

// This file provides the interface that Orizon code can call to access
// kernel functionality. These functions are exported and can be called
// from compiled Orizon programs.

// ============================================================================
// Memory management API
// ============================================================================

//export orizon_kernel_alloc_page
func orizon_kernel_alloc_page() uintptr {
	return KernelAllocatePage()
}

//export orizon_kernel_free_page
func orizon_kernel_free_page(addr uintptr) bool {
	return KernelFreePage(addr)
}

//export orizon_kernel_get_memory_info
func orizon_kernel_get_memory_info(total, free, used *uint64) {
	*total, *free, *used = KernelGetMemoryInfo()
}

// ============================================================================
// Process management API
// ============================================================================

//export orizon_kernel_create_process
func orizon_kernel_create_process(name_ptr uintptr, name_len uintptr, entry_point uintptr, stack_size uintptr) uint32 {
	name := cStringToGoFromPtr(name_ptr, name_len)
	return KernelCreateProcess(name, entry_point, stack_size)
}

//export orizon_kernel_kill_process
func orizon_kernel_kill_process(pid uint32) bool {
	return KernelKillProcess(pid)
}

//export orizon_kernel_get_current_pid
func orizon_kernel_get_current_pid() uint32 {
	return KernelGetCurrentPID()
}

//export orizon_kernel_yield
func orizon_kernel_yield() {
	KernelYield()
}

//export orizon_kernel_sleep
func orizon_kernel_sleep(ms uint64) {
	KernelSleep(ms)
}

//export orizon_kernel_get_uptime
func orizon_kernel_get_uptime() uint64 {
	return KernelGetUptime()
}

// ============================================================================
// File system API
// ============================================================================

//export orizon_kernel_open_file
func orizon_kernel_open_file(path_ptr uintptr, path_len uintptr, flags uint32) uintptr {
	path := cStringToGoFromPtr(path_ptr, path_len)
	return KernelOpenFile(path, flags)
}

//export orizon_kernel_close_file
func orizon_kernel_close_file(file_ptr uintptr) bool {
	return KernelCloseFile(file_ptr)
}

//export orizon_kernel_read_file
func orizon_kernel_read_file(file_ptr uintptr, buffer_ptr uintptr, buffer_len uintptr) int {
	buffer := (*[1 << 30]byte)(unsafe.Pointer(buffer_ptr))[:buffer_len:buffer_len]
	return KernelReadFile(file_ptr, buffer)
}

//export orizon_kernel_write_file
func orizon_kernel_write_file(file_ptr uintptr, data_ptr uintptr, data_len uintptr) int {
	data := (*[1 << 30]byte)(unsafe.Pointer(data_ptr))[:data_len:data_len]
	return KernelWriteFile(file_ptr, data)
}

//export orizon_kernel_create_file
func orizon_kernel_create_file(path_ptr uintptr, path_len uintptr, permissions uint16) uintptr {
	path := cStringToGoFromPtr(path_ptr, path_len)
	return KernelCreateFile(path, permissions)
}

//export orizon_kernel_mkdir
func orizon_kernel_mkdir(path_ptr uintptr, path_len uintptr, permissions uint16) bool {
	path := cStringToGoFromPtr(path_ptr, path_len)
	return KernelMkdir(path, permissions)
}

// ============================================================================
// Hardware abstraction API
// ============================================================================

//export orizon_kernel_disable_interrupts
func orizon_kernel_disable_interrupts() {
	KernelDisableInterrupts()
}

//export orizon_kernel_enable_interrupts
func orizon_kernel_enable_interrupts() {
	KernelEnableInterrupts()
}

//export orizon_kernel_register_interrupt_handler
func orizon_kernel_register_interrupt_handler(interrupt uint8, handler uintptr) bool {
	return KernelRegisterInterruptHandler(interrupt, handler)
}

//export orizon_kernel_system_call
func orizon_kernel_system_call(num, arg1, arg2, arg3, arg4, arg5, arg6 uint64) uint64 {
	return KernelSystemCall(num, arg1, arg2, arg3, arg4, arg5, arg6)
}

// ============================================================================
// Console/Device API
// ============================================================================

//export orizon_kernel_write_console
func orizon_kernel_write_console(data_ptr uintptr, data_len uintptr) int {
	data := (*[1 << 30]byte)(unsafe.Pointer(data_ptr))[:data_len:data_len]
	return KernelWriteConsole(data)
}

// ============================================================================
// Port I/O API (for device drivers)
// ============================================================================

//export orizon_kernel_port_read_byte
func orizon_kernel_port_read_byte(port uint16) uint8 {
	ioPort := NewIOPort(port)
	return ioPort.ReadByte()
}

//export orizon_kernel_port_write_byte
func orizon_kernel_port_write_byte(port uint16, value uint8) {
	ioPort := NewIOPort(port)
	ioPort.WriteByte(value)
}

//export orizon_kernel_port_read_word
func orizon_kernel_port_read_word(port uint16) uint16 {
	ioPort := NewIOPort(port)
	return ioPort.ReadWord()
}

//export orizon_kernel_port_write_word
func orizon_kernel_port_write_word(port uint16, value uint16) {
	ioPort := NewIOPort(port)
	ioPort.WriteWord(value)
}

//export orizon_kernel_port_read_dword
func orizon_kernel_port_read_dword(port uint16) uint32 {
	ioPort := NewIOPort(port)
	return ioPort.ReadDWord()
}

//export orizon_kernel_port_write_dword
func orizon_kernel_port_write_dword(port uint16, value uint32) {
	ioPort := NewIOPort(port)
	ioPort.WriteDWord(value)
}

// ============================================================================
// Memory-mapped I/O API
// ============================================================================

//export orizon_kernel_read_volatile8
func orizon_kernel_read_volatile8(addr uintptr) uint8 {
	return ReadVolatile8(addr)
}

//export orizon_kernel_write_volatile8
func orizon_kernel_write_volatile8(addr uintptr, value uint8) {
	WriteVolatile8(addr, value)
}

//export orizon_kernel_read_volatile16
func orizon_kernel_read_volatile16(addr uintptr) uint16 {
	return ReadVolatile16(addr)
}

//export orizon_kernel_write_volatile16
func orizon_kernel_write_volatile16(addr uintptr, value uint16) {
	WriteVolatile16(addr, value)
}

//export orizon_kernel_read_volatile32
func orizon_kernel_read_volatile32(addr uintptr) uint32 {
	return ReadVolatile32(addr)
}

//export orizon_kernel_write_volatile32
func orizon_kernel_write_volatile32(addr uintptr, value uint32) {
	WriteVolatile32(addr, value)
}

//export orizon_kernel_read_volatile64
func orizon_kernel_read_volatile64(addr uintptr) uint64 {
	return ReadVolatile64(addr)
}

//export orizon_kernel_write_volatile64
func orizon_kernel_write_volatile64(addr uintptr, value uint64) {
	WriteVolatile64(addr, value)
}

// ============================================================================
// Kernel initialization API
// ============================================================================

//export orizon_kernel_bootstrap
func orizon_kernel_bootstrap() bool {
	// Simple bootstrap without boot info
	err := BootStrap(nil)
	if err != nil {
		fmt.Printf("Kernel bootstrap failed: %v\n", err)
		return false
	}
	return true
}

//export orizon_kernel_main
func orizon_kernel_main() {
	KernelMain()
}

// ============================================================================
// Helper functions
// ============================================================================

// cStringToGoFromPtr converts a pointer and length to a Go string
func cStringToGoFromPtr(ptr uintptr, length uintptr) string {
	if ptr == 0 || length == 0 {
		return ""
	}

	data := (*[1 << 30]byte)(unsafe.Pointer(ptr))[:length:length]
	return string(data)
}

// ============================================================================
// Platform-specific implementations
// ============================================================================

// These functions need to be implemented with platform-specific assembly
// or syscalls for actual hardware access. For now, they're placeholders.

// For x86_64 Linux, we can use syscalls to simulate some hardware access
func platformSpecificPortRead(port uint16) uint8 {
	// On real hardware, this would use the `in` instruction
	// For simulation on Linux, we could use /dev/port or ioperm/iopl syscalls
	return 0
}

func platformSpecificPortWrite(port uint16, value uint8) {
	// On real hardware, this would use the `out` instruction
	// For simulation, we would write to /dev/port or use syscalls
}

// For actual OS development, these would be implemented in assembly:
/*
Assembly implementations for x86_64:

.global orizon_asm_port_read_byte
orizon_asm_port_read_byte:
    mov %di, %dx      // port number to DX
    in %dx, %al       // read byte from port
    ret

.global orizon_asm_port_write_byte
orizon_asm_port_write_byte:
    mov %di, %dx      // port number to DX
    mov %si, %ax      // value to AL
    out %al, %dx      // write byte to port
    ret

.global orizon_asm_halt
orizon_asm_halt:
    hlt
    jmp orizon_asm_halt

.global orizon_asm_enable_interrupts
orizon_asm_enable_interrupts:
    sti
    ret

.global orizon_asm_disable_interrupts
orizon_asm_disable_interrupts:
    cli
    ret
*/

// ============================================================================
// Kernel test and demonstration functions
// ============================================================================

//export orizon_kernel_test_memory
func orizon_kernel_test_memory() {
	fmt.Println("Testing kernel memory management...")

	// Test page allocation
	page1 := KernelAllocatePage()
	page2 := KernelAllocatePage()

	fmt.Printf("Allocated pages: 0x%x, 0x%x\n", page1, page2)

	// Test memory info
	total, free, used := KernelGetMemoryInfo()
	fmt.Printf("Memory: %d KB total, %d KB free, %d KB used\n",
		total/1024, free/1024, used/1024)

	// Free pages
	KernelFreePage(page1)
	KernelFreePage(page2)

	fmt.Println("Memory test completed")
}

//export orizon_kernel_test_filesystem
func orizon_kernel_test_filesystem() {
	fmt.Println("Testing kernel file system...")

	// Create a test file
	file := KernelCreateFile("/test.txt", 0o644)
	if file == 0 {
		fmt.Println("Failed to create test file")
		return
	}

	// Write to file
	testData := []byte("Hello from Orizon kernel!")
	written := KernelWriteFile(file, testData)
	fmt.Printf("Wrote %d bytes to file\n", written)

	// Close file
	KernelCloseFile(file)

	// Reopen and read
	file = KernelOpenFile("/test.txt", 0)
	if file == 0 {
		fmt.Println("Failed to open test file")
		return
	}

	readBuffer := make([]byte, 100)
	readBytes := KernelReadFile(file, readBuffer)
	fmt.Printf("Read %d bytes: %s\n", readBytes, string(readBuffer[:readBytes]))

	KernelCloseFile(file)

	fmt.Println("File system test completed")
}

//export orizon_kernel_demo_os
func orizon_kernel_demo_os() {
	fmt.Println("=== Orizon OS Demo ===")

	// Initialize kernel
	if !orizon_kernel_bootstrap() {
		fmt.Println("Kernel bootstrap failed!")
		return
	}

	fmt.Println("Kernel initialized successfully!")

	// Test memory management
	orizon_kernel_test_memory()

	// Test file system
	orizon_kernel_test_filesystem()

	// Create a test process
	testProcessName := "test_process"
	pid := orizon_kernel_create_process(
		uintptr(unsafe.Pointer(unsafe.StringData(testProcessName))),
		uintptr(len(testProcessName)),
		0x400000, // Entry point
		64*1024,  // Stack size
	)

	if pid != 0 {
		fmt.Printf("Created test process with PID %d\n", pid)
		fmt.Printf("Current PID: %d\n", orizon_kernel_get_current_pid())
	}

	// Test console output
	message := "Hello from Orizon OS!\n"
	orizon_kernel_write_console(
		uintptr(unsafe.Pointer(unsafe.StringData(message))),
		uintptr(len(message)),
	)

	fmt.Printf("System uptime: %d ms\n", orizon_kernel_get_uptime())

	fmt.Println("=== Demo completed ===")
}
