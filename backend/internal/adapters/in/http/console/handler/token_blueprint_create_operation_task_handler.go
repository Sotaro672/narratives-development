// backend/internal/adapters/in/http/console/handler/token_blueprint_create_operation_task_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	usecase "narratives/internal/application/usecase"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

const (
	tokenBlueprintCreateOperationTaskBasePath = "/internal/token-blueprint/create-operations"

	tokenBlueprintCreateOperationTaskMaxBodyBytes int64 = 64 << 10
)

type TokenBlueprintCreateOperationTaskHandler struct {
	uc *usecase.TokenBlueprintCreateOperationUsecase
}

type NewTokenBlueprintCreateOperationTaskHandlerParams struct {
	UC *usecase.TokenBlueprintCreateOperationUsecase
}

type tokenBlueprintCreateOperationTaskRequest struct {
	OperationID string `json:"operationId"`
}

type tokenBlueprintCreateOperationTaskErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func NewTokenBlueprintCreateOperationTaskHandler(
	p NewTokenBlueprintCreateOperationTaskHandlerParams,
) http.Handler {
	return &TokenBlueprintCreateOperationTaskHandler{
		uc: p.UC,
	}
}

func (
	h *TokenBlueprintCreateOperationTaskHandler,
) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil ||
		h.uc == nil {
		writeTokenBlueprintCreateOperationTaskError(
			w,
			http.StatusInternalServerError,
			"usecase_unavailable",
			"token blueprint create operation usecase is nil",
		)
		return
	}

	if r == nil {
		writeTokenBlueprintCreateOperationTaskError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"request is nil",
		)
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set(
			"Allow",
			http.MethodPost,
		)

		writeTokenBlueprintCreateOperationTaskError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			"method not allowed",
		)
		return
	}

	operationID, err :=
		decodeTokenBlueprintCreateOperationTaskPath(
			r.URL.Path,
		)
	if err != nil {
		writeTokenBlueprintCreateOperationTaskError(
			w,
			http.StatusNotFound,
			"not_found",
			err.Error(),
		)
		return
	}

	bodyOperationID, err :=
		decodeTokenBlueprintCreateOperationTaskRequest(
			w,
			r,
		)
	if err != nil {
		writeTokenBlueprintCreateOperationTaskError(
			w,
			http.StatusBadRequest,
			"invalid_json",
			err.Error(),
		)
		return
	}

	if bodyOperationID != "" &&
		bodyOperationID != operationID {
		writeTokenBlueprintCreateOperationTaskError(
			w,
			http.StatusBadRequest,
			"operation_id_mismatch",
			"operationId in request body does not match path",
		)
		return
	}

	operation, err :=
		h.uc.Get(
			r.Context(),
			operationID,
		)
	if err != nil {
		writeTokenBlueprintCreateOperationTaskUsecaseError(
			w,
			err,
		)
		return
	}

	switch operation.Status {
	case tbdom.CreateOperationStatusCompleted,
		tbdom.CreateOperationStatusFailedFatal:
		w.WriteHeader(
			http.StatusNoContent,
		)
		return

	case tbdom.CreateOperationStatusFailedRetryable:
		operation, err =
			h.uc.Retry(
				r.Context(),
				operationID,
			)

	case tbdom.CreateOperationStatusQueued,
		tbdom.CreateOperationStatusProcessing:
		operation, err =
			h.uc.Execute(
				r.Context(),
				operationID,
			)

	case tbdom.CreateOperationStatusWaitingUpload:
		writeTokenBlueprintCreateOperationTaskError(
			w,
			http.StatusUnprocessableEntity,
			"upload_not_committed",
			"token blueprint create operation is still waiting for uploads",
		)
		return

	default:
		writeTokenBlueprintCreateOperationTaskError(
			w,
			http.StatusUnprocessableEntity,
			"unsupported_status",
			fmt.Sprintf(
				"unsupported token blueprint create operation status %q",
				operation.Status,
			),
		)
		return
	}

	if err != nil {
		if acknowledgeTokenBlueprintCreateOperationTaskResult(
			operation,
			err,
		) {
			w.WriteHeader(
				http.StatusNoContent,
			)
			return
		}

		writeTokenBlueprintCreateOperationTaskUsecaseError(
			w,
			err,
		)
		return
	}

	if operation.Status ==
		tbdom.CreateOperationStatusFailedRetryable {
		writeTokenBlueprintCreateOperationTaskError(
			w,
			http.StatusServiceUnavailable,
			"create_operation_failed_retryable",
			tokenBlueprintCreateOperationTaskMessage(
				operation.LastError,
			),
		)
		return
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}

func decodeTokenBlueprintCreateOperationTaskPath(
	path string,
) (
	string,
	error,
) {
	path =
		strings.TrimSuffix(
			strings.TrimSpace(
				path,
			),
			"/",
		)

	prefix :=
		tokenBlueprintCreateOperationTaskBasePath +
			"/"

	if !strings.HasPrefix(
		path,
		prefix,
	) {
		return "",
			errors.New(
				"endpoint not found",
			)
	}

	rest :=
		strings.TrimPrefix(
			path,
			prefix,
		)

	parts :=
		strings.Split(
			rest,
			"/",
		)

	if len(parts) != 2 ||
		parts[1] != "execute" {
		return "",
			errors.New(
				"endpoint not found",
			)
	}

	operationID, err :=
		url.PathUnescape(
			parts[0],
		)
	if err != nil {
		return "",
			errors.New(
				"operationId contains invalid URL encoding",
			)
	}

	operationID =
		strings.TrimSpace(
			operationID,
		)

	if err :=
		validateTokenBlueprintCreateOperationTaskOperationID(
			operationID,
		); err != nil {
		return "", err
	}

	return operationID, nil
}

func decodeTokenBlueprintCreateOperationTaskRequest(
	w http.ResponseWriter,
	r *http.Request,
) (
	string,
	error,
) {
	if r == nil ||
		r.Body == nil {
		return "", nil
	}

	body :=
		http.MaxBytesReader(
			w,
			r.Body,
			tokenBlueprintCreateOperationTaskMaxBodyBytes,
		)

	decoder :=
		json.NewDecoder(
			body,
		)

	decoder.DisallowUnknownFields()

	var request tokenBlueprintCreateOperationTaskRequest

	if err :=
		decoder.Decode(
			&request,
		); err != nil {
		if errors.Is(
			err,
			io.EOF,
		) {
			return "", nil
		}

		return "",
			fmt.Errorf(
				"decode request body: %w",
				err,
			)
	}

	var extra any

	if err :=
		decoder.Decode(
			&extra,
		); !errors.Is(
		err,
		io.EOF,
	) {
		return "",
			errors.New(
				"request body must contain exactly one JSON object",
			)
	}

	request.OperationID =
		strings.TrimSpace(
			request.OperationID,
		)

	if request.OperationID == "" {
		return "",
			errors.New(
				"operationId is required",
			)
	}

	if err :=
		validateTokenBlueprintCreateOperationTaskOperationID(
			request.OperationID,
		); err != nil {
		return "", err
	}

	return request.OperationID, nil
}

func validateTokenBlueprintCreateOperationTaskOperationID(
	operationID string,
) error {
	operationID =
		strings.TrimSpace(
			operationID,
		)

	if operationID == "" {
		return errors.New(
			"operationId is required",
		)
	}

	if len(operationID) > 512 {
		return errors.New(
			"operationId must not exceed 512 characters",
		)
	}

	if strings.Contains(
		operationID,
		"/",
	) ||
		strings.Contains(
			operationID,
			"://",
		) ||
		strings.ContainsAny(
			operationID,
			"\r\n\x00",
		) {
		return errors.New(
			"operationId is invalid",
		)
	}

	return nil
}

func acknowledgeTokenBlueprintCreateOperationTaskResult(
	operation tbdom.CreateOperation,
	err error,
) bool {
	if err == nil {
		return true
	}

	if operation.Status ==
		tbdom.CreateOperationStatusCompleted ||
		operation.Status ==
			tbdom.CreateOperationStatusFailedFatal {
		return true
	}

	if errors.Is(
		err,
		tbdom.ErrCreateOperationNotFound,
	) ||
		errors.Is(
			err,
			tbdom.ErrNotFound,
		) {
		return true
	}

	return false
}

func writeTokenBlueprintCreateOperationTaskUsecaseError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		context.DeadlineExceeded,
	):
		writeTokenBlueprintCreateOperationTaskError(
			w,
			http.StatusServiceUnavailable,
			"deadline_exceeded",
			"token blueprint create operation task deadline exceeded",
		)

	case errors.Is(
		err,
		context.Canceled,
	):
		writeTokenBlueprintCreateOperationTaskError(
			w,
			http.StatusServiceUnavailable,
			"request_canceled",
			"token blueprint create operation task was canceled",
		)

	case errors.Is(
		err,
		tbdom.ErrCreateOperationNotFound,
	),
		errors.Is(
			err,
			tbdom.ErrNotFound,
		):
		// OperationまたはTokenBlueprint本体が既に存在しない場合、
		// Cloud Tasksへ再試行させても復旧しないためACKする。
		w.WriteHeader(
			http.StatusNoContent,
		)

	case errors.Is(
		err,
		tbdom.ErrCreateOperationConflict,
	),
		errors.Is(
			err,
			tbdom.ErrConflict,
		):
		// 楽観ロック競合などは再実行可能。
		writeTokenBlueprintCreateOperationTaskError(
			w,
			http.StatusServiceUnavailable,
			"create_operation_conflict",
			"token blueprint create operation update conflict",
		)

	case errors.Is(
		err,
		tbdom.ErrCreateOperationNotRetryable,
	),
		errors.Is(
			err,
			tbdom.ErrCreateOperationRetryExhausted,
		):
		writeTokenBlueprintCreateOperationTaskError(
			w,
			http.StatusConflict,
			"create_operation_not_retryable",
			err.Error(),
		)

	case errors.Is(
		err,
		tbdom.ErrInvalidCreateOperation,
	),
		errors.Is(
			err,
			tbdom.ErrInvalidCreateOperationTransition,
		),
		errors.Is(
			err,
			tbdom.ErrCreateOperationUploadIncomplete,
		),
		errors.Is(
			err,
			tbdom.ErrCreateOperationAssetNotFound,
		),
		errors.Is(
			err,
			tbdom.ErrCreateOperationAssetConflict,
		):
		writeTokenBlueprintCreateOperationTaskError(
			w,
			http.StatusUnprocessableEntity,
			"invalid_create_operation",
			err.Error(),
		)

	default:
		writeTokenBlueprintCreateOperationTaskError(
			w,
			http.StatusServiceUnavailable,
			"task_execution_failed",
			"token blueprint create operation task execution failed",
		)
	}
}

func writeTokenBlueprintCreateOperationTaskError(
	w http.ResponseWriter,
	statusCode int,
	code string,
	message string,
) {
	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	w.WriteHeader(
		statusCode,
	)

	_ = json.NewEncoder(
		w,
	).Encode(
		tokenBlueprintCreateOperationTaskErrorResponse{
			Error: code,

			Message: message,
		},
	)
}

func tokenBlueprintCreateOperationTaskMessage(
	message string,
) string {
	message =
		strings.TrimSpace(
			message,
		)

	if message == "" {
		return "token blueprint create operation failed and can be retried"
	}

	return message
}
