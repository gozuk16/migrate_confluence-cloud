.PHONY: build test coverage lint clean sync-and-build hugo-serve

# ビルド
build:
	go build -o migConfluence .

# 実行（サンプル: 単一ページ取得）
run-page:
	LOG_LEVEL=DEBUG go run . page --page-id 5931010 -c config.toml

# 実行（サンプル: スペース全ページ取得）
run-space:
	LOG_LEVEL=DEBUG go run . space --space-key SCRUM -c config.toml

# テスト
test:
	go test -v ./...

# テストカバレッジ
coverage:
	go test -cover ./...
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# リント
lint:
	golangci-lint run

clean:
	rm -f migConfluence coverage.out coverage.html debug.log

# Hugo サイト: Confluence データ取得 → ビルド → Pagefind インデックス生成
sync-and-build: build
	./migConfluence space
	mkdir -p hugo-site/content
	cp -r output/markdown/. hugo-site/content/
	cd hugo-site && hugo --minify
	npx pagefind --site hugo-site/public
	@echo "完了: hugo-site/public/ を社内サーバーに配置してください"

# Hugo 開発サーバー（ローカル確認用、Pagefind なし）
hugo-serve:
	cd hugo-site && hugo server --bind 0.0.0.0
