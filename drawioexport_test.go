package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestFindDrawioCLI(t *testing.T) {
	dir := t.TempDir()
	installed := filepath.Join(dir, "draw.io")
	if err := os.WriteFile(installed, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	notFound := func(string) (string, error) { return "", errors.New("not found") }
	inPath := func(name string) (string, error) {
		if name == "drawio" {
			return "/usr/local/bin/drawio", nil
		}
		return "", errors.New("not found")
	}
	onlyConfigured := func(name string) (string, error) {
		if name == "/opt/drawio" {
			return name, nil
		}
		return "", errors.New("not found")
	}

	tests := []struct {
		name       string
		configured string
		lookPath   func(string) (string, error)
		defaults   []string
		want       string
		wantErr    string
	}{
		{name: "設定のパスを優先する", configured: "/opt/drawio", lookPath: onlyConfigured, defaults: []string{installed}, want: "/opt/drawio"},
		{name: "設定のパスを実行できなければエラー（ほかは探さない）", configured: "/nowhere/drawio", lookPath: inPath, defaults: []string{installed}, wantErr: "drawio.cli_path"},
		{name: "PATH から探す", lookPath: inPath, defaults: []string{installed}, want: "/usr/local/bin/drawio"},
		{name: "PATH に無ければ既定のインストール先", lookPath: notFound, defaults: []string{filepath.Join(dir, "none"), installed}, want: installed},
		{name: "フォルダは既定のインストール先として扱わない", lookPath: notFound, defaults: []string{dir}, wantErr: "見つかりません"},
		{name: "どこにも無ければエラー", lookPath: notFound, wantErr: "見つかりません"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findDrawioCLI(tt.configured, tt.lookPath, tt.defaults)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q を含むエラー", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInlineSVG(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="10px"><g><svg x="1"></svg></g></svg>`
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "XML 宣言と DOCTYPE を取り除く", in: "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE svg PUBLIC \"-//W3C//DTD SVG 1.1//EN\" \"http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd\">\n" + svg + "\n", want: svg},
		{name: "<svg> だけならそのまま", in: svg, want: svg},
		{name: "SVG でなければエラー", in: "Error: something", wantErr: true},
		{name: "閉じタグが無ければエラー", in: "<svg>", wantErr: true},
		{name: "script 要素を含めばエラー", in: "<svg><SCRIPT>alert(1)</SCRIPT></svg>", wantErr: true},
		{name: "javascript: のリンクを含めばエラー", in: `<svg><a xlink:href="JavaScript:alert(1)">x</a></svg>`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := inlineSVG([]byte(tt.in))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("got %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestCountLocalFileRefs は、SVG に残った file:// の参照を数えるテスト
func TestCountLocalFileRefs(t *testing.T) {
	tests := []struct {
		name string
		svg  string
		want int
	}{
		{name: "参照が無い", svg: `<svg><rect/></svg>`, want: 0},
		{name: "埋め込まれた画像と外部の画像は数えない",
			svg: `<svg><image xlink:href="data:image/svg+xml;base64,AAAA"/><image xlink:href="https://example.com/a.png"/></svg>`, want: 0},
		{name: "draw.io のアプリの中のアイコン",
			svg: `<svg><image xlink:href="file:///Applications/draw.io.app/Contents/Resources/app.asar/drawio/src/main/webapp/img/lib/azure2/other/x.svg"/></svg>`, want: 1},
		{name: "href でも大文字でも数える",
			svg: `<svg><image href="FILE:///a.png"/><image xlink:href='file:///b.png'/></svg>`, want: 2},
		{name: "図の中の文字に書かれた file:// は数えない", svg: `<svg><text>file:///etc/hosts</text></svg>`, want: 0},
		{name: "img 要素の src も数える", svg: `<svg><foreignObject><img src="file:///a.png"/></foreignObject></svg>`, want: 1},
		{name: "図形に付けたリンクは数えない", svg: `<svg><a xlink:href="file://server/share/doc.pdf"><rect/></a></svg>`, want: 0},
		{name: "空白があっても数える", svg: `<svg><image xlink:href = " file:///a.png"/></svg>`, want: 1},
		{name: "CSS の url() は数えない（draw.io が書き出さない形）", svg: `<svg><rect style="fill:url(file:///a.png)"/></svg>`, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countLocalFileRefs([]byte(tt.svg)); got != tt.want {
				t.Errorf("countLocalFileRefs() = %d, want %d", got, tt.want)
			}
		})
	}
}

// fakeDrawioCLI は draw.io のコマンドの代わりになるシェルスクリプトを作る。
// 受け取った引数を args.txt に残し、script の内容を実行する（$out が出力先、$in が入力）
func fakeDrawioCLI(t *testing.T, script string) (cli, argsFile string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("シェルスクリプトを使うため Windows では実行しない")
	}
	dir := t.TempDir()
	argsFile = filepath.Join(dir, "args.txt")
	cli = filepath.Join(dir, "drawio")
	body := "#!/bin/sh\necho \"$@\" > '" + argsFile + "'\n" +
		"while [ $# -gt 1 ]; do if [ \"$1\" = \"-o\" ]; then out=\"$2\"; fi; shift; done\nin=\"$1\"\n" + script + "\n"
	if err := os.WriteFile(cli, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
	return cli, argsFile
}

func TestDrawioCLI_ExportSVG(t *testing.T) {
	// 入力の内容を SVG の中に写すことで、拡張子の無い図ファイルが一時フォルダ経由で渡ることも確かめる
	cli, argsFile := fakeDrawioCLI(t, `printf '<?xml version="1.0"?>\n<svg data-src="%s"></svg>\n' "$(cat "$in")" > "$out"`)
	src := filepath.Join(t.TempDir(), "図 (1)")
	if err := os.WriteFile(src, []byte("mxfile-body"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := (&DrawioCLI{path: cli, timeout: drawioExportTimeout, waitDelay: drawioWaitDelay}).ExportSVG(src, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := `<svg data-src="mxfile-body"></svg>`; string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"-x -f svg", "--svg-theme light", "--embed-svg-fonts false", "--embed-svg-images", "--page-index 2", "diagram.drawio"} {
		if !strings.Contains(string(args), want) {
			t.Errorf("args = %q, want %q を含む", args, want)
		}
	}
}

func TestDrawioCLI_ExportSVG_Errors(t *testing.T) {
	src := filepath.Join(t.TempDir(), "a.drawio")
	if err := os.WriteFile(src, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		script  string
		src     string
		wantErr string
	}{
		{name: "コマンドが失敗", script: `echo "Error: boom" >&2; exit 1`, src: src, wantErr: "Error: boom"},
		{name: "SVG を出力しない", script: `exit 0`, src: src, wantErr: "出力しませんでした"},
		{name: "出力が SVG でない", script: `echo "not svg" > "$out"`, src: src, wantErr: "SVG ではありません"},
		{name: "図ファイルが無い", script: `exit 0`, src: filepath.Join(t.TempDir(), "none"), wantErr: "図ファイルを読めません"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli, _ := fakeDrawioCLI(t, tt.script)
			_, err := (&DrawioCLI{path: cli, timeout: drawioExportTimeout, waitDelay: drawioWaitDelay}).ExportSVG(tt.src, 0)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("err = %v, want %q を含むエラー", err, tt.wantErr)
			}
		})
	}
}

// 変換が終わらないときは打ち切る。コマンドが残した子プロセスが出力を握っていても戻ってくる
func TestDrawioCLI_ExportSVG_Timeout(t *testing.T) {
	cli, _ := fakeDrawioCLI(t, `sleep 30 &
sleep 30`)
	src := filepath.Join(t.TempDir(), "a.drawio")
	if err := os.WriteFile(src, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, err := (&DrawioCLI{path: cli, timeout: 300 * time.Millisecond, waitDelay: 300 * time.Millisecond}).ExportSVG(src, 0)
	if err == nil || !strings.Contains(err.Error(), "終わりませんでした") {
		t.Errorf("err = %v, want 打ち切りのエラー", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("打ち切りに %s かかりました", elapsed)
	}
}

func TestDrawioPageIndex(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	// 2 ページ目は中身が圧縮された形。ページの中の要素の id は拾わない
	multi := write("multi", `<mxfile host="x"><diagram id="pg1" name="1"><mxGraphModel><root><mxCell id="0"/><mxCell id="pg3x"/></root></mxGraphModel></diagram>`+
		`<diagram name="2" id="pg2">7VhLc9owEP41</diagram><diagram id="pg3" name="3"><mxGraphModel/></diagram></mxfile>`)
	tests := []struct {
		name    string
		src     string
		pageID  string
		want    int
		wantErr string
	}{
		{name: "1 ページ目", src: multi, pageID: "pg1", want: 0},
		{name: "2 ページ目", src: multi, pageID: "pg2", want: 1},
		{name: "3 ページ目", src: multi, pageID: "pg3", want: 2},
		{name: "無いページ", src: multi, pageID: "gone", wantErr: errDrawioPageNotFound.Error()},
		{name: "ページの中の要素の id", src: multi, pageID: "pg3x", wantErr: errDrawioPageNotFound.Error()},
		{name: "図ファイルが無い", src: filepath.Join(dir, "none"), pageID: "pg1", wantErr: "図ファイルを読めません"},
		{name: "XML でない", src: write("broken", `<mxfile><diagram id="a"></mxfile>`), pageID: "pg1", wantErr: errDrawioPageNotFound.Error() + "（図ファイルを解析できません"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := drawioPageIndex(tt.src, tt.pageID)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q を含むエラー", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("drawioPageIndex() = %d, %v, want %d", got, err, tt.want)
			}
		})
	}
	if _, err := drawioPageIndex(multi, "gone"); !errors.Is(err, errDrawioPageNotFound) {
		t.Errorf("err = %v, want errDrawioPageNotFound", err)
	}
}
