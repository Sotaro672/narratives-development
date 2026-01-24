// backend/internal/infra/arweave/uploader.go
package arweave

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Irys Uploader (Cloud Run) などの HTTP API を叩く実装
type HTTPUploader struct {
	client  *http.Client
	baseURL string // 例: "https://narratives-irys-uploader-xxxx.asia-northeast1.run.app"
	apiKey  string // 認証が必要な場合に使用（IRYS_SERVICE_API_KEY など）
}

// NewHTTPUploader は Arweave/Irys 用の HTTP uploader を生成します。
func NewHTTPUploader(baseURL, apiKey string) *HTTPUploader {
	baseURL = strings.TrimSpace(baseURL)
	baseURL = strings.TrimRight(baseURL, "/") // 末尾の "/" を削っておく

	return &HTTPUploader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// ----------------------------------------------------------------------
// ArweaveUploader インターフェース実装
// ----------------------------------------------------------------------
//
// usecase.ArweaveUploader は:
//
//	UploadMetadata(ctx context.Context, data []byte) (string, error)
//
// を要求している想定なので、そのまま UploadJSON に委譲します。
func (u *HTTPUploader) UploadMetadata(ctx context.Context, data []byte) (string, error) {
	log.Printf("[arweave] UploadMetadata called (len=%d)", len(data))
	return u.UploadJSON(ctx, data)
}

// UploadJSON は metadataJSON を Irys Uploader 経由で Arweave にアップロードし、その URL を返します。
// 呼び出し側（TokenMetadataBuilder / Usecase）が JSON をエンコードした []byte を渡してくる前提です。
func (u *HTTPUploader) UploadJSON(ctx context.Context, metadataJSON []byte) (string, error) {
	if len(metadataJSON) == 0 {
		return "", fmt.Errorf("metadataJSON is empty")
	}

	if u.baseURL == "" {
		return "", fmt.Errorf("baseURL is empty; arweave endpoint not configured")
	}

	log.Printf("[arweave] UploadJSON start baseURL=%s", u.baseURL)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		u.baseURL+"/upload/json",
		bytes.NewReader(metadataJSON),
	)
	if err != nil {
		log.Printf("[arweave] create request FAILED err=%v", err)
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if u.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+u.apiKey)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		log.Printf("[arweave] http request FAILED err=%v", err)
		return "", fmt.Errorf("upload metadata to arweave: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf(
			"[arweave] upload metadata FAILED status=%d body=%s",
			resp.StatusCode,
			string(bodyBytes),
		)
		return "", fmt.Errorf("upload metadata failed: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var res struct {
		URI string `json:"uri"` // 例: "https://gateway.irys.xyz/xxxx"
	}
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		log.Printf("[arweave] decode upload response FAILED err=%v body=%s", err, string(bodyBytes))
		return "", fmt.Errorf("decode upload response: %w", err)
	}
	if res.URI == "" {
		log.Printf("[arweave] upload response has empty uri body=%s", string(bodyBytes))
		return "", fmt.Errorf("upload response has empty uri")
	}

	log.Printf("[arweave] UploadJSON OK uri=%s", res.URI)
	return res.URI, nil
}
