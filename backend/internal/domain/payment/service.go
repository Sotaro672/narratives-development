package payment

import "time"

// ========================================
// PaymentCard 表示用ヘルパー関数
// ========================================

// 支払い方法のラベルを取得（日本語）
func GetPaymentMethodLabel(method PaymentMethod) string {
	switch method {
	case "credit_card":
		return "クレジットカード"
	case "bank_transfer":
		return "銀行振込"
	case "cod":
		return "代金引換"
	case "digital_wallet":
		return "デジタルウォレット"
	default:
		return string(method)
	}
}

// 支払完了日（注文日）をフォーマット（YYYY/MM/DD）
func FormatOrderDate(orderDate time.Time) string {
	return orderDate.Format("2006/01/02")
}

// 移譲完了日をフォーマット（YYYY/MM/DD）。未設定なら nil を返す
func FormatTransferredDate(transferredDate *time.Time) *string {
	if transferredDate == nil {
		return nil
	}
	s := transferredDate.Format("2006/01/02")
	return &s
}

// 移譲完了日が存在するかどうか
func HasTransferredDate(transferredDate *time.Time) bool {
	return transferredDate != nil
}

// カードタイトル
func GetCardTitle() string {
	return "支払い情報"
}

// 支払い方法セクションのラベル
func GetPaymentMethodSectionLabel() string {
	return "支払い方法"
}

// 支払完了日セクションのラベル
func GetOrderDateSectionLabel() string {
	return "支払完了日"
}

// 移譲完了日セクションのラベル
func GetTransferredDateSectionLabel() string {
	return "移譲完了日"
}
