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
