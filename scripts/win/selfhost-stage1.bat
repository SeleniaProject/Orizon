@echo off
setlocal enabledelayedexpansion

:: Get script directory and set root
set "ROOT_DIR=%~dp0"
for %%i in ("!ROOT_DIR!..\..") do set "ROOT_DIR=%%~fi"

echo === Orizon Self-hosting Stage 1 ===
echo Compiling Orizon using Stage 0 Orizon tools...
echo Root directory: !ROOT_DIR!
echo Build directory: !ROOT_DIR!\build\stage1
echo Stage 0 tools: !ROOT_DIR!\build\stage0

:: Setup directories
if not exist "!ROOT_DIR!\build\stage1" mkdir "!ROOT_DIR!\build\stage1"
if not exist "!ROOT_DIR!\artifacts\selfhost" mkdir "!ROOT_DIR!\artifacts\selfhost"

:: Add Stage 0 tools to PATH
set "PATH=!ROOT_DIR!\build\stage0;!PATH!"

echo.
echo === Building Orizon using Orizon (Stage 1) ===

:: Build each tool using orizon-compiler from Stage 0
echo Building orizon with Stage 0 tools...
"!ROOT_DIR!\build\stage0\orizon-compiler.exe" -o "!ROOT_DIR!\build\stage1\orizon.exe" "!ROOT_DIR!\cmd\orizon"
if !ERRORLEVEL! neq 0 (
    echo ❌ orizon compilation failed
    exit /b 1
)
echo ✅ orizon built successfully

echo Building orizon-compiler with Stage 0 tools...
"!ROOT_DIR!\build\stage0\orizon-compiler.exe" -o "!ROOT_DIR!\build\stage1\orizon-compiler.exe" "!ROOT_DIR!\cmd\orizon-compiler"
if !ERRORLEVEL! neq 0 (
    echo ❌ orizon-compiler compilation failed
    exit /b 1
)
echo ✅ orizon-compiler built successfully

echo Building orizon-bootstrap with Stage 0 tools...
"!ROOT_DIR!\build\stage0\orizon-compiler.exe" -o "!ROOT_DIR!\build\stage1\orizon-bootstrap.exe" "!ROOT_DIR!\cmd\orizon-bootstrap"
if !ERRORLEVEL! neq 0 (
    echo ❌ orizon-bootstrap compilation failed
    exit /b 1
)
echo ✅ orizon-bootstrap built successfully

echo Building orizon-fmt with Stage 0 tools...
"!ROOT_DIR!\build\stage0\orizon-compiler.exe" -o "!ROOT_DIR!\build\stage1\orizon-fmt.exe" "!ROOT_DIR!\cmd\orizon-fmt"
if !ERRORLEVEL! neq 0 (
    echo ❌ orizon-fmt compilation failed
    exit /b 1
)
echo ✅ orizon-fmt built successfully

echo Building orizon-lsp with Stage 0 tools...
"!ROOT_DIR!\build\stage0\orizon-compiler.exe" -o "!ROOT_DIR!\build\stage1\orizon-lsp.exe" "!ROOT_DIR!\cmd\orizon-lsp"
if !ERRORLEVEL! neq 0 (
    echo ❌ orizon-lsp compilation failed
    exit /b 1
)
echo ✅ orizon-lsp built successfully

echo Building orizon-test with Stage 0 tools...
"!ROOT_DIR!\build\stage0\orizon-compiler.exe" -o "!ROOT_DIR!\build\stage1\orizon-test.exe" "!ROOT_DIR!\cmd\orizon-test"
if !ERRORLEVEL! neq 0 (
    echo ❌ orizon-test compilation failed
    exit /b 1
)
echo ✅ orizon-test built successfully

echo Building orizon-pkg with Stage 0 tools...
"!ROOT_DIR!\build\stage0\orizon-compiler.exe" -o "!ROOT_DIR!\build\stage1\orizon-pkg.exe" "!ROOT_DIR!\cmd\orizon-pkg"
if !ERRORLEVEL! neq 0 (
    echo ❌ orizon-pkg compilation failed
    exit /b 1
)
echo ✅ orizon-pkg built successfully

echo.
echo === Validating Stage 1 Tools ===

:: Test Stage 1 tools
echo Testing Stage 1 orizon-fmt...
echo fn main() { let x = 1 + 2; } | "!ROOT_DIR!\build\stage1\orizon-fmt.exe" --stdin > nul
if !ERRORLEVEL! neq 0 (
    echo ❌ Stage 1 orizon-fmt validation failed
    exit /b 1
)
echo ✅ Stage 1 orizon-fmt validation passed

echo Testing Stage 1 orizon-test...
"!ROOT_DIR!\build\stage1\orizon-test.exe" --help > nul
if !ERRORLEVEL! neq 0 (
    echo ❌ Stage 1 orizon-test validation failed
    exit /b 1
)
echo ✅ Stage 1 orizon-test validation passed

echo Testing Stage 1 orizon-pkg...
"!ROOT_DIR!\build\stage1\orizon-pkg.exe" --help > nul 2>&1
if !ERRORLEVEL! neq 0 (
    echo ❌ Stage 1 orizon-pkg validation failed
    exit /b 1
)
echo ✅ Stage 1 orizon-pkg validation passed

echo.
echo === Generating Stage 1 Build Metadata ===

:: Copy tools to artifacts directory
echo Copying Stage 1 tools to artifacts directory...
copy "!ROOT_DIR!\build\stage1\*.exe" "!ROOT_DIR!\artifacts\selfhost\" > nul

:: Generate metadata
for /f "delims=" %%i in ('git rev-parse --short HEAD 2^>nul') do set GIT_COMMIT=%%i
for /f "delims=" %%i in ('go version 2^>nul') do set GO_VERSION=%%i

echo Build Time: %DATE% %TIME% > "!ROOT_DIR!\artifacts\selfhost\stage1-build-info.txt"
echo Git Commit: %GIT_COMMIT% >> "!ROOT_DIR!\artifacts\selfhost\stage1-build-info.txt"
echo Go Version: %GO_VERSION% >> "!ROOT_DIR!\artifacts\selfhost\stage1-build-info.txt"
echo Build Method: Orizon self-compilation (Stage 1) >> "!ROOT_DIR!\artifacts\selfhost\stage1-build-info.txt"
echo Stage 0 Tools Used: Yes >> "!ROOT_DIR!\artifacts\selfhost\stage1-build-info.txt"

echo.
echo === Stage 1 Self-hosting Completed Successfully ===
echo Stage 1 artifacts available in: !ROOT_DIR!\artifacts\selfhost
echo Stage 1 tools built using Stage 0 Orizon tools
echo Ready for Stage 2 binary comparison...

exit /b 0
