// backend/internal/application/usecase/mint_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	invdom "narratives/internal/domain/inventory"
	mintdom "narratives/internal/domain/mint"
	tokendom "narratives/internal/domain/token"
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
// 取得および更新を行うためのポートです。
//
// 現在のフローでは「1商品=1Mint」モードのみを想定しています。
type MintRequestPort interface {
	// LoadForMinting:
	// - ミント実行に必要な情報をロードします。
	// - TokenBlueprintID / ActorID / ToAddress / ProductIDs / BlueprintName /
	//   BlueprintSymbol / MetadataURI を返す想定です。
	LoadForMinting(ctx context.Context, id string) (*MintRequestForUsecase, error)

	// MarkProductsAsMinted:
	// - productId ごとに 1 ミントした結果一覧で MintRequest / Token 情報を更新します。
	// - 実装側で:
	//   - productId, mintAddress の 1:1 マッピングを tokens コレクション等に保存
	//   - MintRequest (mints テーブル) 自体も minted=true にする
	MarkProductsAsMinted(ctx context.Context, id string, minted []MintedTokenForUsecase) error
}

// ============================================================
// Token mint dependency
// ============================================================

type TokenMintPort interface {
	MintProducts(ctx context.Context, input MintProductsInput) ([]MintedTokenForUsecase, error)
}

// ============================================================
// TokenBlueprint dependencies
// ============================================================

type TokenBlueprintMetadataEnsurer interface {
	EnsureMetadataURI(ctx context.Context, tb *tbdom.TokenBlueprint, actorID string) (*tbdom.TokenBlueprint, error)
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
// MintResultMapper
// ============================================================

type MintResultMapper struct{}

func NewMintResultMapper() *MintResultMapper {
	return &MintResultMapper{}
}

func (m *MintResultMapper) FromMint(ent mintdom.Mint) *tokendom.MintResult {
	return &tokendom.MintResult{
		Signature:   ent.OnChainTxSignature,
		MintAddress: "",
		Slot:        0,
	}
}

func (m *MintResultMapper) ApplyOnchainResult(ent *mintdom.Mint, result *tokendom.MintResult) error {
	if ent == nil {
		return errors.New("mint entity is nil")
	}
	if result == nil {
		return nil
	}

	if result.Signature != "" {
		ent.OnChainTxSignature = result.Signature
	}

	return nil
}

// ============================================================
// MintUsecase
// ============================================================

type MintUsecase struct {
	prodRepo mintdom.MintProductionRepo

	tbRepo tbdom.RepositoryPort

	mintRepo mintdom.MintRepository

	mintRequestPort MintRequestPort

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

	return &MintUsecase{
		prodRepo:            prodRepo,
		tbRepo:              tbRepo,
		mintRepo:            mintRepo,
		mintRequestPort:     mintRequestPort,
		mintResultMapper:    NewMintResultMapper(),
		passedProductLister: passedProductLister,
		tokenMinter:         tokenMinter,
		inventoryUC:         nil,
		tbMetadataEnsurer:   nil,
		tbMintMarker:        nil,
	}
}

func (u *MintUsecase) SetInventoryUsecase(uc *InventoryUsecase) {
	if u == nil {
		return
	}

	var _ InventoryUpserter = uc
	u.inventoryUC = uc
}

func (u *MintUsecase) SetTokenBlueprintMetadataEnsurer(e TokenBlueprintMetadataEnsurer) {
	if u == nil {
		return
	}
	u.tbMetadataEnsurer = e
}

func (u *MintUsecase) SetTokenBlueprintMintMarker(marker TokenBlueprintMintMarker) {
	if u == nil {
		return
	}
	u.tbMintMarker = marker
}

func (u *MintUsecase) UpdateRequestInfo(
	ctx context.Context,
	productionID string,
	tokenBlueprintID string,
	scheduledBurnDate *string,
) (*tokendom.MintResult, error) {
	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if u.mintRepo == nil {
		return nil, errors.New("mint repo is nil")
	}
	if u.passedProductLister == nil {
		return nil, errors.New("passedProductLister is nil")
	}
	if u.tbRepo == nil {
		return nil, errors.New("tokenBlueprint repo is nil")
	}

	pid := productionID
	if pid == "" {
		return nil, errors.New("productionID is empty")
	}

	tbID := tokenBlueprintID
	if tbID == "" {
		return nil, errors.New("tokenBlueprintID is empty")
	}

	memberID := MemberIDFromContext(ctx)
	if memberID == "" {
		return nil, errors.New("memberID not found in context")
	}

	now := time.Now().UTC()

	tb, err := u.tbRepo.GetByID(ctx, tbID)
	if err != nil {
		return nil, err
	}
	if tb == nil {
		return nil, errors.New("tokenBlueprint not found")
	}

	brandID := tb.BrandID
	if brandID == "" {
		return nil, errors.New("brandID is empty on tokenBlueprint")
	}

	passedProductIDs, err := u.passedProductLister.ListPassedProductIDsByProductionID(ctx, pid)
	if err != nil {
		return nil, err
	}
	if len(passedProductIDs) == 0 {
		return nil, errors.New("no passed products for this production")
	}

	mintEntity, err := mintdom.NewMint(
		pid,
		brandID,
		tbID,
		passedProductIDs,
		memberID,
		now,
	)
	if err != nil {
		return nil, err
	}

	mintEntity.ID = pid
	mintEntity.Minted = false
	mintEntity.MintedAt = nil

	if scheduledBurnDate != nil {
		if s := *scheduledBurnDate; s != "" {
			t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
			if err != nil {
				return nil, errors.New("invalid scheduledBurnDate format (expected YYYY-MM-DD)")
			}
			utc := t.UTC()
			mintEntity.ScheduledBurnDate = &utc
		}
	}

	if _, err := u.mintRepo.Create(ctx, mintEntity); err != nil {
		return nil, err
	}

	result, err := u.MintFromMintRequest(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("onchain mint failed after mint request was created: %w", err)
	}

	if result == nil {
		return nil, errors.New("onchain mint returned nil result")
	}

	return result, nil
}

func (u *MintUsecase) resolveProductBlueprintIDFromProduction(ctx context.Context, productionID string) string {
	if u == nil || u.prodRepo == nil {
		return ""
	}
	if productionID == "" {
		return ""
	}

	productBlueprintID, err := u.prodRepo.GetProductBlueprintIDByProductionID(ctx, productionID)
	if err != nil {
		return ""
	}

	return productBlueprintID
}

func validateProductIDs(productIDs []string) error {
	seen := make(map[string]struct{}, len(productIDs))

	for _, id := range productIDs {
		if id == "" {
			return mintdom.ErrInvalidProducts
		}
		if _, ok := seen[id]; ok {
			return mintdom.ErrInvalidProducts
		}
		seen[id] = struct{}{}
	}

	return nil
}

func normalizeProductIDs(productIDs []string) []string {
	out := make([]string, 0, len(productIDs))
	for _, id := range productIDs {
		p := strings.TrimSpace(id)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func lastMintResult(minted []MintedTokenForUsecase) *tokendom.MintResult {
	for i := len(minted) - 1; i >= 0; i-- {
		if minted[i].Result != nil {
			return minted[i].Result
		}
	}
	return nil
}

func (u *MintUsecase) ensureMetadataURI(
	ctx context.Context,
	tokenBlueprintID string,
	actorID string,
	currentMetadataURI string,
) (string, error) {
	metadataURI := strings.TrimSpace(currentMetadataURI)

	tbID := strings.TrimSpace(tokenBlueprintID)
	if tbID == "" {
		return metadataURI, nil
	}

	if u.tbMetadataEnsurer == nil {
		return metadataURI, nil
	}

	if u.tbRepo == nil {
		return "", fmt.Errorf("tokenBlueprint repo is nil")
	}

	tb, err := u.tbRepo.GetByID(ctx, tbID)
	if err != nil {
		return "", fmt.Errorf("get tokenBlueprint for metadata ensure: %w", err)
	}
	if tb == nil {
		return "", fmt.Errorf("tokenBlueprint not found (id=%s)", tbID)
	}

	updated, err := u.tbMetadataEnsurer.EnsureMetadataURI(ctx, tb, actorID)
	if err != nil {
		return "", fmt.Errorf("ensure metadata uri: %w", err)
	}
	if updated == nil {
		updated = tb
	}

	return strings.TrimSpace(updated.MetadataURI), nil
}

func (u *MintUsecase) MintFromMintRequest(ctx context.Context, mintRequestID string) (*tokendom.MintResult, error) {
	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if mintRequestID == "" {
		return nil, errors.New("mintRequestID is empty")
	}
	if u.mintRepo == nil {
		return nil, errors.New("mint repo is nil")
	}
	if u.mintResultMapper == nil {
		return nil, errors.New("mint result mapper is nil")
	}

	mintEntValue, err := u.mintRepo.GetByID(ctx, mintRequestID)
	if err != nil {
		return nil, err
	}
	mintEnt := &mintEntValue

	passedProductIDs := normalizeProductIDs(mintEnt.Products)
	if err := validateProductIDs(passedProductIDs); err != nil {
		return nil, err
	}

	tbID := strings.TrimSpace(mintEnt.TokenBlueprintID)
	if tbID == "" {
		return nil, errors.New("tokenBlueprintID is empty on mint")
	}

	pbID := u.resolveProductBlueprintIDFromProduction(ctx, mintRequestID)
	if pbID == "" {
		return nil, errors.New("productBlueprintID is empty (cannot upsert inventory)")
	}

	if len(passedProductIDs) == 0 {
		return nil, errors.New("no passed products for this mint request")
	}

	var result *tokendom.MintResult

	if mintEnt.Minted {
		result = u.mintResultMapper.FromMint(*mintEnt)
	} else {
		if u.tokenMinter == nil {
			return nil, errors.New("token minter is nil")
		}
		if u.mintRequestPort == nil {
			return nil, errors.New("mint request port is nil")
		}

		req, err := u.mintRequestPort.LoadForMinting(ctx, mintRequestID)
		if err != nil {
			return nil, fmt.Errorf("load mint request for minting: %w", err)
		}
		if req == nil {
			return nil, fmt.Errorf("mint request %s is nil", mintRequestID)
		}

		reqID := strings.TrimSpace(req.ID)
		if reqID == "" {
			reqID = mintRequestID
		}

		reqTBID := strings.TrimSpace(req.TokenBlueprintID)
		if reqTBID == "" {
			reqTBID = tbID
		}

		actorID := strings.TrimSpace(req.ActorID)
		if actorID == "" {
			actorID = strings.TrimSpace(mintEnt.CreatedBy)
		}
		if actorID == "" {
			actorID = strings.TrimSpace(MemberIDFromContext(ctx))
		}

		productIDs := normalizeProductIDs(req.ProductIDs)
		if len(productIDs) == 0 {
			productIDs = passedProductIDs
		}
		if err := validateProductIDs(productIDs); err != nil {
			return nil, err
		}

		metadataURI, err := u.ensureMetadataURI(
			ctx,
			reqTBID,
			actorID,
			req.MetadataURI,
		)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(metadataURI) == "" {
			return nil, fmt.Errorf("mint request %s has empty MetadataURI", reqID)
		}

		toAddress := strings.TrimSpace(req.ToAddress)
		if toAddress == "" {
			return nil, fmt.Errorf("mint request %s has empty ToAddress", reqID)
		}

		name := strings.TrimSpace(req.BlueprintName)
		symbol := strings.TrimSpace(req.BlueprintSymbol)
		if name == "" || symbol == "" {
			return nil, fmt.Errorf("mint request %s has empty name or symbol", reqID)
		}

		minted, err := u.tokenMinter.MintProducts(ctx, MintProductsInput{
			ToAddress:       toAddress,
			ProductIDs:      productIDs,
			BlueprintName:   name,
			BlueprintSymbol: symbol,
			MetadataURI:     metadataURI,
		})
		if err != nil {
			return nil, err
		}
		if len(minted) == 0 {
			return nil, fmt.Errorf("onchain mint succeeded but minted list is empty (mintRequestId=%s)", mintRequestID)
		}

		if err := u.mintRequestPort.MarkProductsAsMinted(ctx, reqID, minted); err != nil {
			result = lastMintResult(minted)
			return result, fmt.Errorf("mark mint request as minted (per-product): %w", err)
		}

		if u.tbMintMarker != nil && reqTBID != "" {
			_, _ = u.tbMintMarker.MarkTokenBlueprintMinted(ctx, reqTBID, actorID)
		}

		result = lastMintResult(minted)
		if result == nil {
			return nil, fmt.Errorf("onchain mint succeeded but result is nil (mintRequestId=%s)", mintRequestID)
		}
	}

	if u.inventoryUC == nil {
		return nil, errors.New("inventory usecase is nil (cannot upsert inventory)")
	}

	if _, invErr := u.inventoryUC.UpsertFromMint(
		ctx,
		tbID,
		pbID,
		passedProductIDs,
	); invErr != nil {
		return nil, invErr
	}

	return result, nil
}
