package handler

import (
	"encoding/json"
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

	switch {
	// GET /users/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/users/"):
		id := strings.TrimPrefix(r.URL.Path, "/users/")
		h.get(w, r, id)

	// POST /users
	case r.Method == http.MethodPost && r.URL.Path == "/users":
		h.post(w, r)

	// PATCH /users/{id}
	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/users/"):
		id := strings.TrimPrefix(r.URL.Path, "/users/")
		h.patch(w, r, id)

	// DELETE /users/{id}
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/users/"):
		id := strings.TrimPrefix(r.URL.Path, "/users/")
		h.delete(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
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
func (h *UserHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var in userdom.CreateUserInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
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
	// ドメイン仕様：deletedAt は NOT NULL 相当（ゼロ禁止）なので、未指定なら createdAt を入れる
	deletedAt := createdAt
	if in.DeletedAt != nil && !in.DeletedAt.IsZero() {
		d := in.DeletedAt.UTC()
		if d.After(createdAt) {
			deletedAt = d
		}
	}

	// ✅ Usecase は userdom.User を受け取る契約なので、ここで詰める
	v, err := userdom.New(
		"", // Firestore 側で採番される前提なら空でOK（New が空IDを許容しないなら方針変更が必要）
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

	u, err := h.uc.Create(ctx, v)
	if err != nil {
		writeUserErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
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

	var in userdom.UpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// Save は userdom.User を受け取るので、ID を必ず入れて、更新したいフィールドだけ入れる
	// ※ domain.New を使うと「deletedAt 0値禁止」などで詰むので、ここは素直に構造体で組み立てる
	// repo.Save 側が UpdateUserInput へ変換して PATCH する設計なら OK
	v := userdom.User{
		ID:            id,
		FirstName:     in.FirstName,
		FirstNameKana: in.FirstNameKana,
		LastNameKana:  in.LastNameKana,
		LastName:      in.LastName,
		UpdatedAt:     time.Now().UTC(), // Save 側で使われる前提（不要なら repo 側で上書きでもOK）
		// DeletedAt は UpdateUserInput にあれば入れる。なければ zero のまま（repo.Save 側で nil 扱いにする）
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
