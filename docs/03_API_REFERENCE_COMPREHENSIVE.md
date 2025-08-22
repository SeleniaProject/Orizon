# Orizon Programming Language - API・外部インターフェースリファレンス

## 概要

本ドキュメントでは、Orizonプログラミング言語が提供する主要なAPIエンドポイント、内部インターフェース、および外部システムとの統合ポイントについて詳細に解説します。開発者がOrizonシステムを効果的に活用するための完全なリファレンスです。

## コンパイラAPI

### orizon-compiler CLI インターフェース

Orizonコンパイラのメインインターフェースです。

#### 基本コンパイルコマンド

```bash
# 基本構文
orizon-compiler [OPTIONS] <INPUT_FILE>

# 使用例
orizon-compiler hello.oriz
orizon-compiler --optimize-level=aggressive main.oriz
orizon-compiler --emit-debug --dwarf-out-dir=./debug program.oriz
```

#### コマンドラインオプション

| オプション         | 型     | デフォルト | 説明                                           |
| ------------------ | ------ | ---------- | ---------------------------------------------- |
| `--version`        | flag   | false      | バージョン情報を表示                           |
| `--help`           | flag   | false      | ヘルプメッセージを表示                         |
| `--json`           | flag   | false      | バージョン情報をJSON形式で出力                 |
| `--debug-lexer`    | flag   | false      | 字句解析デバッグ出力を有効化                   |
| `--parse`          | flag   | false      | パース結果のAST表示                            |
| `--optimize-level` | string | ""         | 最適化レベル: none\|basic\|default\|aggressive |
| `--emit-debug`     | flag   | false      | デバッグ情報JSON + DWARFセクション出力         |
| `--emit-sourcemap` | flag   | false      | ソースマップJSON出力                           |
| `--debug-out`      | string | ""         | デバッグJSONのファイル出力先                   |
| `--sourcemap-out`  | string | ""         | ソースマップJSONのファイル出力先               |
| `--dwarf-out-dir`  | string | ""         | DWARFセクションのディレクトリ出力先            |
| `--emit-elf`       | string | ""         | 最小ELF64オブジェクト出力                      |
| `--emit-coff`      | string | ""         | 最小COFF(AMD64)オブジェクト出力                |
| `--emit-macho`     | string | ""         | 最小Mach-O(x86_64)オブジェクト出力             |
| `--emit-mir`       | flag   | false      | MIRテキストダンプ出力                          |
| `--emit-lir`       | flag   | false      | LIRテキストダンプ出力                          |
| `--emit-x64`       | flag   | false      | 診断用x64アセンブリテキスト出力                |
| `--x64-out`        | string | ""         | 診断用x64アセンブリのファイル出力先            |

#### 環境変数設定

| 環境変数                  | 説明                             | 有効値                 |
| ------------------------- | -------------------------------- | ---------------------- |
| `ORIZON_DEBUG_OBJ_OUT`    | オブジェクトファイルの自動出力先 | ファイルパス           |
| `ORIZON_DEBUG_OBJ_FORMAT` | オブジェクトファイル形式の指定   | auto\|elf\|coff\|macho |

#### 戻り値・ステータスコード

| ステータス | 説明                       |
| ---------- | -------------------------- |
| 0          | 正常終了                   |
| 1          | 引数エラー、ファイル不存在 |
| 2          | 字句解析エラー             |
| 3          | 構文解析エラー             |
| 4          | 型検査エラー               |
| 5          | コード生成エラー           |

#### 使用例

```bash
# 1. 基本的なコンパイル
orizon-compiler hello.oriz

# 2. 字句解析のデバッグ
orizon-compiler --debug-lexer test.oriz

# 3. パース結果の確認
orizon-compiler --parse example.oriz

# 4. 最適化付きコンパイル
orizon-compiler --optimize-level=aggressive performance_critical.oriz

# 5. デバッグ情報付きコンパイル
orizon-compiler --emit-debug --debug-out=debug.json program.oriz

# 6. 全ての中間表現を出力
orizon-compiler --emit-mir --emit-lir --emit-x64 --x64-out=output.s complex.oriz

# 7. クロスプラットフォームオブジェクト生成
orizon-compiler --emit-elf=output.o --emit-coff=output.obj --emit-macho=output.dylib multi_target.oriz
```

---

## Language Server Protocol (LSP) API

### orizon-lsp サーバーインターフェース

VS Code、Vim、Emacsなどのエディタとの統合のためのLSPサーバーです。

#### 起動・設定

```bash
# LSPサーバー起動
orizon-lsp --stdio

# TCP接続での起動
orizon-lsp --tcp --port=7777

# デバッグモード
orizon-lsp --stdio --verbose --log-file=lsp.log
```

#### サポートするLSP機能

##### テキスト同期機能
```json
{
  "textDocumentSync": {
    "openClose": true,
    "change": 2,  // incremental
    "save": {
      "includeText": true
    }
  }
}
```

##### 補完機能
```json
{
  "completionProvider": {
    "triggerCharacters": [".", ":", "(", "<"],
    "resolveProvider": true
  }
}
```

**リクエスト例**:
```json
{
  "method": "textDocument/completion",
  "params": {
    "textDocument": {
      "uri": "file:///path/to/file.oriz"
    },
    "position": {
      "line": 10,
      "character": 15
    }
  }
}
```

**レスポンス例**:
```json
{
  "result": {
    "items": [
      {
        "label": "println",
        "kind": 3,  // Function
        "detail": "func println(format: String, ...args)",
        "documentation": "Print formatted string to stdout",
        "insertText": "println(\"$1\")",
        "insertTextFormat": 2  // Snippet
      }
    ]
  }
}
```

##### 診断機能（エラー・警告）
```json
{
  "method": "textDocument/publishDiagnostics",
  "params": {
    "uri": "file:///path/to/file.oriz",
    "diagnostics": [
      {
        "range": {
          "start": {"line": 5, "character": 10},
          "end": {"line": 5, "character": 20}
        },
        "severity": 1,  // Error
        "code": "E0001",
        "source": "orizon",
        "message": "Type mismatch: expected `i32`, found `String`",
        "relatedInformation": [
          {
            "location": {
              "uri": "file:///path/to/file.oriz",
              "range": {
                "start": {"line": 2, "character": 5},
                "end": {"line": 2, "character": 10}
              }
            },
            "message": "Variable declared here"
          }
        ]
      }
    ]
  }
}
```

##### ホバー情報
```json
{
  "method": "textDocument/hover",
  "params": {
    "textDocument": {"uri": "file:///path/to/file.oriz"},
    "position": {"line": 8, "character": 12}
  }
}
```

**レスポンス**:
```json
{
  "result": {
    "contents": {
      "kind": "markdown",
      "value": "```orizon\nfunc add(a: i32, b: i32) -> i32\n```\n\nAdds two integers and returns the result.\n\n**Parameters:**\n- `a`: First integer\n- `b`: Second integer\n\n**Returns:** Sum of a and b"
    },
    "range": {
      "start": {"line": 8, "character": 10},
      "end": {"line": 8, "character": 15}
    }
  }
}
```

##### 定義ジャンプ
```json
{
  "method": "textDocument/definition",
  "params": {
    "textDocument": {"uri": "file:///path/to/file.oriz"},
    "position": {"line": 10, "character": 5}
  }
}
```

##### 参照検索
```json
{
  "method": "textDocument/references",
  "params": {
    "textDocument": {"uri": "file:///path/to/file.oriz"},
    "position": {"line": 5, "character": 10},
    "context": {"includeDeclaration": true}
  }
}
```

---

## フォーマッターAPI

### orizon-fmt インターフェース

コードフォーマッティングツールのAPIです。

#### コマンドライン使用

```bash
# ファイルをフォーマット（上書き）
orizon-fmt --write file.oriz

# フォーマット結果を標準出力
orizon-fmt file.oriz

# 複数ファイルの一括フォーマット
orizon-fmt --write src/**/*.oriz

# 設定ファイル指定
orizon-fmt --config=.orizon-fmt.toml --write project/
```

#### プログラマティック API

```go
package main

import (
    "fmt"
    "github.com/orizon-lang/orizon/internal/format"
)

func main() {
    source := `func main(){println("Hello, World!");}`
    
    formatter := format.NewFormatter()
    formatted, err := formatter.Format(source)
    if err != nil {
        panic(err)
    }
    
    fmt.Println(formatted)
    // 出力:
    // func main() {
    //     println("Hello, World!");
    // }
}
```

#### フォーマット設定オプション

```toml
# .orizon-fmt.toml
[format]
indent_size = 4
use_tabs = false
max_line_length = 100
trailing_comma = true
bracket_spacing = true

[imports]
group_imports = true
sort_imports = true

[functions]
space_before_paren = false
space_inside_paren = false
```

---

## テストフレームワークAPI

### orizon-test インターフェース

Orizon用テストランナーのAPIです。

#### コマンドライン使用

```bash
# 全テスト実行
orizon-test

# 特定のテストファイル実行
orizon-test tests/unit_test.oriz

# パターンマッチでテスト選択
orizon-test --pattern="test_calc_*"

# 並列実行
orizon-test --parallel=4

# カバレッジ計測
orizon-test --coverage --coverage-out=coverage.html

# JSON出力
orizon-test --json --output=test_results.json
```

#### テストファイル記述形式

```orizon
// test/example_test.oriz

#[test]
func test_addition() {
    let result = add(2, 3);
    assert_eq!(result, 5);
}

#[test]
func test_subtraction() {
    let result = subtract(5, 3);
    assert_eq!(result, 2);
}

#[test]
#[should_panic]
func test_division_by_zero() {
    let result = divide(10, 0);  // パニックするはず
}

#[bench]
func bench_fibonacci() {
    for i in 0..100 {
        fibonacci(20);
    }
}
```

#### アサーション関数

| 関数                                      | 説明                       | 使用例                                  |
| ----------------------------------------- | -------------------------- | --------------------------------------- |
| `assert!(condition)`                      | 条件がtrueであることを確認 | `assert!(x > 0)`                        |
| `assert_eq!(left, right)`                 | 値が等しいことを確認       | `assert_eq!(result, 42)`                |
| `assert_ne!(left, right)`                 | 値が異なることを確認       | `assert_ne!(x, y)`                      |
| `assert_approx_eq!(left, right, epsilon)` | 浮動小数点数の近似比較     | `assert_approx_eq!(pi, 3.14159, 0.001)` |

---

## パッケージマネージャーAPI

### orizon-pkg インターフェース

パッケージ管理システムのAPIです。

#### プロジェクト管理

```bash
# 新規プロジェクト作成
orizon-pkg new my_project
orizon-pkg new --lib my_library

# 既存プロジェクトの初期化
orizon-pkg init

# 依存関係の追加
orizon-pkg add math_utils@1.2.3
orizon-pkg add --dev test_helpers@0.5.0

# 依存関係の更新
orizon-pkg update
orizon-pkg update math_utils

# パッケージのビルド
orizon-pkg build
orizon-pkg build --release

# パッケージの実行
orizon-pkg run
orizon-pkg run --bin server

# パッケージの公開
orizon-pkg publish
```

#### Package.toml 設定ファイル

```toml
[package]
name = "my_awesome_project"
version = "0.1.0"
authors = ["Your Name <your.email@example.com>"]
license = "MIT"
description = "An awesome Orizon project"
repository = "https://github.com/yourname/my_awesome_project"
keywords = ["systems", "performance", "safe"]

[dependencies]
math_utils = "1.2.3"
json_parser = { version = "2.0.0", features = ["serde"] }
async_io = { git = "https://github.com/orizon/async_io.git", branch = "main" }

[dev-dependencies]
test_helpers = "0.5.0"
benchmark_tools = "1.0.0"

[build-dependencies]
build_script = "0.3.0"

[[bin]]
name = "server"
path = "src/bin/server.oriz"

[[bin]]
name = "client"
path = "src/bin/client.oriz"

[features]
default = ["json"]
json = ["json_parser"]
network = ["async_io"]
full = ["json", "network"]
```

---

## デバッガーAPI

### orizon-debug インターフェース（GDB RSP対応）

GDB Remote Serial Protocolに対応したデバッガーインターフェースです。

#### 起動と接続

```bash
# デバッガーサーバー起動
orizon-debug --target=./program --port=1234

# GDBからの接続
gdb -ex "target remote localhost:1234" ./program
```

#### デバッグコマンド例

```bash
# ブレークポイント設定
(gdb) break main
(gdb) break hello.oriz:15

# プログラム実行
(gdb) run
(gdb) continue

# ステップ実行
(gdb) step      # ソースレベルステップ
(gdb) next      # 次の行
(gdb) finish    # 関数から抜ける

# 変数の確認
(gdb) print variable_name
(gdb) info locals
(gdb) info args

# スタックトレース
(gdb) backtrace
(gdb) up
(gdb) down
```

#### デバッグ情報の形式

Orizonコンパイラが生成するDWARF v4デバッグ情報：

```
.debug_info     # DIE (Debug Information Entry) 
.debug_line     # ソースコード行番号情報
.debug_str      # 文字列テーブル
.debug_abbrev   # 略語テーブル
.debug_frame    # コールフレーム情報
```

---

## 外部システム統合API

### C ABI 互換インターフェース

#### Orizon関数のC呼び出し

```orizon
// lib.oriz
#[export]
#[no_mangle]
func add_numbers(a: i32, b: i32) -> i32 {
    return a + b;
}

#[export]
#[no_mangle]
func process_string(s: *const u8, len: usize) -> *mut u8 {
    // C文字列処理
    // ...
}
```

生成されるCヘッダー（自動生成）:

```c
// lib.h
#ifdef __cplusplus
extern "C" {
#endif

int32_t add_numbers(int32_t a, int32_t b);
uint8_t* process_string(const uint8_t* s, size_t len);

#ifdef __cplusplus
}
#endif
```

#### CライブラリのOrizon呼び出し

```orizon
// external.oriz
extern "C" {
    func malloc(size: usize) -> *mut u8;
    func free(ptr: *mut u8);
    func strlen(s: *const u8) -> usize;
    func printf(format: *const u8, ...) -> i32;
}

func use_c_library() {
    let ptr = malloc(1024);
    defer free(ptr);  // 自動クリーンアップ
    
    let message = "Hello from C!";
    printf("%s\n".as_ptr(), message.as_ptr());
}
```

### WebAssembly インターフェース

#### WebAssembly エクスポート

```orizon
// wasm_module.oriz
#[wasm_export]
func fibonacci(n: i32) -> i32 {
    if n <= 1 {
        return n;
    }
    return fibonacci(n - 1) + fibonacci(n - 2);
}

#[wasm_export]
func process_array(ptr: *mut i32, len: usize) {
    let slice = unsafe { std::slice::from_raw_parts_mut(ptr, len) };
    for i in 0..len {
        slice[i] *= 2;
    }
}
```

生成されるWebAssembly:

```wat
(module
  (func $fibonacci (param $n i32) (result i32)
    local.get $n
    i32.const 1
    i32.le_s
    if (result i32)
      local.get $n
    else
      local.get $n
      i32.const 1
      i32.sub
      call $fibonacci
      local.get $n
      i32.const 2
      i32.sub
      call $fibonacci
      i32.add
    end
  )
  (export "fibonacci" (func $fibonacci))
)
```

#### JavaScript統合

```javascript
// main.js
import init, { fibonacci, process_array } from './wasm_module.js';

async function main() {
    await init();
    
    // Orizonで実装された関数の呼び出し
    const result = fibonacci(10);
    console.log(`Fibonacci(10) = ${result}`);
    
    // メモリ共有
    const array = new Int32Array([1, 2, 3, 4, 5]);
    const ptr = wasm.malloc(array.length * 4);
    const wasmArray = new Int32Array(wasm.memory.buffer, ptr, array.length);
    wasmArray.set(array);
    
    process_array(ptr, array.length);
    console.log('Processed array:', Array.from(wasmArray));
    
    wasm.free(ptr);
}

main();
```

---

## エラー処理・ステータスコード

### 共通エラー応答形式

#### コンパイラエラー

```json
{
  "error": {
    "code": "E0001",
    "severity": "error",
    "message": "Type mismatch in function call",
    "span": {
      "file": "src/main.oriz",
      "start": {"line": 10, "column": 15},
      "end": {"line": 10, "column": 25}
    },
    "help": "Expected type `i32`, but found `String`. Consider using `parse()` method to convert string to integer.",
    "related": [
      {
        "span": {
          "file": "src/main.oriz", 
          "start": {"line": 5, "column": 10},
          "end": {"line": 5, "column": 20}
        },
        "message": "Function signature defined here"
      }
    ]
  }
}
```

#### LSPエラー

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32600,
    "message": "Invalid Request",
    "data": {
      "reason": "Missing required parameter 'textDocument'",
      "method": "textDocument/hover"
    }
  }
}
```

### HTTPステータスコード（LSP over HTTP使用時）

| コード | 意味                  | 説明                                 |
| ------ | --------------------- | ------------------------------------ |
| 200    | OK                    | 正常なレスポンス                     |
| 400    | Bad Request           | 不正なリクエスト形式                 |
| 404    | Not Found             | ファイルまたはシンボルが見つからない |
| 500    | Internal Server Error | サーバー内部エラー                   |
| 503    | Service Unavailable   | 言語サーバーが応答不能               |

---

## パフォーマンス・制限

### APIレスポンス時間

| 操作      | 期待レスポンス時間 | 最大レスポンス時間 |
| --------- | ------------------ | ------------------ |
| 字句解析  | < 1ms (1MB)        | < 10ms             |
| 構文解析  | < 5ms (1MB)        | < 50ms             |
| 型検査    | < 10ms (1000行)    | < 100ms            |
| LSP補完   | < 50ms             | < 200ms            |
| LSPホバー | < 20ms             | < 100ms            |

### メモリ使用量制限

| コンポーネント | 基本使用量 | 最大使用量 |
| -------------- | ---------- | ---------- |
| コンパイラ     | 10MB       | 1GB        |
| LSPサーバー    | 50MB       | 2GB        |
| デバッガー     | 5MB        | 500MB      |

### 同時接続制限

- LSPサーバー: 10クライアント同時接続
- デバッガー: 1デバッグセッション
- パッケージサーバー: 100同時ダウンロード

このAPIリファレンスにより、Orizonプログラミングエコシステムのすべてのインターフェースを効率的に活用することができます。
