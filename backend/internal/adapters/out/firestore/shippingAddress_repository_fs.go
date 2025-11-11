// backend/internal/adapters/out/firestore/shippingAddress_repository_fs.go
package firestore

import (
	"context"
	"errors"
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
// (Firestore implementation corresponding to ShippingAddressRepositoryPG)
// ============================================================

type ShippingAddressRepositoryFS struct {
	Client *firestore.Client
}

func NewShippingAddressRepositoryFS(client *firestore.Client) *ShippingAddressRepositoryFS {
	return &ShippingAddressRepositoryFS{Client: client}
}

func (r *ShippingAddressRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("shipping_addresses")
}

// ============================================================
// Facade to satisfy usecase.ShippingAddressRepo
// ============================================================

// GetByID returns value (not pointer) to match usecase interface.
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

// Exists checks if an address with given ID exists.
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

// Create inserts using full domain entity and returns the created document.
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

// Save provides an upsert-like behavior:
// - if v.ID == ""           -> Create
// - if v.ID exists          -> Update
// - if v.ID doesn't exist   -> Create new with that ID
func (r *ShippingAddressRepositoryFS) Save(ctx context.Context, v shipdom.ShippingAddress) (shipdom.ShippingAddress, error) {
	if r.Client == nil {
		return shipdom.ShippingAddress{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	if id == "" {
		return r.Create(ctx, v)
	}

	exists, err := r.Exists(ctx, id)
	if err != nil {
		return shipdom.ShippingAddress{}, err
	}
	if !exists {
		// Create new with given ID
		now := time.Now().UTC()
		if v.CreatedAt.IsZero() {
			v.CreatedAt = now
		}
		if v.UpdatedAt.IsZero() {
			v.UpdatedAt = now
		}
		ref := r.col().Doc(id)
		v.ID = id
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

	// exists -> Update using UpdateShippingAddressInput (similar semantics to PG版)
	patch := shipdom.UpdateShippingAddressInput{
		AddressLine1: optString(v.Street),
		// AddressLine2 omitted; no separate field in domain ShippingAddress.
		City:       optString(v.City),
		Prefecture: optString(v.State),
		PostalCode: optString(v.ZipCode),
		Country:    optString(v.Country),
	}

	updated, err := r.updateInternal(ctx, id, patch)
	if err != nil {
		return shipdom.ShippingAddress{}, err
	}
	return updated, nil
}

// Delete performs hard delete.
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

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

// Reset deletes all shipping address documents (for tests/dev).
func (r *ShippingAddressRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
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

	// OLD (deprecated):
	// b := r.Client.Batch()
	// for i, snap := range snaps {
	//   b.Delete(snap.Ref)
	//   if (i+1)%400 == 0 {
	//     if _, err := b.Commit(ctx); err != nil { return err }
	//     b = r.Client.Batch()
	//   }
	// }
	// if _, err := b.Commit(ctx); err != nil { return err }

	// NEW (transaction, chunked):
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

// ============================================================
// List / Count (Filter + Sort + Paging)
// ============================================================

func (r *ShippingAddressRepositoryFS) List(
	ctx context.Context,
	filter shipdom.Filter,
	sort shipdom.Sort,
	page shipdom.Page,
) (shipdom.PageResult, error) {
	if r.Client == nil {
		return shipdom.PageResult{}, errors.New("firestore client is nil")
	}

	q := r.col().Query
	q = applyAddrOrderByFS(q, sort)

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
// Internal Update logic (Firestore equivalent of updateInternal in PG版)
// ============================================================

func (r *ShippingAddressRepositoryFS) updateInternal(
	ctx context.Context,
	id string,
	in shipdom.UpdateShippingAddressInput,
) (shipdom.ShippingAddress, error) {
	if r.Client == nil {
		return shipdom.ShippingAddress{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return shipdom.ShippingAddress{}, shipdom.ErrNotFound
	}

	ref := r.col().Doc(id)
	snap, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return shipdom.ShippingAddress{}, shipdom.ErrNotFound
	}
	if err != nil {
		return shipdom.ShippingAddress{}, err
	}

	current, err := docToShippingAddress(snap)
	if err != nil {
		return shipdom.ShippingAddress{}, err
	}

	// Apply patch to current (similar semantics to PG版)
	if in.AddressLine1 != nil {
		current.Street = strings.TrimSpace(*in.AddressLine1)
	}
	if in.AddressLine2 != nil {
		line2 := strings.TrimSpace(*in.AddressLine2)
		if line2 != "" {
			if current.Street == "" {
				current.Street = line2
			} else {
				current.Street = strings.TrimSpace(current.Street + " " + line2)
			}
		}
	}
	if in.City != nil {
		current.City = strings.TrimSpace(*in.City)
	}
	if in.Prefecture != nil {
		current.State = strings.TrimSpace(*in.Prefecture)
	}
	if in.PostalCode != nil {
		current.ZipCode = strings.TrimSpace(*in.PostalCode)
	}
	if in.Country != nil {
		current.Country = strings.TrimSpace(*in.Country)
	}

	// Always bump UpdatedAt
	current.UpdatedAt = time.Now().UTC()

	data := shippingAddressToDocData(current)

	if _, err := ref.Set(ctx, data, firestore.MergeAll); err != nil {
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

// ============================================================
// Mapping Helpers
// ============================================================

func docToShippingAddress(doc *firestore.DocumentSnapshot) (shipdom.ShippingAddress, error) {
	var raw struct {
		UserID    string    `firestore:"userId"`
		Street    string    `firestore:"street"`
		City      string    `firestore:"city"`
		State     string    `firestore:"state"`
		ZipCode   string    `firestore:"zipCode"`
		Country   string    `firestore:"country"`
		CreatedAt time.Time `firestore:"createdAt"`
		UpdatedAt time.Time `firestore:"updatedAt"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return shipdom.ShippingAddress{}, err
	}

	createdAt := raw.CreatedAt.UTC()
	if raw.CreatedAt.IsZero() {
		createdAt = time.Time{}
	}
	updatedAt := raw.UpdatedAt.UTC()
	if raw.UpdatedAt.IsZero() {
		updatedAt = createdAt
	}

	return shipdom.ShippingAddress{
		ID:        strings.TrimSpace(doc.Ref.ID),
		UserID:    strings.TrimSpace(raw.UserID),
		Street:    strings.TrimSpace(raw.Street),
		City:      strings.TrimSpace(raw.City),
		State:     strings.TrimSpace(raw.State),
		ZipCode:   strings.TrimSpace(raw.ZipCode),
		Country:   strings.TrimSpace(raw.Country),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func shippingAddressToDocData(v shipdom.ShippingAddress) map[string]any {
	data := map[string]any{
		"userId":  strings.TrimSpace(v.UserID),
		"street":  strings.TrimSpace(v.Street),
		"city":    strings.TrimSpace(v.City),
		"state":   strings.TrimSpace(v.State),
		"zipCode": strings.TrimSpace(v.ZipCode),
		"country": strings.TrimSpace(v.Country),
	}

	if !v.CreatedAt.IsZero() {
		data["createdAt"] = v.CreatedAt.UTC()
	}
	if !v.UpdatedAt.IsZero() {
		data["updatedAt"] = v.UpdatedAt.UTC()
	}

	return data
}

// ============================================================
// Filter / Sort Helpers
// ============================================================

// matchAddrFilter applies shipdom.Filter in-memory.
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

	if f.CreatedFrom != nil && a.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && !a.CreatedAt.Before(f.CreatedTo.UTC()) {
		return false
	}
	if f.UpdatedFrom != nil && a.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
		return false
	}
	if f.UpdatedTo != nil && !a.UpdatedAt.Before(f.UpdatedTo.UTC()) {
		return false
	}

	return true
}

// applyAddrOrderByFS maps shipdom.Sort to Firestore orderBy.
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
		// default: updatedAt DESC, id DESC
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

// ============================================================
// Small helpers
// ============================================================

// optString converts non-empty string to *string, else nil.
func optString(s string) *string {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	return &t
}
