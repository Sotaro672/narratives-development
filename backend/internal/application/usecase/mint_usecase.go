// backend\internal\application\usecase\mint_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	resolver "narratives/internal/application/resolver"
	branddom "narratives/internal/domain/brand"
	domcommon "narratives/internal/domain/common"
	inspectiondom "narratives/internal/domain/inspection"
	invdom "narratives/internal/domain/inventory"
	mintdom "narratives/internal/domain/mint"
	pbpdom "narratives/internal/domain/productBlueprint"
	tokendom "narratives/internal/domain/token"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

var ErrCompanyIDMissing = errors.New("companyId not found in context")

type TokenMintPort interface {
	MintFromMintRequest(ctx context.Context, mintID string) (*tokendom.MintResult, error)
}

type InventoryUpserter interface {
	UpsertFromMintByModel(
		ctx context.Context,
		tokenBlueprintID string,
		productBlueprintID string,
		modelID string,
		productIDs []string,
	) (invdom.Mint, error)
}

type TokenBlueprintBucketEnsurer interface {
	EnsureKeepObjects(ctx context.Context, tokenBlueprintID string) error
}

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

type MintUsecase struct {
	pbRepo    mintdom.MintProductBlueprintRepo
	prodRepo  mintdom.MintProductionRepo
	inspRepo  mintdom.MintInspectionRepo
	modelRepo mintdom.MintModelRepo

	tbBucketEnsurer   TokenBlueprintBucketEnsurer
	tbMetadataEnsurer TokenBlueprintMetadataEnsurer
	tbMintMarker      TokenBlueprintMintMarker

	tbRepo tbdom.RepositoryPort

	brandSvc *branddom.Service

	mintRepo mintdom.MintRepository

	mintResultMapper *MintResultMapper

	passedProductLister mintdom.PassedProductLister

	tokenMinter TokenMintPort

	inventoryUC InventoryUpserter

	nameResolver *resolver.NameResolver
}

func NewMintUsecase(
	pbRepo mintdom.MintProductBlueprintRepo,
	prodRepo mintdom.MintProductionRepo,
	inspRepo mintdom.MintInspectionRepo,
	modelRepo mintdom.MintModelRepo,
	tbRepo tbdom.RepositoryPort,
	brandSvc *branddom.Service,
	mintRepo mintdom.MintRepository,
	passedProductLister mintdom.PassedProductLister,
	tokenMinter TokenMintPort,
) *MintUsecase {
	return &MintUsecase{
		pbRepo:              pbRepo,
		prodRepo:            prodRepo,
		inspRepo:            inspRepo,
		modelRepo:           modelRepo,
		tbRepo:              tbRepo,
		brandSvc:            brandSvc,
		mintRepo:            mintRepo,
		mintResultMapper:    NewMintResultMapper(),
		passedProductLister: passedProductLister,
		tokenMinter:         tokenMinter,
		tbBucketEnsurer:     nil,
		tbMetadataEnsurer:   nil,
		tbMintMarker:        nil,
		inventoryUC:         nil,
		nameResolver:        nil,
	}
}

func (u *MintUsecase) SetNameResolver(r *resolver.NameResolver) {
	if u == nil {
		return
	}
	u.nameResolver = r
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
) (inspectiondom.InspectionBatch, error) {
	var empty inspectiondom.InspectionBatch

	if u == nil {
		return empty, errors.New("mint usecase is nil")
	}
	if u.inspRepo == nil {
		return empty, errors.New("inspection repo is nil")
	}
	if u.mintRepo == nil {
		return empty, errors.New("mint repo is nil")
	}
	if u.passedProductLister == nil {
		return empty, errors.New("passedProductLister is nil")
	}
	if u.tbRepo == nil {
		return empty, errors.New("tokenBlueprint repo is nil")
	}

	pid := productionID
	if pid == "" {
		return empty, errors.New("productionID is empty")
	}

	tbID := tokenBlueprintID
	if tbID == "" {
		return empty, errors.New("tokenBlueprintID is empty")
	}

	memberID := MemberIDFromContext(ctx)
	if memberID == "" {
		return empty, errors.New("memberID not found in context")
	}

	now := time.Now().UTC()

	tb, err := u.tbRepo.GetByID(ctx, tbID)
	if err != nil {
		return empty, err
	}
	if tb == nil {
		return empty, errors.New("tokenBlueprint not found")
	}

	brandID := tb.BrandID
	if brandID == "" {
		return empty, errors.New("brandID is empty on tokenBlueprint")
	}

	passedProductIDs, err := u.passedProductLister.ListPassedProductIDsByProductionID(ctx, pid)
	if err != nil {
		return empty, err
	}
	if len(passedProductIDs) == 0 {
		return empty, errors.New("no passed products for this production")
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
		return empty, err
	}

	mintEntity.ID = pid
	mintEntity.Minted = false
	mintEntity.MintedAt = nil

	if scheduledBurnDate != nil {
		if s := *scheduledBurnDate; s != "" {
			t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
			if err != nil {
				return empty, errors.New("invalid scheduledBurnDate format (expected YYYY-MM-DD)")
			}
			utc := t.UTC()
			mintEntity.ScheduledBurnDate = &utc
		}
	}

	if _, err := u.mintRepo.Create(ctx, mintEntity); err != nil {
		return empty, err
	}

	batch, err := u.inspRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return empty, err
	}

	if u.tokenMinter == nil {
		log.Printf(
			"[mint][UpdateRequestInfo] tokenMinter is nil; mint record created but onchain mint was skipped productionID=%s tokenBlueprintID=%s",
			pid,
			tbID,
		)

		return batch, errors.New("token minter is not configured")
	}

	result, err := u.MintFromMintRequest(ctx, pid)
	if err != nil {
		log.Printf(
			"[mint][UpdateRequestInfo] onchain mint failed after mint record created productionID=%s tokenBlueprintID=%s err=%v",
			pid,
			tbID,
			err,
		)

		return batch, fmt.Errorf("onchain mint failed after mint request was created: %w", err)
	}

	if result == nil {
		log.Printf(
			"[mint][UpdateRequestInfo] onchain mint returned nil result productionID=%s tokenBlueprintID=%s",
			pid,
			tbID,
		)

		return batch, errors.New("onchain mint returned nil result")
	}

	log.Printf(
		"[mint][UpdateRequestInfo] onchain mint succeeded productionID=%s tokenBlueprintID=%s signature=%s mintAddress=%s",
		pid,
		tbID,
		result.Signature,
		result.MintAddress,
	)

	return batch, nil
}

func (u *MintUsecase) ListMintsByProductionIDs(
	ctx context.Context,
	productionIDs []string,
) (map[string]mintdom.Mint, error) {
	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if u.mintRepo == nil {
		return nil, errors.New("mint repo is nil")
	}

	seen := make(map[string]struct{}, len(productionIDs))
	ids := make([]string, 0, len(productionIDs))

	for _, id := range productionIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return map[string]mintdom.Mint{}, nil
	}

	sort.Strings(ids)

	return u.mintRepo.ListByProductionID(ctx, ids)
}

func (u *MintUsecase) GetProductBlueprintPatchByID(
	ctx context.Context,
	productBlueprintID string,
) (pbpdom.Patch, error) {
	if u == nil {
		return pbpdom.Patch{}, errors.New("mint usecase is nil")
	}
	if u.pbRepo == nil {
		return pbpdom.Patch{}, errors.New("productBlueprint repo is nil")
	}

	if productBlueprintID == "" {
		return pbpdom.Patch{}, errors.New("productBlueprintID is empty")
	}

	return u.pbRepo.GetPatchByID(ctx, productBlueprintID)
}

func (u *MintUsecase) ListBrandsForCurrentCompany(
	ctx context.Context,
	page branddom.Page,
) (branddom.PageResult[branddom.Brand], error) {
	var empty branddom.PageResult[branddom.Brand]

	if u == nil {
		return empty, errors.New("mint usecase is nil")
	}
	if u.brandSvc == nil {
		return empty, errors.New("brand service is nil")
	}

	companyID := CompanyIDFromContext(ctx)
	if companyID == "" {
		return empty, ErrCompanyIDMissing
	}

	return u.brandSvc.ListByCompanyID(ctx, companyID, page)
}

func (u *MintUsecase) ListTokenBlueprintsByBrand(
	ctx context.Context,
	brandID string,
	page domcommon.Page,
) (domcommon.PageResult[tbdom.TokenBlueprint], error) {
	var empty domcommon.PageResult[tbdom.TokenBlueprint]

	if u == nil {
		return empty, errors.New("mint usecase is nil")
	}
	if u.tbRepo == nil {
		return empty, errors.New("tokenBlueprint repo is nil")
	}

	if brandID == "" {
		return empty, errors.New("brandID is empty")
	}

	return tbdom.ListByBrandID(ctx, u.tbRepo, brandID, page)
}

func (u *MintUsecase) ListInspectionBatchesByProductionIDs(
	ctx context.Context,
	productionIDs []string,
) ([]inspectiondom.InspectionBatch, error) {
	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if u.inspRepo == nil {
		return nil, errors.New("inspection repo is nil")
	}

	seen := make(map[string]struct{}, len(productionIDs))
	ids := make([]string, 0, len(productionIDs))

	for _, id := range productionIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return []inspectiondom.InspectionBatch{}, nil
	}

	sort.Strings(ids)

	return u.inspRepo.ListByProductionID(ctx, ids)
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

func (u *MintUsecase) loadInspectionBatchByProductionID(
	ctx context.Context,
	productionID string,
) (*inspectiondom.InspectionBatch, error) {
	if u == nil || u.inspRepo == nil {
		return nil, errors.New("inspection repo is nil")
	}

	if productionID == "" {
		return nil, inspectiondom.ErrInvalidInspectionProductionID
	}

	batch, err := u.inspRepo.GetByProductionID(ctx, productionID)
	if err != nil {
		return nil, err
	}

	return &batch, nil
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

func (u *MintUsecase) MintFromMintRequest(ctx context.Context, mintRequestID string) (*tokendom.MintResult, error) {
	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if mintRequestID == "" {
		return nil, errors.New("mintRequestID is empty")
	}
	if u.tokenMinter == nil {
		return nil, errors.New("token minter is nil")
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

	passedProductIDs := mintEnt.Products
	if err := validateProductIDs(passedProductIDs); err != nil {
		return nil, err
	}

	actorID := mintEnt.CreatedBy
	if actorID == "" {
		actorID = MemberIDFromContext(ctx)
	}
	if actorID == "" {
		return nil, errors.New("actorID is missing (mint.createdBy and context memberId are empty)")
	}

	tbID := mintEnt.TokenBlueprintID
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
		if u.tbBucketEnsurer != nil {
			if err := u.tbBucketEnsurer.EnsureKeepObjects(ctx, tbID); err != nil {
				return nil, err
			}
		}

		if u.tbMetadataEnsurer == nil {
			return nil, fmt.Errorf("tokenBlueprint metadata ensurer is nil")
		}
		if u.tbRepo == nil {
			return nil, fmt.Errorf("tokenBlueprint repo is nil")
		}

		tb, err := u.tbRepo.GetByID(ctx, tbID)
		if err != nil {
			return nil, err
		}
		if tb == nil {
			return nil, fmt.Errorf("tokenBlueprint not found (id=%s)", tbID)
		}

		updated, err := u.tbMetadataEnsurer.EnsureMetadataURI(ctx, tb, actorID)
		if err != nil {
			return nil, err
		}
		if updated == nil {
			updated = tb
		}

		uri := strings.TrimSpace(updated.MetadataURI)
		if uri == "" {
			return nil, fmt.Errorf("metadataUri is empty after ensure (tokenBlueprintId=%s)", tbID)
		}

		result, err = u.tokenMinter.MintFromMintRequest(ctx, mintRequestID)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, fmt.Errorf("onchain mint succeeded but result is nil (mintRequestId=%s)", mintRequestID)
		}

		if u.tbMintMarker != nil {
			_, _ = u.tbMintMarker.MarkTokenBlueprintMinted(ctx, tbID, actorID)
		}

		if u.mintRepo != nil {
			if updater, ok := any(u.mintRepo).(interface {
				Update(ctx context.Context, m mintdom.Mint) (mintdom.Mint, error)
			}); ok {
				now := time.Now().UTC()
				m := *mintEnt

				m.ID = mintRequestID
				m.Minted = true
				m.MintedAt = &now

				_ = u.mintResultMapper.ApplyOnchainResult(&m, result)

				_, _ = updater.Update(ctx, m)
			}
		}
	}

	if u.inventoryUC == nil {
		return nil, errors.New("inventory usecase is nil (cannot upsert inventory)")
	}

	batch, berr := u.loadInspectionBatchByProductionID(ctx, mintRequestID)
	if berr != nil || batch == nil {
		if berr != nil {
			return nil, berr
		}
		return nil, errors.New("inspection batch is nil")
	}

	passedSet := make(map[string]struct{}, len(passedProductIDs))
	for _, p := range passedProductIDs {
		passedSet[p] = struct{}{}
	}

	byModel := map[string][]string{}
	for _, it := range batch.Inspections {
		pid := it.ProductID
		if pid == "" {
			return nil, mintdom.ErrInvalidProducts
		}
		if _, ok := passedSet[pid]; !ok {
			continue
		}

		mid := it.ModelID
		if mid == "" {
			continue
		}

		byModel[mid] = append(byModel[mid], pid)
	}

	modelIDs := make([]string, 0, len(byModel))
	for mid := range byModel {
		modelIDs = append(modelIDs, mid)
	}
	sort.Strings(modelIDs)

	if len(modelIDs) == 0 {
		return nil, errors.New("no model groups found from inspection batch for passed products")
	}

	for _, mid := range modelIDs {
		pids := byModel[mid]
		if err := validateProductIDs(pids); err != nil {
			return nil, err
		}
		if len(pids) == 0 {
			continue
		}

		invEnt, invErr := u.inventoryUC.UpsertFromMintByModel(ctx, tbID, pbID, mid, pids)
		if invErr != nil {
			return nil, invErr
		}

		var _ invdom.Mint = invEnt
	}

	return result, nil
}
