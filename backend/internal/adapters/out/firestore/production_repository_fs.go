// backend/internal/adapters/out/firestore/production_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	proddom "narratives/internal/domain/production"
)

// ============================================================
// Firestore-based Production Repository
// (Firestore implementation corresponding to PG version)
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
// Facade methods used from HTTP handler via Usecase
// ============================================================

// GetByID returns a Production by document ID.
func (r *ProductionRepositoryFS) GetByID(ctx context.Context, id string) (proddom.Production, error) {
	if r.Client == nil {
		return proddom.Production{}, errors.New("firestore client is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return proddom.Production{}, proddom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return proddom.Production{}, proddom.ErrNotFound
	}
	if err != nil {
		return proddom.Production{}, err
	}

	return docToProduction(snap)
}

// Exists returns true if a document with that ID exists.
func (r *ProductionRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}

	_, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Create creates a new Production document.
// - If ID is empty -> auto ID
// - If Status is empty -> "manufacturing"
// - CreatedAt/UpdatedAt default to now (UTC) if zero
func (r *ProductionRepositoryFS) Create(ctx context.Context, p proddom.Production) (proddom.Production, error) {
	if r.Client == nil {
		return proddom.Production{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	// Defaults
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now
	}
	if strings.TrimSpace(string(p.Status)) == "" {
		p.Status = proddom.ProductionStatus("manufacturing")
	}

	// Firestore doc ref
	var ref *firestore.DocumentRef
	if strings.TrimSpace(p.ID) == "" {
		ref = r.col().NewDoc()
		p.ID = ref.ID
	} else {
		ref = r.col().Doc(strings.TrimSpace(p.ID))
	}

	data := productionToDoc(p)

	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return proddom.Production{}, proddom.ErrConflict
		}
		return proddom.Production{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return proddom.Production{}, err
	}
	return docToProduction(snap)
}

// Save is upsert-ish:
// - If ID is empty -> Create (auto ID)
// - If ID exists -> update via Set(MergeAll)
// - If ID does not exist -> treated as Create with that ID
func (r *ProductionRepositoryFS) Save(ctx context.Context, p proddom.Production) (proddom.Production, error) {
	if r.Client == nil {
		return proddom.Production{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now
	}
	if strings.TrimSpace(string(p.Status)) == "" {
		p.Status = proddom.ProductionStatus("manufacturing")
	}

	var ref *firestore.DocumentRef
	id := strings.TrimSpace(p.ID)
	if id == "" {
		ref = r.col().NewDoc()
		p.ID = ref.ID
	} else {
		ref = r.col().Doc(id)
	}

	data := productionToDoc(p)

	if _, err := ref.Set(ctx, data, firestore.MergeAll); err != nil {
		return proddom.Production{}, err
	}

	snap, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return proddom.Production{}, proddom.ErrNotFound
	}
	if err != nil {
		return proddom.Production{}, err
	}
	return docToProduction(snap)
}

// Delete performs a hard delete of the document.
func (r *ProductionRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	id = strings.TrimSpace(id)
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

// List returns all productions (sorted by createdAt DESC, then document ID DESC).
func (r *ProductionRepositoryFS) List(ctx context.Context) ([]proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	q := r.col().Query.
		OrderBy("createdAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []proddom.Production
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		p, err := docToProduction(doc)
		if err != nil {
			return nil, err
		}
		all = append(all, p)
	}

	return all, nil
}

// ListByProductBlueprintIDs は、指定された productBlueprintId のいずれかを持つ
// Production をすべて取得します。
// Firestore の "in" オペレータ制限（最大10要素）に対応するため、IDs をチャンクに分けて問い合わせます。
func (r *ProductionRepositoryFS) ListByProductBlueprintIDs(
	ctx context.Context,
	productBlueprintIDs []string,
) ([]proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	// 空なら即終了
	if len(productBlueprintIDs) == 0 {
		return []proddom.Production{}, nil
	}

	// 空文字を取り除きつつ trim & 重複排除
	uniq := make(map[string]struct{}, len(productBlueprintIDs))
	var ids []string
	for _, id := range productBlueprintIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := uniq[id]; ok {
			continue
		}
		uniq[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return []proddom.Production{}, nil
	}

	const maxIn = 10
	var results []proddom.Production

	for start := 0; start < len(ids); start += maxIn {
		end := start + maxIn
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]

		q := r.col().
			Where("productBlueprintId", "in", chunk)

		it := q.Documents(ctx)
		defer it.Stop()

		for {
			doc, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return nil, err
			}
			p, err := docToProduction(doc)
			if err != nil {
				return nil, err
			}
			results = append(results, p)
		}
	}

	return results, nil
}

// ============================================================
// Mapping Helpers
// ============================================================

func docToProduction(doc *firestore.DocumentSnapshot) (proddom.Production, error) {
	var raw struct {
		ProductBlueprintID string                  `firestore:"productBlueprintId"`
		AssigneeID         string                  `firestore:"assigneeId"`
		Models             []proddom.ModelQuantity `firestore:"models"`
		Status             string                  `firestore:"status"`
		PrintedAt          *time.Time              `firestore:"printedAt"`
		InspectedAt        *time.Time              `firestore:"inspectedAt"`
		CreatedBy          *string                 `firestore:"createdBy"`
		CreatedAt          time.Time               `firestore:"createdAt"`
		UpdatedBy          *string                 `firestore:"updatedBy"`
		UpdatedAt          *time.Time              `firestore:"updatedAt"`
		DeletedBy          *string                 `firestore:"deletedBy"`
		DeletedAt          *time.Time              `firestore:"deletedAt"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return proddom.Production{}, err
	}

	statusStr := strings.TrimSpace(raw.Status)
	if statusStr == "" {
		statusStr = "manufacturing"
	}

	out := proddom.Production{
		ID:                 doc.Ref.ID,
		ProductBlueprintID: strings.TrimSpace(raw.ProductBlueprintID),
		AssigneeID:         strings.TrimSpace(raw.AssigneeID),
		Models:             raw.Models,
		Status:             proddom.ProductionStatus(statusStr),
		PrintedAt:          normalizeTimePtr(raw.PrintedAt),
		InspectedAt:        normalizeTimePtr(raw.InspectedAt),
		CreatedBy:          fscommon.TrimPtr(raw.CreatedBy),
		CreatedAt:          raw.CreatedAt.UTC(),
		UpdatedBy:          fscommon.TrimPtr(raw.UpdatedBy),
		DeletedBy:          fscommon.TrimPtr(raw.DeletedBy),
	}

	if raw.UpdatedAt != nil && !raw.UpdatedAt.IsZero() {
		out.UpdatedAt = raw.UpdatedAt.UTC()
	} else {
		out.UpdatedAt = out.CreatedAt
	}
	if raw.DeletedAt != nil && !raw.DeletedAt.IsZero() {
		t := raw.DeletedAt.UTC()
		out.DeletedAt = &t
	}

	return out, nil
}

func productionToDoc(p proddom.Production) map[string]any {
	status := strings.TrimSpace(string(p.Status))
	if status == "" {
		status = "manufacturing"
	}

	m := map[string]any{
		"productBlueprintId": strings.TrimSpace(p.ProductBlueprintID),
		"assigneeId":         strings.TrimSpace(p.AssigneeID),
		"models":             p.Models,
		"status":             status,
		"createdAt":          p.CreatedAt.UTC(),
		"updatedAt":          p.UpdatedAt.UTC(),
	}

	if p.CreatedBy != nil {
		if s := strings.TrimSpace(*p.CreatedBy); s != "" {
			m["createdBy"] = s
		}
	}
	if p.PrintedAt != nil && !p.PrintedAt.IsZero() {
		m["printedAt"] = p.PrintedAt.UTC()
	}
	if p.InspectedAt != nil && !p.InspectedAt.IsZero() {
		m["inspectedAt"] = p.InspectedAt.UTC()
	}
	if p.UpdatedBy != nil {
		if s := strings.TrimSpace(*p.UpdatedBy); s != "" {
			m["updatedBy"] = s
		}
	}
	if p.DeletedAt != nil && !p.DeletedAt.IsZero() {
		m["deletedAt"] = p.DeletedAt.UTC()
	}
	if p.DeletedBy != nil {
		if s := strings.TrimSpace(*p.DeletedBy); s != "" {
			m["deletedBy"] = s
		}
	}

	return m
}

func normalizeTimePtr(t *time.Time) *time.Time {
	if t == nil || t.IsZero() {
		return nil
	}
	tt := t.UTC()
	return &tt
}
