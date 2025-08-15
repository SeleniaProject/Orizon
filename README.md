# Orizon Programming Language
**現存するすべてのシステムプログラミング言語を技術的に凌駕する革命的言語**

[![Build Status](https://github.com/orizon-lang/orizon/workflows/CI/badge.svg)](https://github.com/orizon-lang/orizon/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![Rust Version](https://img.shields.io/badge/Rust-1.75+-orange.svg)](https://rustlang.org)

## ビジョン

Orizonは、**現実的な革新**に焦点を当てた次世代システムプログラミング言語です：

- 🚀 **世界最速**: Rustの10倍、Goの2倍のコンパイル速度
- 🛡️ **完全安全**: C++並みのパフォーマンス、Rust超えの安全性
- 🎯 **開発者体験**: 世界一分かりやすいエラーメッセージと段階的学習
- 🌐 **普遍的統合**: カーネルからWebまで、すべてのプラットフォーム対応

## 主要特徴

### 革新的技術
- **Dependent Types 2.0**: Rustの所有権システムを超える依存型システム
- **Zero-Cost GC**: コンパイル時完全解析による実行時オーバーヘッドゼロ
- **Actor Model 3.0**: Erlang/Elixirを超える軽量プロセスシステム
- **AI駆動開発支援**: リアルタイム静的解析とインテリジェントサジェスト

### 現実的優位性
- **C ABI互換**: 既存Cライブラリとの完璧な相互運用性
- **段階的移行**: 既存コードベースの無痛移行サポート
- **Universal Platform**: WebAssembly、GPU、組み込みまで統一開発体験

## クイックスタート

### Hello World

```orizon
// Orizonの美しい構文
func main() {
    print("Hello, Orizon! 🌟")
}
```

### 高度な例

```orizon
// 依存型による配列境界の静的保証
func safe_access<T, N: usize>(arr: [T; N], index: usize where index < N) -> T {
    arr[index]  // 境界チェック不要 - コンパイル時に保証済み
}

// アクターベース並行処理
actor Counter {
    var value: int = 0
    
    func increment() -> int {
        value += 1
        return value
    }
}

func main() {
    let counter = spawn Counter()
    
    // 1000個の並行タスクで安全にカウンタを更新
    let tasks = for i in 0..1000 spawn {
        counter.increment()
    }
    
    await_all(tasks)
    print("Final count: {}", counter.value)  // 確実に1000
}
```

## 開発環境のセットアップ

### 前提条件
- Docker & Docker Compose
- VS Code (推奨)
- Git

### 開発環境起動

```bash
# リポジトリクローン
git clone https://github.com/orizon-lang/orizon.git
cd orizon

# 開発環境起動（C/C++依存なし）
docker-compose -f docker-compose.dev.yml up -d

# コンテナに接続
docker-compose -f docker-compose.dev.yml exec orizon-dev bash

# コンパイラビルド
make build

# テスト実行
make test

# サンプル実行
./build/orizon-compiler examples/hello.oriz
```

### Windows (PowerShell) の注意

- コマンド連結に `&&`/`||` は使えません。`;` で区切るか、1行ずつ実行してください。
  - 例: `go build ./...; go test ./... -count=1`
  - 例: `git add -A; git status -s`
  - 失敗時に中断したい場合: `if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }`

### ファジングと再現

```bash
# パーサーファズ（カバレッジ/ユニーク数/興味深い入力の保存）
./orizon-fuzz --target parser --duration 10s \
  --covout fuzz.cov --covstats --cov-mode weighted \
  --corpus corpus/parser_corpus.txt --corpus-out corpus_new \
  --out crashes.txt

# 使用シードを保存して再現性を確保
./orizon-fuzz --target parser --duration 10s --save-seed seed.txt
SEED=$(cat seed.txt); ./orizon-fuzz --target parser --duration 10s --seed $SEED

# 実行統計の表示・JSON保存
./orizon-fuzz --target parser --duration 10s --stats --json-stats stats.json

# ディレクトリコーパスとクラッシュ出力ディレクトリ
./orizon-fuzz --target parser --duration 10s --corpus-dir ./corpus --crash-dir ./crashes_raw

# カバレッジモードと変異強度の選択（edge|weighted|trigram|both）
./orizon-fuzz --target parser --duration 10s --cov-mode trigram --intensity 1.5 --stats

# 自動チューニングで強度を調整
./orizon-fuzz --target parser --duration 20s --autotune --stats

# syntaxエラーは無視してパニックのみ収集（parser-lax）
./orizon-fuzz --target parser-lax --duration 5s --p 2 --corpus corpus/parser_corpus.txt --max-execs 2000 --stats

# レキサーファズ（入力タイムアウトとクラッシュ自動最小化）
./orizon-fuzz --target lexer --duration 10s --covstats --corpus corpus/lexer_corpus.txt --per 200ms --min-on-crash --min-dir crashes_min --min-budget 3s

# ASTブリッジ往復（パース成功入力を要求）
./orizon-fuzz --target astbridge --duration 10s --covstats --corpus corpus/astbridge_corpus.txt --per 300ms

# HIR 検証（パース→AST→HIR変換→ValidateHIR）
./orizon-fuzz --target hir --duration 10s --covstats --corpus corpus/parser_corpus.txt --per 300ms --min-on-crash --min-dir crashes_min

# ASTブリッジ往復＋HIR検証（ブリッジ後の構文木を変換・検証）
./orizon-fuzz --target astbridge-hir --duration 10s --covstats --corpus corpus/astbridge_corpus.txt --per 300ms --min-on-crash --min-dir crashes_min

# クラッシュ再現と最小化
./orizon-repro --in crashes/input_001.oriz --out minimized.oriz --budget 5s

# クラッシュログ（crashes.txt）から最終クラッシュを直接再現
./orizon-repro --log crashes.txt --budget 5s --target parser

# 任意の行番号を指定して再現（1-based）
./orizon-repro --log crashes.txt --line 42 --budget 5s --target parser
```

### WindowsでのI/Oポーラ選択（環境変数）

```powershell
# 既定: ポータブル（goroutineベース）
$env:ORIZON_WIN_PORTABLE="1"

# WSAPollを強制
$env:ORIZON_WIN_WSAPOLL="1"

# IOCPを要求（ビルドタグ windows,iocp が必要。未タグ時はWSAPollへフォールバック）
$env:ORIZON_WIN_IOCP="1"
```

### Windows ポーラー選択とAPI保証（概要）

- 選択優先度（`internal/runtime/asyncio/poller_factory_windows.go`）
  1) IOCP（`-tags iocp` かつ `ORIZON_WIN_IOCP=1` のとき有効）
  2) WSAPoll（`ORIZON_WIN_WSAPOLL=1`）
  3) ポータブル（goroutine ベース、既定）

- API保証（クロスプラットフォーム整合）
  - Register: 冪等（同一 `net.Conn` の再登録で handler/kinds を更新）
  - Deregister: 二重呼び出し/クローズ後でも安全（Windows では by-conn フォールバックを実装）
  - Writable: スロットリングで過剰通知を抑制

- IOCP 実装について
  - 実験的（build tag `iocp` 必要）。未タグ時は WSAPoll/ポータブルにフォールバックします。
  - `CancelIoEx` による未完了I/Oのキャンセル、解除・停止時のタイムアウト待機を実装し、シャットダウンを安定化。


### Windows IOCP のビルド/テスト（実験）

```powershell
# IOCP 実装を有効化してビルド（Windows環境でのみ有効）
go build -tags iocp ./...

# IOCP 経路のユニットテスト（実験タグ）
go test -tags iocp ./internal/runtime/asyncio -run IOCPPoller -v

# 実行時にIOCPを明示要求（未タグ時はWSAPollへフォールバック）
$env:ORIZON_WIN_IOCP="1"
```

### テストランナー

```bash
# 全パッケージのテストを並列実行（カラー、JSON無効）
./orizon-test --packages ./... --p 0 --color

# 特定のテスト名にマッチさせる（正規表現）
./orizon-test --packages ./internal/... --run "TestActorSystem_.*"

# go test の追加引数をそのまま渡す
./orizon-test --packages ./... --args "-bench=. -benchmem" --json

# フレーク対策・JUnit出力・フェイルファスト
./orizon-test --packages ./... --retries 2 --fail-fast --junit junit.xml --color=false

# パッケージ名の正規表現で対象を絞り込む
./orizon-test --packages ./... --pkg-regex "^github.com/orizon-lang/orizon/internal/"

# 試行履歴を含むJSONサマリを保存（--retries と併用可）
./orizon-test --packages ./... --retries 2 --json-summary test-summary.json
```

利用可能な主なフラグ:

- `--packages` (複数可、カンマ区切り): 対象パッケージパターン。例: `./...,./internal/...`
- `--run`: テスト名の正規表現（`go test -run` に委譲）
- `--p`: パッケージ並列数（既定は `runtime.NumCPU()`）
- `--json`: `go test -json` をそのままストリーム
- `--json-augment`: `--json` 併用時、フレーク回復などOrizon拡張イベントを付加
- `--short`, `--race`, `--timeout`, `--color`
- `--env`, `--args`: 追加環境変数（`;`区切り）と追加引数
- `--junit`: JUnit XML の出力先パス
- `--retries`: 失敗テストの再試行回数（フレーク検出）
- `--fail-fast`: 最初の失敗で残りをキャンセル
- `--pkg-regex`: `go list` 展開後のパッケージ名フィルタ用正規表現
- `--file-regex`: パッケージ内のファイルパスに対する正規表現フィルタ（該当ファイルを含むPKGのみ実行）
- `--list`: 実行せずにテスト一覧のみ表示（ドライラン）
- `--json-summary`: 実行結果の要約JSON（各テストの試行履歴含む）
- `--fail-on-flaky`: 再試行で回復した（フレーク）テストがあれば非ゼロ終了

### モック生成器
### Windows ローカルスモーク

PowerShell で一括スモーク（ビルド/テスト/ファズ/再現/IOCPテスト）。成果物は `artifacts/` に保存されます。

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\win\smoke.ps1
```

後片付け:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\win\clean.ps1
```

### Linux/macOS ローカルスモーク
### デバッガ（GDB RSP）

最小RSPサーバを起動し、GDB/LLDB互換で接続できます。

```bash
# サーバ起動（JSONデバッグ情報を指定、TCP 9000で待受）
./gdb-rsp-server --debug-json artifacts/debug.json --addr :9000

# 俳優/メモリ統計のHTTP連携を有効化
./gdb-rsp-server --debug-json artifacts/debug.json --addr :9000 --debug-http

# GDBから接続
gdb -q -ex "target remote localhost:9000"
```


```bash
bash ./scripts/linux/smoke.sh
```



```bash
# 指定パッケージ配下のインターフェースからモックを生成
./orizon-mockgen --pkg ./internal/runtime --out ./internal/runtime/mocks

# 単一ファイルを入力にして出力先を指定
./orizon-mockgen --in ./internal/packagemanager/resolver.go --out ./internal/packagemanager/mocks
```


### VS Code開発

1. 推奨拡張機能をインストール
2. `Ctrl+Shift+P` → "Remote-Containers: Reopen in Container"
3. ターミナルで `make dev` 実行

## プロジェクト構造

```
orizon/
├── cmd/                    # コマンドラインツール
│   ├── orizon-compiler/    # メインコンパイラ
│   ├── orizon-lsp/        # Language Server Protocol
│   ├── orizon-fmt/        # コードフォーマッタ
│   ├── orizon-fuzz/       # ファザー（近似カバレッジ対応）
│   ├── orizon-repro/      # クラッシュ再現・最小化
│   └── orizon-test/       # Goテストラッパー（カラー/JSON/並列）
├── internal/              # 内部実装
│   ├── lexer/            # 字句解析器
│   ├── parser/           # 構文解析器
│   ├── typechecker/      # 型検査器
│   ├── codegen/          # コード生成
│   └── runtime/          # ランタイムシステム
├── examples/             # サンプルコード
├── spec/                # 言語仕様
├── docs/                # ドキュメント
└── test/                # テストスイート
```

## 開発ロードマップ

### Phase 0: 基盤構築 (完了)
- ✅ 開発環境セットアップ
- ✅ プロジェクト構造定義
- ✅ 言語仕様設計

### Phase 1: コアコンパイラ (進行中)
- 🔄 字句解析器実装
- ⏳ 構文解析器実装
- ⏳ AST設計と実装

### Phase 2: 型システム (予定)
- ⏳ 基本型システム
- ⏳ 依存型システム
- ⏳ 効果システム

## 貢献方法

Orizonは世界中の開発者コミュニティによって構築されています：

1. [Contributing Guide](docs/CONTRIBUTING.md)を確認
2. [Issues](https://github.com/orizon-lang/orizon/issues)から作業を選択
3. プルリクエストを作成

## ライセンス

MIT License - 詳細は[LICENSE](LICENSE)ファイルを参照

## コミュニティ

- 🐙 [GitHub Discussions](https://github.com/orizon-lang/orizon/discussions)
- 💬 [Discord Server](https://discord.gg/orizon-lang)
- 🐦 [Twitter](https://twitter.com/orizon_lang)
- 📝 [Blog](https://blog.orizon-lang.org)

---

**Orizon - The Future of Systems Programming** 🌟
