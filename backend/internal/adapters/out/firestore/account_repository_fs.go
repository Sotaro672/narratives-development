// backend/internal/adapters/out/firestore/account_repository_fs.go
package firestore

import (
	"context"
	"errors"
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
// NewID
// ========================================
// 新しい Account ID を発行します。
// Firestore の自動IDに AccountIDPrefix を付与します。
func (r *AccountRepositoryFS) NewID(
	ctx context.Context,
) (string, error) {
	if r == nil || r.Client == nil {
		return "", errors.New("account: repository is nil")
	}

	ref := r.Client.Collection("accounts").NewDoc()
	if ref == nil || ref.ID == "" {
		return "", accdom.ErrInvalidID
	}

	return accdom.AccountIDPrefix + ref.ID, nil
}

// ========================================
// ListByCompanyID
// ========================================
// 指定 CompanyID に紐づくアカウント一覧を取得。
func (r *AccountRepositoryFS) ListByCompanyID(
	ctx context.Context,
	companyID string,
) ([]accdom.Account, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("account: repository is nil")
	}
	if companyID == "" {
		return nil, accdom.ErrInvalidCompanyID
	}

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

		a.ID = doc.Ref.ID
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
	if r == nil || r.Client == nil {
		return accdom.Account{}, errors.New("account: repository is nil")
	}
	if id == "" {
		return accdom.Account{}, accdom.ErrInvalidID
	}

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

	a.ID = doc.Ref.ID

	return a, nil
}

// ========================================
// Create
// ========================================
// 新しいアカウントを作成。
// Account は Brand とは独立して作成でき、
// Brand 側が AccountID を参照します。
func (r *AccountRepositoryFS) Create(
	ctx context.Context,
	a accdom.Account,
) (accdom.Account, error) {
	if r == nil || r.Client == nil {
		return accdom.Account{}, errors.New("account: repository is nil")
	}

	if a.ID == "" {
		id, err := r.NewID(ctx)
		if err != nil {
			return accdom.Account{}, err
		}
		a.ID = id
	}

	ref := r.Client.Collection("accounts").Doc(a.ID)

	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = a.CreatedAt
	}

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
	if r == nil || r.Client == nil {
		return accdom.Account{}, errors.New("account: repository is nil")
	}
	if id == "" {
		return accdom.Account{}, accdom.ErrInvalidID
	}

	ref := r.Client.Collection("accounts").Doc(id)

	current, err := r.GetByID(ctx, id)
	if err != nil {
		return accdom.Account{}, err
	}

	updates := []firestore.Update{}
	now := time.Now().UTC()

	if patch.CompanyID != nil {
		updates = append(updates, firestore.Update{
			Path:  "companyId",
			Value: *patch.CompanyID,
		})
	}
	if patch.StripeAccountID != nil {
		updates = append(updates, firestore.Update{
			Path:  "stripeAccountId",
			Value: *patch.StripeAccountID,
		})
	}
	if patch.MemberID != nil {
		updates = append(updates, firestore.Update{
			Path:  "memberId",
			Value: *patch.MemberID,
		})
	}
	if patch.BankName != nil {
		updates = append(updates, firestore.Update{
			Path:  "bankName",
			Value: *patch.BankName,
		})
	}
	if patch.BranchName != nil {
		updates = append(updates, firestore.Update{
			Path:  "branchName",
			Value: *patch.BranchName,
		})
	}
	if patch.AccountNumber != nil {
		updates = append(updates, firestore.Update{
			Path:  "accountNumber",
			Value: *patch.AccountNumber,
		})
	}
	if patch.AccountType != nil {
		updates = append(updates, firestore.Update{
			Path:  "accountType",
			Value: *patch.AccountType,
		})
	}
	if patch.Currency != nil {
		updates = append(updates, firestore.Update{
			Path:  "currency",
			Value: *patch.Currency,
		})
	}
	if patch.Status != nil {
		updates = append(updates, firestore.Update{
			Path:  "status",
			Value: *patch.Status,
		})
	}
	if patch.UpdatedBy != nil {
		updates = append(updates, firestore.Update{
			Path:  "updatedBy",
			Value: *patch.UpdatedBy,
		})
	}
	if patch.DeletedAt != nil {
		updates = append(updates, firestore.Update{
			Path:  "deletedAt",
			Value: *patch.DeletedAt,
		})
	}
	if patch.DeletedBy != nil {
		updates = append(updates, firestore.Update{
			Path:  "deletedBy",
			Value: *patch.DeletedBy,
		})
	}

	if len(updates) == 0 {
		return current, nil
	}

	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: now,
	})

	_, err = ref.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return accdom.Account{}, accdom.ErrNotFound
		}
		return accdom.Account{}, err
	}

	return r.GetByID(ctx, id)
}
