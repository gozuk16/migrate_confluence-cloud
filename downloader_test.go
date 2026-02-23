package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestSanitizeFilename はsanitizeFilenameのテスト
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "スラッシュを置換",
			input:    "path/to/file.txt",
			expected: "path_to_file.txt",
		},
		{
			name:     "バックスラッシュを置換",
			input:    "path\\to\\file.txt",
			expected: "path_to_file.txt",
		},
		{
			name:     "コロンを置換",
			input:    "file:name.txt",
			expected: "file_name.txt",
		},
		{
			name:     "通常のファイル名はそのまま",
			input:    "normal-file_name.png",
			expected: "normal-file_name.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("期待と異なります\n期待: %q\n実際: %q", tt.expected, result)
			}
		})
	}
}

// TestIsImageFile はIsImageFileのテスト
func TestIsImageFile(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"image.png", true},
		{"photo.jpg", true},
		{"photo.jpeg", true},
		{"animation.gif", true},
		{"icon.svg", true},
		{"picture.webp", true},
		{"document.pdf", false},
		{"spreadsheet.xlsx", false},
		{"archive.zip", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsImageFile(tt.filename)
			if result != tt.want {
				t.Errorf("IsImageFile(%q) = %v, 期待: %v", tt.filename, result, tt.want)
			}
		})
	}
}

// TestDownloadAttachments はDownloadAttachmentsのテスト
func TestDownloadAttachments(t *testing.T) {
	// テスト用HTTPサーバー
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Basic認証の確認
		_, _, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		io.WriteString(w, "fake-image-content")
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	downloader := NewDownloader("test@example.com", "token")

	attachments := []Attachment{
		{
			ID:       "attach1",
			Title:    "test-image.png",
			PageID:   "12345",
			Links:    Links{Download: server.URL + "/download/test-image.png"},
		},
	}

	files, err := downloader.DownloadAttachments(server.URL, attachments, tmpDir)
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
		return
	}

	if len(files) != 1 {
		t.Errorf("ダウンロードファイル数が期待と異なります\n期待: 1\n実際: %d", len(files))
		return
	}

	// ファイルが存在するか確認
	destPath := filepath.Join(tmpDir, "test-image.png")
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("ダウンロードされたファイルが存在しません: %s", destPath)
	}
}

// TestDownloadAttachments_SkipExisting は既存ファイルをスキップするテスト
func TestDownloadAttachments_SkipExisting(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "image/png")
		io.WriteString(w, "fake-image-content")
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	// 事前にファイルを作成
	existingFile := filepath.Join(tmpDir, "existing.png")
	if err := os.WriteFile(existingFile, []byte("existing-content"), 0644); err != nil {
		t.Fatalf("既存ファイルの作成に失敗: %v", err)
	}

	downloader := NewDownloader("test@example.com", "token")
	attachments := []Attachment{
		{
			ID:     "attach1",
			Title:  "existing.png",
			PageID: "12345",
			Links:  Links{Download: server.URL + "/download/existing.png"},
		},
	}

	_, err := downloader.DownloadAttachments(server.URL, attachments, tmpDir)
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}

	// APIが呼ばれていないことを確認
	if callCount != 0 {
		t.Errorf("既存ファイルに対してAPIが呼ばれました (callCount: %d)", callCount)
	}
}
