# 設計ドキュメント: HTML出力追加と中間ファイル名称変更

**日付:** 2026-05-13  
**ステータス:** 承認済み

---

## 背景・動機

現在のツールは Confluence Storage Format（XHTML）を中間ファイルとして保存し、Markdown のみを出力している。以下の課題がある。

- 中間ファイルが `.xhtml` という名称だが、その役割（中間ファイル）が名前から読み取りにくい
- 人間が読める HTML 出力がない（Markdown は AI 向けで、ブラウザで確認しにくい）

---

## 要件

1. **生データ保存**: Confluence Storage Format はそのまま保存する（Converter 修正時に再取得不要）
2. **HTML 出力（新規）**: ブラウザで読める HTML5 を生成する。Confluence の見た目にできるだけ近い形
3. **Markdown 出力（既存）**: AI 向け。マクロの完全再現は不要
4. **再変換可能**: `convert` コマンドで中間ファイルから HTML・Markdown 両方を再生成できる（API 不要）

---

## アーキテクチャ

```
Confluence API
     ↓
[中間ファイル: content.xhtml]  ← Confluence Storage Format そのまま保存
     ↓ 再処理可能（API なし）
┌────┴─────┐
↓          ↓
index.html  index.md
（人間向け）  （AI 向け）
```

### ディレクトリ構成

```
output/
  intermediate/           ← 中間ファイル（旧 xhtml/）
    {SPACE}/
      {PAGE}/
        content.xhtml     ← Confluence Storage Format（内容は変更なし）
        metadata.toml
        comments/
          comment_001.xhtml
          comment_001.toml
  html/                   ← 新規: HTML Living Standard 出力
    {SPACE}/
      {PAGE}/
        index.html
  markdown/               ← 既存（変更なし）
    {SPACE}/
      {PAGE}/
        index.md
```

---

## 変更一覧

### 1. 名称変更（中間ファイル関連）

| 変更前 | 変更後 | 種別 |
|---|---|---|
| `XHTMLSaver` | `IntermediateSaver` | Go 構造体名 |
| `NewXHTMLSaver()` | `NewIntermediateSaver()` | Go 関数名 |
| `xhtmlSaver` | `intermediateSaver` | Go 変数名 |
| `saveXHTML` | `saveIntermediate` | Go 変数名 |
| `xhtmlsaver.go` | `intermediatesaver.go` | ファイル名 |
| `xhtmlsaver_test.go` | `intermediatesaver_test.go` | ファイル名 |
| `XHTMLDir` | `IntermediateDir` | Go フィールド名 |
| `xhtml_dir` | `intermediate_dir` | config.toml キー |
| `output/xhtml` | `output/intermediate` | デフォルトパス |
| `--save-xhtml` | `--save-intermediate` | CLI フラグ |
| `convertFromXHTML()` | `convertFromIntermediate()` | Go 関数名 |
| `convert` コマンド説明文 | 「保存済み中間ファイルから変換」に更新 | CLI help |

**ファイル内容（`content.xhtml`、`comment_001.xhtml`）は変更なし。**

### 2. Converter に `ToHTML()` メソッドを追加

```go
// ToHTML は Confluence Storage Format を標準 HTML ボディ文字列に変換する
// （DOCTYPE・html/head/body タグは含まない。HTMLWriter が組み立てる）
func (c *Converter) ToHTML(xhtml string) (string, error)
```

- 既存の `preprocess()` を内部で呼び出す（`preprocess()` 自体は private のまま）
- 返り値はボディ部分のみ（`<p>...</p>` 等の HTML フラグメント）
- HTML5 ドキュメント全体の組み立ては `HTMLWriter` が担う
- `preprocess()` は引き続き `Convert()` からも呼ばれる（変更なし）

### 3. HTMLWriter の新規追加（`htmlwriter.go`）

```go
type HTMLWriter struct {
    outputDir string
    conv      *Converter
}

func NewHTMLWriter(outputDir string, conv *Converter) *HTMLWriter

func (w *HTMLWriter) WritePage(
    page *Page,
    spaceKey, spaceName, parentTitle string,
    labels []Label,
    comments []Comment,
    attachments []Attachment,
) error
```

出力する HTML5 の構造:

```html
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="UTF-8">
  <title>{PAGE_TITLE}</title>
  <style>/* 最低限のスタイル（コードブロック・テーブル・パネル等） */</style>
</head>
<body>
  <header>
    <h1>{PAGE_TITLE}</h1>
    <p>スペース: {SPACE} | 更新日時: {UPDATED_AT} | ラベル: {LABELS}</p>
    <p><a href="{CONFLUENCE_URL}">Confluenceで開く</a></p>
  </header>
  <main>
    {変換済み HTML 本文}
  </main>
  <section class="comments"><!-- コメントがあれば --></section>
  <footer>
    <p>元ページ: <a href="{CONFLUENCE_URL}">{CONFLUENCE_URL}</a></p>
  </footer>
</body>
</html>
```

出力パス: `{html_dir}/{spaceKey}/{sanitizedTitle}/index.html`

### 4. config.go の変更

`OutputConfig` に `HTMLDir` フィールドを追加:

```go
type OutputConfig struct {
    MarkdownDir    string `toml:"markdown_dir"`
    AttachmentsDir string `toml:"attachments_dir"`
    IntermediateDir string `toml:"intermediate_dir"` // 旧 XHTMLDir
    HTMLDir        string `toml:"html_dir"`           // 新規
}
```

デフォルト値:
- `IntermediateDir`: `"output/intermediate"`
- `HTMLDir`: `"output/html"`

`config.toml.example` も更新する。

### 5. main.go の変更

- `page` / `space` コマンド: `HTMLWriter` を生成し、Markdown と HTML を両方出力
- `convert` コマンド: 中間ファイルから HTML・Markdown 両方を再生成
- 全フラグ・変数名を上記名称変更に従い更新

---

## エラー処理方針

- HTML 生成に失敗しても Markdown 生成は継続する（`slog.Warn` でログを出す）
- 中間ファイル保存に失敗しても変換処理は継続する（既存の方針を維持）

---

## テスト方針

- `TestIntermediateSaver_*`: ファイル名変更に伴うリネーム
- `TestConverter_ToHTML`: 新規テスト（HTML5 ドキュメントが正しく生成されるか）
- `TestHTMLWriter_WritePage`: 新規テスト（ファイルが正しいパスに生成されるか）

---

## 対象外

- 既存の `Convert()` メソッドのシグネチャ変更（変更なし）
- Markdown 出力フォーマットの変更（変更なし）
- 中間ファイルの内容（`content.xhtml` の中身）の変更（変更なし）
- HTML の CSS デザインの凝った実装（最低限のスタイルのみ）
