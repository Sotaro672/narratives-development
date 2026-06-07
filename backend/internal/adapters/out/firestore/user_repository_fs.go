// backend/internal/adapters/out/firestore/user_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	udom "narratives/internal/domain/user"
)

// RepositoryPort 実装チェック
var _ udom.RepositoryPort = (*UserRepositoryFS)(nil)

// =====================================================
// Firestore User Repository
// =====================================================
//
// Single source of truth: domain/user/entity.go / repository_port.go
//
// - docId = uid (= user.ID)
// - field keys:
//   first_name, first_name_kana, last_name_kana, last_name
//   createdAt, updatedAt, deletedAt
//
// - Create は users/{id} を新規作成する。既存なら ErrConflict
// - Update は users/{id} を部分更新する。存在しなければ ErrNotFound
// - Delete は users/{id} を削除する。存在しなければ ErrNotFound
// - Upsert は RepositoryPort に無いため実装しない
//
// - DeletedAt は nil/zero/non-zero を扱う
//   nil      = 未指定
//   zero     = not deleted
//   non-zero = soft deleted
// =====================================================

type UserRepositoryFS struct {
	Client *firestore.Client
}

func NewUserRepositoryFS(client *firestore.Client) *UserRepositoryFS {
	return &UserRepositoryFS{Client: client}
}

func (r *UserRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("users")
}

// --------------------
// Read
// --------------------

func (r *UserRepositoryFS) GetByID(ctx context.Context, id string) (*udom.User, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if id == "" {
		return nil, udom.ErrInvalidID
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, udom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	u, err := docToUser(snap)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// GetEmailByID returns users/{userID}.email for payment post-paid mail.
//
// RepositoryPort には含めない。
// PaymentUsecase.UserRepoForPayment が要求する最小 contract 用。
func (r *UserRepositoryFS) GetEmailByID(ctx context.Context, userID string) (string, error) {
	if r == nil || r.Client == nil {
		return "", errors.New("firestore client is nil")
	}

	if userID == "" {
		return "", udom.ErrInvalidID
	}

	snap, err := r.col().Doc(userID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return "", udom.ErrNotFound
	}
	if err != nil {
		return "", err
	}

	data := snap.Data()
	if data == nil {
		return "", udom.ErrNotFound
	}

	v, ok := data["email"]
	if !ok {
		return "", nil
	}

	email, ok := v.(string)
	if !ok {
		return "", nil
	}

	return email, nil
}

// --------------------
// Write
// --------------------

// Create creates users/{id}.
// RepositoryPort contract:
//
//	Create(ctx context.Context, id string, in CreateUserInput) (*User, error)
//
// - id は caller が必ず渡す
// - docId = uid
// - 既存 document があれば ErrConflict
// - CreatedAt / UpdatedAt が未指定なら repository 側で now を補完
func (r *UserRepositoryFS) Create(ctx context.Context, id string, in udom.CreateUserInput) (*udom.User, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if id == "" {
		return nil, udom.ErrInvalidID
	}

	now := time.Now().UTC()

	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	updatedAt := now
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updatedAt = in.UpdatedAt.UTC()
	}

	if updatedAt.Before(createdAt) {
		return nil, udom.ErrInvalidUpdatedAt
	}

	deletedAt := time.Time{}
	if in.DeletedAt != nil {
		deletedAt = in.DeletedAt.UTC()

		if !deletedAt.IsZero() && deletedAt.Before(createdAt) {
			return nil, udom.ErrInvalidDeletedAt
		}
	}

	data := map[string]any{
		"createdAt": createdAt,
		"updatedAt": updatedAt,
		"deletedAt": deletedAt,
	}

	setStringIfPresent := func(key string, p *string) {
		if p == nil {
			return
		}

		v := *p
		if v == "" {
			return
		}

		data[key] = v
	}

	setStringIfPresent("first_name", in.FirstName)
	setStringIfPresent("first_name_kana", in.FirstNameKana)
	setStringIfPresent("last_name_kana", in.LastNameKana)
	setStringIfPresent("last_name", in.LastName)

	ref := r.col().Doc(id)

	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, udom.ErrConflict
		}

		return nil, err
	}

	return r.GetByID(ctx, id)
}

// Update updates users/{id} partially.
// RepositoryPort contract:
//
//	Update(ctx context.Context, id string, in UpdateUserInput) (*User, error)
//
// - nil は未指定
// - 空文字はフィールド削除
// - UpdatedAt が未指定なら repository 側で now を補完
// - DeletedAt は nil なら変更なし、non-nil なら zero も含めて反映
func (r *UserRepositoryFS) Update(ctx context.Context, id string, in udom.UpdateUserInput) (*udom.User, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if id == "" {
		return nil, udom.ErrInvalidID
	}

	ref := r.col().Doc(id)

	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, udom.ErrNotFound
		}

		return nil, err
	}

	updates := make([]firestore.Update, 0)

	setStringUpdate := func(path string, p *string) {
		if p == nil {
			return
		}

		v := *p
		if v == "" {
			updates = append(updates, firestore.Update{
				Path:  path,
				Value: firestore.Delete,
			})
			return
		}

		updates = append(updates, firestore.Update{
			Path:  path,
			Value: v,
		})
	}

	setStringUpdate("first_name", in.FirstName)
	setStringUpdate("first_name_kana", in.FirstNameKana)
	setStringUpdate("last_name_kana", in.LastNameKana)
	setStringUpdate("last_name", in.LastName)

	updatedAt := time.Now().UTC()
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updatedAt = in.UpdatedAt.UTC()
	}

	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: updatedAt,
	})

	if in.DeletedAt != nil {
		updates = append(updates, firestore.Update{
			Path:  "deletedAt",
			Value: in.DeletedAt.UTC(),
		})
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, udom.ErrNotFound
		}

		return nil, err
	}

	return r.GetByID(ctx, id)
}

// Delete deletes users/{id}.
// RepositoryPort contract:
//
//	Delete(ctx context.Context, id string) error
func (r *UserRepositoryFS) Delete(ctx context.Context, id string) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}

	if id == "" {
		return udom.ErrInvalidID
	}

	ref := r.col().Doc(id)

	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return udom.ErrNotFound
		}

		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return udom.ErrNotFound
		}

		return err
	}

	return nil
}

// --------------------
// Firestore -> Domain
// --------------------

func docToUser(doc *firestore.DocumentSnapshot) (udom.User, error) {
	if doc == nil {
		return udom.User{}, udom.ErrNotFound
	}

	data := doc.Data()
	if data == nil {
		return udom.User{}, udom.ErrNotFound
	}

	getStringPtr := func(key string) *string {
		v, ok := data[key]
		if !ok {
			return nil
		}

		s, ok := v.(string)
		if !ok {
			return nil
		}

		if s == "" {
			return nil
		}

		return &s
	}

	getTime := func(key string) time.Time {
		v, ok := data[key]
		if !ok {
			return time.Time{}
		}

		t, ok := v.(time.Time)
		if !ok {
			return time.Time{}
		}

		return t.UTC()
	}

	return udom.User{
		ID:            doc.Ref.ID,
		FirstName:     getStringPtr("first_name"),
		FirstNameKana: getStringPtr("first_name_kana"),
		LastNameKana:  getStringPtr("last_name_kana"),
		LastName:      getStringPtr("last_name"),
		CreatedAt:     getTime("createdAt"),
		UpdatedAt:     getTime("updatedAt"),
		DeletedAt:     getTime("deletedAt"),
	}, nil
}
