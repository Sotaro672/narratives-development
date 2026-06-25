// backend/internal/application/query/console/inquiry_management_query.go
package query

import (
	"context"
	"errors"
	"fmt"
	"strings"

	inquirydom "narratives/internal/domain/inquiry"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	productblueprintdom "narratives/internal/domain/productBlueprint"
)

// InquiryManagementQuery は Console 管理画面向けの Inquiry read model を扱います。
//
// usecase は command 専用に寄せるため、管理画面で必要な get/list/count と
// Inquiry.Images 由来の画像取得はこちらへ集約します。
//
// Inquiry.ProductID から Product.ModelID を解決し、
// ModelVariation.GetProductBlueprintID() から productBlueprintId を解決し、
// ProductBlueprint.CompanyID を解決します。
//
// ListByCompanyID では、ログイン中 member の companyId と一致する inquiry のみを返します。
type InquiryManagementQuery struct {
	repo                 inquirydom.Repository
	productRepo          productdom.Repository
	modelRepo            modeldom.RepositoryPort
	productBlueprintRepo productblueprintdom.Repository
}

// NewInquiryManagementQuery は InquiryManagementQuery を初期化します。
func NewInquiryManagementQuery(
	repo inquirydom.Repository,
	productRepo productdom.Repository,
	modelRepo modeldom.RepositoryPort,
	productBlueprintRepo productblueprintdom.Repository,
) *InquiryManagementQuery {
	return &InquiryManagementQuery{
		repo:                 repo,
		productRepo:          productRepo,
		modelRepo:            modelRepo,
		productBlueprintRepo: productBlueprintRepo,
	}
}

// InquiryManagementItem は Console 管理画面向けの Inquiry 一覧 item です。
//
// Inquiry.ProductID から解決した modelId / productBlueprintId / companyId を含めます。
type InquiryManagementItem struct {
	Inquiry            inquirydom.Inquiry `json:"inquiry"`
	ModelID            string             `json:"modelId"`
	ProductBlueprintID string             `json:"productBlueprintId"`
	CompanyID          string             `json:"companyId"`
}

// InquiryDetail は Console 管理画面向けの Inquiry 詳細 read model です。
//
// Inquiry.ProductID から解決した modelId / productBlueprintId / companyId を含めます。
type InquiryDetail struct {
	Inquiry            inquirydom.Inquiry `json:"inquiry"`
	ModelID            string             `json:"modelId"`
	ProductBlueprintID string             `json:"productBlueprintId"`
	CompanyID          string             `json:"companyId"`
}

// InquiryAggregate は Inquiry とその画像一覧をまとめた管理画面向けビューです。
//
// inquiryImage ドメインは inquiry ドメインへ統合済みのため、
// Images は Inquiry.Images から取得します。
// また、Inquiry.ProductID から解決した modelId / productBlueprintId / companyId を含めます。
type InquiryAggregate struct {
	Inquiry            inquirydom.Inquiry     `json:"inquiry"`
	Images             []inquirydom.ImageFile `json:"images"`
	ModelID            string                 `json:"modelId"`
	ProductBlueprintID string                 `json:"productBlueprintId"`
	CompanyID          string                 `json:"companyId"`
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

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return inquirydom.PageResult[InquiryManagementItem]{}, fmt.Errorf("inquiry management query: companyId is empty")
	}

	result, err := q.repo.ListByCompanyID(ctx, companyID, filter, sort, page)
	if err != nil {
		return inquirydom.PageResult[InquiryManagementItem]{}, err
	}

	items := make([]InquiryManagementItem, 0, len(result.Items))
	for _, inq := range result.Items {
		modelID, productBlueprintID, resolvedCompanyID, err := q.resolveProductModelRefByInquiryProductID(ctx, inq.ProductID)
		if err != nil {
			return inquirydom.PageResult[InquiryManagementItem]{}, err
		}

		// companyId を解決できない Inquiry は company boundary を保証できないため返さない。
		if resolvedCompanyID == "" {
			continue
		}

		// ログイン中 member の companyId と一致する Inquiry のみ返す。
		if resolvedCompanyID != companyID {
			continue
		}

		items = append(items, InquiryManagementItem{
			Inquiry:            inq,
			ModelID:            modelID,
			ProductBlueprintID: productBlueprintID,
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

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return 0, fmt.Errorf("inquiry management query: companyId is empty")
	}

	return q.repo.CountUnreadByCompanyID(ctx, companyID, filter)
}

// GetByID は Inquiry を返します。
//
// command 処理前の現在状態取得など、domain entity が必要な場合に使います。
// 画面表示で modelId / productBlueprintId / companyId も必要な場合は GetDetailByID を使ってください。
func (q *InquiryManagementQuery) GetByID(ctx context.Context, id string) (inquirydom.Inquiry, error) {
	if q == nil || q.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("inquiry management query: repository is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidID
	}

	return q.repo.GetByID(ctx, id)
}

// GetDetailByID は Inquiry 詳細 read model を返します。
//
// Inquiry.ProductID から product repository を使って Product.ModelID を取得し、
// model repository を使って ModelVariation.GetProductBlueprintID() を取得し、
// productBlueprint repository を使って ProductBlueprint.CompanyID を取得します。
func (q *InquiryManagementQuery) GetDetailByID(ctx context.Context, id string) (InquiryDetail, error) {
	inq, err := q.GetByID(ctx, id)
	if err != nil {
		return InquiryDetail{}, err
	}

	modelID, productBlueprintID, companyID, err := q.resolveProductModelRefByInquiryProductID(ctx, inq.ProductID)
	if err != nil {
		return InquiryDetail{}, err
	}

	return InquiryDetail{
		Inquiry:            inq,
		ModelID:            modelID,
		ProductBlueprintID: productBlueprintID,
		CompanyID:          companyID,
	}, nil
}

// GetDetailByIDForCompany は company boundary 確認込みで Inquiry 詳細 read model を返します。
//
// Console 管理画面の詳細表示で companyId 境界を保証したい場合はこちらを使います。
func (q *InquiryManagementQuery) GetDetailByIDForCompany(
	ctx context.Context,
	id string,
	companyID string,
) (InquiryDetail, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return InquiryDetail{}, fmt.Errorf("inquiry management query: companyId is empty")
	}

	detail, err := q.GetDetailByID(ctx, id)
	if err != nil {
		return InquiryDetail{}, err
	}

	if strings.TrimSpace(detail.CompanyID) != companyID {
		return InquiryDetail{}, inquirydom.ErrNotFound
	}

	return detail, nil
}

// GetImages は Inquiry に紐づく画像一覧を返します。
//
// inquiryImage ドメインは廃止済みのため、別 repository へは問い合わせず、
// Inquiry.Images をそのまま返します。
func (q *InquiryManagementQuery) GetImages(ctx context.Context, inquiryID string) ([]inquirydom.ImageFile, error) {
	if q == nil || q.repo == nil {
		return nil, fmt.Errorf("inquiry management query: repository is nil")
	}

	inq, err := q.GetByID(ctx, inquiryID)
	if err != nil {
		return nil, err
	}
	if len(inq.Images) == 0 {
		return []inquirydom.ImageFile{}, nil
	}
	return inq.Images, nil
}

// GetImagesForCompany は company boundary 確認込みで Inquiry 画像一覧を返します。
func (q *InquiryManagementQuery) GetImagesForCompany(
	ctx context.Context,
	inquiryID string,
	companyID string,
) ([]inquirydom.ImageFile, error) {
	detail, err := q.GetDetailByIDForCompany(ctx, inquiryID, companyID)
	if err != nil {
		return nil, err
	}

	if len(detail.Inquiry.Images) == 0 {
		return []inquirydom.ImageFile{}, nil
	}

	return detail.Inquiry.Images, nil
}

// GetAggregate は Inquiry と画像一覧をまとめて返します。
//
// 画像は Inquiry.Images を正として扱います。
// また、Inquiry.ProductID から解決した modelId / productBlueprintId / companyId を含めます。
func (q *InquiryManagementQuery) GetAggregate(ctx context.Context, id string) (InquiryAggregate, error) {
	if q == nil || q.repo == nil {
		return InquiryAggregate{}, fmt.Errorf("inquiry management query: repository is nil")
	}

	inq, err := q.GetByID(ctx, id)
	if err != nil {
		return InquiryAggregate{}, err
	}

	images := inq.Images
	if images == nil {
		images = []inquirydom.ImageFile{}
	}

	modelID, productBlueprintID, companyID, err := q.resolveProductModelRefByInquiryProductID(ctx, inq.ProductID)
	if err != nil {
		return InquiryAggregate{}, err
	}

	return InquiryAggregate{
		Inquiry:            inq,
		Images:             images,
		ModelID:            modelID,
		ProductBlueprintID: productBlueprintID,
		CompanyID:          companyID,
	}, nil
}

// GetAggregateForCompany は company boundary 確認込みで Inquiry aggregate を返します。
//
// Console 管理画面の詳細表示で companyId 境界を保証したい場合はこちらを使います。
func (q *InquiryManagementQuery) GetAggregateForCompany(
	ctx context.Context,
	id string,
	companyID string,
) (InquiryAggregate, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return InquiryAggregate{}, fmt.Errorf("inquiry management query: companyId is empty")
	}

	aggregate, err := q.GetAggregate(ctx, id)
	if err != nil {
		return InquiryAggregate{}, err
	}

	if strings.TrimSpace(aggregate.CompanyID) != companyID {
		return InquiryAggregate{}, inquirydom.ErrNotFound
	}

	return aggregate, nil
}

func (q *InquiryManagementQuery) resolveProductModelRefByInquiryProductID(
	ctx context.Context,
	productID string,
) (modelID string, productBlueprintID string, companyID string, err error) {
	if q == nil {
		return "", "", "", fmt.Errorf("inquiry management query: query is nil")
	}

	productID = strings.TrimSpace(productID)
	if productID == "" {
		return "", "", "", nil
	}

	if q.productRepo == nil {
		return "", "", "", fmt.Errorf("inquiry management query: product repository is nil")
	}

	product, err := q.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, productdom.ErrNotFound) {
			return "", "", "", nil
		}
		return "", "", "", err
	}

	modelID = strings.TrimSpace(product.ModelID)
	if modelID == "" {
		return "", "", "", nil
	}

	if q.modelRepo == nil {
		return modelID, "", "", fmt.Errorf("inquiry management query: model repository is nil")
	}

	model, err := q.modelRepo.GetByID(ctx, modelID)
	if err != nil {
		if errors.Is(err, modeldom.ErrNotFound) {
			return modelID, "", "", nil
		}
		return modelID, "", "", err
	}

	productBlueprintID = strings.TrimSpace(model.GetProductBlueprintID())
	if productBlueprintID == "" {
		return modelID, "", "", nil
	}

	if q.productBlueprintRepo == nil {
		return modelID, productBlueprintID, "", fmt.Errorf("inquiry management query: product blueprint repository is nil")
	}

	productBlueprint, err := q.productBlueprintRepo.GetByID(ctx, productBlueprintID)
	if err != nil {
		return modelID, productBlueprintID, "", err
	}

	return modelID, productBlueprintID, strings.TrimSpace(productBlueprint.CompanyID), nil
}
