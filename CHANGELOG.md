# CHANGELOG

## [Unreleased]

### Added
- プロジェクト初期セットアップ（go.mod, Makefile, .gitignore, TODO.md, CHANGELOG.md）
- 設定管理（config.go）: TOML設定ファイルの読み込み・バリデーション
- Confluence REST API v2クライアント（confluenceclient.go）: Basic認証、ページ/スペース/添付/コメント/ラベル取得、カーソルページネーション対応
- 添付ファイルダウンロード（downloader.go）: Basic認証、冪等性、ファイル名サニタイズ
- Storage Format（XHTML）→ Markdown変換エンジン（converter.go）: 2段階変換アプローチ、見出し/リスト/テーブル/コード/パネル/タスクリスト/画像/リンク/絵文字対応
- XHTML中間ファイル保存・読み込み（xhtmlsaver.go）: APIレスポンスのXHTMLをそのまま中間ファイルとして保存
- Markdownファイル出力（mdwriter.go）: Hugo Leaf Bundle形式、Hugo Front Matter (TOML)生成
- CLIエントリーポイント（main.go）: page/space/convertの3コマンド、LOG_LEVEL=DEBUG対応
- README.md: プロジェクト概要、セットアップ手順、使用方法
