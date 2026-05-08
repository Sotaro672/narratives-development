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

// MintAuthority は Secret Manager に保存してある Devnet 用 mint ウォレットを表します。
type MintAuthority struct {
	Account types.Account
}

// MintAuthorityKey は Narratives の「唯一のミント権限ウォレット」の ed25519 鍵ペアを表します。
type MintAuthorityKey struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// LoadMintAuthority は SOLANA_MINT_KEY_SECRET に指定した Secret から
// solana-keygen の keypair(JSON配列 [u8;64]) を復元して、types.Account を返します。
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

	// シークレットの中身は solana-keygen の keypair JSON。
	// 正式には [u8;64] を想定するが、後方互換のため [int,...] 形式も許容する。
	keyBytes, err := decodeKeypairJSON(resp.Payload.Data)
	if err != nil {
		return nil, err
	}

	acc, err := types.AccountFromBytes(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("AccountFromBytes: %w", err)
	}

	// ★ マスターウォレット（ミント権限）との接続確認ログ
	//   - Secret Manager からの取得が成功し、Account を復元できたタイミングで出す
	log.Printf(
		"[narratives-mint] loaded mint authority from Secret Manager: secret=%s pubkey=%s",
		secretName,
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

	// types.Account.PrivateKey は 64 バイトの ed25519 秘密鍵（seed + public key）を想定
	priv := ed25519.PrivateKey(mint.Account.PrivateKey)
	if len(priv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("unexpected private key length: got %d, want %d", len(priv), ed25519.PrivateKeySize)
	}

	pubBytes := mint.Account.PublicKey.Bytes()
	if len(pubBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("unexpected public key length: got %d, want %d", len(pubBytes), ed25519.PublicKeySize)
	}
	pub := ed25519.PublicKey(pubBytes)

	// ★ ed25519 鍵ペアとしても問題なく復元できたことを追加でログ（公開鍵のみ）
	log.Printf(
		"[narratives-mint] mint authority ed25519 key restored: pubkey=%s",
		mint.Account.PublicKey.ToBase58(),
	)

	return &MintAuthorityKey{
		PrivateKey: priv,
		PublicKey:  pub,
	}, nil
}

// 互換用: 旧 infra 実装のシグネチャを維持したラッパ
// projectID, secretID は現在は使用せず、SOLANA_MINT_KEY_SECRET に委譲します。
func LoadMintAuthorityKey(
	ctx context.Context,
	projectID string,
	secretID string,
) (*MintAuthorityKey, error) {
	_ = projectID
	_ = secretID

	return LoadMintAuthorityKeyFromEnv(ctx)
}

// decodeKeypairJSON は Secret Manager に保存した keypair JSON から
// 64 バイトの鍵配列を復元します。
// - 正: [u8;64] を []byte で受け取る
// - 互換: [int,...] を []int で受けてから []byte に変換
func decodeKeypairJSON(data []byte) ([]byte, error) {
	// まずは []byte としてのデコードを試みる
	var keyBytes []byte
	if err := json.Unmarshal(data, &keyBytes); err == nil {
		if len(keyBytes) == ed25519.PrivateKeySize {
			return keyBytes, nil
		}
		// 長さが想定外の場合は後続のパスでエラーにする
	}

	// フォールバック: [int,int,...] の形式
	var ints []int
	if err := json.Unmarshal(data, &ints); err != nil {
		return nil, fmt.Errorf("unmarshal keypair json: %w", err)
	}

	if len(ints) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("unexpected secret key length: got %d, want %d", len(ints), ed25519.PrivateKeySize)
	}

	keyBytes = make([]byte, len(ints))
	for i, v := range ints {
		keyBytes[i] = byte(v)
	}

	return keyBytes, nil
}
