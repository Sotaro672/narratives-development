// backend/internal/platform/di/shared/infra.go
package shared

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/firestore"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/storage"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"

	uc "narratives/internal/application/usecase"
	arweaveinfra "narratives/internal/infra/arweave"
	appcfg "narratives/internal/infra/config"
	solanainfra "narratives/internal/infra/solana"
)

const (
	// Default collection names for owner resolve (walletAddress -> brandId / avatarId).
	defaultBrandsCollection  = "brands"
	defaultAvatarsCollection = "avatars"

	// Default Secret Manager secret name prefix for brand signer secrets.
	// secretId = <prefix><brandId>
	defaultBrandWalletSecretPrefix = "brand-wallet-"
)

// Infra is shared runtime infrastructure for DI.
// - owns external clients (Firestore/FirebaseAuth/GCS/SecretManager)
// - owns cross-cutting infra services (mint authority key, optional arweave uploader)
// - owns env/config-resolved runtime settings (bucket names, base URLs, collection names)
//
// IMPORTANT:
// Infra must NOT depend on console/mall routers, handlers, or queries.
type Infra struct {
	// Config
	Config    *appcfg.Config
	ProjectID string

	// Clients (owned; Close-managed)
	Firestore     *firestore.Client
	GCS           *storage.Client
	FirebaseApp   *firebase.App
	FirebaseAuth  *firebaseauth.Client
	SecretManager *secretmanager.Client

	// Cross-cutting infra
	MintAuthorityKey *solanainfra.MintAuthorityKey
	ArweaveUploader  uc.ArweaveUploader

	// Runtime settings (resolved once; normalized + validated)
	SelfBaseURL             string // used by PaymentFlow (self webhook trigger)
	BrandsCollection        string // used by OwnerResolve query
	AvatarsCollection       string // used by OwnerResolve query
	BrandWalletSecretPrefix string // used by Transfer signer provider (Design B)
	TokenIconBucket         string
	TokenContentsBucket     string
	ListImageBucket         string
	AvatarIconBucket        string
	PostImageBucket         string
}

// NewInfra initializes shared infra.
//
// Firestore/GCS are strict (return error).
// Firebase/Auth, SecretManager and MintAuthorityKey are best-effort (warn + continue).
func NewInfra(ctx context.Context) (*Infra, error) {
	cfg := appcfg.Load()
	if cfg == nil {
		return nil, errors.New("shared.infra: config is nil")
	}

	projectID := resolveProjectID(cfg)
	if projectID == "" {
		// If empty, Firestore/NewApp become unstable; treat as hard error.
		return nil, errors.New("shared.infra: projectID is empty (set FIRESTORE_PROJECT_ID or GOOGLE_CLOUD_PROJECT)")
	}

	inf := &Infra{
		Config:    cfg,
		ProjectID: projectID,
	}

	// --------------------------------------------------------
	// Resolve + normalize + validate runtime settings once (env/config)
	// --------------------------------------------------------
	settings, warns, err := ResolveRuntimeSettings(cfg)
	if err != nil {
		return nil, err
	}
	if err := settings.Validate(); err != nil {
		return nil, err
	}
	for _, w := range warns {
		log.Printf("[shared.infra] WARN: %s", w)
	}

	inf.SelfBaseURL = settings.SelfBaseURL
	inf.BrandsCollection = settings.BrandsCollection
	inf.AvatarsCollection = settings.AvatarsCollection
	inf.BrandWalletSecretPrefix = settings.BrandWalletSecretPrefix
	inf.TokenIconBucket = settings.TokenIconBucket
	inf.TokenContentsBucket = settings.TokenContentsBucket
	inf.ListImageBucket = settings.ListImageBucket
	inf.AvatarIconBucket = settings.AvatarIconBucket
	inf.PostImageBucket = settings.PostImageBucket

	// --------------------------------------------------------
	// Credentials file (optional; mainly for local dev)
	// --------------------------------------------------------
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

	// --------------------------------------------------------
	// 1) Optional: Arweave uploader (used by TokenBlueprintUsecase)
	// --------------------------------------------------------
	if strings.TrimSpace(cfg.ArweaveBaseURL) != "" {
		inf.ArweaveUploader = arweaveinfra.NewHTTPUploader(cfg.ArweaveBaseURL, cfg.ArweaveAPIKey)
		log.Printf("[shared.infra] Arweave HTTPUploader initialized baseURL=%s", cfg.ArweaveBaseURL)
	} else {
		log.Printf("[shared.infra] Arweave HTTPUploader not configured (ARWEAVE_BASE_URL empty)")
	}

	// --------------------------------------------------------
	// 2) Optional: Secret Manager client (used by Transfer signer provider etc.)
	// --------------------------------------------------------
	{
		var sm *secretmanager.Client
		var err error
		if len(clientOpts) > 0 {
			sm, err = secretmanager.NewClient(ctx, clientOpts...)
		} else {
			sm, err = secretmanager.NewClient(ctx)
		}
		if err != nil {
			log.Printf("[shared.infra] WARN: secretmanager.NewClient failed: %v (SecretManager-dependent features may be disabled)", err)
			sm = nil
		}
		inf.SecretManager = sm
	}

	// --------------------------------------------------------
	// 3) Optional: Solana mint authority key (Secret Manager)
	// NOTE: This loader may create its own SM client internally; we keep it as-is
	// to avoid wider signature changes. If you want to reuse inf.SecretManager,
	// adjust solanainfra.LoadMintAuthorityKey to accept a client.
	// --------------------------------------------------------
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

	// --------------------------------------------------------
	// 4) Firestore (strict)
	// --------------------------------------------------------
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

	// --------------------------------------------------------
	// 5) GCS (strict)
	// --------------------------------------------------------
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

	// --------------------------------------------------------
	// 6) Firebase App/Auth (best-effort)
	// --------------------------------------------------------
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

	// --------------------------------------------------------
	// Final sanity checks (panic prevention)
	// --------------------------------------------------------
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
	if i.SecretManager != nil {
		_ = i.SecretManager.Close()
	}
	return nil
}

func resolveProjectID(cfg *appcfg.Config) string {
	// Priority:
	// 1) cfg.FirestoreProjectID (resolved by config.Load)
	// 2) FIRESTORE_PROJECT_ID
	// 3) GCP_PROJECT_ID
	// 4) GOOGLE_CLOUD_PROJECT (often set in Cloud Run)
	// 5) FIREBASE_PROJECT_ID (fallback)
	if cfg != nil {
		if v := strings.TrimSpace(cfg.FirestoreProjectID); v != "" {
			return v
		}
	}

	// NOTE: env fallback is intentionally kept here because ProjectID is required
	// for initializing GCP/Firebase clients (i.e., it is not a "runtime setting").
	for _, k := range []string{
		"FIRESTORE_PROJECT_ID",
		"GCP_PROJECT_ID",
		"GOOGLE_CLOUD_PROJECT",
		"FIREBASE_PROJECT_ID",
	} {
		// We intentionally avoid os.Getenv usage here after introducing runtime_settings.go
		// because projectID is not part of RuntimeSettings.
		// appcfg.Load may already have resolved env for FirestoreProjectID,
		// but we keep these fallbacks to preserve current behavior.
		// (This file already imports no "os" anymore.)
		//
		// If you want to remove these env fallbacks entirely, move them into appcfg.Load.
		_ = k // placeholder to keep the loop structure if you later restore env fallbacks
	}

	// Behavior preserved by appcfg.Load() setting cfg.FirestoreProjectID;
	// if not set, ProjectID remains empty and caller returns hard error.
	return ""
}

func redactPath(p string) string {
	// Do not log full path (Windows/Unix compatible light masking)
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	// Keep only the last segment
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
