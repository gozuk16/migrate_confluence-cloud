# ADF テーブル変換の GFM 近似強化 設計ドキュメント

- 日付: 2026-07-04
- ステータス: 承認済み
- 対象: `adfconverter.go`（ページ本文の ADF → Markdown 変換パス）

## 背景・課題

Confluence Cloud から変換した Markdown のテーブルで、以下の要素が正しく変換されない。

実際の変換結果（`output/markdown/SCRUM/2026-5-13 テスト議事録/index.md`）:

```markdown
| **ヘッダ** |  |  |  |
| --- | --- | --- | --- |
| - a   - b     - c | 縦結合 | <!-- macro: nested-table --> | 122 |
| > 引用 | 横結合 |
```

### 原因

| 症状 | 原因箇所 |
|---|---|
| セル内リスト・引用が潰れる | `renderTableCell` が改行をスペースに置換して平坦化している |
| 縦結合・横結合で列がずれる | ADF の `rowspan` / `colspan` 属性を無視している |
| 入れ子テーブルがコメントになる | `extensionKey: "nested-table"` 拡張ノードの `parameters.adf`（内側テーブルの ADF JSON）を捨てている |
| 配置（左寄せ・中央・右寄せ）が消える | セル内段落の `alignment` マークを無視している |

## 方針

**テーブルは GFM 記法を維持し、GFM で表現できないセル内要素は HTML タグとしてセル内に埋め込む。**

これは姉妹プロジェクト migrate_jira-cloud の `jira_tables.go`
（`convertCellListsToHTML`: セル内リスト → `<ul>/<ol>/<li>`、残改行 → `<br>`）と同じ考え方。
`hugo-site/hugo.toml` に `[markup.goldmark.renderer] unsafe = true` が設定済みのため、
セル内の HTML は Hugo（goldmark）がそのまま描画する。

検討した代替案:

- **複雑なテーブルのみ HTML `<table>` 全体にフォールバック**: 忠実性は最も高いが、Markdown ソースの一貫性が失われるため不採用。
- **全角スペースインデント + `<br>` による疑似リスト**: HTML に依存しないが、描画が本物のリストにならず migrate_jira-cloud との一貫性もないため不採用。

## 設計

変更対象は `adfconverter.go` のみ。コメント用の Storage Format パス（`converter.go`、
html-to-markdown のテーブルプラグインが処理）はスコープ外。

### 1. セル内ブロック要素（`renderTableCell` の書き換え）

改行潰しをやめ、ノード種別ごとにセル内表現へ変換する。

| ADF ノード | セル内出力 |
|---|---|
| paragraph（複数） | `<br>` で結合 |
| bulletList / orderedList（入れ子含む） | `<ul><li>a<ul><li>b</li></ul></li></ul>` を1行の HTML で埋め込み |
| blockquote | `<blockquote>内容</blockquote>` |
| codeBlock | `<code>内容</code>`（コード内改行は `<br>`） |
| taskList | `<ul>` + ☑ / ☐ プレフィックス |
| panel / expand | 内容テキストを `<br>` 結合 |
| nested-table 拡張 | 下記 4. 参照 |

セル内テキストの `|` は `\|` にエスケープする（現状踏襲）。

### 2. 縦結合（rowspan）・横結合（colspan）: グリッド展開

テーブル全体を一度仮想グリッド（2次元配列）に展開して出力する。

- 各行の走査時に、前行からの `rowspan` で占有済みの列位置をスキップして配置する
- `colspan=N` のセルは内容を先頭位置に置き、残り N-1 個は空セルで埋める
- `rowspan=N` のセルは内容を最初の行に置き、以降 N-1 行の同列位置に空セルを置く
- 列数はグリッド展開後の最大幅で統一する

これにより全行のセル数が揃い、GFM テーブルとして列ずれなく描画される
（見た目のセル結合は GFM の構造上再現できず、空セルによる近似となる）。

### 3. ヘッダー無しテーブル

ADF の1行目のセルが `tableHeader` でなく `tableCell` の場合、
migrate_jira-cloud 方式と同様に**空ヘッダー行 + 区切り行を自動生成**する
（現状は1行目のデータ行がヘッダー扱いになってしまう）。

判定は1行目に `tableHeader` が1つ以上含まれるかで行う。

### 4. 入れ子テーブル

Confluence Cloud API は入れ子テーブルを
`extensionType: "com.atlassian.confluence.migration"` / `extensionKey: "nested-table"`
の拡張ノードとして返し、`attrs.parameters.adf` に内側テーブルの ADF JSON 文字列が入っている。

- `renderExtension` で `extensionKey == "nested-table"` を検出したら `parameters.adf` を
  再帰的にパース・変換する
- 出力は**セル内に `<table><tr><th>…</th></tr><tr><td>…</td></tr></table>` を
  1行の HTML として埋め込む**（セル内 HTML 方式との一貫性を優先）
- 内側テーブルのセル内要素も同じセル内 HTML 規則で変換する（再帰）
- `parameters.adf` が欠落・パース不能な場合は現状のコメント出力
  `<!-- macro: nested-table -->` にフォールバックする

### 5. 配置（alignment）

セル内段落の `alignment` マーク（`align: "center"` / `"end"`）を列単位に集約し、
GFM の区切り行記法に反映する。

- 列内に配置指定を持つセルが1つ以上あれば、最初に見つかった指定を列の配置とする
- `center` → `:---:`、`end` → `---:`、無指定・`start` → `---`（テーマ CSS のデフォルトが左寄せ）
- 段落単位の配置は列単位に丸められる（GFM の制約による近似）
- テーブル外の本文段落の alignment は本件のスコープ外（従来通り無視）

## エラーハンドリング

- 未知のノード種別がセル内に現れた場合は、既存の `renderNode` の結果を
  改行→`<br>` 置換して埋め込む（情報を捨てない）
- 入れ子テーブルの ADF パース失敗時はコメント出力にフォールバック（前述）
- `rowspan` / `colspan` が数値でない・欠落している場合は 1 として扱う

## テスト

`adfconverter_test.go` に以下の単体テストを追加する。

1. セル内 bulletList（3階層入れ子）→ `<ul>` 入れ子 HTML
2. セル内 orderedList → `<ol>` HTML
3. セル内 blockquote → `<blockquote>`
4. セル内複数段落 → `<br>` 結合
5. `rowspan=2` → 継続行の同列位置に空セル
6. `colspan=2` → 同行の後続位置に空セル
7. rowspan + colspan 混在（SCRUM サンプル相当）
8. 入れ子テーブル（nested-table 拡張）→ セル内 `<table>` HTML
9. 入れ子テーブルの ADF パース失敗 → コメントフォールバック
10. ヘッダー無しテーブル → 空ヘッダー行の自動生成
11. alignment マーク（center / end）→ 区切り行 `:---:` / `---:`

受け入れ確認として、SCRUM サンプルページ（`2026-5-13 テスト議事録`）を再変換し、
`hugo server` でブラウザ描画をビジュアル確認する。

## スコープ外

- コメント変換（Storage Format / XHTML パス、`converter.go`）
- テーブル外の本文段落の alignment
- セル背景色・列幅（`colwidth`）などの装飾属性
- Hugo テーマ側の変更（既存 CSS で描画可能なため不要）
