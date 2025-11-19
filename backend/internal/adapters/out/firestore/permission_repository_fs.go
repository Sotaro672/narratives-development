// backend/internal/adapters/out/firestore/permission_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	permission "narratives/internal/domain/permission"
)

// Firestore-based implementation of permission.Repository (read-only).
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
// Port (domain permission.Repository) implementation
//   - Read-only: List / GetByID のみを公開
// ============================================================

// GetByID implements permission.Repository.GetByID.
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

// List implements permission.Repository.List and supports filter + sort + paging.
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
