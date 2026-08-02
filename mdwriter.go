package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MDWriter はMarkdownファイルの出力を管理する
type MDWriter struct {
	outputDir   string
	converter   *Converter
	resolveUser func(accountID string) string // accountId → 表示名。nil の場合は解決しない
}

// NewMDWriter は新しいMDWriterを作成する
func NewMDWriter(outputDir string, converter *Converter, resolveUser func(accountID string) string) *MDWriter {
	return &MDWriter{
		outputDir:   outputDir,
		converter:   converter,
		resolveUser: resolveUser,
	}
}

// WritePage はページをMarkdownファイルとして書き出す
func (w *MDWriter) WritePage(page *Page, spaceKey, spaceTitle, parentTitle string, labels []Label, comments []Comment, attachments []Attachment) error {
	// 出力ディレクトリ: outputDir/SPACE_KEY/PAGE_TITLE/
	safeTitle := sanitizeFilename(page.Title)
	pageDir := filepath.Join(w.outputDir, spaceKey, safeTitle)
	if err := os.MkdirAll(pageDir, 0755); err != nil {
		return fmt.Errorf("出力ディレクトリの作成に失敗しました: %w", err)
	}

	// Markdown本文の生成
	content, err := w.generateContent(page, spaceKey, spaceTitle, parentTitle, labels, comments, attachments)
	if err != nil {
		return fmt.Errorf("Markdownコンテンツ生成エラー: %w", err)
	}

	// index.mdに書き出し
	mdPath := filepath.Join(pageDir, "index.md")
	if err := os.WriteFile(mdPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("Markdownファイル書き出しエラー: %w", err)
	}

	return nil
}

// generateContent はMarkdownコンテンツ全体を生成する
func (w *MDWriter) generateContent(page *Page, spaceKey, spaceTitle, parentTitle string, labels []Label, comments []Comment, attachments []Attachment) (string, error) {
	var sb strings.Builder

	// Front Matter
	sb.WriteString(w.generateFrontMatter(page, spaceKey, spaceTitle, parentTitle, labels))

	// ページ本文
	attachmentMap := buildAttachmentMap(attachments)
	bodyMarkdown, err := w.converter.ConvertADF(page.Body.AtlasDocFormat.Value, attachmentMap)
	if err != nil {
		// 変換エラーの場合は生 ADF JSON をコードブロックとして出力
		sb.WriteString("\n<!-- 変換エラーのため元のADF JSONを表示します -->\n")
		sb.WriteString("```json\n")
		sb.WriteString(page.Body.AtlasDocFormat.Value)
		sb.WriteString("\n```\n")
	} else {
		sb.WriteString("\n")
		sb.WriteString(bodyMarkdown)
		sb.WriteString("\n")
	}

	// 添付ファイルセクション
	if len(attachments) > 0 {
		sb.WriteString("\n## 添付ファイル\n\n")
		for _, att := range attachments {
			if IsImageFile(att.Title) {
				sb.WriteString(fmt.Sprintf("![%s](%s)\n\n", att.Title, att.Title))
			} else {
				sb.WriteString(fmt.Sprintf("- [%s](%s)\n", att.Title, att.Title))
			}
		}
	}

	// コメントセクション
	if len(comments) > 0 {
		sb.WriteString("\n## コメント\n\n")
		// numbering は各深さのカレント番号を保持する（numbering[0]=トップレベルコメント番号,
		// numbering[1]=直近の親に対する返信番号, ...）。Depth が浅くなったら深い側をリセットする。
		numbering := []int{}
		for _, comment := range comments {
			depth := comment.Depth
			if depth < 0 {
				depth = 0
			}
			for len(numbering) <= depth {
				numbering = append(numbering, 0)
			}
			numbering[depth]++
			numbering = numbering[:depth+1]

			parts := make([]string, len(numbering))
			for i, n := range numbering {
				parts[i] = fmt.Sprintf("%d", n)
			}
			label := strings.Join(parts, "-")

			headingLevel := 3 + depth
			if headingLevel > 6 {
				headingLevel = 6
			}
			heading := strings.Repeat("#", headingLevel)

			authorID := comment.Version.AuthorID
			if authorID == "" {
				authorID = "unknown"
			}
			authorName := authorID
			if w.resolveUser != nil && authorID != "unknown" {
				authorName = w.resolveUser(authorID)
			}
			createdAt := formatDate(comment.Version.CreatedAt)

			// Depth>=1 のリプライは div でインデント
			if depth > 0 {
				marginLeft := depth * 2
				sb.WriteString(fmt.Sprintf("<div style=\"margin-left: %dem\">\n\n", marginLeft))
			}

			sb.WriteString(fmt.Sprintf("%s コメント %s\n\n", heading, label))
			sb.WriteString(fmt.Sprintf("**投稿者:** %s  \n", authorName))
			sb.WriteString(fmt.Sprintf("**日時:** %s\n\n", createdAt))

			commentMarkdown, err := w.converter.Convert(comment.Body.Storage.Value)
			if err != nil {
				sb.WriteString(fmt.Sprintf("<!-- 変換エラー: %v -->\n", err))
			} else {
				sb.WriteString(commentMarkdown)
				sb.WriteString("\n\n")
			}

			// Depth>=1 のリプライは div をクローズ
			if depth > 0 {
				sb.WriteString("</div>\n\n")
			}
		}
	}

	return sb.String(), nil
}

// generateFrontMatter はHugo Front Matter (TOML形式) を生成する
func (w *MDWriter) generateFrontMatter(page *Page, spaceKey, spaceTitle, parentTitle string, labels []Label) string {
	var sb strings.Builder

	sb.WriteString("+++\n")
	sb.WriteString(fmt.Sprintf("title = %q\n", page.Title))

	// 作成日時
	if page.Version.CreatedAt != "" {
		date := formatDateForFrontMatter(page.Version.CreatedAt)
		sb.WriteString(fmt.Sprintf("date = %q\n", date))
		sb.WriteString(fmt.Sprintf("lastmod = %q\n", date))
	}

	sb.WriteString(fmt.Sprintf("space = %q\n", spaceKey))
	if spaceTitle != "" {
		sb.WriteString(fmt.Sprintf("space_title = %q\n", spaceTitle))
	}
	sb.WriteString(fmt.Sprintf("page_id = %q\n", page.ID))

	if parentTitle != "" {
		sb.WriteString(fmt.Sprintf("parent = %q\n", parentTitle))
	}

	// ラベル
	if len(labels) > 0 {
		labelNames := make([]string, 0, len(labels))
		for _, l := range labels {
			labelNames = append(labelNames, fmt.Sprintf("%q", l.Name))
		}
		sb.WriteString(fmt.Sprintf("labels = [%s]\n", strings.Join(labelNames, ", ")))
	}

	// Confluence WebUI URL
	if page.Links.WebUI != "" {
		sb.WriteString(fmt.Sprintf("confluence_url = %q\n", page.Links.WebUI))
	}

	sb.WriteString("+++\n")

	return sb.String()
}

// formatDate は日時文字列を読みやすい形式に変換する
func formatDate(dateStr string) string {
	if dateStr == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		// RFC3339でパースできない場合はそのまま返す
		// Confluenceの日付形式 "2024-01-01T00:00:00.000Z" を試みる
		t, err = time.Parse("2006-01-02T15:04:05.000Z", dateStr)
		if err != nil {
			return dateStr
		}
	}
	return t.Format("2006-01-02 15:04:05")
}

// formatDateForFrontMatter はFront Matter用の日時形式に変換する
func formatDateForFrontMatter(dateStr string) string {
	if dateStr == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000Z", dateStr)
		if err != nil {
			return dateStr
		}
	}
	return t.UTC().Format(time.RFC3339)
}

// buildAttachmentMap は添付ファイル一覧から UUID → ファイル名マップを構築する
func buildAttachmentMap(attachments []Attachment) map[string]string {
	if len(attachments) == 0 {
		return nil
	}
	m := make(map[string]string, len(attachments))
	for _, a := range attachments {
		m[a.ID] = a.Title
	}
	return m
}
