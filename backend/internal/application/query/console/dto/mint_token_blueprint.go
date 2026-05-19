package dto

type ListTokenBlueprintsForMintInput struct {
	BrandID string
	Page    int
	PerPage int
}

type TokenBlueprintForMintDTO struct {
	ID string `json:"id"`

	// 既存 UI 互換: selector 表示用
	Name string `json:"name"`

	// TokenBlueprintCard 表示用: backend の正フィールド
	TokenName string `json:"tokenName,omitempty"`

	Symbol string `json:"symbol"`

	BrandID   string `json:"brandId,omitempty"`
	BrandName string `json:"brandName,omitempty"`
	CompanyID string `json:"companyId,omitempty"`

	Description string `json:"description,omitempty"`
	Minted      bool   `json:"minted"`

	MetadataURI string `json:"metadataUri,omitempty"`
	IconURL     string `json:"iconUrl,omitempty"`
}
