package main

import "strings"

// PageProperty は Confluence のページプロパティ（details マクロ）の1項目。
// Value は本文の表のセルと同じ形式（Markdown とインライン HTML の混在）
type PageProperty struct {
	Key   string
	Value string
}

// collectPageProperties は details マクロの中の表から「1つ目のセル → 2つ目のセル」を項目として集める。
// 同じ項目名は先に出てきた方を採用する（1ページに details が複数ある場合も同様）。
func (r *adfRenderer) collectPageProperties(nodes []ADFNode) {
	for _, node := range nodes {
		if node.Type != "table" {
			r.collectPageProperties(node.Content)
			continue
		}
		for _, row := range node.Content {
			if row.Type != "tableRow" || len(row.Content) < 2 {
				continue
			}
			key := strings.TrimSpace(plainText(row.Content[0]))
			if key == "" {
				continue
			}
			if r.seenProps == nil {
				r.seenProps = map[string]bool{}
			}
			if r.seenProps[key] {
				continue
			}
			r.seenProps[key] = true
			value := strings.ReplaceAll(r.renderCellChildren(row.Content[1].Content), "\n", "<br>")
			r.properties = append(r.properties, PageProperty{Key: key, Value: strings.TrimSpace(value)})
		}
	}
}

// plainText はノード配下のテキストを書式なしで連結する（改行は空白にする）
func plainText(node ADFNode) string {
	switch node.Type {
	case "text":
		return node.Text
	case "hardBreak":
		return " "
	}
	var sb strings.Builder
	for _, child := range node.Content {
		sb.WriteString(plainText(child))
	}
	return sb.String()
}
