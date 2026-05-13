package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadConfig はLoadConfig関数のテスト
func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		wantErr     bool
		errContains string
	}{
		{
			name: "正常系: 有効な設定ファイル",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.toml")
				content := `[confluence]
url = "https://test.atlassian.net"
email = "test@example.com"
api_token = "test-token-123"

[output]
markdown_dir = "output/markdown"
intermediate_dir = "output/intermediate"

[search]
default_space_key = "TEST"
`
				if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
					t.Fatalf("テスト用設定ファイルの作成に失敗: %v", err)
				}
				return configPath
			},
			wantErr: false,
		},
		{
			name: "異常系: ファイルが存在しない",
			setupFunc: func(t *testing.T) string {
				return "/path/to/nonexistent/config.toml"
			},
			wantErr:     true,
			errContains: "設定ファイルが見つかりません",
		},
		{
			name: "異常系: 無効なTOML形式",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.toml")
				content := `[confluence
invalid toml syntax
`
				if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
					t.Fatalf("テスト用設定ファイルの作成に失敗: %v", err)
				}
				return configPath
			},
			wantErr:     true,
			errContains: "設定ファイルの読み込みに失敗しました",
		},
		{
			name: "異常系: 必須項目が欠落（confluence.url）",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.toml")
				content := `[confluence]
email = "test@example.com"
api_token = "test-token-123"
`
				if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
					t.Fatalf("テスト用設定ファイルの作成に失敗: %v", err)
				}
				return configPath
			},
			wantErr:     true,
			errContains: "confluence.urlが設定されていません",
		},
		{
			name: "異常系: 必須項目が欠落（confluence.email）",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.toml")
				content := `[confluence]
url = "https://test.atlassian.net"
api_token = "test-token-123"
`
				if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
					t.Fatalf("テスト用設定ファイルの作成に失敗: %v", err)
				}
				return configPath
			},
			wantErr:     true,
			errContains: "confluence.emailが設定されていません",
		},
		{
			name: "異常系: 必須項目が欠落（confluence.api_token）",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.toml")
				content := `[confluence]
url = "https://test.atlassian.net"
email = "test@example.com"
`
				if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
					t.Fatalf("テスト用設定ファイルの作成に失敗: %v", err)
				}
				return configPath
			},
			wantErr:     true,
			errContains: "confluence.api_tokenが設定されていません",
		},
		{
			name: "正常系: デフォルト値が設定される（output未指定）",
			setupFunc: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.toml")
				content := `[confluence]
url = "https://test.atlassian.net"
email = "test@example.com"
api_token = "test-token-123"
`
				if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
					t.Fatalf("テスト用設定ファイルの作成に失敗: %v", err)
				}
				return configPath
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupFunc(t)
			config, err := LoadConfig(configPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("エラーが期待されましたが、nilが返されました")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("エラーメッセージが期待と異なります\n期待: %q を含む\n実際: %q",
						tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("予期しないエラー: %v", err)
				return
			}

			if config == nil {
				t.Error("設定がnilです")
				return
			}

			if config.Confluence.URL == "" {
				t.Error("Confluence URLが空です")
			}
			if config.Confluence.Email == "" {
				t.Error("Confluence Emailが空です")
			}
			if config.Confluence.APIToken == "" {
				t.Error("Confluence APITokenが空です")
			}

			// デフォルト値のテスト
			if strings.Contains(tt.name, "デフォルト値") {
				if config.Output.MarkdownDir != "output/markdown" {
					t.Errorf("MarkdownDirのデフォルト値が期待と異なります: %q", config.Output.MarkdownDir)
				}
				if config.Output.IntermediateDir != "output/intermediate" {
					t.Errorf("IntermediateDirのデフォルト値が期待と異なります: %q", config.Output.IntermediateDir)
				}
			}
		})
	}
}

// TestValidate はValidateメソッドのテスト
func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantErr     bool
		errContains string
	}{
		{
			name: "正常系: すべての必須項目が設定されている",
			config: Config{
				Confluence: ConfluenceConfig{
					URL:      "https://test.atlassian.net",
					Email:    "test@example.com",
					APIToken: "test-token-123",
				},
				Output: OutputConfig{
					MarkdownDir:     "output/markdown",
					IntermediateDir: "output/intermediate",
				},
			},
			wantErr: false,
		},
		{
			name: "異常系: confluence.urlが空",
			config: Config{
				Confluence: ConfluenceConfig{
					URL:      "",
					Email:    "test@example.com",
					APIToken: "test-token-123",
				},
			},
			wantErr:     true,
			errContains: "confluence.urlが設定されていません",
		},
		{
			name: "異常系: confluence.emailが空",
			config: Config{
				Confluence: ConfluenceConfig{
					URL:      "https://test.atlassian.net",
					Email:    "",
					APIToken: "test-token-123",
				},
			},
			wantErr:     true,
			errContains: "confluence.emailが設定されていません",
		},
		{
			name: "異常系: confluence.api_tokenが空",
			config: Config{
				Confluence: ConfluenceConfig{
					URL:      "https://test.atlassian.net",
					Email:    "test@example.com",
					APIToken: "",
				},
			},
			wantErr:     true,
			errContains: "confluence.api_tokenが設定されていません",
		},
		{
			name: "正常系: デフォルト値が設定される",
			config: Config{
				Confluence: ConfluenceConfig{
					URL:      "https://test.atlassian.net",
					Email:    "test@example.com",
					APIToken: "test-token-123",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("エラーが期待されましたが、nilが返されました")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("エラーメッセージが期待と異なります\n期待: %q を含む\n実際: %q",
						tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("予期しないエラー: %v", err)
				return
			}

			// デフォルト値のテスト
			if strings.Contains(tt.name, "デフォルト値") {
				if tt.config.Output.MarkdownDir != "output/markdown" {
					t.Errorf("MarkdownDirのデフォルト値が期待と異なります: %q", tt.config.Output.MarkdownDir)
				}
				if tt.config.Output.IntermediateDir != "output/intermediate" {
					t.Errorf("IntermediateDirのデフォルト値が期待と異なります: %q", tt.config.Output.IntermediateDir)
				}
			}
		})
	}
}
