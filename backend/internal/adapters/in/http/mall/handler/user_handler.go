// backend/internal/adapters/in/http/mall/handler/user_handler.go
package mallHandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

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
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not_found",
		})
		return
	}
}

// ============================================================
// Request body
// - id は認証UIDを使用するため受け取らない
// - createdAt/updatedAt はUsecase側で設定するため受け取らない
// ============================================================

type userBody struct {
	FirstName     *string `json:"first_name"`
	FirstNameKana *string `json:"first_name_kana"`
	LastNameKana  *string `json:"last_name_kana"`
	LastName      *string `json:"last_name"`
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
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "user_usecase_not_initialized",
		})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	ctx := r.Context()

	if u, err := h.uc.GetByID(ctx, uid); err == nil {
		writeJSON(w, http.StatusOK, u)
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

			writeJSON(w, http.StatusOK, got)
			return
		}

		writeUserErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, u)
}

// ============================================================
// UNIFIED: POST /mall/me/users
// - uid を docID として強制
// - request body から id、createdAt、updatedAt は受け取らない
// - Create のみ行う
// - 既存 users/{uid} がある場合は ErrConflict -> 409
// - createdAt/updatedAt は usecase が server now を入れる
// ============================================================

func (h *UserHandler) postMe(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "user_usecase_not_initialized",
		})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	ctx := r.Context()

	var b userBody
	if err := readJSONBody(r, &b); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid json",
		})
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

	writeJSON(w, http.StatusCreated, u)
}

// ============================================================
// UNIFIED: PATCH /mall/me/users
// - uid を docID として強制
// - nil は「未指定」
// - 空文字は「フィールド削除」
// - updatedAt は usecase が server now を入れる
// ============================================================

func (h *UserHandler) patchMe(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "user_usecase_not_initialized",
		})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	ctx := r.Context()

	var b userBody
	if err := readJSONBody(r, &b); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid json",
		})
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

	writeJSON(w, http.StatusOK, u)
}

// ============================================================
// UNIFIED: DELETE /mall/me/users
// ============================================================

func (h *UserHandler) deleteMe(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "user_usecase_not_initialized",
		})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	if err := h.uc.Delete(r.Context(), uid); err != nil {
		writeUserErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// エラーハンドリング
func writeUserErr(w http.ResponseWriter, err error) {
	if err == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "unknown",
		})
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

	writeJSON(w, code, map[string]string{
		"error": err.Error(),
	})
}
