// backend/internal/adapters/in/http/mall/handler/resale_chat_handler.go
package mallHandler

import "net/http"

func (h *ResaleHandler) listOwnedResaleChats(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()

	if h == nil || h.query == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "not_implemented",
		})
		return
	}

	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	items, err := h.query.ListChatItems(ctx, avatarID)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":      items,
		"totalCount": len(items),
	})
}

func (h *ResaleHandler) getOwnedResaleChatBadgeCount(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()

	if h == nil || h.query == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "not_implemented",
		})
		return
	}

	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	count, err := h.query.CountUnreadCommentsByAvatarID(ctx, avatarID)
	if err != nil {
		writeResaleErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"unreadCommentCount": count,
	})
}
