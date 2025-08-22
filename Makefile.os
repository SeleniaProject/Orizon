# Orizon OS Build System
# Complete build system for creating a bootable Orizon OS

# Configuration
KERNEL_NAME = orizon_kernel
BOOTLOADER = boot.bin
OS_IMAGE = orizon_os.img
KERNEL_SIZE = 63  # sectors

# Tools
NASM = nasm
QEMU = qemu-system-x86_64
GO = go
OBJCOPY = objcopy

# Directories
BOOT_DIR = boot
KERNEL_DIR = internal/runtime/kernel
BUILD_DIR = build
EXAMPLES_DIR = examples

# Go build flags
GO_FLAGS = -ldflags="-s -w" -tags="kernel"

.PHONY: all clean bootloader kernel os-image run test install

all: os-image

# Build bootloader
bootloader: $(BUILD_DIR)/$(BOOTLOADER)

$(BUILD_DIR)/$(BOOTLOADER): $(BOOT_DIR)/boot.asm | $(BUILD_DIR)
	@echo "Building bootloader..."
	$(NASM) -f bin $< -o $@
	@echo "Bootloader built successfully"

# Build kernel
kernel: $(BUILD_DIR)/$(KERNEL_NAME).bin

$(BUILD_DIR)/$(KERNEL_NAME).bin: $(KERNEL_DIR)/*.go | $(BUILD_DIR)
	@echo "Building Orizon kernel..."
	cd $(KERNEL_DIR) && $(GO) build $(GO_FLAGS) -o ../../../$(BUILD_DIR)/$(KERNEL_NAME) .
	$(OBJCOPY) -O binary $(BUILD_DIR)/$(KERNEL_NAME) $@
	@echo "Kernel built successfully"

# Create OS image
os-image: $(BUILD_DIR)/$(OS_IMAGE)

$(BUILD_DIR)/$(OS_IMAGE): $(BUILD_DIR)/$(BOOTLOADER) $(BUILD_DIR)/$(KERNEL_NAME).bin | $(BUILD_DIR)
	@echo "Creating OS image..."
	# Create 1.44MB floppy image
	dd if=/dev/zero of=$@ bs=512 count=2880 2>/dev/null
	# Write bootloader to first sector
	dd if=$(BUILD_DIR)/$(BOOTLOADER) of=$@ bs=512 count=1 conv=notrunc 2>/dev/null
	# Write kernel starting from sector 2
	dd if=$(BUILD_DIR)/$(KERNEL_NAME).bin of=$@ bs=512 seek=1 count=$(KERNEL_SIZE) conv=notrunc 2>/dev/null
	@echo "OS image created: $(BUILD_DIR)/$(OS_IMAGE)"

# Build directory
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Run in QEMU
run: os-image
	@echo "Starting Orizon OS in QEMU..."
	$(QEMU) -m 512M -fda $(BUILD_DIR)/$(OS_IMAGE) -boot a -monitor stdio

# Run with debugging
debug: os-image
	@echo "Starting Orizon OS in QEMU with debugging..."
	$(QEMU) -m 512M -fda $(BUILD_DIR)/$(OS_IMAGE) -boot a -s -S -monitor stdio

# Test with verbose output
test: os-image
	@echo "Testing Orizon OS..."
	$(QEMU) -m 512M -fda $(BUILD_DIR)/$(OS_IMAGE) -boot a -nographic -serial mon:stdio -no-reboot

# Install QEMU and build tools (Windows)
install-windows:
	@echo "Installing build dependencies for Windows..."
	@echo "Please install the following manually:"
	@echo "1. NASM: https://www.nasm.us/pub/nasm/releasebuilds/"
	@echo "2. QEMU: https://www.qemu.org/download/#windows"
	@echo "3. TDM-GCC: https://jmeubank.github.io/tdm-gcc/"
	@echo "4. Make sure tools are in PATH"

# Install build tools (Linux/WSL)
install-linux:
	@echo "Installing build dependencies..."
	sudo apt-get update
	sudo apt-get install -y nasm qemu-system-x86 build-essential

# Compile Orizon program examples
compile-examples: $(BUILD_DIR)
	@echo "Compiling Orizon examples..."
	# This would invoke the Orizon compiler when it's ready
	cp $(EXAMPLES_DIR)/orizon_os.oriz $(BUILD_DIR)/
	@echo "Examples ready"

# Create bootable USB (Linux only)
usb: os-image
	@echo "Warning: This will overwrite the USB device!"
	@read -p "Enter USB device (e.g., /dev/sdb): " device; \
	sudo dd if=$(BUILD_DIR)/$(OS_IMAGE) of=$$device bs=512

# Create ISO image
iso: os-image $(BUILD_DIR)
	@echo "Creating ISO image..."
	mkdir -p $(BUILD_DIR)/iso
	cp $(BUILD_DIR)/$(OS_IMAGE) $(BUILD_DIR)/iso/
	genisoimage -r -b $(OS_IMAGE) -no-emul-boot -boot-load-size 4 -boot-info-table -o $(BUILD_DIR)/orizon_os.iso $(BUILD_DIR)/iso/
	@echo "ISO created: $(BUILD_DIR)/orizon_os.iso"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Show build information
info:
	@echo "Orizon OS Build System"
	@echo "======================"
	@echo "Bootloader: $(BOOT_DIR)/boot.asm"
	@echo "Kernel: $(KERNEL_DIR)/"
	@echo "Output: $(BUILD_DIR)/$(OS_IMAGE)"
	@echo ""
	@echo "Available targets:"
	@echo "  all       - Build complete OS image"
	@echo "  bootloader - Build bootloader only"
	@echo "  kernel    - Build kernel only"
	@echo "  os-image  - Create bootable OS image"
	@echo "  run       - Run OS in QEMU"
	@echo "  debug     - Run OS in QEMU with debugging"
	@echo "  test      - Test OS with console output"
	@echo "  iso       - Create ISO image"
	@echo "  usb       - Write to USB device (Linux)"
	@echo "  clean     - Clean build artifacts"
	@echo "  install-* - Install build dependencies"

# Help target
help: info

# Check if required tools are available
check-tools:
	@echo "Checking build tools..."
	@which $(NASM) >/dev/null 2>&1 || (echo "NASM not found"; exit 1)
	@which $(QEMU) >/dev/null 2>&1 || (echo "QEMU not found"; exit 1)
	@which $(GO) >/dev/null 2>&1 || (echo "Go not found"; exit 1)
	@echo "All tools available"

# Size information
size: os-image
	@echo "Build size information:"
	@echo "Bootloader: $$(wc -c < $(BUILD_DIR)/$(BOOTLOADER)) bytes"
	@echo "Kernel: $$(wc -c < $(BUILD_DIR)/$(KERNEL_NAME).bin) bytes"
	@echo "OS Image: $$(wc -c < $(BUILD_DIR)/$(OS_IMAGE)) bytes"

# Quick test - build and run
quick: all run
