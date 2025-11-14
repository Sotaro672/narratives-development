// backend/internal/application/usecase/member_usecase.go
package usecase

import (
	"context"
	"strings"
	"time"

	common "narratives/internal/domain/common"
	memdom "narratives/internal/domain/member"
)

type MemberUsecase struct {
	repo memdom.Repository
	now  func() time.Time
}

func NewMemberUsecase(repo memdom.Repository) *MemberUsecase {
	return &MemberUsecase{
		repo: repo,
		now:  time.Now,
	}
}

// Queries

func (u *MemberUsecase) GetByID(ctx context.Context, id string) (memdom.Member, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *MemberUsecase) GetByEmail(ctx context.Context, email string) (memdom.Member, error) {
	return u.repo.GetByEmail(ctx, strings.TrimSpace(email))
}

func (u *MemberUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

func (u *MemberUsecase) Count(ctx context.Context, f memdom.Filter) (int, error) {
	return u.repo.Count(ctx, f)
}

// ★戻り値型を common.PageResult[memdom.Member] に統一
func (u *MemberUsecase) List(
	ctx context.Context,
	f memdom.Filter,
	s common.Sort,
	p common.Page,
) (common.PageResult[memdom.Member], error) {
	return u.repo.List(ctx, f, s, p)
}

// Commands

type CreateMemberInput struct {
	ID             string
	FirstName      string
	LastName       string
	FirstNameKana  string
	LastNameKana   string
	Email          string
	Permissions    []string
	AssignedBrands []string
	// CreatedAt を指定しない場合は現在時刻
	CreatedAt *time.Time
}

func (u *MemberUsecase) Create(ctx context.Context, in CreateMemberInput) (memdom.Member, error) {
	createdAt := in.CreatedAt
	if createdAt == nil || createdAt.IsZero() {
		t := u.now().UTC()
		createdAt = &t
	}
	m := memdom.Member{
		ID:             strings.TrimSpace(in.ID),
		FirstName:      strings.TrimSpace(in.FirstName),
		LastName:       strings.TrimSpace(in.LastName),
		FirstNameKana:  strings.TrimSpace(in.FirstNameKana),
		LastNameKana:   strings.TrimSpace(in.LastNameKana),
		Email:          strings.TrimSpace(in.Email),
		Permissions:    dedupStrings(in.Permissions),
		AssignedBrands: dedupStrings(in.AssignedBrands),
		CreatedAt:      *createdAt,
		UpdatedAt:      nil,
	}
	return u.repo.Create(ctx, m)
}

type UpdateMemberInput struct {
	ID             string
	FirstName      *string
	LastName       *string
	FirstNameKana  *string
	LastNameKana   *string
	Email          *string
	Permissions    *[]string
	AssignedBrands *[]string
}

// Update は現在の Member を読み出して上書きし、repo.Save() に投げる。
// UpdatedAt は repo.Save/upsert 側で NOW() に更新される前提。
func (u *MemberUsecase) Update(ctx context.Context, in UpdateMemberInput) (memdom.Member, error) {
	current, err := u.repo.GetByID(ctx, strings.TrimSpace(in.ID))
	if err != nil {
		return memdom.Member{}, err
	}

	if in.FirstName != nil {
		current.FirstName = strings.TrimSpace(*in.FirstName)
	}
	if in.LastName != nil {
		current.LastName = strings.TrimSpace(*in.LastName)
	}
	if in.FirstNameKana != nil {
		current.FirstNameKana = strings.TrimSpace(*in.FirstNameKana)
	}
	if in.LastNameKana != nil {
		current.LastNameKana = strings.TrimSpace(*in.LastNameKana)
	}
	if in.Email != nil {
		current.Email = strings.TrimSpace(*in.Email)
	}
	if in.Permissions != nil {
		current.Permissions = dedupStrings(*in.Permissions)
	}
	if in.AssignedBrands != nil {
		current.AssignedBrands = dedupStrings(*in.AssignedBrands)
	}

	return u.repo.Save(ctx, current, nil)
}

func (u *MemberUsecase) Save(ctx context.Context, m memdom.Member) (memdom.Member, error) {
	// Save は Upsert。CreatedAt がゼロなら現在時刻を付与。
	if m.CreatedAt.IsZero() {
		m.CreatedAt = u.now().UTC()
	}
	return u.repo.Save(ctx, m, nil)
}

func (u *MemberUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}

func (u *MemberUsecase) Reset(ctx context.Context) error {
	return u.repo.Reset(ctx)
}
