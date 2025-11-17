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
// InvitationInfo（memberId / companyId / brand / permissions）を取得するユースケースです。
type InvitationQueryPort interface {
	GetInvitationInfo(ctx context.Context, token string) (*memdom.InvitationInfo, error)
}

// ==============================
// Outbound Ports（Query / Command 共通）
// ==============================

// InvitationTokenRepository は、招待トークンと MemberID の対応を扱うアウトバウンドポート。
// - ResolveMemberIDByToken: 招待URLアクセス時に token → memberID を解決する
// - CreateInvitationToken  : 招待メール送信時に memberID に紐づく新しい token を発行する
type InvitationTokenRepository interface {
	ResolveMemberIDByToken(ctx context.Context, token string) (string, error)
	CreateInvitationToken(ctx context.Context, memberID string) (string, error)
}

// InvitationMailerPort は「招待メールを送るためのアウトバウンドポート」です。
// adapters/out/mail.InvitationMailer がこれを実装する想定です。
type InvitationMailerPort interface {
	SendInvitationEmail(ctx context.Context, toEmail string, token string, info memdom.InvitationInfo) error
}

// memberRepo には memdom.Repository（= MemberRepositoryFS などの実装）がぶら下がる想定。
// adapters 側の具体型（MemberRepositoryFS）には依存しません。

// ==============================
// Usecase / Service (Query)
// ==============================

// InvitationService は InvitationQueryPort の実装です。
type InvitationService struct {
	invitationTokenRepo InvitationTokenRepository
	memberRepo          memdom.Repository
}

// NewInvitationService はヘキサゴナルアーキテクチャにおける DI 用コンストラクタです。
// - invitationTokenRepo: 招待トークン → MemberID 解決用のアウトバウンドポート
// - memberRepo:          Member を取得するためのリポジトリ（memdom.Repository）
func NewInvitationService(
	invitationTokenRepo InvitationTokenRepository,
	memberRepo memdom.Repository,
) InvitationQueryPort {
	return &InvitationService{
		invitationTokenRepo: invitationTokenRepo,
		memberRepo:          memberRepo,
	}
}

// GetInvitationInfo は、招待トークンから InvitationInfo を取得します。
// 1) トークンから MemberID を解決
// 2) MemberRepository から Member を取得
// 3) InvitationInfo に詰め替えて返却
func (s *InvitationService) GetInvitationInfo(
	ctx context.Context,
	token string,
) (*memdom.InvitationInfo, error) {
	t := strings.TrimSpace(token)
	if t == "" {
		// 空トークンは NotFound 扱い（バリデーションエラーにしたい場合は別エラーを定義）
		return nil, memdom.ErrNotFound
	}

	// 1) 招待トークンから MemberID を解決
	memberID, err := s.invitationTokenRepo.ResolveMemberIDByToken(ctx, t)
	if err != nil {
		return nil, err
	}

	// 2) MemberRepository から Member を取得
	m, err := s.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		return nil, err
	}

	// 3) InvitationInfo を組み立てて返却
	return &memdom.InvitationInfo{
		MemberID:         m.ID,
		CompanyID:        m.CompanyID,
		AssignedBrandIDs: m.AssignedBrands,
		Permissions:      m.Permissions,
	}, nil
}

// ==============================
// Inbound Port (Command)
// ==============================

// InvitationCommandPort は、「メンバーIDを指定して招待トークンを発行し、
// 招待メールを送信する」ユースケースのインバウンドポートです。
//
// handlers.MemberInvitationHandler がこのポートを呼び出します。
type InvitationCommandPort interface {
	// memberID に紐づく招待トークンを発行し、招待メールを送信する。
	// 正常終了時は発行された token を返す。
	CreateInvitationAndSend(ctx context.Context, memberID string) (string, error)
}

// InvitationCommandService は InvitationCommandPort の実装です。
type InvitationCommandService struct {
	invitationTokenRepo InvitationTokenRepository
	memberRepo          memdom.Repository
	mailer              InvitationMailerPort
}

// NewInvitationCommandService はコマンド用ユースケースのコンストラクタです。
//
//   - invitationTokenRepo: 招待トークンの発行・保存を行うリポジトリ
//   - memberRepo         : メンバー情報を取得するリポジトリ
//   - mailer             : 招待メール送信アウトバウンドポート
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

// CreateInvitationAndSend は、memberID に紐づく招待トークンを新規発行し、
// メール送信を行うコマンドユースケースです。
func (s *InvitationCommandService) CreateInvitationAndSend(
	ctx context.Context,
	memberID string,
) (string, error) {
	id := strings.TrimSpace(memberID)
	if id == "" {
		return "", fmt.Errorf("memberID is empty")
	}

	// 1) メンバー取得（メールアドレス等を使う）
	m, err := s.memberRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(m.Email) == "" {
		return "", fmt.Errorf("member email is empty")
	}

	// 2) 招待トークン発行
	token, err := s.invitationTokenRepo.CreateInvitationToken(ctx, id)
	if err != nil {
		return "", err
	}

	// 3) メール送信用の InvitationInfo を組み立て
	info := memdom.InvitationInfo{
		MemberID:         m.ID,
		CompanyID:        m.CompanyID,
		AssignedBrandIDs: m.AssignedBrands,
		Permissions:      m.Permissions,
	}

	// 4) メール送信
	if err := s.mailer.SendInvitationEmail(ctx, m.Email, token, info); err != nil {
		return "", err
	}

	return token, nil
}
