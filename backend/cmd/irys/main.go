// backend\cmd\irys\main.go
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"narratives/internal/infra/arweave"
)

func main() {
	baseURL := os.Getenv("ARWEAVE_BASE_URL")
	if baseURL == "" {
		log.Fatal("ARWEAVE_BASE_URL is empty")
	}

	u := arweave.NewHTTPUploader(baseURL, "")

	payload := map[string]any{
		"hello": "from backend debug",
		"ts":    time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal json: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("[debug-irys] UploadMetadata to %s ...", baseURL)
	uri, err := u.UploadMetadata(ctx, data)
	if err != nil {
		log.Fatalf("UploadMetadata failed: %v", err)
	}

	log.Printf("[debug-irys] OK uri=%s", uri)
}
