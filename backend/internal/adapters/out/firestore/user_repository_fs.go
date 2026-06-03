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

// =====================================================
// Firestore User Repository
// =====================================================
//
// ✅ Single source of truth: domain/user/entity.go
// - docId = uid (= user.ID)
// - field keys are unified to entity.go JSON tags:
//   first_name, first_name_kana, last_name_kana, last_name
//   createdAt, updatedAt, deletedAt
//
// - DeletedAt is allowed to be zero (not deleted)
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
// PaymentUsecase.UserRepoForPayment が要求する最小 contract。
// userID は order.UserID を想定する。
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

	emailValue, ok := data["email"]
	if !ok {
		return "", nil
	}

	email, ok := emailValue.(string)
	if !ok {
		return "", nil
	}

	return email, nil
}

// --------------------
// Write
// --------------------

// Create: users/{id} を作成。既存なら ErrConflict。
// ✅ docId = uid を必ず caller が渡す契約（RepositoryPort 準拠）
// ✅ createdAt/updatedAt は caller が渡した値を保存（usecase が server now を入れる前提）
// ✅ deletedAt は nil/zero/非zero を許容（entity.go に合わせる）
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

	deletedAt := time.Time{} // zero means "not deleted"
	if in.DeletedAt != nil {
		deletedAt = in.DeletedAt.UTC()
		if !deletedAt.IsZero() && deletedAt.Before(createdAt) {
			return nil, udom.ErrInvalidDeletedAt
		}
	}

	data := map[string]any{
		// ✅ entity.go times (camelCase)
		"createdAt": createdAt,
		"updatedAt": updatedAt,
		"deletedAt": deletedAt,
	}

	setIfNonEmpty := func(key string, p *string) {
		if p == nil {
			return
		}
		s := *p
		if s == "" {
			return
		}
		data[key] = s
	}

	// ✅ entity.go names (snake_case)
	setIfNonEmpty("first_name", in.FirstName)
	setIfNonEmpty("first_name_kana", in.FirstNameKana)
	setIfNonEmpty("last_name_kana", in.LastNameKana)
	setIfNonEmpty("last_name", in.LastName)

	ref := r.col().Doc(id)

	_, err := ref.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, udom.ErrConflict
		}
		return nil, err
	}

	return r.GetByID(ctx, id)
}

// Update: users/{id} を部分更新（nil は変更なし）
// - 空文字は「フィールド削除」
// - updatedAt は必須運用（nil/zero なら NOW を入れる）
func (r *UserRepositoryFS) Update(ctx context.Context, id string, in udom.UpdateUserInput) (*udom.User, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if id == "" {
		return nil, udom.ErrInvalidID
	}

	ref := r.col().Doc(id)

	// exists?
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return nil, udom.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	var updates []firestore.Update

	setStr := func(path string, p *string) {
		if p == nil {
			return
		}
		v := *p
		if v == "" {
			updates = append(updates, firestore.Update{Path: path, Value: firestore.Delete})
			return
		}
		updates = append(updates, firestore.Update{Path: path, Value: v})
	}

	// ✅ entity.go names (snake_case)
	setStr("first_name", in.FirstName)
	setStr("first_name_kana", in.FirstNameKana)
	setStr("last_name_kana", in.LastNameKana)
	setStr("last_name", in.LastName)

	// ✅ updatedAt (camelCase)
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updates = append(updates, firestore.Update{Path: "updatedAt", Value: in.UpdatedAt.UTC()})
	} else {
		updates = append(updates, firestore.Update{Path: "updatedAt", Value: time.Now().UTC()})
	}

	// ✅ deletedAt: nil なら変更なし / non-nil は zero も含めて反映
	if in.DeletedAt != nil {
		updates = append(updates, firestore.Update{Path: "deletedAt", Value: in.DeletedAt.UTC()})
	}

	if len(updates) == 0 {
		return r.GetByID(ctx, id)
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, udom.ErrNotFound
		}
		if status.Code(err) == codes.AlreadyExists {
			return nil, udom.ErrConflict
		}
		return nil, err
	}

	return r.GetByID(ctx, id)
}

func (r *UserRepositoryFS) Delete(ctx context.Context, id string) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}

	if id == "" {
		return udom.ErrInvalidID
	}

	ref := r.col().Doc(id)

	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return udom.ErrNotFound
	} else if err != nil {
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

// --------------------
// Firestore -> Domain
// --------------------

func docToUser(doc *firestore.DocumentSnapshot) (udom.User, error) {
	data := doc.Data()
	if data == nil {
		return udom.User{}, udom.ErrNotFound
	}

	getStrPtr := func(key string) *string {
		v, ok := data[key]
		if !ok {
			return nil
		}
		s, ok := v.(string)
		if !ok {
			return nil
		}
		t := s
		if t == "" {
			return nil
		}
		return &t
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
		FirstName:     getStrPtr("first_name"),
		FirstNameKana: getStrPtr("first_name_kana"),
		LastNameKana:  getStrPtr("last_name_kana"),
		LastName:      getStrPtr("last_name"),
		CreatedAt:     getTime("createdAt"),
		UpdatedAt:     getTime("updatedAt"),
		DeletedAt:     getTime("deletedAt"),
	}, nil
}
