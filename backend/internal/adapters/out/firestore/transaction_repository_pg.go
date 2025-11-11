// backend/internal/adapters/out/firestore/transaction_repository_fs.go
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
	tr "narratives/internal/domain/transaction"
)

// =====================================================
// Firestore Transaction Repository
// (PostgreSQL 実装相当のインターフェースを Firestore で提供)
// =====================================================

type TransactionRepositoryFS struct {
	Client *firestore.Client
}

func NewTransactionRepositoryFS(client *firestore.Client) *TransactionRepositoryFS {
	return &TransactionRepositoryFS{Client: client}
}

func (r *TransactionRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("transactions")
}

// =====================================================
// RepositoryPort impl 相当
// =====================================================

// GetAllTransactions returns all transactions ordered by timestamp desc, id desc.
func (r *TransactionRepositoryFS) GetAllTransactions(ctx context.Context) ([]*tr.Transaction, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	it := r.col().
		OrderBy("timestamp", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc).
		Documents(ctx)
	defer it.Stop()

	var out []*tr.Transaction
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		t, err := docToTransaction(snap)
		if err != nil {
			return nil, err
		}
		tt := t
		out = append(out, &tt)
	}
	return out, nil
}

// GetTransactionByID returns a single transaction by ID.
func (r *TransactionRepositoryFS) GetTransactionByID(ctx context.Context, id string) (*tr.Transaction, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, tr.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, tr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	t, err := docToTransaction(snap)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetTransactionsByBrand retrieves transactions filtered by brandName.
func (r *TransactionRepositoryFS) GetTransactionsByBrand(ctx context.Context, brandName string) ([]*tr.Transaction, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	brandName = strings.TrimSpace(brandName)
	if brandName == "" {
		return []*tr.Transaction{}, nil
	}

	q := r.col().
		Where("brandName", "==", brandName).
		OrderBy("timestamp", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	it := q.Documents(ctx)
	defer it.Stop()

	var out []*tr.Transaction
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		t, err := docToTransaction(snap)
		if err != nil {
			return nil, err
		}
		tt := t
		out = append(out, &tt)
	}
	return out, nil
}

// GetTransactionsByAccount retrieves transactions filtered by accountID.
func (r *TransactionRepositoryFS) GetTransactionsByAccount(ctx context.Context, accountID string) ([]*tr.Transaction, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return []*tr.Transaction{}, nil
	}

	q := r.col().
		Where("accountId", "==", accountID).
		OrderBy("timestamp", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	it := q.Documents(ctx)
	defer it.Stop()

	var out []*tr.Transaction
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		t, err := docToTransaction(snap)
		if err != nil {
			return nil, err
		}
		tt := t
		out = append(out, &tt)
	}
	return out, nil
}

// SearchTransactions performs search with TransactionSearchCriteria.
// Firestore では制約があるため、全件取得してメモリ上でフィルタ/ソート/ページングする。
func (r *TransactionRepositoryFS) SearchTransactions(
	ctx context.Context,
	criteria tr.TransactionSearchCriteria,
) (txs []*tr.Transaction, total int, err error) {
	if r.Client == nil {
		return nil, 0, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var all []tr.Transaction
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		t, err := docToTransaction(snap)
		if err != nil {
			return nil, 0, err
		}
		if matchTxCriteria(t, criteria) {
			all = append(all, t)
		}
	}

	// sort
	sortTransactions(all, criteria.Sort)

	total = len(all)

	// paging
	perPage := 50
	offset := 0
	if criteria.Pagination != nil {
		_, perPage, offset = dbcommon.NormalizePage(
			criteria.Pagination.Page,
			criteria.Pagination.PerPage,
			50,
			200,
		)
	}

	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}

	out := make([]*tr.Transaction, 0, end-offset)
	for _, t := range all[offset:end] {
		tt := t
		out = append(out, &tt)
	}

	return out, total, nil
}

// CreateTransaction inserts a new transaction document.
func (r *TransactionRepositoryFS) CreateTransaction(ctx context.Context, in tr.CreateTransactionInput) (*tr.Transaction, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	ref := r.col().NewDoc()

	data := map[string]any{
		"accountId":   strings.TrimSpace(in.AccountID),
		"brandName":   strings.TrimSpace(in.BrandName),
		"type":        strings.TrimSpace(string(in.Type)),
		"amount":      in.Amount,
		"currency":    strings.ToUpper(strings.TrimSpace(in.Currency)),
		"fromAccount": strings.TrimSpace(in.FromAccount),
		"toAccount":   strings.TrimSpace(in.ToAccount),
		"timestamp":   in.Timestamp.UTC(),
		"description": in.Description,
	}

	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, tr.ErrConflict
		}
		return nil, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	t, err := docToTransaction(snap)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateTransaction applies partial updates to a transaction document.
func (r *TransactionRepositoryFS) UpdateTransaction(
	ctx context.Context,
	id string,
	in tr.UpdateTransactionInput,
) (*tr.Transaction, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, tr.ErrNotFound
	}

	ref := r.col().Doc(id)

	// ensure exists
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return nil, tr.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	var updates []firestore.Update

	setStr := func(path string, p *string) {
		if p != nil {
			updates = append(updates, firestore.Update{
				Path:  path,
				Value: strings.TrimSpace(*p),
			})
		}
	}
	setEnum := func(path string, p *tr.TransactionType) {
		if p != nil {
			updates = append(updates, firestore.Update{
				Path:  path,
				Value: strings.TrimSpace(string(*p)),
			})
		}
	}
	setInt := func(path string, p *int) {
		if p != nil {
			updates = append(updates, firestore.Update{
				Path:  path,
				Value: *p,
			})
		}
	}
	setTime := func(path string, p *time.Time) {
		if p != nil {
			updates = append(updates, firestore.Update{
				Path:  path,
				Value: p.UTC(),
			})
		}
	}

	setStr("accountId", in.AccountID)
	setStr("brandName", in.BrandName)
	setEnum("type", in.Type)
	setInt("amount", in.Amount)

	if in.Currency != nil {
		updates = append(updates, firestore.Update{
			Path:  "currency",
			Value: strings.ToUpper(strings.TrimSpace(*in.Currency)),
		})
	}

	setStr("fromAccount", in.FromAccount)
	setStr("toAccount", in.ToAccount)
	setTime("timestamp", in.Timestamp)
	setStr("description", in.Description)

	if len(updates) == 0 {
		// no-op => just reload
		return r.GetTransactionByID(ctx, id)
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, tr.ErrNotFound
		}
		if status.Code(err) == codes.AlreadyExists {
			return nil, tr.ErrConflict
		}
		return nil, err
	}

	return r.GetTransactionByID(ctx, id)
}

// DeleteTransaction deletes a transaction by ID.
func (r *TransactionRepositoryFS) DeleteTransaction(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return tr.ErrNotFound
	}

	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return tr.ErrNotFound
	} else if err != nil {
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

// ResetTransactions deletes all transaction documents (dev/test use).
func (r *TransactionRepositoryFS) ResetTransactions(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var refs []*firestore.DocumentRef
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		refs = append(refs, snap.Ref)
	}

	if len(refs) == 0 {
		return nil
	}

	const chunkSize = 400
	for i := 0; i < len(refs); i += chunkSize {
		end := i + chunkSize
		if end > len(refs) {
			end = len(refs)
		}
		if err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, ref := range refs[i:end] {
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

// WithTx is a simple wrapper; if multi-doc Tx is needed, use Client.RunTransaction.
func (r *TransactionRepositoryFS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	return fn(ctx)
}

// =====================================================
// Helpers: Firestore -> Domain
// =====================================================

func docToTransaction(doc *firestore.DocumentSnapshot) (tr.Transaction, error) {
	data := doc.Data()
	if data == nil {
		return tr.Transaction{}, tr.ErrNotFound
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	getInt64 := func(keys ...string) int64 {
		for _, k := range keys {
			switch v := data[k].(type) {
			case int64:
				return v
			case int:
				return int64(v)
			case float64:
				return int64(v)
			}
		}
		return 0
	}
	getTime := func(keys ...string) time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok {
				return v.UTC()
			}
		}
		return time.Time{}
	}

	amount := getInt64("amount")
	return tr.Transaction{
		ID:          strings.TrimSpace(doc.Ref.ID),
		AccountID:   getStr("accountId", "account_id"),
		BrandName:   getStr("brandName", "brand_name"),
		Type:        tr.TransactionType(getStr("type")),
		Amount:      int(amount),
		Currency:    strings.ToUpper(getStr("currency")),
		FromAccount: getStr("fromAccount", "from_account"),
		ToAccount:   getStr("toAccount", "to_account"),
		Timestamp:   getTime("timestamp"),
		Description: getStr("description"),
	}, nil
}

// =====================================================
// Helpers: Filter / Sort (in-memory, PG版 buildTxWhere/buildTxOrderBy 相当)
// =====================================================

func matchTxCriteria(t tr.Transaction, c tr.TransactionSearchCriteria) bool {
	f := c.Filters

	// AccountIDs
	if len(f.AccountIDs) > 0 {
		ok := false
		for _, v := range f.AccountIDs {
			if strings.TrimSpace(v) == t.AccountID {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// Brands
	if len(f.Brands) > 0 {
		ok := false
		for _, v := range f.Brands {
			if strings.TrimSpace(v) == t.BrandName {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// Currencies
	if len(f.Currencies) > 0 {
		cur := strings.ToUpper(strings.TrimSpace(t.Currency))
		if cur == "" {
			return false
		}
		ok := false
		for _, v := range f.Currencies {
			if strings.ToUpper(strings.TrimSpace(v)) == cur {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// FromAccounts
	if len(f.FromAccounts) > 0 {
		ok := false
		for _, v := range f.FromAccounts {
			if strings.TrimSpace(v) == t.FromAccount {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// ToAccounts
	if len(f.ToAccounts) > 0 {
		ok := false
		for _, v := range f.ToAccounts {
			if strings.TrimSpace(v) == t.ToAccount {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// Types
	if len(f.Types) > 0 {
		ok := false
		for _, tp := range f.Types {
			if t.Type == tp {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// DateFrom / DateTo
	if f.DateFrom != nil && t.Timestamp.Before(f.DateFrom.UTC()) {
		return false
	}
	if f.DateTo != nil && !t.Timestamp.Before(f.DateTo.UTC()) {
		return false
	}

	// AmountMin / AmountMax
	if f.AmountMin != nil && int64(t.Amount) < int64(*f.AmountMin) {
		return false
	}
	if f.AmountMax != nil && int64(t.Amount) > int64(*f.AmountMax) {
		return false
	}

	// DescriptionLike
	if v := strings.TrimSpace(f.DescriptionLike); v != "" {
		if !strings.Contains(
			strings.ToLower(t.Description),
			strings.ToLower(v),
		) {
			return false
		}
	}

	// SearchTerm across brandName, currency, fromAccount, toAccount, description
	if v := strings.TrimSpace(c.SearchTerm); v != "" {
		p := strings.ToLower(v)
		if !(strings.Contains(strings.ToLower(t.BrandName), p) ||
			strings.Contains(strings.ToLower(t.Currency), p) ||
			strings.Contains(strings.ToLower(t.FromAccount), p) ||
			strings.Contains(strings.ToLower(t.ToAccount), p) ||
			strings.Contains(strings.ToLower(t.Description), p)) {
			return false
		}
	}

	return true
}

func sortTransactions(items []tr.Transaction, s tr.TransactionSort) {
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
		case "timestamp":
			if a.Timestamp.Equal(b.Timestamp) {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return a.Timestamp.Before(b.Timestamp)
			}
			return a.Timestamp.After(b.Timestamp)

		case "amount":
			if a.Amount == b.Amount {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return a.Amount < b.Amount
			}
			return a.Amount > b.Amount

		case "brandname", "brand_name":
			if a.BrandName == b.BrandName {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return a.BrandName < b.BrandName
			}
			return a.BrandName > b.BrandName

		case "accountid", "account_id":
			if a.AccountID == b.AccountID {
				if asc {
					return a.ID < b.ID
				}
				return a.ID > b.ID
			}
			if asc {
				return a.AccountID < b.AccountID
			}
			return a.AccountID > b.AccountID

		default:
			// デフォルト: timestamp DESC, id DESC
			if a.Timestamp.Equal(b.Timestamp) {
				return a.ID > b.ID
			}
			return a.Timestamp.After(b.Timestamp)
		}
	}

	sort.SliceStable(items, less)
}
