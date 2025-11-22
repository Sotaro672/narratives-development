// backend/internal/app/invitation_wiring.go
package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/firestore"

	"narratives/internal/application/usecase"
	memdom "narratives/internal/domain/member"

	fsadapter "narratives/internal/adapters/out/firestore"
	mailadapter "narratives/internal/adapters/out/mail"
)

// ============================================================
// InvitationTokenRepository のアプリケーション層向けアダプタ
//   - Firestore 実装 (InvitationTokenRepositoryFS) をラップして
//     usecase.InvitationTokenRepository インターフェースを満たす
// ============================================================

// invitationTokenRepoAdapter は、Firestore ベースの InvitationTokenRepositoryFS を
// usecase.InvitationTokenRepository に適合させるアダプタです。
type invitationTokenRepoAdapter struct {
	fsRepo *fsadapter.InvitationTokenRepositoryFS
}

// ResolveInvitationInfoByToken は token から InvitationInfo を解決します。
func (a *invitationTokenRepoAdapter) ResolveInvitationInfoByToken(
	ctx context.Context,
	token string,
) (memdom.InvitationInfo, error) {
	it, err := a.fsRepo.FindByToken(ctx, token)
	if err != nil {
		return memdom.InvitationInfo{}, err
	}

	return memdom.InvitationInfo{
		MemberID:         strings.TrimSpace(it.MemberID),
		CompanyID:        strings.TrimSpace(it.CompanyID),
		AssignedBrandIDs: it.AssignedBrandIDs,
		Permissions:      it.Permissions,
	}, nil
}

// CreateInvitationToken は InvitationInfo を基に招待トークンを新規発行し、
// 発行された token 文字列を返します。
func (a *invitationTokenRepoAdapter) CreateInvitationToken(
	ctx context.Context,
	info memdom.InvitationInfo,
) (string, error) {
	// Firestore に保存する InvitationToken を作成
	t := memdom.InvitationToken{
		Token:            "", // 空なら Save 側で NewDoc() により採番される想定
		MemberID:         strings.TrimSpace(info.MemberID),
		CompanyID:        strings.TrimSpace(info.CompanyID),
		AssignedBrandIDs: info.AssignedBrandIDs,
		Permissions:      info.Permissions,
		// CreatedAt / UpdatedAt は FS 側の Save で補完される想定
	}

	saved, err := a.fsRepo.Save(ctx, t)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(saved.Token), nil
}

// ============================================================
// NewInvitationCommandServiceWithSendGrid
// ============================================================

// MemberRepo をどう作っているかはプロジェクトによりますが、
// ここでは memdom.Repository を引数でもらう形にしておくと安全です。
func NewInvitationCommandServiceWithSendGrid(
	fsClient *firestore.Client,
	memberRepo memdom.Repository,
) (usecase.InvitationCommandPort, error) {
	if fsClient == nil {
		return nil, fmt.Errorf("firestore client is nil")
	}

	// 1) Firestore ベースの招待トークンリポジトリ
	fsTokenRepo := fsadapter.NewInvitationTokenRepositoryFS(fsClient)

	// 2) Application 用の InvitationTokenRepository アダプタ
	tokenRepo := &invitationTokenRepoAdapter{
		fsRepo: fsTokenRepo,
	}

	// 3) SendGrid バックエンドの InvitationMailer
	mailer := mailadapter.NewInvitationMailerWithSendGrid()

	// 4) InvitationCommandService を SendGrid 付きで生成
	cmdSvc := usecase.NewInvitationCommandService(
		tokenRepo, // ← usecase.InvitationTokenRepository を満たすアダプタ
		memberRepo,
		mailer,
	)

	log.Printf("[app] InvitationCommandService with SendGrid initialized")

	return cmdSvc, nil
}
