package main

import (
	"math"
	"net/url"
	"strings"
	"testing"
	"time"
)

func mustRoadmapTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := parseRoadmapTime(s)
	if err != nil {
		t.Fatalf("parseRoadmapTime(%q): %v", s, err)
	}
	return v
}

func TestRoadmapScale_Month(t *testing.T) {
	sc := newRoadmapScale(mustRoadmapTime(t, "2026-10-04 00:00:00"), mustRoadmapTime(t, "2027-09-04 00:00:00"), "MONTH")
	if sc.cols != 12 {
		t.Fatalf("cols = %d, want 12（2026年10月〜2027年9月）", sc.cols)
	}
	cases := []struct {
		date string
		want float64
	}{
		{"2026-10-01 00:00:00", 0},
		{"2026-11-01 00:00:00", 1},
		{"2027-01-01 00:00:00", 3},
		{"2026-12-18 04:30:53", 2 + (17+(4*3600+30*60+53)/86400.0)/31},
	}
	for _, c := range cases {
		if got := sc.pos(mustRoadmapTime(t, c.date)); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("pos(%s) = %v, want %v", c.date, got, c.want)
		}
	}
	labels := sc.labels()
	if labels[0].text != "10月" || labels[0].year != "2026" {
		t.Errorf("labels[0] = %+v, want 10月 / 2026", labels[0])
	}
	if labels[1].year != "" {
		t.Errorf("labels[1].year = %q, want empty", labels[1].year)
	}
	if labels[3].text != "1月" || labels[3].year != "2027" {
		t.Errorf("labels[3] = %+v, want 1月 / 2027", labels[3])
	}
}

func TestRoadmapScale_Week(t *testing.T) {
	// 2026-10-04 は日曜日。その週の月曜日 9/28 から始まる
	sc := newRoadmapScale(mustRoadmapTime(t, "2026-10-04 00:00:00"), mustRoadmapTime(t, "2027-04-30 00:00:00"), "WEEK")
	// 2027-04-30（金）の週の月曜日は 4/26。9/28 から 30 週後
	if sc.cols != 31 {
		t.Fatalf("cols = %d, want 31", sc.cols)
	}
	if got := sc.pos(mustRoadmapTime(t, "2026-10-01 00:00:00")); math.Abs(got-3.0/7) > 1e-9 {
		t.Errorf("pos(10/1) = %v, want 3/7", got)
	}
	labels := sc.labels()
	if labels[0].text != "28-9月" || labels[0].year != "2026" {
		t.Errorf("labels[0] = %+v, want 28-9月 / 2026", labels[0])
	}
	if labels[1].text != "05-10月" || labels[1].year != "" {
		t.Errorf("labels[1] = %+v, want 05-10月", labels[1])
	}
	// 2027年最初の列（2026-12-28 の週の次、2027-01-04）に年を出す
	var got2027 string
	for _, l := range labels {
		if l.year == "2027" {
			got2027 = l.text
		}
	}
	if got2027 != "04-1月" {
		t.Errorf("2027 の年ラベルの列 = %q, want 04-1月", got2027)
	}
}

func TestTruncateRoadmapText(t *testing.T) {
	if got := truncateRoadmapText("バー1", 200, 12); got != "バー1" {
		t.Errorf("収まる文字列は変えない: got %q", got)
	}
	got := truncateRoadmapText("長い文字が出たときに省略", 60, 12)
	if !strings.HasSuffix(got, "…") || roadmapTextWidth(got, 12) > 60 {
		t.Errorf("省略後 %q（幅 %v）が 60 に収まり … で終わること", got, roadmapTextWidth(got, 12))
	}
	if got := truncateRoadmapText("長い", 5, 12); got != "" {
		t.Errorf("… も入らない幅では空にする: got %q", got)
	}
}

func TestWrapRoadmapText(t *testing.T) {
	lines := wrapRoadmapText("長い文字が出たときに折り返しと省略", 84, 12, 2)
	if len(lines) != 2 {
		t.Fatalf("lines = %q, want 2 lines", lines)
	}
	if lines[0] != "長い文字が出た" {
		t.Errorf("lines[0] = %q, want 長い文字が出た", lines[0])
	}
	if !strings.HasSuffix(lines[1], "…") {
		t.Errorf("lines[1] = %q, want … で終わる", lines[1])
	}
	if got := wrapRoadmapText("マーカー1", 84, 12, 2); len(got) != 1 || got[0] != "マーカー1" {
		t.Errorf("短い文字列は1行: got %q", got)
	}
}

const testRoadmapSource = `{"title":"ロードマップ プランナー","timeline":{"startDate":"2026-10-04 00:00:00","endDate":"2027-04-30 00:00:00","displayOption":"WEEK"},` +
	`"lanes":[{"title":"レーン1","color":{"lane":"#f6c342","bar":"#f6c342","text":"#594300"},"bars":[` +
	`{"title":"バー1","description":"最初のバーです。","startDate":"2026-10-01 00:00:00","duration":8.714285714285714,"rowIndex":1},` +
	`{"title":"長い文字が出たときに省略","description":"説明<b>","startDate":"2026-09-28 01:39:48","duration":1,"rowIndex":0},` +
	`{"title":"範囲外","startDate":"2028-01-01 00:00:00","duration":1,"rowIndex":2}]},` +
	`{"title":"空のレーン","color":{"lane":"#d04437","bar":"#d04437","text":"#ffffff"},"bars":[]}],` +
	`"markers":[{"title":"マーカー1","markerDate":"2026-10-15 00:00:00"}]}`

func TestRenderRoadmapSVG(t *testing.T) {
	got, err := renderRoadmapSVG(testRoadmapSource)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got, `<div class="roadmap"`) || !strings.HasSuffix(got, "</svg></div>") {
		t.Errorf("div で囲んだ SVG であること: %q", got[:80])
	}
	if strings.Contains(got, "\n\n") {
		t.Error("Markdown の HTML ブロックが途切れないよう空行を含めないこと")
	}
	for _, want := range []string{
		`aria-label="ロードマップ プランナー"`,
		`fill="#f6c342"`, `fill="#594300"`, `fill="#d04437"`,
		`>バー1</text>`,
		`<title>バー1`, `最初のバーです。`,
		`<title>長い文字が出たときに省略`, `説明&lt;b&gt;`,
		`>28-9月</text>`, `>2026</text>`,
		`>マーカー1</text>`,
		`>空のレーン</text>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("出力に %q が含まれていません", want)
		}
	}
	if strings.Contains(got, "範囲外") {
		t.Error("表示期間の外のバーは出力しないこと")
	}
	// 1件目のバーは1列の幅に収まらないので省略される（全文はツールチップ）
	if strings.Contains(got, `>長い文字が出たときに省略</text>`) {
		t.Error("長いバー名は省略されること")
	}
}

func TestRenderRoadmapSVG_InvalidSource(t *testing.T) {
	if _, err := renderRoadmapSVG("{broken"); err == nil {
		t.Error("壊れた JSON はエラーにすること")
	}
	if _, err := renderRoadmapSVG(`{"timeline":{"startDate":"x","endDate":"2027-01-01 00:00:00","displayOption":"MONTH"}}`); err == nil {
		t.Error("解釈できない日付はエラーにすること")
	}
}

func TestConvertADFPage_RoadmapMacro(t *testing.T) {
	adf := adfDoc(`{"type":"extension","attrs":{"extensionKey":"roadmap","parameters":{"macroParams":{"source":{"value":"` +
		url.PathEscape(testRoadmapSource) + `"}}}}}`)
	res, err := convertADFPage(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(res.Markdown, `<div class="roadmap"`) {
		t.Errorf("ロードマップの SVG が出力されること: %q", res.Markdown)
	}

	bad := adfDoc(`{"type":"extension","attrs":{"extensionKey":"roadmap","parameters":{"macroParams":{"source":{"value":"%7Bbroken"}}}}}`)
	res, err = convertADFPage(bad, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Markdown != "<!-- macro: roadmap -->" {
		t.Errorf("読めない source は従来どおりのコメント: got %q", res.Markdown)
	}
	if len(res.Warnings) != 1 || !strings.HasPrefix(res.Warnings[0], "ロードマップ: ") {
		t.Errorf("warnings = %q, want 1件（ロードマップ: で始まる）", res.Warnings)
	}
}
