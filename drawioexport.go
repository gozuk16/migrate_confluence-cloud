package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// DrawioExporter は draw.io の図ファイル（.drawio）を、ページに直接埋め込める SVG に変換する
type DrawioExporter interface {
	ExportSVG(srcPath string) ([]byte, error)
}

// drawioCLINames は PATH から探す draw.io デスクトップ版のコマンド名
var drawioCLINames = []string{"drawio", "draw.io"}

// drawioCLIDefaultPaths は PATH に無いときに探す、draw.io デスクトップ版の既定のインストール先
var drawioCLIDefaultPaths = []string{
	"/Applications/draw.io.app/Contents/MacOS/draw.io",
	`C:\Program Files\draw.io\draw.io.exe`,
}

// drawioExportTimeout は 1 つの図の変換を待つ時間
const drawioExportTimeout = 60 * time.Second

// drawioWaitDelay は変換を打ち切ったあと、コマンドの出力が閉じるのを待つ時間
const drawioWaitDelay = 5 * time.Second

// DrawioCLI は draw.io デスクトップ版のコマンドラインで SVG に変換する DrawioExporter
type DrawioCLI struct {
	path      string
	timeout   time.Duration // 1 つの図の変換を待つ時間
	waitDelay time.Duration // 変換を打ち切ったあと、コマンドの出力が閉じるのを待つ時間
}

// NewDrawioCLI は draw.io デスクトップ版のコマンドを探して DrawioCLI を作る。
// configured（設定ファイルの drawio.cli_path）があればそれを使い、無ければ PATH と既定のインストール先を探す
func NewDrawioCLI(configured string) (*DrawioCLI, error) {
	path, err := findDrawioCLI(configured, exec.LookPath, drawioCLIDefaultPaths)
	if err != nil {
		return nil, err
	}
	return &DrawioCLI{path: path, timeout: drawioExportTimeout, waitDelay: drawioWaitDelay}, nil
}

// findDrawioCLI は draw.io デスクトップ版のコマンドのパスを返す
func findDrawioCLI(configured string, lookPath func(string) (string, error), defaultPaths []string) (string, error) {
	if configured != "" {
		if p, err := lookPath(configured); err == nil {
			return p, nil
		}
		return "", fmt.Errorf("設定の drawio.cli_path（%s）を実行できません", configured)
	}
	for _, name := range drawioCLINames {
		if p, err := lookPath(name); err == nil {
			return p, nil
		}
	}
	for _, p := range defaultPaths {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, nil
		}
	}
	return "", errors.New("draw.io デスクトップ版のコマンドが見つかりません（インストールするか、設定の drawio.cli_path で場所を指定してください）")
}

// ExportSVG は図ファイルの 1 ページ目を SVG に変換する。
// 明るい配色に固定し（--svg-theme light。既定の auto は OS のダークモードで線と文字が白くなる）、
// フォントは埋め込まない（--embed-svg-fonts false。文字をそのまま残し、外部のフォントも取りに行かない）。
// 画像は SVG の中に埋め込む（--embed-svg-images）。付けないと、図形ライブラリのアイコン（Azure・AWS など）が
// draw.io のアプリの中のファイル（file:///…/app.asar/…）を指したままになり、ブラウザで表示できない。
// 外部 URL の画像も取得して埋め込むが、取得できなかったものは URL のまま残る。
// 時間内に終わらなければ（応答の返らない URL の画像があるときなど）、画像を埋め込まずに 1 回だけやり直す
// 添付ファイルは拡張子が無いことや、コマンドの引数として扱いにくい名前のことがあるので、一時フォルダに写してから変換する
func (c *DrawioCLI) ExportSVG(srcPath string) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "drawio-export-")
	if err != nil {
		return nil, fmt.Errorf("一時フォルダの作成に失敗しました: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	in := filepath.Join(tmpDir, "diagram.drawio")
	out := filepath.Join(tmpDir, "diagram.svg")
	if err := copyFile(srcPath, in); err != nil {
		return nil, fmt.Errorf("図ファイルを読めません: %w", err)
	}

	err = c.run(in, out, true)
	if errors.Is(err, errDrawioTimeout) {
		// 応答の返らない URL の画像があると、draw.io が画像の取得を待ち続けて終わらない。
		// 図そのものは表示できるように、画像を埋め込まずにやり直す
		slog.Warn("draw.io の変換が時間内に終わらないため、画像を埋め込まずにやり直します（図の中の画像の URL に届かない可能性があります）",
			"file", srcPath, "timeout", c.timeout)
		err = c.run(in, out, false)
	}
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(out)
	if err != nil {
		return nil, errors.New("draw.io が SVG を出力しませんでした")
	}
	return inlineSVG(data)
}

// errDrawioTimeout は draw.io の変換が時間内に終わらなかったことを表す
var errDrawioTimeout = errors.New("draw.io の変換が時間内に終わりませんでした")

// run は draw.io のコマンドで in を SVG にして out に書き出す。embedImages が true なら画像を SVG の中に埋め込む。
// timeout で打ち切ったときは errDrawioTimeout を包んだエラーを返す
func (c *DrawioCLI) run(in, out string, embedImages bool) error {
	args := []string{"-x", "-f", "svg", "--svg-theme", "light", "--embed-svg-fonts", "false"}
	if embedImages {
		args = append(args, "--embed-svg-images")
	}
	args = append(args, "-o", out, in)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.path, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	// draw.io は Electron で、打ち切ったあとも補助プロセスがエラー出力を握ったまま残ることがある。
	// その場合でも Run が戻るように、打ち切りから少し待って出力の読み取りをやめる
	cmd.WaitDelay = c.waitDelay
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w（%s）", errDrawioTimeout, c.timeout)
		}
		return fmt.Errorf("draw.io の変換に失敗しました: %w: %s", err, lastLine(stderr.String()))
	}
	if _, err := os.Stat(out); err != nil {
		return fmt.Errorf("draw.io が SVG を出力しませんでした: %s", lastLine(stderr.String()))
	}
	return nil
}

// inlineSVG は SVG ファイルの内容から、HTML に直接埋め込める <svg>…</svg> の部分だけを取り出す
// （XML 宣言と DOCTYPE は HTML の中では不正なので取り除く）。
// スクリプト（<script>、javascript: のリンク）を含む SVG はエラーにする。イベント属性（onclick など）までは調べないので、
// 図の内容は信頼できるもの（自社の Confluence の図）であることを前提にしている
func inlineSVG(data []byte) ([]byte, error) {
	start := bytes.Index(data, []byte("<svg"))
	end := bytes.LastIndex(data, []byte("</svg>"))
	if start < 0 || end < start {
		return nil, errors.New("draw.io の出力が SVG ではありません")
	}
	svg := data[start : end+len("</svg>")]
	// ページに直接埋め込む SVG の中のスクリプトは、サイトのページとして実行されてしまうので埋め込まない
	lower := bytes.ToLower(svg)
	for _, bad := range []string{"<script", "javascript:"} {
		if bytes.Contains(lower, []byte(bad)) {
			return nil, fmt.Errorf("SVG にスクリプト（%s）が含まれるため埋め込みません", bad)
		}
	}
	return svg, nil
}

// localFileRefPattern は、SVG の属性（href・xlink:href・src）の値が file: で始まる箇所。
// 警告を出すためのものなので、draw.io が書き出す形だけを拾う（CSS の url(…) や文字参照で書かれたものは拾わない）
var localFileRefPattern = regexp.MustCompile(`(?i)(?:href|src)\s*=\s*["']\s*file:`)

// countLocalFileRefs は、SVG の中の、ローカルのファイル（file://）への参照の数を返す。
// ブラウザはサイトのページから file:// を読めないので、残っていればその画像は表示されない
// （--embed-svg-images が効かない古い draw.io、draw.io に無い図形ライブラリの画像、埋め込まずにやり直した図で起こりうる）
func countLocalFileRefs(svg []byte) int {
	return len(localFileRefPattern.FindAll(svg, -1))
}

// copyFile は src を dst に写す
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// lastLine は空でない最後の行を返す（コマンドのエラー出力の要約に使う）
func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
