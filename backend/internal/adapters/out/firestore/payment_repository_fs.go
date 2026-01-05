// backend/internal/adapters/out/firestore/payment_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	paymentdom "narratives/internal/domain/payment"
)

// Firestore-based implementation of Payment repository.
type PaymentRepositoryFS struct {
	Client *firestore.Client
}

func NewPaymentRepositoryFS(client *firestore.Client) *PaymentRepositoryFS {
	return &PaymentRepositoryFS{Client: client}
}

func (r *PaymentRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("payments")
}

// ============================================================
// usecase.PaymentRepo (aligned with payment.RepositoryPort)
// ============================================================

func (r *PaymentRepositoryFS) GetByID(ctx context.Context, id string) (*paymentdom.Payment, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, paymentdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, paymentdom.ErrNotFound
		}
		return nil, err
	}

	p, err := docToPayment(snap)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// docId = invoiceId 前提のため、まず Doc(id=invoiceId) を優先。
// 互換のため、invoiceId フィールド検索もフォールバックで試す（フィールドが無いなら空になるだけ）。
func (r *PaymentRepositoryFS) GetByInvoiceID(ctx context.Context, invoiceID string) ([]paymentdom.Payment, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	invoiceID = strings.TrimSpace(invoiceID)
	if invoiceID == "" {
		return []paymentdom.Payment{}, nil
	}

	// 1) docId = invoiceId として取得
	if p, err := r.GetByID(ctx, invoiceID); err == nil && p != nil {
		return []paymentdom.Payment{*p}, nil
	}

	// 2) fallback: invoiceId フィールドで検索（古いデータ互換）
	it := r.col().Where("invoiceId", "==", invoiceID).Documents(ctx)
	defer it.Stop()

	out := make([]paymentdom.Payment, 0, 1)
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		p, err := docToPayment(doc)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *PaymentRepositoryFS) List(
	ctx context.Context,
	filter paymentdom.Filter,
	sort paymentdom.Sort,
	page paymentdom.Page,
) (paymentdom.PageResult, error) {
	if r == nil || r.Client == nil {
		return paymentdom.PageResult{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	q := r.col().Query
	q = applyPaymentSort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	all := make([]paymentdom.Payment, 0, 64)
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return paymentdom.PageResult{}, err
		}
		p, err := docToPayment(doc)
		if err != nil {
			return paymentdom.PageResult{}, err
		}
		if matchPaymentFilter(p, filter, doc.Ref.ID) {
			all = append(all, p)
		}
	}

	total := len(all)
	if total == 0 {
		return paymentdom.PageResult{
			Items:      []paymentdom.Payment{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}

	return paymentdom.PageResult{
		Items:      all[offset:end],
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *PaymentRepositoryFS) Count(ctx context.Context, filter paymentdom.Filter) (int, error) {
	if r == nil || r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	total := 0
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		p, err := docToPayment(doc)
		if err != nil {
			return 0, err
		}
		if matchPaymentFilter(p, filter, doc.Ref.ID) {
			total++
		}
	}
	return total, nil
}

func (r *PaymentRepositoryFS) Create(ctx context.Context, in paymentdom.CreatePaymentInput) (*paymentdom.Payment, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	// docId = invoiceId
	invoiceID := strings.TrimSpace(getStringField(in, "InvoiceID"))
	if invoiceID == "" {
		// domain 側エラーが存在するか不明なため、汎用エラーに寄せる
		return nil, errors.New("payment: invoiceId is required")
	}

	now := time.Now().UTC()

	docRef := r.col().Doc(invoiceID)

	// 書き込みドキュメント（invoiceId は docId にするので必須ではないが、互換のため保持してもよい）
	data := map[string]any{
		"billingAddressId": strings.TrimSpace(getStringField(in, "BillingAddressID")),
		"amount":           getIntField(in, "Amount"),
		"status":           strings.TrimSpace(getStringLikeField(in, "Status")),
		"createdAt":        now,
		"invoiceId":        invoiceID, // 互換・検索用（不要なら後で消してOK）
	}
	if et, ok := getPtrStringField(in, "ErrorType"); ok && et != nil && strings.TrimSpace(*et) != "" {
		data["errorType"] = strings.TrimSpace(*et)
	}

	_, err := docRef.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, paymentdom.ErrConflict
		}
		return nil, err
	}

	// 返却 Payment を組み立て（entity.go のフィールド変更に強いよう reflection でセット）
	var p paymentdom.Payment
	setIfExists(&p, "InvoiceID", invoiceID) // もし残っていれば
	setIfExists(&p, "ID", invoiceID)        // もし ID を残していれば
	setIfExists(&p, "BillingAddressID", data["billingAddressId"].(string))
	setIfExists(&p, "Amount", data["amount"].(int))
	setIfExists(&p, "Status", paymentdom.PaymentStatus(data["status"].(string)))
	if v, ok := data["errorType"]; ok {
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s != "" {
			setIfExists(&p, "ErrorType", &s)
		}
	}
	setIfExists(&p, "CreatedAt", now)

	return &p, nil
}

func (r *PaymentRepositoryFS) Update(ctx context.Context, id string, patch paymentdom.UpdatePaymentInput) (*paymentdom.Payment, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, paymentdom.ErrNotFound
	}

	docRef := r.col().Doc(id)

	updates := make([]firestore.Update, 0, 8)

	// string ptr field helper: empty => delete field
	setStr := func(path string, p *string) {
		if p == nil {
			return
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			updates = append(updates, firestore.Update{Path: path, Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: path, Value: v})
		}
	}

	// patch fields are optional; use reflection to be resilient to field removals
	if p, ok := getPtrStringFieldFromAny(patch, "BillingAddressID"); ok {
		setStr("billingAddressId", p)
	}
	if p, ok := getPtrIntFieldFromAny(patch, "Amount"); ok && p != nil {
		updates = append(updates, firestore.Update{Path: "amount", Value: *p})
	}
	if p, ok := getPtrStringLikeFieldFromAny(patch, "Status"); ok && p != nil {
		updates = append(updates, firestore.Update{Path: "status", Value: strings.TrimSpace(*p)})
	}
	if p, ok := getPtrStringFieldFromAny(patch, "ErrorType"); ok {
		setStr("errorType", p)
	}

	if len(updates) == 0 {
		return r.GetByID(ctx, id)
	}

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, paymentdom.ErrNotFound
		}
		if status.Code(err) == codes.AlreadyExists {
			return nil, paymentdom.ErrConflict
		}
		return nil, err
	}

	return r.GetByID(ctx, id)
}

// ✅ この Delete が無い（or 別名）だと、今回の compile error になります
func (r *PaymentRepositoryFS) Delete(ctx context.Context, id string) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return paymentdom.ErrNotFound
	}

	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return paymentdom.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *PaymentRepositoryFS) Reset(ctx context.Context) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var refs []*firestore.DocumentRef
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		refs = append(refs, doc.Ref)
	}

	if len(refs) == 0 {
		return nil
	}

	const chunkSize = 400
	for start := 0; start < len(refs); start += chunkSize {
		end := start + chunkSize
		if end > len(refs) {
			end = len(refs)
		}
		chunk := refs[start:end]

		if err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, ref := range chunk {
				if err := tx.Delete(ref); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

// ============================================================
// Helpers
// ============================================================

func docToPayment(doc *firestore.DocumentSnapshot) (paymentdom.Payment, error) {
	data := doc.Data()
	if data == nil {
		return paymentdom.Payment{}, fmt.Errorf("empty payment document: %s", doc.Ref.ID)
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	getStrPtr := func(keys ...string) *string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				s := strings.TrimSpace(v)
				if s != "" {
					return &s
				}
			}
		}
		return nil
	}
	getTime := func(keys ...string) time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok && !v.IsZero() {
				return v.UTC()
			}
		}
		return time.Time{}
	}
	getInt := func(key string) int {
		if v, ok := data[key]; ok {
			switch n := v.(type) {
			case int:
				return n
			case int64:
				return int(n)
			case float64:
				return int(n)
			}
		}
		return 0
	}

	var p paymentdom.Payment

	// docId から推測できる場合のみ（entity にフィールドが存在すればセットされる）
	setIfExists(&p, "ID", doc.Ref.ID)
	setIfExists(&p, "InvoiceID", doc.Ref.ID) // docId=invoiceId を採用する場合の互換

	setIfExists(&p, "BillingAddressID", getStr("billingAddressId", "billing_address_id"))
	setIfExists(&p, "Amount", getInt("amount"))
	setIfExists(&p, "Status", paymentdom.PaymentStatus(getStr("status")))
	setIfExists(&p, "ErrorType", getStrPtr("errorType", "error_type"))

	// timestamps: CreatedAt だけ残す設計を想定（他が残っていても存在すればセットされる）
	ct := getTime("createdAt", "created_at")
	setIfExists(&p, "CreatedAt", ct)
	setIfExists(&p, "UpdatedAt", getTime("updatedAt", "updated_at"))
	setIfExists(&p, "DeletedAt", timePtrOrNil(getTime("deletedAt", "deleted_at")))

	return p, nil
}

func timePtrOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	utc := t.UTC()
	return &utc
}

// matchPaymentFilter applies Filter in-memory (フィールドの増減に強くするため reflection で読む)
func matchPaymentFilter(p paymentdom.Payment, f paymentdom.Filter, docID string) bool {
	trimEq := func(a, b string) bool { return strings.TrimSpace(a) == strings.TrimSpace(b) }

	// ID (docID) / InvoiceID のどちらで来ても対応
	if v := strings.TrimSpace(getStringField(f, "ID")); v != "" {
		if !trimEq(docID, v) && !trimEq(getStringField(p, "ID"), v) {
			return false
		}
	}
	if v := strings.TrimSpace(getStringField(f, "InvoiceID")); v != "" {
		// docId=invoiceId
		if !trimEq(docID, v) && !trimEq(getStringField(p, "InvoiceID"), v) {
			return false
		}
	}

	if v := strings.TrimSpace(getStringField(f, "BillingAddressID")); v != "" {
		if !trimEq(getStringField(p, "BillingAddressID"), v) {
			return false
		}
	}

	// Statuses []PaymentStatus
	if sts, ok := getSliceStringLikeFieldFromAny(f, "Statuses"); ok && len(sts) > 0 {
		ps := strings.TrimSpace(getStringLikeField(p, "Status"))
		match := false
		for _, s := range sts {
			if strings.TrimSpace(s) == ps {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	if v := strings.TrimSpace(getStringField(f, "ErrorType")); v != "" {
		et, _ := getPtrStringFieldFromAny(p, "ErrorType")
		if et == nil || strings.TrimSpace(*et) != v {
			return false
		}
	}

	if min, ok := getPtrIntFieldFromAny(f, "MinAmount"); ok && min != nil {
		if getIntField(p, "Amount") < *min {
			return false
		}
	}
	if max, ok := getPtrIntFieldFromAny(f, "MaxAmount"); ok && max != nil {
		if getIntField(p, "Amount") > *max {
			return false
		}
	}

	// Created time range
	if from, ok := getPtrTimeFieldFromAny(f, "CreatedFrom"); ok && from != nil {
		ct := getTimeField(p, "CreatedAt")
		if !ct.IsZero() && ct.Before(from.UTC()) {
			return false
		}
	}
	if to, ok := getPtrTimeFieldFromAny(f, "CreatedTo"); ok && to != nil {
		ct := getTimeField(p, "CreatedAt")
		if !ct.IsZero() && !ct.Before(to.UTC()) {
			return false
		}
	}

	return true
}

// applyPaymentSort maps Sort to Firestore orderBy.
func applyPaymentSort(q firestore.Query, sort paymentdom.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
	var field string

	switch col {
	case "createdat", "created_at":
		field = "createdAt"
	case "amount":
		field = "amount"
	case "status":
		field = "status"
	case "updatedat", "updated_at":
		// UpdatedAt が無い設計でも、存在すれば並べ替えできる（無ければ createdAt を使う）
		field = "updatedAt"
	default:
		return q.OrderBy("createdAt", firestore.Desc).
			OrderBy(firestore.DocumentID, firestore.Desc)
	}

	dir := firestore.Asc
	if strings.EqualFold(string(sort.Order), "desc") {
		dir = firestore.Desc
	}

	return q.OrderBy(field, dir).OrderBy(firestore.DocumentID, dir)
}

// ------------------------------------------------------------
// reflection helpers (compile-safe even if entity fields change)
// ------------------------------------------------------------

func setIfExists(dst any, field string, val any) {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return
	}
	ev := rv.Elem()
	if ev.Kind() != reflect.Struct {
		return
	}
	fv := ev.FieldByName(field)
	if !fv.IsValid() || !fv.CanSet() {
		return
	}

	v := reflect.ValueOf(val)
	// nil
	if !v.IsValid() {
		return
	}

	// direct assignable
	if v.Type().AssignableTo(fv.Type()) {
		fv.Set(v)
		return
	}
	// convertible (e.g., PaymentStatus underlying string)
	if v.Type().ConvertibleTo(fv.Type()) {
		fv.Set(v.Convert(fv.Type()))
		return
	}

	// pointer assign (e.g., *string)
	if fv.Kind() == reflect.Ptr && v.Kind() == reflect.Ptr && v.Type().AssignableTo(fv.Type()) {
		fv.Set(v)
		return
	}
}

func getStringField(obj any, field string) string {
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}
	f := rv.FieldByName(field)
	if !f.IsValid() {
		return ""
	}
	if f.Kind() == reflect.String {
		return strings.TrimSpace(f.String())
	}
	return ""
}

func getStringLikeField(obj any, field string) string {
	// string もしくは underlying string type を string として取得
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}
	f := rv.FieldByName(field)
	if !f.IsValid() {
		return ""
	}
	if f.Kind() == reflect.String {
		return strings.TrimSpace(f.String())
	}
	return ""
}

func getIntField(obj any, field string) int {
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return 0
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return 0
	}
	f := rv.FieldByName(field)
	if !f.IsValid() {
		return 0
	}
	switch f.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64:
		return int(f.Int())
	}
	return 0
}

func getTimeField(obj any, field string) time.Time {
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return time.Time{}
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return time.Time{}
	}
	f := rv.FieldByName(field)
	if !f.IsValid() {
		return time.Time{}
	}
	if t, ok := f.Interface().(time.Time); ok {
		return t.UTC()
	}
	return time.Time{}
}

func getPtrStringField(obj any, field string) (*string, bool) {
	return getPtrStringFieldFromAny(obj, field)
}

func getPtrStringFieldFromAny(obj any, field string) (*string, bool) {
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, true
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, false
	}
	f := rv.FieldByName(field)
	if !f.IsValid() {
		return nil, false
	}
	if f.Kind() == reflect.Ptr && f.Type().Elem().Kind() == reflect.String {
		if f.IsNil() {
			return nil, true
		}
		s := strings.TrimSpace(f.Elem().String())
		return &s, true
	}
	return nil, false
}

func getPtrIntFieldFromAny(obj any, field string) (*int, bool) {
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, true
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, false
	}
	f := rv.FieldByName(field)
	if !f.IsValid() {
		return nil, false
	}
	if f.Kind() == reflect.Ptr && (f.Type().Elem().Kind() == reflect.Int || f.Type().Elem().Kind() == reflect.Int64) {
		if f.IsNil() {
			return nil, true
		}
		v := int(f.Elem().Int())
		return &v, true
	}
	return nil, false
}

func getPtrTimeFieldFromAny(obj any, field string) (*time.Time, bool) {
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, true
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, false
	}
	f := rv.FieldByName(field)
	if !f.IsValid() {
		return nil, false
	}
	if f.Kind() == reflect.Ptr && f.Type().Elem() == reflect.TypeOf(time.Time{}) {
		if f.IsNil() {
			return nil, true
		}
		t := f.Elem().Interface().(time.Time).UTC()
		return &t, true
	}
	return nil, false
}

// status ptr may be *PaymentStatus (underlying string) -> treat as *string
func getPtrStringLikeFieldFromAny(obj any, field string) (*string, bool) {
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, true
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, false
	}
	f := rv.FieldByName(field)
	if !f.IsValid() {
		return nil, false
	}
	if f.Kind() == reflect.Ptr && f.Type().Elem().Kind() == reflect.String {
		if f.IsNil() {
			return nil, true
		}
		s := strings.TrimSpace(f.Elem().String())
		return &s, true
	}
	// *definedStringType
	if f.Kind() == reflect.Ptr && f.Type().Elem().Kind() == reflect.String {
		if f.IsNil() {
			return nil, true
		}
		s := strings.TrimSpace(f.Elem().String())
		return &s, true
	}
	// *custom string type (Kind is String)
	if f.Kind() == reflect.Ptr && f.Type().Elem().Kind() == reflect.String {
		if f.IsNil() {
			return nil, true
		}
		s := strings.TrimSpace(f.Elem().String())
		return &s, true
	}
	return nil, false
}

func getSliceStringLikeFieldFromAny(obj any, field string) ([]string, bool) {
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, true
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, false
	}
	f := rv.FieldByName(field)
	if !f.IsValid() {
		return nil, false
	}
	if f.Kind() != reflect.Slice {
		return nil, false
	}
	out := make([]string, 0, f.Len())
	for i := 0; i < f.Len(); i++ {
		e := f.Index(i)
		if e.Kind() == reflect.String {
			out = append(out, strings.TrimSpace(e.String()))
			continue
		}
		if e.Kind() == reflect.Interface && !e.IsNil() {
			if s, ok := e.Interface().(string); ok {
				out = append(out, strings.TrimSpace(s))
			}
		}
	}
	return out, true
}
