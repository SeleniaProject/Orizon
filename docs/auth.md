# レジストリ認証ガイド

## 方針（まず面倒な設定は不要）

- 既定モードは `write`: 書き込み（/publish）のみ認証必須。読み取りは公開（トークン不要）。
- 必要なときだけ `readwrite` に切り替えてトークンを使う。
- トークンは Bearer（平文）。本番は必ず HTTPS を使う。

## サーバ設定（PowerShell）

```powershell
# 読取は公開、書込は保護（既定）
$env:ORIZON_REGISTRY_AUTH_MODE = "write"
$env:ORIZON_REGISTRY_TOKEN = "s3cr3t"
./build/orizon.exe pkg serve --addr :9321

# 全エンドポイントを保護したい場合
$env:ORIZON_REGISTRY_AUTH_MODE = "readwrite"
$env:ORIZON_REGISTRY_TOKEN = "s3cr3t"
./build/orizon.exe pkg serve --addr :9321
```

### HTTPS 配信（推奨: トークン併用時）

```powershell
# PEM証明書/鍵を用意後（自己署名でも可）
./build/orizon.exe pkg serve --addr :9321 --tls-cert .\.certs\server.crt --tls-key .\.certs\server.key --token s3cr3t
```
クライアント側は `https://` のURLを `ORIZON_REGISTRY` に設定してください。

## クライアント設定（必要なときだけ）

```powershell
$env:ORIZON_REGISTRY = "http://localhost:9321"
./build/orizon.exe pkg auth login --registry http://localhost:9321 --token s3cr3t

# 以降は自動でトークンが使われる
./build/orizon.exe pkg list
```

`credentials.json` は `.orizon/credentials.json` に保存されます:

```json
{
  "registries": {
    "http://localhost:9321": { "token": "s3cr3t" }
  }
}
```

## 環境変数での直接指定（任意・簡易）

```powershell
$env:ORIZON_REGISTRY = "http://localhost:9321"
$env:ORIZON_REGISTRY_TOKEN = "s3cr3t"
./build/orizon.exe pkg list
```

## トークンの作り方（例）

シンプルに十分な長さの乱数文字列を使えばOK（生成元は任意）。

PowerShell 一例:

```powershell
[Convert]::ToBase64String((New-Object byte[] 32 | %{$_=0}; [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($_); $_))
```

OpenSSL 一例:

```powershell
openssl rand -base64 32
```

## トラブルシュート

- 401 Unauthorized のとき
  - サーバが `readwrite` になっていないか（読取でもトークンが要る）
  - トークンが未登録なら `pkg auth login` で登録
  - サーバとクライアントのトークンが一致しているか
  - `credentials.json` のレジストリURLが実URLと一致しているか（末尾の `/` は付けない）
