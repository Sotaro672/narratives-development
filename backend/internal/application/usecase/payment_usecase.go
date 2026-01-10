// backend/internal/application/usecase/payment_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"

	common "narratives/internal/domain/common"
	invoicedom "narratives/internal/domain/invoice"
	paymentdom "narratives/internal/domain/payment"
)

// PaymentRepo defines the minimal persistence port needed by PaymentUsecase.
// ✅ aligned with domain contract (payment.RepositoryPort / CreatePaymentInput)
type PaymentRepo interface {
	// Reads
	GetByID(ctx context.Context, id string) (*paymentdom.Payment, error)
	GetByInvoiceID(ctx context.Context, invoiceID string) ([]paymentdom.Payment, error)
	List(ctx context.Context, filter paymentdom.Filter, sort paymentdom.Sort, page paymentdom.Page) (paymentdom.PageResult, error)
	Count(ctx context.Context, filter paymentdom.Filter) (int, error)

	// Writes
	Create(ctx context.Context, in paymentdom.CreatePaymentInput) (*paymentdom.Payment, error)
	Update(ctx context.Context, id string, patch paymentdom.UpdatePaymentInput) (*paymentdom.Payment, error)
	Delete(ctx context.Context, id string) error

	// Dev/Test
	Reset(ctx context.Context) error
}

// ✅ Invoice paid を更新するための最小ポート（PaymentUsecase 側）
type InvoiceRepoForPayment interface {
	GetByOrderID(ctx context.Context, orderID string) (invoicedom.Invoice, error)
	Save(ctx context.Context, inv invoicedom.Invoice, opts *common.SaveOptions) (invoicedom.Invoice, error)
}

// ✅ Cart clear の最小ポート（PaymentUsecase 側）
// carts/{cartId} を空にする（cartId は avatarId と同義の運用）
type CartRepoForPayment interface {
	Clear(ctx context.Context, cartID string) error
}

// ✅ Inventory reserve の最小ポート（PaymentUsecase 側）
// payment paid と同タイミングで reservedByOrder / reservedCount を更新する
type InventoryRepoForPayment interface {
	// ReserveByOrder sets:
	// - stock[modelId].reservedByOrder[orderId] = qty
	// - reservedCount = sum(reservedByOrder) (repo側で正規化)
	ReserveByOrder(ctx context.Context, inventoryID string, modelID string, orderID string, qty int) error
}

// PaymentUsecase orchestrates payment operations.
type PaymentUsecase struct {
	repo          PaymentRepo
	invoiceRepo   InvoiceRepoForPayment
	cartRepo      CartRepoForPayment
	inventoryRepo InventoryRepoForPayment

	// ✅ order から cartId/avatarId/items を拾うため（型に依存しないよう any + reflection）
	orderRepo any

	now func() time.Time
}

func NewPaymentUsecase(repo PaymentRepo) *PaymentUsecase {
	return &PaymentUsecase{
		repo: repo,
		now:  time.Now,
	}
}

// ✅ optional injection: payment 起票後に invoice.paid を更新したい場合に注入する
func (u *PaymentUsecase) WithInvoiceRepoForPayment(repo InvoiceRepoForPayment) *PaymentUsecase {
	u.invoiceRepo = repo
	return u
}

// ✅ optional injection: paid 時に cart を空にしたい場合に注入する
func (u *PaymentUsecase) WithCartRepoForPayment(repo CartRepoForPayment) *PaymentUsecase {
	u.cartRepo = repo
	return u
}

// ✅ optional injection: paid 時に inventory の reservedByOrder / reservedCount を更新したい場合に注入する
func (u *PaymentUsecase) WithInventoryRepoForPayment(repo InventoryRepoForPayment) *PaymentUsecase {
	u.inventoryRepo = repo
	return u
}

// ✅ optional injection: payment に cartId が無い場合、order から拾うために注入する
// 期待するメソッド（どれか1つあればOK）:
// - GetByID(ctx, id) (T, error) or (*T, error)
// - Get(ctx, id) (T, error) or (*T, error)
// - FindByID(ctx, id) (T, error) or (*T, error)
func (u *PaymentUsecase) WithOrderRepoForPayment(repo any) *PaymentUsecase {
	u.orderRepo = repo
	return u
}

// ============================================================
// Queries
// ============================================================

func (u *PaymentUsecase) GetByID(ctx context.Context, id string) (*paymentdom.Payment, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

// docId=invoiceId 前提なら「invoiceID=paymentID」なので GetByID と実質同じ。
// ただし domain port に合わせて残す。
func (u *PaymentUsecase) GetByInvoiceID(ctx context.Context, invoiceID string) ([]paymentdom.Payment, error) {
	return u.repo.GetByInvoiceID(ctx, strings.TrimSpace(invoiceID))
}

func (u *PaymentUsecase) List(ctx context.Context, filter paymentdom.Filter, sort paymentdom.Sort, page paymentdom.Page) (paymentdom.PageResult, error) {
	return u.repo.List(ctx, filter, sort, page)
}

func (u *PaymentUsecase) Count(ctx context.Context, filter paymentdom.Filter) (int, error) {
	return u.repo.Count(ctx, filter)
}

// ============================================================
// Commands
// ============================================================

func (u *PaymentUsecase) Create(ctx context.Context, in paymentdom.CreatePaymentInput) (*paymentdom.Payment, error) {
	in.InvoiceID = strings.TrimSpace(in.InvoiceID)
	in.BillingAddressID = strings.TrimSpace(in.BillingAddressID)

	p, err := u.repo.Create(ctx, in)
	if err != nil {
		return nil, err
	}

	// paid/succeeded のときだけ後続処理
	if p == nil || !isPaidStatus(p.Status) {
		return p, nil
	}

	orderID := strings.TrimSpace(p.InvoiceID) // invoiceId == orderId 前提

	// ✅ order を 1 回だけ best-effort で取得（inventory reserve / cartId 解決に使う）
	var orderAny any
	if u.orderRepo != nil && orderID != "" && (u.inventoryRepo != nil || u.cartRepo != nil) {
		o, getErr := callOrderGetByIDBestEffort(u.orderRepo, ctx, orderID)
		if getErr != nil {
			log.Printf("[payment_uc] WARN: order fetch failed orderId=%s err=%v", maskID(orderID), getErr)
		} else {
			orderAny = o
		}
	}

	// ✅ 1) invoice.paid=true を立てる（best-effort）
	if u.invoiceRepo != nil {
		if mkErr := u.markInvoicePaid(ctx, p.InvoiceID); mkErr != nil {
			log.Printf("[payment_uc] WARN: invoice mark paid failed invoiceId=%s err=%v", maskID(p.InvoiceID), mkErr)
		}
	}

	// ✅ 2) inventory reserve（best-effort）
	// cart を空にするのと同じタイミングで、
	// stock[modelId].reservedByOrder[orderId]=qty を入れ、reservedCount を加算（repoで正規化）
	if u.inventoryRepo != nil && orderAny != nil && orderID != "" {
		items := extractOrderItemsBestEffort(orderAny)
		if len(items) == 0 {
			log.Printf("[payment_uc] WARN: no order items (skip reserve) orderId=%s", maskID(orderID))
		} else {
			agg := aggregateReserveItems(items)
			for _, it := range agg {
				invID := normalizeInventoryDocIDBestEffort(it.InventoryID)
				if invID == "" || it.ModelID == "" || it.Qty <= 0 {
					continue
				}
				if rErr := u.inventoryRepo.ReserveByOrder(ctx, invID, it.ModelID, orderID, it.Qty); rErr != nil {
					log.Printf(
						"[payment_uc] WARN: inventory reserve failed inventoryId=%s modelId=%s orderId=%s qty=%d err=%v",
						maskID(invID), maskID(it.ModelID), maskID(orderID), it.Qty, rErr,
					)
				} else {
					log.Printf(
						"[payment_uc] inventory reserved inventoryId=%s modelId=%s orderId=%s qty=%d",
						maskID(invID), maskID(it.ModelID), maskID(orderID), it.Qty,
					)
				}
			}
		}
	}

	// ✅ 3) cart を空にする（best-effort）
	if u.cartRepo != nil {
		cartID := u.resolveCartIDBestEffort(ctx, p, p.InvoiceID, orderAny)
		if strings.TrimSpace(cartID) == "" {
			log.Printf("[payment_uc] WARN: cartId empty (skip clear) invoiceId=%s", maskID(p.InvoiceID))
		} else {
			if clrErr := u.cartRepo.Clear(ctx, cartID); clrErr != nil {
				log.Printf("[payment_uc] WARN: cart clear failed cartId=%s invoiceId=%s err=%v", maskID(cartID), maskID(p.InvoiceID), clrErr)
			} else {
				log.Printf("[payment_uc] cart cleared cartId=%s invoiceId=%s", maskID(cartID), maskID(p.InvoiceID))
			}
		}
	}

	return p, nil
}

func (u *PaymentUsecase) Update(ctx context.Context, id string, patch paymentdom.UpdatePaymentInput) (*paymentdom.Payment, error) {
	return u.repo.Update(ctx, strings.TrimSpace(id), patch)
}

func (u *PaymentUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}

// Dev/Test helper
func (u *PaymentUsecase) Reset(ctx context.Context) error {
	return u.repo.Reset(ctx)
}

// ============================================================
// Internal helpers
// ============================================================

func isPaidStatus(st paymentdom.PaymentStatus) bool {
	s := strings.TrimSpace(string(st))
	if s == "" {
		return false
	}
	// 現状は mock で "paid" を使っている
	if strings.EqualFold(s, "paid") {
		return true
	}
	// 将来の provider に備えた許容（必要なければ消してOK）
	if strings.EqualFold(s, "succeeded") || strings.EqualFold(s, "success") {
		return true
	}
	return false
}

func (u *PaymentUsecase) markInvoicePaid(ctx context.Context, invoiceID string) error {
	invoiceID = strings.TrimSpace(invoiceID)
	if invoiceID == "" || u.invoiceRepo == nil {
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

// resolveCartIDBestEffort priority:
// 1) payment.CartID / payment.AvatarID が取れればそれ
// 2) orderAny があれば order.CartID -> order.AvatarID の順で使う
// 3) orderRepo から order を取得し、order.CartID -> order.AvatarID の順で使う
// 4) それも無ければ ""（skip）
func (u *PaymentUsecase) resolveCartIDBestEffort(ctx context.Context, payment *paymentdom.Payment, invoiceID string, orderAny any) string {
	// 1) payment から拾う（将来 Payment に cartId を入れた場合に効く）
	if payment != nil {
		if s := getStringFieldBestEffort(payment, "CartID", "CartId", "cartId"); s != "" {
			return s
		}
		// carts/{avatarId} 運用なので avatarId でも良い
		if s := getStringFieldBestEffort(payment, "AvatarID", "AvatarId", "avatarId"); s != "" {
			return s
		}
	}

	// 2) 既に取れている order を使う
	if orderAny != nil {
		if s := getStringFieldBestEffort(orderAny, "CartID", "CartId", "cartId"); s != "" {
			return s
		}
		if s := getStringFieldBestEffort(orderAny, "AvatarID", "AvatarId", "avatarId"); s != "" {
			return s
		}
	}

	// 3) orderRepo から拾う
	if u.orderRepo == nil {
		return ""
	}
	oid := strings.TrimSpace(invoiceID)
	if oid == "" {
		return ""
	}

	o, err := callOrderGetByIDBestEffort(u.orderRepo, ctx, oid)
	if err != nil {
		log.Printf("[payment_uc] WARN: resolve cartId via order failed invoiceId=%s err=%v", maskID(oid), err)
		return ""
	}
	if o == nil {
		return ""
	}

	if s := getStringFieldBestEffort(o, "CartID", "CartId", "cartId"); s != "" {
		return s
	}
	if s := getStringFieldBestEffort(o, "AvatarID", "AvatarId", "avatarId"); s != "" {
		return s
	}
	return ""
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

// ------------------------------------------------------------
// inventory reserve helpers
// ------------------------------------------------------------

type reserveItem struct {
	InventoryID string
	ModelID     string
	Qty         int
}

func extractOrderItemsBestEffort(orderAny any) []reserveItem {
	if orderAny == nil {
		return nil
	}

	// order.Items (slice/array) を探す
	sv, ok := getSliceFieldBestEffort(orderAny, "Items", "items")
	if !ok {
		return nil
	}

	out := make([]reserveItem, 0, sv.Len())
	for i := 0; i < sv.Len(); i++ {
		el := sv.Index(i)
		if !el.IsValid() {
			continue
		}
		if el.Kind() == reflect.Pointer {
			if el.IsNil() {
				continue
			}
			el = el.Elem()
		}

		// struct / map の両対応
		var invID, modelID string
		var qty int

		switch el.Kind() {
		case reflect.Struct:
			invID = getStringFieldFromValueBestEffort(el, "InventoryID", "InventoryId", "inventoryId")
			modelID = getStringFieldFromValueBestEffort(el, "ModelID", "ModelId", "modelId")
			qty = getIntFieldFromValueBestEffort(el, "Qty", "qty", "Quantity", "quantity")
		case reflect.Map:
			// map[string]any
			invID = getStringMapValueBestEffort(el, "inventoryId", "inventoryID", "InventoryID", "InventoryId")
			modelID = getStringMapValueBestEffort(el, "modelId", "modelID", "ModelID", "ModelId")
			qty = getIntMapValueBestEffort(el, "qty", "Qty", "quantity", "Quantity")
		default:
			continue
		}

		invID = strings.TrimSpace(invID)
		modelID = strings.TrimSpace(modelID)
		if qty <= 0 || invID == "" || modelID == "" {
			continue
		}

		out = append(out, reserveItem{InventoryID: invID, ModelID: modelID, Qty: qty})
	}

	return out
}

func aggregateReserveItems(items []reserveItem) []reserveItem {
	if len(items) == 0 {
		return nil
	}

	type key struct {
		Inv string
		Mdl string
	}
	m := map[key]int{}
	for _, it := range items {
		inv := strings.TrimSpace(it.InventoryID)
		mdl := strings.TrimSpace(it.ModelID)
		if inv == "" || mdl == "" || it.Qty <= 0 {
			continue
		}
		m[key{Inv: inv, Mdl: mdl}] += it.Qty
	}

	out := make([]reserveItem, 0, len(m))
	for k, q := range m {
		if q <= 0 {
			continue
		}
		out = append(out, reserveItem{InventoryID: k.Inv, ModelID: k.Mdl, Qty: q})
	}

	// stable order for logs (optional)
	sort.Slice(out, func(i, j int) bool {
		if out[i].InventoryID == out[j].InventoryID {
			return out[i].ModelID < out[j].ModelID
		}
		return out[i].InventoryID < out[j].InventoryID
	})

	return out
}

// cart/order/items で使っている inventoryId が
//
//	product__token__list__model
//
// のように肥大化しているケースがあるため、docId(product__token) に正規化する。
func normalizeInventoryDocIDBestEffort(inventoryID string) string {
	inventoryID = strings.TrimSpace(inventoryID)
	if inventoryID == "" {
		return ""
	}

	parts := strings.Split(inventoryID, "__")
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[0]) + "__" + strings.TrimSpace(parts[1])
	}
	return inventoryID
}

// ------------------------------------------------------------
// reflection helpers (orderRepo is any)
// ------------------------------------------------------------

func callOrderGetByIDBestEffort(orderRepo any, ctx context.Context, orderID string) (any, error) {
	if orderRepo == nil {
		return nil, errors.New("order_repo_not_initialized")
	}

	rv := reflect.ValueOf(orderRepo)
	if !rv.IsValid() {
		return nil, errors.New("order_repo_not_initialized")
	}

	// try methods in order
	methodNames := []string{"GetByID", "Get", "FindByID"}

	var m reflect.Value
	for _, name := range methodNames {
		m = rv.MethodByName(name)
		if m.IsValid() {
			break
		}
		// if value receiver not found, try addressable
		if rv.Kind() != reflect.Pointer && rv.CanAddr() {
			m = rv.Addr().MethodByName(name)
			if m.IsValid() {
				break
			}
		}
	}

	if !m.IsValid() {
		return nil, errors.New("order_repo_missing_method_GetByID_or_equivalent")
	}

	// signature: (context.Context, string) (T, error)
	if m.Type().NumIn() != 2 || m.Type().NumOut() != 2 {
		return nil, errors.New("order_repo_invalid_signature")
	}

	outs := m.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(orderID)})
	if len(outs) != 2 {
		return nil, errors.New("order_repo_invalid_signature")
	}

	var err error
	// outs[1] should be error (interface), may be nil
	if outs[1].IsValid() && outs[1].Kind() == reflect.Interface && !outs[1].IsNil() {
		if e, ok := outs[1].Interface().(error); ok {
			err = e
		} else {
			err = errors.New("order_repo_returned_non_error")
		}
	}

	return outs[0].Interface(), err
}

func getStringFieldBestEffort(v any, fieldNames ...string) string {
	if v == nil {
		return ""
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return ""
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return ""
	}

	for _, name := range fieldNames {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}

		// direct string or named string type
		if f.Kind() == reflect.String {
			s := strings.TrimSpace(f.String())
			if s != "" && s != "<nil>" {
				return s
			}
			continue
		}

		// pointer to string
		if f.Kind() == reflect.Pointer && f.Type().Elem().Kind() == reflect.String && !f.IsNil() {
			s := strings.TrimSpace(f.Elem().String())
			if s != "" && s != "<nil>" {
				return s
			}
			continue
		}

		// fallback: fmt.Sprint for other kinds (rare)
		if f.CanInterface() {
			s := strings.TrimSpace(fmt.Sprint(f.Interface()))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}

	return ""
}

func getSliceFieldBestEffort(v any, fieldNames ...string) (reflect.Value, bool) {
	if v == nil {
		return reflect.Value{}, false
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return reflect.Value{}, false
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return reflect.Value{}, false
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	for _, name := range fieldNames {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.Slice || f.Kind() == reflect.Array {
			return f, true
		}
	}
	return reflect.Value{}, false
}

func getStringFieldFromValueBestEffort(rv reflect.Value, fieldNames ...string) string {
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return ""
	}
	for _, name := range fieldNames {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.String {
			return strings.TrimSpace(f.String())
		}
		if f.Kind() == reflect.Pointer && f.Type().Elem().Kind() == reflect.String && !f.IsNil() {
			return strings.TrimSpace(f.Elem().String())
		}
		if f.CanInterface() {
			s := strings.TrimSpace(fmt.Sprint(f.Interface()))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func getIntFieldFromValueBestEffort(rv reflect.Value, fieldNames ...string) int {
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return 0
	}
	for _, name := range fieldNames {
		f := rv.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		switch f.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int(f.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int(f.Uint())
		case reflect.Float32, reflect.Float64:
			return int(f.Float())
		}
		if f.CanInterface() {
			// very last resort
			if n, ok := parseIntFromAny(f.Interface()); ok {
				return n
			}
		}
	}
	return 0
}

func getStringMapValueBestEffort(mv reflect.Value, keys ...string) string {
	if !mv.IsValid() || mv.Kind() != reflect.Map {
		return ""
	}
	for _, k := range keys {
		kv := reflect.ValueOf(k)
		v := mv.MapIndex(kv)
		if !v.IsValid() {
			continue
		}
		if v.Kind() == reflect.Interface {
			if v.IsNil() {
				continue
			}
			v = v.Elem()
		}
		if v.Kind() == reflect.String {
			s := strings.TrimSpace(v.String())
			if s != "" && s != "<nil>" {
				return s
			}
			continue
		}
		if v.CanInterface() {
			s := strings.TrimSpace(fmt.Sprint(v.Interface()))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func getIntMapValueBestEffort(mv reflect.Value, keys ...string) int {
	if !mv.IsValid() || mv.Kind() != reflect.Map {
		return 0
	}
	for _, k := range keys {
		kv := reflect.ValueOf(k)
		v := mv.MapIndex(kv)
		if !v.IsValid() {
			continue
		}
		if v.Kind() == reflect.Interface {
			if v.IsNil() {
				continue
			}
			v = v.Elem()
		}
		if v.CanInterface() {
			if n, ok := parseIntFromAny(v.Interface()); ok {
				return n
			}
		}
	}
	return 0
}

func parseIntFromAny(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case uint:
		return int(x), true
	case uint32:
		return int(x), true
	case uint64:
		return int(x), true
	case float32:
		return int(x), true
	case float64:
		return int(x), true
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, false
		}
		// allow "1.0" etc
		var n int
		_, err := fmt.Sscanf(s, "%d", &n)
		if err == nil {
			return n, true
		}
		var f float64
		_, err2 := fmt.Sscanf(s, "%f", &f)
		if err2 == nil {
			return int(f), true
		}
		return 0, false
	default:
		return 0, false
	}
}

func maskID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if len(id) <= 8 {
		return "***"
	}
	return id[:4] + "***" + id[len(id)-4:]
}
