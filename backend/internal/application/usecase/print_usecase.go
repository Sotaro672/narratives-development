// backend/internal/application/usecase/print_usecase.go
package usecase

import (
	"context"
	"fmt"
	"sort"
	"time"

	inspectiondom "narratives/internal/domain/inspection"
	productdom "narratives/internal/domain/product"
	productblueprintdom "narratives/internal/domain/productBlueprint"
)

const publicQRBaseURL = "https://amol.jp"

type ProductRepo interface {
	GetByID(ctx context.Context, id string) (productdom.Product, error)
	Create(ctx context.Context, p productdom.Product) (productdom.Product, error)
	Update(ctx context.Context, id string, p productdom.Product) (productdom.Product, error)
	ListByProductionID(ctx context.Context, productionID string) ([]productdom.Product, error)
}

type PrintLogRepo interface {
	Create(ctx context.Context, log productdom.PrintLog) (productdom.PrintLog, error)
}

type InspectionRepo interface {
	Create(ctx context.Context, batch inspectiondom.InspectionBatch) (inspectiondom.InspectionBatch, error)
}

type PrintUsecase struct {
	repo                 ProductRepo
	printLogRepo         PrintLogRepo
	inspectionRepo       InspectionRepo
	productBlueprintRepo productblueprintdom.Repository
}

func NewPrintUsecase(
	repo ProductRepo,
	printLogRepo PrintLogRepo,
	inspectionRepo InspectionRepo,
	productBlueprintRepo productblueprintdom.Repository,
) *PrintUsecase {
	return &PrintUsecase{
		repo:                 repo,
		printLogRepo:         printLogRepo,
		inspectionRepo:       inspectionRepo,
		productBlueprintRepo: productBlueprintRepo,
	}
}

func (u *PrintUsecase) CreatePrintLogForProduction(ctx context.Context, productionID string) (productdom.PrintLog, error) {
	if u.printLogRepo == nil {
		return productdom.PrintLog{}, fmt.Errorf("printLogRepo is nil")
	}
	if u.inspectionRepo == nil {
		return productdom.PrintLog{}, fmt.Errorf("inspectionRepo is nil")
	}
	if u.productBlueprintRepo == nil {
		return productdom.PrintLog{}, fmt.Errorf("productBlueprintRepo is nil")
	}

	pid := productionID
	if pid == "" {
		return productdom.PrintLog{}, productdom.ErrInvalidPrintLogProductionID
	}

	products, err := u.repo.ListByProductionID(ctx, pid)
	if err != nil {
		return productdom.PrintLog{}, err
	}
	if len(products) == 0 {
		return productdom.PrintLog{}, fmt.Errorf("no products found for productionId=%s", pid)
	}

	productIDs := make([]string, 0, len(products))
	modelIDByProductID := make(map[string]string, len(products))
	productBlueprintIDSet := make(map[string]struct{})

	for _, p := range products {
		id := p.ID
		if id == "" {
			continue
		}

		productIDs = append(productIDs, id)
		modelIDByProductID[id] = p.ModelID

		if p.ModelID == "" {
			return productdom.PrintLog{}, fmt.Errorf("modelId is empty for productId=%s", p.ID)
		}

		productBlueprintID, err := u.productBlueprintRepo.GetIDByModelID(ctx, p.ModelID)
		if err != nil {
			return productdom.PrintLog{}, fmt.Errorf("get productBlueprintId by modelId failed: modelId=%s: %w", p.ModelID, err)
		}
		if productBlueprintID == "" {
			return productdom.PrintLog{}, fmt.Errorf("productBlueprintId not found for modelId=%s", p.ModelID)
		}

		productBlueprintIDSet[productBlueprintID] = struct{}{}
	}

	if len(productIDs) == 0 {
		return productdom.PrintLog{}, inspectiondom.ErrInvalidInspectionProductIDs
	}

	var printedAt time.Time
	for _, p := range products {
		if p.PrintedAt != nil && !p.PrintedAt.IsZero() {
			printedAt = p.PrintedAt.UTC()
			break
		}
	}
	if printedAt.IsZero() {
		printedAt = time.Now().UTC()
	}

	displayOrderByModelID := make(map[string]int, len(products))
	for _, p := range products {
		modelID := p.ModelID
		if modelID == "" {
			return productdom.PrintLog{}, fmt.Errorf("modelId is empty for productId=%s", p.ID)
		}
		if _, exists := displayOrderByModelID[modelID]; exists {
			continue
		}

		modelRefs, err := u.productBlueprintRepo.GetModelRefsByModelID(ctx, modelID)
		if err != nil {
			return productdom.PrintLog{}, fmt.Errorf("get modelRefs by modelId failed: modelId=%s: %w", modelID, err)
		}
		if len(modelRefs) == 0 {
			return productdom.PrintLog{}, fmt.Errorf("modelRefs not found for modelId=%s", modelID)
		}

		found := false
		for _, ref := range modelRefs {
			if ref.ModelID != modelID {
				continue
			}
			if ref.DisplayOrder <= 0 {
				return productdom.PrintLog{}, fmt.Errorf("invalid displayOrder for modelId=%s", modelID)
			}

			displayOrderByModelID[modelID] = ref.DisplayOrder
			found = true
			break
		}
		if !found {
			return productdom.PrintLog{}, fmt.Errorf("displayOrder not found in modelRefs for modelId=%s", modelID)
		}
	}

	items := make([]productdom.PrintedItem, 0, len(productIDs))
	for _, productID := range productIDs {
		modelID := modelIDByProductID[productID]
		if modelID == "" {
			return productdom.PrintLog{}, fmt.Errorf("modelId is empty for productId=%s", productID)
		}

		displayOrder, ok := displayOrderByModelID[modelID]
		if !ok {
			return productdom.PrintLog{}, fmt.Errorf("displayOrder not resolved for modelId=%s", modelID)
		}

		items = append(items, productdom.PrintedItem{
			ProductID:    productID,
			DisplayOrder: displayOrder,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].DisplayOrder < items[j].DisplayOrder
	})

	logID := fmt.Sprintf("%s-%d", pid, printedAt.UnixNano())
	log, err := productdom.NewPrintLog(
		logID,
		pid,
		items,
	)
	if err != nil {
		return productdom.PrintLog{}, err
	}

	batch, err := inspectiondom.NewInspectionBatch(
		pid,
		inspectiondom.InspectionStatusInspecting,
		productIDs,
	)
	if err != nil {
		return productdom.PrintLog{}, err
	}

	for i := range batch.Inspections {
		productID := batch.Inspections[i].ProductID
		if mid, ok := modelIDByProductID[productID]; ok {
			batch.Inspections[i].ModelID = mid
		}
	}

	if _, err := u.inspectionRepo.Create(ctx, batch); err != nil {
		return productdom.PrintLog{}, err
	}

	created, err := u.printLogRepo.Create(ctx, log)
	if err != nil {
		return productdom.PrintLog{}, err
	}

	for productBlueprintID := range productBlueprintIDSet {
		if _, err := u.productBlueprintRepo.MarkPrinted(ctx, productBlueprintID); err != nil {
			return productdom.PrintLog{}, fmt.Errorf("mark productBlueprint printed failed: productBlueprintId=%s: %w", productBlueprintID, err)
		}
	}

	var payloads []string
	for _, it := range created.Items {
		if it.ProductID == "" {
			continue
		}
		payloads = append(payloads, fmt.Sprintf("%s/%s", publicQRBaseURL, it.ProductID))
	}
	created.QrPayloads = payloads

	return created, nil
}

func (u *PrintUsecase) Create(ctx context.Context, p productdom.Product) (productdom.Product, error) {
	created, err := u.repo.Create(ctx, p)
	if err != nil {
		return productdom.Product{}, err
	}
	return created, nil
}
