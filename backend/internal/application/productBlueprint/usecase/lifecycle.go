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

	// 越境防止
	if strings.TrimSpace(pb.CompanyID) == "" || strings.TrimSpace(pb.CompanyID) != cid {
		return productbpdom.ErrForbidden
	}

	now := time.Now().UTC()
	const softDeleteTTL = 90 * 24 * time.Hour

	// domain 側の制約（printed なら ErrForbidden 等）を尊重
	if err := pb.SoftDelete(now, deletedBy, softDeleteTTL); err != nil {
		return err
	}

	// companyId は context を正として上書き（念のため）
	pb.CompanyID = cid

	// ✅ Patch に deleted 系が無いので Save を使う（port 側に Save を残す）
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

	// 越境防止
	if strings.TrimSpace(pb.CompanyID) == "" || strings.TrimSpace(pb.CompanyID) != cid {
		return productbpdom.ErrForbidden
	}

	now := time.Now().UTC()

	// domain 側の制約（printed なら ErrForbidden 等）を尊重
	if err := pb.Restore(now, restoredBy); err != nil {
		return err
	}

	// companyId は context を正として上書き（念のため）
	pb.CompanyID = cid

	// ✅ Patch に deleted 系が無いので Save を使う（port 側に Save を残す）
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
