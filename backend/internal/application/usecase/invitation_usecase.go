// backend/internal/application/usecase/invitation_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"

	memdom "narratives/internal/domain/member"
)

// ==============================
// Inbound Port (Query)
// ==============================

// InvitationQueryPort は、招待リンク（トークン）から
// InvitationInfo（memberId / companyId / assignedBrandIds / permissions / email）
// を取得するユースケースです。
type InvitationQueryPort interface {
	GetInvitationInfo(ctx context.Context, token string) (*memdom.InvitationInfo, error)
}

// ==============================
// Outbound Ports（Query / Command 共通）
// ==============================

// InvitationTokenRepository は、招待トークンと InvitationInfo の対応を扱うアウトバウンドポート。
//
// ResolveInvitationInfoByToken: token → InvitationInfo を解決
// CreateInvitationToken       : InvitationInfo に紐づく token を生成
type InvitationTokenRepository interface {
	ResolveInvitationInfoByToken(ctx context.Context, token string) (memdom.InvitationInfo, error)
	CreateInvitationToken(ctx context.Context, info memdom.InvitationInfo) (string, error)
}

// 招待メール送信用ポート
type InvitationMailerPort interface {
	SendInvitationEmail(ctx context.Context, toEmail string, token string, info memdom.InvitationInfo) error
}

// ==============================
// Usecase / Service (Query)
// ==============================

type InvitationService struct {
	invitationTokenRepo InvitationTokenRepository
	memberRepo          memdom.Repository
}

func NewInvitationService(
	invitationTokenRepo InvitationTokenRepository,
	memberRepo memdom.Repository,
) InvitationQueryPort {
	return &InvitationService{
		invitationTokenRepo: invitationTokenRepo,
		memberRepo:          memberRepo,
	}
}

// GET /api/invitation?token=...
func (s *InvitationService) GetInvitationInfo(
	ctx context.Context,
	token string,
) (*memdom.InvitationInfo, error) {

	t := strings.TrimSpace(token)
	if t == "" {
		return nil, memdom.ErrNotFound
	}

	// ★ 値型で返る → ポインタに変換して返却
	info, err := s.invitationTokenRepo.ResolveInvitationInfoByToken(ctx, t)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

// ==============================
// Inbound Port (Command)
// ==============================

// 招待メール送信ユースケース
type InvitationCommandPort interface {
	CreateInvitationAndSend(ctx context.Context, memberID string) (string, error)
}

// ==============================
// Command Service
// ==============================

type InvitationCommandService struct {
	invitationTokenRepo InvitationTokenRepository
	memberRepo          memdom.Repository
	mailer              InvitationMailerPort
}

func NewInvitationCommandService(
	invitationTokenRepo InvitationTokenRepository,
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
	memberID string,
) (string, error) {

	id := strings.TrimSpace(memberID)
	if id == "" {
		return "", fmt.Errorf("memberID is empty")
	}

	// 1) メンバー取得
	m, err := s.memberRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	email := strings.TrimSpace(m.Email)
	if email == "" {
		return "", fmt.Errorf("member email is empty")
	}

	// 2) Token に保存する InvitationInfo を構築
	info := memdom.InvitationInfo{
		MemberID:         m.ID,
		CompanyID:        m.CompanyID,
		AssignedBrandIDs: append([]string(nil), m.AssignedBrands...),
		Permissions:      append([]string(nil), m.Permissions...),
		Email:            email, // ★ 新規追加
	}

	// 3) Firestore に invitationToken を作成
	token, err := s.invitationTokenRepo.CreateInvitationToken(ctx, info)
	if err != nil {
		return "", err
	}

	// 4) 招待メール送信
	if err := s.mailer.SendInvitationEmail(ctx, email, token, info); err != nil {
		return "", err
	}

	return token, nil
}
