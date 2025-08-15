param(
    [switch]$DryRun
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Should-SkipFile($path) {
    $name = [System.IO.Path]::GetFileName($path)
    if ($name -match '\.exe$') { return $true }
    if ($name -eq 'junit_local.xml') { return $true }
    if ($name -eq 'output.txt') { return $true }
    return $false
}

function Guess-ScopeAndType($path, [string]$status) {
    $type = 'chore'
    $scope = ''
    if ($path -like '.github/*' -or $path -eq 'Makefile' -or $path -eq 'go.mod' -or $path -eq 'go.sum') { $type='build' }
    if ($path -like 'scripts/*') { $type='chore'; $scope='scripts' }
    if ($path -like 'cmd/*') { $type='tool' }
    if ($path -like 'internal/runtime/asyncio/*') { $type='runtime'; $scope='asyncio' }
    if ($path -like 'internal/testrunner/*' -or $path -like '*_test.go') { $type='test' }
    if ($path -like 'spec/*' -or $path -like '*.md') { $type='docs' }

    $basename = [System.IO.Path]::GetFileName($path)
    if ([string]::IsNullOrWhiteSpace($scope)) { $msg = "$type: update $basename" }
    else { $msg = "$type($scope): update $basename" }

    if ($status -eq '??') {
        if ($type -eq 'docs') { $msg = "docs: add $basename" }
        elseif ($type -eq 'test') { $msg = "test: add $basename" }
        elseif ($type -eq 'tool') { $msg = "feat(tool): add $basename" }
        elseif ($type -eq 'runtime') { $msg = "feat($scope): add $basename" }
        else { $msg = "feat: add $basename" }
    }
    elseif ($status -like '*D*') {
        $msg = "chore: remove $basename"
    }
    return $msg
}

function Commit-Path($status, $path) {
    if (Should-SkipFile $path) { Write-Host "skip $path"; return }
    $msg = Guess-ScopeAndType -path $path -status $status

    if ($status -like '*D*') {
        $cmd = "git rm -- "$path""
        if ($DryRun) { Write-Host "DRY $cmd" }
        else { git rm -- "$path" | Out-Null }
    } else {
        $cmd = "git add -- "$path""
        if ($DryRun) { Write-Host "DRY $cmd" }
        else { git add -- "$path" | Out-Null }
    }

    if ($DryRun) {
        Write-Host "DRY git commit -m '$msg'"
    } else {
        git commit -m "$msg" | Out-Null
        Write-Host "committed: $msg"
    }
}

# Ensure predictable paths
& git -c core.quotepath=off status --porcelain=v1 | ForEach-Object {
    if ($_ -match '^(..)[ ](.+)$') {
        $st = $matches[1]
        $pth = $matches[2]
        Commit-Path -status $st -path $pth
    }
}
