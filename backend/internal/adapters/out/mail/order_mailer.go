// backend/internal/adapters/out/mail/order_mailer.go
package mail

import (
	"context"
	"fmt"
	"strings"

	orderdom "narratives/internal/domain/order"
)

type OrderMailer struct {
	client *ResendClient
}

func NewOrderMailer(client *ResendClient) *OrderMailer {
	return &OrderMailer{client: client}
}

func (m *OrderMailer) SendOrderConfirmation(ctx context.Context, from, to string, ord orderdom.Order) error {
	if m == nil || m.client == nil {
		return fmt.Errorf("order mailer is nil")
	}
	if from == "" {
		return fmt.Errorf("from address is empty")
	}
	if to == "" {
		return fmt.Errorf("to address is empty")
	}
	if ord.ID == "" {
		return fmt.Errorf("order id is empty")
	}

	subject := buildOrderConfirmationMailSubject(ord)
	body := buildOrderConfirmationMailBody(ord)

	return m.client.Send(ctx, from, to, subject, body)
}

func buildOrderConfirmationMailSubject(_ orderdom.Order) string {
	return "【Narratives】ご注文が確定しました"
}

func buildOrderConfirmationMailBody(ord orderdom.Order) string {
	var b strings.Builder

	totalQty := 0
	totalPrice := 0
	for _, it := range ord.Items {
		totalQty += it.Qty
		totalPrice += it.Price * it.Qty
	}

	b.WriteString("ご注文ありがとうございます。\n")
	b.WriteString("ご注文の確定が完了しました。\n\n")

	b.WriteString(fmt.Sprintf("注文ID: %s\n", ord.ID))
	if !ord.CreatedAt.IsZero() {
		b.WriteString(fmt.Sprintf("注文日時: %s\n", ord.CreatedAt.UTC().Format("2006-01-02 15:04:05 MST")))
	}
	b.WriteString(fmt.Sprintf("商品点数: %d\n", totalQty))
	b.WriteString(fmt.Sprintf("合計金額: %d\n", totalPrice))
	b.WriteString("\n")

	b.WriteString("配送先:\n")
	if ord.ShippingSnapshot.ZipCode != "" {
		b.WriteString(fmt.Sprintf("郵便番号: %s\n", ord.ShippingSnapshot.ZipCode))
	}
	if ord.ShippingSnapshot.State != "" {
		b.WriteString(fmt.Sprintf("都道府県/州: %s\n", ord.ShippingSnapshot.State))
	}
	if ord.ShippingSnapshot.City != "" {
		b.WriteString(fmt.Sprintf("市区町村: %s\n", ord.ShippingSnapshot.City))
	}
	if ord.ShippingSnapshot.Street != "" {
		b.WriteString(fmt.Sprintf("住所1: %s\n", ord.ShippingSnapshot.Street))
	}
	if ord.ShippingSnapshot.Street2 != "" {
		b.WriteString(fmt.Sprintf("住所2: %s\n", ord.ShippingSnapshot.Street2))
	}
	if ord.ShippingSnapshot.Country != "" {
		b.WriteString(fmt.Sprintf("国: %s\n", ord.ShippingSnapshot.Country))
	}
	b.WriteString("\n")

	b.WriteString("注文商品:\n")
	for i, it := range ord.Items {
		b.WriteString(fmt.Sprintf(
			"%d. modelId=%s inventoryId=%s listId=%s qty=%d price=%d\n",
			i+1,
			it.ModelID,
			it.InventoryID,
			it.ListID,
			it.Qty,
			it.Price,
		))
	}

	b.WriteString("\n")
	b.WriteString("本メールは自動送信です。\n")

	return b.String()
}
