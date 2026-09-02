// backend/internal/adapters/in/http/mall/handler/like_handler.go
package mallHandler

import (
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	likedom "narratives/internal/domain/like"
)

const mallMeLikesPath = "/mall/me/likes"

type LikeHandler struct {
	uc *usecase.LikeUsecase
}

func NewLikeHandler(uc *usecase.LikeUsecase) http.Handler {
	return &LikeHandler{uc: uc}
}

func (h *LikeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if h == nil || h.uc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "like_usecase_not_configured",
		})
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	if path == mallMeLikesPath {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.listLikes(w, r)
		return
	}

	if !strings.HasPrefix(path, mallMeLikesPath+"/") {
		notFound(w)
		return
	}

	rest := strings.TrimPrefix(path, mallMeLikesPath+"/")
	parts := strings.Split(rest, "/")

	if len(parts) != 2 {
		notFound(w)
		return
	}

	targetType, ok := parseLikeTargetType(parts[0])
	if !ok {
		notFound(w)
		return
	}

	targetID := strings.TrimSpace(parts[1])
	if targetID == "" {
		notFound(w)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getLikeStatus(w, r, targetType, targetID)
	case http.MethodPut:
		h.addLike(w, r, targetType, targetID)
	case http.MethodDelete:
		h.removeLike(w, r, targetType, targetID)
	default:
		methodNotAllowed(w)
	}
}

func (h *LikeHandler) listLikes(w http.ResponseWriter, r *http.Request) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	filter := likedom.Filter{}

	if rawTargetType := strings.TrimSpace(r.URL.Query().Get("targetType")); rawTargetType != "" {
		targetType, ok := parseLikeTargetType(rawTargetType)
		if !ok {
			writeLikeErr(w, likedom.ErrInvalidTargetType)
			return
		}

		filter.TargetType = &targetType
	}

	pageNumber := parsePositiveIntDefault(r.URL.Query().Get("page"), 1)
	perPage := parsePositiveIntDefault(r.URL.Query().Get("perPage"), 20)
	if perPage > 100 {
		perPage = 100
	}

	result, err := h.uc.ListByAvatarID(
		r.Context(),
		avatarID,
		filter,
		likedom.Sort{
			Column: "createdAt",
			Order:  likedom.SortDesc,
		},
		likedom.Page{
			Number:  pageNumber,
			PerPage: perPage,
		},
	)
	if err != nil {
		writeLikeErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":      result.Items,
		"totalCount": result.TotalCount,
		"totalPages": result.TotalPages,
		"page":       result.Page,
		"perPage":    result.PerPage,
	})
}

func (h *LikeHandler) getLikeStatus(
	w http.ResponseWriter,
	r *http.Request,
	targetType likedom.TargetType,
	targetID string,
) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	result, err := h.uc.GetStatus(
		r.Context(),
		avatarID,
		targetType,
		targetID,
	)
	if err != nil {
		writeLikeErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": result,
	})
}

func (h *LikeHandler) addLike(
	w http.ResponseWriter,
	r *http.Request,
	targetType likedom.TargetType,
	targetID string,
) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	result, err := h.uc.Add(
		r.Context(),
		avatarID,
		targetType,
		targetID,
	)
	if err != nil {
		writeLikeErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": result,
	})
}

func (h *LikeHandler) removeLike(
	w http.ResponseWriter,
	r *http.Request,
	targetType likedom.TargetType,
	targetID string,
) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	result, err := h.uc.Remove(
		r.Context(),
		avatarID,
		targetType,
		targetID,
	)
	if err != nil {
		writeLikeErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": result,
	})
}

func parseLikeTargetType(raw string) (likedom.TargetType, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(likedom.TargetTypeList):
		return likedom.TargetTypeList, true
	case string(likedom.TargetTypeResale):
		return likedom.TargetTypeResale, true
	default:
		return "", false
	}
}

func writeLikeErr(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "internal_error",
		})

	case errors.Is(err, usecase.ErrLikeRepositoryMissing),
		errors.Is(err, usecase.ErrLikeListRepositoryMissing),
		errors.Is(err, usecase.ErrLikeResaleRepositoryMissing):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "like_service_not_configured",
		})

	case likedom.IsInvalid(err):
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})

	case likedom.IsNotFound(err):
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not_found",
		})

	case likedom.IsConflict(err):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "conflict",
		})

	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "internal_error",
		})
	}
}
