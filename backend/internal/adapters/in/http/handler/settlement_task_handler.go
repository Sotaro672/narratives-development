// backend/internal/adapters/in/http/handler/settlement_task_handler.go
package internalHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"google.golang.org/api/idtoken"

	uc "narratives/internal/application/usecase"
	settlementdom "narratives/internal/domain/settlement"
)

const (
	envSettlementCloudTasksAudience = "CLOUD_TASKS_AUDIENCE"

	envSettlementCloudTasksServiceAccount = "CLOUD_TASKS_SERVICE_ACCOUNT"

	envSettlementInternalBaseURL = "INTERNAL_BASE_URL"

	envSettlementSelfBaseURL = "SELF_BASE_URL"

	maxSettlementTaskRequestBodyBytes int64 = 64 * 1024
)

var (
	errSettlementTaskAuthNotConfigured = errors.New(
		"settlement task authentication is not configured",
	)

	errSettlementTaskUnauthorized = errors.New(
		"settlement task request is unauthorized",
	)

	errSettlementTaskForbidden = errors.New(
		"settlement task request is forbidden",
	)
)

// ============================================================
// Usecase Port
// ============================================================

// SettlementTaskUsecase is the minimal application contract required by the
// Cloud Tasks settlement worker and reconciliation endpoint.
//
// *usecase.SettlementUsecase satisfies this interface.
type SettlementTaskUsecase interface {
	TransferByID(
		ctx context.Context,
		settlementID string,
	) (settlementdom.Settlement, error)

	DispatchDue(
		ctx context.Context,
		queue uc.SettlementTransferQueue,
		limit int,
	) (int, error)
}

// ============================================================
// Handler
// ============================================================

// SettlementTaskHandler processes Stripe Connect settlement tasks.
//
// Cloud Tasks:
//
//	POST /internal/settlements/process
//
// Cloud Scheduler:
//
//	POST /internal/settlements/dispatch-due
//
// Process body:
//
//	{
//	  "settlementId": "..."
//	}
//
// DispatchDue body:
//
//	{
//	  "limit": 50
//	}
//
// An empty DispatchDue body is also valid and uses the application default
// limit.
//
// The process request never contains:
//
// - transfer amount
// - Stripe destination Account
// - Charge ID
// - TransferGroup
//
// Those values are always loaded from the authoritative Settlement document
// by SettlementUsecase.
type SettlementTaskHandler struct {
	settlementUC SettlementTaskUsecase

	settlementQueue uc.SettlementTransferQueue

	audience string

	serviceAccountEmail string
}

type processSettlementTaskRequest struct {
	SettlementID string `json:"settlementId"`
}

type dispatchDueSettlementTaskRequest struct {
	Limit int `json:"limit"`
}

type dispatchDueSettlementTaskResponse struct {
	Enqueued int `json:"enqueued"`

	Error string `json:"error,omitempty"`
}

type settlementTaskErrorResponse struct {
	Error string `json:"error"`
}

// NewSettlementTaskHandler creates the internal Settlement handler.
func NewSettlementTaskHandler(
	settlementUC SettlementTaskUsecase,
	settlementQueue uc.SettlementTransferQueue,
) *SettlementTaskHandler {
	audience :=
		firstNonEmptySettlementTaskEnvironmentValue(
			envSettlementCloudTasksAudience,
			envSettlementInternalBaseURL,
			envSettlementSelfBaseURL,
		)

	serviceAccountEmail :=
		firstNonEmptySettlementTaskEnvironmentValue(
			envSettlementCloudTasksServiceAccount,
		)

	return &SettlementTaskHandler{
		settlementUC: settlementUC,

		settlementQueue: settlementQueue,

		audience: strings.TrimRight(
			audience,
			"/",
		),

		serviceAccountEmail: strings.ToLower(
			serviceAccountEmail,
		),
	}
}

func (h *SettlementTaskHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	path :=
		strings.TrimRight(
			r.URL.Path,
			"/",
		)

	switch path {
	case "/internal/settlements/process":
		h.Process(
			w,
			r,
		)

	case "/internal/settlements/dispatch-due":
		h.DispatchDue(
			w,
			r,
		)

	default:
		writeSettlementTaskError(
			w,
			http.StatusNotFound,
			"not_found",
		)
	}
}

// ============================================================
// Process
// ============================================================

// Process executes one seller-side Stripe Connect Settlement.
//
// Successful and terminal states return 204 so Cloud Tasks acknowledges the
// task.
//
// Retryable or uncertain states return 500 so Cloud Tasks keeps retrying.
//
// Important:
//
// SettlementUsecase and SettlementRepositoryFS provide financial idempotency:
//
//  1. ClaimForTransfer performs an atomic Firestore claim.
//  2. Stripe Transfer uses a deterministic Idempotency-Key.
//  3. CompleteTransfer records the resulting tr_xxx atomically.
func (h *SettlementTaskHandler) Process(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set(
			"Allow",
			http.MethodPost,
		)

		writeSettlementTaskError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
		)

		return
	}

	if h == nil ||
		h.settlementUC == nil {
		writeSettlementTaskError(
			w,
			http.StatusServiceUnavailable,
			"settlement_usecase_unavailable",
		)

		return
	}

	if err :=
		h.authorizeInternalRequest(
			r,
			true,
		); err != nil {
		h.writeAuthorizationError(
			w,
			err,
		)

		return
	}

	var request processSettlementTaskRequest

	if err :=
		decodeRequiredSettlementTaskJSON(
			w,
			r,
			&request,
		); err != nil {
		writeSettlementTaskError(
			w,
			http.StatusBadRequest,
			"invalid_json_body",
		)

		return
	}

	if request.SettlementID == "" ||
		strings.Contains(
			request.SettlementID,
			"/",
		) {
		writeSettlementTaskError(
			w,
			http.StatusBadRequest,
			"settlement_id_required",
		)

		return
	}

	settlement, err :=
		h.settlementUC.TransferByID(
			r.Context(),
			request.SettlementID,
		)

	if err == nil {
		// transferred is the normal successful result.
		//
		// TransferByID may also return an already-transferred Settlement as an
		// idempotent success.
		w.WriteHeader(
			http.StatusNoContent,
		)

		return
	}

	h.writeProcessingResult(
		w,
		settlement,
		err,
	)
}

// ============================================================
// Dispatch Due
// ============================================================

// DispatchDue reconciles seller-side Settlements that may have lost their
// original Cloud Task.
//
// The endpoint is intended for periodic execution by Cloud Scheduler.
//
// It enqueues:
//
// - ready
// - failed_retryable
// - stale transferring
//
// Settlements through SettlementUsecase.DispatchDue.
//
// Unlike Process, this endpoint does not require X-CloudTasks-TaskName because
// the caller is expected to be Cloud Scheduler rather than Cloud Tasks.
//
// OIDC audience and service-account validation are still required.
func (h *SettlementTaskHandler) DispatchDue(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set(
			"Allow",
			http.MethodPost,
		)

		writeSettlementTaskError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
		)

		return
	}

	if h == nil ||
		h.settlementUC == nil {
		writeSettlementTaskError(
			w,
			http.StatusServiceUnavailable,
			"settlement_usecase_unavailable",
		)

		return
	}

	if h.settlementQueue == nil {
		writeSettlementTaskError(
			w,
			http.StatusServiceUnavailable,
			"settlement_queue_unavailable",
		)

		return
	}

	if err :=
		h.authorizeInternalRequest(
			r,
			false,
		); err != nil {
		h.writeAuthorizationError(
			w,
			err,
		)

		return
	}

	var request dispatchDueSettlementTaskRequest

	if err :=
		decodeOptionalSettlementTaskJSON(
			w,
			r,
			&request,
		); err != nil {
		writeSettlementTaskError(
			w,
			http.StatusBadRequest,
			"invalid_json_body",
		)

		return
	}

	enqueuedCount, err :=
		h.settlementUC.DispatchDue(
			r.Context(),
			h.settlementQueue,
			request.Limit,
		)

	if err != nil {
		writeSettlementTaskJSON(
			w,
			http.StatusInternalServerError,
			dispatchDueSettlementTaskResponse{
				Enqueued: enqueuedCount,

				Error: "settlement_dispatch_due_failed",
			},
		)

		return
	}

	writeSettlementTaskJSON(
		w,
		http.StatusOK,
		dispatchDueSettlementTaskResponse{
			Enqueued: enqueuedCount,
		},
	)
}

// ============================================================
// Processing result
// ============================================================

func (h *SettlementTaskHandler) writeProcessingResult(
	w http.ResponseWriter,
	settlement settlementdom.Settlement,
	err error,
) {
	// The Stripe request failed with a retryable error and
	// SettlementUsecase successfully persisted failed_retryable.
	//
	// Return non-2xx so Cloud Tasks retries the same task.
	if settlement.Status ==
		settlementdom.StatusFailedRetryable {
		writeSettlementTaskError(
			w,
			http.StatusInternalServerError,
			"settlement_transfer_retryable",
		)

		return
	}

	// A Settlement left in transferring represents an uncertain state.
	//
	// Do not acknowledge the Cloud Task. Acknowledging here could permanently
	// lose a settlement after a process/network failure.
	if settlement.Status ==
		settlementdom.StatusTransferring {
		writeSettlementTaskError(
			w,
			http.StatusInternalServerError,
			"settlement_transfer_in_progress",
		)

		return
	}

	// A terminal Stripe failure has already been persisted.
	//
	// Automatic retry must stop. Operational recovery should explicitly move
	// the Settlement into a retryable state after the underlying issue is
	// corrected.
	if settlement.Status ==
		settlementdom.StatusFailed {
		w.WriteHeader(
			http.StatusNoContent,
		)

		return
	}

	// These states require no further Stripe Transfer.
	switch settlement.Status {
	case settlementdom.StatusTransferred,
		settlementdom.StatusCanceled,
		settlementdom.StatusReversed:
		w.WriteHeader(
			http.StatusNoContent,
		)

		return
	}

	// TransferByID can return ErrSettlementTransferNotReady when another worker
	// already holds the Settlement claim or the Settlement is not currently
	// eligible to transfer.
	//
	// For an unknown/non-terminal state we intentionally retain the task.
	if errors.Is(
		err,
		uc.ErrSettlementTransferNotReady,
	) {
		writeSettlementTaskError(
			w,
			http.StatusInternalServerError,
			"settlement_transfer_not_ready",
		)

		return
	}

	// Repository, Stripe transport, persistence, or other infrastructure
	// failures are retryable from the HTTP worker perspective.
	//
	// The same deterministic Stripe Idempotency-Key will be used on the next
	// attempt, preventing an uncertain Stripe response from creating a second
	// Transfer.
	writeSettlementTaskError(
		w,
		http.StatusInternalServerError,
		"settlement_processing_failed",
	)
}

// ============================================================
// Authorization
// ============================================================

// authorizeInternalRequest validates:
//
//  1. Authorization: Bearer <OIDC token>
//  2. token audience
//  3. service account email
//  4. email_verified
//  5. X-CloudTasks-TaskName when requireCloudTaskHeader=true
//
// INTERNAL_BASE_URL or CLOUD_TASKS_AUDIENCE must match the audience configured
// by SettlementQueue and the Cloud Scheduler OIDC request.
func (h *SettlementTaskHandler) authorizeInternalRequest(
	r *http.Request,
	requireCloudTaskHeader bool,
) error {
	if h == nil {
		return errSettlementTaskAuthNotConfigured
	}

	audience :=
		h.audience

	serviceAccountEmail :=
		strings.ToLower(
			h.serviceAccountEmail,
		)

	if audience == "" ||
		serviceAccountEmail == "" {
		return errSettlementTaskAuthNotConfigured
	}

	rawToken, ok :=
		settlementTaskBearerToken(
			r.Header.Get(
				"Authorization",
			),
		)
	if !ok {
		return errSettlementTaskUnauthorized
	}

	payload, err := idtoken.Validate(
		r.Context(),
		rawToken,
		audience,
	)
	if err != nil ||
		payload == nil {
		return errSettlementTaskUnauthorized
	}

	tokenEmail, _ :=
		payload.Claims["email"].(string)

	tokenEmail =
		strings.ToLower(
			tokenEmail,
		)

	if tokenEmail == "" ||
		tokenEmail !=
			serviceAccountEmail {
		return errSettlementTaskForbidden
	}

	if !settlementTaskEmailVerified(
		payload.Claims["email_verified"],
	) {
		return errSettlementTaskForbidden
	}

	if requireCloudTaskHeader &&
		r.Header.Get(
			"X-CloudTasks-TaskName",
		) == "" {
		return errSettlementTaskForbidden
	}

	return nil
}

func (h *SettlementTaskHandler) writeAuthorizationError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		errSettlementTaskAuthNotConfigured,
	):
		writeSettlementTaskError(
			w,
			http.StatusServiceUnavailable,
			"settlement_auth_unavailable",
		)

	case errors.Is(
		err,
		errSettlementTaskForbidden,
	):
		writeSettlementTaskError(
			w,
			http.StatusForbidden,
			"forbidden",
		)

	default:
		writeSettlementTaskError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
	}
}

func settlementTaskBearerToken(
	authorizationHeader string,
) (string, bool) {
	parts :=
		strings.Fields(
			authorizationHeader,
		)

	if len(parts) != 2 {
		return "", false
	}

	if !strings.EqualFold(
		parts[0],
		"Bearer",
	) {
		return "", false
	}

	token :=
		parts[1]

	if token == "" {
		return "", false
	}

	return token, true
}

func settlementTaskEmailVerified(
	value any,
) bool {
	switch verified :=
		value.(type) {
	case bool:
		return verified

	case string:
		return strings.EqualFold(
			verified,
			"true",
		)

	default:
		return false
	}
}

// ============================================================
// JSON
// ============================================================

func decodeRequiredSettlementTaskJSON(
	w http.ResponseWriter,
	r *http.Request,
	destination any,
) error {
	if r.Body == nil {
		return io.EOF
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxSettlementTaskRequestBodyBytes,
	)

	decoder :=
		json.NewDecoder(
			r.Body,
		)

	decoder.DisallowUnknownFields()

	if err :=
		decoder.Decode(
			destination,
		); err != nil {
		return err
	}

	return ensureSingleSettlementTaskJSONValue(
		decoder,
	)
}

func decodeOptionalSettlementTaskJSON(
	w http.ResponseWriter,
	r *http.Request,
	destination any,
) error {
	if r.Body == nil {
		return nil
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxSettlementTaskRequestBodyBytes,
	)

	decoder :=
		json.NewDecoder(
			r.Body,
		)

	decoder.DisallowUnknownFields()

	if err :=
		decoder.Decode(
			destination,
		); err != nil {
		if errors.Is(
			err,
			io.EOF,
		) {
			return nil
		}

		return err
	}

	return ensureSingleSettlementTaskJSONValue(
		decoder,
	)
}

func ensureSingleSettlementTaskJSONValue(
	decoder *json.Decoder,
) error {
	var extra any

	err :=
		decoder.Decode(
			&extra,
		)

	if errors.Is(
		err,
		io.EOF,
	) {
		return nil
	}

	if err == nil {
		return errors.New(
			"multiple JSON values are not allowed",
		)
	}

	return err
}

// ============================================================
// Response
// ============================================================

func writeSettlementTaskError(
	w http.ResponseWriter,
	statusCode int,
	message string,
) {
	writeSettlementTaskJSON(
		w,
		statusCode,
		settlementTaskErrorResponse{
			Error: message,
		},
	)
}

func writeSettlementTaskJSON(
	w http.ResponseWriter,
	statusCode int,
	value any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	w.WriteHeader(
		statusCode,
	)

	_ = json.NewEncoder(
		w,
	).Encode(
		value,
	)
}

// ============================================================
// Environment
// ============================================================

func firstNonEmptySettlementTaskEnvironmentValue(
	keys ...string,
) string {
	for _, key := range keys {
		if key == "" {
			continue
		}

		value :=
			os.Getenv(
				key,
			)

		if value != "" {
			return value
		}
	}

	return ""
}
