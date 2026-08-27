// backend/internal/adapters/in/http/handler/refund_completion_notification_handler.go
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
	envRefundCompletionNotificationCloudTasksAudience       = "CLOUD_TASKS_AUDIENCE"
	envRefundCompletionNotificationCloudTasksServiceAccount = "CLOUD_TASKS_SERVICE_ACCOUNT"
	envRefundCompletionNotificationInternalBaseURL          = "INTERNAL_BASE_URL"
	envRefundCompletionNotificationSelfBaseURL              = "SELF_BASE_URL"

	maxRefundCompletionNotificationRequestBodyBytes int64 = 64 * 1024
)

var (
	errRefundCompletionNotificationAuthNotConfigured = errors.New(
		"refund completion notification authentication is not configured",
	)
	errRefundCompletionNotificationUnauthorized = errors.New(
		"refund completion notification request is unauthorized",
	)
	errRefundCompletionNotificationForbidden = errors.New(
		"refund completion notification request is forbidden",
	)
)

type RefundCompletionNotificationHandler struct {
	notificationUC      uc.RefundCompletionNotificationUsecasePort
	audience            string
	serviceAccountEmail string
}

type processRefundCompletionNotificationRequest struct {
	DeliveryID string `json:"deliveryId"`
}

type dispatchRefundCompletionNotificationsRequest struct {
	Limit int `json:"limit"`
}

type dispatchRefundCompletionNotificationsResponse struct {
	Enqueued int `json:"enqueued"`
}

type refundCompletionNotificationErrorResponse struct {
	Error    string `json:"error"`
	Enqueued int    `json:"enqueued,omitempty"`
}

func NewRefundCompletionNotificationHandler(
	notificationUC uc.RefundCompletionNotificationUsecasePort,
) *RefundCompletionNotificationHandler {
	audience := firstNonEmptyRefundCompletionNotificationEnvironmentValue(
		envRefundCompletionNotificationCloudTasksAudience,
		envRefundCompletionNotificationInternalBaseURL,
		envRefundCompletionNotificationSelfBaseURL,
	)

	serviceAccountEmail := firstNonEmptyRefundCompletionNotificationEnvironmentValue(
		envRefundCompletionNotificationCloudTasksServiceAccount,
	)

	return &RefundCompletionNotificationHandler{
		notificationUC: notificationUC,
		audience:       strings.TrimRight(audience, "/"),
		serviceAccountEmail: strings.ToLower(
			strings.TrimSpace(serviceAccountEmail),
		),
	}
}

func (h *RefundCompletionNotificationHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.Process(w, r)
}

// ProcessはCloud Tasksから渡されたdelivery IDを処理します。
// 正常終了、重複task、処理済みtaskは204を返します。
func (h *RefundCompletionNotificationHandler) Process(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeRefundCompletionNotificationError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			0,
		)
		return
	}

	if h == nil || h.notificationUC == nil {
		writeRefundCompletionNotificationError(
			w,
			http.StatusServiceUnavailable,
			"refund_completion_notification_usecase_unavailable",
			0,
		)
		return
	}

	if err := h.authorizeInternalRequest(r, true); err != nil {
		h.writeAuthorizationError(w, err)
		return
	}

	var request processRefundCompletionNotificationRequest

	if err := decodeRequiredRefundCompletionNotificationJSON(
		w,
		r,
		&request,
	); err != nil {
		writeRefundCompletionNotificationError(
			w,
			http.StatusBadRequest,
			"invalid_json_body",
			0,
		)
		return
	}

	request.DeliveryID = strings.TrimSpace(request.DeliveryID)
	if request.DeliveryID == "" {
		writeRefundCompletionNotificationError(
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
		writeRefundCompletionNotificationError(
			w,
			http.StatusInternalServerError,
			"refund_completion_notification_processing_failed",
			0,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DispatchDueは送信時刻を迎えたdeliveryをCloud Tasksへ投入します。
// Cloud SchedulerなどからOIDC付きで呼び出すことを想定しています。
// bodyは省略可能です。
//
//	{
//	  "limit": 50
//	}
func (h *RefundCompletionNotificationHandler) DispatchDue(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeRefundCompletionNotificationError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			0,
		)
		return
	}

	if h == nil || h.notificationUC == nil {
		writeRefundCompletionNotificationError(
			w,
			http.StatusServiceUnavailable,
			"refund_completion_notification_usecase_unavailable",
			0,
		)
		return
	}

	if err := h.authorizeInternalRequest(r, false); err != nil {
		h.writeAuthorizationError(w, err)
		return
	}

	var request dispatchRefundCompletionNotificationsRequest

	if err := decodeOptionalRefundCompletionNotificationJSON(
		w,
		r,
		&request,
	); err != nil {
		writeRefundCompletionNotificationError(
			w,
			http.StatusBadRequest,
			"invalid_json_body",
			0,
		)
		return
	}

	if request.Limit < 0 {
		writeRefundCompletionNotificationError(
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
		writeRefundCompletionNotificationError(
			w,
			http.StatusInternalServerError,
			"refund_completion_notification_dispatch_failed",
			enqueuedCount,
		)
		return
	}

	writeRefundCompletionNotificationJSON(
		w,
		http.StatusOK,
		dispatchRefundCompletionNotificationsResponse{
			Enqueued: enqueuedCount,
		},
	)
}

func (h *RefundCompletionNotificationHandler) authorizeInternalRequest(
	r *http.Request,
	requireCloudTasksHeader bool,
) error {
	if h == nil {
		return errRefundCompletionNotificationAuthNotConfigured
	}

	audience := strings.TrimSpace(h.audience)
	serviceAccountEmail := strings.ToLower(
		strings.TrimSpace(h.serviceAccountEmail),
	)

	if audience == "" || serviceAccountEmail == "" {
		return errRefundCompletionNotificationAuthNotConfigured
	}

	rawToken, ok := refundCompletionNotificationBearerToken(
		r.Header.Get("Authorization"),
	)
	if !ok {
		return errRefundCompletionNotificationUnauthorized
	}

	payload, err := idtoken.Validate(
		r.Context(),
		rawToken,
		audience,
	)
	if err != nil || payload == nil {
		return errRefundCompletionNotificationUnauthorized
	}

	tokenEmail, _ := payload.Claims["email"].(string)
	tokenEmail = strings.ToLower(
		strings.TrimSpace(tokenEmail),
	)

	if tokenEmail == "" || tokenEmail != serviceAccountEmail {
		return errRefundCompletionNotificationForbidden
	}

	if !refundCompletionNotificationEmailVerified(
		payload.Claims["email_verified"],
	) {
		return errRefundCompletionNotificationForbidden
	}

	if requireCloudTasksHeader &&
		strings.TrimSpace(
			r.Header.Get("X-CloudTasks-TaskName"),
		) == "" {
		return errRefundCompletionNotificationForbidden
	}

	return nil
}

func (h *RefundCompletionNotificationHandler) writeAuthorizationError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		errRefundCompletionNotificationAuthNotConfigured,
	):
		writeRefundCompletionNotificationError(
			w,
			http.StatusServiceUnavailable,
			"refund_completion_notification_auth_unavailable",
			0,
		)

	case errors.Is(
		err,
		errRefundCompletionNotificationForbidden,
	):
		writeRefundCompletionNotificationError(
			w,
			http.StatusForbidden,
			"forbidden",
			0,
		)

	default:
		writeRefundCompletionNotificationError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			0,
		)
	}
}

func refundCompletionNotificationBearerToken(
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

func refundCompletionNotificationEmailVerified(
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

func decodeRequiredRefundCompletionNotificationJSON(
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
		maxRefundCompletionNotificationRequestBodyBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	return ensureSingleRefundCompletionNotificationJSONValue(
		decoder,
	)
}

func decodeOptionalRefundCompletionNotificationJSON(
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
		maxRefundCompletionNotificationRequestBodyBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}

		return err
	}

	return ensureSingleRefundCompletionNotificationJSONValue(
		decoder,
	)
}

func ensureSingleRefundCompletionNotificationJSONValue(
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

func writeRefundCompletionNotificationError(
	w http.ResponseWriter,
	statusCode int,
	message string,
	enqueuedCount int,
) {
	writeRefundCompletionNotificationJSON(
		w,
		statusCode,
		refundCompletionNotificationErrorResponse{
			Error:    message,
			Enqueued: enqueuedCount,
		},
	)
}

func writeRefundCompletionNotificationJSON(
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

func firstNonEmptyRefundCompletionNotificationEnvironmentValue(
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
