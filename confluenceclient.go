package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// ConfluenceClient はConfluence REST API v2クライアント
type ConfluenceClient struct {
	baseURL    string
	email      string
	apiToken   string
	httpClient *http.Client
	spaceCache *SpaceCache
	userCache  *UserCache
}

// SpaceCache はスペース情報のキャッシュ
type SpaceCache struct {
	mu     sync.RWMutex
	spaces map[string]*Space // key -> Space
}

// UserCache はユーザー情報のキャッシュ
type UserCache struct {
	mu    sync.RWMutex
	users map[string]string // accountId -> displayName
}

// Page はConfluenceページ情報
type Page struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Status   string   `json:"status"`
	SpaceID  string   `json:"spaceId"`
	ParentID string   `json:"parentId"`
	Body     PageBody `json:"body"`
	Version  Version  `json:"version"`
	Links    Links    `json:"_links"`
}

// PageBody はページのボディコンテンツ
type PageBody struct {
	Storage        Storage        `json:"storage"`
	AtlasDocFormat AtlasDocFormat `json:"atlas_doc_format"`
}

// AtlasDocFormat は ADF 形式のコンテンツ（value はさらに JSON 文字列）
type AtlasDocFormat struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

// Storage はStorage Format（XHTML）のコンテンツ
type Storage struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

// Version はページのバージョン情報
type Version struct {
	Number    int    `json:"number"`
	CreatedAt string `json:"createdAt"`
	AuthorID  string `json:"authorId"`
}

// Links はAPIレスポンスのリンク情報
type Links struct {
	Base     string `json:"base"`
	Next     string `json:"next"`
	WebUI    string `json:"webui"`
	Download string `json:"download"`
}

// Space はConfluenceスペース情報
type Space struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Attachment はConfluenceの添付ファイル情報
type Attachment struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Status    string  `json:"status"`
	FileSize  int64   `json:"fileSize"`
	MediaType string  `json:"mediaType"`
	PageID    string  `json:"pageId"`
	Version   Version `json:"version"`
	Links     Links   `json:"_links"`
}

// Comment はConfluenceのコメント情報
type Comment struct {
	ID      string      `json:"id"`
	Status  string      `json:"status"`
	Body    CommentBody `json:"body"`
	Version Version     `json:"version"`
	PageID  string      `json:"pageId"`
}

// CommentBody はコメントのボディコンテンツ
type CommentBody struct {
	Storage Storage `json:"storage"`
}

// Label はConfluenceのラベル情報
type Label struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Prefix string `json:"prefix"`
}

// PageListResponse はページ一覧のAPIレスポンス
type PageListResponse struct {
	Results []Page `json:"results"`
	Links   Links  `json:"_links"`
}

// SpaceListResponse はスペース一覧のAPIレスポンス
type SpaceListResponse struct {
	Results []Space `json:"results"`
	Links   Links   `json:"_links"`
}

// AttachmentListResponse は添付ファイル一覧のAPIレスポンス
type AttachmentListResponse struct {
	Results []Attachment `json:"results"`
	Links   Links        `json:"_links"`
}

// CommentListResponse はコメント一覧のAPIレスポンス
type CommentListResponse struct {
	Results []Comment `json:"results"`
	Links   Links     `json:"_links"`
}

// LabelListResponse はラベル一覧のAPIレスポンス
type LabelListResponse struct {
	Results []Label `json:"results"`
	Links   Links   `json:"_links"`
}

// NewConfluenceClient は新しいConfluenceクライアントを作成
func NewConfluenceClient(baseURL, email, apiToken string) *ConfluenceClient {
	return &ConfluenceClient{
		baseURL:  baseURL,
		email:    email,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		spaceCache: &SpaceCache{
			spaces: make(map[string]*Space),
		},
		userCache: &UserCache{
			users: make(map[string]string),
		},
	}
}

// doRequest はHTTPリクエストを実行する共通メソッド
func (cc *ConfluenceClient) doRequest(method, apiURL string) ([]byte, error) {
	req, err := http.NewRequest(method, apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(cc.email, cc.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := cc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("confluence API error: %d %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// GetPage は単一ページを ADF（Atlas Doc Format）で取得する
func (cc *ConfluenceClient) GetPage(pageID string) (*Page, error) {
	apiURL := fmt.Sprintf("%s/wiki/api/v2/pages/%s?body-format=atlas_doc_format", cc.baseURL, pageID)

	body, err := cc.doRequest("GET", apiURL)
	if err != nil {
		return nil, fmt.Errorf("ページ取得エラー (ID: %s): %w", pageID, err)
	}

	var page Page
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("ページJSONパースエラー (ID: %s): %w", pageID, err)
	}

	return &page, nil
}

// GetChildPages はページの子ページ一覧を取得する
func (cc *ConfluenceClient) GetChildPages(pageID string) ([]Page, error) {
	var allPages []Page
	cursor := ""

	for {
		apiURL := fmt.Sprintf("%s/wiki/api/v2/pages/%s/children?body-format=atlas_doc_format&limit=250", cc.baseURL, pageID)
		if cursor != "" {
			apiURL += "&cursor=" + url.QueryEscape(cursor)
		}

		body, err := cc.doRequest("GET", apiURL)
		if err != nil {
			return nil, fmt.Errorf("子ページ取得エラー (親ID: %s): %w", pageID, err)
		}

		var resp PageListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("子ページJSONパースエラー (親ID: %s): %w", pageID, err)
		}

		allPages = append(allPages, resp.Results...)

		if resp.Links.Next == "" {
			break
		}
		cursor = extractCursor(resp.Links.Next)
		if cursor == "" {
			break
		}
	}

	return allPages, nil
}

// GetSpaceByKey はスペースキーからスペース情報を取得する
func (cc *ConfluenceClient) GetSpaceByKey(key string) (*Space, error) {
	// キャッシュチェック
	cc.spaceCache.mu.RLock()
	if space, ok := cc.spaceCache.spaces[key]; ok {
		cc.spaceCache.mu.RUnlock()
		return space, nil
	}
	cc.spaceCache.mu.RUnlock()

	apiURL := fmt.Sprintf("%s/wiki/api/v2/spaces?keys=%s&limit=1", cc.baseURL, url.QueryEscape(key))

	body, err := cc.doRequest("GET", apiURL)
	if err != nil {
		return nil, fmt.Errorf("スペース取得エラー (key: %s): %w", key, err)
	}

	var resp SpaceListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("スペースJSONパースエラー (key: %s): %w", key, err)
	}

	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("スペースが見つかりません (key: %s)", key)
	}

	space := &resp.Results[0]

	// キャッシュに保存
	cc.spaceCache.mu.Lock()
	cc.spaceCache.spaces[key] = space
	cc.spaceCache.mu.Unlock()

	return space, nil
}

// GetSpaceByID はスペースIDからスペース情報を取得する
func (cc *ConfluenceClient) GetSpaceByID(spaceID string) (*Space, error) {
	apiURL := fmt.Sprintf("%s/wiki/api/v2/spaces/%s", cc.baseURL, spaceID)

	body, err := cc.doRequest("GET", apiURL)
	if err != nil {
		return nil, fmt.Errorf("スペース取得エラー (ID: %s): %w", spaceID, err)
	}

	var space Space
	if err := json.Unmarshal(body, &space); err != nil {
		return nil, fmt.Errorf("スペースJSONパースエラー (ID: %s): %w", spaceID, err)
	}

	return &space, nil
}

// GetPages はスペース内の全ページを取得する
func (cc *ConfluenceClient) GetPages(spaceID string) ([]Page, error) {
	var allPages []Page
	cursor := ""

	for {
		apiURL := fmt.Sprintf("%s/wiki/api/v2/pages?space-id=%s&limit=250&status=current", cc.baseURL, spaceID)
		if cursor != "" {
			apiURL += "&cursor=" + url.QueryEscape(cursor)
		}

		body, err := cc.doRequest("GET", apiURL)
		if err != nil {
			return nil, fmt.Errorf("ページ一覧取得エラー (spaceID: %s): %w", spaceID, err)
		}

		var resp PageListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("ページ一覧JSONパースエラー (spaceID: %s): %w", spaceID, err)
		}

		allPages = append(allPages, resp.Results...)

		if resp.Links.Next == "" {
			break
		}
		cursor = extractCursor(resp.Links.Next)
		if cursor == "" {
			break
		}
	}

	return allPages, nil
}

// GetPageAttachments はページの添付ファイル一覧を取得する
func (cc *ConfluenceClient) GetPageAttachments(pageID string) ([]Attachment, error) {
	var allAttachments []Attachment
	cursor := ""

	for {
		apiURL := fmt.Sprintf("%s/wiki/api/v2/pages/%s/attachments?limit=250", cc.baseURL, pageID)
		if cursor != "" {
			apiURL += "&cursor=" + url.QueryEscape(cursor)
		}

		body, err := cc.doRequest("GET", apiURL)
		if err != nil {
			return nil, fmt.Errorf("添付ファイル取得エラー (pageID: %s): %w", pageID, err)
		}

		var resp AttachmentListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("添付ファイルJSONパースエラー (pageID: %s): %w", pageID, err)
		}

		allAttachments = append(allAttachments, resp.Results...)

		if resp.Links.Next == "" {
			break
		}
		cursor = extractCursor(resp.Links.Next)
		if cursor == "" {
			break
		}
	}

	return allAttachments, nil
}

// GetPageFooterComments はページのフッターコメントを取得する
func (cc *ConfluenceClient) GetPageFooterComments(pageID string) ([]Comment, error) {
	var allComments []Comment
	cursor := ""

	for {
		apiURL := fmt.Sprintf("%s/wiki/api/v2/pages/%s/footer-comments?body-format=storage&limit=250", cc.baseURL, pageID)
		if cursor != "" {
			apiURL += "&cursor=" + url.QueryEscape(cursor)
		}

		body, err := cc.doRequest("GET", apiURL)
		if err != nil {
			return nil, fmt.Errorf("コメント取得エラー (pageID: %s): %w", pageID, err)
		}

		var resp CommentListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("コメントJSONパースエラー (pageID: %s): %w", pageID, err)
		}

		allComments = append(allComments, resp.Results...)

		if resp.Links.Next == "" {
			break
		}
		cursor = extractCursor(resp.Links.Next)
		if cursor == "" {
			break
		}
	}

	return allComments, nil
}

// GetPageLabels はページのラベル一覧を取得する
func (cc *ConfluenceClient) GetPageLabels(pageID string) ([]Label, error) {
	var allLabels []Label
	cursor := ""

	for {
		apiURL := fmt.Sprintf("%s/wiki/api/v2/pages/%s/labels?limit=250", cc.baseURL, pageID)
		if cursor != "" {
			apiURL += "&cursor=" + url.QueryEscape(cursor)
		}

		body, err := cc.doRequest("GET", apiURL)
		if err != nil {
			return nil, fmt.Errorf("ラベル取得エラー (pageID: %s): %w", pageID, err)
		}

		var resp LabelListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("ラベルJSONパースエラー (pageID: %s): %w", pageID, err)
		}

		allLabels = append(allLabels, resp.Results...)

		if resp.Links.Next == "" {
			break
		}
		cursor = extractCursor(resp.Links.Next)
		if cursor == "" {
			break
		}
	}

	return allLabels, nil
}

// GetUserDisplayName はアカウントIDからユーザーの表示名を取得する
func (cc *ConfluenceClient) GetUserDisplayName(accountID string, deletedUsers map[string]string) string {
	// キャッシュチェック
	cc.userCache.mu.RLock()
	if name, ok := cc.userCache.users[accountID]; ok {
		cc.userCache.mu.RUnlock()
		return name
	}
	cc.userCache.mu.RUnlock()

	// 削除済みユーザーのマッピングチェック
	if deletedUsers != nil {
		if name, ok := deletedUsers[accountID]; ok {
			cc.userCache.mu.Lock()
			cc.userCache.users[accountID] = name
			cc.userCache.mu.Unlock()
			return name
		}
	}

	// APIから取得
	apiURL := fmt.Sprintf("%s/wiki/rest/api/user?accountId=%s", cc.baseURL, url.QueryEscape(accountID))

	body, err := cc.doRequest("GET", apiURL)
	if err != nil {
		return accountID
	}

	var userResp struct {
		DisplayName string `json:"displayName"`
	}
	if err := json.Unmarshal(body, &userResp); err != nil || userResp.DisplayName == "" {
		return accountID
	}

	// キャッシュに保存
	cc.userCache.mu.Lock()
	cc.userCache.users[accountID] = userResp.DisplayName
	cc.userCache.mu.Unlock()

	return userResp.DisplayName
}

// extractCursor はnext URLからcursorパラメータを抽出する
func extractCursor(nextURL string) string {
	parsed, err := url.Parse(nextURL)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("cursor")
}
