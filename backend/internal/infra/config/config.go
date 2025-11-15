// backend/internal/infra/config/config.go
package config

import "os"

// Config はアプリケーション全体の環境変数設定を保持します。
type Config struct {
	GCSBucket                string
	GCPCreds                 string
	Port                     string
	FirestoreProjectID       string
	FirestoreCredentialsFile string

	// ★ 追加: Firebase Auth 用のプロジェクトID
	FirebaseProjectID string
}

// Load は環境変数を読み込み Config を返します。
func Load() *Config {
	// ベースとなる GCP プロジェクト ID
	defaultProject := getenvDefault("GCP_PROJECT_ID", "narratives-development-26c2d")

	cfg := &Config{
		GCSBucket:                os.Getenv("GCS_BUCKET"),
		GCPCreds:                 os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		Port:                     getenvDefault("PORT", "8080"),
		FirestoreProjectID:       getenvDefault("FIRESTORE_PROJECT_ID", defaultProject),
		FirestoreCredentialsFile: os.Getenv("FIRESTORE_CREDENTIALS_FILE"),

		// ★ FIREBASE_PROJECT_ID が未指定なら GCP のデフォルトを使う
		FirebaseProjectID: getenvDefault("FIREBASE_PROJECT_ID", defaultProject),
	}

	return cfg
}

// GetFirestoreProjectID は Firestore/GCP プロジェクト ID を返します。
func (c *Config) GetFirestoreProjectID() string {
	return c.FirestoreProjectID
}

// Firebase 用の ProjectID を返すヘルパー（あると便利）
func (c *Config) GetFirebaseProjectID() string {
	return c.FirebaseProjectID
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
