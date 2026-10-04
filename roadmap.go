package main

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Confluence のロードマッププランナー（roadmap マクロ）を SVG で描く。
// マクロの source に URL エンコードされた JSON（期間・レーン・バー・マーカー）が入っている。

// roadmapData は roadmap マクロの source（JSON）の内容
type roadmapData struct {
	Title    string `json:"title"`
	Timeline struct {
		StartDate     string `json:"startDate"`
		EndDate       string `json:"endDate"`
		DisplayOption string `json:"displayOption"` // "MONTH" / "WEEK"
	} `json:"timeline"`
	Lanes []struct {
		Title string `json:"title"`
		Color struct {
			Lane string `json:"lane"`
			Bar  string `json:"bar"`
			Text string `json:"text"`
		} `json:"color"`
		Bars []struct {
			Title       string  `json:"title"`
			Description string  `json:"description"`
			StartDate   string  `json:"startDate"`
			Duration    float64 `json:"duration"` // 列（月または週）の数
			RowIndex    int     `json:"rowIndex"`
		} `json:"bars"`
	} `json:"lanes"`
	Markers []struct {
		Title      string `json:"title"`
		MarkerDate string `json:"markerDate"`
	} `json:"markers"`
}

// 描画寸法（CSS ピクセル）。Confluence の表示に合わせている
const (
	rmColW       = 85 // 1列（1か月または1週）の幅
	rmTitleW     = 32 // 左端のレーン名の幅
	rmPadL       = 8  // レーン名と最初の列の間
	rmPadR       = 8
	rmHeaderH    = 48 // 年と月（週）の見出しの高さ
	rmRowH       = 38 // バー1行分の高さ
	rmBarH       = 32
	rmBarGap     = 6 // レーン上端・行間の余白
	rmMinRows    = 2 // バーが無いレーンも2行分の高さにする
	rmFontSize   = 12
	rmMarkerH    = 48 // マーカー名（2行）を描く下側の余白
	rmMarkerLine = 15
)

var roadmapColorRe = regexp.MustCompile(`^#[0-9A-Fa-f]{3,8}$`)

// renderRoadmap は roadmap マクロを SVG にする。source を読めないときは警告を記録し、従来どおりのコメントを返す
func (r *adfRenderer) renderRoadmap(node ADFNode) string {
	src, err := url.PathUnescape(macroParams(node)["source"])
	if err == nil {
		var svg string
		if svg, err = renderRoadmapSVG(src); err == nil {
			return svg
		}
	}
	r.warnings = append(r.warnings, "ロードマップ: マクロのデータを読めないため表示を省略しました: "+err.Error())
	return "<!-- macro: roadmap -->"
}

// renderRoadmapSVG は source の JSON から、横スクロールする div で囲んだ SVG を作る。
// Markdown の HTML ブロックが途切れないよう、出力は1行（空行なし）にする
func renderRoadmapSVG(src string) (string, error) {
	var d roadmapData
	if err := json.Unmarshal([]byte(src), &d); err != nil {
		return "", fmt.Errorf("JSON を解釈できません: %w", err)
	}
	start, err := parseRoadmapTime(d.Timeline.StartDate)
	if err != nil {
		return "", err
	}
	end, err := parseRoadmapTime(d.Timeline.EndDate)
	if err != nil {
		return "", err
	}
	sc := newRoadmapScale(start, end, d.Timeline.DisplayOption)
	x := func(pos float64) float64 { return rmTitleW + rmPadL + pos*rmColW }
	width := int(x(float64(sc.cols))) + rmPadR

	// レーンの縦位置
	laneTops := make([]int, len(d.Lanes))
	laneHs := make([]int, len(d.Lanes))
	y := rmHeaderH
	for i, lane := range d.Lanes {
		rows := rmMinRows
		for _, b := range lane.Bars {
			rows = max(rows, b.RowIndex+1)
		}
		laneTops[i], laneHs[i] = y, rows*rmRowH+rmBarGap
		y += laneHs[i]
	}
	chartBottom := y
	height := chartBottom + rmBarGap
	if len(d.Markers) > 0 {
		height = chartBottom + rmMarkerH
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, `<div class="roadmap" style="overflow-x:auto;margin:1rem 0">`)
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="%s" font-size="%d" style="display:block;max-width:none">`,
		width, height, width, height, escapeSVG(d.Title), rmFontSize)

	// 見出し（年は最初の列と年が変わる列の上、月・週は列の中央）
	for i, l := range sc.labels() {
		cx := x(float64(i) + 0.5)
		if l.year != "" {
			fmt.Fprintf(&sb, `<text x="%.1f" y="18" text-anchor="middle" font-weight="bold" fill="#6b778c">%s</text>`, cx, l.year)
		}
		fmt.Fprintf(&sb, `<text x="%.1f" y="36" text-anchor="middle" fill="#6b778c">%s</text>`, cx, escapeSVG(l.text))
	}

	// 罫線（列の区切りは点線、レーンの区切りは実線）
	for i := 0; i <= sc.cols; i++ {
		fmt.Fprintf(&sb, `<line x1="%.1f" y1="%d" x2="%.1f" y2="%d" stroke="#c1c7d0" stroke-dasharray="4 4"/>`, x(float64(i)), rmHeaderH, x(float64(i)), chartBottom)
	}
	fmt.Fprintf(&sb, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#dfe1e6"/>`, rmTitleW, rmHeaderH, width, rmHeaderH)
	for i := range d.Lanes {
		b := laneTops[i] + laneHs[i]
		fmt.Fprintf(&sb, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#dfe1e6"/>`, rmTitleW, b, width, b)
	}

	for i, lane := range d.Lanes {
		laneColor := roadmapColor(lane.Color.Lane, "#c1c7d0")
		barColor := roadmapColor(lane.Color.Bar, laneColor)
		textColor := roadmapColor(lane.Color.Text, "#172b4d")

		// レーン名（左端に縦書き。高さに収まらない分は省略）
		top, h := laneTops[i], laneHs[i]
		cy := float64(top) + float64(h)/2
		fmt.Fprintf(&sb, `<g><title>%s</title><rect x="0" y="%d" width="%d" height="%d" fill="%s"/>`, escapeSVG(lane.Title), top, rmTitleW, h, laneColor)
		fmt.Fprintf(&sb, `<text x="%d" y="%.1f" transform="rotate(-90 %d %.1f)" text-anchor="middle" dominant-baseline="central" fill="%s">%s</text></g>`,
			rmTitleW/2, cy, rmTitleW/2, cy, textColor, escapeSVG(truncateRoadmapText(lane.Title, float64(h-12), rmFontSize)))

		// バー（表示期間の外にはみ出した部分は切り取る）
		for _, b := range lane.Bars {
			bs, err := parseRoadmapTime(b.StartDate)
			if err != nil || b.Duration <= 0 {
				continue
			}
			s := sc.pos(bs)
			e := math.Min(s+b.Duration, float64(sc.cols))
			s = math.Max(s, 0)
			if e <= s {
				continue
			}
			bx, bw := x(s), (e-s)*rmColW
			by := top + rmBarGap + b.RowIndex*rmRowH
			tip := b.Title
			if b.Description != "" {
				tip += "\n" + b.Description
			}
			fmt.Fprintf(&sb, `<g><title>%s</title><rect x="%.1f" y="%d" width="%.1f" height="%d" rx="3" fill="%s"/>`, escapeSVG(tip), bx, by, bw, rmBarH, barColor)
			fmt.Fprintf(&sb, `<text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="central" font-weight="bold" fill="%s">%s</text></g>`,
				bx+bw/2, float64(by)+rmBarH/2.0, textColor, escapeSVG(truncateRoadmapText(b.Title, bw-12, rmFontSize)))
		}
	}

	// マーカー（全レーンを貫く縦線と、下側に2行までの名前）
	for _, m := range d.Markers {
		mt, err := parseRoadmapTime(m.MarkerDate)
		if err != nil {
			continue
		}
		p := sc.pos(mt)
		if p < 0 || p > float64(sc.cols) {
			continue
		}
		mx := x(p)
		fmt.Fprintf(&sb, `<g><title>%s</title><line x1="%.1f" y1="%d" x2="%.1f" y2="%d" stroke="#d04437" stroke-width="1.5"/>`, escapeSVG(m.Title), mx, rmHeaderH, mx, chartBottom+8)
		for li, line := range wrapRoadmapText(m.Title, rmColW, rmFontSize, 2) {
			fmt.Fprintf(&sb, `<text x="%.1f" y="%d" text-anchor="middle" fill="#d04437">%s</text>`, mx, chartBottom+22+li*rmMarkerLine, escapeSVG(line))
		}
		sb.WriteString(`</g>`)
	}

	sb.WriteString(`</svg></div>`)
	return sb.String(), nil
}

// roadmapScale は日付を「先頭の列から何列目か」（小数）に換算する
type roadmapScale struct {
	week  bool
	start time.Time // 先頭の列の始まり（月の1日、または週の月曜日）
	cols  int
}

type roadmapLabel struct {
	text string // "10月" / "28-9月"
	year string // 最初の列と年が変わる列だけ
}

// newRoadmapScale は表示期間の列を作る。開始日を含む月（週）から、終了日を含む月（週）まで
func newRoadmapScale(start, end time.Time, option string) roadmapScale {
	if strings.EqualFold(option, "WEEK") {
		s := mondayOf(start)
		cols := int(mondayOf(end).Sub(s).Hours()/24/7) + 1
		return roadmapScale{week: true, start: s, cols: max(cols, 1)}
	}
	s := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	cols := (end.Year()-s.Year())*12 + int(end.Month()-s.Month()) + 1
	return roadmapScale{start: s, cols: max(cols, 1)}
}

func (sc roadmapScale) pos(t time.Time) float64 {
	if sc.week {
		return t.Sub(sc.start).Hours() / 24 / 7
	}
	months := (t.Year()-sc.start.Year())*12 + int(t.Month()-sc.start.Month())
	daysInMonth := time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	dayFrac := float64(t.Day()-1) + float64(t.Hour()*3600+t.Minute()*60+t.Second())/86400
	return float64(months) + dayFrac/float64(daysInMonth)
}

func (sc roadmapScale) labels() []roadmapLabel {
	labels := make([]roadmapLabel, sc.cols)
	prevYear := 0
	for i := range labels {
		var t time.Time
		if sc.week {
			t = sc.start.AddDate(0, 0, 7*i)
			labels[i].text = fmt.Sprintf("%02d-%d月", t.Day(), int(t.Month()))
		} else {
			t = sc.start.AddDate(0, i, 0)
			labels[i].text = fmt.Sprintf("%d月", int(t.Month()))
		}
		if t.Year() != prevYear {
			labels[i].year = fmt.Sprint(t.Year())
			prevYear = t.Year()
		}
	}
	return labels
}

func mondayOf(t time.Time) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	offset := (int(d.Weekday()) + 6) % 7 // 月曜日を 0 とする
	return d.AddDate(0, 0, -offset)
}

// parseRoadmapTime はロードマップの日時（"2006-01-02 15:04:05"）を読む
func parseRoadmapTime(s string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("日時を解釈できません: %q", s)
}

// roadmapColor は "#rrggbb" 形式の色だけを受け付け、それ以外は def を返す（属性値への混入を防ぐ）
func roadmapColor(c, def string) string {
	if roadmapColorRe.MatchString(c) {
		return c
	}
	return def
}

// roadmapTextWidth は文字列の表示幅を概算する（半角は 0.6 文字分、それ以外は 1 文字分）。
// SVG では文字が折り返されず、ブラウザ側の実際の幅も分からないため、省略位置の目安に使う
func roadmapTextWidth(s string, fontSize float64) float64 {
	w := 0.0
	for _, r := range s {
		if r < 0x80 {
			w += 0.6 * fontSize
		} else {
			w += fontSize
		}
	}
	return w
}

// truncateRoadmapText は幅に収まらない文字列を「…」で省略する。「…」も入らない幅なら空にする
func truncateRoadmapText(s string, maxW, fontSize float64) string {
	if roadmapTextWidth(s, fontSize) <= maxW {
		return s
	}
	limit := maxW - fontSize // 「…」の分
	if limit < 0 {
		return ""
	}
	var sb strings.Builder
	w := 0.0
	for _, r := range s {
		rw := roadmapTextWidth(string(r), fontSize)
		if w+rw > limit {
			break
		}
		sb.WriteRune(r)
		w += rw
	}
	return sb.String() + "…"
}

// wrapRoadmapText は文字列を幅 maxW で折り返し、maxLines 行を超える分は最後の行を「…」で省略する
func wrapRoadmapText(s string, maxW, fontSize float64, maxLines int) []string {
	var lines []string
	rest := []rune(s)
	for len(rest) > 0 && len(lines) < maxLines {
		if len(lines) == maxLines-1 {
			lines = append(lines, truncateRoadmapText(string(rest), maxW, fontSize))
			break
		}
		w, n := 0.0, 0
		for n < len(rest) {
			rw := roadmapTextWidth(string(rest[n]), fontSize)
			if w+rw > maxW && n > 0 {
				break
			}
			w += rw
			n++
		}
		lines = append(lines, string(rest[:n]))
		rest = rest[n:]
	}
	return lines
}

// escapeSVG は SVG（XML）の文字列・属性値用にエスケープする
func escapeSVG(s string) string {
	return html.EscapeString(s)
}
