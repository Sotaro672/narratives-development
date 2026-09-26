// backend/internal/adapters/in/http/mall/handler/tokenBlueprint_report_handler.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	mallquery "narratives/internal/application/query/mall"
	appusecase "narratives/internal/application/usecase"
	reportdom "narratives/internal/domain/report"
	tokenblueprint "narratives/internal/domain/tokenBlueprint"
)

// TokenBlueprintReportService owns avatar-side reporting of TokenBlueprints.
type TokenBlueprintReportService interface {
	ReportTokenBlueprintByAvatar(
		ctx context.Context,
		input appusecase.ReportTokenBlueprintByAvatarInput,
	) (reportdom.AddReportResult, error)
}

// TokenBlueprintModerationService owns Mall-side reads of the AMOL moderation
// status for TokenBlueprints.
type TokenBlueprintModerationService interface {
	GetModerationStatus(
		ctx context.Context,
		input mallquery.GetTokenBlueprintModerationStatusInput,
	) (mallquery.TokenBlueprintModerationStatusReadModel, error)
}

type TokenBlueprintReportHandler struct {
	reportSvc     TokenBlueprintReportService
	moderationSvc TokenBlueprintModerationService
}

func NewTokenBlueprintReportHandler(
	reportSvc TokenBlueprintReportService,
	moderationSvc TokenBlueprintModerationService,
) *TokenBlueprintReportHandler {
	return &TokenBlueprintReportHandler{
		reportSvc:     reportSvc,
		moderationSvc: moderationSvc,
	}
}

func (h *TokenBlueprintReportHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	path := strings.TrimSuffix(r.URL.Path, "/")

	tokenBlueprintID, action, ok := parseTokenBlueprintActionPath(path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch action {
	case "reports":
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if h == nil || h.reportSvc == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "report service not configured")
			return
		}
		h.handleReport(w, r, tokenBlueprintID)

	case "moderation-status":
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if h == nil || h.moderationSvc == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "moderation service not configured")
			return
		}
		h.handleModerationStatus(w, r, tokenBlueprintID)

	default:
		http.NotFound(w, r)
	}
}

func (h *TokenBlueprintReportHandler) handleReport(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
) {
	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing avatarId")
		return
	}

	tokenBlueprintID = strings.TrimSpace(tokenBlueprintID)
	if tokenBlueprintID == "" {
		writeJSONError(w, http.StatusBadRequest, "tokenBlueprintId is required")
		return
	}

	reason, detail, ok := decodeReportRequest(w, r)
	if !ok {
		return
	}

	result, err := h.reportSvc.ReportTokenBlueprintByAvatar(
		r.Context(),
		appusecase.ReportTokenBlueprintByAvatarInput{
			TokenBlueprintID: tokenBlueprintID,
			AvatarID:         avatarID,
			Reason:           reason,
			Detail:           detail,
		},
	)
	if err != nil {
		writeTokenBlueprintReportError(w, err)
		return
	}

	writeReportResult(w, result)
}

func (h *TokenBlueprintReportHandler) handleModerationStatus(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
) {
	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing avatarId")
		return
	}

	tokenBlueprintID = strings.TrimSpace(tokenBlueprintID)
	if tokenBlueprintID == "" {
		writeJSONError(w, http.StatusBadRequest, "tokenBlueprintId is required")
		return
	}

	result, err := h.moderationSvc.GetModerationStatus(
		r.Context(),
		mallquery.GetTokenBlueprintModerationStatusInput{
			AvatarID:         avatarID,
			TokenBlueprintID: tokenBlueprintID,
		},
	)
	if err != nil {
		writeTokenBlueprintModerationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func parseTokenBlueprintActionPath(
	path string,
) (tokenBlueprintID string, action string, ok bool) {
	const prefix = "/mall/me/token-blueprints/"

	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}

	relative := strings.TrimPrefix(path, prefix)
	parts := strings.Split(relative, "/")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}

	switch parts[1] {
	case "reports", "moderation-status":
		return parts[0], parts[1], true
	default:
		return "", "", false
	}
}

func writeTokenBlueprintReportError(
	w http.ResponseWriter,
	err error,
) {
	if err == nil {
		writeJSONError(w, http.StatusInternalServerError, "unknown error")
		return
	}

	switch {
	case tokenblueprint.IsNotFound(err):
		writeJSONError(w, http.StatusNotFound, "token blueprint not found")
	case tokenblueprint.IsInvalid(err):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	default:
		writeReportError(w, err)
	}
}

func writeTokenBlueprintModerationError(
	w http.ResponseWriter,
	err error,
) {
	if err == nil {
		writeJSONError(w, http.StatusInternalServerError, "unknown error")
		return
	}

	switch {
	case tokenblueprint.IsNotFound(err):
		writeJSONError(w, http.StatusNotFound, "token blueprint not found")

	case tokenblueprint.IsInvalid(err):
		writeJSONError(w, http.StatusBadRequest, err.Error())

	case errors.Is(
		err,
		mallquery.ErrMallTokenBlueprintModerationAvatarIDRequired,
	):
		writeJSONError(w, http.StatusBadRequest, "avatarId is required")

	case errors.Is(
		err,
		mallquery.ErrMallTokenBlueprintModerationTokenBlueprintIDRequired,
	):
		writeJSONError(w, http.StatusBadRequest, "tokenBlueprintId is required")

	case errors.Is(
		err,
		mallquery.ErrMallTokenBlueprintModerationForbidden,
	):
		writeJSONError(w, http.StatusForbidden, "forbidden")

	case errors.Is(
		err,
		mallquery.ErrMallTokenBlueprintModerationQueryNotConfigured,
	):
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"moderation service not configured",
		)

	default:
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"failed to get token blueprint moderation status",
		)
	}
}
