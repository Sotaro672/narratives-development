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
	if memberDocID == "" {
		return "", fmt.Errorf("memberDocID is empty")
	}

	m, err := s.memberRepo.GetByID(ctx, memberDocID)
	if err != nil {
		return "", fmt.Errorf("get member by doc id failed: %w", err)
	}
	if m.Email == "" {
		return "", fmt.Errorf("member email is empty")
	}

	info := memdom.InvitationInfo{
		MemberID:         memberDocID,
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
		m.Status = "inactive"
		if _, err := s.memberRepo.SaveByDocID(ctx, memberDocID, m, nil); err != nil {
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

	if info.MemberID == "" {
		return memdom.ErrNotFound
	}

	if info.Email != "" && !strings.EqualFold(info.Email, in.Email) {
		return fmt.Errorf("email_mismatch")
	}

	member, err := s.memberRepo.GetByID(ctx, info.MemberID)
	if err != nil {
		return fmt.Errorf("get member by invitation member doc id failed: %w", err)
	}

	// docId = firebaseUID 前提:
	// すでに同じ UID の member doc が存在する場合、
	// provisional memberDocID と同一 doc でない限り衝突とみなす。
	if _, err := s.memberRepo.GetByFirebaseUID(ctx, in.UID); err == nil {
		if info.MemberID != in.UID {
			return fmt.Errorf("firebase_uid_already_in_use")
		}
	} else if err != memdom.ErrNotFound {
		return fmt.Errorf("check firebase uid member failed: %w", err)
	}

	member.LastName = in.LastName
	member.LastNameKana = in.LastNameKana
	member.FirstName = in.FirstName
	member.FirstNameKana = in.FirstNameKana
	member.Email = in.Email
	member.Status = "active"

	if _, err := s.memberRepo.SaveByDocID(ctx, in.UID, member, nil); err != nil {
		return fmt.Errorf("save member by firebase uid failed: %w", err)
	}

	if info.MemberID != "" && info.MemberID != in.UID {
		if err := s.memberRepo.Delete(ctx, info.MemberID); err != nil {
			return fmt.Errorf("delete provisional member failed: %w", err)
		}
	}

	if err := s.invitationTokenRepo.ConsumeInvitationToken(ctx, in.Token); err != nil {
		return fmt.Errorf("consume invitation token failed: %w", err)
	}

	return nil
}
