# PowerShell script to fix all remaining struct literal field order issues
param(
    [string]$RootPath = "C:\Users\Aqua\Programming\SeleniaProject\Orizon"
)

Write-Host "Fixing struct literal field order issues..." -ForegroundColor Yellow

# Fix intrinsics_test.go - more struct literals
$intrinsicsTest = "$RootPath\internal\intrinsics\intrinsics_test.go"
if (Test-Path $intrinsicsTest) {
    $content = Get-Content $intrinsicsTest -Raw
    
    # Fix PlatformSupport struct literals
    $content = $content -replace 'platform\s+string\s+expected\s+PlatformSupport', 'expected PlatformSupport platform string'
    $content = $content -replace '\{PlatformAll,\s*"all"\}', '{PlatformAll, "all"}'
    $content = $content -replace '\{PlatformX64,\s*"x64"\}', '{PlatformX64, "x64"}'
    $content = $content -replace '\{PlatformARM64,\s*"arm64"\}', '{PlatformARM64, "arm64"}'
    
    # Fix CallingConvention struct literals
    $content = $content -replace 'convention\s+string\s+expected\s+CallingConvention', 'expected CallingConvention convention string'
    $content = $content -replace '\{CallingC,\s*"C"\}', '{CallingC, "C"}'
    $content = $content -replace '\{CallingStdcall,\s*"stdcall"\}', '{CallingStdcall, "stdcall"}'
    
    $content | Set-Content $intrinsicsTest -Encoding UTF8
    Write-Host "Fixed $intrinsicsTest" -ForegroundColor Green
}

# Fix lexer_test.go
$lexerTest = "$RootPath\internal\lexer\lexer_test.go"
if (Test-Path $lexerTest) {
    $content = Get-Content $lexerTest -Raw
    
    # Fix TokenType struct literals - reverse the field order in struct definition
    $content = $content -replace 'expectedType\s+string\s+tokenType\s+TokenType', 'tokenType TokenType expectedType string'
    
    $content | Set-Content $lexerTest -Encoding UTF8
    Write-Host "Fixed $lexerTest" -ForegroundColor Green
}

# Fix modules_test.go - Version struct literals
$modulesTest = "$RootPath\internal\modules\modules_test.go"
if (Test-Path $modulesTest) {
    $content = Get-Content $modulesTest -Raw
    
    # Replace all Version{x, y, z, "", ""} with proper field names
    $content = $content -replace 'Version\{(\d+),\s*(\d+),\s*(\d+),\s*"([^"]*)",\s*"([^"]*)"\}', 'Version{PreRelease: "$4", BuildMetadata: "$5", Major: $1, Minor: $2, Patch: $3}'
    
    $content | Set-Content $modulesTest -Encoding UTF8
    Write-Host "Fixed $modulesTest" -ForegroundColor Green
}

# Fix types effects_test.go
$effectsTest = "$RootPath\internal\types\effects_test.go"
if (Test-Path $effectsTest) {
    $content = Get-Content $effectsTest -Raw
    
    # Fix EffectKind struct literals
    $content = $content -replace 'effectKind\s+string\s+expected\s+EffectKind', 'expected EffectKind effectKind string'
    
    $content | Set-Content $effectsTest -Encoding UTF8
    Write-Host "Fixed $effectsTest" -ForegroundColor Green
}

# Fix parser hir_test.go
$hirTest = "$RootPath\internal\parser\hir_test.go"
if (Test-Path $hirTest) {
    $content = Get-Content $hirTest -Raw
    
    # Fix HIRTypeKind struct literals  
    $content = $content -replace 'expectedKind\s+string\s+hirType\s+HIRTypeKind', 'hirType HIRTypeKind expectedKind string'
    
    $content | Set-Content $hirTest -Encoding UTF8
    Write-Host "Fixed $hirTest" -ForegroundColor Green
}

Write-Host "Struct literal fixes completed!" -ForegroundColor Cyan
