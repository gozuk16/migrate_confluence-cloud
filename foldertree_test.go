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

	t.Run("ParentTypeが空でParentIDがページID集合に含まれない場合はフォルダとして収集する", func(t *testing.T) {
		// Confluence REST API v2 のページ一覧が parentType を返さないケースを想定。
		// 親IDがこのスペースのページ集合に無いため、フォルダとして解決を試みる。
		getter := &fakeFolderGetter{folders: map[string]*Folder{
			"10": {ID: "10", Title: "設計", ParentType: "page", ParentID: "1"},
		}}
		pages := []Page{
			{ID: "1", Title: "Home"},
			{ID: "2", Title: "子", ParentID: "10", ParentType: ""},
		}

		folders := CollectFolders(getter, pages)

		if len(folders) != 1 {
			t.Fatalf("フォルダ数 = %d, want 1", len(folders))
		}
		if folders[0].ID != "10" || folders[0].Title != "設計" {
			t.Errorf("folders[0] = %+v, want ID=10 Title=設計", folders[0])
		}
	})

	t.Run("ParentTypeが空でParentIDがページID集合に含まれる場合はフォルダとして収集しない", func(t *testing.T) {
		// 親IDが同じスペースのページ集合に含まれる場合は、親がページだと判定できるため
		// フォルダ解決を試みない。
		getter := &fakeFolderGetter{folders: map[string]*Folder{}}
		pages := []Page{
			{ID: "1", Title: "Home"},
			{ID: "2", Title: "子", ParentID: "1", ParentType: ""},
		}

		folders := CollectFolders(getter, pages)

		if len(folders) != 0 {
			t.Fatalf("フォルダ数 = %d, want 0", len(folders))
		}
		if len(getter.calls) != 0 {
			t.Errorf("GetFolderが呼ばれました: %v", getter.calls)
		}
	})

	t.Run("ParentTypeがpageの場合は従来どおり収集しない", func(t *testing.T) {
		// ParentIDがページID集合に含まれない場合でも、ParentTypeが明示的に
		// "page" ならフォルダ解決を試みてはならない。
		getter := &fakeFolderGetter{folders: map[string]*Folder{}}
		pages := []Page{
			{ID: "2", Title: "子", ParentID: "999", ParentType: "page"},
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
