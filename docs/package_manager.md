# パッケージ管理 詳細ガイド

このドキュメントは `orizon pkg` サブコマンド群の詳細と、ローカル/リモートレジストリ運用、署名、ベンダリング、トラブルシュートをまとめたものです。

## 用語
- マニフェスト: `orizon.json`
- ロックファイル: `orizon.lock`
- レジストリ: パッケージの保存と検索を担うバックエンド（ファイル or HTTP）
- CID: Content ID。パッケージ内容から算出されるコンテンツハッシュ

## レジストリの選択
環境変数 `ORIZON_REGISTRY` の値で自動切り替え:
- 未設定: `.\ .orizon\registry` を使うファイルレジストリ
- `http://` or `https://` で始まる場合: HTTP レジストリ

HTTP の場合、`ORIZON_REGISTRY_TOKEN` が設定されているとクライアントは自動で `Authorization: Bearer <token>` を付与します。サーバ側も同環境変数が設定されている場合のみトークンを要求します。

## コマンド一覧

### init
`orizon.json` を作成（既存時は何もしない）。

### add
依存を追加。例: `--dep foo@^1.2.0`

### resolve / lock / verify
- resolve: 現在の制約に対する解決結果を表示
- lock: 解決した結果から `orizon.lock` を作成
- verify: ロックファイルの妥当性を検証

### list / fetch
- list: レジストリの全パッケージ（name@version）を列挙
- fetch: name@constraint をダウンロードし `.orizon/cache/<cid>` に保存

### update / remove
- update: 依存を再解決してロックを書き換え
- remove: 依存削除（必要ならロックも再生成）

### graph / why / outdated
- graph: 依存グラフを表示。`--dot` で Graphviz DOT を出力
- why: 根から対象への経路を表示
- outdated: current / allowed / latest を一覧表示

### vendor
ロックファイルに基づいて `.orizon/vendor` に全パッケージを保存（オフラインビルド向け）

### sign / verify-sig / audit
- sign: CID を自己署名（デモ）。署名バンドルを `.orizon/signatures` に保存
- verify-sig: バンドル内のルート鍵を信頼して署名検証（デモ）
- audit: 既知のアドバイザリに基づく脆弱性監査（デモ）

### serve
ローカルファイルレジストリを HTTP で提供。
- `--addr :9321` でアドレス指定
- `--token <secret>` で Bearer 認証を有効化（内部的に `ORIZON_REGISTRY_TOKEN` を設定）

## よくある質問 / トラブルシュート

- Q: 401 Unauthorized が返る
  - A: サーバ側がトークンを要求しています。`$env:ORIZON_REGISTRY_TOKEN = "<token>"` を設定してください。
- Q: find は成功するが fetch が 404 になる
  - A: レジストリに CID 対応の blob が存在しない可能性があります。`pkg publish` ログとレジストリの `blobs/` を確認してください。
- Q: 解決が遅い
  - A: HTTP クライアントは接続再利用とリトライを行います。ネットワーク遅延が大きい場合、レジストリを近傍に配置し、`vendor` を活用してください。

## ベストプラクティス
- CI では常に `pkg lock` を生成し、成果物として保存
- リリース前に `pkg verify` / `verify-sig` を実行
- モノリポでは `vendor` ディレクトリをキャッシュし、オフラインビルドに備える

---

最終更新: 2025-08-16
