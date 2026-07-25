// backend/internal/adapters/out/firestore/print_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
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
	if r.Client == nil {
		return printdom.PrintLog{}, errors.New("firestore client is nil")
	}

	if v.ProductionID == "" {
		return printdom.PrintLog{}, printdom.ErrInvalidPrintLogProductionID
	}

	// print_logs/{productionId} として保存する。
	// PrintLog.ID も productionId と一致させる。
	v.ID = v.ProductionID
	docRef := r.col().Doc(v.ProductionID)

	data := printLogToDoc(v)

	_, err := docRef.Create(ctx, data)
	if err != nil {
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
	if r.Client == nil {
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

func docToPrintLog(doc *firestore.DocumentSnapshot) (printdom.PrintLog, error) {
	data := doc.Data()
	if data == nil {
		return printdom.PrintLog{}, fmt.Errorf("empty print_log document: %s", doc.Ref.ID)
	}

	var items []printdom.PrintedItem
	if raw, ok := data["items"]; ok {
		switch vv := raw.(type) {
		case []interface{}:
			for _, x := range vv {
				m, ok := x.(map[string]interface{})
				if !ok {
					continue
				}

				pidAny := m["productId"]
				orderAny := m["displayOrder"]

				pid, _ := pidAny.(string)

				var order int
				switch t := orderAny.(type) {
				case int:
					order = t
				case int64:
					order = int(t)
				case float64:
					order = int(t)
				default:
					order = 0
				}

				if pid == "" || order <= 0 {
					continue
				}

				items = append(items, printdom.PrintedItem{
					ProductID:    pid,
					DisplayOrder: order,
				})
			}
		}
	}

	productionID := fscommon.AsString(data["productionId"])
	if productionID == "" {
		productionID = doc.Ref.ID
	}

	return printdom.NewPrintLog(
		doc.Ref.ID,
		productionID,
		items,
	)
}
func (r *PrintLogRepositoryFS) ExistsByProductionID(ctx context.Context, productionID string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	if productionID == "" {
		return false, printdom.ErrInvalidPrintLogProductionID
	}

	_, err := r.col().Doc(productionID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
func printLogToDoc(v printdom.PrintLog) map[string]any {
	items := make([]map[string]any, 0, len(v.Items))
	for _, it := range v.Items {
		items = append(items, map[string]any{
			"productId":    it.ProductID,
			"displayOrder": it.DisplayOrder,
		})
	}

	return map[string]any{
		"productionId": v.ProductionID,
		"items":        items,
	}
}
