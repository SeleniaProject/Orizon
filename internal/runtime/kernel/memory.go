// Package kernel provides low-level kernel functionality for OS development in Orizon.
// This package implements essential kernel services that can be called from Orizon code
// to enable operating system development.
package kernel

import (
	"fmt"
	"unsafe"
)

// MemoryRegion represents a contiguous memory region
type MemoryRegion struct {
	Base     uintptr // Physical base address
	Size     uintptr // Size in bytes
	Type     MemoryType
	Flags    MemoryFlags
	Reserved bool
}

// MemoryType represents different types of memory
type MemoryType uint32

const (
	MemoryTypeRAM MemoryType = iota
	MemoryTypeReserved
	MemoryTypeACPI
	MemoryTypeNVS
	MemoryTypeUnusable
	MemoryTypePersistent
)

// MemoryFlags represents memory region flags
type MemoryFlags uint32

const (
	MemoryFlagReadable MemoryFlags = 1 << iota
	MemoryFlagWritable
	MemoryFlagExecutable
	MemoryFlagCacheable
	MemoryFlagWriteThrough
	MemoryFlagWriteCombining
)

// PageSize represents common page sizes
const (
	PageSize4KB     = 4096
	PageSize2MB     = 2 * 1024 * 1024
	PageSize1GB     = 1024 * 1024 * 1024
	DefaultPageSize = PageSize4KB
)

// PhysicalMemoryManager manages physical memory allocation
type PhysicalMemoryManager struct {
	regions       []MemoryRegion
	freePagesList []uintptr
	usedPages     map[uintptr]bool
	totalPages    uint64
	freePages     uint64
}

// NewPhysicalMemoryManager creates a new physical memory manager
func NewPhysicalMemoryManager() *PhysicalMemoryManager {
	return &PhysicalMemoryManager{
		regions:       make([]MemoryRegion, 0),
		freePagesList: make([]uintptr, 0),
		usedPages:     make(map[uintptr]bool),
	}
}

// AddRegion adds a memory region to the manager
func (pmm *PhysicalMemoryManager) AddRegion(base, size uintptr, memType MemoryType) error {
	if size == 0 {
		return fmt.Errorf("invalid memory region size: %d", size)
	}

	region := MemoryRegion{
		Base:  base,
		Size:  size,
		Type:  memType,
		Flags: MemoryFlagReadable | MemoryFlagWritable,
	}

	pmm.regions = append(pmm.regions, region)

	// If it's RAM, add pages to free list
	if memType == MemoryTypeRAM {
		pageCount := size / DefaultPageSize
		for i := uintptr(0); i < pageCount; i++ {
			pageAddr := base + i*DefaultPageSize
			pmm.freePagesList = append(pmm.freePagesList, pageAddr)
		}
		pmm.totalPages += uint64(pageCount)
		pmm.freePages += uint64(pageCount)
	}

	return nil
}

// AllocatePage allocates a single physical page
func (pmm *PhysicalMemoryManager) AllocatePage() (uintptr, error) {
	if len(pmm.freePagesList) == 0 {
		return 0, fmt.Errorf("out of physical memory")
	}

	// Take the first free page
	pageAddr := pmm.freePagesList[0]
	pmm.freePagesList = pmm.freePagesList[1:]
	pmm.usedPages[pageAddr] = true
	pmm.freePages--

	return pageAddr, nil
}

// FreePage frees a physical page
func (pmm *PhysicalMemoryManager) FreePage(pageAddr uintptr) error {
	if !pmm.usedPages[pageAddr] {
		return fmt.Errorf("page not allocated: 0x%x", pageAddr)
	}

	delete(pmm.usedPages, pageAddr)
	pmm.freePagesList = append(pmm.freePagesList, pageAddr)
	pmm.freePages++

	return nil
}

// GetMemoryInfo returns memory statistics
func (pmm *PhysicalMemoryManager) GetMemoryInfo() (total, free, used uint64) {
	return pmm.totalPages * DefaultPageSize,
		pmm.freePages * DefaultPageSize,
		(pmm.totalPages - pmm.freePages) * DefaultPageSize
}

// ============================================================================
// Hardware abstraction for port I/O
// ============================================================================

// IOPort represents a hardware I/O port
type IOPort struct {
	port uint16
}

// NewIOPort creates a new I/O port
func NewIOPort(port uint16) *IOPort {
	return &IOPort{port: port}
}

// ReadByte reads a byte from the I/O port
func (p *IOPort) ReadByte() uint8 {
	// Platform-specific implementation
	// On x86_64, this would use the `in` instruction
	// For now, simulate with syscall (this is a placeholder)
	return 0
}

// WriteByte writes a byte to the I/O port
func (p *IOPort) WriteByte(value uint8) {
	// Platform-specific implementation
	// On x86_64, this would use the `out` instruction
	// For now, simulate with syscall (this is a placeholder)
}

// ReadWord reads a 16-bit word from the I/O port
func (p *IOPort) ReadWord() uint16 {
	return 0 // Placeholder
}

// WriteWord writes a 16-bit word to the I/O port
func (p *IOPort) WriteWord(value uint16) {
	// Placeholder
}

// ReadDWord reads a 32-bit double word from the I/O port
func (p *IOPort) ReadDWord() uint32 {
	return 0 // Placeholder
}

// WriteDWord writes a 32-bit double word to the I/O port
func (p *IOPort) WriteDWord(value uint32) {
	// Placeholder
}

// ============================================================================
// Memory-mapped I/O
// ============================================================================

// ReadVolatile performs a volatile memory read
func ReadVolatile8(addr uintptr) uint8 {
	return *(*uint8)(unsafe.Pointer(addr))
}

// WriteVolatile performs a volatile memory write
func WriteVolatile8(addr uintptr, value uint8) {
	*(*uint8)(unsafe.Pointer(addr)) = value
}

// ReadVolatile16 performs a volatile 16-bit memory read
func ReadVolatile16(addr uintptr) uint16 {
	return *(*uint16)(unsafe.Pointer(addr))
}

// WriteVolatile16 performs a volatile 16-bit memory write
func WriteVolatile16(addr uintptr, value uint16) {
	*(*uint16)(unsafe.Pointer(addr)) = value
}

// ReadVolatile32 performs a volatile 32-bit memory read
func ReadVolatile32(addr uintptr) uint32 {
	return *(*uint32)(unsafe.Pointer(addr))
}

// WriteVolatile32 performs a volatile 32-bit memory write
func WriteVolatile32(addr uintptr, value uint32) {
	*(*uint32)(unsafe.Pointer(addr)) = value
}

// ReadVolatile64 performs a volatile 64-bit memory read
func ReadVolatile64(addr uintptr) uint64 {
	return *(*uint64)(unsafe.Pointer(addr))
}

// WriteVolatile64 performs a volatile 64-bit memory write
func WriteVolatile64(addr uintptr, value uint64) {
	*(*uint64)(unsafe.Pointer(addr)) = value
}

// ============================================================================
// CPU control functions
// ============================================================================

// HaltCPU halts the CPU until the next interrupt
func HaltCPU() {
	// Platform-specific implementation
	// On x86_64, this would use the `hlt` instruction
	// For now, use a tight loop as placeholder
	for {
		// Busy wait - in real implementation, this would be `hlt`
	}
}

// DisableInterrupts disables CPU interrupts
func DisableInterrupts() {
	// Platform-specific implementation
	// On x86_64, this would use the `cli` instruction
}

// EnableInterrupts enables CPU interrupts
func EnableInterrupts() {
	// Platform-specific implementation
	// On x86_64, this would use the `sti` instruction
}

// GetInterruptFlag returns the current interrupt flag state
func GetInterruptFlag() bool {
	// Platform-specific implementation
	return true // Placeholder
}

// ============================================================================
// Address space management
// ============================================================================

// VirtualAddressSpace represents a virtual address space
type VirtualAddressSpace struct {
	pageTable uintptr // Pointer to page table root
	pid       uint32  // Process ID
}

// NewVirtualAddressSpace creates a new virtual address space
func NewVirtualAddressSpace(pid uint32) *VirtualAddressSpace {
	return &VirtualAddressSpace{
		pid: pid,
	}
}

// MapPage maps a virtual page to a physical page
func (vas *VirtualAddressSpace) MapPage(virtualAddr, physicalAddr uintptr, flags MemoryFlags) error {
	// Platform-specific page table manipulation
	// This would involve setting up page directory/page table entries
	return nil // Placeholder
}

// UnmapPage unmaps a virtual page
func (vas *VirtualAddressSpace) UnmapPage(virtualAddr uintptr) error {
	// Platform-specific page table manipulation
	return nil // Placeholder
}

// ============================================================================
// Global kernel instance
// ============================================================================

// GlobalKernel provides access to kernel services
var GlobalKernel *KernelInstance

// KernelInstance represents the main kernel instance
type KernelInstance struct {
	PhysicalMemory *PhysicalMemoryManager
	initialized    bool
}

// InitializeKernel initializes the kernel subsystems
func InitializeKernel() error {
	if GlobalKernel != nil && GlobalKernel.initialized {
		return fmt.Errorf("kernel already initialized")
	}

	GlobalKernel = &KernelInstance{
		PhysicalMemory: NewPhysicalMemoryManager(),
	}

	// Initialize basic memory regions (placeholder values)
	// In a real OS, this would come from bootloader
	err := GlobalKernel.PhysicalMemory.AddRegion(
		0x100000,  // 1MB
		0x1000000, // 16MB
		MemoryTypeRAM,
	)
	if err != nil {
		return fmt.Errorf("failed to add memory region: %w", err)
	}

	GlobalKernel.initialized = true
	return nil
}

// ============================================================================
// Kernel API functions (callable from Orizon)
// ============================================================================

// These functions provide the interface between Orizon code and kernel services

// KernelAllocatePage allocates a physical page
func KernelAllocatePage() uintptr {
	if GlobalKernel == nil {
		return 0
	}

	addr, err := GlobalKernel.PhysicalMemory.AllocatePage()
	if err != nil {
		return 0
	}

	return addr
}

// KernelFreePage frees a physical page
func KernelFreePage(addr uintptr) bool {
	if GlobalKernel == nil {
		return false
	}

	err := GlobalKernel.PhysicalMemory.FreePage(addr)
	return err == nil
}

// KernelGetMemoryInfo returns memory information
func KernelGetMemoryInfo() (total, free, used uint64) {
	if GlobalKernel == nil {
		return 0, 0, 0
	}

	return GlobalKernel.PhysicalMemory.GetMemoryInfo()
}
