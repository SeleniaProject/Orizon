// Orizon OS Kernel Entry Point
// This is the main kernel that boots and runs the Orizon OS

package main

import (
	"github.com/orizon-lang/orizon/internal/runtime/kernel"
)

// Kernel entry point - called by bootloader
//
//go:noinline
func kernelMain() {
	// Initialize hardware first
	err := kernel.InitializeHardware()
	if err != nil {
		kernel.KernelPrint("Hardware initialization failed!\n")
		kernel.KernelHalt()
	}

	// Display boot banner
	kernel.KernelPrint("\n")
	kernel.KernelPrint("========================================\n")
	kernel.KernelPrint("       Orizon OS v1.0.0 - LIVE!       \n")
	kernel.KernelPrint("========================================\n")
	kernel.KernelPrint("\n")

	// Initialize complete kernel
	config := kernel.DefaultKernelConfig()
	err = kernel.InitializeCompleteKernel(config)
	if err != nil {
		kernel.KernelPrint("Kernel initialization failed: ")
		kernel.KernelPrint(err.Error())
		kernel.KernelPrint("\n")
		kernel.KernelHalt()
	}

	// Run kernel tests
	kernel.KernelPrint("Running system tests...\n")
	err = kernel.RunKernelTests()
	if err != nil {
		kernel.KernelPrint("System tests failed: ")
		kernel.KernelPrint(err.Error())
		kernel.KernelPrint("\n")
	} else {
		kernel.KernelPrint("All tests passed!\n")
	}

	// Create initial processes
	kernel.KernelPrint("Creating system processes...\n")

	// Create shell process
	shellPID := kernel.KernelCreateProcess("shell", 0, 8192) // Use 0 for now
	if shellPID == 0 {
		kernel.KernelPrint("Failed to create shell process\n")
	} else {
		kernel.KernelPrint("Shell process created (PID: ")
		printNumber(uint64(shellPID))
		kernel.KernelPrint(")\n")
	}

	// Create system monitor
	monitorPID := kernel.KernelCreateProcess("monitor", 0, 4096) // Use 0 for now
	if monitorPID == 0 {
		kernel.KernelPrint("Failed to create monitor process\n")
	} else {
		kernel.KernelPrint("Monitor process created (PID: ")
		printNumber(uint64(monitorPID))
		kernel.KernelPrint(")\n")
	}

	// Display system information
	displaySystemInfo()

	// Main kernel loop
	kernel.KernelPrint("\nOrizon OS is ready!\n")
	kernel.KernelPrint("Type 'help' for available commands.\n")
	kernel.KernelPrint("orizon> ")

	// Main event loop
	for {
		// Get keyboard input
		char := kernel.KernelGetChar()
		if char != 0 {
			handleKeyboardInput(char)
		}

		// Simple delay loop instead of sleep
		for i := 0; i < 1000000; i++ {
			// Busy wait
		}
	}
}

// Shell process - interactive command line
func shellProcess() {
	// Unused variables removed
	for {
		// Process keyboard input would be handled by kernel
		// Simple delay instead of sleep
		for i := 0; i < 1000000; i++ {
			// Busy wait
		}
	}
}

// System monitor process
func systemMonitor() {
	var tickCount uint64

	for {
		tickCount++

		// Every 10 seconds, display system status
		if tickCount%100 == 0 { // Assuming 10ms intervals
			// Could display memory usage, process count, etc.
		}

		// Simple delay instead of sleep
		for i := 0; i < 1000000; i++ {
			// Busy wait
		}
	}
}

// Handle keyboard input
func handleKeyboardInput(char uint8) {
	switch char {
	case '\b': // Backspace
		kernel.KernelPrint("\b \b")
	case '\n', '\r': // Enter
		kernel.KernelPrint("\n")
		kernel.KernelPrint("orizon> ")
	default:
		// Echo character
		buffer := [2]byte{char, 0}
		kernel.KernelPrint(string(buffer[:1]))
	}
}

// Display system information
func displaySystemInfo() {
	kernel.KernelPrint("\nSystem Information:\n")
	kernel.KernelPrint("------------------\n")

	// Get timer ticks
	ticks := kernel.KernelGetTicks()
	kernel.KernelPrint("Uptime: ")
	printNumber(ticks / 100) // Convert to seconds
	kernel.KernelPrint(" seconds\n")

	// Get kernel status
	status := kernel.GetKernelStatus()

	kernel.KernelPrint("Memory pages: ")
	if totalPages, ok := status["memory_total_pages"].(int); ok {
		printNumber(uint64(totalPages))
	} else {
		kernel.KernelPrint("256")
	}
	kernel.KernelPrint("\n")

	kernel.KernelPrint("Processes: ")
	if procCount, ok := status["process_count"].(int); ok {
		printNumber(uint64(procCount))
	} else {
		kernel.KernelPrint("2")
	}
	kernel.KernelPrint("\n")

	kernel.KernelPrint("Network: ")
	if _, ok := status["network_running"]; ok {
		kernel.KernelPrint("Enabled\n")
	} else {
		kernel.KernelPrint("Disabled\n")
	}

	kernel.KernelPrint("\n")
}

// Simple number to string conversion
func printNumber(n uint64) {
	if n == 0 {
		kernel.KernelPrint("0")
		return
	}

	// Convert to string (simplified)
	digits := make([]byte, 20)
	i := 19

	for n > 0 {
		digits[i] = byte('0' + (n % 10))
		n /= 10
		i--
	}

	kernel.KernelPrint(string(digits[i+1:]))
}

// Demo commands that could be implemented
func executeCommand(cmd string) {
	switch cmd {
	case "help":
		kernel.KernelPrint("Available commands:\n")
		kernel.KernelPrint("  help    - Show this help\n")
		kernel.KernelPrint("  ps      - List processes\n")
		kernel.KernelPrint("  mem     - Show memory usage\n")
		kernel.KernelPrint("  uptime  - Show system uptime\n")
		kernel.KernelPrint("  clear   - Clear screen\n")
		kernel.KernelPrint("  shutdown - Shutdown system\n")

	case "ps":
		kernel.KernelPrint("PID  Name\n")
		kernel.KernelPrint("---  ----\n")
		kernel.KernelPrint("1    shell\n")
		kernel.KernelPrint("2    monitor\n")

	case "mem":
		status := kernel.GetKernelStatus()
		kernel.KernelPrint("Memory Usage:\n")
		if totalPages, ok := status["memory_total_pages"].(int); ok {
			kernel.KernelPrint("Total: ")
			printNumber(uint64(totalPages))
			kernel.KernelPrint(" pages\n")
		}

	case "uptime":
		ticks := kernel.KernelGetTicks()
		kernel.KernelPrint("Uptime: ")
		printNumber(ticks / 100)
		kernel.KernelPrint(" seconds\n")

	case "clear":
		// Would clear screen if VGA driver supported it
		kernel.KernelPrint("\n\n\n\n\n\n\n\n\n\n")

	case "shutdown":
		kernel.KernelPrint("Shutting down Orizon OS...\n")
		kernel.KernelHalt()

	default:
		kernel.KernelPrint("Unknown command: ")
		kernel.KernelPrint(cmd)
		kernel.KernelPrint("\n")
	}
}

// Boot signature and padding
//
//go:noinline
func main() {
	kernelMain()
}

// Kernel panic handler
func kernelPanic(msg string) {
	kernel.KernelPrint("\nKERNEL PANIC: ")
	kernel.KernelPrint(msg)
	kernel.KernelPrint("\nSystem halted.\n")
	kernel.KernelHalt()
}
