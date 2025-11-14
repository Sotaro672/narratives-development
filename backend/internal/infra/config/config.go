// backend/internal/infra/config/config.go
package config

import (
	"os"
)

// Config はアプリケーション全体の環境変数設定を保持します。
// Firestore / GCS 前提のため DATABASE_URL は削除してあります。
type Config struct {
	GCSBucket                string
	GCPCreds                 string
	Port                     string
	FirestoreProjectID       string
	FirestoreCredentialsFile string
}

// Load は環境変数を読み込み Config を返します。
// Firestore と GCS の設定を統一的に管理します。
func Load() *Config {
	cfg := &Config{
		GCSBucket:                os.Getenv("GCS_BUCKET"),
		GCPCreds:                 os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		Port:                     getenvDefault("PORT", "8080"),
		FirestoreProjectID:       getenvDefault("FIRESTORE_PROJECT_ID", "narratives-development-26c2d"),
		FirestoreCredentialsFile: os.Getenv("FIRESTORE_CREDENTIALS_FILE"), // 空文字なら ADC を使用
	}

	return cfg
}

// GetFirestoreProjectID は Firestore/GCP プロジェクト ID を返します。
func (c *Config) GetFirestoreProjectID() string {
	return c.FirestoreProjectID
}

// getenvDefault は環境変数が未設定のときにデフォルト値を返します。
func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
