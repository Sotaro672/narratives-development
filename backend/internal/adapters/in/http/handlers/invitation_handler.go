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
GET /api/invitation?token=INV_xxx
*/

type InvitationHandler struct {
	InvitationQuery usecase.InvitationQueryPort
}

func NewInvitationHandler(inv usecase.InvitationQueryPort) *InvitationHandler {
	return &InvitationHandler{
		InvitationQuery: inv,
	}
}

type invitationInfoResponse struct {
	MemberID         string   `json:"memberId"`
	CompanyID        string   `json:"companyId"`
	AssignedBrandIDs []string `json:"assignedBrandIds"`
	Permissions      []string `json:"permissions"`
	Email            string   `json:"email,omitempty"`
}

// =====================================
// GET /api/invitation?token=xxx
// =====================================
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
		if errors.Is(err, memdom.ErrInvitationTokenNotFound) || errors.Is(err, memdom.ErrNotFound) {
			http.Error(w, "invitation token not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to resolve invitation token", http.StatusInternalServerError)
		return
	}

	resp := invitationInfoResponse{
		MemberID:         info.MemberID,
		CompanyID:        info.CompanyID,
		AssignedBrandIDs: info.AssignedBrandIDs,
		Permissions:      info.Permissions,
		Email:            info.Email,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

/*
=====================================
POST /api/invitation/complete
（サインイン後の member 確定）
=====================================
*/

type invitationCompleteRequest struct {
	Token         string `json:"token"`
	UID           string `json:"uid"`
	LastName      string `json:"lastName"`
	LastNameKana  string `json:"lastNameKana"`
	FirstName     string `json:"firstName"`
	FirstNameKana string `json:"firstNameKana"`
	Email         string `json:"email"`
}

func (h *InvitationHandler) Complete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.InvitationQuery == nil {
		http.Error(w, "invitation usecase not configured", http.StatusInternalServerError)
		return
	}

	var req invitationCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	token := strings.TrimSpace(req.Token)
	if token == "" || strings.TrimSpace(req.UID) == "" {
		http.Error(w, "token / uid required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// 1) token → InvitationInfo
	info, err := h.InvitationQuery.GetInvitationInfo(ctx, token)
	if err != nil {
		if errors.Is(err, memdom.ErrInvitationTokenNotFound) {
			http.Error(w, "invitation token not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to resolve invitation token", http.StatusInternalServerError)
		return
	}

	// ★ 暫定対応：未実装でも「info を使った」扱いにする
	_ = info.MemberID

	// ★ ここに MemberUsecase.CompleteInvitation(...) を後で実装して呼び出す
	// h.MemberUsecase.CompleteInvitation(ctx, *info, req)

	w.WriteHeader(http.StatusNoContent)
}

/*
=====================================
MemberInvitationHandler
POST /members/{id}/invitation
=====================================
*/

type MemberInvitationHandler struct {
	InvitationCommand usecase.InvitationCommandPort
}

func NewMemberInvitationHandler(cmd usecase.InvitationCommandPort) *MemberInvitationHandler {
	return &MemberInvitationHandler{
		InvitationCommand: cmd,
	}
}

func (h *MemberInvitationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

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
