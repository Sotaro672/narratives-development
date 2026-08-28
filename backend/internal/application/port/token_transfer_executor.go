// backend/internal/application/port/token_transfer_executor.go
package port

import "context"

// TokenTransferExecutor executes the on-chain token transfer.
type TokenTransferExecutor interface {
	ExecuteTransfer(
		ctx context.Context,
		in ExecuteTransferInput,
	) (ExecuteTransferResult, error)
}

// ExecuteTransferInput contains the data required to execute one token transfer.
type ExecuteTransferInput struct {
	ProductID   string
	OperationID string

	FromAvatarID     string
	ToAvatarID       string
	FromBrandID      string
	BrandID          string
	ModelID          string
	TokenBlueprintID string

	AssetID string

	FromWalletAddress string
	ToWalletAddress   string
}

// ExecuteTransferResult contains the result of a completed token transfer.
type ExecuteTransferResult struct {
	TxSignature string
}
