@echo off
setlocal enabledelayedexpansion

:: Get script directory and set root
set "ROOT_DIR=%~dp0"
for %%i in ("!ROOT_DIR!..\..") do set "ROOT_DIR=%%~fi"

echo === Orizon Self-hosting Stage 2 ===
echo Compiling Orizon using Stage 1 Orizon tools and comparing...
echo Root directory: !ROOT_DIR!
echo Build directory: !ROOT_DIR!\build\stage2
echo Stage 1 tools: !ROOT_DIR!\build\stage1

:: Setup directories
if not exist "!ROOT_DIR!\build\stage2" mkdir "!ROOT_DIR!\build\stage2"
if not exist "!ROOT_DIR!\artifacts\selfhost" mkdir "!ROOT_DIR!\artifacts\selfhost"

:: Add Stage 1 tools to PATH
set "PATH=!ROOT_DIR!\build\stage1;!PATH!"

echo.
echo === Building Orizon using Stage 1 Orizon tools ===

:: Build each tool using orizon-compiler from Stage 1
echo Building orizon with Stage 1 tools...
"!ROOT_DIR!\build\stage1\orizon-compiler.exe" -o "!ROOT_DIR!\build\stage2\orizon.exe" "!ROOT_DIR!\cmd\orizon"
if !ERRORLEVEL! neq 0 (
    echo ❌ Stage 2 orizon compilation failed
    exit /b 1
)
echo ✅ Stage 2 orizon built successfully

echo Building orizon-compiler with Stage 1 tools...
"!ROOT_DIR!\build\stage1\orizon-compiler.exe" -o "!ROOT_DIR!\build\stage2\orizon-compiler.exe" "!ROOT_DIR!\cmd\orizon-compiler"
if !ERRORLEVEL! neq 0 (
    echo ❌ Stage 2 orizon-compiler compilation failed
    exit /b 1
)
echo ✅ Stage 2 orizon-compiler built successfully

echo Building orizon-bootstrap with Stage 1 tools...
"!ROOT_DIR!\build\stage1\orizon-compiler.exe" -o "!ROOT_DIR!\build\stage2\orizon-bootstrap.exe" "!ROOT_DIR!\cmd\orizon-bootstrap"
if !ERRORLEVEL! neq 0 (
    echo ❌ Stage 2 orizon-bootstrap compilation failed
    exit /b 1
)
echo ✅ Stage 2 orizon-bootstrap built successfully

echo Building orizon-fmt with Stage 1 tools...
"!ROOT_DIR!\build\stage1\orizon-compiler.exe" -o "!ROOT_DIR!\build\stage2\orizon-fmt.exe" "!ROOT_DIR!\cmd\orizon-fmt"
if !ERRORLEVEL! neq 0 (
    echo ❌ Stage 2 orizon-fmt compilation failed
    exit /b 1
)
echo ✅ Stage 2 orizon-fmt built successfully

echo Building orizon-lsp with Stage 1 tools...
"!ROOT_DIR!\build\stage1\orizon-compiler.exe" -o "!ROOT_DIR!\build\stage2\orizon-lsp.exe" "!ROOT_DIR!\cmd\orizon-lsp"
if !ERRORLEVEL! neq 0 (
    echo ❌ Stage 2 orizon-lsp compilation failed
    exit /b 1
)
echo ✅ Stage 2 orizon-lsp built successfully

echo Building orizon-test with Stage 1 tools...
"!ROOT_DIR!\build\stage1\orizon-compiler.exe" -o "!ROOT_DIR!\build\stage2\orizon-test.exe" "!ROOT_DIR!\cmd\orizon-test"
if !ERRORLEVEL! neq 0 (
    echo ❌ Stage 2 orizon-test compilation failed
    exit /b 1
)
echo ✅ Stage 2 orizon-test built successfully

echo Building orizon-pkg with Stage 1 tools...
"!ROOT_DIR!\build\stage1\orizon-compiler.exe" -o "!ROOT_DIR!\build\stage2\orizon-pkg.exe" "!ROOT_DIR!\cmd\orizon-pkg"
if !ERRORLEVEL! neq 0 (
    echo ❌ Stage 2 orizon-pkg compilation failed
    exit /b 1
)
echo ✅ Stage 2 orizon-pkg built successfully

echo.
echo === Comparing Stage 1 and Stage 2 Binaries ===

set "COMPARISON_PASSED=true"

:: Compare each binary
for %%f in (orizon.exe orizon-compiler.exe orizon-bootstrap.exe orizon-fmt.exe orizon-lsp.exe orizon-test.exe orizon-pkg.exe) do (
    echo Comparing %%f...
    fc /b "!ROOT_DIR!\build\stage1\%%f" "!ROOT_DIR!\build\stage2\%%f" > nul 2>&1
    if !ERRORLEVEL! equ 0 (
        echo ✅ %%f is identical between Stage 1 and Stage 2
    ) else (
        echo ❌ %%f differs between Stage 1 and Stage 2
        set "COMPARISON_PASSED=false"
    )
)

echo.
echo === Generating Stage 2 Build Report ===

:: Generate detailed comparison report
for /f "delims=" %%i in ('git rev-parse --short HEAD 2^>nul') do set GIT_COMMIT=%%i
for /f "delims=" %%i in ('go version 2^>nul') do set GO_VERSION=%%i

echo Build Time: %DATE% %TIME% > "!ROOT_DIR!\artifacts\selfhost\stage2-comparison-report.txt"
echo Git Commit: %GIT_COMMIT% >> "!ROOT_DIR!\artifacts\selfhost\stage2-comparison-report.txt"
echo Go Version: %GO_VERSION% >> "!ROOT_DIR!\artifacts\selfhost\stage2-comparison-report.txt"
echo Build Method: Orizon self-compilation (Stage 2) >> "!ROOT_DIR!\artifacts\selfhost\stage2-comparison-report.txt"
echo Stage 1 Tools Used: Yes >> "!ROOT_DIR!\artifacts\selfhost\stage2-comparison-report.txt"
echo Binary Comparison: !COMPARISON_PASSED! >> "!ROOT_DIR!\artifacts\selfhost\stage2-comparison-report.txt"
echo. >> "!ROOT_DIR!\artifacts\selfhost\stage2-comparison-report.txt"

echo === Binary Comparison Details === >> "!ROOT_DIR!\artifacts\selfhost\stage2-comparison-report.txt"
for %%f in (orizon.exe orizon-compiler.exe orizon-bootstrap.exe orizon-fmt.exe orizon-lsp.exe orizon-test.exe orizon-pkg.exe) do (
    fc /b "!ROOT_DIR!\build\stage1\%%f" "!ROOT_DIR!\build\stage2\%%f" > nul 2>&1
    if !ERRORLEVEL! equ 0 (
        echo %%f: IDENTICAL >> "!ROOT_DIR!\artifacts\selfhost\stage2-comparison-report.txt"
    ) else (
        echo %%f: DIFFERENT >> "!ROOT_DIR!\artifacts\selfhost\stage2-comparison-report.txt"
    )
)

echo.
if "!COMPARISON_PASSED!"=="true" (
    echo === Stage 2 Self-hosting Completed Successfully ===
    echo ✅ All binaries are identical between Stage 1 and Stage 2
    echo ✅ Reproducible build achieved
    echo ✅ Self-hosting verification complete
    echo.
    echo Stage 2 artifacts and reports available in: !ROOT_DIR!\artifacts\selfhost
    exit /b 0
) else (
    echo === Stage 2 Self-hosting Failed ===
    echo ❌ Binary differences detected between Stage 1 and Stage 2
    echo ❌ Reproducible build not achieved
    echo.
    echo Check comparison report: !ROOT_DIR!\artifacts\selfhost\stage2-comparison-report.txt
    exit /b 1
)
