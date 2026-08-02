package main

import (
	"encoding/json"
	"fmt"
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
		t.Errorf("got %q, want backtick foo backtick", got)
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

// TestConvertADF_TextColorInvalidValue は color 値に不正な文字列（HTML属性突破を狙った値）が
// 与えられた場合に span を生成せず素のテキストを返すことを確認する
func TestConvertADF_TextColorInvalidValue(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","content":[` +
		`{"type":"text","text":"危険","marks":[{"type":"textColor","attrs":{"color":"red\"><script>alert(1)</script>"}}]}` +
		`]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "危険"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConvertADF_InternalLink(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"page","marks":[{"type":"link","attrs":{"href":"https://example.atlassian.net/wiki/spaces/KEY/pages/12345/My%20Page"}}]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "/index.md") {
		t.Errorf("got %q, internal link should be converted to relative path", got)
	}
}

// adfDoc と adfText は後続タスクのテストでも使われるため、このファイルで宣言する
var _ = fmt.Sprintf // suppress unused import warning

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

// TestConvertADF_TableCellAlignmentNoDiv はテーブルセル内の alignment マークが
// 段落と同様の div ラップ対象にならない（列の GFM アライメントのみで表現される）ことを確認する
func TestConvertADF_TableCellAlignmentNoDiv(t *testing.T) {
	adf := adfDoc(`{"type":"table","content":[
        {"type":"tableRow","content":[
            {"type":"tableCell","content":[{"type":"paragraph","marks":[{"type":"alignment","attrs":{"align":"center"}}],"content":[{"type":"text","text":"Centered"}]}]}
        ]}
    ]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(got, "<div") {
		t.Errorf("got %q, table cell content should not contain div for alignment", got)
	}
	if !strings.Contains(got, ":---:") {
		t.Errorf("got %q, want centered column alignment separator", got)
	}
	if !strings.Contains(got, "| Centered |") {
		t.Errorf("got %q, want cell content", got)
	}
}

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
	want := "<table><thead><tr><th>内側H</th></tr></thead><tbody><tr><td>入れ子</td></tr></tbody></table>"
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want to contain %q", got, want)
	}
	if strings.Contains(got, "\n<table>") {
		t.Errorf("got %q, nested table must be inline (no leading newline)", got)
	}
}

func TestConvertADF_TableNestedTableTheadTbody(t *testing.T) {
	// thead/tbody を省略すると、ブラウザが全行を1つの tbody にまとめてしまい、
	// ゼブラストライプ用CSS（tr:nth-child(2n)）がヘッダー行を数に含めてしまうため、
	// データ行の縞模様がヘッダー行1つ分ずれて誤って着色される（実際に発生した不具合）。
	// 外側テーブル（renderTable）と同様に thead/tbody で構造を分離し、
	// ブラウザによる暗黙のtbody統合を防ぐ。
	inner := `{"type":"doc","content":[{"type":"table","content":[
        {"type":"tableRow","content":[
            {"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"H1"}]}]},
            {"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"H2"}]}]}
        ]},
        {"type":"tableRow","content":[
            {"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"A"}]}]},
            {"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"B"}]}]}
        ]},
        {"type":"tableRow","content":[
            {"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"C"}]}]},
            {"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"D"}]}]}
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
	want := "<table><thead><tr><th>H1</th><th>H2</th></tr></thead>" +
		"<tbody><tr><td>A</td><td>B</td></tr><tr><td>C</td><td>D</td></tr></tbody></table>"
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want to contain %q", got, want)
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

func TestConvertADF_BoldTrailingSpace(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"hello ","marks":[{"type":"strong"}]},{"type":"text","text":"world"}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "**hello** world") {
		t.Errorf("got %q, want to contain %q", got, "**hello** world")
	}
	if strings.Contains(got, "hello **") {
		t.Errorf("got %q, closing delimiter must not be preceded by a space", got)
	}
}

func TestConvertADF_ItalicLeadingSpace(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"hello "},{"type":"text","text":" hi","marks":[{"type":"em"}]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "hello  *hi*") {
		t.Errorf("got %q, want to contain %q", got, "hello  *hi*")
	}
	if strings.Contains(got, "* hi") {
		t.Errorf("got %q, opening delimiter must not be followed by a space", got)
	}
}

func TestConvertADF_StrikethroughSurroundingSpace(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"text "},{"type":"text","text":" del ","marks":[{"type":"strike"}]},{"type":"text","text":" more"}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, " ~~del~~ ") {
		t.Errorf("got %q, want to contain %q", got, " ~~del~~ ")
	}
}

func TestConvertADF_BoldWhitespaceOnly(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"   ","marks":[{"type":"strong"}]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(got, "**") {
		t.Errorf("got %q, whitespace-only text must not be wrapped in delimiters", got)
	}
}

func TestConvertADF_BoldItalicOverlapWithSpaces(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"prefix "},{"type":"text","text":" foo ","marks":[{"type":"em"},{"type":"strong"}]},{"type":"text","text":" suffix"}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, " ***foo*** ") {
		t.Errorf("got %q, want to contain %q", got, " ***foo*** ")
	}
}

func TestConvertADF_PanelHeadingBoldTrailingSpace(t *testing.T) {
	adf := adfDoc(`{"type":"panel","attrs":{"panelType":"info"},"content":[{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"新しいスペースへようこそ! ","marks":[{"type":"strong"}]}]},{"type":"paragraph","content":[{"type":"text","text":"content"}]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "**新しいスペースへようこそ!** ") {
		t.Errorf("got %q, want NOTE panel heading to render as closed bold", got)
	}
	if strings.Contains(got, "! **") {
		t.Errorf("got %q, regression: literal ** must not appear (original bug)", got)
	}
}

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
	want := `**春**は*あけ<span style="color: #ffc400">ぼ</span>の*`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

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
