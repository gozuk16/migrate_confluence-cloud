# ADF テーブル変換の GFM 近似強化 実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Confluence ADF のテーブル（セル内リスト・引用・結合セル・入れ子テーブル・配置）を GFM テーブル + セル内 HTML 埋め込みで正しく Markdown に変換する。

**Architecture:** `adfconverter.go` の `renderTable` / `renderTableCell` を書き換える。セル内ブロック要素は HTML タグ（`<ul>`, `<blockquote>`, `<br>` 等）としてセル内に埋め込み、rowspan/colspan は仮想グリッド展開で空セル近似、入れ子テーブルは `nested-table` 拡張の `parameters.adf` を再帰変換してセル内 `<table>` HTML にする。migrate_jira-cloud の `jira_tables.go` と同じ方式。

**Tech Stack:** Go 標準ライブラリのみ（`encoding/json`, `html`, `strings`, `fmt`）。テストは標準 `testing`。

**Spec:** `docs/superpowers/specs/2026-07-04-adf-table-conversion-design.md`

## Global Constraints

- 回答・コミットメッセージ・コメントは日本語（CLAUDE.md）
- main ブランチへの直接コミット禁止。作業ブランチは `feature/adf-table-conversion`（作成済み・スペックコミット済み）
- テスト実行コマンド: `go test -v ./...`（または `make test`）
- 変更対象は `adfconverter.go` と `adfconverter_test.go` のみ（Task 5 の統合確認・ドキュメント更新を除く）
- コメント変換パス（`converter.go`）・本文段落の alignment・セル背景色・列幅はスコープ外
- 既存テスト `TestConvertADF_Table` / `TestConvertADF_TableSingleRow` は `strings.Contains` ベースの検証のため修正不要（全タスク完了後も PASS すること）
- git commit の末尾: `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`

---

### Task 1: セル内ブロック要素の HTML 埋め込み変換

`renderTableCell`（改行をスペースに潰す実装）を廃止し、ノード種別ごとにセル内 HTML 表現へ変換する `renderCellContent` / `renderCellBlock` / `renderCellListHTML` を実装する。

**Files:**
- Modify: `adfconverter.go`（`renderTableCell` を削除し新関数群に置換、`renderTable` 内の呼び出しを変更、import に `"html"` を追加）
- Test: `adfconverter_test.go`

**Interfaces:**
- Consumes: 既存の `r.renderInlineNodes(nodes []ADFNode) string`, `r.renderNode(node ADFNode, indent int) string`
- Produces:
  - `func (r *adfRenderer) renderCellContent(cell ADFNode) string` — セル全体を改行なし・`|` エスケープ済みの1行文字列にする（Task 2 のグリッド展開が使用）
  - `func (r *adfRenderer) renderCellBlock(node ADFNode) string` — セル内の1ブロック要素を改行なし文字列にする（Task 4 の `renderTableInlineHTML` も使用）
  - `func (r *adfRenderer) renderCellListHTML(node ADFNode) string` — リストを `<ul>/<ol>/<li>` HTML にする

- [ ] **Step 1: 失敗するテストを書く**

`adfconverter_test.go` の末尾に追加:

```go
func TestConvertADF_TableCellBulletList(t *testing.T) {
	// 3階層の入れ子リストを持つセル
	cell := `{"type":"tableCell","attrs":{"colspan":1,"rowspan":1},"content":[
        {"type":"bulletList","content":[
            {"type":"listItem","content":[
                {"type":"paragraph","content":[{"type":"text","text":"a"}]},
                {"type":"bulletList","content":[
                    {"type":"listItem","content":[
                        {"type":"paragraph","content":[{"type":"text","text":"b"}]},
                        {"type":"bulletList","content":[
                            {"type":"listItem","content":[
                                {"type":"paragraph","content":[{"type":"text","text":"c"}]}
                            ]}
                        ]}
                    ]}
                ]}
            ]}
        ]}
    ]}`
	adf := adfDoc(`{"type":"table","content":[{"type":"tableRow","content":[` + cell + `]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "<ul><li>a<ul><li>b<ul><li>c</li></ul></li></ul></li></ul>"
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want to contain %q", got, want)
	}
}

func TestConvertADF_TableCellOrderedList(t *testing.T) {
	cell := `{"type":"tableCell","content":[
        {"type":"orderedList","content":[
            {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"one"}]}]},
            {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"two"}]}]}
        ]}
    ]}`
	adf := adfDoc(`{"type":"table","content":[{"type":"tableRow","content":[` + cell + `]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "<ol><li>one</li><li>two</li></ol>"
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want to contain %q", got, want)
	}
}

func TestConvertADF_TableCellBlockquote(t *testing.T) {
	cell := `{"type":"tableCell","content":[
        {"type":"blockquote","content":[{"type":"paragraph","content":[{"type":"text","text":"引用"}]}]}
    ]}`
	adf := adfDoc(`{"type":"table","content":[{"type":"tableRow","content":[` + cell + `]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "<blockquote>引用</blockquote>"
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want to contain %q", got, want)
	}
}

func TestConvertADF_TableCellMultiParagraph(t *testing.T) {
	cell := `{"type":"tableCell","content":[
        {"type":"paragraph","content":[{"type":"text","text":"1行目"}]},
        {"type":"paragraph","content":[{"type":"text","text":"2行目"}]}
    ]}`
	adf := adfDoc(`{"type":"table","content":[{"type":"tableRow","content":[` + cell + `]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "1行目<br>2行目"
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want to contain %q", got, want)
	}
}

func TestConvertADF_TableCellCodeBlock(t *testing.T) {
	cell := `{"type":"tableCell","content":[
        {"type":"codeBlock","attrs":{"language":"go"},"content":[{"type":"text","text":"a < b\nc"}]}
    ]}`
	adf := adfDoc(`{"type":"table","content":[{"type":"tableRow","content":[` + cell + `]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "<code>a &lt; b<br>c</code>"
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want to contain %q", got, want)
	}
}

func TestConvertADF_TableCellTaskList(t *testing.T) {
	cell := `{"type":"tableCell","content":[
        {"type":"taskList","content":[
            {"type":"taskItem","attrs":{"state":"DONE"},"content":[{"type":"text","text":"done"}]},
            {"type":"taskItem","attrs":{"state":"TODO"},"content":[{"type":"text","text":"todo"}]}
        ]}
    ]}`
	adf := adfDoc(`{"type":"table","content":[{"type":"tableRow","content":[` + cell + `]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "<ul><li>☑ done</li><li>☐ todo</li></ul>"
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want to contain %q", got, want)
	}
}

func TestConvertADF_TableCellPipeEscape(t *testing.T) {
	cell := `{"type":"tableCell","content":[
        {"type":"paragraph","content":[{"type":"text","text":"a|b"}]}
    ]}`
	adf := adfDoc(`{"type":"table","content":[{"type":"tableRow","content":[` + cell + `]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, `a\|b`) {
		t.Errorf("got %q, want escaped pipe", got)
	}
}
```

- [ ] **Step 2: テストが失敗することを確認**

Run: `go test -run 'TestConvertADF_TableCell' -v ./...`
Expected: `TestConvertADF_TableCellPipeEscape` 以外は FAIL（現状は `- a   - b     - c` のような平坦化出力のため）。`PipeEscape` は既存動作でも PASS する（リグレッション防止用に追加）。

- [ ] **Step 3: 実装する**

`adfconverter.go` の import に `"html"` を追加:

```go
import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)
```

既存の `renderTableCell`（`func (r *adfRenderer) renderTableCell(node ADFNode) string` 全体）を削除し、同じ場所に以下を追加:

```go
// renderCellContent はセル内のブロック要素群を GFM セル用の1行文字列に変換する。
// GFM で表現できないブロック要素はセル内 HTML として埋め込む（migrate_jira-cloud 方式）。
func (r *adfRenderer) renderCellContent(cell ADFNode) string {
	var parts []string
	for _, child := range cell.Content {
		if s := r.renderCellBlock(child); s != "" {
			parts = append(parts, s)
		}
	}
	joined := strings.Join(parts, "<br>")
	// hardBreak 等が残した改行をすべて <br> に置換してから | をエスケープする
	joined = strings.ReplaceAll(joined, "\n", "<br>")
	return strings.ReplaceAll(joined, "|", "\\|")
}

// renderCellBlock はセル内の1ブロック要素を改行なしの文字列に変換する
func (r *adfRenderer) renderCellBlock(node ADFNode) string {
	switch node.Type {
	case "paragraph":
		return r.renderInlineNodes(node.Content)
	case "bulletList", "orderedList":
		return r.renderCellListHTML(node)
	case "blockquote":
		return "<blockquote>" + r.renderCellChildren(node.Content) + "</blockquote>"
	case "codeBlock":
		var sb strings.Builder
		for _, child := range node.Content {
			if child.Type == "text" {
				sb.WriteString(child.Text)
			}
		}
		code := html.EscapeString(sb.String())
		return "<code>" + strings.ReplaceAll(code, "\n", "<br>") + "</code>"
	case "taskList":
		var sb strings.Builder
		sb.WriteString("<ul>")
		for _, item := range node.Content {
			if item.Type != "taskItem" {
				continue
			}
			check := "☐ "
			if item.Attrs != nil {
				if s, ok := item.Attrs["state"].(string); ok && s == "DONE" {
					check = "☑ "
				}
			}
			sb.WriteString("<li>" + check + r.renderInlineNodes(item.Content) + "</li>")
		}
		sb.WriteString("</ul>")
		return sb.String()
	case "panel", "expand", "nestedExpand":
		return r.renderCellChildren(node.Content)
	default:
		// 未知のブロック要素は通常変換の結果を採用（改行は renderCellContent が <br> 化する）
		return strings.TrimSpace(r.renderNode(node, 0))
	}
}

// renderCellChildren は子ブロック要素群を <br> 区切りで結合する
func (r *adfRenderer) renderCellChildren(nodes []ADFNode) string {
	var parts []string
	for _, child := range nodes {
		if s := r.renderCellBlock(child); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "<br>")
}

// renderCellListHTML は bulletList / orderedList をセル内 HTML リストに変換する
func (r *adfRenderer) renderCellListHTML(node ADFNode) string {
	tag := "ul"
	if node.Type == "orderedList" {
		tag = "ol"
	}
	var sb strings.Builder
	sb.WriteString("<" + tag + ">")
	for _, item := range node.Content {
		if item.Type != "listItem" {
			continue
		}
		sb.WriteString("<li>")
		for _, child := range item.Content {
			switch child.Type {
			case "paragraph":
				sb.WriteString(r.renderInlineNodes(child.Content))
			case "bulletList", "orderedList":
				sb.WriteString(r.renderCellListHTML(child))
			}
		}
		sb.WriteString("</li>")
	}
	sb.WriteString("</" + tag + ">")
	return sb.String()
}
```

既存 `renderTable` 内の `cellText := r.renderTableCell(cell)` を `cellText := r.renderCellContent(cell)` に変更する。

- [ ] **Step 4: テストが通ることを確認**

Run: `go test -v ./...`
Expected: 追加テスト全て PASS、既存テスト（`TestConvertADF_Table` 等）も PASS

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "feat: テーブルセル内のリスト・引用・複数段落等をHTML埋め込みで変換

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 2: グリッド展開による rowspan/colspan 対応とヘッダー無しテーブル

`renderTable` を仮想グリッド展開方式に書き換え、結合セルの列ずれ解消とヘッダー無しテーブルの空ヘッダー自動生成を行う。

**Files:**
- Modify: `adfconverter.go`（`renderTable` を全面書き換え、`tableCellData` 型と `intAttr` ヘルパーを追加）
- Test: `adfconverter_test.go`

**Interfaces:**
- Consumes: Task 1 の `r.renderCellContent(cell ADFNode) string`
- Produces:
  - `type tableCellData struct { content string; align string }` — グリッドのセル情報（`align` は Task 3 で使用、本タスクでは常に空文字）
  - `func intAttr(node ADFNode, key string, defaultVal int) int` — attrs から正の整数を取得（Task 4 の `renderTableInlineHTML` も使用）
  - `renderTable` はグリッド `[][]*tableCellData` を構築してから出力する構造になる（Task 3 がセパレーター出力部を修正する）

- [ ] **Step 1: 失敗するテストを書く**

`adfconverter_test.go` の末尾に追加:

```go
func TestConvertADF_TableRowspan(t *testing.T) {
	// 2列テーブル: 1行目 A(rowspan=2), B / 2行目 C のみ
	adf := adfDoc(`{"type":"table","content":[
        {"type":"tableRow","content":[
            {"type":"tableCell","attrs":{"colspan":1,"rowspan":2},"content":[{"type":"paragraph","content":[{"type":"text","text":"A"}]}]},
            {"type":"tableCell","attrs":{"colspan":1,"rowspan":1},"content":[{"type":"paragraph","content":[{"type":"text","text":"B"}]}]}
        ]},
        {"type":"tableRow","content":[
            {"type":"tableCell","attrs":{"colspan":1,"rowspan":1},"content":[{"type":"paragraph","content":[{"type":"text","text":"C"}]}]}
        ]}
    ]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "| A | B |") {
		t.Errorf("got %q, want first data row '| A | B |'", got)
	}
	// rowspan で占有された位置は空セルになり、C は2列目に配置される
	if !strings.Contains(got, "|  | C |") {
		t.Errorf("got %q, want second data row '|  | C |'", got)
	}
}

func TestConvertADF_TableColspan(t *testing.T) {
	// 2列テーブル: 2行目 D(colspan=2)
	adf := adfDoc(`{"type":"table","content":[
        {"type":"tableRow","content":[
            {"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"H1"}]}]},
            {"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"H2"}]}]}
        ]},
        {"type":"tableRow","content":[
            {"type":"tableCell","attrs":{"colspan":2,"rowspan":1},"content":[{"type":"paragraph","content":[{"type":"text","text":"D"}]}]}
        ]}
    ]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// colspan の残り位置は空セルで埋める
	if !strings.Contains(got, "| D |  |") {
		t.Errorf("got %q, want data row '| D |  |'", got)
	}
}

func TestConvertADF_TableNoHeader(t *testing.T) {
	// 1行目が tableCell のみ → 空ヘッダー行を自動生成し、1行目はデータ行として出力
	adf := adfDoc(`{"type":"table","content":[
        {"type":"tableRow","content":[
            {"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"A"}]}]},
            {"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"B"}]}]}
        ]}
    ]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(got, "\n")
	if len(lines) < 3 {
		t.Fatalf("got %q, want 3 lines (empty header, separator, data)", got)
	}
	if lines[0] != "|  |  |" {
		t.Errorf("got %q, want empty header row '|  |  |'", lines[0])
	}
	if lines[1] != "| --- | --- |" {
		t.Errorf("got %q, want separator row", lines[1])
	}
	if lines[2] != "| A | B |" {
		t.Errorf("got %q, want data row '| A | B |'", lines[2])
	}
}

func TestConvertADF_TableRowspanColspanMixed(t *testing.T) {
	// SCRUM サンプル相当: 4列、2行目に rowspan=2 / 3行目に colspan=2 が混在
	adf := adfDoc(`{"type":"table","content":[
        {"type":"tableRow","content":[
            {"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"H1"}]}]},
            {"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"H2"}]}]},
            {"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"H3"}]}]},
            {"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"H4"}]}]}
        ]},
        {"type":"tableRow","content":[
            {"type":"tableCell","attrs":{"colspan":1,"rowspan":1},"content":[{"type":"paragraph","content":[{"type":"text","text":"a"}]}]},
            {"type":"tableCell","attrs":{"colspan":1,"rowspan":2},"content":[{"type":"paragraph","content":[{"type":"text","text":"縦結合"}]}]},
            {"type":"tableCell","attrs":{"colspan":1,"rowspan":1},"content":[{"type":"paragraph","content":[{"type":"text","text":"b"}]}]},
            {"type":"tableCell","attrs":{"colspan":1,"rowspan":1},"content":[{"type":"paragraph","content":[{"type":"text","text":"c"}]}]}
        ]},
        {"type":"tableRow","content":[
            {"type":"tableCell","attrs":{"colspan":1,"rowspan":1},"content":[{"type":"paragraph","content":[{"type":"text","text":"d"}]}]},
            {"type":"tableCell","attrs":{"colspan":2,"rowspan":1},"content":[{"type":"paragraph","content":[{"type":"text","text":"横結合"}]}]}
        ]}
    ]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "| a | 縦結合 | b | c |") {
		t.Errorf("got %q, want row2", got)
	}
	// 3行目: d, (縦結合の占有=空), 横結合, (colspanの占有=空)
	if !strings.Contains(got, "| d |  | 横結合 |  |") {
		t.Errorf("got %q, want row3 with padded cells", got)
	}
}
```

- [ ] **Step 2: テストが失敗することを確認**

Run: `go test -run 'TestConvertADF_Table' -v ./...`
Expected: `TestConvertADF_TableRowspan` / `TableColspan` / `TableNoHeader` / `TableRowspanColspanMixed` が FAIL（現状は span 無視・1行目強制ヘッダーのため）。Task 1 の `TableCell*` テストは PASS のまま。

- [ ] **Step 3: 実装する**

`adfconverter.go` の既存 `renderTable`（`func (r *adfRenderer) renderTable(node ADFNode) string` 全体）を以下に置き換え、直前に型とヘルパーを追加:

```go
// tableCellData はグリッド展開後のセル情報
type tableCellData struct {
	content string
	align   string // "": 未指定, "center": 中央, "end": 右寄せ
}

// intAttr は attrs から正の整数属性を取得する（欠落・非数値・0以下は defaultVal）
func intAttr(node ADFNode, key string, defaultVal int) int {
	if node.Attrs != nil {
		if v, ok := node.Attrs[key].(float64); ok && int(v) > 0 {
			return int(v)
		}
	}
	return defaultVal
}

func (r *adfRenderer) renderTable(node ADFNode) string {
	adfRows := make([]ADFNode, 0, len(node.Content))
	for _, row := range node.Content {
		if row.Type == "tableRow" {
			adfRows = append(adfRows, row)
		}
	}
	if len(adfRows) == 0 {
		return ""
	}

	// 1行目に tableHeader が1つでもあればヘッダー行とみなす
	hasHeader := false
	for _, cell := range adfRows[0].Content {
		if cell.Type == "tableHeader" {
			hasHeader = true
			break
		}
	}

	// 仮想グリッド展開: rowspan/colspan の占有位置を空セルで確保して列ずれを防ぐ
	var grid [][]*tableCellData
	ensureCell := func(row, col int) {
		for len(grid) <= row {
			grid = append(grid, nil)
		}
		for len(grid[row]) <= col {
			grid[row] = append(grid[row], nil)
		}
	}
	for ri, row := range adfRows {
		col := 0
		for _, cell := range row.Content {
			if cell.Type != "tableCell" && cell.Type != "tableHeader" {
				continue
			}
			ensureCell(ri, col)
			for grid[ri][col] != nil {
				col++
				ensureCell(ri, col)
			}
			colspan := intAttr(cell, "colspan", 1)
			rowspan := intAttr(cell, "rowspan", 1)
			for dr := 0; dr < rowspan; dr++ {
				for dc := 0; dc < colspan; dc++ {
					ensureCell(ri+dr, col+dc)
					grid[ri+dr][col+dc] = &tableCellData{}
				}
			}
			grid[ri][col] = &tableCellData{content: r.renderCellContent(cell)}
			col += colspan
		}
	}

	width := 0
	for _, row := range grid {
		if len(row) > width {
			width = len(row)
		}
	}
	if width == 0 {
		return ""
	}

	var sb strings.Builder
	writeRow := func(row []*tableCellData) {
		sb.WriteString("|")
		for i := 0; i < width; i++ {
			content := ""
			if i < len(row) && row[i] != nil {
				content = row[i].content
			}
			sb.WriteString(" " + content + " |")
		}
		sb.WriteString("\n")
	}
	writeSeparator := func() {
		sb.WriteString("|")
		for i := 0; i < width; i++ {
			sb.WriteString(" --- |")
		}
		sb.WriteString("\n")
	}

	rows := grid
	if hasHeader {
		writeRow(rows[0])
		writeSeparator()
		rows = rows[1:]
	} else {
		// ヘッダー無しテーブル: 空ヘッダー行を自動生成する
		writeRow(nil)
		writeSeparator()
	}
	for _, row := range rows {
		writeRow(row)
	}
	return strings.TrimRight(sb.String(), "\n")
}
```

- [ ] **Step 4: テストが通ることを確認**

Run: `go test -v ./...`
Expected: 全テスト PASS（既存の `TestConvertADF_Table` / `TestConvertADF_TableSingleRow` を含む）

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "feat: テーブルのrowspan/colspanをグリッド展開で近似・ヘッダー無しテーブルに空ヘッダーを生成

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: alignment マークの GFM 列アライメント反映

セル内段落の `alignment` マーク（`align: "center"` / `"end"`）を列単位に集約し、区切り行を `:---:` / `---:` にする。

**Files:**
- Modify: `adfconverter.go`（`cellAlignment` ヘルパー追加、`renderTable` のセル配置とセパレーター出力を修正）
- Test: `adfconverter_test.go`

**Interfaces:**
- Consumes: Task 2 の `tableCellData`（`align` フィールド）、`renderTable` のグリッド構造
- Produces: `func cellAlignment(cell ADFNode) string` — セルの配置（`""` / `"center"` / `"end"`）

- [ ] **Step 1: 失敗するテストを書く**

`adfconverter_test.go` の末尾に追加:

```go
func TestConvertADF_TableAlignment(t *testing.T) {
	// 1列目: align=end, 2列目: align=center, 3列目: 指定なし
	adf := adfDoc(`{"type":"table","content":[
        {"type":"tableRow","content":[
            {"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"H1"}]}]},
            {"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"H2"}]}]},
            {"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"H3"}]}]}
        ]},
        {"type":"tableRow","content":[
            {"type":"tableCell","content":[{"type":"paragraph","marks":[{"type":"alignment","attrs":{"align":"end"}}],"content":[{"type":"text","text":"122"}]}]},
            {"type":"tableCell","content":[{"type":"paragraph","marks":[{"type":"alignment","attrs":{"align":"center"}}],"content":[{"type":"text","text":"mid"}]}]},
            {"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"plain"}]}]}
        ]}
    ]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "| ---: | :---: | --- |") {
		t.Errorf("got %q, want separator '| ---: | :---: | --- |'", got)
	}
}
```

- [ ] **Step 2: テストが失敗することを確認**

Run: `go test -run 'TestConvertADF_TableAlignment' -v ./...`
Expected: FAIL（現状のセパレーターは常に `| --- |`）

- [ ] **Step 3: 実装する**

`adfconverter.go` に `cellAlignment` を追加（`intAttr` の直後）:

```go
// cellAlignment はセル内段落の alignment マークから配置を返す（"" / "center" / "end"）
func cellAlignment(cell ADFNode) string {
	for _, child := range cell.Content {
		if child.Type != "paragraph" {
			continue
		}
		for _, m := range child.Marks {
			if m.Type == "alignment" && m.Attrs != nil {
				if a, ok := m.Attrs["align"].(string); ok {
					return a
				}
			}
		}
	}
	return ""
}
```

`renderTable` のセル配置行を修正:

```go
			grid[ri][col] = &tableCellData{
				content: r.renderCellContent(cell),
				align:   cellAlignment(cell),
			}
```

`renderTable` の `writeSeparator` を、列ごとの配置集約（各列で最初に見つかった非空 align を採用）に置き換え:

```go
	colAligns := make([]string, width)
	for i := 0; i < width; i++ {
		for _, row := range grid {
			if i < len(row) && row[i] != nil && row[i].align != "" {
				colAligns[i] = row[i].align
				break
			}
		}
	}
	writeSeparator := func() {
		sb.WriteString("|")
		for i := 0; i < width; i++ {
			switch colAligns[i] {
			case "center":
				sb.WriteString(" :---: |")
			case "end":
				sb.WriteString(" ---: |")
			default:
				sb.WriteString(" --- |")
			}
		}
		sb.WriteString("\n")
	}
```

注意: `colAligns` の計算は `width` 確定後・`writeSeparator` 定義前に置くこと。

- [ ] **Step 4: テストが通ることを確認**

Run: `go test -v ./...`
Expected: 全テスト PASS

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "feat: セルのalignmentマークをGFM列アライメント記法に反映

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 4: 入れ子テーブル（nested-table 拡張）のセル内 HTML 変換

`extensionKey: "nested-table"` の `parameters.adf` を再帰的にパースし、セル内に1行の `<table>` HTML として埋め込む。パース不能時は既存のコメント出力にフォールバック。

**Files:**
- Modify: `adfconverter.go`（`renderNestedTableHTML` / `renderTableInlineHTML` を追加、`renderCellBlock` に `extension` ケースを追加）
- Test: `adfconverter_test.go`

**Interfaces:**
- Consumes: Task 1 の `r.renderCellBlock` / `r.renderCellChildren`、Task 2 の `intAttr`
- Produces:
  - `func (r *adfRenderer) renderNestedTableHTML(node ADFNode) (string, bool)` — nested-table 拡張なら HTML と true、それ以外・失敗時は false
  - `func (r *adfRenderer) renderTableInlineHTML(node ADFNode) string` — table ノードを1行の `<table>` HTML にする

- [ ] **Step 1: 失敗するテストを書く**

`adfconverter_test.go` の import に `"encoding/json"` を追加し、末尾にテストを追加:

```go
func TestConvertADF_TableNestedTable(t *testing.T) {
	inner := `{"type":"doc","content":[{"type":"table","content":[
        {"type":"tableRow","content":[
            {"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"内側H"}]}]}
        ]},
        {"type":"tableRow","content":[
            {"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"入れ子"}]}]}
        ]}
    ]}]}`
	quoted, err := json.Marshal(inner)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	ext := `{"type":"extension","attrs":{"extensionType":"com.atlassian.confluence.migration","extensionKey":"nested-table","parameters":{"adf":` + string(quoted) + `}}}`
	cell := `{"type":"tableCell","content":[` + ext + `]}`
	adf := adfDoc(`{"type":"table","content":[{"type":"tableRow","content":[` + cell + `]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "<table><tr><th>内側H</th></tr><tr><td>入れ子</td></tr></table>"
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want to contain %q", got, want)
	}
	if strings.Contains(got, "\n<table>") {
		t.Errorf("got %q, nested table must be inline (no leading newline)", got)
	}
}

func TestConvertADF_TableNestedTableRowspanPreserved(t *testing.T) {
	// 入れ子テーブル内の結合は HTML 属性としてそのまま保持される
	inner := `{"type":"doc","content":[{"type":"table","content":[
        {"type":"tableRow","content":[
            {"type":"tableCell","attrs":{"colspan":2,"rowspan":1},"content":[{"type":"paragraph","content":[{"type":"text","text":"W"}]}]}
        ]}
    ]}]}`
	quoted, err := json.Marshal(inner)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	ext := `{"type":"extension","attrs":{"extensionKey":"nested-table","parameters":{"adf":` + string(quoted) + `}}}`
	cell := `{"type":"tableCell","content":[` + ext + `]}`
	adf := adfDoc(`{"type":"table","content":[{"type":"tableRow","content":[` + cell + `]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, `<td colspan="2">W</td>`) {
		t.Errorf("got %q, want colspan attribute preserved", got)
	}
}

func TestConvertADF_TableNestedTableParseError(t *testing.T) {
	// parameters.adf が不正 JSON → コメントフォールバック
	ext := `{"type":"extension","attrs":{"extensionKey":"nested-table","parameters":{"adf":"not json"}}}`
	cell := `{"type":"tableCell","content":[` + ext + `]}`
	adf := adfDoc(`{"type":"table","content":[{"type":"tableRow","content":[` + cell + `]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "<!-- macro: nested-table -->") {
		t.Errorf("got %q, want comment fallback", got)
	}
}
```

- [ ] **Step 2: テストが失敗することを確認**

Run: `go test -run 'TestConvertADF_TableNested' -v ./...`
Expected: `TableNestedTable` / `TableNestedTableRowspanPreserved` が FAIL（現状はコメント出力）。`TableNestedTableParseError` は現状でも PASS する（フォールバック仕様が既存動作と同じため）。

- [ ] **Step 3: 実装する**

`adfconverter.go` の `renderCellBlock` の `switch` に `extension` ケースを追加（`case "panel", ...` の後）:

```go
	case "extension":
		if s, ok := r.renderNestedTableHTML(node); ok {
			return s
		}
		return strings.TrimSpace(r.renderNode(node, 0))
```

ファイル末尾（`renderEmbedCard` の後）に追加:

```go
// renderNestedTableHTML は nested-table 拡張ノードをセル内 <table> HTML に変換する。
// nested-table 以外の拡張・parameters.adf の欠落・パース失敗時は false を返す
// （呼び出し側が通常の extension 処理にフォールバックする）。
func (r *adfRenderer) renderNestedTableHTML(node ADFNode) (string, bool) {
	if node.Attrs == nil {
		return "", false
	}
	if k, _ := node.Attrs["extensionKey"].(string); k != "nested-table" {
		return "", false
	}
	params, _ := node.Attrs["parameters"].(map[string]any)
	if params == nil {
		return "", false
	}
	adfStr, _ := params["adf"].(string)
	if adfStr == "" {
		return "", false
	}
	var doc ADFNode
	if err := json.Unmarshal([]byte(adfStr), &doc); err != nil {
		return "", false
	}
	var table *ADFNode
	if doc.Type == "table" {
		table = &doc
	} else {
		for i := range doc.Content {
			if doc.Content[i].Type == "table" {
				table = &doc.Content[i]
				break
			}
		}
	}
	if table == nil {
		return "", false
	}
	return r.renderTableInlineHTML(*table), true
}

// renderTableInlineHTML は table ノードを1行の <table> HTML に変換する。
// GFM セル内はインライン文脈のため、結合は HTML の colspan/rowspan 属性でそのまま保持できる。
func (r *adfRenderer) renderTableInlineHTML(node ADFNode) string {
	var sb strings.Builder
	sb.WriteString("<table>")
	for _, row := range node.Content {
		if row.Type != "tableRow" {
			continue
		}
		sb.WriteString("<tr>")
		for _, cell := range row.Content {
			var tag string
			switch cell.Type {
			case "tableHeader":
				tag = "th"
			case "tableCell":
				tag = "td"
			default:
				continue
			}
			attrs := ""
			if cs := intAttr(cell, "colspan", 1); cs > 1 {
				attrs += fmt.Sprintf(` colspan="%d"`, cs)
			}
			if rs := intAttr(cell, "rowspan", 1); rs > 1 {
				attrs += fmt.Sprintf(` rowspan="%d"`, rs)
			}
			sb.WriteString("<" + tag + attrs + ">" + r.renderCellChildren(cell.Content) + "</" + tag + ">")
		}
		sb.WriteString("</tr>")
	}
	sb.WriteString("</table>")
	return sb.String()
}
```

- [ ] **Step 4: テストが通ることを確認**

Run: `go test -v ./...`
Expected: 全テスト PASS

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "feat: 入れ子テーブル(nested-table拡張)をセル内HTMLテーブルとして再帰変換

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 5: 統合確認・ドキュメント更新・PR 作成

SCRUM サンプルを再変換して実データで確認し、Hugo でビジュアル確認後、TODO.md / CHANGELOG.md を更新して PR を作成する。

**Files:**
- Modify: `TODO.md`, `CHANGELOG.md`
- 生成物確認: `output/markdown/SCRUM/2026-5-13 テスト議事録/index.md`

**Interfaces:**
- Consumes: Task 1〜4 の全実装
- Produces: PR（feature/adf-table-conversion → main）

- [ ] **Step 1: ビルドして中間ファイルから再変換する**

```bash
make build
./migConfluence convert -k SCRUM
```

Expected: `完了: 合計 N ページを変換しました`（エラーなし）

- [ ] **Step 2: 変換結果を確認する**

```bash
grep -A5 "ヘッダ" "output/markdown/SCRUM/2026-5-13 テスト議事録/index.md"
```

Expected（旧出力との比較で以下をすべて満たすこと）:
- セル内リストが `<ul><li>a<ul><li>b<ul><li>c</li></ul></li></ul></li></ul>` になっている
- 全行のセル数が4で揃っている（`| d |  | 横結合 |  |` のように空セル埋め）
- `<!-- macro: nested-table -->` が消え、セル内に `<table>...</table>` がある
- 「122」列の区切りが `---:` になっている
- 引用セルが `<blockquote>引用</blockquote>` になっている

- [ ] **Step 3: Hugo でビジュアル確認する**

```bash
make sync-and-build
make hugo-serve &
```

ブラウザ（Playwright 可）で `http://localhost:1313` の該当ページ（2026-5-13 テスト議事録）を開き、以下を確認:
- セル内に入れ子リストが階層表示される
- 入れ子テーブルがセル内に小さな表として描画される
- 列ずれがない
- 「122」セルが右寄せになっている

確認後、hugo-serve のプロセスを停止する。

- [ ] **Step 4: TODO.md / CHANGELOG.md を更新する**

`TODO.md` に完了項目として追記（既存の書式に合わせる）:

```markdown
- [x] ADF テーブル変換の GFM 近似強化（セル内リスト/引用/結合セル/入れ子テーブル/alignment 対応）
```

`CHANGELOG.md` に追記（既存の書式に合わせる）:

```markdown
### 追加
- テーブルセル内のリスト・引用・複数段落・コードブロック・タスクリストを HTML 埋め込みで変換
- テーブルの縦結合（rowspan）・横結合（colspan）をグリッド展開で近似（列ずれ解消）
- 入れ子テーブル（nested-table 拡張）をセル内 HTML テーブルとして再帰変換
- セルの配置（alignment マーク）を GFM 列アライメント記法に反映
- ヘッダー無しテーブルに空ヘッダー行を自動生成
```

- [ ] **Step 5: 最終テストとコミット**

```bash
go test -v ./...
git add TODO.md CHANGELOG.md
git commit -m "docs: ADFテーブル変換強化のTODO・CHANGELOGを更新

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

Expected: 全テスト PASS

- [ ] **Step 6: PR を作成する**

```bash
git push -u origin feature/adf-table-conversion
gh pr create --title "feat: ADFテーブル変換のGFM近似強化" --body "$(cat <<'EOF'
## 概要
Confluence ADF のテーブルで未対応だった以下の要素を変換できるようにしました。

- セル内リスト（入れ子含む）・引用・複数段落・コードブロック・タスクリスト → セル内 HTML 埋め込み
- 縦結合（rowspan）・横結合（colspan） → 仮想グリッド展開による空セル近似（列ずれ解消）
- 入れ子テーブル（nested-table 拡張） → parameters.adf を再帰変換しセル内 <table> HTML に
- 配置（alignment マーク） → GFM 列アライメント記法（:---: / ---:）
- ヘッダー無しテーブル → 空ヘッダー行の自動生成

設計: docs/superpowers/specs/2026-07-04-adf-table-conversion-design.md

## テスト
- adfconverter_test.go に単体テスト 15 件を追加、go test -v ./... 全件 PASS
- SCRUM サンプル（2026-5-13 テスト議事録）を再変換し hugo server でビジュアル確認済み

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

Expected: PR の URL が出力される
