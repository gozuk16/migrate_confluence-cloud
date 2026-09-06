# 左サイドバーのリサイズ・開閉 実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 左サイドバーを境界のドラッグで左右にリサイズでき、ヘッダーのボタンで開閉でき、その状態がページを移動しても保たれるようにする。

**Architecture:** 幅と開閉状態を `<html>` 要素に持たせ（CSSカスタムプロパティ `--sidebar-width` と属性 `data-sidebar`）、CSS Grid の列幅がそれを参照する。`<head>` 内のインラインスクリプトが localStorage の値を初回描画前に反映してちらつきを防ぎ、`assets/js/sidebar.js` がドラッグとトグルの操作を担当する。

**Tech Stack:** Hugo v0.163.3 extended、素のCSS（CSS Grid・カスタムプロパティ）、依存ライブラリなしのJavaScript（Pointer Events・localStorage）、検証にPlaywright。

## Global Constraints

- 設計の正典は `docs/superpowers/specs/2026-08-11-sidebar-resize-collapse-design.md`。矛盾が出たらスペックを優先し、逸脱するなら先に相談する。
- **対象はPCの画面幅のみ**。モバイル向けのオーバーレイ表示は対象外。メディアクエリによる分岐を足さないこと。
- 幅の範囲は **最小160px・最大480px・既定240px**。この3つの数値はインラインスクリプト内で一度だけ定義し、`window.sidebarConfig` として公開して `sidebar.js` から参照する（CSSの既定値240pxだけは、JSが無い場合のために別途CSSにも書く）。
- **JavaScriptライブラリを追加しないこと**。素のJSで書く。
- コード内コメント・コミットメッセージは日本語。既存コードの慣習に合わせる。
- 作業ブランチは親リポジトリ・テーマsubmoduleとも `feature/sidebar-resize-collapse`（どちらも `main` から作成済み）。mainへ直接コミットしない。
- **テーマは git submodule（別リポジトリ `git@github.com:gozuk16/hugo-theme-docs.git`）**。`hugo-site/themes/hugo-theme-docs/` 配下の変更はそのsubmoduleリポジトリ内でコミットし、親リポジトリではポインタ更新を別途コミットする。push と PR 作成は最終タスクでまとめて行う。
- 各タスクの最後に必ずコミットする。

## 前提の検証結果（実施済み・再確認不要）

WebKit（Playwright）で実機確認済み。実装時に疑う必要はない。

- **CSS変数の二段構えは意図どおり動く。** `--sidebar-width`（JSがインラインで設定）を `--sidebar-current: var(--sidebar-width)` が受け、`html[data-sidebar="collapsed"]` が `--sidebar-current: 0px` で上書きする構成で、既定240px → JSが360pxに変更 → 閉じると0px → 開くと360pxに復帰、を確認した。**インラインスタイルで設定した `--sidebar-width` が残っていても、閉じた状態の0pxが正しく勝つ**（ここが要）。
- **CSSの `clamp()` に頼ってはいけない。** `--sidebar-current: clamp(var(--min), var(--sidebar-width), var(--max))` と書いても、WebKitでは範囲外の値（50px、9999px）がそのまま通り、クランプされなかった。**クランプはJS側で行うこと。**
- **不正な値は必ず弾くこと。** `--sidebar-width` に `abc` のような不正値が入ると、グリッドの列がコンテンツ幅で自動サイズになりレイアウトが崩れる（実測で567.5pxになった）。localStorageから読んだ値は必ず数値として検証してから適用する。
- **ホームページは影響を受けない。** `body.home` は `grid-template` を1カラムで上書きしているため、`data-sidebar` を切り替えても表示は変わらない。

## File Structure

**テーマ（submodule `hugo-site/themes/hugo-theme-docs`）**

| ファイル | 責務 | 変更種別 |
|---|---|---|
| `layouts/baseof.html` | `<html>` に `data-sidebar` を付与。`aside` をスクロール層と掴み手に分割。ヘッダーにトグルボタンを追加 | 変更 |
| `layouts/_partials/head.html` | ちらつき防止のインラインスクリプトと `sidebar.js` の読み込みを追加 | 変更 |
| `assets/js/sidebar.js` | ドラッグ・トグル・キーボード操作・localStorageへの保存 | 新規 |
| `assets/css/main.css` | 列幅の変数化、`aside` の構造変更、掴み手とトグルボタンのスタイル | 変更 |

**親リポジトリ**

| ファイル | 責務 | 変更種別 |
|---|---|---|
| `TODO.md` / `CHANGELOG.md` | 変更履歴の記録 | 変更 |
| submoduleポインタ | テーマの最新コミットを指す | 変更 |

**トグルボタンを `header.html` に入れてはいけない（重要）**

`baseof.html` はヘッダーを `partialCached "header.html" . "global"` で描画している。キャッシュキーが `"global"` なので、**最初にレンダリングされた1ページ分の出力が全ページで使い回される**。トグルボタンはホームページでは出力しない必要があるため、`header.html` の中に置くと「ホームで最初にレンダリングされたらボタンが全ページで消える」「非ホームで最初にレンダリングされたらホームにもボタンが出る」のどちらかになる。**ボタンは `baseof.html` の `<header>` 要素内に、`{{ if not .IsHome }}` で囲んで直接書くこと。**

---

### Task 1: CSSの土台とDOM構造

**Files（すべてsubmodule `hugo-site/themes/hugo-theme-docs` 内）:**
- Modify: `layouts/baseof.html`
- Modify: `assets/css/main.css`

**Interfaces:**
- Produces: 後続タスクが参照するDOM要素とCSSフック
  - `<html data-sidebar="open">`（属性は常に存在する。`open` または `collapsed`）
  - `#sidebar-left`（`<aside class="sidebar-left">`）
  - `.sidebar-scroll`（縦スクロールを担当する内側の要素）
  - `.sidebar-resizer`（右端の掴み手。`role="separator"`）
  - `.sidebar-toggle`（ヘッダーのトグルボタン）
  - CSSカスタムプロパティ `--sidebar-width`（既定240px）と `--sidebar-current`
  - クラス `js`（JSが有効なときに付く）、`sidebar-dragging`（ドラッグ中に付く）

このタスクの時点ではJavaScriptが無いため、`js` クラスが付かず、**トグルボタンと掴み手は表示されない**。見た目は変更前と同一になるのが正しい。

- [ ] **Step 1: `baseof.html` を書き換える**

`layouts/baseof.html` を以下の内容にする（変更点は `<html>` の属性、`<header>` 内のボタン、`<aside>` の構造の3箇所）:

```html
<!DOCTYPE html>
<html lang="{{ site.Language.Locale }}" data-sidebar="open">
<head>
  <title>{{ if .IsHome }}{{ site.Title }}{{ else }}{{ .Title }} - {{ site.Title }}{{ end }}</title>
  {{ partialCached "head.html" . "global" }}
</head>
<body class="{{ .Kind }}"{{ if eq .Kind "page" }} data-pagefind-body{{ end }}>
  <header>
    {{- if not .IsHome }}
    <button type="button" class="sidebar-toggle" aria-expanded="true" aria-controls="sidebar-left" aria-label="サイドバーの表示切り替え">☰</button>
    {{- end }}
    {{ partialCached "header.html" . "global" }}
  </header>
  {{- if .IsHome }}
  <main>
    {{ block "main" . }}{{ end }}
  </main>
  {{- else }}
  <aside class="sidebar-left" id="sidebar-left">
    <div class="sidebar-scroll">
      {{ block "sidebar-left" . }}{{ end }}
    </div>
    <div class="sidebar-resizer" role="separator" tabindex="0" aria-orientation="vertical"
         aria-controls="sidebar-left" aria-valuenow="240" aria-valuemin="160" aria-valuemax="480"></div>
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

- [ ] **Step 2: `main.css` のレイアウト部分を書き換える**

`assets/css/main.css` の先頭にある `body` の定義を、列幅が変数を参照するように変更する。ファイル冒頭（`/* レイアウト */` の直前）に変数定義を追加する:

```css
/* サイドバーの幅。--sidebar-width は JS がインラインで上書きする。
   --sidebar-current を経由させるのは、閉じた状態の 0px が
   インライン指定の --sidebar-width より確実に優先されるようにするため。 */
:root {
  --sidebar-width: 240px;
  --sidebar-current: var(--sidebar-width);
}
html[data-sidebar="collapsed"] { --sidebar-current: 0px; }
```

`body` の `grid-template` の列指定を `240px 1fr` から `var(--sidebar-current) 1fr` に変更する:

```css
body {
  display: grid;
  grid-template:
    "header header" auto
    "sidebar main" 1fr
    "footer footer" auto
    / var(--sidebar-current) 1fr;
  min-height: 100vh;
  margin: 0;
  font-family: sans-serif;
}
```

`.sidebar-left` の定義を、スクロールを内側に移す形に変更する（既存の `overflow-y: auto` と `padding: 1rem` を `.sidebar-scroll` へ移す）:

```css
.sidebar-left {
  grid-area: sidebar;
  position: relative;
  overflow: hidden;
  border-right: 1px solid #ddd;
}
html[data-sidebar="collapsed"] .sidebar-left { border-right: none; }
.sidebar-scroll {
  height: 100%;
  overflow-y: auto;
  padding: 1rem;
  box-sizing: border-box;
}
```

- [ ] **Step 3: 掴み手とトグルボタンのスタイルを追加する**

`main.css` の末尾に追加する:

```css
/* サイドバーの開閉・リサイズ操作
   JSが無い環境では操作できないため、既定では隠し、
   インラインスクリプトが html に js クラスを付けたときだけ表示する。 */
.sidebar-toggle,
.sidebar-resizer { display: none; }

html.js .sidebar-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  padding: 0;
  border: 1px solid #ddd;
  border-radius: 4px;
  background: #fff;
  color: #333;
  font-size: 1rem;
  line-height: 1;
  cursor: pointer;
}
html.js .sidebar-toggle:hover { background: #f0f0f0; }

html.js .sidebar-resizer {
  display: block;
  position: absolute;
  top: 0;
  right: 0;
  width: 6px;
  height: 100%;
  cursor: col-resize;
  background: transparent;
}
html.js .sidebar-resizer:hover,
html.js .sidebar-resizer:focus-visible { background: rgba(0, 0, 0, .12); }

/* 閉じている間は掴み手を出さない（tabindex は JS 側で -1 にする） */
html.js[data-sidebar="collapsed"] .sidebar-resizer { display: none; }

/* ドラッグ中は本文のテキスト選択を止める */
html.sidebar-dragging { cursor: col-resize; }
html.sidebar-dragging body { user-select: none; }
```

- [ ] **Step 4: ビルドして見た目が変わっていないことを確認する**

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud
hugo --source hugo-site --destination /tmp/hugo-t1 --quiet
```

Expected: エラーなし（出力なし）。

生成HTMLに新しい構造が入っていることを確認する:

```bash
grep -o 'data-sidebar="open"\|sidebar-scroll\|sidebar-resizer\|sidebar-toggle' /tmp/hugo-t1/sample/tree-deep/index.html | sort -u
```

Expected: 4つすべてが出力される。

ホームページにはトグルボタンが無いことを確認する:

```bash
grep -c 'sidebar-toggle' /tmp/hugo-t1/index.html
```

Expected: `0`

- [ ] **Step 5: 階層ツリーが壊れていないことを確認する**

```bash
grep -o '<details[^>]*>\|is-current\|tree-label">[^<]*' /tmp/hugo-t1/sample/tree-deep/index.html
```

Expected: `<details open>` が2つ、`is-current` が1つ、フォルダラベルが2つ（前回の実装と同じ結果）。

- [ ] **Step 6: コミット**

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud/hugo-site/themes/hugo-theme-docs
git add layouts/baseof.html assets/css/main.css
git commit -m "feat: サイドバーの幅を変数化し開閉・リサイズ用のDOM構造を追加

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 2: ちらつき防止のインラインスクリプトと sidebar.js の読み込み

**Files（submodule内）:**
- Modify: `layouts/_partials/head.html`

**Interfaces:**
- Consumes: Task 1の `data-sidebar` 属性、`--sidebar-width`
- Produces:
  - `window.sidebarConfig` = `{ min: 160, max: 480, default: 240, widthKey: 'sidebar-width', collapsedKey: 'sidebar-collapsed' }`
  - `<html>` に `js` クラスが付く
  - localStorageに保存済みの幅・開閉状態が初回描画前に反映される

このタスクの時点では `assets/js/sidebar.js` がまだ無い。Hugoの `resources.Get` は対象が無いと空を返し `with` の中が実行されないだけなのでビルドは通る。読み込みタグの追加を先にしておくことで、Task 3以降はJSファイルを置くだけで済む。

- [ ] **Step 1: `head.html` にインラインスクリプトと読み込みタグを追加する**

`layouts/_partials/head.html` を以下の内容にする（既存の内容は変えず、末尾に2ブロック追加する）:

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
<script>
  // 保存済みのサイドバー状態を初回描画の前に反映する。
  // 外部ファイルにすると読み込みを待つ間に既定幅で描画され、ちらつくためインラインで持つ。
  (function () {
    var config = {
      min: 160,
      max: 480,
      default: 240,
      widthKey: 'sidebar-width',
      collapsedKey: 'sidebar-collapsed'
    };
    window.sidebarConfig = config;
    var root = document.documentElement;
    root.classList.add('js');
    try {
      var width = parseInt(localStorage.getItem(config.widthKey), 10);
      // 不正な値を入れるとグリッドの列がコンテンツ幅で自動サイズになり崩れるため、必ず検証する
      if (!isNaN(width) && width >= config.min && width <= config.max) {
        root.style.setProperty('--sidebar-width', width + 'px');
      }
      if (localStorage.getItem(config.collapsedKey) === '1') {
        root.setAttribute('data-sidebar', 'collapsed');
      }
    } catch (e) {
      // localStorage が使えない環境（プライベートモード等）では既定値のまま表示する
    }
  })();
</script>
{{- with resources.Get "js/sidebar.js" }}
  {{- if hugo.IsDevelopment }}
    <script src="{{ .RelPermalink }}" defer></script>
  {{- else }}
    {{- with . | minify | fingerprint }}
      <script src="{{ .RelPermalink }}" integrity="{{ .Data.Integrity }}" crossorigin="anonymous" defer></script>
    {{- end }}
  {{- end }}
{{- end }}
```

- [ ] **Step 2: ビルドして `js` クラスが付くことを確認する**

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud
hugo --source hugo-site --destination /tmp/hugo-t2 --quiet
grep -c 'sidebarConfig' /tmp/hugo-t2/sample/tree-deep/index.html
```

Expected: `1`（インラインスクリプトが出力されている）

- [ ] **Step 3: ブラウザで初期状態を確認する**

配信して確認する（Playwrightは `file:` を開けないため、必ずHTTPで配信する）:

```bash
cd /tmp/hugo-t2 && python3 -m http.server 8899 &
sleep 2
```

Playwrightで `http://localhost:8899/sample/tree-deep/` を開き、以下を評価する:

```js
() => ({
  jsClass: document.documentElement.classList.contains('js'),
  dataSidebar: document.documentElement.getAttribute('data-sidebar'),
  width: Math.round(document.getElementById('sidebar-left').getBoundingClientRect().width),
  toggleVisible: getComputedStyle(document.querySelector('.sidebar-toggle')).display !== 'none',
  resizerVisible: getComputedStyle(document.querySelector('.sidebar-resizer')).display !== 'none'
})
```

Expected: `jsClass: true`、`dataSidebar: "open"`、`width: 240`、`toggleVisible: true`、`resizerVisible: true`

- [ ] **Step 4: 保存済みの値が復元されることを確認する**

Playwrightで以下を評価してからページを再読み込みし、幅が復元されることを確認する:

```js
() => { localStorage.setItem('sidebar-width', '320'); return localStorage.getItem('sidebar-width'); }
```

再読み込み後:

```js
() => Math.round(document.getElementById('sidebar-left').getBoundingClientRect().width)
```

Expected: `320`

さらに不正な値を入れて、既定値に落ちることを確認する:

```js
() => { localStorage.setItem('sidebar-width', 'abc'); return true; }
```

再読み込み後の幅が `240` であること（`abc` が適用されて崩れていないこと）を確認する。

確認後、`localStorage.clear()` を評価し、サーバーを停止する:

```bash
pkill -f "http.server 8899"
```

- [ ] **Step 5: コミット**

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud/hugo-site/themes/hugo-theme-docs
git add layouts/_partials/head.html
git commit -m "feat: サイドバー状態を初回描画前に復元するインラインスクリプトを追加

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 3: 開閉トグル

**Files（submodule内）:**
- Create: `assets/js/sidebar.js`

**Interfaces:**
- Consumes: Task 1のDOM要素、Task 2の `window.sidebarConfig`
- Produces: `assets/js/sidebar.js`。以降のタスクはこのファイルに機能を追加していく

- [ ] **Step 1: `sidebar.js` を作成する**

`assets/js/sidebar.js` を新規作成する:

```js
// サイドバーの開閉とリサイズ。
// 保存済み状態の復元は head 内のインラインスクリプトが担当しており、
// このファイルは利用者の操作を扱う。
(function () {
  var config = window.sidebarConfig || {
    min: 160, max: 480, default: 240,
    widthKey: 'sidebar-width', collapsedKey: 'sidebar-collapsed'
  };

  var root = document.documentElement;
  var aside = document.getElementById('sidebar-left');
  var toggle = document.querySelector('.sidebar-toggle');
  var resizer = document.querySelector('.sidebar-resizer');

  // ホームページなどサイドバーが無いページでは何もしない
  if (!aside || !toggle || !resizer) {
    return;
  }

  function store(key, value) {
    try {
      localStorage.setItem(key, value);
    } catch (e) {
      // 保存できない環境ではページ内の操作だけ有効にする
    }
  }

  function currentWidth() {
    var value = parseInt(getComputedStyle(root).getPropertyValue('--sidebar-width'), 10);
    return isNaN(value) ? config.default : value;
  }

  function isCollapsed() {
    return root.getAttribute('data-sidebar') === 'collapsed';
  }

  function setCollapsed(collapsed) {
    root.setAttribute('data-sidebar', collapsed ? 'collapsed' : 'open');
    toggle.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
    resizer.setAttribute('tabindex', collapsed ? '-1' : '0');
    store(config.collapsedKey, collapsed ? '1' : '0');
  }

  // 読み込み時点の状態を支援技術向けの属性に反映する（保存はしない）
  toggle.setAttribute('aria-expanded', isCollapsed() ? 'false' : 'true');
  resizer.setAttribute('tabindex', isCollapsed() ? '-1' : '0');
  resizer.setAttribute('aria-valuenow', String(currentWidth()));

  toggle.addEventListener('click', function () {
    setCollapsed(!isCollapsed());
  });
})();
```

- [ ] **Step 2: ビルドしてJSが読み込まれることを確認する**

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud
hugo --source hugo-site --destination /tmp/hugo-t3 --quiet
grep -o 'sidebar\.[a-f0-9]*\.js\|sidebar\.js' /tmp/hugo-t3/sample/tree-deep/index.html | head -1
```

Expected: `sidebar.<ハッシュ>.js` の形式で1件出力される（本番ビルドではfingerprintが付く）。

- [ ] **Step 3: ブラウザで開閉を確認する**

```bash
cd /tmp/hugo-t3 && python3 -m http.server 8899 &
sleep 2
```

Playwrightで `http://localhost:8899/sample/tree-deep/` を開き、トグルボタンをクリックしてから以下を評価する:

```js
() => ({
  dataSidebar: document.documentElement.getAttribute('data-sidebar'),
  ariaExpanded: document.querySelector('.sidebar-toggle').getAttribute('aria-expanded'),
  width: Math.round(document.getElementById('sidebar-left').getBoundingClientRect().width),
  resizerTabindex: document.querySelector('.sidebar-resizer').getAttribute('tabindex'),
  stored: localStorage.getItem('sidebar-collapsed')
})
```

Expected: `dataSidebar: "collapsed"`、`ariaExpanded: "false"`、`width: 0`、`resizerTabindex: "-1"`、`stored: "1"`

もう一度クリックして、`dataSidebar: "open"`、`ariaExpanded: "true"`、`width: 240`、`resizerTabindex: "0"`、`stored: "0"` に戻ることを確認する。

- [ ] **Step 4: ページを移動しても閉じたままであることを確認する**

閉じた状態にしてから、サイドバー内のリンク（`/sample/tree-child/`）へ遷移し、以下を評価する:

```js
() => ({
  dataSidebar: document.documentElement.getAttribute('data-sidebar'),
  width: Math.round(document.getElementById('sidebar-left').getBoundingClientRect().width)
})
```

Expected: `dataSidebar: "collapsed"`、`width: 0`

**注意**: 閉じているとサイドバーのリンクはクリックできないため、遷移はURLを直接開いて行うこと。

確認後 `localStorage.clear()` を評価し、サーバーを停止する。

- [ ] **Step 5: コミット**

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud/hugo-site/themes/hugo-theme-docs
git add assets/js/sidebar.js
git commit -m "feat: ヘッダーのボタンでサイドバーを開閉できるようにする

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 4: ドラッグでのリサイズ

**Files（submodule内）:**
- Modify: `assets/js/sidebar.js`

**Interfaces:**
- Consumes: Task 3の `store` / `currentWidth` / `isCollapsed`
- Produces: `clamp(width)` と `applyWidth(width)`。Task 5のキーボード操作が使う
  - `clamp(width)`: 160〜480に収めた数値を返す
  - `applyWidth(width)`: クランプして `--sidebar-width` に適用し、`aria-valuenow` を更新し、適用後の値を返す

- [ ] **Step 1: `sidebar.js` にクランプと適用の関数を追加する**

`currentWidth` 関数の直後に追加する:

```js
  function clamp(width) {
    return Math.min(config.max, Math.max(config.min, width));
  }

  // CSSの clamp() は範囲外の値を丸めてくれないことを実機で確認済みのため、
  // 幅の制限は必ずここで行う。
  function applyWidth(width) {
    var value = clamp(Math.round(width));
    root.style.setProperty('--sidebar-width', value + 'px');
    resizer.setAttribute('aria-valuenow', String(value));
    return value;
  }
```

- [ ] **Step 2: ドラッグの処理を追加する**

`toggle.addEventListener('click', ...)` の直後に追加する:

```js
  var dragging = false;

  resizer.addEventListener('pointerdown', function (event) {
    if (isCollapsed()) {
      return;
    }
    dragging = true;
    resizer.setPointerCapture(event.pointerId);
    root.classList.add('sidebar-dragging');
    event.preventDefault();
  });

  resizer.addEventListener('pointermove', function (event) {
    if (!dragging) {
      return;
    }
    // ビューポート左端ではなくサイドバー左端からの距離を使う
    // （将来レイアウトに余白が入ってもずれないようにするため）
    applyWidth(event.clientX - aside.getBoundingClientRect().left);
  });

  function endDrag(event) {
    if (!dragging) {
      return;
    }
    dragging = false;
    try {
      resizer.releasePointerCapture(event.pointerId);
    } catch (e) {
      // ポインタが既に解放されている場合は無視する
    }
    root.classList.remove('sidebar-dragging');
    // 保存はドラッグ終了時だけ行う（移動中に書き込むと負荷が高いため）
    store(config.widthKey, String(currentWidth()));
  }

  resizer.addEventListener('pointerup', endDrag);
  resizer.addEventListener('pointercancel', endDrag);
```

- [ ] **Step 3: ビルドしてブラウザでドラッグを確認する**

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud
hugo --source hugo-site --destination /tmp/hugo-t4 --quiet
cd /tmp/hugo-t4 && python3 -m http.server 8899 &
sleep 2
```

Playwrightで `http://localhost:8899/sample/tree-deep/` を開き、`browser_drag` で `.sidebar-resizer` を右方向へドラッグする。ドラッグ後に以下を評価する:

```js
() => ({
  width: Math.round(document.getElementById('sidebar-left').getBoundingClientRect().width),
  ariaValueNow: document.querySelector('.sidebar-resizer').getAttribute('aria-valuenow'),
  stored: localStorage.getItem('sidebar-width')
})
```

Expected: 幅が240pxから変化しており、`ariaValueNow` と `stored` がその幅と一致すること。

- [ ] **Step 4: 上下限でクランプされることを確認する**

`browser_drag` ではクランプの境界を正確に狙いにくいため、`pointermove` を直接発火させて確認する:

```js
() => {
  var resizer = document.querySelector('.sidebar-resizer');
  var aside = document.getElementById('sidebar-left');
  var left = aside.getBoundingClientRect().left;
  function drag(x) {
    resizer.dispatchEvent(new PointerEvent('pointerdown', { pointerId: 1, clientX: left + 240, bubbles: true }));
    resizer.dispatchEvent(new PointerEvent('pointermove', { pointerId: 1, clientX: left + x, bubbles: true }));
    resizer.dispatchEvent(new PointerEvent('pointerup',   { pointerId: 1, clientX: left + x, bubbles: true }));
    return Math.round(aside.getBoundingClientRect().width);
  }
  return { 下限未満: drag(20), 上限超: drag(2000), 範囲内: drag(300) };
}
```

Expected: `下限未満: 160`、`上限超: 480`、`範囲内: 300`

- [ ] **Step 5: ページを移動しても幅が保たれることを確認する**

幅を300pxにした状態で `/sample/tree-child/` へ遷移し、幅が `300` であることを確認する。

確認後 `localStorage.clear()` を評価し、サーバーを停止する。

- [ ] **Step 6: コミット**

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud/hugo-site/themes/hugo-theme-docs
git add assets/js/sidebar.js
git commit -m "feat: 境界のドラッグでサイドバーの幅を変更できるようにする

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 5: キーボード操作とダブルクリックでのリセット

**Files（submodule内）:**
- Modify: `assets/js/sidebar.js`

**Interfaces:**
- Consumes: Task 4の `applyWidth` / `currentWidth`、Task 3の `store`

- [ ] **Step 1: キーボード操作とダブルクリックの処理を追加する**

`resizer.addEventListener('pointercancel', endDrag);` の直後に追加する:

```js
  var KEYBOARD_STEP = 16;

  resizer.addEventListener('keydown', function (event) {
    var width = currentWidth();
    if (event.key === 'ArrowLeft') {
      applyWidth(width - KEYBOARD_STEP);
    } else if (event.key === 'ArrowRight') {
      applyWidth(width + KEYBOARD_STEP);
    } else if (event.key === 'Home') {
      applyWidth(config.default);
    } else {
      return;
    }
    event.preventDefault();
    store(config.widthKey, String(currentWidth()));
  });

  resizer.addEventListener('dblclick', function () {
    applyWidth(config.default);
    store(config.widthKey, String(currentWidth()));
  });
```

- [ ] **Step 2: ビルドしてキーボード操作を確認する**

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud
hugo --source hugo-site --destination /tmp/hugo-t5 --quiet
cd /tmp/hugo-t5 && python3 -m http.server 8899 &
sleep 2
```

Playwrightで `http://localhost:8899/sample/tree-deep/` を開き、掴み手にフォーカスしてからキーを送る:

```js
() => { document.querySelector('.sidebar-resizer').focus(); return document.activeElement.className; }
```

Expected: `sidebar-resizer`（Tabでも到達できることの確認になる）

`browser_press_key` で `ArrowRight` を3回押してから:

```js
() => ({
  width: Math.round(document.getElementById('sidebar-left').getBoundingClientRect().width),
  stored: localStorage.getItem('sidebar-width')
})
```

Expected: `width: 288`（240 + 16×3）、`stored: "288"`

`Home` を押して `width: 240` に戻ることを確認する。

- [ ] **Step 3: ダブルクリックでのリセットを確認する**

幅を変えてから掴み手をダブルクリックし、`240` に戻ることを確認する:

```js
() => {
  document.documentElement.style.setProperty('--sidebar-width', '400px');
  document.querySelector('.sidebar-resizer').dispatchEvent(new MouseEvent('dblclick', { bubbles: true }));
  return Math.round(document.getElementById('sidebar-left').getBoundingClientRect().width);
}
```

Expected: `240`

確認後 `localStorage.clear()` を評価し、サーバーを停止する。

- [ ] **Step 4: コミット**

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud/hugo-site/themes/hugo-theme-docs
git add assets/js/sidebar.js
git commit -m "feat: キーボードとダブルクリックでサイドバーの幅を操作できるようにする

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 6: 総合検証（ちらつき・非回帰・性能）

**Files:** なし（検証のみ。問題が見つかった場合のみ修正する）

このタスクは新しい機能を足さない。スペックの「テスト」節に挙げた観点を通しで確認する。

- [ ] **Step 1: ちらつきが起きないことを確認する**

これは実装の要なので必ず確認する。幅を保存した状態でページを読み込み、**最初の描画時点から**その幅になっていることを見る。

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud
hugo --source hugo-site --destination /tmp/hugo-t6 --quiet
cd /tmp/hugo-t6 && python3 -m http.server 8899 &
sleep 2
```

Playwrightで開き、`localStorage.setItem('sidebar-width', '400')` を評価してから再読み込みする。読み込み直後に以下を評価する:

```js
() => {
  // DOMContentLoaded より前に確定しているべき値
  var cs = getComputedStyle(document.documentElement).getPropertyValue('--sidebar-width').trim();
  return { inlineStyle: document.documentElement.style.getPropertyValue('--sidebar-width'), computed: cs };
}
```

Expected: `inlineStyle: "400px"`（インラインスクリプトが設定済み）、`computed: "400px"`

さらにスクリーンショットを撮り、サイドバーが400pxで表示されていることを目視で確認する。

- [ ] **Step 2: JavaScriptを無効にした場合を確認する**

Playwrightで新しいコンテキストをJavaScript無効にして同じページを開き、以下をHTMLから確認する:

```bash
curl -s http://localhost:8899/sample/tree-deep/ | grep -o 'class="sidebar-toggle"\|class="sidebar-resizer"' | sort -u
```

Expected: 両方ともHTMLには存在する（CSSで隠れているだけ）。

JavaScript無効のブラウザで開いたとき、トグルボタンと掴み手が**表示されていない**こと、サイドバーが240pxで開いていることをスクリーンショットで確認する。

- [ ] **Step 3: 階層ツリーの非回帰を確認する**

```bash
grep -o '<details[^>]*>\|is-current\|tree-label">[^<]*' /tmp/hugo-t6/sample/tree-deep/index.html
grep -c '<details open>' /tmp/hugo-t6/sample/tree-home/index.html
ls /tmp/hugo-t6/sample/
```

Expected: `tree-deep` では `<details open>` が2つ・`is-current` が1つ・フォルダラベルが2つ。`tree-home` では `<details open>` が `0`。`ls` の結果に `tree-folder` と `tree-subfolder` が**無い**こと（フォルダのHTMLは生成されない）。

- [ ] **Step 4: 性能が悪化していないことを確認する**

前回の階層ツリー実装で400ページのビルドを0.87秒まで短縮している。その水準を保てているかを見る。

400ページの合成コンテンツを生成する:

```bash
BIG=/tmp/sidebar-perf-content
rm -rf $BIG && mkdir -p $BIG/space
python3 - "$BIG" <<'PY'
import os, sys
base = sys.argv[1] + "/space"
for i in range(400):
    parent = "" if i < 3 else f"p{(i//3-1):04d}"
    d = os.path.join(base, f"page-{i:04d}")
    os.makedirs(d, exist_ok=True)
    fm = [f'title = "ページ{i:04d}"', f'page_id = "p{i:04d}"', f'weight = {i%7+1}']
    if parent:
        fm.append(f'parent_id = "{parent}"')
    open(os.path.join(d, "index.md"), "w").write("+++\n" + "\n".join(fm) + "\n+++\n\n本文\n")
print("400ページ生成")
PY
time hugo --source /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud/hugo-site --contentDir "$BIG" --destination /tmp/hugo-perf-t6 --quiet
```

Expected: 2秒以内（前回0.87秒。本タスクの変更はテンプレートのループを増やさないため、悪化しないはず）。2秒を超えた場合は原因を調べて報告すること。

- [ ] **Step 5: 実データを含むビルドが通ることを確認する**

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud
hugo --source hugo-site --destination /tmp/hugo-t6-real --quiet
```

Expected: エラーなし。

サーバーを停止する:

```bash
pkill -f "http.server 8899"
```

- [ ] **Step 6: 検証結果を記録する**

このタスクではコミットする変更が無い場合もある。その場合はコミットせず、確認した項目と結果（スクリーンショットを含む）を報告に残すこと。Step 1〜5のいずれかで問題が見つかった場合は修正し、修正内容をコミットする。

---

### Task 7: ドキュメント更新・submodule反映・PR作成

**Files:**
- Modify: `TODO.md`、`CHANGELOG.md`（親リポジトリ）
- Modify: submoduleポインタ `hugo-site/themes/hugo-theme-docs`

- [ ] **Step 1: TODO.md と CHANGELOG.md を更新する**

両ファイルの既存の書き方（見出しの階層、箇条書きのスタイル）に合わせて追記する。記載する内容:

- 左サイドバーを境界のドラッグで左右にリサイズできるようにした（160px〜480px、既定240px）
- ヘッダーのボタンでサイドバーを開閉できるようにした
- 幅と開閉状態を localStorage に保存し、ページを移動しても次回訪問時も維持されるようにした
- 掴み手はキーボードでも操作できる（Tabで到達し、左右キーで16pxずつ、`Home` で既定値に戻る）。ダブルクリックでも既定値に戻る
- JavaScriptが無効な環境では、サイドバーは既定の240pxで開いたまま表示され、操作用のボタンと掴み手は表示されない
- 対象はPCの画面幅のみ。モバイル向けのオーバーレイ表示は含まない

- [ ] **Step 2: submoduleをpushしてPRを作成する**

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud/hugo-site/themes/hugo-theme-docs
git push -u origin feature/sidebar-resize-collapse
gh pr create --title "feat: サイドバーのリサイズと開閉に対応する" --body "$(cat <<'EOF'
## 概要

左サイドバーを境界のドラッグで左右にリサイズでき、ヘッダーのボタンで開閉できるようにしました。状態は localStorage に保存され、ページを移動しても次回訪問時も維持されます。

## 変更内容

- `layouts/baseof.html`: `<html>` に `data-sidebar` を追加。`aside` をスクロール層（`.sidebar-scroll`）と掴み手（`.sidebar-resizer`）に分割。ヘッダーにトグルボタンを追加
- `layouts/_partials/head.html`: 保存済み状態を初回描画前に反映するインラインスクリプトと、`sidebar.js` の読み込みを追加
- `assets/js/sidebar.js`（新規）: ドラッグ・トグル・キーボード操作
- `assets/css/main.css`: 列幅の変数化、掴み手とトグルボタンのスタイル

## 設計上の判断

**幅は `--sidebar-width` と `--sidebar-current` の2段構えにしています。** JSは `--sidebar-width` をインラインスタイルで設定しますが、インラインスタイルは通常のCSSルールより優先されるため、閉じた状態を表現するには経由用の変数が必要でした。`html[data-sidebar="collapsed"]` が `--sidebar-current: 0px` を上書きする形にしています。

**幅のクランプはJSで行っています。** CSSの `clamp()` を使う案も試しましたが、WebKitでは範囲外の値がそのまま通り機能しませんでした。また不正な値が入るとグリッドの列がコンテンツ幅で自動サイズになり崩れるため、localStorageから読んだ値は必ず検証しています。

**トグルボタンは `header.html` ではなく `baseof.html` に置いています。** ヘッダーは `partialCached` でキャッシュキー `"global"` により全ページ共通で使い回されるため、ホームページだけボタンを出し分ける処理を `header.html` に入れると破綻します。

## アクセシビリティ

- 掴み手は `role="separator"`、`aria-orientation="vertical"`、`aria-valuenow` / `aria-valuemin` / `aria-valuemax` を持ち、Tabで到達できます
- トグルボタンは `aria-expanded` と `aria-controls` を持ちます
- JavaScriptが無効な環境では、動かない操作要素を見せないよう両方とも非表示にしています

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 3: 親リポジトリでsubmoduleポインタを更新してコミットする**

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud
git add hugo-site/themes/hugo-theme-docs TODO.md CHANGELOG.md
git commit -m "chore: テーマsubmoduleをサイドバーのリサイズ・開閉対応に更新

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

- [ ] **Step 4: 親リポジトリをpushしてPRを作成する**

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud
git push -u origin feature/sidebar-resize-collapse
gh pr create --title "feat: 左サイドバーをリサイズ・開閉できるようにする" --body "$(cat <<'EOF'
## 概要

左サイドバーを境界のドラッグで左右にリサイズでき、ヘッダーのボタンで開閉できるようにしました。幅と開閉状態は localStorage に保存され、ページを移動しても次回訪問時も維持されます。

- 設計: `docs/superpowers/specs/2026-08-11-sidebar-resize-collapse-design.md`
- 実装計画: `docs/superpowers/plans/2026-08-11-sidebar-resize-collapse.md`
- テーマ側の PR: （テーマPRのURLをここに記入する）**先にこちらをマージしてください**

## 変更内容

実装はすべてテーマ（submodule）側です。親リポジトリの変更はsubmoduleポインタの更新とドキュメントのみです。

- 境界のドラッグで幅を変更（160px〜480px、既定240px）
- ヘッダーのボタンで開閉
- 状態を localStorage に保存
- 掴み手はキーボードでも操作可能（Tab → 左右キーで16pxずつ、`Home` で既定値）。ダブルクリックでも既定値に戻る

## 制限事項

- 対象はPCの画面幅のみです。モバイル向けのオーバーレイ表示は含みません
- JavaScriptが無効な環境では、サイドバーは既定の240pxで開いたまま表示され、操作用のボタンと掴み手は表示されません

## 検証

Playwrightで実ブラウザ操作を確認しています。ドラッグでの幅変更、上下限でのクランプ、開閉、ページ移動をまたいだ状態の保持、読み込み時にちらつかないこと、キーボード操作。あわせて階層ツリー表示の非回帰と、400ページのビルド時間が悪化していないことも確認しています。

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

**注意**: Step 4のPR本文にはテーマPRのURLを記入する箇所がある。Step 2で作成したPRのURLに置き換えること。

---

## Self-Review

**スペック網羅性チェック**

| スペックの要求 | 対応タスク |
|---|---|
| `--sidebar-width` / `--sidebar-current` / `data-sidebar` / `js` クラスによる状態表現 | Task 1（CSS）、Task 2（`js` クラス） |
| 閉じたとき列幅0・`overflow: hidden`・`display: none` にしない | Task 1 |
| DOM構造（`.sidebar-scroll` と `.sidebar-resizer` への分割） | Task 1 |
| ヘッダーのトグルボタン、ホームでは出力しない | Task 1 |
| `<head>` のインラインスクリプトによるちらつき防止 | Task 2、検証は Task 6 Step 1 |
| `resources.Get` → `minify` → `fingerprint` でJSを扱う | Task 2 |
| ドラッグ（`setPointerCapture`、サイドバー左端からの距離、終了時のみ保存） | Task 4 |
| ドラッグ中のテキスト選択抑止 | Task 1（CSS）、Task 4（クラス付与） |
| 幅の範囲160〜480のクランプ | Task 4（`clamp` 関数） |
| ダブルクリック・`Home` での既定値リセット | Task 5 |
| キーボードでの16pxずつの変更 | Task 5 |
| 閉じている間は掴み手を非表示・Tab対象外 | Task 1（CSS）、Task 3（`tabindex`） |
| 開閉時の `aria-expanded` 更新、幅変更時の `aria-valuenow` 更新 | Task 3、Task 4 |
| localStorageの2キーと値の検証 | Task 2（読み出し時の検証）、Task 3・4・5（書き込み） |
| localStorageが使えない場合に機能を止めない | Task 2（インライン側）、Task 3（`store` 関数） |
| JavaScript無効時の挙動 | Task 1（CSS）、検証は Task 6 Step 2 |
| ビルドが通ること | 各タスク |
| 階層ツリーの非回帰 | Task 1 Step 5、Task 6 Step 3 |
| 400ページでの性能非回帰 | Task 6 Step 4 |
| ブラウザ操作の確認（ドラッグ・クランプ・開閉・状態保持・ちらつき・キーボード） | Task 3〜5の各Step、Task 6 Step 1 |

スペックの要求で対応タスクが無いものは見つからなかった。

**プレースホルダ確認**: 「適切に処理する」等の曖昧な指示は無し。全コードステップに実コードを記載済み。Task 7 Step 1（TODO/CHANGELOG）は既存ファイルの記法に合わせる必要があるため記載内容を箇条書きで指定し、Step 4のPR本文にはテーマPRのURLを差し込む箇所を明示した。

**型・名前の整合性**:
- `window.sidebarConfig` のキー（`min`/`max`/`default`/`widthKey`/`collapsedKey`）はTask 2で定義しTask 3・4・5で同名で使用
- `store` / `currentWidth` / `isCollapsed` はTask 3で定義し、Task 4・5で使用
- `clamp` / `applyWidth` はTask 4で定義し、Task 5で使用
- CSSクラス名（`sidebar-scroll` / `sidebar-resizer` / `sidebar-toggle` / `js` / `sidebar-dragging`）はTask 1のCSSとHTML、Task 3・4のJSで一致
- localStorageのキー名（`sidebar-width` / `sidebar-collapsed`）はTask 2の定義とTask 3〜5の検証コマンドで一致
- `data-sidebar` の値（`open` / `collapsed`）は全タスクで一致
