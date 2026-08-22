// backend/internal/adapters/in/http/handler/order_dispatch_notification_handler.go
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
	envOrderDispatchNotificationCloudTasksAudience       = "CLOUD_TASKS_AUDIENCE"
	envOrderDispatchNotificationCloudTasksServiceAccount = "CLOUD_TASKS_SERVICE_ACCOUNT"
	envOrderDispatchNotificationInternalBaseURL          = "INTERNAL_BASE_URL"
	envOrderDispatchNotificationSelfBaseURL              = "SELF_BASE_URL"

	maxOrderDispatchNotificationRequestBodyBytes int64 = 64 * 1024
)

var (
	errOrderDispatchNotificationAuthNotConfigured = errors.New(
		"order dispatch notification authentication is not configured",
	)
	errOrderDispatchNotificationUnauthorized = errors.New(
		"order dispatch notification request is unauthorized",
	)
	errOrderDispatchNotificationForbidden = errors.New(
		"order dispatch notification request is forbidden",
	)
)

type OrderDispatchNotificationHandler struct {
	notificationUC      uc.OrderDispatchNotificationUsecasePort
	audience            string
	serviceAccountEmail string
}

type processOrderDispatchNotificationRequest struct {
	DeliveryID string `json:"deliveryId"`
}

type dispatchOrderDispatchNotificationsRequest struct {
	Limit int `json:"limit"`
}

type dispatchOrderDispatchNotificationsResponse struct {
	Enqueued int `json:"enqueued"`
}

type orderDispatchNotificationErrorResponse struct {
	Error    string `json:"error"`
	Enqueued int    `json:"enqueued,omitempty"`
}

func NewOrderDispatchNotificationHandler(
	notificationUC uc.OrderDispatchNotificationUsecasePort,
) *OrderDispatchNotificationHandler {
	audience := firstNonEmptyOrderDispatchNotificationEnvironmentValue(
		envOrderDispatchNotificationCloudTasksAudience,
		envOrderDispatchNotificationInternalBaseURL,
		envOrderDispatchNotificationSelfBaseURL,
	)

	serviceAccountEmail := firstNonEmptyOrderDispatchNotificationEnvironmentValue(
		envOrderDispatchNotificationCloudTasksServiceAccount,
	)

	return &OrderDispatchNotificationHandler{
		notificationUC: notificationUC,
		audience:       strings.TrimRight(audience, "/"),
		serviceAccountEmail: strings.ToLower(
			strings.TrimSpace(serviceAccountEmail),
		),
	}
}

func (h *OrderDispatchNotificationHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.Process(w, r)
}

// ProcessはCloud Tasksから渡されたdelivery IDを処理します。
// 正常終了、重複task、処理済みtaskは204を返します。
func (h *OrderDispatchNotificationHandler) Process(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeOrderDispatchNotificationError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			0,
		)
		return
	}

	if h == nil || h.notificationUC == nil {
		writeOrderDispatchNotificationError(
			w,
			http.StatusServiceUnavailable,
			"order_dispatch_notification_usecase_unavailable",
			0,
		)
		return
	}

	if err := h.authorizeInternalRequest(r, true); err != nil {
		h.writeAuthorizationError(w, err)
		return
	}

	var request processOrderDispatchNotificationRequest

	if err := decodeRequiredOrderDispatchNotificationJSON(
		w,
		r,
		&request,
	); err != nil {
		writeOrderDispatchNotificationError(
			w,
			http.StatusBadRequest,
			"invalid_json_body",
			0,
		)
		return
	}

	request.DeliveryID = strings.TrimSpace(request.DeliveryID)
	if request.DeliveryID == "" {
		writeOrderDispatchNotificationError(
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
		writeOrderDispatchNotificationError(
			w,
			http.StatusInternalServerError,
			"order_dispatch_notification_processing_failed",
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
func (h *OrderDispatchNotificationHandler) DispatchDue(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeOrderDispatchNotificationError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			0,
		)
		return
	}

	if h == nil || h.notificationUC == nil {
		writeOrderDispatchNotificationError(
			w,
			http.StatusServiceUnavailable,
			"order_dispatch_notification_usecase_unavailable",
			0,
		)
		return
	}

	if err := h.authorizeInternalRequest(r, false); err != nil {
		h.writeAuthorizationError(w, err)
		return
	}

	var request dispatchOrderDispatchNotificationsRequest

	if err := decodeOptionalOrderDispatchNotificationJSON(
		w,
		r,
		&request,
	); err != nil {
		writeOrderDispatchNotificationError(
			w,
			http.StatusBadRequest,
			"invalid_json_body",
			0,
		)
		return
	}

	if request.Limit < 0 {
		writeOrderDispatchNotificationError(
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
		writeOrderDispatchNotificationError(
			w,
			http.StatusInternalServerError,
			"order_dispatch_notification_dispatch_failed",
			enqueuedCount,
		)
		return
	}

	writeOrderDispatchNotificationJSON(
		w,
		http.StatusOK,
		dispatchOrderDispatchNotificationsResponse{
			Enqueued: enqueuedCount,
		},
	)
}

func (h *OrderDispatchNotificationHandler) authorizeInternalRequest(
	r *http.Request,
	requireCloudTasksHeader bool,
) error {
	if h == nil {
		return errOrderDispatchNotificationAuthNotConfigured
	}

	audience := strings.TrimSpace(h.audience)
	serviceAccountEmail := strings.ToLower(
		strings.TrimSpace(h.serviceAccountEmail),
	)

	if audience == "" || serviceAccountEmail == "" {
		return errOrderDispatchNotificationAuthNotConfigured
	}

	rawToken, ok := orderDispatchNotificationBearerToken(
		r.Header.Get("Authorization"),
	)
	if !ok {
		return errOrderDispatchNotificationUnauthorized
	}

	payload, err := idtoken.Validate(
		r.Context(),
		rawToken,
		audience,
	)
	if err != nil || payload == nil {
		return errOrderDispatchNotificationUnauthorized
	}

	tokenEmail, _ := payload.Claims["email"].(string)
	tokenEmail = strings.ToLower(
		strings.TrimSpace(tokenEmail),
	)

	if tokenEmail == "" || tokenEmail != serviceAccountEmail {
		return errOrderDispatchNotificationForbidden
	}

	if !orderDispatchNotificationEmailVerified(
		payload.Claims["email_verified"],
	) {
		return errOrderDispatchNotificationForbidden
	}

	if requireCloudTasksHeader &&
		strings.TrimSpace(
			r.Header.Get("X-CloudTasks-TaskName"),
		) == "" {
		return errOrderDispatchNotificationForbidden
	}

	return nil
}

func (h *OrderDispatchNotificationHandler) writeAuthorizationError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		errOrderDispatchNotificationAuthNotConfigured,
	):
		writeOrderDispatchNotificationError(
			w,
			http.StatusServiceUnavailable,
			"order_dispatch_notification_auth_unavailable",
			0,
		)

	case errors.Is(
		err,
		errOrderDispatchNotificationForbidden,
	):
		writeOrderDispatchNotificationError(
			w,
			http.StatusForbidden,
			"forbidden",
			0,
		)

	default:
		writeOrderDispatchNotificationError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			0,
		)
	}
}

func orderDispatchNotificationBearerToken(
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

func orderDispatchNotificationEmailVerified(
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

func decodeRequiredOrderDispatchNotificationJSON(
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
		maxOrderDispatchNotificationRequestBodyBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	return ensureSingleOrderDispatchNotificationJSONValue(
		decoder,
	)
}

func decodeOptionalOrderDispatchNotificationJSON(
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
		maxOrderDispatchNotificationRequestBodyBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}

		return err
	}

	return ensureSingleOrderDispatchNotificationJSONValue(
		decoder,
	)
}

func ensureSingleOrderDispatchNotificationJSONValue(
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

func writeOrderDispatchNotificationError(
	w http.ResponseWriter,
	statusCode int,
	message string,
	enqueuedCount int,
) {
	writeOrderDispatchNotificationJSON(
		w,
		statusCode,
		orderDispatchNotificationErrorResponse{
			Error:    message,
			Enqueued: enqueuedCount,
		},
	)
}

func writeOrderDispatchNotificationJSON(
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

func firstNonEmptyOrderDispatchNotificationEnvironmentValue(
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
