// backend/internal/application/usecase/report_product_blueprint_review.go
package usecase

import (
	"context"
	"time"

	pbr "narratives/internal/domain/productBlueprintReview"
	reportdom "narratives/internal/domain/report"
)

// ReportProductReviewModerator owns Admin moderation of product reviews.
type ReportProductReviewModerator interface {
	RemoveProductBlueprintReviewByAdmin(
		ctx context.Context,
		in RemoveProductBlueprintReviewByAdminInput,
	) (pbr.Review, error)
}

type ReportProductBlueprintReviewByAvatarInput struct {
	ProductBlueprintID string
	ReviewID           string
	AvatarID           string
	Reason             reportdom.ReportReason
	Detail             string
}

func (u *ReportUsecase) ReportProductBlueprintReviewByAvatar(
	ctx context.Context,
	input ReportProductBlueprintReviewByAvatarInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.productReviewRepo == nil || u.productPurchaseResolver == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}

	review, err := u.productReviewRepo.GetByProductBlueprintID(
		ctx,
		input.ProductBlueprintID,
		input.ReviewID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if review.ProductBlueprintID != input.ProductBlueprintID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetParentID
	}
	if review.Status == pbr.ReviewStatusRemoved {
		return reportdom.AddReportResult{}, reportdom.ErrCannotReportRemovedTarget
	}
	if review.AvatarID == input.AvatarID {
		return reportdom.AddReportResult{}, ErrReportSelfReport
	}

	allowed, err := u.productPurchaseResolver.HasOwnedProductBlueprint(
		ctx,
		input.AvatarID,
		input.ProductBlueprintID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if !allowed {
		return reportdom.AddReportResult{}, ErrReportForbidden
	}

	return u.addProductBlueprintReviewReport(
		ctx,
		review,
		reportdom.ActorTypeAvatar,
		input.AvatarID,
		"",
		input.Reason,
		input.Detail,
	)
}

type ReportProductBlueprintReviewByBrandInput struct {
	ProductBlueprintID string
	ReviewID           string
	BrandID            string
	CompanyID          string
	Reason             reportdom.ReportReason
	Detail             string
}

func (u *ReportUsecase) ReportProductBlueprintReviewByBrand(
	ctx context.Context,
	input ReportProductBlueprintReviewByBrandInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.productReviewRepo == nil || u.productBlueprintRepo == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}

	productBlueprint, err := u.productBlueprintRepo.GetByID(ctx, input.ProductBlueprintID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if productBlueprint.ID != input.ProductBlueprintID ||
		productBlueprint.CompanyID != input.CompanyID ||
		productBlueprint.BrandID != input.BrandID {
		return reportdom.AddReportResult{}, ErrReportForbidden
	}

	review, err := u.productReviewRepo.GetByProductBlueprintID(
		ctx,
		input.ProductBlueprintID,
		input.ReviewID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if review.ProductBlueprintID != input.ProductBlueprintID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetParentID
	}
	if review.Status == pbr.ReviewStatusRemoved {
		return reportdom.AddReportResult{}, reportdom.ErrCannotReportRemovedTarget
	}

	return u.addProductBlueprintReviewReport(
		ctx,
		review,
		reportdom.ActorTypeBrand,
		input.BrandID,
		input.CompanyID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReportUsecase) addProductBlueprintReviewReport(
	ctx context.Context,
	review pbr.Review,
	reporterType reportdom.ActorType,
	reporterID string,
	companyID string,
	reason reportdom.ReportReason,
	detail string,
) (reportdom.AddReportResult, error) {
	now := u.now().UTC()
	rating := int(review.Rating)

	reportCase, err := reportdom.NewReportCase(
		reportNewProductReviewCaseParams(
			review,
			rating,
			now,
		),
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	report, err := reportdom.NewReport(reportdom.NewReportParams{
		CaseID:       reportCase.ID,
		ReporterType: reporterType,
		ReporterID:   reporterID,
		CompanyID:    companyID,
		Reason:       reason,
		Detail:       detail,
		CreatedAt:    now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	return u.reportRepo.AddReport(ctx, reportCase, report)
}

func reportNewProductReviewCaseParams(
	review pbr.Review,
	rating int,
	now time.Time,
) reportdom.NewReportCaseParams {
	return reportdom.NewReportCaseParams{
		TargetType:       reportdom.TargetTypeProductBlueprintReview,
		TargetID:         string(review.ID),
		TargetParentID:   review.ProductBlueprintID,
		TargetAuthorID:   review.AvatarID,
		TargetAuthorType: reportdom.ActorTypeAvatar,
		SnapshotBody:     review.Body,
		SnapshotRating:   &rating,
		CreatedAt:        now,
	}
}

func (u *ReportUsecase) removeProductBlueprintReviewTarget(
	ctx context.Context,
	reportCase reportdom.ReportCase,
	reason string,
	adminID string,
) error {
	if u.productReviewModerator == nil {
		return ErrReportUsecaseNotConfigured
	}

	_, err := u.productReviewModerator.RemoveProductBlueprintReviewByAdmin(
		ctx,
		RemoveProductBlueprintReviewByAdminInput{
			ProductBlueprintID: reportCase.TargetParentID,
			ReviewID:           reportCase.TargetID,
			Reason:             reason,
			AdminID:            adminID,
		},
	)
	return err
}
