// backend/internal/adapters/out/firestore/mall/company_name_reader_fs.go
package mall

import (
	"context"

	"cloud.google.com/go/firestore"

	outfs "narratives/internal/adapters/out/firestore"
	companydom "narratives/internal/domain/company"
)

type CompanyNameReaderFS struct {
	repo *outfs.CompanyRepositoryFS
}

func NewCompanyNameReaderFS(client *firestore.Client) *CompanyNameReaderFS {
	return &CompanyNameReaderFS{
		repo: outfs.NewCompanyRepositoryFS(client),
	}
}

func (r *CompanyNameReaderFS) TryGetCompanyName(ctx context.Context, companyID string) (string, bool, error) {
	if r == nil || r.repo == nil || companyID == "" {
		return "", false, nil
	}

	c, err := r.repo.GetByID(ctx, companyID)
	if err != nil {
		if err == companydom.ErrNotFound {
			return "", false, nil
		}
		return "", false, err
	}

	if c.Name == "" {
		return "", false, nil
	}

	return c.Name, true, nil
}
