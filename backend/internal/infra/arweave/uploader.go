// backend/internal/infra/arweave/uploader.go
package arweave

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Bundlr や Irys の HTTP API を叩く実装例
type HTTPUploader struct {
	client  *http.Client
	baseURL string // 例: "https://node1.bundlr.network"
	apiKey  string // 認証が必要な場合に使用
}

// NewHTTPUploader は Arweave/Bundlr 用の HTTP uploader を生成します。
func NewHTTPUploader(baseURL, apiKey string) *HTTPUploader {
	return &HTTPUploader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// UploadJSON は metadataJSON を Arweave にアップロードし、その URL を返します。
func (u *HTTPUploader) UploadJSON(ctx context.Context, metadataJSON []byte) (string, error) {
	if len(metadataJSON) == 0 {
		return "", fmt.Errorf("metadataJSON is empty")
	}

	// 実際のエンドポイントは使うサービスに合わせて調整してください。
	// （例）POST /upload/json など
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.baseURL+"/upload/json", bytes.NewReader(metadataJSON))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if u.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+u.apiKey)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload metadata to arweave: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload metadata failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var res struct {
		URI string `json:"uri"` // 例: "https://arweave.net/xxxx"
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode upload response: %w", err)
	}
	if res.URI == "" {
		return "", fmt.Errorf("upload response has empty uri")
	}

	return res.URI, nil
}
