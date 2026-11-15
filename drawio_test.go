package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

const drawioKeyPrefix = "1afdce52-d22e-4d27-84db-5be989c3c83b/92d899e8-452f-4c9b-9f60-823ab11c8253/static/"

// drawioExtension は Confluence Cloud の draw.io（Forge アプリ）マクロの ADF ノードを作る
func drawioExtension(macro, guestParams string) string {
	return `{"type":"extension","attrs":{"layout":"default","extensionType":"com.atlassian.ecosystem",` +
		`"extensionKey":"` + drawioKeyPrefix + macro + `","text":"draw.io Diagram",` +
		`"parameters":{"layout":"extension","guestParams":` + guestParams + `}}}`
}

func TestConvertADFPage_Drawio(t *testing.T) {
	tests := []struct {
		name        string
		macro       string
		guestParams string
		attachments map[string]string
		want        string
		wantWarn    bool
	}{
		{
			name:        "図: プレビュー画像を埋め込み元の図へリンク",
			macro:       "drawio",
			guestParams: `{"diagramName":"test1.drawio","diagramDisplayName":"test1.drawio","revision":2}`,
			attachments: map[string]string{"a1": "test1.drawio", "a2": "test1.drawio.png"},
			want:        "[![test1.drawio](test1.drawio.png)](test1.drawio)",
		},
		{
			name:        "ボード（drawio-sketch）・拡張子なしの図",
			macro:       "drawio-sketch",
			guestParams: `{"diagramName":"test3","diagramDisplayName":"test3"}`,
			attachments: map[string]string{"a1": "test3", "a2": "test3.png"},
			want:        "[![test3](test3.png)](test3)",
		},
		{
			name:        "表示名が無ければ図の名前を代替テキストにする",
			macro:       "drawio",
			guestParams: `{"diagramName":"名称未設定ファイル-1.drawio"}`,
			attachments: map[string]string{"a1": "名称未設定ファイル-1.drawio", "a2": "名称未設定ファイル-1.drawio.png"},
			want:        "[![名称未設定ファイル-1.drawio](名称未設定ファイル-1.drawio.png)](名称未設定ファイル-1.drawio)",
		},
		{
			name:        "空白・括弧はリンク先でエスケープし、角括弧は代替テキストでエスケープする",
			macro:       "drawio",
			guestParams: `{"diagramName":"my diagram (1)","diagramDisplayName":"図 [案]"}`,
			attachments: map[string]string{"a1": "my diagram (1)", "a2": "my diagram (1).png"},
			want:        `[![図 \[案\]](my%20diagram%20%281%29.png)](my%20diagram%20%281%29)`,
		},
		{
			name:        "保存時のファイル名に合わせてサニタイズする",
			macro:       "drawio",
			guestParams: `{"diagramName":"a/b:c"}`,
			attachments: map[string]string{"a1": "a/b:c", "a2": "a/b:c.png"},
			want:        "[![a/b:c](a_b_c.png)](a_b_c)",
		},
		{
			name:        "添付一覧が不明（convert コマンド）なら画像を出す",
			macro:       "drawio",
			guestParams: `{"diagramName":"test1.drawio"}`,
			attachments: nil,
			want:        "[![test1.drawio](test1.drawio.png)](test1.drawio)",
		},
		{
			name:        "プレビュー画像が添付に無ければ警告してリンクのみ",
			macro:       "drawio",
			guestParams: `{"diagramName":"test1.drawio"}`,
			attachments: map[string]string{"a1": "test1.drawio"},
			want:        "[test1.drawio](test1.drawio)",
			wantWarn:    true,
		},
		{
			name:        "添付一覧が 0 件なら警告してリンクのみ",
			macro:       "drawio",
			guestParams: `{"diagramName":"test1.drawio"}`,
			attachments: map[string]string{},
			want:        "[test1.drawio](test1.drawio)",
			wantWarn:    true,
		},
		{
			name:        "# と ? はリンク先でエンコードする",
			macro:       "drawio",
			guestParams: `{"diagramName":"v#2?.drawio"}`,
			attachments: map[string]string{"a1": "v#2?.drawio", "a2": "v#2?.drawio.png"},
			want:        "[![v#2?.drawio](v%232%3F.drawio.png)](v%232%3F.drawio)",
		},
		{
			name:        "リンクのみのとき表示名の強調・コード・HTML の記号をエスケープする",
			macro:       "drawio",
			guestParams: "{\"diagramName\":\"d.drawio\",\"diagramDisplayName\":\"*draft* `x` <b> a_b\"}",
			attachments: map[string]string{"a1": "d.drawio"},
			want:        "[\\*draft\\* \\`x\\` \\<b> a\\_b](d.drawio)",
			wantWarn:    true,
		},
		{
			name:        "図の名前が無ければ警告してコメントを残す",
			macro:       "drawio",
			guestParams: `{}`,
			attachments: map[string]string{"a1": "test1.drawio"},
			want:        "<!-- macro: draw.io（図の名前がありません） -->",
			wantWarn:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := convertADFPage(adfDoc(drawioExtension(tt.macro, tt.guestParams)), tt.attachments)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Markdown != tt.want {
				t.Errorf("got %q, want %q", res.Markdown, tt.want)
			}
			if tt.wantWarn {
				if len(res.Warnings) != 1 || !strings.HasPrefix(res.Warnings[0], "draw.io: ") {
					t.Errorf("warnings = %q, want 1件（draw.io: で始まる）", res.Warnings)
				}
			} else if len(res.Warnings) != 0 {
				t.Errorf("warnings = %q, want none", res.Warnings)
			}
		})
	}
}

// draw.io 以外の Forge マクロは従来どおりコメントにする
func TestConvertADFPage_OtherForgeMacroStaysComment(t *testing.T) {
	adf := adfDoc(`{"type":"extension","attrs":{"extensionType":"com.atlassian.ecosystem","extensionKey":"app/env/static/other"}}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "<!-- macro: app/env/static/other -->"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// 段落内に置かれた draw.io（inlineExtension）も変換する
func TestConvertADFPage_DrawioInline(t *testing.T) {
	node := `{"type":"inlineExtension","attrs":{"extensionType":"com.atlassian.ecosystem","extensionKey":"` + drawioKeyPrefix +
		`drawio","parameters":{"guestParams":{"diagramName":"test1.drawio"}}}}`
	adf := adfDoc(`{"type":"paragraph","content":[` + adfText("図: ") + `,` + node + `]}`)
	got, err := convertADF(adf, map[string]string{"a1": "test1.drawio", "a2": "test1.drawio.png"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "図: [![test1.drawio](test1.drawio.png)](test1.drawio)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// 添付一覧が不明（nil）のときだけ nil を返し、0 件なら空の map を返す
func TestBuildAttachmentMap_NilAndEmpty(t *testing.T) {
	if m := buildAttachmentMap(nil); m != nil {
		t.Errorf("buildAttachmentMap(nil) = %v, want nil", m)
	}
	if m := buildAttachmentMap([]Attachment{}); m == nil || len(m) != 0 {
		t.Errorf("buildAttachmentMap(空) = %v, want 空の map（非 nil）", m)
	}
}

// SVG にできた図は、SVG を書き出して drawio ショートコードにする（リンクで包まない）
func TestConvertADFPage_DrawioSVG(t *testing.T) {
	var asked []string
	opts := adfOptions{
		attachmentMap: map[string]string{"a1": "test1.drawio", "a2": "test1.drawio.png", "a3": "a/b \"x\""},
		drawioSVG: func(name, _ string) ([]byte, error) {
			asked = append(asked, name)
			return []byte("<svg>" + name + "</svg>"), nil
		},
	}
	adf := adfDoc(drawioExtension("drawio", `{"diagramName":"test1.drawio","diagramDisplayName":"構成図","simple":true,"lbox":true}`) + "," +
		drawioExtension("drawio-sketch", `{"diagramName":"a/b \"x\""}`))
	res, err := convertADFPageWith(adf, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{{< drawio src="confluence-drawio-1.svg" file="test1.drawio" title="構成図" >}}` + "\n\n" +
		`{{< drawio src="confluence-drawio-2.svg" file="a_b \"x\"" title="a/b \"x\"" >}}`
	if res.Markdown != want {
		t.Errorf("got  %q\nwant %q", res.Markdown, want)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("warnings = %q, want none", res.Warnings)
	}
	if len(asked) != 2 || asked[0] != "test1.drawio" || asked[1] != `a/b "x"` {
		t.Errorf("SVG にする図の名前 = %q", asked)
	}
	if len(res.Files) != 2 || res.Files[0].Name != "confluence-drawio-1.svg" || string(res.Files[0].Data) != "<svg>test1.drawio</svg>" ||
		res.Files[1].Name != "confluence-drawio-2.svg" {
		t.Errorf("files = %+v", res.Files)
	}
}

// SVG にできない図は、警告を出して PNG の表示に戻す。ほかの図の番号は詰める
func TestConvertADFPage_DrawioSVGFallback(t *testing.T) {
	opts := adfOptions{
		attachmentMap: map[string]string{"a1": "ng.drawio", "a2": "ng.drawio.png", "a3": "ok.drawio"},
		drawioSVG: func(name, _ string) ([]byte, error) {
			if name == "ng.drawio" {
				return nil, errors.New("変換に失敗")
			}
			return []byte("<svg/>"), nil
		},
	}
	adf := adfDoc(drawioExtension("drawio", `{"diagramName":"ng.drawio"}`) + "," + drawioExtension("drawio", `{"diagramName":"ok.drawio"}`))
	res, err := convertADFPageWith(adf, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "[![ng.drawio](ng.drawio.png)](ng.drawio)\n\n" + `{{< drawio src="confluence-drawio-1.svg" file="ok.drawio" title="ok.drawio" >}}`
	if res.Markdown != want {
		t.Errorf("got  %q\nwant %q", res.Markdown, want)
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "ng.drawio を SVG にできないため PNG で表示します: 変換に失敗") {
		t.Errorf("warnings = %q", res.Warnings)
	}
	if len(res.Files) != 1 || res.Files[0].Name != "confluence-drawio-1.svg" {
		t.Errorf("files = %+v", res.Files)
	}
}

func TestDrawioDownloadLink(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"test1.drawio", `<a href="test1.drawio" download>test1.drawio</a>`},
		{"Test.DRAWIO", `<a href="Test.DRAWIO" download>Test.DRAWIO</a>`},
		// 拡張子が無い図は .drawio を付けた名前で保存させる
		{"test3", `<a href="test3" download="test3.drawio">test3</a>`},
		{`a/b <1> "x" #2`, `<a href="a_b%20%3C1%3E%20%22x%22%20%232" download="a_b &lt;1&gt; &#34;x&#34; #2.drawio">a/b &lt;1&gt; &#34;x&#34; #2</a>`},
	}
	for _, tt := range tests {
		if got := drawioDownloadLink(tt.title); got != tt.want {
			t.Errorf("drawioDownloadLink(%q)\n got  %s\n want %s", tt.title, got, tt.want)
		}
	}
}

// 同じ図を何度も埋め込んでいても、変換と警告は 1 回だけで、同じ SVG を使う
func TestConvertADFPage_DrawioSVGSameDiagramTwice(t *testing.T) {
	calls := map[string]int{}
	opts := adfOptions{drawioSVG: func(name, _ string) ([]byte, error) {
		calls[name]++
		if name == "ng.drawio" {
			return nil, errors.New("失敗")
		}
		return []byte("<svg/>"), nil
	}}
	ok := drawioExtension("drawio", `{"diagramName":"ok.drawio"}`)
	ng := drawioExtension("drawio", `{"diagramName":"ng.drawio"}`)
	other := drawioExtension("drawio", `{"diagramName":"other.drawio"}`)
	res, err := convertADFPageWith(adfDoc(ok+","+ng+","+ok+","+ng+","+other), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	okSC := `{{< drawio src="confluence-drawio-1.svg" file="ok.drawio" title="ok.drawio" >}}`
	ngMD := "[![ng.drawio](ng.drawio.png)](ng.drawio)"
	want := okSC + "\n\n" + ngMD + "\n\n" + okSC + "\n\n" + ngMD + "\n\n" +
		`{{< drawio src="confluence-drawio-2.svg" file="other.drawio" title="other.drawio" >}}`
	if res.Markdown != want {
		t.Errorf("got  %q\nwant %q", res.Markdown, want)
	}
	if calls["ok.drawio"] != 1 || calls["ng.drawio"] != 1 || len(res.Files) != 2 || len(res.Warnings) != 1 {
		t.Errorf("calls = %v, files = %d, warnings = %q", calls, len(res.Files), res.Warnings)
	}
}

// draw.io の編集中の下書き（~ で始まり .tmp で終わる mxfile）だけを添付一覧から外す
func TestExcludeDrawioDrafts(t *testing.T) {
	keep := []Attachment{
		{Title: "test1.drawio", MediaType: drawioMediaType},
		{Title: "test3", MediaType: drawioMediaType},
		{Title: "test1.drawio.png", MediaType: "image/png"},
		// 名前が下書きの形でも、図ファイルでなければ普通の添付として残す
		{Title: "~メモ.drawio.tmp", MediaType: "text/plain"},
		{Title: "~backup.tmp", MediaType: "application/octet-stream"},
		// 図ファイルでも、~ で始まるだけ・.tmp で終わるだけのものは残す
		{Title: "~案.drawio", MediaType: drawioMediaType},
		{Title: "構成図.drawio.tmp", MediaType: drawioMediaType},
		{Title: "a~b.drawio.tmp", MediaType: drawioMediaType},
	}
	drafts := []Attachment{
		{Title: "~名称未設定ファイル-1794638497741.drawio.tmp", MediaType: drawioMediaType},
		{Title: "~drawio~557058:6eed56ba~test1.drawio.tmp", MediaType: drawioMediaType},
		// 拡張子の無い図（ボード）の下書き
		{Title: "~test3.tmp", MediaType: drawioMediaType},
		{Title: "~Test.DRAWIO.TMP", MediaType: drawioMediaType},
	}
	for _, a := range keep {
		if isDrawioDraft(a) {
			t.Errorf("%q（%s）は下書きとして扱わないこと", a.Title, a.MediaType)
		}
	}
	for _, a := range drafts {
		if !isDrawioDraft(a) {
			t.Errorf("%q（%s）は下書きとして扱うこと", a.Title, a.MediaType)
		}
	}

	// 順序を保って下書きだけを外す
	mixed := []Attachment{keep[0], drafts[0], keep[1], drafts[1], keep[2]}
	got := excludeDrawioDrafts(mixed)
	if len(got) != 3 || got[0].Title != "test1.drawio" || got[1].Title != "test3" || got[2].Title != "test1.drawio.png" {
		t.Errorf("got %+v", got)
	}
	// 添付一覧が不明（nil）なら nil のまま、全部が下書きでも空のスライス（非 nil）を返す（buildAttachmentMap 参照）
	if got := excludeDrawioDrafts(nil); got != nil {
		t.Errorf("excludeDrawioDrafts(nil) = %v, want nil", got)
	}
	if got := excludeDrawioDrafts(drafts[:1]); got == nil || len(got) != 0 {
		t.Errorf("got %#v, want 空のスライス（非 nil）", got)
	}
}

// SVG に file:// の画像が残っていたら、警告を記録したうえで SVG は埋め込む
func TestDrawioSVGFile_LocalFileRefWarning(t *testing.T) {
	tests := []struct {
		name     string
		svg      string
		wantWarn bool
	}{
		{name: "file:// の画像が残っている", svg: `<svg><image xlink:href="file:///Applications/draw.io.app/x.svg"/></svg>`, wantWarn: true},
		{name: "画像が埋め込まれている", svg: `<svg><image xlink:href="data:image/svg+xml;base64,AAAA"/></svg>`, wantWarn: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &adfRenderer{drawioSVG: func(string, string) ([]byte, error) { return []byte(tt.svg), nil }}
			// 同じ図を 2 回埋め込んでも、警告は 1 回だけ
			for i := 0; i < 2; i++ {
				if got, _ := r.drawioSVGFile("図.drawio", ""); got != "confluence-drawio-1.svg" {
					t.Errorf("drawioSVGFile() = %q, want confluence-drawio-1.svg", got)
				}
			}
			if len(r.files) != 1 || string(r.files[0].Data) != tt.svg {
				t.Errorf("SVG が書き出し予定に入っていません: %v", r.files)
			}
			if tt.wantWarn {
				if len(r.warnings) != 1 || !strings.Contains(r.warnings[0], "図.drawio") || !strings.Contains(r.warnings[0], "file://") {
					t.Errorf("警告が期待と異なります: %v", r.warnings)
				}
			} else if len(r.warnings) != 0 {
				t.Errorf("警告が出ました: %v", r.warnings)
			}
		})
	}
}

// マクロがページを指定していれば（aspect の先頭）、そのページを SVG にして、拡大表示にもページを渡す
func TestConvertADFPage_DrawioPage(t *testing.T) {
	var asked []string
	opts := adfOptions{drawioSVG: func(name, pageID string) ([]byte, error) {
		asked = append(asked, name+"#"+pageID)
		if pageID == "gone" {
			return nil, fmt.Errorf("確認: %w", errDrawioPageNotFound)
		}
		return []byte("<svg>" + pageID + "</svg>"), nil
	}}
	adf := adfDoc(strings.Join([]string{
		drawioExtension("drawio", `{"diagramName":"d.drawio","aspect":"pg2 1 L2","aspectHash":"abc"}`),
		drawioExtension("drawio", `{"diagramName":"d.drawio"}`),
		drawioExtension("drawio", `{"diagramName":"d.drawio","aspect":"pg2 1"}`),
		drawioExtension("drawio", `{"diagramName":"d.drawio","aspect":"gone 1"}`),
		drawioExtension("drawio", `{"diagramName":"d.drawio","aspect":"gone 1"}`),
		drawioExtension("drawio", `{"diagramName":"d.drawio","aspect":"pg3"}`),
		drawioExtension("drawio", `{"diagramName":"d.drawio","aspect":"undefined"}`),
		drawioExtension("drawio", `{"diagramName":"d.drawio","aspect":"  "}`),
	}, ","))
	res, err := convertADFPageWith(adf, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	page2 := `{{< drawio src="confluence-drawio-1.svg" file="d.drawio" page="pg2" title="d.drawio" >}}`
	first := `{{< drawio src="confluence-drawio-2.svg" file="d.drawio" title="d.drawio" >}}`
	page3 := `{{< drawio src="confluence-drawio-3.svg" file="d.drawio" page="pg3" title="d.drawio" >}}`
	// レイヤーの指定だけが違う図は同じ SVG を使う。無いページの指定は 1 ページ目にする
	want := strings.Join([]string{page2, first, page2, first, first, page3, first, first}, "\n\n")
	if res.Markdown != want {
		t.Errorf("got  %q\nwant %q", res.Markdown, want)
	}
	if got, want := strings.Join(asked, " "), "d.drawio#pg2 d.drawio# d.drawio#gone d.drawio#pg3"; got != want {
		t.Errorf("SVG にした図 = %q, want %q", got, want)
	}
	if len(res.Files) != 3 || string(res.Files[0].Data) != "<svg>pg2</svg>" || string(res.Files[1].Data) != "<svg></svg>" ||
		string(res.Files[2].Data) != "<svg>pg3</svg>" {
		t.Errorf("files = %+v", res.Files)
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "ページ（gone）の代わりに 1 ページ目を使います: 確認: "+errDrawioPageNotFound.Error()) {
		t.Errorf("warnings = %q, want 無いページの警告を 1 件", res.Warnings)
	}
}

// 無いページを指定した図が先にあっても、1 ページ目の SVG を作って使う
func TestConvertADFPage_DrawioPageNotFoundFirst(t *testing.T) {
	opts := adfOptions{drawioSVG: func(_, pageID string) ([]byte, error) {
		if pageID != "" {
			return nil, errDrawioPageNotFound
		}
		return []byte("<svg/>"), nil
	}}
	res, err := convertADFPageWith(adfDoc(drawioExtension("drawio", `{"diagramName":"d.drawio","aspect":"gone 1"}`)), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := `{{< drawio src="confluence-drawio-1.svg" file="d.drawio" title="d.drawio" >}}`; res.Markdown != want {
		t.Errorf("got %q, want %q", res.Markdown, want)
	}
	if len(res.Files) != 1 || len(res.Warnings) != 1 {
		t.Errorf("files = %d 件, warnings = %q", len(res.Files), res.Warnings)
	}
}

// PNG の表示に戻すとき、ページを指定した図は、そのページのプレビュー画像（<図の名前>-<aspectHash>.png）があれば使う
func TestConvertADFPage_DrawioPagePreview(t *testing.T) {
	tests := []struct {
		name        string
		guestParams string
		attachments map[string]string
		want        string
	}{
		{"ページのプレビューがある", `{"diagramName":"d.drawio","aspect":"pg2 1","aspectHash":"abc"}`,
			map[string]string{"a": "d.drawio", "b": "d.drawio.png", "c": "d.drawio-abc.png"}, "[![d.drawio](d.drawio-abc.png)](d.drawio)"},
		{"ページのプレビューが無い", `{"diagramName":"d.drawio","aspect":"pg2 1","aspectHash":"abc"}`,
			map[string]string{"a": "d.drawio", "b": "d.drawio.png"}, "[![d.drawio](d.drawio.png)](d.drawio)"},
		{"ページの指定が無い", `{"diagramName":"d.drawio","aspectHash":"abc"}`,
			map[string]string{"a": "d.drawio", "b": "d.drawio.png", "c": "d.drawio-abc.png"}, "[![d.drawio](d.drawio.png)](d.drawio)"},
		{"添付一覧が不明", `{"diagramName":"d.drawio","aspect":"pg2 1","aspectHash":"abc"}`, nil, "[![d.drawio](d.drawio.png)](d.drawio)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertADF(adfDoc(drawioExtension("drawio", tt.guestParams)), tt.attachments)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// 指定のページが図ファイルに無く、1 ページ目も SVG にできないときは、1 ページ目のプレビュー画像に戻す
func TestConvertADFPage_DrawioPageNotFoundPNG(t *testing.T) {
	opts := adfOptions{
		attachmentMap: map[string]string{"a": "d.drawio", "b": "d.drawio.png", "c": "d.drawio-abc.png"},
		drawioSVG: func(_, pageID string) ([]byte, error) {
			if pageID != "" {
				return nil, errDrawioPageNotFound
			}
			return nil, errors.New("失敗")
		},
	}
	res, err := convertADFPageWith(adfDoc(drawioExtension("drawio", `{"diagramName":"d.drawio","aspect":"gone 1","aspectHash":"abc"}`)), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "[![d.drawio](d.drawio.png)](d.drawio)"; res.Markdown != want {
		t.Errorf("got %q, want %q", res.Markdown, want)
	}
	if len(res.Warnings) != 2 {
		t.Errorf("warnings = %q, want 2 件（無いページ、SVG にできない）", res.Warnings)
	}
}

// 実際のマクロのパラメータ（2026-11-15 に Confluence Cloud で確認）。aspect は「<ページの ID> <レイヤーの ID>」で、aspectHash は無い
func TestConvertADFPage_DrawioPageRealParams(t *testing.T) {
	var asked string
	opts := adfOptions{drawioSVG: func(name, pageID string) ([]byte, error) {
		asked = name + "#" + pageID
		return []byte("<svg/>"), nil
	}}
	guest := `{"diagramName":"multipage.drawio","aspect":"L4zVVx1Hhkx3Ez1uyK2Q 1","width":1560,"links":"auto","zoom":100,"tbstyle":"top",` +
		`"pageId":"68747267","diagramDisplayName":"multipage.drawio","lbox":true,"revision":2,"height":1371}`
	res, err := convertADFPageWith(adfDoc(drawioExtension("drawio", guest)), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := `{{< drawio src="confluence-drawio-1.svg" file="multipage.drawio" page="L4zVVx1Hhkx3Ez1uyK2Q" title="multipage.drawio" >}}`; res.Markdown != want {
		t.Errorf("got %q, want %q", res.Markdown, want)
	}
	// guestParams の pageId は Confluence のページの ID で、図のページの指定には使わない
	if want := "multipage.drawio#L4zVVx1Hhkx3Ez1uyK2Q"; asked != want {
		t.Errorf("SVG にした図 = %q, want %q", asked, want)
	}
}

// 表のセルの中の図は、図の名前の | をエスケープしない（\| にすると、拡大表示が読む図ファイルの名前にバックスラッシュが入る）。
// PNG の表示に戻すときは Markdown のリンクなので \| にする
func TestConvertADFPage_DrawioInTableCellPipe(t *testing.T) {
	adf := adfDoc(`{"type":"table","content":[{"type":"tableRow","content":[` +
		`{"type":"tableCell","content":[` + drawioExtension("drawio", `{"diagramName":"a|b.drawio"}`) + `]},` +
		`{"type":"tableCell","content":[{"type":"paragraph","content":[` + adfText("x|y") + `]}]}]}]}`)
	lastLine := func(md string) string {
		lines := strings.Split(md, "\n")
		return lines[len(lines)-1]
	}

	res, err := convertADFPageWith(adf, adfOptions{drawioSVG: func(string, string) ([]byte, error) { return []byte("<svg/>"), nil }})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := `| {{< drawio src="confluence-drawio-1.svg" file="a|b.drawio" title="a|b.drawio" >}} | x\|y |`; lastLine(res.Markdown) != want {
		t.Errorf("SVG: got %q, want %q", lastLine(res.Markdown), want)
	}

	res, err = convertADFPageWith(adf, adfOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := `| [![a\|b.drawio](a\|b.drawio.png)](a\|b.drawio) | x\|y |`; lastLine(res.Markdown) != want {
		t.Errorf("PNG: got %q, want %q", lastLine(res.Markdown), want)
	}
}
