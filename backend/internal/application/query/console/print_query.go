// backend/internal/application/query/console/print_query.go
package query

import (
	"context"
	"fmt"
	"strings"

	printdom "narratives/internal/domain/print"
	productdom "narratives/internal/domain/product"
)

const publicQRBaseURL = "https://amol.jp"

// ProductPrintQueryRepo は print 画面構築用に product 一覧を取得する最小ポートです。
type ProductPrintQueryRepo interface {
	ListByProductionID(ctx context.Context, productionID string) ([]productdom.Product, error)
}

type PrintLogPrintQueryRepo interface {
	GetByProductionID(ctx context.Context, productionID string) (printdom.PrintLog, error)
}

// ModelNumberResolver は modelId から modelNumber を解決する最小ポートです。
type ModelNumberResolver interface {
	ResolveModelNumber(ctx context.Context, variationID string) string
}

type PrintQueryService struct {
	productRepo         ProductPrintQueryRepo
	printLogRepo        PrintLogPrintQueryRepo
	modelNumberResolver ModelNumberResolver
}

func NewPrintQueryService(
	productRepo ProductPrintQueryRepo,
	printLogRepo PrintLogPrintQueryRepo,
	modelNumberResolver ModelNumberResolver,
) *PrintQueryService {
	return &PrintQueryService{
		productRepo:         productRepo,
		printLogRepo:        printLogRepo,
		modelNumberResolver: modelNumberResolver,
	}
}

type ProductSummaryForPrintDTO struct {
	ID           string `json:"id"`
	ModelID      string `json:"modelId"`
	ProductionID string `json:"productionId"`
	ModelNumber  string `json:"modelNumber"`
}

type PrintedItemForPrintDTO struct {
	ProductID    string `json:"productId"`
	DisplayOrder int    `json:"displayOrder"`
	QRPayload    string `json:"qrPayload"`
	ModelNumber  string `json:"modelNumber"`
}

type PrintLogForPrintDTO struct {
	ID           string                   `json:"id"`
	ProductionID string                   `json:"productionId"`
	Items        []PrintedItemForPrintDTO `json:"items"`
}

func (q *PrintQueryService) ListProductsByProductionID(
	ctx context.Context,
	productionID string,
) ([]ProductSummaryForPrintDTO, error) {
	if q == nil || q.productRepo == nil {
		return nil, fmt.Errorf("print product query repo is nil")
	}

	if q.modelNumberResolver == nil {
		return nil, fmt.Errorf("model number resolver is nil")
	}

	pid := strings.Trim(productionID, " \t\r\n/")
	if pid == "" {
		return nil, printdom.ErrInvalidPrintLogProductionID
	}

	products, err := q.productRepo.ListByProductionID(ctx, pid)
	if err != nil {
		return nil, err
	}

	out := make([]ProductSummaryForPrintDTO, 0, len(products))
	for _, product := range products {
		modelNumber := q.modelNumberResolver.ResolveModelNumber(ctx, product.ModelID)

		out = append(out, ProductSummaryForPrintDTO{
			ID:           product.ID,
			ModelID:      product.ModelID,
			ProductionID: product.ProductionID,
			ModelNumber:  modelNumber,
		})
	}

	return out, nil
}

func (q *PrintQueryService) ListPrintLogsByProductionID(
	ctx context.Context,
	productionID string,
) ([]PrintLogForPrintDTO, error) {
	if q == nil || q.printLogRepo == nil {
		return nil, fmt.Errorf("print log query repo is nil")
	}
	if q.productRepo == nil {
		return nil, fmt.Errorf("print product query repo is nil")
	}
	if q.modelNumberResolver == nil {
		return nil, fmt.Errorf("model number resolver is nil")
	}

	pid := strings.Trim(productionID, " \t\r\n/")
	if pid == "" {
		return nil, printdom.ErrInvalidPrintLogProductionID
	}

	log, err := q.printLogRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return nil, err
	}

	products, err := q.productRepo.ListByProductionID(ctx, pid)
	if err != nil {
		return nil, err
	}

	modelNumberByProductID := make(map[string]string, len(products))
	for _, product := range products {
		modelNumberByProductID[product.ID] =
			q.modelNumberResolver.ResolveModelNumber(ctx, product.ModelID)
	}

	return []PrintLogForPrintDTO{
		buildPrintLogForPrintDTO(log, modelNumberByProductID),
	}, nil
}

func buildPrintLogForPrintDTO(
	log printdom.PrintLog,
	modelNumberByProductID map[string]string,
) PrintLogForPrintDTO {
	items := make([]PrintedItemForPrintDTO, 0, len(log.Items))

	for _, item := range log.Items {
		if item.ProductID == "" {
			continue
		}

		items = append(items, PrintedItemForPrintDTO{
			ProductID:    item.ProductID,
			DisplayOrder: item.DisplayOrder,
			QRPayload:    fmt.Sprintf("%s/%s", publicQRBaseURL, item.ProductID),
			ModelNumber:  modelNumberByProductID[item.ProductID],
		})
	}

	return PrintLogForPrintDTO{
		ID:           log.ID,
		ProductionID: log.ProductionID,
		Items:        items,
	}
}
