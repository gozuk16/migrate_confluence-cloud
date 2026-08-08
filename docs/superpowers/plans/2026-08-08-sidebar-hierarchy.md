# 左サイドバー階層ツリー化（フォルダ対応）実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Hugoサイトの左サイドバーを、Confluenceと同じ親子階層＋フォルダ対応の開閉式ツリーにする。

**Architecture:** Go移行ツールが各ページのフロントマターに `parent_id` と `weight` を出力し、Confluenceのフォルダは「レンダリングされないページ」（`build.render = "never"`）としてスタブ出力する。Hugoテーマ側は `page_id`/`parent_id` を辿る再帰パーシャルで `<details>`/`<summary>` のツリーを描画し、現在ページの祖先を自動展開する。URL・ディレクトリ構造はフラットのまま変更しない。

**Tech Stack:** Go 1.x（標準ライブラリ＋`net/http/httptest`でのテスト）、Confluence REST API v2、Hugo v0.163.3 extended、素のCSS。

## Global Constraints

- 設計の正典は `docs/superpowers/specs/2026-08-08-sidebar-hierarchy-design.md`。矛盾が出たらスペックを優先し、逸脱するなら先に相談する。
- **URL・ディレクトリ構造は変更しない**。出力先は現行どおり `outputDir/<スペースキー>/<サニタイズ済みタイトル>/index.md`。
- **フォルダはリンクにしない**。クリックで開閉のみ。ページはリンクする。
- フロントマターはTOML（`+++` 区切り）。既存の `parent`（親タイトル）キーは互換のため残す。
- `weight = position + 1`（必ず1以上）。positionがnull・未取得なら `weight = 9999`。
- **テーマは git submodule（別リポジトリ `git@github.com:gozuk16/hugo-theme-docs.git`）**。`hugo-site/themes/hugo-theme-docs/` 配下の変更はそのサブモジュールリポジトリ内でブランチを切ってコミットし、親リポジトリではサブモジュールのポインタ更新を別途コミットする。親リポジトリで `git add hugo-site/themes/hugo-theme-docs` するとポインタだけが記録される。
- 作業ブランチは親リポジトリが `feature/sidebar-hierarchy`（作成済み・スペックコミット済み）。mainへ直接コミットしない。
- コード内コメント・コミットメッセージ・ログメッセージは日本語（既存コードの慣習に合わせる）。
- 各タスクの最後に必ずコミットする。

## 前提の検証結果（実施済み・再確認不要）

Hugo v0.163.3 で以下を実機確認済み。実装時に疑う必要はない。

- フロントマターの `[build]` と `[_build]` はどちらも有効。本計画では `[build]` を使う。
- `build.render = "never"` かつ `list = "always"` のページは、HTMLが生成されず `.RelPermalink` が空になる一方、`.RegularPages` には残る。フォルダスタブの要件を満たす。
- 本計画に載せた `sidebar-left.html` / `sidebar-tree.html` のテンプレートコードは、階層描画・祖先の自動展開・現在ページ強調・解決不能な親のルートフォールバック・weight順ソートが動作することを検証済み。

## File Structure

**Goツール（親リポジトリ）**

| ファイル | 責務 | 変更種別 |
|---|---|---|
| `confluenceclient.go` | `Page` に `ParentType`/`Position` を追加、`Folder` 型と `GetFolder` を追加 | 変更 |
| `foldertree.go` | ページ群から親フォルダを再帰収集する `CollectFolders`（API呼び出しはインターフェース経由） | 新規 |
| `foldertree_test.go` | `CollectFolders` のテスト | 新規 |
| `mdwriter.go` | ページのフロントマターに `parent_id`/`weight` を追加、フォルダスタブを書く `WriteFolder` を追加 | 変更 |
| `intermediatesaver.go` | 中間ファイルのメタデータに `parent_type`/`position` を保存・復元 | 変更 |
| `main.go` | `space` コマンドでフォルダを収集して出力する配線 | 変更 |

**テーマ（サブモジュール `hugo-site/themes/hugo-theme-docs`）**

| ファイル | 責務 | 変更種別 |
|---|---|---|
| `layouts/_partials/sidebar-tree.html` | ツリー1階層分を描画し子を再帰呼び出しする | 新規 |
| `layouts/_partials/sidebar-left.html` | 索引・祖先集合・ルート集合を組み立て、`sidebar-tree.html` を呼ぶ | 全面書き換え |
| `layouts/page.html` / `layouts/section.html` | `partialCached` → `partial` に変更（後述の理由） | 変更 |
| `assets/css/main.css` | ツリーの階層インデント・開閉マーカー・フォルダ/現在ページのスタイル | 変更 |

**`partialCached` を外す理由（重要）**: 現在 `page.html` と `section.html` は `partialCached "sidebar-left.html" . .CurrentSection.RelPermalink` を使っており、キャッシュキーがセクション単位になっている。サイドバーの中身が「現在ページ」に依存するようになる（祖先の自動展開・現在ページ強調）ため、キャッシュしたままだとセクション内の全ページで最初にレンダリングされた1ページ分のサイドバーが使い回され、展開状態が誤る。`partial` に変更する。

---

### Task 1: APIクライアントにフォルダ取得とposition/parentTypeを追加

**Files:**
- Modify: `confluenceclient.go`（`Page` 構造体 35-45行目付近、末尾に `Folder` 型と `GetFolder`）
- Test: `confluenceclient_test.go`（末尾に追記）

**Interfaces:**
- Consumes: 既存の `cc.doRequest("GET", apiURL)`、`newTestConfluenceClient(serverURL)`
- Produces:
  - `Page.ParentType string`（JSONタグ `parentType`）、`Page.Position *int`（JSONタグ `position`）
  - `type Folder struct { ID, Title, Status, ParentID, ParentType string; Position *int }`
  - `func (cc *ConfluenceClient) GetFolder(folderID string) (*Folder, error)`

`Position` を `*int` にするのは、APIが `null` を返す場合と `0`（先頭）を区別するため。

- [ ] **Step 1: 失敗するテストを書く**

`confluenceclient_test.go` の末尾に追記:

```go
// TestGetFolder はGetFolderのテスト
func TestGetFolder(t *testing.T) {
	tests := []struct {
		name         string
		folderID     string
		handler      http.HandlerFunc
		wantErr      bool
		wantTitle    string
		wantParentID string
		wantPosition *int
	}{
		{
			name:     "正常系: フォルダ取得成功",
			folderID: "555",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/wiki/api/v2/folders/555" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"id":"555","title":"設計ドキュメント","status":"current","parentId":"111","parentType":"folder","position":3}`))
			},
			wantTitle:    "設計ドキュメント",
			wantParentID: "111",
			wantPosition: intPtr(3),
		},
		{
			name:     "正常系: positionがnullの場合はnil",
			folderID: "556",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"id":"556","title":"未整理","status":"current","position":null}`))
			},
			wantTitle:    "未整理",
			wantParentID: "",
			wantPosition: nil,
		},
		{
			name:     "異常系: 404",
			folderID: "999",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			cc := newTestConfluenceClient(server.URL)
			folder, err := cc.GetFolder(tt.folderID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("エラーを期待しましたが nil でした")
				}
				return
			}
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}
			if folder.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", folder.Title, tt.wantTitle)
			}
			if folder.ParentID != tt.wantParentID {
				t.Errorf("ParentID = %q, want %q", folder.ParentID, tt.wantParentID)
			}
			if tt.wantPosition == nil {
				if folder.Position != nil {
					t.Errorf("Position = %v, want nil", *folder.Position)
				}
			} else if folder.Position == nil || *folder.Position != *tt.wantPosition {
				t.Errorf("Position = %v, want %d", folder.Position, *tt.wantPosition)
			}
		})
	}
}

// TestPageParentTypeAndPosition はページJSONのparentType/positionが読めることを確認する
func TestPageParentTypeAndPosition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"12345","title":"テストページ","spaceId":"67890","parentId":"555","parentType":"folder","position":7}`))
	}))
	defer server.Close()

	cc := newTestConfluenceClient(server.URL)
	page, err := cc.GetPage("12345")
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if page.ParentType != "folder" {
		t.Errorf("ParentType = %q, want %q", page.ParentType, "folder")
	}
	if page.Position == nil || *page.Position != 7 {
		t.Errorf("Position = %v, want 7", page.Position)
	}
}

// intPtr はテスト用に int のポインタを返す
func intPtr(v int) *int { return &v }
```

- [ ] **Step 2: テストが失敗することを確認**

Run: `go test ./... -run 'TestGetFolder|TestPageParentTypeAndPosition' -v`
Expected: コンパイルエラー（`cc.GetFolder` undefined、`page.ParentType` undefined）

- [ ] **Step 3: 実装する**

`confluenceclient.go` の `Page` 構造体に2フィールド追加:

```go
// Page はConfluenceページ情報
type Page struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Status     string   `json:"status"`
	SpaceID    string   `json:"spaceId"`
	ParentID   string   `json:"parentId"`
	ParentType string   `json:"parentType"` // "page" または "folder"
	Position   *int     `json:"position"`   // 同一階層内の並び順。null と 0 を区別するためポインタ
	Body       PageBody `json:"body"`
	Version    Version  `json:"version"`
	Links      Links    `json:"_links"`
}
```

`confluenceclient.go` の末尾に `Folder` 型と `GetFolder` を追加:

```go
// Folder はConfluenceのフォルダ情報（v2 API の folder コンテンツタイプ）
type Folder struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	ParentID   string `json:"parentId"`
	ParentType string `json:"parentType"` // "page" または "folder"
	Position   *int   `json:"position"`
}

// GetFolder はフォルダ情報を取得する
func (cc *ConfluenceClient) GetFolder(folderID string) (*Folder, error) {
	apiURL := fmt.Sprintf("%s/wiki/api/v2/folders/%s", cc.baseURL, folderID)

	body, err := cc.doRequest("GET", apiURL)
	if err != nil {
		return nil, fmt.Errorf("フォルダ取得エラー (ID: %s): %w", folderID, err)
	}

	var folder Folder
	if err := json.Unmarshal(body, &folder); err != nil {
		return nil, fmt.Errorf("フォルダJSONパースエラー (ID: %s): %w", folderID, err)
	}

	return &folder, nil
}
```

- [ ] **Step 4: テストが通ることを確認**

Run: `go test ./... -run 'TestGetFolder|TestPageParentTypeAndPosition' -v`
Expected: PASS（3サブテスト＋1テスト）

- [ ] **Step 5: 全テストとlintを実行**

Run: `make test && make lint`
Expected: 既存テストも含めて全てPASS

- [ ] **Step 6: コミット**

```bash
git add confluenceclient.go confluenceclient_test.go
git commit -m "feat: Confluenceフォルダ取得APIとページのparentType/positionに対応

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

- [ ] **Step 7: 実APIでのフィールド存在確認（要認証・任意だが強く推奨）**

`config.toml` に有効な認証情報がある場合、v2 APIのページレスポンスに実際に `parentType`/`position` が含まれるか確認する。含まれない場合は後続タスクの前提が崩れるため、ここで判明させる。

```bash
LOG_LEVEL=DEBUG ./migConfluence page --id <既知のページID> 2>&1 | head -40
```

確認できない環境ならスキップし、Task 9の目視確認時に実データで検証する。

---

### Task 2: ページ群から親フォルダを再帰収集する

**Files:**
- Create: `foldertree.go`
- Test: `foldertree.go` に対する `foldertree_test.go`

**Interfaces:**
- Consumes: Task 1の `Folder` 型、`Page.ParentType`、`Page.ParentID`
- Produces:
  - `type FolderGetter interface { GetFolder(folderID string) (*Folder, error) }`（`*ConfluenceClient` が自動的に満たす）
  - `func CollectFolders(getter FolderGetter, pages []Page) []Folder` — ページ群の親を辿って必要なフォルダだけを集める。取得失敗したフォルダはスキップし警告ログを出す。戻り値はID昇順で安定させる

APIエラーを戻り値にせずスキップするのは、1つのフォルダが取れなくても移行全体を止めないため（スペックのエッジケース方針）。

- [ ] **Step 1: 失敗するテストを書く**

`foldertree_test.go` を新規作成:

```go
package main

import (
	"fmt"
	"testing"
)

// fakeFolderGetter はテスト用のFolderGetter実装
type fakeFolderGetter struct {
	folders map[string]*Folder
	calls   []string // 呼び出されたフォルダIDの記録（重複取得の検出用）
}

func (f *fakeFolderGetter) GetFolder(folderID string) (*Folder, error) {
	f.calls = append(f.calls, folderID)
	folder, ok := f.folders[folderID]
	if !ok {
		return nil, fmt.Errorf("フォルダが見つかりません: %s", folderID)
	}
	return folder, nil
}

func TestCollectFolders(t *testing.T) {
	t.Run("親がフォルダのページからフォルダを収集する", func(t *testing.T) {
		getter := &fakeFolderGetter{folders: map[string]*Folder{
			"10": {ID: "10", Title: "設計", ParentType: "page", ParentID: "1"},
		}}
		pages := []Page{
			{ID: "1", Title: "Home"},
			{ID: "2", Title: "子", ParentID: "10", ParentType: "folder"},
		}

		folders := CollectFolders(getter, pages)

		if len(folders) != 1 {
			t.Fatalf("フォルダ数 = %d, want 1", len(folders))
		}
		if folders[0].ID != "10" || folders[0].Title != "設計" {
			t.Errorf("folders[0] = %+v, want ID=10 Title=設計", folders[0])
		}
	})

	t.Run("フォルダの親がフォルダの場合は再帰的に収集する", func(t *testing.T) {
		getter := &fakeFolderGetter{folders: map[string]*Folder{
			"10": {ID: "10", Title: "子フォルダ", ParentType: "folder", ParentID: "20"},
			"20": {ID: "20", Title: "親フォルダ", ParentType: "page", ParentID: "1"},
		}}
		pages := []Page{
			{ID: "2", Title: "孫ページ", ParentID: "10", ParentType: "folder"},
		}

		folders := CollectFolders(getter, pages)

		if len(folders) != 2 {
			t.Fatalf("フォルダ数 = %d, want 2 (%+v)", len(folders), folders)
		}
		if folders[0].ID != "10" || folders[1].ID != "20" {
			t.Errorf("ID順が想定と違います: %s, %s", folders[0].ID, folders[1].ID)
		}
	})

	t.Run("同じフォルダを複数ページが参照しても1回しか取得しない", func(t *testing.T) {
		getter := &fakeFolderGetter{folders: map[string]*Folder{
			"10": {ID: "10", Title: "共有フォルダ", ParentType: "page", ParentID: "1"},
		}}
		pages := []Page{
			{ID: "2", ParentID: "10", ParentType: "folder"},
			{ID: "3", ParentID: "10", ParentType: "folder"},
			{ID: "4", ParentID: "10", ParentType: "folder"},
		}

		folders := CollectFolders(getter, pages)

		if len(folders) != 1 {
			t.Fatalf("フォルダ数 = %d, want 1", len(folders))
		}
		if len(getter.calls) != 1 {
			t.Errorf("GetFolder呼び出し回数 = %d, want 1 (%v)", len(getter.calls), getter.calls)
		}
	})

	t.Run("取得に失敗したフォルダはスキップして処理を続ける", func(t *testing.T) {
		getter := &fakeFolderGetter{folders: map[string]*Folder{
			"10": {ID: "10", Title: "取得できるフォルダ", ParentType: "page", ParentID: "1"},
		}}
		pages := []Page{
			{ID: "2", ParentID: "10", ParentType: "folder"},
			{ID: "3", ParentID: "99", ParentType: "folder"}, // 存在しない
		}

		folders := CollectFolders(getter, pages)

		if len(folders) != 1 {
			t.Fatalf("フォルダ数 = %d, want 1 (%+v)", len(folders), folders)
		}
		if folders[0].ID != "10" {
			t.Errorf("folders[0].ID = %s, want 10", folders[0].ID)
		}
	})

	t.Run("親がページのみの場合はフォルダを取得しない", func(t *testing.T) {
		getter := &fakeFolderGetter{folders: map[string]*Folder{}}
		pages := []Page{
			{ID: "2", ParentID: "1", ParentType: "page"},
			{ID: "3", ParentID: "", ParentType: ""},
		}

		folders := CollectFolders(getter, pages)

		if len(folders) != 0 {
			t.Fatalf("フォルダ数 = %d, want 0", len(folders))
		}
		if len(getter.calls) != 0 {
			t.Errorf("GetFolderが呼ばれました: %v", getter.calls)
		}
	})
}
```

- [ ] **Step 2: テストが失敗することを確認**

Run: `go test ./... -run TestCollectFolders -v`
Expected: コンパイルエラー（`CollectFolders` undefined）

- [ ] **Step 3: 実装する**

`foldertree.go` を新規作成:

```go
package main

import (
	"log/slog"
	"sort"
)

// FolderGetter はフォルダ取得のインターフェース（テスト差し替え用）
type FolderGetter interface {
	GetFolder(folderID string) (*Folder, error)
}

// CollectFolders はページ群の親を辿って、サイドバーに必要なフォルダだけを収集する。
// 親がフォルダのページからフォルダIDを集め、そのフォルダの親がさらにフォルダなら再帰的に取得する。
// 取得に失敗したフォルダは警告ログを出してスキップし、処理は継続する。
func CollectFolders(getter FolderGetter, pages []Page) []Folder {
	found := make(map[string]*Folder)

	// 未取得のフォルダIDをキューで処理する（フォルダの親がフォルダのケースに対応）
	queue := make([]string, 0)
	for _, page := range pages {
		if page.ParentType == "folder" && page.ParentID != "" {
			queue = append(queue, page.ParentID)
		}
	}

	for len(queue) > 0 {
		folderID := queue[0]
		queue = queue[1:]

		if _, ok := found[folderID]; ok {
			continue
		}

		folder, err := getter.GetFolder(folderID)
		if err != nil {
			slog.Warn("フォルダ取得エラー（スキップします）", "folderID", folderID, "error", err)
			continue
		}
		found[folderID] = folder

		if folder.ParentType == "folder" && folder.ParentID != "" {
			queue = append(queue, folder.ParentID)
		}
	}

	// ID順に並べて結果を安定させる
	ids := make([]string, 0, len(found))
	for id := range found {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	folders := make([]Folder, 0, len(ids))
	for _, id := range ids {
		folders = append(folders, *found[id])
	}

	return folders
}
```

注意: 取得失敗したフォルダIDは `found` に入らないため、同じIDが複数ページから参照されると失敗時に複数回リトライされる。ページ数が多い場合の無駄を避けたければ「試行済みID」の集合を別途持つ改善が考えられるが、YAGNIのため現時点では実装しない。

- [ ] **Step 4: テストが通ることを確認**

Run: `go test ./... -run TestCollectFolders -v`
Expected: PASS（5サブテスト）

- [ ] **Step 5: コミット**

```bash
git add foldertree.go foldertree_test.go
git commit -m "feat: ページの親を辿ってConfluenceフォルダを収集するCollectFoldersを追加

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 3: ページのフロントマターに parent_id と weight を出力する

**Files:**
- Modify: `mdwriter.go`（`generateFrontMatter` 153-193行目）
- Test: `mdwriter_test.go`（末尾に追記）

**Interfaces:**
- Consumes: Task 1の `Page.ParentID`、`Page.Position`
- Produces: `func frontMatterWeight(position *int) int` — `position+1`（最低1）を返し、nilなら9999

`WritePage` のシグネチャは変更しない。`ParentID`/`Position` は既に `*Page` から読めるため。

- [ ] **Step 1: 失敗するテストを書く**

`mdwriter_test.go` の末尾に追記:

```go
func TestFrontMatterWeight(t *testing.T) {
	tests := []struct {
		name     string
		position *int
		want     int
	}{
		{name: "position=0 は weight=1", position: intPtr(0), want: 1},
		{name: "position=5 は weight=6", position: intPtr(5), want: 6},
		{name: "position=nil は weight=9999", position: nil, want: 9999},
		{name: "負のpositionでも1以上", position: intPtr(-3), want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := frontMatterWeight(tt.position); got != tt.want {
				t.Errorf("frontMatterWeight() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMDWriter_FrontMatterHierarchy(t *testing.T) {
	t.Run("parent_idとweightが出力される", func(t *testing.T) {
		tmpDir := t.TempDir()
		writer := newTestMDWriter(tmpDir)

		page := &Page{
			ID:         "12345",
			Title:      "子ページ",
			SpaceID:    "67890",
			ParentID:   "555",
			ParentType: "folder",
			Position:   intPtr(2),
			Body: PageBody{
				AtlasDocFormat: AtlasDocFormat{
					Value: `{"version":1,"type":"doc","content":[]}`,
				},
			},
		}

		if err := writer.WritePage(page, "TEST", "テストスペース", "親フォルダ", nil, nil, nil); err != nil {
			t.Fatalf("WritePage エラー: %v", err)
		}

		mdPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title), "index.md")
		data, err := os.ReadFile(mdPath)
		if err != nil {
			t.Fatalf("読み込みエラー: %v", err)
		}
		content := string(data)

		for _, want := range []string{
			`parent_id = "555"`,
			`weight = 3`,
			`parent = "親フォルダ"`, // 既存キーは互換のため残す
		} {
			if !strings.Contains(content, want) {
				t.Errorf("フロントマターに %q が含まれていません:\n%s", want, content)
			}
		}
	})

	t.Run("親がない場合はparent_idを出力しない", func(t *testing.T) {
		tmpDir := t.TempDir()
		writer := newTestMDWriter(tmpDir)

		page := &Page{
			ID:      "1",
			Title:   "ルートページ",
			SpaceID: "67890",
			Body: PageBody{
				AtlasDocFormat: AtlasDocFormat{Value: `{"version":1,"type":"doc","content":[]}`},
			},
		}

		if err := writer.WritePage(page, "TEST", "テストスペース", "", nil, nil, nil); err != nil {
			t.Fatalf("WritePage エラー: %v", err)
		}

		mdPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(page.Title), "index.md")
		data, err := os.ReadFile(mdPath)
		if err != nil {
			t.Fatalf("読み込みエラー: %v", err)
		}
		if strings.Contains(string(data), "parent_id") {
			t.Errorf("parent_id が出力されています:\n%s", string(data))
		}
		if !strings.Contains(string(data), "weight = 9999") {
			t.Errorf("position未設定時の weight = 9999 が出力されていません:\n%s", string(data))
		}
	})
}
```

- [ ] **Step 2: テストが失敗することを確認**

Run: `go test ./... -run 'TestFrontMatterWeight|TestMDWriter_FrontMatterHierarchy' -v`
Expected: コンパイルエラー（`frontMatterWeight` undefined）

- [ ] **Step 3: 実装する**

`mdwriter.go` の `generateFrontMatter` 内、既存の `parent` 出力ブロックの直後に追記:

```go
	if parentTitle != "" {
		sb.WriteString(fmt.Sprintf("parent = %q\n", parentTitle))
	}

	// 階層構造用: 親のID（ページ・フォルダ共通）とサイドバーの並び順
	if page.ParentID != "" {
		sb.WriteString(fmt.Sprintf("parent_id = %q\n", page.ParentID))
	}
	sb.WriteString(fmt.Sprintf("weight = %d\n", frontMatterWeight(page.Position)))
```

`mdwriter.go` の末尾（`buildAttachmentMap` の後）にヘルパーを追加:

```go
// frontMatterWeight は Confluence の position を Hugo の weight に変換する。
// Hugo の weight 昇順ソートでは 0 が先頭に来てしまうため必ず 1 以上にし、
// position が取得できない場合は 9999 として末尾に寄せる。
func frontMatterWeight(position *int) int {
	if position == nil {
		return 9999
	}
	weight := *position + 1
	if weight < 1 {
		weight = 1
	}
	return weight
}
```

- [ ] **Step 4: テストが通ることを確認**

Run: `go test ./... -run 'TestFrontMatterWeight|TestMDWriter_FrontMatterHierarchy' -v`
Expected: PASS

- [ ] **Step 5: 既存テストが壊れていないか確認**

Run: `make test`
Expected: 全てPASS。既存の `TestMDWriter_WritePage` などがフロントマター全文を比較している場合は `weight` 行の追加で失敗しうるので、その場合は期待値に `weight` 行を足して修正する。

- [ ] **Step 6: コミット**

```bash
git add mdwriter.go mdwriter_test.go
git commit -m "feat: ページのフロントマターにparent_idとweightを出力

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 4: フォルダスタブを出力する WriteFolder

**Files:**
- Modify: `mdwriter.go`（`WritePage` の後に `WriteFolder` を追加）
- Test: `mdwriter_test.go`（末尾に追記）

**Interfaces:**
- Consumes: Task 1の `Folder` 型、Task 3の `frontMatterWeight`
- Produces: `func (w *MDWriter) WriteFolder(folder *Folder, spaceKey, spaceTitle string) error` — `outputDir/<spaceKey>/<サニタイズ済みタイトル>/index.md` にレンダリングされないスタブを書く。同名ディレクトリに既にフォルダ以外のindex.mdがある場合は `<タイトル>_<フォルダID>` にフォールバックする

- [ ] **Step 1: 失敗するテストを書く**

`mdwriter_test.go` の末尾に追記:

```go
func TestMDWriter_WriteFolder(t *testing.T) {
	t.Run("フォルダスタブが出力される", func(t *testing.T) {
		tmpDir := t.TempDir()
		writer := newTestMDWriter(tmpDir)

		folder := &Folder{
			ID:         "555",
			Title:      "設計ドキュメント",
			ParentID:   "1",
			ParentType: "page",
			Position:   intPtr(2),
		}

		if err := writer.WriteFolder(folder, "TEST", "テストスペース"); err != nil {
			t.Fatalf("WriteFolder エラー: %v", err)
		}

		mdPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(folder.Title), "index.md")
		data, err := os.ReadFile(mdPath)
		if err != nil {
			t.Fatalf("読み込みエラー: %v", err)
		}
		content := string(data)

		for _, want := range []string{
			`title = "設計ドキュメント"`,
			`page_id = "555"`,
			`parent_id = "1"`,
			`weight = 3`,
			`is_folder = true`,
			`space = "TEST"`,
			`space_title = "テストスペース"`,
			`[build]`,
			`render = "never"`,
			`list = "always"`,
		} {
			if !strings.Contains(content, want) {
				t.Errorf("スタブに %q が含まれていません:\n%s", want, content)
			}
		}
	})

	t.Run("親がない場合はparent_idを出力しない", func(t *testing.T) {
		tmpDir := t.TempDir()
		writer := newTestMDWriter(tmpDir)

		folder := &Folder{ID: "555", Title: "ルートフォルダ"}

		if err := writer.WriteFolder(folder, "TEST", ""); err != nil {
			t.Fatalf("WriteFolder エラー: %v", err)
		}

		mdPath := filepath.Join(tmpDir, "TEST", sanitizeFilename(folder.Title), "index.md")
		data, err := os.ReadFile(mdPath)
		if err != nil {
			t.Fatalf("読み込みエラー: %v", err)
		}
		if strings.Contains(string(data), "parent_id") {
			t.Errorf("parent_id が出力されています:\n%s", string(data))
		}
	})

	t.Run("同名ページが既にある場合はID付きディレクトリにフォールバックする", func(t *testing.T) {
		tmpDir := t.TempDir()
		writer := newTestMDWriter(tmpDir)

		// 先に同名のページを書く
		page := &Page{
			ID:      "1",
			Title:   "重複名",
			SpaceID: "67890",
			Body: PageBody{
				AtlasDocFormat: AtlasDocFormat{Value: `{"version":1,"type":"doc","content":[]}`},
			},
		}
		if err := writer.WritePage(page, "TEST", "テストスペース", "", nil, nil, nil); err != nil {
			t.Fatalf("WritePage エラー: %v", err)
		}

		folder := &Folder{ID: "555", Title: "重複名"}
		if err := writer.WriteFolder(folder, "TEST", "テストスペース"); err != nil {
			t.Fatalf("WriteFolder エラー: %v", err)
		}

		// ページ側は上書きされていない
		pageData, err := os.ReadFile(filepath.Join(tmpDir, "TEST", "重複名", "index.md"))
		if err != nil {
			t.Fatalf("ページ読み込みエラー: %v", err)
		}
		if strings.Contains(string(pageData), "is_folder") {
			t.Errorf("ページがフォルダスタブに上書きされました:\n%s", string(pageData))
		}

		// フォルダはID付きディレクトリに出ている
		folderData, err := os.ReadFile(filepath.Join(tmpDir, "TEST", "重複名_555", "index.md"))
		if err != nil {
			t.Fatalf("フォルダスタブが見つかりません: %v", err)
		}
		if !strings.Contains(string(folderData), "is_folder = true") {
			t.Errorf("フォルダスタブの内容が不正です:\n%s", string(folderData))
		}
	})

	t.Run("同じフォルダを2回書いても同じディレクトリを使う", func(t *testing.T) {
		tmpDir := t.TempDir()
		writer := newTestMDWriter(tmpDir)

		folder := &Folder{ID: "555", Title: "再実行フォルダ"}
		if err := writer.WriteFolder(folder, "TEST", ""); err != nil {
			t.Fatalf("1回目 WriteFolder エラー: %v", err)
		}
		if err := writer.WriteFolder(folder, "TEST", ""); err != nil {
			t.Fatalf("2回目 WriteFolder エラー: %v", err)
		}

		if _, err := os.Stat(filepath.Join(tmpDir, "TEST", "再実行フォルダ_555")); err == nil {
			t.Error("2回目の実行でID付きディレクトリが作られました（冪等ではありません）")
		}
	})
}
```

- [ ] **Step 2: テストが失敗することを確認**

Run: `go test ./... -run TestMDWriter_WriteFolder -v`
Expected: コンパイルエラー（`writer.WriteFolder` undefined）

- [ ] **Step 3: 実装する**

`mdwriter.go` の `WritePage` の直後に追加:

```go
// WriteFolder はConfluenceのフォルダを「レンダリングされないページ」スタブとして書き出す。
// Hugoの build.render = "never" によりURLもHTMLも生成されないが、
// list = "always" によりテンプレートのページ一覧には現れるため、サイドバーのツリーに使える。
func (w *MDWriter) WriteFolder(folder *Folder, spaceKey, spaceTitle string) error {
	folderDir, err := w.folderDir(folder, spaceKey)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(folderDir, 0755); err != nil {
		return fmt.Errorf("フォルダディレクトリの作成に失敗しました: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("+++\n")
	sb.WriteString(fmt.Sprintf("title = %q\n", folder.Title))
	sb.WriteString(fmt.Sprintf("space = %q\n", spaceKey))
	if spaceTitle != "" {
		sb.WriteString(fmt.Sprintf("space_title = %q\n", spaceTitle))
	}
	sb.WriteString(fmt.Sprintf("page_id = %q\n", folder.ID))
	if folder.ParentID != "" {
		sb.WriteString(fmt.Sprintf("parent_id = %q\n", folder.ParentID))
	}
	sb.WriteString(fmt.Sprintf("weight = %d\n", frontMatterWeight(folder.Position)))
	sb.WriteString("is_folder = true\n")
	sb.WriteString("[build]\n")
	sb.WriteString("  render = \"never\"\n")
	sb.WriteString("  list = \"always\"\n")
	sb.WriteString("+++\n")

	mdPath := filepath.Join(folderDir, "index.md")
	if err := os.WriteFile(mdPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("フォルダスタブ書き出しエラー: %w", err)
	}

	return nil
}

// folderDir はフォルダスタブの出力先ディレクトリを返す。
// 同名ディレクトリに既にフォルダ以外のindex.mdがある場合は、
// ページを上書きしないよう "<タイトル>_<フォルダID>" にフォールバックする。
func (w *MDWriter) folderDir(folder *Folder, spaceKey string) (string, error) {
	safeTitle := sanitizeFilename(folder.Title)
	dir := filepath.Join(w.outputDir, spaceKey, safeTitle)

	data, err := os.ReadFile(filepath.Join(dir, "index.md"))
	if err != nil {
		if os.IsNotExist(err) {
			return dir, nil // 未使用のディレクトリ名なのでそのまま使う
		}
		return "", fmt.Errorf("既存ファイルの確認に失敗しました (%s): %w", dir, err)
	}

	// 自分自身のスタブなら同じディレクトリを再利用する（再実行時の冪等性）
	if strings.Contains(string(data), fmt.Sprintf("page_id = %q", folder.ID)) {
		return dir, nil
	}

	return filepath.Join(w.outputDir, spaceKey, fmt.Sprintf("%s_%s", safeTitle, folder.ID)), nil
}
```

- [ ] **Step 4: テストが通ることを確認**

Run: `go test ./... -run TestMDWriter_WriteFolder -v`
Expected: PASS（4サブテスト）

- [ ] **Step 5: 全テストとlintを実行**

Run: `make test && make lint`
Expected: 全てPASS

- [ ] **Step 6: コミット**

```bash
git add mdwriter.go mdwriter_test.go
git commit -m "feat: Confluenceフォルダをレンダリングされないページスタブとして出力

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 5: 中間ファイルに parent_type と position を保存する

**Files:**
- Modify: `intermediatesaver.go`（`PageMetadata` 13-27行目、`SavePage` 71-84行目、`LoadPage` 153-172行目）
- Test: `intermediatesaver_test.go`（末尾に追記）

**Interfaces:**
- Consumes: Task 1の `Page.ParentType`/`Page.Position`
- Produces: `PageMetadata` に `ParentType string`（TOMLキー `parent_type`）と `Position *int`（TOMLキー `position`）を追加。`LoadPage` がそれらを `*Page` に復元する

これがないと `convert` コマンド（保存済み中間ファイルからの再生成）で階層情報が失われる。

- [ ] **Step 1: 失敗するテストを書く**

`intermediatesaver_test.go` の末尾に追記:

```go
func TestIntermediateSaver_HierarchyRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	saver := NewIntermediateSaver(tmpDir)

	page := &Page{
		ID:         "12345",
		Title:      "階層テストページ",
		Status:     "current",
		SpaceID:    "67890",
		ParentID:   "555",
		ParentType: "folder",
		Position:   intPtr(4),
		Body: PageBody{
			AtlasDocFormat: AtlasDocFormat{Value: `{"version":1,"type":"doc","content":[]}`},
		},
		Version: Version{Number: 1, CreatedAt: "2024-01-01T00:00:00.000Z"},
	}

	if err := saver.SavePage(page, "TEST", nil); err != nil {
		t.Fatalf("SavePage エラー: %v", err)
	}

	loaded, _, err := saver.LoadPage("TEST", page.Title)
	if err != nil {
		t.Fatalf("LoadPage エラー: %v", err)
	}

	if loaded.ParentID != "555" {
		t.Errorf("ParentID = %q, want %q", loaded.ParentID, "555")
	}
	if loaded.ParentType != "folder" {
		t.Errorf("ParentType = %q, want %q", loaded.ParentType, "folder")
	}
	if loaded.Position == nil || *loaded.Position != 4 {
		t.Errorf("Position = %v, want 4", loaded.Position)
	}
}

func TestIntermediateSaver_PositionNilRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	saver := NewIntermediateSaver(tmpDir)

	page := &Page{
		ID:      "1",
		Title:   "position無しページ",
		SpaceID: "67890",
		Body: PageBody{
			AtlasDocFormat: AtlasDocFormat{Value: `{"version":1,"type":"doc","content":[]}`},
		},
	}

	if err := saver.SavePage(page, "TEST", nil); err != nil {
		t.Fatalf("SavePage エラー: %v", err)
	}

	loaded, _, err := saver.LoadPage("TEST", page.Title)
	if err != nil {
		t.Fatalf("LoadPage エラー: %v", err)
	}
	if loaded.Position != nil {
		t.Errorf("Position = %v, want nil", *loaded.Position)
	}
}
```

- [ ] **Step 2: テストが失敗することを確認**

Run: `go test ./... -run 'TestIntermediateSaver_HierarchyRoundTrip|TestIntermediateSaver_PositionNilRoundTrip' -v`
Expected: FAIL（`ParentType` が空、`Position` が nil）

- [ ] **Step 3: 実装する**

`intermediatesaver.go` の `PageMetadata` にフィールドを追加（`ParentTitle` の後）:

```go
// PageMetadata はページのメタデータを表す構造体（TOML保存用）
type PageMetadata struct {
	ID          string   `toml:"id"`
	Title       string   `toml:"title"`
	Status      string   `toml:"status"`
	SpaceID     string   `toml:"space_id"`
	SpaceKey    string   `toml:"space_key"`
	ParentID    string   `toml:"parent_id"`
	ParentTitle string   `toml:"parent_title"`
	ParentType  string   `toml:"parent_type"`
	Position    *int     `toml:"position"`
	CreatedAt   string   `toml:"created_at"`
	UpdatedAt   string   `toml:"updated_at"`
	AuthorID    string   `toml:"author_id"`
	Version     int      `toml:"version"`
	Labels      []string `toml:"labels"`
	WebURL      string   `toml:"web_url"`
}
```

`SavePage` の `meta := PageMetadata{...}` に2行追加:

```go
	meta := PageMetadata{
		ID:         page.ID,
		Title:      page.Title,
		Status:     page.Status,
		SpaceID:    page.SpaceID,
		SpaceKey:   spaceKey,
		ParentID:   page.ParentID,
		ParentType: page.ParentType,
		Position:   page.Position,
		CreatedAt:  page.Version.CreatedAt,
		UpdatedAt:  page.Version.CreatedAt,
		AuthorID:   page.Version.AuthorID,
		Version:    page.Version.Number,
		Labels:     labelNames,
		WebURL:     page.Links.WebUI,
	}
```

`LoadPage` の `page := &Page{...}` に3行追加（既存の `ParentID` 未設定も併せて修正する）:

```go
	page := &Page{
		ID:         meta.ID,
		Title:      meta.Title,
		Status:     meta.Status,
		SpaceID:    meta.SpaceID,
		ParentID:   meta.ParentID,
		ParentType: meta.ParentType,
		Position:   meta.Position,
		Body: PageBody{
			AtlasDocFormat: AtlasDocFormat{
				Value:          string(jsonData),
				Representation: "atlas_doc_format",
			},
		},
		Version: Version{
			Number:    meta.Version,
			CreatedAt: meta.CreatedAt,
			AuthorID:  meta.AuthorID,
		},
		Links: Links{
			WebUI: meta.WebURL,
		},
	}
```

- [ ] **Step 4: テストが通ることを確認**

Run: `go test ./... -run 'TestIntermediateSaver' -v`
Expected: PASS

- [ ] **Step 5: 全テストとlintを実行**

Run: `make test && make lint`
Expected: 全てPASS

- [ ] **Step 6: コミット**

```bash
git add intermediatesaver.go intermediatesaver_test.go
git commit -m "feat: 中間ファイルにparent_type/positionを保存しconvert経路でも階層を維持

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 6: space コマンドでフォルダを収集して出力する配線

**Files:**
- Modify: `main.go`（`fetchSpace` 266-332行目）

**Interfaces:**
- Consumes: Task 2の `CollectFolders`、Task 4の `WriteFolder`
- Produces: なし（配線のみ）

`page` コマンド（単一ページ）はフォルダ収集の対象外とする。スペース全体を見ないとフォルダツリーを組めず、単一ページ取得の用途では階層表示が目的ではないため。

- [ ] **Step 1: fetchSpace にフォルダ収集を追加**

`main.go` の `fetchSpace` 内、各ページ処理ループ（`for i, page := range pages {`）の**直前**に追加:

```go
	// フォルダの収集と出力（ページの親を辿って必要なフォルダだけ取得する）
	folders := CollectFolders(client, pages)
	if len(folders) > 0 {
		fmt.Printf("フォルダ: %d 件\n", len(folders))
		for _, folder := range folders {
			if err := writer.WriteFolder(&folder, space.Key, space.Name); err != nil {
				slog.Warn("フォルダ出力エラー", "folderID", folder.ID, "title", folder.Title, "error", err)
			}
		}
	}
```

`CollectFolders(client, pages)` の `client` は `*ConfluenceClient` で、Task 1で追加した `GetFolder` により `FolderGetter` を満たす。

注意: Goのバージョンによってはループ変数 `folder` のアドレス取得に注意が必要だが、`&folder` を `WriteFolder` に渡した直後に使い切るため問題ない（Go 1.22以降はループ変数がイテレーションごとに新しくなる）。

- [ ] **Step 2: ビルドが通ることを確認**

Run: `make build`
Expected: エラーなし

- [ ] **Step 3: 全テストとlintを実行**

Run: `make test && make lint`
Expected: 全てPASS

- [ ] **Step 4: コミット**

```bash
git add main.go
git commit -m "feat: spaceコマンドでフォルダを収集してスタブを出力

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 7: サイドバーを再帰ツリーに書き換える（テーマ側）

**Files（すべてサブモジュール `hugo-site/themes/hugo-theme-docs` 内）:**
- Create: `layouts/_partials/sidebar-tree.html`
- Modify: `layouts/_partials/sidebar-left.html`（全面書き換え）
- Modify: `layouts/page.html`（1行目の `partialCached` → `partial`）
- Modify: `layouts/section.html`（1行目の `partialCached` → `partial`）
- Modify: `assets/css/main.css`（「サイドバーナビ」セクション 50-62行目を置き換え）

**Interfaces:**
- Consumes: Task 3/4が出力するフロントマター（`page_id`、`parent_id`、`weight`、`is_folder`、`space_title`）
- Produces: `sidebar-tree.html` は `dict` で以下を受け取る再帰パーシャル
  - `nodes`: 描画対象ページのスライス
  - `pages`: セクション内の全ページ（子の検索に使う）
  - `ancestors`: 現在ページの祖先 `page_id` のスライス（自動展開判定）
  - `current`: 現在ページ
  - `depth`: 再帰の深さ（無限ループ防止に10で打ち切り）

以下のテンプレートコードはHugo v0.163.3で動作検証済み。そのまま使うこと。

- [ ] **Step 1: サブモジュールで作業ブランチを作る**

```bash
cd hugo-site/themes/hugo-theme-docs
git checkout main
git pull --ff-only 2>/dev/null || true
git checkout -b feature/sidebar-hierarchy
```

サブモジュールは detached HEAD になっていることがあるため、必ず `main` からブランチを切る。

- [ ] **Step 2: 再帰パーシャル `sidebar-tree.html` を作成**

`hugo-site/themes/hugo-theme-docs/layouts/_partials/sidebar-tree.html`:

```html
{{- /*
  ツリー1階層分を描画し、子があれば自身を再帰呼び出しする。
  引数: nodes（描画対象）, pages（セクション全ページ）, ancestors（現在ページの祖先ID）,
        current（現在ページ）, depth（再帰の深さ）
*/ -}}
{{- $pages := .pages }}
{{- $ancestors := .ancestors }}
{{- $current := .current }}
{{- $depth := .depth }}
{{- if and .nodes (lt $depth 10) }}
  <ul>
    {{- /* weight昇順。同一weightはタイトル順で安定させるため二段ソートする */}}
    {{- range $node := sort (sort .nodes "Title") "Weight" }}
      {{- $id := $node.Params.page_id | default "" }}
      {{- $children := slice }}
      {{- if $id }}
        {{- range $p := $pages }}
          {{- if eq ($p.Params.parent_id | default "") $id }}
            {{- $children = $children | append $p }}
          {{- end }}
        {{- end }}
      {{- end }}
      {{- $isFolder := $node.Params.is_folder }}
      {{- $isCurrent := eq $node $current }}
      {{- $isOpen := or (in $ancestors $id) $isCurrent }}
      <li>
        {{- if $children }}
          <details{{ if $isOpen }} open{{ end }}>
            <summary class="tree-item{{ if $isFolder }} is-folder{{ end }}">
              {{- if $isFolder }}
                <span class="tree-label">{{ $node.Title }}</span>
              {{- else }}
                <a class="tree-link{{ if $isCurrent }} is-current{{ end }}" href="{{ $node.RelPermalink }}">{{ $node.Title }}</a>
              {{- end }}
            </summary>
            {{- partial "sidebar-tree.html" (dict "nodes" $children "pages" $pages "ancestors" $ancestors "current" $current "depth" (add $depth 1)) }}
          </details>
        {{- else }}
          {{- if $isFolder }}
            <span class="tree-item is-folder is-leaf"><span class="tree-label">{{ $node.Title }}</span></span>
          {{- else }}
            <span class="tree-item is-leaf"><a class="tree-link{{ if $isCurrent }} is-current{{ end }}" href="{{ $node.RelPermalink }}">{{ $node.Title }}</a></span>
          {{- end }}
        {{- end }}
      </li>
    {{- end }}
  </ul>
{{- end }}
```

- [ ] **Step 3: `sidebar-left.html` を書き換え**

`hugo-site/themes/hugo-theme-docs/layouts/_partials/sidebar-left.html` を以下で全面置換:

```html
{{- $page := . }}
{{- $section := .CurrentSection }}
{{- with $section }}
  {{- $pages := .RegularPages }}

  {{- /* スペース表示名: いずれかのページの space_title を採用（無ければセクション名） */}}
  {{- $title := .Title }}
  {{- range $p := $pages }}
    {{- if and (eq $title $section.Title) $p.Params.space_title }}
      {{- $title = $p.Params.space_title }}
    {{- end }}
  {{- end }}

  {{- /* page_id → ページ の索引 */}}
  {{- $byID := dict }}
  {{- range $p := $pages }}
    {{- with $p.Params.page_id }}
      {{- $byID = merge $byID (dict . $p) }}
    {{- end }}
  {{- end }}

  {{- /* 現在ページの祖先 page_id 集合（自動展開に使う）。深さは20で打ち切る */}}
  {{- $ancestors := slice }}
  {{- $pid := $page.Params.parent_id | default "" }}
  {{- range seq 20 }}
    {{- if $pid }}
      {{- $ancestors = $ancestors | append $pid }}
      {{- $parent := index $byID $pid }}
      {{- if $parent }}
        {{- $pid = $parent.Params.parent_id | default "" }}
      {{- else }}
        {{- $pid = "" }}
      {{- end }}
    {{- end }}
  {{- end }}

  {{- /* ルート: parent_id が空、または親が同セクションに存在しない（解決不能な親はルートに出す） */}}
  {{- $roots := slice }}
  {{- range $p := $pages }}
    {{- $ppid := $p.Params.parent_id | default "" }}
    {{- if or (not $ppid) (not (index $byID $ppid)) }}
      {{- $roots = $roots | append $p }}
    {{- end }}
  {{- end }}

  <nav aria-label="ページ一覧" class="page-tree">
    <h2><a href="{{ .RelPermalink }}">{{ $title }}</a></h2>
    {{- partial "sidebar-tree.html" (dict "nodes" $roots "pages" $pages "ancestors" $ancestors "current" $page "depth" 0) }}
  </nav>
{{- end }}
```

- [ ] **Step 4: `partialCached` を `partial` に変更**

`layouts/page.html` の1〜3行目を以下に変更:

```html
{{ define "sidebar-left" }}
  {{ partial "sidebar-left.html" . }}
{{ end }}
```

`layouts/section.html` の1〜3行目も同様に変更:

```html
{{ define "sidebar-left" }}
  {{ partial "sidebar-left.html" . }}
{{ end }}
```

サイドバーの内容が現在ページに依存するようになったため、セクション単位のキャッシュは誤った展開状態を全ページに配ってしまう。

- [ ] **Step 5: CSSを更新**

`assets/css/main.css` の「サイドバーナビ」セクション（`.sidebar-left nav h2` から `.sidebar-left nav ul li a:hover` までの既存6ルール）を、以下で置き換える:

```css
/* サイドバーナビ（階層ツリー） */
.sidebar-left nav h2 { font-size: .95rem; margin: 0 0 .5rem; }
.sidebar-left nav h2 a { text-decoration: none; color: inherit; }
.sidebar-left nav ul { list-style: none; padding: 0; margin: 0; }
/* 2階層目以降を字下げする */
.sidebar-left nav ul ul { padding-left: .75rem; border-left: 1px solid #eee; margin-left: .35rem; }

/* パディングはリンク／ラベル側だけに持たせる（.tree-item は入れ物なので付けない） */
.sidebar-left .tree-item { display: block; }
.sidebar-left .tree-link {
  display: block;
  padding: .25rem .4rem;
  font-size: .9rem;
  color: #333;
  text-decoration: none;
  border-radius: 3px;
}
.sidebar-left .tree-link:hover { background: #f0f0f0; }
.sidebar-left .tree-link.is-current { background: #e8f0fe; font-weight: bold; }

/* 開閉トグル: summary の三角マーカーを自前の▶に置き換える */
.sidebar-left details > summary {
  list-style: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: .2rem;
  padding: 0;
}
.sidebar-left details > summary::-webkit-details-marker { display: none; }
.sidebar-left details > summary::before {
  content: "▶";
  flex: 0 0 auto;
  font-size: .6rem;
  color: #888;
  transition: transform .15s;
  padding-left: .2rem;
}
.sidebar-left details[open] > summary::before { transform: rotate(90deg); }
.sidebar-left details > summary .tree-link,
.sidebar-left details > summary .tree-label { flex: 1 1 auto; }

/* フォルダはリンクではないラベル */
.sidebar-left .tree-label {
  display: block;
  padding: .25rem .4rem;
  font-size: .9rem;
  color: #555;
  border-radius: 3px;
}
.sidebar-left .is-folder > .tree-label::before { content: "📁 "; font-size: .8rem; }
.sidebar-left details > summary:hover .tree-label { background: #f0f0f0; }

/* 子を持たない項目は▶の分だけ字下げして開閉項目と縦位置を揃える */
.sidebar-left .tree-item.is-leaf { padding-left: .9rem; }
```

- [ ] **Step 6: ビルドして描画を確認**

```bash
cd hugo-site && hugo --quiet --destination /tmp/hugo-check
```

Expected: エラーなし。この時点では実データに `parent_id` が無いため全ページがフラットに並ぶ（すべてルート扱い）が、それが正しい挙動。エラーが出ないことだけ確認する。

- [ ] **Step 7: サブモジュールでコミット**

```bash
cd hugo-site/themes/hugo-theme-docs
git add layouts/_partials/sidebar-tree.html layouts/_partials/sidebar-left.html layouts/page.html layouts/section.html assets/css/main.css
git commit -m "feat: 左サイドバーをフォルダ対応の開閉式階層ツリーにする

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

サブモジュールのpushとPRは Task 9 でまとめて行う。

---

### Task 8: 階層フィクスチャで表示を検証する

**Files:**
- Create: `hugo-site/content/sample/` 配下に階層フィクスチャ一式（親リポジトリ）

`content/sample/` は既存の動作確認用コンテンツで、`content/SCRUM/` は移行ツールの出力（gitで追跡していない）。フィクスチャは `sample` 側に置く。

- [ ] **Step 1: フィクスチャを作成**

以下のファイルを作成する。既存の `hugo-site/content/sample/test-page/index.md` は変更しない。

`hugo-site/content/sample/tree-home/index.md`:

```
+++
title = "ツリー検証ホーム"
page_id = "t1"
weight = 1
space_title = "ツリー検証スペース"
+++

階層表示の検証用ルートページ。
```

`hugo-site/content/sample/tree-folder/index.md`:

```
+++
title = "検証フォルダ"
page_id = "t2"
weight = 2
is_folder = true
[build]
  render = "never"
  list = "always"
+++
```

`hugo-site/content/sample/tree-child/index.md`:

```
+++
title = "フォルダ直下のページ"
page_id = "t3"
parent_id = "t2"
weight = 1
+++

フォルダの子ページ。
```

`hugo-site/content/sample/tree-subfolder/index.md`:

```
+++
title = "入れ子フォルダ"
page_id = "t4"
parent_id = "t2"
weight = 2
is_folder = true
[build]
  render = "never"
  list = "always"
+++
```

`hugo-site/content/sample/tree-deep/index.md`:

```
+++
title = "入れ子フォルダの中のページ"
page_id = "t5"
parent_id = "t4"
weight = 1
+++

2階層下のページ。
```

`hugo-site/content/sample/tree-orphan/index.md`:

```
+++
title = "親が解決できないページ"
page_id = "t6"
parent_id = "存在しないID"
weight = 3
+++

親が見つからないためルート直下に出るはずのページ。
```

- [ ] **Step 2: ビルドして構造を検証**

```bash
cd hugo-site && hugo --quiet --destination /tmp/hugo-check
```

Expected: エラーなし。フォルダのHTMLは生成されない:

```bash
ls /tmp/hugo-check/sample/
```

Expected: `tree-home`, `tree-child`, `tree-deep`, `tree-orphan`, `test-page` はあるが、`tree-folder` と `tree-subfolder` は無い。

- [ ] **Step 3: 自動展開と階層を検証**

```bash
grep -o '<details[^>]*>\|is-current\|tree-label">[^<]*' /tmp/hugo-check/sample/tree-deep/index.html
```

Expected: `<details open>` が2つ（検証フォルダと入れ子フォルダの両方が展開）、`is-current` が1つ、フォルダラベルが2つ。

```bash
grep -c '<details open>' /tmp/hugo-check/sample/tree-home/index.html
```

Expected: `0`（ツリー検証ホームはどのフォルダの子孫でもないため、フォルダは折りたたまれている）

- [ ] **Step 4: 目視確認**

```bash
make hugo-serve
```

ブラウザで `http://localhost:1313/sample/tree-deep/` を開き、以下を確認する:

- サイドバーが階層構造（インデント）で表示される
- フォルダに📁アイコンが付き、クリックしてもページ遷移せず開閉だけする
- ページはクリックで遷移する
- 現在ページがハイライトされ、その祖先フォルダが開いた状態になっている
- 「親が解決できないページ」がルート直下に出ている
- weight順（ツリー検証ホーム → 検証フォルダ → 親が解決できないページ）に並んでいる

確認後 Ctrl+C でサーバーを止める。

- [ ] **Step 5: コミット**

```bash
git add hugo-site/content/sample/
git commit -m "test: サイドバー階層表示の検証用フィクスチャを追加

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 9: ドキュメント更新・サブモジュール反映・PR作成

**Files:**
- Modify: `TODO.md`、`CHANGELOG.md`（親リポジトリ）
- Modify: サブモジュールポインタ `hugo-site/themes/hugo-theme-docs`

- [ ] **Step 1: サブモジュールをpushしてPRを作る**

```bash
cd hugo-site/themes/hugo-theme-docs
git push -u origin feature/sidebar-hierarchy
gh pr create --title "feat: 左サイドバーをフォルダ対応の階層ツリーにする" --body "$(cat <<'EOF'
## 概要
左サイドバーを、ページの `parent_id` を辿る再帰ツリーに変更しました。Confluenceのフォルダ（`is_folder = true` かつ `build.render = "never"` のスタブページ）にも対応します。

## 変更内容
- `layouts/_partials/sidebar-tree.html` を新規追加（1階層を描画して子を再帰呼び出し）
- `layouts/_partials/sidebar-left.html` を書き換え（索引・祖先集合・ルート集合の組み立て）
- `layouts/page.html` / `layouts/section.html` の `partialCached` を `partial` に変更（サイドバーが現在ページに依存するようになったため、セクション単位のキャッシュでは展開状態が誤る）
- `assets/css/main.css` にツリーのインデント・開閉マーカー・フォルダ/現在ページのスタイルを追加

## 動作
- 子を持つ項目は `<details>`/`<summary>` で開閉（JS不要）
- 現在ページの祖先は自動展開、現在ページはハイライト
- フォルダはリンクせず開閉のみ
- 並び順は `weight` 昇順（同一weightはタイトル順）
- 親が解決できないページはルート直下にフォールバック

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 2: 親リポジトリでサブモジュールポインタを更新**

```bash
cd /Users/gozu/go/src/github.com/gozuk16/migrate_confluence-cloud
git add hugo-site/themes/hugo-theme-docs
git commit -m "chore: テーマサブモジュールを階層サイドバー対応に更新

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

注意: サブモジュールのPRがマージされたら、マージ後のコミットを指すようポインタを更新し直すこと（`cd hugo-site/themes/hugo-theme-docs && git checkout main && git pull` の後、親で再度 `git add` してコミット）。

- [ ] **Step 3: TODO.md と CHANGELOG.md を更新**

`TODO.md` に完了項目として、`CHANGELOG.md` に変更内容として、それぞれ既存の記法に合わせて追記する。記載する内容:

- 左サイドバーをConfluence同様の階層ツリーに変更（開閉式・フォルダ対応）
- Confluenceフォルダの取得（`GET /wiki/api/v2/folders/{id}`）と非表示ページスタブ出力に対応
- フロントマターに `parent_id` / `weight` / `is_folder` を追加
- Confluenceの並び順（position）を `weight` として再現
- 既存の出力には階層情報が無いため、反映には移行の再実行（`make sync-and-build` など）が必要

- [ ] **Step 4: 全テスト・lint・ビルドの最終確認**

```bash
make test && make lint && make build
cd hugo-site && hugo --quiet --destination /tmp/hugo-check-final
```

Expected: すべてエラーなし

- [ ] **Step 5: コミットしてPRを作成**

```bash
git add TODO.md CHANGELOG.md
git commit -m "docs: TODO.md/CHANGELOG.mdに左サイドバー階層化を記録

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
git push -u origin feature/sidebar-hierarchy
gh pr create --title "feat: 左サイドバーをConfluence同様の階層ツリーにする（フォルダ対応）" --body "$(cat <<'EOF'
## 概要
左サイドバーのページ一覧を、Confluenceと同じ親子階層の開閉式ツリーに変更しました。Confluenceのフォルダにも対応します。

設計: `docs/superpowers/specs/2026-08-08-sidebar-hierarchy-design.md`
計画: `docs/superpowers/plans/2026-08-08-sidebar-hierarchy.md`

## 変更内容
### Goツール
- `GetFolder`（`GET /wiki/api/v2/folders/{id}`）を追加し、`Page` に `parentType` / `position` を追加
- `CollectFolders`: ページの親を辿って必要なフォルダだけを再帰収集（取得失敗はスキップして続行）
- ページのフロントマターに `parent_id` / `weight` を出力
- フォルダを `build.render = "never"` の非表示ページスタブとして出力（URLもHTMLも生成されない）
- 中間ファイルに `parent_type` / `position` を保存し、`convert` 経路でも階層を維持

### テーマ（サブモジュール）
- 再帰パーシャルによる `<details>`/`<summary>` の開閉式ツリー（JS不要）
- 現在ページの祖先を自動展開、現在ページをハイライト
- フォルダはリンクせず開閉のみ

## 設計上の決定
- URL・ディレクトリ構造はフラットのまま維持（既存リンクを壊さないため）
- 並び順はConfluenceのpositionを `weight = position + 1` として再現（未取得は9999で末尾）
- 親が解決できないページはルート直下にフォールバックし、警告ログを出す

## 制限事項
- ページを1つも含まない空フォルダはサイドバーに表示されない（ページの親を辿る収集方式のため）
- 既存の出力には階層情報が無いため、反映には移行の再実行が必要

## テスト
- `GetFolder` / `CollectFolders` / フロントマター生成 / フォルダスタブ出力 / 中間ファイル往復のユニットテストを追加
- `hugo-site/content/sample/` に階層フィクスチャを追加し、階層描画・自動展開・フォルダ非生成・ルートフォールバックを検証

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## Self-Review

**スペック網羅性チェック**

| スペックの要求 | 対応タスク |
|---|---|
| `parentType` / `position` の取得 | Task 1 |
| `GetFolder` の新設 | Task 1 |
| ページの親を辿るフォルダ列挙（フォルダの親がフォルダでも再帰） | Task 2 |
| ページのフロントマターに `parent_id` / `weight` | Task 3 |
| フォルダの非表示ページスタブ（`is_folder`、`build.render = never`） | Task 4 |
| フォルダのディレクトリ名衝突時のIDサフィックス | Task 4 |
| `weight = position + 1`（最低1）、未取得は9999 | Task 3（`frontMatterWeight`）、Task 4で再利用 |
| 再帰パーシャルによるツリー描画 | Task 7 |
| `weight` 順ソート | Task 7（二段ソート） |
| `<details>`/`<summary>` の開閉、フォルダはリンクなし | Task 7 |
| 現在ページの祖先の自動展開・現在ページ強調 | Task 7 |
| 親が解決できない場合のルートフォールバック | Task 2（警告ログ）、Task 7（ルート判定）、Task 8（検証） |
| 空フォルダは表示されない（制限事項として明記） | Task 9（PR本文・CHANGELOG） |
| 既存データは移行の再実行が必要 | Task 9（ドキュメント） |
| Goユニットテスト（httptestモック） | Task 1、2、3、4、5 |
| Hugoフィクスチャと目視確認 | Task 8 |

スペックの `convert` 経路への言及は無かったが、`parent_id` を出力する以上その経路で階層が欠落するのは実質的な不具合になるため、Task 5として追加した。

**プレースホルダ確認**: TBD・TODO・「適切に処理する」といった曖昧な指示は無し。全コードステップに実コードを記載済み。Task 9のStep 3のみ既存ファイルの記法に合わせる必要があるため記載内容を箇条書きで指定している。

**型・名前の整合性**:
- `Folder` の全フィールド（`ID`/`Title`/`Status`/`ParentID`/`ParentType`/`Position`）はTask 1で定義し、Task 2・4・6で同名で使用
- `Position *int` はTask 1・3・4・5で一貫
- `frontMatterWeight` はTask 3で定義しTask 4で再利用
- `intPtr` ヘルパーはTask 1のテストで定義し、Task 3・5のテストで再利用（同一パッケージ内のため参照可能。Task 1を先に実装する前提）
- `CollectFolders(getter, pages)` の引数順はTask 2の定義とTask 6の呼び出しで一致
- テンプレートの `dict` キー（`nodes`/`pages`/`ancestors`/`current`/`depth`）はTask 7の2ファイル間で一致
- CSSクラス名（`tree-item`/`tree-link`/`tree-label`/`is-folder`/`is-current`/`is-leaf`）はTask 7のテンプレートとCSSで一致
