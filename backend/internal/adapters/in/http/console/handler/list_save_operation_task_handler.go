// backend/internal/adapters/in/http/console/handler/list_save_operation_task_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	usecase "narratives/internal/application/usecase"
	listdom "narratives/internal/domain/list"
	"net/http"
	"net/url"
	"strings"
)

const (
	listSaveOperationTaskBasePath           = "/internal/list/save-operations"
	listSaveOperationTaskMaxBodyBytes int64 = 64 << 10
)

type ListSaveOperationTaskHandler struct {
	uc *usecase.ListSaveOperationUsecase
}
type NewListSaveOperationTaskHandlerParams struct {
	UC *usecase.ListSaveOperationUsecase
}
type listSaveOperationTaskRequest struct {
	OperationID string `json:"operationId"`
}
type listSaveOperationTaskErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func NewListSaveOperationTaskHandler(p NewListSaveOperationTaskHandlerParams) http.Handler {
	return &ListSaveOperationTaskHandler{
		uc: p.UC,
	}
}
func (h *ListSaveOperationTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		writeListSaveOperationTaskError(
			w,
			http.StatusInternalServerError,
			"usecase_unavailable",
			"list save operation usecase is nil",
		)
		return
	}
	if r == nil {
		writeListSaveOperationTaskError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"request is nil",
		)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeListSaveOperationTaskError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			"method not allowed",
		)
		return
	}
	operationID, err := decodeListSaveOperationTaskPath(r.URL.Path)
	if err != nil {
		writeListSaveOperationTaskError(
			w,
			http.StatusNotFound,
			"not_found",
			err.Error(),
		)
		return
	}
	bodyOperationID, err := decodeListSaveOperationTaskRequest(w, r)
	if err != nil {
		writeListSaveOperationTaskError(
			w,
			http.StatusBadRequest,
			"invalid_json",
			err.Error(),
		)
		return
	}
	if bodyOperationID != "" && bodyOperationID != operationID {
		writeListSaveOperationTaskError(
			w,
			http.StatusBadRequest,
			"operation_id_mismatch",
			"operationId in request body does not match path",
		)
		return
	}
	operation, err := h.uc.Get(r.Context(), operationID)
	if err != nil {
		writeListSaveOperationTaskUsecaseError(w, err)
		return
	}
	switch operation.Status {
	case listdom.SaveOperationStatusCompleted,
		listdom.SaveOperationStatusCompensated,
		listdom.SaveOperationStatusFailedFatal:
		w.WriteHeader(http.StatusNoContent)
		return
	case listdom.SaveOperationStatusFailedRetryable:
		operation, err = h.uc.Retry(r.Context(), operationID)
	case listdom.SaveOperationStatusCompensating,
		listdom.SaveOperationStatusPending,
		listdom.SaveOperationStatusUploading,
		listdom.SaveOperationStatusRegisteringImages,
		listdom.SaveOperationStatusDeletingImages,
		listdom.SaveOperationStatusUpdatingList,
		listdom.SaveOperationStatusSettingPrimary:
		operation, err = h.uc.Execute(r.Context(), operationID)
	default:
		writeListSaveOperationTaskError(
			w,
			http.StatusUnprocessableEntity,
			"unsupported_status",
			fmt.Sprintf("unsupported list save operation status %q", operation.Status),
		)
		return
	}
	if err != nil {
		if acknowledgeListSaveOperationTaskResult(operation, err) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeListSaveOperationTaskUsecaseError(w, err)
		return
	}
	if operation.Status == listdom.SaveOperationStatusFailedRetryable {
		writeListSaveOperationTaskError(
			w,
			http.StatusServiceUnavailable,
			"save_operation_failed_retryable",
			listSaveOperationTaskMessage(operation.LastError),
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func decodeListSaveOperationTaskPath(path string) (string, error) {
	path = strings.TrimSuffix(strings.TrimSpace(path), "/")
	prefix := listSaveOperationTaskBasePath + "/"
	if !strings.HasPrefix(path, prefix) {
		return "", errors.New("endpoint not found")
	}
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != "retry" {
		return "", errors.New("endpoint not found")
	}
	operationID, err := url.PathUnescape(parts[0])
	if err != nil {
		return "", errors.New("operationId contains invalid URL encoding")
	}
	operationID = strings.TrimSpace(operationID)
	if err := validateListSaveOperationTaskOperationID(operationID); err != nil {
		return "", err
	}
	return operationID, nil
}
func decodeListSaveOperationTaskRequest(w http.ResponseWriter, r *http.Request) (string, error) {
	if r == nil || r.Body == nil {
		return "", nil
	}
	body := http.MaxBytesReader(w, r.Body, listSaveOperationTaskMaxBodyBytes)
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	var request listSaveOperationTaskRequest
	if err := decoder.Decode(&request); err != nil {
		if errors.Is(err, io.EOF) {
			return "", nil
		}
		return "", fmt.Errorf("decode request body: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return "", errors.New("request body must contain exactly one JSON object")
	}
	request.OperationID = strings.TrimSpace(request.OperationID)
	if request.OperationID == "" {
		return "", errors.New("operationId is required")
	}
	if err := validateListSaveOperationTaskOperationID(request.OperationID); err != nil {
		return "", err
	}
	return request.OperationID, nil
}
func validateListSaveOperationTaskOperationID(operationID string) error {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return errors.New("operationId is required")
	}
	if len(operationID) > 512 {
		return errors.New("operationId must not exceed 512 characters")
	}
	if strings.Contains(operationID, "/") ||
		strings.Contains(operationID, "://") ||
		strings.ContainsAny(operationID, "\r\n\x00") {
		return errors.New("operationId is invalid")
	}
	return nil
}
func acknowledgeListSaveOperationTaskResult(operation listdom.SaveOperation, err error) bool {
	if err == nil {
		return true
	}
	if operation.Status == listdom.SaveOperationStatusCompleted ||
		operation.Status == listdom.SaveOperationStatusCompensated ||
		operation.Status == listdom.SaveOperationStatusFailedFatal {
		return true
	}
	if errors.Is(err, listdom.ErrSaveOperationNotFound) ||
		errors.Is(err, listdom.ErrNotFound) {
		return true
	}
	return false
}
func writeListSaveOperationTaskUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		writeListSaveOperationTaskError(
			w,
			http.StatusServiceUnavailable,
			"deadline_exceeded",
			"list save operation task deadline exceeded",
		)
	case errors.Is(err, context.Canceled):
		writeListSaveOperationTaskError(
			w,
			http.StatusServiceUnavailable,
			"request_canceled",
			"list save operation task was canceled",
		)
	case errors.Is(err, listdom.ErrSaveOperationNotFound),
		errors.Is(err, listdom.ErrNotFound):
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, listdom.ErrSaveOperationConflict),
		errors.Is(err, listdom.ErrConflict):
		writeListSaveOperationTaskError(
			w,
			http.StatusServiceUnavailable,
			"save_operation_conflict",
			"list save operation update conflict",
		)
	case errors.Is(err, listdom.ErrSaveOperationNotRetryable):
		writeListSaveOperationTaskError(
			w,
			http.StatusConflict,
			"save_operation_not_retryable",
			err.Error(),
		)
	case errors.Is(err, listdom.ErrInvalidSaveOperation),
		errors.Is(err, listdom.ErrInvalidSaveOperationTransition):
		writeListSaveOperationTaskError(
			w,
			http.StatusUnprocessableEntity,
			"invalid_save_operation",
			err.Error(),
		)
	default:
		writeListSaveOperationTaskError(
			w,
			http.StatusServiceUnavailable,
			"task_execution_failed",
			"list save operation task execution failed",
		)
	}
}
func writeListSaveOperationTaskError(w http.ResponseWriter, statusCode int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(listSaveOperationTaskErrorResponse{
		Error:   code,
		Message: message,
	})
}
func listSaveOperationTaskMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "list save operation failed and can be retried"
	}
	return message
}
