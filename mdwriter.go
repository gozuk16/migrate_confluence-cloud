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

// WriteFolder はConfluenceのフォルダを「レンダリングされないページ」スタブとして書き出す。
// Hugoの build.render = "never" によりURLもHTMLも生成されないが、
// list = "always" によりテンプレートのページ一覧には現れるため、サイドバーのツリーに使える。
func (w *MDWriter) WriteFolder(folder *Folder, spaceKey, spaceTitle string) error {
	folderDir, err := w.folderDir(folder, spaceKey)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(folderDir, 0755); err != nil {
		return fmt.Errorf("フォルダディレクトリの作成に失敗しました: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("+++\n")
	sb.WriteString(fmt.Sprintf("title = %q\n", folder.Title))
	sb.WriteString(fmt.Sprintf("space = %q\n", spaceKey))
	if spaceTitle != "" {
		sb.WriteString(fmt.Sprintf("space_title = %q\n", spaceTitle))
	}
	sb.WriteString(fmt.Sprintf("page_id = %q\n", folder.ID))
	if folder.ParentID != "" {
		sb.WriteString(fmt.Sprintf("parent_id = %q\n", folder.ParentID))
	}
	sb.WriteString(fmt.Sprintf("weight = %d\n", frontMatterWeight(folder.Position)))
	sb.WriteString("is_folder = true\n")
	sb.WriteString("[build]\n")
	sb.WriteString("  render = \"never\"\n")
	sb.WriteString("  list = \"always\"\n")
	sb.WriteString("+++\n")

	mdPath := filepath.Join(folderDir, "index.md")
	if err := os.WriteFile(mdPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("フォルダスタブ書き出しエラー: %w", err)
	}

	return nil
}

// folderDir はフォルダスタブの出力先ディレクトリを返す。
// 同名ディレクトリに既にページやフォルダが存在する場合、
// 安全性を最優先に以下の順で判定する:
// 1. ファイルが存在しない → プライマリディレクトリを使用
// 2. ファイルが自分自身のスタブ（同じ page_id で is_folder=true） → プライマリディレクトリを再利用（冪等性）
// 3. ファイルが別コンテンツ → フォールバック "<タイトル>_<フォルダID>" を試す
// 4. フォールバックも使えない → エラー（データ破損を避けるため黙って上書きしない）
func (w *MDWriter) folderDir(folder *Folder, spaceKey string) (string, error) {
	safeTitle := sanitizeFilename(folder.Title)
	dir := filepath.Join(w.outputDir, spaceKey, safeTitle)

	data, err := os.ReadFile(filepath.Join(dir, "index.md"))
	if err != nil {
		if os.IsNotExist(err) {
			return dir, nil // 未使用のディレクトリ名なのでそのまま使う
		}
		return "", fmt.Errorf("既存ファイルの確認に失敗しました (%s): %w", dir, err)
	}

	// 自分自身のスタブなら同じディレクトリを再利用する（再実行時の冪等性）
	if isSelfFolderStub(data, folder.ID) {
		return dir, nil
	}

	// プライマリは使えない。フォールバック先を試す
	fallbackDir := filepath.Join(w.outputDir, spaceKey, fmt.Sprintf("%s_%s", safeTitle, folder.ID))
	fallbackData, err := os.ReadFile(filepath.Join(fallbackDir, "index.md"))
	if err != nil {
		if os.IsNotExist(err) {
			return fallbackDir, nil // フォールバックが未使用なので使用可
		}
		return "", fmt.Errorf("フォールバックディレクトリの確認に失敗しました (%s): %w", fallbackDir, err)
	}

	// フォールバック先も存在する。自分自身のスタブか確認
	if isSelfFolderStub(fallbackData, folder.ID) {
		return fallbackDir, nil
	}

	// プライマリもフォールバックも他人のコンテンツを持っている
	return "", fmt.Errorf("フォルダ %q を出力できるディレクトリがありません: プライマリ %q と フォールバック %q の両方にコンテンツが存在しています", folder.Title, dir, fallbackDir)
}

// isSelfFolderStub は、与えられたファイルデータが「このフォルダ自身のスタブ」かどうかを判定する。
// フロントマター内に page_id が一致し、かつ is_folder = true がある場合のみ true を返す。
// これにより、他の無関係なページの本文中に偶然 page_id という文字列が含まれていても
// 誤認を防ぐことができる。
func isSelfFolderStub(data []byte, folderID string) bool {
	frontMatter := extractFrontMatter(data)
	if frontMatter == "" {
		return false // フロントマターがない、または壊れている
	}

	// フロントマター内に page_id と is_folder = true の両方が含まれるか確認
	pageIDMatch := fmt.Sprintf("page_id = %q", folderID)
	return strings.Contains(frontMatter, pageIDMatch) && strings.Contains(frontMatter, "is_folder = true")
}

// extractFrontMatter はMarkdownファイルのフロントマター（最初の +++ から次の +++ まで）を抽出する。
// フロントマターが無い場合や壊れている場合は空文字列を返す。
func extractFrontMatter(data []byte) string {
	s := string(data)
	if !strings.HasPrefix(s, "+++\n") {
		return "" // フロントマターが存在しない
	}

	// 最初の "+++" をスキップして、次の "+++" を探す
	rest := s[4:] // "+++\n" の4文字をスキップ
	idx := strings.Index(rest, "\n+++")
	if idx == -1 {
		return "" // 閉じの "+++" が見つからない（壊れたファイル）
	}

	return rest[:idx] // フロントマター内容（最後の改行含む）
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

	// 階層構造用: 親のID（ページ・フォルダ共通）とサイドバーの並び順
	if page.ParentID != "" {
		sb.WriteString(fmt.Sprintf("parent_id = %q\n", page.ParentID))
	}
	sb.WriteString(fmt.Sprintf("weight = %d\n", frontMatterWeight(page.Position)))

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

// frontMatterWeight は Confluence の position を Hugo の weight に変換する。
// Hugo の weight 昇順ソートでは 0 が先頭に来てしまうため必ず 1 以上にし、
// position が取得できない場合は 9999 として末尾に寄せる。
func frontMatterWeight(position *int) int {
	if position == nil {
		return 9999
	}
	weight := *position + 1
	if weight < 1 {
		weight = 1
	}
	return weight
}
