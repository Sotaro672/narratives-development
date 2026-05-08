// backend\internal\application\mint\dto\list.go
package dto

// MintListRowDTO は「一覧画面で必要な最小限」の DTO です。
// frontend の MintListRowDTO と JSON key を合わせます。
type MintListRowDTO struct {
	// inspectionId (= productionId)
	InspectionID string `json:"inspectionId"`

	// 参照（デバッグ/将来用に返しておくと便利）
	MintID         string `json:"mintId"`
	TokenBlueprint string `json:"tokenBlueprintId"`

	// 表示用（名前解決後）
	TokenName     string  `json:"tokenName"`
	CreatedByName string  `json:"createdByName"`
	MintedAt      *string `json:"mintedAt"` // RFC3339 (nil なら未mint)
}
