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

// ─────────────────────────────────────────────────────────────
// Auth/Multitenancy: companyId の取得（ミドルウェアで注入された値を拾う）
// ─────────────────────────────────────────────────────────────

func companyIDFromContext(ctx context.Context) string {
	if v := ctx.Value("companyId"); v != nil {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	if v := ctx.Value("auth.companyId"); v != nil {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// ─────────────────────────────────────────────────────────────
// Queries
// ─────────────────────────────────────────────────────────────

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
	if cid := companyIDFromContext(ctx); cid != "" {
		f.CompanyID = cid
	}
	return u.repo.Count(ctx, f)
}

func (u *MemberUsecase) List(
	ctx context.Context,
	f memdom.Filter,
	s common.Sort,
	p common.Page,
) (common.PageResult[memdom.Member], error) {
	if cid := companyIDFromContext(ctx); cid != "" {
		f.CompanyID = cid
	}
	return u.repo.List(ctx, f, s, p)
}

// ─────────────────────────────────────────────────────────────
// Commands
// ─────────────────────────────────────────────────────────────

type CreateMemberInput struct {
	ID             string
	FirstName      string
	LastName       string
	FirstNameKana  string
	LastNameKana   string
	Email          string
	Permissions    []string
	AssignedBrands []string

	CompanyID   string
	Status      string
	FirebaseUID string // ★ 追加

	CreatedAt *time.Time
}

func (u *MemberUsecase) Create(ctx context.Context, in CreateMemberInput) (memdom.Member, error) {
	createdAt := in.CreatedAt
	if createdAt == nil || createdAt.IsZero() {
		t := u.now().UTC()
		createdAt = &t
	}

	// 強制 companyId
	cid := companyIDFromContext(ctx)
	companyID := strings.TrimSpace(in.CompanyID)
	if cid != "" {
		companyID = cid
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

		CompanyID:   companyID,
		Status:      strings.TrimSpace(in.Status),
		FirebaseUID: strings.TrimSpace(in.FirebaseUID), // ★ 追加

		CreatedAt: *createdAt,
		UpdatedAt: nil,
	}

	return u.repo.Create(ctx, m)
}

// ---------------------------- Update ----------------------------

type UpdateMemberInput struct {
	ID             string
	FirstName      *string
	LastName       *string
	FirstNameKana  *string
	LastNameKana   *string
	Email          *string
	Permissions    *[]string
	AssignedBrands *[]string
	CompanyID      *string
	Status         *string
	FirebaseUID    *string // ★ 追加
}

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

	// 強制 companyId
	if cid := companyIDFromContext(ctx); cid != "" {
		current.CompanyID = cid
	}

	// Status
	if in.Status != nil {
		current.Status = strings.TrimSpace(*in.Status)
	}

	// ★ Firebase UID 更新
	if in.FirebaseUID != nil {
		current.FirebaseUID = strings.TrimSpace(*in.FirebaseUID)
	}

	return u.repo.Save(ctx, current, nil)
}

// ---------------------------- Save ----------------------------

func (u *MemberUsecase) Save(ctx context.Context, m memdom.Member) (memdom.Member, error) {
	if m.CreatedAt.IsZero() {
		m.CreatedAt = u.now().UTC()
	}
	if cid := companyIDFromContext(ctx); cid != "" {
		m.CompanyID = cid
	}
	return u.repo.Save(ctx, m, nil)
}

func (u *MemberUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}

func (u *MemberUsecase) Reset(ctx context.Context) error {
	return u.repo.Reset(ctx)
}
