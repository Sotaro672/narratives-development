// backend/internal/application/query/admin/contract_tokenBlueprint_query.go
package query

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
	reportdom "narratives/internal/domain/report"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
	tokenblueprintreviewdom "narratives/internal/domain/tokenBlueprint_review"
)

var ErrContractTokenBlueprintQueryNotConfigured = errors.New(
	"contract token blueprint query is not configured",
)

type contractTokenBlueprintCompanyReader interface {
	GetByID(ctx context.Context, id string) (companydom.Company, error)
}

type contractTokenBlueprintBrandReader interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

type contractTokenBlueprintMemberReader interface {
	GetByID(ctx context.Context, id string) (memberdom.Record, error)
	GetByUID(ctx context.Context, uid string) (memberdom.Record, error)
}

type contractTokenBlueprintReader interface {
	GetByID(ctx context.Context, id string) (*tokenblueprintdom.TokenBlueprint, error)
}

type contractTokenBlueprintReviewAggregateReader interface {
	GetAggregate(
		ctx context.Context,
		tokenBlueprintID string,
	) (tokenblueprintreviewdom.TokenBlueprintReviewAggregate, error)
}

type contractTokenBlueprintReportCaseReader interface {
	GetCase(
		ctx context.Context,
		caseID reportdom.CaseID,
	) (reportdom.ReportCase, error)
}

type ContractTokenBlueprintQuery struct {
	companyRepo           contractTokenBlueprintCompanyReader
	brandRepo             contractTokenBlueprintBrandReader
	memberRepo            contractTokenBlueprintMemberReader
	tokenBlueprintRepo    contractTokenBlueprintReader
	reviewAggregateReader contractTokenBlueprintReviewAggregateReader
	reportCaseRepo        contractTokenBlueprintReportCaseReader
}

func NewContractTokenBlueprintQuery(
	companyRepo contractTokenBlueprintCompanyReader,
	brandRepo contractTokenBlueprintBrandReader,
	memberRepo contractTokenBlueprintMemberReader,
	tokenBlueprintRepo contractTokenBlueprintReader,
	reviewAggregateReader contractTokenBlueprintReviewAggregateReader,
	reportCaseRepo contractTokenBlueprintReportCaseReader,
) *ContractTokenBlueprintQuery {
	return &ContractTokenBlueprintQuery{
		companyRepo:           companyRepo,
		brandRepo:             brandRepo,
		memberRepo:            memberRepo,
		tokenBlueprintRepo:    tokenBlueprintRepo,
		reviewAggregateReader: reviewAggregateReader,
		reportCaseRepo:        reportCaseRepo,
	}
}

type ContractTokenBlueprintDetailResult struct {
	Company        ContractCompanyRow              `json:"company"`
	TokenBlueprint ContractTokenBlueprintDetailRow `json:"tokenBlueprint"`
}

type ContractTokenBlueprintDetailRow struct {
	ID               string                              `json:"id"`
	Name             string                              `json:"name"`
	Symbol           string                              `json:"symbol"`
	BrandID          string                              `json:"brandId"`
	BrandName        string                              `json:"brandName"`
	CompanyID        string                              `json:"companyId"`
	Description      string                              `json:"description"`
	AssigneeID       string                              `json:"assigneeId"`
	AssigneeName     string                              `json:"assigneeName"`
	Minted           bool                                `json:"minted"`
	ModerationStatus string                              `json:"moderationStatus"`
	MetadataURI      string                              `json:"metadataUri"`
	IconURL          string                              `json:"iconUrl"`
	ContentFiles     []ContractTokenBlueprintContentFile `json:"contentFiles"`
	LikeCount        int64                               `json:"likeCount"`
	DislikeCount     int64                               `json:"dislikeCount"`
	CommentCount     int64                               `json:"commentCount"`
	ReportCount      int                                 `json:"reportCount"`
	CreatedAt        string                              `json:"createdAt"`
	UpdatedAt        string                              `json:"updatedAt"`
}

type ContractTokenBlueprintContentFile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	ContentType string `json:"contentType"`
	URL         string `json:"url"`
	IsPublic    bool   `json:"isPublic"`
	Size        int64  `json:"size"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

func (q *ContractTokenBlueprintQuery) Get(
	ctx context.Context,
	companyID string,
	tokenBlueprintID string,
) (ContractTokenBlueprintDetailResult, error) {
	if q == nil || q.companyRepo == nil || q.brandRepo == nil || q.memberRepo == nil || q.tokenBlueprintRepo == nil || q.reviewAggregateReader == nil || q.reportCaseRepo == nil {
		return ContractTokenBlueprintDetailResult{}, ErrContractTokenBlueprintQueryNotConfigured
	}
	if companyID == "" {
		return ContractTokenBlueprintDetailResult{}, companydom.ErrInvalidID
	}
	if tokenBlueprintID == "" {
		return ContractTokenBlueprintDetailResult{}, tokenblueprintdom.ErrInvalidID
	}

	company, err := q.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return ContractTokenBlueprintDetailResult{}, err
	}

	tokenBlueprint, err := q.tokenBlueprintRepo.GetByID(ctx, tokenBlueprintID)
	if err != nil {
		return ContractTokenBlueprintDetailResult{}, err
	}
	if tokenBlueprint == nil || tokenBlueprint.ID != tokenBlueprintID || tokenBlueprint.CompanyID != companyID {
		return ContractTokenBlueprintDetailResult{}, tokenblueprintdom.ErrNotFound
	}

	contentFiles := make([]ContractTokenBlueprintContentFile, 0, len(tokenBlueprint.ContentFiles))
	for _, contentFile := range tokenBlueprint.ContentFiles {
		contentFiles = append(contentFiles, ContractTokenBlueprintContentFile{
			ID:          contentFile.ID,
			Name:        contentFile.Name,
			Type:        string(contentFile.Type),
			ContentType: contentFile.ContentType,
			URL:         contentFile.URL,
			IsPublic:    contentFile.IsPublic,
			Size:        contentFile.Size,
			CreatedAt:   formatContractDetailTime(contentFile.CreatedAt),
			UpdatedAt:   formatContractDetailTime(contentFile.UpdatedAt),
		})
	}

	var likeCount int64
	var dislikeCount int64
	var commentCount int64

	aggregate, err := q.reviewAggregateReader.GetAggregate(ctx, tokenBlueprintID)
	if err == nil {
		likeCount = aggregate.LikeCount
		dislikeCount = aggregate.DislikeCount
		commentCount = aggregate.TopLevelCommentCount
	}

	reportCount, err := q.resolveReportCount(ctx, tokenBlueprintID)
	if err != nil {
		return ContractTokenBlueprintDetailResult{}, err
	}

	representativeName := q.resolveMemberName(ctx, company.Admin)
	if representativeName == "" || representativeName == company.Admin {
		representativeName = "-"
	}

	return ContractTokenBlueprintDetailResult{
		Company: ContractCompanyRow{
			ID:                 company.ID,
			Name:               company.Name,
			RepresentativeName: representativeName,
			IsActive:           company.IsActive,
			CreatedAt:          formatContractDetailTime(company.CreatedAt),
			UpdatedAt:          formatContractDetailTime(company.UpdatedAt),
		},
		TokenBlueprint: ContractTokenBlueprintDetailRow{
			ID:               tokenBlueprint.ID,
			Name:             tokenBlueprint.Name,
			Symbol:           tokenBlueprint.Symbol,
			BrandID:          tokenBlueprint.BrandID,
			BrandName:        q.resolveBrandName(ctx, tokenBlueprint.BrandID),
			CompanyID:        tokenBlueprint.CompanyID,
			Description:      tokenBlueprint.Description,
			AssigneeID:       tokenBlueprint.AssigneeID,
			AssigneeName:     q.resolveMemberName(ctx, tokenBlueprint.AssigneeID),
			Minted:           tokenBlueprint.Minted,
			ModerationStatus: string(tokenBlueprint.EffectiveModerationStatus()),
			MetadataURI:      tokenBlueprint.MetadataURI,
			IconURL:          tokenBlueprint.IconURL,
			ContentFiles:     contentFiles,
			LikeCount:        likeCount,
			DislikeCount:     dislikeCount,
			CommentCount:     commentCount,
			ReportCount:      reportCount,
			CreatedAt:        formatContractDetailTime(tokenBlueprint.CreatedAt),
			UpdatedAt:        formatContractDetailTime(tokenBlueprint.UpdatedAt),
		},
	}, nil
}

func (q *ContractTokenBlueprintQuery) resolveReportCount(
	ctx context.Context,
	tokenBlueprintID string,
) (int, error) {
	caseID, err := reportdom.BuildCaseID(
		reportdom.TargetTypeTokenBlueprint,
		tokenBlueprintID,
	)
	if err != nil {
		return 0, err
	}

	reportCase, err := q.reportCaseRepo.GetCase(ctx, caseID)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return 0, nil
		}
		return 0, err
	}

	if reportCase.TargetType != reportdom.TargetTypeTokenBlueprint {
		return 0, reportdom.ErrInvalidTargetType
	}
	if reportCase.TargetID != tokenBlueprintID {
		return 0, reportdom.ErrInvalidTargetID
	}

	return reportCase.ReportCount, nil
}

func (q *ContractTokenBlueprintQuery) resolveBrandName(ctx context.Context, brandID string) string {
	if brandID == "" {
		return "-"
	}
	brand, err := q.brandRepo.GetByID(ctx, brandID)
	if err != nil || brand.Name == "" {
		return brandID
	}
	return brand.Name
}

func (q *ContractTokenBlueprintQuery) resolveMemberName(ctx context.Context, memberID string) string {
	if memberID == "" {
		return "-"
	}

	record, err := q.memberRepo.GetByID(ctx, memberID)
	if err == nil {
		name := memberdom.FormatLastFirst(record.Member.LastName, record.Member.FirstName)
		if name != "" {
			return name
		}
	}

	record, err = q.memberRepo.GetByUID(ctx, memberID)
	if err == nil {
		name := memberdom.FormatLastFirst(record.Member.LastName, record.Member.FirstName)
		if name != "" {
			return name
		}
	}

	return memberID
}
