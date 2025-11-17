// backend/internal/application/usecase/invitation_usecase.go
package usecase

import (
	"context"
	"strings"

	memdom "narratives/internal/domain/member"
)

// ==============================
// Inbound Port
// ==============================

// InvitationQueryPort は、招待リンク（トークン）から
// InvitationInfo（memberId / companyId / brand / permissions）を取得するユースケースです。
type InvitationQueryPort interface {
	GetInvitationInfo(ctx context.Context, token string) (*memdom.InvitationInfo, error)
}

// ==============================
// Outbound Ports
// ==============================

// InvitationTokenRepository は、招待トークンから MemberID を解決するためのアウトバウンドポート。
// 実装は adapters/out/firestore などで行い、ヘキサゴナル的にここではインターフェースのみに依存します。
type InvitationTokenRepository interface {
	ResolveMemberIDByToken(ctx context.Context, token string) (string, error)
}

// memberRepo には memdom.Repository（= MemberRepositoryFS などの実装）がぶら下がる想定。
// adapters 側の具体型（MemberRepositoryFS）には依存しません。

// ==============================
// Usecase / Service
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
