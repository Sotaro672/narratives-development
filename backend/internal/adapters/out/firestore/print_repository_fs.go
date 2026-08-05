// backend/internal/adapters/out/firestore/print_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	printdom "narratives/internal/domain/print"
)

// PrintLogRepositoryFS is a Firestore-based implementation of the print log repository.
type PrintLogRepositoryFS struct {
	Client *firestore.Client
}

func NewPrintLogRepositoryFS(
	client *firestore.Client,
) *PrintLogRepositoryFS {
	return &PrintLogRepositoryFS{
		Client: client,
	}
}

func (r *PrintLogRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("print_logs")
}

func (r *PrintLogRepositoryFS) Create(
	ctx context.Context,
	v printdom.PrintLog,
) (printdom.PrintLog, error) {
	if r == nil || r.Client == nil {
		return printdom.PrintLog{},
			errors.New("firestore client is nil")
	}

	if v.ProductionID == "" {
		return printdom.PrintLog{},
			printdom.ErrInvalidPrintLogProductionID
	}

	// print_logs/{productionId} として保存する。
	// PrintLog.ID も productionId と一致させる。
	v.ID = v.ProductionID

	docRef := r.col().Doc(v.ProductionID)
	data := printLogToDoc(v)

	if _, err := docRef.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return printdom.PrintLog{},
				printdom.ErrConflict
		}

		return printdom.PrintLog{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return printdom.PrintLog{}, err
	}

	return docToPrintLog(snap)
}

func (r *PrintLogRepositoryFS) GetByProductionID(
	ctx context.Context,
	productionID string,
) (printdom.PrintLog, error) {
	if r == nil || r.Client == nil {
		return printdom.PrintLog{},
			errors.New("firestore client is nil")
	}

	if productionID == "" {
		return printdom.PrintLog{},
			printdom.ErrInvalidPrintLogProductionID
	}

	snap, err := r.col().Doc(productionID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return printdom.PrintLog{},
				printdom.ErrNotFound
		}

		return printdom.PrintLog{}, err
	}

	return docToPrintLog(snap)
}

func printLogToDoc(
	v printdom.PrintLog,
) map[string]any {
	items := make([]map[string]any, 0, len(v.Items))

	for _, item := range v.Items {
		items = append(items, map[string]any{
			"productId":    item.ProductID,
			"displayOrder": item.DisplayOrder,
		})
	}

	return map[string]any{
		"productionId": v.ProductionID,
		"items":        items,
	}
}

func docToPrintLog(
	doc *firestore.DocumentSnapshot,
) (printdom.PrintLog, error) {
	data := doc.Data()
	if data == nil {
		return printdom.PrintLog{},
			fmt.Errorf(
				"empty print_log document: %s",
				doc.Ref.ID,
			)
	}

	items := make([]printdom.PrintedItem, 0)

	rawItems, ok := data["items"]
	if ok && rawItems != nil {
		switch values := rawItems.(type) {
		case []any:
			for _, value := range values {
				itemData, ok := value.(map[string]any)
				if !ok {
					continue
				}

				productID := asString(
					itemData["productId"],
				)
				displayOrder := asInt(
					itemData["displayOrder"],
				)

				if productID == "" || displayOrder <= 0 {
					continue
				}

				items = append(
					items,
					printdom.PrintedItem{
						ProductID:    productID,
						DisplayOrder: displayOrder,
					},
				)
			}
		}
	}

	productionID := asString(
		data["productionId"],
	)
	if productionID == "" {
		productionID = doc.Ref.ID
	}

	return printdom.NewPrintLog(
		doc.Ref.ID,
		productionID,
		items,
	)
}
