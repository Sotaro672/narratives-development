// backend/internal/infra/solana/mint_client.go
package solana

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tokendom "narratives/internal/domain/token"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/common"
	ata "github.com/blocto/solana-go-sdk/program/associated_token_account"
	"github.com/blocto/solana-go-sdk/program/metaplex/token_metadata"
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

func sellerFeeBpsFromEnv() uint16 {
	// 例: "500" = 5%
	// 未設定なら 0（ロイヤリティなし）にしておく
	v := strings.TrimSpace(os.Getenv("SOLANA_SELLER_FEE_BPS"))
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 || n > 10000 {
		log.Printf("[narratives-mint] invalid SOLANA_SELLER_FEE_BPS=%q -> fallback 0", v)
		return 0
	}
	return uint16(n)
}

// MintToken は tokendom.MintAuthorityWalletPort インターフェースの実装です。
// Solana devnet 上で、1 つの新規 Mint アカウントを作成し、指定ウォレット宛てに
// Amount 枚（NFT なら通常 1）のトークンをミントします。
//
// 重要:
//   - Explorer に表示させるため、Metaplex Token Metadata (CreateMetadataAccountV3) と
//     MasterEdition (CreateMasterEditionV3) を同一トランザクションで作成します。
//   - これにより mintAddress から導出される metadata PDA が必ず存在します。
func (c *MintClient) MintToken(
	ctx context.Context,
	params tokendom.MintParams,
) (*tokendom.MintResult, error) {
	if c == nil || c.key == nil {
		return nil, fmt.Errorf("mint client is not initialized (missing mint authority key)")
	}

	to := strings.TrimSpace(params.ToAddress)
	if to == "" {
		return nil, fmt.Errorf("ToAddress is empty")
	}

	name := strings.TrimSpace(params.Name)
	symbol := strings.TrimSpace(params.Symbol)
	if name == "" || symbol == "" {
		return nil, fmt.Errorf("token name or symbol is empty")
	}

	metadataURI := strings.TrimSpace(params.MetadataURI)
	if metadataURI == "" {
		return nil, fmt.Errorf("MetadataURI is empty")
	}

	amount := params.Amount
	if amount == 0 {
		amount = 1
	}

	// NFT 的運用（1商品=1Mint）前提なら amount=1 を推奨
	// ただし既存仕様を壊さないため、ここでは強制せずログのみ
	if amount != 1 {
		log.Printf("[narratives-mint] warning: mint amount is %d (NFT-like tokens typically use 1)", amount)
	}

	// ------------------------------------------------------------
	// 1) Solana RPC クライアント (devnet)
	// ------------------------------------------------------------
	rpcURL := strings.TrimSpace(os.Getenv("SOLANA_RPC_URL"))
	if rpcURL == "" {
		rpcURL = rpc.DevnetRPCEndpoint
	}
	cl := client.NewClient(rpcURL)

	log.Printf("[narratives-mint] MintToken start rpc=%s to=%s amount=%d name=%s symbol=%s uri=%s",
		rpcURL, to, amount, name, symbol, metadataURI)

	// ------------------------------------------------------------
	// 2) ミント権限ウォレット（fee payer）を復元
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
	recipientPub := common.PublicKeyFromString(to)

	// ------------------------------------------------------------
	// 3) 新しい Mint アカウントを作成
	// ------------------------------------------------------------
	mintAccount := types.NewAccount()
	log.Printf("[narratives-mint] new mint pubkey=%s", mintAccount.PublicKey.ToBase58())

	// Associated Token Account (ATA)
	ataPubkey, _, err := common.FindAssociatedTokenAddress(recipientPub, mintAccount.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive ATA: %w", err)
	}
	log.Printf("[narratives-mint] ATA=%s", ataPubkey.ToBase58())

	// Metadata / MasterEdition PDA（mint から決定される）
	metadataPubkey, err := token_metadata.GetTokenMetaPubkey(mintAccount.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("GetTokenMetaPubkey: %w", err)
	}
	masterEditionPubkey, err := token_metadata.GetMasterEdition(mintAccount.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("GetMasterEdition: %w", err)
	}

	log.Printf("[narratives-mint] derived PDA: mint=%s metadata=%s masterEdition=%s",
		mintAccount.PublicKey.ToBase58(),
		metadataPubkey.ToBase58(),
		masterEditionPubkey.ToBase58(),
	)

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

	// MasterEdition MaxSupply（1商品=1Mint の運用なら 1 を推奨）
	maxSupply := uint64(1)

	// ------------------------------------------------------------
	// 4) トランザクションの Instruction 群を組み立て
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
		// 2) Mint 初期化 (decimals=0)
		token.InitializeMint(token.InitializeMintParam{
			Decimals:   0,
			Mint:       mintAccount.PublicKey,
			MintAuth:   feePayer.PublicKey,
			FreezeAuth: &feePayer.PublicKey,
		}),
		// 3) Metaplex Metadata アカウント作成（これが無いと Explorer は Unknown Token になりやすい）
		token_metadata.CreateMetadataAccountV3(
			token_metadata.CreateMetadataAccountV3Param{
				Metadata:                metadataPubkey,
				Mint:                    mintAccount.PublicKey,
				MintAuthority:           feePayer.PublicKey,
				UpdateAuthority:         feePayer.PublicKey,
				Payer:                   feePayer.PublicKey,
				UpdateAuthorityIsSigner: true,
				IsMutable:               true,
				Data: token_metadata.DataV2{
					Name:                 name,
					Symbol:               symbol,
					Uri:                  metadataURI,
					SellerFeeBasisPoints: sellerFeeBpsFromEnv(),
					Creators: &[]token_metadata.Creator{
						{
							Address:  feePayer.PublicKey,
							Verified: true,
							Share:    100,
						},
					},
				},
				CollectionDetails: nil,
			},
		),
		// 4) 受取人用 ATA 作成
		ata.CreateAssociatedTokenAccount(
			ata.CreateAssociatedTokenAccountParam{
				Funder:                 feePayer.PublicKey,
				Owner:                  recipientPub,
				Mint:                   mintAccount.PublicKey,
				AssociatedTokenAccount: ataPubkey,
			},
		),
		// 5) MintTo でミント実行
		token.MintTo(token.MintToParam{
			Mint:   mintAccount.PublicKey,
			To:     ataPubkey,
			Auth:   feePayer.PublicKey,
			Amount: amount,
		}),
		// 6) MasterEdition v3 作成（NFT 的運用をするなら推奨）
		token_metadata.CreateMasterEditionV3(
			token_metadata.CreateMasterEditionParam{
				Edition:         masterEditionPubkey,
				Mint:            mintAccount.PublicKey,
				UpdateAuthority: feePayer.PublicKey,
				MintAuthority:   feePayer.PublicKey,
				Metadata:        metadataPubkey,
				Payer:           feePayer.PublicKey,
				MaxSupply:       &maxSupply,
			},
		),
	}

	// ------------------------------------------------------------
	// 5) メッセージ & トランザクション作成
	// ------------------------------------------------------------
	msg := types.NewMessage(types.NewMessageParam{
		FeePayer:        feePayer.PublicKey,
		RecentBlockhash: recent.Blockhash,
		Instructions:    instructions,
	})

	tx, err := types.NewTransaction(types.NewTransactionParam{
		Message: msg,
		Signers: []types.Account{
			feePayer,
			mintAccount,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction: %w", err)
	}

	// ------------------------------------------------------------
	// 6) トランザクション送信
	// ------------------------------------------------------------
	sig, err := cl.SendTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}
	log.Printf("[narratives-mint] tx sent sig=%s", sig)

	// 簡易 wait（本番は confirmed/finalized の明示チェック推奨）
	ctxWait, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, _ = cl.GetSignatureStatuses(ctxWait, []string{sig})

	// 送信後に metadata PDA の存在を（best-effortで）確認してログに出す
	// ※ SDK の戻り型差異がある場合はここだけ調整してください（意図は「存在確認」）
	// 送信後に metadata PDA の存在を（best-effortで）確認してログに出す
	info, ierr := cl.GetAccountInfo(ctx, metadataPubkey.ToBase58())
	if ierr != nil {
		log.Printf("[narratives-mint] metadata account fetch error: metadata=%s err=%v", metadataPubkey.ToBase58(), ierr)
	} else {
		// AccountInfo が値型で返るため nil 判定は不可。data 長や owner をログに出す。
		log.Printf("[narratives-mint] metadata account fetched: metadata=%s lamports=%d owner=%s dataLen=%d",
			metadataPubkey.ToBase58(),
			info.Lamports,
			info.Owner,
			len(info.Data),
		)
	}

	log.Printf(
		"[narratives-mint] minted token (with metaplex metadata): mint=%s sig=%s owner=%s name=%s symbol=%s uri=%s metadata=%s",
		mintAccount.PublicKey.ToBase58(),
		sig,
		to,
		name,
		symbol,
		metadataURI,
		metadataPubkey.ToBase58(),
	)

	return &tokendom.MintResult{
		Signature:   sig,
		MintAddress: mintAccount.PublicKey.ToBase58(),
		Slot:        0,
	}, nil
}
