# Fix corrupted Japanese comments in pipeline.go
$filePath = "C:\Users\Aqua\Programming\SeleniaProject\Orizon\internal\codegen\pipeline.go"
$content = Get-Content $filePath -Raw -Encoding UTF8

# Fix line 330 - split corrupted comment from code
$content = $content -replace '// 2\.0\) [^`]*if ue, ok := e\.\(\*hir\.HIRUnaryExpression\); ok && ue\.Operator == "\+" \{', @'
	// 2.0) Unary +x is a no-op (pass through as-is)
	if ue, ok := e.(*hir.HIRUnaryExpression); ok && ue.Operator == "+" {
'@

# Fix line 333
$content = $content -replace '// 2\.1\) [^`]*if ue, ok := e\.\(\*hir\.HIRUnaryExpression\); ok && ue\.Operator == "!" \{', @'
	// 2.1) Unary logical negation !x as condition check, returns 0/1
	if ue, ok := e.(*hir.HIRUnaryExpression); ok && ue.Operator == "!" {
'@

# Fix line 336
$content = $content -replace '// 2\.1\.1\) [^`]*if ue, ok := e\.\(\*hir\.HIRUnaryExpression\); ok && ue\.Operator == "~" \{', @'
	// 2.1.1) Unary bitwise negation ~x as condition check, returns 0 or -1
	if ue, ok := e.(*hir.HIRUnaryExpression); ok && ue.Operator == "~" {
'@

# Fix other corrupted comments
$content = $content -replace '// lowerHIRExpr [^`]*', '// lowerHIRExpr processes MIR calls according to necessity and handles value return condition checks.'
$content = $content -replace '// [^`]*髱槭Ο[^`]*', '// Local variables are handled as symbol references (best effort)'
$content = $content -replace '// [^`]*隲也炊貍[^`]*', '// Logical evaluation constants generated as 0/1 values in value context'
$content = $content -replace '// newTemp [^`]*荳ｭ縺ｧ[^`]*', '// newTemp is used within calls and uses separate counters to avoid conflicts with other counters'
$content = $content -replace '// [^`]*蛟､縺ｨ縺励※[^`]*', '// As a value, the general case Class is set'
$content = $content -replace '// 6\) [^`]*繧ｭ繝｣繧ｹ繝[^`]*', '// 6) Cast: pass values through as-is'
$content = $content -replace '// [^`]*繝吶せ繝[^`]*', '// Best effort: Local variables return symbol references as addresses'

# Write back with UTF8 encoding
$content | Out-File $filePath -Encoding UTF8 -NoNewline

Write-Host "Fixed corrupted comments in pipeline.go"
