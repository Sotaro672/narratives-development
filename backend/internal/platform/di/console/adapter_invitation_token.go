// backend/internal/platform/di/console/adapter_invitation_token.go
package console

import (
	"context"
	"errors"
	"strings"

	memdom "narratives/internal/domain/member"
)

// invitationTokenRepositoryAdapter adapts memdom.InvitationTokenRepository
// to the application invitation token repository contract.
type invitationTokenRepositoryAdapter struct {
	repo memdom.InvitationTokenRepository
}

func (a *invitationTokenRepositoryAdapter) ResolveInvitationInfoByToken(
	ctx context.Context,
	token string,
) (memdom.InvitationInfo, error) {
	if a == nil || a.repo == nil {
		return memdom.InvitationInfo{}, errors.New("invitationTokenRepositoryAdapter.ResolveInvitationInfoByToken: repo is nil")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return memdom.InvitationInfo{}, memdom.ErrInvitationTokenNotFound
	}

	return a.repo.ResolveInvitationInfoByToken(ctx, token)
}

func (a *invitationTokenRepositoryAdapter) CreateInvitationToken(
	ctx context.Context,
	info memdom.InvitationInfo,
) (string, error) {
	if a == nil || a.repo == nil {
		return "", errors.New("invitationTokenRepositoryAdapter.CreateInvitationToken: repo is nil")
	}

	info.MemberID = strings.TrimSpace(info.MemberID)
	info.CompanyID = strings.TrimSpace(info.CompanyID)
	info.Email = strings.TrimSpace(info.Email)

	return a.repo.CreateInvitationToken(ctx, info)
}

func (a *invitationTokenRepositoryAdapter) ConsumeInvitationToken(
	ctx context.Context,
	token string,
) error {
	if a == nil || a.repo == nil {
		return errors.New("invitationTokenRepositoryAdapter.ConsumeInvitationToken: repo is nil")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return memdom.ErrInvitationTokenNotFound
	}

	return a.repo.ConsumeInvitationToken(ctx, token)
}
