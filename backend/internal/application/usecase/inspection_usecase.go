package usecase

import (
	"context"
	"fmt"
	"time"

	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
	modeldom "narratives/internal/domain/model"
)

// ------------------------------------------------------------
// Ports
// ------------------------------------------------------------
//
// inspections 永続化ポートは domain 側（inspection.Repository）へ移譲しています。
// ここでは inspection 以外の境界（products / mints / models）に関する最小ポートのみ定義します。

// ProductInspectionRepo は products テーブル側の inspectionResult を更新するための
// 最小限のポートです。
type ProductInspectionRepo interface {
	UpdateInspectionResult(
		ctx context.Context,
		productID string,
		result inspectiondom.InspectionResult,
	) error
}

// InspectionMintGetter は productionId / mintId 共通IDから mint を 1 件取得するための最小ポートです。
//
// AMOL/Narratives では production / inspection / mint の docId は同一値として扱うため、
// GetByInspectionID ではなく GetByID に統一します。
type InspectionMintGetter interface {
	GetByID(ctx context.Context, id string) (mintdom.Mint, error)
}

// ------------------------------------------------------------
// Usecase
// ------------------------------------------------------------

type InspectionUsecase struct {
	inspectionRepo inspectiondom.Repository
	productRepo    ProductInspectionRepo
	mintRepo       InspectionMintGetter // nil 許容
	modelRepo      ModelVariationGetter // nil 許容
}

// NewInspectionUsecase を唯一の出入り口にするため、必要な依存はすべてここで受け取る。
// mintRepo / modelRepo は不要なら nil を渡せる。
func NewInspectionUsecase(
	inspectionRepo inspectiondom.Repository,
	productRepo ProductInspectionRepo,
	mintRepo InspectionMintGetter,
	modelRepo ModelVariationGetter,
) *InspectionUsecase {
	return &InspectionUsecase{
		inspectionRepo: inspectionRepo,
		productRepo:    productRepo,
		mintRepo:       mintRepo,
		modelRepo:      modelRepo,
	}
}

// ------------------------------------------------------------
// Commands
// ------------------------------------------------------------

// CompleteInspectionForProduction は検品を完了します。
//
// ネガティブ制では、failed / notManufactured として明示的に入力されなかった
// notYet の productId を Complete 時に passed として確定します。
func (u *InspectionUsecase) CompleteInspectionForProduction(
	ctx context.Context,
	productionID string,
	by string,
	at time.Time,
) (inspectiondom.InspectionBatch, error) {
	if u.inspectionRepo == nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := productionID
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	if err := batch.Complete(by, at); err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	if u.productRepo != nil {
		for _, item := range updated.Inspections {
			if item.InspectionResult == nil {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
			}

			result := *item.InspectionResult
			if !inspectiondom.IsValidInspectionResult(result) {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
			}

			pdID := item.ProductID
			if pdID == "" {
				return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductIDs
			}

			if err := u.productRepo.UpdateInspectionResult(ctx, pdID, result); err != nil {
				return inspectiondom.InspectionBatch{}, err
			}
		}
	}

	return updated, nil
}

// UpdateInspectionForProduct は inspections 内の 1 productId 分を更新します。
//
// ネガティブ制では、通常は failed / notManufactured を明示的に入力します。
// ただし、誤って failed / notManufactured にした productId を戻すため、
// 修正操作として passed への更新も許可します。
func (u *InspectionUsecase) UpdateInspectionForProduct(
	ctx context.Context,
	productionID string,
	productID string,
	result *inspectiondom.InspectionResult,
	inspectedBy *string,
	inspectedAt *time.Time,
) (inspectiondom.InspectionBatch, error) {
	if u.inspectionRepo == nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := productionID
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	pdID := productID
	if pdID == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductIDs
	}

	if result == nil {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
	}

	if *result != inspectiondom.InspectionPassed &&
		*result != inspectiondom.InspectionFailed &&
		*result != inspectiondom.InspectionNotManufactured {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
	}

	if inspectedBy == nil || *inspectedBy == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedBy
	}

	if inspectedAt == nil {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedAt
	}

	atUTC := inspectedAt.UTC()
	if atUTC.IsZero() {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectedAt
	}

	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	switch *result {
	case inspectiondom.InspectionPassed:
		if err := batch.MarkPassed(pdID, *inspectedBy, atUTC); err != nil {
			return inspectiondom.InspectionBatch{}, err
		}

	case inspectiondom.InspectionFailed:
		if err := batch.MarkFailed(pdID, *inspectedBy, atUTC); err != nil {
			return inspectiondom.InspectionBatch{}, err
		}

	case inspectiondom.InspectionNotManufactured:
		if err := batch.MarkNotManufactured(pdID, *inspectedBy, atUTC); err != nil {
			return inspectiondom.InspectionBatch{}, err
		}

	default:
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionResult
	}

	updated, err := u.inspectionRepo.Save(ctx, batch)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	if u.productRepo != nil {
		if err := u.productRepo.UpdateInspectionResult(ctx, pdID, *result); err != nil {
			return inspectiondom.InspectionBatch{}, err
		}
	}

	return updated, nil
}

// ------------------------------------------------------------
// Queries
// ------------------------------------------------------------

// GetBatchByProductionID は productionId から inspections バッチをそのまま返します。
func (u *InspectionUsecase) GetBatchByProductionID(
	ctx context.Context,
	productionID string,
) (inspectiondom.InspectionBatch, error) {
	if u.inspectionRepo == nil {
		return inspectiondom.InspectionBatch{}, fmt.Errorf("inspectionRepo is nil")
	}

	pid := productionID
	if pid == "" {
		return inspectiondom.InspectionBatch{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	batch, err := u.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return inspectiondom.InspectionBatch{}, err
	}

	return batch, nil
}

// GetMintByInspectionID は inspectionId に紐づく mint を 1 件取得します。
//
// 現在の設計では inspectionId / productionId / mintId は同一値として扱うため、
// 実際の取得は mintRepo.GetByID に統一します。
func (u *InspectionUsecase) GetMintByInspectionID(
	ctx context.Context,
	inspectionID string,
) (mintdom.Mint, error) {
	if u.mintRepo == nil {
		return mintdom.Mint{}, fmt.Errorf("mintRepo is nil")
	}

	iid := inspectionID
	if iid == "" {
		return mintdom.Mint{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	return u.mintRepo.GetByID(ctx, iid)
}

// GetModelVariationByID は ModelVariation を 1 件取得します。
// UI で modelId → size/color/rgb 等の解決に使う想定です。
func (u *InspectionUsecase) GetModelVariationByID(
	ctx context.Context,
	variationID string,
) (*modeldom.ModelVariation, error) {
	if u.modelRepo == nil {
		return nil, fmt.Errorf("modelRepo is nil")
	}

	vid := variationID
	if vid == "" {
		return nil, modeldom.ErrInvalid
	}

	return u.modelRepo.GetModelVariationByID(ctx, vid)
}

// GetBatchForScreenByProductionID は画面用 DTO（InspectionBatch + Mint）を返します。
func (u *InspectionUsecase) GetBatchForScreenByProductionID(
	ctx context.Context,
	productionID string,
) (InspectionBatchForScreenDTO, error) {
	batch, err := u.GetBatchByProductionID(ctx, productionID)
	if err != nil {
		return InspectionBatchForScreenDTO{}, err
	}

	var mintDTO *MintDTO
	if u.mintRepo != nil {
		mintID := batch.ProductionID

		if batch.MintID != nil && *batch.MintID != "" {
			mintID = *batch.MintID
		}

		if mintID != "" {
			m, err := u.mintRepo.GetByID(ctx, mintID)
			if err == nil {
				dto := NewMintDTO(m, batch.ProductionID)
				mintDTO = &dto
			} else if err != mintdom.ErrNotFound {
				return InspectionBatchForScreenDTO{}, err
			}
		}
	}

	return NewInspectionBatchForScreenDTO(batch, mintDTO), nil
}

// ------------------------------------------------------------
// Inspector Screen DTO
// - InspectionBatch + Mint joined
// ------------------------------------------------------------

type InspectionItemDTO struct {
	ProductID        string  `json:"productId"`
	ModelID          string  `json:"modelId"`
	InspectionResult *string `json:"inspectionResult,omitempty"`
	InspectedBy      *string `json:"inspectedBy,omitempty"`
	InspectedAt      *string `json:"inspectedAt,omitempty"` // RFC3339
}

type InspectionBatchForScreenDTO struct {
	ProductionID string              `json:"productionId"`
	MintID       *string             `json:"mintId,omitempty"`
	Quantity     int                 `json:"quantity"`
	Status       string              `json:"status"`
	TotalPassed  int                 `json:"totalPassed"`
	Inspections  []InspectionItemDTO `json:"inspections"`

	// joined
	Mint *MintDTO `json:"mint,omitempty"`
}

func NewInspectionBatchForScreenDTO(
	b inspectiondom.InspectionBatch,
	mint *MintDTO,
) InspectionBatchForScreenDTO {
	items := make([]InspectionItemDTO, 0, len(b.Inspections))

	for _, it := range b.Inspections {
		var res *string
		if it.InspectionResult != nil {
			s := string(*it.InspectionResult)
			res = &s
		}

		var inspectedAt *string
		if it.InspectedAt != nil && !it.InspectedAt.IsZero() {
			s := it.InspectedAt.UTC().Format(time.RFC3339)
			inspectedAt = &s
		}

		items = append(items, InspectionItemDTO{
			ProductID:        it.ProductID,
			ModelID:          it.ModelID,
			InspectionResult: res,
			InspectedBy:      it.InspectedBy,
			InspectedAt:      inspectedAt,
		})
	}

	return InspectionBatchForScreenDTO{
		ProductionID: b.ProductionID,
		MintID:       b.MintID,
		Quantity:     b.Quantity,
		Status:       string(b.Status),
		TotalPassed:  b.TotalPassed,
		Inspections:  items,
		Mint:         mint,
	}
}

// ------------------------------------------------------------
// Mint DTO
// ------------------------------------------------------------

type MintDTO struct {
	MintID            string  `json:"mintId"`
	InspectionID      string  `json:"inspectionId"`
	BrandID           string  `json:"brandId"`
	TokenBlueprintID  string  `json:"tokenBlueprintId"`
	CreatedAt         string  `json:"createdAt"` // RFC3339
	CreatedBy         string  `json:"createdBy"`
	Minted            bool    `json:"minted"`
	MintedAt          *string `json:"mintedAt,omitempty"`          // RFC3339
	ScheduledBurnDate *string `json:"scheduledBurnDate,omitempty"` // RFC3339
}

// NewMintDTO は inspectionID に batch.ProductionID (= inspectionId 扱い) を渡す想定です。
func NewMintDTO(m mintdom.Mint, inspectionID string) MintDTO {
	createdAt := ""
	if !m.CreatedAt.IsZero() {
		createdAt = m.CreatedAt.UTC().Format(time.RFC3339)
	}

	var mintedAt *string
	if m.MintedAt != nil && !m.MintedAt.IsZero() {
		s := m.MintedAt.UTC().Format(time.RFC3339)
		mintedAt = &s
	}

	var scheduled *string
	if m.ScheduledBurnDate != nil && !m.ScheduledBurnDate.IsZero() {
		s := m.ScheduledBurnDate.UTC().Format(time.RFC3339)
		scheduled = &s
	}

	return MintDTO{
		MintID:            m.ID,
		InspectionID:      inspectionID,
		BrandID:           m.BrandID,
		TokenBlueprintID:  m.TokenBlueprintID,
		CreatedAt:         createdAt,
		CreatedBy:         m.CreatedBy,
		Minted:            m.Minted,
		MintedAt:          mintedAt,
		ScheduledBurnDate: scheduled,
	}
}
