package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ADFNode は Atlas Doc Format のドキュメントノード
type ADFNode struct {
	Type    string                 `json:"type"`
	Attrs   map[string]interface{} `json:"attrs,omitempty"`
	Content []ADFNode              `json:"content,omitempty"`
	Marks   []ADFMark              `json:"marks,omitempty"`
	Text    string                 `json:"text,omitempty"`
}

// ADFMark はインラインフォーマットマーク
type ADFMark struct {
	Type  string                 `json:"type"`
	Attrs map[string]interface{} `json:"attrs,omitempty"`
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
