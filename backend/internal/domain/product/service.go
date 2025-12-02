// backend/internal/domain/product/service.go
package product

import (
	"fmt"
	"strings"
	"time"
)

// ======================================
// Product 用 QR コードサービス
// ======================================
//
// ・productId をそのまま QR にするのではなく、必要に応じて
//   ベース URL と組み合わせた「QR に埋め込む文字列」を生成する。
//
// ・実際の QR 画像生成（PNG など）はインフラ層 / フロントエンド側で
//   この文字列を渡して行う想定。
// ======================================

// QRService は Product 用の QR ペイロードを生成するドメインサービスです。
type QRService struct {
	// BaseURL は QR コードに埋め込む際のベース URL。
	// 例: https://narratives.jp/p
	//
	// 空文字の場合は、単純に productId 自体を返します。
	BaseURL string
}

// NewQRService は QRService を初期化します。
func NewQRService(baseURL string) *QRService {
	return &QRService{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
	}
}

// BuildProductQRValue は、指定された productId から
// 「QR コードに埋め込む文字列」を生成します。
//
// BaseURL が設定されていれば
//
//	{BaseURL}/{productId}
//
// という形式の URL を返し、
// BaseURL が空なら productId 自体を返します。
func (s *QRService) BuildProductQRValue(productID string) (string, error) {
	id := strings.TrimSpace(productID)
	if id == "" {
		return "", ErrInvalidID
	}

	// BaseURL 未設定なら productId のみを QR ペイロードとして扱う
	if s.BaseURL == "" {
		return id, nil
	}

	return fmt.Sprintf("%s/%s", s.BaseURL, id), nil
}

// --------------------------------------
// 補助関数（スタティックに使いたい場合用）
// --------------------------------------

// BuildProductQRValue は、サービスを生成せずに直接呼び出したい場合のヘルパーです。
func BuildProductQRValue(baseURL, productID string) (string, error) {
	svc := NewQRService(baseURL)
	return svc.BuildProductQRValue(productID)
}

// ======================================
// InspectionBatch 用サービス
// ======================================

// Complete は、まだ検品されていない明細（InspectionResult が nil または notYet）を
// notManufactured に更新し、バッチ全体の Status を completed にします。
// その際、inspectedBy / inspectedAt には引数の値をセットします。
func (b *InspectionBatch) Complete(by string, at time.Time) error {
	by = strings.TrimSpace(by)
	if by == "" {
		return ErrInvalidInspectedBy
	}
	if at.IsZero() {
		return ErrInvalidInspectedAt
	}
	at = at.UTC()

	// バッチ全体を「検品完了」ステータスに
	b.Status = InspectionStatusCompleted

	for i := range b.Inspections {
		ins := &b.Inspections[i]

		// 対象: 未検品（nil or notYet）
		if ins.InspectionResult == nil || *ins.InspectionResult == InspectionNotYet {
			r := InspectionNotManufactured
			ins.InspectionResult = &r

			// 一括完了した担当者 / 時刻を入れておく
			ins.InspectedBy = &by
			t := at
			ins.InspectedAt = &t
		}
	}

	// ドメインルールに従っているか最終チェック
	return b.Validate()
}
