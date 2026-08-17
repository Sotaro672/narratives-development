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

func NewPrintLogRepositoryFS(client *firestore.Client) *PrintLogRepositoryFS {
	return &PrintLogRepositoryFS{Client: client}
}

func (r *PrintLogRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("print_logs")
}

func (r *PrintLogRepositoryFS) Create(ctx context.Context, v printdom.PrintLog) (printdom.PrintLog, error) {
	if r == nil || r.Client == nil {
		return printdom.PrintLog{}, errors.New("firestore client is nil")
	}
	if v.ProductionID == "" {
		return printdom.PrintLog{}, printdom.ErrInvalidPrintLogProductionID
	}
	if v.ID != "" && v.ID != v.ProductionID {
		return printdom.PrintLog{}, printdom.ErrInvalidPrintLogID
	}
	if err := validateCanonicalPrintedItems(v.Items); err != nil {
		return printdom.PrintLog{}, err
	}

	// print_logs/{productionId} を唯一の identity とする。
	v.ID = v.ProductionID

	data, err := printLogToDoc(v)
	if err != nil {
		return printdom.PrintLog{}, err
	}

	docRef := r.col().Doc(v.ProductionID)
	if _, err := docRef.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return printdom.PrintLog{}, printdom.ErrConflict
		}
		return printdom.PrintLog{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return printdom.PrintLog{}, err
	}

	return docToPrintLog(snap)
}

func (r *PrintLogRepositoryFS) GetByProductionID(ctx context.Context, productionID string) (printdom.PrintLog, error) {
	if r == nil || r.Client == nil {
		return printdom.PrintLog{}, errors.New("firestore client is nil")
	}
	if productionID == "" {
		return printdom.PrintLog{}, printdom.ErrInvalidPrintLogProductionID
	}

	snap, err := r.col().Doc(productionID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return printdom.PrintLog{}, printdom.ErrNotFound
		}
		return printdom.PrintLog{}, err
	}

	return docToPrintLog(snap)
}

// ------------------------------------------------------------
// Firestore DTO
// ------------------------------------------------------------

type printLogDoc struct {
	Items []printLogItemDoc `firestore:"items"`
}

type printLogItemDoc struct {
	ProductID    string `firestore:"productId"`
	DisplayOrder int64  `firestore:"displayOrder"`
}

// ------------------------------------------------------------
// Encode
// ------------------------------------------------------------

func printLogToDoc(v printdom.PrintLog) (map[string]any, error) {
	if v.ID == "" || v.ID != v.ProductionID {
		return nil, printdom.ErrInvalidPrintLogID
	}
	if err := validateCanonicalPrintedItems(v.Items); err != nil {
		return nil, err
	}

	items := make([]map[string]any, 0, len(v.Items))
	for _, item := range v.Items {
		items = append(items, map[string]any{
			"productId":    item.ProductID,
			"displayOrder": item.DisplayOrder,
		})
	}

	// productionId は document ID と重複するため保存しない。
	return map[string]any{"items": items}, nil
}

// ------------------------------------------------------------
// Decode
// ------------------------------------------------------------

func docToPrintLog(doc *firestore.DocumentSnapshot) (printdom.PrintLog, error) {
	if doc == nil || doc.Ref == nil || doc.Ref.ID == "" {
		return printdom.PrintLog{}, printdom.ErrInvalidPrintLogID
	}

	var raw printLogDoc
	if err := doc.DataTo(&raw); err != nil {
		return printdom.PrintLog{}, fmt.Errorf("decode print_log document %q: %w", doc.Ref.ID, err)
	}

	items := make([]printdom.PrintedItem, 0, len(raw.Items))
	for _, rawItem := range raw.Items {
		displayOrder := int(rawItem.DisplayOrder)
		if int64(displayOrder) != rawItem.DisplayOrder {
			return printdom.PrintLog{}, fmt.Errorf("invalid print_log document %q: displayOrder is out of int range", doc.Ref.ID)
		}

		items = append(items, printdom.PrintedItem{
			ProductID:    rawItem.ProductID,
			DisplayOrder: displayOrder,
		})
	}

	// NewPrintLog は内部で normalize/sort するため、その前にFirestore値自体が
	// 既にcanonicalであることを検証し、read時の修復を防ぐ。
	if err := validateCanonicalPrintedItems(items); err != nil {
		return printdom.PrintLog{}, fmt.Errorf("invalid print_log document %q: %w", doc.Ref.ID, err)
	}

	printLog, err := printdom.NewPrintLog(doc.Ref.ID, doc.Ref.ID, items)
	if err != nil {
		return printdom.PrintLog{}, fmt.Errorf("invalid print_log document %q: %w", doc.Ref.ID, err)
	}

	return printLog, nil
}

// validateCanonicalPrintedItems はFirestoreへ保存する/保存されているitemsが
// domainのcanonical表現そのものであることを検証する。
// read時に空ID除外・重複除外・displayOrder補正・sortは行わない。
func validateCanonicalPrintedItems(items []printdom.PrintedItem) error {
	if len(items) == 0 {
		return printdom.ErrInvalidPrintLogItems
	}

	seen := make(map[string]struct{}, len(items))
	for i, item := range items {
		if item.ProductID == "" {
			return printdom.ErrInvalidPrintLogItem
		}
		if item.DisplayOrder <= 0 {
			return printdom.ErrInvalidPrintLogDisplayOrder
		}
		if _, exists := seen[item.ProductID]; exists {
			return printdom.ErrInvalidPrintLogItems
		}
		seen[item.ProductID] = struct{}{}

		if i == 0 {
			continue
		}

		prev := items[i-1]
		if item.DisplayOrder < prev.DisplayOrder {
			return printdom.ErrInvalidPrintLogItems
		}
		if item.DisplayOrder == prev.DisplayOrder && item.ProductID < prev.ProductID {
			return printdom.ErrInvalidPrintLogItems
		}
	}

	return nil
}
