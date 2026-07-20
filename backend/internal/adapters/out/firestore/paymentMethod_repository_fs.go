// backend/internal/adapters/out/firestore/paymentMethod_repository_fs.go
package firestore

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pm "narratives/internal/domain/paymentMethod"
)

// PaymentMethodRepositoryFSは、paymentMethod.RepositoryPortの
// Firestore実装です。
type PaymentMethodRepositoryFS struct {
	Client *firestore.Client
}

func NewPaymentMethodRepositoryFS(
	client *firestore.Client,
) *PaymentMethodRepositoryFS {
	return &PaymentMethodRepositoryFS{
		Client: client,
	}
}

func (r *PaymentMethodRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("paymentMethods")
}

// customerColは、SetupIntentフローで使用する
// userIdとstripeCustomerIdの対応関係を保存します。
func (r *PaymentMethodRepositoryFS) customerCol() *firestore.CollectionRef {
	return r.Client.Collection("paymentMethodCustomers")
}

// RepositoryPortを実装していることをコンパイル時に確認します。
var _ pm.RepositoryPort = (*PaymentMethodRepositoryFS)(nil)

// ============================================================
// Public API
// ============================================================

func (r *PaymentMethodRepositoryFS) GetByID(
	ctx context.Context,
	id string,
) (*pm.PaymentMethod, error) {
	if id == "" {
		return nil, pm.ErrNotFound
	}

	doc, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, pm.ErrNotFound
		}
		return nil, err
	}

	item, err := r.docToDomain(doc)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

// GetByUserは、ユーザーに紐づくすべてのPaymentMethodを返します。
//
// 並び順:
//
//  1. isDefaultの降順
//  2. updatedAtの降順
//  3. Document IDの降順
func (r *PaymentMethodRepositoryFS) GetByUser(
	ctx context.Context,
	userID string,
) ([]pm.PaymentMethod, error) {
	if userID == "" {
		return []pm.PaymentMethod{}, nil
	}

	query := r.col().
		Where("userId", "==", userID).
		OrderBy("isDefault", firestore.Desc).
		OrderBy("updatedAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var items []pm.PaymentMethod

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		item, err := r.docToDomain(doc)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}

func (r *PaymentMethodRepositoryFS) GetDefaultByUser(
	ctx context.Context,
	userID string,
) (*pm.PaymentMethod, error) {
	if userID == "" {
		return nil, pm.ErrNotFound
	}

	iter := r.col().
		Where("userId", "==", userID).
		Where("isDefault", "==", true).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, pm.ErrNotFound
		}
		return nil, err
	}

	item, err := r.docToDomain(doc)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *PaymentMethodRepositoryFS) GetByStripePaymentMethodID(
	ctx context.Context,
	stripePaymentMethodID string,
) (*pm.PaymentMethod, error) {
	if stripePaymentMethodID == "" {
		return nil, pm.ErrNotFound
	}

	iter := r.col().
		Where(
			"stripePaymentMethodId",
			"==",
			stripePaymentMethodID,
		).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, pm.ErrNotFound
		}
		return nil, err
	}

	item, err := r.docToDomain(doc)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

// GetStripeCustomerIDByUserは、ユーザーに対応する
// Stripe Customer IDを返します。
func (r *PaymentMethodRepositoryFS) GetStripeCustomerIDByUser(
	ctx context.Context,
	userID string,
) (string, error) {
	if userID == "" {
		return "", pm.ErrInvalidUserID
	}

	doc, err := r.customerCol().Doc(userID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", pm.ErrNotFound
		}
		return "", err
	}

	var raw struct {
		UserID           string    `firestore:"userId"`
		StripeCustomerID string    `firestore:"stripeCustomerId"`
		CreatedAt        time.Time `firestore:"createdAt"`
		UpdatedAt        time.Time `firestore:"updatedAt"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return "", err
	}
	if raw.UserID == "" || raw.StripeCustomerID == "" {
		return "", pm.ErrNotFound
	}

	return raw.StripeCustomerID, nil
}

// SaveStripeCustomerIDByUserは、ユーザーとStripe Customer IDの
// 対応関係を作成または更新します。
func (r *PaymentMethodRepositoryFS) SaveStripeCustomerIDByUser(
	ctx context.Context,
	userID string,
	stripeCustomerID string,
) error {
	if userID == "" {
		return pm.ErrInvalidUserID
	}
	if stripeCustomerID == "" {
		return pm.ErrInvalidStripeCustomerID
	}

	now := time.Now().UTC()
	ref := r.customerCol().Doc(userID)

	_, err := ref.Set(
		ctx,
		map[string]any{
			"userId":           userID,
			"stripeCustomerId": stripeCustomerID,
			"updatedAt":        now,
			"createdAt":        now,
		},
		firestore.MergeAll,
	)

	return err
}

// CreateはPaymentMethodを作成します。
//
// in.IsDefaultがtrueの場合は、次の処理を同一Transaction内で
// 原子的に実行します。
//
//  1. 同じユーザーの既存既定カードを取得する
//  2. 既存既定カードのisDefaultをfalseにする
//  3. 新しいPaymentMethodをisDefault=trueで作成する
//
// Transaction内のいずれかの処理が失敗した場合、
// 既存の既定設定を含むすべての変更がロールバックされます。
func (r *PaymentMethodRepositoryFS) Create(
	ctx context.Context,
	in pm.CreatePaymentMethodInput,
) (*pm.PaymentMethod, error) {
	if in.UserID == "" {
		return nil, pm.ErrInvalidUserID
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

	docID, err := newRandomPaymentMethodDocID(24)
	if err != nil {
		return nil, err
	}

	item, err := pm.New(
		docID,
		in.UserID,
		in.StripeCustomerID,
		in.StripePaymentMethodID,
		in.Brand,
		in.Last4,
		in.ExpMonth,
		in.ExpYear,
		in.CardholderName,
		in.IsDefault,
		createdAt,
		updatedAt,
	)
	if err != nil {
		return nil, err
	}

	paymentMethodRef := r.col().Doc(docID)
	customerRef := r.customerCol().Doc(in.UserID)

	err = r.Client.RunTransaction(
		ctx,
		func(
			_ context.Context,
			tx *firestore.Transaction,
		) error {
			var defaultRefs []*firestore.DocumentRef

			// Firestore Transactionでは、書き込みより先に
			// 必要な読み取りをすべて完了させます。
			if in.IsDefault {
				defaults, err :=
					r.defaultPaymentMethodRefsInTransaction(
						tx,
						in.UserID,
					)
				if err != nil {
					return err
				}

				defaultRefs = defaults
			}

			for _, defaultRef := range defaultRefs {
				if err := tx.Update(
					defaultRef,
					[]firestore.Update{
						{
							Path:  "isDefault",
							Value: false,
						},
						{
							Path:  "updatedAt",
							Value: now,
						},
					},
				); err != nil {
					return err
				}
			}

			if err := tx.Create(
				paymentMethodRef,
				r.domainToDocData(item),
			); err != nil {
				return err
			}

			// PaymentMethodの作成とCustomer IDの保存も
			// 同じTransactionに含めます。
			if err := tx.Set(
				customerRef,
				map[string]any{
					"userId":           in.UserID,
					"stripeCustomerId": in.StripeCustomerID,
					"updatedAt":        now,
					"createdAt":        now,
				},
				firestore.MergeAll,
			); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, pm.ErrConflict
		}
		return nil, err
	}

	return &item, nil
}

func (r *PaymentMethodRepositoryFS) Delete(
	ctx context.Context,
	id string,
) error {
	if id == "" {
		return pm.ErrNotFound
	}

	ref := r.col().Doc(id)

	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return pm.ErrNotFound
		}
		return err
	}

	_, err := ref.Delete(ctx)
	return err
}

// SetDefaultは、指定PaymentMethodをユーザーの既定に設定します。
//
// 次の処理を同一Transaction内で原子的に実行します。
//
//  1. 対象PaymentMethodを取得する
//  2. 対象が指定ユーザーの所有物であることを確認する
//  3. 同じユーザーの既存既定カードを取得する
//  4. 既存既定カードのisDefaultをfalseにする
//  5. 対象PaymentMethodのisDefaultをtrueにする
//
// Transaction内のいずれかの処理が失敗した場合、
// 既存の既定設定を含むすべての変更がロールバックされます。
func (r *PaymentMethodRepositoryFS) SetDefault(
	ctx context.Context,
	id string,
	userID string,
	updatedAt time.Time,
) (*pm.PaymentMethod, error) {
	if id == "" {
		return nil, pm.ErrNotFound
	}
	if userID == "" {
		return nil, pm.ErrInvalidUserID
	}

	now := updatedAt.UTC()
	if updatedAt.IsZero() {
		now = time.Now().UTC()
	}

	targetRef := r.col().Doc(id)

	var updatedItem pm.PaymentMethod

	err := r.Client.RunTransaction(
		ctx,
		func(
			_ context.Context,
			tx *firestore.Transaction,
		) error {
			// 対象DocumentをTransaction内で読み取ります。
			targetDoc, err := tx.Get(targetRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return pm.ErrNotFound
				}
				return err
			}

			current, err := r.docToDomain(targetDoc)
			if err != nil {
				return err
			}
			if current.UserID != userID {
				return pm.ErrNotFound
			}

			// 書き込みを始める前に、既存の既定カードを
			// Transaction内ですべて読み取ります。
			defaultRefs, err :=
				r.defaultPaymentMethodRefsInTransaction(
					tx,
					userID,
				)
			if err != nil {
				return err
			}

			candidate, err := pm.New(
				current.ID,
				current.UserID,
				current.StripeCustomerID,
				current.StripePaymentMethodID,
				current.Brand,
				current.Last4,
				current.ExpMonth,
				current.ExpYear,
				current.CardholderName,
				true,
				current.CreatedAt,
				now,
			)
			if err != nil {
				return err
			}

			updatedItem = candidate

			for _, defaultRef := range defaultRefs {
				// 対象自身が既に既定の場合は、一度falseにせず、
				// 最後のtrue更新だけを実行します。
				if defaultRef.ID == targetRef.ID {
					continue
				}

				if err := tx.Update(
					defaultRef,
					[]firestore.Update{
						{
							Path:  "isDefault",
							Value: false,
						},
						{
							Path:  "updatedAt",
							Value: now,
						},
					},
				); err != nil {
					return err
				}
			}

			if err := tx.Update(
				targetRef,
				[]firestore.Update{
					{
						Path:  "isDefault",
						Value: true,
					},
					{
						Path:  "updatedAt",
						Value: now,
					},
				},
			); err != nil {
				if status.Code(err) == codes.NotFound {
					return pm.ErrNotFound
				}
				return err
			}

			return nil
		},
	)
	if err != nil {
		if errors.Is(err, pm.ErrNotFound) ||
			status.Code(err) == codes.NotFound {
			return nil, pm.ErrNotFound
		}
		return nil, err
	}

	return &updatedItem, nil
}

// ============================================================
// Transaction helpers
// ============================================================

// defaultPaymentMethodRefsInTransactionは、指定ユーザーの
// isDefault=trueのDocument参照をTransaction内で取得します。
//
// このメソッドは、Transaction内で書き込みを開始する前に
// 呼び出す必要があります。
func (r *PaymentMethodRepositoryFS) defaultPaymentMethodRefsInTransaction(
	tx *firestore.Transaction,
	userID string,
) ([]*firestore.DocumentRef, error) {
	query := r.col().
		Where("userId", "==", userID).
		Where("isDefault", "==", true)

	iter := tx.Documents(query)
	defer iter.Stop()

	refs := make([]*firestore.DocumentRef, 0, 1)

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		refs = append(refs, doc.Ref)
	}

	return refs, nil
}

// ============================================================
// Conversion helpers
// ============================================================

func (r *PaymentMethodRepositoryFS) docToDomain(
	doc *firestore.DocumentSnapshot,
) (pm.PaymentMethod, error) {
	var raw struct {
		UserID                string    `firestore:"userId"`
		StripeCustomerID      string    `firestore:"stripeCustomerId"`
		StripePaymentMethodID string    `firestore:"stripePaymentMethodId"`
		Brand                 string    `firestore:"brand"`
		Last4                 string    `firestore:"last4"`
		ExpMonth              int       `firestore:"expMonth"`
		ExpYear               int       `firestore:"expYear"`
		CardholderName        string    `firestore:"cardholderName"`
		IsDefault             bool      `firestore:"isDefault"`
		CreatedAt             time.Time `firestore:"createdAt"`
		UpdatedAt             time.Time `firestore:"updatedAt"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return pm.PaymentMethod{}, err
	}

	createdAt := raw.CreatedAt.UTC()
	updatedAt := raw.UpdatedAt.UTC()

	if createdAt.IsZero() {
		createdAt = updatedAt
	}
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	if createdAt.IsZero() && updatedAt.IsZero() {
		now := time.Now().UTC()
		createdAt = now
		updatedAt = now
	}
	if updatedAt.Before(createdAt) {
		updatedAt = createdAt
	}

	return pm.New(
		doc.Ref.ID,
		raw.UserID,
		raw.StripeCustomerID,
		raw.StripePaymentMethodID,
		raw.Brand,
		raw.Last4,
		raw.ExpMonth,
		raw.ExpYear,
		raw.CardholderName,
		raw.IsDefault,
		createdAt,
		updatedAt,
	)
}

func (r *PaymentMethodRepositoryFS) domainToDocData(
	item pm.PaymentMethod,
) map[string]any {
	return map[string]any{
		"userId":                item.UserID,
		"stripeCustomerId":      item.StripeCustomerID,
		"stripePaymentMethodId": item.StripePaymentMethodID,
		"brand":                 item.Brand,
		"last4":                 item.Last4,
		"expMonth":              item.ExpMonth,
		"expYear":               item.ExpYear,
		"cardholderName":        item.CardholderName,
		"isDefault":             item.IsDefault,
		"createdAt":             item.CreatedAt.UTC(),
		"updatedAt":             item.UpdatedAt.UTC(),
	}
}

// newRandomPaymentMethodDocIDは、パディングなしの
// URL-safeなランダムDocument IDを生成します。
func newRandomPaymentMethodDocID(
	nBytes int,
) (string, error) {
	if nBytes <= 0 {
		nBytes = 24
	}

	randomBytes := make([]byte, nBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(
		randomBytes,
	), nil
}
