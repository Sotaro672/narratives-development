// backend/internal/adapters/out/firestore/permission_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	permission "narratives/internal/domain/permission"
)

// Firestore-based implementation of Permission repository.
type PermissionRepositoryFS struct {
	Client *firestore.Client
}

func NewPermissionRepositoryFS(client *firestore.Client) *PermissionRepositoryFS {
	return &PermissionRepositoryFS{Client: client}
}

func (r *PermissionRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("permissions")
}

// ============================================================
// Port (usecase.PermissionRepo) implementation
// ============================================================

// GetByID implements PermissionRepo.GetByID.
func (r *PermissionRepositoryFS) GetByID(ctx context.Context, id string) (permission.Permission, error) {
	if r.Client == nil {
		return permission.Permission{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return permission.Permission{}, permission.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return permission.Permission{}, permission.ErrNotFound
		}
		return permission.Permission{}, err
	}

	p, err := docToPermission(snap)
	if err != nil {
		return permission.Permission{}, err
	}
	return p, nil
}

// Exists implements PermissionRepo.Exists.
func (r *PermissionRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
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

// Create implements PermissionRepo.Create.
//
// If v.ID is empty, Firestore auto-ID is used.
// createdAt/updatedAt are managed internally (stored in Firestore, not on domain object).
func (r *PermissionRepositoryFS) Create(ctx context.Context, v permission.Permission) (permission.Permission, error) {
	if r.Client == nil {
		return permission.Permission{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	id := strings.TrimSpace(v.ID)
	var docRef *firestore.DocumentRef
	if id == "" {
		docRef = r.col().NewDoc()
		v.ID = docRef.ID
	} else {
		docRef = r.col().Doc(id)
		v.ID = id
	}

	data := permissionToDoc(v, &now)

	_, err := docRef.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return permission.Permission{}, permission.ErrConflict
		}
		return permission.Permission{}, err
	}

	// Domain Permission has no CreatedAt, so we just return v with normalized fields.
	return v, nil
}

// Save implements PermissionRepo.Save.
//
// Upsert behavior:
// - If ID is empty: behaves like Create (new doc).
// - If ID exists:
//   - Updates name/category/description.
//   - Preserves earliest createdAt (stored only in Firestore).
//   - Updates updatedAt to now.
func (r *PermissionRepositoryFS) Save(ctx context.Context, v permission.Permission) (permission.Permission, error) {
	if r.Client == nil {
		return permission.Permission{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	if id == "" {
		return r.Create(ctx, v)
	}

	docRef := r.col().Doc(id)
	now := time.Now().UTC()

	// Load existing to preserve createdAt if present.
	var createdAt time.Time
	snap, err := docRef.Get(ctx)
	if err == nil {
		if data := snap.Data(); data != nil {
			if t, ok := data["createdAt"].(time.Time); ok && !t.IsZero() {
				createdAt = t.UTC()
			}
		}
	} else if status.Code(err) != codes.NotFound {
		return permission.Permission{}, err
	}

	if createdAt.IsZero() {
		createdAt = now
	}

	v.ID = id
	data := permissionToDoc(v, &createdAt)

	_, err = docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return permission.Permission{}, err
	}

	return v, nil
}

// Delete implements PermissionRepo.Delete.
func (r *PermissionRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return permission.ErrNotFound
	}

	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return permission.ErrNotFound
		}
		return err
	}
	return nil
}

// ============================================================
// Extra queries (List / Update) - convenience, not required by Port
// ============================================================

// List supports filter + sort + paging.
func (r *PermissionRepositoryFS) List(
	ctx context.Context,
	filter permission.Filter,
	sort permission.Sort,
	page permission.Page,
) (permission.PageResult[permission.Permission], error) {
	if r.Client == nil {
		return permission.PageResult[permission.Permission]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	q := r.col().Query
	q = applyPermissionSort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []permission.Permission
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return permission.PageResult[permission.Permission]{}, err
		}
		p, err := docToPermission(doc)
		if err != nil {
			return permission.PageResult[permission.Permission]{}, err
		}
		if matchPermissionFilter(p, filter) {
			all = append(all, p)
		}
	}

	total := len(all)
	if total == 0 {
		return permission.PageResult[permission.Permission]{
			Items:      []permission.Permission{},
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

	return permission.PageResult[permission.Permission]{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// Update is a partial update helper (not part of the core Port).
func (r *PermissionRepositoryFS) Update(ctx context.Context, id string, patch permission.PermissionPatch) (permission.Permission, error) {
	if r.Client == nil {
		return permission.Permission{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return permission.Permission{}, permission.ErrNotFound
	}

	docRef := r.col().Doc(id)
	var updates []firestore.Update

	if patch.Name != nil {
		updates = append(updates, firestore.Update{
			Path:  "name",
			Value: strings.TrimSpace(*patch.Name),
		})
	}
	if patch.Category != nil {
		updates = append(updates, firestore.Update{
			Path:  "category",
			Value: strings.TrimSpace(string(*patch.Category)),
		})
	}
	if patch.Description != nil {
		updates = append(updates, firestore.Update{
			Path:  "description",
			Value: strings.TrimSpace(*patch.Description),
		})
	}

	if len(updates) == 0 {
		return r.GetByID(ctx, id)
	}

	// bump updatedAt
	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: time.Now().UTC(),
	})

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return permission.Permission{}, permission.ErrNotFound
		}
		return permission.Permission{}, err
	}

	return r.GetByID(ctx, id)
}

// ============================================================
// Helpers
// ============================================================

func docToPermission(doc *firestore.DocumentSnapshot) (permission.Permission, error) {
	data := doc.Data()
	if data == nil {
		return permission.Permission{}, fmt.Errorf("empty permission document: %s", doc.Ref.ID)
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}

	p := permission.Permission{
		ID:          doc.Ref.ID,
		Name:        getStr("name"),
		Category:    permission.PermissionCategory(getStr("category")),
		Description: getStr("description"),
	}

	return p, nil
}

func permissionToDoc(v permission.Permission, createdAt *time.Time) map[string]any {
	m := map[string]any{
		"name":     strings.TrimSpace(v.Name),
		"category": strings.TrimSpace(string(v.Category)),
	}

	if s := strings.TrimSpace(v.Description); s != "" {
		m["description"] = s
	}

	if createdAt != nil && !createdAt.IsZero() {
		m["createdAt"] = createdAt.UTC()
	}
	m["updatedAt"] = time.Now().UTC()

	return m
}

// matchPermissionFilter applies Filter in-memory (Firestore analogue of the SQL WHERE).
func matchPermissionFilter(p permission.Permission, f permission.Filter) bool {
	// SearchQuery: partial match on id, name, description (case-insensitive)
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		lsq := strings.ToLower(sq)
		haystack := strings.ToLower(
			strings.TrimSpace(p.ID) + " " +
				strings.TrimSpace(p.Name) + " " +
				strings.TrimSpace(p.Description),
		)
		if !strings.Contains(haystack, lsq) {
			return false
		}
	}

	// Categories: inclusion
	if len(f.Categories) > 0 {
		ok := false
		for _, c := range f.Categories {
			if strings.TrimSpace(string(c)) == strings.TrimSpace(string(p.Category)) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	return true
}

// applyPermissionSort maps Sort to Firestore orderBy.
func applyPermissionSort(q firestore.Query, sort permission.Sort) firestore.Query {
	col := strings.TrimSpace(string(sort.Column))
	if col == "" {
		// default: createdAt DESC (fallback to name if no timestamp)
		return q.OrderBy("createdAt", firestore.Desc).
			OrderBy("name", firestore.Asc).
			OrderBy(firestore.DocumentID, firestore.Asc)
	}

	dir := firestore.Asc
	if strings.EqualFold(string(sort.Order), "desc") {
		dir = firestore.Desc
	}

	switch col {
	case "name":
		return q.OrderBy("name", dir).OrderBy(firestore.DocumentID, dir)
	case "category":
		return q.OrderBy("category", dir).OrderBy(firestore.DocumentID, dir)
	case "createdAt", "created_at":
		return q.OrderBy("createdAt", dir).OrderBy(firestore.DocumentID, dir)
	default:
		// Fallback to createdAt DESC like PG
		return q.OrderBy("createdAt", firestore.Desc).
			OrderBy("name", firestore.Asc).
			OrderBy(firestore.DocumentID, firestore.Asc)
	}
}
