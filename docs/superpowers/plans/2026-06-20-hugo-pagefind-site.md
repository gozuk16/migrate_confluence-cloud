# Hugo + Pagefind 静的ドキュメントサイト 実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `migConfluence` が出力した Markdown を Hugo + Pagefind で社内向け静的ドキュメントサイトとして公開できる状態にする

**Architecture:** `hugo-site/` ディレクトリを同リポジトリ内に新規作成し、Hugo Book テーマで静的 HTML を生成する。`hugo build` 後に Pagefind でインデックスを生成することで全文検索を実現する。`migConfluence` 本体のコードは変更しない。

**Tech Stack:** Hugo（静的サイトジェネレーター）、Hugo Book テーマ（git submodule）、Pagefind（全文検索）

## Global Constraints

- `migConfluence` 本体（Go コード）は一切変更しない
- Hugo のバージョン: 0.120.0 以上（Extended 版不要）
- Hugo Book テーマは git submodule として `hugo-site/themes/hugo-book/` に配置する
- Pagefind は npx 経由で実行する（Node.js 18 以上が必要）。社内サーバーに Node.js がない場合は [Pagefind バイナリ](https://github.com/CloudCannon/pagefind/releases) を使用する
- コンテンツは `hugo-site/content/` に配置する（`migConfluence` の出力先 `output/markdown/` からコピー）
- `hugo-site/public/` と `hugo-site/resources/` は `.gitignore` に追加する

---

## ファイル一覧

| 操作 | パス | 内容 |
|---|---|---|
| 新規作成 | `hugo-site/hugo.toml` | Hugo 設定ファイル |
| 新規作成 | `hugo-site/themes/hugo-book/` | Hugo Book テーマ（git submodule） |
| 新規作成 | `hugo-site/content/.gitkeep` | content ディレクトリを git 管理するためのダミーファイル |
| 新規作成 | `hugo-site/layouts/partials/docs/inject/head.html` | Pagefind UI 埋め込みパーシャル |
| 変更 | `Makefile` | `sync-and-build`、`hugo-serve` ターゲット追加 |
| 変更 | `.gitignore` | `hugo-site/public/`、`hugo-site/resources/` を追加 |

---

## Task 1: Hugo プロジェクト初期化 + Hugo Book テーマ設定

**Files:**
- Create: `hugo-site/hugo.toml`
- Create: `hugo-site/themes/hugo-book/` (git submodule)
- Create: `hugo-site/content/.gitkeep`
- Modify: `.gitignore`

**Interfaces:**
- Produces: `hugo-site/` — Task 2 が Pagefind UI パーシャルを追加する土台

- [ ] **Step 1: Hugo のインストール確認**

```bash
hugo version
```

Expected: `hugo v0.120.0` 以上。インストールされていない場合:

```bash
# macOS
brew install hugo

# Linux (snap)
snap install hugo
```

- [ ] **Step 2: Hugo プロジェクトを初期化する**

```bash
hugo new site hugo-site
```

Expected: `Congratulations! Your new Hugo site was created in .../hugo-site.` と表示される。

- [ ] **Step 3: Hugo Book テーマを git submodule として追加する**

```bash
git submodule add https://github.com/alex-shpak/hugo-book hugo-site/themes/hugo-book
```

Expected: `hugo-site/themes/hugo-book/` にテーマファイルが展開される。

- [ ] **Step 4: hugo.toml を設定する**

`hugo-site/hugo.toml` を以下の内容に書き換える（`hugo new site` が生成したデフォルト内容を置き換える）:

```toml
baseURL = "/"
languageCode = "ja"
title = "社内ドキュメント"
theme = "hugo-book"

[params]
  # Hugo Book 組み込み検索を無効化（Pagefind を使うため）
  BookSearch = false
  # 右サイドバーに目次を表示
  BookToC = true
  # ディレクトリ構造をサイドバーナビに使用
  BookMenuBundle = false
  # 日付フォーマット
  BookDateFormat = "2006-01-02"

[markup.goldmark.renderer]
  # HTML タグ（<details> など）を Markdown 内で使えるようにする
  unsafe = true
```

- [ ] **Step 5: content ディレクトリを git 管理対象にする**

```bash
touch hugo-site/content/.gitkeep
```

- [ ] **Step 6: .gitignore にビルド成果物を追加する**

`.gitignore` の末尾に追記する（ファイルが存在しない場合は新規作成）:

```
# Hugo ビルド成果物
hugo-site/public/
hugo-site/resources/
```

- [ ] **Step 7: サンプルコンテンツで動作確認する**

テスト用ページを作成:

```bash
mkdir -p hugo-site/content/sample/test-page
cat > hugo-site/content/sample/test-page/index.md << 'EOF'
+++
title = "テストページ"
date = "2026-06-20T00:00:00Z"
+++

# テストページ

これは動作確認用のサンプルページです。

## セクション1

本文テキスト。

## セクション2

- リスト項目1
- リスト項目2
EOF
```

Hugo 開発サーバーを起動して確認:

```bash
cd hugo-site && hugo server --bind 0.0.0.0
```

ブラウザで `http://localhost:1313` を開き、以下を確認:
- 左サイドバーに「sample > test-page」が表示される
- ページ本文が正しく表示される

確認後、`Ctrl+C` でサーバーを停止。

- [ ] **Step 8: コミットする**

```bash
cd ..  # リポジトリルートに戻る
git add hugo-site/ .gitignore .gitmodules
git commit -m "feat: Hugo + Hugo Book テーマで静的サイトを初期化"
```

---

## Task 2: Pagefind 全文検索の組み込み

**Files:**
- Create: `hugo-site/layouts/partials/docs/inject/head.html`

**Interfaces:**
- Consumes: `hugo-site/hugo.toml`、`hugo-site/themes/hugo-book/` (Task 1 で作成)
- Produces: `hugo-site/layouts/partials/docs/inject/head.html` — Hugo Book のヘッダーに Pagefind UI を注入するパーシャル

- [ ] **Step 1: Pagefind のインストール確認**

```bash
npx pagefind --version
```

Expected: `pagefind 1.x.x` と表示される。Node.js がない場合は [Pagefind リリースページ](https://github.com/CloudCannon/pagefind/releases) からバイナリをダウンロードして PATH に追加する。

- [ ] **Step 2: Pagefind UI を注入するパーシャルを作成する**

Hugo Book テーマは `layouts/partials/docs/inject/head.html` を自動的に読み込む（テーマ本体は変更不要）。

```bash
mkdir -p hugo-site/layouts/partials/docs/inject
```

`hugo-site/layouts/partials/docs/inject/head.html` を以下の内容で作成する:

```html
<link href="/_pagefind/pagefind-ui.css" rel="stylesheet">
<script src="/_pagefind/pagefind-ui.js"></script>
<div id="pagefind-search" style="padding: 1rem 0;"></div>
<script>
  window.addEventListener('DOMContentLoaded', function () {
    new PagefindUI({
      element: "#pagefind-search",
      showSubResults: true,
      translations: {
        placeholder: "検索...",
        zero_results: "[SEARCH_TERM] に一致するページが見つかりませんでした"
      }
    });
  });
</script>
```

- [ ] **Step 3: Hugo でビルドしてから Pagefind インデックスを生成する**

```bash
cd hugo-site
hugo --minify
npx pagefind --site public
```

Expected:
- `hugo --minify` が `public/` を生成する
- `npx pagefind` が `Running Pagefind v1.x.x` を表示し `public/_pagefind/` を生成する

- [ ] **Step 4: 検索 UI の動作確認**

```bash
python3 -m http.server 8080 --directory public
```

ブラウザで `http://localhost:8080` を開き、以下を確認:
- ページ上部に検索ボックスが表示される
- 「テスト」と入力すると Task 1 で作成したサンプルページが候補に出る
- 候補をクリックするとページに遷移する

確認後、`Ctrl+C` でサーバーを停止。

- [ ] **Step 5: コミットする**

```bash
cd ..  # リポジトリルートに戻る
git add hugo-site/layouts/
git commit -m "feat: Pagefind 全文検索 UI を Hugo Book テーマに組み込む"
```

---

## Task 3: Makefile ターゲット追加

**Files:**
- Modify: `Makefile`

**Interfaces:**
- Consumes: `hugo-site/` (Task 1)、`hugo-site/layouts/` (Task 2)
- Produces: `make sync-and-build`、`make hugo-serve` — Confluence 同期 → サイト公開をワンコマンド化

- [ ] **Step 1: Makefile に Hugo 関連ターゲットを追加する**

`Makefile` の `.PHONY` 行と末尾を以下のように変更する。

`.PHONY` 行を以下に変更:

```makefile
.PHONY: build test coverage lint clean sync-and-build hugo-serve
```

ファイル末尾に以下を追記:

```makefile
# Hugo サイト: Confluence データ取得 → ビルド → Pagefind インデックス生成
sync-and-build: build
	./migConfluence space
	mkdir -p hugo-site/content
	cp -r output/markdown/. hugo-site/content/
	cd hugo-site && hugo --minify
	npx pagefind --site hugo-site/public
	@echo "完了: hugo-site/public/ を社内サーバーに配置してください"

# Hugo 開発サーバー（ローカル確認用、Pagefind なし）
hugo-serve:
	cd hugo-site && hugo server --bind 0.0.0.0
```

- [ ] **Step 2: `make hugo-serve` の動作確認**

```bash
make hugo-serve
```

Expected: `Web Server is available at http://localhost:1313/` と表示される。ブラウザで確認後 `Ctrl+C` で停止。

- [ ] **Step 3: コミットする**

```bash
git add Makefile
git commit -m "feat: Makefile に sync-and-build・hugo-serve ターゲットを追加"
```

---

## セルフレビュー（実装者向けチェックリスト）

実装完了後、以下を確認する:

- [ ] `make hugo-serve` でサイドバーナビが表示される
- [ ] `cd hugo-site && hugo --minify && npx pagefind --site public` 後に `public/_pagefind/` が生成される
- [ ] `python3 -m http.server 8080 --directory hugo-site/public` で検索ボックスが表示され、検索が動作する
- [ ] `migConfluence` の実際の出力（`output/markdown/`）を `hugo-site/content/` にコピーしてビルドしたとき、Confluence のページ階層がサイドバーに反映される
- [ ] `hugo-site/public/` が `.gitignore` に含まれており `git status` で追跡されていない
