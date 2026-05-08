// backend/internal/domain/product/print_log.go
package product

import (
	"errors"
	"sort"
)

// PrintedItem は「印刷した Product」と「並び順(displayOrder)」の組を表します。
// PrintLog.Items は displayOrder 昇順で保持されることを期待します。
type PrintedItem struct {
	ProductID    string `json:"productId"`
	DisplayOrder int    `json:"displayOrder"`
}

// PrintLog は「印刷した Product の履歴」を保持するエンティティ。
// 1 レコードで 1 回の印刷バッチを表し、items にそのとき印刷された Product と displayOrder を持ちます。
// printedAt / printedBy は Production 側で責務を持つためここでは扱いません。
type PrintLog struct {
	ID           string        `json:"id"`
	ProductionID string        `json:"productionId"`
	Items        []PrintedItem `json:"items"`

	// QR ペイロード一覧（例: 各 productId に対応する URL）
	// Firestore には保存せず、レスポンス専用に使う想定。
	// Items の並び（displayOrder 昇順）に合わせて詰めることを期待します。
	QrPayloads []string `json:"qrPayloads,omitempty"`
}

// PrintLog 用エラー
var (
	ErrInvalidPrintLogID           = errors.New("printLog: invalid id")
	ErrInvalidPrintLogProductionID = errors.New("printLog: invalid productionId")
	ErrInvalidPrintLogItems        = errors.New("printLog: invalid items")
	ErrInvalidPrintLogItem         = errors.New("printLog: invalid item")
	ErrInvalidPrintLogDisplayOrder = errors.New("printLog: invalid displayOrder")
)

// NewPrintLog は PrintLog エンティティのコンストラクタです。
// items を正規化したうえで displayOrder 昇順にソートし、バリデーションします。
// QrPayloads はここでは扱わず、後続の処理（usecase など）で必要に応じて詰める想定です。
func NewPrintLog(
	id string,
	productionID string,
	items []PrintedItem,
) (PrintLog, error) {
	pl := PrintLog{
		ID:           id,
		ProductionID: productionID,
		Items:        normalizeAndSortItems(items),
		// QrPayloads は任意フィールドなのでデフォルト nil のまま
	}
	if err := pl.validate(); err != nil {
		return PrintLog{}, err
	}
	return pl, nil
}

func (pl PrintLog) validate() error {
	if pl.ID == "" {
		return ErrInvalidPrintLogID
	}
	if pl.ProductionID == "" {
		return ErrInvalidPrintLogProductionID
	}
	if len(pl.Items) == 0 {
		return ErrInvalidPrintLogItems
	}

	seen := make(map[string]struct{}, len(pl.Items))
	for _, it := range pl.Items {
		if it.ProductID == "" {
			return ErrInvalidPrintLogItem
		}
		if it.DisplayOrder <= 0 {
			return ErrInvalidPrintLogDisplayOrder
		}
		if _, ok := seen[it.ProductID]; ok {
			return ErrInvalidPrintLogItems
		}
		seen[it.ProductID] = struct{}{}
	}

	// QrPayloads は任意なのでここではバリデーションしない
	return nil
}

// normalizeAndSortItems は items を正規化し displayOrder 昇順で返します。
// - ProductID 空は除外
// - ProductID 重複は除外（先勝ち）
// - DisplayOrder <= 0 は除外（validate でも弾くが、ここで落とす）
// - displayOrder 昇順でソート（同値は ProductID 昇順で安定化）
func normalizeAndSortItems(items []PrintedItem) []PrintedItem {
	seen := make(map[string]struct{}, len(items))
	out := make([]PrintedItem, 0, len(items))

	for _, it := range items {
		if it.ProductID == "" {
			continue
		}
		if it.DisplayOrder <= 0 {
			continue
		}
		if _, ok := seen[it.ProductID]; ok {
			continue
		}
		seen[it.ProductID] = struct{}{}
		out = append(out, PrintedItem{
			ProductID:    it.ProductID,
			DisplayOrder: it.DisplayOrder,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].DisplayOrder != out[j].DisplayOrder {
			return out[i].DisplayOrder < out[j].DisplayOrder
		}
		return out[i].ProductID < out[j].ProductID
	})

	return out
}
