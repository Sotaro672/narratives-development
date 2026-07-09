// backend/internal/infra/solana/mint_client.go
package solana

import (
	"context"
	"encoding/hex"
	"fmt"
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
	"github.com/blocto/solana-go-sdk/types"
)

const lamportsPerSOL = 1_000_000_000

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

func autoAirdropEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("SOLANA_AUTO_AIRDROP_ENABLED")))

	return v == "true" || v == "1" || v == "yes" || v == "on"
}

func envSOLToLamports(key string, defaultSOL float64) (uint64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		if defaultSOL <= 0 {
			return 0, nil
		}
		return uint64(defaultSOL * lamportsPerSOL), nil
	}

	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number: %w", key, err)
	}
	if v < 0 {
		return 0, fmt.Errorf("%s must be >= 0", key)
	}

	return uint64(v * lamportsPerSOL), nil
}

type solanaGetBalanceResult struct {
	Context struct {
		Slot uint64 `json:"slot"`
	} `json:"context"`
	Value uint64 `json:"value"`
}

func getSolanaBalance(
	ctx context.Context,
	endpoint string,
	address string,
) (uint64, error) {
	if endpoint == "" {
		return 0, fmt.Errorf("solana rpc endpoint is empty")
	}
	if address == "" {
		return 0, fmt.Errorf("solana address is empty")
	}

	rpcClient := NewJSONRPCClientWithEndpoint(endpoint)

	var out solanaGetBalanceResult
	if err := rpcClient.call(ctx, "getBalance", []any{
		address,
		map[string]any{
			"commitment": "confirmed",
		},
	}, &out); err != nil {
		return 0, err
	}

	return out.Value, nil
}

func requestDevnetAirdrop(
	ctx context.Context,
	endpoint string,
	address string,
	lamports uint64,
) (string, error) {
	if endpoint == "" {
		return "", fmt.Errorf("solana airdrop rpc endpoint is empty")
	}
	if address == "" {
		return "", fmt.Errorf("solana address is empty")
	}
	if lamports == 0 {
		return "", fmt.Errorf("airdrop lamports is zero")
	}

	rpcClient := NewJSONRPCClientWithEndpoint(endpoint)

	var signature string
	if err := rpcClient.call(ctx, "requestAirdrop", []any{
		address,
		lamports,
	}, &signature); err != nil {
		return "", err
	}

	if signature == "" {
		return "", fmt.Errorf("requestAirdrop returned empty signature")
	}

	return signature, nil
}

// ensureFeePayerBalance は devnet 専用の開発補助機能です。
// SOLANA_AUTO_AIRDROP_ENABLED=true の場合のみ、fee payer の残高を確認し、
// SOLANA_MIN_FEE_PAYER_BALANCE_SOL 未満なら SOLANA_AIRDROP_AMOUNT_SOL 分だけ airdrop します。
func (c *MintClient) ensureFeePayerBalance(
	ctx context.Context,
	feePayer common.PublicKey,
	mintRPCURL string,
) error {
	if !autoAirdropEnabled() {
		return nil
	}

	address := feePayer.ToBase58()
	if address == "" {
		return fmt.Errorf("fee payer address is empty")
	}

	minLamports, err := envSOLToLamports("SOLANA_MIN_FEE_PAYER_BALANCE_SOL", 2)
	if err != nil {
		return err
	}
	if minLamports == 0 {
		return nil
	}

	airdropLamports, err := envSOLToLamports("SOLANA_AIRDROP_AMOUNT_SOL", 2)
	if err != nil {
		return err
	}
	if airdropLamports == 0 {
		return fmt.Errorf("SOLANA_AIRDROP_AMOUNT_SOL must be greater than 0 when auto airdrop is enabled")
	}

	balanceRPCURL := mintRPCURL
	if balanceRPCURL == "" {
		balanceRPCURL, err = solanaRPCURLFromEnv()
		if err != nil {
			return err
		}
	}

	var balance uint64
	if err := withSolanaRPCRetry(ctx, "get fee payer balance", func() error {
		v, err := getSolanaBalance(ctx, balanceRPCURL, address)
		if err != nil {
			return err
		}
		balance = v
		return nil
	}); err != nil {
		return fmt.Errorf("get fee payer balance: %w", err)
	}

	if balance >= minLamports {
		return nil
	}

	airdropRPCURL := strings.TrimSpace(os.Getenv("SOLANA_AIRDROP_RPC_URL"))
	if airdropRPCURL == "" {
		airdropRPCURL = balanceRPCURL
	}

	var sig string
	if err := withSolanaRPCRetry(ctx, "request devnet airdrop", func() error {
		v, err := requestDevnetAirdrop(ctx, airdropRPCURL, address, airdropLamports)
		if err != nil {
			return err
		}
		sig = v
		return nil
	}); err != nil {
		return fmt.Errorf("request devnet airdrop for fee payer %s: %w", address, err)
	}

	ctxWait, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if err := waitForSignatureConfirmed(ctxWait, airdropRPCURL, sig); err != nil {
		return fmt.Errorf("confirm devnet airdrop signature=%s: %w", sig, err)
	}

	var updatedBalance uint64
	if err := withSolanaRPCRetry(ctx, "get fee payer balance after airdrop", func() error {
		v, err := getSolanaBalance(ctx, balanceRPCURL, address)
		if err != nil {
			return err
		}
		updatedBalance = v
		return nil
	}); err != nil {
		return fmt.Errorf("get fee payer balance after airdrop: %w", err)
	}

	if updatedBalance < minLamports {
		return fmt.Errorf(
			"fee payer balance is still below minimum after airdrop: address=%s balance=%d min=%d",
			address,
			updatedBalance,
			minLamports,
		)
	}

	return nil
}

// MintToken は tokendom.MintAuthorityWalletPort インターフェースの実装です。
// Solana 上で、1 つの新規 Mint アカウントを作成し、指定ウォレット宛てに
// Amount 枚（NFT なら通常 1）のトークンをミントします。
//
// 重要:
//   - Explorer に表示させるため、Metaplex Token Metadata (CreateMetadataAccountV3) と
//     MasterEdition (CreateMasterEditionV3) を同一トランザクションで作成します。
//   - これにより mintAddress から導出される metadata PDA が必ず存在します。
//   - SendTransaction 後、confirmed / finalized になるまで確認してから成功として返します。
//   - SOLANA_AUTO_AIRDROP_ENABLED=true の場合のみ、devnet SOL 残高不足時に自動airdropします。
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

	rpcURL, err := solanaRPCURLFromEnv()
	if err != nil {
		return nil, err
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

	if err := c.ensureFeePayerBalance(ctx, feePayer.PublicKey, rpcURL); err != nil {
		return nil, fmt.Errorf("ensure fee payer balance: %w", err)
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

	var mintRent uint64
	if err := withSolanaRPCRetry(ctx, "get minimum balance for mint account", func() error {
		v, err := cl.GetMinimumBalanceForRentExemption(ctx, token.MintAccountSize)
		if err != nil {
			return err
		}
		mintRent = v
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to get rent for mint account: %w", err)
	}

	var recentBlockhash string
	if err := withSolanaRPCRetry(ctx, "get latest blockhash", func() error {
		v, err := cl.GetLatestBlockhash(ctx)
		if err != nil {
			return err
		}
		recentBlockhash = v.Blockhash
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to get latest blockhash: %w", err)
	}

	if recentBlockhash == "" {
		return nil, fmt.Errorf("latest blockhash is empty")
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
		RecentBlockhash: recentBlockhash,
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

	var sig string
	if err := withSolanaRPCRetry(ctx, "send transaction", func() error {
		v, err := cl.SendTransaction(ctx, tx)
		if err != nil {
			return err
		}
		sig = v
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	if sig == "" {
		return nil, fmt.Errorf("send transaction returned empty signature")
	}

	ctxWait, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	if err := waitForSignatureConfirmed(ctxWait, rpcURL, sig); err != nil {
		return nil, fmt.Errorf("failed to confirm transaction signature=%s: %w", sig, err)
	}

	return &tokendom.MintResult{
		Signature:   sig,
		MintAddress: mintAccount.PublicKey.ToBase58(),
		Slot:        0,
	}, nil
}
