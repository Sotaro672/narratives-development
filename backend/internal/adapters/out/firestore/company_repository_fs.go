// backend/internal/adapters/out/firestore/company_repository_fs.go
package firestore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	common "narratives/internal/domain/common"
	compdom "narratives/internal/domain/company"
)

// CompanyRepositoryFS implements the company repository using Firestore.
type CompanyRepositoryFS struct {
	Client *firestore.Client
}

func NewCompanyRepositoryFS(client *firestore.Client) *CompanyRepositoryFS {
	return &CompanyRepositoryFS{Client: client}
}

func (r *CompanyRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("companies")
}

// ==============================
// List (offset pagination)
// ==============================

func (r *CompanyRepositoryFS) List(
	ctx context.Context,
	filter compdom.Filter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[compdom.Company], error) {
	q := r.col().Query
	q = applyCompanySort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []compdom.Company
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[compdom.Company]{}, err
		}

		c, err := docToCompany(doc)
		if err != nil {
			return common.PageResult[compdom.Company]{}, err
		}
		if matchCompanyFilter(c, filter) {
			all = append(all, c)
		}
	}

	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 50
	}
	number := page.Number
	if number <= 0 {
		number = 1
	}
	offset := (number - 1) * perPage

	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}

	items := all[offset:end]

	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}

	return common.PageResult[compdom.Company]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

// ==============================
// ListByCursor (id-based keyset)
// ==============================

func (r *CompanyRepositoryFS) ListByCursor(
	ctx context.Context,
	filter compdom.Filter,
	sort common.Sort,
	cpage common.CursorPage,
) (common.CursorPageResult[compdom.Company], error) {
	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// Use id ASC for a stable keyset; sort is not fully honored here.
	q := r.col().OrderBy("id", firestore.Asc)

	it := q.Documents(ctx)
	defer it.Stop()

	after := strings.TrimSpace(cpage.After)
	skipping := after != ""

	var (
		items  []compdom.Company
		lastID string
	)

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.CursorPageResult[compdom.Company]{}, err
		}

		c, err := docToCompany(doc)
		if err != nil {
			return common.CursorPageResult[compdom.Company]{}, err
		}
		if !matchCompanyFilter(c, filter) {
			continue
		}

		if skipping {
			if c.ID <= after {
				continue
			}
			skipping = false
		}

		items = append(items, c)
		lastID = c.ID

		if len(items) >= limit+1 {
			break
		}
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &lastID
	}

	return common.CursorPageResult[compdom.Company]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// ==============================
// Get / Exists / Count
// ==============================

func (r *CompanyRepositoryFS) GetByID(ctx context.Context, id string) (compdom.Company, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return compdom.Company{}, compdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return compdom.Company{}, compdom.ErrNotFound
		}
		return compdom.Company{}, err
	}
	return docToCompany(snap)
}

func (r *CompanyRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
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

func (r *CompanyRepositoryFS) Count(ctx context.Context, filter compdom.Filter) (int, error) {
	q := r.col().Query
	it := q.Documents(ctx)
	defer it.Stop()

	count := 0
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		c, err := docToCompany(doc)
		if err != nil {
			return 0, err
		}
		if matchCompanyFilter(c, filter) {
			count++
		}
	}
	return count, nil
}

// ==============================
// Mutations
// ==============================

func (r *CompanyRepositoryFS) Create(ctx context.Context, c compdom.Company) (compdom.Company, error) {
	now := time.Now().UTC()

	var docRef *firestore.DocumentRef
	if strings.TrimSpace(c.ID) == "" {
		docRef = r.col().NewDoc()
		c.ID = docRef.ID
	} else {
		docRef = r.col().Doc(strings.TrimSpace(c.ID))
	}

	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}

	data := companyToDocData(c)
	data["id"] = c.ID

	_, err := docRef.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return compdom.Company{}, compdom.ErrConflict
		}
		return compdom.Company{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return compdom.Company{}, err
	}
	return docToCompany(snap)
}

func (r *CompanyRepositoryFS) Update(ctx context.Context, id string, patch compdom.CompanyPatch) (compdom.Company, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return compdom.Company{}, compdom.ErrNotFound
	}

	docRef := r.col().Doc(id)
	var updates []firestore.Update

	if patch.Name != nil {
		updates = append(updates, firestore.Update{Path: "name", Value: strings.TrimSpace(*patch.Name)})
	}
	if patch.Admin != nil {
		updates = append(updates, firestore.Update{Path: "admin", Value: strings.TrimSpace(*patch.Admin)})
	}
	if patch.IsActive != nil {
		updates = append(updates, firestore.Update{Path: "isActive", Value: *patch.IsActive})
	}
	if patch.UpdatedAt != nil {
		if patch.UpdatedAt.IsZero() {
			updates = append(updates, firestore.Update{Path: "updatedAt", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "updatedAt", Value: patch.UpdatedAt.UTC()})
		}
	}
	if patch.UpdatedBy != nil {
		v := strings.TrimSpace(*patch.UpdatedBy)
		if v == "" {
			updates = append(updates, firestore.Update{Path: "updatedBy", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "updatedBy", Value: v})
		}
	}
	if patch.DeletedAt != nil {
		if patch.DeletedAt.IsZero() {
			updates = append(updates, firestore.Update{Path: "deletedAt", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "deletedAt", Value: patch.DeletedAt.UTC()})
		}
	}
	if patch.DeletedBy != nil {
		v := strings.TrimSpace(*patch.DeletedBy)
		if v == "" {
			updates = append(updates, firestore.Update{Path: "deletedBy", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "deletedBy", Value: v})
		}
	}

	if len(updates) == 0 {
		return r.GetByID(ctx, id)
	}

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return compdom.Company{}, compdom.ErrNotFound
		}
		return compdom.Company{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *CompanyRepositoryFS) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return compdom.ErrNotFound
	}

	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return compdom.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *CompanyRepositoryFS) Save(ctx context.Context, c compdom.Company, _ *common.SaveOptions) (compdom.Company, error) {
	now := time.Now().UTC()

	var docRef *firestore.DocumentRef
	if strings.TrimSpace(c.ID) == "" {
		docRef = r.col().NewDoc()
		c.ID = docRef.ID
	} else {
		docRef = r.col().Doc(strings.TrimSpace(c.ID))
	}

	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}

	data := companyToDocData(c)
	data["id"] = c.ID

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return compdom.Company{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return compdom.Company{}, err
	}
	return docToCompany(snap)
}

// ==============================
// Helpers
// ==============================

func companyToDocData(c compdom.Company) map[string]any {
	m := map[string]any{
		"id":        strings.TrimSpace(c.ID),
		"name":      strings.TrimSpace(c.Name),
		"admin":     strings.TrimSpace(c.Admin),
		"isActive":  c.IsActive,
		"createdAt": c.CreatedAt.UTC(),
	}

	if strings.TrimSpace(c.CreatedBy) != "" {
		m["createdBy"] = strings.TrimSpace(c.CreatedBy)
	}
	if !c.UpdatedAt.IsZero() {
		m["updatedAt"] = c.UpdatedAt.UTC()
	}
	if strings.TrimSpace(c.UpdatedBy) != "" {
		m["updatedBy"] = strings.TrimSpace(c.UpdatedBy)
	}
	if c.DeletedAt != nil && !c.DeletedAt.IsZero() {
		m["deletedAt"] = c.DeletedAt.UTC()
	}
	if c.DeletedBy != nil && strings.TrimSpace(*c.DeletedBy) != "" {
		m["deletedBy"] = strings.TrimSpace(*c.DeletedBy)
	}

	return m
}

func docToCompany(doc *firestore.DocumentSnapshot) (compdom.Company, error) {
	data := doc.Data()
	if data == nil {
		return compdom.Company{}, fmt.Errorf("empty company document: %s", doc.Ref.ID)
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	getBool := func(keys ...string) bool {
		for _, k := range keys {
			if v, ok := data[k].(bool); ok {
				return v
			}
		}
		return false
	}
	getTimePtr := func(keys ...string) *time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok {
				t := v.UTC()
				return &t
			}
		}
		return nil
	}
	getTimeVal := func(keys ...string) time.Time {
		if pt := getTimePtr(keys...); pt != nil {
			return *pt
		}
		return time.Time{}
	}

	var c compdom.Company

	c.ID = getStr("id")
	if c.ID == "" {
		c.ID = doc.Ref.ID
	}
	c.Name = getStr("name")
	c.Admin = getStr("admin")
	c.IsActive = getBool("isActive", "is_active")

	if t := getTimeVal("createdAt", "created_at"); !t.IsZero() {
		c.CreatedAt = t
	}

	c.CreatedBy = getStr("createdBy", "created_by")

	if pt := getTimePtr("updatedAt", "updated_at"); pt != nil {
		c.UpdatedAt = *pt
	}

	if s := getStr("updatedBy", "updated_by"); s != "" {
		c.UpdatedBy = s
	}

	if pt := getTimePtr("deletedAt", "deleted_at"); pt != nil {
		c.DeletedAt = pt
	}
	if s := getStr("deletedBy", "deleted_by"); s != "" {
		c.DeletedBy = &s
	}

	return c, nil
}

func matchCompanyFilter(c compdom.Company, f compdom.Filter) bool {
	// SearchQuery: partial match on name/admin
	if strings.TrimSpace(f.SearchQuery) != "" {
		q := strings.ToLower(strings.TrimSpace(f.SearchQuery))
		if !strings.Contains(strings.ToLower(c.Name), q) &&
			!strings.Contains(strings.ToLower(c.Admin), q) {
			return false
		}
	}

	// IDs
	if len(f.IDs) > 0 && !containsStringIn(f.IDs, c.ID) {
		return false
	}

	if f.Name != nil && strings.TrimSpace(*f.Name) != "" &&
		c.Name != strings.TrimSpace(*f.Name) {
		return false
	}
	if f.Admin != nil && strings.TrimSpace(*f.Admin) != "" &&
		c.Admin != strings.TrimSpace(*f.Admin) {
		return false
	}
	if f.IsActive != nil && c.IsActive != *f.IsActive {
		return false
	}

	if f.CreatedBy != nil && strings.TrimSpace(*f.CreatedBy) != "" &&
		c.CreatedBy != strings.TrimSpace(*f.CreatedBy) {
		return false
	}
	if f.UpdatedBy != nil && strings.TrimSpace(*f.UpdatedBy) != "" &&
		c.UpdatedBy != strings.TrimSpace(*f.UpdatedBy) {
		return false
	}
	if f.DeletedBy != nil && strings.TrimSpace(*f.DeletedBy) != "" {
		if c.DeletedBy == nil || *c.DeletedBy != strings.TrimSpace(*f.DeletedBy) {
			return false
		}
	}

	// Time ranges
	if f.CreatedFrom != nil && c.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && c.CreatedAt.After(f.CreatedTo.UTC()) {
		return false
	}
	if f.UpdatedFrom != nil && !c.UpdatedAt.IsZero() && c.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
		return false
	}
	if f.UpdatedTo != nil && !c.UpdatedAt.IsZero() && c.UpdatedAt.After(f.UpdatedTo.UTC()) {
		return false
	}
	if f.DeletedFrom != nil {
		if c.DeletedAt == nil || c.DeletedAt.Before(f.DeletedFrom.UTC()) {
			return false
		}
	}
	if f.DeletedTo != nil {
		if c.DeletedAt == nil || c.DeletedAt.After(f.DeletedTo.UTC()) {
			return false
		}
	}

	// Deleted tri-state
	if f.Deleted != nil {
		if *f.Deleted {
			if c.DeletedAt == nil {
				return false
			}
		} else {
			if c.DeletedAt != nil {
				return false
			}
		}
	}

	return true
}

func applyCompanySort(q firestore.Query, sort common.Sort) firestore.Query {
	col, dir := mapCompanySort(sort)
	if col == "" {
		// default: createdAt DESC, id DESC
		return q.OrderBy("createdAt", firestore.Desc).OrderBy("id", firestore.Desc)
	}
	// stable: secondary id
	return q.OrderBy(col, dir).OrderBy("id", firestore.Asc)
}

func mapCompanySort(sort common.Sort) (string, firestore.Direction) {
	col := strings.TrimSpace(string(sort.Column))
	col = strings.ToLower(col)

	var field string
	switch col {
	case "id":
		field = "id"
	case "name":
		field = "name"
	case "admin":
		field = "admin"
	case "isactive", "is_active":
		field = "isActive"
	case "createdat", "created_at":
		field = "createdAt"
	case "updatedat", "updated_at":
		field = "updatedAt"
	case "deletedat", "deleted_at":
		field = "deletedAt"
	default:
		field = ""
	}

	if field == "" {
		return "", firestore.Desc
	}

	dir := firestore.Asc
	if strings.EqualFold(string(sort.Order), "desc") {
		dir = firestore.Desc
	}
	return field, dir
}

func containsStringIn(list []string, v string) bool {
	v = strings.TrimSpace(v)
	for _, s := range list {
		if strings.TrimSpace(s) == v {
			return true
		}
	}
	return false
}
