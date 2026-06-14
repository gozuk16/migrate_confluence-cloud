# 設計: Atlas Doc Format (ADF) への完全移行

**日付:** 2026-06-14
**ブランチ:** feature/html-output → 新ブランチで実装予定
**ステータス:** 承認済み

## 背景と目的

現在 Confluence API から `body-format=storage` (Confluence 独自の XHTML) でページコンテンツを取得し、`converter.go` の `preprocess()` で `ac:`/`ri:` タグを手動パースして標準 HTML に変換した後、`html-to-markdown` ライブラリで Markdown に変換している。

この方式の課題:
- `ac:structured-macro` のパース漏れや変換精度が低い（特に情報パネル）
- 新しいマクロへの対応コストが高い
- 変換パイプラインが多段でデバッグしにくい

**移行目的:** `body-format=atlas_doc_format` (ADF) に切り替え、型付き JSON 構造から Markdown を直接生成することでマクロ変換の精度と保守性を向上させる。

## アーキテクチャ

### 現在のパイプライン

```
Confluence API (storage: XHTML)
  → converter.preprocess(): ac:/ri: タグ → 標準 HTML
  → html-to-markdown: HTML → Markdown
```

### 移行後のパイプライン

```
Confluence API (atlas_doc_format: JSON)
  → adfconverter.go: ADF JSON → Markdown 直接変換
```

中間 HTML 層を廃止し、ADF の型付きノードツリーから Markdown を直接生成する。

## 変更ファイル一覧

| ファイル | 変更内容 |
|---|---|
| `confluenceclient.go` | `body-format=storage` → `body-format=atlas_doc_format` に変更。`PageBody` / `Storage` 構造体を ADF 用に更新 |
| `converter.go` | `preprocess()` / html-to-markdown パイプラインを削除。`Convert()` が `adfconverter.go` を呼ぶ薄いラッパーになる |
| `intermediatesaver.go` | 中間ファイルとして ADF JSON を保存するよう変更（デバッグ・再処理用） |
| `htmlwriter.go` | **削除**（中間 HTML がなくなるため不要） |
| `htmlwriter_test.go` | **削除** |
| `go.mod` / `go.sum` | `html-to-markdown` ライブラリを削除 |

## 新規追加ファイル

### `adfconverter.go`

ADF JSON → Markdown の直接変換器。

**データ構造:**

```go
type ADFNode struct {
    Type    string                 `json:"type"`
    Attrs   map[string]interface{} `json:"attrs,omitempty"`
    Content []ADFNode              `json:"content,omitempty"`
    Marks   []ADFMark              `json:"marks,omitempty"`
    Text    string                 `json:"text,omitempty"`
}

type ADFMark struct {
    Type  string                 `json:"type"`
    Attrs map[string]interface{} `json:"attrs,omitempty"`
}
```

## ADF → Markdown 変換マッピング

### ブロック要素

| ADF `type` | attrs | Markdown 出力 |
|---|---|---|
| `heading` | `level: 1-6` | `# ` ～ `###### ` |
| `paragraph` | - | テキスト + 空行 |
| `bulletList` / `listItem` | - | `- ` |
| `orderedList` / `listItem` | - | `1. ` |
| `blockquote` | - | `> ` |
| `codeBlock` | `language: X` | ` ```X\n...\n``` ` |
| `rule` | - | `---` |
| `panel` | `panelType: info` | `> [!NOTE]` |
| `panel` | `panelType: note` | `> [!WARNING]` |
| `panel` | `panelType: warning` | `> [!CAUTION]` |
| `panel` | `panelType: tip` / `success` | `> [!TIP]` |
| `expand` | `title: T` | `<details><summary>T</summary>...</details>` |
| `table` | - | GFM テーブル `\| --- \|` |
| `tableRow` | - | テーブル行 |
| `tableHeader` | - | ヘッダセル（1行目扱い） |
| `tableCell` | - | データセル |
| `taskList` / `taskItem` | `state: TODO/DONE` | `- [ ] ` / `- [x] ` |
| `status` | `color`, `text` | `🟢[text]` 等（色→絵文字） |
| `mediaSingle` + `media` (external) | `url` | `![alt](url)` |
| `mediaSingle` + `media` (file) | `id` (UUID) | `![alt](filename)` ※後述 |
| `mention` | `text` | `**@text**` |
| `emoji` | `shortName` | Unicode 絵文字 or `:name:` |
| `hardBreak` | - | `\n` |
| `decisionList` / `decisionItem` | - | `- ` (通常リストとして出力) |
| `extension` / `inlineExtension` | `extensionKey` | `<!-- macro: key -->` |
| `bodiedExtension` | `extensionKey` | content を展開（content が空の場合のみ `<!-- macro: key -->`）|
| `inlineCard` / `blockCard` | `url` | `[url](url)` |
| `date` | `timestamp` | ISO 日付文字列 |

### インライン marks

| mark `type` | Markdown |
|---|---|
| `strong` | `**text**` |
| `em` | `*text*` |
| `code` | `` `text` `` |
| `strike` | `~~text~~` |
| `link` | `[text](href)` |
| `underline` | `<u>text</u>` (HTML fallback) |
| `textColor` | テキストのみ出力（色情報は破棄） |
| `annotation` | テキストのみ出力 |

### status の color → 絵文字マッピング

| color | 絵文字 |
|---|---|
| green | 🟢 |
| yellow | 🟡 |
| red | 🔴 |
| blue | 🔵 |
| purple | 🟣 |
| neutral / grey | ⚫ |

## 既知の課題と対処方針

### 1. 添付ファイルの ID 解決

ADF の `media` ノード（`attrs.type: "file"`）はファイルを UUID (`attrs.id`) で参照する。Storage format の `ri:filename` と異なり、ファイル名が直接取れない。

**対処:** `GetPageAttachments()` で取得した添付ファイル一覧から `id → filename` マップを事前構築し、`media` ノード変換時に参照する。取得できない場合は `attachment-{id}` をファイル名として使用する。

### 2. 内部ページリンクの URL 変換

ADF の `inlineCard` / `link` mark は絶対 Confluence URL (`https://xxx.atlassian.net/wiki/spaces/KEY/pages/12345/Title`) を持つ。

**対処:** URL パターン (`/wiki/spaces/.*/pages/\d+/(.+)`) からページタイトルを抽出し、`sanitizeFilename(title) + "/index.md"` の相対パスに変換する。タイトル抽出に失敗した場合は元の URL をそのまま使用する。

### 3. ネストリスト

ADF のリストは `listItem` の `content` に `bulletList`/`orderedList` がネストする。インデントレベルを追跡してスペース（2スペース）でネストを表現する。

### 4. テーブルのスパン

ADF の `tableHeader`/`tableCell` には `colspan`/`rowspan` attrs があるが、Markdown テーブルはスパン非対応。スパン情報は無視し、セル内容のみ出力する。

## intermediatesaver.go の変更方針

現在は中間 HTML ファイルを `intermediate/` ディレクトリに保存している。ADF 移行後は **ADF JSON をそのまま保存**する。

- 拡張子: `.json`
- 用途: デバッグ・変換ロジック改善時の再処理
- `--skip-download` フラグで再利用可能（現行と同様）

## テスト方針

- `adfconverter_test.go` を新規作成
- 各ノードタイプごとの単体テスト（入力 ADF JSON → 期待 Markdown）
- パネル・コードブロック・テーブル・タスクリスト・ネストリストを重点テスト
- 既存の `converter_test.go` は削除または ADF ベースに書き換え

## 削除される依存ライブラリ

- `github.com/JohannesKaufmann/html-to-markdown/v2`
- `golang.org/x/net/html`（html-to-markdown 経由の依存）

## 非対応のまま維持するもの

ADF でも `extension` ノードになるマクロ（動的・クエリ系）は引き続きコメントアウト:
- `jira` (JQL クエリ形式)
- `children`, `pagetree`
- `contentbylabel`, `recently-updated`
- `gallery`, `multimedia`
- その他カスタムマクロ
