# ページプロパティレポート再現 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Confluence の `detailssummary`（ページプロパティレポート）マクロを、Hugo サイト上で一覧表として再現する。

**Architecture:** 変換器（Go）が `details` マクロの表から項目を取り出して front matter の `[[properties]]` に保存し、`detailssummary` の CQL を平らな絞り込み条件に解析して `{{< page-properties-report ... >}}` ショートコードを出力する。テーマ側ショートコードがビルド時にサイト全体のページを絞り込み、表を出力する。

**Tech Stack:** Go 1.x（標準ライブラリ、`log/slog`）、Hugo v0.163（`layouts/_shortcodes`、`layouts/_partials`）、Playwright MCP（動作確認）

**Spec:** `docs/superpowers/specs/2026-10-03-page-properties-report-design.md`

## Global Constraints

- 応答・コメント・コミットメッセージは日本語。
- main に直接コミットしない。親リポジトリは `feature/page-properties-report`（作成済み）、テーマ（`hugo-site/themes/hugo-theme-docs`）は同名ブランチを `git checkout -b feature/page-properties-report --no-track origin/main` で作る。
- `push.default=upstream` のため、初回 push は必ず `git push -u origin HEAD:refs/heads/feature/page-properties-report`。push 後 `git ls-remote origin refs/heads/main refs/heads/feature/page-properties-report` で main が動いていないことを確認する。
- git worktree は使わない。
- 未追跡の `docs/confluence-storage-format-reference.md` と `hugo-site/content/SCRUM/` はコミットしない。
- コミットメッセージの末尾:
  ```
  Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_0181xW8n5yaw8Fn3wgvsojYp
  ```
- テストは `go test ./...`（`make test` と同じ）、整形は `gofmt -l .` が空であること。
- ショートコード引数名: `labels` `labels_mode` `labels_exclude` `space` `scope` `root` `title_is` `title_contains` `created_from` `created_to` `lastmod_from` `lastmod_to` `headings` `sort_by` `reverse` `first_column` `page_size`。
- front matter のキー: `[[properties]]` の各要素は `key` / `value`（小文字）。

## File Structure

| ファイル | 役割 |
|---|---|
| `adfconverter.go`（変更） | `adfResult` / `convertADFPage` の追加、`details` のプロパティ収集呼び出し、`detailssummary` の分岐、`extensionKey` ヘルパー |
| `pageproperties.go`（新規） | `PageProperty` 型、`details` の表からの項目抽出（`collectPageProperties`、`plainText`） |
| `reportcql.go`（新規） | CQL の字句解析・構文解析・`reportFilter` への変換、ショートコード文字列の組み立て |
| `converter.go`（変更） | `Converter.ConvertADFPage` の追加 |
| `mdwriter.go`（変更） | 本文変換を先に行い、プロパティを front matter 末尾に出力、警告をログ出力 |
| `pageproperties_test.go` / `reportcql_test.go`（新規）、`mdwriter_test.go`（変更） | テスト |
| テーマ `layouts/_shortcodes/page-properties-report.html`（新規） | 絞り込みと表の出力 |
| テーマ `layouts/_partials/ppr-date.html`（新規） | `now-4w` 等をビルド時刻基準の `YYYY-MM-DD` に変換 |
| テーマ `layouts/_shortcodes/toc.html`（新規） | サイト側から移動 |
| `hugo-site/layouts/shortcodes/toc.html`（削除） | テーマへ移動 |

---

### Task 1: details マクロからページプロパティを抽出する

**Files:**
- Create: `pageproperties.go`
- Create: `pageproperties_test.go`
- Modify: `adfconverter.go`（`adfRenderer` 構造体、`convertADF`、`renderExtension`、`renderBodiedExtension`）
- Modify: `converter.go:208-212`

**Interfaces:**
- Produces:
  - `type PageProperty struct { Key, Value string }`
  - `type adfResult struct { Markdown string; Properties []PageProperty; Warnings []string }`
  - `func convertADFPage(adfJSON string, attachmentMap map[string]string) (adfResult, error)`
  - `func (c *Converter) ConvertADFPage(adfJSON string, attachmentMap map[string]string) (adfResult, error)`
  - `func extensionKey(node ADFNode) string`
  - `adfRenderer` のフィールド `properties []PageProperty`、`seenProps map[string]bool`、`warnings []string`
  - 既存の `convertADF(adfJSON, attachmentMap) (string, error)` はシグネチャを変えず `convertADFPage` の `Markdown` を返すラッパーにする

- [ ] **Step 1: 失敗するテストを書く**

`pageproperties_test.go`:

```go
package main

import (
	"reflect"
	"strings"
	"testing"
)

// detailsTable は details マクロ（縦型の表）の ADF を組み立てる。rows は [見出しセルADF, 値セルADF] の組
func detailsMacro(rows ...[2]string) string {
	var trs []string
	for _, r := range rows {
		trs = append(trs, `{"type":"tableRow","content":[`+
			`{"type":"tableHeader","content":[{"type":"paragraph","content":[`+r[0]+`]}]},`+
			`{"type":"tableCell","content":[{"type":"paragraph","content":[`+r[1]+`]}]}]}`)
	}
	return `{"type":"bodiedExtension","attrs":{"extensionType":"com.atlassian.confluence.macro.core","extensionKey":"details"},` +
		`"content":[{"type":"table","content":[` + strings.Join(trs, ",") + `]}]}`
}

func strongText(s string) string {
	return `{"type":"text","text":"` + s + `","marks":[{"type":"strong"}]}`
}

func TestConvertADFPage_DetailsProperties(t *testing.T) {
	adf := adfDoc(detailsMacro(
		[2]string{strongText("日付"), `{"type":"text","text":"2026-10-03 "}`},
		[2]string{strongText("ステータス"), `{"type":"status","attrs":{"color":"green","text":"release"}}`},
		[2]string{adfText("なにか"), adfText("あるか")},
	))
	res, err := convertADFPage(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Properties) != 3 {
		t.Fatalf("properties = %+v, want 3 items", res.Properties)
	}
	want := []string{"日付", "ステータス", "なにか"}
	for i, k := range want {
		if res.Properties[i].Key != k {
			t.Errorf("Properties[%d].Key = %q, want %q", i, res.Properties[i].Key, k)
		}
	}
	if res.Properties[0].Value != "2026-10-03" {
		t.Errorf("日付の値 = %q, want %q", res.Properties[0].Value, "2026-10-03")
	}
	if !strings.Contains(res.Properties[1].Value, ">release</span>") {
		t.Errorf("ステータスの値 = %q, want status span", res.Properties[1].Value)
	}
	// 本文の表はこれまでどおり出力される
	if !strings.Contains(res.Markdown, "| **日付** |") {
		t.Errorf("本文に表がありません: %q", res.Markdown)
	}
}

func TestConvertADFPage_MultipleDetailsFirstWins(t *testing.T) {
	adf := adfDoc(
		detailsMacro([2]string{adfText("A"), adfText("1")}, [2]string{adfText("B"), adfText("2")}) + "," +
			detailsMacro([2]string{adfText("A"), adfText("999")}, [2]string{adfText("C"), adfText("3")}),
	)
	res, err := convertADFPage(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := res.Properties
	want := []PageProperty{{"A", "1"}, {"B", "2"}, {"C", "3"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("properties = %+v, want %+v", got, want)
	}
}

func TestConvertADFPage_NoDetails(t *testing.T) {
	res, err := convertADFPage(adfDoc(`{"type":"paragraph","content":[`+adfText("x")+`]}`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Properties) != 0 {
		t.Errorf("properties = %+v, want none", res.Properties)
	}
}

func TestConvertADFPage_SkipsEmptyKeyAndSingleCellRow(t *testing.T) {
	adf := adfDoc(`{"type":"bodiedExtension","attrs":{"extensionKey":"details"},"content":[{"type":"table","content":[` +
		`{"type":"tableRow","content":[{"type":"tableHeader","content":[{"type":"paragraph"}]},{"type":"tableCell","content":[{"type":"paragraph","content":[` + adfText("v") + `]}]}]},` +
		`{"type":"tableRow","content":[{"type":"tableHeader","content":[{"type":"paragraph","content":[` + adfText("only") + `]}]}]}` +
		`]}]}`)
	res, err := convertADFPage(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Properties) != 0 {
		t.Errorf("properties = %+v, want none", res.Properties)
	}
}
```

- [ ] **Step 2: テストが失敗することを確認する**

Run: `go test -run 'TestConvertADFPage' ./...`
Expected: コンパイルエラー（`convertADFPage` / `PageProperty` が未定義）

- [ ] **Step 3: 実装する**

`pageproperties.go`:

```go
package main

import "strings"

// PageProperty は Confluence のページプロパティ（details マクロ）の1項目。
// Value は本文の表のセルと同じ形式（Markdown とインライン HTML の混在）
type PageProperty struct {
	Key   string
	Value string
}

// collectPageProperties は details マクロの中の表から「1つ目のセル → 2つ目のセル」を項目として集める。
// 同じ項目名は先に出てきた方を採用する（1ページに details が複数ある場合も同様）。
func (r *adfRenderer) collectPageProperties(nodes []ADFNode) {
	for _, node := range nodes {
		if node.Type != "table" {
			r.collectPageProperties(node.Content)
			continue
		}
		for _, row := range node.Content {
			if row.Type != "tableRow" || len(row.Content) < 2 {
				continue
			}
			key := strings.TrimSpace(plainText(row.Content[0]))
			if key == "" {
				continue
			}
			if r.seenProps == nil {
				r.seenProps = map[string]bool{}
			}
			if r.seenProps[key] {
				continue
			}
			r.seenProps[key] = true
			value := strings.ReplaceAll(r.renderCellChildren(row.Content[1].Content), "\n", "<br>")
			r.properties = append(r.properties, PageProperty{Key: key, Value: strings.TrimSpace(value)})
		}
	}
}

// plainText はノード配下のテキストを書式なしで連結する（改行は空白にする）
func plainText(node ADFNode) string {
	switch node.Type {
	case "text":
		return node.Text
	case "hardBreak":
		return " "
	}
	var sb strings.Builder
	for _, child := range node.Content {
		sb.WriteString(plainText(child))
	}
	return sb.String()
}
```

`adfconverter.go` の変更:

```go
// adfRenderer は ADF ノードツリーを Markdown に変換する
type adfRenderer struct {
	attachmentMap map[string]string // media UUID → ファイル名
	properties    []PageProperty    // details マクロから集めたページプロパティ（出現順）
	seenProps     map[string]bool   // 収集済みの項目名
	warnings      []string          // 変換時の警告（呼び出し側がログに出す）
}

// adfResult はページ本文の変換結果
type adfResult struct {
	Markdown   string
	Properties []PageProperty
	Warnings   []string
}

// convertADF は ADF JSON 文字列を Markdown に変換するエントリーポイント
func convertADF(adfJSON string, attachmentMap map[string]string) (string, error) {
	res, err := convertADFPage(adfJSON, attachmentMap)
	return res.Markdown, err
}

// convertADFPage は ADF JSON 文字列を Markdown に変換し、ページプロパティと警告もあわせて返す
func convertADFPage(adfJSON string, attachmentMap map[string]string) (adfResult, error) {
	if adfJSON == "" {
		return adfResult{}, nil
	}
	var root ADFNode
	if err := json.Unmarshal([]byte(adfJSON), &root); err != nil {
		return adfResult{}, fmt.Errorf("ADF JSONパースエラー: %w", err)
	}
	r := &adfRenderer{attachmentMap: attachmentMap}
	md := strings.TrimSpace(r.renderNode(root, ""))
	return adfResult{Markdown: md, Properties: r.properties, Warnings: r.warnings}, nil
}
```

`renderExtension` / `renderBodiedExtension` を次のように置き換える:

```go
// extensionKey は拡張ノードのマクロ名（extensionKey）を返す
func extensionKey(node ADFNode) string {
	if node.Attrs == nil {
		return ""
	}
	k, _ := node.Attrs["extensionKey"].(string)
	return k
}

func (r *adfRenderer) renderExtension(node ADFNode) string {
	key := extensionKey(node)
	if key == "toc" {
		return "{{< toc >}}"
	}
	return "<!-- macro: " + key + " -->"
}

func (r *adfRenderer) renderBodiedExtension(node ADFNode) string {
	if extensionKey(node) == "details" {
		r.collectPageProperties(node.Content)
	}
	if len(node.Content) > 0 {
		return r.renderBlockChildren(node.Content, "")
	}
	return r.renderExtension(node)
}
```

`converter.go` の `ConvertADF` の直後に追加:

```go
// ConvertADFPage は ADF JSON 文字列を Markdown に変換し、ページプロパティと警告もあわせて返す（ページ本文用）
func (c *Converter) ConvertADFPage(adfJSON string, attachmentMap map[string]string) (adfResult, error) {
	return convertADFPage(adfJSON, attachmentMap)
}
```

- [ ] **Step 4: テストが通ることを確認する**

Run: `go test ./... && gofmt -l .`
Expected: すべて PASS、gofmt の出力なし

- [ ] **Step 5: コミット**

```bash
git add pageproperties.go pageproperties_test.go adfconverter.go converter.go
git commit -m "feat: detailsマクロの表からページプロパティを抽出する"
```

---

### Task 2: ページプロパティを front matter に出力し、警告をログに出す

**Files:**
- Modify: `mdwriter.go`（import、`generateContent`、`generateFrontMatter`）
- Test: `mdwriter_test.go`

**Interfaces:**
- Consumes: `(*Converter).ConvertADFPage`、`adfResult`、`PageProperty`（Task 1）
- Produces: `func (w *MDWriter) generateFrontMatter(page *Page, spaceKey, spaceTitle, parentTitle string, labels []Label, props []PageProperty) string`

- [ ] **Step 1: 失敗するテストを書く**

`mdwriter_test.go` の末尾に追加:

```go
func TestMDWriter_FrontMatterProperties(t *testing.T) {
	tmpDir := t.TempDir()
	writer := newTestMDWriter(tmpDir)

	adf := `{"version":1,"type":"doc","content":[{"type":"bodiedExtension","attrs":{"extensionKey":"details"},"content":[{"type":"table","content":[` +
		`{"type":"tableRow","content":[{"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"日付"}]}]},{"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"2026-10-03"}]}]}]},` +
		`{"type":"tableRow","content":[{"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"ステータス"}]}]},{"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"status","attrs":{"color":"green","text":"release"}}]}]}]}` +
		`]}]}]}`
	page := &Page{
		ID:      "1",
		Title:   "プロパティつき",
		SpaceID: "67890",
		Body:    PageBody{AtlasDocFormat: AtlasDocFormat{Value: adf}},
		Links:   Links{WebUI: "/spaces/TEST/pages/1"},
	}
	labels := []Label{{Name: "memo"}}
	if err := writer.WritePage(page, "TEST", "テストスペース", "", labels, nil, nil); err != nil {
		t.Fatalf("WritePage エラー: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title), "index.md"))
	if err != nil {
		t.Fatalf("読み込みエラー: %v", err)
	}
	content := string(data)
	fm := extractFrontMatter(data)

	// [[properties]] は他のキーより後ろ（TOML の配列テーブルのため）
	idxProps := strings.Index(fm, "[[properties]]")
	idxURL := strings.Index(fm, "confluence_url")
	idxLabels := strings.Index(fm, "labels =")
	if idxProps < 0 || idxProps < idxURL || idxProps < idxLabels {
		t.Fatalf("[[properties]] が front matter の末尾にありません:\n%s", fm)
	}
	for _, want := range []string{
		"  key = \"日付\"\n  value = \"2026-10-03\"\n",
		"  key = \"ステータス\"\n",
		`>release</span>`,
	} {
		if !strings.Contains(fm, want) {
			t.Errorf("front matter に %q がありません:\n%s", want, fm)
		}
	}
	// 本文の表も残る
	if !strings.Contains(content, "| 日付 | 2026-10-03 |") {
		t.Errorf("本文の表がありません:\n%s", content)
	}
}

func TestMDWriter_NoPropertiesNoTable(t *testing.T) {
	tmpDir := t.TempDir()
	writer := newTestMDWriter(tmpDir)
	page := &Page{
		ID:      "2",
		Title:   "プロパティなし",
		SpaceID: "67890",
		Body:    PageBody{AtlasDocFormat: AtlasDocFormat{Value: `{"version":1,"type":"doc","content":[]}`}},
	}
	if err := writer.WritePage(page, "TEST", "", "", nil, nil, nil); err != nil {
		t.Fatalf("WritePage エラー: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title), "index.md"))
	if strings.Contains(string(data), "[[properties]]") {
		t.Errorf("プロパティが無いのに [[properties]] が出力されています:\n%s", data)
	}
}
```

（`extractFrontMatter` は `mdwriter.go:149` の既存関数で、`+++` に挟まれた部分を返す。）

- [ ] **Step 2: テストが失敗することを確認する**

Run: `go test -run 'TestMDWriter_FrontMatterProperties|TestMDWriter_NoPropertiesNoTable' ./...`
Expected: `TestMDWriter_FrontMatterProperties` が FAIL（`[[properties]] が front matter の末尾にありません`）

- [ ] **Step 3: 実装する**

`mdwriter.go` の import に `"log/slog"` を追加する。

`generateContent` の先頭（`var sb strings.Builder` から本文出力まで）を次に置き換える:

```go
	var sb strings.Builder

	// 本文を先に変換する（ページプロパティを front matter に書くため）
	attachmentMap := buildAttachmentMap(attachments)
	res, convErr := w.converter.ConvertADFPage(page.Body.AtlasDocFormat.Value, attachmentMap)
	for _, msg := range res.Warnings {
		slog.Warn("変換時の警告", "pageTitle", page.Title, "detail", msg)
	}

	// Front Matter
	sb.WriteString(w.generateFrontMatter(page, spaceKey, spaceTitle, parentTitle, labels, res.Properties))

	// ページ本文
	if convErr != nil {
		// 変換エラーの場合は生 ADF JSON をコードブロックとして出力
		sb.WriteString("\n<!-- 変換エラーのため元のADF JSONを表示します -->\n")
		sb.WriteString("```json\n")
		sb.WriteString(page.Body.AtlasDocFormat.Value)
		sb.WriteString("\n```\n")
	} else {
		sb.WriteString("\n")
		sb.WriteString(res.Markdown)
		sb.WriteString("\n")
	}
```

`generateFrontMatter` のシグネチャに `props []PageProperty` を追加し、`confluence_url` の出力の直後・`sb.WriteString("+++\n")` の直前に追加:

```go
	// ページプロパティ（details マクロ）。TOML の配列テーブルなので必ず最後に書く
	for _, p := range props {
		sb.WriteString("[[properties]]\n")
		sb.WriteString(fmt.Sprintf("  key = %q\n", p.Key))
		sb.WriteString(fmt.Sprintf("  value = %q\n", p.Value))
	}
```

- [ ] **Step 4: テストが通ることを確認する**

Run: `go test ./... && gofmt -l .`
Expected: すべて PASS、gofmt の出力なし

- [ ] **Step 5: コミット**

```bash
git add mdwriter.go mdwriter_test.go
git commit -m "feat: ページプロパティをfront matterの[[properties]]に出力する"
```

---

### Task 3: CQL を絞り込み条件に解析する

**Files:**
- Create: `reportcql.go`
- Create: `reportcql_test.go`

**Interfaces:**
- Produces:
  - `type reportFilter struct { Labels []string; LabelsMode string; LabelsExclude []string; Space, Scope, Root, TitleIs, TitleContains, CreatedFrom, CreatedTo, LastmodFrom, LastmodTo string }`
  - `func parseReportCQL(cql string) (reportFilter, []string)` — 2つ目の戻り値は警告メッセージ

**解析ルール（仕様書の対応表どおり）:**
- 字句: 文字列（`"..."` / `'...'`、`\` エスケープ）、`(` `)` `,`、演算子 `!=` `!~` `>=` `<=` `=` `~` `>` `<`、それ以外の連続文字は語（数値・識別子）。キーワード・関数名・項目名は大文字小文字を区別しない。
- 句: `項目 演算子 値`。演算子は記号のほか `in` / `not in`（値は括弧つきの一覧）。値は文字列・語・関数呼び出し `名前 ( 引数... )`。
- `(` で始まる括弧グループは、中身が「肯定の label 条件（`=` / `in`）だけを `or` でつないだもの」なら1つの「いずれかのラベル」条件として扱う。それ以外は警告して除外する。
- `or`: 左右が両方とも肯定の label 条件ならまとめて「いずれかのラベル」にする。それ以外は警告して左右の句を除外する。残った句はすべて `and` として扱う。
- label: `and` の `=` だけなら `LabelsMode="all"`。`in` / `or` のグループが1つだけ（`and` の `=` なし）なら `"any"`。それ以外の混在は警告して全ラベルを `"any"` にまとめる。
- 解析エラー（字句・構文）の場合は警告を1つ出し、`reportFilter{Space: "current"}` を返す。空文字（空白のみ含む）の場合は警告なしで同じ値を返す。

- [ ] **Step 1: 失敗するテストを書く**

`reportcql_test.go`:

```go
package main

import (
	"reflect"
	"testing"
)

func TestParseReportCQL(t *testing.T) {
	tests := []struct {
		name      string
		cql       string
		want      reportFilter
		wantWarns int
	}{
		{"空", "  ", reportFilter{Space: "current"}, 0},
		{"実データ", `label = "memo" and space = currentSpace ( ) and parent = currentContent ( )`,
			reportFilter{Labels: []string{"memo"}, LabelsMode: "all", Space: "current", Scope: "children", Root: "current"}, 0},
		{"ラベルand", `label = "a" AND label = 'b'`, reportFilter{Labels: []string{"a", "b"}, LabelsMode: "all"}, 0},
		{"ラベルin", `label in ("a", "b")`, reportFilter{Labels: []string{"a", "b"}, LabelsMode: "any"}, 0},
		{"ラベルor", `label = "a" or label = "b"`, reportFilter{Labels: []string{"a", "b"}, LabelsMode: "any"}, 0},
		{"ラベルor括弧", `(label = "a" or label = "b") and space = "DEV"`,
			reportFilter{Labels: []string{"a", "b"}, LabelsMode: "any", Space: "DEV"}, 0},
		{"ラベル除外", `label != "x" and label not in ("y","z")`, reportFilter{LabelsExclude: []string{"x", "y", "z"}}, 0},
		{"ラベルandとorの混在", `label = "a" and label in ("b","c")`,
			reportFilter{Labels: []string{"a", "b", "c"}, LabelsMode: "any"}, 1},
		{"スペース引用なし", `space = DEV`, reportFilter{Space: "DEV"}, 0},
		{"親ID", `parent = 12345`, reportFilter{Scope: "children", Root: "12345"}, 0},
		{"祖先", `ancestor = currentContent()`, reportFilter{Scope: "descendants", Root: "current"}, 0},
		{"タイトル完全一致", `title = "週報"`, reportFilter{TitleIs: "週報"}, 0},
		{"タイトル部分一致", `title ~ "週報*"`, reportFilter{TitleContains: "週報"}, 0},
		{"作成日以降", `created >= "2026/01/05"`, reportFilter{CreatedFrom: "2026-01-05"}, 0},
		{"作成日より後", `created > "2026-01-05"`, reportFilter{CreatedFrom: "2026-01-06"}, 0},
		{"更新日より前", `lastmodified < "2026-03-01"`, reportFilter{LastmodTo: "2026-02-28"}, 0},
		{"作成日一致", `created = "2026-01-05"`, reportFilter{CreatedFrom: "2026-01-05", CreatedTo: "2026-01-05"}, 0},
		{"相対日付", `lastModified > now("-4w")`, reportFilter{LastmodFrom: "now-4w"}, 0},
		{"now引数なし", `created <= now()`, reportFilter{CreatedTo: "now"}, 0},
		{"type=pageは無視", `type = page and label = "a"`, reportFilter{Labels: []string{"a"}, LabelsMode: "all"}, 0},
		{"未対応の項目", `creator = currentUser() and label = "a"`, reportFilter{Labels: []string{"a"}, LabelsMode: "all"}, 1},
		{"異なる項目のor", `label = "a" or title ~ "x" and space = currentSpace()`, reportFilter{Space: "current"}, 1},
		{"括弧の入れ子は除外", `(title ~ "a" and label = "b") and space = "X"`, reportFilter{Space: "X"}, 1},
		{"解釈できない日付", `created > "yesterday"`, reportFilter{}, 1},
		{"構文エラー", `label = `, reportFilter{Space: "current"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, warns := parseReportCQL(tt.cql)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseReportCQL(%q)\n got  %+v\n want %+v", tt.cql, got, tt.want)
			}
			if len(warns) != tt.wantWarns {
				t.Errorf("warnings = %q, want %d 件", warns, tt.wantWarns)
			}
		})
	}
}
```

- [ ] **Step 2: テストが失敗することを確認する**

Run: `go test -run TestParseReportCQL ./...`
Expected: コンパイルエラー（`parseReportCQL` / `reportFilter` が未定義）

- [ ] **Step 3: 実装する**

`reportcql.go`:

```go
package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// reportFilter は detailssummary の CQL を、ショートコードで扱える平らな絞り込み条件にしたもの
type reportFilter struct {
	Labels        []string
	LabelsMode    string // "all" / "any"（Labels が空なら ""）
	LabelsExclude []string
	Space         string // "current" / スペースキー / ""（全スペース）
	Scope         string // "children" / "descendants" / ""
	Root          string // "current" / ページID
	TitleIs       string
	TitleContains string
	CreatedFrom   string // "YYYY-MM-DD" / "now" / "now-4w" 等
	CreatedTo     string
	LastmodFrom   string
	LastmodTo     string
}

type cqlToken struct {
	kind string // "word" / "string" / "op" / "(" / ")" / ","
	text string
}

type cqlValue struct {
	text string
	fn   bool     // 関数呼び出し（currentContent() 等）
	args []string // 関数の引数
}

type cqlClause struct {
	field  string // 小文字
	op     string // "=" "!=" "~" "!~" ">" ">=" "<" "<=" "in" "not in"
	values []cqlValue
	raw    string // 警告表示用
	// group が空でなければ括弧グループ（中身の句）
	group []cqlClause
	// groupOK は group が「肯定の label 条件の or」だけでできているか
	groupOK bool
}

// parseReportCQL は CQL を解析して reportFilter に変換する。表せない条件は警告を返して除外する
func parseReportCQL(cql string) (reportFilter, []string) {
	if strings.TrimSpace(cql) == "" {
		return reportFilter{Space: "current"}, nil
	}
	tokens, err := tokenizeCQL(cql)
	if err != nil {
		return reportFilter{Space: "current"}, []string{fmt.Sprintf("CQLを解析できませんでした（同じスペースの全ページを対象にします）: %v: %s", err, cql)}
	}
	p := &cqlParser{tokens: tokens}
	clauses, conns, err := p.parseExpr()
	if err == nil && p.pos < len(p.tokens) {
		err = fmt.Errorf("余分な字句 %q", p.tokens[p.pos].text)
	}
	if err != nil {
		return reportFilter{Space: "current"}, []string{fmt.Sprintf("CQLを解析できませんでした（同じスペースの全ページを対象にします）: %v: %s", err, cql)}
	}

	var warns []string
	var f reportFilter

	// or を処理する: 両側が肯定の label 条件ならまとめ、それ以外は両側を除外する
	type item struct {
		clauses []cqlClause // 1件なら通常の句、複数なら label の or グループ
		dropped bool
	}
	items := []item{{clauses: []cqlClause{clauses[0]}}}
	for i, conn := range conns {
		next := clauses[i+1]
		last := &items[len(items)-1]
		if conn == "and" {
			items = append(items, item{clauses: []cqlClause{next}})
			continue
		}
		if !last.dropped && allPositiveLabel(last.clauses) && isPositiveLabel(next) {
			last.clauses = append(last.clauses, next)
			continue
		}
		if !last.dropped {
			warns = append(warns, "異なる項目どうしの or は未対応のため除外しました: "+joinRaw(last.clauses)+" or "+next.raw)
		} else {
			warns[len(warns)-1] += " or " + next.raw
		}
		last.dropped = true
	}

	var allLabels []string
	var anyGroups [][]string
	for _, it := range items {
		if it.dropped {
			continue
		}
		if len(it.clauses) > 1 {
			anyGroups = append(anyGroups, labelValues(it.clauses))
			continue
		}
		c := it.clauses[0]
		if c.group != nil {
			if !c.groupOK {
				warns = append(warns, "括弧によるグループ化は label の or 以外は未対応のため除外しました: "+c.raw)
				continue
			}
			anyGroups = append(anyGroups, labelValues(c.group))
			continue
		}
		if c.field == "label" && c.op == "=" {
			allLabels = append(allLabels, c.values[0].text)
			continue
		}
		if c.field == "label" && c.op == "in" {
			anyGroups = append(anyGroups, labelValues([]cqlClause{c}))
			continue
		}
		if w := applyClause(&f, c); w != "" {
			warns = append(warns, w)
		}
	}

	switch {
	case len(anyGroups) == 0 && len(allLabels) > 0:
		f.Labels, f.LabelsMode = allLabels, "all"
	case len(anyGroups) == 1 && len(allLabels) == 0:
		f.Labels, f.LabelsMode = anyGroups[0], "any"
	case len(anyGroups) > 0:
		labels := allLabels
		for _, g := range anyGroups {
			labels = append(labels, g...)
		}
		f.Labels, f.LabelsMode = labels, "any"
		warns = append(warns, "label の and と or の混在は未対応のため「いずれかのラベル」として扱います: "+strings.Join(labels, ", "))
	}
	return f, warns
}

// applyClause は label の肯定条件以外の句を f に反映する。表せない場合は警告を返す
func applyClause(f *reportFilter, c cqlClause) string {
	v := cqlValue{}
	if len(c.values) > 0 {
		v = c.values[0]
	}
	unsupported := "未対応の条件のため除外しました: " + c.raw
	switch c.field {
	case "label":
		if c.op == "!=" || c.op == "not in" {
			for _, x := range c.values {
				f.LabelsExclude = append(f.LabelsExclude, x.text)
			}
			return ""
		}
	case "space":
		if c.op == "=" {
			if v.fn && strings.EqualFold(v.text, "currentspace") {
				f.Space = "current"
				return ""
			}
			if !v.fn {
				f.Space = v.text
				return ""
			}
		}
	case "parent", "ancestor":
		if c.op == "=" {
			root := ""
			if v.fn && strings.EqualFold(v.text, "currentcontent") {
				root = "current"
			} else if !v.fn {
				root = v.text
			}
			if root != "" {
				f.Root = root
				f.Scope = map[string]string{"parent": "children", "ancestor": "descendants"}[c.field]
				return ""
			}
		}
	case "title":
		switch c.op {
		case "=":
			f.TitleIs = v.text
			return ""
		case "~":
			f.TitleContains = strings.ReplaceAll(v.text, "*", "")
			return ""
		}
	case "type":
		if c.op == "=" && strings.EqualFold(v.text, "page") {
			return ""
		}
	case "created", "lastmodified":
		from, to := &f.CreatedFrom, &f.CreatedTo
		if c.field == "lastmodified" {
			from, to = &f.LastmodFrom, &f.LastmodTo
		}
		date, relative, ok := parseCQLDate(v)
		if !ok {
			return "解釈できない日付のため除外しました: " + c.raw
		}
		shift := func(days int) string {
			if relative {
				return date
			}
			t, _ := time.Parse("2006-01-02", date)
			return t.AddDate(0, 0, days).Format("2006-01-02")
		}
		switch c.op {
		case ">":
			*from = shift(1)
		case ">=":
			*from = date
		case "<":
			*to = shift(-1)
		case "<=":
			*to = date
		case "=":
			*from, *to = date, date
		default:
			return unsupported
		}
		return ""
	}
	return unsupported
}

var cqlRelativeArg = regexp.MustCompile(`^[+-]\d+[dwMy]$`)

// parseCQLDate は日付の値を "YYYY-MM-DD" または "now" / "now-4w" 形式にする
func parseCQLDate(v cqlValue) (date string, relative bool, ok bool) {
	if v.fn {
		if !strings.EqualFold(v.text, "now") {
			return "", false, false
		}
		if len(v.args) == 0 || v.args[0] == "" {
			return "now", true, true
		}
		if cqlRelativeArg.MatchString(v.args[0]) {
			return "now" + v.args[0], true, true
		}
		return "", false, false
	}
	s := strings.ReplaceAll(strings.TrimSpace(v.text), "/", "-")
	if len(s) > 10 {
		s = s[:10]
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return "", false, false
	}
	return t.Format("2006-01-02"), false, true
}

func isPositiveLabel(c cqlClause) bool {
	if c.group != nil {
		return c.groupOK
	}
	return c.field == "label" && (c.op == "=" || c.op == "in")
}

func allPositiveLabel(cs []cqlClause) bool {
	for _, c := range cs {
		if !isPositiveLabel(c) {
			return false
		}
	}
	return true
}

// labelValues は肯定の label 条件（括弧グループを含む）からラベル名を出現順に集める
func labelValues(cs []cqlClause) []string {
	var out []string
	for _, c := range cs {
		if c.group != nil {
			out = append(out, labelValues(c.group)...)
			continue
		}
		for _, v := range c.values {
			out = append(out, v.text)
		}
	}
	return out
}

func joinRaw(cs []cqlClause) string {
	parts := make([]string, len(cs))
	for i, c := range cs {
		parts[i] = c.raw
	}
	return strings.Join(parts, " or ")
}

// tokenizeCQL は CQL を字句に分ける
func tokenizeCQL(s string) ([]cqlToken, error) {
	var tokens []cqlToken
	rs := []rune(s)
	for i := 0; i < len(rs); {
		c := rs[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '"' || c == '\'':
			var sb strings.Builder
			j := i + 1
			for ; j < len(rs) && rs[j] != c; j++ {
				if rs[j] == '\\' && j+1 < len(rs) {
					j++
				}
				sb.WriteRune(rs[j])
			}
			if j >= len(rs) {
				return nil, fmt.Errorf("閉じていない引用符")
			}
			tokens = append(tokens, cqlToken{"string", sb.String()})
			i = j + 1
		case c == '(' || c == ')' || c == ',':
			tokens = append(tokens, cqlToken{string(c), string(c)})
			i++
		case c == '!' || c == '>' || c == '<' || c == '=' || c == '~':
			if i+1 < len(rs) && (rs[i+1] == '=' || (c == '!' && rs[i+1] == '~')) {
				tokens = append(tokens, cqlToken{"op", string(rs[i : i+2])})
				i += 2
				continue
			}
			if c == '!' {
				return nil, fmt.Errorf("不正な演算子 !")
			}
			tokens = append(tokens, cqlToken{"op", string(c)})
			i++
		default:
			j := i
			for j < len(rs) && !strings.ContainsRune(" \t\n\r\"'(),!<>=~", rs[j]) {
				j++
			}
			tokens = append(tokens, cqlToken{"word", string(rs[i:j])})
			i = j
		}
	}
	return tokens, nil
}

type cqlParser struct {
	tokens []cqlToken
	pos    int
}

func (p *cqlParser) peek() (cqlToken, bool) {
	if p.pos >= len(p.tokens) {
		return cqlToken{}, false
	}
	return p.tokens[p.pos], true
}

func (p *cqlParser) next() (cqlToken, bool) {
	t, ok := p.peek()
	if ok {
		p.pos++
	}
	return t, ok
}

// parseExpr は「句 (and|or 句)*」を読み、句と接続詞の並びを返す（閉じ括弧の手前で止まる）
func (p *cqlParser) parseExpr() ([]cqlClause, []string, error) {
	var clauses []cqlClause
	var conns []string
	for {
		c, err := p.parseClause()
		if err != nil {
			return nil, nil, err
		}
		clauses = append(clauses, c)
		t, ok := p.peek()
		if !ok || t.kind == ")" {
			return clauses, conns, nil
		}
		if t.kind != "word" || (!strings.EqualFold(t.text, "and") && !strings.EqualFold(t.text, "or")) {
			return nil, nil, fmt.Errorf("and / or が必要な位置に %q があります", t.text)
		}
		p.pos++
		conns = append(conns, strings.ToLower(t.text))
	}
}

func (p *cqlParser) parseClause() (cqlClause, error) {
	start := p.pos
	t, ok := p.next()
	if !ok {
		return cqlClause{}, fmt.Errorf("条件が途中で終わっています")
	}
	if t.kind == "(" {
		inner, conns, err := p.parseExpr()
		if err != nil {
			return cqlClause{}, err
		}
		if end, ok := p.next(); !ok || end.kind != ")" {
			return cqlClause{}, fmt.Errorf("閉じ括弧がありません")
		}
		okGroup := allPositiveLabel(inner)
		for _, c := range conns {
			if c != "or" {
				okGroup = false
			}
		}
		return cqlClause{group: inner, groupOK: okGroup, raw: p.raw(start)}, nil
	}
	if t.kind != "word" {
		return cqlClause{}, fmt.Errorf("項目名が必要な位置に %q があります", t.text)
	}
	c := cqlClause{field: strings.ToLower(t.text)}

	opTok, ok := p.next()
	if !ok {
		return cqlClause{}, fmt.Errorf("演算子がありません")
	}
	switch {
	case opTok.kind == "op":
		c.op = opTok.text
	case opTok.kind == "word" && strings.EqualFold(opTok.text, "in"):
		c.op = "in"
	case opTok.kind == "word" && strings.EqualFold(opTok.text, "not"):
		if in, ok := p.next(); !ok || !strings.EqualFold(in.text, "in") {
			return cqlClause{}, fmt.Errorf("not の後に in がありません")
		}
		c.op = "not in"
	default:
		return cqlClause{}, fmt.Errorf("演算子が必要な位置に %q があります", opTok.text)
	}

	if c.op == "in" || c.op == "not in" {
		if lp, ok := p.next(); !ok || lp.kind != "(" {
			return cqlClause{}, fmt.Errorf("in の後に ( がありません")
		}
		for {
			v, err := p.parseValue()
			if err != nil {
				return cqlClause{}, err
			}
			c.values = append(c.values, v)
			sep, ok := p.next()
			if !ok {
				return cqlClause{}, fmt.Errorf("閉じ括弧がありません")
			}
			if sep.kind == ")" {
				break
			}
			if sep.kind != "," {
				return cqlClause{}, fmt.Errorf("一覧の区切りに %q があります", sep.text)
			}
		}
	} else {
		v, err := p.parseValue()
		if err != nil {
			return cqlClause{}, err
		}
		c.values = []cqlValue{v}
	}
	c.raw = p.raw(start)
	return c, nil
}

func (p *cqlParser) parseValue() (cqlValue, error) {
	t, ok := p.next()
	if !ok {
		return cqlValue{}, fmt.Errorf("値がありません")
	}
	if t.kind == "string" {
		return cqlValue{text: t.text}, nil
	}
	if t.kind != "word" {
		return cqlValue{}, fmt.Errorf("値が必要な位置に %q があります", t.text)
	}
	if lp, ok := p.peek(); ok && lp.kind == "(" {
		p.pos++
		v := cqlValue{text: t.text, fn: true}
		for {
			a, ok := p.next()
			if !ok {
				return cqlValue{}, fmt.Errorf("関数の閉じ括弧がありません")
			}
			if a.kind == ")" {
				return v, nil
			}
			if a.kind == "," {
				continue
			}
			v.args = append(v.args, a.text)
		}
	}
	return cqlValue{text: t.text}, nil
}

// raw は start から現在位置までの字句を空白区切りで連結する（警告表示用）
func (p *cqlParser) raw(start int) string {
	parts := make([]string, 0, p.pos-start)
	for _, t := range p.tokens[start:p.pos] {
		if t.kind == "string" {
			parts = append(parts, fmt.Sprintf("%q", t.text))
		} else {
			parts = append(parts, t.text)
		}
	}
	return strings.Join(parts, " ")
}
```

- [ ] **Step 4: テストが通ることを確認する**

Run: `go test -run TestParseReportCQL -v ./... && go test ./... && gofmt -l .`
Expected: すべて PASS、gofmt の出力なし。失敗したケースは期待値ではなく実装側を仕様書の対応表に合わせて直す。

- [ ] **Step 5: コミット**

```bash
git add reportcql.go reportcql_test.go
git commit -m "feat: ページプロパティレポートのCQLを絞り込み条件に解析する"
```

---

### Task 4: detailssummary をショートコードに変換する

**Files:**
- Modify: `reportcql.go`（ショートコード組み立てを追加）
- Modify: `adfconverter.go`（`renderExtension` に分岐を追加）
- Test: `reportcql_test.go`

**Interfaces:**
- Consumes: `parseReportCQL`、`reportFilter`（Task 3）、`extensionKey`、`adfRenderer.warnings`、`convertADFPage`（Task 1）
- Produces:
  - `func macroParams(node ADFNode) map[string]string` — `attrs.parameters.macroParams.<名前>.value` を文字列で返す
  - `func buildPropertiesReportShortcode(f reportFilter, opts map[string]string) string`
  - `func (r *adfRenderer) renderPropertiesReport(node ADFNode) string`

- [ ] **Step 1: 失敗するテストを書く**

`reportcql_test.go` に追加（import に `"strings"` を追加）:

```go
func TestBuildPropertiesReportShortcode(t *testing.T) {
	f := reportFilter{Labels: []string{"memo", "a"}, LabelsMode: "all", Space: "current", Scope: "children", Root: "current", TitleContains: `x"y`}
	opts := map[string]string{"headings": "日付,ステータス", "sortBy": "日付", "reverseSort": "true", "firstcolumn": "ページ", "pageSize": "10", "showCommentsCount": "true"}
	got := buildPropertiesReportShortcode(f, opts)
	want := `{{< page-properties-report labels="memo,a" labels_mode="all" space="current" scope="children" root="current" title_contains="x\"y" headings="日付,ステータス" sort_by="日付" reverse="true" first_column="ページ" page_size="10" >}}`
	if got != want {
		t.Errorf("\n got  %s\n want %s", got, want)
	}
}

func TestBuildPropertiesReportShortcode_Minimal(t *testing.T) {
	got := buildPropertiesReportShortcode(reportFilter{Space: "current"}, map[string]string{"reverseSort": "false"})
	want := `{{< page-properties-report space="current" >}}`
	if got != want {
		t.Errorf("\n got  %s\n want %s", got, want)
	}
}

func TestConvertADFPage_DetailsSummary(t *testing.T) {
	adf := adfDoc(`{"type":"extension","attrs":{"extensionType":"com.atlassian.confluence.macro.core","extensionKey":"detailssummary",` +
		`"parameters":{"macroParams":{"cql":{"value":"label = \"memo\" and space = currentSpace ( ) and parent = currentContent ( )"}}}}}`)
	res, err := convertADFPage(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{{< page-properties-report labels="memo" labels_mode="all" space="current" scope="children" root="current" >}}`
	if res.Markdown != want {
		t.Errorf("\n got  %s\n want %s", res.Markdown, want)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("warnings = %q, want none", res.Warnings)
	}
}

func TestConvertADFPage_DetailsSummaryWarnings(t *testing.T) {
	adf := adfDoc(`{"type":"extension","attrs":{"extensionKey":"detailssummary",` +
		`"parameters":{"macroParams":{"cql":{"value":"creator = currentUser() and label = \"a\""}}}}}`)
	res, err := convertADFPage(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Warnings) != 1 || !strings.HasPrefix(res.Warnings[0], "ページプロパティレポート: ") {
		t.Errorf("warnings = %q, want 1件（ページプロパティレポート: で始まる）", res.Warnings)
	}
}
```

- [ ] **Step 2: テストが失敗することを確認する**

Run: `go test -run 'TestBuildPropertiesReportShortcode|TestConvertADFPage_DetailsSummary' ./...`
Expected: コンパイルエラー（`buildPropertiesReportShortcode` が未定義）

- [ ] **Step 3: 実装する**

`reportcql.go` の末尾に追加:

```go
// macroParams は拡張ノードのマクロ引数（parameters.macroParams.<名前>.value）を文字列で返す
func macroParams(node ADFNode) map[string]string {
	out := map[string]string{}
	params, _ := node.Attrs["parameters"].(map[string]any)
	mp, _ := params["macroParams"].(map[string]any)
	for k, v := range mp {
		if m, ok := v.(map[string]any); ok {
			if s, ok := m["value"].(string); ok {
				out[k] = s
			}
		}
	}
	return out
}

// buildPropertiesReportShortcode は絞り込み条件と表示オプションから page-properties-report ショートコードを組み立てる。
// 値が空の引数は出力しない
func buildPropertiesReportShortcode(f reportFilter, opts map[string]string) string {
	reverse := ""
	if strings.EqualFold(opts["reverseSort"], "true") {
		reverse = "true"
	}
	args := [][2]string{
		{"labels", strings.Join(f.Labels, ",")},
		{"labels_mode", f.LabelsMode},
		{"labels_exclude", strings.Join(f.LabelsExclude, ",")},
		{"space", f.Space},
		{"scope", f.Scope},
		{"root", f.Root},
		{"title_is", f.TitleIs},
		{"title_contains", f.TitleContains},
		{"created_from", f.CreatedFrom},
		{"created_to", f.CreatedTo},
		{"lastmod_from", f.LastmodFrom},
		{"lastmod_to", f.LastmodTo},
		{"headings", opts["headings"]},
		{"sort_by", opts["sortBy"]},
		{"reverse", reverse},
		{"first_column", opts["firstcolumn"]},
		{"page_size", opts["pageSize"]},
	}
	var sb strings.Builder
	sb.WriteString("{{< page-properties-report")
	for _, a := range args {
		if a[1] == "" {
			continue
		}
		sb.WriteString(" " + a[0] + `="` + strings.ReplaceAll(a[1], `"`, `\"`) + `"`)
	}
	sb.WriteString(" >}}")
	return sb.String()
}

// renderPropertiesReport は detailssummary マクロをショートコードに変換し、CQL 解析の警告を記録する
func (r *adfRenderer) renderPropertiesReport(node ADFNode) string {
	opts := macroParams(node)
	f, warns := parseReportCQL(opts["cql"])
	for _, w := range warns {
		r.warnings = append(r.warnings, "ページプロパティレポート: "+w)
	}
	return buildPropertiesReportShortcode(f, opts)
}
```

`adfconverter.go` の `renderExtension` に分岐を追加:

```go
func (r *adfRenderer) renderExtension(node ADFNode) string {
	switch key := extensionKey(node); key {
	case "toc":
		return "{{< toc >}}"
	case "detailssummary":
		return r.renderPropertiesReport(node)
	default:
		return "<!-- macro: " + key + " -->"
	}
}
```

- [ ] **Step 4: テストが通ることを確認する**

Run: `go test ./... && gofmt -l .`
Expected: すべて PASS、gofmt の出力なし

- [ ] **Step 5: コミット**

```bash
git add reportcql.go reportcql_test.go adfconverter.go
git commit -m "feat: detailssummaryマクロをpage-properties-reportショートコードに変換する"
```

---

### Task 5: テーマにショートコードを追加し、toc を移す

作業ディレクトリ: `hugo-site/themes/hugo-theme-docs`（テーマのリポジトリ）。

**Files:**
- Create: `layouts/_partials/ppr-date.html`
- Create: `layouts/_shortcodes/page-properties-report.html`
- Create: `layouts/_shortcodes/toc.html`

**Interfaces:**
- Consumes: Task 4 のショートコード引数名、Task 2 の front matter（`properties[].key` / `properties[].value`、既存の `space` `page_id` `parent_id` `labels` `is_folder`、`.Date`、`.Lastmod`）
- Produces: `{{< page-properties-report ... >}}`、`{{< toc >}}`

- [ ] **Step 1: テーマでブランチを作る**

```bash
cd hugo-site/themes/hugo-theme-docs
git fetch origin
git checkout main && git pull --ff-only
git checkout -b feature/page-properties-report --no-track origin/main
```

- [ ] **Step 2: 確認用の一時サイトを作る（先に失敗を確認する）**

作業用ディレクトリ（`$SCRATCH` はセッションの scratchpad。例: `/private/tmp/claude-501/.../scratchpad`）に一時サイトを作るスクリプト `$SCRATCH/ppr-site/make.sh`:

```bash
#!/bin/sh
# ページプロパティレポートの確認用一時サイトを作る
set -eu
SITE="$(cd "$(dirname "$0")" && pwd)"
THEMES="$1"   # hugo-site/themes の絶対パス
rm -rf "$SITE/content" "$SITE/public"
mkdir -p "$SITE/content/S"
cat > "$SITE/hugo.toml" <<EOF
baseURL = "/"
locale = "ja"
title = "ppr"
theme = "hugo-theme-docs"
themesDir = "$THEMES"
[markup.goldmark.renderer]
  unsafe = true
EOF
page() { # page ディレクトリ名 front matter本文
  mkdir -p "$SITE/content/S/$1"
  printf '+++\n%s\n+++\n\n%s\n' "$2" "$3" > "$SITE/content/S/$1/index.md"
}
TODAY=$(date -u +%Y-%m-%d)
page root  'title = "root"
date = "2026-01-01T00:00:00Z"
space = "S"
page_id = "1"' "## children
{{< page-properties-report labels=\"memo\" labels_mode=\"all\" space=\"current\" scope=\"children\" root=\"current\" >}}

## descendants
{{< page-properties-report scope=\"descendants\" root=\"current\" >}}

## any-exclude
{{< page-properties-report labels=\"memo,x\" labels_mode=\"any\" labels_exclude=\"skip\" >}}

## title
{{< page-properties-report title_contains=\"B\" >}}

## title-quote
{{< page-properties-report title_is=\"q\\\"t\" >}}

## created
{{< page-properties-report created_from=\"2026-02-01\" created_to=\"2026-02-28\" >}}

## relative
{{< page-properties-report lastmod_from=\"now-1w\" >}}

## sort
{{< page-properties-report sort_by=\"日付\" reverse=\"true\" headings=\"日付\" first_column=\"ページ\" page_size=\"2\" >}}

## none
{{< page-properties-report labels=\"nothing\" >}}"
page a 'title = "A"
date = "2026-02-10T00:00:00Z"
lastmod = "2026-02-10T00:00:00Z"
space = "S"
page_id = "10"
parent_id = "1"
labels = ["memo"]
[[properties]]
  key = "日付"
  value = "2026-02-10"
[[properties]]
  key = "ステータス"
  value = "<span class=\"st\">release</span>"' ''
page b 'title = "B"
date = "2026-03-01T00:00:00Z"
lastmod = "'"$TODAY"'T00:00:00Z"
space = "S"
page_id = "11"
parent_id = "1"
labels = ["x"]
[[properties]]
  key = "日付"
  value = "2026-03-01"
[[properties]]
  key = "なにか"
  value = "**太字**"' ''
page c 'title = "C"
date = "2026-01-15T00:00:00Z"
space = "S"
page_id = "12"
parent_id = "11"
labels = ["memo", "skip"]
[[properties]]
  key = "日付"
  value = "2026-01-15"' ''
page q 'title = "q\"t"
space = "S"
page_id = "13"
parent_id = "99"
[[properties]]
  key = "日付"
  value = "2025-12-31"' ''
page noprops 'title = "noprops"
space = "S"
page_id = "14"
parent_id = "1"
labels = ["memo"]' ''
```

```bash
SCRATCH=<scratchpad の絶対パス>
mkdir -p "$SCRATCH/ppr-site"   # 上の make.sh をここに保存
sh "$SCRATCH/ppr-site/make.sh" "$(pwd)/.."
hugo --source "$SCRATCH/ppr-site" --quiet
```

Expected: `shortcode "page-properties-report" not found` でビルドが失敗する。

- [ ] **Step 3: 日付変換パーシャルを書く**

`layouts/_partials/ppr-date.html`:

```go-html-template
{{- /*
  ページプロパティレポートの日付引数を "YYYY-MM-DD" にする。
  "now" / "now-4w" / "now+1d" はビルド時刻を基準に計算する（単位: d=日 w=週 M=月 y=年）。
  それ以外（変換器が正規化した "YYYY-MM-DD"）はそのまま返す。空なら空を返す。
*/ -}}
{{- $v := . | default "" }}
{{- $out := $v }}
{{- with findRESubmatch `^now(?:([+-]\d+)([dwMy]))?$` $v 1 }}
  {{- $g := index . 0 }}
  {{- $t := now.UTC }}
  {{- with index $g 1 }}
    {{- $n := int (strings.TrimPrefix "+" .) }}
    {{- $u := index $g 2 }}
    {{- if eq $u "d" }}{{ $t = $t.AddDate 0 0 $n }}
    {{- else if eq $u "w" }}{{ $t = $t.AddDate 0 0 (mul $n 7) }}
    {{- else if eq $u "M" }}{{ $t = $t.AddDate 0 $n 0 }}
    {{- else }}{{ $t = $t.AddDate $n 0 0 }}
    {{- end }}
  {{- end }}
  {{- $out = $t.Format "2006-01-02" }}
{{- end }}
{{- return $out }}
```

- [ ] **Step 4: ショートコードを書く**

`layouts/_shortcodes/page-properties-report.html`:

```go-html-template
{{- /*
  ページプロパティレポート（Confluence の detailssummary マクロ）。
  引数は変換器が CQL から生成する。front matter の properties（key / value）を持つページを絞り込み、一覧表にする。
  - labels / labels_mode（all|any）/ labels_exclude: カンマ区切りのラベル
  - space: "current" はこのページと同じスペース。無指定なら全スペース
  - scope（children|descendants）/ root（"current" か page_id）
  - title_is / title_contains（大文字小文字を区別しない）
  - created_from / created_to / lastmod_from / lastmod_to: "YYYY-MM-DD" か "now-4w" 形式（境界を含む）
  - headings / sort_by / reverse / first_column / page_size: 表示オプション
*/ -}}
{{- $page := .Page }}
{{- $labels := slice }}
{{- with .Get "labels" }}{{ range split . "," }}{{ $labels = $labels | append (trim . " ") }}{{ end }}{{ end }}
{{- $mode := .Get "labels_mode" | default "all" }}
{{- $exclude := slice }}
{{- with .Get "labels_exclude" }}{{ range split . "," }}{{ $exclude = $exclude | append (trim . " ") }}{{ end }}{{ end }}
{{- $space := .Get "space" | default "" }}
{{- if eq $space "current" }}{{ $space = $page.Params.space | default "" }}{{ end }}
{{- $scope := .Get "scope" | default "" }}
{{- $root := .Get "root" | default "" }}
{{- if eq $root "current" }}{{ $root = $page.Params.page_id | default "" }}{{ end }}
{{- $titleIs := .Get "title_is" | default "" }}
{{- $titleContains := lower (.Get "title_contains" | default "") }}
{{- $createdFrom := partial "ppr-date.html" (.Get "created_from") }}
{{- $createdTo := partial "ppr-date.html" (.Get "created_to") }}
{{- $lastmodFrom := partial "ppr-date.html" (.Get "lastmod_from") }}
{{- $lastmodTo := partial "ppr-date.html" (.Get "lastmod_to") }}
{{- $sortBy := .Get "sort_by" | default "" }}
{{- $reverse := eq (.Get "reverse") "true" }}
{{- $firstColumn := .Get "first_column" | default "タイトル" }}
{{- $pageSize := int (.Get "page_size" | default "0") }}

{{- /* descendants 用: page_id → parent_id の索引 */}}
{{- $parentOf := newScratch }}
{{- if eq $scope "descendants" }}
  {{- range $p := site.RegularPages }}
    {{- with $p.Params.page_id }}{{ $parentOf.Set . ($p.Params.parent_id | default "") }}{{ end }}
  {{- end }}
{{- end }}

{{- $items := slice }}
{{- range $p := site.RegularPages }}
  {{- $ok := and $p.Params.properties (not $p.Params.is_folder) (ne $p $page) }}
  {{- if and $ok $space }}{{ $ok = eq ($p.Params.space | default "") $space }}{{ end }}
  {{- if and $ok (eq $scope "children") }}{{ $ok = eq ($p.Params.parent_id | default "") $root }}{{ end }}
  {{- if and $ok (eq $scope "descendants") }}
    {{- $found := false }}
    {{- $pid := $p.Params.parent_id | default "" }}
    {{- range seq 20 }}
      {{- if and $pid (not $found) }}
        {{- if eq $pid $root }}{{ $found = true }}{{ else }}{{ $pid = $parentOf.Get $pid | default "" }}{{ end }}
      {{- end }}
    {{- end }}
    {{- $ok = $found }}
  {{- end }}
  {{- $pl := slice }}
  {{- range ($p.Params.labels | default slice) }}{{ $pl = $pl | append (string .) }}{{ end }}
  {{- if and $ok $labels }}
    {{- $hit := len (intersect $labels $pl) }}
    {{- if eq $mode "any" }}{{ $ok = gt $hit 0 }}{{ else }}{{ $ok = eq $hit (len (uniq $labels)) }}{{ end }}
  {{- end }}
  {{- if and $ok $exclude }}{{ $ok = eq (len (intersect $exclude $pl)) 0 }}{{ end }}
  {{- if and $ok $titleIs }}{{ $ok = eq (lower $p.Title) (lower $titleIs) }}{{ end }}
  {{- if and $ok $titleContains }}{{ $ok = strings.Contains (lower $p.Title) $titleContains }}{{ end }}
  {{- $created := $p.Date.UTC.Format "2006-01-02" }}
  {{- $lastmod := $p.Lastmod.UTC.Format "2006-01-02" }}
  {{- if and $ok $createdFrom }}{{ $ok = ge $created $createdFrom }}{{ end }}
  {{- if and $ok $createdTo }}{{ $ok = le $created $createdTo }}{{ end }}
  {{- if and $ok $lastmodFrom }}{{ $ok = ge $lastmod $lastmodFrom }}{{ end }}
  {{- if and $ok $lastmodTo }}{{ $ok = le $lastmod $lastmodTo }}{{ end }}
  {{- if $ok }}
    {{- $props := newScratch }}
    {{- range $p.Params.properties }}{{ $props.Set .key .value }}{{ end }}
    {{- $sortKey := $p.Title }}
    {{- if $sortBy }}{{ $sortKey = (($props.Get $sortBy) | default "" | markdownify | plainify | string) }}{{ end }}
    {{- $items = $items | append (dict "page" $p "props" $props.Values "title" $p.Title "sort" $sortKey) }}
  {{- end }}
{{- end }}

{{- /* 既定はタイトル昇順。sort_by の同値はタイトル順（sort は安定ソート） */}}
{{- $items = sort (sort $items "title") "sort" (cond $reverse "desc" "asc") }}
{{- if gt $pageSize 0 }}{{ $items = first $pageSize $items }}{{ end }}

{{- $headings := slice }}
{{- with .Get "headings" }}
  {{- range split . "," }}{{ $headings = $headings | append (trim . " ") }}{{ end }}
{{- else }}
  {{- range $items }}
    {{- range .page.Params.properties }}{{ if not (in $headings .key) }}{{ $headings = $headings | append .key }}{{ end }}{{ end }}
  {{- end }}
{{- end }}

{{- if $items }}
<table class="page-properties-report">
<thead><tr><th>{{ $firstColumn }}</th>{{ range $headings }}<th>{{ . }}</th>{{ end }}</tr></thead>
<tbody>
{{- range $items }}
{{- $props := .props }}
<tr><td><a href="{{ .page.RelPermalink }}">{{ .page.Title }}</a></td>{{ range $headings }}<td>{{ with index $props . }}{{ . | markdownify }}{{ end }}</td>{{ end }}</tr>
{{- end }}
</tbody>
</table>
{{- else }}
<p class="page-properties-report-empty">条件に一致するページはありません</p>
{{- end }}
```

- [ ] **Step 5: toc をテーマへ移す**

`layouts/_shortcodes/toc.html`（`hugo-site/layouts/shortcodes/toc.html` と同じ内容。サイト側の削除は Task 6 で行う）:

```go-html-template
<nav class="page-toc">
{{ .Page.TableOfContents }}
</nav>
```

- [ ] **Step 6: 一時サイトで確認する**

```bash
sh "$SCRATCH/ppr-site/make.sh" "$(pwd)/.."
hugo --source "$SCRATCH/ppr-site" --quiet && echo BUILD-OK
```

Expected: `BUILD-OK`。続けて `$SCRATCH/ppr-site/public/s/root/index.html` を読み、各見出しの直後の表が次のとおりであることを確認する（行はタイトル列の並び）。

| 見出し | 期待する行 | 期待する列 |
|---|---|---|
| children | A | タイトル / 日付 / ステータス（ステータスのセルが `<span class="st">release</span>`） |
| descendants | A, B, C | タイトル / 日付 / ステータス / なにか（B の なにか が `<strong>太字</strong>`） |
| any-exclude | A, B（C は skip で除外、noprops はプロパティなしで除外） | |
| title | B | |
| title-quote | q"t（引数の `\"` が引用符として解釈されること） | |
| created | A | |
| relative | B | |
| sort | ページ見出し、日付列のみ、B → A の2行（日付の降順で先頭2件） | ページ / 日付 |
| none | 「条件に一致するページはありません」 | |

想定と違う場合はショートコードを直して再確認する。特に `intersect` が front matter のラベル配列と一致しない、`index $props .` が値を引けない、`sort` に3引数を渡せない、などがあれば該当箇所を修正する。

- [ ] **Step 7: コミット（テーマ）**

```bash
git add layouts/_partials/ppr-date.html layouts/_shortcodes/page-properties-report.html layouts/_shortcodes/toc.html
git commit -m "feat: ページプロパティレポートのショートコードを追加し、tocショートコードをテーマへ移す"
```

---

### Task 6: 親リポジトリへ反映し、実データで確認する

作業ディレクトリ: リポジトリのルート。

**Files:**
- Delete: `hugo-site/layouts/shortcodes/toc.html`
- Modify: `hugo-site/themes/hugo-theme-docs`（submodule ポインタ）
- Modify: `TODO.md`、`CHANGELOG.md`、`README.md`（対応マクロの記述があれば追記）

- [ ] **Step 1: サイト側の toc を削除する**

```bash
git rm hugo-site/layouts/shortcodes/toc.html
```

（`hugo-site/layouts` が空になればディレクトリごと消える。）

- [ ] **Step 2: 実データを再変換してビルドする**

```bash
go build -o migConfluence . && ./migConfluence convert --space-key SCRUM
cp -r output/markdown/SCRUM/. hugo-site/content/SCRUM/
(cd hugo-site && hugo --quiet) && echo BUILD-OK
```

Expected: `BUILD-OK`。`hugo-site/content/SCRUM/2026-8-8/index.md` に `{{< page-properties-report labels="memo" labels_mode="all" space="current" scope="children" root="current" >}}`、`hugo-site/content/SCRUM/メモ/index.md` の front matter 末尾に `[[properties]]` が3件あること。変換ログに警告が出ていないこと。

- [ ] **Step 3: ブラウザで確認する**

Hugo サーバー（1313）が動いていなければ `cd hugo-site && nohup hugo server -p 1313 > /tmp/hugo-1313.log 2>&1 &` で起動する。Playwright で `http://localhost:1313/scrum/2026-8-8/` を開き、次を確認する。

- 「タイトル / 日付 / ステータス / なにか」の表に、メモ（2026-10-03、緑の release、あるか）とメモ2（2026-10-02、黄色の 作業中、ないよ）の2行がタイトル順で出ている
- タイトルのリンクから各ページへ移動できる
- `http://localhost:1313/scrum/2026-5-13-テスト議事録/`（目次を持つページ）で目次が従来どおり表示される（toc の移動確認）

スクリーンショットを撮ってユーザーに見せる。

- [ ] **Step 4: ドキュメントを更新する**

`CHANGELOG.md` の `## [Unreleased]` 直下に追加:

```markdown
### Added（ページプロパティレポート再現）
- Confluenceの「ページプロパティレポート」（detailssummaryマクロ）を一覧表として再現できるようにした。従来は `<!-- macro: detailssummary -->` というコメントになり何も表示されなかった
- ページプロパティ（detailsマクロ）の表の各行を、front matterの `[[properties]]`（`key` / `value`）に出力するようにした。本文の表はこれまでどおり表示される
- レポートの条件（CQL）は変換時に解析し、テーマの `page-properties-report` ショートコードの引数にする。対応する条件は label（=、in、!=、not in、label どうしの or）、space、parent、ancestor、title（=、~）、created / lastmodified（絶対日付と `now("-4w")` 形式）。相対日付はHugoのビルド時刻を基準に評価する。対応していない条件は変換ログに警告を出して除外する
- 表示オプション headings、sortBy、reverseSort、firstcolumn、pageSize に対応した

### Changed（目次ショートコードの配置）
- `toc` ショートコードをサイト側（`hugo-site/layouts/shortcodes/`）からテーマ側（`layouts/_shortcodes/`）へ移した。表示は変わらない
```

`TODO.md` の「進行中」から「ページプロパティレポート（detailssummary）の再現」を「完了」へ移し、確認した内容（実データ・一時サイトでの確認項目）を記入する。TODO.md 45行目付近の「toc.html はサイト直下に配置…将来的な改善候補」には「→ テーマへ移動済み」と追記する。

README.md に対応マクロの一覧があれば `detailssummary` / `details` を追加する（`grep -n "toc\|マクロ" README.md` で確認）。

- [ ] **Step 5: submodule ポインタを更新してコミットする**

```bash
git add hugo-site/themes/hugo-theme-docs TODO.md CHANGELOG.md README.md
git commit -m "feat: ページプロパティレポートを実データで確認し、テーマを更新"
git status --short
```

Expected: `git status --short` に未コミットの TODO.md が残っていない（未追跡の `docs/confluence-storage-format-reference.md` と `hugo-site/content/SCRUM/` は残ってよい）。

---

### Task 7: PR を作る

- [ ] **Step 1: テーマの PR**

```bash
cd hugo-site/themes/hugo-theme-docs
git push -u origin HEAD:refs/heads/feature/page-properties-report
git ls-remote origin refs/heads/main refs/heads/feature/page-properties-report
gh pr create --base main --head feature/page-properties-report --title "ページプロパティレポートのショートコードを追加" --body "<変更内容・確認内容。末尾に Claude Code の署名>"
```

- [ ] **Step 2: 親リポジトリの PR**

```bash
cd ../../..
git push -u origin HEAD:refs/heads/feature/page-properties-report
git ls-remote origin refs/heads/main refs/heads/feature/page-properties-report
gh pr create --base main --head feature/page-properties-report --title "Confluenceのページプロパティレポートを再現する" --body "<変更内容・確認内容・テーマPRへのリンク。末尾に Claude Code の署名>"
```

PR 本文の末尾:

```
🤖 Generated with [Claude Code](https://claude.com/claude-code)

https://claude.ai/code/session_0181xW8n5yaw8Fn3wgvsojYp
```

- [ ] **Step 3: code-reviewer サブエージェントでレビューする**

両 PR の差分を code-reviewer エージェントに渡してレビューし、指摘を反映したら追加コミット・push する。最後に `git status --short` で TODO.md のコミット漏れが無いことを確認する。
