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
