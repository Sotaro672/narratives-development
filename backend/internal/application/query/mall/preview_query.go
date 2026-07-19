// backend/internal/application/query/mall/preview_query.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"strings"

	dto "narratives/internal/application/query/mall/dto"
	sharedquery "narratives/internal/application/query/shared"
	appresolver "narratives/internal/application/resolver"
	appusecase "narratives/internal/application/usecase"
	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	commondom "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
	productdom "narratives/internal/domain/product"
	pbdom "narratives/internal/domain/productBlueprint"
	pbcatdom "narratives/internal/domain/productBlueprintCategory"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ------------------------------------------------------------
// Errors
// ------------------------------------------------------------

var (
	ErrPreviewQueryNotConfigured         = errors.New("preview_query: not configured")
	ErrInvalidProductID                  = errors.New("preview_query: invalid productId")
	ErrInvalidModelID                    = errors.New("preview_query: invalid modelId")
	ErrModelIDEmpty                      = errors.New("preview_query: resolved modelId is empty")
	ErrModelVariationNotFound            = errors.New("preview_query: model variation not found")
	ErrProductBlueprintRepoNotConfigured = errors.New("preview_query: productBlueprint repo not configured")
	ErrProductBlueprintIDEmpty           = errors.New("preview_query: resolved productBlueprintId is empty")

	ErrOrderPurchasedQueryNotConfigured = errors.New("order_purchased_query: not configured")
	ErrInvalidAvatarID                  = errors.New("order_purchased_query: invalid avatarId")

	ErrOrderScanVerifyQueryNotConfigured  = errors.New("order_scan_verify_query: not configured")
	ErrOrderScanVerifyAvatarIDEmpty       = errors.New("order_scan_verify_query: avatarId is empty")
	ErrOrderScanVerifyProductIDEmpty      = errors.New("order_scan_verify_query: productId is empty")
	ErrOrderScanVerifyTokenNotFound       = errors.New("order_scan_verify_query: token not found for productId")
	ErrOrderScanVerifyTokenBlueprintEmpty = errors.New("order_scan_verify_query: tokenBlueprintId is empty")
)

// ------------------------------------------------------------
// Ports (dependency interfaces)
// ------------------------------------------------------------

// ProductReader is a minimal read port for preview usecases.
// We only need: productId -> product -> modelId.
type ProductReader interface {
	GetByID(ctx context.Context, productID string) (productdom.Product, error)
}

// ProductBlueprintReader is a minimal read port for ProductBlueprint.
// We need: modelId -> productBlueprintId -> productBlueprint(+patch if needed).
type ProductBlueprintReader interface {
	GetIDByModelID(ctx context.Context, modelID string) (string, []pbdom.ModelRef, error)

	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

// TokenReader is a minimal read port for Token information by productId.
// 想定: tokens/{productId} を読む（存在しない=未mint は nil を返してOK）
type TokenReader interface {
	GetByProductID(ctx context.Context, productID string) (*dto.TokenInfo, error)
}

// BrandReader resolves brandId -> Brand.
//
// brand.RepositoryPort / brand.Repository の GetByID(ctx, id string) に合わせる。
// preview で必要な brandName / brandIcon は GetByID の結果から組み立てる。
type BrandReader interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

// AvatarNameIconReader resolves avatarId -> Avatar.
// avatar 側は GetByID port に統一する。
type AvatarNameIconReader interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
}

// TransferReader resolves mintAddress -> transfer records.
type TransferReader interface {
	ListByMintAddress(
		ctx context.Context,
		mintAddress string,
	) ([]dto.PreviewTransferInfo, error)
}

// OrderTransferItemReader reads the canonical orderTransferItems projection.
// It must not load Order documents and filter their items in memory.
type OrderTransferItemReader interface {
	ListEligibleTransferItemsByAvatarID(
		ctx context.Context,
		avatarID string,
	) ([]orderdom.EligibleTransferItem, error)

	FindEligibleTransferItem(
		ctx context.Context,
		in appusecase.FindEligibleTransferItemInput,
	) (appusecase.TransferTargetItem, error)
}

// ------------------------------------------------------------
// Purchased DTOs
// ------------------------------------------------------------

// PurchasedPair is a resolved purchased item pair derived from an eligible order item.
//
// list item:
// - ItemType == "list"
// - ModelID + TokenBlueprintID で scan item と照合する
//
// resale item:
// - ItemType == "resale"
// - ProductID + TokenBlueprintID で scan item と照合する
type PurchasedPair struct {
	OrderID string `json:"orderId"`

	ItemType  orderdom.OrderItemType `json:"itemType"`
	ItemIndex int                    `json:"itemIndex"`

	// list item identifiers
	ModelID     string `json:"modelId,omitempty"`
	InventoryID string `json:"inventoryId,omitempty"`
	ListID      string `json:"listId,omitempty"`

	// resale item identifiers
	ResaleID string `json:"resaleId,omitempty"`

	// product identifiers
	ProductID          string `json:"productId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId"`
	BrandID            string `json:"brandId,omitempty"`
}

// OrderPurchasedResult is the purchased-side query output.
// - Pairs は orderId/item 単位で返す（同一 pair が複数回出る可能性あり）
type OrderPurchasedResult struct {
	AvatarID string          `json:"avatarId"`
	Pairs    []PurchasedPair `json:"pairs"`
}

// ------------------------------------------------------------
// Query
// ------------------------------------------------------------

// PreviewQuery resolves preview entry info from productId.
// This struct is intended to be injected as cont.PreviewQ.
//
// It also owns scan verification dependencies so NewPreviewQuery is the
// single construction entry point for preview + order scan verification.
type PreviewQuery struct {
	ProductRepo          ProductReader
	ProductBlueprintRepo ProductBlueprintReader

	// order scan verify / purchased-side resolver
	OrderTransferItemRepo OrderTransferItemReader

	// modelId -> apparel/alcohol display fields
	NameResolver *appresolver.NameResolver

	// tokens/{productId} を読む
	TokenRepo TokenReader

	// tokenBlueprint を読む
	TokenBlueprintRepo tbdom.RepositoryPort

	// tokens.toAddress -> owner を解決
	OwnerResolveQ *sharedquery.OwnerResolveQuery

	// display-only name resolvers
	BrandRepo          BrandReader
	AvatarNameIconRepo AvatarNameIconReader

	// mintAddress -> transfers を解決
	TransferRepo TransferReader
}

// ------------------------------------------------------------
// Constructor
// ------------------------------------------------------------

// NewPreviewQuery constructs PreviewQuery.
// This is the only entry point for wiring preview and scan verification dependencies.
func NewPreviewQuery(
	productRepo ProductReader,
	pbRepo ProductBlueprintReader,
	orderTransferItemRepo OrderTransferItemReader,
	nameResolver *appresolver.NameResolver,
	tokenRepo TokenReader,
	tokenBlueprintRepo tbdom.RepositoryPort,
	ownerResolveQ *sharedquery.OwnerResolveQuery,
	brandRepo BrandReader,
	avatarNameIconRepo AvatarNameIconReader,
	transferRepo TransferReader,
) *PreviewQuery {
	return &PreviewQuery{
		ProductRepo:           productRepo,
		ProductBlueprintRepo:  pbRepo,
		OrderTransferItemRepo: orderTransferItemRepo,
		NameResolver:          nameResolver,
		TokenRepo:             tokenRepo,
		TokenBlueprintRepo:    tokenBlueprintRepo,
		OwnerResolveQ:         ownerResolveQ,
		BrandRepo:             brandRepo,
		AvatarNameIconRepo:    avatarNameIconRepo,
		TransferRepo:          transferRepo,
	}
}

// ResolveModelIDByProductID resolves modelId from productId.
func (q *PreviewQuery) ResolveModelIDByProductID(
	ctx context.Context,
	productID string,
) (string, error) {
	if q == nil || q.ProductRepo == nil {
		return "", ErrPreviewQueryNotConfigured
	}

	id := strings.TrimSpace(productID)
	if id == "" {
		return "", ErrInvalidProductID
	}

	p, err := q.ProductRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	modelID := strings.TrimSpace(p.ModelID)
	if modelID == "" {
		return "", ErrModelIDEmpty
	}

	return modelID, nil
}

// ResolveModelInfoByProductID resolves modelId AND variation fields
// from productId.
//
// It supports both apparel and alcohol:
// - apparel: modelNumber / size / color / rgb / measurements
// - alcohol: modelNumber / volumeValue / volumeUnit
func (q *PreviewQuery) ResolveModelInfoByProductID(
	ctx context.Context,
	productID string,
) (*dto.PreviewModelInfo, error) {
	if q == nil || q.ProductRepo == nil || q.NameResolver == nil {
		return nil, ErrPreviewQueryNotConfigured
	}

	id := strings.TrimSpace(productID)
	if id == "" {
		return nil, ErrInvalidProductID
	}

	p, err := q.ProductRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	modelID := strings.TrimSpace(p.ModelID)
	if modelID == "" {
		return nil, ErrModelIDEmpty
	}

	// modelId -> productBlueprintId -> productBlueprint(全フィールド) + patch
	if q.ProductBlueprintRepo == nil {
		return nil, ErrProductBlueprintRepoNotConfigured
	}

	pbID, _, err := q.ProductBlueprintRepo.GetIDByModelID(ctx, modelID)
	if err != nil {
		return nil, err
	}

	pbID = strings.TrimSpace(pbID)
	if pbID == "" {
		return nil, ErrProductBlueprintIDEmpty
	}

	pb, err := q.ProductBlueprintRepo.GetByID(ctx, pbID)
	if err != nil {
		return nil, err
	}

	category := pb.ProductBlueprintCategory

	out := &dto.PreviewModelInfo{
		ProductID: id,
		ModelID:   modelID,

		ProductBlueprintCategoryCode: category.Code,
		ProductBlueprintCategoryKind: category.Kind,
		ProductBlueprintCategoryName: category.NameJa,
		ProductBlueprintCategory:     &category,

		ProductBlueprintID: pbID,
		ProductBlueprint:   &pb,

		// nullではなく[]を返す。
		Transfers: make([]dto.PreviewTransferInfo, 0),
	}

	pbPatch := productBlueprintPatchForPreview(pb)
	out.ProductBlueprintPatch = &pbPatch

	if schema, ok := pbcatdom.GetCategoryInputSchema(category.Code); ok {
		out.CategoryInputSchema = &schema
	}

	if err := q.fillResolvedModelInfo(ctx, out, modelID, category.Kind); err != nil {
		return nil, err
	}

	if q.TokenRepo == nil {
		return out, nil
	}

	// tokens/{productId}（存在すれば付与）
	tok, err := q.TokenRepo.GetByProductID(ctx, id)
	if err != nil {
		return nil, err
	}

	out.Token = tok

	if tok == nil {
		return out, nil
	}

	// brandId -> brandName（tokens側）
	if brandID := strings.TrimSpace(tok.BrandID); brandID != "" {
		if brandName := q.resolveBrandNameForPreview(
			ctx,
			brandID,
			out,
		); brandName != "" {
			tok.BrandName = brandName
		}
	}

	// tokenBlueprint.Patch は domain/tokenBlueprint 側の Patch を再利用する。
	if q.TokenBlueprintRepo != nil {
		tokenBlueprintID := strings.TrimSpace(tok.TokenBlueprintID)

		if tokenBlueprintID != "" {
			tb, perr := q.TokenBlueprintRepo.GetByID(
				ctx,
				tokenBlueprintID,
			)
			if perr == nil && tb != nil {
				tbPatch := tbdom.NewPatchFromTokenBlueprint(tb)

				if strings.TrimSpace(tbPatch.BrandID) != "" &&
					strings.TrimSpace(tbPatch.BrandName) == "" {
					tbPatch.BrandName = q.resolveBrandNameForPreview(
						ctx,
						tbPatch.BrandID,
						out,
					)
				}

				out.TokenBlueprintPatch = &tbPatch
			}
		}
	}

	// 現在owner解決
	q.resolveCurrentOwner(ctx, tok, out)

	// transfer履歴と各owner解決
	q.resolvePreviewTransfers(ctx, tok, out)

	return out, nil
}

func (q *PreviewQuery) resolveCurrentOwner(
	ctx context.Context,
	tok *dto.TokenInfo,
	out *dto.PreviewModelInfo,
) {
	if out == nil || tok == nil {
		return
	}

	if q == nil || q.OwnerResolveQ == nil {
		return
	}

	walletAddress := strings.TrimSpace(tok.ToAddress)
	if walletAddress == "" {
		return
	}

	res, err := q.OwnerResolveQ.Resolve(ctx, walletAddress)
	if err != nil {
		return
	}

	if res == nil {
		return
	}

	switch res.OwnerType {
	case sharedquery.OwnerTypeAvatar:
		res.BrandID = ""
		res.BrandName = ""

	case sharedquery.OwnerTypeBrand:
		res.AvatarID = ""
		res.AvatarName = ""
	}

	out.Owner = res
}

func (q *PreviewQuery) resolvePreviewTransfers(
	ctx context.Context,
	tok *dto.TokenInfo,
	out *dto.PreviewModelInfo,
) {
	if out == nil {
		return
	}

	// 呼び出し元で初期化済みだが、防御的に保証する。
	if out.Transfers == nil {
		out.Transfers = make([]dto.PreviewTransferInfo, 0)
	}

	if q == nil || q.TransferRepo == nil {
		return
	}

	if tok == nil {
		return
	}

	mintAddress := strings.TrimSpace(tok.MintAddress)
	if mintAddress == "" {
		return
	}

	transfers, err := q.TransferRepo.ListByMintAddress(
		ctx,
		mintAddress,
	)
	if err != nil {
		return
	}

	if transfers == nil {
		transfers = make([]dto.PreviewTransferInfo, 0)
	}

	resolved := q.resolveTransferOwners(ctx, transfers)
	if resolved == nil {
		resolved = make([]dto.PreviewTransferInfo, 0)
	}

	out.Transfers = resolved
}

// ListEligiblePairsByAvatarID resolves eligible transfer pairs from the
// canonical orderTransferItems projection.
//
// Projection-side condition:
// - avatarId == avatarID
// - paid == true
// - transferred == false
//
// All identifiers, including tokenBlueprintId and itemIndex, must be stored as
// canonical fields. Legacy derivation and fallback are not supported.
func (q *PreviewQuery) ListEligiblePairsByAvatarID(
	ctx context.Context,
	avatarID string,
) (OrderPurchasedResult, error) {
	if q == nil || q.OrderTransferItemRepo == nil {
		return OrderPurchasedResult{}, ErrOrderPurchasedQueryNotConfigured
	}

	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return OrderPurchasedResult{}, ErrInvalidAvatarID
	}

	items, err := q.OrderTransferItemRepo.ListEligibleTransferItemsByAvatarID(ctx, aid)
	if err != nil {
		return OrderPurchasedResult{}, err
	}

	pairs := make([]PurchasedPair, 0, len(items))

	for _, item := range items {
		switch item.ItemType {
		case orderdom.OrderItemTypeResale:
			pair, ok := purchasedPairFromResaleItem(item)
			if !ok {
				continue
			}
			pairs = append(pairs, pair)

		case orderdom.OrderItemTypeList:
			pair, ok := purchasedPairFromListItem(item)
			if !ok {
				continue
			}
			pairs = append(pairs, pair)

		default:
			continue
		}
	}

	return OrderPurchasedResult{
		AvatarID: aid,
		Pairs:    pairs,
	}, nil
}

// VerifyMatch verifies whether the scanned product exists in purchased(untransferred) pairs.
func (q *PreviewQuery) VerifyMatch(
	ctx context.Context,
	in appusecase.VerifyInput,
) (appusecase.VerifyResult, error) {
	if q == nil ||
		q.OrderTransferItemRepo == nil ||
		q.ProductRepo == nil ||
		q.NameResolver == nil {
		return appusecase.VerifyResult{},
			ErrOrderScanVerifyQueryNotConfigured
	}

	avatarID := strings.TrimSpace(in.AvatarID)
	productID := strings.TrimSpace(in.ProductID)

	if avatarID == "" {
		return appusecase.VerifyResult{},
			ErrOrderScanVerifyAvatarIDEmpty
	}

	if productID == "" {
		return appusecase.VerifyResult{},
			ErrOrderScanVerifyProductIDEmpty
	}

	// 1) scan side: productId -> modelId + tokenBlueprintId(tokens/{productId}.tokenBlueprintId)
	info, err := q.ResolveModelInfoByProductID(ctx, productID)
	if err != nil {
		return appusecase.VerifyResult{},
			fmt.Errorf(
				"order_scan_verify_query: preview resolve failed: %w",
				err,
			)
	}

	if info == nil {
		return appusecase.VerifyResult{},
			fmt.Errorf(
				"order_scan_verify_query: preview resolve returned nil",
			)
	}

	scannedModelID := strings.TrimSpace(info.ModelID)
	if scannedModelID == "" {
		return appusecase.VerifyResult{},
			fmt.Errorf(
				"order_scan_verify_query: scanned modelId is empty",
			)
	}

	if info.Token == nil {
		return appusecase.VerifyResult{},
			ErrOrderScanVerifyTokenNotFound
	}

	scannedTokenBlueprintID := strings.TrimSpace(
		info.Token.TokenBlueprintID,
	)
	if scannedTokenBlueprintID == "" {
		return appusecase.VerifyResult{},
			ErrOrderScanVerifyTokenBlueprintEmpty
	}

	// 2) purchased side: perform one indexed lookup against orderTransferItems.
	target, err := q.OrderTransferItemRepo.FindEligibleTransferItem(
		ctx,
		appusecase.FindEligibleTransferItemInput{
			AvatarID:         avatarID,
			ProductID:        productID,
			ModelID:          scannedModelID,
			TokenBlueprintID: scannedTokenBlueprintID,
		},
	)
	if err != nil {
		if errors.Is(err, orderdom.ErrNotFound) {
			return appusecase.VerifyResult{
				AvatarID:                avatarID,
				ProductID:               productID,
				ScannedModelID:          scannedModelID,
				ScannedTokenBlueprintID: scannedTokenBlueprintID,
				PurchasedPairs:          make([]appusecase.ModelTokenPair, 0),
				Matched:                 false,
				Match:                   nil,
			}, nil
		}

		return appusecase.VerifyResult{},
			fmt.Errorf(
				"order_scan_verify_query: eligible transfer item lookup failed: %w",
				err,
			)
	}

	if target.OrderID == "" || target.ItemIndex < 0 {
		return appusecase.VerifyResult{},
			fmt.Errorf(
				"order_scan_verify_query: invalid eligible transfer item",
			)
	}

	match := appusecase.ModelTokenPair{
		ModelID:          scannedModelID,
		TokenBlueprintID: scannedTokenBlueprintID,
	}

	return appusecase.VerifyResult{
		AvatarID:                avatarID,
		ProductID:               productID,
		ScannedModelID:          scannedModelID,
		ScannedTokenBlueprintID: scannedTokenBlueprintID,
		PurchasedPairs: []appusecase.ModelTokenPair{
			match,
		},
		Matched: true,
		Match:   &match,
	}, nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func purchasedPairFromListItem(
	item orderdom.EligibleTransferItem,
) (PurchasedPair, bool) {
	modelID := strings.TrimSpace(item.ModelID)
	inventoryID := strings.TrimSpace(item.InventoryID)

	if modelID == "" || inventoryID == "" {
		return PurchasedPair{}, false
	}

	tokenBlueprintID := strings.TrimSpace(
		item.TokenBlueprintID,
	)
	if tokenBlueprintID == "" {
		return PurchasedPair{}, false
	}

	return PurchasedPair{
		OrderID: item.OrderID,

		ItemType:  orderdom.OrderItemTypeList,
		ItemIndex: item.ItemIndex,

		ModelID:     modelID,
		InventoryID: inventoryID,
		ListID:      strings.TrimSpace(item.ListID),

		TokenBlueprintID: tokenBlueprintID,
	}, true
}

func purchasedPairFromResaleItem(
	item orderdom.EligibleTransferItem,
) (PurchasedPair, bool) {
	resaleID := strings.TrimSpace(item.ResaleID)
	productID := strings.TrimSpace(item.ProductID)
	tokenBlueprintID := strings.TrimSpace(
		item.TokenBlueprintID,
	)

	if resaleID == "" ||
		productID == "" ||
		tokenBlueprintID == "" {
		return PurchasedPair{}, false
	}

	return PurchasedPair{
		OrderID: item.OrderID,

		ItemType:  orderdom.OrderItemTypeResale,
		ItemIndex: item.ItemIndex,

		ResaleID: resaleID,

		ProductID:          productID,
		ProductBlueprintID: strings.TrimSpace(item.ProductBlueprintID),
		TokenBlueprintID:   tokenBlueprintID,
		BrandID:            strings.TrimSpace(item.BrandID),
	}, true
}

func productBlueprintPatchForPreview(
	pb pbdom.ProductBlueprint,
) pbdom.Patch {
	return pbdom.Patch{
		ProductName:              stringPtrOrNil(pb.ProductName),
		Description:              stringPtrOrNil(pb.Description),
		BrandID:                  stringPtrOrNil(pb.BrandID),
		CompanyID:                stringPtrOrNil(pb.CompanyID),
		ProductBlueprintCategory: &pb.ProductBlueprintCategory,
		CategoryFields:           &pb.CategoryFields,
		ProductIdTag:             &pb.ProductIdTag,
		AssigneeID:               stringPtrOrNil(pb.AssigneeID),
		ModelRefs:                &pb.ModelRefs,
	}
}

func stringPtrOrNil(value string) *string {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil
	}
	return &v
}

func (q *PreviewQuery) getBrandNameIcon(
	ctx context.Context,
	brandID string,
) (branddom.NameIcon, error) {
	if q == nil || q.BrandRepo == nil {
		return branddom.NameIcon{},
			ErrPreviewQueryNotConfigured
	}

	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return branddom.NameIcon{},
			branddom.ErrInvalidID
	}

	b, err := q.BrandRepo.GetByID(ctx, brandID)
	if err != nil {
		return branddom.NameIcon{}, err
	}

	return branddom.NameIcon{
		Name:      b.Name,
		BrandIcon: b.BrandIcon,
	}, nil
}

func (q *PreviewQuery) resolveBrandNameForPreview(
	ctx context.Context,
	brandID string,
	out *dto.PreviewModelInfo,
) string {
	if q == nil ||
		q.BrandRepo == nil ||
		strings.TrimSpace(brandID) == "" {
		return ""
	}

	ni, err := q.getBrandNameIcon(ctx, brandID)
	if err != nil {
		return ""
	}

	if strings.TrimSpace(ni.Name) == "" {
		return ""
	}

	if out != nil && strings.TrimSpace(out.BrandName) == "" {
		out.BrandName = ni.Name
	}

	return ni.Name
}

func (q *PreviewQuery) fillResolvedModelInfo(
	ctx context.Context,
	out *dto.PreviewModelInfo,
	modelID string,
	categoryKind commondom.ProductCategoryKind,
) error {
	if out == nil {
		return ErrPreviewQueryNotConfigured
	}

	if q == nil || q.NameResolver == nil {
		return ErrPreviewQueryNotConfigured
	}

	resolved := q.NameResolver.ResolveModelResolved(
		ctx,
		modelID,
	)
	if resolved.Kind == "" &&
		resolved.ModelNumber == "" {
		return ErrModelVariationNotFound
	}

	modelKind := resolved.Kind
	if modelKind == "" {
		modelKind = string(categoryKind)
	}

	out.ModelKind = commondom.ProductCategoryKind(
		modelKind,
	)
	out.ModelNumber = resolved.ModelNumber
	out.ModelLabel = buildPreviewModelLabel(
		modelKind,
		resolved.ModelNumber,
		resolved.Size,
		resolved.Color,
		resolved.VolumeValue,
		resolved.VolumeUnit,
	)

	out.Size = resolved.Size
	out.Color = resolved.Color

	if resolved.RGB != nil {
		out.RGB = *resolved.RGB
	}

	out.VolumeValue = resolved.VolumeValue
	out.VolumeUnit = resolved.VolumeUnit
	out.Measurements = cloneMeasurements(
		resolved.Measurements,
	)

	return nil
}

func buildPreviewModelLabel(
	kind string,
	modelNumber string,
	size string,
	color string,
	volumeValue *int,
	volumeUnit string,
) string {
	switch kind {
	case "alcohol":
		if volumeValue != nil && volumeUnit != "" {
			if modelNumber != "" {
				return fmt.Sprintf(
					"%s / %d%s",
					modelNumber,
					*volumeValue,
					volumeUnit,
				)
			}

			return fmt.Sprintf(
				"%d%s",
				*volumeValue,
				volumeUnit,
			)
		}

		return modelNumber

	default:
		if modelNumber != "" &&
			size != "" &&
			color != "" {
			return fmt.Sprintf(
				"%s / %s / %s",
				modelNumber,
				size,
				color,
			)
		}

		if modelNumber != "" && size != "" {
			return fmt.Sprintf(
				"%s / %s",
				modelNumber,
				size,
			)
		}

		if modelNumber != "" && color != "" {
			return fmt.Sprintf(
				"%s / %s",
				modelNumber,
				color,
			)
		}

		if size != "" && color != "" {
			return fmt.Sprintf(
				"%s / %s",
				size,
				color,
			)
		}

		if modelNumber != "" {
			return modelNumber
		}

		if size != "" {
			return size
		}

		return color
	}
}

func (q *PreviewQuery) resolveTransferOwners(
	ctx context.Context,
	transfers []dto.PreviewTransferInfo,
) []dto.PreviewTransferInfo {
	if len(transfers) == 0 {
		return make([]dto.PreviewTransferInfo, 0)
	}

	out := make(
		[]dto.PreviewTransferInfo,
		0,
		len(transfers),
	)

	for _, tr := range transfers {
		fromWalletAddress := strings.TrimSpace(
			tr.FromWalletAddress,
		)
		toWalletAddress := strings.TrimSpace(
			tr.ToWalletAddress,
		)

		// 画面へ返す transfer 履歴には、所有者を識別する ID だけを設定する。
		// 元の transfer が持つ walletAddress、署名、日時、表示名、アイコンなどは
		// dto へコピーしない。
		item := dto.PreviewTransferInfo{}

		if q == nil || q.OwnerResolveQ == nil {
			out = append(out, item)
			continue
		}

		q.resolveTransferFromOwnerID(
			ctx,
			fromWalletAddress,
			&item,
		)

		q.resolveTransferToOwnerID(
			ctx,
			toWalletAddress,
			&item,
		)

		out = append(out, item)
	}

	return out
}

func (q *PreviewQuery) resolveTransferFromOwnerID(
	ctx context.Context,
	walletAddress string,
	item *dto.PreviewTransferInfo,
) {
	if q == nil ||
		q.OwnerResolveQ == nil ||
		item == nil {
		return
	}

	walletAddress = strings.TrimSpace(walletAddress)
	if walletAddress == "" {
		return
	}

	res, err := q.OwnerResolveQ.Resolve(
		ctx,
		walletAddress,
	)
	if err != nil {
		return
	}

	if res == nil {
		return
	}

	switch res.OwnerType {
	case sharedquery.OwnerTypeAvatar:
		item.FromAvatarID = strings.TrimSpace(
			res.AvatarID,
		)
		item.FromBrandID = ""

	case sharedquery.OwnerTypeBrand:
		item.FromBrandID = strings.TrimSpace(
			res.BrandID,
		)
		item.FromAvatarID = ""
	}
}

func (q *PreviewQuery) resolveTransferToOwnerID(
	ctx context.Context,
	walletAddress string,
	item *dto.PreviewTransferInfo,
) {
	if q == nil ||
		q.OwnerResolveQ == nil ||
		item == nil {
		return
	}

	walletAddress = strings.TrimSpace(walletAddress)
	if walletAddress == "" {
		return
	}

	res, err := q.OwnerResolveQ.Resolve(
		ctx,
		walletAddress,
	)
	if err != nil {
		return
	}

	if res == nil {
		return
	}

	switch res.OwnerType {
	case sharedquery.OwnerTypeAvatar:
		item.ToAvatarID = strings.TrimSpace(
			res.AvatarID,
		)
		item.ToBrandID = ""

	case sharedquery.OwnerTypeBrand:
		item.ToBrandID = strings.TrimSpace(
			res.BrandID,
		)
		item.ToAvatarID = ""
	}
}
