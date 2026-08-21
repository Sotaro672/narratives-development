// backend/internal/adapters/in/http/mall/handler/shipping_quote_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	listdom "narratives/internal/domain/list"
	modeldom "narratives/internal/domain/model"
	shippingaddressdom "narratives/internal/domain/shippingAddress"
	transportationdom "narratives/internal/domain/transportation"
)

const maxShippingQuoteBodyBytes int64 = 1 << 20 // 1 MiB

type ShippingQuoteHandler struct {
	uc *usecase.ShippingQuoteUsecase
}

func NewShippingQuoteHandler(
	uc *usecase.ShippingQuoteUsecase,
) http.Handler {
	return &ShippingQuoteHandler{
		uc: uc,
	}
}

type shippingQuoteItemRequest struct {
	ListID  string `json:"listId"`
	ModelID string `json:"modelId"`
	Qty     int    `json:"qty"`
}

type shippingQuoteRequest struct {
	Items []shippingQuoteItemRequest `json:"items"`

	ShippingAddressID string `json:"shippingAddressId"`
}

type shippingQuoteItemResponse struct {
	ListID  string `json:"listId"`
	ModelID string `json:"modelId"`
	Qty     int    `json:"qty"`

	Carrier string `json:"carrier"`

	TransportationID string `json:"transportationId,omitempty"`

	Size int `json:"size"`

	UnitAmount int64 `json:"unitAmount"`
	Amount     int64 `json:"amount"`

	Currency string `json:"currency"`
}

type shippingQuoteResponse struct {
	Items []shippingQuoteItemResponse `json:"items"`

	ShippingAmount int64 `json:"shippingAmount"`

	Currency string `json:"currency"`
}

func (h *ShippingQuoteHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	path := strings.TrimSuffix(
		r.URL.Path,
		"/",
	)

	switch {
	case r.Method == http.MethodOptions &&
		path == "/mall/me/shipping-quotes":
		w.WriteHeader(
			http.StatusNoContent,
		)
		return

	case r.Method == http.MethodPost &&
		path == "/mall/me/shipping-quotes":
		h.post(
			w,
			r,
		)
		return

	default:
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "not_found",
			},
		)
	}
}

func (h *ShippingQuoteHandler) post(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		h.requireUID(
			w,
			r,
		)
	if !ok {
		return
	}

	if !h.requireUsecase(w) {
		return
	}

	var request shippingQuoteRequest

	if err :=
		decodeShippingQuoteJSON(
			w,
			r,
			&request,
		); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid_json",
			},
		)
		return
	}

	if request.ShippingAddressID == "" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "shippingAddressId is required",
			},
		)
		return
	}

	if len(request.Items) == 0 {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "items is required",
			},
		)
		return
	}

	items :=
		make(
			[]shippingQuoteItemResponse,
			0,
			len(request.Items),
		)

	var shippingAmount int64

	for _, item := range request.Items {
		if item.ListID == "" {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": "listId is required",
				},
			)
			return
		}

		if item.ModelID == "" {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": "modelId is required",
				},
			)
			return
		}

		if item.Qty <= 0 {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": "qty must be greater than zero",
				},
			)
			return
		}

		quote, err :=
			h.uc.Quote(
				r.Context(),
				usecase.ShippingQuoteInput{
					UserID: userID,

					ListID: item.ListID,

					ModelID: item.ModelID,

					DestinationShippingAddressID: request.ShippingAddressID,
				},
			)
		if err != nil {
			writeShippingQuoteErr(
				w,
				err,
			)
			return
		}

		if quote.Amount < 0 {
			writeShippingQuoteErr(
				w,
				transportationdom.ErrInvalidRateAmount,
			)
			return
		}

		lineAmount :=
			quote.Amount *
				int64(item.Qty)

		if quote.Amount > 0 &&
			lineAmount/quote.Amount != int64(item.Qty) {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": "shipping amount overflow",
				},
			)
			return
		}

		nextShippingAmount :=
			shippingAmount +
				lineAmount

		if lineAmount > 0 &&
			nextShippingAmount < shippingAmount {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": "shipping amount overflow",
				},
			)
			return
		}

		shippingAmount =
			nextShippingAmount

		items =
			append(
				items,
				shippingQuoteItemResponse{
					ListID: quote.ListID,

					ModelID: quote.ModelID,

					Qty: item.Qty,

					Carrier: string(
						quote.TransportationOption,
					),

					TransportationID: quote.TransportationID,

					Size: quote.Size,

					UnitAmount: quote.Amount,

					Amount: lineAmount,

					Currency: quote.Currency,
				},
			)
	}

	writeJSON(
		w,
		http.StatusOK,
		shippingQuoteResponse{
			Items: items,

			ShippingAmount: shippingAmount,

			Currency: "JPY",
		},
	)
}

func (h *ShippingQuoteHandler) requireUsecase(
	w http.ResponseWriter,
) bool {
	if h != nil &&
		h.uc != nil {
		return true
	}

	writeJSON(
		w,
		http.StatusServiceUnavailable,
		map[string]string{
			"error": "shipping_quote_usecase_not_initialized",
		},
	)

	return false
}

func (h *ShippingQuoteHandler) requireUID(
	w http.ResponseWriter,
	r *http.Request,
) (string, bool) {
	uid, ok :=
		middleware.CurrentUserUID(
			r,
		)

	if ok &&
		uid != "" {
		return uid, true
	}

	writeJSON(
		w,
		http.StatusUnauthorized,
		map[string]string{
			"error": "unauthorized",
		},
	)

	return "", false
}

func decodeShippingQuoteJSON(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
) error {
	r.Body =
		http.MaxBytesReader(
			w,
			r.Body,
			maxShippingQuoteBodyBytes,
		)

	decoder :=
		json.NewDecoder(
			r.Body,
		)

	decoder.DisallowUnknownFields()

	if err :=
		decoder.Decode(
			dst,
		); err != nil {
		return err
	}

	var trailing any

	err :=
		decoder.Decode(
			&trailing,
		)

	if !errors.Is(
		err,
		io.EOF,
	) {
		if err == nil {
			return errors.New(
				"request body must contain exactly one JSON value",
			)
		}

		return err
	}

	return nil
}

func writeShippingQuoteErr(
	w http.ResponseWriter,
	err error,
) {
	if err == nil {
		return
	}

	statusCode :=
		http.StatusInternalServerError

	switch {
	case errors.Is(
		err,
		listdom.ErrNotFound,
	),
		errors.Is(
			err,
			shippingaddressdom.ErrNotFound,
		),
		errors.Is(
			err,
			transportationdom.ErrNotFound,
		):
		statusCode =
			http.StatusNotFound

	case strings.HasPrefix(
		err.Error(),
		"usecase: invalid request",
	):
		statusCode =
			http.StatusBadRequest

	case errors.Is(
		err,
		listdom.ErrInvalidTransportationOption,
	),
		errors.Is(
			err,
			modeldom.ErrInvalidShippingPackage,
		),
		errors.Is(
			err,
			transportationdom.ErrInvalidCarrier,
		),
		errors.Is(
			err,
			transportationdom.ErrInvalidPackage,
		),
		errors.Is(
			err,
			transportationdom.ErrInvalidAddress,
		),
		errors.Is(
			err,
			transportationdom.ErrUnsupportedCountry,
		),
		errors.Is(
			err,
			transportationdom.ErrInvalidPrefectureCode,
		),
		errors.Is(
			err,
			transportationdom.ErrInvalidRateAmount,
		):
		statusCode =
			http.StatusBadRequest

	case errors.Is(
		err,
		transportationdom.ErrYamatoPackageTooLarge,
	),
		errors.Is(
			err,
			transportationdom.ErrSagawaPackageTooLarge,
		),
		errors.Is(
			err,
			transportationdom.ErrPostPackageTooLarge,
		),
		errors.Is(
			err,
			transportationdom.ErrPostPackageTooHeavy,
		),
		errors.Is(
			err,
			transportationdom.ErrSagawaIslandSurchargeRequired,
		),
		errors.Is(
			err,
			transportationdom.ErrYamatoRateNotFound,
		),
		errors.Is(
			err,
			transportationdom.ErrSagawaRateNotFound,
		),
		errors.Is(
			err,
			transportationdom.ErrPostRateNotFound,
		):
		statusCode =
			http.StatusUnprocessableEntity

	case errors.Is(
		err,
		transportationdom.ErrCarrierRateNotConfigured,
	),
		errors.Is(
			err,
			transportationdom.ErrServiceUnavailable,
		),
		strings.HasPrefix(
			err.Error(),
			"usecase: operation not supported",
		):
		statusCode =
			http.StatusServiceUnavailable
	}

	writeJSON(
		w,
		statusCode,
		map[string]string{
			"error": err.Error(),
		},
	)
}
