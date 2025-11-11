// backend/internal/adapters/out/firestore/payment_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	paymentdom "narratives/internal/domain/payment"
)

// Firestore-based implementation of Payment repository.
type PaymentRepositoryFS struct {
	Client *firestore.Client
}

func NewPaymentRepositoryFS(client *firestore.Client) *PaymentRepositoryFS {
	return &PaymentRepositoryFS{Client: client}
}

func (r *PaymentRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("payments")
}

// ============================================================
// PaymentRepo Port implementation
// ============================================================

// GetByID implements PaymentRepo.GetByID.
func (r *PaymentRepositoryFS) GetByID(ctx context.Context, id string) (paymentdom.Payment, error) {
	if r.Client == nil {
		return paymentdom.Payment{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return paymentdom.Payment{}, paymentdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return paymentdom.Payment{}, paymentdom.ErrNotFound
		}
		return paymentdom.Payment{}, err
	}

	p, err := docToPayment(snap)
	if err != nil {
		return paymentdom.Payment{}, err
	}
	return p, nil
}

// Exists implements PaymentRepo.Exists.
func (r *PaymentRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

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

// Create implements PaymentRepo.Create.
func (r *PaymentRepositoryFS) Create(ctx context.Context, v paymentdom.Payment) (paymentdom.Payment, error) {
	if r.Client == nil {
		return paymentdom.Payment{}, errors.New("firestore client is nil")
	}

	nowUTC := time.Now().UTC()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = nowUTC
	}
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = nowUTC
	}

	id := strings.TrimSpace(v.ID)
	var docRef *firestore.DocumentRef

	if id == "" {
		docRef = r.col().NewDoc()
		v.ID = docRef.ID
	} else {
		docRef = r.col().Doc(id)
		v.ID = id
	}

	data := paymentToDoc(v)

	_, err := docRef.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return paymentdom.Payment{}, paymentdom.ErrConflict
		}
		return paymentdom.Payment{}, err
	}

	// We know what we wrote; return v with normalized fields.
	return v, nil
}

// Save implements PaymentRepo.Save (upsert-like behavior).
func (r *PaymentRepositoryFS) Save(ctx context.Context, v paymentdom.Payment) (paymentdom.Payment, error) {
	if r.Client == nil {
		return paymentdom.Payment{}, errors.New("firestore client is nil")
	}

	nowUTC := time.Now().UTC()
	id := strings.TrimSpace(v.ID)

	// If no ID, behave as Create with auto-ID.
	if id == "" {
		return r.Create(ctx, v)
	}

	docRef := r.col().Doc(id)

	// Try to load existing to preserve createdAt semantics.
	existingSnap, err := docRef.Get(ctx)
	if err == nil {
		// Existing document
		existing, err := docToPayment(existingSnap)
		if err != nil {
			return paymentdom.Payment{}, err
		}
		// Preserve earliest CreatedAt if not provided or later.
		if v.CreatedAt.IsZero() || v.CreatedAt.After(existing.CreatedAt) {
			v.CreatedAt = existing.CreatedAt
		}
	} else if status.Code(err) == codes.NotFound {
		// New document; ensure CreatedAt.
		if v.CreatedAt.IsZero() {
			v.CreatedAt = nowUTC
		}
	} else {
		return paymentdom.Payment{}, err
	}

	if v.UpdatedAt.IsZero() || v.UpdatedAt.Before(nowUTC) {
		v.UpdatedAt = nowUTC
	}

	v.ID = id
	data := paymentToDoc(v)

	// MergeAll approximates ON CONFLICT DO UPDATE semantics.
	_, err = docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	// Return latest representation (we can trust v as source of truth here).
	return v, nil
}

// Delete implements PaymentRepo.Delete.
func (r *PaymentRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return paymentdom.ErrNotFound
	}

	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return paymentdom.ErrNotFound
		}
		return err
	}
	return nil
}

// ============================================================
// Extra methods (not strictly required by Port, kept for parity)
// ============================================================

func (r *PaymentRepositoryFS) GetByInvoiceID(ctx context.Context, invoiceID string) ([]paymentdom.Payment, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	invoiceID = strings.TrimSpace(invoiceID)
	if invoiceID == "" {
		return []paymentdom.Payment{}, nil
	}

	q := r.col().Where("invoiceId", "==", invoiceID)
	it := q.Documents(ctx)
	defer it.Stop()

	var out []paymentdom.Payment
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		p, err := docToPayment(doc)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *PaymentRepositoryFS) List(
	ctx context.Context,
	filter paymentdom.Filter,
	sort paymentdom.Sort,
	page paymentdom.Page,
) (paymentdom.PageResult, error) {
	if r.Client == nil {
		return paymentdom.PageResult{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	q := r.col().Query
	q = applyPaymentSort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []paymentdom.Payment
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return paymentdom.PageResult{}, err
		}
		p, err := docToPayment(doc)
		if err != nil {
			return paymentdom.PageResult{}, err
		}
		if matchPaymentFilter(p, filter) {
			all = append(all, p)
		}
	}

	total := len(all)
	if total == 0 {
		return paymentdom.PageResult{
			Items:      []paymentdom.Payment{},
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

	return paymentdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *PaymentRepositoryFS) Count(ctx context.Context, filter paymentdom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
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
		p, err := docToPayment(doc)
		if err != nil {
			return 0, err
		}
		if matchPaymentFilter(p, filter) {
			total++
		}
	}
	return total, nil
}

// Update is a convenience helper (not necessarily in the Port).
func (r *PaymentRepositoryFS) Update(ctx context.Context, id string, patch paymentdom.UpdatePaymentInput) (paymentdom.Payment, error) {
	if r.Client == nil {
		return paymentdom.Payment{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return paymentdom.Payment{}, paymentdom.ErrNotFound
	}

	docRef := r.col().Doc(id)
	var updates []firestore.Update

	setStr := func(path string, p *string) {
		if p != nil {
			v := strings.TrimSpace(*p)
			if v == "" {
				updates = append(updates, firestore.Update{
					Path:  path,
					Value: firestore.Delete,
				})
			} else {
				updates = append(updates, firestore.Update{
					Path:  path,
					Value: v,
				})
			}
		}
	}

	if patch.InvoiceID != nil {
		setStr("invoiceId", patch.InvoiceID)
	}
	if patch.BillingAddressID != nil {
		setStr("billingAddressId", patch.BillingAddressID)
	}
	if patch.Amount != nil {
		updates = append(updates, firestore.Update{
			Path:  "amount",
			Value: *patch.Amount,
		})
	}
	if patch.Status != nil {
		updates = append(updates, firestore.Update{
			Path:  "status",
			Value: string(*patch.Status),
		})
	}
	if patch.ErrorType != nil {
		setStr("errorType", patch.ErrorType)
	}

	// always bump updatedAt
	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: time.Now().UTC(),
	})

	if len(updates) == 0 {
		return r.GetByID(ctx, id)
	}

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return paymentdom.Payment{}, paymentdom.ErrNotFound
		}
		if status.Code(err) == codes.AlreadyExists {
			return paymentdom.Payment{}, paymentdom.ErrConflict
		}
		return paymentdom.Payment{}, err
	}

	return r.GetByID(ctx, id)
}

// Reset is a test utility that deletes all payments using Transactions instead of WriteBatch.
func (r *PaymentRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var refs []*firestore.DocumentRef
	for {
		doc, err := it.Next()
		if err == iterator.Done {
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
// Helpers
// ============================================================

func docToPayment(doc *firestore.DocumentSnapshot) (paymentdom.Payment, error) {
	data := doc.Data()
	if data == nil {
		return paymentdom.Payment{}, fmt.Errorf("empty payment document: %s", doc.Ref.ID)
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	getStrPtr := func(keys ...string) *string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				s := strings.TrimSpace(v)
				if s != "" {
					return &s
				}
			}
		}
		return nil
	}
	getTimePtr := func(keys ...string) *time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok && !v.IsZero() {
				t := v.UTC()
				return &t
			}
		}
		return nil
	}
	getInt := func(key string) int {
		if v, ok := data[key]; ok {
			switch n := v.(type) {
			case int:
				return n
			case int64:
				return int(n)
			case float64:
				return int(n)
			}
		}
		return 0
	}

	p := paymentdom.Payment{
		ID:               doc.Ref.ID,
		InvoiceID:        getStr("invoiceId", "invoice_id"),
		BillingAddressID: getStr("billingAddressId", "billing_address_id"),
		Amount:           getInt("amount"),
		Status:           paymentdom.PaymentStatus(getStr("status")),
		ErrorType:        getStrPtr("errorType", "error_type"),
		CreatedAt:        timeOrZero(getTimePtr("createdAt", "created_at")),
		UpdatedAt:        timeOrZero(getTimePtr("updatedAt", "updated_at")),
		DeletedAt:        getTimePtr("deletedAt", "deleted_at"),
	}

	return p, nil
}

func paymentToDoc(v paymentdom.Payment) map[string]any {
	m := map[string]any{
		"invoiceId":        strings.TrimSpace(v.InvoiceID),
		"billingAddressId": strings.TrimSpace(v.BillingAddressID),
		"amount":           v.Amount,
		"status":           string(v.Status),
	}

	if v.ErrorType != nil {
		if s := strings.TrimSpace(*v.ErrorType); s != "" {
			m["errorType"] = s
		}
	}

	// timestamps
	if !v.CreatedAt.IsZero() {
		m["createdAt"] = v.CreatedAt.UTC()
	}
	if !v.UpdatedAt.IsZero() {
		m["updatedAt"] = v.UpdatedAt.UTC()
	}
	if v.DeletedAt != nil && !v.DeletedAt.IsZero() {
		m["deletedAt"] = v.DeletedAt.UTC()
	}

	return m
}

func timeOrZero(p *time.Time) time.Time {
	if p == nil {
		return time.Time{}
	}
	return p.UTC()
}

// matchPaymentFilter applies Filter in-memory (Firestore-friendly mirror of buildPaymentWhere).
func matchPaymentFilter(p paymentdom.Payment, f paymentdom.Filter) bool {
	trimEq := func(a, b string) bool {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}

	if strings.TrimSpace(f.ID) != "" && !trimEq(p.ID, f.ID) {
		return false
	}
	if strings.TrimSpace(f.InvoiceID) != "" && !trimEq(p.InvoiceID, f.InvoiceID) {
		return false
	}
	if strings.TrimSpace(f.BillingAddressID) != "" && !trimEq(p.BillingAddressID, f.BillingAddressID) {
		return false
	}

	if len(f.Statuses) > 0 {
		ok := false
		for _, st := range f.Statuses {
			if string(p.Status) == string(st) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	if v := strings.TrimSpace(f.ErrorType); v != "" {
		if p.ErrorType == nil || strings.TrimSpace(*p.ErrorType) != v {
			return false
		}
	}

	if f.MinAmount != nil && p.Amount < *f.MinAmount {
		return false
	}
	if f.MaxAmount != nil && p.Amount > *f.MaxAmount {
		return false
	}

	// Time ranges (CreatedAt/UpdatedAt/DeletedAt)
	if f.CreatedFrom != nil && p.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && !p.CreatedAt.Before(f.CreatedTo.UTC()) {
		return false
	}
	if f.UpdatedFrom != nil && p.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
		return false
	}
	if f.UpdatedTo != nil && !p.UpdatedAt.Before(f.UpdatedTo.UTC()) {
		return false
	}
	if f.DeletedFrom != nil {
		if p.DeletedAt == nil || p.DeletedAt.Before(f.DeletedFrom.UTC()) {
			return false
		}
	}
	if f.DeletedTo != nil {
		if p.DeletedAt == nil || !p.DeletedAt.Before(f.DeletedTo.UTC()) {
			return false
		}
	}

	// Deleted tri-state
	if f.Deleted != nil {
		if *f.Deleted {
			if p.DeletedAt == nil {
				return false
			}
		} else {
			if p.DeletedAt != nil {
				return false
			}
		}
	}

	return true
}

// applyPaymentSort maps Sort to Firestore orderBy.
func applyPaymentSort(q firestore.Query, sort paymentdom.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
	var field string

	switch col {
	case "createdat", "created_at":
		field = "createdAt"
	case "updatedat", "updated_at":
		field = "updatedAt"
	case "amount":
		field = "amount"
	case "status":
		field = "status"
	default:
		// Default: createdAt DESC, id DESC
		return q.OrderBy("createdAt", firestore.Desc).
			OrderBy(firestore.DocumentID, firestore.Desc)
	}

	dir := firestore.Asc
	if strings.EqualFold(string(sort.Order), "desc") {
		dir = firestore.Desc
	}

	return q.OrderBy(field, dir).OrderBy(firestore.DocumentID, dir)
}
