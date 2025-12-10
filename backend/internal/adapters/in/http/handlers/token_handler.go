// backend\internal\adapters\in\http\handlers\token_handler.go
package handlers

import (
	"encoding/json"
	"net/http"

	"narratives/internal/application/usecase"
)

type DevnetMintHandler struct {
	TokenUC *usecase.TokenUsecase
}

type devnetMintRequest struct {
	ToAddress   string `json:"toAddress"`
	MetadataURI string `json:"metadataUri"`
	Amount      uint64 `json:"amount"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
}

type devnetMintResponse struct {
	Signature   string `json:"signature"`
	MintAddress string `json:"mintAddress"`
	Slot        uint64 `json:"slot"`
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

	var req devnetMintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body: "+err.Error(), http.StatusBadRequest)
		return
	}

	input := usecase.MintDirectInput{
		ToAddress:   req.ToAddress,
		Amount:      req.Amount,
		MetadataURI: req.MetadataURI,
		Name:        req.Name,
		Symbol:      req.Symbol,
	}

	ctx := r.Context()
	result, err := h.TokenUC.MintDirect(ctx, input)
	if err != nil {
		http.Error(w, "mint failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := devnetMintResponse{
		Signature:   result.Signature,
		MintAddress: result.MintAddress,
		Slot:        result.Slot,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
