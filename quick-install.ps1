# Orizon OS - 簡単インストール（管理者権限で実行）

Write-Host "Orizon OS開発環境をインストール中..." -ForegroundColor Blue
Write-Host ""

# Chocolateyインストール（既にある場合はスキップ）
if (!(Get-Command choco -ErrorAction SilentlyContinue)) {
    Write-Host "Chocolateyをインストール中..." -ForegroundColor Yellow
    Set-ExecutionPolicy Bypass -Scope Process -Force
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
    iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
}

# 必要なツールをインストール
Write-Host "NASMをインストール中..." -ForegroundColor Yellow
choco install nasm -y

Write-Host "QEMUをインストール中..." -ForegroundColor Yellow
choco install qemu -y

Write-Host "Gitをインストール中..." -ForegroundColor Yellow
choco install git -y

# 環境変数リフレッシュ
refreshenv

Write-Host ""
Write-Host "インストール完了！" -ForegroundColor Green
Write-Host "新しいPowerShellを開いて以下を実行してください：" -ForegroundColor Blue
Write-Host "  .\build.ps1 build" -ForegroundColor White
Write-Host "  .\build.ps1 run" -ForegroundColor White
Write-Host ""
pause
