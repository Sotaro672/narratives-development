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

	doc := r.col().NewDoc()
	return doc.ID, nil
}

// ==============================
// Get
// ==============================

func (r *CompanyRepositoryFS) GetByID(ctx context.Context, id string) (compdom.Company, error) {
	if r == nil || r.Client == nil {
		return compdom.Company{}, errors.New("company repository: client is nil")
	}
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

func (r *CompanyRepositoryFS) ListAll(ctx context.Context) ([]compdom.Company, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("company repository: client is nil")
	}

	snapshots, err := r.col().
		OrderBy("createdAt", firestore.Desc).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, err
	}

	companies := make([]compdom.Company, 0, len(snapshots))

	for _, snapshot := range snapshots {
		company, err := docToCompany(snapshot)
		if err != nil {
			return nil, err
		}

		companies = append(companies, company)
	}

	return companies, nil
}

// ==============================
// Mutations
// ==============================

func (r *CompanyRepositoryFS) Create(ctx context.Context, c compdom.Company) (compdom.Company, error) {
	if r == nil || r.Client == nil {
		return compdom.Company{}, errors.New("company repository: client is nil")
	}

	var docRef *firestore.DocumentRef
	if c.ID == "" {
		docRef = r.col().NewDoc()
		c.ID = docRef.ID
	} else {
		docRef = r.col().Doc(c.ID)
	}

	validated, err := compdom.NewCompany(
		c.ID,
		c.Name,
		c.Admin,
		c.CreatedBy,
		c.UpdatedBy,
		c.CreatedAt,
		c.UpdatedAt,
		c.IsActive,
		c.DeletedAt,
		c.DeletedBy,
	)
	if err != nil {
		return compdom.Company{}, err
	}

	if _, err := docRef.Create(ctx, companyToDocData(validated)); err != nil {
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
	if r == nil || r.Client == nil {
		return compdom.Company{}, errors.New("company repository: client is nil")
	}
	if id == "" {
		return compdom.Company{}, compdom.ErrNotFound
	}

	docRef := r.col().Doc(id)
	var result compdom.Company

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(docRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return compdom.ErrNotFound
			}
			return err
		}

		current, err := docToCompany(snap)
		if err != nil {
			return err
		}

		if patch.Name != nil {
			current.Name = *patch.Name
		}
		if patch.Admin != nil {
			current.Admin = *patch.Admin
		}
		if patch.IsActive != nil {
			current.IsActive = *patch.IsActive
		}
		if patch.UpdatedAt != nil {
			if patch.UpdatedAt.IsZero() {
				return compdom.ErrInvalidUpdatedAt
			}
			current.UpdatedAt = *patch.UpdatedAt
		}
		if patch.UpdatedBy != nil {
			if *patch.UpdatedBy == "" {
				return compdom.ErrInvalidUpdatedBy
			}
			current.UpdatedBy = *patch.UpdatedBy
		}
		if patch.DeletedAt != nil {
			if patch.DeletedAt.IsZero() {
				current.DeletedAt = nil
			} else {
				t := *patch.DeletedAt
				current.DeletedAt = &t
			}
		}
		if patch.DeletedBy != nil {
			if *patch.DeletedBy == "" {
				current.DeletedBy = nil
			} else {
				v := *patch.DeletedBy
				current.DeletedBy = &v
			}
		}

		validated, err := compdom.NewCompany(
			current.ID,
			current.Name,
			current.Admin,
			current.CreatedBy,
			current.UpdatedBy,
			current.CreatedAt,
			current.UpdatedAt,
			current.IsActive,
			current.DeletedAt,
			current.DeletedBy,
		)
		if err != nil {
			return err
		}

		if err := tx.Set(docRef, companyToDocData(validated)); err != nil {
			return err
		}

		result = validated
		return nil
	})
	if err != nil {
		return compdom.Company{}, err
	}

	return result, nil
}

func (r *CompanyRepositoryFS) Delete(ctx context.Context, id string) error {
	if r == nil || r.Client == nil {
		return errors.New("company repository: client is nil")
	}
	if id == "" {
		return compdom.ErrNotFound
	}

	docRef := r.col().Doc(id)

	return r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if _, err := tx.Get(docRef); err != nil {
			if status.Code(err) == codes.NotFound {
				return compdom.ErrNotFound
			}
			return err
		}

		return tx.Delete(docRef)
	})
}

// ==============================
// Firestore DTO
// ==============================

type companyDoc struct {
	Name      string     `firestore:"name"`
	Admin     string     `firestore:"admin"`
	IsActive  *bool      `firestore:"isActive"`
	CreatedAt time.Time  `firestore:"createdAt"`
	CreatedBy string     `firestore:"createdBy"`
	UpdatedAt time.Time  `firestore:"updatedAt"`
	UpdatedBy string     `firestore:"updatedBy"`
	DeletedAt *time.Time `firestore:"deletedAt"`
	DeletedBy *string    `firestore:"deletedBy"`
}

// ==============================
// Helpers
// ==============================

func companyToDocData(c compdom.Company) map[string]any {
	m := map[string]any{
		"name":      c.Name,
		"admin":     c.Admin,
		"isActive":  c.IsActive,
		"createdAt": c.CreatedAt.UTC(),
		"createdBy": c.CreatedBy,
		"updatedAt": c.UpdatedAt.UTC(),
		"updatedBy": c.UpdatedBy,
	}

	if c.DeletedAt != nil {
		m["deletedAt"] = c.DeletedAt.UTC()
	}
	if c.DeletedBy != nil {
		m["deletedBy"] = *c.DeletedBy
	}

	return m
}

func docToCompany(doc *firestore.DocumentSnapshot) (compdom.Company, error) {
	if doc == nil || doc.Ref == nil || doc.Ref.ID == "" {
		return compdom.Company{}, compdom.ErrInvalidID
	}

	var raw companyDoc
	if err := doc.DataTo(&raw); err != nil {
		return compdom.Company{}, fmt.Errorf("decode company document %q: %w", doc.Ref.ID, err)
	}
	if raw.IsActive == nil {
		return compdom.Company{}, fmt.Errorf("invalid company document %q: isActive is missing", doc.Ref.ID)
	}

	company, err := compdom.NewCompany(
		doc.Ref.ID,
		raw.Name,
		raw.Admin,
		raw.CreatedBy,
		raw.UpdatedBy,
		raw.CreatedAt,
		raw.UpdatedAt,
		*raw.IsActive,
		raw.DeletedAt,
		raw.DeletedBy,
	)
	if err != nil {
		return compdom.Company{}, fmt.Errorf("invalid company document %q: %w", doc.Ref.ID, err)
	}

	return company, nil
}

// ==============================
// compile-time interface checks
// ==============================

var _ compdom.Repository = (*CompanyRepositoryFS)(nil)
