---
title: Confluence移行先 静的サイト構成設計
date: 2026-06-20
status: approved
---

# Confluence移行先 静的サイト構成設計

## 概要

`migConfluence` で変換した Markdown を Hugo + Pagefind を使って静的ドキュメントサイトとして公開する。対象は社内イントラネット。全文検索が必須要件。

## 全体アーキテクチャ

```
[Confluence Cloud]
       ↓ REST API
[migConfluence (Go CLI)]
       ↓ Markdown + TOML Front Matter (Hugo Leaf Bundle形式)
[Hugo コンテンツディレクトリ]
       ↓ hugo build
[public/ (静的HTML)]
       ↓ Pagefind インデックス生成
[社内Webサーバー (nginx等)]
```

### パイプラインの流れ

1. `migConfluence` が Confluence から ADF → Markdown に変換し、Hugo Leaf Bundle 形式で出力
2. Hugo がコンテンツとテーマを組み合わせて静的 HTML に変換
3. Pagefind が `public/` を走査して全文検索インデックスを生成
4. `public/` ごと社内サーバーに配置して完了

Confluence を更新したら「migConfluence 実行 → hugo build → 配置」の3ステップで再同期できる。

## Hugo サイト構成

### ディレクトリ構造

```
hugo-site/                    ← Hugo プロジェクトルート（新規作成）
├── hugo.toml                 ← サイト設定
├── themes/
│   └── hugo-book/            ← テーマ（git submodule）
├── layouts/
│   └── partials/
│       └── docs/
│           └── inject/
│               └── head.html ← Pagefind UI 埋め込み用（テーマ本体は変更不要）
├── content/                  ← migConfluence の出力をそのまま配置
│   └── SPACE_KEY/
│       └── PAGE_TITLE/
│           └── index.md
└── public/                   ← hugo build の出力先
```

### テーマ：Hugo Book

- 左サイドバーナビゲーションを自動生成
- セットアップが簡単（Node.js 不要）
- Pagefind との相性が良い
- シンプルな社内 Wiki 向けに適している

### Front Matter 互換性

`migConfluence` が出力する TOML Front Matter（`+++...+++`）は Hugo がそのまま読める。`title`、`date`、`lastmod` は Hugo 標準フィールドのため追加対応不要。

## 全文検索：Pagefind

### 仕組み

Pagefind は `hugo build` 後に `public/` を走査してインデックスを生成する。バックエンド不要で、生成された JS/CSS を読み込むだけで検索 UI が動作する。

### 組み込み手順

```bash
# 1. Hugo ビルド
hugo --minify

# 2. Pagefind インデックス生成
pagefind --site public

# → public/_pagefind/ にインデックスが生成される
```

### Hugo Book テーマへの検索 UI 埋め込み

`layouts/partials/docs/inject/head.html` を作成して以下を追加（テーマ本体は変更不要）:

```html
<link href="/_pagefind/pagefind-ui.css" rel="stylesheet">
<script src="/_pagefind/pagefind-ui.js"></script>
<div id="search"></div>
<script>
  new PagefindUI({ element: "#search", showSubResults: true });
</script>
```

## 運用フロー

### ローカル確認

```bash
# Hugo の開発サーバーで即時確認（Pagefind なしでも閲覧可）
cd hugo-site && hugo server

# 検索も含めてフル確認する場合
hugo --minify && pagefind --site public
python -m http.server -d public 8080
```

### Confluence 同期 → 公開（Makefile ターゲット）

```makefile
sync-and-build:
    ./migConfluence space
    cp -r output/markdown/* hugo-site/content/
    cd hugo-site && hugo --minify
    pagefind --site hugo-site/public
```

社内サーバー側は nginx で `public/` を静的配信するだけで完結する。

## このプロジェクトへの影響範囲

| 対象 | 変更内容 |
|---|---|
| `migConfluence` 本体 | **変更なし**（出力形式そのまま） |
| `hugo-site/` | 新規ディレクトリ（同リポジトリ内または別リポジトリ） |
| `Makefile` | `sync-and-build` ターゲット追加のみ |

`migConfluence` 本体のコードには手を入れず、Hugo サイトを「別途用意する成果物置き場」として扱う。

## 選定理由

- **Hugo を選んだ理由**: `migConfluence` の出力が既に Hugo Leaf Bundle 形式のため、変換ツール側の改修が不要。単一バイナリで動作し、社内サーバーへの導入が容易。
- **Pagefind を選んだ理由**: `hugo build` 後にワンコマンドで全文検索インデックスを生成できる。JS のみで動作しバックエンド不要。社内イントラネット環境に最適。
- **自前 HTML 生成を選ばなかった理由**: テンプレート・CSS・ナビゲーション・検索インデックス生成をすべて自前実装する必要があり、工数対効果が悪い。SSG が既に解決済みの問題を再発明することになる。
