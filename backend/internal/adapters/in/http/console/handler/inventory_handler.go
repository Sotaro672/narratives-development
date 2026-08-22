// backend/internal/adapters/in/http/console/handler/inventory_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	invquery "narratives/internal/application/query/console"
	usecase "narratives/internal/application/usecase"
	invdom "narratives/internal/domain/inventory"
	shadom "narratives/internal/domain/shippingAddress"
	transportationdom "narratives/internal/domain/transportation"
)

type InventoryHandler struct {
	// Command/Usecase for inventory mutation
	UC *usecase.InventoryUsecase

	// Read-model(Query) for management list (view-only)
	// only: currentMember.companyId -> productBlueprintIds -> inventories(docId)
	MQ *invquery.InventoryManagementQuery

	// Read-model(Query) for inventory detail
	DQ *invquery.InventoryDetailQuery

	// listCreate 画面用 Query
	LQ *invquery.ListCreateQuery
}

func NewInventoryHandler(
	uc *usecase.InventoryUsecase,
	mq *invquery.InventoryManagementQuery,
	dq *invquery.InventoryDetailQuery,
) *InventoryHandler {
	return &InventoryHandler{
		UC: uc,
		MQ: mq,
		DQ: dq,
		LQ: nil,
	}
}

// ListCreateQuery も注入できるコンストラクタ
func NewInventoryHandlerWithListCreateQuery(
	uc *usecase.InventoryUsecase,
	mq *invquery.InventoryManagementQuery,
	dq *invquery.InventoryDetailQuery,
	lq *invquery.ListCreateQuery,
) *InventoryHandler {
	return &InventoryHandler{
		UC: uc,
		MQ: mq,
		DQ: dq,
		LQ: lq,
	}
}

type updateInventoryShippingAddressRequest struct {
	ShippingAddressID string `json:"shippingAddressId"`
}

type updateInventoryTransportationRequest struct {
	TransportationOption invdom.TransportationOption `json:"transportationOption"`
	TransportationID     string                      `json:"transportationId"`
}

func (h *InventoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")

	// ============================================================
	// Query endpoints (read-only DTO)
	// ============================================================

	// GET /inventory/list-create/{inventoryId}
	if strings.HasPrefix(path, "/inventory/list-create/") {
		switch r.Method {
		case http.MethodGet:
			h.GetListCreateByPathQuery(w, r, path)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	// GET /inventory
	if path == "/inventory" {
		switch r.Method {
		case http.MethodGet:
			h.ListByCurrentCompanyQuery(w, r)
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	// ============================================================
	// Command endpoint
	// PATCH /inventory/{inventoryId}/shipping-address
	// ============================================================

	if strings.HasPrefix(path, "/inventory/") &&
		strings.HasSuffix(path, "/shipping-address") {

		switch r.Method {
		case http.MethodPatch:
			h.UpdateShippingAddressByPath(
				w,
				r,
				path,
			)
			return

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	// ============================================================
	// Command endpoint
	// PATCH /inventory/{inventoryId}/transportation
	// ============================================================

	if strings.HasPrefix(path, "/inventory/") &&
		strings.HasSuffix(path, "/transportation") {

		switch r.Method {
		case http.MethodPatch:
			h.UpdateTransportationByPath(
				w,
				r,
				path,
			)
			return

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	// GET /inventory/{id}
	// /inventory/ids は廃止したため、ここで弾くだけ残す（誤ルーティング防止）
	if strings.HasPrefix(path, "/inventory/") {
		switch r.Method {
		case http.MethodGet:
			id := strings.TrimPrefix(path, "/inventory/")
			if id == "" || id == "ids" {
				writeInventoryError(w, http.StatusBadRequest, "invalid inventory id")
				return
			}

			// fallback 削除: Query で確定
			h.GetDetailByIDQuery(w, r, id)
			return

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

// ============================================================
// Query endpoints
// ============================================================

func (h *InventoryHandler) ListByCurrentCompanyQuery(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.MQ == nil {
		writeInventoryError(w, http.StatusNotImplemented, "inventory management query is not configured")
		return
	}

	ctx := r.Context()

	rows, err := h.MQ.ListByCurrentCompany(ctx)
	if err != nil {
		writeInventoryError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeInventoryJSON(w, http.StatusOK, rows)
}

// ============================================================
// ListCreate DTO endpoint
// - GET /inventory/list-create/{inventoryId}
// ============================================================

func (h *InventoryHandler) GetListCreateByPathQuery(w http.ResponseWriter, r *http.Request, path string) {
	if h == nil || h.LQ == nil {
		writeInventoryError(w, http.StatusNotImplemented, "list create query is not configured")
		return
	}

	ctx := r.Context()

	rest := strings.TrimPrefix(path, "/inventory/list-create/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		writeInventoryError(w, http.StatusBadRequest, "missing params")
		return
	}

	// inventoryId は docId をそのまま受け取る（pb/tb を path で受けない）
	inventoryID := rest
	if inventoryID == "" {
		writeInventoryError(w, http.StatusBadRequest, "inventoryId is required")
		return
	}

	dto, err := h.LQ.GetByInventoryID(ctx, inventoryID)
	if err != nil {
		// validation系は 400、それ以外は 500 に寄せる
		if isInventoryProbablyBadRequest(err) {
			writeInventoryError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeInventoryError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeInventoryJSON(w, http.StatusOK, dto)
}

func isInventoryProbablyBadRequest(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "required") ||
		strings.Contains(msg, "missing") ||
		strings.Contains(msg, "invalid")
}

// ============================================================
// ShippingAddress assignment endpoint
// - PATCH /inventory/{inventoryId}/shipping-address
// - current company に属する shippingAddress のみ設定可能
// - 保存成功後は更新済み InventoryDetailDTO を返す
// ============================================================

func (h *InventoryHandler) UpdateShippingAddressByPath(
	w http.ResponseWriter,
	r *http.Request,
	path string,
) {
	if h == nil || h.UC == nil {
		writeInventoryError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	if h.DQ == nil {
		writeInventoryError(w, http.StatusNotImplemented, "inventory detail query is not configured")
		return
	}

	rest := strings.TrimPrefix(
		path,
		"/inventory/",
	)

	rest = strings.TrimSuffix(
		rest,
		"/shipping-address",
	)

	inventoryID := strings.Trim(
		rest,
		"/",
	)

	if inventoryID == "" ||
		inventoryID == "ids" ||
		strings.Contains(inventoryID, "/") {

		writeInventoryError(
			w,
			http.StatusBadRequest,
			"invalid inventory id",
		)
		return
	}

	var req updateInventoryShippingAddressRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(
		&req,
	); err != nil {

		writeInventoryError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	if req.ShippingAddressID == "" {
		writeInventoryError(
			w,
			http.StatusBadRequest,
			"shippingAddressId is required",
		)
		return
	}

	ctx := r.Context()

	companyID :=
		usecase.CompanyIDFromContext(
			ctx,
		)

	if companyID == "" {
		writeInventoryError(
			w,
			http.StatusBadRequest,
			"companyId is required",
		)
		return
	}

	err := h.UC.SetShippingAddress(
		ctx,
		inventoryID,
		companyID,
		req.ShippingAddressID,
	)
	if err != nil {
		if errors.Is(
			err,
			invdom.ErrNotFound,
		) ||
			errors.Is(
				err,
				shadom.ErrNotFound,
			) {

			writeInventoryError(
				w,
				http.StatusNotFound,
				err.Error(),
			)
			return
		}

		if isInventoryProbablyBadRequest(
			err,
		) {
			writeInventoryError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)
			return
		}

		writeInventoryError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	dto, err :=
		h.DQ.GetDetailByID(
			ctx,
			inventoryID,
		)
	if err != nil {
		if errors.Is(
			err,
			invdom.ErrNotFound,
		) {
			writeInventoryError(
				w,
				http.StatusNotFound,
				err.Error(),
			)
			return
		}

		writeInventoryError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeInventoryJSON(
		w,
		http.StatusOK,
		dto,
	)
}

// ============================================================
// Transportation assignment endpoint
// - PATCH /inventory/{inventoryId}/transportation
// - yamato / sagawa / post は transportationId を持たない
// - custom は current company に属する transportationId のみ設定可能
// - 保存成功後は更新済み InventoryDetailDTO を返す
// ============================================================

func (h *InventoryHandler) UpdateTransportationByPath(
	w http.ResponseWriter,
	r *http.Request,
	path string,
) {
	if h == nil || h.UC == nil {
		writeInventoryError(w, http.StatusNotImplemented, "inventory usecase is not configured")
		return
	}

	if h.DQ == nil {
		writeInventoryError(w, http.StatusNotImplemented, "inventory detail query is not configured")
		return
	}

	rest := strings.TrimPrefix(
		path,
		"/inventory/",
	)

	rest = strings.TrimSuffix(
		rest,
		"/transportation",
	)

	inventoryID := strings.Trim(
		rest,
		"/",
	)

	if inventoryID == "" ||
		inventoryID == "ids" ||
		strings.Contains(inventoryID, "/") {

		writeInventoryError(
			w,
			http.StatusBadRequest,
			"invalid inventory id",
		)
		return
	}

	var req updateInventoryTransportationRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(
		&req,
	); err != nil {

		writeInventoryError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	if req.TransportationOption == "" {
		writeInventoryError(
			w,
			http.StatusBadRequest,
			"transportationOption is required",
		)
		return
	}

	if !invdom.IsValidTransportationOption(
		req.TransportationOption,
	) {
		writeInventoryError(
			w,
			http.StatusBadRequest,
			invdom.ErrInvalidTransportationOption.Error(),
		)
		return
	}

	if req.TransportationOption ==
		invdom.TransportationOptionCustom {

		if req.TransportationID == "" {
			writeInventoryError(
				w,
				http.StatusBadRequest,
				invdom.ErrTransportationIDRequired.Error(),
			)
			return
		}
	} else if req.TransportationID != "" {
		writeInventoryError(
			w,
			http.StatusBadRequest,
			invdom.ErrTransportationIDNotAllowed.Error(),
		)
		return
	}

	ctx := r.Context()

	companyID :=
		usecase.CompanyIDFromContext(
			ctx,
		)

	if companyID == "" {
		writeInventoryError(
			w,
			http.StatusBadRequest,
			"companyId is required",
		)
		return
	}

	err := h.UC.SetTransportation(
		ctx,
		inventoryID,
		companyID,
		req.TransportationOption,
		req.TransportationID,
	)
	if err != nil {
		if errors.Is(
			err,
			invdom.ErrNotFound,
		) ||
			errors.Is(
				err,
				transportationdom.ErrNotFound,
			) {

			writeInventoryError(
				w,
				http.StatusNotFound,
				err.Error(),
			)
			return
		}

		if errors.Is(
			err,
			invdom.ErrInvalidTransportationOption,
		) ||
			errors.Is(
				err,
				invdom.ErrTransportationIDRequired,
			) ||
			errors.Is(
				err,
				invdom.ErrTransportationIDNotAllowed,
			) {

			writeInventoryError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)
			return
		}

		if isInventoryProbablyBadRequest(
			err,
		) {
			writeInventoryError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)
			return
		}

		writeInventoryError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	dto, err :=
		h.DQ.GetDetailByID(
			ctx,
			inventoryID,
		)
	if err != nil {
		if errors.Is(
			err,
			invdom.ErrNotFound,
		) {
			writeInventoryError(
				w,
				http.StatusNotFound,
				err.Error(),
			)
			return
		}

		writeInventoryError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeInventoryJSON(
		w,
		http.StatusOK,
		dto,
	)
}

// ============================================================
// Detail endpoint（確定）
// ============================================================

func (h *InventoryHandler) GetDetailByIDQuery(w http.ResponseWriter, r *http.Request, inventoryID string) {
	if h == nil || h.DQ == nil {
		writeInventoryError(w, http.StatusNotImplemented, "inventory detail query is not configured")
		return
	}

	ctx := r.Context()
	if inventoryID == "" {
		writeInventoryError(w, http.StatusBadRequest, "missing id")
		return
	}

	dto, err := h.DQ.GetDetailByID(ctx, inventoryID)
	if err != nil {
		if errors.Is(err, invdom.ErrNotFound) {
			writeInventoryError(w, http.StatusNotFound, err.Error())
			return
		}
		writeInventoryError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeInventoryJSON(w, http.StatusOK, dto)
}

// ============================================================
// HTTP helpers
// ============================================================

func writeInventoryJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeInventoryError(w http.ResponseWriter, status int, msg string) {
	writeInventoryJSON(w, status, map[string]any{"error": msg})
}
