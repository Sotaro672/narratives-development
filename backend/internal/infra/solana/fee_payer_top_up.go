// backend/internal/infra/solana/fee_payer_top_up.go
package solana

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/common"
	"github.com/blocto/solana-go-sdk/program/system"
	"github.com/blocto/solana-go-sdk/types"
)

var (
	ErrFeePayerTopUpDisabled            = errors.New("fee payer auto top-up is disabled")
	ErrReserveWalletMissing             = errors.New("reserve wallet is not configured")
	ErrReserveWalletBalanceInsufficient = errors.New("reserve wallet balance is insufficient")
	ErrFeePayerTopUpInvalidConfig       = errors.New("fee payer top-up config is invalid")
)

const (
	defaultMinFeePayerBalanceSOL       = 0.5
	defaultFeePayerTargetBalanceSOL    = 2.0
	defaultReserveMinRemainingSOL      = 0.5
	defaultReserveTxFeeBufferSOL       = 0.01
	defaultFeePayerTopUpConfirmTimeout = 60 * time.Second
)

type FeePayerTopUpResult struct {
	FeePayerAddress       string
	ReserveAddress        string
	FeePayerBalanceBefore uint64
	FeePayerBalanceAfter  uint64
	ReserveBalanceBefore  uint64
	ReserveBalanceAfter   uint64
	TargetBalance         uint64
	TransferredLamports   uint64
	Signature             string
}

type feePayerTopUpConfig struct {
	MinBalance          uint64
	TargetBalance       uint64
	ReserveMinRemaining uint64
	ReserveTxFeeBuffer  uint64
}

// AutoTopUpEnabled は reserve wallet から fee payer への
// native SOL 自動補充を有効にするかを返します。
func AutoTopUpEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("SOLANA_AUTO_TOP_UP_ENABLED")))
	return v == "true" || v == "1" || v == "yes" || v == "on"
}

func loadFeePayerTopUpConfig() (feePayerTopUpConfig, error) {
	minBalance, err := envSOLToLamports(
		"SOLANA_MIN_FEE_PAYER_BALANCE_SOL",
		defaultMinFeePayerBalanceSOL,
	)
	if err != nil {
		return feePayerTopUpConfig{}, err
	}

	targetBalance, err := envSOLToLamports(
		"SOLANA_FEE_PAYER_TARGET_BALANCE_SOL",
		defaultFeePayerTargetBalanceSOL,
	)
	if err != nil {
		return feePayerTopUpConfig{}, err
	}

	reserveMinRemaining, err := envSOLToLamports(
		"SOLANA_RESERVE_MIN_REMAINING_SOL",
		defaultReserveMinRemainingSOL,
	)
	if err != nil {
		return feePayerTopUpConfig{}, err
	}

	reserveTxFeeBuffer, err := envSOLToLamports(
		"SOLANA_RESERVE_TX_FEE_BUFFER_SOL",
		defaultReserveTxFeeBufferSOL,
	)
	if err != nil {
		return feePayerTopUpConfig{}, err
	}

	if targetBalance == 0 {
		return feePayerTopUpConfig{}, fmt.Errorf(
			"%w: SOLANA_FEE_PAYER_TARGET_BALANCE_SOL must be greater than 0",
			ErrFeePayerTopUpInvalidConfig,
		)
	}

	if targetBalance <= minBalance {
		return feePayerTopUpConfig{}, fmt.Errorf(
			"%w: target balance must be greater than minimum balance: target=%d min=%d",
			ErrFeePayerTopUpInvalidConfig,
			targetBalance,
			minBalance,
		)
	}

	return feePayerTopUpConfig{
		MinBalance:          minBalance,
		TargetBalance:       targetBalance,
		ReserveMinRemaining: reserveMinRemaining,
		ReserveTxFeeBuffer:  reserveTxFeeBuffer,
	}, nil
}

// topUpFeePayerFromReserve は fee payer の残高を再確認し、
// minimum 未満の場合のみ reserve wallet から target balance まで補充します。
//
// 処理:
//  1. fee payer 残高を再取得
//  2. minimum 以上なら何もせず終了
//  3. reserve 残高を確認
//  4. target balance との差額を native SOL transfer
//  5. transaction confirmation 待ち
//  6. fee payer / reserve の残高を再確認
func topUpFeePayerFromReserve(
	ctx context.Context,
	rpcURL string,
	reserve *ReserveAuthority,
	feePayer common.PublicKey,
) (*FeePayerTopUpResult, error) {
	if !AutoTopUpEnabled() {
		return nil, ErrFeePayerTopUpDisabled
	}

	if reserve == nil {
		return nil, ErrReserveWalletMissing
	}

	if rpcURL == "" {
		var err error
		rpcURL, err = solanaRPCURLFromEnv()
		if err != nil {
			return nil, err
		}
	}

	var zeroPublicKey common.PublicKey
	if feePayer == zeroPublicKey {
		return nil, fmt.Errorf("fee payer public key is empty")
	}

	feePayerAddress := feePayer.ToBase58()
	reserveAddress := reserve.Account.PublicKey.ToBase58()

	if feePayerAddress == "" {
		return nil, fmt.Errorf("fee payer address is empty")
	}

	if reserveAddress == "" {
		return nil, fmt.Errorf("reserve wallet address is empty")
	}

	if reserveAddress == feePayerAddress {
		return nil, fmt.Errorf(
			"reserve wallet and fee payer must be different wallets: address=%s",
			feePayerAddress,
		)
	}

	if len(reserve.Account.PrivateKey) == 0 {
		return nil, fmt.Errorf("reserve wallet private key is empty")
	}

	cfg, err := loadFeePayerTopUpConfig()
	if err != nil {
		return nil, err
	}

	// 呼び出し側ですでに残高確認していても、ここでも再確認します。
	// 別workerが直前に補充済みなら二重送金を避けられます。
	feePayerBalance, err := getSolanaBalance(
		ctx,
		rpcURL,
		feePayerAddress,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get fee payer balance before top-up: %w",
			err,
		)
	}

	result := &FeePayerTopUpResult{
		FeePayerAddress:       feePayerAddress,
		ReserveAddress:        reserveAddress,
		FeePayerBalanceBefore: feePayerBalance,
		FeePayerBalanceAfter:  feePayerBalance,
		TargetBalance:         cfg.TargetBalance,
	}

	if feePayerBalance >= cfg.MinBalance {
		return result, nil
	}

	topUpLamports := cfg.TargetBalance - feePayerBalance
	if topUpLamports == 0 {
		return result, nil
	}

	reserveBalance, err := getSolanaBalance(
		ctx,
		rpcURL,
		reserveAddress,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get reserve wallet balance before top-up: %w",
			err,
		)
	}

	result.ReserveBalanceBefore = reserveBalance
	result.ReserveBalanceAfter = reserveBalance

	// reserve自身が送金transactionのfee payerになるため、
	// top-up額 + 最低残高 + fee buffer を要求します。
	requiredReserveBalance, err := addLamports(
		topUpLamports,
		cfg.ReserveMinRemaining,
		cfg.ReserveTxFeeBuffer,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"calculate required reserve balance: %w",
			err,
		)
	}

	if reserveBalance < requiredReserveBalance {
		return nil, fmt.Errorf(
			"%w: address=%s balance=%d required=%d topUp=%d minRemaining=%d feeBuffer=%d",
			ErrReserveWalletBalanceInsufficient,
			reserveAddress,
			reserveBalance,
			requiredReserveBalance,
			topUpLamports,
			cfg.ReserveMinRemaining,
			cfg.ReserveTxFeeBuffer,
		)
	}

	log.Printf(
		"[fee-payer-top-up] start reserve=%s feePayer=%s feePayerBalance=%d target=%d transfer=%d reserveBalance=%d",
		reserveAddress,
		feePayerAddress,
		feePayerBalance,
		cfg.TargetBalance,
		topUpLamports,
		reserveBalance,
	)

	signature, err := transferSOL(
		ctx,
		rpcURL,
		reserve.Account,
		feePayer,
		topUpLamports,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"transfer reserve SOL to fee payer: %w",
			err,
		)
	}

	result.Signature = signature
	result.TransferredLamports = topUpLamports

	updatedFeePayerBalance, err := getSolanaBalance(
		ctx,
		rpcURL,
		feePayerAddress,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get fee payer balance after top-up: %w",
			err,
		)
	}

	result.FeePayerBalanceAfter = updatedFeePayerBalance

	updatedReserveBalance, err := getSolanaBalance(
		ctx,
		rpcURL,
		reserveAddress,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get reserve wallet balance after top-up: %w",
			err,
		)
	}

	result.ReserveBalanceAfter = updatedReserveBalance

	if updatedFeePayerBalance < cfg.MinBalance {
		return nil, fmt.Errorf(
			"fee payer balance is still below minimum after reserve top-up: address=%s balance=%d min=%d signature=%s",
			feePayerAddress,
			updatedFeePayerBalance,
			cfg.MinBalance,
			signature,
		)
	}

	if updatedReserveBalance < cfg.ReserveMinRemaining {
		return nil, fmt.Errorf(
			"reserve wallet balance fell below minimum after top-up: address=%s balance=%d minRemaining=%d signature=%s",
			reserveAddress,
			updatedReserveBalance,
			cfg.ReserveMinRemaining,
			signature,
		)
	}

	log.Printf(
		"[fee-payer-top-up] completed reserve=%s feePayer=%s transfer=%d signature=%s feePayerBalanceBefore=%d feePayerBalanceAfter=%d reserveBalanceBefore=%d reserveBalanceAfter=%d",
		reserveAddress,
		feePayerAddress,
		topUpLamports,
		signature,
		result.FeePayerBalanceBefore,
		result.FeePayerBalanceAfter,
		result.ReserveBalanceBefore,
		result.ReserveBalanceAfter,
	)

	return result, nil
}

// transferSOL は native SOL を from から to へ送金します。
// reserve wallet 自身を FeePayer として署名し、
// 送金transactionのfeeもreserve側が負担します。
func transferSOL(
	ctx context.Context,
	rpcURL string,
	from types.Account,
	to common.PublicKey,
	lamports uint64,
) (string, error) {
	if rpcURL == "" {
		return "", fmt.Errorf("solana rpc url is empty")
	}

	if len(from.PrivateKey) == 0 {
		return "", fmt.Errorf(
			"SOL transfer signer private key is empty",
		)
	}

	var zeroPublicKey common.PublicKey

	if from.PublicKey == zeroPublicKey {
		return "", fmt.Errorf(
			"SOL transfer from public key is empty",
		)
	}

	if to == zeroPublicKey {
		return "", fmt.Errorf(
			"SOL transfer destination public key is empty",
		)
	}

	if from.PublicKey == to {
		return "", fmt.Errorf(
			"SOL transfer source and destination are identical: %s",
			from.PublicKey.ToBase58(),
		)
	}

	if lamports == 0 {
		return "", fmt.Errorf(
			"SOL transfer lamports is zero",
		)
	}

	cl := client.NewClient(rpcURL)

	var recentBlockhash string

	if err := withSolanaRPCRetry(
		ctx,
		"get latest blockhash for fee payer top-up",
		func() error {
			latest, err := cl.GetLatestBlockhash(ctx)
			if err != nil {
				return err
			}

			recentBlockhash = latest.Blockhash
			return nil
		},
	); err != nil {
		return "", fmt.Errorf(
			"get latest blockhash for fee payer top-up: %w",
			err,
		)
	}

	if recentBlockhash == "" {
		return "", fmt.Errorf(
			"latest blockhash is empty for fee payer top-up",
		)
	}

	instruction := system.Transfer(
		system.TransferParam{
			From:   from.PublicKey,
			To:     to,
			Amount: lamports,
		},
	)

	tx, err := types.NewTransaction(
		types.NewTransactionParam{
			Message: types.NewMessage(
				types.NewMessageParam{
					FeePayer:        from.PublicKey,
					RecentBlockhash: recentBlockhash,
					Instructions: []types.Instruction{
						instruction,
					},
				},
			),
			Signers: []types.Account{
				from,
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf(
			"build fee payer top-up transaction: %w",
			err,
		)
	}

	var signature string

	// 同じ署名済みtransactionをbounded retryします。
	// 応答だけ失敗して実際には送信済みだったケースでも、
	// 次回Cloud Task実行時にfee payer残高を再確認するため、
	// 不要な追加top-upを抑止できます。
	if err := withSolanaRPCRetry(
		ctx,
		"send fee payer top-up transaction",
		func() error {
			var sendErr error

			signature, sendErr = cl.SendTransaction(
				ctx,
				tx,
			)

			return sendErr
		},
	); err != nil {
		return "", fmt.Errorf(
			"send fee payer top-up transaction: %w",
			err,
		)
	}

	if signature == "" {
		return "", fmt.Errorf(
			"fee payer top-up transaction returned empty signature",
		)
	}

	confirmCtx, cancel := context.WithTimeout(
		ctx,
		defaultFeePayerTopUpConfirmTimeout,
	)
	defer cancel()

	if err := waitForSignatureConfirmed(
		confirmCtx,
		rpcURL,
		signature,
	); err != nil {
		return "", fmt.Errorf(
			"confirm fee payer top-up transaction signature=%s: %w",
			signature,
			err,
		)
	}

	return signature, nil
}

func addLamports(values ...uint64) (uint64, error) {
	var total uint64

	for _, value := range values {
		if ^uint64(0)-total < value {
			return 0, fmt.Errorf(
				"lamports overflow",
			)
		}

		total += value
	}

	return total, nil
}
