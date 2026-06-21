---
title: hugo-theme-docs 最小実装設計
date: 2026-06-21
status: approved
---

# hugo-theme-docs 最小実装設計

## 概要

`migConfluence` が出力した Confluence ページ（Hugo Leaf Bundle形式）を表示する専用 Hugo テーマ `hugo-theme-docs` を作成する。`hugo-theme-issues`（Jira 移行ツール用テーマ）の構造を参考に、Confluence ドキュメント向けに必要最低限の機能で実装する。

## 参考テーマ

`/Users/gozu/go/src/github.com/gozuk16/migrate_jira-cloud/hugo-jira/themes/hugo-theme-issues`

- `baseof.html` のレイアウト構造（header / sidebar-left / main / footer）を踏襲する
- パーシャルの分割方法を踏襲する
- Pagefind の組み込み方（`head.html` 内に CSS/JS を直接記述）を踏襲する
- Jira 固有の部分（issue_key、issue_type アイコン、rank ソート等）は含めない

## コンテンツのデータモデル

`migConfluence` が出力する Front Matter（TOML形式）:

```toml
+++
title = "ページタイトル"
date = "2024-01-01T00:00:00Z"
lastmod = "2024-01-01T00:00:00Z"
space = "MYSPACE"
space_title = "スペース名"
page_id = "123456789"
parent = "親ページタイトル"   # 任意
labels = ["label1", "label2"] # 任意
confluence_url = "/wiki/spaces/MYSPACE/pages/..."  # 任意
+++
```

ディレクトリ構造: `content/SPACE_KEY/PAGE_TITLE/index.md`

## ファイル構成

```
hugo-site/themes/hugo-theme-docs/
├── hugo.toml                          ← テーマ最低バージョン要件
├── layouts/
│   ├── baseof.html                    ← HTML骨格（header/sidebar-left/main/footer）
│   ├── home.html                      ← トップページ（スペース一覧）
│   ├── page.html                      ← 個別ページ
│   ├── section.html                   ← スペースページ（ページ一覧）
│   ├── _partials/
│   │   ├── head.html                  ← <head>内容（CSS/JS/Pagefind）
│   │   ├── header.html                ← サイトタイトル + 検索ボックス
│   │   ├── sidebar-left.html          ← スペース内ページのフラットリスト
│   │   └── footer.html                ← フッター
│   └── _markup/
│       └── render-link.html           ← リンクレンダリング
└── assets/
    └── css/
        └── main.css                   ← 最小限 CSS（Grid レイアウト）
```

## レイアウト詳細

### baseof.html

`hugo-theme-issues` の `baseof.html` を Confluence 向けに簡略化する。サイドバーリサイズ機能・Cookie保存は不要なため削除する。

```html
<!DOCTYPE html>
<html lang="{{ site.Language.LanguageCode }}">
<head>
  {{ partialCached "head.html" . "global" }}
</head>
<body class="{{ .Kind }}"{{ if eq .Kind "page" }} data-pagefind-body{{ end }}>
  <header>{{ partialCached "header.html" . "global" }}</header>
  {{- if .IsHome }}
    <main>{{ block "main" . }}{{ end }}</main>
  {{- else }}
    <aside class="sidebar-left">{{ block "sidebar-left" . }}{{ end }}</aside>
    <main>{{ block "main" . }}{{ end }}</main>
  {{- end }}
  <footer>{{ partialCached "footer.html" . "global" }}</footer>
</body>
</html>
```

### head.html

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

### header.html

サイトタイトル（トップへのリンク）と Pagefind 検索ボックスを表示する。

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

### sidebar-left.html

現在のセクション（スペース）内のページをタイトルのアルファベット順でフラットリスト表示する。`parent` フィールドによる階層表示は行わない（最低限実装のため）。

```html
{{- $section := .CurrentSection }}
{{- with $section }}
  <nav>
    <h2><a href="{{ .RelPermalink }}">{{ .Title }}</a></h2>
    <ul>
      {{- range sort .RegularPages "Title" "asc" }}
        <li><a href="{{ .RelPermalink }}">{{ .Title }}</a></li>
      {{- end }}
    </ul>
  </nav>
{{- end }}
```

### home.html

スペース（`site.Sections`）をタイトル順に一覧表示する。`space_title` パラメータがあれば優先して表示する。

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

### section.html

スペース内の全ページをタイトル + 日付のリスト表示。左サイドバーを表示する。

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

### page.html

ページタイトル（`<h1>`）+ 本文（`.Content`）を表示する。`confluence_url` と `labels` があれば末尾に表示する。

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

### footer.html

シンプルなフッター。

```html
<p>{{ site.Title }}</p>
```

### render-link.html

`hugo-theme-issues` から流用。Markdown 内のリンクを `target="_blank"` なしで素直にレンダリングする。

```html
<a href="{{ .Destination }}"{{ with .Title }} title="{{ . }}"{{ end }}>{{ .Text | safeHTML }}</a>
```

## CSS 設計

CSS Grid を使った2カラムレイアウト（sidebar-left / main）。フォント・色はブラウザデフォルトを活用し、レイアウトと最低限の本文スタイルのみ定義する。

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
}
header { grid-area: header; padding: .5rem 1rem; border-bottom: 1px solid #ddd; }
.sidebar-left { grid-area: sidebar; overflow-y: auto; padding: 1rem; border-right: 1px solid #ddd; }
main { grid-area: main; padding: 1rem 2rem; }
footer { grid-area: footer; padding: .5rem 1rem; border-top: 1px solid #ddd; font-size: .8rem; }

/* ホームはサイドバーなし */
body.home {
  grid-template:
    "header" auto
    "main" 1fr
    "footer" auto
    / 1fr;
}

/* 検索ボックス */
.header-search { display: inline-block; margin-left: 1rem; vertical-align: middle; }

/* 本文 */
table { border-collapse: collapse; width: 100%; margin: 1rem 0; }
th, td { border: 1px solid #ccc; padding: .4rem .6rem; text-align: left; }
th { background: #f0f0f0; }
pre { background: #f5f5f5; padding: .8rem; overflow-x: auto; border-radius: 4px; }
code { background: #f5f5f5; padding: .1rem .3rem; border-radius: 2px; }
pre code { background: none; padding: 0; }
img { max-width: 100%; }

/* ページメタ */
.page-meta { margin-top: 2rem; padding-top: 1rem; border-top: 1px solid #ddd; font-size: .85rem; }
.label { background: #e8e8e8; padding: .1rem .4rem; border-radius: 3px; margin-right: .3rem; font-size: .8rem; }

/* サイドバーナビ */
.sidebar-left nav ul { list-style: none; padding: 0; margin: 0; }
.sidebar-left nav ul li a { display: block; padding: .25rem .5rem; text-decoration: none; border-radius: 3px; }
.sidebar-left nav ul li a:hover { background: #f0f0f0; }
```

## hugo.toml の変更

### `hugo-site/themes/hugo-theme-docs/hugo.toml`（テーマ定義）

```toml
[module.hugoVersion]
  extended = false
  min = "0.146.0"
```

### `hugo-site/hugo.toml`（サイト設定）の変更点

```toml
theme = "hugo-theme-docs"   # "hugo-book" から変更

# [params] セクション: Hugo Book 固有パラメータをすべて削除

[markup.goldmark.renderer]
  unsafe = true   # <details> 等のため維持
```

## このプロジェクトへの影響範囲

| 対象 | 変更内容 |
|---|---|
| `hugo-site/themes/hugo-theme-docs/` | テーマファイルを新規作成（現在は LICENSE のみ） |
| `hugo-site/themes/hugo-book/` | 削除（git submodule を削除） |
| `hugo-site/hugo.toml` | theme 変更・Hugo Book パラメータ削除 |
| `hugo-site/layouts/` | 削除（Pagefind inject は新テーマに内包） |
| `migConfluence` 本体（Go コード） | 変更なし |
