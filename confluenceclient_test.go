package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestConfluenceClient はテスト用のConfluenceクライアントを作成する
func newTestConfluenceClient(serverURL string) *ConfluenceClient {
	return NewConfluenceClient(serverURL, "test@example.com", "test-token")
}

// TestGetPage はGetPageのテスト
func TestGetPage(t *testing.T) {
	tests := []struct {
		name        string
		pageID      string
		handler     http.HandlerFunc
		wantErr     bool
		errContains string
		wantTitle   string
	}{
		{
			name:   "正常系: ページ取得成功",
			pageID: "12345",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/wiki/api/v2/pages/12345" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(Page{
					ID:      "12345",
					Title:   "テストページ",
					SpaceID: "67890",
					Body: PageBody{
						Storage: Storage{
							Value:          "<p>テストコンテンツ</p>",
							Representation: "storage",
						},
					},
					Version: Version{
						Number:    1,
						CreatedAt: "2024-01-01T00:00:00.000Z",
					},
				})
			},
			wantErr:   false,
			wantTitle: "テストページ",
		},
		{
			name:   "異常系: ページが存在しない",
			pageID: "99999",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, `{"message":"Page not found"}`, http.StatusNotFound)
			},
			wantErr:     true,
			errContains: "confluence API error: 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := newTestConfluenceClient(server.URL)
			page, err := client.GetPage(tt.pageID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("エラーが期待されましたが、nilが返されました")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("エラーメッセージが期待と異なります\n期待: %q を含む\n実際: %q",
						tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("予期しないエラー: %v", err)
				return
			}

			if page.Title != tt.wantTitle {
				t.Errorf("タイトルが期待と異なります\n期待: %q\n実際: %q", tt.wantTitle, page.Title)
			}
		})
	}
}

// TestGetSpaceByKey はGetSpaceByKeyのテスト
func TestGetSpaceByKey(t *testing.T) {
	tests := []struct {
		name        string
		spaceKey    string
		handler     http.HandlerFunc
		wantErr     bool
		errContains string
		wantKey     string
	}{
		{
			name:     "正常系: スペース取得成功",
			spaceKey: "TEST",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/wiki/api/v2/spaces" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(SpaceListResponse{
					Results: []Space{
						{
							ID:   "67890",
							Key:  "TEST",
							Name: "テストスペース",
							Type: "global",
						},
					},
					Links: Links{},
				})
			},
			wantErr: false,
			wantKey: "TEST",
		},
		{
			name:     "異常系: スペースが見つからない",
			spaceKey: "NOTFOUND",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(SpaceListResponse{
					Results: []Space{},
					Links:   Links{},
				})
			},
			wantErr:     true,
			errContains: "スペースが見つかりません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := newTestConfluenceClient(server.URL)
			space, err := client.GetSpaceByKey(tt.spaceKey)

			if tt.wantErr {
				if err == nil {
					t.Errorf("エラーが期待されましたが、nilが返されました")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("エラーメッセージが期待と異なります\n期待: %q を含む\n実際: %q",
						tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("予期しないエラー: %v", err)
				return
			}

			if space.Key != tt.wantKey {
				t.Errorf("スペースキーが期待と異なります\n期待: %q\n実際: %q", tt.wantKey, space.Key)
			}
		})
	}
}

// TestGetPages はGetPagesのテスト（ページネーション含む）
func TestGetPages(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		cursor := r.URL.Query().Get("cursor")
		if cursor == "" {
			// 1ページ目
			json.NewEncoder(w).Encode(PageListResponse{
				Results: []Page{
					{ID: "1", Title: "ページ1", SpaceID: "space1"},
					{ID: "2", Title: "ページ2", SpaceID: "space1"},
				},
				Links: Links{Next: "/wiki/api/v2/pages?space-id=space1&cursor=next-cursor"},
			})
		} else {
			// 2ページ目（最終）
			json.NewEncoder(w).Encode(PageListResponse{
				Results: []Page{
					{ID: "3", Title: "ページ3", SpaceID: "space1"},
				},
				Links: Links{},
			})
		}
	}))
	defer server.Close()

	client := newTestConfluenceClient(server.URL)
	pages, err := client.GetPages("space1")

	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
		return
	}

	if len(pages) != 3 {
		t.Errorf("ページ数が期待と異なります\n期待: 3\n実際: %d", len(pages))
	}

	if callCount != 2 {
		t.Errorf("APIコール数が期待と異なります\n期待: 2\n実際: %d", callCount)
	}
}

// TestGetPageLabels はGetPageLabelsのテスト
func TestGetPageLabels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/labels") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LabelListResponse{
			Results: []Label{
				{ID: "1", Name: "golang", Prefix: "global"},
				{ID: "2", Name: "backend", Prefix: "global"},
			},
			Links: Links{},
		})
	}))
	defer server.Close()

	client := newTestConfluenceClient(server.URL)
	labels, err := client.GetPageLabels("12345")

	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
		return
	}

	if len(labels) != 2 {
		t.Errorf("ラベル数が期待と異なります\n期待: 2\n実際: %d", len(labels))
	}

	if labels[0].Name != "golang" {
		t.Errorf("ラベル名が期待と異なります\n期待: %q\n実際: %q", "golang", labels[0].Name)
	}
}

// TestExtractCursor はextractCursorのテスト
func TestExtractCursor(t *testing.T) {
	tests := []struct {
		name       string
		nextURL    string
		wantCursor string
	}{
		{
			name:       "正常系: cursorパラメータが存在する",
			nextURL:    "/wiki/api/v2/pages?space-id=123&limit=250&cursor=abc123",
			wantCursor: "abc123",
		},
		{
			name:       "正常系: cursorパラメータが存在しない",
			nextURL:    "/wiki/api/v2/pages?space-id=123",
			wantCursor: "",
		},
		{
			name:       "異常系: 無効なURL",
			nextURL:    "://invalid",
			wantCursor: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := extractCursor(tt.nextURL)
			if cursor != tt.wantCursor {
				t.Errorf("カーソルが期待と異なります\n期待: %q\n実際: %q", tt.wantCursor, cursor)
			}
		})
	}
}
