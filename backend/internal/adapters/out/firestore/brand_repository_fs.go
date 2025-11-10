// backend/internal/adapters/out/firestore/brand_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	branddom "narratives/internal/domain/brand"
)

// ========================================
// Firestore Repository Implementation
// ========================================

type BrandRepositoryFS struct {
	Client *firestore.Client
}

func NewBrandRepositoryFS(client *firestore.Client) *BrandRepositoryFS {
	return &BrandRepositoryFS{Client: client}
}

func (r *BrandRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("brands")
}

// Ensure interface implementation
var _ branddom.Repository = (*BrandRepositoryFS)(nil)

// ========================================
// Create
// ========================================

func (r *BrandRepositoryFS) Create(ctx context.Context, b branddom.Brand) (branddom.Brand, error) {
	now := time.Now().UTC()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	if b.UpdatedAt == nil || b.UpdatedAt.IsZero() {
		b.UpdatedAt = ptrTime(b.CreatedAt)
	}

	// Firestore: generate ID if empty
	var ref *firestore.DocumentRef
	if strings.TrimSpace(b.ID) == "" {
		ref = r.col().NewDoc()
		b.ID = ref.ID
	} else {
		ref = r.col().Doc(b.ID)
	}

	// DeletedAt/DeletedBy: keep as-is (may be nil)
	data := r.domainToDocData(b)

	if _, err := ref.Create(ctx, data); err != nil {
		// If already exists, surface conflict-ish error
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

// ========================================
// GetByID
// ========================================

func (r *BrandRepositoryFS) GetByID(ctx context.Context, id string) (branddom.Brand, error) {
	id = strings.TrimSpace(id)
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

// ========================================
// Exists
// ========================================

func (r *BrandRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
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

// ========================================
// Count
// ========================================

func (r *BrandRepositoryFS) Count(ctx context.Context, filter branddom.Filter) (int, error) {
	q := r.col().Query
	q = applyBrandFilterToQuery(q, filter)

	iter := q.Documents(ctx)
	defer iter.Stop()

	count := 0
	for {
		_, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}
		count++
	}
	return count, nil
}

// ========================================
// Update (partial)
// ========================================

func (r *BrandRepositoryFS) Update(ctx context.Context, id string, patch branddom.BrandPatch) (branddom.Brand, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return branddom.Brand{}, branddom.ErrNotFound
	}
	ref := r.col().Doc(id)

	// ensure exists
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return branddom.Brand{}, branddom.ErrNotFound
	} else if err != nil {
		return branddom.Brand{}, err
	}

	var updates []firestore.Update

	if patch.CompanyID != nil {
		updates = append(updates, firestore.Update{
			Path:  "companyId",
			Value: strings.TrimSpace(*patch.CompanyID),
		})
	}
	if patch.Name != nil {
		updates = append(updates, firestore.Update{
			Path:  "name",
			Value: strings.TrimSpace(*patch.Name),
		})
	}
	if patch.Description != nil {
		updates = append(updates, firestore.Update{
			Path:  "description",
			Value: optionalStringValue(patch.Description),
		})
	}
	if patch.URL != nil {
		updates = append(updates, firestore.Update{
			Path:  "websiteUrl",
			Value: optionalStringValue(patch.URL),
		})
	}
	if patch.IsActive != nil {
		updates = append(updates, firestore.Update{
			Path:  "isActive",
			Value: *patch.IsActive,
		})
	}
	if patch.ManagerID != nil {
		updates = append(updates, firestore.Update{
			Path:  "managerId",
			Value: optionalStringValue(patch.ManagerID),
		})
	}
	if patch.WalletAddress != nil {
		updates = append(updates, firestore.Update{
			Path:  "walletAddress",
			Value: optionalStringValue(patch.WalletAddress),
		})
	}
	if patch.CreatedBy != nil {
		updates = append(updates, firestore.Update{
			Path:  "createdBy",
			Value: optionalStringValue(patch.CreatedBy),
		})
	}
	if patch.UpdatedAt != nil {
		// if zero => clear, else set
		if patch.UpdatedAt.IsZero() {
			updates = append(updates, firestore.Update{
				Path:  "updatedAt",
				Value: nil,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "updatedAt",
				Value: patch.UpdatedAt.UTC(),
			})
		}
	}
	if patch.UpdatedBy != nil {
		updates = append(updates, firestore.Update{
			Path:  "updatedBy",
			Value: optionalStringValue(patch.UpdatedBy),
		})
	}
	if patch.DeletedAt != nil {
		if patch.DeletedAt.IsZero() {
			updates = append(updates, firestore.Update{
				Path:  "deletedAt",
				Value: nil,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "deletedAt",
				Value: patch.DeletedAt.UTC(),
			})
		}
	}
	if patch.DeletedBy != nil {
		updates = append(updates, firestore.Update{
			Path:  "deletedBy",
			Value: optionalStringValue(patch.DeletedBy),
		})
	}

	// if nothing to update, just return current
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

	// always bump updatedAt if not explicitly controlled
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

// ========================================
// Delete (hard delete)
// ========================================

func (r *BrandRepositoryFS) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
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

// ========================================
// List (filter/sort/pagination)
// ========================================

func (r *BrandRepositoryFS) List(
	ctx context.Context,
	filter branddom.Filter,
	sort branddom.Sort,
	page branddom.Page,
) (branddom.PageResult[branddom.Brand], error) {

	q := r.col().Query
	q = applyBrandFilterToQuery(q, filter)

	// sort
	field, dir := mapBrandSort(sort)
	q = q.OrderBy(field, dir).OrderBy("id", firestore.Asc) // secondary stable sort

	// paging (offset-based; simple)
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
	offset := (number - 1) * perPage

	if offset > 0 {
		q = q.Offset(offset)
	}
	q = q.Limit(perPage)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var items []branddom.Brand
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

	// NOTE:
	// 正確な TotalCount / TotalPages を出すには COUNT クエリや別集計が必要。
	// ここでは簡易に「取得件数 = TotalCount」として返し、ページ情報は呼び出し側で調整してください。
	totalCount := len(items)

	return branddom.PageResult[branddom.Brand]{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: number, // 厳密ではないがインターフェース互換のため設定
		Page:       number,
		PerPage:    perPage,
	}, nil
}

// ========================================
// ListByCursor (not implemented for now)
// ========================================

func (r *BrandRepositoryFS) ListByCursor(
	_ context.Context,
	_ branddom.Filter,
	_ branddom.Sort,
	_ branddom.CursorPage,
) (branddom.CursorPageResult[branddom.Brand], error) {
	return branddom.CursorPageResult[branddom.Brand]{}, errors.New("ListByCursor not implemented for Firestore")
}

// ========================================
// Save (Upsert)
// ========================================

func (r *BrandRepositoryFS) Save(ctx context.Context, b branddom.Brand, _ *branddom.SaveOptions) (branddom.Brand, error) {
	// If no ID, new doc; else upsert existing doc.
	now := time.Now().UTC()

	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	if b.UpdatedAt == nil || b.UpdatedAt.IsZero() {
		b.UpdatedAt = ptrTime(now)
	}

	var ref *firestore.DocumentRef
	if strings.TrimSpace(b.ID) == "" {
		ref = r.col().NewDoc()
		b.ID = ref.ID
	} else {
		ref = r.col().Doc(b.ID)
	}

	data := r.domainToDocData(b)

	_, err := ref.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return branddom.Brand{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return branddom.Brand{}, err
	}
	return r.docToDomain(snap)
}

// ========================================
// Reset (development/testing)
// ========================================

func (r *BrandRepositoryFS) Reset(ctx context.Context) error {
	iter := r.col().Documents(ctx)
	batch := r.Client.Batch()
	count := 0

	for {
		doc, err := iter.Next()
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
	log.Printf("[firestore] Reset brands: deleted %d docs\n", count)
	return nil
}

// ========================================
// Mapping Helpers
// ========================================

func (r *BrandRepositoryFS) docToDomain(doc *firestore.DocumentSnapshot) (branddom.Brand, error) {
	var raw struct {
		CompanyID     string     `firestore:"companyId"`
		Name          string     `firestore:"name"`
		Description   string     `firestore:"description"`
		WebsiteURL    string     `firestore:"websiteUrl"`
		IsActive      bool       `firestore:"isActive"`
		ManagerID     *string    `firestore:"managerId"`
		WalletAddress string     `firestore:"walletAddress"`
		CreatedAt     time.Time  `firestore:"createdAt"`
		CreatedBy     *string    `firestore:"createdBy"`
		UpdatedAt     *time.Time `firestore:"updatedAt"`
		UpdatedBy     *string    `firestore:"updatedBy"`
		DeletedAt     *time.Time `firestore:"deletedAt"`
		DeletedBy     *string    `firestore:"deletedBy"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return branddom.Brand{}, err
	}

	b := branddom.Brand{
		ID:            doc.Ref.ID,
		CompanyID:     strings.TrimSpace(raw.CompanyID),
		Name:          strings.TrimSpace(raw.Name),
		Description:   strings.TrimSpace(raw.Description),
		URL:           strings.TrimSpace(raw.WebsiteURL),
		IsActive:      raw.IsActive,
		ManagerID:     trimPtr(raw.ManagerID),
		WalletAddress: strings.TrimSpace(raw.WalletAddress),
		CreatedAt:     raw.CreatedAt.UTC(),
		CreatedBy:     trimPtr(raw.CreatedBy),
		UpdatedBy:     trimPtr(raw.UpdatedBy),
		DeletedBy:     trimPtr(raw.DeletedBy),
	}

	if raw.UpdatedAt != nil && !raw.UpdatedAt.IsZero() {
		t := raw.UpdatedAt.UTC()
		b.UpdatedAt = &t
	}
	if raw.DeletedAt != nil && !raw.DeletedAt.IsZero() {
		t := raw.DeletedAt.UTC()
		b.DeletedAt = &t
	}

	return b, nil
}

func (r *BrandRepositoryFS) domainToDocData(b branddom.Brand) map[string]any {
	data := map[string]any{
		"companyId":     strings.TrimSpace(b.CompanyID),
		"name":          strings.TrimSpace(b.Name),
		"description":   strings.TrimSpace(b.Description),
		"websiteUrl":    strings.TrimSpace(b.URL),
		"isActive":      b.IsActive,
		"walletAddress": strings.TrimSpace(b.WalletAddress),
		"createdAt":     b.CreatedAt.UTC(),
	}

	if b.ManagerID != nil && strings.TrimSpace(*b.ManagerID) != "" {
		data["managerId"] = strings.TrimSpace(*b.ManagerID)
	}
	if b.CreatedBy != nil && strings.TrimSpace(*b.CreatedBy) != "" {
		data["createdBy"] = strings.TrimSpace(*b.CreatedBy)
	}
	if b.UpdatedAt != nil && !b.UpdatedAt.IsZero() {
		data["updatedAt"] = b.UpdatedAt.UTC()
	}
	if b.UpdatedBy != nil && strings.TrimSpace(*b.UpdatedBy) != "" {
		data["updatedBy"] = strings.TrimSpace(*b.UpdatedBy)
	}
	if b.DeletedAt != nil && !b.DeletedAt.IsZero() {
		data["deletedAt"] = b.DeletedAt.UTC()
	}
	if b.DeletedBy != nil && strings.TrimSpace(*b.DeletedBy) != "" {
		data["deletedBy"] = strings.TrimSpace(*b.DeletedBy)
	}

	return data
}

// ========================================
// Query / Sort Helpers
// ========================================

func applyBrandFilterToQuery(q firestore.Query, f branddom.Filter) firestore.Query {
	// NOTE:
	// Firestore のクエリ制約により、複雑な AND/OR, 部分一致, 多数の IN は制限されます。
	// ここでは代表的な条件のみをサポートし、残りは呼び出し側で絞り込み想定。

	// CompanyID
	if f.CompanyID != nil && strings.TrimSpace(*f.CompanyID) != "" {
		q = q.Where("companyId", "==", strings.TrimSpace(*f.CompanyID))
	}
	// ManagerID
	if f.ManagerID != nil && strings.TrimSpace(*f.ManagerID) != "" {
		q = q.Where("managerId", "==", strings.TrimSpace(*f.ManagerID))
	}
	// IsActive
	if f.IsActive != nil {
		q = q.Where("isActive", "==", *f.IsActive)
	}
	// WalletAddress
	if f.WalletAddress != nil && strings.TrimSpace(*f.WalletAddress) != "" {
		q = q.Where("walletAddress", "==", strings.TrimSpace(*f.WalletAddress))
	}
	// Deleted flag
	if f.Deleted != nil {
		if *f.Deleted {
			// 削除済のみ: deletedAt != nil 的な表現はクエリで難しいため、
			// deletedAt フィールド存在チェックなどの別設計が必要。
			// ここでは簡易に deletedAt > 0 を想定する場合など、要件に応じて調整。
		} else {
			// 未削除のみ: deletedAt == nil を明示するには設計が必要。
		}
	}

	// SearchQuery, 日付レンジなどは必要に応じて
	// インデックス設計 or アプリ側フィルタで対応してください。

	return q
}

func mapBrandSort(s branddom.Sort) (field string, dir firestore.Direction) {
	col := strings.ToLower(strings.TrimSpace(s.Column))
	switch col {
	case "name":
		field = "name"
	case "is_active", "isactive":
		field = "isActive"
	case "updated_at", "updatedat":
		field = "updatedAt"
	case "created_at", "createdat":
		field = "createdAt"
	default:
		field = "createdAt"
	}

	if strings.EqualFold(string(s.Order), "asc") {
		dir = firestore.Asc
	} else {
		dir = firestore.Desc
	}
	return
}

// ========================================
// Small utilities
// ========================================

func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

func optionalStringValue(p *string) any {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return s
}

func ptrTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	tt := t.UTC()
	return &tt
}
