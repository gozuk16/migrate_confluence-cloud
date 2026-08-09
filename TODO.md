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
