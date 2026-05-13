package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

func main() {
	// ログレベルの設定（環境変数 LOG_LEVEL で制御）
	logLevel := slog.LevelInfo
	if level := os.Getenv("LOG_LEVEL"); level == "DEBUG" {
		logLevel = slog.LevelDebug
	}

	// ログ出力先の設定
	var logWriter io.Writer = os.Stderr

	// DEBUG レベルの場合はファイルにも出力
	if logLevel == slog.LevelDebug {
		logFile, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("警告: ログファイルの作成に失敗しました: %v\n", err)
		} else {
			defer logFile.Close()
			logWriter = io.MultiWriter(os.Stderr, logFile)
		}
	}

	// slog のセットアップ
	logger := slog.New(slog.NewTextHandler(logWriter, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	if logLevel == slog.LevelDebug {
		fmt.Println("デバッグモード: ログを debug.log に保存します")
	}

	app := &cli.Command{
		Name:  "migConfluence",
		Usage: "Confluence CloudのページをAPIで取得してMarkdownに変換する",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "config.toml",
				Usage:   "設定ファイルのパス",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "page",
				Aliases: []string{"p"},
				Usage:   "単一ページを取得してMarkdownに変換する",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "page-id",
						Aliases:  []string{"id"},
						Usage:    "ページID",
						Required: true,
					},
					&cli.BoolFlag{
						Name:    "recursive",
						Aliases: []string{"r"},
						Usage:   "子ページも再帰的に取得する",
					},
					&cli.BoolFlag{
						Name:  "save-intermediate",
						Value: true,
						Usage: "中間ファイル（Confluence Storage Format）を保存する（デフォルト: true）",
					},
					&cli.BoolFlag{
						Name:  "download-attachments",
						Usage: "添付ファイルをダウンロードする",
					},
				},
				Action: fetchPage,
			},
			{
				Name:    "space",
				Aliases: []string{"s"},
				Usage:   "スペース内の全ページを取得してMarkdownに変換する",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "space-key",
						Aliases: []string{"k"},
						Usage:   "スペースキー（省略時は設定ファイルのdefault_space_keyを使用）",
					},
					&cli.BoolFlag{
						Name:  "save-intermediate",
						Value: true,
						Usage: "中間ファイル（Confluence Storage Format）を保存する（デフォルト: true）",
					},
					&cli.BoolFlag{
						Name:  "download-attachments",
						Usage: "添付ファイルをダウンロードする",
					},
				},
				Action: fetchSpace,
			},
			{
				Name:    "convert",
				Aliases: []string{"conv"},
				Usage:   "保存済み中間ファイルからMarkdownとHTMLに変換する（APIアクセス不要）",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "space-key",
						Aliases: []string{"k"},
						Usage:   "変換対象スペースキー（省略時は全スペース）",
					},
				},
				Action: convertFromIntermediate,
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
		os.Exit(1)
	}
}

// fetchPage は単一ページを取得してMarkdownに変換するコマンドハンドラ
func fetchPage(ctx context.Context, cmd *cli.Command) error {
	cfg, err := LoadConfig(cmd.Root().String("config"))
	if err != nil {
		return fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	pageID := cmd.String("page-id")
	recursive := cmd.Bool("recursive")
	saveIntermediate := cmd.Bool("save-intermediate")
	downloadAttachments := cmd.Bool("download-attachments")

	client := NewConfluenceClient(cfg.Confluence.URL, cfg.Confluence.Email, cfg.Confluence.APIToken)
	conv := NewConverter(cfg.Display.IgnoredMacros, cfg.DeletedUsers)
	writer := NewMDWriter(cfg.Output.MarkdownDir, conv)

	var intermediateSaver *IntermediateSaver
	if saveIntermediate {
		intermediateSaver = NewIntermediateSaver(cfg.Output.IntermediateDir)
	}

	var downloader *Downloader
	if downloadAttachments {
		downloader = NewDownloader(cfg.Confluence.Email, cfg.Confluence.APIToken)
	}

	if err := processPage(client, writer, intermediateSaver, downloader, cfg, pageID, recursive); err != nil {
		return err
	}

	// 未対応要素レポートの出力
	reportPath := filepath.Join(cfg.Output.MarkdownDir, "unsupported_elements.md")
	if writeErr := conv.WriteUnsupportedReport(reportPath); writeErr != nil {
		slog.Warn("未対応要素レポート出力エラー", "error", writeErr)
	}

	return nil
}

// processPage は1ページとオプションで子ページを処理する
func processPage(client *ConfluenceClient, writer *MDWriter, intermediateSaver *IntermediateSaver, downloader *Downloader, cfg *Config, pageID string, recursive bool) error {
	slog.Info("ページ取得中", "pageID", pageID)

	// ページ取得
	page, err := client.GetPage(pageID)
	if err != nil {
		return fmt.Errorf("ページ取得エラー: %w", err)
	}

	// スペース情報取得
	space, err := client.GetSpaceByID(page.SpaceID)
	if err != nil {
		slog.Warn("スペース情報取得エラー", "spaceID", page.SpaceID, "error", err)
		space = &Space{ID: page.SpaceID, Key: page.SpaceID}
	}

	// ラベル取得
	labels, err := client.GetPageLabels(pageID)
	if err != nil {
		slog.Warn("ラベル取得エラー", "pageID", pageID, "error", err)
		labels = []Label{}
	}

	// コメント取得
	comments, err := client.GetPageFooterComments(pageID)
	if err != nil {
		slog.Warn("コメント取得エラー", "pageID", pageID, "error", err)
		comments = []Comment{}
	}

	// 添付ファイル取得
	attachments, err := client.GetPageAttachments(pageID)
	if err != nil {
		slog.Warn("添付ファイル取得エラー", "pageID", pageID, "error", err)
		attachments = []Attachment{}
	}

	// 親ページのタイトルを取得
	parentTitle := ""
	if page.ParentID != "" {
		parentPage, err := client.GetPage(page.ParentID)
		if err == nil {
			parentTitle = parentPage.Title
		}
	}

	// 中間ファイルの保存
	if intermediateSaver != nil {
		if err := intermediateSaver.SavePage(page, space.Key, labels); err != nil {
			slog.Warn("中間ファイル保存エラー", "pageID", pageID, "error", err)
		}
		if len(comments) > 0 {
			if err := intermediateSaver.SaveComments(page.Title, space.Key, comments); err != nil {
				slog.Warn("コメント中間ファイル保存エラー", "pageID", pageID, "error", err)
			}
		}
	}

	// 添付ファイルのダウンロード
	if downloader != nil && len(attachments) > 0 {
		attachmentsDir := getAttachmentsDir(cfg, space.Key, page.Title)
		downloadedFiles, err := downloader.DownloadAttachments(cfg.Confluence.URL, attachments, attachmentsDir)
		if err != nil {
			slog.Warn("添付ファイルダウンロードエラー", "pageID", pageID, "error", err)
		} else {
			slog.Info("添付ファイルダウンロード完了", "count", len(downloadedFiles))
		}
	}

	// Markdown生成
	if err := writer.WritePage(page, space.Key, space.Name, parentTitle, labels, comments, attachments); err != nil {
		return fmt.Errorf("Markdown生成エラー: %w", err)
	}

	fmt.Printf("変換完了: %s/%s (%s)\n", space.Key, page.Title, pageID)

	// 子ページの再帰処理
	if recursive {
		childPages, err := client.GetChildPages(pageID)
		if err != nil {
			slog.Warn("子ページ取得エラー", "pageID", pageID, "error", err)
			return nil
		}
		for _, childPage := range childPages {
			if err := processPage(client, writer, intermediateSaver, downloader, cfg, childPage.ID, recursive); err != nil {
				slog.Warn("子ページ処理エラー", "pageID", childPage.ID, "error", err)
			}
		}
	}

	return nil
}

// fetchSpace はスペース内の全ページを取得するコマンドハンドラ
func fetchSpace(ctx context.Context, cmd *cli.Command) error {
	cfg, err := LoadConfig(cmd.Root().String("config"))
	if err != nil {
		return fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	spaceKey := cmd.String("space-key")
	if spaceKey == "" {
		spaceKey = cfg.Search.DefaultSpaceKey
	}
	if spaceKey == "" {
		return fmt.Errorf("スペースキーを指定してください（--space-key または設定ファイルのdefault_space_key）")
	}

	saveIntermediate := cmd.Bool("save-intermediate")
	downloadAttachments := cmd.Bool("download-attachments")

	client := NewConfluenceClient(cfg.Confluence.URL, cfg.Confluence.Email, cfg.Confluence.APIToken)
	conv := NewConverter(cfg.Display.IgnoredMacros, cfg.DeletedUsers)
	writer := NewMDWriter(cfg.Output.MarkdownDir, conv)

	var intermediateSaver *IntermediateSaver
	if saveIntermediate {
		intermediateSaver = NewIntermediateSaver(cfg.Output.IntermediateDir)
	}

	var downloader *Downloader
	if downloadAttachments {
		downloader = NewDownloader(cfg.Confluence.Email, cfg.Confluence.APIToken)
	}

	// スペース情報取得
	space, err := client.GetSpaceByKey(spaceKey)
	if err != nil {
		return fmt.Errorf("スペース取得エラー: %w", err)
	}

	slog.Info("スペース取得完了", "key", space.Key, "name", space.Name)

	// スペース内の全ページ取得
	pages, err := client.GetPages(space.ID)
	if err != nil {
		return fmt.Errorf("ページ一覧取得エラー: %w", err)
	}

	fmt.Printf("スペース %s (%s): %d ページ\n", space.Name, space.Key, len(pages))

	// 各ページを処理
	for i, page := range pages {
		fmt.Printf("[%d/%d] 処理中: %s (%s)\n", i+1, len(pages), page.Title, page.ID)
		if err := processPage(client, writer, intermediateSaver, downloader, cfg, page.ID, false); err != nil {
			slog.Warn("ページ処理エラー", "pageID", page.ID, "title", page.Title, "error", err)
		}
	}

	fmt.Printf("完了: %d ページを変換しました\n", len(pages))

	// 未対応要素レポートの出力
	reportPath := filepath.Join(cfg.Output.MarkdownDir, "unsupported_elements.md")
	if writeErr := conv.WriteUnsupportedReport(reportPath); writeErr != nil {
		slog.Warn("未対応要素レポート出力エラー", "error", writeErr)
	}

	return nil
}

// convertFromIntermediate は保存済み中間ファイルからMarkdownを生成するコマンドハンドラ
func convertFromIntermediate(ctx context.Context, cmd *cli.Command) error {
	cfg, err := LoadConfig(cmd.Root().String("config"))
	if err != nil {
		return fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	spaceKeyFilter := cmd.String("space-key")

	intermediateSaver := NewIntermediateSaver(cfg.Output.IntermediateDir)
	conv := NewConverter(cfg.Display.IgnoredMacros, cfg.DeletedUsers)
	writer := NewMDWriter(cfg.Output.MarkdownDir, conv)

	// 中間ファイルディレクトリを走査
	intermediateDir := cfg.Output.IntermediateDir
	entries, err := os.ReadDir(intermediateDir)
	if err != nil {
		return fmt.Errorf("中間ファイルディレクトリの読み込みに失敗しました (%s): %w", intermediateDir, err)
	}

	totalConverted := 0

	for _, spaceEntry := range entries {
		if !spaceEntry.IsDir() {
			continue
		}
		spaceKey := spaceEntry.Name()

		// スペースフィルタ
		if spaceKeyFilter != "" && spaceKey != spaceKeyFilter {
			continue
		}

		pages, err := intermediateSaver.ListPages(spaceKey)
		if err != nil {
			slog.Warn("ページ一覧取得エラー", "spaceKey", spaceKey, "error", err)
			continue
		}

		fmt.Printf("スペース %s: %d ページ\n", spaceKey, len(pages))

		for i, pageTitle := range pages {
			fmt.Printf("[%d/%d] 変換中: %s\n", i+1, len(pages), pageTitle)

			page, labels, err := intermediateSaver.LoadPage(spaceKey, pageTitle)
			if err != nil {
				slog.Warn("ページ読み込みエラー", "spaceKey", spaceKey, "pageTitle", pageTitle, "error", err)
				continue
			}

			comments, err := intermediateSaver.LoadComments(spaceKey, pageTitle)
			if err != nil {
				slog.Warn("コメント読み込みエラー", "error", err)
				comments = []Comment{}
			}

			if err := writer.WritePage(page, spaceKey, "", "", labels, comments, nil); err != nil {
				slog.Warn("Markdown生成エラー", "pageTitle", pageTitle, "error", err)
				continue
			}

			totalConverted++
		}
	}

	fmt.Printf("完了: 合計 %d ページを変換しました\n", totalConverted)

	// 未対応要素レポートの出力
	reportPath := filepath.Join(cfg.Output.MarkdownDir, "unsupported_elements.md")
	if writeErr := conv.WriteUnsupportedReport(reportPath); writeErr != nil {
		slog.Warn("未対応要素レポート出力エラー", "error", writeErr)
	}

	return nil
}

// getAttachmentsDir は添付ファイルの保存先ディレクトリを返す
func getAttachmentsDir(cfg *Config, spaceKey, pageTitle string) string {
	if cfg.Output.AttachmentsDir != "" {
		return filepath.Join(cfg.Output.AttachmentsDir, spaceKey, sanitizeFilename(pageTitle))
	}
	// デフォルトはMarkdownディレクトリ内のページディレクトリ
	return filepath.Join(cfg.Output.MarkdownDir, spaceKey, sanitizeFilename(pageTitle))
}
