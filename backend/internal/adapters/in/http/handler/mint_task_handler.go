// backend/internal/adapters/in/http/handler/mint_task_handler.go
package internalHandler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	mintapp "narratives/internal/application/usecase"
	mintdom "narratives/internal/domain/mint"
)

type MintTaskHandler struct {
	mintUC *mintapp.MintUsecase
}

func NewMintTaskHandler(mintUC *mintapp.MintUsecase) http.Handler {
	return &MintTaskHandler{
		mintUC: mintUC,
	}
}

type mintTaskRequest struct {
	MintID string `json:"mintId"`
}

type mintTaskResponse struct {
	MintID      string `json:"mintId"`
	Status      string `json:"status"`
	Signature   string `json:"signature,omitempty"`
	MintAddress string `json:"mintAddress,omitempty"`
	Slot        uint64 `json:"slot,omitempty"`
	Message     string `json:"message,omitempty"`
}

func (h *MintTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	log.Printf(
		"[mint-task] request received method=%s path=%s userAgent=%s remoteAddr=%s",
		r.Method,
		r.URL.Path,
		r.UserAgent(),
		r.RemoteAddr,
	)

	if r.Method != http.MethodPost {
		log.Printf(
			"[mint-task] method not allowed method=%s path=%s",
			r.Method,
			r.URL.Path,
		)

		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method not allowed",
		})
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/internal/mint/tasks/") ||
		!strings.HasSuffix(r.URL.Path, "/execute") {
		log.Printf(
			"[mint-task] route not found path=%s",
			r.URL.Path,
		)

		http.NotFound(w, r)
		return
	}

	h.executeNextMintTask(w, r)
}

func (h *MintTaskHandler) executeNextMintTask(w http.ResponseWriter, r *http.Request) {
	mintID := extractMintIDFromPath(r.URL.Path)

	defer func() {
		if rec := recover(); rec != nil {
			log.Printf(
				"[mint-task] panic mintID=%s path=%s panic=%v",
				mintID,
				r.URL.Path,
				rec,
			)

			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "mint task panic",
				"panic": fmt.Sprint(rec),
			})
		}
	}()

	if h.mintUC == nil {
		log.Printf("[mint-task] mint usecase is not configured")

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "mint usecase is not configured",
		})
		return
	}

	var body mintTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Printf(
			"[mint-task] request body decode failed path=%s error=%v",
			r.URL.Path,
			err,
		)
	}

	if mintID == "" {
		mintID = strings.TrimSpace(body.MintID)
	}

	if mintID == "" {
		log.Printf(
			"[mint-task] mintId is empty path=%s bodyMintID=%s",
			r.URL.Path,
			body.MintID,
		)

		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "mintId is empty",
		})
		return
	}

	if strings.Contains(mintID, "/") {
		log.Printf(
			"[mint-task] invalid mintId contains slash mintID=%s path=%s",
			mintID,
			r.URL.Path,
		)

		http.NotFound(w, r)
		return
	}

	log.Printf("[mint-task] execute start mintID=%s", mintID)

	result, err := h.mintUC.ExecuteNextMintTask(r.Context(), mintID)
	if err != nil {
		// 実行可能な task がない場合:
		// - 全件完了済み
		// - FAILED_FATAL のみ残っている
		// - すでに別 worker が処理済み
		//
		// Cloud Tasks に 5xx を返すと不要な retry が走るため、
		// ここは 200 で終了扱いにします。
		if errors.Is(err, mintdom.ErrMintProductTaskNotFound) {
			log.Printf(
				"[mint-task] no executable task mintID=%s error=%v",
				mintID,
				err,
			)

			writeJSON(w, http.StatusOK, mintTaskResponse{
				MintID:  mintID,
				Status:  "NO_EXECUTABLE_TASK",
				Message: "no executable mint product task found",
			})
			return
		}

		log.Printf(
			"[mint-task] execute failed mintID=%s error=%v",
			mintID,
			err,
		)

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	resp := mintTaskResponse{
		MintID:  mintID,
		Status:  "MINT_TASK_EXECUTED",
		Message: "one mint product task was executed",
	}

	if result != nil {
		resp.Signature = result.Signature
		resp.MintAddress = result.MintAddress
		resp.Slot = result.Slot

		log.Printf(
			"[mint-task] execute succeeded mintID=%s mintAddress=%s signature=%s slot=%d",
			mintID,
			result.MintAddress,
			result.Signature,
			result.Slot,
		)
	} else {
		log.Printf(
			"[mint-task] execute succeeded mintID=%s result=nil",
			mintID,
		)
	}

	writeJSON(w, http.StatusOK, resp)
}

func extractMintIDFromPath(path string) string {
	p := strings.TrimSpace(path)
	p = strings.TrimPrefix(p, "/internal/mint/tasks/")
	p = strings.TrimSuffix(p, "/execute")
	p = strings.Trim(p, "/")
	return p
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
