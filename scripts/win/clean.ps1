$ErrorActionPreference = 'SilentlyContinue'

Set-Location -Path (Resolve-Path "$PSScriptRoot\..\..")

function Rm($p) {
    if (Test-Path $p) { Remove-Item -Recurse -Force $p }
}

# Artifacts and build outputs
Rm artifacts
Rm build

# Fuzz outputs
Rm crashes_min
Rm fuzz.cov
Rm crashes.txt
Rm seed.txt

# Test reports
Rm junit.xml
Rm junit_summary.json
Rm summary.json

# Binaries in repo root (keep source tree clean)
Get-ChildItem -Path . -Filter 'orizon-*.exe' | Remove-Item -Force
Get-ChildItem -Path . -Filter '*.exe' | Where-Object { $_.Name -in @('gdb-rsp-server.exe','numa-test.exe') } | Remove-Item -Force

Write-Host "Workspace cleaned."


