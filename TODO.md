# TODO

## 進行中

（なし）

## 未着手

（なし）

## 完了

- [x] Step 1: プロジェクト基盤セットアップ
- [x] Step 2: 設定管理（config.go）
- [x] Step 3: Confluence APIクライアント（confluenceclient.go）
- [x] Step 4: 添付ファイルダウンロード（downloader.go）
- [x] Step 5: Storage Format → Markdown変換エンジン（converter.go）
- [x] Step 6: XHTML中間ファイル保存・読み込み（xhtmlsaver.go）
- [x] Step 7: Markdownファイル出力（mdwriter.go）
- [x] Step 8: CLIエントリーポイント（main.go）
- [x] Step 9: README.md作成
- [x] Step 10: 変換品質改善（GFM Alerts, タスクリスト, 追加マクロ対応, 未対応要素レポート）
- [x] Step 11: HTML出力追加と中間ファイル名称変更
  - config.go: XHTMLDir→IntermediateDir、HTMLDir 追加
  - IntermediateSaver（旧 XHTMLSaver）へのリネーム
  - Converter.ToHTML() 追加
  - HTMLWriter 新規実装
  - main.go への HTMLWriter ワイヤリング
- [x] Step 12: ADF移行（atlas_doc_format）
  - HTML出力廃止・Markdownのみに変更
  - PageBody に AtlasDocFormat フィールド追加・API URL を atlas_doc_format に変更
  - adfconverter.go 新規実装（ADF JSON → Markdown 直接変換）
  - Converter.ConvertADF() 追加
  - IntermediateSaver を ADF JSON 保存・読み込みに変更
  - MDWriter のページ本文変換を ConvertADF に切り替え
- [x] ADF テーブル変換の GFM 近似強化（セル内リスト/引用/結合セル/入れ子テーブル/alignment 対応）
- [x] ADF強調マーク（strong/em/strike）の前後空白によるMarkdown崩れ修正
  - wrapDelimiter ヘルパー追加（前後空白をデリミタ外に退避）
  - NOTEパネル内などで前後空白付きテキストを太字/斜体/取り消し線にした際にリテラル `**` 表示される不具合を解消
- [x] 目次（toc）マクロのHugoショートコード変換対応（`{{< toc >}}` + toc.htmlショートコード追加）
  - main（PR #14マージ済み時点）へrebaseして統合。mainとの重複変更（mdwriter.goへのWriteFolder追加、コメント見出し変更）は行番号のみでコンフリクトなし
  - 実データ（`SCRUM/2026-5-13 テスト議事録`）で `make convert` 相当の再変換とHugo表示を確認。目次にセクション見出しのみ表示されコメント見出しは除外されることを確認
  - `hugo-site/layouts/shortcodes/toc.html` はサイト直下に配置（テーマ`hugo-site/themes/hugo-theme-docs`側ではない）。Hugoの仕様上サイト側がテーマより優先されるため動作は問題ないが、他のテーマ関連ファイルとの一貫性の観点でテーマ側への統一は将来的な改善候補として残す → テーマへ移動済み（ページプロパティレポート対応時）
- [x] コメントリプライ取得とstatusバッジ修正（PR #9に追加）
  - フッターコメントのリプライを `/footer-comments/{id}/children` から再帰取得し階層見出しで出力
  - リプライを深さに応じて `<div style="margin-left: {N}em">` でインデント表示
  - status マクロを Confluence 風 lozenge バッジ（インラインCSS span）で再現
- [x] ADF変換の未対応要素修正（設計: docs/superpowers/specs/2026-08-01-adf-conversion-fixes-design.md）
  - [x] A: リスト内ブロック要素の欠落修正（listItem内codeBlock、入れ子taskList）
  - [x] B: リスト入れ子インデントのマーカー幅ベース化と隣接強調runの結合
  - [x] C: 文字色（textColor→span）・配置（alignment→div）のHTML再現
  - [x] D: コメント投稿者名の解決（GetUserDisplayNameのワイヤリング）
- [x] 左サイドバーの階層ツリー化（フォルダ対応）
  - Confluence REST API v2 の `GET /wiki/api/v2/folders/{id}` を叩く `GetFolder` を追加
  - ページの親を辿って必要なフォルダだけを再帰的に収集する `CollectFolders` を追加（foldertree.go）
  - ページのフロントマターに `parent_id` と `weight`（並び順）を追加
  - フォルダは `is_folder = true` / `[build] render = "never"` の「レンダリングされないページ」スタブとして出力（URL・HTMLは生成されないがサイドバーのツリーには現れる）
  - 中間ファイルのメタデータに `parent_type` / `position` を追加し、convert コマンド（オフライン再変換）でもページの親子関係（`parent_id`）と並び順（`weight`）を維持。ただしフォルダスタブは中間ファイルに保存されないため convert では出力されず、フォルダ配下のページはルート直下に並ぶ（フォルダスタブの出力は space コマンドのみ）
  - 中間ファイルから読み戻す際に親ID（ParentID）が復元されていなかった不具合を修正
  - テーマ側（hugo-theme-docs submodule）で `<details>`/`<summary>` による開閉式階層ツリー表示を実装（JavaScript不要、現在ページの祖先フォルダは自動展開・ハイライト）
  - サイドバーの子検索を索引化してビルド時間を短縮（400ページの合成コンテンツで 19.9秒 → 0.87秒。出力HTMLは同一）
  - 利用者向け注意: 既存の出力には階層情報が含まれないため反映には移行の再実行が必要。空フォルダ（ページを含まない）はサイドバーに表示されない。親が取得できないページはルート直下に表示され移行ログに警告が出る。`weight` を持たないページ（手書き追加ページ等）は Hugo の仕様上サイドバー先頭に並ぶ
- [x] 左サイドバーのリサイズ・開閉（設計: docs/superpowers/specs/2026-08-11-sidebar-resize-collapse-design.md）
  - 左サイドバーを境界のドラッグで左右にリサイズできるようにした（最小160px・最大480px・既定240px）
  - ヘッダーのボタン（☰）でサイドバーを開閉できるようにした。閉じると本文が左いっぱいまで寄る（本文の最大幅は従来どおり `max-width: 900px` のまま）
  - 閉じている間はサイドバーの中身をタブ順・支援技術のツリーからも外す（`visibility: hidden`）
  - 幅と開閉状態をlocalStorageに保存し、ページを移動しても次回訪問時も維持されるようにした
  - 掴み手はキーボードでも操作できる（Tabで到達し、左右キーで16pxずつ、Homeキーで既定値に戻る）。掴み手のダブルクリックでも既定値に戻る
  - 支援技術向けに、掴み手に `role="separator"` と `aria-valuenow` / `aria-valuemin` / `aria-valuemax`、トグルボタンに `aria-expanded` / `aria-controls` を持たせた
  - 幅はCSSカスタムプロパティ `--sidebar-width` で持ち、`--sidebar-current` を経由してCSS Gridの列幅に反映する二段構え。閉じた状態は `--sidebar-current: 0px` で上書きする
  - `<head>` 内のインラインスクリプトが、保存済みの状態を初回描画の前に反映することで、読み込み時に既定幅で一瞬描画される「ちらつき」を防いでいる
  - ドラッグはPointer Events（`setPointerCapture`）で実装。保存はドラッグ終了時のみ行う
  - 幅のクランプ（160〜480）はJavaScript側で行う。CSSの`clamp()`はWebKitで範囲外の値がそのまま通り機能しないため
  - テーマが自前のJavaScriptを持つのは今回が初めて。既存のCSSと同じく`resources.Get`→`minify`→`fingerprint`で扱う
  - 利用者向け注意: 対象はPCの画面幅のみでモバイル向けのオーバーレイ表示は含まない。JavaScriptが無効な環境ではサイドバーは既定の240pxで開いたまま表示され、操作用のボタンと掴み手は表示されない（動かない操作要素を見せないため）。localStorageが使えない環境（プライベートモードなど）ではそのページ内では操作できるが状態は保存されない
- [x] 左サイドバーのツリー開閉マーカー（▶）の表示修正
  - `.tree-item { display: block }` が `summary { display: flex }` より詳細度が高く勝っていたため、▶が項目名と別行になり1行を無駄にしていた。さらに inline 要素には transform が効かないため、開いても▶のまま（▼に回転しない）で開閉状態が分からなかった
  - `display: block` を葉（`.is-leaf`）だけに限定して summary の flex を有効にした。▶と項目名が同じ行に並び、開くと▼になる
  - Playwright（Firefox）で閉じた状態・開いた状態の両方を確認
- [x] 左サイドバーのツリー開閉状態をページ移動後も保持する
  - 従来はページごとにHugoが「現在ページの祖先だけ開いた状態」で描画し直すため、手で開いたノードが移動のたびに閉じていた
  - 各 `<details>` に `data-page-id` を付与し、開いているノードのID一覧をlocalStorage（`sidebar-tree-open`）に保存。読み込み時点で開いているノード（現在ページの祖先）も記録し、以降は `toggle` イベントで追加・削除する
  - 読み込み時はHugoの描画状態（現在ページの祖先は開く）に、保存済みの「開く」だけを上乗せする。閉じる方向の上書きはしないので移動先のページは必ずツリー上で見える。復元はツリー直後のインラインスクリプトで行い、ちらつきを防ぐ（幅の復元と同じ考え方）
  - ブラウザは `open` 付きで描画された details に読み込み時 toggle イベントを飛ばすが、リスナーより先に飛ぶことがあるためイベントには頼らず明示的に記録する
  - Playwright（Firefox）で「開いて移動→開いたまま」「閉じて移動→閉じたまま」「閉じたフォルダ配下のページへ移動→自動で開く」を確認
  - 利用者向け注意: JavaScript無効時は従来どおり（現在ページの祖先のみ開く）。localStorageが使えない環境ではページ内の操作のみ有効
- [x] Confluenceの「レイアウト」（複数カラム表示）の再現
  - 現状調査: ADF変換では `layoutSection`/`layoutColumn` の子要素を単純連結しており、カラム幅・横並びの情報が失われていた（実データで確認: 3カラム均等幅 `width: 33.33` × 3 が単なる縦並びテキストになっていた）
  - 再現方式はCSSクラス＋テーマ側スタイル方式を採用（Goコードは構造とカラム幅（CSSカスタムプロパティ `--col-width`）だけを出力し、横並び・レスポンシブ折り返しはテーマのCSSで一元管理。既存のテーブル・ツリー表示と同じ設計パターン）
  - `adfconverter.go`: `layoutSection` → `<div class="layout-section">`、`layoutColumn` → `<div class="layout-column" style="--col-width: N%">` に変換（TDDで実装、width未指定時はstyle属性なし）
  - テーマ側（hugo-theme-docs submodule）の `assets/css/main.css` に `.layout-section`（flex, flex-wrap: wrap）/`.layout-column`（flex-basisに`--col-width`、min-width: 200pxで折り返し）を追加
  - 利用者向け注意: `%`指定のflex-basisはgapを考慮しないため、gap 1個分を差し引いて近似している（厳密な計算ではないが実用上ははみ出さない）
  - 実データ（`SCRUM/2026-5-13 テスト議事録`、3カラムレイアウトを含む）で `convert` 再変換とHugo表示（PC幅・モバイル幅420px）を確認。PC幅では3カラム横並び、420px幅では自動的に縦積みに折り返されることを確認
- [x] ページプロパティレポート（detailssummary）の再現
  - [x] 現状調査: `SCRUM/2026-8-8` の detailssummary が `<!-- macro: detailssummary -->` になり何も表示されない。子ページ メモ / メモ2 に details マクロとラベル memo がある
  - [x] 設計（CQLはGoで解析しショートコード引数に変換、集計はHugo側）: `docs/superpowers/specs/2026-10-03-page-properties-report-design.md`
  - [x] 仕様書のユーザー確認
  - [x] 実装計画の作成: `docs/superpowers/plans/2026-10-03-page-properties-report.md`
  - [x] 実装（Go: details抽出・CQL解析・ショートコード出力 / テーマ: page-properties-report ショートコード、toc のテーマ移動）
  - [x] 実データ・一時サイトでの動作確認
    - `SCRUM` を再変換（警告なし）。`2026-8-8` に `{{< page-properties-report labels="memo" labels_mode="all" space="current" scope="children" root="current" >}}`、`メモ` の front matter に `[[properties]]` 3件
    - `hugo` ビルド成功。ブラウザで `/scrum/2026-8-8/` に メモ（2026-10-03・release・あるか）と メモ2（2026-10-02・作業中・ないよ）の2行がタイトル順で表示、タイトルリンクで各ページへ移動可
    - `/scrum/2026-5-13-テスト議事録/` の目次がテーマ側 toc で従来どおり表示
  - [x] 最終レビューの指摘を修正: ショートコード引数の `\` 末尾・改行でHugoビルドが失敗する問題、root が空のときの全件一致、全条件除外時の対象範囲、単一句の括弧、変換エラーのログ、README の Hugo 要件（0.146.0 以上）
  - [x] PR作成（テーマ gozuk16/hugo-theme-docs#7 と親リポジトリ。親側で submodule ポインタも ed8048c に更新）
  - 既知の制限（CHANGELOG に記載）: created 条件は front matter の date が最終更新日時のため lastmodified と同じ判定になる。日付比較は UTC の日単位。`- ` や `1. ` で始まるプロパティ値はレポートでリスト表示されることがある
  - 後続候補: v2 API の Page.createdAt を取得して date に出す / front matter 出力を TOML 用のエスケープにする（現状は Go の %q）
- [x] 子ページ一覧（children マクロ、Confluence の「子アイテム」）の再現
  - [x] 現状調査: `SCRUM/2026-8-8` の children マクロ（allChildren=true, depth=2）が `<!-- macro: children -->` になり何も表示されない。Confluence では「メモ ＞ 2026（フォルダ）、メモ2」と表示される
  - [x] 設計（ユーザー承認済み）: 変換器は `{{< children ... >}}` を出力し、テーマのショートコードが parent_id をたどって一覧にする。主要オプション（depth・allChildren・sortAndReverse・first・page・style）に対応、抜粋（excerpt）は対象外
  - [x] 実装（Go: `childrenmacro.go`、テーマ: `layouts/_shortcodes/children.html` / `layouts/_partials/children-list.html`）
  - [x] 動作確認: 一時サイトで深さ・並び順・件数・見出し・起点ページ・フォルダ・エスケープを確認。実データで Confluence と同じ「メモ ＞ 2026（フォルダ）、メモ2」の表示を確認
  - [x] コードレビュー（サブエージェント）: 「API: 概要」のようなタイトルをスペースキー付きと誤判定する問題、先頭ゼロの数値、索引の毎回構築を修正。フォルダが日付順で最も古い扱いになる点は既知の制限として記載
  - [x] PR作成（テーマ gozuk16/hugo-theme-docs#8 と親リポジトリ。親側で submodule ポインタも 12b7656 に更新）
