// backend/internal/application/usecase/transfer_usecase.go
package usecase

import (
	"context"
	"time"

	applicationport "narratives/internal/application/port"
	orderdom "narratives/internal/domain/order"
)

// ============================================================
// Ports and DTOs
// ============================================================

type ModelTokenPair struct {
	ModelID          string `json:"modelId"`
	TokenBlueprintID string `json:"tokenBlueprintId"`
}

type VerifyInput struct {
	AvatarID  string `json:"avatarId"`
	ProductID string `json:"productId"`
}

type VerifyResult struct {
	AvatarID  string `json:"avatarId"`
	ProductID string `json:"productId"`

	ScannedModelID          string `json:"scannedModelId"`
	ScannedTokenBlueprintID string `json:"scannedTokenBlueprintId"`

	PurchasedPairs []ModelTokenPair `json:"purchasedPairs"`

	Matched bool            `json:"matched"`
	Match   *ModelTokenPair `json:"match,omitempty"`
}

type ScanVerifier interface {
	VerifyMatch(
		ctx context.Context,
		in VerifyInput,
	) (VerifyResult, error)
}

type ReturnOpeningHandler interface {
	PromoteUnopenedToOpened(
		ctx context.Context,
		in PromoteUnopenedToOpenedInput,
	) (ReturnRequestResult, error)
}

// ============================================================
// Usecase
// ============================================================

type TransferUsecase struct {
	verifier  ScanVerifier
	orderRepo applicationport.OrderRepoForTransfer
	tokenRepo applicationport.TokenResolver

	brandWallet  applicationport.BrandWalletResolver
	avatarWallet applicationport.AvatarWalletResolver

	brandDisplay  applicationport.BrandGetter
	avatarDisplay applicationport.AvatarDisplayResolver

	resaleRepo applicationport.ResaleGetter

	salesReceivableUC *SalesReceivableUsecase
	bankPayoutUC      *BankPayoutUsecase

	returnOpening ReturnOpeningHandler

	executionUC *TokenTransferExecutionUsecase
	inventoryUC *InventoryUsecase

	now func() time.Time
}

func NewTransferUsecase(
	verifier ScanVerifier,
	orderRepo applicationport.OrderRepoForTransfer,
	tokenRepo applicationport.TokenResolver,
	brandWallet applicationport.BrandWalletResolver,
	avatarWallet applicationport.AvatarWalletResolver,
	brandDisplay applicationport.BrandGetter,
	avatarDisplay applicationport.AvatarDisplayResolver,
	executionUC *TokenTransferExecutionUsecase,
	inventoryUC *InventoryUsecase,
) *TransferUsecase {
	return &TransferUsecase{
		verifier:  verifier,
		orderRepo: orderRepo,
		tokenRepo: tokenRepo,

		brandWallet:  brandWallet,
		avatarWallet: avatarWallet,

		brandDisplay:  brandDisplay,
		avatarDisplay: avatarDisplay,

		executionUC: executionUC,
		inventoryUC: inventoryUC,

		now: time.Now,
	}
}

func (u *TransferUsecase) WithResaleTransferDependencies(
	resaleRepo applicationport.ResaleGetter,
) *TransferUsecase {
	if u != nil {
		u.resaleRepo = resaleRepo
	}

	return u
}

func (u *TransferUsecase) WithResaleReceivableDependencies(
	salesReceivableUC *SalesReceivableUsecase,
) *TransferUsecase {
	if u != nil {
		u.salesReceivableUC = salesReceivableUC
	}

	return u
}

func (u *TransferUsecase) WithBankPayoutDependencies(
	bankPayoutUC *BankPayoutUsecase,
) *TransferUsecase {
	if u != nil {
		u.bankPayoutUC = bankPayoutUC
	}

	return u
}

func (u *TransferUsecase) WithReturnOpeningHandler(
	returnOpening ReturnOpeningHandler,
) *TransferUsecase {
	if u != nil {
		u.returnOpening = returnOpening
	}

	return u
}

// ============================================================
// Verified scan input / result
// ============================================================

type TransferByVerifiedScanInput struct {
	AvatarID    string
	ProductID   string
	OperationID string
}

type TransferByVerifiedScanResult struct {
	MatchedOrderID     string
	MatchedInventoryID string
	MatchedModelID     string

	MatchedItemIndex int
	MatchedItemType  orderdom.OrderItemType
	MatchedResaleID  string

	ProductID        string
	AssetID          string
	TokenBlueprintID string

	FromWallet  string
	ToWallet    string
	TxSignature string

	FromDisplayName string
	ToDisplayName   string
}
