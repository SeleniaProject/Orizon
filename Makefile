# Orizon Programming Language - Makefile
# Phase 0.1.1: 開発環境セットアップの自動化

.PHONY: help build test clean dev docker-dev install-tools fmt lint smoke smoke-win smoke-mac bootstrap bootstrap-golden bootstrap-verify

# Cross-platform env prefix for disabling CGO per command
ifeq ($(OS),Windows_NT)
	SETCGO := set CGO_ENABLED=0 &&
	EXE := .exe
else
	SETCGO := CGO_ENABLED=0
	EXE :=
endif

# Self-host snapshot対象（現時点で安定してパースできる最小セット）
SELFHOST_EXAMPLES := \
	bootstrap_samples \
	examples/hello.oriz \
	examples/simple.oriz \
	examples/macro_example.oriz

# デフォルトターゲット
help: ## このヘルプメッセージを表示
	@echo "Orizon Programming Language - 開発コマンド"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

# ビルド関連
build: ## コンパイラをビルド
	@echo "🔨 Orizonコンパイラをビルド中..."
	@$(SETCGO) go build -o build/orizon-compiler$(EXE) ./cmd/orizon-compiler
	@$(SETCGO) go build -o build/orizon-lsp$(EXE) ./cmd/orizon-lsp
	@$(SETCGO) go build -o build/orizon-fmt$(EXE) ./cmd/orizon-fmt
	@$(SETCGO) go build -o build/orizon-fuzz$(EXE) ./cmd/orizon-fuzz
	@$(SETCGO) go build -o build/orizon-repro$(EXE) ./cmd/orizon-repro
	@$(SETCGO) go build -o build/orizon-test$(EXE) ./cmd/orizon-test
	@echo "✅ ビルド完了"

build-all: ## 全ツールをビルド（CI用）
	@echo "🔨 全Orizonツールをビルド中..."
	@$(SETCGO) go build -o build/orizon$(EXE) ./cmd/orizon
	@$(SETCGO) go build -o build/orizon-compiler$(EXE) ./cmd/orizon-compiler
	@$(SETCGO) go build -o build/orizon-bootstrap$(EXE) ./cmd/orizon-bootstrap
	@$(SETCGO) go build -o build/orizon-lsp$(EXE) ./cmd/orizon-lsp
	@$(SETCGO) go build -o build/orizon-fmt$(EXE) ./cmd/orizon-fmt
	@$(SETCGO) go build -o build/orizon-fuzz$(EXE) ./cmd/orizon-fuzz
	@$(SETCGO) go build -o build/orizon-repro$(EXE) ./cmd/orizon-repro
	@$(SETCGO) go build -o build/orizon-test$(EXE) ./cmd/orizon-test
	@$(SETCGO) go build -o build/orizon-pkg$(EXE) ./cmd/orizon-pkg
	@echo "✅ 全ツールビルド完了"

bootstrap: ## ブートストラップ補助ツールをビルド
	@echo "🔨 orizon-bootstrap をビルド中..."
	@$(SETCGO) go build -o build/orizon-bootstrap$(EXE) ./cmd/orizon-bootstrap
	@echo "✅ orizon-bootstrap ビルド完了"

bootstrap-golden: bootstrap ## SELFHOST_EXAMPLES からスナップショット生成→ゴールデン更新
	@echo "📸 ゴールデン更新中..."
	@./build/orizon-bootstrap$(EXE) --out-dir artifacts/selfhost --golden-dir test/golden/selfhost --update-golden $(SELFHOST_EXAMPLES)
	@echo "✅ ゴールデン更新完了"

bootstrap-verify: bootstrap ## 生成物とゴールデンの差分検証
	@./build/orizon-bootstrap$(EXE) --out-dir artifacts/selfhost --golden-dir test/golden/selfhost $(SELFHOST_EXAMPLES)
	@echo "✅ bootstrap verify OK"
fuzz-parser-sample: build ## 簡易パーサーファズ（小コーパス）
	@echo "🧪 Parser fuzz (sample corpus) ..."
	@./build/orizon-fuzz$(EXE) --target parser --duration 5s --p 2 --corpus corpus/parser_corpus.txt --covout fuzz.cov --covstats --out crashes.txt --min-on-crash --min-dir crashes_min --min-budget 2s

fuzz-lexer-sample: build ## 簡易レキサーファズ（小コーパス）
	@echo "🧪 Lexer fuzz (sample corpus) ..."
	@./build/orizon-fuzz$(EXE) --target lexer --duration 5s --p 2 --corpus corpus/lexer_corpus.txt --covstats --per 200ms --min-on-crash --min-dir crashes_min --min-budget 2s --out crashes.txt

fuzz-astbridge-sample: build ## 簡易ASTブリッジファズ（小コーパス）
	@echo "🧪 AST bridge fuzz (sample corpus) ..."
	@./build/orizon-fuzz$(EXE) --target astbridge --duration 5s --p 2 --corpus corpus/astbridge_corpus.txt --covstats --per 300ms --min-on-crash --min-dir crashes_min --min-budget 2s --out crashes.txt

fuzz-hir-sample: build ## HIR変換+検証ファズ（小コーパス）
	@echo "🧪 HIR fuzz (transform + validate) ..."
	@./build/orizon-fuzz$(EXE) --target hir --duration 5s --p 2 --corpus corpus/parser_corpus.txt --covstats --per 300ms --min-on-crash --min-dir crashes_min --min-budget 2s --out crashes.txt

fuzz-astbridge-hir-sample: build ## ASTブリッジ往復後にHIR検証（小コーパス）
	@echo "🧪 AST bridge + HIR validate fuzz ..."
	@./build/orizon-fuzz$(EXE) --target astbridge-hir --duration 5s --p 2 --corpus corpus/astbridge_corpus.txt --covstats --per 300ms --min-on-crash --min-dir crashes_min --min-budget 2s --out crashes.txt

repro-last-crash: build ## crashes.txtの最終クラッシュを再現
	@echo "🔁 Reproduce last crash from crashes.txt ..."
	@./build/orizon-repro$(EXE) --log crashes.txt --budget 5s --target parser

minimize-last-crash: build ## crashes.txtの最終クラッシュを最小化
	@echo "🪄 Minimize last crash from crashes.txt ..."
	@./build/orizon-repro$(EXE) --log crashes.txt --out minimized.bin --budget 5s --target parser

build-release: ## リリース用ビルド（最適化有効）
	@echo "🚀 リリース用ビルド中..."
	@$(SETCGO) go build -ldflags="-s -w" -o build/orizon-compiler$(EXE) ./cmd/orizon-compiler
	@$(SETCGO) go build -ldflags="-s -w" -o build/orizon-lsp$(EXE) ./cmd/orizon-lsp
	@$(SETCGO) go build -ldflags="-s -w" -o build/orizon-fmt$(EXE) ./cmd/orizon-fmt
	@echo "✅ リリースビルド完了"

# テスト関連
test: ## 単体テストを実行
	@echo "🧪 単体テスト実行中..."
	@go test ./...
	@echo "✅ テスト完了"

smoke: ## LSP/フォーマッタのスモークテスト実行
	@echo "💨 スモークテスト実行中..."
	@go run ./cmd/orizon-smoke-test
	@echo "✅ スモークテスト完了"
test: ## 全テストを実行
	@echo "🧪 テスト実行中..."
	@$(SETCGO) go test -v ./...
	@echo "✅ テスト完了"

test-coverage: ## カバレッジ付きテスト実行
	@echo "📊 カバレッジテスト実行中..."
	@$(SETCGO) go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ カバレッジレポート生成: coverage.html"

benchmark: ## ベンチマークテスト実行
	@echo "⚡ ベンチマーク実行中..."
	@$(SETCGO) go test -bench=. -benchmem ./...

# 開発環境
dev: ## 開発モードでコンパイラを起動
	@echo "🔄 開発モード起動中..."
	@$(SETCGO) go run ./cmd/orizon-compiler --help

docker-dev: ## Docker開発環境を起動
	@echo "🐳 Docker開発環境起動中..."
	@docker-compose -f docker-compose.dev.yml up -d
	@echo "✅ 開発環境起動完了"
	@echo "接続: docker-compose -f docker-compose.dev.yml exec orizon-dev bash"

docker-stop: ## Docker開発環境を停止
	@docker-compose -f docker-compose.dev.yml down

# コード品質
fmt: ## コードフォーマット
	@echo "🎨 コードフォーマット中..."
	@go fmt ./...
	@echo "✅ フォーマット完了"

lint: ## コード品質チェック
	@echo "🔍 コード品質チェック中..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint がインストールされていません"; \
		echo "インストール: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

install-tools: ## 開発ツールをインストール
	@echo "🛠️ 開発ツールインストール中..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/go-delve/delve/cmd/dlv@latest
	@echo "✅ ツールインストール完了"

# クリーンアップ
clean: ## ビルド成果物をクリーンアップ
	@echo "🧹 クリーンアップ中..."
	@rm -rf build/
	@rm -f coverage.out coverage.html
	@go clean -cache
	@echo "✅ クリーンアップ完了"

# サンプル実行
examples: build ## サンプルコードを実行
	@echo "📝 サンプル実行中..."
	@./build/orizon-compiler$(EXE) examples/hello.oriz
	@echo "✅ サンプル実行完了"

# ドキュメント生成
docs: ## ドキュメント生成
	@echo "📚 ドキュメント生成中..."
	@go doc -all ./... > docs/api.md
	@echo "✅ ドキュメント生成完了"

# CI/CDチェック
ci: install-tools fmt lint test ## CI環境での全チェック実行
	@echo "🚀 CI/CDチェック完了"

# スモーク（ローカル）
smoke: ## Linux/macOS 向けスモーク（tests/fuzz/repro 集約）
	@bash ./scripts/linux/smoke.sh

smoke-win: ## Windows 向けスモーク（PowerShell）
	@powershell -ExecutionPolicy Bypass -File .\scripts\win\smoke.ps1

smoke-mac: smoke ## macOS 向けスモーク（Linuxと同一スクリプトを利用）

# バージョン情報
version: ## バージョン情報表示
	@echo "Orizon Programming Language v0.1.0-alpha"
	@echo "Go version: $(shell go version)"
	@echo "Build date: $(shell date -u +%Y-%m-%dT%H:%M:%SZ)"
