// backend/internal/adapters/out/firestore/billingAddress_repository_fs.go
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

	fscmn "narratives/internal/adapters/out/firestore/common"
	badom "narratives/internal/domain/billingAddress"
)

// BillingAddressRepositoryFS is the Firestore implementation of billingAddress.RepositoryPort.
type BillingAddressRepositoryFS struct {
	Client *firestore.Client
}

func NewBillingAddressRepositoryFS(client *firestore.Client) *BillingAddressRepositoryFS {
	return &BillingAddressRepositoryFS{Client: client}
}

func (r *BillingAddressRepositoryFS) col() *firestore.CollectionRef {
	// ✅ shippingAddresses と同様の命名に寄せる（camelCase）
	// 期待値: billingAddresses コレクション
	return r.Client.Collection("billingAddresses")
}

// Compile-time check
var _ badom.RepositoryPort = (*BillingAddressRepositoryFS)(nil)

// ========== Public API ==========

// docId = UID 前提
func (r *BillingAddressRepositoryFS) GetByID(ctx context.Context, id string) (*badom.BillingAddress, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, badom.ErrNotFound
	}

	doc, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, badom.ErrNotFound
		}
		return nil, err
	}

	ba, err := r.docToDomain(doc)
	if err != nil {
		return nil, err
	}
	return &ba, nil
}

// ✅ userId フィールドは廃止し、docId=UID で引く
func (r *BillingAddressRepositoryFS) GetByUser(ctx context.Context, userID string) ([]badom.BillingAddress, error) {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return []badom.BillingAddress{}, nil
	}

	ba, err := r.GetByID(ctx, uid)
	if err != nil {
		if errors.Is(err, badom.ErrNotFound) {
			return []badom.BillingAddress{}, nil
		}
		return nil, err
	}
	return []badom.BillingAddress{*ba}, nil
}

// 互換: 「最新1件」を返す（docId=UID なので実質 0 or 1）
func (r *BillingAddressRepositoryFS) GetDefaultByUser(ctx context.Context, userID string) (*badom.BillingAddress, error) {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return nil, badom.ErrNotFound
	}
	return r.GetByID(ctx, uid)
}

func (r *BillingAddressRepositoryFS) List(
	ctx context.Context,
	filter badom.Filter,
	sort badom.Sort,
	page badom.Page,
) (badom.PageResult, error) {
	q := r.col().Query
	q = applyBillingAddressFilterToQuery(q, filter)

	// Sort
	orderField, orderDir := mapSort(sort)
	if orderField == "" {
		orderField = "updatedAt"
		orderDir = firestore.Desc
	}

	// docId=UID なので id も DocumentID で安定ソート
	q = q.OrderBy(orderField, orderDir).OrderBy(firestore.DocumentID, firestore.Desc)

	// Paging (offset ベース)
	perPage, number, offset := fscmn.NormalizePage(page.Number, page.PerPage, 50, 200)
	if offset > 0 {
		q = q.Offset(offset)
	}
	q = q.Limit(perPage)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var items []badom.BillingAddress
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return badom.PageResult{}, err
		}
		ba, err := r.docToDomain(doc)
		if err != nil {
			return badom.PageResult{}, err
		}
		items = append(items, ba)
	}

	return badom.PageResult{
		Items:      items,
		TotalCount: len(items), // 簡易
		TotalPages: number,     // 簡易
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *BillingAddressRepositoryFS) Count(ctx context.Context, filter badom.Filter) (int, error) {
	q := r.col().Query
	q = applyBillingAddressFilterToQuery(q, filter)

	iter := q.Documents(ctx)
	defer iter.Stop()

	cnt := 0
	for {
		_, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}
		cnt++
	}
	return cnt, nil
}

// Create implements RepositoryPort.Create.
// ✅ docId = UID 前提（userId フィールドは保存しない）
func (r *BillingAddressRepositoryFS) Create(ctx context.Context, in badom.CreateBillingAddressInput) (*badom.BillingAddress, error) {
	now := time.Now().UTC()

	// ✅ 互換のため input.UserID を「docId(=UID)」として扱う
	uid := strings.TrimSpace(in.UserID)
	if uid == "" {
		return nil, badom.ErrInvalidUserID
	}

	ref := r.col().Doc(uid)

	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}
	updatedAt := createdAt
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updatedAt = in.UpdatedAt.UTC()
	}

	ba := badom.BillingAddress{
		ID:             uid, // ✅ docId=UID
		UserID:         "",  // ✅ フィールド廃止（レスポンスにも載せたくないなら空に）
		CardNumber:     strings.TrimSpace(in.CardNumber),
		CardholderName: strings.TrimSpace(in.CardholderName),
		CVC:            strings.TrimSpace(in.CVC),
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}

	if _, err := ref.Create(ctx, r.domainToDocData(ba)); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, badom.ErrConflict
		}
		return nil, err
	}

	return &ba, nil
}

// Update: docId=UID 前提（userId フィールドは保存しない）
func (r *BillingAddressRepositoryFS) Update(ctx context.Context, id string, in badom.UpdateBillingAddressInput) (*badom.BillingAddress, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, badom.ErrNotFound
	}

	ref := r.col().Doc(id)

	// 存在確認
	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, badom.ErrNotFound
		}
		return nil, err
	}

	var updates []firestore.Update

	if in.CardNumber != nil {
		updates = append(updates, firestore.Update{
			Path:  "cardNumber",
			Value: strings.TrimSpace(*in.CardNumber),
		})
	}
	if in.CardholderName != nil {
		updates = append(updates, firestore.Update{
			Path:  "cardholderName",
			Value: strings.TrimSpace(*in.CardholderName),
		})
	}
	if in.CVC != nil {
		updates = append(updates, firestore.Update{
			Path:  "cvc",
			Value: strings.TrimSpace(*in.CVC),
		})
	}

	// updatedAt
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updates = append(updates, firestore.Update{
			Path:  "updatedAt",
			Value: in.UpdatedAt.UTC(),
		})
	} else {
		updates = append(updates, firestore.Update{
			Path:  "updatedAt",
			Value: time.Now().UTC(),
		})
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, badom.ErrNotFound
		}
		return nil, err
	}

	doc, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, badom.ErrNotFound
		}
		return nil, err
	}

	ba, err := r.docToDomain(doc)
	if err != nil {
		return nil, err
	}

	// ✅ docId=UID なので userId は常に空
	ba.UserID = ""
	return &ba, nil
}

func (r *BillingAddressRepositoryFS) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return badom.ErrNotFound
	}

	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return badom.ErrNotFound
		}
		return err
	}
	_, err := ref.Delete(ctx)
	return err
}

// SetDefault: 旧インターフェース互換のため残すが、現ドメインでは不要。
func (r *BillingAddressRepositoryFS) SetDefault(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return badom.ErrNotFound
	}
	_, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return badom.ErrNotFound
		}
		return err
	}
	return nil
}

// WithTx: Firestore のトランザクション境界（シンプルラッパ）
func (r *BillingAddressRepositoryFS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.Client.RunTransaction(ctx, func(txCtx context.Context, _ *firestore.Transaction) error {
		return fn(txCtx)
	})
}

// Reset (development/testing):
func (r *BillingAddressRepositoryFS) Reset(ctx context.Context) error {
	iter := r.col().Documents(ctx)
	defer iter.Stop()

	var refs []*firestore.DocumentRef
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		refs = append(refs, doc.Ref)
	}

	if len(refs) == 0 {
		log.Printf("[firestore] Reset billingAddresses: no docs to delete\n")
		return nil
	}

	const chunkSize = 400
	deletedCount := 0

	for start := 0; start < len(refs); start += chunkSize {
		end := start + chunkSize
		if end > len(refs) {
			end = len(refs)
		}
		chunk := refs[start:end]

		err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, ref := range chunk {
				if err := tx.Delete(ref); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		deletedCount += len(chunk)
	}

	log.Printf("[firestore] Reset billingAddresses (transactional): deleted %d docs\n", deletedCount)
	return nil
}

// ========== Helpers ==========

func (r *BillingAddressRepositoryFS) docToDomain(doc *firestore.DocumentSnapshot) (badom.BillingAddress, error) {
	var raw struct {
		CardNumber     string    `firestore:"cardNumber"`
		CardholderName string    `firestore:"cardholderName"`
		CVC            string    `firestore:"cvc"`
		CreatedAt      time.Time `firestore:"createdAt"`
		UpdatedAt      time.Time `firestore:"updatedAt"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return badom.BillingAddress{}, err
	}

	return badom.BillingAddress{
		ID:             strings.TrimSpace(doc.Ref.ID), // ✅ docId=UID
		UserID:         "",                            // ✅ 保存しない
		CardNumber:     strings.TrimSpace(raw.CardNumber),
		CardholderName: strings.TrimSpace(raw.CardholderName),
		CVC:            strings.TrimSpace(raw.CVC),
		CreatedAt:      raw.CreatedAt.UTC(),
		UpdatedAt:      raw.UpdatedAt.UTC(),
	}, nil
}

func (r *BillingAddressRepositoryFS) domainToDocData(ba badom.BillingAddress) map[string]any {
	// ✅ userId は保存しない
	return map[string]any{
		"cardNumber":     strings.TrimSpace(ba.CardNumber),
		"cardholderName": strings.TrimSpace(ba.CardholderName),
		"cvc":            strings.TrimSpace(ba.CVC),
		"createdAt":      ba.CreatedAt.UTC(),
		"updatedAt":      ba.UpdatedAt.UTC(),
	}
}

// 部分的に Filter を Firestore クエリへ反映
// ✅ userId フィールドが無いので DocumentID で絞る
func applyBillingAddressFilterToQuery(q firestore.Query, f badom.Filter) firestore.Query {
	if len(f.UserIDs) == 1 {
		q = q.Where(firestore.DocumentID, "==", strings.TrimSpace(f.UserIDs[0]))
	}
	return q
}

func mapSort(sort badom.Sort) (field string, dir firestore.Direction) {
	col := strings.ToLower(string(sort.Column))
	switch col {
	case "createdat", "created_at":
		field = "createdAt"
	case "updatedat", "updated_at":
		fallthrough
	default:
		field = "updatedAt"
	}

	switch strings.ToUpper(string(sort.Order)) {
	case "ASC":
		dir = firestore.Asc
	default:
		dir = firestore.Desc
	}
	return
}
