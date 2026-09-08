// backend/internal/adapters/in/http/mall/handler/resale_comment_handler.go
package mallHandler

import (
	"encoding/json"
	"net/http"

	usecase "narratives/internal/application/usecase"
)

type resaleCommentRequest struct {
	Body string `json:"body"`
}

func (h *ResaleHandler) listOwnedResaleComments(
	w http.ResponseWriter,
	r *http.Request,
	resaleID string,
) {
	ctx := r.Context()

	if h == nil || h.resaleReviewUC == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "not_implemented",
		})
		return
	}

	if _, ok := h.getOwnedResale(
		w,
		r,
		ctx,
		resaleID,
	); !ok {
		return
	}

	result, err := h.resaleReviewUC.ListComments(
		ctx,
		resaleID,
		buildResaleReviewPageFromQuery(r),
	)
	if err != nil {
		writeResaleReviewErr(w, err)
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

func (h *ResaleHandler) createOwnedResaleComment(
	w http.ResponseWriter,
	r *http.Request,
	resaleID string,
) {
	ctx := r.Context()

	if h == nil || h.resaleReviewUC == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "not_implemented",
		})
		return
	}

	item, ok := h.getOwnedResale(
		w,
		r,
		ctx,
		resaleID,
	)
	if !ok {
		return
	}

	var req resaleCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid_json",
		})
		return
	}

	comment, err := h.resaleReviewUC.CreateComment(
		ctx,
		usecase.CreateResaleReviewCommentInput{
			ResaleID: resaleID,
			AvatarID: item.AvatarID,
			Body:     req.Body,
		},
	)
	if err != nil {
		writeResaleReviewErr(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"data": comment,
	})
}

func (h *ResaleHandler) markOwnedResaleCommentsRead(
	w http.ResponseWriter,
	r *http.Request,
	resaleID string,
) {
	ctx := r.Context()

	if h == nil || h.resaleReviewUC == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "not_implemented",
		})
		return
	}

	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	markedCount, err := h.resaleReviewUC.MarkCommentsRead(
		ctx,
		resaleID,
		avatarID,
	)
	if err != nil {
		writeResaleReviewErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"resaleId":    resaleID,
		"markedCount": markedCount,
	})
}

func (h *ResaleHandler) deleteOwnedResaleComment(
	w http.ResponseWriter,
	r *http.Request,
	resaleID string,
	commentID string,
) {
	ctx := r.Context()

	if h == nil || h.resaleReviewUC == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "not_implemented",
		})
		return
	}

	item, ok := h.getOwnedResale(
		w,
		r,
		ctx,
		resaleID,
	)
	if !ok {
		return
	}

	err := h.resaleReviewUC.DeleteComment(
		ctx,
		resaleID,
		commentID,
		item.AvatarID,
	)
	if err != nil {
		writeResaleReviewErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
	})
}
