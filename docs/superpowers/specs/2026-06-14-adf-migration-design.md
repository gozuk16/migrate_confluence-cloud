# 設計: Atlas Doc Format (ADF) への完全移行

**日付:** 2026-06-14
**ブランチ:** feature/html-output → 新ブランチで実装予定
**ステータス:** 承認済み（Opusレビュー反映済み）

## 背景と目的

現在 Confluence API から `body-format=storage` (Confluence 独自の XHTML) でページコンテンツを取得し、`converter.go` の `preprocess()` で `ac:`/`ri:` タグを手動パースして標準 HTML に変換した後、`html-to-markdown` ライブラリで Markdown に変換している。

この方式の課題:
- `ac:structured-macro` のパース漏れや変換精度が低い（特に情報パネル）
- 新しいマクロへの対応コストが高い
- 変換パイプラインが多段でデバッグしにくい

**移行目的:** `body-format=atlas_doc_format` (ADF) に切り替え、型付き JSON 構造から Markdown を直接生成することでマクロ変換の精度と保守性を向上させる。

**出力形式の変更:** HTML 出力（`htmlwriter.go`）は廃止し、Markdown 出力のみとする。

## アーキテクチャ

### 現在のパイプライン

```
Confluence API (storage: XHTML)
  → converter.preprocess(): ac:/ri: タグ → 標準 HTML
  → html-to-markdown: HTML → Markdown
                                          → (HTML出力: htmlwriter.go)
```

### 移行後のパイプライン

```
Confluence API (atlas_doc_format: JSON)
  → adfconverter.go: ADF JSON → Markdown 直接変換
```

中間 HTML 層・HTML 出力を廃止し、ADF の型付きノードツリーから Markdown を直接生成する。

## 変更ファイル一覧

| ファイル | 変更内容 |
|---|---|
| `confluenceclient.go` | ページ・コメント取得を `body-format=atlas_doc_format` に変更。`PageBody` に `AtlasDocFormat` フィールド追加（`Storage` は `CommentBody` と共用のため削除しない）|
| `converter.go` | `preprocess()` / html-to-markdown パイプラインを削除。`Convert()` が `adfconverter.go` を呼ぶ薄いラッパーになる |
| `intermediatesaver.go` | 中間ファイルとして ADF JSON を保存・読み込みするよう変更 |
| `config.go` | `HTMLDir` 設定を削除 |
| `main.go` | `HTMLWriter` のワイヤリングをすべて削除 |
| `htmlwriter.go` | **削除**（HTML 出力機能を廃止） |
| `htmlwriter_test.go` | **削除** |
| `converter_test.go` | ADF ベースに書き換え |
| `intermediatesaver_test.go` | ADF JSON 保存・読み込みに対応した内容に書き換え |
| `go.mod` / `go.sum` | `html-to-markdown` ライブラリを削除（`go mod tidy` で整理） |

## 新規追加ファイル

| ファイル | 内容 |
|---|---|
| `adfconverter.go` | ADF JSON → Markdown 直接変換器 |
| `adfconverter_test.go` | 各ノードタイプの単体テスト |

## データ構造

### ADF ノード

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

### PageBody 構造体の変更方針

`Storage` 型は `CommentBody` でも共用しているため改変しない。`PageBody` に `AtlasDocFormat` フィールドを追加する。

```go
// PageBody はページのボディコンテンツ
type PageBody struct {
    Storage        Storage        `json:"storage"`         // 既存（CommentBody と共用のため残す）
    AtlasDocFormat AtlasDocFormat `json:"atlas_doc_format"` // 新規追加
}

// AtlasDocFormat は ADF 形式のコンテンツ
type AtlasDocFormat struct {
    Value          string `json:"value"`          // ADF JSON 文字列（二重デコードが必要）
    Representation string `json:"representation"`
}
```

ADF の `value` フィールドは **JSON 文字列として返される**ため、`json.Unmarshal` で `AtlasDocFormat.Value` を得た後、さらに `json.Unmarshal([]byte(value), &ADFNode{})` でパースする（二重デコード）。

コメント（`CommentBody`）は `Storage` のまま変えない。**コメント変換は `converter.go` の storage → Markdown パイプラインを残して対応する**（ページ本文のみ ADF 化）。

## ADF → Markdown 変換マッピング

### ブロック要素

| ADF `type` | attrs | Markdown 出力 |
|---|---|---|
| `heading` | `level: 1-6` | `# ` ～ `###### ` |
| `paragraph` | - | テキスト + 空行 |
| `bulletList` / `listItem` | - | `- `（ネスト時は 2 スペースインデント）|
| `orderedList` / `listItem` | - | `1. `（ネスト時は 2 スペースインデント）|
| `blockquote` | - | `> ` |
| `codeBlock` | `language: X` | ` ```X\n...\n``` ` |
| `rule` | - | `---` |
| `panel` | `panelType: info` | `> [!NOTE]` |
| `panel` | `panelType: note` | `> [!WARNING]` |
| `panel` | `panelType: warning` | `> [!CAUTION]` |
| `panel` | `panelType: success` | `> [!TIP]` |
| `panel` | `panelType: error` | `> [!CAUTION]`（エラーは CAUTION 扱い）|
| `expand` / `nestedExpand` | `title: T` | `<details><summary>T</summary>...</details>` |
| `table` | - | GFM テーブル `\| --- \|` |
| `tableRow` | - | テーブル行 |
| `tableHeader` | - | ヘッダセル（1 行目扱い）|
| `tableCell` | - | データセル（colspan/rowspan は無視）|
| `taskList` / `taskItem` | `state: TODO/DONE` | `- [ ] ` / `- [x] ` |
| `decisionList` / `decisionItem` | - | `- `（通常リストとして出力）|
| `status` | `color`, `text` | `🟢[text]` 等（色→絵文字）|
| `mediaSingle` + `media` (external) | `url`, `alt` | `![alt](url)` |
| `mediaSingle` + `media` (file) | `id` (UUID) | `![alt](filename)` ※後述 |
| `mediaGroup` + `media` | 複数メディア | 各 `media` を個別に `![alt](url)` 出力 |
| `mediaInline` | `url` / `id` | `![](url)` or `![](filename)` |
| `layoutSection` / `layoutColumn` | - | 子要素を順次展開（段組み解除）|
| `mention` | `text` | `**@text**` |
| `emoji` | `shortName` | Unicode 絵文字 or `:name:` |
| `date` | `timestamp` (ms epoch 文字列) | `time.UnixMilli()` で変換した `YYYY-MM-DD` |
| `hardBreak` | - | `\n` |
| `extension` / `inlineExtension` | `extensionKey` | `<!-- macro: key -->` |
| `bodiedExtension` | `extensionKey` | content を展開（content が空の場合のみ `<!-- macro: key -->`）|
| `inlineCard` / `blockCard` | `url` | `[url](url)`（内部 URL は相対パスに変換）|
| `embedCard` | `url` | `<!-- embed: url -->` |

### インライン marks

| mark `type` | Markdown |
|---|---|
| `strong` | `**text**` |
| `em` | `*text*` |
| `code` | `` `text` `` |
| `strike` | `~~text~~` |
| `link` | `[text](href)`（内部 URL は相対パスに変換）|
| `underline` | `<u>text</u>`（HTML fallback）|
| `subsup` | `attrs.type: "sub"` → `<sub>`, `"sup"` → `<sup>` |
| `textColor` | テキストのみ出力（色情報は破棄）|
| `backgroundColor` | テキストのみ出力（色情報は破棄）|
| `annotation` | テキストのみ出力 |

### status の color → 絵文字マッピング

| color | 絵文字 |
|---|---|
| green | 🟢 |
| yellow | 🟡 |
| red | 🔴 |
| blue | 🔵 |
| purple | 🟣 |
| neutral / grey / gray | ⚫ |

## 既知の課題と対処方針

### 1. 添付ファイルの ID 解決

ADF の `media` ノード（`attrs.type: "file"`）はファイルを UUID (`attrs.id`) で参照する。`GetPageAttachments()` が返す `Attachment.ID` は数値形式（`att...`）で UUID とは異なる可能性がある。

**対処:**
1. 実際の API レスポンスで `media.attrs.id`（UUID）と `Attachment.ID` が対応するか確認する。対応する場合は `id → filename` マップを事前構築して参照。
2. 対応しない場合は、`Attachment.Links.Download` URL を直接使用するか、ファイル名一致でのフォールバック検索を試みる。
3. 解決できない場合は `attachment-{id}` をファイル名として使用し、未解決ファイルをログ出力する。

### 2. 内部ページリンクの URL 変換

ADF の `inlineCard` / `link` mark の内部リンクは絶対 Confluence URL を持つ。現行の `ri:content-title` 直取得より精度が落ちる可能性がある。

**対処:** URL パターン `/wiki/spaces/[^/]+/pages/\d+/([^#?]+)` からページタイトルを抽出。URL デコード（`url.PathUnescape`）を適用後、`sanitizeFilename(title) + "/index.md"` の相対パスに変換する。アンカー（`#...`）は末尾に付与。タイトル抽出に失敗した場合は元の URL をそのまま使用する。

### 3. ネストリスト

ADF のリストは `listItem` の `content` に `bulletList`/`orderedList` がネストする。インデントレベルを引数として渡し、2 スペース × レベルでネストを表現する。

### 4. テーブルのスパン

`colspan`/`rowspan` attrs は Markdown テーブルで表現不能。スパン情報は無視しセル内容のみ出力する。

### 5. `--skip-download` での ADF JSON 再利用

中間ファイルとして ADF JSON を `intermediate/<pageID>.json` に保存する。`--skip-download` 使用時は JSON ファイルを読み込み、`json.Unmarshal` で `ADFNode` にデシリアライズして変換器に渡す。保存フォーマットは ADF の `value` フィールドの文字列（= 生の ADF JSON）をそのままファイルに書き出す。

### 6. 変換失敗時のフォールバック

`mdwriter.go` の変換失敗時フォールバック（現行: 元 XHTML をコードブロック出力）を維持する。ADF 版では生 ADF JSON をコードブロックとして出力する。

## コメント変換の方針

フッターコメント（`GetPageFooterComments()`）は引き続き `body-format=storage` で取得し、`converter.go` の既存 storage → Markdown パイプラインで処理する。

`converter.go` には以下の 2 メソッドを維持:
- `Convert(xhtml string) (string, error)` — storage XHTML → Markdown（コメント用）
- `ConvertADF(adfJSON string) (string, error)` — ADF JSON → Markdown（ページ本文用）

`preprocess()` と `html-to-markdown` 依存はコメント変換のために残す。

## テスト方針

- `adfconverter_test.go` を新規作成。各ノードタイプの単体テスト（入力 ADF JSON → 期待 Markdown）
- パネル・コードブロック・テーブル・タスクリスト・ネストリスト・メディアを重点テスト
- `converter_test.go`: ADF 変換テストを追加、storage 変換テストはコメント用として残す
- `intermediatesaver_test.go`: ADF JSON 保存・読み込みに対応した内容に書き換え

## 削除される依存ライブラリ

- `github.com/JohannesKaufmann/html-to-markdown/v2` — コメント変換がなくなった段階で削除

> **注意:** コメント変換が storage のままである間は `html-to-markdown` および `golang.org/x/net/html` を残す。コメントの ADF 移行は別 Issue として扱う。

## 非対応のまま維持するもの

ADF でも `extension` ノードになるマクロ（動的・クエリ系）は引き続きコメントアウト:
- `jira`（JQL クエリ形式）
- `children`, `pagetree`
- `contentbylabel`, `recently-updated`
- `gallery`, `multimedia`
- その他カスタムマクロ
