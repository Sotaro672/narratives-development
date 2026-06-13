// backend/internal/infra/solana/mint_client.go
package solana

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
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
	_ = ctx

	if c == nil || c.key == nil {
		return "", fmt.Errorf("mint client is not initialized (missing mint authority key)")
	}
	if len(c.key.PublicKey) == 0 {
		return "", fmt.Errorf("mint authority public key is empty")
	}

	return hex.EncodeToString(c.key.PublicKey), nil
}

func sellerFeeBpsFromEnv() uint16 {
	v := os.Getenv("SOLANA_SELLER_FEE_BPS")
	if v == "" {
		return 0
	}

	n, err := strconv.Atoi(v)
	if err != nil || n < 0 || n > 10000 {
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

	to := params.ToAddress
	if to == "" {
		return nil, fmt.Errorf("ToAddress is empty")
	}

	name := params.Name
	symbol := params.Symbol
	if name == "" || symbol == "" {
		return nil, fmt.Errorf("token name or symbol is empty")
	}

	metadataURI := params.MetadataURI
	if metadataURI == "" {
		return nil, fmt.Errorf("MetadataURI is empty")
	}

	amount := params.Amount
	if amount == 0 {
		amount = 1
	}

	rpcURL := os.Getenv("SOLANA_RPC_URL")
	if rpcURL == "" {
		rpcURL = rpc.DevnetRPCEndpoint
	}
	cl := client.NewClient(rpcURL)

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

	recipientPub := common.PublicKeyFromString(to)

	mintAccount := types.NewAccount()

	ataPubkey, _, err := common.FindAssociatedTokenAddress(recipientPub, mintAccount.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive ATA: %w", err)
	}

	metadataPubkey, err := token_metadata.GetTokenMetaPubkey(mintAccount.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("GetTokenMetaPubkey: %w", err)
	}

	masterEditionPubkey, err := token_metadata.GetMasterEdition(mintAccount.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("GetMasterEdition: %w", err)
	}

	mintRent, err := cl.GetMinimumBalanceForRentExemption(ctx, token.MintAccountSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get rent for mint account: %w", err)
	}

	recent, err := cl.GetLatestBlockhash(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest blockhash: %w", err)
	}

	maxSupply := uint64(1)

	instructions := []types.Instruction{
		system.CreateAccount(system.CreateAccountParam{
			From:     feePayer.PublicKey,
			New:      mintAccount.PublicKey,
			Lamports: mintRent,
			Space:    token.MintAccountSize,
			Owner:    common.TokenProgramID,
		}),
		token.InitializeMint(token.InitializeMintParam{
			Decimals:   0,
			Mint:       mintAccount.PublicKey,
			MintAuth:   feePayer.PublicKey,
			FreezeAuth: &feePayer.PublicKey,
		}),
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
		ata.CreateAssociatedTokenAccount(
			ata.CreateAssociatedTokenAccountParam{
				Funder:                 feePayer.PublicKey,
				Owner:                  recipientPub,
				Mint:                   mintAccount.PublicKey,
				AssociatedTokenAccount: ataPubkey,
			},
		),
		token.MintTo(token.MintToParam{
			Mint:   mintAccount.PublicKey,
			To:     ataPubkey,
			Auth:   feePayer.PublicKey,
			Amount: amount,
		}),
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

	sig, err := cl.SendTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	ctxWait, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, _ = cl.GetSignatureStatuses(ctxWait, []string{sig})

	return &tokendom.MintResult{
		Signature:   sig,
		MintAddress: mintAccount.PublicKey.ToBase58(),
		Slot:        0,
	}, nil
}
