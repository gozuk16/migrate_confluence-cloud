package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// PageMetadata はページのメタデータを表す構造体（TOML保存用）
type PageMetadata struct {
	ID          string   `toml:"id"`
	Title       string   `toml:"title"`
	Status      string   `toml:"status"`
	SpaceID     string   `toml:"space_id"`
	SpaceKey    string   `toml:"space_key"`
	ParentID    string   `toml:"parent_id"`
	ParentTitle string   `toml:"parent_title"`
	CreatedAt   string   `toml:"created_at"`
	UpdatedAt   string   `toml:"updated_at"`
	AuthorID    string   `toml:"author_id"`
	Version     int      `toml:"version"`
	Labels      []string `toml:"labels"`
	WebURL      string   `toml:"web_url"`
}

// CommentMetadata はコメントのメタデータ
type CommentMetadata struct {
	ID        string `toml:"id"`
	CreatedAt string `toml:"created_at"`
	AuthorID  string `toml:"author_id"`
}

// IntermediateSaver は中間ファイル（Confluence Storage Format）の保存・読み込みを管理する
type IntermediateSaver struct {
	baseDir string
}

// NewIntermediateSaver は新しいIntermediateSaverを作成する
func NewIntermediateSaver(baseDir string) *IntermediateSaver {
	return &IntermediateSaver{baseDir: baseDir}
}

// pageDir はページのディレクトリパスを返す
func (s *IntermediateSaver) pageDir(spaceKey, pageTitle string) string {
	safeTitle := sanitizeFilename(pageTitle)
	return filepath.Join(s.baseDir, spaceKey, safeTitle)
}

// SavePage はページのXHTMLとメタデータをファイルに保存する
func (s *IntermediateSaver) SavePage(page *Page, spaceKey string, labels []Label) error {
	dir := s.pageDir(spaceKey, page.Title)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ディレクトリ作成エラー: %w", err)
	}

	xhtmlPath := filepath.Join(dir, "content.xhtml")
	if err := os.WriteFile(xhtmlPath, []byte(page.Body.Storage.Value), 0644); err != nil {
		return fmt.Errorf("中間ファイル保存エラー: %w", err)
	}

	// ラベル名の抽出
	labelNames := make([]string, 0, len(labels))
	for _, l := range labels {
		labelNames = append(labelNames, l.Name)
	}

	// メタデータの保存
	meta := PageMetadata{
		ID:        page.ID,
		Title:     page.Title,
		Status:    page.Status,
		SpaceID:   page.SpaceID,
		SpaceKey:  spaceKey,
		ParentID:  page.ParentID,
		CreatedAt: page.Version.CreatedAt,
		UpdatedAt: page.Version.CreatedAt,
		AuthorID:  page.Version.AuthorID,
		Version:   page.Version.Number,
		Labels:    labelNames,
		WebURL:    page.Links.WebUI,
	}

	metaPath := filepath.Join(dir, "metadata.toml")
	f, err := os.Create(metaPath)
	if err != nil {
		return fmt.Errorf("メタデータファイル作成エラー: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(meta); err != nil {
		return fmt.Errorf("メタデータ保存エラー: %w", err)
	}

	return nil
}

// SaveComments はコメントの中間ファイルを保存する
func (s *IntermediateSaver) SaveComments(pageTitle, spaceKey string, comments []Comment) error {
	if len(comments) == 0 {
		return nil
	}

	commentsDir := filepath.Join(s.pageDir(spaceKey, pageTitle), "comments")
	if err := os.MkdirAll(commentsDir, 0755); err != nil {
		return fmt.Errorf("コメントディレクトリ作成エラー: %w", err)
	}

	for i, comment := range comments {
		xhtmlPath := filepath.Join(commentsDir, fmt.Sprintf("comment_%03d.xhtml", i+1))
		if err := os.WriteFile(xhtmlPath, []byte(comment.Body.Storage.Value), 0644); err != nil {
			return fmt.Errorf("コメント中間ファイル保存エラー (ID: %s): %w", comment.ID, err)
		}

		meta := CommentMetadata{
			ID:        comment.ID,
			CreatedAt: comment.Version.CreatedAt,
			AuthorID:  comment.Version.AuthorID,
		}
		metaPath := filepath.Join(commentsDir, fmt.Sprintf("comment_%03d.toml", i+1))
		f, err := os.Create(metaPath)
		if err != nil {
			return fmt.Errorf("コメントメタデータ作成エラー: %w", err)
		}
		encErr := toml.NewEncoder(f).Encode(meta)
		f.Close()
		if encErr != nil {
			return fmt.Errorf("コメントメタデータ保存エラー: %w", encErr)
		}
	}

	return nil
}

// LoadPage は中間ファイルとメタデータファイルからページを読み込む
func (s *IntermediateSaver) LoadPage(spaceKey, pageTitle string) (*Page, []Label, error) {
	dir := s.pageDir(spaceKey, pageTitle)

	xhtmlPath := filepath.Join(dir, "content.xhtml")
	xhtmlData, err := os.ReadFile(xhtmlPath)
	if err != nil {
		return nil, nil, fmt.Errorf("中間ファイル読み込みエラー: %w", err)
	}

	metaPath := filepath.Join(dir, "metadata.toml")
	var meta PageMetadata
	if _, err := toml.DecodeFile(metaPath, &meta); err != nil {
		return nil, nil, fmt.Errorf("メタデータ読み込みエラー: %w", err)
	}

	page := &Page{
		ID:      meta.ID,
		Title:   meta.Title,
		Status:  meta.Status,
		SpaceID: meta.SpaceID,
		Body: PageBody{
			Storage: Storage{
				Value:          string(xhtmlData),
				Representation: "storage",
			},
		},
		Version: Version{
			Number:    meta.Version,
			CreatedAt: meta.CreatedAt,
			AuthorID:  meta.AuthorID,
		},
		Links: Links{
			WebUI: meta.WebURL,
		},
	}

	labels := make([]Label, 0, len(meta.Labels))
	for _, name := range meta.Labels {
		labels = append(labels, Label{Name: name})
	}

	return page, labels, nil
}

// LoadComments はコメントXHTMLとメタデータを読み込む
func (s *IntermediateSaver) LoadComments(spaceKey, pageTitle string) ([]Comment, error) {
	commentsDir := filepath.Join(s.pageDir(spaceKey, pageTitle), "comments")

	if _, err := os.Stat(commentsDir); os.IsNotExist(err) {
		return []Comment{}, nil
	}

	entries, err := os.ReadDir(commentsDir)
	if err != nil {
		return nil, fmt.Errorf("コメントディレクトリ読み込みエラー: %w", err)
	}

	var comments []Comment
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".xhtml") {
			continue
		}

		xhtmlPath := filepath.Join(commentsDir, entry.Name())
		xhtmlData, err := os.ReadFile(xhtmlPath)
		if err != nil {
			return nil, fmt.Errorf("コメントXHTML読み込みエラー: %w", err)
		}

		metaFile := strings.TrimSuffix(entry.Name(), ".xhtml") + ".toml"
		metaPath := filepath.Join(commentsDir, metaFile)

		var meta CommentMetadata
		if _, err := toml.DecodeFile(metaPath, &meta); err != nil {
			meta = CommentMetadata{}
		}

		comment := Comment{
			ID: meta.ID,
			Body: CommentBody{
				Storage: Storage{
					Value:          string(xhtmlData),
					Representation: "storage",
				},
			},
			Version: Version{
				CreatedAt: meta.CreatedAt,
				AuthorID:  meta.AuthorID,
			},
		}
		comments = append(comments, comment)
	}

	return comments, nil
}

// ListPages はspaceKey配下の保存済みページタイトル一覧を返す
func (s *IntermediateSaver) ListPages(spaceKey string) ([]string, error) {
	spaceDir := filepath.Join(s.baseDir, spaceKey)

	entries, err := os.ReadDir(spaceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("スペースディレクトリ読み込みエラー: %w", err)
	}

	var pageTitles []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(spaceDir, entry.Name(), "metadata.toml")
		if _, err := os.Stat(metaPath); err == nil {
			pageTitles = append(pageTitles, entry.Name())
		}
	}

	return pageTitles, nil
}
