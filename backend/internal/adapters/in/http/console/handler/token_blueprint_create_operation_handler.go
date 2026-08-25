// backend/internal/adapters/in/http/console/handler/token_blueprint_create_operation_handler.go
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
	"time"

	usecase "narratives/internal/application/usecase"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

const (
	tokenBlueprintCreateOperationBasePath = "/token-blueprints/create-operations"

	tokenBlueprintCreateOperationMaxBodyBytes int64 = 1 << 20
)

type TokenBlueprintCreateOperationHandler struct {
	uc *usecase.TokenBlueprintCreateOperationUsecase
}

type NewTokenBlueprintCreateOperationHandlerParams struct {
	UC *usecase.TokenBlueprintCreateOperationUsecase
}

type startTokenBlueprintCreateOperationRequest struct {
	OperationID    string `json:"operationId,omitempty"`
	IdempotencyKey string `json:"idempotencyKey,omitempty"`

	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	Description string `json:"description,omitempty"`
	AssigneeID  string `json:"assigneeId"`

	Icon *tokenBlueprintCreateOperationIconRequest `json:"icon,omitempty"`

	Contents []tokenBlueprintCreateOperationContentRequest `json:"contents,omitempty"`

	MaxRetries int `json:"maxRetries,omitempty"`
}

type tokenBlueprintCreateOperationIconRequest struct {
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

type tokenBlueprintCreateOperationContentRequest struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Type        tbdom.ContentFileType `json:"type"`
	ContentType string                `json:"contentType"`
	Size        int64                 `json:"size"`
}

type registerTokenBlueprintCreateOperationIconRequest struct {
	URL         string `json:"url"`
	ObjectPath  string `json:"objectPath"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

type registerTokenBlueprintCreateOperationContentRequest struct {
	URL         string `json:"url"`
	ObjectPath  string `json:"objectPath"`
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

type tokenBlueprintCreateOperationIconResponse struct {
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`

	URL        string `json:"url,omitempty"`
	ObjectPath string `json:"objectPath,omitempty"`

	Uploaded   bool       `json:"uploaded"`
	UploadedAt *time.Time `json:"uploadedAt,omitempty"`
}

type tokenBlueprintCreateOperationContentResponse struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Type        tbdom.ContentFileType `json:"type"`
	ContentType string                `json:"contentType"`
	Size        int64                 `json:"size"`

	URL        string `json:"url,omitempty"`
	ObjectPath string `json:"objectPath,omitempty"`

	Uploaded   bool       `json:"uploaded"`
	UploadedAt *time.Time `json:"uploadedAt,omitempty"`
}

type tokenBlueprintCreateOperationResponse struct {
	ID               string `json:"id"`
	TokenBlueprintID string `json:"tokenBlueprintId"`
	Status           string `json:"status"`
	ResumeStatus     string `json:"resumeStatus,omitempty"`

	Icon *tokenBlueprintCreateOperationIconResponse `json:"icon,omitempty"`

	Contents []tokenBlueprintCreateOperationContentResponse `json:"contents"`

	ExpectedUploadCount  int `json:"expectedUploadCount"`
	CompletedUploadCount int `json:"completedUploadCount"`

	RetryCount int    `json:"retryCount"`
	MaxRetries int    `json:"maxRetries"`
	LastError  string `json:"lastError,omitempty"`

	Version int64 `json:"version"`

	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	FailedAt    *time.Time `json:"failedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

type tokenBlueprintCreateOperationErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`

	Operation *tokenBlueprintCreateOperationResponse `json:"operation,omitempty"`
}

func NewTokenBlueprintCreateOperationHandler(
	p NewTokenBlueprintCreateOperationHandlerParams,
) http.Handler {
	return &TokenBlueprintCreateOperationHandler{
		uc: p.UC,
	}
}

func (
	h *TokenBlueprintCreateOperationHandler,
) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	if h == nil ||
		h.uc == nil {
		writeTokenBlueprintCreateOperationError(
			w,
			http.StatusInternalServerError,
			"usecase_unavailable",
			"token blueprint create operation usecase is nil",
			nil,
		)
		return
	}

	path := strings.TrimSuffix(
		r.URL.Path,
		"/",
	)

	if path == "" {
		path = "/"
	}

	if path ==
		tokenBlueprintCreateOperationBasePath {
		if r.Method != http.MethodPost {
			writeTokenBlueprintCreateOperationMethodNotAllowed(
				w,
				http.MethodPost,
			)
			return
		}

		h.start(
			w,
			r,
		)
		return
	}

	if !strings.HasPrefix(
		path,
		tokenBlueprintCreateOperationBasePath+"/",
	) {
		writeTokenBlueprintCreateOperationError(
			w,
			http.StatusNotFound,
			"not_found",
			"endpoint not found",
			nil,
		)
		return
	}

	rest := strings.TrimPrefix(
		path,
		tokenBlueprintCreateOperationBasePath+"/",
	)

	parts := strings.Split(
		rest,
		"/",
	)

	if len(parts) == 0 {
		writeTokenBlueprintCreateOperationError(
			w,
			http.StatusNotFound,
			"not_found",
			"endpoint not found",
			nil,
		)
		return
	}

	operationID, err :=
		decodeTokenBlueprintCreateOperationPathID(
			parts[0],
		)
	if err != nil {
		writeTokenBlueprintCreateOperationError(
			w,
			http.StatusBadRequest,
			"invalid_operation_id",
			err.Error(),
			nil,
		)
		return
	}

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeTokenBlueprintCreateOperationMethodNotAllowed(
				w,
				http.MethodGet,
			)
			return
		}

		h.get(
			w,
			r,
			operationID,
		)
		return
	}

	if len(parts) == 2 {
		switch parts[1] {
		case "icon":
			if r.Method != http.MethodPut {
				writeTokenBlueprintCreateOperationMethodNotAllowed(
					w,
					http.MethodPut,
				)
				return
			}

			h.registerIcon(
				w,
				r,
				operationID,
			)
			return

		case "commit":
			if r.Method != http.MethodPost {
				writeTokenBlueprintCreateOperationMethodNotAllowed(
					w,
					http.MethodPost,
				)
				return
			}

			h.commit(
				w,
				r,
				operationID,
			)
			return

		default:
			writeTokenBlueprintCreateOperationError(
				w,
				http.StatusNotFound,
				"not_found",
				"endpoint not found",
				nil,
			)
			return
		}
	}

	if len(parts) == 3 &&
		parts[1] == "contents" {
		if r.Method != http.MethodPut {
			writeTokenBlueprintCreateOperationMethodNotAllowed(
				w,
				http.MethodPut,
			)
			return
		}

		contentID, err :=
			decodeTokenBlueprintCreateOperationPathID(
				parts[2],
			)
		if err != nil {
			writeTokenBlueprintCreateOperationError(
				w,
				http.StatusBadRequest,
				"invalid_content_id",
				err.Error(),
				nil,
			)
			return
		}

		h.registerContent(
			w,
			r,
			operationID,
			contentID,
		)
		return
	}

	writeTokenBlueprintCreateOperationError(
		w,
		http.StatusNotFound,
		"not_found",
		"endpoint not found",
		nil,
	)
}

func (
	h *TokenBlueprintCreateOperationHandler,
) start(
	w http.ResponseWriter,
	r *http.Request,
) {
	companyID :=
		usecase.CompanyIDFromContext(
			r.Context(),
		)
	if companyID == "" {
		writeTokenBlueprintCreateOperationError(
			w,
			http.StatusForbidden,
			"company_context_required",
			"companyId not found in context",
			nil,
		)
		return
	}

	actorID :=
		usecase.MemberIDFromContext(
			r.Context(),
		)
	if actorID == "" {
		writeTokenBlueprintCreateOperationError(
			w,
			http.StatusForbidden,
			"actor_context_required",
			"memberId not found in context",
			nil,
		)
		return
	}

	var req startTokenBlueprintCreateOperationRequest

	if err :=
		decodeTokenBlueprintCreateOperationJSON(
			w,
			r,
			&req,
		); err != nil {
		writeTokenBlueprintCreateOperationError(
			w,
			http.StatusBadRequest,
			"invalid_json",
			err.Error(),
			nil,
		)
		return
	}

	idempotencyKey, err :=
		resolveTokenBlueprintCreateOperationIdempotencyKey(
			r,
			req.IdempotencyKey,
		)
	if err != nil {
		writeTokenBlueprintCreateOperationError(
			w,
			http.StatusBadRequest,
			"invalid_idempotency_key",
			err.Error(),
			nil,
		)
		return
	}

	req.OperationID =
		strings.TrimSpace(
			req.OperationID,
		)

	if req.OperationID != "" {
		if err :=
			validateTokenBlueprintCreateOperationPathID(
				req.OperationID,
			); err != nil {
			writeTokenBlueprintCreateOperationError(
				w,
				http.StatusBadRequest,
				"invalid_operation_id",
				err.Error(),
				nil,
			)
			return
		}
	}

	var icon *tbdom.CreateOperationIcon

	if req.Icon != nil {
		icon =
			&tbdom.CreateOperationIcon{
				FileName: req.Icon.FileName,

				ContentType: req.Icon.ContentType,

				Size: req.Icon.Size,
			}
	}

	contents := make(
		[]tbdom.CreateOperationContent,
		0,
		len(req.Contents),
	)

	for _, content := range req.Contents {
		contents = append(
			contents,
			tbdom.CreateOperationContent{
				ID: content.ID,

				Name: content.Name,

				Type: content.Type,

				ContentType: content.ContentType,

				Size: content.Size,
			},
		)
	}

	operation, err :=
		h.uc.Start(
			r.Context(),
			usecase.StartTokenBlueprintCreateOperationInput{
				OperationID: req.OperationID,

				IdempotencyKey: idempotencyKey,

				Name: req.Name,

				Symbol: req.Symbol,

				BrandID: req.BrandID,

				CompanyID: companyID,

				Description: req.Description,

				AssigneeID: req.AssigneeID,

				ActorID: actorID,

				Icon: icon,

				Contents: contents,

				MaxRetries: req.MaxRetries,
			},
		)
	if err != nil {
		writeTokenBlueprintCreateOperationUsecaseError(
			w,
			err,
			operation,
		)
		return
	}

	if !tokenBlueprintCreateOperationBelongsToActor(
		operation,
		companyID,
		actorID,
	) {
		// IdempotencyKeyはrepository全体で一意のため、
		// 他tenantまたは他actorのOperationを返さない。
		writeTokenBlueprintCreateOperationError(
			w,
			http.StatusNotFound,
			"create_operation_not_found",
			"token blueprint create operation not found",
			nil,
		)
		return
	}

	if operation.ID != "" {
		w.Header().Set(
			"Location",
			tokenBlueprintCreateOperationBasePath+
				"/"+
				url.PathEscape(
					operation.ID,
				),
		)
	}

	writeTokenBlueprintCreateOperationJSON(
		w,
		http.StatusAccepted,
		toTokenBlueprintCreateOperationResponse(
			operation,
		),
	)
}

func (
	h *TokenBlueprintCreateOperationHandler,
) get(
	w http.ResponseWriter,
	r *http.Request,
	operationID string,
) {
	operation, ok :=
		h.loadAuthorizedOperation(
			w,
			r,
			operationID,
			false,
		)
	if !ok {
		return
	}

	writeTokenBlueprintCreateOperationJSON(
		w,
		http.StatusOK,
		toTokenBlueprintCreateOperationResponse(
			operation,
		),
	)
}

func (
	h *TokenBlueprintCreateOperationHandler,
) registerIcon(
	w http.ResponseWriter,
	r *http.Request,
	operationID string,
) {
	_, ok :=
		h.loadAuthorizedOperation(
			w,
			r,
			operationID,
			true,
		)
	if !ok {
		return
	}

	var req registerTokenBlueprintCreateOperationIconRequest

	if err :=
		decodeTokenBlueprintCreateOperationJSON(
			w,
			r,
			&req,
		); err != nil {
		writeTokenBlueprintCreateOperationError(
			w,
			http.StatusBadRequest,
			"invalid_json",
			err.Error(),
			nil,
		)
		return
	}

	operation, err :=
		h.uc.RegisterUploadedIcon(
			r.Context(),
			operationID,
			tbdom.RegisterCreateOperationIconUploadInput{
				URL: req.URL,

				ObjectPath: req.ObjectPath,

				FileName: req.FileName,

				ContentType: req.ContentType,

				Size: req.Size,
			},
		)
	if err != nil {
		writeTokenBlueprintCreateOperationUsecaseError(
			w,
			err,
			operation,
		)
		return
	}

	writeTokenBlueprintCreateOperationJSON(
		w,
		http.StatusOK,
		toTokenBlueprintCreateOperationResponse(
			operation,
		),
	)
}

func (
	h *TokenBlueprintCreateOperationHandler,
) registerContent(
	w http.ResponseWriter,
	r *http.Request,
	operationID string,
	contentID string,
) {
	_, ok :=
		h.loadAuthorizedOperation(
			w,
			r,
			operationID,
			true,
		)
	if !ok {
		return
	}

	var req registerTokenBlueprintCreateOperationContentRequest

	if err :=
		decodeTokenBlueprintCreateOperationJSON(
			w,
			r,
			&req,
		); err != nil {
		writeTokenBlueprintCreateOperationError(
			w,
			http.StatusBadRequest,
			"invalid_json",
			err.Error(),
			nil,
		)
		return
	}

	operation, err :=
		h.uc.RegisterUploadedContent(
			r.Context(),
			operationID,
			tbdom.RegisterCreateOperationContentUploadInput{
				ContentID: contentID,

				URL: req.URL,

				ObjectPath: req.ObjectPath,

				Name: req.Name,

				ContentType: req.ContentType,

				Size: req.Size,
			},
		)
	if err != nil {
		writeTokenBlueprintCreateOperationUsecaseError(
			w,
			err,
			operation,
		)
		return
	}

	writeTokenBlueprintCreateOperationJSON(
		w,
		http.StatusOK,
		toTokenBlueprintCreateOperationResponse(
			operation,
		),
	)
}

func (
	h *TokenBlueprintCreateOperationHandler,
) commit(
	w http.ResponseWriter,
	r *http.Request,
	operationID string,
) {
	_, ok :=
		h.loadAuthorizedOperation(
			w,
			r,
			operationID,
			true,
		)
	if !ok {
		return
	}

	operation, err :=
		h.uc.Commit(
			r.Context(),
			operationID,
		)
	if err != nil {
		writeTokenBlueprintCreateOperationUsecaseError(
			w,
			err,
			operation,
		)
		return
	}

	writeTokenBlueprintCreateOperationJSON(
		w,
		http.StatusAccepted,
		toTokenBlueprintCreateOperationResponse(
			operation,
		),
	)
}

func (
	h *TokenBlueprintCreateOperationHandler,
) loadAuthorizedOperation(
	w http.ResponseWriter,
	r *http.Request,
	operationID string,
	requireActor bool,
) (
	tbdom.CreateOperation,
	bool,
) {
	companyID :=
		usecase.CompanyIDFromContext(
			r.Context(),
		)
	if companyID == "" {
		writeTokenBlueprintCreateOperationError(
			w,
			http.StatusForbidden,
			"company_context_required",
			"companyId not found in context",
			nil,
		)

		return tbdom.CreateOperation{},
			false
	}

	actorID := ""

	if requireActor {
		actorID =
			usecase.MemberIDFromContext(
				r.Context(),
			)

		if actorID == "" {
			writeTokenBlueprintCreateOperationError(
				w,
				http.StatusForbidden,
				"actor_context_required",
				"memberId not found in context",
				nil,
			)

			return tbdom.CreateOperation{},
				false
		}
	}

	operation, err :=
		h.uc.Get(
			r.Context(),
			operationID,
		)
	if err != nil {
		writeTokenBlueprintCreateOperationUsecaseError(
			w,
			err,
			operation,
		)

		return tbdom.CreateOperation{},
			false
	}

	if operation.CompanyID !=
		companyID {
		// tenantの存在情報を漏らさないため404扱い。
		writeTokenBlueprintCreateOperationError(
			w,
			http.StatusNotFound,
			"create_operation_not_found",
			"token blueprint create operation not found",
			nil,
		)

		return tbdom.CreateOperation{},
			false
	}

	if requireActor &&
		operation.ActorID !=
			actorID {
		writeTokenBlueprintCreateOperationError(
			w,
			http.StatusNotFound,
			"create_operation_not_found",
			"token blueprint create operation not found",
			nil,
		)

		return tbdom.CreateOperation{},
			false
	}

	return operation, true
}

func tokenBlueprintCreateOperationBelongsToActor(
	operation tbdom.CreateOperation,
	companyID string,
	actorID string,
) bool {
	return operation.CompanyID ==
		companyID &&
		operation.ActorID ==
			actorID
}

func decodeTokenBlueprintCreateOperationJSON(
	w http.ResponseWriter,
	r *http.Request,
	destination any,
) error {
	if r == nil ||
		r.Body == nil {
		return errors.New(
			"request body is required",
		)
	}

	body := http.MaxBytesReader(
		w,
		r.Body,
		tokenBlueprintCreateOperationMaxBodyBytes,
	)

	decoder :=
		json.NewDecoder(
			body,
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
			return errors.New(
				"request body is required",
			)
		}

		return fmt.Errorf(
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
		return errors.New(
			"request body must contain exactly one JSON object",
		)
	}

	return nil
}

func resolveTokenBlueprintCreateOperationIdempotencyKey(
	r *http.Request,
	bodyValue string,
) (
	string,
	error,
) {
	headerValue := ""

	if r != nil {
		headerValue =
			strings.TrimSpace(
				r.Header.Get(
					"Idempotency-Key",
				),
			)
	}

	bodyValue =
		strings.TrimSpace(
			bodyValue,
		)

	if headerValue != "" &&
		bodyValue != "" &&
		headerValue != bodyValue {
		return "",
			errors.New(
				"Idempotency-Key header and idempotencyKey body value must match",
			)
	}

	value := headerValue

	if value == "" {
		value = bodyValue
	}

	if value == "" {
		return "",
			errors.New(
				"Idempotency-Key header or idempotencyKey body value is required",
			)
	}

	if len(value) > 512 {
		return "",
			errors.New(
				"idempotency key must not exceed 512 characters",
			)
	}

	if strings.ContainsAny(
		value,
		"\r\n\x00",
	) {
		return "",
			errors.New(
				"idempotency key contains invalid characters",
			)
	}

	return value, nil
}

func decodeTokenBlueprintCreateOperationPathID(
	value string,
) (
	string,
	error,
) {
	decoded, err :=
		url.PathUnescape(
			value,
		)
	if err != nil {
		return "",
			errors.New(
				"id contains invalid URL encoding",
			)
	}

	decoded =
		strings.TrimSpace(
			decoded,
		)

	if err :=
		validateTokenBlueprintCreateOperationPathID(
			decoded,
		); err != nil {
		return "", err
	}

	return decoded, nil
}

func validateTokenBlueprintCreateOperationPathID(
	value string,
) error {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return errors.New(
			"id is required",
		)
	}

	if len(value) > 512 {
		return errors.New(
			"id must not exceed 512 characters",
		)
	}

	if strings.Contains(
		value,
		"/",
	) ||
		strings.Contains(
			value,
			"://",
		) ||
		strings.ContainsAny(
			value,
			"\r\n\x00",
		) {
		return errors.New(
			"id is invalid",
		)
	}

	return nil
}

func writeTokenBlueprintCreateOperationUsecaseError(
	w http.ResponseWriter,
	err error,
	operation tbdom.CreateOperation,
) {
	statusCode :=
		tokenBlueprintCreateOperationHTTPStatus(
			err,
			operation,
		)

	errorCode :=
		tokenBlueprintCreateOperationErrorCode(
			err,
			operation,
		)

	message :=
		tokenBlueprintCreateOperationErrorMessage(
			err,
			statusCode,
		)

	var operationResponse *tokenBlueprintCreateOperationResponse

	if strings.TrimSpace(
		operation.ID,
	) != "" {
		value :=
			toTokenBlueprintCreateOperationResponse(
				operation,
			)

		operationResponse =
			&value
	}

	writeTokenBlueprintCreateOperationError(
		w,
		statusCode,
		errorCode,
		message,
		operationResponse,
	)
}

func tokenBlueprintCreateOperationHTTPStatus(
	err error,
	operation tbdom.CreateOperation,
) int {
	switch {
	case err == nil:
		return http.StatusOK

	case errors.Is(
		err,
		context.DeadlineExceeded,
	):
		return http.StatusGatewayTimeout

	case errors.Is(
		err,
		context.Canceled,
	):
		return http.StatusRequestTimeout

	case errors.Is(
		err,
		tbdom.ErrInvalidCreateOperation,
	):
		return http.StatusBadRequest

	case errors.Is(
		err,
		tbdom.ErrInvalid,
	):
		return http.StatusBadRequest

	case errors.Is(
		err,
		tbdom.ErrCreateOperationNotFound,
	):
		return http.StatusNotFound

	case errors.Is(
		err,
		tbdom.ErrNotFound,
	):
		return http.StatusNotFound

	case errors.Is(
		err,
		tbdom.ErrCreateOperationConflict,
	):
		return http.StatusConflict

	case errors.Is(
		err,
		tbdom.ErrCreateOperationIdempotencyConflict,
	):
		return http.StatusConflict

	case errors.Is(
		err,
		tbdom.ErrInvalidCreateOperationTransition,
	):
		return http.StatusConflict

	case errors.Is(
		err,
		tbdom.ErrCreateOperationUploadIncomplete,
	):
		return http.StatusConflict

	case errors.Is(
		err,
		tbdom.ErrCreateOperationAssetNotFound,
	):
		return http.StatusConflict

	case errors.Is(
		err,
		tbdom.ErrCreateOperationAssetConflict,
	):
		return http.StatusConflict

	case errors.Is(
		err,
		tbdom.ErrCreateOperationNotRetryable,
	):
		return http.StatusConflict

	case errors.Is(
		err,
		tbdom.ErrCreateOperationRetryExhausted,
	):
		return http.StatusConflict

	case errors.Is(
		err,
		tbdom.ErrConflict,
	):
		return http.StatusConflict

	case operation.Status ==
		tbdom.CreateOperationStatusFailedRetryable:
		return http.StatusServiceUnavailable

	case operation.Status ==
		tbdom.CreateOperationStatusFailedFatal:
		return http.StatusUnprocessableEntity

	default:
		return http.StatusInternalServerError
	}
}

func tokenBlueprintCreateOperationErrorCode(
	err error,
	operation tbdom.CreateOperation,
) string {
	switch {
	case errors.Is(
		err,
		context.DeadlineExceeded,
	):
		return "deadline_exceeded"

	case errors.Is(
		err,
		context.Canceled,
	):
		return "request_canceled"

	case errors.Is(
		err,
		tbdom.ErrInvalidCreateOperation,
	):
		return "invalid_create_operation"

	case errors.Is(
		err,
		tbdom.ErrInvalid,
	):
		return "invalid_token_blueprint"

	case errors.Is(
		err,
		tbdom.ErrCreateOperationNotFound,
	):
		return "create_operation_not_found"

	case errors.Is(
		err,
		tbdom.ErrNotFound,
	):
		return "token_blueprint_not_found"

	case errors.Is(
		err,
		tbdom.ErrCreateOperationIdempotencyConflict,
	):
		return "idempotency_conflict"

	case errors.Is(
		err,
		tbdom.ErrCreateOperationConflict,
	):
		return "create_operation_conflict"

	case errors.Is(
		err,
		tbdom.ErrInvalidCreateOperationTransition,
	):
		return "invalid_status_transition"

	case errors.Is(
		err,
		tbdom.ErrCreateOperationUploadIncomplete,
	):
		return "upload_incomplete"

	case errors.Is(
		err,
		tbdom.ErrCreateOperationAssetNotFound,
	):
		return "asset_not_found"

	case errors.Is(
		err,
		tbdom.ErrCreateOperationAssetConflict,
	):
		return "asset_conflict"

	case errors.Is(
		err,
		tbdom.ErrCreateOperationNotRetryable,
	):
		return "create_operation_not_retryable"

	case errors.Is(
		err,
		tbdom.ErrCreateOperationRetryExhausted,
	):
		return "retry_exhausted"

	case errors.Is(
		err,
		tbdom.ErrConflict,
	):
		return "conflict"

	case operation.Status ==
		tbdom.CreateOperationStatusFailedRetryable:
		return "create_operation_failed_retryable"

	case operation.Status ==
		tbdom.CreateOperationStatusFailedFatal:
		return "create_operation_failed_fatal"

	default:
		return "internal_error"
	}
}

func tokenBlueprintCreateOperationErrorMessage(
	err error,
	statusCode int,
) string {
	if statusCode >=
		http.StatusInternalServerError &&
		statusCode !=
			http.StatusServiceUnavailable &&
		statusCode !=
			http.StatusGatewayTimeout {
		return "internal server error"
	}

	if err == nil {
		return ""
	}

	return err.Error()
}

func writeTokenBlueprintCreateOperationMethodNotAllowed(
	w http.ResponseWriter,
	allowedMethods ...string,
) {
	w.Header().Set(
		"Allow",
		strings.Join(
			allowedMethods,
			", ",
		),
	)

	writeTokenBlueprintCreateOperationError(
		w,
		http.StatusMethodNotAllowed,
		"method_not_allowed",
		"method not allowed",
		nil,
	)
}

func writeTokenBlueprintCreateOperationError(
	w http.ResponseWriter,
	statusCode int,
	code string,
	message string,
	operation *tokenBlueprintCreateOperationResponse,
) {
	writeTokenBlueprintCreateOperationJSON(
		w,
		statusCode,
		tokenBlueprintCreateOperationErrorResponse{
			Error:     code,
			Message:   message,
			Operation: operation,
		},
	)
}

func writeTokenBlueprintCreateOperationJSON(
	w http.ResponseWriter,
	statusCode int,
	value any,
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
		value,
	)
}

func toTokenBlueprintCreateOperationResponse(
	operation tbdom.CreateOperation,
) tokenBlueprintCreateOperationResponse {
	var icon *tokenBlueprintCreateOperationIconResponse

	if operation.Icon != nil {
		icon =
			&tokenBlueprintCreateOperationIconResponse{
				FileName:    operation.Icon.FileName,
				ContentType: operation.Icon.ContentType,
				Size:        operation.Icon.Size,
				URL:         operation.Icon.URL,
				ObjectPath:  operation.Icon.ObjectPath,
				Uploaded:    operation.Icon.Uploaded,
				UploadedAt:  operation.Icon.UploadedAt,
			}
	}

	contents := make(
		[]tokenBlueprintCreateOperationContentResponse,
		0,
		len(operation.Contents),
	)

	for _, content := range operation.Contents {
		contents = append(
			contents,
			tokenBlueprintCreateOperationContentResponse{
				ID:          content.ID,
				Name:        content.Name,
				Type:        content.Type,
				ContentType: content.ContentType,
				Size:        content.Size,
				URL:         content.URL,
				ObjectPath:  content.ObjectPath,
				Uploaded:    content.Uploaded,
				UploadedAt:  content.UploadedAt,
			},
		)
	}

	return tokenBlueprintCreateOperationResponse{
		ID:                   operation.ID,
		TokenBlueprintID:     operation.TokenBlueprintID,
		Status:               string(operation.Status),
		ResumeStatus:         string(operation.ResumeStatus),
		Icon:                 icon,
		Contents:             contents,
		ExpectedUploadCount:  operation.ExpectedUploadCount(),
		CompletedUploadCount: operation.CompletedUploadCount(),
		RetryCount:           operation.RetryCount,
		MaxRetries:           operation.MaxRetries,
		LastError:            operation.LastError,
		Version:              operation.Version,
		CreatedAt:            operation.CreatedAt,
		UpdatedAt:            operation.UpdatedAt,
		FailedAt:             operation.FailedAt,
		CompletedAt:          operation.CompletedAt,
	}
}
