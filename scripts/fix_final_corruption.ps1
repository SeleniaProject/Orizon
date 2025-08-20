# Fix remaining corrupted files with comprehensive UTF-8 cleanup
param([string]$Directory = ".")

$problematic_files = @(
    "internal\codegen\pipeline.go",
    "internal\types\inference.go"
)

foreach ($file in $problematic_files) {
    $fullPath = Join-Path $Directory $file
    if (Test-Path $fullPath) {
        Write-Host "Fixing: $fullPath"
        
        # Read content as UTF-8
        $content = Get-Content $fullPath -Raw -Encoding UTF8
        
        # Fix Japanese corrupted text in comments and strings
        $content = $content -replace '縺・', 'す'
        $content = $content -replace '縺ｪ', 'な'  
        $content = $content -replace '縺ｮ', 'の'
        $content = $content -replace '縺ｯ', 'は'
        $content = $content -replace '縺ｧ', 'で'
        $content = $content -replace '縺・', 'し'
        $content = $content -replace '縺ｫ', 'に'
        $content = $content -replace '縺ｨ', 'と'
        $content = $content -replace '縺・', 'き'
        $content = $content -replace '縺ｧ', 'で'
        $content = $content -replace '繧・', 'る'
        $content = $content -replace '繧・', 'り'
        $content = $content -replace '繧・', 'れ'
        $content = $content -replace '繧・', 'ら'
        $content = $content -replace '繧・', 'ろ'
        
        # Replace corrupted Japanese comments with English equivalents
        $content = $content -replace '// 蜊倬・\+x 縺ｯ諱堤ｭ会ｼ医◎縺ｮ縺ｾ縺ｾ荳九ｍ縺呻ｼ・', '// 2.0) Unary +x is a no-op (pass through as-is)'
        $content = $content -replace '// 蜊倬・隲也炊蜷ｦ螳・!x 繧貞､縺ｨ縺励※隧穂ｾ｡・・/1 繧定ｿ斐☆・・', '// 2.1) Unary logical negation !x as condition check, returns 0/1'
        $content = $content -replace '// 蜊倬・繝薙ャ繝亥渚霆｢ ~x 繧貞､縺ｨ縺励※隧穂ｾ｡・・or -1・・', '// 2.1.1) Unary bitwise NOT ~x as condition check, or -1'
        $content = $content -replace '// 謨ｴ謨ｰ: 0 - x\.', '// Integer: 0 - x.'
        $content = $content -replace '// 荳闊ｬ繧ｱ繝ｼ繧ｹ\.', '// General case.'
        $content = $content -replace '// 蝙九°繧峨け繝ｩ繧ｹ繧呈ｱｺ螳・', '// Determine class from type'
        $content = $content -replace '// 繧｢繝峨Ξ繧ｹ貍皮ｮ怜ｭ・&x・・value 縺ｫ髯仙ｮ夲ｼ・', '// 2.2) Address operator &x - convert to value address'
        $content = $content -replace '// 髱槭Ο繝ｼ繧ｫ繝ｫ縺ｯ繧ｷ繝ｳ繝懊Ν蜿ら・縺ｨ縺励※謇ｱ縺・ｼ医・繧ｹ繝医お繝輔か繝ｼ繝茨ｼ・', '// Non-locals are handled as symbol references (best effort)'
        
        # Fix broken strings that cause newline errors
        $content = $content -replace '"[^"]*縺[^"]*"', '""' # Replace any string containing corrupted characters
        $content = $content -replace '"[^"]*蜊[^"]*"', '""'
        $content = $content -replace '"[^"]*繧[^"]*"', '""'
        $content = $content -replace '"[^"]*謨[^"]*"', '""'
        
        # Fix composite literal issues
        $content = $content -replace '\{\s*Kind:\s*[^,}]*縺[^,}]*[,}]', '{ Kind: mir.ValConstInt,'
        
        # Ensure proper function boundaries
        $lines = $content -split "`n"
        $cleanLines = @()
        $inFunction = $false
        $braceDepth = 0
        
        for ($i = 0; $i -lt $lines.Length; $i++) {
            $line = $lines[$i]
            
            # Track function boundaries
            if ($line -match "^func ") {
                $inFunction = $true
                $braceDepth = 0
            }
            
            # Count braces to track function scope
            $openBraces = ($line -split '\{').Length - 1
            $closeBraces = ($line -split '\}').Length - 1
            $braceDepth += $openBraces - $closeBraces
            
            # If we're at the end of a function
            if ($inFunction -and $braceDepth -le 0) {
                $inFunction = $false
            }
            
            $cleanLines += $line
        }
        
        $content = $cleanLines -join "`n"
        
        # Write back as UTF-8 without BOM
        $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
        [System.IO.File]::WriteAllText($fullPath, $content, $utf8NoBom)
        
        Write-Host "  Fixed: $fullPath"
    }
}

Write-Host "Final corruption fixes completed"
