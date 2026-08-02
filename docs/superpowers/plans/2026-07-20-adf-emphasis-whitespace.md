# ADF強調マークの前後空白によるMarkdown崩れ修正 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ADF（ページ本文）の `strong`/`em`/`strike` マークを持つテキストの前後に空白がある場合、CommonMarkのデリミタフランキング規則で装飾が閉じられずリテラル表示される不具合を修正する。

**Architecture:** `adfconverter.go` の `renderText` に、テキスト前後の空白をデリミタの外側へ退避させるヘルパー関数 `wrapDelimiter` を追加し、`strong`/`em`/`strike` の3ケースをこのヘルパー経由に置き換える。`code`/`underline`/`link`/`subsup` はCommonMarkのデリミタフランキング規則の対象外のため変更しない。

**Tech Stack:** Go（標準ライブラリ `strings` のみ）、`go test` によるユニットテスト。

## Global Constraints

- 変更対象は `adfconverter.go` のみ（`converter.go` は検証の結果、問題が再現しないためスコープ外）
- `code`（バッククォート）・`underline`（`<u>`タグ）・`link`（`[text](url)`）・`subsup`（`<sup>/<sub>`タグ）は変更しない
- 既存テスト（`TestConvertADF_Bold`, `TestConvertADF_Italic`, `TestConvertADF_Strikethrough` 等、前後空白のない通常ケース）の挙動は変えない

---

## 参照仕様

`docs/superpowers/specs/2026-07-20-adf-emphasis-whitespace-design.md`

## Task 1: wrapDelimiter ヘルパーによる strong/em/strike の空白退避

**Files:**
- Modify: `adfconverter.go:144-180`（`renderText` 関数。直前に `wrapDelimiter` ヘルパーを新規追加）
- Test: `adfconverter_test.go`（末尾に新規テスト関数を追加）

**Interfaces:**
- Consumes: なし（既存の `ADFNode`/`ADFMark` 構造体、既存の `adfDoc`/`adfText` テストヘルパーのみ使用）
- Produces: `func wrapDelimiter(text, delimiter string) string`（`renderText` 内でのみ使用。他タスクからの依存なし）

- [ ] **Step 1: 失敗するテストを書く**

`adfconverter_test.go` の末尾（`TestConvertADF_InternalLink` の後）に以下を追加する。

```go
func TestConvertADF_BoldTrailingSpace(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"hello ","marks":[{"type":"strong"}]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "**hello** ") {
		t.Errorf("got %q, want to contain %q", got, "**hello** ")
	}
	if strings.Contains(got, "hello **") {
		t.Errorf("got %q, closing delimiter must not be preceded by a space", got)
	}
}

func TestConvertADF_ItalicLeadingSpace(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":" hi","marks":[{"type":"em"}]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, " *hi*") {
		t.Errorf("got %q, want to contain %q", got, " *hi*")
	}
	if strings.Contains(got, "* hi") {
		t.Errorf("got %q, opening delimiter must not be followed by a space", got)
	}
}

func TestConvertADF_StrikethroughSurroundingSpace(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":" del ","marks":[{"type":"strike"}]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, " ~~del~~ ") {
		t.Errorf("got %q, want to contain %q", got, " ~~del~~ ")
	}
}

func TestConvertADF_BoldWhitespaceOnly(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"   ","marks":[{"type":"strong"}]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(got, "**") {
		t.Errorf("got %q, whitespace-only text must not be wrapped in delimiters", got)
	}
}

func TestConvertADF_BoldItalicOverlapWithSpaces(t *testing.T) {
	adf := adfDoc(`{"type":"paragraph","content":[{"type":"text","text":" foo ","marks":[{"type":"em"},{"type":"strong"}]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, " ***foo*** ") {
		t.Errorf("got %q, want to contain %q", got, " ***foo*** ")
	}
}

func TestConvertADF_PanelHeadingBoldTrailingSpace(t *testing.T) {
	adf := adfDoc(`{"type":"panel","attrs":{"panelType":"info"},"content":[{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"新しいスペースへようこそ! ","marks":[{"type":"strong"}]}]}]}`)
	got, err := convertADF(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "**新しいスペースへようこそ!** ") {
		t.Errorf("got %q, want NOTE panel heading to render as closed bold", got)
	}
	if strings.Contains(got, "! **") {
		t.Errorf("got %q, regression: literal ** must not appear (original bug)", got)
	}
}
```

- [ ] **Step 2: テストを実行し、失敗することを確認する**

Run: `go test ./... -run 'TestConvertADF_BoldTrailingSpace|TestConvertADF_ItalicLeadingSpace|TestConvertADF_StrikethroughSurroundingSpace|TestConvertADF_BoldWhitespaceOnly|TestConvertADF_BoldItalicOverlapWithSpaces|TestConvertADF_PanelHeadingBoldTrailingSpace' -v`

Expected: 6件とも FAIL（`want to contain` のメッセージ、またはリテラル `**` が含まれてしまっているというメッセージ）

- [ ] **Step 3: wrapDelimiter ヘルパーを実装する**

`adfconverter.go:143`（`renderText` 関数定義の直前）に以下を追加する。

```go
// wrapDelimiter は前後の空白をデリミタの外側に保ったまま text をデリミタで囲む
func wrapDelimiter(text, delimiter string) string {
	core := strings.Trim(text, " \t\n")
	if core == "" {
		return text
	}
	start := strings.Index(text, core)
	lead := text[:start]
	trail := text[start+len(core):]
	return lead + delimiter + core + delimiter + trail
}
```

続けて `renderText` 内（[adfconverter.go:150-157](adfconverter.go#L150)）の `strong`/`em`/`strike` の3ケースを書き換える。

変更前:

```go
		case "strong":
			text = "**" + text + "**"
		case "em":
			text = "*" + text + "*"
		case "code":
			text = "`" + text + "`"
		case "strike":
			text = "~~" + text + "~~"
```

変更後:

```go
		case "strong":
			text = wrapDelimiter(text, "**")
		case "em":
			text = wrapDelimiter(text, "*")
		case "code":
			text = "`" + text + "`"
		case "strike":
			text = wrapDelimiter(text, "~~")
```

- [ ] **Step 4: テストを実行し、成功することを確認する**

Run: `go test ./... -run 'TestConvertADF_BoldTrailingSpace|TestConvertADF_ItalicLeadingSpace|TestConvertADF_StrikethroughSurroundingSpace|TestConvertADF_BoldWhitespaceOnly|TestConvertADF_BoldItalicOverlapWithSpaces|TestConvertADF_PanelHeadingBoldTrailingSpace' -v`

Expected: 6件とも PASS

- [ ] **Step 5: 全テストスイートを実行し、既存挙動に回帰がないことを確認する**

Run: `go test ./...`

Expected: PASS（既存の `TestConvertADF_Bold`, `TestConvertADF_Italic`, `TestConvertADF_Strikethrough`, `TestConvertADF_PanelInfo` 等を含め全件成功）

- [ ] **Step 6: コミット**

```bash
git add adfconverter.go adfconverter_test.go
git commit -m "fix: ADF強調マーク(strong/em/strike)の前後空白をデリミタ外に退避

CommonMarkのデリミタフランキング規則により、末尾/先頭に空白を含む
テキストを太字/斜体/取り消し線にすると閉じデリミタが認識されず
リテラル表示されていた問題を修正。NOTEパネル内の見出し太字などで
特に顕在化していたが、修正はrenderText全体に適用される。"
```

- [ ] **Step 7: 手動受け入れ確認（Hugoでの表示確認）**

`content/SCRUM/test-migration Home/index.md` の実データで問題を再現していた箇所を、
修正後の変換結果に合わせて手動修正し、Hugoでの描画を目視確認する。

`hugo-site/content/SCRUM/test-migration Home/index.md` の該当行を編集:

変更前:
```markdown
> ## **新しいスペースへようこそ! **
```

変更後（Step 3 の修正により実際に生成される形）:
```markdown
> ## **新しいスペースへようこそ!** 
```

確認コマンド（`hugo-site` ディレクトリで実行）:

```bash
cd hugo-site
hugo --gc --minify=false -d /tmp/hugo_verify
grep -A2 'alert-note' /tmp/hugo_verify/scrum/test-migration-home/index.html
```

Expected: 出力が `<h2 id="...">​<strong>新しいスペースへようこそ!</strong></h2>` のように、
`<strong>` タグとして正しく変換され、リテラルな `**` が残らないこと
（修正前は `<h2 id="...">**新しいスペースへようこそ! **</h2>` のようにリテラル表示されていた）。
`/tmp/hugo_verify` は確認用の一時出力なので、コミット対象に含めない。

