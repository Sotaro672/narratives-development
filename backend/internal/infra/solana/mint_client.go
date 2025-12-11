// backend/internal/infra/solana/mint_client.go
package solana

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tokendom "narratives/internal/domain/token"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/common"
	ata "github.com/blocto/solana-go-sdk/program/associated_token_account"
	"github.com/blocto/solana-go-sdk/program/system"
	"github.com/blocto/solana-go-sdk/program/token"
	"github.com/blocto/solana-go-sdk/rpc"
	"github.com/blocto/solana-go-sdk/types"
)

// MintClient は「Narratives が唯一保持するミント権限ウォレット」を使って
// 実際にチェーン上でミント処理を行うクライアントです。
// usecase.TokenUsecase からは tokendom.MintAuthorityWalletPort として利用されます。
type MintClient struct {
	key *MintAuthorityKey
}

// インターフェース実装チェック:
// MintClient が tokendom.MintAuthorityWalletPort を満たしていなければコンパイルエラーになります。
var _ tokendom.MintAuthorityWalletPort = (*MintClient)(nil)

// NewMintClient はミント権限キーを受け取って MintClient を初期化します。
func NewMintClient(key *MintAuthorityKey) *MintClient {
	return &MintClient{key: key}
}

// PublicKey は tokendom.MintAuthorityWalletPort の実装です。
// ミント権限ウォレットの公開鍵を string として返します。
//
// TODO: 将来的には Solana の base58 アドレス形式に揃える。
// ひとまずは ed25519.PublicKey ([]byte) を hex 文字列にして返しています。
func (c *MintClient) PublicKey(ctx context.Context) (string, error) {
	_ = ctx // 現時点では ctx を使用していないため unused 回避

	if c == nil || c.key == nil {
		return "", fmt.Errorf("mint client is not initialized (missing mint authority key)")
	}
	if len(c.key.PublicKey) == 0 {
		return "", fmt.Errorf("mint authority public key is empty")
	}

	// ed25519.PublicKey ([]byte) → string
	// ※ 実運用では base58 エンコードに変更する想定
	return hex.EncodeToString(c.key.PublicKey), nil
}

// MintToken は tokendom.MintAuthorityWalletPort インターフェースの実装です。
// Solana devnet 上で、1 つの新規 Mint アカウントを作成し、指定ウォレット宛てに
// Amount 枚（NFT なら通常 1）のトークンをミントします。
func (c *MintClient) MintToken(
	ctx context.Context,
	params tokendom.MintParams,
) (*tokendom.MintResult, error) {
	if c == nil || c.key == nil {
		return nil, fmt.Errorf("mint client is not initialized (missing mint authority key)")
	}

	if params.ToAddress == "" {
		return nil, fmt.Errorf("ToAddress is empty")
	}
	if params.Name == "" || params.Symbol == "" {
		return nil, fmt.Errorf("token name or symbol is empty")
	}
	amount := params.Amount
	if amount == 0 {
		amount = 1
	}

	// ------------------------------------------------------------
	// 1) Solana RPC クライアント (devnet)
	// ------------------------------------------------------------
	rpcURL := os.Getenv("SOLANA_RPC_URL")
	if strings.TrimSpace(rpcURL) == "" {
		// 環境変数未設定なら devnet のデフォルトを使う
		rpcURL = rpc.DevnetRPCEndpoint
	}
	cl := client.NewClient(rpcURL)

	log.Printf("[narratives-mint] MintToken start rpc=%s to=%s amount=%d metadata=%s name=%s symbol=%s",
		rpcURL, params.ToAddress, amount, params.MetadataURI, params.Name, params.Symbol)

	// ------------------------------------------------------------
	// 2) ミント権限ウォレット（fee payer）を復元
	//    MintAuthorityKey.PrivateKey は ed25519.PrivateKey ([]byte, 64 bytes 想定)
	//    MintAuthorityKey.PublicKey は ed25519.PublicKey ([]byte, 32 bytes 想定)
	// ------------------------------------------------------------
	if len(c.key.PrivateKey) == 0 {
		return nil, fmt.Errorf("mint authority private key is empty")
	}
	if len(c.key.PublicKey) == 0 {
		return nil, fmt.Errorf("mint authority public key is empty")
	}

	feePayer := types.Account{
		PrivateKey: c.key.PrivateKey,
		PublicKey:  common.PublicKeyFromBytes(c.key.PublicKey),
	}
	log.Printf("[narratives-mint] fee payer pubkey=%s", feePayer.PublicKey.ToBase58())

	// 受取人のウォレットアドレス (base58 → PublicKey)
	recipientPub := common.PublicKeyFromString(params.ToAddress)

	// ------------------------------------------------------------
	// 3) 新しい Mint アカウントを作成
	// ------------------------------------------------------------
	mintAccount := types.NewAccount()
	log.Printf("[narratives-mint] new mint pubkey=%s", mintAccount.PublicKey.ToBase58())

	// RentExempt に必要な lamports を取得
	mintRent, err := cl.GetMinimumBalanceForRentExemption(ctx, token.MintAccountSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get rent for mint account: %w", err)
	}

	// 最新 blockhash 取得
	recent, err := cl.GetLatestBlockhash(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest blockhash: %w", err)
	}

	// ------------------------------------------------------------
	// 4) Associated Token Account (ATA) アドレス計算
	//    nft_mint.go と同じく FindAssociatedTokenAddress を利用
	// ------------------------------------------------------------
	ataPubkey, _, err := common.FindAssociatedTokenAddress(recipientPub, mintAccount.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive ATA: %w", err)
	}
	log.Printf("[narratives-mint] ATA=%s", ataPubkey.ToBase58())

	// ------------------------------------------------------------
	// 5) トランザクションの Instruction 群を組み立て
	// ------------------------------------------------------------
	instructions := []types.Instruction{
		// 1) Mint アカウント作成
		system.CreateAccount(system.CreateAccountParam{
			From:     feePayer.PublicKey,
			New:      mintAccount.PublicKey,
			Lamports: mintRent,
			Space:    token.MintAccountSize,
			Owner:    common.TokenProgramID,
		}),
		// 2) Mint 初期化 (decimals=0 の NFT 的トークン)
		token.InitializeMint(token.InitializeMintParam{
			Decimals:   0,
			Mint:       mintAccount.PublicKey,
			MintAuth:   feePayer.PublicKey,
			FreezeAuth: &feePayer.PublicKey,
		}),
		// 3) 受取人用 ATA 作成
		ata.CreateAssociatedTokenAccount(
			ata.CreateAssociatedTokenAccountParam{
				Funder:                 feePayer.PublicKey,
				Owner:                  recipientPub,
				Mint:                   mintAccount.PublicKey,
				AssociatedTokenAccount: ataPubkey,
			},
		),
		// 4) MintTo でミント実行
		token.MintTo(token.MintToParam{
			Mint:   mintAccount.PublicKey,
			To:     ataPubkey,
			Auth:   feePayer.PublicKey,
			Amount: amount,
		}),
	}

	// ------------------------------------------------------------
	// 6) メッセージ & トランザクション作成
	// ------------------------------------------------------------
	msg := types.NewMessage(types.NewMessageParam{
		FeePayer:        feePayer.PublicKey,
		RecentBlockhash: recent.Blockhash,
		Instructions:    instructions,
	})

	tx, err := types.NewTransaction(types.NewTransactionParam{
		Message: msg,
		Signers: []types.Account{
			feePayer,    // fee payer & mint authority
			mintAccount, // 新規 mint アカウント
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction: %w", err)
	}

	// ------------------------------------------------------------
	// 7) トランザクション送信
	// ------------------------------------------------------------
	sig, err := cl.SendTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}
	log.Printf("[narratives-mint] tx sent sig=%s", sig)

	// オプション: ある程度確定するまで軽く待つ（本番ではきちんとステータス確認した方が良い）
	ctxWait, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, _ = cl.GetSignatureStatuses(ctxWait, []string{sig})

	return &tokendom.MintResult{
		Signature:   sig,
		MintAddress: mintAccount.PublicKey.ToBase58(),
		Slot:        0, // 必要なら GetSlot などで取得して詰める
	}, nil
}
