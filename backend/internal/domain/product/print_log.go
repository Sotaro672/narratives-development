// backend/internal/domain/product/print_log.go
package product

import (
	"errors"
	"strings"
	"time"
)

// PrintLog は「印刷した Product の履歴」を保持するエンティティ。
// 1 レコードで 1 回の印刷バッチを表し、productIds にそのとき印刷された Product ID 一覧を持ちます。
type PrintLog struct {
	ID           string    `json:"id"`
	ProductionID string    `json:"productionId"`
	ProductIDs   []string  `json:"productIds"`
	PrintedBy    string    `json:"printedBy"`
	PrintedAt    time.Time `json:"printedAt"`
	// QR ペイロード一覧（例: 各 productId に対応する URL）
	// Firestore には保存せず、レスポンス専用に使う想定。
	QrPayloads []string `json:"qrPayloads,omitempty"`
}

// PrintLog 用エラー
var (
	ErrInvalidPrintLogID           = errors.New("printLog: invalid id")
	ErrInvalidPrintLogProductionID = errors.New("printLog: invalid productionId")
	ErrInvalidPrintLogProductIDs   = errors.New("printLog: invalid productIds")
	ErrInvalidPrintLogPrintedBy    = errors.New("printLog: invalid printedBy")
	ErrInvalidPrintLogPrintedAt    = errors.New("printLog: invalid printedAt")
)

// NewPrintLog は PrintLog エンティティのコンストラクタです。
// 空白除去などを行ったうえでバリデーションします。
// QrPayloads はここでは扱わず、後続の処理（usecase など）で必要に応じて詰める想定です。
func NewPrintLog(
	id string,
	productionID string,
	productIDs []string,
	printedBy string,
	printedAt time.Time,
) (PrintLog, error) {
	pl := PrintLog{
		ID:           strings.TrimSpace(id),
		ProductionID: strings.TrimSpace(productionID),
		ProductIDs:   normalizeIDList(productIDs),
		PrintedBy:    strings.TrimSpace(printedBy),
		PrintedAt:    printedAt.UTC(),
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
	if len(pl.ProductIDs) == 0 {
		return ErrInvalidPrintLogProductIDs
	}
	for _, pid := range pl.ProductIDs {
		if strings.TrimSpace(pid) == "" {
			return ErrInvalidPrintLogProductIDs
		}
	}
	if strings.TrimSpace(pl.PrintedBy) == "" {
		return ErrInvalidPrintLogPrintedBy
	}
	if pl.PrintedAt.IsZero() {
		return ErrInvalidPrintLogPrintedAt
	}
	// QrPayloads は任意なのでここではバリデーションしない
	return nil
}
