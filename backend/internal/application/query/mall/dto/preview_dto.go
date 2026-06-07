// backend/internal/application/query/mall/dto/preview_dto.go
package dto

import (
	"time"

	sharedquery "narratives/internal/application/query/shared"
	commondom "narratives/internal/domain/common"
	pbdom "narratives/internal/domain/productBlueprint"
	pbcatdom "narratives/internal/domain/productBlueprintCategory"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

type PreviewDTO struct {
	AvatarID string `json:"avatarId"`
	ItemKey  string `json:"itemKey"`

	InventoryID string `json:"inventoryId,omitempty"`
	ListID      string `json:"listId,omitempty"`
	ModelID     string `json:"modelId,omitempty"`
	Qty         int    `json:"qty,omitempty"`

	// list
	Title     string `json:"title,omitempty"`
	ListImage string `json:"listImage,omitempty"`
	Price     *int   `json:"price,omitempty"`

	// ids
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	// product
	ProductName        string `json:"productName,omitempty"`
	ProductBrandID     string `json:"productBrandId,omitempty"`
	ProductCompanyID   string `json:"productCompanyId,omitempty"`
	ProductBrandName   string `json:"productBrandName,omitempty"`
	ProductCompanyName string `json:"productCompanyName,omitempty"`

	// product category
	ProductBlueprintCategoryCode string                        `json:"productBlueprintCategoryCode,omitempty"`
	ProductBlueprintCategoryKind commondom.ProductCategoryKind `json:"productBlueprintCategoryKind,omitempty"`
	ProductBlueprintCategoryName string                        `json:"productBlueprintCategoryName,omitempty"`

	// token flat fields
	//
	// preview 表示で頻繁に使う値は flat field として維持する。
	// 正規の tokenBlueprint 表示情報は TokenBlueprintPatch に入れる。
	TokenName   string `json:"tokenName,omitempty"`
	BrandName   string `json:"brandName,omitempty"`
	CompanyName string `json:"companyName,omitempty"`
	Description string `json:"description,omitempty"`
	IconURL     string `json:"iconUrl,omitempty"`

	// tokenBlueprint patch
	//
	// backend/internal/domain/tokenBlueprint/repository_port.go の Patch を再利用する。
	// preview 独自 Patch は持たない。
	TokenBlueprintPatch *tbdom.Patch `json:"tokenBlueprintPatch,omitempty"`

	// model common
	ModelKind   commondom.ProductCategoryKind `json:"modelKind,omitempty"`
	ModelNumber string                        `json:"modelNumber,omitempty"`
	ModelLabel  string                        `json:"modelLabel,omitempty"`

	// apparel model
	Size  string `json:"size,omitempty"`
	Color string `json:"color,omitempty"`
	RGB   *int   `json:"rgb,omitempty"`

	// alcohol model
	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`
}

// TokenInfo is a minimal view for token doc (tokens/{productId}) used by preview.
type TokenInfo struct {
	ProductID string `json:"productId"`

	BrandID   string `json:"brandId,omitempty"`
	BrandName string `json:"brandName,omitempty"`

	TokenBlueprintID string `json:"tokenBlueprintId,omitempty"`

	ToAddress   string `json:"toAddress,omitempty"`
	MetadataURI string `json:"metadataUri,omitempty"`

	MintAddress        string `json:"mintAddress,omitempty"`
	OnChainTxSignature string `json:"onChainTxSignature,omitempty"`

	MintedAt string `json:"mintedAt,omitempty"`
}

// PreviewTransferInfo is a preview-friendly transfer DTO.
type PreviewTransferInfo struct {
	TransferredAt *time.Time `json:"transferredAt,omitempty"`

	FromWalletAddress string `json:"fromWalletAddress,omitempty"`
	ToWalletAddress   string `json:"toWalletAddress,omitempty"`

	FromAvatarID   string `json:"fromAvatarId,omitempty"`
	FromAvatarName string `json:"fromAvatarName,omitempty"`
	FromAvatarIcon string `json:"fromAvatarIcon,omitempty"`
	FromBrandID    string `json:"fromBrandId,omitempty"`
	FromBrandName  string `json:"fromBrandName,omitempty"`
	FromBrandIcon  string `json:"fromBrandIcon,omitempty"`

	ToAvatarID   string `json:"toAvatarId,omitempty"`
	ToAvatarName string `json:"toAvatarName,omitempty"`
	ToAvatarIcon string `json:"toAvatarIcon,omitempty"`
	ToBrandID    string `json:"toBrandId,omitempty"`
	ToBrandName  string `json:"toBrandName,omitempty"`
	ToBrandIcon  string `json:"toBrandIcon,omitempty"`
}

// PreviewModelInfo is what preview.dart eventually wants to display.
type PreviewModelInfo struct {
	ProductID string `json:"productId"`
	ModelID   string `json:"modelId"`

	// category
	ProductBlueprintCategoryCode string                                  `json:"productBlueprintCategoryCode,omitempty"`
	ProductBlueprintCategoryKind commondom.ProductCategoryKind           `json:"productBlueprintCategoryKind,omitempty"`
	ProductBlueprintCategoryName string                                  `json:"productBlueprintCategoryName,omitempty"`
	ProductBlueprintCategory     *pbdom.ProductBlueprintCategorySnapshot `json:"productBlueprintCategory,omitempty"`
	CategoryInputSchema          *pbcatdom.CategoryInputSchema           `json:"categoryInputSchema,omitempty"`

	// model common
	ModelKind   commondom.ProductCategoryKind `json:"modelKind,omitempty"`
	ModelNumber string                        `json:"modelNumber"`
	ModelLabel  string                        `json:"modelLabel,omitempty"`

	// apparel model
	Size         string         `json:"size,omitempty"`
	Color        string         `json:"color,omitempty"`
	RGB          int            `json:"rgb,omitempty"`
	Measurements map[string]int `json:"measurements,omitempty"`

	// alcohol model
	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`

	BrandName   string `json:"brandName,omitempty"`
	CompanyName string `json:"companyName,omitempty"`

	ProductBlueprintID string                  `json:"productBlueprintId,omitempty"`
	ProductBlueprint   *pbdom.ProductBlueprint `json:"productBlueprint,omitempty"`

	ProductBlueprintPatch *pbdom.Patch `json:"productBlueprintPatch,omitempty"`

	Token *TokenInfo `json:"token,omitempty"`

	// tokenBlueprint patch
	//
	// backend/internal/domain/tokenBlueprint/repository_port.go の Patch を再利用する。
	// preview 独自 Patch は持たない。
	TokenBlueprintPatch *tbdom.Patch `json:"tokenBlueprintPatch,omitempty"`

	Owner *sharedquery.OwnerResolveResult `json:"owner,omitempty"`

	Transfers []PreviewTransferInfo `json:"transfers,omitempty"`
}
