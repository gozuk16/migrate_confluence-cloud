package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config はアプリケーション設定を表す構造体
type Config struct {
	Confluence   ConfluenceConfig  `toml:"confluence"`
	Output       OutputConfig      `toml:"output"`
	Search       SearchConfig      `toml:"search"`
	Display      DisplayConfig     `toml:"display"`
	DeletedUsers map[string]string `toml:"deletedUsers"` // 削除済みユーザーのマッピング（accountId -> displayName）
}

// ConfluenceConfig はConfluence接続情報を表す構造体
type ConfluenceConfig struct {
	URL      string `toml:"url"`       // Confluence Cloud URL (例: https://your-domain.atlassian.net)
	Email    string `toml:"email"`     // Confluenceユーザーのメールアドレス
	APIToken string `toml:"api_token"` // Confluence API Token
}

// OutputConfig は出力設定を表す構造体
type OutputConfig struct {
	MarkdownDir     string `toml:"markdown_dir"`     // Markdown出力ディレクトリ
	AttachmentsDir  string `toml:"attachments_dir"`  // 添付ファイル保存ディレクトリ（空の場合はmarkdown_dir内に配置）
	IntermediateDir string `toml:"intermediate_dir"` // 中間ファイル保存ディレクトリ（Confluence Storage Format）
	HTMLDir         string `toml:"html_dir"`         // HTML出力ディレクトリ
}

// SearchConfig は検索設定を表す構造体
type SearchConfig struct {
	DefaultSpaceKey string `toml:"default_space_key"` // デフォルトのスペースキー
}

// DisplayConfig は表示設定を表す構造体
type DisplayConfig struct {
	IgnoredMacros []string `toml:"ignored_macros"` // 無視するマクロ名のリスト
}

// LoadConfig は指定されたパスからTOML設定ファイルを読み込む
func LoadConfig(path string) (*Config, error) {
	var config Config

	// ファイルの存在確認
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("設定ファイルが見つかりません: %s", path)
	}

	// TOMLファイルをデコード
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	// バリデーション
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("設定ファイルのバリデーションエラー: %w", err)
	}

	return &config, nil
}

// Validate は設定値の妥当性をチェックする
func (c *Config) Validate() error {
	if c.Confluence.URL == "" {
		return fmt.Errorf("confluence.urlが設定されていません")
	}
	if c.Confluence.Email == "" {
		return fmt.Errorf("confluence.emailが設定されていません")
	}
	if c.Confluence.APIToken == "" {
		return fmt.Errorf("confluence.api_tokenが設定されていません")
	}

	// デフォルト値の設定
	if c.Output.MarkdownDir == "" {
		c.Output.MarkdownDir = "output/markdown"
	}
	if c.Output.IntermediateDir == "" {
		c.Output.IntermediateDir = "output/intermediate"
	}
	if c.Output.HTMLDir == "" {
		c.Output.HTMLDir = "output/html"
	}

	return nil
}
