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
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"

	stripeadapter "narratives/internal/adapters/out/stripe"
	appcfg "narratives/internal/infra/config"
	solanainfra "narratives/internal/infra/solana"
)

const (
	defaultBrandsCollection  = "brands"
	defaultAvatarsCollection = "avatars"

	defaultBrandWalletSecretPrefix  = "brand-wallet-"
	defaultAvatarWalletSecretPrefix = "avatar-wallet-"

	stripeSecretKeySecretID = "stripe-secret-key"
)

// Infra is shared runtime infrastructure for DI.
//
// GCS は廃止済みのため、この shared infra では storage.Client / bucket 設定を持ちません。
type Infra struct {
	// Config
	Config    *appcfg.Config
	ProjectID string

	// Clients
	Firestore     *firestore.Client
	FirebaseApp   *firebase.App
	FirebaseAuth  *firebaseauth.Client
	SecretManager *secretmanager.Client

	// Cross-cutting infra
	MintAuthorityKey *solanainfra.MintAuthorityKey

	// Adapters / gateways
	PaymentMethodGateway *stripeadapter.PaymentMethodGateway

	// Runtime settings
	SelfBaseURL              string
	BrandsCollection         string
	AvatarsCollection        string
	BrandWalletSecretPrefix  string
	AvatarWalletSecretPrefix string
}

func NewInfra(ctx context.Context) (*Infra, error) {
	cfg := appcfg.Load()
	if cfg == nil {
		return nil, errors.New("shared.infra: config is nil")
	}

	projectID := resolveProjectID(cfg)
	if projectID == "" {
		return nil, errors.New("shared.infra: projectID is empty (set FIRESTORE_PROJECT_ID or GOOGLE_CLOUD_PROJECT)")
	}

	inf := &Infra{
		Config:    cfg,
		ProjectID: projectID,
	}

	// --------------------------------------------------------
	// Runtime settings
	// --------------------------------------------------------
	settings, warns, err := ResolveRuntimeSettings(cfg)
	if err != nil {
		return nil, err
	}

	// NOTE:
	// GCS bucket 廃止後、RuntimeSettings.Validate() が bucket 必須を検証している場合は、
	// ResolveRuntimeSettings / Validate 側から bucket 必須条件も削除してください。
	if err := settings.Validate(); err != nil {
		return nil, err
	}

	for _, w := range warns {
		log.Printf("[shared.infra] WARN: %s", w)
	}

	inf.SelfBaseURL = settings.SelfBaseURL
	inf.BrandsCollection = settings.BrandsCollection
	if inf.BrandsCollection == "" {
		inf.BrandsCollection = defaultBrandsCollection
	}

	inf.AvatarsCollection = settings.AvatarsCollection
	if inf.AvatarsCollection == "" {
		inf.AvatarsCollection = defaultAvatarsCollection
	}

	inf.BrandWalletSecretPrefix = settings.BrandWalletSecretPrefix
	if inf.BrandWalletSecretPrefix == "" {
		inf.BrandWalletSecretPrefix = defaultBrandWalletSecretPrefix
	}

	inf.AvatarWalletSecretPrefix = settings.AvatarWalletSecretPrefix
	if inf.AvatarWalletSecretPrefix == "" {
		inf.AvatarWalletSecretPrefix = defaultAvatarWalletSecretPrefix
	}

	// --------------------------------------------------------
	// Credentials file
	// --------------------------------------------------------
	credFile := cfg.FirestoreCredentialsFile
	if credFile == "" {
		credFile = cfg.GCPCreds
	}

	var clientOpts []option.ClientOption
	if credFile != "" {
		clientOpts = append(clientOpts, option.WithCredentialsFile(credFile))
		log.Printf("[shared.infra] Using credentials file for GCP clients: %s", redactPath(credFile))
	} else {
		log.Printf("[shared.infra] Using Application Default Credentials")
	}

	// --------------------------------------------------------
	// Secret Manager
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
			// stripe-secret-key を使うため Secret Manager は実質必須。
			return nil, fmt.Errorf("shared.infra: secretmanager.NewClient failed: %w", err)
		}

		inf.SecretManager = sm
		log.Printf("[shared.infra] Secret Manager initialized")
	}

	// --------------------------------------------------------
	// Solana mint authority key
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
	// Firestore
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
			_ = inf.Close()
			return nil, fmt.Errorf("shared.infra: firestore.NewClient failed (project=%s): %w", inf.ProjectID, err)
		}

		inf.Firestore = fsClient
		log.Printf("[shared.infra] Firestore connected project=%s", inf.ProjectID)
	}

	// --------------------------------------------------------
	// Firebase App/Auth
	// --------------------------------------------------------
	{
		fbCfg := &firebase.Config{ProjectID: inf.ProjectID}

		var fbApp *firebase.App
		var err error

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

	if inf.Firestore == nil {
		_ = inf.Close()
		return nil, errors.New("shared.infra: firestore client is nil after initialization")
	}
	if inf.SecretManager == nil {
		_ = inf.Close()
		return nil, errors.New("shared.infra: secret manager client is nil after initialization")
	}

	return inf, nil
}

// AccessSecretVersion reads a secret value from Google Secret Manager.
func (i *Infra) AccessSecretVersion(ctx context.Context, secretID string) (string, error) {
	if i == nil {
		return "", errors.New("shared.infra: infra is nil")
	}
	if i.SecretManager == nil {
		return "", errors.New("shared.infra: secret manager client is nil")
	}

	secretID = strings.TrimSpace(secretID)
	if secretID == "" {
		return "", errors.New("shared.infra: secretID is empty")
	}

	projectID := strings.TrimSpace(i.ProjectID)
	if projectID == "" {
		return "", errors.New("shared.infra: projectID is empty")
	}

	name := "projects/" + projectID + "/secrets/" + secretID + "/versions/latest"

	result, err := i.SecretManager.AccessSecretVersion(
		ctx,
		&secretmanagerpb.AccessSecretVersionRequest{Name: name},
	)
	if err != nil {
		return "", err
	}

	value := strings.TrimSpace(string(result.Payload.Data))
	if value == "" {
		return "", errors.New("shared.infra: secret value is empty: " + secretID)
	}

	return value, nil
}

// RegisterPaymentMethodGateway registers Stripe payment method gateway into shared infra.
func (i *Infra) RegisterPaymentMethodGateway(
	stripeSecretKey string,
	customerStore stripeadapter.PaymentMethodCustomerStore,
) error {
	if i == nil {
		return errors.New("shared.infra: infra is nil")
	}

	stripeSecretKey = strings.TrimSpace(stripeSecretKey)
	if stripeSecretKey == "" {
		return errors.New("shared.infra: stripe secret key is empty")
	}
	if !strings.HasPrefix(stripeSecretKey, "sk_") {
		return errors.New("shared.infra: stripe secret key is invalid")
	}
	if customerStore == nil {
		return errors.New("shared.infra: payment method customer store is nil")
	}

	i.PaymentMethodGateway = stripeadapter.NewPaymentMethodGateway(
		stripeSecretKey,
		customerStore,
	)

	if i.PaymentMethodGateway == nil {
		return errors.New("shared.infra: stripe payment method gateway is nil after registration")
	}

	log.Printf("[shared.infra] Stripe PaymentMethodGateway registered from provided secret")
	return nil
}

// RegisterPaymentMethodGatewayFromSecret reads stripe-secret-key from Secret Manager
// and registers the Stripe gateway.
func (i *Infra) RegisterPaymentMethodGatewayFromSecret(
	ctx context.Context,
	customerStore stripeadapter.PaymentMethodCustomerStore,
) error {
	if i == nil {
		return errors.New("shared.infra: infra is nil")
	}
	if customerStore == nil {
		return errors.New("shared.infra: payment method customer store is nil")
	}

	stripeSecretKey, err := i.AccessSecretVersion(ctx, stripeSecretKeySecretID)
	if err != nil {
		return fmt.Errorf("shared.infra: failed to access %s: %w", stripeSecretKeySecretID, err)
	}

	if err := i.RegisterPaymentMethodGateway(stripeSecretKey, customerStore); err != nil {
		return err
	}

	log.Printf("[shared.infra] Stripe PaymentMethodGateway registered from Secret Manager secret=%q", stripeSecretKeySecretID)
	return nil
}

func (i *Infra) Close() error {
	if i == nil {
		return nil
	}

	if i.Firestore != nil {
		_ = i.Firestore.Close()
		i.Firestore = nil
	}

	if i.SecretManager != nil {
		_ = i.SecretManager.Close()
		i.SecretManager = nil
	}

	return nil
}

func resolveProjectID(cfg *appcfg.Config) string {
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
		"PROJECT_ID",
	} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}

	return ""
}

func redactPath(p string) string {
	if p == "" {
		return ""
	}

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
