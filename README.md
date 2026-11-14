# migrate_confluence-cloud

Confluence Cloud REST API v2 を使用してページデータを取得し、Storage Format（XHTML）から Markdown へ変換する Go 製 CLI ツールです。

出力は [Hugo Leaf Bundle](https://gohugo.io/content-management/page-bundles/) 形式（`SPACE_KEY/PAGE_TITLE/index.md`）に対応しています。

## 機能一覧

- Confluence Cloud REST API v2 によるページデータの取得
- Storage Format（XHTML）から Markdown への変換
- スペース単位での一括取得
- 子ページの再帰的取得
- 添付ファイルのダウンロード
- 中間ファイルの保存と再変換（APIアクセス不要）
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
| ページプロパティ（details） | 本文の表 + front matter の `[[properties]]` |
| ページプロパティレポート（detailssummary） | `{{< page-properties-report >}}` ショートコード（条件はCQLから変換） |
| 子ページ一覧（children） | `{{< children >}}` ショートコード（入れ子の箇条書き） |
| ロードマッププランナー（roadmap） | SVG の図（`confluence-roadmap-N.svg` を変換時に生成し、`{{< roadmap >}}` ショートコードで差し込む） |
| draw.io の図・ボード（draw.io アプリ） | SVG の図（図ファイルから draw.io デスクトップ版で `confluence-drawio-N.svg` を作り、`{{< drawio >}}` ショートコードで本文に差し込む。「拡大表示」ボタンで draw.io ビューアを開く）。SVG にできないときは添付のプレビュー画像 `<図の名前>.png` |

### draw.io の図

draw.io アプリの図とボードは、移行時に図ファイル（添付の `.drawio`）から SVG を作り、ページに直接埋め込みます。図の中の文字は選択できます。Confluence 側の表示設定（シンプルビューア、ライトボックスなど）は再現せず、どの図も同じ表示になります。

- **必要なもの**: `--download-attachments`（図ファイルを保存する）と、draw.io デスクトップ版（SVG を作る）。どちらかが無い図は、変換ログに警告を出して、添付のプレビュー画像（PNG）の表示に戻します
- **拡大表示**: 図の下の「拡大表示」ボタンで、テーマに同梱した draw.io ビューアがページの上に重ねて図を表示します。図のデータはブラウザの中で処理し、外部には送りません。ビューアは数式用のスクリプトと特殊な図形ライブラリの定義を、必要に応じて `viewer.diagrams.net` から読み込みます
- **元の図ファイル**: ページ末尾の添付ファイル欄のリンクからダウンロードできます
- **前提**: SVG はページに直接埋め込むため、図の内容は信頼できるもの（自社の Confluence の図）であることを前提にしています。`<script>` や `javascript:` のリンクを含む SVG は埋め込まず PNG の表示に戻しますが、それ以外の検査はしていません
- **制限**: 複数ページの図は 1 ページ目だけを埋め込みます（拡大表示ではページを切り替えられます）。図は添付ファイルの最新版から作ります（マクロが指す過去の版は再現しません）。画面の無い Linux で draw.io デスクトップ版を動かすには `xvfb` などが必要です

## 動作要件

- Go 1.25 以上
- Atlassian API トークン（[生成ページ](https://id.atlassian.com/manage-profile/security/api-tokens)）
- `golangci-lint`（`make lint` を使用する場合）
- `hugo`（Hugo CLI）0.146.0 以上
- `Node.js` 18 以上（`npx pagefind` を使用する場合）
- [draw.io デスクトップ版](https://github.com/jgraph/drawio-desktop/releases)（draw.io の図を SVG で埋め込む場合。無ければ図は PNG で表示する）

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
# 中間ファイル保存ディレクトリ（デフォルト: output/intermediate）
intermediate_dir = "output/intermediate"

[search]
# デフォルトのスペースキー（space コマンドでスペースキーを省略した場合に使用）
default_space_key = ""

[display]
# 変換時に無視する Confluence マクロ名のリスト
ignored_macros = []

[drawio]
# draw.io デスクトップ版のコマンドのパス（空の場合は PATH と既定のインストール先から探す）
cli_path = ""

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
| `output` | `intermediate_dir` | 中間ファイルの保存先ディレクトリ |
| `search` | `default_space_key` | `space` コマンドのデフォルトスペースキー |
| `display` | `ignored_macros` | 変換時に無視するマクロ名のリスト |
| `drawio` | `cli_path` | draw.io デスクトップ版のコマンドのパス（空の場合は PATH の `drawio`・`draw.io` と既定のインストール先から探す） |
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
| `--save-intermediate` | — | 中間ファイルを保存する |
| `--download-attachments` | — | 添付ファイルをダウンロードする |

**使用例**

```bash
# 単一ページを変換
./migConfluence page --page-id 123456789

# 子ページも含めて再帰的に取得
./migConfluence page --page-id 123456789 --recursive

# 中間ファイルも保存しながら変換
./migConfluence page --page-id 123456789 --save-intermediate

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
| `--save-intermediate` | — | 中間ファイルを保存する |
| `--download-attachments` | — | 添付ファイルをダウンロードする |

**使用例**

```bash
# スペース内の全ページを変換
./migConfluence space --space-key MYSPACE

# 設定ファイルの default_space_key を使用
./migConfluence space

# 中間ファイル保存と添付ファイルダウンロードを同時に実行
./migConfluence space --space-key MYSPACE --save-intermediate --download-attachments
```

---

### `convert` コマンド — 保存済み中間ファイルからの再変換

APIアクセスなしに、保存済みの中間ファイルから Markdown を再生成します。

```bash
migConfluence convert [--space-key <スペースキー>]
```

| フラグ | エイリアス | 説明 |
|---|---|---|
| `--space-key` | `-k` | 変換対象のスペースキー（省略時は全スペースを対象） |

**使用例**

```bash
# 全スペースの中間ファイルを再変換
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

### 中間ファイル（`--save-intermediate` 使用時）

```
output/intermediate/
└── SPACE_KEY/
    └── PAGE_TITLE/
        ├── content.json        # ADF JSON 形式の中間ファイル
        ├── metadata.toml       # ページメタデータ
        └── comments/           # フッターコメント（任意）
            ├── comment_001.json
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
| `make sync-and-build` | Confluence 同期 → Hugo ビルド → Pagefind インデックス生成 |
| `make hugo-serve` | Hugo 開発サーバーを起動（ローカル確認用） |

## ライセンス

[MIT License](LICENSE)
