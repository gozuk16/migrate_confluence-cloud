package main

import (
	"net/url"
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

func TestFrontMatterWeight(t *testing.T) {
	tests := []struct {
		name     string
		position *int
		want     int
	}{
		{name: "position=0 は weight=1", position: intPtr(0), want: 1},
		{name: "position=5 は weight=6", position: intPtr(5), want: 6},
		{name: "position=nil は weight=9999", position: nil, want: 9999},
		{name: "負のpositionでも1以上", position: intPtr(-3), want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := frontMatterWeight(tt.position); got != tt.want {
				t.Errorf("frontMatterWeight() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMDWriter_FrontMatterHierarchy(t *testing.T) {
	t.Run("parent_idとweightが出力される", func(t *testing.T) {
		tmpDir := t.TempDir()
		writer := newTestMDWriter(tmpDir)

		page := &Page{
			ID:         "12345",
			Title:      "子ページ",
			SpaceID:    "67890",
			ParentID:   "555",
			ParentType: "folder",
			Position:   intPtr(2),
			Body: PageBody{
				AtlasDocFormat: AtlasDocFormat{
					Value: `{"version":1,"type":"doc","content":[]}`,
				},
			},
		}

		if err := writer.WritePage(page, "TEST", "テストスペース", "親フォルダ", nil, nil, nil); err != nil {
			t.Fatalf("WritePage エラー: %v", err)
		}

		mdPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title), "index.md")
		data, err := os.ReadFile(mdPath)
		if err != nil {
			t.Fatalf("読み込みエラー: %v", err)
		}
		content := string(data)

		for _, want := range []string{
			`parent_id = "555"`,
			`weight = 3`,
			`parent = "親フォルダ"`, // 既存キーは互換のため残す
		} {
			if !strings.Contains(content, want) {
				t.Errorf("フロントマターに %q が含まれていません:\n%s", want, content)
			}
		}
	})

	t.Run("親がない場合はparent_idを出力しない", func(t *testing.T) {
		tmpDir := t.TempDir()
		writer := newTestMDWriter(tmpDir)

		page := &Page{
			ID:      "1",
			Title:   "ルートページ",
			SpaceID: "67890",
			Body: PageBody{
				AtlasDocFormat: AtlasDocFormat{Value: `{"version":1,"type":"doc","content":[]}`},
			},
		}

		if err := writer.WritePage(page, "TEST", "テストスペース", "", nil, nil, nil); err != nil {
			t.Fatalf("WritePage エラー: %v", err)
		}

		mdPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title), "index.md")
		data, err := os.ReadFile(mdPath)
		if err != nil {
			t.Fatalf("読み込みエラー: %v", err)
		}
		if strings.Contains(string(data), "parent_id") {
			t.Errorf("parent_id が出力されています:\n%s", string(data))
		}
		if !strings.Contains(string(data), "weight = 9999") {
			t.Errorf("position未設定時の weight = 9999 が出力されていません:\n%s", string(data))
		}
	})
}

func TestMDWriter_WriteFolder(t *testing.T) {
	t.Run("フォルダスタブが出力される", func(t *testing.T) {
		tmpDir := t.TempDir()
		writer := newTestMDWriter(tmpDir)

		folder := &Folder{
			ID:         "555",
			Title:      "設計ドキュメント",
			ParentID:   "1",
			ParentType: "page",
			Position:   intPtr(2),
		}

		if err := writer.WriteFolder(folder, "TEST", "テストスペース"); err != nil {
			t.Fatalf("WriteFolder エラー: %v", err)
		}

		mdPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(folder.Title), "index.md")
		data, err := os.ReadFile(mdPath)
		if err != nil {
			t.Fatalf("読み込みエラー: %v", err)
		}
		content := string(data)

		for _, want := range []string{
			`title = "設計ドキュメント"`,
			`page_id = "555"`,
			`parent_id = "1"`,
			`weight = 3`,
			`is_folder = true`,
			`space = "TEST"`,
			`space_title = "テストスペース"`,
			`[build]`,
			`render = "never"`,
			`list = "always"`,
		} {
			if !strings.Contains(content, want) {
				t.Errorf("スタブに %q が含まれていません:\n%s", want, content)
			}
		}
	})

	t.Run("親がない場合はparent_idを出力しない", func(t *testing.T) {
		tmpDir := t.TempDir()
		writer := newTestMDWriter(tmpDir)

		folder := &Folder{ID: "555", Title: "ルートフォルダ"}

		if err := writer.WriteFolder(folder, "TEST", ""); err != nil {
			t.Fatalf("WriteFolder エラー: %v", err)
		}

		mdPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(folder.Title), "index.md")
		data, err := os.ReadFile(mdPath)
		if err != nil {
			t.Fatalf("読み込みエラー: %v", err)
		}
		if strings.Contains(string(data), "parent_id") {
			t.Errorf("parent_id が出力されています:\n%s", string(data))
		}
	})

	t.Run("同名ページが既にある場合はID付きディレクトリにフォールバックする", func(t *testing.T) {
		tmpDir := t.TempDir()
		writer := newTestMDWriter(tmpDir)

		// 先に同名のページを書く
		page := &Page{
			ID:      "1",
			Title:   "重複名",
			SpaceID: "67890",
			Body: PageBody{
				AtlasDocFormat: AtlasDocFormat{Value: `{"version":1,"type":"doc","content":[]}`},
			},
		}
		if err := writer.WritePage(page, "TEST", "テストスペース", "", nil, nil, nil); err != nil {
			t.Fatalf("WritePage エラー: %v", err)
		}

		folder := &Folder{ID: "555", Title: "重複名"}
		if err := writer.WriteFolder(folder, "TEST", "テストスペース"); err != nil {
			t.Fatalf("WriteFolder エラー: %v", err)
		}

		// ページ側は上書きされていない
		pageData, err := os.ReadFile(filepath.Join(tmpDir, "TEST", "重複名", "index.md"))
		if err != nil {
			t.Fatalf("ページ読み込みエラー: %v", err)
		}
		if strings.Contains(string(pageData), "is_folder") {
			t.Errorf("ページがフォルダスタブに上書きされました:\n%s", string(pageData))
		}

		// フォルダはID付きディレクトリに出ている
		folderData, err := os.ReadFile(filepath.Join(tmpDir, "TEST", "重複名_555", "index.md"))
		if err != nil {
			t.Fatalf("フォルダスタブが見つかりません: %v", err)
		}
		if !strings.Contains(string(folderData), "is_folder = true") {
			t.Errorf("フォルダスタブの内容が不正です:\n%s", string(folderData))
		}
	})

	t.Run("同じフォルダを2回書いても同じディレクトリを使う", func(t *testing.T) {
		tmpDir := t.TempDir()
		writer := newTestMDWriter(tmpDir)

		folder := &Folder{ID: "555", Title: "再実行フォルダ"}
		if err := writer.WriteFolder(folder, "TEST", ""); err != nil {
			t.Fatalf("1回目 WriteFolder エラー: %v", err)
		}
		if err := writer.WriteFolder(folder, "TEST", ""); err != nil {
			t.Fatalf("2回目 WriteFolder エラー: %v", err)
		}

		if _, err := os.Stat(filepath.Join(tmpDir, "TEST", "再実行フォルダ_555")); err == nil {
			t.Error("2回目の実行でID付きディレクトリが作られました（冪等ではありません）")
		}
	})

	t.Run("ページ本文中のpage_idは誤認されない", func(t *testing.T) {
		tmpDir := t.TempDir()
		writer := newTestMDWriter(tmpDir)

		// ページの本文中に page_id = "555" が含まれているページを先に書く
		page := &Page{
			ID:      "1",
			Title:   "技術解説ページ",
			SpaceID: "67890",
			Body: PageBody{
				AtlasDocFormat: AtlasDocFormat{
					Value: `{"version":1,"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"このページではpage_id = \"555\"について説明します"}]}]}`,
				},
			},
		}
		if err := writer.WritePage(page, "TEST", "", "", nil, nil, nil); err != nil {
			t.Fatalf("WritePage エラー: %v", err)
		}

		// 同じタイトルのフォルダ（別ID）を書く
		folder := &Folder{ID: "555", Title: "技術解説ページ"}
		if err := writer.WriteFolder(folder, "TEST", ""); err != nil {
			t.Fatalf("WriteFolder エラー: %v", err)
		}

		// ページが上書きされていないことを確認（フロントマターに is_folder がない）
		pageData, err := os.ReadFile(filepath.Join(tmpDir, "TEST", "技術解説ページ", "index.md"))
		if err != nil {
			t.Fatalf("ページ読み込みエラー: %v", err)
		}
		if strings.Contains(string(pageData), "is_folder") {
			t.Errorf("ページがフォルダスタブに上書きされました:\n%s", string(pageData))
		}

		// フォルダはID付きディレクトリに出ている
		folderData, err := os.ReadFile(filepath.Join(tmpDir, "TEST", "技術解説ページ_555", "index.md"))
		if err != nil {
			t.Fatalf("フォルダスタブが見つかりません: %v", err)
		}
		if !strings.Contains(string(folderData), "is_folder = true") {
			t.Errorf("フォルダスタブの内容が不正です:\n%s", string(folderData))
		}
	})

	t.Run("フォールバック先にも別コンテンツがある場合はエラーを返す", func(t *testing.T) {
		tmpDir := t.TempDir()
		writer := newTestMDWriter(tmpDir)

		// プライマリディレクトリにページを作成
		page1 := &Page{
			ID:      "1",
			Title:   "同名ページ",
			SpaceID: "67890",
			Body:    PageBody{AtlasDocFormat: AtlasDocFormat{Value: `{"version":1,"type":"doc","content":[]}`}},
		}
		if err := writer.WritePage(page1, "TEST", "", "", nil, nil, nil); err != nil {
			t.Fatalf("WritePage エラー: %v", err)
		}

		// フォールバックディレクトリにも別のページを作成
		if err := os.MkdirAll(filepath.Join(tmpDir, "TEST", "同名ページ_555"), 0755); err != nil {
			t.Fatalf("フォールバックディレクトリ作成エラー: %v", err)
		}
		fallbackPath := filepath.Join(tmpDir, "TEST", "同名ページ_555", "index.md")
		if err := os.WriteFile(fallbackPath, []byte("+++\ntitle = \"別の内容\"\n+++\n"), 0644); err != nil {
			t.Fatalf("フォールバック先への書き込みエラー: %v", err)
		}

		// フォルダを書こうとする
		folder := &Folder{ID: "555", Title: "同名ページ"}
		err := writer.WriteFolder(folder, "TEST", "")

		// エラーが返されることを確認
		if err == nil {
			t.Error("WriteFolder がエラーを返すべきですが、成功してしまいました")
		}

		// プライマリとフォールバック先の両方のコンテンツが保護されていることを確認
		page1Data, _ := os.ReadFile(filepath.Join(tmpDir, "TEST", "同名ページ", "index.md"))
		if strings.Contains(string(page1Data), "is_folder") {
			t.Errorf("プライマリがフォルダスタブに上書きされました:\n%s", string(page1Data))
		}

		fallbackData, _ := os.ReadFile(fallbackPath)
		if strings.Contains(string(fallbackData), "is_folder") {
			t.Errorf("フォールバックがフォルダスタブに上書きされました:\n%s", string(fallbackData))
		}
	})
}

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

func TestMDWriter_WritesRoadmapSVG(t *testing.T) {
	tmpDir := t.TempDir()
	writer := newTestMDWriter(tmpDir)
	src := `{"title":"R","timeline":{"startDate":"2026-10-01 00:00:00","endDate":"2026-12-01 00:00:00","displayOption":"MONTH"},"lanes":[],"markers":[]}`
	adf := `{"version":1,"type":"doc","content":[{"type":"extension","attrs":{"extensionKey":"roadmap","parameters":{"macroParams":{"source":{"value":"` +
		url.PathEscape(src) + `"}}}}}]}`
	page := &Page{ID: "1", Title: "ロードマップ", SpaceID: "1", Body: PageBody{AtlasDocFormat: AtlasDocFormat{Value: adf}}}

	pageDir := filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title))
	if err := os.MkdirAll(pageDir, 0755); err != nil {
		t.Fatal(err)
	}
	// 前回の変換で残った SVG は消える
	if err := os.WriteFile(filepath.Join(pageDir, "confluence-roadmap-9.svg"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := writer.WritePage(page, "TEST", "", "", nil, nil, nil); err != nil {
		t.Fatalf("WritePage エラー: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(pageDir, "confluence-roadmap-1.svg"))
	if err != nil {
		t.Fatalf("confluence-roadmap-1.svg が書き出されていません: %v", err)
	}
	if !strings.HasPrefix(string(data), "<svg ") {
		t.Errorf("SVG の内容ではありません: %q", data[:min(40, len(data))])
	}
	if _, err := os.Stat(filepath.Join(pageDir, "confluence-roadmap-9.svg")); !os.IsNotExist(err) {
		t.Error("前回の変換で残った confluence-roadmap-9.svg を消すこと")
	}
	md, _ := os.ReadFile(filepath.Join(pageDir, "index.md"))
	if !strings.Contains(string(md), `{{< roadmap src="confluence-roadmap-1.svg" >}}`) {
		t.Errorf("本文にショートコードがありません:\n%s", md)
	}
}

func TestMDWriter_RoadmapSVGKeepsAttachments(t *testing.T) {
	tmpDir := t.TempDir()
	writer := newTestMDWriter(tmpDir)
	src := `{"title":"R","timeline":{"startDate":"2026-10-01 00:00:00","endDate":"2026-12-01 00:00:00","displayOption":"MONTH"},"lanes":[],"markers":[]}`
	macro := `{"type":"extension","attrs":{"extensionKey":"roadmap","parameters":{"macroParams":{"source":{"value":"` + url.PathEscape(src) + `"}}}}}`
	page := &Page{ID: "1", Title: "二つのロードマップ", SpaceID: "1",
		Body: PageBody{AtlasDocFormat: AtlasDocFormat{Value: `{"version":1,"type":"doc","content":[` + macro + `,` + macro + `]}`}}}

	pageDir := filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title))
	if err := os.MkdirAll(pageDir, 0755); err != nil {
		t.Fatal(err)
	}
	// 生成するファイルと同じ名前の添付ファイル（ダウンロード済み）は消さない
	attached := filepath.Join(pageDir, "confluence-roadmap-3.svg")
	if err := os.WriteFile(attached, []byte("attachment"), 0644); err != nil {
		t.Fatal(err)
	}
	attachments := []Attachment{{Title: "confluence-roadmap-3.svg"}}

	if err := writer.WritePage(page, "TEST", "", "", nil, nil, attachments); err != nil {
		t.Fatalf("WritePage エラー: %v", err)
	}
	for _, name := range []string{"confluence-roadmap-1.svg", "confluence-roadmap-2.svg"} {
		if _, err := os.Stat(filepath.Join(pageDir, name)); err != nil {
			t.Errorf("%s が書き出されていません: %v", name, err)
		}
	}
	if data, err := os.ReadFile(attached); err != nil || string(data) != "attachment" {
		t.Errorf("添付ファイルを消したり上書きしたりしないこと: %q, %v", data, err)
	}
}
