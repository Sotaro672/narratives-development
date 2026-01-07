// backend/internal/platform/di/shared/infra.go
package shared

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"

	arweaveinfra "narratives/internal/infra/arweave"
	appcfg "narratives/internal/infra/config"
	solanainfra "narratives/internal/infra/solana"

	uc "narratives/internal/application/usecase"
)

// Infra is shared runtime infrastructure for DI.
// - owns external clients (Firestore/FirebaseAuth/GCS)
// - owns cross-cutting infra services (mint authority key, optional arweave uploader)
// - owns bucket names resolved from env/config
//
// IMPORTANT:
// Infra must NOT depend on console/mall routers, handlers, or queries.
type Infra struct {
	// Config
	Config    *appcfg.Config
	ProjectID string

	// Clients
	Firestore    *firestore.Client
	GCS          *storage.Client
	FirebaseApp  *firebase.App
	FirebaseAuth *firebaseauth.Client

	// Cross-cutting infra
	MintAuthorityKey *solanainfra.MintAuthorityKey
	ArweaveUploader  uc.ArweaveUploader

	// Buckets (resolved once)
	TokenIconBucket     string
	TokenContentsBucket string
	ListImageBucket     string
	AvatarIconBucket    string
	PostImageBucket     string
}

// NewInfra initializes shared infra.
// Firestore/GCS are strict (return error).
// Firebase/Auth and MintAuthorityKey are best-effort (warn + continue).
func NewInfra(ctx context.Context) (*Infra, error) {
	cfg := appcfg.Load()
	if cfg == nil {
		return nil, errors.New("shared.infra: config is nil")
	}

	projectID := resolveProjectID(cfg)
	if projectID == "" {
		// ここが空だと Firestore/NewApp とも不安定になるため、必ず error にする
		return nil, errors.New("shared.infra: projectID is empty (set FIRESTORE_PROJECT_ID or GOOGLE_CLOUD_PROJECT)")
	}

	inf := &Infra{
		Config:    cfg,
		ProjectID: projectID,
	}

	// credentials file (optional; mainly for local dev)
	credFile := strings.TrimSpace(cfg.FirestoreCredentialsFile)
	if credFile == "" {
		credFile = strings.TrimSpace(cfg.GCPCreds) // GOOGLE_APPLICATION_CREDENTIALS
	}
	var clientOpts []option.ClientOption
	if credFile != "" {
		clientOpts = append(clientOpts, option.WithCredentialsFile(credFile))
		log.Printf("[shared.infra] Using credentials file for GCP clients: %s", redactPath(credFile))
	} else {
		log.Printf("[shared.infra] Using Application Default Credentials (no credentials file configured)")
	}

	// 1) Optional: Arweave uploader (used by TokenBlueprintUsecase)
	if strings.TrimSpace(cfg.ArweaveBaseURL) != "" {
		inf.ArweaveUploader = arweaveinfra.NewHTTPUploader(cfg.ArweaveBaseURL, cfg.ArweaveAPIKey)
		log.Printf("[shared.infra] Arweave HTTPUploader initialized baseURL=%s", cfg.ArweaveBaseURL)
	} else {
		log.Printf("[shared.infra] Arweave HTTPUploader not configured (ARWEAVE_BASE_URL empty)")
	}

	// 2) Optional: Solana mint authority key (Secret Manager)
	{
		mintKey, err := solanainfra.LoadMintAuthorityKey(
			ctx,
			inf.ProjectID,
			"narratives-solana-mint-authority",
		)
		if err != nil {
			log.Printf("[shared.infra] WARN: failed to load mint authority key: %v", err)
			mintKey = nil
		}
		inf.MintAuthorityKey = mintKey
	}

	// 3) Firestore (strict)
	{
		var fsClient *firestore.Client
		var err error
		if len(clientOpts) > 0 {
			fsClient, err = firestore.NewClient(ctx, inf.ProjectID, clientOpts...)
		} else {
			fsClient, err = firestore.NewClient(ctx, inf.ProjectID)
		}
		if err != nil {
			return nil, fmt.Errorf("shared.infra: firestore.NewClient failed (project=%s): %w", inf.ProjectID, err)
		}
		inf.Firestore = fsClient
		log.Printf("[shared.infra] Firestore connected project=%s", inf.ProjectID)
	}

	// 4) GCS (strict)
	{
		var gcsClient *storage.Client
		var err error
		if len(clientOpts) > 0 {
			gcsClient, err = storage.NewClient(ctx, clientOpts...)
		} else {
			gcsClient, err = storage.NewClient(ctx)
		}
		if err != nil {
			_ = inf.Firestore.Close()
			return nil, fmt.Errorf("shared.infra: storage.NewClient failed: %w", err)
		}
		inf.GCS = gcsClient
		log.Printf("[shared.infra] GCS storage client initialized")
	}

	// 5) Firebase App/Auth (best-effort)
	{
		var fbApp *firebase.App
		var err error

		fbCfg := &firebase.Config{ProjectID: inf.ProjectID}
		if len(clientOpts) > 0 {
			fbApp, err = firebase.NewApp(ctx, fbCfg, clientOpts...)
		} else {
			fbApp, err = firebase.NewApp(ctx, fbCfg)
		}

		if err != nil {
			log.Printf("[shared.infra] WARN: firebase app init failed: %v", err)
		} else {
			inf.FirebaseApp = fbApp
			authClient, err := fbApp.Auth(ctx)
			if err != nil {
				log.Printf("[shared.infra] WARN: firebase auth init failed: %v", err)
			} else {
				inf.FirebaseAuth = authClient
				log.Printf("[shared.infra] Firebase Auth initialized")
			}
		}
	}

	// 6) Buckets (resolve once)
	// Token buckets
	inf.TokenIconBucket = strings.TrimSpace(cfg.TokenIconBucket)
	if inf.TokenIconBucket == "" {
		// ここは “空のまま進むと後で失敗が分かりづらい” ので明示的に WARN
		log.Printf("[shared.infra] WARN: TOKEN_ICON_BUCKET is empty (token icon features may fail)")
	}
	inf.TokenContentsBucket = strings.TrimSpace(cfg.TokenContentsBucket)
	if inf.TokenContentsBucket == "" {
		// 既存互換: env fallback + default
		inf.TokenContentsBucket = getenvOrDefault("TOKEN_CONTENTS_BUCKET", "narratives-development-token")
	}
	if inf.TokenContentsBucket == "" {
		log.Printf("[shared.infra] WARN: TOKEN_CONTENTS_BUCKET is empty (token contents features may fail)")
	}

	// List images bucket:
	// deploy-backend.ps1 は LIST_BUCKET を渡しているため、まず LIST_BUCKET を見る。
	// 旧名 LIST_IMAGE_BUCKET がある場合も拾う。
	inf.ListImageBucket = strings.TrimSpace(os.Getenv("LIST_BUCKET"))
	if inf.ListImageBucket == "" {
		inf.ListImageBucket = strings.TrimSpace(os.Getenv("LIST_IMAGE_BUCKET"))
	}
	if inf.ListImageBucket == "" {
		log.Printf("[shared.infra] WARN: LIST_BUCKET/LIST_IMAGE_BUCKET is empty (list image features may fail)")
	}

	// Avatar/Post buckets
	inf.AvatarIconBucket = getenvOrDefault("AVATAR_ICON_BUCKET", "narratives-development_avatar_icon")
	inf.PostImageBucket = getenvOrDefault("POST_IMAGE_BUCKET", "narratives-development-posts")

	// Final sanity (panic 防止のための最終チェック)
	if inf.Firestore == nil {
		_ = inf.Close()
		return nil, errors.New("shared.infra: firestore client is nil after initialization (unexpected)")
	}
	if inf.GCS == nil {
		_ = inf.Close()
		return nil, errors.New("shared.infra: gcs client is nil after initialization (unexpected)")
	}

	return inf, nil
}

func (i *Infra) Close() error {
	if i == nil {
		return nil
	}
	if i.Firestore != nil {
		_ = i.Firestore.Close()
	}
	if i.GCS != nil {
		_ = i.GCS.Close()
	}
	return nil
}

func resolveProjectID(cfg *appcfg.Config) string {
	// 優先順位:
	// 1) cfg.FirestoreProjectID（config.Load で解決済み想定）
	// 2) FIRESTORE_PROJECT_ID
	// 3) GCP_PROJECT_ID
	// 4) GOOGLE_CLOUD_PROJECT（Cloud Run ではこれが入ってることが多い）
	// 5) FIREBASE_PROJECT_ID（保険）
	if cfg != nil {
		if v := strings.TrimSpace(cfg.FirestoreProjectID); v != "" {
			return v
		}
	}

	for _, k := range []string{
		"FIRESTORE_PROJECT_ID",
		"GCP_PROJECT_ID",
		"GOOGLE_CLOUD_PROJECT",
		"FIREBASE_PROJECT_ID",
	} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}

	return ""
}

func getenvOrDefault(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func redactPath(p string) string {
	// ログにパス全文を出さない（Windows/Unix 両対応の軽いマスク）
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	// 最後の要素だけ残す
	p = strings.ReplaceAll(p, "\\", "/")
	parts := strings.Split(p, "/")
	if len(parts) == 0 {
		return "***"
	}
	last := parts[len(parts)-1]
	if last == "" {
		return "***"
	}
	return "***" + "/" + last
}
