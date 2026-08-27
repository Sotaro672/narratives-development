// backend/internal/application/query/console/inquiry_detail_query.go
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
	inventorydom "narratives/internal/domain/inventory"
	modeldom "narratives/internal/domain/model"
	orderdom "narratives/internal/domain/order"
	productdom "narratives/internal/domain/product"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	tokendom "narratives/internal/domain/token"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
	transferdom "narratives/internal/domain/transfer"
	userdom "narratives/internal/domain/user"
)

// InquiryDetailQuery は Console 問い合わせ詳細画面向けの read model を扱います。
//
// 詳細画面で必要な重い解決をこちらに集約します。
// product Inquiry は Inquiry.ProductID から従来どおり商品・注文情報を解決します。
// return Inquiry は Inquiry.OrderID + Inquiry.OrderItemIndex を正として Order item を直接解決し、
// ProductID が存在しない未開封返品でも Model / List / Resale / ProductBlueprint / Brand を解決します。
// ProductBlueprint.CompanyID / ProductName / BrandID を解決し、
// さらに Brand.GetByID() から BrandName / BrandIcon を解決します。
// ProductID が存在する場合は tokens/{productId} から Token.AssetID を取得します。
// product Inquiry では AssetID から Transfer.TransferredAt を解決します。
// return Inquiry では対象 Order item の TransferredAt を正として扱います。
// Inquiry.AvatarID から Avatar.GetByID() を使って UserID を解決します。
// 解決した UserID から User.GetByID() を使って UserName を解決します。
// Console read model には AvatarName / AvatarIcon を返しません。
// ReplyRepository から replies を取得し、詳細 BFF の完成 DTO に含めます。
// Order item の tokenBlueprintId / tokenName は Order snapshot を優先し、必要時のみ InventoryID から補完し、TokenBlueprint.BrandID から BrandName を解決します。
type InquiryDetailQuery struct {
	repo                 inquirydom.Repository
	replyRepo            inquirydom.ReplyRepository
	inventoryRepo        inventorydom.RepositoryPort
	productRepo          productdom.Repository
	modelRepo            modeldom.RepositoryPort
	productBlueprintRepo productblueprintdom.Repository
	tokenBlueprintRepo   tokenblueprintdom.RepositoryPort
	tokenQueryRepo       tokendom.TokenQueryPort
	transferQueryRepo    transferdom.RepositoryPort
	brandRepo            branddom.Repository
	avatarRepo           avatardom.Repository
	userRepo             userdom.RepositoryPort
	orderRepo            orderdom.Repository
}

// NewInquiryDetailQuery は InquiryDetailQuery を初期化します。
func NewInquiryDetailQuery(
	repo inquirydom.Repository,
	replyRepo inquirydom.ReplyRepository,
	inventoryRepo inventorydom.RepositoryPort,
	productRepo productdom.Repository,
	modelRepo modeldom.RepositoryPort,
	productBlueprintRepo productblueprintdom.Repository,
	tokenBlueprintRepo tokenblueprintdom.RepositoryPort,
	tokenQueryRepo tokendom.TokenQueryPort,
	transferQueryRepo transferdom.RepositoryPort,
	brandRepo branddom.Repository,
	avatarRepo avatardom.Repository,
	userRepo userdom.RepositoryPort,
	orderRepo orderdom.Repository,
) *InquiryDetailQuery {
	return &InquiryDetailQuery{
		repo: repo, replyRepo: replyRepo, inventoryRepo: inventoryRepo, productRepo: productRepo, modelRepo: modelRepo,
		productBlueprintRepo: productBlueprintRepo, tokenBlueprintRepo: tokenBlueprintRepo,
		tokenQueryRepo: tokenQueryRepo, transferQueryRepo: transferQueryRepo,
		brandRepo: brandRepo, avatarRepo: avatarRepo, userRepo: userRepo,
		orderRepo: orderRepo,
	}
}

// InquiryDetail は Console 管理画面向けの Inquiry 詳細 read model です。
type InquiryDetail struct {
	Inquiry            inquirydom.Inquiry    `json:"inquiry"`
	Replies            []inquirydom.Reply    `json:"replies"`
	ModelID            string                `json:"modelId"`
	ProductBlueprintID string                `json:"productBlueprintId"`
	ProductName        string                `json:"productName"`
	BrandID            string                `json:"brandId"`
	BrandName          string                `json:"brandName"`
	BrandIcon          string                `json:"brandIcon"`
	AssetID            string                `json:"assetId"`
	TransferredAt      *time.Time            `json:"transferredAt,omitempty"`
	UserID             string                `json:"userId"`
	UserName           string                `json:"userName"`
	Orders             []InquiryOrderSummary `json:"orders"`
	CompanyID          string                `json:"companyId"`
}

// InquiryAggregate は Inquiry とその画像一覧をまとめた管理画面向けビューです。
//
// inquiryImage ドメインは inquiry ドメインへ統合済みのため、
// Images は Inquiry.Images から取得します。
type InquiryAggregate struct {
	Inquiry            inquirydom.Inquiry     `json:"inquiry"`
	Images             []inquirydom.ImageFile `json:"images"`
	ModelID            string                 `json:"modelId"`
	ProductBlueprintID string                 `json:"productBlueprintId"`
	ProductName        string                 `json:"productName"`
	BrandID            string                 `json:"brandId"`
	BrandName          string                 `json:"brandName"`
	AssetID            string                 `json:"assetId"`
	TransferredAt      *time.Time             `json:"transferredAt,omitempty"`
	UserID             string                 `json:"userId"`
	UserFullName       string                 `json:"userFullName"`
	Orders             []InquiryOrderSummary  `json:"orders"`
	CompanyID          string                 `json:"companyId"`
}

// InquiryOrderSummary は Inquiry 詳細画面向けの注文 read model です。
//
// Order.ShippingSnapshot は問い合わせ詳細では使用しないため含めません。
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

// InquiryOrderItemSummary は Inquiry 詳細画面向けの注文 item read model です。
type InquiryOrderItemSummary struct {
	ItemIndex               int                    `json:"itemIndex"`
	ItemType                orderdom.OrderItemType `json:"itemType"`
	ModelID                 string                 `json:"modelId"`
	InventoryID             string                 `json:"inventoryId"`
	ListID                  string                 `json:"listId"`
	ResaleID                string                 `json:"resaleId"`
	ProductID               string                 `json:"productId"`
	ProductBlueprintID      string                 `json:"productBlueprintId"`
	TokenBlueprintID        string                 `json:"tokenBlueprintId"`
	TokenName               string                 `json:"tokenName"`
	TokenBrandID            string                 `json:"tokenBrandId"`
	TokenBrandName          string                 `json:"tokenBrandName"`
	BrandID                 string                 `json:"brandId"`
	Qty                     int                    `json:"qty"`
	Price                   int                    `json:"price"`
	IsCancelled             bool                   `json:"isCancelled"`
	IsDispatched            bool                   `json:"isDispatched"`
	IsReturnRequested       bool                   `json:"isReturnRequested"`
	ReturnRequestedAt       *time.Time             `json:"returnRequestedAt,omitempty"`
	IsReturnCompleted       bool                   `json:"isReturnCompleted"`
	ReturnCompletedAt       *time.Time             `json:"returnCompletedAt,omitempty"`
	TokenTransferVerifiedAt *time.Time             `json:"tokenTransferVerifiedAt,omitempty"`
	Transferred             bool                   `json:"transferred"`
	TransferredAt           *time.Time             `json:"transferredAt,omitempty"`
}

// GetByID は Inquiry を返します。
func (q *InquiryDetailQuery) GetByID(ctx context.Context, id string) (inquirydom.Inquiry, error) {
	if q == nil || q.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("inquiry detail query: repository is nil")
	}
	if id == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidID
	}
	return q.repo.GetByID(ctx, id)
}

// GetDetailByID は Inquiry 詳細 read model を返します。
func (q *InquiryDetailQuery) GetDetailByID(ctx context.Context, id string) (InquiryDetail, error) {
	detail, err := q.getDetailBaseByID(ctx, id)
	if err != nil {
		return InquiryDetail{}, err
	}
	replies, err := q.resolveRepliesByInquiryID(ctx, detail.Inquiry.ID)
	if err != nil {
		return InquiryDetail{}, err
	}
	detail.Replies = replies
	return detail, nil
}

// GetDetailByIDForCompany は company boundary 確認込みで Inquiry 詳細 read model を返します。
// company boundary を確認してから replies を取得し、他 company の reply subcollection を先に読まないようにします。
func (q *InquiryDetailQuery) GetDetailByIDForCompany(ctx context.Context, id string, companyID string) (InquiryDetail, error) {
	if companyID == "" {
		return InquiryDetail{}, fmt.Errorf("inquiry detail query: companyId is empty")
	}

	detail, err := q.getDetailBaseByID(ctx, id)
	if err != nil {
		return InquiryDetail{}, err
	}
	if detail.CompanyID != companyID {
		return InquiryDetail{}, inquirydom.ErrNotFound
	}

	replies, err := q.resolveRepliesByInquiryID(ctx, detail.Inquiry.ID)
	if err != nil {
		return InquiryDetail{}, err
	}
	detail.Replies = replies
	return detail, nil
}

func (q *InquiryDetailQuery) getDetailBaseByID(ctx context.Context, id string) (InquiryDetail, error) {
	inq, err := q.GetByID(ctx, id)
	if err != nil {
		return InquiryDetail{}, err
	}

	resolved, err := q.resolveInquiryDetailRefs(ctx, inq)
	if err != nil {
		return InquiryDetail{}, err
	}

	userID, userName, err := q.resolveUserRefByAvatarID(ctx, inq.AvatarID)
	if err != nil {
		return InquiryDetail{}, err
	}

	brandIcon, err := q.resolveBrandIconByBrandID(ctx, resolved.BrandID)
	if err != nil {
		return InquiryDetail{}, err
	}

	return InquiryDetail{
		Inquiry: inq, Replies: []inquirydom.Reply{}, ModelID: resolved.ModelID, ProductBlueprintID: resolved.ProductBlueprintID,
		ProductName: resolved.ProductName, BrandID: resolved.BrandID, BrandName: resolved.BrandName, BrandIcon: brandIcon,
		AssetID: resolved.AssetID, TransferredAt: resolved.TransferredAt,
		UserID: userID, UserName: userName,
		Orders: resolved.Orders, CompanyID: resolved.CompanyID,
	}, nil
}

// GetImages は Inquiry に紐づく画像一覧を返します。
//
// inquiryImage ドメインは廃止済みのため、別 repository へは問い合わせず、
// Inquiry.Images をそのまま返します。
func (q *InquiryDetailQuery) GetImages(ctx context.Context, inquiryID string) ([]inquirydom.ImageFile, error) {
	if q == nil || q.repo == nil {
		return nil, fmt.Errorf("inquiry detail query: repository is nil")
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
func (q *InquiryDetailQuery) GetImagesForCompany(ctx context.Context, inquiryID string, companyID string) ([]inquirydom.ImageFile, error) {
	detail, err := q.getDetailBaseByID(ctx, inquiryID)
	if err != nil {
		return nil, err
	}
	if detail.CompanyID != companyID {
		return nil, inquirydom.ErrNotFound
	}
	if len(detail.Inquiry.Images) == 0 {
		return []inquirydom.ImageFile{}, nil
	}
	return detail.Inquiry.Images, nil
}

// GetAggregate は Inquiry と画像一覧をまとめて返します。
//
// 画像は Inquiry.Images を正として扱います。
func (q *InquiryDetailQuery) GetAggregate(ctx context.Context, id string) (InquiryAggregate, error) {
	if q == nil || q.repo == nil {
		return InquiryAggregate{}, fmt.Errorf("inquiry detail query: repository is nil")
	}

	inq, err := q.GetByID(ctx, id)
	if err != nil {
		return InquiryAggregate{}, err
	}

	images := inq.Images
	if images == nil {
		images = []inquirydom.ImageFile{}
	}

	resolved, err := q.resolveInquiryDetailRefs(ctx, inq)
	if err != nil {
		return InquiryAggregate{}, err
	}
	userID, userFullName, err := q.resolveUserRefByAvatarID(ctx, inq.AvatarID)
	if err != nil {
		return InquiryAggregate{}, err
	}

	return InquiryAggregate{
		Inquiry: inq, Images: images, ModelID: resolved.ModelID,
		ProductBlueprintID: resolved.ProductBlueprintID, ProductName: resolved.ProductName,
		BrandID: resolved.BrandID, BrandName: resolved.BrandName, AssetID: resolved.AssetID,
		TransferredAt: resolved.TransferredAt, UserID: userID,
		UserFullName: userFullName,
		Orders:       resolved.Orders, CompanyID: resolved.CompanyID,
	}, nil
}

// GetAggregateForCompany は company boundary 確認込みで Inquiry aggregate を返します。
func (q *InquiryDetailQuery) GetAggregateForCompany(ctx context.Context, id string, companyID string) (InquiryAggregate, error) {
	if companyID == "" {
		return InquiryAggregate{}, fmt.Errorf("inquiry detail query: companyId is empty")
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

type inquiryDetailResolvedRefs struct {
	ModelID            string
	ProductBlueprintID string
	ProductName        string
	BrandID            string
	BrandName          string
	AssetID            string
	TransferredAt      *time.Time
	Orders             []InquiryOrderSummary
	CompanyID          string
}

func (q *InquiryDetailQuery) resolveInquiryDetailRefs(
	ctx context.Context,
	inq inquirydom.Inquiry,
) (inquiryDetailResolvedRefs, error) {
	switch inq.InquiryType {
	case inquirydom.InquiryTypeReturnUnopened,
		inquirydom.InquiryTypeReturnOpened:
		return q.resolveReturnInquiryDetailRefs(
			ctx,
			inq,
		)

	default:
		return q.resolveProductInquiryDetailRefs(
			ctx,
			inq,
		)
	}
}

func (q *InquiryDetailQuery) resolveProductInquiryDetailRefs(
	ctx context.Context,
	inq inquirydom.Inquiry,
) (inquiryDetailResolvedRefs, error) {
	modelID, productBlueprintID, productName, brandID, brandName, companyID, err :=
		q.resolveProductModelRefByInquiryProductID(
			ctx,
			inq.ProductID,
		)
	if err != nil {
		return inquiryDetailResolvedRefs{}, err
	}

	assetID, err :=
		q.resolveAssetIDByProductID(
			ctx,
			inq.ProductID,
		)
	if err != nil {
		return inquiryDetailResolvedRefs{}, err
	}

	transferredAt, err :=
		q.resolveTransferredAtByAssetID(
			ctx,
			assetID,
		)
	if err != nil {
		return inquiryDetailResolvedRefs{}, err
	}

	orders, err :=
		q.resolveOrdersByAvatarIDModelIDAndTransferredAt(
			ctx,
			inq.AvatarID,
			modelID,
			transferredAt,
		)
	if err != nil {
		return inquiryDetailResolvedRefs{}, err
	}

	return inquiryDetailResolvedRefs{
		ModelID:            modelID,
		ProductBlueprintID: productBlueprintID,
		ProductName:        productName,
		BrandID:            brandID,
		BrandName:          brandName,
		AssetID:            assetID,
		TransferredAt:      transferredAt,
		Orders:             orders,
		CompanyID:          companyID,
	}, nil
}

func (q *InquiryDetailQuery) resolveReturnInquiryDetailRefs(
	ctx context.Context,
	inq inquirydom.Inquiry,
) (inquiryDetailResolvedRefs, error) {
	if q == nil {
		return inquiryDetailResolvedRefs{},
			fmt.Errorf("inquiry detail query: query is nil")
	}
	if q.orderRepo == nil {
		return inquiryDetailResolvedRefs{},
			fmt.Errorf("inquiry detail query: order repository is nil")
	}
	if inq.OrderID == "" {
		return inquiryDetailResolvedRefs{},
			inquirydom.ErrInvalidOrderID
	}
	if inq.OrderItemIndex == nil ||
		*inq.OrderItemIndex < 0 {
		return inquiryDetailResolvedRefs{},
			inquirydom.ErrInvalidOrderItemIndex
	}

	order, err :=
		q.orderRepo.GetByID(
			ctx,
			inq.OrderID,
		)
	if err != nil {
		if errors.Is(err, orderdom.ErrNotFound) {
			return inquiryDetailResolvedRefs{},
				inquirydom.ErrNotFound
		}

		return inquiryDetailResolvedRefs{}, err
	}

	if order.AvatarID != inq.AvatarID {
		return inquiryDetailResolvedRefs{},
			inquirydom.ErrNotFound
	}

	itemIndex := *inq.OrderItemIndex
	if itemIndex >= len(order.Items) {
		return inquiryDetailResolvedRefs{},
			inquirydom.ErrInvalidOrderItemIndex
	}

	item := order.Items[itemIndex]

	productID := inq.ProductID
	if productID == "" {
		productID = item.ProductID
	} else if item.ProductID != "" &&
		item.ProductID != productID {
		return inquiryDetailResolvedRefs{},
			inquirydom.ErrInquiryInvalidWorkflow
	}

	modelID,
		productBlueprintID,
		productName,
		brandID,
		brandName,
		companyID,
		err :=
		q.resolveProductModelRefByOrderItem(
			ctx,
			item,
			productID,
		)
	if err != nil {
		return inquiryDetailResolvedRefs{}, err
	}

	assetID, err :=
		q.resolveAssetIDByProductID(
			ctx,
			productID,
		)
	if err != nil {
		return inquiryDetailResolvedRefs{}, err
	}

	var transferredAt *time.Time
	if item.TransferredAt != nil &&
		!item.TransferredAt.IsZero() {
		value := item.TransferredAt.UTC()
		transferredAt = &value
	}

	itemSummary, err :=
		q.buildInquiryOrderItemSummary(
			ctx,
			itemIndex,
			item,
		)
	if err != nil {
		return inquiryDetailResolvedRefs{}, err
	}

	orders := []InquiryOrderSummary{
		{
			ID:        order.ID,
			UserID:    order.UserID,
			AvatarID:  order.AvatarID,
			CartID:    order.CartID,
			Paid:      order.Paid,
			Items:     []InquiryOrderItemSummary{itemSummary},
			CreatedAt: order.CreatedAt,
		},
	}

	return inquiryDetailResolvedRefs{
		ModelID:            modelID,
		ProductBlueprintID: productBlueprintID,
		ProductName:        productName,
		BrandID:            brandID,
		BrandName:          brandName,
		AssetID:            assetID,
		TransferredAt:      transferredAt,
		Orders:             orders,
		CompanyID:          companyID,
	}, nil
}

func (q *InquiryDetailQuery) resolveProductModelRefByOrderItem(
	ctx context.Context,
	item orderdom.OrderItemSnapshot,
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
		return "", "", "", "", "", "",
			fmt.Errorf("inquiry detail query: query is nil")
	}

	modelID = item.ModelID
	productBlueprintID =
		item.ProductBlueprintID

	if modelID == "" &&
		productID != "" {
		if q.productRepo == nil {
			return "", "", "", "", "", "",
				fmt.Errorf("inquiry detail query: product repository is nil")
		}

		product, productErr :=
			q.productRepo.GetByID(
				ctx,
				productID,
			)
		if productErr != nil {
			if !errors.Is(
				productErr,
				productdom.ErrNotFound,
			) {
				return "", "", "", "", "", "",
					productErr
			}
		} else {
			modelID = product.ModelID
		}
	}

	if productBlueprintID == "" &&
		modelID != "" {
		if q.modelRepo == nil {
			return modelID, "", "", "", "", "",
				fmt.Errorf("inquiry detail query: model repository is nil")
		}

		model, modelErr :=
			q.modelRepo.GetByID(
				ctx,
				modelID,
			)
		if modelErr != nil {
			if !errors.Is(
				modelErr,
				modeldom.ErrNotFound,
			) {
				return modelID, "", "", "", "", "",
					modelErr
			}
		} else {
			productBlueprintID =
				model.GetProductBlueprintID()
		}
	}

	if productBlueprintID == "" {
		return modelID, "", "", "", "", "", nil
	}

	return q.resolveProductBlueprintDisplay(
		ctx,
		modelID,
		productBlueprintID,
	)
}

func (q *InquiryDetailQuery) resolveProductBlueprintDisplay(
	ctx context.Context,
	modelID string,
	productBlueprintID string,
) (
	resolvedModelID string,
	resolvedProductBlueprintID string,
	productName string,
	brandID string,
	brandName string,
	companyID string,
	err error,
) {
	if q == nil {
		return "", "", "", "", "", "",
			fmt.Errorf("inquiry detail query: query is nil")
	}
	if productBlueprintID == "" {
		return modelID, "", "", "", "", "", nil
	}
	if q.productBlueprintRepo == nil {
		return modelID, productBlueprintID, "", "", "", "",
			fmt.Errorf("inquiry detail query: product blueprint repository is nil")
	}

	productBlueprint, err :=
		q.productBlueprintRepo.GetByID(
			ctx,
			productBlueprintID,
		)
	if err != nil {
		return modelID, productBlueprintID, "", "", "", "", err
	}

	productName = productBlueprint.ProductName
	brandID = productBlueprint.BrandID
	companyID = productBlueprint.CompanyID

	if brandID == "" {
		return modelID, productBlueprintID,
			productName, "", "", companyID, nil
	}
	if q.brandRepo == nil {
		return modelID, productBlueprintID,
			productName, brandID, "", companyID,
			fmt.Errorf("inquiry detail query: brand repository is nil")
	}

	brand, err :=
		q.brandRepo.GetByID(
			ctx,
			brandID,
		)
	if err != nil {
		if errors.Is(err, branddom.ErrNotFound) {
			return modelID, productBlueprintID,
				productName, brandID, "", companyID, nil
		}

		return modelID, productBlueprintID,
			productName, brandID, "", companyID, err
	}

	brandName = brand.Name

	return modelID, productBlueprintID,
		productName, brandID, brandName, companyID, nil
}

func (q *InquiryDetailQuery) resolveProductModelRefByInquiryProductID(
	ctx context.Context,
	productID string,
) (modelID string, productBlueprintID string, productName string, brandID string, brandName string, companyID string, err error) {
	if q == nil {
		return "", "", "", "", "", "", fmt.Errorf("inquiry detail query: query is nil")
	}
	if productID == "" {
		return "", "", "", "", "", "", nil
	}
	if q.productRepo == nil {
		return "", "", "", "", "", "", fmt.Errorf("inquiry detail query: product repository is nil")
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
		return modelID, "", "", "", "", "", fmt.Errorf("inquiry detail query: model repository is nil")
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
		return modelID, productBlueprintID, "", "", "", "", fmt.Errorf("inquiry detail query: product blueprint repository is nil")
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
		return modelID, productBlueprintID, productName, brandID, "", companyID, fmt.Errorf("inquiry detail query: brand repository is nil")
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

func (q *InquiryDetailQuery) resolveAssetIDByProductID(ctx context.Context, productID string) (string, error) {
	if q == nil {
		return "", fmt.Errorf("inquiry detail query: query is nil")
	}
	if productID == "" {
		return "", nil
	}
	if q.tokenQueryRepo == nil {
		return "", fmt.Errorf("inquiry detail query: token query repository is nil")
	}

	token, err := q.tokenQueryRepo.GetTokenByProductID(ctx, productID)
	if err != nil {
		if errors.Is(err, tokendom.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	return strings.Trim(token.AssetID, " \t\r\n"), nil
}

func (q *InquiryDetailQuery) resolveTransferredAtByAssetID(ctx context.Context, assetID string) (*time.Time, error) {
	if q == nil {
		return nil, fmt.Errorf("inquiry detail query: query is nil")
	}

	assetID = strings.Trim(assetID, " \t\r\n")
	if assetID == "" {
		return nil, nil
	}
	if q.transferQueryRepo == nil {
		return nil, fmt.Errorf("inquiry detail query: transfer query repository is nil")
	}

	result, err := q.transferQueryRepo.ResolveTransferredAtByAssetID(ctx, assetID)
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

func (q *InquiryDetailQuery) resolveBrandIconByBrandID(
	ctx context.Context,
	brandID string,
) (string, error) {
	if q == nil {
		return "", fmt.Errorf("inquiry detail query: query is nil")
	}
	if brandID == "" {
		return "", nil
	}
	if q.brandRepo == nil {
		return "", fmt.Errorf("inquiry detail query: brand repository is nil")
	}

	brand, err := q.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		if errors.Is(err, branddom.ErrNotFound) {
			return "", nil
		}
		return "", err
	}

	return brand.BrandIcon, nil
}

func (q *InquiryDetailQuery) resolveUserRefByAvatarID(
	ctx context.Context,
	avatarID string,
) (userID string, userName string, err error) {
	if q == nil {
		return "", "", fmt.Errorf("inquiry detail query: query is nil")
	}
	if avatarID == "" {
		return "", "", nil
	}
	if q.avatarRepo == nil {
		return "", "", fmt.Errorf("inquiry detail query: avatar repository is nil")
	}

	avatar, err := q.avatarRepo.GetByID(ctx, avatarID)
	if err != nil {
		return "", "", err
	}

	userID = avatar.UserID
	if userID == "" {
		return "", "", nil
	}
	if q.userRepo == nil {
		return userID, "", fmt.Errorf("inquiry detail query: user repository is nil")
	}

	user, err := q.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, userdom.ErrNotFound) {
			return userID, "", nil
		}
		return userID, "", err
	}

	userName = userdom.FormatName(user)
	return userID, userName, nil
}

func (q *InquiryDetailQuery) resolveRepliesByInquiryID(ctx context.Context, inquiryID string) ([]inquirydom.Reply, error) {
	if q == nil {
		return nil, fmt.Errorf("inquiry detail query: query is nil")
	}
	if inquiryID == "" {
		return []inquirydom.Reply{}, nil
	}
	if q.replyRepo == nil {
		return nil, fmt.Errorf("inquiry detail query: reply repository is nil")
	}

	replies, err := q.replyRepo.ListByInquiryID(ctx, inquiryID)
	if err != nil {
		return nil, err
	}
	if replies == nil {
		return []inquirydom.Reply{}, nil
	}
	return replies, nil
}

func (q *InquiryDetailQuery) resolveOrdersByAvatarIDModelIDAndTransferredAt(
	ctx context.Context,
	avatarID string,
	modelID string,
	transferredAt *time.Time,
) ([]InquiryOrderSummary, error) {
	if q == nil {
		return nil, fmt.Errorf("inquiry detail query: query is nil")
	}
	if avatarID == "" || modelID == "" || transferredAt == nil || transferredAt.IsZero() {
		return []InquiryOrderSummary{}, nil
	}
	if q.orderRepo == nil {
		return nil, fmt.Errorf("inquiry detail query: order repository is nil")
	}

	result, err := q.orderRepo.ListByAvatarID(
		ctx,
		avatarID,
		orderdom.Sort{Column: orderdom.SortByCreatedAt, Order: orderdom.SortDesc},
		orderdom.Page{Number: 1, PerPage: 100},
	)
	if err != nil {
		if errors.Is(err, orderdom.ErrNotFound) {
			return []InquiryOrderSummary{}, nil
		}
		return nil, err
	}

	orders := make([]InquiryOrderSummary, 0, len(result.Items))
	for _, order := range result.Items {
		items, err := q.filterInquiryOrderItemsByModelIDAndTransferredAt(ctx, order.Items, modelID, transferredAt)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			continue
		}

		orders = append(orders, InquiryOrderSummary{
			ID: order.ID, UserID: order.UserID, AvatarID: order.AvatarID,
			CartID: order.CartID, Paid: order.Paid, Items: items, CreatedAt: order.CreatedAt,
		})
	}
	return orders, nil
}

func (q *InquiryDetailQuery) filterInquiryOrderItemsByModelIDAndTransferredAt(
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

	for itemIndex, item := range items {
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

		summary, err :=
			q.buildInquiryOrderItemSummary(
				ctx,
				itemIndex,
				item,
			)
		if err != nil {
			return nil, err
		}

		filtered = append(
			filtered,
			summary,
		)
	}
	return filtered, nil
}

func (q *InquiryDetailQuery) buildInquiryOrderItemSummary(
	ctx context.Context,
	itemIndex int,
	item orderdom.OrderItemSnapshot,
) (InquiryOrderItemSummary, error) {
	tokenBlueprintID,
		tokenName,
		tokenBrandID,
		tokenBrandName,
		err :=
		q.resolveTokenBlueprintSnapshotByOrderItem(
			ctx,
			item,
		)
	if err != nil {
		return InquiryOrderItemSummary{}, err
	}

	return InquiryOrderItemSummary{
		ItemIndex: itemIndex,
		ItemType:  item.Type,

		ModelID:     item.ModelID,
		InventoryID: item.InventoryID,
		ListID:      item.ListID,
		ResaleID:    item.ResaleID,

		ProductID:          item.ProductID,
		ProductBlueprintID: item.ProductBlueprintID,
		TokenBlueprintID:   tokenBlueprintID,
		TokenName:          tokenName,
		TokenBrandID:       tokenBrandID,
		TokenBrandName:     tokenBrandName,
		BrandID:            item.BrandID,

		Qty:   item.Qty,
		Price: item.Price,

		IsCancelled:       item.IsCancelled,
		IsDispatched:      item.IsDispatched,
		IsReturnRequested: item.IsReturnRequested,
		ReturnRequestedAt: normalizeInquiryDetailTimePointer(
			item.ReturnRequestedAt,
		),
		IsReturnCompleted: item.IsReturnCompleted,
		ReturnCompletedAt: normalizeInquiryDetailTimePointer(
			item.ReturnCompletedAt,
		),
		TokenTransferVerifiedAt: normalizeInquiryDetailTimePointer(
			item.TokenTransferVerifiedAt,
		),

		Transferred: item.Transferred,
		TransferredAt: normalizeInquiryDetailTimePointer(
			item.TransferredAt,
		),
	}, nil
}

func (q *InquiryDetailQuery) resolveTokenBlueprintSnapshotByOrderItem(
	ctx context.Context,
	item orderdom.OrderItemSnapshot,
) (
	tokenBlueprintID string,
	tokenName string,
	tokenBrandID string,
	tokenBrandName string,
	err error,
) {
	if q == nil {
		return "", "", "", "",
			fmt.Errorf("inquiry detail query: query is nil")
	}

	tokenBlueprintID = item.TokenBlueprintID

	if tokenBlueprintID == "" {
		return q.resolveTokenBlueprintSnapshotByInventoryID(
			ctx,
			item.InventoryID,
		)
	}

	return q.resolveTokenBlueprintDisplayByID(
		ctx,
		tokenBlueprintID,
	)
}

func normalizeInquiryDetailTimePointer(
	in *time.Time,
) *time.Time {
	if in == nil ||
		in.IsZero() {
		return nil
	}

	value := in.UTC()

	return &value
}

func (q *InquiryDetailQuery) resolveTokenBlueprintSnapshotByInventoryID(
	ctx context.Context,
	inventoryID string,
) (
	tokenBlueprintID string,
	tokenName string,
	tokenBrandID string,
	tokenBrandName string,
	err error,
) {
	if q == nil {
		return "", "", "", "", fmt.Errorf("inquiry detail query: query is nil")
	}
	if inventoryID == "" {
		return "", "", "", "", nil
	}
	if q.inventoryRepo == nil {
		return "", "", "", "", fmt.Errorf("inquiry detail query: inventory repository is nil")
	}

	_, tokenBlueprintID, err = q.inventoryRepo.ResolveBlueprintIDsByInventoryID(ctx, inventoryID)
	if err != nil {
		if errors.Is(err, inventorydom.ErrNotFound) {
			return "", "", "", "", nil
		}
		return "", "", "", "", err
	}
	if tokenBlueprintID == "" {
		return "", "", "", "", inventorydom.ErrInvalidTokenBlueprintID
	}

	return q.resolveTokenBlueprintDisplayByID(
		ctx,
		tokenBlueprintID,
	)
}

func (q *InquiryDetailQuery) resolveTokenBlueprintDisplayByID(
	ctx context.Context,
	tokenBlueprintID string,
) (
	resolvedTokenBlueprintID string,
	tokenName string,
	tokenBrandID string,
	tokenBrandName string,
	err error,
) {
	if q == nil {
		return "", "", "", "",
			fmt.Errorf("inquiry detail query: query is nil")
	}
	if tokenBlueprintID == "" {
		return "", "", "", "", nil
	}
	if q.tokenBlueprintRepo == nil {
		return "", "", "", "",
			fmt.Errorf("inquiry detail query: token blueprint repository is nil")
	}

	tb, err :=
		q.tokenBlueprintRepo.GetByID(
			ctx,
			tokenBlueprintID,
		)
	if err != nil {
		return "", "", "", "", err
	}
	if tb == nil {
		return "", "", "", "",
			fmt.Errorf(
				"inquiry detail query: token blueprint is nil: tokenBlueprintId=%q",
				tokenBlueprintID,
			)
	}

	resolvedTokenBlueprintID = tb.ID
	tokenName = tb.Name
	tokenBrandID = tb.BrandID

	if tokenBrandID == "" {
		return resolvedTokenBlueprintID,
			tokenName, "", "", nil
	}
	if q.brandRepo == nil {
		return resolvedTokenBlueprintID,
			tokenName, tokenBrandID, "",
			fmt.Errorf("inquiry detail query: brand repository is nil")
	}

	brand, err :=
		q.brandRepo.GetByID(
			ctx,
			tokenBrandID,
		)
	if err != nil {
		if errors.Is(err, branddom.ErrNotFound) {
			return resolvedTokenBlueprintID,
				tokenName, tokenBrandID, "", nil
		}

		return resolvedTokenBlueprintID,
			tokenName, tokenBrandID, "", err
	}

	tokenBrandName = brand.Name

	return resolvedTokenBlueprintID,
		tokenName, tokenBrandID, tokenBrandName, nil
}
