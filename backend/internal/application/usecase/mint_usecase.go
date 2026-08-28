// backend/internal/application/usecase/mint_usecase.go
package usecase

import (
	"context"
	"errors"

	invdom "narratives/internal/domain/inventory"
	mintdom "narratives/internal/domain/mint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

var ErrCompanyIDMissing = errors.New("companyId not found in context")

// ============================================================
// Mint request ports
// ============================================================

// MintRequestForUsecase は、MintUsecase が mint 実行フローを進めるために
// 必要となる MintRequest 情報だけを集約した DTO です。
type MintRequestForUsecase struct {
	ID string

	// TokenBlueprintID は、metadata URI の確保や tokenBlueprint minted 化に使います。
	TokenBlueprintID string

	// ActorID は、metadata URI 確保や tokenBlueprint minted 化の実行者として使います。
	ActorID string

	// 受取先アドレス（ブランドウォレット等）
	// NOTE:
	// - これは「NFT/トークンを受け取るアドレス」であり、FeePayer（ガス支払い）ではありません。
	// - FeePayer はインフラ側（mint/transfer 実装）で master wallet に統一しています。
	ToAddress string

	// productId ごとに 1 ミントしたい場合の productId 一覧。
	ProductIDs []string

	BlueprintName   string
	BlueprintSymbol string

	MetadataURI string
}

// MintRequestPort は、MintUsecase から見た「ミント対象 MintRequest」の
// 取得を行うためのポートです。
type MintRequestPort interface {
	// LoadForMinting:
	// - ミント実行に必要な情報をロードします。
	// - TokenBlueprintID / ActorID / ToAddress / ProductIDs / BlueprintName /
	//   BlueprintSymbol / MetadataURI を返す想定です。
	LoadForMinting(ctx context.Context, id string) (*MintRequestForUsecase, error)
}

// MintProductMintRecorder は、1 product の mint 成功結果を保存するためのポートです.
//
// - Firestore 実装側では productId と assetId の 1:1 token record を保存します。
// - 親 Mint の status=MINTED 更新はここでは行わず、全 task 完了時に MintUsecase 側で行います。
type MintProductMintRecorder interface {
	RecordProductAsMinted(
		ctx context.Context,
		mintID string,
		minted MintedTokenForUsecase,
	) error
}

// ============================================================
// Token mint dependency
// ============================================================

type TokenMintPort interface {
	MintProducts(ctx context.Context, input MintProductsInput) ([]MintedTokenForUsecase, error)
}

// ============================================================
// Mint task dependency
// ============================================================

// MintTaskEnqueuer は、Cloud Tasks 等に「次の1件mint処理」を投入するためのポートです。
type MintTaskEnqueuer interface {
	EnqueueMintTask(ctx context.Context, mintID string) error
}

// ============================================================
// TokenBlueprint dependencies
// ============================================================

type TokenBlueprintMetadataEnsurer interface {
	EnsureMetadataURI(
		ctx context.Context,
		tb *tbdom.TokenBlueprint,
		actorID string,
	) (*tbdom.TokenBlueprint, error)
}

type TokenBlueprintMintMarker interface {
	MarkTokenBlueprintMinted(
		ctx context.Context,
		tokenBlueprintID string,
		actorID string,
	) (*tbdom.TokenBlueprint, error)
}

// ============================================================
// Inventory dependency
// ============================================================

type InventoryUpserter interface {
	UpsertFromMint(
		ctx context.Context,
		tokenBlueprintID string,
		productBlueprintID string,
		productIDs []string,
	) ([]invdom.Mint, error)
}

// ============================================================
// MintUsecase
// ============================================================

type MintUsecase struct {
	prodRepo mintdom.MintProductionRepo

	tbRepo tbdom.RepositoryPort

	mintRepo     mintdom.MintRepository
	mintTaskRepo mintdom.MintProductTaskRepository

	mintRequestPort       MintRequestPort
	mintProductMintRecord MintProductMintRecorder

	mintTaskEnqueuer MintTaskEnqueuer

	mintResultMapper *MintResultMapper

	passedProductLister mintdom.PassedProductLister

	tokenMinter TokenMintPort

	inventoryUC InventoryUpserter

	tbMetadataEnsurer TokenBlueprintMetadataEnsurer
	tbMintMarker      TokenBlueprintMintMarker
}

func NewMintUsecase(
	prodRepo mintdom.MintProductionRepo,
	tbRepo tbdom.RepositoryPort,
	mintRepo mintdom.MintRepository,
	passedProductLister mintdom.PassedProductLister,
	tokenMinter TokenMintPort,
) *MintUsecase {
	var mintRequestPort MintRequestPort
	if p, ok := any(mintRepo).(MintRequestPort); ok {
		mintRequestPort = p
	}

	var mintProductMintRecord MintProductMintRecorder
	if p, ok := any(mintRepo).(MintProductMintRecorder); ok {
		mintProductMintRecord = p
	}

	return &MintUsecase{
		prodRepo:              prodRepo,
		tbRepo:                tbRepo,
		mintRepo:              mintRepo,
		mintTaskRepo:          nil,
		mintRequestPort:       mintRequestPort,
		mintProductMintRecord: mintProductMintRecord,
		mintTaskEnqueuer:      nil,
		mintResultMapper:      NewMintResultMapper(),
		passedProductLister:   passedProductLister,
		tokenMinter:           tokenMinter,
		inventoryUC:           nil,
		tbMetadataEnsurer:     nil,
		tbMintMarker:          nil,
	}
}

func (u *MintUsecase) SetInventoryUsecase(uc *InventoryUsecase) {
	if u == nil {
		return
	}

	var _ InventoryUpserter = uc
	u.inventoryUC = uc
}

func (u *MintUsecase) SetMintTaskRepository(
	repo mintdom.MintProductTaskRepository,
) {
	if u == nil {
		return
	}

	u.mintTaskRepo = repo
}

func (u *MintUsecase) SetMintTaskEnqueuer(enqueuer MintTaskEnqueuer) {
	if u == nil {
		return
	}

	u.mintTaskEnqueuer = enqueuer
}

func (u *MintUsecase) SetMintProductMintRecorder(
	recorder MintProductMintRecorder,
) {
	if u == nil {
		return
	}

	u.mintProductMintRecord = recorder
}

func (u *MintUsecase) SetTokenBlueprintMetadataEnsurer(
	e TokenBlueprintMetadataEnsurer,
) {
	if u == nil {
		return
	}

	u.tbMetadataEnsurer = e
}

func (u *MintUsecase) SetTokenBlueprintMintMarker(
	marker TokenBlueprintMintMarker,
) {
	if u == nil {
		return
	}

	u.tbMintMarker = marker
}
