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
- **拡大表示**: 図の下の「拡大表示」ボタンで、テーマに同梱した draw.io ビューアがページの上に重ねて図を表示します。図のデータはブラウザの中で処理し、外部には送りません。ビューアは数式用のスクリプトと特殊な図形ライブラリの定義を `viewer.diagrams.net` から、図形ライブラリのアイコン（Azure・AWS など）を `app.diagrams.net` から、必要に応じて読み込みます。外部につながらない環境では、拡大表示の中でそれらが表示されないことがあります
- **元の図ファイル**: ページ末尾の添付ファイル欄のリンクからダウンロードできます
- **図の中の画像**: 図形ライブラリのアイコンや外部 URL の画像は、移行時に取得して SVG の中に埋め込みます。このとき、移行する端末から図の中の画像の URL へ接続し、取得した画像はサイトに載ります。取得できなかった画像は URL のまま残ります。接続しても応答の返らない URL の画像があると、draw.io が読み込みを待ち続けて変換が時間切れ（60 秒）になり、その図は PNG の表示に戻ります
- **前提**: SVG はページに直接埋め込むため、図の内容は信頼できるもの（自社の Confluence の図）であることを前提にしています。`<script>` や `javascript:` のリンクを含む SVG は埋め込まず PNG の表示に戻しますが、それ以外の検査はしていません
- **複数ページの図**: マクロが表示するページを指定していれば（マクロのパラメータ `aspect` の先頭にあるページの ID）、そのページを埋め込み、拡大表示もそのページを開きます。指定が無ければ 1 ページ目を埋め込みます。指定されたページを図ファイルから見つけられないとき（そのページが無い、図ファイルを XML として読めない）は、変換ログに警告を出して 1 ページ目を埋め込みます。どの場合も、拡大表示ではページを切り替えられます。`aspect` の形式は実際のマクロで確かめました（先頭が図ファイルの `<diagram id>` と一致）。ただし、その図は 1 ページだったので、2 ページ目以降を指定した場合の動作はテスト用の図で確かめたものです
- **制限**: マクロのレイヤーの指定（`aspect` の 2 つ目以降）は再現しません。どのレイヤーも、図ファイルに保存された表示・非表示のまま描きます。図は添付ファイルの最新版から作ります（マクロが指す過去の版は再現しません）。画面の無い Linux で draw.io デスクトップ版を動かすには `xvfb` などが必要です

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
# space コマンドで移行するスペースキーのリスト。指定した順番に処理する
# ユーザースペース（個人用スペース）のキーは「~」で始まる
space_keys = ["MYSPACE", "~5570586eed56ba9b9b4a87ad7418b7086f1063"]
# デフォルトのスペースキー（space_keys が空のときに使う。従来の設定との互換のため残している）
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
| `search` | `space_keys` | `space` コマンドで移行するスペースキーのリスト（指定した順番に処理する） |
| `search` | `default_space_key` | `space_keys` が空のときに移行するスペースキー（従来の設定との互換のため残している） |
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
| `--download-attachments` | — | 添付ファイルをダウンロードする。保存済みのファイルは、Confluence 側で更新されたものだけ取り直す |

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
migConfluence space [--space-key <スペースキー>]... [オプション]
```

| フラグ | エイリアス | 説明 |
|---|---|---|
| `--space-key` | `-k` | 取得するスペースのキー。複数回指定するか、カンマで区切ると、複数のスペースを指定した順番に処理する |
| `--save-intermediate` | — | 中間ファイルを保存する |
| `--download-attachments` | — | 添付ファイルをダウンロードする。保存済みのファイルは、Confluence 側で更新されたものだけ取り直す |

**移行するスペースの決め方**

移行するスペースは、次の順番で最初に見つかった指定を使います。上位の指定があれば、下位の指定は使いません（合わせることはしません）。

1. 引数の `--space-key`
2. 設定ファイルの `space_keys`
3. 設定ファイルの `default_space_key`

`default_space_key` だけを書いた従来の設定ファイルは、そのまま使えます。`space_keys` と `default_space_key` の両方があるときは `space_keys` を使い、`default_space_key` は使いません。スペースキーの前後の空白は取り除き、同じキーを 2 回以上指定しても 1 回だけ処理します。

**スペースの移行に失敗したとき**

あるスペースの取得に失敗しても（スペースが無い、権限が無いなど）、残りのスペースの移行を続けます。すべてのスペースを処理したあとで、失敗したスペースとその理由をまとめて表示し、コマンドは終了コード 1 で終わります。成功したスペースの出力は残ります。ページ単位の失敗は、これまでどおり警告を出して次のページに進みます（スペースの失敗には数えません）。

**スペースのセクションのタイトル**

スペースごとに `SPACE_KEY/_index.md` を出力し、`title` にスペース名を書きます。Hugo のトップページとサイドバーには、スペースキーではなくスペース名が表示されます。`_index.md` は移行のたびに上書きするので、Confluence でスペース名を変えると次の移行で反映されます。`convert` コマンドはスペース名を持たないため、`_index.md` を書き出しません。

**移行対象から外したスペースの出力**

`space_keys` や `--space-key` からスペースを外しても、そのスペースの出力（`markdown_dir` の下のスペースキーのディレクトリ）は消えずに残り、Hugo のサイトにも載り続けます。サイトから外すには、そのディレクトリを手動で削除してください。サンプル以外のスペースのディレクトリは git の管理対象から外しているので、`git status` には現れません。

**使用例**

```bash
# スペース内の全ページを変換
./migConfluence space --space-key MYSPACE

# 複数のスペースを変換（次の 2 つは同じ）
./migConfluence space --space-key MYSPACE --space-key '~5570586eed56ba9b9b4a87ad7418b7086f1063'
./migConfluence space --space-key 'MYSPACE,~5570586eed56ba9b9b4a87ad7418b7086f1063'

# 設定ファイルの space_keys（無ければ default_space_key）を使用
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
└── SPACE_KEY/              # ユーザースペースは ~<アカウント ID> のまま（URL も /~<アカウント ID>/）
    ├── _index.md           # スペースのセクション（title = スペース名）
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
| `make sync-and-build` | Confluence 同期 → Hugo ビルド → Pagefind インデックス生成。移行するスペースは設定ファイルの `space_keys`（無ければ `default_space_key`）で決まる。`make sync-and-build SPACE_KEYS=A,B` で引数として渡すこともできる。失敗したスペースがあると、Hugo のビルドに進まずに止まる |
| `make hugo-serve` | Hugo 開発サーバーを起動（ローカル確認用） |

## ライセンス

[MIT License](LICENSE)
