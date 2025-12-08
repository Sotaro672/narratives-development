// backend/internal/infra/solana/mint_authority_loader.go
package solana

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	smpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// MintAuthorityKey は Narratives の「唯一のミント権限ウォレット」の鍵ペアを表します。
type MintAuthorityKey struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// LoadMintAuthorityKey は GCP Secret Manager から
// narratives-mint-authority.json（[int,...]）を読み込み、ed25519 の鍵として復元します。
//
// projectID: narratives-development-26c2d など
// secretID : "narratives-solana-mint-authority"
func LoadMintAuthorityKey(
	ctx context.Context,
	projectID string,
	secretID string,
) (*MintAuthorityKey, error) {

	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("secretmanager.NewClient: %w", err)
	}
	defer client.Close()

	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretID)

	res, err := client.AccessSecretVersion(ctx, &smpb.AccessSecretVersionRequest{
		Name: name,
	})
	if err != nil {
		return nil, fmt.Errorf("access secret version %s: %w", name, err)
	}

	// generate_narratives_mint_wallet.go が出力した形式は [int,int,...] の配列
	var ints []int
	if err := json.Unmarshal(res.Payload.Data, &ints); err != nil {
		return nil, fmt.Errorf("unmarshal secret json: %w", err)
	}

	if len(ints) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("unexpected secret key length: got %d, want %d", len(ints), ed25519.PrivateKeySize)
	}

	b := make([]byte, len(ints))
	for i, v := range ints {
		b[i] = byte(v)
	}

	priv := ed25519.PrivateKey(b)
	pub := priv.Public().(ed25519.PublicKey)

	return &MintAuthorityKey{
		PrivateKey: priv,
		PublicKey:  pub,
	}, nil
}
