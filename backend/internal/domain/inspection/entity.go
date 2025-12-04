// backend/internal/domain/inspection/entity.go
package inspection

import (
	"errors"
	"strings"
	"time"
)

// ===============================
// InspectionResult（検査結果の種類）
// ===============================

type InspectionResult string

const (
	InspectionNotYet          InspectionResult = "notYet"          // 未検査
	InspectionPassed          InspectionResult = "passed"          // 合格
	InspectionFailed          InspectionResult = "failed"          // 不合格
	InspectionNotManufactured InspectionResult = "notManufactured" // 生産されていない（欠品など）
)

// ===============================
// InspectionStatus（バッチ全体の状態）
// ===============================

type InspectionStatus string

const (
	InspectionStatusInspecting InspectionStatus = "inspecting"
	InspectionStatusCompleted  InspectionStatus = "completed"
)

// ------------------------------------------------------
// InspectionItem: productId ごとの検査結果
// ------------------------------------------------------

type InspectionItem struct {
	ProductID        string            `json:"productId"`
	ModelID          string            `json:"modelId"`
	ModelNumber      *string           `json:"modelNumber,omitempty"` // modelId から解決した型番
	InspectionResult *InspectionResult `json:"inspectionResult"`
	InspectedBy      *string           `json:"inspectedBy"`
	InspectedAt      *time.Time        `json:"inspectedAt"`
}

// ------------------------------------------------------
// InspectionBatch: inspections テーブル 1 レコード
// ------------------------------------------------------

type InspectionBatch struct {
	ProductionID string           `json:"productionId"`
	Status       InspectionStatus `json:"status"`

	// 追加フィールド
	Quantity          int              `json:"quantity"`          // item の合計数
	TotalPassed       int              `json:"totalPassed"`       // 合格数
	RequestedBy       *string          `json:"requestedBy"`       // リクエストしたユーザー（作成時 null）
	RequestedAt       *time.Time       `json:"requestedAt"`       // リクエスト日時（作成時 null）
	MintedAt          *time.Time       `json:"mintedAt"`          // NFT ミント完了日時（作成時 null）
	ScheduledBurnDate *time.Time       `json:"scheduledBurnDate"` // バーン予定日時（作成時 null）
	TokenBlueprintID  *string          `json:"tokenBlueprintId"`  // トークン設計ID（作成時 null）
	Inspections       []InspectionItem `json:"inspections"`
}

// ===============================
// Errors（inspection 専用）
// ===============================

var (
	ErrInvalidInspectionProductionID = errors.New("inspection: invalid productionId")
	ErrInvalidInspectionStatus       = errors.New("inspection: invalid status")
	ErrInvalidInspectionProductIDs   = errors.New("inspection: invalid productIds")

	ErrInvalidInspectionResult = errors.New("inspection: invalid inspectionResult")
	ErrInvalidInspectedBy      = errors.New("inspection: invalid inspectedBy")
	ErrInvalidInspectedAt      = errors.New("inspection: invalid inspectedAt")
	ErrNotFound                = errors.New("inspection: not found")
)

// ===============================
// Constructors
// ===============================

// quantity / totalPassed / requestedX / mintedAt / scheduledBurnDate / tokenBlueprintId は
// コンストラクタ内で初期化（tokenBlueprintId / scheduledBurnDate は常に nil）
func NewInspectionBatch(
	productionID string,
	status InspectionStatus,
	productIDs []string,
) (InspectionBatch, error) {

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return InspectionBatch{}, ErrInvalidInspectionProductionID
	}

	if !IsValidInspectionStatus(status) {
		return InspectionBatch{}, ErrInvalidInspectionStatus
	}

	ids := normalizeIDList(productIDs)
	if len(ids) == 0 {
		return InspectionBatch{}, ErrInvalidInspectionProductIDs
	}

	inspections := make([]InspectionItem, 0, len(ids))
	for _, id := range ids {
		r := InspectionNotYet
		inspections = append(inspections, InspectionItem{
			ProductID:        id,
			ModelID:          "",  // modelId はアプリケーション層で埋める
			ModelNumber:      nil, // modelNumber も後から解決
			InspectionResult: &r,
			InspectedBy:      nil,
			InspectedAt:      nil,
		})
	}

	batch := InspectionBatch{
		ProductionID:      pid,
		Status:            status,
		Quantity:          len(inspections),
		TotalPassed:       0,
		RequestedBy:       nil,
		RequestedAt:       nil,
		MintedAt:          nil,
		ScheduledBurnDate: nil,
		TokenBlueprintID:  nil,
		Inspections:       inspections,
	}

	if err := batch.validate(); err != nil {
		return InspectionBatch{}, err
	}

	return batch, nil
}

// ===============================
// Behavior / Validation
// ===============================

func (b InspectionBatch) validate() error {
	if strings.TrimSpace(b.ProductionID) == "" {
		return ErrInvalidInspectionProductionID
	}
	if !IsValidInspectionStatus(b.Status) {
		return ErrInvalidInspectionStatus
	}
	if len(b.Inspections) == 0 {
		return ErrInvalidInspectionProductIDs
	}

	if b.Quantity != len(b.Inspections) || b.Quantity <= 0 {
		return errors.New("inspection: invalid quantity")
	}
	if b.TotalPassed < 0 {
		return errors.New("inspection: invalid totalPassed")
	}

	for _, ins := range b.Inspections {
		if strings.TrimSpace(ins.ProductID) == "" {
			return ErrInvalidInspectionProductIDs
		}

		// InspectionResult が nil の場合は「まだ何も書いていない」扱いにして
		// inspectedBy/inspectedAt が入っていてもエラーにしない。
		if ins.InspectionResult == nil {
			continue
		}

		if !IsValidInspectionResult(*ins.InspectionResult) {
			return ErrInvalidInspectionResult
		}

		switch *ins.InspectionResult {

		// ★ 検査結果が確定している状態は by / at 必須
		case InspectionPassed, InspectionFailed, InspectionNotManufactured:
			if ins.InspectedBy == nil || strings.TrimSpace(*ins.InspectedBy) == "" {
				return ErrInvalidInspectedBy
			}
			if ins.InspectedAt == nil || ins.InspectedAt.IsZero() {
				return ErrInvalidInspectedAt
			}

		// ★ notYet の場合は互換性のため、by/at が入っていてもエラーにしない
		case InspectionNotYet:
			// 何もしない（coherence はチェックしない）
		}
	}
	return nil
}

// Exported wrapper
func (b InspectionBatch) Validate() error {
	return b.validate()
}

// ------------------------------------------------------
// Complete: 検品完了処理（usecase から利用される）
// ------------------------------------------------------
//
// - 引数 by / at が不正なら ErrInvalidInspectedBy / ErrInvalidInspectedAt を返す
// - InspectionResult が nil or notYet の行は notManufactured にし、by/at を埋める
// - バッチの Status を completed に変更
func (b *InspectionBatch) Complete(by string, at time.Time) error {
	inspector := strings.TrimSpace(by)
	if inspector == "" {
		return ErrInvalidInspectedBy
	}

	// at は UTC に正規化しておく
	atUTC := at.UTC()
	if atUTC.IsZero() {
		return ErrInvalidInspectedAt
	}

	for i := range b.Inspections {
		item := &b.Inspections[i]

		// result が nil or notYet の場合は notManufactured にする
		if item.InspectionResult == nil || *item.InspectionResult == InspectionNotYet {
			r := InspectionNotManufactured
			item.InspectionResult = &r

			// by/at を上書き（未設定 or 既存値に関わらず）
			item.InspectedBy = &inspector
			item.InspectedAt = &atUTC
			continue
		}

		// 既に passed/failed/notManufactured の結果が入っている場合は、
		// ここでは特に変更しない（usecase 側で totalPassed は再集計される）。
	}

	// ステータスを completed に変更
	b.Status = InspectionStatusCompleted

	// 一貫性の最終チェック
	return b.validate()
}

// ===============================
// Status / Result validator
// ===============================

func IsValidInspectionStatus(s InspectionStatus) bool {
	return s == InspectionStatusInspecting || s == InspectionStatusCompleted
}

func IsValidInspectionResult(r InspectionResult) bool {
	switch r {
	case InspectionNotYet, InspectionPassed, InspectionFailed, InspectionNotManufactured:
		return true
	default:
		return false
	}
}

// ===============================
// Helpers
// ===============================

// normalizeIDList は ID の配列をトリムし、空文字を除外し、重複を取り除きます。
func normalizeIDList(raw []string) []string {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))

	for _, id := range raw {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}

	return out
}
