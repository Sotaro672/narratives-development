// backend/internal/adapters/in/http/mall/handler/resale_error.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	resaledom "narratives/internal/domain/resale"
)

func writeResaleErr(w http.ResponseWriter, err error) {
	writeJSON(w, resaleHTTPStatus(err), map[string]string{
		"error": resaleErrorMessage(err),
	})
}

func resaleHTTPStatus(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	message := err.Error()

	switch {
	case errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		return http.StatusRequestTimeout

	case usecase.IsResaleServiceSuspended(err):
		return http.StatusForbidden

	case errors.Is(err, resaledom.ErrNotFound),
		errors.Is(err, resaledom.ErrConditionImageNotFound):
		return http.StatusNotFound

	case errors.Is(err, resaledom.ErrConflict),
		errors.Is(err, resaledom.ErrConditionImageConflict),
		errors.Is(err, resaledom.ErrSoldResaleCannotBeDeleted):
		return http.StatusConflict

	case errors.Is(err, resaledom.ErrInvalidID),
		errors.Is(err, resaledom.ErrInvalidStatus),
		errors.Is(err, resaledom.ErrInvalidAssetID),
		errors.Is(err, resaledom.ErrInvalidTokenBlueprintID),
		errors.Is(err, resaledom.ErrInvalidProductID),
		errors.Is(err, resaledom.ErrInvalidBrandID),
		errors.Is(err, resaledom.ErrInvalidProductBlueprintID),
		errors.Is(err, resaledom.ErrInvalidAvatarID),
		errors.Is(err, resaledom.ErrInvalidPrice),
		errors.Is(err, resaledom.ErrInvalidCondition),
		errors.Is(err, resaledom.ErrInvalidDescription),
		errors.Is(err, resaledom.ErrInvalidCreatedBy),
		errors.Is(err, resaledom.ErrInvalidCreatedAt),
		errors.Is(err, resaledom.ErrInvalidUpdatedAt),
		errors.Is(err, resaledom.ErrInvalidUpdatedBy),
		errors.Is(err, resaledom.ErrEmptyImageID),
		errors.Is(err, resaledom.ErrInvalidImageID),
		errors.Is(err, resaledom.ErrInvalidConditionImageID),
		errors.Is(err, resaledom.ErrInvalidConditionImageResaleID),
		errors.Is(err, resaledom.ErrInvalidConditionImageURL),
		errors.Is(err, resaledom.ErrInvalidConditionImageDisplayOrder),
		errors.Is(err, resaledom.ErrInvalidConditionImageCreatedAt),
		errors.Is(err, resaledom.ErrInvalidConditionImageCreatedBy),
		errors.Is(err, resaledom.ErrInvalidConditionImageUpdatedAt),
		errors.Is(err, resaledom.ErrInvalidConditionImageUpdatedBy),
		message == "invalid_image_id":
		return http.StatusBadRequest

	case strings.Contains(message, "not supported"):
		return http.StatusNotImplemented

	default:
		return http.StatusInternalServerError
	}
}

func resaleErrorMessage(err error) string {
	if err == nil {
		return "internal_error"
	}

	message := err.Error()

	switch {
	case errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		return "request_timeout"

	case usecase.IsResaleServiceSuspended(err):
		return "resale_service_suspended"

	case strings.Contains(message, "not supported"):
		return "not_implemented"

	case resaleHTTPStatus(err) == http.StatusInternalServerError:
		return "internal_error"

	default:
		return message
	}
}
