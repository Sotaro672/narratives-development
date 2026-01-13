// backend/internal/adapters/in/http/mall/handler/preview_me_handler.go
package mallHandler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	sharedquery "narratives/internal/application/query/shared"

	// ✅ avatarId を context から取得する
	"narratives/internal/adapters/in/http/middleware"
)

type PreviewMeHandler struct {
	q      PreviewQuery
	ownerQ *sharedquery.OwnerResolveQuery // optional (DIで注入できるように)
}

func NewPreviewMeHandler(q PreviewQuery) http.Handler {
	return &PreviewMeHandler{q: q, ownerQ: nil}
}

// ✅ DI(register.go) から呼ぶ想定の constructor（owner resolve を handler 側でもbest-effortで付与できる）
func NewPreviewMeHandlerWithOwner(q PreviewQuery, ownerQ *sharedquery.OwnerResolveQuery) http.Handler {
	return &PreviewMeHandler{q: q, ownerQ: ownerQ}
}

func (h *PreviewMeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "method not allowed",
		})
		return
	}

	if h == nil || h.q == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "preview query not configured",
		})
		return
	}

	// ✅ /mall/me/preview は認証前提（通常は middleware で検証される）
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "authorization header is required",
		})
		return
	}

	// ✅ AvatarContextMiddleware が入っていればここで取れる（best-effort）
	avatarID, _ := middleware.CurrentAvatarID(r)

	productID := strings.TrimSpace(r.URL.Query().Get("productId"))
	if productID == "" {
		// 互換: /mall/me/preview/{productId}
		productID = extractLastPathSegment(r.URL.Path, "/mall/me/preview")
	}

	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "productId is required",
		})
		return
	}

	// ✅ 入口ログ
	authPrefix := ""
	if len(auth) > 12 {
		authPrefix = auth[:12]
	} else {
		authPrefix = auth
	}
	log.Printf(
		`[mall.preview.me] incoming method=%s path=%s rawQuery=%q hasAuth=%t authPrefix=%q avatarId=%q`,
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
		auth != "",
		authPrefix,
		avatarID,
	)

	log.Printf(`[mall.preview.me] resolving model info productId=%q avatarId=%q`, productID, avatarID)

	info, err := h.q.ResolveModelInfoByProductID(r.Context(), productID)
	if err != nil {
		log.Printf("[mall.preview.me] ResolveModelInfoByProductID failed: %v", err)

		if isNotFound(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error":     "not found",
				"productId": productID,
				"avatarId":  avatarID,
			})
			return
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			writeJSON(w, http.StatusRequestTimeout, map[string]any{
				"error":     "request canceled",
				"productId": productID,
				"avatarId":  avatarID,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "resolve failed",
			"productId": productID,
			"avatarId":  avatarID,
		})
		return
	}
	if info == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "resolve failed (nil result)",
			"productId": productID,
			"avatarId":  avatarID,
		})
		return
	}

	// ------------------------------------------------------------
	// ✅ owner resolve (best-effort)
	// - PreviewQuery 側で owner を付与している可能性があるが、
	//   DI で ownerQ が注入されている場合は handler 側でも補完する。
	// - token.toAddress が取れた時のみ試す（空ならスキップ）
	// ------------------------------------------------------------
	if info.Owner == nil && h.ownerQ != nil && info.Token != nil {
		addr := strings.TrimSpace(info.Token.ToAddress)
		if addr != "" {
			res, rerr := h.ownerQ.Resolve(r.Context(), addr)
			if rerr == nil {
				info.Owner = res
				oid := strings.TrimSpace(res.AvatarID)
				if oid == "" {
					oid = strings.TrimSpace(res.BrandID)
				}
				log.Printf(`[mall.preview.me] owner resolved walletAddress=%q ownerType=%q id=%q`, res.WalletAddress, res.OwnerType, oid)
			} else if errors.Is(rerr, sharedquery.ErrOwnerNotFound) || errors.Is(rerr, sharedquery.ErrInvalidWalletAddress) {
				// not fatal
				log.Printf(`[mall.preview.me] owner resolve skipped walletAddress=%q err=%v`, addr, rerr)
			} else if errors.Is(rerr, context.Canceled) || errors.Is(rerr, context.DeadlineExceeded) {
				log.Printf(`[mall.preview.me] owner resolve canceled walletAddress=%q err=%v`, addr, rerr)
			} else if errors.Is(rerr, sharedquery.ErrOwnerResolveNotConfigured) {
				log.Printf(`[mall.preview.me] owner resolve not configured walletAddress=%q err=%v`, addr, rerr)
			} else {
				log.Printf(`[mall.preview.me] owner resolve failed walletAddress=%q err=%v`, addr, rerr)
			}
		}
	}

	log.Printf(
		`[mall.preview.me] resolved avatarId=%q productId=%q modelId=%q modelNumber=%q size=%q color=%q rgb=%d measurements=%v productBlueprintId=%q productBlueprintPatch=%v token=%t owner=%t`,
		avatarID,
		info.ProductID,
		info.ModelID,
		info.ModelNumber,
		info.Size,
		info.Color,
		info.RGB,
		info.Measurements,
		info.ProductBlueprintID,
		info.ProductBlueprintPatch,
		info.Token != nil,
		info.Owner != nil,
	)

	// ✅ avatarId も一緒に返す（PreviewModelInfo にフィールド追加しない方針）
	writeJSON(w, http.StatusOK, map[string]any{
		"data":     info,
		"avatarId": avatarID,
	})
}
