# Orizon 開発環境セットアップガイド

## システム要件

### 最小要件
- **OS**: Windows 10/11 (x64), Ubuntu 20.04+, macOS 10.15+
- **CPU**: x86_64 アーキテクチャ、SSE4.1サポート
- **メモリ**: 8GB RAM（開発用、16GB推奨）
- **ストレージ**: 10GB 空き容量（SSD推奨）
- **Go**: 1.24.3 以上

### 推奨要件
- **CPU**: AVX2/AVX512サポート、8コア以上
- **メモリ**: 32GB RAM（大規模プロジェクト用）
- **ストレージ**: NVMe SSD、50GB以上
- **NUMA**: マルチソケット環境（パフォーマンス最適化）

### OS開発要件
- **仮想化**: VMware Workstation/Fusion、VirtualBox、QEMU
- **デバッガー**: GDB 9.0+、またはLLDB
- **エミュレーター**: QEMU 6.0+（ARM64開発時）

---

## インストール手順

### 1. Goツールチェーンのセットアップ

**Windows (PowerShell)**:
```powershell
# Go 1.24.3 インストール確認
go version

# 環境変数設定
$env:GO111MODULE = "on"
$env:GOPROXY = "https://proxy.golang.org,direct"

# ワークスペース準備
mkdir C:\dev\orizon
cd C:\dev\orizon
```

**Linux/macOS**:
```bash
# Go バージョン確認
go version

# 環境変数設定
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org,direct

# ワークスペース準備
mkdir -p ~/dev/orizon
cd ~/dev/orizon
```

### 2. Orizonソースコード取得

```bash
# リポジトリクローン
git clone https://github.com/orizon-lang/orizon.git
cd orizon

# 依存関係の確認
go mod download
go mod verify
```

### 3. 開発ツールのビルド

**完全ビルド（推奨）**:
```bash
# 全コンポーネントのビルド
make all

# またはクイックビルド
make quick-build
```

**個別コンポーネントビルド**:
```bash
# コンパイラのみ
make compiler

# LSPサーバーのみ
make lsp

# フォーマッターのみ
make fmt

# テストランナーのみ
make test-runner
```

### 4. インストール確認

```bash
# ビルド結果確認
ls -la build/

# コンパイラテスト
./build/orizon-compiler --version

# LSPサーバーテスト
./build/orizon-lsp --help

# サンプルコンパイル
./build/orizon-compiler examples/01_hello_world.oriz
```

---

## 統合開発環境 (IDE) セットアップ

### Visual Studio Code

**1. 拡張機能インストール**:
```json
{
    "recommendations": [
        "orizon-lang.orizon-language-support",
        "ms-vscode.cpptools",
        "vadimcn.vscode-lldb",
        "ms-vscode.hexeditor"
    ]
}
```

**2. 設定ファイル** (`.vscode/settings.json`):
```json
{
    "orizon.compiler.path": "./build/orizon-compiler",
    "orizon.lsp.path": "./build/orizon-lsp",
    "orizon.formatter.path": "./build/orizon-fmt",
    "orizon.enableInlayHints": true,
    "orizon.enableSemanticHighlighting": true,
    "orizon.diagnostics.enableExperimental": true,
    
    "editor.semanticHighlighting.enabled": true,
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
        "source.fixAll.orizon": true,
        "source.organizeImports.orizon": true
    },
    
    "files.associations": {
        "*.oriz": "orizon"
    }
}
```

**3. タスク設定** (`.vscode/tasks.json`):
```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Build Orizon Project",
            "type": "shell",
            "command": "./build/orizon-compiler",
            "args": ["${workspaceFolder}/src/main.oriz"],
            "group": {
                "kind": "build",
                "isDefault": true
            },
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            },
            "problemMatcher": "$gcc"
        },
        {
            "label": "Run Orizon Tests",
            "type": "shell",
            "command": "./build/orizon-test",
            "args": ["${workspaceFolder}/tests/"],
            "group": "test",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            }
        },
        {
            "label": "Format Orizon Code",
            "type": "shell",
            "command": "./build/orizon-fmt",
            "args": ["${workspaceFolder}/src/"],
            "group": "build"
        }
    ]
}
```

**4. デバッグ設定** (`.vscode/launch.json`):
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Orizon Program",
            "type": "cppdbg",
            "request": "launch",
            "program": "${workspaceFolder}/build/output",
            "args": [],
            "stopAtEntry": false,
            "cwd": "${workspaceFolder}",
            "environment": [],
            "externalConsole": false,
            "MIMode": "gdb",
            "setupCommands": [
                {
                    "description": "Enable pretty-printing for gdb",
                    "text": "-enable-pretty-printing",
                    "ignoreFailures": true
                }
            ]
        },
        {
            "name": "Debug Orizon OS (QEMU)",
            "type": "cppdbg",
            "request": "launch",
            "program": "${workspaceFolder}/build/orizon_kernel.exe",
            "miDebuggerServerAddress": "localhost:1234",
            "MIMode": "gdb",
            "setupCommands": [
                {
                    "text": "target remote localhost:1234"
                },
                {
                    "text": "symbol-file build/orizon_kernel.exe"
                }
            ]
        }
    ]
}
```

### NeoVim/Vim セットアップ

**1. プラグイン設定** (`init.lua`):
```lua
-- LSP設定
local lspconfig = require('lspconfig')

-- Orizon LSP設定
lspconfig.orizon_lsp.setup{
    cmd = {"./build/orizon-lsp", "--stdio"},
    filetypes = {"orizon"},
    root_dir = lspconfig.util.root_pattern("go.mod", ".git"),
    settings = {
        orizon = {
            compiler = {
                path = "./build/orizon-compiler"
            },
            diagnostics = {
                enable = true,
                experimental = true
            }
        }
    }
}

-- ファイルタイプ設定
vim.cmd([[
    augroup OrizonFileType
        autocmd!
        autocmd BufRead,BufNewFile *.oriz set filetype=orizon
    augroup END
]])

-- キーバインド
vim.keymap.set('n', '<leader>cc', ':!./build/orizon-compiler %<CR>')
vim.keymap.set('n', '<leader>cf', ':!./build/orizon-fmt %<CR>')
vim.keymap.set('n', '<leader>ct', ':!./build/orizon-test<CR>')
```

---

## 開発ワークフロー

### 1. 新規プロジェクト作成

```bash
# プロジェクトディレクトリ作成
mkdir my-orizon-project
cd my-orizon-project

# プロジェクト初期化
../orizon/build/orizon-pkg init .

# 基本ファイル構造作成
mkdir -p {src,tests,docs,build}

# メインファイル作成
cat > src/main.oriz << 'EOF'
import std::io;

fn main() -> i32 {
    io::println("Hello, Orizon!");
    return 0;
}
EOF
```

### 2. コンパイルとテスト

```bash
# デバッグビルド
../orizon/build/orizon-compiler -o build/main src/main.oriz

# リリースビルド
../orizon/build/orizon-compiler -O3 -o build/main_optimized src/main.oriz

# テスト実行
../orizon/build/orizon-test tests/

# ベンチマーク実行
../orizon/build/orizon-test --bench tests/
```

### 3. コード品質チェック

```bash
# フォーマット
../orizon/build/orizon-fmt src/

# リント
../orizon/build/orizon-compiler --lint src/

# 依存関係チェック
../orizon/build/orizon-pkg check

# セキュリティ監査
../orizon/build/orizon-compiler --audit src/
```

### 4. OS開発ワークフロー

**QEMUでのテスト**:
```bash
# OSイメージ作成
make os-image

# QEMU起動（デバッグ用）
qemu-system-x86_64 \
    -m 512M \
    -smp 4 \
    -drive file=build/orizon_os.img,format=raw \
    -serial stdio \
    -monitor telnet:localhost:55555,server,nowait \
    -gdb tcp::1234 \
    -S

# 別ターミナルでGDB接続
gdb build/orizon_kernel.exe
(gdb) target remote localhost:1234
(gdb) continue
```

**VMwareでのテスト**:
```bash
# VMware用イメージ変換
qemu-img convert -f raw -O vmdk build/orizon_os.img build/orizon_os.vmdk

# VMware設定ファイル生成
../scripts/generate_vmx.sh build/orizon_os.vmdk
```

---

## パフォーマンス最適化

### 1. プロファイリング環境

**CPU プロファイリング**:
```bash
# プロファイル情報付きビルド
../orizon/build/orizon-compiler -profile -o build/main_profile src/main.oriz

# 実行時プロファイリング
../orizon/build/orizon-profile build/main_profile

# 結果分析
../orizon/build/orizon-profile --analyze profile.data
```

**メモリプロファイリング**:
```bash
# メモリリーク検出
valgrind --tool=memcheck --leak-check=full ./build/main

# NUMA 最適化確認
numactl --hardware
numactl --show
```

### 2. ベンチマーク環境

**マイクロベンチマーク**:
```orizon
import std::benchmark;
import std::time;

#[bench]
fn bench_hash_function(b: &mut benchmark::Bencher) {
    let data = generate_test_data(1024);
    
    b.iter(|| {
        hash_function(&data)
    });
}

#[bench]
fn bench_network_throughput(b: &mut benchmark::Bencher) {
    let server = start_test_server();
    let client = create_test_client();
    
    b.bytes = 1024 * 1024; // 1MB
    b.iter(|| {
        client.send_data(&generate_data(1024 * 1024));
    });
}
```

**システムベンチマーク**:
```bash
# Rustとの性能比較
../scripts/benchmark_vs_rust.sh

# C/C++との性能比較
../scripts/benchmark_vs_cpp.sh

# レポート生成
../scripts/generate_performance_report.sh
```

---

## トラブルシューティング

### よくある問題と解決方法

**1. コンパイルエラー**:
```bash
# 詳細エラー情報の取得
../orizon/build/orizon-compiler --verbose --explain-errors src/main.oriz

# キャッシュクリア
rm -rf ~/.cache/orizon/
../orizon/build/orizon-compiler --clear-cache
```

**2. LSP接続問題**:
```bash
# LSPサーバーのログ確認
../orizon/build/orizon-lsp --log-level=debug --log-file=lsp.log

# プロセス確認
ps aux | grep orizon-lsp
```

**3. 性能問題**:
```bash
# システム情報確認
../orizon/build/orizon-compiler --system-info

# 最適化レベル確認
../orizon/build/orizon-compiler --print-optimization-info
```

### デバッグテクニック

**1. アサーション活用**:
```orizon
fn critical_function(data: &[u8]) {
    debug_assert!(data.len() > 0, "Data cannot be empty");
    debug_assert!(data.len() <= MAX_SIZE, "Data too large: {}", data.len());
    
    // 処理...
}
```

**2. ログ出力**:
```orizon
import std::log;

fn network_handler() {
    log::trace!("Entering network handler");
    log::debug!("Processing {} connections", connection_count);
    log::info!("Server started on port {}", port);
    log::warn!("High memory usage: {}MB", memory_usage);
    log::error!("Connection failed: {}", error);
}
```

### パフォーマンスチューニング

**1. コンパイル最適化**:
```bash
# 最大最適化
../orizon/build/orizon-compiler -O3 -march=native -mtune=native

# リンク時最適化
../orizon/build/orizon-compiler -O3 -flto

# プロファイル誘導最適化
../orizon/build/orizon-compiler -fprofile-generate -o build/main_pgo src/main.oriz
./build/main_pgo # プロファイル収集
../orizon/build/orizon-compiler -fprofile-use -O3 -o build/main_optimized src/main.oriz
```

**2. 実行時最適化**:
```bash
# NUMA最適化
numactl --cpunodebind=0 --membind=0 ./build/main

# CPU親和性設定
taskset -c 0-7 ./build/main

# 優先度設定
nice -n -20 ./build/main
```

---

このセットアップガイドに従うことで、Orizonの高性能な開発環境を構築し、Rustを超える性能のシステムソフトウェアを効率的に開発できます。
