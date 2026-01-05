// backend/internal/adapters/in/http/handlers/token_handler.go
package handlers

import (
	"net/http"

	"narratives/internal/application/usecase"
)

type DevnetMintHandler struct {
	TokenUC *usecase.TokenUsecase
}

func (h *DevnetMintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if h.TokenUC == nil {
		http.Error(w, "TokenUsecase is not initialized", http.StatusInternalServerError)
		return
	}

	// MintDirect は TokenUsecase から削除されたため、
	// この devnet 用エンドポイントは現在無効化しています。
	http.Error(w, "devnet mint endpoint is disabled (MintDirect removed)", http.StatusNotImplemented)
}
