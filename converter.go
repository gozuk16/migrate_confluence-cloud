package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"golang.org/x/net/html"
)

// Converter はConfluence Storage Format（XHTML）をMarkdownに変換する
type Converter struct {
	ignoredMacros []string
	deletedUsers  map[string]string
}

// NewConverter は新しいConverterを作成する
func NewConverter(ignoredMacros []string, deletedUsers map[string]string) *Converter {
	return &Converter{
		ignoredMacros: ignoredMacros,
		deletedUsers:  deletedUsers,
	}
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
		),
	)
	result, err := conv.ConvertString(preprocessed)
	if err != nil {
		return "", fmt.Errorf("Markdown変換エラー: %w", err)
	}

	return strings.TrimSpace(result), nil
}

// preprocess はConfluence固有要素を標準HTMLに変換する
func (c *Converter) preprocess(xhtml string) (string, error) {
	// HTMLパーサーが扱えるようにwrapする
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

	// その他のノードは子要素を処理
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
	case "ac:task":
		// ac:task-listの子要素なのでスキップ（processTaskListで処理）
		return
	case "ac:emoticon":
		c.processEmoticon(buf, node)
		return
	case "ac:plain-text-body", "ac:plain-text-link-body":
		// テキストコンテンツをそのまま出力
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == html.TextNode {
				buf.WriteString(html.EscapeString(child.Data))
			}
		}
		return
	case "ac:rich-text-body", "ac:parameter":
		// 子要素を処理
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			c.processNode(buf, child)
		}
		return
	case "ri:attachment", "ri:page", "ri:user", "ri:url", "ri:space":
		// リンク処理の中で扱われるためここでは無視
		return
	// body, html, headタグはスキップして子要素を処理
	case "html", "head", "body":
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			c.processNode(buf, child)
		}
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
	case "info", "note":
		c.processAlertMacro(buf, node, "NOTE")
	case "warning":
		c.processAlertMacro(buf, node, "WARNING")
	case "tip":
		c.processAlertMacro(buf, node, "TIP")
	case "expand":
		c.processExpandMacro(buf, node)
	case "panel":
		c.processPanelMacro(buf, node)
	case "status":
		c.processStatusMacro(buf, node)
	case "toc", "toc-zone":
		// 目次マクロは省略
	default:
		// 未対応マクロ: rich-text-bodyの内容を出力、なければコメントとして残す
		body := findChild(node, "ac:rich-text-body")
		if body != nil {
			for child := body.FirstChild; child != nil; child = child.NextSibling {
				c.processNode(buf, child)
			}
		} else {
			buf.WriteString(fmt.Sprintf("<!-- macro: %s -->", macroName))
		}
	}
}

// processCodeMacro はコードブロックマクロを処理する
func (c *Converter) processCodeMacro(buf *bytes.Buffer, node *html.Node) {
	lang := getMacroParam(node, "language")
	title := getMacroParam(node, "title")

	// コードコンテンツの取得
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

// processAlertMacro はinfo/note/warning/tipマクロをGFM Alertsに変換する
func (c *Converter) processAlertMacro(buf *bytes.Buffer, node *html.Node, alertType string) {
	title := getMacroParam(node, "title")

	buf.WriteString(fmt.Sprintf("<blockquote>\n<p><strong>[%s]</strong>", alertType))
	if title != "" {
		buf.WriteString(fmt.Sprintf(" %s", html.EscapeString(title)))
	}
	buf.WriteString("</p>\n")

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

// processStatusMacro はstatusマクロをテキスト表現に変換する
func (c *Converter) processStatusMacro(buf *bytes.Buffer, node *html.Node) {
	title := getMacroParam(node, "title")
	if title == "" {
		title = "STATUS"
	}
	buf.WriteString(fmt.Sprintf("<strong>[%s]</strong>", html.EscapeString(title)))
}

// processImage はac:imageを処理する
func (c *Converter) processImage(buf *bytes.Buffer, node *html.Node) {
	alt := getAttr(node, "ac:alt")
	if alt == "" {
		alt = getAttr(node, "ac:title")
	}

	// ri:attachmentを探す
	riAttachment := findChild(node, "ri:attachment")
	if riAttachment != nil {
		filename := getAttr(riAttachment, "ri:filename")
		if filename != "" {
			buf.WriteString(fmt.Sprintf(`<img src="%s" alt="%s">`,
				html.EscapeString(filename), html.EscapeString(alt)))
			return
		}
	}

	// ri:urlを探す
	riURL := findChild(node, "ri:url")
	if riURL != nil {
		imgURL := getAttr(riURL, "ri:value")
		if imgURL != "" {
			buf.WriteString(fmt.Sprintf(`<img src="%s" alt="%s">`,
				html.EscapeString(imgURL), html.EscapeString(alt)))
			return
		}
	}
}

// processLink はac:linkを処理する
func (c *Converter) processLink(buf *bytes.Buffer, node *html.Node) {
	// リンクテキストの取得
	linkText := ""
	plainLinkBody := findChild(node, "ac:plain-text-link-body")
	if plainLinkBody != nil && plainLinkBody.FirstChild != nil {
		linkText = plainLinkBody.FirstChild.Data
	}
	richLinkBody := findChild(node, "ac:rich-text-link-body")

	// ri:pageへのリンク（内部ページリンク）
	riPage := findChild(node, "ri:page")
	if riPage != nil {
		pageTitle := getAttr(riPage, "ri:content-title")
		if pageTitle == "" {
			pageTitle = "ページ"
		}
		if linkText == "" {
			linkText = pageTitle
		}
		// ページタイトルをファイル名に変換（スペースをハイフンに）
		pageFile := strings.ReplaceAll(pageTitle, " ", "-") + "/index.md"
		buf.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`,
			html.EscapeString(pageFile), html.EscapeString(linkText)))
		return
	}

	// ri:userへのリンク（ユーザーメンション）
	riUser := findChild(node, "ri:user")
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
	riURL := findChild(node, "ri:url")
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
	riAttachment := findChild(node, "ri:attachment")
	if riAttachment != nil {
		filename := getAttr(riAttachment, "ri:filename")
		if linkText == "" {
			linkText = filename
		}
		buf.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`,
			html.EscapeString(filename), html.EscapeString(linkText)))
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

// processTaskList はac:task-listをMarkdownチェックリストに変換する
func (c *Converter) processTaskList(buf *bytes.Buffer, node *html.Node) {
	buf.WriteString("<ul>\n")
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

			checked := ""
			if status == "complete" {
				checked = " checked"
			}
			buf.WriteString(fmt.Sprintf(`<li><input type="checkbox"%s disabled> %s</li>`+"\n", checked, taskBody))
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

	// ac:nameで変換
	name := getAttr(node, "ac:name")
	emoji := emoticonToEmoji(name)
	buf.WriteString(emoji)
}

// emoticonToEmoji はConfluenceの絵文字名をUnicode絵文字に変換する
func emoticonToEmoji(name string) string {
	emoticonMap := map[string]string{
		"tick":            "✅",
		"cross":           "❌",
		"warning":         "⚠️",
		"information":     "ℹ️",
		"plus":            "➕",
		"minus":           "➖",
		"cheeky":          "😜",
		"laugh":           "😄",
		"wink":            "😉",
		"smile":           "😊",
		"sad":             "😢",
		"thumbs-up":       "👍",
		"thumbs-down":     "👎",
		"blue-star":       "⭐",
		"red-star":        "🔴",
		"yellow-star":     "⭐",
		"green-star":      "🟢",
		"light-on":        "💡",
		"light-off":       "💡",
		"yellow-message":  "💬",
		"green-message":   "💬",
		"red-message":     "💬",
		"blue-message":    "💬",
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
	// void要素は閉じタグ不要
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
		// 名前空間付き属性の処理（例: "ac:name" → namespace="ac", key="name"）
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
