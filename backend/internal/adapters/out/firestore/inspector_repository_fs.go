package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	productdom "narratives/internal/domain/product"
)

// ------------------------------------------------------------
// InspectionRepositoryFS
// ------------------------------------------------------------

type InspectionRepositoryFS struct {
	Client *firestore.Client
}

func NewInspectionRepositoryFS(client *firestore.Client) *InspectionRepositoryFS {
	return &InspectionRepositoryFS{Client: client}
}

func (r *InspectionRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("inspections")
}

// Create: inspections/{productionId} を新規作成
func (r *InspectionRepositoryFS) Create(
	ctx context.Context,
	v productdom.InspectionBatch,
) (productdom.InspectionBatch, error) {

	if r.Client == nil {
		return productdom.InspectionBatch{}, errors.New("firestore client is nil")
	}

	pid := strings.TrimSpace(v.ProductionID)
	if pid == "" {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductionID
	}

	docRef := r.col().Doc(pid)
	data := inspectionBatchToDoc(v)

	_, err := docRef.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return productdom.InspectionBatch{}, err
		}
		return productdom.InspectionBatch{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}
	return docToInspectionBatch(snap)
}

// GetByProductionID: inspections/{productionId} を取得
func (r *InspectionRepositoryFS) GetByProductionID(
	ctx context.Context,
	productionID string,
) (productdom.InspectionBatch, error) {

	if r.Client == nil {
		return productdom.InspectionBatch{}, errors.New("firestore client is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductionID
	}

	snap, err := r.col().Doc(pid).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return productdom.InspectionBatch{}, productdom.ErrNotFound
		}
		return productdom.InspectionBatch{}, err
	}

	return docToInspectionBatch(snap)
}

// Save: Upsert
func (r *InspectionRepositoryFS) Save(
	ctx context.Context,
	v productdom.InspectionBatch,
) (productdom.InspectionBatch, error) {

	if r.Client == nil {
		return productdom.InspectionBatch{}, errors.New("firestore client is nil")
	}

	pid := strings.TrimSpace(v.ProductionID)
	if pid == "" {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductionID
	}

	docRef := r.col().Doc(pid)
	data := inspectionBatchToDoc(v)

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return productdom.InspectionBatch{}, err
	}

	return docToInspectionBatch(snap)
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func inspectionBatchToDoc(v productdom.InspectionBatch) map[string]any {
	items := make([]map[string]any, 0, len(v.Inspections))
	for _, ins := range v.Inspections {
		m := map[string]any{
			"productId": strings.TrimSpace(ins.ProductID),
		}

		if ins.InspectionResult != nil {
			m["inspectionResult"] = string(*ins.InspectionResult)
		} else {
			m["inspectionResult"] = nil
		}

		if ins.InspectedBy != nil {
			m["inspectedBy"] = strings.TrimSpace(*ins.InspectedBy)
		} else {
			m["inspectedBy"] = nil
		}

		if ins.InspectedAt != nil && !ins.InspectedAt.IsZero() {
			m["inspectedAt"] = ins.InspectedAt.UTC()
		} else {
			m["inspectedAt"] = nil
		}

		items = append(items, m)
	}

	return map[string]any{
		"productionId": strings.TrimSpace(v.ProductionID),
		"status":       string(v.Status),
		"inspections":  items,
	}
}

func docToInspectionBatch(
	doc *firestore.DocumentSnapshot,
) (productdom.InspectionBatch, error) {

	data := doc.Data()
	if data == nil {
		return productdom.InspectionBatch{}, fmt.Errorf("empty inspection document: %s", doc.Ref.ID)
	}

	batch := productdom.InspectionBatch{
		ProductionID: strings.TrimSpace(asString(data["productionId"])),
		Status:       productdom.InspectionStatus(strings.TrimSpace(asString(data["status"]))),
	}

	raw, ok := data["inspections"]
	if !ok || raw == nil {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductIDs
	}

	switch vv := raw.(type) {
	case []interface{}:
		for _, e := range vv {
			m, ok := e.(map[string]interface{})
			if !ok {
				continue
			}

			item := productdom.InspectionItem{}

			if v, ok := m["productId"].(string); ok {
				item.ProductID = strings.TrimSpace(v)
			}

			if v, ok := m["inspectionResult"].(string); ok {
				r := productdom.InspectionResult(strings.TrimSpace(v))
				item.InspectionResult = &r
			}

			if v, ok := m["inspectedBy"].(string); ok {
				s := strings.TrimSpace(v)
				item.InspectedBy = &s
			}

			if t, ok := m["inspectedAt"].(time.Time); ok {
				tt := t.UTC()
				item.InspectedAt = &tt
			}

			batch.Inspections = append(batch.Inspections, item)
		}
	}

	if batch.ProductionID == "" || len(batch.Inspections) == 0 {
		return productdom.InspectionBatch{}, productdom.ErrInvalidInspectionProductIDs
	}

	return batch, nil
}
