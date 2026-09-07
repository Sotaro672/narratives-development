// backend/internal/adapters/in/http/mall/handler/avatar_review_handler.go
package mallHandler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	usecase "narratives/internal/application/usecase"
	avatarreviewdom "narratives/internal/domain/avatar_review"
)

const (
	mallAvatarReviewsPath   = "/mall/avatar-reviews"
	mallMeAvatarReviewsPath = "/mall/me/avatar-reviews"
)

// AvatarReviewHandler handles public Avatar review reads and authenticated
// post-transfer Avatar review creation in Mall.
//
// Supported:
//
//	GET  /mall/avatar-reviews/{avatarId}
//	POST /mall/me/avatar-reviews
//
// Public GET returns reviews received by the specified Avatar.
//
// POST is available only for completed Avatar-to-Avatar Resale transactions.
// Reviewer identity is resolved from AvatarContextMiddleware and is never
// accepted from the request body. Reviewee identity is resolved from the
// authoritative Trade and Order snapshots.
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
//	GET  /mall/avatar-reviews/{avatarId}?page=1&perPage=20
//	POST /mall/me/avatar-reviews
func (h *AvatarReviewHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")

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

	if avatarID, ok := publicAvatarReviewIDFromPath(r.URL.Path); ok {
		switch r.Method {
		case http.MethodGet:
			h.listByAvatar(w, r, avatarID)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			methodNotAllowed(w)
		}
		return
	}

	if r.URL.Path == mallMeAvatarReviewsPath ||
		r.URL.Path == mallMeAvatarReviewsPath+"/" {
		switch r.Method {
		case http.MethodPost:
			h.create(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			methodNotAllowed(w)
		}
		return
	}

	notFound(w)
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
// Public list
// ============================================================

// GET /mall/avatar-reviews/{avatarId}?page=1&perPage=20
//
// Returns one page of reviews received by the specified Avatar together with
// public aggregate evaluation counts.
func (h *AvatarReviewHandler) listByAvatar(
	w http.ResponseWriter,
	r *http.Request,
	avatarID string,
) {
	page, ok := parseAvatarReviewPositiveIntQuery(
		w,
		r,
		"page",
	)
	if !ok {
		return
	}

	perPage, ok := parseAvatarReviewPositiveIntQuery(
		w,
		r,
		"perPage",
	)
	if !ok {
		return
	}

	result, err := h.uc.ListByRevieweeAvatarID(
		r.Context(),
		usecase.ListAvatarReviewsInput{
			RevieweeAvatarID: avatarID,
			Page:             page,
			PerPage:          perPage,
		},
	)
	if err != nil {
		writeAvatarReviewErr(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		map[string]any{
			"data": result,
		},
	)
}

func publicAvatarReviewIDFromPath(
	path string,
) (string, bool) {
	prefix := mallAvatarReviewsPath + "/"

	if !strings.HasPrefix(path, prefix) {
		return "", false
	}

	avatarID := strings.TrimSpace(
		strings.TrimPrefix(path, prefix),
	)
	if avatarID == "" || strings.Contains(avatarID, "/") {
		return "", false
	}

	return avatarID, true
}

// parseAvatarReviewPositiveIntQuery parses an optional positive integer query
// parameter. An omitted value is returned as zero so the usecase can apply its
// default value.
func parseAvatarReviewPositiveIntQuery(
	w http.ResponseWriter,
	r *http.Request,
	name string,
) (int, bool) {
	raw := strings.TrimSpace(
		r.URL.Query().Get(name),
	)
	if raw == "" {
		return 0, true
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		badRequest(
			w,
			"invalid_"+name,
		)
		return 0, false
	}

	return value, true
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

	req.OrderID = strings.TrimSpace(req.OrderID)

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
			OrderID:          req.OrderID,
			OrderItemIndex:   *req.OrderItemIndex,
			ReviewerAvatarID: avatarID,
			Evaluation:       req.Evaluation,
			Comment:          req.Comment,
		},
	)
	if err != nil {
		writeAvatarReviewErr(w, err)
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
	// Public read validation
	// --------------------------------------------------------

	case errors.Is(
		err,
		avatarreviewdom.ErrInvalidRevieweeAvatarID,
	):
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid_avatar_id",
			},
		)

	case errors.Is(
		err,
		avatarreviewdom.ErrInvalidPagination,
	):
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid_pagination",
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
