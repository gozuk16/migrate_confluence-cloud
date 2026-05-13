package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/strikethrough"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"golang.org/x/net/html"
)

// Converter はConfluence Storage Format（XHTML）をMarkdownに変換する
type Converter struct {
	ignoredMacros     []string
	deletedUsers      map[string]string
	unsupportedMu     sync.Mutex
	unsupportedMacros map[string]int
	unsupportedElems  map[string]int
}

// NewConverter は新しいConverterを作成する
func NewConverter(ignoredMacros []string, deletedUsers map[string]string) *Converter {
	return &Converter{
		ignoredMacros:     ignoredMacros,
		deletedUsers:      deletedUsers,
		unsupportedMacros: make(map[string]int),
		unsupportedElems:  make(map[string]int),
	}
}

func (c *Converter) trackUnsupportedMacro(name string) {
	c.unsupportedMu.Lock()
	defer c.unsupportedMu.Unlock()
	c.unsupportedMacros[name]++
}

func (c *Converter) trackUnsupportedElem(name string) {
	c.unsupportedMu.Lock()
	defer c.unsupportedMu.Unlock()
	c.unsupportedElems[name]++
}

// GetUnsupportedReport は未対応要素のレポートデータを返す
func (c *Converter) GetUnsupportedReport() (macros map[string]int, elems map[string]int) {
	c.unsupportedMu.Lock()
	defer c.unsupportedMu.Unlock()
	macros = make(map[string]int, len(c.unsupportedMacros))
	for k, v := range c.unsupportedMacros {
		macros[k] = v
	}
	elems = make(map[string]int, len(c.unsupportedElems))
	for k, v := range c.unsupportedElems {
		elems[k] = v
	}
	return macros, elems
}

// WriteUnsupportedReport は未対応要素の一覧をMarkdownファイルとして書き出す
func (c *Converter) WriteUnsupportedReport(path string) error {
	macros, elems := c.GetUnsupportedReport()
	if len(macros) == 0 && len(elems) == 0 {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("# 未対応のConfluence要素\n\n")
	sb.WriteString("> このファイルはmigConfluenceが自動生成したレポートです。\n")
	sb.WriteString("> 変換時に対応していなかったConfluence固有の要素の一覧です。\n\n")

	if len(macros) > 0 {
		sb.WriteString("## 未対応マクロ (ac:structured-macro)\n\n")
		sb.WriteString("| マクロ名 | 出現回数 | 処理内容 |\n")
		sb.WriteString("|---------|---------|----------|\n")

		keys := make([]string, 0, len(macros))
		for k := range macros {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("| `%s` | %d | rich-text-bodyの内容を出力、またはコメントとして保持 |\n", k, macros[k]))
		}
		sb.WriteString("\n")
	}

	if len(elems) > 0 {
		sb.WriteString("## 未対応要素\n\n")
		sb.WriteString("| 要素名 | 出現回数 | 処理内容 |\n")
		sb.WriteString("|-------|---------|----------|\n")

		keys := make([]string, 0, len(elems))
		for k := range elems {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("| `%s` | %d | 無視 |\n", k, elems[k]))
		}
	}

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// gfmAlertRe はエスケープされた [!TYPE] パターンを検出する
var gfmAlertRe = regexp.MustCompile(`(?m)^(> )\\?\[!(NOTE|WARNING|CAUTION|TIP)\\?\]$`)

// taskListPlugin はGFMタスクリストを処理するカスタムプラグイン
type taskListPlugin struct{}

func (p *taskListPlugin) Name() string { return "task-list" }

func (p *taskListPlugin) Init(conv *converter.Converter) error {
	conv.Register.RendererFor("ul", converter.TagTypeBlock,
		func(ctx converter.Context, w converter.Writer, n *html.Node) converter.RenderStatus {
			isTaskList := false
			for _, attr := range n.Attr {
				if attr.Key == "data-task-list" && attr.Val == "true" {
					isTaskList = true
					break
				}
			}
			if !isTaskList {
				return converter.RenderTryNext
			}

			w.WriteString("\n\n")
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				if child.Type != html.ElementNode || child.Data != "li" {
					continue
				}

				status := ""
				for _, attr := range child.Attr {
					if attr.Key == "data-task-status" {
						status = attr.Val
						break
					}
				}

				if status == "complete" {
					w.WriteString("- [x] ")
				} else {
					w.WriteString("- [ ] ")
				}

				var buf bytes.Buffer
				ctx.RenderChildNodes(ctx, &buf, child)
				content := bytes.TrimSpace(buf.Bytes())
				content = ctx.UnEscapeContent(content)
				w.Write(content)
				w.WriteRune('\n')
			}
			w.WriteString("\n")
			return converter.RenderSuccess
		}, converter.PriorityEarly)
	return nil
}

// Convert はStorage Format（XHTML）をMarkdownに変換する
func (c *Converter) Convert(xhtml string) (string, error) {
	if xhtml == "" {
		return "", nil
	}

	// Step 1: Confluence固有の要素を前処理（標準HTMLに変換）
	preprocessed, err := c.preprocess(xhtml)
	if err != nil {
		return "", fmt.Errorf("前処理エラー: %w", err)
	}

	// Step 2: html-to-markdown で標準HTML → Markdown変換
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			table.NewTablePlugin(),
			strikethrough.NewStrikethroughPlugin(),
			&taskListPlugin{},
		),
	)

	// GFM Alerts の後処理: \[!NOTE\] → [!NOTE]
	conv.Register.PostRenderer(func(ctx converter.Context, content []byte) []byte {
		return gfmAlertRe.ReplaceAll(content, []byte("$1[!$2]"))
	}, converter.PriorityLate+50)

	result, err := conv.ConvertString(preprocessed)
	if err != nil {
		return "", fmt.Errorf("Markdown変換エラー: %w", err)
	}

	return strings.TrimSpace(result), nil
}

// ToHTML は Confluence Storage Format を標準 HTML ボディフラグメントに変換する。
// Converter.preprocess() の公開ラッパー。HTMLWriter から利用される。
func (c *Converter) ToHTML(xhtml string) (string, error) {
	if xhtml == "" {
		return "", nil
	}
	return c.preprocess(xhtml)
}

// preprocess はConfluence固有要素を標準HTMLに変換する
func (c *Converter) preprocess(xhtml string) (string, error) {
	wrapped := "<html><body>" + xhtml + "</body></html>"

	doc, err := html.Parse(strings.NewReader(wrapped))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	c.processNode(&buf, doc)
	return buf.String(), nil
}

// processNode はHTMLノードを再帰的に処理する
func (c *Converter) processNode(buf *bytes.Buffer, node *html.Node) {
	switch node.Type {
	case html.TextNode:
		buf.WriteString(html.EscapeString(node.Data))
		return
	case html.ElementNode:
		c.processElement(buf, node)
		return
	case html.DocumentNode:
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			c.processNode(buf, child)
		}
		return
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		c.processNode(buf, child)
	}
}

// processElement はHTML要素を処理する
func (c *Converter) processElement(buf *bytes.Buffer, node *html.Node) {
	tagName := node.Data

	// Confluence固有要素の処理
	switch tagName {
	case "ac:structured-macro":
		c.processMacro(buf, node)
		return
	case "ac:image":
		c.processImage(buf, node)
		return
	case "ac:link":
		c.processLink(buf, node)
		return
	case "ac:task-list":
		c.processTaskList(buf, node)
		return
	case "ac:task", "ac:task-id":
		// ac:task-listの子要素なのでスキップ（processTaskListで処理）
		return
	case "ac:emoticon":
		c.processEmoticon(buf, node)
		return
	case "ac:inline-comment-marker":
		// インラインコメントマーカーはタグを除去して子要素を処理
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			c.processNode(buf, child)
		}
		return
	case "ac:placeholder":
		// プレースホルダーはコメントとして出力
		text := ""
		if node.FirstChild != nil && node.FirstChild.Type == html.TextNode {
			text = node.FirstChild.Data
		}
		if text != "" {
			buf.WriteString(fmt.Sprintf("<!-- Placeholder: %s -->", html.EscapeString(text)))
		}
		return
	case "ac:layout", "ac:layout-section", "ac:layout-cell":
		// レイアウト要素は子要素を順次処理（段組みを解除）
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			c.processNode(buf, child)
		}
		return
	case "ac:plain-text-body", "ac:plain-text-link-body":
		// テキストコンテンツをそのまま出力
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == html.TextNode {
				buf.WriteString(html.EscapeString(child.Data))
			}
		}
		return
	case "ac:rich-text-body", "ac:parameter", "ac:task-body", "ac:task-status":
		// 子要素を処理
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			c.processNode(buf, child)
		}
		return
	case "ri:attachment", "ri:page", "ri:user", "ri:url", "ri:space", "ri:blog-post", "ri:content-entity":
		// リンク処理の中で扱われるためここでは無視
		return
	case "fieldset":
		// fieldsetはスキップして子要素を処理
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			c.processNode(buf, child)
		}
		return
	case "html", "head", "body":
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			c.processNode(buf, child)
		}
		return
	}

	// 未知のac:*要素はトラッキング
	if strings.HasPrefix(tagName, "ac:") || strings.HasPrefix(tagName, "ri:") {
		c.trackUnsupportedElem(tagName)
		return
	}

	// 標準HTML要素はそのまま出力
	c.writeOpenTag(buf, node)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		c.processNode(buf, child)
	}
	c.writeCloseTag(buf, node)
}

// processMacro はac:structured-macroを処理する
func (c *Converter) processMacro(buf *bytes.Buffer, node *html.Node) {
	macroName := getAttr(node, "ac:name")

	// 無視するマクロのチェック
	for _, ignored := range c.ignoredMacros {
		if macroName == ignored {
			return
		}
	}

	switch macroName {
	case "code":
		c.processCodeMacro(buf, node)
	case "noformat":
		c.processNoformatMacro(buf, node)
	case "info":
		c.processAlertMacro(buf, node, "NOTE")
	case "note":
		c.processAlertMacro(buf, node, "WARNING")
	case "warning":
		c.processAlertMacro(buf, node, "CAUTION")
	case "tip":
		c.processAlertMacro(buf, node, "TIP")
	case "expand":
		c.processExpandMacro(buf, node)
	case "panel":
		c.processPanelMacro(buf, node)
	case "quote":
		c.processQuoteMacro(buf, node)
	case "status":
		c.processStatusMacro(buf, node)
	case "section", "column":
		// レイアウト: 子要素をそのまま処理（段組みを解除）
		body := findChild(node, "ac:rich-text-body")
		if body != nil {
			for child := body.FirstChild; child != nil; child = child.NextSibling {
				c.processNode(buf, child)
			}
		}
	case "anchor":
		anchorName := getMacroParam(node, "")
		if anchorName != "" {
			buf.WriteString(fmt.Sprintf(`<span id="%s"></span>`, html.EscapeString(anchorName)))
		}
	case "excerpt":
		// 抜粋定義: 内容を保持
		body := findChild(node, "ac:rich-text-body")
		if body != nil {
			for child := body.FirstChild; child != nil; child = child.NextSibling {
				c.processNode(buf, child)
			}
		}
	case "details":
		// Page Properties: 内容を保持
		body := findChild(node, "ac:rich-text-body")
		if body != nil {
			for child := body.FirstChild; child != nil; child = child.NextSibling {
				c.processNode(buf, child)
			}
		}
	case "jira":
		c.processJiraMacro(buf, node)
	case "excerpt-include":
		page := getMacroParam(node, "")
		if page == "" {
			page = getMacroParam(node, "page")
		}
		buf.WriteString(fmt.Sprintf("<!-- excerpt-include: %s -->", html.EscapeString(page)))
	case "include":
		page := getMacroParam(node, "")
		buf.WriteString(fmt.Sprintf("<!-- include: %s -->", html.EscapeString(page)))
	case "toc", "toc-zone":
		// 目次マクロは省略
	case "children", "pagetree":
		buf.WriteString(fmt.Sprintf("<!-- macro: %s -->", macroName))
	case "recently-updated", "blog-posts", "contentbylabel":
		buf.WriteString(fmt.Sprintf("<!-- macro: %s -->", macroName))
	case "widget":
		url := getMacroParam(node, "url")
		if url != "" {
			buf.WriteString(fmt.Sprintf("<!-- widget: %s -->", html.EscapeString(url)))
		} else {
			buf.WriteString("<!-- macro: widget -->")
		}
	case "gallery", "multimedia", "jirachart":
		buf.WriteString(fmt.Sprintf("<!-- macro: %s -->", macroName))
	default:
		// 未対応マクロ: rich-text-bodyの内容を出力、なければコメント
		body := findChild(node, "ac:rich-text-body")
		if body != nil {
			c.trackUnsupportedMacro(macroName)
			for child := body.FirstChild; child != nil; child = child.NextSibling {
				c.processNode(buf, child)
			}
		} else {
			c.trackUnsupportedMacro(macroName)
			buf.WriteString(fmt.Sprintf("<!-- macro: %s -->", macroName))
		}
	}
}

// processCodeMacro はコードブロックマクロを処理する
func (c *Converter) processCodeMacro(buf *bytes.Buffer, node *html.Node) {
	lang := getMacroParam(node, "language")
	title := getMacroParam(node, "title")

	codeContent := ""
	plainBody := findChild(node, "ac:plain-text-body")
	if plainBody != nil && plainBody.FirstChild != nil {
		codeContent = plainBody.FirstChild.Data
	}

	if title != "" {
		buf.WriteString(fmt.Sprintf("<p><strong>%s</strong></p>\n", html.EscapeString(title)))
	}
	buf.WriteString(fmt.Sprintf("<pre><code class=\"language-%s\">%s</code></pre>", lang, html.EscapeString(codeContent)))
}

// processNoformatMacro はnoformatマクロをコードブロックに変換する
func (c *Converter) processNoformatMacro(buf *bytes.Buffer, node *html.Node) {
	content := ""
	plainBody := findChild(node, "ac:plain-text-body")
	if plainBody != nil && plainBody.FirstChild != nil {
		content = plainBody.FirstChild.Data
	}
	buf.WriteString(fmt.Sprintf("<pre><code>%s</code></pre>", html.EscapeString(content)))
}

// processAlertMacro はinfo/note/warning/tipマクロをGFM Alertsに変換する
func (c *Converter) processAlertMacro(buf *bytes.Buffer, node *html.Node, alertType string) {
	title := getMacroParam(node, "title")

	buf.WriteString("<blockquote>\n")
	buf.WriteString(fmt.Sprintf("<p>[!%s]</p>\n", alertType))
	if title != "" {
		buf.WriteString(fmt.Sprintf("<p><strong>%s</strong></p>\n", html.EscapeString(title)))
	}

	body := findChild(node, "ac:rich-text-body")
	if body != nil {
		for child := body.FirstChild; child != nil; child = child.NextSibling {
			c.processNode(buf, child)
		}
	}

	buf.WriteString("</blockquote>\n")
}

// processExpandMacro はexpandマクロをdetails/summaryに変換する
func (c *Converter) processExpandMacro(buf *bytes.Buffer, node *html.Node) {
	title := getMacroParam(node, "title")
	if title == "" {
		title = "詳細"
	}

	buf.WriteString(fmt.Sprintf("<details><summary>%s</summary>\n", html.EscapeString(title)))

	body := findChild(node, "ac:rich-text-body")
	if body != nil {
		for child := body.FirstChild; child != nil; child = child.NextSibling {
			c.processNode(buf, child)
		}
	}

	buf.WriteString("</details>\n")
}

// processPanelMacro はpanelマクロを引用ブロックに変換する
func (c *Converter) processPanelMacro(buf *bytes.Buffer, node *html.Node) {
	title := getMacroParam(node, "title")

	buf.WriteString("<blockquote>\n")
	if title != "" {
		buf.WriteString(fmt.Sprintf("<p><strong>%s</strong></p>\n", html.EscapeString(title)))
	}

	body := findChild(node, "ac:rich-text-body")
	if body != nil {
		for child := body.FirstChild; child != nil; child = child.NextSibling {
			c.processNode(buf, child)
		}
	}

	buf.WriteString("</blockquote>\n")
}

// processQuoteMacro はquoteマクロを引用ブロックに変換する
func (c *Converter) processQuoteMacro(buf *bytes.Buffer, node *html.Node) {
	buf.WriteString("<blockquote>\n")
	body := findChild(node, "ac:rich-text-body")
	if body != nil {
		for child := body.FirstChild; child != nil; child = child.NextSibling {
			c.processNode(buf, child)
		}
	}
	buf.WriteString("</blockquote>\n")
}

// processStatusMacro はstatusマクロをテキスト表現に変換する
func (c *Converter) processStatusMacro(buf *bytes.Buffer, node *html.Node) {
	title := getMacroParam(node, "title")
	colour := getMacroParam(node, "colour")

	label := title
	if label == "" {
		label = "STATUS"
	}

	// 色に応じた絵文字プレフィックスを付ける
	colorEmoji := ""
	switch strings.ToLower(colour) {
	case "green":
		colorEmoji = "🟢 "
	case "yellow":
		colorEmoji = "🟡 "
	case "red":
		colorEmoji = "🔴 "
	case "blue":
		colorEmoji = "🔵 "
	case "purple":
		colorEmoji = "🟣 "
	case "grey", "gray":
		colorEmoji = "⚫ "
	}

	buf.WriteString(fmt.Sprintf("<strong>%s[%s]</strong>", colorEmoji, html.EscapeString(label)))
}

// processJiraMacro はjiraマクロをリンクに変換する
func (c *Converter) processJiraMacro(buf *bytes.Buffer, node *html.Node) {
	key := getMacroParam(node, "key")
	if key != "" {
		// シングルイシューリンク
		buf.WriteString(fmt.Sprintf("<strong>%s</strong>", html.EscapeString(key)))
		return
	}
	// JQLクエリの場合はコメントとして保持
	jql := getMacroParam(node, "jqlQuery")
	if jql != "" {
		buf.WriteString(fmt.Sprintf("<!-- jira-query: %s -->", html.EscapeString(jql)))
		return
	}
	buf.WriteString("<!-- macro: jira -->")
}

// processImage はac:imageを処理する
func (c *Converter) processImage(buf *bytes.Buffer, node *html.Node) {
	alt := getAttr(node, "ac:alt")
	if alt == "" {
		alt = getAttr(node, "ac:title")
	}

	width := getAttr(node, "ac:width")
	heightAttr := ""
	if width != "" {
		heightAttr = fmt.Sprintf(` width="%s"`, html.EscapeString(width))
	}

	// ri:attachmentを探す
	riAttachment := findChild(node, "ri:attachment")
	if riAttachment != nil {
		filename := getAttr(riAttachment, "ri:filename")
		if filename != "" {
			if heightAttr != "" {
				buf.WriteString(fmt.Sprintf(`<img src="%s" alt="%s"%s>`,
					html.EscapeString(filename), html.EscapeString(alt), heightAttr))
			} else {
				buf.WriteString(fmt.Sprintf(`<img src="%s" alt="%s">`,
					html.EscapeString(filename), html.EscapeString(alt)))
			}
			return
		}
	}

	// ri:urlを探す
	riURL := findChild(node, "ri:url")
	if riURL != nil {
		imgURL := getAttr(riURL, "ri:value")
		if imgURL != "" {
			if heightAttr != "" {
				buf.WriteString(fmt.Sprintf(`<img src="%s" alt="%s"%s>`,
					html.EscapeString(imgURL), html.EscapeString(alt), heightAttr))
			} else {
				buf.WriteString(fmt.Sprintf(`<img src="%s" alt="%s">`,
					html.EscapeString(imgURL), html.EscapeString(alt)))
			}
			return
		}
	}
}

// processLink はac:linkを処理する
func (c *Converter) processLink(buf *bytes.Buffer, node *html.Node) {
	anchor := getAttr(node, "ac:anchor")

	// リンクテキストの取得
	linkText := ""
	plainLinkBody := findChild(node, "ac:plain-text-link-body")
	if plainLinkBody != nil && plainLinkBody.FirstChild != nil {
		linkText = plainLinkBody.FirstChild.Data
	}
	richLinkBody := findChild(node, "ac:rich-text-link-body")
	if richLinkBody == nil {
		richLinkBody = findChild(node, "ac:link-body")
	}

	riPage := findChild(node, "ri:page")
	riUser := findChild(node, "ri:user")
	riURL := findChild(node, "ri:url")
	riAttachment := findChild(node, "ri:attachment")
	riBlogPost := findChild(node, "ri:blog-post")

	// ri:pageへのリンク（内部ページリンク、アンカー付き可）
	if riPage != nil {
		pageTitle := getAttr(riPage, "ri:content-title")
		if pageTitle == "" {
			pageTitle = "ページ"
		}
		if linkText == "" {
			linkText = pageTitle
		}
		pageFile := sanitizeFilename(pageTitle) + "/index.md"
		if anchor != "" {
			pageFile += "#" + anchor
		}
		buf.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`,
			html.EscapeString(pageFile), html.EscapeString(linkText)))
		return
	}

	// 現在ページのアンカーリンク
	if anchor != "" {
		if linkText == "" {
			linkText = anchor
		}
		buf.WriteString(fmt.Sprintf(`<a href="#%s">%s</a>`,
			html.EscapeString(anchor), html.EscapeString(linkText)))
		return
	}

	// ri:userへのリンク（ユーザーメンション）
	if riUser != nil {
		accountID := getAttr(riUser, "ri:account-id")
		displayName := accountID
		if c.deletedUsers != nil {
			if name, ok := c.deletedUsers[accountID]; ok {
				displayName = name
			}
		}
		buf.WriteString(fmt.Sprintf("<strong>@%s</strong>", html.EscapeString(displayName)))
		return
	}

	// ri:urlへのリンク（外部リンク）
	if riURL != nil {
		href := getAttr(riURL, "ri:value")
		if linkText == "" {
			linkText = href
		}
		buf.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`,
			html.EscapeString(href), html.EscapeString(linkText)))
		return
	}

	// ri:attachmentへのリンク（添付ファイルリンク）
	if riAttachment != nil {
		filename := getAttr(riAttachment, "ri:filename")
		if linkText == "" {
			linkText = filename
		}
		buf.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`,
			html.EscapeString(filename), html.EscapeString(linkText)))
		return
	}

	// ri:blog-postへのリンク
	if riBlogPost != nil {
		blogTitle := getAttr(riBlogPost, "ri:content-title")
		if linkText == "" {
			if blogTitle != "" {
				linkText = blogTitle
			} else {
				linkText = "ブログ投稿"
			}
		}
		if blogTitle != "" {
			blogFile := sanitizeFilename(blogTitle) + "/index.md"
			buf.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`,
				html.EscapeString(blogFile), html.EscapeString(linkText)))
		} else {
			buf.WriteString(html.EscapeString(linkText))
		}
		return
	}

	// リッチテキストボディがある場合はそのまま出力
	if richLinkBody != nil {
		for child := richLinkBody.FirstChild; child != nil; child = child.NextSibling {
			c.processNode(buf, child)
		}
		return
	}

	// その他: テキストのみ出力
	if linkText != "" {
		buf.WriteString(html.EscapeString(linkText))
	}
}

// processTaskList はac:task-listをGFMタスクリストに変換する
func (c *Converter) processTaskList(buf *bytes.Buffer, node *html.Node) {
	buf.WriteString("<ul data-task-list=\"true\">\n")
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "ac:task" {
			status := ""
			taskBody := ""

			for taskChild := child.FirstChild; taskChild != nil; taskChild = taskChild.NextSibling {
				if taskChild.Type != html.ElementNode {
					continue
				}
				switch taskChild.Data {
				case "ac:task-status":
					if taskChild.FirstChild != nil {
						status = taskChild.FirstChild.Data
					}
				case "ac:task-body":
					var bodyBuf bytes.Buffer
					for bodyChild := taskChild.FirstChild; bodyChild != nil; bodyChild = bodyChild.NextSibling {
						c.processNode(&bodyBuf, bodyChild)
					}
					taskBody = bodyBuf.String()
				}
			}

			buf.WriteString(fmt.Sprintf(`<li data-task-status="%s">%s</li>`+"\n", status, taskBody))
		}
	}
	buf.WriteString("</ul>\n")
}

// processEmoticon はac:emoticonを絵文字に変換する
func (c *Converter) processEmoticon(buf *bytes.Buffer, node *html.Node) {
	// ac:emoji-fallbackがあればそれを使う
	fallback := getAttr(node, "ac:emoji-fallback")
	if fallback != "" {
		buf.WriteString(html.EscapeString(fallback))
		return
	}

	// ac:emoji-idがあればUnicodeに変換（例: "1f600" → 😀）
	emojiID := getAttr(node, "ac:emoji-id")
	if emojiID != "" {
		// 数値IDはfallbackで処理済みなのでnameにフォールスルー
	}

	name := getAttr(node, "ac:name")
	emoji := emoticonToEmoji(name)
	buf.WriteString(emoji)
}

// emoticonToEmoji はConfluenceの絵文字名をUnicode絵文字に変換する
func emoticonToEmoji(name string) string {
	emoticonMap := map[string]string{
		"tick":           "✅",
		"cross":          "❌",
		"warning":        "⚠️",
		"information":    "ℹ️",
		"plus":           "➕",
		"minus":          "➖",
		"cheeky":         "😜",
		"laugh":          "😄",
		"wink":           "😉",
		"smile":          "😊",
		"sad":            "😢",
		"thumbs-up":      "👍",
		"thumbs-down":    "👎",
		"blue-star":      "⭐",
		"red-star":       "🔴",
		"yellow-star":    "⭐",
		"green-star":     "🟢",
		"light-on":       "💡",
		"light-off":      "💡",
		"yellow-message": "💬",
		"green-message":  "💬",
		"red-message":    "💬",
		"blue-message":   "💬",
		"heart":          "❤️",
		"question":       "❓",
		"start":          "⭐",
	}
	if emoji, ok := emoticonMap[name]; ok {
		return emoji
	}
	return ":" + name + ":"
}

// writeOpenTag は開きタグを書き出す
func (c *Converter) writeOpenTag(buf *bytes.Buffer, node *html.Node) {
	buf.WriteString("<")
	buf.WriteString(node.Data)
	for _, attr := range node.Attr {
		buf.WriteString(" ")
		if attr.Namespace != "" {
			buf.WriteString(attr.Namespace + ":")
		}
		buf.WriteString(attr.Key)
		buf.WriteString(`="`)
		buf.WriteString(html.EscapeString(attr.Val))
		buf.WriteString(`"`)
	}
	buf.WriteString(">")
}

// writeCloseTag は閉じタグを書き出す
func (c *Converter) writeCloseTag(buf *bytes.Buffer, node *html.Node) {
	voidElements := map[string]bool{
		"area": true, "base": true, "br": true, "col": true, "embed": true,
		"hr": true, "img": true, "input": true, "link": true, "meta": true,
		"param": true, "source": true, "track": true, "wbr": true,
	}
	if voidElements[node.Data] {
		return
	}
	buf.WriteString("</")
	buf.WriteString(node.Data)
	buf.WriteString(">")
}

// getAttr はHTMLノードから属性値を取得する
func getAttr(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		fullName := attr.Key
		if attr.Namespace != "" {
			fullName = attr.Namespace + ":" + attr.Key
		}
		if fullName == name || attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

// getMacroParam はac:structured-macroのac:parameterを取得する
func getMacroParam(node *html.Node, paramName string) string {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "ac:parameter" {
			name := getAttr(child, "ac:name")
			if name == paramName && child.FirstChild != nil {
				return child.FirstChild.Data
			}
		}
	}
	return ""
}

// findChild は指定タグ名の最初の子ノードを検索する
func findChild(node *html.Node, tagName string) *html.Node {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == tagName {
			return child
		}
	}
	return nil
}
