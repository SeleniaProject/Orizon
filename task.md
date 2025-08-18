# Orizon セルフホスト到達ロードマップ（タスクリスト）

本ファイルはセルフホスト（Orizon 自身を Orizon でビルド）達成までの実行タスクをチェックリスト形式で管理します。達成条件は以下。
- Orizon の全ソースを Orizon コンパイラでビルドできる
- 生成バイナリで `go` 実装と同等の自己ビルド（Stage2）が成功し、機能テストが緑
- CI で自己ビルドが定期的に検証される（再現性/回帰）

---

## 0. 現状サマリ（達成済み）
- [x] レキサー: 構造体系キーワード（struct/enum/trait/impl/import/export）、ライフタイム記号（`'a` 等）をサポート
- [x] パーサー: 型パースの拡充（参照/可変参照/ポインタ/配列/関数型/ジェネリクス/ライフタイム）
- [x] AST ブリッジ: 複合型の損失なし保持（pretty-print）と逆変換（round-trip）+ ユニットテスト
- [x] 全体テスト: `go test ./...` 緑（現状）
- [x] MIR/LIR の最小導入と x64 emitter（Win64 風, 診断用）
- [x] Bootstrap スナップショット/展開の E2E テスト（golden 更新+検証）
- [x] newtype のエンドツーエンド（AST/Parser/HIR/変換/テスト/仕様）
- [x] core AST に import/export ノードを追加し、AST ブリッジ往復 + ユニットテスト
- [x] impl の最大表現を HIR に導入（inherent/trait, generics, where, メソッドメタ）+ モジュールに Impls 収集
- [x] 構文仕様（spec/syntax.md）を実装に整合（fn エイリアス、newtype/export のセミコロン任意、impl_item 定義）
- [x] パーサーの宣言ブロック単位の同期点と局所回復/診断の強化（struct/enum/trait/impl/import）
- [x] パーサーのメモリガード導入（エラー/サジェスト上限、上限到達時にサジェスト停止）
- [x] ツール: orizon-fmt の最小整形を CLI/LSP で共有化（末尾空白削除・単一末尾改行・CRLF 保持）+ 単体テスト
- [x] LSP: import 破損の修復とドキュメント整形の共有化

- [x] 型検査の厳密化: Implements 制約のフル検査（インターフェース準拠: メソッド存在/署名の反変・共変/静的・インスタンス整合）
- [x] 比較/関係演算子の型規則を厳格化（同カテゴリ比較・結果の bool 保障、% の網羅）＋単体テスト追加
 - [x] 制御構造の型/効果チェック拡充（if/while/for の条件は bool、条件評価の効果包含チェック、for の init/update の型検査）
 - [x] 代入文の効果包含チェック（文自体の Effects を関数の効果集合で拘束）
 - [x] break/continue の使用位置検査（ループ外禁止）と同一ブロック内の到達不能検出
 - [x] 純粋性の強制: パラメータ既定値/配列サイズ式は副作用なし必須（不純なら効果違反）
 - [x] 純粋性の強制: グローバル変数の初期化子は純粋必須（不純なら効果違反）
 - [x] 例外構文（HIR: throw/try-catch）と型検査の統合（捕捉による例外効果の局所化、throw 後の到達不能検出、try-catch の終了判定強化）
 - [x] 効果集約の正規化（未指定 EffectSet を純として扱い、try-catch 集約での偽陽性を解消）
 - [x] サブ式評価の効果包含チェックの網羅（演算子/呼出/フィールド/インデックス/キャスト/配列/構造体の要素評価）
 - [x] 未指定 EffectSet の throw の扱い: 純粋関数ではマッチする catch が無い限り違反として報告（catch で局所処理される場合は許容）+ 回帰テスト追加
- [x] MIR 生成（SSA/基本ブロック、最小最適化）: HIR から MIR への完全変換システム、SSA 形式、基本ブロック構造、定数伝播・デッドコード削除・ブロック結合の最適化

## 1. フロントエンド（Lexer/Parser）
- [x] パーサー: 宣言パースの実装
  - struct 宣言
    - [x] フィールド/可視性
    - [x] ジェネリクス
  - enum 宣言
    - [x] バリアント（データ付き/無し）
    - [x] ジェネリクス
  - trait 宣言
    - [x] メソッドシグネチャ
    - [x] 関連型
  - impl ブロック
    - [x] 単独/trait 実装（構文）
    - [x] where 制約
  - [x] HIR への反映
  - type alias / newtype（core AST の `TypeDeclaration` と整合）
    - [x] type alias
  - [x] newtype
  - import/export
    - [x] モジュール名/パス
    - [x] 別名
    - [x] ワイルドカード（最小）
- [x] エラー回復/同期点の拡充
  - [x] 基本的な宣言スキップ復帰
  - [x] 宣言ブロック単位の同期点/良質な診断
- [x] パース単体テスト
  - [x] 正常系（宣言/型/マクロ）
  - [x] エラー系/回復系
  - [x] 基本的な回復テスト（malformed import 後の trait、malformed newtype からの復帰）

## 2. AST/AST ブリッジ
- [x] 新規宣言ノード（struct/enum/trait/impl/import/export）を core AST と合意
  - [x] 追加するか既存表現へ写像する設計確定（最小侵襲）
  - [x] import/export ノード追加（core AST）
  - [x] 方針ドキュメント追加: docs/astbridge_declaration_design.md
    - 要旨: struct/enum/trait/impl は parser/HIR で扱い core AST へは導入しない。type alias/newtype は core AST の TypeDeclaration で表現し、import/export は core AST に追加して往復対応。
- [x] パーサー AST ↔ core AST の往復変換を実装（Span/位置情報含む）
  - [x] import/export の往復変換（Span 含む往復保持を確認）
  - [x] struct/enum/trait/impl/type alias/newtype の往復変換（Span/Generics/Fields/Variants を保持）
  - [x] 関数パラメータ名など微細な識別子 Span の保持強化
- [x] 追加ユニットテスト（宣言の保持・変換の完全性）
  - [x] import/export のラウンドトリップテスト
  - [x] struct フィールド/ジェネリクス Span の往復テスト
  - [x] enum バリアント名 Span の往復テスト
  - [x] trait メソッド引数名 Span の往復テスト

## 3. 型システム/型検査
- [x] 基本型/関数/ジェネリクスの型検査（MVP: 基本型/関数/演算子/配列/構造体/最小ジェネリクス制約/参照演算子）
- [x] 参照/ポインタ/ライフタイム整合（借用規則の最大版）
  - [x] Borrow checker 実装（基本的な借用規則: 可変/不変借用の競合検出、use-after-move 検出、Copy 型の特別扱い、スコープベースの生存期間管理）
  - [x] 借用チェッカーのテストスイート（5つの主要テストケース: 基本借用、可変借用競合、move後使用、Copy型、ネストされたスコープ）
  - [x] デバッグモード対応（ORIZON_DEBUG_BORROW 環境変数でデバッグ出力制御）
- [ ] trait 境界/impl 解決（選択/内在化の最大）
  - [x] Implements 制約の型検査（インターフェース準拠チェック: メソッド存在・署名適合[反変/共変]・static/instance 一致）
  - [x] 基本的なtrait解決システム実装（TraitResolver: trait定義検索、impl検索、メソッド解決、実装制約チェック）
  - [x] trait resolverのテストスイート（基本的な型マッチング、実装制約チェック）
  - [x] 高度な実装解決（重複/選択/内在化、スコープ/優先度）
  - [x] 関連型/where 句の拘束と整合性検査
  - [x] 効果/例外仕様の整合
    - [x] 効果仕様の整合（MVP: 純粋性/包含のサブセット・呼出整合）
    - [x] 例外仕様の最小整合（関数宣言の効果包含に例外効果を含め検証／catch による局所処理を考慮）
    - [x] 例外仕様の完全整合（トレイト/impl/境界での throws 整合）
- [x] 型推論/単一化（最小スコープ）
- [x] 型検査テスト（成功/失敗/診断メッセージ）
  - [x] Implements/比較/関係演算/算術(%) の回帰テスト
  - [x] 参照/ポインタ/ミュータビリティ/アドレス可能性の回帰テスト
  - [x] 効果包含/純粋性/呼出整合の回帰テスト
  - [x] 経路感知の欠落 return/到達不能コード検出の回帰テスト
  - [x] for 文条件の型・効果テスト（非 bool 条件、純粋関数での不純条件を検出）
  - [x] for の init/update の効果テスト（純粋関数での不純初期化/更新を検出）
  - [x] break/continue: ループ外エラー/ループ内 OK、break/continue 後の同一ブロック到達不能の検出
  - [x] 純粋性: パラメータ既定値と配列サイズ式の不純検出
  - [x] 純粋性: グローバル初期化子の不純検出
  - [x] 代入禁止回帰: const と関数識別子への代入はミュータビリティ違反として検出
  - [x] 例外: throw/try-catch の回帰テスト（捕捉時の例外効果ホワイトニング、finally の効果制約、throw 後の到達不能）
  - [x] 未指定 EffectSet の throw に関する回帰テスト（純粋関数での違反検出／try-catch での許容）

## 4. HIR/MIR/LIR 変換
- ✅ HIR 変換の網羅（宣言/複合型/ジェネリクス）
  - [x] newtype の変換
  - [x] impl（inherent/trait, generics/where, メソッドメタ/モジュール集約）
- ✅ MIR 生成（SSA/基本ブロック、最小最適化: const-prop, DCE）
- [x] 参照/ライフタイムの低レベル化方針を確立（所有/借用の表現）
- [x] LIR 生成と x64 emitter 連携（最大・診断用、関数/コール/比較/分岐/メモリ）
- ✅ 段階テスト（小さな関数から e2e まで）

## 5. コード生成/ABI
- ✅ 呼出規約/スタックフレーム/レジスタ割付（最大実装）
  - [x] 呼出規約（Win64 風: rcx/rdx/r8/r9, xmm0-3, shadow space）
  - [x] スタックフレーム整列/簡易スロット割当（擬似）
  - ✅ レジスタ割付（本格）
- ✅ 配列/スライス/文字列のレイアウト定義
- ✅ 例外/パニック処理（最小; アボート戦略でも可）
- [ ] デバッグ情報（任意/後回し可）

## 6. ランタイム/標準ライブラリ（ブートストラップ）
- ✅ メモリアロケータ（最小; システム/arena ベース）
- ✅ 基本 I/O、スレッド/async プリミティブ（self-host 範囲で最小）
  - [x] ファイル I/O（開く/閉じる/読む/書く/シーク）
  - [x] コンソール I/O（標準入出力/エラー出力）
  - [x] スレッド管理（作成/開始/結合/状態管理）
  - [x] 同期プリミティブ（Mutex/RWMutex/ConditionVariable）
  - [x] チャネル通信（バッファ付き/無し、送受信）
  - [x] MIR 統合（I/O 操作の関数生成）
  - [x] x64 アセンブリ生成（Windows API 統合）
  - [x] 統計収集とリソース管理
  - [x] エラーハンドリングとタイプセーフ API
- [x] Core 型: Option, Result, Slice, String, Vec 等
  - [x] Option<T> 型の実装（Some/None バリアント、型安全チェック、モナド操作）
  - [x] Result<T,E> 型の実装（Ok/Err バリアント、エラーハンドリング）
  - [x] Slice<T> 型の実装（境界チェック、部分配列操作）
  - [x] String 型の実装（UTF-8 サポート、ハッシュ、文字列プール、連結）
  - [x] Vec<T> 型の実装（動的配列、容量管理、Push/Pop 操作）
  - [x] TypeInfo システム（サイズ/アライメント/プリミティブ型分類）
  - [x] CoreTypeManager（アロケータ統合、リソース管理）
  - [x] メモリ管理（アロケータとの統合、自動リソース解放）
  - [x] MIR 統合（プレースホルダー、コード生成準備）
  - [x] 包括的テストスイート（機能テスト、メモリ管理、ベンチマーク）
- [x] コンパイラ依存の intrinsics/extern を定義
  - [x] IntrinsicRegistry（40+ intrinsic functions with complete signatures）
  - [x] Memory management intrinsics（orizon_alloc, orizon_free, orizon_realloc, orizon_memcpy, orizon_memset）
  - [x] Atomic operations（orizon_atomic_load, orizon_atomic_store, orizon_atomic_cas）
  - [x] Bit operations（orizon_popcount, count leading/trailing zeros）
  - [x] Arithmetic with overflow（orizon_add_overflow, orizon_sub_overflow, orizon_mul_overflow）
  - [x] Compiler magic（orizon_sizeof, orizon_alignof, unreachable, assume）
  - [x] SIMD operations（vector add/sub/mul/div）
  - [x] Architecture-specific intrinsics（rdtsc, cpuid, prefetch）
  - [x] Platform support classification（All/X64/ARM64）
  - [x] ExternRegistry（C runtime functions, system calls）
  - [x] HIR integration framework（placeholder types for future HIR/MIR integration）
  - [x] Complete test suite（100% pass rate, performance benchmarks）
- [x] 単体/e2e テスト
  - [x] 包括的テストフレームワーク実装（internal/testing）
  - [x] ベンチマークシステム統合（test/benchmark）
  - [x] E2Eテストインフラ（test/e2e、test/integration、test/unit）
  - [x] 自動レポート生成とCI統合

## 7. ツール/エコシステム
- [x] orizon-fmt の安定化（AST 対応拡充、差分整形）
  - [x] 最小整形（末尾空白/末尾改行/CRLF 保持）を実装し CLI/LSP で共有化
  - [x] 単体テスト整備（CLI/内部パッケージ）
  - [x] AST 対応整形（インデント・構文ベース）
  - [x] 差分整形／範囲整形の精度改善
  - [x] 改行スタイル強制オプション（LF/CRLF）
  - [x] ASTFormattingOptions実装（インデント制御、タブ設定、行長制限、フィールド整列、演算子周りスペース、末尾カンマ管理）
  - [x] DiffFormatter実装（unified/context/side-by-side差分モード、設定可能コンテキスト行、行番号表示、Myersアルゴリズム概念）
  - [x] トークンベースフォールバック（AST解析不可時の字句解析ベース整形）
  - [x] orizon-fmt CLI拡張（-ast、-diff、-mode フラグによる高度整形機能）
- [x] orizon-lsp: 定義へ移動/シンボル/補完（最小）
  - [x] ドキュメント全体整形を共有フォーマッタに移行
  - [x] range/onType formatting の差分最小化（必要最小の編集範囲に限定）
  - [x] AST対応フォーマッティング（enhanced mode、-ast フラグサポート）
  - [x] 差分ベース最小編集（DiffFormatterによる最小変更検出と適用）
  - [x] 拡張されたonTypeFormatting（智能的文字入力対応、AST aware mode）
- [x] orizon-test: ランナー/スナップショット/ゴールデン
  - [x] 既存テストランナー拡張（snapshot/golden file テスト機能）
  - [x] SnapshotManager実装（自動スナップショット比較、更新モード、cleanup機能）
  - [x] Golden file testing（期待値ファイル比較テスト）
  - [x] CLIフラグ追加（--update-snapshots、--cleanup-snapshots、--golden、--snapshot-dir）
  - [x] 差分生成とレポート機能（スナップショットテスト結果統計）
- [x] パッケージマネージャ: ローカル解決・ビルドの最小
  - [x] LocalManager実装（package.oriz マニフェスト管理）
  - [x] ローカル依存関係解決（相対パス、workspace packages、cache対応）
  - [x] 基本ビルド機能（build-info.json生成、依存関係メタデータ）
  - [x] CLI実装：orizon-pkg（init、add、remove、install、build、clean、list）
  - [x] パッケージライフサイクル管理（manifest作成、依存関係追加・削除、ビルド成果物管理）

## 8. セルフホスト E2E ステージ
- [x] Stage0: Go 実装で Orizon をビルド
  - [x] 自動化スクリプト作成（Windows/Unix対応）
  - [x] 7つのツール全て正常ビルド（orizon, orizon-compiler, orizon-bootstrap, orizon-fmt, orizon-lsp, orizon-test, orizon-pkg）
  - [x] ツール検証とメタデータ生成
  - [x] 包括的テスト実行（一部コンパイルエラーあるが実行可能）
- [ ] Stage1: Orizon で Orizon をビルド（成功、テスト実行）
  - [x] Stage1スクリプト作成（実機能Orizonコンパイラ待ち）
- [ ] Stage2: Stage1 バイナリで再ビルドし差分比較/同一性検証
  - [x] Stage2比較スクリプト作成（バイナリ同一性検証）
- [ ] 再現性ビルド（タイムスタンプ/埋め込み値の制御）

## 9. 品質ゲート/CI
- [x] 単体テストパス（`go test ./...`）
- [x] Lint/Format の CI 化（PR ブロッカー）
- [x] e2e セルフホストジョブ（夜間/タグ時）
- [x] 失敗時のログ/成果物保存
- [x] Windows/Linux の `go test` を GitHub Actions に追加（.github/workflows/go-ci.yml）
- [x] LSP/フォーマッタのスモーク E2E（簡易クライアントで整形検証）
  - [x] orizon-smoke-test実装（フォーマッタ直接テスト、LSP JSON-RPC通信テスト）
  - [x] CI統合（lint-formatジョブでスモークテスト実行）
  - [x] ビルドログ収集スクリプト（失敗時診断情報自動収集）

## 10. パフォーマンス/安定化
- [x] 代表ベンチ（パース/型検査/コード生成）
  - [x] パーサーベンチマーク（test/benchmark/compiler_bench_test.go）
  - [x] 型検査ベンチマーク（HIR変換含む）
  - [x] コード生成ベンチマーク（MIR/LIR/x64）
  - [x] メモリ使用量プロファイリング
  - [x] CI統合とパフォーマンス回帰検出
- [ ] 大規模入力での増分解析の実測
- [ ] メモリプロファイリング/ホットスポット最適化

## 11. ドキュメント/運用
- [ ] 言語仕様の凍結対象とバージョニング方針
- [ ] 開発者ガイド/貢献ガイド/リリース手順
- [ ] ユーザー向けクイックスタート/トラブルシュート

---

## 直近の実装順（推奨）
1) パーサー宣言群の実装 + テスト
2) AST ブリッジで宣言の往復対応（完了）
3) HIR 型/宣言対応の拡張
4) 最小 self-host subset（必要言語機能/標準ライブラリ）のスコープ確定
5) Stage1 ビルドまでを e2e で通して課題洗い出し
6) orizon-fmt: AST 対応整形/差分整形の基礎（共有フォーマッタ継続）
7) LSP: range/onType formatting の差分最小化と E2E スモーク
8) CI: Windows/Linux で `go test` + フォーマット検証を追加
