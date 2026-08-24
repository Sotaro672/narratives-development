// backend/internal/adapters/out/firestore/account_repository_fs.go
package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	accdom "narratives/internal/domain/account"
)

// ========================================
// AccountRepositoryFS
// ========================================
// Firestore 実装。コレクション名は "accounts"。
type AccountRepositoryFS struct {
	Client *firestore.Client
}

// NewAccountRepositoryFS creates a new Firestore-backed account repository.
func NewAccountRepositoryFS(client *firestore.Client) *AccountRepositoryFS {
	return &AccountRepositoryFS{Client: client}
}

// ========================================
// ListByCompanyID
// ========================================
// 指定 CompanyID に紐づくアカウント一覧を取得。
func (r *AccountRepositoryFS) ListByCompanyID(
	ctx context.Context,
	companyID string,
) ([]accdom.Account, error) {
	iter := r.Client.Collection("accounts").
		Where("companyId", "==", companyID).
		Documents(ctx)
	defer iter.Stop()

	accounts := make([]accdom.Account, 0)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var a accdom.Account
		if err := doc.DataTo(&a); err != nil {
			return nil, err
		}

		// FirestoreのDocIDをIDに反映
		if a.ID == "" {
			a.ID = doc.Ref.ID
		}

		accounts = append(accounts, a)
	}

	return accounts, nil
}

// ========================================
// GetByID
// ========================================
// 指定 ID のアカウントを Firestore から取得。
func (r *AccountRepositoryFS) GetByID(
	ctx context.Context,
	id string,
) (accdom.Account, error) {
	doc, err := r.Client.Collection("accounts").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return accdom.Account{}, accdom.ErrNotFound
		}
		return accdom.Account{}, err
	}

	var a accdom.Account
	if err := doc.DataTo(&a); err != nil {
		return accdom.Account{}, err
	}

	// FirestoreのDocIDをIDに反映
	if a.ID == "" {
		a.ID = doc.Ref.ID
	}

	return a, nil
}

// ========================================
// GetByBrandID
// ========================================
// 指定 BrandID に紐づくアカウントを取得。
// 1 Brand = 1 Account を前提とする。
func (r *AccountRepositoryFS) GetByBrandID(
	ctx context.Context,
	brandID string,
) (accdom.Account, error) {
	iter := r.Client.Collection("accounts").
		Where("brandId", "==", brandID).
		Limit(2).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return accdom.Account{}, accdom.ErrNotFound
	}
	if err != nil {
		return accdom.Account{}, err
	}

	var a accdom.Account
	if err := doc.DataTo(&a); err != nil {
		return accdom.Account{}, err
	}

	// FirestoreのDocIDをIDに反映
	if a.ID == "" {
		a.ID = doc.Ref.ID
	}

	// 2件目が存在する場合は
	// 1 Brand = 1 Account の制約に違反している。
	_, err = iter.Next()
	if err == nil {
		return accdom.Account{}, accdom.ErrConflict
	}
	if err != iterator.Done {
		return accdom.Account{}, err
	}

	return a, nil
}

// ========================================
// Create
// ========================================
// 新しいアカウントを作成。
// IDが空ならFirestoreの自動IDを採用。
// 同一 BrandID に既存Accountが存在する場合は ErrConflict。
func (r *AccountRepositoryFS) Create(
	ctx context.Context,
	a accdom.Account,
) (accdom.Account, error) {
	// 1 Brand = 1 Account を保証する。
	if a.BrandID != "" {
		existing, err := r.GetByBrandID(ctx, a.BrandID)
		if err == nil && existing.ID != "" {
			return accdom.Account{}, accdom.ErrConflict
		}
		if err != nil && err != accdom.ErrNotFound {
			return accdom.Account{}, err
		}
	}

	ref := r.Client.Collection("accounts").Doc(a.ID)
	if a.ID == "" {
		ref = r.Client.Collection("accounts").NewDoc()
		a.ID = ref.ID
	}

	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now

	_, err := ref.Create(ctx, a)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return accdom.Account{}, accdom.ErrConflict
		}
		return accdom.Account{}, err
	}

	return a, nil
}

// ========================================
// Update
// ========================================
// 部分更新: Firestore の Update を使用。
func (r *AccountRepositoryFS) Update(
	ctx context.Context,
	id string,
	patch accdom.AccountPatch,
) (accdom.Account, error) {
	ref := r.Client.Collection("accounts").Doc(id)

	current, err := r.GetByID(ctx, id)
	if err != nil {
		return accdom.Account{}, err
	}

	// BrandIDを変更する場合、
	// 変更先Brandに別Accountが存在しないことを確認する。
	if patch.BrandID != nil &&
		*patch.BrandID != "" &&
		*patch.BrandID != current.BrandID {
		existing, err := r.GetByBrandID(
			ctx,
			*patch.BrandID,
		)
		if err == nil &&
			existing.ID != "" &&
			existing.ID != id {
			return accdom.Account{}, accdom.ErrConflict
		}
		if err != nil && err != accdom.ErrNotFound {
			return accdom.Account{}, err
		}
	}

	updates := []firestore.Update{}
	now := time.Now().UTC()

	if patch.CompanyID != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "companyId",
				Value: *patch.CompanyID,
			},
		)
	}
	if patch.BrandID != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "brandId",
				Value: *patch.BrandID,
			},
		)
	}
	if patch.StripeAccountID != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "stripeAccountId",
				Value: *patch.StripeAccountID,
			},
		)
	}
	if patch.MemberID != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "memberId",
				Value: *patch.MemberID,
			},
		)
	}
	if patch.BankName != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "bankName",
				Value: *patch.BankName,
			},
		)
	}
	if patch.BranchName != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "branchName",
				Value: *patch.BranchName,
			},
		)
	}
	if patch.AccountNumber != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "accountNumber",
				Value: *patch.AccountNumber,
			},
		)
	}
	if patch.AccountType != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "accountType",
				Value: *patch.AccountType,
			},
		)
	}
	if patch.Currency != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "currency",
				Value: *patch.Currency,
			},
		)
	}
	if patch.Status != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "status",
				Value: *patch.Status,
			},
		)
	}
	if patch.UpdatedBy != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "updatedBy",
				Value: *patch.UpdatedBy,
			},
		)
	}
	if patch.DeletedAt != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "deletedAt",
				Value: *patch.DeletedAt,
			},
		)
	}
	if patch.DeletedBy != nil {
		updates = append(
			updates,
			firestore.Update{
				Path:  "deletedBy",
				Value: *patch.DeletedBy,
			},
		)
	}

	if len(updates) == 0 {
		return current, nil
	}

	// 常に updatedAt を更新
	updates = append(
		updates,
		firestore.Update{
			Path:  "updatedAt",
			Value: now,
		},
	)

	_, err = ref.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return accdom.Account{}, accdom.ErrNotFound
		}
		return accdom.Account{}, err
	}

	return r.GetByID(ctx, id)
}
