// backend/internal/application/usecase/invitation_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"

	memdom "narratives/internal/domain/member"
)

// ==============================
// Inbound Ports (Query)
// ==============================

// InvitationQueryPort は、招待リンク（トークン）から
// InvitationInfo（memberId / companyId / assignedBrandIds / permissions / email）
// を取得するユースケースです。
type InvitationQueryPort interface {
	GetInvitationInfo(ctx context.Context, token string) (*memdom.InvitationInfo, error)
}

// ==============================
// Inbound Ports (Command)
// ==============================

// 招待メール送信ユースケース
type InvitationCommandPort interface {
	CreateInvitationAndSend(ctx context.Context, memberDocID string) (string, error)
}

// 招待完了ユースケース
type InvitationCompletePort interface {
	CompleteInvitation(ctx context.Context, in CompleteInvitationInput) error
}

// ==============================
// Outbound Ports
// ==============================

// 招待メール送信用ポート
type InvitationMailerPort interface {
	SendInvitationEmail(ctx context.Context, toEmail string, token string, info memdom.InvitationInfo) error
}

// ==============================
// Query Service
// ==============================

type InvitationService struct {
	invitationTokenRepo memdom.InvitationTokenRepository
}

func NewInvitationService(
	invitationTokenRepo memdom.InvitationTokenRepository,
	_ memdom.Repository,
) InvitationQueryPort {
	return &InvitationService{
		invitationTokenRepo: invitationTokenRepo,
	}
}

// GET /api/invitation?token=...
// POST /api/invitation/validate
func (s *InvitationService) GetInvitationInfo(
	ctx context.Context,
	token string,
) (*memdom.InvitationInfo, error) {
	if s == nil || s.invitationTokenRepo == nil {
		return nil, fmt.Errorf("invitation token repository is not configured")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return nil, memdom.ErrInvitationTokenNotFound
	}

	info, err := s.invitationTokenRepo.ResolveInvitationInfoByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

// ==============================
// Command Service (Create & Send)
// ==============================

type InvitationCommandService struct {
	invitationTokenRepo memdom.InvitationTokenRepository
	memberRepo          memdom.Repository
	mailer              InvitationMailerPort
}

func NewInvitationCommandService(
	invitationTokenRepo memdom.InvitationTokenRepository,
	memberRepo memdom.Repository,
	mailer InvitationMailerPort,
) InvitationCommandPort {
	return &InvitationCommandService{
		invitationTokenRepo: invitationTokenRepo,
		memberRepo:          memberRepo,
		mailer:              mailer,
	}
}

func (s *InvitationCommandService) CreateInvitationAndSend(
	ctx context.Context,
	memberDocID string,
) (string, error) {
	if s == nil {
		return "", fmt.Errorf("invitation command service is nil")
	}
	if s.invitationTokenRepo == nil {
		return "", fmt.Errorf("invitation token repository is not configured")
	}
	if s.memberRepo == nil {
		return "", fmt.Errorf("member repository is not configured")
	}
	if s.mailer == nil {
		return "", fmt.Errorf("invitation mailer is not configured")
	}

	memberDocID = strings.TrimSpace(memberDocID)
	if memberDocID == "" {
		return "", fmt.Errorf("memberDocID is empty")
	}

	rec, err := s.memberRepo.GetByID(ctx, memberDocID)
	if err != nil {
		return "", fmt.Errorf("get member by id failed: %w", err)
	}

	m := rec.Member
	if strings.TrimSpace(m.Email) == "" {
		return "", fmt.Errorf("member email is empty")
	}

	info := memdom.InvitationInfo{
		MemberID:         rec.DocID,
		CompanyID:        m.CompanyID,
		AssignedBrandIDs: append([]string(nil), m.AssignedBrands...),
		Permissions:      append([]string(nil), m.Permissions...),
		Email:            m.Email,
	}

	token, err := s.invitationTokenRepo.CreateInvitationToken(ctx, info)
	if err != nil {
		return "", fmt.Errorf("create invitation token failed: %w", err)
	}

	if err := s.mailer.SendInvitationEmail(ctx, m.Email, token, info); err != nil {
		return "", fmt.Errorf("send invitation email failed: %w", err)
	}

	if !strings.EqualFold(m.Status, "active") {
		status := "inactive"
		if _, err := s.memberRepo.Update(ctx, rec.DocID, memdom.MemberPatch{
			Status: &status,
		}); err != nil {
			return "", fmt.Errorf("update member status after invitation failed: %w", err)
		}
	}

	return token, nil
}

// ==============================
// Command Service (Complete)
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

type InvitationCompleteService struct {
	invitationTokenRepo memdom.InvitationTokenRepository
	memberRepo          memdom.Repository
}

func NewInvitationCompleteService(
	invitationTokenRepo memdom.InvitationTokenRepository,
	memberRepo memdom.Repository,
) InvitationCompletePort {
	return &InvitationCompleteService{
		invitationTokenRepo: invitationTokenRepo,
		memberRepo:          memberRepo,
	}
}

func (s *InvitationCompleteService) CompleteInvitation(
	ctx context.Context,
	in CompleteInvitationInput,
) error {
	if s == nil {
		return fmt.Errorf("invitation complete service is nil")
	}
	if s.invitationTokenRepo == nil {
		return fmt.Errorf("invitation token repository is not configured")
	}
	if s.memberRepo == nil {
		return fmt.Errorf("member repository is not configured")
	}

	in.Token = strings.TrimSpace(in.Token)
	in.UID = strings.TrimSpace(in.UID)
	in.LastName = strings.TrimSpace(in.LastName)
	in.LastNameKana = strings.TrimSpace(in.LastNameKana)
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.FirstNameKana = strings.TrimSpace(in.FirstNameKana)
	in.Email = strings.TrimSpace(in.Email)

	if in.Token == "" || in.UID == "" {
		return fmt.Errorf("token_or_uid_required")
	}
	if in.LastName == "" || in.LastNameKana == "" || in.FirstName == "" || in.FirstNameKana == "" {
		return fmt.Errorf("name_fields_required")
	}
	if in.Email == "" {
		return fmt.Errorf("email_required")
	}

	info, err := s.invitationTokenRepo.ResolveInvitationInfoByToken(ctx, in.Token)
	if err != nil {
		return err
	}

	info.MemberID = strings.TrimSpace(info.MemberID)
	info.CompanyID = strings.TrimSpace(info.CompanyID)
	info.Email = strings.TrimSpace(info.Email)

	if info.MemberID == "" {
		return memdom.ErrNotFound
	}

	if info.Email != "" && !strings.EqualFold(info.Email, in.Email) {
		return fmt.Errorf("email_mismatch")
	}

	rec, err := s.memberRepo.GetByID(ctx, info.MemberID)
	if err != nil {
		return fmt.Errorf("get member by invitation member id failed: %w", err)
	}

	companyID := info.CompanyID
	if companyID == "" {
		companyID = strings.TrimSpace(rec.Member.CompanyID)
	}
	if companyID == "" {
		return fmt.Errorf("companyId is empty")
	}

	// repository port には GetByFirebaseUID がないため、
	// company scope 内の ListByCompanyID + Filter.UID で UID 重複を確認する。
	found, err := s.memberRepo.ListByCompanyID(ctx, companyID, memdom.Filter{
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

	if _, err := s.memberRepo.Update(ctx, rec.DocID, patch); err != nil {
		return fmt.Errorf("update invited member failed: %w", err)
	}

	if err := s.invitationTokenRepo.ConsumeInvitationToken(ctx, in.Token); err != nil {
		return fmt.Errorf("consume invitation token failed: %w", err)
	}

	return nil
}
