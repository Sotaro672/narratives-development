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
// Facade methods matching usecase.ProductionRepo expectations
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
// Semantics are aligned with the PG版:
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
		// unlikely right after Set, but keep symmetry
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

// ============================================================
// Extra methods (List / Count / Update / Marks / Reset ...)
// These mirror the PG repo behavior as reasonably as possible
// under Firestore constraints.
// ============================================================

// GetByModelID returns productions that include given modelID in Models.
func (r *ProductionRepositoryFS) GetByModelID(ctx context.Context, modelID string) ([]proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return []proddom.Production{}, nil
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var out []proddom.Production
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
		for _, mq := range p.Models {
			if strings.TrimSpace(mq.ModelID) == modelID {
				out = append(out, p)
				break
			}
		}
	}
	return out, nil
}

// List runs a Firestore query, then applies Filter/Sort/Paging in-memory
// (similar semantics to PG版; for large collections consider refining).
func (r *ProductionRepositoryFS) List(
	ctx context.Context,
	filter proddom.Filter,
	sort proddom.Sort,
	page proddom.Page,
) (proddom.PageResult, error) {
	if r.Client == nil {
		return proddom.PageResult{}, errors.New("firestore client is nil")
	}

	// Base query; we keep it simple and push minimal equality filters if desired.
	q := r.col().Query
	q = applyProductionOrderBy(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []proddom.Production
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return proddom.PageResult{}, err
		}
		p, err := docToProduction(doc)
		if err != nil {
			return proddom.PageResult{}, err
		}
		if matchProductionFilter(p, filter) {
			all = append(all, p)
		}
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	total := len(all)
	if total == 0 {
		return proddom.PageResult{
			Items:      []proddom.Production{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}
	items := all[offset:end]

	return proddom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// Count counts productions matching Filter (by scanning; for large sets consider optimizing).
func (r *ProductionRepositoryFS) Count(ctx context.Context, filter proddom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	total := 0
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}
		p, err := docToProduction(doc)
		if err != nil {
			return 0, err
		}
		if matchProductionFilter(p, filter) {
			total++
		}
	}
	return total, nil
}

// Update applies UpdateProductionInput as a partial update.
func (r *ProductionRepositoryFS) Update(
	ctx context.Context,
	id string,
	patch proddom.UpdateProductionInput,
) (*proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, proddom.ErrNotFound
	}

	ref := r.col().Doc(id)

	// Ensure exists
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return nil, proddom.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	var updates []firestore.Update

	setStr := func(path string, p *string) {
		if p != nil {
			v := strings.TrimSpace(*p)
			if v == "" {
				updates = append(updates, firestore.Update{Path: path, Value: nil})
			} else {
				updates = append(updates, firestore.Update{Path: path, Value: v})
			}
		}
	}

	if patch.ProductBlueprintID != nil {
		setStr("productBlueprintId", patch.ProductBlueprintID)
	}
	if patch.AssigneeID != nil {
		setStr("assigneeId", patch.AssigneeID)
	}
	if patch.Models != nil {
		updates = append(updates, firestore.Update{
			Path:  "models",
			Value: *patch.Models,
		})
	}
	if patch.Status != nil {
		v := strings.TrimSpace(string(*patch.Status))
		if v == "" {
			// If explicitly empty, clear or set default; here we clear.
			updates = append(updates, firestore.Update{Path: "status", Value: nil})
		} else {
			updates = append(updates, firestore.Update{Path: "status", Value: v})
		}
	}
	if patch.PrintedAt != nil {
		if patch.PrintedAt.IsZero() {
			updates = append(updates, firestore.Update{Path: "printedAt", Value: nil})
		} else {
			updates = append(updates, firestore.Update{Path: "printedAt", Value: patch.PrintedAt.UTC()})
		}
	}
	if patch.InspectedAt != nil {
		if patch.InspectedAt.IsZero() {
			updates = append(updates, firestore.Update{Path: "inspectedAt", Value: nil})
		} else {
			updates = append(updates, firestore.Update{Path: "inspectedAt", Value: patch.InspectedAt.UTC()})
		}
	}
	if patch.DeletedAt != nil {
		if patch.DeletedAt.IsZero() {
			updates = append(updates, firestore.Update{Path: "deletedAt", Value: nil})
		} else {
			updates = append(updates, firestore.Update{Path: "deletedAt", Value: patch.DeletedAt.UTC()})
		}
	}
	if patch.DeletedBy != nil {
		setStr("deletedBy", patch.DeletedBy)
	}
	if patch.UpdatedBy != nil {
		setStr("updatedBy", patch.UpdatedBy)
	}

	// Always bump updatedAt if not explicitly controlled
	hasUpdatedAt := false
	for _, u := range updates {
		if u.Path == "updatedAt" {
			hasUpdatedAt = true
			break
		}
	}
	if !hasUpdatedAt {
		updates = append(updates, firestore.Update{
			Path:  "updatedAt",
			Value: time.Now().UTC(),
		})
	}

	if len(updates) == 0 {
		// Nothing to update; just return current
		snap, err := ref.Get(ctx)
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

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, proddom.ErrNotFound
		}
		return nil, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, proddom.ErrNotFound
		}
		return nil, err
	}
	p, err := docToProduction(snap)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// MarkPrinted sets status='printed' and printedAt.
func (r *ProductionRepositoryFS) MarkPrinted(ctx context.Context, id string, in proddom.MarkPrintedInput) (*proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, proddom.ErrNotFound
	}

	ref := r.col().Doc(id)
	at := in.At.UTC()

	_, err := ref.Update(ctx, []firestore.Update{
		{Path: "status", Value: "printed"},
		{Path: "printedAt", Value: at},
		{Path: "updatedAt", Value: time.Now().UTC()},
	})
	if status.Code(err) == codes.NotFound {
		return nil, proddom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}
	p, err := docToProduction(snap)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// MarkInspected sets status='inspected' and inspectedAt.
func (r *ProductionRepositoryFS) MarkInspected(ctx context.Context, id string, in proddom.MarkInspectedInput) (*proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, proddom.ErrNotFound
	}

	ref := r.col().Doc(id)
	at := in.At.UTC()

	_, err := ref.Update(ctx, []firestore.Update{
		{Path: "status", Value: "inspected"},
		{Path: "inspectedAt", Value: at},
		{Path: "updatedAt", Value: time.Now().UTC()},
	})
	if status.Code(err) == codes.NotFound {
		return nil, proddom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}
	p, err := docToProduction(snap)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ResetToManufacturing clears printed/inspected fields and sets status back to 'manufacturing'.
func (r *ProductionRepositoryFS) ResetToManufacturing(ctx context.Context, id string) (*proddom.Production, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, proddom.ErrNotFound
	}

	ref := r.col().Doc(id)

	_, err := ref.Update(ctx, []firestore.Update{
		{Path: "status", Value: "manufacturing"},
		{Path: "printedAt", Value: nil},
		{Path: "inspectedAt", Value: nil},
		{Path: "updatedAt", Value: time.Now().UTC()},
	})
	if status.Code(err) == codes.NotFound {
		return nil, proddom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}
	p, err := docToProduction(snap)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Reset is mainly for tests; deletes all documents in the collection.
func (r *ProductionRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	batch := r.Client.Batch()
	count := 0

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		batch.Delete(doc.Ref)
		count++
		if count%400 == 0 {
			if _, err := batch.Commit(ctx); err != nil {
				return err
			}
			batch = r.Client.Batch()
		}
	}
	if count > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

// WithTx: Firestore版は簡易対応としてそのまま fn(ctx) を実行。
// （PG版のような SQL Tx 互換用ヘルパー呼び出し箇所と整合させるためのダミー実装）
func (r *ProductionRepositoryFS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	// For real Firestore transactions, you'd wrap fn inside Client.RunTransaction.
	// Here we call fn directly to keep interface compatibility.
	return fn(ctx)
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

	// Normalize status
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
	}
	if raw.UpdatedAt == nil || raw.UpdatedAt.IsZero() {
		// fallback to CreatedAt if missing
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
	if t == nil {
		return nil
	}
	if t.IsZero() {
		return nil
	}
	tt := t.UTC()
	return &tt
}

// ============================================================
// Filter / Sort Helpers (Firestore analogue of build* helpers)
// ============================================================

// matchProductionFilter applies proddom.Filter in-memory.
func matchProductionFilter(p proddom.Production, f proddom.Filter) bool {
	trimEq := func(a, b string) bool {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}

	if v := strings.TrimSpace(f.ID); v != "" && !trimEq(p.ID, v) {
		return false
	}
	if v := strings.TrimSpace(f.ProductBlueprintID); v != "" && !trimEq(p.ProductBlueprintID, v) {
		return false
	}
	if v := strings.TrimSpace(f.AssigneeID); v != "" && !trimEq(p.AssigneeID, v) {
		return false
	}

	// ModelID containment
	if v := strings.TrimSpace(f.ModelID); v != "" {
		found := false
		for _, mq := range p.Models {
			if trimEq(mq.ModelID, v) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Status IN
	if len(f.Statuses) > 0 {
		cur := strings.TrimSpace(string(p.Status))
		ok := false
		for _, s := range f.Statuses {
			if strings.TrimSpace(string(s)) == cur {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// Printed range
	if f.PrintedFrom != nil {
		if p.PrintedAt == nil || p.PrintedAt.Before(f.PrintedFrom.UTC()) {
			return false
		}
	}
	if f.PrintedTo != nil {
		if p.PrintedAt == nil || !p.PrintedAt.Before(f.PrintedTo.UTC()) {
			return false
		}
	}

	// Inspected range
	if f.InspectedFrom != nil {
		if p.InspectedAt == nil || p.InspectedAt.Before(f.InspectedFrom.UTC()) {
			return false
		}
	}
	if f.InspectedTo != nil {
		if p.InspectedAt == nil || !p.InspectedAt.Before(f.InspectedTo.UTC()) {
			return false
		}
	}

	// Created range
	if f.CreatedFrom != nil && p.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && !p.CreatedAt.Before(f.CreatedTo.UTC()) {
		return false
	}

	// Updated range
	if f.UpdatedFrom != nil && p.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
		return false
	}
	if f.UpdatedTo != nil && !p.UpdatedAt.Before(f.UpdatedTo.UTC()) {
		return false
	}

	// Deleted range
	if f.DeletedFrom != nil {
		if p.DeletedAt == nil || p.DeletedAt.Before(f.DeletedFrom.UTC()) {
			return false
		}
	}
	if f.DeletedTo != nil {
		if p.DeletedAt == nil || !p.DeletedAt.Before(f.DeletedTo.UTC()) {
			return false
		}
	}

	// Deleted tri-state
	if f.Deleted != nil {
		if *f.Deleted {
			if p.DeletedAt == nil {
				return false
			}
		} else {
			if p.DeletedAt != nil {
				return false
			}
		}
	}

	return true
}

// applyProductionOrderBy maps proddom.Sort to Firestore orderBy.
// Firestore requires we chain orderBy fields; we also tie-break by DocumentID.
func applyProductionOrderBy(q firestore.Query, s proddom.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	var field string

	switch col {
	case "id":
		field = firestore.DocumentID
	case "createdat", "created_at":
		field = "createdAt"
	case "updatedat", "updated_at":
		field = "updatedAt"
	case "printedat", "printed_at":
		field = "printedAt"
	case "inspectedat", "inspected_at":
		field = "inspectedAt"
	case "status":
		field = "status"
	default:
		// default: createdAt DESC, then ID DESC
		return q.OrderBy("createdAt", firestore.Desc).
			OrderBy(firestore.DocumentID, firestore.Desc)
	}

	dir := firestore.Desc
	if strings.EqualFold(string(s.Order), "asc") {
		dir = firestore.Asc
	}

	// tie-break by ID with same direction for determinism
	if field == firestore.DocumentID {
		return q.OrderBy(field, dir)
	}
	return q.OrderBy(field, dir).
		OrderBy(firestore.DocumentID, dir)
}
