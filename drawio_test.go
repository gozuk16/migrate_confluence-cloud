package main

import (
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
