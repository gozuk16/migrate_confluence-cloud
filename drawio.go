package main

import (
	"fmt"
	"html"
	"strings"
)

// drawioFilePrefix は書き出す SVG のファイル名の接頭辞（ページ内の出現順に confluence-drawio-1.svg, confluence-drawio-2.svg…）。
// 添付ファイルと同じフォルダに置くので、名前がぶつかりにくいものにする
const drawioFilePrefix = "confluence-drawio-"

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

// renderDrawio は draw.io のマクロを、図を SVG にして本文へ直接差し込む drawio ショートコードにする。
// Confluence 側の表示設定（simple・lbox など）は再現せず、どの図も同じように SVG にする。
// SVG にできないとき（draw.io のコマンドが無い、図ファイルが保存されていない、変換に失敗した）は、
// 警告を出して、プレビュー画像（<図の名前>.png）を元の図ファイルへのリンクで包んだ Markdown に戻す。
// そのとき、添付一覧があるのにプレビュー画像が見つからなければ、警告を出して図ファイルへのリンクだけにする。
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
	if svg := r.drawioSVGFile(name); svg != "" {
		return "{{< drawio src=" + quoteShortcodeParam(svg) + " file=" + quoteShortcodeParam(sanitizeFilename(name)) +
			" title=" + quoteShortcodeParam(alt) + " >}}"
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

// drawioSVGFile は図を SVG にして、本文と一緒に書き出すファイルに加え、そのファイル名を返す。
// SVG にしない設定のとき（drawioSVG が nil）は何もせず空文字を返す。SVG にできなかったときは警告を記録して空文字を返す。
// SVG に file:// の画像が残っているときは、警告を記録したうえで SVG を使う（その画像のほかは表示できるため）。
// 同じ図をページ内で何度も埋め込んでいても、変換と警告は 1 回だけにして同じファイルを使う
func (r *adfRenderer) drawioSVGFile(diagramName string) string {
	if r.drawioSVG == nil {
		return ""
	}
	if name, done := r.drawioFiles[diagramName]; done {
		return name
	}
	if r.drawioFiles == nil {
		r.drawioFiles = map[string]string{}
	}
	data, err := r.drawioSVG(diagramName)
	if err != nil {
		r.warnings = append(r.warnings, "draw.io: "+diagramName+" を SVG にできないため PNG で表示します: "+err.Error())
		r.drawioFiles[diagramName] = ""
		return ""
	}
	if n := countLocalFileRefs(data); n > 0 {
		r.warnings = append(r.warnings, fmt.Sprintf(
			"draw.io: %s の SVG に、ローカルのファイル（file://）への参照が %d 件残っています。その画像はブラウザでは表示されません"+
				"（draw.io デスクトップ版が古いか、draw.io に無い図形ライブラリの画像を指している可能性があります）", diagramName, n))
	}
	name := fmt.Sprintf("%s%d.svg", drawioFilePrefix, len(r.files)-countNonDrawioFiles(r.files)+1)
	r.files = append(r.files, PageFile{Name: name, Data: data})
	r.drawioFiles[diagramName] = name
	return name
}

// countNonDrawioFiles は書き出し予定のファイルのうち、draw.io の SVG 以外の数を返す
func countNonDrawioFiles(files []PageFile) int {
	n := 0
	for _, f := range files {
		if !strings.HasPrefix(f.Name, drawioFilePrefix) {
			n++
		}
	}
	return n
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

// drawioMediaType は draw.io の図ファイル（添付ファイル）のメディアタイプ
const drawioMediaType = "application/vnd.jgraph.mxfile"

// drawioDownloadLink は draw.io の図ファイルをダウンロードする HTML のリンクを返す。
// Markdown のリンクでは download 属性を付けられないので HTML で出す。
// 拡張子が無い図（ボードなど）は、保存したあと draw.io で開けるように .drawio を付けた名前で保存させる
func drawioDownloadLink(title string) string {
	saveAs := ""
	if !strings.HasSuffix(strings.ToLower(title), ".drawio") {
		saveAs = `="` + html.EscapeString(sanitizeFilename(title)+".drawio") + `"`
	}
	href := html.EscapeString(markdownLinkDest(sanitizeFilename(title)))
	return `<a href="` + href + `" download` + saveAs + `>` + html.EscapeString(title) + `</a>`
}

// isDrawioDraft は添付ファイルが draw.io の編集中の下書きかどうかを返す。
// draw.io は図を編集している間、「~<図の名前>.tmp」や「~drawio~<アカウントID>~<図の名前>.tmp」という名前で
// 図ファイル（mxfile）の下書きを添付する。保存後はゴミ箱に入るが、編集を中断すると残ることがある。
// 普通の添付を誤って外さないよう、図ファイルであること・名前が ~ で始まること・.tmp で終わることのすべてを条件にする
func isDrawioDraft(a Attachment) bool {
	return a.MediaType == drawioMediaType &&
		strings.HasPrefix(a.Title, "~") &&
		strings.HasSuffix(strings.ToLower(a.Title), ".tmp")
}

// excludeDrawioDrafts は添付一覧から draw.io の編集中の下書きを外す。
// 下書きはダウンロードせず、添付ファイル欄にも出さない。
// 添付一覧が不明（nil）なら nil のまま返し、それ以外は空でも nil でないスライスを返す（buildAttachmentMap 参照）
func excludeDrawioDrafts(attachments []Attachment) []Attachment {
	if attachments == nil {
		return nil
	}
	kept := make([]Attachment, 0, len(attachments))
	for _, a := range attachments {
		if !isDrawioDraft(a) {
			kept = append(kept, a)
		}
	}
	return kept
}
