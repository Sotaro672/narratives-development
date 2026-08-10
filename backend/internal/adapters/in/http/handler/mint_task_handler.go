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

func (h *MintTaskHandler) executeNextMintTask(
	w http.ResponseWriter,
	r *http.Request,
) {
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
		log.Printf(
			"[mint-task] mint usecase is not configured",
		)

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

	log.Printf(
		"[mint-task] execute start mintID=%s",
		mintID,
	)

	result, err := h.mintUC.ExecuteNextMintTask(
		r.Context(),
		mintID,
	)
	if err != nil {
		// 実行可能な task がない場合:
		// - 全件完了済み
		// - FAILED_FATAL のみ残っている
		// - すでに別 worker が処理済み
		//
		// Cloud Tasks に 5xx を返すと不要な retry が走るため、
		// ここは 200 で終了扱いにします。
		if errors.Is(
			err,
			mintdom.ErrMintProductTaskNotFound,
		) {
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

		// 資金補充や設定変更など、人間の対応が必要なエラーは
		// Cloud Tasks の自動再配送では自然復旧しません。
		//
		// そのため HTTP 200 を返して Cloud Tasks の retry loop を停止します。
		//
		// MintUsecase 側では、これらの再開可能エラーを
		// FAILED_RETRYABLE として保存する必要があります。
		//
		// 対応完了後に mint task を1件 enqueue することで、
		// FAILED_RETRYABLE -> MINTING -> MINTED と再開します。
		if status, ok := manualRetryStatus(err); ok {
			log.Printf(
				"[mint-task] manual retry required mintID=%s status=%s error=%v",
				mintID,
				status,
				err,
			)

			writeJSON(w, http.StatusOK, mintTaskResponse{
				MintID:  mintID,
				Status:  status,
				Message: err.Error(),
			})
			return
		}

		// RPC 429 / 5xx / timeout / connection failure など、
		// 短時間で自然復旧する可能性があるエラーは 500 を返します。
		//
		// Cloud Tasks が configured retry policy に従って
		// task を再配送します。
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

	writeJSON(
		w,
		http.StatusOK,
		resp,
	)
}

func manualRetryStatus(err error) (string, bool) {
	if err == nil {
		return "", false
	}

	msg := strings.ToLower(err.Error())

	// reserve wallet の資金不足。
	//
	// reserve wallet に SOL を補充すれば再開できます。
	if strings.Contains(
		msg,
		"reserve wallet balance is insufficient",
	) ||
		strings.Contains(
			msg,
			"reserve wallet balance fell below minimum after top-up",
		) {
		return "WAITING_FOR_RESERVE_BALANCE", true
	}

	// reserve wallet / auto top-up の設定不備。
	//
	// 環境変数や Secret Manager の設定を修正して
	// Cloud Run を再deployした後に再開します。
	if strings.Contains(
		msg,
		"reserve wallet is not configured",
	) ||
		strings.Contains(
			msg,
			"fee payer top-up config is invalid",
		) {
		return "WAITING_FOR_SOLANA_CONFIGURATION", true
	}

	// auto top-up が無効な環境で fee payer の残高が不足した場合の
	// fallbackです。
	//
	// 通常のCloud Run環境では
	// SOLANA_AUTO_TOP_UP_ENABLED=true を設定するため、
	// この分岐には基本的に入りません。
	if strings.Contains(
		msg,
		"fee payer balance is below minimum",
	) ||
		strings.Contains(
			msg,
			"fee payer auto top-up is disabled",
		) {
		return "WAITING_FOR_FEE_PAYER_BALANCE", true
	}

	return "", false
}

func extractMintIDFromPath(path string) string {
	p := strings.TrimSpace(path)
	p = strings.TrimPrefix(
		p,
		"/internal/mint/tasks/",
	)
	p = strings.TrimSuffix(
		p,
		"/execute",
	)
	p = strings.Trim(
		p,
		"/",
	)

	return p
}

func writeJSON(
	w http.ResponseWriter,
	statusCode int,
	payload any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(payload)
}
