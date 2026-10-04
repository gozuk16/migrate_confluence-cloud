package main

import "testing"

func TestBuildChildrenShortcode(t *testing.T) {
	tests := []struct {
		name string
		opts map[string]string
		want string
	}{
		{"引数なし", map[string]string{}, `{{< children >}}`},
		{"実データ", map[string]string{"depth": "2", "allChildren": "true", "style": "", "sortAndReverse": "", "first": "0"},
			`{{< children all="true" depth="2" >}}`},
		{"allChildrenがfalse", map[string]string{"allChildren": "false", "depth": "3"}, `{{< children depth="3" >}}`},
		{"depth0は上限なし", map[string]string{"allChildren": "TRUE", "depth": "0"}, `{{< children all="true" >}}`},
		{"数字以外のdepthとfirstは無視", map[string]string{"depth": "abc", "first": "-1"}, `{{< children >}}`},
		{"並び順と逆順", map[string]string{"sortAndReverse": "title,reverse"}, `{{< children sort="title" reverse="true" >}}`},
		{"並び順のみ", map[string]string{"sortAndReverse": "Modified"}, `{{< children sort="modified" >}}`},
		{"旧形式のsortとreverse", map[string]string{"sort": "creation", "reverse": "true"}, `{{< children sort="creation" reverse="true" >}}`},
		{"未知の並び順は無視", map[string]string{"sortAndReverse": "random"}, `{{< children >}}`},
		{"件数", map[string]string{"first": "5"}, `{{< children first="5" >}}`},
		{"見出し形式", map[string]string{"style": "h3"}, `{{< children style="h3" >}}`},
		{"見出し以外のstyleは無視", map[string]string{"style": "h7"}, `{{< children >}}`},
		{"別ページ", map[string]string{"page": "議事録"}, `{{< children page="議事録" >}}`},
		{"スペースつきの別ページ", map[string]string{"page": "DEV:設計: 概要"}, `{{< children page="設計: 概要" space="DEV" >}}`},
		{"スペースキーでない接頭辞は分けない", map[string]string{"page": "Note: 議事録"}, `{{< children page="Note: 議事録" >}}`},
		{"引用符を含むページ名", map[string]string{"page": `a"b`}, `{{< children page="a\"b" >}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildChildrenShortcode(tt.opts); got != tt.want {
				t.Errorf("\n got  %s\n want %s", got, tt.want)
			}
		})
	}
}

func TestConvertADF_ChildrenMacro(t *testing.T) {
	adf := adfDoc(`{"type":"extension","attrs":{"extensionType":"com.atlassian.confluence.macro.core","extensionKey":"children",` +
		`"parameters":{"macroParams":{"depth":{"value":"2"},"allChildren":{"value":"true"},"style":{"value":""},"sortAndReverse":{"value":""},"first":{"value":"0"}}}}}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := `{{< children all="true" depth="2" >}}`; got != want {
		t.Errorf("\n got  %s\n want %s", got, want)
	}
}
