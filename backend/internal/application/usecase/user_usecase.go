// backend/internal/application/usecase/user_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	userdom "narratives/internal/domain/user"
)

// UserRepo defines the minimal persistence port needed by UserUsecase.
//
// ✅ Firestore(UserRepositoryFS) に合わせてシグネチャを統一
type UserRepo interface {
	GetByID(ctx context.Context, id string) (*userdom.User, error)
	Create(ctx context.Context, in userdom.CreateUserInput) (*userdom.User, error)
	Update(ctx context.Context, id string, in userdom.UpdateUserInput) (*userdom.User, error)
	Delete(ctx context.Context, id string) error

	// optional: 表示用（lastName -> firstName）
	GetNameByID(ctx context.Context, id string) (string, error)
}

// UserUsecase orchestrates user operations.
type UserUsecase struct {
	repo UserRepo
	now  func() time.Time
}

func NewUserUsecase(repo UserRepo) *UserUsecase {
	return &UserUsecase{
		repo: repo,
		now:  time.Now,
	}
}

func (u *UserUsecase) WithNow(now func() time.Time) *UserUsecase {
	if now != nil {
		u.now = now
	}
	return u
}

func (u *UserUsecase) ensureRepo() error {
	if u == nil || u.repo == nil {
		return errors.New("user repo not configured")
	}
	return nil
}

// --------------------
// Queries
// --------------------

func (u *UserUsecase) GetByID(ctx context.Context, id string) (*userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

// Exists は Repo に Exists が無い前提で best-effort 実装。
// - ErrNotFound -> false,nil
func (u *UserUsecase) Exists(ctx context.Context, id string) (bool, error) {
	if err := u.ensureRepo(); err != nil {
		return false, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return false, userdom.ErrInvalidID
	}

	_, err := u.repo.GetByID(ctx, id)
	if err == nil {
		return true, nil
	}
	if err == userdom.ErrNotFound {
		return false, nil
	}
	return false, err
}

// ✅ NEW: 画面表示用（lastName -> firstName の順で返す想定）
func (u *UserUsecase) GetNameByID(ctx context.Context, id string) (string, error) {
	if err := u.ensureRepo(); err != nil {
		return "", err
	}
	name, err := u.repo.GetNameByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(name), nil
}

// --------------------
// Commands
// --------------------

// ✅ 互換: 既存コードが userdom.User を渡しているケースを救済
func (u *UserUsecase) CreateFromEntity(ctx context.Context, v userdom.User) (*userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	normalizeUserEntity(&v)

	now := u.now().UTC()
	in := userdom.CreateUserInput{
		FirstName:     trimPtr(v.FirstName),
		FirstNameKana: trimPtr(v.FirstNameKana),
		LastNameKana:  trimPtr(v.LastNameKana),
		LastName:      trimPtr(v.LastName),
	}

	// entity 側の CreatedAt/UpdatedAt が入っていれば優先、なければ NOW
	if !v.CreatedAt.IsZero() {
		t := v.CreatedAt.UTC()
		in.CreatedAt = &t
	} else {
		in.CreatedAt = &now
	}
	if !v.UpdatedAt.IsZero() {
		t := v.UpdatedAt.UTC()
		in.UpdatedAt = &t
	} else {
		in.UpdatedAt = &now
	}
	if !v.DeletedAt.IsZero() {
		t := v.DeletedAt.UTC()
		in.DeletedAt = &t
	}

	return u.repo.Create(ctx, in)
}

// ✅ 本命: CreateUserInput で作る（Firestore repo に合わせた正）
func (u *UserUsecase) Create(ctx context.Context, in userdom.CreateUserInput) (*userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}
	normalizeCreateInput(&in)

	now := u.now().UTC()
	if in.CreatedAt == nil {
		in.CreatedAt = &now
	}
	if in.UpdatedAt == nil {
		in.UpdatedAt = &now
	}

	return u.repo.Create(ctx, in)
}

// ✅ 互換: 既存 handler が Save(userdom.User) を呼んでいるので復活させる
//   - Firestore repo は Save を持たない前提なので
//     ここでは「存在すれば Update / 無ければ Create」を usecase で吸収する。
func (u *UserUsecase) Save(ctx context.Context, v userdom.User) (userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return userdom.User{}, err
	}

	normalizeUserEntity(&v)

	id := strings.TrimSpace(v.ID)
	if id == "" {
		return userdom.User{}, userdom.ErrInvalidID
	}

	// exists?
	exists, err := u.Exists(ctx, id)
	if err != nil {
		return userdom.User{}, err
	}

	if !exists {
		created, err := u.CreateFromEntity(ctx, v)
		if err != nil {
			return userdom.User{}, err
		}
		if created == nil {
			return userdom.User{}, userdom.ErrNotFound
		}
		// CreateFromEntity は repo.Create を呼ぶため、repo 実装が ID=UID を前提にしているなら、
		// handler 側で ID の取り扱いは従来通りにしてください。
		return *created, nil
	}

	// Update
	now := u.now().UTC()
	in := userdom.UpdateUserInput{
		FirstName:     trimPtr(v.FirstName),
		FirstNameKana: trimPtr(v.FirstNameKana),
		LastNameKana:  trimPtr(v.LastNameKana),
		LastName:      trimPtr(v.LastName),
		UpdatedAt:     &now,
	}
	if !v.DeletedAt.IsZero() {
		t := v.DeletedAt.UTC()
		in.DeletedAt = &t
	}

	updated, err := u.repo.Update(ctx, id, in)
	if err != nil {
		return userdom.User{}, err
	}
	if updated == nil {
		return userdom.User{}, userdom.ErrNotFound
	}
	return *updated, nil
}

// Update: UpdateUserInput を正規化して repo.Update に渡す
func (u *UserUsecase) Update(ctx context.Context, id string, in userdom.UpdateUserInput) (*userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, userdom.ErrInvalidID
	}

	normalizeUpdateInput(&in)

	// updatedAt 未指定なら NOW
	if in.UpdatedAt == nil {
		now := u.now().UTC()
		in.UpdatedAt = &now
	}

	return u.repo.Update(ctx, id, in)
}

func (u *UserUsecase) Delete(ctx context.Context, id string) error {
	if err := u.ensureRepo(); err != nil {
		return err
	}
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}

// Patch は handler 側で「部分フィールドだけ入った userdom.User を作って Save」しないための補助。
// - 現在値を GetByID で読み、入力で指定されたものだけ上書きし、UpdatedAt を更新して Update します。
func (u *UserUsecase) Patch(ctx context.Context, id string, in userdom.UpdateUserInput) (*userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, userdom.ErrInvalidID
	}

	// current
	cur, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, userdom.ErrNotFound
	}

	// merge（nil は維持）
	if in.FirstName != nil {
		cur.FirstName = trimPtr(in.FirstName)
	}
	if in.FirstNameKana != nil {
		cur.FirstNameKana = trimPtr(in.FirstNameKana)
	}
	if in.LastNameKana != nil {
		cur.LastNameKana = trimPtr(in.LastNameKana)
	}
	if in.LastName != nil {
		cur.LastName = trimPtr(in.LastName)
	}

	// deletedAt
	if in.DeletedAt != nil && !in.DeletedAt.IsZero() {
		t := in.DeletedAt.UTC()
		cur.DeletedAt = t
	}

	patch := userdom.UpdateUserInput{
		FirstName:     trimPtr(cur.FirstName),
		FirstNameKana: trimPtr(cur.FirstNameKana),
		LastNameKana:  trimPtr(cur.LastNameKana),
		LastName:      trimPtr(cur.LastName),
		UpdatedAt: func() *time.Time {
			t := u.now().UTC()
			return &t
		}(),
	}

	if !cur.DeletedAt.IsZero() {
		t := cur.DeletedAt.UTC()
		patch.DeletedAt = &t
	}

	normalizeUpdateInput(&patch)
	return u.repo.Update(ctx, id, patch)
}

// --------------------
// helpers
// --------------------

func normalizeCreateInput(in *userdom.CreateUserInput) {
	if in == nil {
		return
	}
	in.FirstName = trimPtr(in.FirstName)
	in.FirstNameKana = trimPtr(in.FirstNameKana)
	in.LastNameKana = trimPtr(in.LastNameKana)
	in.LastName = trimPtr(in.LastName)
}

func normalizeUpdateInput(in *userdom.UpdateUserInput) {
	if in == nil {
		return
	}
	in.FirstName = trimPtr(in.FirstName)
	in.FirstNameKana = trimPtr(in.FirstNameKana)
	in.LastNameKana = trimPtr(in.LastNameKana)
	in.LastName = trimPtr(in.LastName)
}

func normalizeUserEntity(v *userdom.User) {
	if v == nil {
		return
	}
	v.ID = strings.TrimSpace(v.ID)
	v.FirstName = trimPtr(v.FirstName)
	v.FirstNameKana = trimPtr(v.FirstNameKana)
	v.LastNameKana = trimPtr(v.LastNameKana)
	v.LastName = trimPtr(v.LastName)
}
