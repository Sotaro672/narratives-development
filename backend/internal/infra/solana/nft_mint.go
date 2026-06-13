// backend/internal/infra/solana/nft_mint.go
package solana

import (
	"context"
	"fmt"
	"os"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/common"
	"github.com/blocto/solana-go-sdk/program/associated_token_account"
	"github.com/blocto/solana-go-sdk/program/metaplex/token_metadata"
	"github.com/blocto/solana-go-sdk/program/system"
	"github.com/blocto/solana-go-sdk/program/token"
	"github.com/blocto/solana-go-sdk/rpc"
	"github.com/blocto/solana-go-sdk/types"
)

// Narratives から渡す NFT メタデータの入力
type NFTMetadataInput struct {
	Name                 string
	Symbol               string
	URI                  string // GCS/Arweave 上の metadata.json の URL
	SellerFeeBasisPoints uint16 // 例: 500 = 5%
}

// ownerWallet: 受取ウォレットの base58
// 戻り値: mint アドレス base58, tx シグネチャ
func MintNFTToOwner(
	ctx context.Context,
	ownerWallet string,
	meta NFTMetadataInput,
) (mintAddr string, signature string, err error) {
	rpcURL := os.Getenv("SOLANA_RPC_URL")
	if rpcURL == "" {
		rpcURL = rpc.DevnetRPCEndpoint
	}

	c := client.NewClient(rpcURL)

	auth, err := LoadMintAuthority(ctx)
	if err != nil {
		return "", "", fmt.Errorf("LoadMintAuthority: %w", err)
	}

	feePayer := auth.Account

	owner := common.PublicKeyFromString(ownerWallet)
	mint := types.NewAccount()

	ata, _, err := common.FindAssociatedTokenAddress(owner, mint.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("FindAssociatedTokenAddress: %w", err)
	}

	metadataPubkey, err := token_metadata.GetTokenMetaPubkey(mint.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("GetTokenMetaPubkey: %w", err)
	}

	masterEditionPubkey, err := token_metadata.GetMasterEdition(mint.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("GetMasterEdition: %w", err)
	}

	mintRent, err := c.GetMinimumBalanceForRentExemption(ctx, token.MintAccountSize)
	if err != nil {
		return "", "", fmt.Errorf("GetMinimumBalanceForRentExemption: %w", err)
	}

	recent, err := c.GetLatestBlockhash(ctx)
	if err != nil {
		return "", "", fmt.Errorf("GetLatestBlockhash: %w", err)
	}

	maxSupply := uint64(1)

	tx, err := types.NewTransaction(types.NewTransactionParam{
		Signers: []types.Account{mint, feePayer},
		Message: types.NewMessage(types.NewMessageParam{
			FeePayer:        feePayer.PublicKey,
			RecentBlockhash: recent.Blockhash,
			Instructions: []types.Instruction{
				system.CreateAccount(system.CreateAccountParam{
					From:     feePayer.PublicKey,
					New:      mint.PublicKey,
					Owner:    common.TokenProgramID,
					Lamports: mintRent,
					Space:    token.MintAccountSize,
				}),
				token.InitializeMint(token.InitializeMintParam{
					Decimals:   0,
					Mint:       mint.PublicKey,
					MintAuth:   feePayer.PublicKey,
					FreezeAuth: &feePayer.PublicKey,
				}),
				token_metadata.CreateMetadataAccountV3(
					token_metadata.CreateMetadataAccountV3Param{
						Metadata:                metadataPubkey,
						Mint:                    mint.PublicKey,
						MintAuthority:           feePayer.PublicKey,
						UpdateAuthority:         feePayer.PublicKey,
						Payer:                   feePayer.PublicKey,
						UpdateAuthorityIsSigner: true,
						IsMutable:               true,
						Data: token_metadata.DataV2{
							Name:                 meta.Name,
							Symbol:               meta.Symbol,
							Uri:                  meta.URI,
							SellerFeeBasisPoints: meta.SellerFeeBasisPoints,
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
				associated_token_account.CreateAssociatedTokenAccount(
					associated_token_account.CreateAssociatedTokenAccountParam{
						Funder:                 feePayer.PublicKey,
						Owner:                  owner,
						Mint:                   mint.PublicKey,
						AssociatedTokenAccount: ata,
					},
				),
				token.MintTo(token.MintToParam{
					Mint:   mint.PublicKey,
					To:     ata,
					Auth:   feePayer.PublicKey,
					Amount: 1,
				}),
				token_metadata.CreateMasterEditionV3(
					token_metadata.CreateMasterEditionParam{
						Edition:         masterEditionPubkey,
						Mint:            mint.PublicKey,
						UpdateAuthority: feePayer.PublicKey,
						MintAuthority:   feePayer.PublicKey,
						Metadata:        metadataPubkey,
						Payer:           feePayer.PublicKey,
						MaxSupply:       &maxSupply,
					},
				),
			},
		}),
	})
	if err != nil {
		return "", "", fmt.Errorf("NewTransaction: %w", err)
	}

	sig, err := c.SendTransaction(ctx, tx)
	if err != nil {
		return "", "", fmt.Errorf("SendTransaction: %w", err)
	}

	mintAddr = mint.PublicKey.ToBase58()

	return mintAddr, sig, nil
}
