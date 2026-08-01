package main

import (
	"encoding/json"
	"fmt"
	"html"
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
	return strings.TrimSpace(r.renderNode(root, "")), nil
}

// renderNode はノードタイプに応じて変換を dispatch する
func (r *adfRenderer) renderNode(node ADFNode, indent string) string {
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
		return r.renderTaskList(node, indent)
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
func (r *adfRenderer) renderBlockChildren(nodes []ADFNode, indent string) string {
	var parts []string
	for _, n := range nodes {
		if s := r.renderNode(n, indent); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "\n\n")
}

// mdDelimiters はデリミタ系マークの正規順序（先頭が最外側）と対応デリミタ
var mdDelimiters = []struct {
	mark  string
	delim string
}{
	{"strong", "**"},
	{"em", "*"},
	{"strike", "~~"},
}

// delimiterMarks は text ノードが持つデリミタ系マークを正規順序で返す。
// code/link/subsup/underline を含むノードと text 以外のノードはグループ化対象外（ok=false）
func delimiterMarks(node ADFNode) ([]string, bool) {
	if node.Type != "text" {
		return nil, false
	}
	present := map[string]bool{}
	for _, m := range node.Marks {
		switch m.Type {
		case "strong", "em", "strike":
			present[m.Type] = true
		case "code", "link", "subsup", "underline":
			return nil, false
		}
	}
	var marks []string
	for _, d := range mdDelimiters {
		if present[d.mark] {
			marks = append(marks, d.mark)
		}
	}
	return marks, true
}

// renderNonDelimiterText はデリミタ系マーク以外を適用したテキストを返す
// （デリミタ系はグループ単位で renderInlineNodes が適用する）
func (r *adfRenderer) renderNonDelimiterText(node ADFNode) string {
	return node.Text
}

// renderInlineNodes はインライン要素を連結する。
// 同じデリミタ系マークを持つ隣接テキストノードは1つのrunに結合してから囲む
func (r *adfRenderer) renderInlineNodes(nodes []ADFNode) string {
	delimOf := map[string]string{}
	for _, d := range mdDelimiters {
		delimOf[d.mark] = d.delim
	}
	var sb strings.Builder
	for i := 0; i < len(nodes); {
		marks, ok := delimiterMarks(nodes[i])
		if !ok {
			sb.WriteString(r.renderInline(nodes[i]))
			i++
			continue
		}
		sig := strings.Join(marks, ",")
		var group strings.Builder
		j := i
		for j < len(nodes) {
			m2, ok2 := delimiterMarks(nodes[j])
			if !ok2 || strings.Join(m2, ",") != sig {
				break
			}
			group.WriteString(r.renderNonDelimiterText(nodes[j]))
			j++
		}
		text := group.String()
		for k := len(marks) - 1; k >= 0; k-- {
			text = wrapDelimiter(text, delimOf[marks[k]])
		}
		sb.WriteString(text)
		i = j
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

// wrapDelimiter は前後の空白をデリミタの外側に保ったまま text をデリミタで囲む
func wrapDelimiter(text, delimiter string) string {
	core := strings.Trim(text, " \t\n")
	if core == "" {
		return text
	}
	start := strings.Index(text, core)
	lead := text[:start]
	trail := text[start+len(core):]
	return lead + delimiter + core + delimiter + trail
}

// renderText はテキストノードにマークを適用して変換する
func (r *adfRenderer) renderText(node ADFNode) string {
	text := node.Text
	// マークを逆順に適用（内側から外側へラップ）
	for i := len(node.Marks) - 1; i >= 0; i-- {
		mark := node.Marks[i]
		switch mark.Type {
		case "strong":
			text = wrapDelimiter(text, "**")
		case "em":
			text = wrapDelimiter(text, "*")
		case "code":
			text = "`" + text + "`"
		case "strike":
			text = wrapDelimiter(text, "~~")
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

func (r *adfRenderer) renderBulletList(node ADFNode, indent string) string {
	var lines []string
	for _, item := range node.Content {
		if item.Type == "listItem" {
			lines = append(lines, r.renderListItem(item, indent, "- "))
		}
	}
	return strings.Join(lines, "\n")
}

func (r *adfRenderer) renderOrderedList(node ADFNode, indent string) string {
	var lines []string
	for i, item := range node.Content {
		if item.Type == "listItem" {
			lines = append(lines, r.renderListItem(item, indent, fmt.Sprintf("%d. ", i+1)))
		}
	}
	return strings.Join(lines, "\n")
}

// renderListItem はリスト項目を変換する。子要素のインデントはマーカー幅から算出する
func (r *adfRenderer) renderListItem(node ADFNode, indent string, prefix string) string {
	childIndent := indent + strings.Repeat(" ", len(prefix))
	var lines []string
	first := true
	for _, child := range node.Content {
		switch child.Type {
		case "paragraph":
			text := r.renderInlineNodes(child.Content)
			if first {
				lines = append(lines, indent+prefix+text)
				first = false
			} else {
				lines = append(lines, childIndent+text)
			}
		case "bulletList":
			lines = append(lines, r.renderBulletList(child, childIndent))
		case "orderedList":
			lines = append(lines, r.renderOrderedList(child, childIndent))
		case "codeBlock":
			blockLines := strings.Split(r.renderCodeBlock(child), "\n")
			rest := blockLines
			if first {
				lines = append(lines, indent+prefix+blockLines[0])
				rest = blockLines[1:]
				first = false
			}
			for _, bl := range rest {
				lines = append(lines, childIndent+bl)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func (r *adfRenderer) renderBlockquote(node ADFNode) string {
	inner := r.renderBlockChildren(node.Content, "")
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

// tableCellData はグリッド展開後のセル情報
type tableCellData struct {
	content string
	align   string // "": 未指定, "center": 中央, "end": 右寄せ
}

// intAttr は attrs から正の整数属性を取得する（欠落・非数値・0以下は defaultVal）
func intAttr(node ADFNode, key string, defaultVal int) int {
	if node.Attrs != nil {
		if v, ok := node.Attrs[key].(float64); ok && int(v) > 0 {
			return int(v)
		}
	}
	return defaultVal
}

// cellAlignment はセル内段落の alignment マークから配置を返す（"" / "center" / "end"）
func cellAlignment(cell ADFNode) string {
	for _, child := range cell.Content {
		if child.Type != "paragraph" {
			continue
		}
		for _, m := range child.Marks {
			if m.Type == "alignment" && m.Attrs != nil {
				if a, ok := m.Attrs["align"].(string); ok {
					return a
				}
			}
		}
	}
	return ""
}

func (r *adfRenderer) renderTable(node ADFNode) string {
	adfRows := make([]ADFNode, 0, len(node.Content))
	for _, row := range node.Content {
		if row.Type == "tableRow" {
			adfRows = append(adfRows, row)
		}
	}
	if len(adfRows) == 0 {
		return ""
	}

	// 1行目に tableHeader が1つでもあればヘッダー行とみなす
	hasHeader := false
	for _, cell := range adfRows[0].Content {
		if cell.Type == "tableHeader" {
			hasHeader = true
			break
		}
	}

	// 仮想グリッド展開: rowspan/colspan の占有位置を空セルで確保して列ずれを防ぐ
	var grid [][]*tableCellData
	ensureCell := func(row, col int) {
		for len(grid) <= row {
			grid = append(grid, nil)
		}
		for len(grid[row]) <= col {
			grid[row] = append(grid[row], nil)
		}
	}
	for ri, row := range adfRows {
		col := 0
		for _, cell := range row.Content {
			if cell.Type != "tableCell" && cell.Type != "tableHeader" {
				continue
			}
			ensureCell(ri, col)
			for grid[ri][col] != nil {
				col++
				ensureCell(ri, col)
			}
			colspan := intAttr(cell, "colspan", 1)
			rowspan := intAttr(cell, "rowspan", 1)
			for dr := 0; dr < rowspan; dr++ {
				for dc := 0; dc < colspan; dc++ {
					ensureCell(ri+dr, col+dc)
					grid[ri+dr][col+dc] = &tableCellData{}
				}
			}
			grid[ri][col] = &tableCellData{
				content: r.renderCellContent(cell),
				align:   cellAlignment(cell),
			}
			col += colspan
		}
	}

	width := 0
	for _, row := range grid {
		if len(row) > width {
			width = len(row)
		}
	}
	if width == 0 {
		return ""
	}

	colAligns := make([]string, width)
	for i := 0; i < width; i++ {
		for _, row := range grid {
			if i < len(row) && row[i] != nil && row[i].align != "" {
				colAligns[i] = row[i].align
				break
			}
		}
	}

	var sb strings.Builder
	writeRow := func(row []*tableCellData) {
		sb.WriteString("|")
		for i := 0; i < width; i++ {
			content := ""
			if i < len(row) && row[i] != nil {
				content = row[i].content
			}
			sb.WriteString(" " + content + " |")
		}
		sb.WriteString("\n")
	}
	writeSeparator := func() {
		sb.WriteString("|")
		for i := 0; i < width; i++ {
			switch colAligns[i] {
			case "center":
				sb.WriteString(" :---: |")
			case "end":
				sb.WriteString(" ---: |")
			default:
				sb.WriteString(" --- |")
			}
		}
		sb.WriteString("\n")
	}

	rows := grid
	if hasHeader {
		writeRow(rows[0])
		writeSeparator()
		rows = rows[1:]
	} else {
		// ヘッダー無しテーブル: 空ヘッダー行を自動生成する
		writeRow(nil)
		writeSeparator()
	}
	for _, row := range rows {
		writeRow(row)
	}
	return strings.TrimRight(sb.String(), "\n")
}

// renderCellContent はセル内のブロック要素群を GFM セル用の1行文字列に変換する。
// GFM で表現できないブロック要素はセル内 HTML として埋め込む（migrate_jira-cloud 方式）。
func (r *adfRenderer) renderCellContent(cell ADFNode) string {
	var parts []string
	for _, child := range cell.Content {
		if s := r.renderCellBlock(child); s != "" {
			parts = append(parts, s)
		}
	}
	joined := strings.Join(parts, "<br>")
	// hardBreak 等が残した改行をすべて <br> に置換してから | をエスケープする
	joined = strings.ReplaceAll(joined, "\n", "<br>")
	return strings.ReplaceAll(joined, "|", "\\|")
}

// renderCellBlock はセル内の1ブロック要素を改行なしの文字列に変換する
func (r *adfRenderer) renderCellBlock(node ADFNode) string {
	switch node.Type {
	case "paragraph":
		return r.renderInlineNodes(node.Content)
	case "bulletList", "orderedList":
		return r.renderCellListHTML(node)
	case "blockquote":
		return "<blockquote>" + r.renderCellChildren(node.Content) + "</blockquote>"
	case "codeBlock":
		var sb strings.Builder
		for _, child := range node.Content {
			if child.Type == "text" {
				sb.WriteString(child.Text)
			}
		}
		code := html.EscapeString(sb.String())
		return "<code>" + strings.ReplaceAll(code, "\n", "<br>") + "</code>"
	case "taskList":
		var sb strings.Builder
		sb.WriteString("<ul>")
		for _, item := range node.Content {
			if item.Type != "taskItem" {
				continue
			}
			check := "☐ "
			if item.Attrs != nil {
				if s, ok := item.Attrs["state"].(string); ok && s == "DONE" {
					check = "☑ "
				}
			}
			sb.WriteString("<li>" + check + r.renderInlineNodes(item.Content) + "</li>")
		}
		sb.WriteString("</ul>")
		return sb.String()
	case "panel", "expand", "nestedExpand":
		return r.renderCellChildren(node.Content)
	case "extension":
		if s, ok := r.renderNestedTableHTML(node); ok {
			return s
		}
		return strings.TrimSpace(r.renderNode(node, ""))
	default:
		// 未知のブロック要素は通常変換の結果を採用（改行は renderCellContent が <br> 化する）
		return strings.TrimSpace(r.renderNode(node, ""))
	}
}

// renderCellChildren は子ブロック要素群を <br> 区切りで結合する
func (r *adfRenderer) renderCellChildren(nodes []ADFNode) string {
	var parts []string
	for _, child := range nodes {
		if s := r.renderCellBlock(child); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "<br>")
}

// renderCellListHTML は bulletList / orderedList をセル内 HTML リストに変換する
func (r *adfRenderer) renderCellListHTML(node ADFNode) string {
	tag := "ul"
	if node.Type == "orderedList" {
		tag = "ol"
	}
	var sb strings.Builder
	sb.WriteString("<" + tag + ">")
	for _, item := range node.Content {
		if item.Type != "listItem" {
			continue
		}
		sb.WriteString("<li>")
		for _, child := range item.Content {
			switch child.Type {
			case "paragraph":
				sb.WriteString(r.renderInlineNodes(child.Content))
			case "bulletList", "orderedList":
				sb.WriteString(r.renderCellListHTML(child))
			}
		}
		sb.WriteString("</li>")
	}
	sb.WriteString("</" + tag + ">")
	return sb.String()
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
	inner := r.renderBlockChildren(node.Content, "")
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

// renderTaskList はタスクリストを変換する。入れ子の taskList はインデントを深くして再帰する
func (r *adfRenderer) renderTaskList(node ADFNode, indent string) string {
	var lines []string
	for _, item := range node.Content {
		switch item.Type {
		case "taskItem":
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
			lines = append(lines, indent+check+r.renderInlineNodes(item.Content))
		case "taskList":
			lines = append(lines, r.renderTaskList(item, indent+"  "))
		}
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
	inner := r.renderBlockChildren(node.Content, "")
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
		return r.renderBlockChildren(node.Content, "")
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

// renderNestedTableHTML は nested-table 拡張ノードをセル内 <table> HTML に変換する。
// nested-table 以外の拡張・parameters.adf の欠落・パース失敗時は false を返す
// （呼び出し側が通常の extension 処理にフォールバックする）。
func (r *adfRenderer) renderNestedTableHTML(node ADFNode) (string, bool) {
	if node.Attrs == nil {
		return "", false
	}
	if k, _ := node.Attrs["extensionKey"].(string); k != "nested-table" {
		return "", false
	}
	params, _ := node.Attrs["parameters"].(map[string]any)
	if params == nil {
		return "", false
	}
	adfStr, _ := params["adf"].(string)
	if adfStr == "" {
		return "", false
	}
	var doc ADFNode
	if err := json.Unmarshal([]byte(adfStr), &doc); err != nil {
		return "", false
	}
	var table *ADFNode
	if doc.Type == "table" {
		table = &doc
	} else {
		for i := range doc.Content {
			if doc.Content[i].Type == "table" {
				table = &doc.Content[i]
				break
			}
		}
	}
	if table == nil {
		return "", false
	}
	return r.renderTableInlineHTML(*table), true
}

// renderTableRowHTML は tableRow ノードを1行の <tr> HTML に変換する
func (r *adfRenderer) renderTableRowHTML(row ADFNode) string {
	var sb strings.Builder
	sb.WriteString("<tr>")
	for _, cell := range row.Content {
		var tag string
		switch cell.Type {
		case "tableHeader":
			tag = "th"
		case "tableCell":
			tag = "td"
		default:
			continue
		}
		attrs := ""
		if cs := intAttr(cell, "colspan", 1); cs > 1 {
			attrs += fmt.Sprintf(` colspan="%d"`, cs)
		}
		if rs := intAttr(cell, "rowspan", 1); rs > 1 {
			attrs += fmt.Sprintf(` rowspan="%d"`, rs)
		}
		sb.WriteString("<" + tag + attrs + ">" + r.renderCellChildren(cell.Content) + "</" + tag + ">")
	}
	sb.WriteString("</tr>")
	return sb.String()
}

// renderTableInlineHTML は table ノードを1行の <table> HTML に変換する。
// GFM セル内はインライン文脈のため、結合は HTML の colspan/rowspan 属性でそのまま保持できる。
// ヘッダー行は <thead>、それ以外は <tbody> で明示的に囲む必要がある。
// これを省略すると、ブラウザが全 <tr> を1つの暗黙 <tbody> にまとめてしまい、
// ゼブラストライプ用CSS（tr:nth-child(2n) 等）がヘッダー行を数に含めてしまうため、
// データ行の縞模様がヘッダー行1つ分ずれて誤って着色される。
func (r *adfRenderer) renderTableInlineHTML(node ADFNode) string {
	rows := make([]ADFNode, 0, len(node.Content))
	for _, row := range node.Content {
		if row.Type == "tableRow" {
			rows = append(rows, row)
		}
	}

	hasHeader := false
	if len(rows) > 0 {
		for _, cell := range rows[0].Content {
			if cell.Type == "tableHeader" {
				hasHeader = true
				break
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("<table>")
	if hasHeader {
		sb.WriteString("<thead>" + r.renderTableRowHTML(rows[0]) + "</thead>")
		rows = rows[1:]
	}
	if len(rows) > 0 {
		sb.WriteString("<tbody>")
		for _, row := range rows {
			sb.WriteString(r.renderTableRowHTML(row))
		}
		sb.WriteString("</tbody>")
	}
	sb.WriteString("</table>")
	return sb.String()
}
