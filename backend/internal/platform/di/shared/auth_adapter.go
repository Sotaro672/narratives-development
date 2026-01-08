// backend/internal/platform/di/shared/auth_adapter.go
package shared

import (
	"context"
	"errors"

	fs "narratives/internal/adapters/out/firestore"
	companydom "narratives/internal/domain/company"
	memdom "narratives/internal/domain/member"
)

//
// ========================================
// auth.BootstrapService 用アダプタ（shared）
// - mall/console/sns 等から再利用する想定
// ========================================
//

// AuthMemberRepoAdapter adapts memdom.Repository to auth.MemberRepository-like ports.
type AuthMemberRepoAdapter struct {
	repo memdom.Repository
}

func NewAuthMemberRepoAdapter(repo memdom.Repository) *AuthMemberRepoAdapter {
	return &AuthMemberRepoAdapter{repo: repo}
}

func (a *AuthMemberRepoAdapter) Save(ctx context.Context, m *memdom.Member) error {
	if a == nil || a.repo == nil {
		return errors.New("shared.AuthMemberRepoAdapter.Save: repo is nil")
	}
	if m == nil {
		return errors.New("shared.AuthMemberRepoAdapter.Save: nil member")
	}

	saved, err := a.repo.Save(ctx, *m, nil)
	if err != nil {
		return err
	}
	*m = saved
	return nil
}

func (a *AuthMemberRepoAdapter) GetByID(ctx context.Context, id string) (*memdom.Member, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("shared.AuthMemberRepoAdapter.GetByID: repo is nil")
	}
	v, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// AuthCompanyRepoAdapter adapts CompanyRepositoryFS to auth.CompanyRepository-like ports.
type AuthCompanyRepoAdapter struct {
	repo *fs.CompanyRepositoryFS
}

func NewAuthCompanyRepoAdapter(repo *fs.CompanyRepositoryFS) *AuthCompanyRepoAdapter {
	return &AuthCompanyRepoAdapter{repo: repo}
}

func (a *AuthCompanyRepoAdapter) NewID(ctx context.Context) (string, error) {
	_ = ctx
	if a == nil || a.repo == nil || a.repo.Client == nil {
		return "", errors.New("shared.AuthCompanyRepoAdapter.NewID: repo or client is nil")
	}
	doc := a.repo.Client.Collection("companies").NewDoc()
	return doc.ID, nil
}

func (a *AuthCompanyRepoAdapter) Save(ctx context.Context, c *companydom.Company) error {
	if a == nil || a.repo == nil {
		return errors.New("shared.AuthCompanyRepoAdapter.Save: repo is nil")
	}
	if c == nil {
		return errors.New("shared.AuthCompanyRepoAdapter.Save: nil company")
	}

	saved, err := a.repo.Save(ctx, *c, nil)
	if err != nil {
		return err
	}
	*c = saved
	return nil
}

//
// ========================================
// InvitationTokenRepository 用アダプタ（shared）
// ========================================
//

// InvitationTokenRepoAdapter adapts InvitationTokenRepositoryFS to member invitation ports.
type InvitationTokenRepoAdapter struct {
	fsRepo *fs.InvitationTokenRepositoryFS
}

func NewInvitationTokenRepoAdapter(fsRepo *fs.InvitationTokenRepositoryFS) *InvitationTokenRepoAdapter {
	return &InvitationTokenRepoAdapter{fsRepo: fsRepo}
}

func (a *InvitationTokenRepoAdapter) ResolveInvitationInfoByToken(
	ctx context.Context,
	token string,
) (memdom.InvitationInfo, error) {
	if a == nil || a.fsRepo == nil {
		return memdom.InvitationInfo{}, errors.New("shared.InvitationTokenRepoAdapter.ResolveInvitationInfoByToken: fsRepo is nil")
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

func (a *InvitationTokenRepoAdapter) CreateInvitationToken(
	ctx context.Context,
	info memdom.InvitationInfo,
) (string, error) {
	if a == nil || a.fsRepo == nil {
		return "", errors.New("shared.InvitationTokenRepoAdapter.CreateInvitationToken: fsRepo is nil")
	}
	return a.fsRepo.CreateInvitationToken(ctx, info)
}
