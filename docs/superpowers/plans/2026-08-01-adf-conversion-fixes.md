# ADF変換の未対応要素修正 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ADF → Markdown 変換で欠落・崩れが起きる7項目（リスト内codeBlock、入れ子taskList、リストインデント、隣接強調run、textColor、alignment、コメント投稿者名）を修正する。

**Architecture:** `adfconverter.go` のレンダラを拡張（Task 1〜6）し、`mdwriter.go`/`main.go` にユーザー名解決関数を注入する（Task 7）。リストのインデントは「レベル数×2スペース」から「累積インデント文字列＋マーカー幅」へ移行する。

**Tech Stack:** Go（標準ライブラリのみ）、`go test`、Hugo (Goldmark) でのレンダリング確認。

**Spec:** `docs/superpowers/specs/2026-08-01-adf-conversion-fixes-design.md`

## Global Constraints

- 回答・コミットメッセージ・コメントは日本語。
- mainブランチに直接コミットしない。作業ブランチは `feature/adf-conversion-fixes`。
- git worktree を使わない。
- TDD: 失敗するテストを先に書き、実行して失敗を確認してから実装する。
- コミットメッセージ末尾に `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>` を付ける。
- テスト実行は `make test`（全体）または `go test -run <TestName> ./...`（個別）。

---

### Task 1: リスト項目内の codeBlock を出力する（A-1）

**Files:**
- Modify: `adfconverter.go`（`renderListItem`、現在249行目付近）
- Test: `adfconverter_test.go`

**Interfaces:**
- Consumes: `renderCodeBlock(node ADFNode) string`（既存。```` ```lang\n…\n``` ```` を返す）
- Produces: `renderListItem` が `codeBlock` 子ノードをインデント付きで出力する（Task 3 でインデント幅を再調整する）

- [ ] **Step 1: 失敗するテストを書く**

`adfconverter_test.go` の末尾に追加:

```go
// TestConvertADF_CodeBlockInListItem はリスト項目内のコードブロックが出力されることを確認する
func TestConvertADF_CodeBlockInListItem(t *testing.T) {
	adf := adfDoc(`{"type":"bulletList","content":[` +
		`{"type":"listItem","content":[` +
		`{"type":"paragraph","content":[` + adfText("aaa") + `]},` +
		`{"type":"codeBlock","content":[` + adfText("echo hi") + `]}` +
		`]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "- aaa\n  ```\n  echo hi\n  ```"
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want to contain %q", got, want)
	}
}

// TestConvertADF_CodeBlockAsFirstListChild は listItem の先頭子要素が codeBlock の場合を確認する
func TestConvertADF_CodeBlockAsFirstListChild(t *testing.T) {
	adf := adfDoc(`{"type":"bulletList","content":[` +
		`{"type":"listItem","content":[` +
		`{"type":"codeBlock","content":[` + adfText("aaaaaa") + `]}` +
		`]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "- ```\n  aaaaaa\n  ```"
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want to contain %q", got, want)
	}
}
```

- [ ] **Step 2: テストが失敗することを確認する**

Run: `go test -run 'TestConvertADF_CodeBlock.*List' ./...`
Expected: FAIL（codeBlock が出力されず want を含まない）

- [ ] **Step 3: 最小実装を書く**

`renderListItem` の `switch child.Type` に `codeBlock` ケースを追加:

```go
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
		case "codeBlock":
			blockLines := strings.Split(r.renderCodeBlock(child), "\n")
			rest := blockLines
			if first {
				lines = append(lines, indentStr+prefix+blockLines[0])
				rest = blockLines[1:]
				first = false
			}
			for _, bl := range rest {
				lines = append(lines, indentStr+"  "+bl)
			}
		}
	}
	return strings.Join(lines, "\n")
}
```

- [ ] **Step 4: テストが通ることを確認する**

Run: `make test`
Expected: PASS（既存テスト含め全件）

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "fix: リスト項目内のcodeBlockがMarkdown出力から欠落する問題を修正

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 2: 入れ子の taskList を出力する（A-2）

**Files:**
- Modify: `adfconverter.go`（`renderTaskList` 現在584行目付近、`renderNode` の `taskList` ケース 74行目付近）
- Test: `adfconverter_test.go`

**Interfaces:**
- Produces: `renderTaskList(node ADFNode, indent string) string`（シグネチャ変更。Task 3 はこの形を維持する）

- [ ] **Step 1: 失敗するテストを書く**

```go
// TestConvertADF_NestedTaskList は入れ子のタスクリストが出力されることを確認する
func TestConvertADF_NestedTaskList(t *testing.T) {
	adf := adfDoc(`{"type":"taskList","content":[` +
		`{"type":"taskItem","attrs":{"state":"TODO"},"content":[` + adfText("親タスク") + `]},` +
		`{"type":"taskList","content":[` +
		`{"type":"taskItem","attrs":{"state":"DONE"},"content":[` + adfText("子タスク") + `]}` +
		`]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "- [ ] 親タスク\n  - [x] 子タスク"
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want to contain %q", got, want)
	}
}
```

- [ ] **Step 2: テストが失敗することを確認する**

Run: `go test -run TestConvertADF_NestedTaskList ./...`
Expected: FAIL（子タスクが出力されない）

- [ ] **Step 3: 最小実装を書く**

`renderTaskList` を置き換え、`renderNode` の呼び出しを合わせる:

```go
// renderTaskList はタスクリストを変換する。入れ子の taskList はインデントを深くして再帰する
func (r *adfRenderer) renderTaskList(node ADFNode, indent string) string {
	var lines []string
	for _, item := range node.Content {
		switch item.Type {
		case "taskItem":
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
			lines = append(lines, indent+check+r.renderInlineNodes(item.Content))
		case "taskList":
			lines = append(lines, r.renderTaskList(item, indent+"  "))
		}
	}
	return strings.Join(lines, "\n")
}
```

`renderNode` の case を変更:

```go
	case "taskList":
		return r.renderTaskList(node, "")
```

- [ ] **Step 4: テストが通ることを確認する**

Run: `make test`
Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "fix: 入れ子のtaskListがMarkdown出力から欠落する問題を修正

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: リストのインデントをマーカー幅ベースに変更する（B-1）

**Files:**
- Modify: `adfconverter.go`（`renderNode` / `renderBlockChildren` / `renderBulletList` / `renderOrderedList` / `renderListItem` / `convertADF` および `renderNode(…, 0)`・`renderBlockChildren(…, 0)` の全呼び出し箇所）
- Test: `adfconverter_test.go`

**Interfaces:**
- Produces: `renderNode(node ADFNode, indent string) string`、`renderBlockChildren(nodes []ADFNode, indent string) string`、`renderBulletList/renderOrderedList(node ADFNode, indent string) string`、`renderListItem(node ADFNode, indent string, prefix string) string`
- 子要素のインデント = 親インデント + `len(prefix)` 個の半角スペース（`- `=2、`1. `=3、`10. `=4）

- [ ] **Step 1: 失敗するテストを書く**

```go
// TestConvertADF_NestedListInOrderedList は番号付きリスト配下の入れ子がマーカー幅(3)でインデントされることを確認する
func TestConvertADF_NestedListInOrderedList(t *testing.T) {
	adf := adfDoc(`{"type":"orderedList","content":[` +
		`{"type":"listItem","content":[` +
		`{"type":"paragraph","content":[` + adfText("番号付きリスト") + `]},` +
		`{"type":"bulletList","content":[` +
		`{"type":"listItem","content":[{"type":"paragraph","content":[` + adfText("リスト") + `]}]}` +
		`]}]},` +
		`{"type":"listItem","content":[{"type":"paragraph","content":[` + adfText("二番") + `]}]},` +
		`{"type":"listItem","content":[` +
		`{"type":"paragraph","content":[` + adfText("三番") + `]},` +
		`{"type":"codeBlock","content":[` + adfText("aaaaa") + `]}` +
		`]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "1. 番号付きリスト\n   - リスト\n2. 二番\n3. 三番\n   ```\n   aaaaa\n   ```"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: テストが失敗することを確認する**

Run: `go test -run TestConvertADF_NestedListInOrderedList ./...`
Expected: FAIL（入れ子が2スペースインデントになる）

- [ ] **Step 3: 実装する**

インデント引数を `int` から `string` に変更する。機械的な置換箇所:

1. `convertADF`: `r.renderNode(root, 0)` → `r.renderNode(root, "")`
2. `renderNode(node ADFNode, indent int)` → `renderNode(node ADFNode, indent string)`（`bulletList`/`orderedList`/`layoutSection`/`layoutColumn`/`doc` ケースはそのまま `indent` を渡す）
3. `renderBlockChildren(nodes []ADFNode, indent int)` → `(nodes []ADFNode, indent string)`
4. `renderBlockquote`（273行目付近）、`renderCellBlock` 内の `renderNode(node, 0)` 2箇所（510・513行目付近）、`renderPanel`（571行目付近）、`renderExpand`（622行目付近）、`renderBodiedExtension`（750行目付近）: `0` → `""`

リスト系レンダラを置き換え:

```go
func (r *adfRenderer) renderBulletList(node ADFNode, indent string) string {
	var lines []string
	for _, item := range node.Content {
		if item.Type == "listItem" {
			lines = append(lines, r.renderListItem(item, indent, "- "))
		}
	}
	return strings.Join(lines, "\n")
}

func (r *adfRenderer) renderOrderedList(node ADFNode, indent string) string {
	var lines []string
	for i, item := range node.Content {
		if item.Type == "listItem" {
			lines = append(lines, r.renderListItem(item, indent, fmt.Sprintf("%d. ", i+1)))
		}
	}
	return strings.Join(lines, "\n")
}

// renderListItem はリスト項目を変換する。子要素のインデントはマーカー幅から算出する
func (r *adfRenderer) renderListItem(node ADFNode, indent string, prefix string) string {
	childIndent := indent + strings.Repeat(" ", len(prefix))
	var lines []string
	first := true
	for _, child := range node.Content {
		switch child.Type {
		case "paragraph":
			text := r.renderInlineNodes(child.Content)
			if first {
				lines = append(lines, indent+prefix+text)
				first = false
			} else {
				lines = append(lines, childIndent+text)
			}
		case "bulletList":
			lines = append(lines, r.renderBulletList(child, childIndent))
		case "orderedList":
			lines = append(lines, r.renderOrderedList(child, childIndent))
		case "codeBlock":
			blockLines := strings.Split(r.renderCodeBlock(child), "\n")
			rest := blockLines
			if first {
				lines = append(lines, indent+prefix+blockLines[0])
				rest = blockLines[1:]
				first = false
			}
			for _, bl := range rest {
				lines = append(lines, childIndent+bl)
			}
		}
	}
	return strings.Join(lines, "\n")
}
```

- [ ] **Step 4: テストが通ることを確認する**

Run: `make test`
Expected: PASS（既存の `TestConvertADF_NestedBulletList` は `- ` 幅=2 のため出力不変で通ること）

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "fix: リスト入れ子のインデントをマーカー幅ベースに変更し番号付きリストの分断を解消

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 4: 隣接する強調runを結合する（B-2）

**Files:**
- Modify: `adfconverter.go`（`renderInlineNodes` 111行目付近、新規ヘルパー追加）
- Test: `adfconverter_test.go`

**Interfaces:**
- Consumes: `wrapDelimiter(text, delimiter string) string`（既存）
- Produces: `delimiterMarks(node ADFNode) ([]string, bool)`、`renderNonDelimiterText(node ADFNode) string`（Task 5 が textColor 適用を追加する）

- [ ] **Step 1: 失敗するテストを書く**

```go
// TestConvertADF_AdjacentEmphasisRuns は同じ強調マークを持つ隣接テキストが1組のデリミタに結合されることを確認する
func TestConvertADF_AdjacentEmphasisRuns(t *testing.T) {
	// 春[strong] は あけ[em] ぼ[textColor+em] の[em] → **春**は*あけぼの*
	adf := adfDoc(`{"type":"paragraph","content":[` +
		`{"type":"text","text":"春","marks":[{"type":"strong"}]},` +
		`{"type":"text","text":"は"},` +
		`{"type":"text","text":"あけ","marks":[{"type":"em"}]},` +
		`{"type":"text","text":"ぼ","marks":[{"type":"textColor","attrs":{"color":"#ffc400"}},{"type":"em"}]},` +
		`{"type":"text","text":"の","marks":[{"type":"em"}]}` +
		`]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "**春**は*あけぼの*"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: テストが失敗することを確認する**

Run: `go test -run TestConvertADF_AdjacentEmphasisRuns ./...`
Expected: FAIL（`*あけ**ぼ**の*` になる）

- [ ] **Step 3: 実装する**

ヘルパーを追加し `renderInlineNodes` を置き換える:

```go
// mdDelimiters はデリミタ系マークの正規順序（先頭が最外側）と対応デリミタ
var mdDelimiters = []struct {
	mark  string
	delim string
}{
	{"strong", "**"},
	{"em", "*"},
	{"strike", "~~"},
}

// delimiterMarks は text ノードが持つデリミタ系マークを正規順序で返す。
// code/link/subsup/underline を含むノードと text 以外のノードはグループ化対象外（ok=false）
func delimiterMarks(node ADFNode) ([]string, bool) {
	if node.Type != "text" {
		return nil, false
	}
	present := map[string]bool{}
	for _, m := range node.Marks {
		switch m.Type {
		case "strong", "em", "strike":
			present[m.Type] = true
		case "code", "link", "subsup", "underline":
			return nil, false
		}
	}
	var marks []string
	for _, d := range mdDelimiters {
		if present[d.mark] {
			marks = append(marks, d.mark)
		}
	}
	return marks, true
}

// renderNonDelimiterText はデリミタ系マーク以外を適用したテキストを返す
// （デリミタ系はグループ単位で renderInlineNodes が適用する）
func (r *adfRenderer) renderNonDelimiterText(node ADFNode) string {
	return node.Text
}

// renderInlineNodes はインライン要素を連結する。
// 同じデリミタ系マークを持つ隣接テキストノードは1つのrunに結合してから囲む
func (r *adfRenderer) renderInlineNodes(nodes []ADFNode) string {
	delimOf := map[string]string{}
	for _, d := range mdDelimiters {
		delimOf[d.mark] = d.delim
	}
	var sb strings.Builder
	for i := 0; i < len(nodes); {
		marks, ok := delimiterMarks(nodes[i])
		if !ok {
			sb.WriteString(r.renderInline(nodes[i]))
			i++
			continue
		}
		sig := strings.Join(marks, ",")
		var group strings.Builder
		j := i
		for j < len(nodes) {
			m2, ok2 := delimiterMarks(nodes[j])
			if !ok2 || strings.Join(m2, ",") != sig {
				break
			}
			group.WriteString(r.renderNonDelimiterText(nodes[j]))
			j++
		}
		text := group.String()
		for k := len(marks) - 1; k >= 0; k-- {
			text = wrapDelimiter(text, delimOf[marks[k]])
		}
		sb.WriteString(text)
		i = j
	}
	return sb.String()
}
```

- [ ] **Step 4: テストが通ることを確認する**

Run: `make test`
Expected: PASS（既存の Bold/Italic/Strikethrough テストは1ノードグループとして同じ出力になる）

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "fix: 隣接する同種強調runを結合しMarkdownデリミタの衝突を解消

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 5: textColor を span で再現する（C-1）

**Files:**
- Modify: `adfconverter.go`（`renderText`、`renderNonDelimiterText`、新規ヘルパー `applyTextColor`）
- Test: `adfconverter_test.go`（既存 `TestConvertADF_TextColorIgnored` を置き換え）

**Interfaces:**
- Produces: `applyTextColor(text string, mark ADFMark) string`。textColor は `<span style="color: #RRGGBB">…</span>` になり、デリミタの内側に置かれる

- [ ] **Step 1: 失敗するテストを書く**

既存の `TestConvertADF_TextColorIgnored`（127行目付近）を削除し、以下に置き換える:

```go
// TestConvertADF_TextColor は文字色が span で保持されることを確認する
func TestConvertADF_TextColor(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","content":[` +
		`{"type":"text","text":"赤い字","marks":[{"type":"textColor","attrs":{"color":"#ff5630"}}]}` +
		`]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `<span style="color: #ff5630">赤い字</span>`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestConvertADF_TextColorInsideEmphasis は色付き文字が強調runの内側で span になることを確認する
func TestConvertADF_TextColorInsideEmphasis(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","content":[` +
		`{"type":"text","text":"あけ","marks":[{"type":"em"}]},` +
		`{"type":"text","text":"ぼ","marks":[{"type":"textColor","attrs":{"color":"#ffc400"}},{"type":"em"}]},` +
		`{"type":"text","text":"の","marks":[{"type":"em"}]}` +
		`]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `*あけ<span style="color: #ffc400">ぼ</span>の*`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

注意: Task 4 のテスト `TestConvertADF_AdjacentEmphasisRuns` は textColor 無視を前提に `*あけぼの*` を期待しているため、期待値を `*あけ<span style="color: #ffc400">ぼ</span>の*` を含む形に更新する:

```go
	want := `**春**は*あけ<span style="color: #ffc400">ぼ</span>の*`
```

- [ ] **Step 2: テストが失敗することを確認する**

Run: `go test -run 'TestConvertADF_TextColor|TestConvertADF_AdjacentEmphasisRuns' ./...`
Expected: FAIL

- [ ] **Step 3: 実装する**

```go
// applyTextColor は textColor マークを span タグに変換する
func applyTextColor(text string, mark ADFMark) string {
	color, _ := mark.Attrs["color"].(string)
	if color == "" {
		return text
	}
	return `<span style="color: ` + color + `">` + text + `</span>`
}
```

`renderNonDelimiterText` を置き換え:

```go
// renderNonDelimiterText はデリミタ系マーク以外を適用したテキストを返す
// （デリミタ系はグループ単位で renderInlineNodes が適用する）
func (r *adfRenderer) renderNonDelimiterText(node ADFNode) string {
	text := node.Text
	for i := len(node.Marks) - 1; i >= 0; i-- {
		if node.Marks[i].Type == "textColor" {
			text = applyTextColor(text, node.Marks[i])
		}
	}
	return text
}
```

`renderText` のマーク適用 switch に case を追加（`// textColor, backgroundColor, annotation はテキストのみ保持` コメント行を置き換え）:

```go
		case "textColor":
			text = applyTextColor(text, mark)
		// backgroundColor, annotation はテキストのみ保持
```

- [ ] **Step 4: テストが通ることを確認する**

Run: `make test`
Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "feat: textColorマークをspanタグで再現

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 6: 段落の alignment を div で再現する（C-2）

**Files:**
- Modify: `adfconverter.go`（`renderNode` の `paragraph` ケース、新規ヘルパー `alignmentStyle`）
- Test: `adfconverter_test.go`

**Interfaces:**
- Produces: `alignmentStyle(node ADFNode) string`（`center`→`"center"`、`end`→`"right"`、それ以外→`""`）

- [ ] **Step 1: 失敗するテストを書く**

```go
// TestConvertADF_AlignmentCenter は中央寄せ段落が div で包まれることを確認する
func TestConvertADF_AlignmentCenter(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","marks":[{"type":"alignment","attrs":{"align":"center"}}],"content":[` + adfText("あ中央あ") + `]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "<div style=\"text-align: center\">\n\nあ中央あ\n\n</div>"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestConvertADF_AlignmentEnd は右寄せ段落が div で包まれることを確認する
func TestConvertADF_AlignmentEnd(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","marks":[{"type":"alignment","attrs":{"align":"end"}}],"content":[` + adfText("右") + `]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "<div style=\"text-align: right\">\n\n右\n\n</div>"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: テストが失敗することを確認する**

Run: `go test -run TestConvertADF_Alignment ./...`
Expected: FAIL（div なしの素のテキストが返る）

- [ ] **Step 3: 実装する**

```go
// alignmentStyle は段落の alignment マークを CSS text-align 値に変換する
func alignmentStyle(node ADFNode) string {
	for _, m := range node.Marks {
		if m.Type != "alignment" || m.Attrs == nil {
			continue
		}
		switch m.Attrs["align"] {
		case "center":
			return "center"
		case "end":
			return "right"
		}
	}
	return ""
}
```

`renderNode` の `paragraph` ケースを置き換え:

```go
	case "paragraph":
		text := r.renderInlineNodes(node.Content)
		if align := alignmentStyle(node); align != "" && text != "" {
			return `<div style="text-align: ` + align + `">` + "\n\n" + text + "\n\n</div>"
		}
		return text
```

- [ ] **Step 4: テストが通ることを確認する**

Run: `make test`
Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "feat: 段落のalignmentマークをdivタグで再現

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 7: コメント投稿者名を表示名に解決する（D）

**Files:**
- Modify: `mdwriter.go`（`MDWriter` 構造体、`NewMDWriter`、コメント出力 94行目付近）
- Modify: `main.go`（`NewMDWriter` 呼び出し3箇所: 144・283・341行目付近）
- Test: `mdwriter_test.go`（`newTestMDWriter` ヘルパーと新規テスト）

**Interfaces:**
- Consumes: `ConfluenceClient.GetUserDisplayName(accountID string, deletedUsers map[string]string) string`（既存・キャッシュ付き）、`cfg.DeletedUsers map[string]string`
- Produces: `NewMDWriter(outputDir string, converter *Converter, resolveUser func(accountID string) string) *MDWriter`（resolveUser は nil 可。nil なら accountId をそのまま出力）

- [ ] **Step 1: 失敗するテストを書く**

`mdwriter_test.go` の `newTestMDWriter` を先に修正（コンパイルを通すため既存呼び出しは nil を渡す）:

```go
func newTestMDWriter(dir string) *MDWriter {
	conv := NewConverter(nil, nil)
	return NewMDWriter(dir, conv, nil)
}
```

新規テストを追加:

```go
// TestMDWriter_WritePage_ResolvesCommentAuthor はコメント投稿者が表示名に解決されることを確認する
func TestMDWriter_WritePage_ResolvesCommentAuthor(t *testing.T) {
	tmpDir := t.TempDir()
	conv := NewConverter(nil, nil)
	writer := NewMDWriter(tmpDir, conv, func(accountID string) string {
		if accountID == "user123" {
			return "山田 太郎"
		}
		return accountID
	})

	page := &Page{
		ID:    "12345",
		Title: "コメント付きページ",
		Body: PageBody{
			Storage: Storage{Value: "<p>本文</p>"},
			AtlasDocFormat: AtlasDocFormat{
				Value:          `{"version":1,"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"本文"}]}]}`,
				Representation: "atlas_doc_format",
			},
		},
		Version: Version{Number: 1, CreatedAt: "2024-01-01T00:00:00.000Z"},
	}
	comments := []Comment{
		{
			ID:      "c001",
			Body:    CommentBody{Storage: Storage{Value: "<p>コメント内容</p>"}},
			Version: Version{CreatedAt: "2024-01-02T00:00:00.000Z", AuthorID: "user123"},
		},
	}

	err := writer.WritePage(page, "TEST", "テストスペース", "", nil, comments, nil)
	if err != nil {
		t.Fatalf("WritePage エラー: %v", err)
	}

	mdPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title), "index.md")
	content, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("ファイル読み込みエラー: %v", err)
	}
	if !strings.Contains(string(content), "**投稿者:** 山田 太郎") {
		t.Errorf("投稿者が表示名に解決されていません\n内容: %q", string(content))
	}
}
```

- [ ] **Step 2: テストが失敗することを確認する**

Run: `go test -run TestMDWriter_WritePage_ResolvesCommentAuthor ./...`
Expected: コンパイルエラー（`NewMDWriter` の引数が2個のため）。これが「失敗」の確認になる

- [ ] **Step 3: 実装する**

`mdwriter.go` の構造体とコンストラクタを置き換え:

```go
type MDWriter struct {
	outputDir   string
	converter   *Converter
	resolveUser func(accountID string) string // accountId → 表示名。nil の場合は解決しない
}

// NewMDWriter は新しいMDWriterを作成する
func NewMDWriter(outputDir string, converter *Converter, resolveUser func(accountID string) string) *MDWriter {
	return &MDWriter{outputDir: outputDir, converter: converter, resolveUser: resolveUser}
}
```

コメント出力（`generateContent` 内 94行目付近）を置き換え:

```go
			authorName := authorID
			if w.resolveUser != nil && authorID != "unknown" {
				authorName = w.resolveUser(authorID)
			}

			sb.WriteString(fmt.Sprintf("### コメント %d\n\n", i+1))
			sb.WriteString(fmt.Sprintf("**投稿者:** %s  \n", authorName))
```

`main.go` の3箇所を修正。`fetchPage`（144行目付近）と `fetchSpace`（283行目付近）は API クライアントで解決:

```go
	writer := NewMDWriter(cfg.Output.MarkdownDir, conv, func(accountID string) string {
		return client.GetUserDisplayName(accountID, cfg.DeletedUsers)
	})
```

`convert` コマンド（341行目付近）はオフライン動作のため deletedUsers マッピングのみで解決:

```go
	writer := NewMDWriter(cfg.Output.MarkdownDir, conv, func(accountID string) string {
		if name, ok := cfg.DeletedUsers[accountID]; ok {
			return name
		}
		return accountID
	})
```

- [ ] **Step 4: テストが通ることを確認する**

Run: `make test`
Expected: PASS（既存の `TestMDWriter_WritePage_WithComments` は resolver=nil なので従来出力のまま通る）

- [ ] **Step 5: コミット**

```bash
git add mdwriter.go mdwriter_test.go main.go
git commit -m "fix: コメント投稿者のaccountIdを表示名に解決（GetUserDisplayNameをワイヤリング）

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 8: 実ページでの検証とドキュメント更新

**Files:**
- Modify: `TODO.md`、`CHANGELOG.md`
- 検証対象: `hugo-site/content/SCRUM/2026-5-13 テスト議事録/index.md`

**Interfaces:**
- Consumes: Task 1〜7 のすべての修正、`./migConfluence page` コマンド、稼働中の Hugo dev サーバ（`http://localhost:1313`）

- [ ] **Step 1: ビルドして実ページを再変換する**

```bash
make build
./migConfluence page --page-id 27656193 --save-intermediate
```

Expected: エラーなく完了し `hugo-site/content/SCRUM/2026-5-13 テスト議事録/index.md` が更新される

- [ ] **Step 2: Markdown出力で7項目の解消を確認する**

`index.md` を読み、以下をすべて確認する:

1. 「bbb」の下にフェンスコードブロック `aaaaaa` がある
2. 「- [ ] チェックボック」の下にインデント付きで「- [ ] チェックボックスインデント」がある
3. 番号付きリストの入れ子「- リスト」が3スペースインデント、三番の下にコードブロック `aaaaa` がある
4. `*あけ<span…>ぼ</span>の*` の形になっている（`**ぼ**` がない）
5. `<span style="color: #ff5630">は</span>` など色 span がある
6. `<div style="text-align: center">` と `<div style="text-align: right">` がある
7. 「**投稿者:** Kenichiro GOZU」のように表示名になっている（accountId `557058:…` が出ない）

- [ ] **Step 3: Hugoレンダリングを確認する**

```bash
curl -s "http://localhost:1313/scrum/2026-5-13-%E3%83%86%E3%82%B9%E3%83%88%E8%AD%B0%E4%BA%8B%E9%8C%B2/" | grep -c "<pre"
```

Expected: リスト内コードブロック2個が `<pre>` として出力される（0 でないこと）。あわせて HTML 内に `<ol>` が分断されていないこと（`<ol start="2">` が消えること）、`text-align: center` が出ることを確認する

- [ ] **Step 4: TODO.md と CHANGELOG.md を更新する**

- TODO.md: 「進行中」の当該項目のチェックを付けて「完了」セクションへ移動する
- CHANGELOG.md: 既存の書式に合わせて今回の修正内容（7項目）を追記する

- [ ] **Step 5: コミット**

```bash
git add TODO.md CHANGELOG.md
git commit -m "docs: TODO.md/CHANGELOG.mdにADF変換の未対応要素修正を記録

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

- [ ] **Step 6: PR方針の確認**

親ブランチ `feature/adf-emphasis-whitespace` が未マージ・PR未作成のため、ユーザーに確認する:
1. 先に `feature/adf-emphasis-whitespace` のPRを作成・マージしてから本ブランチのPRを作成する
2. 本ブランチから1つのPRにまとめる（強調マーク空白修正のコミットも含まれる）
