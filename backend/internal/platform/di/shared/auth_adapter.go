// backend/internal/platform/di/shared/auth_adapter.go
package shared

import (
	"context"
	"errors"
	"strings"

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
//
// New policy:
// - Firestore document ID and Firebase Auth UID are separated.
// - members/{autoDocID}.uid stores Firebase Auth UID.
// - Bootstrap lookup must use uid field, not document ID.
// - Bootstrap creation must create a normal auto-ID member document.
type AuthMemberRepoAdapter struct {
	repo memdom.Repository
}

func NewAuthMemberRepoAdapter(repo memdom.Repository) *AuthMemberRepoAdapter {
	return &AuthMemberRepoAdapter{repo: repo}
}

// GetByFirebaseUID returns a member whose uid field matches Firebase Auth UID.
func (a *AuthMemberRepoAdapter) GetByFirebaseUID(ctx context.Context, firebaseUID string) (*memdom.Member, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("shared.AuthMemberRepoAdapter.GetByFirebaseUID: repo is nil")
	}

	firebaseUID = strings.TrimSpace(firebaseUID)
	if firebaseUID == "" {
		return nil, memdom.ErrNotFound
	}

	v, err := a.repo.GetByFirebaseUID(ctx, firebaseUID)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

// Create creates a new member using repository default document ID behavior.
//
// The caller is responsible for setting m.UID when the member is already
// associated with a Firebase Auth user, such as the first company owner
// created through /auth/bootstrap.
func (a *AuthMemberRepoAdapter) Create(ctx context.Context, m *memdom.Member) error {
	if a == nil || a.repo == nil {
		return errors.New("shared.AuthMemberRepoAdapter.Create: repo is nil")
	}
	if m == nil {
		return errors.New("shared.AuthMemberRepoAdapter.Create: nil member")
	}

	m.UID = strings.TrimSpace(m.UID)
	m.Email = strings.TrimSpace(m.Email)
	m.FirstName = strings.TrimSpace(m.FirstName)
	m.LastName = strings.TrimSpace(m.LastName)
	m.FirstNameKana = strings.TrimSpace(m.FirstNameKana)
	m.LastNameKana = strings.TrimSpace(m.LastNameKana)
	m.CompanyID = strings.TrimSpace(m.CompanyID)
	m.Status = strings.TrimSpace(m.Status)

	saved, err := a.repo.Create(ctx, *m)
	if err != nil {
		return err
	}

	*m = saved
	return nil
}

// Save is kept as a convenience adapter for callers that still expect a
// simple save-style method. It does not assume document ID equals Firebase UID.
// Prefer Create for BootstrapService.
func (a *AuthMemberRepoAdapter) Save(ctx context.Context, m *memdom.Member) error {
	if a == nil || a.repo == nil {
		return errors.New("shared.AuthMemberRepoAdapter.Save: repo is nil")
	}
	if m == nil {
		return errors.New("shared.AuthMemberRepoAdapter.Save: nil member")
	}

	m.UID = strings.TrimSpace(m.UID)
	m.Email = strings.TrimSpace(m.Email)
	m.FirstName = strings.TrimSpace(m.FirstName)
	m.LastName = strings.TrimSpace(m.LastName)
	m.FirstNameKana = strings.TrimSpace(m.FirstNameKana)
	m.LastNameKana = strings.TrimSpace(m.LastNameKana)
	m.CompanyID = strings.TrimSpace(m.CompanyID)
	m.Status = strings.TrimSpace(m.Status)

	saved, err := a.repo.Save(ctx, *m, nil)
	if err != nil {
		return err
	}

	*m = saved
	return nil
}

// GetByID is kept only for legacy callers that still need document ID lookup.
// BootstrapService should not use this method.
func (a *AuthMemberRepoAdapter) GetByID(ctx context.Context, id string) (*memdom.Member, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("shared.AuthMemberRepoAdapter.GetByID: repo is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, memdom.ErrNotFound
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

	token = strings.TrimSpace(token)
	if token == "" {
		return memdom.InvitationInfo{}, memdom.ErrInvitationTokenNotFound
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
		Email:            it.Email,
	}, nil
}

func (a *InvitationTokenRepoAdapter) CreateInvitationToken(
	ctx context.Context,
	info memdom.InvitationInfo,
) (string, error) {
	if a == nil || a.fsRepo == nil {
		return "", errors.New("shared.InvitationTokenRepoAdapter.CreateInvitationToken: fsRepo is nil")
	}

	info.MemberID = strings.TrimSpace(info.MemberID)
	info.CompanyID = strings.TrimSpace(info.CompanyID)
	info.Email = strings.TrimSpace(info.Email)

	return a.fsRepo.CreateInvitationToken(ctx, info)
}

func (a *InvitationTokenRepoAdapter) ConsumeInvitationToken(
	ctx context.Context,
	token string,
) error {
	if a == nil || a.fsRepo == nil {
		return errors.New("shared.InvitationTokenRepoAdapter.ConsumeInvitationToken: fsRepo is nil")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return memdom.ErrInvitationTokenNotFound
	}

	return a.fsRepo.ConsumeInvitationToken(ctx, token)
}