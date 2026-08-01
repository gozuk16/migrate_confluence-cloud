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
- [x] ADF変換の未対応要素修正（設計: docs/superpowers/specs/2026-08-01-adf-conversion-fixes-design.md）
  - [x] A: リスト内ブロック要素の欠落修正（listItem内codeBlock、入れ子taskList）
  - [x] B: リスト入れ子インデントのマーカー幅ベース化と隣接強調runの結合
  - [x] C: 文字色（textColor→span）・配置（alignment→div）のHTML再現
  - [x] D: コメント投稿者名の解決（GetUserDisplayNameのワイヤリング）
