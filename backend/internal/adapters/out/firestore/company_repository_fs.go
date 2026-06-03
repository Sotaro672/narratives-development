// backend/internal/adapters/out/firestore/company_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
// ID
// ==============================

func (r *CompanyRepositoryFS) NewID(ctx context.Context) (string, error) {
	if r == nil || r.Client == nil {
		return "", errors.New("company repository: client is nil")
	}

	doc := r.Client.Collection("companies").NewDoc()
	return doc.ID, nil
}

// ==============================
// Get
// ==============================

func (r *CompanyRepositoryFS) GetByID(ctx context.Context, id string) (compdom.Company, error) {
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

// ==============================
// Mutations
// ==============================

func (r *CompanyRepositoryFS) Create(ctx context.Context, c compdom.Company) (compdom.Company, error) {
	now := time.Now().UTC()

	var docRef *firestore.DocumentRef
	if c.ID == "" {
		docRef = r.col().NewDoc()
		c.ID = docRef.ID
	} else {
		docRef = r.col().Doc(c.ID)
	}

	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}

	data := companyToDocData(c)

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
	if id == "" {
		return compdom.Company{}, compdom.ErrNotFound
	}

	docRef := r.col().Doc(id)
	var updates []firestore.Update

	if patch.Name != nil {
		updates = append(updates, firestore.Update{Path: "name", Value: *patch.Name})
	}
	if patch.Admin != nil {
		updates = append(updates, firestore.Update{Path: "admin", Value: *patch.Admin})
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
		if *patch.UpdatedBy == "" {
			updates = append(updates, firestore.Update{Path: "updatedBy", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "updatedBy", Value: *patch.UpdatedBy})
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
		if *patch.DeletedBy == "" {
			updates = append(updates, firestore.Update{Path: "deletedBy", Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: "deletedBy", Value: *patch.DeletedBy})
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

// ==============================
// Helpers
// ==============================

func companyToDocData(c compdom.Company) map[string]any {
	m := map[string]any{
		"id":        c.ID,
		"name":      c.Name,
		"admin":     c.Admin,
		"isActive":  c.IsActive,
		"createdAt": c.CreatedAt.UTC(),
	}

	if c.CreatedBy != "" {
		m["createdBy"] = c.CreatedBy
	}
	if !c.UpdatedAt.IsZero() {
		m["updatedAt"] = c.UpdatedAt.UTC()
	}
	if c.UpdatedBy != "" {
		m["updatedBy"] = c.UpdatedBy
	}
	if c.DeletedAt != nil && !c.DeletedAt.IsZero() {
		m["deletedAt"] = c.DeletedAt.UTC()
	}
	if c.DeletedBy != nil && *c.DeletedBy != "" {
		m["deletedBy"] = *c.DeletedBy
	}

	return m
}

func docToCompany(doc *firestore.DocumentSnapshot) (compdom.Company, error) {
	data := doc.Data()
	if data == nil {
		return compdom.Company{}, fmt.Errorf("empty company document: %s", doc.Ref.ID)
	}

	getStr := func(key string) string {
		if v, ok := data[key].(string); ok {
			return v
		}
		return ""
	}

	getBool := func(key string) bool {
		if v, ok := data[key].(bool); ok {
			return v
		}
		return false
	}

	getTimePtr := func(key string) *time.Time {
		if v, ok := data[key].(time.Time); ok {
			t := v.UTC()
			return &t
		}
		return nil
	}

	getTimeVal := func(key string) time.Time {
		if pt := getTimePtr(key); pt != nil {
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
	c.IsActive = getBool("isActive")

	if t := getTimeVal("createdAt"); !t.IsZero() {
		c.CreatedAt = t
	}

	c.CreatedBy = getStr("createdBy")

	if pt := getTimePtr("updatedAt"); pt != nil {
		c.UpdatedAt = *pt
	}

	if s := getStr("updatedBy"); s != "" {
		c.UpdatedBy = s
	}

	if pt := getTimePtr("deletedAt"); pt != nil {
		c.DeletedAt = pt
	}

	if s := getStr("deletedBy"); s != "" {
		c.DeletedBy = &s
	}

	return c, nil
}

// ==============================
// compile-time interface checks
// ==============================

var _ compdom.Repository = (*CompanyRepositoryFS)(nil)
