// High-performance device driver framework for Orizon OS
// Provides zero-overhead abstractions for hardware access
package drivers

import (
	"fmt"
	"sync"
)

// DeviceType represents different types of hardware devices
type DeviceType uint32

const (
	DeviceTypeUnknown DeviceType = iota
	DeviceTypeStorage
	DeviceTypeNetwork
	DeviceTypeGraphics
	DeviceTypeAudio
	DeviceTypeInput
	DeviceTypeUSB
	DeviceTypePCI
	DeviceTypeACPI
	DeviceTypeInterrupt
	DeviceTypeTimer
	DeviceTypeDMA
)

// Device represents a hardware device
type Device interface {
	Initialize() error
	Start() error
	Stop() error
	Reset() error
	GetInfo() DeviceInfo
	HandleInterrupt(vector uint8) bool
	Read(offset uintptr, size uintptr) ([]byte, error)
	Write(offset uintptr, data []byte) error
}

// DeviceInfo contains device information
type DeviceInfo struct {
	Name         string
	Type         DeviceType
	VendorID     uint16
	DeviceID     uint16
	SubsystemID  uint16
	Class        uint8
	SubClass     uint8
	ProgIF       uint8
	Revision     uint8
	BusNumber    uint8
	DeviceNumber uint8
	Function     uint8
	IRQ          uint8
	BaseAddress  []uintptr
	IOPorts      []IOPortRange
	MemoryMap    []MemoryMapRegion
}

// IOPortRange represents an I/O port range
type IOPortRange struct {
	Start uint16
	End   uint16
	Type  IOPortType
}

type IOPortType uint8

const (
	IOPortTypeRead IOPortType = iota
	IOPortTypeWrite
	IOPortTypeReadWrite
)

// MemoryMapRegion represents a memory-mapped I/O region
type MemoryMapRegion struct {
	PhysicalAddress uintptr
	VirtualAddress  uintptr
	Size            uintptr
	Flags           uint32 // Memory flags
}

// PCIDevice represents a PCI device
type PCIDevice struct {
	info        DeviceInfo
	configSpace *PCIConfigSpace
	driver      DeviceDriver
	mutex       sync.RWMutex
	initialized bool
	running     bool
}

// PCIConfigSpace represents PCI configuration space
type PCIConfigSpace struct {
	VendorID          uint16
	DeviceID          uint16
	Command           uint16
	Status            uint16
	RevisionID        uint8
	ProgIF            uint8
	SubClass          uint8
	ClassCode         uint8
	CacheLineSize     uint8
	LatencyTimer      uint8
	HeaderType        uint8
	BIST              uint8
	BAR               [6]uint32
	SubsystemVendorID uint16
	SubsystemID       uint16
	ExpansionROMBase  uint32
	CapabilitiesPtr   uint8
	InterruptLine     uint8
	InterruptPin      uint8
	MinGrant          uint8
	MaxLatency        uint8
}

// DeviceDriver interface for device-specific drivers
type DeviceDriver interface {
	Probe(device *PCIDevice) bool
	Attach(device *PCIDevice) error
	Detach(device *PCIDevice) error
	Suspend(device *PCIDevice) error
	Resume(device *PCIDevice) error
	HandleIRQ(device *PCIDevice, vector uint8) bool
}

// DeviceManager manages all hardware devices
type DeviceManager struct {
	devices     map[string]*PCIDevice
	drivers     map[DeviceType][]DeviceDriver
	irqHandlers map[uint8][]*PCIDevice
	mutex       sync.RWMutex
}

// Global device manager instance
var globalDeviceManager *DeviceManager

// Initialize the device manager
func InitDeviceManager() error {
	manager := &DeviceManager{
		devices:     make(map[string]*PCIDevice),
		drivers:     make(map[DeviceType][]DeviceDriver),
		irqHandlers: make(map[uint8][]*PCIDevice),
	}

	globalDeviceManager = manager

	// Scan for PCI devices
	return manager.ScanPCIDevices()
}

// RegisterDriver registers a device driver
func RegisterDriver(deviceType DeviceType, driver DeviceDriver) {
	if globalDeviceManager == nil {
		return
	}

	globalDeviceManager.mutex.Lock()
	defer globalDeviceManager.mutex.Unlock()

	globalDeviceManager.drivers[deviceType] = append(globalDeviceManager.drivers[deviceType], driver)
}

// ScanPCIDevices scans for PCI devices
func (dm *DeviceManager) ScanPCIDevices() error {
	// Scan all PCI buses, devices, and functions
	for bus := uint8(0); bus < 16; bus++ {
		for device := uint8(0); device < 32; device++ {
			for function := uint8(0); function < 8; function++ {
				if dm.probePCIDevice(bus, device, function) {
					// Device found, create PCIDevice instance
					pciDevice := &PCIDevice{
						info: DeviceInfo{
							BusNumber:    bus,
							DeviceNumber: device,
							Function:     function,
						},
					}

					// Read PCI configuration space
					if err := dm.readPCIConfig(pciDevice); err != nil {
						continue
					}

					// Try to find a suitable driver
					dm.matchDriver(pciDevice)

					deviceID := dm.makeDeviceID(bus, device, function)
					dm.devices[deviceID] = pciDevice
				}
			}
		}
	}

	return nil
}

// probePCIDevice checks if a PCI device exists at the given location
func (dm *DeviceManager) probePCIDevice(bus, device, function uint8) bool {
	vendorID := dm.readPCIConfigWord(bus, device, function, 0x00)
	return vendorID != 0xFFFF
}

// readPCIConfig reads the PCI configuration space
func (dm *DeviceManager) readPCIConfig(device *PCIDevice) error {
	bus := device.info.BusNumber
	dev := device.info.DeviceNumber
	func_ := device.info.Function

	config := &PCIConfigSpace{}

	config.VendorID = dm.readPCIConfigWord(bus, dev, func_, 0x00)
	config.DeviceID = dm.readPCIConfigWord(bus, dev, func_, 0x02)
	config.Command = dm.readPCIConfigWord(bus, dev, func_, 0x04)
	config.Status = dm.readPCIConfigWord(bus, dev, func_, 0x06)

	// Read class code information
	classInfo := dm.readPCIConfigDWord(bus, dev, func_, 0x08)
	config.RevisionID = uint8(classInfo & 0xFF)
	config.ProgIF = uint8((classInfo >> 8) & 0xFF)
	config.SubClass = uint8((classInfo >> 16) & 0xFF)
	config.ClassCode = uint8((classInfo >> 24) & 0xFF)

	// Read BARs
	for i := 0; i < 6; i++ {
		config.BAR[i] = dm.readPCIConfigDWord(bus, dev, func_, uint8(0x10+i*4))
	}

	// Read interrupt information
	config.InterruptLine = uint8(dm.readPCIConfigWord(bus, dev, func_, 0x3C) & 0xFF)
	config.InterruptPin = uint8((dm.readPCIConfigWord(bus, dev, func_, 0x3C) >> 8) & 0xFF)

	device.configSpace = config

	// Update device info
	device.info.VendorID = config.VendorID
	device.info.DeviceID = config.DeviceID
	device.info.Class = config.ClassCode
	device.info.SubClass = config.SubClass
	device.info.ProgIF = config.ProgIF
	device.info.Revision = config.RevisionID
	device.info.IRQ = config.InterruptLine

	// Determine device type from class code
	device.info.Type = dm.getDeviceTypeFromClass(config.ClassCode, config.SubClass)

	return nil
}

// matchDriver finds and attaches a suitable driver to the device
func (dm *DeviceManager) matchDriver(device *PCIDevice) {
	drivers, exists := dm.drivers[device.info.Type]
	if !exists {
		return
	}

	for _, driver := range drivers {
		if driver.Probe(device) {
			device.driver = driver
			if err := driver.Attach(device); err == nil {
				device.initialized = true
			}
			break
		}
	}
}

// getDeviceTypeFromClass determines device type from PCI class code
func (dm *DeviceManager) getDeviceTypeFromClass(class, subclass uint8) DeviceType {
	switch class {
	case 0x01: // Mass Storage Controller
		return DeviceTypeStorage
	case 0x02: // Network Controller
		return DeviceTypeNetwork
	case 0x03: // Display Controller
		return DeviceTypeGraphics
	case 0x04: // Multimedia Controller
		return DeviceTypeAudio
	case 0x09: // Input Device Controller
		return DeviceTypeInput
	case 0x0C: // Serial Bus Controller
		if subclass == 0x03 {
			return DeviceTypeUSB
		}
	}
	return DeviceTypeUnknown
}

// makeDeviceID creates a unique device identifier
func (dm *DeviceManager) makeDeviceID(bus, device, function uint8) string {
	return fmt.Sprintf("%02x:%02x.%x", bus, device, function)
}

// PCI configuration space access functions
func (dm *DeviceManager) readPCIConfigWord(bus, device, function, offset uint8) uint16 {
	address := dm.makePCIAddress(bus, device, function, offset)

	// Write address to CONFIG_ADDRESS
	OutPort32(0xCF8, address)

	// Read data from CONFIG_DATA
	data := InPort32(0xCFC)

	// Extract the requested word
	shift := (offset & 2) * 8
	return uint16((data >> shift) & 0xFFFF)
}

func (dm *DeviceManager) readPCIConfigDWord(bus, device, function, offset uint8) uint32 {
	address := dm.makePCIAddress(bus, device, function, offset)

	// Write address to CONFIG_ADDRESS
	OutPort32(0xCF8, address)

	// Read data from CONFIG_DATA
	return InPort32(0xCFC)
}

func (dm *DeviceManager) writePCIConfigWord(bus, device, function, offset uint8, value uint16) {
	address := dm.makePCIAddress(bus, device, function, offset)

	// Read current dword
	OutPort32(0xCF8, address)
	data := InPort32(0xCFC)

	// Modify the requested word
	shift := (offset & 2) * 8
	mask := uint32(0xFFFF << shift)
	data = (data &^ mask) | (uint32(value) << shift)

	// Write back
	OutPort32(0xCF8, address)
	OutPort32(0xCFC, data)
}

func (dm *DeviceManager) makePCIAddress(bus, device, function, offset uint8) uint32 {
	return (1 << 31) | // Enable bit
		(uint32(bus) << 16) |
		(uint32(device) << 11) |
		(uint32(function) << 8) |
		(uint32(offset) & 0xFC)
}

// PCIDevice implementation of Device interface
func (d *PCIDevice) Initialize() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.initialized {
		return nil
	}

	if d.driver != nil {
		if err := d.driver.Attach(d); err != nil {
			return err
		}
	}

	d.initialized = true
	return nil
}

func (d *PCIDevice) Start() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if !d.initialized {
		return fmt.Errorf("device not initialized")
	}

	if d.running {
		return nil
	}

	// Enable the device
	d.enablePCIDevice()

	d.running = true
	return nil
}

func (d *PCIDevice) Stop() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if !d.running {
		return nil
	}

	// Disable the device
	d.disablePCIDevice()

	d.running = false
	return nil
}

func (d *PCIDevice) Reset() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Perform device reset
	// This would involve device-specific reset procedures

	return nil
}

func (d *PCIDevice) GetInfo() DeviceInfo {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.info
}

func (d *PCIDevice) HandleInterrupt(vector uint8) bool {
	if d.driver != nil {
		return d.driver.HandleIRQ(d, vector)
	}
	return false
}

func (d *PCIDevice) Read(offset uintptr, size uintptr) ([]byte, error) {
	// Read from device memory or I/O space
	// Implementation depends on device type
	return nil, fmt.Errorf("not implemented")
}

func (d *PCIDevice) Write(offset uintptr, data []byte) error {
	// Write to device memory or I/O space
	// Implementation depends on device type
	return fmt.Errorf("not implemented")
}

func (d *PCIDevice) enablePCIDevice() {
	// Enable memory and I/O space access
	command := globalDeviceManager.readPCIConfigWord(
		d.info.BusNumber, d.info.DeviceNumber, d.info.Function, 0x04)
	command |= 0x0007 // Enable I/O Space, Memory Space, Bus Master
	globalDeviceManager.writePCIConfigWord(
		d.info.BusNumber, d.info.DeviceNumber, d.info.Function, 0x04, command)
}

func (d *PCIDevice) disablePCIDevice() {
	// Disable memory and I/O space access
	command := globalDeviceManager.readPCIConfigWord(
		d.info.BusNumber, d.info.DeviceNumber, d.info.Function, 0x04)
	command &^= 0x0007 // Disable I/O Space, Memory Space, Bus Master
	globalDeviceManager.writePCIConfigWord(
		d.info.BusNumber, d.info.DeviceNumber, d.info.Function, 0x04, command)
}

// RegisterInterruptHandler registers an interrupt handler for a device
func RegisterInterruptHandler(irq uint8, device *PCIDevice) error {
	if globalDeviceManager == nil {
		return fmt.Errorf("device manager not initialized")
	}

	globalDeviceManager.mutex.Lock()
	defer globalDeviceManager.mutex.Unlock()

	globalDeviceManager.irqHandlers[irq] = append(globalDeviceManager.irqHandlers[irq], device)

	return nil
}

// HandleGlobalInterrupt handles global interrupt and dispatches to appropriate devices
func HandleGlobalInterrupt(vector uint8) {
	if globalDeviceManager == nil {
		return
	}

	globalDeviceManager.mutex.RLock()
	devices := globalDeviceManager.irqHandlers[vector]
	globalDeviceManager.mutex.RUnlock()

	for _, device := range devices {
		if device.HandleInterrupt(vector) {
			break // Interrupt was handled
		}
	}
}

// GetAllDevices returns all detected devices
func GetAllDevices() map[string]*PCIDevice {
	if globalDeviceManager == nil {
		return nil
	}

	globalDeviceManager.mutex.RLock()
	defer globalDeviceManager.mutex.RUnlock()

	// Return a copy to prevent external modification
	devices := make(map[string]*PCIDevice)
	for k, v := range globalDeviceManager.devices {
		devices[k] = v
	}

	return devices
}

// GetDeviceByType returns devices of a specific type
func GetDeviceByType(deviceType DeviceType) []*PCIDevice {
	if globalDeviceManager == nil {
		return nil
	}

	globalDeviceManager.mutex.RLock()
	defer globalDeviceManager.mutex.RUnlock()

	var devices []*PCIDevice
	for _, device := range globalDeviceManager.devices {
		if device.info.Type == deviceType {
			devices = append(devices, device)
		}
	}

	return devices
}

// Port I/O helper functions for HAL integration
func init() {
	// These would be implemented in the HAL package
}

// Placeholder for HAL functions that would be implemented
func OutPort32(port uint16, value uint32) {
	// This would be implemented in HAL
}

func InPort32(port uint16) uint32 {
	// This would be implemented in HAL
	return 0
}
