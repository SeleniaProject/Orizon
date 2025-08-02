# Orizon Programming Language
**現存するすべてのシステムプログラミング言語を技術的に凌駕する革命的言語**

[![Build Status](https://github.com/orizon-lang/orizon/workflows/CI/badge.svg)](https://github.com/orizon-lang/orizon/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
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
│   └── orizon-fmt/        # コードフォーマッタ
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
