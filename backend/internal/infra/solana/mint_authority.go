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

// ReserveAuthority は fee payer へ SOL を補充するための
// reserve ウォレットを表します。
type ReserveAuthority struct {
	Account types.Account
}

// MintAuthorityKey は Narratives の「唯一のミント権限ウォレット」の
// ed25519 鍵ペアを表します。
type MintAuthorityKey struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// LoadMintAuthority は SOLANA_MINT_KEY_SECRET に指定した Secret から
// solana-keygen の keypair JSON 配列を復元して、MintAuthority を返します.
//
// SOLANA_MINT_KEY_SECRET には:
//
//	"projects/<PROJECT_ID>/secrets/<SECRET_ID>/versions/latest"
//
// のような Secret Version のフルパスを設定してください。
func LoadMintAuthority(ctx context.Context) (*MintAuthority, error) {
	acc, err := loadSolanaAccountFromSecretEnv(
		ctx,
		"SOLANA_MINT_KEY_SECRET",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"load mint authority: %w",
			err,
		)
	}

	log.Printf(
		"[narratives-mint] loaded mint authority from Secret Manager: pubkey=%s",
		acc.PublicKey.ToBase58(),
	)

	return &MintAuthority{
		Account: *acc,
	}, nil
}

// LoadReserveAuthority は SOLANA_RESERVE_KEY_SECRET に指定した Secret から
// solana-keygen の keypair JSON 配列を復元して、ReserveAuthority を返します.
//
// reserve wallet は fee payer の残高が
// SOLANA_MIN_FEE_PAYER_BALANCE_SOL を下回った場合に、
// fee payer へ native SOL を自動補充するために使用します.
//
// SOLANA_RESERVE_KEY_SECRET には:
//
//	"projects/<PROJECT_ID>/secrets/<SECRET_ID>/versions/latest"
//
// のような Secret Version のフルパスを設定してください。
func LoadReserveAuthority(ctx context.Context) (*ReserveAuthority, error) {
	acc, err := loadSolanaAccountFromSecretEnv(
		ctx,
		"SOLANA_RESERVE_KEY_SECRET",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"load reserve authority: %w",
			err,
		)
	}

	log.Printf(
		"[narratives-mint] loaded reserve authority from Secret Manager: pubkey=%s",
		acc.PublicKey.ToBase58(),
	)

	return &ReserveAuthority{
		Account: *acc,
	}, nil
}

// LoadMintAuthorityKeyFromEnv は LoadMintAuthority と同じ Secret から
// ed25519 の鍵ペアを復元して返します。
func LoadMintAuthorityKeyFromEnv(ctx context.Context) (*MintAuthorityKey, error) {
	mint, err := LoadMintAuthority(ctx)
	if err != nil {
		return nil, err
	}

	priv := ed25519.PrivateKey(mint.Account.PrivateKey)
	if len(priv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf(
			"unexpected private key length: got %d, want %d",
			len(priv),
			ed25519.PrivateKeySize,
		)
	}

	pubBytes := mint.Account.PublicKey.Bytes()
	if len(pubBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf(
			"unexpected public key length: got %d, want %d",
			len(pubBytes),
			ed25519.PublicKeySize,
		)
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

// ValidateMintAndReserveAuthorities は mint wallet と reserve wallet が
// 同一ウォレットではないことを検証します.
//
// 同一ウォレットの場合、自分自身から自分自身へ SOL を補充することになるため、
// auto top-up の構成として無効です。
func ValidateMintAndReserveAuthorities(
	mint *MintAuthority,
	reserve *ReserveAuthority,
) error {
	if mint == nil {
		return fmt.Errorf("mint authority is nil")
	}

	if reserve == nil {
		return fmt.Errorf("reserve authority is nil")
	}

	mintAddress := mint.Account.PublicKey.ToBase58()
	if mintAddress == "" {
		return fmt.Errorf("mint authority public key is empty")
	}

	reserveAddress := reserve.Account.PublicKey.ToBase58()
	if reserveAddress == "" {
		return fmt.Errorf("reserve authority public key is empty")
	}

	if mintAddress == reserveAddress {
		return fmt.Errorf(
			"mint authority and reserve authority must be different wallets: address=%s",
			mintAddress,
		)
	}

	return nil
}

// loadSolanaAccountFromSecretEnv は envKey に指定された環境変数から
// Secret Manager の Secret Version フルパスを取得し、
// solana-keygen の keypair JSON 配列を types.Account に復元します.
//
// 対象例:
//
//	SOLANA_MINT_KEY_SECRET
//	SOLANA_RESERVE_KEY_SECRET
func loadSolanaAccountFromSecretEnv(
	ctx context.Context,
	envKey string,
) (*types.Account, error) {
	if envKey == "" {
		return nil, fmt.Errorf("secret env key is empty")
	}

	secretName := os.Getenv(envKey)
	if secretName == "" {
		return nil, fmt.Errorf("%s not set", envKey)
	}

	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"secretmanager.NewClient for %s: %w",
			envKey,
			err,
		)
	}
	defer client.Close()

	resp, err := client.AccessSecretVersion(
		ctx,
		&secretspb.AccessSecretVersionRequest{
			Name: secretName,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"AccessSecretVersion for %s: %w",
			envKey,
			err,
		)
	}

	if resp == nil || resp.Payload == nil {
		return nil, fmt.Errorf(
			"secret payload is nil for %s",
			envKey,
		)
	}

	keyBytes, err := decodeKeypairJSON(
		resp.Payload.Data,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"decode keypair for %s: %w",
			envKey,
			err,
		)
	}

	acc, err := types.AccountFromBytes(keyBytes)
	if err != nil {
		return nil, fmt.Errorf(
			"AccountFromBytes for %s: %w",
			envKey,
			err,
		)
	}

	if acc.PublicKey.ToBase58() == "" {
		return nil, fmt.Errorf(
			"restored public key is empty for %s",
			envKey,
		)
	}

	return &acc, nil
}

// decodeKeypairJSON は Secret Manager に保存した
// solana-keygen keypair JSON から64バイトの鍵配列を復元します。
func decodeKeypairJSON(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("keypair secret data is empty")
	}

	var keyBytes []byte
	if err := json.Unmarshal(data, &keyBytes); err != nil {
		return nil, fmt.Errorf(
			"unmarshal keypair json: %w",
			err,
		)
	}

	if len(keyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf(
			"unexpected secret key length: got %d, want %d",
			len(keyBytes),
			ed25519.PrivateKeySize,
		)
	}

	return keyBytes, nil
}
