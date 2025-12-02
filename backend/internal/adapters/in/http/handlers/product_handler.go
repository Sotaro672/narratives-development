// backend/internal/adapters/in/http/handlers/product_handler.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	uc "narratives/internal/application/usecase"
)

// ProductHandler は /inspector/products 関連のエンドポイントを担当します。
// 現時点では検品アプリ用の「検品詳細取得」のみを扱います。
type ProductHandler struct {
	uc *uc.ProductUsecase
}

// NewProductHandler は ProductUsecase を受け取って HTTP ハンドラを生成します。
func NewProductHandler(ucase *uc.ProductUsecase) http.Handler {
	return &ProductHandler{uc: ucase}
}

// ServeHTTP は /inspector/products/{productId} を扱います。
//
// フロント側（inspector アプリ）からのリクエスト仕様:
//
//	GET /inspector/products/{productId}
//	Header: Authorization: Bearer <idToken>
//
// レスポンス: InspectorProductDetail 互換 JSON
func (h *ProductHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	// ------------------------------------------------------------
	// GET /inspector/products/{productId}
	// ------------------------------------------------------------
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/inspector/products/"):

		productID := strings.TrimPrefix(r.URL.Path, "/inspector/products/")
		productID = strings.TrimSpace(productID)
		if productID == "" {
			http.Error(w, `{"error":"productId is required"}`, http.StatusBadRequest)
			return
		}

		detail, err := h.uc.GetInspectorProductDetail(r.Context(), productID)
		if err != nil {
			// ここではシンプルに 500 を返す。
			// 必要に応じてドメインエラーごとに 404 / 400 など出し分けてもよい。
			http.Error(w, `{"error":"`+escapeForJSON(err.Error())+`"}`, http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(detail); err != nil {
			http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
			return
		}

	default:
		http.NotFound(w, r)
	}
}

// escapeForJSON はエラーメッセージを簡易的に JSON 文字列に埋め込める形にします。
// （ここではダブルクォートと改行だけ最低限エスケープ）
func escapeForJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}
