// backend/internal/application/production/dto/command.go
package dto

// ModelQuantityCommand は「どのモデルを何枚生産するか」を表すコマンド DTO です。
// ProductionCreate 画面の ProductionQuantityCard で編集している行に対応します。
type ModelQuantityCommand struct {
	ModelID  string `json:"modelId"`  // モデル（ModelVariation）の ID
	Quantity int    `json:"quantity"` // 生産数量
}

// CreateProductionCommand は「生産計画の作成」用の入力 DTO です。
// frontend の ProductionCreate から送られてくる情報に相当します。
type CreateProductionCommand struct {
	// 商品設計の ID（画面では productRows の id / selectedProductId）
	ProductBlueprintID string `json:"productBlueprintId"`

	// 担当者の memberId（AdminCard で選択した assignee）
	AssigneeID string `json:"assigneeId"`

	// モデル別の生産数量一覧（ProductionQuantityCard で編集した rows）
	Models []ModelQuantityCommand `json:"models"`

	// 初期ステータス（省略可。空なら usecase 側で "manufacturing" などにデフォルト）
	Status string `json:"status,omitempty"`
}

// UpdateProductionCommand は将来的な「生産計画の更新」用 DTO の例です。
// すでに編集モード付きの ProductionDetail があるので、こちらも用意しておくと便利です。
type UpdateProductionCommand struct {
	// 更新対象の Production ID
	ID string `json:"id"`

	// 担当者の memberId
	AssigneeID string `json:"assigneeId"`

	// モデル別の生産数量一覧
	Models []ModelQuantityCommand `json:"models"`

	// ステータス更新（任意）
	Status string `json:"status,omitempty"`
}
