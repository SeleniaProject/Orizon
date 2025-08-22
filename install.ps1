# Orizon OS - 自動セットアップスクリプト
# NASMとQEMUを自動でインストールします

param(
    [switch]$Force
)

# 管理者権限チェック
if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Host "管理者権限が必要です。PowerShellを管理者として実行してください。" -ForegroundColor Red
    Write-Host "右クリック -> [管理者として実行] を選択してください。" -ForegroundColor Yellow
    pause
    exit 1
}

function Write-Status {
    param([string]$Message)
    Write-Host $Message -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-Error {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor Red
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠ $Message" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Blue
Write-Host "    Orizon OS - 自動セットアップ     " -ForegroundColor Blue
Write-Host "========================================" -ForegroundColor Blue
Write-Host ""

# Chocolateyのインストールチェック
Write-Status "Chocolateyパッケージマネージャーをチェック中..."
if (!(Get-Command choco -ErrorAction SilentlyContinue)) {
    Write-Warning "Chocolateyが見つかりません。インストールします..."
    
    # Chocolateyインストール
    Set-ExecutionPolicy Bypass -Scope Process -Force
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
    Invoke-Expression ((New-Object System.Net.WebClient).DownloadString("https://community.chocolatey.org/install.ps1"))
    
    # PATHリフレッシュ
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
    
    if (Get-Command choco -ErrorAction SilentlyContinue) {
        Write-Success "Chocolateyインストール完了"
    }
    else {
        Write-Error "Chocolateyのインストールに失敗しました"
        exit 1
    }
}
else {
    Write-Success "Chocolatey既にインストール済み"
}

# NASMのインストール
Write-Status "NASMアセンブラーをインストール中..."
if (Get-Command nasm -ErrorAction SilentlyContinue) {
    if (!$Force) {
        Write-Success "NASM既にインストール済み"
    }
    else {
        Write-Warning "強制再インストール中..."
        choco install nasm -y --force
    }
}
else {
    choco install nasm -y
    if ($LASTEXITCODE -eq 0) {
        Write-Success "NASMインストール完了"
    }
    else {
        Write-Error "NASMのインストールに失敗しました"
    }
}

# QEMUのインストール
Write-Status "QEMUエミュレーターをインストール中..."
if (Get-Command qemu-system-x86_64 -ErrorAction SilentlyContinue) {
    if (!$Force) {
        Write-Success "QEMU既にインストール済み"
    }
    else {
        Write-Warning "強制再インストール中..."
        choco install qemu -y --force
    }
}
else {
    choco install qemu -y
    if ($LASTEXITCODE -eq 0) {
        Write-Success "QEMUインストール完了"
    }
    else {
        Write-Error "QEMUのインストールに失敗しました"
    }
}

# Gitのインストール（必要に応じて）
Write-Status "Gitをチェック中..."
if (!(Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Warning "Gitが見つかりません。インストールします..."
    choco install git -y
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Gitインストール完了"
    }
}
else {
    Write-Success "Git既にインストール済み"
}

# PATHリフレッシュ
Write-Status "環境変数を更新中..."
$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")

# インストール確認
Write-Status "インストール確認中..."
$tools = @{
    "nasm"               = "NASM Assembler"
    "qemu-system-x86_64" = "QEMU Emulator"
    "git"                = "Git"
    "go"                 = "Go Compiler"
}

$allOK = $true
foreach ($tool in $tools.Keys) {
    if (Get-Command $tool -ErrorAction SilentlyContinue) {
        $version = ""
        try {
            switch ($tool) {
                "nasm" { $version = (nasm -v 2>&1 | Select-Object -First 1) }
                "qemu-system-x86_64" { $version = (qemu-system-x86_64 --version 2>&1 | Select-Object -First 1) }
                "git" { $version = (git --version) }
                "go" { $version = (go version) }
            }
        }
        catch {}
        Write-Success "$($tools[$tool]) - $version"
    }
    else {
        Write-Error "$($tools[$tool]) not found"
        $allOK = $false
    }
}

Write-Host ""
if ($allOK) {
    Write-Success "🎉 すべてのツールが正常にインストールされました！"
    Write-Host ""
    Write-Status "次のステップ："
    Write-Host "1. PowerShellを再起動してください" -ForegroundColor White
    Write-Host "2. .\build.ps1 build でOrizon OSをビルド" -ForegroundColor White
    Write-Host "3. .\build.ps1 run でQEMUで実行" -ForegroundColor White
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "  Orizon OS開発環境構築完了！      " -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
}
else {
    Write-Error "一部のツールのインストールに失敗しました"
    Write-Warning "PowerShellを再起動してから再度実行してください"
}

Write-Host ""
Write-Host "このウィンドウを閉じて、新しいPowerShellを開いてください。" -ForegroundColor White
pause
