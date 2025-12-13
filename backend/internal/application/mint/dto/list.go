// backend/internal/application/mint/dto/list.go
package dto

// MintListRowDTO は MintRequestManagement（一覧）向けの最小 DTO です。
// - tokenBlueprintId はバックエンド側で resolver を通して tokenName に変換して返す前提
// - 一覧では createdByName / mintedAt(yyyy/mm/dd) のみを返す
// - その他の詳細フィールドは detail.go が担う
type MintListRowDTO struct {
	TokenName     string  `json:"tokenName"`               // resolver 済み
	CreatedByName *string `json:"createdByName,omitempty"` // 無ければ "-" 表示想定（フロント側で fallback）
	MintedAt      *string `json:"mintedAt,omitempty"`      // minted のときだけ "yyyy/mm/dd" を入れる想定
}
