package solana

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretspb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var (
	ErrWalletSecretNotConfigured = errors.New("wallet_secret_provider: not configured")
	ErrWalletSecretEmptyWallet   = errors.New("wallet_secret_provider: walletAddress is empty")
	ErrWalletSecretNotFound      = errors.New("wallet_secret_provider: secret not found")
)

// WalletSecretProviderSM returns signer material as `any`.
// We return string(JSON int array) so TokenTransferExecutorSolana can normalize it.
type WalletSecretProviderSM struct {
	Client    *secretmanager.Client
	ProjectID string

	// Secret name resolver:
	// secretId = prefix + walletAddress
	// default prefix = "solana-wallet-"
	SecretIDPrefix string
}

func NewWalletSecretProviderSM(ctx context.Context, projectID string) (*WalletSecretProviderSM, error) {
	pid := strings.TrimSpace(projectID)
	if pid == "" {
		pid = strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	}
	if pid == "" {
		pid = strings.TrimSpace(os.Getenv("GCP_PROJECT"))
	}
	if pid == "" {
		return nil, fmt.Errorf("%w: projectID is empty", ErrWalletSecretNotConfigured)
	}

	c, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	prefix := strings.TrimSpace(os.Getenv("SOLANA_WALLET_SECRET_PREFIX"))
	if prefix == "" {
		prefix = "solana-wallet-"
	}

	return &WalletSecretProviderSM{
		Client:         c,
		ProjectID:      pid,
		SecretIDPrefix: prefix,
	}, nil
}

func (p *WalletSecretProviderSM) GetSigner(ctx context.Context, walletAddress string) (any, error) {
	if p == nil || p.Client == nil {
		return nil, ErrWalletSecretNotConfigured
	}

	w := strings.TrimSpace(walletAddress)
	if w == "" {
		return nil, ErrWalletSecretEmptyWallet
	}

	secretID := strings.TrimSpace(p.SecretIDPrefix) + w
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", p.ProjectID, secretID)

	res, err := p.Client.AccessSecretVersion(ctx, &secretspb.AccessSecretVersionRequest{Name: name})
	if err != nil {
		// 実運用では codes.NotFound 判定して ErrWalletSecretNotFound に寄せてもOK
		return nil, fmt.Errorf("%w: %v", ErrWalletSecretNotFound, err)
	}
	if res == nil || res.Payload == nil {
		return nil, ErrWalletSecretNotFound
	}

	s := strings.TrimSpace(string(res.Payload.Data))
	if s == "" {
		return nil, ErrWalletSecretNotFound
	}

	// ✅ TokenTransferExecutorSolana が string(JSON int array) を受け取れる
	return s, nil
}
