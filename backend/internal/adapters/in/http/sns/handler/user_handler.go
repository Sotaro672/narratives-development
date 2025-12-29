// backend/internal/adapters/in/http/sns/handler/user_handler.go
package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
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
	trace := r.Header.Get("X-Cloud-Trace-Context")
	log.Printf("[sns_user_handler] enter method=%s path=%s trace=%q contentType=%q contentLen=%d",
		r.Method, r.URL.Path, trace, r.Header.Get("Content-Type"), r.ContentLength)

	w.Header().Set("Content-Type", "application/json")

	// ✅ 末尾スラッシュを吸収
	path := strings.TrimSuffix(r.URL.Path, "/")

	// ✅ /sns プレフィックスを吸収（/sns/users -> /users）
	if strings.HasPrefix(path, "/sns/") {
		path = strings.TrimPrefix(path, "/sns")
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
		log.Printf("[sns_user_handler] not_found method=%s path=%s (raw=%s)", r.Method, path, r.URL.Path)
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
		log.Printf("[sns_user_handler] get bad_request empty_id")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[sns_user_handler] get start id=%q", id)
	u, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[sns_user_handler] get failed id=%q err=%v", id, err)
		writeUserErr(w, err)
		return
	}
	log.Printf("[sns_user_handler] get ok id=%q", id)
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
		log.Printf("[sns_user_handler] post read body failed err=%v", readErr)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	log.Printf("[sns_user_handler] post raw body len=%d head=%q", len(raw), headString(raw, 300))

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
		log.Printf("[sns_user_handler] post decode failed err=%v bodyHead=%q", err, headString(raw, 300))
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
		log.Printf("[sns_user_handler] post bad_request empty_id")
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

	log.Printf("[sns_user_handler] post parsed id=%q firstName=%v lastName=%v createdAt=%s updatedAt=%s deletedAt=%s",
		id, sPtr(in.FirstName), sPtr(in.LastName),
		createdAt.Format(time.RFC3339Nano), updatedAt.Format(time.RFC3339Nano), deletedAt.Format(time.RFC3339Nano))

	// 既存判定（201/200 のため）
	existed := true
	if _, err := h.uc.GetByID(ctx, id); err != nil {
		if err == userdom.ErrNotFound {
			existed = false
		} else {
			// Firestore/ctx 系のエラーの可能性もあるのでここで落とす
			log.Printf("[sns_user_handler] post precheck GetByID failed id=%q err=%v", id, err)
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
		log.Printf("[sns_user_handler] post domain.New failed id=%q err=%v", id, err)
		writeUserErr(w, err)
		return
	}

	// ✅ Create ではなく Save（Upsert）に寄せる
	log.Printf("[sns_user_handler] post usecase.Save start id=%q existed=%v", id, existed)
	u, err := h.uc.Save(ctx, v)
	if err != nil {
		log.Printf("[sns_user_handler] post usecase.Save failed id=%q err=%v", id, err)
		writeUserErr(w, err)
		return
	}

	if existed {
		log.Printf("[sns_user_handler] post ok (updated) id=%q", u.ID)
		w.WriteHeader(http.StatusOK)
	} else {
		log.Printf("[sns_user_handler] post ok (created) id=%q", u.ID)
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(u)
}

// PATCH /users/{id}
func (h *UserHandler) patch(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		log.Printf("[sns_user_handler] patch bad_request empty_id")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	raw, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		log.Printf("[sns_user_handler] patch read body failed id=%q err=%v", id, readErr)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))

	log.Printf("[sns_user_handler] patch raw body id=%q len=%d head=%q", id, len(raw), headString(raw, 300))

	var in userdom.UpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		log.Printf("[sns_user_handler] patch decode failed id=%q err=%v bodyHead=%q", id, err, headString(raw, 300))
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

	log.Printf("[sns_user_handler] patch usecase.Save start id=%q firstName=%v lastName=%v",
		id, sPtr(in.FirstName), sPtr(in.LastName))

	u, err := h.uc.Save(ctx, v)
	if err != nil {
		log.Printf("[sns_user_handler] patch usecase.Save failed id=%q err=%v", id, err)
		writeUserErr(w, err)
		return
	}

	log.Printf("[sns_user_handler] patch ok id=%q", id)
	_ = json.NewEncoder(w).Encode(u)
}

// DELETE /users/{id}
func (h *UserHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		log.Printf("[sns_user_handler] delete bad_request empty_id")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[sns_user_handler] delete start id=%q", id)
	if err := h.uc.Delete(ctx, id); err != nil {
		log.Printf("[sns_user_handler] delete failed id=%q err=%v", id, err)
		writeUserErr(w, err)
		return
	}

	log.Printf("[sns_user_handler] delete ok id=%q", id)
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

// helpers
func headString(b []byte, max int) string {
	if len(b) == 0 {
		return ""
	}
	if len(b) > max {
		b = b[:max]
	}
	s := string(b)
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

func sPtr(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return strings.TrimSpace(*p)
}
