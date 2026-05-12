// backend/internal/platform/di/console/adapter_auth.go
package console

import (
	"context"
	"errors"
	"strings"

	fsrepo "narratives/internal/adapters/out/firestore"
	companydom "narratives/internal/domain/company"
	memdom "narratives/internal/domain/member"
)

// authMemberRepoAdapter adapts memdom.Repository to auth.MemberRepository.
type authMemberRepoAdapter struct {
	repo memdom.Repository
}

func (a *authMemberRepoAdapter) GetByFirebaseUID(ctx context.Context, firebaseUID string) (*memdom.Member, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("authMemberRepoAdapter.GetByFirebaseUID: repo is nil")
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

func (a *authMemberRepoAdapter) Create(ctx context.Context, m *memdom.Member) error {
	if a == nil || a.repo == nil {
		return errors.New("authMemberRepoAdapter.Create: repo is nil")
	}
	if m == nil {
		return errors.New("authMemberRepoAdapter.Create: nil member")
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

// authCompanyRepoAdapter adapts CompanyRepositoryFS to auth.CompanyRepository.
type authCompanyRepoAdapter struct {
	repo *fsrepo.CompanyRepositoryFS
}

func (a *authCompanyRepoAdapter) NewID(ctx context.Context) (string, error) {
	if a == nil || a.repo == nil || a.repo.Client == nil {
		return "", errors.New("authCompanyRepoAdapter.NewID: repo or client is nil")
	}

	doc := a.repo.Client.Collection("companies").NewDoc()
	return doc.ID, nil
}

func (a *authCompanyRepoAdapter) Save(ctx context.Context, c *companydom.Company) error {
	if a == nil || a.repo == nil {
		return errors.New("authCompanyRepoAdapter.Save: repo is nil")
	}
	if c == nil {
		return errors.New("authCompanyRepoAdapter.Save: nil company")
	}

	saved, err := a.repo.Save(ctx, *c, nil)
	if err != nil {
		return err
	}

	*c = saved
	return nil
}
