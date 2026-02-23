package main

import (
	"strings"
	"testing"
)

func newTestConverter() *Converter {
	return NewConverter(nil, nil)
}

// TestConvert_Empty は空文字列の変換テスト
func TestConvert_Empty(t *testing.T) {
	c := newTestConverter()
	result, err := c.Convert("")
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}
	if result != "" {
		t.Errorf("期待: %q, 実際: %q", "", result)
	}
}

// TestConvert_Paragraph は段落の変換テスト
func TestConvert_Paragraph(t *testing.T) {
	c := newTestConverter()
	result, err := c.Convert("<p>Hello, World!</p>")
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}
	if !strings.Contains(result, "Hello, World!") {
		t.Errorf("段落が変換されていません。結果: %q", result)
	}
}

// TestConvert_Heading は見出しの変換テスト
func TestConvert_Heading(t *testing.T) {
	c := newTestConverter()

	tests := []struct {
		input    string
		contains string
	}{
		{"<h1>タイトル</h1>", "# タイトル"},
		{"<h2>セクション</h2>", "## セクション"},
		{"<h3>サブセクション</h3>", "### サブセクション"},
	}

	for _, tt := range tests {
		result, err := c.Convert(tt.input)
		if err != nil {
			t.Errorf("変換エラー: %v", err)
			continue
		}
		if !strings.Contains(result, tt.contains) {
			t.Errorf("見出しが変換されていません\n入力: %q\n期待を含む: %q\n実際: %q", tt.input, tt.contains, result)
		}
	}
}

// TestConvert_Bold は太字の変換テスト
func TestConvert_Bold(t *testing.T) {
	c := newTestConverter()
	result, err := c.Convert("<p><strong>太字テキスト</strong></p>")
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}
	if !strings.Contains(result, "**太字テキスト**") {
		t.Errorf("太字が変換されていません。結果: %q", result)
	}
}

// TestConvert_CodeMacro はコードブロックマクロの変換テスト
func TestConvert_CodeMacro(t *testing.T) {
	c := newTestConverter()
	input := `<ac:structured-macro ac:name="code" ac:schema-version="1">
<ac:parameter ac:name="language">go</ac:parameter>
<ac:plain-text-body><![CDATA[func main() {
    fmt.Println("Hello")
}]]></ac:plain-text-body>
</ac:structured-macro>`

	result, err := c.Convert(input)
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}
	if !strings.Contains(result, "```") {
		t.Errorf("コードブロックが変換されていません。結果: %q", result)
	}
}

// TestConvert_InfoMacro はinfoマクロの変換テスト
func TestConvert_InfoMacro(t *testing.T) {
	c := newTestConverter()
	input := `<ac:structured-macro ac:name="info">
<ac:parameter ac:name="title">情報タイトル</ac:parameter>
<ac:rich-text-body><p>情報コンテンツ</p></ac:rich-text-body>
</ac:structured-macro>`

	result, err := c.Convert(input)
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}
	if !strings.Contains(result, "NOTE") {
		t.Errorf("infoマクロが変換されていません。結果: %q", result)
	}
}

// TestConvert_WarningMacro はwarningマクロの変換テスト
func TestConvert_WarningMacro(t *testing.T) {
	c := newTestConverter()
	input := `<ac:structured-macro ac:name="warning">
<ac:rich-text-body><p>警告コンテンツ</p></ac:rich-text-body>
</ac:structured-macro>`

	result, err := c.Convert(input)
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}
	if !strings.Contains(result, "WARNING") {
		t.Errorf("warningマクロが変換されていません。結果: %q", result)
	}
}

// TestConvert_ExpandMacro はexpandマクロの変換テスト
func TestConvert_ExpandMacro(t *testing.T) {
	c := newTestConverter()
	input := `<ac:structured-macro ac:name="expand">
<ac:parameter ac:name="title">クリックで展開</ac:parameter>
<ac:rich-text-body><p>隠されたコンテンツ</p></ac:rich-text-body>
</ac:structured-macro>`

	result, err := c.Convert(input)
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}
	// タイトルと本文コンテンツが含まれていることを確認
	if !strings.Contains(result, "クリックで展開") {
		t.Errorf("expandマクロのタイトルが変換されていません。結果: %q", result)
	}
	if !strings.Contains(result, "隠されたコンテンツ") {
		t.Errorf("expandマクロの本文が変換されていません。結果: %q", result)
	}
}

// TestConvert_Image は画像の変換テスト
func TestConvert_Image(t *testing.T) {
	c := newTestConverter()

	// 添付ファイル画像
	input := `<ac:image ac:alt="説明">
<ri:attachment ri:filename="image.png" />
</ac:image>`

	result, err := c.Convert(input)
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}
	if !strings.Contains(result, "image.png") {
		t.Errorf("画像が変換されていません。結果: %q", result)
	}
}

// TestConvert_TaskList はタスクリストの変換テスト
func TestConvert_TaskList(t *testing.T) {
	c := newTestConverter()
	input := `<ac:task-list>
<ac:task>
<ac:task-id>1</ac:task-id>
<ac:task-status>complete</ac:task-status>
<ac:task-body>完了したタスク</ac:task-body>
</ac:task>
<ac:task>
<ac:task-id>2</ac:task-id>
<ac:task-status>incomplete</ac:task-status>
<ac:task-body>未完了のタスク</ac:task-body>
</ac:task>
</ac:task-list>`

	result, err := c.Convert(input)
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}
	// タスクの内容が含まれていることを確認
	if !strings.Contains(result, "完了したタスク") {
		t.Errorf("完了タスクのボディが変換されていません。結果: %q", result)
	}
	if !strings.Contains(result, "未完了のタスク") {
		t.Errorf("未完了タスクのボディが変換されていません。結果: %q", result)
	}
}

// TestConvert_IgnoredMacro は無視マクロのテスト
func TestConvert_IgnoredMacro(t *testing.T) {
	c := NewConverter([]string{"toc"}, nil)
	input := `<ac:structured-macro ac:name="toc"></ac:structured-macro>`

	result, err := c.Convert(input)
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}
	if strings.Contains(result, "toc") {
		t.Errorf("無視するマクロが出力されています。結果: %q", result)
	}
}

// TestConvert_Emoticon は絵文字の変換テスト
func TestConvert_Emoticon(t *testing.T) {
	c := newTestConverter()

	tests := []struct {
		input    string
		contains string
	}{
		{`<ac:emoticon ac:name="tick"/>`, "✅"},
		{`<ac:emoticon ac:name="cross"/>`, "❌"},
		{`<ac:emoticon ac:name="warning"/>`, "⚠️"},
	}

	for _, tt := range tests {
		result, err := c.Convert(tt.input)
		if err != nil {
			t.Errorf("変換エラー: %v", err)
			continue
		}
		if !strings.Contains(result, tt.contains) {
			t.Errorf("絵文字が変換されていません\n入力: %q\n期待を含む: %q\n実際: %q",
				tt.input, tt.contains, result)
		}
	}
}

// TestConvert_Table はテーブルの変換テスト
func TestConvert_Table(t *testing.T) {
	c := newTestConverter()
	input := `<table>
<thead>
<tr><th>列1</th><th>列2</th></tr>
</thead>
<tbody>
<tr><td>値1</td><td>値2</td></tr>
</tbody>
</table>`

	result, err := c.Convert(input)
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}
	if !strings.Contains(result, "列1") || !strings.Contains(result, "値1") {
		t.Errorf("テーブルが変換されていません。結果: %q", result)
	}
}

// TestEmoticonToEmoji はemoticonToEmojiのテスト
func TestEmoticonToEmoji(t *testing.T) {
	tests := []struct {
		name  string
		emoji string
	}{
		{"tick", "✅"},
		{"cross", "❌"},
		{"warning", "⚠️"},
		{"unknown-emoji", ":unknown-emoji:"},
	}

	for _, tt := range tests {
		result := emoticonToEmoji(tt.name)
		if result != tt.emoji {
			t.Errorf("emoticonToEmoji(%q) = %q, 期待: %q", tt.name, result, tt.emoji)
		}
	}
}
