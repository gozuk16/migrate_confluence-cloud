# CHANGELOG

## [Unreleased]

### Changed（ADF移行）
- ページ取得フォーマットを `body-format=storage`（独自 XHTML）から `body-format=atlas_doc_format`（ADF JSON）に移行
- 中間ファイルを `content.xhtml` から `content.json`（ADF JSON）に変更
- ページ本文変換パイプラインを `Converter.Convert()` (XHTML→HTML→MD) から `Converter.ConvertADF()` (ADF JSON→MD 直接変換) に切り替え
- `config.toml` の `html_dir` 設定を廃止

### Removed（ADF移行）
- HTML 出力機能（`HTMLWriter`、`Converter.ToHTML()`）を廃止。Markdown 出力のみに統一
- `config.go` の `HTMLDir` フィールドを削除

### Added（ADF移行）
- `adfconverter.go`: ADF JSON → Markdown 直接変換エンジン
  - テキスト/マーク変換（bold, italic, code, strike, underline, link, sub/superscript）
  - 見出し・リスト（bulletList, orderedList, ネスト対応）・引用・水平線
  - コードブロック（言語指定付き）
  - パネル → GFM Alerts（info→`[!NOTE]`, note→`[!WARNING]`, warning/error→`[!CAUTION]`, success→`[!TIP]`）
  - テーブル（tableHeader/tableCell 対応）
  - タスクリスト・decisionList・expand（`<details>`/`<summary>`）
  - status（色絵文字）・mention・emoji・date（エポックms→ISO日付）
  - メディア（external URL・添付ファイル UUID解決）・レイアウト・extension・card
  - Confluence 内部 URL を相対パスに変換
- `Converter.ConvertADF()`: ADF JSON → Markdown のパブリック API
- `buildAttachmentMap()`: 添付ファイル UUID → ファイル名マップ構築ヘルパー

### Changed
- `XHTMLSaver` → `IntermediateSaver` にリネーム（中間ファイルの役割を明示）
- CLIフラグ `--save-xhtml` → `--save-intermediate` に変更
- 設定キー `xhtml_dir` → `intermediate_dir` に変更、デフォルト `output/intermediate`

### Added
- `Converter.ToHTML()`: Confluence Storage Format → 標準 HTML ボディフラグメント変換
- `HTMLWriter`: HTML Living Standard 出力（`output/html/{SPACE}/{PAGE}/index.html`）
  - ページタイトル・メタ情報・ラベル・Confluence URL のヘッダー
  - コメントセクション
  - 最低限の CSS スタイリング（テーブル・コードブロック・パネル・details）
- 設定キー `html_dir` を追加（デフォルト: `output/html`）
- `page` / `space` / `convert` コマンドで Markdown と HTML を同時出力
- 変換品質改善（converter.go）:
  - GFM Alerts対応: `> [!NOTE]` / `> [!WARNING]` / `> [!CAUTION]` / `> [!TIP]` 形式への変換
  - タスクリスト修正: `- [ ]` / `- [x]` のGFMチェックボックス形式に変換
  - 追加マクロ対応: noformat, quote, section, column, anchor, excerpt, details, jira, widget, gallery, multimedia, jirachart, children, pagetree, recently-updated, blog-posts, contentbylabel, excerpt-include, include
  - 追加要素対応: fieldset, ac:inline-comment-marker, ac:placeholder, ac:layout/section/cell（ネストレイアウト解除）
  - strikethrough（`~~text~~`）プラグイン追加
  - 未対応要素レポート出力（unsupported_elements.md）: 未変換のマクロ・要素名と出現回数を記録
  - ステータスマクロに色絵文字（🟢🟡🔴🔵🟣⚫）を追加
- プロジェクト初期セットアップ（go.mod, Makefile, .gitignore, TODO.md, CHANGELOG.md）
- 設定管理（config.go）: TOML設定ファイルの読み込み・バリデーション
- Confluence REST API v2クライアント（confluenceclient.go）: Basic認証、ページ/スペース/添付/コメント/ラベル取得、カーソルページネーション対応
- 添付ファイルダウンロード（downloader.go）: Basic認証、冪等性、ファイル名サニタイズ
- Storage Format（XHTML）→ Markdown変換エンジン（converter.go）: 2段階変換アプローチ、見出し/リスト/テーブル/コード/パネル/タスクリスト/画像/リンク/絵文字対応
- XHTML中間ファイル保存・読み込み（xhtmlsaver.go）: APIレスポンスのXHTMLをそのまま中間ファイルとして保存
- Markdownファイル出力（mdwriter.go）: Hugo Leaf Bundle形式、Hugo Front Matter (TOML)生成
- CLIエントリーポイント（main.go）: page/space/convertの3コマンド、LOG_LEVEL=DEBUG対応
- README.md: プロジェクト概要、セットアップ手順、使用方法
