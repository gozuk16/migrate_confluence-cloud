.PHONY: build test coverage lint clean

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
