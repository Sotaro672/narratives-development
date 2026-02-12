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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

// docId = invoiceId 前提のため、Doc(id=invoiceId) のみを参照する。
// 旧データ互換（invoiceId フィールド検索フォールバック）は廃止。
func (r *PaymentRepositoryFS) GetByInvoiceID(ctx context.Context, invoiceID string) ([]paymentdom.Payment, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	invoiceID = strings.TrimSpace(invoiceID)
	if invoiceID == "" {
		return []paymentdom.Payment{}, nil
	}

	p, err := r.GetByID(ctx, invoiceID)
	if err != nil {
		if errors.Is(err, paymentdom.ErrNotFound) {
			return []paymentdom.Payment{}, nil
		}
		return nil, err
	}
	if p == nil {
		return []paymentdom.Payment{}, nil
	}
	return []paymentdom.Payment{*p}, nil
}

func (r *PaymentRepositoryFS) Create(ctx context.Context, in paymentdom.CreatePaymentInput) (*paymentdom.Payment, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	// docId = invoiceId
	invoiceID := strings.TrimSpace(getStringField(in, "InvoiceID"))
	if invoiceID == "" {
		return nil, errors.New("payment: invoiceId is required")
	}

	now := time.Now().UTC()
	docRef := r.col().Doc(invoiceID)

	// invoiceId は docId にするため冗長。旧互換も廃止するのでフィールドとしては保存しない。
	data := map[string]any{
		"billingAddressId": strings.TrimSpace(getStringField(in, "BillingAddressID")),
		"amount":           getIntField(in, "Amount"),
		"status":           strings.TrimSpace(getStringLikeField(in, "Status")),
		"createdAt":        now,
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
	setIfExists(&p, "ID", invoiceID)
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

// ============================================================
// Helpers
// ============================================================

func docToPayment(doc *firestore.DocumentSnapshot) (paymentdom.Payment, error) {
	data := doc.Data()
	if data == nil {
		return paymentdom.Payment{}, fmt.Errorf("empty payment document: %s", doc.Ref.ID)
	}

	getStr := func(key string) string {
		if v, ok := data[key].(string); ok {
			return strings.TrimSpace(v)
		}
		return ""
	}
	getStrPtr := func(key string) *string {
		if v, ok := data[key].(string); ok {
			s := strings.TrimSpace(v)
			if s != "" {
				return &s
			}
		}
		return nil
	}
	getTime := func(key string) time.Time {
		if v, ok := data[key].(time.Time); ok && !v.IsZero() {
			return v.UTC()
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

	// ✅ 互換廃止: docId から ID のみセット（InvoiceID への互換セットはしない）
	setIfExists(&p, "ID", strings.TrimSpace(doc.Ref.ID))

	setIfExists(&p, "BillingAddressID", getStr("billingAddressId"))
	setIfExists(&p, "Amount", getInt("amount"))
	setIfExists(&p, "Status", paymentdom.PaymentStatus(getStr("status")))
	setIfExists(&p, "ErrorType", getStrPtr("errorType"))

	// timestamps
	setIfExists(&p, "CreatedAt", getTime("createdAt"))
	setIfExists(&p, "UpdatedAt", getTime("updatedAt"))
	setIfExists(&p, "DeletedAt", timePtrOrNil(getTime("deletedAt")))

	return p, nil
}

func timePtrOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	utc := t.UTC()
	return &utc
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
	if !v.IsValid() {
		return
	}

	if v.Type().AssignableTo(fv.Type()) {
		fv.Set(v)
		return
	}
	if v.Type().ConvertibleTo(fv.Type()) {
		fv.Set(v.Convert(fv.Type()))
		return
	}

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
	return nil, false
}
