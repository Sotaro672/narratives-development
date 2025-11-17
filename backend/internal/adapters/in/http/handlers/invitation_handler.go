// backend/internal/adapters/in/http/handlers/invitation_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"narratives/internal/application/usecase"
	memdom "narratives/internal/domain/member"
)

/*
InvitationHandler

招待メール内のリンク
  https://console.example.com/invitation?token=INV_xxx

からフロントエンドが叩くエンドポイント用ハンドラ。

責務:
  - クエリパラメータ token を受け取る
  - usecase.InvitationQueryPort 経由で InvitationInfo を取得
  - JSON 形式で companyId / assignedBrandIds / permissions / memberId を返す

利用例（ルーティング側）:
  invHandler := web.NewInvitationHandler(invitationUsecase)
  mux.Handle("/api/invitation", invHandler)
*/

type InvitationHandler struct {
	InvitationQuery usecase.InvitationQueryPort
}

// NewInvitationHandler は招待情報取得用ハンドラを生成します。
func NewInvitationHandler(inv usecase.InvitationQueryPort) *InvitationHandler {
	return &InvitationHandler{
		InvitationQuery: inv,
	}
}

// invitationInfoResponse はフロント向けのレスポンス DTO です。
type invitationInfoResponse struct {
	MemberID         string   `json:"memberId"`
	CompanyID        string   `json:"companyId"`
	AssignedBrandIDs []string `json:"assignedBrandIds"`
	Permissions      []string `json:"permissions"`
}

// ServeHTTP implements http.Handler.
// GET /api/invitation?token=INV_xxx
func (h *InvitationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.InvitationQuery == nil {
		http.Error(w, "invitation usecase not configured", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		http.Error(w, "missing token query parameter", http.StatusBadRequest)
		return
	}

	info, err := h.InvitationQuery.GetInvitationInfo(ctx, token)
	if err != nil {
		// 招待トークンが不正 or 見つからない場合は 404
		if errors.Is(err, memdom.ErrInvitationTokenNotFound) {
			http.Error(w, "invitation token not found", http.StatusNotFound)
			return
		}
		// その他は 500
		http.Error(w, "failed to resolve invitation token", http.StatusInternalServerError)
		return
	}

	resp := invitationInfoResponse{
		MemberID:         info.MemberID,
		CompanyID:        info.CompanyID,
		AssignedBrandIDs: info.AssignedBrandIDs,
		Permissions:      info.Permissions,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
		return
	}
}

/*
MemberInvitationHandler

メンバー作成後に「招待メール送信」を行うためのエンドポイント。

責務:
  - パス /members/{id}/invitation から memberId を抽出
  - usecase.InvitationCommandPort 経由で
      1) 招待トークンを発行
      2) 招待メールを送信
  - JSON 形式で memberId / token を返す

利用例（ルーティング側）:
  memberInvHandler := handlers.NewMemberInvitationHandler(invitationUsecase)
  mux.Handle("/members/", memberInvHandler) // もしくは専用パスにマウント
*/

type MemberInvitationHandler struct {
	InvitationCommand usecase.InvitationCommandPort
}

func NewMemberInvitationHandler(cmd usecase.InvitationCommandPort) *MemberInvitationHandler {
	return &MemberInvitationHandler{
		InvitationCommand: cmd,
	}
}

// POST /members/{id}/invitation
func (h *MemberInvitationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// パスから memberId 抽出
	// 例: /members/12345/invitation
	path := strings.TrimPrefix(r.URL.Path, "/members/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "invitation" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	memberID := parts[0]

	if h.InvitationCommand == nil {
		http.Error(w, "invitation command usecase not configured", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	token, err := h.InvitationCommand.CreateInvitationAndSend(ctx, memberID)
	if err != nil {
		http.Error(w, `{"error":"cannot_send_invitation"}`, http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"memberId": memberID,
		"token":    token,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}
