// backend/internal/adapters/in/http/mall/handler/avatar_review_handler.go
package mallHandler

import (
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	avatarreviewdom "narratives/internal/domain/avatar_review"
)

const (
	mallMeAvatarReviewsPath = "/mall/me/avatar-reviews"
)

// AvatarReviewHandler handles post-transfer Avatar reviews in Mall.
//
// Avatar Review is available only for completed Avatar-to-Avatar Resale
// transactions.
//
// Supported:
//
//	POST /mall/me/avatar-reviews
//
// Reviewer Avatar identity is always resolved from AvatarContextMiddleware.
// It is never accepted from the request body.
//
// Reviewee Avatar identity is also never accepted from the request body.
// AvatarReviewUsecase resolves the seller from the authoritative Trade and
// Order snapshots.
type AvatarReviewHandler struct {
	uc *usecase.AvatarReviewUsecase
}

func NewAvatarReviewHandler(
	uc *usecase.AvatarReviewUsecase,
) http.Handler {
	return &AvatarReviewHandler{
		uc: uc,
	}
}

// ServeHTTP is the routing entry point.
//
// Supported:
//
//	POST /mall/me/avatar-reviews
//
// Request body:
//
//	{
//	  "orderId": "...",
//	  "orderItemIndex": 0,
//	  "evaluation": "good",
//	  "comment": "丁寧に対応していただきました"
//	}
//
// evaluation:
//
//	"good"
//	"disappointed"
//
// orderId + orderItemIndex identify the Trade indirectly.
// tradeId and revieweeAvatarId are intentionally not accepted from the client.
func (h *AvatarReviewHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	if h == nil || h.uc == nil {
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			map[string]string{
				"error": "avatar_review_not_configured",
			},
		)
		return
	}

	if r.URL.Path != mallMeAvatarReviewsPath &&
		r.URL.Path != mallMeAvatarReviewsPath+"/" {
		notFound(w)
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.create(w, r)

	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)

	default:
		methodNotAllowed(w)
	}
}

// ============================================================
// Request / Response
// ============================================================

type createAvatarReviewRequest struct {
	OrderID string `json:"orderId"`

	// Pointer is intentional.
	//
	// item index 0 is valid, therefore a plain int cannot distinguish:
	//
	//	orderItemIndex omitted
	//
	// from:
	//
	//	orderItemIndex: 0
	OrderItemIndex *int `json:"orderItemIndex"`

	Evaluation avatarreviewdom.Evaluation `json:"evaluation"`
	Comment    string                     `json:"comment"`
}

// ============================================================
// Create
// ============================================================

// POST /mall/me/avatar-reviews
//
// Creates one immutable buyer-to-seller Avatar Review.
//
// The authenticated Avatar is used as ReviewerAvatarID.
//
// Security-sensitive values are not accepted from the client:
//
//   - tradeId
//   - reviewerAvatarId
//   - revieweeAvatarId
//
// AvatarReviewUsecase resolves and validates:
//
//   - Trade from orderId + orderItemIndex
//   - authenticated Avatar is the Trade buyer
//   - Trade seller is an Avatar
//   - Order belongs to the same buyer
//   - Order item is a Resale item
//   - Trade seller matches Order SellerSnapshot
//   - token transfer has completed
//
// One Trade can have at most one Avatar Review.
func (h *AvatarReviewHandler) create(
	w http.ResponseWriter,
	r *http.Request,
) {
	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	var req createAvatarReviewRequest
	if err := readJSON(r, &req); err != nil {
		badRequest(
			w,
			"invalid_json",
		)
		return
	}

	req.OrderID = strings.TrimSpace(
		req.OrderID,
	)

	if req.OrderID == "" {
		badRequest(
			w,
			"order_id_required",
		)
		return
	}

	if req.OrderItemIndex == nil {
		badRequest(
			w,
			"order_item_index_required",
		)
		return
	}

	if *req.OrderItemIndex < 0 {
		badRequest(
			w,
			"invalid_order_item_index",
		)
		return
	}

	created, err := h.uc.Create(
		r.Context(),
		usecase.CreateAvatarReviewInput{
			OrderID:        req.OrderID,
			OrderItemIndex: *req.OrderItemIndex,

			ReviewerAvatarID: avatarID,

			Evaluation: req.Evaluation,
			Comment:    req.Comment,
		},
	)
	if err != nil {
		writeAvatarReviewErr(
			w,
			err,
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		map[string]any{
			"data": created,
		},
	)
}

// ============================================================
// Error mapping
// ============================================================

func writeAvatarReviewErr(
	w http.ResponseWriter,
	err error,
) {
	if err == nil {
		return
	}

	switch {
	// --------------------------------------------------------
	// Infrastructure / configuration
	// --------------------------------------------------------

	case errors.Is(
		err,
		usecase.ErrAvatarReviewUsecaseNotConfigured,
	):
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			map[string]string{
				"error": "avatar_review_not_configured",
			},
		)

	// --------------------------------------------------------
	// Authentication
	// --------------------------------------------------------

	case errors.Is(
		err,
		usecase.ErrAvatarReviewReviewerRequired,
	):
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "avatar_context_required",
			},
		)

	// --------------------------------------------------------
	// Authorization
	// --------------------------------------------------------

	case errors.Is(
		err,
		usecase.ErrAvatarReviewForbidden,
	):
		writeJSON(
			w,
			http.StatusForbidden,
			map[string]string{
				"error": "avatar_review_forbidden",
			},
		)

	// --------------------------------------------------------
	// Not found
	// --------------------------------------------------------

	case errors.Is(
		err,
		usecase.ErrAvatarReviewTradeNotFound,
	):
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "trade_not_found",
			},
		)

	case errors.Is(
		err,
		usecase.ErrAvatarReviewOrderNotFound,
	):
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "order_not_found",
			},
		)

	case errors.Is(
		err,
		avatarreviewdom.ErrNotFound,
	):
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "avatar_review_not_found",
			},
		)

	// --------------------------------------------------------
	// Conflict / business state
	// --------------------------------------------------------

	case errors.Is(
		err,
		avatarreviewdom.ErrAlreadyExists,
	):
		writeJSON(
			w,
			http.StatusConflict,
			map[string]string{
				"error": "avatar_review_already_exists",
			},
		)

	case errors.Is(
		err,
		usecase.ErrAvatarReviewUnsupportedTrade,
	):
		writeJSON(
			w,
			http.StatusConflict,
			map[string]string{
				"error": "avatar_review_unsupported_trade",
			},
		)

	case errors.Is(
		err,
		usecase.ErrAvatarReviewOrderMismatch,
	):
		writeJSON(
			w,
			http.StatusConflict,
			map[string]string{
				"error": "avatar_review_order_mismatch",
			},
		)

	case errors.Is(
		err,
		usecase.ErrAvatarReviewTransferIncomplete,
	):
		writeJSON(
			w,
			http.StatusConflict,
			map[string]string{
				"error": "avatar_review_transfer_incomplete",
			},
		)

	// --------------------------------------------------------
	// Domain validation
	// --------------------------------------------------------

	case avatarreviewdom.IsInvalid(err):
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid_avatar_review",
			},
		)

	// --------------------------------------------------------
	// Unexpected
	// --------------------------------------------------------

	default:
		internalError(
			w,
			"avatar_review_internal_error",
		)
	}
}
