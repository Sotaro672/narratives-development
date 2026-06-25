// backend/internal/application/query/console/inquiry_management_query.go
package query

import (
	"context"
	"errors"
	"fmt"

	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	inquirydom "narratives/internal/domain/inquiry"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	userdom "narratives/internal/domain/user"
)

// InquiryManagementQuery は Console 管理画面向けの Inquiry 一覧 read model を扱います。
//
// 一覧画面で必要な軽量情報のみを解決します。
// Inquiry.ProductID から Product.ModelID を解決し、
// ModelVariation.GetProductBlueprintID() から productBlueprintId を解決し、
// ProductBlueprint.CompanyID / ProductName / BrandID を解決します。
// さらに Brand.GetByID() から BrandName を解決します。
// Inquiry.AvatarID から Avatar.GetByID() を使って UserID を解決し、
// User.GetByID() を使って UserFullName を解決します。
//
// 一覧では detail 用の mintAddress / transferredAt / shippingAddresses / orders は解決しません。
//
// ListByCompanyID では、ログイン中 member の companyId と一致する inquiry のみを返します。
type InquiryManagementQuery struct {
	repo                 inquirydom.Repository
	productRepo          productdom.Repository
	modelRepo            modeldom.RepositoryPort
	productBlueprintRepo productblueprintdom.Repository
	brandRepo            branddom.RepositoryPort
	avatarRepo           avatardom.Repository
	userRepo             userdom.RepositoryPort
}

// NewInquiryManagementQuery は InquiryManagementQuery を初期化します。
func NewInquiryManagementQuery(
	repo inquirydom.Repository,
	productRepo productdom.Repository,
	modelRepo modeldom.RepositoryPort,
	productBlueprintRepo productblueprintdom.Repository,
	brandRepo branddom.RepositoryPort,
	avatarRepo avatardom.Repository,
	userRepo userdom.RepositoryPort,
) *InquiryManagementQuery {
	return &InquiryManagementQuery{
		repo:                 repo,
		productRepo:          productRepo,
		modelRepo:            modelRepo,
		productBlueprintRepo: productBlueprintRepo,
		brandRepo:            brandRepo,
		avatarRepo:           avatarRepo,
		userRepo:             userRepo,
	}
}

// InquiryManagementItem は Console 管理画面向けの Inquiry 一覧 item です。
//
// ManagementPage で表示・絞り込みに必要な項目のみを含めます。
type InquiryManagementItem struct {
	Inquiry            inquirydom.Inquiry `json:"inquiry"`
	ModelID            string             `json:"modelId"`
	ProductBlueprintID string             `json:"productBlueprintId"`
	ProductName        string             `json:"productName"`
	BrandID            string             `json:"brandId"`
	BrandName          string             `json:"brandName"`
	UserFullName       string             `json:"userFullName"`
	CompanyID          string             `json:"companyId"`
}

// ListByCompanyID は companyID に紐づく Inquiry 一覧を返します。
//
// companyID は middleware.MemberAuth により context に格納された
// ログイン中 member の companyId を handler 側で取り出して渡す想定です。
//
// Inquiry 自体は companyId を保持しないため、
// Inquiry.ProductID -> Product.ModelID -> ProductBlueprintID -> CompanyID を query 側で解決し、
// resolved companyId がログイン中 member の companyId と一致する Inquiry のみを返します。
func (q *InquiryManagementQuery) ListByCompanyID(
	ctx context.Context,
	companyID string,
	filter inquirydom.Filter,
	sort inquirydom.Sort,
	page inquirydom.Page,
) (inquirydom.PageResult[InquiryManagementItem], error) {
	if q == nil || q.repo == nil {
		return inquirydom.PageResult[InquiryManagementItem]{}, fmt.Errorf("inquiry management query: repository is nil")
	}

	if companyID == "" {
		return inquirydom.PageResult[InquiryManagementItem]{}, fmt.Errorf("inquiry management query: companyId is empty")
	}

	result, err := q.repo.ListByCompanyID(ctx, companyID, filter, sort, page)
	if err != nil {
		return inquirydom.PageResult[InquiryManagementItem]{}, err
	}

	items := make([]InquiryManagementItem, 0, len(result.Items))
	for _, inq := range result.Items {
		modelID, productBlueprintID, productName, brandID, brandName, resolvedCompanyID, err := q.resolveProductModelRefByInquiryProductID(ctx, inq.ProductID)
		if err != nil {
			return inquirydom.PageResult[InquiryManagementItem]{}, err
		}

		if resolvedCompanyID == "" {
			continue
		}

		if resolvedCompanyID != companyID {
			continue
		}

		userFullName, err := q.resolveUserFullNameByAvatarID(ctx, inq.AvatarID)
		if err != nil {
			return inquirydom.PageResult[InquiryManagementItem]{}, err
		}

		items = append(items, InquiryManagementItem{
			Inquiry:            inq,
			ModelID:            modelID,
			ProductBlueprintID: productBlueprintID,
			ProductName:        productName,
			BrandID:            brandID,
			BrandName:          brandName,
			UserFullName:       userFullName,
			CompanyID:          resolvedCompanyID,
		})
	}

	return inquirydom.PageResult[InquiryManagementItem]{
		Items: items,
	}, nil
}

// CountUnreadByCompanyID は companyID に紐づく未読 Inquiry 件数を返します。
func (q *InquiryManagementQuery) CountUnreadByCompanyID(
	ctx context.Context,
	companyID string,
	filter inquirydom.Filter,
) (int, error) {
	if q == nil || q.repo == nil {
		return 0, fmt.Errorf("inquiry management query: repository is nil")
	}

	if companyID == "" {
		return 0, fmt.Errorf("inquiry management query: companyId is empty")
	}

	return q.repo.CountUnreadByCompanyID(ctx, companyID, filter)
}

// GetByID は Inquiry を返します。
//
// command 処理前の現在状態取得など、domain entity が必要な場合に使います。
// 詳細画面表示では InquiryDetailQuery.GetDetailByID を使ってください。
func (q *InquiryManagementQuery) GetByID(ctx context.Context, id string) (inquirydom.Inquiry, error) {
	if q == nil || q.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("inquiry management query: repository is nil")
	}

	if id == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidID
	}

	return q.repo.GetByID(ctx, id)
}

func (q *InquiryManagementQuery) resolveProductModelRefByInquiryProductID(
	ctx context.Context,
	productID string,
) (
	modelID string,
	productBlueprintID string,
	productName string,
	brandID string,
	brandName string,
	companyID string,
	err error,
) {
	if q == nil {
		return "", "", "", "", "", "", fmt.Errorf("inquiry management query: query is nil")
	}

	if productID == "" {
		return "", "", "", "", "", "", nil
	}

	if q.productRepo == nil {
		return "", "", "", "", "", "", fmt.Errorf("inquiry management query: product repository is nil")
	}

	product, err := q.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, productdom.ErrNotFound) {
			return "", "", "", "", "", "", nil
		}
		return "", "", "", "", "", "", err
	}

	modelID = product.ModelID
	if modelID == "" {
		return "", "", "", "", "", "", nil
	}

	if q.modelRepo == nil {
		return modelID, "", "", "", "", "", fmt.Errorf("inquiry management query: model repository is nil")
	}

	model, err := q.modelRepo.GetByID(ctx, modelID)
	if err != nil {
		if errors.Is(err, modeldom.ErrNotFound) {
			return modelID, "", "", "", "", "", nil
		}
		return modelID, "", "", "", "", "", err
	}

	productBlueprintID = model.GetProductBlueprintID()
	if productBlueprintID == "" {
		return modelID, "", "", "", "", "", nil
	}

	if q.productBlueprintRepo == nil {
		return modelID, productBlueprintID, "", "", "", "", fmt.Errorf("inquiry management query: product blueprint repository is nil")
	}

	productBlueprint, err := q.productBlueprintRepo.GetByID(ctx, productBlueprintID)
	if err != nil {
		return modelID, productBlueprintID, "", "", "", "", err
	}

	productName = productBlueprint.ProductName
	brandID = productBlueprint.BrandID
	companyID = productBlueprint.CompanyID

	if brandID == "" {
		return modelID, productBlueprintID, productName, "", "", companyID, nil
	}

	if q.brandRepo == nil {
		return modelID, productBlueprintID, productName, brandID, "", companyID, fmt.Errorf("inquiry management query: brand repository is nil")
	}

	brand, err := q.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		if errors.Is(err, branddom.ErrNotFound) {
			return modelID, productBlueprintID, productName, brandID, "", companyID, nil
		}
		return modelID, productBlueprintID, productName, brandID, "", companyID, err
	}

	brandName = brand.Name

	return modelID, productBlueprintID, productName, brandID, brandName, companyID, nil
}

func (q *InquiryManagementQuery) resolveUserFullNameByAvatarID(
	ctx context.Context,
	avatarID string,
) (string, error) {
	if q == nil {
		return "", fmt.Errorf("inquiry management query: query is nil")
	}

	if avatarID == "" {
		return "", nil
	}

	if q.avatarRepo == nil {
		return "", fmt.Errorf("inquiry management query: avatar repository is nil")
	}

	avatar, err := q.avatarRepo.GetByID(ctx, avatarID)
	if err != nil {
		return "", err
	}

	if avatar.UserID == "" {
		return "", nil
	}

	if q.userRepo == nil {
		return "", fmt.Errorf("inquiry management query: user repository is nil")
	}

	user, err := q.userRepo.GetByID(ctx, avatar.UserID)
	if err != nil {
		if errors.Is(err, userdom.ErrNotFound) {
			return "", nil
		}
		return "", err
	}

	return userdom.FormatName(user), nil
}
