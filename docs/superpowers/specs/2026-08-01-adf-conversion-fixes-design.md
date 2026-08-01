# ADF変換の未対応要素修正 設計ドキュメント

作成日: 2026-08-01

## 背景

Hugoサイト（`http://localhost:1313/scrum/2026-5-13-テスト議事録/`）と移行元のConfluence Cloudページ（page_id: 27656193）を比較した結果、ADF → Markdown変換で以下の未対応・崩れが確認された。

### 抽出した問題一覧

| # | 分類 | 問題 | 原因箇所 |
|---|------|------|----------|
| 1 | A: コンテンツ欠落 | リスト項目内のコードブロックが出力されない（2箇所） | `renderListItem` が `codeBlock` を処理しない |
| 2 | A: コンテンツ欠落 | 入れ子のタスクリスト（チェックボックス）が出力されない | `renderTaskList` が子の `taskList` をスキップ |
| 3 | B: レンダリング崩れ | 番号付きリストの入れ子が2スペースインデントで分断される | インデントが固定2スペースでマーカー幅（`1. `=3）未満 |
| 4 | B: レンダリング崩れ | 連続する斜体runが `*あけ**ぼ**の*` となり太字に化ける | テキストノードごとにデリミタを付与し隣接runを結合しない |
| 5 | C: 装飾喪失 | 文字色（textColor）が失われる | `renderText` が textColor を無視 |
| 6 | C: 装飾喪失 | 中央寄せ・右寄せ（alignment）が失われる | 段落の alignment マークを無視 |
| 7 | D: 情報喪失 | コメント投稿者が accountId のまま表示される | `GetUserDisplayName` が実装済みだが未使用 |

## スコープ

- A〜D すべてを修正対象とする（ユーザー確認済み）。
- 修正順序は A → B → C → D。1項目ずつTDDで修正しコミットする。
- ブランチ: `feature/adf-conversion-fixes`（`feature/adf-emphasis-whitespace` から分岐。強調マーク処理 `wrapDelimiter` に依存するため）。

## 設計

### A. リスト内ブロック要素の欠落修正

**対象:** `adfconverter.go` の `renderListItem`、`renderTaskList`

- `renderListItem` に `codeBlock` ケースを追加する。`renderCodeBlock` の出力（フェンス ``` を含む）を継続行としてマーカー幅分インデントする。
- listItem の先頭子要素がブロック要素の場合（実データに存在）は、マーカー行にブロックの1行目（フェンス開始）を置き、以降の行をインデントする。
- `renderTaskList` を再帰対応にし、子ノードが `taskList` の場合はインデントを2スペース深くして再帰出力する。`- [ ] ` マーカーの入れ子はGFM/Goldmarkとも2スペースで認識される。

### B. インデント方式の是正と強調runの結合

**対象:** `adfconverter.go` のリスト系レンダラ、`renderInlineNodes`

- リスト系レンダラ（`renderBulletList` / `renderOrderedList` / `renderListItem`）のインデント管理を「レベル数 × 2スペース」から「累積インデント文字列」に変更する。子要素のインデント量は実際のマーカー幅から算出する（`- `=2、`1. `=3、`10. `=4）。継続行・入れ子リスト・A項のブロック要素すべてに適用する。
- `renderInlineNodes` で、隣接するテキストノードのうち「Markdownデリミタ系マーク（strong / em / strike）の組が同一」のものをグループ化し、グループ全体を1組のデリミタで囲む。
  - textColor 等の非デリミタマークはグループ内の各ノードに個別適用する。
  - `code` / `link` / `subsup` / `underline` を含むノードはグループ境界として扱う（グループ化しない）。
  - 例: 「春はあけぼの」→ `**春**<span style="color: #ff5630">は</span>*あけ<span style="color: #ffc400">ぼ</span>の*`

### C. 文字色・配置のHTML再現

**対象:** `adfconverter.go` の `renderText`、`renderNode`（paragraph）

- `renderText`: `textColor` マークを `<span style="color: #RRGGBB">…</span>` に変換する。デリミタ系マークより内側に配置する。
- 段落の `alignment` マーク（`align: center` / `end`）を検出し、`<div style="text-align: center|right">` + 空行 + 本文 + 空行 + `</div>` で包む（既存の `<details>` 出力と同じGoldmark互換パターン）。`end` は `right` に写像する。
- テーブルセル内は既存のHTML化処理（`cellAlignment` 等）があるため対象外とする。
- 前提: Hugo側でraw HTMLレンダリングが有効であること（既存出力の `<u>` / `<details>` が正常レンダリングされており確認済み）。

### D. コメント投稿者名の解決

**対象:** `mdwriter.go`、`main.go`

- `MDWriter` に `resolveUser func(accountID string) string` フィールドを追加し、コンストラクタで注入する。
- `main.go` で `client.GetUserDisplayName(id, cfg.DeletedUsers)` を包むクロージャを渡す。`GetUserDisplayName` は実装済み（キャッシュ・deletedUsersマッピング・APIフォールバック対応、confluenceclient.go:443）。
- コメント出力（mdwriter.go:94）で accountId の代わりに解決した表示名を出力する。resolver が nil の場合は従来どおり accountId を出力する（テスト互換）。

## エラーハンドリング

- ユーザー解決APIが失敗した場合、`GetUserDisplayName` は accountId をそのまま返す（既存挙動）。変換処理は継続する。
- 未知のブロック要素がリスト項目内に現れた場合は従来どおりスキップするが、既存の未対応要素レポート機構の対象とする。

## テスト・検証

1. 各項目ごとに `adfconverter_test.go` / `mdwriter_test.go` にユニットテストを追加する（TDD: 失敗するテストを先に書く）。
   - A: listItem内codeBlock（先頭・非先頭）、入れ子taskList
   - B: 番号付きリスト下の入れ子リスト・codeBlockのインデント幅、隣接em run結合（textColor混在含む）
   - C: textColorのspan変換、alignment center/endのdiv変換
   - D: resolver注入時の表示名出力、nil時のaccountId出力
2. `make test` で全テストがパスすること。
3. 実ページ（page_id: 27656193）で再変換し、Hugoレンダリング結果のHTMLで7項目すべての解消を確認する。

## 修正しないこと

- ページプロパティ（縦型ヘッダテーブル）のGFM 1行目ヘッダへの近似は現状維持（妥当と判断）。
- backgroundColor マークは今回のページに現れないため対象外。
- テーブルセル内の変換ロジックは変更しない。
