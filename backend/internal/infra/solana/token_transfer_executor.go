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
	"sync"
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

	// ✅ Safety: if FromWalletAddress is provided, it must match signer public key.
	ErrTokenTransferSignerWalletMismatch = errors.New("token_transfer_executor: signer public key does not match fromWalletAddress")
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

	// master fee payer (loaded lazily from Secret Manager via SOLANA_MINT_KEY_SECRET)
	masterOnce sync.Once
	masterAcc  *types.Account
	masterErr  error
}

// NewTokenTransferExecutorSolana constructs executor.
// RPC URL resolves from SOLANA_RPC_URL if url is empty.
func NewTokenTransferExecutorSolana(rpcURL string) *TokenTransferExecutorSolana {
	u := rpcURL
	if u == "" {
		u = os.Getenv("SOLANA_RPC_URL")
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

// loadMasterFeePayer loads master payer account once (best-effort cached).
// NOTE: LoadMintAuthority must exist in the same package (backend/internal/infra/solana).
func (e *TokenTransferExecutorSolana) loadMasterFeePayer(ctx context.Context) (*types.Account, error) {
	if e == nil {
		return nil, ErrTokenTransferNotConfigured
	}

	e.masterOnce.Do(func() {
		auth, err := LoadMintAuthority(ctx)
		if err != nil {
			e.masterErr = fmt.Errorf("token_transfer_executor: LoadMintAuthority: %w", err)
			return
		}
		acc := auth.Account
		e.masterAcc = &acc

		log.Printf(
			"[token_transfer_executor] loaded master fee payer: pubkey=%s",
			acc.PublicKey.ToBase58(),
		)
	})

	if e.masterErr != nil {
		return nil, e.masterErr
	}
	if e.masterAcc == nil {
		return nil, fmt.Errorf("token_transfer_executor: master fee payer is nil after load")
	}
	return e.masterAcc, nil
}

// ExecuteTransfer does:
// - derive ATA(from owner, mint) / ATA(to owner, mint)
// - create destination ATA if missing (payer=master)
// - SPL token transfer (amount default=1) (auth=from owner, signer=from)
// - send tx (FeePayer=master; signed by from signer + master payer)
func (e *TokenTransferExecutorSolana) ExecuteTransfer(ctx context.Context, in usecase.ExecuteTransferInput) (usecase.ExecuteTransferResult, error) {
	if e == nil || e.RPC == nil {
		return usecase.ExecuteTransferResult{}, ErrTokenTransferNotConfigured
	}

	// best-effort timeout wrapper (do not override caller cancellation)
	rpcCtx := ctx
	if e.Timeout > 0 {
		var cancel context.CancelFunc
		rpcCtx, cancel = context.WithTimeout(ctx, e.Timeout)
		defer cancel()
	}

	toWallet := in.ToWalletAddress
	if toWallet == "" {
		return usecase.ExecuteTransferResult{}, ErrTokenTransferToWalletEmpty
	}
	mintAddr := in.MintAddress
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

	// ✅ master fee payer (gas payer)
	masterAcc, err := e.loadMasterFeePayer(ctx)
	if err != nil {
		return usecase.ExecuteTransferResult{}, err
	}
	feePayer := *masterAcc

	// ✅ Safety: if FromWalletAddress is provided, ensure it matches signer pubkey
	fromWalletAddr := in.FromWalletAddress
	if fromWalletAddr != "" {
		if fromAcc.PublicKey.ToBase58() != fromWalletAddr {
			return usecase.ExecuteTransferResult{}, fmt.Errorf(
				"%w: signer=%s fromWalletAddress=%s",
				ErrTokenTransferSignerWalletMismatch,
				fromAcc.PublicKey.ToBase58(),
				fromWalletAddr,
			)
		}
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

	log.Printf("[token_transfer_executor] mint=%s fromOwner=%s toOwner=%s fromATA=%s toATA=%s feePayer=%s",
		mintAddr,
		fromOwner.ToBase58(),
		toOwner.ToBase58(),
		fromATA.ToBase58(),
		toATA.ToBase58(),
		feePayer.PublicKey.ToBase58(),
	)

	log.Printf(
		"[token_transfer_executor] start productId=%s avatarId=%s brandId=%s mint=%s amount=%d from=%s to=%s feePayer=%s",
		in.ProductID,
		in.AvatarID,
		in.BrandID,
		mintAddr,
		amt,
		fromOwner.ToBase58(),
		toWallet,
		feePayer.PublicKey.ToBase58(),
	)

	// 1) existence checks
	fromExists, err := e.accountExists(rpcCtx, fromATA.ToBase58())
	if err != nil {
		return usecase.ExecuteTransferResult{}, fmt.Errorf("token_transfer_executor: check from ATA failed: %w", err)
	}
	if !fromExists {
		return usecase.ExecuteTransferResult{}, ErrTokenTransferSourceAtaAbsent
	}

	toExists, err := e.accountExists(rpcCtx, toATA.ToBase58())
	if err != nil {
		return usecase.ExecuteTransferResult{}, fmt.Errorf("token_transfer_executor: check to ATA failed: %w", err)
	}

	log.Printf("[token_transfer_executor] exists: fromATA=%t toATA=%t", fromExists, toExists)

	// 2) build instructions
	ins := make([]types.Instruction, 0, 3)

	// 2-1) create dest ATA if missing (payer=master fee payer)
	if !toExists {
		ins = append(ins, buildCreateAssociatedTokenAccountIx(feePayer.PublicKey, toOwner, mint, toATA))
		log.Printf(
			"[token_transfer_executor] will create ATA: owner=%s mint=%s ata=%s payer=%s",
			toWallet,
			mintAddr,
			toATA.ToBase58(),
			feePayer.PublicKey.ToBase58(),
		)
	}

	// 2-2) token transfer (auth=fromOwner)
	ins = append(ins, token.Transfer(token.TransferParam{
		From:   fromATA,
		To:     toATA,
		Auth:   fromOwner,
		Amount: amt,
	}))

	log.Printf("[token_transfer_executor] instruction_count=%d will_create_ata=%t", len(ins), !toExists)

	// 3) recent blockhash
	latest, err := e.RPC.GetLatestBlockhash(rpcCtx)
	if err != nil {
		return usecase.ExecuteTransferResult{}, fmt.Errorf("token_transfer_executor: GetLatestBlockhash: %w", err)
	}

	// 4) tx (FeePayer=master)
	signers := make([]types.Account, 0, 2)
	signers = append(signers, fromAcc)
	if feePayer.PublicKey != fromAcc.PublicKey {
		signers = append(signers, feePayer)
	}

	tx, err := types.NewTransaction(types.NewTransactionParam{
		Message: types.NewMessage(types.NewMessageParam{
			FeePayer:        feePayer.PublicKey,
			RecentBlockhash: latest.Blockhash,
			Instructions:    ins,
		}),
		Signers: signers,
	})
	if err != nil {
		return usecase.ExecuteTransferResult{}, fmt.Errorf("token_transfer_executor: NewTransaction: %w", err)
	}

	// 5) send
	sig, err := e.RPC.SendTransaction(rpcCtx, tx)
	if err != nil {
		return usecase.ExecuteTransferResult{}, fmt.Errorf("token_transfer_executor: SendTransaction: %w", err)
	}

	log.Printf(
		"[token_transfer_executor] submitted tx=%s mint=%s fromATA=%s toATA=%s createdATA=%t feePayer=%s",
		sig,
		mintAddr,
		fromATA.ToBase58(),
		toATA.ToBase58(),
		!toExists,
		feePayer.PublicKey.ToBase58(),
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
	addr := address
	if addr == "" {
		return false, nil
	}

	info, err := e.RPC.GetAccountInfo(ctx, addr)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "not found") ||
			strings.Contains(msg, "could not find account") ||
			strings.Contains(msg, "invalid param") ||
			strings.Contains(msg, "account does not exist") {
			return false, nil
		}
		return false, err
	}

	// ✅ IMPORTANT:
	// blocto SDK's GetAccountInfo returns a struct. In some cases, "not found" can be expressed
	// as a zero-value struct with err=nil. Treat it as "not exists".
	if isZeroAccountInfo(info) {
		return false, nil
	}

	return true, nil
}

func isZeroAccountInfo(info client.AccountInfo) bool {
	// Heuristic: an existing account cannot have lamports=0 and empty owner and empty data simultaneously.
	// (Lamports can be 0 only if account is not really present or has been reclaimed.)
	if info.Lamports != 0 {
		return false
	}
	if !isZeroPublicKey(info.Owner) {
		return false
	}
	if len(info.Data) != 0 {
		return false
	}
	if info.Executable {
		return false
	}
	if info.RentEpoch != 0 {
		return false
	}
	return true
}

func isZeroPublicKey(pk common.PublicKey) bool {
	// common.PublicKey is a fixed-size byte array type; compare against zero value.
	var z common.PublicKey
	return pk == z
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
		s := t
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
