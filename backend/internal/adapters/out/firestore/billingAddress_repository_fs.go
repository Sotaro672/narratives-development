// backend/internal/adapters/out/firestore/billingAddress_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
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
	return r.Client.Collection("billing_addresses")
}

// Compile-time check
var _ badom.RepositoryPort = (*BillingAddressRepositoryFS)(nil)

// ========== Public API ==========

func (r *BillingAddressRepositoryFS) GetByID(ctx context.Context, id string) (*badom.BillingAddress, error) {
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

func (r *BillingAddressRepositoryFS) GetByUser(ctx context.Context, userID string) ([]badom.BillingAddress, error) {
	q := r.col().
		Where("userId", "==", strings.TrimSpace(userID)).
		OrderBy("updatedAt", firestore.Desc).
		OrderBy("id", firestore.Desc)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var list []badom.BillingAddress
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		ba, err := r.docToDomain(doc)
		if err != nil {
			return nil, err
		}
		list = append(list, ba)
	}
	return list, nil
}

// 旧仕様は isDefault を持っていたが、現ドメイン（billing_address.dart 入力準拠）では不要。
// 互換のため「最新1件」を default とみなして返す。
func (r *BillingAddressRepositoryFS) GetDefaultByUser(ctx context.Context, userID string) (*badom.BillingAddress, error) {
	q := r.col().
		Where("userId", "==", strings.TrimSpace(userID)).
		OrderBy("updatedAt", firestore.Desc).
		OrderBy("id", firestore.Desc).
		Limit(1)

	iter := q.Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return nil, badom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	ba, err := r.docToDomain(doc)
	if err != nil {
		return nil, err
	}
	return &ba, nil
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
	q = q.OrderBy(orderField, orderDir).OrderBy("id", firestore.Desc)

	// Paging (offset ベース)
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
		TotalCount: len(items), // 簡易。正確な件数が必要なら別途集計。
		TotalPages: number,
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
func (r *BillingAddressRepositoryFS) Create(ctx context.Context, in badom.CreateBillingAddressInput) (*badom.BillingAddress, error) {
	now := time.Now().UTC()
	ref := r.col().NewDoc()

	// CreatedAt
	var createdAt time.Time
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	} else {
		createdAt = now
	}

	// UpdatedAt
	var updatedAt time.Time
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updatedAt = in.UpdatedAt.UTC()
	} else {
		updatedAt = createdAt
	}

	ba := badom.BillingAddress{
		ID:             ref.ID,
		UserID:         strings.TrimSpace(in.UserID),
		CardNumber:     strings.TrimSpace(in.CardNumber),
		CardholderName: strings.TrimSpace(in.CardholderName),
		CVC:            strings.TrimSpace(in.CVC),
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}

	if _, err := ref.Create(ctx, r.domainToDocData(ba)); err != nil {
		return nil, err
	}

	return &ba, nil
}

func (r *BillingAddressRepositoryFS) Update(ctx context.Context, id string, in badom.UpdateBillingAddressInput) (*badom.BillingAddress, error) {
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

	// no-op のときは現状値を返す（updatedAt は常に入るので基本 no-op にならないが念のため）
	if len(updates) == 0 {
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
		return &ba, nil
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, badom.ErrNotFound
		}
		return nil, err
	}

	doc, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}
	ba, err := r.docToDomain(doc)
	if err != nil {
		return nil, err
	}
	return &ba, nil
}

func (r *BillingAddressRepositoryFS) Delete(ctx context.Context, id string) error {
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
// 仕様上は「何もしない」で成功とする（既存呼び出しがあっても落とさない）。
func (r *BillingAddressRepositoryFS) SetDefault(ctx context.Context, id string) error {
	// id の存在だけ確認しておく（NotFound は返す）
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
// WriteBatch を使用せず、トランザクションを用いて全ドキュメント削除
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
		log.Printf("[firestore] Reset billing_addresses: no docs to delete\n")
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

	log.Printf("[firestore] Reset billing_addresses (transactional): deleted %d docs\n", deletedCount)
	return nil
}

// ========== Helpers ==========

func (r *BillingAddressRepositoryFS) docToDomain(doc *firestore.DocumentSnapshot) (badom.BillingAddress, error) {
	var raw struct {
		UserID         string    `firestore:"userId"`
		CardNumber     string    `firestore:"cardNumber"`
		CardholderName string    `firestore:"cardholderName"`
		CVC            string    `firestore:"cvc"`
		CreatedAt      time.Time `firestore:"createdAt"`
		UpdatedAt      time.Time `firestore:"updatedAt"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return badom.BillingAddress{}, err
	}

	ba := badom.BillingAddress{
		ID:             doc.Ref.ID,
		UserID:         strings.TrimSpace(raw.UserID),
		CardNumber:     strings.TrimSpace(raw.CardNumber),
		CardholderName: strings.TrimSpace(raw.CardholderName),
		CVC:            strings.TrimSpace(raw.CVC),
		CreatedAt:      raw.CreatedAt.UTC(),
		UpdatedAt:      raw.UpdatedAt.UTC(),
	}
	return ba, nil
}

func (r *BillingAddressRepositoryFS) domainToDocData(ba badom.BillingAddress) map[string]any {
	data := map[string]any{
		"userId":         strings.TrimSpace(ba.UserID),
		"cardNumber":     strings.TrimSpace(ba.CardNumber),
		"cardholderName": strings.TrimSpace(ba.CardholderName),
		"cvc":            strings.TrimSpace(ba.CVC),
		"createdAt":      ba.CreatedAt.UTC(),
		"updatedAt":      ba.UpdatedAt.UTC(),
	}
	return data
}

// 部分的に Filter を Firestore クエリへ反映
func applyBillingAddressFilterToQuery(q firestore.Query, f badom.Filter) firestore.Query {
	if len(f.UserIDs) == 1 {
		q = q.Where("userId", "==", strings.TrimSpace(f.UserIDs[0]))
	}
	// 旧: BillingTypes/CardBrands/IsDefault は現ドメインでは無いので無視（互換のため落とさない）
	_ = fscmn.TrimPtr // unused import guard (このファイルでは未使用になる可能性があるため参照)
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

var _ = fmt.Sprintf // fmt が未使用になった場合のガード（将来削除してOK）
