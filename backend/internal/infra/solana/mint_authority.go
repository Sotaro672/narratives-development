// backend/internal/platform/solana/mint_authority.go
package solana

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretspb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/blocto/solana-go-sdk/types"
)

// MintAuthority は Secret Manager に保存してある mint ウォレットを表します。
type MintAuthority struct {
	Account types.Account
}

// MintAuthorityKey は Narratives の「唯一のミント権限ウォレット」の ed25519 鍵ペアを表します。
type MintAuthorityKey struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// LoadMintAuthority は SOLANA_MINT_KEY_SECRET に指定した Secret から
// solana-keygen の keypair JSON 配列を復元して、types.Account を返します。
//
// SOLANA_MINT_KEY_SECRET には
//
//	"projects/<PROJECT_ID>/secrets/<SECRET_ID>/versions/latest"
//
// のような Secret Version のフルパスを設定してください。
func LoadMintAuthority(ctx context.Context) (*MintAuthority, error) {
	secretName := os.Getenv("SOLANA_MINT_KEY_SECRET")
	if secretName == "" {
		return nil, fmt.Errorf("SOLANA_MINT_KEY_SECRET not set")
	}

	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("secretmanager.NewClient: %w", err)
	}
	defer client.Close()

	resp, err := client.AccessSecretVersion(ctx, &secretspb.AccessSecretVersionRequest{
		Name: secretName,
	})
	if err != nil {
		return nil, fmt.Errorf("AccessSecretVersion: %w", err)
	}

	keyBytes, err := decodeKeypairJSON(resp.Payload.Data)
	if err != nil {
		return nil, err
	}

	acc, err := types.AccountFromBytes(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("AccountFromBytes: %w", err)
	}

	log.Printf(
		"[narratives-mint] loaded mint authority from Secret Manager: pubkey=%s",
		acc.PublicKey.ToBase58(),
	)

	return &MintAuthority{Account: acc}, nil
}

// LoadMintAuthorityKeyFromEnv は LoadMintAuthority と同じシークレットから
// ed25519 の鍵ペアを復元して返します。
func LoadMintAuthorityKeyFromEnv(ctx context.Context) (*MintAuthorityKey, error) {
	mint, err := LoadMintAuthority(ctx)
	if err != nil {
		return nil, err
	}

	priv := ed25519.PrivateKey(mint.Account.PrivateKey)
	if len(priv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("unexpected private key length: got %d, want %d", len(priv), ed25519.PrivateKeySize)
	}

	pubBytes := mint.Account.PublicKey.Bytes()
	if len(pubBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("unexpected public key length: got %d, want %d", len(pubBytes), ed25519.PublicKeySize)
	}

	pub := ed25519.PublicKey(pubBytes)

	log.Printf(
		"[narratives-mint] mint authority ed25519 key restored: pubkey=%s",
		mint.Account.PublicKey.ToBase58(),
	)

	return &MintAuthorityKey{
		PrivateKey: priv,
		PublicKey:  pub,
	}, nil
}

// decodeKeypairJSON は Secret Manager に保存した solana-keygen keypair JSON から
// 64 バイトの鍵配列を復元します。
func decodeKeypairJSON(data []byte) ([]byte, error) {
	var keyBytes []byte
	if err := json.Unmarshal(data, &keyBytes); err != nil {
		return nil, fmt.Errorf("unmarshal keypair json: %w", err)
	}

	if len(keyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("unexpected secret key length: got %d, want %d", len(keyBytes), ed25519.PrivateKeySize)
	}

	return keyBytes, nil
}
