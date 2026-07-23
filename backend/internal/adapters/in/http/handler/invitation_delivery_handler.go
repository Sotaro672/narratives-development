// backend\internal\adapters\in\http\handler\invitation_delivery_handler.go
package internalHandler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"google.golang.org/api/idtoken"

	uc "narratives/internal/application/usecase"
)

const (
	envInvitationDeliveryCloudTasksAudience = "INVITATION_DELIVERY_CLOUD_TASKS_AUDIENCE"

	envInvitationDeliveryCloudTasksServiceAccount = "INVITATION_DELIVERY_CLOUD_TASKS_SERVICE_ACCOUNT"

	envCloudTasksAudience = "CLOUD_TASKS_AUDIENCE"

	envCloudTasksServiceAccount = "CLOUD_TASKS_SERVICE_ACCOUNT"

	envInternalBaseURL = "INTERNAL_BASE_URL"

	envSelfBaseURL = "SELF_BASE_URL"

	maxInvitationDeliveryRequestBodyBytes int64 = 64 * 1024
)

var (
	errInvitationDeliveryAuthNotConfigured = errors.New(
		"invitation delivery authentication is not configured",
	)

	errInvitationDeliveryUnauthorized = errors.New(
		"invitation delivery request is unauthorized",
	)

	errInvitationDeliveryForbidden = errors.New(
		"invitation delivery request is forbidden",
	)
)

type InvitationDeliveryHandler struct {
	deliveryUC          uc.InvitationDeliveryUsecasePort
	audience            string
	serviceAccountEmail string
}

type processInvitationDeliveryRequest struct {
	DeliveryID string `json:"deliveryId"`
}

type dispatchInvitationDeliveriesRequest struct {
	Limit int `json:"limit"`
}

type dispatchInvitationDeliveriesResponse struct {
	Enqueued int `json:"enqueued"`
}

type invitationDeliveryErrorResponse struct {
	Error    string `json:"error"`
	Enqueued int    `json:"enqueued,omitempty"`
}

// NewInvitationDeliveryHandlerは、招待メールdelivery用のinternal handlerを
// 生成します。
//
// 認証設定は、招待専用環境変数を優先し、未設定の場合は共通の
// Cloud Tasks環境変数へフォールバックします。
func NewInvitationDeliveryHandler(
	deliveryUC uc.InvitationDeliveryUsecasePort,
) *InvitationDeliveryHandler {
	audience := firstNonEmptyInvitationDeliveryEnvironmentValue(
		envInvitationDeliveryCloudTasksAudience,
		envCloudTasksAudience,
		envInternalBaseURL,
		envSelfBaseURL,
	)

	serviceAccountEmail :=
		firstNonEmptyInvitationDeliveryEnvironmentValue(
			envInvitationDeliveryCloudTasksServiceAccount,
			envCloudTasksServiceAccount,
		)

	return &InvitationDeliveryHandler{
		deliveryUC: deliveryUC,
		audience: strings.TrimRight(
			audience,
			"/",
		),
		serviceAccountEmail: strings.ToLower(
			strings.TrimSpace(serviceAccountEmail),
		),
	}
}

// ServeHTTPは、個別delivery処理endpointとして動作します。
func (h *InvitationDeliveryHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.Process(w, r)
}

// Processは、Cloud Tasksから渡されたdelivery IDを処理します。
//
// 正常終了、重複task、処理済みtaskは204を返します。
// infrastructure errorの場合は500を返し、Cloud Tasks標準retryの対象とします。
func (h *InvitationDeliveryHandler) Process(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeInvitationDeliveryError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			0,
		)
		return
	}

	if h == nil || h.deliveryUC == nil {
		writeInvitationDeliveryError(
			w,
			http.StatusServiceUnavailable,
			"invitation_delivery_usecase_unavailable",
			0,
		)
		return
	}

	if err := h.authorizeInternalRequest(r, true); err != nil {
		h.writeAuthorizationError(w, err)
		return
	}

	var request processInvitationDeliveryRequest

	if err := decodeRequiredInvitationDeliveryJSON(
		w,
		r,
		&request,
	); err != nil {
		writeInvitationDeliveryError(
			w,
			http.StatusBadRequest,
			"invalid_json_body",
			0,
		)
		return
	}

	request.DeliveryID = strings.TrimSpace(
		request.DeliveryID,
	)
	if request.DeliveryID == "" {
		writeInvitationDeliveryError(
			w,
			http.StatusBadRequest,
			"delivery_id_required",
			0,
		)
		return
	}

	if err := h.deliveryUC.Process(
		r.Context(),
		request.DeliveryID,
	); err != nil {
		log.Printf(
			"[invitation-delivery-handler] process failed deliveryId=%q taskName=%q queueName=%q retryCount=%q executionCount=%q err=%v",
			request.DeliveryID,
			r.Header.Get("X-CloudTasks-TaskName"),
			r.Header.Get("X-CloudTasks-QueueName"),
			r.Header.Get("X-CloudTasks-TaskRetryCount"),
			r.Header.Get("X-CloudTasks-TaskExecutionCount"),
			err,
		)

		writeInvitationDeliveryError(
			w,
			http.StatusInternalServerError,
			"invitation_delivery_processing_failed",
			0,
		)
		return
	}

	log.Printf(
		"[invitation-delivery-handler] process completed deliveryId=%q taskName=%q queueName=%q",
		request.DeliveryID,
		r.Header.Get("X-CloudTasks-TaskName"),
		r.Header.Get("X-CloudTasks-QueueName"),
	)

	w.WriteHeader(http.StatusNoContent)
}

// DispatchDueは、送信時刻を迎えたoutboxを検索し、Cloud Tasksへ投入します。
//
// Cloud SchedulerなどからOIDC付きで呼び出すことを想定しています。
// bodyは省略可能です。
//
//	{
//	  "limit": 50
//	}
func (h *InvitationDeliveryHandler) DispatchDue(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeInvitationDeliveryError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			0,
		)
		return
	}

	if h == nil || h.deliveryUC == nil {
		writeInvitationDeliveryError(
			w,
			http.StatusServiceUnavailable,
			"invitation_delivery_usecase_unavailable",
			0,
		)
		return
	}

	if err := h.authorizeInternalRequest(r, false); err != nil {
		h.writeAuthorizationError(w, err)
		return
	}

	var request dispatchInvitationDeliveriesRequest

	if err := decodeOptionalInvitationDeliveryJSON(
		w,
		r,
		&request,
	); err != nil {
		writeInvitationDeliveryError(
			w,
			http.StatusBadRequest,
			"invalid_json_body",
			0,
		)
		return
	}

	if request.Limit < 0 {
		writeInvitationDeliveryError(
			w,
			http.StatusBadRequest,
			"limit_must_not_be_negative",
			0,
		)
		return
	}

	enqueuedCount, err := h.deliveryUC.DispatchDue(
		r.Context(),
		request.Limit,
	)
	if err != nil {
		log.Printf(
			"[invitation-delivery-handler] dispatch due failed enqueued=%d err=%v",
			enqueuedCount,
			err,
		)

		writeInvitationDeliveryError(
			w,
			http.StatusInternalServerError,
			"invitation_delivery_dispatch_failed",
			enqueuedCount,
		)
		return
	}

	log.Printf(
		"[invitation-delivery-handler] dispatch due completed enqueued=%d",
		enqueuedCount,
	)

	writeInvitationDeliveryJSON(
		w,
		http.StatusOK,
		dispatchInvitationDeliveriesResponse{
			Enqueued: enqueuedCount,
		},
	)
}

func (h *InvitationDeliveryHandler) authorizeInternalRequest(
	r *http.Request,
	requireCloudTasksHeader bool,
) error {
	if h == nil {
		return errInvitationDeliveryAuthNotConfigured
	}

	audience := strings.TrimSpace(h.audience)
	serviceAccountEmail := strings.ToLower(
		strings.TrimSpace(h.serviceAccountEmail),
	)

	if audience == "" || serviceAccountEmail == "" {
		log.Printf(
			"[invitation-delivery-handler] authentication configuration is missing audienceConfigured=%t serviceAccountConfigured=%t",
			audience != "",
			serviceAccountEmail != "",
		)

		return errInvitationDeliveryAuthNotConfigured
	}

	rawToken, ok := invitationDeliveryBearerToken(
		r.Header.Get("Authorization"),
	)
	if !ok {
		return errInvitationDeliveryUnauthorized
	}

	payload, err := idtoken.Validate(
		r.Context(),
		rawToken,
		audience,
	)
	if err != nil {
		log.Printf(
			"[invitation-delivery-handler] OIDC validation failed err=%v",
			err,
		)

		return errInvitationDeliveryUnauthorized
	}

	if payload == nil {
		return errInvitationDeliveryUnauthorized
	}

	tokenEmail, _ := payload.Claims["email"].(string)
	tokenEmail = strings.ToLower(
		strings.TrimSpace(tokenEmail),
	)

	if tokenEmail == "" ||
		tokenEmail != serviceAccountEmail {
		log.Printf(
			"[invitation-delivery-handler] OIDC service account mismatch tokenEmail=%q",
			tokenEmail,
		)

		return errInvitationDeliveryForbidden
	}

	if !invitationDeliveryEmailVerified(
		payload.Claims["email_verified"],
	) {
		return errInvitationDeliveryForbidden
	}

	if requireCloudTasksHeader &&
		strings.TrimSpace(
			r.Header.Get("X-CloudTasks-TaskName"),
		) == "" {
		return errInvitationDeliveryForbidden
	}

	return nil
}

func (h *InvitationDeliveryHandler) writeAuthorizationError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		errInvitationDeliveryAuthNotConfigured,
	):
		writeInvitationDeliveryError(
			w,
			http.StatusServiceUnavailable,
			"invitation_delivery_auth_unavailable",
			0,
		)

	case errors.Is(
		err,
		errInvitationDeliveryForbidden,
	):
		writeInvitationDeliveryError(
			w,
			http.StatusForbidden,
			"forbidden",
			0,
		)

	default:
		writeInvitationDeliveryError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
			0,
		)
	}
}

func invitationDeliveryBearerToken(
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

func invitationDeliveryEmailVerified(
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

func decodeRequiredInvitationDeliveryJSON(
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
		maxInvitationDeliveryRequestBodyBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	return ensureSingleInvitationDeliveryJSONValue(
		decoder,
	)
}

func decodeOptionalInvitationDeliveryJSON(
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
		maxInvitationDeliveryRequestBodyBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}

		return err
	}

	return ensureSingleInvitationDeliveryJSONValue(
		decoder,
	)
}

func ensureSingleInvitationDeliveryJSONValue(
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

func writeInvitationDeliveryError(
	w http.ResponseWriter,
	statusCode int,
	message string,
	enqueuedCount int,
) {
	writeInvitationDeliveryJSON(
		w,
		statusCode,
		invitationDeliveryErrorResponse{
			Error:    message,
			Enqueued: enqueuedCount,
		},
	)
}

func writeInvitationDeliveryJSON(
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

	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf(
			"[invitation-delivery-handler] encode response failed status=%d err=%v",
			statusCode,
			err,
		)
	}
}

func firstNonEmptyInvitationDeliveryEnvironmentValue(
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

func (h *InvitationDeliveryHandler) String() string {
	if h == nil {
		return "InvitationDeliveryHandler<nil>"
	}

	return fmt.Sprintf(
		"InvitationDeliveryHandler<audience=%q,serviceAccountConfigured=%t>",
		h.audience,
		h.serviceAccountEmail != "",
	)
}
