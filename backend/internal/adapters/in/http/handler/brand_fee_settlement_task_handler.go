// backend/internal/adapters/in/http/handler/brand_fee_settlement_task_handler.go
package internalHandler

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"google.golang.org/api/idtoken"

	uc "narratives/internal/application/usecase"
	brandfeesettlementdom "narratives/internal/domain/brandFeeSettlement"
)

const (
	envBrandFeeSettlementTaskCloudTasksAudience = "CLOUD_TASKS_AUDIENCE"

	envBrandFeeSettlementTaskCloudTasksServiceAccount = "CLOUD_TASKS_SERVICE_ACCOUNT"

	envBrandFeeSettlementTaskInternalBaseURL = "INTERNAL_BASE_URL"

	envBrandFeeSettlementTaskSelfBaseURL = "SELF_BASE_URL"

	brandFeeSettlementProcessPath = "/internal/brand-fee-settlements/process"

	brandFeeSettlementDispatchDuePath = "/internal/brand-fee-settlements/dispatch-due"
)

var (
	errBrandFeeSettlementTaskAuthNotConfigured = errors.New(
		"brand fee settlement task authentication is not configured",
	)

	errBrandFeeSettlementTaskUnauthorized = errors.New(
		"brand fee settlement task request is unauthorized",
	)

	errBrandFeeSettlementTaskForbidden = errors.New(
		"brand fee settlement task request is forbidden",
	)
)

// ============================================================
// Usecase Port
// ============================================================

// BrandFeeSettlementTaskUsecase is the minimal application contract required
// by the Brand fee Cloud Tasks worker and reconciliation endpoint.
//
// *usecase.BrandFeeSettlementTransferUsecase satisfies this interface.
type BrandFeeSettlementTaskUsecase interface {
	TransferByID(
		ctx context.Context,
		brandFeeSettlementID string,
	) (brandfeesettlementdom.BrandFeeSettlement, error)

	DispatchDue(
		ctx context.Context,
		queue uc.BrandFeeSettlementTransferQueue,
		limit int,
	) (int, error)
}

// ============================================================
// Handler
// ============================================================

// BrandFeeSettlementTaskHandler processes Stripe Connect Brand fee transfers.
//
// Cloud Tasks:
//
//	POST /internal/brand-fee-settlements/process
//
// Cloud Scheduler:
//
//	POST /internal/brand-fee-settlements/dispatch-due
//
// Process body:
//
//	{
//	  "brandFeeSettlementId": "..."
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
// The process request never contains BrandFeeAmount, destination Stripe Account,
// Charge ID, TransferGroup, Brand identity, or any other financial data. Those
// values are always loaded from the authoritative BrandFeeSettlement document.
type BrandFeeSettlementTaskHandler struct {
	brandFeeSettlementUC BrandFeeSettlementTaskUsecase

	brandFeeSettlementQueue uc.BrandFeeSettlementTransferQueue

	audience string

	serviceAccountEmail string
}

type processBrandFeeSettlementTaskRequest struct {
	BrandFeeSettlementID string `json:"brandFeeSettlementId"`
}

type dispatchDueBrandFeeSettlementTaskRequest struct {
	Limit int `json:"limit"`
}

type dispatchDueBrandFeeSettlementTaskResponse struct {
	Enqueued int `json:"enqueued"`

	Error string `json:"error,omitempty"`
}

type brandFeeSettlementTaskErrorResponse struct {
	Error string `json:"error"`
}

// NewBrandFeeSettlementTaskHandler creates the internal Brand fee Settlement
// handler.
func NewBrandFeeSettlementTaskHandler(
	brandFeeSettlementUC BrandFeeSettlementTaskUsecase,
	brandFeeSettlementQueue uc.BrandFeeSettlementTransferQueue,
) *BrandFeeSettlementTaskHandler {
	audience := firstNonEmptyBrandFeeSettlementTaskEnvironmentValue(
		envBrandFeeSettlementTaskCloudTasksAudience,
		envBrandFeeSettlementTaskInternalBaseURL,
		envBrandFeeSettlementTaskSelfBaseURL,
	)

	serviceAccountEmail := firstNonEmptyBrandFeeSettlementTaskEnvironmentValue(
		envBrandFeeSettlementTaskCloudTasksServiceAccount,
	)

	return &BrandFeeSettlementTaskHandler{
		brandFeeSettlementUC:    brandFeeSettlementUC,
		brandFeeSettlementQueue: brandFeeSettlementQueue,
		audience:                strings.TrimRight(audience, "/"),
		serviceAccountEmail:     strings.ToLower(serviceAccountEmail),
	}
}

func (h *BrandFeeSettlementTaskHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	path := strings.TrimRight(
		r.URL.Path,
		"/",
	)

	switch path {
	case brandFeeSettlementProcessPath:
		h.Process(w, r)

	case brandFeeSettlementDispatchDuePath:
		h.DispatchDue(w, r)

	default:
		writeBrandFeeSettlementTaskError(
			w,
			http.StatusNotFound,
			"not_found",
		)
	}
}

// ============================================================
// Process
// ============================================================

// Process executes one productBlueprint Brand fee Stripe Connect Transfer.
//
// Successful and terminal states return 204 so Cloud Tasks acknowledges the
// task.
//
// Retryable or uncertain states return 500 so Cloud Tasks keeps retrying.
//
// Financial idempotency is provided by:
//
//  1. BrandFeeSettlementRepositoryFS.ClaimForTransfer.
//  2. deterministic Stripe Idempotency-Key.
//  3. CompleteTransfer persisting the resulting tr_xxx.
func (h *BrandFeeSettlementTaskHandler) Process(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set(
			"Allow",
			http.MethodPost,
		)

		writeBrandFeeSettlementTaskError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
		)

		return
	}

	if h == nil ||
		h.brandFeeSettlementUC == nil {
		writeBrandFeeSettlementTaskError(
			w,
			http.StatusServiceUnavailable,
			"brand_fee_settlement_usecase_unavailable",
		)

		return
	}

	if err := h.authorizeInternalRequest(
		r,
		true,
	); err != nil {
		h.writeAuthorizationError(
			w,
			err,
		)

		return
	}

	var request processBrandFeeSettlementTaskRequest

	if err := decodeRequiredSettlementTaskJSON(
		w,
		r,
		&request,
	); err != nil {
		writeBrandFeeSettlementTaskError(
			w,
			http.StatusBadRequest,
			"invalid_json_body",
		)

		return
	}

	if request.BrandFeeSettlementID == "" ||
		strings.Contains(
			request.BrandFeeSettlementID,
			"/",
		) {
		writeBrandFeeSettlementTaskError(
			w,
			http.StatusBadRequest,
			"brand_fee_settlement_id_required",
		)

		return
	}

	brandFeeSettlement, err := h.brandFeeSettlementUC.TransferByID(
		r.Context(),
		request.BrandFeeSettlementID,
	)

	if err == nil {
		// transferred is the normal successful result.
		//
		// TransferByID may also return an already-transferred
		// BrandFeeSettlement as an idempotent success.
		w.WriteHeader(
			http.StatusNoContent,
		)

		return
	}

	h.writeProcessingResult(
		w,
		brandFeeSettlement,
		err,
	)
}

// ============================================================
// Dispatch Due
// ============================================================

// DispatchDue reconciles BrandFeeSettlements that may have lost their original
// Cloud Task or require retry recovery.
//
// The endpoint is intended for periodic execution by Cloud Scheduler.
//
// It enqueues:
//
//   - ready
//   - failed_retryable
//   - stale transferring
//
// pending is intentionally excluded.
//
// Unlike Process, this endpoint does not require X-CloudTasks-TaskName because
// the caller is expected to be Cloud Scheduler rather than Cloud Tasks.
//
// OIDC audience and service-account validation are still required.
func (h *BrandFeeSettlementTaskHandler) DispatchDue(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set(
			"Allow",
			http.MethodPost,
		)

		writeBrandFeeSettlementTaskError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
		)

		return
	}

	if h == nil ||
		h.brandFeeSettlementUC == nil {
		writeBrandFeeSettlementTaskError(
			w,
			http.StatusServiceUnavailable,
			"brand_fee_settlement_usecase_unavailable",
		)

		return
	}

	if h.brandFeeSettlementQueue == nil {
		writeBrandFeeSettlementTaskError(
			w,
			http.StatusServiceUnavailable,
			"brand_fee_settlement_queue_unavailable",
		)

		return
	}

	if err := h.authorizeInternalRequest(
		r,
		false,
	); err != nil {
		h.writeAuthorizationError(
			w,
			err,
		)

		return
	}

	var request dispatchDueBrandFeeSettlementTaskRequest

	if err := decodeOptionalSettlementTaskJSON(
		w,
		r,
		&request,
	); err != nil {
		writeBrandFeeSettlementTaskError(
			w,
			http.StatusBadRequest,
			"invalid_json_body",
		)

		return
	}

	enqueuedCount, err := h.brandFeeSettlementUC.DispatchDue(
		r.Context(),
		h.brandFeeSettlementQueue,
		request.Limit,
	)

	if err != nil {
		writeBrandFeeSettlementTaskJSON(
			w,
			http.StatusInternalServerError,
			dispatchDueBrandFeeSettlementTaskResponse{
				Enqueued: enqueuedCount,
				Error:    "brand_fee_settlement_dispatch_due_failed",
			},
		)

		return
	}

	writeBrandFeeSettlementTaskJSON(
		w,
		http.StatusOK,
		dispatchDueBrandFeeSettlementTaskResponse{
			Enqueued: enqueuedCount,
		},
	)
}

// ============================================================
// Processing result
// ============================================================

func (h *BrandFeeSettlementTaskHandler) writeProcessingResult(
	w http.ResponseWriter,
	brandFeeSettlement brandfeesettlementdom.BrandFeeSettlement,
	err error,
) {
	// Stripe failed with a retryable error and failed_retryable was persisted.
	// Return non-2xx so Cloud Tasks retries the same deterministic task.
	if brandFeeSettlement.Status ==
		brandfeesettlementdom.StatusFailedRetryable {
		writeBrandFeeSettlementTaskError(
			w,
			http.StatusInternalServerError,
			"brand_fee_settlement_transfer_retryable",
		)

		return
	}

	// transferring represents an uncertain execution state. Do not acknowledge
	// the task because Stripe may already have accepted the Transfer.
	if brandFeeSettlement.Status ==
		brandfeesettlementdom.StatusTransferring {
		writeBrandFeeSettlementTaskError(
			w,
			http.StatusInternalServerError,
			"brand_fee_settlement_transfer_in_progress",
		)

		return
	}

	// A terminal Stripe failure has already been persisted. Automatic Cloud
	// Tasks retry must stop.
	if brandFeeSettlement.Status ==
		brandfeesettlementdom.StatusFailed {
		w.WriteHeader(
			http.StatusNoContent,
		)

		return
	}

	// These states require no further Stripe Transfer.
	switch brandFeeSettlement.Status {
	case brandfeesettlementdom.StatusTransferred,
		brandfeesettlementdom.StatusCanceled,
		brandfeesettlementdom.StatusReversed:
		w.WriteHeader(
			http.StatusNoContent,
		)

		return
	}

	// Another worker may already hold the claim, or the record may not currently
	// be eligible for transfer. Keep the task retryable instead of acknowledging
	// an unknown/non-terminal state.
	if errors.Is(
		err,
		uc.ErrBrandFeeSettlementTransferNotReady,
	) {
		writeBrandFeeSettlementTaskError(
			w,
			http.StatusInternalServerError,
			"brand_fee_settlement_transfer_not_ready",
		)

		return
	}

	// Repository, Stripe transport, persistence, or other infrastructure errors
	// are retryable from the HTTP worker perspective. TransferByID will reuse
	// the same deterministic Stripe Idempotency-Key on the next attempt.
	writeBrandFeeSettlementTaskError(
		w,
		http.StatusInternalServerError,
		"brand_fee_settlement_processing_failed",
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
// by BrandFeeSettlementQueue and the Cloud Scheduler OIDC request.
func (h *BrandFeeSettlementTaskHandler) authorizeInternalRequest(
	r *http.Request,
	requireCloudTaskHeader bool,
) error {
	if h == nil {
		return errBrandFeeSettlementTaskAuthNotConfigured
	}

	audience := h.audience
	serviceAccountEmail := strings.ToLower(
		h.serviceAccountEmail,
	)

	if audience == "" ||
		serviceAccountEmail == "" {
		return errBrandFeeSettlementTaskAuthNotConfigured
	}

	rawToken, ok := settlementTaskBearerToken(
		r.Header.Get(
			"Authorization",
		),
	)
	if !ok {
		return errBrandFeeSettlementTaskUnauthorized
	}

	payload, err := idtoken.Validate(
		r.Context(),
		rawToken,
		audience,
	)
	if err != nil ||
		payload == nil {
		return errBrandFeeSettlementTaskUnauthorized
	}

	tokenEmail, _ := payload.Claims["email"].(string)

	tokenEmail = strings.ToLower(
		tokenEmail,
	)

	if tokenEmail == "" ||
		tokenEmail != serviceAccountEmail {
		return errBrandFeeSettlementTaskForbidden
	}

	if !settlementTaskEmailVerified(
		payload.Claims["email_verified"],
	) {
		return errBrandFeeSettlementTaskForbidden
	}

	if requireCloudTaskHeader &&
		r.Header.Get(
			"X-CloudTasks-TaskName",
		) == "" {
		return errBrandFeeSettlementTaskForbidden
	}

	return nil
}

func (h *BrandFeeSettlementTaskHandler) writeAuthorizationError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		errBrandFeeSettlementTaskAuthNotConfigured,
	):
		writeBrandFeeSettlementTaskError(
			w,
			http.StatusServiceUnavailable,
			"brand_fee_settlement_auth_unavailable",
		)

	case errors.Is(
		err,
		errBrandFeeSettlementTaskForbidden,
	):
		writeBrandFeeSettlementTaskError(
			w,
			http.StatusForbidden,
			"forbidden",
		)

	default:
		writeBrandFeeSettlementTaskError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
	}
}

// ============================================================
// Response
// ============================================================

func writeBrandFeeSettlementTaskError(
	w http.ResponseWriter,
	statusCode int,
	message string,
) {
	writeBrandFeeSettlementTaskJSON(
		w,
		statusCode,
		brandFeeSettlementTaskErrorResponse{
			Error: message,
		},
	)
}

func writeBrandFeeSettlementTaskJSON(
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

	// Reuse the existing strict Settlement task JSON response encoder so both
	// internal financial workers follow the same response behavior.
	writeBrandFeeSettlementTaskJSONBody(
		w,
		value,
	)
}

func writeBrandFeeSettlementTaskJSONBody(
	w http.ResponseWriter,
	value any,
) {
	// writeSettlementTaskJSON cannot be used here because it also writes the
	// HTTP status. The encoder itself is intentionally kept local.
	_ = encodeBrandFeeSettlementTaskJSON(
		w,
		value,
	)
}

// ============================================================
// Environment
// ============================================================

func firstNonEmptyBrandFeeSettlementTaskEnvironmentValue(
	keys ...string,
) string {
	for _, key := range keys {
		if key == "" {
			continue
		}

		value := os.Getenv(
			key,
		)
		if value != "" {
			return value
		}
	}

	return ""
}
