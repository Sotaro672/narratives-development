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

	// ★ 追加: Arweave / Bundlr / Irys 用設定
	// 例) https://node1.bundlr.network や 自前の Cloud Run ラッパ API の URL
	ArweaveBaseURL string
	// 認証が必要な場合に使用（不要なら空でOK）
	ArweaveAPIKey string
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

		// ★ Arweave / Bundlr / Irys 関連
		// 環境変数が未設定なら空文字のまま → Arweave 連携はスキップされる
		ArweaveBaseURL: os.Getenv("ARWEAVE_BASE_URL"),
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
