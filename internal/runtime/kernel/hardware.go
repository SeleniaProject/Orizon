// Package kernel provides hardware abstraction layer for OS development
package kernel

import (
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// Hardware abstraction layer
// ============================================================================

// PlatformArchitecture represents the target architecture
type PlatformArchitecture uint8

const (
	ArchX86_64 PlatformArchitecture = iota
	ArchARM64
	ArchRISCV64
)

// HardwareInfo contains information about the hardware platform
type HardwareInfo struct {
	Architecture PlatformArchitecture
	CPUCount     uint32
	MemorySize   uint64
	CPUFeatures  CPUFeatures
}

// CPUFeatures represents supported CPU features
type CPUFeatures struct {
	SSE    bool
	SSE2   bool
	SSE3   bool
	SSSE3  bool
	SSE41  bool
	SSE42  bool
	AVX    bool
	AVX2   bool
	AVX512 bool
	AES    bool
	RDRAND bool
	RDSEED bool
}

// ============================================================================
// Timer and clock management
// ============================================================================

// TimerManager handles system timing
type TimerManager struct {
	tickRate    uint64    // Ticks per second
	tickCount   uint64    // Total ticks since boot
	bootTime    time.Time // Boot time
	mutex       sync.RWMutex
	initialized bool
}

// GlobalTimerManager provides global timer access
var GlobalTimerManager *TimerManager

// InitializeTimers initializes the timer system
func InitializeTimers() error {
	if GlobalTimerManager != nil && GlobalTimerManager.initialized {
		return fmt.Errorf("timers already initialized")
	}

	GlobalTimerManager = &TimerManager{
		tickRate: 1000, // 1000 Hz (1ms resolution)
		bootTime: time.Now(),
	}

	// Set up timer interrupt (IRQ 0)
	if GlobalInterruptManager != nil {
		GlobalInterruptManager.SetHandler(32, TimerInterruptHandler) // IRQ 0 mapped to interrupt 32
	}

	GlobalTimerManager.initialized = true
	return nil
}

// TimerInterruptHandler handles timer interrupts
func TimerInterruptHandler(ctx *InterruptContext) {
	if GlobalTimerManager != nil {
		GlobalTimerManager.mutex.Lock()
		GlobalTimerManager.tickCount++
		GlobalTimerManager.mutex.Unlock()

		// Schedule next process (simple round-robin)
		if GlobalProcessManager != nil {
			GlobalProcessManager.Schedule()
		}
	}
}

// GetTicks returns the current tick count
func (tm *TimerManager) GetTicks() uint64 {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	return tm.tickCount
}

// GetUptime returns the system uptime in milliseconds
func (tm *TimerManager) GetUptime() uint64 {
	return tm.GetTicks() * 1000 / tm.tickRate
}

// Sleep sleeps for the specified number of milliseconds
func (tm *TimerManager) Sleep(ms uint64) {
	targetTicks := tm.GetTicks() + (ms * tm.tickRate / 1000)
	for tm.GetTicks() < targetTicks {
		// Yield to other processes
		if GlobalProcessManager != nil {
			GlobalProcessManager.Yield()
		}
	}
}

// ============================================================================
// Process management
// ============================================================================

// ProcessState represents the state of a process
type ProcessState uint8

const (
	ProcessRunning ProcessState = iota
	ProcessReady
	ProcessBlocked
	ProcessTerminated
)

// Process represents a running process
type Process struct {
	PID       uint32
	ParentPID uint32
	State     ProcessState
	Priority  uint8
	Context   *InterruptContext
	VirtualAS *VirtualAddressSpace
	StackBase uintptr
	StackSize uintptr
	HeapBase  uintptr
	HeapSize  uintptr
	Name      string
	StartTime uint64
	CPUTime   uint64
}

// ProcessManager manages processes and scheduling
type ProcessManager struct {
	processes   map[uint32]*Process
	readyQueue  []*Process
	currentPID  uint32
	nextPID     uint32
	mutex       sync.RWMutex
	initialized bool
}

// GlobalProcessManager provides global process management
var GlobalProcessManager *ProcessManager

// InitializeProcessManager initializes the process management system
func InitializeProcessManager() error {
	if GlobalProcessManager != nil && GlobalProcessManager.initialized {
		return fmt.Errorf("process manager already initialized")
	}

	GlobalProcessManager = &ProcessManager{
		processes:  make(map[uint32]*Process),
		readyQueue: make([]*Process, 0),
		nextPID:    1,
	}

	// Create kernel process (PID 0)
	kernelProcess := &Process{
		PID:       0,
		ParentPID: 0,
		State:     ProcessRunning,
		Priority:  255, // Highest priority
		Name:      "kernel",
		StartTime: GlobalTimerManager.GetTicks(),
	}

	GlobalProcessManager.processes[0] = kernelProcess
	GlobalProcessManager.currentPID = 0
	GlobalProcessManager.initialized = true

	return nil
}

// CreateProcess creates a new process
func (pm *ProcessManager) CreateProcess(name string, entryPoint uintptr, stackSize uintptr) (*Process, error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Allocate PID
	pid := pm.nextPID
	pm.nextPID++

	// Allocate virtual address space
	vas := NewVirtualAddressSpace(pid)

	// Allocate stack
	stackPages := (stackSize + DefaultPageSize - 1) / DefaultPageSize
	stackBase := uintptr(0x7FFFFF000000) // High memory for stack

	for i := uintptr(0); i < stackPages; i++ {
		physPage, err := GlobalKernel.PhysicalMemory.AllocatePage()
		if err != nil {
			return nil, fmt.Errorf("failed to allocate stack page: %w", err)
		}

		virtAddr := stackBase - (i+1)*DefaultPageSize
		err = vas.MapPage(virtAddr, physPage, MemoryFlagReadable|MemoryFlagWritable)
		if err != nil {
			return nil, fmt.Errorf("failed to map stack page: %w", err)
		}
	}

	// Create initial context
	context := &InterruptContext{
		RIP:    uint64(entryPoint),
		RSP:    uint64(stackBase), // Stack grows downward
		CS:     0x08,              // Kernel code segment
		DS:     0x10,              // Kernel data segment
		RFLAGS: 0x202,             // Interrupts enabled
	}

	// Create process
	process := &Process{
		PID:       pid,
		ParentPID: pm.currentPID,
		State:     ProcessReady,
		Priority:  128, // Normal priority
		Context:   context,
		VirtualAS: vas,
		StackBase: stackBase,
		StackSize: stackSize,
		HeapBase:  0x400000, // 4MB base for heap
		HeapSize:  0,
		Name:      name,
		StartTime: GlobalTimerManager.GetTicks(),
	}

	pm.processes[pid] = process
	pm.readyQueue = append(pm.readyQueue, process)

	return process, nil
}

// Schedule performs process scheduling (simple round-robin)
func (pm *ProcessManager) Schedule() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if len(pm.readyQueue) == 0 {
		return
	}

	// Get current process
	currentProcess := pm.processes[pm.currentPID]
	if currentProcess != nil && currentProcess.State == ProcessRunning {
		currentProcess.State = ProcessReady
		pm.readyQueue = append(pm.readyQueue, currentProcess)
	}

	// Get next process
	nextProcess := pm.readyQueue[0]
	pm.readyQueue = pm.readyQueue[1:]

	nextProcess.State = ProcessRunning
	pm.currentPID = nextProcess.PID

	// Context switch (this would involve saving/restoring CPU state)
	pm.contextSwitch(currentProcess, nextProcess)
}

// contextSwitch performs a context switch between processes
func (pm *ProcessManager) contextSwitch(from, to *Process) {
	// In a real kernel, this would:
	// 1. Save current CPU state to 'from' process
	// 2. Switch page tables to 'to' process address space
	// 3. Restore CPU state from 'to' process
	// 4. Jump to the new process

	fmt.Printf("Context switch: %s (PID %d) -> %s (PID %d)\n",
		from.Name, from.PID, to.Name, to.PID)
}

// Yield yields the CPU to other processes
func (pm *ProcessManager) Yield() {
	pm.Schedule()
}

// KillProcess terminates a process
func (pm *ProcessManager) KillProcess(pid uint32) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	process, exists := pm.processes[pid]
	if !exists {
		return fmt.Errorf("process %d not found", pid)
	}

	if pid == 0 {
		return fmt.Errorf("cannot kill kernel process")
	}

	process.State = ProcessTerminated

	// Free memory
	// This would involve unmapping all pages and freeing physical memory

	// Remove from ready queue
	for i, p := range pm.readyQueue {
		if p.PID == pid {
			pm.readyQueue = append(pm.readyQueue[:i], pm.readyQueue[i+1:]...)
			break
		}
	}

	delete(pm.processes, pid)

	return nil
}

// ============================================================================
// Device management
// ============================================================================

// DeviceType represents different types of devices
type DeviceType uint8

const (
	DeviceTypeCharacter DeviceType = iota
	DeviceTypeBlock
	DeviceTypeNetwork
	DeviceTypeInput
)

// Device represents a hardware device
type Device struct {
	Name     string
	Type     DeviceType
	MajorNum uint32
	MinorNum uint32
	BaseAddr uintptr
	IRQ      uint8
	Driver   DeviceDriver
}

// DeviceDriver interface for device drivers
type DeviceDriver interface {
	Initialize() error
	Read(offset uint64, buffer []byte) (int, error)
	Write(offset uint64, buffer []byte) (int, error)
	Control(cmd uint32, arg uintptr) error
	Cleanup() error
}

// DeviceManager manages hardware devices
type DeviceManager struct {
	devices     map[string]*Device
	drivers     map[DeviceType][]DeviceDriver
	mutex       sync.RWMutex
	initialized bool
}

// GlobalDeviceManager provides global device management
var GlobalDeviceManager *DeviceManager

// InitializeDeviceManager initializes the device management system
func InitializeDeviceManager() error {
	if GlobalDeviceManager != nil && GlobalDeviceManager.initialized {
		return fmt.Errorf("device manager already initialized")
	}

	GlobalDeviceManager = &DeviceManager{
		devices: make(map[string]*Device),
		drivers: make(map[DeviceType][]DeviceDriver),
	}

	// Register built-in drivers
	err := GlobalDeviceManager.registerBuiltinDrivers()
	if err != nil {
		return fmt.Errorf("failed to register builtin drivers: %w", err)
	}

	GlobalDeviceManager.initialized = true
	return nil
}

// registerBuiltinDrivers registers built-in device drivers
func (dm *DeviceManager) registerBuiltinDrivers() error {
	// Register console driver
	consoleDriver := &ConsoleDriver{}
	dm.RegisterDriver(DeviceTypeCharacter, consoleDriver)

	// Register null device driver
	nullDriver := &NullDriver{}
	dm.RegisterDriver(DeviceTypeCharacter, nullDriver)

	return nil
}

// RegisterDriver registers a device driver
func (dm *DeviceManager) RegisterDriver(deviceType DeviceType, driver DeviceDriver) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	dm.drivers[deviceType] = append(dm.drivers[deviceType], driver)
}

// RegisterDevice registers a hardware device
func (dm *DeviceManager) RegisterDevice(device *Device) error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	if _, exists := dm.devices[device.Name]; exists {
		return fmt.Errorf("device %s already exists", device.Name)
	}

	// Initialize the device driver
	if device.Driver != nil {
		err := device.Driver.Initialize()
		if err != nil {
			return fmt.Errorf("failed to initialize device %s: %w", device.Name, err)
		}
	}

	dm.devices[device.Name] = device
	return nil
}

// ============================================================================
// Built-in device drivers
// ============================================================================

// ConsoleDriver implements a simple console driver
type ConsoleDriver struct {
	mutex sync.Mutex
}

// Initialize initializes the console driver
func (cd *ConsoleDriver) Initialize() error {
	return nil
}

// Read reads from the console (keyboard input)
func (cd *ConsoleDriver) Read(offset uint64, buffer []byte) (int, error) {
	// Placeholder: in real implementation, this would read from keyboard buffer
	return 0, nil
}

// Write writes to the console (screen output)
func (cd *ConsoleDriver) Write(offset uint64, buffer []byte) (int, error) {
	cd.mutex.Lock()
	defer cd.mutex.Unlock()

	// Simple VGA text mode output
	const VGABuffer = 0xB8000
	static_cursor := uint16(0)

	for _, b := range buffer {
		if b == '\n' {
			// Move to next line
			static_cursor = (static_cursor/160 + 1) * 160
		} else {
			// Write character and attribute
			WriteVolatile8(VGABuffer+uintptr(static_cursor), b)
			WriteVolatile8(VGABuffer+uintptr(static_cursor)+1, 0x07) // White on black
			static_cursor += 2
		}

		// Wrap around if we reach the end of screen
		if static_cursor >= 80*25*2 {
			static_cursor = 0
		}
	}

	return len(buffer), nil
}

// Control handles device control operations
func (cd *ConsoleDriver) Control(cmd uint32, arg uintptr) error {
	return nil
}

// Cleanup cleans up the console driver
func (cd *ConsoleDriver) Cleanup() error {
	return nil
}

// NullDriver implements a null device driver (/dev/null equivalent)
type NullDriver struct{}

// Initialize initializes the null driver
func (nd *NullDriver) Initialize() error {
	return nil
}

// Read always returns EOF
func (nd *NullDriver) Read(offset uint64, buffer []byte) (int, error) {
	return 0, nil // EOF
}

// Write discards all data
func (nd *NullDriver) Write(offset uint64, buffer []byte) (int, error) {
	return len(buffer), nil // Pretend we wrote everything
}

// Control does nothing
func (nd *NullDriver) Control(cmd uint32, arg uintptr) error {
	return nil
}

// Cleanup does nothing
func (nd *NullDriver) Cleanup() error {
	return nil
}

// ============================================================================
// Kernel API functions for Orizon
// ============================================================================

// KernelCreateProcess creates a new process
func KernelCreateProcess(name string, entryPoint uintptr, stackSize uintptr) uint32 {
	if GlobalProcessManager == nil {
		return 0
	}

	process, err := GlobalProcessManager.CreateProcess(name, entryPoint, stackSize)
	if err != nil {
		fmt.Printf("Failed to create process: %v\n", err)
		return 0
	}

	return process.PID
}

// KernelKillProcess kills a process
func KernelKillProcess(pid uint32) bool {
	if GlobalProcessManager == nil {
		return false
	}

	err := GlobalProcessManager.KillProcess(pid)
	return err == nil
}

// KernelGetCurrentPID returns the current process ID
func KernelGetCurrentPID() uint32 {
	if GlobalProcessManager == nil {
		return 0
	}

	GlobalProcessManager.mutex.RLock()
	defer GlobalProcessManager.mutex.RUnlock()
	return GlobalProcessManager.currentPID
}

// KernelYield yields the CPU to other processes
func KernelYield() {
	if GlobalProcessManager != nil {
		GlobalProcessManager.Yield()
	}
}

// KernelSleep sleeps for the specified number of milliseconds
func KernelSleep(ms uint64) {
	if GlobalTimerManager != nil {
		GlobalTimerManager.Sleep(ms)
	}
}

// KernelGetUptime returns the system uptime in milliseconds
func KernelGetUptime() uint64 {
	if GlobalTimerManager != nil {
		return GlobalTimerManager.GetUptime()
	}
	return 0
}

// KernelWriteConsole writes data to the console
func KernelWriteConsole(data []byte) int {
	if GlobalDeviceManager == nil {
		return 0
	}

	console, exists := GlobalDeviceManager.devices["console"]
	if !exists || console.Driver == nil {
		return 0
	}

	n, err := console.Driver.Write(0, data)
	if err != nil {
		return 0
	}

	return n
}
