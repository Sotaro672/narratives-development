// backend/internal/adapters/in/http/mall/handler/user_handler.go
package mallHandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"

	usecase "narratives/internal/application/usecase"
	userdom "narratives/internal/domain/user"
)

// UserHandler は /users 関連のエンドポイントを担当します。
type UserHandler struct {
	uc *usecase.UserUsecase
}

// NewUserHandler はHTTPハンドラを初期化します。
func NewUserHandler(uc *usecase.UserUsecase) http.Handler {
	return &UserHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Allow CORS preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 末尾スラッシュを吸収
	path := strings.TrimSuffix(r.URL.Path, "/")

	// /mall プレフィックスを吸収（/mall/me/users -> /me/users）
	if strings.HasPrefix(path, "/mall/") {
		path = strings.TrimPrefix(path, "/mall")
	}

	switch {
	// ============================================================
	// UNIFIED: /mall/me/users (= /me/users)
	// ============================================================

	// GET /mall/me/users
	case r.Method == http.MethodGet && path == "/me/users":
		h.getMe(w, r)
		return

	// POST /mall/me/users
	case r.Method == http.MethodPost && path == "/me/users":
		h.postMe(w, r)
		return

	// PATCH /mall/me/users
	case r.Method == http.MethodPatch && path == "/me/users":
		h.patchMe(w, r)
		return

	// DELETE /mall/me/users
	case r.Method == http.MethodDelete && path == "/me/users":
		h.deleteMe(w, r)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// ============================================================
// Single source of truth: entity.go に合わせる
// - names: snake_case
// - times: camelCase (createdAt/updatedAt)
// ============================================================

type userBody struct {
	ID            string     `json:"id"`
	FirstName     *string    `json:"first_name"`
	FirstNameKana *string    `json:"first_name_kana"`
	LastNameKana  *string    `json:"last_name_kana"`
	LastName      *string    `json:"last_name"`
	CreatedAt     *time.Time `json:"createdAt"`
	UpdatedAt     *time.Time `json:"updatedAt"`
}

func readJSONBody(r *http.Request, dst any) error {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	r.Body = io.NopCloser(bytes.NewReader(raw))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	return dec.Decode(dst)
}

// ============================================================
// UNIFIED: GET /mall/me/users
// - uid を docID として users/{uid} を返す
// - 無ければ “空の user” を Create して 200 で返す
// ============================================================
func (h *UserHandler) getMe(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user_usecase_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	ctx := r.Context()

	if u, err := h.uc.GetByID(ctx, uid); err == nil {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(u)
		return
	} else if !errors.Is(err, userdom.ErrNotFound) {
		writeUserErr(w, err)
		return
	}

	in := userdom.CreateUserInput{
		FirstName:     nil,
		FirstNameKana: nil,
		LastNameKana:  nil,
		LastName:      nil,
		// createdAt/updatedAt は usecase 側で server now を入れる想定
	}

	u, err := h.uc.Create(ctx, uid, in)
	if err != nil {
		// 競合（並行で作られた）なら取り直して返す
		if errors.Is(err, userdom.ErrConflict) {
			got, gerr := h.uc.GetByID(ctx, uid)
			if gerr != nil {
				writeUserErr(w, gerr)
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(got)
			return
		}

		writeUserErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(u)
}

// ============================================================
// UNIFIED: POST /mall/me/users
// - uid を docID として強制
// - body の id は信用しない
// - Create のみ行う
// - 既存 users/{uid} がある場合は ErrConflict -> 409
// - createdAt/updatedAt は usecase が server now を入れる
// ============================================================
func (h *UserHandler) postMe(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user_usecase_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	ctx := r.Context()

	var b userBody
	if err := readJSONBody(r, &b); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	in := userdom.CreateUserInput{
		FirstName:     b.FirstName,
		FirstNameKana: b.FirstNameKana,
		LastNameKana:  b.LastNameKana,
		LastName:      b.LastName,
	}

	u, err := h.uc.Create(ctx, uid, in)
	if err != nil {
		writeUserErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(u)
}

// ============================================================
// UNIFIED: PATCH /mall/me/users
// - uid を docID として強制
// - nil は「未指定」
// - 空文字は「フィールド削除」
// ============================================================
func (h *UserHandler) patchMe(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user_usecase_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	ctx := r.Context()

	var b userBody
	if err := readJSONBody(r, &b); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	in := userdom.UpdateUserInput{
		FirstName:     b.FirstName,
		FirstNameKana: b.FirstNameKana,
		LastNameKana:  b.LastNameKana,
		LastName:      b.LastName,
		// UpdatedAt は usecase が差し込む
	}

	u, err := h.uc.Update(ctx, uid, in)
	if err != nil {
		writeUserErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(u)
}

// ============================================================
// UNIFIED: DELETE /mall/me/users
// ============================================================
func (h *UserHandler) deleteMe(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user_usecase_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.uc.Delete(r.Context(), uid); err != nil {
		writeUserErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// エラーハンドリング
func writeUserErr(w http.ResponseWriter, err error) {
	if err == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unknown"})
		return
	}

	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, userdom.ErrInvalidID),
		errors.Is(err, userdom.ErrInvalidFirstName),
		errors.Is(err, userdom.ErrInvalidFirstNameKana),
		errors.Is(err, userdom.ErrInvalidLastNameKana),
		errors.Is(err, userdom.ErrInvalidLastName),
		errors.Is(err, userdom.ErrInvalidCreatedAt),
		errors.Is(err, userdom.ErrInvalidUpdatedAt):
		code = http.StatusBadRequest

	case errors.Is(err, userdom.ErrNotFound):
		code = http.StatusNotFound

	case errors.Is(err, userdom.ErrConflict):
		code = http.StatusConflict
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
