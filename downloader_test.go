package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
		if _, err := io.WriteString(w, "fake-image-content"); err != nil {
			t.Errorf("レスポンスの書き込みに失敗しました: %v", err)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	downloader := NewDownloader("test@example.com", "token")

	attachments := []Attachment{
		{
			ID:     "attach1",
			Title:  "test-image.png",
			PageID: "12345",
			Links:  Links{Download: server.URL + "/download/test-image.png"},
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

// newCountingServer は、呼ばれた回数を数えながら body を返すテスト用サーバーを作る
func newCountingServer(t *testing.T, body string, status int) (*httptest.Server, *int) {
	t.Helper()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(status)
		if _, err := io.WriteString(w, body); err != nil {
			t.Errorf("レスポンスの書き込みに失敗しました: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	return server, &callCount
}

// TestDownloadAttachments_Redownload は、保存済みのファイルを取り直すかどうかの判定のテスト。
// 保存したファイルの更新日時は Confluence の版の作成日時に合わせてあり、
// 次回はその日時とサイズを Confluence 側と比べる
func TestDownloadAttachments_Redownload(t *testing.T) {
	const oldContent = "old-content"
	const newContent = "new-content-from-confluence"
	const versionAt = "2026-11-14T10:12:28.798Z"
	versionTime, err := time.Parse(time.RFC3339Nano, versionAt)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		localMtime   time.Time // 保存済みファイルの更新日時
		createdAt    string    // Confluence の版の作成日時
		fileSize     int64     // Confluence のファイルサイズ
		wantDownload bool
	}{
		{
			name:         "更新日時とサイズが同じなら取り直さない",
			localMtime:   versionTime,
			createdAt:    versionAt,
			fileSize:     int64(len(oldContent)),
			wantDownload: false,
		},
		{
			name:         "Confluence 側が新しければ取り直す",
			localMtime:   versionTime.Add(-3 * time.Hour),
			createdAt:    versionAt,
			fileSize:     int64(len(newContent)),
			wantDownload: true,
		},
		{
			name:         "サイズが同じでも更新日時が違えば取り直す",
			localMtime:   versionTime.Add(-3 * time.Hour),
			createdAt:    versionAt,
			fileSize:     int64(len(oldContent)),
			wantDownload: true,
		},
		{
			name:         "手元のファイルの方が新しくても、版の日時と違えば取り直す（修正前に保存したファイル）",
			localMtime:   versionTime.Add(5 * time.Hour),
			createdAt:    versionAt,
			fileSize:     int64(len(oldContent)),
			wantDownload: true,
		},
		{
			name:         "更新日時が同じでもサイズが違えば取り直す（途中で切れたファイル）",
			localMtime:   versionTime,
			createdAt:    versionAt,
			fileSize:     int64(len(newContent)),
			wantDownload: true,
		},
		{
			name:         "更新日時の細かさによる違い（2 秒以内）は同じとみなす",
			localMtime:   versionTime.Add(-1500 * time.Millisecond),
			createdAt:    versionAt,
			fileSize:     int64(len(oldContent)),
			wantDownload: false,
		},
		{
			name:         "版の作成日時が分からなければ取り直す",
			localMtime:   versionTime,
			createdAt:    "",
			fileSize:     int64(len(oldContent)),
			wantDownload: true,
		},
		{
			name:         "サイズが分からなければ更新日時だけで比べる",
			localMtime:   versionTime,
			createdAt:    versionAt,
			fileSize:     0,
			wantDownload: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, callCount := newCountingServer(t, newContent, http.StatusOK)
			tmpDir := t.TempDir()
			destPath := filepath.Join(tmpDir, "diagram.drawio")
			if err := os.WriteFile(destPath, []byte(oldContent), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(destPath, tt.localMtime, tt.localMtime); err != nil {
				t.Fatal(err)
			}

			attachments := []Attachment{{
				ID:       "attach1",
				Title:    "diagram.drawio",
				PageID:   "12345",
				FileSize: tt.fileSize,
				Version:  Version{Number: 3, CreatedAt: tt.createdAt},
				Links:    Links{Download: server.URL + "/download/diagram.drawio"},
			}}
			files, err := NewDownloader("test@example.com", "token").DownloadAttachments(server.URL, attachments, tmpDir)
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}
			if len(files) != 1 || files[0] != "diagram.drawio" {
				t.Errorf("戻り値が期待と異なります: %v", files)
			}

			got, err := os.ReadFile(destPath)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantDownload {
				if *callCount != 1 || string(got) != newContent {
					t.Errorf("取り直されていません (callCount: %d, 内容: %q)", *callCount, got)
				}
			} else {
				if *callCount != 0 || string(got) != oldContent {
					t.Errorf("取り直されました (callCount: %d, 内容: %q)", *callCount, got)
				}
			}
		})
	}
}

// TestDownloadAttachments_SetsModTime は、保存したファイルの更新日時を版の作成日時に合わせ、
// 2 回目は取り直さないことのテスト
func TestDownloadAttachments_SetsModTime(t *testing.T) {
	const content = "fake-image-content"
	const versionAt = "2026-11-14T06:35:12.431Z"
	server, callCount := newCountingServer(t, content, http.StatusOK)
	tmpDir := t.TempDir()
	attachments := []Attachment{{
		ID:       "attach1",
		Title:    "test1.drawio",
		PageID:   "12345",
		FileSize: int64(len(content)),
		Version:  Version{Number: 2, CreatedAt: versionAt},
		Links:    Links{Download: server.URL + "/download/test1.drawio"},
	}}
	downloader := NewDownloader("test@example.com", "token")

	for i := 0; i < 2; i++ {
		if _, err := downloader.DownloadAttachments(server.URL, attachments, tmpDir); err != nil {
			t.Fatalf("予期しないエラー (%d 回目): %v", i+1, err)
		}
	}
	if *callCount != 1 {
		t.Errorf("ダウンロードの回数が期待と異なります\n期待: 1\n実際: %d", *callCount)
	}

	info, err := os.Stat(filepath.Join(tmpDir, "test1.drawio"))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := time.Parse(time.RFC3339Nano, versionAt)
	if diff := info.ModTime().Sub(want).Abs(); diff > modTimeTolerance {
		t.Errorf("更新日時が版の作成日時になっていません\n期待: %v\n実際: %v", want, info.ModTime())
	}
}

// TestDownloadAttachments_FailureKeepsExisting は、取り直しに失敗したときに
// 保存済みのファイルを壊さず、作業用のファイルも残さないことのテスト
func TestDownloadAttachments_FailureKeepsExisting(t *testing.T) {
	server, _ := newCountingServer(t, "error", http.StatusInternalServerError)
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "diagram.drawio")
	if err := os.WriteFile(destPath, []byte("old-content"), 0644); err != nil {
		t.Fatal(err)
	}

	attachments := []Attachment{{
		ID:       "attach1",
		Title:    "diagram.drawio",
		PageID:   "12345",
		FileSize: 999,
		Version:  Version{Number: 3, CreatedAt: "2026-11-14T10:12:28.798Z"},
		Links:    Links{Download: server.URL + "/download/diagram.drawio"},
	}}
	if _, err := NewDownloader("test@example.com", "token").DownloadAttachments(server.URL, attachments, tmpDir); err == nil {
		t.Error("エラーが返りませんでした")
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old-content" {
		t.Errorf("保存済みのファイルが変わりました: %q", got)
	}
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("作業用のファイルが残っています: %v", entries)
	}
}

// TestDownloadAttachments_ContinuesAfterFailure は、一部の添付で失敗しても
// 残りの添付をダウンロードし、失敗をエラーで返すことのテスト
func TestDownloadAttachments_ContinuesAfterFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/broken.png") {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		if _, err := io.WriteString(w, "content"); err != nil {
			t.Errorf("レスポンスの書き込みに失敗しました: %v", err)
		}
	}))
	defer server.Close()
	tmpDir := t.TempDir()

	var attachments []Attachment
	for _, title := range []string{"first.png", "broken.png", "last.png"} {
		attachments = append(attachments, Attachment{
			ID:      title,
			Title:   title,
			PageID:  "12345",
			Version: Version{Number: 1, CreatedAt: "2026-11-14T10:12:28.798Z"},
			Links:   Links{Download: server.URL + "/download/" + title},
		})
	}
	files, err := NewDownloader("test@example.com", "token").DownloadAttachments(server.URL, attachments, tmpDir)
	if err == nil || !strings.Contains(err.Error(), "broken.png") {
		t.Errorf("失敗した添付の名前を含むエラーが返りませんでした: %v", err)
	}
	if len(files) != 2 || files[0] != "first.png" || files[1] != "last.png" {
		t.Errorf("保存できたファイルが期待と異なります: %v", files)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "last.png")); err != nil {
		t.Errorf("失敗した添付の後の添付が保存されていません: %v", err)
	}
}

// TestDownloadAttachments_TruncatedBody は、本文が途中で切れたときのテスト。
// 保存済みのファイルを壊さず、作業用のファイルも残さない
func TestDownloadAttachments_TruncatedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content-Length より短い本文を送って接続を切る
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, "short"); err != nil {
			t.Errorf("レスポンスの書き込みに失敗しました: %v", err)
		}
		// 送ってから切らないと、ヘッダーも届かずにリクエスト自体の失敗になる
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("レスポンスを途中で送れません")
			return
		}
		flusher.Flush()
		panic(http.ErrAbortHandler)
	}))
	defer server.Close()
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "diagram.drawio")
	if err := os.WriteFile(destPath, []byte("old-content"), 0644); err != nil {
		t.Fatal(err)
	}

	attachments := []Attachment{{
		ID:       "attach1",
		Title:    "diagram.drawio",
		PageID:   "12345",
		FileSize: 100,
		Version:  Version{Number: 3, CreatedAt: "2026-11-14T10:12:28.798Z"},
		Links:    Links{Download: server.URL + "/download/diagram.drawio"},
	}}
	_, err := NewDownloader("test@example.com", "token").DownloadAttachments(server.URL, attachments, tmpDir)
	if err == nil || !strings.Contains(err.Error(), "ファイルの書き込みに失敗しました") {
		t.Errorf("書き込みの失敗のエラーが返りませんでした: %v", err)
	}
	assertOnlyFile(t, tmpDir, "diagram.drawio", "old-content")
}

// assertOnlyFile は、dir に name だけがあり（作業用のファイルが残っていない）、内容が want であることを確かめる
func assertOnlyFile(t *testing.T, dir, name, want string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != name {
		t.Errorf("%s 以外のファイルがあります: %v", name, entries)
	}
	got, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("内容が期待と異なります\n期待: %q\n実際: %q", want, got)
	}
}

// failingReader は、少し読ませてからエラーを返す io.Reader
type failingReader struct{ sent bool }

func (r *failingReader) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		return copy(p, "partial"), nil
	}
	return 0, errors.New("読み込みエラー")
}

// TestSaveAttachment は saveAttachment のテスト
func TestSaveAttachment(t *testing.T) {
	versionTime := time.Date(2026, 11, 14, 10, 12, 28, 0, time.UTC)
	// 版の作成日時を保存の印に使うので、「合わせていない」ことは現在時刻に近いことで確かめる
	isVersionTime := func(t *testing.T, path string) bool {
		t.Helper()
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		return info.ModTime().Sub(versionTime).Abs() <= modTimeTolerance
	}

	t.Run("読み込みに失敗したら保存済みのファイルを残し、作業用のファイルを消す", func(t *testing.T) {
		dir := t.TempDir()
		destPath := filepath.Join(dir, "a.drawio")
		if err := os.WriteFile(destPath, []byte("old-content"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := saveAttachment(destPath, &failingReader{}, 0, versionTime); err == nil {
			t.Error("エラーが返りませんでした")
		}
		assertOnlyFile(t, dir, "a.drawio", "old-content")
	})

	t.Run("名前を変えられなければ作業用のファイルを消す", func(t *testing.T) {
		dir := t.TempDir()
		// 保存先が空でないディレクトリだと、名前を変えられない
		destPath := filepath.Join(dir, "a.drawio")
		if err := os.MkdirAll(filepath.Join(destPath, "sub"), 0755); err != nil {
			t.Fatal(err)
		}
		err := saveAttachment(destPath, strings.NewReader("content"), 0, versionTime)
		if err == nil || !strings.Contains(err.Error(), "ファイルの保存に失敗しました") {
			t.Errorf("保存の失敗のエラーが返りませんでした: %v", err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 {
			t.Errorf("作業用のファイルが残っています: %v", entries)
		}
	})

	t.Run("サイズが合えば更新日時を版の作成日時に合わせる", func(t *testing.T) {
		dir := t.TempDir()
		destPath := filepath.Join(dir, "a.drawio")
		if err := saveAttachment(destPath, strings.NewReader("content"), int64(len("content")), versionTime); err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}
		assertOnlyFile(t, dir, "a.drawio", "content")
		if !isVersionTime(t, destPath) {
			t.Error("更新日時が版の作成日時になっていません")
		}
		info, err := os.Stat(destPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0644 {
			t.Errorf("権限が期待と異なります\n期待: 0644\n実際: %o", info.Mode().Perm())
		}
	})

	t.Run("サイズが Confluence の情報と違えば、保存はするが更新日時を合わせない（次回取り直す）", func(t *testing.T) {
		dir := t.TempDir()
		destPath := filepath.Join(dir, "a.drawio")
		if err := saveAttachment(destPath, strings.NewReader("short"), 100, versionTime); err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}
		assertOnlyFile(t, dir, "a.drawio", "short")
		if isVersionTime(t, destPath) {
			t.Error("更新日時が版の作成日時に合わせられています")
		}
	})
}

// TestAttachmentDownloadURL は添付ファイルのダウンロードURLの組み立てのテスト。
// v2 API の _links.download は /wiki を基準にした相対パスで返る
func TestAttachmentDownloadURL(t *testing.T) {
	const base = "https://example.atlassian.net"
	tests := []struct {
		name     string
		baseURL  string
		download string
		want     string
	}{
		{
			name:     "v2 API の相対パスは /wiki を補う",
			baseURL:  base,
			download: "/rest/api/content/123/child/attachment/att1/download",
			want:     base + "/wiki/rest/api/content/123/child/attachment/att1/download",
		},
		{
			name:     "/wiki から始まる相対パスはそのまま付ける",
			baseURL:  base,
			download: "/wiki/download/attachments/123/a.png",
			want:     base + "/wiki/download/attachments/123/a.png",
		},
		{
			name:     "URL の末尾にスラッシュがあっても二重にしない",
			baseURL:  base + "/",
			download: "/rest/api/content/123/child/attachment/att1/download",
			want:     base + "/wiki/rest/api/content/123/child/attachment/att1/download",
		},
		{
			name:     "絶対URLはそのまま使う",
			baseURL:  base,
			download: "https://other.example.com/file.png",
			want:     "https://other.example.com/file.png",
		},
		{
			name:     "download が無ければ v1 のダウンロードURLを組み立てる",
			baseURL:  base,
			download: "",
			want:     base + "/wiki/rest/api/content/123/child/attachment/att1/download",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Attachment{ID: "att1", PageID: "123", Links: Links{Download: tt.download}}
			if got := attachmentDownloadURL(tt.baseURL, a); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
