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
	// warning → CAUTION (GFM Alerts形式)
	if !strings.Contains(result, "CAUTION") {
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
	// GFMタスクリスト形式 - [x] / - [ ] で変換されていることを確認
	if !strings.Contains(result, "- [x]") {
		t.Errorf("完了タスクが - [x] 形式になっていません。結果: %q", result)
	}
	if !strings.Contains(result, "- [ ]") {
		t.Errorf("未完了タスクが - [ ] 形式になっていません。結果: %q", result)
	}
	if !strings.Contains(result, "完了したタスク") {
		t.Errorf("完了タスクのボディが変換されていません。結果: %q", result)
	}
	if !strings.Contains(result, "未完了のタスク") {
		t.Errorf("未完了タスクのボディが変換されていません。結果: %q", result)
	}
}

// TestConvert_InfoMacroGFMAlert はinfoマクロのGFM Alert形式テスト
func TestConvert_InfoMacroGFMAlert(t *testing.T) {
	c := newTestConverter()
	input := `<ac:structured-macro ac:name="info">
<ac:parameter ac:name="title">重要な情報</ac:parameter>
<ac:rich-text-body><p>詳細内容</p></ac:rich-text-body>
</ac:structured-macro>`

	result, err := c.Convert(input)
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}
	if !strings.Contains(result, "> [!NOTE]") {
		t.Errorf("GFM Alert形式になっていません。結果: %q", result)
	}
	if !strings.Contains(result, "重要な情報") {
		t.Errorf("タイトルが含まれていません。結果: %q", result)
	}
}

// TestConvert_NewMacros は追加マクロの変換テスト
func TestConvert_NewMacros(t *testing.T) {
	c := newTestConverter()

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			"quote",
			`<ac:structured-macro ac:name="quote"><ac:rich-text-body><p>引用テキスト</p></ac:rich-text-body></ac:structured-macro>`,
			"引用テキスト",
		},
		{
			"noformat",
			`<ac:structured-macro ac:name="noformat"><ac:plain-text-body><![CDATA[整形なし]]></ac:plain-text-body></ac:structured-macro>`,
			"整形なし",
		},
		{
			"status-color",
			`<ac:structured-macro ac:name="status"><ac:parameter ac:name="colour">Red</ac:parameter><ac:parameter ac:name="title">障害</ac:parameter></ac:structured-macro>`,
			"障害",
		},
		{
			"status-green",
			`<ac:structured-macro ac:name="status"><ac:parameter ac:name="colour">Green</ac:parameter><ac:parameter ac:name="title">完了</ac:parameter></ac:structured-macro>`,
			"完了",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.Convert(tt.input)
			if err != nil {
				t.Errorf("変換エラー: %v", err)
				return
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("期待する内容が含まれていません\n期待: %q\n実際: %q", tt.contains, result)
			}
		})
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
