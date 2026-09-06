// backend/internal/adapters/in/http/mall/handler/avatar_me_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	avataruc "narratives/internal/application/usecase"
	avatardom "narratives/internal/domain/avatar"
	reviewreport "narratives/internal/domain/reviewReport"
)

// Policy (me-only):
// - uidは認証コンテキストから取得し、クライアント入力では受けない
// - avatarIdはサーバー側でuidから解決する
//
// Endpoints:
// - GET    /mall/me/avatars
// - PATCH  /mall/me/avatars
// - DELETE /mall/me/avatars
// - POST   /mall/me/avatars/{targetAvatarId}/reports
type MeAvatarResolver interface {
	ResolveAvatarByUID(ctx context.Context, uid string) (avatarID string, walletAddress string, err error)
}

type MeAvatarHandler struct {
	Repo           MeAvatarResolver
	AvatarUC       *avataruc.AvatarUsecase
	ReviewReportUC *avataruc.ReviewReportUsecase
}

type meAvatarResponse struct {
	AvatarID      string  `json:"avatarId"`
	UserID        string  `json:"userId"`
	AvatarName    string  `json:"avatarName"`
	AvatarIcon    *string `json:"avatarIcon,omitempty"`
	WalletAddress string  `json:"walletAddress"`
	Profile       *string `json:"profile,omitempty"`
	ExternalLink  *string `json:"externalLink,omitempty"`
}

type meAvatarReportRequest struct {
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

type meAvatarReportResponse struct {
	CaseID        string                  `json:"caseId"`
	ReportID      string                  `json:"reportId"`
	ReportCount   int                     `json:"reportCount"`
	Status        reviewreport.CaseStatus `json:"status"`
	CaseCreated   bool                    `json:"caseCreated"`
	ReportCreated bool                    `json:"reportCreated"`
}

func NewMeAvatarHandler(
	repo MeAvatarResolver,
	avatarUC *avataruc.AvatarUsecase,
	reviewReportUC *avataruc.ReviewReportUsecase,
) http.Handler {
	return &MeAvatarHandler{
		Repo:           repo,
		AvatarUC:       avatarUC,
		ReviewReportUC: reviewReportUC,
	}
}

const meAvatarsPath = "/mall/me/avatars"

func (h *MeAvatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if h == nil || h.Repo == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "me_avatar_handler_not_initialized",
		})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "unauthorized: missing uid",
		})
		return
	}

	path0 := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodGet && path0 == meAvatarsPath:
		h.handleGet(w, r, uid)
		return

	case r.Method == http.MethodPatch && path0 == meAvatarsPath:
		h.handlePatch(w, r, uid)
		return

	case r.Method == http.MethodDelete && path0 == meAvatarsPath:
		h.handleDelete(w, r, uid)
		return

	case r.Method == http.MethodPost:
		targetAvatarID, ok := parseMeAvatarReportPath(path0)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "not_found",
			})
			return
		}
		h.handleReport(w, r, uid, targetAvatarID)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "not_found",
		})
		return
	}
}

func parseMeAvatarReportPath(path string) (string, bool) {
	const prefix = meAvatarsPath + "/"

	if !strings.HasPrefix(path, prefix) {
		return "", false
	}

	relative := strings.TrimPrefix(path, prefix)
	parts := strings.Split(relative, "/")

	if len(parts) != 2 ||
		parts[0] == "" ||
		parts[1] != "reports" {
		return "", false
	}

	return parts[0], true
}

func httpAvatarIcon(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}

	if !strings.HasPrefix(*value, "http://") &&
		!strings.HasPrefix(*value, "https://") {
		return nil
	}

	return value
}

func newMeAvatarResponse(
	avatarID string,
	patch avatardom.AvatarPatch,
) (meAvatarResponse, error) {
	if avatarID == "" {
		return meAvatarResponse{}, avatardom.ErrInvalidID
	}

	if patch.UserID == "" {
		return meAvatarResponse{}, avatardom.ErrInvalidUserID
	}

	if patch.AvatarName == nil || *patch.AvatarName == "" {
		return meAvatarResponse{}, avatardom.ErrInvalidAvatarName
	}

	if patch.WalletAddress == nil || *patch.WalletAddress == "" {
		return meAvatarResponse{}, avatardom.ErrInvalidWalletAddressLink
	}

	return meAvatarResponse{
		AvatarID:      avatarID,
		UserID:        patch.UserID,
		AvatarName:    *patch.AvatarName,
		AvatarIcon:    httpAvatarIcon(patch.AvatarIcon),
		WalletAddress: *patch.WalletAddress,
		Profile:       patch.Profile,
		ExternalLink:  patch.ExternalLink,
	}, nil
}

func (h *MeAvatarHandler) ResolveAvatarByUID(
	ctx context.Context,
	uid string,
) (string, string, avatardom.AvatarPatch, error) {
	if h == nil || h.Repo == nil {
		return "", "", avatardom.AvatarPatch{}, errors.New("me avatar handler not configured")
	}

	if h.AvatarUC == nil {
		return "", "", avatardom.AvatarPatch{}, errors.New("avatar usecase not configured")
	}

	avatarID, walletAddress, err := h.Repo.ResolveAvatarByUID(ctx, uid)
	if err != nil {
		return "", "", avatardom.AvatarPatch{}, err
	}

	if avatarID == "" {
		return "", walletAddress, avatardom.AvatarPatch{}, avatardom.ErrInvalidID
	}

	av, err := h.AvatarUC.GetByID(ctx, avatarID)
	if err != nil {
		return "", walletAddress, avatardom.AvatarPatch{}, err
	}

	patch := avatardom.AvatarPatch{
		UserID:        av.UserID,
		AvatarName:    &av.AvatarName,
		AvatarIcon:    av.AvatarIcon,
		WalletAddress: av.WalletAddress,
		Profile:       av.Profile,
		ExternalLink:  av.ExternalLink,
	}

	return avatarID, walletAddress, patch, nil
}

func (h *MeAvatarHandler) updateAvatarPatchByUID(
	ctx context.Context,
	uid string,
	patch avatardom.AvatarPatch,
) (string, avatardom.AvatarPatch, error) {
	if h == nil || h.Repo == nil {
		return "", avatardom.AvatarPatch{}, errors.New("me avatar handler not configured")
	}

	if h.AvatarUC == nil {
		return "", avatardom.AvatarPatch{}, errors.New("avatar usecase not configured")
	}

	avatarID, _, _, err := h.ResolveAvatarByUID(ctx, uid)
	if err != nil {
		return "", avatardom.AvatarPatch{}, err
	}

	// userIdとwalletAddressは本人向けPATCH APIの更新対象外。
	patch.UserID = ""
	patch.WalletAddress = nil

	updated, err := h.AvatarUC.Update(ctx, avatarID, patch)
	if err != nil {
		return "", avatardom.AvatarPatch{}, err
	}

	out := avatardom.AvatarPatch{
		UserID:        updated.UserID,
		AvatarName:    &updated.AvatarName,
		AvatarIcon:    updated.AvatarIcon,
		WalletAddress: updated.WalletAddress,
		Profile:       updated.Profile,
		ExternalLink:  updated.ExternalLink,
	}

	return avatarID, out, nil
}

func (h *MeAvatarHandler) handleGet(
	w http.ResponseWriter,
	r *http.Request,
	uid string,
) {
	avatarID, _, patch, err := h.ResolveAvatarByUID(r.Context(), uid)
	if err != nil {
		writeMeAvatarErr(w, err)
		return
	}

	if avatarID == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "avatar_not_found_for_uid",
		})
		return
	}

	if patch.WalletAddress == nil || *patch.WalletAddress == "" {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "wallet_address_not_initialized",
		})
		return
	}

	out, err := newMeAvatarResponse(avatarID, patch)
	if err != nil {
		writeMeAvatarErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}

func (h *MeAvatarHandler) handlePatch(
	w http.ResponseWriter,
	r *http.Request,
	uid string,
) {
	type meAvatarUpdateRequest struct {
		AvatarName   *string `json:"avatarName,omitempty"`
		Profile      *string `json:"profile,omitempty"`
		ExternalLink *string `json:"externalLink,omitempty"`
		AvatarIcon   *string `json:"avatarIcon,omitempty"`
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_body",
		})
		return
	}

	if len(raw) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "empty_body",
		})
		return
	}

	var req meAvatarUpdateRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_json",
		})
		return
	}

	if req.AvatarName == nil &&
		req.Profile == nil &&
		req.ExternalLink == nil &&
		req.AvatarIcon == nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "no_fields_to_update",
		})
		return
	}

	patch := avatardom.AvatarPatch{
		AvatarName:   req.AvatarName,
		AvatarIcon:   req.AvatarIcon,
		Profile:      req.Profile,
		ExternalLink: req.ExternalLink,
	}

	avatarID, outPatch, err := h.updateAvatarPatchByUID(
		r.Context(),
		uid,
		patch,
	)
	if err != nil {
		writeMeAvatarErr(w, err)
		return
	}

	if avatarID == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "avatar_not_found_for_uid",
		})
		return
	}

	if outPatch.WalletAddress == nil || *outPatch.WalletAddress == "" {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "wallet_address_not_initialized",
		})
		return
	}

	out, err := newMeAvatarResponse(avatarID, outPatch)
	if err != nil {
		writeMeAvatarErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}

func (h *MeAvatarHandler) handleDelete(
	w http.ResponseWriter,
	r *http.Request,
	uid string,
) {
	if h == nil || h.AvatarUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "avatar usecase not configured",
		})
		return
	}

	avatarID, _, _, err := h.ResolveAvatarByUID(r.Context(), uid)
	if err != nil {
		writeMeAvatarErr(w, err)
		return
	}

	if err := h.AvatarUC.Delete(r.Context(), avatarID); err != nil {
		writeMeAvatarErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *MeAvatarHandler) handleReport(
	w http.ResponseWriter,
	r *http.Request,
	uid string,
	targetAvatarID string,
) {
	if h == nil || h.ReviewReportUC == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "review report service not configured")
		return
	}

	targetAvatarID = strings.TrimSpace(targetAvatarID)
	if targetAvatarID == "" {
		writeJSONError(w, http.StatusBadRequest, "target avatarId is required")
		return
	}

	reporterAvatarID, _, _, err := h.ResolveAvatarByUID(r.Context(), uid)
	if err != nil {
		writeMeAvatarErr(w, err)
		return
	}
	if reporterAvatarID == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing avatarId")
		return
	}

	var req meAvatarReportRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	reason := reviewreport.ReportReason(
		strings.ToUpper(strings.TrimSpace(req.Reason)),
	)
	if err := reason.Validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid report reason")
		return
	}

	req.Detail = strings.TrimSpace(req.Detail)
	if reason == reviewreport.ReportReasonOther && req.Detail == "" {
		writeJSONError(w, http.StatusBadRequest, "report detail required")
		return
	}

	result, err := h.ReviewReportUC.ReportAvatarByAvatar(
		r.Context(),
		avataruc.ReportAvatarByAvatarInput{
			TargetAvatarID:   targetAvatarID,
			ReporterAvatarID: reporterAvatarID,
			Reason:           reason,
			Detail:           req.Detail,
		},
	)
	if err != nil {
		writeMeAvatarReportErr(w, err)
		return
	}

	statusCode := http.StatusCreated
	if !result.ReportCreated {
		statusCode = http.StatusOK
	}

	writeJSON(w, statusCode, meAvatarReportResponse{
		CaseID:        string(result.Case.ID),
		ReportID:      string(result.Report.ID),
		ReportCount:   result.Case.ReportCount,
		Status:        result.Case.Status,
		CaseCreated:   result.CaseCreated,
		ReportCreated: result.ReportCreated,
	})
}

func writeMeAvatarErr(w http.ResponseWriter, err error) {
	code := meAvatarHTTPStatus(err)
	message := meAvatarErrorMessage(err)

	writeJSON(w, code, map[string]string{
		"error": message,
	})
}

func writeMeAvatarReportErr(w http.ResponseWriter, err error) {
	if err == nil {
		writeJSONError(w, http.StatusInternalServerError, "unknown error")
		return
	}

	if isNotFound(err) {
		writeJSONError(w, http.StatusNotFound, "target avatar not found")
		return
	}

	writeReviewReportError(w, err)
}

func meAvatarHTTPStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusInternalServerError

	case isNotFound(err):
		return http.StatusNotFound

	case errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		return http.StatusRequestTimeout

	case errors.Is(err, avatardom.ErrInvalidID):
		return http.StatusNotFound

	case errors.Is(err, avatardom.ErrInvalidAvatarName),
		errors.Is(err, avatardom.ErrInvalidAvatarIcon),
		errors.Is(err, avatardom.ErrInvalidProfile),
		errors.Is(err, avatardom.ErrInvalidExternalLink):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}

func meAvatarErrorMessage(err error) string {
	switch {
	case err == nil:
		return "internal_error"

	case isNotFound(err):
		return "avatar_not_found_for_uid"

	case errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		return "request_timeout"

	case errors.Is(err, avatardom.ErrInvalidID):
		return "avatar_not_found_for_uid"

	case errors.Is(err, avatardom.ErrInvalidAvatarName),
		errors.Is(err, avatardom.ErrInvalidAvatarIcon),
		errors.Is(err, avatardom.ErrInvalidProfile),
		errors.Is(err, avatardom.ErrInvalidExternalLink):
		return err.Error()

	default:
		return "internal_error"
	}
}
