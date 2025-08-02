# Orizon Development - Git Strategy

## ブランチ戦略

### メインブランチ
- `main`: 安定版コード（常にリリース可能）
- `develop`: 開発版統合ブランチ

### フィーチャーブランチ
- `feature/phase-{phase}-{task}`: 各フェーズのタスク実装
- `feature/lexer-unicode`: 具体的機能実装
- `feature/error-system`: 特定システム開発

### リリースブランチ
- `release/v{major}.{minor}.{patch}`: リリース準備
- `hotfix/v{major}.{minor}.{patch+1}`: 緊急修正

## コミットメッセージ規約

### 形式
```
<type>(<scope>): <description>

<body>

<footer>
```

### タイプ
- `feat`: 新機能追加
- `fix`: バグ修正
- `docs`: ドキュメント更新
- `style`: コードスタイル修正
- `refactor`: リファクタリング
- `test`: テスト追加/修正
- `chore`: ビルド設定等

### スコープ
- `lexer`: 字句解析器
- `parser`: 構文解析器
- `typechecker`: 型検査器
- `codegen`: コード生成
- `runtime`: ランタイム
- `cli`: コマンドライン
- `lsp`: Language Server

### 例
```
feat(lexer): Unicode識別子サポート追加

絵文字を含む全Unicode文字を識別子として使用可能に
- Unicode正規化処理実装
- エラー回復機能追加
- テストケース拡充

Closes #123
```

## ワークフロー

### 1. 開発開始
```bash
git checkout develop
git pull origin develop
git checkout -b feature/phase-1-lexer-unicode
```

### 2. 開発・コミット
```bash
git add .
git commit -m "feat(lexer): Unicode正規化処理実装"
```

### 3. プルリクエスト
```bash
git push origin feature/phase-1-lexer-unicode
# GitHub上でPR作成
```

### 4. レビュー・マージ
- CI/CDパス確認
- コードレビュー実施
- `develop`ブランチにマージ

### 5. リリース
```bash
git checkout -b release/v0.1.0 develop
# バージョン番号更新
git commit -m "chore: v0.1.0リリース準備"
git checkout main
git merge --no-ff release/v0.1.0
git tag v0.1.0
```
