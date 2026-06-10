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
//   createdAt, updatedAt
//
// - Create は users/{id} を新規作成する。既存なら ErrConflict
// - Update は users/{id} を部分更新する。存在しなければ ErrNotFound
// - Delete は users/{id} を削除する。存在しなければ ErrNotFound
// - Upsert は RepositoryPort に無いため実装しない
//
// - first_name / first_name_kana / last_name_kana / last_name は必須
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
// - first_name / first_name_kana / last_name_kana / last_name は必須
func (r *UserRepositoryFS) Create(ctx context.Context, id string, in udom.CreateUserInput) (*udom.User, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	if id == "" {
		return nil, udom.ErrInvalidID
	}

	firstName, err := requiredString(in.FirstName, udom.ErrInvalidFirstName)
	if err != nil {
		return nil, err
	}

	firstNameKana, err := requiredString(in.FirstNameKana, udom.ErrInvalidFirstNameKana)
	if err != nil {
		return nil, err
	}

	lastNameKana, err := requiredString(in.LastNameKana, udom.ErrInvalidLastNameKana)
	if err != nil {
		return nil, err
	}

	lastName, err := requiredString(in.LastName, udom.ErrInvalidLastName)
	if err != nil {
		return nil, err
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

	u, err := udom.New(
		id,
		firstName,
		firstNameKana,
		lastNameKana,
		lastName,
		createdAt,
		updatedAt,
	)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"first_name":      u.FirstName,
		"first_name_kana": u.FirstNameKana,
		"last_name_kana":  u.LastNameKana,
		"last_name":       u.LastName,
		"createdAt":       u.CreatedAt,
		"updatedAt":       u.UpdatedAt,
	}

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
// - 空文字は必須項目違反として ErrInvalidXxx を返す
// - UpdatedAt が未指定なら repository 側で now を補完
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

	setStringUpdate := func(path string, p *string, invalidErr error) error {
		if p == nil {
			return nil
		}

		v := *p
		if v == "" {
			return invalidErr
		}

		if len([]rune(v)) > udom.MaxNameLength {
			return invalidErr
		}

		updates = append(updates, firestore.Update{
			Path:  path,
			Value: v,
		})

		return nil
	}

	if err := setStringUpdate("first_name", in.FirstName, udom.ErrInvalidFirstName); err != nil {
		return nil, err
	}
	if err := setStringUpdate("first_name_kana", in.FirstNameKana, udom.ErrInvalidFirstNameKana); err != nil {
		return nil, err
	}
	if err := setStringUpdate("last_name_kana", in.LastNameKana, udom.ErrInvalidLastNameKana); err != nil {
		return nil, err
	}
	if err := setStringUpdate("last_name", in.LastName, udom.ErrInvalidLastName); err != nil {
		return nil, err
	}

	updatedAt := time.Now().UTC()
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updatedAt = in.UpdatedAt.UTC()
	}

	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: updatedAt,
	})

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

	getString := func(key string) string {
		v, ok := data[key]
		if !ok {
			return ""
		}

		s, ok := v.(string)
		if !ok {
			return ""
		}

		return s
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

	return udom.New(
		doc.Ref.ID,
		getString("first_name"),
		getString("first_name_kana"),
		getString("last_name_kana"),
		getString("last_name"),
		getTime("createdAt"),
		getTime("updatedAt"),
	)
}

func requiredString(p *string, invalidErr error) (string, error) {
	if p == nil {
		return "", invalidErr
	}

	v := *p
	if v == "" {
		return "", invalidErr
	}

	if len([]rune(v)) > udom.MaxNameLength {
		return "", invalidErr
	}

	return v, nil
}
