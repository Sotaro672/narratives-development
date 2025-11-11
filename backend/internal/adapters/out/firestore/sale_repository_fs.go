// backend/internal/adapters/out/firestore/sale_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	saledom "narratives/internal/domain/sale"
)

// ============================================================
// Firestore-based Sale Repository
// (Firestore implementation corresponding to SaleRepositoryPG)
// ============================================================

type SaleRepositoryFS struct {
	Client *firestore.Client
}

func NewSaleRepositoryFS(client *firestore.Client) *SaleRepositoryFS {
	return &SaleRepositoryFS{Client: client}
}

func (r *SaleRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("sales")
}

// ============================================================
// Facade methods matching usecase.SaleRepo
// ============================================================

// GetByID returns a Sale by document ID (value, not pointer).
func (r *SaleRepositoryFS) GetByID(ctx context.Context, id string) (saledom.Sale, error) {
	if r.Client == nil {
		return saledom.Sale{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return saledom.Sale{}, saledom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return saledom.Sale{}, saledom.ErrNotFound
	}
	if err != nil {
		return saledom.Sale{}, err
	}

	return docToSale(snap)
}

// Exists reports whether a Sale with the given ID exists.
func (r *SaleRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
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

// Create creates a new Sale document.
//
// Semantics aligned with PG版:
// - ID is generated (gen_random_uuid in PG; here Firestore auto-ID).
// - list_id (ListID) is required.
// - discount_id (DiscountID) is optional.
// - prices is []SalePrice, stored as array.
func (r *SaleRepositoryFS) Create(ctx context.Context, v saledom.Sale) (saledom.Sale, error) {
	if r.Client == nil {
		return saledom.Sale{}, errors.New("firestore client is nil")
	}

	// Firestore: generate ID
	ref := r.col().NewDoc()
	v.ID = ref.ID

	data := saleToDocData(v)

	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return saledom.Sale{}, saledom.ErrConflict
		}
		return saledom.Sale{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return saledom.Sale{}, err
	}
	return docToSale(snap)
}

// Save is upsert-like:
// - If v.ID == ""  -> Create
// - If v.ID != "" and not exists -> Create (like PG版: ignores provided ID there; here reuses it)
// - If exists      -> Update via UpdateSaleInput.
func (r *SaleRepositoryFS) Save(ctx context.Context, v saledom.Sale) (saledom.Sale, error) {
	if r.Client == nil {
		return saledom.Sale{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	if id == "" {
		// treat as new
		return r.Create(ctx, v)
	}

	exists, err := r.Exists(ctx, id)
	if err != nil {
		return saledom.Sale{}, err
	}

	if !exists {
		// Firestore版では指定IDで新規作成して問題ない想定。
		ref := r.col().Doc(id)
		v.ID = id
		data := saleToDocData(v)
		if _, err := ref.Create(ctx, data); err != nil {
			if status.Code(err) == codes.AlreadyExists {
				return saledom.Sale{}, saledom.ErrConflict
			}
			return saledom.Sale{}, err
		}
		snap, err := ref.Get(ctx)
		if err != nil {
			return saledom.Sale{}, err
		}
		return docToSale(snap)
	}

	// Map domain Sale -> UpdateSaleInput (same logic as PG版)
	patch := saledom.UpdateSaleInput{
		ListID: func(s string) *string {
			s = strings.TrimSpace(s)
			if s == "" {
				return nil
			}
			return &s
		}(v.ListID),
		DiscountID: func(p *string) *string {
			if p == nil {
				return nil
			}
			s := strings.TrimSpace(*p)
			return &s // empty string handled in Update
		}(v.DiscountID),
		Prices: func(prices []saledom.SalePrice) *[]saledom.SalePrice {
			cp := make([]saledom.SalePrice, len(prices))
			copy(cp, prices)
			return &cp
		}(v.Prices),
	}

	updated, err := r.Update(ctx, id, patch)
	if err != nil {
		return saledom.Sale{}, err
	}
	if updated == nil {
		return saledom.Sale{}, saledom.ErrNotFound
	}
	return *updated, nil
}

// Delete removes a Sale document (hard delete).
func (r *SaleRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return saledom.ErrNotFound
	}

	ref := r.col().Doc(id)
	_, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return saledom.ErrNotFound
	}
	if err != nil {
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

// Reset deletes all sales (for tests/dev), using Transactions instead of WriteBatch.
func (r *SaleRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var refs []*firestore.DocumentRef
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		refs = append(refs, doc.Ref)
	}

	if len(refs) == 0 {
		return nil
	}

	const chunkSize = 400
	for start := 0; start < len(refs); start += chunkSize {
		end := start + chunkSize
		if end > len(refs) {
			end = len(refs)
		}
		chunk := refs[start:end]

		if err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, ref := range chunk {
				if err := tx.Delete(ref); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

// ============================================================
// List / Count (Filter + Sort + Paging)
// ============================================================

func (r *SaleRepositoryFS) List(
	ctx context.Context,
	filter saledom.Filter,
	sort saledom.Sort,
	page saledom.Page,
) (saledom.PageResult, error) {
	if r.Client == nil {
		return saledom.PageResult{}, errors.New("firestore client is nil")
	}

	// Base query: filter mostly in-memory.
	q := r.col().Query
	q = applySaleOrderBy(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []saledom.Sale
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return saledom.PageResult{}, err
		}
		s, err := docToSale(doc)
		if err != nil {
			return saledom.PageResult{}, err
		}
		if matchSaleFilter(s, filter) {
			all = append(all, s)
		}
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	total := len(all)
	if total == 0 {
		return saledom.PageResult{
			Items:      []saledom.Sale{},
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

	return saledom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *SaleRepositoryFS) Count(ctx context.Context, filter saledom.Filter) (int, error) {
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
		s, err := docToSale(doc)
		if err != nil {
			return 0, err
		}
		if matchSaleFilter(s, filter) {
			total++
		}
	}
	return total, nil
}

// ============================================================
// Update (partial) - Firestore版 of PG's UPDATE ... RETURNING
// ============================================================

func (r *SaleRepositoryFS) Update(
	ctx context.Context,
	id string,
	in saledom.UpdateSaleInput,
) (*saledom.Sale, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, saledom.ErrNotFound
	}

	ref := r.col().Doc(id)

	// Ensure exists first
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return nil, saledom.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	var updates []firestore.Update

	if in.ListID != nil {
		v := strings.TrimSpace(*in.ListID)
		if v == "" {
			// listId は必須想定だが、念のため nil クリア。
			updates = append(updates, firestore.Update{Path: "listId", Value: nil})
		} else {
			updates = append(updates, firestore.Update{Path: "listId", Value: v})
		}
	}

	if in.DiscountID != nil {
		v := strings.TrimSpace(*in.DiscountID)
		if v == "" {
			// empty string => clear
			updates = append(updates, firestore.Update{Path: "discountId", Value: nil})
		} else {
			updates = append(updates, firestore.Update{Path: "discountId", Value: v})
		}
	}

	if in.Prices != nil {
		updates = append(updates, firestore.Update{
			Path:  "prices",
			Value: *in.Prices,
		})
	}

	if len(updates) == 0 {
		// no-op: just return current
		snap, err := ref.Get(ctx)
		if status.Code(err) == codes.NotFound {
			return nil, saledom.ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		s, err := docToSale(snap)
		if err != nil {
			return nil, err
		}
		return &s, nil
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, saledom.ErrNotFound
		}
		return nil, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, saledom.ErrNotFound
		}
		return nil, err
	}
	s, err := docToSale(snap)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ============================================================
// Mapping Helpers
// ============================================================

func docToSale(doc *firestore.DocumentSnapshot) (saledom.Sale, error) {
	var raw struct {
		ListID     string              `firestore:"listId"`
		DiscountID *string             `firestore:"discountId"`
		Prices     []saledom.SalePrice `firestore:"prices"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return saledom.Sale{}, err
	}

	var discountID *string
	if raw.DiscountID != nil {
		if s := strings.TrimSpace(*raw.DiscountID); s != "" {
			discountID = &s
		}
	}

	return saledom.Sale{
		ID:         strings.TrimSpace(doc.Ref.ID),
		ListID:     strings.TrimSpace(raw.ListID),
		DiscountID: discountID,
		Prices:     raw.Prices,
	}, nil
}

func saleToDocData(v saledom.Sale) map[string]any {
	data := map[string]any{
		"listId": strings.TrimSpace(v.ListID),
		"prices": v.Prices,
	}

	if v.DiscountID != nil {
		if s := strings.TrimSpace(*v.DiscountID); s != "" {
			data["discountId"] = s
		}
	}

	return data
}

// ============================================================
// Filter / Sort Helpers (Firestore analogue of buildSaleWhere/orderBy)
// ============================================================

// matchSaleFilter applies saledom.Filter in-memory.
func matchSaleFilter(s saledom.Sale, f saledom.Filter) bool {
	trim := func(x string) string { return strings.TrimSpace(x) }

	if v := trim(f.ID); v != "" && trim(s.ID) != v {
		return false
	}
	if v := trim(f.ListID); v != "" && trim(s.ListID) != v {
		return false
	}

	if f.HasDiscount != nil {
		if *f.HasDiscount {
			if s.DiscountID == nil || trim(*s.DiscountID) == "" {
				return false
			}
		} else {
			if s.DiscountID != nil && trim(*s.DiscountID) != "" {
				return false
			}
		}
	}

	// ModelNumber inside prices
	if v := trim(f.ModelNumber); v != "" {
		found := false
		for _, p := range s.Prices {
			if trim(p.ModelNumber) == v {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// MinAnyPrice / MaxAnyPrice across any price entry
	if f.MinAnyPrice != nil {
		ok := false
		for _, p := range s.Prices {
			if p.Price >= *f.MinAnyPrice {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if f.MaxAnyPrice != nil {
		ok := false
		for _, p := range s.Prices {
			if p.Price <= *f.MaxAnyPrice {
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

// applySaleOrderBy maps saledom.Sort to Firestore orderBy.
func applySaleOrderBy(q firestore.Query, s saledom.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	var field string

	switch col {
	case "id":
		field = firestore.DocumentID
	case "listid", "list_id":
		field = "listId"
	default:
		// default: ORDER BY id ASC
		return q.OrderBy(firestore.DocumentID, firestore.Asc)
	}

	dir := firestore.Asc
	if strings.EqualFold(string(s.Order), "desc") {
		dir = firestore.Desc
	}

	// tie-break by DocumentID for stable ordering when field != ID
	if field == firestore.DocumentID {
		return q.OrderBy(field, dir)
	}
	return q.OrderBy(field, dir).
		OrderBy(firestore.DocumentID, dir)
}

// ============================================================
// (Optional) Compile-time check (uncomment if you have the interface)
// ============================================================
// var _ saledom.Repository = (*SaleRepositoryFS)(nil)
