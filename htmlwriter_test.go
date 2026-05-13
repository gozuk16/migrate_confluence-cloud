package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestHTMLWriter(dir string) *HTMLWriter {
	conv := NewConverter(nil, nil)
	return NewHTMLWriter(dir, conv)
}

// TestHTMLWriter_WritePage は WritePage のテスト
func TestHTMLWriter_WritePage(t *testing.T) {
	tmpDir := t.TempDir()
	writer := newTestHTMLWriter(tmpDir)

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
	}

	err := writer.WritePage(page, "TEST", "テストスペース", "", labels, nil, nil)
	if err != nil {
		t.Fatalf("WritePage エラー: %v", err)
	}

	// index.html が生成されたか確認
	htmlPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title), "index.html")
	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		t.Fatalf("index.html が生成されていません: %s", htmlPath)
	}

	data, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("ファイル読み込みエラー: %v", err)
	}
	content := string(data)

	// HTML5 基本構造の確認
	checks := []string{
		"<!DOCTYPE html>",
		"<html lang=\"ja\">",
		"<meta charset=\"UTF-8\">",
		"テストページ",     // タイトル
		"テストコンテンツ", // 本文
		"golang",          // ラベル
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("index.html に %q が含まれていません", want)
		}
	}
}

// TestHTMLWriter_WritePage_WithComments はコメント付きページのテスト
func TestHTMLWriter_WritePage_WithComments(t *testing.T) {
	tmpDir := t.TempDir()
	writer := newTestHTMLWriter(tmpDir)

	page := &Page{
		ID:    "1",
		Title: "コメントテスト",
		Body:  PageBody{Storage: Storage{Value: "<p>本文</p>"}},
		Version: Version{
			Number:    1,
			CreatedAt: "2024-01-01T00:00:00.000Z",
		},
	}

	comments := []Comment{
		{
			ID:   "c1",
			Body: CommentBody{Storage: Storage{Value: "<p>コメント内容</p>"}},
			Version: Version{
				CreatedAt: "2024-01-02T00:00:00.000Z",
			},
		},
	}

	err := writer.WritePage(page, "TEST", "", "", nil, comments, nil)
	if err != nil {
		t.Fatalf("WritePage エラー: %v", err)
	}

	htmlPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title), "index.html")
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("ファイル読み込みエラー: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "コメント内容") {
		t.Errorf("コメント内容が HTML に含まれていません")
	}
	if !strings.Contains(content, "コメント") {
		t.Errorf("コメントセクションヘッダーが含まれていません")
	}
}
