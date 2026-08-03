package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestMDWriter(dir string) *MDWriter {
	conv := NewConverter(nil, nil)
	return NewMDWriter(dir, conv, nil)
}

// TestMDWriter_WritePage はWritePageのテスト
func TestMDWriter_WritePage(t *testing.T) {
	tmpDir := t.TempDir()
	writer := newTestMDWriter(tmpDir)

	page := &Page{
		ID:      "12345",
		Title:   "テストページ",
		Status:  "current",
		SpaceID: "67890",
		Body: PageBody{
			Storage: Storage{
				Value:          "<p>テストコンテンツ</p>",
				Representation: "storage",
			},
			AtlasDocFormat: AtlasDocFormat{
				Value:          `{"version":1,"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"テストコンテンツ"}]}]}`,
				Representation: "atlas_doc_format",
			},
		},
		Version: Version{
			Number:    1,
			CreatedAt: "2024-01-01T00:00:00.000Z",
		},
		Links: Links{
			WebUI: "/wiki/spaces/TEST/pages/12345",
		},
	}

	labels := []Label{
		{Name: "golang"},
		{Name: "backend"},
	}

	err := writer.WritePage(page, "TEST", "テストスペース", "", labels, nil, nil)
	if err != nil {
		t.Fatalf("WritePage エラー: %v", err)
	}

	// ファイルが生成されたか確認
	mdPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title), "index.md")
	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		t.Errorf("Markdownファイルが生成されていません: %s", mdPath)
		return
	}

	// ファイルの内容を確認
	content, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("ファイル読み込みエラー: %v", err)
	}
	contentStr := string(content)

	// Front Matterの確認
	if !strings.Contains(contentStr, `title = "テストページ"`) {
		t.Errorf("titleがFront Matterに含まれていません\n内容: %q", contentStr)
	}
	if !strings.Contains(contentStr, `space = "TEST"`) {
		t.Errorf("spaceがFront Matterに含まれていません\n内容: %q", contentStr)
	}
	if !strings.Contains(contentStr, `page_id = "12345"`) {
		t.Errorf("page_idがFront Matterに含まれていません\n内容: %q", contentStr)
	}
	if !strings.Contains(contentStr, `"golang"`) {
		t.Errorf("ラベルがFront Matterに含まれていません\n内容: %q", contentStr)
	}

	// 本文の確認
	if !strings.Contains(contentStr, "テストコンテンツ") {
		t.Errorf("ページ本文が含まれていません\n内容: %q", contentStr)
	}
}

// TestMDWriter_WritePage_WithComments はコメント付きページのテスト
func TestMDWriter_WritePage_WithComments(t *testing.T) {
	tmpDir := t.TempDir()
	writer := newTestMDWriter(tmpDir)

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
		Version: Version{
			Number:    1,
			CreatedAt: "2024-01-01T00:00:00.000Z",
		},
	}

	comments := []Comment{
		{
			ID: "c001",
			Body: CommentBody{
				Storage: Storage{Value: "<p>コメント内容</p>"},
			},
			Version: Version{
				CreatedAt: "2024-01-02T00:00:00.000Z",
				AuthorID:  "user123",
			},
		},
	}

	err := writer.WritePage(page, "TEST", "テストスペース", "親ページ", nil, comments, nil)
	if err != nil {
		t.Fatalf("WritePage エラー: %v", err)
	}

	mdPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title), "index.md")
	content, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("ファイル読み込みエラー: %v", err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, "## コメント") {
		t.Errorf("コメントセクションが含まれていません\n内容: %q", contentStr)
	}
	if !strings.Contains(contentStr, "コメント内容") {
		t.Errorf("コメント本文が含まれていません\n内容: %q", contentStr)
	}

	// 親ページ
	if !strings.Contains(contentStr, `parent = "親ページ"`) {
		t.Errorf("親ページがFront Matterに含まれていません\n内容: %q", contentStr)
	}
}

// TestMDWriter_WritePage_WithAttachments は添付ファイル付きページのテスト
func TestMDWriter_WritePage_WithAttachments(t *testing.T) {
	tmpDir := t.TempDir()
	writer := newTestMDWriter(tmpDir)

	page := &Page{
		ID:    "12345",
		Title: "添付ファイル付きページ",
		Body: PageBody{
			Storage: Storage{Value: "<p>本文</p>"},
			AtlasDocFormat: AtlasDocFormat{
				Value:          `{"version":1,"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"本文"}]}]}`,
				Representation: "atlas_doc_format",
			},
		},
		Version: Version{
			Number:    1,
			CreatedAt: "2024-01-01T00:00:00.000Z",
		},
	}

	attachments := []Attachment{
		{ID: "a1", Title: "document.pdf", PageID: "12345"},
		{ID: "a2", Title: "image.png", PageID: "12345"},
	}

	err := writer.WritePage(page, "TEST", "テストスペース", "", nil, nil, attachments)
	if err != nil {
		t.Fatalf("WritePage エラー: %v", err)
	}

	mdPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title), "index.md")
	content, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("ファイル読み込みエラー: %v", err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, "## 添付ファイル") {
		t.Errorf("添付ファイルセクションが含まれていません\n内容: %q", contentStr)
	}
	if !strings.Contains(contentStr, "document.pdf") {
		t.Errorf("PDFファイルが含まれていません\n内容: %q", contentStr)
	}
	if !strings.Contains(contentStr, "image.png") {
		t.Errorf("画像ファイルが含まれていません\n内容: %q", contentStr)
	}
}

// TestFormatDate は日付フォーマットのテスト
func TestFormatDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2024-01-15T10:30:00.000Z", "2024-01-15 10:30:00"},
		{"2024-12-31T23:59:59Z", "2024-12-31 23:59:59"},
		{"", ""},
	}

	for _, tt := range tests {
		result := formatDate(tt.input)
		if result != tt.expected {
			t.Errorf("formatDate(%q) = %q, 期待: %q", tt.input, result, tt.expected)
		}
	}
}

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

// TestMDWriter_WritePage_WithCommentReplies は返信コメント（子コメント）の見出し階層のテスト
func TestMDWriter_WritePage_WithCommentReplies(t *testing.T) {
	tmpDir := t.TempDir()
	writer := newTestMDWriter(tmpDir)

	page := &Page{
		ID:    "12345",
		Title: "返信付きページ",
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
			Depth:   0,
			Body:    CommentBody{Storage: Storage{Value: "<p>親コメント</p>"}},
			Version: Version{CreatedAt: "2024-01-02T00:00:00.000Z", AuthorID: "user123"},
		},
		{
			ID:      "c002",
			Depth:   1,
			Body:    CommentBody{Storage: Storage{Value: "<p>返信コメント</p>"}},
			Version: Version{CreatedAt: "2024-01-03T00:00:00.000Z", AuthorID: "user456"},
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
	contentStr := string(content)

	// トップレベルコメント（Depth=0）は divなし、HTML タグで出力
	if !strings.Contains(contentStr, "<h3>コメント 1</h3>") {
		t.Errorf("親コメントの見出しが期待と異なります（HTML タグでない）\n内容: %q", contentStr)
	}
	// Markdown見出し ### が含まれないこと（目次から除外）
	if strings.Contains(contentStr, "### コメント 1") {
		t.Errorf("親コメントが Markdown 見出しで出力されています（目次から除外されません）\n内容: %q", contentStr)
	}

	// 返信コメント（Depth=1）は div でインデント
	// <div style="margin-left: 2em">
	// <h4>コメント 1-1</h4>
	// ... 本文 ...
	// </div>
	if !strings.Contains(contentStr, "<div style=\"margin-left: 2em\">") {
		t.Errorf("返信コメントが div でインデントされていません\n内容: %q", contentStr)
	}
	if !strings.Contains(contentStr, "<h4>コメント 1-1</h4>") {
		t.Errorf("返信コメントの見出しが期待と異なります（HTML タグでない）\n内容: %q", contentStr)
	}
	// Markdown見出し #### が含まれないこと（目次から除外）
	if strings.Contains(contentStr, "#### コメント 1-1") {
		t.Errorf("返信コメントが Markdown 見出しで出力されています（目次から除外されません）\n内容: %q", contentStr)
	}
	if !strings.Contains(contentStr, "返信コメント") {
		t.Errorf("返信コメントの本文が含まれていません\n内容: %q", contentStr)
	}
	if !strings.Contains(contentStr, "</div>") {
		t.Errorf("返信コメント div のクローズタグが含まれていません\n内容: %q", contentStr)
	}
}
