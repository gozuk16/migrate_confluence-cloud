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
