// backend/internal/adapters/out/firestore/transfer_repository_fs.go
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

	dbcommon "narratives/internal/adapters/out/firestore/common"
	trdom "narratives/internal/domain/transfer"
)

// =====================================================
// Firestore Transfer Repository
// (Firestore 実装; 旧 PG 実装と互換のメソッドも提供)
// =====================================================

type TransferRepositoryFS struct {
	Client *firestore.Client
}

func NewTransferRepositoryFS(client *firestore.Client) *TransferRepositoryFS {
	return &TransferRepositoryFS{Client: client}
}

func (r *TransferRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("transfers")
}

// =====================================================
// RepositoryPort 相当メソッド
// =====================================================

// GetByID returns a Transfer by ID.
func (r *TransferRepositoryFS) GetByID(ctx context.Context, id string) (*trdom.Transfer, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, trdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, trdom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	tr, err := docToTransfer(snap)
	if err != nil {
		return nil, err
	}
	return &tr, nil
}

// List applies Filter/Sort/Page semantics in-memory over transfers collection.
// （Firestore の制約回避のため、一旦取得してメモリ上で絞り込み/ソート/ページング）
func (r *TransferRepositoryFS) List(
	ctx context.Context,
	filter trdom.Filter,
	sortOpt trdom.Sort,
	page trdom.Page,
) (trdom.PageResult, error) {
	if r.Client == nil {
		return trdom.PageResult{}, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var all []trdom.Transfer
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return trdom.PageResult{}, err
		}
		tr, err := docToTransfer(doc)
		if err != nil {
			return trdom.PageResult{}, err
		}
		if matchTransferFilter(tr, filter) {
			all = append(all, tr)
		}
	}

	// sort
	sortTransfers(all, sortOpt)

	// paging
	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)
	total := len(all)

	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}
	paged := all[offset:end]

	return trdom.PageResult{
		Items:      paged,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// Count counts transfers matching filter (in-memory, same 条件 as List).
func (r *TransferRepositoryFS) Count(ctx context.Context, filter trdom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	count := 0
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}
		tr, err := docToTransfer(doc)
		if err != nil {
			return 0, err
		}
		if matchTransferFilter(tr, filter) {
			count++
		}
	}
	return count, nil
}

// Create inserts a new Transfer (status="requested", no errorType, no transferredAt).
func (r *TransferRepositoryFS) Create(ctx context.Context, in trdom.CreateTransferInput) (*trdom.Transfer, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	ref := r.col().NewDoc()
	data := map[string]any{
		"mintAddress": strings.TrimSpace(in.MintAddress),
		"fromAddress": strings.TrimSpace(in.FromAddress),
		"toAddress":   strings.TrimSpace(in.ToAddress),
		"requestedAt": in.RequestedAt.UTC(),
		"status":      "requested", // PG版と同じ初期値
	}

	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, trdom.ErrConflict
		}
		return nil, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	tr, err := docToTransfer(snap)
	if err != nil {
		return nil, err
	}
	return &tr, nil
}

// Update partially updates a Transfer by ID.
func (r *TransferRepositoryFS) Update(ctx context.Context, id string, in trdom.UpdateTransferInput) (*trdom.Transfer, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, trdom.ErrNotFound
	}

	ref := r.col().Doc(id)

	// ensure exists
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return nil, trdom.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	var updates []firestore.Update

	// status
	if in.Status != nil {
		updates = append(updates, firestore.Update{
			Path:  "status",
			Value: strings.TrimSpace(string(*in.Status)),
		})
	}

	// errorType: empty => delete
	if in.ErrorType != nil {
		v := strings.TrimSpace(string(*in.ErrorType))
		if v == "" {
			updates = append(updates, firestore.Update{
				Path:  "errorType",
				Value: firestore.Delete,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "errorType",
				Value: v,
			})
		}
	}

	// transferredAt: zero => delete
	if in.TransferredAt != nil {
		if in.TransferredAt.IsZero() {
			updates = append(updates, firestore.Update{
				Path:  "transferredAt",
				Value: firestore.Delete,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "transferredAt",
				Value: in.TransferredAt.UTC(),
			})
		}
	}

	if len(updates) == 0 {
		// no change -> reload
		return r.GetByID(ctx, id)
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, trdom.ErrNotFound
		}
		if status.Code(err) == codes.AlreadyExists {
			return nil, trdom.ErrConflict
		}
		return nil, err
	}

	return r.GetByID(ctx, id)
}

// Delete removes a Transfer by ID.
func (r *TransferRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return trdom.ErrNotFound
	}

	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return trdom.ErrNotFound
	} else if err != nil {
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

// WithTx: simple wrapper; if strict multi-doc Tx needed, use Client.RunTransaction.
func (r *TransferRepositoryFS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	return fn(ctx)
}

// Reset: delete all transfers (dev/test use).
func (r *TransferRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	batch := r.Client.Batch()
	count := 0

	for {
		doc, err := it.Next()
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
	return nil
}

// =====================================================
// Compatibility methods (旧 TransferRepositoryPG と同名)
// =====================================================

func (r *TransferRepositoryFS) GetAllTransfers(ctx context.Context) ([]*trdom.Transfer, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	it := r.col().
		OrderBy("requestedAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc).
		Documents(ctx)
	defer it.Stop()

	var out []*trdom.Transfer
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		tr, err := docToTransfer(doc)
		if err != nil {
			return nil, err
		}
		tt := tr
		out = append(out, &tt)
	}
	return out, nil
}

func (r *TransferRepositoryFS) GetTransferByID(ctx context.Context, id string) (*trdom.Transfer, error) {
	return r.GetByID(ctx, id)
}

func (r *TransferRepositoryFS) GetTransfersByFromAddress(ctx context.Context, fromAddress string) ([]*trdom.Transfer, error) {
	return r.getTransfersByField(ctx, "fromAddress", strings.TrimSpace(fromAddress))
}

func (r *TransferRepositoryFS) GetTransfersByToAddress(ctx context.Context, toAddress string) ([]*trdom.Transfer, error) {
	return r.getTransfersByField(ctx, "toAddress", strings.TrimSpace(toAddress))
}

func (r *TransferRepositoryFS) GetTransfersByMintAddress(ctx context.Context, mintAddress string) ([]*trdom.Transfer, error) {
	return r.getTransfersByField(ctx, "mintAddress", strings.TrimSpace(mintAddress))
}

func (r *TransferRepositoryFS) GetTransfersByStatus(ctx context.Context, status string) ([]*trdom.Transfer, error) {
	return r.getTransfersByField(ctx, "status", strings.TrimSpace(status))
}

func (r *TransferRepositoryFS) CreateTransfer(ctx context.Context, in trdom.CreateTransferInput) (*trdom.Transfer, error) {
	return r.Create(ctx, in)
}

func (r *TransferRepositoryFS) UpdateTransfer(ctx context.Context, id string, in trdom.UpdateTransferInput) (*trdom.Transfer, error) {
	return r.Update(ctx, id, in)
}

func (r *TransferRepositoryFS) DeleteTransfer(ctx context.Context, id string) error {
	return r.Delete(ctx, id)
}

func (r *TransferRepositoryFS) ResetTransfers(ctx context.Context) error {
	return r.Reset(ctx)
}

func (r *TransferRepositoryFS) getTransfersByField(ctx context.Context, field, val string) ([]*trdom.Transfer, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	val = strings.TrimSpace(val)
	if val == "" {
		return []*trdom.Transfer{}, nil
	}

	q := r.col().
		Where(field, "==", val).
		OrderBy("requestedAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	it := q.Documents(ctx)
	defer it.Stop()

	var out []*trdom.Transfer
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		tr, err := docToTransfer(doc)
		if err != nil {
			return nil, err
		}
		tt := tr
		out = append(out, &tt)
	}
	return out, nil
}

// =====================================================
// Helpers: Firestore -> Domain
// =====================================================

func docToTransfer(doc *firestore.DocumentSnapshot) (trdom.Transfer, error) {
	data := doc.Data()
	if data == nil {
		return trdom.Transfer{}, trdom.ErrNotFound
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
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
		if t := getTimePtr(keys...); t != nil {
			return *t
		}
		return time.Time{}
	}
	getStatus := func(key string) trdom.TransferStatus {
		if v, ok := data[key].(string); ok {
			return trdom.TransferStatus(strings.TrimSpace(v))
		}
		return ""
	}
	getErrType := func(keys ...string) *trdom.TransferErrorType {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				s := strings.TrimSpace(v)
				if s != "" {
					et := trdom.TransferErrorType(s)
					return &et
				}
			}
		}
		return nil
	}

	return trdom.Transfer{
		ID:            strings.TrimSpace(doc.Ref.ID),
		MintAddress:   getStr("mintAddress", "mint_address"),
		FromAddress:   getStr("fromAddress", "from_address"),
		ToAddress:     getStr("toAddress", "to_address"),
		RequestedAt:   getTimeVal("requestedAt", "requested_at"),
		TransferredAt: getTimePtr("transferredAt", "transferred_at"),
		Status:        getStatus("status"),
		ErrorType:     getErrType("errorType", "error_type"),
	}, nil
}

// =====================================================
// Helpers: Filter / Sort
// =====================================================

func matchTransferFilter(t trdom.Transfer, f trdom.Filter) bool {
	// ID
	if v := strings.TrimSpace(f.ID); v != "" && t.ID != v {
		return false
	}
	// MintAddress
	if v := strings.TrimSpace(f.MintAddress); v != "" && t.MintAddress != v {
		return false
	}
	// FromAddress
	if v := strings.TrimSpace(f.FromAddress); v != "" && t.FromAddress != v {
		return false
	}
	// ToAddress
	if v := strings.TrimSpace(f.ToAddress); v != "" && t.ToAddress != v {
		return false
	}

	// Statuses
	if len(f.Statuses) > 0 {
		match := false
		for _, s := range f.Statuses {
			if t.Status == s {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	// ErrorTypes
	if len(f.ErrorTypes) > 0 {
		match := false
		for _, et := range f.ErrorTypes {
			if t.ErrorType != nil && *t.ErrorType == et {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	// HasError
	if f.HasError != nil {
		if *f.HasError && t.ErrorType == nil {
			return false
		}
		if !*f.HasError && t.ErrorType != nil {
			return false
		}
	}

	// RequestedAt range
	if f.RequestedFrom != nil && t.RequestedAt.Before(f.RequestedFrom.UTC()) {
		return false
	}
	if f.RequestedTo != nil && !t.RequestedAt.Before(f.RequestedTo.UTC()) {
		return false
	}

	// TransferredAt range
	if f.TransferedFrom != nil {
		if t.TransferredAt == nil || t.TransferredAt.Before(f.TransferedFrom.UTC()) {
			return false
		}
	}
	if f.TransferedTo != nil {
		if t.TransferredAt == nil || !t.TransferredAt.Before(f.TransferedTo.UTC()) {
			return false
		}
	}

	return true
}

func sortTransfers(items []trdom.Transfer, s trdom.Sort) {
	// default: requestedAt DESC, id DESC
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	asc := dir == "ASC"

	less := func(i, j int) bool {
		a, b := items[i], items[j]

		switch col {
		case "requestedat", "requested_at":
			if a.RequestedAt.Equal(b.RequestedAt) {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return a.RequestedAt.Before(b.RequestedAt)
			}
			return a.RequestedAt.After(b.RequestedAt)

		case "transferredat", "transferred_at":
			var at, bt time.Time
			if a.TransferredAt != nil {
				at = *a.TransferredAt
			}
			if b.TransferredAt != nil {
				bt = *b.TransferredAt
			}
			if at.Equal(bt) {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return at.Before(bt)
			}
			return at.After(bt)

		case "status":
			if a.Status == b.Status {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return string(a.Status) < string(b.Status)
			}
			return string(a.Status) > string(b.Status)

		default:
			// fallback to requestedAt DESC, id DESC
			if a.RequestedAt.Equal(b.RequestedAt) {
				return a.ID > b.ID
			}
			return a.RequestedAt.After(b.RequestedAt)
		}
	}

	sort.SliceStable(items, less)
}
