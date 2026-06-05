// backend/internal/application/usecase/print_usecase.go
package usecase

import (
	"context"
	"fmt"
	"sort"
	"time"

	inspectiondom "narratives/internal/domain/inspection"
	printdom "narratives/internal/domain/print"
	productdom "narratives/internal/domain/product"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	productiondom "narratives/internal/domain/production"
)

const publicQRBaseURL = "https://amol.jp"

type ProductRepo interface {
	Create(ctx context.Context, p productdom.Product) (productdom.Product, error)
	ListByProductionID(ctx context.Context, productionID string) ([]productdom.Product, error)
}

type PrintLogRepo interface {
	Create(ctx context.Context, log printdom.PrintLog) (printdom.PrintLog, error)
	GetByProductionID(ctx context.Context, productionID string) (printdom.PrintLog, error)
	ExistsByProductionID(ctx context.Context, productionID string) (bool, error)
}

type InspectionRepo interface {
	Create(ctx context.Context, batch inspectiondom.InspectionBatch) (inspectiondom.InspectionBatch, error)
}

type PrintProductionRepo interface {
	GetByID(ctx context.Context, id string) (*productiondom.Production, error)
	Update(ctx context.Context, production productiondom.Production) (*productiondom.Production, error)
}

type PrintUsecase struct {
	productionRepo       PrintProductionRepo
	repo                 ProductRepo
	printLogRepo         PrintLogRepo
	inspectionRepo       InspectionRepo
	productBlueprintRepo productblueprintdom.Repository
	now                  func() time.Time
}

func NewPrintUsecase(
	productionRepo PrintProductionRepo,
	repo ProductRepo,
	printLogRepo PrintLogRepo,
	inspectionRepo InspectionRepo,
	productBlueprintRepo productblueprintdom.Repository,
) *PrintUsecase {
	return &PrintUsecase{
		productionRepo:       productionRepo,
		repo:                 repo,
		printLogRepo:         printLogRepo,
		inspectionRepo:       inspectionRepo,
		productBlueprintRepo: productBlueprintRepo,
		now:                  time.Now,
	}
}

func (u *PrintUsecase) CreatePrintLogForProduction(ctx context.Context, productionID string) (printdom.PrintLog, error) {
	if u.productionRepo == nil {
		return printdom.PrintLog{}, fmt.Errorf("productionRepo is nil")
	}
	if u.repo == nil {
		return printdom.PrintLog{}, fmt.Errorf("productRepo is nil")
	}
	if u.printLogRepo == nil {
		return printdom.PrintLog{}, fmt.Errorf("printLogRepo is nil")
	}
	if u.inspectionRepo == nil {
		return printdom.PrintLog{}, fmt.Errorf("inspectionRepo is nil")
	}
	if u.productBlueprintRepo == nil {
		return printdom.PrintLog{}, fmt.Errorf("productBlueprintRepo is nil")
	}

	pid := productionID
	if pid == "" {
		return printdom.PrintLog{}, printdom.ErrInvalidPrintLogProductionID
	}

	production, err := u.productionRepo.GetByID(ctx, pid)
	if err != nil {
		return printdom.PrintLog{}, err
	}
	if production == nil {
		return printdom.PrintLog{}, productiondom.ErrNotFound
	}

	printedAt := u.now().UTC()

	exists, err := u.printLogRepo.ExistsByProductionID(ctx, pid)
	if err != nil {
		return printdom.PrintLog{}, err
	}
	if exists {
		existing, err := u.printLogRepo.GetByProductionID(ctx, pid)
		if err != nil {
			return printdom.PrintLog{}, err
		}

		if err := u.markProductionPrinted(ctx, *production, printedAt); err != nil {
			return printdom.PrintLog{}, err
		}

		existing.QrPayloads = buildQrPayloads(existing.Items)
		return existing, nil
	}

	products, err := u.repo.ListByProductionID(ctx, pid)
	if err != nil {
		return printdom.PrintLog{}, err
	}

	if len(products) == 0 {
		if err := u.createProductsForProduction(ctx, *production, printedAt); err != nil {
			return printdom.PrintLog{}, err
		}

		products, err = u.repo.ListByProductionID(ctx, pid)
		if err != nil {
			return printdom.PrintLog{}, err
		}
	}

	if len(products) == 0 {
		return printdom.PrintLog{}, fmt.Errorf("no products found for productionId=%s", pid)
	}

	productIDs := make([]string, 0, len(products))
	modelIDByProductID := make(map[string]string, len(products))
	displayOrderByModelID := make(map[string]int, len(products))

	for _, p := range products {
		productID := p.ID
		if productID == "" {
			continue
		}

		modelID := p.ModelID
		if modelID == "" {
			return printdom.PrintLog{}, fmt.Errorf("modelId is empty for productId=%s", productID)
		}

		productIDs = append(productIDs, productID)
		modelIDByProductID[productID] = modelID

		if _, exists := displayOrderByModelID[modelID]; exists {
			continue
		}

		productBlueprintID, modelRefs, err := u.productBlueprintRepo.GetIDByModelID(ctx, modelID)
		if err != nil {
			return printdom.PrintLog{}, fmt.Errorf("get productBlueprint by modelId failed: modelId=%s: %w", modelID, err)
		}
		if productBlueprintID == "" {
			return printdom.PrintLog{}, fmt.Errorf("productBlueprintId not found for modelId=%s", modelID)
		}
		if len(modelRefs) == 0 {
			return printdom.PrintLog{}, fmt.Errorf("modelRefs not found for modelId=%s", modelID)
		}

		found := false
		for _, ref := range modelRefs {
			if ref.ModelID != modelID {
				continue
			}
			if ref.DisplayOrder <= 0 {
				return printdom.PrintLog{}, fmt.Errorf("invalid displayOrder for modelId=%s", modelID)
			}

			displayOrderByModelID[modelID] = ref.DisplayOrder
			found = true
			break
		}
		if !found {
			return printdom.PrintLog{}, fmt.Errorf("displayOrder not found in modelRefs for modelId=%s", modelID)
		}
	}

	if len(productIDs) == 0 {
		return printdom.PrintLog{}, inspectiondom.ErrInvalidInspectionProductIDs
	}

	items := make([]printdom.PrintedItem, 0, len(productIDs))
	for _, productID := range productIDs {
		modelID := modelIDByProductID[productID]
		if modelID == "" {
			return printdom.PrintLog{}, fmt.Errorf("modelId is empty for productId=%s", productID)
		}

		displayOrder, ok := displayOrderByModelID[modelID]
		if !ok {
			return printdom.PrintLog{}, fmt.Errorf("displayOrder not resolved for modelId=%s", modelID)
		}

		items = append(items, printdom.PrintedItem{
			ProductID:    productID,
			DisplayOrder: displayOrder,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].DisplayOrder < items[j].DisplayOrder
	})

	logID := pid
	log, err := printdom.NewPrintLog(
		logID,
		pid,
		items,
	)
	if err != nil {
		return printdom.PrintLog{}, err
	}

	batch, err := inspectiondom.NewInspectionBatch(
		pid,
		inspectiondom.InspectionStatusInspecting,
		productIDs,
	)
	if err != nil {
		return printdom.PrintLog{}, err
	}

	for i := range batch.Inspections {
		productID := batch.Inspections[i].ProductID
		if mid, ok := modelIDByProductID[productID]; ok {
			batch.Inspections[i].ModelID = mid
		}
	}

	if _, err := u.inspectionRepo.Create(ctx, batch); err != nil {
		return printdom.PrintLog{}, err
	}

	created, err := u.printLogRepo.Create(ctx, log)
	if err != nil {
		return printdom.PrintLog{}, err
	}

	if err := u.markProductionPrinted(ctx, *production, printedAt); err != nil {
		return printdom.PrintLog{}, err
	}

	created.QrPayloads = buildQrPayloads(created.Items)

	return created, nil
}

func (u *PrintUsecase) createProductsForProduction(
	ctx context.Context,
	production productiondom.Production,
	printedAt time.Time,
) error {
	if production.ID == "" {
		return productiondom.ErrInvalidID
	}

	if len(production.Models) == 0 {
		return fmt.Errorf("production models is empty: productionId=%s", production.ID)
	}

	for _, model := range production.Models {
		if model.ModelID == "" {
			return fmt.Errorf("modelId is empty: productionId=%s", production.ID)
		}

		if model.Quantity <= 0 {
			return fmt.Errorf(
				"invalid quantity: productionId=%s modelId=%s quantity=%d",
				production.ID,
				model.ModelID,
				model.Quantity,
			)
		}

		for i := 0; i < model.Quantity; i++ {
			product := productdom.Product{
				ModelID:          model.ModelID,
				ProductionID:     production.ID,
				InspectionResult: productdom.InspectionNotYet,
				PrintedAt:        &printedAt,
				InspectedAt:      nil,
				InspectedBy:      nil,
			}

			if _, err := u.repo.Create(ctx, product); err != nil {
				return fmt.Errorf(
					"create product failed: productionId=%s modelId=%s: %w",
					production.ID,
					model.ModelID,
					err,
				)
			}
		}
	}

	return nil
}

func (u *PrintUsecase) markProductionPrinted(
	ctx context.Context,
	production productiondom.Production,
	printedAt time.Time,
) error {
	printed := true

	if err := production.ApplyUpdate(
		production.AssigneeID,
		production.Models,
		&printed,
		&printedAt,
		nil,
		nil,
		u.now().UTC(),
	); err != nil {
		return err
	}

	updated, err := u.productionRepo.Update(ctx, production)
	if err != nil {
		return err
	}
	if updated == nil {
		return productiondom.ErrNotFound
	}

	return updated.Validate()
}

func (u *PrintUsecase) Create(ctx context.Context, p productdom.Product) (productdom.Product, error) {
	created, err := u.repo.Create(ctx, p)
	if err != nil {
		return productdom.Product{}, err
	}

	return created, nil
}

func buildQrPayloads(items []printdom.PrintedItem) []string {
	payloads := make([]string, 0, len(items))

	for _, it := range items {
		if it.ProductID == "" {
			continue
		}

		payloads = append(payloads, fmt.Sprintf("%s/%s", publicQRBaseURL, it.ProductID))
	}

	return payloads
}
