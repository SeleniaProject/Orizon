# Fix specific corrupted line 330 in pipeline.go
$filePath = "C:\Users\Aqua\Programming\SeleniaProject\Orizon\internal\codegen\pipeline.go"

# Read file as bytes to handle UTF-8 properly
$bytes = [System.IO.File]::ReadAllBytes($filePath)
$content = [System.Text.Encoding]::UTF8.GetString($bytes)

# Find and replace the specific problematic line
$oldPattern = '\t// 2\.0\) 蜊倬・\+x 縺ｯ諱堤ｭ会ｼ医◎縺ｮ縺ｾ縺ｾ荳九ｍ縺呻ｼ・\tif ue, ok := e\.\(\*hir\.HIRUnaryExpression\); ok && ue\.Operator == "\+" \{'
$newPattern = "`t// 2.0) Unary +x is a no-op (pass through as-is)`n`tif ue, ok := e.(*hir.HIRUnaryExpression); ok && ue.Operator == `"+" {"

$content = $content -replace [regex]::Escape($oldPattern), $newPattern

# Write back
[System.IO.File]::WriteAllText($filePath, $content, [System.Text.Encoding]::UTF8)

Write-Host "Fixed line 330 in pipeline.go"
