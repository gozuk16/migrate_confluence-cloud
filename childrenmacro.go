package main

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	headingStyleRe = regexp.MustCompile(`^h[1-6]$`)
	// "スペースキー:タイトル" 形式。キーは大文字英数字、個人スペースは "~" で始まる。
	// "API: 概要" のようにコロンの後に空白があるものは通常のタイトルとして扱う
	spaceKeyTitleRe = regexp.MustCompile(`^([A-Z0-9]+|~[A-Za-z0-9-]+):(\S.*)$`)
)

// renderChildren は children（子ページ一覧）マクロを children ショートコードに変換する
func (r *adfRenderer) renderChildren(node ADFNode) string {
	return buildChildrenShortcode(macroParams(node))
}

// buildChildrenShortcode は children マクロの引数を children ショートコードに変換する。
// 値が無い・解釈できない引数は出力しない（ショートコード側の既定値になる）
func buildChildrenShortcode(opts map[string]string) string {
	page, space := opts["page"], ""
	if m := spaceKeyTitleRe.FindStringSubmatch(page); m != nil {
		space, page = m[1], m[2]
	}

	all := ""
	if strings.EqualFold(opts["allChildren"], "true") {
		all = "true"
	}

	// 並び順は Cloud では sortAndReverse（例: "title,reverse"）、旧形式では sort と reverse
	sortKey, reverse := "", ""
	tokens := strings.FieldsFunc(strings.ToLower(opts["sortAndReverse"]+","+opts["sort"]), func(c rune) bool {
		return c == ',' || c == ' ' || c == ';'
	})
	for _, tok := range tokens {
		switch tok {
		case "creation", "title", "modified":
			sortKey = tok
		case "reverse":
			reverse = "true"
		}
	}
	if strings.EqualFold(opts["reverse"], "true") {
		reverse = "true"
	}

	args := [][2]string{
		{"page", page},
		{"space", space},
		{"all", all},
		{"depth", positiveIntOrEmpty(opts["depth"])},
		{"sort", sortKey},
		{"reverse", reverse},
		{"first", positiveIntOrEmpty(opts["first"])},
		{"style", matchOrEmpty(headingStyleRe, opts["style"])},
	}
	var sb strings.Builder
	sb.WriteString("{{< children")
	for _, a := range args {
		if a[1] == "" {
			continue
		}
		sb.WriteString(" " + a[0] + "=" + quoteShortcodeParam(a[1]))
	}
	sb.WriteString(" >}}")
	return sb.String()
}

// positiveIntOrEmpty は v が正の整数なら先頭のゼロを除いた10進表記を、それ以外は空文字を返す
func positiveIntOrEmpty(v string) string {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n <= 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// matchOrEmpty は v が re に一致すればそのまま、一致しなければ空文字を返す
func matchOrEmpty(re *regexp.Regexp, v string) string {
	v = strings.TrimSpace(v)
	if re.MatchString(v) {
		return v
	}
	return ""
}
