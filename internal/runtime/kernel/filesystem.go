// Package kernel provides file system and boot functionality for OS development
package kernel

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"
)

// ============================================================================
// File system interface
// ============================================================================

// FileType represents different types of files
type FileType uint8

const (
	FileTypeRegular FileType = iota
	FileTypeDirectory
	FileTypeSymlink
	FileTypeDevice
	FileTypePipe
	FileTypeSocket
)

// FilePermissions represents file permissions (Unix-style)
type FilePermissions uint16

const (
	PermRead    FilePermissions = 0o444
	PermWrite   FilePermissions = 0o222
	PermExecute FilePermissions = 0o111
	PermAll     FilePermissions = 0o777
)

// FileInfo contains information about a file
type FileInfo struct {
	Name        string
	Size        uint64
	Type        FileType
	Permissions FilePermissions
	CreatedAt   time.Time
	ModifiedAt  time.Time
	AccessedAt  time.Time
	OwnerUID    uint32
	GroupGID    uint32
}

// File represents an open file
type File struct {
	Info     *FileInfo
	Position uint64
	Flags    uint32
	Inode    *Inode
}

// Inode represents a file system inode
type Inode struct {
	Number   uint64
	Info     *FileInfo
	Data     []byte
	Children map[string]*Inode // For directories
	Parent   *Inode
	RefCount uint32
	mutex    sync.RWMutex
}

// FileSystem interface for different file system implementations
type FileSystem interface {
	Mount(device string, mountPoint string) error
	Unmount(mountPoint string) error
	Open(path string, flags uint32) (*File, error)
	Close(file *File) error
	Read(file *File, buffer []byte) (int, error)
	Write(file *File, data []byte) (int, error)
	Seek(file *File, offset int64, whence int) (int64, error)
	Stat(path string) (*FileInfo, error)
	Create(path string, permissions FilePermissions) (*File, error)
	Remove(path string) error
	Mkdir(path string, permissions FilePermissions) error
	Rmdir(path string) error
	List(path string) ([]*FileInfo, error)
}

// VirtualFileSystem provides a unified file system interface
type VirtualFileSystem struct {
	root       *Inode
	mounts     map[string]FileSystem
	openFiles  map[uint64]*File
	nextFileID uint64
	mutex      sync.RWMutex
}

// GlobalVFS provides global file system access
var GlobalVFS *VirtualFileSystem

// InitializeVFS initializes the virtual file system
func InitializeVFS() error {
	if GlobalVFS != nil {
		return fmt.Errorf("VFS already initialized")
	}

	// Create root directory
	rootInfo := &FileInfo{
		Name:        "/",
		Size:        0,
		Type:        FileTypeDirectory,
		Permissions: PermAll,
		CreatedAt:   time.Now(),
		ModifiedAt:  time.Now(),
		AccessedAt:  time.Now(),
	}

	root := &Inode{
		Number:   1,
		Info:     rootInfo,
		Children: make(map[string]*Inode),
	}

	GlobalVFS = &VirtualFileSystem{
		root:       root,
		mounts:     make(map[string]FileSystem),
		openFiles:  make(map[uint64]*File),
		nextFileID: 1,
	}

	// Create basic directory structure
	err := GlobalVFS.createBasicDirectories()
	if err != nil {
		return fmt.Errorf("failed to create basic directories: %w", err)
	}

	return nil
}

// createBasicDirectories creates basic Unix-like directory structure
func (vfs *VirtualFileSystem) createBasicDirectories() error {
	dirs := []string{
		"/bin",
		"/sbin",
		"/usr",
		"/usr/bin",
		"/usr/sbin",
		"/etc",
		"/var",
		"/tmp",
		"/dev",
		"/proc",
		"/sys",
		"/home",
	}

	for _, dir := range dirs {
		err := vfs.Mkdir(dir, PermAll)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create basic device files
	devices := map[string]string{
		"/dev/null":    "null",
		"/dev/zero":    "zero",
		"/dev/console": "console",
		"/dev/tty":     "tty",
	}

	for path, deviceName := range devices {
		err := vfs.createDeviceFile(path, deviceName)
		if err != nil {
			return fmt.Errorf("failed to create device %s: %w", path, err)
		}
	}

	return nil
}

// createDeviceFile creates a device file
func (vfs *VirtualFileSystem) createDeviceFile(path, deviceName string) error {
	parent, filename := vfs.splitPath(path)

	parentInode, err := vfs.findInode(parent)
	if err != nil {
		return err
	}

	deviceInfo := &FileInfo{
		Name:        filename,
		Size:        0,
		Type:        FileTypeDevice,
		Permissions: PermRead | PermWrite,
		CreatedAt:   time.Now(),
		ModifiedAt:  time.Now(),
		AccessedAt:  time.Now(),
	}

	deviceInode := &Inode{
		Number:   vfs.nextInodeNumber(),
		Info:     deviceInfo,
		Parent:   parentInode,
		Children: make(map[string]*Inode),
	}

	parentInode.mutex.Lock()
	parentInode.Children[filename] = deviceInode
	parentInode.mutex.Unlock()

	return nil
}

// Open opens a file
func (vfs *VirtualFileSystem) Open(path string, flags uint32) (*File, error) {
	vfs.mutex.Lock()
	defer vfs.mutex.Unlock()

	inode, err := vfs.findInode(path)
	if err != nil {
		return nil, err
	}

	file := &File{
		Info:     inode.Info,
		Position: 0,
		Flags:    flags,
		Inode:    inode,
	}

	fileID := vfs.nextFileID
	vfs.nextFileID++
	vfs.openFiles[fileID] = file

	inode.mutex.Lock()
	inode.RefCount++
	inode.mutex.Unlock()

	return file, nil
}

// Close closes a file
func (vfs *VirtualFileSystem) Close(file *File) error {
	vfs.mutex.Lock()
	defer vfs.mutex.Unlock()

	// Find and remove from open files
	for id, f := range vfs.openFiles {
		if f == file {
			delete(vfs.openFiles, id)
			break
		}
	}

	file.Inode.mutex.Lock()
	file.Inode.RefCount--
	file.Inode.mutex.Unlock()

	return nil
}

// Read reads from a file
func (vfs *VirtualFileSystem) Read(file *File, buffer []byte) (int, error) {
	file.Inode.mutex.RLock()
	defer file.Inode.mutex.RUnlock()

	// Handle device files
	if file.Info.Type == FileTypeDevice {
		return vfs.readDevice(file, buffer)
	}

	// Regular file read
	if file.Position >= uint64(len(file.Inode.Data)) {
		return 0, nil // EOF
	}

	remaining := uint64(len(file.Inode.Data)) - file.Position
	toRead := uint64(len(buffer))
	if toRead > remaining {
		toRead = remaining
	}

	copy(buffer, file.Inode.Data[file.Position:file.Position+toRead])
	file.Position += toRead

	return int(toRead), nil
}

// Write writes to a file
func (vfs *VirtualFileSystem) Write(file *File, data []byte) (int, error) {
	file.Inode.mutex.Lock()
	defer file.Inode.mutex.Unlock()

	// Handle device files
	if file.Info.Type == FileTypeDevice {
		return vfs.writeDevice(file, data)
	}

	// Expand file data if necessary
	requiredSize := file.Position + uint64(len(data))
	if requiredSize > uint64(len(file.Inode.Data)) {
		newData := make([]byte, requiredSize)
		copy(newData, file.Inode.Data)
		file.Inode.Data = newData
	}

	copy(file.Inode.Data[file.Position:], data)
	file.Position += uint64(len(data))
	file.Info.Size = uint64(len(file.Inode.Data))
	file.Info.ModifiedAt = time.Now()

	return len(data), nil
}

// readDevice handles reading from device files
func (vfs *VirtualFileSystem) readDevice(file *File, buffer []byte) (int, error) {
	switch file.Info.Name {
	case "null":
		return 0, nil // Always EOF
	case "zero":
		// Fill buffer with zeros
		for i := range buffer {
			buffer[i] = 0
		}
		return len(buffer), nil
	case "console", "tty":
		// Read from keyboard (placeholder)
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown device: %s", file.Info.Name)
	}
}

// writeDevice handles writing to device files
func (vfs *VirtualFileSystem) writeDevice(file *File, data []byte) (int, error) {
	switch file.Info.Name {
	case "null":
		return len(data), nil // Discard all data
	case "zero":
		return len(data), nil // Discard all data
	case "console", "tty":
		// Write to console
		if GlobalDeviceManager != nil {
			return KernelWriteConsole(data), nil
		}
		return len(data), nil
	default:
		return 0, fmt.Errorf("unknown device: %s", file.Info.Name)
	}
}

// Mkdir creates a directory
func (vfs *VirtualFileSystem) Mkdir(path string, permissions FilePermissions) error {
	parent, filename := vfs.splitPath(path)

	parentInode, err := vfs.findInode(parent)
	if err != nil {
		return err
	}

	parentInode.mutex.Lock()
	defer parentInode.mutex.Unlock()

	// Check if directory already exists
	if _, exists := parentInode.Children[filename]; exists {
		return fmt.Errorf("directory %s already exists", path)
	}

	dirInfo := &FileInfo{
		Name:        filename,
		Size:        0,
		Type:        FileTypeDirectory,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		ModifiedAt:  time.Now(),
		AccessedAt:  time.Now(),
	}

	dirInode := &Inode{
		Number:   vfs.nextInodeNumber(),
		Info:     dirInfo,
		Parent:   parentInode,
		Children: make(map[string]*Inode),
	}

	parentInode.Children[filename] = dirInode

	return nil
}

// Create creates a regular file
func (vfs *VirtualFileSystem) Create(path string, permissions FilePermissions) (*File, error) {
	parent, filename := vfs.splitPath(path)

	parentInode, err := vfs.findInode(parent)
	if err != nil {
		return nil, err
	}

	parentInode.mutex.Lock()
	defer parentInode.mutex.Unlock()

	fileInfo := &FileInfo{
		Name:        filename,
		Size:        0,
		Type:        FileTypeRegular,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		ModifiedAt:  time.Now(),
		AccessedAt:  time.Now(),
	}

	fileInode := &Inode{
		Number:   vfs.nextInodeNumber(),
		Info:     fileInfo,
		Parent:   parentInode,
		Data:     make([]byte, 0),
		Children: make(map[string]*Inode),
	}

	parentInode.Children[filename] = fileInode

	// Return opened file
	file := &File{
		Info:     fileInfo,
		Position: 0,
		Flags:    0,
		Inode:    fileInode,
	}

	vfs.mutex.Lock()
	fileID := vfs.nextFileID
	vfs.nextFileID++
	vfs.openFiles[fileID] = file
	vfs.mutex.Unlock()

	fileInode.RefCount++

	return file, nil
}

// List lists directory contents
func (vfs *VirtualFileSystem) List(path string) ([]*FileInfo, error) {
	inode, err := vfs.findInode(path)
	if err != nil {
		return nil, err
	}

	if inode.Info.Type != FileTypeDirectory {
		return nil, fmt.Errorf("%s is not a directory", path)
	}

	inode.mutex.RLock()
	defer inode.mutex.RUnlock()

	var files []*FileInfo
	for _, child := range inode.Children {
		files = append(files, child.Info)
	}

	return files, nil
}

// Helper functions

// findInode finds an inode by path
func (vfs *VirtualFileSystem) findInode(path string) (*Inode, error) {
	if path == "/" {
		return vfs.root, nil
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	current := vfs.root

	for _, part := range parts {
		if part == "" {
			continue
		}

		current.mutex.RLock()
		child, exists := current.Children[part]
		current.mutex.RUnlock()

		if !exists {
			return nil, fmt.Errorf("path not found: %s", path)
		}

		current = child
	}

	return current, nil
}

// splitPath splits a path into parent directory and filename
func (vfs *VirtualFileSystem) splitPath(path string) (string, string) {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 1 {
		return "/", parts[0]
	}

	parent := "/" + strings.Join(parts[:len(parts)-1], "/")
	filename := parts[len(parts)-1]

	return parent, filename
}

// nextInodeNumber returns the next available inode number
func (vfs *VirtualFileSystem) nextInodeNumber() uint64 {
	// Simple incrementing counter (in real FS, this would be more sophisticated)
	static_counter := uint64(2) // Start after root inode (1)
	static_counter++
	return static_counter
}

// ============================================================================
// Boot and initialization
// ============================================================================

// BootInfo contains information from the bootloader
type BootInfo struct {
	MemoryMapEntries []MemoryMapEntry
	KernelBase       uintptr
	KernelSize       uintptr
	InitrdBase       uintptr
	InitrdSize       uintptr
	CommandLine      string
}

// MemoryMapEntry represents a memory map entry from the bootloader
type MemoryMapEntry struct {
	Base   uint64
	Length uint64
	Type   uint32
}

// BootStrap performs initial kernel bootstrapping
func BootStrap(bootInfo *BootInfo) error {
	fmt.Println("Orizon Kernel Bootstrap starting...")

	// Initialize memory management first
	err := InitializeKernel()
	if err != nil {
		return fmt.Errorf("failed to initialize kernel: %w", err)
	}

	// Add memory regions from boot info
	if bootInfo != nil {
		for _, entry := range bootInfo.MemoryMapEntries {
			if entry.Type == 1 { // Available RAM
				err = GlobalKernel.PhysicalMemory.AddRegion(
					uintptr(entry.Base),
					uintptr(entry.Length),
					MemoryTypeRAM,
				)
				if err != nil {
					fmt.Printf("Warning: failed to add memory region: %v\n", err)
				}
			}
		}
	}

	// Initialize interrupt handling
	err = InitializeInterrupts()
	if err != nil {
		return fmt.Errorf("failed to initialize interrupts: %w", err)
	}

	// Initialize timers
	err = InitializeTimers()
	if err != nil {
		return fmt.Errorf("failed to initialize timers: %w", err)
	}

	// Initialize process management
	err = InitializeProcessManager()
	if err != nil {
		return fmt.Errorf("failed to initialize process manager: %w", err)
	}

	// Initialize device management
	err = InitializeDeviceManager()
	if err != nil {
		return fmt.Errorf("failed to initialize device manager: %w", err)
	}

	// Initialize file system
	err = InitializeVFS()
	if err != nil {
		return fmt.Errorf("failed to initialize VFS: %w", err)
	}

	fmt.Println("Kernel bootstrap completed successfully!")
	return nil
}

// KernelMain is the main kernel entry point after bootstrap
func KernelMain() {
	fmt.Println("Orizon Kernel starting...")

	// Print system information
	total, free, used := GlobalKernel.PhysicalMemory.GetMemoryInfo()
	fmt.Printf("Memory: %d KB total, %d KB free, %d KB used\n",
		total/1024, free/1024, used/1024)

	// Create init process
	if GlobalProcessManager != nil {
		initProcess, err := GlobalProcessManager.CreateProcess("init",
			uintptr(0x400000), // Entry point
			64*1024)           // 64KB stack
		if err != nil {
			fmt.Printf("Failed to create init process: %v\n", err)
		} else {
			fmt.Printf("Created init process with PID %d\n", initProcess.PID)
		}
	}

	// Enable interrupts
	EnableInterrupts()

	// Main kernel loop
	for {
		// Yield to other processes
		if GlobalProcessManager != nil {
			GlobalProcessManager.Yield()
		}

		// Sleep a bit to avoid busy waiting
		if GlobalTimerManager != nil {
			GlobalTimerManager.Sleep(10) // 10ms
		}
	}
}

// ============================================================================
// Kernel API functions for Orizon
// ============================================================================

// KernelOpenFile opens a file
func KernelOpenFile(path string, flags uint32) uintptr {
	if GlobalVFS == nil {
		return 0
	}

	file, err := GlobalVFS.Open(path, flags)
	if err != nil {
		return 0
	}

	// Return file pointer (simplified)
	return uintptr(unsafe.Pointer(file))
}

// KernelCloseFile closes a file
func KernelCloseFile(filePtr uintptr) bool {
	if GlobalVFS == nil || filePtr == 0 {
		return false
	}

	file := (*File)(unsafe.Pointer(filePtr))
	err := GlobalVFS.Close(file)
	return err == nil
}

// KernelReadFile reads from a file
func KernelReadFile(filePtr uintptr, buffer []byte) int {
	if GlobalVFS == nil || filePtr == 0 {
		return 0
	}

	file := (*File)(unsafe.Pointer(filePtr))
	n, err := GlobalVFS.Read(file, buffer)
	if err != nil {
		return 0
	}

	return n
}

// KernelWriteFile writes to a file
func KernelWriteFile(filePtr uintptr, data []byte) int {
	if GlobalVFS == nil || filePtr == 0 {
		return 0
	}

	file := (*File)(unsafe.Pointer(filePtr))
	n, err := GlobalVFS.Write(file, data)
	if err != nil {
		return 0
	}

	return n
}

// KernelCreateFile creates a new file
func KernelCreateFile(path string, permissions uint16) uintptr {
	if GlobalVFS == nil {
		return 0
	}

	file, err := GlobalVFS.Create(path, FilePermissions(permissions))
	if err != nil {
		return 0
	}

	return uintptr(unsafe.Pointer(file))
}

// KernelMkdir creates a directory
func KernelMkdir(path string, permissions uint16) bool {
	if GlobalVFS == nil {
		return false
	}

	err := GlobalVFS.Mkdir(path, FilePermissions(permissions))
	return err == nil
}
