// backend/internal/adapters/in/http/sns/handler/order_handler.go
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	usecase "narratives/internal/application/usecase"
	orderdom "narratives/internal/domain/order"
)

// OrderHandler は /sns/orders 関連のエンドポイントを担当します（GET/POST）。
type OrderHandler struct {
	uc *usecase.OrderUsecase
}

// NewOrderHandler はHTTPハンドラを初期化します。
func NewOrderHandler(uc *usecase.OrderUsecase) http.Handler {
	return &OrderHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *OrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// ルータ実装差分吸収:
	// - 直マウント: /sns/orders, /sns/orders/{id}
	// - prefix strip される構成: /orders, /orders/{id}
	path := r.URL.Path

	// normalize: accept both /sns/orders and /orders
	switch {
	case strings.HasPrefix(path, "/sns/orders"):
		path = strings.TrimPrefix(path, "/sns")
	case strings.HasPrefix(path, "/orders"):
		// ok
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	// now path is like /orders or /orders/{id}
	switch {
	case r.Method == http.MethodPost && (path == "/orders" || path == "/orders/"):
		h.post(w, r)
		return

	case r.Method == http.MethodGet && strings.HasPrefix(path, "/orders/"):
		id := strings.TrimPrefix(path, "/orders/")
		h.get(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// --- POST request shape (minimal) ---

type createOrderRequest struct {
	AvatarID string `json:"avatarId"`
}

// POST /sns/orders
//
// ✅ NOTE:
// OrderUsecase のシグネチャが未提示のため、reflect で一般的な候補にフォールバックします。
// - Create(ctx, req)
// - Create(ctx, avatarId)
// - CreateFromCart(ctx, avatarId)
// - Place(ctx, req) / Place(ctx, avatarId)
// 上記が1つも無い場合は 501 を返します。
func (h *OrderHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	aid := strings.TrimSpace(req.AvatarID)
	if aid == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatarId is required"})
		return
	}

	// --- Try calling usecase methods (reflection-based, compile-safe) ---
	ucv := reflect.ValueOf(h.uc)

	// candidates: func(context.Context, <arg>) (<any>, error)
	candidates := []struct {
		name string
		arg  any
	}{
		{"Create", req},
		{"Create", aid},
		{"CreateFromCart", aid},
		{"Place", req},
		{"Place", aid},
	}

	for _, c := range candidates {
		m := ucv.MethodByName(c.name)
		if !m.IsValid() {
			continue
		}

		out, ok := callUsecase2Ret(ctx, m, c.arg)
		if !ok {
			continue
		}

		// out: (result, err)
		if err, _ := out[1].Interface().(error); err != nil {
			writeOrderErr(w, err)
			return
		}

		// result can be any JSON-marshalable value
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(out[0].Interface())
		return
	}

	// no compatible method found
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "order_create_not_implemented_in_usecase",
	})
}

// GET /sns/orders/{id}
//
// ✅ NOTE:
// OrderUsecase のシグネチャが未提示のため、reflect で一般的な候補にフォールバックします。
// - GetByID(ctx, id)
// - Get(ctx, id)
// 上記が1つも無い場合は 501 を返します。
func (h *OrderHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	ucv := reflect.ValueOf(h.uc)

	// candidates: func(context.Context, string) (<any>, error)
	for _, name := range []string{"GetByID", "Get"} {
		m := ucv.MethodByName(name)
		if !m.IsValid() {
			continue
		}

		out, ok := callUsecase2Ret(ctx, m, id)
		if !ok {
			continue
		}

		if err, _ := out[1].Interface().(error); err != nil {
			writeOrderErr(w, err)
			return
		}

		_ = json.NewEncoder(w).Encode(out[0].Interface())
		return
	}

	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "order_get_not_implemented_in_usecase",
	})
}

// ------------------------------------------------------------
// helpers
// ------------------------------------------------------------

// callUsecase2Ret tries calling method(ctx, arg) and expects exactly 2 return values: (any, error).
// It returns (out, true) if it could call with compatible args and shape.
// It returns (nil, false) if signature mismatch.
func callUsecase2Ret(ctx any, m reflect.Value, arg any) ([]reflect.Value, bool) {
	mt := m.Type()
	if mt.NumIn() != 2 || mt.NumOut() != 2 {
		return nil, false
	}

	// in[0] should be assignable from ctx
	ctxv := reflect.ValueOf(ctx)
	if !ctxv.Type().AssignableTo(mt.In(0)) {
		// allow interface{}-like
		if !ctxv.Type().ConvertibleTo(mt.In(0)) {
			return nil, false
		}
		ctxv = ctxv.Convert(mt.In(0))
	}

	argv := reflect.ValueOf(arg)
	if !argv.IsValid() {
		return nil, false
	}
	if !argv.Type().AssignableTo(mt.In(1)) {
		if !argv.Type().ConvertibleTo(mt.In(1)) {
			return nil, false
		}
		argv = argv.Convert(mt.In(1))
	}

	// out[1] should be error
	errType := reflect.TypeOf((*error)(nil)).Elem()
	if !mt.Out(1).Implements(errType) {
		return nil, false
	}

	out := m.Call([]reflect.Value{ctxv, argv})
	return out, true
}

// エラーハンドリング
func writeOrderErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	// NotFound の sentinel が未確定なので、InvalidID だけ厳密に扱い、NotFound は文字列で寄せる
	if errors.Is(err, orderdom.ErrInvalidID) {
		code = http.StatusBadRequest
	} else {
		msg := strings.ToLower(strings.TrimSpace(err.Error()))
		if msg == "not_found" || strings.Contains(msg, "not found") || strings.Contains(msg, "not_found") {
			code = http.StatusNotFound
		}
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
