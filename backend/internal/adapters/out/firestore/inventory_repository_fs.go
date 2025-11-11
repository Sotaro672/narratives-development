// backend/internal/adapters/out/firestore/inventory_repository_fs.go
package firestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	invdom "narratives/internal/domain/inventory"
)

// InventoryRepositoryFS implements inventory.Repository with Firestore.
type InventoryRepositoryFS struct {
	Client *firestore.Client
}

func NewInventoryRepositoryFS(client *firestore.Client) *InventoryRepositoryFS {
	return &InventoryRepositoryFS{Client: client}
}

func (r *InventoryRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("inventories")
}

// Compile-time check
var _ invdom.Repository = (*InventoryRepositoryFS)(nil)

// =======================
// Queries
// =======================

func (r *InventoryRepositoryFS) GetByID(ctx context.Context, id string) (invdom.Inventory, error) {
	if r.Client == nil {
		return invdom.Inventory{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.Inventory{}, invdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return invdom.Inventory{}, invdom.ErrNotFound
		}
		return invdom.Inventory{}, err
	}

	return docToInventory(snap)
}

func (r *InventoryRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}

	_, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Count: best-effort via scanning and applying Filter in-memory.
func (r *InventoryRepositoryFS) Count(ctx context.Context, filter invdom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	total := 0
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		inv, err := docToInventory(doc)
		if err != nil {
			return 0, err
		}
		if matchInventoryFilter(inv, filter) {
			total++
		}
	}
	return total, nil
}

func (r *InventoryRepositoryFS) List(
	ctx context.Context,
	filter invdom.Filter,
	sort invdom.Sort,
	page invdom.Page,
) (invdom.PageResult[invdom.Inventory], error) {
	if r.Client == nil {
		return invdom.PageResult[invdom.Inventory]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, _ := fscommon.NormalizePage(page.Number, page.PerPage, 50, 0)

	q := r.col().Query
	q = applyInventorySort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []invdom.Inventory
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return invdom.PageResult[invdom.Inventory]{}, err
		}
		inv, err := docToInventory(doc)
		if err != nil {
			return invdom.PageResult[invdom.Inventory]{}, err
		}
		if matchInventoryFilter(inv, filter) {
			all = append(all, inv)
		}
	}

	total := len(all)
	if total == 0 {
		return invdom.PageResult[invdom.Inventory]{
			Items:      []invdom.Inventory{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	offset := (pageNum - 1) * perPage
	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}
	items := all[offset:end]

	totalPages := fscommon.ComputeTotalPages(total, perPage)

	return invdom.PageResult[invdom.Inventory]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *InventoryRepositoryFS) ListByCursor(
	ctx context.Context,
	filter invdom.Filter,
	_ invdom.Sort,
	cpage invdom.CursorPage,
) (invdom.CursorPageResult[invdom.Inventory], error) {
	if r.Client == nil {
		return invdom.CursorPageResult[invdom.Inventory]{}, errors.New("firestore client is nil")
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// Simple cursor by id ASC (string compare)
	q := r.col().OrderBy("id", firestore.Asc)

	it := q.Documents(ctx)
	defer it.Stop()

	after := strings.TrimSpace(cpage.After)
	skipping := after != ""

	var (
		items  []invdom.Inventory
		lastID string
	)

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return invdom.CursorPageResult[invdom.Inventory]{}, err
		}
		inv, err := docToInventory(doc)
		if err != nil {
			return invdom.CursorPageResult[invdom.Inventory]{}, err
		}
		if !matchInventoryFilter(inv, filter) {
			continue
		}

		if skipping {
			if inv.ID <= after {
				continue
			}
			skipping = false
		}

		items = append(items, inv)
		lastID = inv.ID

		if len(items) >= limit+1 {
			break
		}
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	return invdom.CursorPageResult[invdom.Inventory]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// =======================
// Mutations
// =======================

func (r *InventoryRepositoryFS) Create(
	ctx context.Context,
	inv invdom.Inventory,
) (invdom.Inventory, error) {
	if r.Client == nil {
		return invdom.Inventory{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(inv.ID)
	if id == "" {
		return invdom.Inventory{}, errors.New("missing id")
	}

	now := time.Now().UTC()
	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = now
	}
	if inv.UpdatedAt.IsZero() {
		inv.UpdatedAt = now
	}

	inv.ID = id
	docRef := r.col().Doc(id)

	data := inventoryToDocData(inv)

	_, err := docRef.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return invdom.Inventory{}, invdom.ErrConflict
		}
		return invdom.Inventory{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return invdom.Inventory{}, err
	}
	return docToInventory(snap)
}

func (r *InventoryRepositoryFS) Update(
	ctx context.Context,
	id string,
	patch invdom.InventoryPatch,
) (invdom.Inventory, error) {
	if r.Client == nil {
		return invdom.Inventory{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.Inventory{}, invdom.ErrNotFound
	}

	docRef := r.col().Doc(id)

	var updates []firestore.Update

	if patch.Models != nil {
		updates = append(updates, firestore.Update{
			Path:  "models",
			Value: *patch.Models,
		})
	}
	if patch.Location != nil {
		updates = append(updates, firestore.Update{
			Path:  "location",
			Value: strings.TrimSpace(*patch.Location),
		})
	}
	if patch.Status != nil {
		updates = append(updates, firestore.Update{
			Path:  "status",
			Value: string(*patch.Status),
		})
	}
	if patch.ConnectedToken != nil {
		v := strings.TrimSpace(*patch.ConnectedToken)
		if v == "" {
			updates = append(updates, firestore.Update{
				Path:  "connectedToken",
				Value: firestore.Delete,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "connectedToken",
				Value: v,
			})
		}
	}
	if patch.UpdatedBy != nil {
		v := strings.TrimSpace(*patch.UpdatedBy)
		if v == "" {
			updates = append(updates, firestore.Update{
				Path:  "updatedBy",
				Value: firestore.Delete,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "updatedBy",
				Value: v,
			})
		}
	}

	// updatedAt: explicit or NOW() when there are updates
	if patch.UpdatedAt != nil {
		if patch.UpdatedAt.IsZero() {
			updates = append(updates, firestore.Update{
				Path:  "updatedAt",
				Value: firestore.Delete,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "updatedAt",
				Value: patch.UpdatedAt.UTC(),
			})
		}
	} else if len(updates) > 0 {
		updates = append(updates, firestore.Update{
			Path:  "updatedAt",
			Value: time.Now().UTC(),
		})
	}

	if len(updates) == 0 {
		// no-op
		return r.GetByID(ctx, id)
	}

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return invdom.Inventory{}, invdom.ErrNotFound
		}
		return invdom.Inventory{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *InventoryRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.ErrNotFound
	}

	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return invdom.ErrNotFound
		}
		return err
	}
	return nil
}

// Save upserts an Inventory.
func (r *InventoryRepositoryFS) Save(
	ctx context.Context,
	inv invdom.Inventory,
	_ *invdom.SaveOptions,
) (invdom.Inventory, error) {
	if r.Client == nil {
		return invdom.Inventory{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(inv.ID)
	if id == "" {
		return invdom.Inventory{}, errors.New("missing id")
	}

	now := time.Now().UTC()
	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = now
	}
	if inv.UpdatedAt.IsZero() {
		inv.UpdatedAt = now
	}

	inv.ID = id
	docRef := r.col().Doc(id)

	data := inventoryToDocData(inv)

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return invdom.Inventory{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return invdom.Inventory{}, err
	}
	return docToInventory(snap)
}

// =======================
// Helpers
// =======================

func inventoryToDocData(inv invdom.Inventory) map[string]any {
	m := map[string]any{
		"id":        strings.TrimSpace(inv.ID),
		"models":    inv.Models,
		"location":  strings.TrimSpace(inv.Location),
		"status":    string(inv.Status),
		"createdBy": strings.TrimSpace(inv.CreatedBy),
		"createdAt": inv.CreatedAt.UTC(),
		"updatedBy": strings.TrimSpace(inv.UpdatedBy),
		"updatedAt": inv.UpdatedAt.UTC(),
	}

	if v := fscommon.TrimPtr(inv.ConnectedToken); v != nil {
		m["connectedToken"] = *v
	}

	return m
}

func docToInventory(doc *firestore.DocumentSnapshot) (invdom.Inventory, error) {
	data := doc.Data()
	if data == nil {
		return invdom.Inventory{}, fmt.Errorf("empty inventory document: %s", doc.Ref.ID)
	}

	getStr := func(key string) string {
		if v, ok := data[key].(string); ok {
			return strings.TrimSpace(v)
		}
		return ""
	}
	getPtrStr := func(key string) *string {
		if v, ok := data[key].(string); ok {
			s := strings.TrimSpace(v)
			if s != "" {
				return &s
			}
		}
		return nil
	}
	getTime := func(key string) (time.Time, bool) {
		if v, ok := data[key].(time.Time); ok {
			return v.UTC(), !v.IsZero()
		}
		return time.Time{}, false
	}

	// models: Firestore may store as array of maps; re-marshal into []InventoryModel.
	var models []invdom.InventoryModel
	if raw, ok := data["models"]; ok && raw != nil {
		if b, err := json.Marshal(raw); err == nil {
			_ = json.Unmarshal(b, &models)
		}
	}
	if models == nil {
		models = []invdom.InventoryModel{}
	}

	var inv invdom.Inventory

	inv.ID = getStr("id")
	if inv.ID == "" {
		inv.ID = doc.Ref.ID
	}

	inv.ConnectedToken = getPtrStr("connectedToken")
	inv.Models = models
	inv.Location = getStr("location")
	inv.Status = invdom.InventoryStatus(getStr("status"))
	inv.CreatedBy = getStr("createdBy")
	if t, ok := getTime("createdAt"); ok {
		inv.CreatedAt = t
	}
	inv.UpdatedBy = getStr("updatedBy")
	if t, ok := getTime("updatedAt"); ok {
		inv.UpdatedAt = t
	}

	return inv, nil
}

func matchInventoryFilter(inv invdom.Inventory, f invdom.Filter) bool {
	// Free text: id, location, createdBy, updatedBy
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		lq := strings.ToLower(sq)
		haystack := strings.ToLower(
			inv.ID + " " +
				inv.Location + " " +
				inv.CreatedBy + " " +
				inv.UpdatedBy,
		)
		if !strings.Contains(haystack, lq) {
			return false
		}
	}

	// IDs
	if len(f.IDs) > 0 {
		found := false
		for _, v := range f.IDs {
			if strings.TrimSpace(v) == inv.ID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// ConnectedToken
	if f.ConnectedToken != nil {
		want := strings.TrimSpace(*f.ConnectedToken)
		has := fscommon.TrimPtr(inv.ConnectedToken)
		if want == "" {
			if has != nil {
				return false
			}
		} else {
			if has == nil || *has != want {
				return false
			}
		}
	}

	// Location
	if f.Location != nil && strings.TrimSpace(*f.Location) != "" {
		if inv.Location != strings.TrimSpace(*f.Location) {
			return false
		}
	}

	// Status
	if f.Status != nil && strings.TrimSpace(string(*f.Status)) != "" {
		if inv.Status != *f.Status {
			return false
		}
	}
	if len(f.Statuses) > 0 {
		ok := false
		for _, st := range f.Statuses {
			if inv.Status == st {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// CreatedBy / UpdatedBy
	if f.CreatedBy != nil && strings.TrimSpace(*f.CreatedBy) != "" {
		if inv.CreatedBy != strings.TrimSpace(*f.CreatedBy) {
			return false
		}
	}
	if f.UpdatedBy != nil && strings.TrimSpace(*f.UpdatedBy) != "" {
		if inv.UpdatedBy != strings.TrimSpace(*f.UpdatedBy) {
			return false
		}
	}

	// Date ranges
	if f.CreatedFrom != nil && inv.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && !inv.CreatedAt.Before(f.CreatedTo.UTC()) {
		return false
	}
	if f.UpdatedFrom != nil && inv.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
		return false
	}
	if f.UpdatedTo != nil && !inv.UpdatedAt.Before(f.UpdatedTo.UTC()) {
		return false
	}

	return true
}

func applyInventorySort(q firestore.Query, sort invdom.Sort) firestore.Query {
	col, dir := mapInventorySort(sort)
	if col == "" {
		// default: updatedAt DESC, id DESC
		return q.OrderBy("updatedAt", firestore.Desc).OrderBy("id", firestore.Desc)
	}
	return q.OrderBy(col, dir).OrderBy("id", firestore.Asc)
}

func mapInventorySort(sort invdom.Sort) (string, firestore.Direction) {
	col := strings.ToLower(string(sort.Column))
	var field string

	switch col {
	case "id":
		field = "id"
	case "location":
		field = "location"
	case "status":
		field = "status"
	case "connectedtoken", "connected_token":
		field = "connectedToken"
	case "createdat", "created_at":
		field = "createdAt"
	case "updatedat", "updated_at":
		field = "updatedAt"
	default:
		return "", firestore.Desc
	}

	dir := firestore.Asc
	if strings.EqualFold(string(sort.Order), "desc") {
		dir = firestore.Desc
	}
	return field, dir
}
