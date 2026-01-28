// backend/internal/application/productBlueprint/usecase/lifecycle.go
package productBlueprintUsecase

import (
	"context"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	productbpdom "narratives/internal/domain/productBlueprint"
)

// ------------------------------------------------------------
// SoftDelete / Restore
// ------------------------------------------------------------

func (u *ProductBlueprintUsecase) SoftDeleteWithModels(ctx context.Context, id string, deletedBy *string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return productbpdom.ErrInvalidID
	}

	cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))
	if cid == "" {
		return productbpdom.ErrInvalidCompanyID
	}

	pb, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	const softDeleteTTL = 90 * 24 * time.Hour
	pb.SoftDelete(now, deletedBy, softDeleteTTL)

	pb.CompanyID = cid

	saved, err := u.repo.Save(ctx, pb)
	if err != nil {
		return err
	}

	if u.historyRepo != nil {
		if err := u.historyRepo.SaveSnapshot(ctx, saved); err != nil {
			return err
		}
	}

	return nil
}

func (u *ProductBlueprintUsecase) RestoreWithModels(ctx context.Context, id string, restoredBy *string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return productbpdom.ErrInvalidID
	}

	cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx))
	if cid == "" {
		return productbpdom.ErrInvalidCompanyID
	}

	pb, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	pb.DeletedAt = nil
	pb.DeletedBy = nil
	pb.ExpireAt = nil

	pb.UpdatedAt = now
	if restoredBy != nil {
		if trimmed := strings.TrimSpace(*restoredBy); trimmed != "" {
			rb := trimmed
			pb.UpdatedBy = &rb
		} else {
			pb.UpdatedBy = nil
		}
	}

	pb.CompanyID = cid

	saved, err := u.repo.Save(ctx, pb)
	if err != nil {
		return err
	}

	if u.historyRepo != nil {
		if err := u.historyRepo.SaveSnapshot(ctx, saved); err != nil {
			return err
		}
	}

	return nil
}
