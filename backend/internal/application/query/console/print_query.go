// backend/internal/application/query/console/print_query.go
package query

import (
	"context"
	"fmt"
	"strings"

	productdom "narratives/internal/domain/product"
)

const publicQRBaseURL = "https://amol.jp"

// ProductPrintQueryRepo は print 画面構築用に product 一覧を取得する最小ポートです。
type ProductPrintQueryRepo interface {
	ListByProductionID(ctx context.Context, productionID string) ([]productdom.Product, error)
}

// PrintLogPrintQueryRepo は print 画面構築用に print_log 一覧を取得する最小ポートです。
type PrintLogPrintQueryRepo interface {
	ListByProductionID(ctx context.Context, productionID string) ([]productdom.PrintLog, error)
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
}

type PrintLogForPrintDTO struct {
	ID           string                   `json:"id"`
	ProductionID string                   `json:"productionId"`
	Items        []PrintedItemForPrintDTO `json:"items"`
	QrPayloads   []string                 `json:"qrPayloads"`
}

func (q *PrintQueryService) ListProductsByProductionID(
	ctx context.Context,
	productionID string,
) ([]ProductSummaryForPrintDTO, error) {
	if q == nil || q.productRepo == nil {
		return nil, fmt.Errorf("print product query repo is nil")
	}

	pid := strings.Trim(productionID, " \t\r\n/")
	if pid == "" {
		return nil, productdom.ErrInvalidPrintLogProductionID
	}

	products, err := q.productRepo.ListByProductionID(ctx, pid)
	if err != nil {
		return nil, err
	}

	out := make([]ProductSummaryForPrintDTO, 0, len(products))
	for _, p := range products {
		modelID := strings.Trim(p.ModelID, " \t\r\n/")
		modelNumber := ""

		if modelID != "" && q.modelNumberResolver != nil {
			modelNumber = strings.Trim(
				q.modelNumberResolver.ResolveModelNumber(ctx, modelID),
				" \t\r\n/",
			)
		}

		out = append(out, ProductSummaryForPrintDTO{
			ID:           p.ID,
			ModelID:      p.ModelID,
			ProductionID: p.ProductionID,
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

	pid := strings.Trim(productionID, " \t\r\n/")
	if pid == "" {
		return nil, productdom.ErrInvalidPrintLogProductionID
	}

	logs, err := q.printLogRepo.ListByProductionID(ctx, pid)
	if err != nil {
		return nil, err
	}

	out := make([]PrintLogForPrintDTO, 0, len(logs))
	for _, log := range logs {
		items := make([]PrintedItemForPrintDTO, 0, len(log.Items))
		payloads := make([]string, 0, len(log.Items))

		for _, item := range log.Items {
			if item.ProductID == "" {
				continue
			}

			items = append(items, PrintedItemForPrintDTO{
				ProductID:    item.ProductID,
				DisplayOrder: item.DisplayOrder,
			})

			payloads = append(payloads, fmt.Sprintf("%s/%s", publicQRBaseURL, item.ProductID))
		}

		out = append(out, PrintLogForPrintDTO{
			ID:           log.ID,
			ProductionID: log.ProductionID,
			Items:        items,
			QrPayloads:   payloads,
		})
	}

	return out, nil
}
