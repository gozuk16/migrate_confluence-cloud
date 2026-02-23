# migrate_confluence-cloud

Confluence Cloud REST API v2 を使用してページデータを取得し、Storage Format（XHTML）から Markdown へ変換する Go 製 CLI ツールです。

出力は [Hugo Leaf Bundle](https://gohugo.io/content-management/page-bundles/) 形式（`SPACE_KEY/PAGE_TITLE/index.md`）に対応しています。

## 機能一覧

- Confluence Cloud REST API v2 によるページデータの取得
- Storage Format（XHTML）から Markdown への変換
- スペース単位での一括取得
- 子ページの再帰的取得
- 添付ファイルのダウンロード
- XHTML 中間ファイルの保存と再変換（APIアクセス不要）
- Hugo Leaf Bundle 形式での出力

### 変換対応する Confluence 要素

| Confluence 要素 | 変換後の Markdown |
|---|---|
| 見出し（h1〜h6） | `#`〜`######` |
| 段落、太字、斜体、取り消し線 | 標準 Markdown 書式 |
| 順序なし・順序ありリスト | `-` / `1.` |
| テーブル | GFM テーブル |
| コードブロックマクロ | コードフェンス（言語ハイライト対応） |
| Info / Note / Warning / Tip パネル | GFM Alerts（`> [!NOTE]` 等） |
| 展開マクロ（expand） | `<details>` / `<summary>` |
| タスクリスト | チェックボックスリスト |
| 画像（添付ファイル・外部URL） | `![]()` |
| 内部ページリンク | 相対パスリンク |
| ユーザーメンション | `@ユーザー名` |
| 絵文字（emoticon） | Unicode 絵文字 |

## 動作要件

- Go 1.25 以上
- Atlassian API トークン（[生成ページ](https://id.atlassian.com/manage-profile/security/api-tokens)）
- `golangci-lint`（`make lint` を使用する場合）

## セットアップ

### 1. リポジトリのクローン

```bash
git clone https://github.com/gozuk16/migrate_confluence-cloud.git
cd migrate_confluence-cloud
```

### 2. 依存パッケージのインストール

```bash
go mod download
```

### 3. 設定ファイルの作成

`config.toml.example` を `config.toml` にコピーし、各項目を設定します。

```bash
cp config.toml.example config.toml
```

### 4. ビルド

```bash
make build
```

ビルドが成功すると、カレントディレクトリに `migConfluence` バイナリが生成されます。

## 設定ファイル

`config.toml` に Confluence の接続情報と出力設定を記述します。

```toml
[confluence]
# Confluence Cloud の URL（例: https://your-domain.atlassian.net）
url = "https://your-domain.atlassian.net"
# Confluence ユーザーのメールアドレス
email = "your-email@example.com"
# Atlassian API トークン
api_token = "your-api-token"

[output]
# Markdown 出力ディレクトリ（デフォルト: output/markdown）
markdown_dir = "output/markdown"
# 添付ファイル保存ディレクトリ（空の場合は markdown_dir 内のページディレクトリに配置）
attachments_dir = ""
# XHTML 中間ファイル保存ディレクトリ（デフォルト: output/xhtml）
xhtml_dir = "output/xhtml"

[search]
# デフォルトのスペースキー（space コマンドでスペースキーを省略した場合に使用）
default_space_key = ""

[display]
# 変換時に無視する Confluence マクロ名のリスト
ignored_macros = []

# 削除済みユーザーのマッピング（accountId -> 表示名）
# [deletedUsers]
# "123456:abcdef" = "山田 太郎"
```

### 設定項目の説明

| セクション | キー | 説明 |
|---|---|---|
| `confluence` | `url` | Confluence Cloud の URL |
| `confluence` | `email` | ログインに使用するメールアドレス |
| `confluence` | `api_token` | Atlassian API トークン |
| `output` | `markdown_dir` | Markdown ファイルの出力先ディレクトリ |
| `output` | `attachments_dir` | 添付ファイルの保存先（空の場合は Markdown と同じディレクトリ） |
| `output` | `xhtml_dir` | XHTML 中間ファイルの保存先ディレクトリ |
| `search` | `default_space_key` | `space` コマンドのデフォルトスペースキー |
| `display` | `ignored_macros` | 変換時に無視するマクロ名のリスト |
| `deletedUsers` | `"accountId"` | 削除済みアカウントの表示名マッピング |

## 使い方

### 共通フラグ

| フラグ | エイリアス | デフォルト | 説明 |
|---|---|---|---|
| `--config` | `-c` | `config.toml` | 設定ファイルのパス |

---

### `page` コマンド — 単一ページの取得と変換

```bash
migConfluence page --page-id <ページID> [オプション]
```

| フラグ | エイリアス | 説明 |
|---|---|---|
| `--page-id` | `--id` | 取得するページのID（必須） |
| `--recursive` | `-r` | 子ページを再帰的に取得する |
| `--save-xhtml` | — | XHTML 中間ファイルを保存する |
| `--download-attachments` | — | 添付ファイルをダウンロードする |

**使用例**

```bash
# 単一ページを変換
./migConfluence page --page-id 123456789

# 子ページも含めて再帰的に取得
./migConfluence page --page-id 123456789 --recursive

# XHTML 中間ファイルも保存しながら変換
./migConfluence page --page-id 123456789 --save-xhtml

# 添付ファイルも一緒にダウンロード
./migConfluence page --page-id 123456789 --download-attachments

# 別の設定ファイルを使用
./migConfluence --config /path/to/config.toml page --page-id 123456789
```

---

### `space` コマンド — スペース内の全ページを一括取得

```bash
migConfluence space [--space-key <スペースキー>] [オプション]
```

| フラグ | エイリアス | 説明 |
|---|---|---|
| `--space-key` | `-k` | 取得するスペースのキー（省略時は設定ファイルの `default_space_key` を使用） |
| `--save-xhtml` | — | XHTML 中間ファイルを保存する |
| `--download-attachments` | — | 添付ファイルをダウンロードする |

**使用例**

```bash
# スペース内の全ページを変換
./migConfluence space --space-key MYSPACE

# 設定ファイルの default_space_key を使用
./migConfluence space

# XHTML 保存と添付ファイルダウンロードを同時に実行
./migConfluence space --space-key MYSPACE --save-xhtml --download-attachments
```

---

### `convert` コマンド — 保存済み XHTML からの再変換

APIアクセスなしに、保存済みの XHTML 中間ファイルから Markdown を再生成します。

```bash
migConfluence convert [--space-key <スペースキー>]
```

| フラグ | エイリアス | 説明 |
|---|---|---|
| `--space-key` | `-k` | 変換対象のスペースキー（省略時は全スペースを対象） |

**使用例**

```bash
# 全スペースの XHTML を再変換
./migConfluence convert

# 特定スペースのみ再変換
./migConfluence convert --space-key MYSPACE
```

---

### デバッグモード

環境変数 `LOG_LEVEL=DEBUG` を設定するとデバッグログが有効になり、`debug.log` ファイルに出力されます。

```bash
LOG_LEVEL=DEBUG ./migConfluence page --page-id 123456789
```

## 出力ファイル構成

### Markdown 出力（Hugo Leaf Bundle 形式）

```
output/markdown/
└── SPACE_KEY/
    └── PAGE_TITLE/
        ├── index.md        # 変換された Markdown ファイル
        └── image.png       # ダウンロードされた添付ファイル（任意）
```

### XHTML 中間ファイル（`--save-xhtml` 使用時）

```
output/xhtml/
└── SPACE_KEY/
    └── PAGE_TITLE/
        ├── content.xhtml       # Confluence Storage Format
        ├── metadata.toml       # ページメタデータ
        └── comments/           # フッターコメント（任意）
            ├── comment_001.xhtml
            └── comment_001.toml
```

## Makefile コマンド

| コマンド | 説明 |
|---|---|
| `make build` | バイナリをビルドして `migConfluence` を生成 |
| `make test` | テストを実行 |
| `make coverage` | テストカバレッジを計測し `coverage.html` を生成 |
| `make lint` | `golangci-lint` でリントを実行 |
| `make clean` | ビルド成果物・ログファイルを削除 |

## ライセンス

[MIT License](LICENSE)
