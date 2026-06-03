// backend/internal/application/usecase/invitation_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"

	invdom "narratives/internal/domain/invitation"
	memdom "narratives/internal/domain/member"
)

// ==============================
// Inbound Ports
// ==============================

// InvitationQueryPort は、招待リンク（トークン）から
// InvitationInfo（memberId / companyId / assignedBrandIds / permissions / email）
// を取得するユースケースです。
type InvitationQueryPort interface {
	GetInvitationInfo(ctx context.Context, token string) (*invdom.InvitationInfo, error)
}

// InvitationCommandPort は、招待トークン作成と招待メール送信を行うユースケースです。
type InvitationCommandPort interface {
	CreateInvitationAndSend(ctx context.Context, memberDocID string) (string, error)
}

// InvitationCompletePort は、招待完了処理を行うユースケースです。
type InvitationCompletePort interface {
	CompleteInvitation(ctx context.Context, in CompleteInvitationInput) error
}

// InvitationUsecasePort は、招待に関する Query / Command / Complete をまとめた入口です。
type InvitationUsecasePort interface {
	InvitationQueryPort
	InvitationCommandPort
	InvitationCompletePort
}

// ==============================
// Outbound Ports
// ==============================

// InvitationMailerPort は、招待メール送信用ポートです。
type InvitationMailerPort interface {
	SendInvitationEmail(ctx context.Context, toEmail string, token string, info invdom.InvitationInfo) error
}

// ==============================
// Usecase
// ==============================

type invitationUsecase struct {
	invitationTokenRepo invdom.Repository
	memberRepo          memdom.Repository
	mailer              InvitationMailerPort
}

// NewInvitationUsecase は、招待ユースケースの唯一の生成入口です。
// Query / Command / Complete で必要な依存をここに集中させます。
func NewInvitationUsecase(
	invitationTokenRepo invdom.Repository,
	memberRepo memdom.Repository,
	mailer InvitationMailerPort,
) InvitationUsecasePort {
	return &invitationUsecase{
		invitationTokenRepo: invitationTokenRepo,
		memberRepo:          memberRepo,
		mailer:              mailer,
	}
}

// ==============================
// Query
// ==============================

// GET /api/invitation?token=...
// POST /api/invitation/validate
func (u *invitationUsecase) GetInvitationInfo(
	ctx context.Context,
	token string,
) (*invdom.InvitationInfo, error) {
	if u == nil {
		return nil, fmt.Errorf("invitation usecase is nil")
	}
	if u.invitationTokenRepo == nil {
		return nil, fmt.Errorf("invitation token repository is not configured")
	}

	if token == "" {
		return nil, invdom.ErrInvitationTokenNotFound
	}

	info, err := u.invitationTokenRepo.ResolveInvitationInfoByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

// ==============================
// Command: Create & Send
// ==============================

func (u *invitationUsecase) CreateInvitationAndSend(
	ctx context.Context,
	memberDocID string,
) (string, error) {
	if u == nil {
		return "", fmt.Errorf("invitation usecase is nil")
	}
	if u.invitationTokenRepo == nil {
		return "", fmt.Errorf("invitation token repository is not configured")
	}
	if u.memberRepo == nil {
		return "", fmt.Errorf("member repository is not configured")
	}
	if u.mailer == nil {
		return "", fmt.Errorf("invitation mailer is not configured")
	}

	if memberDocID == "" {
		return "", fmt.Errorf("memberDocID is empty")
	}

	companyID := CompanyIDFromContext(ctx)
	if companyID == "" {
		return "", fmt.Errorf("companyID is empty")
	}

	rec, err := u.memberRepo.GetByID(ctx, memberDocID)
	if err != nil {
		return "", fmt.Errorf("find member by doc id failed: %w", err)
	}

	if rec.Member.CompanyID != companyID {
		return "", memdom.ErrNotFound
	}

	m := rec.Member
	if m.Email == "" {
		return "", fmt.Errorf("member email is empty")
	}

	info := invdom.InvitationInfo{
		MemberID:         rec.DocID,
		CompanyID:        m.CompanyID,
		AssignedBrandIDs: append([]string(nil), m.AssignedBrands...),
		Permissions:      append([]string(nil), m.Permissions...),
		Email:            m.Email,
	}

	token, err := u.invitationTokenRepo.CreateInvitationToken(ctx, info)
	if err != nil {
		return "", fmt.Errorf("create invitation token failed: %w", err)
	}

	if err := u.mailer.SendInvitationEmail(ctx, m.Email, token, info); err != nil {
		return "", fmt.Errorf("send invitation email failed: %w", err)
	}

	if !strings.EqualFold(m.Status, "active") {
		status := "inactive"

		if _, err := u.memberRepo.Update(ctx, rec.DocID, memdom.MemberPatch{
			Status: &status,
		}); err != nil {
			return "", fmt.Errorf("update member status after invitation failed: %w", err)
		}
	}

	return token, nil
}

// ==============================
// Command: Complete
// ==============================

type CompleteInvitationInput struct {
	Token         string
	UID           string
	LastName      string
	LastNameKana  string
	FirstName     string
	FirstNameKana string
	Email         string
}

func (u *invitationUsecase) CompleteInvitation(
	ctx context.Context,
	in CompleteInvitationInput,
) error {
	if u == nil {
		return fmt.Errorf("invitation usecase is nil")
	}
	if u.invitationTokenRepo == nil {
		return fmt.Errorf("invitation token repository is not configured")
	}
	if u.memberRepo == nil {
		return fmt.Errorf("member repository is not configured")
	}

	if in.Token == "" || in.UID == "" {
		return fmt.Errorf("token_or_uid_required")
	}
	if in.LastName == "" || in.LastNameKana == "" || in.FirstName == "" || in.FirstNameKana == "" {
		return fmt.Errorf("name_fields_required")
	}
	if in.Email == "" {
		return fmt.Errorf("email_required")
	}

	info, err := u.invitationTokenRepo.ResolveInvitationInfoByToken(ctx, in.Token)
	if err != nil {
		return err
	}

	if info.MemberID == "" {
		return memdom.ErrNotFound
	}

	if info.Email != "" && !strings.EqualFold(info.Email, in.Email) {
		return fmt.Errorf("email_mismatch")
	}

	if info.CompanyID == "" {
		return fmt.Errorf("companyId is empty")
	}

	rec, err := u.memberRepo.GetByID(ctx, info.MemberID)
	if err != nil {
		return fmt.Errorf("find member by invitation member id failed: %w", err)
	}

	if rec.Member.CompanyID != info.CompanyID {
		return memdom.ErrNotFound
	}

	companyID := info.CompanyID
	if companyID == "" {
		companyID = rec.Member.CompanyID
	}
	if companyID == "" {
		return fmt.Errorf("companyId is empty")
	}

	found, err := u.memberRepo.ListByCompanyID(ctx, companyID, memdom.Filter{
		UID: in.UID,
	}, memdom.Page{
		Number:  1,
		PerPage: 2,
	})
	if err != nil {
		return fmt.Errorf("check firebase uid member failed: %w", err)
	}

	for _, item := range found.Items {
		if item.DocID != rec.DocID {
			return fmt.Errorf("firebase_uid_already_in_use")
		}
	}

	status := "active"
	patch := memdom.MemberPatch{
		UID:            &in.UID,
		LastName:       &in.LastName,
		LastNameKana:   &in.LastNameKana,
		FirstName:      &in.FirstName,
		FirstNameKana:  &in.FirstNameKana,
		Email:          &in.Email,
		CompanyID:      &companyID,
		Status:         &status,
		Permissions:    &info.Permissions,
		AssignedBrands: &info.AssignedBrandIDs,
	}

	if _, err := u.memberRepo.Update(ctx, rec.DocID, patch); err != nil {
		return fmt.Errorf("update invited member failed: %w", err)
	}

	if err := u.invitationTokenRepo.ConsumeInvitationToken(ctx, in.Token); err != nil {
		return fmt.Errorf("consume invitation token failed: %w", err)
	}

	return nil
}
