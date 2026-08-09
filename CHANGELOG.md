# CHANGELOG

## [Unreleased]

### Added（サイドバー階層表示）
- 既存の出力には階層情報が含まれないため、本機能を反映するには移行の再実行が必要（`make sync-and-build` など）
- 左サイドバーのページ一覧をConfluence同様の階層ツリー表示に変更（従来はフラットな一覧）。子を持つ項目は `<details>`/`<summary>` による開閉式（JavaScript不要）で、現在表示中のページの祖先フォルダは自動的に展開され、現在ページはハイライトされる
- Confluenceのフォルダに対応。フォルダはリンクではなく、クリックで開閉のみ行うグループラベルとして表示（ページを1つも含まない空のフォルダはサイドバーに表示されない）
- Confluence REST API v2 の `GET /wiki/api/v2/folders/{id}` を叩く `GetFolder` を追加
- ページの親を辿って必要なフォルダだけを再帰的に収集する `CollectFolders` を追加（foldertree.go）。親が取得できないページ（権限不足や未対応のコンテンツタイプが親の場合など）はルート直下に表示され、移行ログに警告が出る
- ページのフロントマターに `parent_id`（親のID）と `weight`（並び順）を追加
- フォルダを `is_folder = true` と `[build] render = "never"` を持つ「レンダリングされないページ」スタブとして出力（URLもHTMLも生成されないが、サイドバーのツリーには現れる）
- 中間ファイルのメタデータに `parent_type` と `position` を追加し、convert コマンド（オフライン再変換）でもページの親子関係（`parent_id`）と並び順（`weight`）が維持されるように対応。ただしフォルダスタブは中間ファイルに保存されないため、convert コマンド単独ではフォルダ配下のページの親を解決できずルート直下に並ぶ（フォルダスタブの出力は space コマンドのみ）

### Changed（サイドバー階層表示）
- サイドバーの並び順をConfluenceでの並び順（position）を再現するように変更（従来はタイトルの文字コード順）。`weight` を持たないページ（手書きで追加したページなど）は Hugo の仕様上サイドバーの先頭に並ぶ
- URLとディレクトリ構造は従来どおりフラットのまま変更なし（既存リンクを壊さないための設計判断）

### Fixed（サイドバー階層表示）
- 中間ファイルから読み戻す際に親ID（ParentID）が復元されていなかった不具合を修正

### Fixed
- ADFの `strong`/`em`/`strike` マークが前後に空白を含むテキストに付与された場合、CommonMarkのデリミタフランキング規則により装飾が閉じられずリテラル `**text **` のように表示される不具合を修正（`wrapDelimiter` ヘルパーで空白をデリミタ外に退避）。NOTEパネル内の見出し太字などで顕在化していた。
- リスト項目内の `codeBlock` が Markdown 出力から欠落する不具合を修正（`renderListItem` に codeBlock 対応を追加）
- 入れ子のタスクリスト（チェックボックス）が Markdown 出力から欠落する不具合を修正（`renderTaskList` の再帰対応）
- 番号付きリストの入れ子が 2 スペースインデントで分断される不具合を修正（マーカー幅ベースの累積インデント計算に変更）
- 隣接する同種強調 run（`em` / `strong` / `strike` など）が `*あけ**ぼ**の*` のように化ける不具合を修正（`renderInlineNodes` でグループ化し 1 組のデリミタで囲む）
- ADF の `textColor` 属性が失われる不具合を修正（`<span style="color: #RRGGBB">` による HTML 埋め込みで再現）
- ADF の `alignment`（中央寄せ・右寄せ）が失われる不具合を修正（`<div style="text-align: center|right">` による HTML 埋め込みで再現）
- Confluence コメント投稿者が `accountId` のまま表示される不具合を修正（`GetUserDisplayName` を MDWriter にワイヤリングし表示名を出力）。convert コマンドはオフライン動作のため deletedUsers マッピングのみで解決する（API 解決は page/space コマンドのみ）
- フッターコメントへのリプライ（子コメント）が取得されない不具合を修正（`/footer-comments/{id}/children` を再帰取得し、`#### コメント 1-1` のような階層見出しで出力）。中間ファイル経由の convert コマンドでは階層情報が保存されないためフラット表示となる（既知の制限）
- `status` マクロが `🔵[ok]` のような絵文字テキストになる問題を修正（Confluence 風 lozenge バッジをインライン CSS の `<span>` で再現、テキストは HTML エスケープ）

### Added（ADFテーブル変換強化）
- テーブルセル内のリスト・引用・複数段落・コードブロック・タスクリストを HTML 埋め込みで変換
- テーブルの縦結合（rowspan）・横結合（colspan）をグリッド展開で近似（列ずれ解消）
- 入れ子テーブル（nested-table 拡張）をセル内 HTML テーブルとして再帰変換
- セルの配置（alignment マーク）を GFM 列アライメント記法に反映
- ヘッダー無しテーブルに空ヘッダー行を自動生成

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
