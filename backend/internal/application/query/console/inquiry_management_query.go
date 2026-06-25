// backend/internal/application/query/console/inquiry_management_query.go
package query

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	inquirydom "narratives/internal/domain/inquiry"
	modeldom "narratives/internal/domain/model"
	orderdom "narratives/internal/domain/order"
	productdom "narratives/internal/domain/product"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	shippingaddressdom "narratives/internal/domain/shippingAddress"
	tokendom "narratives/internal/domain/token"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
	transferdom "narratives/internal/domain/transfer"
	userdom "narratives/internal/domain/user"
)

// InquiryManagementQuery は Console 管理画面向けの Inquiry read model を扱います。
//
// usecase は command 専用に寄せるため、管理画面で必要な get/list/count と
// Inquiry.Images 由来の画像取得はこちらへ集約します。
//
// Inquiry.ProductID から Product.ModelID を解決し、
// ModelVariation.GetProductBlueprintID() から productBlueprintId を解決し、
// ProductBlueprint.CompanyID / ProductName / BrandID を解決します。
// さらに Brand.GetByID() から BrandName を解決します。
// Inquiry.ProductID を tokens/{productId} として解決し、Token.MintAddress を取得します。
// 取得した MintAddress から Transfer.TransferredAt を解決します。
// Inquiry.AvatarID から Avatar.GetByID() を使って AvatarName / UserID を解決します。
// 解決した UserID から User.GetByID() を使って UserFullName を解決します。
// 解決した UserID から ShippingAddress.ListByUserID() を使って配送先住所一覧を解決します。
// Inquiry.AvatarID から Order.ListByAvatarID() を使って注文一覧を取得し、
// Inquiry.ProductID 由来の modelId と MintAddress 由来の transferredAt が一致する Order.Items を持つ注文のみ返します。
// OrderItem.InventoryID の "_" 以降を tokenBlueprintId として解決し、TokenBlueprint.GetByID() から tokenName を取得します。
//
// ListByCompanyID では、ログイン中 member の companyId と一致する inquiry のみを返します。
type InquiryManagementQuery struct {
	repo                 inquirydom.Repository
	productRepo          productdom.Repository
	modelRepo            modeldom.RepositoryPort
	productBlueprintRepo productblueprintdom.Repository
	tokenQueryRepo       tokendom.TokenQueryPort
	transferQueryRepo    transferdom.TransferQueryPort
	tokenBlueprintRepo   tokenblueprintdom.RepositoryPort
	brandRepo            branddom.RepositoryPort
	avatarRepo           avatardom.Repository
	userRepo             userdom.RepositoryPort
	shippingAddressRepo  shippingaddressdom.RepositoryPort
	orderRepo            orderdom.Repository
}

// NewInquiryManagementQuery は InquiryManagementQuery を初期化します。
func NewInquiryManagementQuery(
	repo inquirydom.Repository,
	productRepo productdom.Repository,
	modelRepo modeldom.RepositoryPort,
	productBlueprintRepo productblueprintdom.Repository,
	tokenQueryRepo tokendom.TokenQueryPort,
	transferQueryRepo transferdom.TransferQueryPort,
	tokenBlueprintRepo tokenblueprintdom.RepositoryPort,
	brandRepo branddom.RepositoryPort,
	avatarRepo avatardom.Repository,
	userRepo userdom.RepositoryPort,
	shippingAddressRepo shippingaddressdom.RepositoryPort,
	orderRepo orderdom.Repository,
) *InquiryManagementQuery {
	return &InquiryManagementQuery{
		repo:                 repo,
		productRepo:          productRepo,
		modelRepo:            modelRepo,
		productBlueprintRepo: productBlueprintRepo,
		tokenQueryRepo:       tokenQueryRepo,
		transferQueryRepo:    transferQueryRepo,
		tokenBlueprintRepo:   tokenBlueprintRepo,
		brandRepo:            brandRepo,
		avatarRepo:           avatarRepo,
		userRepo:             userRepo,
		shippingAddressRepo:  shippingAddressRepo,
		orderRepo:            orderRepo,
	}
}

// InquiryManagementItem は Console 管理画面向けの Inquiry 一覧 item です。
//
// Inquiry.ProductID から解決した modelId / productBlueprintId / productName / brandId / brandName / companyId / mintAddress / transferredAt と、
// Inquiry.AvatarID から解決した avatarName / userId / userFullName / shippingAddresses / orders を含めます。
type InquiryManagementItem struct {
	Inquiry            inquirydom.Inquiry                   `json:"inquiry"`
	ModelID            string                               `json:"modelId"`
	ProductBlueprintID string                               `json:"productBlueprintId"`
	ProductName        string                               `json:"productName"`
	BrandID            string                               `json:"brandId"`
	BrandName          string                               `json:"brandName"`
	MintAddress        string                               `json:"mintAddress"`
	TransferredAt      *time.Time                           `json:"transferredAt,omitempty"`
	AvatarName         string                               `json:"avatarName"`
	UserID             string                               `json:"userId"`
	UserFullName       string                               `json:"userFullName"`
	ShippingAddresses  []shippingaddressdom.ShippingAddress `json:"shippingAddresses"`
	Orders             []InquiryOrderSummary                `json:"orders"`
	CompanyID          string                               `json:"companyId"`
}

// InquiryDetail は Console 管理画面向けの Inquiry 詳細 read model です。
//
// Inquiry.ProductID から解決した modelId / productBlueprintId / productName / brandId / brandName / companyId / mintAddress / transferredAt と、
// Inquiry.AvatarID から解決した avatarName / userId / userFullName / shippingAddresses / orders を含めます。
type InquiryDetail struct {
	Inquiry            inquirydom.Inquiry                   `json:"inquiry"`
	ModelID            string                               `json:"modelId"`
	ProductBlueprintID string                               `json:"productBlueprintId"`
	ProductName        string                               `json:"productName"`
	BrandID            string                               `json:"brandId"`
	BrandName          string                               `json:"brandName"`
	MintAddress        string                               `json:"mintAddress"`
	TransferredAt      *time.Time                           `json:"transferredAt,omitempty"`
	AvatarName         string                               `json:"avatarName"`
	UserID             string                               `json:"userId"`
	UserFullName       string                               `json:"userFullName"`
	ShippingAddresses  []shippingaddressdom.ShippingAddress `json:"shippingAddresses"`
	Orders             []InquiryOrderSummary                `json:"orders"`
	CompanyID          string                               `json:"companyId"`
}

// InquiryAggregate は Inquiry とその画像一覧をまとめた管理画面向けビューです。
//
// inquiryImage ドメインは inquiry ドメインへ統合済みのため、
// Images は Inquiry.Images から取得します。
// また、Inquiry.ProductID から解決した modelId / productBlueprintId / productName / brandId / brandName / companyId / mintAddress / transferredAt と、
// Inquiry.AvatarID から解決した avatarName / userId / userFullName / shippingAddresses / orders を含めます。
type InquiryAggregate struct {
	Inquiry            inquirydom.Inquiry                   `json:"inquiry"`
	Images             []inquirydom.ImageFile               `json:"images"`
	ModelID            string                               `json:"modelId"`
	ProductBlueprintID string                               `json:"productBlueprintId"`
	ProductName        string                               `json:"productName"`
	BrandID            string                               `json:"brandId"`
	BrandName          string                               `json:"brandName"`
	MintAddress        string                               `json:"mintAddress"`
	TransferredAt      *time.Time                           `json:"transferredAt,omitempty"`
	AvatarName         string                               `json:"avatarName"`
	UserID             string                               `json:"userId"`
	UserFullName       string                               `json:"userFullName"`
	ShippingAddresses  []shippingaddressdom.ShippingAddress `json:"shippingAddresses"`
	Orders             []InquiryOrderSummary                `json:"orders"`
	CompanyID          string                               `json:"companyId"`
}

// InquiryOrderSummary は Inquiry 管理画面向けの注文 read model です。
//
// 配送先情報は ShippingAddress から取得済みのため、Order.ShippingSnapshot は含めません。
// 決済情報も別用途のため、Order.PaymentMethodSnapshot は含めません。
type InquiryOrderSummary struct {
	ID        string                    `json:"id"`
	UserID    string                    `json:"userId"`
	AvatarID  string                    `json:"avatarId"`
	CartID    string                    `json:"cartId"`
	Paid      bool                      `json:"paid"`
	Items     []InquiryOrderItemSummary `json:"items"`
	CreatedAt time.Time                 `json:"createdAt"`
}

// InquiryOrderItemSummary は Inquiry 管理画面向けの注文 item read model です。
type InquiryOrderItemSummary struct {
	ModelID          string     `json:"modelId"`
	InventoryID      string     `json:"inventoryId"`
	TokenBlueprintID string     `json:"tokenBlueprintId"`
	TokenName        string     `json:"tokenName"`
	ListID           string     `json:"listId"`
	Qty              int        `json:"qty"`
	Price            int        `json:"price"`
	IsCanceled       bool       `json:"isCanceled"`
	IsDispatched     bool       `json:"isDispatched"`
	Transferred      bool       `json:"transferred"`
	TransferredAt    *time.Time `json:"transferredAt,omitempty"`
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

		mintAddress, err := q.resolveMintAddressByProductID(ctx, inq.ProductID)
		if err != nil {
			return inquirydom.PageResult[InquiryManagementItem]{}, err
		}

		transferredAt, err := q.resolveTransferredAtByMintAddress(ctx, mintAddress)
		if err != nil {
			return inquirydom.PageResult[InquiryManagementItem]{}, err
		}

		avatarName, userID, userFullName, shippingAddresses, err := q.resolveAvatarUserRefByAvatarID(ctx, inq.AvatarID)
		if err != nil {
			return inquirydom.PageResult[InquiryManagementItem]{}, err
		}

		orders, err := q.resolveOrdersByAvatarIDModelIDAndTransferredAt(ctx, inq.AvatarID, modelID, transferredAt)
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
			MintAddress:        mintAddress,
			TransferredAt:      transferredAt,
			AvatarName:         avatarName,
			UserID:             userID,
			UserFullName:       userFullName,
			ShippingAddresses:  shippingAddresses,
			Orders:             orders,
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
// 画面表示で modelId / productBlueprintId / productName / brandId / brandName / avatarName / companyId も必要な場合は GetDetailByID を使ってください。
func (q *InquiryManagementQuery) GetByID(ctx context.Context, id string) (inquirydom.Inquiry, error) {
	if q == nil || q.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("inquiry management query: repository is nil")
	}

	if id == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidID
	}

	return q.repo.GetByID(ctx, id)
}

// GetDetailByID は Inquiry 詳細 read model を返します。
//
// Inquiry.ProductID から product repository を使って Product.ModelID を取得し、
// token query repository を使って tokens/{productId} から MintAddress を取得し、
// transfer query repository を使って MintAddress から TransferredAt を取得し、
// model repository を使って ModelVariation.GetProductBlueprintID() を取得し、
// productBlueprint repository を使って ProductBlueprint.CompanyID / ProductName / BrandID を取得します。
// brand repository を使って BrandName を取得します。
// avatar repository を使って AvatarName / UserID を取得します。
// user repository を使って UserFullName を取得します。
// shippingAddress repository を使って ShippingAddresses を取得します。
// order repository を使って、avatarId / modelId / transferredAt が一致する Orders を取得します。
func (q *InquiryManagementQuery) GetDetailByID(ctx context.Context, id string) (InquiryDetail, error) {
	inq, err := q.GetByID(ctx, id)
	if err != nil {
		return InquiryDetail{}, err
	}

	modelID, productBlueprintID, productName, brandID, brandName, companyID, err := q.resolveProductModelRefByInquiryProductID(ctx, inq.ProductID)
	if err != nil {
		return InquiryDetail{}, err
	}

	mintAddress, err := q.resolveMintAddressByProductID(ctx, inq.ProductID)
	if err != nil {
		return InquiryDetail{}, err
	}

	transferredAt, err := q.resolveTransferredAtByMintAddress(ctx, mintAddress)
	if err != nil {
		return InquiryDetail{}, err
	}

	avatarName, userID, userFullName, shippingAddresses, err := q.resolveAvatarUserRefByAvatarID(ctx, inq.AvatarID)
	if err != nil {
		return InquiryDetail{}, err
	}

	orders, err := q.resolveOrdersByAvatarIDModelIDAndTransferredAt(ctx, inq.AvatarID, modelID, transferredAt)
	if err != nil {
		return InquiryDetail{}, err
	}

	return InquiryDetail{
		Inquiry:            inq,
		ModelID:            modelID,
		ProductBlueprintID: productBlueprintID,
		ProductName:        productName,
		BrandID:            brandID,
		BrandName:          brandName,
		MintAddress:        mintAddress,
		TransferredAt:      transferredAt,
		AvatarName:         avatarName,
		UserID:             userID,
		UserFullName:       userFullName,
		ShippingAddresses:  shippingAddresses,
		Orders:             orders,
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
	if companyID == "" {
		return InquiryDetail{}, fmt.Errorf("inquiry management query: companyId is empty")
	}

	detail, err := q.GetDetailByID(ctx, id)
	if err != nil {
		return InquiryDetail{}, err
	}

	if detail.CompanyID != companyID {
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
// また、Inquiry.ProductID から解決した modelId / productBlueprintId / productName / brandId / brandName / companyId / mintAddress / transferredAt と、
// Inquiry.AvatarID から解決した avatarName / userId / userFullName / shippingAddresses / orders を含めます。
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

	modelID, productBlueprintID, productName, brandID, brandName, companyID, err := q.resolveProductModelRefByInquiryProductID(ctx, inq.ProductID)
	if err != nil {
		return InquiryAggregate{}, err
	}

	mintAddress, err := q.resolveMintAddressByProductID(ctx, inq.ProductID)
	if err != nil {
		return InquiryAggregate{}, err
	}

	transferredAt, err := q.resolveTransferredAtByMintAddress(ctx, mintAddress)
	if err != nil {
		return InquiryAggregate{}, err
	}

	avatarName, userID, userFullName, shippingAddresses, err := q.resolveAvatarUserRefByAvatarID(ctx, inq.AvatarID)
	if err != nil {
		return InquiryAggregate{}, err
	}

	orders, err := q.resolveOrdersByAvatarIDModelIDAndTransferredAt(ctx, inq.AvatarID, modelID, transferredAt)
	if err != nil {
		return InquiryAggregate{}, err
	}

	return InquiryAggregate{
		Inquiry:            inq,
		Images:             images,
		ModelID:            modelID,
		ProductBlueprintID: productBlueprintID,
		ProductName:        productName,
		BrandID:            brandID,
		BrandName:          brandName,
		MintAddress:        mintAddress,
		TransferredAt:      transferredAt,
		AvatarName:         avatarName,
		UserID:             userID,
		UserFullName:       userFullName,
		ShippingAddresses:  shippingAddresses,
		Orders:             orders,
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
	if companyID == "" {
		return InquiryAggregate{}, fmt.Errorf("inquiry management query: companyId is empty")
	}

	aggregate, err := q.GetAggregate(ctx, id)
	if err != nil {
		return InquiryAggregate{}, err
	}

	if aggregate.CompanyID != companyID {
		return InquiryAggregate{}, inquirydom.ErrNotFound
	}

	return aggregate, nil
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

func (q *InquiryManagementQuery) resolveMintAddressByProductID(
	ctx context.Context,
	productID string,
) (string, error) {
	if q == nil {
		return "", fmt.Errorf("inquiry management query: query is nil")
	}

	if productID == "" {
		return "", nil
	}

	if q.tokenQueryRepo == nil {
		return "", fmt.Errorf("inquiry management query: token query repository is nil")
	}

	token, err := q.tokenQueryRepo.GetTokenByProductID(ctx, productID)
	if err != nil {
		if errors.Is(err, tokendom.ErrNotFound) {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(token.MintAddress), nil
}

func (q *InquiryManagementQuery) resolveTransferredAtByMintAddress(
	ctx context.Context,
	mintAddress string,
) (*time.Time, error) {
	if q == nil {
		return nil, fmt.Errorf("inquiry management query: query is nil")
	}

	m := strings.TrimSpace(mintAddress)
	if m == "" {
		return nil, nil
	}

	if q.transferQueryRepo == nil {
		return nil, fmt.Errorf("inquiry management query: transfer query repository is nil")
	}

	result, err := q.transferQueryRepo.ResolveTransferredAtByMintAddress(ctx, m)
	if err != nil {
		if errors.Is(err, transferdom.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	transferredAt := result.TransferredAt.UTC()
	if transferredAt.IsZero() {
		return nil, nil
	}

	return &transferredAt, nil
}

func (q *InquiryManagementQuery) resolveAvatarUserRefByAvatarID(
	ctx context.Context,
	avatarID string,
) (
	avatarName string,
	userID string,
	userFullName string,
	shippingAddresses []shippingaddressdom.ShippingAddress,
	err error,
) {
	if q == nil {
		return "", "", "", nil, fmt.Errorf("inquiry management query: query is nil")
	}

	if avatarID == "" {
		return "", "", "", []shippingaddressdom.ShippingAddress{}, nil
	}

	if q.avatarRepo == nil {
		return "", "", "", nil, fmt.Errorf("inquiry management query: avatar repository is nil")
	}

	avatar, err := q.avatarRepo.GetByID(ctx, avatarID)
	if err != nil {
		return "", "", "", nil, err
	}

	avatarName = avatar.AvatarName
	userID = avatar.UserID

	if userID == "" {
		return avatarName, "", "", []shippingaddressdom.ShippingAddress{}, nil
	}

	if q.userRepo == nil {
		return avatarName, userID, "", nil, fmt.Errorf("inquiry management query: user repository is nil")
	}

	user, err := q.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, userdom.ErrNotFound) {
			userFullName = ""
		} else {
			return avatarName, userID, "", nil, err
		}
	} else {
		userFullName = userdom.FormatName(user)
	}

	if q.shippingAddressRepo == nil {
		return avatarName, userID, userFullName, nil, fmt.Errorf("inquiry management query: shipping address repository is nil")
	}

	shippingAddresses, err = q.shippingAddressRepo.ListByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, shippingaddressdom.ErrNotFound) {
			return avatarName, userID, userFullName, []shippingaddressdom.ShippingAddress{}, nil
		}
		return avatarName, userID, userFullName, nil, err
	}

	if shippingAddresses == nil {
		shippingAddresses = []shippingaddressdom.ShippingAddress{}
	}

	return avatarName, userID, userFullName, shippingAddresses, nil
}

func (q *InquiryManagementQuery) resolveOrdersByAvatarIDModelIDAndTransferredAt(
	ctx context.Context,
	avatarID string,
	modelID string,
	transferredAt *time.Time,
) ([]InquiryOrderSummary, error) {
	if q == nil {
		return nil, fmt.Errorf("inquiry management query: query is nil")
	}

	if avatarID == "" || modelID == "" || transferredAt == nil || transferredAt.IsZero() {
		return []InquiryOrderSummary{}, nil
	}

	if q.orderRepo == nil {
		return nil, fmt.Errorf("inquiry management query: order repository is nil")
	}

	result, err := q.orderRepo.ListByAvatarID(
		ctx,
		avatarID,
		orderdom.Sort{
			Column: orderdom.SortByCreatedAt,
			Order:  orderdom.SortDesc,
		},
		orderdom.Page{
			Number:  1,
			PerPage: 100,
		},
	)
	if err != nil {
		if errors.Is(err, orderdom.ErrNotFound) {
			return []InquiryOrderSummary{}, nil
		}
		return nil, err
	}

	orders := make([]InquiryOrderSummary, 0, len(result.Items))
	for _, order := range result.Items {
		items, err := q.filterOrderItemsByModelIDAndTransferredAt(ctx, order.Items, modelID, transferredAt)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			continue
		}

		orders = append(orders, InquiryOrderSummary{
			ID:        order.ID,
			UserID:    order.UserID,
			AvatarID:  order.AvatarID,
			CartID:    order.CartID,
			Paid:      order.Paid,
			Items:     items,
			CreatedAt: order.CreatedAt,
		})
	}

	return orders, nil
}

func (q *InquiryManagementQuery) filterOrderItemsByModelIDAndTransferredAt(
	ctx context.Context,
	items []orderdom.OrderItemSnapshot,
	modelID string,
	transferredAt *time.Time,
) ([]InquiryOrderItemSummary, error) {
	if modelID == "" || transferredAt == nil || transferredAt.IsZero() || len(items) == 0 {
		return []InquiryOrderItemSummary{}, nil
	}

	expectedTransferredAt := transferredAt.UTC()

	filtered := make([]InquiryOrderItemSummary, 0, len(items))
	for _, item := range items {
		if item.ModelID != modelID {
			continue
		}

		if item.TransferredAt == nil || item.TransferredAt.IsZero() {
			continue
		}

		itemTransferredAt := item.TransferredAt.UTC()
		if !itemTransferredAt.Equal(expectedTransferredAt) {
			continue
		}

		tokenBlueprintID := tokenBlueprintIDFromInventoryID(item.InventoryID)
		tokenName, err := q.resolveTokenNameByTokenBlueprintID(ctx, tokenBlueprintID)
		if err != nil {
			return nil, err
		}

		filtered = append(filtered, InquiryOrderItemSummary{
			ModelID:          item.ModelID,
			InventoryID:      item.InventoryID,
			TokenBlueprintID: tokenBlueprintID,
			TokenName:        tokenName,
			ListID:           item.ListID,
			Qty:              item.Qty,
			Price:            item.Price,
			IsCanceled:       item.IsCanceled,
			IsDispatched:     item.IsDispatched,
			Transferred:      item.Transferred,
			TransferredAt:    item.TransferredAt,
		})
	}

	return filtered, nil
}

func tokenBlueprintIDFromInventoryID(inventoryID string) string {
	inventoryID = strings.TrimSpace(inventoryID)
	if inventoryID == "" {
		return ""
	}

	index := strings.LastIndex(inventoryID, "_")
	if index < 0 || index == len(inventoryID)-1 {
		return ""
	}

	return strings.TrimSpace(inventoryID[index+1:])
}

func (q *InquiryManagementQuery) resolveTokenNameByTokenBlueprintID(
	ctx context.Context,
	tokenBlueprintID string,
) (string, error) {
	if tokenBlueprintID == "" {
		return "", nil
	}

	if q == nil {
		return "", fmt.Errorf("inquiry management query: query is nil")
	}

	if q.tokenBlueprintRepo == nil {
		return "", fmt.Errorf("inquiry management query: token blueprint repository is nil")
	}

	tokenBlueprint, err := q.tokenBlueprintRepo.GetByID(ctx, tokenBlueprintID)
	if err != nil {
		return "", err
	}

	if tokenBlueprint == nil {
		return "", nil
	}

	return tokenBlueprint.Name, nil
}
