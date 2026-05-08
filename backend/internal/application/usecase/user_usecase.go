// backend/internal/application/usecase/user_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"

	userdom "narratives/internal/domain/user"
)

// ✅ domain/user/repository_port.go を正として採用
type UserRepo interface {
	GetByID(ctx context.Context, id string) (*userdom.User, error)
	List(ctx context.Context, filter userdom.Filter, page userdom.Page) (userdom.PageResult, error)

	GetNameByID(ctx context.Context, id string) (string, error)

	// docId = uid を caller が必ず渡す
	Create(ctx context.Context, id string, in userdom.CreateUserInput) (*userdom.User, error)
	Update(ctx context.Context, id string, in userdom.UpdateUserInput) (*userdom.User, error)
	Delete(ctx context.Context, id string) error
}

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
	if id == "" {
		return nil, userdom.ErrInvalidID
	}
	return u.repo.GetByID(ctx, id)
}

func (u *UserUsecase) List(ctx context.Context, filter userdom.Filter, page userdom.Page) (userdom.PageResult, error) {
	if err := u.ensureRepo(); err != nil {
		return userdom.PageResult{}, err
	}
	return u.repo.List(ctx, filter, page)
}

func (u *UserUsecase) GetNameByID(ctx context.Context, id string) (string, error) {
	if err := u.ensureRepo(); err != nil {
		return "", err
	}
	if id == "" {
		return "", userdom.ErrInvalidID
	}
	name, err := u.repo.GetNameByID(ctx, id)
	if err != nil {
		return "", err
	}
	return name, nil
}

// --------------------
// Commands
// --------------------

// Create: docId=uid で新規作成（既存なら ErrConflict）
// ✅ createdAt/updatedAt はサーバが決める（入力は上書き）
// ✅ deletedAt は caller 指定を尊重（nil/zero/非zero OK）
func (u *UserUsecase) Create(ctx context.Context, id string, in userdom.CreateUserInput) (*userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	if id == "" {
		return nil, userdom.ErrInvalidID
	}

	normalizeCreateInput(&in)

	now := u.now().UTC()
	in.CreatedAt = &now
	in.UpdatedAt = &now

	return u.repo.Create(ctx, id, in)
}

// Upsert: docId=uid で「無ければ Create / あれば Update」
// ✅ Upsert 専用 input は作らない（domain 追加不要）
func (u *UserUsecase) Upsert(ctx context.Context, id string, in userdom.CreateUserInput) (*userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	if id == "" {
		return nil, userdom.ErrInvalidID
	}

	normalizeCreateInput(&in)

	// exists?
	_, err := u.repo.GetByID(ctx, id)
	if err != nil {
		if err == userdom.ErrNotFound {
			now := u.now().UTC()
			in.CreatedAt = &now
			in.UpdatedAt = &now
			return u.repo.Create(ctx, id, in)
		}
		return nil, err
	}

	// Update
	now := u.now().UTC()
	uin := userdom.UpdateUserInput{
		FirstName:     in.FirstName,
		FirstNameKana: in.FirstNameKana,
		LastNameKana:  in.LastNameKana,
		LastName:      in.LastName,
		UpdatedAt:     &now,
		DeletedAt:     in.DeletedAt, // nil=変更なし / zero=not deleted へ戻す
	}
	return u.repo.Update(ctx, id, uin)
}

func (u *UserUsecase) Update(ctx context.Context, id string, in userdom.UpdateUserInput) (*userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	if id == "" {
		return nil, userdom.ErrInvalidID
	}

	normalizeUpdateInput(&in)

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
	if id == "" {
		return userdom.ErrInvalidID
	}
	return u.repo.Delete(ctx, id)
}

// --------------------
// input normalizers
// --------------------

func trimStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := *p
	if s == "" {
		return nil
	}
	return &s
}

func normalizeCreateInput(in *userdom.CreateUserInput) {
	if in == nil {
		return
	}
	in.FirstName = trimStrPtr(in.FirstName)
	in.FirstNameKana = trimStrPtr(in.FirstNameKana)
	in.LastNameKana = trimStrPtr(in.LastNameKana)
	in.LastName = trimStrPtr(in.LastName)
	// times are server-controlled in Create/Upsert
}

func normalizeUpdateInput(in *userdom.UpdateUserInput) {
	if in == nil {
		return
	}
	in.FirstName = trimStrPtr(in.FirstName)
	in.FirstNameKana = trimStrPtr(in.FirstNameKana)
	in.LastNameKana = trimStrPtr(in.LastNameKana)
	in.LastName = trimStrPtr(in.LastName)
}
