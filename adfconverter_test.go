package main

import (
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
