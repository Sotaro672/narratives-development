// backend/internal/infra/solana/token_transfer_executor.go
package solana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/common"
	"github.com/blocto/solana-go-sdk/program/token"
	"github.com/blocto/solana-go-sdk/types"

	usecase "narratives/internal/application/usecase"
)

var (
	ErrTokenTransferNotConfigured   = errors.New("token_transfer_executor: not configured")
	ErrTokenTransferMintEmpty       = errors.New("token_transfer_executor: mintAddress is empty")
	ErrTokenTransferToWalletEmpty   = errors.New("token_transfer_executor: toWalletAddress is empty")
	ErrTokenTransferSignerEmpty     = errors.New("token_transfer_executor: signer is nil")
	ErrTokenTransferInvalidSigner   = errors.New("token_transfer_executor: invalid signer type")
	ErrTokenTransferInvalidPrivKey  = errors.New("token_transfer_executor: invalid private key bytes")
	ErrTokenTransferSourceAtaAbsent = errors.New("token_transfer_executor: source ATA not found")
)

const (
	defaultDevnetRPC = "https://api.devnet.solana.com"

	// Associated Token Account Program
	associatedTokenProgramID = "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL"
)

type TokenTransferExecutorSolana struct {
	RPC *client.Client

	Commitment string        // e.g. "finalized"
	Timeout    time.Duration // RPC timeout hint (best-effort)
}

// NewTokenTransferExecutorSolana constructs executor.
// RPC URL resolves from SOLANA_RPC_URL if url is empty.
func NewTokenTransferExecutorSolana(rpcURL string) *TokenTransferExecutorSolana {
	u := strings.TrimSpace(rpcURL)
	if u == "" {
		u = strings.TrimSpace(os.Getenv("SOLANA_RPC_URL"))
	}
	if u == "" {
		u = defaultDevnetRPC
	}
	return &TokenTransferExecutorSolana{
		RPC:        client.NewClient(u),
		Commitment: "finalized",
		Timeout:    20 * time.Second,
	}
}

// ExecuteTransfer does:
// - derive ATA(from owner, mint) / ATA(to owner, mint)
// - create destination ATA if missing (payer=from)
// - SPL token transfer (amount default=1)
// - send tx (signed by from signer)
func (e *TokenTransferExecutorSolana) ExecuteTransfer(ctx context.Context, in usecase.ExecuteTransferInput) (usecase.ExecuteTransferResult, error) {
	if e == nil || e.RPC == nil {
		return usecase.ExecuteTransferResult{}, ErrTokenTransferNotConfigured
	}

	toWallet := strings.TrimSpace(in.ToWalletAddress)
	if toWallet == "" {
		return usecase.ExecuteTransferResult{}, ErrTokenTransferToWalletEmpty
	}
	mintAddr := strings.TrimSpace(in.MintAddress)
	if mintAddr == "" {
		return usecase.ExecuteTransferResult{}, ErrTokenTransferMintEmpty
	}
	amt := in.Amount
	if amt == 0 {
		amt = 1
	}

	// ✅ 署名者（source owner）: SPL transfer は送付側(owner of source ATA)が署名
	signerAny := in.FromSigner
	if signerAny == nil {
		signerAny = in.ToSigner // legacy fallback
	}
	if signerAny == nil {
		return usecase.ExecuteTransferResult{}, ErrTokenTransferSignerEmpty
	}

	fromAcc, err := normalizeToAccount(signerAny)
	if err != nil {
		return usecase.ExecuteTransferResult{}, err
	}

	mint := common.PublicKeyFromString(mintAddr)
	toOwner := common.PublicKeyFromString(toWallet)

	fromOwner := fromAcc.PublicKey
	fromATA, _, err := common.FindAssociatedTokenAddress(fromOwner, mint)
	if err != nil {
		return usecase.ExecuteTransferResult{}, fmt.Errorf("token_transfer_executor: derive from ATA failed: %w", err)
	}
	toATA, _, err := common.FindAssociatedTokenAddress(toOwner, mint)
	if err != nil {
		return usecase.ExecuteTransferResult{}, fmt.Errorf("token_transfer_executor: derive to ATA failed: %w", err)
	}

	log.Printf(
		"[token_transfer_executor] start productId=%s avatarId=%s brandId=%s mint=%s amount=%d from=%s to=%s",
		maskShort(in.ProductID),
		maskShort(in.AvatarID),
		maskShort(in.BrandID),
		maskShort(mintAddr),
		amt,
		maskShort(fromOwner.ToBase58()),
		maskShort(toWallet),
	)

	// 1) existence checks
	fromExists, err := e.accountExists(ctx, fromATA.ToBase58())
	if err != nil {
		return usecase.ExecuteTransferResult{}, fmt.Errorf("token_transfer_executor: check from ATA failed: %w", err)
	}
	if !fromExists {
		return usecase.ExecuteTransferResult{}, ErrTokenTransferSourceAtaAbsent
	}

	toExists, err := e.accountExists(ctx, toATA.ToBase58())
	if err != nil {
		return usecase.ExecuteTransferResult{}, fmt.Errorf("token_transfer_executor: check to ATA failed: %w", err)
	}

	// 2) build instructions
	ins := make([]types.Instruction, 0, 3)

	// 2-1) create dest ATA if missing (payer=fromOwner)
	if !toExists {
		ins = append(ins, buildCreateAssociatedTokenAccountIx(fromOwner, toOwner, mint, toATA))
		log.Printf(
			"[token_transfer_executor] will create ATA: owner=%s mint=%s ata=%s payer=%s",
			maskShort(toWallet),
			maskShort(mintAddr),
			maskShort(toATA.ToBase58()),
			maskShort(fromOwner.ToBase58()),
		)
	}

	// 2-2) token transfer
	ins = append(ins, token.Transfer(token.TransferParam{
		From:   fromATA,
		To:     toATA,
		Auth:   fromOwner,
		Amount: amt,
	}))

	// 3) recent blockhash
	latest, err := e.RPC.GetLatestBlockhash(ctx)
	if err != nil {
		return usecase.ExecuteTransferResult{}, fmt.Errorf("token_transfer_executor: GetLatestBlockhash: %w", err)
	}

	// 4) tx
	tx, err := types.NewTransaction(types.NewTransactionParam{
		Message: types.NewMessage(types.NewMessageParam{
			FeePayer:        fromOwner,
			RecentBlockhash: latest.Blockhash,
			Instructions:    ins,
		}),
		Signers: []types.Account{fromAcc},
	})
	if err != nil {
		return usecase.ExecuteTransferResult{}, fmt.Errorf("token_transfer_executor: NewTransaction: %w", err)
	}

	// 5) send
	sig, err := e.RPC.SendTransaction(ctx, tx)
	if err != nil {
		return usecase.ExecuteTransferResult{}, fmt.Errorf("token_transfer_executor: SendTransaction: %w", err)
	}

	log.Printf(
		"[token_transfer_executor] submitted tx=%s mint=%s fromATA=%s toATA=%s createdATA=%t",
		maskShort(sig),
		maskShort(mintAddr),
		maskShort(fromATA.ToBase58()),
		maskShort(toATA.ToBase58()),
		!toExists,
	)

	return usecase.ExecuteTransferResult{TxSignature: sig}, nil
}

// buildCreateAssociatedTokenAccountIx builds ATA creation instruction without depending on SDK subpackage.
// Accounts:
// 0. [writable,signer] payer
// 1. [writable] associated token account address
// 2. [] owner
// 3. [] mint
// 4. [] system program
// 5. [] token program
// 6. [] rent sysvar
func buildCreateAssociatedTokenAccountIx(payer, owner, mint, ata common.PublicKey) types.Instruction {
	ataProg := common.PublicKeyFromString(associatedTokenProgramID)

	// well-known program/sysvar ids
	systemProgram := common.PublicKeyFromString("11111111111111111111111111111111")
	tokenProgram := common.PublicKeyFromString("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	rentSysvar := common.PublicKeyFromString("SysvarRent111111111111111111111111111111111")

	return types.Instruction{
		ProgramID: ataProg,
		Accounts: []types.AccountMeta{
			{PubKey: payer, IsSigner: true, IsWritable: true},
			{PubKey: ata, IsSigner: false, IsWritable: true},
			{PubKey: owner, IsSigner: false, IsWritable: false},
			{PubKey: mint, IsSigner: false, IsWritable: false},
			{PubKey: systemProgram, IsSigner: false, IsWritable: false},
			{PubKey: tokenProgram, IsSigner: false, IsWritable: false},
			{PubKey: rentSysvar, IsSigner: false, IsWritable: false},
		},
		// ATA program: Create instruction has empty data (0 bytes)
		Data: []byte{},
	}
}

func (e *TokenTransferExecutorSolana) accountExists(ctx context.Context, address string) (bool, error) {
	addr := strings.TrimSpace(address)
	if addr == "" {
		return false, nil
	}

	_, err := e.RPC.GetAccountInfo(ctx, addr)
	if err == nil {
		return true, nil
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "not found") ||
		strings.Contains(msg, "could not find account") ||
		strings.Contains(msg, "invalid param") ||
		strings.Contains(msg, "account does not exist") {
		return false, nil
	}
	return false, err
}

// normalizeToAccount converts signerAny to blocto types.Account.
// Supports:
// - types.Account / *types.Account
// - []byte (len 64)
// - string: JSON array "[1,2,...]" (your SecretManager format)
func normalizeToAccount(signerAny any) (types.Account, error) {
	switch t := signerAny.(type) {
	case types.Account:
		return t, nil
	case *types.Account:
		if t == nil {
			return types.Account{}, ErrTokenTransferSignerEmpty
		}
		return *t, nil
	case []byte:
		if len(t) != 64 {
			return types.Account{}, fmt.Errorf("%w: want 64 bytes, got %d", ErrTokenTransferInvalidPrivKey, len(t))
		}
		acc, err := types.AccountFromBytes(t)
		if err != nil {
			return types.Account{}, fmt.Errorf("token_transfer_executor: AccountFromBytes: %w", err)
		}
		return acc, nil
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return types.Account{}, ErrTokenTransferSignerEmpty
		}
		var ints []int
		if err := json.Unmarshal([]byte(s), &ints); err != nil {
			return types.Account{}, fmt.Errorf("%w: signer string is not json int array: %v", ErrTokenTransferInvalidSigner, err)
		}
		b := make([]byte, len(ints))
		for i, v := range ints {
			if v < 0 || v > 255 {
				return types.Account{}, fmt.Errorf("%w: byte out of range at %d: %d", ErrTokenTransferInvalidPrivKey, i, v)
			}
			b[i] = byte(v)
		}
		if len(b) != 64 {
			return types.Account{}, fmt.Errorf("%w: want 64 bytes, got %d", ErrTokenTransferInvalidPrivKey, len(b))
		}
		acc, err := types.AccountFromBytes(b)
		if err != nil {
			return types.Account{}, fmt.Errorf("token_transfer_executor: AccountFromBytes(json): %w", err)
		}
		return acc, nil
	default:
		return types.Account{}, fmt.Errorf("%w: %T", ErrTokenTransferInvalidSigner, signerAny)
	}
}

func maskShort(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	if len(t) <= 10 {
		return t
	}
	return t[:4] + "***" + t[len(t)-4:]
}
