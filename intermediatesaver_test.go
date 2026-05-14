package main

import (
	"os"
	"path/filepath"
	"testing"
)

func makeTestPage() *Page {
	return &Page{
		ID:      "12345",
		Title:   "テストページ",
		Status:  "current",
		SpaceID: "67890",
		Body: PageBody{
			Storage: Storage{
				Value:          "<p>テストコンテンツ</p>",
				Representation: "storage",
			},
		},
		Version: Version{
			Number:    3,
			CreatedAt: "2024-01-01T00:00:00.000Z",
			AuthorID:  "user123",
		},
		Links: Links{
			WebUI: "/wiki/spaces/TEST/pages/12345",
		},
	}
}

// TestIntermediateSaver_SaveAndLoadPage は保存と読み込みの往復テスト
func TestIntermediateSaver_SaveAndLoadPage(t *testing.T) {
	tmpDir := t.TempDir()
	saver := NewIntermediateSaver(tmpDir)

	page := makeTestPage()
	labels := []Label{
		{Name: "golang"},
		{Name: "backend"},
	}

	if err := saver.SavePage(page, "TEST", labels); err != nil {
		t.Fatalf("保存エラー: %v", err)
	}

	xhtmlPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title), "content.xhtml")
	if _, err := os.Stat(xhtmlPath); os.IsNotExist(err) {
		t.Errorf("XHTMLファイルが作成されていません: %s", xhtmlPath)
	}

	metaPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title), "metadata.toml")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Errorf("メタデータファイルが作成されていません: %s", metaPath)
	}

	loadedPage, loadedLabels, err := saver.LoadPage("TEST", sanitizeFilename(page.Title))
	if err != nil {
		t.Fatalf("読み込みエラー: %v", err)
	}

	if loadedPage.ID != page.ID {
		t.Errorf("ページIDが一致しません\n期待: %q\n実際: %q", page.ID, loadedPage.ID)
	}
	if loadedPage.Title != page.Title {
		t.Errorf("タイトルが一致しません\n期待: %q\n実際: %q", page.Title, loadedPage.Title)
	}
	if loadedPage.Body.Storage.Value != page.Body.Storage.Value {
		t.Errorf("XHTMLコンテンツが一致しません\n期待: %q\n実際: %q",
			page.Body.Storage.Value, loadedPage.Body.Storage.Value)
	}
	if loadedPage.Version.Number != page.Version.Number {
		t.Errorf("バージョン番号が一致しません\n期待: %d\n実際: %d",
			page.Version.Number, loadedPage.Version.Number)
	}

	if len(loadedLabels) != len(labels) {
		t.Errorf("ラベル数が一致しません\n期待: %d\n実際: %d", len(labels), len(loadedLabels))
		return
	}
	if loadedLabels[0].Name != "golang" {
		t.Errorf("ラベルが一致しません\n期待: %q\n実際: %q", "golang", loadedLabels[0].Name)
	}
}

// TestIntermediateSaver_SaveAndLoadComments はコメントの保存と読み込みテスト
func TestIntermediateSaver_SaveAndLoadComments(t *testing.T) {
	tmpDir := t.TempDir()
	saver := NewIntermediateSaver(tmpDir)

	page := makeTestPage()
	if err := saver.SavePage(page, "TEST", nil); err != nil {
		t.Fatalf("ページ保存エラー: %v", err)
	}

	comments := []Comment{
		{
			ID: "c001",
			Body: CommentBody{
				Storage: Storage{
					Value:          "<p>コメント1</p>",
					Representation: "storage",
				},
			},
			Version: Version{
				CreatedAt: "2024-01-02T00:00:00.000Z",
				AuthorID:  "user456",
			},
		},
		{
			ID: "c002",
			Body: CommentBody{
				Storage: Storage{
					Value:          "<p>コメント2</p>",
					Representation: "storage",
				},
			},
			Version: Version{
				CreatedAt: "2024-01-03T00:00:00.000Z",
				AuthorID:  "user789",
			},
		},
	}

	if err := saver.SaveComments(page.Title, "TEST", comments); err != nil {
		t.Fatalf("コメント保存エラー: %v", err)
	}

	loadedComments, err := saver.LoadComments("TEST", sanitizeFilename(page.Title))
	if err != nil {
		t.Fatalf("コメント読み込みエラー: %v", err)
	}

	if len(loadedComments) != len(comments) {
		t.Errorf("コメント数が一致しません\n期待: %d\n実際: %d", len(comments), len(loadedComments))
		return
	}

	if loadedComments[0].Body.Storage.Value != comments[0].Body.Storage.Value {
		t.Errorf("コメント1のXHTMLが一致しません")
	}
}

// TestIntermediateSaver_LoadComments_NoDirectory はコメントディレクトリなしの場合のテスト
func TestIntermediateSaver_LoadComments_NoDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	saver := NewIntermediateSaver(tmpDir)

	comments, err := saver.LoadComments("TEST", "存在しないページ")
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("コメントが存在しないはずが: %d件", len(comments))
	}
}

// TestIntermediateSaver_ListPages は保存済みページ一覧のテスト
func TestIntermediateSaver_ListPages(t *testing.T) {
	tmpDir := t.TempDir()
	saver := NewIntermediateSaver(tmpDir)

	pages := []*Page{
		{ID: "1", Title: "ページA", Status: "current", SpaceID: "TEST",
			Body:    PageBody{Storage: Storage{Value: "<p>A</p>"}},
			Version: Version{Number: 1}},
		{ID: "2", Title: "ページB", Status: "current", SpaceID: "TEST",
			Body:    PageBody{Storage: Storage{Value: "<p>B</p>"}},
			Version: Version{Number: 1}},
	}

	for _, p := range pages {
		if err := saver.SavePage(p, "TEST", nil); err != nil {
			t.Fatalf("ページ保存エラー: %v", err)
		}
	}

	titles, err := saver.ListPages("TEST")
	if err != nil {
		t.Fatalf("ページ一覧取得エラー: %v", err)
	}

	if len(titles) != 2 {
		t.Errorf("ページ数が期待と異なります\n期待: 2\n実際: %d", len(titles))
	}
}
