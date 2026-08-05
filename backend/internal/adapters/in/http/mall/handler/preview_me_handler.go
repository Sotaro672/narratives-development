// backend/internal/adapters/in/http/mall/handler/preview_me_handler.go
package mallHandler

import (
	"net/http"

	"narratives/internal/adapters/in/http/middleware"
	sharedquery "narratives/internal/application/query/shared"
)

type PreviewMeHandler struct {
	q PreviewQuery

	// optional
	ownerQ *sharedquery.OwnerResolveQuery

	// tokenBlueprint patch (optional)
	tbRepo TokenBlueprintPatchReader

	// name resolver (optional)
	nameR PreviewNameResolver
}

func NewPreviewMeHandler(
	q PreviewQuery,
	ownerQ *sharedquery.OwnerResolveQuery,
	tbRepo TokenBlueprintPatchReader,
	nameR PreviewNameResolver,
) http.Handler {
	return &PreviewMeHandler{
		q:      q,
		ownerQ: ownerQ,
		tbRepo: tbRepo,
		nameR:  nameR,
	}
}

func (h *PreviewMeHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if !validatePreviewGETRequest(w, r) {
		return
	}

	auth := r.Header.Get("Authorization")
	if auth == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "authorization header is required",
		})
		return
	}

	avatarID, _ := middleware.CurrentAvatarID(r)

	var (
		q      PreviewQuery
		ownerQ *sharedquery.OwnerResolveQuery
		tbRepo TokenBlueprintPatchReader
		nameR  PreviewNameResolver
	)

	if h != nil {
		q = h.q
		ownerQ = h.ownerQ
		tbRepo = h.tbRepo
		nameR = h.nameR
	}

	info, ok := resolvePreviewModelInfoFromRequest(
		w,
		r,
		q,
		map[string]any{
			"avatarId": avatarID,
		},
	)
	if !ok {
		return
	}

	data := buildPreviewData(
		r.Context(),
		info,
		ownerQ,
		tbRepo,
		nameR,
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"data":     data,
		"avatarId": avatarID,
	})
}
