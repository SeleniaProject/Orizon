# Orizon Programming Language - Makefile
# Phase 0.1.1: 開発環境セットアップの自動化

.PHONY: help build test clean dev docker-dev install-tools fmt lint

# デフォルトターゲット
help: ## このヘルプメッセージを表示
	@echo "Orizon Programming Language - 開発コマンド"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

# ビルド関連
build: ## コンパイラをビルド
	@echo "🔨 Orizonコンパイラをビルド中..."
	@CGO_ENABLED=0 go build -o build/orizon-compiler ./cmd/orizon-compiler
	@CGO_ENABLED=0 go build -o build/orizon-lsp ./cmd/orizon-lsp
	@CGO_ENABLED=0 go build -o build/orizon-fmt ./cmd/orizon-fmt
	@echo "✅ ビルド完了"

build-release: ## リリース用ビルド（最適化有効）
	@echo "🚀 リリース用ビルド中..."
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o build/orizon-compiler ./cmd/orizon-compiler
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o build/orizon-lsp ./cmd/orizon-lsp
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o build/orizon-fmt ./cmd/orizon-fmt
	@echo "✅ リリースビルド完了"

# テスト関連
test: ## 全テストを実行
	@echo "🧪 テスト実行中..."
	@CGO_ENABLED=0 go test -v ./...
	@echo "✅ テスト完了"

test-coverage: ## カバレッジ付きテスト実行
	@echo "📊 カバレッジテスト実行中..."
	@CGO_ENABLED=0 go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ カバレッジレポート生成: coverage.html"

benchmark: ## ベンチマークテスト実行
	@echo "⚡ ベンチマーク実行中..."
	@CGO_ENABLED=0 go test -bench=. -benchmem ./...

# 開発環境
dev: ## 開発モードでコンパイラを起動
	@echo "🔄 開発モード起動中..."
	@CGO_ENABLED=0 go run ./cmd/orizon-compiler --help

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
	@./build/orizon-compiler examples/hello.oriz
	@echo "✅ サンプル実行完了"

# ドキュメント生成
docs: ## ドキュメント生成
	@echo "📚 ドキュメント生成中..."
	@go doc -all ./... > docs/api.md
	@echo "✅ ドキュメント生成完了"

# CI/CDチェック
ci: install-tools fmt lint test ## CI環境での全チェック実行
	@echo "🚀 CI/CDチェック完了"

# バージョン情報
version: ## バージョン情報表示
	@echo "Orizon Programming Language v0.1.0-alpha"
	@echo "Go version: $(shell go version)"
	@echo "Build date: $(shell date -u +%Y-%m-%dT%H:%M:%SZ)"
