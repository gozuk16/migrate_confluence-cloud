package main

import (
	"reflect"
	"testing"
)

func TestParseReportCQL(t *testing.T) {
	tests := []struct {
		name      string
		cql       string
		want      reportFilter
		wantWarns int
	}{
		{"空", "  ", reportFilter{Space: "current"}, 0},
		{"実データ", `label = "memo" and space = currentSpace ( ) and parent = currentContent ( )`,
			reportFilter{Labels: []string{"memo"}, LabelsMode: "all", Space: "current", Scope: "children", Root: "current"}, 0},
		{"ラベルand", `label = "a" AND label = 'b'`, reportFilter{Labels: []string{"a", "b"}, LabelsMode: "all"}, 0},
		{"ラベルin", `label in ("a", "b")`, reportFilter{Labels: []string{"a", "b"}, LabelsMode: "any"}, 0},
		{"ラベルor", `label = "a" or label = "b"`, reportFilter{Labels: []string{"a", "b"}, LabelsMode: "any"}, 0},
		{"ラベルor括弧", `(label = "a" or label = "b") and space = "DEV"`,
			reportFilter{Labels: []string{"a", "b"}, LabelsMode: "any", Space: "DEV"}, 0},
		{"ラベル除外", `label != "x" and label not in ("y","z")`, reportFilter{LabelsExclude: []string{"x", "y", "z"}}, 0},
		{"ラベルandとorの混在", `label = "a" and label in ("b","c")`,
			reportFilter{Labels: []string{"a", "b", "c"}, LabelsMode: "any"}, 1},
		{"スペース引用なし", `space = DEV`, reportFilter{Space: "DEV"}, 0},
		{"親ID", `parent = 12345`, reportFilter{Scope: "children", Root: "12345"}, 0},
		{"祖先", `ancestor = currentContent()`, reportFilter{Scope: "descendants", Root: "current"}, 0},
		{"タイトル完全一致", `title = "週報"`, reportFilter{TitleIs: "週報"}, 0},
		{"タイトル部分一致", `title ~ "週報*"`, reportFilter{TitleContains: "週報"}, 0},
		{"作成日以降", `created >= "2026/01/05"`, reportFilter{CreatedFrom: "2026-01-05"}, 0},
		{"作成日より後", `created > "2026-01-05"`, reportFilter{CreatedFrom: "2026-01-06"}, 0},
		{"更新日より前", `lastmodified < "2026-03-01"`, reportFilter{LastmodTo: "2026-02-28"}, 0},
		{"作成日一致", `created = "2026-01-05"`, reportFilter{CreatedFrom: "2026-01-05", CreatedTo: "2026-01-05"}, 0},
		{"相対日付", `lastModified > now("-4w")`, reportFilter{LastmodFrom: "now-4w"}, 0},
		{"now引数なし", `created <= now()`, reportFilter{CreatedTo: "now"}, 0},
		{"type=pageは無視", `type = page and label = "a"`, reportFilter{Labels: []string{"a"}, LabelsMode: "all"}, 0},
		{"未対応の項目", `creator = currentUser() and label = "a"`, reportFilter{Labels: []string{"a"}, LabelsMode: "all"}, 1},
		{"異なる項目のor", `label = "a" or title ~ "x" and space = currentSpace()`, reportFilter{Space: "current"}, 1},
		{"括弧の入れ子は除外", `(title ~ "a" and label = "b") and space = "X"`, reportFilter{Space: "X"}, 1},
		{"解釈できない日付", `created > "yesterday"`, reportFilter{}, 1},
		{"構文エラー", `label = `, reportFilter{Space: "current"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, warns := parseReportCQL(tt.cql)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseReportCQL(%q)\n got  %+v\n want %+v", tt.cql, got, tt.want)
			}
			if len(warns) != tt.wantWarns {
				t.Errorf("warnings = %q, want %d 件", warns, tt.wantWarns)
			}
		})
	}
}
