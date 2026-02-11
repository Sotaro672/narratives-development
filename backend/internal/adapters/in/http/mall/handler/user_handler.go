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

	// ✅ Allow CORS preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// ✅ 末尾スラッシュを吸収
	path := strings.TrimSuffix(r.URL.Path, "/")

	// ✅ /mall プレフィックスを吸収（/mall/users -> /users）
	if strings.HasPrefix(path, "/mall/") {
		path = strings.TrimPrefix(path, "/mall")
	}

	switch {
	// ============================================================
	// ✅ UNIFIED: /mall/me/users (= /me/users)
	// ============================================================

	// GET /mall/me/users
	case r.Method == http.MethodGet && path == "/me/users":
		h.getMe(w, r)
		return

	// POST /mall/me/users (Upsert)
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

	// ============================================================
	// existing: /users (admin/debug etc.)
	// ============================================================

	// GET /users/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/users/"):
		id := strings.TrimPrefix(path, "/users/")
		h.get(w, r, id)
		return

	// POST /users
	case r.Method == http.MethodPost && path == "/users":
		h.post(w, r)
		return

	// PATCH /users/{id}
	case r.Method == http.MethodPatch && strings.HasPrefix(path, "/users/"):
		id := strings.TrimPrefix(path, "/users/")
		h.patch(w, r, id)
		return

	// DELETE /users/{id}
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/users/"):
		id := strings.TrimPrefix(path, "/users/")
		h.delete(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// ============================================================
// ✅ Single source of truth: entity.go に合わせる
// - names: snake_case
// - times: camelCase (createdAt/updatedAt/deletedAt)
// ============================================================

type userBody struct {
	ID            string     `json:"id"`
	FirstName     *string    `json:"first_name"`
	FirstNameKana *string    `json:"first_name_kana"`
	LastNameKana  *string    `json:"last_name_kana"`
	LastName      *string    `json:"last_name"`
	CreatedAt     *time.Time `json:"createdAt"`
	UpdatedAt     *time.Time `json:"updatedAt"`
	DeletedAt     *time.Time `json:"deletedAt"`
}

func normalizeStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
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
// ✅ UNIFIED: GET /mall/me/users
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
	if !ok || strings.TrimSpace(uid) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	uid = strings.TrimSpace(uid)
	ctx := r.Context()

	// exists -> return
	if u, err := h.uc.GetByID(ctx, uid); err == nil {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(u)
		return
	}
	// create empty
	in := userdom.CreateUserInput{
		FirstName:     nil,
		FirstNameKana: nil,
		LastNameKana:  nil,
		LastName:      nil,
		// createdAt/updatedAt は usecase 側で server now を入れる想定
		// deletedAt は未指定(nil) = not deleted
	}

	u, err := h.uc.Create(ctx, uid, in)
	if err != nil {
		// 競合（並行で作られた）なら取り直して返す
		if err == userdom.ErrConflict {
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
// ✅ UNIFIED: POST /mall/me/users (Upsert)
// - uid を docID として強制
// - createdAt/updatedAt はサーバが決める（入力が来ても無視）
// - deletedAt は入力があれば採用（nil=未指定 / zero=not deleted / 非zero=soft delete）
//
// ✅ IMPORTANT:
// - domain に UpsertUserInput は作らない方針なので、CreateUserInput を受けて usecase.Upsert に渡す
// ============================================================
func (h *UserHandler) postMe(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user_usecase_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || strings.TrimSpace(uid) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	uid = strings.TrimSpace(uid)
	ctx := r.Context()

	var b userBody
	if err := readJSONBody(r, &b); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	in := userdom.CreateUserInput{
		FirstName:     normalizeStrPtr(b.FirstName),
		FirstNameKana: normalizeStrPtr(b.FirstNameKana),
		LastNameKana:  normalizeStrPtr(b.LastNameKana),
		LastName:      normalizeStrPtr(b.LastName),
	}

	// deletedAt: nil=未指定、指定がある場合は zero も含めて反映したい
	if b.DeletedAt != nil {
		t := b.DeletedAt.UTC() // zero でも OK（not deleted に戻す）
		in.DeletedAt = &t
	}

	u, err := h.uc.Upsert(ctx, uid, in)
	if err != nil {
		writeUserErr(w, err)
		return
	}

	// Upsert は 200 固定（フロント用途）
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(u)
}

// ============================================================
// ✅ UNIFIED: PATCH /mall/me/users
// - uid を docID として強制
// - nil は「未指定」、空文字は「フィールド削除」
// - deletedAt は nil=未指定 / zero=not deleted / 非zero=soft delete
// ============================================================
func (h *UserHandler) patchMe(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user_usecase_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || strings.TrimSpace(uid) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	uid = strings.TrimSpace(uid)
	ctx := r.Context()

	var b userBody
	if err := readJSONBody(r, &b); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	in := userdom.UpdateUserInput{
		FirstName:     b.FirstName,     // nil/""/value をそのまま渡す（repo が "" を delete 扱い）
		FirstNameKana: b.FirstNameKana, // 同上
		LastNameKana:  b.LastNameKana,
		LastName:      b.LastName,
		// UpdatedAt は usecase が差し込む
	}

	if b.DeletedAt != nil {
		t := b.DeletedAt.UTC()
		in.DeletedAt = &t // zero も含めて反映
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
// ✅ UNIFIED: DELETE /mall/me/users
// ============================================================
func (h *UserHandler) deleteMe(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user_usecase_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || strings.TrimSpace(uid) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	uid = strings.TrimSpace(uid)

	if err := h.uc.Delete(r.Context(), uid); err != nil {
		writeUserErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GET /users/{id}
func (h *UserHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user_usecase_not_initialized"})
		return
	}

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	u, err := h.uc.GetByID(r.Context(), id)
	if err != nil {
		writeUserErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(u)
}

// POST /users
// - id は必須（docID）
func (h *UserHandler) post(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user_usecase_not_initialized"})
		return
	}

	var b userBody
	if err := readJSONBody(r, &b); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	id := strings.TrimSpace(b.ID)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	in := userdom.CreateUserInput{
		FirstName:     normalizeStrPtr(b.FirstName),
		FirstNameKana: normalizeStrPtr(b.FirstNameKana),
		LastNameKana:  normalizeStrPtr(b.LastNameKana),
		LastName:      normalizeStrPtr(b.LastName),
	}
	if b.DeletedAt != nil {
		t := b.DeletedAt.UTC()
		in.DeletedAt = &t
	}

	u, err := h.uc.Create(r.Context(), id, in)
	if err != nil {
		writeUserErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(u)
}

// PATCH /users/{id}
func (h *UserHandler) patch(w http.ResponseWriter, r *http.Request, id string) {
	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user_usecase_not_initialized"})
		return
	}

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

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
	}
	if b.DeletedAt != nil {
		t := b.DeletedAt.UTC()
		in.DeletedAt = &t
	}

	u, err := h.uc.Update(r.Context(), id, in)
	if err != nil {
		writeUserErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(u)
}

// DELETE /users/{id}
func (h *UserHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user_usecase_not_initialized"})
		return
	}

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.Delete(r.Context(), id); err != nil {
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
		errors.Is(err, userdom.ErrInvalidUpdatedAt),
		errors.Is(err, userdom.ErrInvalidDeletedAt):
		code = http.StatusBadRequest

	case errors.Is(err, userdom.ErrNotFound):
		code = http.StatusNotFound

	case errors.Is(err, userdom.ErrConflict):
		code = http.StatusConflict
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
