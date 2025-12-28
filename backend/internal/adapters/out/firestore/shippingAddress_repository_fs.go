// backend/internal/adapters/out/firestore/shippingAddress_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	shipdom "narratives/internal/domain/shippingAddress"
)

// ============================================================
// Firestore-based ShippingAddress Repository
// ============================================================

type ShippingAddressRepositoryFS struct {
	Client *firestore.Client
}

func NewShippingAddressRepositoryFS(client *firestore.Client) *ShippingAddressRepositoryFS {
	return &ShippingAddressRepositoryFS{Client: client}
}

func (r *ShippingAddressRepositoryFS) col() *firestore.CollectionRef {
	// ✅ 期待値: shippingAddresses コレクション
	return r.Client.Collection("shippingAddresses")
}

// ============================================================
// Facade (usecase port)
// ============================================================

func (r *ShippingAddressRepositoryFS) GetByID(ctx context.Context, id string) (shipdom.ShippingAddress, error) {
	if r.Client == nil {
		return shipdom.ShippingAddress{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return shipdom.ShippingAddress{}, shipdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return shipdom.ShippingAddress{}, shipdom.ErrNotFound
	}
	if err != nil {
		return shipdom.ShippingAddress{}, err
	}

	return docToShippingAddress(snap)
}

func (r *ShippingAddressRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
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

func (r *ShippingAddressRepositoryFS) Create(ctx context.Context, v shipdom.ShippingAddress) (shipdom.ShippingAddress, error) {
	if r.Client == nil {
		return shipdom.ShippingAddress{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = v.CreatedAt
	}

	// Firestore: generate ID if empty, otherwise use provided ID.
	var ref *firestore.DocumentRef
	id := strings.TrimSpace(v.ID)
	if id == "" {
		ref = r.col().NewDoc()
		v.ID = ref.ID
	} else {
		ref = r.col().Doc(id)
		v.ID = id
	}

	// ✅ UI入力が無い場合は実装側でJP
	if strings.TrimSpace(v.Country) == "" {
		v.Country = "JP"
	}

	data := shippingAddressToDocData(v)

	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return shipdom.ShippingAddress{}, shipdom.ErrConflict
		}
		return shipdom.ShippingAddress{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return shipdom.ShippingAddress{}, err
	}
	return docToShippingAddress(snap)
}

func (r *ShippingAddressRepositoryFS) Save(ctx context.Context, v shipdom.ShippingAddress) (shipdom.ShippingAddress, error) {
	if r.Client == nil {
		return shipdom.ShippingAddress{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	if id == "" {
		return r.Create(ctx, v)
	}

	ref := r.col().Doc(id)

	snap, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		now := time.Now().UTC()
		if v.CreatedAt.IsZero() {
			v.CreatedAt = now
		}
		if v.UpdatedAt.IsZero() {
			v.UpdatedAt = v.CreatedAt
		}
		v.ID = id
		if strings.TrimSpace(v.Country) == "" {
			v.Country = "JP"
		}
		data := shippingAddressToDocData(v)
		if _, err := ref.Create(ctx, data); err != nil {
			if status.Code(err) == codes.AlreadyExists {
				return shipdom.ShippingAddress{}, shipdom.ErrConflict
			}
			return shipdom.ShippingAddress{}, err
		}
		snap, err := ref.Get(ctx)
		if err != nil {
			return shipdom.ShippingAddress{}, err
		}
		return docToShippingAddress(snap)
	}
	if err != nil {
		return shipdom.ShippingAddress{}, err
	}

	current, err := docToShippingAddress(snap)
	if err != nil {
		return shipdom.ShippingAddress{}, err
	}

	createdAt := current.CreatedAt
	if !v.CreatedAt.IsZero() {
		createdAt = v.CreatedAt.UTC()
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	updatedAt := time.Now().UTC()
	if !v.UpdatedAt.IsZero() && v.UpdatedAt.After(createdAt) {
		updatedAt = v.UpdatedAt.UTC()
	}

	country := strings.TrimSpace(v.Country)
	if country == "" {
		country = strings.TrimSpace(current.Country)
	}
	if country == "" {
		country = "JP"
	}

	next := shipdom.ShippingAddress{
		ID:        id,
		UserID:    pickNonEmpty(v.UserID, current.UserID),
		ZipCode:   pickNonEmpty(v.ZipCode, current.ZipCode),
		State:     pickNonEmpty(v.State, current.State),
		City:      pickNonEmpty(v.City, current.City),
		Street:    pickNonEmpty(v.Street, current.Street),
		Street2:   pickStreet2(v.Street2),
		Country:   country,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	data := shippingAddressToDocData(next)

	if _, err := ref.Set(ctx, data, firestore.MergeAll); err != nil {
		if status.Code(err) == codes.NotFound {
			return shipdom.ShippingAddress{}, shipdom.ErrNotFound
		}
		if status.Code(err) == codes.AlreadyExists {
			return shipdom.ShippingAddress{}, shipdom.ErrConflict
		}
		return shipdom.ShippingAddress{}, err
	}

	snap, err = ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return shipdom.ShippingAddress{}, shipdom.ErrNotFound
		}
		return shipdom.ShippingAddress{}, err
	}

	return docToShippingAddress(snap)
}

func (r *ShippingAddressRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return shipdom.ErrNotFound
	}

	ref := r.col().Doc(id)
	_, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return shipdom.ErrNotFound
	}
	if err != nil {
		return err
	}

	_, err = ref.Delete(ctx)
	return err
}

func (r *ShippingAddressRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var snaps []*firestore.DocumentSnapshot
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		snaps = append(snaps, doc)
	}

	const chunkSize = 400
	for i := 0; i < len(snaps); i += chunkSize {
		end := i + chunkSize
		if end > len(snaps) {
			end = len(snaps)
		}
		if err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, s := range snaps[i:end] {
				if err := tx.Delete(s.Ref); err != nil {
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

func (r *ShippingAddressRepositoryFS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	return fn(ctx)
}

// ============================================================
// List / Count
// ============================================================

func (r *ShippingAddressRepositoryFS) List(
	ctx context.Context,
	filter shipdom.Filter,
	sortOpt shipdom.Sort,
	page shipdom.Page,
) (shipdom.PageResult, error) {
	if r.Client == nil {
		return shipdom.PageResult{}, errors.New("firestore client is nil")
	}

	q := r.col().Query
	q = applyAddrOrderByFS(q, sortOpt)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []shipdom.ShippingAddress
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return shipdom.PageResult{}, err
		}
		a, err := docToShippingAddress(doc)
		if err != nil {
			return shipdom.PageResult{}, err
		}
		if matchAddrFilter(a, filter) {
			all = append(all, a)
		}
	}

	sortShippingAddresses(all, sortOpt)

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)
	total := len(all)

	if total == 0 {
		return shipdom.PageResult{
			Items:      []shipdom.ShippingAddress{},
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

	return shipdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *ShippingAddressRepositoryFS) Count(ctx context.Context, filter shipdom.Filter) (int, error) {
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
		a, err := docToShippingAddress(doc)
		if err != nil {
			return 0, err
		}
		if matchAddrFilter(a, filter) {
			total++
		}
	}
	return total, nil
}

// ============================================================
// Mapping
// ============================================================

func docToShippingAddress(doc *firestore.DocumentSnapshot) (shipdom.ShippingAddress, error) {
	var raw struct {
		UserID    string    `firestore:"userId"`
		ZipCode   string    `firestore:"zipCode"`
		State     string    `firestore:"state"`
		City      string    `firestore:"city"`
		Street    string    `firestore:"street"`
		Street2   string    `firestore:"street2"`
		Country   string    `firestore:"country"`
		CreatedAt time.Time `firestore:"createdAt"`
		UpdatedAt time.Time `firestore:"updatedAt"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return shipdom.ShippingAddress{}, err
	}

	createdAt := raw.CreatedAt.UTC()
	updatedAt := raw.UpdatedAt.UTC()
	if raw.CreatedAt.IsZero() {
		createdAt = time.Time{}
	}
	if raw.UpdatedAt.IsZero() {
		updatedAt = createdAt
	}

	return shipdom.ShippingAddress{
		ID:        strings.TrimSpace(doc.Ref.ID),
		UserID:    strings.TrimSpace(raw.UserID),
		ZipCode:   strings.TrimSpace(raw.ZipCode),
		State:     strings.TrimSpace(raw.State),
		City:      strings.TrimSpace(raw.City),
		Street:    strings.TrimSpace(raw.Street),
		Street2:   strings.TrimSpace(raw.Street2),
		Country:   strings.TrimSpace(raw.Country),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func shippingAddressToDocData(v shipdom.ShippingAddress) map[string]any {
	data := map[string]any{
		"userId":    strings.TrimSpace(v.UserID),
		"zipCode":   strings.TrimSpace(v.ZipCode),
		"state":     strings.TrimSpace(v.State),
		"city":      strings.TrimSpace(v.City),
		"street":    strings.TrimSpace(v.Street),
		"street2":   strings.TrimSpace(v.Street2),
		"country":   strings.TrimSpace(v.Country),
		"createdAt": v.CreatedAt.UTC(),
		"updatedAt": v.UpdatedAt.UTC(),
	}

	if v.CreatedAt.IsZero() {
		delete(data, "createdAt")
	}
	if v.UpdatedAt.IsZero() {
		delete(data, "updatedAt")
	}

	return data
}

// ============================================================
// Filter / Sort
// ============================================================

func matchAddrFilter(a shipdom.ShippingAddress, f shipdom.Filter) bool {
	trim := func(s string) string { return strings.TrimSpace(s) }

	if v := trim(f.ID); v != "" && trim(a.ID) != v {
		return false
	}
	if v := trim(f.UserID); v != "" && trim(a.UserID) != v {
		return false
	}
	if v := trim(f.City); v != "" && !strings.EqualFold(trim(a.City), v) {
		return false
	}
	if v := trim(f.State); v != "" && !strings.EqualFold(trim(a.State), v) {
		return false
	}
	if v := trim(f.ZipCode); v != "" && trim(a.ZipCode) != v {
		return false
	}
	if v := trim(f.Country); v != "" && !strings.EqualFold(trim(a.Country), v) {
		return false
	}

	if f.CreatedFrom != nil && !a.CreatedAt.IsZero() && a.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && !a.CreatedAt.IsZero() && !a.CreatedAt.Before(f.CreatedTo.UTC()) {
		return false
	}
	if f.UpdatedFrom != nil && !a.UpdatedAt.IsZero() && a.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
		return false
	}
	if f.UpdatedTo != nil && !a.UpdatedAt.IsZero() && !a.UpdatedAt.Before(f.UpdatedTo.UTC()) {
		return false
	}

	return true
}

func applyAddrOrderByFS(q firestore.Query, s shipdom.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	var field string

	switch col {
	case "id":
		field = firestore.DocumentID
	case "createdat", "created_at":
		field = "createdAt"
	case "updatedat", "updated_at":
		field = "updatedAt"
	case "city":
		field = "city"
	case "state":
		field = "state"
	case "zipcode", "zip_code":
		field = "zipCode"
	default:
		return q.OrderBy("updatedAt", firestore.Desc).
			OrderBy(firestore.DocumentID, firestore.Desc)
	}

	dir := firestore.Desc
	if strings.EqualFold(string(s.Order), "asc") {
		dir = firestore.Asc
	}

	if field == firestore.DocumentID {
		return q.OrderBy(field, dir)
	}
	return q.OrderBy(field, dir).
		OrderBy(firestore.DocumentID, dir)
}

func sortShippingAddresses(items []shipdom.ShippingAddress, s shipdom.Sort) {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	asc := dir == "ASC"

	less := func(i, j int) bool {
		a := items[i]
		b := items[j]

		switch col {
		case "createdat", "created_at":
			if a.CreatedAt.Equal(b.CreatedAt) {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return a.CreatedAt.Before(b.CreatedAt)
			}
			return a.CreatedAt.After(b.CreatedAt)

		case "updatedat", "updated_at":
			if a.UpdatedAt.Equal(b.UpdatedAt) {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return a.UpdatedAt.Before(b.UpdatedAt)
			}
			return a.UpdatedAt.After(b.UpdatedAt)

		case "city":
			if a.City == b.City {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return a.City < b.City
			}
			return a.City > b.City

		case "state":
			if a.State == b.State {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return a.State < b.State
			}
			return a.State > b.State

		case "zipcode", "zip_code":
			if a.ZipCode == b.ZipCode {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return a.ZipCode < b.ZipCode
			}
			return a.ZipCode > b.ZipCode

		default:
			if a.UpdatedAt.Equal(b.UpdatedAt) {
				return a.ID > b.ID
			}
			return a.UpdatedAt.After(b.UpdatedAt)
		}
	}

	sort.SliceStable(items, less)
}

// ============================================================
// Small helpers
// ============================================================

func pickNonEmpty(a, b string) string {
	aa := strings.TrimSpace(a)
	if aa != "" {
		return aa
	}
	return strings.TrimSpace(b)
}

// Street2 は任意。Save(v) は「エンティティを正として保存」なので v を採用。
// （空文字なら空文字で保存＝削除扱い）
func pickStreet2(v string) string {
	return strings.TrimSpace(v)
}
