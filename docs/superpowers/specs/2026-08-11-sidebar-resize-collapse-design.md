# 左サイドバーのリサイズ・開閉 設計

日付: 2026-08-11
ステータス: 設計承認済み

## 背景と目的

Hugoサイトの左サイドバーは幅240pxで固定されており、閉じることもできない。ページタイトルが長いと折り返して読みにくく、逆に本文を広く使いたいときにも場所を取り続ける。サイドバーを左右にドラッグしてリサイズでき、不要なときは閉じられるようにする。

前提となる実装: [2026-08-08-sidebar-hierarchy-design.md](2026-08-08-sidebar-hierarchy-design.md)（階層ツリー表示）。本設計はその「入れ物」側の話であり、ツリーの中身には手を入れない。

## 要件（確定事項）

- **対象はPCの画面幅のみ**。モバイル向けのオーバーレイ表示は本設計の対象外とする
- **閉じたときは完全に隠す**。再度開くための操作はヘッダーのトグルボタンで行う
- **状態はlocalStorageに永続化する**。ページを移動しても、次回訪問時も維持される
- **リサイズは境界のドラッグで行う**。CSSの `resize: horizontal` は使わない（掴み手が右下隅の三角に固定され、境界をドラッグする操作感にならないため）

## アーキテクチャ

状態は `<html>` 要素に持たせ、CSSがそれを参照してレイアウトを決める。

| 状態 | 表現 | 既定値 |
|---|---|---|
| サイドバーの幅 | CSSカスタムプロパティ `--sidebar-width` | `240px` |
| 開閉 | 属性 `data-sidebar`（`open` / `collapsed`） | `open` |
| JS有効 | クラス `js` | なし（JSが付与する） |

```css
:root { --sidebar-width: 240px; }
body { grid-template-columns: var(--sidebar-width) 1fr; }
html[data-sidebar="collapsed"] body { grid-template-columns: 0 1fr; }
```

閉じた状態では列幅が0になり、`aside` は `overflow: hidden` で中身がはみ出さない。`display: none` にしないのは、将来アニメーションを付ける余地を残すためと、開くたびの再構築を避けるため。

JavaScriptは2箇所に分かれる。

1. **`<head>` 内のインラインスクリプト**（10行程度）: localStorageの値を初回描画の**前**に `<html>` へ反映し、ちらつきを防ぐ。あわせて `js` クラスを付ける。外部ファイルにはできない（読み込みを待つ間に既定幅で描画されてしまうため）
2. **`assets/js/sidebar.js`**（`defer` で読み込み）: ドラッグとトグルの操作を担当する

テーマが自前のJavaScriptを持つのは今回が初めてなので、既存のCSSと同じく `resources.Get` → `minify` → `fingerprint` の流れで扱う。

## DOM構造の変更

掴み手を右端に固定するため、スクロールを内側の要素に移す。

```html
<aside class="sidebar-left" id="sidebar-left">
  <div class="sidebar-scroll"><!-- 既存のページ一覧 --></div>
  <div class="sidebar-resizer" role="separator" tabindex="0"
       aria-orientation="vertical" aria-controls="sidebar-left"
       aria-valuenow="240" aria-valuemin="160" aria-valuemax="480"></div>
</aside>
```

- `aside`: `position: relative`、`overflow: hidden`
- `.sidebar-scroll`: 縦スクロールを担当（従来 `aside` にあった `overflow-y: auto` はここへ移す）
- `.sidebar-resizer`: 右端に絶対配置した幅6pxの帯。カーソルは `col-resize`

ヘッダーの左端にトグルボタンを追加する。

```html
<button class="sidebar-toggle" aria-expanded="true" aria-controls="sidebar-left">☰</button>
```

サイドバーが存在しないホームページでは、トグルボタンも掴み手も出力しない（`baseof.html` が既にホームを分岐しているので、その分岐に合わせる）。

## リサイズの挙動

- **ドラッグ**: `pointerdown` で `setPointerCapture` し、以降の `pointermove` を取りこぼさないようにする。新しい幅は `event.clientX - aside.getBoundingClientRect().left` で求める（ビューポート左端からの距離をそのまま使うと、将来レイアウトに余白が入ったときにずれるため）。求めた値をクランプして `--sidebar-width` に書き込む。`pointerup` で localStorage に保存する。**保存はドラッグ終了時のみ**行い、移動中は書き込まない
- **ドラッグ中**: `<html>` にクラスを付け、本文のテキスト選択を止める（`user-select: none`）
- **幅の範囲**: **160px〜480px**にクランプする
  - 下限160px: 日本語のページタイトルが最低限読める幅
  - 上限480px: 本文が狭くなりすぎない範囲の目安
  - この2つの値はCSSカスタムプロパティとして1箇所にまとめ、JS側もそこから読む
- **既定値へのリセット**: 掴み手のダブルクリック、または `Home` キー
- **キーボード操作**: 掴み手はTabで到達でき、左右キーで16pxずつ変更する
- **閉じている間**: 掴み手は非表示にし、`tabindex="-1"` でTabの到達対象から外す

## 開閉の挙動

トグルボタンを押すたびに `data-sidebar` が `open` と `collapsed` を行き来し、`aria-expanded` も連動して更新する。状態は localStorage に保存する。

閉じても幅の値は保持し、開いたときは閉じる前の幅に戻る。

## 永続化

localStorageのキーは2つ。

| キー | 値 | 検証 |
|---|---|---|
| `sidebar-width` | 幅のピクセル数（文字列） | 数値として解釈でき、160〜480に収まること。外れていれば既定値240を使う |
| `sidebar-collapsed` | `"1"` / `"0"` | `"1"` のときだけ閉じた状態として扱う |

読み出した値をそのまま信用しない。壊れた値や範囲外の値で表示が崩れないようにするため、必ず検証してから適用する。

localStorageが使えない環境（プライベートモード、ストレージ無効化など）では例外を握りつぶし、**そのページ内では操作できるが保存はされない**という挙動にする。機能全体が止まってはいけない。

## JavaScriptが無効な場合

サイドバーは既定の240pxで開いたまま表示される。動かないボタンを見せないよう、トグルボタンと掴み手は**既定で非表示**にしておき、インラインスクリプトが `<html>` に `js` クラスを付けたときだけ表示する。

```css
.sidebar-toggle, .sidebar-resizer { display: none; }
html.js .sidebar-toggle { display: inline-flex; }
html.js .sidebar-resizer { display: block; }
/* 閉じている間は掴み手を出さない（JSは tabindex="-1" も併せて設定する） */
html.js[data-sidebar="collapsed"] .sidebar-resizer { display: none; }
```

## アクセシビリティ

- 掴み手: `role="separator"`、`aria-orientation="vertical"`、`aria-valuenow` / `aria-valuemin` / `aria-valuemax`。幅の変更時に `aria-valuenow` を更新する
- トグルボタン: `aria-expanded` と `aria-controls`。状態変化時に `aria-expanded` を更新する
- どちらもキーボードだけで操作できること

## テスト

- **ビルド**: Hugoのビルドがエラーなく通ること
- **既存機能の非回帰**: `content/sample` の階層フィクスチャで、ツリー表示（階層のネスト、祖先の自動展開、現在ページのハイライト、フォルダの非リンク化）が壊れていないこと
- **性能の非回帰**: 400ページの合成コンテンツでビルド時間が悪化していないこと。前回の最適化で0.87秒まで短縮しているので、その水準を維持できているかを確認する
- **ブラウザ操作**（Playwrightを使用。スクリーンショットも取得する）
  - ドラッグで幅が変わること
  - 160px未満・480px超にドラッグしても、その範囲にクランプされること
  - トグルボタンで開閉できること
  - ページを移動しても幅と開閉状態が保たれること
  - 読み込み時に既定幅で一瞬描画される「ちらつき」が起きないこと
  - キーボード（Tab → 左右キー、`Home`）で幅を変更できること

## 検討した代替案

- **CSSの `resize: horizontal`**: ブラウザ標準の機能でドラッグ処理を書かずに済むが、掴み手が右下隅の小さな三角に固定され、境界をドラッグする操作感にならない。`overflow` の指定にも制約が付き、幅の保存には結局ResizeObserverが必要でJSの節約幅も小さいため不採用
- **幅を数段階のプリセットから選ぶ**: 実装は最も単純だが、「左右にリサイズする」という要件から外れるため不採用
- **モバイル向けオーバーレイ表示**: 現在のテーマにレスポンシブ対応が無く、本設計に含めると実装量が大きく増える。PCでの作業性改善に絞るため対象外とした（将来的に別設計として扱う）
