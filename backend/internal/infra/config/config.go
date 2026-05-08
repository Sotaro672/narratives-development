// backend/internal/infra/config/config.go
package config

import "os"

// Config はアプリケーション全体の環境変数設定を保持します。
//
// GCS bucket 関連は廃止済み。
// Stripe key は Google Secret Manager を正とするため、ここでは読み込まない。
type Config struct {
	GCPCreds                 string
	Port                     string
	FirestoreProjectID       string
	FirestoreCredentialsFile string

	// Firebase Auth 用のプロジェクトID
	FirebaseProjectID string

	// Arweave / Bundlr / Irys 用設定
	ArweaveBaseURL string
	ArweaveAPIKey  string
}

// Load は環境変数を読み込み Config を返します。
func Load() *Config {
	// ベースとなる GCP プロジェクト ID
	defaultProject := getenvDefault("GCP_PROJECT_ID", "narratives-development-26c2d")

	cfg := &Config{
		GCPCreds:                 os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		Port:                     getenvDefault("PORT", "8080"),
		FirestoreProjectID:       getenvDefault("FIRESTORE_PROJECT_ID", defaultProject),
		FirestoreCredentialsFile: os.Getenv("FIRESTORE_CREDENTIALS_FILE"),

		// FIREBASE_PROJECT_ID が未指定なら GCP のデフォルトを使う
		FirebaseProjectID: getenvDefault("FIREBASE_PROJECT_ID", defaultProject),

		// Arweave / Bundlr / Irys 関連
		ArweaveBaseURL: os.Getenv("ARWEAVE_BASE_URL"),
		ArweaveAPIKey:  os.Getenv("ARWEAVE_API_KEY"),
	}

	return cfg
}

// GetFirestoreProjectID は Firestore/GCP プロジェクト ID を返します。
func (c *Config) GetFirestoreProjectID() string {
	return c.FirestoreProjectID
}

// Firebase 用の ProjectID を返すヘルパー
func (c *Config) GetFirebaseProjectID() string {
	return c.FirebaseProjectID
}

// Arweave / Bundlr / Irys のベース URL を返すヘルパー
func (c *Config) GetArweaveBaseURL() string {
	return c.ArweaveBaseURL
}

// Arweave 用の API Key を返すヘルパー
func (c *Config) GetArweaveAPIKey() string {
	return c.ArweaveAPIKey
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return def
}
