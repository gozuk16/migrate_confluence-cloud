# HTML出力追加と中間ファイル名称変更 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 中間ファイル関連の識別子を `Intermediate` に統一し、Confluence Storage Format から HTML Living Standard を生成する `HTMLWriter` を追加する。

**Architecture:** 既存の `Converter.preprocess()`（private）を `ToHTML()`（public）としてラップし、`HTMLWriter` がこれを使って HTML5 ドキュメントを生成する。`IntermediateSaver`（旧 `XHTMLSaver`）は生データの保存・読み込みを担い内容は変更なし。`main.go` で `MDWriter` と `HTMLWriter` の両方を実行する。

**Tech Stack:** Go 1.21+、`golang.org/x/net/html`、`github.com/BurntSushi/toml`、`github.com/urfave/cli/v3`

---

## 事前準備

新しいフィーチャーブランチを作成する。

```bash
git checkout main
git pull origin main
git checkout -b feature/html-output
```

---

## ファイル構成

| ファイル | 変更種別 | 概要 |
|---|---|---|
| `config.go` | 修正 | `XHTMLDir→IntermediateDir`、`HTMLDir` 追加 |
| `config_test.go` | 修正 | フィールド名・デフォルト値を更新 |
| `intermediatesaver.go` | 新規（旧 `xhtmlsaver.go`） | 識別子を `IntermediateSaver` に変更 |
| `intermediatesaver_test.go` | 新規（旧 `xhtmlsaver_test.go`） | テスト識別子を更新 |
| `xhtmlsaver.go` | 削除 | |
| `xhtmlsaver_test.go` | 削除 | |
| `converter.go` | 修正 | `ToHTML()` メソッド追加 |
| `converter_test.go` | 修正 | `TestConverter_ToHTML` テスト追加 |
| `htmlwriter.go` | 新規 | `HTMLWriter` 実装 |
| `htmlwriter_test.go` | 新規 | `HTMLWriter` テスト |
| `main.go` | 修正 | 識別子更新・`HTMLWriter` ワイヤリング |
| `config.toml.example` | 修正 | キー名更新・`html_dir` 追加 |

---

## Task 1: config.go と config_test.go を更新

**Files:**
- Modify: `config.go`
- Modify: `config_test.go`

- [ ] **Step 1: config_test.go を更新してテストを失敗状態にする**

`config_test.go` の `XHTMLDir` 参照をすべて `IntermediateDir` に、`"output/xhtml"` を `"output/intermediate"` に置き換える。

```go
// config_test.go の変更箇所（2カ所）

// TestLoadConfig 内のテスト設定ファイルの内容を変更:
content := `[confluence]
url = "https://test.atlassian.net"
email = "test@example.com"
api_token = "test-token-123"

[output]
markdown_dir = "output/markdown"
intermediate_dir = "output/intermediate"

[search]
default_space_key = "TEST"
`

// TestLoadConfig のデフォルト値チェックを変更:
if config.Output.IntermediateDir != "output/intermediate" {
    t.Errorf("IntermediateDirのデフォルト値が期待と異なります: %q", config.Output.IntermediateDir)
}

// TestValidate 内の OutputConfig リテラルを変更:
Output: OutputConfig{
    MarkdownDir:     "output/markdown",
    IntermediateDir: "output/intermediate",
},

// TestValidate のデフォルト値チェックを変更:
if tt.config.Output.IntermediateDir != "output/intermediate" {
    t.Errorf("IntermediateDirのデフォルト値が期待と異なります: %q", tt.config.Output.IntermediateDir)
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./... -run TestLoadConfig
```

期待: `FAIL` — `config.Output.XHTMLDir` フィールドが存在しないコンパイルエラーが出る（まだ config.go を変更していないため）

- [ ] **Step 3: config.go を更新**

`config.go` の `OutputConfig` 構造体と `Validate()` を以下に置き換える。

```go
// OutputConfig は出力設定を表す構造体
type OutputConfig struct {
	MarkdownDir     string `toml:"markdown_dir"`
	AttachmentsDir  string `toml:"attachments_dir"`
	IntermediateDir string `toml:"intermediate_dir"`
	HTMLDir         string `toml:"html_dir"`
}
```

```go
// Validate 内のデフォルト値設定を変更
if c.Output.IntermediateDir == "" {
    c.Output.IntermediateDir = "output/intermediate"
}
if c.Output.HTMLDir == "" {
    c.Output.HTMLDir = "output/html"
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./... -run "TestLoadConfig|TestValidate"
```

期待: `PASS`

- [ ] **Step 5: コミット**

```bash
git add config.go config_test.go
git commit -m "feat: OutputConfigのXHTMLDir→IntermediateDir変更とHTMLDir追加"
```

---

## Task 2: XHTMLSaver → IntermediateSaver リネーム

**Files:**
- Create: `intermediatesaver.go`
- Create: `intermediatesaver_test.go`
- Delete: `xhtmlsaver.go`
- Delete: `xhtmlsaver_test.go`
- Modify: `main.go`（識別子参照の更新のみ）

- [ ] **Step 1: intermediatesaver.go を作成する**

```go
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

	// XHTML本文の保存
	xhtmlPath := filepath.Join(dir, "content.xhtml")
	if err := os.WriteFile(xhtmlPath, []byte(page.Body.Storage.Value), 0644); err != nil {
		return fmt.Errorf("XHTMLファイル保存エラー: %w", err)
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

// SaveComments はコメントのXHTMLをファイルに保存する
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
			return fmt.Errorf("コメントXHTMLファイル保存エラー (ID: %s): %w", comment.ID, err)
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

// LoadPage はXHTMLとメタデータファイルからページを読み込む
func (s *IntermediateSaver) LoadPage(spaceKey, pageTitle string) (*Page, []Label, error) {
	dir := s.pageDir(spaceKey, pageTitle)

	xhtmlPath := filepath.Join(dir, "content.xhtml")
	xhtmlData, err := os.ReadFile(xhtmlPath)
	if err != nil {
		return nil, nil, fmt.Errorf("XHTMLファイル読み込みエラー: %w", err)
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
```

- [ ] **Step 2: intermediatesaver_test.go を作成する**

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
```

- [ ] **Step 3: main.go の IntermediateSaver 関連の識別子を更新する**

`main.go` の以下の変更を行う（コード全体は長いため差分を示す）。

```go
// 変更前 → 変更後（全出現箇所）

// 型名
*XHTMLSaver → *IntermediateSaver
NewXHTMLSaver → NewIntermediateSaver

// 変数名
xhtmlSaver → intermediateSaver
saveXHTML  → saveIntermediate

// CLIフラグ（Nameフィールド）
"save-xhtml" → "save-intermediate"

// Usageテキスト
"XHTML中間ファイルを保存する（デフォルト: true）" → "中間ファイル（Confluence Storage Format）を保存する（デフォルト: true）"
"保存済みXHTMLからMarkdownに変換する（APIアクセス不要）" → "保存済み中間ファイルからMarkdownとHTMLに変換する（APIアクセス不要）"

// 設定フィールド
cfg.Output.XHTMLDir → cfg.Output.IntermediateDir

// 関数名
convertFromXHTML → convertFromIntermediate

// 変数 xhtmlDir
xhtmlDir → intermediateDir
```

- [ ] **Step 4: xhtmlsaver.go と xhtmlsaver_test.go を削除する**

```bash
rm xhtmlsaver.go xhtmlsaver_test.go
```

- [ ] **Step 5: ビルドとテストが通ることを確認**

```bash
go build ./...
go test ./... -run "TestIntermediateSaver"
```

期待: `PASS`

- [ ] **Step 6: コミット**

```bash
git add intermediatesaver.go intermediatesaver_test.go main.go
git rm xhtmlsaver.go xhtmlsaver_test.go
git commit -m "refactor: XHTMLSaver→IntermediateSaver リネームとCLIフラグ更新"
```

---

## Task 3: Converter.ToHTML() を追加

**Files:**
- Modify: `converter.go`
- Modify: `converter_test.go`

- [ ] **Step 1: 失敗するテストを converter_test.go に追加する**

```go
// TestConverter_ToHTML は ToHTML メソッドのテスト
func TestConverter_ToHTML(t *testing.T) {
	c := newTestConverter()

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "空文字列",
			input:    "",
			contains: "",
		},
		{
			name:     "段落",
			input:    "<p>Hello, World!</p>",
			contains: "Hello, World!",
		},
		{
			name:     "コードブロックマクロ",
			input:    `<ac:structured-macro ac:name="code"><ac:parameter ac:name="language">go</ac:parameter><ac:plain-text-body><![CDATA[fmt.Println("hello")]]></ac:plain-text-body></ac:structured-macro>`,
			contains: "<pre>",
		},
		{
			name:     "infoマクロ",
			input:    `<ac:structured-macro ac:name="info"><ac:rich-text-body><p>メモ</p></ac:rich-text-body></ac:structured-macro>`,
			contains: "<blockquote>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.ToHTML(tt.input)
			if err != nil {
				t.Errorf("予期しないエラー: %v", err)
			}
			if tt.contains == "" {
				if result != "" {
					t.Errorf("空を期待しましたが %q が返りました", result)
				}
				return
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("結果に %q が含まれていません。結果: %q", tt.contains, result)
			}
		})
	}
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./... -run TestConverter_ToHTML
```

期待: `FAIL` — `c.ToHTML undefined`

- [ ] **Step 3: converter.go に ToHTML() を追加する**

`Convert()` メソッドの直後（行 206 付近）に追加する。

```go
// ToHTML は Confluence Storage Format を標準 HTML ボディフラグメントに変換する。
// Converter.preprocess() の公開ラッパー。HTMLWriter から利用される。
func (c *Converter) ToHTML(xhtml string) (string, error) {
	if xhtml == "" {
		return "", nil
	}
	return c.preprocess(xhtml)
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./... -run TestConverter_ToHTML
```

期待: `PASS`

- [ ] **Step 5: コミット**

```bash
git add converter.go converter_test.go
git commit -m "feat: Converter.ToHTML() を追加"
```

---

## Task 4: HTMLWriter を新規作成

**Files:**
- Create: `htmlwriter_test.go`
- Create: `htmlwriter.go`

- [ ] **Step 1: 失敗するテストを htmlwriter_test.go に作成する**

```go
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
		"テストページ",        // タイトル
		"テストコンテンツ",    // 本文
		"golang",             // ラベル
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
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./... -run TestHTMLWriter
```

期待: `FAIL` — `HTMLWriter undefined`

- [ ] **Step 3: htmlwriter.go を作成する**

```go
package main

import (
	"fmt"
	gohtml "html"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const htmlCSS = `body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;max-width:960px;margin:2em auto;padding:0 1.5em;line-height:1.6;color:#172b4d}h1,h2,h3,h4,h5,h6{color:#172b4d;margin-top:1.5em}table{border-collapse:collapse;width:100%;margin:1em 0}th,td{border:1px solid #dfe1e6;padding:.5em .75em;text-align:left;vertical-align:top}th{background:#f4f5f7;font-weight:600}tr:nth-child(even){background:#fafbfc}pre{background:#f4f5f7;border-radius:3px;padding:1em;overflow-x:auto;margin:1em 0}code{font-family:"SFMono-Regular",Consolas,monospace;font-size:.9em;background:#f4f5f7;padding:.1em .3em;border-radius:2px}pre code{background:none;padding:0;font-size:.85em}blockquote{border-left:4px solid #dfe1e6;margin:1em 0;padding:.5em 1em;background:#f8f9fa;color:#42526e}details{border:1px solid #dfe1e6;border-radius:3px;padding:.5em 1em;margin:1em 0}summary{cursor:pointer;font-weight:600;padding:.25em 0}img{max-width:100%;height:auto}.page-meta{color:#6b778c;font-size:.9em;margin-bottom:1.5em;padding-bottom:1em;border-bottom:1px solid #dfe1e6}.page-meta a{color:#0052cc}.comments{border-top:2px solid #dfe1e6;margin-top:2em;padding-top:1em}.comment{border:1px solid #dfe1e6;border-radius:3px;padding:1em;margin:.75em 0;background:#fff}.comment-meta{color:#6b778c;font-size:.85em;margin-bottom:.5em}footer{border-top:1px solid #dfe1e6;margin-top:2em;padding-top:1em;color:#6b778c;font-size:.85em}footer a{color:#0052cc}`

// HTMLWriter は HTML Living Standard ファイルの出力を管理する
type HTMLWriter struct {
	outputDir string
	conv      *Converter
}

// NewHTMLWriter は新しいHTMLWriterを作成する
func NewHTMLWriter(outputDir string, conv *Converter) *HTMLWriter {
	return &HTMLWriter{outputDir: outputDir, conv: conv}
}

// WritePage はページを HTML5 ファイルとして書き出す
func (w *HTMLWriter) WritePage(page *Page, spaceKey, spaceName, parentTitle string, labels []Label, comments []Comment, attachments []Attachment) error {
	bodyHTML, err := w.conv.ToHTML(page.Body.Storage.Value)
	if err != nil {
		bodyHTML = fmt.Sprintf("<p><em>変換エラー: %s</em></p>", gohtml.EscapeString(err.Error()))
	}

	doc := w.buildDocument(page, spaceKey, spaceName, bodyHTML, labels, comments)

	safeTitle := sanitizeFilename(page.Title)
	pageDir := filepath.Join(w.outputDir, spaceKey, safeTitle)
	if err := os.MkdirAll(pageDir, 0755); err != nil {
		return fmt.Errorf("出力ディレクトリ作成エラー: %w", err)
	}

	outPath := filepath.Join(pageDir, "index.html")
	return os.WriteFile(outPath, []byte(doc), 0644)
}

func (w *HTMLWriter) buildDocument(page *Page, spaceKey, spaceName, bodyHTML string, labels []Label, comments []Comment) string {
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n")
	sb.WriteString("<html lang=\"ja\">\n<head>\n")
	sb.WriteString("  <meta charset=\"UTF-8\">\n")
	sb.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", gohtml.EscapeString(page.Title)))
	sb.WriteString("  <style>" + htmlCSS + "</style>\n")
	sb.WriteString("</head>\n<body>\n")

	// ヘッダー
	sb.WriteString("<header>\n")
	sb.WriteString(fmt.Sprintf("  <h1>%s</h1>\n", gohtml.EscapeString(page.Title)))
	sb.WriteString("  <div class=\"page-meta\">\n")

	var metaParts []string
	if spaceName != "" {
		metaParts = append(metaParts, fmt.Sprintf("スペース: %s", gohtml.EscapeString(spaceName)))
	}
	if page.Version.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, page.Version.CreatedAt); err == nil {
			metaParts = append(metaParts, fmt.Sprintf("更新日時: %s", t.Format("2006-01-02 15:04")))
		}
	}
	if len(labels) > 0 {
		names := make([]string, len(labels))
		for i, l := range labels {
			names[i] = gohtml.EscapeString(l.Name)
		}
		metaParts = append(metaParts, fmt.Sprintf("ラベル: %s", strings.Join(names, ", ")))
	}
	if len(metaParts) > 0 {
		sb.WriteString("    <span>" + strings.Join(metaParts, " | ") + "</span>")
	}
	if page.Links.WebUI != "" {
		sb.WriteString(fmt.Sprintf("\n    <br><a href=\"%s\">Confluenceで開く ↗</a>",
			gohtml.EscapeString(page.Links.WebUI)))
	}
	sb.WriteString("\n  </div>\n</header>\n")

	// 本文
	sb.WriteString("<main>\n")
	sb.WriteString(bodyHTML)
	sb.WriteString("\n</main>\n")

	// コメント
	if len(comments) > 0 {
		sb.WriteString("<section class=\"comments\">\n")
		sb.WriteString("  <h2>コメント</h2>\n")
		for _, comment := range comments {
			commentHTML, err := w.conv.ToHTML(comment.Body.Storage.Value)
			if err != nil {
				commentHTML = fmt.Sprintf("<p><em>変換エラー: %s</em></p>", gohtml.EscapeString(err.Error()))
			}
			sb.WriteString("  <div class=\"comment\">\n")
			if comment.Version.CreatedAt != "" {
				if t, err := time.Parse(time.RFC3339, comment.Version.CreatedAt); err == nil {
					sb.WriteString(fmt.Sprintf("    <div class=\"comment-meta\">%s</div>\n", t.Format("2006-01-02 15:04")))
				}
			}
			sb.WriteString(commentHTML)
			sb.WriteString("\n  </div>\n")
		}
		sb.WriteString("</section>\n")
	}

	// フッター
	if page.Links.WebUI != "" {
		sb.WriteString("<footer>\n")
		sb.WriteString(fmt.Sprintf("  <p>元ページ: <a href=\"%s\">%s</a></p>\n",
			gohtml.EscapeString(page.Links.WebUI),
			gohtml.EscapeString(page.Links.WebUI)))
		sb.WriteString("</footer>\n")
	}

	sb.WriteString("</body>\n</html>")
	return sb.String()
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./... -run TestHTMLWriter
```

期待: `PASS`

- [ ] **Step 5: コミット**

```bash
git add htmlwriter.go htmlwriter_test.go
git commit -m "feat: HTMLWriter を新規追加"
```

---

## Task 5: main.go に HTMLWriter をワイヤリング

**Files:**
- Modify: `main.go`

- [ ] **Step 1: fetchPage の HTMLWriter 生成を追加する**

`fetchPage` 関数内の `writer := NewMDWriter(...)` 行の直後に以下を追加する。

```go
htmlWriter := NewHTMLWriter(cfg.Output.HTMLDir, conv)
```

- [ ] **Step 2: fetchSpace の HTMLWriter 生成を追加する**

`fetchSpace` 関数内の `writer := NewMDWriter(...)` 行の直後に以下を追加する。

```go
htmlWriter := NewHTMLWriter(cfg.Output.HTMLDir, conv)
```

- [ ] **Step 3: processPage のシグネチャと本体に HTMLWriter を追加する**

```go
// 変更前
func processPage(client *ConfluenceClient, writer *MDWriter, intermediateSaver *IntermediateSaver, downloader *Downloader, cfg *Config, pageID string, recursive bool) error {

// 変更後
func processPage(client *ConfluenceClient, writer *MDWriter, htmlWriter *HTMLWriter, intermediateSaver *IntermediateSaver, downloader *Downloader, cfg *Config, pageID string, recursive bool) error {
```

`processPage` 内の `// Markdown生成` ブロックの直後（`writer.WritePage(...)` 呼び出しの後）に以下を追加する。

```go
// HTML生成
if err := htmlWriter.WritePage(page, space.Key, space.Name, parentTitle, labels, comments, attachments); err != nil {
    slog.Warn("HTML生成エラー", "pageID", pageID, "error", err)
}
```

- [ ] **Step 4: fetchPage と fetchSpace の processPage 呼び出しを更新する**

```go
// fetchPage 内
if err := processPage(client, writer, htmlWriter, intermediateSaver, downloader, cfg, pageID, recursive); err != nil {

// fetchSpace 内
if err := processPage(client, writer, htmlWriter, intermediateSaver, downloader, cfg, page.ID, false); err != nil {
```

- [ ] **Step 5: convertFromIntermediate に HTMLWriter を追加する**

`convertFromIntermediate` 関数内の `writer := NewMDWriter(...)` 行の直後に以下を追加する。

```go
htmlWriter := NewHTMLWriter(cfg.Output.HTMLDir, conv)
```

`convertFromIntermediate` 内の `writer.WritePage(...)` 呼び出し（`continue` を含む既存ブロック）の後、`totalConverted++` の前に以下を追加する。

```go
// HTML生成（失敗しても変換済みカウントは増加）
if err := htmlWriter.WritePage(page, spaceKey, "", "", labels, comments, nil); err != nil {
    slog.Warn("HTML生成エラー", "pageTitle", pageTitle, "error", err)
}
```

- [ ] **Step 6: ビルドとテスト確認**

```bash
go build ./...
go test ./...
```

期待: ビルド成功、全テスト `PASS`

- [ ] **Step 7: コミット**

```bash
git add main.go
git commit -m "feat: main.go に HTMLWriter をワイヤリング"
```

---

## Task 6: 設定ファイルの更新と最終確認

**Files:**
- Modify: `config.toml.example`

- [ ] **Step 1: config.toml.example を更新する**

```toml
[confluence]
# Confluence Cloud のURL（例: https://your-domain.atlassian.net）
url = "https://your-domain.atlassian.net"
# Confluenceユーザーのメールアドレス
email = "your-email@example.com"
# Confluence APIトークン（https://id.atlassian.com/manage-profile/security/api-tokens で生成）
api_token = "your-api-token"

[output]
# Markdown出力ディレクトリ（デフォルト: output/markdown）
markdown_dir = "output/markdown"
# HTML出力ディレクトリ（デフォルト: output/html）
html_dir = "output/html"
# 添付ファイル保存ディレクトリ（空の場合はmarkdown_dir内のページディレクトリに配置）
attachments_dir = ""
# 中間ファイル保存ディレクトリ（デフォルト: output/intermediate）
intermediate_dir = "output/intermediate"

[search]
# デフォルトのスペースキー（spaceコマンドでスペースキーを省略した場合に使用）
default_space_key = ""

[display]
# 変換時に無視するConfluenceマクロ名のリスト
ignored_macros = []

# 削除済みユーザーのマッピング（accountId -> 表示名）
# 例:
# [deletedUsers]
# "123456:abcdef" = "山田 太郎"
```

- [ ] **Step 2: 全テストと最終ビルドの確認**

```bash
go test ./...
go build ./...
```

期待: 全テスト `PASS`、ビルド成功

- [ ] **Step 3: TODO.md と CHANGELOG.md を更新する**

`TODO.md` に完了タスクとして追記、`CHANGELOG.md` に変更内容を記録する。

`TODO.md` の完了セクションに追加:
```markdown
- [x] Step 11: HTML出力追加と中間ファイル名称変更
  - config.go: XHTMLDir→IntermediateDir、HTMLDir 追加
  - IntermediateSaver（旧 XHTMLSaver）へのリネーム
  - Converter.ToHTML() 追加
  - HTMLWriter 新規実装
  - main.go への HTMLWriter ワイヤリング
```

`CHANGELOG.md` の `[Unreleased]` セクションに追加:
```markdown
### Changed
- `XHTMLSaver` → `IntermediateSaver` にリネーム（中間ファイルの役割を明示）
- CLIフラグ `--save-xhtml` → `--save-intermediate` に変更
- 設定キー `xhtml_dir` → `intermediate_dir` に変更、デフォルト `output/intermediate`

### Added
- `Converter.ToHTML()`: Confluence Storage Format → 標準 HTML ボディフラグメント変換
- `HTMLWriter`: HTML Living Standard 出力（`output/html/{SPACE}/{PAGE}/index.html`）
  - ページタイトル・メタ情報・ラベル・Confluence URL のヘッダー
  - コメントセクション
  - 最低限の CSS スタイリング（テーブル・コードブロック・パネル・details）
- 設定キー `html_dir` を追加（デフォルト: `output/html`）
- `page` / `space` / `convert` コマンドで Markdown と HTML を同時出力
```

- [ ] **Step 4: 最終コミット**

```bash
git add config.toml.example TODO.md CHANGELOG.md
git commit -m "chore: config.toml.example更新・TODO/CHANGELOG記録"
```

- [ ] **Step 5: プルリクエストを作成する**

```bash
git push -u origin feature/html-output
gh pr create \
  --title "feat: HTML出力追加と中間ファイル（IntermediateSaver）名称変更" \
  --body "$(cat <<'EOF'
## Summary

- Confluence Storage Format の中間ファイル関連識別子を `Intermediate` に統一（`XHTMLSaver` → `IntermediateSaver`、`--save-xhtml` → `--save-intermediate`、`xhtml_dir` → `intermediate_dir`）
- `Converter.ToHTML()` を追加（`preprocess()` の公開ラッパー）
- `HTMLWriter` を新規実装: Confluence Storage Format → HTML Living Standard（`output/html/{SPACE}/{PAGE}/index.html`）
- `page` / `space` / `convert` コマンドで Markdown と HTML を同時出力

## Test plan

- [ ] `go test ./...` が全 PASS
- [ ] `go build ./...` が成功
- [ ] `migConfluence page --page-id <ID>` で `output/html/` と `output/markdown/` 両方にファイルが生成される
- [ ] `migConfluence convert` で中間ファイルから HTML と Markdown が再生成される
- [ ] 生成された `index.html` をブラウザで開いて内容が読めることを確認

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```
