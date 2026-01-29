// backend/internal/platform/di/console/adapter_invitation_token.go
package console

import (
	"context"
	"errors"

	fs "narratives/internal/adapters/out/firestore"
	memdom "narratives/internal/domain/member"
)

//
// ========================================
// InvitationTokenRepository 用アダプタ
// ========================================
//

type invitationTokenRepoAdapter struct {
	fsRepo *fs.InvitationTokenRepositoryFS
}

// ResolveInvitationInfoByToken は token から InvitationInfo を取得します。
func (a *invitationTokenRepoAdapter) ResolveInvitationInfoByToken(
	ctx context.Context,
	token string,
) (memdom.InvitationInfo, error) {
	if a.fsRepo == nil {
		return memdom.InvitationInfo{}, errors.New("invitationTokenRepoAdapter.ResolveInvitationInfoByToken: fsRepo is nil")
	}

	it, err := a.fsRepo.FindByToken(ctx, token)
	if err != nil {
		return memdom.InvitationInfo{}, err
	}

	return memdom.InvitationInfo{
		MemberID:         it.MemberID,
		CompanyID:        it.CompanyID,
		AssignedBrandIDs: it.AssignedBrandIDs,
		Permissions:      it.Permissions,
	}, nil
}

// CreateInvitationToken は InvitationInfo を受け取り、Firestore 側に招待トークンを作成して token 文字列を返します。
func (a *invitationTokenRepoAdapter) CreateInvitationToken(
	ctx context.Context,
	info memdom.InvitationInfo,
) (string, error) {
	if a.fsRepo == nil {
		return "", errors.New("invitationTokenRepoAdapter.CreateInvitationToken: fsRepo is nil")
	}
	return a.fsRepo.CreateInvitationToken(ctx, info)
}
