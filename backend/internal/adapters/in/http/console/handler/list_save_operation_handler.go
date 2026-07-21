// backend\internal\adapters\in\http\console\handler\list_save_operation_handler.go
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
	"time"
)

const (
	listSaveOperationBasePath           = "/lists/save-operations"
	listSaveOperationMaxBodyBytes int64 = 1 << 20
)

type ListSaveOperationHandler struct {
	uc *usecase.ListSaveOperationUsecase
}
type NewListSaveOperationHandlerParams struct {
	UC *usecase.ListSaveOperationUsecase
}
type startListSaveOperationRequest struct {
	OperationID    string                       `json:"operationId,omitempty"`
	IdempotencyKey string                       `json:"idempotencyKey,omitempty"`
	ListID         string                       `json:"listId,omitempty"`
	Type           listdom.SaveOperationType    `json:"type"`
	TargetList     listdom.List                 `json:"targetList"`
	NewImages      []listdom.SaveOperationImage `json:"newImages,omitempty"`
	DeleteImageIDs []string                     `json:"deleteImageIds,omitempty"`
	PrimaryImageID *string                      `json:"primaryImageId,omitempty"`
	MaxRetries     int                          `json:"maxRetries,omitempty"`
}
type listSaveOperationImageResponse struct {
	ImageID      string `json:"imageId"`
	URL          string `json:"url"`
	StoragePath  string `json:"storagePath"`
	DisplayOrder int    `json:"displayOrder"`
}
type listSaveOperationPayloadResponse struct {
	TargetList             listdom.List                     `json:"targetList"`
	PreviousList           *listdom.List                    `json:"previousList,omitempty"`
	NewImages              []listSaveOperationImageResponse `json:"newImages"`
	DeleteImageIDs         []string                         `json:"deleteImageIds"`
	PreviousImages         []listdom.ListImage              `json:"previousImages"`
	PrimaryImageID         string                           `json:"primaryImageId"`
	PreviousPrimaryImageID string                           `json:"previousPrimaryImageId"`
}
type listSaveOperationProgressResponse struct {
	UploadedImageIDs        []string `json:"uploadedImageIds"`
	RegisteredImageIDs      []string `json:"registeredImageIds"`
	DeletedImageIDs         []string `json:"deletedImageIds"`
	CompensatedStoragePaths []string `json:"compensatedStoragePaths"`
	ListUpdated             bool     `json:"listUpdated"`
	PrimaryImageUpdated     bool     `json:"primaryImageUpdated"`
}
type listSaveOperationResponse struct {
	ID             string                            `json:"id"`
	IdempotencyKey string                            `json:"idempotencyKey"`
	ListID         string                            `json:"listId"`
	Type           string                            `json:"type"`
	Status         string                            `json:"status"`
	ResumeStatus   string                            `json:"resumeStatus,omitempty"`
	Payload        listSaveOperationPayloadResponse  `json:"payload"`
	Progress       listSaveOperationProgressResponse `json:"progress"`
	RetryCount     int                               `json:"retryCount"`
	MaxRetries     int                               `json:"maxRetries"`
	LastError      string                            `json:"lastError,omitempty"`
	Version        int64                             `json:"version"`
	CreatedAt      time.Time                         `json:"createdAt"`
	UpdatedAt      time.Time                         `json:"updatedAt"`
	FailedAt       *time.Time                        `json:"failedAt,omitempty"`
	CompletedAt    *time.Time                        `json:"completedAt,omitempty"`
	CompensatedAt  *time.Time                        `json:"compensatedAt,omitempty"`
}
type listSaveOperationErrorResponse struct {
	Error     string                     `json:"error"`
	Message   string                     `json:"message"`
	Operation *listSaveOperationResponse `json:"operation,omitempty"`
}

func NewListSaveOperationHandler(p NewListSaveOperationHandlerParams) http.Handler {
	return &ListSaveOperationHandler{
		uc: p.UC,
	}
}
func (h *ListSaveOperationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h == nil || h.uc == nil {
		writeListSaveOperationError(w, http.StatusInternalServerError, "usecase_unavailable", "list save operation usecase is nil", nil)
		return
	}
	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == "" {
		path = "/"
	}
	if path == listSaveOperationBasePath {
		if r.Method != http.MethodPost {
			writeListSaveOperationMethodNotAllowed(w, http.MethodPost)
			return
		}
		h.start(w, r)
		return
	}
	if !strings.HasPrefix(path, listSaveOperationBasePath+"/") {
		writeListSaveOperationError(w, http.StatusNotFound, "not_found", "endpoint not found", nil)
		return
	}
	rest := strings.TrimPrefix(path, listSaveOperationBasePath+"/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || len(parts) > 2 {
		writeListSaveOperationError(w, http.StatusNotFound, "not_found", "endpoint not found", nil)
		return
	}
	operationID, err := decodeListSaveOperationPathID(parts[0])
	if err != nil {
		writeListSaveOperationError(w, http.StatusBadRequest, "invalid_operation_id", err.Error(), nil)
		return
	}
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeListSaveOperationMethodNotAllowed(w, http.MethodGet)
			return
		}
		h.get(w, r, operationID)
		return
	}
	switch parts[1] {
	case "retry":
		if r.Method != http.MethodPost {
			writeListSaveOperationMethodNotAllowed(w, http.MethodPost)
			return
		}
		h.retry(w, r, operationID)
	case "compensate":
		if r.Method != http.MethodPost {
			writeListSaveOperationMethodNotAllowed(w, http.MethodPost)
			return
		}
		h.compensate(w, r, operationID)
	default:
		writeListSaveOperationError(w, http.StatusNotFound, "not_found", "endpoint not found", nil)
	}
}
func (h *ListSaveOperationHandler) start(w http.ResponseWriter, r *http.Request) {
	var req startListSaveOperationRequest
	if err := decodeListSaveOperationJSON(w, r, &req); err != nil {
		writeListSaveOperationError(w, http.StatusBadRequest, "invalid_json", err.Error(), nil)
		return
	}
	idempotencyKey, err := resolveListSaveOperationIdempotencyKey(r, req.IdempotencyKey)
	if err != nil {
		writeListSaveOperationError(w, http.StatusBadRequest, "invalid_idempotency_key", err.Error(), nil)
		return
	}
	req.OperationID = strings.TrimSpace(req.OperationID)
	req.ListID = strings.TrimSpace(req.ListID)
	if req.OperationID != "" {
		if err := validateListSaveOperationPathID(req.OperationID); err != nil {
			writeListSaveOperationError(w, http.StatusBadRequest, "invalid_operation_id", err.Error(), nil)
			return
		}
	}
	operation, err := h.uc.Start(
		r.Context(),
		usecase.StartListSaveOperationInput{
			OperationID:    req.OperationID,
			IdempotencyKey: idempotencyKey,
			ListID:         req.ListID,
			Type:           req.Type,
			TargetList:     req.TargetList,
			NewImages:      req.NewImages,
			DeleteImageIDs: req.DeleteImageIDs,
			PrimaryImageID: req.PrimaryImageID,
			MaxRetries:     req.MaxRetries,
		},
	)
	if err != nil {
		writeListSaveOperationUsecaseError(w, err, operation)
		return
	}
	if operation.ID != "" {
		w.Header().Set(
			"Location",
			listSaveOperationBasePath+"/"+url.PathEscape(operation.ID),
		)
	}
	writeListSaveOperationJSON(w, http.StatusAccepted, toListSaveOperationResponse(operation))
}
func (h *ListSaveOperationHandler) get(w http.ResponseWriter, r *http.Request, operationID string) {
	operation, err := h.uc.Get(r.Context(), operationID)
	if err != nil {
		writeListSaveOperationUsecaseError(w, err, operation)
		return
	}
	writeListSaveOperationJSON(w, http.StatusOK, toListSaveOperationResponse(operation))
}
func (h *ListSaveOperationHandler) retry(w http.ResponseWriter, r *http.Request, operationID string) {
	operation, err := h.uc.Retry(r.Context(), operationID)
	if err != nil {
		writeListSaveOperationUsecaseError(w, err, operation)
		return
	}
	writeListSaveOperationJSON(w, http.StatusOK, toListSaveOperationResponse(operation))
}
func (h *ListSaveOperationHandler) compensate(w http.ResponseWriter, r *http.Request, operationID string) {
	operation, err := h.uc.Compensate(r.Context(), operationID)
	if err != nil {
		writeListSaveOperationUsecaseError(w, err, operation)
		return
	}
	writeListSaveOperationJSON(w, http.StatusOK, toListSaveOperationResponse(operation))
}
func decodeListSaveOperationJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	if r == nil || r.Body == nil {
		return errors.New("request body is required")
	}
	body := http.MaxBytesReader(w, r.Body, listSaveOperationMaxBodyBytes)
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is required")
		}
		return fmt.Errorf("decode request body: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain exactly one JSON object")
	}
	return nil
}
func resolveListSaveOperationIdempotencyKey(r *http.Request, bodyValue string) (string, error) {
	headerValue := ""
	if r != nil {
		headerValue = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	}
	bodyValue = strings.TrimSpace(bodyValue)
	if headerValue != "" && bodyValue != "" && headerValue != bodyValue {
		return "", errors.New("Idempotency-Key header and idempotencyKey body value must match")
	}
	value := headerValue
	if value == "" {
		value = bodyValue
	}
	if value == "" {
		return "", errors.New("Idempotency-Key header or idempotencyKey body value is required")
	}
	if len(value) > 512 {
		return "", errors.New("idempotency key must not exceed 512 characters")
	}
	if strings.ContainsAny(value, "\r\n\x00") {
		return "", errors.New("idempotency key contains invalid characters")
	}
	return value, nil
}
func decodeListSaveOperationPathID(value string) (string, error) {
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return "", errors.New("operationId contains invalid URL encoding")
	}
	decoded = strings.TrimSpace(decoded)
	if err := validateListSaveOperationPathID(decoded); err != nil {
		return "", err
	}
	return decoded, nil
}
func validateListSaveOperationPathID(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("operationId is required")
	}
	if strings.Contains(value, "/") || strings.Contains(value, "://") {
		return errors.New("operationId is invalid")
	}
	if strings.ContainsAny(value, "\r\n\x00") {
		return errors.New("operationId contains invalid characters")
	}
	return nil
}
func writeListSaveOperationUsecaseError(w http.ResponseWriter, err error, operation listdom.SaveOperation) {
	statusCode := listSaveOperationHTTPStatus(err, operation)
	errorCode := listSaveOperationErrorCode(err, operation)
	message := listSaveOperationErrorMessage(err, statusCode)
	var operationResponse *listSaveOperationResponse
	if strings.TrimSpace(operation.ID) != "" {
		value := toListSaveOperationResponse(operation)
		operationResponse = &value
	}
	writeListSaveOperationError(w, statusCode, errorCode, message, operationResponse)
}
func listSaveOperationHTTPStatus(err error, operation listdom.SaveOperation) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout
	case errors.Is(err, context.Canceled):
		return http.StatusRequestTimeout
	case errors.Is(err, listdom.ErrInvalidSaveOperation):
		return http.StatusBadRequest
	case errors.Is(err, listdom.ErrInvalidListImageListID):
		return http.StatusBadRequest
	case errors.Is(err, listdom.ErrInvalidListImageID):
		return http.StatusBadRequest
	case errors.Is(err, listdom.ErrInvalidListImageURL):
		return http.StatusBadRequest
	case errors.Is(err, listdom.ErrInvalidListImageCreatedBy):
		return http.StatusBadRequest
	case errors.Is(err, listdom.ErrInvalidListImageDisplayOrder):
		return http.StatusBadRequest
	case errors.Is(err, listdom.ErrSaveOperationNotFound):
		return http.StatusNotFound
	case errors.Is(err, listdom.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, listdom.ErrSaveOperationConflict):
		return http.StatusConflict
	case errors.Is(err, listdom.ErrSaveOperationIdempotencyConflict):
		return http.StatusConflict
	case errors.Is(err, listdom.ErrSaveOperationNotRetryable):
		return http.StatusConflict
	case errors.Is(err, listdom.ErrSaveOperationNotCompensatable):
		return http.StatusConflict
	case errors.Is(err, listdom.ErrConflict):
		return http.StatusConflict
	case operation.Status == listdom.SaveOperationStatusFailedRetryable:
		return http.StatusServiceUnavailable
	case operation.Status == listdom.SaveOperationStatusFailedFatal:
		return http.StatusUnprocessableEntity
	case operation.Status == listdom.SaveOperationStatusCompensated:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}
func listSaveOperationErrorCode(err error, operation listdom.SaveOperation) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "request_canceled"
	case errors.Is(err, listdom.ErrInvalidSaveOperation):
		return "invalid_save_operation"
	case errors.Is(err, listdom.ErrInvalidListImageListID):
		return "invalid_list_id"
	case errors.Is(err, listdom.ErrInvalidListImageID):
		return "invalid_image_id"
	case errors.Is(err, listdom.ErrInvalidListImageURL):
		return "invalid_image_url"
	case errors.Is(err, listdom.ErrInvalidListImageCreatedBy):
		return "invalid_created_by"
	case errors.Is(err, listdom.ErrInvalidListImageDisplayOrder):
		return "invalid_display_order"
	case errors.Is(err, listdom.ErrSaveOperationNotFound):
		return "save_operation_not_found"
	case errors.Is(err, listdom.ErrNotFound):
		return "not_found"
	case errors.Is(err, listdom.ErrSaveOperationIdempotencyConflict):
		return "idempotency_conflict"
	case errors.Is(err, listdom.ErrSaveOperationConflict):
		return "save_operation_conflict"
	case errors.Is(err, listdom.ErrSaveOperationNotRetryable):
		return "save_operation_not_retryable"
	case errors.Is(err, listdom.ErrSaveOperationNotCompensatable):
		return "save_operation_not_compensatable"
	case errors.Is(err, listdom.ErrConflict):
		return "conflict"
	case operation.Status == listdom.SaveOperationStatusFailedRetryable:
		return "save_operation_failed_retryable"
	case operation.Status == listdom.SaveOperationStatusFailedFatal:
		return "save_operation_failed_fatal"
	case operation.Status == listdom.SaveOperationStatusCompensated:
		return "save_operation_compensated"
	default:
		return "internal_error"
	}
}
func listSaveOperationErrorMessage(err error, statusCode int) string {
	if statusCode >= http.StatusInternalServerError && statusCode != http.StatusServiceUnavailable && statusCode != http.StatusGatewayTimeout {
		return "internal server error"
	}
	if err == nil {
		return ""
	}
	return err.Error()
}
func writeListSaveOperationMethodNotAllowed(w http.ResponseWriter, allowedMethods ...string) {
	w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
	writeListSaveOperationError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", nil)
}
func writeListSaveOperationError(w http.ResponseWriter, statusCode int, code string, message string, operation *listSaveOperationResponse) {
	writeListSaveOperationJSON(
		w,
		statusCode,
		listSaveOperationErrorResponse{
			Error:     code,
			Message:   message,
			Operation: operation,
		},
	)
}
func writeListSaveOperationJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}
func toListSaveOperationResponse(operation listdom.SaveOperation) listSaveOperationResponse {
	newImages := make([]listSaveOperationImageResponse, 0, len(operation.Payload.NewImages))
	for _, image := range operation.Payload.NewImages {
		newImages = append(
			newImages,
			listSaveOperationImageResponse{
				ImageID:      image.ImageID,
				URL:          image.URL,
				StoragePath:  image.StoragePath,
				DisplayOrder: image.DisplayOrder,
			},
		)
	}
	deleteImageIDs := append([]string{}, operation.Payload.DeleteImageIDs...)
	previousImages := append([]listdom.ListImage{}, operation.Payload.PreviousImages...)
	uploadedImageIDs := append([]string{}, operation.Progress.UploadedImageIDs...)
	registeredImageIDs := append([]string{}, operation.Progress.RegisteredImageIDs...)
	deletedImageIDs := append([]string{}, operation.Progress.DeletedImageIDs...)
	compensatedStoragePaths := append([]string{}, operation.Progress.CompensatedStoragePaths...)
	return listSaveOperationResponse{
		ID:             operation.ID,
		IdempotencyKey: operation.IdempotencyKey,
		ListID:         operation.ListID,
		Type:           string(operation.Type),
		Status:         string(operation.Status),
		ResumeStatus:   string(operation.ResumeStatus),
		Payload: listSaveOperationPayloadResponse{
			TargetList:             operation.Payload.TargetList,
			PreviousList:           operation.Payload.PreviousList,
			NewImages:              newImages,
			DeleteImageIDs:         deleteImageIDs,
			PreviousImages:         previousImages,
			PrimaryImageID:         operation.Payload.PrimaryImageID,
			PreviousPrimaryImageID: operation.Payload.PreviousPrimaryImageID,
		},
		Progress: listSaveOperationProgressResponse{
			UploadedImageIDs:        uploadedImageIDs,
			RegisteredImageIDs:      registeredImageIDs,
			DeletedImageIDs:         deletedImageIDs,
			CompensatedStoragePaths: compensatedStoragePaths,
			ListUpdated:             operation.Progress.ListUpdated,
			PrimaryImageUpdated:     operation.Progress.PrimaryImageUpdated,
		},
		RetryCount:    operation.RetryCount,
		MaxRetries:    operation.MaxRetries,
		LastError:     operation.LastError,
		Version:       operation.Version,
		CreatedAt:     operation.CreatedAt,
		UpdatedAt:     operation.UpdatedAt,
		FailedAt:      operation.FailedAt,
		CompletedAt:   operation.CompletedAt,
		CompensatedAt: operation.CompensatedAt,
	}
}
