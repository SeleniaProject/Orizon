# Orizon Programming Language - 開発環境セットアップガイド

## 概要

本ガイドでは、Orizonプログラミング言語の開発環境を一から構築し、実際にプロジェクトをビルド・実行するまでの完全なステップバイステップ手順を提供します。新規参加開発者が迅速かつ確実に開発を開始できるよう、詳細な説明とトラブルシューティング情報を含めています。

---

## 前提条件

### ハードウェア要件

#### 最小要件
- **CPU**: x86_64 または ARM64 アーキテクチャ
- **メモリ**: 4GB RAM
- **ストレージ**: 10GB の空き容量
- **ネットワーク**: インターネット接続（依存関係ダウンロード用）

#### 推奨要件
- **CPU**: 4コア以上の現代的プロセッサ
- **メモリ**: 16GB RAM 以上
- **ストレージ**: SSD で 50GB 以上の空き容量
- **ネットワーク**: 高速インターネット接続

### オペレーティングシステム対応

#### Windows
- **Windows 10** (バージョン 1903 以降)
- **Windows 11** (全バージョン)
- **Windows Server 2019/2022**

#### macOS
- **macOS 10.15 Catalina** 以降
- **macOS 11 Big Sur** 以降（推奨）
- **macOS 12 Monterey/13 Ventura/14 Sonoma**（最適）

#### Linux
- **Ubuntu 20.04 LTS / 22.04 LTS / 24.04 LTS**
- **Debian 11 / 12**
- **CentOS 8 / Rocky Linux 8/9**
- **Fedora 36-40**
- **Arch Linux**（最新）

### 必要ソフトウェア

#### 必須ツール
1. **Git** (バージョン 2.25 以降)
2. **Docker** (バージョン 20.10 以降)
3. **Docker Compose** (バージョン 2.0 以降)
4. **Go** (バージョン 1.24.3 以降) - コンパイラビルド用

#### 推奨ツール
1. **Visual Studio Code** (最新版)
2. **PowerShell 7** (Windows の場合)
3. **GNU Make** (Linux/macOS)

---

## ステップ1: 基本ツールのインストール

### Windows でのインストール

#### Git のインストール
```powershell
# Chocolatey を使用（推奨）
choco install git

# または公式インストーラーを使用
# https://git-scm.com/download/win からダウンロード
```

#### Docker Desktop のインストール
```powershell
# 公式サイトからダウンロード・インストール
# https://desktop.docker.com/win/stable/Docker%20Desktop%20Installer.exe

# インストール後、WSL2 バックエンドを有効化
```

#### Go のインストール
```powershell
# Chocolatey を使用
choco install golang

# または公式インストーラー
# https://golang.org/dl/ から最新版をダウンロード
```

#### PowerShell 7 のインストール（推奨）
```powershell
# Windows PowerShell 5.1 で実行
winget install Microsoft.PowerShell

# または GitHub Releases から
# https://github.com/PowerShell/PowerShell/releases
```

### macOS でのインストール

#### Homebrew のインストール（まだの場合）
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

#### 必要ツールのインストール
```bash
# Git（通常は既にインストール済み）
brew install git

# Docker Desktop
brew install --cask docker

# Go
brew install go

# Make（通常は既にインストール済み）
brew install make
```

### Linux (Ubuntu/Debian) でのインストール

#### パッケージマネージャーの更新
```bash
sudo apt update && sudo apt upgrade -y
```

#### Git のインストール
```bash
sudo apt install -y git
```

#### Docker のインストール
```bash
# Docker の公式GPGキーを追加
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

# Docker リポジトリを追加
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Docker をインストール
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# 現在のユーザーを docker グループに追加
sudo usermod -aG docker $USER

# 再ログインまたは以下を実行
newgrp docker
```

#### Go のインストール
```bash
# 公式バイナリをダウンロード・インストール
curl -fsSL https://golang.org/dl/go1.24.3.linux-amd64.tar.gz | sudo tar -C /usr/local -xzf -

# 環境変数を設定
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc
```

#### 追加ツールのインストール
```bash
sudo apt install -y build-essential make curl wget
```

---

## ステップ2: プロジェクトのクローンと設定

### プロジェクトのクローン

```bash
# Orizon リポジトリをクローン
git clone https://github.com/orizon-lang/orizon.git
cd orizon

# Git設定の確認
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"
```

### プロジェクト構造の確認

```bash
# プロジェクト構造を確認
ls -la

# 期待される出力:
# drwxr-xr-x  3 user user  4096 Jan 15 10:00 cmd/
# drwxr-xr-x  3 user user  4096 Jan 15 10:00 internal/
# drwxr-xr-x  3 user user  4096 Jan 15 10:00 examples/
# drwxr-xr-x  3 user user  4096 Jan 15 10:00 docs/
# -rw-r--r--  1 user user  1234 Jan 15 10:00 go.mod
# -rw-r--r--  1 user user  5678 Jan 15 10:00 README.md
# -rw-r--r--  1 user user   890 Jan 15 10:00 Makefile
```

---

## ステップ3: 開発環境の起動

### Docker を使用した開発環境（推奨）

#### 開発コンテナの起動
```bash
# Docker Compose で開発環境を起動
docker-compose -f docker-compose.dev.yml up -d

# 起動確認
docker-compose -f docker-compose.dev.yml ps

# 期待される出力:
# NAME                COMMAND             SERVICE             STATUS              PORTS
# orizon-dev-1        "/bin/bash"         orizon-dev          running             0.0.0.0:8080->8080/tcp
```

#### 開発コンテナへの接続
```bash
# コンテナに接続
docker-compose -f docker-compose.dev.yml exec orizon-dev bash

# または PowerShell の場合
docker-compose -f docker-compose.dev.yml exec orizon-dev pwsh
```

### ローカル環境での直接開発

#### Go モジュールの初期化
```bash
# Go モジュールの依存関係をダウンロード
go mod download

# 依存関係の確認
go mod verify

# Go バージョンの確認
go version
# 期待される出力: go version go1.24.3 linux/amd64
```

---

## ステップ4: 依存関係のインストール

### Go 依存関係の詳細確認

```bash
# 現在の依存関係を表示
go list -m all

# 期待される主要依存関係:
# github.com/orizon-lang/orizon
# github.com/Masterminds/semver/v3 v3.2.1
# github.com/fsnotify/fsnotify v1.9.0
# github.com/quic-go/quic-go v0.54.0
# golang.org/x/sync v0.16.0
# golang.org/x/sys v0.35.0
# golang.org/x/tools v0.35.0
```

### 追加ツールのインストール

#### 開発用ツールの確認
```bash
# Orizon コンパイラのビルド確認
go build -o build/orizon-compiler ./cmd/orizon-compiler

# LSP サーバーのビルド確認
go build -o build/orizon-lsp ./cmd/orizon-lsp

# フォーマッターのビルド確認
go build -o build/orizon-fmt ./cmd/orizon-fmt

# ビルド結果の確認
ls -la build/
```

---

## ステップ5: 環境設定

### 環境変数の設定

#### Windows (PowerShell)
```powershell
# Orizon 開発用環境変数
$env:ORIZON_ROOT = $PWD
$env:ORIZON_BUILD_DIR = "$PWD\build"
$env:PATH += ";$PWD\build"

# 永続化（オプション）
[Environment]::SetEnvironmentVariable("ORIZON_ROOT", $PWD, "User")
[Environment]::SetEnvironmentVariable("PATH", $env:PATH + ";$PWD\build", "User")
```

#### macOS/Linux (Bash)
```bash
# ~/.bashrc または ~/.zshrc に追加
export ORIZON_ROOT="$(pwd)"
export ORIZON_BUILD_DIR="$ORIZON_ROOT/build"
export PATH="$PATH:$ORIZON_BUILD_DIR"

# 即座に適用
source ~/.bashrc  # または source ~/.zshrc
```

### 設定ファイルの作成

#### .orizon-config.toml の作成
```toml
# .orizon-config.toml
[build]
optimization_level = "default"
debug_info = true
target_triple = "x86_64-unknown-linux-gnu"  # 環境に応じて調整

[lsp]
enable_diagnostics = true
enable_completion = true
enable_hover = true
max_completion_items = 50

[fmt]
indent_size = 4
use_tabs = false
max_line_length = 100

[test]
parallel_jobs = 4
verbose = false
coverage = true
```

---

## ステップ6: プロジェクトのビルド

### Makefile を使用したビルド（推奨）

```bash
# 全体をビルド
make build

# 期待される出力:
# Building Orizon compiler...
# go build -o build/orizon-compiler ./cmd/orizon-compiler
# Building Orizon LSP server...
# go build -o build/orizon-lsp ./cmd/orizon-lsp
# Building Orizon formatter...
# go build -o build/orizon-fmt ./cmd/orizon-fmt
# Build completed successfully!
```

### 個別コンポーネントのビルド

```bash
# コンパイラのみビルド
make build-compiler

# LSP サーバーのみビルド
make build-lsp

# 全ツールのビルド
make build-tools

# リリースビルド（最適化付き）
make build-release
```

### Windows PowerShell でのビルド

```powershell
# PowerShell ビルドスクリプトを使用
.\build.ps1 build

# または手動ビルド
go build -o build\orizon-compiler.exe .\cmd\orizon-compiler
go build -o build\orizon-lsp.exe .\cmd\orizon-lsp
go build -o build\orizon-fmt.exe .\cmd\orizon-fmt
```

---

## ステップ7: ビルド結果の検証

### バイナリの動作確認

```bash
# コンパイラのバージョン確認
./build/orizon-compiler --version

# 期待される出力:
# Orizon Compiler 0.1.0-alpha
# Build: dev
# Go version: go1.24.3

# LSP サーバーの動作確認
./build/orizon-lsp --version

# フォーマッターの動作確認
./build/orizon-fmt --version
```

### サンプルコードのコンパイル

```bash
# Hello World サンプルのパース
./build/orizon-compiler --parse examples/01_hello_world.oriz

# 期待される出力:
# 📦 Parsed AST (parser):
# Program {
#   Declarations: [
#     FunctionDeclaration {
#       Name: "main"
#       Parameters: []
#       Body: BlockStatement {
#         Statements: [
#           ExpressionStatement {
#             Expression: CallExpression {
#               Callee: "println"
#               Arguments: ["Hello, Orizon World!"]
#             }
#           }
#         ]
#       }
#     }
#   ]
# }
```

---

## ステップ8: テストの実行

### 単体テストの実行

```bash
# 全単体テストの実行
make test

# 特定パッケージのテスト
go test ./internal/lexer -v
go test ./internal/parser -v
go test ./internal/ast -v

# カバレッジ付きテスト
go test -cover ./...
```

### 統合テストの実行

```bash
# 統合テストの実行
make test-integration

# 全例サンプルファイルのパーステスト
for file in examples/*.oriz; do
    echo "Testing $file..."
    ./build/orizon-compiler --parse "$file"
done
```

### ベンチマークテストの実行

```bash
# パフォーマンステスト
go test -bench=. ./internal/lexer
go test -bench=. ./internal/parser

# メモリプロファイル
go test -benchmem -bench=. ./internal/parser
```

---

## ステップ9: VS Code 開発環境の設定

### VS Code 拡張機能のインストール

#### 必須拡張機能
```json
// .vscode/extensions.json
{
    "recommendations": [
        "golang.go",
        "ms-vscode.vscode-json",
        "redhat.vscode-yaml",
        "ms-vscode.makefile-tools",
        "ms-azuretools.vscode-docker"
    ]
}
```

#### VS Code 設定
```json
// .vscode/settings.json
{
    "go.toolsManagement.checkForUpdates": "local",
    "go.useLanguageServer": true,
    "go.gopath": "${workspaceFolder}",
    "go.goroot": "/usr/local/go",
    "files.associations": {
        "*.oriz": "go"  // 一時的に Orizon ファイルを Go として認識
    },
    "editor.tabSize": 4,
    "editor.insertSpaces": true,
    "editor.formatOnSave": true
}
```

#### デバッグ設定
```json
// .vscode/launch.json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Orizon Compiler",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/orizon-compiler",
            "args": ["--parse", "examples/01_hello_world.oriz"],
            "cwd": "${workspaceFolder}"
        },
        {
            "name": "Debug LSP Server",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/orizon-lsp",
            "args": ["--stdio", "--verbose"]
        }
    ]
}
```

#### タスク設定
```json
// .vscode/tasks.json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "build",
            "type": "shell",
            "command": "make",
            "args": ["build"],
            "group": {
                "kind": "build",
                "isDefault": true
            },
            "presentation": {
                "reveal": "always",
                "panel": "new"
            }
        },
        {
            "label": "test",
            "type": "shell",
            "command": "make",
            "args": ["test"],
            "group": "test",
            "presentation": {
                "reveal": "always",
                "panel": "new"
            }
        },
        {
            "label": "parse-example",
            "type": "shell",
            "command": "./build/orizon-compiler",
            "args": ["--parse", "${file}"],
            "group": "build"
        }
    ]
}
```

---

## ステップ10: 高度な設定とカスタマイズ

### Git フックの設定

#### Pre-commit フックの設定
```bash
# .git/hooks/pre-commit ファイルを作成
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
set -e

echo "Running pre-commit checks..."

# Go コードのフォーマット確認
if ! go fmt ./...; then
    echo "Error: Code formatting issues found. Run 'go fmt ./...' to fix."
    exit 1
fi

# テストの実行
if ! go test ./...; then
    echo "Error: Tests failed."
    exit 1
fi

# リントチェック（golangci-lint がインストールされている場合）
if command -v golangci-lint >/dev/null 2>&1; then
    if ! golangci-lint run; then
        echo "Error: Linting issues found."
        exit 1
    fi
fi

echo "Pre-commit checks passed!"
EOF

# 実行権限を付与
chmod +x .git/hooks/pre-commit
```

### パフォーマンス監視の設定

#### プロファイリング環境の設定
```bash
# pprof ツールのインストール
go install github.com/google/pprof@latest

# プロファイリング用のビルド
go build -o build/orizon-compiler-profile -ldflags="-X main.enableProfiling=true" ./cmd/orizon-compiler
```

### CI/CD パイプライン設定（GitHub Actions）

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: [1.24.3]

    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ matrix.go-version }}

    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-

    - name: Install dependencies
      run: go mod download

    - name: Run tests
      run: make test

    - name: Build
      run: make build

    - name: Run integration tests
      run: make test-integration
```

---

## トラブルシューティング

### 一般的な問題と解決策

#### 1. Go バージョンの不一致

**問題**: `go version` が 1.24.3 より古い

**解決策**:
```bash
# 最新版 Go をインストール
# Linux/macOS
sudo rm -rf /usr/local/go
curl -fsSL https://golang.org/dl/go1.24.3.linux-amd64.tar.gz | sudo tar -C /usr/local -xzf -

# Windows
# https://golang.org/dl/ から最新版をダウンロード・インストール
```

#### 2. Docker 権限エラー

**問題**: `permission denied while trying to connect to the Docker daemon socket`

**解決策**:
```bash
# Linux の場合
sudo usermod -aG docker $USER
newgrp docker

# または sudo で実行
sudo docker-compose -f docker-compose.dev.yml up -d
```

#### 3. ポート競合エラー

**問題**: `port 8080 is already in use`

**解決策**:
```bash
# 使用中のプロセスを確認
sudo lsof -i :8080

# プロセスを停止するか、別のポートを使用
# docker-compose.dev.yml でポート番号を変更
```

#### 4. メモリ不足エラー

**問題**: ビルド中に `fatal error: out of memory`

**解決策**:
```bash
# 並列ビルド数を制限
export GOMAXPROCS=2
make build

# または Docker の場合、メモリ制限を増加
# Docker Desktop の設定で Memory を 8GB 以上に設定
```

#### 5. 依存関係ダウンロードエラー

**問題**: `go mod download` でエラー

**解決策**:
```bash
# プロキシキャッシュをクリア
go clean -modcache

# 依存関係を再ダウンロード
go mod download

# ネットワーク問題の場合、プロキシを設定
go env -w GOPROXY=https://proxy.golang.org,direct
```

### デバッグ支援

#### ログレベルの設定
```bash
# 詳細ログでコンパイラを実行
ORIZON_LOG_LEVEL=debug ./build/orizon-compiler --parse examples/01_hello_world.oriz

# LSP サーバーのデバッグ
./build/orizon-lsp --stdio --verbose --log-file=lsp.log
```

#### プロファイリング実行
```bash
# CPU プロファイル
go test -cpuprofile=cpu.prof -bench=. ./internal/parser

# メモリプロファイル
go test -memprofile=mem.prof -bench=. ./internal/parser

# プロファイル解析
go tool pprof cpu.prof
```

---

## 次のステップ

### 開発の開始

1. **サンプルコードの理解**: `examples/` ディレクトリのファイルを順番に確認
2. **コントリビューション**: [CONTRIBUTING.md](../CONTRIBUTING.md) を確認
3. **コミュニティ参加**: GitHub Discussions や Discord に参加

### 学習リソース

1. **言語仕様**: `spec/` ディレクトリの設計文書
2. **API ドキュメント**: `docs/` ディレクトリの詳細資料
3. **実装理解**: `internal/` ディレクトリのソースコード

### 高度な開発

1. **新機能の実装**: 言語機能の追加・拡張
2. **パフォーマンス最適化**: プロファイリングとボトルネック解析
3. **ツール開発**: IDE拡張やデバッグツールの開発

このセットアップガイドにより、Orizonプログラミング言語の開発環境を完全に構築し、効率的な開発を開始することができます。質問や問題が発生した場合は、GitHub Issues やコミュニティフォーラムでサポートを受けることができます。
