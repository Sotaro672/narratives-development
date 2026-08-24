// backend/internal/adapters/in/http/console/handler/transaction_handler.go
package consoleHandler

import (
	"encoding/json"
	"net/http"

	transactionq "narratives/internal/application/query/console"
	common "narratives/internal/domain/common"
)

// TransactionHandler handles:
//
//	GET /transactions
//
// authenticated Company に属する Settlement を基に、
// Stripe Connect の入出金履歴を返す。
type TransactionHandler struct {
	q *transactionq.TransactionManagementQuery
}

func NewTransactionHandler(
	q *transactionq.TransactionManagementQuery,
) http.Handler {
	return &TransactionHandler{
		q: q,
	}
}

func (h *TransactionHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	switch {
	case r.Method == http.MethodGet &&
		(r.URL.Path == "/transactions" ||
			r.URL.Path == "/transactions/"):
		h.list(w, r)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "not_found",
			},
		)
		return
	}
}

func (h *TransactionHandler) list(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.q == nil {
		w.WriteHeader(
			http.StatusInternalServerError,
		)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "transaction_management_query_not_wired",
			},
		)
		return
	}

	filter, page, err :=
		parseOrderListParams(
			r,
		)
	if err != nil {
		w.WriteHeader(
			http.StatusBadRequest,
		)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": err.Error(),
			},
		)
		return
	}

	result, err :=
		h.q.List(
			r.Context(),
			filter,
			common.Sort{},
			page,
		)
	if err != nil {
		w.WriteHeader(
			http.StatusInternalServerError,
		)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": err.Error(),
			},
		)
		return
	}

	w.WriteHeader(
		http.StatusOK,
	)

	_ = json.NewEncoder(w).Encode(
		result,
	)
}
