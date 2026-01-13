// backend/internal/application/usecase/payment_invoice_paid.go
package usecase

/*
責任と機能:
- invoice.paid=true を更新する処理を担当する（PaymentUsecase の paid 後処理の一部）。
- invoice ドメインのフィールド構造が変わっても壊れにくいように reflection で best-effort 更新する。
*/

import (
	"context"
	"reflect"
	"time"
)

func (u *PaymentUsecase) markInvoicePaid(ctx context.Context, invoiceID string) error {
	invoiceID = trimSpace(invoiceID)
	if invoiceID == "" || u == nil || u.invoiceRepo == nil {
		return nil
	}

	inv, err := u.invoiceRepo.GetByOrderID(ctx, invoiceID)
	if err != nil {
		return err
	}

	now := u.now().UTC()
	changed := setInvoicePaidBestEffort(&inv, now)
	if !changed {
		// 触れなかった（フィールドが無い等）場合も Save はしない
		return nil
	}

	_, err = u.invoiceRepo.Save(ctx, inv, nil)
	return err
}

// setInvoicePaidBestEffort tries to set:
// - inv.Paid = true
// - inv.UpdatedAt = &now (if exists and settable)
// It returns true if it set Paid or UpdatedAt.
func setInvoicePaidBestEffort(inv any, now time.Time) bool {
	if inv == nil {
		return false
	}

	rv := reflect.ValueOf(inv)
	if !rv.IsValid() {
		return false
	}
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return false
	}

	ev := rv.Elem()
	if !ev.IsValid() || ev.Kind() != reflect.Struct {
		return false
	}

	changed := false

	// Paid bool
	if f := ev.FieldByName("Paid"); f.IsValid() && f.CanSet() && f.Kind() == reflect.Bool {
		if f.Bool() == false {
			f.SetBool(true)
			changed = true
		}
	}

	// UpdatedAt *time.Time
	if f := ev.FieldByName("UpdatedAt"); f.IsValid() && f.CanSet() {
		// accept *time.Time only
		if f.Kind() == reflect.Pointer && f.Type().Elem() == reflect.TypeOf(time.Time{}) {
			t := now
			f.Set(reflect.ValueOf(&t))
			changed = true
		}
	}

	return changed
}
