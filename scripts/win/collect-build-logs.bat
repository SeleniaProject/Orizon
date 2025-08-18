@echo off
setlocal enabledelayedexpansion

:: Get script directory and set root
set "ROOT_DIR=%~dp0"
for %%i in ("!ROOT_DIR!..\..") do set "ROOT_DIR=%%~fi"

echo === Orizon Build Log Collection ===
echo Collecting diagnostic information...

set "LOG_DIR=!ROOT_DIR!\artifacts\build-logs"
if not exist "!LOG_DIR!" mkdir "!LOG_DIR!"

set "TIMESTAMP=%DATE:~10,4%-%DATE:~4,2%-%DATE:~7,2%_%TIME:~0,2%-%TIME:~3,2%-%TIME:~6,2%"
set "TIMESTAMP=!TIMESTAMP: =0!"

set "LOG_BASE=!LOG_DIR!\build-log-!TIMESTAMP!"

echo Timestamp: !TIMESTAMP! > "!LOG_BASE!.txt"
echo Root Directory: !ROOT_DIR! >> "!LOG_BASE!.txt"
echo Current Directory: %CD% >> "!LOG_BASE!.txt"
echo.>> "!LOG_BASE!.txt"

echo === System Information === >> "!LOG_BASE!.txt"
systeminfo | findstr /C:"OS Name" /C:"OS Version" /C:"System Type" >> "!LOG_BASE!.txt"
echo.>> "!LOG_BASE!.txt"

echo === Go Environment === >> "!LOG_BASE!.txt"
go version >> "!LOG_BASE!.txt" 2>&1
go env GOOS GOARCH GOROOT GOPATH >> "!LOG_BASE!.txt" 2>&1
echo.>> "!LOG_BASE!.txt"

echo === Git Information === >> "!LOG_BASE!.txt"
git rev-parse --short HEAD >> "!LOG_BASE!.txt" 2>&1
git status --porcelain >> "!LOG_BASE!.txt" 2>&1
echo.>> "!LOG_BASE!.txt"

echo === Build Directory Contents === >> "!LOG_BASE!.txt"
if exist "!ROOT_DIR!\build" (
    dir "!ROOT_DIR!\build" >> "!LOG_BASE!.txt" 2>&1
) else (
    echo Build directory does not exist >> "!LOG_BASE!.txt"
)
echo.>> "!LOG_BASE!.txt"

echo === Test Output === >> "!LOG_BASE!.txt"
cd "!ROOT_DIR!"
go test ./... >> "!LOG_BASE!.txt" 2>&1
echo.>> "!LOG_BASE!.txt"

echo === Build All Tools === >> "!LOG_BASE!.txt"
make build-all >> "!LOG_BASE!.txt" 2>&1
echo.>> "!LOG_BASE!.txt"

echo === Tool Versions === >> "!LOG_BASE!.txt"
for %%t in (orizon.exe orizon-compiler.exe orizon-fmt.exe orizon-lsp.exe orizon-test.exe orizon-pkg.exe) do (
    if exist "!ROOT_DIR!\build\%%t" (
        echo Testing %%t: >> "!LOG_BASE!.txt"
        "!ROOT_DIR!\build\%%t" --version >> "!LOG_BASE!.txt" 2>&1
        echo. >> "!LOG_BASE!.txt"
    ) else (
        echo %%t not found >> "!LOG_BASE!.txt"
    )
)

echo === Recent Error Logs === >> "!LOG_BASE!.txt"
if exist "!ROOT_DIR!\tmp" (
    echo Tmp directory contents: >> "!LOG_BASE!.txt"
    dir "!ROOT_DIR!\tmp\*.err" "!ROOT_DIR!\tmp\*.log" >> "!LOG_BASE!.txt" 2>&1
    echo. >> "!LOG_BASE!.txt"
    
    :: Include recent error files content
    for %%f in ("!ROOT_DIR!\tmp\*.err" "!ROOT_DIR!\tmp\*.log") do (
        if exist "%%f" (
            echo === Content of %%~nxf === >> "!LOG_BASE!.txt"
            type "%%f" >> "!LOG_BASE!.txt" 2>&1
            echo. >> "!LOG_BASE!.txt"
        )
    )
)

echo Log collection completed: !LOG_BASE!.txt
echo Use this file for troubleshooting build issues.

exit /b 0
