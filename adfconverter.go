package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ADFNode は Atlas Doc Format のドキュメントノード
type ADFNode struct {
	Type    string         `json:"type"`
	Attrs   map[string]any `json:"attrs,omitempty"`
	Content []ADFNode      `json:"content,omitempty"`
	Marks   []ADFMark      `json:"marks,omitempty"`
	Text    string         `json:"text,omitempty"`
}

// ADFMark はインラインフォーマットマーク
type ADFMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// adfRenderer は ADF ノードツリーを Markdown に変換する
type adfRenderer struct {
	attachmentMap map[string]string // media UUID → ファイル名
}

// convertADF は ADF JSON 文字列を Markdown に変換するエントリーポイント
func convertADF(adfJSON string, attachmentMap map[string]string) (string, error) {
	if adfJSON == "" {
		return "", nil
	}
	var root ADFNode
	if err := json.Unmarshal([]byte(adfJSON), &root); err != nil {
		return "", fmt.Errorf("ADF JSONパースエラー: %w", err)
	}
	r := &adfRenderer{attachmentMap: attachmentMap}
	return strings.TrimSpace(r.renderNode(root, 0)), nil
}

// renderNode はノードタイプに応じて変換を dispatch する
func (r *adfRenderer) renderNode(node ADFNode, indent int) string {
	switch node.Type {
	case "doc":
		return r.renderBlockChildren(node.Content, indent)
	case "paragraph":
		return r.renderInlineNodes(node.Content)
	case "text":
		return r.renderText(node)
	case "hardBreak":
		return "\n"
	case "heading":
		return r.renderHeading(node)
	case "bulletList":
		return r.renderBulletList(node, indent)
	case "orderedList":
		return r.renderOrderedList(node, indent)
	case "blockquote":
		return r.renderBlockquote(node)
	case "rule":
		return "---"
	case "codeBlock":
		return r.renderCodeBlock(node)
	case "panel":
		return r.renderPanel(node)
	case "table":
		return r.renderTable(node)
	case "taskList":
		return r.renderTaskList(node)
	case "decisionList":
		return r.renderDecisionList(node)
	case "expand", "nestedExpand":
		return r.renderExpand(node)
	case "mediaSingle", "mediaGroup":
		return r.renderMediaContainer(node)
	case "layoutSection":
		return r.renderBlockChildren(node.Content, indent)
	case "layoutColumn":
		return r.renderBlockChildren(node.Content, indent)
	case "extension", "inlineExtension":
		return r.renderExtension(node)
	case "bodiedExtension":
		return r.renderBodiedExtension(node)
	case "blockCard":
		return r.renderCard(node)
	case "embedCard":
		return r.renderEmbedCard(node)
	default:
		return ""
	}
}

// renderBlockChildren はブロック要素の子ノードを空行区切りで結合する
func (r *adfRenderer) renderBlockChildren(nodes []ADFNode, indent int) string {
	var parts []string
	for _, n := range nodes {
		if s := r.renderNode(n, indent); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "\n\n")
}

// renderInlineNodes はインライン要素を連結する
func (r *adfRenderer) renderInlineNodes(nodes []ADFNode) string {
	var sb strings.Builder
	for _, n := range nodes {
		sb.WriteString(r.renderInline(n))
	}
	return sb.String()
}

// renderInline はインライン要素を変換する
func (r *adfRenderer) renderInline(node ADFNode) string {
	switch node.Type {
	case "text":
		return r.renderText(node)
	case "hardBreak":
		return "\n"
	case "mention":
		return r.renderMention(node)
	case "emoji":
		return r.renderEmoji(node)
	case "status":
		return r.renderStatus(node)
	case "date":
		return r.renderDate(node)
	case "inlineCard":
		return r.renderCard(node)
	case "mediaInline":
		return r.renderMediaInline(node)
	default:
		return ""
	}
}

// renderText はテキストノードにマークを適用して変換する
func (r *adfRenderer) renderText(node ADFNode) string {
	text := node.Text
	// マークを逆順に適用（内側から外側へラップ）
	for i := len(node.Marks) - 1; i >= 0; i-- {
		mark := node.Marks[i]
		switch mark.Type {
		case "strong":
			text = "**" + text + "**"
		case "em":
			text = "*" + text + "*"
		case "code":
			text = "`" + text + "`"
		case "strike":
			text = "~~" + text + "~~"
		case "underline":
			text = "<u>" + text + "</u>"
		case "link":
			href := ""
			if mark.Attrs != nil {
				if h, ok := mark.Attrs["href"].(string); ok {
					href = convertInternalURL(h)
				}
			}
			text = "[" + text + "](" + href + ")"
		case "subsup":
			tag := "sup"
			if mark.Attrs != nil {
				if t, ok := mark.Attrs["type"].(string); ok && t == "sub" {
					tag = "sub"
				}
			}
			text = "<" + tag + ">" + text + "</" + tag + ">"
		// textColor, backgroundColor, annotation はテキストのみ保持
		}
	}
	return text
}

// internalURLRe は Confluence 内部ページ URL のパターン
var internalURLRe = regexp.MustCompile(`/wiki/spaces/[^/]+/pages/\d+/([^#?]+)`)

// convertInternalURL は絶対 Confluence URL を相対パスに変換する
func convertInternalURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	matches := internalURLRe.FindStringSubmatch(parsed.Path)
	if len(matches) < 2 {
		return rawURL
	}
	title, err := url.PathUnescape(matches[1])
	if err != nil {
		return rawURL
	}
	relPath := sanitizeFilename(title) + "/index.md"
	if parsed.Fragment != "" {
		relPath += "#" + parsed.Fragment
	}
	return relPath
}

func (r *adfRenderer) renderHeading(node ADFNode) string {
	level := 1
	if node.Attrs != nil {
		if l, ok := node.Attrs["level"].(float64); ok {
			level = int(l)
		}
	}
	prefix := strings.Repeat("#", level)
	return prefix + " " + r.renderInlineNodes(node.Content)
}

func (r *adfRenderer) renderBulletList(node ADFNode, indent int) string {
	var lines []string
	for _, item := range node.Content {
		if item.Type == "listItem" {
			lines = append(lines, r.renderListItem(item, indent, "- "))
		}
	}
	return strings.Join(lines, "\n")
}

func (r *adfRenderer) renderOrderedList(node ADFNode, indent int) string {
	var lines []string
	for i, item := range node.Content {
		if item.Type == "listItem" {
			lines = append(lines, r.renderListItem(item, indent, fmt.Sprintf("%d. ", i+1)))
		}
	}
	return strings.Join(lines, "\n")
}

func (r *adfRenderer) renderListItem(node ADFNode, indent int, prefix string) string {
	indentStr := strings.Repeat("  ", indent)
	var lines []string
	first := true
	for _, child := range node.Content {
		switch child.Type {
		case "paragraph":
			text := r.renderInlineNodes(child.Content)
			if first {
				lines = append(lines, indentStr+prefix+text)
				first = false
			} else {
				lines = append(lines, indentStr+"  "+text)
			}
		case "bulletList":
			lines = append(lines, r.renderBulletList(child, indent+1))
		case "orderedList":
			lines = append(lines, r.renderOrderedList(child, indent+1))
		}
	}
	return strings.Join(lines, "\n")
}

func (r *adfRenderer) renderBlockquote(node ADFNode) string {
	inner := r.renderBlockChildren(node.Content, 0)
	var sb strings.Builder
	for line := range strings.SplitSeq(inner, "\n") {
		if line == "" {
			sb.WriteString(">\n")
		} else {
			sb.WriteString("> " + line + "\n")
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (r *adfRenderer) renderCodeBlock(node ADFNode) string {
	lang := ""
	if node.Attrs != nil {
		if l, ok := node.Attrs["language"].(string); ok {
			lang = l
		}
	}
	var sb strings.Builder
	for _, child := range node.Content {
		if child.Type == "text" {
			sb.WriteString(child.Text)
		}
	}
	return "```" + lang + "\n" + sb.String() + "\n```"
}

func (r *adfRenderer) renderTable(node ADFNode) string {
	var sb strings.Builder
	firstRow := true
	var colCount int
	for _, row := range node.Content {
		if row.Type != "tableRow" {
			continue
		}
		if firstRow {
			colCount = len(row.Content)
		}
		sb.WriteString("|")
		for _, cell := range row.Content {
			cellText := r.renderTableCell(cell)
			sb.WriteString(" " + cellText + " |")
		}
		sb.WriteString("\n")
		if firstRow && colCount > 0 {
			sb.WriteString("|")
			for i := 0; i < colCount; i++ {
				sb.WriteString(" --- |")
			}
			sb.WriteString("\n")
			firstRow = false
		} else {
			firstRow = false
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (r *adfRenderer) renderTableCell(node ADFNode) string {
	var parts []string
	for _, child := range node.Content {
		text := strings.TrimSpace(r.renderNode(child, 0))
		text = strings.ReplaceAll(text, "\n", " ")
		text = strings.ReplaceAll(text, "|", "\\|")
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

func (r *adfRenderer) renderPanel(node ADFNode) string {
	panelType := "info"
	if node.Attrs != nil {
		if pt, ok := node.Attrs["panelType"].(string); ok {
			panelType = pt
		}
	}
	alertType := "NOTE"
	switch panelType {
	case "note":
		alertType = "WARNING"
	case "warning", "error":
		alertType = "CAUTION"
	case "success":
		alertType = "TIP"
	}
	inner := r.renderBlockChildren(node.Content, 0)
	var sb strings.Builder
	sb.WriteString("> [!" + alertType + "]\n")
	for line := range strings.SplitSeq(inner, "\n") {
		if line == "" {
			sb.WriteString(">\n")
		} else {
			sb.WriteString("> " + line + "\n")
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (r *adfRenderer) renderTaskList(node ADFNode) string {
	var lines []string
	for _, item := range node.Content {
		if item.Type != "taskItem" {
			continue
		}
		state := ""
		if item.Attrs != nil {
			if s, ok := item.Attrs["state"].(string); ok {
				state = s
			}
		}
		check := "- [ ] "
		if state == "DONE" {
			check = "- [x] "
		}
		lines = append(lines, check+r.renderInlineNodes(item.Content))
	}
	return strings.Join(lines, "\n")
}

func (r *adfRenderer) renderDecisionList(node ADFNode) string {
	var lines []string
	for _, item := range node.Content {
		if item.Type == "decisionItem" {
			lines = append(lines, "- "+r.renderInlineNodes(item.Content))
		}
	}
	return strings.Join(lines, "\n")
}

func (r *adfRenderer) renderExpand(node ADFNode) string {
	title := "詳細"
	if node.Attrs != nil {
		if ttl, ok := node.Attrs["title"].(string); ok && ttl != "" {
			title = ttl
		}
	}
	inner := r.renderBlockChildren(node.Content, 0)
	return "<details><summary>" + title + "</summary>\n\n" + inner + "\n\n</details>"
}

func (r *adfRenderer) renderStatus(node ADFNode) string {
	color := ""
	text := "STATUS"
	if node.Attrs != nil {
		if c, ok := node.Attrs["color"].(string); ok {
			color = c
		}
		if t, ok := node.Attrs["text"].(string); ok && t != "" {
			text = t
		}
	}
	emoji := statusColorEmoji(color)
	return emoji + "[" + text + "]"
}

func statusColorEmoji(color string) string {
	switch strings.ToLower(color) {
	case "green":
		return "🟢"
	case "yellow":
		return "🟡"
	case "red":
		return "🔴"
	case "blue":
		return "🔵"
	case "purple":
		return "🟣"
	default:
		return "⚫"
	}
}

func (r *adfRenderer) renderMention(node ADFNode) string {
	text := ""
	if node.Attrs != nil {
		if t, ok := node.Attrs["text"].(string); ok {
			text = t
		}
	}
	return "**" + text + "**"
}

func (r *adfRenderer) renderEmoji(node ADFNode) string {
	if node.Attrs == nil {
		return ""
	}
	if s, ok := node.Attrs["text"].(string); ok && s != "" {
		return s
	}
	if s, ok := node.Attrs["shortName"].(string); ok {
		return s
	}
	return ""
}

func (r *adfRenderer) renderDate(node ADFNode) string {
	if node.Attrs == nil {
		return ""
	}
	ts, ok := node.Attrs["timestamp"].(string)
	if !ok {
		return ""
	}
	ms, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return ts
	}
	t := time.UnixMilli(ms).UTC()
	return t.Format("2006-01-02")
}

func (r *adfRenderer) renderMediaContainer(node ADFNode) string {
	var parts []string
	for _, child := range node.Content {
		if child.Type == "media" {
			parts = append(parts, r.renderMedia(child))
		}
	}
	return strings.Join(parts, "\n")
}

func (r *adfRenderer) renderMedia(node ADFNode) string {
	if node.Attrs == nil {
		return ""
	}
	alt := ""
	if a, ok := node.Attrs["alt"].(string); ok {
		alt = a
	}
	mediaType, _ := node.Attrs["type"].(string)
	switch mediaType {
	case "external":
		u, _ := node.Attrs["url"].(string)
		return "![" + alt + "](" + u + ")"
	case "file":
		id, _ := node.Attrs["id"].(string)
		filename := "attachment-" + id
		if r.attachmentMap != nil {
			if f, ok := r.attachmentMap[id]; ok {
				filename = f
			}
		}
		return "![" + alt + "](" + filename + ")"
	default:
		return ""
	}
}

func (r *adfRenderer) renderMediaInline(node ADFNode) string {
	return r.renderMedia(node)
}

func (r *adfRenderer) renderExtension(node ADFNode) string {
	key := ""
	if node.Attrs != nil {
		if k, ok := node.Attrs["extensionKey"].(string); ok {
			key = k
		}
	}
	return "<!-- macro: " + key + " -->"
}

func (r *adfRenderer) renderBodiedExtension(node ADFNode) string {
	if len(node.Content) > 0 {
		return r.renderBlockChildren(node.Content, 0)
	}
	return r.renderExtension(node)
}

func (r *adfRenderer) renderCard(node ADFNode) string {
	u := ""
	if node.Attrs != nil {
		if v, ok := node.Attrs["url"].(string); ok {
			u = convertInternalURL(v)
		}
	}
	return "[" + u + "](" + u + ")"
}

func (r *adfRenderer) renderEmbedCard(node ADFNode) string {
	u := ""
	if node.Attrs != nil {
		if v, ok := node.Attrs["url"].(string); ok {
			u = v
		}
	}
	return "<!-- embed: " + u + " -->"
}
