// backend\internal\adapters\in\http\mall\handler\user_handler.go
package mallHandler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

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

	// ✅ 末尾スラッシュを吸収
	path := strings.TrimSuffix(r.URL.Path, "/")

	// ✅ /mall プレフィックスを吸収（/mall/users -> /users）
	if strings.HasPrefix(path, "/mall/") {
		path = strings.TrimPrefix(path, "/mall")
	}

	switch {
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

// GET /users/{id}
func (h *UserHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	u, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeUserErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(u)
}

// POST /users
//
// ✅ 方針: docID = Firebase UID を前提にする（= ID 必須）
// - body の id を使って Upsert（存在すれば更新 / 無ければ作成）
// - 新規なら 201 / 既存なら 200
func (h *UserHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	raw, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))

	// snake_case / camelCase 両対応の入力
	type postBody struct {
		ID     string `json:"id"`
		UserID string `json:"userId"`

		FirstName     *string `json:"first_name"`
		FirstNameKana *string `json:"first_name_kana"`
		LastNameKana  *string `json:"last_name_kana"`
		LastName      *string `json:"last_name"`

		FirstNameCC     *string `json:"firstName"`
		FirstNameKanaCC *string `json:"firstNameKana"`
		LastNameKanaCC  *string `json:"lastNameKana"`
		LastNameCC      *string `json:"lastName"`

		CreatedAt *time.Time `json:"created_at"`
		UpdatedAt *time.Time `json:"updated_at"`
		DeletedAt *time.Time `json:"deleted_at"`

		CreatedAtCC *time.Time `json:"createdAt"`
		UpdatedAtCC *time.Time `json:"updatedAt"`
		DeletedAtCC *time.Time `json:"deletedAt"`
	}

	var b postBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// ID（UID）確定
	id := strings.TrimSpace(b.ID)
	if id == "" {
		id = strings.TrimSpace(b.UserID)
	}
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	// フィールド coalesce
	coalesceStrPtr := func(a, b *string) *string {
		if a != nil {
			s := strings.TrimSpace(*a)
			if s == "" {
				return nil
			}
			return &s
		}
		if b != nil {
			s := strings.TrimSpace(*b)
			if s == "" {
				return nil
			}
			return &s
		}
		return nil
	}
	coalesceTimePtr := func(a, b *time.Time) *time.Time {
		if a != nil && !a.IsZero() {
			t := a.UTC()
			return &t
		}
		if b != nil && !b.IsZero() {
			t := b.UTC()
			return &t
		}
		return nil
	}

	in := userdom.CreateUserInput{
		FirstName:     coalesceStrPtr(b.FirstName, b.FirstNameCC),
		FirstNameKana: coalesceStrPtr(b.FirstNameKana, b.FirstNameKanaCC),
		LastNameKana:  coalesceStrPtr(b.LastNameKana, b.LastNameKanaCC),
		LastName:      coalesceStrPtr(b.LastName, b.LastNameCC),
		CreatedAt:     coalesceTimePtr(b.CreatedAt, b.CreatedAtCC),
		UpdatedAt:     coalesceTimePtr(b.UpdatedAt, b.UpdatedAtCC),
		DeletedAt:     coalesceTimePtr(b.DeletedAt, b.DeletedAtCC),
	}

	now := time.Now().UTC()

	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}
	updatedAt := now
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updatedAt = in.UpdatedAt.UTC()
	}

	// ドメイン仕様：deletedAt は NOT NULL 相当（ゼロ禁止）なので、未指定なら createdAt
	deletedAt := createdAt
	if in.DeletedAt != nil && !in.DeletedAt.IsZero() {
		d := in.DeletedAt.UTC()
		if d.After(createdAt) {
			deletedAt = d
		}
	}

	// 既存判定（201/200 のため）
	existed := true
	if _, err := h.uc.GetByID(ctx, id); err != nil {
		if err == userdom.ErrNotFound {
			existed = false
		} else {
			writeUserErr(w, err)
			return
		}
	}

	// ✅ ID（UID）を必ず渡す
	v, err := userdom.New(
		id,
		in.FirstName,
		in.FirstNameKana,
		in.LastNameKana,
		in.LastName,
		createdAt,
		updatedAt,
		deletedAt,
	)
	if err != nil {
		writeUserErr(w, err)
		return
	}

	// ✅ Create ではなく Save（Upsert）に寄せる
	u, err := h.uc.Save(ctx, v)
	if err != nil {
		writeUserErr(w, err)
		return
	}

	if existed {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(u)
}

// PATCH /users/{id}
func (h *UserHandler) patch(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	raw, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))

	var in userdom.UpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	v := userdom.User{
		ID:            id,
		FirstName:     in.FirstName,
		FirstNameKana: in.FirstNameKana,
		LastNameKana:  in.LastNameKana,
		LastName:      in.LastName,
		UpdatedAt:     time.Now().UTC(),
	}
	if in.DeletedAt != nil && !in.DeletedAt.IsZero() {
		v.DeletedAt = in.DeletedAt.UTC()
	}

	u, err := h.uc.Save(ctx, v)
	if err != nil {
		writeUserErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(u)
}

// DELETE /users/{id}
func (h *UserHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.Delete(ctx, id); err != nil {
		writeUserErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// エラーハンドリング
func writeUserErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case userdom.ErrInvalidID:
		code = http.StatusBadRequest
	case userdom.ErrNotFound:
		code = http.StatusNotFound
	case userdom.ErrConflict:
		code = http.StatusConflict
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
