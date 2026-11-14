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

	got, err := (&DrawioCLI{path: cli, timeout: drawioExportTimeout, waitDelay: drawioWaitDelay}).ExportSVG(src)
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
	for _, want := range []string{"-x -f svg", "--svg-theme light", "--embed-svg-fonts false", "diagram.drawio"} {
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
			_, err := (&DrawioCLI{path: cli, timeout: drawioExportTimeout, waitDelay: drawioWaitDelay}).ExportSVG(tt.src)
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
	_, err := (&DrawioCLI{path: cli, timeout: 300 * time.Millisecond, waitDelay: 300 * time.Millisecond}).ExportSVG(src)
	if err == nil || !strings.Contains(err.Error(), "終わりませんでした") {
		t.Errorf("err = %v, want 打ち切りのエラー", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("打ち切りに %s かかりました", elapsed)
	}
}
