package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	common "narratives/internal/domain/common"
	memdom "narratives/internal/domain/member"
)

// -----------------------------------------------------------------------------
// Ports
// -----------------------------------------------------------------------------

// InvitationMailer は招待メール送信用ポート
type InvitationMailer interface {
	SendInvitationEmail(ctx context.Context, toEmail string, token string, info memdom.InvitationInfo) error
}

// -----------------------------------------------------------------------------
// Usecase 本体
// -----------------------------------------------------------------------------

type MemberUsecase struct {
	repo             memdom.Repository
	now              func() time.Time
	invitationMailer InvitationMailer
}

// 既存互換の最小構成
func NewMemberUsecase(repo memdom.Repository) *MemberUsecase {
	return &MemberUsecase{
		repo: repo,
		now:  time.Now,
	}
}

// Mailer 付き構成
func NewMemberUsecaseWithMailer(repo memdom.Repository, mailer InvitationMailer) *MemberUsecase {
	return &MemberUsecase{
		repo:             repo,
		now:              time.Now,
		invitationMailer: mailer,
	}
}

// -----------------------------------------------------------------------------
// Multitenancy: companyId 取得
// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------
// Queries
// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------
// Commands
// -----------------------------------------------------------------------------

type CreateMemberInput struct {
	ID             string
	FirstName      string
	LastName       string
	FirstNameKana  string
	LastNameKana   string
	Email          string
	Permissions    []string
	AssignedBrands []string

	CompanyID string
	Status    string
	CreatedAt *time.Time
}

func (u *MemberUsecase) Create(ctx context.Context, in CreateMemberInput) (memdom.Member, error) {
	createdAt := in.CreatedAt
	if createdAt == nil || createdAt.IsZero() {
		t := u.now().UTC()
		createdAt = &t
	}

	// context から companyId を強制上書き
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

		CompanyID: companyID,
		Status:    strings.TrimSpace(in.Status),

		CreatedAt: *createdAt,
		UpdatedAt: nil,
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

	CompanyID *string
	Status    *string
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

	// context の companyId を強制
	if cid := companyIDFromContext(ctx); cid != "" {
		current.CompanyID = cid
	}
	if in.Status != nil {
		current.Status = strings.TrimSpace(*in.Status)
	}

	return u.repo.Save(ctx, current, nil)
}

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

// -----------------------------------------------------------------------------
// Invitation (招待メール送信)
// -----------------------------------------------------------------------------

func (u *MemberUsecase) SendInvitation(ctx context.Context, memberID string) error {
	if u.invitationMailer == nil {
		return errors.New("invitation mailer is not configured")
	}

	m, err := u.repo.GetByID(ctx, strings.TrimSpace(memberID))
	if err != nil {
		return err
	}
	if strings.TrimSpace(m.Email) == "" {
		return fmt.Errorf("member %s has no email", m.ID)
	}

	token, err := generateInvitationToken()
	if err != nil {
		return fmt.Errorf("failed to generate invitation token: %w", err)
	}

	info := memdom.InvitationInfo{
		MemberID:         m.ID,
		CompanyID:        m.CompanyID,
		AssignedBrandIDs: append([]string(nil), m.AssignedBrands...),
		Permissions:      append([]string(nil), m.Permissions...),
		Email:            m.Email, // ★ 追加
	}

	return u.invitationMailer.SendInvitationEmail(ctx, m.Email, token, info)
}

// ----- Invitation token generator -----

func generateInvitationToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("INV_%x", buf), nil
}
