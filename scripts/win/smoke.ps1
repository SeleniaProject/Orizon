$ErrorActionPreference = 'Stop'

# Change to repo root if invoked from elsewhere
Set-Location -Path (Resolve-Path "$PSScriptRoot\..\..")

$artifacts = "artifacts"
if (-not (Test-Path $artifacts)) { New-Item -ItemType Directory -Path $artifacts | Out-Null }

function Exec([string]$cmd) {
    Write-Host "==> $cmd"
    Invoke-Expression $cmd
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed: $cmd"
    }
}

# 1) Build CLIs
Exec 'go build -o orizon-test.exe ./cmd/orizon-test'
Exec 'go build -o orizon-fuzz.exe ./cmd/orizon-fuzz'
Exec 'go build -o orizon-repro.exe ./cmd/orizon-repro'
Exec 'go build -o orizon-summary.exe ./cmd/orizon-summary'
Exec 'go build -o orizon-mockgen.exe ./cmd/orizon-mockgen'

# 2) Unit tests (short, fail-fast with retries) + JUnit/JSON summary
Exec ".\orizon-test.exe --packages ./... --short --retries 1 --fail-fast --junit $artifacts/junit.xml --json-summary $artifacts/junit_summary.json --color=false"

# 2.1) Go test coverage (optional)
try {
    go test -coverprofile "$artifacts/cover.out" ./... | Out-Host
    go tool cover -func "$artifacts/cover.out" | Out-File -FilePath "$artifacts/cover.txt" -Encoding utf8
} catch {}

# 3) Fuzz smoke runs (2s each). Use corpus when available
function HasFile([string]$p) { return Test-Path $p }

$parserCorpus = 'corpus/parser_corpus.txt'
$lexerCorpus = 'corpus/lexer_corpus.txt'
$abCorpus = 'corpus/astbridge_corpus.txt'

$parserCmd = ".\orizon-fuzz.exe --target parser --duration 2s --p 2 --max-execs 4000 --cov-mode trigram --stats --json-stats $artifacts/stats_parser.json --save-seed $artifacts/seed.txt --out $artifacts/crashes.txt --min-on-crash --min-dir $artifacts/crashes_min --min-budget 2s"
if (HasFile $parserCorpus) { $parserCmd += " --corpus $parserCorpus --covout $artifacts/parser.cov --covstats" }
Exec $parserCmd

$parserLaxCmd = ".\orizon-fuzz.exe --target parser-lax --duration 2s --p 2 --max-execs 3000 --stats --json-stats $artifacts/stats_parser_lax.json"
if (HasFile $parserCorpus) { $parserLaxCmd += " --corpus $parserCorpus" }
Exec $parserLaxCmd

$lexerCmd = ".\orizon-fuzz.exe --target lexer --duration 2s --p 2 --max-execs 3000 --stats"
if (HasFile $lexerCorpus) { $lexerCmd += " --corpus $lexerCorpus" }
Exec $lexerCmd

$abCmd = ".\orizon-fuzz.exe --target astbridge --duration 2s --p 2 --max-execs 3000 --stats"
if (HasFile $abCorpus) { $abCmd += " --corpus $abCorpus" }
Exec $abCmd

$hirCmd = ".\orizon-fuzz.exe --target hir --duration 2s --p 2 --max-execs 3000 --stats"
if (HasFile $parserCorpus) { $hirCmd += " --corpus $parserCorpus" }
Exec $hirCmd

$abHirCmd = ".\orizon-fuzz.exe --target astbridge-hir --duration 2s --p 2 --max-execs 3000 --stats"
if (HasFile $abCorpus) { $abHirCmd += " --corpus $abCorpus" }
Exec $abHirCmd

# 4) Reproduce and minimize last crash if any
if (Test-Path "$artifacts/crashes.txt") {
    $lines = Get-Content "$artifacts/crashes.txt" -ErrorAction SilentlyContinue | Where-Object { $_.Trim() -ne '' }
    if ($lines.Count -gt 0) {
        Exec ".\orizon-repro.exe --log $artifacts/crashes.txt --out $artifacts/minimized.bin --budget 2s --target parser"
    } else {
        Write-Host "No non-empty crashes in $artifacts/crashes.txt"
    }
}

# 5) Compose summary
try {
    .\orizon-summary.exe --junit-summary "$artifacts/junit_summary.json" --stats "$artifacts/stats_parser.json,$artifacts/stats_parser_lax.json" --cover "$artifacts/cover.txt" --out "$artifacts/summary.md" | Out-Host
    if ($env:GITHUB_STEP_SUMMARY -and (Test-Path "$artifacts/summary.md")) {
        Get-Content "$artifacts/summary.md" | Out-File -FilePath $env:GITHUB_STEP_SUMMARY -Append -Encoding utf8
    }
    if ($env:GITHUB_STEP_SUMMARY -and (Test-Path "$artifacts/cover.txt")) {
        Add-Content -Path $env:GITHUB_STEP_SUMMARY -Value "`n#### Coverage"
        Get-Content "$artifacts/cover.txt" | Select-Object -Last 1 | ForEach-Object { Add-Content -Path $env:GITHUB_STEP_SUMMARY -Value ("- " + $_) }
        Add-Content -Path $env:GITHUB_STEP_SUMMARY -Value ""
    }
} catch {}

# 6) Optional: IOCP experimental unit tests on Windows
if ($env:OS -match 'Windows') {
    $env:ORIZON_WIN_IOCP = '1'
    try {
        Write-Host "Running IOCP experimental tests..."
        go test -tags iocp ./internal/runtime/asyncio -run IOCPPoller -v | Out-Host
    } catch {
        Write-Warning "IOCP tests failed: $_"
    }
}

Write-Host "\nAll smoke steps completed. Artifacts in '$artifacts'."


