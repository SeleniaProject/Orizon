# Complete struct literal field order fix script
param(
    [string]$RootPath = "C:\Users\Aqua\Programming\SeleniaProject\Orizon"
)

Write-Host "Comprehensive struct literal field order fixes..." -ForegroundColor Yellow

function Fix-StructFieldOrder {
    param(
        [string]$FilePath,
        [string]$StructPattern,
        [string]$CorrectOrder
    )
    
    if (Test-Path $FilePath) {
        $content = Get-Content $FilePath -Raw
        $updated = $content -replace $StructPattern, $CorrectOrder
        if ($content -ne $updated) {
            $updated | Set-Content $FilePath -Encoding UTF8
            Write-Host "Fixed struct literals in $FilePath" -ForegroundColor Green
        }
    }
}

# Fix lexer macro_test.go
Fix-StructFieldOrder "$RootPath\internal\lexer\macro_test.go" `
    "(\s+)expectedValue\s+string\s+expectedType\s+TokenType" `
    "`$1expectedType TokenType`n`$1expectedValue string"

# Fix types exception_effects_test.go - ExceptionSeverity
Fix-StructFieldOrder "$RootPath\internal\types\exception_effects_test.go" `
    "(\s+)expected\s+string\s+severity\s+ExceptionSeverity" `
    "`$1severity ExceptionSeverity`n`$1expected string"

# Fix parser hir_test.go - HIRExpressionKind  
Fix-StructFieldOrder "$RootPath\internal\parser\hir_test.go" `
    "(\s+)exprData\s+interface\{\}\s+name\s+string\s+exprKind\s+HIRExpressionKind" `
    "`$1name string`n`$1exprKind HIRExpressionKind`n`$1exprData interface{}"

# Additional fixes for any remaining issues
$files = Get-ChildItem -Path "$RootPath\internal" -Include "*_test.go" -Recurse

foreach ($file in $files) {
    $content = Get-Content $file.FullName -Raw -ErrorAction SilentlyContinue
    if ($content) {
        $updated = $content
        
        # Generic patterns - fix common struct literal issues
        $updated = $updated -replace '(\s+)expected\s+string\s+(\w+)\s+(\w+Type|\w+Kind)', '$1$2 $3$4expected string'
        $updated = $updated -replace '(\s+)expectedValue\s+string\s+expectedType\s+(\w+)', '$1expectedType $2$3expectedValue string'
        $updated = $updated -replace '(\s+)(\w+Data)\s+interface\{\}\s+name\s+string\s+(\w+Kind)\s+(\w+)', '$1name string$2$3 $4$5$2 interface{}'
        
        if ($content -ne $updated) {
            $updated | Set-Content $file.FullName -Encoding UTF8
            Write-Host "Applied generic fixes to $($file.Name)" -ForegroundColor Cyan
        }
    }
}

Write-Host "Comprehensive struct literal fixes completed!" -ForegroundColor Green
