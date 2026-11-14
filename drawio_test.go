package main

import (
	"errors"
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
		drawioSVG: func(name string) ([]byte, error) {
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
		drawioSVG: func(name string) ([]byte, error) {
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
	opts := adfOptions{drawioSVG: func(name string) ([]byte, error) {
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
