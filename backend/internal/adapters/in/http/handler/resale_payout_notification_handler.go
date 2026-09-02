// backend/internal/adapters/in/http/handler/resale_payout_notification_handler.go
package internalHandler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"google.golang.org/api/idtoken"

	uc "narratives/internal/application/usecase"
)

const (
	envResalePayoutNotificationCloudTasksAudience       = "CLOUD_TASKS_AUDIENCE"
	envResalePayoutNotificationCloudTasksServiceAccount = "CLOUD_TASKS_SERVICE_ACCOUNT"
	envResalePayoutNotificationInternalBaseURL          = "INTERNAL_BASE_URL"
	envResalePayoutNotificationSelfBaseURL              = "SELF_BASE_URL"

	maxResalePayoutNotificationRequestBodyBytes int64 = 64 * 1024
)

var (
	errResalePayoutNotificationAuthNotConfigured = errors.New(
		"resale payout notification authentication is not configured",
	)
	errResalePayoutNotificationUnauthorized = errors.New(
		"resale payout notification request is unauthorized",
	)
	errResalePayoutNotificationForbidden = errors.New(
		"resale payout notification request is forbidden",
	)
)

type ResalePayoutNotificationHandler struct {
	notificationUC      uc.ResalePayoutNotificationUsecasePort
	audience            string
	serviceAccountEmail string
}

type processResalePayoutNotificationRequest struct {
	DeliveryID string `json:"deliveryId"`
}

type dispatchResalePayoutNotificationsRequest struct {
	Limit int `json:"limit"`
}

type dispatchResalePayoutNotificationsResponse struct {
	Enqueued int `json:"enqueued"`
}

type resalePayoutNotificationErrorResponse struct {
	Error    string `json:"error"`
	Enqueued int    `json:"enqueued,omitempty"`
}

func NewResalePayoutNotificationHandler(
	notificationUC uc.ResalePayoutNotificationUsecasePort,
) *ResalePayoutNotificationHandler {
	audience := firstNonEmptyResalePayoutNotificationEnvironmentValue(
		envResalePayoutNotificationCloudTasksAudience,
		envResalePayoutNotificationInternalBaseURL,
		envResalePayoutNotificationSelfBaseURL,
	)

	serviceAccountEmail := firstNonEmptyResalePayoutNotificationEnvironmentValue(
		envResalePayoutNotificationCloudTasksServiceAccount,
	)

	return &ResalePayoutNotificationHandler{
		notificationUC: notificationUC,
		audience:       strings.TrimRight(audience, "/"),
		serviceAccountEmail: strings.ToLower(
			strings.TrimSpace(serviceAccountEmail),
		),
	}
}

func (h *ResalePayoutNotificationHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.Process(w, r)
}

// Process processes one payout-notification delivery requested by Cloud Tasks.
// Successful, duplicate, and already-processed deliveries return 204.
func (h *ResalePayoutNotificationHandler) Process(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeResalePayoutNotificationError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			0,
		)
		return
	}

	if h == nil || h.notificationUC == nil {
		writeResalePayoutNotificationError(
			w,
			http.StatusServiceUnavailable,
			"resale_payout_notification_usecase_unavailable",
			0,
		)
		return
	}

	if err := h.authorizeInternalRequest(r, true); err != nil {
		h.writeAuthorizationError(w, err)
		return
	}

	var request processResalePayoutNotificationRequest

	if err := decodeRequiredResalePayoutNotificationJSON(
		w,
		r,
		&request,
	); err != nil {
		writeResalePayoutNotificationError(
			w,
			http.StatusBadRequest,
			"invalid_json_body",
			0,
		)
		return
	}

	request.DeliveryID = strings.TrimSpace(request.DeliveryID)
	if request.DeliveryID == "" {
		writeResalePayoutNotificationError(
			w,
			http.StatusBadRequest,
			"delivery_id_required",
			0,
		)
		return
	}

	if err := h.notificationUC.Process(
		r.Context(),
		request.DeliveryID,
	); err != nil {
		writeResalePayoutNotificationError(
			w,
			http.StatusInternalServerError,
			"resale_payout_notification_processing_failed",
			0,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DispatchDue enqueues due payout-notification deliveries into Cloud Tasks.
// This endpoint is intended to be called by Cloud Scheduler with OIDC.
//
// Body is optional:
//
//	{
//	  "limit": 50
//	}
func (h *ResalePayoutNotificationHandler) DispatchDue(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeResalePayoutNotificationError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			0,
		)
		return
	}

	if h == nil || h.notificationUC == nil {
		writeResalePayoutNotificationError(
			w,
			http.StatusServiceUnavailable,
			"resale_payout_notification_usecase_unavailable",
			0,
		)
		return
	}

	if err := h.authorizeInternalRequest(r, false); err != nil {
		h.writeAuthorizationError(w, err)
		return
	}

	var request dispatchResalePayoutNotificationsRequest

	if err := decodeOptionalResalePayoutNotificationJSON(
		w,
		r,
		&request,
	); err != nil {
		writeResalePayoutNotificationError(
			w,
			http.StatusBadRequest,
			"invalid_json_body",
			0,
		)
		return
	}

	if request.Limit < 0 {
		writeResalePayoutNotificationError(
			w,
			http.StatusBadRequest,
			"limit_must_not_be_negative",
			0,
		)
		return
	}

	enqueuedCount, err := h.notificationUC.DispatchDue(
		r.Context(),
		request.Limit,
	)
	if err != nil {
		writeResalePayoutNotificationError(
			w,
			http.StatusInternalServerError,
			"resale_payout_notification_dispatch_failed",
			enqueuedCount,
		)
		return
	}

	writeResalePayoutNotificationJSON(
		w,
		http.StatusOK,
		dispatchResalePayoutNotificationsResponse{
			Enqueued: enqueuedCount,
		},
	)
}

func (h *ResalePayoutNotificationHandler) authorizeInternalRequest(
	r *http.Request,
	requireCloudTasksHeader bool,
) error {
	if h == nil {
		return errResalePayoutNotificationAuthNotConfigured
	}

	audience := strings.TrimSpace(h.audience)
	serviceAccountEmail := strings.ToLower(
		strings.TrimSpace(h.serviceAccountEmail),
	)

	if audience == "" || serviceAccountEmail == "" {
		return errResalePayoutNotificationAuthNotConfigured
	}

	rawToken, ok := resalePayoutNotificationBearerToken(
		r.Header.Get("Authorization"),
	)
	if !ok {
		return errResalePayoutNotificationUnauthorized
	}

	payload, err := idtoken.Validate(
		r.Context(),
		rawToken,
		audience,
	)
	if err != nil || payload == nil {
		return errResalePayoutNotificationUnauthorized
	}

	tokenEmail, _ := payload.Claims["email"].(string)
	tokenEmail = strings.ToLower(
		strings.TrimSpace(tokenEmail),
	)

	if tokenEmail == "" || tokenEmail != serviceAccountEmail {
		return errResalePayoutNotificationForbidden
	}

	if !resalePayoutNotificationEmailVerified(
		payload.Claims["email_verified"],
	) {
		return errResalePayoutNotificationForbidden
	}

	if requireCloudTasksHeader &&
		strings.TrimSpace(
			r.Header.Get("X-CloudTasks-TaskName"),
		) == "" {
		return errResalePayoutNotificationForbidden
	}

	return nil
}

func (h *ResalePayoutNotificationHandler) writeAuthorizationError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		errResalePayoutNotificationAuthNotConfigured,
	):
		writeResalePayoutNotificationError(
			w,
			http.StatusServiceUnavailable,
			"resale_payout_notification_auth_unavailable",
			0,
		)

	case errors.Is(
		err,
		errResalePayoutNotificationForbidden,
	):
		writeResalePayoutNotificationError(
			w,
			http.StatusForbidden,
			"forbidden",
			0,
		)

	default:
		writeResalePayoutNotificationError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			0,
		)
	}
}

func resalePayoutNotificationBearerToken(
	authorizationHeader string,
) (string, bool) {
	parts := strings.Fields(
		strings.TrimSpace(authorizationHeader),
	)

	if len(parts) != 2 {
		return "", false
	}

	if !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}

	return token, true
}

func resalePayoutNotificationEmailVerified(
	value any,
) bool {
	switch verified := value.(type) {
	case bool:
		return verified

	case string:
		return strings.EqualFold(
			strings.TrimSpace(verified),
			"true",
		)

	default:
		return false
	}
}

func decodeRequiredResalePayoutNotificationJSON(
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
		maxResalePayoutNotificationRequestBodyBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	return ensureSingleResalePayoutNotificationJSONValue(
		decoder,
	)
}

func decodeOptionalResalePayoutNotificationJSON(
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
		maxResalePayoutNotificationRequestBodyBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}

		return err
	}

	return ensureSingleResalePayoutNotificationJSONValue(
		decoder,
	)
}

func ensureSingleResalePayoutNotificationJSONValue(
	decoder *json.Decoder,
) error {
	var extra any

	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}

	if err == nil {
		return errors.New(
			"multiple JSON values are not allowed",
		)
	}

	return err
}

func writeResalePayoutNotificationError(
	w http.ResponseWriter,
	statusCode int,
	message string,
	enqueuedCount int,
) {
	writeResalePayoutNotificationJSON(
		w,
		statusCode,
		resalePayoutNotificationErrorResponse{
			Error:    message,
			Enqueued: enqueuedCount,
		},
	)
}

func writeResalePayoutNotificationJSON(
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

	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}

func firstNonEmptyResalePayoutNotificationEnvironmentValue(
	keys ...string,
) string {
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		value := strings.TrimSpace(
			os.Getenv(key),
		)
		if value != "" {
			return value
		}
	}

	return ""
}
