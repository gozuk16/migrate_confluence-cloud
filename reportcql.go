package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// reportFilter は detailssummary の CQL を、ショートコードで扱える平らな絞り込み条件にしたもの
type reportFilter struct {
	Labels        []string
	LabelsMode    string // "all" / "any"（Labels が空なら ""）
	LabelsExclude []string
	Space         string // "current" / スペースキー / ""（全スペース）
	Scope         string // "children" / "descendants" / ""
	Root          string // "current" / ページID
	TitleIs       string
	TitleContains string
	CreatedFrom   string // "YYYY-MM-DD" / "now" / "now-4w" 等
	CreatedTo     string
	LastmodFrom   string
	LastmodTo     string
}

type cqlToken struct {
	kind string // "word" / "string" / "op" / "(" / ")" / ","
	text string
}

type cqlValue struct {
	text string
	fn   bool     // 関数呼び出し（currentContent() 等）
	args []string // 関数の引数
}

type cqlClause struct {
	field  string // 小文字
	op     string // "=" "!=" "~" "!~" ">" ">=" "<" "<=" "in" "not in"
	values []cqlValue
	raw    string // 警告表示用
	// group が空でなければ括弧グループ（中身の句）
	group []cqlClause
	// groupOK は group が「肯定の label 条件の or」だけでできているか
	groupOK bool
}

// parseReportCQL は CQL を解析して reportFilter に変換する。表せない条件は警告を返して除外する
func parseReportCQL(cql string) (reportFilter, []string) {
	if strings.TrimSpace(cql) == "" {
		return reportFilter{Space: "current"}, nil
	}
	tokens, err := tokenizeCQL(cql)
	if err != nil {
		return reportFilter{Space: "current"}, []string{fmt.Sprintf("CQLを解析できませんでした（同じスペースの全ページを対象にします）: %v: %s", err, cql)}
	}
	p := &cqlParser{tokens: tokens}
	clauses, conns, err := p.parseExpr()
	if err == nil && p.pos < len(p.tokens) {
		err = fmt.Errorf("余分な字句 %q", p.tokens[p.pos].text)
	}
	if err != nil {
		return reportFilter{Space: "current"}, []string{fmt.Sprintf("CQLを解析できませんでした（同じスペースの全ページを対象にします）: %v: %s", err, cql)}
	}

	var warns []string
	var f reportFilter

	// or を処理する: 両側が肯定の label 条件ならまとめ、それ以外は両側を除外する
	type item struct {
		clauses []cqlClause // 1件なら通常の句、複数なら label の or グループ
		dropped bool
	}
	items := []item{{clauses: []cqlClause{clauses[0]}}}
	for i, conn := range conns {
		next := clauses[i+1]
		last := &items[len(items)-1]
		if conn == "and" {
			items = append(items, item{clauses: []cqlClause{next}})
			continue
		}
		if !last.dropped && allPositiveLabel(last.clauses) && isPositiveLabel(next) {
			last.clauses = append(last.clauses, next)
			continue
		}
		if !last.dropped {
			warns = append(warns, "異なる項目どうしの or は未対応のため除外しました: "+joinRaw(last.clauses)+" or "+next.raw)
		} else {
			warns[len(warns)-1] += " or " + next.raw
		}
		last.dropped = true
	}

	var allLabels []string
	var anyGroups [][]string
	for _, it := range items {
		if it.dropped {
			continue
		}
		if len(it.clauses) > 1 {
			anyGroups = append(anyGroups, labelValues(it.clauses))
			continue
		}
		c := it.clauses[0]
		if c.group != nil {
			if !c.groupOK {
				warns = append(warns, "括弧によるグループ化は label の or 以外は未対応のため除外しました: "+c.raw)
				continue
			}
			anyGroups = append(anyGroups, labelValues(c.group))
			continue
		}
		if c.field == "label" && c.op == "=" {
			allLabels = append(allLabels, c.values[0].text)
			continue
		}
		if c.field == "label" && c.op == "in" {
			anyGroups = append(anyGroups, labelValues([]cqlClause{c}))
			continue
		}
		if w := applyClause(&f, c); w != "" {
			warns = append(warns, w)
		}
	}

	switch {
	case len(anyGroups) == 0 && len(allLabels) > 0:
		f.Labels, f.LabelsMode = allLabels, "all"
	case len(anyGroups) == 1 && len(allLabels) == 0:
		f.Labels, f.LabelsMode = anyGroups[0], "any"
	case len(anyGroups) > 0:
		labels := allLabels
		for _, g := range anyGroups {
			labels = append(labels, g...)
		}
		f.Labels, f.LabelsMode = labels, "any"
		warns = append(warns, "label の and と or の混在は未対応のため「いずれかのラベル」として扱います: "+strings.Join(labels, ", "))
	}
	return f, warns
}

// applyClause は label の肯定条件以外の句を f に反映する。表せない場合は警告を返す
func applyClause(f *reportFilter, c cqlClause) string {
	v := cqlValue{}
	if len(c.values) > 0 {
		v = c.values[0]
	}
	unsupported := "未対応の条件のため除外しました: " + c.raw
	switch c.field {
	case "label":
		if c.op == "!=" || c.op == "not in" {
			for _, x := range c.values {
				f.LabelsExclude = append(f.LabelsExclude, x.text)
			}
			return ""
		}
	case "space":
		if c.op == "=" {
			if v.fn && strings.EqualFold(v.text, "currentspace") {
				f.Space = "current"
				return ""
			}
			if !v.fn {
				f.Space = v.text
				return ""
			}
		}
	case "parent", "ancestor":
		if c.op == "=" {
			root := ""
			if v.fn && strings.EqualFold(v.text, "currentcontent") {
				root = "current"
			} else if !v.fn {
				root = v.text
			}
			if root != "" {
				f.Root = root
				f.Scope = map[string]string{"parent": "children", "ancestor": "descendants"}[c.field]
				return ""
			}
		}
	case "title":
		switch c.op {
		case "=":
			f.TitleIs = v.text
			return ""
		case "~":
			f.TitleContains = strings.ReplaceAll(v.text, "*", "")
			return ""
		}
	case "type":
		if c.op == "=" && strings.EqualFold(v.text, "page") {
			return ""
		}
	case "created", "lastmodified":
		from, to := &f.CreatedFrom, &f.CreatedTo
		if c.field == "lastmodified" {
			from, to = &f.LastmodFrom, &f.LastmodTo
		}
		date, relative, ok := parseCQLDate(v)
		if !ok {
			return "解釈できない日付のため除外しました: " + c.raw
		}
		shift := func(days int) string {
			if relative {
				return date
			}
			t, _ := time.Parse("2006-01-02", date)
			return t.AddDate(0, 0, days).Format("2006-01-02")
		}
		switch c.op {
		case ">":
			*from = shift(1)
		case ">=":
			*from = date
		case "<":
			*to = shift(-1)
		case "<=":
			*to = date
		case "=":
			*from, *to = date, date
		default:
			return unsupported
		}
		return ""
	}
	return unsupported
}

var cqlRelativeArg = regexp.MustCompile(`^[+-]\d+[dwMy]$`)

// parseCQLDate は日付の値を "YYYY-MM-DD" または "now" / "now-4w" 形式にする
func parseCQLDate(v cqlValue) (date string, relative bool, ok bool) {
	if v.fn {
		if !strings.EqualFold(v.text, "now") {
			return "", false, false
		}
		if len(v.args) == 0 || v.args[0] == "" {
			return "now", true, true
		}
		if cqlRelativeArg.MatchString(v.args[0]) {
			return "now" + v.args[0], true, true
		}
		return "", false, false
	}
	s := strings.ReplaceAll(strings.TrimSpace(v.text), "/", "-")
	if len(s) > 10 {
		s = s[:10]
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return "", false, false
	}
	return t.Format("2006-01-02"), false, true
}

func isPositiveLabel(c cqlClause) bool {
	if c.group != nil {
		return c.groupOK
	}
	return c.field == "label" && (c.op == "=" || c.op == "in")
}

func allPositiveLabel(cs []cqlClause) bool {
	for _, c := range cs {
		if !isPositiveLabel(c) {
			return false
		}
	}
	return true
}

// labelValues は肯定の label 条件（括弧グループを含む）からラベル名を出現順に集める
func labelValues(cs []cqlClause) []string {
	var out []string
	for _, c := range cs {
		if c.group != nil {
			out = append(out, labelValues(c.group)...)
			continue
		}
		for _, v := range c.values {
			out = append(out, v.text)
		}
	}
	return out
}

func joinRaw(cs []cqlClause) string {
	parts := make([]string, len(cs))
	for i, c := range cs {
		parts[i] = c.raw
	}
	return strings.Join(parts, " or ")
}

// tokenizeCQL は CQL を字句に分ける
func tokenizeCQL(s string) ([]cqlToken, error) {
	var tokens []cqlToken
	rs := []rune(s)
	for i := 0; i < len(rs); {
		c := rs[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '"' || c == '\'':
			var sb strings.Builder
			j := i + 1
			for ; j < len(rs) && rs[j] != c; j++ {
				if rs[j] == '\\' && j+1 < len(rs) {
					j++
				}
				sb.WriteRune(rs[j])
			}
			if j >= len(rs) {
				return nil, fmt.Errorf("閉じていない引用符")
			}
			tokens = append(tokens, cqlToken{"string", sb.String()})
			i = j + 1
		case c == '(' || c == ')' || c == ',':
			tokens = append(tokens, cqlToken{string(c), string(c)})
			i++
		case c == '!' || c == '>' || c == '<' || c == '=' || c == '~':
			if i+1 < len(rs) && (rs[i+1] == '=' || (c == '!' && rs[i+1] == '~')) {
				tokens = append(tokens, cqlToken{"op", string(rs[i : i+2])})
				i += 2
				continue
			}
			if c == '!' {
				return nil, fmt.Errorf("不正な演算子 !")
			}
			tokens = append(tokens, cqlToken{"op", string(c)})
			i++
		default:
			j := i
			for j < len(rs) && !strings.ContainsRune(" \t\n\r\"'(),!<>=~", rs[j]) {
				j++
			}
			tokens = append(tokens, cqlToken{"word", string(rs[i:j])})
			i = j
		}
	}
	return tokens, nil
}

type cqlParser struct {
	tokens []cqlToken
	pos    int
}

func (p *cqlParser) peek() (cqlToken, bool) {
	if p.pos >= len(p.tokens) {
		return cqlToken{}, false
	}
	return p.tokens[p.pos], true
}

func (p *cqlParser) next() (cqlToken, bool) {
	t, ok := p.peek()
	if ok {
		p.pos++
	}
	return t, ok
}

// parseExpr は「句 (and|or 句)*」を読み、句と接続詞の並びを返す（閉じ括弧の手前で止まる）
func (p *cqlParser) parseExpr() ([]cqlClause, []string, error) {
	var clauses []cqlClause
	var conns []string
	for {
		c, err := p.parseClause()
		if err != nil {
			return nil, nil, err
		}
		clauses = append(clauses, c)
		t, ok := p.peek()
		if !ok || t.kind == ")" {
			return clauses, conns, nil
		}
		if t.kind != "word" || (!strings.EqualFold(t.text, "and") && !strings.EqualFold(t.text, "or")) {
			return nil, nil, fmt.Errorf("and / or が必要な位置に %q があります", t.text)
		}
		p.pos++
		conns = append(conns, strings.ToLower(t.text))
	}
}

func (p *cqlParser) parseClause() (cqlClause, error) {
	start := p.pos
	t, ok := p.next()
	if !ok {
		return cqlClause{}, fmt.Errorf("条件が途中で終わっています")
	}
	if t.kind == "(" {
		inner, conns, err := p.parseExpr()
		if err != nil {
			return cqlClause{}, err
		}
		if end, ok := p.next(); !ok || end.kind != ")" {
			return cqlClause{}, fmt.Errorf("閉じ括弧がありません")
		}
		okGroup := allPositiveLabel(inner)
		for _, c := range conns {
			if c != "or" {
				okGroup = false
			}
		}
		return cqlClause{group: inner, groupOK: okGroup, raw: p.raw(start)}, nil
	}
	if t.kind != "word" {
		return cqlClause{}, fmt.Errorf("項目名が必要な位置に %q があります", t.text)
	}
	c := cqlClause{field: strings.ToLower(t.text)}

	opTok, ok := p.next()
	if !ok {
		return cqlClause{}, fmt.Errorf("演算子がありません")
	}
	switch {
	case opTok.kind == "op":
		c.op = opTok.text
	case opTok.kind == "word" && strings.EqualFold(opTok.text, "in"):
		c.op = "in"
	case opTok.kind == "word" && strings.EqualFold(opTok.text, "not"):
		if in, ok := p.next(); !ok || !strings.EqualFold(in.text, "in") {
			return cqlClause{}, fmt.Errorf("not の後に in がありません")
		}
		c.op = "not in"
	default:
		return cqlClause{}, fmt.Errorf("演算子が必要な位置に %q があります", opTok.text)
	}

	if c.op == "in" || c.op == "not in" {
		if lp, ok := p.next(); !ok || lp.kind != "(" {
			return cqlClause{}, fmt.Errorf("in の後に ( がありません")
		}
		for {
			v, err := p.parseValue()
			if err != nil {
				return cqlClause{}, err
			}
			c.values = append(c.values, v)
			sep, ok := p.next()
			if !ok {
				return cqlClause{}, fmt.Errorf("閉じ括弧がありません")
			}
			if sep.kind == ")" {
				break
			}
			if sep.kind != "," {
				return cqlClause{}, fmt.Errorf("一覧の区切りに %q があります", sep.text)
			}
		}
	} else {
		v, err := p.parseValue()
		if err != nil {
			return cqlClause{}, err
		}
		c.values = []cqlValue{v}
	}
	c.raw = p.raw(start)
	return c, nil
}

func (p *cqlParser) parseValue() (cqlValue, error) {
	t, ok := p.next()
	if !ok {
		return cqlValue{}, fmt.Errorf("値がありません")
	}
	if t.kind == "string" {
		return cqlValue{text: t.text}, nil
	}
	if t.kind != "word" {
		return cqlValue{}, fmt.Errorf("値が必要な位置に %q があります", t.text)
	}
	if lp, ok := p.peek(); ok && lp.kind == "(" {
		p.pos++
		v := cqlValue{text: t.text, fn: true}
		for {
			a, ok := p.next()
			if !ok {
				return cqlValue{}, fmt.Errorf("関数の閉じ括弧がありません")
			}
			if a.kind == ")" {
				return v, nil
			}
			if a.kind == "," {
				continue
			}
			v.args = append(v.args, a.text)
		}
	}
	return cqlValue{text: t.text}, nil
}

// raw は start から現在位置までの字句を空白区切りで連結する（警告表示用）
func (p *cqlParser) raw(start int) string {
	parts := make([]string, 0, p.pos-start)
	for _, t := range p.tokens[start:p.pos] {
		if t.kind == "string" {
			parts = append(parts, fmt.Sprintf("%q", t.text))
		} else {
			parts = append(parts, t.text)
		}
	}
	return strings.Join(parts, " ")
}
