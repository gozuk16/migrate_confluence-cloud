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

// parseDateTime は RFC3339 または "2006-01-02T15:04:05.000Z" 形式の日時文字列をパースする
func parseDateTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02T15:04:05.000Z", s)
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
		if t, err := parseDateTime(page.Version.CreatedAt); err == nil {
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
				if t, err := parseDateTime(comment.Version.CreatedAt); err == nil {
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
