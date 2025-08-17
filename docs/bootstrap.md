# ブートストラップ手順（雛形）

本書は、Orizon のブートストラップ（自己ホスト準備）の再現手順を固定するための雛形です。

## 前提
- Go がインストール済み
- 本リポジトリをチェックアウト済み

## スナップショット生成とゴールデン検証
1) ツールをビルド
```
make bootstrap
```
2) 例からスナップショットを生成し、ゴールデンを更新
```
make bootstrap-golden
```
3) 検証実行（差分がないことを確認）
```
./build/orizon-bootstrap --out-dir artifacts/selfhost --golden-dir test/golden/selfhost examples
```

### 実験: マクロ展開パス（オプション）
- 事前にマクロを展開してからASTブリッジする実験的フラグです。
- 既存ゴールデンに影響を与えないよう、デフォルトはOFFです。
- 注意: フラグは入力ファイル/ディレクトリ指定より前に置いてください。

例（検証のみ・ゴールデン比較なし）:
```
./build/orizon-bootstrap --out-dir artifacts/selfhost_expanded --expand-macros examples/macro_example.oriz
```

## CI チェック（推奨）
- `test/selfhost_bootstrap_test.go` が最小標本での生成/比較を自動検証
- 将来的に self-hosting サンプルを増やした場合、同テストの対象を追加

## 参考
- `cmd/orizon-bootstrap`
- `artifacts/selfhost`, `test/golden/selfhost`
- `--expand-macros`（実験）
- `docs/self_hosting_partial.md`, `docs/self_hosting_full.md`
