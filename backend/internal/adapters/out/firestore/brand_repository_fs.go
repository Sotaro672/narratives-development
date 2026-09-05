// backend/internal/adapters/out/firestore/brand_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"math"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	branddom "narratives/internal/domain/brand"
)

type BrandRepositoryFS struct {
	Client *firestore.Client
}

func NewBrandRepositoryFS(client *firestore.Client) *BrandRepositoryFS {
	return &BrandRepositoryFS{Client: client}
}

func (r *BrandRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("brands")
}

var _ branddom.Repository = (*BrandRepositoryFS)(nil)

func (r *BrandRepositoryFS) ListByCompanyID(ctx context.Context, companyID string, page branddom.Page) (branddom.PageResult[branddom.Brand], error) {
	if companyID == "" {
		return branddom.PageResult[branddom.Brand]{}, branddom.ErrInvalidID
	}

	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 50
	}
	if perPage > 200 {
		perPage = 200
	}

	number := page.Number
	if number <= 0 {
		number = 1
	}

	baseQuery := r.col().Query.Where("companyId", "==", companyID)

	countIter := baseQuery.Documents(ctx)
	totalCount := 0
	for {
		_, err := countIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			countIter.Stop()
			return branddom.PageResult[branddom.Brand]{}, err
		}
		totalCount++
	}
	countIter.Stop()

	offset := (number - 1) * perPage
	q := baseQuery.OrderBy("createdAt", firestore.Desc)
	if offset > 0 {
		q = q.Offset(offset)
	}
	q = q.Limit(perPage)

	iter := q.Documents(ctx)
	defer iter.Stop()

	items := make([]branddom.Brand, 0, perPage)
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return branddom.PageResult[branddom.Brand]{}, err
		}

		b, err := r.docToDomain(doc)
		if err != nil {
			return branddom.PageResult[branddom.Brand]{}, err
		}
		items = append(items, b)
	}

	totalPages := 0
	if perPage > 0 {
		totalPages = int(math.Ceil(float64(totalCount) / float64(perPage)))
	}

	return branddom.PageResult[branddom.Brand]{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *BrandRepositoryFS) GetByID(ctx context.Context, id string) (branddom.Brand, error) {
	if id == "" {
		return branddom.Brand{}, branddom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return branddom.Brand{}, branddom.ErrNotFound
	}
	if err != nil {
		return branddom.Brand{}, err
	}

	return r.docToDomain(snap)
}

func (r *BrandRepositoryFS) Create(ctx context.Context, b branddom.Brand) (branddom.Brand, error) {
	now := time.Now().UTC()

	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	if b.UpdatedAt == nil || b.UpdatedAt.IsZero() {
		b.UpdatedAt = ptrTime(b.CreatedAt)
	}

	var ref *firestore.DocumentRef
	if b.ID == "" {
		ref = r.col().NewDoc()
		b.ID = ref.ID
	} else {
		ref = r.col().Doc(b.ID)
	}

	data := r.domainToDocData(b)

	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return branddom.Brand{}, branddom.ErrConflict
		}
		return branddom.Brand{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return branddom.Brand{}, err
	}

	return r.docToDomain(snap)
}

func (r *BrandRepositoryFS) Update(ctx context.Context, id string, patch branddom.BrandPatch) (branddom.Brand, error) {
	if id == "" {
		return branddom.Brand{}, branddom.ErrNotFound
	}

	ref := r.col().Doc(id)

	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return branddom.Brand{}, branddom.ErrNotFound
	} else if err != nil {
		return branddom.Brand{}, err
	}

	var updates []firestore.Update

	if patch.CompanyID != nil {
		updates = append(updates, firestore.Update{Path: "companyId", Value: *patch.CompanyID})
	}
	if patch.AccountID != nil {
		updates = append(updates, firestore.Update{Path: "accountId", Value: *patch.AccountID})
	}
	if patch.Name != nil {
		updates = append(updates, firestore.Update{Path: "name", Value: *patch.Name})
	}
	if patch.Description != nil {
		updates = appendOptionalStringUpdate(updates, "description", patch.Description)
	}
	if patch.URL != nil {
		updates = appendOptionalStringUpdate(updates, "websiteUrl", patch.URL)
	}
	if patch.BrandIcon != nil {
		updates = appendOptionalStringUpdate(updates, "brandIcon", patch.BrandIcon)
	}
	if patch.BrandBackgroundImage != nil {
		updates = appendOptionalStringUpdate(updates, "brandBackgroundImage", patch.BrandBackgroundImage)
	}
	if patch.IsActive != nil {
		updates = append(updates, firestore.Update{Path: "isActive", Value: *patch.IsActive})
	}
	if patch.ManagerID != nil {
		updates = appendOptionalStringUpdate(updates, "manager", patch.ManagerID)
	}
	if patch.WalletAddress != nil {
		updates = appendOptionalStringUpdate(updates, "walletAddress", patch.WalletAddress)
	}
	if patch.CreatedBy != nil {
		updates = appendOptionalStringUpdate(updates, "createdBy", patch.CreatedBy)
	}
	if patch.UpdatedAt != nil {
		if patch.UpdatedAt.IsZero() {
			updates = append(updates, firestore.Update{Path: "updatedAt", Value: nil})
		} else {
			updates = append(updates, firestore.Update{Path: "updatedAt", Value: patch.UpdatedAt.UTC()})
		}
	}
	if patch.UpdatedBy != nil {
		updates = appendOptionalStringUpdate(updates, "updatedBy", patch.UpdatedBy)
	}

	if len(updates) == 0 {
		snap, err := ref.Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return branddom.Brand{}, branddom.ErrNotFound
			}
			return branddom.Brand{}, err
		}
		return r.docToDomain(snap)
	}

	hasUpdatedAt := false
	for _, update := range updates {
		if update.Path == "updatedAt" {
			hasUpdatedAt = true
			break
		}
	}

	if !hasUpdatedAt {
		updates = append(updates, firestore.Update{Path: "updatedAt", Value: time.Now().UTC()})
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return branddom.Brand{}, branddom.ErrNotFound
		}
		return branddom.Brand{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return branddom.Brand{}, err
	}

	return r.docToDomain(snap)
}

func (r *BrandRepositoryFS) Delete(ctx context.Context, id string) error {
	if id == "" {
		return branddom.ErrNotFound
	}

	ref := r.col().Doc(id)

	_, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return branddom.ErrNotFound
	}
	if err != nil {
		return err
	}

	_, err = ref.Delete(ctx)
	return err
}

func (r *BrandRepositoryFS) docToDomain(doc *firestore.DocumentSnapshot) (branddom.Brand, error) {
	var raw struct {
		CompanyID            string     `firestore:"companyId"`
		AccountID            string     `firestore:"accountId"`
		Name                 string     `firestore:"name"`
		Description          string     `firestore:"description"`
		WebsiteURL           string     `firestore:"websiteUrl"`
		BrandIcon            string     `firestore:"brandIcon"`
		BrandBackgroundImage string     `firestore:"brandBackgroundImage"`
		IsActive             bool       `firestore:"isActive"`
		ManagerID            *string    `firestore:"manager"`
		WalletAddress        string     `firestore:"walletAddress"`
		CreatedAt            time.Time  `firestore:"createdAt"`
		CreatedBy            *string    `firestore:"createdBy"`
		UpdatedAt            *time.Time `firestore:"updatedAt"`
		UpdatedBy            *string    `firestore:"updatedBy"`
		DeletedAt            *time.Time `firestore:"deletedAt"`
		DeletedBy            *string    `firestore:"deletedBy"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return branddom.Brand{}, err
	}

	b := branddom.Brand{
		ID:                   doc.Ref.ID,
		CompanyID:            raw.CompanyID,
		AccountID:            raw.AccountID,
		Name:                 raw.Name,
		Description:          raw.Description,
		URL:                  raw.WebsiteURL,
		BrandIcon:            raw.BrandIcon,
		BrandBackgroundImage: raw.BrandBackgroundImage,
		IsActive:             raw.IsActive,
		ManagerID:            normalizeOptionalString(raw.ManagerID),
		WalletAddress:        raw.WalletAddress,
		CreatedAt:            raw.CreatedAt.UTC(),
		CreatedBy:            normalizeOptionalString(raw.CreatedBy),
		UpdatedBy:            normalizeOptionalString(raw.UpdatedBy),
	}

	if raw.UpdatedAt != nil && !raw.UpdatedAt.IsZero() {
		t := raw.UpdatedAt.UTC()
		b.UpdatedAt = &t
	}

	return b, nil
}

func (r *BrandRepositoryFS) domainToDocData(b branddom.Brand) map[string]any {
	data := map[string]any{
		"companyId":            b.CompanyID,
		"accountId":            b.AccountID,
		"name":                 b.Name,
		"description":          b.Description,
		"websiteUrl":           b.URL,
		"brandIcon":            b.BrandIcon,
		"brandBackgroundImage": b.BrandBackgroundImage,
		"isActive":             b.IsActive,
		"walletAddress":        b.WalletAddress,
		"createdAt":            b.CreatedAt.UTC(),
	}

	setOptionalString(data, "manager", b.ManagerID)
	setOptionalString(data, "createdBy", b.CreatedBy)
	if b.UpdatedAt != nil && !b.UpdatedAt.IsZero() {
		data["updatedAt"] = b.UpdatedAt.UTC()
	}
	setOptionalString(data, "updatedBy", b.UpdatedBy)

	return data
}

func ptrTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}

	tt := t.UTC()
	return &tt
}
