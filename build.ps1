# Orizon OS Build Script for Windows PowerShell
# Run this script to build and test Orizon OS

param(
    [string]$Action = "build"
)

# Configuration
$KERNEL_NAME = "orizon_kernel"
$BOOTLOADER = "boot.bin"
$OS_IMAGE = "orizon_os.img"
$BUILD_DIR = "build"

# Colors for output
$Red = "`e[31m"
$Green = "`e[32m"
$Yellow = "`e[33m"
$Blue = "`e[34m"
$Reset = "`e[0m"

function Write-Status {
    param([string]$Message, [string]$Color = $Blue)
    Write-Host "$Color$Message$Reset"
}

function Write-Success {
    param([string]$Message)
    Write-Host "$Green✓ $Message$Reset"
}

function Write-Error {
    param([string]$Message)
    Write-Host "$Red✗ $Message$Reset"
}

function Write-Warning {
    param([string]$Message)
    Write-Host "$Yellow⚠ $Message$Reset"
}

function Check-Tools {
    Write-Status "Checking build tools..."
    
    # 環境変数PATHを更新
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
    
    # Chocolateyの標準インストールパスを追加
    $chocoPath = "C:\ProgramData\chocolatey\bin"
    $nasmPath = "C:\Program Files\NASM"
    $qemuPath = "C:\Program Files\qemu"
    
    if (Test-Path $chocoPath) { $env:Path += ";$chocoPath" }
    if (Test-Path $nasmPath) { $env:Path += ";$nasmPath" }
    if (Test-Path $qemuPath) { $env:Path += ";$qemuPath" }
    
    $tools = @{
        "nasm"               = "NASM assembler"
        "qemu-system-x86_64" = "QEMU emulator"
        "go"                 = "Go compiler"
    }
    
    $missing = @()
    foreach ($tool in $tools.Keys) {
        try {
            $command = Get-Command $tool -ErrorAction Stop
            Write-Success "$($tools[$tool]) found at: $($command.Source)"
        }
        catch {
            # 代替パスで検索
            $found = $false
            $alternatePaths = @(
                "C:\Program Files\NASM\$tool.exe",
                "C:\Program Files\qemu\$tool.exe",
                "C:\ProgramData\chocolatey\bin\$tool.exe",
                "C:\tools\$tool\$tool.exe"
            )
            
            foreach ($altPath in $alternatePaths) {
                if (Test-Path $altPath) {
                    Write-Success "$($tools[$tool]) found at: $altPath"
                    $found = $true
                    break
                }
            }
            
            if (-not $found) {
                $missing += $tool
                Write-Error "$($tools[$tool]) not found"
            }
        }
    }
    
    if ($missing.Count -gt 0) {
        Write-Warning "Missing tools: $($missing -join ', ')"
        Write-Status "To install missing tools, run as administrator:"
        Write-Host "  choco install nasm qemu -y" -ForegroundColor Yellow
        Write-Host ""
        Write-Status "Or use our install script:"
        Write-Host "  .\install.ps1" -ForegroundColor Yellow
        return $false
    }
    
    return $true
}

function Build-Bootloader {
    Write-Status "Building bootloader..."
    
    if (!(Test-Path $BUILD_DIR)) {
        New-Item -ItemType Directory -Path $BUILD_DIR | Out-Null
    }
    
    try {
        nasm -f bin boot\boot_minimal_fixed.asm -o "$BUILD_DIR\$BOOTLOADER"
        Write-Success "Bootloader built successfully"
        return $true
    }
    catch {
        Write-Error "Bootloader build failed: $($_.Exception.Message)"
        return $false
    }
}

function Build-Kernel {
    Write-Status "Building Orizon kernel..."
    
    try {
        # Build the kernel
        Set-Location cmd\orizon-kernel
        go build -ldflags="-s -w" -o "..\..\$BUILD_DIR\$KERNEL_NAME.exe" .
        Set-Location ..\..
        
        Write-Success "Kernel built successfully"
        return $true
    }
    catch {
        Write-Error "Kernel build failed: $($_.Exception.Message)"
        Set-Location ..\..
        return $false
    }
}

function Create-OSImage {
    Write-Status "Creating OS image..."
    
    try {
        # Create 10MB disk image instead of 1.44MB floppy
        $imageSize = 10 * 1024 * 1024  # 10MB
        $zeroBytes = New-Object byte[] $imageSize
        [System.IO.File]::WriteAllBytes("$BUILD_DIR\$OS_IMAGE", $zeroBytes)
        
        # Write bootloader to first sector
        $bootloader = [System.IO.File]::ReadAllBytes("$BUILD_DIR\$BOOTLOADER")
        $image = [System.IO.File]::ReadAllBytes("$BUILD_DIR\$OS_IMAGE")
        
        # Copy bootloader to start of image
        for ($i = 0; $i -lt $bootloader.Length -and $i -lt 512; $i++) {
            $image[$i] = $bootloader[$i]
        }
        
        # Try to read actual kernel binary
        $kernelPath = "$BUILD_DIR\$KERNEL_NAME.exe"
        if (Test-Path $kernelPath) {
            $kernelBytes = [System.IO.File]::ReadAllBytes($kernelPath)
            
            # Place kernel starting at sector 2 (offset 1024)
            $kernelOffset = 1024
            $maxKernelSize = $imageSize - $kernelOffset - 1024  # Leave some space
            
            if ($kernelBytes.Length -le $maxKernelSize) {
                for ($i = 0; $i -lt $kernelBytes.Length; $i++) {
                    $image[$kernelOffset + $i] = $kernelBytes[$i]
                }
                Write-Success "Kernel embedded: $($kernelBytes.Length) bytes"
            }
            else {
                Write-Warning "Kernel too large ($($kernelBytes.Length) bytes), using placeholder"
                # Use placeholder if kernel is too large
                $kernelPlaceholder = [System.Text.Encoding]::ASCII.GetBytes("ORIZON KERNEL PLACEHOLDER - KERNEL TOO LARGE")
                for ($i = 0; $i -lt $kernelPlaceholder.Length; $i++) {
                    $image[1024 + $i] = $kernelPlaceholder[$i]
                }
            }
        }
        else {
            Write-Warning "Kernel binary not found, using placeholder"
            # Use placeholder if kernel not found
            $kernelPlaceholder = [System.Text.Encoding]::ASCII.GetBytes("ORIZON KERNEL PLACEHOLDER - NO KERNEL FOUND")
            for ($i = 0; $i -lt $kernelPlaceholder.Length; $i++) {
                $image[1024 + $i] = $kernelPlaceholder[$i]
            }
        }
        
        [System.IO.File]::WriteAllBytes("$BUILD_DIR\$OS_IMAGE", $image)
        
        Write-Success "OS image created: $BUILD_DIR\$OS_IMAGE"
        return $true
    }
    catch {
        Write-Error "OS image creation failed: $($_.Exception.Message)"
        return $false
    }
}

function Run-OS {
    Write-Status "Starting Orizon OS in QEMU..."
    
    if (!(Test-Path "$BUILD_DIR\$OS_IMAGE")) {
        Write-Error "OS image not found. Run build first."
        return
    }
    
    try {
        qemu-system-x86_64 -m 512M -drive file="$BUILD_DIR\$OS_IMAGE", format=raw, if=ide -boot c -monitor stdio
    }
    catch {
        Write-Error "Failed to start QEMU: $($_.Exception.Message)"
    }
}

function Test-OS {
    Write-Status "Testing Orizon OS (console mode)..."
    
    if (!(Test-Path "$BUILD_DIR\$OS_IMAGE")) {
        Write-Error "OS image not found. Run build first."
        return
    }
    
    try {
        qemu-system-x86_64 -m 512M -drive file="$BUILD_DIR\$OS_IMAGE", format=raw, if=ide -boot c -nographic -serial mon:stdio -no-reboot
    }
    catch {
        Write-Error "Failed to test in QEMU: $($_.Exception.Message)"
    }
}

function Clean-Build {
    Write-Status "Cleaning build artifacts..."
    
    if (Test-Path $BUILD_DIR) {
        Remove-Item -Recurse -Force $BUILD_DIR
        Write-Success "Build directory cleaned"
    }
    else {
        Write-Warning "Build directory doesn't exist"
    }
}

function Show-Info {
    Write-Host ""
    Write-Host "${Blue}Orizon OS Build System${Reset}"
    Write-Host "${Blue}=====================${Reset}"
    Write-Host ""
    Write-Host "Available commands:"
    Write-Host "  .\build.ps1 build      - Build complete OS"
    Write-Host "  .\build.ps1 run        - Build and run in QEMU"
    Write-Host "  .\build.ps1 test       - Build and test (console)"
    Write-Host "  .\build.ps1 clean      - Clean build artifacts"
    Write-Host "  .\build.ps1 check      - Check build tools"
    Write-Host "  .\build.ps1 info       - Show this information"
    Write-Host ""
    Write-Host "Files:"
    Write-Host "  Bootloader: boot\boot.asm"
    Write-Host "  Kernel: cmd\orizon-kernel\main.go"
    Write-Host "  Output: $BUILD_DIR\$OS_IMAGE"
    Write-Host ""
}

function Show-Size {
    if (Test-Path "$BUILD_DIR\$BOOTLOADER") {
        $bootSize = (Get-Item "$BUILD_DIR\$BOOTLOADER").Length
        Write-Host "Bootloader: $bootSize bytes"
    }
    
    if (Test-Path "$BUILD_DIR\$KERNEL_NAME.exe") {
        $kernelSize = (Get-Item "$BUILD_DIR\$KERNEL_NAME.exe").Length
        Write-Host "Kernel: $kernelSize bytes"
    }
    
    if (Test-Path "$BUILD_DIR\$OS_IMAGE") {
        $imageSize = (Get-Item "$BUILD_DIR\$OS_IMAGE").Length
        Write-Host "OS Image: $imageSize bytes"
    }
}

# Main script logic
switch ($Action.ToLower()) {
    "build" {
        Write-Status "Building Orizon OS..."
        
        if (!(Check-Tools)) {
            exit 1
        }
        
        if (!(Build-Bootloader)) {
            exit 1
        }
        
        if (!(Build-Kernel)) {
            exit 1
        }
        
        if (!(Create-OSImage)) {
            exit 1
        }
        
        Write-Success "Orizon OS built successfully!"
        Show-Size
    }
    
    "run" {
        & $MyInvocation.MyCommand.Path build
        if ($LASTEXITCODE -eq 0) {
            Run-OS
        }
    }
    
    "test" {
        & $MyInvocation.MyCommand.Path build
        if ($LASTEXITCODE -eq 0) {
            Test-OS
        }
    }
    
    "clean" {
        Clean-Build
    }
    
    "check" {
        Check-Tools
    }
    
    "info" {
        Show-Info
    }
    
    "size" {
        Show-Size
    }
    
    default {
        Write-Error "Unknown action: $Action"
        Show-Info
        exit 1
    }
}

Write-Host ""
