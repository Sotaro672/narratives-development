// backend/internal/application/usecase/payment_postpaid.go
package usecase

/*
責任と機能:
- Payment が paid/succeeded になった後の「後続処理のオーケストレーション」を担う。
  前提: payment / invoice / order の docId は全て同一（= rootID）である。
  具体的には:
  0) order.Paid=true 更新（best-effort）
  1) invoice.paid=true 更新（best-effort）
  2) inventory reserve 更新（best-effort）
  3) cart clear（best-effort）
- 外部依存（invoiceRepo/cartRepo/inventoryRepo/orderRepo）は PaymentUsecase が保持し、
  このファイルは処理順序と優先順位を管理する。
- ✅ このファイルでは名揺れ吸収を行わず、order domain(entity.go) を正として厳密に扱う。
*/

import (
	"context"
	"errors"
	"log"
	"reflect"
	"strings"

	paymentdom "narratives/internal/domain/payment"
)

func isPaidStatus(st paymentdom.PaymentStatus) bool {
	s := strings.TrimSpace(string(st))
	if s == "" {
		return false
	}
	if strings.EqualFold(s, "paid") {
		return true
	}
	if strings.EqualFold(s, "succeeded") || strings.EqualFold(s, "success") {
		return true
	}
	return false
}

// handlePostPaidBestEffort runs post-paid side effects in best-effort manner.
//
// ✅ 前提: payment / invoice / order の docId は全て同じ。
// したがって rootID = payment.InvoiceID をそのまま使う。
func (u *PaymentUsecase) handlePostPaidBestEffort(ctx context.Context, p *paymentdom.Payment) {
	if u == nil || p == nil {
		return
	}

	rootID := trimSpace(p.InvoiceID)
	if rootID == "" {
		return
	}

	// order を 1 回だけ best-effort で取得（order.Paid更新 / inventory reserve / cartId解決に使う）
	var orderAny any
	if u.orderRepo != nil {
		o, getErr := callOrderGetByIDBestEffort(u.orderRepo, ctx, rootID)
		if getErr != nil {
			log.Printf("[payment_uc] WARN: order fetch failed orderId=%s err=%v", maskID(rootID), getErr)
		} else {
			orderAny = o
		}
	}

	// 0) order.Paid=true（best-effort）
	if u.orderRepo != nil {
		if mkErr := u.markOrderPaidTrueBestEffort(ctx, rootID, orderAny); mkErr != nil {
			log.Printf("[payment_uc] WARN: order mark paid failed orderId=%s err=%v", maskID(rootID), mkErr)
		}
	}

	// 1) invoice.paid=true（best-effort）
	if u.invoiceRepo != nil {
		if mkErr := u.markInvoicePaid(ctx, rootID); mkErr != nil {
			log.Printf("[payment_uc] WARN: invoice mark paid failed invoiceId=%s err=%v", maskID(rootID), mkErr)
		}
	}

	// 2) inventory reserve（best-effort）
	if u.inventoryRepo != nil && orderAny != nil {
		items := extractOrderItemsBestEffort(orderAny)
		if len(items) == 0 {
			log.Printf("[payment_uc] WARN: no order items (skip reserve) orderId=%s", maskID(rootID))
		} else {
			agg := aggregateReserveItems(items)
			for _, it := range agg {
				invID := normalizeInventoryDocIDBestEffort(it.InventoryID)
				if invID == "" || it.ModelID == "" || it.Qty <= 0 {
					continue
				}
				if rErr := u.inventoryRepo.ReserveByOrder(ctx, invID, it.ModelID, rootID, it.Qty); rErr != nil {
					log.Printf(
						"[payment_uc] WARN: inventory reserve failed inventoryId=%s modelId=%s orderId=%s qty=%d err=%v",
						maskID(invID), maskID(it.ModelID), maskID(rootID), it.Qty, rErr,
					)
				} else {
					log.Printf(
						"[payment_uc] inventory reserved inventoryId=%s modelId=%s orderId=%s qty=%d",
						maskID(invID), maskID(it.ModelID), maskID(rootID), it.Qty,
					)
				}
			}
		}
	}

	// 3) cart clear（best-effort）
	if u.cartRepo != nil {
		cartID := u.resolveCartIDBestEffort(ctx, p, rootID, orderAny)
		if strings.TrimSpace(cartID) == "" {
			log.Printf("[payment_uc] WARN: cartId empty (skip clear) rootId=%s", maskID(rootID))
		} else {
			if clrErr := u.cartRepo.Clear(ctx, cartID); clrErr != nil {
				log.Printf("[payment_uc] WARN: cart clear failed cartId=%s rootId=%s err=%v", maskID(cartID), maskID(rootID), clrErr)
			} else {
				log.Printf("[payment_uc] cart cleared cartId=%s rootId=%s", maskID(cartID), maskID(rootID))
			}
		}
	}
}

// ------------------------------------------------------------
// order.Paid = true (best-effort)  ※名揺れ吸収なし
// ------------------------------------------------------------

// markOrderPaidTrueBestEffort updates order.Paid to true.
// It tries, in order:
// 1) orderRepo.Update(ctx, rootID, patch)
// 2) orderRepo.Save(ctx, orderValue, nil)
func (u *PaymentUsecase) markOrderPaidTrueBestEffort(ctx context.Context, rootID string, orderAny any) error {
	if u == nil || u.orderRepo == nil {
		return nil
	}
	id := trimSpace(rootID)
	if id == "" {
		return nil
	}

	// 1) Prefer Update(ctx, id, patch)
	if ok, err := callOrderUpdateSetPaidTrueBestEffort(u.orderRepo, ctx, id); ok {
		return err
	}

	// 2) Fallback: Save(ctx, order, nil)
	o := orderAny
	if o == nil {
		fetched, err := callOrderGetByIDBestEffort(u.orderRepo, ctx, id)
		if err != nil {
			return err
		}
		o = fetched
	}
	if o == nil {
		return errors.New("order_nil")
	}

	updated, changed := setOrderPaidBestEffort(o, true)
	if !changed {
		// Paid フィールドが無い/触れない等の場合は何もしない
		return nil
	}

	if ok, err := callOrderSaveBestEffort(u.orderRepo, ctx, updated); ok {
		return err
	}

	return errors.New("order_repo_missing_Update_or_Save")
}

func setOrderPaidBestEffort(order any, val bool) (any, bool) {
	rv := reflect.ValueOf(order)
	if !rv.IsValid() {
		return order, false
	}

	// pointer -> mutate in place
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return order, false
		}
		ev := rv.Elem()
		if !ev.IsValid() || ev.Kind() != reflect.Struct {
			return order, false
		}
		return order, setOrderPaidOnStructValueBestEffort(ev, val)
	}

	// struct value -> create copy to make it settable
	if rv.Kind() == reflect.Struct {
		cp := reflect.New(rv.Type()).Elem()
		cp.Set(rv)
		changed := setOrderPaidOnStructValueBestEffort(cp, val)
		return cp.Interface(), changed
	}

	return order, false
}

func setOrderPaidOnStructValueBestEffort(ev reflect.Value, val bool) bool {
	if !ev.IsValid() || ev.Kind() != reflect.Struct {
		return false
	}
	f := ev.FieldByName("Paid")
	if !f.IsValid() || !f.CanSet() || f.Kind() != reflect.Bool {
		return false
	}
	if f.Bool() == val {
		return false
	}
	f.SetBool(val)
	return true
}

func callOrderUpdateSetPaidTrueBestEffort(orderRepo any, ctx context.Context, rootID string) (bool, error) {
	rv := reflect.ValueOf(orderRepo)
	if !rv.IsValid() {
		return false, nil
	}

	m := rv.MethodByName("Update")
	if !m.IsValid() {
		if rv.Kind() != reflect.Pointer && rv.CanAddr() {
			m = rv.Addr().MethodByName("Update")
		}
	}
	if !m.IsValid() {
		return false, nil
	}

	// signature: Update(ctx, id, patch) -> (T, error) or (*T, error)
	if m.Type().NumIn() != 3 || m.Type().NumOut() != 2 {
		return true, errors.New("order_repo_Update_invalid_signature")
	}

	patchType := m.Type().In(2)
	patchArg, buildErr := buildOrderUpdatePatchArgPaidOnly(patchType)
	if buildErr != nil {
		return true, buildErr
	}

	outs := m.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(rootID),
		patchArg,
	})

	if len(outs) != 2 {
		return true, errors.New("order_repo_Update_invalid_signature")
	}
	if outs[1].IsNil() {
		return true, nil
	}
	if e, ok := outs[1].Interface().(error); ok {
		return true, e
	}
	return true, errors.New("order_repo_Update_returned_non_error")
}

func buildOrderUpdatePatchArgPaidOnly(t reflect.Type) (reflect.Value, error) {
	// allow pointer
	if t.Kind() == reflect.Pointer {
		if t.Elem().Kind() == reflect.Struct {
			v := reflect.New(t.Elem()).Elem()
			setOrderPatchPaidOnly(v, true)
			p := reflect.New(t.Elem())
			p.Elem().Set(v)
			return p, nil
		}
		return reflect.Value{}, errors.New("order_repo_Update_patch_unsupported_type")
	}

	if t.Kind() == reflect.Struct {
		v := reflect.New(t).Elem()
		setOrderPatchPaidOnly(v, true)
		return v, nil
	}

	return reflect.Value{}, errors.New("order_repo_Update_patch_unsupported_type")
}

func setOrderPatchPaidOnly(v reflect.Value, val bool) {
	f := v.FieldByName("Paid")
	if !f.IsValid() || !f.CanSet() {
		return
	}
	switch f.Kind() {
	case reflect.Bool:
		f.SetBool(val)
	case reflect.Pointer:
		// *bool
		if f.Type().Elem().Kind() == reflect.Bool {
			b := val
			f.Set(reflect.ValueOf(&b))
		}
	}
}

func callOrderSaveBestEffort(orderRepo any, ctx context.Context, order any) (bool, error) {
	rv := reflect.ValueOf(orderRepo)
	if !rv.IsValid() {
		return false, nil
	}

	m := rv.MethodByName("Save")
	if !m.IsValid() {
		if rv.Kind() != reflect.Pointer && rv.CanAddr() {
			m = rv.Addr().MethodByName("Save")
		}
	}
	if !m.IsValid() {
		return false, nil
	}

	// signature: Save(ctx, order, opts) -> (T, error) or (*T, error)
	if m.Type().NumIn() != 3 || m.Type().NumOut() != 2 {
		return true, errors.New("order_repo_Save_invalid_signature")
	}

	orderArg, ok := coerceArgBestEffort(m.Type().In(1), order)
	if !ok {
		return true, errors.New("order_repo_Save_order_arg_not_assignable")
	}

	// opts: nil (typed)
	optsArg := reflect.Zero(m.Type().In(2))

	outs := m.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		orderArg,
		optsArg,
	})

	if len(outs) != 2 {
		return true, errors.New("order_repo_Save_invalid_signature")
	}
	if outs[1].IsNil() {
		return true, nil
	}
	if e, ok := outs[1].Interface().(error); ok {
		return true, e
	}
	return true, errors.New("order_repo_Save_returned_non_error")
}

// coerceArgBestEffort is a tiny helper to pass "order" into repo.Save(...) regardless of pointer/value.
func coerceArgBestEffort(want reflect.Type, v any) (reflect.Value, bool) {
	x := reflect.ValueOf(v)
	if !x.IsValid() {
		return reflect.Value{}, false
	}

	if x.Type().AssignableTo(want) {
		return x, true
	}
	if x.Type().ConvertibleTo(want) {
		return x.Convert(want), true
	}

	// want pointer but got value
	if want.Kind() == reflect.Pointer && x.Kind() != reflect.Pointer {
		if reflect.PointerTo(x.Type()).AssignableTo(want) {
			px := reflect.New(x.Type())
			px.Elem().Set(x)
			return px, true
		}
	}

	// want value but got pointer
	if want.Kind() != reflect.Pointer && x.Kind() == reflect.Pointer && !x.IsNil() {
		if x.Elem().Type().AssignableTo(want) {
			return x.Elem(), true
		}
		if x.Elem().Type().ConvertibleTo(want) {
			return x.Elem().Convert(want), true
		}
	}

	return reflect.Value{}, false
}

// ------------------------------------------------------------
// cartId resolve（名揺れ吸収なし）
// ------------------------------------------------------------

// resolveCartIDBestEffort priority:
// 1) payment.CartID / payment.AvatarID
// 2) order.CartID -> order.AvatarID
// 3) orderRepo.GetByID(rootID) の order.CartID -> order.AvatarID
// 4) ""（skip）
func (u *PaymentUsecase) resolveCartIDBestEffort(ctx context.Context, payment *paymentdom.Payment, rootID string, orderAny any) string {
	// 1) payment
	if payment != nil {
		if s := getStringFieldBestEffort(payment, "CartID"); s != "" {
			return s
		}
		if s := getStringFieldBestEffort(payment, "AvatarID"); s != "" {
			return s
		}
	}

	// 2) already fetched order
	if orderAny != nil {
		if s := getStringFieldBestEffort(orderAny, "CartID"); s != "" {
			return s
		}
		if s := getStringFieldBestEffort(orderAny, "AvatarID"); s != "" {
			return s
		}
	}

	// 3) fetch order by rootID
	if u.orderRepo == nil {
		return ""
	}
	id := trimSpace(rootID)
	if id == "" {
		return ""
	}

	o, err := callOrderGetByIDBestEffort(u.orderRepo, ctx, id)
	if err != nil {
		log.Printf("[payment_uc] WARN: resolve cartId via order failed rootId=%s err=%v", maskID(id), err)
		return ""
	}
	if o == nil {
		return ""
	}

	if s := getStringFieldBestEffort(o, "CartID"); s != "" {
		return s
	}
	if s := getStringFieldBestEffort(o, "AvatarID"); s != "" {
		return s
	}
	return ""
}
