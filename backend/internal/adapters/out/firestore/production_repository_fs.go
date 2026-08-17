// backend/internal/adapters/out/firestore/production_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	proddom "narratives/internal/domain/production"
)

// ============================================================
// Firestore-based Production Repository
// ============================================================

type ProductionRepositoryFS struct {
	Client *firestore.Client
}

func NewProductionRepositoryFS(client *firestore.Client) *ProductionRepositoryFS {
	return &ProductionRepositoryFS{Client: client}
}

func (r *ProductionRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("productions")
}

// ============================================================
// RepositoryPort 実装
// ============================================================

// GetByID returns a Production by document ID.
func (r *ProductionRepositoryFS) GetByID(ctx context.Context, id string) (*proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if id == "" {
		return nil, proddom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, proddom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	p, err := docToProduction(snap)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

// Create creates a new Production document from CreateProductionInput.
// - ID は CreateProductionInput には含まれないため、常に Firestore の auto ID を採番
// - Printed が nil の場合は false 扱い
// - CreatedAt/UpdatedAt は省略時 now(UTC)
func (r *ProductionRepositoryFS) Create(ctx context.Context, in proddom.CreateProductionInput) (*proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	printed := false
	if in.Printed != nil {
		printed = *in.Printed
	}

	var printedAt *time.Time
	if printed {
		if in.PrintedAt != nil && !in.PrintedAt.IsZero() {
			t := in.PrintedAt.UTC()
			printedAt = &t
		} else {
			t := now
			printedAt = &t
		}
	}

	p := proddom.Production{
		ProductBlueprintID: in.ProductBlueprintID,
		AssigneeID:         in.AssigneeID,
		Models:             in.Models,
		Printed:            printed,
		PrintedAt:          printedAt,
		PrintedBy:          nil,
		CreatedBy:          in.CreatedBy,
		CreatedAt:          createdAt,
		UpdatedAt:          createdAt,
		UpdatedBy:          nil,
	}

	ref := r.col().NewDoc()
	p.ID = ref.ID

	if err := p.Validate(); err != nil {
		return nil, err
	}

	if _, err := ref.Create(ctx, productionToDoc(p)); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, proddom.ErrConflict
		}
		return nil, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	out, err := docToProduction(snap)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

// Update updates an existing Production document.
// - 新規作成は行わない
// - ID が空、または対象 document が存在しない場合は ErrNotFound を返す
func (r *ProductionRepositoryFS) Update(ctx context.Context, p proddom.Production) (*proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if p.ID == "" {
		return nil, proddom.ErrNotFound
	}

	now := time.Now().UTC()

	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now
	}

	if p.Printed {
		if p.PrintedAt == nil || p.PrintedAt.IsZero() {
			t := now
			p.PrintedAt = &t
		} else {
			t := p.PrintedAt.UTC()
			p.PrintedAt = &t
		}
	} else {
		p.PrintedAt = nil
		p.PrintedBy = nil
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	ref := r.col().Doc(p.ID)

	_, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, proddom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if _, err := ref.Set(ctx, productionToDoc(p), firestore.MergeAll); err != nil {
		return nil, err
	}

	snap, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, proddom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	out, err := docToProduction(snap)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

// Delete performs a hard delete of the document.
func (r *ProductionRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	if id == "" {
		return proddom.ErrNotFound
	}

	ref := r.col().Doc(id)

	_, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return proddom.ErrNotFound
	}
	if err != nil {
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}

	return nil
}

// ListByProductBlueprintID は、指定された productBlueprintId のいずれかを持つ Production をすべて取得します。
// Firestore の "in" オペレータ制限に対応するため、IDs をチャンクに分けて問い合わせます。
// ID は Firestore の値として完全一致で扱い、大文字小文字の正規化は行いません。
func (r *ProductionRepositoryFS) ListByProductBlueprintID(ctx context.Context, productBlueprintIDs []string) ([]proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if len(productBlueprintIDs) == 0 {
		return []proddom.Production{}, nil
	}

	seen := make(map[string]struct{}, len(productBlueprintIDs))
	ids := make([]string, 0, len(productBlueprintIDs))

	for _, id := range productBlueprintIDs {
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}

		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return []proddom.Production{}, nil
	}

	const maxIn = 10
	results := make([]proddom.Production, 0)

	for start := 0; start < len(ids); start += maxIn {
		end := start + maxIn
		if end > len(ids) {
			end = len(ids)
		}

		chunk := ids[start:end]
		it := r.col().Where("productBlueprintId", "in", chunk).Documents(ctx)

		for {
			doc, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				it.Stop()
				return nil, err
			}

			p, err := docToProduction(doc)
			if err != nil {
				it.Stop()
				return nil, err
			}

			results = append(results, p)
		}

		it.Stop()
	}

	return results, nil
}

// GetTotalQuantityByModelID は、productBlueprintIDs 配下の Production.Models を集計し、modelId ごとの totalQuantity を返す。
// modelId は完全一致で扱い、大文字小文字の正規化・不正値の読み飛ばしは行いません。
func (r *ProductionRepositoryFS) GetTotalQuantityByModelID(ctx context.Context, productBlueprintIDs []string) ([]proddom.ModelTotalQuantity, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	prods, err := r.ListByProductBlueprintID(ctx, productBlueprintIDs)
	if err != nil {
		return nil, err
	}
	if len(prods) == 0 {
		return []proddom.ModelTotalQuantity{}, nil
	}

	totalByModelID := make(map[string]int, 64)

	for _, p := range prods {
		for _, mq := range p.Models {
			totalByModelID[mq.ModelID] += mq.Quantity
		}
	}

	out := make([]proddom.ModelTotalQuantity, 0, len(totalByModelID))
	for modelID, total := range totalByModelID {
		out = append(out, proddom.ModelTotalQuantity{
			ModelID:       modelID,
			TotalQuantity: total,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ModelID < out[j].ModelID
	})

	return out, nil
}

// GetProductBlueprintIDByProductionID は productionId → productBlueprintId を返します。
func (r *ProductionRepositoryFS) GetProductBlueprintIDByProductionID(ctx context.Context, productionID string) (string, error) {
	p, err := r.GetByID(ctx, productionID)
	if err != nil {
		return "", err
	}
	if p == nil {
		return "", proddom.ErrNotFound
	}

	return p.ProductBlueprintID, nil
}

// ============================================================
// Mapping Helpers
// ============================================================

func docToProduction(doc *firestore.DocumentSnapshot) (proddom.Production, error) {
	if doc == nil || doc.Ref == nil {
		return proddom.Production{}, errors.New("production document snapshot is nil")
	}

	var raw struct {
		ProductBlueprintID string                  `firestore:"productBlueprintId"`
		AssigneeID         string                  `firestore:"assigneeId"`
		Models             []proddom.ModelQuantity `firestore:"models"`
		Printed            *bool                   `firestore:"printed"`
		PrintedAt          *time.Time              `firestore:"printedAt"`
		PrintedBy          *string                 `firestore:"printedBy"`
		CreatedBy          *string                 `firestore:"createdBy"`
		CreatedAt          time.Time               `firestore:"createdAt"`
		UpdatedBy          *string                 `firestore:"updatedBy"`
		UpdatedAt          *time.Time              `firestore:"updatedAt"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return proddom.Production{}, fmt.Errorf("decode production document %q: %w", doc.Ref.ID, err)
	}

	if raw.Printed == nil {
		return proddom.Production{}, fmt.Errorf("invalid production document %q: printed is missing", doc.Ref.ID)
	}

	out := proddom.Production{
		ID:                 doc.Ref.ID,
		ProductBlueprintID: raw.ProductBlueprintID,
		AssigneeID:         raw.AssigneeID,
		Models:             raw.Models,
		Printed:            *raw.Printed,
		PrintedAt:          raw.PrintedAt,
		PrintedBy:          raw.PrintedBy,
		CreatedBy:          raw.CreatedBy,
		CreatedAt:          raw.CreatedAt,
		UpdatedBy:          raw.UpdatedBy,
	}

	if raw.UpdatedAt != nil {
		out.UpdatedAt = *raw.UpdatedAt
	}

	if err := out.Validate(); err != nil {
		return proddom.Production{}, fmt.Errorf("invalid production document %q: %w", doc.Ref.ID, err)
	}

	return out, nil
}

func productionToDoc(p proddom.Production) map[string]any {
	m := map[string]any{
		"productBlueprintId": p.ProductBlueprintID,
		"assigneeId":         p.AssigneeID,
		"models":             p.Models,
		"printed":            p.Printed,
		"createdAt":          p.CreatedAt.UTC(),
		"updatedAt":          p.UpdatedAt.UTC(),
	}

	if p.CreatedBy != nil {
		m["createdBy"] = *p.CreatedBy
	}

	if p.Printed {
		if p.PrintedAt != nil && !p.PrintedAt.IsZero() {
			m["printedAt"] = p.PrintedAt.UTC()
		}
		if p.PrintedBy != nil && *p.PrintedBy != "" {
			m["printedBy"] = *p.PrintedBy
		}
	} else {
		m["printedAt"] = nil
		m["printedBy"] = nil
	}

	if p.UpdatedBy != nil && *p.UpdatedBy != "" {
		m["updatedBy"] = *p.UpdatedBy
	}

	return m
}
