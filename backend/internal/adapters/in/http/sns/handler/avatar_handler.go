// backend/internal/adapters/in/http/sns/handler/avatar_handler.go
package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	uc "narratives/internal/application/usecase"
	avatardom "narratives/internal/domain/avatar"
)

// AvatarHandler は /avatars 関連のエンドポイントを担当します。
// 新しい usecase.AvatarUsecase を利用します。
type AvatarHandler struct {
	uc *uc.AvatarUsecase
}

// NewAvatarHandler はHTTPハンドラを初期化します。
func NewAvatarHandler(avatarUC *uc.AvatarUsecase) http.Handler {
	return &AvatarHandler{uc: avatarUC}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *AvatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 元のパス（末尾 / を落とす）
	path := strings.TrimSuffix(r.URL.Path, "/")

	// ✅ /sns/avatars にも対応するため /sns を剥がして /avatars 系に正規化する
	// - /sns/avatars        -> /avatars
	// - /sns/avatars/{id}   -> /avatars/{id}
	// - /avatars            -> /avatars (そのまま)
	if strings.HasPrefix(path, "/sns/") {
		path = strings.TrimPrefix(path, "/sns")
		if path == "" {
			path = "/"
		}
	}

	switch {
	case r.Method == http.MethodGet && path == "/avatars":
		// 現行の AvatarUsecase は一覧取得を提供しないため 501 で返す
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})

	case r.Method == http.MethodPost && path == "/avatars":
		h.post(w, r)

	case r.Method == http.MethodGet && strings.HasPrefix(path, "/avatars/"):
		id := strings.TrimPrefix(path, "/avatars/")
		h.get(w, r, id)

	// ✅ NEW: wallet open endpoint
	// POST /avatars/{id}/wallet
	case r.Method == http.MethodPost && strings.HasPrefix(path, "/avatars/") && strings.HasSuffix(path, "/wallet"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/avatars/"), "/wallet")
		h.openWallet(w, r, id)

	case (r.Method == http.MethodPatch || r.Method == http.MethodPut) && strings.HasPrefix(path, "/avatars/"):
		id := strings.TrimPrefix(path, "/avatars/")
		h.update(w, r, id)

	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/avatars/"):
		id := strings.TrimPrefix(path, "/avatars/")
		h.delete(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /avatars/{id}
// aggregate=1|true を付けると Avatar + State + Icons の集約を返します。
func (h *AvatarHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[sns.avatar] GET /avatars/%s aggregate=%q\n", id, r.URL.Query().Get("aggregate"))

	q := r.URL.Query()
	agg := strings.EqualFold(q.Get("aggregate"), "1") || strings.EqualFold(q.Get("aggregate"), "true")

	if agg {
		data, err := h.uc.GetAggregate(ctx, id)
		if err != nil {
			writeAvatarErr(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(data)
		return
	}

	avatar, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeAvatarErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(avatar)
}

// POST /avatars
//
// body:
//
//	{
//	  "userId": "xxx",
//	  "userUid": "firebase-auth-uid",
//	  "avatarName": "name",
//	  "avatarIcon": "https://... or gs://... or path (optional)",
//	  "profile": "... (optional)",
//	  "externalLink": "https://... (optional)"
//	}
//
// ✅ 期待値: AvatarUsecase 側で avatar 作成と同時に wallet を開設。
// ✅ LOG: 受け取ったデータ（PIIはマスク/長さのみ）
func (h *AvatarHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body struct {
		UserID       string  `json:"userId"`
		UserUID      string  `json:"userUid"`
		AvatarName   string  `json:"avatarName"`
		AvatarIcon   *string `json:"avatarIcon,omitempty"`
		Profile      *string `json:"profile,omitempty"`
		ExternalLink *string `json:"externalLink,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	in := uc.CreateAvatarInput{
		UserID:       strings.TrimSpace(body.UserID),
		UserUID:      strings.TrimSpace(body.UserUID),
		AvatarName:   strings.TrimSpace(body.AvatarName),
		AvatarIcon:   trimPtr(body.AvatarIcon),   // shared helper_handler.go
		Profile:      trimPtr(body.Profile),      // shared helper_handler.go
		ExternalLink: trimPtr(body.ExternalLink), // shared helper_handler.go
	}

	log.Printf(
		"[sns.avatar] POST /avatars request userId=%q userUid=%q avatarName=%q avatarIcon=%q profile_len=%d externalLink=%q\n",
		in.UserID,
		maskUID(in.UserUID), // shared
		in.AvatarName,
		ptrStr(in.AvatarIcon),   // shared
		ptrLen(in.Profile),      // shared
		ptrStr(in.ExternalLink), // shared
	)

	created, err := h.uc.Create(ctx, in)
	if err != nil {
		log.Printf("[sns.avatar] POST /avatars error=%v\n", err)
		writeAvatarErr(w, err)
		return
	}

	log.Printf(
		"[sns.avatar] POST /avatars ok avatarId=%q walletAddress=%q\n",
		created.ID,
		ptrStr(created.WalletAddress),
	)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// POST /avatars/{id}/wallet
//
// 任意で wallet 開設を単独実行したい場合のエンドポイント。
// - 既に walletAddress がある場合は 409 Conflict で返す。
func (h *AvatarHandler) openWallet(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[sns.avatar] POST /avatars/%s/wallet request\n", id)

	// すでに walletAddress があるなら衝突扱い
	a, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[sns.avatar] POST /avatars/%s/wallet get error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}
	if a.WalletAddress != nil && strings.TrimSpace(*a.WalletAddress) != "" {
		log.Printf("[sns.avatar] POST /avatars/%s/wallet conflict walletAddress=%q\n", id, ptrStr(a.WalletAddress))
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet already opened"})
		return
	}

	updated, err := h.uc.OpenWallet(ctx, id)
	if err != nil {
		log.Printf("[sns.avatar] POST /avatars/%s/wallet error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}

	log.Printf(
		"[sns.avatar] POST /avatars/%s/wallet ok walletAddress=%q\n",
		id,
		ptrStr(updated.WalletAddress),
	)
	_ = json.NewEncoder(w).Encode(updated)
}

// PATCH/PUT /avatars/{id}
//
// body (patch):
//
//	{
//	  "avatarName": "name (optional)",
//	  "avatarIcon": "https://... or path (optional, empty => null)",
//	  "profile": "... (optional, empty => null)",
//	  "externalLink": "https://... (optional, empty => null)"
//	}
//
// ✅ SECURITY: walletAddress は受け付けない（開設は POST /avatars または POST /avatars/{id}/wallet のみ）
// ✅ LOG: 受け取ったデータ（raw JSON 先頭のみも出す）
func (h *AvatarHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	// Raw で受ける（walletAddress 混入を検知する）
	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// walletAddress が含まれていたら明示的に拒否
	if _, ok := raw["walletAddress"]; ok {
		log.Printf("[sns.avatar] PATCH/PUT /avatars/%s rejected: walletAddress field is not allowed\n", id)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "walletAddress is not allowed in update"})
		return
	}

	// raw をログ（PIIを避けるため先頭だけ）
	bs, merr := json.Marshal(raw)
	if merr != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}
	log.Printf("[sns.avatar] PATCH/PUT /avatars/%s raw=%q\n", id, headString(bs, 300))

	// raw → typed
	var body struct {
		AvatarName   *string `json:"avatarName,omitempty"`
		AvatarIcon   *string `json:"avatarIcon,omitempty"`
		Profile      *string `json:"profile,omitempty"`
		ExternalLink *string `json:"externalLink,omitempty"`
	}
	if err := json.Unmarshal(bs, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	patch := avatardom.AvatarPatch{
		AvatarName:   trimPtrNilAware(body.AvatarName),   // nil => no update, "" => invalid の想定
		AvatarIcon:   trimPtrNilAware(body.AvatarIcon),   // "" => nil (clear)
		Profile:      trimPtrNilAware(body.Profile),      // "" => nil (clear)
		ExternalLink: trimPtrNilAware(body.ExternalLink), // "" => nil (clear)
	}

	log.Printf(
		"[sns.avatar] PATCH/PUT /avatars/%s request avatarName=%q avatarIcon=%q profile_len=%d externalLink=%q\n",
		id,
		ptrStr(patch.AvatarName),
		ptrStr(patch.AvatarIcon),
		ptrLen(patch.Profile),
		ptrStr(patch.ExternalLink),
	)

	updated, err := h.uc.Update(ctx, id, patch)
	if err != nil {
		log.Printf("[sns.avatar] PATCH/PUT /avatars/%s error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}

	log.Printf("[sns.avatar] PATCH/PUT /avatars/%s ok\n", id)
	_ = json.NewEncoder(w).Encode(updated)
}

// DELETE /avatars/{id}
func (h *AvatarHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[sns.avatar] DELETE /avatars/%s request\n", id)

	if err := h.uc.Delete(ctx, id); err != nil {
		log.Printf("[sns.avatar] DELETE /avatars/%s error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}

	log.Printf("[sns.avatar] DELETE /avatars/%s ok\n", id)
	w.WriteHeader(http.StatusNoContent)
}

// エラーハンドリング
func writeAvatarErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	// invalid id / invalid input
	if errors.Is(err, avatardom.ErrInvalidID) ||
		errors.Is(err, avatardom.ErrInvalidUserID) ||
		errors.Is(err, uc.ErrInvalidUserUID) ||
		errors.Is(err, avatardom.ErrInvalidAvatarName) ||
		errors.Is(err, avatardom.ErrInvalidProfile) ||
		errors.Is(err, avatardom.ErrInvalidExternalLink) {
		code = http.StatusBadRequest
	}

	// NotFound が存在する場合だけ 404 にする（存在しない環境でもコンパイルを壊さない）
	if hasErrNotFound(err) {
		code = http.StatusNotFound
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// nil は「更新しない」
// non-nil かつ空文字は「null にする（クリア）」扱いにしたいフィールドで使う
func trimPtrNilAware(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		// 明示クリア（repository 側は optionalString により nil 保存）
		return ptr("")
	}
	return &s
}

func ptr[T any](v T) *T { return &v }

// avatardom.ErrNotFound が無い環境でもコンパイルを壊さないための判定。
// - message 文字列に頼る best-effort（暫定）
func hasErrNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "not_found")
}
