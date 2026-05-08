// backend\internal\adapters\in\http\console\handler\invitation_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	compdom "narratives/internal/domain/company"
	memdom "narratives/internal/domain/member"
)

/*
InvitationHandler
- GET  /api/invitation?token=INV_xxx
- POST /api/invitation/validate
- POST /api/invitation/complete
*/

type InvitationHandler struct {
	InvitationQuery    usecase.InvitationQueryPort
	InvitationComplete usecase.InvitationCompletePort
	CompanyService     *compdom.Service
	BrandService       *branddom.Service
}

func NewInvitationHandler(
	inv usecase.InvitationQueryPort,
	complete usecase.InvitationCompletePort,
	companyService *compdom.Service,
	brandService *branddom.Service,
) *InvitationHandler {
	return &InvitationHandler{
		InvitationQuery:    inv,
		InvitationComplete: complete,
		CompanyService:     companyService,
		BrandService:       brandService,
	}
}

// --------------------------------------------------
// 共通レスポンス型（GET / validate 共通）
// --------------------------------------------------

type invitationInfoResponse struct {
	MemberID         string   `json:"memberId,omitempty"`
	CompanyID        string   `json:"companyId,omitempty"`
	AssignedBrandIDs []string `json:"assignedBrandIds,omitempty"`
	Permissions      []string `json:"permissions,omitempty"`
	Email            string   `json:"email,omitempty"`
}

type invitationValidateRequest struct {
	Token string `json:"token"`
}

// =====================================
// ルーティング分岐
// =====================================

func (h *InvitationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	path := strings.TrimRight(strings.TrimPrefix(r.URL.Path, "/api/invitation"), "/")
	if path == "" {
		h.handleResolveInfo(w, r)
		return
	}

	switch path {
	case "/validate":
		h.handleResolveInfo(w, r)
	case "/complete":
		h.handleComplete(w, r)
	default:
		writeInvitationJSONError(w, http.StatusNotFound, "not_found")
	}
}

// =====================================
// GET /api/invitation?token=xxx
// POST /api/invitation/validate
// 共通処理
// =====================================

func (h *InvitationHandler) handleResolveInfo(w http.ResponseWriter, r *http.Request) {
	if h.InvitationQuery == nil {
		writeInvitationJSONError(w, http.StatusInternalServerError, "invitation_usecase_not_configured")
		return
	}

	var token string

	switch r.Method {
	case http.MethodGet:
		token = r.URL.Query().Get("token")
		if token == "" {
			writeInvitationJSONError(w, http.StatusBadRequest, "missing_token")
			return
		}

	case http.MethodPost:
		var req invitationValidateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeInvitationJSONError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		token = req.Token
		if token == "" {
			writeInvitationJSONError(w, http.StatusBadRequest, "token_required")
			return
		}

	default:
		w.Header().Set("Allow", "GET, POST")
		writeInvitationJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	ctx := r.Context()
	info, err := h.InvitationQuery.GetInvitationInfo(ctx, token)
	if err != nil {
		log.Printf("[InvitationHandler.handleResolveInfo] failed: method=%s token=%s err=%v", r.Method, token, err)
		if errors.Is(err, memdom.ErrInvitationTokenNotFound) || errors.Is(err, memdom.ErrNotFound) {
			writeInvitationJSONError(w, http.StatusNotFound, "invitation_token_not_found")
			return
		}
		writeInvitationJSONError(w, http.StatusInternalServerError, "failed_to_resolve_invitation_token")
		return
	}

	companyName := info.CompanyID
	if h.CompanyService != nil && info.CompanyID != "" {
		name, err := h.CompanyService.GetCompanyNameByID(ctx, info.CompanyID)
		if err != nil {
			if !errors.Is(err, compdom.ErrNotFound) {
				log.Printf("[InvitationHandler.handleResolveInfo] failed to resolve company name: companyID=%s err=%v", info.CompanyID, err)
				writeInvitationJSONError(w, http.StatusInternalServerError, "failed_to_resolve_company_name")
				return
			}
		} else {
			companyName = name
		}
	}

	assignedBrandNames := info.AssignedBrandIDs
	if h.BrandService != nil && len(info.AssignedBrandIDs) > 0 {
		resolved := make([]string, 0, len(info.AssignedBrandIDs))
		for _, brandID := range info.AssignedBrandIDs {
			if brandID == "" {
				continue
			}

			name, err := h.BrandService.GetNameByID(ctx, brandID)
			if err != nil {
				if errors.Is(err, branddom.ErrNotFound) || errors.Is(err, branddom.ErrInvalidID) {
					resolved = append(resolved, brandID)
					continue
				}
				log.Printf("[InvitationHandler.handleResolveInfo] failed to resolve brand name: brandID=%s err=%v", brandID, err)
				writeInvitationJSONError(w, http.StatusInternalServerError, "failed_to_resolve_brand_name")
				return
			}

			resolved = append(resolved, name)
		}
		assignedBrandNames = resolved
	}

	resp := invitationInfoResponse{
		MemberID:         info.MemberID,
		CompanyID:        companyName,        // 画面には companyId キーで会社名を渡す
		AssignedBrandIDs: assignedBrandNames, // 画面には assignedBrandIds キーでブランド名を渡す
		Permissions:      info.Permissions,
		Email:            info.Email,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

/*
=====================================
POST /api/invitation/complete
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

func (h *InvitationHandler) handleComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeInvitationJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if h.InvitationComplete == nil {
		writeInvitationJSONError(w, http.StatusInternalServerError, "invitation_complete_usecase_not_configured")
		return
	}

	var req invitationCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInvitationJSONError(w, http.StatusBadRequest, "invalid_body")
		return
	}

	in := usecase.CompleteInvitationInput{
		Token:         req.Token,
		UID:           req.UID,
		LastName:      req.LastName,
		LastNameKana:  req.LastNameKana,
		FirstName:     req.FirstName,
		FirstNameKana: req.FirstNameKana,
		Email:         req.Email,
	}

	err := h.InvitationComplete.CompleteInvitation(r.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, memdom.ErrInvitationTokenNotFound), errors.Is(err, memdom.ErrNotFound):
			writeInvitationJSONError(w, http.StatusNotFound, "invitation_token_or_member_not_found")
			return
		case err.Error() == "token_or_uid_required":
			writeInvitationJSONError(w, http.StatusBadRequest, "token_or_uid_required")
			return
		case err.Error() == "name_fields_required":
			writeInvitationJSONError(w, http.StatusBadRequest, "name_fields_required")
			return
		case err.Error() == "email_required":
			writeInvitationJSONError(w, http.StatusBadRequest, "email_required")
			return
		case err.Error() == "email_mismatch":
			writeInvitationJSONError(w, http.StatusBadRequest, "email_mismatch")
			return
		default:
			log.Printf("[InvitationHandler.handleComplete] failed: token=%s uid=%s err=%v", req.Token, req.UID, err)
			writeInvitationJSONError(w, http.StatusInternalServerError, "failed_to_complete_invitation")
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

/*
=====================================
MemberInvitationHandler
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeInvitationJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	path := strings.TrimRight(strings.TrimPrefix(r.URL.Path, "/members/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "invitation" {
		writeInvitationJSONError(w, http.StatusNotFound, "not_found")
		return
	}
	memberID := parts[0]

	if h.InvitationCommand == nil {
		writeInvitationJSONError(w, http.StatusInternalServerError, "invitation_command_usecase_not_configured")
		return
	}

	ctx := r.Context()
	token, err := h.InvitationCommand.CreateInvitationAndSend(ctx, memberID)
	if err != nil {
		log.Printf("[MemberInvitationHandler] failed: memberID=%s err=%v", memberID, err)
		writeInvitationJSONError(w, http.StatusInternalServerError, "cannot_send_invitation")
		return
	}

	resp := map[string]string{
		"memberId": memberID,
		"token":    token,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func writeInvitationJSONError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
