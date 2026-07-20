# ADF 強調マークの前後空白によるMarkdown崩れ修正 設計ドキュメント

- 日付: 2026-07-20
- ステータス: 承認済み
- 対象: `adfconverter.go`（ページ本文の ADF → Markdown 変換パス）

## 背景・課題

NOTEパネル（GFM Alert `> [!NOTE]`）内で太字にした見出しが、実際のサイト
（`http://localhost:1313/scrum/test-migration-home/`）で `**新しいスペースへようこそ! **`
のようにリテラル表示され、太字として描画されない。

実データ（`content/SCRUM/test-migration Home/index.md`）:

```markdown
> [!NOTE]
> ## **新しいスペースへようこそ! **
```

### 原因

`!` の後ろに空白があり、その直後に閉じの `**` が続いている。CommonMark の強調(emphasis)
デリミタ規則では、閉じ側のデリミタは直前が空白だと「右フランキング」にならず閉じ扱いに
ならない。そのため `**` がリテラル文字として出力される。

検証の結果、この問題は **NOTEパネル固有ではなく**、末尾または先頭に空白を含むテキストを
`strong` / `em` / `strike` マークで装飾した場合に一般的に発生することを確認した
（独立したテストページで、NOTE内外を問わず同一の現象を再現・確認済み）。

原因箇所は [adfconverter.go:144-180](../../../adfconverter.go#L144) の `renderText` で、
`strong` / `em` / `strike` マークを適用する際にテキスト前後の空白を考慮せず
デリミタで単純に囲んでいる。

なお、コメント変換パス（`converter.go`、`html-to-markdown/v2` ライブラリ経由）は
同様のケース（`<strong>world </strong>` → `**world** foo`）で空白を正しくマーカー外に
退避させることを確認済みであり、本件の影響を受けない。

## 方針

`renderText` で `strong` / `em` / `strike` マークを適用する直前に、テキスト前後の空白を
デリミタの外側へ退避させるヘルパー関数 `wrapDelimiter` を追加し、該当3ケースをこの
ヘルパー経由に置き換える。

検討した代替案:

- **テキスト全体を事前に一括 trim してから最外周に空白を戻す**: 複数マークが重なる場合
  （例: strong+em）に対応するには結局各マーク適用時の処理が必要になり、実装が複雑化する
  ため不採用。今回の方式（各マーク適用ごとに独立して空白退避）は既存の「マークを逆順に
  適用（内側から外側へラップ）」ループにそのまま組み込め、シンプル。
- **変換後の文字列に対して正規表現で後処理する**: マーク適用箇所を後から特定するのが
  困難（ネストや複数出現に対応しづらい）ため不採用。

`code`（バッククォート）・`underline`（`<u>` タグ）・`link`（`[text](url)`）・
`subsup`（`<sup>/<sub>` タグ）は CommonMark のデリミタフランキング規則の対象外
（バッククォートはフランキング規則を持たず、他は HTML タグ/リンク記法で空白の影響を
受けない）ため変更不要。

## 設計

### wrapDelimiter ヘルパー

```go
// wrapDelimiter は前後の空白をデリミタの外側に保ったまま text をデリミタで囲む
func wrapDelimiter(text, delimiter string) string {
    core := strings.Trim(text, " \t\n")
    if core == "" {
        return text // 空白のみなら装飾しない
    }
    lead := text[:strings.Index(text, core)]
    trail := text[strings.Index(text, core)+len(core):]
    return lead + delimiter + core + delimiter + trail
}
```

### renderText の変更

`strong` / `em` / `strike` の3ケースを `text = wrapDelimiter(text, "**")` 等に置き換える。

```go
case "strong":
    text = wrapDelimiter(text, "**")
case "em":
    text = wrapDelimiter(text, "*")
case "strike":
    text = wrapDelimiter(text, "~~")
```

マークは既存どおり配列を逆順（内側から外側）に適用するため、複数マークが重なる場合
（例: `" foo "` に strike→strong の順で適用）も、各段階で独立して空白退避が行われ、
最終的に空白は最も外側に残る（例: `" ~~**foo**~~ "`）。

## エラーハンドリング

- テキストが空白のみ（`strings.Trim` の結果が空文字列）の場合はデリミタを付与せず
  元のテキストをそのまま返す（意味のない `****` のような出力を避ける）
- 前後空白を含まない通常のテキストは従来と同じ出力になる（回帰なし）

## テスト

`adfconverter_test.go` に以下の単体テストを追加する。

1. 末尾スペース付きテキストに `strong` マーク（今回の再現ケース）
2. 先頭スペース付きテキストに `em` マーク
3. 前後スペース付きテキストに `strike` マーク
4. 空白のみのテキストに `strong` マークが付くケース（装飾なしで空白のみ出力されることを確認）
5. 前後空白なしの通常テキストへの `strong`/`em`/`strike`（既存挙動の回帰確認）
6. `strong` + `em` など複数マークが重なり、かつ前後に空白があるケース

受け入れ確認として、`content/SCRUM/test-migration Home/index.md` を実データとして
`hugo server` でブラウザ描画し、NOTEパネル内の見出しが正しく太字表示されることを確認する。

## スコープ外

- コメント変換パス（`converter.go`、Storage Format）: 検証の結果、問題が再現しないため対応不要
- `textColor` / `backgroundColor` / `annotation` マーク（現状どおり意図的に無視、別課題）
- NOTE以外のブロック要素（テーブル・リスト等）内の強調マークも本修正の対象に含まれるが、
  これは仕様上の副次効果であり、個別の追加対応は不要（同じ `renderText` を通るため自動的に解消）
