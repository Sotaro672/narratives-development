package config

import (
	"log"
	"os"
)

// Config はアプリケーション全体の環境変数設定を保持します。
type Config struct {
	DatabaseURL              string
	GCSBucket                string
	GCPCreds                 string
	Port                     string
	FirestoreProjectID       string
	FirestoreCredentialsFile string
}

// Load は環境変数を読み込み Config を返します。
// Firestore や GCS の設定も統一的に扱います。
func Load() *Config {
	cfg := &Config{
		DatabaseURL:              os.Getenv("DATABASE_URL"),
		GCSBucket:                os.Getenv("GCS_BUCKET"),
		GCPCreds:                 os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		Port:                     getenvDefault("PORT", "8080"),
		FirestoreProjectID:       getenvDefault("FIRESTORE_PROJECT_ID", "narratives-development-26c2d"),
		FirestoreCredentialsFile: os.Getenv("FIRESTORE_CREDENTIALS_FILE"), // 空文字ならADCを使う
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is missing")
	}

	return cfg
}

// getenvDefault は環境変数が未設定のときにデフォルト値を返します。
func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
