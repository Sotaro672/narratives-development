// backend/internal/adapters/in/http/mall/handler/preview_handler.go
package mallHandler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	mallQuery "narratives/internal/application/query/mall"
	sharedquery "narratives/internal/application/query/shared"
)

// ✅ 組み立て用（DI）前提:
// backend/internal/application/query/mall/preview_query.go が提供する Query を注入して使う想定。
// この handler は「productId → model info（型番/サイズ/色/RGB/measurements + productBlueprintPatch + token + owner）」を返す。
//
// 想定エンドポイント:
// - GET /mall/preview?productId=...
//
// 互換で以下も吸収:
// - GET /mall/preview/{productId}
type PreviewQuery interface {
	// ✅ 推奨: productId から表示用情報を一括で返す（measurements を含む）
	ResolveModelInfoByProductID(ctx context.Context, productID string) (*mallQuery.PreviewModelInfo, error)
}

type PreviewHandler struct {
	q      PreviewQuery
	ownerQ *sharedquery.OwnerResolveQuery // optional (DIで注入できるように)
}

func NewPreviewHandler(q PreviewQuery) http.Handler {
	return &PreviewHandler{q: q, ownerQ: nil}
}

// ✅ DI(register.go) から呼ぶ想定の constructor（owner resolve を handler 側でもbest-effortで付与できる）
func NewPreviewHandlerWithOwner(q PreviewQuery, ownerQ *sharedquery.OwnerResolveQuery) http.Handler {
	return &PreviewHandler{q: q, ownerQ: ownerQ}
}

func (h *PreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	productID := strings.TrimSpace(r.URL.Query().Get("productId"))
	if productID == "" {
		// 互換: /mall/preview/{productId}
		productID = extractLastPathSegment(r.URL.Path, "/mall/preview")
	}

	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "productId is required",
		})
		return
	}

	// ✅ 入口ログ（Cloud Run ログで「叩かれてるか」を確実に追う）
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	authPrefix := ""
	if auth != "" {
		if len(auth) > 12 {
			authPrefix = auth[:12]
		} else {
			authPrefix = auth
		}
	}
	log.Printf(
		`[mall.preview] incoming method=%s path=%s rawQuery=%q hasAuth=%t authPrefix=%q`,
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
		auth != "",
		authPrefix,
	)

	log.Printf(`[mall.preview] resolving model info productId=%q`, productID)

	info, err := h.q.ResolveModelInfoByProductID(r.Context(), productID)
	if err != nil {
		log.Printf("[mall.preview] ResolveModelInfoByProductID failed: %v", err)

		if isNotFound(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error":     "not found",
				"productId": productID,
			})
			return
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			writeJSON(w, http.StatusRequestTimeout, map[string]any{
				"error":     "request canceled",
				"productId": productID,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "resolve failed",
			"productId": productID,
		})
		return
	}
	if info == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":     "resolve failed (nil result)",
			"productId": productID,
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
				log.Printf(`[mall.preview] owner resolved walletAddress=%q ownerType=%q id=%q`, res.WalletAddress, res.OwnerType, oid)
			} else if errors.Is(rerr, sharedquery.ErrOwnerNotFound) || errors.Is(rerr, sharedquery.ErrInvalidWalletAddress) {
				// not fatal
				log.Printf(`[mall.preview] owner resolve skipped walletAddress=%q err=%v`, addr, rerr)
			} else if errors.Is(rerr, context.Canceled) || errors.Is(rerr, context.DeadlineExceeded) {
				log.Printf(`[mall.preview] owner resolve canceled walletAddress=%q err=%v`, addr, rerr)
			} else if errors.Is(rerr, sharedquery.ErrOwnerResolveNotConfigured) {
				log.Printf(`[mall.preview] owner resolve not configured walletAddress=%q err=%v`, addr, rerr)
			} else {
				log.Printf(`[mall.preview] owner resolve failed walletAddress=%q err=%v`, addr, rerr)
			}
		}
	}

	log.Printf(
		`[mall.preview] resolved productId=%q modelId=%q modelNumber=%q size=%q color=%q rgb=%d measurements=%v productBlueprintId=%q productBlueprintPatch=%v token=%t owner=%t`,
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

	// ✅ info をそのまま返す（owner は best-effort で付与される）
	writeJSON(w, http.StatusOK, map[string]any{
		"data": info,
	})
}

func extractLastPathSegment(path string, prefix string) string {
	p := strings.TrimSuffix(path, "/")
	prefix = strings.TrimSuffix(prefix, "/")

	if p == prefix {
		return ""
	}
	if !strings.HasPrefix(p, prefix+"/") {
		return ""
	}
	rest := strings.TrimPrefix(p, prefix+"/")
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return ""
	}
	if i := strings.Index(rest, "/"); i >= 0 {
		rest = rest[:i]
	}
	return strings.TrimSpace(rest)
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "not found") || strings.Contains(msg, "no such") {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return false
}
