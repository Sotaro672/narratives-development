// backend/internal/adapters/out/firestore/fulfillment_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fdom "narratives/internal/domain/fulfillment"
)

const (
	fulfillmentsCol   = "fulfillments"
	defaultPageSize   = 50
	maxCursorPageSize = 200
)

// FulfillmentRepositoryFS implements fulfillment repository using Firestore.
type FulfillmentRepositoryFS struct {
	Client *firestore.Client
}

func NewFulfillmentRepositoryFS(client *firestore.Client) *FulfillmentRepositoryFS {
	return &FulfillmentRepositoryFS{Client: client}
}

func (r *FulfillmentRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection(fulfillmentsCol)
}

// =======================
// Queries
// =======================

func (r *FulfillmentRepositoryFS) GetByID(ctx context.Context, id string) (*fdom.Fulfillment, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fdom.ErrNotFound
		}
		return nil, err
	}

	f, err := docToFulfillment(snap)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *FulfillmentRepositoryFS) GetByOrderID(ctx context.Context, orderID string) ([]fdom.Fulfillment, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return []fdom.Fulfillment{}, nil
	}

	q := r.col().
		Where("order_id", "==", orderID).
		OrderBy("created_at", firestore.Asc).
		OrderBy("id", firestore.Asc)

	it := q.Documents(ctx)
	defer it.Stop()

	var out []fdom.Fulfillment
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		f, err := docToFulfillment(doc)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

func (r *FulfillmentRepositoryFS) GetLatestByOrderID(ctx context.Context, orderID string) (*fdom.Fulfillment, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, fdom.ErrNotFound
	}

	q := r.col().
		Where("order_id", "==", orderID).
		OrderBy("updated_at", firestore.Desc).
		OrderBy("created_at", firestore.Desc).
		OrderBy("id", firestore.Desc).
		Limit(1)

	it := q.Documents(ctx)
	defer it.Stop()

	doc, err := it.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, fdom.ErrNotFound
		}
		return nil, err
	}

	f, err := docToFulfillment(doc)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *FulfillmentRepositoryFS) List(
	ctx context.Context,
	filter fdom.Filter,
	sortSpec fdom.Sort,
	page fdom.Page,
) (fdom.PageResult, error) {
	// NOTE: For simplicity (and Firestore's query limitations), we:
	//  1. Load all docs (or a sorted stream),
	//  2. Filter in-memory,
	//  3. Sort in-memory,
	//  4. Apply offset pagination.
	// Optimize with indexed where-clauses if needed.

	q := r.col().Query
	q = applyFulfillmentSortToQuery(q, sortSpec)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []fdom.Fulfillment
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fdom.PageResult{}, err
		}
		f, err := docToFulfillment(doc)
		if err != nil {
			return fdom.PageResult{}, err
		}
		if matchFulfillmentFilter(f, filter) {
			all = append(all, f)
		}
	}

	// In-memory sort as a safety (in case query sort is partial)
	sortFulfillments(all, sortSpec)

	perPage := page.PerPage
	if perPage <= 0 {
		perPage = defaultPageSize
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
	if perPage > 0 && total > 0 {
		totalPages = (total + perPage - 1) / perPage
	}

	return fdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *FulfillmentRepositoryFS) Count(ctx context.Context, filter fdom.Filter) (int, error) {
	q := r.col().Query
	it := q.Documents(ctx)
	defer it.Stop()

	total := 0
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		f, err := docToFulfillment(doc)
		if err != nil {
			return 0, err
		}
		if matchFulfillmentFilter(f, filter) {
			total++
		}
	}
	return total, nil
}

// =======================
// Mutations
// =======================

func (r *FulfillmentRepositoryFS) Create(ctx context.Context, in fdom.CreateFulfillmentInput) (*fdom.Fulfillment, error) {
	orderID := strings.TrimSpace(in.OrderID)
	if orderID == "" {
		return nil, errors.New("missing order_id")
	}

	paymentID := strings.TrimSpace(in.PaymentID)
	statusStr := strings.TrimSpace(string(in.Status))
	if statusStr == "" {
		return nil, errors.New("missing status")
	}

	now := time.Now().UTC()
	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}
	updatedAt := createdAt
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updatedAt = in.UpdatedAt.UTC()
	}

	docRef := r.col().NewDoc()
	data := map[string]any{
		"order_id":   orderID,
		"payment_id": paymentID,
		"status":     statusStr,
		"created_at": createdAt,
		"updated_at": updatedAt,
	}

	if _, err := docRef.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, fdom.ErrConflict
		}
		return nil, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return nil, err
	}
	f, err := docToFulfillment(snap)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *FulfillmentRepositoryFS) Update(ctx context.Context, id string, in fdom.UpdateFulfillmentInput) (*fdom.Fulfillment, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fdom.ErrNotFound
	}

	docRef := r.col().Doc(id)
	var updates []firestore.Update

	if in.Status != nil {
		statusStr := strings.TrimSpace(string(*in.Status))
		if statusStr != "" {
			updates = append(updates, firestore.Update{
				Path:  "status",
				Value: statusStr,
			})
		}
	}

	// updated_at: explicit or NOW() if any field changed
	if in.UpdatedAt != nil {
		updates = append(updates, firestore.Update{
			Path:  "updated_at",
			Value: in.UpdatedAt.UTC(),
		})
	} else if len(updates) > 0 {
		updates = append(updates, firestore.Update{
			Path:  "updated_at",
			Value: time.Now().UTC(),
		})
	}

	if len(updates) == 0 {
		// nothing to update; just return current
		return r.GetByID(ctx, id)
	}

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fdom.ErrNotFound
		}
		return nil, err
	}

	return r.GetByID(ctx, id)
}

func (r *FulfillmentRepositoryFS) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fdom.ErrNotFound
	}

	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fdom.ErrNotFound
		}
		return err
	}
	return nil
}

// Reset clears all fulfillments in Firestore (best-effort; mainly for tests/tools).
func (r *FulfillmentRepositoryFS) Reset(ctx context.Context) error {
	it := r.col().Documents(ctx)
	batch := r.Client.Batch()
	count := 0

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		batch.Delete(doc.Ref)
		count++
		// Commit periodically to avoid huge batches (limit ~500)
		if count >= 450 {
			if _, err := batch.Commit(ctx); err != nil {
				return err
			}
			batch = r.Client.Batch()
			count = 0
		}
	}
	if count > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

// WithTx: Firestore-style transaction wrapper.
// Note: The repository methods above don't take a transaction directly;
// this provides a hook if you want to orchestrate multiple operations.
// Callers that need full transactional semantics should implement them
// inside fn using the firestore.Transaction APIs directly.
func (r *FulfillmentRepositoryFS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	// Best-effort adapter: run fn without injecting tx-bound methods.
	// If stricter semantics are required, adjust repo methods to accept/use *firestore.Transaction.
	return fn(ctx)
}

// =======================
// Helpers
// =======================

func docToFulfillment(doc *firestore.DocumentSnapshot) (fdom.Fulfillment, error) {
	data := doc.Data()
	if data == nil {
		return fdom.Fulfillment{}, fmt.Errorf("empty fulfillment document: %s", doc.Ref.ID)
	}

	getStr := func(key string) string {
		if v, ok := data[key].(string); ok {
			return strings.TrimSpace(v)
		}
		return ""
	}
	getTime := func(key string) time.Time {
		if v, ok := data[key].(time.Time); ok {
			return v.UTC()
		}
		return time.Time{}
	}

	id := strings.TrimSpace(doc.Ref.ID)
	orderID := getStr("order_id")
	paymentID := getStr("payment_id")
	statusStr := getStr("status")
	createdAt := getTime("created_at")
	updatedAt := getTime("updated_at")

	return fdom.Fulfillment{
		ID:        id,
		OrderID:   orderID,
		PaymentID: paymentID,
		Status:    fdom.FulfillmentStatus(statusStr),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func matchFulfillmentFilter(f fdom.Fulfillment, flt fdom.Filter) bool {
	// IDs
	if len(flt.IDs) > 0 && !containsStringFulfill(flt.IDs, f.ID) {
		return false
	}

	// OrderIDs
	if len(flt.OrderIDs) > 0 && !containsStringFulfill(flt.OrderIDs, f.OrderID) {
		return false
	}

	// PaymentIDs
	if len(flt.PaymentIDs) > 0 && !containsStringFulfill(flt.PaymentIDs, f.PaymentID) {
		return false
	}

	// Statuses
	if len(flt.Statuses) > 0 {
		match := false
		for _, st := range flt.Statuses {
			if strings.TrimSpace(string(st)) == string(f.Status) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	// Created range
	if flt.CreatedFrom != nil && f.CreatedAt.Before(flt.CreatedFrom.UTC()) {
		return false
	}
	if flt.CreatedTo != nil && f.CreatedAt.After(flt.CreatedTo.UTC()) {
		return false
	}

	// Updated range
	if flt.UpdatedFrom != nil && f.UpdatedAt.Before(flt.UpdatedFrom.UTC()) {
		return false
	}
	if flt.UpdatedTo != nil && f.UpdatedAt.After(flt.UpdatedTo.UTC()) {
		return false
	}

	return true
}

func applyFulfillmentSortToQuery(q firestore.Query, sortSpec fdom.Sort) firestore.Query {
	col, dir := mapFulfillmentSort(sortSpec)
	if col == "" {
		// Default: created_at DESC, id DESC
		return q.OrderBy("created_at", firestore.Desc).OrderBy("id", firestore.Desc)
	}
	// Stable tie-breaker by id
	return q.OrderBy(col, dir).OrderBy("id", firestore.Asc)
}

func mapFulfillmentSort(sortSpec fdom.Sort) (string, firestore.Direction) {
	col := strings.ToLower(string(sortSpec.Column))

	var field string
	switch col {
	case strings.ToLower(string(fdom.SortByCreatedAt)):
		field = "created_at"
	case strings.ToLower(string(fdom.SortByUpdatedAt)):
		field = "updated_at"
	case strings.ToLower(string(fdom.SortByStatus)):
		field = "status"
	default:
		field = ""
	}

	if field == "" {
		return "", firestore.Desc
	}

	dir := firestore.Asc
	if strings.EqualFold(string(sortSpec.Order), "desc") {
		dir = firestore.Desc
	}
	return field, dir
}

func sortFulfillments(list []fdom.Fulfillment, sortSpec fdom.Sort) {
	col := strings.ToLower(string(sortSpec.Column))
	desc := strings.EqualFold(string(sortSpec.Order), "desc")

	less := func(i, j int) bool {
		a, b := list[i], list[j]
		switch col {
		case strings.ToLower(string(fdom.SortByCreatedAt)):
			if a.CreatedAt.Equal(b.CreatedAt) {
				return a.ID < b.ID
			}
			if desc {
				return a.CreatedAt.After(b.CreatedAt)
			}
			return a.CreatedAt.Before(b.CreatedAt)
		case strings.ToLower(string(fdom.SortByUpdatedAt)):
			if a.UpdatedAt.Equal(b.UpdatedAt) {
				return a.ID < b.ID
			}
			if desc {
				return a.UpdatedAt.After(b.UpdatedAt)
			}
			return a.UpdatedAt.Before(b.UpdatedAt)
		case strings.ToLower(string(fdom.SortByStatus)):
			if a.Status == b.Status {
				if desc {
					return a.CreatedAt.After(b.CreatedAt)
				}
				return a.CreatedAt.Before(b.CreatedAt)
			}
			if desc {
				return string(a.Status) > string(b.Status)
			}
			return string(a.Status) < string(b.Status)
		default:
			// default consistent with repo default: created_at DESC, id DESC
			if a.CreatedAt.Equal(b.CreatedAt) {
				if desc {
					return a.ID > b.ID
				}
				return a.ID < b.ID
			}
			// use DESC when unspecified to mimic SQL impl
			return a.CreatedAt.After(b.CreatedAt)
		}
	}

	sort.SliceStable(list, less)
}

func containsStringFulfill(list []string, v string) bool {
	v = strings.TrimSpace(v)
	for _, s := range list {
		if strings.TrimSpace(s) == v {
			return true
		}
	}
	return false
}
