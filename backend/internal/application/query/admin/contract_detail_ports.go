// backend/internal/application/query/admin/contract_detail_ports.go
package query

import (
	"context"

	announcementdom "narratives/internal/domain/announcement"
	branddom "narratives/internal/domain/brand"
	common "narratives/internal/domain/common"
	companydom "narratives/internal/domain/company"
	inventorydom "narratives/internal/domain/inventory"
	listdom "narratives/internal/domain/list"
	memberdom "narratives/internal/domain/member"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	productblueprintreviewdom "narratives/internal/domain/productBlueprintReview"
	reportdom "narratives/internal/domain/report"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

type contractDetailCompanyReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (companydom.Company, error)
}

type contractDetailBrandReader interface {
	ListByCompanyID(
		ctx context.Context,
		companyID string,
		page branddom.Page,
	) (branddom.PageResult[branddom.Brand], error)

	GetByID(
		ctx context.Context,
		id string,
	) (branddom.Brand, error)
}

type contractDetailAnnouncementReader interface {
	ListByTargetToken(
		ctx context.Context,
		tokenBlueprintID string,
		page announcementdom.Page,
	) (announcementdom.PageResult[announcementdom.Announcement], error)
}

type contractDetailMemberReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (memberdom.Record, error)

	GetByUID(
		ctx context.Context,
		uid string,
	) (memberdom.Record, error)
}

type contractDetailProductBlueprintReader interface {
	ListByCompanyID(
		ctx context.Context,
		companyID string,
	) ([]productblueprintdom.ProductBlueprint, error)
}

type contractDetailProductBlueprintReviewReader interface {
	ListByProductBlueprintID(
		ctx context.Context,
		productBlueprintID string,
		status productblueprintreviewdom.ReviewStatus,
		page common.Page,
	) (common.PageResult[productblueprintreviewdom.Review], error)
}

type contractDetailTokenBlueprintReader interface {
	ListByCompanyID(
		ctx context.Context,
		companyID string,
		page common.Page,
	) (common.PageResult[tokenblueprintdom.TokenBlueprint], error)
}

type contractDetailInventoryReader interface {
	ListByProductBlueprintID(
		ctx context.Context,
		productBlueprintID string,
	) ([]inventorydom.Mint, error)
}

type contractDetailListReader interface {
	ListByInventoryID(
		ctx context.Context,
		inventoryID string,
	) ([]listdom.List, error)
}

type contractDetailReportCaseReader interface {
	GetCase(
		ctx context.Context,
		caseID reportdom.CaseID,
	) (reportdom.ReportCase, error)
}
