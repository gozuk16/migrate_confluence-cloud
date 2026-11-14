# Confluence Cloud Storage Format (XHTML) 完全リファレンス

**調査日**: 2026-02-25
**対象**: Confluence Cloud REST API v2 Storage Format
**目的**: Confluence XHTML から Markdown へのコンバーター構築のための要素リファレンス

---

## 調査サマリー

### スコープと目的
Confluence Cloud の REST API v2 で取得できる Storage Format (XHTML) に含まれる全要素を網羅的に調査し、各要素の XML 構造と理想的な Markdown 変換先を定義する。

### キーファインディング
- Storage Format は厳密には XHTML ではなく XML (カスタム名前空間 `ac:`, `ri:` を含む)
- Confluence Cloud では `ac:structured-macro` が使用される (旧形式 `ac:macro` はレガシー)
- Cloud 版では `ri:user` の属性が `ri:username` から `ri:account-id` に変更されている (GDPR 対応)
- マクロのボディは3種類: `ac:rich-text-body` (XHTML), `ac:plain-text-body` (CDATA), ボディなし
- `ac:layout` 系要素はマクロベースのレイアウト (`section`/`column`) とは別系統

---

## 1. ac:structured-macro 一覧

Confluence Cloud では全てのマクロが `<ac:structured-macro ac:name="マクロ名" ac:schema-version="1" ac:macro-id="UUID">` の形式で格納される。

### 共通構造

```xml
<ac:structured-macro ac:name="マクロ名" ac:schema-version="1" ac:macro-id="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx">
  <ac:parameter ac:name="パラメータ名">値</ac:parameter>
  <!-- ボディ (3種類のいずれか、またはなし) -->
  <ac:rich-text-body>XHTML コンテンツ</ac:rich-text-body>
  <ac:plain-text-body><![CDATA[プレーンテキスト]]></ac:plain-text-body>
</ac:structured-macro>
```

### 1.1 通知・情報系マクロ

#### info (情報パネル)
```xml
<ac:structured-macro ac:name="info">
  <ac:parameter ac:name="title">タイトル</ac:parameter>
  <ac:parameter ac:name="icon">false</ac:parameter>
  <ac:rich-text-body>
    <p>情報テキスト</p>
  </ac:rich-text-body>
</ac:structured-macro>
```
**Markdown 変換:**
```markdown
> **Info: タイトル**
> 情報テキスト
```
または GitHub スタイル:
```markdown
> [!NOTE]
> **タイトル**
> 情報テキスト
```

#### note (注意パネル)
```xml
<ac:structured-macro ac:name="note">
  <ac:parameter ac:name="title">タイトル</ac:parameter>
  <ac:rich-text-body>
    <p>注意テキスト</p>
  </ac:rich-text-body>
</ac:structured-macro>
```
**Markdown 変換:**
```markdown
> [!WARNING]
> **タイトル**
> 注意テキスト
```

#### warning (警告パネル)
```xml
<ac:structured-macro ac:name="warning">
  <ac:parameter ac:name="title">タイトル</ac:parameter>
  <ac:rich-text-body>
    <p>警告テキスト</p>
  </ac:rich-text-body>
</ac:structured-macro>
```
**Markdown 変換:**
```markdown
> [!CAUTION]
> **タイトル**
> 警告テキスト
```

#### tip (ヒントパネル)
```xml
<ac:structured-macro ac:name="tip">
  <ac:parameter ac:name="title">タイトル</ac:parameter>
  <ac:rich-text-body>
    <p>ヒントテキスト</p>
  </ac:rich-text-body>
</ac:structured-macro>
```
**Markdown 変換:**
```markdown
> [!TIP]
> **タイトル**
> ヒントテキスト
```

### 1.2 コード・整形済みテキスト系マクロ

#### code (コードブロック)
```xml
<ac:structured-macro ac:name="code">
  <ac:parameter ac:name="language">java</ac:parameter>
  <ac:parameter ac:name="title">サンプルコード</ac:parameter>
  <ac:parameter ac:name="linenumbers">true</ac:parameter>
  <ac:parameter ac:name="theme">Emacs</ac:parameter>
  <ac:parameter ac:name="firstline">1</ac:parameter>
  <ac:parameter ac:name="collapse">false</ac:parameter>
  <ac:plain-text-body><![CDATA[public class Hello {
    public static void main(String[] args) {
        System.out.println("Hello");
    }
}]]></ac:plain-text-body>
</ac:structured-macro>
```
**パラメータ:** `language`, `title`, `linenumbers`, `theme`, `firstline`, `collapse`
**Markdown 変換:**
````markdown
```java
public class Hello {
    public static void main(String[] args) {
        System.out.println("Hello");
    }
}
```
````

#### noformat (整形なしテキスト)
```xml
<ac:structured-macro ac:name="noformat">
  <ac:parameter ac:name="nopanel">true</ac:parameter>
  <ac:plain-text-body><![CDATA[整形なしテキスト]]></ac:plain-text-body>
</ac:structured-macro>
```
**パラメータ:** `nopanel`
**Markdown 変換:**
````markdown
```
整形なしテキスト
```
````

### 1.3 レイアウト・構造系マクロ

#### panel (パネル)
```xml
<ac:structured-macro ac:name="panel">
  <ac:parameter ac:name="title">パネルタイトル</ac:parameter>
  <ac:parameter ac:name="bgColor">#ffffff</ac:parameter>
  <ac:parameter ac:name="titleBGColor">#cccccc</ac:parameter>
  <ac:parameter ac:name="borderStyle">solid</ac:parameter>
  <ac:parameter ac:name="borderColor">#cccccc</ac:parameter>
  <ac:parameter ac:name="titleColor">#000000</ac:parameter>
  <ac:rich-text-body>
    <p>パネル内容</p>
  </ac:rich-text-body>
</ac:structured-macro>
```
**パラメータ:** `title`, `bgColor`, `titleBGColor`, `borderStyle`, `borderColor`, `titleColor`
**Markdown 変換:**
```markdown
> **パネルタイトル**
>
> パネル内容
```

#### expand (折りたたみ/展開)
```xml
<ac:structured-macro ac:name="expand">
  <ac:parameter ac:name="title">クリックして展開</ac:parameter>
  <ac:rich-text-body>
    <p>折りたたまれた内容</p>
  </ac:rich-text-body>
</ac:structured-macro>
```
**パラメータ:** `title` (default-parameter として設定される場合もある)
**Markdown 変換:**
```markdown
<details>
<summary>クリックして展開</summary>

折りたたまれた内容

</details>
```

#### section (セクション)
```xml
<ac:structured-macro ac:name="section">
  <ac:parameter ac:name="border">true</ac:parameter>
  <ac:rich-text-body>
    <ac:structured-macro ac:name="column">
      <ac:parameter ac:name="width">50%</ac:parameter>
      <ac:rich-text-body>
        <p>左カラム</p>
      </ac:rich-text-body>
    </ac:structured-macro>
    <ac:structured-macro ac:name="column">
      <ac:parameter ac:name="width">50%</ac:parameter>
      <ac:rich-text-body>
        <p>右カラム</p>
      </ac:rich-text-body>
    </ac:structured-macro>
  </ac:rich-text-body>
</ac:structured-macro>
```
**Markdown 変換:** (Markdown ではカラムレイアウト不可、順次展開)
```markdown
左カラム

右カラム
```
または HTML テーブルで代替:
```markdown
| 左カラム | 右カラム |
|----------|----------|
| 内容     | 内容     |
```

#### column (カラム) -- section 内で使用
```xml
<ac:structured-macro ac:name="column">
  <ac:parameter ac:name="width">50%</ac:parameter>
  <ac:rich-text-body>
    <p>カラム内容</p>
  </ac:rich-text-body>
</ac:structured-macro>
```
**パラメータ:** `width`

#### quote (引用)
```xml
<ac:structured-macro ac:name="quote">
  <ac:rich-text-body>
    <p>引用テキスト</p>
  </ac:rich-text-body>
</ac:structured-macro>
```
**Markdown 変換:**
```markdown
> 引用テキスト
```

### 1.4 目次・ナビゲーション系マクロ

#### toc (目次)
```xml
<ac:structured-macro ac:name="toc">
  <ac:parameter ac:name="type">list</ac:parameter>
  <ac:parameter ac:name="maxLevel">3</ac:parameter>
  <ac:parameter ac:name="minLevel">1</ac:parameter>
  <ac:parameter ac:name="style">disc</ac:parameter>
  <ac:parameter ac:name="indent">10px</ac:parameter>
  <ac:parameter ac:name="printable">true</ac:parameter>
  <ac:parameter ac:name="outline">false</ac:parameter>
  <ac:parameter ac:name="class">myclass</ac:parameter>
  <ac:parameter ac:name="separator">brackets</ac:parameter>
  <ac:parameter ac:name="include">.*</ac:parameter>
  <ac:parameter ac:name="exclude">概要</ac:parameter>
</ac:structured-macro>
```
**パラメータ:** `type`, `maxLevel`, `minLevel`, `style`, `indent`, `printable`, `outline`, `class`, `separator`, `include`, `exclude`
**Markdown 変換:** (Markdown には動的 TOC なし。変換時にヘッダーから生成するか省略)
```markdown
<!-- TOC -->
- [見出し1](#見出し1)
  - [見出し2](#見出し2)
<!-- /TOC -->
```
または単に省略して `[TOC]` (一部 Markdown エンジン対応)。

#### toc-zone (目次ゾーン)
```xml
<ac:structured-macro ac:name="toc-zone">
  <ac:parameter ac:name="type">list</ac:parameter>
  <ac:parameter ac:name="maxLevel">2</ac:parameter>
  <ac:parameter ac:name="minLevel">1</ac:parameter>
  <ac:rich-text-body>
    <h1>セクション1</h1>
    <p>テキスト</p>
    <h2>サブセクション</h2>
    <p>テキスト</p>
  </ac:rich-text-body>
</ac:structured-macro>
```
**Markdown 変換:** ゾーン内コンテンツをそのまま変換し、TOC は省略または生成。

#### anchor (アンカー)
```xml
<ac:structured-macro ac:name="anchor">
  <ac:parameter ac:name="">アンカー名</ac:parameter>
</ac:structured-macro>
```
注: Cloud では `ac:parameter ac:name=""` (空文字名) でデフォルトパラメータを表現する場合がある。旧形式では `ac:default-parameter`。
**Markdown 変換:**
```markdown
<a id="アンカー名"></a>
```

#### children (子ページ一覧)
```xml
<ac:structured-macro ac:name="children">
  <ac:parameter ac:name="sort">creation</ac:parameter>
  <ac:parameter ac:name="reverse">false</ac:parameter>
  <ac:parameter ac:name="style">h4</ac:parameter>
  <ac:parameter ac:name="page">親ページ名</ac:parameter>
  <ac:parameter ac:name="excerpt">none</ac:parameter>
  <ac:parameter ac:name="first">10</ac:parameter>
  <ac:parameter ac:name="depth">2</ac:parameter>
  <ac:parameter ac:name="all">false</ac:parameter>
</ac:structured-macro>
```
**パラメータ:** `sort`, `reverse`, `style`, `page`, `excerpt`, `first`, `depth`, `all`
**Markdown 変換:** (動的コンテンツのため、コメントまたはプレースホルダー)
```markdown
<!-- Children pages: 子ページ一覧はここに表示されます -->
```

#### page-tree (ページツリー)
```xml
<ac:structured-macro ac:name="pagetree">
  <ac:parameter ac:name="root">ページ名</ac:parameter>
  <ac:parameter ac:name="sort">natural</ac:parameter>
  <ac:parameter ac:name="excerpt">true</ac:parameter>
  <ac:parameter ac:name="reverse">false</ac:parameter>
  <ac:parameter ac:name="startDepth">1</ac:parameter>
  <ac:parameter ac:name="expandCollapseAll">true</ac:parameter>
  <ac:parameter ac:name="searchBox">true</ac:parameter>
</ac:structured-macro>
```
**Markdown 変換:**
```markdown
<!-- Page tree: ページツリーはここに表示されます -->
```

#### page-tree-search (ページツリー検索)
```xml
<ac:structured-macro ac:name="pagetreesearch">
  <ac:parameter ac:name="root">ページ名</ac:parameter>
</ac:structured-macro>
```

#### recently-updated (最近更新されたコンテンツ)
```xml
<ac:structured-macro ac:name="recently-updated">
  <ac:parameter ac:name="spaces">SPACE1,SPACE2</ac:parameter>
  <ac:parameter ac:name="max">10</ac:parameter>
  <ac:parameter ac:name="theme">sidebar</ac:parameter>
  <ac:parameter ac:name="labels">ラベル1,ラベル2</ac:parameter>
  <ac:parameter ac:name="types">page</ac:parameter>
  <ac:parameter ac:name="showProfilePic">true</ac:parameter>
  <ac:parameter ac:name="hideHeading">true</ac:parameter>
  <ac:parameter ac:name="width">50%</ac:parameter>
</ac:structured-macro>
```
**パラメータ:** `spaces`, `max`, `theme`, `labels`, `types`, `showProfilePic`, `hideHeading`, `width`, `author`
**Markdown 変換:**
```markdown
<!-- Recently updated content -->
```

### 1.5 コンテンツ埋め込み・参照系マクロ

#### excerpt (抜粋定義)
```xml
<ac:structured-macro ac:name="excerpt">
  <ac:parameter ac:name="hidden">true</ac:parameter>
  <ac:parameter ac:name="atlassian-macro-output-type">BLOCK</ac:parameter>
  <ac:rich-text-body>
    <p>この部分が抜粋として他ページに表示されます</p>
  </ac:rich-text-body>
</ac:structured-macro>
```
**パラメータ:** `hidden`, `atlassian-macro-output-type`
**Markdown 変換:** `hidden=true` ならコメント、そうでなければ本文として展開。
```markdown
この部分が抜粋として他ページに表示されます
```

#### excerpt-include (抜粋埋め込み)
```xml
<ac:structured-macro ac:name="excerpt-include">
  <ac:parameter ac:name="nopanel">true</ac:parameter>
  <ac:parameter ac:name="">ページ名</ac:parameter>
</ac:structured-macro>
```
注: Cloud 版では ri:page で参照する形式もある:
```xml
<ac:structured-macro ac:name="excerpt-include">
  <ac:parameter ac:name="nopanel">true</ac:parameter>
  <ac:rich-text-body>
    <ac:link>
      <ri:page ri:content-title="参照先ページ" ri:space-key="SPACE" />
    </ac:link>
  </ac:rich-text-body>
</ac:structured-macro>
```
**Markdown 変換:**
```markdown
<!-- Excerpt from: ページ名 -->
```

#### include (ページ埋め込み)
```xml
<ac:structured-macro ac:name="include">
  <ac:parameter ac:name="">SPACE:ページ名</ac:parameter>
</ac:structured-macro>
```
Cloud 版:
```xml
<ac:structured-macro ac:name="include">
  <ac:parameter ac:name=""><ac:link><ri:page ri:content-title="ページ名" ri:space-key="SPACE" /></ac:link></ac:parameter>
</ac:structured-macro>
```
**Markdown 変換:**
```markdown
<!-- Include: SPACE:ページ名 -->
```

### 1.6 ステータス・メタデータ系マクロ

#### status (ステータスラベル)
```xml
<ac:structured-macro ac:name="status">
  <ac:parameter ac:name="colour">Green</ac:parameter>
  <ac:parameter ac:name="title">完了</ac:parameter>
  <ac:parameter ac:name="subtle">false</ac:parameter>
</ac:structured-macro>
```
**パラメータ:** `colour` (Green/Yellow/Red/Blue/Purple/Grey), `title`, `subtle`
**Markdown 変換:**
```markdown
**[完了]**
```
またはバッジ風:
```markdown
![Status: 完了](https://img.shields.io/badge/Status-完了-green)
```

### 1.7 Jira 連携系マクロ

#### jira (Jira 課題)
```xml
<ac:structured-macro ac:name="jira">
  <ac:parameter ac:name="server">Jira Server</ac:parameter>
  <ac:parameter ac:name="serverId">144880e9-a353-312f-9412-e5028e8166fa</ac:parameter>
  <ac:parameter ac:name="key">PROJ-123</ac:parameter>
</ac:structured-macro>
```
Cloud 版 (JQL ベース):
```xml
<ac:structured-macro ac:name="jira">
  <ac:parameter ac:name="server">System JIRA</ac:parameter>
  <ac:parameter ac:name="jqlQuery">project = PROJ AND status = Open</ac:parameter>
  <ac:parameter ac:name="columns">type;key;summary;status</ac:parameter>
  <ac:parameter ac:name="maximumIssues">20</ac:parameter>
</ac:structured-macro>
```
**パラメータ:** `server`, `serverId`, `key`, `jqlQuery`, `columns`, `maximumIssues`, `count`
**Markdown 変換:**
```markdown
[PROJ-123](https://your-domain.atlassian.net/browse/PROJ-123)
```
JQL の場合:
```markdown
<!-- Jira query: project = PROJ AND status = Open -->
```

#### jirachart (Jira チャート)
```xml
<ac:structured-macro ac:name="jirachart">
  <ac:parameter ac:name="chartType">pie</ac:parameter>
  <ac:parameter ac:name="statType">issuetype</ac:parameter>
  <ac:parameter ac:name="jql">project = PROJ</ac:parameter>
  <ac:parameter ac:name="server">System JIRA</ac:parameter>
  <ac:parameter ac:name="serverId">UUID</ac:parameter>
  <ac:parameter ac:name="width">500</ac:parameter>
  <ac:parameter ac:name="border">true</ac:parameter>
  <ac:parameter ac:name="isAuthenticated">true</ac:parameter>
</ac:structured-macro>
```
**パラメータ:** `chartType`, `statType`, `jql`, `server`, `serverId`, `width`, `border`, `isAuthenticated`, `versionLabel`, `showinfor`
**Markdown 変換:**
```markdown
<!-- Jira chart: chartType=pie, jql=project = PROJ -->
```

### 1.8 その他の一般的なマクロ

#### widget (Widget Connector / 外部コンテンツ埋め込み)
```xml
<ac:structured-macro ac:name="widget">
  <ac:parameter ac:name="url">https://www.youtube.com/watch?v=xxxxx</ac:parameter>
  <ac:parameter ac:name="width">640</ac:parameter>
  <ac:parameter ac:name="height">480</ac:parameter>
</ac:structured-macro>
```
**パラメータ:** `url`, `width`, `height`
**Markdown 変換:**
```markdown
[動画リンク](https://www.youtube.com/watch?v=xxxxx)
```
YouTube の場合は埋め込み:
```markdown
[![動画](https://img.youtube.com/vi/xxxxx/0.jpg)](https://www.youtube.com/watch?v=xxxxx)
```

#### multimedia (マルチメディア)
```xml
<ac:structured-macro ac:name="multimedia">
  <ac:parameter ac:name="name">video.mp4</ac:parameter>
  <ac:parameter ac:name="page">ページ名</ac:parameter>
  <ac:parameter ac:name="space">SPACE</ac:parameter>
  <ac:parameter ac:name="width">640</ac:parameter>
  <ac:parameter ac:name="height">480</ac:parameter>
  <ac:parameter ac:name="autostart">false</ac:parameter>
</ac:structured-macro>
```
**パラメータ:** `name`, `page`, `space`, `width`, `height`, `autostart`
**Markdown 変換:**
```markdown
[video.mp4](添付ファイルURL)
```

#### gallery (ギャラリー)
```xml
<ac:structured-macro ac:name="gallery">
  <ac:parameter ac:name="title">画像ギャラリー</ac:parameter>
  <ac:parameter ac:name="columns">4</ac:parameter>
  <ac:parameter ac:name="sort">name</ac:parameter>
  <ac:parameter ac:name="reverse">false</ac:parameter>
  <ac:parameter ac:name="page">ページ名</ac:parameter>
  <ac:parameter ac:name="includeLabel">ラベル</ac:parameter>
  <ac:parameter ac:name="excludeLabel">除外ラベル</ac:parameter>
  <ac:parameter ac:name="include">.*\.png</ac:parameter>
  <ac:parameter ac:name="exclude">thumb_.*</ac:parameter>
</ac:structured-macro>
```
**パラメータ:** `title`, `columns`, `sort`, `reverse`, `page`, `includeLabel`, `excludeLabel`, `include`, `exclude`
**Markdown 変換:**
```markdown
<!-- Gallery: 画像ギャラリー -->
```

#### attachments (添付ファイル一覧)
```xml
<ac:structured-macro ac:name="attachments">
  <ac:parameter ac:name="old">false</ac:parameter>
  <ac:parameter ac:name="patterns">.*png,.*jpg</ac:parameter>
  <ac:parameter ac:name="sortBy">name</ac:parameter>
  <ac:parameter ac:name="page">ページ名</ac:parameter>
  <ac:parameter ac:name="sortOrder">ascending</ac:parameter>
  <ac:parameter ac:name="labels">ラベル</ac:parameter>
  <ac:parameter ac:name="upload">true</ac:parameter>
</ac:structured-macro>
```
**Markdown 変換:**
```markdown
<!-- Attachments list -->
```

#### blog-posts (ブログ投稿一覧)
```xml
<ac:structured-macro ac:name="blog-posts">
  <ac:parameter ac:name="content">titles</ac:parameter>
  <ac:parameter ac:name="spaces">@self,SPACE</ac:parameter>
  <ac:parameter ac:name="max">10</ac:parameter>
  <ac:parameter ac:name="label">ラベル</ac:parameter>
  <ac:parameter ac:name="sort">creation</ac:parameter>
  <ac:parameter ac:name="reverse">true</ac:parameter>
  <ac:parameter ac:name="author">ユーザー名</ac:parameter>
  <ac:parameter ac:name="time">4w</ac:parameter>
</ac:structured-macro>
```
**Markdown 変換:**
```markdown
<!-- Blog posts list -->
```

#### contentbylabel (ラベル別コンテンツ)
```xml
<ac:structured-macro ac:name="contentbylabel">
  <ac:parameter ac:name="labels">ラベル1,ラベル2</ac:parameter>
  <ac:parameter ac:name="spaces">@self</ac:parameter>
  <ac:parameter ac:name="max">10</ac:parameter>
  <ac:parameter ac:name="sort">modified</ac:parameter>
  <ac:parameter ac:name="reverse">true</ac:parameter>
  <ac:parameter ac:name="excerpt">true</ac:parameter>
  <ac:parameter ac:name="showLabels">true</ac:parameter>
  <ac:parameter ac:name="showSpace">true</ac:parameter>
  <ac:parameter ac:name="type">page</ac:parameter>
  <ac:parameter ac:name="operator">AND</ac:parameter>
</ac:structured-macro>
```

#### details (Page Properties / ページプロパティ)
```xml
<ac:structured-macro ac:name="details">
  <ac:parameter ac:name="hidden">false</ac:parameter>
  <ac:parameter ac:name="label">status</ac:parameter>
  <ac:rich-text-body>
    <table>
      <tbody>
        <tr><td>ステータス</td><td>完了</td></tr>
        <tr><td>担当者</td><td>田中</td></tr>
      </tbody>
    </table>
  </ac:rich-text-body>
</ac:structured-macro>
```
**Markdown 変換:**
```markdown
| プロパティ | 値   |
|-----------|------|
| ステータス | 完了 |
| 担当者     | 田中 |
```

#### detailssummary (Page Properties Report)
```xml
<ac:structured-macro ac:name="detailssummary">
  <ac:parameter ac:name="label">status</ac:parameter>
</ac:structured-macro>
```

#### chart (チャート)
```xml
<ac:structured-macro ac:name="chart">
  <ac:parameter ac:name="type">pie</ac:parameter>
  <ac:parameter ac:name="title">売上分布</ac:parameter>
  <ac:rich-text-body>
    <table>
      <tbody>
        <tr><th>カテゴリ</th><th>値</th></tr>
        <tr><td>A</td><td>100</td></tr>
        <tr><td>B</td><td>200</td></tr>
      </tbody>
    </table>
  </ac:rich-text-body>
</ac:structured-macro>
```
**パラメータ:** `type`, `title`, `stacked`, `height`, `width`, `legend`, `dataDisplay`, `colors`
**Markdown 変換:** テーブルとして展開:
```markdown
**売上分布**

| カテゴリ | 値  |
|---------|-----|
| A       | 100 |
| B       | 200 |
```

#### html (HTML マクロ)
```xml
<ac:structured-macro ac:name="html">
  <ac:plain-text-body><![CDATA[<div>HTML コンテンツ</div>]]></ac:plain-text-body>
</ac:structured-macro>
```
**Markdown 変換:** HTML をそのまま埋め込むか、変換可能な部分を Markdown 化。

#### html-include (HTML インクルード)
```xml
<ac:structured-macro ac:name="html-include">
  <ac:parameter ac:name="url">https://example.com/content.html</ac:parameter>
</ac:structured-macro>
```

#### change-history (変更履歴)
```xml
<ac:structured-macro ac:name="change-history" />
```

#### contributors (コントリビューター)
```xml
<ac:structured-macro ac:name="contributors">
  <ac:parameter ac:name="limit">10</ac:parameter>
  <ac:parameter ac:name="spaces">SPACE</ac:parameter>
  <ac:parameter ac:name="mode">list</ac:parameter>
</ac:structured-macro>
```

#### contributors-summary (コントリビューターサマリー)
```xml
<ac:structured-macro ac:name="contributors-summary">
  <ac:parameter ac:name="groupby">contributors</ac:parameter>
  <ac:parameter ac:name="columns">edits,comments</ac:parameter>
</ac:structured-macro>
```

#### content-report-table (コンテンツレポートテーブル)
```xml
<ac:structured-macro ac:name="content-report-table">
  <ac:parameter ac:name="spaces">SPACE</ac:parameter>
  <ac:parameter ac:name="labels">ラベル</ac:parameter>
</ac:structured-macro>
```

#### create-space-button (スペース作成ボタン)
```xml
<ac:structured-macro ac:name="create-space-button">
  <ac:parameter ac:name="size">small</ac:parameter>
</ac:structured-macro>
```

#### gadget (ガジェット)
```xml
<ac:structured-macro ac:name="gadget">
  <ac:parameter ac:name="url">ガジェットURL</ac:parameter>
  <ac:parameter ac:name="width">500</ac:parameter>
  <ac:parameter ac:name="border">true</ac:parameter>
</ac:structured-macro>
```

#### livesearch (ライブ検索)
```xml
<ac:structured-macro ac:name="livesearch">
  <ac:parameter ac:name="spaceKey">SPACE</ac:parameter>
</ac:structured-macro>
```

#### listlabels (ラベル一覧)
```xml
<ac:structured-macro ac:name="listlabels">
  <ac:parameter ac:name="spaceKey">SPACE</ac:parameter>
</ac:structured-macro>
```

#### loremipsum (ダミーテキスト)
```xml
<ac:structured-macro ac:name="loremipsum">
  <ac:parameter ac:name="">3</ac:parameter>
</ac:structured-macro>
```

#### roadmap (ロードマップ)
```xml
<ac:structured-macro ac:name="roadmap">
  <ac:parameter ac:name="title">プロジェクトロードマップ</ac:parameter>
  <!-- ロードマップ固有のパラメータ -->
  <ac:rich-text-body>
    <!-- ロードマップデータ -->
  </ac:rich-text-body>
</ac:structured-macro>
```
注: ロードマッププランナーマクロは Storage Format の直接編集では追加不可。

#### user-profile (ユーザープロファイル)
```xml
<ac:structured-macro ac:name="profile">
  <ac:parameter ac:name="user">
    <ri:user ri:account-id="xxxx" />
  </ac:parameter>
</ac:structured-macro>
```

#### favpages (お気に入りページ)
```xml
<ac:structured-macro ac:name="favpages" />
```

#### global-reports (グローバルレポート)
```xml
<ac:structured-macro ac:name="global-reports">
  <ac:parameter ac:name="width">50%</ac:parameter>
</ac:structured-macro>
```

---

## 2. ac:* 要素タイプ (マクロ以外)

### 2.1 ac:image (画像)

添付ファイルの画像:
```xml
<ac:image ac:align="center" ac:width="600" ac:height="400"
          ac:alt="代替テキスト" ac:title="画像タイトル"
          ac:border="true" ac:thumbnail="true"
          ac:class="custom-class" ac:style="border: 1px solid #ccc;"
          ac:vspace="5" ac:hspace="5"
          ac:original-height="800" ac:original-width="1200">
  <ri:attachment ri:filename="screenshot.png" />
</ac:image>
```

他ページの添付ファイル:
```xml
<ac:image>
  <ri:attachment ri:filename="image.png">
    <ri:page ri:content-title="別ページ" ri:space-key="SPACE" />
  </ri:attachment>
</ac:image>
```

外部 URL の画像:
```xml
<ac:image ac:alt="外部画像">
  <ri:url ri:value="https://example.com/image.png" />
</ac:image>
```

**属性一覧:**
| 属性 | 説明 | 値の例 |
|------|------|--------|
| `ac:align` | 配置 | `left`, `center`, `right` |
| `ac:border` | 枠線 | `true`, `false` |
| `ac:class` | CSS クラス | 文字列 |
| `ac:title` | ツールチップ | 文字列 |
| `ac:style` | インラインスタイル | CSS 文字列 |
| `ac:thumbnail` | サムネイル表示 | `true`, `false` |
| `ac:alt` | 代替テキスト | 文字列 |
| `ac:height` | 表示高さ | ピクセル値 |
| `ac:width` | 表示幅 | ピクセル値 |
| `ac:vspace` | 垂直余白 | ピクセル値 |
| `ac:hspace` | 水平余白 | ピクセル値 |
| `ac:original-height` | 元の高さ | ピクセル値 |
| `ac:original-width` | 元の幅 | ピクセル値 |

**Markdown 変換:**
```markdown
![代替テキスト](screenshot.png)
```
外部 URL の場合:
```markdown
![代替テキスト](https://example.com/image.png)
```

### 2.2 ac:link (リンク)

内部ページリンク:
```xml
<ac:link>
  <ri:page ri:content-title="ページタイトル" ri:space-key="SPACE" />
  <ac:plain-text-link-body><![CDATA[リンクテキスト]]></ac:plain-text-link-body>
</ac:link>
```

リッチテキストリンクボディ:
```xml
<ac:link>
  <ri:page ri:content-title="Home" ri:space-key="SANDBOX" />
  <ac:link-body>Some <strong>Rich</strong> Text</ac:link-body>
</ac:link>
```

添付ファイルへのリンク:
```xml
<ac:link>
  <ri:attachment ri:filename="document.pdf" />
  <ac:plain-text-link-body><![CDATA[ドキュメント]]></ac:plain-text-link-body>
</ac:link>
```

他ページの添付ファイルへのリンク:
```xml
<ac:link>
  <ri:attachment ri:filename="document.pdf">
    <ri:page ri:content-title="別ページ" ri:space-key="SPACE" />
  </ri:attachment>
  <ac:plain-text-link-body><![CDATA[ドキュメント]]></ac:plain-text-link-body>
</ac:link>
```

外部 URL リンク:
```xml
<ac:link>
  <ri:url ri:value="https://example.com" />
  <ac:plain-text-link-body><![CDATA[Example]]></ac:plain-text-link-body>
</ac:link>
```

アンカーリンク:
```xml
<ac:link ac:anchor="セクション名">
  <ac:plain-text-link-body><![CDATA[セクションへ]]></ac:plain-text-link-body>
</ac:link>
```

ページ内アンカーリンク (別ページ):
```xml
<ac:link ac:anchor="セクション名">
  <ri:page ri:content-title="別ページ" ri:space-key="SPACE" />
  <ac:plain-text-link-body><![CDATA[別ページのセクションへ]]></ac:plain-text-link-body>
</ac:link>
```

ブログ投稿リンク:
```xml
<ac:link>
  <ri:blog-post ri:content-title="ブログタイトル" ri:space-key="SPACE" ri:posting-day="2024/01/15" />
  <ac:plain-text-link-body><![CDATA[ブログ記事]]></ac:plain-text-link-body>
</ac:link>
```

ユーザーメンション (リンク形式):
```xml
<ac:link>
  <ri:user ri:account-id="5a1234567890abcdef123456" />
</ac:link>
```

スペースリンク:
```xml
<ac:link>
  <ri:space ri:space-key="SPACE" />
  <ac:plain-text-link-body><![CDATA[スペース名]]></ac:plain-text-link-body>
</ac:link>
```

**リンクボディの子要素:**
- `ac:plain-text-link-body` -- CDATA でラップされたプレーンテキスト
- `ac:link-body` -- 制限付き XHTML (`<b>`, `<strong>`, `<em>`, `<i>`, `<code>`, `<tt>`, `<sub>`, `<sup>`, `<br>`, `<span>`)

**Markdown 変換:**
```markdown
[リンクテキスト](ページURL)
```

### 2.3 ac:emoticon (絵文字)
```xml
<ac:emoticon ac:name="smile" />
<ac:emoticon ac:name="sad" />
<ac:emoticon ac:name="cheeky" />
<ac:emoticon ac:name="laugh" />
<ac:emoticon ac:name="wink" />
<ac:emoticon ac:name="thumbs-up" />
<ac:emoticon ac:name="thumbs-down" />
<ac:emoticon ac:name="information" />
<ac:emoticon ac:name="tick" />
<ac:emoticon ac:name="cross" />
<ac:emoticon ac:name="warning" />
<ac:emoticon ac:name="plus" />
<ac:emoticon ac:name="minus" />
<ac:emoticon ac:name="question" />
<ac:emoticon ac:name="light-on" />
<ac:emoticon ac:name="light-off" />
<ac:emoticon ac:name="yellow-star" />
<ac:emoticon ac:name="red-star" />
<ac:emoticon ac:name="green-star" />
<ac:emoticon ac:name="blue-star" />
<ac:emoticon ac:name="heart" />
<ac:emoticon ac:name="broken-heart" />
```

**属性:** `ac:name` (絵文字名), `ac:emoji-id` (Cloud 版のカスタム絵文字), `ac:emoji-shortname` (`:shortname:` 形式), `ac:emoji-fallback` (フォールバック文字)

**Markdown 変換マッピング:**
| ac:name | Markdown / Unicode |
|---------|-------------------|
| `smile` | `:)` |
| `sad` | `:(` |
| `cheeky` | `:P` |
| `laugh` | `:D` |
| `wink` | `;)` |
| `thumbs-up` | `:thumbsup:` |
| `thumbs-down` | `:thumbsdown:` |
| `information` | `:information_source:` |
| `tick` | `:white_check_mark:` |
| `cross` | `:x:` |
| `warning` | `:warning:` |
| `plus` | `:heavy_plus_sign:` |
| `minus` | `:heavy_minus_sign:` |
| `question` | `:question:` |
| `light-on` | `:bulb:` |
| `yellow-star` | `:star:` |
| `heart` | `:heart:` |

### 2.4 ac:task-list / ac:task (タスクリスト)
```xml
<ac:task-list>
  <ac:task>
    <ac:task-id>1</ac:task-id>
    <ac:task-status>complete</ac:task-status>
    <ac:task-body>
      <span class="placeholder-inline-tasks">
        完了済みタスク
        <ac:link>
          <ri:user ri:account-id="5a1234567890abcdef123456" />
        </ac:link>
      </span>
    </ac:task-body>
  </ac:task>
  <ac:task>
    <ac:task-id>2</ac:task-id>
    <ac:task-status>incomplete</ac:task-status>
    <ac:task-body>
      <span class="placeholder-inline-tasks">未完了タスク</span>
    </ac:task-body>
  </ac:task>
</ac:task-list>
```

**子要素構造:**
- `ac:task-list` -- タスクリストコンテナ
  - `ac:task` -- 個別タスク
    - `ac:task-id` -- タスク ID (数値)
    - `ac:task-status` -- `complete` または `incomplete`
    - `ac:task-body` -- タスクの内容 (XHTML)

**Markdown 変換:**
```markdown
- [x] 完了済みタスク @ユーザー
- [ ] 未完了タスク
```

### 2.5 ac:inline-comment-marker (インラインコメントマーカー)
```xml
<p>通常テキスト
  <ac:inline-comment-marker ac:ref="comment-id-12345">
    コメントが付けられたテキスト範囲
  </ac:inline-comment-marker>
  通常テキスト続き
</p>
```

**属性:** `ac:ref` (コメント ID への参照)
**Markdown 変換:** コメントマーカーは除去し、テキストのみ残す:
```markdown
通常テキストコメントが付けられたテキスト範囲通常テキスト続き
```

### 2.6 ac:placeholder (プレースホルダー)
```xml
<ac:placeholder>ここにテキストを入力してください</ac:placeholder>
<ac:placeholder ac:type="mention">@メンション入力</ac:placeholder>
```

**属性:** `ac:type` (プレースホルダーのタイプ。例: `mention`)
**Markdown 変換:** プレースホルダーテキストを残すか、省略:
```markdown
<!-- Placeholder: ここにテキストを入力してください -->
```

### 2.7 ac:layout / ac:layout-section / ac:layout-cell (ページレイアウト)

これらは `ac:structured-macro` の `section`/`column` とは独立したネイティブレイアウトシステム。

```xml
<ac:layout>
  <ac:layout-section ac:type="single">
    <ac:layout-cell>
      <p>全幅コンテンツ</p>
    </ac:layout-cell>
  </ac:layout-section>
  <ac:layout-section ac:type="two_equal">
    <ac:layout-cell>
      <p>左カラム (50%)</p>
    </ac:layout-cell>
    <ac:layout-cell>
      <p>右カラム (50%)</p>
    </ac:layout-cell>
  </ac:layout-section>
  <ac:layout-section ac:type="two_left_sidebar">
    <ac:layout-cell>
      <p>サイドバー (狭い)</p>
    </ac:layout-cell>
    <ac:layout-cell>
      <p>メインコンテンツ (広い)</p>
    </ac:layout-cell>
  </ac:layout-section>
  <ac:layout-section ac:type="two_right_sidebar">
    <ac:layout-cell>
      <p>メインコンテンツ (広い)</p>
    </ac:layout-cell>
    <ac:layout-cell>
      <p>サイドバー (狭い)</p>
    </ac:layout-cell>
  </ac:layout-section>
  <ac:layout-section ac:type="three_equal">
    <ac:layout-cell>
      <p>左</p>
    </ac:layout-cell>
    <ac:layout-cell>
      <p>中央</p>
    </ac:layout-cell>
    <ac:layout-cell>
      <p>右</p>
    </ac:layout-cell>
  </ac:layout-section>
  <ac:layout-section ac:type="three_with_sidebars">
    <ac:layout-cell>
      <p>左サイドバー</p>
    </ac:layout-cell>
    <ac:layout-cell>
      <p>メインコンテンツ</p>
    </ac:layout-cell>
    <ac:layout-cell>
      <p>右サイドバー</p>
    </ac:layout-cell>
  </ac:layout-section>
</ac:layout>
```

**ac:layout-section の ac:type 値:**
| type | 説明 | セル数 |
|------|------|--------|
| `single` | 全幅1カラム | 1 |
| `two_equal` | 均等2カラム | 2 |
| `two_left_sidebar` | 左サイドバー + メイン | 2 |
| `two_right_sidebar` | メイン + 右サイドバー | 2 |
| `three_equal` | 均等3カラム | 3 |
| `three_with_sidebars` | サイドバー + メイン + サイドバー | 3 |

**Markdown 変換:** カラムレイアウトは Markdown で表現不可のため、各セルの内容を順次展開:
```markdown
全幅コンテンツ

---

左カラム (50%)

右カラム (50%)
```

---

## 3. ri:* 要素タイプ (リソース識別子)

### 3.1 ri:page (ページ参照)
```xml
<ri:page ri:content-title="ページタイトル" ri:space-key="SPACE" />
```
**属性:**
| 属性 | 必須 | 説明 |
|------|------|------|
| `ri:content-title` | はい | ページタイトル |
| `ri:space-key` | いいえ | スペースキー (省略時は現在のスペース) |

### 3.2 ri:blog-post (ブログ投稿参照)
```xml
<ri:blog-post ri:content-title="ブログタイトル" ri:space-key="SPACE" ri:posting-day="2024/01/15" />
```
**属性:**
| 属性 | 必須 | 説明 |
|------|------|------|
| `ri:content-title` | はい | ブログ投稿タイトル |
| `ri:space-key` | いいえ | スペースキー |
| `ri:posting-day` | いいえ | 投稿日 (YYYY/MM/DD 形式) |

### 3.3 ri:attachment (添付ファイル参照)
単体 (現在ページの添付ファイル):
```xml
<ri:attachment ri:filename="document.pdf" />
```

他ページの添付ファイル:
```xml
<ri:attachment ri:filename="document.pdf">
  <ri:page ri:content-title="別ページ" ri:space-key="SPACE" />
</ri:attachment>
```

ブログ投稿の添付ファイル:
```xml
<ri:attachment ri:filename="image.png">
  <ri:blog-post ri:content-title="ブログタイトル" ri:space-key="SPACE" ri:posting-day="2024/01/15" />
</ri:attachment>
```

**属性:**
| 属性 | 必須 | 説明 |
|------|------|------|
| `ri:filename` | はい | ファイル名 |

**子要素:** `ri:page` または `ri:blog-post` (他コンテンツの添付ファイルを参照する場合)

### 3.4 ri:user (ユーザー参照)
Confluence Cloud:
```xml
<ri:user ri:account-id="5a1234567890abcdef123456" />
```

旧形式 (Server/Data Center):
```xml
<ri:user ri:username="taro" />
<ri:user ri:userkey="8a7f808246c0a55a0146c27a4ae70002" />
```

**属性 (Cloud):**
| 属性 | 説明 |
|------|------|
| `ri:account-id` | Atlassian Account ID (Cloud 固有, GDPR 対応) |
| `ri:local-id` | ローカル ID (一部で使用) |

**属性 (Server/Data Center レガシー):**
| 属性 | 説明 |
|------|------|
| `ri:username` | ユーザー名 (非推奨) |
| `ri:userkey` | ユーザーキー (非推奨) |

**Markdown 変換:**
```markdown
@ユーザー名
```
(変換時に account-id からユーザー名を API で解決する必要あり)

### 3.5 ri:url (外部 URL 参照)
```xml
<ri:url ri:value="https://example.com/path/to/resource" />
```
**属性:**
| 属性 | 必須 | 説明 |
|------|------|------|
| `ri:value` | はい | URL 文字列 |

### 3.6 ri:space (スペース参照)
```xml
<ri:space ri:space-key="PROJECTA" />
```
**属性:**
| 属性 | 必須 | 説明 |
|------|------|------|
| `ri:space-key` | はい | スペースキー |

### 3.7 ri:content-entity (汎用コンテンツ参照)
```xml
<ri:content-entity ri:content-id="123456789" />
```
**属性:**
| 属性 | 必須 | 説明 |
|------|------|------|
| `ri:content-id` | はい | コンテンツ ID (数値) |

注: ページ、ブログ、カスタムコンテンツなど任意のコンテンツタイプを ID で参照できる汎用要素。

### 3.8 ri:shortcut (ショートカット参照)
```xml
<ri:shortcut ri:key="jira" ri:parameter="ABC-123" />
```
**属性:**
| 属性 | 必須 | 説明 |
|------|------|------|
| `ri:key` | はい | ショートカットキー |
| `ri:parameter` | はい | パラメータ |

---

## 4. 標準 XHTML 要素 (Confluence で使用)

Storage Format には標準的な XHTML 要素も含まれる。

### 4.1 ブロック要素
| 要素 | Markdown 変換 |
|------|--------------|
| `<p>` | 段落 (空行区切り) |
| `<h1>` ~ `<h6>` | `#` ~ `######` |
| `<ul>`, `<li>` | `- ` 箇条書き |
| `<ol>`, `<li>` | `1. ` 番号付きリスト |
| `<table>`, `<tr>`, `<th>`, `<td>` | Markdown テーブル |
| `<blockquote>` | `> ` 引用 |
| `<pre>` | `` ``` `` コードブロック |
| `<hr />` | `---` |

### 4.2 インライン要素
| 要素 | Markdown 変換 |
|------|--------------|
| `<strong>`, `<b>` | `**太字**` |
| `<em>`, `<i>` | `*イタリック*` |
| `<code>`, `<tt>` | `` `コード` `` |
| `<del>`, `<s>` | `~~取り消し線~~` |
| `<sub>` | `<sub>下付き</sub>` (HTML そのまま) |
| `<sup>` | `<sup>上付き</sup>` (HTML そのまま) |
| `<br />` | 末尾2スペース + 改行 |
| `<a href="...">` | `[テキスト](URL)` |
| `<span>` | テキストのみ抽出 (スタイルは破棄) |
| `<u>` | `<u>下線</u>` (HTML そのまま) |

### 4.3 テーブル (拡張)
Confluence テーブルでは以下の属性が使用される:
```xml
<table class="confluenceTable">
  <colgroup><col /><col /></colgroup>
  <thead>
    <tr>
      <th class="confluenceTh"><p>ヘッダー1</p></th>
      <th class="confluenceTh"><p>ヘッダー2</p></th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td class="confluenceTd"><p>セル1</p></td>
      <td class="confluenceTd"><p>セル2</p></td>
    </tr>
    <tr>
      <td class="confluenceTd" colspan="2"><p>結合セル</p></td>
    </tr>
  </tbody>
</table>
```

注: `colspan`, `rowspan` は Markdown テーブルで表現不可。HTML テーブルとして出力するか、結合を解除して変換する。

---

## 5. Cloud 固有の注意事項

### 5.1 ac:structured-macro vs ac:macro
- **Confluence Cloud**: `ac:structured-macro` (V2) を使用。`ac:macro-id` 属性を持つ。
- **旧形式 (Server)**: `ac:macro` (V1) を使用。`ac:default-parameter` を子要素として持つ。
- Cloud からエクスポートしたデータは基本的に `ac:structured-macro` 形式。

### 5.2 ユーザー識別子
- Cloud: `ri:account-id` (Atlassian Account ID)
- Server/DC: `ri:username` または `ri:userkey`
- Cloud からの移行データには `ri:account-id` が使用される。

### 5.3 マクロ ID
Cloud 版では各マクロに UUID 形式の `ac:macro-id` が付与される:
```xml
<ac:structured-macro ac:name="code" ac:schema-version="1"
    ac:macro-id="5d87ef61-1fcf-4d30-8d3b-82305e13233a">
```
変換時には `ac:macro-id` と `ac:schema-version` は無視して問題ない。

### 5.4 名前空間宣言
Storage Format のルートには以下の名前空間が宣言される (ただし REST API レスポンスでは省略されることがある):
- `ac` = Atlassian Confluence
- `ri` = Resource Identifier

---

## 6. 変換優先度の推奨

コンバーター実装において、出現頻度と重要度に基づく処理優先度:

### Priority 1 (必須 -- ほぼ全ページに存在)
1. 標準 XHTML 要素 (`<p>`, `<h1>`-`<h6>`, `<ul>`, `<ol>`, `<table>`, `<strong>`, `<em>`, `<code>`, `<a>`, `<br>`)
2. `ac:image` + `ri:attachment` / `ri:url`
3. `ac:link` + `ri:page` / `ri:attachment` / `ri:url`
4. `ac:structured-macro ac:name="code"` (コードブロック)
5. `ac:structured-macro ac:name="info|note|warning|tip"` (通知パネル)

### Priority 2 (高頻度)
6. `ac:task-list` / `ac:task` (タスクリスト)
7. `ac:emoticon` (絵文字)
8. `ac:inline-comment-marker` (コメントマーカー -- 除去)
9. `ac:structured-macro ac:name="panel"` (パネル)
10. `ac:structured-macro ac:name="expand"` (折りたたみ)
11. `ac:structured-macro ac:name="status"` (ステータス)
12. `ac:structured-macro ac:name="toc"` (目次)
13. `ac:layout` / `ac:layout-section` / `ac:layout-cell` (レイアウト)

### Priority 3 (中頻度)
14. `ac:structured-macro ac:name="excerpt"` (抜粋)
15. `ac:structured-macro ac:name="noformat"` (整形なし)
16. `ac:structured-macro ac:name="quote"` (引用)
17. `ac:structured-macro ac:name="section"` / `column` (セクション/カラム)
18. `ac:structured-macro ac:name="anchor"` (アンカー)
19. `ac:structured-macro ac:name="jira"` (Jira)
20. `ac:structured-macro ac:name="include"` / `excerpt-include` (埋め込み)
21. `ac:structured-macro ac:name="children"` (子ページ)
22. `ac:structured-macro ac:name="details"` (ページプロパティ)

### Priority 4 (低頻度 -- プレースホルダー出力で十分)
23. `ac:structured-macro ac:name="widget"` (Widget Connector)
24. `ac:structured-macro ac:name="gallery"` (ギャラリー)
25. `ac:structured-macro ac:name="multimedia"` (マルチメディア)
26. `ac:structured-macro ac:name="chart"` (チャート)
27. `ac:structured-macro ac:name="jirachart"` (Jira チャート)
28. `ac:structured-macro ac:name="recently-updated"` (最近更新)
29. `ac:structured-macro ac:name="pagetree"` (ページツリー)
30. `ac:structured-macro ac:name="blog-posts"` (ブログ一覧)
31. `ac:structured-macro ac:name="contentbylabel"` (ラベル別コンテンツ)
32. `ac:structured-macro ac:name="attachments"` (添付一覧)
33. その他のマクロ

---

## 7. 参考リソース

- [Confluence Storage Format (Data Center 10.2)](https://confluence.atlassian.com/doc/confluence-storage-format-790796544.html)
- [Confluence Storage Format for Macros (5.0)](https://confluence.atlassian.com/spaces/CONF50/pages/329980084/Confluence+Storage+Format+for+Macros)
- [Confluence Cloud REST API v2](https://developer.atlassian.com/cloud/confluence/rest/v2/intro/)
- [Info, Tip, Note, Warning Macros](https://confluence.atlassian.com/doc/info-tip-note-and-warning-macros-51872369.html)
- [Code Block Macro](https://confluence.atlassian.com/doc/code-block-macro-139390.html)
- [Panel Macro](https://confluence.atlassian.com/conf59/panel-macro-792499179.html)
- [Status Macro](https://confluence.atlassian.com/conf59/status-macro-792499207.html)
- [Jira Chart Macro](https://confluence.atlassian.com/doc/jira-chart-macro-427623467.html)
- [Jira Issues Macro](https://confluence.atlassian.com/doc/jira-issues-macro-139380.html)
- [Table of Contents Macro](https://confluence.atlassian.com/doc/table-of-contents-macro-182682099.html)
- [Expand Macro](https://confluence.atlassian.com/doc/expand-macro-223222352.html)
- [Noformat Macro](https://confluence.atlassian.com/doc/noformat-macro-139545.html)
- [Page Tree Macro](https://confluence.atlassian.com/doc/page-tree-macro-163414255.html)
- [Children Display Macro](https://confluence.atlassian.com/doc/children-display-macro-139501.html)
- [Recently Updated Macro](https://confluence.atlassian.com/display/DOC/Recently+Updated+Macro)
- [Excerpt Macro](https://confluence.atlassian.com/display/DOC/Excerpt+Macro)
- [Include Page Macro](https://confluence.atlassian.com/doc/include-page-macro-139514.html)
- [ac:structured-macro vs ac:macro (Developer Community)](https://community.developer.atlassian.com/t/difference-between-ac-structured-macro-and-ac-macro/7688)
- [Confluence Cloud ri:user account-id](https://community.developer.atlassian.com/t/when-are-user-keys-used-in-mentions-and-how-to-resolve-them/76933)
- [confluence-to-markdown (GitHub)](https://github.com/meridius/confluence-to-markdown)
- [confluence-markdown-exporter (GitHub)](https://github.com/Spenhouet/confluence-markdown-exporter)
