package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config はアプリケーション設定を表す構造体
type Config struct {
	Confluence   ConfluenceConfig  `toml:"confluence"`
	Output       OutputConfig      `toml:"output"`
	Search       SearchConfig      `toml:"search"`
	Display      DisplayConfig     `toml:"display"`
	Drawio       DrawioConfig      `toml:"drawio"`
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
}

// SearchConfig は検索設定を表す構造体
type SearchConfig struct {
	DefaultSpaceKey string   `toml:"default_space_key"` // デフォルトのスペースキー（space_keys が無いときに使う。後方互換のため残す）
	SpaceKeys       []string `toml:"space_keys"`        // space コマンドで移行するスペースキーのリスト（default_space_key より優先する）
}

// DisplayConfig は表示設定を表す構造体
type DisplayConfig struct {
	IgnoredMacros []string `toml:"ignored_macros"` // 無視するマクロ名のリスト
}

// DrawioConfig は draw.io の図の変換設定を表す構造体
type DrawioConfig struct {
	CLIPath string `toml:"cli_path"` // draw.io デスクトップ版のコマンドのパス（空の場合は PATH と既定のインストール先から探す）
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

	return nil
}

// ResolveSpaceKeys は space コマンドで移行するスペースキーを決める。
// 優先順位は、引数（--space-key）> 設定ファイルの space_keys > default_space_key。
// 上位の指定が空（空白だけの要素を含む）なら、次の指定を使う。
// 各要素はカンマで区切って空白を取り除き、空の要素と重複を除く（最初に現れた順番を保つ）。
func ResolveSpaceKeys(cliKeys []string, search SearchConfig) []string {
	for _, keys := range [][]string{cliKeys, search.SpaceKeys, {search.DefaultSpaceKey}} {
		if normalized := normalizeSpaceKeys(keys); len(normalized) > 0 {
			return normalized
		}
	}
	return nil
}

// normalizeSpaceKeys はスペースキーのリストを、カンマで区切り、空白・空の要素・重複を除いた形にする
func normalizeSpaceKeys(keys []string) []string {
	var result []string
	seen := map[string]bool{}
	for _, k := range keys {
		for _, part := range strings.Split(k, ",") {
			part = strings.TrimSpace(part)
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			result = append(result, part)
		}
	}
	return result
}
