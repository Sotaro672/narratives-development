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

	companyID := CompanyIDFromContext(ctx)
	if companyID == "" {
		return "", fmt.Errorf("companyID is empty")
	}

	rec, err := findMemberByDocIDInCompany(ctx, s.memberRepo, companyID, memberDocID)
	if err != nil {
		return "", fmt.Errorf("find member by doc id failed: %w", err)
	}

	m := rec.Member
	if m.Email == "" {
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
		updateKey := m.UID
		if updateKey == "" {
			updateKey = rec.DocID
		}

		if _, err := s.memberRepo.Update(ctx, updateKey, memdom.MemberPatch{
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

	if info.CompanyID == "" {
		return fmt.Errorf("companyId is empty")
	}

	rec, err := findMemberByDocIDInCompany(ctx, s.memberRepo, info.CompanyID, info.MemberID)
	if err != nil {
		return fmt.Errorf("find member by invitation member id failed: %w", err)
	}

	companyID := info.CompanyID
	if companyID == "" {
		companyID = rec.Member.CompanyID
	}
	if companyID == "" {
		return fmt.Errorf("companyId is empty")
	}

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

	updateKey := rec.Member.UID
	if updateKey == "" {
		updateKey = rec.DocID
	}

	if _, err := s.memberRepo.Update(ctx, updateKey, patch); err != nil {
		return fmt.Errorf("update invited member failed: %w", err)
	}

	if err := s.invitationTokenRepo.ConsumeInvitationToken(ctx, in.Token); err != nil {
		return fmt.Errorf("consume invitation token failed: %w", err)
	}

	return nil
}

// ==============================
// Helpers
// ==============================

func findMemberByDocIDInCompany(
	ctx context.Context,
	repo memdom.Repository,
	companyID string,
	docID string,
) (memdom.Record, error) {
	if repo == nil {
		return memdom.Record{}, fmt.Errorf("member repository is not configured")
	}

	if companyID == "" || docID == "" {
		return memdom.Record{}, memdom.ErrNotFound
	}

	pageNumber := 1

	for {
		res, err := repo.ListByCompanyID(ctx, companyID, memdom.Filter{}, memdom.Page{
			Number:  pageNumber,
			PerPage: 200,
		})
		if err != nil {
			return memdom.Record{}, err
		}

		for _, item := range res.Items {
			if item.DocID == docID {
				return item, nil
			}
		}

		if len(res.Items) == 0 || pageNumber >= res.TotalPages {
			break
		}

		pageNumber++
	}

	return memdom.Record{}, memdom.ErrNotFound
}
