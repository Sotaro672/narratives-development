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

// ========== Public API (implements badom.RepositoryPort) ==========

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
		OrderBy("isDefault", firestore.Desc).
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

func (r *BillingAddressRepositoryFS) GetDefaultByUser(ctx context.Context, userID string) (*badom.BillingAddress, error) {
	q := r.col().
		Where("userId", "==", strings.TrimSpace(userID)).
		Where("isDefault", "==", true).
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

	// --- Sort ---
	orderField, orderDir := mapSort(sort)
	if orderField == "" {
		orderField = "updatedAt"
		orderDir = firestore.Desc
	}
	q = q.OrderBy(orderField, orderDir).OrderBy("id", firestore.Desc)

	// --- Paging (offset ベースの簡易実装) ---
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

	// Count: フィルタを適用した正確な件数はコストが高いので、
	// ここでは簡易に「取得件数のみ」を TotalCount に設定。
	// 必要であれば別途集計コレクションなどで対応してください。
	return badom.PageResult{
		Items:      items,
		TotalCount: len(items),
		TotalPages: number, // 厳密ではないがインターフェース維持用
		Page:       number,
		PerPage:    perPage,
	}, nil
}

func (r *BillingAddressRepositoryFS) Count(ctx context.Context, filter badom.Filter) (int, error) {
	// Firestore では COUNT 専用 API もあるが、ここでは全件走査のシンプル実装に留める。
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

func (r *BillingAddressRepositoryFS) Create(ctx context.Context, in badom.CreateBillingAddressInput) (*badom.BillingAddress, error) {
	now := time.Now().UTC()

	// Firestore では ID を自動採番
	ref := r.col().NewDoc()

	createdAt := in.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	updatedAt := in.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	ba := badom.BillingAddress{
		ID:            ref.ID,
		UserID:        strings.TrimSpace(in.UserID),
		NameOnAccount: trimPtr(in.NameOnAccount),
		BillingType:   strings.TrimSpace(in.BillingType),
		CardBrand:     trimPtr(in.CardBrand),
		CardLast4:     trimPtr(in.CardLast4),
		CardExpMonth:  in.CardExpMonth,
		CardExpYear:   in.CardExpYear,
		CardToken:     trimPtr(in.CardToken),
		PostalCode:    in.PostalCode,
		State:         trimPtr(in.State),
		City:          trimPtr(in.City),
		Street:        trimPtr(in.Street),
		Country:       trimPtr(in.Country),
		IsDefault:     in.IsDefault,
		CreatedAt:     createdAt.UTC(),
		UpdatedAt:     updatedAt.UTC(),
	}

	_, err := ref.Create(ctx, r.domainToDocData(ba))
	if err != nil {
		// Firestore ではユニーク制約はないので、必要であればアプリ側で検出する。
		return nil, err
	}

	// IsDefault=true の場合は SetDefault 相当の処理をここで行ってもよい（要件次第）。
	return &ba, nil
}

func (r *BillingAddressRepositoryFS) Update(ctx context.Context, id string, in badom.UpdateBillingAddressInput) (*badom.BillingAddress, error) {
	ref := r.col().Doc(id)

	// まず存在確認
	_, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, badom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var updates []firestore.Update

	if in.BillingType != nil {
		updates = append(updates, firestore.Update{Path: "billingType", Value: strings.TrimSpace(*in.BillingType)})
	}
	if in.NameOnAccount != nil {
		updates = append(updates, firestore.Update{Path: "nameOnAccount", Value: trimPtr(in.NameOnAccount)})
	}
	if in.CardBrand != nil {
		updates = append(updates, firestore.Update{Path: "cardBrand", Value: trimPtr(in.CardBrand)})
	}
	if in.CardLast4 != nil {
		updates = append(updates, firestore.Update{Path: "cardLast4", Value: trimPtr(in.CardLast4)})
	}
	if in.CardExpMonth != nil {
		updates = append(updates, firestore.Update{Path: "cardExpMonth", Value: in.CardExpMonth})
	}
	if in.CardExpYear != nil {
		updates = append(updates, firestore.Update{Path: "cardExpYear", Value: in.CardExpYear})
	}
	if in.CardToken != nil {
		updates = append(updates, firestore.Update{Path: "cardToken", Value: trimPtr(in.CardToken)})
	}
	if in.PostalCode != nil {
		updates = append(updates, firestore.Update{Path: "postalCode", Value: in.PostalCode})
	}
	if in.State != nil {
		updates = append(updates, firestore.Update{Path: "state", Value: trimPtr(in.State)})
	}
	if in.City != nil {
		updates = append(updates, firestore.Update{Path: "city", Value: trimPtr(in.City)})
	}
	if in.Street != nil {
		updates = append(updates, firestore.Update{Path: "street", Value: trimPtr(in.Street)})
	}
	if in.Country != nil {
		updates = append(updates, firestore.Update{Path: "country", Value: trimPtr(in.Country)})
	}
	if in.IsDefault != nil {
		updates = append(updates, firestore.Update{Path: "isDefault", Value: *in.IsDefault})
	}

	// updatedAt
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updates = append(updates, firestore.Update{Path: "updatedAt", Value: in.UpdatedAt.UTC()})
	} else {
		updates = append(updates, firestore.Update{Path: "updatedAt", Value: time.Now().UTC()})
	}

	if len(updates) == 0 {
		// no-op: 現状値を返す
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
	_, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return badom.ErrNotFound
	}
	if err != nil {
		return err
	}
	_, err = ref.Delete(ctx)
	return err
}

// SetDefault sets the specified billing address as default for its user (and unsets others).
func (r *BillingAddressRepositoryFS) SetDefault(ctx context.Context, id string) error {
	return r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// 対象ドキュメント取得
		ref := r.col().Doc(id)
		snap, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return badom.ErrNotFound
		}
		if err != nil {
			return err
		}

		var raw struct {
			UserID string `firestore:"userId"`
		}
		if err := snap.DataTo(&raw); err != nil {
			return err
		}
		userID := strings.TrimSpace(raw.UserID)
		if userID == "" {
			return fmt.Errorf("billing address has no userId")
		}

		// 同一 userId の既存 default を解除
		q := r.col().
			Where("userId", "==", userID).
			Where("isDefault", "==", true)

		iter := tx.Documents(q)
		for {
			doc, err := iter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return err
			}
			// 対象と同じIDは後で上書きするので一旦無視
			if doc.Ref.ID == id {
				continue
			}
			if err := tx.Update(doc.Ref, []firestore.Update{
				{Path: "isDefault", Value: false},
				{Path: "updatedAt", Value: time.Now().UTC()},
			}); err != nil {
				return err
			}
		}

		// 対象を default に設定
		if err := tx.Update(ref, []firestore.Update{
			{Path: "isDefault", Value: true},
			{Path: "updatedAt", Value: time.Now().UTC()},
		}); err != nil {
			return err
		}

		return nil
	})
}

// Reset (development/testing): 全削除
func (r *BillingAddressRepositoryFS) Reset(ctx context.Context) error {
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
	log.Printf("[firestore] Reset billing_addresses: deleted %d docs\n", count)
	return nil
}

// ========== Helpers ==========

func (r *BillingAddressRepositoryFS) docToDomain(doc *firestore.DocumentSnapshot) (badom.BillingAddress, error) {
	var raw struct {
		UserID        string    `firestore:"userId"`
		NameOnAccount *string   `firestore:"nameOnAccount"`
		BillingType   string    `firestore:"billingType"`
		CardBrand     *string   `firestore:"cardBrand"`
		CardLast4     *string   `firestore:"cardLast4"`
		CardExpMonth  *int      `firestore:"cardExpMonth"`
		CardExpYear   *int      `firestore:"cardExpYear"`
		CardToken     *string   `firestore:"cardToken"`
		PostalCode    *int      `firestore:"postalCode"`
		State         *string   `firestore:"state"`
		City          *string   `firestore:"city"`
		Street        *string   `firestore:"street"`
		Country       *string   `firestore:"country"`
		IsDefault     bool      `firestore:"isDefault"`
		CreatedAt     time.Time `firestore:"createdAt"`
		UpdatedAt     time.Time `firestore:"updatedAt"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return badom.BillingAddress{}, err
	}

	ba := badom.BillingAddress{
		ID:            doc.Ref.ID,
		UserID:        strings.TrimSpace(raw.UserID),
		NameOnAccount: trimPtr(raw.NameOnAccount),
		BillingType:   strings.TrimSpace(raw.BillingType),
		CardBrand:     trimPtr(raw.CardBrand),
		CardLast4:     trimPtr(raw.CardLast4),
		CardExpMonth:  raw.CardExpMonth,
		CardExpYear:   raw.CardExpYear,
		CardToken:     trimPtr(raw.CardToken),
		PostalCode:    raw.PostalCode,
		State:         trimPtr(raw.State),
		City:          trimPtr(raw.City),
		Street:        trimPtr(raw.Street),
		Country:       trimPtr(raw.Country),
		IsDefault:     raw.IsDefault,
		CreatedAt:     raw.CreatedAt.UTC(),
		UpdatedAt:     raw.UpdatedAt.UTC(),
	}
	return ba, nil
}

func (r *BillingAddressRepositoryFS) domainToDocData(ba badom.BillingAddress) map[string]any {
	data := map[string]any{
		"userId":      strings.TrimSpace(ba.UserID),
		"billingType": strings.TrimSpace(ba.BillingType),
		"isDefault":   ba.IsDefault,
		"createdAt":   ba.CreatedAt.UTC(),
		"updatedAt":   ba.UpdatedAt.UTC(),
	}

	if ba.NameOnAccount != nil {
		data["nameOnAccount"] = strings.TrimSpace(*ba.NameOnAccount)
	}
	if ba.CardBrand != nil {
		data["cardBrand"] = strings.TrimSpace(*ba.CardBrand)
	}
	if ba.CardLast4 != nil {
		data["cardLast4"] = strings.TrimSpace(*ba.CardLast4)
	}
	if ba.CardExpMonth != nil {
		data["cardExpMonth"] = *ba.CardExpMonth
	}
	if ba.CardExpYear != nil {
		data["cardExpYear"] = *ba.CardExpYear
	}
	if ba.CardToken != nil {
		data["cardToken"] = strings.TrimSpace(*ba.CardToken)
	}
	if ba.PostalCode != nil {
		data["postalCode"] = *ba.PostalCode
	}
	if ba.State != nil {
		data["state"] = strings.TrimSpace(*ba.State)
	}
	if ba.City != nil {
		data["city"] = strings.TrimSpace(*ba.City)
	}
	if ba.Street != nil {
		data["street"] = strings.TrimSpace(*ba.Street)
	}
	if ba.Country != nil {
		data["country"] = strings.TrimSpace(*ba.Country)
	}

	return data
}

// applyBillingAddressFilterToQuery applies a subset of Filter to Firestore query.
// Firestoreの制約上、範囲条件やLIKEは限定的に対応。
func applyBillingAddressFilterToQuery(q firestore.Query, f badom.Filter) firestore.Query {
	// IDs: Firestore では OR/IN 複合が難しいため、ここでは未対応または呼び出し側で分割想定。
	// UserIDs
	if len(f.UserIDs) == 1 {
		q = q.Where("userId", "==", strings.TrimSpace(f.UserIDs[0]))
	}

	// BillingTypes (単一 or IN 最大10件までなど、要件に応じて拡張可)
	if len(f.BillingTypes) == 1 {
		q = q.Where("billingType", "==", strings.TrimSpace(f.BillingTypes[0]))
	}

	// CardBrands (同上)
	if len(f.CardBrands) == 1 {
		q = q.Where("cardBrand", "==", strings.TrimSpace(f.CardBrands[0]))
	}

	// IsDefault
	if f.IsDefault != nil {
		q = q.Where("isDefault", "==", *f.IsDefault)
	}

	// PostalCode range / CreatedAt / UpdatedAt range / Like系は
	// Firestoreのクエリ制約により全てを安全に表現しづらいため、
	// 必要に応じて別インデックス設計 or アプリ側フィルタを検討。
	return q
}

func mapSort(sort badom.Sort) (field string, dir firestore.Direction) {
	col := strings.ToLower(string(sort.Column))
	switch col {
	case "createdat", "created_at":
		field = "createdAt"
	case "billingtype", "billing_type":
		field = "billingType"
	case "isdefault", "is_default":
		field = "isDefault"
	case "postalcode", "postal_code":
		field = "postalCode"
	case "updatedat", "updated_at":
		fallthrough
	default:
		// デフォルトは updatedAt
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
