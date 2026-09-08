// backend/internal/application/usecase/report_list.go
package usecase

import (
	"context"

	inventory "narratives/internal/domain/inventory"
	listdom "narratives/internal/domain/list"
	reportdom "narratives/internal/domain/report"
)

// ReportListInventoryReader owns the minimum inventory read required to resolve
// the ProductBlueprint, Brand, and Company behind a List.
type ReportListInventoryReader interface {
	GetByID(ctx context.Context, id string) (inventory.Mint, error)
}

// ReportListModerator owns Admin moderation of Lists.
// LIST + REMOVE in Report means suspending the List from Mall publication.
// It must not physically delete the List or its images.
type ReportListModerator interface {
	SuspendListByAdmin(
		ctx context.Context,
		input SuspendListByAdminInput,
	) error
}

type ReportListByAvatarInput struct {
	ListID   string
	AvatarID string
	Reason   reportdom.ReportReason
	Detail   string
}

type reportListTargetContext struct {
	List      listdom.List
	BrandID   string
	CompanyID string
}

func (u *ReportUsecase) ReportListByAvatar(
	ctx context.Context,
	input ReportListByAvatarInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.listRepo == nil || u.inventoryRepo == nil || u.productBlueprintRepo == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}
	if input.ListID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}
	if input.AvatarID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidReporterID
	}

	targetContext, err := u.resolveListTargetContext(ctx, input.ListID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if targetContext.List.Status != listdom.StatusListing {
		return reportdom.AddReportResult{}, reportdom.ErrCannotReportRemovedTarget
	}

	return u.addListReport(
		ctx,
		targetContext,
		input.AvatarID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReportUsecase) addListReport(
	ctx context.Context,
	targetContext reportListTargetContext,
	reporterAvatarID string,
	reason reportdom.ReportReason,
	detail string,
) (reportdom.AddReportResult, error) {
	now := u.now().UTC()
	target := targetContext.List

	reportCase, err := reportdom.NewReportCase(reportdom.NewReportCaseParams{
		TargetType:       reportdom.TargetTypeList,
		TargetID:         target.ID,
		TargetParentID:   target.ID,
		TargetAuthorID:   targetContext.BrandID,
		TargetAuthorType: reportdom.ActorTypeBrand,
		SnapshotTitle:    target.Title,
		SnapshotBody:     target.Description,
		SnapshotRating:   nil,
		CreatedAt:        now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	report, err := reportdom.NewReport(reportdom.NewReportParams{
		CaseID:       reportCase.ID,
		ReporterType: reportdom.ActorTypeAvatar,
		ReporterID:   reporterAvatarID,
		CompanyID:    "",
		Reason:       reason,
		Detail:       detail,
		CreatedAt:    now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	return u.reportRepo.AddReport(ctx, reportCase, report)
}

func (u *ReportUsecase) resolveListTargetContext(
	ctx context.Context,
	listID string,
) (reportListTargetContext, error) {
	if u == nil || u.listRepo == nil || u.inventoryRepo == nil || u.productBlueprintRepo == nil {
		return reportListTargetContext{}, ErrReportUsecaseNotConfigured
	}
	if listID == "" {
		return reportListTargetContext{}, reportdom.ErrInvalidTargetID
	}

	target, err := u.listRepo.GetByID(ctx, listID)
	if err != nil {
		return reportListTargetContext{}, err
	}
	if target.ID != listID {
		return reportListTargetContext{}, reportdom.ErrInvalidTargetID
	}
	if target.InventoryID == "" {
		return reportListTargetContext{}, reportdom.ErrInvalidTargetParentID
	}

	inventoryEntity, err := u.inventoryRepo.GetByID(ctx, target.InventoryID)
	if err != nil {
		return reportListTargetContext{}, err
	}
	if inventoryEntity.ProductBlueprintID == "" {
		return reportListTargetContext{}, reportdom.ErrInvalidTargetParentID
	}

	productBlueprint, err := u.productBlueprintRepo.GetByID(
		ctx,
		inventoryEntity.ProductBlueprintID,
	)
	if err != nil {
		return reportListTargetContext{}, err
	}
	if productBlueprint.ID != inventoryEntity.ProductBlueprintID {
		return reportListTargetContext{}, reportdom.ErrInvalidTargetParentID
	}
	if productBlueprint.BrandID == "" {
		return reportListTargetContext{}, reportdom.ErrInvalidTargetAuthorID
	}
	if productBlueprint.CompanyID == "" {
		return reportListTargetContext{}, reportdom.ErrInvalidCompanyID
	}

	return reportListTargetContext{
		List:      target,
		BrandID:   productBlueprint.BrandID,
		CompanyID: productBlueprint.CompanyID,
	}, nil
}

func (u *ReportUsecase) suspendListTarget(
	ctx context.Context,
	reportCase reportdom.ReportCase,
	reason string,
	adminID string,
) error {
	if u.listModerator == nil {
		return ErrReportUsecaseNotConfigured
	}

	return u.listModerator.SuspendListByAdmin(
		ctx,
		SuspendListByAdminInput{
			ListID:  reportCase.TargetID,
			Reason:  reason,
			AdminID: adminID,
		},
	)
}
