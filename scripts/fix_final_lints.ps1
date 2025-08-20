# Final lint fixes script

Write-Host "Fixing final lint violations..."

# Fix testpackage violations
$testFiles = @(
    "internal\typechecker\trait_resolver_test.go",
    "internal\typechecker\type_inference_test.go"
)

foreach ($file in $testFiles) {
    if (Test-Path $file) {
        Write-Host "Fixing testpackage in $file"
        $content = Get-Content $file -Raw
        $content = $content -replace "^package typechecker", "package typechecker_test"
        Set-Content $file $content -NoNewline
    }
}

Write-Host "Fixed testpackage violations"

# Remove unused fields and functions from trait_resolver.go
Write-Host "Removing unused code from trait_resolver.go"

# Fix type_inference.go issues by creating separate internal types file
Write-Host "Creating separate types file to reduce public structs"

# The main issues will be fixed by manual edits
Write-Host "Script completed. Manual fixes still needed."
