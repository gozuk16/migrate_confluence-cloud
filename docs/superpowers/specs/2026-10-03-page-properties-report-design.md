# ページプロパティレポート（detailssummary）再現 設計

## 背景

Confluence の「ページプロパティレポート」（`detailssummary` マクロ）は、CQL の条件に合うページを集め、
各ページの「ページプロパティ」（`details` マクロ内の表）を一覧表にして表示する。
現状の変換では `<!-- macro: detailssummary -->` というコメントになり、何も表示されない。

対象例: `SCRUM/2026-8-8`

- `detailssummary` の CQL: `label = "memo" and space = currentSpace ( ) and parent = currentContent ( )`
- 子ページ `メモ` / `メモ2` に `details` マクロ（日付・ステータス・なにか）とラベル `memo` がある

## 方式

CQL の解析は Go（変換器）で行い、平らな絞り込み条件にしてショートコードの引数にする。
ページの集計と表の作成は Hugo のショートコードがビルド時に行う。

- 構文解析を Go の単体テストで固められる
- 子ページだけを再変換・追加しても、ビルドすれば常に最新の結果になる
- `now("-4w")` のような相対日付はビルド時刻を基準に評価する

## 変換器（Go）

### details（ページプロパティ）

- 本文の出力はこれまでと同じ（表をそのまま出す）。
- 表の各行の「1つ目のセル（見出し）→ 2つ目のセル（値）」を取り出し、front matter に順序つきで保存する。
  - 項目名はセル内のテキスト（書式を除いたもの）。
  - 値はセルの中身を本文の表のセルと同じ形式（Markdown とインライン HTML の混在）にしたもの。
    Hugo 側で `markdownify` して表示するので、ステータスバッジ・日付・リンクなどがレポートでも同じ見た目になる。
- `[[properties]]` は TOML の配列テーブルなので、front matter の末尾（他のキーをすべて書いた後）に出力する。
- 1ページに `details` が複数ある場合はすべての項目をまとめる。同じ項目名は先に出てきた方を採用する。
- `details` の `id` パラメータによる区別は行わない。

```toml
[[properties]]
  key = "日付"
  value = "2026-10-03"
[[properties]]
  key = "ステータス"
  value = "<span style=\"...\">release</span>"
```

### detailssummary（ページプロパティレポート）

`cql` を解析し、表示オプションと合わせて次のショートコードを出力する（値が無い引数は出力しない）。

```
{{< page-properties-report labels="memo" labels_mode="all" space="current" scope="children" root="current" >}}
```

#### CQL の対応表

| CQL | 引数 | 意味 |
|---|---|---|
| `label = "a"`、`label = "a" and label = "b"` | `labels="a,b"` `labels_mode="all"` | すべてのラベルを持つ |
| `label in ("a","b")`、`label = "a" or label = "b"` | `labels="a,b"` `labels_mode="any"` | いずれかのラベルを持つ |
| `label != "x"`、`label not in (...)` | `labels_exclude="x"` | 除外 |
| `space = currentSpace()` / `space = "KEY"` | `space="current"` / `space="KEY"` | スペース指定（無ければ全スペース） |
| `parent = currentContent()` / `parent = 123` | `scope="children"` `root="current"` / `root="123"` | 直下の子ページ |
| `ancestor = currentContent()` / `ancestor = 123` | `scope="descendants"` `root=...` | 配下のすべてのページ |
| `title = "x"` | `title_is="x"` | タイトル完全一致 |
| `title ~ "x"` | `title_contains="x"` | 部分一致（大文字小文字を区別しない、`*` は除去） |
| `created` の `> >= < <= =` | `created_from` / `created_to` | 作成日（front matter の `date`） |
| `lastmodified` の `> >= < <= =` | `lastmod_from` / `lastmod_to` | 更新日（front matter の `lastmod`） |
| `type = page` | （無視） | 移行後はページのみのため |

- 日付の値は `"2026-01-01"`、`"2026/01/01"`、`now("-4w")` を受け付ける。
  - 絶対日付は `YYYY-MM-DD` に正規化して渡す。
  - `now(...)` は `now-4w` の形で渡し、Hugo 側でビルド時刻を基準に計算する。単位は `d` / `w` / `M`（月）/ `y`。`now()` は `now` として渡す。
  - `>` / `>=` は `*_from`、`<` / `<=` は `*_to`、`=` は同じ日の `*_from` と `*_to` にする。
  - 比較は日単位で行う。`>` は翌日以降、`<` は前日以前として扱う（`from` / `to` はどちらも境界を含む）。
    ただし `now(...)` の相対日付は Go 側で日をずらせないため、`>` / `<` も `>=` / `<=` と同じに扱う（1日分の差は許容する）。
- 関数名・キーワード・演算子の大文字小文字は区別しない。`currentSpace ( )` のような空白も許す。
- `or` は label どうしの場合だけ受け付ける。label の `and` と `or` が混在する場合は警告を出し、label 条件を `labels_mode="any"` として扱う。
- 次の場合は警告を出し、その条件を除外して残りで絞り込む。
  - 異なる項目どうしの `or`、括弧の入れ子
  - 対応表に無い項目（creator、contributor、text など）
  - 解釈できない日付
- `cql` が空、またはまったく解析できない場合は `space="current"` のみ（同じスペースの全ページが対象）。

#### 表示オプション

| マクロ引数 | ショートコード引数 | 内容 |
|---|---|---|
| `headings` | `headings` | 表示する列（カンマ区切り、その順に表示） |
| `sortBy` | `sort_by` | 並べ替えに使うプロパティ名 |
| `reverseSort` | `reverse` | `true` で逆順 |
| `firstcolumn` | `first_column` | 1列目の見出し |
| `pageSize` | `page_size` | 表示件数の上限 |

`showCommentsCount` / `showLikesCount` などその他の引数は無視する。

#### 警告

CQL 解析の警告は変換時のログに「ページ名・除外した条件」を出す。

## Hugo 側

ショートコード `layouts/_shortcodes/page-properties-report.html` をテーマ（hugo-theme-docs）に置く。
あわせて、サイト側にある `hugo-site/layouts/shortcodes/toc.html` もテーマへ移す。

### 絞り込み

サイト全体の通常ページ（`site.RegularPages`）から、次の条件をすべて満たすページを選ぶ。

1. `.Params.properties` が1件以上ある
2. `space`: `current` ならレポートのページと同じ `.Params.space`、それ以外はその値と一致
3. `scope="children"`: `.Params.parent_id` が `root` のページID（`current` ならレポートのページの `page_id`）と一致
4. `scope="descendants"`: `parent_id` をたどった先祖に `root` のページIDが含まれる（ページIDから親を引ける辞書を作ってたどる。循環に備えて深さの上限を設ける）
5. `labels`: `labels_mode="all"` なら全ラベルを含む、`any` ならいずれかを含む
6. `labels_exclude`: どれも含まない
7. `title_is` / `title_contains`
8. `created_*` / `lastmod_*`: `.Date` / `.Lastmod` を日付（`YYYY-MM-DD`）で比較

### 表

- 1列目はページタイトル（リンク）。見出しは既定で「タイトル」、`first_column` があればその名前。
- 2列目以降はプロパティ項目名。既定は Confluence と同じく項目名の文字コード順（重複は除く）、`headings` があればその列だけをその順に表示する。
- ページに無い項目のセルは空欄。
- 並び順は Confluence と同じく、既定で最終更新日（`lastmod`）の新しい順。`reverse="true"` なら古い順。`sort_by` があればそのプロパティ値（HTML タグを除いた文字列）の昇順、`reverse="true"` で降順。同値はタイトル順。
- `page_size` があればその件数で打ち切る（ページ送りは付けない）。
- 該当ページが無い場合は「条件に一致するページはありません」と表示する。
- 表には `.page-properties-report` クラスを付け、見た目は既存の表のスタイルを使う。

## テスト

- Go（TDD）
  - CQL 解析: 対応表の各行、大文字小文字・空白、警告になるケース、空の CQL
  - `details` からのプロパティ抽出（複数 details、重複項目、書式つきの見出し）
  - front matter への `[[properties]]` 出力（TOML のエスケープ）
  - `detailssummary` のショートコード出力（表示オプションを含む）
- Hugo / 実データ
  - SCRUM スペースを再変換・ビルドし、Playwright で `2026-8-8` に メモ / メモ2 の2行（日付・ステータス・なにか）が表示されることを確認する
  - 条件違い（descendants、labels any/exclude、title、日付、sort_by/reverse、headings、page_size、該当なし）は、作業用ディレクトリに一時的な Hugo サイトを作って確認する

## 進め方

- テーマ（hugo-theme-docs）と親リポジトリでそれぞれブランチを作り、PR を2本作る。
- 前回保留した submodule ポインタの更新を、親リポジトリの PR に含める。
