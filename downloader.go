package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Downloader は添付ファイルのダウンロードを管理する
type Downloader struct {
	client   *http.Client
	email    string
	apiToken string
}

// NewDownloader は新しいDownloaderを作成する
func NewDownloader(email, apiToken string) *Downloader {
	return &Downloader{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		email:    email,
		apiToken: apiToken,
	}
}

// DownloadAttachments はページの添付ファイルを指定ディレクトリにすべてダウンロードする。
// 一部の添付で失敗しても残りの添付は続け、失敗をまとめたエラーを、保存できたファイル名と一緒に返す
func (d *Downloader) DownloadAttachments(baseURL string, attachments []Attachment, targetDir string) ([]string, error) {
	if len(attachments) == 0 {
		return []string{}, nil
	}

	// 出力ディレクトリの作成
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("添付ファイルディレクトリの作成に失敗しました: %w", err)
	}

	var downloadedFiles []string
	var errs []error

	for _, attachment := range attachments {
		filename, err := d.downloadFile(baseURL, attachment, targetDir)
		if err != nil {
			errs = append(errs, fmt.Errorf("添付ファイル %s のダウンロードに失敗しました: %w", attachment.Title, err))
			continue
		}
		downloadedFiles = append(downloadedFiles, filename)
	}

	return downloadedFiles, errors.Join(errs...)
}

// downloadFile は単一の添付ファイルを指定ディレクトリにダウンロードする。
// 保存済みのファイルが Confluence 側の最新の版と同じなら、ダウンロードしない（isUpToDate 参照）
func (d *Downloader) downloadFile(baseURL string, attachment Attachment, targetDir string) (string, error) {
	safeFilename := sanitizeFilename(attachment.Title)
	destPath := filepath.Join(targetDir, safeFilename)

	versionTime := attachmentVersionTime(attachment)
	info, statErr := os.Stat(destPath)
	if statErr == nil {
		if isUpToDate(info, attachment, versionTime) {
			slog.Debug("添付ファイルは最新のためダウンロードしません", "file", safeFilename)
			return safeFilename, nil
		}
		slog.Info("保存済みの添付ファイルが Confluence の最新の版と違うためダウンロードし直します",
			"file", safeFilename, "version", attachment.Version.Number, "updatedAt", attachment.Version.CreatedAt)
	}

	// HTTPリクエストの作成
	req, err := http.NewRequest("GET", attachmentDownloadURL(baseURL, attachment), nil)
	if err != nil {
		return "", fmt.Errorf("HTTPリクエストの作成に失敗しました: %w", err)
	}

	// Basic認証ヘッダーの設定
	req.SetBasicAuth(d.email, d.apiToken)

	// ファイルのダウンロード
	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTPリクエストに失敗しました: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Warn("レスポンスボディのクローズに失敗しました", "file", safeFilename, "error", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ダウンロードに失敗しました。ステータスコード: %d", resp.StatusCode)
	}

	if err := saveAttachment(destPath, resp.Body, attachment.FileSize, versionTime); err != nil {
		return "", err
	}
	return safeFilename, nil
}

// modTimeTolerance は、保存済みのファイルの更新日時と版の作成日時を同じとみなす差の上限
const modTimeTolerance = 2 * time.Second

// attachmentVersionTime は添付ファイルの最新の版の作成日時を返す。分からなければゼロ値を返す
func attachmentVersionTime(attachment Attachment) time.Time {
	t, err := time.Parse(time.RFC3339Nano, attachment.Version.CreatedAt)
	if err != nil {
		return time.Time{}
	}
	return t
}

// isUpToDate は、保存済みのファイルが Confluence 側の最新の版と同じかどうかを返す。
// 保存時にファイルの更新日時を版の作成日時に合わせてあるので（saveAttachment 参照）、
// 更新日時が版の作成日時と同じで、サイズも同じなら、同じ版とみなす。
// 「手元の方が新しいか」ではなく「同じか」で比べるのは、更新日時を合わせる前に保存したファイルや、
// Confluence で古い版に戻された添付も取り直すため。
// 版の作成日時が分からないときは、確かめようがないので false を返す（取り直す）
func isUpToDate(info os.FileInfo, attachment Attachment, versionTime time.Time) bool {
	if versionTime.IsZero() {
		return false
	}
	// Confluence がサイズを返さない（0）ときは、更新日時だけで比べる
	if attachment.FileSize > 0 && info.Size() != attachment.FileSize {
		return false
	}
	// ファイルシステムによって更新日時の細かさが違う（2 秒単位のものもある）ので、
	// modTimeTolerance 以内の違いは同じとみなす
	diff := info.ModTime().Sub(versionTime)
	return diff >= -modTimeTolerance && diff <= modTimeTolerance
}

// saveAttachment は body を destPath に保存し、更新日時を modTime に合わせる。
// 同じディレクトリの作業用のファイルに書いてから名前を変えるので、
// 途中で失敗しても、保存済みのファイルは元のまま残る。
// 更新日時は「この版を保存した」という印になる（isUpToDate 参照）ので、次のときは合わせない。
// その場合、次回の移行でもう一度ダウンロードする
//   - modTime がゼロ値（版の作成日時が分からない）
//   - 書いた長さが wantSize（Confluence のファイルサイズ。0 なら確かめない）と違う（途中で切れた本文など）
//   - 更新日時を設定できなかった
func saveAttachment(destPath string, body io.Reader, wantSize int64, modTime time.Time) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(destPath), ".download-*.tmp")
	if err != nil {
		return fmt.Errorf("ファイルの作成に失敗しました: %w", err)
	}
	tmpPath := tmpFile.Name()
	saved := false
	defer func() {
		if saved {
			return
		}
		if rerr := os.Remove(tmpPath); rerr != nil && !os.IsNotExist(rerr) {
			slog.Warn("作業用のファイルの削除に失敗しました", "path", tmpPath, "error", rerr)
		}
	}()

	written, err := io.Copy(tmpFile, body)
	if err != nil {
		if cerr := tmpFile.Close(); cerr != nil {
			slog.Warn("添付ファイルのクローズに失敗しました", "path", tmpPath, "error", cerr)
		}
		return fmt.Errorf("ファイルの書き込みに失敗しました: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("ファイルの書き込みに失敗しました: %w", err)
	}
	// os.CreateTemp は 0600 で作るので、ほかの出力ファイルと同じ 0644 にする
	if err := os.Chmod(tmpPath, 0644); err != nil {
		slog.Warn("添付ファイルの権限の設定に失敗しました", "path", destPath, "error", err)
	}
	switch {
	case modTime.IsZero():
	case wantSize > 0 && written != wantSize:
		slog.Warn("添付ファイルのサイズが Confluence の情報と違います。次回の移行でダウンロードし直します",
			"path", destPath, "size", written, "expected", wantSize)
	default:
		if err := os.Chtimes(tmpPath, modTime, modTime); err != nil {
			slog.Warn("添付ファイルの更新日時の設定に失敗しました。次回の移行でダウンロードし直します",
				"path", destPath, "error", err)
		}
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("ファイルの保存に失敗しました: %w", err)
	}
	saved = true
	return nil
}

// attachmentDownloadURL は添付ファイルのダウンロードURLを返す。
// _links.download があればそれを使い、なければ v1 のダウンロードURLを組み立てる。
// v2 API の _links.download は /wiki を基準にした相対パス（/rest/api/content/…/download）なので、
// baseURL（https://xxx.atlassian.net）に付けるときは /wiki を補う
func attachmentDownloadURL(baseURL string, attachment Attachment) string {
	baseURL = strings.TrimRight(baseURL, "/")
	downloadURL := attachment.Links.Download
	switch {
	case downloadURL == "":
		return fmt.Sprintf("%s/wiki/rest/api/content/%s/child/attachment/%s/download",
			baseURL, attachment.PageID, attachment.ID)
	case strings.HasPrefix(downloadURL, "http://"), strings.HasPrefix(downloadURL, "https://"):
		return downloadURL
	case strings.HasPrefix(downloadURL, "/wiki/"):
		return baseURL + downloadURL
	default:
		return baseURL + "/wiki" + downloadURL
	}
}

// sanitizeFilename はファイル名を安全な形式にサニタイズする
func sanitizeFilename(filename string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		"..", "_",
		":", "_",
	)
	return replacer.Replace(filename)
}

// IsImageFile はファイル名が画像ファイルかどうかを判定する
func IsImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	imageExts := []string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".bmp", ".ico"}
	for _, imageExt := range imageExts {
		if ext == imageExt {
			return true
		}
	}
	return false
}
