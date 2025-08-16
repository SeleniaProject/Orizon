# stdlib/io

高水準 I/O 抽象を提供する標準ライブラリ・ファサード。

- FS: OS/Mem バックエンド。Open/Create/Mkdir/MkdirAll/Remove/RemoveAll/Stat/ReadDir/Walk
  - 追加ヘルパ: Exists/ReadFile/WriteFile/CopyFile/Rename/Move
  - Watcher: SimpleWatcher によるポーリング監視、Join/Clean のパスユーティリティ
- Net: TCP/UDP/TLS/HTTP3 を統一APIで提供（メトリクス含む）
- JSON: JSONEncode/JSONEncodeIndent/JSONDecode の薄いヘルパ

簡単な例:

```go
fs := io.OS()
_ = fs.MkdirAll("work", 0o755)
_ = fs.WriteFile("work/msg.txt", []byte("hello"), 0o644)
body, _ := fs.ReadFile("work/msg.txt")

srv := io.NewTCPServer("127.0.0.1:0")
_ = srv.Start(nil, func(c net.Conn){ defer c.Close(); c.Write([]byte("ok")) })
```
