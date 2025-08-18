@echo off
REM Self-hosting Stage 0: Build Orizon using Go implementation
REM This script builds the Orizon compiler and tools using the current Go implementation

setlocal enabledelayedexpansion

echo === Orizon Self-hosting Stage 0 ===
echo Building Orizon using Go implementation...

REM Set up environment
set "SCRIPT_DIR=%~dp0"
for %%i in ("%SCRIPT_DIR%\..\..") do set "ROOT_DIR=%%~fi"
set "BUILD_DIR=%ROOT_DIR%\build\stage0"
set "ARTIFACTS_DIR=%ROOT_DIR%\artifacts\selfhost"

echo Root directory: %ROOT_DIR%
echo Build directory: %BUILD_DIR%
echo Artifacts directory: %ARTIFACTS_DIR%

REM Clean and create build directories
if exist "%BUILD_DIR%" rmdir /s /q "%BUILD_DIR%"
mkdir "%BUILD_DIR%"
if not exist "%ARTIFACTS_DIR%" mkdir "%ARTIFACTS_DIR%"

cd /d "%ROOT_DIR%"

REM Build all Orizon tools using Go
echo.
echo === Building Orizon Tools ===

set tools=orizon orizon-compiler orizon-bootstrap orizon-fmt orizon-lsp orizon-test orizon-pkg

for %%t in (%tools%) do (
    echo Building %%t...
    go build -o "%BUILD_DIR%\%%t.exe" ".\cmd\%%t"
    if !errorlevel! equ 0 (
        echo ✓ %%t built successfully
    ) else (
        echo ✗ Failed to build %%t
        exit /b 1
    )
)

REM Run basic validation tests
echo.
echo === Validating Built Tools ===

REM Test orizon-fmt
echo Testing orizon-fmt...
echo fn main() { let x = 1 + 2; } | "%BUILD_DIR%\orizon-fmt.exe" -stdin
if !errorlevel! equ 0 (
    echo ✓ orizon-fmt validation passed
) else (
    echo ✗ orizon-fmt validation failed
    exit /b 1
)

REM Test orizon-test
echo Testing orizon-test...
"%BUILD_DIR%\orizon-test.exe" --help >nul
if !errorlevel! equ 0 (
    echo ✓ orizon-test validation passed
) else (
    echo ✗ orizon-test validation failed
    exit /b 1
)

REM Test orizon-pkg
echo Testing orizon-pkg...
"%BUILD_DIR%\orizon-pkg.exe" help >nul
if !errorlevel! equ 0 (
    echo ✓ orizon-pkg validation passed
) else (
    echo ✗ orizon-pkg validation failed
    exit /b 1
)

REM Generate build metadata
echo.
echo === Generating Build Metadata ===

for /f "tokens=1-4 delims=/ " %%a in ('date /t') do set BUILD_DATE=%%d-%%a-%%b
for /f "tokens=1-2 delims=: " %%a in ('time /t') do set BUILD_TIME=%%a:%%b
set BUILD_TIMESTAMP=%BUILD_DATE%T%BUILD_TIME%:00Z

for /f "tokens=*" %%a in ('git rev-parse HEAD 2^>nul') do set GIT_COMMIT=%%a
if "!GIT_COMMIT!"=="" set GIT_COMMIT=unknown

for /f "tokens=*" %%a in ('go version') do set GO_VERSION=%%a

REM Create metadata JSON
(
echo {
echo   "stage": 0,
echo   "description": "Orizon built using Go implementation",
echo   "build_time": "%BUILD_TIMESTAMP%",
echo   "git_commit": "%GIT_COMMIT%",
echo   "go_version": "%GO_VERSION%",
echo   "built_tools": [
echo     "orizon",
echo     "orizon-compiler",
echo     "orizon-bootstrap", 
echo     "orizon-fmt",
echo     "orizon-lsp",
echo     "orizon-test",
echo     "orizon-pkg"
echo   ],
echo   "build_directory": "%BUILD_DIR%",
echo   "artifacts_directory": "%ARTIFACTS_DIR%"
echo }
) > "%ARTIFACTS_DIR%\stage0-metadata.json"

REM Copy tools to artifacts directory
echo Copying tools to artifacts directory...
copy "%BUILD_DIR%\*.exe" "%ARTIFACTS_DIR%\" >nul

REM Run comprehensive tests
echo.
echo === Running Comprehensive Tests ===

REM Run Go tests
echo Running Go test suite...
go test .\...
if !errorlevel! equ 0 (
    echo ✓ Go test suite passed
) else (
    echo ✗ Go test suite failed
    exit /b 1
)

REM Generate test report
"%BUILD_DIR%\orizon-test.exe" -packages ".\..." -json > "%ARTIFACTS_DIR%\stage0-test-results.json" 2>nul || echo Test results generation completed

REM Create summary
echo.
echo === Stage 0 Build Summary ===
echo Build completed successfully!
echo Tools built: 7
echo Build artifacts: %ARTIFACTS_DIR%
echo Build time: %BUILD_TIMESTAMP%
echo Git commit: %GIT_COMMIT%

dir "%ARTIFACTS_DIR%"

echo.
echo Stage 0 completed. Ready for Stage 1 (self-compilation).

endlocal
