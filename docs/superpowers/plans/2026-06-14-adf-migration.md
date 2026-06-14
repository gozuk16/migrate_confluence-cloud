# ADF Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Confluence ページ取得フォーマットを `storage`（独自 XHTML）から `atlas_doc_format`（ADF JSON）に移行し、型付き JSON から Markdown を直接生成することでマクロ変換精度を向上させる。

**Architecture:** ADF JSON → `adfconverter.go` → Markdown 直接変換。HTML 中間層・HTML 出力機能・html-to-markdown ライブラリを廃止。コメント変換は既存の storage パイプラインを維持。

**Tech Stack:** Go 1.25、標準ライブラリのみ（`encoding/json`, `net/url`, `regexp`, `strings`）

---

## ファイル構成

| ファイル | 対応 |
|---|---|
| `htmlwriter.go` | 削除 |
| `htmlwriter_test.go` | 削除 |
| `config.go` | `HTMLDir` フィールド・デフォルト値・バリデーションを削除 |
| `config_test.go` | `HTMLDir` 参照を削除 |
| `main.go` | `htmlWriter` 変数・`NewHTMLWriter` 呼び出し・`WritePage` 呼び出しをすべて削除 |
| `confluenceclient.go` | `PageBody` に `AtlasDocFormat` フィールド追加、`GetPage`・`GetChildPages` の URL を `atlas_doc_format` に変更 |
| `adfconverter.go` | **新規作成** — ADF JSON → Markdown 変換器 |
| `adfconverter_test.go` | **新規作成** — 各ノードタイプの単体テスト |
| `converter.go` | `ConvertADF(adfJSON string, attachmentMap map[string]string) (string, error)` メソッドを追加 |
| `converter_test.go` | `ConvertADF` テストを追加 |
| `intermediatesaver.go` | `SavePage` を ADF JSON 保存に、`LoadPage` を ADF JSON 読み込みに変更 |
| `intermediatesaver_test.go` | ADF JSON 保存・読み込みテストに書き換え |
| `mdwriter.go` | `generateContent` でページ本文の変換を `ConvertADF` に切り替え |

---

## Task 1: HTML 出力機能の削除

**Files:**
- Delete: `htmlwriter.go`
- Delete: `htmlwriter_test.go`
- Modify: `config.go`
- Modify: `config_test.go`
- Modify: `main.go`

- [ ] **Step 1: htmlwriter.go と htmlwriter_test.go を削除**

```bash
rm /path/to/project/htmlwriter.go /path/to/project/htmlwriter_test.go
```

- [ ] **Step 2: config.go から HTMLDir を削除**

`config.go` の `OutputConfig` 構造体から `HTMLDir` フィールドを削除し、`Validate()` の `HTMLDir` デフォルト設定行も削除する。

変更前:
```go
type OutputConfig struct {
    MarkdownDir     string `toml:"markdown_dir"`
    AttachmentsDir  string `toml:"attachments_dir"`
    IntermediateDir string `toml:"intermediate_dir"`
    HTMLDir         string `toml:"html_dir"`
}
```

変更後:
```go
type OutputConfig struct {
    MarkdownDir     string `toml:"markdown_dir"`
    AttachmentsDir  string `toml:"attachments_dir"`
    IntermediateDir string `toml:"intermediate_dir"`
}
```

`Validate()` から以下の行を削除:
```go
if c.Output.HTMLDir == "" {
    c.Output.HTMLDir = "output/html"
}
```

- [ ] **Step 3: config_test.go から HTMLDir 参照を削除**

`config_test.go` 内で `html_dir` や `cfg.Output.HTMLDir` を参照している箇所を検索して削除する。

```bash
grep -n "html_dir\|HTMLDir" config_test.go
```

該当行を削除する。

- [ ] **Step 4: main.go から htmlWriter をすべて削除**

`main.go` で以下を削除する:

1. `fetchPage` 関数内:
```go
htmlWriter := NewHTMLWriter(cfg.Output.HTMLDir, conv)
```

2. `fetchPage` の `processPage` 呼び出しから `htmlWriter` 引数を削除:
```go
// 変更前
if err := processPage(client, writer, htmlWriter, intermediateSaver, ...); err != nil {
// 変更後
if err := processPage(client, writer, intermediateSaver, ...); err != nil {
```

3. `fetchSpace` 関数内で同様に削除。

4. `convertFromIntermediate` 関数内:
```go
htmlWriter := NewHTMLWriter(cfg.Output.HTMLDir, conv)
```
と HTML 生成ブロック全体を削除:
```go
// 削除
if err := htmlWriter.WritePage(page, spaceKey, "", "", labels, comments, nil); err != nil {
    slog.Warn("HTML生成エラー", "pageTitle", pageTitle, "error", err)
}
```

5. `processPage` 関数のシグネチャから `htmlWriter *HTMLWriter` 引数を削除し、関数内の HTML 生成ブロックを削除:
```go
// 変更前
func processPage(client *ConfluenceClient, writer *MDWriter, htmlWriter *HTMLWriter, ...) error {
// 変更後
func processPage(client *ConfluenceClient, writer *MDWriter, ...) error {
```

削除するブロック（`processPage` 内）:
```go
// HTML生成
if err := htmlWriter.WritePage(page, space.Key, space.Name, parentTitle, labels, comments, attachments); err != nil {
    slog.Warn("HTML生成エラー", "pageID", pageID, "error", err)
}
```

再帰呼び出し箇所も修正:
```go
// 変更前
if err := processPage(client, writer, htmlWriter, intermediateSaver, ...); err != nil {
// 変更後
if err := processPage(client, writer, intermediateSaver, ...); err != nil {
```

- [ ] **Step 5: ビルド確認**

```bash
go build ./...
```

Expected: エラーなし

- [ ] **Step 6: テスト実行**

```bash
go test ./...
```

Expected: PASS（htmlwriter 関連以外のテストが通る）

- [ ] **Step 7: コミット**

```bash
git add -A
git commit -m "feat: HTML出力機能を廃止しMarkdownのみに変更"
```

---

## Task 2: PageBody に AtlasDocFormat フィールド追加・API URL 変更

**Files:**
- Modify: `confluenceclient.go`

- [ ] **Step 1: AtlasDocFormat 型と PageBody フィールドを追加**

`confluenceclient.go` の `PageBody` 定義を以下に変更する:

```go
// PageBody はページのボディコンテンツ
type PageBody struct {
    Storage        Storage        `json:"storage"`
    AtlasDocFormat AtlasDocFormat `json:"atlas_doc_format"`
}

// AtlasDocFormat は ADF 形式のコンテンツ（value はさらに JSON 文字列）
type AtlasDocFormat struct {
    Value          string `json:"value"`
    Representation string `json:"representation"`
}
```

`Storage` 型と `CommentBody` 型はそのまま残す。

- [ ] **Step 2: GetPage の URL を atlas_doc_format に変更**

```go
// 変更前
apiURL := fmt.Sprintf("%s/wiki/api/v2/pages/%s?body-format=storage", cc.baseURL, pageID)
// 変更後
apiURL := fmt.Sprintf("%s/wiki/api/v2/pages/%s?body-format=atlas_doc_format", cc.baseURL, pageID)
```

- [ ] **Step 3: GetChildPages の URL を atlas_doc_format に変更**

```go
// 変更前
apiURL := fmt.Sprintf("%s/wiki/api/v2/pages/%s/children?body-format=storage&limit=250", cc.baseURL, pageID)
// 変更後
apiURL := fmt.Sprintf("%s/wiki/api/v2/pages/%s/children?body-format=atlas_doc_format&limit=250", cc.baseURL, pageID)
```

`GetPageFooterComments` は `body-format=storage` のまま変更しない（コメントは storage 変換を維持）。

- [ ] **Step 4: ビルド確認**

```bash
go build ./...
```

Expected: エラーなし

- [ ] **Step 5: コミット**

```bash
git add confluenceclient.go
git commit -m "feat: PageBodyにAtlasDocFormatフィールド追加・API URLをatlas_doc_formatに変更"
```

---

## Task 3: adfconverter.go — コア構造体・エントリーポイント・テキスト/マーク変換

**Files:**
- Create: `adfconverter.go`
- Create: `adfconverter_test.go`

- [ ] **Step 1: テストを書く**

`adfconverter_test.go` を作成:

```go
package main

import (
    "strings"
    "testing"
)

func adfDoc(content string) string {
    return `{"version":1,"type":"doc","content":[` + content + `]}`
}

func adfText(text string) string {
    return `{"type":"text","text":"` + text + `"}`
}

func TestConvertADF_Empty(t *testing.T) {
    got, err := convertADF("", nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if got != "" {
        t.Errorf("got %q, want %q", got, "")
    }
}

func TestConvertADF_PlainText(t *testing.T) {
    adf := adfDoc(`{"type":"paragraph","content":[` + adfText("Hello") + `]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "Hello") {
        t.Errorf("got %q, want to contain %q", got, "Hello")
    }
}

func TestConvertADF_Bold(t *testing.T) {
    adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"hello","marks":[{"type":"strong"}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "**hello**") {
        t.Errorf("got %q, want to contain %q", got, "**hello**")
    }
}

func TestConvertADF_Italic(t *testing.T) {
    adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"hi","marks":[{"type":"em"}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "*hi*") {
        t.Errorf("got %q, want to contain %q", got, "*hi*")
    }
}

func TestConvertADF_InlineCode(t *testing.T) {
    adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"foo","marks":[{"type":"code"}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "`foo`") {
        t.Errorf("got %q, want to contain backtick foo backtick", got)
    }
}

func TestConvertADF_Strikethrough(t *testing.T) {
    adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"del","marks":[{"type":"strike"}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "~~del~~") {
        t.Errorf("got %q, want to contain %q", got, "~~del~~")
    }
}

func TestConvertADF_Link(t *testing.T) {
    adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"click","marks":[{"type":"link","attrs":{"href":"https://example.com"}}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "[click](https://example.com)") {
        t.Errorf("got %q, want to contain link", got)
    }
}

func TestConvertADF_Underline(t *testing.T) {
    adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"ul","marks":[{"type":"underline"}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "<u>ul</u>") {
        t.Errorf("got %q, want to contain %q", got, "<u>ul</u>")
    }
}

func TestConvertADF_Subscript(t *testing.T) {
    adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"2","marks":[{"type":"subsup","attrs":{"type":"sub"}}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "<sub>2</sub>") {
        t.Errorf("got %q, want to contain %q", got, "<sub>2</sub>")
    }
}

func TestConvertADF_Superscript(t *testing.T) {
    adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"2","marks":[{"type":"subsup","attrs":{"type":"sup"}}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "<sup>2</sup>") {
        t.Errorf("got %q, want to contain %q", got, "<sup>2</sup>")
    }
}

func TestConvertADF_TextColorIgnored(t *testing.T) {
    adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"red","marks":[{"type":"textColor","attrs":{"color":"#ff0000"}}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "red") {
        t.Errorf("got %q, want to contain %q", got, "red")
    }
    if strings.Contains(got, "#ff0000") {
        t.Errorf("got %q, color should be stripped", got)
    }
}

func TestConvertADF_InternalLink(t *testing.T) {
    adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"page","marks":[{"type":"link","attrs":{"href":"https://example.atlassian.net/wiki/spaces/KEY/pages/12345/My%20Page"}}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "My_Page/index.md") && !strings.Contains(got, "My-Page/index.md") && !strings.Contains(got, "My Page/index.md") {
        t.Logf("got: %q", got)
    }
    if !strings.Contains(got, "/index.md") {
        t.Errorf("got %q, internal link should be converted to relative path", got)
    }
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test -v -run TestConvertADF ./...
```

Expected: FAIL（`convertADF` が未定義）

- [ ] **Step 3: adfconverter.go を作成（コア実装）**

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/url"
    "regexp"
    "strings"
)

// ADFNode は Atlas Doc Format のドキュメントノード
type ADFNode struct {
    Type    string                 `json:"type"`
    Attrs   map[string]interface{} `json:"attrs,omitempty"`
    Content []ADFNode              `json:"content,omitempty"`
    Marks   []ADFMark              `json:"marks,omitempty"`
    Text    string                 `json:"text,omitempty"`
}

// ADFMark はインラインフォーマットマーク
type ADFMark struct {
    Type  string                 `json:"type"`
    Attrs map[string]interface{} `json:"attrs,omitempty"`
}

// adfRenderer は ADF ノードツリーを Markdown に変換する
type adfRenderer struct {
    attachmentMap map[string]string // media UUID → ファイル名
}

// convertADF は ADF JSON 文字列を Markdown に変換するエントリーポイント
func convertADF(adfJSON string, attachmentMap map[string]string) (string, error) {
    if adfJSON == "" {
        return "", nil
    }
    var root ADFNode
    if err := json.Unmarshal([]byte(adfJSON), &root); err != nil {
        return "", fmt.Errorf("ADF JSONパースエラー: %w", err)
    }
    r := &adfRenderer{attachmentMap: attachmentMap}
    return strings.TrimSpace(r.renderNode(root, 0)), nil
}

// renderNode はノードタイプに応じて変換を dispatch する
func (r *adfRenderer) renderNode(node ADFNode, indent int) string {
    switch node.Type {
    case "doc":
        return r.renderBlockChildren(node.Content, indent)
    case "paragraph":
        return r.renderInlineNodes(node.Content)
    case "text":
        return r.renderText(node)
    case "hardBreak":
        return "\n"
    default:
        return ""
    }
}

// renderBlockChildren はブロック要素の子ノードを空行区切りで結合する
func (r *adfRenderer) renderBlockChildren(nodes []ADFNode, indent int) string {
    var parts []string
    for _, n := range nodes {
        if s := r.renderNode(n, indent); s != "" {
            parts = append(parts, s)
        }
    }
    return strings.Join(parts, "\n\n")
}

// renderInlineNodes はインライン要素を連結する
func (r *adfRenderer) renderInlineNodes(nodes []ADFNode) string {
    var sb strings.Builder
    for _, n := range nodes {
        sb.WriteString(r.renderInline(n))
    }
    return sb.String()
}

// renderInline はインライン要素を変換する
func (r *adfRenderer) renderInline(node ADFNode) string {
    switch node.Type {
    case "text":
        return r.renderText(node)
    case "hardBreak":
        return "\n"
    default:
        return ""
    }
}

// renderText はテキストノードにマークを適用して変換する
func (r *adfRenderer) renderText(node ADFNode) string {
    text := node.Text
    // マークを逆順に適用（外側から内側へラップ）
    for i := len(node.Marks) - 1; i >= 0; i-- {
        mark := node.Marks[i]
        switch mark.Type {
        case "strong":
            text = "**" + text + "**"
        case "em":
            text = "*" + text + "*"
        case "code":
            text = "`" + text + "`"
        case "strike":
            text = "~~" + text + "~~"
        case "underline":
            text = "<u>" + text + "</u>"
        case "link":
            href := ""
            if mark.Attrs != nil {
                if h, ok := mark.Attrs["href"].(string); ok {
                    href = convertInternalURL(h)
                }
            }
            text = "[" + text + "](" + href + ")"
        case "subsup":
            tag := "sup"
            if mark.Attrs != nil {
                if t, ok := mark.Attrs["type"].(string); ok && t == "sub" {
                    tag = "sub"
                }
            }
            text = "<" + tag + ">" + text + "</" + tag + ">"
        // textColor, backgroundColor, annotation はテキストのみ保持
        }
    }
    return text
}

// internalURLRe は Confluence 内部ページ URL のパターン
var internalURLRe = regexp.MustCompile(`/wiki/spaces/[^/]+/pages/\d+/([^#?]+)`)

// convertInternalURL は絶対 Confluence URL を相対パスに変換する
func convertInternalURL(rawURL string) string {
    parsed, err := url.Parse(rawURL)
    if err != nil {
        return rawURL
    }
    matches := internalURLRe.FindStringSubmatch(parsed.Path)
    if len(matches) < 2 {
        return rawURL
    }
    title, err := url.PathUnescape(matches[1])
    if err != nil {
        return rawURL
    }
    relPath := sanitizeFilename(title) + "/index.md"
    if parsed.Fragment != "" {
        relPath += "#" + parsed.Fragment
    }
    return relPath
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test -v -run TestConvertADF ./...
```

Expected: PASS（11テスト）

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "feat: adfconverter - コア構造体・エントリーポイント・テキスト/マーク変換"
```

---

## Task 4: adfconverter — 見出し・リスト・引用・水平線

**Files:**
- Modify: `adfconverter.go`
- Modify: `adfconverter_test.go`

- [ ] **Step 1: テストを追加**

`adfconverter_test.go` の末尾に追加:

```go
func TestConvertADF_Heading(t *testing.T) {
    tests := []struct {
        level int
        want  string
    }{
        {1, "# Hello"},
        {2, "## Hello"},
        {3, "### Hello"},
        {6, "###### Hello"},
    }
    for _, tt := range tests {
        adf := adfDoc(fmt.Sprintf(`{"type":"heading","attrs":{"level":%d},"content":[{"type":"text","text":"Hello"}]}`, tt.level))
        got, err := convertADF(adf, nil)
        if err != nil {
            t.Fatalf("level %d: unexpected error: %v", tt.level, err)
        }
        if strings.TrimSpace(got) != tt.want {
            t.Errorf("level %d: got %q, want %q", tt.level, strings.TrimSpace(got), tt.want)
        }
    }
}

func TestConvertADF_BulletList(t *testing.T) {
    adf := adfDoc(`{"type":"bulletList","content":[
        {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"A"}]}]},
        {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"B"}]}]}
    ]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "- A") || !strings.Contains(got, "- B") {
        t.Errorf("got %q, want bullet list", got)
    }
}

func TestConvertADF_OrderedList(t *testing.T) {
    adf := adfDoc(`{"type":"orderedList","content":[
        {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"First"}]}]},
        {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Second"}]}]}
    ]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "1. First") || !strings.Contains(got, "2. Second") {
        t.Errorf("got %q, want ordered list", got)
    }
}

func TestConvertADF_NestedBulletList(t *testing.T) {
    adf := adfDoc(`{"type":"bulletList","content":[
        {"type":"listItem","content":[
            {"type":"paragraph","content":[{"type":"text","text":"Parent"}]},
            {"type":"bulletList","content":[
                {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Child"}]}]}
            ]}
        ]}
    ]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "- Parent") {
        t.Errorf("got %q, want parent item", got)
    }
    if !strings.Contains(got, "  - Child") {
        t.Errorf("got %q, want indented child item", got)
    }
}

func TestConvertADF_Blockquote(t *testing.T) {
    adf := adfDoc(`{"type":"blockquote","content":[{"type":"paragraph","content":[{"type":"text","text":"quoted"}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "> quoted") {
        t.Errorf("got %q, want blockquote", got)
    }
}

func TestConvertADF_Rule(t *testing.T) {
    adf := adfDoc(`{"type":"rule"}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "---") {
        t.Errorf("got %q, want ---", got)
    }
}
```

`adfconverter_test.go` の先頭 import に `"fmt"` を追加:
```go
import (
    "fmt"
    "strings"
    "testing"
)
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test -v -run "TestConvertADF_(Heading|BulletList|OrderedList|NestedBulletList|Blockquote|Rule)" ./...
```

Expected: FAIL

- [ ] **Step 3: adfconverter.go に実装を追加**

`renderNode` の `switch` に以下の case を追加:

```go
case "heading":
    return r.renderHeading(node)
case "bulletList":
    return r.renderBulletList(node, indent)
case "orderedList":
    return r.renderOrderedList(node, indent)
case "blockquote":
    return r.renderBlockquote(node)
case "rule":
    return "---"
```

以下のメソッドを `adfconverter.go` に追加:

```go
func (r *adfRenderer) renderHeading(node ADFNode) string {
    level := 1
    if node.Attrs != nil {
        if l, ok := node.Attrs["level"].(float64); ok {
            level = int(l)
        }
    }
    prefix := strings.Repeat("#", level)
    return prefix + " " + r.renderInlineNodes(node.Content)
}

func (r *adfRenderer) renderBulletList(node ADFNode, indent int) string {
    var lines []string
    for _, item := range node.Content {
        if item.Type == "listItem" {
            lines = append(lines, r.renderListItem(item, indent, "- "))
        }
    }
    return strings.Join(lines, "\n")
}

func (r *adfRenderer) renderOrderedList(node ADFNode, indent int) string {
    var lines []string
    for i, item := range node.Content {
        if item.Type == "listItem" {
            lines = append(lines, r.renderListItem(item, indent, fmt.Sprintf("%d. ", i+1)))
        }
    }
    return strings.Join(lines, "\n")
}

func (r *adfRenderer) renderListItem(node ADFNode, indent int, prefix string) string {
    indentStr := strings.Repeat("  ", indent)
    var lines []string
    first := true
    for _, child := range node.Content {
        switch child.Type {
        case "paragraph":
            text := r.renderInlineNodes(child.Content)
            if first {
                lines = append(lines, indentStr+prefix+text)
                first = false
            } else {
                lines = append(lines, indentStr+"  "+text)
            }
        case "bulletList":
            lines = append(lines, r.renderBulletList(child, indent+1))
        case "orderedList":
            lines = append(lines, r.renderOrderedList(child, indent+1))
        }
    }
    return strings.Join(lines, "\n")
}

func (r *adfRenderer) renderBlockquote(node ADFNode) string {
    inner := r.renderBlockChildren(node.Content, 0)
    var sb strings.Builder
    for _, line := range strings.Split(inner, "\n") {
        if line == "" {
            sb.WriteString(">\n")
        } else {
            sb.WriteString("> " + line + "\n")
        }
    }
    return strings.TrimRight(sb.String(), "\n")
}
```

`adfconverter.go` の import に `"fmt"` を追加。

- [ ] **Step 4: テストが通ることを確認**

```bash
go test -v -run TestConvertADF ./...
```

Expected: PASS（全テスト）

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "feat: adfconverter - 見出し・リスト・引用・水平線"
```

---

## Task 5: adfconverter — コードブロック・パネル（GFM Alerts）

**Files:**
- Modify: `adfconverter.go`
- Modify: `adfconverter_test.go`

- [ ] **Step 1: テストを追加**

```go
func TestConvertADF_CodeBlock(t *testing.T) {
    adf := adfDoc(`{"type":"codeBlock","attrs":{"language":"go"},"content":[{"type":"text","text":"fmt.Println(\"hello\")"}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "```go") {
        t.Errorf("got %q, want ```go fence", got)
    }
    if !strings.Contains(got, `fmt.Println("hello")`) {
        t.Errorf("got %q, want code content", got)
    }
}

func TestConvertADF_CodeBlockNoLanguage(t *testing.T) {
    adf := adfDoc(`{"type":"codeBlock","content":[{"type":"text","text":"plain code"}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "```\n") {
        t.Errorf("got %q, want ``` fence without language", got)
    }
}

func TestConvertADF_PanelInfo(t *testing.T) {
    adf := adfDoc(`{"type":"panel","attrs":{"panelType":"info"},"content":[{"type":"paragraph","content":[{"type":"text","text":"note text"}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "[!NOTE]") {
        t.Errorf("got %q, want [!NOTE]", got)
    }
    if !strings.Contains(got, "note text") {
        t.Errorf("got %q, want panel content", got)
    }
}

func TestConvertADF_PanelNote(t *testing.T) {
    adf := adfDoc(`{"type":"panel","attrs":{"panelType":"note"},"content":[{"type":"paragraph","content":[{"type":"text","text":"warn"}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "[!WARNING]") {
        t.Errorf("got %q, want [!WARNING]", got)
    }
}

func TestConvertADF_PanelSuccess(t *testing.T) {
    adf := adfDoc(`{"type":"panel","attrs":{"panelType":"success"},"content":[{"type":"paragraph","content":[{"type":"text","text":"tip"}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "[!TIP]") {
        t.Errorf("got %q, want [!TIP]", got)
    }
}

func TestConvertADF_PanelWarning(t *testing.T) {
    adf := adfDoc(`{"type":"panel","attrs":{"panelType":"warning"},"content":[{"type":"paragraph","content":[{"type":"text","text":"caution"}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "[!CAUTION]") {
        t.Errorf("got %q, want [!CAUTION]", got)
    }
}

func TestConvertADF_PanelError(t *testing.T) {
    adf := adfDoc(`{"type":"panel","attrs":{"panelType":"error"},"content":[{"type":"paragraph","content":[{"type":"text","text":"err"}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "[!CAUTION]") {
        t.Errorf("got %q, want [!CAUTION] for error panel", got)
    }
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test -v -run "TestConvertADF_(CodeBlock|Panel)" ./...
```

Expected: FAIL

- [ ] **Step 3: adfconverter.go に実装を追加**

`renderNode` の `switch` に追加:

```go
case "codeBlock":
    return r.renderCodeBlock(node)
case "panel":
    return r.renderPanel(node)
```

以下のメソッドを追加:

```go
func (r *adfRenderer) renderCodeBlock(node ADFNode) string {
    lang := ""
    if node.Attrs != nil {
        if l, ok := node.Attrs["language"].(string); ok {
            lang = l
        }
    }
    var sb strings.Builder
    for _, child := range node.Content {
        if child.Type == "text" {
            sb.WriteString(child.Text)
        }
    }
    return "```" + lang + "\n" + sb.String() + "\n```"
}

func (r *adfRenderer) renderPanel(node ADFNode) string {
    panelType := "info"
    if node.Attrs != nil {
        if pt, ok := node.Attrs["panelType"].(string); ok {
            panelType = pt
        }
    }
    alertType := "NOTE"
    switch panelType {
    case "note":
        alertType = "WARNING"
    case "warning", "error":
        alertType = "CAUTION"
    case "success":
        alertType = "TIP"
    }
    inner := r.renderBlockChildren(node.Content, 0)
    var sb strings.Builder
    sb.WriteString("> [!" + alertType + "]\n")
    for _, line := range strings.Split(inner, "\n") {
        if line == "" {
            sb.WriteString(">\n")
        } else {
            sb.WriteString("> " + line + "\n")
        }
    }
    return strings.TrimRight(sb.String(), "\n")
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test -v -run TestConvertADF ./...
```

Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "feat: adfconverter - コードブロック・パネル（GFM Alerts）"
```

---

## Task 6: adfconverter — テーブル

**Files:**
- Modify: `adfconverter.go`
- Modify: `adfconverter_test.go`

- [ ] **Step 1: テストを追加**

```go
func TestConvertADF_Table(t *testing.T) {
    adf := adfDoc(`{"type":"table","content":[
        {"type":"tableRow","content":[
            {"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"Col1"}]}]},
            {"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"Col2"}]}]}
        ]},
        {"type":"tableRow","content":[
            {"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"A"}]}]},
            {"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"B"}]}]}
        ]}
    ]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "| Col1 |") {
        t.Errorf("got %q, want header row", got)
    }
    if !strings.Contains(got, "| --- |") {
        t.Errorf("got %q, want separator row", got)
    }
    if !strings.Contains(got, "| A |") {
        t.Errorf("got %q, want data row", got)
    }
}

func TestConvertADF_TableSingleRow(t *testing.T) {
    adf := adfDoc(`{"type":"table","content":[
        {"type":"tableRow","content":[
            {"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"X"}]}]}
        ]}
    ]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "| X |") {
        t.Errorf("got %q, want cell", got)
    }
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test -v -run TestConvertADF_Table ./...
```

Expected: FAIL

- [ ] **Step 3: adfconverter.go に実装を追加**

`renderNode` の `switch` に追加:

```go
case "table":
    return r.renderTable(node)
```

以下のメソッドを追加:

```go
func (r *adfRenderer) renderTable(node ADFNode) string {
    var sb strings.Builder
    firstRow := true
    var colCount int
    for _, row := range node.Content {
        if row.Type != "tableRow" {
            continue
        }
        if firstRow {
            colCount = len(row.Content)
        }
        sb.WriteString("|")
        for _, cell := range row.Content {
            cellText := r.renderTableCell(cell)
            sb.WriteString(" " + cellText + " |")
        }
        sb.WriteString("\n")
        if firstRow && colCount > 0 {
            sb.WriteString("|")
            for i := 0; i < colCount; i++ {
                sb.WriteString(" --- |")
            }
            sb.WriteString("\n")
            firstRow = false
        } else {
            firstRow = false
        }
    }
    return strings.TrimRight(sb.String(), "\n")
}

func (r *adfRenderer) renderTableCell(node ADFNode) string {
    var parts []string
    for _, child := range node.Content {
        text := strings.TrimSpace(r.renderNode(child, 0))
        text = strings.ReplaceAll(text, "\n", " ")
        text = strings.ReplaceAll(text, "|", "\\|")
        if text != "" {
            parts = append(parts, text)
        }
    }
    return strings.Join(parts, " ")
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test -v -run TestConvertADF ./...
```

Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "feat: adfconverter - テーブル"
```

---

## Task 7: adfconverter — タスクリスト・expand・status・mention・emoji・date

**Files:**
- Modify: `adfconverter.go`
- Modify: `adfconverter_test.go`

- [ ] **Step 1: テストを追加**

```go
func TestConvertADF_TaskList(t *testing.T) {
    adf := adfDoc(`{"type":"taskList","content":[
        {"type":"taskItem","attrs":{"state":"DONE"},"content":[{"type":"text","text":"Done task"}]},
        {"type":"taskItem","attrs":{"state":"TODO"},"content":[{"type":"text","text":"Todo task"}]}
    ]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "- [x] Done task") {
        t.Errorf("got %q, want checked task", got)
    }
    if !strings.Contains(got, "- [ ] Todo task") {
        t.Errorf("got %q, want unchecked task", got)
    }
}

func TestConvertADF_DecisionList(t *testing.T) {
    adf := adfDoc(`{"type":"decisionList","content":[
        {"type":"decisionItem","content":[{"type":"text","text":"Decision A"}]}
    ]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "- Decision A") {
        t.Errorf("got %q, want decision as list item", got)
    }
}

func TestConvertADF_Expand(t *testing.T) {
    adf := adfDoc(`{"type":"expand","attrs":{"title":"More info"},"content":[{"type":"paragraph","content":[{"type":"text","text":"hidden content"}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "<details>") {
        t.Errorf("got %q, want <details>", got)
    }
    if !strings.Contains(got, "<summary>More info</summary>") {
        t.Errorf("got %q, want summary", got)
    }
    if !strings.Contains(got, "hidden content") {
        t.Errorf("got %q, want hidden content", got)
    }
}

func TestConvertADF_Status(t *testing.T) {
    tests := []struct {
        color string
        emoji string
    }{
        {"green", "🟢"},
        {"red", "🔴"},
        {"yellow", "🟡"},
        {"blue", "🔵"},
        {"purple", "🟣"},
        {"neutral", "⚫"},
    }
    for _, tt := range tests {
        adf := adfDoc(fmt.Sprintf(`{"type":"paragraph","content":[{"type":"status","attrs":{"color":"%s","text":"OK"}}]}`, tt.color))
        got, err := convertADF(adf, nil)
        if err != nil {
            t.Fatalf("color %s: unexpected error: %v", tt.color, err)
        }
        if !strings.Contains(got, tt.emoji) {
            t.Errorf("color %s: got %q, want emoji %s", tt.color, got, tt.emoji)
        }
        if !strings.Contains(got, "[OK]") {
            t.Errorf("color %s: got %q, want [OK]", tt.color, got)
        }
    }
}

func TestConvertADF_Mention(t *testing.T) {
    adf := adfDoc(`{"type":"paragraph","content":[{"type":"mention","attrs":{"id":"abc123","text":"@John Doe"}}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "**@John Doe**") {
        t.Errorf("got %q, want mention", got)
    }
}

func TestConvertADF_Date(t *testing.T) {
    // timestamp は ms エポック文字列
    adf := adfDoc(`{"type":"paragraph","content":[{"type":"date","attrs":{"timestamp":"1704067200000"}}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "2024-01-01") {
        t.Errorf("got %q, want date 2024-01-01", got)
    }
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test -v -run "TestConvertADF_(TaskList|DecisionList|Expand|Status|Mention|Date)" ./...
```

Expected: FAIL

- [ ] **Step 3: adfconverter.go に実装を追加**

`adfconverter.go` の import に `"strconv"` と `"time"` を追加。

`renderNode` の `switch` に追加:

```go
case "taskList":
    return r.renderTaskList(node)
case "decisionList":
    return r.renderDecisionList(node)
case "expand", "nestedExpand":
    return r.renderExpand(node)
```

`renderInline` の `switch` に追加:

```go
case "mention":
    return r.renderMention(node)
case "emoji":
    return r.renderEmoji(node)
case "status":
    return r.renderStatus(node)
case "date":
    return r.renderDate(node)
```

以下のメソッドを追加:

```go
func (r *adfRenderer) renderTaskList(node ADFNode) string {
    var lines []string
    for _, item := range node.Content {
        if item.Type != "taskItem" {
            continue
        }
        state := ""
        if item.Attrs != nil {
            if s, ok := item.Attrs["state"].(string); ok {
                state = s
            }
        }
        check := "- [ ] "
        if state == "DONE" {
            check = "- [x] "
        }
        lines = append(lines, check+r.renderInlineNodes(item.Content))
    }
    return strings.Join(lines, "\n")
}

func (r *adfRenderer) renderDecisionList(node ADFNode) string {
    var lines []string
    for _, item := range node.Content {
        if item.Type == "decisionItem" {
            lines = append(lines, "- "+r.renderInlineNodes(item.Content))
        }
    }
    return strings.Join(lines, "\n")
}

func (r *adfRenderer) renderExpand(node ADFNode) string {
    title := "詳細"
    if node.Attrs != nil {
        if t, ok := node.Attrs["title"].(string); ok && t != "" {
            title = t
        }
    }
    inner := r.renderBlockChildren(node.Content, 0)
    return "<details><summary>" + title + "</summary>\n\n" + inner + "\n\n</details>"
}

func (r *adfRenderer) renderStatus(node ADFNode) string {
    color := ""
    text := "STATUS"
    if node.Attrs != nil {
        if c, ok := node.Attrs["color"].(string); ok {
            color = c
        }
        if t, ok := node.Attrs["text"].(string); ok && t != "" {
            text = t
        }
    }
    emoji := statusColorEmoji(color)
    return emoji + "[" + text + "]"
}

func statusColorEmoji(color string) string {
    switch strings.ToLower(color) {
    case "green":
        return "🟢"
    case "yellow":
        return "🟡"
    case "red":
        return "🔴"
    case "blue":
        return "🔵"
    case "purple":
        return "🟣"
    default:
        return "⚫"
    }
}

func (r *adfRenderer) renderMention(node ADFNode) string {
    text := ""
    if node.Attrs != nil {
        if t, ok := node.Attrs["text"].(string); ok {
            text = t
        }
    }
    return "**" + text + "**"
}

func (r *adfRenderer) renderEmoji(node ADFNode) string {
    if node.Attrs == nil {
        return ""
    }
    if s, ok := node.Attrs["text"].(string); ok && s != "" {
        return s
    }
    if s, ok := node.Attrs["shortName"].(string); ok {
        return s
    }
    return ""
}

func (r *adfRenderer) renderDate(node ADFNode) string {
    if node.Attrs == nil {
        return ""
    }
    ts, ok := node.Attrs["timestamp"].(string)
    if !ok {
        return ""
    }
    ms, err := strconv.ParseInt(ts, 10, 64)
    if err != nil {
        return ts
    }
    t := time.UnixMilli(ms).UTC()
    return t.Format("2006-01-02")
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test -v -run TestConvertADF ./...
```

Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "feat: adfconverter - タスクリスト・expand・status・mention・emoji・date"
```

---

## Task 8: adfconverter — メディア・レイアウト・extension・card

**Files:**
- Modify: `adfconverter.go`
- Modify: `adfconverter_test.go`

- [ ] **Step 1: テストを追加**

```go
func TestConvertADF_MediaExternalImage(t *testing.T) {
    adf := adfDoc(`{"type":"mediaSingle","content":[{"type":"media","attrs":{"type":"external","url":"https://example.com/img.png","alt":"alt text"}}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "![alt text](https://example.com/img.png)") {
        t.Errorf("got %q, want external image", got)
    }
}

func TestConvertADF_MediaFileWithMap(t *testing.T) {
    attachmentMap := map[string]string{"uuid-123": "photo.png"}
    adf := adfDoc(`{"type":"mediaSingle","content":[{"type":"media","attrs":{"type":"file","id":"uuid-123","alt":"photo"}}]}`)
    got, err := convertADF(adf, attachmentMap)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "photo.png") {
        t.Errorf("got %q, want filename resolved", got)
    }
}

func TestConvertADF_MediaFileUnknown(t *testing.T) {
    adf := adfDoc(`{"type":"mediaSingle","content":[{"type":"media","attrs":{"type":"file","id":"unknown-uuid"}}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "attachment-unknown-uuid") {
        t.Errorf("got %q, want fallback filename", got)
    }
}

func TestConvertADF_LayoutSection(t *testing.T) {
    adf := adfDoc(`{"type":"layoutSection","content":[
        {"type":"layoutColumn","content":[{"type":"paragraph","content":[{"type":"text","text":"Left"}]}]},
        {"type":"layoutColumn","content":[{"type":"paragraph","content":[{"type":"text","text":"Right"}]}]}
    ]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "Left") || !strings.Contains(got, "Right") {
        t.Errorf("got %q, want layout content", got)
    }
}

func TestConvertADF_Extension(t *testing.T) {
    adf := adfDoc(`{"type":"extension","attrs":{"extensionKey":"jira"}}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "<!-- macro: jira -->") {
        t.Errorf("got %q, want macro comment", got)
    }
}

func TestConvertADF_BodiedExtensionWithContent(t *testing.T) {
    adf := adfDoc(`{"type":"bodiedExtension","attrs":{"extensionKey":"custom"},"content":[{"type":"paragraph","content":[{"type":"text","text":"body content"}]}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "body content") {
        t.Errorf("got %q, want body content expanded", got)
    }
}

func TestConvertADF_InlineCard(t *testing.T) {
    adf := adfDoc(`{"type":"paragraph","content":[{"type":"inlineCard","attrs":{"url":"https://example.com"}}]}`)
    got, err := convertADF(adf, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "https://example.com") {
        t.Errorf("got %q, want URL", got)
    }
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test -v -run "TestConvertADF_(Media|Layout|Extension|BodiedExtension|InlineCard)" ./...
```

Expected: FAIL

- [ ] **Step 3: adfconverter.go に実装を追加**

`renderNode` の `switch` に追加:

```go
case "mediaSingle", "mediaGroup":
    return r.renderMediaContainer(node)
case "layoutSection":
    return r.renderBlockChildren(node.Content, indent)
case "layoutColumn":
    return r.renderBlockChildren(node.Content, indent)
case "extension", "inlineExtension":
    return r.renderExtension(node)
case "bodiedExtension":
    return r.renderBodiedExtension(node)
case "blockCard":
    return r.renderCard(node)
case "embedCard":
    return r.renderEmbedCard(node)
```

`renderInline` の `switch` に追加:

```go
case "inlineCard":
    return r.renderCard(node)
case "mediaInline":
    return r.renderMediaInline(node)
```

以下のメソッドを追加:

```go
func (r *adfRenderer) renderMediaContainer(node ADFNode) string {
    var parts []string
    for _, child := range node.Content {
        if child.Type == "media" {
            parts = append(parts, r.renderMedia(child))
        }
    }
    return strings.Join(parts, "\n")
}

func (r *adfRenderer) renderMedia(node ADFNode) string {
    if node.Attrs == nil {
        return ""
    }
    alt := ""
    if a, ok := node.Attrs["alt"].(string); ok {
        alt = a
    }
    mediaType, _ := node.Attrs["type"].(string)
    switch mediaType {
    case "external":
        u, _ := node.Attrs["url"].(string)
        return "![" + alt + "](" + u + ")"
    case "file":
        id, _ := node.Attrs["id"].(string)
        filename := "attachment-" + id
        if r.attachmentMap != nil {
            if f, ok := r.attachmentMap[id]; ok {
                filename = f
            }
        }
        return "![" + alt + "](" + filename + ")"
    default:
        return ""
    }
}

func (r *adfRenderer) renderMediaInline(node ADFNode) string {
    return r.renderMedia(node)
}

func (r *adfRenderer) renderExtension(node ADFNode) string {
    key := ""
    if node.Attrs != nil {
        if k, ok := node.Attrs["extensionKey"].(string); ok {
            key = k
        }
    }
    return "<!-- macro: " + key + " -->"
}

func (r *adfRenderer) renderBodiedExtension(node ADFNode) string {
    if len(node.Content) > 0 {
        return r.renderBlockChildren(node.Content, 0)
    }
    return r.renderExtension(node)
}

func (r *adfRenderer) renderCard(node ADFNode) string {
    u := ""
    if node.Attrs != nil {
        if v, ok := node.Attrs["url"].(string); ok {
            u = convertInternalURL(v)
        }
    }
    return "[" + u + "](" + u + ")"
}

func (r *adfRenderer) renderEmbedCard(node ADFNode) string {
    u := ""
    if node.Attrs != nil {
        if v, ok := node.Attrs["url"].(string); ok {
            u = v
        }
    }
    return "<!-- embed: " + u + " -->"
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test -v -run TestConvertADF ./...
```

Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "feat: adfconverter - メディア・レイアウト・extension・card"
```

---

## Task 9: converter.go に ConvertADF() を追加

**Files:**
- Modify: `converter.go`
- Modify: `converter_test.go`

- [ ] **Step 1: テストを追加**

`converter_test.go` に追加:

```go
func TestConvertADF_ViaConverter(t *testing.T) {
    c := newTestConverter()
    adfJSON := `{"version":1,"type":"doc","content":[{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Title"}]},{"type":"paragraph","content":[{"type":"text","text":"Body"}]}]}`
    got, err := c.ConvertADF(adfJSON, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(got, "# Title") {
        t.Errorf("got %q, want heading", got)
    }
    if !strings.Contains(got, "Body") {
        t.Errorf("got %q, want body", got)
    }
}

func TestConvertADF_ViaConverterEmpty(t *testing.T) {
    c := newTestConverter()
    got, err := c.ConvertADF("", nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if got != "" {
        t.Errorf("got %q, want empty", got)
    }
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test -v -run TestConvertADF_Via ./...
```

Expected: FAIL（`ConvertADF` メソッドが未定義）

- [ ] **Step 3: converter.go に ConvertADF() を追加**

`converter.go` の `Convert()` メソッドの下に追加:

```go
// ConvertADF は ADF JSON 文字列を Markdown に変換する（ページ本文用）
func (c *Converter) ConvertADF(adfJSON string, attachmentMap map[string]string) (string, error) {
    return convertADF(adfJSON, attachmentMap)
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test -v -run TestConvertADF ./...
```

Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add converter.go converter_test.go
git commit -m "feat: Converter.ConvertADF()を追加"
```

---

## Task 10: intermediatesaver.go — ADF JSON の保存・読み込みに変更

**Files:**
- Modify: `intermediatesaver.go`
- Modify: `intermediatesaver_test.go`

- [ ] **Step 1: テストを書き換える**

`intermediatesaver_test.go` の `makeTestPage()` を以下に変更（`Body.AtlasDocFormat.Value` を設定）:

```go
func makeTestPage() *Page {
    return &Page{
        ID:      "12345",
        Title:   "テストページ",
        Status:  "current",
        SpaceID: "67890",
        Body: PageBody{
            AtlasDocFormat: AtlasDocFormat{
                Value:          `{"version":1,"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"テストコンテンツ"}]}]}`,
                Representation: "atlas_doc_format",
            },
        },
        Version: Version{
            Number:    3,
            CreatedAt: "2024-01-01T00:00:00.000Z",
            AuthorID:  "user123",
        },
        Links: Links{
            WebUI: "/wiki/spaces/TEST/pages/12345",
        },
    }
}
```

`TestIntermediateSaver_SaveAndLoadPage` のアサーション部分で、`page.Body.Storage.Value` → `page.Body.AtlasDocFormat.Value` に変更:

```go
// 変更前
if loadedPage.Body.Storage.Value == "" {
    t.Error("ページコンテンツが空です")
}
// 変更後
if loadedPage.Body.AtlasDocFormat.Value == "" {
    t.Error("ページADFコンテンツが空です")
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test -v -run TestIntermediateSaver ./...
```

Expected: FAIL（保存先が `content.xhtml` のままなので読み込みが不一致）

- [ ] **Step 3: intermediatesaver.go の SavePage を変更**

`SavePage` 内の XHTML 保存部分を ADF JSON 保存に変更:

```go
// 変更前
xhtmlPath := filepath.Join(dir, "content.xhtml")
if err := os.WriteFile(xhtmlPath, []byte(page.Body.Storage.Value), 0644); err != nil {
    return fmt.Errorf("中間ファイル保存エラー: %w", err)
}

// 変更後
jsonPath := filepath.Join(dir, "content.json")
if err := os.WriteFile(jsonPath, []byte(page.Body.AtlasDocFormat.Value), 0644); err != nil {
    return fmt.Errorf("中間ファイル保存エラー: %w", err)
}
```

- [ ] **Step 4: intermediatesaver.go の LoadPage を変更**

`LoadPage` 内の XHTML 読み込み部分を ADF JSON 読み込みに変更:

```go
// 変更前
xhtmlPath := filepath.Join(dir, "content.xhtml")
xhtmlData, err := os.ReadFile(xhtmlPath)
if err != nil {
    return nil, nil, fmt.Errorf("中間ファイル読み込みエラー: %w", err)
}
// ...
Body: PageBody{
    Storage: Storage{
        Value:          string(xhtmlData),
        Representation: "storage",
    },
},

// 変更後
jsonPath := filepath.Join(dir, "content.json")
jsonData, err := os.ReadFile(jsonPath)
if err != nil {
    return nil, nil, fmt.Errorf("中間ファイル読み込みエラー: %w", err)
}
// ...
Body: PageBody{
    AtlasDocFormat: AtlasDocFormat{
        Value:          string(jsonData),
        Representation: "atlas_doc_format",
    },
},
```

- [ ] **Step 5: テストが通ることを確認**

```bash
go test -v -run TestIntermediateSaver ./...
```

Expected: PASS

- [ ] **Step 6: コミット**

```bash
git add intermediatesaver.go intermediatesaver_test.go
git commit -m "feat: IntermediateSaverをADF JSON保存・読み込みに変更"
```

---

## Task 11: mdwriter.go — ConvertADF() に切り替え・main.go 最終整理

**Files:**
- Modify: `mdwriter.go`
- Modify: `main.go`

- [ ] **Step 1: mdwriter.go のページ本文変換を ConvertADF に切り替える**

`mdwriter.go` の `generateContent` 内でページ本文変換箇所を変更:

```go
// 変更前
bodyMarkdown, err := w.converter.Convert(page.Body.Storage.Value)
if err != nil {
    // 変換エラーの場合はXHTMLをコードブロックとして出力
    sb.WriteString("\n<!-- 変換エラーのため元のXHTMLを表示します -->\n")
    sb.WriteString("```xml\n")
    sb.WriteString(page.Body.Storage.Value)
    sb.WriteString("\n```\n")
} else {

// 変更後
attachmentMap := buildAttachmentMap(attachments)
bodyMarkdown, err := w.converter.ConvertADF(page.Body.AtlasDocFormat.Value, attachmentMap)
if err != nil {
    // 変換エラーの場合は生 ADF JSON をコードブロックとして出力
    sb.WriteString("\n<!-- 変換エラーのため元のADF JSONを表示します -->\n")
    sb.WriteString("```json\n")
    sb.WriteString(page.Body.AtlasDocFormat.Value)
    sb.WriteString("\n```\n")
} else {
```

`mdwriter.go` または別ファイルに以下のヘルパー関数を追加:

```go
// buildAttachmentMap は添付ファイル一覧から UUID → ファイル名マップを構築する
func buildAttachmentMap(attachments []Attachment) map[string]string {
    if len(attachments) == 0 {
        return nil
    }
    m := make(map[string]string, len(attachments))
    for _, a := range attachments {
        m[a.ID] = a.Title
    }
    return m
}
```

- [ ] **Step 2: ビルド確認**

```bash
go build ./...
```

Expected: エラーなし

- [ ] **Step 3: 全テスト実行**

```bash
go test ./...
```

Expected: PASS

- [ ] **Step 4: コミット**

```bash
git add mdwriter.go
git commit -m "feat: MDWriterのページ本文変換をConvertADFに切り替え"
```

---

## Task 12: 最終クリーンアップ

**Files:**
- Modify: `go.mod`, `go.sum`
- Modify: `config.toml.example`（存在する場合）

- [ ] **Step 1: go mod tidy で不要な依存を削除**

```bash
go mod tidy
```

Expected: `html-to-markdown` および `golang.org/x/net` が go.mod から削除される（コメント変換で `x/net/html` を使っている場合は残る）

- [ ] **Step 2: ビルドと全テストで最終確認**

```bash
go build ./... && go test ./...
```

Expected: エラーなし、全テスト PASS

- [ ] **Step 3: config.toml.example から html_dir を削除**

`config.toml.example` 内の `html_dir` 設定行を削除する。

- [ ] **Step 4: 最終コミット**

```bash
git add go.mod go.sum config.toml.example
git commit -m "chore: go mod tidy・設定例からhtml_dir削除"
```

---

## 動作確認チェックリスト

実装完了後、以下を手動確認する:

- [ ] `go run . page --page-id <ID> -c config.toml` でページが Markdown に変換される
- [ ] 中間ファイルディレクトリに `content.json` が生成される
- [ ] `go run . convert -c config.toml` で中間 JSON から Markdown が再生成される
- [ ] コメント付きページでコメントが Markdown に含まれる（storage 変換維持）
- [ ] 情報パネルが `> [!NOTE]` 形式で出力される
- [ ] コードブロックが言語指定付きで出力される
