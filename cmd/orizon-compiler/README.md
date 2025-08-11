# orizon-compiler CLI

このドキュメントでは、`cmd/orizon-compiler` が提供するコマンドラインフラグと基本的な使い方を説明します。

This document describes the command-line flags and basic usage of `cmd/orizon-compiler`.

## フラグ / Flags

- `--version`: バージョン情報を表示します。
  - Show version information.
- `--help`: ヘルプ情報を表示します。
  - Show help information.
- `--debug-lexer`: 字句解析のデバッグ出力を有効化します。
  - Enable lexer debug output.
- `--parse`: 入力を構文解析し、`parser` AST を表示します。
  - Parse the input and print the parser AST.
- `--optimize-level <level>`: AST ブリッジ経由で最適化を実行します。`none|basic|default|aggressive` を指定できます。
  - Run optimizations via the AST bridge. Levels: `none|basic|default|aggressive`.

## 使い方 / Usage

- 入力ファイルを解析のみする:
  ```sh
  orizon-compiler --parse path/to/source.oriz
  ```

- 既定レベルで最適化を有効化してASTを出力する:
  ```sh
  orizon-compiler --optimize-level default path/to/source.oriz
  ```

- 解析と最適化を同時に行う（最適化レベルは `basic` の例）:
  ```sh
  orizon-compiler --parse --optimize-level basic path/to/source.oriz
  ```

## 注意事項 / Notes

- `--optimize-level` は内部で `internal/parser` ↔ `internal/ast` の変換を行い、`internal/ast` の最適化パイプラインを適用した結果を `parser` AST として出力します。
- The `--optimize-level` flag converts between `internal/parser` and `internal/ast`, applies the `internal/ast` optimization pipeline, and prints the result as a `parser` AST.


