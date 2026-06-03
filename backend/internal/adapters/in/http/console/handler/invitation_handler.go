// backend/internal/adapters/in/http/console/handler/invitation_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	compdom "narratives/internal/domain/company"
	invdom "narratives/internal/domain/invitation"
	memdom "narratives/internal/domain/member"
)

/*
InvitationHandler
- POST /invitations
- POST /invitations/validate
- POST /invitations/complete
*/

type InvitationHandler struct {
	InvitationUC usecase.InvitationUsecasePort
	CompanyRepo  compdom.Repository
	BrandRepo    branddom.Repository
}

func NewInvitationHandler(
	invitationUC usecase.InvitationUsecasePort,
	companyRepo compdom.Repository,
	brandRepo branddom.Repository,
) *InvitationHandler {
	return &InvitationHandler{
		InvitationUC: invitationUC,
		CompanyRepo:  companyRepo,
		BrandRepo:    brandRepo,
	}
}

type invitationInfoResponse struct {
	MemberID         string   `json:"memberId,omitempty"`
	CompanyID        string   `json:"companyId,omitempty"`
	CompanyName      string   `json:"companyName,omitempty"`
	AssignedBrandIDs []string `json:"assignedBrandIds,omitempty"`
	BrandNames       []string `json:"brandNames,omitempty"`
	Permissions      []string `json:"permissions,omitempty"`
	Email            string   `json:"email,omitempty"`
}

type invitationValidateRequest struct {
	Token string `json:"token"`
}

type createInvitationRequest struct {
	MemberID string `json:"memberId"`
}

type createInvitationResponse struct {
	MemberID string `json:"memberId"`
	Token    string `json:"token"`
}

func (h *InvitationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	path := strings.TrimRight(r.URL.Path, "/")
	if path == "" {
		path = "/"
	}

	switch path {
	case "/invitations":
		h.handleCreateInvitation(w, r)
		return

	case "/invitations/validate":
		h.handleResolveInfo(w, r)
		return

	case "/invitations/complete":
		h.handleComplete(w, r)
		return

	default:
		writeInvitationJSONError(w, http.StatusNotFound, "not_found")
		return
	}
}

func (h *InvitationHandler) handleCreateInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeInvitationJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if h.InvitationUC == nil {
		writeInvitationJSONError(w, http.StatusInternalServerError, "invitation_usecase_not_configured")
		return
	}

	var req createInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInvitationJSONError(w, http.StatusBadRequest, "invalid_body")
		return
	}

	memberID := strings.TrimSpace(req.MemberID)
	if memberID == "" {
		writeInvitationJSONError(w, http.StatusBadRequest, "memberId_required")
		return
	}

	token, err := h.InvitationUC.CreateInvitationAndSend(r.Context(), memberID)
	if err != nil {
		switch {
		case errors.Is(err, memdom.ErrNotFound):
			writeInvitationJSONError(w, http.StatusNotFound, "member_not_found")
			return
		default:
			writeInvitationJSONError(w, http.StatusInternalServerError, "cannot_send_invitation")
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(createInvitationResponse{
		MemberID: memberID,
		Token:    token,
	})
}

func (h *InvitationHandler) handleResolveInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeInvitationJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if h.InvitationUC == nil {
		writeInvitationJSONError(w, http.StatusInternalServerError, "invitation_usecase_not_configured")
		return
	}

	var req invitationValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInvitationJSONError(w, http.StatusBadRequest, "invalid_body")
		return
	}

	token := strings.TrimSpace(req.Token)
	if token == "" {
		writeInvitationJSONError(w, http.StatusBadRequest, "token_required")
		return
	}

	ctx := r.Context()
	info, err := h.InvitationUC.GetInvitationInfo(ctx, token)
	if err != nil {
		if errors.Is(err, invdom.ErrInvitationTokenNotFound) || errors.Is(err, memdom.ErrNotFound) {
			writeInvitationJSONError(w, http.StatusNotFound, "invitation_token_not_found")
			return
		}
		writeInvitationJSONError(w, http.StatusInternalServerError, "failed_to_resolve_invitation_token")
		return
	}

	companyName := info.CompanyID
	if h.CompanyRepo != nil && info.CompanyID != "" {
		companyEntity, err := h.CompanyRepo.GetByID(ctx, info.CompanyID)
		if err != nil {
			if !errors.Is(err, compdom.ErrNotFound) {
				writeInvitationJSONError(w, http.StatusInternalServerError, "failed_to_resolve_company_name")
				return
			}
		} else if companyEntity.Name != "" {
			companyName = companyEntity.Name
		}
	}

	brandNames := info.AssignedBrandIDs
	if h.BrandRepo != nil && len(info.AssignedBrandIDs) > 0 {
		resolved := make([]string, 0, len(info.AssignedBrandIDs))

		for _, brandID := range info.AssignedBrandIDs {
			brandID = strings.TrimSpace(brandID)
			if brandID == "" {
				continue
			}

			brand, err := h.BrandRepo.GetByID(ctx, brandID)
			if err != nil {
				if errors.Is(err, branddom.ErrNotFound) || errors.Is(err, branddom.ErrInvalidID) {
					resolved = append(resolved, brandID)
					continue
				}

				writeInvitationJSONError(w, http.StatusInternalServerError, "failed_to_resolve_brand_name")
				return
			}

			resolved = append(resolved, brand.Name)
		}

		brandNames = resolved
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(invitationInfoResponse{
		MemberID:         info.MemberID,
		CompanyID:        info.CompanyID,
		CompanyName:      companyName,
		AssignedBrandIDs: info.AssignedBrandIDs,
		BrandNames:       brandNames,
		Permissions:      info.Permissions,
		Email:            info.Email,
	})
}

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

	if h.InvitationUC == nil {
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

	err := h.InvitationUC.CompleteInvitation(r.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, invdom.ErrInvitationTokenNotFound), errors.Is(err, memdom.ErrNotFound):
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
			writeInvitationJSONError(w, http.StatusInternalServerError, "failed_to_complete_invitation")
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeInvitationJSONError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
