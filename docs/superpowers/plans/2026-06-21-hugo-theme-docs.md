# hugo-theme-docs 最小実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `hugo-theme-docs` カスタムテーマを最小限のファイルで実装し、Confluence 移行ページを Hugo で表示・Pagefind 検索できる状態にする

**Architecture:** `hugo-theme-issues`（Jira 移行ツール用テーマ）の構造を踏襲し、Jira 固有の部分（issue_key/rank/アイコン）を除去して Confluence ドキュメント向けに書き換える。hugo-book submodule を削除して hugo-theme-docs に切り替え、Pagefind はテーマ内の head.html に組み込む。

**Tech Stack:** Hugo 0.146.0+、hugo-theme-docs（git submodule）、Pagefind（npx）、CSS Grid

## Global Constraints

- `migConfluence` 本体（Go コード）は一切変更しない
- Hugo バージョン: 0.146.0 以上（Extended 版不要）
- `hugo-theme-docs` は git submodule として `hugo-site/themes/hugo-theme-docs/` に既に追加済み
- Pagefind は `npx pagefind --site public` で実行する（Node.js 18 以上が必要）
- CSS はゼロから書く（hugo-theme-issues の main.css は流用しない）
- 左サイドバーはフラットリスト（`parent` による階層表示はしない）
- ホームページはスペース一覧（`site.Sections` を表示）

---

## ファイル一覧

| 操作 | パス | 内容 |
|---|---|---|
| 削除 | `hugo-site/themes/hugo-book/` | git submodule を削除 |
| 削除 | `hugo-site/layouts/` | 旧 Pagefind inject パーシャルを削除 |
| 変更 | `hugo-site/hugo.toml` | theme 変更・Hugo Book パラメータ削除 |
| 新規作成 | `hugo-site/themes/hugo-theme-docs/hugo.toml` | テーマ定義 |
| 新規作成 | `hugo-site/themes/hugo-theme-docs/layouts/baseof.html` | HTML 骨格 |
| 新規作成 | `hugo-site/themes/hugo-theme-docs/layouts/home.html` | トップページ（スペース一覧） |
| 新規作成 | `hugo-site/themes/hugo-theme-docs/layouts/section.html` | スペースページ（ページ一覧） |
| 新規作成 | `hugo-site/themes/hugo-theme-docs/layouts/page.html` | 個別ページ |
| 新規作成 | `hugo-site/themes/hugo-theme-docs/layouts/_partials/head.html` | `<head>` 内容（CSS/Pagefind） |
| 新規作成 | `hugo-site/themes/hugo-theme-docs/layouts/_partials/header.html` | タイトル + 検索ボックス |
| 新規作成 | `hugo-site/themes/hugo-theme-docs/layouts/_partials/sidebar-left.html` | スペース内ページ一覧 |
| 新規作成 | `hugo-site/themes/hugo-theme-docs/layouts/_partials/footer.html` | フッター |
| 新規作成 | `hugo-site/themes/hugo-theme-docs/layouts/_markup/render-link.html` | リンクレンダリング |
| 新規作成 | `hugo-site/themes/hugo-theme-docs/assets/css/main.css` | 最小限 CSS |

---

## Task 1: hugo-book 削除 + hugo-theme-docs 全テーマファイル作成

**Files:**
- Delete: `hugo-site/themes/hugo-book/` (git submodule)
- Delete: `hugo-site/layouts/`
- Modify: `hugo-site/hugo.toml`
- Create: `hugo-site/themes/hugo-theme-docs/hugo.toml`
- Create: `hugo-site/themes/hugo-theme-docs/layouts/baseof.html`
- Create: `hugo-site/themes/hugo-theme-docs/layouts/home.html`
- Create: `hugo-site/themes/hugo-theme-docs/layouts/section.html`
- Create: `hugo-site/themes/hugo-theme-docs/layouts/page.html`
- Create: `hugo-site/themes/hugo-theme-docs/layouts/_partials/head.html`
- Create: `hugo-site/themes/hugo-theme-docs/layouts/_partials/header.html`
- Create: `hugo-site/themes/hugo-theme-docs/layouts/_partials/sidebar-left.html`
- Create: `hugo-site/themes/hugo-theme-docs/layouts/_partials/footer.html`
- Create: `hugo-site/themes/hugo-theme-docs/layouts/_markup/render-link.html`
- Create: `hugo-site/themes/hugo-theme-docs/assets/css/main.css`

**Interfaces:**
- Produces: `hugo build` が通り `hugo server` でページが表示できる状態

- [ ] **Step 1: hugo-book submodule を削除する**

リポジトリルート（`/Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud`）で実行:

```bash
git submodule deinit -f hugo-site/themes/hugo-book
git rm -f hugo-site/themes/hugo-book
rm -rf .git/modules/hugo-site/themes/hugo-book
```

Expected:
- `hugo-site/themes/hugo-book/` ディレクトリが消える
- `.gitmodules` から `[submodule "hugo-site/themes/hugo-book"]` ブロックが削除される

- [ ] **Step 2: 旧 Pagefind inject パーシャルを削除する**

```bash
git rm -r hugo-site/layouts/
```

Expected: `hugo-site/layouts/partials/docs/inject/head.html` が削除される

- [ ] **Step 3: hugo-site/hugo.toml を更新する**

`hugo-site/hugo.toml` を以下の内容に完全に書き換える:

```toml
baseURL = "/"
languageCode = "ja"
title = "社内ドキュメント"
theme = "hugo-theme-docs"

[markup.goldmark.renderer]
  unsafe = true
```

- [ ] **Step 4: hugo-theme-docs/hugo.toml を作成する**

`hugo-site/themes/hugo-theme-docs/hugo.toml` を以下の内容で作成する:

```toml
[module.hugoVersion]
  extended = false
  min = "0.146.0"
```

- [ ] **Step 5: baseof.html を作成する**

`hugo-site/themes/hugo-theme-docs/layouts/baseof.html`:

```html
<!DOCTYPE html>
<html lang="{{ site.Language.LanguageCode }}">
<head>
  <title>{{ if .IsHome }}{{ site.Title }}{{ else }}{{ .Title }} - {{ site.Title }}{{ end }}</title>
  {{ partialCached "head.html" . "global" }}
</head>
<body class="{{ .Kind }}"{{ if eq .Kind "page" }} data-pagefind-body{{ end }}>
  <header>
    {{ partialCached "header.html" . "global" }}
  </header>
  {{- if .IsHome }}
  <main>
    {{ block "main" . }}{{ end }}
  </main>
  {{- else }}
  <aside class="sidebar-left">
    {{ block "sidebar-left" . }}{{ end }}
  </aside>
  <main>
    {{ block "main" . }}{{ end }}
  </main>
  {{- end }}
  <footer>
    {{ partialCached "footer.html" . "global" }}
  </footer>
</body>
</html>
```

- [ ] **Step 6: head.html を作成する**

`hugo-site/themes/hugo-theme-docs/layouts/_partials/head.html`:

```html
<meta charset="utf-8">
<meta name="viewport" content="width=device-width">
{{- with resources.Get "css/main.css" }}
  {{- if hugo.IsDevelopment }}
    <link rel="stylesheet" href="{{ .RelPermalink }}">
  {{- else }}
    {{- with . | minify | fingerprint }}
      <link rel="stylesheet" href="{{ .RelPermalink }}" integrity="{{ .Data.Integrity }}" crossorigin="anonymous">
    {{- end }}
  {{- end }}
{{- end }}
<link href="{{ "pagefind/pagefind-ui.css" | relURL }}" rel="stylesheet">
<script src="{{ "pagefind/pagefind-ui.js" | relURL }}"></script>
```

- [ ] **Step 7: header.html を作成する**

`hugo-site/themes/hugo-theme-docs/layouts/_partials/header.html`:

```html
<h1><a href="{{ site.BaseURL }}">{{ site.Title }}</a></h1>
<div id="search" class="header-search"></div>
<script>
  window.addEventListener('DOMContentLoaded', function () {
    new PagefindUI({
      element: "#search",
      showSubResults: true,
      translations: { placeholder: "検索..." }
    });
  });
</script>
```

- [ ] **Step 8: sidebar-left.html を作成する**

`hugo-site/themes/hugo-theme-docs/layouts/_partials/sidebar-left.html`:

```html
{{- $section := .CurrentSection }}
{{- with $section }}
  <nav aria-label="ページ一覧">
    <h2><a href="{{ .RelPermalink }}">{{ .Title }}</a></h2>
    <ul>
      {{- range sort .RegularPages "Title" "asc" }}
        <li><a href="{{ .RelPermalink }}">{{ .Title }}</a></li>
      {{- end }}
    </ul>
  </nav>
{{- end }}
```

- [ ] **Step 9: footer.html を作成する**

`hugo-site/themes/hugo-theme-docs/layouts/_partials/footer.html`:

```html
<p>{{ site.Title }}</p>
```

- [ ] **Step 10: home.html を作成する**

`hugo-site/themes/hugo-theme-docs/layouts/home.html`:

```html
{{ define "main" }}
  <h1>{{ site.Title }}</h1>
  <ul>
    {{- range sort site.Sections "Title" "asc" }}
      <li>
        <a href="{{ .RelPermalink }}">
          {{- with .Params.space_title }}{{ . }}{{- else }}{{ .Title }}{{- end }}
        </a>
      </li>
    {{- end }}
  </ul>
{{ end }}
```

- [ ] **Step 11: section.html を作成する**

`hugo-site/themes/hugo-theme-docs/layouts/section.html`:

```html
{{ define "sidebar-left" }}
  {{ partialCached "sidebar-left.html" . .CurrentSection.RelPermalink }}
{{ end }}

{{ define "main" }}
  <h1>{{ .Title }}</h1>
  <ul>
    {{- range sort .RegularPages "Title" "asc" }}
      <li>
        <a href="{{ .RelPermalink }}">{{ .Title }}</a>
        {{- with .Date }} <span class="date">{{ .Format "2006-01-02" }}</span>{{- end }}
      </li>
    {{- end }}
  </ul>
{{ end }}
```

- [ ] **Step 12: page.html を作成する**

`hugo-site/themes/hugo-theme-docs/layouts/page.html`:

```html
{{ define "sidebar-left" }}
  {{ partialCached "sidebar-left.html" . .CurrentSection.RelPermalink }}
{{ end }}

{{ define "main" }}
  <article>
    <h1>{{ .Title }}</h1>
    {{ .Content }}
    {{- if or .Params.confluence_url .Params.labels }}
      <footer class="page-meta">
        {{- with .Params.confluence_url }}
          <p><a href="{{ . }}" target="_blank" rel="noopener">Confluence で開く</a></p>
        {{- end }}
        {{- with .Params.labels }}
          <p>ラベル:
            {{- range . }} <span class="label">{{ . }}</span>{{- end }}
          </p>
        {{- end }}
      </footer>
    {{- end }}
  </article>
{{ end }}
```

- [ ] **Step 13: render-link.html を作成する**

`hugo-site/themes/hugo-theme-docs/layouts/_markup/render-link.html`:

```html
<a href="{{ .Destination | safeURL }}"{{ with .Title }} title="{{ . }}"{{ end }}>{{ .Text | safeHTML }}</a>
```

- [ ] **Step 14: main.css を作成する**

`hugo-site/themes/hugo-theme-docs/assets/css/main.css`:

```css
/* レイアウト */
body {
  display: grid;
  grid-template:
    "header" auto
    "sidebar main" 1fr
    "footer" auto
    / 240px 1fr;
  min-height: 100vh;
  margin: 0;
  font-family: sans-serif;
}
header {
  grid-area: header;
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: .5rem 1rem;
  border-bottom: 1px solid #ddd;
}
header h1 { margin: 0; font-size: 1.1rem; }
header h1 a { text-decoration: none; color: inherit; }
.sidebar-left {
  grid-area: sidebar;
  overflow-y: auto;
  padding: 1rem;
  border-right: 1px solid #ddd;
}
main { grid-area: main; padding: 1rem 2rem; max-width: 900px; }
footer {
  grid-area: footer;
  padding: .5rem 1rem;
  border-top: 1px solid #ddd;
  font-size: .8rem;
  color: #666;
}

/* ホームはサイドバーなし */
body.home {
  grid-template:
    "header" auto
    "main" 1fr
    "footer" auto
    / 1fr;
}

/* 検索ボックス */
.header-search { flex: 1; max-width: 400px; }

/* サイドバーナビ */
.sidebar-left nav h2 { font-size: .95rem; margin: 0 0 .5rem; }
.sidebar-left nav h2 a { text-decoration: none; color: inherit; }
.sidebar-left nav ul { list-style: none; padding: 0; margin: 0; }
.sidebar-left nav ul li a {
  display: block;
  padding: .25rem .5rem;
  text-decoration: none;
  border-radius: 3px;
  font-size: .9rem;
  color: #333;
}
.sidebar-left nav ul li a:hover { background: #f0f0f0; }

/* 日付 */
.date { font-size: .8rem; color: #888; margin-left: .5rem; }

/* 本文 */
article h1 { margin-top: 0; }
table { border-collapse: collapse; width: 100%; margin: 1rem 0; }
th, td { border: 1px solid #ccc; padding: .4rem .6rem; text-align: left; }
th { background: #f0f0f0; }
pre { background: #f5f5f5; padding: .8rem; overflow-x: auto; border-radius: 4px; margin: 1rem 0; }
code { background: #f5f5f5; padding: .1rem .3rem; border-radius: 2px; font-size: .9em; }
pre code { background: none; padding: 0; }
img { max-width: 100%; height: auto; }
blockquote { border-left: 4px solid #ddd; margin: 1rem 0; padding: .5rem 1rem; color: #555; }

/* ページメタ */
.page-meta {
  margin-top: 2rem;
  padding-top: 1rem;
  border-top: 1px solid #ddd;
  font-size: .85rem;
}
.label {
  background: #e8e8e8;
  padding: .1rem .4rem;
  border-radius: 3px;
  margin-right: .3rem;
  font-size: .8rem;
}
```

- [ ] **Step 15: `hugo build` が通ることを確認する**

```bash
cd hugo-site && hugo
```

Expected: `Built in Xms` のメッセージが出て、エラーがないこと。警告（WARN）があれば内容を確認する。

- [ ] **Step 16: `hugo server` でページが表示されることを確認する**

```bash
cd hugo-site && hugo server --bind 0.0.0.0
```

ブラウザで `http://localhost:1313` を開き、以下を確認:
- トップページにスペース一覧（`sample` セクション）が表示される
- `sample/test-page/` のページが表示される
- 左サイドバーにページ一覧が表示される

確認後 `Ctrl+C` で停止。

- [ ] **Step 17: コミットする**

```bash
cd ..  # リポジトリルートに戻る
git add hugo-site/themes/hugo-theme-docs/ hugo-site/hugo.toml .gitmodules
git commit -m "feat: hugo-theme-docs カスタムテーマを最小実装・hugo-book を削除"
```

---

## Task 2: Pagefind 動作確認

**Files:**
- 変更なし（動作確認のみ。問題があれば該当ファイルを修正してコミット）

**Interfaces:**
- Consumes: Task 1 で作成したすべてのテーマファイル

- [ ] **Step 1: Hugo でビルドする**

```bash
cd hugo-site && hugo --minify
```

Expected: `Built in Xms` のメッセージが出て、`public/` が生成される。エラーがないこと。

- [ ] **Step 2: Pagefind インデックスを生成する**

```bash
npx pagefind --site public
```

Expected: `Running Pagefind v1.x.x` と表示され、`public/pagefind/` が生成される。

- [ ] **Step 3: 検索の動作を確認する**

```bash
python3 -m http.server 8080 --directory public
```

ブラウザで `http://localhost:8080` を開き、以下を確認:
- ヘッダーに検索ボックスが表示される
- 「テスト」と入力すると `hugo-site/content/sample/test-page/index.md` のページが候補に出る
- 候補をクリックするとページに遷移する

確認後 `Ctrl+C` で停止。

- [ ] **Step 4: セルフレビューチェックリスト**

```bash
cd hugo-site && hugo
```

以下を確認:
- [ ] `hugo-site/public/pagefind/` が生成されている（`ls public/pagefind/`）
- [ ] `hugo-site/.hugo_build.lock` が `.gitignore` に含まれていることを確認（`grep hugo-site/.hugo_build.lock ../.gitignore`）
- [ ] `hugo-site/themes/hugo-book/` が存在しないことを確認（`ls hugo-site/themes/`）

- [ ] **Step 5: 問題がなければコミットは不要。修正した場合はコミットする**

修正ファイルがある場合:
```bash
git add <修正したファイル>
git commit -m "fix: Pagefind 動作確認で発見した問題を修正"
```
