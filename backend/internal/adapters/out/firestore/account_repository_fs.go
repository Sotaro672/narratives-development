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
	common "narratives/internal/domain/common"
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
// GetByID
// ========================================
// 指定 ID のアカウントを Firestore から取得。
func (r *AccountRepositoryFS) GetByID(ctx context.Context, id string) (accdom.Account, error) {
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
// Exists
// ========================================
// Firestore上に指定IDのアカウントが存在するかをチェック。
func (r *AccountRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	_, err := r.Client.Collection("accounts").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ========================================
// Create
// ========================================
// 新しいアカウントを作成。
// IDが空ならFirestoreの自動IDを採用。
func (r *AccountRepositoryFS) Create(ctx context.Context, a accdom.Account) (accdom.Account, error) {
	ref := r.Client.Collection("accounts").Doc(a.ID)
	if a.ID == "" {
		ref = r.Client.Collection("accounts").NewDoc()
		a.ID = ref.ID
	}

	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now

	_, err := ref.Set(ctx, a)
	if err != nil {
		return accdom.Account{}, err
	}
	return a, nil
}

// ========================================
// Save
// ========================================
// Upsert 相当: 既存ドキュメントを上書き、存在しなければ新規作成。
func (r *AccountRepositoryFS) Save(ctx context.Context, a accdom.Account, _ *common.SaveOptions) (accdom.Account, error) {
	ref := r.Client.Collection("accounts").Doc(a.ID)
	if a.ID == "" {
		ref = r.Client.Collection("accounts").NewDoc()
		a.ID = ref.ID
	}

	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now

	_, err := ref.Set(ctx, a)
	if err != nil {
		return accdom.Account{}, err
	}

	return a, nil
}

// ========================================
// Update
// ========================================
// 部分更新: Firestore の Update を使用。
func (r *AccountRepositoryFS) Update(ctx context.Context, id string, patch accdom.AccountPatch) (accdom.Account, error) {
	ref := r.Client.Collection("accounts").Doc(id)

	updates := []firestore.Update{}
	now := time.Now().UTC()

	if patch.BankName != nil {
		updates = append(updates, firestore.Update{Path: "bankName", Value: *patch.BankName})
	}
	if patch.BranchName != nil {
		updates = append(updates, firestore.Update{Path: "branchName", Value: *patch.BranchName})
	}
	if patch.AccountNumber != nil {
		updates = append(updates, firestore.Update{Path: "accountNumber", Value: *patch.AccountNumber})
	}
	if patch.AccountType != nil {
		updates = append(updates, firestore.Update{Path: "accountType", Value: *patch.AccountType})
	}
	if patch.Currency != nil {
		updates = append(updates, firestore.Update{Path: "currency", Value: *patch.Currency})
	}
	if patch.Status != nil {
		updates = append(updates, firestore.Update{Path: "status", Value: *patch.Status})
	}
	if patch.UpdatedBy != nil {
		updates = append(updates, firestore.Update{Path: "updatedBy", Value: *patch.UpdatedBy})
	}
	if patch.DeletedAt != nil {
		updates = append(updates, firestore.Update{Path: "deletedAt", Value: *patch.DeletedAt})
	}
	if patch.DeletedBy != nil {
		updates = append(updates, firestore.Update{Path: "deletedBy", Value: *patch.DeletedBy})
	}

	// 常に updatedAt を更新
	updates = append(updates, firestore.Update{Path: "updatedAt", Value: now})

	if len(updates) == 0 {
		// 更新対象なし → 現在のデータをそのまま返す
		return r.GetByID(ctx, id)
	}

	_, err := ref.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return accdom.Account{}, accdom.ErrNotFound
		}
		return accdom.Account{}, err
	}

	return r.GetByID(ctx, id)
}

// ========================================
// Delete
// ========================================
// 指定 ID のアカウントを削除。
// 存在しない場合は ErrNotFound を返す。
func (r *AccountRepositoryFS) Delete(ctx context.Context, id string) error {
	ref := r.Client.Collection("accounts").Doc(id)
	_, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return accdom.ErrNotFound
		}
		return err
	}

	_, err = ref.Delete(ctx)
	if err != nil {
		return err
	}

	return nil
}

// ========================================
// Count / List / ListByCursor
// ========================================

func (r *AccountRepositoryFS) Count(ctx context.Context, _ accdom.Filter) (int, error) {
	iter := r.Client.Collection("accounts").Documents(ctx)
	defer iter.Stop()

	count := 0
	for {
		_, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		count++
	}
	return count, nil
}

func (r *AccountRepositoryFS) List(ctx context.Context, _ accdom.Filter, _ common.Sort, _ common.Page) (common.PageResult[accdom.Account], error) {
	iter := r.Client.Collection("accounts").
		OrderBy("createdAt", firestore.Desc).
		Documents(ctx)
	defer iter.Stop()

	var items []accdom.Account
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[accdom.Account]{}, err
		}
		var a accdom.Account
		if err := doc.DataTo(&a); err == nil {
			if a.ID == "" {
				a.ID = doc.Ref.ID
			}
			items = append(items, a)
		}
	}
	return common.PageResult[accdom.Account]{
		Items:      items,
		TotalCount: len(items),
		Page:       1,
		PerPage:    len(items),
	}, nil
}

func (r *AccountRepositoryFS) ListByCursor(ctx context.Context, _ accdom.Filter, _ common.Sort, _ common.CursorPage) (common.CursorPageResult[accdom.Account], error) {
	// 必要になったらカーソル実装を追加
	return common.CursorPageResult[accdom.Account]{}, nil
}
