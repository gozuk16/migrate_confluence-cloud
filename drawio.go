package main

import "strings"

// draw.io（Forge アプリ）のマクロは extensionType が com.atlassian.ecosystem で、
// extensionKey が「アプリID/環境ID/static/drawio」（図）または「…/static/drawio-sketch」（ボード）になる。
// 図の本体はページの添付ファイル（diagramName）で、draw.io が同じ名前に .png を付けたプレビュー画像も添付する。
const forgeExtensionType = "com.atlassian.ecosystem"

var drawioMacroSuffixes = []string{"/static/drawio", "/static/drawio-sketch"}

// isDrawioMacro は拡張ノードが draw.io の図・ボードのマクロかどうかを返す
func isDrawioMacro(node ADFNode) bool {
	if node.Attrs == nil {
		return false
	}
	if t, _ := node.Attrs["extensionType"].(string); t != forgeExtensionType {
		return false
	}
	key := extensionKey(node)
	for _, s := range drawioMacroSuffixes {
		if strings.HasSuffix(key, s) {
			return true
		}
	}
	return false
}

// renderDrawio は draw.io のマクロを、プレビュー画像（<図の名前>.png）を元の図ファイルへのリンクで包んだ Markdown にする。
// 添付一覧があるのにプレビュー画像が見つからない場合は、警告を出して図ファイルへのリンクだけにする。
// 添付一覧が無い場合（中間ファイルからの convert）は確かめられないため、プレビュー画像があるものとして出力する。
func (r *adfRenderer) renderDrawio(node ADFNode) string {
	params, _ := node.Attrs["parameters"].(map[string]any)
	guest, _ := params["guestParams"].(map[string]any)
	name, _ := guest["diagramName"].(string)
	if name == "" {
		r.warnings = append(r.warnings, "draw.io: 図の名前（diagramName）が無いため表示を省略しました")
		return "<!-- macro: draw.io（図の名前がありません） -->"
	}
	alt, _ := guest["diagramDisplayName"].(string)
	if alt == "" {
		alt = name
	}
	alt = escapeLinkText(alt)

	// 添付ファイルはサニタイズした名前で保存されるので、リンク先もそれに合わせる
	diagram := markdownLinkDest(sanitizeFilename(name))
	preview := name + ".png"
	if r.attachmentMap != nil && !hasAttachmentTitle(r.attachmentMap, preview) {
		r.warnings = append(r.warnings, "draw.io: プレビュー画像 "+preview+" が添付ファイルに無いため、図へのリンクだけを出力しました")
		return "[" + alt + "](" + diagram + ")"
	}
	return "[![" + alt + "](" + markdownLinkDest(sanitizeFilename(preview)) + ")](" + diagram + ")"
}

// hasAttachmentTitle は添付ファイル（UUID → ファイル名）に指定の名前があるかを返す
func hasAttachmentTitle(attachmentMap map[string]string, title string) bool {
	for _, t := range attachmentMap {
		if t == title {
			return true
		}
	}
	return false
}

// escapeLinkText はリンク・画像のテキストに使えるよう、角括弧と、強調・コード・HTML になる記号をエスケープする
func escapeLinkText(s string) string {
	return strings.NewReplacer(`\`, `\\`, "[", `\[`, "]", `\]`, "*", `\*`, "_", `\_`, "`", "\\`", "<", `\<`, "&", `\&`).Replace(s)
}

// markdownLinkDest は Markdown のリンク先で区切りになる空白・括弧・山括弧・引用符と、
// URL のフラグメント・クエリになる # と ? をパーセントエンコードする。
// 日本語などはそのまま残す（ブラウザが送るときにエンコードする）
func markdownLinkDest(s string) string {
	return strings.NewReplacer("%", "%25", " ", "%20", "(", "%28", ")", "%29",
		"<", "%3C", ">", "%3E", `"`, "%22", "#", "%23", "?", "%3F").Replace(s)
}
